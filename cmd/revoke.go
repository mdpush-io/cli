package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/mdpush-io/cli/internal/auth"
)

var revokeCmd = &cobra.Command{
	Use:   "revoke <id>",
	Short: "Revoke a document so readers can no longer access it",
	Args:  cobra.ExactArgs(1),
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		client, _, err := auth.AuthenticatedClient()
		if err != nil {
			return err
		}

		resp, err := client.RevokeDoc(args[0])
		if err != nil {
			return fmt.Errorf("revoking document: %w", err)
		}

		if resp.Revoked {
			fmt.Printf("  ✓ Revoked %s — readers can no longer access it\n", resp.ID)
		} else {
			fmt.Printf("  Document %s was already revoked\n", resp.ID)
		}
		return nil
	},
}

var restoreCmd = &cobra.Command{
	Use:   "restore <id>",
	Short: "Restore a revoked document so readers can access it again",
	Args:  cobra.ExactArgs(1),
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		client, _, err := auth.AuthenticatedClient()
		if err != nil {
			return err
		}

		resp, err := client.RestoreDoc(args[0])
		if err != nil {
			return fmt.Errorf("restoring document: %w", err)
		}

		if !resp.Revoked {
			fmt.Printf("  ✓ Restored %s — readers can access it again\n", resp.ID)
		} else {
			fmt.Printf("  Document %s is still revoked\n", resp.ID)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(revokeCmd)
	rootCmd.AddCommand(restoreCmd)
}
