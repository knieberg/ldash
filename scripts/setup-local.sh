#!/usr/bin/env bash
# Creates gitignored maintainer docs in the project root. Safe to re-run; never overwrites existing files.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

mkdir -p local

create_if_missing() {
  local path="$1"
  if [[ -f "$path" ]]; then
    echo "exists: $path"
    return 0
  fi
  cat > "$path"
  echo "created: $path"
}

create_if_missing "MAINTAINER.local.md" <<'EOF'
# ldash — Maintainer Rules (local, private — NEVER commit)

## Zero traceability

- The public GitHub repository must stay **fully generic**.
- No operator infrastructure, domains, private IPs, hostnames, or usernames in commits.
- Site-specific LDAP data lives only in:
  - `~/.config/ldash/` (runtime config)
  - Section **Local deployment context** in `GOALS.local.md`
  - This file and other `*.local.md` files (gitignored)

## Development

- Work as a normal user — **never as root** inside the project tree.
- Project path: `~/ldash`
- All project files owned by your user account.
- SSH workflow: `cd ~/ldash && git && make && go`

## Git hygiene

1. Commit `.gitignore` patterns before adding other files.
2. Never commit: `.cursor/`, `*.local.md`, `local/`, secrets, `bin/`, live configs.
3. Use only `example.com` / `dc=example,dc=com` in tracked docs and tests.
4. Use a neutral public maintainer name/email for commits (no internal hostnames).

## Push checklist

- [ ] `git status` — no unexpected files
- [ ] `git diff` — no private DNs, IPs, URLs, usernames
- [ ] No absolute home paths in tracked files
- [ ] No credentials or `.local` artifacts staged

## Product scope reminders

- **In scope:** LDAP admin TUI — users, groups, passwords, mail, Samba attrs, integration hints from local config.
- **Out of scope:** ppolicy / force password change on next login, OIDC provider management, user self-service web UI, DNS/firewall/proxy setup.
- Integration Guide values are **runtime-only** from loaded config — never hardcoded in source.

## Optional local hook

Place a private pre-push script outside the repo, e.g. `~/.config/ldash-dev/pre-push`, to scan for forbidden patterns before push.
EOF

create_if_missing "GOALS.local.md" <<'EOF'
# ldash — Project Goals (local, private — NEVER commit)

## Public project vision

- Generic OpenLDAP administration TUI (terminal user interface)
- Open source, GPL-3.0
- English UI, README, and documentation
- GitHub-ready: install script, releases, CI, comprehensive docs

## MVP (v0.1)

- [ ] Connection profiles and TLS modes (plain / StartTLS / LDAPS)
- [ ] Dashboard with connection health
- [ ] User list, search, filter
- [ ] User create / edit / delete (configurable template)
- [ ] Admin password reset
- [ ] Email (`mail`) set / change / remove
- [ ] Custom attribute add / remove
- [ ] Samba status (smbk5pwd detection, sambaSID, sambaAcctFlags)
- [ ] Integration guide panel (generated from local config at runtime)
- [ ] `ldash config init`, install script, update command skeleton

## v0.2+

- [ ] Group membership management
- [ ] LDIF export / import
- [ ] Extended user templates

## Explicitly out of scope

- Password policy overlay (ppolicy) / pwdReset / force change on next login
- User self-service web applications
- OIDC identity provider configuration
- Reverse proxy, DNS, firewall
- OpenLDAP server installation (link to generic docs only)

## Local deployment context (PRIVATE)

<!-- Fill in after cloning — this section never leaves your machine -->

- LDAP host:
- Base DN:
- People OU:
- Groups OU:
- Schemas / overlays (e.g. samba, smbk5pwd):
- OIDC bridge (if any):
- Self-service password URL (if any):
- uid/gid start:
- Staging / test user:
- Notes:

EOF

chmod 600 MAINTAINER.local.md GOALS.local.md 2>/dev/null || true
echo "Done. Edit GOALS.local.md (Local deployment context) and keep *.local.md out of git."
