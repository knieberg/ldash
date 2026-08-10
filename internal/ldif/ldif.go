package ldif

import (
	"fmt"
	"os"
	"strings"

	"github.com/go-ldap/ldap/v3"
	ldifpkg "github.com/go-ldap/ldif"

	"github.com/knieberg/ldash/internal/config"
	ldapclient "github.com/knieberg/ldash/internal/ldap"
)

var sensitiveAttrs = map[string]bool{
	"userpassword":     true,
	"sambantpassword":  true,
	"sambalmpassword":  true,
	"sambapwdlastset":  true,
}

// ApplyResult summarizes an LDIF import run.
type ApplyResult struct {
	Applied int
	Failed  int
	Skipped int
	Errors  []string
}

// Export writes entries under scope to path, redacting sensitive attributes.
func Export(client *ldapclient.Client, cfg *config.Config, scope, path string) (int, error) {
	bases := scopeBases(cfg, scope)
	if len(bases) == 0 {
		return 0, fmt.Errorf("invalid export scope %q", scope)
	}
	var entries []*ldap.Entry
	for _, base := range bases {
		req := ldap.NewSearchRequest(base, ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 0, 0, false, "(objectClass=*)", []string{"*"}, nil)
		res, err := client.RawSearch(req)
		if err != nil {
			return 0, err
		}
		entries = append(entries, res.Entries...)
	}
	if _, err := os.Stat(path); err == nil {
		return 0, fmt.Errorf("export file already exists: %s", path)
	}
	f, err := os.Create(path)
	if err != nil {
		return 0, err
	}
	defer func() { _ = f.Close() }()
	n := 0
	for _, e := range entries {
		block := entryToLDIF(e)
		if block == "" {
			continue
		}
		if _, err := f.WriteString(block + "\n"); err != nil {
			return n, err
		}
		n++
	}
	return n, nil
}

func scopeBases(cfg *config.Config, scope string) []string {
	switch strings.ToLower(scope) {
	case "people":
		return []string{cfg.PeopleDN()}
	case "groups":
		return []string{cfg.GroupsDN()}
	case "both":
		return []string{cfg.PeopleDN(), cfg.GroupsDN()}
	default:
		return nil
	}
}

func entryToLDIF(e *ldap.Entry) string {
	var b strings.Builder
	b.WriteString("dn: " + e.DN + "\n")
	for _, attr := range e.Attributes {
		if sensitiveAttrs[strings.ToLower(attr.Name)] {
			continue
		}
		for _, v := range attr.Values {
			b.WriteString(attr.Name + ": " + v + "\n")
		}
	}
	return b.String()
}

// PreviewImport counts LDIF records without applying.
func PreviewImport(path string) (int, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	list, err := ldifpkg.Parse(string(data))
	if err != nil {
		return 0, fmt.Errorf("parse ldif: %w", err)
	}
	return len(list.Entries), nil
}

// Import applies LDIF records; skips entries that set password attributes.
func Import(client *ldapclient.Client, path string) (*ApplyResult, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	list, err := ldifpkg.Parse(string(data))
	if err != nil {
		return nil, fmt.Errorf("parse ldif: %w", err)
	}
	res := &ApplyResult{}
	for _, rec := range list.Entries {
		if rec == nil {
			continue
		}
		switch {
		case rec.Add != nil:
			if addHasSensitive(rec.Add) {
				res.Skipped++
				res.Errors = append(res.Errors, fmt.Sprintf("skipped add %s: password attribute", rec.Add.DN))
				continue
			}
			if err := client.RawAdd(rec.Add); err != nil {
				res.Failed++
				res.Errors = append(res.Errors, fmt.Sprintf("add %s: %v", rec.Add.DN, err))
			} else {
				res.Applied++
			}
		case rec.Modify != nil:
			if modifyHasSensitive(rec.Modify) {
				res.Skipped++
				res.Errors = append(res.Errors, fmt.Sprintf("skipped modify %s: password attribute", rec.Modify.DN))
				continue
			}
			if err := client.RawModify(rec.Modify); err != nil {
				res.Failed++
				res.Errors = append(res.Errors, fmt.Sprintf("modify %s: %v", rec.Modify.DN, err))
			} else {
				res.Applied++
			}
		case rec.Del != nil:
			if err := client.RawDel(rec.Del); err != nil {
				res.Failed++
				res.Errors = append(res.Errors, fmt.Sprintf("delete %s: %v", rec.Del.DN, err))
			} else {
				res.Applied++
			}
		case rec.Entry != nil:
			if entryHasSensitive(rec.Entry) {
				res.Skipped++
				res.Errors = append(res.Errors, fmt.Sprintf("skipped entry %s: password attribute", rec.Entry.DN))
				continue
			}
			add := ldap.NewAddRequest(rec.Entry.DN, nil)
			for _, a := range rec.Entry.Attributes {
				add.Attribute(a.Name, a.Values)
			}
			if err := client.RawAdd(add); err != nil {
				res.Failed++
				res.Errors = append(res.Errors, fmt.Sprintf("add %s: %v", rec.Entry.DN, err))
			} else {
				res.Applied++
			}
		}
	}
	return res, nil
}

func addHasSensitive(req *ldap.AddRequest) bool {
	for _, a := range req.Attributes {
		if sensitiveAttrs[strings.ToLower(a.Type)] {
			return true
		}
	}
	return false
}

func modifyHasSensitive(req *ldap.ModifyRequest) bool {
	for _, ch := range req.Changes {
		if sensitiveAttrs[strings.ToLower(ch.Modification.Type)] {
			return true
		}
	}
	return false
}

func entryHasSensitive(e *ldap.Entry) bool {
	for _, a := range e.Attributes {
		if sensitiveAttrs[strings.ToLower(a.Name)] {
			return true
		}
	}
	return false
}
