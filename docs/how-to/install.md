# Install, upgrade and uninstall

GitShift is a single binary. It needs Git and an OpenSSH client at run time, the [GitHub CLI](https://cli.github.com/) for GitHub accounts, and GnuPG only if you use commit signing.

## Install with Homebrew

On macOS (the cask also declares Linux):

```bash
brew tap techishthoughts-org/tap
brew trust techishthoughts-org/tap
brew install --cask techishthoughts-org/tap/gitshift
gitshift --version
```

`brew trust` is required: Homebrew refuses to install casks from a tap you have not trusted. The cask clears the macOS quarantine attribute, so the binary runs even though it is not notarized.

## Install from a release archive

Each [GitHub release](https://github.com/techishthoughts-org/gitshift/releases) has one archive per platform plus `checksums.txt`. The file name uses the version without the leading `v`:

| System | Archive |
| --- | --- |
| macOS Apple silicon | `gitshift_2.0.0_darwin_arm64.tar.gz` |
| macOS Intel | `gitshift_2.0.0_darwin_amd64.tar.gz` |
| Linux x86-64 | `gitshift_2.0.0_linux_amd64.tar.gz` |
| Linux ARM64 | `gitshift_2.0.0_linux_arm64.tar.gz` |
| Windows x86-64 | `gitshift_2.0.0_windows_amd64.zip` |
| Windows ARM64 | `gitshift_2.0.0_windows_arm64.zip` |

Each archive contains `gitshift` (`gitshift.exe` on Windows), `README.md` and `CHANGELOG.md`.

### Linux

```bash
version=2.0.0
asset="gitshift_${version}_linux_amd64.tar.gz"    # or linux_arm64
base="https://github.com/techishthoughts-org/gitshift/releases/download/v${version}"
curl -LO "$base/$asset"
curl -LO "$base/checksums.txt"
sha256sum --check --ignore-missing checksums.txt   # must print: <asset>: OK
tar -xzf "$asset" gitshift
install -Dm755 gitshift "$HOME/.local/bin/gitshift"
gitshift --version
```

Make sure `$HOME/.local/bin` is on your `PATH`.

### macOS

```bash
version=2.0.0
asset="gitshift_${version}_darwin_arm64.tar.gz"   # or darwin_amd64
base="https://github.com/techishthoughts-org/gitshift/releases/download/v${version}"
curl -LO "$base/$asset"
curl -LO "$base/checksums.txt"
shasum -a 256 --check --ignore-missing checksums.txt
tar -xzf "$asset" gitshift
xattr -d com.apple.quarantine ./gitshift 2>/dev/null || true
mkdir -p "$HOME/.local/bin" && install -m 755 gitshift "$HOME/.local/bin/gitshift"
```

The binaries are not notarized. Without the `xattr` step, Gatekeeper blocks a binary downloaded with a browser. Remove the quarantine only after the checksum matches.

### Windows (PowerShell)

```powershell
$version = "2.0.0"
$asset = "gitshift_${version}_windows_amd64.zip"   # or windows_arm64
$base = "https://github.com/techishthoughts-org/gitshift/releases/download/v$version"
Invoke-WebRequest "$base/$asset" -OutFile $asset
Invoke-WebRequest "$base/checksums.txt" -OutFile checksums.txt
$expected = (Select-String -Path checksums.txt -Pattern ([regex]::Escape($asset) + '$')).Line.Split(' ')[0]
$actual = (Get-FileHash $asset -Algorithm SHA256).Hash.ToLower()
if ($actual -ne $expected) { throw "Checksum mismatch for $asset" }
Expand-Archive $asset -DestinationPath "$env:LOCALAPPDATA\Programs\GitShift"
```

Add `%LOCALAPPDATA%\Programs\GitShift` to your user `PATH`, open a new terminal and run `gitshift --version`. The directory auto-switch hook is zsh-only, so it is not available in PowerShell.

## Build from source

You need Go 1.26 and Make.

```bash
git clone https://github.com/techishthoughts-org/gitshift.git
cd gitshift
make build            # ./gitshift
make install          # copies it to ${GOPATH:-$HOME/go}/bin
```

`go install github.com/...@latest` is not supported: the module path (`github.com/techishthoughts/gitshift`) differs from the repository path (`techishthoughts-org/gitshift`).

## Shell completion

```bash
gitshift completion zsh  > "${fpath[1]}/_gitshift"   # zsh
gitshift completion bash > ~/.local/share/bash-completion/completions/gitshift
gitshift completion fish > ~/.config/fish/completions/gitshift.fish
```

Run `gitshift completion SHELL --help` for shell-specific setup, including PowerShell.

## Upgrade

Homebrew:

```bash
brew upgrade --cask gitshift
```

Archive or source install: replace the binary. Your configuration (`~/.config/gitshift/config.yaml`) is separate and is kept.

After upgrading, check:

```bash
gitshift --version
gitshift list
gitshift current
```

### Upgrading from v1.x

Older releases are not what their tags suggest:

- The v1.0.0 to v1.0.3 GitHub release binaries are legacy GitHub-only builds. They report `v0.1.0` and ignore `--config`.
- Tags v1.1.0 to v1.4.2 were source-only and have no binaries.

v2.0.0 is the first release whose binaries match the source. To move to it:

1. Back up `~/.config/gitshift/config.yaml`, `~/.gitconfig` and `~/.ssh/config`.
2. Remove the old binary from wherever you installed it (`command -v gitshift` shows where), then install v2.0.0 with one of the methods above.
3. Run `gitshift list`. Like most commands, it rewrites a file that has no `config_version` (or `1.0.0`) as version `1.1.0`, and sets the file to mode `0600`. A file with any other version is rejected and left untouched. Read-only commands (`current --short`, `auto list`, `auto resolve`, `discover --dry-run`) never rewrite it.
4. Accounts without a `platform` are treated as GitHub. To make that explicit: `gitshift update ALIAS --platform github --username USERNAME`.
5. Run `gitshift switch ALIAS --validate`, then `gitshift switch ALIAS`.

Behaviour that changed since the v1 source:

- Only `switch` (CLI, terminal UI or auto-switch) marks an account as current. `add --default` and `update --default` change only the default.
- `gitshift auto default` sets an account for directories outside every rule.
- `gitshift current --short` prints only the active alias, for prompts.
- Commands no longer print `Using config file`.

To downgrade, restore the configuration backup from step 1. Older binaries may not read a file written by v2.

## Move to another machine

Copying `config.yaml` is not enough. SSH private keys, GPG secret keys, GitHub CLI logins and trusted host keys live elsewhere. Move keys with a method your organization approves, run `gh auth login` for each GitHub account, fix key paths with `gitshift update ALIAS --ssh-key PATH`, recreate directory rules (they store absolute paths), and run `gitshift switch ALIAS --validate` before switching.

## Uninstall

```bash
brew uninstall --cask gitshift   # or delete the binary you installed
```

This leaves your configuration and the Git, SSH and GitHub CLI state of the last switch in place. To remove everything GitShift created:

1. Delete `~/.config/gitshift/`.
2. In `~/.ssh/config`, delete the `# >>> gitshift managed: HOST` to `# <<< gitshift managed: HOST` blocks and uncomment any lines prefixed with `# gitshift-disabled: `. Your config from before GitShift's first write is in `~/.ssh/config.gitshift-orig`.
3. Review `git config --global --list` for `user.name`, `user.email`, `core.sshCommand`, `user.signingkey`, `commit.gpgsign` and `tag.gpgsign`.

The [security model](../explanation/security-model.md) lists everything GitShift may have changed.
