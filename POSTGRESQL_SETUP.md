# PostgreSQL 数据库配置指南

**状态**：✅ 后端已配置为仅支持 PostgreSQL  
**更新日期**：2026-04-18

---

## 📋 概述

系统已**禁用 SQLite**，现在**仅支持 PostgreSQL** 数据库。所有部署必须使用 PostgreSQL。

| 项 | 值 |
|----|---|
| 数据库类型 | PostgreSQL (pgx 驱动) |
| 默认端口 | 5432 |
| 默认用户 | siapp |
| 默认密码 | 需配置 |
| 默认数据库 | siapp |
| SSL 模式 | disable（本地开发）/ require（生产） |

---

## 🔧 配置环境变量

### 本地开发配置

编辑 `backend/.env`：

```dotenv
# 数据库配置
SIAPP_DATABASE_TYPE=postgres
SIAPP_DB_HOST=localhost
SIAPP_DB_PORT=5432
SIAPP_DB_USER=siapp
SIAPP_DB_PASSWORD=your_secure_password
SIAPP_DB_NAME=siapp
SIAPP_DB_SSLMODE=disable

# 服务配置
SIAPP_ADDR=:8080
JWT_SECRET_KEY=dev-secret-change-me
JWT_TOKEN_DURATION=12h
ALLOWED_ORIGINS=http://localhost:3000
APP_ENV=development
LOG_LEVEL=debug
```

### 生产配置（Supabase 示例）

```dotenv
# PostgreSQL 配置
SIAPP_DATABASE_TYPE=postgres
SIAPP_DB_HOST=aws-1-xxxxx.pooler.supabase.com
SIAPP_DB_PORT=5432
SIAPP_DB_USER=postgres.your-project-ref
SIAPP_DB_PASSWORD=your_secure_password
SIAPP_DB_NAME=postgres
SIAPP_DB_SSLMODE=require

# 服务配置
SIAPP_ADDR=:8080
JWT_SECRET_KEY=production-secret-key
JWT_TOKEN_DURATION=12h
ALLOWED_ORIGINS=https://your-domain.com
APP_ENV=production
LOG_LEVEL=info
```

---

## 📦 步骤 1：安装和启动 PostgreSQL

### macOS（使用 Homebrew）

```bash
# 安装 PostgreSQL
brew install postgresql@15

# 启动服务
brew services start postgresql@15

# 验证安装
psql --version
```

### Ubuntu/Debian

```bash
# 安装 PostgreSQL
sudo apt update
sudo apt install postgresql postgresql-contrib

# 启动服务
sudo systemctl start postgresql

# 验证状态
sudo systemctl status postgresql
```

### Docker（推荐）

```bash
# 启动 PostgreSQL 容器
docker run -d \
  --name postgresql-siapp \
  -e POSTGRES_USER=siapp \
  -e POSTGRES_PASSWORD=your_secure_password \
  -e POSTGRES_DB=siapp \
  -p 5432:5432 \
  -v postgresql_data:/var/lib/postgresql/data \
  postgres:15-alpine

# 验证容器运行
docker ps | grep postgresql-siapp

# 查看日志
docker logs postgresql-siapp
```

---

## 📊 步骤 2：创建数据库和用户

### 方式 A：使用 psql 命令行

```bash
# 连接到 PostgreSQL
psql -U postgres -h localhost

# 在 psql 中执行以下命令
CREATE USER siapp WITH ENCRYPTED PASSWORD 'your_secure_password';
CREATE DATABASE siapp OWNER siapp;

# 授予权限
GRANT ALL PRIVILEGES ON DATABASE siapp TO siapp;

# 连接到新数据库
\c siapp

# 授予 schema 权限
GRANT ALL PRIVILEGES ON SCHEMA public TO siapp;
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL PRIVILEGES ON TABLES TO siapp;

# 验证
\du          -- 列出用户
\l           -- 列出数据库
\q           -- 退出
```

### 方式 B：使用 SQL 脚本

创建文件 `scripts/init-postgresql.sql`：

```sql
-- 创建用户
CREATE USER siapp WITH ENCRYPTED PASSWORD 'your_secure_password';

-- 创建数据库
CREATE DATABASE siapp OWNER siapp;

-- 授予权限
GRANT ALL PRIVILEGES ON DATABASE siapp TO siapp;

-- 连接到数据库后执行
\c siapp

GRANT ALL PRIVILEGES ON SCHEMA public TO siapp;
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL PRIVILEGES ON TABLES TO siapp;
```

执行脚本：

```bash
psql -U postgres -h localhost -f scripts/init-postgresql.sql
```

### 方式 C：Docker 中的 PostgreSQL

```bash
# 进入 PostgreSQL 容器
docker exec -it postgresql-siapp psql -U postgres

# 然后执行上述 SQL 命令
```

---

## 🔗 步骤 3：验证连接

### 使用 psql

```bash
# 以 siapp 用户连接
psql -U siapp -h localhost -d siapp

# 如果提示输入密码，输入配置的密码
# 成功后会显示 siapp=>#
```

### 使用 Go 应用

后端启动时会自动测试连接：

```bash
cd backend
export SIAPP_DATABASE_TYPE=postgres
export SIAPP_DB_HOST=localhost
export SIAPP_DB_PORT=5432
export SIAPP_DB_USER=siapp
export SIAPP_DB_PASSWORD=your_secure_password
export SIAPP_DB_NAME=siapp

go run .
```

**预期输出**：
```
[INFO] Connecting to PostgreSQL database: host=localhost port=5432 dbname=siapp user=siapp sslmode=disable
[INFO] AutoMigrate completed for User, Period, SourceFile, ...
[INFO] HTTP server listening on :8080
```

### 使用 pg_isready（快速检查）

```bash
# 检查 PostgreSQL 是否准备好
pg_isready -h localhost -p 5432 -U siapp

# 输出应该是：
# localhost:5432 - accepting connections
```

---

## 🚀 步骤 4：启动后端服务

### 本地开发

```bash
cd backend

# 确保 .env 已配置
cat .env | grep SIAPP_DATABASE

# 启动后端
CGO_ENABLED=1 go run .

# 或编译后运行
go build -o siapp .
./siapp
```

### Docker Compose

编辑 `docker-compose.yml`：

```yaml
version: '3.8'

services:
  postgres:
    image: postgres:15-alpine
    environment:
      POSTGRES_USER: siapp
      POSTGRES_PASSWORD: your_secure_password
      POSTGRES_DB: siapp
    ports:
      - "5432:5432"
    volumes:
      - postgresql_data:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U siapp -d siapp"]
      interval: 10s
      timeout: 5s
      retries: 5

  backend:
    build:
      context: ./backend
      dockerfile: Dockerfile
    environment:
      SIAPP_DATABASE_TYPE: postgres
      SIAPP_DB_HOST: postgres
      SIAPP_DB_PORT: 5432
      SIAPP_DB_USER: siapp
      SIAPP_DB_PASSWORD: your_secure_password
      SIAPP_DB_NAME: siapp
      SIAPP_DB_SSLMODE: disable
      SIAPP_ADDR: :8080
      JWT_SECRET_KEY: dev-secret-change-me
    ports:
      - "8080:8080"
    depends_on:
      postgres:
        condition: service_healthy
    command: sh -c "sleep 5 && ./siapp"

volumes:
  postgresql_data:
```

启动：

```bash
docker-compose up -d

# 查看日志
docker-compose logs -f backend
```

---

## 🔍 常见问题排查

### 问题 1：连接被拒绝

**症状**：`connection refused` 或 `ECONNREFUSED`

**排查步骤**：

1. 检查 PostgreSQL 是否运行：
   ```bash
   pg_isready -h localhost -p 5432
   # 应输出：accepting connections
   ```

2. 检查防火墙：
   ```bash
   # 检查 5432 端口是否开放
   lsof -i :5432
   ```

3. 检查配置的主机和端口：
   ```bash
   echo "Host: $SIAPP_DB_HOST"
   echo "Port: $SIAPP_DB_PORT"
   ```

### 问题 2：认证失败

**症状**：`FATAL: password authentication failed` 或 `role "siapp" does not exist`

**排查步骤**：

1. 验证用户存在：
   ```bash
   psql -U postgres -h localhost -c "\du"
   # 应该看到 siapp 用户
   ```

2. 重置密码：
   ```bash
   psql -U postgres -h localhost -c "ALTER USER siapp WITH PASSWORD 'new_password';"
   ```

3. 检查 .env 中的密码是否正确：
   ```bash
   grep SIAPP_DB_PASSWORD backend/.env
   ```

### 问题 3：数据库不存在

**症状**：`database "siapp" does not exist`

**解决方案**：

```bash
# 以 postgres 用户创建数据库
psql -U postgres -h localhost -c "CREATE DATABASE siapp OWNER siapp;"

# 或使用脚本
psql -U postgres -h localhost -f scripts/init-postgresql.sql
```

### 问题 4：迁移失败

**症状**：`AutoMigrate failed` 或 SQL 错误

**排查步骤**：

1. 检查权限：
   ```bash
   psql -U postgres -h localhost -d siapp -c "GRANT ALL PRIVILEGES ON SCHEMA public TO siapp;"
   ```

2. 查看详细错误日志：
   ```bash
   # 后端启动时添加 LOG_LEVEL=debug
   LOG_LEVEL=debug go run .
   ```

3. 手动检查表：
   ```bash
   psql -U siapp -h localhost -d siapp -c "\dt"
   ```

### 问题 5：连接超时

**症状**：`dial tcp: i/o timeout` 或连接挂起

**原因**：
- 网络问题
- PostgreSQL 响应缓慢
- 连接池耗尽

**解决**：

```bash
# 检查 PostgreSQL 是否响应
pg_isready -h localhost -p 5432 -U siapp

# 增加连接超时（编辑 backend/.env）
# 后端代码中 DialFunc 设置了 30 秒超时

# 重启 PostgreSQL
sudo systemctl restart postgresql
```

---

## 📝 环境变量完整列表

| 变量 | 说明 | 默认值 | 示例 |
|------|------|--------|------|
| `SIAPP_DATABASE_TYPE` | 数据库类型 | - | `postgres` |
| `SIAPP_DB_HOST` | 主机地址 | `localhost` | `localhost` |
| `SIAPP_DB_PORT` | 端口号 | `5432` | `5432` |
| `SIAPP_DB_USER` | 数据库用户 | `siapp` | `siapp` |
| `SIAPP_DB_PASSWORD` | 数据库密码 | - | `secure_pass` |
| `SIAPP_DB_NAME` | 数据库名 | `siapp` | `siapp` |
| `SIAPP_DB_SSLMODE` | SSL 模式 | `require` | `disable`（本地） |
| `SIAPP_ADDR` | 服务监听地址 | `:8080` | `:8080` |
| `JWT_SECRET_KEY` | JWT 签名密钥 | - | `your-secret-key` |
| `JWT_TOKEN_DURATION` | Token 有效期 | - | `12h` |
| `ALLOWED_ORIGINS` | CORS 允许源 | - | `http://localhost:3000` |
| `LOG_LEVEL` | 日志级别 | `info` | `debug` |
| `APP_ENV` | 应用环境 | `development` | `production` |

---

## 🔐 生产环境检查清单

部署到生产环境前，请确认：

- [ ] PostgreSQL 已安装并运行（版本 12+）
- [ ] 数据库和用户已创建
- [ ] SSL 模式已设置为 `require`
- [ ] 强密码已设置（符合安全策略）
- [ ] 防火墙规则已配置（允许 5432 入站）
- [ ] 备份策略已实施
- [ ] 监控已配置（CPU、内存、磁盘、连接数）
- [ ] 日志收集已配置
- [ ] 事务日志（WAL）备份已启用
- [ ] 定期健康检查已实施

---

## 📚 参考资源

- [PostgreSQL 官方文档](https://www.postgresql.org/docs/)
- [pgx 驱动文档](https://pkg.go.dev/github.com/jackc/pgx)
- [GORM PostgreSQL 文档](https://gorm.io/docs/connecting_to_the_database.html#PostgreSQL)

---

## 🎯 总结

✅ **后端已配置为仅支持 PostgreSQL**  
✅ **SQLite 已禁用**  
✅ **支持本地和 Supabase 部署**  

现在可以按照上述步骤配置 PostgreSQL 并启动后端服务。
