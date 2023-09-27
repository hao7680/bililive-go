#!/bin/sh

# 设置错误处理，如果任何命令失败，脚本将退出
set -o errexit
set -o nounset

# 定义Docker镜像名称和版本
IMAGE_NAME=yuhaohwang/bililive-go
VERSION=$(git describe --tags --abbrev=0)

# 构建Docker镜像的标签，包括版本标签和latest标签（如果不是rc版本）
IMAGE_TAG=$IMAGE_NAME:$VERSION

# 添加latest标签的函数，仅当版本不包含"rc"时才添加latest标签
add_latest_tag() {
  if ! echo $VERSION | grep "rc" >/dev/null; then
    echo "-t $IMAGE_NAME:latest"
  fi
}

# 使用Docker Buildx构建镜像
docker buildx build \
  --platform=linux/amd64 \  # 指定目标平台
  -t $IMAGE_TAG $(add_latest_tag) \  # 设置镜像标签
  --build-arg "tag=${VERSION}" \  # 传递版本参数给构建
  --progress plain \  # 设置构建进度输出为普通模式
  --push \  # 推送镜像到Docker仓库
  ./  # 构建当前目录的Docker镜像
