package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadExample(t *testing.T) {
	path := filepath.Join("..", "..", "config.example.yaml")
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.BaseDN != "dc=example,dc=com" {
		t.Fatalf("unexpected base_dn: %s", cfg.BaseDN)
	}
	if cfg.PeopleDN() != "ou=People,dc=example,dc=com" {
		t.Fatalf("unexpected people dn: %s", cfg.PeopleDN())
	}
	if cfg.Samba.DomainSID == "" {
		t.Fatal("expected samba.domain_sid in example")
	}
	if !strings.Contains(cfg.Search.ListUsersFilter, "posixAccount") {
		t.Fatalf("example list_users_filter should include posixAccount: %q", cfg.Search.ListUsersFilter)
	}
}

func TestValidateTLSMode(t *testing.T) {
	cfg := &Config{
		Server: ServerConfig{URL: "ldap://ldap.example.com", TLSMode: "invalid"},
		BaseDN: "dc=example,dc=com",
		BindDN: "cn=admin,dc=example,dc=com",
	}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected validation error")
	}
}

func TestValidateUserFilter(t *testing.T) {
	base := Config{
		Server: ServerConfig{URL: "ldap://ldap.example.com", TLSMode: "plain"},
		BaseDN: "dc=example,dc=com",
		BindDN: "cn=admin,dc=example,dc=com",
	}

	t.Run("exactly one", func(t *testing.T) {
		cfg := base
		cfg.Search.UserFilter = "(&(objectClass=inetOrgPerson)(uid=%s))"
		if err := cfg.Validate(); err != nil {
			t.Fatalf("Validate: %v", err)
		}
	})

	t.Run("default when empty", func(t *testing.T) {
		cfg := base
		cfg.Search.UserFilter = ""
		cfg.Search.ListUsersFilter = ""
		if err := cfg.Validate(); err != nil {
			t.Fatalf("Validate: %v", err)
		}
		if cfg.Search.UserFilter != DefaultUserFilter {
			t.Fatalf("user_filter default: got %q want %q", cfg.Search.UserFilter, DefaultUserFilter)
		}
		if cfg.Search.ListUsersFilter != DefaultListUsersFilter {
			t.Fatalf("list_users_filter default: got %q want %q", cfg.Search.ListUsersFilter, DefaultListUsersFilter)
		}
		if !strings.Contains(cfg.Search.ListUsersFilter, "posixAccount") || !strings.Contains(cfg.Search.ListUsersFilter, "inetOrgPerson") {
			t.Fatalf("default list filter should cover posixAccount and inetOrgPerson: %q", cfg.Search.ListUsersFilter)
		}
	})

	t.Run("zero placeholders", func(t *testing.T) {
		cfg := base
		cfg.Search.UserFilter = "(objectClass=inetOrgPerson)"
		if err := cfg.Validate(); err == nil {
			t.Fatal("expected validation error")
		}
	})

	t.Run("two placeholders", func(t *testing.T) {
		cfg := base
		cfg.Search.UserFilter = "(&(uid=%s)(mail=%s))"
		if err := cfg.Validate(); err == nil {
			t.Fatal("expected validation error")
		}
	})
}

func TestReadBindPasswordFile(t *testing.T) {
	dir := t.TempDir()

	t.Run("trim and accept", func(t *testing.T) {
		path := filepath.Join(dir, "creds-ok")
		if err := os.WriteFile(path, []byte("  secret-pass\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		got, err := ReadBindPasswordFile(path)
		if err != nil {
			t.Fatalf("ReadBindPasswordFile: %v", err)
		}
		if got != "secret-pass" {
			t.Fatalf("got %q want secret-pass", got)
		}
	})

	t.Run("reject empty", func(t *testing.T) {
		path := filepath.Join(dir, "creds-empty")
		if err := os.WriteFile(path, []byte("  \n\t"), 0o600); err != nil {
			t.Fatal(err)
		}
		if _, err := ReadBindPasswordFile(path); err == nil {
			t.Fatal("expected error for empty credential file")
		}
	})

	t.Run("reject open permissions", func(t *testing.T) {
		path := filepath.Join(dir, "creds-open")
		if err := os.WriteFile(path, []byte("secret\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := ReadBindPasswordFile(path); err == nil {
			t.Fatal("expected permissions error")
		}
	})
}

func TestExpandPath(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	got, err := ExpandPath("~/foo")
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(home, "foo")
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestSambaSID(t *testing.T) {
	cfg := &Config{Samba: SambaConfig{DomainSID: "S-1-5-21-1-2-3"}}
	got, err := cfg.SambaSID(10000)
	if err != nil {
		t.Fatal(err)
	}
	want := "S-1-5-21-1-2-3-21000"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestSambaSIDMissing(t *testing.T) {
	cfg := &Config{}
	if _, err := cfg.SambaSID(10000); err == nil {
		t.Fatal("expected error")
	}
}
