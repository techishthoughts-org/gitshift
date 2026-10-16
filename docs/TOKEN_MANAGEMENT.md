# Credentials and GitHub CLI

GitShift does not collect or store provider access tokens. GitHub authentication remains owned by GitHub CLI and its configured credential storage; GitLab authentication remains outside GitShift.

## Prepare GitHub accounts

Install [GitHub CLI](https://cli.github.com/), then authenticate each identity and hostname:

```bash
gh auth login --hostname github.com
gh auth status --hostname github.com
```

Repeat login for additional accounts. During `gitshift switch ALIAS`, a GitHub profile selects the matching username with:

```bash
gh auth switch --hostname HOSTNAME --user USERNAME
```

`gitshift discover --dry-run` enumerates all users on all GitHub CLI hosts, not only the active user. To enrich a discovered profile it asks GitHub CLI for that explicit host/user credential and passes it only to a short-lived `gh api` process. It does not switch accounts, print the credential, or write it to GitShift configuration.

If `gh` is absent, the hostname has no active account, or the username has not been authenticated, a normal GitHub switch fails. `--force` can continue with a warning but does not create valid authentication. Fix the GitHub CLI login rather than putting a token in GitShift's YAML.

## GitLab and HTTPS credentials

GitShift does not invoke `glab` and does not alter a Git credential helper. Configure GitLab tokens, browser/device login, and HTTPS credentials with the provider and Git tooling you already use.

## Revoke access

Removing a GitShift account does not log out, revoke a token, or remove an SSH key from a provider. Use `gh auth logout --hostname HOSTNAME --user USERNAME` or the provider's security settings when access must be revoked. Delete local private keys only after confirming they are no longer needed.
