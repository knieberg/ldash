package ldap

import (
	"testing"

	"github.com/knieberg/ldash/internal/config"
)

func TestFilepathJoin(t *testing.T) {
	got := filepathJoin("/home", "alice")
	if got != "/home/alice" {
		t.Fatalf("got %q", got)
	}
}

func TestEntryToUserMapping(t *testing.T) {
	// Ensure SambaSID helper used by create path stays consistent with config tests.
	cfg := &config.Config{Samba: config.SambaConfig{DomainSID: "S-1-5-21-9-8-7"}}
	sid, err := cfg.SambaSID(10001)
	if err != nil {
		t.Fatal(err)
	}
	if sid != "S-1-5-21-9-8-7-21002" {
		t.Fatalf("unexpected sid %s", sid)
	}
}
