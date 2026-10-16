# Security

## Key generation

gitshift never accepts SSH or GPG passphrases as command-line flags. Command
arguments can be visible in shell history and process listings. The built-in
generators therefore create passwordless keys. Use native interactive tooling
for protected keys:

```sh
ssh-keygen -t ed25519 -f "$HOME/.ssh/id_ed25519_work"
gitshift update work --ssh-key "$HOME/.ssh/id_ed25519_work"
```

Protect private keys with filesystem permissions and an agent appropriate for
your operating system.

## Credentials

gitshift does not store GitHub tokens. GitHub CLI owns credentials in the
system credential store. The legacy `scripts/sync-github-tokens.sh` filename
is retained, but the script only selects an already-authenticated host/user
and never reads or copies a token.

## Scope

Switching modifies user Git configuration, SSH configuration/agent state, GPG
settings, and possibly GitHub CLI active-account state. Use `switch --validate`
first. Do not run untrusted configuration or key paths.

Report suspected vulnerabilities privately to the maintainers; do not include
keys, tokens, or complete configuration files in reports.
