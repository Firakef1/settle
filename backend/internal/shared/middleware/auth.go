package middleware

import (
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

const (
	ContextUserIDKey = "user_id"
	ContextOrgIDKey  = "org_id"
	ContextRoleKey   = "role"
)

type CustomClaims struct {
	UserID string `json:"user_id"`
	OrgID  string `json:"org_id,omitempty"`
	Role   string `json:"role,omitempty"`
	jwt.RegisteredClaims
}

func getJWTSecret() []byte {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		secret = "dev-secret"
	}
	return []byte(secret)
}

// AuthRequired validates the JWT Bearer token from the Authorization header.
func AuthRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "authorization header is required"})
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid authorization header format"})
			return
		}

		tokenStr := parts[1]
		claims := &CustomClaims{}

		token, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			return getJWTSecret(), nil
		})

		if err != nil || !token.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired token"})
			return
		}

		c.Set(ContextUserIDKey, claims.UserID)
		if claims.OrgID != "" {
			c.Set(ContextOrgIDKey, claims.OrgID)
		}
		if claims.Role != "" {
			c.Set(ContextRoleKey, claims.Role)
		}

		c.Next()
	}
}

// RequireRole checks if the authenticated user has one of the allowed roles.
func RequireRole(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userRole := GetRole(c)
		if userRole == "" {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "forbidden: no role assigned"})
			return
		}

		for _, r := range roles {
			if strings.EqualFold(r, userRole) {
				c.Next()
				return
			}
		}

		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "forbidden: insufficient permissions"})
	}
}

// GetUserID retrieves the user ID from the Gin context.
func GetUserID(c *gin.Context) string {
	if val, exists := c.Get(ContextUserIDKey); exists {
		if s, ok := val.(string); ok {
			return s
		}
	}
	return ""
}

// GetOrgID retrieves the organization ID from the Gin context.
func GetOrgID(c *gin.Context) string {
	if val, exists := c.Get(ContextOrgIDKey); exists {
		if s, ok := val.(string); ok {
			return s
		}
	}
	return ""
}

// GetRole retrieves the user role from the Gin context.
func GetRole(c *gin.Context) string {
	if val, exists := c.Get(ContextRoleKey); exists {
		if s, ok := val.(string); ok {
			return s
		}
	}
	return ""
}
