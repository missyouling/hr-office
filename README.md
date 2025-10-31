# 人事行政管理系统 (hr-office)

一个现代化的人事与社保账期管理平台，代号 `hr-office`，采用 Go 后端 + Next.js 前端架构，支持多险种文件上传、数据处理和 Excel 报表生成。

## 📅 最新更新

### v1.3.0 (2025-10-21) 🚀
- ✅ **Docker Hub发布**: 构建并发布生产就绪的Docker镜像到Docker Hub
- ✅ **生产环境优化**: 默认使用PostgreSQL，完整的生产环境配置
- ✅ **容器化部署**: 完善的Docker Compose生产环境配置
- ✅ **镜像优化**: 多阶段构建，后端镜像仅69.9MB，大幅优化镜像大小
- ✅ **部署测试**: 完整的生产环境模拟部署测试，所有服务运行正常

### v1.2.0 (2025-01-20)
- ✅ **修复邮箱验证**: 解决了邮箱验证页面 404 错误，新增 `/verify-email` 路由
- ✅ **数据库约束优化**: 修复了文件上传和花名册导入的外键约束问题
- ✅ **模型结构调整**: 所有数据模型的 UserID 字段改为可选，支持无用户关联的操作
- ✅ **系统稳定性**: 完成生产环境部署测试，系统运行稳定
- ✅ **中文本地化**: 所有用户界面和错误信息完成中文本地化

### 当前状态
- 🟢 **生产就绪**: 系统已完成生产环境测试，所有核心功能正常运行
- 🟢 **Docker Hub**: 官方镜像已发布，支持一键部署
- 🟢 **数据库支持**: PostgreSQL 生产环境配置完成
- 🟢 **邮件服务**: SMTP 邮件发送服务配置完成
- 🟢 **容器化**: Docker 容器化部署完成

## 🚀 功能特性

### 核心业务功能
- **账期管理**: 创建和管理社保账期（如 "2025-08"）
- **文件上传**: 支持单个或批量上传险种文件（Excel/CSV 格式）
- **花名册管理**: 上传员工花名册，补充部门信息
- **数据处理**: 自动解析保险文件并生成汇总统计
- **报表导出**: 生成个人和单位扣款明细 Excel 报表
- **多险种支持**: 养老、基本医疗、大额/生育、失业、工伤保险

### 系统安全与监控
- **用户认证**: JWT 身份验证和注册登录系统
- **邮箱验证**: 用户注册邮箱验证，阻止未验证用户登录
- **密码重置**: 安全的密码重置流程，令牌过期和一次性使用
- **操作审计**: 完整的用户操作审计日志系统
- **系统监控**: 健康检查、指标监控、数据库状态监控
- **数据库支持**: SQLite（默认）+ PostgreSQL（生产推荐）

## 🏗️ 系统架构

### 技术栈

**后端**
- Go 1.24+
- Chi 路由器
- GORM + SQLite/PostgreSQL (双数据库支持)
- JWT 身份认证
- 完整审计日志系统
- 系统监控与健康检查
- Excelize (Excel 处理)

**前端**
- Next.js 15
- React 19
- TypeScript
- Tailwind CSS
- shadcn/ui 组件库

**部署**
- Docker & Docker Compose
- 支持开发和生产环境

### 核心数据模型

- `Period`: 社保账期
- `SourceFile`: 上传的险种文件
- `RawRecord`: 解析的个人保险记录
- `PeriodSummary`: 按险种/缴费部分聚合的统计数据
- `PersonalCharge`/`UnitCharge`: 个人/单位扣款明细
- `RosterEntry`: 员工花名册数据

## 📦 快速开始

### 环境要求

- Go 1.24+
- Node.js 18+
- PostgreSQL 15+ (推荐) 或 SQLite (默认)
- Docker (可选)

### 本地开发

1. **克隆项目**
```bash
git clone git@github.com:missyouling/hr-office.git
cd hr-office
```

2. **启动后端服务**
```bash
cd backend
go run .
# 服务运行在 http://localhost:8080
```

3. **启动前端服务**
```bash
cd frontend
npm install
cp .env.local.example .env.local
npm run dev
# 服务运行在 http://localhost:3000
```

### Docker 部署

#### 🐳 使用 Docker Hub 镜像 (推荐)

我们已将生产就绪的镜像发布到 Docker Hub：

- **后端镜像**: `koujiang2025/hr-office-backend:latest` (69.9MB)
- **前端镜像**: `koujiang2025/hr-office-frontend:latest` (290MB)

**一键部署生产环境**:
```bash
# 1. 创建部署目录
mkdir -p ~/siapp-prod && cd ~/siapp-prod

# 2. 下载生产环境配置
curl -O https://raw.githubusercontent.com/missyouling/hr-office/master/docker-compose.production.yml
curl -O https://raw.githubusercontent.com/missyouling/hr-office/master/.env.production.example

# 3. 配置环境变量
cp .env.production.example .env.production
# 编辑 .env.production 文件，设置数据库密码、域名、SMTP等参数

# 4. 创建数据持久化目录
sudo mkdir -p /var/lib/siapp/{postgres,data}
sudo chown -R 999:999 /var/lib/siapp/postgres
sudo chown -R 1000:1000 /var/lib/siapp/data

# 5. 启动服务
docker compose -f docker-compose.production.yml up -d
```

**验证部署**:
```bash
# 检查服务状态
docker compose -f docker-compose.production.yml ps

# 后端健康检查
curl http://localhost:8080/health

# 前端页面 (端口可在.env.production中配置)
curl http://localhost:3000
```

#### 本地开发部署

```bash
# 构建并启动所有服务
docker-compose up -d

# 或使用部署脚本
./deploy.sh
```

## 🔧 配置说明

### 后端环境变量

#### 服务器配置
- `SIAPP_ADDR`: HTTP 监听地址（默认 `:8080`）

#### 数据库配置
- `SIAPP_DATABASE_TYPE`: 数据库类型 (`sqlite` 或 `postgres`，默认 `sqlite`)
- `SIAPP_DATABASE_PATH`: SQLite 文件路径（默认 `./data/siapp.db`）

#### PostgreSQL 配置（当 SIAPP_DATABASE_TYPE=postgres 时）
- `SIAPP_DB_HOST`: PostgreSQL 主机地址（默认 `localhost`）
- `SIAPP_DB_PORT`: PostgreSQL 端口（默认 `5432`）
- `SIAPP_DB_USER`: 数据库用户名（默认 `siapp`）
- `SIAPP_DB_PASSWORD`: 数据库密码（必需）
- `SIAPP_DB_NAME`: 数据库名称（默认 `siapp`）
- `SIAPP_DB_SSLMODE`: SSL 模式（默认 `require`）

#### 安全配置
- `JWT_SECRET_KEY`: JWT 签名密钥（必需）
- `JWT_TOKEN_DURATION`: JWT 令牌有效期（默认 `24h`）
- `ALLOWED_ORIGINS`: CORS 允许的域名列表

#### 邮件配置（SMTP）
- `SMTP_HOST`: SMTP 服务器地址
- `SMTP_PORT`: SMTP 端口（默认 `587`）
- `SMTP_USERNAME`: SMTP 用户名
- `SMTP_PASSWORD`: SMTP 密码
- `SMTP_FROM`: 发件人邮箱地址

### 前端环境变量

复制 `frontend/.env.local.example` 为 `.env.local`:
```bash
NEXT_PUBLIC_API_BASE_URL=http://localhost:8080
```

## 📋 使用流程

1. **创建账期**: 通过界面创建社保账期（如 "2025-08"）
2. **上传文件**: 上传各险种的保险文件（Excel/CSV）
3. **上传花名册**: 可选上传员工花名册补充部门信息
4. **处理数据**: 点击处理按钮生成汇总和扣款明细
5. **导出报表**: 下载个人或单位扣款明细 Excel 文件

## 📊 支持的险种

| 险种 | 代码 | 说明 |
|------|------|------|
| 养老保险 | `pension` | 基本养老保险 |
| 基本医疗 | `medical` | 基本医疗保险（8.5%） |
| 大额/生育 | `serious_illness` | 大额医疗/生育保险（1.5%） |
| 失业保险 | `unemployment` | 失业保险 |
| 工伤保险 | `injury` | 工伤保险 |

## 🔌 API 接口

### 用户认证（公开接口）
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

### 账期管理（需要认证）
- `GET /api/periods` - 获取账期列表
- `POST /api/periods` - 创建新账期

### 文件上传（需要认证）
- `POST /api/periods/{id}/files` - 单文件上传
- `POST /api/periods/{id}/files/batch` - 批量文件上传
- `POST /api/periods/{id}/roster` - 花名册上传

### 数据处理（需要认证）
- `POST /api/periods/{id}/process` - 处理数据和聚合
- `GET /api/periods/{id}/summary` - 查看汇总统计
- `GET /api/periods/{id}/charges` - 查看扣款明细

### 报表导出（需要认证）
- `GET /api/periods/{id}/charges/export?part=personal` - 导出个人扣款明细
- `GET /api/periods/{id}/charges/export?part=unit` - 导出单位扣款明细

## 🐳 Docker 镜像信息

### 镜像仓库
- **Docker Hub**: [koujiang2025/hr-office-backend](https://hub.docker.com/r/koujiang2025/hr-office-backend)
- **Docker Hub**: [koujiang2025/hr-office-frontend](https://hub.docker.com/r/koujiang2025/hr-office-frontend)

### 镜像特性
| 镜像 | 大小 | 基础镜像 | 特性 |
|------|------|----------|------|
| 后端 | 69.9MB | golang:1.25-alpine | 多阶段构建、PostgreSQL优化、健康检查 |
| 前端 | 290MB | node:20-alpine | Next.js 15生产构建、环境变量支持 |

### 生产环境配置
- **默认数据库**: PostgreSQL 15
- **网络模式**: 专用Docker网络 (172.25.0.0/16)
- **安全配置**: 非root用户运行、容器隔离
- **数据持久化**: 绑定挂载 `/var/lib/siapp`
- **健康检查**: 自动服务状态监控

## 📁 项目结构

```
├── backend/                 # Go 后端
│   ├── internal/
│   │   ├── api/            # HTTP 处理器
│   │   ├── models/         # 数据模型
│   │   └── service/        # 业务逻辑
│   ├── data/               # 数据库文件
│   └── main.go            # 入口文件
├── frontend/               # Next.js 前端
│   ├── app/               # App Router 页面
│   ├── components/        # 组件库
│   └── lib/               # 工具函数
├── docs/                          # 文档
├── docker-compose.yml             # 开发环境 Docker 编排
├── docker-compose.production.yml  # 生产环境 Docker 编排
├── .env.production.example        # 生产环境配置模板
├── DEPLOYMENT_SUMMARY.md          # 部署操作总结
└── deploy.sh                     # 部署脚本
```

## 🔒 安全说明

- 数据库文件和上传文件已在 `.gitignore` 中排除
- 环境配置文件不会被提交到版本控制
- 支持 CORS 配置用于跨域访问

## 🤝 贡献指南

1. Fork 项目
2. 创建功能分支 (`git checkout -b feature/AmazingFeature`)
3. 提交更改 (`git commit -m 'Add some AmazingFeature'`)
4. 推送到分支 (`git push origin feature/AmazingFeature`)
5. 打开 Pull Request

## 📄 许可证

本项目采用 MIT 许可证 - 查看 [LICENSE](LICENSE) 文件了解详情。

## 📖 部署文档

详细的部署配置和操作指南请参考：

- [部署操作总结](DEPLOYMENT_SUMMARY.md) - 最新的生产环境部署详细指南
- [Docker Hub 镜像](https://hub.docker.com/u/koujiang2025) - 官方发布的容器镜像
- [生产环境配置](.env.production.example) - 完整的环境变量配置模板

## 📞 联系方式

如有问题或建议，请通过以下方式联系：

- 创建 [Issue](https://github.com/missyouling/hr-office/issues)
- 发送 Pull Request

---

⭐ 如果这个项目对你有帮助，请给它一个星标！
