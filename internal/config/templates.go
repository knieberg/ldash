package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/knieberg/ldash/internal/embedassets"
)

// CustomAttribute is an extra LDAP attribute declared in a template.
type CustomAttribute struct {
	Name     string `yaml:"name"`
	Label    string `yaml:"label"`
	Help     string `yaml:"help"`
	Required bool   `yaml:"required"`
}

// GroupTemplate describes object classes and defaults for group creation.
type GroupTemplate struct {
	Name               string            `yaml:"name"`
	Description        string            `yaml:"description"`
	ObjectClasses      []string          `yaml:"object_classes"`
	MemberAttribute    string            `yaml:"member_attribute"`
	Defaults           map[string]string `yaml:"defaults"`
	RequiredAttributes []string          `yaml:"required_attributes"`
	OptionalAttributes []string          `yaml:"optional_attributes"`
	CustomAttributes   []CustomAttribute `yaml:"custom_attributes"`
}

// UserTemplate describes object classes and defaults for user creation.
type UserTemplate struct {
	Name               string            `yaml:"name"`
	Description        string            `yaml:"description"`
	ObjectClasses      []string          `yaml:"object_classes"`
	Defaults           map[string]string `yaml:"defaults"`
	RequiredAttributes []string          `yaml:"required_attributes"`
	OptionalAttributes []string          `yaml:"optional_attributes"`
	CustomAttributes   []CustomAttribute `yaml:"custom_attributes"`
	CreatePrimaryGroup bool              `yaml:"create_primary_group"`
}

func validateTemplateAttrs(required, optional []string, custom []CustomAttribute) error {
	seen := map[string]bool{}
	for _, a := range required {
		k := strings.ToLower(strings.TrimSpace(a))
		if k == "" {
			return fmt.Errorf("empty required attribute name")
		}
		if seen[k] {
			return fmt.Errorf("duplicate attribute %q in required_attributes", a)
		}
		seen[k] = true
	}
	for _, a := range optional {
		k := strings.ToLower(strings.TrimSpace(a))
		if k == "" {
			return fmt.Errorf("empty optional attribute name")
		}
		if seen[k] {
			return fmt.Errorf("duplicate attribute %q in optional_attributes", a)
		}
		seen[k] = true
	}
	for _, c := range custom {
		k := strings.ToLower(strings.TrimSpace(c.Name))
		if k == "" {
			return fmt.Errorf("custom attribute missing name")
		}
		if seen[k] {
			return fmt.Errorf("attribute %q appears in both built-in lists and custom_attributes", c.Name)
		}
		seen[k] = true
	}
	return nil
}

func (t *UserTemplate) Validate() error {
	if len(t.ObjectClasses) == 0 {
		return fmt.Errorf("user template has no object_classes")
	}
	if t.Defaults == nil {
		t.Defaults = map[string]string{}
	}
	return validateTemplateAttrs(t.RequiredAttributes, t.OptionalAttributes, t.CustomAttributes)
}

func (t *GroupTemplate) Validate() error {
	if len(t.ObjectClasses) == 0 {
		return fmt.Errorf("group template has no object_classes")
	}
	if t.MemberAttribute == "" {
		t.MemberAttribute = "memberUid"
	}
	if t.Defaults == nil {
		t.Defaults = map[string]string{}
	}
	return validateTemplateAttrs(t.RequiredAttributes, t.OptionalAttributes, t.CustomAttributes)
}

func (t *UserTemplate) HasSambaAccount() bool {
	for _, oc := range t.ObjectClasses {
		if strings.EqualFold(oc, "sambaSamAccount") {
			return true
		}
	}
	return false
}

// AllFormFields returns built-in + custom attribute names for create/edit forms.
func (t *UserTemplate) AllFormFields(includePassword bool) []FormFieldSpec {
	fields := make([]FormFieldSpec, 0, len(t.RequiredAttributes)+len(t.OptionalAttributes)+len(t.CustomAttributes)+1)
	add := func(name string, required bool) {
		for _, c := range t.CustomAttributes {
			if strings.EqualFold(c.Name, name) {
				fields = append(fields, FormFieldSpec{
					Attr:     c.Name,
					Required: c.Required || required,
					Custom:   true,
					Label:    c.Label,
					Help:     c.Help,
				})
				return
			}
		}
		fields = append(fields, FormFieldSpec{
			Attr:     name,
			Required: required,
		})
	}
	for _, a := range t.RequiredAttributes {
		add(a, true)
	}
	for _, a := range t.OptionalAttributes {
		add(a, false)
	}
	for _, c := range t.CustomAttributes {
		found := false
		for _, a := range append(t.RequiredAttributes, t.OptionalAttributes...) {
			if strings.EqualFold(a, c.Name) {
				found = true
				break
			}
		}
		if !found {
			fields = append(fields, FormFieldSpec{
				Attr:     c.Name,
				Required: c.Required,
				Custom:   true,
				Label:    c.Label,
				Help:     c.Help,
			})
		}
	}
	if includePassword {
		fields = append(fields, FormFieldSpec{Attr: "password", Required: false, Secret: true})
	}
	return fields
}

func (t *GroupTemplate) AllFormFields() []FormFieldSpec {
	fields := make([]FormFieldSpec, 0, len(t.RequiredAttributes)+len(t.OptionalAttributes)+len(t.CustomAttributes))
	add := func(name string, required bool) {
		for _, c := range t.CustomAttributes {
			if strings.EqualFold(c.Name, name) {
				fields = append(fields, FormFieldSpec{
					Attr:     c.Name,
					Required: c.Required || required,
					Custom:   true,
					Label:    c.Label,
					Help:     c.Help,
				})
				return
			}
		}
		fields = append(fields, FormFieldSpec{
			Attr:     name,
			Required: required,
		})
	}
	for _, a := range t.RequiredAttributes {
		add(a, true)
	}
	for _, a := range t.OptionalAttributes {
		add(a, false)
	}
	for _, c := range t.CustomAttributes {
		found := false
		for _, a := range append(t.RequiredAttributes, t.OptionalAttributes...) {
			if strings.EqualFold(a, c.Name) {
				found = true
				break
			}
		}
		if !found {
			fields = append(fields, FormFieldSpec{
				Attr:     c.Name,
				Required: c.Required,
				Custom:   true,
				Label:    c.Label,
				Help:     c.Help,
			})
		}
	}
	return fields
}

// FormFieldSpec describes one form field derived from a template.
type FormFieldSpec struct {
	Attr     string
	Required bool
	Custom   bool
	Label    string
	Help     string
	Secret   bool
}

func loadYAML(path string, dest any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read template %s: %w", path, err)
	}
	if err := yaml.Unmarshal(data, dest); err != nil {
		return fmt.Errorf("parse template %s: %w", path, err)
	}
	return nil
}

func LoadUserTemplate(cfg *Config) (*UserTemplate, error) {
	dir, err := cfg.ResolvedTemplatesDir()
	if err != nil {
		return nil, err
	}
	path := filepath.Join(dir, "user_samba_posix.yaml")
	var tpl UserTemplate
	if err := loadYAML(path, &tpl); err != nil {
		return nil, err
	}
	if err := tpl.Validate(); err != nil {
		return nil, fmt.Errorf("user template: %w", err)
	}
	return &tpl, nil
}

func LoadGroupTemplate(cfg *Config) (*GroupTemplate, error) {
	dir, err := cfg.ResolvedTemplatesDir()
	if err != nil {
		return nil, err
	}
	path := filepath.Join(dir, "group_posix.yaml")
	var tpl GroupTemplate
	if err := loadYAML(path, &tpl); err != nil {
		return nil, err
	}
	if err := tpl.Validate(); err != nil {
		return nil, fmt.Errorf("group template: %w", err)
	}
	return &tpl, nil
}

// InitResult reports what InitFromEmbedded wrote.
type InitResult struct {
	ConfigPath string
	Added      []string // paths relative to config dir
}

// InitFromEmbedded writes embedded config and templates into the user config directory.
// If config.yaml already exists it is left unchanged; missing template files are still added (SkipIf files are never overwritten).
func InitFromEmbedded() (InitResult, error) {
	cfgDir, err := DefaultConfigDir()
	if err != nil {
		return InitResult{}, err
	}
	if err := os.MkdirAll(cfgDir, 0o700); err != nil {
		return InitResult{}, fmt.Errorf("create config dir: %w", err)
	}
	dest := filepath.Join(cfgDir, ConfigFile)
	configExists := false
	if _, err := os.Stat(dest); err == nil {
		configExists = true
	}
	var added []string
	for _, f := range embedassets.InitFiles() {
		if configExists && f.Name == ConfigFile {
			continue
		}
		target := filepath.Join(cfgDir, f.Name)
		if f.SkipIf {
			if _, err := os.Stat(target); err == nil {
				continue
			}
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o700); err != nil {
			return InitResult{ConfigPath: dest}, fmt.Errorf("create dir for %s: %w", f.Name, err)
		}
		if err := os.WriteFile(target, f.Content, os.FileMode(f.Mode)); err != nil {
			return InitResult{ConfigPath: dest}, fmt.Errorf("write %s: %w", f.Name, err)
		}
		added = append(added, f.Name)
	}
	if configExists && len(added) == 0 {
		return InitResult{ConfigPath: dest}, fmt.Errorf("config already exists at %s", dest)
	}
	return InitResult{ConfigPath: dest, Added: added}, nil
}

// InitFromExample copies example files from disk paths (development fallback).
func InitFromExample(examplePath, templateExamplePath string) (string, error) {
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
	if templateExamplePath != "" {
		tplDest := filepath.Join(templatesDir, "user_samba_posix.yaml")
		if _, err := os.Stat(tplDest); os.IsNotExist(err) {
			if err := copyFile(templateExamplePath, tplDest, 0o600); err != nil {
				return dest, fmt.Errorf("copy user template: %w", err)
			}
		}
	}
	return dest, nil
}
