#!/bin/bash

cd $(dirname $0)/../

_OS=${1}
_ARCH=${2}
_OUTPUT_PATH=${3}

GOOS="${_OS}" GOARCH="${_ARCH}" go build -ldflags "-X main.metaVersion=$(git describe --tags --abbrev=0) -X main.metaRevision=$(git rev-parse --short HEAD)" -o "${_OUTPUT_PATH}" ./src
