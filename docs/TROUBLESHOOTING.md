# Troubleshooting

## Inspect the account

```sh
gitshift list
gitshift current
gitshift update work
gitshift switch work --validate
```

## SSH failures

```sh
gitshift ssh-test work --verbose
ssh-add -l
ssh -G git@github.com
```

Confirm the configured private key exists, is owned by you, and has restrictive
permissions. Generate a new passwordless key with `gitshift ssh-keygen`, or
generate a protected key interactively with `ssh-keygen` and attach it using
`gitshift update --ssh-key`.

## GitHub CLI failures

```sh
gh auth status --hostname github.com
gh auth login --hostname github.com
gh auth switch --hostname github.com --user YOUR_LOGIN
```

GitLab accounts should not require `gh`. If the wrong hostname is used, check
the account's platform and domain.

## Configuration isolation

For diagnosis or tests, pass a separate file with `gitshift --config PATH ...`.
There are no `status`, `config`, `project`, or `github-token` commands.
