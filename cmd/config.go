package cmd

import (
	"fmt"
	"strings"

	"github.com/mdpush-io/cli/internal/auth"
	"github.com/mdpush-io/cli/internal/config"
	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage sender defaults",
	Long:  `View and manage default values for mdpush share flags.`,
}

var configSetCmd = &cobra.Command{
	Use:           "set <key> <value>",
	Short:         "Set a default value",
	Args:          cobra.ExactArgs(2),
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		key, value := args[0], args[1]

		cfg, err := config.Load()
		if err != nil {
			return err
		}

		if err := cfg.Set(key, value); err != nil {
			return err
		}

		if err := config.Save(cfg); err != nil {
			return err
		}

		// Sync to server (best-effort)
		if client, _, err := auth.AuthenticatedClient(); err == nil {
			_ = config.PushToServer(client, cfg)
		}

		fmt.Printf("  ✓ %s = %s\n", key, value)
		return nil
	},
}

var configListCmd = &cobra.Command{
	Use:           "list",
	Short:         "Show current defaults",
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}

		eff := cfg.Effective()

		fmt.Println()
		printConfigRow("lock", eff.Lock, cfg.Lock)
		printConfigRow("expires", displayOr(eff.Expires, "(none)"), cfg.Expires)
		printConfigRow("views", displayIntOr(eff.Views, "(unlimited)"), cfg.Views > 0)
		printConfigRow("theme", eff.Theme, cfg.Theme)
		printConfigRow("category", displayOr(eff.Category, "(none)"), cfg.Category)
		printConfigRow("project", displayOr(eff.Project, "(none)"), cfg.Project)
		printConfigRow("pw-theme", eff.PwTheme, cfg.PwTheme)
		fmt.Println()

		return nil
	},
}

var configResetCmd = &cobra.Command{
	Use:           "reset",
	Short:         "Reset all defaults to system values",
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := config.Reset(); err != nil {
			return err
		}
		fmt.Println("  ✓ All defaults reset to system values")
		return nil
	},
}

func init() {
	configCmd.AddCommand(configSetCmd)
	configCmd.AddCommand(configListCmd)
	configCmd.AddCommand(configResetCmd)
	rootCmd.AddCommand(configCmd)
}

// --- Helpers ---

func printConfigRow(key, value string, isCustom any) {
	marker := " "
	switch v := isCustom.(type) {
	case string:
		if v != "" {
			marker = "●"
		}
	case bool:
		if v {
			marker = "●"
		}
	}
	fmt.Printf("  %s %-12s %s\n", marker, key, value)
}

func displayOr(val, fallback string) string {
	if val == "" {
		return fallback
	}
	return val
}

func displayIntOr(val int, fallback string) string {
	if val <= 0 {
		return fallback
	}
	return fmt.Sprintf("%d", val)
}

// ValidKeysCompletion returns valid keys for shell completion.
func init() {
	configSetCmd.ValidArgsFunction = func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) == 0 {
			return config.ValidKeys, cobra.ShellCompDirectiveNoFileComp
		}
		// Suggest values for known keys
		switch args[0] {
		case "lock":
			return []string{"light", "strong"}, cobra.ShellCompDirectiveNoFileComp
		case "theme":
			return []string{"clean", "dark", "github", "technical"}, cobra.ShellCompDirectiveNoFileComp
		case "category":
			return []string{"new-feature", "debugging", "rambling"}, cobra.ShellCompDirectiveNoFileComp
		case "pw-theme":
			return []string{"books", "animals", "numbers"}, cobra.ShellCompDirectiveNoFileComp
		}
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	_ = strings.Join // ensure import
}
