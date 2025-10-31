"use client";

import { useState, useEffect, Suspense } from "react";
import { useRouter, useSearchParams } from "next/navigation";
import { toast } from "sonner";
import { verifyEmail as apiVerifyEmail } from "@/lib/api";

import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";

function VerifyEmailContent() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const [isLoading, setIsLoading] = useState(true);
  const [isSuccess, setIsSuccess] = useState(false);
  const [errorMessage, setErrorMessage] = useState("");

  useEffect(() => {
    const token = searchParams.get("token");

    if (!token) {
      setIsLoading(false);
      setErrorMessage("验证链接无效：缺少验证令牌");
      return;
    }

    // Call the verification API
    verifyEmail(token);
  }, [searchParams]);

  const verifyEmail = async (token: string) => {
    try {
      await apiVerifyEmail(token);

      setIsSuccess(true);
      toast.success("邮箱验证成功！");
    } catch (error) {
      console.error("Email verification error:", error);
      const message = error instanceof Error ? error.message : "邮箱验证失败";
      setErrorMessage(message);
      toast.error(message);
    } finally {
      setIsLoading(false);
    }
  };

  const handleGoToLogin = () => {
    router.push("/auth");
  };

  const handleGoHome = () => {
    router.push("/");
  };

  if (isLoading) {
    return (
      <div className="min-h-screen bg-muted/30 flex items-center justify-center p-6">
        <div className="text-center space-y-4">
          <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary mx-auto"></div>
          <p className="text-muted-foreground">正在验证邮箱...</p>
        </div>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-muted/30 flex items-center justify-center p-6">
      <div className="w-full max-w-md">
        <div className="mb-6 text-center">
          <h1 className="text-3xl font-bold tracking-tight">人事行政管理系统 (hr-office)</h1>
          <p className="text-muted-foreground mt-2">
            邮箱验证结果
          </p>
        </div>

        <Card>
          <CardHeader>
            <CardTitle className={isSuccess ? "text-green-600" : "text-red-600"}>
              {isSuccess ? "验证成功" : "验证失败"}
            </CardTitle>
            <CardDescription>
              {isSuccess
                ? "您的邮箱已成功验证，现在可以登录系统了。"
                : "邮箱验证过程中发生错误。"
              }
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            {isSuccess ? (
              <div className="space-y-4">
                <div className="flex items-center justify-center">
                  <div className="rounded-full bg-green-100 p-3">
                    <svg
                      className="h-6 w-6 text-green-600"
                      fill="none"
                      stroke="currentColor"
                      viewBox="0 0 24 24"
                    >
                      <path
                        strokeLinecap="round"
                        strokeLinejoin="round"
                        strokeWidth={2}
                        d="M5 13l4 4L19 7"
                      />
                    </svg>
                  </div>
                </div>
                <p className="text-center text-sm text-muted-foreground">
                  邮箱验证成功！您现在可以使用注册时的用户名和密码登录系统。
                </p>
                <div className="flex flex-col space-y-2">
                  <Button onClick={handleGoToLogin} className="w-full">
                    立即登录
                  </Button>
                  <Button
                    onClick={handleGoHome}
                    variant="outline"
                    className="w-full"
                  >
                    返回首页
                  </Button>
                </div>
              </div>
            ) : (
              <div className="space-y-4">
                <div className="flex items-center justify-center">
                  <div className="rounded-full bg-red-100 p-3">
                    <svg
                      className="h-6 w-6 text-red-600"
                      fill="none"
                      stroke="currentColor"
                      viewBox="0 0 24 24"
                    >
                      <path
                        strokeLinecap="round"
                        strokeLinejoin="round"
                        strokeWidth={2}
                        d="M6 18L18 6M6 6l12 12"
                      />
                    </svg>
                  </div>
                </div>
                <div className="text-center">
                  <p className="text-sm text-red-600 mb-3">
                    {errorMessage}
                  </p>
                  <p className="text-xs text-muted-foreground">
                    可能的原因：
                  </p>
                  <ul className="text-xs text-muted-foreground mt-1 space-y-1">
                    <li>• 验证链接已过期（48小时有效期）</li>
                    <li>• 验证链接已被使用</li>
                    <li>• 邮箱已经验证过了</li>
                    <li>• 验证链接格式错误</li>
                  </ul>
                </div>
                <div className="flex flex-col space-y-2">
                  <Button onClick={handleGoToLogin} className="w-full">
                    重新登录
                  </Button>
                  <Button
                    onClick={handleGoHome}
                    variant="outline"
                    className="w-full"
                  >
                    返回首页
                  </Button>
                </div>
              </div>
            )}
          </CardContent>
        </Card>
      </div>
    </div>
  );
}

export default function VerifyEmailPage() {
  return (
    <Suspense fallback={
      <div className="min-h-screen bg-muted/30 flex items-center justify-center p-6">
        <div className="text-center space-y-4">
          <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary mx-auto"></div>
          <p className="text-muted-foreground">加载中...</p>
        </div>
      </div>
    }>
      <VerifyEmailContent />
    </Suspense>
  );
}
