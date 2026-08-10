package tui

import (
	ldapclient "github.com/knieberg/ldash/internal/ldap"
	tea "github.com/charmbracelet/bubbletea"
)

type sambaUserReloadMsg struct {
	user    ldapclient.User
	present bool
	err     error
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
