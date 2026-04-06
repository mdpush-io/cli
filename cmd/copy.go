package cmd

import (
	"fmt"

	"github.com/atotto/clipboard"
	"github.com/spf13/cobra"

	"github.com/mdpush-io/cli/internal/crypto"
	"github.com/mdpush-io/cli/internal/keystore"
)

var copyCmd = &cobra.Command{
	Use:   "copy <id>",
	Short: "Copy a document's share URL to the clipboard",
	Args:  cobra.ExactArgs(1),
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		docID := args[0]

		ks, err := keystore.New()
		if err != nil {
			return fmt.Errorf("opening key store: %w", err)
		}
		if err := ks.Load(); err != nil {
			return fmt.Errorf("loading key store: %w", err)
		}

		docKey := ks.Get(docID)
		if docKey == nil {
			return fmt.Errorf("no local key for %s — can only copy links for docs shared from this device", docID)
		}

		fragment := crypto.KeyToFragment(docKey)
		url := fmt.Sprintf("https://www.mdpush.io/d/%s#%s", docID, fragment)

		if err := clipboard.WriteAll(url); err != nil {
			// Fallback: print the URL if clipboard fails
			fmt.Println(url)
			return nil
		}

		fmt.Printf("  ✓ Copied link for %s to clipboard\n", docID)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(copyCmd)
}
