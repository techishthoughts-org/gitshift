# GitShift

GitShift switches the Git identity, SSH key, and optional GPG signing settings used for different GitHub and GitLab accounts. Accounts are stored under memorable aliases such as `work` and `personal`.

The current source supports GitHub, GitHub Enterprise, GitLab.com, and self-managed GitLab. GitShift does not clone repositories, manage arbitrary Git hosts, or store provider access tokens.

> **Release compatibility:** these `main`-branch docs describe the current source. The latest public assets were verified on 2026-10-16 and still contain an older GitHub-only CLI, report embedded version `v0.1.0`, ignore `--config`, and were observed creating the default YAML as `0644` on macOS. Build from source for the documented product until replacement assets are published and verified.

## Install the current source

Go 1.26 and Git are required:

```bash
git clone https://github.com/techishthoughts-org/gitshift.git
cd gitshift
make test
make build
./gitshift --version
make install
```

`make install` writes to `${GOPATH:-$HOME/go}/bin`; ensure that directory is on `PATH`. Direct `go install` is not supported by the current repository/module layout.

## Download the legacy release

Download the binary for your operating system from [GitHub Releases](https://github.com/techishthoughts-org/gitshift/releases/latest):

| System | Processor | Asset |
| --- | --- | --- |
| Linux | x86-64 | `gitshift-linux-amd64` |
| Linux | ARM64 | `gitshift-linux-arm64` |
| macOS | Intel | `gitshift-darwin-amd64` |
| macOS | Apple silicon | `gitshift-darwin-arm64` |
| Windows | x86-64 | `gitshift-windows-amd64.exe` |

Linux x86-64 example:

```bash
curl -LO https://github.com/techishthoughts-org/gitshift/releases/latest/download/gitshift-linux-amd64
curl -LO https://github.com/techishthoughts-org/gitshift/releases/latest/download/checksums.txt
grep 'gitshift-linux-amd64$' checksums.txt | sha256sum --check
install -Dm755 gitshift-linux-amd64 "$HOME/.local/bin/gitshift"
gitshift --version
```

These downloads are legacy GitHub-only builds. After first use, run `chmod 600 "$HOME/.config/gitshift/config.yaml"`; do not rely on `--config`. See the [installation guide](docs/INSTALLATION.md) for safe legacy usage, macOS, Windows, checksums, upgrades, and uninstallation.

## Requirements

- Git
- An OpenSSH client when using SSH keys
- [GitHub CLI](https://cli.github.com/) installed and authenticated for GitHub account switching
- GnuPG only when using commit signing

## Quick start

First authenticate every GitHub identity with GitHub CLI. GitLab identities do not use `gh`.

```bash
gh auth login --hostname github.com

gitshift add personal \
  --name "Alex Example" \
  --email alex@example.com \
  --github-username alex \
  --ssh-key "$HOME/.ssh/id_ed25519_personal"

gitshift list
gitshift switch personal --validate
gitshift switch personal
gitshift current
```

`switch --validate` checks the account before changing anything. A successful switch updates global Git identity, the current repository's local Git settings when run inside a repository, SSH configuration, optional GPG settings, and the active GitHub CLI account for GitHub profiles. Read [Security and side effects](docs/SECURITY.md) before using `--force` or `--skip-validation`.

To switch automatically when entering directory trees, add explicit rules and opt in from zsh:

```bash
gitshift auto add "$HOME/dev/personal" personal
gitshift auto add "$HOME/dev/work" work
gitshift auto list
printf '\nsource <(gitshift shell-init zsh)\n' >> ~/.zshrc
```

Use the aliases shown by `gitshift list`. Rules only accept existing directories and existing aliases. Restart zsh or source the hook once. Set `GITSHIFT_AUTO_SWITCH_DISABLE=1` to disable the hook for a shell; other values do not disable it.

## Commands

| Command | Purpose |
| --- | --- |
| `add ALIAS` | Add an account interactively or with flags |
| `update ALIAS` | Change an existing account |
| `list` | List accounts; supports `--format table` and `--format json` |
| `current` | Show the selected account; supports `--json` |
| `switch ALIAS` | Apply an account, optionally with `--validate` |
| `auto add/list/remove/resolve` | Manage and inspect directory rules |
| `auto-switch [PATH]` | Apply the matching rule; normally called by the hook |
| `shell-init zsh` | Print the opt-in zsh hook without editing shell files |
| `remove ALIAS` | Remove an account from GitShift |
| `discover` | Inspect all GitHub CLI identities plus local SSH/GPG metadata |
| `ssh-keygen ALIAS` | Generate a passwordless SSH key for an account |
| `gpg-keygen ALIAS` | Generate and optionally enable a GPG signing key |
| `ssh-test [ALIAS]` | Test SSH authentication for one or all accounts |
| `completion SHELL` | Generate shell completion through Cobra |

Use `gitshift COMMAND --help` for the authoritative flags in your installed version.

To name multiple discovered accounts yourself, preview their stable identities and assign one alias per account:

```bash
gitshift discover --dry-run
gitshift discover \
  --alias 'github|github.com|thukabjj=personal' \
  --alias 'github|github.com|arthurcosta-marks-and-spencer=work'
```

Alternatively, `gitshift discover --prompt-aliases` asks only for accounts that do not already have an explicit `--alias`; pressing Enter accepts the suggestion. Add `--dry-run` to preview without writing.

## Documentation

- [Installation and upgrades](docs/INSTALLATION.md)
- [User guide](docs/USER_GUIDE.md)
- [Command reference](docs/COMMAND_REFERENCE.md)
- [Configuration reference](docs/CONFIGURATION.md)
- [GitHub and GitLab support](docs/MULTI_PLATFORM_SUPPORT.md)
- [Credentials and GitHub CLI](docs/TOKEN_MANAGEMENT.md)
- [Security and side effects](docs/SECURITY.md)
- [Troubleshooting](docs/TROUBLESHOOTING.md)
- [Contributing](docs/CONTRIBUTING.md)
