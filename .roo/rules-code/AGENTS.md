# 项目编码规则（仅非显而易见）

## 后端关键模式

- **SQLite CGO 编译**: 所有后端代码在使用 SQLite 时**必须**使用 `CGO_ENABLED=1` 编译。在 Windows 上直接运行 `go run .` 会失败，除非安装了 GCC。
- **文件类型枚举逻辑**: `models.FileType` 枚举控制数据处理模式：
  - `FileTypeNormal`: 覆盖相同方案/部分的现有数据
  - `FileTypeAdjustment`: 累加数据（添加到现有费用）
  - 在错误的上下文中混用这些类型会破坏 `processor.go` 中的聚合逻辑
- **中文 Excel 表头映射**: `service/processor.go` 中的 `headerMap` 将中文列名转换为结构体字段。添加新列时，需要更新此映射。
- **花名册回退优先级**: `buildAggregates()` 函数（第 462 行）对 `name` 和 `department` 字段优先使用花名册数据而非文件数据。这是为了提高数据质量而有意为之。
- **JWT 中间件顺序**: 在路由组中**必须**先应用 JWT 中间件，再应用审计中间件。顺序颠倒会破坏审计日志中的用户上下文。

## 前端关键模式

- **API 基础检测**: `lib/api.ts` 中的 `getApiBase()` 在导入时运行，**不是**在请求时。更改环境变量需要重新构建。
- **localStorage 存储 JWT**: 令牌存储使用 `localStorage.getItem("token")`，**不是** cookies。所有 `request()` 函数调用都会自动注入此令牌。
- **Turbopack 要求**: 所有 npm 脚本**必须**包含 `--turbopack` 标志。标准的 Next.js 命令会构建失败。
- **组织类型约束**: 组织类型的 TypeScript 联合类型**必须**使用显式字面量 `"group" | "subsidiary" | "department"`，**不能**用 `string` 或 `any`。否则编译器无法捕获运行时错误。

## 数据库模式

- **启动时自动迁移**: GORM 在 `main.go` 中自动迁移所有模型。**不使用**手动迁移脚本。
- **管理员用户自动创建**: 如果缺失，首次运行时会创建默认管理员账户。查看 `main.go` 中的 `initializeDefaultAdmin()`。
- **双数据库支持**: 系统通过 `SIAPP_DATABASE_TYPE` 环境变量检测数据库类型。PostgreSQL 需要密码；SQLite 需要 GCC 编译器。

## 代码风格陷阱

- **Windows 命令**: PowerShell 不支持 `&&`。在脚本中使用 `;` 或分开命令。
- **无根 package.json**: Monorepo **没有**根 `package.json`。运行命令前必须 `cd` 进入 `frontend/` 或 `backend/`。
- **ESLint FlatCompat**: 在 `eslint.config.mjs` 中使用向后兼容包装器用于 Next.js 配置。不要转换为纯扁平配置。
- 项目代码注释及文档全部使用简体中文。