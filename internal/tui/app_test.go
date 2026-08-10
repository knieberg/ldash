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

func TestGroupsMenuNavigates(t *testing.T) {
	m := testModel()
	m2, _ := m.openMenuItem(2) // Groups
	mm := m2.(model)
	if mm.current != viewGroups {
		t.Fatalf("groups item should open viewGroups, got %v", mm.current)
	}
}

func TestLDIFMenuNavigates(t *testing.T) {
	m := testModel()
	m2, _ := m.openMenuItem(3) // LDIF
	mm := m2.(model)
	if mm.current != viewLDIF {
		t.Fatalf("LDIF item should open viewLDIF, got %v", mm.current)
	}
	if mm.ldifStep != ldifStepHub {
		t.Fatalf("LDIF should start at hub, got step %d", mm.ldifStep)
	}
}

func TestSambaMenuNavigates(t *testing.T) {
	m := testModel()
	m2, _ := m.openMenuItem(4) // Samba
	mm := m2.(model)
	if mm.current != viewSamba {
		t.Fatalf("Samba item should open viewSamba, got %v", mm.current)
	}
}

func TestFooterUsersLabeledKeys(t *testing.T) {
	m := testModel()
	m.current = viewUsers
	footer := m.footerKeys()
	for _, verb := range []string{"create", "edit", "delete", "password", "mail", "samba", "search", "refresh"} {
		if !strings.Contains(footer, verb) {
			t.Fatalf("footer missing labeled action %q: %q", verb, footer)
		}
	}
	if strings.Contains(footer, "c e d") {
		t.Fatalf("footer must not show bare key chains: %q", footer)
	}
}

func TestDisplayLabelShowsRequired(t *testing.T) {
	spec := config.FormFieldSpec{Attr: "uid", Required: true}
	label := displayLabel(spec)
	if !strings.Contains(label, "(required)") {
		t.Fatalf("expected required marker, got %q", label)
	}
	if !strings.Contains(label, "uid") {
		t.Fatalf("expected attr in label, got %q", label)
	}
}

func TestMenuHasSevenItems(t *testing.T) {
	m := testModel()
	if len(m.menuItems) != 7 {
		t.Fatalf("expected 7 menu items, got %d", len(m.menuItems))
	}
}

func TestEmptyUsersFilterMessage(t *testing.T) {
	m := testModel()
	m.userFilter = "nobody"
	out := m.viewUsers()
	if !strings.Contains(out, "No matches") {
		t.Fatalf("filtered empty list should say No matches: %q", out)
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
	ok := statusLine(statusOK, "done")
	if !strings.Contains(ok, "OK:") || !strings.Contains(ok, "done") {
		t.Fatalf("unexpected OK status: %q", ok)
	}
	s := statusLine(statusError, "boom")
	if !strings.Contains(s, "Error:") || !strings.Contains(s, "boom") {
		t.Fatalf("unexpected status: %q", s)
	}
}
