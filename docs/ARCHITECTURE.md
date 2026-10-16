# Architecture

gitshift is a Cobra CLI. Commands live in `cmd`, configuration persistence in
`internal/config`, account data in `internal/models`, discovery in
`internal/discovery`, and Git/SSH/GPG helpers in their corresponding internal
packages. `pkg/platform` contains provider behavior used to distinguish
GitHub, GitLab, and custom hosts.

Commands load the configured account, validate it, call the required local
tools, and persist account state. External processes include Git, OpenSSH,
GPG, and optionally GitHub CLI. These tools are local integration boundaries;
their failures must be returned rather than treated as successful switches.

The project does not claim a daemon, remote service, encrypted configuration,
plugin system, dependency-injection container, security audit engine, or MCP
token synchronization layer.
