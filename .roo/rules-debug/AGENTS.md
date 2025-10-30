# 项目调试规则（仅非显而易见）

## 后端调试

- **CGO 构建错误**: 如果看到 "Binary was compiled with 'CGO_ENABLED=0'"，说明 SQLite 驱动需要 CGO。在 Windows 上，安装 MinGW-w64 以获取 GCC，然后设置 `CGO_ENABLED=1`。
- **数据库连接问题**: 检查 `SIAPP_DATABASE_TYPE` 环境变量。PostgreSQL 需要设置 `SIAPP_DB_PASSWORD`；缺失会导致静默启动失败。
- **默认管理员登录**: 首次设置会创建 `admin:admin123` 账户。如果登录失败，检查 `main.go:initializeDefaultAdmin()`。
- **中间件顺序 Bug**: JWT 中间件**必须**在审计中间件之前。顺序颠倒会破坏用户上下文提取。
- **文件类型混淆**: 如果数据处理产生意外结果，检查 `source_files` 表中的 `file_type` 列。`normal` 和 `adjustment` 类型有不同的聚合逻辑。

## 前端调试

- **API 连接问题**: 检查浏览器控制台中 `lib/api.ts:getApiBase()` 的 `[API检测]` 日志。Localhost 应使用 `:8081`，生产环境使用 HTTPS + 当前域名。
- **认证令牌错误**: JWT 存储在 `localStorage.getItem("token")`。如果登录后仍收到 401 错误，清除 localStorage 并重新登录。
- **构建失败**: 所有 npm 命令**需要** `--turbopack` 标志。缺失会导致隐晦的构建错误。
- **生产环境类型错误**: `AuditStats` 类型使用嵌套的 `stats.total_events`，**不是**扁平的 `total_logs`。这是导致生产构建失败的 bug。
- **组织类型错误**: 如果使用 `string` 或 `any`，TypeScript 不会捕获无效的组织类型。必须使用显式联合：`"group" | "subsidiary" | "department"`。

## Docker 调试

- **端口冲突**: 后端映射 `:8080` → 主机 `:8081`，前端 `:8080` → 主机 `:10086`。如果端口冲突，检查 `docker-compose.yml`。
- **卷持久化**: 后端数据存储在 `backend_data` 卷中。如果数据库意外重置，卷可能被删除了。
- **容器日志**: 使用 `docker-compose logs -f backend` 或 `docker-compose logs -f frontend` 查看实时日志。

## 数据库调试

- **记录缺失**: 检查 `file_type` 字段。`normal` 文件覆盖以前的数据；`adjustment` 文件累加。
- **花名册未应用**: 仅当源文件中的 `department` 或 `name` 字段为空时，才使用花名册数据。参见 `processor.go` 中的 `buildAggregates()`。
- **自动迁移问题**: GORM 在启动时自动迁移。如果架构更改未应用，检查服务器日志中的迁移错误。

## Windows 特定问题

- **PowerShell 命令错误**: 不支持 `&&` 操作符。改用 `;` 或分开运行命令。
- **找不到 GCC**: SQLite 驱动需要 C 编译器。安装 MinGW-w64 并添加到 PATH，或使用 Docker 避免此问题。
- **行尾问题**: Git 可能在 Windows 上将 LF 转换为 CRLF。这通常不会造成问题，但可能影响 Docker 中的 shell 脚本。