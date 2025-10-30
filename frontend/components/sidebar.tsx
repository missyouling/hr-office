"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { User, Settings, LogOut, Calculator, Database, BarChart3, Users } from "lucide-react";
import { useAuth } from "@/lib/auth";
import { toast } from "sonner";

import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { changePassword } from "@/lib/api";

interface SidebarProps {
  currentView: string;
  onViewChange: (view: string) => void;
}

interface ChangePasswordData {
  currentPassword: string;
  newPassword: string;
  confirmPassword: string;
}

export function Sidebar({ currentView, onViewChange }: SidebarProps) {
  const { user, logout } = useAuth();
  const router = useRouter();
  const [showUserCenter, setShowUserCenter] = useState(false);
  const [showChangePassword, setShowChangePassword] = useState(false);
  const [isChangingPassword, setIsChangingPassword] = useState(false);
  const [passwordData, setPasswordData] = useState<ChangePasswordData>({
    currentPassword: "",
    newPassword: "",
    confirmPassword: "",
  });

  const menuItems = [
    {
      id: "employee",
      label: "员工管理",
      icon: Users,
      description: "员工花名册和档案管理",
      category: "人事管理",
    },
    {
      id: "insurance",
      label: "社保整合",
      icon: Calculator,
      description: "社保数据处理和整合",
      category: "人事管理",
    },
    // 未来可以添加更多人事管理模块
    // {
    //   id: "hr",
    //   label: "人事管理",
    //   icon: Users,
    //   description: "员工信息管理",
    //   category: "人事管理",
    // },
    // {
    //   id: "salary",
    //   label: "薪资管理",
    //   icon: CreditCard,
    //   description: "薪资计算和发放",
    //   category: "人事管理",
    // },
    // {
    //   id: "attendance",
    //   label: "考勤管理",
    //   icon: Clock,
    //   description: "考勤打卡和统计",
    //   category: "人事管理",
    // },
    ...(user?.username === "admin" ? [
      {
        id: "organization",
        label: "组织机构",
        icon: Settings,
        description: "集团公司组织架构管理",
        category: "系统管理",
      },
      {
        id: "audit",
        label: "审计日志",
        icon: BarChart3,
        description: "查看系统操作日志",
        category: "系统管理",
      },
      {
        id: "monitoring",
        label: "系统监控",
        icon: Database,
        description: "系统状态和性能监控",
        category: "系统管理",
      },
    ] : []),
  ];

  const handleLogout = async () => {
    try {
      await logout();
      toast.success("已退出登录");
      router.push("/auth");
    } catch (error) {
      console.error("Logout error:", error);
      toast.error("退出登录失败");
    }
  };

  const handleChangePassword = async (e: React.FormEvent) => {
    e.preventDefault();

    if (passwordData.newPassword !== passwordData.confirmPassword) {
      toast.error("两次输入的新密码不一致");
      return;
    }

    if (passwordData.newPassword.length < 6) {
      toast.error("新密码至少6位字符");
      return;
    }

    setIsChangingPassword(true);

    try {
      await changePassword(passwordData.currentPassword, passwordData.newPassword);

      toast.success("密码修改成功");
      setShowChangePassword(false);
      setPasswordData({
        currentPassword: "",
        newPassword: "",
        confirmPassword: "",
      });
    } catch (error) {
      console.error("Change password error:", error);
      toast.error(error instanceof Error ? error.message : "密码修改失败");
    } finally {
      setIsChangingPassword(false);
    }
  };

  return (
    <div className="fixed top-0 left-0 w-64 h-screen bg-muted/30 border-r flex flex-col z-50">
      {/* Header */}
      <div className="p-6 border-b">
        <h1 className="text-lg font-bold">人事行政管理系统</h1>
        <p className="text-sm text-muted-foreground mt-1">
          欢迎，{user?.full_name || user?.username}
        </p>
      </div>

      {/* Navigation Menu */}
      <div className="flex-1 p-4 space-y-2">
        {menuItems.map((item) => {
          const Icon = item.icon;
          const isActive = currentView === item.id;

          return (
            <Button
              key={item.id}
              variant={isActive ? "default" : "ghost"}
              className="w-full justify-start h-auto p-3"
              onClick={() => onViewChange(item.id)}
            >
              <Icon className="h-4 w-4 mr-3" />
              <div className="text-left">
                <div className="font-medium">{item.label}</div>
                <div className="text-xs text-muted-foreground">
                  {item.description}
                </div>
              </div>
            </Button>
          );
        })}
      </div>

      {/* User Center */}
      <div className="p-4 border-t">
        <Dialog open={showUserCenter} onOpenChange={setShowUserCenter}>
          <DialogTrigger asChild>
            <Button variant="ghost" className="w-full justify-start">
              <User className="h-4 w-4 mr-3" />
              用户中心
            </Button>
          </DialogTrigger>
          <DialogContent className="sm:max-w-md">
            <DialogHeader>
              <DialogTitle>用户中心</DialogTitle>
              <DialogDescription>
                管理您的账户信息和设置
              </DialogDescription>
            </DialogHeader>

            <div className="space-y-4">
              {/* User Info */}
              <Card>
                <CardHeader className="pb-3">
                  <CardTitle className="text-sm">用户信息</CardTitle>
                </CardHeader>
                <CardContent className="space-y-2">
                  <div className="flex justify-between">
                    <span className="text-sm text-muted-foreground">用户名:</span>
                    <span className="text-sm">{user?.username}</span>
                  </div>
                  <div className="flex justify-between">
                    <span className="text-sm text-muted-foreground">邮箱:</span>
                    <span className="text-sm">{user?.email}</span>
                  </div>
                  <div className="flex justify-between">
                    <span className="text-sm text-muted-foreground">姓名:</span>
                    <span className="text-sm">{user?.full_name || "未设置"}</span>
                  </div>
                </CardContent>
              </Card>

              {/* Actions */}
              <div className="space-y-2">
                <Dialog open={showChangePassword} onOpenChange={setShowChangePassword}>
                  <DialogTrigger asChild>
                    <Button variant="outline" className="w-full">
                      <Settings className="h-4 w-4 mr-2" />
                      修改密码
                    </Button>
                  </DialogTrigger>
                  <DialogContent className="sm:max-w-md">
                    <DialogHeader>
                      <DialogTitle>修改密码</DialogTitle>
                      <DialogDescription>
                        请输入当前密码和新密码
                      </DialogDescription>
                    </DialogHeader>
                    <form onSubmit={handleChangePassword} className="space-y-4">
                      <div className="space-y-2">
                        <Label htmlFor="current-password">当前密码</Label>
                        <Input
                          id="current-password"
                          type="password"
                          value={passwordData.currentPassword}
                          onChange={(e) =>
                            setPasswordData({ ...passwordData, currentPassword: e.target.value })
                          }
                          required
                        />
                      </div>
                      <div className="space-y-2">
                        <Label htmlFor="new-password">新密码</Label>
                        <Input
                          id="new-password"
                          type="password"
                          value={passwordData.newPassword}
                          onChange={(e) =>
                            setPasswordData({ ...passwordData, newPassword: e.target.value })
                          }
                          required
                        />
                      </div>
                      <div className="space-y-2">
                        <Label htmlFor="confirm-password">确认新密码</Label>
                        <Input
                          id="confirm-password"
                          type="password"
                          value={passwordData.confirmPassword}
                          onChange={(e) =>
                            setPasswordData({ ...passwordData, confirmPassword: e.target.value })
                          }
                          required
                        />
                      </div>
                      <DialogFooter>
                        <Button
                          type="button"
                          variant="outline"
                          onClick={() => setShowChangePassword(false)}
                        >
                          取消
                        </Button>
                        <Button type="submit" disabled={isChangingPassword}>
                          {isChangingPassword ? "修改中..." : "确认修改"}
                        </Button>
                      </DialogFooter>
                    </form>
                  </DialogContent>
                </Dialog>

                <Button
                  variant="outline"
                  className="w-full text-destructive hover:text-destructive"
                  onClick={handleLogout}
                >
                  <LogOut className="h-4 w-4 mr-2" />
                  退出登录
                </Button>
              </div>
            </div>
          </DialogContent>
        </Dialog>
      </div>
    </div>
  );
}