package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/mdpush-io/cli/internal/api"
	"github.com/mdpush-io/cli/internal/auth"
	"github.com/mdpush-io/cli/internal/crypto"
	"github.com/mdpush-io/cli/internal/keystore"
)

var openCmd = &cobra.Command{
	Use:   "open <title>",
	Short: "Open a locally-saved document by title",
	Long: `Open a document from your local key store by searching its title.

The search is case-insensitive and matches partial titles:
  mdpush open "meeting notes"
  mdpush open bug

If multiple documents match, a numbered list is shown so you can pick one.
Only documents shared from this device (with keys in the local store) are searchable.`,
	Args:          cobra.MinimumNArgs(1),
	Aliases:       []string{"read"},
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE:          runOpen,
}

func init() {
	rootCmd.AddCommand(openCmd)
}

// localDoc holds a decrypted doc from the local key store.
type localDoc struct {
	id  string
	key []byte
	crypto.Document
}

func runOpen(cmd *cobra.Command, args []string) error {
	query := strings.ToLower(strings.Join(args, " "))

	// 1. Load the local key store
	ks, err := keystore.New()
	if err != nil {
		return fmt.Errorf("opening key store: %w", err)
	}
	if err := ks.Load(); err != nil {
		return fmt.Errorf("loading key store: %w", err)
	}

	// 2. We need the encrypted payloads from the server to decrypt titles
	client := api.NewClient()
	if session, _ := auth.LoadAuth(); session != nil && session.IsValid() {
		client = client.WithToken(session.Token)
	} else {
		return fmt.Errorf("not logged in — run `mdpush` to set up")
	}

	resp, err := client.ListDocs()
	if err != nil {
		return fmt.Errorf("fetching docs: %w", err)
	}

	// 3. Decrypt metadata for docs we have keys for, and match against query
	var matches []localDoc
	for _, enc := range resp.Sent {
		key := ks.Get(enc.ID)
		if key == nil {
			continue
		}

		doc, err := crypto.DecryptPayload(enc.EncryptedPayload, key)
		if err != nil {
			continue
		}

		// Match against title, category, project
		if strings.Contains(strings.ToLower(doc.Title), query) ||
			strings.Contains(strings.ToLower(doc.Category), query) ||
			strings.Contains(strings.ToLower(doc.Project), query) {
			matches = append(matches, localDoc{id: enc.ID, key: key, Document: doc})
		}
	}

	if len(matches) == 0 {
		return fmt.Errorf("no docs matching %q found in your local key store\nUse `mdpush fetch <url>` to fetch a doc from a link", query)
	}

	// 4. If exactly one match, render it directly
	if len(matches) == 1 {
		renderDoc(matches[0].Document)
		return nil
	}

	// 5. Multiple matches — list them for selection
	fmt.Println()
	fmt.Printf("  \033[1m%d docs match %q:\033[0m\n\n", len(matches), query)
	for i, m := range matches {
		meta := ""
		var parts []string
		if m.Category != "" {
			parts = append(parts, m.Category)
		}
		if m.Project != "" {
			parts = append(parts, m.Project)
		}
		if len(parts) > 0 {
			meta = "  \033[90m" + strings.Join(parts, " · ") + "\033[0m"
		}
		fmt.Printf("  \033[32m%d)\033[0m %s%s\n", i+1, m.Title, meta)
	}
	fmt.Println()
	fmt.Printf("  \033[90mRun: mdpush open \"%s\" —or— refine your search\033[0m\n\n", matches[0].Title)

	return nil
}
