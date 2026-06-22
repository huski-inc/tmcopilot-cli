# Release Guide

Release flow:

1. Ensure the working tree is clean.
2. Run `make test vet openapi-check dist`.
3. Create and push a version tag, for example `v0.1.0` or `v0.1.0-experimental.0`.
4. GitHub Actions builds release artifacts, attaches checksums, syncs the npm package version from the tag, and publishes `@tmcopilot/cli`.
5. Update the Homebrew formula checksums from `dist/checksums.txt`.

GitHub Actions publishes to npm through npm Trusted Publishing. Configure the npm package's trusted publisher as:

```text
Provider: GitHub Actions
Organization or user: huski-inc
Repository: tmcopilot-cli
Workflow filename: release.yml
Environment name: leave empty
Allowed actions: npm publish
```

The workflow uses GitHub OIDC, so no `NPM_TOKEN` secret is required for release publishing. Keep token-based publishing disabled after the trusted publisher has been verified.

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

The Git tag keeps the leading `v`, for example `v0.1.0`. Artifact names omit it, for example `tmc-0.1.0-windows-amd64.zip`. Each archive includes the primary `tmc` command and the `tmcopilot` alias.

Local install from source:

```bash
make install VERSION=dev
```

Install from the latest stable release:

```bash
npx @tmcopilot/cli@latest install
```

The npm installer persists `tmc` and `tmcopilot` to the npm global `bin` directory or another common CLI install directory. Set `TMC_INSTALL_DIR` to override the destination.

Install from an experimental release:

```bash
npx @tmcopilot/cli@experimental install
```

macOS/Linux shell fallback:

```bash
curl -fsSL https://raw.githubusercontent.com/huski-inc/tmcopilot-cli/main/scripts/install.sh | sh
```
