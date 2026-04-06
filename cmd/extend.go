package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/mdpush-io/cli/internal/api"
	"github.com/mdpush-io/cli/internal/auth"
)

var (
	flagExtendExpires string
	flagExtendViews   int
)

var extendCmd = &cobra.Command{
	Use:   "extend <id>",
	Short: "Extend a document's expiration or view limit",
	Long: `Add more time or views to an existing document.

Examples:
  mdpush extend abc123 --expires 7d
  mdpush extend abc123 --views 10
  mdpush extend abc123 --expires 3d --views 5`,
	Args:          cobra.ExactArgs(1),
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		docID := args[0]

		if flagExtendExpires == "" && flagExtendViews == 0 {
			return fmt.Errorf("specify --expires and/or --views to extend")
		}

		client, _, err := auth.AuthenticatedClient()
		if err != nil {
			return err
		}

		req := api.ExtendDocRequest{}

		if flagExtendExpires != "" {
			seconds, err := parseExpiration(flagExtendExpires)
			if err != nil {
				return err
			}
			req.AddSeconds = &seconds
		}

		if flagExtendViews > 0 {
			req.AddViews = &flagExtendViews
		}

		resp, err := client.ExtendDoc(docID, req)
		if err != nil {
			return fmt.Errorf("extending document: %w", err)
		}

		var parts []string
		if resp.ExpiresAt != nil {
			parts = append(parts, fmt.Sprintf("expires %s", *resp.ExpiresAt))
		}
		if resp.MaxViews != nil {
			parts = append(parts, fmt.Sprintf("max %d views", *resp.MaxViews))
		}

		fmt.Printf("  ✓ Extended %s — %s\n", resp.ID, strings.Join(parts, ", "))
		return nil
	},
}

func init() {
	extendCmd.Flags().StringVar(&flagExtendExpires, "expires", "", "add time: 1h, 3d, 7d, 30d")
	extendCmd.Flags().IntVar(&flagExtendViews, "views", 0, "add views")

	rootCmd.AddCommand(extendCmd)
}
