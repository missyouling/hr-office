# 数据库迁移指南：从 SQLite 升级到 PostgreSQL

本文档提供了将社保整合系统从 SQLite 数据库迁移到 PostgreSQL 的详细步骤。

## 概述

PostgreSQL 相比 SQLite 提供了以下优势：
- 更好的并发性能和多用户支持
- 更强的数据完整性约束
- 更丰富的数据类型和索引
- 更好的备份和恢复选项
- 更适合生产环境的可扩展性

## 前置要求

### 1. 系统要求确认

**当前系统要求**: Go 1.24+ 已支持完整的 PostgreSQL 功能。

```bash
# 检查当前 Go 版本
go version
# 应该显示 go version go1.24+ linux/amd64

# 如需升级到 Go 1.24+
wget https://golang.org/dl/go1.24.linux-amd64.tar.gz
sudo rm -rf /usr/local/go
sudo tar -C /usr/local -xzf go1.24.linux-amd64.tar.gz

# 验证升级
go version
```

### 2. PostgreSQL 驱动已启用

当前系统已完全支持 PostgreSQL，在 `backend/main.go` 中已包含：

```go
// PostgreSQL 驱动已启用
"gorm.io/driver/postgres"

// PostgreSQL 连接功能已完全可用
```

## 迁移步骤

### 步骤 1: 备份现有 SQLite 数据

```bash
# 停止应用程序
pkill siapp

# 备份数据库文件
cp ./data/siapp.db ./data/siapp.db.backup.$(date +%Y%m%d_%H%M%S)

# 导出数据为 SQL 格式 (可选)
sqlite3 ./data/siapp.db .dump > siapp_backup.sql
```

### 步骤 2: 设置 PostgreSQL

#### 选项 A: 使用 Docker Compose (推荐)

```bash
# 启动 PostgreSQL 容器
docker-compose -f docker-compose.postgres.yml up -d postgres

# 等待数据库启动
docker-compose -f docker-compose.postgres.yml logs -f postgres
```

#### 选项 B: 本地安装 PostgreSQL

```bash
# Ubuntu/Debian
sudo apt update
sudo apt install postgresql postgresql-contrib

# 启动服务
sudo systemctl start postgresql
sudo systemctl enable postgresql

# 创建数据库和用户
sudo -u postgres psql
CREATE DATABASE siapp;
CREATE USER siapp WITH PASSWORD 'your-password';
GRANT ALL PRIVILEGES ON DATABASE siapp TO siapp;
\\q
```

### 步骤 3: 配置环境变量

复制并修改 PostgreSQL 配置：

```bash
cp .env.postgres.example .env
```

编辑 `.env` 文件：

```bash
# 数据库配置
SIAPP_DATABASE_TYPE=postgres
SIAPP_DB_HOST=localhost
SIAPP_DB_PORT=5432
SIAPP_DB_USER=siapp
SIAPP_DB_PASSWORD=your-secure-password
SIAPP_DB_NAME=siapp
SIAPP_DB_SSLMODE=require

# 其他配置保持不变...
```

### 步骤 4: 初始化 PostgreSQL 数据库

```bash
# 编译应用程序
cd backend
go build .

# 启动应用程序 - GORM 将自动创建表结构
./siapp
```

应用程序启动时，GORM 会自动执行 AutoMigrate 创建所有必要的表，包括：
- 用户认证相关表（users, password_reset_tokens, email_verification_tokens）
- 业务数据表（periods, source_files, raw_records, etc.）
- 审计日志表（audit_logs）

### 步骤 5: 数据迁移 (如果有现有数据)

如果你有现有的 SQLite 数据需要迁移，可以使用以下方法：

#### 方法 A: 使用导出/导入工具

```bash
# 1. 导出 SQLite 数据
sqlite3 ./data/siapp.db <<EOF
.headers on
.mode csv
.output users.csv
SELECT * FROM users;
.output periods.csv
SELECT * FROM periods;
.output audit_logs.csv
SELECT * FROM audit_logs;
-- 为其他表重复此过程
EOF

# 2. 使用 PostgreSQL COPY 命令导入
psql -h localhost -U siapp -d siapp <<EOF
\\COPY users FROM 'users.csv' DELIMITER ',' CSV HEADER;
\\COPY periods FROM 'periods.csv' DELIMITER ',' CSV HEADER;
\\COPY audit_logs FROM 'audit_logs.csv' DELIMITER ',' CSV HEADER;
-- 为其他表重复此过程
EOF
```

#### 方法 B: 使用数据迁移脚本

创建自定义迁移脚本来转换和导入数据：

```go
// migrate.go - 示例迁移脚本
package main

import (
    "log"
    "gorm.io/driver/sqlite"
    "gorm.io/driver/postgres"
    "gorm.io/gorm"
)

func main() {
    // 连接 SQLite
    sqliteDB, err := gorm.Open(sqlite.Open("./data/siapp.db"), &gorm.Config{})
    if err != nil {
        log.Fatal("Failed to connect to SQLite:", err)
    }

    // 连接 PostgreSQL
    dsn := "host=localhost user=siapp password=your-password dbname=siapp port=5432 sslmode=require"
    pgDB, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
    if err != nil {
        log.Fatal("Failed to connect to PostgreSQL:", err)
    }

    // 迁移数据...
    // 实现具体的数据迁移逻辑
}
```

### 步骤 6: 验证迁移

```bash
# 启动应用程序
./siapp

# 检查健康状态
curl http://localhost:8080/health

# 检查数据库状态
curl http://localhost:8080/api/monitoring/database

# 检查系统信息
curl http://localhost:8080/api/monitoring/info
```

### 步骤 7: 性能优化

迁移完成后，执行 PostgreSQL 性能索引脚本：

```sql
-- 连接到数据库
psql -h localhost -U siapp -d siapp

-- 执行性能优化函数
SELECT create_performance_indexes();

-- 分析表以优化查询计划
ANALYZE;
```

## 回滚程序

如果迁移过程中出现问题，可以回滚到 SQLite：

```bash
# 1. 停止应用程序
pkill siapp

# 2. 恢复环境配置
cp .env.example .env
# 编辑 .env 设置 SIAPP_DATABASE_TYPE=sqlite

# 3. 恢复数据库文件
cp ./data/siapp.db.backup.* ./data/siapp.db

# 4. 重新编译并启动
go build .
./siapp
```

系统会自动根据 `SIAPP_DATABASE_TYPE` 环境变量选择数据库类型，无需修改代码。

## 生产环境迁移注意事项

1. **停机时间规划**: 根据数据量估算迁移时间
2. **数据一致性**: 在迁移期间确保没有写入操作
3. **备份策略**: 迁移前后都要进行完整备份
4. **监控**: 迁移后密切监控系统性能和错误日志
5. **SSL 配置**: 生产环境必须使用 SSL 连接 (`SIAPP_DB_SSLMODE=require`)

## 性能对比

### SQLite vs PostgreSQL

| 功能 | SQLite | PostgreSQL |
|------|--------|------------|
| 并发写入 | 限制 | 优秀 |
| 多用户支持 | 基础 | 完整 |
| 数据完整性 | 基础 | 高级 |
| 备份恢复 | 文件复制 | 专业工具 |
| 监控 | 有限 | 丰富 |
| 扩展性 | 有限 | 优秀 |

## 故障排除

### 常见问题

1. **连接被拒绝**
   ```
   错误: connection refused
   解决: 检查 PostgreSQL 服务状态和端口配置
   ```

2. **认证失败**
   ```
   错误: password authentication failed
   解决: 验证用户名、密码和数据库名称
   ```

3. **SSL 连接错误**
   ```
   错误: SSL connection required
   解决: 设置 SIAPP_DB_SSLMODE=disable (开发) 或配置正确的 SSL
   ```

4. **数据库类型错误**
   ```
   错误: unsupported database type
   解决: 确保 SIAPP_DATABASE_TYPE 设置为 sqlite 或 postgres
   ```

## 监控和维护

迁移到 PostgreSQL 后的日常维护：

```bash
# 检查数据库状态
curl http://localhost:8080/api/monitoring/database

# 查看连接池状态
curl http://localhost:8080/api/monitoring/metrics

# 执行维护任务
curl -X POST http://localhost:8080/api/monitoring/maintenance
```

## 总结

这个迁移指南提供了从 SQLite 到 PostgreSQL 的完整迁移路径。记住：

1. 迁移前务必备份数据
2. 在测试环境先验证迁移过程
3. 生产环境迁移要做好停机时间规划
4. 迁移后进行全面的功能测试

如有问题，请参考故障排除部分或查看应用程序日志。