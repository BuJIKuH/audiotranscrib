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
	*Repository
}

func NewMeetingRepo(storage *DBStorage, logger *zap.Logger) *MeetingRepo {
	return &MeetingRepo{
		Repository: NewRepository(storage, logger),
	}
}

func (r *MeetingRepo) SaveMeeting(ctx context.Context, m *Meeting) error {
	err := r.QueryRow(
		ctx,
		`INSERT INTO meetings (user_id, file_name, transcription, summary, created_at)
		 VALUES ($1,$2,$3,$4, NOW()) RETURNING id`,
		m.UserID, m.FileName, m.Transcription, m.Summary,
	).Scan(&m.ID)

	if err != nil {
		r.logger.Error("failed to save meeting",
			zap.Int("user_id", m.UserID),
			zap.String("file_name", m.FileName),
			zap.Error(err),
		)
		return err
	}

	r.logger.Info("meeting saved",
		zap.Int("meeting_id", m.ID),
		zap.Int("user_id", m.UserID),
	)
	return nil
}

func (r *MeetingRepo) ListMeetingsByUser(ctx context.Context, userID int) ([]Meeting, error) {
	rows, err := r.Query(
		ctx,
		`SELECT id, user_id, file_name, transcription, summary, created_at
		 FROM meetings
		 WHERE user_id=$1
		 ORDER BY created_at DESC`,
		userID,
	)
	if err != nil {
		r.logger.Error("failed to list meetings",
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
			r.logger.Warn("failed to scan meeting row", zap.Error(err))
			continue
		}
		meetings = append(meetings, m)
	}

	if err := rows.Err(); err != nil {
		r.logger.Error("rows iteration error",
			zap.Int("user_id", userID),
			zap.Error(err),
		)
		return nil, err
	}

	r.logger.Info("meetings fetched",
		zap.Int("user_id", userID),
		zap.Int("count", len(meetings)),
	)
	return meetings, nil
}

func (r *MeetingRepo) GetMeetingByID(ctx context.Context, id int) (*Meeting, error) {
	row := r.QueryRow(
		ctx,
		`SELECT id, user_id, file_name, transcription, summary, created_at
		 FROM meetings
		 WHERE id=$1`,
		id,
	)

	m := &Meeting{}
	if err := row.Scan(&m.ID, &m.UserID, &m.FileName, &m.Transcription, &m.Summary, &m.CreatedAt); err != nil {
		r.logger.Error("failed to get meeting by ID",
			zap.Int("meeting_id", id),
			zap.Error(err),
		)
		return nil, err
	}

	r.logger.Info("meeting fetched by ID",
		zap.Int("meeting_id", m.ID),
		zap.Int("user_id", m.UserID),
	)
	return m, nil
}
