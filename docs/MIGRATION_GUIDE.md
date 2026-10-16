# Migration and upgrades

## Upgrade safely

1. Back up `$HOME/.config/gitshift/config.yaml`, `~/.gitconfig`, and `~/.ssh/config`.
2. Download the replacement binary and verify it against `checksums.txt`.
3. Replace the old executable without deleting the configuration directory.
4. Run `gitshift --version`, `gitshift list`, and `gitshift current`.
5. Test one account with `gitshift switch ALIAS --validate`.

See [Installation](INSTALLATION.md) for platform-specific commands.

## Older account files

Accounts without `platform` use GitHub compatibility behavior. The legacy `github_username` field remains readable, but new and edited accounts should use:

```yaml
platform: github
username: alex
domain: github.com
```

Use the CLI to migrate a record:

```bash
gitshift update ALIAS --platform github --username USERNAME --domain github.com
```

Back up before hand-editing. Loading may secure file permissions and add/save missing timestamps. Do not remove fields you do not recognize merely because they are absent from the minimal examples.

## Moving between machines

Copying the YAML file is not sufficient. SSH private keys, GPG secret keys, provider registrations, GitHub CLI authentication, and enterprise host trust are separate. Transfer keys using your organization's approved method, repair paths with `gitshift update`, authenticate `gh`, then validate before switching.

## Downgrades

Older binaries may not understand fields written by a newer version. Restore the configuration backup made before upgrading rather than assuming the file is backward compatible.
