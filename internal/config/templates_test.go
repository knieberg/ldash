package config

import "testing"

func TestUserTemplateDuplicateCustomAttr(t *testing.T) {
	tpl := &UserTemplate{
		ObjectClasses:      []string{"inetOrgPerson"},
		RequiredAttributes: []string{"uid", "cn"},
		OptionalAttributes: []string{"mail"},
		CustomAttributes: []CustomAttribute{
			{Name: "mail", Label: "Email duplicate", Help: "should fail"},
		},
	}
	if err := tpl.Validate(); err == nil {
		t.Fatal("expected duplicate attribute error")
	}
}

func TestUserTemplateAllFormFields(t *testing.T) {
	tpl := &UserTemplate{
		ObjectClasses:      []string{"inetOrgPerson"},
		RequiredAttributes: []string{"uid", "cn"},
		OptionalAttributes: []string{"mail"},
		CustomAttributes: []CustomAttribute{
			{Name: "title", Label: "Title", Help: "Job title"},
		},
	}
	fields := tpl.AllFormFields(true)
	if len(fields) < 4 {
		t.Fatalf("expected uid cn mail title + password, got %d fields", len(fields))
	}
}

func TestGroupTemplateValidateMemberAttrDefault(t *testing.T) {
	tpl := &GroupTemplate{ObjectClasses: []string{"posixGroup"}}
	if err := tpl.Validate(); err != nil {
		t.Fatal(err)
	}
	if tpl.MemberAttribute != "memberUid" {
		t.Fatalf("default member_attribute: got %q", tpl.MemberAttribute)
	}
}
