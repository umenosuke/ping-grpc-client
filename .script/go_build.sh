#!/bin/bash

cd "$(dirname $(readlink -f $0))/../"
_BASE_DIR="$(pwd)"

source ./.script/_conf.sh

_OS=${1}
_ARCH=${2}
_SRC_PATH=${3}
_OUTPUT_PATH=${4}

CGO_ENABLED=0 GOOS="${_OS}" GOARCH="${_ARCH}" go build \
 -ldflags "-X main.metaVersion=${_GIT_TAG} -X main.metaRevision=${_GIT_HASH}" \
 -o "${_OUTPUT_PATH}" \
 "${_SRC_PATH}"
