# Apache SkyWalking MCP Release Guide

This document describes how to create Apache SkyWalking MCP release candidates locally or through GitHub Actions.

## Prerequisites

Before running a release, make sure the release environment has these tools available:

- `git`
- `make`
- `svn`
- `gpg`
- `tar`
- `shasum`
- Go toolchain

You also need:

- a **clean Git working tree**
- permission to push tags to `origin`
- Apache SVN credentials for `https://dist.apache.org/repos/dist/dev/skywalking/`
- a configured GPG key that can sign release artifacts non-interactively if your environment requires it

## Local release candidate flow

Set the target release version and run the release target:

```bash
export VERSION=0.1.0
make tag-release
```

This target will:

1. create the annotated Git tag `v$VERSION` if it does not already exist
2. build the source and binary release archives
3. generate `.asc` and `.sha512` files for both archives
4. push the Git tag to `origin`
5. upload the release candidate artifacts to Apache SVN dev dist
6. print a vote email template

## GitHub Actions release candidate flow

The repository includes a manual GitHub Actions workflow at `.github/workflows/release-candidate.yaml`.

Use this when you want CI to run the same `make tag-release` flow on a clean runner.

From the GitHub Actions UI, run the `release-candidate` workflow and provide:

- `version` — required, for example `0.1.0`
- `ref` — optional Git ref to release from, defaults to `main`

The workflow checks out the selected ref, imports the release GPG key, configures Subversion authentication, runs `make tag-release`, and uploads the generated release files as workflow artifacts.

### Required GitHub Actions secrets

Configure these repository secrets before running the workflow:

- `RELEASE_GPG_PRIVATE_KEY` — ASCII-armored private key used to sign release artifacts
- `RELEASE_GPG_PASSPHRASE` — passphrase for the GPG private key
- `APACHE_SVN_USERNAME` — Apache SVN username for `dist.apache.org`
- `APACHE_SVN_PASSWORD` — Apache SVN password for `dist.apache.org`

### When to use local vs CI release flow

- Use local `make tag-release` if you already have GPG and SVN configured on your release machine.
- Use the GitHub Actions `release-candidate` workflow if you prefer a clean, repeatable environment and secret-managed credentials.

## Release artifacts

The release flow produces these files in `build/`:

- `skywalking-mcp-$VERSION-src.tgz`
- `skywalking-mcp-$VERSION-src.tgz.asc`
- `skywalking-mcp-$VERSION-src.tgz.sha512`
- `skywalking-mcp-$VERSION.tgz`
- `skywalking-mcp-$VERSION.tgz.asc`
- `skywalking-mcp-$VERSION.tgz.sha512`

## Notes

- `make tag-release` fails fast if the Git working tree is dirty.
- The tag is pushed only after artifact assembly and signing succeed.
- If the tag already exists locally, the script reuses it and continues.
- The source release archive excludes local metadata such as `.git/` and `.claude/`.
- The GitHub Actions workflow uploads the generated release artifacts so they can be downloaded from the workflow run page.