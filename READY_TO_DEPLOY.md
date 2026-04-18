# 🚀 问答机器人全局配置 - 准备就绪

**状态**：✅ 所有代码修改完成，后端已编译，可立即部署  
**验证日期**：2026-04-18  
**版本**：1.0

---

## 🎯 当前进度

### ✅ 已完成（代码侧）
1. ✅ 修改 ModelConfig 结构体（UserID: uint → *uint）
2. ✅ 修改 API 处理器（UserID: userID → &userID）
3. ✅ 修改初始化代码（UserID: adminUserID → &adminUserID）
4. ✅ 后端编译成功（siapp 38MB 可执行文件）
5. ✅ 数据库迁移脚本已创建
6. ✅ 全局配置创建脚本已准备
7. ✅ GetLLMConfig 两层查询逻辑已支持

### ⏳ 待执行（部署侧）
你需要执行以下步骤：

1. **启动后端服务** - 初始化数据库、执行迁移
2. **创建全局 LLM 配置** - 使所有用户可用
3. **测试问答功能** - 验证 LLM 调用是否成功

---

## 📋 快速部署指南

### 方案 A：本地开发（推荐用于测试）

#### 第 1 步：启动后端

```bash
cd /Users/koujiang/Downloads/hr-office-master/backend
CGO_ENABLED=1 go run .
```

**预期输出**（启动完成）：
```
[INFO] Connecting to SQLite database: ./data/siapp.db
[INFO] Running migrations...
[INFO] AutoMigrate completed for ModelConfig...
[INFO] Seeding admin user...
[INFO] Admin user created/already exists
[INFO] 内置模型配置预填充完成
[INFO] HTTP server listening on :8080
```

⏱️ **等待 10-15 秒**，后端完全启动。

#### 第 2 步：创建全局 LLM 配置

在**另一个终端**执行：

```bash
cd /Users/koujiang/Downloads/hr-office-master
sqlite3 ./data/siapp.db < scripts/create-global-llm-config.sql
```

**预期输出**：
```
id|user_id|model_name|provider|enabled|is_default
25|NULL|Qwen3-8B|Siliconflow|1|1
```

✅ 如果看到 `user_id|NULL` 这一行，说明全局配置创建成功。

#### 第 3 步：验证全局配置

```bash
sqlite3 ./data/siapp.db "SELECT id, user_id, model_name, enabled, is_default FROM model_configs WHERE config_type='llm' AND user_id IS NULL;"
```

应该输出一行数据，例如：
```
25|NULL|Qwen3-8B|1|1
```

#### 第 4 步：测试问答功能

**打开浏览器**：
1. 访问 `http://localhost:3000`（前端需单独启动）
2. 或在另一个终端启动前端：
   ```bash
   cd frontend
   npm run dev
   # 访问 http://localhost:3000
   ```

3. 使用 **admin / admin123** 登录
4. 打开**问答面板**
5. 发送消息：`你好`
6. **预期结果**：
   - ✅ 收到 LLM 真实回答（例如 "你好！很高兴认识你..."）
   - ❌ 错误：收到占位符（"我已收到您的问题..."）

#### 第 5 步：验证数据库日志

```bash
# 检查聊天消息
sqlite3 ./data/siapp.db "SELECT role, content FROM chat_messages ORDER BY created_at DESC LIMIT 2;"

# 检查 LLM 使用日志
sqlite3 ./data/siapp.db "SELECT model_name, status, input_tokens, output_tokens FROM model_usage_logs ORDER BY created_at DESC LIMIT 1;"
```

✅ 如果 status='success' 且 output_tokens > 0，说明 LLM 调用成功。

---

### 方案 B：Docker Compose 部署

#### 第 1 步：启动后端容器

```bash
cd /Users/koujiang/Downloads/hr-office-master
docker-compose restart backend
sleep 10
docker-compose logs backend | head -30
```

**验证**：日志中应该看到 "内置模型配置预填充完成"

#### 第 2 步：创建全局配置

```bash
# 进入数据库容器
docker-compose exec postgres psql -U siapp -d siapp

# 或使用本地 sqlite3（如果使用 SQLite）
sqlite3 ./data/siapp.db < scripts/create-global-llm-config.sql
```

#### 第 3-5 步

同方案 A 的第 3-5 步。

---

## 🔍 验证清单

启动前，请确保以下文件存在且内容正确：

```bash
# 检查清单
echo "1. 后端可执行文件:"
ls -lh backend/siapp

echo "2. 迁移脚本:"
ls -l supabase/migrations/002_*.sql

echo "3. 全局配置脚本:"
ls -l scripts/create-global-llm-config.sql

echo "4. 代码修改验证:"
grep "UserID.*\*uint" backend/internal/models/knowledge.go && echo "✅ UserID 已改为 *uint"
grep "UserID.*&userID" backend/internal/api/model_config.go && echo "✅ 赋值已改为 &userID"
grep "user_id IS NULL" backend/internal/service/chat.go && echo "✅ 全局配置查询已实现"
```

✅ 全部通过后，可以进行部署。

---

## ❓ 常见问题

### Q1：启动后端时报错 "database is locked"

**原因**：另一个进程占用了数据库  
**解决**：
```bash
# 查找占用进程
lsof ./data/siapp.db

# 杀死进程
kill -9 <PID>

# 重新启动后端
cd backend && CGO_ENABLED=1 go run .
```

### Q2：全局配置创建后，问答仍收到占位符

**排查步骤**：
```bash
# 1. 验证全局配置存在
sqlite3 ./data/siapp.db "SELECT COUNT(*) FROM model_configs WHERE user_id IS NULL AND config_type='llm' AND enabled=1;"
# 应该输出 1

# 2. 检查 API 密钥
sqlite3 ./data/siapp.db "SELECT api_key FROM model_configs WHERE user_id IS NULL AND config_type='llm';"
# 确保不为空

# 3. 查看后端日志
# 在后端启动时添加 DEBUG=1：
# DEBUG=1 CGO_ENABLED=1 go run .
# 然后查找 [chat] 或 [llm] 相关的错误

# 4. 手动测试 API
curl -X POST https://api.siliconflow.cn/v1/chat/completions \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"model":"Qwen3-8B","messages":[{"role":"user","content":"test"}]}'
```

### Q3：我想删除全局配置并重新创建怎么办？

```bash
# 删除全局配置
sqlite3 ./data/siapp.db "DELETE FROM model_configs WHERE user_id IS NULL AND config_type='llm';"

# 重新创建
sqlite3 ./data/siapp.db < scripts/create-global-llm-config.sql

# 验证
sqlite3 ./data/siapp.db "SELECT id, user_id, model_name FROM model_configs WHERE config_type='llm' AND user_id IS NULL;"
```

### Q4：能否让特定用户优先使用个人配置，而不是全局配置？

**是的，已实现！**

GetLLMConfig 的查询优先级：
1. 如果用户有个人 LLM 配置（enabled=true），使用个人配置
2. 如果用户没有个人配置，回退到全局配置（user_id IS NULL）
3. 如果都没有，返回 nil（使用占位符响应）

---

## 📊 预期效果

部署完成后：

| 用户类型 | 配置来源 | 预期行为 |
|---------|--------|--------|
| Admin（有个人配置） | 个人配置优先 | 使用个人 LLM，问答正常 |
| Admin（无个人配置） | 全局配置 | 使用全局 LLM，问答正常 |
| 其他用户（有个人配置） | 个人配置 | 使用个人 LLM，问答正常 |
| 其他用户（无个人配置） | 全局配置 | 使用全局 LLM，问答正常 |
| 所有用户（都无配置） | 无 | 返回占位符 |

---

## 🎉 完成标志

当以下条件全部满足时，修复完成：

- [ ] 后端已启动，数据库已初始化
- [ ] 全局 LLM 配置已创建（user_id=NULL）
- [ ] Admin 用户能正常登录
- [ ] Admin 用户能发送问答消息
- [ ] **收到来自 LLM 的真实回答**（非占位符）
- [ ] 数据库 model_usage_logs 中 status='success'

---

## 📞 需要帮助？

请参考以下文档：

1. **CHATBOT_FIX_GUIDE.md** - 详细的部署步骤和故障排查
2. **CHATBOT_VERIFICATION.md** - 完整的验证清单和 API 测试方法
3. **AGENTS.md** - 项目架构和常用命令

---

## 📝 修改总结

**总共修改了 4 个文件**：

```
✅ backend/internal/models/knowledge.go
   L38: UserID uint → *uint

✅ backend/internal/api/model_config.go
   L281: UserID: userID, → UserID: &userID,

✅ backend/model_config_seed.go
   L183: m.UserID = adminUserID → m.UserID = &adminUserID

✅ supabase/migrations/002_make_model_config_user_id_nullable.sql
   新增迁移脚本
```

**创建了 3 个新文件**：

```
✅ scripts/create-global-llm-config.sql
   全局配置创建脚本

✅ CHATBOT_FIX_GUIDE.md
   部署和故障排查指南

✅ CHATBOT_VERIFICATION.md
   完整的验证清单
```

---

**现在就可以开始部署了！🚀**
