# Samba and LDAP

ldash supports viewing and managing Samba-related attributes when the **Samba schema** is loaded on your OpenLDAP server and entries use `sambaSamAccount`.

## Common attributes

| Attribute | Purpose |
| --- | --- |
| `sambaSID` | Security identifier for the Samba account |
| `sambaAcctFlags` | Account flags (e.g. disabled) |
| `sambaNTPassword` | NT hash (often synced from `userPassword`) |

## smbk5pwd overlay

Some OpenLDAP deployments use the **smbk5pwd** overlay to synchronize `userPassword` changes to `sambaNTPassword`. When present, ldash notes this on the Samba status view.

ldash does not install or configure overlays; it detects and reports when possible.

## User template

The example templates include:

- `user_samba_posix.example.yaml` — `inetOrgPerson`, `posixAccount`, `sambaSamAccount`
- `user_samba_account.example.yaml` — `account`, `posixAccount`, `sambaSamAccount`

Adjust object classes in your local template under `~/.config/ldash/templates/` to match entries already in the directory (see `docs/configuration.md`).

## Standalone vs domain controller

ldash does not configure Samba itself. It manages LDAP entries only. Samba server role (standalone, domain member, AD) is outside ldash scope.
