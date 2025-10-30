#!/bin/bash

# Supabase 环境变量快速配置脚本
# 用法: bash scripts/configure-supabase.sh

echo "🔧 Supabase 环境变量配置工具"
echo "================================"
echo ""

# 读取配置信息
read -p "请输入 Project URL: " SUPABASE_URL
read -p "请输入 anon public key: " ANON_KEY
read -p "请输入 service_role key: " SERVICE_KEY
read -p "请输入 JWT Secret: " JWT_SECRET

echo ""
echo "📝 正在配置环境变量..."

# 配置前端环境变量
cat > frontend/.env.local << EOF
# Supabase 配置
NEXT_PUBLIC_SUPABASE_URL=$SUPABASE_URL
NEXT_PUBLIC_SUPABASE_ANON_KEY=$ANON_KEY

# BFF (Go 后端) 配置
NEXT_PUBLIC_BFF_URL=http://localhost:8080/api
EOF

echo "✅ frontend/.env.local 已更新"

# 配置后端环境变量
cat > backend/.env << EOF
# Supabase 配置
SUPABASE_URL=$SUPABASE_URL
SUPABASE_SERVICE_KEY=$SERVICE_KEY
SUPABASE_JWT_SECRET=$JWT_SECRET

# BFF 服务器配置
BFF_PORT=8080
ALLOWED_ORIGINS=http://localhost:3000,http://localhost:10086

# 日志配置
LOG_LEVEL=info
APP_ENV=development
EOF

echo "✅ backend/.env 已更新"

echo ""
echo "🎉 配置完成！"
echo ""
echo "下一步:"
echo "1. 在 Supabase SQL Editor 执行: supabase/migrations/001_initial_schema.sql"
echo "2. 启动开发服务器:"
echo "   前端: cd frontend && npm run dev"
echo "   后端: cd backend && go run ."
echo ""
echo "⚠️  安全提示: 请勿将 .env 文件提交到 Git！"