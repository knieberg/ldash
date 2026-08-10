package embedassets

import _ "embed"

//go:embed config.example.yaml
var ConfigExample []byte

//go:embed templates/user_samba_posix.example.yaml
var UserSambaPosixExample []byte

//go:embed templates/user_samba_account.example.yaml
var UserSambaAccountExample []byte

//go:embed templates/group_posix.example.yaml
var GroupPosixExample []byte

//go:embed templates/group_of_names.example.yaml
var GroupOfNamesExample []byte

// InitFile describes a file written by config init.
type InitFile struct {
	Name    string
	Content []byte
	Mode    uint32
	SkipIf  bool // skip copy if destination already exists
}

// InitFiles returns all files for ldash config init.
func InitFiles() []InitFile {
	return []InitFile{
		{Name: "config.yaml", Content: ConfigExample, Mode: 0o600},
		{Name: "templates/user_samba_posix.yaml", Content: UserSambaPosixExample, Mode: 0o600, SkipIf: true},
		{Name: "templates/user_samba_account.example.yaml", Content: UserSambaAccountExample, Mode: 0o600, SkipIf: true},
		{Name: "templates/group_posix.yaml", Content: GroupPosixExample, Mode: 0o600, SkipIf: true},
		{Name: "templates/group_of_names.example.yaml", Content: GroupOfNamesExample, Mode: 0o600, SkipIf: true},
	}
}
