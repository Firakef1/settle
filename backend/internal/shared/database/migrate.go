package database

import (
	"errors"
	"fmt"
	"log"

	"github.com/Firakef1/settle/backend/internal/shared/config"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

// RunMigrations connects to the database and applies all pending migrations.
// It skips gracefully if no new migrations are found.
func RunMigrations() error {
	dsn := config.AppConfig.DatabaseURL
	if dsn == "" {
		return fmt.Errorf("DATABASE_URL environment variable is not set")
	}

	log.Println("Initializing database migrations...")

	// We point migrate to the local "migrations" folder.
	// We use "file://migrations" assuming the binary is run from the root of the backend folder.
	m, err := migrate.New("file://migrations", dsn)
	if err != nil {
		return fmt.Errorf("failed to initialize migrations: %w", err)
	}
	defer m.Close()

	if err := m.Up(); err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			log.Println("Database schema is up to date. No new migrations applied.")
			return nil
		}
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	log.Println("Successfully applied pending database migrations!")
	return nil
}
