# PostgreSQL 快速启动（5 分钟）

## 🚀 最快方式：使用 Docker

```bash
# 1. 启动 PostgreSQL 容器
docker run -d \
  --name postgresql-siapp \
  -e POSTGRES_USER=siapp \
  -e POSTGRES_PASSWORD=siapp_password \
  -e POSTGRES_DB=siapp \
  -p 5432:5432 \
  postgres:15-alpine

# 2. 等待容器启动（约 5 秒）
sleep 5

# 3. 验证连接
psql -U siapp -h localhost -d siapp -c "SELECT 1;"
# 输出应该是：?column?
#     1

# 4. 配置后端环境变量
export SIAPP_DATABASE_TYPE=postgres
export SIAPP_DB_HOST=localhost
export SIAPP_DB_PORT=5432
export SIAPP_DB_USER=siapp
export SIAPP_DB_PASSWORD=siapp_password
export SIAPP_DB_NAME=siapp
export SIAPP_DB_SSLMODE=disable

# 5. 启动后端
cd backend
CGO_ENABLED=1 go run .

# 预期输出：
# [INFO] Connecting to PostgreSQL database: host=localhost port=5432 dbname=siapp user=siapp sslmode=disable
# [INFO] HTTP server listening on :8080
```

---

## 🏠 本地 PostgreSQL（macOS）

```bash
# 1. 安装
brew install postgresql@15
brew services start postgresql@15

# 2. 创建用户和数据库
psql -U postgres -h localhost << EOF
CREATE USER siapp WITH ENCRYPTED PASSWORD 'siapp_password';
CREATE DATABASE siapp OWNER siapp;
GRANT ALL PRIVILEGES ON DATABASE siapp TO siapp;
EOF

# 3. 配置后端
export SIAPP_DATABASE_TYPE=postgres
export SIAPP_DB_HOST=localhost
export SIAPP_DB_PORT=5432
export SIAPP_DB_USER=siapp
export SIAPP_DB_PASSWORD=siapp_password
export SIAPP_DB_NAME=siapp
export SIAPP_DB_SSLMODE=disable

# 4. 启动后端
cd backend && CGO_ENABLED=1 go run .
```

---

## 🐧 本地 PostgreSQL（Ubuntu/Debian）

```bash
# 1. 安装
sudo apt update
sudo apt install postgresql postgresql-contrib

# 2. 创建用户和数据库
sudo -u postgres psql << EOF
CREATE USER siapp WITH ENCRYPTED PASSWORD 'siapp_password';
CREATE DATABASE siapp OWNER siapp;
GRANT ALL PRIVILEGES ON DATABASE siapp TO siapp;
EOF

# 3. 后续同 macOS
export SIAPP_DATABASE_TYPE=postgres
export SIAPP_DB_HOST=localhost
...
```

---

## ☁️ Supabase 云端部署

```bash
# 1. 在 Supabase 创建项目
# 访问 https://supabase.com 创建新项目

# 2. 获取连接信息
# Dashboard → Database → Connection → Pooler
# 复制连接字符串信息

# 3. 配置后端
export SIAPP_DATABASE_TYPE=postgres
export SIAPP_DB_HOST=aws-1-xxxxx.pooler.supabase.com
export SIAPP_DB_PORT=5432
export SIAPP_DB_USER=postgres.xxxx
export SIAPP_DB_PASSWORD=your_supabase_password
export SIAPP_DB_NAME=postgres
export SIAPP_DB_SSLMODE=require

# 4. 启动后端
cd backend && CGO_ENABLED=1 go run .
```

---

## ✅ 验证连接

```bash
# 方式 1：使用 psql
psql -U siapp -h localhost -d siapp -c "SELECT version();"

# 方式 2：使用 pg_isready
pg_isready -h localhost -p 5432 -U siapp

# 方式 3：查看后端日志（成功会显示）
# [INFO] Connecting to PostgreSQL database: host=localhost port=5432 ...
# [INFO] HTTP server listening on :8080
```

---

## 🔧 环境变量速查

```bash
# 最小配置（本地开发）
SIAPP_DATABASE_TYPE=postgres
SIAPP_DB_HOST=localhost
SIAPP_DB_PORT=5432
SIAPP_DB_USER=siapp
SIAPP_DB_PASSWORD=siapp_password
SIAPP_DB_NAME=siapp
SIAPP_DB_SSLMODE=disable
```

---

## ❌ SQLite 已禁用

```
❌ SQLite 不再支持
❌ SIAPP_DATABASE_PATH 被忽略
✅ 仅支持 PostgreSQL
✅ 必须设置 SIAPP_DATABASE_TYPE=postgres
```

---

## 🆘 快速故障排查

| 错误 | 原因 | 解决 |
|------|------|------|
| `connection refused` | PostgreSQL 未运行 | `pg_isready` 检查或启动服务 |
| `password authentication failed` | 密码错误 | 确认 `SIAPP_DB_PASSWORD` 正确 |
| `database "siapp" does not exist` | 数据库未创建 | 执行上述创建命令 |
| `Connecting to PostgreSQL database: host=localhost...` 后无响应 | 连接卡住 | 检查防火墙或 PostgreSQL 状态 |

---

✅ **现在可以启动后端了！**
