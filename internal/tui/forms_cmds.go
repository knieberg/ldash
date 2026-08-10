package tui

import (
	"fmt"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/knieberg/ldash/internal/config"
	ldapclient "github.com/knieberg/ldash/internal/ldap"
)

var builtinUserAttrs = map[string]bool{
	"uid": true, "cn": true, "sn": true, "givenname": true, "mail": true,
	"password": true, "gecos": true, "loginshell": true, "homedirectory": true,
	"uidnumber": true, "gidnumber": true,
}

var builtinGroupAttrs = map[string]bool{
	"cn": true, "gidnumber": true, "description": true,
}

func splitUserValues(specs []config.FormFieldSpec, vals map[string]string) (ldapclient.CreateUserInput, []string, error) {
	in := ldapclient.CreateUserInput{Extra: map[string]string{}}
	var clear []string
	for _, s := range specs {
		v := vals[s.Attr]
		key := strings.ToLower(s.Attr)
		switch key {
		case "uid":
			in.UID = v
		case "cn":
			in.CN = v
		case "sn":
			in.SN = v
		case "givenname":
			in.GivenName = v
		case "mail":
			in.Mail = v
		case "password":
			in.Password = v
		case "gecos":
			in.Gecos = v
		case "loginshell":
			in.LoginShell = v
		case "homedirectory":
			in.HomeDirectory = v
		default:
			if strings.TrimSpace(v) == "" {
				if !s.Required {
					clear = append(clear, s.Attr)
				}
			} else {
				in.Extra[s.Attr] = v
			}
		}
	}
	return in, clear, nil
}

func splitUserUpdateValues(specs []config.FormFieldSpec, vals map[string]string) (ldapclient.UpdateUserInput, []string) {
	in := ldapclient.UpdateUserInput{Extra: map[string]string{}}
	var clear []string
	for _, s := range specs {
		v := vals[s.Attr]
		key := strings.ToLower(s.Attr)
		switch key {
		case "cn":
			in.CN = v
		case "sn":
			in.SN = v
		case "givenname":
			in.GivenName = v
		case "mail":
			in.Mail = v
		case "gecos":
			in.Gecos = v
		case "loginshell":
			in.LoginShell = v
		case "homedirectory":
			in.HomeDirectory = v
		default:
			if strings.TrimSpace(v) == "" {
				clear = append(clear, s.Attr)
			} else {
				in.Extra[s.Attr] = v
			}
		}
	}
	return in, clear
}

func splitGroupCreateValues(specs []config.FormFieldSpec, vals map[string]string) ldapclient.CreateGroupInput {
	in := ldapclient.CreateGroupInput{Extra: map[string]string{}}
	for _, s := range specs {
		v := vals[s.Attr]
		switch strings.ToLower(s.Attr) {
		case "cn":
			in.CN = v
		case "gidnumber":
			if v != "" && v != "auto" {
				if n, err := strconv.Atoi(v); err == nil {
					in.GIDNumber = n
				}
			}
		case "description":
			in.Description = v
		default:
			if strings.TrimSpace(v) != "" {
				in.Extra[s.Attr] = v
			}
		}
	}
	return in
}

func splitGroupUpdateValues(specs []config.FormFieldSpec, vals map[string]string) (ldapclient.UpdateGroupInput, []string) {
	in := ldapclient.UpdateGroupInput{Extra: map[string]string{}}
	var clear []string
	for _, s := range specs {
		v := vals[s.Attr]
		switch strings.ToLower(s.Attr) {
		case "description":
			in.Description = v
		default:
			if strings.TrimSpace(v) == "" {
				clear = append(clear, s.Attr)
			} else {
				in.Extra[s.Attr] = v
			}
		}
	}
	return in, clear
}

func (m model) createUserCmd() tea.Cmd {
	specs := append([]config.FormFieldSpec(nil), m.formSpecs...)
	vals := m.formValues()
	cfg := m.cfg
	client := m.client
	return func() tea.Msg {
		if err := validateRequired(specs, vals); err != nil {
			return actionDoneMsg{err: err}
		}
		tpl, err := config.LoadUserTemplate(cfg)
		if err != nil {
			return actionDoneMsg{err: err}
		}
		gtpl, err := config.LoadGroupTemplate(cfg)
		if err != nil && tpl.CreatePrimaryGroup {
			return actionDoneMsg{err: err}
		}
		in, _, err := splitUserValues(specs, vals)
		if err != nil {
			return actionDoneMsg{err: err}
		}
		in.Template = tpl
		in.GroupTemplate = gtpl
		if err := client.CreateUser(in); err != nil {
			return actionDoneMsg{err: err}
		}
		return actionDoneMsg{message: fmt.Sprintf("Created user %s", in.UID), reloadView: viewUsers}
	}
}

func (m model) updateUserCmd() tea.Cmd {
	specs := append([]config.FormFieldSpec(nil), m.formSpecs...)
	vals := m.formValues()
	dn := m.formDN
	client := m.client
	return func() tea.Msg {
		if err := validateRequired(specs, vals); err != nil {
			return actionDoneMsg{err: err}
		}
		in, clear := splitUserUpdateValues(specs, vals)
		in.DN = dn
		in.ClearExtra = clear
		if err := client.UpdateUser(in); err != nil {
			return actionDoneMsg{err: err}
		}
		return actionDoneMsg{message: "User updated", reloadView: viewUsers}
	}
}

func (m model) createGroupCmd() tea.Cmd {
	specs := append([]config.FormFieldSpec(nil), m.formSpecs...)
	vals := m.formValues()
	cfg := m.cfg
	client := m.client
	return func() tea.Msg {
		if err := validateRequired(specs, vals); err != nil {
			return actionDoneMsg{err: err}
		}
		tpl, err := config.LoadGroupTemplate(cfg)
		if err != nil {
			return actionDoneMsg{err: err}
		}
		in := splitGroupCreateValues(specs, vals)
		in.Template = tpl
		if err := client.CreateGroup(in); err != nil {
			return actionDoneMsg{err: err}
		}
		return actionDoneMsg{message: fmt.Sprintf("Created group %s", in.CN), reloadView: viewGroups}
	}
}

func (m model) updateGroupCmd() tea.Cmd {
	specs := append([]config.FormFieldSpec(nil), m.formSpecs...)
	vals := m.formValues()
	dn := m.formDN
	client := m.client
	return func() tea.Msg {
		in, clear := splitGroupUpdateValues(specs, vals)
		in.DN = dn
		in.ClearExtra = clear
		if err := client.UpdateGroup(in); err != nil {
			return actionDoneMsg{err: err}
		}
		return actionDoneMsg{message: "Group updated", reloadView: viewGroups}
	}
}

func (m model) sambaFlagsCmd() tea.Cmd {
	flags := strings.TrimSpace(m.formInputs[0].Value())
	dn := m.formDN
	client := m.client
	uid := m.formUID
	if m.sambaUser != nil {
		uid = m.sambaUser.UID
	}
	return func() tea.Msg {
		if err := client.SetSambaFlags(dn, flags); err != nil {
			return actionDoneMsg{err: err}
		}
		return actionDoneMsg{message: fmt.Sprintf("Samba flags updated for %s", uid), reloadView: viewSambaUser}
	}
}

func (m model) deleteUserCmd() tea.Cmd {
	dn := m.formDN
	uid := m.formUID
	client := m.client
	return func() tea.Msg {
		if _, err := client.DeleteUser(uid, dn); err != nil {
			return actionDoneMsg{err: err}
		}
		return actionDoneMsg{message: fmt.Sprintf("Deleted user %s", uid), reloadView: viewUsers}
	}
}

func (m model) deleteGroupCmd() tea.Cmd {
	dn := m.formDN
	cn := m.formUID
	gid := m.formGID
	client := m.client
	return func() tea.Msg {
		if err := client.DeleteGroup(dn, gid); err != nil {
			return actionDoneMsg{err: err}
		}
		return actionDoneMsg{message: fmt.Sprintf("Deleted group %s", cn), reloadView: viewGroups}
	}
}
