package cmd

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/mdpush-io/cli/internal/api"
	"github.com/mdpush-io/cli/internal/auth"
	"github.com/mdpush-io/cli/internal/config"
	"github.com/mdpush-io/cli/internal/crypto"
	"github.com/mdpush-io/cli/internal/keystore"
	"github.com/spf13/cobra"
)

var (
	flagLock     string
	flagTheme    string
	flagExpires  string
	flagViews    int
	flagCategory string
	flagProject  string
	flagPassword string
	flagPwTheme  string
	flagTitle    string
)

var shareCmd = &cobra.Command{
	Use:   "share <file.md>",
	Short: "Encrypt and share a markdown document",
	Long: `Encrypt a markdown file client-side and upload it to mdpush.
Returns a URL with the decryption key in the fragment — the server never sees it.

Use "-" to read from stdin:
  cat report.md | mdpush share -
  echo "# Quick note" | mdpush share - --title "Quick note"`,
	Args:          cobra.ExactArgs(1),
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE:          runShare,
}

func init() {
	// No hardcoded defaults — they come from config. Empty string / zero = "not set by flag".
	shareCmd.Flags().StringVar(&flagLock, "lock", "", "lock type: light or strong")
	shareCmd.Flags().StringVar(&flagTheme, "theme", "", "reading theme: clean, dark, github, technical")
	shareCmd.Flags().StringVar(&flagExpires, "expires", "", "expiration: 1h, 12h, 3d, 7d, 30d, never")
	shareCmd.Flags().IntVar(&flagViews, "views", 0, "max view count (0 = unlimited)")
	shareCmd.Flags().StringVar(&flagCategory, "category", "", "category: new-feature, debugging, rambling")
	shareCmd.Flags().StringVar(&flagProject, "project", "", "project name")
	shareCmd.Flags().StringVar(&flagPassword, "password", "", "custom strong lock password")
	shareCmd.Flags().StringVar(&flagPwTheme, "pw-theme", "", "password theme: books, animals, dates")
	shareCmd.Flags().StringVar(&flagTitle, "title", "", "document title (default: extracted from # heading or filename)")

	rootCmd.AddCommand(shareCmd)
}

func runShare(cmd *cobra.Command, args []string) error {
	filePath := args[0]

	// 1. Load auth
	client, session, err := auth.AuthenticatedClient()
	if err != nil {
		return fmt.Errorf("not logged in: %w\nRun 'mdpush' to set up your account", err)
	}

	// 1b. Resolve config defaults — flag > config > system default
	cfg, _ := config.Load()
	defaults := cfg.Effective()

	if flagLock == "" {
		flagLock = defaults.Lock
	}
	if flagTheme == "" {
		flagTheme = defaults.Theme
	}
	if flagExpires == "" && defaults.Expires != "" {
		flagExpires = defaults.Expires
	}
	if flagViews == 0 && defaults.Views > 0 {
		flagViews = defaults.Views
	}
	if flagCategory == "" && defaults.Category != "" {
		flagCategory = defaults.Category
	}
	if flagProject == "" && defaults.Project != "" {
		flagProject = defaults.Project
	}
	if flagPwTheme == "" {
		flagPwTheme = defaults.PwTheme
	}

	// 2. Read the file (or stdin with "-")
	var content []byte
	if filePath == "-" {
		content, err = io.ReadAll(os.Stdin)
		if err != nil {
			return fmt.Errorf("reading stdin: %w", err)
		}
	} else {
		content, err = os.ReadFile(filePath)
		if err != nil {
			return fmt.Errorf("reading file: %w", err)
		}
	}

	// 3. Build document metadata
	title := flagTitle
	if title == "" {
		title = buildTitle(filePath, string(content))
	}
	doc := crypto.Document{
		Title:    title,
		Content:  string(content),
		Category: flagCategory,
		Project:  flagProject,
	}

	// 4. Generate doc key
	docKey, err := crypto.GenerateDocKey()
	if err != nil {
		return fmt.Errorf("generating doc key: %w", err)
	}

	// 5. Encrypt payload
	encryptedPayload, err := crypto.EncryptPayload(doc, docKey)
	if err != nil {
		return fmt.Errorf("encrypting payload: %w", err)
	}

	// 6. Build lock credentials
	lockType := flagLock
	var lockCredentialHashes []string
	var displayPassword string

	switch lockType {
	case "light":
		hashes, err := buildLightLockHashes(client, session)
		if err != nil {
			return err
		}
		lockCredentialHashes = hashes

	case "strong":
		password, hash, err := resolveStrongLockPassword(client)
		if err != nil {
			return err
		}
		lockCredentialHashes = []string{hash}
		displayPassword = password

	default:
		return fmt.Errorf("invalid lock type %q — use 'light' or 'strong'", lockType)
	}

	// 7. Parse expiration
	var expiresIn *int
	if flagExpires != "" && flagExpires != "never" {
		seconds, err := parseExpiration(flagExpires)
		if err != nil {
			return err
		}
		expiresIn = &seconds
	}

	var maxViews *int
	if flagViews > 0 {
		maxViews = &flagViews
	}

	// 8. Upload
	readingTheme := flagTheme
	if readingTheme == "" {
		readingTheme = "clean"
	}

	resp, err := client.CreateDoc(api.CreateDocRequest{
		EncryptedPayload:     encryptedPayload,
		LockType:             lockType,
		LockCredentialHashes: lockCredentialHashes,
		ReadingTheme:         readingTheme,
		ExpiresIn:            expiresIn,
		MaxViews:             maxViews,
	})
	if err != nil {
		return fmt.Errorf("uploading document: %w", err)
	}

	// 9. Save doc key to local store (best-effort — dashboard needs it)
	if ks, err := keystore.New(); err == nil {
		_ = ks.Load()
		_ = ks.Put(resp.ID, docKey)
	}

	// 10. Build share URL with key in fragment
	fragment := crypto.KeyToFragment(docKey)
	shareURL := fmt.Sprintf("https://www.mdpush.io/d/%s#%s", resp.ID, fragment)

	// 11. Print results
	fmt.Println()
	fmt.Printf("  ✓ Uploaded: %s\n", title)
	fmt.Printf("  🔗 %s\n", shareURL)

	if lockType == "light" {
		fmt.Println("  🔒 Light lock — readers answer \"Who sent you this?\"")
	} else {
		fmt.Printf("  🔒 Password: %s\n", displayPassword)
	}

	if flagCategory != "" || flagProject != "" {
		labels := []string{}
		if flagCategory != "" {
			labels = append(labels, flagCategory)
		}
		if flagProject != "" {
			labels = append(labels, flagProject)
		}
		fmt.Printf("  🏷️  %s\n", strings.Join(labels, " · "))
	}

	if expiresIn != nil || maxViews != nil {
		parts := []string{}
		if expiresIn != nil {
			parts = append(parts, fmt.Sprintf("expires in %s", flagExpires))
		}
		if maxViews != nil {
			parts = append(parts, fmt.Sprintf("max %d views", *maxViews))
		}
		fmt.Printf("  ⏳ %s\n", strings.Join(parts, " or "))
	}

	fmt.Println()

	return nil
}

// buildTitle extracts a title from the file content or falls back to filename.
func buildTitle(filePath, content string) string {
	// Try to extract from first # heading
	re := regexp.MustCompile(`(?m)^#\s+(.+)`)
	if matches := re.FindStringSubmatch(content); len(matches) > 1 {
		return strings.TrimSpace(matches[1])
	}
	return filepath.Base(filePath)
}

// buildLightLockHashes builds credential hashes from the sender's identity.
func buildLightLockHashes(client *api.Client, session *auth.Session) ([]string, error) {
	identity, err := auth.GetIdentity(client, session.UserID)
	if err != nil {
		return nil, fmt.Errorf("fetching sender identity for light lock: %w", err)
	}

	if identity.Email == "" {
		return nil, fmt.Errorf("could not determine your email for light lock")
	}

	return crypto.BuildLightLockHashes(identity.Email), nil
}

// resolveStrongLockPassword returns the password and its hash.
func resolveStrongLockPassword(client *api.Client) (password, hash string, err error) {
	if flagPassword != "" {
		// Custom password
		normalized := crypto.NormalizeLockCredential(flagPassword)
		return flagPassword, crypto.SHA256Hex(normalized), nil
	}

	// Generate themed password from server
	resp, err := client.GeneratePassword(flagPwTheme)
	if err != nil {
		return "", "", fmt.Errorf("generating password: %w", err)
	}

	return resp.Password, resp.Hash, nil
}

// parseExpiration converts a duration string like "3d" or "12h" to seconds.
func parseExpiration(s string) (int, error) {
	s = strings.TrimSpace(strings.ToLower(s))

	if s == "" || s == "never" {
		return 0, fmt.Errorf("no expiration")
	}

	if len(s) < 2 {
		return 0, fmt.Errorf("invalid expiration %q — use format like 1h, 3d, 7d", s)
	}

	unit := s[len(s)-1]
	numStr := s[:len(s)-1]

	num, err := strconv.Atoi(numStr)
	if err != nil || num <= 0 {
		return 0, fmt.Errorf("invalid expiration %q — use format like 1h, 3d, 7d", s)
	}

	switch unit {
	case 'h':
		return num * 3600, nil
	case 'd':
		return num * 86400, nil
	default:
		return 0, fmt.Errorf("invalid expiration unit %q — use 'h' (hours) or 'd' (days)", string(unit))
	}
}
