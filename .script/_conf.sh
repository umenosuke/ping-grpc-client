#!/bin/bash

export _PRJ_NAME='ping-grpc-client'
export COMPOSE_PROJECT_NAME="${_PRJ_NAME}"

export _GIT_TAG="$(git describe --tags --abbrev=0)"
export _GIT_HASH="$(git rev-parse --short HEAD)"

export _USER="$(id -u):$(id -g)" 
