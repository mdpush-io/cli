package cmd

import (
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/charmbracelet/glamour"
	"github.com/spf13/cobra"

	"github.com/mdpush-io/cli/internal/api"
	"github.com/mdpush-io/cli/internal/auth"
	"github.com/mdpush-io/cli/internal/crypto"
	"github.com/mdpush-io/cli/internal/keystore"
)

var flagCredential string

var fetchCmd = &cobra.Command{
	Use:   "fetch <url-or-id>",
	Short: "Fetch a document from the server, decrypt, and display it",
	Long: `Fetch a shared mdpush document from the server and display it in the terminal.

Accepts a full URL (with the decryption key in the fragment):
  mdpush fetch https://www.mdpush.io/d/abc123#<key>

Or just a document ID (uses the local key store for the decryption key):
  mdpush fetch abc123

If the document has a lock, provide the credential with --credential.`,
	Args:          cobra.ExactArgs(1),
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE:          runFetch,
}

func init() {
	fetchCmd.Flags().StringVar(&flagCredential, "credential", "", "lock credential (email for light lock, password for strong lock)")

	rootCmd.AddCommand(fetchCmd)
}

func runFetch(cmd *cobra.Command, args []string) error {
	input := args[0]

	// 1. Parse the input — extract doc ID and optional key from fragment
	docID, docKey, err := parseDocInput(input)
	if err != nil {
		return err
	}

	// 2. If no key from URL, try the local key store
	if docKey == nil {
		if ks, err := keystore.New(); err == nil {
			_ = ks.Load()
			docKey = ks.Get(docID)
		}
	}

	if docKey == nil {
		return fmt.Errorf("no decryption key available for %s\nProvide a full URL with the key in the fragment, or share the doc from this device first", docID)
	}

	// 3. Build a client — use auth if available, otherwise anonymous
	client := api.NewClient()
	if session, _ := auth.LoadAuth(); session != nil && session.IsValid() {
		client = client.WithToken(session.Token)
	}

	// 4. Resolve lock credential
	credential := flagCredential
	if credential == "" {
		credential = tryAutoCredential(client)
	}

	// 5. Fetch the document
	resp, err := client.GetDoc(docID, credential)
	if err != nil {
		var apiErr *api.APIError
		if ok := isAPIError(err, &apiErr); ok {
			if apiErr.LockType != "" && credential == "" {
				lockHint := "email or name"
				if apiErr.LockType == "strong" {
					lockHint = "password"
				}
				return fmt.Errorf("this doc requires a %s lock credential\nUse: mdpush fetch %s --credential <your-%s>", apiErr.LockType, input, lockHint)
			}
		}
		return fmt.Errorf("fetching document: %w", err)
	}

	// 6. Decrypt
	doc, err := crypto.DecryptPayload(resp.EncryptedPayload, docKey)
	if err != nil {
		return fmt.Errorf("decryption failed — wrong key?")
	}

	// 7. Save key to local store for future access (best-effort)
	if ks, err := keystore.New(); err == nil {
		_ = ks.Load()
		_ = ks.Put(docID, docKey)
	}

	// 8. Render
	renderDoc(doc)
	return nil
}

// parseDocInput extracts doc ID and optional key from a URL or plain ID.
func parseDocInput(input string) (string, []byte, error) {
	// Full URL: https://www.mdpush.io/d/abc123#<key>
	if strings.Contains(input, "://") || strings.HasPrefix(input, "www.") {
		u, err := url.Parse(input)
		if err != nil {
			return "", nil, fmt.Errorf("invalid URL: %w", err)
		}

		// Extract ID from path: /d/<id>
		path := strings.TrimPrefix(u.Path, "/d/")
		path = strings.TrimPrefix(path, "/")
		path = strings.Split(path, "/")[0]
		if path == "" {
			return "", nil, fmt.Errorf("could not extract document ID from URL")
		}

		// Extract key from fragment
		var docKey []byte
		if u.Fragment != "" {
			key, err := crypto.ParseKeyFromFragment(u.Fragment)
			if err == nil && len(key) == 32 {
				docKey = key
			}
		}

		return path, docKey, nil
	}

	// Plain ID
	id := strings.TrimSpace(input)
	if id == "" {
		return "", nil, fmt.Errorf("empty document ID")
	}
	return id, nil, nil
}

// tryAutoCredential returns the sender's email as a credential if authenticated.
// This handles the case where the user opens their own light-locked doc.
func tryAutoCredential(client *api.Client) string {
	if client.Token == "" {
		return ""
	}
	me, err := client.GetMe()
	if err != nil {
		return ""
	}
	return crypto.NormalizeLockCredential(me.Email)
}

// isAPIError checks if err is an *api.APIError and assigns it.
func isAPIError(err error, target **api.APIError) bool {
	if e, ok := err.(*api.APIError); ok {
		*target = e
		return true
	}
	return false
}

// renderDoc prints a decrypted document with glamour markdown rendering.
func renderDoc(doc crypto.Document) {
	rendered, err := glamour.Render(doc.Content, "dark")
	if err != nil {
		rendered = doc.Content
	}

	fmt.Println()
	fmt.Printf("  \033[1m%s\033[0m\n", doc.Title)
	var meta []string
	if doc.Category != "" {
		meta = append(meta, doc.Category)
	}
	if doc.Project != "" {
		meta = append(meta, doc.Project)
	}
	if len(meta) > 0 {
		fmt.Printf("  \033[90m%s\033[0m\n", strings.Join(meta, " · "))
	}
	fmt.Println()
	fmt.Fprint(os.Stdout, rendered)
}
