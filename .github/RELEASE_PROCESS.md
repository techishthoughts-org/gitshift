# Release process

GitShift uses one manually dispatched workflow. The version is an explicit maintainer decision; it is never inferred from commit-message substrings.

## Preconditions

- All intended changes are merged to `main` and CI is green.
- Choose a stable semantic version greater than every existing stable tag, written exactly as `vMAJOR.MINOR.PATCH`.
- Confirm the public API/behavior warrants that major, minor, or patch increment.
- Do not create the tag by hand; the workflow creates one annotated tag after verification.

The repository no longer has a `VERSION` file. Release workflow input is the release-version authority; Git tags remain the authority for local builds and published history.

## Dispatch

From GitHub, open Actions → Release → Run workflow, select `main`, and enter the exact version.

With GitHub CLI:

```bash
gh workflow run release.yml --ref main -f version=v1.5.0
gh run watch
```

Manual workflow dispatch requires repository write access. Repository rules should limit release authority appropriately.

## Safety gates

Before any remote mutation, the workflow:

1. requires dispatch from `main`;
2. confirms the dispatched commit is still `origin/main` HEAD, then checks again immediately before tagging;
3. validates strict stable semver with no leading-zero components or prerelease suffix;
4. rejects an existing tag or release;
5. requires the requested version to exceed the highest strict stable semver tag;
6. requires that prior tag to be an ancestor with at least one later commit;
7. runs race-enabled tests and vet;
8. builds and verifies five binaries and `checksums.txt`.

Only then does it create an annotated tag at the immutable dispatch SHA and publish one GitHub Release containing the verified files. The same workflow performs both operations, so it does not depend on tag-push event recursion or a personal access token.

## Verify a release

```bash
gh release view v1.5.0 --json tagName,isDraft,isPrerelease,assets
verify_dir="$(mktemp -d)"
gh release download v1.5.0 --dir "$verify_dir"
cd "$verify_dir"
sha256sum --check checksums.txt
```

Run the native binary's `--version` and `--help` on representative operating systems. Confirm the reported version includes the full `v`-prefixed tag and that all six expected assets are present.

## Failure recovery

Validation, testing, or building failures occur before remote mutation: fix the problem on `main` and dispatch again with the same intended version.

Tag creation and release publication are separate remote operations. If the annotated tag was pushed but `gh release create` failed:

1. Do not move or delete the tag merely to rerun automation.
2. Confirm the tag is annotated and targets the verified `main` commit.
3. Rebuild that exact tagged commit using the workflow's documented commands or retrieve intact verified run artifacts if available.
4. Create the missing release against the existing tag with `gh release create TAG dist/* --verify-tag --generate-notes --latest`.
5. Download the published assets and verify `checksums.txt`.

If a release is partially created, keep it as a draft while correcting its assets. Never silently replace assets on an already announced immutable release; publish a new patch version when provenance is uncertain.

The installation guide's legacy-release warning must remain until a release from this pipeline is downloaded and tested independently.
