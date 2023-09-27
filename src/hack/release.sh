#!/bin/sh

# 设置错误处理，如果任何命令失败，脚本将退出
set -o errexit
set -o nounset

# 只读变量，指定二进制文件输出路径
readonly BIN_PATH=bin

# 定义打包函数，根据传入的文件名和打包类型进行打包
package() {
  last_dir=$(pwd)
  cd $BIN_PATH
  file=$1
  type=$2
  case $type in
  zip)
    res=${file%.exe}.zip
    zip $res ${file} -j ../config.yml >/dev/null 2>&1
    ;;
  tar)
    res=${file}.tar.gz
    tar zcvf $res ${file} -C ../ config.yml >/dev/null 2>&1
    ;;
  7z)
    res=${file}.7z
    7z a $res ${file} ../config.yml >/dev/null 2>&1
    ;;
  *) ;;

  esac
  cd "$last_dir"
  echo $BIN_PATH/$res
}

# 遍历Go支持的所有平台和架构
for dist in $(go tool dist list); do
  case $dist in
  linux/loong64 | android/* | ios/* | js/wasm )
    continue  # 跳过不支持的平台和架构
    ;;
  *) ;;

  esac
  platform=$(echo ${dist} | cut -d'/' -f1)
  arch=$(echo ${dist} | cut -d'/' -f2)
  make PLATFORM=${platform} ARCH=${arch} bililive  # 使用Makefile构建二进制文件
done

# 打包生成的二进制文件
for file in $(ls $BIN_PATH); do
  case $file in
  *.tar.gz | *.zip | *.7z | *.yml | *.yaml)
    continue  # 跳过已经是压缩文件或配置文件的文件
    ;;
  *windows*)
    package_type=zip  # 对于Windows平台，使用ZIP打包
    ;;
  *)
    package_type=tar  # 对于其他平台，使用TAR打包
    ;;
  esac
  res=$(package $file $package_type)  # 调用打包函数
  rm -f $BIN_PATH/$file  # 删除原始的二进制文件
done
