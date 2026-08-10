# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- Main menu navigation and user management TUI (list, search, create, edit, delete)
- TUI app shell (header / content panel / sticky footer), help overlay (`?`), and English [TUI guide](docs/tui.md) with generic screenshots
- Predictable navigation: `Esc` one level back, `q` quit on main menu only, menu number shortcuts, disabled Groups/Samba entries
- Users list paging (`PgUp`/`PgDn`, Home/End), adaptive columns, scroll window, and status prefixes (`OK` / `Error` / `Warn` / `Loading`)
- Admin password reset and mail attribute management
- User template copy on `ldash config init`
- Samba SID generation from `samba.domain_sid` and uidNumber
- Integration guide panel driven by local runtime config
- Module path and GitHub references: `knieberg/ldash`
- CI jobs: test, lint, vuln (`govulncheck`), hygiene; local `make ci` parity

### Changed

- TUI layout and navigation: full-bleed shell, sticky footer, Esc/q separation, help overlay
- Stronger `.gitignore` for editor/AI metadata; lean contribution checklist
- Default user search filters cover `inetOrgPerson` **or** `posixAccount` (Samba-friendly)
- Docs: schema matching, `ldapsearch` objectClass diagnosis, alternate Samba account template example

### Security

- TLS certificate verification for StartTLS and LDAPS (hostname from server URL)
- Password changes via Password Modify extended operation only (no plaintext `userPassword` writes)
- Credential file: trim whitespace, reject empty, expect mode `0600`
- Validate `search.user_filter` has exactly one `%s`; roll back CreateUser entry if password set fails

## [0.0.1] — scaffold

### Added

- Initial project scaffold (Go, Bubble Tea, Cobra)
- Config loader and `ldash config init`
- Dashboard with LDAP connection test
- Comprehensive `.gitignore` and `make setup-local` for gitignored maintainer docs
- Documentation: installation, configuration, integration, Samba notes
