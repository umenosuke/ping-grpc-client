#!/bin/bash -x

cd "$(dirname $(readlink -f $0))/../"
_BASE_DIR="$(pwd)"

source ./.script/_conf.sh

_WORK_DIR='build/release'

arrays=(
"linux amd64 ${_PRJ_NAME}"
"windows amd64 ${_PRJ_NAME}.exe"
)

mkdir -p ${_WORK_DIR}

for array in "${arrays[@]}"
do
  cd "${_BASE_DIR}"

  array=(${array})
  _OUT_PATH="${array[2]}"
  _ZIP_PATH="${_PRJ_NAME}-${_GIT_TAG}-${array[0]}-${array[1]}.zip"
  _TAR_PATH="${_PRJ_NAME}-${_GIT_TAG}-${array[0]}-${array[1]}.tar.gz"

  rm "${_WORK_DIR}/${_OUT_PATH}"
  rm "${_WORK_DIR}/${_ZIP_PATH}"
  rm "${_WORK_DIR}/${_TAR_PATH}"

  .script/go_build.sh "${array[0]}" "${array[1]}" './src' "${_WORK_DIR}/${_OUT_PATH}"

  cd "${_WORK_DIR}"

  zip -9 "${_ZIP_PATH}" "${_OUT_PATH}"
  tar -zcvf "${_TAR_PATH}" "${_OUT_PATH}"
done
