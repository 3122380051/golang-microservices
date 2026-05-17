package database

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

// Migrate runs database migrations from the migrations directory.
func Migrate(databaseURL, migrationsDir, action string) error {
	absPath, err := filepath.Abs(migrationsDir)
	if err != nil {
		return fmt.Errorf("resolve migrations dir: %w", err)
	}

	m, err := migrate.New("file://"+absPath, databaseURL)
	if err != nil {
		return fmt.Errorf("create migrator: %w", err)
	}
	defer m.Close()

	switch strings.ToLower(action) {
	case "up":
		if err := m.Up(); err != nil && err != migrate.ErrNoChange {
			return fmt.Errorf("migrate up: %w", err)
		}
	case "down":
		if err := m.Down(); err != nil && err != migrate.ErrNoChange {
			return fmt.Errorf("migrate down: %w", err)
		}
	case "status":
		version, dirty, err := m.Version()
		if err != nil {
			return fmt.Errorf("migrate status: %w", err)
		}
		_ = context.Background()
		_ = version
		_ = dirty
	default:
		return fmt.Errorf("unsupported migration action: %s", action)
	}

	return nil
}
