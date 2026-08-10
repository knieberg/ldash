package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	ldapclient "github.com/knieberg/ldash/internal/ldap"
)

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
		if err := m.initGroupCreateForm(); err != nil {
			m.setStatus(statusError, err.Error())
			return m, nil
		}
		m.current = viewGroupCreate
		return m, textinput.Blink
	case keyEdit:
		if g := m.selectedGroup(); g != nil {
			if err := m.initGroupEditForm(*g); err != nil {
				m.setStatus(statusWarn, "Group template not loaded; editing description only")
			}
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

func (m model) loadGroupsFullCmd() tea.Cmd {
	filter := m.groupFilter
	client := m.client
	return func() tea.Msg {
		groups, err := client.ListGroups(filter)
		return groupsLoadedFullMsg{groups: groups, err: err}
	}
}

func (m model) updateGroupSearch(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
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

func (m model) addMemberCmd(uid string) tea.Cmd {
	dn := m.formDN
	attr := m.memberAttr
	client := m.client
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

func (m model) loadGroupMembersCmd() tea.Cmd {
	cn := m.formUID
	client := m.client
	return func() tea.Msg {
		g, err := client.GetGroup(cn)
		if err != nil {
			return actionDoneMsg{err: err}
		}
		return groupMembersLoadedMsg{members: g.Members, memberAttr: g.MemberAttribute}
	}
}
