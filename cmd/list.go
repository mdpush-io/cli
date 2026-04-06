package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/mdpush-io/cli/internal/auth"
	"github.com/mdpush-io/cli/internal/crypto"
	"github.com/mdpush-io/cli/internal/keystore"
)

var (
	flagListJSON     bool
	flagListProject  string
	flagListCategory string
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List your documents",
	Long: `List your shared documents with decrypted metadata.

Supports filtering and JSON output for scripting:
  mdpush list
  mdpush list --project mdpush
  mdpush list --category debugging
  mdpush list --json | jq '.[].title'`,
	Aliases:       []string{"ls"},
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE:          runList,
}

func init() {
	listCmd.Flags().BoolVar(&flagListJSON, "json", false, "output as JSON")
	listCmd.Flags().StringVar(&flagListProject, "project", "", "filter by project")
	listCmd.Flags().StringVar(&flagListCategory, "category", "", "filter by category")

	rootCmd.AddCommand(listCmd)
}

type listDoc struct {
	ID       string  `json:"id"`
	Title    string  `json:"title"`
	Category string  `json:"category,omitempty"`
	Project  string  `json:"project,omitempty"`
	LockType string  `json:"lockType"`
	Views    string  `json:"views"`
	Revoked  bool    `json:"revoked,omitempty"`
	Date     string  `json:"date"`
	HasKey   bool    `json:"hasKey"`
}

func runList(cmd *cobra.Command, args []string) error {
	client, _, err := auth.AuthenticatedClient()
	if err != nil {
		return err
	}

	ks, _ := keystore.New()
	if ks != nil {
		_ = ks.Load()
	}

	resp, err := client.ListDocs()
	if err != nil {
		return fmt.Errorf("fetching docs: %w", err)
	}

	var docs []listDoc
	for _, enc := range resp.Sent {
		d := listDoc{
			ID:       enc.ID,
			Title:    "[encrypted]",
			LockType: enc.LockType,
			Revoked:  enc.Revoked,
			HasKey:   false,
		}

		// Views
		d.Views = fmt.Sprintf("%d", enc.CurrentViews)
		if enc.MaxViews != nil {
			d.Views = fmt.Sprintf("%d/%d", enc.CurrentViews, *enc.MaxViews)
		}

		// Date
		if t, err := time.Parse(time.RFC3339, enc.CreatedAt); err == nil {
			d.Date = t.Format("2006-01-02")
		}

		// Decrypt metadata if key available
		if ks != nil {
			if key := ks.Get(enc.ID); key != nil {
				d.HasKey = true
				if doc, err := crypto.DecryptPayload(enc.EncryptedPayload, key); err == nil {
					d.Title = doc.Title
					d.Category = doc.Category
					d.Project = doc.Project
				}
			}
		}

		// Apply filters
		if flagListProject != "" && !strings.EqualFold(d.Project, flagListProject) {
			continue
		}
		if flagListCategory != "" && !strings.EqualFold(d.Category, flagListCategory) {
			continue
		}

		docs = append(docs, d)
	}

	if flagListJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(docs)
	}

	if len(docs) == 0 {
		fmt.Println("  No documents found.")
		return nil
	}

	// Table output
	fmt.Println()
	for _, d := range docs {
		status := ""
		if d.Revoked {
			status = "  \033[31mREVOKED\033[0m"
		}

		lock := "○"
		if d.LockType == "strong" {
			lock = "●"
		}

		tags := ""
		var parts []string
		if d.Category != "" {
			parts = append(parts, d.Category)
		}
		if d.Project != "" {
			parts = append(parts, d.Project)
		}
		if len(parts) > 0 {
			tags = "\033[90m" + strings.Join(parts, " · ") + "\033[0m"
		}

		title := d.Title
		if !d.HasKey {
			title = "\033[90m" + title + "\033[0m"
		}

		fmt.Printf("  \033[90m%s\033[0m  %-40s  %-25s  %s  %-8s  %s%s\n",
			d.ID, title, tags, lock, d.Views, d.Date, status)
	}
	fmt.Println()

	return nil
}
