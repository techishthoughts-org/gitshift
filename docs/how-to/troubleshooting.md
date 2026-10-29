# Troubleshooting

Start with:

```bash
gitshift --version
gitshift list --format table
gitshift current
gitshift COMMAND --help
```

When a command fails, GitShift prints the error, then the command's usage, then the error again. The first line is the one that matters.

## `gitshift: command not found`

Run `command -v gitshift`. If it prints nothing, the install directory is not on your `PATH`: for archive installs add `$HOME/.local/bin`, for source installs `${GOPATH:-$HOME/go}/bin`. On Windows add the folder containing `gitshift.exe` to your user `PATH` and open a new terminal.

## macOS says the binary cannot be opened

Manual downloads are quarantined and the binary is not notarized. After verifying the checksum, run `xattr -d com.apple.quarantine "$(command -v gitshift)"`. The Homebrew cask does this for you. See [install](install.md).

## `gitshift current` says `no account is active`

No switch has applied an account yet (or the current account was removed). Run `gitshift switch ALIAS`; see the [identity model](../explanation/identity-model.md).

## Switch fails: `SSH key ... has permissions 644`

GitShift refuses to use a private key that group or other users can access (`600` and `400` are fine), and it does not change your key files:

```bash
chmod 600 ~/.ssh/id_ed25519_work
```

`switch --validate` reports the same problem without switching.

## Switch fails: `SSH preflight failed`

The key did not authenticate to the account's host.

```bash
gitshift ssh-test work --verbose
ssh -F /dev/null -i ~/.ssh/id_ed25519_work -o IdentitiesOnly=yes -T git@github.com
```

- `Permission denied (publickey)`: the public key is not registered on that provider account.
- `Host key verification failed`: the host is missing from `~/.ssh/known_hosts`. Verify the fingerprint and add it; see [GitLab and Enterprise](gitlab-and-enterprise.md#trust-the-host-key).
- Timeouts: network or firewall. `--skip-validation` lets you switch anyway.

## Switch fails on the GitHub CLI

Errors such as `GitHub CLI not found`, `determine current GitHub CLI account`, or `account 'USER' not found in GitHub CLI` mean `gh` is missing or that exact user is not logged in on that exact host:

```bash
gh auth status --hostname github.com
gh auth login --hostname github.com
```

Fix the login rather than using `--force`. Check the account's username and host with `gitshift list`, and correct them with `gitshift update ALIAS --username USER --domain HOST`.

## Commits use the wrong identity

Run inside the repository:

```bash
git config --show-origin --get-regexp '^(user\.|core\.sshcommand)'
```

A repository-local value overrides the global one. Switching inside a repository writes local values that stay there; see [repository-local identity](switch-accounts.md#repository-local-identity).

## Git over SSH uses the wrong key

While an account with a key is active, Git runs `ssh -i KEY -o IdentitiesOnly=yes`, so the account's key is offered first and your `~/.ssh/config` still applies. If the provider authenticates you as another user, the account's key is probably not registered there and a key from an `IdentityFile` line answered instead. Check with `gitshift ssh-test ALIAS`; see the [security model](../explanation/security-model.md#git-ssh-command).

## GPG signing fails

```bash
gpg --list-secret-keys --keyid-format LONG
git config --show-origin --get user.signingkey
git config --show-origin --get commit.gpgsign
```

Make sure the secret key is in your keyring and any pinentry your key needs is available. To stop signing for an account, see [keys](keys.md#use-an-existing-gpg-key).

## The directory hook does not switch

1. `gitshift auto resolve "$PWD"` shows what should be selected. Nothing printed means no rule matches and no default is set.
2. `gitshift auto list` shows the stored paths. They are physical paths, so a symlinked directory matches its target. A sibling such as `work-old` does not match a `work` rule.
3. Check that the shell is interactive zsh, that `GITSHIFT_AUTO_SWITCH_DISABLE` is not `1`, and that `~/.zshrc` contains `source <(gitshift shell-init zsh)`.
4. Run `gitshift auto-switch "$PWD"` to see the error the hook hides.
5. After a `brew upgrade`, open a new shell: the old hook points at the previous version's path.

To remove a rule whose directory was deleted, pass the exact path from `gitshift auto list` to `gitshift auto remove`.

## Discovery misses or merges accounts

```bash
gh auth status --json hosts
gitshift discover --dry-run
```

Discovery includes every GitHub CLI user that is logged in successfully, active or not. SSH and GPG evidence is attached to an identity only when the match is unambiguous; otherwise it stays a separate result. Assign aliases with `--alias 'provider|host|user=ALIAS'` (quoted) and review the dry run before using `--overwrite`.

- `unknown selector`: the selector does not exactly match an identity key or suggested alias from this scan.
- `ambiguous selector`: more than one result has that suggested alias; use the full identity key.
- An alias collision: another account or pending account already uses it (case-insensitively).

## `configuration changed on disk; reload and retry`

Another GitShift process or an editor changed the file after this command read it. Run the command again.

## `timed out waiting for config lock` or `switch lock`

Make sure no other GitShift process is running (`pgrep -fl gitshift`). Then delete the lock file named in the error (`config.yaml.lock` or `config.yaml.switch.lock` next to your config) and retry.

## `unsupported configuration version`

The file was written by a newer GitShift, or its `config_version` was edited. GitShift refuses to load it and leaves it unchanged. Upgrade GitShift or restore a backup.

## A failed switch left mixed state

If the error includes `rollback also failed`, check each item in the [security model](../explanation/security-model.md#what-a-switch-changes): global and local Git config, `~/.ssh/config` (the previous version is in `~/.ssh/config.backup`), GPG settings and `gh auth status`. Then run a normal `gitshift switch ALIAS`.

## Reporting a bug

Include the GitShift version, operating system, the command with personal details removed, the exact error, and whether it ran inside a Git repository. Never include private keys or tokens.
