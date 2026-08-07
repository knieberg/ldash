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
  user_filter: "(&(objectClass=inetOrgPerson)(uid=%s))"
  group_filter: "(objectClass=posixGroup)"
  list_users_filter: "(objectClass=inetOrgPerson)"

templates_dir: "~/.config/ldash/templates"
```

## TLS modes

| Mode | Description |
| --- | --- |
| `plain` | Unencrypted LDAP (port 389) |
| `starttls` | Upgrade to TLS after connect |
| `ldaps` | LDAP over TLS (typically port 636) |

## User templates

Copy `internal/templates/user_samba_posix.example.yaml` to:

```
~/.config/ldash/templates/user_samba_posix.yaml
```

Customize object classes and defaults for your directory layout.

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
