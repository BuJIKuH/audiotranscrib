package storage

import (
	"context"
	"time"

	"go.uber.org/zap"
)

type User struct {
	ID         int
	TelegramID int64
	Username   string
	CreatedAt  time.Time
}

type UserRepo struct {
	storage *DBStorage
	logger  *zap.Logger
}

func NewUserRepo(storage *DBStorage, logger *zap.Logger) *UserRepo {
	return &UserRepo{
		storage: storage,
		logger:  logger,
	}
}

func (r *UserRepo) CreateUser(ctx context.Context, telegramID int64, username string) (*User, error) {
	var id int
	err := r.storage.DB.QueryRowContext(
		ctx,
		`INSERT INTO users (telegram_id, username)
		 VALUES ($1, $2)
		 ON CONFLICT (telegram_id) DO UPDATE
		 SET username = EXCLUDED.username
		 RETURNING id`,
		telegramID,
		username,
	).Scan(&id)
	if err != nil {
		r.logger.Error("failed to create user",
			zap.Int64("telegram_id", telegramID),
			zap.String("username", username),
			zap.Error(err),
		)
		return nil, err
	}

	user := &User{
		ID:         id,
		TelegramID: telegramID,
		Username:   username,
		CreatedAt:  time.Now(),
	}

	r.logger.Info("user created/updated",
		zap.Int("id", user.ID),
		zap.Int64("telegram_id", user.TelegramID),
		zap.String("username", user.Username),
	)

	return user, nil
}

func (r *UserRepo) GetUserByTelegramID(ctx context.Context, telegramID int64) (*User, error) {
	u := &User{}
	err := r.storage.DB.QueryRowContext(
		ctx,
		`SELECT id, telegram_id, username, created_at
		 FROM users
		 WHERE telegram_id = $1`,
		telegramID,
	).Scan(&u.ID, &u.TelegramID, &u.Username, &u.CreatedAt)
	if err != nil {
		r.logger.Error("failed to get user by telegram_id",
			zap.Int64("telegram_id", telegramID),
			zap.Error(err),
		)
		return nil, err
	}

	r.logger.Info("user fetched",
		zap.Int("id", u.ID),
		zap.Int64("telegram_id", u.TelegramID),
		zap.String("username", u.Username),
	)

	return u, nil
}
