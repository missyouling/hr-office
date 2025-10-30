# Supabase Dashboard 配置指南

本文档详细说明如何在 Supabase Dashboard 中获取项目配置信息。

## 📍 访问 Supabase Dashboard

1. 打开浏览器访问：https://supabase.com
2. 登录你的账户
3. 选择项目：`cdkoffice`

## 🔑 获取 API 配置信息

### 步骤 1: 进入 Settings 页面

在项目主页，点击左侧边栏底部的 **⚙️ Settings（设置）**

### 步骤 2: 进入 API 配置页面

在 Settings 页面，点击左侧的 **API** 选项

你会看到以下配置区域：

---

## 📋 配置信息位置

### 1. Project URL（项目 URL）

**位置**: API Settings 页面顶部的 "Configuration" 部分

```
Project URL
https://vjpvrzphtnxawkqwuogn.supabase.co
```

**说明**: 这是你的 Supabase 项目的唯一 URL，所有 API 请求都发送到这个地址。

---

### 2. API Keys（API 密钥）

**位置**: "Project API keys" 部分

#### anon public key（匿名公钥）

```
anon public
eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
```

- ✅ **可以在前端使用**
- ✅ **可以公开**
- 用于客户端应用（浏览器、移动应用）
- 受行级安全策略（RLS）保护

#### service_role key（服务角色密钥）

```
service_role
eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
[!] This key has the ability to bypass Row Level Security. Never share it publicly.
```

- ⚠️ **只能在后端使用**
- ⚠️ **绝不能公开**
- ⚠️ **可以绕过 RLS 策略**
- 点击右侧的 **"Reveal"** 按钮显示完整密钥
- 用于服务端操作（管理员操作、批量数据处理）

---

### 3. JWT Secret（JWT 密钥）

**位置**: 页面下方的 "JWT Settings" 部分

```
JWT Secret
your-super-secret-jwt-token-with-at-least-32-characters-long
```

- 点击右侧的 **"Reveal"** 按钮显示
- 用于后端验证 Supabase 签发的 JWT token
- ⚠️ **绝不能公开**

---

## 📝 如何复制配置信息

### 方法 1: 手动复制

1. 找到对应的配置项
2. 点击右侧的 **"Copy"** 按钮（📋 图标）
3. 密钥默认隐藏，需要先点击 **"Reveal"** 显示后再复制

### 方法 2: 使用代码片段

在 API Settings 页面底部，有各种语言的示例代码：

- **JavaScript/TypeScript**
- **Dart**
- **Python**
- **Go**

可以直接复制这些代码片段，里面包含了 URL 和密钥。

---

## 🔧 配置文件更新

### 前端配置（frontend/.env.local）

```env
# Supabase 配置
NEXT_PUBLIC_SUPABASE_URL=https://vjpvrzphtnxawkqwuogn.supabase.co
NEXT_PUBLIC_SUPABASE_ANON_KEY=你的anon_key

# BFF (Go 后端) 配置
NEXT_PUBLIC_BFF_URL=http://localhost:8080/api
```

### 后端配置（backend/.env）

```env
# Supabase 配置
SUPABASE_URL=https://vjpvrzphtnxawkqwuogn.supabase.co
SUPABASE_SERVICE_KEY=你的service_role_key
SUPABASE_JWT_SECRET=你的jwt_secret

# BFF 服务器配置
BFF_PORT=8080
ALLOWED_ORIGINS=http://localhost:3000,http://localhost:10086
```

---

## 🔍 常见问题

### Q1: 找不到 "Reveal" 按钮？

**A**: service_role key 和 JWT Secret 默认是隐藏的，在密钥右侧有一个眼睛图标（👁️）或"Reveal"按钮，点击即可显示。

### Q2: 复制的密钥很长是正常的吗？

**A**: 是的！JWT token 通常有几百个字符长，这是正常的。完整的密钥格式类似：

```
eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpc3MiOiJzdXBhYmFzZSIsInJlZiI6InZqcHZyenBodG54YXdrcXd1b2duIiwicm9sZSI6ImFub24iLCJpYXQiOjE3NjE3ODQyMTAsImV4cCI6MjA3NzM2MDIxMH0.lyrwgC2eisDLmkAqIE6-YfFHgUgt2f4eLUT04U7Sphc
```

### Q3: service_role key 和 anon key 有什么区别？

**A**: 
- **anon key**: 用于前端，受 RLS 策略限制，安全
- **service_role key**: 用于后端，可以绕过 RLS，功能强大但需要严格保密

### Q4: 我把 service_role key 暴露了怎么办？

**A**: 立即在 Supabase Dashboard → Settings → API 中点击 "Reset API keys" 重置密钥。

---

## 🎯 配置检查清单

在继续之前，请确认你已经：

- [ ] 找到了 Project URL
- [ ] 复制了 anon public key
- [ ] 点击 "Reveal" 并复制了 service_role key
- [ ] 点击 "Reveal" 并复制了 JWT Secret
- [ ] 更新了 `frontend/.env.local`
- [ ] 更新了 `backend/.env`
- [ ] ⚠️ 确认 service_role key 和 JWT Secret 没有提交到 Git

---

## 📸 页面布局参考

```
┌─────────────────────────────────────────────────────────┐
│  Supabase Dashboard - cdkoffice                        │
├─────────────────────────────────────────────────────────┤
│  Settings > API                                         │
│                                                          │
│  Configuration                                          │
│  ┌────────────────────────────────────────────────┐   │
│  │ Project URL                                     │   │
│  │ https://vjpvrzphtnxawkqwuogn.supabase.co [Copy]│   │
│  └────────────────────────────────────────────────┘   │
│                                                          │
│  Project API keys                                       │
│  ┌────────────────────────────────────────────────┐   │
│  │ anon public                                     │   │
│  │ eyJhbGc... [Copy]                              │   │
│  │                                                 │   │
│  │ service_role [!] Can bypass RLS                │   │
│  │ *************** [Reveal] [Copy]                │   │
│  └────────────────────────────────────────────────┘   │
│                                                          │
│  JWT Settings                                           │
│  ┌────────────────────────────────────────────────┐   │
│  │ JWT Secret                                      │   │
│  │ *************** [Reveal] [Copy]                │   │
│  └────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────┘
```

---

## 🔐 安全提示

1. **永远不要**将 service_role key 和 JWT Secret 提交到 Git
2. **永远不要**在前端代码中使用 service_role key
3. **永远不要**在公开的文档或 Issue 中分享这些密钥
4. 使用 `.env.local` 和 `.env` 文件（已在 .gitignore 中）
5. 如果密钥泄露，立即重置

---

## 📚 相关文档

- [Supabase API 文档](https://supabase.com/docs/guides/api)
- [Supabase 安全最佳实践](https://supabase.com/docs/guides/auth/auth-deep-dive/auth-deep-dive-jwts)
- [项目迁移检查清单](./SUPABASE_MIGRATION_CHECKLIST.md)

---

**版本**: v1.0.0  
**最后更新**: 2025-10-30