package tui

import (
	"fmt"
	"strings"

	"github.com/knieberg/ldash/internal/config"
)

type fieldMeta struct {
	Label    string
	Help     string
	Secret   bool
	Required bool
}

var builtinGlossary = map[string]fieldMeta{
	"uid":            {Label: "Login name", Help: "Account identifier used to sign in (uid)", Required: true},
	"cn":             {Label: "Full name", Help: "Display name shown in directories (cn)", Required: true},
	"sn":             {Label: "Surname", Help: "Family name (LDAP sn)", Required: true},
	"givenName":      {Label: "Given name", Help: "First or given name (givenName)"},
	"mail":           {Label: "Email", Help: "Primary email address (mail)"},
	"password":       {Label: "Initial password", Help: "Optional; set via Password Modify after create", Secret: true},
	"gecos":          {Label: "GECOS", Help: "Full name or comment field (gecos)"},
	"loginShell":     {Label: "Login shell", Help: "Default shell path (loginShell)"},
	"homeDirectory":  {Label: "Home directory", Help: "Home folder path (homeDirectory)"},
	"gidNumber":      {Label: "GID number", Help: "Numeric group ID (gidNumber)", Required: true},
	"description":    {Label: "Description", Help: "Optional group description"},
	"sambaAcctFlags": {Label: "Samba account flags", Help: "Samba flags such as [U          ] or [D          ] for disabled"},
}

func metaForField(spec config.FormFieldSpec) fieldMeta {
	if spec.Custom && spec.Label != "" {
		return fieldMeta{
			Label:    spec.Label,
			Help:     spec.Help,
			Secret:   spec.Secret,
			Required: spec.Required,
		}
	}
	if m, ok := builtinGlossary[strings.ToLower(spec.Attr)]; ok {
		m.Required = m.Required || spec.Required
		if spec.Secret {
			m.Secret = true
		}
		return m
	}
	if m, ok := builtinGlossary[spec.Attr]; ok {
		m.Required = m.Required || spec.Required
		return m
	}
	return fieldMeta{
		Label:    spec.Attr,
		Help:     fmt.Sprintf("LDAP attribute %s", spec.Attr),
		Required: spec.Required,
		Secret:   spec.Secret,
	}
}

func displayLabel(spec config.FormFieldSpec) string {
	m := metaForField(spec)
	label := m.Label
	if !strings.Contains(strings.ToLower(label), strings.ToLower(spec.Attr)) && spec.Attr != "password" {
		label = fmt.Sprintf("%s (%s)", m.Label, spec.Attr)
	}
	if m.Required {
		label += " (required)"
	}
	return label
}

func sambaFieldHelp(attr string) (label, help string) {
	switch strings.ToLower(attr) {
	case "sambasid":
		return "Samba SID", "Security identifier for the Samba account (sambaSID)"
	case "sambaacctflags":
		return "Samba account flags", "Account flags; [D          ] disables the account"
	case "sambantpassword":
		return "Samba NT password", "NT hash (never shown; Present/Missing only)"
	default:
		return attr, "Samba-related LDAP attribute"
	}
}

func sambaPresent(val string) string {
	if strings.TrimSpace(val) == "" {
		return "Missing"
	}
	if strings.EqualFold(val, "sambaNTPassword") || strings.Contains(strings.ToLower(val), "password") {
		return "Present"
	}
	return val
}
