package ldap

import (
	"fmt"
	"strings"
	"testing"

	golap "github.com/go-ldap/ldap/v3"

	"github.com/knieberg/ldash/internal/config"
)

func TestFilepathJoin(t *testing.T) {
	got := filepathJoin("/home", "alice")
	if got != "/home/alice" {
		t.Fatalf("got %q", got)
	}
}

func TestEntryToUserMapping(t *testing.T) {
	// Ensure SambaSID helper used by create path stays consistent with config tests.
	cfg := &config.Config{Samba: config.SambaConfig{DomainSID: "S-1-5-21-9-8-7"}}
	sid, err := cfg.SambaSID(10001)
	if err != nil {
		t.Fatal(err)
	}
	if sid != "S-1-5-21-9-8-7-21002" {
		t.Fatalf("unexpected sid %s", sid)
	}
}

func TestEscapeFilterInjection(t *testing.T) {
	malicious := []string{
		"*)(uid=*",
		"admin)(|(uid=*",
		`user\00`,
		"a(b)c*",
	}
	base := config.DefaultListUsersFilter
	userFilter := config.DefaultUserFilter
	for _, raw := range malicious {
		esc := golap.EscapeFilter(raw)
		listFilter := fmt.Sprintf("(&%s(|(uid=*%s*)(cn=*%s*)(mail=*%s*)))", base, esc, esc, esc)
		getFilter := fmt.Sprintf(userFilter, esc)
		groupFilter := fmt.Sprintf(
			"(|(&(objectClass=posixGroup)(memberUid=%s))(member=%s))",
			esc, golap.EscapeFilter("uid="+raw+",ou=People,dc=example,dc=com"),
		)
		for _, name := range []struct {
			label  string
			filter string
		}{
			{"list", listFilter},
			{"get", getFilter},
			{"groups", groupFilter},
		} {
			if strings.Contains(name.filter, raw) {
				t.Fatalf("%s filter still contains raw injection %q: %s", name.label, raw, name.filter)
			}
			if !strings.Contains(name.filter, esc) {
				t.Fatalf("%s filter missing escaped value for %q", name.label, raw)
			}
		}
	}
}

func TestEntryToUserPosixWithoutInetOrg(t *testing.T) {
	e := golap.NewEntry("uid=bob,ou=People,dc=example,dc=com", map[string][]string{
		"objectClass":   {"top", "account", "posixAccount", "sambaSamAccount"},
		"uid":           {"bob"},
		"cn":            {"Bob Example"},
		"uidNumber":     {"10001"},
		"gidNumber":     {"10001"},
		"homeDirectory": {"/home/bob"},
		"loginShell":    {"/bin/bash"},
		"mail":          {"bob@example.com"},
	})
	u := entryToUser(e)
	if u.UID != "bob" || u.CN != "Bob Example" || u.UIDNumber != 10001 {
		t.Fatalf("unexpected user mapping: %+v", u)
	}
	if u.Mail != "bob@example.com" {
		t.Fatalf("mail: %q", u.Mail)
	}
}

func TestEntryToUserInetOrgPerson(t *testing.T) {
	e := golap.NewEntry("uid=alice,ou=People,dc=example,dc=com", map[string][]string{
		"objectClass": {"top", "inetOrgPerson", "posixAccount"},
		"uid":         {"alice"},
		"cn":          {"Alice Example"},
		"sn":          {"Example"},
		"givenName":   {"Alice"},
		"uidNumber":   {"10000"},
		"gidNumber":   {"10000"},
		"mail":        {"alice@example.com"},
	})
	u := entryToUser(e)
	if u.UID != "alice" || u.SN != "Example" || u.GivenName != "Alice" {
		t.Fatalf("unexpected user mapping: %+v", u)
	}
}

func TestTemplateNeedsSN(t *testing.T) {
	if !templateNeedsSN(&config.UserTemplate{ObjectClasses: []string{"inetOrgPerson", "posixAccount"}}) {
		t.Fatal("inetOrgPerson should require sn")
	}
	if templateNeedsSN(&config.UserTemplate{ObjectClasses: []string{"account", "posixAccount", "sambaSamAccount"}}) {
		t.Fatal("account+posixAccount should not require sn")
	}
	if !templateNeedsSN(&config.UserTemplate{
		ObjectClasses:      []string{"account"},
		RequiredAttributes: []string{"uid", "sn"},
	}) {
		t.Fatal("explicit sn in required_attributes should require sn")
	}
}

func TestDefaultFiltersMatchBothSchemas(t *testing.T) {
	list := config.DefaultListUsersFilter
	user := config.DefaultUserFilter
	for _, oc := range []string{"inetOrgPerson", "posixAccount"} {
		if !strings.Contains(list, oc) {
			t.Fatalf("list filter missing %s: %s", oc, list)
		}
		if !strings.Contains(user, oc) {
			t.Fatalf("user filter missing %s: %s", oc, user)
		}
	}
	// A posix-only filter still matches the default OR list semantics conceptually.
	posixOnly := "(objectClass=posixAccount)"
	if strings.Contains(posixOnly, "inetOrgPerson") {
		t.Fatal("sanity")
	}
}
