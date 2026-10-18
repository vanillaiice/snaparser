# Releasing Snaparser

The current release version is `2.0.0`, defined once in
`cmd/snaparser/main.go`. Release preparation creates local artifacts; publishing
is a separate step after reviewing the draft release.

## Build and review

Use Go 1.26 or newer. Packaging requires Make, `tar`, `zip`, and either
`sha256sum` or `shasum`. All release steps live in the Makefile.

```sh
make check
make build
bin/snaparser --version
make release
```

`make release` includes the checks. It builds with CGO disabled for these targets:

| Platform | Architectures | Format |
| --- | --- | --- |
| Linux | amd64, arm64 | `.tar.gz` |
| macOS (`darwin`) | amd64, arm64 | `.tar.gz` |
| Windows | amd64, arm64 | `.zip` |

Artifacts are named `snaparser_2.0.0_<os>_<arch>` and placed in `dist/v2.0.0/`.
Each archive includes the binary, README and screenshots, changelog, example TOML,
fictional JoJo demo export, project license, and frontend documentation/license.
Only explicitly listed release files are
packaged; private exports, databases, and local configuration are excluded.

Verify the checksums and run the binary from the archive for your platform:

```sh
cd dist/v2.0.0
sha256sum -c SHA256SUMS
# macOS: shasum -a 256 -c SHA256SUMS
```

Before tagging, review the changelog, inspect the working tree, commit the intended
changes, and run the packaged native binary with `--version`, `--help`, and a
configuration copied from `snaparser.example.toml`. Check the browser with your
export. Cross-compiling verifies builds; native runtime checks on each supported
platform remain useful before publishing.

## Draft and publish

When the reviewed commit is ready:

```sh
git tag -a v2.0.0 -m "Snaparser v2.0.0"
git push origin master
git push origin v2.0.0
```

Pushing a version tag triggers `.github/workflows/release.yml`. It reruns the
checks, verifies the tag matches the version compiled into the CLI, builds the
archives and checksums, and attaches them to a draft GitHub release using
`CHANGELOG.md` as the release notes. No extra repository secret is required;
the workflow uses the repository's `GITHUB_TOKEN` with release write permission.

Review the draft notes and download links before publishing the release in
GitHub. The workflow leaves the release as a draft.

## Subsequent releases

Update the version constant, add the new changelog entry, and update the README's
versioned examples. The Makefile reads the version from the CLI source;
`RELEASE_VERSION` can assert the expected version but cannot change the binary's
version. Set `RELEASE_DIR` to choose a different output directory. CI checks the
minimum supported Go version and the current stable Go
version. The release workflow builds with the current stable Go version.
