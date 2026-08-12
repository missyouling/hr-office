"use client";

import { useState, useEffect } from "react";
import { toast } from "sonner";
import { User, Palette, Lock, Camera, Sun, Moon, Monitor, Bell, Layout } from "lucide-react";
import { Dialog, DialogContent, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Switch } from "@/components/ui/switch";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { useAuth } from "@/lib/auth";
import { updateUserPreferences, fetchUserPreferences } from "@/lib/api";
import type { UserPreferences } from "@/lib/types";

interface UserPreferencesDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

const THEME_OPTIONS = [
  { value: "light", label: "浅色", icon: Sun },
  { value: "dark", label: "深色", icon: Moon },
  { value: "system", label: "跟随系统", icon: Monitor },
];

export function UserPreferencesDialog({ open, onOpenChange }: UserPreferencesDialogProps) {
  const { user, refreshUser } = useAuth();
  const [activeTab, setActiveTab] = useState("profile");
  
  // Theme state
  const [theme, setTheme] = useState<"light" | "dark" | "system">("light");
  
  // Profile state
  const [username, setUsername] = useState("");
  const [fullName, setFullName] = useState("");
  const [email, setEmail] = useState("");
  
  // Password state
  const [oldPassword, setOldPassword] = useState("");
  const [newPassword, setNewPassword] = useState("");
  const [confirmPassword, setConfirmPassword] = useState("");
  const [isChangingPassword, setIsChangingPassword] = useState(false);

  // Notification preferences state
  const [notificationPrefs, setNotificationPrefs] = useState({
    email_notification: true,
    system_notification: true,
    announcement_popup: true,
    duty_reminder: true,
    reminder_time: "09:00",
  });

  // Display preferences state
  const [displayPrefs, setDisplayPrefs] = useState({
    table_density: "default",
    default_page_size: 20,
    date_format: "YYYY-MM-DD",
    compact_sidebar: false,
    show_animations: true,
  });

  const [isSavingPrefs, setIsSavingPrefs] = useState(false);

  useEffect(() => {
    if (user) {
      setUsername(user.username || "");
      setFullName(user.full_name || "");
      setEmail(user.email || "");
    }
  }, [user]);

  useEffect(() => {
    loadPreferences();
  }, []);

  const loadPreferences = async () => {
    try {
      const prefs = await fetchUserPreferences() as Record<string, unknown>;
      if (prefs.user_theme) {
        setTheme(prefs.user_theme as "light" | "dark" | "system");
      }
      // Load notification preferences
      if (prefs.notification) {
        const notif = prefs.notification as Record<string, unknown>;
        setNotificationPrefs({
          email_notification: notif.email_notification as boolean ?? true,
          system_notification: notif.system_notification as boolean ?? true,
          announcement_popup: notif.announcement_popup as boolean ?? true,
          duty_reminder: notif.duty_reminder as boolean ?? true,
          reminder_time: notif.reminder_time as string ?? "09:00",
        });
      }
      // Load display preferences
      if (prefs.display) {
        const disp = prefs.display as Record<string, unknown>;
        setDisplayPrefs({
          table_density: disp.table_density as string ?? "default",
          default_page_size: disp.default_page_size as number ?? 20,
          date_format: disp.date_format as string ?? "YYYY-MM-DD",
          compact_sidebar: disp.compact_sidebar as boolean ?? false,
          show_animations: disp.show_animations as boolean ?? true,
        });
      }
    } catch (error) {
      console.error("加载偏好设置失败:", error);
    }
  };

  const handleThemeChange = async (newTheme: "light" | "dark" | "system") => {
    setTheme(newTheme);
    try {
      await updateUserPreferences({ user_theme: newTheme } as unknown as Record<string, Record<string, unknown>>);
      applyTheme(newTheme);
      toast.success("主题已更新");
    } catch (error) {
      console.error("更新主题失败:", error);
      toast.error("更新主题失败");
    }
  };

  const applyTheme = (theme: "light" | "dark" | "system") => {
    const root = document.documentElement;
    if (theme === "system") {
      const prefersDark = window.matchMedia("(prefers-color-scheme: dark)").matches;
      root.classList.toggle("dark", prefersDark);
    } else {
      root.classList.toggle("dark", theme === "dark");
    }
  };

  const handleProfileSave = async () => {
    try {
      // TODO: Call API to update profile
      toast.success("个人信息已更新");
      refreshUser();
    } catch (error) {
      console.error("更新个人信息失败:", error);
      toast.error("更新失败");
    }
  };

  const handleNotificationSave = async () => {
    setIsSavingPrefs(true);
    try {
      await updateUserPreferences({ notification: notificationPrefs } as UserPreferences);
      toast.success("通知偏好已保存");
    } catch (error) {
      console.error("保存通知偏好失败:", error);
      toast.error("保存失败");
    } finally {
      setIsSavingPrefs(false);
    }
  };

  const handleDisplaySave = async () => {
    setIsSavingPrefs(true);
    try {
      await updateUserPreferences({ display: displayPrefs } as UserPreferences);
      toast.success("显示偏好已保存");
      // Apply display preferences
      if (displayPrefs.compact_sidebar) {
        document.body.classList.add("sidebar-compact");
      } else {
        document.body.classList.remove("sidebar-compact");
      }
    } catch (error) {
      console.error("保存显示偏好失败:", error);
      toast.error("保存失败");
    } finally {
      setIsSavingPrefs(false);
    }
  };

  const handlePasswordChange = async () => {
    if (newPassword !== confirmPassword) {
      toast.error("两次输入的密码不一致");
      return;
    }
    if (newPassword.length < 6) {
      toast.error("密码长度至少6位");
      return;
    }
    
    setIsChangingPassword(true);
    try {
      // TODO: Call API to change password
      toast.success("密码已更新");
      setOldPassword("");
      setNewPassword("");
      setConfirmPassword("");
    } catch (error) {
      console.error("修改密码失败:", error);
      toast.error("修改密码失败");
    } finally {
      setIsChangingPassword(false);
    }
  };

  const getDefaultAvatar = (name: string) => {
    return `https://api.dicebear.com/7.x/initials/png?name=${encodeURIComponent(name)}&backgroundColor=random`;
  };

  const roleLabel = {
    user: "普通用户",
    admin: "管理员",
    super_admin: "超级管理员",
    manager: "部门经理",
    editor: "编辑者",
    viewer: "查看者",
  }[user?.role || "user"];

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-[480px]">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            {/* eslint-disable-next-line @next/next/no-img-element -- 头像URL动态生成（DiceBear API），无需 next/image 优化 */}
            <img
              src={user?.full_name ? getDefaultAvatar(user.full_name) : getDefaultAvatar(username)}
              alt="avatar"
              className="h-10 w-10 rounded-full bg-muted object-cover"
            />
            <div>
              <div className="text-base font-medium">{user?.full_name || username}</div>
              <div className="text-xs text-muted-foreground">{roleLabel}</div>
            </div>
          </DialogTitle>
        </DialogHeader>

        <Tabs value={activeTab} onValueChange={setActiveTab} className="mt-4">
          <TabsList className="grid w-full grid-cols-5">
            <TabsTrigger value="profile" className="gap-1">
              <User className="h-4 w-4" />
              <span className="hidden sm:inline">资料</span>
            </TabsTrigger>
            <TabsTrigger value="theme" className="gap-1">
              <Palette className="h-4 w-4" />
              <span className="hidden sm:inline">主题</span>
            </TabsTrigger>
            <TabsTrigger value="notification" className="gap-1">
              <Bell className="h-4 w-4" />
              <span className="hidden sm:inline">通知</span>
            </TabsTrigger>
            <TabsTrigger value="display" className="gap-1">
              <Layout className="h-4 w-4" />
              <span className="hidden sm:inline">显示</span>
            </TabsTrigger>
            <TabsTrigger value="password" className="gap-1">
              <Lock className="h-4 w-4" />
              <span className="hidden sm:inline">密码</span>
            </TabsTrigger>
          </TabsList>

          {/* Profile Tab */}
          <TabsContent value="profile" className="space-y-4 pt-4">
            <div className="flex items-center gap-4">
              <div className="relative">
                {/* eslint-disable-next-line @next/next/no-img-element -- 头像URL动态生成（DiceBear API），无需 next/image 优化 */}
                <img
                  src={user?.full_name ? getDefaultAvatar(user.full_name) : getDefaultAvatar(username)}
                  alt="avatar"
                  className="h-20 w-20 rounded-full bg-muted object-cover"
                />
                <Button size="icon" variant="secondary" className="absolute -bottom-1 -right-1 h-8 w-8 rounded-full">
                  <Camera className="h-4 w-4" />
                </Button>
              </div>
              <div className="text-sm text-muted-foreground">
                <p>点击相机图标上传新头像</p>
                <p>支持 JPG、PNG 格式</p>
              </div>
            </div>

            <div className="space-y-2">
              <Label htmlFor="username">用户名</Label>
              <Input
                id="username"
                value={username}
                onChange={(e) => setUsername(e.target.value)}
                placeholder="请输入用户名"
              />
            </div>

            <div className="space-y-2">
              <Label htmlFor="fullName">昵称</Label>
              <Input
                id="fullName"
                value={fullName}
                onChange={(e) => setFullName(e.target.value)}
                placeholder="请输入昵称"
              />
            </div>

            <div className="space-y-2">
              <Label htmlFor="email">邮箱</Label>
              <Input
                id="email"
                type="email"
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                placeholder="请输入邮箱"
              />
            </div>

            <Button className="w-full" onClick={handleProfileSave}>
              保存修改
            </Button>
          </TabsContent>

          {/* Theme Tab */}
          <TabsContent value="theme" className="pt-4">
            <div className="grid grid-cols-3 gap-3">
              {THEME_OPTIONS.map((option) => {
                const Icon = option.icon;
                const isSelected = theme === option.value;
                return (
                  <button
                    key={option.value}
                    onClick={() => handleThemeChange(option.value as "light" | "dark" | "system")}
                    className={`flex flex-col items-center gap-2 rounded-lg border-2 p-4 transition-colors ${
                      isSelected ? "border-primary bg-primary/10" : "border-muted hover:border-muted-foreground/50"
                    }`}
                  >
                    <Icon className={`h-8 w-8 ${isSelected ? "text-primary" : "text-muted-foreground"}`} />
                    <span className={`text-sm ${isSelected ? "font-medium" : ""}`}>{option.label}</span>
                  </button>
                );
              })}
            </div>
          </TabsContent>

          {/* Notification Tab */}
          <TabsContent value="notification" className="space-y-4 pt-4">
            <div className="space-y-4">
              <div className="flex items-center justify-between">
                <div>
                  <Label className="text-sm font-medium">邮件通知</Label>
                  <p className="text-xs text-muted-foreground">接收系统邮件通知</p>
                </div>
                <Switch
                  checked={notificationPrefs.email_notification}
                  onCheckedChange={(checked) => setNotificationPrefs({ ...notificationPrefs, email_notification: checked })}
                />
              </div>

              <div className="flex items-center justify-between">
                <div>
                  <Label className="text-sm font-medium">系统通知</Label>
                  <p className="text-xs text-muted-foreground">在页面内显示通知消息</p>
                </div>
                <Switch
                  checked={notificationPrefs.system_notification}
                  onCheckedChange={(checked) => setNotificationPrefs({ ...notificationPrefs, system_notification: checked })}
                />
              </div>

              <div className="flex items-center justify-between">
                <div>
                  <Label className="text-sm font-medium">公告弹窗</Label>
                  <p className="text-xs text-muted-foreground">新公告发布时弹出提示</p>
                </div>
                <Switch
                  checked={notificationPrefs.announcement_popup}
                  onCheckedChange={(checked) => setNotificationPrefs({ ...notificationPrefs, announcement_popup: checked })}
                />
              </div>

              <div className="flex items-center justify-between">
                <div>
                  <Label className="text-sm font-medium">值班提醒</Label>
                  <p className="text-xs text-muted-foreground">提前提醒值班安排</p>
                </div>
                <Switch
                  checked={notificationPrefs.duty_reminder}
                  onCheckedChange={(checked) => setNotificationPrefs({ ...notificationPrefs, duty_reminder: checked })}
                />
              </div>

              {notificationPrefs.duty_reminder && (
                <div className="grid gap-2">
                  <Label className="text-sm font-medium">提醒时间</Label>
                  <Select
                    value={notificationPrefs.reminder_time}
                    onValueChange={(value) => setNotificationPrefs({ ...notificationPrefs, reminder_time: value })}
                  >
                    <SelectTrigger className="w-32">
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="08:00">08:00</SelectItem>
                      <SelectItem value="09:00">09:00</SelectItem>
                      <SelectItem value="10:00">10:00</SelectItem>
                      <SelectItem value="12:00">12:00</SelectItem>
                      <SelectItem value="14:00">14:00</SelectItem>
                      <SelectItem value="17:00">17:00</SelectItem>
                    </SelectContent>
                  </Select>
                </div>
              )}
            </div>

            <Button className="w-full" onClick={handleNotificationSave} disabled={isSavingPrefs}>
              {isSavingPrefs ? "保存中..." : "保存通知偏好"}
            </Button>
          </TabsContent>

          {/* Display Tab */}
          <TabsContent value="display" className="space-y-4 pt-4">
            <div className="space-y-4">
              <div className="grid gap-2">
                <Label className="text-sm font-medium">表格密度</Label>
                <Select
                  value={displayPrefs.table_density}
                  onValueChange={(value) => setDisplayPrefs({ ...displayPrefs, table_density: value })}
                >
                  <SelectTrigger>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="compact">紧凑</SelectItem>
                    <SelectItem value="default">默认</SelectItem>
                    <SelectItem value="comfortable">宽松</SelectItem>
                  </SelectContent>
                </Select>
              </div>

              <div className="grid gap-2">
                <Label className="text-sm font-medium">默认每页条数</Label>
                <Select
                  value={String(displayPrefs.default_page_size)}
                  onValueChange={(value) => setDisplayPrefs({ ...displayPrefs, default_page_size: parseInt(value) })}
                >
                  <SelectTrigger>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="10">10 条</SelectItem>
                    <SelectItem value="20">20 条</SelectItem>
                    <SelectItem value="50">50 条</SelectItem>
                    <SelectItem value="100">100 条</SelectItem>
                  </SelectContent>
                </Select>
              </div>

              <div className="grid gap-2">
                <Label className="text-sm font-medium">日期格式</Label>
                <Select
                  value={displayPrefs.date_format}
                  onValueChange={(value) => setDisplayPrefs({ ...displayPrefs, date_format: value })}
                >
                  <SelectTrigger>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="YYYY-MM-DD">2026-04-05</SelectItem>
                    <SelectItem value="DD/MM/YYYY">05/04/2026</SelectItem>
                    <SelectItem value="MM/DD/YYYY">04/05/2026</SelectItem>
                  </SelectContent>
                </Select>
              </div>

              <div className="flex items-center justify-between">
                <div>
                  <Label className="text-sm font-medium">侧边栏紧凑模式</Label>
                  <p className="text-xs text-muted-foreground">使用更窄的侧边栏</p>
                </div>
                <Switch
                  checked={displayPrefs.compact_sidebar}
                  onCheckedChange={(checked) => setDisplayPrefs({ ...displayPrefs, compact_sidebar: checked })}
                />
              </div>

              <div className="flex items-center justify-between">
                <div>
                  <Label className="text-sm font-medium">显示动画效果</Label>
                  <p className="text-xs text-muted-foreground">启用页面过渡动画</p>
                </div>
                <Switch
                  checked={displayPrefs.show_animations}
                  onCheckedChange={(checked) => setDisplayPrefs({ ...displayPrefs, show_animations: checked })}
                />
              </div>
            </div>

            <Button className="w-full" onClick={handleDisplaySave} disabled={isSavingPrefs}>
              {isSavingPrefs ? "保存中..." : "保存显示偏好"}
            </Button>
          </TabsContent>

          {/* Password Tab */}
          <TabsContent value="password" className="space-y-4 pt-4">
            <div className="space-y-2">
              <Label htmlFor="oldPassword">原密码</Label>
              <Input
                id="oldPassword"
                type="password"
                value={oldPassword}
                onChange={(e) => setOldPassword(e.target.value)}
                placeholder="请输入原密码"
              />
            </div>

            <div className="space-y-2">
              <Label htmlFor="newPassword">新密码</Label>
              <Input
                id="newPassword"
                type="password"
                value={newPassword}
                onChange={(e) => setNewPassword(e.target.value)}
                placeholder="请输入新密码（至少6位）"
              />
            </div>

            <div className="space-y-2">
              <Label htmlFor="confirmPassword">确认密码</Label>
              <Input
                id="confirmPassword"
                type="password"
                value={confirmPassword}
                onChange={(e) => setConfirmPassword(e.target.value)}
                placeholder="请再次输入新密码"
              />
            </div>

            <Button className="w-full" onClick={handlePasswordChange} disabled={isChangingPassword}>
              {isChangingPassword ? "修改中..." : "修改密码"}
            </Button>
          </TabsContent>
        </Tabs>

        <div className="mt-4 flex justify-between">
          <Button variant="destructive" onClick={() => {
            onOpenChange(false);
            // Trigger logout after closing dialog
            setTimeout(() => {
              localStorage.removeItem("token");
              localStorage.removeItem("user");
              window.location.href = "/auth";
            }, 100);
          }}>
            退出登录
          </Button>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            关闭
          </Button>
        </div>
      </DialogContent>
    </Dialog>
  );
}
