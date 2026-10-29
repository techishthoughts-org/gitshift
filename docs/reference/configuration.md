# Configuration file

GitShift keeps all its state in one YAML file. Prefer `add`, `update`, `remove` and `auto` over editing it by hand; they validate input, and they refuse to overwrite a file that changed after they read it.

## Location

| | Path |
| --- | --- |
| Default | `$HOME/.config/gitshift/config.yaml` (Windows: `%USERPROFILE%\.config\gitshift\config.yaml`) |
| Override | `gitshift --config PATH COMMAND` |

`--config` is an exact file path. A relative path is resolved from the current directory, and a quoted `~` is not expanded. To make the zsh hook use the same file, pass it to `shell-init`: `source <(gitshift --config PATH shell-init zsh)`.

Most commands create the file if it is missing. Read-only commands (`current --short`, `auto list`, `auto resolve`, `auto default` without arguments, `discover --dry-run`) treat a missing file as empty and create nothing. The terminal UI refuses to start without a terminal before touching the file.

## Permissions and writes

- The directory is created with mode `0700` when GitShift creates it. An existing directory's mode is never changed.
- The file is set to `0600` on every normal load, and written through an owner-only temporary file that replaces it atomically.
- Writes take `config.yaml.lock` next to the file for up to two seconds. A switch also holds `config.yaml.switch.lock` for the whole transaction.
- If the file changed since it was read, the write fails with `configuration changed on disk; reload and retry`.

The file holds names, emails, usernames, hostnames and key paths. It never holds tokens, passphrases or key material, but treat it as private.

## Example

A complete, valid multi-platform example is in [examples/config-multi-platform.yaml](../../examples/config-multi-platform.yaml). A minimal file:

```yaml
config_version: "1.1.0"
current_account: work
accounts:
  work:
    alias: work
    name: Alex Example
    email: alex@company.example
    username: alex-company
    github_username: alex-company
    platform: github
    domain: github.com
    ssh_key_path: /Users/alex/.ssh/id_ed25519_work
    is_default: true
    status: active
auto_switch:
  default_account: work
  rules:
    - path: /Users/alex/dev/work
      account: work
      created_at: 2026-10-16T12:00:00Z
```

YAML does not expand `~` or variables. Use absolute paths when editing by hand.

## Top-level fields

| Field | Type | Meaning |
| --- | --- | --- |
| `config_version` | string | `1.1.0`. A missing value or `1.0.0` is upgraded to `1.1.0`; any other value is rejected and the file is left as it is |
| `accounts` | map | Accounts keyed by alias. The key must equal the account's `alias` |
| `pending_accounts` | map | Incomplete results saved by `discover`; see below |
| `current_account` | string | Alias the last successful switch applied. Set only by `switch`, the UI, auto-switch and first-run `discover`; cleared when that account is removed |
| `auto_switch.rules` | list | Directory rules; see below |
| `auto_switch.default_account` | string | Alias auto-switch applies when no rule matches. Empty: leave the current account alone. Set with `gitshift auto default` |
| `global_git_config` | bool | Written as `true`; not read by any command. Setting `false` does not stop global Git writes |
| `auto_detect` | bool | Written as `true`; not read by any command |

## Account fields

| Field | Meaning |
| --- | --- |
| `alias` | Account name: letters, digits, `.`, `_`, `-`, starting with a letter or digit, at most 64 characters |
| `name` | Git `user.name` |
| `email` | Git `user.email` |
| `username` | Provider username. For `github`, at most 39 letters, digits and single hyphens |
| `github_username` | Older field kept for compatibility. `add` fills it for GitHub accounts. Used when `username` is empty |
| `platform` | `github` or `gitlab`. Empty means `github` |
| `domain` | Provider hostname, no scheme, port or path. Empty means `github.com` or `gitlab.com` |
| `api_endpoint` | Optional `https://` URL without credentials. Stored only; no command calls it |
| `ssh_key_path` | Private key used for SSH and `core.sshCommand`. Empty: no SSH changes, and `core.sshCommand` is unset on switch |
| `description` | Free text |
| `is_default` | The default account. GitShift keeps it on one account: the first one added, or the one last set with `--default`. Switching does not change it. See the [identity model](../explanation/identity-model.md) |
| `gpg_key_id` | GPG key used for `user.signingkey`. Empty: switching turns signing off |
| `gpg_enabled` | With a key, sets `commit.gpgsign` and `tag.gpgsign` to `true`; otherwise `false` |
| `gpg_key_type`, `gpg_key_size`, `gpg_key_fingerprint`, `gpg_key_expiry` | Informational details of the attached GPG key |
| `created_at`, `last_used` | Timestamps; `last_used` is updated by each switch |
| `status` | `active` for normal accounts; informational |
| `missing_fields`, `last_validation`, `validation_errors`, `account_metadata` | Informational; not used to make decisions |
| `isolation_level`, `isolation_metadata`, `token_path`, `ssh_socket_path` | Legacy fields kept so older files still load. New accounts no longer write `isolation_metadata`, and no command reads these fields. GitShift stores no tokens and runs no per-account agent |

## Directory rules

Each entry in `auto_switch.rules`:

| Field | Meaning |
| --- | --- |
| `path` | Absolute physical path (symlinks resolved) of the directory tree |
| `account` | Alias of an existing account |
| `created_at` | When the rule was first added |

The deepest matching path wins. Removing an account removes its rules and clears `default_account` if it pointed there. See [auto-switch](../how-to/auto-switch.md).

## Pending accounts

`discover` saves results it could not complete under `pending_accounts`, keyed by alias, with `github_username`, `partial_data` (discovered `name`, `email`, `ssh_key_path`), `missing_fields`, `source`, `confidence` and `created_at`. Pending accounts cannot be switched to. `list` shows them. Re-create one as a real account with `gitshift add ALIAS ...`.

## Environment variables

GitShift reads no configuration from the environment. These variables change behaviour:

| Variable | Effect |
| --- | --- |
| `GITSHIFT_AUTO_SWITCH_DISABLE=1` | The zsh hook does nothing in this shell. Only the exact value `1` counts |
| `NO_COLOR` | The terminal UI renders without colour when set to any value |
| `HOME` | Locates `~/.config/gitshift`, `~/.ssh` and `~/.gnupg` |
| `SSH_AUTH_SOCK` | `ssh-test` uses it to report whether the key is loaded in your agent |

`GITHUB_TOKEN` is not used. GitHub authentication comes only from the GitHub CLI.
