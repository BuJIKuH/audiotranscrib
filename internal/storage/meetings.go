package storage

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
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
	db *pgxpool.Pool
}

func NewMeetingRepo(db *pgxpool.Pool) *MeetingRepo {
	return &MeetingRepo{db: db}
}

func (r *MeetingRepo) SaveMeeting(ctx context.Context, m *Meeting) error {
	err := r.db.QueryRow(ctx,
		`INSERT INTO meetings (user_id, file_name, transcription, summary, created_at)
		 VALUES ($1,$2,$3,$4,$5) RETURNING id`,
		m.UserID, m.FileName, m.Transcription, m.Summary, time.Now(),
	).Scan(&m.ID)
	return err
}

func (r *MeetingRepo) ListMeetingsByUser(ctx context.Context, userID int) ([]Meeting, error) {
	rows, err := r.db.Query(ctx, `SELECT id, user_id, file_name, transcription, summary, created_at 
		FROM meetings WHERE user_id=$1 ORDER BY created_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var meetings []Meeting
	for rows.Next() {
		var m Meeting
		err = rows.Scan(&m.ID, &m.UserID, &m.FileName, &m.Transcription, &m.Summary, &m.CreatedAt)
		if err != nil {
			return nil, err
		}
		meetings = append(meetings, m)
	}

	return meetings, nil
}

func (r *MeetingRepo) GetMeetingByID(ctx context.Context, id int) (*Meeting, error) {
	row := r.db.QueryRow(ctx, `SELECT id, user_id, file_name, transcription, summary, created_at 
		FROM meetings WHERE id=$1`, id)

	m := &Meeting{}
	err := row.Scan(&m.ID, &m.UserID, &m.FileName, &m.Transcription, &m.Summary, &m.CreatedAt)
	if err != nil {
		return nil, err
	}

	return m, nil
}
