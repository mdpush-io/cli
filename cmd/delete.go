package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/mdpush-io/cli/internal/auth"
	"github.com/mdpush-io/cli/internal/keystore"
)

var flagForce bool

var deleteCmd = &cobra.Command{
	Use:   "delete <id>",
	Short: "Permanently delete a document",
	Long: `Permanently delete a document from the server.

This cannot be undone. Use --force to skip confirmation.`,
	Args:          cobra.ExactArgs(1),
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		docID := args[0]

		if !flagForce {
			fmt.Printf("  Delete %s permanently? This cannot be undone.\n", docID)
			fmt.Printf("  Re-run with --force to confirm.\n")
			return nil
		}

		client, _, err := auth.AuthenticatedClient()
		if err != nil {
			return err
		}

		resp, err := client.DeleteDoc(docID)
		if err != nil {
			return fmt.Errorf("deleting document: %w", err)
		}

		if resp.Deleted {
			// Clean up local key store
			if ks, err := keystore.New(); err == nil {
				_ = ks.Load()
				_ = ks.Delete(docID)
			}
			fmt.Printf("  ✓ Deleted %s\n", resp.ID)
		}
		return nil
	},
}

func init() {
	deleteCmd.Flags().BoolVar(&flagForce, "force", false, "skip confirmation")

	rootCmd.AddCommand(deleteCmd)
}
