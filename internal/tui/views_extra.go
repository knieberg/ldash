package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/knieberg/ldash/internal/config"
	ldifpkg "github.com/knieberg/ldash/internal/ldif"
	ldapclient "github.com/knieberg/ldash/internal/ldap"
)

const (
	ldifStepHub = iota
	ldifStepPath
	ldifStepConfirm
	ldifStepSummary
)

type groupsLoadedFullMsg struct {
	groups []ldapclient.Group
	err    error
}

type ldifPreviewMsg struct {
	count int
	err   error
}

type ldifDoneMsg struct {
	result *ldifpkg.ApplyResult
	count  int
	err    error
}

func (m model) updateGroupsList(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	page := m.userPageSize()
	switch msg.String() {
	case keyUp, "up":
		if m.groupCursor > 0 {
			m.groupCursor--
			m.ensureGroupVisible()
		}
	case keyDown, "down":
		if m.groupCursor < len(m.groupsFull)-1 {
			m.groupCursor++
			m.ensureGroupVisible()
		}
	case keyPgUp:
		m.groupCursor -= page
		if m.groupCursor < 0 {
			m.groupCursor = 0
		}
		m.ensureGroupVisible()
	case keyPgDown:
		m.groupCursor += page
		if m.groupCursor >= len(m.groupsFull) {
			m.groupCursor = max(0, len(m.groupsFull)-1)
		}
		m.ensureGroupVisible()
	case keyRefresh:
		m.busy = true
		m.setStatus(statusLoading, "Refreshing groups...")
		return m, m.ensureConnAnd(m.loadGroupsFullCmd())
	case keySearch:
		m.groupSearching = true
		m.groupSearchInput.SetValue(m.groupFilter)
		m.groupSearchInput.Width = formInputWidth(m.width)
		m.groupSearchInput.Focus()
		return m, textinput.Blink
	case keyCreate:
		m.initGroupCreateForm()
		m.current = viewGroupCreate
		return m, textinput.Blink
	case keyEdit:
		if g := m.selectedGroup(); g != nil {
			m.initGroupEditForm(*g)
			m.current = viewGroupEdit
			return m, textinput.Blink
		}
	case keyDelete:
		if g := m.selectedGroup(); g != nil {
			m.formUID = g.CN
			m.formDN = g.DN
			m.formGID = g.GIDNumber
			m.confirm = false
			m.current = viewGroupDelete
		}
	case keyMembers:
		if g := m.selectedGroup(); g != nil {
			m.formUID = g.CN
			m.formDN = g.DN
			m.memberAttr = g.MemberAttribute
			m.members = g.Members
			m.current = viewGroupMembers
		}
	}
	return m, nil
}

func (m *model) ensureGroupVisible() {
	page := m.userPageSize()
	if m.groupCursor < m.groupScrollOffset {
		m.groupScrollOffset = m.groupCursor
	}
	if m.groupCursor >= m.groupScrollOffset+page {
		m.groupScrollOffset = m.groupCursor - page + 1
	}
}

func (m model) selectedGroup() *ldapclient.Group {
	if m.groupCursor < 0 || m.groupCursor >= len(m.groupsFull) {
		return nil
	}
	return &m.groupsFull[m.groupCursor]
}

func (m model) viewGroupsList() string {
	var b strings.Builder
	if m.groupSearching {
		b.WriteString("Filter: ")
		b.WriteString(m.groupSearchInput.View())
		b.WriteString("\n")
	} else if m.groupFilter != "" {
		b.WriteString(mutedStyle.Render(fmt.Sprintf("Filter: %q", m.groupFilter)))
		b.WriteString("\n")
	}
	if len(m.groupsFull) == 0 {
		if m.groupFilter != "" {
			b.WriteString(mutedStyle.Render("No matches for the current filter."))
			b.WriteString("\n")
			b.WriteString(mutedStyle.Render("Press / search to change filter · Esc back · r refresh"))
			return b.String()
		}
		b.WriteString(mutedStyle.Render("No groups found."))
		b.WriteString("\n")
		b.WriteString(mutedStyle.Render("Press c create · / search · r refresh"))
		return b.String()
	}
	b.WriteString(headerStyle.Render("  CN                 GID#     Members"))
	b.WriteString("\n")
	start := m.groupScrollOffset
	end := start + m.userPageSize()
	if end > len(m.groupsFull) {
		end = len(m.groupsFull)
	}
	for i := start; i < end; i++ {
		g := m.groupsFull[i]
		line := fmt.Sprintf("  %-18s %-8d %d", truncRunes(g.CN, 18), g.GIDNumber, g.MemberCount)
		if i == m.groupCursor {
			b.WriteString(selStyle.Render("▸ " + strings.TrimPrefix(line, "  ")))
		} else {
			b.WriteString(line)
		}
		b.WriteString("\n")
	}
	b.WriteString("\n")
	b.WriteString(mutedStyle.Render(fmt.Sprintf("Showing %d–%d of %d", start+1, end, len(m.groupsFull))))
	return b.String()
}

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
		b.WriteString(fmt.Sprintf("  %s — %s\n", label, help))
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

func (m model) viewLDIF() string {
	switch m.ldifStep {
	case ldifStepHub:
		opts := []string{"Export LDIF", "Import LDIF"}
		var b strings.Builder
		b.WriteString(headerStyle.Render("LDIF backup and restore"))
		b.WriteString("\n\n")
		for i, o := range opts {
			prefix := "  "
			if i == m.ldifCursor {
				prefix = "▸ "
				b.WriteString(selStyle.Render(prefix + o))
			} else {
				b.WriteString(prefix + o)
			}
			b.WriteString("\n")
		}
		b.WriteString("\n")
		b.WriteString(mutedStyle.Render("Export redacts password hashes. Import skips password attributes."))
		return b.String()
	case ldifStepPath:
		var b strings.Builder
		if m.ldifAction == "export" {
			b.WriteString(headerStyle.Render("Export LDIF"))
			b.WriteString("\n\nScope: ")
			scopes := []string{"People", "Groups", "Both"}
			for i, s := range scopes {
				if i == m.ldifScopeIdx {
					b.WriteString(selStyle.Render(s))
				} else {
					b.WriteString(s)
				}
				if i < len(scopes)-1 {
					b.WriteString(" · ")
				}
			}
			b.WriteString("\n\nPath:\n")
		} else {
			b.WriteString(headerStyle.Render("Import LDIF"))
			b.WriteString("\n\nPath:\n")
		}
		b.WriteString(m.ldifPathInput.View())
		return b.String()
	case ldifStepConfirm:
		var b strings.Builder
		b.WriteString(warnStyle.Render("Import LDIF?"))
		b.WriteString("\n\n")
		b.WriteString(fmt.Sprintf("File: %s\n", m.ldifPath))
		b.WriteString(fmt.Sprintf("Entries: %d\n\n", m.ldifPreview))
		if m.confirm {
			b.WriteString(errStyle.Render("Final confirmation: press y to apply changes."))
		} else {
			b.WriteString("Press y to continue, n to cancel.")
		}
		return b.String()
	case ldifStepSummary:
		var b strings.Builder
		b.WriteString(headerStyle.Render("LDIF result"))
		b.WriteString("\n\n")
		if m.ldifResult != nil {
			fmt.Fprintf(&b, "Applied: %d  Failed: %d  Skipped: %d\n", m.ldifResult.Applied, m.ldifResult.Failed, m.ldifResult.Skipped)
			for i, e := range m.ldifResult.Errors {
				if i >= 5 {
					b.WriteString(mutedStyle.Render(fmt.Sprintf("… and %d more", len(m.ldifResult.Errors)-5)))
					break
				}
				b.WriteString("  " + e + "\n")
			}
		} else if m.ldifExportCount > 0 {
			fmt.Fprintf(&b, "Exported %d entries to %s\n", m.ldifExportCount, m.ldifPath)
		} else {
			b.WriteString(mutedStyle.Render("No entries applied or exported."))
			b.WriteString("\n")
			b.WriteString(mutedStyle.Render("Check skipped/failed counts above or try another file/scope."))
		}
		b.WriteString("\n")
		b.WriteString(mutedStyle.Render("Press Esc to return to LDIF menu."))
		return b.String()
	}
	return ""
}

func defaultExportPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, fmt.Sprintf("ldash-export-%s.ldif", time.Now().Format("20060102")))
}

func (m model) ldifScopeName() string {
	switch m.ldifScopeIdx {
	case 0:
		return "people"
	case 1:
		return "groups"
	default:
		return "both"
	}
}

func (m model) loadGroupsFullCmd() tea.Cmd {
	filter := m.groupFilter
	client := m.client
	return func() tea.Msg {
		groups, err := client.ListGroups(filter)
		return groupsLoadedFullMsg{groups: groups, err: err}
	}
}

func (m model) ldifPreviewCmd() tea.Cmd {
	path := m.ldifPath
	return func() tea.Msg {
		n, err := ldifpkg.PreviewImport(path)
		return ldifPreviewMsg{count: n, err: err}
	}
}

func (m model) ldifImportCmd() tea.Cmd {
	path := m.ldifPath
	client := m.client
	return func() tea.Msg {
		res, err := ldifpkg.Import(client, path)
		return ldifDoneMsg{result: res, err: err}
	}
}

func (m model) ldifExportCmd() tea.Cmd {
	path := m.ldifPath
	scope := m.ldifScopeName()
	cfg := m.cfg
	client := m.client
	return func() tea.Msg {
		n, err := ldifpkg.Export(client, cfg, scope, path)
		return ldifDoneMsg{count: n, err: err}
	}
}

func (m model) updateLDIF(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.ldifStep {
	case ldifStepHub:
		switch msg.String() {
		case keyUp, "up":
			if m.ldifCursor > 0 {
				m.ldifCursor--
			}
		case keyDown, "down":
			if m.ldifCursor < 1 {
				m.ldifCursor++
			}
		case keyEnter:
			if m.ldifCursor == 0 {
				m.ldifAction = "export"
			} else {
				m.ldifAction = "import"
			}
			m.ldifStep = ldifStepPath
			m.ldifPath = defaultExportPath()
			m.ldifPathInput.SetValue(m.ldifPath)
			m.ldifPathInput.Width = formInputWidth(m.width)
			m.ldifPathInput.Focus()
			return m, textinput.Blink
		}
	case ldifStepPath:
		return m.updateLDIFPath(msg)
	}
	return m, nil
}

func (m model) updateLDIFPath(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case keyBack:
		m.ldifStep = ldifStepHub
		m.ldifPathInput.Blur()
		return m, nil
	case keyEnter:
		m.ldifPath = strings.TrimSpace(m.ldifPathInput.Value())
		if m.ldifPath == "" {
			m.setStatus(statusError, "path is required")
			return m, nil
		}
		m.ldifPathInput.Blur()
		if m.ldifAction == "import" {
			m.busy = true
			m.setStatus(statusLoading, "Reading LDIF...")
			return m, m.ldifPreviewCmd()
		}
		m.busy = true
		m.setStatus(statusLoading, "Exporting LDIF...")
		return m, m.ensureConnAnd(m.ldifExportCmd())
	case "left", "h":
		if m.ldifAction == "export" && m.ldifScopeIdx > 0 {
			m.ldifScopeIdx--
		}
	case "right", "l":
		if m.ldifAction == "export" && m.ldifScopeIdx < 2 {
			m.ldifScopeIdx++
		}
	}
	var cmd tea.Cmd
	m.ldifPathInput, cmd = m.ldifPathInput.Update(msg)
	return m, cmd
}

func (m model) updateLDIFConfirm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case keyBack, "n":
		m.ldifStep = ldifStepPath
		m.confirm = false
		return m, nil
	case "y":
		if !m.confirm {
			m.confirm = true
			m.setStatus(statusWarn, "Press y again to confirm import")
			return m, nil
		}
		m.busy = true
		m.setStatus(statusLoading, "Importing LDIF...")
		return m, m.ensureConnAnd(m.ldifImportCmd())
	}
	return m, nil
}

func (m model) updateGroupSearch(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case keyBack:
		m.groupSearching = false
		m.groupSearchInput.Blur()
		return m, nil
	case keyEnter:
		m.groupFilter = m.groupSearchInput.Value()
		m.groupSearching = false
		m.groupSearchInput.Blur()
		m.busy = true
		return m, m.ensureConnAnd(m.loadGroupsFullCmd())
	}
	var cmd tea.Cmd
	m.groupSearchInput, cmd = m.groupSearchInput.Update(msg)
	return m, cmd
}

func (m model) updateGroupMembers(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case keyCreate:
		m.initMemberAddForm()
		m.current = viewGroupMemberAdd
		return m, textinput.Blink
	case keyDelete:
		if m.memberCursor >= 0 && m.memberCursor < len(m.members) {
			val := m.members[m.memberCursor]
			m.busy = true
			return m, m.ensureConnAnd(m.removeMemberCmd(val))
		}
	case keyUp, "up":
		if m.memberCursor > 0 {
			m.memberCursor--
		}
	case keyDown, "down":
		if m.memberCursor < len(m.members)-1 {
			m.memberCursor++
		}
	}
	return m, nil
}

func (m *model) initMemberAddForm() {
	w := formInputWidth(m.width)
	m.formInputs = []textinput.Model{newInput("uid to add", false, w)}
	m.formFocus = 0
	m.formInputs[0].Focus()
}

func (m model) updateGroupMemberAdd(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case keyBack:
		m.current = viewGroupMembers
		return m, nil
	case keyEnter:
		uid := strings.TrimSpace(m.formInputs[0].Value())
		if uid == "" {
			m.setStatus(statusError, "uid is required")
			return m, nil
		}
		m.busy = true
		return m, m.ensureConnAnd(m.addMemberCmd(uid))
	}
	var cmd tea.Cmd
	m.formInputs[m.formFocus], cmd = m.formInputs[m.formFocus].Update(msg)
	return m, cmd
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

type sambaStatusMsg struct {
	present bool
	err     error
}

func (m model) viewGroupMembers() string {
	var b strings.Builder
	b.WriteString(headerStyle.Render("Members: " + m.formUID))
	b.WriteString("\n\n")
	if len(m.members) == 0 {
		b.WriteString(mutedStyle.Render("No members in this group."))
		b.WriteString("\n")
		b.WriteString(mutedStyle.Render("Press c add · Esc back"))
		return b.String()
	}
	for i, mem := range m.members {
		line := "  " + mem
		if i == m.memberCursor {
			b.WriteString(selStyle.Render("▸ " + mem))
		} else {
			b.WriteString(line)
		}
		b.WriteString("\n")
	}
	b.WriteString("\n")
	b.WriteString(mutedStyle.Render(fmt.Sprintf("%d member(s) via %s", len(m.members), m.memberAttr)))
	return b.String()
}

func (m model) viewMemberAddForm() string {
	var b strings.Builder
	b.WriteString(headerStyle.Render("Add member to " + m.formUID))
	b.WriteString("\n\n")
	label := "Login name (uid)"
	if m.memberAttr == "member" {
		label = "Login name (uid) — DN resolved automatically"
	}
	prefix := "▸ "
	fmt.Fprintf(&b, "%s%-28s %s\n", prefix, label+":", m.formInputs[0].View())
	return b.String()
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

func (m model) addMemberCmd(uid string) tea.Cmd {
	dn := m.formDN
	attr := m.memberAttr
	client := m.client
	cfg := m.cfg
	return func() tea.Msg {
		val := uid
		if attr == "member" {
			u, err := client.GetUser(uid)
			if err != nil {
				return actionDoneMsg{err: err}
			}
			val = u.DN
		}
		if err := client.AddGroupMember(dn, attr, val); err != nil {
			return actionDoneMsg{err: err}
		}
		_ = cfg
		return actionDoneMsg{message: "Member added", reloadView: viewGroupMembers}
	}
}

func (m model) removeMemberCmd(val string) tea.Cmd {
	dn := m.formDN
	attr := m.memberAttr
	client := m.client
	return func() tea.Msg {
		if err := client.RemoveGroupMember(dn, attr, val); err != nil {
			return actionDoneMsg{err: err}
		}
		return actionDoneMsg{message: "Member removed", reloadView: viewGroupMembers}
	}
}
