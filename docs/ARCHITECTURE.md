# Architecture

GitShift is a Go command-line application built with Cobra. Commands in `cmd/` validate input and coordinate internal services; `internal/config` persists account metadata; `internal/git`, `internal/ssh`, `internal/gpg`, and GitHub integration adapt external tools and files; `pkg/platform` defines supported provider behavior.

## Switch transaction

A manual or directory-triggered switch follows this logical sequence:

1. Acquire a portable cross-process transaction lock, then load the account from the selected configuration file.
2. Validate basic account/platform data and SSH readiness. GPG and GitHub CLI are not part of validate-only mode.
3. Capture the external state needed for compensation.
4. Apply SSH, global and repository-local Git, optional GPG, and GitHub CLI changes.
5. In normal mode, test resulting global Git identity and SSH connectivity, then persist the selected account. Force/skip modes omit or tolerate documented checks.
6. On a normal error, attempt rollback and report any rollback failure.

The external tools do not provide one shared transaction, so crash-safe atomicity across all state is impossible. The switch lock prevents two GitShift processes from interleaving external mutations; it cannot coordinate unrelated tools. The config layer itself uses a separate write lock, stale-write detection, owner-only temporary files, atomic replacement, and directory synchronization where supported.

`internal/autoswitch` canonicalizes existing directories with absolute paths and symlink evaluation. Resolution uses `filepath.Rel` for component-aware containment and selects the deepest match. Rules reference account aliases; account removal and discovery replacement remove dangling rules. `auto-switch` calls the same idempotent switch implementation as the manual command and never forces through a failed check.

## Provider boundary

The supported platform identifiers are `github` and `gitlab`. Domains customize those providers. GitHub switching integrates with `gh`; GitLab has no provider-CLI adapter. Adding another provider requires validation, defaults, SSH behavior, tests, and documentation rather than only accepting another string.

## Discovery identity and merging

Authenticated provider identities are keyed as lowercase `provider|hostname|username`; usernames alone are not globally unique. The GitHub CLI scanner enumerates all users from `gh auth status --json hosts` and enriches each by requesting its explicit host/user token without changing the active account. That credential exists only in memory and in the environment of the corresponding `gh api` child process.

SSH and GPG records do not by themselves prove provider ownership. A username inferred from an SSH filename is attached first only when exactly one SSH record maps to exactly one provider identity across all hosts. Remaining evidence is merged by normalized email only when unique on both sides. Shared usernames across hosts, shared emails, duplicate keys, and other ambiguous evidence remain independent records. Final aliases are sanitized and receive deterministic numeric suffixes when their preferred aliases collide.

Optional user aliasing is a command-layer planning step after scanning and before import. Explicit selectors resolve an exact canonical identity key before falling back to an exact, unique suggested alias. Prompting covers only results that were not selected explicitly. The planner returns aliases without mutating discovery or configuration state, validates their format, and checks case-insensitive uniqueness across the whole result plus existing active and pending aliases for additive imports. Only a complete valid plan is applied; overwrite mode excludes old aliases from collision checks and relies on atomic account replacement to remove dangling directory rules.

Configuration has separate mutating and read-only load paths. Report-only discovery uses the read-only path, which treats an absent configuration as empty and performs no directory creation, permission changes, migration writes, or persistence.

## Testing

Unit tests should isolate filesystem and external-command behavior. Integration-oriented switch tests must verify forward order, reverse rollback order, persistence timing, force-mode behavior, and failures at each boundary. See [Contributing](CONTRIBUTING.md).
