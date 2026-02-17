package utils

import (
	"strings"

	"golang.org/x/crypto/bcrypt"
)

// HashClientSecret hashes a client secret using bcrypt
func HashClientSecret(secret string) (string, error) {
	hashedBytes, err := bcrypt.GenerateFromPassword([]byte(secret), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hashedBytes), nil
}

// VerifyClientSecret verifies a plain secret against a hashed secret
func VerifyClientSecret(plainSecret, hashedSecret string) error {
	return bcrypt.CompareHashAndPassword([]byte(hashedSecret), []byte(plainSecret))
}

// ValidateScope checks if requested scopes are within allowed scopes
func ValidateScope(requested, allowed string) bool {
	if requested == "" {
		return true // No specific scope requested, allow
	}

	if allowed == "" {
		return false // No scopes allowed for this client
	}

	// Parse scopes (space-separated)
	requestedScopes := strings.Fields(requested)
	allowedScopes := strings.Fields(allowed)

	// Create a map for fast lookup
	allowedMap := make(map[string]bool)
	for _, scope := range allowedScopes {
		allowedMap[scope] = true
	}

	// Check if all requested scopes are in allowed scopes
	for _, scope := range requestedScopes {
		if !allowedMap[scope] {
			return false
		}
	}

	return true
}

// ParseScopes converts space-separated scope string to slice
func ParseScopes(scopes string) []string {
	if scopes == "" {
		return []string{}
	}
	return strings.Fields(scopes)
}
