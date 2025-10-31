#!/bin/bash

# 人事行政管理系统 (hr-office) Docker 部署脚本

echo "🚀 开始部署人事行政管理系统 (hr-office)..."

# 检查Docker是否安装
if ! command -v docker &> /dev/null; then
    echo "❌ Docker 未安装，请先安装 Docker"
    exit 1
fi

# 检查Docker Compose是否安装
if ! command -v docker &> /dev/null || ! docker compose version &> /dev/null; then
    echo "❌ Docker 或 Docker Compose 未安装，请先安装 Docker"
    exit 1
fi

# 停止现有容器
echo "🛑 停止现有容器..."
docker compose down

# 清理旧的镜像 (可选)
echo "🧹 清理旧镜像..."
docker system prune -f

# 构建并启动服务
echo "🔨 构建镜像并启动服务..."
docker compose up --build -d

# 等待服务启动
echo "⏳ 等待服务启动..."
sleep 10

# 检查服务状态
echo "📊 检查服务状态..."
docker compose ps

# 显示日志
echo "📝 显示最近的日志..."
docker compose logs --tail=20

echo ""
echo "✅ 部署完成！"
echo "🌐 前端访问地址: http://localhost:8080"
echo "🔧 后端API地址: http://localhost:8081/api"
echo ""
echo "📱 常用命令:"
echo "  查看日志: docker compose logs -f"
echo "  重启服务: docker compose restart"
echo "  停止服务: docker compose down"
echo "  查看状态: docker compose ps"
