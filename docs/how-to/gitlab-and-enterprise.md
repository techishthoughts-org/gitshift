# Use self-managed GitLab or GitHub Enterprise

GitShift supports two platforms, `github` and `gitlab`. A custom `--domain` means a self-hosted installation of one of them. Bitbucket and other hosts are not supported.

| Platform | Default host | Custom host | Provider CLI on switch |
| --- | --- | --- | --- |
| `github` | `github.com` | GitHub Enterprise Server or Cloud hostname | `gh auth switch --hostname HOST --user USERNAME` |
| `gitlab` | `gitlab.com` | self-managed GitLab hostname | none |

## GitLab.com

```bash
gitshift add client \
  --platform gitlab \
  --name "Alex Example" --email alex@client.example \
  --username alex-client \
  --ssh-key ~/.ssh/id_ed25519_client
```

## Self-managed GitLab

```bash
gitshift add company \
  --platform gitlab \
  --domain gitlab.company.example \
  --api-endpoint https://gitlab.company.example/api/v4 \
  --name "Alex Example" --email alex@company.example \
  --username alex \
  --ssh-key ~/.ssh/id_ed25519_company
```

GitShift configures Git and SSH for GitLab accounts. It does not log in to GitLab, run `glab`, upload keys or manage tokens. Set up HTTPS credentials or `glab` yourself if you need them.

## GitHub Enterprise

Log the account in to the GitHub CLI on that host first:

```bash
gh auth login --hostname github.company.example
```

Then add it:

```bash
gitshift add enterprise \
  --domain github.company.example \
  --api-endpoint https://github.company.example/api/v3 \
  --name "Alex Example" --email alex@company.example \
  --username alex \
  --ssh-key ~/.ssh/id_ed25519_enterprise
```

`--platform github` is the default, so it can be left out.

## Rules for `--domain` and `--api-endpoint`

- `--domain` is a bare hostname or IP address: no scheme, path, user or port. `https://gitlab.company.example` and `gitlab.company.example:2222` are rejected.
- GitShift connects as `git@HOST` on the standard SSH port. Instances that serve SSH on another port are not supported, because Git runs SSH with your `~/.ssh/config` disabled (see the [security model](../explanation/security-model.md#git-ssh-command)).
- `--api-endpoint` must be an `https://` URL without credentials. It is stored for reference only; no command calls it.
- To change an existing account: `gitshift update ALIAS --domain HOST`, or `--domain ""` to return to `github.com`/`gitlab.com`.

## Trust the host key

`gitshift ssh-keygen` adds host keys only for `github.com` and `gitlab.com`. For your own instance, get the fingerprint from your administrators, then:

```bash
ssh-keyscan gitlab.company.example > /tmp/company_host_keys
ssh-keygen -lf /tmp/company_host_keys          # compare with the published fingerprint
cat /tmp/company_host_keys >> ~/.ssh/known_hosts
gitshift ssh-test company
```

Never accept an unexpected host key to make a test pass.

## Several accounts on one host

Accounts on the same host share one managed block in `~/.ssh/config`; each switch rewrites it with the new key, and `core.sshCommand` pins Git to that key. Clone and push with normal URLs after switching:

```bash
gitshift switch company
git clone git@gitlab.company.example:team/project.git
```

GitShift does not pick an account from a repository URL. Use [directory rules](auto-switch.md) for that.
