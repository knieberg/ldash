package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/knieberg/ldash/internal/config"
	ldapclient "github.com/knieberg/ldash/internal/ldap"
)

func (m model) viewMenu() string {
	var b strings.Builder
	b.WriteString(headerStyle.Render("Navigate"))
	b.WriteString("\n\n")
	aw := appWidth(m.width)
	labelW := 20
	for i, item := range m.menuItems {
		num := fmt.Sprintf("%d", i+1)
		label := item.label
		hint := item.hint
		pad := aw - 8 - labelW - 4
		if pad < 8 {
			pad = 8
		}
		hintShown := truncRunes(hint, pad)
		raw := fmt.Sprintf("%s  %-*s  %s", num, labelW, label, hintShown)
		switch {
		case i == m.menuCursor && item.enabled:
			b.WriteString(selStyle.Render("▸ " + raw))
		case i == m.menuCursor && !item.enabled:
			b.WriteString(selStyle.Render("▸ " + raw))
		case !item.enabled:
			b.WriteString(disabledStyle.Render("  " + raw))
		default:
			b.WriteString("  " + raw)
		}
		b.WriteString("\n")
	}
	b.WriteString("\n")
	b.WriteString(mutedStyle.Render("Press number keys 1–7 or Enter to open a view."))
	return b.String()
}

func (m model) viewDashboard() string {
	var b strings.Builder
	aw := appWidth(m.width)
	inner := aw - 6
	if inner < 20 {
		inner = 20
	}
	lines := []string{
		fmt.Sprintf("Server:  %s (%s)", m.cfg.Server.URL, m.cfg.Server.TLSMode),
		fmt.Sprintf("Base DN: %s", m.cfg.BaseDN),
		fmt.Sprintf("People:  %s", m.cfg.PeopleDN()),
		fmt.Sprintf("Groups:  %s", m.cfg.GroupsDN()),
	}
	b.WriteString(boxStyle.Width(inner).Render(strings.Join(lines, "\n")))
	b.WriteString("\n\n")
	if m.pingResult != nil {
		if m.pingResult.OK {
			b.WriteString(okStyle.Render("Connection: OK"))
		} else {
			b.WriteString(errStyle.Render("Connection: FAILED"))
		}
	} else {
		b.WriteString(mutedStyle.Render("Press r to test LDAP connection"))
	}
	return b.String()
}

func (m model) viewIntegration() string {
	var b strings.Builder
	aw := appWidth(m.width)
	inner := aw - 6
	if inner < 20 {
		inner = 20
	}
	b.WriteString(headerStyle.Render("Values from local runtime config only"))
	b.WriteString("\n\n")
	b.WriteString(boxStyle.Width(inner).Render(strings.Join([]string{
		fmt.Sprintf("Server URI:   %s", m.cfg.Server.URL),
		fmt.Sprintf("Base DN:      %s", m.cfg.BaseDN),
		fmt.Sprintf("Bind DN:      %s", m.cfg.BindDN),
		fmt.Sprintf("Users base:   %s", m.cfg.PeopleDN()),
		fmt.Sprintf("Groups base:  %s", m.cfg.GroupsDN()),
		fmt.Sprintf("User filter:  %s", m.cfg.Search.UserFilter),
	}, "\n")))
	b.WriteString("\n\n")
	if m.integ.SelfServiceURL != "" {
		fmt.Fprintf(&b, "Self-service URL: %s\n", m.integ.SelfServiceURL)
	}
	if m.integ.OIDCIssuer != "" {
		fmt.Fprintf(&b, "OIDC issuer: %s\n", m.integ.OIDCIssuer)
	}
	if m.integ.OIDCProvider != "" {
		fmt.Fprintf(&b, "OIDC provider: %s\n", m.integ.OIDCProvider)
	}
	if len(m.integ.OnboardingChecklist) > 0 {
		b.WriteString("\nOnboarding checklist:\n")
		for _, step := range m.integ.OnboardingChecklist {
			b.WriteString("  - " + step + "\n")
		}
	}
	if m.integ.SelfServiceURL == "" && m.integ.OIDCIssuer == "" && len(m.integ.OnboardingChecklist) == 0 {
		b.WriteString(mutedStyle.Render("Optional: create ~/.config/ldash/integration.yaml for extra hints."))
		b.WriteString("\n")
		b.WriteString(mutedStyle.Render("Press Esc to return to the main menu."))
	}
	return b.String()
}

func (m model) viewSettings() string {
	aw := appWidth(m.width)
	inner := aw - 6
	if inner < 20 {
		inner = 20
	}
	userTpl, userErr := config.LoadUserTemplate(m.cfg)
	groupTpl, groupErr := config.LoadGroupTemplate(m.cfg)
	userLine := "not loaded"
	if userErr == nil && userTpl != nil {
		userLine = userTpl.Name
	} else if userErr != nil {
		userLine = userErr.Error()
	}
	groupLine := "not loaded"
	if groupErr == nil && groupTpl != nil {
		groupLine = groupTpl.Name
	} else if groupErr != nil {
		groupLine = groupErr.Error()
	}
	integLine := "absent"
	if _, err := config.LoadIntegration(); err == nil {
		integLine = "present (optional hints)"
	}
	return boxStyle.Width(inner).Render(strings.Join([]string{
		fmt.Sprintf("Server: %s", m.cfg.Server.URL),
		fmt.Sprintf("TLS mode: %s", m.cfg.Server.TLSMode),
		fmt.Sprintf("Bind DN: %s", m.cfg.BindDN),
		fmt.Sprintf("Samba domain SID set: %v", m.cfg.Samba.DomainSID != ""),
		fmt.Sprintf("User template: %s", userLine),
		fmt.Sprintf("Group template: %s", groupLine),
		fmt.Sprintf("Integration file: %s", integLine),
	}, "\n"))
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
