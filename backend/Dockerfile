# 使用官方Go镜像
FROM golang:1.18-alpine AS builder

# 更换为国内Alpine镜像源并安装构建依赖
RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories && \
    apk update && \
    apk add --no-cache gcc musl-dev sqlite-dev

# 设置工作目录
WORKDIR /app

# 复制go mod文件
COPY go.mod go.sum ./

# 下载依赖
RUN go mod download

# 复制源代码
COPY . .

# 构建二进制文件
RUN CGO_ENABLED=1 GOOS=linux go build -a -installsuffix cgo -o main .

# 最终镜像
FROM alpine:latest

# 安装SQLite3
RUN apk --no-cache add ca-certificates sqlite

WORKDIR /root/

# 从builder阶段复制二进制文件
COPY --from=builder /app/main .

# 创建数据目录
RUN mkdir -p ./data

# 暴露端口
EXPOSE 8080

# 设置环境变量
ENV SIAPP_ADDR=:8080
ENV SIAPP_DATABASE_PATH=./data/siapp.db

# 运行
CMD ["./main"]