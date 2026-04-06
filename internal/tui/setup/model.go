package setup

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/mdpush-io/cli/internal/api"
	"github.com/mdpush-io/cli/internal/auth"
	"github.com/mdpush-io/cli/internal/tui/components"
)

// step tracks which screen we're on.
type step int

const (
	stepMethod step = iota
	stepEmail
	stepCode
	stepGitHub
	stepDone
)

// --- Messages ---

type codeSentMsg struct{}
type codeSentErrMsg struct{ err error }
type verifiedMsg struct {
	token     string
	isNewUser bool
}
type verifiedErrMsg struct{ err error }
type registeredMsg struct {
	result *auth.SetupResult
}
type loggedInMsg struct {
	result *auth.LoginResult
}
type authErrMsg struct{ err error }

// GitHub device flow messages
type ghStartedMsg struct{ start *auth.GitHubStartResult }
type ghStartErrMsg struct{ err error }
type ghPollTickMsg struct{}
type ghPollResultMsg struct{ result auth.GitHubPollResult }

// --- Model ---

// Model is the Bubble Tea model for the first-run setup flow.
type Model struct {
	step  step
	width int

	// Inputs
	emailInput textinput.Model
	codeInput  textinput.Model

	// State
	email             string
	verificationToken string
	isNewUser         bool
	err               string
	loading           bool
	loadingMsg        string

	// GitHub device flow state
	ghDeviceCode      string
	ghUserCode        string
	ghVerificationURI string
	ghInterval        int

	// Method-step cursor (0 = GitHub, 1 = Email)
	methodCursor int

	// If true, Init() will immediately kick off the GitHub device flow
	// (used by `mdpush login --github`).
	autoStartGitHub bool

	// API
	client *api.Client
}

// New creates a new setup TUI model.
func New() Model {
	ei := textinput.New()
	ei.Placeholder = "you@example.com"
	ei.CharLimit = 254
	ei.Focus()

	ci := textinput.New()
	ci.Placeholder = "000000"
	ci.CharLimit = 6

	return Model{
		step:       stepMethod,
		emailInput: ei,
		codeInput:  ci,
		client:     api.NewClient(),
	}
}

// NewEmail creates a setup model that skips the method picker and goes
// straight to email/OTP login. Used by `mdpush login --otp`.
func NewEmail() Model {
	m := New()
	m.step = stepEmail
	m.emailInput.Focus()
	return m
}

// NewGitHub creates a setup model that skips the method picker and starts
// the GitHub device flow immediately. Used by `mdpush login --github`.
func NewGitHub() Model {
	m := New()
	m.step = stepGitHub
	m.loading = true
	m.loadingMsg = "contacting GitHub..."
	m.autoStartGitHub = true
	return m
}

func (m Model) Init() tea.Cmd {
	if m.autoStartGitHub {
		// Mirror what startGitHub does, but as an Init cmd so it survives
		// the program's startup handshake.
		return func() tea.Msg {
			start, err := auth.RequestGitHubDevice(m.client)
			if err != nil {
				return ghStartErrMsg{err}
			}
			return ghStartedMsg{start}
		}
	}
	return textinput.Blink
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		return m, nil

	case tea.KeyMsg:
		// Global quit
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
		// Clear error on any keypress
		if m.err != "" && msg.String() != "enter" {
			m.err = ""
		}
	}

	switch m.step {
	case stepMethod:
		return m.updateMethod(msg)
	case stepEmail:
		return m.updateEmail(msg)
	case stepCode:
		return m.updateCode(msg)
	case stepGitHub:
		return m.updateGitHub(msg)
	case stepDone:
		return m.updateDone(msg)
	}

	return m, nil
}

func (m Model) View() string {
	var s strings.Builder

	s.WriteString("\n")
	s.WriteString(components.RenderLogoWithMotto())
	s.WriteString("\n\n")

	switch m.step {
	case stepMethod:
		m.viewMethod(&s)
	case stepEmail:
		m.viewEmail(&s)
	case stepCode:
		m.viewCode(&s)
	case stepGitHub:
		m.viewGitHub(&s)
	case stepDone:
		m.viewDone(&s)
	}

	M := components.Margin
	if m.err != "" {
		s.WriteString("\n" + M + components.StyleError.Render("✗ "+m.err) + "\n")
	}

	if m.loading {
		s.WriteString("\n" + M + components.StyleMuted.Render(m.loadingMsg) + "\n")
	}

	s.WriteString("\n")
	return s.String()
}

// --- Email step ---

func (m Model) updateEmail(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok && key.String() == "enter" {
		email := strings.TrimSpace(m.emailInput.Value())
		if email == "" || !strings.Contains(email, "@") {
			m.err = "Please enter a valid email address"
			return m, nil
		}
		m.email = email
		m.loading = true
		m.loadingMsg = "sending code..."
		return m, func() tea.Msg {
			_, err := m.client.SendCode(email)
			if err != nil {
				return codeSentErrMsg{err}
			}
			return codeSentMsg{}
		}
	}

	if _, ok := msg.(codeSentMsg); ok {
		m.loading = false
		m.step = stepCode
		m.codeInput.Focus()
		return m, textinput.Blink
	}

	if errMsg, ok := msg.(codeSentErrMsg); ok {
		m.loading = false
		m.err = fmt.Sprintf("Failed to send code: %v", errMsg.err)
		return m, nil
	}

	var cmd tea.Cmd
	m.emailInput, cmd = m.emailInput.Update(msg)
	return m, cmd
}

func (m Model) viewEmail(s *strings.Builder) {
	M := components.Margin
	s.WriteString(M + components.StyleMuted.Render("Welcome! Let's get you set up.") + "\n")
	s.WriteString("\n")
	s.WriteString(M + "Email: " + m.emailInput.View() + "\n")
	s.WriteString(M + components.StyleKeyHint.Render("enter to continue · ctrl+c to quit") + "\n")
}

// --- Code step ---

func (m Model) updateCode(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok && key.String() == "enter" {
		code := strings.TrimSpace(m.codeInput.Value())
		if len(code) != 6 {
			m.err = "Code must be 6 digits"
			return m, nil
		}
		m.loading = true
		m.loadingMsg = "verifying..."
		return m, func() tea.Msg {
			resp, err := m.client.VerifyCode(m.email, code)
			if err != nil {
				return verifiedErrMsg{err}
			}
			return verifiedMsg{token: resp.VerificationToken, isNewUser: resp.IsNewUser}
		}
	}

	if vmsg, ok := msg.(verifiedMsg); ok {
		m.loading = false
		m.verificationToken = vmsg.token
		m.isNewUser = vmsg.isNewUser

		// Register or login immediately — no passphrase needed
		m.loading = true
		if m.isNewUser {
			m.loadingMsg = "creating account..."
			return m, func() tea.Msg {
				result, err := auth.Setup(m.client, m.verificationToken, m.email)
				if err != nil {
					return authErrMsg{err}
				}
				return registeredMsg{result}
			}
		}
		m.loadingMsg = "logging in..."
		return m, func() tea.Msg {
			result, err := auth.Login(m.client, m.verificationToken, m.email)
			if err != nil {
				return authErrMsg{err}
			}
			return loggedInMsg{result}
		}
	}

	if rmsg, ok := msg.(registeredMsg); ok {
		m.loading = false
		auth.Persist(rmsg.result.Session)
		m.step = stepDone
		return m, nil
	}

	if lmsg, ok := msg.(loggedInMsg); ok {
		m.loading = false
		auth.Persist(lmsg.result.Session)
		m.step = stepDone
		return m, nil
	}

	if errMsg, ok := msg.(authErrMsg); ok {
		m.loading = false
		m.err = fmt.Sprintf("%v", errMsg.err)
		return m, nil
	}

	if errMsg, ok := msg.(verifiedErrMsg); ok {
		m.loading = false
		m.err = fmt.Sprintf("Invalid code: %v", errMsg.err)
		m.codeInput.SetValue("")
		return m, nil
	}

	var cmd tea.Cmd
	m.codeInput, cmd = m.codeInput.Update(msg)
	return m, cmd
}

func (m Model) viewCode(s *strings.Builder) {
	M := components.Margin
	s.WriteString(M + components.StyleAccent.Render("✓ Code sent to "+m.email) + "\n")
	s.WriteString("\n")
	s.WriteString(M + "Code: " + m.codeInput.View() + "\n")
	s.WriteString(M + components.StyleKeyHint.Render("enter to verify · ctrl+c to quit") + "\n")
}

// --- Done step ---

func (m Model) updateDone(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok {
		switch key.String() {
		case "enter", "q":
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m Model) viewDone(s *strings.Builder) {
	M := components.Margin
	name := m.email
	if idx := strings.Index(m.email, "@"); idx > 0 {
		name = m.email[:idx]
	}

	if m.isNewUser {
		s.WriteString(M + components.StyleAccent.Render("✓ You're all set, "+name+"!") + "\n")
	} else {
		s.WriteString(M + components.StyleAccent.Render("✓ Welcome back, "+name+"!") + "\n")
	}

	s.WriteString("\n")
	s.WriteString(M + "Run " + components.StyleBold.Render("mdpush share <file.md>") + " to share your first doc.\n")
	s.WriteString("\n")
	s.WriteString(M + components.StyleKeyHint.Render("enter to exit") + "\n")
}

// --- Method step (GitHub vs Email) ---

func (m Model) updateMethod(msg tea.Msg) (tea.Model, tea.Cmd) {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}
	switch key.String() {
	case "up", "k":
		if m.methodCursor > 0 {
			m.methodCursor--
		}
	case "down", "j":
		if m.methodCursor < 1 {
			m.methodCursor++
		}
	case "g", "G":
		m.methodCursor = 0
		return m.startGitHub()
	case "e", "E":
		m.methodCursor = 1
		m.step = stepEmail
		m.emailInput.Focus()
		return m, textinput.Blink
	case "enter":
		if m.methodCursor == 0 {
			return m.startGitHub()
		}
		m.step = stepEmail
		m.emailInput.Focus()
		return m, textinput.Blink
	}
	return m, nil
}

func (m Model) viewMethod(s *strings.Builder) {
	M := components.Margin
	s.WriteString(M + components.StyleMuted.Render("Welcome! How would you like to sign in?") + "\n\n")

	options := []string{"Continue with GitHub", "Use email instead"}
	for i, opt := range options {
		prefix := "  "
		line := opt
		if i == m.methodCursor {
			prefix = "▸ "
			line = components.StyleAccent.Render(opt)
		}
		s.WriteString(M + prefix + line + "\n")
	}
	s.WriteString("\n" + M + components.StyleKeyHint.Render("↑/↓ to choose · enter to confirm · g github · e email · ctrl+c to quit") + "\n")
}

// --- GitHub device flow step ---

func (m Model) startGitHub() (tea.Model, tea.Cmd) {
	m.step = stepGitHub
	m.loading = true
	m.loadingMsg = "contacting GitHub..."
	return m, func() tea.Msg {
		start, err := auth.RequestGitHubDevice(m.client)
		if err != nil {
			return ghStartErrMsg{err}
		}
		return ghStartedMsg{start}
	}
}

func (m Model) updateGitHub(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case ghStartErrMsg:
		m.loading = false
		m.err = fmt.Sprintf("%v", msg.err)
		m.step = stepMethod
		return m, nil

	case ghStartedMsg:
		m.loading = false
		m.ghDeviceCode = msg.start.DeviceCode
		m.ghUserCode = msg.start.UserCode
		m.ghVerificationURI = msg.start.VerificationURI
		m.ghInterval = msg.start.Interval
		// Best-effort browser launch — failure is silent, the user can
		// always type the URL themselves.
		_ = openBrowser(m.ghVerificationURI)
		return m, schedulePoll(m.ghInterval)

	case ghPollTickMsg:
		deviceCode := m.ghDeviceCode
		client := m.client
		return m, func() tea.Msg {
			return ghPollResultMsg{result: auth.PollGitHubDevice(client, deviceCode)}
		}

	case ghPollResultMsg:
		r := msg.result
		switch {
		case r.Err != nil:
			m.err = fmt.Sprintf("%v", r.Err)
			m.step = stepMethod
			return m, nil
		case r.Session != nil:
			if err := auth.Persist(r.Session); err != nil {
				m.err = fmt.Sprintf("saving session: %v", err)
				m.step = stepMethod
				return m, nil
			}
			m.email = r.Session.Email
			m.isNewUser = false // we don't currently distinguish for GitHub
			m.step = stepDone
			return m, nil
		default: // pending
			if r.NewInterval > 0 {
				m.ghInterval = r.NewInterval
			}
			return m, schedulePoll(m.ghInterval)
		}
	}
	return m, nil
}

func (m Model) viewGitHub(s *strings.Builder) {
	M := components.Margin
	if m.ghUserCode == "" {
		// still loading the device code
		return
	}
	s.WriteString(M + components.StyleMuted.Render("To finish signing in:") + "\n\n")
	s.WriteString(M + "1. Open " + components.StyleBold.Render(m.ghVerificationURI) + "\n")
	s.WriteString(M + "2. Enter the code: " + components.StyleAccent.Render(m.ghUserCode) + "\n\n")
	s.WriteString(M + components.StyleMuted.Render("Waiting for you to authorize on GitHub...") + "\n")
	s.WriteString("\n" + M + components.StyleKeyHint.Render("ctrl+c to cancel") + "\n")
}

func schedulePoll(interval int) tea.Cmd {
	if interval <= 0 {
		interval = 5
	}
	return tea.Tick(time.Duration(interval)*time.Second, func(time.Time) tea.Msg {
		return ghPollTickMsg{}
	})
}

func openBrowser(url string) error {
	var cmd string
	var args []string
	switch runtime.GOOS {
	case "darwin":
		cmd = "open"
	case "windows":
		cmd = "rundll32"
		args = []string{"url.dll,FileProtocolHandler"}
	default:
		cmd = "xdg-open"
	}
	args = append(args, url)
	return exec.Command(cmd, args...).Start()
}
