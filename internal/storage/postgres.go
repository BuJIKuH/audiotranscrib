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
	DB *sql.DB
}

func NewDBStorage(
	lc fx.Lifecycle,
	cfg *config.Config,
	logger *zap.Logger,
) (*DBStorage, error) {

	if err := RunMigrations(cfg.DatabaseDNS, logger); err != nil {
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	db, err := sql.Open("postgres", cfg.DatabaseDNS)
	if err != nil {
		return nil, fmt.Errorf("cannot open DB: %w", err)
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(25)
	db.SetConnMaxLifetime(5 * time.Minute)

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("cannot connect to DB: %w", err)
	}

	logger.Info("connected to PostgreSQL")

	st := &DBStorage{
		DB: db,
	}

	lc.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			logger.Info("closing database connection")
			return db.Close()
		},
	})

	return st, nil
}
