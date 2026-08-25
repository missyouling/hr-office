"use client";

import { useEffect, useState } from "react";
import { ArrowLeft, LockKeyhole, Monitor, Moon, ShieldCheck, Sun, UserRound } from "lucide-react";
import { useTheme } from "next-themes";
import { toast } from "sonner";
import { changePassword, fetchUserPreferences, updateUserPreferences, updateUserProfile } from "@/lib/api";
import { useAuth } from "@/lib/auth";
import { UserAvatar } from "@/components/avatar/user-avatar";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";

const MIN_PASSWORD_LENGTH = 6;

export function PersonalSettings({ onBack }: { onBack?: () => void }) {
  const { user, refreshUser, logout } = useAuth();
  const [fullName, setFullName] = useState("");
  const [currentPassword, setCurrentPassword] = useState("");
  const [newPassword, setNewPassword] = useState("");
  const [confirmPassword, setConfirmPassword] = useState("");
  const [isSavingProfile, setIsSavingProfile] = useState(false);
  const [isChangingPassword, setIsChangingPassword] = useState(false);
  const { theme, setTheme } = useTheme();

  useEffect(() => {
    setFullName(user?.full_name ?? "");
  }, [user?.full_name]);

  useEffect(() => {
    fetchUserPreferences().then((preferences) => {
      const storedTheme = preferences.user_theme;
      if (storedTheme === "light" || storedTheme === "dark" || storedTheme === "system") setTheme(storedTheme);
    }).catch(() => {
      // 偏好接口不可用时由 next-themes 使用本地存储中的既有主题。
    });
  }, [setTheme]);

  const handleThemeChange = async (nextTheme: "light" | "dark" | "system") => {
    setTheme(nextTheme);
    try {
      await updateUserPreferences({ user_theme: nextTheme });
    } catch {
      toast.message("外观已在本设备保存，稍后会同步到账号偏好");
    }
  };

  const handleProfileSubmit = async (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    const normalizedName = fullName.trim();
    if (!normalizedName) {
      toast.error("请输入姓名");
      return;
    }

    setIsSavingProfile(true);
    try {
      await updateUserProfile(normalizedName);
      await refreshUser();
      toast.success("个人资料已保存");
    } catch (error) {
      console.error("更新个人资料失败:", error);
      toast.error("个人资料保存失败，请稍后重试");
    } finally {
      setIsSavingProfile(false);
    }
  };

  const handlePasswordSubmit = async (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (!currentPassword) {
      toast.error("请输入当前密码");
      return;
    }
    if (newPassword.length < MIN_PASSWORD_LENGTH) {
      toast.error(`新密码至少需要 ${MIN_PASSWORD_LENGTH} 位`);
      return;
    }
    if (newPassword !== confirmPassword) {
      toast.error("两次输入的新密码不一致");
      return;
    }

    setIsChangingPassword(true);
    try {
      await changePassword(currentPassword, newPassword);
      setCurrentPassword("");
      setNewPassword("");
      setConfirmPassword("");
      toast.success("密码已修改，请重新登录");
      // 密码变更后统一走 logout，确保服务端会话与本地缓存一致失效
      logout();
    } catch (error) {
      console.error("修改密码失败:", error);
      toast.error("密码修改失败，请确认当前密码后重试");
    } finally {
      setIsChangingPassword(false);
    }
  };

  return (
    <section className="mx-auto w-full max-w-5xl space-y-6 pb-8">
      <header className="relative overflow-hidden rounded-3xl border bg-card px-6 py-7 shadow-sm md:px-8">
        <div className="absolute -right-12 -top-16 h-44 w-44 rounded-full bg-primary/10 blur-3xl" />
        <div className="relative flex flex-col gap-4 sm:flex-row sm:items-center">
          <UserAvatar name={user?.full_name || user?.username || "用户"} className="h-16 w-16 rounded-2xl" />
          <div>
            <p className="text-sm font-medium text-primary">账户中心</p>
            <h1 className="mt-1 text-2xl font-semibold tracking-tight">个人设置</h1>
            <p className="mt-1 text-sm text-muted-foreground">管理你的公开资料与登录安全。</p>
          </div>
          {onBack && (
            <Button variant="outline" size="sm" onClick={onBack} className="sm:ml-auto">
              <ArrowLeft className="h-4 w-4" />
              返回
            </Button>
          )}
        </div>
      </header>

      <Tabs defaultValue="account">
        <TabsList aria-label="个人资料设置标签">
          <TabsTrigger value="account">账户资料</TabsTrigger>
          <TabsTrigger value="appearance">外观偏好</TabsTrigger>
        </TabsList>
        <TabsContent value="account" className="space-y-6">
          <div className="grid gap-6 lg:grid-cols-2">
            <Card className="border-border/80 shadow-sm">
              <CardHeader><div className="flex items-center gap-3"><span className="rounded-xl bg-primary/10 p-2.5 text-primary"><UserRound className="h-5 w-5" /></span><div><CardTitle>个人资料</CardTitle><CardDescription>姓名会显示在系统的账户入口中。</CardDescription></div></div></CardHeader>
              <CardContent><form className="space-y-5" onSubmit={handleProfileSubmit}><div className="space-y-2"><Label htmlFor="profile-username">用户名</Label><Input id="profile-username" value={user?.username ?? ""} disabled /></div><div className="space-y-2"><Label htmlFor="profile-email">邮箱</Label><Input id="profile-email" value={user?.email ?? ""} disabled /></div><div className="space-y-2"><Label htmlFor="profile-full-name">姓名</Label><Input id="profile-full-name" value={fullName} onChange={(event) => setFullName(event.target.value)} placeholder="请输入姓名" maxLength={100} required /></div><Button type="submit" disabled={isSavingProfile}>{isSavingProfile ? "保存中…" : "保存资料"}</Button></form></CardContent>
            </Card>
            <Card className="border-border/80 shadow-sm">
              <CardHeader><div className="flex items-center gap-3"><span className="rounded-xl bg-amber-500/10 p-2.5 text-amber-600 dark:text-amber-400"><LockKeyhole className="h-5 w-5" /></span><div><CardTitle>登录密码</CardTitle><CardDescription>使用至少 6 位的新密码保护账户安全。</CardDescription></div></div></CardHeader>
              <CardContent><form className="space-y-5" onSubmit={handlePasswordSubmit}><div className="space-y-2"><Label htmlFor="current-password">当前密码</Label><Input id="current-password" type="password" value={currentPassword} onChange={(event) => setCurrentPassword(event.target.value)} autoComplete="current-password" required /></div><div className="space-y-2"><Label htmlFor="new-password">新密码</Label><Input id="new-password" type="password" value={newPassword} onChange={(event) => setNewPassword(event.target.value)} autoComplete="new-password" minLength={MIN_PASSWORD_LENGTH} required /></div><div className="space-y-2"><Label htmlFor="confirm-password">确认新密码</Label><Input id="confirm-password" type="password" value={confirmPassword} onChange={(event) => setConfirmPassword(event.target.value)} autoComplete="new-password" minLength={MIN_PASSWORD_LENGTH} required /></div><Button type="submit" variant="secondary" disabled={isChangingPassword}>{isChangingPassword ? "修改中…" : "修改密码"}</Button></form></CardContent>
            </Card>
          </div>
          <div className="flex items-center gap-2 px-1 text-sm text-muted-foreground"><ShieldCheck className="h-4 w-4 text-primary" />资料和密码均会在提交成功后提示结果。</div>
        </TabsContent>
        <TabsContent value="appearance">
          <Card className="max-w-2xl border-border/80 shadow-sm">
            <CardHeader><div className="flex items-center gap-3"><span className="rounded-xl bg-primary/10 p-2.5 text-primary"><Monitor className="h-5 w-5" /></span><div><CardTitle>外观偏好</CardTitle><CardDescription>选择浅色、深色或跟随设备系统设置。</CardDescription></div></div></CardHeader>
            <CardContent><div className="grid grid-cols-3 gap-2" role="group" aria-label="外观主题"><ThemeButton active={theme === "light"} icon={<Sun className="h-4 w-4" />} label="浅色" onClick={() => void handleThemeChange("light")} /><ThemeButton active={theme === "dark"} icon={<Moon className="h-4 w-4" />} label="深色" onClick={() => void handleThemeChange("dark")} /><ThemeButton active={theme === "system" || !theme} icon={<Monitor className="h-4 w-4" />} label="跟随系统" onClick={() => void handleThemeChange("system")} /></div></CardContent>
          </Card>
        </TabsContent>
      </Tabs>
    </section>
  );
}

function ThemeButton({ active, icon, label, onClick }: { active: boolean; icon: React.ReactNode; label: string; onClick: () => void }) {
  return <Button type="button" variant={active ? "default" : "outline"} className="h-20 flex-col gap-2" onClick={onClick}>{icon}{label}</Button>;
}
