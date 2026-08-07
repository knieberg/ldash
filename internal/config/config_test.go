package config

import (
	"os"
	"path/filepath"
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
