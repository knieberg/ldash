# Integration guide

ldash can display **copy-paste snippets** for connecting other services to your LDAP directory.

## Important

All values shown in the Integration Guide are read from **your local configuration at runtime**. The ldash source code and repository contain **no** site-specific integration data.

## Generic LDAP client settings

Applications typically need:

| Setting | Source in ldash config |
| --- | --- |
| Server URI | `server.url` |
| Base DN | `base_dn` |
| Bind DN | `bind_dn` |
| Users base | `{organizational_units.people},{base_dn}` |
| Groups base | `{organizational_units.groups},{base_dn}` |
| User search filter | `search.user_filter` |

## OIDC / LDAP bridge

Many deployments use an OIDC provider with an LDAP connector. Configure the connector using the same generic fields as above. Refer to your OIDC provider's documentation for exact YAML or UI field names.

Example attribute mapping (typical):

| OIDC / app field | LDAP attribute |
| --- | --- |
| Username | `uid` |
| Email | `mail` |
| Display name | `cn` |

## Password self-service

User password changes are often handled by a separate self-service application. ldash focuses on **administrative** tasks. Optionally set `self_service_url` in `~/.config/ldash/integration.yaml` to show a reminder in the TUI.

## Example ldapsearch

Replace values with those from your local config:

```bash
ldapsearch -H ldap://ldap.example.com \
  -D "cn=admin,dc=example,dc=com" -W \
  -b "ou=People,dc=example,dc=com" \
  "(objectClass=inetOrgPerson)" uid cn mail
```

## New user onboarding checklist

1. Create user entry under People OU
2. Set `mail` if applications require email
3. Assign group memberships
4. Set initial password (admin) or direct user to self-service
5. Verify login against target application

Customize steps in `~/.config/ldash/integration.yaml`.
