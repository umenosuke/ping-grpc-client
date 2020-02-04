#!/bin/bash
#source /usr/local/nvm/nvm.sh

cd $(dirname $0)/../

PROTO_NAME="pingGrpc"
mkdir -p ./go/${PROTO_NAME}
mkdir -p ./ts/${PROTO_NAME}
protoc -I ./src/ --go_out=plugins=grpc:./go/${PROTO_NAME} ./src/${PROTO_NAME}.proto
protoc -I ./src/ --plugin="protoc-gen-ts=protoc-gen-ts" --tstypes_out=declare_namespace=false,original_names=true:./ts/${PROTO_NAME} ./src/${PROTO_NAME}.proto
