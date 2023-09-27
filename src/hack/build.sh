#!/bin/sh

# 设置错误处理，如果任何命令失败，脚本将退出
set -o errexit
set -o nounset

# 输出目录和常量路径
readonly OUTPUT_PATH=bin
readonly CONSTS_PATH="github.com/yuhaohwang/bililive-go/src/consts"

# 构建函数，接受目标、二进制名称和链接标志参数
_build() {
  target=$1
  bin_name=$2
  ld_flags=$3

  # 使用Go构建命令进行编译
  go build \
    -tags ${TAGS:-"release"} \
    -gcflags="${GCFLAGS:-""}" \
    -o ${OUTPUT_PATH}/${bin_name} \
    -ldflags="${ld_flags}" \
    ./src/cmd/${target}/
}

# 主构建函数，接受目标参数
build() {
  target=$1

  # 如果目标是'bililive'，则获取构建信息，并设置链接标志
  if [ ${target} = 'bililive' ]; then
    now=$(date '+%Y-%m-%d_%H:%M:%S')
    rev=$(echo "${rev:-$(git rev-parse HEAD)}")
    ver=$(git describe --tags --abbrev=0)

    debug_build_flags=""
    if [ ${TAGS} = 'release' ]; then
      debug_build_flags=" -s -w "
    fi
    ld_flags="${debug_build_flags} -X ${CONSTS_PATH}.BuildTime=${now} -X ${CONSTS_PATH}.AppVersion=${ver} -X ${CONSTS_PATH}.GitHash=${rev}"
  fi

  # 根据操作系统和架构设置二进制文件名称
  if [ $(go env GOOS) = "windows" ]; then
    ext=".exe"
  fi

  if [ $(go env GOARCH) = "mips" ]; then
    bin_name="${target}-$(go env GOOS)-$(go env GOARCH)-softfloat${ext:-}"

    export GOMIPS=softfloat
    _build "${target}" "${bin_name}" "${ld_flags:-}"
    unset GOMIPS
  fi

  bin_name="${target}-$(go env GOOS)-$(go env GOARCH)${ext:-}"

  _build "${target}" "${bin_name}" "${ld_flags:-}"

  # 如果启用了UPX，则对二进制文件进行压缩
  if [ ${UPX_ENABLE:-"0"} = "1" ]; then
    case "${bin_name}" in
    *-aix-* | *bsd-* | *-mips64* | *-riscv64 | *-s390x | *-plan9-* | *-windows-arm*) ;;
    *)
      upx --no-progress ${OUTPUT_PATH}/"${bin_name}"
      ;;
    esac
  fi
}

# 主函数，检查目标是否存在并调用构建函数
main() {
  if [ ! -d src/cmd/$1 ]; then
    echo '目标在 src/cmd/ 中不存在'
    exit 1
  fi
  build $1
}

# 调用主函数，并传递命令行参数
main $@
