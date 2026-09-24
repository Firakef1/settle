// Package deps pins scaffold dependencies until feature packages import them.
package deps

import (
	_ "github.com/gin-gonic/gin"
	_ "github.com/golang-jwt/jwt/v5"
	_ "github.com/google/uuid"
	_ "github.com/joho/godotenv"
	_ "github.com/lib/pq"
	_ "github.com/stretchr/testify"
	_ "golang.org/x/crypto/bcrypt"
)
