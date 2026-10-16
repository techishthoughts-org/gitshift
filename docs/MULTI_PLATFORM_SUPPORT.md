# GitHub and GitLab support

GitShift supports two provider types. A custom domain means an enterprise installation of one of these providers; it does not mean an arbitrary Git hosting service.

| Provider | Default domain | Self-hosted domain | Provider CLI action |
| --- | --- | --- | --- |
| GitHub | `github.com` | GitHub Enterprise hostname | Selects an authenticated `gh` account |
| GitLab | `gitlab.com` | Self-managed GitLab hostname | No provider CLI action |

Bitbucket and generic/custom providers are not supported.

## GitHub

```bash
gitshift add personal \
  --name "Alex Example" --email alex@example.com \
  --username alex --platform github \
  --ssh-key "$HOME/.ssh/id_ed25519_personal"
```

GitShift runs the equivalent of `gh auth switch --hostname DOMAIN --user USERNAME` during a normal GitHub switch. Authenticate each account first with `gh auth login --hostname DOMAIN`.

GitHub Enterprise example:

```bash
gitshift add enterprise \
  --name "Alex Example" --email alex@company.example \
  --username alex --platform github \
  --domain github.company.example \
  --api-endpoint https://github.company.example/api/v3 \
  --ssh-key "$HOME/.ssh/id_ed25519_enterprise"
```

## GitLab

```bash
gitshift add client \
  --name "Alex Example" --email alex@client.example \
  --username alex --platform gitlab \
  --domain gitlab.client.example \
  --api-endpoint https://gitlab.client.example/api/v4 \
  --ssh-key "$HOME/.ssh/id_ed25519_client"
```

GitShift configures Git and SSH for GitLab accounts but does not log into GitLab, upload keys, or manage tokens. `api_endpoint` is stored for metadata but is not currently called during switching.

## Repository URLs

Continue to use normal `git clone`, `git fetch`, and `git push` commands. GitShift has no `clone` command and does not automatically choose an account from a repository URL. Switch first, then use Git:

```bash
gitshift switch client --validate
git clone git@gitlab.client.example:team/project.git
```

When several accounts share a hostname, GitShift's SSH configuration and Git `core.sshCommand` select the configured identity file.
