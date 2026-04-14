"use client";

import { useState, useEffect, Suspense } from "react";
import { useRouter, useSearchParams } from "next/navigation";
import { toast } from "sonner";
import {
  validatePasswordResetToken,
  resetPassword as apiResetPassword,
  type PasswordResetTokenValidation,
} from "@/lib/api";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Eye, EyeOff } from "lucide-react";

type PageStatus = "validating" | "ready" | "success" | "error";

function PasswordResetContent() {
  const router = useRouter();
  const searchParams = useSearchParams();

  const [status, setStatus] = useState<PageStatus>("validating");
  const [token, setToken] = useState("");
  const [associatedEmail, setAssociatedEmail] = useState("");
  const [errorMessage, setErrorMessage] = useState("正在校验重置链接，请稍等...");
  const [form, setForm] = useState({ newPassword: "", confirmPassword: "" });
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [showNewPassword, setShowNewPassword] = useState(false);
  const [showConfirmPassword, setShowConfirmPassword] = useState(false);

  useEffect(() => {
    const queryToken = searchParams.get("token");

    if (!queryToken) {
      setStatus("error");
      setErrorMessage("重置链接无效：缺少必要的身份令牌。请重新申请密码重置。");
      return;
    }

    setToken(queryToken);
    setStatus("validating");
    setErrorMessage("正在校验重置链接，请稍等...");

    const validate = async (value: string) => {
      try {
        const result: PasswordResetTokenValidation = await validatePasswordResetToken(value);

        if (!result?.valid) {
          setStatus("error");
          setErrorMessage("重置链接无效或已失效，请重新申请密码重置。\n链接可能已经使用或超过有效期。");
          toast.error("重置链接无效或已失效");
          return;
        }

        setAssociatedEmail(result.email ?? "");
        setStatus("ready");
        toast.success("重置链接校验通过，请设置新密码");
      } catch (error) {
        const message = error instanceof Error ? error.message : "重置链接校验失败";
        setStatus("error");
        setErrorMessage(message || "重置链接校验失败，请稍后重试。");
        toast.error(message);
      }
    };

    validate(queryToken);
  }, [searchParams]);

  const validatePasswordFormat = (value: string) => {
    if (value.length < 6) {
      return "密码长度至少为6位";
    }
    if (value.length > 128) {
      return "密码长度不能超过128位";
    }
    const hasLetter = /[a-zA-Z]/.test(value);
    const hasNumber = /[0-9]/.test(value);
    if (!hasLetter || !hasNumber) {
      return "密码需同时包含字母和数字";
    }
    return "";
  };

  const handleSubmit = async (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();

    const newPassword = form.newPassword.trim();
    const confirmPassword = form.confirmPassword.trim();

    if (!newPassword) {
      toast.error("请输入新密码");
      return;
    }

    const passwordError = validatePasswordFormat(newPassword);
    if (passwordError) {
      toast.error(passwordError);
      return;
    }

    if (!confirmPassword) {
      toast.error("请再次输入新密码");
      return;
    }

    if (newPassword !== confirmPassword) {
      toast.error("两次输入的密码不一致");
      return;
    }

    setIsSubmitting(true);

    try {
      await apiResetPassword({ token, newPassword });
      setStatus("success");
      toast.success("密码重置成功，现在可以使用新密码登录");
    } catch (error) {
      const message = error instanceof Error ? error.message : "密码重置失败";
      toast.error(message);
      setErrorMessage(message);
      setStatus("error");
    } finally {
      setIsSubmitting(false);
    }
  };

  const handleGoToLogin = () => {
    router.push("/auth");
  };

  const handleGoHome = () => {
    router.push("/");
  };

  if (status === "validating") {
    return (
      <div className="min-h-screen bg-muted/30 flex items-center justify-center p-6">
        <div className="text-center space-y-4">
          <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary mx-auto" />
          <p className="text-muted-foreground">{errorMessage}</p>
        </div>
      </div>
    );
  }

  const renderErrorCard = () => (
    <Card>
      <CardHeader>
        <CardTitle className="text-red-600">重置失败</CardTitle>
        <CardDescription>无法继续密码重置流程</CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        <p className="text-sm text-red-600 whitespace-pre-line">{errorMessage}</p>
        <div className="text-xs text-muted-foreground space-y-1">
          <p>可能原因：</p>
          <ul className="space-y-1">
            <li>• 重置链接已过期（48小时有效）</li>
            <li>• 重置链接已经被使用</li>
            <li>• 收到多封邮件，误用了旧链接</li>
            <li>• 链接被篡改或复制不完整</li>
          </ul>
        </div>
        <div className="flex flex-col space-y-2">
          <Button onClick={handleGoToLogin} className="w-full">
            返回登录页重新申请
          </Button>
          <Button onClick={handleGoHome} variant="outline" className="w-full">
            返回首页
          </Button>
        </div>
      </CardContent>
    </Card>
  );

  const renderSuccessCard = () => (
    <Card>
      <CardHeader>
        <CardTitle className="text-green-600">密码重置成功</CardTitle>
        <CardDescription>您可以使用新密码登录系统</CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        <div className="flex items-center justify-center">
          <div className="rounded-full bg-green-100 p-3">
            <svg
              className="h-6 w-6 text-green-600"
              fill="none"
              stroke="currentColor"
              viewBox="0 0 24 24"
            >
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
            </svg>
          </div>
        </div>
        <p className="text-sm text-muted-foreground text-center">
          如果您此前已经登录，为了安全起见建议重新登录一次。
        </p>
        <div className="flex flex-col space-y-2">
          <Button onClick={handleGoToLogin} className="w-full">
            立即登录
          </Button>
          <Button onClick={handleGoHome} variant="outline" className="w-full">
            返回首页
          </Button>
        </div>
      </CardContent>
    </Card>
  );

  const renderFormCard = () => (
    <Card>
      <CardHeader>
        <CardTitle>设置新密码</CardTitle>
        <CardDescription>
          {associatedEmail
            ? `请输入新密码，账号 ${associatedEmail} 将立即生效。`
            : "请输入新密码，设置成功后即可使用该密码登录。"}
        </CardDescription>
      </CardHeader>
      <CardContent>
        <form onSubmit={handleSubmit} className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="new-password">新密码</Label>
            <div className="relative">
              <Input
                id="new-password"
                type={showNewPassword ? "text" : "password"}
                placeholder="至少6位，同时包含字母和数字"
                value={form.newPassword}
                onChange={(event) =>
                  setForm((prev) => ({ ...prev, newPassword: event.target.value }))
                }
                className="pr-10"
                required
              />
              <Button
                type="button"
                variant="ghost"
                size="icon"
                className="absolute right-1 top-1/2 -translate-y-1/2 h-8 w-8"
                onClick={() => setShowNewPassword((prev) => !prev)}
              >
                {showNewPassword ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
              </Button>
            </div>
          </div>
          <div className="space-y-2">
            <Label htmlFor="confirm-password">确认新密码</Label>
            <div className="relative">
              <Input
                id="confirm-password"
                type={showConfirmPassword ? "text" : "password"}
                placeholder="请再次输入新密码"
                value={form.confirmPassword}
                onChange={(event) =>
                  setForm((prev) => ({ ...prev, confirmPassword: event.target.value }))
                }
                className="pr-10"
                required
              />
              <Button
                type="button"
                variant="ghost"
                size="icon"
                className="absolute right-1 top-1/2 -translate-y-1/2 h-8 w-8"
                onClick={() => setShowConfirmPassword((prev) => !prev)}
              >
                {showConfirmPassword ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
              </Button>
            </div>
          </div>
          <p className="text-xs text-muted-foreground">
            新密码需满足以下条件：长度为6-128位，且至少包含一个字母与一个数字。
          </p>
          <div className="flex flex-col space-y-2">
            <Button type="submit" className="w-full" disabled={isSubmitting}>
              {isSubmitting ? "提交中..." : "确认重置密码"}
            </Button>
            <Button type="button" variant="outline" onClick={handleGoToLogin} className="w-full">
              返回登录页
            </Button>
          </div>
        </form>
      </CardContent>
    </Card>
  );

  return (
    <div className="min-h-screen bg-muted/30 flex items-center justify-center p-6">
      <div className="w-full max-w-md">
        <div className="mb-6 text-center">
          <h1 className="text-3xl font-bold tracking-tight">人事行政管理系统 (hr-office)</h1>
          <p className="text-muted-foreground mt-2">密码重置</p>
        </div>
        {status === "error" && renderErrorCard()}
        {status === "success" && renderSuccessCard()}
        {status === "ready" && renderFormCard()}
      </div>
    </div>
  );
}

export default function ResetPasswordPage() {
  return (
    <Suspense
      fallback={
        <div className="min-h-screen bg-muted/30 flex items-center justify-center p-6">
          <div className="text-center space-y-4">
            <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary mx-auto" />
            <p className="text-muted-foreground">加载中...</p>
          </div>
        </div>
      }
    >
      <PasswordResetContent />
    </Suspense>
  );
}
