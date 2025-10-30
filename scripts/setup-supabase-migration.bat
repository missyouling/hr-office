@echo off
REM Supabase 迁移快速设置脚本 (Windows)
REM 用法: scripts\setup-supabase-migration.bat

echo 🚀 开始 Supabase 迁移设置...
echo.

REM 检查前置条件
echo 📋 检查前置条件...

where node >nul 2>nul
if %ERRORLEVEL% NEQ 0 (
    echo ❌ 错误: 需要安装 Node.js
    exit /b 1
)

where go >nul 2>nul
if %ERRORLEVEL% NEQ 0 (
    echo ❌ 错误: 需要安装 Go
    exit /b 1
)

node --version
go version
echo.

REM 1. 安装前端依赖
echo 📦 安装前端依赖...
cd frontend
call npm install @supabase/supabase-js @supabase/ssr @supabase/auth-helpers-nextjs
if %ERRORLEVEL% NEQ 0 (
    echo ❌ 前端依赖安装失败
    cd ..
    exit /b 1
)
echo ✅ 前端依赖安装完成
echo.

REM 2. 安装后端依赖
echo 📦 安装后端依赖...
cd ..\backend
go get github.com/supabase-community/supabase-go
go get github.com/supabase-community/postgrest-go
go mod tidy
if %ERRORLEVEL% NEQ 0 (
    echo ❌ 后端依赖安装失败
    cd ..
    exit /b 1
)
echo ✅ 后端依赖安装完成
echo.

REM 3. 创建环境变量文件
echo 📝 创建环境变量文件...
cd ..

if not exist "frontend\.env.local" (
    copy "frontend\.env.example" "frontend\.env.local" >nul
    echo ✅ 已创建 frontend\.env.local (请填写实际值)
) else (
    echo ⚠️  frontend\.env.local 已存在，跳过
)

if not exist "backend\.env" (
    copy "backend\.env.example" "backend\.env" >nul
    echo ✅ 已创建 backend\.env (请填写实际值)
) else (
    echo ⚠️  backend\.env 已存在，跳过
)

echo.
echo ✨ 设置完成！
echo.
echo 下一步:
echo 1. 在 Supabase Dashboard 创建项目
echo 2. 获取 URL 和 Keys (Settings → API)
echo 3. 更新环境变量文件:
echo    - frontend\.env.local
echo    - backend\.env
echo 4. 在 Supabase SQL Editor 执行:
echo    supabase\migrations\001_initial_schema.sql
echo 5. 阅读迁移文档: SUPABASE_MIGRATION_README.md
echo.
echo 🎉 准备就绪！
pause