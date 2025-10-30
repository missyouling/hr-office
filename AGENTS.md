# AGENTS.md

本文件为AI助手在此代码仓库中工作时提供指导。

## 非显而易见的项目特定模式

### 后端 (Go)

- **SQLite CGO 要求**: 后端**必须**使用 `CGO_ENABLED=1` 和 GCC 编译器才能使用 SQLite 驱动。Windows 上需要先安装 MinGW-w64。使用 Docker 可以绕过此问题。
- **数据库自动切换**: 系统通过 `SIAPP_DATABASE_TYPE` 环境变量自动检测数据库类型。默认是 SQLite，但生产环境推荐使用 PostgreSQL。
- **默认管理员账户**: 系统首次运行时会自动创建 `admin` 用户（密码：`admin123`）如果不存在。位于 `main.go:initializeDefaultAdmin()`。
- **文件类型枚举至关重要**: `models.FileType` 区分 `normal`（覆盖模式）和 `adjustment`（累加模式）。混用这两种类型会破坏数据处理逻辑。
- **中文列名映射**: Excel 解析器在 `service/processor.go` 中使用 `headerMap` 将中文表头（如"姓名"、"证件号码"）映射到内部字段名。必须同时处理中英文列名。
- **花名册回退逻辑**: 部门信息优先使用花名册数据而非源文件数据。参见 `processor.go:462-614` 中的 `buildAggregates()`。

### 前端 (Next.js)

- **动态 API 检测**: `lib/api.ts` 中的 `getApiBase()` 根据主机名自动检测 API URL。Localhost 使用 `:8081`，生产环境使用 `https://${hostname}/api`。运行时**不可**配置。
- **必须使用 Turbopack**: 所有 npm 脚本都使用 `--turbopack` 标志。不加此标志的标准 Next.js 命令无法工作。
- **认证令牌存储在 localStorage**: JWT 存储在 localStorage，**不是** cookies。所有 API 请求自动从 `localStorage.getItem("token")` 注入 `Authorization` 头。
- **类型不匹配陷阱**: 应使用 `AuditStats.stats.total_events`（嵌套），**不是** `AuditStats.total_logs`（原代码中的 bug）。
- **组织类型联合**: 使用组织类型时，必须显式类型化为 `"group" | "subsidiary" | "department"`，**不能**用 `string` 或 `any`。

### 开发环境特性

- **Windows PowerShell**: 不支持 `&&`。使用分号 `;` 或分开命令。
- **端口映射**: Docker 映射后端 `:8080` → 主机 `:8081`，前端 `:8080` → 主机 `:10086`。直接本地开发时后端使用 `:8080`。
- **ESLint 配置**: 使用 FlatCompat 向后兼容 Next.js 配置。参见 `eslint.config.mjs`。
- **Monorepo 结构**: 没有根 `package.json`。必须从 `frontend/` 或 `backend/` 子目录运行命令。

### 关键数据流

1. **文件上传流程**: 上传 → 解析（`ParseSourceFile`）→ 存储 `RawRecord` → 手动触发 `ProcessPeriod` → 生成 `PersonalCharge`/`UnitCharge`
2. **调整流程**: 以 `FileTypeAdjustment` 上传 → `ProcessAdjustments` → 与现有费用合并（**不会**覆盖）
3. **花名册导入**: 一键导入从**任意**期间复制最新花名册。如果不存在则回退到手动上传。

### CLAUDE.md 中的安全说明

- 登录前**必须**验证邮箱（后端强制执行）
- JWT 令牌存储在客户端并带有过期时间
- 所有业务 API 都需要认证（中间件顺序很重要：JWT → Audit）
- CORS 源可通过 `ALLOWED_ORIGINS` 环境变量配置

## 命令参考

### 前端
```bash
cd frontend
npm install
npm run dev          # 使用 Turbopack，绑定到 0.0.0.0:3000
npm run build        # 使用 Turbopack 的生产构建
npm run lint         # ESLint 检查
```

### 后端
```bash
cd backend
go mod tidy

# SQLite 模式（默认，Windows 上需要 GCC）
CGO_ENABLED=1 go run .

# PostgreSQL 模式
SIAPP_DATABASE_TYPE=postgres SIAPP_DB_PASSWORD=xxx go run .

# Docker（Windows 推荐）
docker-compose up -d
```

### Docker 端口映射
- 后端: `localhost:8081` → 容器 `:8080`
- 前端: `localhost:10086` → 容器 `:8080`