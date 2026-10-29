# Reference

## Commands

The command pages are generated from the CLI definitions by `make docs` and match `gitshift COMMAND --help`. Do not edit them by hand; change the command in `cmd/` and regenerate.

Start at the root page: [gitshift](commands/gitshift.md).

| Command | Purpose |
| --- | --- |
| [`add`](commands/gitshift_add.md) | Add an account |
| [`update`](commands/gitshift_update.md) | Change an account's fields |
| [`remove`](commands/gitshift_remove.md) | Remove an account (aliases `rm`, `delete`, `del`) |
| [`list`](commands/gitshift_list.md) | List accounts (alias `ls`) |
| [`current`](commands/gitshift_current.md) | Show the account the last switch applied (aliases `c`, `whoami`) |
| [`switch`](commands/gitshift_switch.md) | Apply an account (aliases `s`, `use`) |
| [`auto`](commands/gitshift_auto.md) | Manage directory rules: [`add`](commands/gitshift_auto_add.md), [`list`](commands/gitshift_auto_list.md), [`remove`](commands/gitshift_auto_remove.md), [`resolve`](commands/gitshift_auto_resolve.md), [`default`](commands/gitshift_auto_default.md) |
| [`auto-switch`](commands/gitshift_auto-switch.md) | Apply the rule or default for a directory; called by the hook |
| [`shell-init`](commands/gitshift_shell-init.md) | Print the zsh hook |
| [`discover`](commands/gitshift_discover.md) | Import identities from the GitHub CLI, SSH and GPG |
| [`ui`](commands/gitshift_ui.md) | Open the terminal UI (alias `tui`; also the default with no command) |
| [`ssh-keygen`](commands/gitshift_ssh-keygen.md) | Create an SSH key for an account |
| [`gpg-keygen`](commands/gitshift_gpg-keygen.md) | Create a GPG signing key for an account |
| [`ssh-test`](commands/gitshift_ssh-test.md) | Test SSH access |

`gitshift completion bash|zsh|fish|powershell` prints shell completion scripts.

Global flags: `--config PATH` selects the configuration file (see [configuration](configuration.md)). `--version`/`-v` on the root command prints the version, build time and Go version.

Exit status is 0 on success and 1 on any error. Human-readable output may change between versions; scripts should use `list --format json`, `current --json`, `current --short`, `auto list --json` and `auto resolve --json`.

## Configuration

- [Configuration file](configuration.md): location, every field, permissions, `--config`, environment variables.
