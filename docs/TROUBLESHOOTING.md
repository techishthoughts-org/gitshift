# Troubleshooting

Start with:

```bash
gitshift --version
gitshift list --format table
gitshift current
gitshift COMMAND --help
```

## `gitshift: command not found`

Confirm the binary is executable and its directory is on `PATH`:

```bash
command -v gitshift
echo "$PATH"
```

For a user installation, add `$HOME/.local/bin` to your shell profile. On Windows, add the directory containing `gitshift.exe` to the user `PATH`, then open a new terminal.

## Download is blocked or looks corrupt

Download `checksums.txt` from the same release and verify SHA-256 as described in [Installation](INSTALLATION.md). macOS may require explicit approval in Privacy & Security for an unsigned binary after verification.

## GitHub account validation fails

Check that GitHub CLI is installed and that the exact username is authenticated on the exact hostname:

```bash
gh auth status --hostname github.com
gh auth switch --hostname github.com --user USERNAME
```

Then verify the GitShift record with `gitshift list`. Correct it with `gitshift update ALIAS --username USERNAME --domain HOSTNAME`.

## Discovery misses or combines an account unexpectedly

First compare the read-only result with GitHub CLI's account inventory:

```bash
gh auth status --json hosts
gitshift discover --dry-run
```

Discovery includes every successfully authenticated host/user pair known to `gh`, whether that account is active or inactive. Entries whose authentication status failed are skipped. If authentication succeeded but profile enrichment fails, discovery keeps the host/user identity at lower confidence without switching accounts. SSH evidence can merge first when its inferred username identifies exactly one provider identity across all hosts; remaining SSH and GPG evidence merges only on a unique email match. Ambiguous usernames, hosts, emails, or keys deliberately remain separate. Use the displayed `provider|hostname|username` identity and generated aliases to distinguish same-provider accounts. Do not use `--overwrite` until the dry-run output is correct.

If suggested aliases are unclear or collide, assign them explicitly:

```bash
gitshift discover --dry-run \
  --alias 'github|github.com|USER=personal'
```

Quote identity selectors because `|` is a shell operator. Selectors and suggested aliases are exact and case-sensitive, while final alias uniqueness is checked case-insensitively. An “ambiguous selector” means more than one result had that suggested alias; use its full identity key instead. An “unknown selector” means it did not exactly match the current scan. Existing active or pending aliases also block an additive import; select another alias or deliberately use `--overwrite` after reviewing the complete dry run.

## SSH authentication fails

```bash
gitshift ssh-test ALIAS --verbose
ls -l /path/to/private-key /path/to/private-key.pub
ssh -vT git@HOSTNAME
```

Confirm the private key exists, is normally mode `0600`, the public key is normally mode `0644`, and the public key is registered with the correct provider account. Verify enterprise host fingerprints independently. `ssh-test --all` tests all configured accounts; it does not repair host keys.

## Commits use the wrong identity

Run these inside the affected repository:

```bash
git config --show-origin --get user.name
git config --show-origin --get user.email
git config --show-origin --get core.sshCommand
```

A repository-local value overrides the global value. Switch again from inside that repository to apply local settings there, or deliberately remove the unwanted local override with Git.

## GPG signing fails

```bash
gpg --list-secret-keys --keyid-format LONG
git config --show-origin --get user.signingkey
git config --show-origin --get commit.gpgsign
```

Confirm the secret key is present and any GPG agent or pinentry required by a protected key is available.

## Configuration changed or is locked

For `configuration changed on disk; reload and retry`, rerun the command; another process or editor changed the file. For a timeout waiting on `config.yaml.lock` or `config.yaml.switch.lock`, first ensure no GitShift process is running. Only then remove the named lock next to the configuration file and retry.

Remember that `--config` is an exact path. Relative paths resolve from the current directory, and a quoted `~` is not expanded.

## The directory hook does not switch

Run `gitshift auto resolve "$PWD"` to inspect matching, then `gitshift auto list`. Paths used for matching must still exist; rules use physical paths, so a symlink resolves to its target. A similarly named sibling such as `personal-old` intentionally does not match a `personal` rule. Confirm the shell is interactive, `GITSHIFT_AUTO_SWITCH_DISABLE` is not `1`, and the hook was loaded with `source <(gitshift shell-init zsh)`. Re-source it to clear same-directory deduplication after adding a rule. Run `gitshift auto-switch "$PWD"` without `--quiet` to see validation errors.

If a configured root was deleted, copy its absolute path from `gitshift auto list` and pass that exact value to `gitshift auto remove PATH`; removal does not require the stale directory to be recreated.

## A failed switch left mixed state

GitShift reports rollback errors when it cannot fully compensate. Inspect global and local Git configuration, `~/.ssh/config`, GPG settings, `gh auth status`, and `gitshift current`. Restore your backups if needed, then run a validated switch. See [Security](SECURITY.md) for the full state list.

When reporting a bug, include the GitShift version, operating system, command (with secrets removed), exact error, and whether the command ran inside a Git repository. Never include private keys or tokens.
