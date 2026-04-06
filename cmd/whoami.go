package cmd

import (
	"fmt"
	"time"

	"github.com/mdpush-io/cli/internal/auth"
	"github.com/spf13/cobra"
)

var whoamiCmd = &cobra.Command{
	Use:           "whoami",
	Short:         "Show current authenticated user",
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		session, err := auth.LoadAuth()
		if err != nil {
			return err
		}
		if session == nil {
			return fmt.Errorf("not logged in — run 'mdpush' to set up")
		}

		fmt.Println()
		fmt.Printf("  email     %s\n", session.Email)
		fmt.Printf("  user id   %s\n", session.UserID)

		if t, err := time.Parse(time.RFC3339, session.ExpiresAt); err == nil {
			remaining := time.Until(t)
			if remaining > 0 {
				fmt.Printf("  session   valid for %dd\n", int(remaining.Hours()/24))
			} else {
				fmt.Printf("  session   expired\n")
			}
		}

		fmt.Println()
		return nil
	},
}

func init() {
	rootCmd.AddCommand(whoamiCmd)
}
