package api

import (
	"crypto/rand"
	"encoding/base32"
	"fmt"
	"strings"
)

// RandomSecret returns 128 bits of randomness as lowercase base32, suitable
// for a generated password.
func RandomSecret() (string, error) {
	raw := make([]byte, 16)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("generate secret: %w", err)
	}
	return strings.ToLower(base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(raw)), nil
}
