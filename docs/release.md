# Release Guide

Release flow:

1. Ensure the working tree is clean.
2. Run `make test vet openapi-check dist`.
3. Create and push a version tag, for example `v0.1.0` or `v0.1.0-experimental.0`.
4. GitHub Actions builds release artifacts, attaches checksums, syncs the npm package version from the tag, and publishes `@tmcopilot/cli`.
5. Update the Homebrew formula checksums from `dist/checksums.txt`.

GitHub Actions npm publishing requires a repository secret named `NPM_TOKEN`.
Create an npm token with publish access to `@tmcopilot/cli`, then add it under:

```text
GitHub repository settings -> Secrets and variables -> Actions -> Repository secrets
```

The release workflow treats the Git tag as the source of truth for npm package version. For example, pushing tag `v0.2.0` runs:

```bash
npm version 0.2.0 --no-git-tag-version --allow-same-version
```

inside the workflow before `npm pack` and `npm publish`. The repository's checked-in `package.json` version does not need to be edited before every tag release.

Pre-release tags such as `v0.1.0-experimental.0` are published to npm with the `experimental` dist-tag and marked as GitHub pre-releases. Stable tags are published with the `latest` dist-tag.

Scoped npm packages are private by default on first publish, so the workflow uses:

```bash
npm publish --access public
```

Supported artifacts:

- `tmc-<version>-darwin-arm64.tar.gz`
- `tmc-<version>-darwin-amd64.tar.gz`
- `tmc-<version>-linux-arm64.tar.gz`
- `tmc-<version>-linux-amd64.tar.gz`
- `tmc-<version>-windows-amd64.zip`
- `checksums.txt`

The Git tag keeps the leading `v`, for example `v0.1.0`. Artifact names omit it, for example `tmc-0.1.0-windows-amd64.zip`.

Local install from source:

```bash
make install VERSION=dev
```

Install from the latest stable release:

```bash
npx @tmcopilot/cli@latest install
```

Install from an experimental release:

```bash
npx @tmcopilot/cli@experimental install
```

macOS/Linux shell fallback:

```bash
curl -fsSL https://raw.githubusercontent.com/huski-inc/tmcopilot-cli/main/scripts/install.sh | sh
```
