# 使用官方Go镜像
FROM golang:1.25-alpine AS builder

# 安装构建依赖
RUN apk add --no-cache git ca-certificates tzdata gcc musl-dev

# 设置工作目录
WORKDIR /app

# 复制go mod文件
COPY go.mod go.sum ./

# 下载依赖
RUN go mod download

# 复制源代码
COPY . .

# 构建二进制文件
RUN CGO_ENABLED=1 GOOS=linux go build -o siapp .

# 最终镜像
FROM alpine:latest

# 安装运行时依赖
RUN apk --no-cache add ca-certificates tzdata sqlite wget && \
    mkdir /app && \
    addgroup -g 1000 siapp && \
    adduser -u 1000 -G siapp -s /bin/sh -D siapp

# 设置时区
ENV TZ=Asia/Shanghai

WORKDIR /app

# 从builder阶段复制二进制文件
COPY --from=builder /app/siapp .

# 创建数据目录
RUN mkdir -p /app/data && chown -R siapp:siapp /app

# 切换到非root用户
USER siapp

# 暴露端口
EXPOSE 8080

# 健康检查
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# 设置环境变量 (生产环境默认使用PostgreSQL)
ENV SIAPP_ADDR=0.0.0.0:8080
ENV SIAPP_DATABASE_TYPE=postgres
ENV SIAPP_DB_HOST=postgres
ENV SIAPP_DB_PORT=5432
ENV SIAPP_DB_USER=siapp
ENV SIAPP_DB_NAME=siapp
ENV SIAPP_DB_SSLMODE=require

# 运行
CMD ["./siapp"]