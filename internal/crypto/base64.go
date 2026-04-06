package crypto

import (
	"encoding/base64"
)

// Base64Encode encodes bytes to standard base64 (with padding).
// This matches the web app's btoa() output.
func Base64Encode(data []byte) string {
	return base64.StdEncoding.EncodeToString(data)
}

// Base64Decode decodes standard base64 or base64url (auto-detects).
// Handles both padded and unpadded input.
func Base64Decode(s string) ([]byte, error) {
	// Try standard base64 first
	if data, err := base64.StdEncoding.DecodeString(s); err == nil {
		return data, nil
	}
	// Try base64url (with or without padding)
	if data, err := base64.URLEncoding.DecodeString(s); err == nil {
		return data, nil
	}
	// Try unpadded variants
	if data, err := base64.RawStdEncoding.DecodeString(s); err == nil {
		return data, nil
	}
	return base64.RawURLEncoding.DecodeString(s)
}

// Base64URLEncode encodes bytes to base64url without padding.
// Used for URL fragments (the doc key in #fragment).
func Base64URLEncode(data []byte) string {
	return base64.RawURLEncoding.EncodeToString(data)
}

// Base64URLDecode decodes base64url (padded or unpadded).
func Base64URLDecode(s string) ([]byte, error) {
	// Try with padding first, then without
	if data, err := base64.URLEncoding.DecodeString(s); err == nil {
		return data, nil
	}
	return base64.RawURLEncoding.DecodeString(s)
}

// KeyToFragment converts a raw key to a base64url string for use in URL fragments.
// Matches the web app's keyToFragment() function.
func KeyToFragment(key []byte) string {
	return Base64URLEncode(key)
}

// ParseKeyFromFragment extracts the document key from a URL hash fragment.
// Matches the web app's parseKeyFromFragment() function.
func ParseKeyFromFragment(hash string) ([]byte, error) {
	fragment := hash
	if len(hash) > 0 && hash[0] == '#' {
		fragment = hash[1:]
	}
	return Base64URLDecode(fragment)
}
