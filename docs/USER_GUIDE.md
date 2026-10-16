# User guide

## Accounts

Create an account non-interactively by supplying its alias, name, email,
platform, username, and optional SSH key. Supported platform values are
`github` and `gitlab`; a custom domain may be supplied with `--domain`.

```sh
gitshift add work --name "Work User" --email work@example.com \
  --platform gitlab --domain gitlab.example.com --username work-user \
  --ssh-key "$HOME/.ssh/id_ed25519_work" --non-interactive
```

Use `--github-username` only for backward compatibility. An optional
`--api-endpoint` can record a provider API endpoint.

```sh
gitshift list
gitshift list --format json
gitshift current
gitshift update work --email new-work@example.com
gitshift remove work
```

## Switching

```sh
gitshift switch work --validate
gitshift switch work
```

Switching updates local Git/SSH/GPG state. GitHub accounts may also run `gh
auth switch --hostname DOMAIN --user USER`. GitLab and custom hosts never use
the GitHub CLI.

## Keys and discovery

```sh
gitshift discover --dry-run
gitshift discover --auto-import
gitshift ssh-keygen work --email work@example.com
gitshift gpg-keygen work --email work@example.com
gitshift ssh-test work
gitshift ssh-test --all
```

The built-in generators intentionally create passwordless keys. For a
passphrase-protected key, use the native `ssh-keygen` or GPG interactive UI,
then attach the resulting key to the account.
