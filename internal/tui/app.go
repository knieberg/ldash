package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/ldash-sh/ldash/internal/config"
	ldapclient "github.com/ldash-sh/ldash/internal/ldap"
)

type view int

const (
	viewDashboard view = iota
)

type pingMsg struct {
	result ldapclient.PingResult
	err    error
}

type model struct {
	cfg        *config.Config
	password   string
	current    view
	width      int
	height     int
	status     string
	pingResult *ldapclient.PingResult
	ready      bool
	quitting   bool
}

var (
	titleStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("69"))
	okStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	errStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	mutedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	boxStyle   = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(1, 2)
)

func New(cfg *config.Config, password string) tea.Model {
	return model{
		cfg:      cfg,
		password: password,
		current:  viewDashboard,
		status:   "Press r to test LDAP connection, q to quit",
	}
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			m.quitting = true
			return m, tea.Quit
		case "r":
			m.status = "Testing connection..."
			return m, m.pingCmd()
		}
	case pingMsg:
		if msg.err != nil {
			m.status = msg.err.Error()
			m.pingResult = nil
		} else {
			m.pingResult = &msg.result
			if msg.result.OK {
				m.status = fmt.Sprintf("Connected in %s", msg.result.Duration.Round(1e6))
			} else {
				m.status = msg.result.Message
			}
		}
		m.ready = true
	}
	return m, nil
}

func (m model) pingCmd() tea.Cmd {
	return func() tea.Msg {
		client := ldapclient.NewClient(m.cfg)
		result := client.Ping(m.password)
		if !result.OK {
			return pingMsg{result: result, err: fmt.Errorf("%s", result.Message)}
		}
		return pingMsg{result: result}
	}
}

func (m model) View() string {
	if m.quitting {
		return ""
	}

	var b strings.Builder
	b.WriteString(titleStyle.Render("ldash — LDAP Admin Shell"))
	b.WriteString("\n")
	b.WriteString(mutedStyle.Render("Terminal UI for OpenLDAP administration"))
	b.WriteString("\n\n")

	lines := []string{
		fmt.Sprintf("Server:  %s (%s)", m.cfg.Server.URL, m.cfg.Server.TLSMode),
		fmt.Sprintf("Base DN: %s", m.cfg.BaseDN),
		fmt.Sprintf("People:  %s", m.cfg.PeopleDN()),
		fmt.Sprintf("Groups:  %s", m.cfg.GroupsDN()),
	}
	b.WriteString(boxStyle.Render(strings.Join(lines, "\n")))
	b.WriteString("\n\n")

	if m.pingResult != nil {
		label := "Connection"
		if m.pingResult.OK {
			b.WriteString(okStyle.Render(label + ": OK"))
		} else {
			b.WriteString(errStyle.Render(label + ": FAILED"))
		}
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(mutedStyle.Render("Navigation (more views coming soon):"))
	b.WriteString("\n")
	b.WriteString("  Users · Groups · Samba · Integration Guide · Settings\n\n")
	b.WriteString(mutedStyle.Render("Keys: r test connection · q quit · ? help (planned)"))
	b.WriteString("\n\n")
	b.WriteString(m.status)

	return b.String()
}

func Run(cfg *config.Config, password string) error {
	p := tea.NewProgram(New(cfg, password), tea.WithAltScreen())
	_, err := p.Run()
	return err
}
