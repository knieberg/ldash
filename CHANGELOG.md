# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.2.0] — 2026-08-10

First public release.

### Added

- **Groups:** list, create, edit, delete, and member management driven by group templates (`posixGroup` / `groupOfNames` examples)
- **LDIF:** export and import from the TUI (People / Groups / Both scopes); password hashes redacted on export and skipped on import
- **Samba:** hub overview plus per-user status (`Present` / `Missing` for hashes); edit `sambaAcctFlags` with format validation
- **Template-driven forms:** user and group create/edit render `required` / `optional` / `custom_attributes` from YAML templates
- **Custom attributes:** separate `custom_attributes` block; duplicate names vs built-in lists fail at template load
- **Discoverability:** labeled key hints in footer and help (English verbs, not bare single-letter keys)
- **LDAP field glossary:** friendly labels, attribute names, helper text, and `(required)` markers on forms
- **`config init` embed:** shipped binary writes `config.yaml`, active user/group templates, and reference copies via `go:embed` (no repo-relative paths)
- **Settings:** shows loaded user/group templates and optional integration file status
- Main menu items **1–7:** Dashboard, Users, Groups, LDIF, Samba, Integration Guide, Settings (all enabled)
- TUI app shell, help overlay, English [TUI guide](docs/tui.md) with SVG screenshots
- User management MVP: list, search, create, edit, delete, password reset, mail attribute
- Primary group creation uses the active group template when `create_primary_group: true`
- Samba SID generation from `samba.domain_sid` and uidNumber
- Integration guide panel from local runtime config
- CI: test, lint, vuln (`govulncheck`), hygiene; local `make ci` parity
- GitHub release installer (`scripts/install.sh`) targeting `~/.local/bin`

### Changed

- TUI navigation: full-bleed shell, sticky footer, `Esc` one level back, `q` quit on main menu only
- Users list paging, adaptive columns, scroll window, textual status prefixes (`OK:` / `Error:` / `Warn:` / `Loading:`)
- Default user search filters cover `inetOrgPerson` **or** `posixAccount`
- Docs and screenshots aligned with 0.2.0 menu layout and labeled keys

### Security

- TLS certificate verification for StartTLS and LDAPS
- Password changes via Password Modify extended operation only; LDIF import skips `userPassword` / Samba hash attributes
- Credential file: trim whitespace, reject empty, expect mode `0600`
- Validate `search.user_filter` has exactly one `%s`; roll back CreateUser entry if password set fails

## [0.0.1] — scaffold

### Added

- Initial project scaffold (Go, Bubble Tea, Cobra)
- Config loader and `ldash config init`
- Dashboard with LDAP connection test
- Comprehensive `.gitignore` and `make setup-local` for gitignored maintainer docs
- Documentation: installation, configuration, integration, Samba notes

[0.2.0]: https://github.com/knieberg/ldash/compare/v0.0.1...v0.2.0
