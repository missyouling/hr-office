# CLAUDE.md

此文件为 Claude Code (claude.ai/code) 在本代码仓库中工作时提供指导。

## 项目概述

这是一个人事行政管理系统（hr-office），采用 Go 后端和 Next.js 前端架构。系统负责处理社保账期管理、多险种文件上传、花名册管理、数据处理和 Excel 报表生成。

## 系统架构

- **后端**: 使用 Chi 路由器的 Go 1.24+ 应用，GORM 配合 SQLite/PostgreSQL，excelize 处理 Excel
- **前端**: Next.js 15 配合 React 19、TypeScript、Tailwind CSS、shadcn/ui 组件
- **数据库**: SQLite（默认）+ PostgreSQL（生产推荐）配合 GORM 自动迁移
- **认证**: JWT 身份验证，支持用户注册、登录、邮箱验证、密码重置
- **监控**: 系统健康检查、指标监控、审计日志、数据库状态监控
- **API**: 支持 CORS 的 RESTful API，完整的认证和授权机制

### 核心数据模型

#### 业务模型 (backend/internal/models/models.go)
- `Period`: 社保账期（如 "2025-08"）
- `SourceFile`: 上传的险种文件
- `RawRecord`: 从上传文件中解析的个人保险记录
- `PeriodSummary`: 按险种/缴费部分聚合的统计数据
- `PersonalCharge`/`UnitCharge`: 个人/单位最终计算的扣款明细
- `RosterEntry`: 员工花名册数据，用于补充部门信息

#### 认证模型 (backend/internal/models/models.go)
- `User`: 用户信息，包含用户名、邮箱、密码等
- `PasswordResetToken`: 密码重置令牌
- `EmailVerificationToken`: 邮箱验证令牌

#### 审计模型 (backend/internal/models/audit_log.go)
- `AuditLog`: 审计日志，记录用户操作和系统事件
- `LogDetails`: 审计日志详细信息，支持结构化数据

### 险种和缴费部分

- **险种**: `pension`（养老）、`medical`（基本医疗）、`serious_illness`（大额/生育）、`unemployment`（失业）、`injury`（工伤）
- **缴费部分**: `personal`（个人缴费）、`unit`（单位缴费）

## 开发命令

### 前端 (frontend/)
```bash
npm install                    # 安装依赖
npm run dev                    # 启动开发服务器 (http://localhost:3000)
npm run build                  # 生产环境构建
npm run lint                   # 运行 ESLint 检查
npm run start                  # 启动生产构建
```

### 后端 (backend/)
```bash
go run .                       # 启动开发服务器 (http://localhost:8080)
go build .                     # 编译二进制文件
go mod tidy                    # 整理依赖

# 数据库相关
# SQLite 模式 (默认)
go run .

# PostgreSQL 模式 (需要先配置环境变量)
SIAPP_DATABASE_TYPE=postgres SIAPP_DB_PASSWORD=your-password go run .
```

### 环境配置

- 前端：复制 `.env.local.example` 为 `.env.local` 并配置 `NEXT_PUBLIC_API_BASE_URL`
- 后端：设置环境变量：

#### 基础配置
  - `SIAPP_ADDR`: HTTP 监听地址（默认 `:8080`）
  - `SIAPP_DATABASE_TYPE`: 数据库类型 (`sqlite` 或 `postgres`，默认 `sqlite`)

#### SQLite 配置
  - `SIAPP_DATABASE_PATH`: SQLite 文件路径（默认 `./data/siapp.db`）

#### PostgreSQL 配置
  - `SIAPP_DB_HOST`: PostgreSQL 主机地址（默认 `localhost`）
  - `SIAPP_DB_PORT`: PostgreSQL 端口（默认 `5432`）
  - `SIAPP_DB_USER`: 数据库用户名（默认 `siapp`）
  - `SIAPP_DB_PASSWORD`: 数据库密码（必需）
  - `SIAPP_DB_NAME`: 数据库名称（默认 `siapp`）
  - `SIAPP_DB_SSLMODE`: SSL 模式（默认 `require`）

#### 安全配置
  - `JWT_SECRET_KEY`: JWT 签名密钥（必需）
  - `ALLOWED_ORIGINS`: CORS 允许的域名列表

#### 邮件配置（可选）
  - `SMTP_HOST`: SMTP 服务器地址
  - `SMTP_PORT`: SMTP 端口（默认 `587`）
  - `SMTP_USERNAME`: SMTP 用户名
  - `SMTP_PASSWORD`: SMTP 密码
  - `SMTP_FROM`: 发件人邮箱地址

## 核心工作流程

### 用户认证流程
1. 通过 `POST /api/auth/register` 注册用户账号
2. 通过邮箱验证链接完成 `GET /api/auth/verify-email` 邮箱验证
3. 通过 `POST /api/auth/login` 登录获取 JWT 令牌
4. 所有后续 API 请求需要在 Authorization 头部携带 JWT 令牌

### 业务操作流程（需要认证）
1. 通过 `POST /api/periods` 创建社保账期
2. 通过 `POST /api/periods/{id}/files` 上传险种文件（单个或批量）
3. 可选通过 `POST /api/periods/{id}/roster` 上传员工花名册
4. 通过 `POST /api/periods/{id}/process` 处理数据生成汇总和扣款明细
5. 通过 `GET /api/periods/{id}/charges/export?part=personal|unit` 导出结果

### 系统监控流程
1. 通过 `GET /health` 检查系统健康状态
2. 通过 `GET /api/monitoring/metrics` 查看系统指标（需要认证）
3. 通过 `GET /api/audit/logs` 查看操作审计日志（需要认证）

## 主要 API 接口

### 认证相关（公开）
- `POST /api/auth/register` - 用户注册
- `POST /api/auth/login` - 用户登录
- `POST /api/auth/request-password-reset` - 请求密码重置
- `POST /api/auth/reset-password` - 重置密码
- `GET /api/auth/verify-email` - 邮箱验证
- `POST /api/auth/resend-verification` - 重发验证邮件

### 用户管理（需要认证）
- `GET /api/auth/profile` - 获取用户信息
- `POST /api/auth/logout` - 用户登出
- `POST /api/auth/change-password` - 修改密码

### 系统监控（公开）
- `GET /health` - 系统健康检查
- `GET /health/readiness` - 就绪检查
- `GET /health/liveness` - 存活检查
- `GET /version` - 版本信息

### 系统监控（需要认证）
- `GET /api/monitoring/metrics` - 系统指标
- `GET /api/monitoring/database` - 数据库状态
- `GET /api/monitoring/info` - 系统信息
- `POST /api/monitoring/maintenance` - 执行维护任务

### 操作审计（需要认证）
- `GET /api/audit/logs` - 查询审计日志
- `GET /api/audit/stats` - 审计统计信息

### 业务接口（需要认证）
- `GET/POST /api/periods` - 账期管理
- `POST /api/periods/{id}/files` - 单文件上传
- `POST /api/periods/{id}/files/batch` - 批量文件上传
- `POST /api/periods/{id}/roster` - 花名册上传（Excel/CSV 需包含 姓名、证件号码、部门 列）
- `POST /api/periods/{id}/process` - 数据处理和聚合
- `GET /api/periods/{id}/summary` - 查看汇总统计
- `GET /api/periods/{id}/charges` - 查看扣款明细
- `GET /api/periods/{id}/charges/export` - 导出 Excel

## 文件结构

```
frontend/
  app/
    page.tsx              # 主界面和交互逻辑
    layout.tsx            # 全局布局和 Provider
    providers.tsx         # Toast 和主题 Provider
  components/ui/          # shadcn/ui 组件
  lib/
    api.ts               # 后端 API 客户端函数
    types.ts             # TypeScript 类型定义

backend/
  main.go                # HTTP 服务器设置和数据库初始化
  internal/
    api/                 # 路由处理器和 HTTP 逻辑
      auth.go            # 认证相关处理器
      audit.go           # 审计日志处理器
      monitoring.go      # 监控相关处理器
    models/              # GORM 模型定义
      models.go          # 业务模型
      audit_log.go       # 审计日志模型
    service/             # Excel 解析和业务逻辑
      audit_service.go           # 审计日志服务
      email_service.go           # 邮件服务
      password_reset_service.go  # 密码重置服务
      email_verification_service.go # 邮箱验证服务
      monitoring_service.go      # 监控服务
    auth/                # 认证相关
      jwt.go             # JWT 处理
    middleware/          # 中间件
      audit.go           # 审计日志中间件

# 配置文件
.env.example                   # 开发环境配置模板
.env.production.example        # 生产环境配置模板
.env.postgres.example          # PostgreSQL 配置模板
docker-compose.postgres.yml    # PostgreSQL Docker 配置
DATABASE_MIGRATION.md          # 数据库迁移指南
```

## 重要说明

### 系统要求
- Go 1.24+ 后端运行环境
- PostgreSQL 15+ 推荐用于生产环境，SQLite 用于开发测试
- 所有 API 请求（除公开接口外）需要 JWT 认证

### 业务逻辑
- 系统支持中文文件名和数据处理
- Excel 文件必须遵循特定的列结构才能正确解析
- 医疗保险分为 `medical`（8.5%）和 `serious_illness`（1.5%）两个文件
- 花名册数据用于补充保险文件中缺失的部门信息
- 所有文件上传支持 Excel (.xls/.xlsx) 和 CSV 格式

### 安全特性
- 用户注册需要邮箱验证才能登录
- 密码重置通过安全令牌机制
- 所有操作记录完整审计日志
- JWT 令牌有过期时间限制

### 监控特性
- 系统健康检查和就绪检查
- 实时系统指标和数据库状态监控
- 操作审计日志查询和统计
- 自动清理过期令牌等维护任务

### 技术特性
- 前端使用 shadcn/ui 组件和 Sonner 进行通知提示
- 支持 SQLite 和 PostgreSQL 双数据库
- Docker 容器化部署支持
- 完整的环境配置模板
