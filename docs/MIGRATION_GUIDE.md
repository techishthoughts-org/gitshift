# Migration notes

Older configuration may use `github_username`; it remains readable. New
scripts should use `--username` with `--platform` and, when required,
`--domain`.

Legacy documentation mentioned `add-github`, `status`, `config`, `project`,
`github-token`, `ssh-keys`, and a future nested command hierarchy. Those
commands were proposals and are not implemented. Use `add`, `current`, `list`,
`update`, `switch`, `ssh-keygen`, and `ssh-test`.

Before changing versions, back up the YAML configuration with ordinary
filesystem tools, run `gitshift list`, and test important accounts with
`gitshift switch ALIAS --validate`.

The historical `scripts/sync-github-tokens.sh` no longer synchronizes tokens.
It selects an existing GitHub CLI account by `--hostname` and `--user` without
reading token plaintext. Authenticate with `gh auth login` first.
