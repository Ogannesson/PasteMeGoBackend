# 使用 golang:alpine3.20 镜像
FROM golang:alpine3.20 AS builder

# 设置工作目录并复制文件
WORKDIR /go/src/github.com/PasteUs/PasteMeGoBackend
COPY ./ /go/src/github.com/PasteUs/PasteMeGoBackend

# 下载依赖并构建应用
RUN go mod download && \
    go build -o pastemed main.go

# 设置目标目录
RUN mkdir /pastemed && \
    cp config.example.json docker-entrypoint.sh /pastemed/ && \
    cp pastemed /pastemed/pastemed

# 创建最终的运行环境
FROM alpine:3.20

# 设置时区和环境变量
ENV TZ=Asia/Shanghai
COPY --from=builder /pastemed /usr/local/pastemed

# 安装所需工具并配置时区
RUN apk --no-cache add tzdata && \
    cp /usr/share/zoneinfo/Asia/Shanghai /etc/localtime && \
    echo "Asia/Shanghai" > /etc/timezone && \
    chmod +x /usr/local/pastemed/pastemed && \
    mkdir -p /data /etc/pastemed/

# 设置容器入口点
CMD ["/usr/bin/env", "sh", "/usr/local/pastemed/docker-entrypoint.sh"]

# 暴露服务端口
EXPOSE 8000
