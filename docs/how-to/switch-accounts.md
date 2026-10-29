# Switch accounts

```bash
gitshift switch work
```

Aliases: `gitshift s work`, `gitshift use work`.

A switch applies the account to your whole user account, not only the current terminal. The [identity model](../explanation/identity-model.md) explains why.

## What a switch changes

In this order:

1. **SSH config** (only if the account has a key): rewrites the GitShift-managed block for the account's host in `~/.ssh/config`.
2. **Git identity**: sets global `user.name`, `user.email` and `core.sshCommand` (unset when the account has no key). If you run the command inside a Git repository, it sets the same three values in that repository's local config.
3. **Commit signing**: applies the account's signing to global `gpg.format`, `user.signingkey`, `commit.gpgsign` and `tag.gpgsign`. With `ssh` or `gpg` signing, both `*.gpgsign` values become `true`; with signing off they become `false`, and `user.signingkey` and `gpg.format` are unset unless the account has a GPG key. See [sign commits and tags](keys.md#sign-commits-and-tags).
4. **GitHub CLI** (GitHub accounts only): `gh auth switch --hostname HOST --user USERNAME`.
5. **Check**: confirms global `user.name`/`user.email` and tests SSH authentication with the account's key.
6. **Record**: saves the alias as `current_account` and marks the account as the default.

The full list of side effects, including backups, is in the [security model](../explanation/security-model.md).

## Check before switching

```bash
gitshift switch work --validate   # or -V
```

`--validate` changes nothing. It checks what the real switch needs: the name and email, that the SSH key exists with owner-only permissions and authenticates to the account's host, for GitHub accounts that the GitHub CLI is logged in as the account's username, and that the account's GPG secret key is in your keyring. With `--validate`, `--force` and `--skip-validation` are ignored.

## When a check fails: `--skip-validation` and `--force`

Use these only after you understand the failing check.

| | Normal | `--skip-validation` (`-s`) | `--force` (`-f`) |
| --- | --- | --- | --- |
| Account fields, platform, key file exists | required | required | required |
| SSH authentication test before changes | required | skipped | failure is a warning |
| Private key mode `600`, SSH config write | required | required | required |
| Git config write | required | required | required |
| GPG settings | required | required | failure is a warning |
| GitHub CLI installed and an active user to restore | required | required | may be missing |
| `gh auth switch` | required | required | failure is a warning |
| Final Git and SSH check | required | skipped | skipped |
| Rollback when a required step fails | yes | yes | yes |

Typical uses:

- `--skip-validation`: you are offline, or the provider's SSH endpoint is unreachable, and you still want the identity applied.
- `--force`: the GitHub CLI is not installed or not logged in to this user, or GPG is broken, and you accept a partial switch.

With `--force` the account is recorded as current even when a tolerated step failed, so `gitshift current` may not match what Git, GPG or the GitHub CLI actually use. Check with the commands under [verify](#verify-the-result).

## What happens when a switch fails

Before changing anything, GitShift saves:

- the contents and mode of `~/.ssh/config` (or that it did not exist),
- the global values, and the repository-local values if you are in a repository, of `user.name`, `user.email`, `core.sshCommand`, `user.signingkey`, `commit.gpgsign`, `tag.gpgsign` and `gpg.format`,
- the active GitHub CLI user for the account's host.

If a required step fails, GitShift restores those in reverse order and leaves `current_account` unchanged. If the restore itself fails, the error says `rollback also failed` and names what could not be restored.

Rollback cannot help if the process is killed, the machine loses power, or another program changes the same settings at the same time. Only one GitShift switch runs at a time (it holds `config.yaml.switch.lock`), but other tools are not coordinated.

## Verify the result

```bash
gitshift current
git config --show-origin --get-regexp '^(user\.|core\.sshcommand|commit\.gpgsign|tag\.gpgsign)'
gh auth status --hostname github.com
gitshift ssh-test work
```

Inside a repository, a local value overrides the global one. `--show-origin` shows which file each value comes from.

## Repository-local identity

Switching inside a repository writes `user.name`, `user.email` and `core.sshCommand` into that repository's `.git/config`. Those values stay after you switch to a different account from outside that repository, so the repository keeps the identity it was switched in. To drop them:

```bash
git config --local --unset user.name
git config --local --unset user.email
git config --local --unset core.sshCommand
```

GPG settings are only ever written globally.
