#!/bin/sh

set -o errexit
set -o nounset

IMAGE_NAME=yuhaohwang/bililive-go
VERSION=$(git describe --tags --always | sed 's/\([^-]*\).*/\1/')

IMAGE_TAG=$IMAGE_NAME:$VERSION

add_latest_tag() {
  if ! echo $VERSION | grep "rc" >/dev/null; then
    echo "-t $IMAGE_NAME:latest"
  fi
}

docker buildx build \
  --platform=linux/amd64 \
  -t $IMAGE_TAG $(add_latest_tag) \
  --build-arg "tag=${VERSION}" \
  --progress plain \
  --push \
  ./
