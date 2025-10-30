#!/bin/bash

# 生产环境安全配置检查脚本
# 用法: ./scripts/security-check.sh [development|production]

ENV=${1:-development}

echo "🔍 正在检查 $ENV 环境的安全配置..."
echo ""

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 检查函数
check_pass() {
    echo -e "${GREEN}✅ $1${NC}"
}

check_warn() {
    echo -e "${YELLOW}⚠️  $1${NC}"
}

check_fail() {
    echo -e "${RED}❌ $1${NC}"
}

check_info() {
    echo -e "${BLUE}ℹ️  $1${NC}"
}

# 检查环境变量文件
ENV_FILE=".env"
if [ "$ENV" = "production" ]; then
    ENV_FILE=".env.production"
fi

if [ ! -f "$ENV_FILE" ]; then
    check_fail "环境变量文件 $ENV_FILE 不存在"
    exit 1
fi

check_pass "找到环境变量文件: $ENV_FILE"

# 加载环境变量
set -a
source "$ENV_FILE"
set +a

echo ""
echo "🔐 JWT 安全检查:"

# 检查JWT密钥
if [ -z "$JWT_SECRET_KEY" ]; then
    check_fail "JWT_SECRET_KEY 未设置"
elif [ ${#JWT_SECRET_KEY} -lt 32 ]; then
    check_fail "JWT_SECRET_KEY 太短 (少于32字符)"
elif [ "$JWT_SECRET_KEY" = "your-secret-key-change-this-in-production" ] || [ "$JWT_SECRET_KEY" = "dev-secret-key-for-local-development-only" ]; then
    if [ "$ENV" = "production" ]; then
        check_fail "生产环境仍在使用默认JWT密钥"
    else
        check_warn "开发环境使用默认JWT密钥"
    fi
else
    check_pass "JWT密钥配置正确 (${#JWT_SECRET_KEY} 字符)"
fi

# 检查JWT过期时间
if [ -z "$JWT_TOKEN_DURATION" ]; then
    check_warn "JWT_TOKEN_DURATION 未设置，使用默认值"
else
    check_pass "JWT过期时间: $JWT_TOKEN_DURATION"
fi

echo ""
echo "🌐 CORS 安全检查:"

# 检查CORS配置
if [ -z "$ALLOWED_ORIGINS" ]; then
    check_fail "ALLOWED_ORIGINS 未设置"
elif echo "$ALLOWED_ORIGINS" | grep -q "localhost\|127.0.0.1"; then
    if [ "$ENV" = "production" ]; then
        check_fail "生产环境允许localhost访问"
    else
        check_pass "开发环境允许localhost访问"
    fi
else
    check_pass "CORS域名配置: $ALLOWED_ORIGINS"
fi

echo ""
echo "🗄️ 数据库安全检查:"

# 检查数据库路径
if [ -n "$SIAPP_DATABASE_PATH" ]; then
    check_pass "数据库路径: $SIAPP_DATABASE_PATH"

    # 检查数据库文件权限
    if [ -f "$SIAPP_DATABASE_PATH" ]; then
        DB_PERMS=$(stat -c "%a" "$SIAPP_DATABASE_PATH" 2>/dev/null || echo "unknown")
        if [ "$DB_PERMS" = "600" ] || [ "$DB_PERMS" = "644" ]; then
            check_pass "数据库文件权限: $DB_PERMS"
        else
            check_warn "数据库文件权限: $DB_PERMS (建议600或644)"
        fi
    else
        check_info "数据库文件不存在，将在启动时创建"
    fi
fi

echo ""
echo "📝 日志安全检查:"

# 检查日志级别
if [ -n "$LOG_LEVEL" ]; then
    if [ "$ENV" = "production" ] && [ "$LOG_LEVEL" = "debug" ]; then
        check_warn "生产环境使用debug日志级别可能泄露敏感信息"
    else
        check_pass "日志级别: $LOG_LEVEL"
    fi
fi

echo ""
echo "🐳 Docker 安全检查:"

# 检查Docker配置
COMPOSE_FILE="docker-compose.yml"
if [ "$ENV" = "production" ]; then
    COMPOSE_FILE="docker-compose.production.yml"
fi

if [ -f "$COMPOSE_FILE" ]; then
    check_pass "找到Docker配置文件: $COMPOSE_FILE"

    # 检查非root用户配置 (仅生产环境)
    if [ "$ENV" = "production" ]; then
        if grep -q "user:" "$COMPOSE_FILE"; then
            check_pass "配置了非root用户运行"
        else
            check_warn "建议配置非root用户运行容器"
        fi

        # 检查资源限制
        if grep -q "resources:" "$COMPOSE_FILE"; then
            check_pass "配置了资源限制"
        else
            check_warn "建议配置容器资源限制"
        fi
    fi
fi

echo ""
echo "📁 文件安全检查:"

# 检查敏感文件是否被git忽略
if [ -f ".gitignore" ]; then
    if grep -q "\.env" ".gitignore"; then
        check_pass "环境变量文件已添加到.gitignore"
    else
        check_fail "环境变量文件未添加到.gitignore"
    fi

    if grep -q "\.log" ".gitignore"; then
        check_pass "日志文件已添加到.gitignore"
    else
        check_warn "建议将日志文件添加到.gitignore"
    fi
fi

echo ""
echo "🔒 生产环境额外检查:"

if [ "$ENV" = "production" ]; then
    # 检查HTTPS配置
    if [ "$HTTPS_ONLY" = "true" ]; then
        check_pass "启用了HTTPS强制"
    else
        check_warn "建议启用HTTPS强制"
    fi

    # 检查速率限制
    if [ "$RATE_LIMIT_ENABLED" = "true" ]; then
        check_pass "启用了API速率限制"
    else
        check_warn "建议启用API速率限制"
    fi

    # 检查监控配置
    if [ "$ENABLE_METRICS" = "true" ]; then
        check_pass "启用了系统监控"
    else
        check_warn "建议启用系统监控"
    fi
fi

echo ""
echo "📋 安全配置总结:"
echo "   环境: $ENV"
echo "   配置文件: $ENV_FILE"
echo "   JWT密钥长度: ${#JWT_SECRET_KEY} 字符"
echo "   CORS域名数量: $(echo "$ALLOWED_ORIGINS" | tr ',' '\n' | wc -l)"

if [ "$ENV" = "production" ]; then
    echo ""
    echo "🚨 生产环境部署前最终检查:"
    echo "   1. 确认JWT密钥已更换为强随机密钥"
    echo "   2. 确认CORS仅允许生产域名"
    echo "   3. 确认已配置HTTPS"
    echo "   4. 确认数据库已备份"
    echo "   5. 确认监控和告警已配置"
fi

echo ""
check_info "安全检查完成！"