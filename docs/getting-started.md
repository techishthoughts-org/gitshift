# Getting started

In this tutorial you install GitShift, register a work and a personal account, switch between them, make your shell switch automatically by directory, and show the active account in your prompt. It takes about ten minutes.

You need:

- macOS with Homebrew (for other systems, see [Install GitShift](how-to/install.md))
- Git and OpenSSH
- an SSH key per account, already added to that account on GitHub or GitLab (no keys yet? see [Create and test keys](how-to/keys.md))
- for GitHub accounts, the [GitHub CLI](https://cli.github.com/)
- zsh, for the auto-switch part

The examples use a GitHub account `alex` for personal work and a GitHub account `alex-company` for work. Replace names, emails, usernames and key paths with your own.

## 1. Install

```bash
brew tap techishthoughts-org/tap
brew trust techishthoughts-org/tap
brew install --cask techishthoughts-org/tap/gitshift
gitshift --version
```

`brew trust` is needed because Homebrew refuses casks from taps you have not trusted.

## 2. Log in to the GitHub CLI with both accounts

GitShift selects the matching GitHub CLI account when it switches, but it never logs in for you. Log in once per account:

```bash
gh auth login --hostname github.com   # sign in as alex
gh auth login --hostname github.com   # sign in again as alex-company
gh auth status
```

`gh auth status` should list both users under `github.com`. Skip this step for GitLab accounts.

## 3. Add the two accounts

```bash
gitshift add personal \
  --name "Alex Example" \
  --email alex@example.com \
  --username alex \
  --ssh-key ~/.ssh/id_ed25519_personal

gitshift add work \
  --name "Alex Example" \
  --email alex@company.example \
  --username alex-company \
  --ssh-key ~/.ssh/id_ed25519_work

gitshift list
```

Adding an account only records it. Nothing in Git, SSH or the GitHub CLI changes yet. The first account you add is marked as the default.

## 4. Switch

Check the work account first. This tests the SSH key against github.com and changes nothing:

```bash
gitshift switch work --validate
```

Then switch:

```bash
gitshift switch work
gitshift current
git config --global user.email    # alex@company.example
gh auth status                    # alex-company is active
```

The switch is global for your user: every terminal now commits as `alex@company.example` and pushes with the work key. If you ran it inside a Git repository, that repository's local identity was set as well. The [identity model](explanation/identity-model.md) explains why.

Switch back with `gitshift switch personal`.

## 5. Switch automatically by directory

Map directory trees to accounts. The directories must exist:

```bash
mkdir -p ~/dev/work ~/dev/personal
gitshift auto add ~/dev/work work
gitshift auto add ~/dev/personal personal
```

Choose the account to use everywhere else. Without one, leaving a rule directory keeps whatever account is active:

```bash
gitshift auto default personal
```

Check what each directory resolves to, without switching:

```bash
gitshift auto resolve ~/dev/work/some-repo   # work
gitshift auto resolve /tmp                   # personal (default)
```

Turn on the zsh hook. GitShift never edits your startup files, so add the line yourself:

```bash
echo 'source <(gitshift shell-init zsh)' >> ~/.zshrc
source ~/.zshrc
```

Now `cd ~/dev/work` switches to `work`, and `cd ~` switches to `personal`.

## 6. Show the active account in your prompt

`gitshift current --short` prints only the active alias, or nothing if no switch has happened yet. It never writes anything, so it is safe to run on every prompt.

With [Starship](https://starship.rs), add to `~/.config/starship.toml`:

```toml
[custom.gitshift]
command = "gitshift current --short"
when = true
shell = ["sh"]
format = "[@$output]($style) "
```

With a plain zsh prompt, add to `~/.zshrc`:

```zsh
setopt prompt_subst
RPROMPT='$(gitshift current --short)'
```

## Next steps

- [Switch accounts](how-to/switch-accounts.md) covers `--force`, `--skip-validation` and what happens when a switch fails.
- [Security model](explanation/security-model.md) lists every file and setting a switch changes. Read it before you rely on GitShift.
- Prefer menus? Run `gitshift` with no arguments for the terminal UI.
