# Supabase 后端集成说明

## 📋 概述

本文档说明Go后端如何集成Supabase认证系统，实现JWT验证和用户身份管理。

## 🔧 实现细节

### 1. 新增文件

#### `backend/internal/supabase/client.go`
- Supabase Admin客户端配置
- 用于后端管理操作（如果需要）

#### `backend/internal/supabase/jwt.go`
- Supabase JWT验证逻辑
- JWKS获取和缓存
- RSA公钥验证
- JWT中间件实现

#### `backend/.env.supabase.example`
- 环境变量配置示例
- 包含所有必需的Supabase配置项

### 2. 修改文件

#### `backend/main.go`
- 导入`siapp/internal/supabase`包
- 将`auth.JWTMiddleware`替换为`supabase.SupabaseJWTMiddleware()`
- 保持其他逻辑不变

#### `backend/internal/middleware/audit.go`
- 支持从Supabase JWT上下文提取用户信息
- 向后兼容旧的JWT认证系统
- 在审计日志中记录Supabase用户ID和邮箱

## 🔐 认证流程

### 前端 → Supabase → 后端

```
1. 用户在前端登录
   ↓
2. Supabase Auth 验证用户
   ↓
3. 返回 JWT token 给前端
   ↓
4. 前端发送请求，携带 Authorization: Bearer <token>
   ↓
5. 后端中间件验证 Supabase JWT
   ↓
6. 从 JWKS 获取公钥验证签名
   ↓
7. 提取用户信息存入请求上下文
   ↓
8. 业务逻辑处理
```

## 📦 环境变量配置

### 必需变量

```bash
# Supabase项目URL
SUPABASE_URL=https://your-project.supabase.co

# Supabase Service Role Key（仅后端使用）
SUPABASE_SERVICE_ROLE_KEY=your-service-role-key

# 数据库配置（如果使用Supabase数据库）
SIAPP_DATABASE_TYPE=postgres
SIAPP_DB_HOST=db.your-project.supabase.co
SIAPP_DB_PORT=5432
SIAPP_DB_USER=postgres
SIAPP_DB_PASSWORD=your-db-password
SIAPP_DB_NAME=postgres
SIAPP_DB_SSLMODE=require

# 服务器配置
SIAPP_ADDR=0.0.0.0:8080
ALLOWED_ORIGINS=http://localhost:3000
```

## 🚀 启动步骤

### 1. 配置环境变量

```bash
cd backend
cp .env.supabase.example .env
# 编辑 .env 填入你的Supabase项目信息
```

### 2. 启动后端服务

**Windows (需要GCC编译器):**
```cmd
set CGO_ENABLED=1
go run .
```

**Linux/Mac:**
```bash
CGO_ENABLED=1 go run .
```

**使用Docker (推荐):**
```bash
docker-compose up -d
```

## 🧪 测试验证

### 1. 健康检查

```bash
curl http://localhost:8080/health
```

预期响应：
```json
{
  "status": "healthy",
  "timestamp": "2024-01-15T10:30:00Z"
}
```

### 2. 测试受保护的API

```bash
# 获取前端登录后的token
TOKEN="your-supabase-jwt-token"

# 测试API调用
curl -H "Authorization: Bearer $TOKEN" \
     http://localhost:8080/api/periods
```

## 📝 关键技术点

### JWT验证流程

1. **提取Token**: 从`Authorization: Bearer <token>`头中提取JWT
2. **获取JWKS**: 从Supabase获取公钥集合（带1小时缓存）
3. **解析Token**: 获取`kid`（Key ID）
4. **匹配公钥**: 从JWKS中找到对应的RSA公钥
5. **验证签名**: 使用RSA公钥验证JWT签名
6. **提取Claims**: 获取用户ID、邮箱、角色等信息
7. **存入上下文**: 将用户信息存入请求上下文供后续使用

### 上下文键值

```go
supabase.UserIDKey    // string - Supabase UUID
supabase.UserEmailKey // string - 用户邮箱
supabase.UserRoleKey  // string - 用户角色
```

### 审计日志兼容

审计中间件同时支持：
- **Supabase JWT**: 提取UUID和邮箱，记录在`custom`字段
- **旧JWT系统**: 提取userID和username（向后兼容）

## ⚠️ 重要注意事项

### 安全性

1. **Service Role Key保密**: 
   - Service Role Key拥有完全权限
   - 仅在后端服务器使用
   - 不要提交到版本控制系统
   - 不要暴露给前端

2. **HTTPS要求**:
   - 生产环境必须使用HTTPS
   - JWT token通过Authorization头传输
   - 避免token泄露

3. **JWKS缓存**:
   - 公钥缓存1小时
   - 减少对Supabase的请求
   - 自动刷新机制

### 数据库迁移

当前后端仍然保留了旧的用户表结构：
- `users`表（GORM管理）
- `password_reset_tokens`表
- `email_verification_tokens`表

**未来可以考虑**：
- 完全移除旧认证表
- 仅使用Supabase Auth
- 保留业务数据表（periods, charges等）

### 兼容性

当前实现向后兼容：
- 支持Supabase JWT验证
- 保留旧JWT代码（未使用）
- 审计日志兼容两种认证方式

## 🔄 迁移路径

### 阶段4: 后端改造 ✅
- [x] 创建Supabase客户端配置
- [x] 实现JWT验证中间件
- [x] 更新main.go集成中间件
- [x] 修改审计中间件兼容Supabase

### 阶段5: 数据迁移 (下一步)
- [ ] 从旧用户表迁移到Supabase Auth
- [ ] 数据一致性验证
- [ ] 回滚方案准备

### 阶段6: 测试验证
- [ ] 单元测试
- [ ] 集成测试
- [ ] 端到端测试
- [ ] 性能测试

### 阶段7: 生产部署
- [ ] 环境变量配置
- [ ] Docker镜像构建
- [ ] 健康检查配置
- [ ] 监控告警设置

## 📚 参考资料

- [Supabase Auth文档](https://supabase.com/docs/guides/auth)
- [JWT.io](https://jwt.io/)
- [Go JWT库文档](https://github.com/golang-jwt/jwt)
- [Supabase Go客户端](https://github.com/supabase-community/supabase-go)

## 🐛 故障排查

### 问题1: "SUPABASE_URL environment variable is required"

**解决方案**: 确保`.env`文件存在并包含正确的SUPABASE_URL

### 问题2: "Invalid token"

**可能原因**:
- Token已过期
- JWKS公钥获取失败
- Token签名无效

**排查步骤**:
1. 检查token是否过期（使用jwt.io解码查看）
2. 确认SUPABASE_URL可访问
3. 检查网络是否能访问`${SUPABASE_URL}/auth/v1/jwks`

### 问题3: 编译失败 "undefined: supabase"

**解决方案**: 
```bash
cd backend
go mod tidy
go build
```

## 📊 性能考虑

- **JWKS缓存**: 减少90%以上的公钥获取请求
- **无数据库查询**: JWT验证不需要查询用户表
- **并发安全**: 支持高并发请求处理
- **内存占用**: 最小化内存使用

## 🎯 后续优化

1. **移除旧认证代码**: 当完全迁移后，删除旧的JWT和认证逻辑
2. **添加rate limiting**: 防止JWT验证滥用
3. **监控指标**: 添加JWT验证成功/失败的监控
4. **日志优化**: 详细记录认证失败原因

---

**文档版本**: v1.0  
**最后更新**: 2025-01-15  
**维护者**: Development Team