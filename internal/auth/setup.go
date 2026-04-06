package auth

import (
	"fmt"
	"os"
	"runtime"

	"github.com/mdpush-io/cli/internal/api"
)

// SetupResult contains everything produced by a successful first-run registration.
type SetupResult struct {
	Session *Session
}

// Setup performs first-run registration: verifies the email and creates an account.
//
// This is pure logic — the caller handles all user interaction
// (prompting for email, code) and passes the values in.
func Setup(client *api.Client, verificationToken, email string) (*SetupResult, error) {
	resp, err := client.Register(api.RegisterRequest{
		VerificationToken: verificationToken,
		DeviceLabel:       deviceLabel(),
	})
	if err != nil {
		return nil, fmt.Errorf("registering account: %w", err)
	}

	session := &Session{
		Token:     resp.SessionToken,
		UserID:    resp.UserID,
		Email:     email,
		ExpiresAt: resp.ExpiresAt,
	}

	return &SetupResult{
		Session: session,
	}, nil
}

// deviceLabel returns a human-readable device label for session tracking.
func deviceLabel() string {
	hostname, _ := os.Hostname()
	if hostname == "" {
		hostname = "unknown"
	}
	return fmt.Sprintf("CLI on %s (%s/%s)", hostname, runtime.GOOS, runtime.GOARCH)
}
