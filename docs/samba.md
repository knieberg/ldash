# Samba and LDAP

ldash supports Samba-related LDAP attributes when the **Samba schema** is loaded on your OpenLDAP server and entries use `sambaSamAccount`.

## TUI (0.2)

| Location | What it shows |
| --- | --- |
| **Samba** (main menu 5) | Hub: `domain_sid` configured, user template has `sambaSamAccount`, attribute glossary |
| **Users → `s samba`** | Per-user status for `sambaSID`, `sambaAcctFlags`, `sambaNTPassword` |
| **Users → Samba → `e edit`** | Edit `sambaAcctFlags` only (format must look like `[U          ]`) |

Password hashes are never displayed or written. `sambaNTPassword` is shown as **Present** or **Missing** only.

If `samba.domain_sid` is missing or the user template lacks `sambaSamAccount`, the Samba hub shows a warning with what to fix.

## Common attributes

| Attribute | Purpose |
| --- | --- |
| `sambaSID` | Security identifier for the Samba account |
| `sambaAcctFlags` | Account flags (e.g. `[D          ]` disables the account) |
| `sambaNTPassword` | NT hash (often synced from `userPassword`) |

## smbk5pwd overlay

Some OpenLDAP deployments use the **smbk5pwd** overlay to synchronize `userPassword` changes to `sambaNTPassword`. ldash does not install or configure overlays; future versions may note detection in the Samba hub.

## User template

Example templates (embedded in `ldash config init`):

- `user_samba_posix.yaml` — active default: `inetOrgPerson`, `posixAccount`, `sambaSamAccount`
- `user_samba_account.example.yaml` — reference: `account`, `posixAccount`, `sambaSamAccount`

Adjust object classes in `~/.config/ldash/templates/` to match entries already in the directory (see [configuration.md](configuration.md)).

## Standalone vs domain controller

ldash does not configure Samba itself. It manages LDAP entries only. Samba server role (standalone, domain member, AD) is outside ldash scope.

## LDIF and passwords

LDIF export redacts `sambaNTPassword` / `sambaLMPassword`. LDIF import skips those attributes (same never-write policy as user create). Set login secrets via **Users → `p password`** (Password Modify extended operation).
