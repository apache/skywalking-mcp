#!/usr/bin/env bash

# Licensed to the Apache Software Foundation (ASF) under one or more
# contributor license agreements.  See the NOTICE file distributed with
# this work for additional information regarding copyright ownership.
# The ASF licenses this file to You under the Apache License, Version 2.0
# (the "License"); you may not use this file except in compliance with
# the License.  You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
set -ex

if [ "$VERSION" == "" ]; then
  echo "VERSION environment variable not found, Please setting the VERSION."
  echo "For example: export VERSION=1.0.0"
  exit 1
fi

VERSION=${VERSION}
TAG_NAME=v${VERSION}
PRODUCT_NAME="skywalking-mcp-${VERSION}"
SVN_DIST_URL=${SVN_DIST_URL:-https://dist.apache.org/repos/dist/dev/skywalking/}

svn_auth_args=()
if [ -n "${SVN_USERNAME:-}" ]; then
  svn_auth_args+=(--username "$SVN_USERNAME")
fi
if [ -n "${SVN_PASSWORD:-}" ]; then
  svn_auth_args+=(--password "$SVN_PASSWORD")
fi
if [ ${#svn_auth_args[@]} -gt 0 ]; then
  svn_auth_args+=(--non-interactive)
fi

require_cmd() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "Error: required command '$1' not found in PATH" >&2
    exit 1
  fi
}

ensure_clean_worktree() {
  if [ -n "$(git status --porcelain)" ]; then
    echo "Error: Git working tree is not clean. Commit or stash changes before creating a release tag." >&2
    git status --short >&2
    exit 1
  fi
}

echo "Release version "${VERSION}
echo "Source tag "${TAG_NAME}

SCRIPTDIR=$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )
ROOTDIR=${SCRIPTDIR}/..
BUILDDIR=${ROOTDIR}/build

require_cmd git
require_cmd make
require_cmd svn
require_cmd gpg
require_cmd tar
require_cmd shasum

ensure_clean_worktree

# --- Tagging ---
# Create the local tag if it doesn't already exist.
if git rev-parse "$TAG_NAME" >/dev/null 2>&1; then
  echo "Git tag ${TAG_NAME} already exists, skipping tag creation."
else
  git tag -a "$TAG_NAME" -m "Release ${TAG_NAME}"
  echo "Created git tag ${TAG_NAME}"
fi

# --- Build & sign release artifacts ---
# Run from the repo root so make targets resolve correctly.
(cd "${ROOTDIR}" && RELEASE_VERSION="${VERSION}" make release-assembly)

# Push the tag only after a successful build so a failed build doesn't
# leave a dangling tag on the remote.
git push origin "$TAG_NAME"

pushd "${BUILDDIR}"
trap 'popd' EXIT

rm -rf "${BUILDDIR}/skywalking"

svn "${svn_auth_args[@]}" co "${SVN_DIST_URL}"
mkdir -p skywalking/mcp/"$VERSION"
cp ${PRODUCT_NAME}-*.tgz skywalking/mcp/"$VERSION"
cp ${PRODUCT_NAME}-*.tgz.asc skywalking/mcp/"$VERSION"
cp ${PRODUCT_NAME}-*.tgz.sha512 skywalking/mcp/"$VERSION"

cd skywalking/mcp && svn "${svn_auth_args[@]}" add "$VERSION" && svn "${svn_auth_args[@]}" commit -m "Draft Apache SkyWalking MCP release $VERSION"
cd "$VERSION"

cat << EOF
=========================================================================
Subject: [VOTE] Release Apache SkyWalking MCP version $VERSION

Content:

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

 * https://github.com/apache/skywalking-mcp/tree/$(git rev-list -n 1 "$TAG_NAME")

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
EOF
