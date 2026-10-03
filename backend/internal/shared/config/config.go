package config

import (
	"errors"
	"log"
	"os"
	"strconv"
	"time"
)

// EmailConfig holds email delivery and verification-code configuration.
type EmailConfig struct {
	SMTPHost     string        // SMTP_HOST
	SMTPPort     string        // SMTP_PORT (default "587")
	SMTPUsername string        // SMTP_USERNAME (optional; if set, password is required)
	SMTPPassword string        // SMTP_PASSWORD
	FromAddress  string        // EMAIL_FROM_ADDRESS
	FromName     string        // EMAIL_FROM_NAME
	SendTimeout  time.Duration // EMAIL_SEND_TIMEOUT (default 10s)
	LogOnly      bool          // EMAIL_LOG_ONLY, dev only: log emails instead of sending
	CodeTTL      time.Duration // EMAIL_CODE_TTL (default 15m)
	CodeSecret   string        // EMAIL_CODE_SECRET (required, >= 32 chars)
}

// Validate returns an error if the EmailConfig is invalid.
func (c EmailConfig) Validate() error {
	if len(c.CodeSecret) < 32 {
		return errors.New("EMAIL_CODE_SECRET must be at least 32 characters")
	}
	if c.CodeTTL <= 0 || c.SendTimeout <= 0 {
		return errors.New("EMAIL_CODE_TTL and EMAIL_SEND_TIMEOUT must be positive")
	}
	if c.LogOnly {
		return nil // dev mode: no SMTP needed
	}
	if c.SMTPHost == "" || c.FromAddress == "" {
		return errors.New("SMTP_HOST and EMAIL_FROM_ADDRESS are required unless EMAIL_LOG_ONLY=true")
	}
	if c.SMTPUsername != "" && c.SMTPPassword == "" {
		return errors.New("SMTP_PASSWORD is required when SMTP_USERNAME is set")
	}
	return nil
}

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
	// Email holds email delivery and code configuration.
	Email EmailConfig
	// PasswordResetSecret key used for HMAC SHA256 hashing of OTPs.
	PasswordResetSecret string
	// PasswordResetOTPTTL lifetime of password reset OTP.
	PasswordResetOTPTTL time.Duration
	// PasswordResetMaxAttempts max failed attempts per OTP.
	PasswordResetMaxAttempts int
	// UploadDir is where uploaded files are stored.
	UploadDir string
	// OCRApiKey is the key for the free OCR API.
	OCRApiKey string
}

// AppConfig is the global application configuration.
var AppConfig Config

// Load reads environment variables and populates AppConfig.
func Load() {
	secretKey := getEnvOrDefault("SECRET_KEY", "settle_default_development_secret_key_change_in_prod")
	codeSecret := getEnvOrDefault("EMAIL_CODE_SECRET", "settle_default_email_code_secret_at_least_32_chars")
	pwdResetSecret := os.Getenv("PASSWORD_RESET_SECRET")
	if pwdResetSecret == "" {
		pwdResetSecret = codeSecret
	}
	if pwdResetSecret == "" {
		pwdResetSecret = secretKey
	}

	AppConfig = Config{
		SecretKey:                secretKey,
		RefreshTokenTTL:          parseDurationOrDefault(os.Getenv("REFRESH_TOKEN_TTL"), 30*24*time.Hour), // default 30 days
		AccessTokenTTL:           parseDurationOrDefault(os.Getenv("ACCESS_TOKEN_TTL"), 24*time.Hour),     // default 24 hours
		DatabaseURL:              getEnvOrDefault("DATABASE_URL", ""),
		AppEnv:                   getEnvOrDefault("APP_ENV", "development"),
		HTTPPort:                 getEnvOrDefault("HTTP_PORT", "8080"),
		PasswordResetSecret:      pwdResetSecret,
		PasswordResetOTPTTL:      parseDurationOrDefault(os.Getenv("PASSWORD_RESET_OTP_TTL"), 15*time.Minute),
		PasswordResetMaxAttempts: parseIntOrDefault(os.Getenv("PASSWORD_RESET_MAX_ATTEMPTS"), 5),
		Email: EmailConfig{
			SMTPHost:     getEnvOrDefault("SMTP_HOST", ""),
			SMTPPort:     getEnvOrDefault("SMTP_PORT", "587"),
			SMTPUsername: getEnvOrDefault("SMTP_USERNAME", ""),
			SMTPPassword: getEnvOrDefault("SMTP_PASSWORD", ""),
			FromAddress:  getEnvOrDefault("EMAIL_FROM_ADDRESS", getEnvOrDefault("SMTP_FROM", os.Getenv("SMTP_USERNAME"))),
			FromName:     getEnvOrDefault("EMAIL_FROM_NAME", "Settle"),
			SendTimeout:  parseDurationOrDefault(os.Getenv("EMAIL_SEND_TIMEOUT"), 10*time.Second),
			LogOnly:      getEnvOrDefault("EMAIL_LOG_ONLY", "true") == "true",
			CodeTTL:      parseDurationOrDefault(os.Getenv("EMAIL_CODE_TTL"), 15*time.Minute),
			CodeSecret:   codeSecret,
		},
		UploadDir: getEnvOrDefault("UPLOAD_DIR", "./uploads"),
		OCRApiKey: getEnvOrDefault("OCR_API_KEY", ""),
	}
	if err := AppConfig.Email.Validate(); err != nil {
		log.Fatalf("invalid email configuration: %v", err)
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

func parseIntOrDefault(val string, defaultVal int) int {
	if val == "" {
		return defaultVal
	}
	n, err := strconv.Atoi(val)
	if err != nil {
		return defaultVal
	}
	return n
}
