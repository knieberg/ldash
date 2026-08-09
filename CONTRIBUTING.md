# Contributing to ldash

Thank you for your interest in contributing.

## Development setup

1. Fork and clone the repository.
2. Work as a normal user — do not develop as root inside the project tree.
3. Run `make setup-local` to create **gitignored** local maintainer files (`GOALS.local.md`, `MAINTAINER.local.md`).
4. Build and test:

```bash
make build
make test
```

5. Optional lint (requires [golangci-lint](https://golangci-lint.run/)):

```bash
make lint
```

## Rules (mandatory)

- **Never commit environment-specific data** — no private IPs, real base DNs, hostnames, or credentials.
- Use only **`example.com`** / **`dc=example,dc=com`** in tracked docs, tests, and examples.
- Do not commit editor/AI project metadata (`.cursor/`, `.agents/`, `AGENTS.md`, `*.plan.md`), `*.local.md`, `local/`, `bin/`, or live configs.
- Keep the repository lean: remove unused scratch files and avoid duplicate docs.
- Integration-related output must come from **runtime config**, not hardcoded values in source.
- English only for user-facing UI strings, README, and documentation in the repository.

## Pull request checklist

- [ ] Tests pass (`make test`)
- [ ] No secrets or site-specific values in the diff
- [ ] No editor/AI metadata staged
- [ ] Documentation updated if behavior or config changed
- [ ] Commit messages are clear and in English
- [ ] Unused temporary files cleaned up

## Local maintainer docs

Files matching `*.local.md` are gitignored and intended for private notes (goals, deployment context, push checklists). They must **never** be added to git.

## License

By contributing, you agree that your contributions will be licensed under the GPL-3.0 license.
