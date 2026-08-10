package ldap

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/go-ldap/ldap/v3"

	"github.com/knieberg/ldash/internal/config"
)

// Group is a directory group entry used by the TUI.
type Group struct {
	DN              string
	CN              string
	GIDNumber       int
	Description     string
	MemberAttribute string
	Members         []string
	MemberCount     int
	Extra           map[string]string
}

// ListGroups returns groups under the Groups OU.
func (c *Client) ListGroups(filterSubstring string) ([]Group, error) {
	if err := c.requireBound(); err != nil {
		return nil, err
	}
	filter := c.cfg.Search.GroupFilter
	if filter == "" {
		filter = "(objectClass=posixGroup)"
	}
	if filterSubstring != "" {
		esc := ldap.EscapeFilter(filterSubstring)
		filter = fmt.Sprintf("(&%s(|(cn=*%s*)(description=*%s*)))", filter, esc, esc)
	}
	req := ldap.NewSearchRequest(
		c.cfg.GroupsDN(),
		ldap.ScopeWholeSubtree,
		ldap.NeverDerefAliases,
		0, 0, false,
		filter,
		[]string{"cn", "gidNumber", "description", "memberUid", "member", "objectClass"},
		nil,
	)
	res, err := c.conn.Search(req)
	if err != nil {
		return nil, fmt.Errorf("list groups: %w", err)
	}
	groups := make([]Group, 0, len(res.Entries))
	for _, e := range res.Entries {
		groups = append(groups, entryToGroup(e))
	}
	sort.Slice(groups, func(i, j int) bool { return groups[i].CN < groups[j].CN })
	return groups, nil
}

// GetGroup loads a group by cn.
func (c *Client) GetGroup(cn string) (*Group, error) {
	if err := c.requireBound(); err != nil {
		return nil, err
	}
	esc := ldap.EscapeFilter(cn)
	filter := c.cfg.Search.GroupFilter
	if filter == "" {
		filter = "(objectClass=posixGroup)"
	}
	filter = fmt.Sprintf("(&%s(cn=%s))", filter, esc)
	req := ldap.NewSearchRequest(
		c.cfg.GroupsDN(),
		ldap.ScopeWholeSubtree,
		ldap.NeverDerefAliases,
		1, 0, false,
		filter,
		[]string{"cn", "gidNumber", "description", "memberUid", "member", "objectClass"},
		nil,
	)
	res, err := c.conn.Search(req)
	if err != nil {
		return nil, fmt.Errorf("get group: %w", err)
	}
	if len(res.Entries) == 0 {
		return nil, fmt.Errorf("group %q not found", cn)
	}
	g := entryToGroup(res.Entries[0])
	return &g, nil
}

// NextGIDNumber finds the next free gidNumber starting at cfg.IDRanges.GIDStart.
func (c *Client) NextGIDNumber() (int, error) {
	if err := c.requireBound(); err != nil {
		return 0, err
	}
	filter := c.cfg.Search.GroupFilter
	if filter == "" {
		filter = "(objectClass=posixGroup)"
	}
	req := ldap.NewSearchRequest(
		c.cfg.GroupsDN(),
		ldap.ScopeWholeSubtree,
		ldap.NeverDerefAliases,
		0, 0, false,
		fmt.Sprintf("(&%s(gidNumber=*))", filter),
		[]string{"gidNumber"},
		nil,
	)
	res, err := c.conn.Search(req)
	if err != nil {
		return 0, fmt.Errorf("search gidNumber: %w", err)
	}
	used := map[int]bool{}
	max := c.cfg.IDRanges.GIDStart - 1
	for _, e := range res.Entries {
		n, err := strconv.Atoi(e.GetAttributeValue("gidNumber"))
		if err != nil {
			continue
		}
		used[n] = true
		if n > max {
			max = n
		}
	}
	candidate := max + 1
	if candidate < c.cfg.IDRanges.GIDStart {
		candidate = c.cfg.IDRanges.GIDStart
	}
	for used[candidate] {
		candidate++
	}
	return candidate, nil
}

// CreateGroupInput holds fields for a new group.
type CreateGroupInput struct {
	CN          string
	GIDNumber   int
	Description string
	Extra       map[string]string
	Template    *config.GroupTemplate
}

// CreateGroup adds a group from the template.
func (c *Client) CreateGroup(in CreateGroupInput) error {
	if err := c.requireBound(); err != nil {
		return err
	}
	if in.Template == nil {
		return fmt.Errorf("group template is required")
	}
	if in.CN == "" {
		return fmt.Errorf("cn is required")
	}
	if in.GIDNumber == 0 {
		n, err := c.NextGIDNumber()
		if err != nil {
			return err
		}
		in.GIDNumber = n
	}
	dn := fmt.Sprintf("cn=%s,%s", ldap.EscapeDN(in.CN), c.cfg.GroupsDN())
	req := ldap.NewAddRequest(dn, nil)
	classes := append([]string{"top"}, in.Template.ObjectClasses...)
	req.Attribute("objectClass", classes)
	req.Attribute("cn", []string{in.CN})
	if in.GIDNumber > 0 && hasObjectClass(in.Template.ObjectClasses, "posixGroup") {
		req.Attribute("gidNumber", []string{strconv.Itoa(in.GIDNumber)})
	}
	if in.Description != "" {
		req.Attribute("description", []string{in.Description})
	}
	for k, v := range in.Extra {
		if strings.TrimSpace(v) != "" {
			req.Attribute(k, []string{v})
		}
	}
	if err := c.conn.Add(req); err != nil {
		return fmt.Errorf("create group: %w", err)
	}
	return nil
}

// UpdateGroupInput holds editable group attributes.
type UpdateGroupInput struct {
	DN          string
	CN          string
	Description string
	Extra       map[string]string
	ClearExtra  []string
}

// UpdateGroup modifies group attributes.
func (c *Client) UpdateGroup(in UpdateGroupInput) error {
	if err := c.requireBound(); err != nil {
		return err
	}
	mod := ldap.NewModifyRequest(in.DN, nil)
	if in.Description == "" {
		mod.Delete("description", nil)
	} else {
		mod.Replace("description", []string{in.Description})
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
		return fmt.Errorf("modify group: %w", err)
	}
	return nil
}

// CountUsersWithGID returns users whose gidNumber matches.
func (c *Client) CountUsersWithGID(gid int) (int, error) {
	if err := c.requireBound(); err != nil {
		return 0, err
	}
	filter := fmt.Sprintf("(&%s(gidNumber=%d))", c.cfg.Search.ListUsersFilter, gid)
	req := ldap.NewSearchRequest(
		c.cfg.PeopleDN(),
		ldap.ScopeWholeSubtree,
		ldap.NeverDerefAliases,
		0, 0, false,
		filter,
		[]string{"uid"},
		nil,
	)
	res, err := c.conn.Search(req)
	if err != nil {
		return 0, fmt.Errorf("search users by gid: %w", err)
	}
	return len(res.Entries), nil
}

// DeleteGroup removes a group if no users use its gidNumber.
func (c *Client) DeleteGroup(dn string, gidNumber int) error {
	if err := c.requireBound(); err != nil {
		return err
	}
	if gidNumber > 0 {
		n, err := c.CountUsersWithGID(gidNumber)
		if err != nil {
			return err
		}
		if n > 0 {
			return fmt.Errorf("cannot delete group: %d user(s) still use gidNumber %d", n, gidNumber)
		}
	}
	if err := c.conn.Del(ldap.NewDelRequest(dn, nil)); err != nil {
		return fmt.Errorf("delete group: %w", err)
	}
	return nil
}

// AddGroupMember adds a user to a group.
func (c *Client) AddGroupMember(groupDN, memberAttr, memberValue string) error {
	if err := c.requireBound(); err != nil {
		return err
	}
	mod := ldap.NewModifyRequest(groupDN, nil)
	mod.Add(memberAttr, []string{memberValue})
	if err := c.conn.Modify(mod); err != nil {
		return fmt.Errorf("add member: %w", err)
	}
	return nil
}

// RemoveGroupMember removes a user from a group.
func (c *Client) RemoveGroupMember(groupDN, memberAttr, memberValue string) error {
	if err := c.requireBound(); err != nil {
		return err
	}
	mod := ldap.NewModifyRequest(groupDN, nil)
	mod.Delete(memberAttr, []string{memberValue})
	if err := c.conn.Modify(mod); err != nil {
		return fmt.Errorf("remove member: %w", err)
	}
	return nil
}

// CreatePrimaryGroup creates a primary group for a user using the group template.
func (c *Client) CreatePrimaryGroup(uid string, gidNumber int, tpl *config.GroupTemplate) (string, error) {
	if tpl == nil {
		return "", fmt.Errorf("group template is required")
	}
	groupDN := fmt.Sprintf("cn=%s,%s", ldap.EscapeDN(uid), c.cfg.GroupsDN())
	g := ldap.NewAddRequest(groupDN, nil)
	classes := append([]string{"top"}, tpl.ObjectClasses...)
	g.Attribute("objectClass", classes)
	g.Attribute("cn", []string{uid})
	if hasObjectClass(tpl.ObjectClasses, "posixGroup") {
		g.Attribute("gidNumber", []string{strconv.Itoa(gidNumber)})
	}
	memberAttr := tpl.MemberAttribute
	if memberAttr == "" {
		memberAttr = "memberUid"
	}
	if memberAttr == "memberUid" {
		g.Attribute("memberUid", []string{uid})
	}
	if err := c.conn.Add(g); err != nil {
		if !ldap.IsErrorWithCode(err, ldap.LDAPResultEntryAlreadyExists) {
			return "", fmt.Errorf("create primary group: %w", err)
		}
	}
	return groupDN, nil
}

func entryToGroup(e *ldap.Entry) Group {
	gid, _ := strconv.Atoi(e.GetAttributeValue("gidNumber"))
	attr := "memberUid"
	members := e.GetAttributeValues("memberUid")
	if len(e.GetAttributeValues("member")) > 0 {
		attr = "member"
		members = e.GetAttributeValues("member")
	}
	return Group{
		DN:              e.DN,
		CN:              e.GetAttributeValue("cn"),
		GIDNumber:       gid,
		Description:     e.GetAttributeValue("description"),
		MemberAttribute: attr,
		Members:         members,
		MemberCount:     len(members),
	}
}

func hasObjectClass(classes []string, want string) bool {
	for _, oc := range classes {
		if strings.EqualFold(oc, want) {
			return true
		}
	}
	return false
}
