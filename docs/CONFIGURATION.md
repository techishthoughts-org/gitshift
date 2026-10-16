# Configuration

gitshift stores YAML configuration under the user's home/configuration area.
Use the global `--config PATH` option when a separate file is required. The
exact schema is represented by the structs in `internal/models`; edit it
through CLI commands where possible.

An account contains an alias, Git author name and email, platform, provider
domain, provider username, optional API endpoint, SSH key path, and optional
GPG metadata. Older configurations may contain `github_username`; it is kept
for compatibility.

Treat the file as private even though it must not contain authentication
tokens. Keep the containing directory private and do not commit it to source
control. Paths beginning with `~` are normalized by commands that accept key
paths; explicit absolute paths are clearest.

Back up the file using normal filesystem tools. There is no `gitshift config`
subcommand.
