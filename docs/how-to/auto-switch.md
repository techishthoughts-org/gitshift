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

## Let Git pick the identity per directory

The hook switches one global identity, so with several terminals open in different directories the last `cd` decides for all of them. Turn on Git includes to have Git choose the identity for each rule directory by itself:

```bash
gitshift auto git-includes on
```

GitShift then:

- writes `git/<alias>.gitconfig` next to its config file (`~/.config/gitshift/git/` by default) for every account a rule uses, with the account's name, email, Git SSH command and [commit signing](keys.md#sign-commits-and-tags);
- adds one `includeIf "gitdir:<rule directory>/"` entry per rule at the end of your global Git config, deeper directories last so they win;
- keeps both in sync whenever you change an account or a rule, and moves its entries back to the end if another tool adds settings after them;
- stops writing identity values into a repository's `.git/config` when you switch inside it, because local values would override the includes.

Every repository under a rule directory now uses that rule's identity in every terminal at once, with no switching. Outside rule directories Git uses the global identity, which the hook still sets from your default account. The hook also keeps switching the GitHub CLI account and `~/.ssh/config`, which Git cannot choose per directory.

```bash
gitshift auto git-includes status   # entries in your Git config
gitshift auto git-includes off      # remove the entries and files
```

Repositories where you switched before turning this on may still hold local values. Check with `git config --local --list` inside the repository and remove `user.name`, `user.email` and `core.sshCommand` there if they are not what you want.

## Show the active account in your prompt

`gitshift current --short` prints the active alias. With Git includes on, inside a rule directory it prints that rule's account, the one Git uses there.

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
