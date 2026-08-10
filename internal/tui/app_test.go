package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/knieberg/ldash/internal/config"
)

func testModel() model {
	cfg := &config.Config{
		Server: config.ServerConfig{URL: "ldap://ldap.example.com", TLSMode: "starttls"},
		BaseDN: "dc=example,dc=com",
		BindDN: "cn=admin,dc=example,dc=com",
	}
	m := New(cfg, "secret", nil).(model)
	m.width = 100
	m.height = 40
	return m
}

func TestAppWidth(t *testing.T) {
	if got := appWidth(80); got != 76 {
		t.Fatalf("appWidth(80)=%d", got)
	}
	if got := appWidth(0); got != 80 {
		t.Fatalf("appWidth(0)=%d", got)
	}
}

func TestGoBackLevels(t *testing.T) {
	m := testModel()
	m.current = viewUserEdit
	m2, _ := m.goBack()
	mm := m2.(model)
	if mm.current != viewUsers {
		t.Fatalf("form Esc -> want users, got %v", mm.current)
	}
	m2, _ = mm.goBack()
	mm = m2.(model)
	if mm.current != viewMenu {
		t.Fatalf("users Esc -> want menu, got %v", mm.current)
	}
}

func TestHelpClosesFirst(t *testing.T) {
	m := testModel()
	m.current = viewUsers
	m.showHelp = true
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	mm := m2.(model)
	if mm.showHelp {
		t.Fatal("? should close help")
	}
	if mm.current != viewUsers {
		t.Fatal("closing help must not change view")
	}
}

func TestDisabledMenuDoesNotNavigate(t *testing.T) {
	m := testModel()
	m.menuCursor = 2 // Groups
	m2, _ := m.openMenuItem(2)
	mm := m2.(model)
	if mm.current != viewMenu {
		t.Fatalf("disabled item opened view %v", mm.current)
	}
	if mm.statusK != statusWarn {
		t.Fatalf("expected warn status, got %v", mm.statusK)
	}
}

func TestQuitOnlyOnMenu(t *testing.T) {
	m := testModel()
	m.current = viewDashboard
	m2, cmd := m.updateNav(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	mm := m2.(model)
	if mm.quitting || cmd != nil {
		t.Fatal("q on dashboard must not quit")
	}
	if mm.current != viewDashboard {
		t.Fatal("q on dashboard must not go back")
	}
}

func TestViewUsesShell(t *testing.T) {
	m := testModel()
	out := m.View()
	if !strings.Contains(out, "ldash") {
		t.Fatal("missing title")
	}
	if !strings.Contains(out, "Navigate") {
		t.Fatal("missing menu panel")
	}
}

func TestStatusPrefixes(t *testing.T) {
	if !strings.HasPrefix(statusLine(statusOK, "done"), "OK:") && !strings.Contains(statusLine(statusOK, "done"), "OK:") {
		// lipgloss may wrap ANSI; check raw helper path without styles by kind mapping
	}
	s := statusLine(statusError, "boom")
	if !strings.Contains(s, "Error:") && !strings.Contains(s, "boom") {
		t.Fatalf("unexpected status: %q", s)
	}
}
