# Terminal UI guide

ldash presents a full-screen terminal UI (TUI) for OpenLDAP administration. The shell uses a sticky header, a content panel, and a footer bar with labeled keys and a textual status line.

## Screenshots

![Main menu](assets/tui-main-menu.svg)

*Main menu with seven enabled items and labeled footer keys.*

![Users list](assets/tui-users.svg)

*Users view with sticky column headers and a sticky footer bar.*

![Help overlay](assets/tui-help.svg)

*Help overlay (`?`) showing the global navigation model.*

## Starting ldash

1. Install and configure: see [Installation](installation.md) and [Configuration](configuration.md).
2. Run `ldash config init` (embedded templates work from the installed binary).
3. Run `ldash` from a real terminal (not a bare pipe).
4. Enter the bind password when prompted (unless a credentials file is configured).

## Navigation model

| Key | Action |
| --- | --- |
| `Esc` | Go **one level** back (help → search → form → list → main menu) |
| `q` | Quit — **main menu only** |
| `Ctrl+C` | Quit anytime |
| `?` | Toggle help overlay (Esc/`?` closes help first) |
| `1`–`7` | Jump to / open a main menu item |

On the main menu, `Esc` does not quit; the status line reminds you to press `q`.

## Main menu

| # | Item | Notes |
| --- | --- | --- |
| 1 | Dashboard | Connection health; press `r test` |
| 2 | Users | List, search, create, edit, delete, password, mail, Samba status |
| 3 | Groups | List, search, create, edit, delete, members |
| 4 | LDIF | Export / import backup (password hashes redacted / skipped) |
| 5 | Samba | Attribute overview; per-user status from Users (`s samba`) |
| 6 | Integration Guide | Runtime snippets from local config (no template dump) |
| 7 | Settings | Connection profile and loaded templates |

Move with `j`/`k` or arrow keys, open with `Enter`, or use number keys.

## Dashboard

- Shows server URL, TLS mode, base DN, people DN, and groups DN from your config.
- `r test` — test LDAP connection (`Loading:` then `OK:` / `Error:`).
- `Esc back` — return to main menu.

## Users

Footer keys use verbs, for example: `c create · e edit · d delete · p password · m mail · s samba · / search · r refresh`.

| Key | Action |
| --- | --- |
| `j` / `k` | Move selection |
| `PgUp` / `PgDn` | Page through the list |
| `g` / `Home` | First user |
| `G` / `End` | Last user |
| `/ search` | Filter (Enter apply; Esc cancel; `Ctrl+U` clear input) |
| `c create` | Create user from active user template |
| `e edit` | Edit selected user |
| `d delete` | Delete selected user (confirm with `y` twice) |
| `p password` | Reset password (Password Modify only) |
| `m mail` | Edit mail |
| `s samba` | Samba attribute status for selected user |
| `r refresh` | Refresh list |
| `Esc back` | Back to main menu |

Create/edit forms show **Friendly name (attr)**, helper text, and `(required)` where applicable. Empty optional/custom fields are cleared on edit.

## Groups

| Key | Action |
| --- | --- |
| `/ search` | Filter groups |
| `c create` | Create group from active group template |
| `e edit` | Edit description and custom attributes |
| `d delete` | Delete group (blocked if users still use its `gidNumber`) |
| `m members` | View / add / remove members |
| `r refresh` | Refresh list |

Member add accepts a **uid**; for `groupOfNames` templates the DN is resolved automatically.

## LDIF

1. Choose **Export LDIF** or **Import LDIF**.
2. For export, pick scope **People**, **Groups**, or **Both**, then enter a file path.
3. For import, preview entry count, then confirm twice (`y`).

Export redacts `userPassword` and Samba hash attributes. Import skips those attributes (same never-write policy as user create).

Summary shows applied / failed / skipped counts.

## Samba

The Samba hub explains `sambaSID`, `sambaAcctFlags`, and `sambaNTPassword` (Present/Missing only). Open **Users**, select an account, press `s samba`, then `e edit` to change flags (format `[U          ]`).

Hashes are never displayed or written from ldash.

## Integration Guide and Settings

- **Integration Guide:** read-only values from `config.yaml` and optional `integration.yaml`.
- **Settings:** connection block plus loaded user/group template names and integration file presence.

Press `Esc back` to return to the main menu. See [Integration guide](integration.md).

## Status line

The footer status uses text prefixes (not color alone):

- `OK:` — success
- `Error:` — failure (fix field / Esc / retry where applicable)
- `Warn:` — destructive confirmation
- `Loading:` — async work in progress

## Key reference (short)

**Global:** `Esc back` · `? help` · `Ctrl+C quit` · `q quit` (main menu)

**Menu:** `j/k move` · `1-7 open` · `Enter open`

**Users:** labeled footer — `c create` · `e edit` · `d delete` · `p password` · `m mail` · `s samba` · `/ search` · `r refresh`

**Groups:** `c create` · `e edit` · `d delete` · `m members` · `/ search` · `r refresh`
