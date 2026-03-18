package storage

import (
	"audiotranscrib/internal/config"
	"context"
	"database/sql"
	"fmt"

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
		logger.Error("can't initialize database migrations", zap.Error(err))
	}

	db, err := sql.Open("postgres", cfg.DatabaseDNS)
	if err != nil {
		return nil, fmt.Errorf("cannot open DB: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("cannot connect to DB: %w", err)
	}

	logger.Info("Connected to PostgreSQL")

	st := &DBStorage{
		DB:     db,
		Logger: logger,
	}

	lc.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			logger.Info("closing database")
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
	return err
}

func (s *DBStorage) SaveMeeting(
	ctx context.Context,
	userID int,
	fileName string,
	transcription string,
	summary string,
) (int, error) {

	query := `
	INSERT INTO meetings (user_id, file_name, transcription, summary)
	VALUES ($1,$2,$3,$4)
	RETURNING id
	`

	var id int

	err := s.DB.QueryRowContext(
		ctx,
		query,
		userID,
		fileName,
		transcription,
		summary,
	).Scan(&id)

	return id, err
}

func (s *DBStorage) ListMeetings(ctx context.Context, userID int) ([]Meeting, error) {

	rows, err := s.DB.QueryContext(ctx, `
	SELECT id, file_name, created_at
	FROM meetings
	WHERE user_id=$1
	ORDER BY created_at DESC
	`, userID)

	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var meetings []Meeting

	for rows.Next() {
		var m Meeting

		err := rows.Scan(
			&m.ID,
			&m.FileName,
			&m.CreatedAt,
		)

		if err != nil {
			return nil, err
		}

		meetings = append(meetings, m)
	}

	return meetings, nil
}

func (s *DBStorage) GetMeeting(ctx context.Context, id int) (*Meeting, error) {

	row := s.DB.QueryRowContext(ctx, `
	SELECT id, transcription, summary
	FROM meetings
	WHERE id=$1
	`, id)

	var m Meeting

	err := row.Scan(
		&m.ID,
		&m.Transcription,
		&m.Summary,
	)

	if err != nil {
		return nil, err
	}

	return &m, nil
}

func (s *DBStorage) FindMeetings(ctx context.Context, keyword string) ([]Meeting, error) {

	rows, err := s.DB.QueryContext(ctx, `
	SELECT id, summary
	FROM meetings
	WHERE to_tsvector('russian', transcription)
	@@ plainto_tsquery($1)
	`, keyword)

	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var result []Meeting

	for rows.Next() {

		var m Meeting

		err := rows.Scan(
			&m.ID,
			&m.Summary,
		)

		if err != nil {
			return nil, err
		}

		result = append(result, m)
	}

	return result, nil
}
