#!/bin/bash

cd $(dirname $0)/../

_WORK_DIR='build/release'

_TAG="$(git describe --tags --abbrev=0)"

arrays=(
"linux amd64 ping-grpc-client"
"windows amd64 ping-grpc-client.exe"
)

mkdir -p ${_WORK_DIR}

for array in "${arrays[@]}"
do
  array=(${array})
  _OUT_PATH="${_WORK_DIR}/${array[2]}"
  _ZIP_PATH="${_WORK_DIR}/ping-grpc-client-${_TAG}-${array[0]}-${array[1]}.zip"
  _TAR_PATH="${_WORK_DIR}/ping-grpc-client-${_TAG}-${array[0]}-${array[1]}.tar.gz"

  rm "${_OUT_PATH}"
  rm "${_ZIP_PATH}"
  rm "${_TAR_PATH}"

 .script/go_build.sh "${array[0]}" "${array[1]}" "${_OUT_PATH}"
  zip -9 "${_ZIP_PATH}" "${_OUT_PATH}"
  tar -zcvf "${_TAR_PATH}" "${_OUT_PATH}"
done
