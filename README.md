# gitshift

`gitshift` is a Go CLI for keeping several Git identities and switching the
Git, SSH, GPG, and (for GitHub accounts) GitHub CLI settings used by the local
machine.

## Status

The implemented account platforms are GitHub, GitLab, and custom SSH Git
hosts. Bitbucket is not implemented. Switching changes user-level state, so
review an account with `gitshift update ALIAS` and use `gitshift switch
ALIAS --validate` before switching important environments.

## Install

```sh
go install github.com/techishthoughts/gitshift@latest
```

Or build the checkout with `make build`.

## Quick start

```sh
# GitHub
gitshift add personal \
  --name "Your Name" \
  --email you@example.com \
  --platform github \
  --username your-login \
  --ssh-key "$HOME/.ssh/id_ed25519"

# GitLab or a self-hosted instance
gitshift add work \
  --name "Your Name" \
  --email you@company.example \
  --platform gitlab \
  --domain gitlab.company.example \
  --username your-login \
  --ssh-key "$HOME/.ssh/id_ed25519_work"

gitshift list
gitshift switch work --validate
gitshift switch work
gitshift current
```

`--github-username` remains an alias-compatible input for older scripts.
Prefer `--username` for new configuration. `--api-endpoint` records a custom
provider API endpoint.

## Commands

- `add`, `list`, `current`, `update`, and `remove` manage accounts.
- `switch` applies an account; GitHub accounts may also switch `gh` with the
  configured domain and username. GitLab/custom accounts use SSH and do not
  invoke `gh`.
- `discover` imports identities found in local SSH and GPG configuration.
- `ssh-keygen` creates a passwordless key without putting a passphrase in a
  process argument. Create protected keys interactively with `ssh-keygen` and
  attach them using `update --ssh-key`.
- `gpg-keygen` creates a passwordless batch key. Use GPG interactively when a
  protected key is required.
- `ssh-test` checks an account's SSH setup.

There are no `status`, `config`, `project`, `github-token`, `ssh-keys`, or
`add-github` commands.

See [the documentation](docs/README.md) for configuration, security, and
troubleshooting details.
