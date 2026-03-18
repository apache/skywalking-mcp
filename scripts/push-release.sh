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

echo "Release version "${VERSION}
echo "Source tag "${TAG_NAME}

SCRIPTDIR=$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )
ROOTDIR=${SCRIPTDIR}/..
BUILDDIR=${ROOTDIR}/build

if ! git -C "${ROOTDIR}" show-ref --tags --verify "refs/tags/${TAG_NAME}" >/dev/null 2>&1; then
  echo "Error: Git tag '${TAG_NAME}' not found. Create and push the tag first:" >&2
  echo "  git tag ${TAG_NAME} && git push origin ${TAG_NAME}" >&2
  exit 1
fi
RELEASE_COMMIT=$(git -C "${ROOTDIR}" rev-list -n 1 "${TAG_NAME}")

pushd ${BUILDDIR}
trap 'popd' EXIT

rm -rf skywalking

svn co https://dist.apache.org/repos/dist/dev/skywalking/
mkdir -p skywalking/mcp/"$VERSION"
BINARY_TGZ="${PRODUCT_NAME}.tgz"
SRC_TGZ="${PRODUCT_NAME}-src.tgz"
cp "${BINARY_TGZ}" skywalking/mcp/"$VERSION"
cp "${BINARY_TGZ}.asc" skywalking/mcp/"$VERSION"
cp "${BINARY_TGZ}.sha512" skywalking/mcp/"$VERSION"
cp "${SRC_TGZ}" skywalking/mcp/"$VERSION"
cp "${SRC_TGZ}.asc" skywalking/mcp/"$VERSION"
cp "${SRC_TGZ}.sha512" skywalking/mcp/"$VERSION"

cd skywalking && svn add --parents mcp/"$VERSION" && svn commit -m "Draft Apache SkyWalking MCP release $VERSION"
cd mcp/"$VERSION"

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
   - $(cat ${PRODUCT_NAME}-src.tgz.sha512)

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
EOF
