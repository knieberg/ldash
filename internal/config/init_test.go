package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInitFromEmbeddedAddsMissingTemplates(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	cfgDir := filepath.Join(home, ConfigDir)
	if err := os.MkdirAll(filepath.Join(cfgDir, "templates"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cfgDir, ConfigFile), []byte("base_dn: dc=test,dc=com\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cfgDir, "templates", "user_samba_posix.yaml"), []byte("name: existing\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	res, err := InitFromEmbedded()
	if err != nil {
		t.Fatalf("InitFromEmbedded: %v", err)
	}
	if res.ConfigPath != filepath.Join(cfgDir, ConfigFile) {
		t.Fatalf("config path: got %q", res.ConfigPath)
	}
	foundGroup := false
	for _, name := range res.Added {
		if name == "templates/group_posix.yaml" {
			foundGroup = true
		}
		if name == ConfigFile {
			t.Fatal("should not rewrite existing config")
		}
		if name == "templates/user_samba_posix.yaml" {
			t.Fatal("should not overwrite existing user template")
		}
	}
	if !foundGroup {
		t.Fatalf("expected group_posix.yaml in Added, got %v", res.Added)
	}
	if _, err := os.Stat(filepath.Join(cfgDir, "templates", "group_posix.yaml")); err != nil {
		t.Fatalf("group template missing: %v", err)
	}
}

func TestInitFromEmbeddedConfigExistsNothingToAdd(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	cfgDir := filepath.Join(home, ConfigDir)
	if err := os.MkdirAll(filepath.Join(cfgDir, "templates"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cfgDir, ConfigFile), []byte("base_dn: dc=test,dc=com\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{
		"templates/user_samba_posix.yaml",
		"templates/group_posix.yaml",
		"templates/user_samba_account.example.yaml",
		"templates/group_of_names.example.yaml",
	} {
		if err := os.WriteFile(filepath.Join(cfgDir, name), []byte("stub\n"), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	if _, err := InitFromEmbedded(); err == nil {
		t.Fatal("expected error when config and all templates exist")
	}
}
