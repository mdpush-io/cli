package keystore

import (
	"bytes"
	"path/filepath"
	"testing"
)

func TestPutGetRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "keys.json")
	store := NewWithPath(path)

	key := []byte("01234567890123456789012345678901") // 32 bytes
	if err := store.Put("doc1", key); err != nil {
		t.Fatalf("Put: %v", err)
	}

	got := store.Get("doc1")
	if !bytes.Equal(got, key) {
		t.Fatalf("Get: got %x, want %x", got, key)
	}
}

func TestGetMissing(t *testing.T) {
	store := NewWithPath(filepath.Join(t.TempDir(), "keys.json"))
	if got := store.Get("nonexistent"); got != nil {
		t.Fatalf("expected nil for missing key, got %x", got)
	}
}

func TestPersistence(t *testing.T) {
	path := filepath.Join(t.TempDir(), "keys.json")
	store := NewWithPath(path)

	key := []byte("abcdefghijklmnopqrstuvwxyz012345")
	store.Put("doc1", key)
	store.Put("doc2", key)

	// Load from a fresh store instance
	store2 := NewWithPath(path)
	if err := store2.Load(); err != nil {
		t.Fatalf("Load: %v", err)
	}

	if store2.Len() != 2 {
		t.Fatalf("expected 2 keys, got %d", store2.Len())
	}
	if got := store2.Get("doc1"); !bytes.Equal(got, key) {
		t.Fatalf("persistence round-trip failed")
	}
}

func TestDelete(t *testing.T) {
	path := filepath.Join(t.TempDir(), "keys.json")
	store := NewWithPath(path)

	key := []byte("01234567890123456789012345678901")
	store.Put("doc1", key)
	store.Delete("doc1")

	if got := store.Get("doc1"); got != nil {
		t.Fatalf("expected nil after delete, got %x", got)
	}
	if store.Len() != 0 {
		t.Fatalf("expected 0 keys, got %d", store.Len())
	}
}

func TestLoadMissingFile(t *testing.T) {
	store := NewWithPath(filepath.Join(t.TempDir(), "nonexistent", "keys.json"))
	if err := store.Load(); err != nil {
		t.Fatalf("Load on missing file should not error: %v", err)
	}
	if store.Len() != 0 {
		t.Fatalf("expected 0 keys, got %d", store.Len())
	}
}
