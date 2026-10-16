# Security and side effects

This guide describes the current source. The published legacy binary does not provide all of these configuration protections; see [Installation](INSTALLATION.md).

GitShift coordinates tools that each own part of your developer identity. Review the effects before the first switch.

## What a switch can change

| State | Change |
| --- | --- |
| Global Git config | `user.name`, `user.email`, SSH command, and optional GPG signing |
| Current repository | Local identity and SSH command when run inside a Git repository; signing settings remain global |
| SSH config | Managed host entries and identity selection in `~/.ssh/config`; writes may create `.bak`/`.backup` copies |
| GitHub CLI | Active account for the configured GitHub hostname |
| GitShift config | Selected/default account and last-used timestamp |

GitLab switches do not change a provider CLI login. Removing GitShift or an account does not revert any external state.

## Safer operation

1. Back up `~/.gitconfig`, `~/.ssh/config`, and `$HOME/.config/gitshift/config.yaml`.
2. Keep private keys owner-readable only, normally mode `0600`.
3. Run `gitshift switch ALIAS --validate` to check account and SSH readiness; separately check GPG and `gh` because validate-only mode does not.
4. Verify with `git config --show-origin --get-regexp '^(user\.|core\.sshCommand|commit\.gpgsign)'`, `gitshift current`, and `gh auth status` for GitHub.
5. Avoid `--force` and `--skip-validation` unless you understand the failed check.

GitShift attempts rollback when a switch step returns an error. It cannot guarantee rollback after a kill, crash, power loss, or failure in the rollback itself. If a switch reports a rollback failure, inspect each state listed above before committing or pushing.

## Automatic switching and concurrency

The zsh hook is opt-in and runs only in interactive shells. It passes the current directory as one quoted argument to an absolute GitShift executable, has re-entry and same-directory guards, deduplicates repeated sourcing, and can be disabled with the exact setting `GITSHIFT_AUTO_SWITCH_DISABLE=1`. Rules store filesystem paths and account aliases, so treat them as private metadata.

Switching changes process-external, user-global state. GitShift serializes manual and automatic switch transactions across its own processes, but another terminal or tool can still change Git, SSH, GPG, or GitHub CLI state afterward. Concurrent work that requires different global identities is inherently unsafe; prefer repository-local isolation or separate OS users/containers for that workload.

## Key generation

`ssh-keygen` and `gpg-keygen` create passwordless keys for unattended use. This is convenient but weaker if the private key is copied. Generate protected keys with the native tools when your policy requires passphrases or hardware-backed keys, then attach their identifiers to GitShift.

SSH key generation can write private/public key files, add a key to `ssh-agent`, append provider keys to `known_hosts`, copy a public key to the clipboard, and optionally upload it through GitHub CLI. GPG generation can create a secret key and export its public key. Review command output and backups before using `--force` to replace paths.

For GPG, `--force` replaces the key associated with the GitShift account; it does not erase the previously generated secret key from the GPG keyring.

## Host trust

GitShift may install hard-coded host keys for `github.com` and `gitlab.com`. Verify fingerprints against the provider's current published values before relying on them. Enterprise host keys must be verified through your organization's trusted channel. Do not work around an unexpected host-key warning until the fingerprint is independently confirmed.

## Configuration privacy

The YAML file contains account aliases, names, emails, usernames, domains, and key paths. It should not contain access tokens, but it is still private metadata. GitShift uses owner-only permissions and atomic replacement where supported. Do not commit it to a repository.

Report suspected vulnerabilities privately to the repository maintainers rather than opening an issue containing credentials or exploit details.
