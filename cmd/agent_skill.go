package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var agentSkillCmd = &cobra.Command{
	Use:   "agent-skill",
	Short: "Print the mdpush skill description for AI agents",
	Long: `Output a structured skill description that teaches AI agents how to use mdpush.

An agent (Claude Code, Codex, Gemini, etc.) can run this command to learn
the available commands, then create its own skill file:

  mdpush agent-skill`,
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Print(agentSkillText)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(agentSkillCmd)
}

const agentSkillText = `# mdpush — AI Agent Skill

mdpush is a CLI for sharing encrypted markdown documents. All content is
encrypted client-side with AES-256-GCM. The server never sees plaintext.
The decryption key lives in the URL fragment (never sent to the server).

## Authentication

The user must be logged in. Check with:
  mdpush whoami

If not logged in, the user should run ` + "`mdpush`" + ` to start the setup flow.
They can also run ` + "`mdpush login --github`" + ` or ` + "`mdpush login --otp`" + ` to skip the picker.

## Sharing a document

Share a markdown file:
  mdpush share <file.md>
  mdpush share <file.md> --lock strong --pw-theme books
  mdpush share <file.md> --category debugging --project my-project
  mdpush share <file.md> --expires 7d --views 10

Share from stdin (useful for generated content):
  echo "# Title\n\nContent" | mdpush share - --title "My Doc"

Options:
  --lock        "light" (default, verifies sender identity) or "strong" (password)
  --theme       reading theme: clean, dark, github, technical
  --expires     expiration: 1h, 12h, 3d, 7d, 30d, never
  --views       max view count (0 = unlimited)
  --category    new-feature, debugging, rambling
  --project     project name (freeform)
  --password    custom strong lock password (instead of generated)
  --pw-theme    password theme: books, animals, dates
  --title       explicit title (default: extracted from first # heading)

The command outputs a URL with the decryption key in the fragment.
The password (if strong lock) is also printed — share it separately.

## Reading a document

Fetch by URL or ID:
  mdpush fetch <url>
  mdpush fetch <doc-id>
  mdpush fetch <doc-id> --credential <password-or-email>

Open a locally-saved doc by title:
  mdpush open "meeting notes"
  mdpush read "bug report"

Export decrypted content to a file:
  mdpush export <doc-id> -o output.md
  mdpush export <doc-id>              # prints to stdout

## Managing documents

List all docs:
  mdpush list
  mdpush list --project my-project --category debugging
  mdpush list --json

Update content (same URL, same key):
  mdpush update <doc-id> <new-file.md>

Copy the share link to clipboard:
  mdpush copy <doc-id>

Extend expiration or views:
  mdpush extend <doc-id> --expires 7d --views 10

Revoke/restore access:
  mdpush revoke <doc-id>
  mdpush restore <doc-id>

Delete permanently:
  mdpush delete <doc-id> --force

## Configuration

Set defaults so you don't repeat flags:
  mdpush config set lock strong
  mdpush config set project my-project
  mdpush config set pw-theme animals
  mdpush config list
  mdpush config reset

## When to use mdpush

Use mdpush when the user asks you to:
- Share a document, note, report, or snippet securely
- Push markdown content to a shareable link
- Send something to a colleague with a link
- Export your output as a shared document

Use "mdpush share -" with stdin when generating content to share directly,
without creating a temporary file.

Use "mdpush open" or "mdpush read" to retrieve docs by title.
Use "mdpush fetch" when you have a URL or document ID.
Use "mdpush list --json" when you need to programmatically inspect the user's docs.
`
