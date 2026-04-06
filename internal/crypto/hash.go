package crypto

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

// SHA256Hex computes the SHA-256 hash of a string and returns it as a lowercase hex string.
// Matches the web app's sha256Hex() in client.ts.
func SHA256Hex(input string) string {
	hash := sha256.Sum256([]byte(input))
	return hex.EncodeToString(hash[:])
}

// NormalizeLockCredential normalizes a lock credential for hashing:
// trims whitespace and lowercases.
// Matches the server's normalization behavior.
func NormalizeLockCredential(credential string) string {
	return strings.ToLower(strings.TrimSpace(credential))
}

// HashLockCredential normalizes and hashes a lock credential.
func HashLockCredential(credential string) string {
	return SHA256Hex(NormalizeLockCredential(credential))
}

// BuildLightLockHashes builds the lock credential hashes for a light lock.
// Given a sender email, it produces hashes for:
// - the full email
// - the local part (before @)
// - the local part with separators replaced by spaces
// - the local part with separators removed
// This matches the web app's upload-form.tsx logic.
func BuildLightLockHashes(email string) []string {
	email = strings.TrimSpace(strings.ToLower(email))
	localPart := email
	if idx := strings.Index(email, "@"); idx >= 0 {
		localPart = email[:idx]
	}

	variants := make(map[string]struct{})
	variants[email] = struct{}{}
	variants[localPart] = struct{}{}

	// Replace separators with spaces
	withSpaces := strings.NewReplacer(".", " ", "_", " ", "-", " ").Replace(localPart)
	if withSpaces != "" {
		variants[withSpaces] = struct{}{}
	}

	// Remove separators entirely
	withoutSeps := strings.NewReplacer(".", "", "_", "", "-", "").Replace(localPart)
	if withoutSeps != "" {
		variants[withoutSeps] = struct{}{}
	}

	// Remove empty strings
	delete(variants, "")

	hashes := make([]string, 0, len(variants))
	for v := range variants {
		hashes = append(hashes, SHA256Hex(v))
	}
	return hashes
}
