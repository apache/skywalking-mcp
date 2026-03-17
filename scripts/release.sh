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

set -euo pipefail
set -x

SCRIPTDIR=$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )
ROOTDIR=${SCRIPTDIR}/..
BUILDDIR=${ROOTDIR}/build

require_cmd() {
    if ! command -v "$1" >/dev/null 2>&1; then
        echo "Error: required command '$1' not found in PATH" >&2
        exit 1
    fi
}

resolve_release_version() {
    # 1) Explicit environment override
    if [ -n "${RELEASE_VERSION:-}" ]; then
        echo "${RELEASE_VERSION}"
        return 0
    fi

    # 2) Derive from Git tags when available
    if command -v git >/dev/null 2>&1 && git rev-parse --is-inside-work-tree >/dev/null 2>&1; then
        if latest_tag=$(git describe --tags "$(git rev-list --tags --max-count=1)" 2>/dev/null); then
            echo "${latest_tag#v}"
            return 0
        fi

        # 3) Fallback to dev-<short-commit> when no tags are present
        if short_commit=$(git rev-parse --short HEAD 2>/dev/null); then
            # Optional guard for CI: require a tag when STRICT_RELEASE_TAG=1
            if [ "${STRICT_RELEASE_TAG:-0}" = "1" ]; then
                echo "Error: STRICT_RELEASE_TAG=1 is set but no Git tag could be resolved for this commit." >&2
                exit 1
            fi
            echo "dev-${short_commit}"
            return 0
        fi
    fi

    # 4) Last-resort fallback when Git is unavailable
    if [ "${STRICT_RELEASE_TAG:-0}" = "1" ]; then
        echo "Error: STRICT_RELEASE_TAG=1 is set but neither RELEASE_VERSION nor Git metadata are available." >&2
        exit 1
    fi

    echo "dev-unknown"
}

RELEASE_VERSION=$(resolve_release_version)

SOURCE_FILE_NAME=skywalking-mcp-${RELEASE_VERSION}-src.tgz
SOURCE_FILE=${BUILDDIR}/${SOURCE_FILE_NAME}

binary(){
    require_cmd tar

    if [ ! -f "${SOURCE_FILE}" ]; then
        echo "Source archive ${SOURCE_FILE} does not exist. Run '${0} -s' first to assemble the source package." >&2
        exit 1
    fi

    (
        tmpdir=$(mktemp -d)
        trap 'rm -rf "${tmpdir}"' EXIT

        tar -xvf "${SOURCE_FILE}" -C "${tmpdir}"
        cd "${tmpdir}"
        make build

        bindir=./build
        mkdir -p "${bindir}/bin"
        # Copy relevant files
        cp -Rfv ./bin/* "${bindir}/bin"
        cp -Rfv ./CHANGES.md "${bindir}"
        cp -Rfv ./README.md "${bindir}"
        cp -Rfv ./dist/* "${bindir}"
        tar -czf "${BUILDDIR}/skywalking-mcp-${RELEASE_VERSION}.tgz" -C "${bindir}" .
    )
}

source(){
    require_cmd tar

    (
        tmpdir=$(mktemp -d)
        trap 'rm -rf "${tmpdir}"' EXIT

        rm -rf "${SOURCE_FILE}"
        cd "${ROOTDIR}"
        echo "RELEASE_VERSION=${RELEASE_VERSION}" > .env
        tar \
        --exclude=".DS_Store" \
        --exclude=".github" \
        --exclude=".gitignore" \
        --exclude=".asf.yaml" \
        --exclude=".idea" \
        --exclude=".vscode" \
        --exclude="bin" \
        -czf "${tmpdir}/${SOURCE_FILE_NAME}" \
        .

        mkdir -p "${BUILDDIR}"
        mv "${tmpdir}/${SOURCE_FILE_NAME}" "${BUILDDIR}"
    )
}

sign(){
    require_cmd gpg
    require_cmd shasum

    pushd "${BUILDDIR}" >/dev/null
    trap 'popd >/dev/null' EXIT

    gpg --batch --yes --armor --detach-sig "skywalking-mcp-${RELEASE_VERSION}-src.tgz"
    shasum -a 512 "skywalking-mcp-${RELEASE_VERSION}-src.tgz" > "skywalking-mcp-${RELEASE_VERSION}-src.tgz.sha512"
    gpg --batch --yes --armor --detach-sig "skywalking-mcp-${RELEASE_VERSION}.tgz"
    shasum -a 512 "skywalking-mcp-${RELEASE_VERSION}.tgz" > "skywalking-mcp-${RELEASE_VERSION}.tgz.sha512"
}

parseCmdLine(){
    if [ $# -eq 0 ]; then
        echo "Exactly one argument required." >&2
        usage
    fi
    while getopts  "bsk:vh" FLAG; do
        case "${FLAG}" in
            b) binary ;;
            s) source ;;
            k) sign "${OPTARG}" ;;
            v) echo "Resolved RELEASE_VERSION=${RELEASE_VERSION}" ;;
            h) usage ;;
            \?) usage ;;
        esac
    done
    return 0
}

usage() {
cat <<EOF
Usage:
    ${0} -[bskvh]

Parameters:
    -b  Build and assemble the binary package
    -s  Assemble the source package
    -k  Sign the specified artifact type (currently 'mcp')
    -v  Print the resolved RELEASE_VERSION and exit
    -h  Show this help.
EOF
    exit 1
}

#
# main
#

require_cmd git

ret=0

parseCmdLine "$@"
ret=$?
[ "${ret}" -ne 0 ] && exit "${ret}"
echo "Done release ${RELEASE_VERSION} (exit ${ret})"
