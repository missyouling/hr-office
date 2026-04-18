# 问答机器人全局配置修复指南

## 问题背景
当前问答机器人仅能访问"用户个人配置"的 LLM，其他用户无法使用 LLM 功能。需要支持"全局 LLM 配置"，使所有系统用户都能访问同一套 LLM 配置。

## 修改内容

### 1. 数据库结构调整
- **修改文件**：`supabase/migrations/002_make_model_config_user_id_nullable.sql`
- **变更**：`model_configs.user_id` 从非空 (`uint`) 改为可空 (`*uint`)
- **含义**：NULL 值表示全局配置，非 NULL 表示用户个人配置

### 2. 后端代码修改
- **修改文件**：
  - `backend/internal/models/knowledge.go`：ModelConfig 结构体的 UserID 字段改为 `*uint`
  - `backend/internal/api/model_config.go`：CreateModelConfig 处理器中 UserID 赋值改为取地址 `&userID`
  - `backend/model_config_seed.go`：初始化代码中 UserID 赋值改为取地址 `&adminUserID`

- **已实现逻辑**（无需修改）：
  - `backend/internal/service/chat.go` 中 GetLLMConfig 已支持两层查询：
    1. 优先查询当前用户的 LLM 配置（user_id = ?）
    2. 若用户无配置，回退到查询全局配置（user_id IS NULL）

## 部署步骤

### 步骤 1：应用代码变更

确保你的代码已包含以下修改：
```bash
cd /path/to/hr-office
git status
# 应该看到以下文件已修改：
# - backend/internal/models/knowledge.go
# - backend/internal/api/model_config.go
# - backend/model_config_seed.go
# - supabase/migrations/002_make_model_config_user_id_nullable.sql
```

### 步骤 2：重新编译后端

```bash
cd backend
go build -o siapp .
# 如果编译成功，输出为空；若有错误，请检查上述文件是否正确修改
```

### 步骤 3：启动后端（重要：必须启动后端来执行迁移！）

根据你的部署方式选择：

#### 选项 A：本地直接运行（用于开发/测试）
```bash
cd backend
CGO_ENABLED=1 go run .
# 后端启动时会自动：
# 1. 执行 supabase/migrations 中的所有 SQL 迁移
# 2. 修改 model_configs 表结构（user_id 改为可空）
# 3. 初始化内置 LLM 模型配置
```

#### 选项 B：Docker Compose（用于容器化部署）
```bash
docker-compose -f docker-compose.yml up -d backend
# 或生产环境：
docker-compose -f docker-compose.production.yml up -d
```

启动完成后，查看日志验证迁移是否成功：
```bash
docker-compose logs backend | grep -i "model\|migration"
# 应看到：Creating 内置模型配置预填充完成
```

### 步骤 4：创建全局 LLM 配置

启动后端后，数据库结构已修改完毕。现在需要创建全局配置。有两种方式：

#### 方式 A：通过 SQL 直接创建（推荐）

执行下面的 SQL 语句。**重要**：根据你的数据库类型选择对应的命令。

**本地开发（SQLite）：**
```bash
sqlite3 ./data/siapp.db < scripts/create-global-llm-config.sql
```

**或使用 Go 提供的 SQL 工具（通用）：**
```bash
cd backend
go run . migrate < ../scripts/create-global-llm-config.sql
```

**或 PostgreSQL（如使用 Supabase）：**
```bash
psql -h your-db-host -U your-user -d your-db -f scripts/create-global-llm-config.sql
```

#### 方式 B：通过前端 UI 创建（可视化）

1. 登录系统（用 admin 账户）
2. 进入 **系统设置** → **模型配置**
3. 点击 **新增配置**
4. 填写信息：
   - **配置类型**：LLM 大语言模型
   - **服务商**：Siliconflow（或现有配置相同）
   - **模型名称**：Qwen3-8B（或现有配置相同）
   - **API 密钥**：粘贴现有 API 密钥
   - **API 端点**：https://api.siliconflow.cn/v1/chat/completions
   - **启用**：✅ 勾选
   - **设为默认**：✅ 勾选
   - **用户选择框**：**留空或选择"全局"**（表示 user_id=NULL）
5. 点击 **保存**

### 步骤 5：验证全局配置

#### 查看数据库中的配置

**SQLite：**
```bash
sqlite3 ./data/siapp.db "SELECT id, user_id, model_name, provider, enabled, is_default FROM model_configs WHERE config_type = 'llm';"
# 应该看到两行数据：
# 1. user_id 非空的行（用户个人配置）
# 2. user_id 为 NULL 的行（全局配置）
```

**PostgreSQL：**
```sql
SELECT id, user_id, model_name, provider, enabled, is_default 
FROM model_configs 
WHERE config_type = 'llm' 
ORDER BY user_id;
```

#### 查看后端日志

启动后端时查看是否有错误：
```bash
# 本地开发
cd backend && go run . 2>&1 | grep -i "error\|failed"

# Docker 容器
docker-compose logs backend | grep -i "error\|failed"
```

### 步骤 6：重启后端服务

如果是容器部署：
```bash
docker-compose restart backend
# 等待 30 秒让服务完全启动
sleep 30
docker-compose ps | grep backend
```

如果是本地开发，重启 Go 进程：
```bash
# Ctrl+C 停止当前进程
# 然后重新启动
cd backend && CGO_ENABLED=1 go run .
```

### 步骤 7：测试问答机器人

1. 打开浏览器，访问系统
2. **用 admin 账户登录**
3. 打开 **问答面板**（通常在侧边栏或顶部菜单）
4. 发送一条测试消息：`你好`
5. **预期结果**：收到来自 LLM 的真实回答（例如 "你好！很高兴认识你..."）
   - ✅ 正确：LLM 真实响应
   - ❌ 错误：收到占位符响应（"我已收到您的问题。根据知识库内容..."）

## 故障排查

### 问题 1：后端启动时数据库迁移失败

**症状**：后端日志显示 SQL 错误或 "ALTER TABLE" 失败

**排查步骤**：
1. 确认 model_configs 表存在
2. 检查 user_id 列当前是否为非空约束
3. 如果是 SQLite，可能没有约束，直接跳过迁移
4. 如果是 PostgreSQL，检查约束：
   ```sql
   SELECT constraint_name, constraint_type 
   FROM information_schema.table_constraints 
   WHERE table_name = 'model_configs';
   ```

### 问题 2：创建全局配置时失败（约束冲突）

**症状**：SQL 执行报错 "duplicate key" 或 "unique constraint"

**原因**：model_configs 表可能有 (user_id, config_type, model_name) 的唯一约束

**解决**：
```sql
-- 禁用约束（PostgreSQL）
ALTER TABLE model_configs DISABLE TRIGGER ALL;

-- 或删除约束（如果可以）
ALTER TABLE model_configs DROP CONSTRAINT unique_config_name;

-- 插入全局配置
INSERT INTO model_configs (...) VALUES (...);

-- 重新启用约束
ALTER TABLE model_configs ENABLE TRIGGER ALL;
```

### 问题 3：Admin 用户问答仍然不工作

**症状**：发送消息后收到占位符响应而非 LLM 回答

**排查步骤**：
1. **检查全局配置是否存在**：
   ```bash
   sqlite3 ./data/siapp.db "SELECT * FROM model_configs WHERE config_type = 'llm' AND user_id IS NULL;"
   ```
   应该有至少一行数据，且 `enabled = 1`

2. **查看后端日志中是否有错误**：
   ```bash
   docker-compose logs backend | grep -i "llm\|chat"
   ```

3. **验证 API 密钥是否正确**：
   ```bash
   # 手动测试 Siliconflow API
   curl -X POST https://api.siliconflow.cn/v1/chat/completions \
     -H "Authorization: Bearer YOUR_API_KEY" \
     -H "Content-Type: application/json" \
     -d '{"model":"Qwen3-8B","messages":[{"role":"user","content":"你好"}]}'
   ```

4. **检查网络连通性**：
   - 后端所在的网络能否访问 `api.siliconflow.cn:443`？
   - 是否有防火墙或代理阻止？

5. **查看数据库中 chat_messages 和 model_usage_logs 表**：
   ```bash
   sqlite3 ./data/siapp.db "SELECT * FROM chat_messages ORDER BY created_at DESC LIMIT 5;"
   sqlite3 ./data/siapp.db "SELECT * FROM model_usage_logs ORDER BY created_at DESC LIMIT 5;"
   ```
   检查是否有记录，以及 status 字段的值（应为 "success" 或 "error"）

### 问题 4：其他用户登录后仍无法使用问答

**预期行为**：任何登录用户应该能使用全局 LLM 配置

**排查**：
1. 确保全局配置存在且启用
2. 检查后端的 GetLLMConfig 逻辑是否被正确调用
3. 如果用户有个人 LLM 配置，会优先使用个人配置；若个人配置无效或禁用，才回退到全局配置

## 验证检查清单

- [ ] 后端代码修改完成且编译成功
- [ ] 数据库迁移脚本已创建
- [ ] 后端已启动并执行了迁移
- [ ] 全局 LLM 配置已创建（user_id IS NULL）
- [ ] 后端已重启
- [ ] Admin 用户能发送问答消息
- [ ] Admin 用户收到来自 LLM 的真实响应（非占位符）
- [ ] 其他用户也能使用问答功能

## 相关文件清单

```
修改文件：
├── backend/internal/models/knowledge.go              ✅ UserID 改为 *uint
├── backend/internal/api/model_config.go              ✅ 赋值改为 &userID
├── backend/model_config_seed.go                      ✅ 赋值改为 &adminUserID
├── supabase/migrations/002_make_model_config_user_id_nullable.sql  ✅ 新增迁移脚本

辅助文件：
├── scripts/create-global-llm-config.sql              ✅ 新增全局配置创建脚本
└── CHATBOT_FIX_GUIDE.md                              ✅ 本文件

无需修改（已支持全局配置）：
└── backend/internal/service/chat.go (GetLLMConfig)  ✅ 两层查询逻辑已实现
```

## 后续建议

1. **权限管理**：当前全局配置对所有用户可见。后期可添加权限控制，限制哪些用户可访问 LLM。
2. **配置审计**：记录全局配置的创建者、修改时间，便于审计。
3. **成本控制**：监控全局 LLM 配置的用量，防止恶意刷用。
4. **多模型支持**：允许创建多个全局 LLM 配置（不同模型、不同提供商），让用户选择。
