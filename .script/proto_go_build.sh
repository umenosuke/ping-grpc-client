#!/bin/bash

cd "$(dirname $(readlink -f $0))/../"
_BASE_DIR="$(pwd)"

_SRC_PATH="${1}"
_OUTPUT_PATH="${2}"

_PROTO_NAMES=$(find "${_SRC_PATH}" -type f -name '*.proto' -printf "%f\n" | sed -e 's@\.proto@@g')
for _PROTO_NAME in ${_PROTO_NAMES[@]}
do
  echo ${_PROTO_NAME}
  mkdir -p "${_OUTPUT_PATH}/${_PROTO_NAME}"
  protoc -I "${_SRC_PATH}" --go_out=plugins=grpc:"${_OUTPUT_PATH}/${_PROTO_NAME}" "${_SRC_PATH}/${_PROTO_NAME}.proto"
done
