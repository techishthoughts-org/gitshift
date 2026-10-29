# Manage accounts

An account is an alias (such as `work`) plus a Git author name and email, a provider username, a platform (`github` or `gitlab`), a hostname and, usually, an SSH private key. The alias is local to GitShift and need not match the provider username.

Aliases use letters, digits, `.`, `_` and `-`, start with a letter or digit, and are at most 64 characters.

## Add an account

With every required value on the command line, `add` runs without prompting:

```bash
gitshift add personal \
  --name "Alex Example" \
  --email alex@example.com \
  --username alex \
  --ssh-key ~/.ssh/id_ed25519_personal \
  --description "Personal GitHub"
```

Required: the alias, `--name`, `--email` and `--username`. If any is missing, `add` prompts for it and also asks for the optional SSH key path and description. Add `--non-interactive` in scripts so a missing value fails instead of prompting.

| Flag | Meaning |
| --- | --- |
| `--name`, `-n` | Git `user.name` |
| `--email`, `-e` | Git `user.email` |
| `--username` | Provider username, without `@` |
| `--github-username`, `-g` | Older spelling of `--username`; `--username` wins if both are given |
| `--platform` | `github` (default) or `gitlab` |
| `--domain` | Provider hostname; defaults to `github.com` or `gitlab.com` |
| `--api-endpoint` | HTTPS API URL, recorded only |
| `--ssh-key`, `-k` | Existing private key; `~/` is expanded and the path stored as absolute |
| `--description`, `-d` | Free-text label |
| `--default` | Mark this account as the default |
| `--enable-gpg` | Record that commits should be signed; you still need a key (see [keys](keys.md)) |
| `--non-interactive` | Never prompt |

`add` checks that the key file exists and is readable, but does not test it against the provider or check its permissions; `switch` later refuses a private key whose mode is not `600`. It never changes Git, SSH or GitHub CLI state, and it never makes the account current. Run `gitshift switch ALIAS` when you want to use it.

For GitHub accounts, log the same username in to the GitHub CLI before you switch: `gh auth login --hostname github.com`. For self-managed GitLab and GitHub Enterprise, see [GitLab and Enterprise](gitlab-and-enterprise.md).

## List and inspect

```bash
gitshift list                 # grouped by platform
gitshift list --format table
gitshift list --format json   # for scripts
gitshift current              # the account the last switch applied
gitshift current --json
gitshift update work          # with no flags, prints the stored fields
```

`list` also shows pending accounts left by discovery. `current` exits with an error if no switch has applied an account yet.

## Update an account

`update` changes only the fields you pass:

```bash
gitshift update work --email alex@new-company.example
gitshift update work --ssh-key ~/.ssh/id_ed25519_work_2026
gitshift update work --platform gitlab --domain gitlab.company.example
gitshift update work --domain ""         # back to the platform default host
gitshift update work --api-endpoint ""   # clear the API endpoint
gitshift update work --auto-gpg          # attach a local GPG key whose email matches
gitshift update work --default
```

Flags: `--name`/`-n`, `--email`/`-e`, `--username`, `--github-username`/`-g` (must match `--username` if both are given), `--ssh-key`/`-k`, `--description`/`-d`, `--platform`/`-p`, `--domain`, `--api-endpoint`, `--auto-gpg`, `--signing ssh|gpg|off`, `--signing-key`, `--default`. `--ssh-key` must point at an existing file.

Changing `--platform` also resets the domain and API endpoint to defaults unless you pass them in the same command.

An update is not applied to Git or SSH until you switch to that account again.

## Remove an account

```bash
gitshift remove work        # asks for confirmation
gitshift remove work -f     # no prompt
```

Aliases: `rm`, `delete`, `del`. Removal deletes the account, its directory rules, and the auto-switch default if it pointed at this account. If it was the current account, GitShift no longer records any current account. If it was the default, the current account becomes the default, or, if that was the one removed, the first remaining alias in alphabetical order.

Removal does not touch Git, SSH, GPG or the GitHub CLI. Your global identity stays as the last switch left it, and keys stay on disk and on the provider.

## Import existing identities with discover

`discover` builds accounts from what is already on the machine: every account logged in to the GitHub CLI (on every host), SSH key pairs in `~/.ssh`, and secret keys in your GPG keyring. Always preview first:

```bash
gitshift discover --dry-run
```

`--dry-run` scans and prints, and never creates or rewrites the config file. Each result shows an identity key such as `github|github.com|alex`, a suggested alias and a confidence score from 1 to 10.

Name the accounts yourself, using the identity key (quote it, `|` is a shell operator):

```bash
gitshift discover \
  --alias 'github|github.com|alex=personal' \
  --alias 'github|github.com|alex-company=work'
```

Or be asked for each alias, pressing Enter to accept a suggestion:

```bash
gitshift discover --prompt-aliases
```

Behaviour to know:

- On a fresh install in an interactive terminal, plain `gitshift discover` asks for aliases, imports, then asks which account to activate and switches to it.
- If accounts already exist, plain `discover` stops and lists them. `--alias` or `--prompt-aliases` adds the new results alongside them. `--overwrite` replaces all existing accounts, but only after a scan that found something; rules for accounts that disappear are removed.
- Results with confidence 6 or more are imported. `--auto-import` also imports complete results below that threshold.
- Incomplete results may be saved as pending accounts. Re-create them with `gitshift add`.
- Aliases are checked for format and for case-insensitive collisions before anything is written.

Discovery never changes the active GitHub CLI account and never stores tokens. To read each GitHub profile it asks `gh auth token` for that user's token and passes it only to one short-lived `gh api user` process. SSH keys are tested one at a time with your `~/.ssh/config` and SSH agent disabled.

## Use the terminal UI

```bash
gitshift            # same as gitshift ui
gitshift ui --inline   # render in the current screen instead of the alternate screen
```

Alias: `gitshift tui`. Both standard input and output must be a terminal.

| Screen | Keys |
| --- | --- |
| Dashboard | `1`/`a` accounts, `2`/`r` directory rules, `3`/`d` discovery |
| Accounts | `j`/`k` or arrows to move, Enter for details, `n` new, `e` edit, `s` switch, `d` remove |
| Directory rules | `n` add a rule, `d` remove |
| Discovery | Space select, `e` edit alias, `i` import selected, `r` rescan |
| Anywhere | `?` help, Esc back, `q` quit |

Switching and removing ask for confirmation (`y` or Enter, `n` or Esc). The UI uses the same validation as the CLI and only does normal switches. `--force`, `--skip-validation`, key generation and `auto default` are CLI-only. Opening, browsing and quitting never write anything. If you quit while a switch or save is running, the UI waits for it to finish.

Set `NO_COLOR` to turn off colours.
