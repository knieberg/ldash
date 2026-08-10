# Installation

## Requirements

- Linux, macOS, or other Unix-like system with a terminal
- OpenLDAP or compatible LDAP server
- LDAP administrative bind DN
- Go 1.24+ (CI uses Go 1.25.12 via `toolchain` in `go.mod`)

## Build from source (recommended until releases are published)

```bash
git clone https://github.com/knieberg/ldash.git
cd ldash
make build
make install-local
```

The binary is installed to `~/.local/bin/ldash`. Ensure that directory is on your `PATH`.

## Maintainer local setup

```bash
make setup-local
```

Creates gitignored files in the project root:

- `GOALS.local.md` — private project goals and local deployment notes
- `MAINTAINER.local.md` — private maintainer rules and push checklist

These files are **never** committed.

## Initialize configuration

```bash
ldash config init
```

This creates `~/.config/ldash/config.yaml` from the shipped example. Edit it with your server details.

```bash
chmod 700 ~/.config/ldash
chmod 600 ~/.config/ldash/config.yaml
```

### Optional credential file

```bash
printf '%s' 'your-bind-password' > ~/.config/ldash/credentials
chmod 600 ~/.config/ldash/credentials
```

Reference it in config:

```yaml
credential_file: "~/.config/ldash/credentials"
```

## First run

```bash
ldash
```

See the [Terminal UI guide](tui.md) for navigation, keys, and screenshots.

## File permissions

| Path | Mode | Purpose |
| --- | --- | --- |
| `~/.config/ldash/` | 0700 | Config directory |
| `~/.config/ldash/config.yaml` | 0600 | Main config |
| `~/.config/ldash/credentials` | 0600 | Bind password |
| `~/.local/bin/ldash` | 0755 | User install binary |
| `/usr/local/bin/ldash` | 0755 | System install binary |

## Prebuilt binary (future)

When releases are available:

```bash
curl -fsSL https://raw.githubusercontent.com/knieberg/ldash/main/scripts/install.sh | bash
```

Until then, use the build-from-source steps above.

## Uninstall

```bash
./scripts/uninstall.sh
rm -rf ~/.config/ldash   # optional — removes local config
```
