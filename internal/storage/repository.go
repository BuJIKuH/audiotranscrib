package storage

import (
	"context"
	"database/sql"

	"go.uber.org/zap"
)

type Repository struct {
	db     *sql.DB
	logger *zap.Logger
}

func NewRepository(storage *DBStorage, logger *zap.Logger) *Repository {
	return &Repository{
		db:     storage.DB,
		logger: logger,
	}
}

func (r *Repository) QueryRow(ctx context.Context, query string, args ...any) *sql.Row {
	return r.db.QueryRowContext(ctx, query, args...)
}

func (r *Repository) Query(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	return r.db.QueryContext(ctx, query, args...)
}

func (r *Repository) Exec(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return r.db.ExecContext(ctx, query, args...)
}
