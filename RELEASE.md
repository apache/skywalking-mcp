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

The tag must exist **before** building artifacts — `release.sh` derives the version from it.

```bash
export VERSION=0.1.0-rc0   # no "v" prefix; used throughout all steps below
git tag v${VERSION}
git push origin v${VERSION}
```

---

## Step 3 — Build and sign the artifacts

```bash
make release-assembly
```

This runs three steps in sequence:
1. `release-source` — creates `build/skywalking-mcp-${VERSION}-src.tgz`
2. `release-binary` — creates `build/skywalking-mcp-${VERSION}.tgz`
3. `release-sign` — creates `build/skywalking-mcp-${VERSION}.tgz.asc` and `.tgz.sha512`

---

## Step 4 — Upload to Apache dist/dev and send vote email

```bash
VERSION=${VERSION} make release-push-candidate
```

This script:
1. SVN-checks out `https://dist.apache.org/repos/dist/dev/skywalking/`
2. Copies the tarballs, signature, and checksum into `skywalking/mcp/${VERSION}/`
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

Hi the SkyWalking Community:

This is a call for vote to release Apache SkyWalking MCP version $VERSION.

Release notes:

 * https://github.com/apache/skywalking-mcp/blob/v$VERSION/CHANGES.md

Release Candidate:

 * https://dist.apache.org/repos/dist/dev/skywalking/mcp/$VERSION
 * sha512 checksums
   - $(cat ${PRODUCT_NAME}.tgz.sha512)

Release Tag :

 * (Git Tag) $TAG_NAME

Release Commit Hash :

 * https://github.com/apache/skywalking-mcp/tree/${RELEASE_COMMIT}

Keys to verify the Release Candidate :

 * https://dist.apache.org/repos/dist/release/skywalking/KEYS

Guide to build the release from source :

 * https://github.com/apache/skywalking-mcp/blob/v$VERSION/README.md

Voting will start now and will remain open for at least 72 hours, all PMC members are required to give their votes.

[ ] +1 Release this package.
[ ] +0 No opinion.
[ ] -1 Do not release this package because....

Thanks.

[1] https://github.com/apache/skywalking/blob/master/docs/en/guides/How-to-release.md#vote-check
