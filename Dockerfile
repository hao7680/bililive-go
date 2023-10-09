# 使用轻量级的alpine作为基础镜像
FROM alpine

# 定义构建参数 "tag"
ARG tag

# 设置环境变量，定义输出目录、配置目录和端口号
ENV OUTPUT_DIR="/srv/bililive" \
    CONF_DIR="/etc/bililive-go" \
    PORT=8080

# 创建所需的目录并更新alpine，然后安装ffmpeg, libc6-compat, curl 和 tzdata包
# 将时区设置为Asia/Shanghai
RUN mkdir -p $OUTPUT_DIR && \
    mkdir -p $CONF_DIR && \
    apk update && \
    apk --no-cache add ffmpeg libc6-compat curl tzdata && \
    cp -r -f /usr/share/zoneinfo/Asia/Shanghai /etc/localtime

# 下载并安装bililive-go，基于架构自动选择正确的二进制文件版本
# 首先，根据架构确定要下载的bililive-go的版本
# 然后，下载并解压相应版本的bililive-go，移动到/usr/bin目录并为其设置执行权限
# 最后，确保下载的bililive-go的版本与指定的tag匹配
RUN sh -c 'case $(arch) in aarch64) go_arch=arm64 ;; arm*) go_arch=arm ;; i386|i686) go_arch=386 ;; x86_64) go_arch=amd64;; esac && \
    cd /tmp && \
    curl -sSLO https://github.com/yuhaohwang/bililive-go/releases/download/${tag}/bililive-linux-${go_arch}.tar.gz && \
    tar zxvf bililive-linux-${go_arch}.tar.gz bililive-linux-${go_arch} && \
    chmod +x bililive-linux-${go_arch} && \
    mv ./bililive-linux-${go_arch} /usr/bin/bililive-go && \
    rm ./bililive-linux-${go_arch}.tar.gz' && \
    sh -c 'if [ ${tag} != $(/usr/bin/bililive-go --version | tr -d '\n') ]; then exit 1; fi'

# 复制配置文件到容器内的配置目录
COPY config.docker.yml $CONF_DIR/config.yml

# 指定存储音频的目录为一个卷，以便外部可以挂载
VOLUME $OUTPUT_DIR

# 暴露定义的端口号
EXPOSE $PORT

# 定义容器启动时运行的默认命令
ENTRYPOINT ["/usr/bin/bililive-go"]
CMD ["-c", "/etc/bililive-go/config.yml"]
