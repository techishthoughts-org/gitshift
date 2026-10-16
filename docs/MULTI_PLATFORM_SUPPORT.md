# Platform behavior

## GitHub

The default domain is `github.com`; GitHub Enterprise uses the account's
configured domain. Switching can invoke GitHub CLI with both the hostname and
username. Authenticate each account beforehand with `gh auth login
--hostname HOST`.

## GitLab

The default domain is `gitlab.com`; self-hosted instances use `--domain`.
Switching uses Git/SSH configuration and does not invoke `gh`.

## Custom hosts

Custom SSH Git hosts can be represented by an explicit domain and username.
Provider-specific API or CLI integration is not promised.

Bitbucket is not currently implemented.
