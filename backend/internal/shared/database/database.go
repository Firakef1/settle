package database

import (
	"database/sql"
	"fmt"
	"log"

	"github.com/Firakef1/settle/backend/internal/shared/config"
	_ "github.com/lib/pq"
)

// DB is the global database connection pool.
var DB *sql.DB

// Connect initializes the database connection using the configuration.
func Connect() error {
	dsn := config.AppConfig.DatabaseURL
	if dsn == "" {
		return fmt.Errorf("DATABASE_URL environment variable is not set")
	}

	var err error
	DB, err = sql.Open("postgres", dsn)
	if err != nil {
		return fmt.Errorf("failed to open database connection: %w", err)
	}

	// Verify the connection
	if err = DB.Ping(); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	// Set connection pool limits
	DB.SetMaxOpenConns(25)
	DB.SetMaxIdleConns(5)

	log.Println("Successfully connected to the database")
	return nil
}

// Close closes the database connection.
func Close() error {
	if DB != nil {
		return DB.Close()
	}
	return nil
}
