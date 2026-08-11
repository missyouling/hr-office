'use client'

import { useState, useEffect, Suspense } from 'react'
import { useRouter, useSearchParams } from 'next/navigation'
import { toast } from 'sonner'
import { validatePasswordResetToken, resetPassword } from '@/lib/api'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'

function ResetPasswordContent() {
  const router = useRouter()
  const searchParams = useSearchParams()
  const [token, setToken] = useState('')
  const [password, setPassword] = useState('')
  const [confirmPassword, setConfirmPassword] = useState('')
  const [loading, setLoading] = useState(true)
  const [submitting, setSubmitting] = useState(false)
  const [error, setError] = useState('')
  const [success, setSuccess] = useState(false)
  const [tokenValid, setTokenValid] = useState(false)

  // 从 URL 获取 token 并验证
  useEffect(() => {
    const tokenParam = searchParams.get('token')

    if (!tokenParam) {
      setLoading(false)
      setError('重置链接无效：缺少重置令牌')
      return
    }

    setToken(tokenParam)

    // 调用后端验证 token 有效性
    validatePasswordResetToken(tokenParam)
      .then((result) => {
        if (!result.valid) {
          setError('重置链接已过期或无效')
          return
        }
        setTokenValid(true)
      })
      .catch((err) => {
        console.error('Token 验证失败:', err)
        setError('重置链接已过期或无效')
      })
      .finally(() => {
        setLoading(false)
      })
  }, [searchParams])

  const handleResetPassword = async (e: React.FormEvent) => {
    e.preventDefault()
    setError('')

    if (password !== confirmPassword) {
      setError('两次输入的密码不一致')
      return
    }

    if (password.length < 6) {
      setError('密码长度至少为6个字符')
      return
    }

    setSubmitting(true)

    try {
      await resetPassword({ token, newPassword: password })
      setSuccess(true)
      toast.success('密码重置成功')
      setTimeout(() => {
        router.push('/auth')
      }, 2000)
    } catch (err) {
      console.error('密码重置失败:', err)
      const message = err instanceof Error ? err.message : '密码重置失败，请重试'
      setError(message)
      toast.error(message)
    } finally {
      setSubmitting(false)
    }
  }

  // 加载中状态
  if (loading) {
    return (
      <div className="flex min-h-screen items-center justify-center">
        <div className="text-center space-y-4">
          <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary mx-auto"></div>
          <p className="text-muted-foreground">正在验证重置令牌...</p>
        </div>
      </div>
    )
  }

  // Token 无效 / 验证失败
  if (!tokenValid) {
    return (
      <div className="flex min-h-screen items-center justify-center">
        <Card className="w-[400px]">
          <CardHeader>
            <CardTitle className="text-red-600">链接无效</CardTitle>
            <CardDescription>{error || '重置链接已过期或无效'}</CardDescription>
          </CardHeader>
          <CardContent>
            <Button className="w-full" onClick={() => router.push('/auth')}>
              返回登录
            </Button>
          </CardContent>
        </Card>
      </div>
    )
  }

  // 重置成功
  if (success) {
    return (
      <div className="flex min-h-screen items-center justify-center">
        <Card className="w-[400px]">
          <CardHeader>
            <CardTitle className="text-green-600">密码重置成功</CardTitle>
            <CardDescription>
              您的密码已成功重置，即将跳转到登录页面...
            </CardDescription>
          </CardHeader>
        </Card>
      </div>
    )
  }

  // 密码重置表单
  return (
    <div className="flex min-h-screen items-center justify-center">
      <Card className="w-[400px]">
        <CardHeader>
          <CardTitle>重置密码</CardTitle>
          <CardDescription>请输入您的新密码</CardDescription>
        </CardHeader>
        <CardContent>
          <form onSubmit={handleResetPassword} className="space-y-4">
            {error && (
              <div className="bg-red-50 border border-red-200 text-red-600 px-4 py-3 rounded">
                {error}
              </div>
            )}

            <div className="space-y-2">
              <Label htmlFor="password">新密码</Label>
              <Input
                id="password"
                type="password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                placeholder="请输入新密码"
                required
                minLength={6}
              />
            </div>

            <div className="space-y-2">
              <Label htmlFor="confirmPassword">确认密码</Label>
              <Input
                id="confirmPassword"
                type="password"
                value={confirmPassword}
                onChange={(e) => setConfirmPassword(e.target.value)}
                placeholder="请再次输入密码"
                required
                minLength={6}
              />
            </div>

            <Button
              type="submit"
              className="w-full"
              disabled={submitting}
            >
              {submitting ? '重置中...' : '重置密码'}
            </Button>
          </form>
        </CardContent>
      </Card>
    </div>
  )
}

export default function ResetPasswordPage() {
  return (
    <Suspense
      fallback={
        <div className="flex min-h-screen items-center justify-center">
          <div className="text-center space-y-4">
            <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary mx-auto"></div>
            <p className="text-muted-foreground">加载中...</p>
          </div>
        </div>
      }
    >
      <ResetPasswordContent />
    </Suspense>
  )
}
