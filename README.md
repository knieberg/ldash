# ldash — LDAP Admin Shell

[![License: GPL v3](https://img.shields.io/badge/License-GPLv3-blue.svg)](LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/ldash-sh/ldash)](https://goreportcard.com/report/github.com/ldash-sh/ldash)

**Terminal UI for OpenLDAP administration**

ldash is a terminal user interface (TUI) for managing users, groups, passwords, mail attributes, Samba-related LDAP fields, and integration hints for connecting LDAP to other services.

> [!WARNING]
> Always back up your LDAP directory (for example via LDIF export) before bulk changes.

## Links

- [Documentation](./docs/)
- [Installation](./docs/installation.md)
- [Configuration](./docs/configuration.md)
- [Integration guide](./docs/integration.md)
- [Contributing](./CONTRIBUTING.md)

## Features

| Feature | Status |
| --- | --- |
| Connection profiles (plain / StartTLS / LDAPS) | MVP |
| Dashboard & connection test | MVP |
| User list / search | Planned |
| User create / edit / delete | Planned |
| Admin password reset | Planned |
| Email (`mail`) management | Planned |
| Custom attributes | Planned |
| Samba attribute view (sambaSID, flags) | Planned |
| Integration guide (from local config) | Planned |
| Group membership | v0.2 |
| LDIF import / export | v0.2 |

## Quick start

### Requirements

- Go 1.22+ (to build from source)
- OpenLDAP or compatible LDAP server
- LDAP bind DN with administrative write access

### Build from source

```bash
git clone https://github.com/ldash-sh/ldash.git
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

### Keys (dashboard)

| Key | Action |
| --- | --- |
| `r` | Test LDAP connection |
| `q` | Quit |

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
    └── user_samba_posix.yaml
```

See [configuration.md](./docs/configuration.md).

## Security

- Do not commit credentials or environment-specific LDAP data.
- Keep `~/.config/ldash/` at mode `0700` and credential files at `0600`.
- ldash does not require root; it needs LDAP admin bind privileges only.

## License

This project is licensed under the [GNU General Public License v3.0](LICENSE).

## Contributing

See [CONTRIBUTING.md](./CONTRIBUTING.md). Bug reports and pull requests are welcome.
