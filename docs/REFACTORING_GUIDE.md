# Maintenance guide

An earlier refactoring proposal described service containers, nested commands,
security audit/auto-fix commands, migrations, token storage, and performance
frameworks. Those proposals were not implemented and are intentionally not
documented as current behavior.

Keep command behavior in sync with `--help`, README examples, and the user
guide. New platform behavior belongs behind the platform abstraction rather
than hostname conditionals spread across commands.

For changes that touch configuration or switching:

1. Preserve fields not explicitly modified.
2. Validate all inputs before changing external state.
3. Return required subprocess failures.
4. Avoid secrets in command arguments, files, logs, and test fixtures.
5. Add isolated tests and run `go test ./...` plus `go vet ./...`.
