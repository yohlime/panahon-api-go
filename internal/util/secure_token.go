package util

import (
	"crypto/rand"
	"encoding/base64"
	"strings"
)

// GenerateSecureToken returns a URL-safe random token with nBytes of entropy.
// Use this for one-time links (registration, password reset), not RandomString.
func GenerateSecureToken(nBytes int) (string, error) {
	b := make([]byte, nBytes)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// NormalizeEmail trims whitespace and lowercases an email address so lookups
// and uniqueness checks are case-insensitive.
func NormalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}
