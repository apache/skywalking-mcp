# How to Release Apache SkyWalking MCP

This document describes the full release process for Apache SkyWalking MCP, following the [Apache Software Foundation release policy](https://www.apache.org/legal/release-policy.html).

## Prerequisites

- Apache committer account with SVN write access to `dist.apache.org`
- GPG key registered in the [Apache KEYS file](https://dist.apache.org/repos/dist/release/skywalking/KEYS)
- Tools installed: `git`, `gpg`, `shasum`, `svn`, `make`

### Verify your GPG key is ready

```bash
gpg --list-secret-keys --keyid-format LONG
```

If no key exists, generate one using your Apache email address:

```bash
gpg --full-generate-key
gpg --keyserver keys.openpgp.org --send-keys <YOUR_KEY_ID>
```

Then append your public key to the [Apache KEYS file](https://dist.apache.org/repos/dist/release/skywalking/KEYS) via SVN.

---

## Step 1 — Update CHANGES.md

Add a section for the new version to `CHANGES.md` and commit to `main`.

---

## Step 2 — Create and push a Git tag

```bash
export VERSION=0.1.0-rc0   # no "v" prefix; used throughout all steps below
git tag v${VERSION}
git push origin v${VERSION}
```

---

## Step 3 — Build and sign the artifacts

`release.sh` resolves the version from the **latest tag reachable in the repo** (via `git describe --tags`), not from the `VERSION` env var. It also resolves a single release commit and uses that same commit for both the source archive and the binary build. To ensure the artifacts are stamped with the intended version, use one of these approaches:

**Option A (recommended): set `RELEASE_VERSION` explicitly**

```bash
RELEASE_VERSION=${VERSION} make release-assembly
```

**Option B: ensure the new tag is the latest in the repo**

If `v${VERSION}` is already the most recent tag, `make release-assembly` picks it up automatically without any env override.

> If another newer tag exists in the repo, Option A is required to avoid building artifacts with the wrong version.

By default, the release commit is resolved from tag `v${VERSION}` when it exists. You can override it explicitly with `RELEASE_GIT_COMMIT=<commit>` when you need to release from a specific commit.

This runs three steps in sequence:
1. `release-source` — creates `build/skywalking-mcp-${VERSION}-src.tgz`
2. `release-binary` — creates `build/skywalking-mcp-${VERSION}.tgz`
3. `release-sign` — creates signatures and checksums for both tarballs:
    - `build/skywalking-mcp-${VERSION}.tgz.asc` and `build/skywalking-mcp-${VERSION}.tgz.sha512`
    - `build/skywalking-mcp-${VERSION}-src.tgz.asc` and `build/skywalking-mcp-${VERSION}-src.tgz.sha512`

---

## Step 4 — Upload to Apache dist/dev and send vote email

```bash
make release-push-candidate
```

This script:
1. SVN-checks out `https://dist.apache.org/repos/dist/dev/skywalking/mcp/`
2. Copies the tarballs, signature, and checksum into `mcp/${VERSION}/`
3. Commits to SVN
4. Prints a vote email template to stdout

Copy the printed email and send it to `dev@skywalking.apache.org`.

---

## Step 5 — Community vote

The vote must remain open for **at least 72 hours**. It requires:
- At least **3 PMC member +1 votes**
- No binding **-1 votes**

Refer to the [SkyWalking vote check guide](https://github.com/apache/skywalking/blob/master/docs/en/guides/How-to-release.md#vote-check) for how to verify a release candidate.

---

## Step 6 — Promote the release (after vote passes)

Move the artifacts from `dist/dev` to `dist/release`:

```bash
svn mv \
  https://dist.apache.org/repos/dist/dev/skywalking/mcp/${VERSION} \
  https://dist.apache.org/repos/dist/release/skywalking/mcp/${VERSION} \
  -m "Release Apache SkyWalking MCP ${VERSION}"
```

Remove the previous release from `dist/release` to keep only the latest:

```bash
svn rm \
  https://dist.apache.org/repos/dist/release/skywalking/mcp/<PREVIOUS_VERSION> \
  -m "Remove old Apache SkyWalking MCP release"
```

---

## Step 7 — Create the GitHub Release

Go to **GitHub → Releases → Create a release**, select tag `v${VERSION}`, and publish it.

This automatically triggers two CI workflows:
- **publish-docker** — pushes `apache/skywalking-mcp:v${VERSION}` to Docker Hub
- **publish-binaries** — builds 4 platform binaries (linux/darwin × amd64/arm64) and attaches them as release assets

---

## Step 8 — Announce the release

Send an announcement email to `announce@apache.org` and `dev@skywalking.apache.org`:

```
Subject: [ANNOUNCE] Apache SkyWalking MCP {VERSION} Released

Hi All,

The Apache SkyWalking Team is glad to announce the release of Apache SkyWalking MCP {VERSION}.

SkyWalking: APM (application performance monitor) tool for distributed systems, especially designed for microservices, cloud native and container-based (Docker, Kubernetes, Mesos) architectures.

SkyWalking MCP: an MCP (Model Context Protocol) server that bridges AI agents with Apache SkyWalking OAP via GraphQL. It exposes SkyWalking's observability data (traces, logs, metrics, topology, alarms, events) as MCP tools, prompts, and resources.

Please refer to the change log for the complete list of changes: https://github.com/apache/skywalking-mcp/releases/tag/v{VERSION}

Apache SkyWalking website: http://skywalking.apache.org/

Downloads: http://skywalking.apache.org/downloads/

Twitter: https://twitter.com/ASFSkyWalking

SkyWalking Resources:
- GitHub: https://github.com/apache/skywalking
- Issue: https://github.com/apache/skywalking/issues
- Mailing list: dev@skywalking.apache.org <mailto:dev@skywalking.apache.org>

- The Apache SkyWalking Team
