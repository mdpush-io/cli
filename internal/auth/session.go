package auth

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Session represents the persisted session state.
type Session struct {
	Token     string `json:"token"`
	UserID    string `json:"userId"`
	Email     string `json:"email"`
	ExpiresAt string `json:"expiresAt"`
}

// IsExpired returns true if the session has expired.
func (s *Session) IsExpired() bool {
	t, err := time.Parse(time.RFC3339, s.ExpiresAt)
	if err != nil {
		return true
	}
	return time.Now().After(t)
}

// IsValid returns true if the session has a token and is not expired.
func (s *Session) IsValid() bool {
	return s.Token != "" && !s.IsExpired()
}

// configDir returns the mdpush config directory path.
func configDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("finding home directory: %w", err)
	}
	return filepath.Join(home, ".config", "mdpush"), nil
}

// sessionPath returns the full path to the session file.
func sessionPath() (string, error) {
	dir, err := configDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "session.json"), nil
}

// LoadSession reads the session from disk.
// Returns nil (no error) if the file doesn't exist.
func LoadSession() (*Session, error) {
	path, err := sessionPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading session file: %w", err)
	}

	var session Session
	if err := json.Unmarshal(data, &session); err != nil {
		return nil, fmt.Errorf("parsing session file: %w", err)
	}

	return &session, nil
}

// SaveSession writes the session to disk, creating the config directory if needed.
func SaveSession(session *Session) error {
	path, err := sessionPath()
	if err != nil {
		return err
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}

	data, err := json.MarshalIndent(session, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling session: %w", err)
	}

	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("writing session file: %w", err)
	}

	return nil
}

// ClearSession deletes the session file.
func ClearSession() error {
	path, err := sessionPath()
	if err != nil {
		return err
	}

	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("removing session file: %w", err)
	}

	return nil
}
