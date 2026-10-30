# Changelog

All notable changes to gitshift are recorded here, newest first. Each version
section matches a Git tag; the Unreleased section collects changes that will
ship in the next release.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [2.1.1] - 2026-10-30

### Changed

- `gitshift auto-switch` (run on every directory change by the shell hook) takes about 0.5 s
  instead of about 4 s when it switches accounts. It skips the SSH connection tests, which
  `gitshift switch` still runs; the key file and the written Git identity are still checked.
  Switching reads the active GitHub CLI user from gh's local config instead of
  `gh auth status`, renders Git include files without running `git`, and reads each Git
  config scope in one call.

### Fixed

- With `auto git-includes on`, repositories now push with their directory's key instead of
  the globally active account's. Switching and `git-includes on` remove gitshift's managed
  `Host` block from `~/.ssh/config`, whose key ssh offered first; other entries are kept.

## [2.1.0] - 2026-10-29

### Added

- `gitshift auto git-includes on|off|status`: Git picks the identity for each rule directory through
  managed `includeIf "gitdir:..."` entries and per-account include files (name, email, Git SSH
  command, commit signing). Every terminal uses the right identity for its directory at once,
  without switching; entries follow account and rule changes and are restored after every switch.

### Changed

- With Git includes on, `switch` no longer writes the identity into the current repository's
  `.git/config`, and `current --short` shows the account of the working directory's rule.

## [2.0.0] - 2026-10-29

### Highlights

- An interactive terminal UI, which `gitshift` now opens when run without a subcommand.
- Directory-based automatic switching with a zsh hook and a fallback default account.
- Multi-account discovery across GitHub CLI hosts, SSH keys and GPG keys, with chosen aliases.
- Safer switching: every change is rolled back if a step fails, and your SSH config is preserved.
- Prebuilt archives for six platforms that report their real version, plus a Homebrew cask.

### Breaking changes for v1.x users

- Only `switch` (CLI, terminal UI or auto-switch) marks an account as current. `add --default`,
  `update --default`, discovery import/overwrite and removing the default account no longer do.
- `switch` no longer writes `GIT_SSH_COMMAND` into shell startup files or clears the shared
  ssh-agent; isolation now comes from the managed `~/.ssh/config` block and `core.sshCommand`.
- A GitHub switch now requires GitHub CLI (`gh`) authenticated as the account's username on its
  host; a failed `gh auth switch` rolls the switch back unless `--force` is given.
- `switch` refuses SSH private keys that group or other users can access (`600` and `400` are
  accepted) instead of working around them.
- Supported platforms are `github` and `gitlab`. Corporate email domains are no longer assumed to
  be GitLab during discovery.
- The configuration file is migrated in place to `config_version: 1.1.0`, rewritten with
  owner-only permissions, and unknown newer versions are rejected.
- `ssh-keygen --passphrase` and `gpg-keygen --passphrase` are removed; generated keys are
  passwordless, so create protected keys with `ssh-keygen`/`gpg` directly.
- Release assets are now `gitshift_<version>_<os>_<arch>.tar.gz` (`.zip` on Windows), replacing
  the bare `gitshift-<os>-<arch>` binaries. Update any download scripts.
- macOS and Linux users can install with `brew install --cask techishthoughts-org/tap/gitshift`.

### Added

- MIT license.
- Per-account commit and tag signing with SSH keys or GPG: `update --signing ssh|gpg|off`,
  `update --signing-key PATH` and `add --signing ssh`. `switch` sets `gpg.format`,
  `user.signingkey`, `commit.gpgsign` and `tag.gpgsign` for the account, `--validate` checks the
  key, and `list`, `update` and the terminal UI show each account's signing.
- `gitshift ui` (alias `tui`, `--inline` for the current screen buffer): dashboard, account
  add/edit, confirmed switch and removal, directory-rule management, and discovery import.
- Running `gitshift` with no subcommand opens the terminal UI; subcommands and
  `--help`/`--version` behave as before.
- `auto add`, `auto list`, `auto remove` and `auto resolve` manage per-directory account rules;
  the deepest matching directory wins.
- `auto default ALIAS` (and `--clear`) sets the account used outside every rule directory.
- `auto-switch` applies the matching rule through the same transaction as `switch`, and
  `shell-init zsh` prints a hook that runs it on directory change.
- `current --short` prints only the active alias, read-only, for shell prompts such as Starship.
- `discover` imports every authenticated GitHub CLI identity on all configured hosts and tests
  each SSH key in isolation to learn which username it authenticates as.
- `discover --alias SELECTOR=ALIAS` and `--prompt-aliases` choose aliases before import; a fresh
  interactive `discover` asks for aliases and which account to activate, then applies it.
- `add` and `update` accept `--username`, `--platform`, `--domain` and `--api-endpoint`, so
  GitLab and self-hosted accounts no longer need hand-edited YAML.
- The global `--config PATH` flag is honoured by every command.
- A generated per-command reference in `docs/reference/commands`.

### Changed

- `switch` captures Git, SSH and GitHub CLI state first and restores it if any step fails;
  concurrent switches are serialized with a lock file.
- Validation probes (`switch` preflight, `ssh-test`) use only the account's key, so another key in
  `~/.ssh/config` or the agent cannot answer instead.
- Git's `core.sshCommand` is `ssh -i KEY -o IdentitiesOnly=yes`: the account's key goes first and
  `~/.ssh/config` (ports, `ProxyJump`, host aliases) and an agent holding that key keep working.
- Configuration writes are atomic, locked, and refused when the file changed since it was read.
- `update` changes only the fields you pass and keeps every other account setting.
- `list --format json` emits properly escaped JSON.
- Release binaries embed their real tag, so `gitshift --version` reports the installed version.
- Builds require Go 1.26.

### Fixed

- Options above the first `Host`/`Match` line in `~/.ssh/config` (such as `Include` or
  `IdentityAgent`) stay at the top instead of moving under the managed host block.
- A hand-written block for the managed host is commented out with `# gitshift-disabled:`
  instead of deleted, and the original config is kept once in `~/.ssh/config.gitshift-orig`.
- Duplicate `Host` entries for the managed domain no longer offer the previous account's key.
- The terminal UI account form no longer pre-fills `github.com`, so GitLab accounts get the
  correct default domain.
- Quit requests in the terminal UI wait until a running switch or removal finishes.
- Commands no longer print `Using config file: …` to stderr on every run.
- Switching no longer moves the default account to the account you switched to.
- Switching to an account without a GPG key no longer turns off SSH commit signing configured
  with `gpg.format ssh`; signing now follows each account's `--signing` choice.
- `switch --validate` also checks key permissions, the GitHub CLI login and the GPG secret key, so it
  no longer passes when the real switch would fail.
- `ssh-keygen --add-to-github` uploads to the named account's GitHub user instead of whichever
  account is active in `gh`.
- `ssh-test` recognises hashed `known_hosts` entries.
- `update --ssh-key` normalizes the path and requires the file to exist; `update --auto-gpg` fails
  with an error when no GPG key matches instead of doing nothing.
- Removing the default account passes the default to the current account, or the first remaining
  alias, instead of an arbitrary one.
- `current` with nothing applied says no account is active instead of "no default account set".
- Errors are printed once, without the full usage text.
- Hints point to `update ALIAS --signing` instead of `gpg-keygen ALIAS --enable`, which fails
  when the account already has a key.
- New accounts no longer write unused `isolation_metadata` suggesting encrypted token storage.
- The zsh hook calls the `gitshift` on your `PATH` (for Homebrew, `$(brew --prefix)/bin/gitshift`)
  instead of a versioned path, so open shells keep working after `brew upgrade`.

### Removed

- Writing `GIT_SSH_COMMAND` to `~/.zshrc`/`~/.zsh_secrets` and clearing ssh-agent during `switch`.
- The `--passphrase` flags of `ssh-keygen` and `gpg-keygen`.
- The `VERSION` file and the chained automatic tag/build/publish workflows.

### Security

- The configuration directory is created `0700` and the file `0600`, and provider tokens are
  never written to configuration or shown in the terminal UI.
- Upgraded to Go 1.26, `golang.org/x/text` v0.39.0 (GO-2026-5970) and `golang.org/x/crypto`
  v0.52.0 (GO-2026-5018).
- Release binaries are built with the latest Go 1.26 patch release (1.26.10 at the time of
  writing; fixes GO-2026-6809 and the standard-library part of GO-2026-6617), and
  `golang.org/x/net` is v0.60.0 (GO-2026-6617).

### Maintenance

- Releases are cut by one manually dispatched workflow that tests, builds with
  GoReleaser, publishes `checksums.txt`, verifies the uploaded assets and updates the Homebrew tap.

## [1.4.2] - 2026-01-04

Source-only tag; no release binaries.

### Fixed

- `.tool-versions` pinned a non-existent Go release; it now matches the `go.mod` toolchain (1.23.2).

## [1.4.1] - 2025-11-30

Source-only tag; no release binaries.

### Added

- `update --platform` and `update --auto-gpg` (finds a local GPG key by the account email).

### Fixed

- `switch` always reapplies SSH and Git configuration, even for the account already recorded as
  current, so a reported switch can no longer leave stale settings.
- `list` groups accounts under their real platform instead of showing every account as GitHub.

## [1.4.0] - 2025-11-29

Source-only tag; no release binaries.

### Added

- Discovery reads signing-capable GPG secret keys (RSA, DSA, EdDSA) and merges them with SSH keys
  by email.
- Discovery guesses the platform from the email: `gitlab`/`github` in the domain, personal
  providers as GitHub, and other corporate domains as GitLab (reversed in 2.0.0).

### Changed

- `make install` installs to `$GOPATH/bin` (default `~/go/bin`) without `sudo`.

### Fixed

- `switch` sets `core.sshCommand` with the account key and `IdentitiesOnly=yes` in global and
  repository Git config, so Git stops using the previous account's key.
- Account aliases containing dots no longer corrupt the configuration file.
- The SSH scanner only takes a GitHub username from a key filename when it is a valid username.

## [1.3.0] - 2025-11-26

Source-only tag; no release binaries.

### Added

- `gpg-keygen` generates RSA, ECC or DSA signing keys for an account and copies the public key.
- `add --enable-gpg`; `switch` configures or disables commit signing for the target account.
- `list` shows GPG status and warns about expiring keys.

### Fixed

- Clipboard copy no longer leaks resources after key generation.

## [1.2.1] - 2025-11-18

Source-only tag; no release binaries.

### Fixed

- Copying the public key works on Linux (`xclip`, `xsel`, `wl-copy`) and Windows (`clip`), not
  only macOS.
- The shell setup step detects bash, zsh, fish and ksh and writes the right file and syntax;
  Windows gets manual instructions.
- Maintenance: `go.mod` returned to Go 1.23 with a pinned 1.23.2 toolchain.

## [1.2.0] - 2025-11-14

Source-only tag; no release binaries.

### Added

- `ssh-keygen` adds GitLab host keys to `known_hosts` and points to the GitLab key settings page.

### Changed

- Command help and examples cover GitHub, GitLab, GitHub Enterprise and self-hosted instances.

## [1.1.2] - 2025-11-12

Source-only tag; no release binaries.

### Changed

- Maintenance: README rewritten around multi-platform use.

## [1.1.1] - 2025-11-12

Source-only tag; no release binaries.

### Changed

- Maintenance: release process documentation.

## [1.1.0] - 2025-11-12

Source-only tag; no release binaries.

### Added

- GitLab and self-hosted/enterprise support through new account fields `platform`, `username`,
  `domain` and `api_endpoint`, with platform-specific SSH host entries.
- `ssh-keygen` adds the new key to ssh-agent and copies the public key to the clipboard.

### Deprecated

- `github_username` in favour of `username` plus `platform`; existing configs keep working.

### Security

- Built with Go 1.24 to pick up standard-library vulnerability fixes.

## [1.0.3] - 2025-09-27

Release binaries were published on GitHub only and report version v0.1.0.

### Changed

- Maintenance: release titles use the `gitshift-vX.Y.Z` format.

## [1.0.2] - 2025-09-27

Release binaries were published on GitHub only and report version v0.1.0.

### Changed

- Maintenance: CI builds binaries for multiple platforms and publishes releases automatically.

## [1.0.1] - 2025-09-27

Release binaries were published on GitHub only and report version v0.1.0.

### Fixed

- Maintenance: release workflow fix.

## [1.0.0] - 2025-09-27

Release binaries were published on GitHub only and report version v0.1.0.

### Added

- First release: SSH-first management of multiple GitHub accounts with commands `add`, `list`,
  `current`, `switch`, `update`, `remove`, `discover`, `ssh-keygen` and `ssh-test`.
- `switch` writes an isolated `~/.ssh/config` entry (`IdentitiesOnly=yes`), reloads ssh-agent
  with the account key, exports `GIT_SSH_COMMAND` in `~/.zsh_secrets`, and updates Git
  `user.name`/`user.email` and the GitHub CLI account.
- `ssh-keygen` creates Ed25519, RSA or ECDSA keys, manages GitHub entries in `known_hosts`, and
  can upload the key with `--add-to-github` via GitHub CLI.
- `discover` finds existing accounts from SSH keys in `~/.ssh`, with `--dry-run`,
  `--auto-import` and `--overwrite`.
- YAML configuration in `~/.config/gitshift/config.yaml`.

[Unreleased]: https://github.com/techishthoughts-org/gitshift/compare/v2.1.1...HEAD
[2.1.1]: https://github.com/techishthoughts-org/gitshift/compare/v2.1.0...v2.1.1
[2.1.0]: https://github.com/techishthoughts-org/gitshift/compare/v2.0.0...v2.1.0
[2.0.0]: https://github.com/techishthoughts-org/gitshift/compare/v1.4.2...v2.0.0
[1.4.2]: https://github.com/techishthoughts-org/gitshift/compare/v1.4.1...v1.4.2
[1.4.1]: https://github.com/techishthoughts-org/gitshift/compare/v1.4.0...v1.4.1
[1.4.0]: https://github.com/techishthoughts-org/gitshift/compare/v1.3.0...v1.4.0
[1.3.0]: https://github.com/techishthoughts-org/gitshift/compare/v1.2.1...v1.3.0
[1.2.1]: https://github.com/techishthoughts-org/gitshift/compare/v1.2.0...v1.2.1
[1.2.0]: https://github.com/techishthoughts-org/gitshift/compare/v1.1.2...v1.2.0
[1.1.2]: https://github.com/techishthoughts-org/gitshift/compare/v1.1.1...v1.1.2
[1.1.1]: https://github.com/techishthoughts-org/gitshift/compare/v1.1.0...v1.1.1
[1.1.0]: https://github.com/techishthoughts-org/gitshift/compare/v1.0.3...v1.1.0
[1.0.3]: https://github.com/techishthoughts-org/gitshift/compare/v1.0.2...v1.0.3
[1.0.2]: https://github.com/techishthoughts-org/gitshift/compare/v1.0.1...v1.0.2
[1.0.1]: https://github.com/techishthoughts-org/gitshift/compare/v1.0.0...v1.0.1
[1.0.0]: https://github.com/techishthoughts-org/gitshift/releases/tag/v1.0.0
