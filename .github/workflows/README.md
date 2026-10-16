# GitHub Actions workflows

This inventory describes the checked-in workflows and whether their findings block a job.

| Workflow | Trigger | Current behavior |
| --- | --- | --- |
| `ci.yml` | Push or pull request targeting `main`/`develop` | Downloads modules, runs race-enabled tests, vet, and a normal build |
| `docs.yml` | Documentation changes on `main`; manual | Checks documentation links, required files, and metrics |
| `security.yml` | Push, PR, daily schedule, manual | Runs `govulncheck`; gosec and Trivy are non-blocking; dependency review runs on PRs; license output is informational |
| `release.yml` | Manual only | Validates an explicit stable semver, tests, builds, verifies, tags, and publishes one release |

## Release ownership

`release.yml` is the only workflow allowed to create tags or GitHub Releases. There is no commit-message version inference, tag-push workflow chain, personal access token, or second publisher.

A maintainer supplies an exact `vMAJOR.MINOR.PATCH` value, such as `v1.5.0`. The workflow rejects malformed, existing, equal, or lower versions; dispatches outside `main`; stale `main` commits; and histories where the highest stable tag is not an ancestor.

The release job has `contents: write`; all other workflow permissions remain job-specific or default. Release runs are serialized and never auto-cancelled.

## Release assets

The release contract is exactly:

- `gitshift-linux-amd64`
- `gitshift-linux-arm64`
- `gitshift-darwin-amd64`
- `gitshift-darwin-arm64`
- `gitshift-windows-amd64.exe`
- `checksums.txt`

Before creating a tag, the workflow verifies tests, vet, checksums, file presence, the runnable Linux version, and the linker version embedded in every binary. After publishing, it verifies the annotated tag target and exact remote asset-name set.

## Known limitations

- gosec and Trivy use `continue-on-error: true`; their findings do not independently fail the security job.
- The license job lists modules and fixed summary text; it is not a complete policy scanner.
- No checked-in workflow currently runs a dedicated secret scanner, publishes a documentation site, uploads code coverage, or signs release artifacts.
- `make release` is a local convenience path and is not the release authority. Docker version metadata comes from its explicit `VERSION` build argument. Git tags supply local Makefile build metadata.

See [the release process](../RELEASE_PROCESS.md) for dispatch and recovery instructions.
