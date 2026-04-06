package keystore

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// Store manages local doc key persistence.
// Keys are stored at ~/.config/mdpush/keys.json as { docId: base64(key) }.
type Store struct {
	mu   sync.Mutex
	keys map[string]string // docId → base64(key)
	path string
}

// New creates a Store pointed at the default config location.
func New() (*Store, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("finding home directory: %w", err)
	}
	path := filepath.Join(home, ".config", "mdpush", "keys.json")
	return &Store{keys: make(map[string]string), path: path}, nil
}

// NewWithPath creates a Store at a custom path (for testing).
func NewWithPath(path string) *Store {
	return &Store{keys: make(map[string]string), path: path}
}

// Load reads the key store from disk. Missing file is not an error.
func (s *Store) Load() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			s.keys = make(map[string]string)
			return nil
		}
		return fmt.Errorf("reading key store: %w", err)
	}

	keys := make(map[string]string)
	if err := json.Unmarshal(data, &keys); err != nil {
		return fmt.Errorf("parsing key store: %w", err)
	}
	s.keys = keys
	return nil
}

// Save writes the key store to disk.
func (s *Store) Save() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.saveLocked()
}

func (s *Store) saveLocked() error {
	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}

	data, err := json.MarshalIndent(s.keys, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling key store: %w", err)
	}

	return os.WriteFile(s.path, data, 0600)
}

// Put stores a doc key and persists to disk.
func (s *Store) Put(docID string, key []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.keys[docID] = base64.StdEncoding.EncodeToString(key)
	return s.saveLocked()
}

// Get retrieves a doc key. Returns nil if not found.
func (s *Store) Get(docID string) []byte {
	s.mu.Lock()
	defer s.mu.Unlock()

	encoded, ok := s.keys[docID]
	if !ok {
		return nil
	}

	key, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil
	}
	return key
}

// Delete removes a doc key and persists to disk.
func (s *Store) Delete(docID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.keys, docID)
	return s.saveLocked()
}

// Len returns the number of stored keys.
func (s *Store) Len() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.keys)
}
