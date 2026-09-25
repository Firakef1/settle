package service

import (
	"golang.org/x/crypto/bcrypt"
)

// HashService handles password hashing and password comparison.
type HashService struct{}

// NewHashService creates a new instance of HashService.
func NewHashService() *HashService {
	return &HashService{}
}

// Hash returns the bcrypt hash of the unencrypted password string.
func (s *HashService) Hash(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

// Compare returns true if the plain password matches the bcrypt hash, false otherwise.
func (s *HashService) Compare(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}
