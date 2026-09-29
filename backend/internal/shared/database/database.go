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
	EnsureVerificationSchema()
	return nil
}

// EnsureVerificationSchema guarantees that required verification tables/columns exist.
func EnsureVerificationSchema() {
	if DB == nil {
		return
	}
	_, _ = DB.Exec(`ALTER TABLE users ADD COLUMN IF NOT EXISTS email_verified BOOLEAN NOT NULL DEFAULT FALSE`)

	query := `
	CREATE TABLE IF NOT EXISTS verification_codes (
		id         VARCHAR(36) PRIMARY KEY,
		email      VARCHAR(255) NOT NULL,
		code       VARCHAR(6) NOT NULL,
		expires_at TIMESTAMPTZ NOT NULL,
		created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
	);
	CREATE INDEX IF NOT EXISTS idx_verification_codes_email ON verification_codes(email);
	CREATE INDEX IF NOT EXISTS idx_verification_codes_expires_at ON verification_codes(expires_at);
	`
	if _, err := DB.Exec(query); err != nil {
		log.Printf("[WARNING] EnsureVerificationSchema: %v", err)
	} else {
		log.Println("Verification database schema verified successfully")
	}
}

// Close closes the database connection.
func Close() error {
	if DB != nil {
		return DB.Close()
	}
	return nil
}
