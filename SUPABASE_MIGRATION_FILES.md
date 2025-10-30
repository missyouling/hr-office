# Supabase 迁移文件索引

本文档列出了 Supabase 迁移所需的所有文件及其用途。

## 📋 文档文件

### 1. 规划文档
- **SUPABASE_MIGRATION_README.md** - 迁移总览和快速启动指南
- **SUPABASE_MIGRATION_PLAN.md** - 详细的架构设计和迁移策略
- **SUPABASE_MIGRATION_IMPLEMENTATION.md** - 实施步骤和代码示例
- **SUPABASE_MIGRATION_CHECKLIST.md** - 完整的检查清单和测试计划
- **SUPABASE_MIGRATION_FILES.md** - 本文件，文件索引

## 🗄️ 数据库迁移文件

### Supabase 目录
```
supabase/
├── config.toml                          # Supabase 本地开发配置
└── migrations/
    └── 001_initial_schema.sql          # 初始数据库架构（477行）
```

**001_initial_schema.sql** 包含：
- 12个业务表定义
- 完整的 RLS（行级安全）策略
- 自动触发器（updated_at、新用户profile）
- 辅助函数（get_user_periods、get_period_stats）
- 用户ID映射表（user_mappings）

## 🎨 前端文件

### Supabase 集成
```
frontend/lib/supabase/
├── client.ts                            # 浏览器端客户端（8行）
├── server.ts                            # 服务端客户端（33行）
└── auth-context.tsx                     # 认证上下文Provider（118行）
```

### 认证页面
```
frontend/app/auth/
├── callback/
│   └── page.tsx                        # 邮箱验证回调页面（37行）
└── reset-password/
    └── page.tsx                        # 密码重置页面（129行）
```

### 环境变量
```
frontend/.env.example                    # 前端环境变量模板
```

需要配置的变量：
- `NEXT_PUBLIC_SUPABASE_URL`
- `NEXT_PUBLIC_SUPABASE_ANON_KEY`
- `NEXT_PUBLIC_BFF_URL`

## 🔧 后端文件

### Supabase 集成
```
backend/internal/supabase/
└── client.go                            # Go Supabase 客户端包装器（49行）
```

### 中间件
```
backend/internal/middleware/
└── supabase_auth.go                     # JWT 验证中间件（97行）
```

### 环境变量
```
backend/.env.example                     # 后端环境变量模板
```

需要配置的变量：
- `SUPABASE_URL`
- `SUPABASE_SERVICE_KEY`
- `SUPABASE_JWT_SECRET`
- `BFF_PORT`

## 🛠️ 脚本文件

### 自动化脚本
```
scripts/
├── setup-supabase-migration.sh         # Linux/Mac 安装脚本（111行）
├── setup-supabase-migration.bat        # Windows 安装脚本（86行）
└── migrate-data-to-supabase.go         # 数据迁移工具（191行）
```

**setup-supabase-migration.sh/bat** 功能：
- 检查前置条件（Node.js、Go）
- 安装前端依赖（@supabase/supabase-js、@supabase/ssr）
- 安装后端依赖（supabase-go）
- 创建环境变量文件
- 显示下一步操作指引

**migrate-data-to-supabase.go** 功能：
- 连接源数据库（SQLite 或 PostgreSQL）
- 读取旧用户数据
- 生成用户迁移报告（JSON格式）
- 提供手动迁移指引

## 📦 依赖包

### 前端依赖
需要安装的npm包：
```bash
npm install @supabase/supabase-js @supabase/ssr @supabase/auth-helpers-nextjs
```

### 后端依赖
需要安装的Go包：
```bash
go get github.com/supabase-community/supabase-go
go get github.com/supabase-community/postgrest-go
```

## 🚀 快速启动流程

### 1. 准备阶段
```bash
# 阅读文档
cat SUPABASE_MIGRATION_README.md
cat SUPABASE_MIGRATION_PLAN.md

# 在 Supabase Dashboard 创建项目
# 获取 URL 和 Keys (Settings → API)
```

### 2. 安装依赖
```bash
# Linux/Mac
bash scripts/setup-supabase-migration.sh

# Windows
scripts\setup-supabase-migration.bat
```

### 3. 配置环境变量
```bash
# 编辑前端环境变量
vim frontend/.env.local

# 编辑后端环境变量
vim backend/.env
```

### 4. 初始化数据库
```sql
-- 在 Supabase SQL Editor 执行
-- 文件: supabase/migrations/001_initial_schema.sql
```

### 5. 迁移数据（可选）
```bash
# 编辑迁移配置
vim migration-config.json

# 运行迁移脚本
cd scripts
go run migrate-data-to-supabase.go
```

### 6. 开始开发
```bash
# 启动前端
cd frontend
npm run dev

# 启动后端
cd backend
go run .
```

## 📊 文件统计

| 类别 | 文件数 | 总行数 |
|------|--------|--------|
| 文档 | 5 | ~2000+ |
| 数据库迁移 | 2 | 555 |
| 前端代码 | 5 | 325 |
| 后端代码 | 2 | 146 |
| 脚本工具 | 3 | 388 |
| **总计** | **17** | **~3414+** |

## 🔍 关键文件说明

### 必须修改的文件
1. **frontend/.env.local** - 填写 Supabase URL 和 ANON_KEY
2. **backend/.env** - 填写 Supabase URL 和 SERVICE_KEY
3. **backend/main.go** - 需要重构，集成 Supabase 中间件
4. **frontend/app/layout.tsx** - 需要包裹 AuthProvider

### 需要删除的文件（迁移后）
1. **backend/internal/auth/jwt.go** - 自建JWT系统
2. **backend/internal/api/auth.go** - 自建认证API
3. **backend/internal/models/models.go** 中的 User 模型
4. **backend/internal/service/email_verification_service.go** - Supabase自带
5. **backend/internal/service/password_reset_service.go** - Supabase自带

### 需要重构的文件
1. **frontend/lib/api.ts** - 区分 Supabase 直接调用和 BFF 调用
2. **frontend/app/auth/page.tsx** - 使用 AuthContext
3. **backend/internal/api/handler.go** - 移除认证路由
4. **所有业务API** - 使用新的 Supabase 中间件

## 📝 迁移检查清单

详细的检查清单请查看：**SUPABASE_MIGRATION_CHECKLIST.md**

包含以下部分：
- ✅ 准备工作（10项）
- ✅ 前端迁移（15项）
- ✅ 后端迁移（20项）
- ✅ 数据库迁移（12项）
- ✅ 测试验证（20项）
- ✅ 部署上线（15项）

## 🆘 故障排除

### 常见问题

**Q1: TypeScript 报错找不到 @supabase/supabase-js**
- A: 运行 `npm install` 安装依赖包

**Q2: Go 编译错误找不到 supabase-go**
- A: 运行 `go get github.com/supabase-community/supabase-go`

**Q3: RLS 策略阻止数据访问**
- A: 检查用户是否已登录，JWT是否有效

**Q4: 数据迁移脚本连接失败**
- A: 检查 migration-config.json 配置是否正确

**Q5: 前端无法调用 BFF API**
- A: 检查 NEXT_PUBLIC_BFF_URL 环境变量

## 📚 相关资源

### Supabase 官方文档
- [快速入门](https://supabase.com/docs/guides/getting-started)
- [认证系统](https://supabase.com/docs/guides/auth)
- [行级安全](https://supabase.com/docs/guides/auth/row-level-security)
- [JavaScript 客户端](https://supabase.com/docs/reference/javascript)
- [Go 客户端](https://github.com/supabase-community/supabase-go)

### Next.js + Supabase
- [Next.js 集成指南](https://supabase.com/docs/guides/auth/auth-helpers/nextjs)
- [SSR 认证](https://supabase.com/docs/guides/auth/server-side)

### 迁移最佳实践
- [从自建认证迁移](https://supabase.com/docs/guides/auth/auth-migration)
- [数据库迁移](https://supabase.com/docs/guides/database/migrations)

## 📞 支持

遇到问题？
1. 查看 **SUPABASE_MIGRATION_CHECKLIST.md** 中的故障排除部分
2. 阅读 **SUPABASE_MIGRATION_IMPLEMENTATION.md** 的详细步骤
3. 参考 Supabase 官方文档
4. 在项目 Issue 中提问

---

**版本**: v1.0.0  
**创建日期**: 2025-10-30  
**最后更新**: 2025-10-30  
**状态**: ✅ 完成