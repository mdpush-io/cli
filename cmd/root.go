package cmd

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"github.com/mdpush-io/cli/internal/api"
	"github.com/mdpush-io/cli/internal/auth"
	"github.com/mdpush-io/cli/internal/keystore"
	"github.com/mdpush-io/cli/internal/tui/dashboard"
	"github.com/mdpush-io/cli/internal/tui/setup"
)

// Version is set at build time by GoReleaser via -ldflags.
var Version = "dev"

var rootCmd = &cobra.Command{
	Use:     "mdpush",
	Short:   "Share markdown docs with zero-knowledge encryption",
	Version: Version,
	Long: `mdpush — Beautiful markdown, shared fast.

Share encrypted markdown documents from your terminal.
All content is encrypted client-side before upload.
The server never sees your documents.

Run 'mdpush share <file.md>' to share your first document.`,
	SilenceErrors: true,
	SilenceUsage:  true,
	// When run with no subcommand: check session → dashboard TUI or setup TUI
	RunE: func(cmd *cobra.Command, args []string) error {
		session, _ := auth.LoadAuth()

		if session != nil && session.IsValid() {
			// Logged in — launch dashboard TUI
			client := api.NewClient().WithToken(session.Token)
			keys, err := keystore.New()
			if err != nil {
				return fmt.Errorf("opening key store: %w", err)
			}
			if err := keys.Load(); err != nil {
				return fmt.Errorf("loading key store: %w", err)
			}

			p := tea.NewProgram(dashboard.New(client, keys), tea.WithAltScreen())
			if _, err := p.Run(); err != nil {
				return fmt.Errorf("dashboard failed: %w", err)
			}
			return nil
		}

		// No session or expired — launch setup/login TUI
		p := tea.NewProgram(setup.New(), tea.WithAltScreen())
		if _, err := p.Run(); err != nil {
			return fmt.Errorf("setup failed: %w", err)
		}
		return nil
	},
}

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
