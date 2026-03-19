package storage

import (
	"context"
	"time"

	"go.uber.org/zap"
)

type Meeting struct {
	ID            int
	UserID        int
	FileName      string
	Transcription string
	Summary       string
	CreatedAt     time.Time
}

type MeetingRepo struct {
	storage *DBStorage
	logger  *zap.Logger
}

func NewMeetingRepo(storage *DBStorage, logger *zap.Logger) *MeetingRepo {
	return &MeetingRepo{
		storage: storage,
		logger:  logger,
	}
}

func (r *MeetingRepo) SaveMeeting(ctx context.Context, m *Meeting) error {
	err := r.storage.DB.QueryRowContext(
		ctx,
		`INSERT INTO meetings (user_id, file_name, transcription, summary, created_at)
		 VALUES ($1,$2,$3,$4,$5) RETURNING id`,
		m.UserID, m.FileName, m.Transcription, m.Summary, time.Now(),
	).Scan(&m.ID)

	if err != nil {
		r.storage.Logger.Error("failed to save meeting",
			zap.Int("user_id", m.UserID),
			zap.String("file_name", m.FileName),
			zap.Error(err),
		)
		return err
	}

	r.storage.Logger.Info("meeting saved",
		zap.Int("meeting_id", m.ID),
		zap.Int("user_id", m.UserID),
	)
	return nil
}

func (r *MeetingRepo) ListMeetingsByUser(ctx context.Context, userID int) ([]Meeting, error) {
	rows, err := r.storage.DB.QueryContext(
		ctx,
		`SELECT id, user_id, file_name, transcription, summary, created_at
		 FROM meetings
		 WHERE user_id=$1
		 ORDER BY created_at DESC`,
		userID,
	)
	if err != nil {
		r.storage.Logger.Error("failed to list meetings",
			zap.Int("user_id", userID),
			zap.Error(err),
		)
		return nil, err
	}
	defer rows.Close()

	var meetings []Meeting
	for rows.Next() {
		var m Meeting
		if err := rows.Scan(&m.ID, &m.UserID, &m.FileName, &m.Transcription, &m.Summary, &m.CreatedAt); err != nil {
			r.storage.Logger.Warn("failed to scan meeting row", zap.Error(err))
			continue
		}
		meetings = append(meetings, m)
	}

	r.storage.Logger.Info("meetings fetched",
		zap.Int("user_id", userID),
		zap.Int("count", len(meetings)),
	)
	return meetings, nil
}

func (r *MeetingRepo) GetMeetingByID(ctx context.Context, id int) (*Meeting, error) {
	row := r.storage.DB.QueryRowContext(
		ctx,
		`SELECT id, user_id, file_name, transcription, summary, created_at
		 FROM meetings
		 WHERE id=$1`,
		id,
	)

	m := &Meeting{}
	if err := row.Scan(&m.ID, &m.UserID, &m.FileName, &m.Transcription, &m.Summary, &m.CreatedAt); err != nil {
		r.storage.Logger.Error("failed to get meeting by ID",
			zap.Int("meeting_id", id),
			zap.Error(err),
		)
		return nil, err
	}

	r.storage.Logger.Info("meeting fetched by ID",
		zap.Int("meeting_id", m.ID),
		zap.Int("user_id", m.UserID),
	)
	return m, nil
}
