'use client'

import { useEffect } from 'react'
import { useRouter } from 'next/navigation'
import { createClient } from '@/lib/supabase/client'

export default function AuthCallbackPage() {
  const router = useRouter()

  useEffect(() => {
    const handleCallback = async () => {
      const supabase = createClient()
      
      // 处理邮箱验证回调
      const { error } = await supabase.auth.exchangeCodeForSession(
        window.location.search.substring(1)
      )

      if (error) {
        console.error('认证回调错误:', error)
        router.push('/auth?error=callback_failed')
      } else {
        // 认证成功，跳转到主页
        router.push('/')
      }
    }

    handleCallback()
  }, [router])

  return (
    <div className="flex min-h-screen items-center justify-center">
      <div className="text-center">
        <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-gray-900 mx-auto mb-4"></div>
        <p className="text-gray-600">正在验证邮箱...</p>
      </div>
    </div>
  )
}