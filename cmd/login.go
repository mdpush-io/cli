package cmd

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"github.com/mdpush-io/cli/internal/auth"
	"github.com/mdpush-io/cli/internal/tui/setup"
)

var (
	loginOTP    bool
	loginGitHub bool
)

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Log in to mdpush",
	Long: `Log in to mdpush.

By default, prompts you to choose between GitHub and email (OTP) login.
Pass --github or --otp to skip the picker and go straight to that flow.`,
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		if loginOTP && loginGitHub {
			return fmt.Errorf("--otp and --github are mutually exclusive")
		}

		// If already logged in with a valid session, bail early so we
		// don't silently overwrite the existing session file.
		if session, _ := auth.LoadAuth(); session != nil && session.IsValid() {
			return fmt.Errorf("already logged in as %s — run `mdpush logout` first", session.Email)
		}

		var model tea.Model
		switch {
		case loginGitHub:
			model = setup.NewGitHub()
		case loginOTP:
			model = setup.NewEmail()
		default:
			model = setup.New()
		}

		p := tea.NewProgram(model, tea.WithAltScreen())
		if _, err := p.Run(); err != nil {
			return fmt.Errorf("login failed: %w", err)
		}
		return nil
	},
}

func init() {
	loginCmd.Flags().BoolVar(&loginGitHub, "github", false, "Log in via GitHub (device flow)")
	loginCmd.Flags().BoolVar(&loginOTP, "otp", false, "Log in via email one-time code")
	rootCmd.AddCommand(loginCmd)
}
