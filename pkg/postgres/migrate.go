package postgres

import (
	"errors"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres" // register postgres driver
	_ "github.com/golang-migrate/migrate/v4/source/file"       // register file source driver
)

// RunMigrations runs all pending database migrations from the given directory.
// The migrationsDir should be a path to a directory containing migration files
// (e.g. "file://./migrations"). If there are no new migrations to apply the
// function returns nil.
func RunMigrations(dsn string, migrationsDir string) error {
	m, err := migrate.New(migrationsDir, dsn)
	if err != nil {
		return fmt.Errorf("postgres: create migrator: %w", err)
	}
	defer m.Close()

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("postgres: run migrations up: %w", err)
	}

	return nil
}

// RunMigrationsDown rolls back all database migrations.
// If there are no migrations to roll back the function returns nil.
func RunMigrationsDown(dsn string, migrationsDir string) error {
	m, err := migrate.New(migrationsDir, dsn)
	if err != nil {
		return fmt.Errorf("postgres: create migrator: %w", err)
	}
	defer m.Close()

	if err := m.Down(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("postgres: run migrations down: %w", err)
	}

	return nil
}
