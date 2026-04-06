package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/mdpush-io/cli/internal/auth"
	"github.com/mdpush-io/cli/internal/crypto"
	"github.com/mdpush-io/cli/internal/keystore"
)

var updateCmd = &cobra.Command{
	Use:   "update <id> <file.md>",
	Short: "Update a document's content",
	Long: `Replace a document's encrypted content without changing its URL or settings.

The document keeps the same ID, link, lock, and expiration — only the
encrypted payload is replaced.

  mdpush update abc123 updated-doc.md`,
	Args:          cobra.ExactArgs(2),
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE:          runUpdate,
}

func init() {
	rootCmd.AddCommand(updateCmd)
}

func runUpdate(cmd *cobra.Command, args []string) error {
	docID := args[0]
	filePath := args[1]

	// 1. Load auth
	client, _, err := auth.AuthenticatedClient()
	if err != nil {
		return err
	}

	// 2. Get the doc key from local store
	ks, err := keystore.New()
	if err != nil {
		return fmt.Errorf("opening key store: %w", err)
	}
	if err := ks.Load(); err != nil {
		return fmt.Errorf("loading key store: %w", err)
	}

	docKey := ks.Get(docID)
	if docKey == nil {
		return fmt.Errorf("no local key for %s — you can only update docs shared from this device", docID)
	}

	// 3. Read the new file
	content, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("reading file: %w", err)
	}

	// 4. Build new payload with updated content, preserving title extraction
	title := buildTitle(filePath, string(content))
	doc := crypto.Document{
		Title:   title,
		Content: string(content),
	}

	// 5. Encrypt with the same doc key
	encryptedPayload, err := crypto.EncryptPayload(doc, docKey)
	if err != nil {
		return fmt.Errorf("encrypting payload: %w", err)
	}

	// 6. Upload
	resp, err := client.UpdateDoc(docID, encryptedPayload)
	if err != nil {
		return fmt.Errorf("updating document: %w", err)
	}

	fmt.Printf("  ✓ Updated %s — \"%s\"\n", resp.ID, title)
	return nil
}
