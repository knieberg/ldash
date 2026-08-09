package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

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
	id    viewID
	label string
	hint  string
}

type model struct {
	cfg      *config.Config
	integ    *config.Integration
	password string
	client   *ldapclient.Client

	current viewID
	width   int
	height  int
	status  string
	quitting bool

	menuCursor int
	menuItems  []menuItem

	pingResult *ldapclient.PingResult

	users       []ldapclient.User
	userCursor  int
	userFilter  string
	searching   bool
	searchInput textinput.Model

	formInputs []textinput.Model
	formFocus  int
	formUID    string // for edit/delete/password/mail target
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
		searchInput: si,
		menuItems: []menuItem{
			{viewDashboard, "Dashboard", "Connection health"},
			{viewUsers, "Users", "List, create, edit, delete"},
			{viewGroups, "Groups", "Coming in v0.2"},
			{viewSamba, "Samba", "Coming soon"},
			{viewIntegration, "Integration Guide", "Runtime snippets from local config"},
			{viewSettings, "Settings", "Shows loaded connection profile"},
		},
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
		return m, nil

	case usersLoadedMsg:
		if msg.err != nil {
			m.status = msg.err.Error()
			m.users = nil
		} else {
			m.users = msg.users
			if m.userCursor >= len(m.users) {
				m.userCursor = 0
			}
			m.status = fmt.Sprintf("%d user(s)", len(m.users))
		}
		return m, nil

	case groupsLoadedMsg:
		if msg.err != nil {
			m.status = msg.err.Error()
		} else {
			m.groups = msg.groups
		}
		return m, nil

	case actionDoneMsg:
		if msg.err != nil {
			m.status = msg.err.Error()
		} else {
			m.status = msg.message
			if msg.reload {
				m.current = viewUsers
				m.confirm = false
				return m, m.loadUsersCmd()
			}
		}
		return m, nil

	case tea.KeyMsg:
		if m.searching && m.current == viewUsers {
			return m.updateUserSearch(msg)
		}
		if m.current == viewUserCreate || m.current == viewUserEdit || m.current == viewUserPassword || m.current == viewUserMail {
			return m.updateForm(msg)
		}
		if m.current == viewUserDelete {
			return m.updateDelete(msg)
		}
		return m.updateNav(msg)
	}
	return m, nil
}

func (m model) updateNav(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		m.quitting = true
		m.closeClient()
		return m, tea.Quit
	case keyQuit:
		if m.current == viewMenu {
			m.quitting = true
			m.closeClient()
			return m, tea.Quit
		}
		m.current = viewMenu
		m.status = "Main menu"
		return m, nil
	case keyBack:
		if m.current != viewMenu {
			m.current = viewMenu
			m.status = "Main menu"
			return m, nil
		}
	}

	switch m.current {
	case viewMenu:
		switch msg.String() {
		case keyUp, "up":
			if m.menuCursor > 0 {
				m.menuCursor--
			}
		case keyDown, "down":
			if m.menuCursor < len(m.menuItems)-1 {
				m.menuCursor++
			}
		case keyEnter:
			item := m.menuItems[m.menuCursor]
			m.current = item.id
			m.status = item.label
			if item.id == viewUsers {
				return m, m.ensureConnAnd(m.loadUsersCmd())
			}
		}
	case viewDashboard:
		if msg.String() == keyRefresh {
			m.status = "Testing connection..."
			return m, m.pingCmd()
		}
	case viewUsers:
		return m.updateUsersList(msg)
	}
	return m, nil
}

func (m model) updateUsersList(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case keyUp, "up":
		if m.userCursor > 0 {
			m.userCursor--
		}
	case keyDown, "down":
		if m.userCursor < len(m.users)-1 {
			m.userCursor++
		}
	case keyRefresh:
		return m, m.ensureConnAnd(m.loadUsersCmd())
	case keySearch:
		m.searching = true
		m.searchInput.SetValue(m.userFilter)
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
	case keyEnter:
		m.userFilter = m.searchInput.Value()
		m.searching = false
		m.searchInput.Blur()
		return m, m.ensureConnAnd(m.loadUsersCmd())
	}
	var cmd tea.Cmd
	m.searchInput, cmd = m.searchInput.Update(msg)
	return m, cmd
}

func (m model) updateForm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case keyBack:
		m.current = viewUsers
		m.status = "Cancelled"
		return m, nil
	case "tab", "down":
		m.formFocus = (m.formFocus + 1) % len(m.formInputs)
		return m, m.focusForm()
	case "shift+tab", "up":
		m.formFocus = (m.formFocus - 1 + len(m.formInputs)) % len(m.formInputs)
		return m, m.focusForm()
	case keyEnter:
		return m.submitForm()
	}
	var cmd tea.Cmd
	m.formInputs[m.formFocus], cmd = m.formInputs[m.formFocus].Update(msg)
	return m, cmd
}

func (m model) updateDelete(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case keyBack, "n":
		m.current = viewUsers
		m.status = "Delete cancelled"
		return m, nil
	case "y":
		if !m.confirm {
			m.confirm = true
			m.status = "Press y again to confirm delete"
			return m, nil
		}
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

func (m model) View() string {
	if m.quitting {
		return ""
	}
	var b strings.Builder
	b.WriteString(titleStyle.Render("ldash — LDAP Admin Shell"))
	b.WriteString("\n")
	b.WriteString(mutedStyle.Render(m.breadcrumb()))
	b.WriteString("\n\n")

	switch m.current {
	case viewMenu:
		b.WriteString(m.viewMenu())
	case viewDashboard:
		b.WriteString(m.viewDashboard())
	case viewUsers:
		b.WriteString(m.viewUsers())
	case viewGroups:
		b.WriteString(mutedStyle.Render("Group membership management is planned for v0.2."))
	case viewSamba:
		b.WriteString(mutedStyle.Render("Samba status view is planned for a later release."))
	case viewIntegration:
		b.WriteString(m.viewIntegration())
	case viewSettings:
		b.WriteString(m.viewSettings())
	case viewUserCreate, viewUserEdit, viewUserPassword, viewUserMail:
		b.WriteString(m.viewForm())
	case viewUserDelete:
		b.WriteString(m.viewDelete())
	}

	b.WriteString("\n\n")
	b.WriteString(mutedStyle.Render(m.footer()))
	b.WriteString("\n")
	if m.status != "" {
		b.WriteString(m.status)
	}
	return b.String()
}

func (m model) breadcrumb() string {
	switch m.current {
	case viewMenu:
		return "Main Menu"
	case viewDashboard:
		return "Main Menu › Dashboard"
	case viewUsers:
		return "Main Menu › Users"
	case viewUserCreate:
		return "Main Menu › Users › Create"
	case viewUserEdit:
		return "Main Menu › Users › Edit"
	case viewUserDelete:
		return "Main Menu › Users › Delete"
	case viewUserPassword:
		return "Main Menu › Users › Password"
	case viewUserMail:
		return "Main Menu › Users › Mail"
	case viewGroups:
		return "Main Menu › Groups"
	case viewSamba:
		return "Main Menu › Samba"
	case viewIntegration:
		return "Main Menu › Integration Guide"
	case viewSettings:
		return "Main Menu › Settings"
	default:
		return ""
	}
}

func (m model) footer() string {
	switch m.current {
	case viewMenu:
		return "Keys: j/k move · Enter open · q quit"
	case viewDashboard:
		return "Keys: r test connection · Esc/q back"
	case viewUsers:
		if m.searching {
			return "Keys: Enter apply filter · Esc cancel"
		}
		return "Keys: j/k · / search · c create · e edit · d delete · p password · m mail · r refresh · Esc/q back"
	case viewUserCreate, viewUserEdit, viewUserPassword, viewUserMail:
		return "Keys: Tab next field · Enter submit · Esc cancel"
	case viewUserDelete:
		return "Keys: y confirm · n/Esc cancel"
	default:
		return "Keys: Esc/q back"
	}
}

func (m model) viewMenu() string {
	var b strings.Builder
	for i, item := range m.menuItems {
		line := fmt.Sprintf("  %-18s %s", item.label, mutedStyle.Render(item.hint))
		if i == m.menuCursor {
			line = selStyle.Render(fmt.Sprintf("▸ %-18s %s", item.label, item.hint))
		}
		b.WriteString(line)
		b.WriteString("\n")
	}
	return b.String()
}

func (m model) viewDashboard() string {
	var b strings.Builder
	lines := []string{
		fmt.Sprintf("Server:  %s (%s)", m.cfg.Server.URL, m.cfg.Server.TLSMode),
		fmt.Sprintf("Base DN: %s", m.cfg.BaseDN),
		fmt.Sprintf("People:  %s", m.cfg.PeopleDN()),
		fmt.Sprintf("Groups:  %s", m.cfg.GroupsDN()),
	}
	b.WriteString(boxStyle.Render(strings.Join(lines, "\n")))
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

func (m model) viewUsers() string {
	var b strings.Builder
	if m.searching {
		b.WriteString("Filter: ")
		b.WriteString(m.searchInput.View())
		b.WriteString("\n\n")
	} else if m.userFilter != "" {
		b.WriteString(mutedStyle.Render(fmt.Sprintf("Filter: %q", m.userFilter)))
		b.WriteString("\n\n")
	}
	if len(m.users) == 0 {
		b.WriteString(mutedStyle.Render("No users loaded. Press r to refresh."))
		return b.String()
	}
	b.WriteString(headerStyle.Render(fmt.Sprintf("%-16s %-24s %-8s %s", "UID", "CN", "UID#", "MAIL")))
	b.WriteString("\n")
	for i, u := range m.users {
		line := fmt.Sprintf("%-16s %-24s %-8d %s", trunc(u.UID, 16), trunc(u.CN, 24), u.UIDNumber, trunc(u.Mail, 32))
		if i == m.userCursor {
			b.WriteString(selStyle.Render("▸ " + line))
		} else {
			b.WriteString("  " + line)
		}
		b.WriteString("\n")
	}
	return b.String()
}

func (m model) viewIntegration() string {
	var b strings.Builder
	b.WriteString(headerStyle.Render("Values from local runtime config only"))
	b.WriteString("\n\n")
	b.WriteString(boxStyle.Render(strings.Join([]string{
		fmt.Sprintf("Server URI:   %s", m.cfg.Server.URL),
		fmt.Sprintf("Base DN:      %s", m.cfg.BaseDN),
		fmt.Sprintf("Bind DN:      %s", m.cfg.BindDN),
		fmt.Sprintf("Users base:   %s", m.cfg.PeopleDN()),
		fmt.Sprintf("Groups base:  %s", m.cfg.GroupsDN()),
		fmt.Sprintf("User filter:  %s", m.cfg.Search.UserFilter),
	}, "\n")))
	b.WriteString("\n\n")
	if m.integ.SelfServiceURL != "" {
		b.WriteString(fmt.Sprintf("Self-service URL: %s\n", m.integ.SelfServiceURL))
	}
	if m.integ.OIDCIssuer != "" {
		b.WriteString(fmt.Sprintf("OIDC issuer: %s\n", m.integ.OIDCIssuer))
	}
	if m.integ.OIDCProvider != "" {
		b.WriteString(fmt.Sprintf("OIDC provider: %s\n", m.integ.OIDCProvider))
	}
	if len(m.integ.OnboardingChecklist) > 0 {
		b.WriteString("\nOnboarding checklist:\n")
		for _, step := range m.integ.OnboardingChecklist {
			b.WriteString("  - " + step + "\n")
		}
	}
	if m.integ.SelfServiceURL == "" && m.integ.OIDCIssuer == "" && len(m.integ.OnboardingChecklist) == 0 {
		b.WriteString(mutedStyle.Render("Optional: create ~/.config/ldash/integration.yaml for extra hints."))
	}
	return b.String()
}

func (m model) viewSettings() string {
	return boxStyle.Render(strings.Join([]string{
		fmt.Sprintf("Config profile: single file (MVP)"),
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
		b.WriteString(fmt.Sprintf("%s%-14s %s\n", prefix, labels[i]+":", in.View()))
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
			b.WriteString(fmt.Sprintf("  - %s (%s)\n", g.CN, g.Attr))
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

func newInput(placeholder string, password bool) textinput.Model {
	ti := textinput.New()
	ti.Placeholder = placeholder
	ti.CharLimit = 128
	ti.Width = 40
	if password {
		ti.EchoMode = textinput.EchoPassword
		ti.EchoCharacter = '•'
	}
	return ti
}

func (m *model) initCreateForm() {
	m.formInputs = []textinput.Model{
		newInput("alice", false),
		newInput("Alice Example", false),
		newInput("Example", false),
		newInput("Alice", false),
		newInput("alice@example.com", false),
		newInput("initial password", true),
	}
	m.formFocus = 0
	m.formInputs[0].Focus()
}

func (m *model) initEditForm(u ldapclient.User) {
	m.formUID = u.UID
	m.formDN = u.DN
	vals := []string{u.CN, u.SN, u.GivenName, u.Mail, u.Gecos, u.LoginShell, u.HomeDirectory}
	m.formInputs = make([]textinput.Model, len(vals))
	for i, v := range vals {
		m.formInputs[i] = newInput("", false)
		m.formInputs[i].SetValue(v)
	}
	m.formFocus = 0
	m.formInputs[0].Focus()
}

func (m *model) initPasswordForm(u ldapclient.User) {
	m.formUID = u.UID
	m.formDN = u.DN
	m.formInputs = []textinput.Model{newInput("new password", true)}
	m.formFocus = 0
	m.formInputs[0].Focus()
}

func (m *model) initMailForm(u ldapclient.User) {
	m.formUID = u.UID
	m.formDN = u.DN
	m.formInputs = []textinput.Model{newInput("user@example.com (empty to clear)", false)}
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
	if len(s) <= n {
		return s
	}
	if n <= 1 {
		return s[:n]
	}
	return s[:n-1] + "…"
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