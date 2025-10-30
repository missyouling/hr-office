
# Supabase 迁移实施指南

> 本文档是 `SUPABASE_MIGRATION_PLAN.md` 的补充，提供详细的实施步骤和代码示例

---

## 📋 目录

1. [前端改造详细步骤](#前端改造详细步骤)
2. [后端改造详细步骤](#后端改造详细步骤)
3. [认证流程实现](#认证流程实现)
4. [数据访问层重构](#数据访问层重构)
5. [测试策略](#测试策略)

---

## 前端改造详细步骤

### 步骤1：安装Supabase依赖

```bash
cd frontend
npm install @supabase/supabase-js @supabase/auth-helpers-nextjs
```

### 步骤2：创建Supabase客户端

创建 `frontend/lib/supabase/client.ts`：

```typescript
import { createBrowserClient } from '@supabase/ssr'

export function createClient() {
  return createBrowserClient(
    process.env.NEXT_PUBLIC_SUPABASE_URL!,
    process.env.NEXT_PUBLIC_SUPABASE_ANON_KEY!
  )
}
```

创建 `frontend/lib/supabase/server.ts`：

```typescript
import { createServerClient, type CookieOptions } from '@supabase/ssr'
import { cookies } from 'next/headers'

export async function createClient() {
  const cookieStore = await cookies()

  return createServerClient(
    process.env.NEXT_PUBLIC_SUPABASE_URL!,
    process.env.NEXT_PUBLIC_SUPABASE_ANON_KEY!,
    {
      cookies: {
        get(name: string) {
          return cookieStore.get(name)?.value
        },
        set(name: string, value: string, options: CookieOptions) {
          try {
            cookieStore.set({ name, value, ...options })
          } catch (error) {
            // 服务端组件中的set操作可能失败
          }
        },
        remove(name: string, options: CookieOptions) {
          try {
            cookieStore.set({ name, value: '', ...options })
          } catch (error) {
            // 服务端组件中的remove操作可能失败
          }
        },
      },
    }
  )
}
```

### 步骤3：创建认证Context

创建 `frontend/lib/auth-context.tsx`：

```typescript
'use client'

import { createContext, useContext, useEffect, useState } from 'react'
import { User, Session } from '@supabase/supabase-js'
import { createClient } from './supabase/client'

interface AuthContextType {
  user: User | null
  session: Session | null
  loading: boolean
  signIn: (email: string, password: string) => Promise<void>
  signUp: (email: string, password: string, metadata?: any) => Promise<void>
  signOut: () => Promise<void>
}

const AuthContext = createContext<AuthContextType | undefined>(undefined)

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const [user, setUser] = useState<User | null>(null)
  const [session, setSession] = useState<Session | null>(null)
  const [loading, setLoading] = useState(true)
  const supabase = createClient()

  useEffect(() => {
    // 获取初始会话
    supabase.auth.getSession().then(({ data: { session } }) => {
      setSession(session)
      setUser(session?.user ?? null)
      setLoading(false)
    })

    // 监听认证状态变化
    const {
      data: { subscription },
    } = supabase.auth.onAuthStateChange((_event, session) => {
      setSession(session)
      setUser(session?.user ?? null)
      setLoading(false)
    })

    return () => subscription.unsubscribe()
  }, [])

  const signIn = async (email: string, password: string) => {
    const { error } = await supabase.auth.signInWithPassword({
      email,
      password,
    })
    if (error) throw error
  }

  const signUp = async (email: string, password: string, metadata?: any) => {
    const { error } = await supabase.auth.signUp({
      email,
      password,
      options: {
        data: metadata,
      },
    })
    if (error) throw error
  }

  const signOut = async () => {
    const { error } = await supabase.auth.signOut()
    if (error) throw error
  }

  return (
    <AuthContext.Provider value={{ user, session, loading, signIn, signUp, signOut }}>
      {children}
    </AuthContext.Provider>
  )
}

export function useAuth() {
  const context = useContext(AuthContext)
  if (context === undefined) {
    throw new Error('useAuth must be used within an AuthProvider')
  }
  return context
}
```

### 步骤4：更新根布局

修改 `frontend/app/layout.tsx`：

```typescript
import { AuthProvider } from '@/lib/auth-context'

export default function RootLayout({
  children,
}: {
  children: React.ReactNode
}) {
  return (
    <html lang="zh-CN">
      <body>
        <AuthProvider>
          {children}
        </AuthProvider>
      </body>
    </html>
  )
}
```

### 步骤5：重构登录页面

修改 `frontend/app/auth/page.tsx`：

```typescript
'use client'

import { useState } from 'react'
import { useRouter } from 'next/navigation'
import { useAuth } from '@/lib/auth-context'
import { createClient } from '@/lib/supabase/client'

export default function AuthPage() {
  const [mode, setMode] = useState<'login' | 'register'>('login')
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [username, setUsername] = useState('')
  const [fullName, setFullName] = useState('')
  const [companyId, setCompanyId] = useState('')
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')
  
  const router = useRouter()
  const { signIn, signUp } = useAuth()
  const supabase = createClient()

  const handleLogin = async (e: React.FormEvent) => {
    e.preventDefault()
    setLoading(true)
    setError('')

    try {
      await signIn(email, password)
      router.push('/')
    } catch (err: any) {
      setError(err.message || '登录失败')
    } finally {
      setLoading(false)
    }
  }

  const handleRegister = async (e: React.FormEvent) => {
    e.preventDefault()
    setLoading(true)
    setError('')

    try {
      // 1. 使用Supabase Auth注册
      await signUp(email, password, {
        username,
        full_name: fullName,
        company_id: companyId,
      })

      // 2. 创建profile（通过数据库触发器或手动）
      const { data: { user } } = await supabase.auth.getUser()
      if (user) {
        const { error: profileError } = await supabase
          .from('profiles')
          .insert({
            id: user.id,
            username,
            full_name: fullName,
            company_id: companyId,
          })

        if (profileError) {
          console.error('创建profile失败:', profileError)
        }
      }

      alert('注册成功！请检查邮箱验证链接')
      setMode('login')
    } catch (err: any) {
      setError(err.message || '注册失败')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="min-h-screen flex items-center justify-center">
      <div className="max-w-md w-full space-y-8">
        <div>
          <h2 className="text-3xl font-bold text-center">
            {mode === 'login' ? '登录' : '注册'}
          </h2>
        </div>

        {error && (
          <div className="bg-red-50 text-red-600 p-3 rounded">
            {error}
          </div>
        )}

        <form onSubmit={mode === 'login' ? handleLogin : handleRegister} className="space-y-4">
          {mode === 'register' && (
            <>
              <input
                type="text"
                placeholder="用户名"
                value={username}
                onChange={(e) => setUsername(e.target.value)}
                required
                className="w-full px-3 py-2 border rounded"
              />
              <input
                type="text"
                placeholder="姓名"
                value={fullName}
                onChange={(e) => setFullName(e.target.value)}
                className="w-full px-3 py-2 border rounded"
              />
              <select
                value={companyId}
                onChange={(e) => setCompanyId(e.target.value)}
                required
                className="w-full px-3 py-2 border rounded"
              >
                <option value="">选择所属公司</option>
                <option value="1">某某集团有限公司</option>
                <option value="2">生产子公司</option>
                <option value="11">营销子公司</option>
              </select>
            </>
          )}

          <input
            type="email"
            placeholder="邮箱"
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            required
            className="w-full px-3 py-2 border rounded"
          />
          <input
            type="password"
            placeholder="密码"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            required
            className="w-full px-3 py-2 border rounded"
          />

          <button
            type="submit"
            disabled={loading}
            className="w-full bg-blue-600 text-white py-2 rounded hover:bg-blue-700 disabled:bg-gray-400"
          >
            {loading ? '处理中...' : mode === 'login' ? '登录' : '注册'}
          </button>
        </form>

        <div className="text-center">
          <button
            onClick={() => setMode(mode === 'login' ? 'register' : 'login')}
            className="text-blue-600 hover:underline"
          >
            {mode === 'login' ? '没有账号？去注册' : '已有账号？去登录'}
          </button>
        </div>
      </div>
    </div>
  )
}
```

### 步骤6：重构API调用层

修改 `frontend/lib/api.ts`，区分Supabase直接调用和BFF调用：

```typescript
import { createClient } from './supabase/client'

const supabase = createClient()

// BFF API基础地址（保留Go后端处理复杂业务逻辑）
const BFF_BASE = process.env.NEXT_PUBLIC_BFF_URL || 'http://localhost:8080/api'

// ========== 认证相关（完全使用Supabase） ==========
export async function login(email: string, password: string) {
  const { data, error } = await supabase.auth.signInWithPassword({
    email,
    password,
  })
  if (error) throw error
  return data
}

export async function register(userData: {
  email: string
  password: string
  username: string
  fullName?: string
  companyId: string
}) {
  const { data, error } = await supabase.auth.signUp({
    email: userData.email,
    password: userData.password,
    options: {
      data: {
        username: userData.username,
        full_name: userData.fullName,
        company_id: userData.companyId,
      },
    },
  })
  if (error) throw error
  return data
}

export async function signOut() {
  const { error } = await supabase.auth.signOut()
  if (error) throw error
}

export async function getUserProfile() {
  const { data: { user } } = await supabase.auth.getUser()
  if (!user) throw new Error('未登录')

  const { data: profile, error } = await supabase
    .from('profiles')
    .select('*')
    .eq('id', user.id)
    .single()

  if (error) throw error
  return { user, profile }
}

// ========== 简单数据查询（可直接使用Supabase） ==========
export async function listPeriods() {
  const { data, error } = await supabase
    .from('periods')
    .select('*')
    .order('year_month', { ascending: false })

  if (error) throw error
  return data
}

export async function getRoster(periodId: number) {
  const { data, error } = await supabase
    .from('roster_entries')
    .select('*')
    .eq('period_id', periodId)

  if (error) throw error
  return data
}

// ========== 复杂业务逻辑（调用Go BFF） ==========
async function bffRequest<T>(
  path: string,
  init?: RequestInit
): Promise<T> {
  // 获取Supabase session token
  const { data: { session } } = await supabase.auth.getSession()
  if (!session) throw new Error('未登录')

  const res = await fetch(`${BFF_BASE}${path}`, {
    ...init,
    headers: {
      'Authorization': `Bearer ${session.access_token}`,
      'Content-Type': 'application/json',
      ...init?.headers,
    },
  })

  if (!res.ok) {
    const error = await res.json()
    throw new Error(error.error || '请求失败')
  }

  return res.json()
}

export async function uploadSourceFile({
  periodId,
  scheme,
  part,
  file,
}: {
  periodId: number
  scheme: string
  part: string
  file: File
}) {
  const { data: { session } } = await supabase.auth.getSession()
  if (!session) throw new Error('未登录')

  const formData = new FormData()
  formData.append('scheme', scheme)
  formData.append('part', part)
  formData.append('file', file)

  const res = await fetch(`${BFF_BASE}/periods/${periodId}/files`, {
    method: 'POST',
    headers: {
      'Authorization': `Bearer ${session.access_token}`,
    },
    body: formData,
  })

  if (!res.ok) {
    const error = await res.json()
    throw new Error(error.error || '上传失败')
  }

  return res.json()
}

export async function processPeriod(periodId: number) {
  return bffRequest(`/periods/${periodId}/process`, {
    method: 'POST',
  })
}

export async function getCharges(periodId: number, part: string) {
  return bffRequest(`/periods/${periodId}/charges?part=${part}`)
}

// ... 其他复杂业务逻辑API
```

### 步骤7：更新环境变量

修改 `frontend/.env.local`：

```bash
# Supabase配置
NEXT_PUBLIC_SUPABASE_URL=https://vjpvrzphtnxawkqwuogn.supabase.co
NEXT_PUBLIC_SUPABASE_ANON_KEY=eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...

# Go BFF地址
NEXT_PUBLIC_BFF_URL=http://localhost:8080/api

# 删除旧的配置
# NEXT_PUBLIC_API_BASE_URL=http://localhost:8081/api
```

---

## 后端改造详细步骤

### 步骤1：安装Supabase Go SDK

```bash
cd backend
go get github.com/supabase-community/supabase-go
go get github.com/supabase-community/postgrest-go
```

### 步骤2：创建Supabase客户端

创建 `backend/internal/supabase/client.go`：

```go
package supabase

import (
    "os"
    supa "github.com/supabase-community/supabase-go"
)

var Client *supa.Client

func InitClient() error {
    supabaseURL := os.Getenv("SUPABASE_URL")
    supabaseKey := os.Getenv("SUPABASE_SERVICE_KEY") // 使用service_role key
    
    if supabaseURL == "" || supabaseKey == "" {
        return fmt.Errorf("SUPABASE_URL and SUPABASE_SERVICE_KEY are required")
    }
    
    client, err := supa.NewClient(supabaseURL, supabaseKey, nil)
    if err != nil {
        return err
    }
    
    Client = client
    return nil
}
```

### 步骤3：创建JWT验证中间件

创建 `backend/internal/middleware/supabase_auth.go`：

```go
package middleware

import (
    "context"
    "net/http"
    "strings"
    
    "github.com/golang-jwt/jwt/v5"
)

type contextKey string

const (
    UserIDKey contextKey = "user_id"
    UserEmailKey contextKey = "user_email"
)

// SupabaseAuthMiddleware 验证Supabase JWT
func SupabaseAuthMiddleware(jwtSecret string) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            authHeader := r.Header.Get("Authorization")
            if authHeader == "" {
                http.Error(w, `{"error":"Unauthorized"}`, http.StatusUnauthorized)
                return
            }

            // 提取token
            tokenString := strings.TrimPrefix(authHeader, "Bearer ")
            if tokenString == authHeader {
                http.Error(w, `{"error":"Invalid authorization format"}`, http.StatusUnauthorized)
                return
            }

            // 验证JWT
            token, err := jwt.Parse(tokenString, 