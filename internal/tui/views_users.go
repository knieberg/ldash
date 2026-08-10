package tui

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/knieberg/ldash/internal/config"
	ldapclient "github.com/knieberg/ldash/internal/ldap"
)

func (m model) updateUsersList(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	page := m.userPageSize()
	switch msg.String() {
	case keyUp, "up":
		if m.userCursor > 0 {
			m.userCursor--
			m.ensureUserVisible()
		}
	case keyDown, "down":
		if m.userCursor < len(m.users)-1 {
			m.userCursor++
			m.ensureUserVisible()
		}
	case keyPgUp:
		m.userCursor -= page
		if m.userCursor < 0 {
			m.userCursor = 0
		}
		m.ensureUserVisible()
	case keyPgDown:
		m.userCursor += page
		if m.userCursor >= len(m.users) {
			m.userCursor = max(0, len(m.users)-1)
		}
		m.ensureUserVisible()
	case keyHome, keyTop:
		m.userCursor = 0
		m.ensureUserVisible()
	case keyEnd, keyBottom:
		if len(m.users) > 0 {
			m.userCursor = len(m.users) - 1
		}
		m.ensureUserVisible()
	case keyRefresh:
		m.busy = true
		m.setStatus(statusLoading, "Refreshing users...")
		return m, m.ensureConnAnd(m.loadUsersCmd())
	case keySearch:
		m.searching = true
		m.searchInput.SetValue(m.userFilter)
		m.searchInput.Width = formInputWidth(m.width)
		m.searchInput.Focus()
		return m, textinput.Blink
	case keyCreate:
		m.initCreateForm()
		m.current = viewUserCreate
		return m, textinput.Blink
	case keyEdit:
		if u := m.selectedUser(); u != nil {
			m.initEditForm(*u)
			m.current = viewUserEdit
			return m, textinput.Blink
		}
	case keyDelete:
		if u := m.selectedUser(); u != nil {
			m.formUID = u.UID
			m.formDN = u.DN
			m.confirm = false
			m.current = viewUserDelete
			m.busy = true
			m.setStatus(statusLoading, "Loading group refs...")
			return m, m.ensureConnAnd(m.loadGroupsCmd(u.UID, u.DN))
		}
	case keyPassword:
		if u := m.selectedUser(); u != nil {
			m.initPasswordForm(*u)
			m.current = viewUserPassword
			return m, textinput.Blink
		}
	case keyMail:
		if u := m.selectedUser(); u != nil {
			m.initMailForm(*u)
			m.current = viewUserMail
			return m, textinput.Blink
		}
	case keySamba:
		if u := m.selectedUser(); u != nil {
			m.sambaUser = u
			m.current = viewSambaUser
			m.busy = true
			m.setStatus(statusLoading, "Loading Samba status...")
			return m, m.ensureConnAnd(m.loadSambaStatusCmd(u.DN))
		}
	}
	return m, nil
}

func (m model) updateUserSearch(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case keyBack:
		m.searching = false
		m.searchInput.Blur()
		return m, nil
	case "ctrl+u":
		m.searchInput.SetValue("")
		return m, nil
	case keyEnter:
		m.userFilter = m.searchInput.Value()
		m.searching = false
		m.searchInput.Blur()
		m.busy = true
		m.setStatus(statusLoading, "Filtering users...")
		return m, m.ensureConnAnd(m.loadUsersCmd())
	case "ctrl+c":
		return m.quit()
	}
	var cmd tea.Cmd
	m.searchInput, cmd = m.searchInput.Update(msg)
	return m, cmd
}

func (m model) selectedUser() *ldapclient.User {
	if m.userCursor < 0 || m.userCursor >= len(m.users) {
		return nil
	}
	return &m.users[m.userCursor]
}

func (m *model) ensureUserVisible() {
	page := m.userPageSize()
	if m.userCursor < m.scrollOffset {
		m.scrollOffset = m.userCursor
	}
	if m.userCursor >= m.scrollOffset+page {
		m.scrollOffset = m.userCursor - page + 1
	}
	if m.scrollOffset < 0 {
		m.scrollOffset = 0
	}
}

func (m model) columnWidths() (uidW, cnW, numW, mailW int) {
	aw := appWidth(m.width) - 4
	if aw < 40 {
		aw = 40
	}
	uidW = 14
	numW = 8
	cnW = aw / 3
	if cnW < 12 {
		cnW = 12
	}
	if cnW > 28 {
		cnW = 28
	}
	mailW = aw - uidW - cnW - numW - 3
	if mailW < 8 {
		mailW = 0
	}
	if mailW > 40 {
		mailW = 40
	}
	return uidW, cnW, numW, mailW
}

func (m model) viewUsers() string {
	var b strings.Builder
	if m.searching {
		b.WriteString("Filter: ")
		b.WriteString(m.searchInput.View())
		b.WriteString("\n")
	} else if m.userFilter != "" {
		b.WriteString(mutedStyle.Render(fmt.Sprintf("Filter: %q", m.userFilter)))
		b.WriteString("\n")
	}

	if len(m.users) == 0 {
		if m.userFilter != "" {
			b.WriteString(mutedStyle.Render("No matches for the current filter."))
			b.WriteString("\n")
			b.WriteString(mutedStyle.Render("Press / search to change filter · Esc back · r refresh"))
			return b.String()
		}
		b.WriteString(mutedStyle.Render("No users found."))
		b.WriteString("\n")
		b.WriteString(mutedStyle.Render("Press r refresh · c create · / search"))
		b.WriteString("\n")
		b.WriteString(warnStyle.Render("Warn: Check search.list_users_filter — does it match your LDAP object classes?"))
		return b.String()
	}

	uidW, cnW, numW, mailW := m.columnWidths()
	headers := []string{"UID", "CN", "UID#"}
	widths := []int{uidW, cnW, numW}
	if mailW > 0 {
		headers = append(headers, "MAIL")
		widths = append(widths, mailW)
	}
	b.WriteString(headerStyle.Render("  "+padColumns(headers, widths)))
	b.WriteString("\n")

	page := m.userPageSize()
	start := m.scrollOffset
	if start < 0 {
		start = 0
	}
	end := start + page
	if end > len(m.users) {
		end = len(m.users)
	}
	for i := start; i < end; i++ {
		u := m.users[i]
		cols := []string{u.UID, u.CN, strconv.Itoa(u.UIDNumber)}
		ws := []int{uidW, cnW, numW}
		if mailW > 0 {
			cols = append(cols, u.Mail)
			ws = append(ws, mailW)
		}
		line := padColumns(cols, ws)
		if i == m.userCursor {
			b.WriteString(selStyle.Render("▸ " + line))
		} else {
			b.WriteString("  " + line)
		}
		b.WriteString("\n")
	}
	b.WriteString("\n")
	b.WriteString(mutedStyle.Render(fmt.Sprintf("Showing %d–%d of %d", start+1, end, len(m.users))))
	return b.String()
}

func (m model) viewSimpleForm() string {
	var b strings.Builder
	titles := map[viewID]string{
		viewUserPassword: "Reset password for " + m.formUID,
		viewUserMail:     "Mail for " + m.formUID,
	}
	b.WriteString(headerStyle.Render(titles[m.current]))
	b.WriteString("\n\n")
	labels := m.formLabels()
	for i, in := range m.formInputs {
		prefix := "  "
		if i == m.formFocus {
			prefix = "▸ "
		}
		fmt.Fprintf(&b, "%s%-14s %s\n", prefix, labels[i]+":", in.View())
	}
	if m.current == viewUserPassword && m.integ.SelfServiceURL != "" {
		b.WriteString("\n")
		b.WriteString(mutedStyle.Render("Users can also change passwords at: " + m.integ.SelfServiceURL))
	}
	return b.String()
}

func (m model) viewDelete() string {
	var b strings.Builder
	if m.current == viewGroupDelete {
		b.WriteString(warnStyle.Render(fmt.Sprintf("Delete group %q?", m.formUID)))
		b.WriteString("\n")
		b.WriteString(mutedStyle.Render(m.formDN))
		b.WriteString("\n\n")
		b.WriteString(mutedStyle.Render("Deletion is blocked if users still have this gidNumber as primary group."))
	} else {
		b.WriteString(warnStyle.Render(fmt.Sprintf("Delete user %q?", m.formUID)))
		b.WriteString("\n")
		b.WriteString(mutedStyle.Render(m.formDN))
		b.WriteString("\n\n")
		if len(m.groups) > 0 {
			b.WriteString("Referenced by groups:\n")
			for _, g := range m.groups {
				fmt.Fprintf(&b, "  - %s (%s)\n", g.CN, g.Attr)
			}
			b.WriteString("\n")
		} else {
			b.WriteString(mutedStyle.Render("No group memberships found (or still loading)."))
			b.WriteString("\n\n")
		}
	}
	if m.confirm {
		b.WriteString(errStyle.Render("Final confirmation: press y to permanently delete."))
	} else {
		b.WriteString("Press y to continue, n to cancel.")
	}
	return b.String()
}

func (m model) formLabels() []string {
	switch m.current {
	case viewUserCreate:
		return []string{"uid", "cn", "sn", "givenName", "mail", "password"}
	case viewUserEdit:
		return []string{"cn", "sn", "givenName", "mail", "gecos", "loginShell", "homeDirectory"}
	case viewUserPassword:
		return []string{"new password"}
	case viewUserMail:
		return []string{"mail"}
	default:
		return nil
	}
}

func (m *model) initEditForm(u ldapclient.User) {
	m.formUID = u.UID
	m.formDN = u.DN
	tpl, err := config.LoadUserTemplate(m.cfg)
	if err != nil {
		m.formSpecs = nil
		return
	}
	extra := map[string]string{}
	if m.client.Connected() {
		if attrs := templateCustomAttrs(tpl); len(attrs) > 0 {
			if got, err := m.client.GetEntryAttrs(u.DN, attrs); err == nil {
				extra = got
			}
		}
	}
	m.initTemplateEditForm(tpl, u, extra)
}

func (m *model) initPasswordForm(u ldapclient.User) {
	w := formInputWidth(m.width)
	m.formUID = u.UID
	m.formDN = u.DN
	m.formInputs = []textinput.Model{newInput("new password", true, w)}
	m.formFocus = 0
	m.formInputs[0].Focus()
}

func (m *model) initMailForm(u ldapclient.User) {
	w := formInputWidth(m.width)
	m.formUID = u.UID
	m.formDN = u.DN
	m.formInputs = []textinput.Model{newInput("user@example.com (empty to clear)", false, w)}
	m.formInputs[0].SetValue(u.Mail)
	m.formFocus = 0
	m.formInputs[0].Focus()
}

func (m model) loadUsersCmd() tea.Cmd {
	filter := m.userFilter
	client := m.client
	return func() tea.Msg {
		users, err := client.ListUsers(filter)
		return usersLoadedMsg{users: users, err: err}
	}
}

func (m model) loadGroupsCmd(uid, dn string) tea.Cmd {
	client := m.client
	return func() tea.Msg {
		groups, err := client.GroupsForUser(uid, dn)
		return groupsLoadedMsg{groups: groups, err: err}
	}
}

func (m model) passwordCmd() tea.Cmd {
	pass := m.formInputs[0].Value()
	dn := m.formDN
	client := m.client
	return func() tea.Msg {
		if pass == "" {
			return actionDoneMsg{err: fmt.Errorf("password must not be empty")}
		}
		if err := client.SetPassword(dn, pass); err != nil {
			return actionDoneMsg{err: err}
		}
		return actionDoneMsg{message: "Password updated", reloadView: viewUsers}
	}
}

func (m model) mailCmd() tea.Cmd {
	mail := strings.TrimSpace(m.formInputs[0].Value())
	dn := m.formDN
	client := m.client
	return func() tea.Msg {
		if err := client.SetMail(dn, mail); err != nil {
			return actionDoneMsg{err: err}
		}
		msg := "Mail updated"
		if mail == "" {
			msg = "Mail attribute removed"
		}
		return actionDoneMsg{message: msg, reloadView: viewUsers}
	}
}
