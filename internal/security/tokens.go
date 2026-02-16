package security

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
)

func GeneratePrefixedToken(prefix string, byteLength int) (string, error) {
	if byteLength <= 0 {
		byteLength = 18
	}

	buffer := make([]byte, byteLength)
	if _, err := rand.Read(buffer); err != nil {
		return "", fmt.Errorf("generate random bytes: %w", err)
	}

	trimmedPrefix := strings.TrimSpace(prefix)
	if trimmedPrefix == "" {
		return hex.EncodeToString(buffer), nil
	}

	return fmt.Sprintf("%s_%s", trimmedPrefix, hex.EncodeToString(buffer)), nil
}

func DigestSHA256(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}
