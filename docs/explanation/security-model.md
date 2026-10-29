# Security model

GitShift coordinates tools that each own part of your identity: Git, OpenSSH, GnuPG and the GitHub CLI. It changes their per-user configuration in place. This page lists everything it changes, how it protects your existing files, what rollback can and cannot undo, and what it never stores.

## What a switch changes

| State | Change |
| --- | --- |
| `~/.gitconfig` | `user.name`, `user.email`, `core.sshCommand` (unset if the account has no key), `gpg.format`, `user.signingkey` (both unset when the account signs with nothing and has no GPG key), `commit.gpgsign`, `tag.gpgsign` |
| `.git/config` of the current repository | `user.name`, `user.email`, `core.sshCommand`, only when you run the switch inside a repository |
| `~/.ssh/config` | the GitShift-managed block for the account's host, when the account has a key; see [below](#ssh-config-handling) |
| `~/.ssh/config.backup`, `~/.ssh/config.gitshift-orig` | backups written before each `~/.ssh/config` change |
| GitHub CLI | active user for the account's host (`gh auth switch`), GitHub accounts only |
| GitShift config | `current_account` and the account's `last_used` |
| `config.yaml.lock`, `config.yaml.switch.lock` | short-lived lock files next to the config |

A GitLab switch changes no provider login. Nothing is changed on the provider's servers. Removing an account or uninstalling GitShift reverts none of this.

## Other commands that change state

| Command | Changes |
| --- | --- |
| `add`, `update`, `remove`, `auto add/remove/default` | the GitShift config only |
| `discover` | the GitShift config (never with `--dry-run`). On a fresh interactive install it also runs a normal switch |
| `ssh-keygen` | new unencrypted key pair `~/.ssh/id_<type>_<alias>`, `ssh-add` to your agent, appends `github.com` and `gitlab.com` host keys to `~/.ssh/known_hosts`, the clipboard, the account's `ssh_key_path`; with `--add-to-github`, a new key on the account's own GitHub user (its `gh` login, without changing the active one) |
| `gpg-keygen` | new unencrypted secret key in your GPG keyring, `~/.gnupg/gitshift-<alias>-public.asc`, the clipboard, the account's GPG fields |
| `ssh-test`, `list`, `current`, `auto list`, `auto resolve`, `shell-init` | nothing outside the config; `ssh-test`, `list` and `current` (without `--short`) may still normalize the config file |

## SSH config handling

Each switch for a host rewrites `~/.ssh/config` like this:

1. If the file exists, its current content is copied to `~/.ssh/config.backup` (replaced on every switch). The first time GitShift ever writes it, the same content is also saved to `~/.ssh/config.gitshift-orig`, which is never overwritten afterwards.
2. GitShift removes its previous managed block for that host, delimited by `# >>> gitshift managed: HOST` and `# <<< gitshift managed: HOST`.
3. Your own blocks whose `Host` line names exactly that host are disabled, because SSH adds `IdentityFile` lines from every matching block and would otherwise offer the wrong key:
   - a block for only that host is commented out line by line with the prefix `# gitshift-disabled: `, so you can restore it;
   - a block for several hosts (`Host github.com example.org`) keeps its other hosts and loses only that one.
4. Global options, the lines before the first `Host` or `Match`, stay at the top. The new managed block follows them, then your remaining blocks.

The managed block is:

```text
# >>> gitshift managed: github.com
# GitHub account: work
Host github.com
    HostName github.com
    User git
    IdentityFile "/Users/alex/.ssh/id_ed25519_work"
    IdentitiesOnly yes

# <<< gitshift managed: github.com
```

The new file is written to a temporary file with mode `0600` and moved into place. GitShift refuses to use a private key that group or other users can access (it accepts `600` and `400`, like OpenSSH), and never changes the permissions of your key files.

## Git SSH command

For an account with a key, GitShift sets:

```text
core.sshCommand = ssh -i '/path/to/key' -o IdentitiesOnly=yes
```

`-i` puts the account's key first, and `IdentitiesOnly=yes` stops ssh from offering other keys held by your agent. Your `~/.ssh/config` still applies, so host aliases, `Port`, `ProxyJump` and the managed block keep working, and an agent that holds this key (for example, with a passphrase) is used.

Two consequences to know:

- A key you list with `IdentityFile` in a matching `Host` block is offered after the account's key. If the account's key is not registered on the provider, the provider may accept that other key instead, as a different user. `gitshift switch --validate` and `ssh-test` check the account's key on its own.
- The setting is global: it applies to every SSH remote while the account is active.

## Backups and rollback

Before changing anything, a switch snapshots `~/.ssh/config` (content and mode, or its absence), the six Git keys above in global scope and, inside a repository, in local scope, and the active GitHub CLI user for the host. When a required step fails, it restores them in reverse order and leaves `current_account` unchanged.

Rollback does not cover:

- the process being killed, a crash or power loss in the middle of a switch;
- steps that `--force` tolerated: the switch is recorded as successful with those parts unapplied;
- changes made by other programs at the same time (GitShift's lock only serializes GitShift switches);
- `~/.ssh/config.backup`, which already holds the pre-switch file;
- a failure of the rollback itself, which is reported as `rollback also failed` with the parts that could not be restored.

For a manual recovery, `~/.ssh/config.backup` is the file before the last switch, and `~/.ssh/config.gitshift-orig` the file before GitShift first touched it.

## Directory hook

The zsh hook is opt-in and printed by `gitshift shell-init zsh`; GitShift never edits shell startup files. It runs only in interactive shells, quotes `$PWD` as one argument, calls the binary by absolute path, guards against re-entry, and can be disabled with `GITSHIFT_AUTO_SWITCH_DISABLE=1`. It runs normal switches only, never `--force`. Each switch it triggers is global, so a `cd` in one terminal changes the identity for all of them.

## What is never stored

- **Provider tokens.** GitHub authentication stays in the GitHub CLI's own credential storage. During `discover`, GitShift asks `gh auth token` for one user's token, keeps it in memory, and passes it only through the environment of one `gh api user` child process. It is never printed or written. `GITHUB_TOKEN` is not read. GitLab credentials are not touched.
- **Passphrases.** GitShift never asks for one. That is why `ssh-keygen` and `gpg-keygen` create keys without a passphrase: passing one on a command line or in a batch file would expose it.
- **Key material.** The config stores key paths and GPG key IDs and fingerprints, never private keys.

The config does store names, emails, usernames, hostnames, key paths and directory paths. It is mode `0600`; do not commit it.

## Keys GitShift generates

Generated keys are unencrypted, so anyone who can read the file can use it. When your policy requires passphrases or hardware-backed keys, create the key with `ssh-keygen` or `gpg` yourself and attach it (see [keys](../how-to/keys.md)), keeping in mind that Git will not use your agent for it.

The host keys `ssh-keygen` appends for `github.com` and `gitlab.com` are built into GitShift. Compare them with the fingerprints GitHub and GitLab publish. Verify enterprise host keys through your organization.

## Discovery probes

Discovery tests each local SSH key against the provider hosts known to the GitHub CLI. Each probe ignores `~/.ssh/config` and the agent and offers only the key being inspected. It authenticates but runs no command and uploads nothing.

## Reporting a vulnerability

Report suspected vulnerabilities privately to the maintainers, not in a public issue. Never include private keys or tokens.
