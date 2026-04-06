package cmd

import (
	"fmt"

	"github.com/mdpush-io/cli/internal/api"
	"github.com/mdpush-io/cli/internal/auth"
	"github.com/spf13/cobra"
)

var logoutCmd = &cobra.Command{
	Use:           "logout",
	Short:         "Clear local session and log out",
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		session, _ := auth.LoadAuth()

		// Build a client to revoke the session on the server (best-effort)
		var client *api.Client
		if session != nil && session.IsValid() {
			client = api.NewClient().WithToken(session.Token)
		}

		if err := auth.Logout(client); err != nil {
			return err
		}

		auth.ClearIdentityCache()

		fmt.Println("  ✓ Logged out")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(logoutCmd)
}
