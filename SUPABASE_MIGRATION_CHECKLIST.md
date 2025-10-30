
# Supabase 迁移检查清单

> 本文档提供完整的迁移检查清单、测试计划和快速启动指南

---

## 🚀 快速启动指南

### 前置条件

- [ ] Supabase账号已创建
- [ ] 项目已在Supabase Dashboard创建
- [ ] 已获取 `SUPABASE_URL` 和 `SUPABASE_ANON_KEY`
- [ ] 已获取 `SUPABASE_SERVICE_KEY`（用于后端）
- [ ] 已安装Node.js 18+、Go 1.21+
- [ ] 已备份现有数据库

### 环境配置

#### 1. Supabase环境变量

创建 `frontend/.env.local`：
```bash
NEXT_PUBLIC_SUPABASE_URL=https://vjpvrzphtnxawkqwuogn.supabase.co
NEXT_PUBLIC_SUPABASE_ANON_KEY=eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
NEXT_PUBLIC_BFF_URL=http://localhost:8080/api
```

创建 `backend/.env`：
```bash
SUPABASE_URL=https://vjpvrzphtnxawkqwuogn.supabase.co
SUPABASE_SERVICE_KEY=eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
SUPABASE_JWT_SECRET=your-jwt-secret-from-supabase-settings

# 数据库连接（用于迁移）
OLD_DATABASE_URL=postgresql://user:password@localhost:5432/siapp

# 服务端口
PORT=8080
```

#### 2. 获取JWT Secret

在Supabase Dashboard中：
1. 进入 Settings → API
2. 复制 `JWT Secret`（用于验证token）
3. 添加到 `SUPABASE_JWT_SECRET` 环境变量

---

## ✅ 迁移检查清单

### 阶段1：准备工作（第1天）

#### Supabase项目设置
- [ ] 创建Supabase项目
- [ ] 记录项目URL和密钥
- [ ] 配置认证设置
  - [ ] 启用Email provider
  - [ ] 配置邮箱确认
  - [ ] 自定义邮件模板（可选）
- [ ] 配置站点URL（用于邮箱验证链接）

#### 本地开发环境
- [ ] 安装前端依赖：`cd frontend && npm install`
- [ ] 安装后端依赖：`cd backend && go mod tidy`
- [ ] 配置环境变量文件
- [ ] 测试环境变量加载

#### 代码仓库准备
- [ ] 创建新分支：`git checkout -b feature/supabase-migration`
- [ ] 提交当前代码：`git add . && git commit -m "Pre-migration checkpoint"`

---

### 阶段2：数据库迁移（第2-3天）

#### 数据库模式创建
- [ ] 在Supabase SQL Editor中创建profiles表
```sql
CREATE TABLE public.profiles (
  id uuid PRIMARY KEY REFERENCES auth.users(id) ON DELETE CASCADE,
  username text UNIQUE NOT NULL,
  full_name text,
  company_id text,
  active boolean DEFAULT true,
  created_at timestamptz DEFAULT now(),
  updated_at timestamptz DEFAULT now()
);
```

- [ ] 创建业务表（periods, source_files等）
- [ ] 配置外键关系
- [ ] 创建索引
- [ ] 启用RLS（Row Level Security）
- [ ] 创建RLS策略

#### 数据迁移脚本
- [ ] 编写用户数据迁移脚本
- [ ] 测试迁移脚本（在开发环境）
- [ ] 验证ID映射关系
- [ ] 迁移关联数据
- [ ] 验证数据完整性

#### RLS策略测试
- [ ] 测试用户只能访问自己的数据
- [ ] 测试匿名用户访问限制
- [ ] 测试跨用户数据隔离

---

### 阶段3：认证系统迁移（第4-5天）

#### Supabase Auth配置
- [ ] 配置认证提供商
- [ ] 设置邮箱模板
- [ ] 配置重定向URL
- [ ] 测试邮箱验证流程

#### 前端认证集成
- [ ] 安装Supabase客户端库
- [ ] 创建Supabase客户端文件
- [ ] 创建AuthContext
- [ ] 更新登录页面
- [ ] 更新注册页面
- [ ] 实现登出功能
- [ ] 添加受保护路由逻辑

#### 用户数据迁移
- [ ] 使用Admin API批量创建用户
- [ ] 创建对应的profiles记录
- [ ] 发送密码重置邮件给所有用户
- [ ] 验证用户可以登录

---

### 阶段4：前端改造（第6-8天）

#### API层重构
- [ ] 创建新的API层（区分Supabase和BFF）
- [ ] 迁移简单查询到Supabase直接调用
- [ ] 保留复杂逻辑调用Go BFF
- [ ] 更新所有API调用点

#### 组件更新
- [ ] 更新认证相关组件
- [ ] 更新用户Profile组件
- [ ] 更新数据展示组件
- [ ] 测试所有交互流程

#### 状态管理
- [ ] 移除localStorage中的旧token
- [ ] 使用Supabase session管理
- [ ] 更新全局状态管理

---

### 阶段5：后端改造（第9-11天）

#### Go后端集成Supabase
- [ ] 安装Supabase Go SDK
- [ ] 创建Supabase客户端
- [ ] 实现JWT验证中间件
- [ ] 从JWT提取用户信息

#### 移除自建认证
- [ ] 删除JWT生成代码
- [ ] 删除密码哈希逻辑
- [ ] 删除用户注册/登录端点
- [ ] 删除密码重置逻辑
- [ ] 删除邮箱验证逻辑

#### 数据访问层重构
- [ ] 移除GORM User模型
- [ ] 更新所有业务逻辑中的用户ID类型（uint → uuid）
- [ ] 使用Supabase SDK查询数据
- [ ] 保留复杂业务逻辑

#### API端点更新
- [ ] 保留文件上传端点
- [ ] 保留数据处理端点
- [ ] 保留导出端点
- [ ] 移除认证相关端点
- [ ] 更新所有端点的认证验证

---

### 阶段6：测试（第12-14天）

#### 功能测试
- [ ] 用户注册流程
  - [ ] 注册新用户
  - [ ] 接收验证邮件
  - [ ] 点击验证链接
  - [ ] 验证成功后登录
- [ ] 用户登录流程
  - [ ] 使用邮箱密码登录
  - [ ] 验证session创建
  - [ ] 验证token有效期
- [ ] 密码管理
  - [ ] 请求密码重置
  - [ ] 接收重置邮件
  - [ ] 设置新密码
  - [ ] 使用新密码登录
- [ ] 业务功能
  - [ ] 创建期间
  - [ ] 上传文件
  - [ ] 处理数据
  - [ ] 导出Excel
  - [ ] 查看花名册

#### 数据隔离测试
- [ ] 创建多个测试用户
- [ ] 验证用户A看不到用户B的数据
- [ ] 测试RLS策略有效性

#### 性能测试
- [ ] 测试登录响应时间
- [ ] 测试数据查询性能
- [ ] 测试文件上传性能
- [ ] 测试并发请求

#### 安全测试
- [ ] 测试未认证访问被拒绝
- [ ] 测试token过期后自动登出
- [ ] 测试SQL注入防护（RLS）
- [ ] 测试XSS防护

---

### 阶段7：部署上线（第15-16天）

#### 生产环境准备
- [ ] 在Supabase创建生产项目
- [ ] 配置生产环境变量
- [ ] 配置生产域名
- [ ] 配置CORS设置

#### 数据迁移（生产）
- [ ] 备份生产数据库
- [ ] 执行生产数据迁移
- [ ] 验证数据完整性
- [ ] 创建数据快照

#### 应用部署
- [ ] 部署前端（Vercel/Netlify）
- [ ] 部署后端（Railway/Fly.io）
- [ ] 配置环境变量
- [ ] 测试生产环境

#### 用户通知
- [ ] 发送系统升级通知
- [ ] 提供密码重置指南
- [ ] 准备用户支持文档

---

## 🧪 测试计划

### 单元测试

#### 前端
```bash
# 测试认证Hook
npm test src/lib/auth-context.test.tsx

# 测试API层
npm test src/lib/api.test.ts
```

#### 后端
```bash
# 测试JWT验证中间件
go test ./internal/middleware

# 测试Supabase集成
go test ./internal/supabase
```

### 集成测试

创建 `tests/integration/auth.test.ts`：

```typescript
import { createClient } from '@supabase/supabase-js'

describe('Authentication Flow', () => {
  const supabase = createClient(
    process.env.SUPABASE_URL!,
    process.env.SUPABASE_ANON_KEY!
  )

  it('should register a new user', async () => {
    const { data, error } = await supabase.auth.signUp({
      email: 'test@example.com',
      password: 'Test123456',
    })
    expect(error).toBeNull()
    expect(data.user).toBeDefined()
  })

  it('should login with correct credentials', async () => {
    const { data, error } = await supabase.auth.signInWithPassword({
      email: 'test@example.com',
      password: 'Test123456',
    })
    expect(error).toBeNull()
    expect(data.session).toBeDefined()
  })

  it('should reject invalid credentials', async () => {
    const { error } = await supabase.auth.signInWithPassword({
      email: 'test@example.com',
      password: 'wrongpassword',
    })
    expect(error).toBeDefined()
  })
})
```

### E2E测试（使用Playwright）

```typescript
import { test, expect } from '@playwright/test'

test('complete user journey', async ({ page }) => {
  // 1. 注册
  await page.goto('/auth')
  await page.fill('[placeholder="邮箱"]', 'e2e@example.com')
  await page.fill('[placeholder="密码"]', 'E2eTest123')
  await page.click('button:has-text("注册")')
  await expect(page.locator('text=注册成功')).toBeVisible()

  // 2. 登录
  await page.fill('[placeholder="邮箱"]', 'e2e@example.com')
  await page.fill('[placeholder="密码"]', 'E2eTest123')
  await page.click('button:has-text("登录")')
  await expect(page).toHaveURL('/')

  // 3. 创建期间
  await page.click('text=创建期间')
  await page.fill('[name="year_month"]', '2024-01')
  await page.click('button:has-text("确定")')
  await expect(page.locator('text=2024-01')).toBeVisible()
})
```

---

## 🔍 验证检查点

### 迁移后验证

#### 1. 数据完整性验证
```sql
-- 检查用户数量
SELECT COUNT(*) FROM auth.users;
SELECT COUNT(*) FROM public.profiles;

-- 检查数据关联
SELECT 
  u.email,
  p.username,
  COUNT(per.id) as period_count
FROM auth.users u
LEFT JOIN public.profiles p ON p.id = u.id
LEFT JOIN public.periods per ON per.user_id = u.id
GROUP BY u.email, p.username;
```

#### 2. 功能验证清单
- [ ] 所有用户都能登录
- [ ] 所有用户的历史数据都可访问
- [ ] 文件上传功能正常
- [ ] 数据处理功能正常
- [ ] Excel导出功能正常
- [ ] 审计日志正常记录

#### 3. 性能验证
- [ ] 登录时间 < 2秒
- [ ] 页面加载时间 < 3秒
- [ ] API响应时间 < 500ms（简单查询）
- [ ] API响应时间 < 2秒（复杂处理）

---

## ⚠️ 风险和注意事项

### 关键风险

1. **用户ID类型变更（uint → uuid）**
   - **风险**：所有外键关系需要更新
   - **缓解**：创建ID映射表，分步迁移

2. **认证系统切换**
   - **风险**：现有用户无法登录
   - **缓解**：强制所有用户重置密码

3. **数据丢失**
   - **风险**：迁移过程中数据损坏
   - **缓解**：多次备份，测试环境先行

4. **RLS配置错误**
   - **风险**：用户可访问他人数据
   - **缓解**：严格测试RLS策略

### 