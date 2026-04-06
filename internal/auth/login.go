package auth

import (
	"fmt"

	"github.com/mdpush-io/cli/internal/api"
)

// LoginResult contains everything produced by a successful login.
type LoginResult struct {
	Session *Session
}

// Login performs returning-user authentication: calls the login endpoint
// and returns the session.
//
// This is pure logic — the caller handles all user interaction.
func Login(client *api.Client, verificationToken, email string) (*LoginResult, error) {
	resp, err := client.Login(api.LoginRequest{
		VerificationToken: verificationToken,
		DeviceLabel:       deviceLabel(),
	})
	if err != nil {
		return nil, fmt.Errorf("logging in: %w", err)
	}

	session := &Session{
		Token:     resp.SessionToken,
		UserID:    resp.UserID,
		Email:     email,
		ExpiresAt: resp.ExpiresAt,
	}

	return &LoginResult{
		Session: session,
	}, nil
}

// Persist saves the session.
// Call this after a successful Setup or Login.
func Persist(session *Session) error {
	if err := SaveSession(session); err != nil {
		return fmt.Errorf("saving session: %w", err)
	}
	return nil
}

// Logout clears the session on the server and locally.
func Logout(client *api.Client) error {
	// Try to logout on server (best-effort — may fail if token already expired)
	if client != nil {
		_, _ = client.Logout()
	}

	// Always clear local state
	if err := ClearSession(); err != nil {
		return fmt.Errorf("clearing session: %w", err)
	}
	return nil
}

// LoadAuth loads the saved session.
// Returns nil if not found (not an error).
func LoadAuth() (*Session, error) {
	session, err := LoadSession()
	if err != nil {
		return nil, fmt.Errorf("loading session: %w", err)
	}
	return session, nil
}

// AuthenticatedClient returns an API client with the saved session token,
// or an error if no valid session exists.
func AuthenticatedClient() (*api.Client, *Session, error) {
	session, err := LoadAuth()
	if err != nil {
		return nil, nil, err
	}
	if session == nil {
		return nil, nil, fmt.Errorf("not logged in — run `mdpush` to set up")
	}
	if !session.IsValid() {
		return nil, nil, fmt.Errorf("session expired — run `mdpush` to log in again")
	}

	client := api.NewClient().WithToken(session.Token)
	return client, session, nil
}
