package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"

	"github.com/knieberg/ldash/internal/config"
	ldapclient "github.com/knieberg/ldash/internal/ldap"
)

func (m *model) initTemplateCreateForm(tpl *config.UserTemplate) {
	m.formSpecs = tpl.AllFormFields(true)
	m.formTemplateName = tpl.Name
	m.formTemplateDesc = tpl.Description
	w := formInputWidth(m.width)
	m.formInputs = make([]textinput.Model, len(m.formSpecs))
	placeholders := map[string]string{
		"uid": "alice", "cn": "Alice Example", "sn": "Example",
		"givenName": "Alice", "mail": "alice@example.com",
	}
	for i, spec := range m.formSpecs {
		ph := placeholders[spec.Attr]
		if ph == "" && !spec.Secret {
			ph = "value"
		}
		if spec.Secret {
			ph = "optional"
		}
		m.formInputs[i] = newInput(ph, spec.Secret, w)
	}
	m.formFocus = 0
	m.formInputs[0].Focus()
}

func (m *model) initTemplateEditForm(tpl *config.UserTemplate, u ldapclient.User, extra map[string]string) {
	m.formSpecs = tpl.AllFormFields(false)
	// edit uses subset without uid/password
	filtered := make([]config.FormFieldSpec, 0, len(m.formSpecs))
	for _, s := range m.formSpecs {
		if s.Attr == "uid" || s.Attr == "password" {
			continue
		}
		filtered = append(filtered, s)
	}
	m.formSpecs = filtered
	m.formTemplateName = tpl.Name
	m.formTemplateDesc = tpl.Description
	w := formInputWidth(m.width)
	m.formInputs = make([]textinput.Model, len(m.formSpecs))
	for i, spec := range m.formSpecs {
		m.formInputs[i] = newInput("", spec.Secret, w)
		m.formInputs[i].SetValue(attrValue(u, extra, spec.Attr))
	}
	m.formFocus = 0
	if len(m.formInputs) > 0 {
		m.formInputs[0].Focus()
	}
}

func attrValue(u ldapclient.User, extra map[string]string, attr string) string {
	switch strings.ToLower(attr) {
	case "cn":
		return u.CN
	case "sn":
		return u.SN
	case "givenname":
		return u.GivenName
	case "mail":
		return u.Mail
	case "gecos":
		return u.Gecos
	case "loginshell":
		return u.LoginShell
	case "homedirectory":
		return u.HomeDirectory
	default:
		if extra != nil {
			return extra[attr]
		}
	}
	return ""
}

func (m *model) initGroupCreateForm() {
	tpl, err := config.LoadGroupTemplate(m.cfg)
	if err != nil {
		m.formSpecs = nil
		return
	}
	m.formSpecs = tpl.AllFormFields()
	m.formTemplateName = tpl.Name
	m.formTemplateDesc = tpl.Description
	w := formInputWidth(m.width)
	m.formInputs = make([]textinput.Model, len(m.formSpecs))
	for i, spec := range m.formSpecs {
		ph := "value"
		if spec.Attr == "cn" {
			ph = "developers"
		}
		if spec.Attr == "gidNumber" {
			ph = "auto"
		}
		m.formInputs[i] = newInput(ph, false, w)
	}
	m.formFocus = 0
	m.formInputs[0].Focus()
}

func (m *model) initGroupEditForm(g ldapclient.Group) {
	tpl, _ := config.LoadGroupTemplate(m.cfg)
	if tpl != nil {
		m.formSpecs = tpl.AllFormFields()
		// only description + custom on edit
		m.formSpecs = []config.FormFieldSpec{{Attr: "description", Required: false}}
		for _, c := range tpl.CustomAttributes {
			m.formSpecs = append(m.formSpecs, config.FormFieldSpec{
				Attr: c.Name, Required: c.Required, Custom: true, Label: c.Label, Help: c.Help,
			})
		}
	}
	m.formTemplateName = tpl.Name
	w := formInputWidth(m.width)
	m.formInputs = make([]textinput.Model, len(m.formSpecs))
	for i, spec := range m.formSpecs {
		m.formInputs[i] = newInput("", false, w)
		if spec.Attr == "description" {
			m.formInputs[i].SetValue(g.Description)
		}
	}
	m.formFocus = 0
	m.formInputs[0].Focus()
}

func (m model) viewTemplateForm(title string) string {
	var b strings.Builder
	b.WriteString(headerStyle.Render(title))
	if m.formTemplateName != "" {
		b.WriteString("\n")
		b.WriteString(mutedStyle.Render(fmt.Sprintf("From template: %s — %s", m.formTemplateName, m.formTemplateDesc)))
	}
	b.WriteString("\n\n")
	narrow := appWidth(m.width) < 80
	for i, spec := range m.formSpecs {
		meta := metaForField(spec)
		label := displayLabel(spec)
		prefix := "  "
		if i == m.formFocus {
			prefix = "▸ "
		}
		fmt.Fprintf(&b, "%s%-28s %s\n", prefix, label+":", m.formInputs[i].View())
		if !narrow || i == m.formFocus {
			if meta.Help != "" {
				b.WriteString(mutedStyle.Render("    " + meta.Help))
				b.WriteString("\n")
			}
		}
	}
	return b.String()
}

func (m model) formValues() map[string]string {
	out := make(map[string]string, len(m.formSpecs))
	for i, spec := range m.formSpecs {
		out[spec.Attr] = strings.TrimSpace(m.formInputs[i].Value())
	}
	return out
}

func templateCustomAttrs(tpl *config.UserTemplate) []string {
	out := make([]string, 0, len(tpl.CustomAttributes))
	for _, c := range tpl.CustomAttributes {
		out = append(out, c.Name)
	}
	return out
}

func validateRequired(specs []config.FormFieldSpec, vals map[string]string) error {
	for _, s := range specs {
		if !s.Required && !metaForField(s).Required {
			continue
		}
		if s.Attr == "password" || s.Attr == "gidNumber" {
			continue
		}
		if strings.TrimSpace(vals[s.Attr]) == "" {
			return fmt.Errorf("%s is required", displayLabel(s))
		}
	}
	return nil
}
