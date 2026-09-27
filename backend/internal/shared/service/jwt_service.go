package service

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"github.com/Firakef1/settle/backend/internal/shared/config"
)

var (
	ErrInvalidToken = errors.New("invalid or expired token")
)

// Claims represents the JWT custom claims payload.
type Claims struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	OrgID  string `json:"org_id,omitempty"`
	Role   string `json:"role,omitempty"`
	jwt.RegisteredClaims
}

// JWTService handles JWT token generation and verification.
type JWTService struct {
	secretKey []byte
	tokenTTL  time.Duration
}

// NewJWTService initializes JWTService with secret key from shared config.
func NewJWTService() *JWTService {
	secret := config.AppConfig.SecretKey
	if secret == "" {
		secret = "settle_default_development_secret_key_change_in_prod"
	}
	ttl := config.AppConfig.AccessTokenTTL
	if ttl == 0 {
		ttl = 24 * time.Hour
	}
	return &JWTService{
		secretKey: []byte(secret),
		tokenTTL:  ttl,
	}
}

// NewJWTServiceWithSecret initializes JWTService with explicit secret key.
func NewJWTServiceWithSecret(secret string) *JWTService {
	return &JWTService{
		secretKey: []byte(secret),
		tokenTTL:  24 * time.Hour,
	}
}

// Generate generates a signed JWT token with a unique JTI.
func (s *JWTService) Generate(userID, email, orgID, role string) (string, error) {
	jti := uuid.New().String()
	claims := Claims{
		UserID: userID,
		Email:  email,
		OrgID:  orgID,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        jti,
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(s.tokenTTL)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Subject:   userID,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.secretKey)
}

// Verify parses and verifies a JWT token string.
func (s *JWTService) Verify(tokenStr string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidToken
		}
		return s.secretKey, nil
	})

	if err != nil || !token.Valid {
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(*Claims)
	if !ok {
		return nil, ErrInvalidToken
	}

	return claims, nil
}
