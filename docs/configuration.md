# Configuration

ldash reads configuration from **`~/.config/ldash/config.yaml`**.

The repository ships only **`config.example.yaml`** with generic `example.com` placeholders.

## Initialize

```bash
ldash config init
ldash config path   # prints ~/.config/ldash/config.yaml
```

## Example structure

```yaml
server:
  url: "ldap://ldap.example.com:389"
  tls_mode: starttls   # plain | starttls | ldaps

base_dn: "dc=example,dc=com"
bind_dn: "cn=admin,dc=example,dc=com"

organizational_units:
  people: "ou=People"
  groups: "ou=Groups"

id_ranges:
  uid_start: 10000
  gid_start: 10000

search:
  user_filter: "(&(|(objectClass=inetOrgPerson)(objectClass=posixAccount))(uid=%s))"
  group_filter: "(objectClass=posixGroup)"
  list_users_filter: "(|(objectClass=inetOrgPerson)(objectClass=posixAccount))"

templates_dir: "~/.config/ldash/templates"

samba:
  domain_sid: "S-1-5-21-1000000000-2000000000-3000000000"
```

## Search filters and directory schema

`list_users_filter` and `user_filter` **must match the object classes** used in your People OU. Wrong filters are the most common reason for an empty or nearly empty user list.

| Filter | Purpose |
| --- | --- |
| `list_users_filter` | Who appears in the Users list |
| `user_filter` | Single-user lookup; must contain exactly one `%s` for the uid |

Defaults use an **OR** of `inetOrgPerson` and `posixAccount` so both classic inetOrg and POSIX/Samba accounts are visible.

Samba / **smbk5pwd** directories often store users as `posixAccount` + `sambaSamAccount` and may **omit** `inetOrgPerson`. In that case you can narrow filters, for example:

```yaml
search:
  list_users_filter: "(objectClass=posixAccount)"
  user_filter: "(&(objectClass=posixAccount)(uid=%s))"
```

### Diagnose object classes

Replace the base DN with yours (example.com shown):

```bash
ldapsearch -H ldap://ldap.example.com \
  -D "cn=admin,dc=example,dc=com" -W \
  -b "ou=People,dc=example,dc=com" "(uid=*)" uid objectClass
```

Align `list_users_filter` / `user_filter` (and the create template object classes) with what you see.

## TLS modes

| Mode | Description |
| --- | --- |
| `plain` | Unencrypted LDAP (port 389) — lab only |
| `starttls` | Upgrade to TLS after connect |
| `ldaps` | LDAP over TLS (typically port 636) |

## Samba domain SID

When creating users with `sambaSamAccount`, ldash sets `sambaSID` as:

`{samba.domain_sid}-{uidNumber * 2 + 1000}`

Put your real domain SID only in `~/.config/ldash/config.yaml` — never in the repository.

## User templates

`ldash config init` copies `internal/templates/user_samba_posix.example.yaml` to:

```
~/.config/ldash/templates/user_samba_posix.yaml
```

**Always adapt the template to your live schema.** Creating users with object classes that differ from existing entries leads to inconsistent directory data.

Shipped examples (generic only):

| File | Typical use |
| --- | --- |
| `user_samba_posix.example.yaml` | `inetOrgPerson` + `posixAccount` + `sambaSamAccount` |
| `user_samba_account.example.yaml` | `account` + `posixAccount` + `sambaSamAccount` (common Samba layout) |

Copy the account-oriented example over your local template when existing users do not use `inetOrgPerson`.

## Integration hints (optional)

Create `~/.config/ldash/integration.yaml` for private notes used by the Integration Guide panel:

```yaml
self_service_url: "https://password.example.com"
oidc_provider: "generic"
onboarding_checklist:
  - "Create user"
  - "Set mail attribute"
  - "Add to default group"
  - "Verify application login"
```

This file is gitignored when placed in the project tree and should live only under `~/.config/ldash/`.

## Security

- Never store bind passwords in the main config file unless you accept the risk; prefer `credentials` file or prompt.
- Restrict directory permissions to your user account.
