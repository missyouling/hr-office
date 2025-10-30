"use client";

import { useState, useEffect } from "react";
import { useRouter } from "next/navigation";
import { toast } from "sonner";
import { useAuth, User } from "@/lib/auth";
import { login as apiLogin, register as apiRegister, getCompanyOptions, type CompanyOption } from "@/lib/api";
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

interface LoginData {
  username: string;
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

// CompanyOption 接口现在从 API 模块导入

export default function AuthPage() {
  const router = useRouter();
  const { login, isAuthenticated, isLoading: authLoading } = useAuth();

  // All useState hooks must come before any conditional returns
  const [isLoading, setIsLoading] = useState(false);
  const [showLoginPassword, setShowLoginPassword] = useState(false);
  const [showRegisterPassword, setShowRegisterPassword] = useState(false);
  const [showConfirmPassword, setShowConfirmPassword] = useState(false);
  const [loginData, setLoginData] = useState<LoginData>({
    username: "",
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
  const [companyOptions, setCompanyOptions] = useState<CompanyOption[]>([]);

  // All useEffect hooks must also come before any conditional returns
  // Redirect to home if already authenticated
  useEffect(() => {
    if (!authLoading && isAuthenticated) {
      router.push("/");
    }
  }, [isAuthenticated, authLoading, router]);

  // 获取公司选项
  useEffect(() => {
    const fetchCompanyOptions = async () => {
      try {
        const options = await getCompanyOptions();
        setCompanyOptions(options);
      } catch (error) {
        console.error('Failed to fetch company options:', error);
        // 使用默认选项作为回退
        setCompanyOptions([
          { id: "1", name: "某某集团有限公司", type: "group" },
          { id: "2", name: "生产子公司", type: "subsidiary" },
          { id: "11", name: "营销子公司", type: "subsidiary" },
        ]);
      }
    };
    fetchCompanyOptions();
  }, []);

  // Show loading screen while checking authentication
  if (authLoading) {
    return (
      <div className="min-h-screen bg-muted/30 flex items-center justify-center">
        <div className="text-center space-y-4">
          <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary mx-auto"></div>
          <p className="text-muted-foreground">检查登录状态...</p>
        </div>
      </div>
    );
  }

  // Don't render if already authenticated (redirect in progress)
  if (isAuthenticated) {
    return null;
  }

  const handleLogin = async (e: React.FormEvent) => {
    e.preventDefault();
    setIsLoading(true);

    try {
      const data = await apiLogin(loginData);

      // Use AuthProvider's login method to properly set state
      login(data.token, data.user as User);

      toast.success("登录成功");
      router.push("/");
    } catch (error) {
      console.error("Login error:", error);
      toast.error(error instanceof Error ? error.message : "登录失败");
    } finally {
      setIsLoading(false);
    }
  };

  // 验证邮箱格式
  const validateEmail = (email: string) => {
    const emailRegex = /^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$/;
    return emailRegex.test(email);
  };

  // 验证用户名格式
  const validateUsername = (username: string) => {
    // 支持小写字母、数字、下划线，6-20个字符
    const usernameRegex = /^[a-z0-9_]{6,20}$/;
    return usernameRegex.test(username);
  };

  // 验证密码格式
  const validatePassword = (password: string) => {
    // 必须包含字母和数字，可以包含大小写字母、符号
    const hasLetter = /[a-zA-Z]/.test(password);
    const hasNumber = /[0-9]/.test(password);
    return password.length >= 6 && hasLetter && hasNumber;
  };

  // 验证姓名格式（中文）
  const validateChineseName = (name: string) => {
    const chineseRegex = /^[\u4e00-\u9fa5]{2,10}$/;
    return chineseRegex.test(name);
  };

  // 计算密码匹配状态
  const getPasswordMatchStatus = () => {
    if (!registerData.password || !registerData.confirmPassword) {
      return null; // 没有输入时不显示状态
    }
    return registerData.password === registerData.confirmPassword;
  };

  // 获取字段验证状态
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

  // 获取字段错误消息
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

  const handleRegister = async (e: React.FormEvent) => {
    e.preventDefault();

    // 验证所有必填字段
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

    // 验证密码一致性
    if (registerData.password !== registerData.confirmPassword) {
      toast.error("两次输入的密码不一致");
      return;
    }

    setIsLoading(true);

    try {
      // 提取不包含confirmPassword的数据发送到后端
      const { confirmPassword, ...apiData } = registerData;
      const data = await apiRegister(apiData);

      // 注册成功，显示邮箱验证提醒
      toast.success(
        `注册成功！我们已向 ${data.email} 发送了验证邮件，请点击邮件中的链接验证您的邮箱，然后返回登录。`,
        {
          duration: 8000,
        }
      );

      // 清空注册表单
      setRegisterData({
        username: "",
        email: "",
        password: "",
        confirmPassword: "",
        fullName: "",
        companyId: "",
      });

      // 切换到登录选项卡
      const loginTab = document.querySelector('[value="login"]') as HTMLElement;
      if (loginTab) {
        loginTab.click();
      }
    } catch (error) {
      console.error("Register error:", error);
      toast.error(error instanceof Error ? error.message : "注册失败");
    } finally {
      setIsLoading(false);
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
                  使用您的账户信息登录系统
                </CardDescription>
              </CardHeader>
              <CardContent>
                <form onSubmit={handleLogin} className="space-y-4">
                  <div className="space-y-2">
                    <Label htmlFor="login-username">用户名</Label>
                    <Input
                      id="login-username"
                      type="text"
                      placeholder="请输入用户名"
                      value={loginData.username}
                      onChange={(e) =>
                        setLoginData({ ...loginData, username: e.target.value })
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
                      onChange={(e) =>
                        setRegisterData({ ...registerData, username: e.target.value })
                      }
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
                  </div>
                  <div className="space-y-2">
                    <Label htmlFor="register-email">邮箱 *</Label>
                    <Input
                      id="register-email"
                      type="email"
                      placeholder="请输入邮箱地址"
                      value={registerData.email}
                      onChange={(e) =>
                        setRegisterData({ ...registerData, email: e.target.value })
                      }
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
                    {/* 密码匹配状态提示 */}
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
      </div>
    </div>
  );
}