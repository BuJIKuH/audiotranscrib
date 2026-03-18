package storage

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type User struct {
	ID         int
	TelegramID int64
	Username   string
	CreatedAt  time.Time
}

type UserRepo struct {
	db *pgxpool.Pool
}

func NewUserRepo(db *pgxpool.Pool) *UserRepo {
	return &UserRepo{db: db}
}

func (r *UserRepo) CreateUser(ctx context.Context, telegramID int64, username string) (*User, error) {
	var id int
	err := r.db.QueryRow(ctx,
		`INSERT INTO users (telegram_id, username) VALUES ($1, $2) RETURNING id`,
		telegramID, username).Scan(&id)
	if err != nil {
		return nil, err
	}

	return &User{
		ID:         id,
		TelegramID: telegramID,
		Username:   username,
		CreatedAt:  time.Now(),
	}, nil
}

func (r *UserRepo) GetUserByTelegramID(ctx context.Context, telegramID int64) (*User, error) {
	row := r.db.QueryRow(ctx, `SELECT id, telegram_id, username, created_at FROM users WHERE telegram_id=$1`, telegramID)

	u := &User{}
	err := row.Scan(&u.ID, &u.TelegramID, &u.Username, &u.CreatedAt)
	if err != nil {
		return nil, err
	}

	return u, nil
}
