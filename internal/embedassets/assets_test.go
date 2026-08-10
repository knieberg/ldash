package embedassets

import (
	"slices"
	"testing"
)

func TestInitFilesList(t *testing.T) {
	files := InitFiles()
	names := make([]string, len(files))
	for i, f := range files {
		names[i] = f.Name
		if len(f.Content) == 0 {
			t.Fatalf("empty content for %s", f.Name)
		}
	}
	want := []string{
		"config.yaml",
		"templates/user_samba_posix.yaml",
		"templates/user_samba_account.example.yaml",
		"templates/group_posix.yaml",
		"templates/group_of_names.example.yaml",
	}
	for _, w := range want {
		if !slices.Contains(names, w) {
			t.Fatalf("missing init file %q; got %v", w, names)
		}
	}
}
