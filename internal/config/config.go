package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/mdpush-io/cli/internal/api"
)

// Config holds user defaults for the share command.
// Zero values mean "use system default".
type Config struct {
	Lock       string `json:"lock,omitempty"`       // "light" or "strong"
	Expires    string `json:"expires,omitempty"`     // "1d", "7d", "never", etc.
	Views      int    `json:"views,omitempty"`       // max views (0 = unlimited)
	Theme      string `json:"theme,omitempty"`       // reading theme
	Category   string `json:"category,omitempty"`    // default category
	Project    string `json:"project,omitempty"`     // default project
	PwTheme    string `json:"pwTheme,omitempty"`     // password theme
}

// SystemDefaults returns the built-in system defaults.
func SystemDefaults() Config {
	return Config{
		Lock:    "light",
		PwTheme: "books",
		Theme:   "clean",
	}
}

// Effective returns the value with fallback: explicit > config > system default.
func (c *Config) Effective() Config {
	sys := SystemDefaults()
	return Config{
		Lock:     firstNonEmpty(c.Lock, sys.Lock),
		Expires:  c.Expires, // no system default (no expiry)
		Views:    c.Views,   // no system default (unlimited)
		Theme:    firstNonEmpty(c.Theme, sys.Theme),
		Category: c.Category, // no system default
		Project:  c.Project,  // no system default
		PwTheme:  firstNonEmpty(c.PwTheme, sys.PwTheme),
	}
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

// ValidKeys lists all settable config keys.
var ValidKeys = []string{"lock", "expires", "views", "theme", "category", "project", "pw-theme"}

// Set updates a single config key. Returns an error for invalid keys.
func (c *Config) Set(key, value string) error {
	switch key {
	case "lock":
		if value != "light" && value != "strong" {
			return fmt.Errorf("lock must be 'light' or 'strong'")
		}
		c.Lock = value
	case "expires":
		c.Expires = value
	case "views":
		if value == "" || value == "0" || value == "unlimited" {
			c.Views = 0
		} else {
			var v int
			if _, err := fmt.Sscanf(value, "%d", &v); err != nil || v < 0 {
				return fmt.Errorf("views must be a positive number or 'unlimited'")
			}
			c.Views = v
		}
	case "theme":
		valid := map[string]bool{"clean": true, "dark": true, "github": true, "technical": true}
		if !valid[value] {
			return fmt.Errorf("theme must be one of: clean, dark, github, technical")
		}
		c.Theme = value
	case "category":
		valid := map[string]bool{"new-feature": true, "debugging": true, "rambling": true, "": true}
		if !valid[value] {
			return fmt.Errorf("category must be one of: new-feature, debugging, rambling (or empty)")
		}
		c.Category = value
	case "project":
		c.Project = value
	case "pw-theme":
		valid := map[string]bool{"books": true, "animals": true, "dates": true}
		if !valid[value] {
			return fmt.Errorf("pw-theme must be one of: books, animals, dates")
		}
		c.PwTheme = value
	default:
		return fmt.Errorf("unknown config key %q — valid keys: %v", key, ValidKeys)
	}
	return nil
}

// --- File persistence ---

func configDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("finding home directory: %w", err)
	}
	return filepath.Join(home, ".config", "mdpush"), nil
}

func configPath() (string, error) {
	dir, err := configDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.json"), nil
}

// Load reads the config from disk. Returns empty config if file doesn't exist.
func Load() (*Config, error) {
	path, err := configPath()
	if err != nil {
		return &Config{}, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Config{}, nil
		}
		return nil, fmt.Errorf("reading config: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}

	return &cfg, nil
}

// Save writes the config to disk.
func Save(cfg *Config) error {
	path, err := configPath()
	if err != nil {
		return err
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}

	return os.WriteFile(path, data, 0600)
}

// Reset clears the config file.
func Reset() error {
	return Save(&Config{})
}

// --- Server sync ---

// PushToServer syncs local config to the server's /api/settings endpoint.
func PushToServer(client *api.Client, cfg *Config) error {
	req := api.UpdateSettingsRequest{}

	if cfg.Lock != "" {
		req.DefaultLockType = &cfg.Lock
	}
	if cfg.Expires != "" {
		req.DefaultExpiration = &cfg.Expires
	}
	if cfg.Category != "" {
		req.DefaultCategory = &cfg.Category
	}
	if cfg.PwTheme != "" {
		req.PasswordTheme = &cfg.PwTheme
	}
	if cfg.Views > 0 {
		req.DefaultMaxViews = &cfg.Views
	}

	_, err := client.UpdateSettings(req)
	return err
}

// PullFromServer fetches server settings and merges into local config.
func PullFromServer(client *api.Client) (*Config, error) {
	settings, err := client.GetSettings()
	if err != nil {
		return nil, err
	}

	cfg := &Config{
		Lock:    settings.DefaultLockType,
		Expires: settings.DefaultExpiration,
		PwTheme: settings.PasswordTheme,
	}

	if settings.DefaultMaxViews != nil {
		cfg.Views = *settings.DefaultMaxViews
	}
	if settings.DefaultCategory != nil {
		cfg.Category = *settings.DefaultCategory
	}

	return cfg, nil
}
