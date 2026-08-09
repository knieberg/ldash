# Security Policy

## Supported versions

| Version | Supported |
| --- | --- |
| latest release | yes |
| older releases | best effort |

## Reporting a vulnerability

Please **do not** open public GitHub issues for security vulnerabilities.

Report issues privately via GitHub Security Advisories on the repository, or contact the maintainers through the channel listed in the repository profile.

Include:

- Description of the issue
- Steps to reproduce
- Impact assessment
- Affected version(s)

We aim to acknowledge reports within a reasonable timeframe and will coordinate disclosure after a fix is available.

## Credential handling

- Never commit LDAP bind passwords or TLS private keys.
- Store credentials in `~/.config/ldash/credentials` with mode `0600`.
- ldash reads configuration from the user's home directory only.

## Transport and passwords

- Prefer `starttls` or `ldaps`. Certificate hostname verification is enforced (no skip-verify).
- `tls_mode: plain` is unencrypted and intended for local lab use only — not for production.
- Password changes use the LDAP Password Modify extended operation only. ldash does not write plaintext `userPassword` attributes as a fallback.
