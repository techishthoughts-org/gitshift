# Configuration reference

This reference applies to the current source. The legacy public binary ignores `--config` and was observed creating the default YAML as `0644`; see [Installation](INSTALLATION.md) before using a download.

## Location and permissions

The default file is `$HOME/.config/gitshift/config.yaml`. On Windows, this normally resolves to `%USERPROFILE%\.config\gitshift\config.yaml`. GitShift creates the directory with owner-only permissions and secures the file as `0600` where the operating system supports Unix modes.

Pass `--config PATH` before the subcommand to use an exact alternate file:

```bash
gitshift --config "$HOME/.config/gitshift/team-a.yaml" list
```

The path is exact: relative paths resolve from the working directory and a quoted `~` is not shell-expanded. Loading creates a missing file; loading an old file may add missing timestamps and save it.

## Prefer CLI changes

Use `add`, `update`, `remove`, and `auto` commands instead of hand-editing YAML. GitShift writes through a same-directory temporary file and lock and refuses to overwrite a configuration that changed since it was loaded.

A minimal illustrative file looks like this:

```yaml
accounts:
  personal:
    alias: personal
    name: Alex Example
    email: alex@example.com
    ssh_key_path: /home/alex/.ssh/id_ed25519_personal
    username: alex
    platform: github
    domain: github.com
    description: Personal account
    is_default: true
    status: active
current_account: personal
config_version: "1.1.0"
auto_switch:
  rules:
    - path: /home/alex/dev/personal
      account: personal
      created_at: 2026-10-16T12:00:00Z
```

Use an absolute SSH key path in hand-written YAML; shell variables are not expanded by YAML itself.

## Important fields

| Field | Meaning |
| --- | --- |
| `accounts` | Map keyed by account alias |
| `current_account` | Selected/default alias; add, default changes, removal, discovery, and switching can update it |
| `name`, `email` | Git author identity |
| `platform` | `github` or `gitlab` |
| `username` | Provider username |
| `domain` | Provider hostname; defaults to `github.com` or `gitlab.com` |
| `api_endpoint` | Optional recorded API URL; current switching does not consume it |
| `ssh_key_path` | Private SSH key path |
| `gpg_key_id`, `gpg_enabled` | Commit-signing configuration |
| `github_username` | Deprecated compatibility field; use `username` |
| `auto_switch.rules` | Canonical directory roots mapped to existing account aliases |

New files use `config_version: "1.1.0"`. GitShift migrates only an omitted version or the known legacy `1.0.0`; malformed and future versions are rejected without rewriting or changing the permissions of the file. The file may contain status, validation, isolation, timestamp, `global_git_config`, and `auto_detect` metadata. Some serialized fields are retained for compatibility but are not honored by current commands: notably, `global_git_config: false` does not prevent global Git writes and `auto_detect` remains inert. Automatic switching is enabled only by explicit `auto_switch.rules` plus the opt-in zsh hook. Unknown or internal-looking fields are not a promise of a public CLI feature.

## Backups and concurrent edits

Back up the file before manual changes. Writes use a two-second lock wait and stale-content detection. Switches also hold `config.yaml.switch.lock` across external mutations. If GitShift reports `configuration changed on disk; reload and retry`, rerun the command so it loads the current file. If it times out on either lock, ensure no GitShift process is running before deleting that lock file.

The configuration is plaintext. It stores identity metadata and paths, not provider access tokens. Treat it as private because names, emails, usernames, and filesystem layout can be sensitive.

See [the maintained example](../examples/config-multi-platform.yaml) for multiple providers.
