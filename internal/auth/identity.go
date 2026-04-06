package auth

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/mdpush-io/cli/internal/api"
)

// Identity holds the sender's email and name, used for light lock credential hashing.
type Identity struct {
	UserID string `json:"userId"`
	Email  string `json:"email"`
	Name   string `json:"name"`
}

// GetIdentity fetches the sender's identity from the server and caches it locally.
// Returns the cached version if available and the session user matches.
func GetIdentity(client *api.Client, userID string) (*Identity, error) {
	// Try cache first
	cached, err := loadCachedIdentity()
	if err == nil && cached != nil && cached.UserID == userID {
		return cached, nil
	}

	// Fetch from server
	resp, err := client.GetMe()
	if err != nil {
		return nil, fmt.Errorf("fetching identity: %w", err)
	}

	identity := &Identity{
		UserID: resp.UserID,
		Email:  resp.Email,
		Name:   resp.Name,
	}

	// Cache for next time (best-effort)
	_ = cacheIdentity(identity)

	return identity, nil
}

// ClearIdentityCache removes the cached identity file.
func ClearIdentityCache() {
	path, _ := identityPath()
	if path != "" {
		os.Remove(path)
	}
}

func identityPath() (string, error) {
	dir, err := configDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "identity.json"), nil
}

func loadCachedIdentity() (*Identity, error) {
	path, err := identityPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var identity Identity
	if err := json.Unmarshal(data, &identity); err != nil {
		return nil, err
	}

	return &identity, nil
}

func cacheIdentity(identity *Identity) error {
	path, err := identityPath()
	if err != nil {
		return err
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}

	data, err := json.MarshalIndent(identity, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0600)
}
