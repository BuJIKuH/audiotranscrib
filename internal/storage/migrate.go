// Package storage содержит функции для работы с базой данных и миграциями.
package storage

import (
	"path/filepath"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"

	"go.uber.org/zap"
)

// RunMigrations выполняет миграции базы данных PostgreSQL.
// dns — строка подключения к базе данных.
// logger — zap.Logger для логирования ошибок и информации.
func RunMigrations(dns string, logger *zap.Logger) error {

	path, err := filepath.Abs("migrations")
	if err != nil {
		return err
	}

	migrationsURL := "file://" + path

	logger.Info("running migrations", zap.String("path", migrationsURL))

	m, err := migrate.New(migrationsURL, dns)
	if err != nil {
		return err
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return err
	}

	logger.Info("migrations applied")

	return nil
}
