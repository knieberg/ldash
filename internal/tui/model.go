package tui

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/knieberg/ldash/internal/config"
	ldifpkg "github.com/knieberg/ldash/internal/ldif"
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
	viewLDIF
	viewGroupCreate
	viewGroupEdit
	viewGroupDelete
	viewGroupMembers
	viewGroupMemberAdd
	viewSambaUser
	viewSambaFlags
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
	message    string
	err        error
	reloadView viewID
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

	groupsFull       []ldapclient.Group
	groupCursor      int
	groupFilter      string
	groupSearching   bool
	groupSearchInput textinput.Model
	groupScrollOffset int
	memberAttr       string
	members          []string
	memberCursor     int
	formGID          int

	formSpecs        []config.FormFieldSpec
	formTemplateName string
	formTemplateDesc string

	ldifStep         int
	ldifAction       string
	ldifScopeIdx     int
	ldifCursor       int
	ldifPath         string
	ldifPathInput    textinput.Model
	ldifPreview      int
	ldifResult       *ldifpkg.ApplyResult
	ldifExportCount  int

	sambaUser        *ldapclient.User
	sambaNTPresent   bool
}

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

type sambaStatusMsg struct {
	present bool
	err     error
}

type sambaUserReloadMsg struct {
	user    ldapclient.User
	present bool
	err     error
}

type groupMembersLoadedMsg struct {
	members    []string
	memberAttr string
}


func New(cfg *config.Config, password string, integ *config.Integration) tea.Model {
	if integ == nil {
		integ = &config.Integration{}
	}
	si := textinput.New()
	si.Placeholder = "filter..."
	si.CharLimit = 64
	si.Width = 40
	gsi := textinput.New()
	gsi.Placeholder = "filter groups..."
	gsi.CharLimit = 64
	gsi.Width = 40
	lpi := textinput.New()
	lpi.CharLimit = 256
	lpi.Width = 60

	return model{
		cfg:              cfg,
		integ:            integ,
		password:         password,
		client:           ldapclient.NewClient(cfg),
		current:          viewMenu,
		status:           "Select a view and press Enter",
		statusK:          statusInfo,
		searchInput:      si,
		groupSearchInput: gsi,
		ldifPathInput:    lpi,
		menuItems: []menuItem{
			{viewDashboard, "Dashboard", "Connection health", true},
			{viewUsers, "Users", "List, create, edit, delete", true},
			{viewGroups, "Groups", "List, create, edit, members", true},
			{viewLDIF, "LDIF", "Export and import backup", true},
			{viewSamba, "Samba", "Samba attribute status", true},
			{viewIntegration, "Integration Guide", "Runtime snippets from local config", true},
			{viewSettings, "Settings", "Connection profile and templates", true},
		},
	}
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

	case groupsLoadedFullMsg:
		m.busy = false
		if msg.err != nil {
			m.setStatus(statusError, msg.err.Error())
			m.groupsFull = nil
		} else {
			m.groupsFull = msg.groups
			if m.groupCursor >= len(m.groupsFull) {
				m.groupCursor = 0
			}
			m.ensureGroupVisible()
			m.setStatus(statusInfo, fmt.Sprintf("%d group(s)", len(m.groupsFull)))
		}
		return m, nil

	case ldifPreviewMsg:
		m.busy = false
		if msg.err != nil {
			m.setStatus(statusError, msg.err.Error())
		} else {
			m.ldifPreview = msg.count
			m.ldifStep = ldifStepConfirm
			m.confirm = false
			m.setStatus(statusInfo, fmt.Sprintf("%d entries in file", msg.count))
		}
		return m, nil

	case ldifDoneMsg:
		m.busy = false
		if msg.err != nil {
			m.setStatus(statusError, msg.err.Error())
		} else {
			m.ldifResult = msg.result
			m.ldifExportCount = msg.count
			m.ldifStep = ldifStepSummary
			if msg.result != nil {
				m.setStatus(statusOK, fmt.Sprintf("Applied %d, skipped %d", msg.result.Applied, msg.result.Skipped))
			} else {
				m.setStatus(statusOK, fmt.Sprintf("Exported %d entries", msg.count))
			}
		}
		return m, nil

	case sambaStatusMsg:
		m.busy = false
		if msg.err != nil {
			m.setStatus(statusError, msg.err.Error())
		} else {
			m.sambaNTPresent = msg.present
			m.setStatus(statusInfo, "Samba status loaded")
		}
		return m, nil

	case sambaUserReloadMsg:
		m.busy = false
		if msg.err != nil {
			m.setStatus(statusError, msg.err.Error())
		} else {
			u := msg.user
			m.sambaUser = &u
			m.sambaNTPresent = msg.present
			m.setStatus(statusInfo, "Samba status loaded")
		}
		return m, nil

	case actionDoneMsg:
		m.busy = false
		if msg.err != nil {
			m.setStatus(statusError, msg.err.Error())
		} else {
			m.setStatus(statusOK, msg.message)
			m.confirm = false
			switch msg.reloadView {
			case viewUsers:
				m.current = viewUsers
				m.busy = true
				m.setStatus(statusLoading, "Refreshing users...")
				return m, m.loadUsersCmd()
			case viewGroups:
				m.current = viewGroups
				m.busy = true
				m.setStatus(statusLoading, "Refreshing groups...")
				return m, m.ensureConnAnd(m.loadGroupsFullCmd())
			case viewGroupMembers:
				m.busy = true
				m.setStatus(statusLoading, "Refreshing members...")
				return m, m.ensureConnAnd(m.loadGroupMembersCmd())
			case viewSambaUser:
				m.current = viewSambaUser
				if m.sambaUser != nil {
					m.busy = true
					return m, m.ensureConnAnd(m.reloadSambaUserCmd())
				}
			}
		}
		return m, nil

	case groupMembersLoadedMsg:
		m.busy = false
		m.members = msg.members
		if msg.memberAttr != "" {
			m.memberAttr = msg.memberAttr
		}
		if m.memberCursor >= len(m.members) {
			m.memberCursor = 0
		}
		m.setStatus(statusInfo, fmt.Sprintf("%d member(s)", len(m.members)))
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
		if m.groupSearching && m.current == viewGroups {
			return m.updateGroupSearch(msg)
		}
		if m.ldifStep == ldifStepPath && m.current == viewLDIF {
			return m.updateLDIFPath(msg)
		}
		if m.isFormView() {
			return m.updateForm(msg)
		}
		if m.current == viewUserDelete || m.current == viewGroupDelete {
			return m.updateDelete(msg)
		}
		if m.current == viewLDIF && m.ldifStep == ldifStepConfirm {
			return m.updateLDIFConfirm(msg)
		}
		if m.current == viewGroupMembers {
			return m.updateGroupMembers(msg)
		}
		if m.current == viewGroupMemberAdd {
			return m.updateGroupMemberAdd(msg)
		}
		if m.current == viewSambaUser {
			return m.updateSambaUser(msg)
		}
		return m.updateNav(msg)
	}
	return m, nil
}

func (m model) isFormView() bool {
	return m.current == viewUserCreate || m.current == viewUserEdit ||
		m.current == viewUserPassword || m.current == viewUserMail ||
		m.current == viewGroupCreate || m.current == viewGroupEdit ||
		m.current == viewSambaFlags || m.current == viewGroupMemberAdd
}

func (m model) isUserChild() bool {
	return m.isFormView() || m.current == viewUserDelete || m.current == viewSambaUser
}

func (m model) isGroupChild() bool {
	return m.current == viewGroupCreate || m.current == viewGroupEdit ||
		m.current == viewGroupDelete || m.current == viewGroupMembers ||
		m.current == viewGroupMemberAdd
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
	if m.groupSearching && m.current == viewGroups {
		m.groupSearching = false
		m.groupSearchInput.Blur()
		return m, nil
	}
	if m.current == viewSambaFlags {
		m.current = viewSambaUser
		m.setStatus(statusInfo, "Cancelled")
		return m, nil
	}
	if m.isUserChild() {
		m.current = viewUsers
		m.confirm = false
		m.setStatus(statusInfo, "Cancelled")
		return m, nil
	}
	if m.isGroupChild() {
		if m.current == viewGroupMemberAdd {
			m.current = viewGroupMembers
			return m, nil
		}
		if m.current == viewGroupMembers {
			m.current = viewGroups
			return m, nil
		}
		m.current = viewGroups
		m.confirm = false
		m.setStatus(statusInfo, "Cancelled")
		return m, nil
	}
	if m.current == viewLDIF {
		switch m.ldifStep {
		case ldifStepSummary, ldifStepConfirm:
			m.ldifStep = ldifStepHub
			m.confirm = false
			return m, nil
		case ldifStepPath:
			m.ldifStep = ldifStepHub
			m.ldifPathInput.Blur()
			return m, nil
		}
	}
	if m.current == viewSambaUser {
		m.current = viewUsers
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
	case viewGroups:
		return m.updateGroupsList(msg)
	case viewLDIF:
		return m.updateLDIF(msg)
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
	if item.id == viewGroups {
		m.busy = true
		m.setStatus(statusLoading, "Loading groups...")
		return m, m.ensureConnAnd(m.loadGroupsFullCmd())
	}
	if item.id == viewLDIF {
		m.ldifStep = ldifStepHub
		m.ldifCursor = 0
	}
	return m, nil
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
		if m.current == viewGroupDelete {
			m.setStatus(statusLoading, "Deleting group...")
			return m, m.ensureConnAnd(m.deleteGroupCmd())
		}
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
		content = m.viewGroupsList()
	case viewSamba:
		content = m.viewSambaHub()
	case viewLDIF:
		content = m.viewLDIF()
	case viewIntegration:
		content = m.viewIntegration()
	case viewSettings:
		content = m.viewSettings()
	case viewUserCreate, viewUserEdit, viewGroupCreate, viewGroupEdit, viewSambaFlags:
		titles := map[viewID]string{
			viewUserCreate:  "Create user",
			viewUserEdit:    "Edit user " + m.formUID,
			viewGroupCreate: "Create group",
			viewGroupEdit:   "Edit group " + m.formUID,
			viewSambaFlags:  "Edit Samba flags for " + m.formUID,
		}
		content = m.viewTemplateForm(titles[m.current])
	case viewUserPassword, viewUserMail:
		content = m.viewSimpleForm()
	case viewUserDelete, viewGroupDelete:
		content = m.viewDelete()
	case viewGroupMembers:
		content = m.viewGroupMembers()
	case viewGroupMemberAdd:
		content = m.viewMemberAddForm()
	case viewSambaUser:
		content = m.viewSambaUser()
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
	b.WriteString("  1-7     open main menu item\n")
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
		return helpLines([]keyAction{{keyUp + "/" + keyDown, "move"}, {"1-7", "open"}, {keyEnter, "open"}})
	case viewDashboard:
		return helpLines([]keyAction{{keyRefresh, "test connection"}})
	case viewUsers:
		if m.searching {
			return helpLines([]keyAction{{keyEnter, "apply filter"}, {keyBack, "cancel"}, {"Ctrl+U", "clear text"}})
		}
		return helpLines([]keyAction{
			{keyUp + "/" + keyDown, "move"}, {"PgUp/PgDn", "page"}, {"g/G", "first/last"},
			{keySearch, "search"}, {keyCreate, "create"}, {keyEdit, "edit"}, {keyDelete, "delete"},
			{keyPassword, "password"}, {keyMail, "mail"}, {keySamba, "samba"}, {keyRefresh, "refresh"},
		})
	case viewGroups:
		if m.groupSearching {
			return helpLines([]keyAction{{keyEnter, "apply filter"}, {keyBack, "cancel"}})
		}
		return helpLines([]keyAction{
			{keyUp + "/" + keyDown, "move"}, {keySearch, "search"}, {keyCreate, "create"},
			{keyEdit, "edit"}, {keyDelete, "delete"}, {keyMembers, "members"}, {keyRefresh, "refresh"},
		})
	case viewLDIF:
		return helpLines([]keyAction{{keyUp + "/" + keyDown, "move"}, {keyEnter, "open"}, {keyBack, "back"}})
	case viewSamba:
		return helpLines([]keyAction{{keyBack, "back"}})
	case viewSambaUser:
		return helpLines([]keyAction{{keyEdit, "edit flags"}, {keyBack, "back"}})
	case viewUserCreate, viewUserEdit, viewGroupCreate, viewGroupEdit, viewSambaFlags:
		return helpLines([]keyAction{{"Tab", "next field"}, {keyEnter, "submit"}, {keyBack, "cancel"}})
	case viewUserPassword, viewUserMail, viewGroupMemberAdd:
		return helpLines([]keyAction{{keyEnter, "submit"}, {keyBack, "cancel"}})
	case viewUserDelete, viewGroupDelete:
		return helpLines([]keyAction{{"y", "confirm (twice)"}, {"n", "cancel"}})
	case viewGroupMembers:
		return helpLines([]keyAction{{keyCreate, "add member"}, {keyDelete, "remove"}, {keyUp + "/" + keyDown, "move"}})
	default:
		return helpLines([]keyAction{{keyBack, "back"}})
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
	case viewGroupCreate:
		return []string{"Main Menu", "Groups", "Create"}
	case viewGroupEdit:
		return []string{"Main Menu", "Groups", "Edit"}
	case viewGroupDelete:
		return []string{"Main Menu", "Groups", "Delete"}
	case viewGroupMembers:
		return []string{"Main Menu", "Groups", "Members"}
	case viewGroupMemberAdd:
		return []string{"Main Menu", "Groups", "Members", "Add"}
	case viewLDIF:
		return []string{"Main Menu", "LDIF"}
	case viewSamba:
		return []string{"Main Menu", "Samba"}
	case viewSambaUser:
		return []string{"Main Menu", "Users", "Samba"}
	case viewSambaFlags:
		return []string{"Main Menu", "Users", "Samba", "Flags"}
	case viewIntegration:
		return []string{"Main Menu", "Integration Guide"}
	case viewSettings:
		return []string{"Main Menu", "Settings"}
	default:
		return nil
	}
}

func (m model) footerKeys() string {
	if m.busy {
		return joinHints([]keyAction{{keyHelp, "help"}}, 0)
	}
	switch m.current {
	case viewMenu:
		return footerMenu()
	case viewDashboard:
		return joinHints([]keyAction{{keyRefresh, "test"}, {keyBack, "back"}, {keyHelp, "help"}}, 0)
	case viewUsers:
		if m.searching {
			return joinHints([]keyAction{{keyEnter, "apply"}, {keyBack, "cancel"}, {keyHelp, "help"}}, 0)
		}
		return footerUsersList(m.userFilter != "")
	case viewGroups:
		if m.groupSearching {
			return joinHints([]keyAction{{keyEnter, "apply"}, {keyBack, "cancel"}, {keyHelp, "help"}}, 0)
		}
		return footerGroupsList()
	case viewLDIF:
		switch m.ldifStep {
		case ldifStepHub:
			return footerLDIFHub()
		case ldifStepPath:
			return joinHints([]keyAction{{keyEnter, "submit"}, {keyBack, "back"}, {keyHelp, "help"}}, 0)
		case ldifStepConfirm:
			return footerDeleteConfirm()
		case ldifStepSummary:
			return joinHints([]keyAction{{keyBack, "back"}, {keyHelp, "help"}}, 0)
		}
	case viewSamba:
		return joinHints([]keyAction{{keyBack, "back"}, {keyHelp, "help"}}, 0)
	case viewSambaUser:
		return joinHints([]keyAction{{keyEdit, "edit flags"}, {keyBack, "back"}, {keyHelp, "help"}}, 0)
	case viewUserCreate, viewUserEdit, viewGroupCreate, viewGroupEdit, viewSambaFlags:
		return footerForm()
	case viewUserPassword, viewUserMail, viewGroupMemberAdd:
		return footerForm()
	case viewUserDelete, viewGroupDelete:
		return footerDeleteConfirm()
	case viewGroupMembers:
		return joinHints([]keyAction{
			{keyCreate, "add"}, {keyDelete, "remove"}, {keyUp + "/" + keyDown, "move"},
			{keyBack, "back"}, {keyHelp, "help"},
		}, 0)
	case viewIntegration, viewSettings:
		return joinHints(footerCommonBackHelp(), 0)
	}
	return joinHints(footerCommonBackHelp(), 0)
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
	case viewGroupCreate:
		return m, m.ensureConnAnd(m.createGroupCmd())
	case viewGroupEdit:
		return m, m.ensureConnAnd(m.updateGroupCmd())
	case viewSambaFlags:
		return m, m.ensureConnAnd(m.sambaFlagsCmd())
	}
	return m, nil
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

func (m *model) closeClient() {
	if m.client != nil {
		m.client.Close()
	}
}
