package dashboard

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"

	"github.com/mdpush-io/cli/internal/api"
	"github.com/mdpush-io/cli/internal/crypto"
	"github.com/mdpush-io/cli/internal/keystore"
	"github.com/mdpush-io/cli/internal/tui/components"
)

// --- State ---

type viewState int

const (
	stateList viewState = iota
	stateViewer
	stateConfirm
)

// groupMode determines how the list is displayed.
type groupMode int

const (
	groupNone groupMode = iota
	groupCategory
	groupProject
)

// confirmAction identifies a pending destructive action.
type confirmAction int

const (
	actionNone confirmAction = iota
	actionDelete
	actionRevoke
	actionRestore
)

// --- Messages ---

type docsLoadedMsg struct {
	docs []DashDoc
}
type docsLoadErrMsg struct{ err error }
type actionDoneMsg struct{ message string }
type actionErrMsg struct{ err error }

// DashDoc is a document with both server metadata and decrypted fields.
type DashDoc struct {
	api.EncryptedDoc
	Title    string
	Category string
	Project  string
	Content  string
	HasKey   bool
}

// --- Model ---

// Model is the Bubble Tea model for the dashboard.
type Model struct {
	state  viewState
	width  int
	height int

	// Data
	docs     []DashDoc
	filtered []DashDoc
	cursor   int
	loading  bool
	err      string
	status   string

	// Search
	searchInput textinput.Model
	searching   bool
	searchQuery string

	// Grouping
	group groupMode

	// Viewer
	viewport    viewport.Model
	viewerReady bool
	viewerDoc   *DashDoc

	// Confirm dialog
	pendingAction confirmAction

	// Dependencies
	client *api.Client
	keys   *keystore.Store
}

// New creates a new dashboard model.
func New(client *api.Client, keys *keystore.Store) Model {
	si := textinput.New()
	si.Placeholder = "type to filter..."
	si.CharLimit = 100

	return Model{
		state:       stateList,
		loading:     true,
		client:      client,
		keys:        keys,
		searchInput: si,
	}
}

func (m Model) Init() tea.Cmd {
	return m.loadDocs()
}

// --- Update ---

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		if m.viewerReady {
			m.viewport.Width = msg.Width
			m.viewport.Height = msg.Height - 6
		}
		return m, nil

	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}

	case docsLoadedMsg:
		m.loading = false
		m.docs = msg.docs
		m.applyFilter()
		return m, nil

	case docsLoadErrMsg:
		m.loading = false
		m.err = fmt.Sprintf("Failed to load docs: %v", msg.err)
		return m, nil

	case actionDoneMsg:
		m.status = msg.message
		m.state = stateList
		m.pendingAction = actionNone
		return m, m.loadDocs()

	case actionErrMsg:
		m.err = fmt.Sprintf("%v", msg.err)
		m.state = stateList
		m.pendingAction = actionNone
		return m, nil
	}

	switch m.state {
	case stateList:
		return m.updateList(msg)
	case stateViewer:
		return m.updateViewer(msg)
	case stateConfirm:
		return m.updateConfirm(msg)
	}

	return m, nil
}

// --- View ---

func (m Model) View() string {
	switch m.state {
	case stateViewer:
		return m.viewViewer()
	case stateConfirm:
		return m.viewConfirm()
	default:
		return m.viewList()
	}
}

// --- Data loading ---

func (m Model) loadDocs() tea.Cmd {
	return func() tea.Msg {
		resp, err := m.client.ListDocs()
		if err != nil {
			return docsLoadErrMsg{err}
		}

		docs := make([]DashDoc, len(resp.Sent))
		for i, enc := range resp.Sent {
			dd := DashDoc{EncryptedDoc: enc}
			key := m.keys.Get(enc.ID)
			if key != nil {
				dd.HasKey = true
				if doc, err := crypto.DecryptPayload(enc.EncryptedPayload, key); err == nil {
					dd.Title = doc.Title
					dd.Category = doc.Category
					dd.Project = doc.Project
					dd.Content = doc.Content
				}
			}
			if dd.Title == "" {
				dd.Title = "[encrypted]"
			}
			docs[i] = dd
		}

		// Sort newest first
		sort.Slice(docs, func(i, j int) bool {
			return docs[i].CreatedAt > docs[j].CreatedAt
		})

		return docsLoadedMsg{docs}
	}
}

// --- List view ---

func (m Model) updateList(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.searching {
		return m.updateSearch(msg)
	}

	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}

	switch key.String() {
	case "q":
		return m, tea.Quit
	case "j", "down":
		if m.cursor < len(m.filtered)-1 {
			m.cursor++
		}
	case "k", "up":
		if m.cursor > 0 {
			m.cursor--
		}
	case "g":
		m.cursor = 0
	case "G":
		if len(m.filtered) > 0 {
			m.cursor = len(m.filtered) - 1
		}
	case "enter":
		if len(m.filtered) > 0 {
			return m.openViewer()
		}
	case "/":
		m.searching = true
		m.searchInput.Focus()
		return m, textinput.Blink
	case "c":
		if len(m.filtered) > 0 {
			m.copyLink()
		}
	case "d":
		if len(m.filtered) > 0 {
			m.pendingAction = actionDelete
			m.state = stateConfirm
		}
	case "r":
		if len(m.filtered) > 0 {
			doc := m.filtered[m.cursor]
			if doc.Revoked {
				m.pendingAction = actionRestore
			} else {
				m.pendingAction = actionRevoke
			}
			m.state = stateConfirm
		}
	case "f":
		m.group = (m.group + 1) % 3
		m.applyFilter()
	case "R":
		m.loading = true
		m.status = ""
		m.err = ""
		return m, m.loadDocs()
	}

	// Clear status on movement
	if key.String() != "R" {
		m.status = ""
	}
	if m.err != "" && key.String() != "enter" {
		m.err = ""
	}

	return m, nil
}

func (m Model) updateSearch(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok {
		switch key.String() {
		case "enter", "esc":
			m.searching = false
			m.searchQuery = m.searchInput.Value()
			m.searchInput.Blur()
			m.applyFilter()
			m.cursor = 0
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.searchInput, cmd = m.searchInput.Update(msg)
	m.searchQuery = m.searchInput.Value()
	m.applyFilter()
	m.cursor = 0
	return m, cmd
}

func (m *Model) applyFilter() {
	query := strings.ToLower(strings.TrimSpace(m.searchQuery))

	var result []DashDoc
	for _, d := range m.docs {
		if query != "" {
			match := strings.Contains(strings.ToLower(d.Title), query) ||
				strings.Contains(strings.ToLower(d.Category), query) ||
				strings.Contains(strings.ToLower(d.Project), query)
			if !match {
				continue
			}
		}
		result = append(result, d)
	}

	// Apply grouping sort
	switch m.group {
	case groupCategory:
		sort.SliceStable(result, func(i, j int) bool {
			return result[i].Category < result[j].Category
		})
	case groupProject:
		sort.SliceStable(result, func(i, j int) bool {
			return result[i].Project < result[j].Project
		})
	}

	m.filtered = result
	if m.cursor >= len(m.filtered) {
		m.cursor = max(0, len(m.filtered)-1)
	}
}

func (m Model) viewList() string {
	var s strings.Builder
	M := components.Margin
	w := m.width
	if w < 60 {
		w = 80
	}
	innerW := w - len(M)*2

	// Logo header
	s.WriteString("\n")
	s.WriteString(components.RenderLogoWithMotto())
	s.WriteString("\n\n")

	if m.loading {
		s.WriteString(M + components.StyleMuted.Render("Loading your documents...") + "\n\n")
		return s.String()
	}

	// Metrics bar
	totalViews := 0
	activeDocs := 0
	for _, d := range m.docs {
		totalViews += d.CurrentViews
		if !d.Revoked {
			activeDocs++
		}
	}
	metrics := M + components.StyleMuted.Render(fmt.Sprintf(
		"%d docs sent  ·  %d active  ·  %d total views",
		len(m.docs), activeDocs, totalViews,
	)) + "\n\n"
	s.WriteString(metrics)

	// Status / error bar
	if m.err != "" {
		s.WriteString(M + components.StyleError.Render("✗ "+m.err) + "\n\n")
	}
	if m.status != "" {
		s.WriteString(M + components.StyleAccent.Render("✓ "+m.status) + "\n\n")
	}

	// Search bar
	if m.searching {
		s.WriteString(M + "/ " + m.searchInput.View() + "\n\n")
	} else if m.searchQuery != "" {
		s.WriteString(M + components.StyleMuted.Render(fmt.Sprintf("/ %q  ·  %d results", m.searchQuery, len(m.filtered))) + "\n\n")
	}

	// Section divider
	label := fmt.Sprintf(" %d docs ", len(m.filtered))
	switch m.group {
	case groupCategory:
		label = fmt.Sprintf(" %d docs · by category ", len(m.filtered))
	case groupProject:
		label = fmt.Sprintf(" %d docs · by project ", len(m.filtered))
	}
	s.WriteString(M + components.Divider(innerW, label) + "\n")

	if len(m.filtered) == 0 {
		s.WriteString("\n")
		if len(m.docs) == 0 {
			empty := lipgloss.NewStyle().
				Foreground(components.ColorMuted).
				Padding(1, 2).
				Render("No documents yet.\n\nRun  mdpush share <file.md>  to share your first doc.")
			s.WriteString(M + empty + "\n")
		} else {
			s.WriteString(M + components.StyleMuted.Render("No matching documents.") + "\n")
		}
	} else {
		s.WriteString("\n")

		// Calculate visible window — reserve space for header and footer
		logoLines := 12
		metricsLines := 2
		headerExtra := 3 + metricsLines
		footerLines := 4
		maxVisible := m.height - logoLines - headerExtra - footerLines
		if m.err != "" || m.status != "" {
			maxVisible -= 2
		}
		if m.searching || m.searchQuery != "" {
			maxVisible -= 2
		}
		if maxVisible < 3 {
			maxVisible = 3
		}

		start := 0
		if m.cursor >= start+maxVisible {
			start = m.cursor - maxVisible + 1
		}
		end := start + maxVisible
		if end > len(m.filtered) {
			end = len(m.filtered)
		}

		prevGroup := ""
		for i := start; i < end; i++ {
			doc := m.filtered[i]

			// Group header
			if m.group != groupNone {
				var currentGroup string
				switch m.group {
				case groupCategory:
					currentGroup = doc.Category
					if currentGroup == "" {
						currentGroup = "uncategorized"
					}
				case groupProject:
					currentGroup = doc.Project
					if currentGroup == "" {
						currentGroup = "no project"
					}
				}
				if currentGroup != prevGroup {
					if prevGroup != "" {
						s.WriteString("\n")
					}
					s.WriteString(M + components.StyleBold.Render("  "+currentGroup) + "\n")
					prevGroup = currentGroup
				}
			}

			selected := i == m.cursor
			line := m.renderDocRow(doc, selected, w)
			s.WriteString(M + line + "\n")
		}

		if len(m.filtered) > maxVisible {
			s.WriteString("\n")
			s.WriteString(M + components.StyleMuted.Render(fmt.Sprintf("showing %d–%d of %d", start+1, end, len(m.filtered))) + "\n")
		}
	}

	// Footer keybinds
	s.WriteString("\n")
	s.WriteString(M + components.Divider(innerW, "") + "\n")
	s.WriteString(M + m.renderKeybar() + "\n")

	return s.String()
}

func (m Model) renderDocRow(doc DashDoc, selected bool, termWidth int) string {
	indicator := "  "
	if selected {
		indicator = components.StyleAccent.Render("▸ ")
	}

	// Doc ID
	docID := components.StyleMuted.Render(fmt.Sprintf("%-10s", doc.ID))

	// Title
	titleWidth := 35
	if termWidth > 100 {
		titleWidth = 45
	}
	if termWidth > 130 {
		titleWidth = 55
	}
	title := doc.Title
	if len(title) > titleWidth {
		title = title[:titleWidth-1] + "…"
	}
	// Pad title to fixed width for alignment
	for len(title) < titleWidth {
		title += " "
	}

	titleStyle := lipgloss.NewStyle().Foreground(components.ColorText)
	if selected {
		titleStyle = titleStyle.Bold(true).Foreground(lipgloss.Color("#a7f3d0")) // emerald-200
	}
	if !doc.HasKey {
		titleStyle = titleStyle.Foreground(components.ColorDim).Italic(true)
	}

	// Tags — build raw string, pad to fixed width, then style
	var tagParts []string
	if doc.Category != "" {
		tagParts = append(tagParts, doc.Category)
	}
	if doc.Project != "" {
		tagParts = append(tagParts, doc.Project)
	}
	tagRaw := strings.Join(tagParts, " · ")
	tagWidth := 25
	tags := components.StyleMuted.Render(fmt.Sprintf("%-*s", tagWidth, tagRaw))

	// Lock icon
	lockIcon := components.StyleMuted.Render("○")
	if doc.LockType == "strong" {
		lockIcon = components.StyleAccent.Render("●")
	}

	// Views — pad the raw string to fixed width before styling
	views := fmt.Sprintf("%dv", doc.CurrentViews)
	if doc.MaxViews != nil {
		views = fmt.Sprintf("%d/%dv", doc.CurrentViews, *doc.MaxViews)
	}
	viewStr := components.StyleMuted.Render(fmt.Sprintf("%-8s", views))

	// Date — always 6 chars ("Jan 02"), pad to be safe
	date := "      "
	if t, err := time.Parse(time.RFC3339, doc.CreatedAt); err == nil {
		date = t.Format("Jan 02")
	}
	dateStr := components.StyleMuted.Render(date)

	// Status badges
	var badges []string
	if doc.Revoked {
		badges = append(badges, lipgloss.NewStyle().Foreground(components.ColorError).Bold(true).Render("REVOKED"))
	}
	if doc.ExpiresAt != nil {
		if t, err := time.Parse(time.RFC3339, *doc.ExpiresAt); err == nil {
			if time.Now().After(t) {
				badges = append(badges, lipgloss.NewStyle().Foreground(components.ColorWarn).Bold(true).Render("EXPIRED"))
			}
		}
	}
	badgeStr := ""
	if len(badges) > 0 {
		badgeStr = "  " + strings.Join(badges, " ")
	}

	return indicator + docID + " " + titleStyle.Render(title) + " " + tags + " " + lockIcon + " " + viewStr + " " + dateStr + badgeStr
}

func (m Model) renderKeybar() string {
	sep := components.StyleDivider.Render(" · ")
	key := func(k, label string) string {
		return components.StyleAccent.Render(k) + components.StyleMuted.Render(" "+label)
	}

	parts := []string{
		key("↑↓", "navigate"),
		key("enter", "view"),
		key("c", "copy"),
		key("d", "delete"),
		key("r", "revoke"),
		key("/", "search"),
		key("f", "group"),
		key("R", "refresh"),
		key("q", "quit"),
	}

	return "  " + strings.Join(parts, sep)
}

// --- Viewer ---

func (m Model) openViewer() (Model, tea.Cmd) {
	if m.cursor >= len(m.filtered) {
		return m, nil
	}
	doc := m.filtered[m.cursor]

	if !doc.HasKey || doc.Content == "" {
		m.err = "No decryption key available for this document"
		return m, nil
	}

	rendered, err := glamour.Render(doc.Content, "dark")
	if err != nil {
		rendered = doc.Content
	}

	vp := viewport.New(m.width, m.height-6)
	vp.SetContent(rendered)

	m.viewport = vp
	m.viewerReady = true
	m.viewerDoc = &doc
	m.state = stateViewer

	return m, nil
}

func (m Model) updateViewer(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok {
		switch key.String() {
		case "q", "esc":
			m.state = stateList
			m.viewerReady = false
			return m, nil
		case "c":
			if m.viewerDoc != nil {
				m.copyLinkForDoc(*m.viewerDoc)
			}
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

func (m Model) viewViewer() string {
	var s strings.Builder
	M := components.Margin
	w := m.width
	if w < 60 {
		w = 80
	}
	innerW := w - len(M)*2

	// Header
	s.WriteString("\n")
	title := "[encrypted]"
	if m.viewerDoc != nil {
		title = m.viewerDoc.Title
	}
	s.WriteString(M + components.StyleAccent.Render("░█") + " " + components.StyleBold.Render(title) + "\n")

	// Metadata line
	if m.viewerDoc != nil {
		var meta []string
		if m.viewerDoc.Category != "" {
			meta = append(meta, m.viewerDoc.Category)
		}
		if m.viewerDoc.Project != "" {
			meta = append(meta, m.viewerDoc.Project)
		}
		if t, err := time.Parse(time.RFC3339, m.viewerDoc.CreatedAt); err == nil {
			meta = append(meta, t.Format("Jan 02, 2006"))
		}
		if len(meta) > 0 {
			s.WriteString(M + components.StyleMuted.Render("  "+strings.Join(meta, " · ")) + "\n")
		}
	}

	s.WriteString(M + components.Divider(innerW, "") + "\n")

	// Viewport content
	s.WriteString(m.viewport.View())

	// Footer
	s.WriteString("\n")
	s.WriteString(M + components.Divider(innerW, "") + "\n")

	if m.status != "" {
		s.WriteString(M + components.StyleAccent.Render("✓ "+m.status) + "\n")
	} else {
		sep := components.StyleDivider.Render(" · ")
		hint := M +
			components.StyleAccent.Render("↑↓") + components.StyleMuted.Render(" scroll") + sep +
			components.StyleAccent.Render("c") + components.StyleMuted.Render(" copy link") + sep +
			components.StyleAccent.Render("q") + components.StyleMuted.Render(" back")
		s.WriteString(hint + "\n")
	}

	return s.String()
}

// --- Confirm dialog ---

func (m Model) updateConfirm(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok {
		switch key.String() {
		case "y", "Y":
			return m.executeAction()
		case "n", "N", "esc":
			m.state = stateList
			m.pendingAction = actionNone
			return m, nil
		}
	}
	return m, nil
}

func (m Model) viewConfirm() string {
	if m.cursor >= len(m.filtered) {
		return ""
	}
	doc := m.filtered[m.cursor]
	w := m.width
	if w < 60 {
		w = 80
	}

	var icon, prompt, detail string
	switch m.pendingAction {
	case actionDelete:
		icon = components.StyleError.Render("✗")
		prompt = "Delete this document?"
		detail = "This cannot be undone. The encrypted payload will be permanently removed."
	case actionRevoke:
		icon = components.StyleBadge.Render("!")
		prompt = "Revoke access?"
		detail = "Readers will no longer be able to view this document."
	case actionRestore:
		icon = components.StyleAccent.Render("↺")
		prompt = "Restore access?"
		detail = "Readers will be able to view this document again."
	}

	title := doc.Title
	titleLine := components.StyleBold.Render(title)

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(components.ColorDim).
		Padding(1, 3).
		Width(min(60, w-8)).
		Render(
			icon + "  " + prompt + "\n\n" +
				"  " + titleLine + "\n\n" +
				components.StyleMuted.Render("  "+detail) + "\n\n" +
				components.StyleAccent.Render("  y") + components.StyleMuted.Render(" confirm") +
				components.StyleDivider.Render("  ·  ") +
				components.StyleAccent.Render("n") + components.StyleMuted.Render(" cancel"),
		)

	M := components.Margin
	var s strings.Builder
	s.WriteString("\n")
	s.WriteString(components.RenderLogo())
	s.WriteString("\n\n")
	for _, line := range strings.Split(box, "\n") {
		s.WriteString(M + "  " + line + "\n")
	}
	s.WriteString("\n")
	return s.String()
}

func (m Model) executeAction() (Model, tea.Cmd) {
	if m.cursor >= len(m.filtered) {
		return m, nil
	}
	doc := m.filtered[m.cursor]
	m.loading = true

	switch m.pendingAction {
	case actionDelete:
		return m, func() tea.Msg {
			if _, err := m.client.DeleteDoc(doc.ID); err != nil {
				return actionErrMsg{err}
			}
			if ks, err := keystore.New(); err == nil {
				_ = ks.Load()
				_ = ks.Delete(doc.ID)
			}
			return actionDoneMsg{fmt.Sprintf("Deleted \"%s\"", doc.Title)}
		}
	case actionRevoke:
		return m, func() tea.Msg {
			if _, err := m.client.RevokeDoc(doc.ID); err != nil {
				return actionErrMsg{err}
			}
			return actionDoneMsg{fmt.Sprintf("Revoked \"%s\"", doc.Title)}
		}
	case actionRestore:
		return m, func() tea.Msg {
			if _, err := m.client.RestoreDoc(doc.ID); err != nil {
				return actionErrMsg{err}
			}
			return actionDoneMsg{fmt.Sprintf("Restored \"%s\"", doc.Title)}
		}
	}

	return m, nil
}

// --- Actions ---

func (m *Model) copyLink() {
	if m.cursor >= len(m.filtered) {
		return
	}
	doc := m.filtered[m.cursor]
	m.copyLinkForDoc(doc)
}

func (m *Model) copyLinkForDoc(doc DashDoc) {
	key := m.keys.Get(doc.ID)
	if key == nil {
		m.err = "No key available — can't build link"
		return
	}

	fragment := crypto.KeyToFragment(key)
	url := fmt.Sprintf("https://www.mdpush.io/d/%s#%s", doc.ID, fragment)

	if err := clipboard.WriteAll(url); err != nil {
		m.status = "Link: " + url
		return
	}
	m.status = "Copied to clipboard"
}
