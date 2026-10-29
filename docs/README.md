# GitShift documentation

These pages describe GitShift v2.0.0. They are grouped by what you are trying to do.

## Tutorial

- [Getting started](getting-started.md): install, add a work and a personal account, switch, set up directory auto-switch, show the active account in your prompt.

## How-to guides

- [Install, upgrade and uninstall](how-to/install.md), including upgrading from v1.x
- [Manage accounts](how-to/manage-accounts.md): add, update, remove, discover, terminal UI
- [Switch accounts](how-to/switch-accounts.md): `--validate`, `--force`, `--skip-validation`, rollback
- [Switch automatically by directory](how-to/auto-switch.md): rules, default account, zsh hook, prompt
- [Create and test keys](how-to/keys.md): `ssh-keygen`, `gpg-keygen`, `ssh-test`
- [Use self-managed GitLab or GitHub Enterprise](how-to/gitlab-and-enterprise.md)
- [Troubleshooting](how-to/troubleshooting.md)

## Reference

- [Command reference](reference/README.md): generated from the CLI
- [Configuration file](reference/configuration.md): location, fields, permissions, environment variables

## Explanation

- [Architecture](explanation/architecture.md): packages and how a switch runs
- [Identity model](explanation/identity-model.md): default, current and auto-switch default accounts, and why switching is global
- [Security model](explanation/security-model.md): every file and setting GitShift changes, backups, rollback limits, what is never stored

## Maintainers

- [Contributing](../CONTRIBUTING.md)
- [Releasing](maintainers/releasing.md)
