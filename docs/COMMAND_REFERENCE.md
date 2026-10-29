# Command reference

Syntax is `gitshift [--config PATH] [COMMAND]`. With no command, GitShift opens
the terminal UI. Run `gitshift COMMAND --help` for the installed version.

This is the current-source command set. The latest public binary is older; see [Installation](INSTALLATION.md).

## Account lifecycle

### `gitshift` / `ui [--inline]`

Running `gitshift` without a subcommand opens the interactive terminal UI.
The explicit `ui` command and its `tui` alias do the same. The UI requires TTY
input and output and fails before configuration initialization when either is redirected.
The default uses the alternate screen; `--inline` renders in the current screen
buffer. `--config PATH` is honored.

The UI supports the dashboard/current account, account list/details and
validated add/edit, confirmed normal switching and removal, directory-rule
list/add/remove, and read-only discovery followed by selected atomic additive
import. It never exposes provider tokens. Force switching, skipped validation,
and key generation remain CLI-only.
Quit requests are deferred while a confirmed mutation is running, until the
operation and configuration refresh complete. Directory rules still require
the separately installed `shell-init zsh` hook.

### `add ALIAS`

Adds an account. Without all required values it prompts interactively; `--non-interactive` disables prompts.

| Flag | Meaning |
| --- | --- |
| `--name`, `-n` | Git author name |
| `--email`, `-e` | Git author email |
| `--username` | Provider username |
| `--github-username`, `-g` | Legacy username spelling |
| `--platform` | `github` or `gitlab`; default `github` |
| `--domain` | Provider hostname |
| `--api-endpoint` | Recorded enterprise API URL |
| `--ssh-key`, `-k` | Existing private-key path |
| `--description`, `-d` | Optional label |
| `--default` | Make this the default account |
| `--enable-gpg` | Enable signing; requires an associated key |
| `--non-interactive` | Fail instead of prompting for missing data |

### `update ALIAS`

Changes only supplied fields. Flags are `--name`, `--email`, `--username`, legacy `--github-username`, `--platform`, `--domain`, `--api-endpoint`, `--ssh-key`, `--description`, `--default`, and `--auto-gpg`. An explicitly empty domain restores the provider default; an explicitly empty API endpoint clears it. `--auto-gpg` finds a local key by the account email.

### `remove ALIAS`

Aliases: `rm`, `delete`, `del`. Prompts before deleting unless `--force`/`-f` is used. Removal only changes GitShift configuration; it does not delete keys, revoke provider access, or switch external state.

## Inspect accounts

### `list`

Alias: `ls`. `--format`/`-f` accepts `default`, `table`, or `json`.

### `current`

Aliases: `c`, `whoami`. Add `--json`/`-j` for machine-readable output.

## Switch

### `switch ALIAS`

Aliases: `s`, `use`.

| Flag | Meaning |
| --- | --- |
| `--validate`, `-V` | Validate account and SSH only; make no switch |
| `--force`, `-f` | Tolerate supported SSH/GPG/`gh` failures; basic preflight still runs |
| `--skip-validation`, `-s` | Skip SSH preflight and final Git/SSH test; basic preflight still runs |

Read [Security](SECURITY.md) for all mutated state and rollback limits.

### `auto add PATH ALIAS`

Adds or replaces the rule for an existing directory. `PATH` is stored as an absolute physical path after symlink resolution, and `ALIAS` must already exist.

### `auto list [--json]`

Lists directory rules in stable path order.

### `auto remove PATH`

Removes the rule for the canonical existing directory. If the directory was deleted, pass the exact absolute path printed by `auto list`. Removing an account also removes every rule that references it.

### `auto resolve [PATH] [--json]`

Prints the alias selected for `PATH`, or the current directory when omitted, without switching. No match is successful with no human output (`null` with `--json`). The deepest directory rule wins, and sibling path prefixes do not match.

### `auto-switch [PATH] [--quiet]`

Applies the matching rule through the same transaction as `switch`. No match and an already-active account are successful no-ops. Automatic switching never enables force mode. This command is intended for the generated hook.

### `shell-init zsh`

Prints a zsh hook using the absolute path of the running GitShift executable. It does not edit startup files. The hook runs only in interactive shells, prevents re-entry, avoids duplicate work in the same directory, deduplicates repeated sourcing, and disables only when `GITSHIFT_AUTO_SWITCH_DISABLE=1`.

## Discovery

### `discover`

Enumerates successfully authenticated GitHub CLI identities on all configured hosts, then scans conventional SSH key pairs and local GPG metadata. Each key is tested in isolation against discovered provider hosts so a generic `id_ed25519` can be matched by the username it actually authenticates as. GitHub identities use `github|hostname|username`, allowing multiple accounts for the same provider and host. Remaining SSH/GPG evidence is merged only for a unique filename username or email match. Ambiguous evidence remains separate.

- `--dry-run` is the only guaranteed report-only mode. It scans even when accounts exist and does not create, chmod, migrate, or rewrite configuration.
- On a fresh interactive installation, plain `discover` asks for aliases and which imported account to activate, then applies it. In redirected/non-interactive use it imports without prompting.
- `--auto-import` bypasses the confidence threshold for otherwise complete results.
- `--overwrite` permits replacement when accounts already exist; replacement occurs only after a successful nonempty scan.
- `--alias SELECTOR=ALIAS` assigns a user-chosen alias and can be repeated. Prefer the displayed canonical identity key as `SELECTOR`; an original suggested alias works only when it identifies exactly one result.
- `--prompt-aliases` asks for an alias for each result not covered by `--alias`. Press Enter to accept the displayed suggestion. End-of-input and read errors abort discovery.

Alias selection is planned and validated before import. Aliases must use letters, digits, dots, underscores, or hyphens, start with a letter or digit, and contain at most 64 characters. They are unique case-insensitively across the discovery result and, for additive imports, configured active and pending accounts. Using either alias option allows an additive scan when accounts already exist; combine it with `--overwrite` only when the discovered set should replace them. `--overwrite` ignores old alias collisions and removes directory rules whose accounts are no longer present.

```bash
gitshift discover --dry-run
gitshift discover \
  --alias 'github|github.com|thukabjj=personal' \
  --alias 'github|github.com|arthurcosta-marks-and-spencer=work'
gitshift discover --prompt-aliases
```

Discovery does not change the active GitHub CLI account and does not persist tokens. Corporate email domains are not assumed to be GitLab.

## Keys and connectivity

### `ssh-keygen ALIAS`

Flags: `--type` (`ed25519`, `rsa`, `ecdsa`), `--bits`, `--email`, `--force`, and GitHub-only `--add-to-github`. The command generates a passwordless key and can affect the SSH agent, `known_hosts`, clipboard, and remote GitHub account.

### `gpg-keygen ALIAS`

Flags: `--type` (`RSA`, `ECC`, `DSA`), `--bits`, `--email`, `--expire-date`, `--enable`, and `--force`. The generated key is passwordless.

### `ssh-test [ALIAS]`

Tests one account, or all accounts with `--all`. `--verbose`/`-v` shows SSH output. It is a diagnostic command and does not repair `known_hosts`.

## Shell completion

`completion bash`, `completion zsh`, `completion fish`, and `completion powershell` print completion scripts and shell-specific setup instructions.

## Global flags and exit behavior

`--config PATH` selects an exact YAML file. Root `--version`/`-v` prints build information. Successful commands exit zero; validation, input, configuration, or external-tool errors exit nonzero. Human-oriented output and wording may evolve, so scripts should prefer JSON options where available rather than parsing decorated text.
