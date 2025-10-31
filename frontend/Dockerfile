# 使用官方Node.js镜像
FROM node:20-alpine AS deps

# 设置工作目录
WORKDIR /app

# 复制package文件
COPY package*.json ./

# 安装依赖
RUN npm ci --only=production && \
    rm -rf ~/.npm

# Builder阶段
FROM node:20-alpine AS builder

WORKDIR /app

# 复制package文件和安装所有依赖
COPY package*.json ./
RUN npm ci && rm -rf ~/.npm

# 复制源代码
COPY . .

# 使用构建参数设置环境变量
ARG NEXT_PUBLIC_API_BASE_URL_DOMAIN=http://localhost:8080/api
ARG NEXT_PUBLIC_API_BASE_URL_IP=http://localhost:8080/api
ARG NEXT_PUBLIC_API_BASE_URL=http://localhost:8080/api
ARG NEXT_PUBLIC_SUPABASE_URL
ARG NEXT_PUBLIC_SUPABASE_ANON_KEY

ENV NEXT_PUBLIC_API_BASE_URL_DOMAIN=$NEXT_PUBLIC_API_BASE_URL_DOMAIN
ENV NEXT_PUBLIC_API_BASE_URL_IP=$NEXT_PUBLIC_API_BASE_URL_IP
ENV NEXT_PUBLIC_API_BASE_URL=$NEXT_PUBLIC_API_BASE_URL
ENV NEXT_PUBLIC_SUPABASE_URL=$NEXT_PUBLIC_SUPABASE_URL
ENV NEXT_PUBLIC_SUPABASE_ANON_KEY=$NEXT_PUBLIC_SUPABASE_ANON_KEY
ENV NODE_ENV=production

# 构建应用
RUN npm run build

# 生产镜像
FROM node:20-alpine AS runner

WORKDIR /app

# 安装必要的包
RUN apk add --no-cache wget && \
    rm -rf /var/cache/apk/*

# 创建非root用户
RUN addgroup --system --gid 1001 nodejs && \
    adduser --system --uid 1001 nextjs

# 复制构建文件
COPY --from=builder /app/public ./public
COPY --from=builder /app/.next/standalone ./
COPY --from=builder /app/.next/static ./.next/static

# 设置权限
RUN chown -R nextjs:nodejs /app
USER nextjs

# 暴露端口
EXPOSE 8080

# 设置环境变量
ENV PORT=8080
ENV NODE_ENV=production
ENV HOSTNAME="0.0.0.0"

# 健康检查
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080 || exit 1

# 启动应用
CMD ["node", "server.js"]
