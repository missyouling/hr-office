# 使用官方Node.js镜像
FROM node:20-alpine AS deps

# 设置工作目录
WORKDIR /app

# 复制package文件
COPY package*.json ./

# 安装依赖
RUN npm ci --only=production

# Builder阶段
FROM node:20-alpine AS builder

WORKDIR /app

# 复制package文件和安装所有依赖
COPY package*.json ./
RUN npm ci

# 复制源代码
COPY . .

# 使用构建参数设置环境变量
ARG NEXT_PUBLIC_API_BASE_URL=https://shebao.mozui.cn/api
ENV NEXT_PUBLIC_API_BASE_URL=$NEXT_PUBLIC_API_BASE_URL

# 构建应用
RUN npm run build

# 生产镜像
FROM node:20-alpine AS runner

WORKDIR /app

# 创建非root用户
RUN addgroup --system --gid 1001 nodejs
RUN adduser --system --uid 1001 nextjs

# 复制构建文件
COPY --from=builder /app/public ./public
COPY --from=builder /app/.next/standalone ./
COPY --from=builder /app/.next/static ./.next/static

# 设置权限
USER nextjs

# 暴露端口 (避免使用3000)
EXPOSE 8080

# 设置环境变量
ENV PORT=8080
ENV NODE_ENV=production

# 启动应用
CMD ["node", "server.js"]