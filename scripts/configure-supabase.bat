@echo off
REM Supabase 环境变量快速配置脚本 (Windows)
REM 用法: scripts\configure-supabase.bat

echo 🔧 Supabase 环境变量配置工具
echo ================================
echo.

REM 读取配置信息
set /p SUPABASE_URL="请输入 Project URL: "
set /p ANON_KEY="请输入 anon public key: "
set /p SERVICE_KEY="请输入 service_role key: "
set /p JWT_SECRET="请输入 JWT Secret: "

echo.
echo 📝 正在配置环境变量...

REM 配置前端环境变量
(
echo # Supabase 配置
echo NEXT_PUBLIC_SUPABASE_URL=%SUPABASE_URL%
echo NEXT_PUBLIC_SUPABASE_ANON_KEY=%ANON_KEY%
echo.
echo # BFF ^(Go 后端^) 配置
echo NEXT_PUBLIC_BFF_URL=http://localhost:8080/api
) > frontend\.env.local

echo ✅ frontend\.env.local 已更新

REM 配置后端环境变量
(
echo # Supabase 配置
echo SUPABASE_URL=%SUPABASE_URL%
echo SUPABASE_SERVICE_KEY=%SERVICE_KEY%
echo SUPABASE_JWT_SECRET=%JWT_SECRET%
echo.
echo # BFF 服务器配置
echo BFF_PORT=8080
echo ALLOWED_ORIGINS=http://localhost:3000,http://localhost:10086
echo.
echo # 日志配置
echo LOG_LEVEL=info
echo APP_ENV=development
) > backend\.env

echo ✅ backend\.env 已更新

echo.
echo 🎉 配置完成！
echo.
echo 下一步:
echo 1. 在 Supabase SQL Editor 执行: supabase\migrations\001_initial_schema.sql
echo 2. 启动开发服务器:
echo    前端: cd frontend ^&^& npm run dev
echo    后端: cd backend ^&^& go run .
echo.
echo ⚠️  安全提示: 请勿将 .env 文件提交到 Git！
pause