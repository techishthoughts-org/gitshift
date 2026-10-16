# Maintainer guide

Keep refactors behavior-preserving unless a user-visible change is explicitly intended and documented.

## Boundaries

- `cmd/`: CLI parsing, prompts, presentation, and orchestration
- `internal/config`: durable configuration and concurrency safety
- `internal/git`: Git configuration adapter
- `internal/ssh`: SSH files, keys, agent, and connectivity
- `internal/gpg`: signing-key integration
- `pkg/gh`: GitHub API helper integration
- `pkg/platform`: provider validation and defaults

Do not hide external mutations in model or formatting code. Keep validation separate from mutation so `switch --validate` remains meaningful.

## Required switch invariants

- No external changes before required validation succeeds unless force mode explicitly permits it.
- Capture rollback state before the corresponding mutation.
- Roll back completed steps in reverse order.
- Persist `current_account` only after external changes succeed.
- Preserve the original failure while also reporting rollback failures.
- Document that process termination cannot be compensated.

## Configuration invariants

- Treat `--config` as an exact file path.
- Use owner-only directory, file, lock, and temporary-file permissions where supported.
- Detect stale writers instead of silently overwriting them.
- Use platform-aware atomic replacement and sync durable state where possible.
- Test Unix and Windows-specific replacement helpers.

## Review checklist

Run formatting, vet, race-enabled tests, and builds. Check command help, docs, examples, workflow claims, migration behavior, and failure tests. A builder should not be the sole approver of security-sensitive or persistence changes.
