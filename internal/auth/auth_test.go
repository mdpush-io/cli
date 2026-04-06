package auth

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// --- Session persistence tests ---

func TestSessionSaveAndLoad(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	session := &Session{
		Token:     "test-token-abc",
		UserID:    "uuid-123",
		Email:     "test@example.com",
		ExpiresAt: time.Now().Add(24 * time.Hour).Format(time.RFC3339),
	}

	if err := SaveSession(session); err != nil {
		t.Fatalf("SaveSession: %v", err)
	}

	// Verify file exists with correct permissions
	path := filepath.Join(tmpDir, ".config", "mdpush", "session.json")
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("session file not found: %v", err)
	}
	if info.Mode().Perm() != 0600 {
		t.Fatalf("expected 0600 permissions, got %o", info.Mode().Perm())
	}

	// Load it back
	loaded, err := LoadSession()
	if err != nil {
		t.Fatalf("LoadSession: %v", err)
	}
	if loaded == nil {
		t.Fatal("expected session, got nil")
	}
	if loaded.Token != session.Token {
		t.Fatalf("token: got %q, want %q", loaded.Token, session.Token)
	}
	if loaded.UserID != session.UserID {
		t.Fatalf("userId: got %q, want %q", loaded.UserID, session.UserID)
	}
	if loaded.Email != session.Email {
		t.Fatalf("email: got %q, want %q", loaded.Email, session.Email)
	}
}

func TestLoadSessionNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	session, err := LoadSession()
	if err != nil {
		t.Fatalf("LoadSession: %v", err)
	}
	if session != nil {
		t.Fatal("expected nil session when file doesn't exist")
	}
}

func TestClearSession(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	session := &Session{
		Token:     "test-token",
		UserID:    "uuid",
		ExpiresAt: time.Now().Add(time.Hour).Format(time.RFC3339),
	}
	SaveSession(session)

	if err := ClearSession(); err != nil {
		t.Fatalf("ClearSession: %v", err)
	}

	loaded, _ := LoadSession()
	if loaded != nil {
		t.Fatal("expected nil after clear")
	}
}

func TestClearSessionIdempotent(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	if err := ClearSession(); err != nil {
		t.Fatalf("ClearSession on empty: %v", err)
	}
}

// --- Session validity tests ---

func TestSessionIsValid(t *testing.T) {
	valid := &Session{
		Token:     "tok",
		ExpiresAt: time.Now().Add(time.Hour).Format(time.RFC3339),
	}
	if !valid.IsValid() {
		t.Fatal("expected valid session")
	}

	expired := &Session{
		Token:     "tok",
		ExpiresAt: time.Now().Add(-time.Hour).Format(time.RFC3339),
	}
	if expired.IsValid() {
		t.Fatal("expected invalid (expired) session")
	}

	noToken := &Session{
		Token:     "",
		ExpiresAt: time.Now().Add(time.Hour).Format(time.RFC3339),
	}
	if noToken.IsValid() {
		t.Fatal("expected invalid (no token) session")
	}
}

// --- Identity cache tests ---

func TestIdentityCacheRoundTrip(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	id := &Identity{
		UserID: "uuid-123",
		Email:  "test@example.com",
		Name:   "test",
	}

	if err := cacheIdentity(id); err != nil {
		t.Fatalf("cacheIdentity: %v", err)
	}

	loaded, err := loadCachedIdentity()
	if err != nil {
		t.Fatalf("loadCachedIdentity: %v", err)
	}
	if loaded.Email != id.Email {
		t.Fatalf("email: got %q, want %q", loaded.Email, id.Email)
	}
	if loaded.UserID != id.UserID {
		t.Fatalf("userId: got %q, want %q", loaded.UserID, id.UserID)
	}
}

func TestIdentityCacheNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	loaded, err := loadCachedIdentity()
	if loaded != nil || err == nil {
		t.Fatal("expected nil/error when no cache exists")
	}
}

// --- LoadAuth integration ---

func TestLoadAuthNoSession(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	session, err := LoadAuth()
	if err != nil {
		t.Fatalf("LoadAuth: %v", err)
	}
	if session != nil {
		t.Fatal("expected nil when nothing is saved")
	}
}

func TestAuthenticatedClientNoSession(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	_, _, err := AuthenticatedClient()
	if err == nil {
		t.Fatal("expected error when not logged in")
	}
}

func TestAuthenticatedClientExpired(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	session := &Session{
		Token:     "tok",
		UserID:    "uuid",
		ExpiresAt: time.Now().Add(-time.Hour).Format(time.RFC3339),
	}
	SaveSession(session)

	_, _, err := AuthenticatedClient()
	if err == nil {
		t.Fatal("expected error for expired session")
	}
}
