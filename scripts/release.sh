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
#

set -ex
SCRIPTDIR=$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )
ROOTDIR=${SCRIPTDIR}/..
BUILDDIR=${ROOTDIR}/build

RELEASE_TAG=$(git describe --tags $(git rev-list --tags --max-count=1))
RELEASE_VERSION=${RELEASE_TAG#"v"}

SOURCE_FILE_NAME=skywalking-mcp-${RELEASE_VERSION}-src.tgz
SOURCE_FILE=${BUILDDIR}/${SOURCE_FILE_NAME}

binary(){
    if [ ! -f "${SOURCE_FILE}" ]; then
        echo "$FILE exists."
        exit 1
    fi
    tmpdir=`mktemp -d`
    trap "rm -rf ${tmpdir}" EXIT
    pushd ${tmpdir}
    trap 'popd' EXIT
    tar -xvf ${SOURCE_FILE}
    TARGET_OS=linux PLATFORMS=linux/amd64,linux/arm64 make release

    bindir=./build
    mkdir -p ${bindir}/bin
    # Copy relevant files
    cp -Rfv ./bin/* ${bindir}/bin
    cp -Rfv ./CHANGES.md ${bindir}
    cp -Rfv ./README.md ${bindir}
    cp -Rfv ./dist/* ${bindir}
    tar -czf ${BUILDDIR}/skywalking-mcp-${RELEASE_VERSION}.tgz -C ${bindir} .
}

source(){
    # Package
    tmpdir=`mktemp -d`
    trap "rm -rf ${tmpdir}" EXIT
    rm -rf ${SOURCE_FILE}
    pushd ${ROOTDIR}
    echo "RELEASE_VERSION=${RELEASE_VERSION}" > .env
    tar \
    --exclude=".DS_Store" \
    --exclude=".github" \
    --exclude=".gitignore" \
    --exclude=".asf.yaml" \
    --exclude=".idea" \
    --exclude=".vscode" \
    --exclude="bin" \
    -czf ${tmpdir}/${SOURCE_FILE_NAME} \
    .

    mkdir -p ${BUILDDIR}
    mv ${tmpdir}/${SOURCE_FILE_NAME} ${BUILDDIR}
    popd
}

sign(){
    pushd ${BUILDDIR}
    gpg --batch --yes --armor --detach-sig skywalking-mcp-${RELEASE_VERSION}.tgz
    shasum -a 512 skywalking-mcp-${RELEASE_VERSION}.tgz > skywalking-mcp-${RELEASE_VERSION}.tgz.sha512
    popd
}

parseCmdLine(){
    ARGS=$1
    if [ $# -eq 0 ]; then
        echo "Exactly one argument required."
        usage
    fi
    while getopts  "bsk:h" FLAG; do
        case "${FLAG}" in
            b) binary ;;
            s) source ;;
            k) sign ;;
            h) usage ;;
            \?) usage ;;
        esac
    done
    return 0
}



usage() {
cat <<EOF
Usage:
    ${0} -[bsh]

Parameters:
    -b  Build and assemble the binary package
    -s  Assemble the source package
    -h  Show this help.
EOF
    exit 1
}

#
# main
#

ret=0

parseCmdLine "$@"
ret=$?
[ $ret -ne 0 ] && exit $ret
echo "Done release ${RELEASE_VERSION} (exit $ret)"
