package repository

import (
	"errors"
	"fmt"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

// ApplyMigrations applies SQL migrations from a directory using golang-migrate.
func ApplyMigrations(databaseURL, migrationsPath string) error {
	if strings.TrimSpace(databaseURL) == "" {
		return errors.New("DATABASE_URL is required")
	}
	if strings.TrimSpace(migrationsPath) == "" {
		return errors.New("MIGRATIONS_PATH is required")
	}
	absolutePath, err := filepath.Abs(migrationsPath)
	if err != nil {
		return fmt.Errorf("resolve migrations path: %w", err)
	}
	if _, err := os.Stat(absolutePath); err != nil {
		return fmt.Errorf("access migrations directory: %w", err)
	}
	migrationURL := (&url.URL{Scheme: "file", Opaque: filepath.ToSlash(absolutePath)}).String()
	m, err := migrate.New(migrationURL, databaseURL)
	if err != nil {
		return fmt.Errorf("initialize migrations: %w", err)
	}
	defer m.Close()
	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("apply migrations: %w", err)
	}
	return nil
}

// ApplyMigrationsFS applies SQL migrations from an fs.FS (for example go:embed).
func ApplyMigrationsFS(databaseURL string, files fs.FS, dir string) error {
	if strings.TrimSpace(databaseURL) == "" {
		return errors.New("DATABASE_URL is required")
	}
	source, err := iofs.New(files, dir)
	if err != nil {
		return fmt.Errorf("open embedded migrations: %w", err)
	}
	m, err := migrate.NewWithSourceInstance("iofs", source, databaseURL)
	if err != nil {
		return fmt.Errorf("initialize migrations: %w", err)
	}
	defer m.Close()
	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("apply migrations: %w", err)
	}
	return nil
}
