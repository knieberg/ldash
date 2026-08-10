package config

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	AppName    = "ldash"
	ConfigDir  = ".config/ldash"
	ConfigFile = "config.yaml"

	// DefaultListUsersFilter matches inetOrgPerson and/or posixAccount (typical POSIX/Samba).
	DefaultListUsersFilter = "(|(objectClass=inetOrgPerson)(objectClass=posixAccount))"
	// DefaultUserFilter looks up a single uid under the same objectClass union.
	DefaultUserFilter = "(&(|(objectClass=inetOrgPerson)(objectClass=posixAccount))(uid=%s))"
)

// Config holds LDAP connection and directory layout settings.
type Config struct {
	Server              ServerConfig  `yaml:"server"`
	BaseDN              string        `yaml:"base_dn"`
	BindDN              string        `yaml:"bind_dn"`
	OrganizationalUnits OUConfig      `yaml:"organizational_units"`
	IDRanges            IDRangeConfig `yaml:"id_ranges"`
	Search              SearchConfig  `yaml:"search"`
	Samba               SambaConfig   `yaml:"samba"`
	TemplatesDir        string        `yaml:"templates_dir"`
	CredentialFile      string        `yaml:"credential_file,omitempty"`
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

type SambaConfig struct {
	DomainSID string `yaml:"domain_sid"`
}

// Integration holds optional local integration hints (never shipped with site data).
type Integration struct {
	SelfServiceURL      string   `yaml:"self_service_url"`
	OIDCIssuer          string   `yaml:"oidc_issuer"`
	OIDCProvider        string   `yaml:"oidc_provider"`
	OnboardingChecklist []string `yaml:"onboarding_checklist"`
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
		c.Search.ListUsersFilter = DefaultListUsersFilter
	}
	if c.Search.UserFilter == "" {
		c.Search.UserFilter = DefaultUserFilter
	}
	if strings.Count(c.Search.UserFilter, "%s") != 1 {
		return fmt.Errorf("search.user_filter must contain exactly one %%s placeholder")
	}
	if c.IDRanges.UIDStart == 0 {
		c.IDRanges.UIDStart = 10000
	}
	if c.IDRanges.GIDStart == 0 {
		c.IDRanges.GIDStart = 10000
	}
	return nil
}

// ReadBindPasswordFile reads a credential file, trims whitespace, rejects empty
// content, and requires permissions no more open than 0600.
func ReadBindPasswordFile(path string) (string, error) {
	if err := CheckPermissions(path, 0o600); err != nil {
		return "", err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	pass := strings.TrimSpace(string(data))
	if pass == "" {
		return "", fmt.Errorf("credential file %s is empty", path)
	}
	return pass, nil
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

func (c *Config) ResolvedTemplatesDir() (string, error) {
	if c.TemplatesDir != "" {
		return ExpandPath(c.TemplatesDir)
	}
	dir, err := DefaultConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "templates"), nil
}

// SambaSID builds sambaSID from domain SID and uidNumber (rid = uid*2+1000).
func (c *Config) SambaSID(uidNumber int) (string, error) {
	sid := strings.TrimSpace(c.Samba.DomainSID)
	if sid == "" {
		return "", fmt.Errorf("samba.domain_sid is not set in config")
	}
	rid := uidNumber*2 + 1000
	return fmt.Sprintf("%s-%d", strings.TrimRight(sid, "-"), rid), nil
}

func LoadIntegration() (*Integration, error) {
	dir, err := DefaultConfigDir()
	if err != nil {
		return nil, err
	}
	path := filepath.Join(dir, "integration.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Integration{}, nil
		}
		return nil, err
	}
	var integ Integration
	if err := yaml.Unmarshal(data, &integ); err != nil {
		return nil, fmt.Errorf("parse integration.yaml: %w", err)
	}
	return &integ, nil
}

func copyFile(src, dst string, mode os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() { _ = in.Close() }()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		return err
	}
	return out.Close()
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
