#! /bin/bash

export VERSION=enterprise-2108
# export GOARCH=arm
export BUILD_IMAGE_BASE_NAME=image.goodrain.com
./release.sh all push
