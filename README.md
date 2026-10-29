# GitShift

GitShift switches your Git identity between GitHub and GitLab accounts by alias. One command (`gitshift switch work`) sets the Git author name and email, the SSH key Git uses, optional GPG commit signing, and, for GitHub, the active GitHub CLI account. You can also have it switch automatically when you `cd` into a directory tree.

It supports github.com, GitHub Enterprise, gitlab.com and self-managed GitLab. It does not clone repositories or store provider access tokens.

## Install

On macOS with Homebrew:

```bash
brew tap techishthoughts-org/tap
brew trust techishthoughts-org/tap
brew install --cask techishthoughts-org/tap/gitshift
gitshift --version
```

Release archives for Linux and Windows, checksum verification and source builds are covered in [Install GitShift](docs/how-to/install.md).

## Quick start

Requirements: Git, an OpenSSH client and, for GitHub accounts, the [GitHub CLI](https://cli.github.com/) logged in to each account (`gh auth login`).

```bash
# Register two accounts with their existing SSH keys
gitshift add personal --name "Alex Example" --email alex@example.com \
  --username alex --ssh-key ~/.ssh/id_ed25519_personal
gitshift add work --name "Alex Example" --email alex@company.example \
  --username alex-company --ssh-key ~/.ssh/id_ed25519_work

# Check, then switch
gitshift switch work --validate
gitshift switch work
gitshift current

# Optional: switch automatically by directory (zsh)
gitshift auto add ~/dev/work work
gitshift auto add ~/dev/personal personal
gitshift auto default personal
echo 'source <(gitshift shell-init zsh)' >> ~/.zshrc
```

Run `gitshift` with no arguments to open the interactive terminal UI.

## Documentation

- [Getting started](docs/getting-started.md): a guided first setup.
- [How-to guides](docs/README.md#how-to-guides): install, manage accounts, switch, auto-switch, keys, GitLab and Enterprise, troubleshooting.
- [Reference](docs/reference/README.md): every command and the configuration file.
- [Explanation](docs/README.md#explanation): architecture, identity model, security model.
- [Contributing](CONTRIBUTING.md) and [changelog](CHANGELOG.md).
