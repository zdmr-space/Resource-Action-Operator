# Contributing

Thanks for contributing to Resource Action Operator.

## Development Setup

1. Install prerequisites:
- Go 1.24+
- Docker
- kubectl
- kind

2. Run tests:

```bash
make test
```

3. Run local demo flow (optional):

```bash
hack/demo/99-run-all.sh
```

## Pull Request Checklist

- Keep changes focused and small.
- Add or update tests for behavior changes.
- Run `make test` before opening a PR.
- Update docs when behavior or UX changes:
  - `README.md`
  - `docs/modules/ROOT/pages/*.adoc`
  - `hack/demo/README.md` (if demo flow changed)

## Coding Guidelines

- Use clear, pragmatic names.
- Keep comments in English.
- Prefer explicit error messages.
- Preserve backward compatibility where practical.

## Commit Style

Use concise, imperative commit messages:

- `controller: validate engine before reconcile`
- `docs: add demo reset workflow`
- `engine: support cron mode compatibility`
