package config

import (
	"testing"
)

func TestSetValidKeys(t *testing.T) {
	cfg := &Config{}

	tests := []struct {
		key, value string
	}{
		{"lock", "light"},
		{"lock", "strong"},
		{"expires", "7d"},
		{"views", "10"},
		{"views", "unlimited"},
		{"theme", "dark"},
		{"category", "debugging"},
		{"project", "my-project"},
		{"pw-theme", "dates"},
	}

	for _, tt := range tests {
		if err := cfg.Set(tt.key, tt.value); err != nil {
			t.Errorf("Set(%q, %q) unexpected error: %v", tt.key, tt.value, err)
		}
	}
}

func TestSetInvalidValues(t *testing.T) {
	cfg := &Config{}

	tests := []struct {
		key, value string
	}{
		{"lock", "invalid"},
		{"theme", "badtheme"},
		{"category", "nonexistent"},
		{"pw-theme", "numbers"},
		{"unknown-key", "value"},
	}

	for _, tt := range tests {
		if err := cfg.Set(tt.key, tt.value); err == nil {
			t.Errorf("Set(%q, %q) expected error, got nil", tt.key, tt.value)
		}
	}
}

func TestEffectiveDefaults(t *testing.T) {
	cfg := &Config{} // all empty
	eff := cfg.Effective()

	if eff.Lock != "light" {
		t.Fatalf("expected default lock=light, got %q", eff.Lock)
	}
	if eff.PwTheme != "books" {
		t.Fatalf("expected default pw-theme=books, got %q", eff.PwTheme)
	}
	if eff.Theme != "clean" {
		t.Fatalf("expected default theme=clean, got %q", eff.Theme)
	}
}

func TestEffectiveOverrides(t *testing.T) {
	cfg := &Config{Lock: "strong", PwTheme: "dates"}
	eff := cfg.Effective()

	if eff.Lock != "strong" {
		t.Fatalf("expected lock=strong, got %q", eff.Lock)
	}
	if eff.PwTheme != "dates" {
		t.Fatalf("expected pw-theme=dates, got %q", eff.PwTheme)
	}
	// Theme should still fall back to system default
	if eff.Theme != "clean" {
		t.Fatalf("expected theme=clean, got %q", eff.Theme)
	}
}

func TestSaveLoadRoundTrip(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	cfg := &Config{
		Lock:     "strong",
		Expires:  "7d",
		Views:    10,
		Category: "debugging",
		Project:  "test-project",
		PwTheme:  "dates",
	}

	if err := Save(cfg); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if loaded.Lock != cfg.Lock {
		t.Fatalf("lock: got %q, want %q", loaded.Lock, cfg.Lock)
	}
	if loaded.Expires != cfg.Expires {
		t.Fatalf("expires: got %q, want %q", loaded.Expires, cfg.Expires)
	}
	if loaded.Views != cfg.Views {
		t.Fatalf("views: got %d, want %d", loaded.Views, cfg.Views)
	}
	if loaded.Category != cfg.Category {
		t.Fatalf("category: got %q, want %q", loaded.Category, cfg.Category)
	}
	if loaded.Project != cfg.Project {
		t.Fatalf("project: got %q, want %q", loaded.Project, cfg.Project)
	}
	if loaded.PwTheme != cfg.PwTheme {
		t.Fatalf("pw-theme: got %q, want %q", loaded.PwTheme, cfg.PwTheme)
	}
}

func TestLoadNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	// Should return empty config, not error
	if cfg.Lock != "" {
		t.Fatalf("expected empty lock, got %q", cfg.Lock)
	}
}

func TestReset(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	Save(&Config{Lock: "strong", Project: "test"})
	Reset()

	cfg, _ := Load()
	if cfg.Lock != "" || cfg.Project != "" {
		t.Fatalf("expected empty config after reset, got lock=%q project=%q", cfg.Lock, cfg.Project)
	}
}
