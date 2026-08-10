package ldif

import (
	"strings"
	"testing"

	"github.com/go-ldap/ldap/v3"
)

func TestEntryToLDIFRedactsPasswords(t *testing.T) {
	e := &ldap.Entry{
		DN: "uid=alice,ou=People,dc=example,dc=com",
		Attributes: []*ldap.EntryAttribute{
			{Name: "uid", Values: []string{"alice"}},
			{Name: "userPassword", Values: []string{"{SSHA}secret"}},
			{Name: "sambaNTPassword", Values: []string{"deadbeef"}},
		},
	}
	out := entryToLDIF(e)
	if strings.Contains(out, "secret") || strings.Contains(out, "deadbeef") {
		t.Fatalf("expected redacted LDIF, got:\n%s", out)
	}
	if !strings.Contains(out, "uid: alice") {
		t.Fatalf("expected uid preserved, got:\n%s", out)
	}
}

func TestAddHasSensitive(t *testing.T) {
	req := ldap.NewAddRequest("uid=alice,ou=People,dc=example,dc=com", nil)
	req.Attribute("uid", []string{"alice"})
	if addHasSensitive(req) {
		t.Fatal("uid-only add should not be sensitive")
	}
	req.Attribute("userPassword", []string{"x"})
	if !addHasSensitive(req) {
		t.Fatal("userPassword add should be sensitive")
	}
}

func TestModifyHasSensitive(t *testing.T) {
	req := ldap.NewModifyRequest("uid=alice,ou=People,dc=example,dc=com", nil)
	req.Replace("mail", []string{"alice@example.com"})
	if modifyHasSensitive(req) {
		t.Fatal("mail modify should not be sensitive")
	}
	req2 := ldap.NewModifyRequest("uid=alice,ou=People,dc=example,dc=com", nil)
	req2.Replace("sambaNTPassword", []string{"x"})
	if !modifyHasSensitive(req2) {
		t.Fatal("sambaNTPassword modify should be sensitive")
	}
}
