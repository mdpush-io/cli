package crypto

import (
	"crypto/rand"
	"fmt"
	"io"
)

// GenerateDocKey generates a random 256-bit (32-byte) document encryption key.
// Matches the web app's generateDocKey().
func GenerateDocKey() ([]byte, error) {
	key := make([]byte, keySize)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		return nil, fmt.Errorf("generating doc key: %w", err)
	}
	return key, nil
}
