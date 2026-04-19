package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strings"
)

// APITokenPrefix identifies API tokens. Useful for secret scanners (e.g., GitHub).
const APITokenPrefix = "bdg_"

// GenerateAPIToken returns a new raw token and its SHA-256 hash. Only the hash
// should be stored; the raw token must be shown to the user once.
func GenerateAPIToken() (raw, hash string, err error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", "", fmt.Errorf("failed to generate api token: %w", err)
	}
	raw = APITokenPrefix + base64.RawURLEncoding.EncodeToString(bytes)
	hash = HashAPIToken(raw)
	return raw, hash, nil
}

// HashAPIToken returns the hex-encoded SHA-256 of the raw token.
func HashAPIToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

// IsAPITokenFormat checks that a string has the expected token prefix.
func IsAPITokenFormat(s string) bool {
	return strings.HasPrefix(s, APITokenPrefix) && len(s) > len(APITokenPrefix)
}
