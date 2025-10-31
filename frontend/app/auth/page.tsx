"use client";

import { useState, useEffect, useMemo } from "react";
import { useRouter } from "next/navigation";
import { toast } from "sonner";
import { useAuth } from "@/lib/supabase/auth-context";
import { createClient } from "@/lib/supabase/client";
import { checkAccountAvailability, resendVerificationEmail } from "@/lib/api";
import { Eye, EyeOff, Building2 } from "lucide-react";

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
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";

interface LoginData {
  email: string;
  password: string;
}

interface RegisterData {
  username: string;
  email: string;
  password: string;
  confirmPassword: string;
  fullName: string;
  companyId: string;
}

interface CompanyOption {
  id: string;
  name: string;
  type: "group" | "subsidiary";
}

export default function AuthPage() {
  const router = useRouter();
  const { user, loading } = useAuth();
  const supabase = useMemo(() => createClient(), []);

  const [isLoading, setIsLoading] = useState(false);
  const [showLoginPassword, setShowLoginPassword] = useState(false);
  const [showRegisterPassword, setShowRegisterPassword] = useState(false);
  const [showConfirmPassword, setShowConfirmPassword] = useState(false);
  
  const [loginData, setLoginData] = useState<LoginData>({
    email: "",
    password: "",
  });
  
  const [registerData, setRegisterData] = useState<RegisterData>({
    username: "",
    email: "",
    password: "",
    confirmPassword: "",
    fullName: "",
    companyId: "",
  });
  
  const [companyOptions] = useState<CompanyOption[]>([
    { id: "1", name: "某某集团有限公司", type: "group" },
    { id: "2", name: "生产子公司", type: "subsidiary" },
    { id: "11", name: "营销子公司", type: "subsidiary" },
  ]);
  const [availabilityHints, setAvailabilityHints] = useState<{ username: string; email: string }>({
    username: "",
    email: "",
  });
  const [showResendDialog, setShowResendDialog] = useState(false);
  const [resendEmailInput, setResendEmailInput] = useState("");
  const [isResendingEmail, setIsResendingEmail] = useState(false);
  const [resendFeedback, setResendFeedback] = useState<string | null>(null);

  // 如果已登录则重定向到首页
  useEffect(() => {
    if (!loading && user) {
      router.push("/");
    }
  }, [user, loading, router]);

  // 加载中显示loading状态
  if (loading) {
    return (
      <div className="min-h-screen bg-muted/30 flex items-center justify-center">
        <div className="text-center space-y-4">
          <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary mx-auto"></div>
          <p className="text-muted-foreground">检查登录状态...</p>
        </div>
      </div>
    );
  }

  // 已登录则不渲染
  if (user) {
    return null;
  }

  // 验证函数
  const validateEmail = (email: string) => {
    const emailRegex = /^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$/;
    return emailRegex.test(email);
  };

  const validateUsername = (username: string) => {
    const usernameRegex = /^[a-z0-9_]{6,20}$/;
    return usernameRegex.test(username);
  };

  const validatePassword = (password: string) => {
    const hasLetter = /[a-zA-Z]/.test(password);
    const hasNumber = /[0-9]/.test(password);
    return password.length >= 6 && hasLetter && hasNumber;
  };

  const validateChineseName = (name: string) => {
    const chineseRegex = /^[\u4e00-\u9fa5]{2,10}$/;
    return chineseRegex.test(name);
  };

  const getPasswordMatchStatus = () => {
    if (!registerData.password || !registerData.confirmPassword) {
      return null;
    }
    return registerData.password === registerData.confirmPassword;
  };

  const getFieldValidationStatus = (field: string, value: string) => {
    if (!value) return null;

    switch (field) {
      case 'email':
        return validateEmail(value);
      case 'username':
        return validateUsername(value);
      case 'password':
        return validatePassword(value);
      case 'fullName':
        return validateChineseName(value);
      default:
        return null;
    }
  };

  const getFieldErrorMessage = (field: string) => {
    const value = registerData[field as keyof RegisterData] as string;
    if (!value) return null;

    switch (field) {
      case 'email':
        return !validateEmail(value) ? '请输入有效的邮箱地址' : null;
      case 'username':
        return !validateUsername(value) ? '用户名必须是6-20个字符，支持小写字母、数字、下划线' : null;
      case 'password':
        return !validatePassword(value) ? '密码必须包含字母和数字，最少6位' : null;
      case 'fullName':
        return !validateChineseName(value) ? '姓名必须为中文，2-10个字符' : null;
      default:
        return null;
    }
  };

  const handleLogin = async (e: React.FormEvent) => {
    e.preventDefault();
    setIsLoading(true);

    try {
      const { error } = await supabase.auth.signInWithPassword({
        email: loginData.email,
        password: loginData.password,
      });

      if (error) throw error;

      toast.success("登录成功");
      router.push("/");
    } catch (error) {
      console.error("Login error:", error);
      const errorMessage = error instanceof Error ? error.message : "登录失败";
      toast.error(errorMessage);
    } finally {
      setIsLoading(false);
    }
  };

  const handleRegister = async (e: React.FormEvent) => {
    e.preventDefault();

    // 验证所有字段
    if (!registerData.username.trim()) {
      toast.error("请输入用户名");
      return;
    }
    if (!validateUsername(registerData.username)) {
      toast.error("用户名格式不正确：必须是6-20个字符，支持小写字母、数字、下划线");
      return;
    }

    if (!registerData.email.trim()) {
      toast.error("请输入邮箱地址");
      return;
    }
    if (!validateEmail(registerData.email)) {
      toast.error("请输入有效的邮箱地址");
      return;
    }

    if (!registerData.fullName.trim()) {
      toast.error("请输入姓名");
      return;
    }
    if (!validateChineseName(registerData.fullName)) {
      toast.error("姓名必须为中文，2-10个字符");
      return;
    }

    if (!registerData.companyId) {
      toast.error("请选择所属公司");
      return;
    }

    if (!registerData.password) {
      toast.error("请输入密码");
      return;
    }
    if (!validatePassword(registerData.password)) {
      toast.error("密码必须包含字母和数字，最少6位");
      return;
    }

    if (!registerData.confirmPassword) {
      toast.error("请输入确认密码");
      return;
    }

    if (registerData.password !== registerData.confirmPassword) {
      toast.error("两次输入的密码不一致");
      return;
    }

    setIsLoading(true);
    setAvailabilityHints({ username: "", email: "" });

    const usernameToCheck = registerData.username.trim();
    const emailToCheck = registerData.email.trim();

    try {
      const availability = await checkAccountAvailability({
        username: usernameToCheck,
        email: emailToCheck,
      });

      const hints: { username: string; email: string } = { username: "", email: "" };
      if (!availability.username_available) {
        hints.username = "用户名已存在，请更换";
      }
      if (!availability.email_available) {
        hints.email = "该邮箱已注册，请直接登录或找回密码";
      }

      if (hints.username || hints.email) {
        setAvailabilityHints(hints);
        toast.error(hints.username || hints.email);
        setIsLoading(false);
        return;
      }
    } catch (err) {
      console.error("checkAccountAvailability error:", err);
      toast.error("检查账号可用性失败，请稍后再试");
      setIsLoading(false);
      return;
    }

    try {
      // 使用Supabase Auth注册
      // 注意：profile会通过数据库触发器自动创建，不需要手动插入
      const { error } = await supabase.auth.signUp({
        email: emailToCheck,
        password: registerData.password,
        options: {
          data: {
            username: usernameToCheck,
            full_name: registerData.fullName,
            company_id: registerData.companyId,
          },
        },
      });

      if (error) throw error;

      // Profile会通过数据库触发器自动创建（见 001_initial_schema.sql 第396-416行）
      // 触发器会从 raw_user_meta_data 中提取 username, full_name, company_id

      toast.success(
        `注册成功！我们已向 ${registerData.email} 发送了验证邮件，请点击邮件中的链接验证您的邮箱，然后返回登录。`,
        { duration: 8000 }
      );

      // 清空表单
      setRegisterData({
        username: "",
        email: "",
        password: "",
        confirmPassword: "",
        fullName: "",
        companyId: "",
      });
      setAvailabilityHints({ username: "", email: "" });

      // 切换到登录标签
      const loginTab = document.querySelector('[value="login"]') as HTMLElement;
      if (loginTab) {
        loginTab.click();
      }
    } catch (error) {
      console.error("Register error:", error);
      let errorMessage = error instanceof Error ? error.message : "注册失败";

      try {
        const availability = await checkAccountAvailability({
          username: usernameToCheck,
          email: emailToCheck,
        });
        const hints: { username: string; email: string } = { username: "", email: "" };
        if (!availability.username_available) {
          hints.username = "用户名已存在，请更换";
        }
        if (!availability.email_available) {
          hints.email = "该邮箱已注册，请直接登录或找回密码";
        }
        if (hints.username || hints.email) {
          setAvailabilityHints(hints);
          errorMessage = hints.username || hints.email || errorMessage;
        }
      } catch (checkErr) {
        console.error("re-check availability failed:", checkErr);
      }

      toast.error(errorMessage);
    } finally {
      setIsLoading(false);
    }
  };

  const handleResendVerification = async (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    const targetEmail = resendEmailInput.trim();
    if (!targetEmail) {
      toast.error("请输入邮箱地址");
      return;
    }

    setIsResendingEmail(true);
    setResendFeedback(null);

    try {
      const result = await resendVerificationEmail(targetEmail);
      setResendFeedback(result.message);
      toast.success("如果邮箱未验证，新的验证邮件已发送");
    } catch (error) {
      const errorMessage = error instanceof Error ? error.message : "发送失败";
      setResendFeedback(errorMessage);
      toast.error(errorMessage);
    } finally {
      setIsResendingEmail(false);
    }
  };

  return (
    <div className="min-h-screen bg-muted/30 flex items-center justify-center p-6">
      <div className="w-full max-w-md">
        <div className="mb-6 text-center">
          <h1 className="text-3xl font-bold tracking-tight">人事行政管理系统</h1>
          <p className="text-muted-foreground mt-2">
            请登录或注册以访问系统
          </p>
        </div>

        <Tabs defaultValue="login" className="w-full">
          <TabsList className="grid w-full grid-cols-2">
            <TabsTrigger value="login">登录</TabsTrigger>
            <TabsTrigger value="register">注册</TabsTrigger>
          </TabsList>

          <TabsContent value="login">
            <Card>
              <CardHeader>
                <CardTitle>登录</CardTitle>
                <CardDescription>
                  使用您的邮箱和密码登录系统
                </CardDescription>
              </CardHeader>
              <CardContent>
                <form onSubmit={handleLogin} className="space-y-4">
                  <div className="space-y-2">
                    <Label htmlFor="login-email">邮箱</Label>
                    <Input
                      id="login-email"
                      type="email"
                      placeholder="请输入邮箱"
                      value={loginData.email}
                      onChange={(e) =>
                        setLoginData({ ...loginData, email: e.target.value })
                      }
                      required
                    />
                  </div>
                  <div className="space-y-2">
                    <Label htmlFor="login-password">密码</Label>
                    <div className="relative">
                      <Input
                        id="login-password"
                        type={showLoginPassword ? "text" : "password"}
                        placeholder="请输入密码"
                        value={loginData.password}
                        onChange={(e) =>
                          setLoginData({ ...loginData, password: e.target.value })
                        }
                        className="pr-10"
                        required
                      />
                      <Button
                        type="button"
                        variant="ghost"
                        size="sm"
                        className="absolute right-0 top-0 h-full px-3 py-2 hover:bg-transparent"
                        onClick={() => setShowLoginPassword(!showLoginPassword)}
                      >
                        {showLoginPassword ? (
                          <EyeOff className="h-4 w-4" />
                        ) : (
                          <Eye className="h-4 w-4" />
                        )}
                      </Button>
                    </div>
                  </div>
                  <Button type="submit" className="w-full" disabled={isLoading}>
                    {isLoading ? "登录中..." : "登录"}
                  </Button>
                </form>
                <div className="mt-2 text-right">
                  <Button
                    type="button"
                    variant="link"
                    size="sm"
                    className="px-0"
                    onClick={() => {
                      setResendEmailInput(loginData.email || registerData.email || "");
                      setResendFeedback(null);
                      setShowResendDialog(true);
                    }}
                  >
                    未收到验证邮件？重新发送
                  </Button>
                </div>
              </CardContent>
            </Card>
          </TabsContent>

          <TabsContent value="register">
            <Card>
              <CardHeader>
                <CardTitle>注册</CardTitle>
                <CardDescription>
                  创建新账户以开始使用系统
                </CardDescription>
              </CardHeader>
              <CardContent>
                <form onSubmit={handleRegister} className="space-y-4">
                  <div className="space-y-2">
                    <Label htmlFor="register-username">用户名 *</Label>
                    <Input
                      id="register-username"
                      type="text"
                      placeholder="6-20个字符，支持小写字母数字下划线"
                      value={registerData.username}
                      onChange={(e) => {
                        setRegisterData({ ...registerData, username: e.target.value });
                        if (availabilityHints.username) {
                          setAvailabilityHints((prev) => ({ ...prev, username: "" }));
                        }
                      }}
                      className={`${registerData.username && getFieldValidationStatus('username', registerData.username) === false ? 'border-red-500' : ''} ${registerData.username && getFieldValidationStatus('username', registerData.username) === true ? 'border-green-500' : ''}`}
                      required
                    />
                    {registerData.username && (
                      <div className={`text-sm flex items-center gap-1 ${
                        getFieldValidationStatus('username', registerData.username)
                          ? "text-green-600"
                          : "text-red-600"
                      }`}>
                        {getFieldValidationStatus('username', registerData.username) ? (
                          <>
                            <span className="text-green-500">✓</span>
                            用户名格式正确
                          </>
                        ) : (
                          <>
                            <span className="text-red-500">✗</span>
                            {getFieldErrorMessage('username')}
                          </>
                        )}
                      </div>
                    )}
                    {availabilityHints.username && (
                      <div className="text-sm text-red-600 flex items-center gap-1">
                        <span className="text-red-500">✗</span>
                        {availabilityHints.username}
                      </div>
                    )}
                  </div>
                  <div className="space-y-2">
                    <Label htmlFor="register-email">邮箱 *</Label>
                    <Input
                      id="register-email"
                      type="email"
                      placeholder="请输入邮箱地址"
                      value={registerData.email}
                      onChange={(e) => {
                        setRegisterData({ ...registerData, email: e.target.value });
                        if (availabilityHints.email) {
                          setAvailabilityHints((prev) => ({ ...prev, email: "" }));
                        }
                      }}
                      className={`${registerData.email && getFieldValidationStatus('email', registerData.email) === false ? 'border-red-500' : ''} ${registerData.email && getFieldValidationStatus('email', registerData.email) === true ? 'border-green-500' : ''}`}
                      required
                    />
                    {registerData.email && (
                      <div className={`text-sm flex items-center gap-1 ${
                        getFieldValidationStatus('email', registerData.email)
                          ? "text-green-600"
                          : "text-red-600"
                      }`}>
                        {getFieldValidationStatus('email', registerData.email) ? (
                          <>
                            <span className="text-green-500">✓</span>
                            邮箱格式正确
                          </>
                        ) : (
                          <>
                            <span className="text-red-500">✗</span>
                            {getFieldErrorMessage('email')}
                          </>
                        )}
                      </div>
                    )}
                    {availabilityHints.email && (
                      <div className="text-sm text-red-600 flex items-center gap-1">
                        <span className="text-red-500">✗</span>
                        {availabilityHints.email}
                      </div>
                    )}
                  </div>
                  <div className="space-y-2">
                    <Label htmlFor="register-fullname">姓名 *</Label>
                    <Input
                      id="register-fullname"
                      type="text"
                      placeholder="请输入真实姓名（2-10个中文字符）"
                      value={registerData.fullName}
                      onChange={(e) =>
                        setRegisterData({ ...registerData, fullName: e.target.value })
                      }
                      className={`${registerData.fullName && getFieldValidationStatus('fullName', registerData.fullName) === false ? 'border-red-500' : ''} ${registerData.fullName && getFieldValidationStatus('fullName', registerData.fullName) === true ? 'border-green-500' : ''}`}
                      required
                    />
                    {registerData.fullName && (
                      <div className={`text-sm flex items-center gap-1 ${
                        getFieldValidationStatus('fullName', registerData.fullName)
                          ? "text-green-600"
                          : "text-red-600"
                      }`}>
                        {getFieldValidationStatus('fullName', registerData.fullName) ? (
                          <>
                            <span className="text-green-500">✓</span>
                            姓名格式正确
                          </>
                        ) : (
                          <>
                            <span className="text-red-500">✗</span>
                            {getFieldErrorMessage('fullName')}
                          </>
                        )}
                      </div>
                    )}
                  </div>
                  <div className="space-y-2">
                    <Label htmlFor="register-company">所属公司 *</Label>
                    <Select
                      value={registerData.companyId}
                      onValueChange={(value) =>
                        setRegisterData({ ...registerData, companyId: value })
                      }
                    >
                      <SelectTrigger>
                        <SelectValue placeholder="请选择所属公司" />
                      </SelectTrigger>
                      <SelectContent>
                        {companyOptions.map((company) => (
                          <SelectItem key={company.id} value={company.id}>
                            <div className="flex items-center gap-2">
                              <Building2 className="h-4 w-4" />
                              <span>{company.name}</span>
                              <span className="text-xs text-muted-foreground">
                                ({company.type === "group" ? "集团" : "子公司"})
                              </span>
                            </div>
                          </SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
                  </div>
                  <div className="space-y-2">
                    <Label htmlFor="register-password">密码 *</Label>
                    <div className="relative">
                      <Input
                        id="register-password"
                        type={showRegisterPassword ? "text" : "password"}
                        placeholder="至少6位，包含字母和数字"
                        value={registerData.password}
                        onChange={(e) =>
                          setRegisterData({ ...registerData, password: e.target.value })
                        }
                        className="pr-10"
                        required
                      />
                      <Button
                        type="button"
                        variant="ghost"
                        size="sm"
                        className="absolute right-0 top-0 h-full px-3 py-2 hover:bg-transparent"
                        onClick={() => setShowRegisterPassword(!showRegisterPassword)}
                      >
                        {showRegisterPassword ? (
                          <EyeOff className="h-4 w-4" />
                        ) : (
                          <Eye className="h-4 w-4" />
                        )}
                      </Button>
                    </div>
                    {registerData.password && (
                      <div className={`text-sm flex items-center gap-1 ${
                        getFieldValidationStatus('password', registerData.password)
                          ? "text-green-600"
                          : "text-red-600"
                      }`}>
                        {getFieldValidationStatus('password', registerData.password) ? (
                          <>
                            <span className="text-green-500">✓</span>
                            密码格式正确
                          </>
                        ) : (
                          <>
                            <span className="text-red-500">✗</span>
                            {getFieldErrorMessage('password')}
                          </>
                        )}
                      </div>
                    )}
                  </div>
                  <div className="space-y-2">
                    <Label htmlFor="register-confirm-password">确认密码 *</Label>
                    <div className="relative">
                      <Input
                        id="register-confirm-password"
                        type={showConfirmPassword ? "text" : "password"}
                        placeholder="请再次输入密码"
                        value={registerData.confirmPassword}
                        onChange={(e) =>
                          setRegisterData({ ...registerData, confirmPassword: e.target.value })
                        }
                        className={`pr-10 ${registerData.confirmPassword && getPasswordMatchStatus() === false ? 'border-red-500' : ''}`}
                        required
                      />
                      <Button
                        type="button"
                        variant="ghost"
                        size="sm"
                        className="absolute right-0 top-0 h-full px-3 py-2 hover:bg-transparent"
                        onClick={() => setShowConfirmPassword(!showConfirmPassword)}
                      >
                        {showConfirmPassword ? (
                          <EyeOff className="h-4 w-4" />
                        ) : (
                          <Eye className="h-4 w-4" />
                        )}
                      </Button>
                    </div>
                    {getPasswordMatchStatus() !== null && (
                      <div className={`text-sm flex items-center gap-1 ${
                        getPasswordMatchStatus()
                          ? "text-green-600"
                          : "text-red-600"
                      }`}>
                        {getPasswordMatchStatus() ? (
                          <>
                            <span className="text-green-500">✓</span>
                            密码匹配
                          </>
                        ) : (
                          <>
                            <span className="text-red-500">✗</span>
                            密码不匹配
                          </>
                        )}
                      </div>
                    )}
                  </div>
                  <Button type="submit" className="w-full" disabled={isLoading}>
                    {isLoading ? "注册中..." : "注册"}
                  </Button>
                </form>
              </CardContent>
            </Card>
          </TabsContent>
        </Tabs>
        <Dialog
          open={showResendDialog}
          onOpenChange={(open) => {
            setShowResendDialog(open);
            if (!open) {
              setResendEmailInput("");
              setResendFeedback(null);
              setIsResendingEmail(false);
            }
          }}
        >
          <DialogContent>
            <DialogHeader>
              <DialogTitle>重新发送验证邮件</DialogTitle>
              <DialogDescription>
                输入注册时使用的邮箱，如果该邮箱尚未验证，我们会重新发送验证链接。
              </DialogDescription>
            </DialogHeader>
            <form onSubmit={handleResendVerification} className="space-y-4">
              <div className="space-y-2">
                <Label htmlFor="resend-email">邮箱</Label>
                <Input
                  id="resend-email"
                  type="email"
                  value={resendEmailInput}
                  onChange={(event) => setResendEmailInput(event.target.value)}
                  placeholder="请输入注册邮箱"
                  required
                />
              </div>
              {resendFeedback && (
                <div className="text-sm text-muted-foreground">{resendFeedback}</div>
              )}
              <DialogFooter>
                <Button
                  type="submit"
                  disabled={isResendingEmail}
                  className="w-full"
                >
                  {isResendingEmail ? "发送中..." : "发送验证邮件"}
                </Button>
              </DialogFooter>
            </form>
          </DialogContent>
        </Dialog>
      </div>
    </div>
  );
}
