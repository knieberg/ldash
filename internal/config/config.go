package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	AppName    = "ldash"
	ConfigDir  = ".config/ldash"
	ConfigFile = "config.yaml"
)

// Config holds LDAP connection and directory layout settings.
type Config struct {
	Server              ServerConfig       `yaml:"server"`
	BaseDN              string             `yaml:"base_dn"`
	BindDN              string             `yaml:"bind_dn"`
	OrganizationalUnits OUConfig           `yaml:"organizational_units"`
	IDRanges            IDRangeConfig      `yaml:"id_ranges"`
	Search              SearchConfig       `yaml:"search"`
	TemplatesDir        string             `yaml:"templates_dir"`
	CredentialFile      string             `yaml:"credential_file,omitempty"`
}

type ServerConfig struct {
	URL     string `yaml:"url"`
	TLSMode string `yaml:"tls_mode"`
}

type OUConfig struct {
	People string `yaml:"people"`
	Groups string `yaml:"groups"`
}

type IDRangeConfig struct {
	UIDStart int `yaml:"uid_start"`
	GIDStart int `yaml:"gid_start"`
}

type SearchConfig struct {
	UserFilter      string `yaml:"user_filter"`
	GroupFilter     string `yaml:"group_filter"`
	ListUsersFilter string `yaml:"list_users_filter"`
}

func DefaultConfigDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ConfigDir), nil
}

func DefaultConfigPath() (string, error) {
	dir, err := DefaultConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, ConfigFile), nil
}

func ExpandPath(path string) (string, error) {
	if path == "" {
		return "", nil
	}
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, path[2:]), nil
	}
	return path, nil
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func (c *Config) Validate() error {
	if c.Server.URL == "" {
		return fmt.Errorf("server.url is required")
	}
	if c.BaseDN == "" {
		return fmt.Errorf("base_dn is required")
	}
	if c.BindDN == "" {
		return fmt.Errorf("bind_dn is required")
	}
	mode := strings.ToLower(c.Server.TLSMode)
	if mode == "" {
		c.Server.TLSMode = "plain"
	} else if mode != "plain" && mode != "starttls" && mode != "ldaps" {
		return fmt.Errorf("server.tls_mode must be plain, starttls, or ldaps")
	}
	if c.Search.ListUsersFilter == "" {
		c.Search.ListUsersFilter = "(objectClass=inetOrgPerson)"
	}
	if c.IDRanges.UIDStart == 0 {
		c.IDRanges.UIDStart = 10000
	}
	if c.IDRanges.GIDStart == 0 {
		c.IDRanges.GIDStart = 10000
	}
	return nil
}

func (c *Config) PeopleDN() string {
	return fmt.Sprintf("%s,%s", c.OrganizationalUnits.People, c.BaseDN)
}

func (c *Config) GroupsDN() string {
	return fmt.Sprintf("%s,%s", c.OrganizationalUnits.Groups, c.BaseDN)
}

func (c *Config) CredentialPath() (string, error) {
	if c.CredentialFile != "" {
		return ExpandPath(c.CredentialFile)
	}
	dir, err := DefaultConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "credentials"), nil
}

// InitFromExample copies the example config into the user config directory.
func InitFromExample(examplePath string) (string, error) {
	cfgDir, err := DefaultConfigDir()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(cfgDir, 0o700); err != nil {
		return "", fmt.Errorf("create config dir: %w", err)
	}
	dest := filepath.Join(cfgDir, ConfigFile)
	if _, err := os.Stat(dest); err == nil {
		return dest, fmt.Errorf("config already exists at %s", dest)
	}
	data, err := os.ReadFile(examplePath)
	if err != nil {
		return "", fmt.Errorf("read example config: %w", err)
	}
	if err := os.WriteFile(dest, data, 0o600); err != nil {
		return "", fmt.Errorf("write config: %w", err)
	}
	templatesDir, err := ExpandPath("~/.config/ldash/templates")
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(templatesDir, 0o700); err != nil {
		return "", fmt.Errorf("create templates dir: %w", err)
	}
	return dest, nil
}

func CheckPermissions(path string, maxPerm os.FileMode) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if info.Mode().Perm()&^maxPerm != 0 {
		return fmt.Errorf("%s permissions %o are too open (max %o)", path, info.Mode().Perm(), maxPerm)
	}
	return nil
}
