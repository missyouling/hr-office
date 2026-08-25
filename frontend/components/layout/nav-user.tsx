"use client";

import { useState } from "react";
import { ChevronUp, LogOut, MessageSquareText, Settings, UserRound } from "lucide-react";
import { UserAvatar } from "@/components/avatar/user-avatar";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { ProfileDialog } from "@/components/layout/profile-dialog";
import { SidebarGroup, SidebarMenu, SidebarMenuItem, useSidebar } from "@/components/ui/sidebar";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { useAuth } from "@/lib/auth";

export type SettingsMode = "personal" | "system";

interface NavUserProps {
  displayName: string;
  subLine?: string;
  onOpenSettings: (mode: SettingsMode) => void;
  onOpenFeedback?: () => void;
  variant?: "default" | "new";
}

export function NavUser({ displayName, subLine, onOpenSettings, onOpenFeedback, variant = "default" }: NavUserProps) {
  const { state, isMobile, setOpenMobile } = useSidebar();
  const { user, hasPermission, logout } = useAuth();
  const isCollapsed = state === "collapsed" && !isMobile;
  const avatarName = user?.full_name || displayName || "用户";
  const canViewSystemSettings = hasPermission("settings", "view");
  const isNewShell = variant === "new";
  // 新壳"个人资料"模态对话框状态：弹窗展示，不再派发 personal 视图切换
  const [isProfileOpen, setIsProfileOpen] = useState(false);

  const handleOpenSettings = (mode: SettingsMode) => {
    onOpenSettings(mode);
    if (isMobile) setOpenMobile(false);
  };

  const handleLogout = () => {
    if (isMobile) setOpenMobile(false);
    logout();
  };

  const trigger = (
    <button
      type="button"
      aria-label={isCollapsed ? "打开账户菜单" : "打开账户菜单：个人信息"}
      className={isNewShell
        // w-full：撑满侧栏 footer 内容区（≈224px），与 DropdownMenuContent(w-56) 同宽同位，
        // 选中态高亮矩形与弹出菜单视觉对齐（原生 button 收缩适配内容会窄 ~41px，高亮偏左不对称）
        ? "group flex h-12 w-full min-w-0 items-center rounded-xl px-3 text-left outline-none transition-colors hover:bg-sidebar-accent focus-visible:bg-sidebar-accent data-[state=open]:bg-sidebar-accent"
        : "group flex h-12 min-w-0 items-center rounded-lg px-3 text-left outline-none transition-colors hover:bg-muted focus-visible:bg-muted data-[state=open]:bg-muted"}
    >
      <UserAvatar name={avatarName} alt={`${avatarName}的头像`} className="h-8 w-8 shrink-0 rounded-full" />
      {!isCollapsed && (
        <>
          <span className="ml-2 min-w-0 flex-1">
            <span className="block truncate text-sm font-medium leading-tight">{displayName || "未登录"}</span>
            {subLine && <span className="mt-0.5 block truncate text-[11px] leading-tight text-muted-foreground">{subLine}</span>}
          </span>
          <ChevronUp className="ml-2 h-4 w-4 shrink-0 text-muted-foreground transition-transform group-data-[state=open]:rotate-180" aria-hidden />
        </>
      )}
    </button>
  );

  return (
    // p-0 收敛 SidebarFooter(p-2) 内嵌 SidebarGroup(默认 p-2) 的双层内边距，
    // 避免头像行可用宽度被压缩；点击域由触发按钮 px-3 补足
    <SidebarGroup className="p-0">
      <SidebarMenu>
        <SidebarMenuItem>
          <DropdownMenu>
            {isCollapsed ? (
              <Tooltip>
                <TooltipTrigger asChild>
                  <DropdownMenuTrigger asChild>{trigger}</DropdownMenuTrigger>
                </TooltipTrigger>
                <TooltipContent side="right">{subLine ? `${displayName} · ${subLine}` : displayName}</TooltipContent>
              </Tooltip>
            ) : (
              <DropdownMenuTrigger asChild>{trigger}</DropdownMenuTrigger>
            )}
            <DropdownMenuContent side="top" align="start" sideOffset={8} className={isNewShell ? "w-56 rounded-2xl border-border/80 p-1.5 shadow-xl" : "w-52 rounded-xl border-border/80 p-1.5 shadow-lg"}>
              {/* 信息头：非交互区块，参照新壳视觉规范（姓名加粗 + subLine 小号弱化） */}
              <div className="px-2.5 py-2" data-slot="nav-user-info">
                <p className="truncate text-[13px] font-semibold text-foreground">{displayName || "未登录"}</p>
                {subLine && <p className="mt-0.5 truncate text-[11px] text-muted-foreground">{subLine}</p>}
              </div>
              <DropdownMenuSeparator />
              {/* 新壳：个人资料走模态对话框（ProfileDialog 自持关闭）；旧壳：保留 personal 视图切换 */}
              {isNewShell ? (
                <DropdownMenuItem className="gap-2 rounded-lg px-2.5 py-2" onSelect={() => { setIsProfileOpen(true); if (isMobile) setOpenMobile(false); }}>
                  <UserRound className="h-4 w-4" aria-hidden />
                  个人资料
                </DropdownMenuItem>
              ) : (
                <DropdownMenuItem className="gap-2 rounded-lg px-2.5 py-2" onSelect={() => handleOpenSettings("personal")}>
                  <UserRound className="h-4 w-4" aria-hidden />
                  个人信息
                </DropdownMenuItem>
              )}
              {isNewShell && onOpenFeedback && (
                <DropdownMenuItem className="gap-2 rounded-lg px-2.5 py-2" onSelect={onOpenFeedback}>
                  <MessageSquareText className="h-4 w-4" aria-hidden />
                  我的反馈
                </DropdownMenuItem>
              )}
              {canViewSystemSettings && (
                <DropdownMenuItem className="gap-2 rounded-lg px-2.5 py-2" onSelect={() => handleOpenSettings("system")}>
                  <Settings className="h-4 w-4" aria-hidden />
                  系统设置
                </DropdownMenuItem>
              )}
              <DropdownMenuSeparator />
              <DropdownMenuItem className="gap-2 rounded-lg px-2.5 py-2 text-destructive focus:bg-destructive/10 focus:text-destructive" onSelect={handleLogout}>
                <LogOut className="h-4 w-4" aria-hidden />
                退出系统
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        </SidebarMenuItem>
      </SidebarMenu>
      {isNewShell && <ProfileDialog open={isProfileOpen} onOpenChange={setIsProfileOpen} />}
    </SidebarGroup>
  );
}
