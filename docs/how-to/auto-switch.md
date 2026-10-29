# Switch automatically by directory

GitShift can switch accounts when you `cd` into a directory tree. You need three things: rules, optionally a default account, and the zsh hook.

## Add rules

```bash
gitshift auto add ~/dev/work work
gitshift auto add ~/dev/work/open-source personal
gitshift auto add ~/dev/personal personal
gitshift auto list            # or: gitshift auto list --json
```

- The directory must exist and the account must already be configured.
- The path is stored as an absolute, physical path: symlinks are resolved, so a rule on a symlink applies to its target.
- A rule covers the directory and everything below it. When several rules match, the deepest one wins, so `~/dev/work/open-source` uses `personal` above.
- Matching is by path component: a rule for `~/dev/work` does not match `~/dev/work-old`.
- Adding a rule for a path that already has one replaces its account.

Remove a rule:

```bash
gitshift auto remove ~/dev/work
```

If the directory no longer exists, pass the exact path printed by `gitshift auto list`. Removing an account also removes its rules.

## Set a default account

Without a default, leaving every rule directory keeps whichever account is active. To switch to a fixed account everywhere else:

```bash
gitshift auto default personal   # set
gitshift auto default            # show
gitshift auto default --clear    # remove
```

The auto-switch default is separate from the account marked default by `add --default`. See the [identity model](../explanation/identity-model.md). Removing the account clears it.

## Check what a directory resolves to

```bash
gitshift auto resolve ~/dev/work/api    # work
gitshift auto resolve /tmp              # personal (default)
gitshift auto resolve                   # current directory
gitshift auto resolve --json /tmp       # null: --json reports only a matching rule
```

`auto resolve` never switches. It prints nothing when no rule matches and no default is set.

## Turn on the zsh hook

```bash
echo 'source <(gitshift shell-init zsh)' >> ~/.zshrc
source ~/.zshrc
```

`shell-init zsh` only prints the hook; GitShift never edits your startup files. zsh is the only supported shell. The hook:

- runs on every directory change and once when it is sourced,
- runs only in interactive shells,
- calls the binary that printed it by an absolute path: the `gitshift` found on your `PATH` (for Homebrew, `$(brew --prefix)/bin/gitshift`) when it runs the same binary, so the hook keeps working after `brew upgrade`; otherwise the binary's physical path. If you move the binary, open a new shell,
- passes `--config` if you used it with `shell-init`, for example `source <(gitshift --config ~/alt.yaml shell-init zsh)`,
- skips the work if you are still in the same directory, and retries on the next `cd` if a switch failed,
- replaces an earlier copy of itself if you source it twice, which also re-evaluates the current directory.

Each `cd` runs `gitshift auto-switch --quiet -- "$PWD"`. That command does nothing when no rule matches and no default is set, or when the selected account is already current. Otherwise it runs a normal switch, with validation and rollback. It never uses `--force`.

Because a switch is global, the hook in one terminal changes the identity for every terminal. If you switch manually inside a rule directory, the next `cd` switches back to the rule or default account.

### Pause the hook

```bash
export GITSHIFT_AUTO_SWITCH_DISABLE=1   # this shell only
unset GITSHIFT_AUTO_SWITCH_DISABLE
```

Only the exact value `1` disables it.

### Debug the hook

Run what the hook runs, without `--quiet`:

```bash
gitshift auto-switch "$PWD"
```

## Show the active account in your prompt

`gitshift current --short` (or `-s`) prints only the alias the last switch applied, or nothing if there is none. It reads the config without writing it and exits 0 when nothing is set, so it is safe on every prompt. Aliases only contain `A-Z a-z 0-9 . _ -`, so the output is safe to embed in a prompt.

### Starship

In `~/.config/starship.toml`:

```toml
[custom.gitshift]
command = "gitshift current --short"
when = true
shell = ["sh"]
format = "[@$output]($style) "
```

### Plain zsh

```zsh
setopt prompt_subst
RPROMPT='$(gitshift current --short)'
```

`current --short` reports what GitShift last applied. If you change Git or the GitHub CLI by hand, it does not notice.
