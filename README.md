# ldash — LDAP Admin Shell

[![License: GPL v3](https://img.shields.io/badge/License-GPLv3-blue.svg)](LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/knieberg/ldash)](https://goreportcard.com/report/github.com/knieberg/ldash)

**Terminal UI for OpenLDAP administration**

ldash is a terminal user interface (TUI) for managing users, groups, passwords, mail attributes, Samba-related LDAP fields, LDIF backup/restore, and integration hints for connecting LDAP to other services.

> [!WARNING]
> Always back up your LDAP directory (for example via LDIF export) before bulk changes.

## Links

- [Documentation](./docs/)
- [Installation](./docs/installation.md)
- [Configuration](./docs/configuration.md)
- [Terminal UI guide](./docs/tui.md)
- [Integration guide](./docs/integration.md)
- [Contributing](./CONTRIBUTING.md)

## Features

| Feature | Status |
| --- | --- |
| Connection profiles (plain / StartTLS / LDAPS) | MVP |
| Dashboard & connection test | MVP |
| Main menu navigation (1–7) | MVP |
| User list / search | MVP |
| User create / edit / delete (template-driven) | MVP |
| Admin password reset | MVP |
| Email (`mail`) management | MVP |
| Custom attributes (template YAML) | MVP |
| Group list / CRUD / members (template-driven) | MVP |
| LDIF export / import (TUI) | MVP |
| Samba status hub + per-user flags edit | MVP |
| Samba SID on user create | MVP |
| Integration guide (from local config) | MVP |
| Embedded `config init` for installed binaries | MVP |

## Quick start

### Requirements

- Go 1.24+ (toolchain 1.25.12 recommended for builds matching CI)
- OpenLDAP or compatible LDAP server
- LDAP bind DN with administrative write access

### Build from source

```bash
git clone https://github.com/knieberg/ldash.git
cd ldash
make setup-local   # optional: creates gitignored maintainer docs locally
make build
make install-local # installs to ~/.local/bin
```

### Configure

```bash
ldash config init
# Edit ~/.config/ldash/config.yaml — replace example.com values with your server
chmod 600 ~/.config/ldash/config.yaml
# Optional: echo 'your-bind-password' > ~/.config/ldash/credentials && chmod 600 ~/.config/ldash/credentials
ldash
```

Example values in the repository always use **`dc=example,dc=com`** and **`ldap.example.com`**.

### Keys (summary)

| Key | Action |
| --- | --- |
| `Esc` | One level back |
| `q` | Quit (main menu only) |
| `Ctrl+C` | Quit anytime |
| `?` | Help overlay |
| `1`–`7` | Open main menu item |
| `r` | Refresh / test connection |

Footer and help show **labeled** actions (for example `c create`, `e edit`) instead of bare letters.

Full key reference and screenshots: [Terminal UI guide](./docs/tui.md).

## Supported backends

- OpenLDAP (primary target)
- Generic LDAP with POSIX / inetOrgPerson schemas
- Samba schema (`sambaSamAccount`) when configured locally

## Configuration layout

Runtime configuration lives in **`~/.config/ldash/`** (never committed):

```
~/.config/ldash/
├── config.yaml
├── credentials          # optional, mode 0600
├── integration.yaml     # optional OIDC / self-service hints
└── templates/
    ├── user_samba_posix.yaml   # active user template
    ├── group_posix.yaml        # active group template
    └── *.example.yaml          # reference copies from config init
```

See [configuration.md](./docs/configuration.md).

## Security

- Do not commit credentials or environment-specific LDAP data.
- Keep `~/.config/ldash/` at mode `0700` and credential files at `0600`.
- ldash does not require root; it needs LDAP admin bind privileges only.
- LDIF export redacts password hashes; import skips password attributes (same policy as user create).

## License

This project is licensed under the [GNU General Public License v3.0](LICENSE).

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md). Bug reports and pull requests are welcome.
