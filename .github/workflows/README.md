# GitHub Actions workflows

| Workflow | Trigger | What it does |
| --- | --- | --- |
| `ci.yml` | Push or pull request targeting `main`/`develop` | Race-enabled tests, vet, build, generated command reference is current, `goreleaser check` |
| `docs.yml` | Markdown changes (push to `main`, pull requests); manual | Checks every relative link and anchor between Markdown files, offline |
| `security.yml` | Push, pull request, daily, manual | `govulncheck`; gosec and Trivy are non-blocking; dependency review on pull requests |
| `release.yml` | Manual only | Validates the version, tests, builds a snapshot, pushes the tag, publishes the release and Homebrew cask with GoReleaser, then verifies both |

Third-party actions are pinned to full commit SHAs with the release tag in a comment. Workflows default to `contents: read`; only `release.yml` has `contents: write`, and checkouts do not persist credentials.

`release.yml` is the only workflow that creates tags, releases, or commits to `techishthoughts-org/homebrew-tap`. The tap commit needs the `HOMEBREW_TAP_TOKEN` secret, scoped to that one repository. See [Releasing](../../docs/maintainers/releasing.md).

## Known limitations

- gosec and Trivy use `continue-on-error: true`; their findings do not fail the job.
- No workflow signs artifacts, publishes provenance attestations, or uploads coverage.
