# Installation

Building the current source is presently the recommended installation method. Prebuilt binaries exist for Linux x86-64/ARM64, macOS Intel/Apple silicon, and Windows x86-64, but the published assets are legacy-only until they are replaced.

## Published release versus current source

The public assets were downloaded and verified on 2026-10-16. They contain the legacy GitHub-only command set and report embedded version `v0.1.0`, despite being attached to a newer release. They also ignore `--config` and create `$HOME/.config/gitshift/config.yaml` as `0644` on macOS. Use a source build for GitLab, GPG, platform/domain flags, custom configuration paths, current switching behavior, and owner-only configuration creation. Maintainers should remove this notice only after publishing and testing replacement assets.

If you must use a legacy download, use an alias identical to the GitHub username, avoid `--config`, and secure the file immediately after its first creation:

```bash
gitshift add alex --name "Alex Example" --email alex@example.com --github-username alex
chmod 600 "$HOME/.config/gitshift/config.yaml"
gitshift switch alex
```

The Windows ACL equivalent must restrict the file to your user. Legacy downloads do not have the current GitLab, GPG, exact-path, or persistence protections documented elsewhere.

## Before installing

Install Git and an OpenSSH client. GitHub users also need [GitHub CLI](https://cli.github.com/) because GitShift selects the matching authenticated `gh` account during a GitHub switch. GnuPG is optional.

## Linux

Choose the asset that matches `uname -m`: `x86_64` uses `amd64`; `aarch64` or `arm64` uses `arm64`.

```bash
asset=gitshift-linux-amd64
curl -LO "https://github.com/techishthoughts-org/gitshift/releases/latest/download/$asset"
curl -LO https://github.com/techishthoughts-org/gitshift/releases/latest/download/checksums.txt
grep " $asset$" checksums.txt | sha256sum --check
install -Dm755 "$asset" "$HOME/.local/bin/gitshift"
```

Ensure `$HOME/.local/bin` is on `PATH`, then run `gitshift --version`.

## macOS

Use `gitshift-darwin-arm64` on Apple silicon and `gitshift-darwin-amd64` on Intel Macs.

```bash
asset=gitshift-darwin-arm64
curl -LO "https://github.com/techishthoughts-org/gitshift/releases/latest/download/$asset"
curl -LO https://github.com/techishthoughts-org/gitshift/releases/latest/download/checksums.txt
grep " $asset$" checksums.txt | shasum -a 256 --check
mkdir -p "$HOME/.local/bin"
install -m 755 "$asset" "$HOME/.local/bin/gitshift"
```

If macOS blocks the unsigned binary after you verify its checksum, open System Settings → Privacy & Security and explicitly allow it. Only do this for a binary downloaded from the project's release page whose checksum matches.

## Windows

In PowerShell:

```powershell
$asset = "gitshift-windows-amd64.exe"
$base = "https://github.com/techishthoughts-org/gitshift/releases/latest/download"
Invoke-WebRequest "$base/$asset" -OutFile $asset
Invoke-WebRequest "$base/checksums.txt" -OutFile checksums.txt
$actual = (Get-FileHash $asset -Algorithm SHA256).Hash.ToLower()
$match = Select-String -Path checksums.txt -Pattern ("  " + [regex]::Escape($asset) + "$")
if (-not $match) { throw "No checksum published for $asset" }
$expected = ($match.Line -split '\s+')[0]
if ($actual -ne $expected) { throw "Checksum mismatch for $asset" }
"Checksum OK: $asset"
```

Move the verified executable to a dedicated directory such as `%LOCALAPPDATA%\Programs\GitShift`, and add that directory to your user `PATH`. Open a new terminal and run `gitshift --version`.

For a reproducible installation, replace `/releases/latest/download/` in these URLs with `/releases/download/vX.Y.Z/` and use the checksum file from that same release.

## Build from source

Building requires the Go version declared in `go.mod` (currently Go 1.26) and Make for the convenience targets.

```bash
git clone https://github.com/techishthoughts-org/gitshift.git
cd gitshift
make test
make build
./gitshift --version
```

`make install` copies the binary to `${GOPATH:-$HOME/go}/bin`. Direct `go install github.com/techishthoughts-org/gitshift@latest` is not supported by the current module path; use a release binary or a source checkout.

## Docker is for isolated development

The repository has a Dockerfile but does not document a published image. Build it locally with `docker build -t gitshift:local .`. The image lacks GitHub CLI and GnuPG, and changes apply inside the container unless you deliberately mount host Git/SSH/config paths and grant suitable permissions. Because those mounts expose sensitive identity files, a native binary is the recommended way to manage the host.

## Upgrade

Download and verify the new release, then replace the existing executable. The configuration file is separate and is retained. Before a major upgrade, back it up:

```bash
cp "$HOME/.config/gitshift/config.yaml" "$HOME/.config/gitshift/config.yaml.backup"
```

Run `gitshift --version`, `gitshift list`, and `gitshift current` after replacement.

Do not rely only on the release-page tag while the compatibility notice above applies. Check `gitshift --help` and confirm the commands you need are present.

## Uninstall

Remove the executable from the location where you installed it. This does not revert Git, SSH, GPG, or GitHub CLI settings and does not remove GitShift's configuration.

To remove stored account metadata too, first back up and then delete `$HOME/.config/gitshift`. On Windows this is normally `%USERPROFILE%\.config\gitshift`. Review [Security](SECURITY.md) before manually reverting settings left by the last switch.
