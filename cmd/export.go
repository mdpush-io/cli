package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/mdpush-io/cli/internal/auth"
	"github.com/mdpush-io/cli/internal/crypto"
	"github.com/mdpush-io/cli/internal/keystore"
)

var flagOutput string

var exportCmd = &cobra.Command{
	Use:   "export <id>",
	Short: "Export a document's decrypted content to a file",
	Long: `Fetch and decrypt a document, then save the raw markdown to a file.

  mdpush export abc123 -o report.md
  mdpush export abc123              # prints to stdout`,
	Args:          cobra.ExactArgs(1),
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE:          runExport,
}

func init() {
	exportCmd.Flags().StringVarP(&flagOutput, "output", "o", "", "output file path (default: stdout)")

	rootCmd.AddCommand(exportCmd)
}

func runExport(cmd *cobra.Command, args []string) error {
	docID := args[0]

	// 1. Get the doc key
	ks, err := keystore.New()
	if err != nil {
		return fmt.Errorf("opening key store: %w", err)
	}
	if err := ks.Load(); err != nil {
		return fmt.Errorf("loading key store: %w", err)
	}

	docKey := ks.Get(docID)
	if docKey == nil {
		return fmt.Errorf("no local key for %s — provide a full URL via `mdpush fetch` first", docID)
	}

	// 2. Fetch from server
	client, _, err := auth.AuthenticatedClient()
	if err != nil {
		return err
	}

	credential := tryAutoCredential(client)

	resp, err := client.GetDoc(docID, credential)
	if err != nil {
		return fmt.Errorf("fetching document: %w", err)
	}

	// 3. Decrypt
	doc, err := crypto.DecryptPayload(resp.EncryptedPayload, docKey)
	if err != nil {
		return fmt.Errorf("decryption failed — wrong key?")
	}

	// 4. Write output
	if flagOutput == "" {
		// stdout
		fmt.Print(doc.Content)
		return nil
	}

	if err := os.WriteFile(flagOutput, []byte(doc.Content), 0644); err != nil {
		return fmt.Errorf("writing file: %w", err)
	}

	fmt.Printf("  ✓ Exported \"%s\" to %s\n", doc.Title, flagOutput)
	return nil
}
