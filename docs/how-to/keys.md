# Create and test keys

GitShift can create SSH and GPG keys for an account and test SSH access. You can also create keys with the native tools and attach them; do that when you need a passphrase or a hardware key.

## Create an SSH key

```bash
gitshift ssh-keygen work
gitshift ssh-keygen work --type rsa --bits 4096
gitshift ssh-keygen scratch --email alex@example.com   # no account needed with --email
```

| Flag | Meaning |
| --- | --- |
| `--type` | `ed25519` (default), `rsa` or `ecdsa` |
| `--bits` | RSA: at least 2048, default 4096. ECDSA: 256 (default), 384 or 521. Ignored for Ed25519 |
| `--email` | Key comment; defaults to the account's email |
| `--force`, `-f` | Allow an existing file at the target path (`ssh-keygen` itself may still ask before overwriting) |
| `--add-to-github` | Upload the public key with `gh ssh-key add` |

What it does:

1. Writes `~/.ssh/id_<type>_<alias>` and `.pub` **without a passphrase**, with modes `600` and `644`.
2. Adds the key to your running `ssh-agent` (`ssh-add`).
3. If the account exists, stores the new key path on it.
4. Appends GitHub's and GitLab's published host keys for `github.com` and `gitlab.com` to `~/.ssh/known_hosts` if missing. These are built into GitShift; compare them with the fingerprints the providers publish.
5. Prints the public key and fingerprint and copies the public key to the clipboard (`pbcopy`, `xclip`, `xsel`, `wl-copy` or `clip`).

Then add the public key to the provider (GitHub: Settings, SSH and GPG keys; GitLab: Preferences, SSH Keys), or use `--add-to-github`.

`--add-to-github` uploads the key to the account you named, using the GitHub CLI login for that account's username and host; the active `gh` account does not change. The account must already exist as a GitHub account, and `gh` must be logged in as its user (`gh auth login --hostname HOST`). It does not work for GitLab.

### Use your own key instead

To protect a key with a passphrase, create it yourself and attach it:

```bash
ssh-keygen -t ed25519 -C alex@company.example -f ~/.ssh/id_ed25519_work
chmod 600 ~/.ssh/id_ed25519_work
gitshift update work --ssh-key ~/.ssh/id_ed25519_work
```

Git uses the key through `core.sshCommand` with the SSH agent disabled, so a passphrase-protected key prompts for its passphrase on every Git operation over SSH. See the [security model](../explanation/security-model.md#git-ssh-command).

## Test SSH access

```bash
gitshift ssh-test work
gitshift ssh-test                # the current account
gitshift ssh-test --all          # every account
gitshift ssh-test work --verbose # show the ssh output
```

For each account it checks, and reports every failure:

- the private key and `.pub` file exist,
- the private key is owner-only, `600` or `400` (a `.pub` mode other than `644` is only a warning),
- `~/.ssh/known_hosts` contains the account's host (hashed entries are recognised),
- `ssh -T git@HOST` authenticates with only this key (your `~/.ssh/config` and agent are ignored, host key checking is strict),
- whether the key is loaded in your agent (information only).

`ssh-test` is read-only. It never edits `known_hosts`.

## Create a GPG signing key

The account must exist and have a name.

```bash
gitshift gpg-keygen work --enable
gitshift gpg-keygen work --type ECC --expire-date 2y --enable
```

| Flag | Meaning |
| --- | --- |
| `--type` | `RSA` (default), `ECC` or `DSA`; case-insensitive, `ed25519` means `ECC`. Unrecognised values become `RSA` |
| `--bits` | Key size, default 4096; ignored for ECC |
| `--email` | Defaults to the account's email; must match a verified email on the provider |
| `--expire-date` | `0` (never, default), `1y`, `2y` and so on |
| `--enable` | Turn on commit and tag signing for the account |
| `--force`, `-f` | Replace the key already attached to the account |

It generates the key **without a passphrase** in your GPG keyring, stores its ID and fingerprint on the account, writes the public key to `~/.gnupg/gitshift-<alias>-public.asc` and copies it to the clipboard. Add the public key to the provider, then `gitshift switch ALIAS` to apply the signing settings.

`--force` only changes which key the account uses. The previous key stays in your keyring.

### Use an existing GPG key

```bash
gitshift update work --auto-gpg
```

This attaches the first secret key in your keyring whose email matches the account's email and turns signing on. It fails with an error if no key matches.

### Turn signing on or off

```bash
gitshift update work --gpg-signing          # sign commits and tags
gitshift update work --gpg-signing=false    # keep the key, stop signing
```

Switch to the account again to apply the change to Git.

`add --enable-gpg` only records the preference; it does not create or select a key.
