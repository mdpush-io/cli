package components

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Color palette — matches the web app's "Terminal Elegance" design system.
var (
	ColorAccent  = lipgloss.Color("#059669") // emerald green
	ColorMuted   = lipgloss.Color("#6b7280") // gray-500
	ColorDim     = lipgloss.Color("#4b5563") // gray-600
	ColorError   = lipgloss.Color("#ef4444") // red
	ColorWarn    = lipgloss.Color("#f59e0b") // amber
	ColorText    = lipgloss.Color("#e5e7eb") // gray-200
	ColorSurface = lipgloss.Color("#1f2937") // gray-800
	ColorBg      = lipgloss.Color("#111827") // gray-900
)

// Shared styles.
var (
	StyleTitle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorText)

	StyleAccent = lipgloss.NewStyle().
			Foreground(ColorAccent)

	StyleMuted = lipgloss.NewStyle().
			Foreground(ColorMuted)

	StyleError = lipgloss.NewStyle().
			Foreground(ColorError)

	StyleBold = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorText)

	StyleBox = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorDim).
			Padding(1, 2)

	StyleHighlightBox = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(ColorAccent).
				Padding(1, 2)

	StyleKeyHint = lipgloss.NewStyle().
			Foreground(ColorMuted).
			Italic(true)

	StyleSelectedRow = lipgloss.NewStyle().
				Background(lipgloss.Color("#0d3320")).
				Foreground(ColorText)

	StyleBadge = lipgloss.NewStyle().
			Foreground(ColorError).
			Bold(true)

	StyleTagBadge = lipgloss.NewStyle().
			Foreground(ColorDim)

	StyleDivider = lipgloss.NewStyle().
			Foreground(ColorDim)
)

// Margin is the left padding applied to all TUI content.
const Margin = "   "

// Logo is the ASCII art banner for mdpush.
const logo = `                       ░██                                  ░██            ░██
                       ░██                                  ░██
░█████████████   ░████████ ░████████  ░██    ░██  ░███████  ░████████      ░██ ░███████
░██   ░██   ░██ ░██    ░██ ░██    ░██ ░██    ░██ ░██        ░██    ░██     ░██░██    ░██
░██   ░██   ░██ ░██    ░██ ░██    ░██ ░██    ░██  ░███████  ░██    ░██     ░██░██    ░██
░██   ░██   ░██ ░██   ░███ ░███   ░██ ░██   ░███        ░██ ░██    ░██     ░██░██    ░██
░██   ░██   ░██  ░█████░██ ░██░█████   ░█████░██  ░███████  ░██    ░██ ░██ ░██ ░███████
                           ░██
                           ░██`

// LogoCompact is a single-line brand mark.
const LogoCompact = "░█ mdpush"

// RenderLogo returns the logo rendered in the accent color with left margin.
func RenderLogo() string {
	var lines []string
	for _, line := range strings.Split(logo, "\n") {
		lines = append(lines, Margin+line)
	}
	return StyleAccent.Render(strings.Join(lines, "\n"))
}

// RenderLogoWithMotto returns the logo with "Markdown shared fast" tagline.
func RenderLogoWithMotto() string {
	return RenderLogo() + "\n" + Margin + StyleMuted.Render("Markdown shared fast.")
}

// Divider returns a horizontal rule of the given width with an optional centered label.
func Divider(width int, label string) string {
	if width < 10 {
		width = 60
	}
	if label == "" {
		return StyleDivider.Render(repeat("─", width))
	}
	labelRendered := " " + label + " "
	sideLen := (width - len(labelRendered)) / 2
	if sideLen < 2 {
		sideLen = 2
	}
	left := repeat("─", sideLen)
	right := repeat("─", width-sideLen-len(labelRendered))
	return StyleDivider.Render(left+labelRendered+right)
}

func repeat(s string, n int) string {
	if n <= 0 {
		return ""
	}
	out := ""
	for range n {
		out += s
	}
	return out
}
