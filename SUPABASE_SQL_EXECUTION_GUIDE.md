# Supabase 数据库迁移执行指南

本指南详细说明如何在 Supabase Dashboard 中执行数据库迁移 SQL 脚本。

## 📋 准备工作

在执行 SQL 之前，请确认：

- ✅ 已登录 Supabase Dashboard
- ✅ 已选择项目：`cdkoffice`
- ✅ 已完成环境变量配置
- ✅ 已备份现有数据（如果有）

## 🚀 执行步骤

### 步骤 1: 打开 SQL Editor

1. 登录 Supabase Dashboard: https://supabase.com
2. 选择项目：`cdkoffice`
3. 在左侧菜单中，点击 **🛠️ SQL Editor**

### 步骤 2: 创建新查询

1. 点击页面右上角的 **"New query"** 按钮
2. 或者使用快捷键：`Ctrl/Cmd + Enter`

### 步骤 3: 复制 SQL 脚本

打开项目中的文件：
```
supabase/migrations/001_initial_schema.sql
```

**方式 1: 在 VS Code 中复制**
1. 打开文件 `supabase/migrations/001_initial_schema.sql`
2. 按 `Ctrl/Cmd + A` 全选
3. 按 `Ctrl/Cmd + C` 复制

**方式 2: 使用命令行**
```bash
# Windows (复制到剪贴板)
type supabase\migrations\001_initial_schema.sql | clip

# Linux/Mac (显示内容)
cat supabase/migrations/001_initial_schema.sql
```

### 步骤 4: 粘贴并执行

1. 将复制的 SQL 脚本粘贴到 SQL Editor 中
2. （可选）为查询命名，例如：`001_initial_schema`
3. 点击右下角的 **"Run"** 按钮或按 `Ctrl/Cmd + Enter`

### 步骤 5: 检查执行结果

**成功标志：**
- ✅ 底部显示绿色提示：`Success. No rows returned`
- ✅ 或显示执行的行数统计
- ✅ 没有红色错误信息

**如果出现错误：**
- ❌ 检查是否有语法错误
- ❌ 检查是否已存在同名表（重复执行）
- ❌ 查看错误信息并根据提示修复

### 步骤 6: 验证表结构

1. 在左侧菜单点击 **📊 Table Editor**
2. 你应该能看到以下新创建的表：
   - `profiles` - 用户扩展信息
   - `periods` - 社保期间
   - `source_files` - 源文件
   - `raw_records` - 原始记录
   - `period_summaries` - 期间汇总
   - `personal_charges` - 个人费用
   - `unit_charges` - 单位费用
   - `roster_entries` - 花名册
   - `audit_logs` - 审计日志

## 📊 SQL 脚本包含的内容

### 创建的表（9个）

| 序号 | 表名 | 说明 | RLS |
|------|------|------|-----|
| 1 | `profiles` | 用户扩展信息 | ✅ |
| 2 | `periods` | 社保期间 | ✅ |
| 3 | `source_files` | 源文件记录 | ✅ |
| 4 | `raw_records` | 原始数据记录 | ✅ |
| 5 | `period_summaries` | 期间汇总统计 | ✅ |
| 6 | `personal_charges` | 个人费用明细 | ✅ |
| 7 | `unit_charges` | 单位费用明细 | ✅ |
| 8 | `roster_entries` | 花名册 | ✅ |
| 9 | `audit_logs` | 审计日志 | ✅ |

### RLS 策略（行级安全）

每个表都配置了 RLS 策略，确保：
- ✅ 用户只能访问自己的数据
- ✅ 多租户数据完全隔离
- ✅ 基于 `auth.uid()` 的权限控制

### 触发器（Triggers）

1. **自动更新 `updated_at`**
   - 所有表在更新时自动更新时间戳
   - 使用函数：`handle_updated_at()`

2. **自动创建用户 Profile**
   - 新用户注册时自动创建 `profiles` 记录
   - 使用函数：`handle_new_user()`
   - 从 `auth.users` 的 `raw_user_meta_data` 提取信息

### 辅助函数（Functions）

1. **`get_user_periods(user_uuid)`**
   - 获取用户的所有期间
   - 返回：id, year_month, status, created_at
   - 按时间倒序排列

2. **`get_period_stats(period_id)`**
   - 获取期间的统计信息
   - 返回 JSON：文件数、记录数、费用数等

## 🔍 验证迁移成功

### 方法 1: 在 Table Editor 中检查

1. 点击 **Table Editor**
2. 查看所有表是否已创建
3. 点击任意表，查看列结构

### 方法 2: 在 SQL Editor 中查询

执行以下 SQL 查询：

```sql
-- 查看所有公共表
SELECT table_name 
FROM information_schema.tables 
WHERE table_schema = 'public'
ORDER BY table_name;

-- 查看 RLS 是否启用
SELECT tablename, rowsecurity 
FROM pg_tables 
WHERE schemaname = 'public';

-- 查看所有触发器
SELECT trigger_name, event_object_table 
FROM information_schema.triggers 
WHERE trigger_schema = 'public';

-- 查看所有函数
SELECT routine_name, routine_type 
FROM information_schema.routines 
WHERE routine_schema = 'public';
```

### 方法 3: 测试创建用户

在 **Authentication** 页面创建一个测试用户，然后检查 `profiles` 表是否自动创建了记录。

## ⚠️ 常见问题

### Q1: 执行时提示"relation already exists"

**原因**: 表已经存在（可能之前执行过）

**解决方案**:
```sql
-- 选项1: 删除所有表（谨慎！）
DROP TABLE IF EXISTS public.audit_logs CASCADE;
DROP TABLE IF EXISTS public.roster_entries CASCADE;
DROP TABLE IF EXISTS public.unit_charges CASCADE;
DROP TABLE IF EXISTS public.personal_charges CASCADE;
DROP TABLE IF EXISTS public.period_summaries CASCADE;
DROP TABLE IF EXISTS public.raw_records CASCADE;
DROP TABLE IF EXISTS public.source_files CASCADE;
DROP TABLE IF EXISTS public.periods CASCADE;
DROP TABLE IF EXISTS public.profiles CASCADE;

-- 然后重新执行完整的迁移脚本
```

### Q2: 执行时提示权限错误

**原因**: 可能没有使用正确的 service_role 权限

**解决方案**: 
- 确保在 SQL Editor 中执行（自动使用 service_role 权限）
- 不要在客户端代码中执行此脚本

### Q3: RLS 策略导致无法插入数据

**原因**: RLS 策略阻止了数据插入

**临时解决方案**（仅用于调试）:
```sql
-- 临时禁用 RLS（开发测试用）
ALTER TABLE public.periods DISABLE ROW LEVEL SECURITY;

-- 完成后记得重新启用
ALTER TABLE public.periods ENABLE ROW LEVEL SECURITY;
```

### Q4: 触发器没有自动创建 profile

**检查方法**:
```sql
-- 查看触发器是否存在
SELECT * FROM pg_trigger WHERE tgname = 'on_auth_user_created';

-- 手动测试函数
SELECT public.handle_new_user();
```

## 📝 执行后的后续步骤

执行成功后，请按以下顺序继续：

1. ✅ **验证表结构** - 确认所有表已创建
2. ✅ **测试 RLS 策略** - 创建测试用户并验证权限
3. ✅ **测试前端连接** - 启动前端开发服务器
4. ✅ **测试后端连接** - 启动后端开发服务器
5. ✅ **运行端到端测试** - 测试完整的用户注册和登录流程

## 🎉 完成标志

当你看到以下情况时，表示迁移成功：

- ✅ SQL Editor 显示执行成功
- ✅ Table Editor 中可以看到所有9个表
- ✅ 每个表都显示有 RLS 启用（🔒 图标）
- ✅ 触发器和函数已创建
- ✅ 可以在前端成功注册新用户

---

**下一步**: 阅读 [SUPABASE_MIGRATION_CHECKLIST.md](./SUPABASE_MIGRATION_CHECKLIST.md) 继续完成迁移的其他步骤。

**版本**: v1.0.0  
**最后更新**: 2025-10-30