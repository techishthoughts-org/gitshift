# User guide

This guide describes the current source on `main`. The compatibility notice in [Installation](INSTALLATION.md) explains why the latest public binary currently has a smaller, GitHub-only command set.

## How GitShift models an account

Each account has a unique alias, a Git author name and email, a provider (`github` or `gitlab`), a username, an optional provider domain, and an optional SSH private-key path. The alias is local to GitShift; it does not need to match the provider username.

## Interactive terminal UI

Run the full-screen interface from an interactive terminal:

```bash
gitshift
gitshift ui
```

Running `gitshift` without a subcommand opens the UI. `gitshift tui` is an
alias for the explicit `ui` command. Use `gitshift ui --inline` when you want
output to remain in the current screen buffer. The global `--config PATH` flag
works in both modes.

The dashboard links to accounts, directory rules, and discovery. Arrow keys or
`j`/`k` move through lists, Enter opens an item, `?` opens help, Escape goes
back, and `q` quits. Add/edit forms use Tab to move between fields and reuse
the same account validation as the CLI. Switching and removal require an
explicit confirmation. The UI uses only the normal validated switch; force,
skip-validation, SSH key generation, and GPG key generation remain CLI-only.
If you request quit while a confirmed write or switch is running, the UI waits
for that transaction and its configuration refresh to finish before exiting.

Discovery starts read-only. Every same-host provider identity is displayed as
a separate row. Select results with Space, edit a result's alias with `e`, then
press `i` and confirm to import only the selected rows. The complete selection
is validated before one atomic additive write; existing accounts are never
silently replaced. Opening, browsing, cancelling, and quitting do not write.

Adding a directory rule in the UI does not install a shell hook. Run
`source <(gitshift shell-init zsh)` (and opt in from `.zshrc` if desired) before
expecting rules to switch identities as you change directories.

Both stdin and stdout must be terminals. A redirected invocation fails before
loading or creating configuration. Colors adapt to light and dark terminal
backgrounds; set `NO_COLOR=1` for unstyled, ASCII-safe output.

## Add accounts

Interactive mode prompts for required values:

```bash
gitshift add personal
```

For scripts, provide every required value and use `--non-interactive`:

```bash
gitshift add personal \
  --name "Alex Example" \
  --email alex@example.com \
  --username alex \
  --platform github \
  --ssh-key "$HOME/.ssh/id_ed25519_personal" \
  --non-interactive
```

For GitHub, authenticate the same username and hostname in GitHub CLI before switching:

```bash
gh auth login --hostname github.com
gh auth status --hostname github.com
```

Add a GitLab.com account:

```bash
gitshift add work \
  --name "Alex Example" \
  --email alex@company.example \
  --username alex-company \
  --platform gitlab \
  --ssh-key "$HOME/.ssh/id_ed25519_work"
```

For GitHub Enterprise or self-managed GitLab, add `--domain git.company.example`. `--api-endpoint` records a custom API URL but does not add support for another provider.

The first account becomes the default alias. Adding a later account with `--default` changes the default, but account creation never applies an identity to Git, SSH, GPG, or GitHub CLI and never marks it current; run `switch` when ready.

## SSH keys

Attach an existing private key with `--ssh-key`. The corresponding public key must already be registered with the provider.

GitShift can create a key:

```bash
gitshift ssh-keygen personal --type ed25519 --email alex@example.com
gitshift ssh-test personal --verbose
```

Generated SSH keys are passwordless so GitShift can use them non-interactively. If you require a passphrase, generate the key with `ssh-keygen`, load it into an agent, register the public key with your provider, and attach the private key with `gitshift update ALIAS --ssh-key PATH`.

`ssh-keygen --add-to-github` uses `gh ssh-key add`; it is GitHub-only. GitShift may add official `github.com` or `gitlab.com` host keys to `known_hosts`, but it will not automatically trust an unknown enterprise host.

## Inspect and switch

```bash
gitshift list --format table
gitshift current
gitshift switch personal --validate
```

`--validate` checks account fields, the SSH key, and SSH connectivity without switching. It does not verify GPG readiness or GitHub CLI authentication. Avoid these escape hatches unless you have checked the account yourself:

- `--force` tolerates supported SSH, GPG, and GitHub CLI failures and can still record the target as current; it does not bypass basic account, platform, key-path, or state-capture checks.
- `--skip-validation` skips SSH preflight and the final Git/SSH test; it does not skip basic account, key-path, platform, or state-capture checks.

In normal non-force mode, GitShift records the selected account after the requested changes and final checks succeed. It attempts compensating rollback if a normal step fails; force mode intentionally permits some partial integration failures, and an abrupt process termination or power loss can still leave partial external state. See [Security](SECURITY.md).

## Edit and remove accounts

```bash
gitshift update work --email new-address@company.example
gitshift update work --domain gitlab.company.example
gitshift remove work
```

Removing an account deletes its GitShift record. It does not delete SSH or GPG keys, revoke credentials, remove provider keys, or apply another account to Git, SSH, GPG, or GitHub CLI state.

## Discover existing settings

Always start with a read-only scan:

```bash
gitshift discover --dry-run
```

Discovery enumerates every successfully authenticated identity reported by `gh auth status --json hosts`, including multiple users on the same host and GitHub Enterprise hosts. Each provider identity is kept separate by the canonical key `provider|hostname|username`. It also examines conventional SSH key pairs and local GPG keys. GitShift tests each key with user SSH configuration and the shared agent disabled, then matches the authenticated username to exactly one discovered provider identity. Remaining SSH/GPG evidence is attached by filename username or email only when the match is unambiguous. A corporate email alone is not treated as evidence of GitLab. Results have a confidence score from 1–10.

- `--dry-run` reports only: it does not create, chmod, migrate, or rewrite the configuration, and it still scans when accounts are already configured.
- On a fresh interactive installation, plain `discover` asks for aliases and the account to activate, then applies that account. Redirected/non-interactive discovery imports without prompting and can add incomplete higher-confidence results as pending metadata.
- `--auto-import` also allows otherwise complete results below confidence 6; it is less restrictive, not merely non-interactive.
- `--overwrite` permits replacement of configured active/pending results after a successful nonempty scan. Existing accounts are preserved when scanning fails or finds nothing.

Choose memorable aliases by addressing each account with the identity shown by the dry run:

```bash
gitshift discover \
  --alias 'github|github.com|thukabjj=personal' \
  --alias 'github|github.com|arthurcosta-marks-and-spencer=work'
```

`--alias SELECTOR=ALIAS` is repeatable. A canonical identity key is the safest selector; an exact suggested alias is accepted only when unambiguous. Use `--prompt-aliases` to be asked about every result not already selected and press Enter to keep a suggestion. Alias validation and case-insensitive collision checks complete before import. With either alias option, discovery can add accounts to an existing configuration; add `--overwrite` only to replace the existing active and pending accounts. A dry run previews the chosen aliases without writing.

GitHub profile enrichment retrieves the credential for each explicit host/user from GitHub CLI and supplies it only to that identity's `gh api` process. Discovery never runs `gh auth switch`, prints a token, or stores a provider token in GitShift configuration. When profile enrichment is unavailable, the authenticated host and username can still be reported with lower confidence.

Discovery is an import aid and does not create directory rules.

## Switch automatically by directory

Rules match an entire directory tree. More-specific rules override broader ones:

```bash
gitshift auto add "$HOME/dev/personal" personal
gitshift auto add "$HOME/dev/work" work
gitshift auto resolve "$HOME/dev/personal/example"
gitshift auto list --json
```

Both directory roots must already exist and both aliases must already be configured. GitShift resolves symlinks before storing or matching paths. Enable the generated zsh hook explicitly:

If a rule's directory is later deleted, use its exact path from `gitshift auto list` with `gitshift auto remove PATH`.

```bash
source <(gitshift shell-init zsh)
```

Put that line in `~/.zshrc` if you want it in future interactive shells. GitShift never edits the file. Temporarily disable switching with `export GITSHIFT_AUTO_SWITCH_DISABLE=1`; unset it to re-enable. Only the exact value `1` disables it. Re-sourcing replaces the existing hook and immediately re-evaluates the current directory, so newly added rules take effect. Automatic switches use normal validation and rollback, never `--force`, and do nothing when no rule matches or the selected alias is already current.

## GPG signing

Generate and enable a key:

```bash
gitshift gpg-keygen personal --type rsa --bits 4096 --email alex@example.com --enable
gitshift update personal --auto-gpg
```

GPG key generation is non-interactive and passwordless. `--auto-gpg` selects a local secret key by account email. Generate a protected key with GnuPG itself when your security policy requires a passphrase, then use `--auto-gpg`. Switching an account with GPG enabled changes Git signing-key and commit-signing settings.

`add --enable-gpg` records a signing preference but does not generate or select a key. Generate one with `gpg-keygen` or associate an existing email-matched key with `update --auto-gpg` before switching.

## Machine-readable output and custom configuration

Use `gitshift list --format json` and `gitshift current --json` for scripts. Every command accepts `--config PATH` to use an exact alternate YAML file:

```bash
gitshift --config "$PWD/test-config.yaml" list --format json
```

Run `gitshift COMMAND --help` to check flags supported by the installed version.
