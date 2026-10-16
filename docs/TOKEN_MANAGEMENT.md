# GitHub credentials

GitHub credentials are managed by GitHub CLI, not by gitshift.

```sh
gh auth login --hostname github.com
gh auth status --hostname github.com
gh auth switch --hostname github.com --user YOUR_LOGIN
```

For GitHub Enterprise, replace the hostname. gitshift's GitHub platform switch
uses the account's domain and username. GitLab/custom accounts do not run
GitHub CLI.

Do not export long-lived tokens from shell startup files or copy them into MCP
configuration. For automation, follow GitHub CLI's documented environment or
standard-input mechanisms in the automation environment itself. gitshift has
no `github-token` command and does not fetch token plaintext with `gh auth
token`.
