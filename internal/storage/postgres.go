package storage

import (
	"audiotranscrib/internal/config"
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type DBStorage struct {
	DB     *sql.DB
	Logger *zap.Logger
}

func NewDBStorage(
	lc fx.Lifecycle,
	cfg *config.Config,
	logger *zap.Logger,
) (*DBStorage, error) {

	if err := RunMigrations(cfg.DatabaseDNS, logger); err != nil {
		logger.Error("failed to run database migrations", zap.Error(err))
	}

	db, err := sql.Open("postgres", cfg.DatabaseDNS)
	if err != nil {
		return nil, fmt.Errorf("cannot open DB: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("cannot connect to DB: %w", err)
	}

	logger.Info("connected to PostgreSQL")

	st := &DBStorage{
		DB:     db,
		Logger: logger,
	}

	lc.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			logger.Info("closing database connection")
			return db.Close()
		},
	})

	return st, nil
}

func (s *DBStorage) CreateUser(ctx context.Context, telegramID int64, username string) error {
	query := `
	INSERT INTO users (telegram_id, username)
	VALUES ($1, $2)
	ON CONFLICT (telegram_id) DO NOTHING
	`

	_, err := s.DB.ExecContext(ctx, query, telegramID, username)
	if err != nil {
		s.Logger.Error("failed to create user",
			zap.Int64("telegram_id", telegramID),
			zap.String("username", username),
			zap.Error(err),
		)
		return err
	}

	s.Logger.Info("user created",
		zap.Int64("telegram_id", telegramID),
		zap.String("username", username),
	)
	return nil
}

func (s *DBStorage) SaveMeeting(ctx context.Context, userID int, fileName, transcription, summary string) (int, error) {
	query := `
	INSERT INTO meetings (user_id, file_name, transcription, summary, created_at)
	VALUES ($1, $2, $3, $4, $5)
	RETURNING id
	`

	var id int
	err := s.DB.QueryRowContext(ctx, query, userID, fileName, transcription, summary, time.Now()).Scan(&id)
	if err != nil {
		s.Logger.Error("failed to save meeting",
			zap.Int("user_id", userID),
			zap.String("file_name", fileName),
			zap.Error(err),
		)
		return 0, err
	}

	s.Logger.Info("meeting saved",
		zap.Int("meeting_id", id),
		zap.Int("user_id", userID),
	)
	return id, nil
}

func (s *DBStorage) ListMeetings(ctx context.Context, userID int) ([]Meeting, error) {
	rows, err := s.DB.QueryContext(ctx, `
		SELECT id, file_name, created_at
		FROM meetings
		WHERE user_id=$1
		ORDER BY created_at DESC
	`, userID)
	if err != nil {
		s.Logger.Error("failed to list meetings",
			zap.Int("user_id", userID),
			zap.Error(err),
		)
		return nil, err
	}
	defer rows.Close()

	var meetings []Meeting
	for rows.Next() {
		var m Meeting
		if err := rows.Scan(&m.ID, &m.FileName, &m.CreatedAt); err != nil {
			s.Logger.Warn("failed to scan meeting row", zap.Error(err))
			continue
		}
		meetings = append(meetings, m)
	}

	s.Logger.Info("meetings fetched",
		zap.Int("user_id", userID),
		zap.Int("count", len(meetings)),
	)
	return meetings, nil
}

func (s *DBStorage) GetMeeting(ctx context.Context, id int) (*Meeting, error) {
	row := s.DB.QueryRowContext(ctx, `
		SELECT id, transcription, summary
		FROM meetings
		WHERE id=$1
	`, id)

	var m Meeting
	if err := row.Scan(&m.ID, &m.Transcription, &m.Summary); err != nil {
		s.Logger.Error("failed to get meeting",
			zap.Int("meeting_id", id),
			zap.Error(err),
		)
		return nil, err
	}

	s.Logger.Info("meeting fetched",
		zap.Int("meeting_id", m.ID),
	)
	return &m, nil
}

func (s *DBStorage) FindMeetings(ctx context.Context, keyword string) ([]Meeting, error) {
	rows, err := s.DB.QueryContext(ctx, `
		SELECT id, summary
		FROM meetings
		WHERE to_tsvector('russian', transcription) @@ plainto_tsquery($1)
	`, keyword)
	if err != nil {
		s.Logger.Error("failed to search meetings",
			zap.String("keyword", keyword),
			zap.Error(err),
		)
		return nil, err
	}
	defer rows.Close()

	var meetings []Meeting
	for rows.Next() {
		var m Meeting
		if err := rows.Scan(&m.ID, &m.Summary); err != nil {
			s.Logger.Warn("failed to scan meeting row during search", zap.Error(err))
			continue
		}
		meetings = append(meetings, m)
	}

	s.Logger.Info("meetings found",
		zap.String("keyword", keyword),
		zap.Int("count", len(meetings)),
	)
	return meetings, nil
}
