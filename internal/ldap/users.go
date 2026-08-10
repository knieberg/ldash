package ldap

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/go-ldap/ldap/v3"

	"github.com/knieberg/ldash/internal/config"
)

// User is a directory person entry used by the TUI.
type User struct {
	DN            string
	UID           string
	CN            string
	SN            string
	GivenName     string
	Mail          string
	UIDNumber     int
	GIDNumber     int
	HomeDirectory string
	LoginShell    string
	Gecos         string
	SambaSID      string
	SambaFlags    string
}

// GroupRef is a group that references a user.
type GroupRef struct {
	DN   string
	CN   string
	Attr string
}

// ListUsers returns users under the People OU matching the configured list filter.
func (c *Client) ListUsers(filterSubstring string) ([]User, error) {
	if err := c.requireBound(); err != nil {
		return nil, err
	}
	filter := c.cfg.Search.ListUsersFilter
	if filterSubstring != "" {
		esc := ldap.EscapeFilter(filterSubstring)
		filter = fmt.Sprintf("(&%s(|(uid=*%s*)(cn=*%s*)(mail=*%s*)))", filter, esc, esc, esc)
	}
	req := ldap.NewSearchRequest(
		c.cfg.PeopleDN(),
		ldap.ScopeWholeSubtree,
		ldap.NeverDerefAliases,
		0, 0, false,
		filter,
		[]string{"uid", "cn", "sn", "givenName", "mail", "uidNumber", "gidNumber", "homeDirectory", "loginShell", "gecos", "sambaSID", "sambaAcctFlags"},
		nil,
	)
	res, err := c.conn.Search(req)
	if err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}
	users := make([]User, 0, len(res.Entries))
	for _, e := range res.Entries {
		users = append(users, entryToUser(e))
	}
	sort.Slice(users, func(i, j int) bool { return users[i].UID < users[j].UID })
	return users, nil
}

// GetUser loads a single user by uid.
func (c *Client) GetUser(uid string) (*User, error) {
	if err := c.requireBound(); err != nil {
		return nil, err
	}
	esc := ldap.EscapeFilter(uid)
	filter := strings.Replace(c.cfg.Search.UserFilter, "%s", esc, 1)
	if strings.Count(c.cfg.Search.UserFilter, "%s") != 1 {
		filter = "(&(|(objectClass=inetOrgPerson)(objectClass=posixAccount))(uid=" + esc + "))"
	}
	req := ldap.NewSearchRequest(
		c.cfg.PeopleDN(),
		ldap.ScopeWholeSubtree,
		ldap.NeverDerefAliases,
		1, 0, false,
		filter,
		[]string{"uid", "cn", "sn", "givenName", "mail", "uidNumber", "gidNumber", "homeDirectory", "loginShell", "gecos", "sambaSID", "sambaAcctFlags"},
		nil,
	)
	res, err := c.conn.Search(req)
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}
	if len(res.Entries) == 0 {
		return nil, fmt.Errorf("user %q not found", uid)
	}
	u := entryToUser(res.Entries[0])
	return &u, nil
}

// NextUIDNumber finds the next free uidNumber starting at cfg.IDRanges.UIDStart.
func (c *Client) NextUIDNumber() (int, error) {
	if err := c.requireBound(); err != nil {
		return 0, err
	}
	req := ldap.NewSearchRequest(
		c.cfg.PeopleDN(),
		ldap.ScopeWholeSubtree,
		ldap.NeverDerefAliases,
		0, 0, false,
		"(uidNumber=*)",
		[]string{"uidNumber"},
		nil,
	)
	res, err := c.conn.Search(req)
	if err != nil {
		return 0, fmt.Errorf("search uidNumber: %w", err)
	}
	used := map[int]bool{}
	max := c.cfg.IDRanges.UIDStart - 1
	for _, e := range res.Entries {
		n, err := strconv.Atoi(e.GetAttributeValue("uidNumber"))
		if err != nil {
			continue
		}
		used[n] = true
		if n > max {
			max = n
		}
	}
	candidate := max + 1
	if candidate < c.cfg.IDRanges.UIDStart {
		candidate = c.cfg.IDRanges.UIDStart
	}
	for used[candidate] {
		candidate++
	}
	return candidate, nil
}

// CreateUserInput holds fields for a new user entry.
type CreateUserInput struct {
	UID           string
	CN            string
	SN            string
	GivenName     string
	Mail          string
	Password      string
	LoginShell    string
	HomeDirectory string
	Gecos         string
	UIDNumber     int
	GIDNumber     int
	Extra         map[string]string
	Template      *config.UserTemplate
	GroupTemplate *config.GroupTemplate
}

// CreateUser adds a user (and optional primary group) from the template.
func (c *Client) CreateUser(in CreateUserInput) error {
	if err := c.requireBound(); err != nil {
		return err
	}
	if in.Template == nil {
		return fmt.Errorf("user template is required")
	}
	if in.UID == "" || in.CN == "" {
		return fmt.Errorf("uid and cn are required")
	}
	if templateNeedsSN(in.Template) && in.SN == "" {
		return fmt.Errorf("sn is required for this user template")
	}
	if in.UIDNumber == 0 {
		n, err := c.NextUIDNumber()
		if err != nil {
			return err
		}
		in.UIDNumber = n
	}
	if in.GIDNumber == 0 {
		in.GIDNumber = in.UIDNumber
	}
	if in.LoginShell == "" {
		in.LoginShell = in.Template.Defaults["login_shell"]
		if in.LoginShell == "" {
			in.LoginShell = "/bin/bash"
		}
	}
	if in.HomeDirectory == "" {
		prefix := in.Template.Defaults["home_prefix"]
		if prefix == "" {
			prefix = "/home"
		}
		in.HomeDirectory = filepathJoin(prefix, in.UID)
	}
	if in.Gecos == "" {
		in.Gecos = in.CN
	}

	needSamba := false
	for _, oc := range in.Template.ObjectClasses {
		if strings.EqualFold(oc, "sambaSamAccount") {
			needSamba = true
			break
		}
	}
	var sambaSID string
	if needSamba {
		sid, err := c.cfg.SambaSID(in.UIDNumber)
		if err != nil {
			return err
		}
		sambaSID = sid
	}

	groupDN := fmt.Sprintf("cn=%s,%s", ldap.EscapeDN(in.UID), c.cfg.GroupsDN())
	createdGroup := false
	if in.Template.CreatePrimaryGroup {
		gtpl := in.GroupTemplate
		if gtpl == nil {
			return fmt.Errorf("group template is required when create_primary_group is true")
		}
		if _, err := c.CreatePrimaryGroup(in.UID, in.GIDNumber, gtpl); err != nil {
			return err
		}
		createdGroup = true
	}

	dn := fmt.Sprintf("uid=%s,%s", ldap.EscapeDN(in.UID), c.cfg.PeopleDN())
	req := ldap.NewAddRequest(dn, nil)
	classes := append([]string{"top"}, in.Template.ObjectClasses...)
	req.Attribute("objectClass", classes)
	req.Attribute("uid", []string{in.UID})
	req.Attribute("cn", []string{in.CN})
	if in.SN != "" {
		req.Attribute("sn", []string{in.SN})
	}
	req.Attribute("uidNumber", []string{strconv.Itoa(in.UIDNumber)})
	req.Attribute("gidNumber", []string{strconv.Itoa(in.GIDNumber)})
	req.Attribute("homeDirectory", []string{in.HomeDirectory})
	req.Attribute("loginShell", []string{in.LoginShell})
	req.Attribute("gecos", []string{in.Gecos})
	if in.GivenName != "" {
		req.Attribute("givenName", []string{in.GivenName})
	}
	if in.Mail != "" {
		req.Attribute("mail", []string{in.Mail})
	}
	for k, v := range in.Extra {
		if strings.TrimSpace(v) != "" {
			req.Attribute(k, []string{v})
		}
	}
	if needSamba {
		req.Attribute("sambaSID", []string{sambaSID})
		req.Attribute("sambaAcctFlags", []string{"[U          ]"})
	}
	if err := c.conn.Add(req); err != nil {
		if createdGroup {
			_ = c.conn.Del(ldap.NewDelRequest(groupDN, nil))
		}
		return fmt.Errorf("create user: %w", err)
	}
	if in.Password != "" {
		if err := c.SetPassword(dn, in.Password); err != nil {
			if delErr := c.conn.Del(ldap.NewDelRequest(dn, nil)); delErr != nil {
				return fmt.Errorf("set password: %w; rollback delete user: %v", err, delErr)
			}
			if createdGroup {
				if delErr := c.conn.Del(ldap.NewDelRequest(groupDN, nil)); delErr != nil {
					return fmt.Errorf("set password: %w; rollback delete group: %v", err, delErr)
				}
			}
			return fmt.Errorf("set password: %w", err)
		}
	}
	return nil
}

func templateNeedsSN(tpl *config.UserTemplate) bool {
	for _, oc := range tpl.ObjectClasses {
		switch strings.ToLower(oc) {
		case "inetorgperson", "person", "organizationalperson":
			return true
		}
	}
	for _, attr := range tpl.RequiredAttributes {
		if strings.EqualFold(attr, "sn") {
			return true
		}
	}
	return false
}

// UpdateUserInput holds editable user attributes.
type UpdateUserInput struct {
	DN            string
	CN            string
	SN            string
	GivenName     string
	Mail          string
	Gecos         string
	LoginShell    string
	HomeDirectory string
	Extra         map[string]string
	ClearExtra    []string
}

// UpdateUser replaces common attributes on an existing entry.
func (c *Client) UpdateUser(in UpdateUserInput) error {
	if err := c.requireBound(); err != nil {
		return err
	}
	if in.DN == "" {
		return fmt.Errorf("dn is required")
	}
	if in.CN == "" || in.SN == "" {
		return fmt.Errorf("cn and sn are required")
	}
	mod := ldap.NewModifyRequest(in.DN, nil)
	mod.Replace("cn", []string{in.CN})
	mod.Replace("sn", []string{in.SN})
	if in.GivenName == "" {
		mod.Delete("givenName", nil)
	} else {
		mod.Replace("givenName", []string{in.GivenName})
	}
	if in.Mail == "" {
		mod.Delete("mail", nil)
	} else {
		mod.Replace("mail", []string{in.Mail})
	}
	if in.Gecos != "" {
		mod.Replace("gecos", []string{in.Gecos})
	}
	if in.LoginShell != "" {
		mod.Replace("loginShell", []string{in.LoginShell})
	}
	if in.HomeDirectory != "" {
		mod.Replace("homeDirectory", []string{in.HomeDirectory})
	}
	for _, attr := range in.ClearExtra {
		mod.Delete(attr, nil)
	}
	for k, v := range in.Extra {
		if strings.TrimSpace(v) == "" {
			mod.Delete(k, nil)
		} else {
			mod.Replace(k, []string{v})
		}
	}
	if err := c.conn.Modify(mod); err != nil {
		return fmt.Errorf("modify user: %w", err)
	}
	return nil
}

// SetMail sets, replaces, or clears the mail attribute.
func (c *Client) SetMail(dn, mail string) error {
	if err := c.requireBound(); err != nil {
		return err
	}
	mod := ldap.NewModifyRequest(dn, nil)
	if mail == "" {
		mod.Delete("mail", nil)
	} else {
		mod.Replace("mail", []string{mail})
	}
	if err := c.conn.Modify(mod); err != nil {
		return fmt.Errorf("set mail: %w", err)
	}
	return nil
}

// SetPassword changes the password via the Password Modify extended operation only.
func (c *Client) SetPassword(dn, newPassword string) error {
	if err := c.requireBound(); err != nil {
		return err
	}
	passReq := ldap.NewPasswordModifyRequest(dn, "", newPassword)
	if _, err := c.conn.PasswordModify(passReq); err != nil {
		return fmt.Errorf("password modify: %w (server must support Password Modify extended operation)", err)
	}
	return nil
}

// GroupsForUser finds groups that include the user via member or memberUid.
func (c *Client) GroupsForUser(uid, userDN string) ([]GroupRef, error) {
	if err := c.requireBound(); err != nil {
		return nil, err
	}
	filter := fmt.Sprintf("(|(&(objectClass=posixGroup)(memberUid=%s))(member=%s))",
		ldap.EscapeFilter(uid), ldap.EscapeFilter(userDN))
	req := ldap.NewSearchRequest(
		c.cfg.GroupsDN(),
		ldap.ScopeWholeSubtree,
		ldap.NeverDerefAliases,
		0, 0, false,
		filter,
		[]string{"cn", "memberUid", "member"},
		nil,
	)
	res, err := c.conn.Search(req)
	if err != nil {
		return nil, fmt.Errorf("search groups: %w", err)
	}
	out := make([]GroupRef, 0, len(res.Entries))
	for _, e := range res.Entries {
		attr := "memberUid"
		if len(e.GetAttributeValues("member")) > 0 {
			attr = "member"
		}
		out = append(out, GroupRef{DN: e.DN, CN: e.GetAttributeValue("cn"), Attr: attr})
	}
	return out, nil
}

// DeleteUser removes a user entry and cleans up group memberships.
func (c *Client) DeleteUser(uid, dn string) (int, error) {
	if err := c.requireBound(); err != nil {
		return 0, err
	}
	removed := 0
	groups, err := c.GroupsForUser(uid, dn)
	if err != nil {
		return 0, err
	}
	for _, g := range groups {
		mod := ldap.NewModifyRequest(g.DN, nil)
		if g.Attr == "memberUid" {
			mod.Delete("memberUid", []string{uid})
		} else {
			mod.Delete("member", []string{dn})
		}
		if err := c.conn.Modify(mod); err != nil {
			return removed, fmt.Errorf("remove membership from %s: %w", g.CN, err)
		}
		removed++
	}
	primaryDN := fmt.Sprintf("cn=%s,%s", ldap.EscapeDN(uid), c.cfg.GroupsDN())
	if g, err := c.GetGroup(uid); err == nil && g.MemberCount <= 1 {
		if err := c.conn.Del(ldap.NewDelRequest(primaryDN, nil)); err != nil && !ldap.IsErrorWithCode(err, ldap.LDAPResultNoSuchObject) {
			return removed, fmt.Errorf("delete primary group: %w", err)
		}
	}
	if err := c.conn.Del(ldap.NewDelRequest(dn, nil)); err != nil {
		return removed, fmt.Errorf("delete user: %w", err)
	}
	return removed, nil
}

// SetSambaFlags updates sambaAcctFlags on a user entry.
func (c *Client) SetSambaFlags(dn, flags string) error {
	if err := c.requireBound(); err != nil {
		return err
	}
	if !strings.HasPrefix(flags, "[") || !strings.HasSuffix(flags, "]") {
		return fmt.Errorf("sambaAcctFlags must look like [U          ]")
	}
	mod := ldap.NewModifyRequest(dn, nil)
	mod.Replace("sambaAcctFlags", []string{flags})
	if err := c.conn.Modify(mod); err != nil {
		return fmt.Errorf("set sambaAcctFlags: %w", err)
	}
	return nil
}

// HasSambaPassword reports whether sambaNTPassword is present (without reading the hash).
func (c *Client) HasSambaPassword(dn string) (bool, error) {
	if err := c.requireBound(); err != nil {
		return false, err
	}
	req := ldap.NewSearchRequest(dn, ldap.ScopeBaseObject, ldap.NeverDerefAliases, 1, 0, false, "(objectClass=*)", []string{"sambaNTPassword"}, nil)
	res, err := c.conn.Search(req)
	if err != nil {
		return false, err
	}
	if len(res.Entries) == 0 {
		return false, nil
	}
	return res.Entries[0].GetAttributeValue("sambaNTPassword") != "", nil
}

// GetEntryAttrs reads attribute values from a single LDAP entry.
func (c *Client) GetEntryAttrs(dn string, attrs []string) (map[string]string, error) {
	if err := c.requireBound(); err != nil {
		return nil, err
	}
	if len(attrs) == 0 {
		return map[string]string{}, nil
	}
	req := ldap.NewSearchRequest(dn, ldap.ScopeBaseObject, ldap.NeverDerefAliases, 1, 0, false, "(objectClass=*)", attrs, nil)
	res, err := c.conn.Search(req)
	if err != nil {
		return nil, fmt.Errorf("read entry attrs: %w", err)
	}
	if len(res.Entries) == 0 {
		return nil, fmt.Errorf("entry not found")
	}
	e := res.Entries[0]
	out := make(map[string]string, len(attrs))
	for _, a := range attrs {
		out[a] = e.GetAttributeValue(a)
	}
	return out, nil
}

func (c *Client) requireBound() error {
	if c.conn == nil || !c.bound {
		return fmt.Errorf("not connected")
	}
	return nil
}

func entryToUser(e *ldap.Entry) User {
	uidNum, _ := strconv.Atoi(e.GetAttributeValue("uidNumber"))
	gidNum, _ := strconv.Atoi(e.GetAttributeValue("gidNumber"))
	return User{
		DN:            e.DN,
		UID:           e.GetAttributeValue("uid"),
		CN:            e.GetAttributeValue("cn"),
		SN:            e.GetAttributeValue("sn"),
		GivenName:     e.GetAttributeValue("givenName"),
		Mail:          e.GetAttributeValue("mail"),
		UIDNumber:     uidNum,
		GIDNumber:     gidNum,
		HomeDirectory: e.GetAttributeValue("homeDirectory"),
		LoginShell:    e.GetAttributeValue("loginShell"),
		Gecos:         e.GetAttributeValue("gecos"),
		SambaSID:      e.GetAttributeValue("sambaSID"),
		SambaFlags:    e.GetAttributeValue("sambaAcctFlags"),
	}
}

func filepathJoin(a, b string) string {
	return strings.TrimRight(a, "/") + "/" + strings.TrimLeft(b, "/")
}
