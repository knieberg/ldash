package tui

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/knieberg/ldash/internal/config"
	ldapclient "github.com/knieberg/ldash/internal/ldap"
)

type viewID int

const (
	viewMenu viewID = iota
	viewDashboard
	viewUsers
	viewGroups
	viewSamba
	viewIntegration
	viewSettings
	viewUserCreate
	viewUserEdit
	viewUserDelete
	viewUserPassword
	viewUserMail
)

type pingMsg struct {
	result ldapclient.PingResult
	err    error
}

type usersLoadedMsg struct {
	users []ldapclient.User
	err   error
}

type actionDoneMsg struct {
	message string
	err     error
	reload  bool
}

type groupsLoadedMsg struct {
	groups []ldapclient.GroupRef
	err    error
}

type menuItem struct {
	id      viewID
	label   string
	hint    string
	enabled bool
}

type model struct {
	cfg      *config.Config
	integ    *config.Integration
	password string
	client   *ldapclient.Client

	current  viewID
	width    int
	height   int
	status   string
	statusK  statusKind
	busy     bool
	showHelp bool
	quitting bool

	menuCursor int
	menuItems  []menuItem

	pingResult *ldapclient.PingResult

	users        []ldapclient.User
	userCursor   int
	userFilter   string
	searching    bool
	searchInput  textinput.Model
	scrollOffset int

	formInputs []textinput.Model
	formFocus  int
	formUID    string
	formDN     string
	confirm    bool
	groups     []ldapclient.GroupRef
}

func New(cfg *config.Config, password string, integ *config.Integration) tea.Model {
	if integ == nil {
		integ = &config.Integration{}
	}
	si := textinput.New()
	si.Placeholder = "filter users..."
	si.CharLimit = 64
	si.Width = 40

	return model{
		cfg:         cfg,
		integ:       integ,
		password:    password,
		client:      ldapclient.NewClient(cfg),
		current:     viewMenu,
		status:      "Select a view and press Enter",
		statusK:     statusInfo,
		searchInput: si,
		menuItems: []menuItem{
			{viewDashboard, "Dashboard", "Connection health", true},
			{viewUsers, "Users", "List, create, edit, delete", true},
			{viewGroups, "Groups", "Coming in v0.2", false},
			{viewSamba, "Samba", "Coming soon", false},
			{viewIntegration, "Integration Guide", "Runtime snippets from local config", true},
			{viewSettings, "Settings", "Shows loaded connection profile", true},
		},
	}
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m *model) setStatus(kind statusKind, msg string) {
	m.statusK = kind
	m.status = msg
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.resizeInputs()
		m.ensureUserVisible()
		return m, nil

	case pingMsg:
		m.busy = false
		if msg.err != nil {
			m.setStatus(statusError, msg.err.Error())
			m.pingResult = nil
		} else {
			m.pingResult = &msg.result
			if msg.result.OK {
				m.setStatus(statusOK, fmt.Sprintf("Connected in %s", msg.result.Duration.Round(1e6)))
			} else {
				m.setStatus(statusError, msg.result.Message)
			}
		}
		return m, nil

	case usersLoadedMsg:
		m.busy = false
		if msg.err != nil {
			m.setStatus(statusError, msg.err.Error())
			m.users = nil
		} else {
			m.users = msg.users
			if m.userCursor >= len(m.users) {
				m.userCursor = 0
			}
			m.ensureUserVisible()
			m.setStatus(statusInfo, fmt.Sprintf("%d user(s)", len(m.users)))
		}
		return m, nil

	case groupsLoadedMsg:
		m.busy = false
		if msg.err != nil {
			m.setStatus(statusError, msg.err.Error())
		} else {
			m.groups = msg.groups
		}
		return m, nil

	case actionDoneMsg:
		m.busy = false
		if msg.err != nil {
			m.setStatus(statusError, msg.err.Error())
		} else {
			m.setStatus(statusOK, msg.message)
			if msg.reload {
				m.current = viewUsers
				m.confirm = false
				m.busy = true
				m.setStatus(statusLoading, "Refreshing users...")
				return m, m.loadUsersCmd()
			}
		}
		return m, nil

	case tea.KeyMsg:
		if m.showHelp {
			switch msg.String() {
			case keyHelp, keyBack:
				m.showHelp = false
				return m, nil
			case "ctrl+c":
				return m.quit()
			}
			return m, nil
		}
		if m.searching && m.current == viewUsers {
			return m.updateUserSearch(msg)
		}
		if m.isFormView() {
			return m.updateForm(msg)
		}
		if m.current == viewUserDelete {
			return m.updateDelete(msg)
		}
		return m.updateNav(msg)
	}
	return m, nil
}

func (m model) isFormView() bool {
	return m.current == viewUserCreate || m.current == viewUserEdit ||
		m.current == viewUserPassword || m.current == viewUserMail
}

func (m model) isUserChild() bool {
	return m.isFormView() || m.current == viewUserDelete
}

func (m model) quit() (tea.Model, tea.Cmd) {
	m.quitting = true
	m.closeClient()
	return m, tea.Quit
}

func (m model) goBack() (tea.Model, tea.Cmd) {
	if m.showHelp {
		m.showHelp = false
		return m, nil
	}
	if m.searching && m.current == viewUsers {
		m.searching = false
		m.searchInput.Blur()
		return m, nil
	}
	if m.isUserChild() {
		m.current = viewUsers
		m.confirm = false
		m.setStatus(statusInfo, "Cancelled")
		return m, nil
	}
	if m.current != viewMenu {
		m.current = viewMenu
		m.setStatus(statusInfo, "Main menu")
		return m, nil
	}
	m.setStatus(statusInfo, "Press q to quit")
	return m, nil
}

func (m model) updateNav(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		return m.quit()
	case keyHelp:
		m.showHelp = true
		return m, nil
	case keyQuit:
		if m.current == viewMenu {
			return m.quit()
		}
		// q is not back on other views
		return m, nil
	case keyBack:
		return m.goBack()
	}

	switch m.current {
	case viewMenu:
		return m.updateMenu(msg)
	case viewDashboard:
		if msg.String() == keyRefresh {
			m.busy = true
			m.setStatus(statusLoading, "Testing connection...")
			return m, m.pingCmd()
		}
	case viewUsers:
		return m.updateUsersList(msg)
	}
	return m, nil
}

func (m model) updateMenu(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	n := len(m.menuItems)
	if n == 0 {
		return m, nil
	}
	switch msg.String() {
	case keyUp, "up":
		m.menuCursor = (m.menuCursor - 1 + n) % n
	case keyDown, "down":
		m.menuCursor = (m.menuCursor + 1) % n
	case keyEnter:
		return m.openMenuItem(m.menuCursor)
	default:
		if len(msg.String()) == 1 {
			if d, err := strconv.Atoi(msg.String()); err == nil && d >= 1 && d <= n {
				m.menuCursor = d - 1
				return m.openMenuItem(m.menuCursor)
			}
		}
	}
	return m, nil
}

func (m model) openMenuItem(idx int) (tea.Model, tea.Cmd) {
	if idx < 0 || idx >= len(m.menuItems) {
		return m, nil
	}
	item := m.menuItems[idx]
	if !item.enabled {
		m.setStatus(statusWarn, item.label+" available in a later release")
		return m, nil
	}
	m.current = item.id
	m.setStatus(statusInfo, item.label)
	if item.id == viewUsers {
		m.busy = true
		m.setStatus(statusLoading, "Loading users...")
		return m, m.ensureConnAnd(m.loadUsersCmd())
	}
	return m, nil
}

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

func (m model) updateForm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case keyBack:
		return m.goBack()
	case keyHelp:
		m.showHelp = true
		return m, nil
	case "ctrl+c":
		return m.quit()
	case "tab", "down":
		m.formFocus = (m.formFocus + 1) % len(m.formInputs)
		return m, m.focusForm()
	case "shift+tab", "up":
		m.formFocus = (m.formFocus - 1 + len(m.formInputs)) % len(m.formInputs)
		return m, m.focusForm()
	case keyEnter:
		m.busy = true
		m.setStatus(statusLoading, "Submitting...")
		return m.submitForm()
	}
	var cmd tea.Cmd
	m.formInputs[m.formFocus], cmd = m.formInputs[m.formFocus].Update(msg)
	return m, cmd
}

func (m model) updateDelete(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case keyBack, "n":
		return m.goBack()
	case keyHelp:
		m.showHelp = true
		return m, nil
	case "ctrl+c":
		return m.quit()
	case "y":
		if !m.confirm {
			m.confirm = true
			m.setStatus(statusWarn, "Press y again to confirm delete")
			return m, nil
		}
		m.busy = true
		m.setStatus(statusLoading, "Deleting user...")
		return m, m.ensureConnAnd(m.deleteUserCmd())
	}
	return m, nil
}

func (m model) focusForm() tea.Cmd {
	for i := range m.formInputs {
		if i == m.formFocus {
			m.formInputs[i].Focus()
		} else {
			m.formInputs[i].Blur()
		}
	}
	return textinput.Blink
}

func (m model) selectedUser() *ldapclient.User {
	if m.userCursor < 0 || m.userCursor >= len(m.users) {
		return nil
	}
	return &m.users[m.userCursor]
}

func (m *model) resizeInputs() {
	w := formInputWidth(m.width)
	m.searchInput.Width = w
	for i := range m.formInputs {
		m.formInputs[i].Width = w
	}
}

func (m model) contentHeight() int {
	// Approximate shell chrome: title + crumb + rule + footer bar.
	chrome := 5
	h := m.height - chrome
	if h < 5 {
		h = 5
	}
	return h
}

func (m model) userPageSize() int {
	h := m.contentHeight() - 6 // panel padding, filter, header, meta
	if h < 3 {
		h = 3
	}
	return h
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

func (m model) View() string {
	if m.quitting {
		return ""
	}
	w := m.width
	if w <= 0 {
		w = 80
	}
	h := m.height
	if h <= 0 {
		h = 24
	}

	header := renderHeader("ldash — LDAP Admin Shell", m.breadcrumbStyled(), w)
	body := m.viewBody()
	if m.showHelp {
		body = m.overlayHelp(body, w)
	}
	footer := renderFooterBar(m.footerKeys(), statusLine(m.statusK, m.status), w)
	return renderShell(header, body, footer, w, h)
}

func (m model) viewBody() string {
	aw := appWidth(m.width)
	var content string
	switch m.current {
	case viewMenu:
		content = m.viewMenu()
	case viewDashboard:
		content = m.viewDashboard()
	case viewUsers:
		content = m.viewUsers()
	case viewGroups:
		content = mutedStyle.Render("Group membership management is planned for v0.2.") +
			"\n" + mutedStyle.Render("Press Esc to return to the main menu.")
	case viewSamba:
		content = mutedStyle.Render("Samba status view is planned for a later release.") +
			"\n" + mutedStyle.Render("Press Esc to return to the main menu.")
	case viewIntegration:
		content = m.viewIntegration()
	case viewSettings:
		content = m.viewSettings()
	case viewUserCreate, viewUserEdit, viewUserPassword, viewUserMail:
		content = m.viewForm()
	case viewUserDelete:
		content = m.viewDelete()
	default:
		content = ""
	}
	return panel(content, aw)
}

func (m model) overlayHelp(body string, width int) string {
	help := m.helpContent()
	box := helpBox(help, width)
	// Place help over the upper content area.
	return lipgloss.Place(width, lipgloss.Height(body), lipgloss.Center, lipgloss.Center, box,
		lipgloss.WithWhitespaceChars(" "),
		lipgloss.WithWhitespaceForeground(colorMuted),
	)
}

func (m model) helpContent() string {
	var b strings.Builder
	b.WriteString(headerStyle.Render("Help"))
	b.WriteString("\n\n")
	b.WriteString(headerStyle.Render("Navigation"))
	b.WriteString("\n")
	b.WriteString("  Esc     one level back\n")
	b.WriteString("  q       quit (main menu only)\n")
	b.WriteString("  Ctrl+C  quit anytime\n")
	b.WriteString("  ?       toggle this help\n")
	b.WriteString("  1-6     open main menu item\n")
	b.WriteString("\n")
	b.WriteString(headerStyle.Render("This view"))
	b.WriteString("\n")
	for _, line := range strings.Split(m.helpViewKeys(), "\n") {
		b.WriteString("  " + line + "\n")
	}
	b.WriteString("\n")
	b.WriteString(mutedStyle.Render("Press Esc or ? to close"))
	return b.String()
}

func (m model) helpViewKeys() string {
	switch m.current {
	case viewMenu:
		return "j/k move · Enter open · disabled items stay on menu"
	case viewDashboard:
		return "r test LDAP connection"
	case viewUsers:
		if m.searching {
			return "Enter apply · Esc cancel · Ctrl+U clear filter text"
		}
		return "j/k move · PgUp/PgDn page · g/G or Home/End · / search · c e d p m · r refresh"
	case viewUserCreate, viewUserEdit, viewUserPassword, viewUserMail:
		return "Tab next field · Enter submit · Esc cancel"
	case viewUserDelete:
		return "y confirm (twice) · n or Esc cancel"
	default:
		return "Esc back"
	}
}

func (m model) breadcrumbStyled() string {
	parts := m.breadcrumbParts()
	if len(parts) == 0 {
		return ""
	}
	var out []string
	for i, p := range parts {
		if i == len(parts)-1 {
			out = append(out, crumbActiveStyle.Render(p))
		} else {
			out = append(out, crumbStyle.Render(p))
		}
	}
	return strings.Join(out, crumbStyle.Render(" › "))
}

func (m model) breadcrumbParts() []string {
	switch m.current {
	case viewMenu:
		return []string{"Main Menu"}
	case viewDashboard:
		return []string{"Main Menu", "Dashboard"}
	case viewUsers:
		return []string{"Main Menu", "Users"}
	case viewUserCreate:
		return []string{"Main Menu", "Users", "Create"}
	case viewUserEdit:
		return []string{"Main Menu", "Users", "Edit"}
	case viewUserDelete:
		return []string{"Main Menu", "Users", "Delete"}
	case viewUserPassword:
		return []string{"Main Menu", "Users", "Password"}
	case viewUserMail:
		return []string{"Main Menu", "Users", "Mail"}
	case viewGroups:
		return []string{"Main Menu", "Groups"}
	case viewSamba:
		return []string{"Main Menu", "Samba"}
	case viewIntegration:
		return []string{"Main Menu", "Integration Guide"}
	case viewSettings:
		return []string{"Main Menu", "Settings"}
	default:
		return nil
	}
}

func (m model) footerKeys() string {
	base := ""
	switch m.current {
	case viewMenu:
		base = "j/k · 1-6 · Enter · ? · q quit"
	case viewDashboard:
		base = "r test · Esc back · ?"
	case viewUsers:
		if m.searching {
			base = "Enter apply · Esc cancel · Ctrl+U clear · ?"
		} else if m.userFilter != "" {
			base = "j/k · PgUp/PgDn · / search · c e d p m · r · Esc back · ? (filter on)"
		} else {
			base = "j/k · PgUp/PgDn · / · c e d p m · r · Esc back · ?"
		}
	case viewUserCreate, viewUserEdit, viewUserPassword, viewUserMail:
		base = "Tab · Enter submit · Esc cancel · ?"
	case viewUserDelete:
		base = "y confirm · n/Esc cancel · ?"
	default:
		base = "Esc back · ?"
	}
	if m.busy {
		return base
	}
	return base
}

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
		if !item.enabled {
			hint = item.hint
		}
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
	b.WriteString(mutedStyle.Render("Disabled items show a warning and stay on this menu."))
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

func (m model) columnWidths() (uidW, cnW, numW, mailW int) {
	aw := appWidth(m.width) - 4 // cursor prefix
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
		b.WriteString(mutedStyle.Render("No users found."))
		b.WriteString("\n")
		b.WriteString(mutedStyle.Render("Press r to refresh · c to create · / to change filter."))
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
	return boxStyle.Width(inner).Render(strings.Join([]string{
		"Config profile: single file (MVP)",
		fmt.Sprintf("Server: %s", m.cfg.Server.URL),
		fmt.Sprintf("TLS mode: %s", m.cfg.Server.TLSMode),
		fmt.Sprintf("Bind DN: %s", m.cfg.BindDN),
		fmt.Sprintf("Samba domain SID set: %v", m.cfg.Samba.DomainSID != ""),
		fmt.Sprintf("Templates dir: %s", m.cfg.TemplatesDir),
	}, "\n"))
}

func (m model) viewForm() string {
	var b strings.Builder
	titles := map[viewID]string{
		viewUserCreate:   "Create user",
		viewUserEdit:     "Edit user " + m.formUID,
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

func newInput(placeholder string, password bool, width int) textinput.Model {
	ti := textinput.New()
	ti.Placeholder = placeholder
	ti.CharLimit = 128
	if width < 20 {
		width = 40
	}
	ti.Width = width
	if password {
		ti.EchoMode = textinput.EchoPassword
		ti.EchoCharacter = '•'
	}
	return ti
}

func (m *model) initCreateForm() {
	w := formInputWidth(m.width)
	m.formInputs = []textinput.Model{
		newInput("alice", false, w),
		newInput("Alice Example", false, w),
		newInput("Example", false, w),
		newInput("Alice", false, w),
		newInput("alice@example.com", false, w),
		newInput("initial password", true, w),
	}
	m.formFocus = 0
	m.formInputs[0].Focus()
}

func (m *model) initEditForm(u ldapclient.User) {
	w := formInputWidth(m.width)
	m.formUID = u.UID
	m.formDN = u.DN
	vals := []string{u.CN, u.SN, u.GivenName, u.Mail, u.Gecos, u.LoginShell, u.HomeDirectory}
	m.formInputs = make([]textinput.Model, len(vals))
	for i, v := range vals {
		m.formInputs[i] = newInput("", false, w)
		m.formInputs[i].SetValue(v)
	}
	m.formFocus = 0
	m.formInputs[0].Focus()
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

func (m model) submitForm() (tea.Model, tea.Cmd) {
	switch m.current {
	case viewUserCreate:
		return m, m.ensureConnAnd(m.createUserCmd())
	case viewUserEdit:
		return m, m.ensureConnAnd(m.updateUserCmd())
	case viewUserPassword:
		return m, m.ensureConnAnd(m.passwordCmd())
	case viewUserMail:
		return m, m.ensureConnAnd(m.mailCmd())
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

func (m model) ensureConnAnd(next tea.Cmd) tea.Cmd {
	return func() tea.Msg {
		if !m.client.Connected() {
			if err := m.client.Connect(m.password); err != nil {
				return actionDoneMsg{err: err}
			}
		}
		if next == nil {
			return nil
		}
		return next()
	}
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

func (m model) createUserCmd() tea.Cmd {
	vals := make([]string, len(m.formInputs))
	for i, in := range m.formInputs {
		vals[i] = strings.TrimSpace(in.Value())
	}
	cfg := m.cfg
	client := m.client
	return func() tea.Msg {
		tpl, err := config.LoadUserTemplate(cfg)
		if err != nil {
			return actionDoneMsg{err: err}
		}
		err = client.CreateUser(ldapclient.CreateUserInput{
			UID:       vals[0],
			CN:        vals[1],
			SN:        vals[2],
			GivenName: vals[3],
			Mail:      vals[4],
			Password:  vals[5],
			Template:  tpl,
		})
		if err != nil {
			return actionDoneMsg{err: err}
		}
		return actionDoneMsg{message: fmt.Sprintf("Created user %s", vals[0]), reload: true}
	}
}

func (m model) updateUserCmd() tea.Cmd {
	vals := make([]string, len(m.formInputs))
	for i, in := range m.formInputs {
		vals[i] = strings.TrimSpace(in.Value())
	}
	dn := m.formDN
	client := m.client
	return func() tea.Msg {
		err := client.UpdateUser(ldapclient.UpdateUserInput{
			DN:            dn,
			CN:            vals[0],
			SN:            vals[1],
			GivenName:     vals[2],
			Mail:          vals[3],
			Gecos:         vals[4],
			LoginShell:    vals[5],
			HomeDirectory: vals[6],
		})
		if err != nil {
			return actionDoneMsg{err: err}
		}
		return actionDoneMsg{message: "User updated", reload: true}
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
		return actionDoneMsg{message: "Password updated", reload: true}
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
		return actionDoneMsg{message: msg, reload: true}
	}
}

func (m model) deleteUserCmd() tea.Cmd {
	dn := m.formDN
	uid := m.formUID
	client := m.client
	return func() tea.Msg {
		if err := client.DeleteUser(dn); err != nil {
			return actionDoneMsg{err: err}
		}
		return actionDoneMsg{message: fmt.Sprintf("Deleted user %s", uid), reload: true}
	}
}

func (m *model) closeClient() {
	if m.client != nil {
		m.client.Close()
	}
}

func trunc(s string, n int) string {
	return truncRunes(s, n)
}

func Run(cfg *config.Config, password string) error {
	integ, err := config.LoadIntegration()
	if err != nil {
		return err
	}
	p := tea.NewProgram(New(cfg, password, integ), tea.WithAltScreen())
	_, err = p.Run()
	return err
}
