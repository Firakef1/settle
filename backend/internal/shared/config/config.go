package config

import (
	"os"
	"time"
)

// Config holds all application configuration loaded from environment variables.
type Config struct {
	// SecretKey is the JWT signing secret, loaded from Secret_key env var.
	SecretKey string
	// RefreshTokenTTL defines the lifetime of a refresh token.
	RefreshTokenTTL time.Duration
	// AccessTokenTTL defines the lifetime of an access token.
	AccessTokenTTL time.Duration
	// DatabaseURL for DB connection.
	DatabaseURL string
	// AppEnv is the application environment (development, production, etc.).
	AppEnv string
	// HTTPPort is the HTTP server port.
	HTTPPort string
}

// AppConfig is the global application configuration.
var AppConfig Config

// Load reads environment variables and populates AppConfig.
func Load() {
	AppConfig = Config{
		SecretKey:       getEnvOrDefault("SECRET_KEY", "settle_default_development_secret_key_change_in_prod"),
		RefreshTokenTTL: parseDurationOrDefault(os.Getenv("REFRESH_TOKEN_TTL"), 30*24*time.Hour), // default 30 days
		AccessTokenTTL:  parseDurationOrDefault(os.Getenv("ACCESS_TOKEN_TTL"), 24*time.Hour),     // default 24 hours
		DatabaseURL:     getEnvOrDefault("DATABASE_URL", ""),
		AppEnv:          getEnvOrDefault("APP_ENV", "development"),
		HTTPPort:        getEnvOrDefault("HTTP_PORT", "8080"),
	}
}

func getEnvOrDefault(key, defaultVal string) string {
	val := os.Getenv(key)
	if val == "" {
		return defaultVal
	}
	return val
}

func parseDurationOrDefault(val string, defaultDur time.Duration) time.Duration {
	if val == "" {
		return defaultDur
	}
	d, err := time.ParseDuration(val)
	if err != nil {
		return defaultDur
	}
	return d
}
