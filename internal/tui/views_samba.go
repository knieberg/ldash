package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/knieberg/ldash/internal/config"
)

func (m model) viewSambaHub() string {
	var b strings.Builder
	b.WriteString(headerStyle.Render("Samba attribute overview"))
	b.WriteString("\n\n")
	tpl, _ := config.LoadUserTemplate(m.cfg)
	hasSambaOC := tpl != nil && tpl.HasSambaAccount()
	lines := []string{
		fmt.Sprintf("domain_sid configured: %v", m.cfg.Samba.DomainSID != ""),
		fmt.Sprintf("User template includes sambaSamAccount: %v", hasSambaOC),
	}
	b.WriteString(boxStyle.Render(strings.Join(lines, "\n")))
	b.WriteString("\n\n")
	b.WriteString("Core attributes:\n")
	for _, attr := range []string{"sambaSID", "sambaAcctFlags", "sambaNTPassword"} {
		label, help := sambaFieldHelp(attr)
		fmt.Fprintf(&b, "  %s — %s\n", label, help)
	}
	b.WriteString("\n")
	if !hasSambaOC || m.cfg.Samba.DomainSID == "" {
		b.WriteString(warnStyle.Render("Warn: Samba user creation needs sambaSamAccount in the user template and samba.domain_sid in config."))
	} else {
		b.WriteString(mutedStyle.Render("Open Users and press s samba on a selected user for per-account status."))
	}
	return b.String()
}

func (m model) viewSambaUser() string {
	var b strings.Builder
	u := m.sambaUser
	if u == nil {
		return mutedStyle.Render("No user selected.")
	}
	b.WriteString(headerStyle.Render("Samba status: " + u.UID))
	b.WriteString("\n\n")
	sidLabel, sidHelp := sambaFieldHelp("sambaSID")
	b.WriteString(fmt.Sprintf("%s: %s\n", sidLabel, sambaPresent(u.SambaSID)))
	b.WriteString(mutedStyle.Render("  "+sidHelp))
	b.WriteString("\n")
	flagLabel, flagHelp := sambaFieldHelp("sambaAcctFlags")
	b.WriteString(fmt.Sprintf("%s: %s\n", flagLabel, sambaPresent(u.SambaFlags)))
	b.WriteString(mutedStyle.Render("  "+flagHelp))
	b.WriteString("\n")
	ntLabel, ntHelp := sambaFieldHelp("sambaNTPassword")
	nt := "Missing"
	if m.sambaNTPresent {
		nt = "Present"
	}
	b.WriteString(fmt.Sprintf("%s: %s\n", ntLabel, nt))
	b.WriteString(mutedStyle.Render("  "+ntHelp))
	b.WriteString("\n\n")
	b.WriteString(mutedStyle.Render("Press e edit flags · Esc back"))
	return b.String()
}

func (m model) updateSambaUser(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case keyEdit:
		m.initSambaFlagsForm()
		m.current = viewSambaFlags
		return m, textinput.Blink
	}
	return m, nil
}

func (m *model) initSambaFlagsForm() {
	w := formInputWidth(m.width)
	flags := ""
	if m.sambaUser != nil {
		flags = m.sambaUser.SambaFlags
		m.formDN = m.sambaUser.DN
		m.formUID = m.sambaUser.UID
	}
	m.formInputs = []textinput.Model{newInput("[U          ]", false, w)}
	m.formInputs[0].SetValue(flags)
	m.formSpecs = []config.FormFieldSpec{{Attr: "sambaAcctFlags", Required: true}}
	m.formFocus = 0
	m.formInputs[0].Focus()
}

func (m model) loadSambaStatusCmd(dn string) tea.Cmd {
	client := m.client
	uid := ""
	if m.sambaUser != nil {
		uid = m.sambaUser.UID
	}
	return func() tea.Msg {
		if uid != "" {
			u, err := client.GetUser(uid)
			if err != nil {
				return sambaUserReloadMsg{err: err}
			}
			ok, err := client.HasSambaPassword(u.DN)
			return sambaUserReloadMsg{user: *u, present: ok, err: err}
		}
		ok, err := client.HasSambaPassword(dn)
		return sambaStatusMsg{present: ok, err: err}
	}
}

func (m model) reloadSambaUserCmd() tea.Cmd {
	uid := ""
	if m.sambaUser != nil {
		uid = m.sambaUser.UID
	}
	client := m.client
	return func() tea.Msg {
		u, err := client.GetUser(uid)
		if err != nil {
			return sambaUserReloadMsg{err: err}
		}
		ok, err := client.HasSambaPassword(u.DN)
		return sambaUserReloadMsg{user: *u, present: ok, err: err}
	}
}
