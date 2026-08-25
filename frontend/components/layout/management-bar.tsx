"use client";

import { useEffect, useState } from "react";
import { AlignJustify, Bell, Home, MessageCircle, MoonIcon, Search } from "lucide-react";

import { useSidebar } from "@/components/ui/sidebar";
import { FloatingDock } from "@/components/ui/floating-dock";
import { useThemeUtils } from "@/hooks/use-theme-utils";
import { getSiteNotificationCount } from "@/lib/dorm-notifications";
import { updateUserPreferences } from "@/lib/api";

const iconClass = "h-4 w-4";

/**
 * 控制坞定位契约：相对右侧白色圆角主容器左下角 16px。
 * ManagementBar 必须渲染在 app-main-container 内部，禁止再挂到视口层。
 */
export const DOCK_POSITION_CLASS = "pointer-events-none absolute bottom-4 left-4 z-50";

export function ManagementBar({ variant = "default" }: { variant?: "default" | "new" }) {
  const themeUtils = useThemeUtils();
  const { toggleSidebar } = useSidebar();
  const [mounted, setMounted] = useState(false);
  const [unreadCount, setUnreadCount] = useState(0);
  const [mobileExpanded, setMobileExpanded] = useState(false);

  useEffect(() => {
    setMounted(true);
    const refreshUnread = () => setUnreadCount(getSiteNotificationCount());
    refreshUnread();
    const interval = window.setInterval(refreshUnread, 30000);
    window.addEventListener("notification:refresh", refreshUnread);
    return () => {
      window.clearInterval(interval);
      window.removeEventListener("notification:refresh", refreshUnread);
    };
  }, []);

  const handleThemeToggle = () => {
    const nextTheme = themeUtils.toggle();
    void updateUserPreferences({ user_theme: nextTheme }).catch(() => {
      // next-themes 已将选择保存到本地存储，接口失败不阻断当前外观切换。
    });
  };

  const dockItems = [
    { title: "侧边栏", icon: <AlignJustify className={iconClass} />, onClick: toggleSidebar },
    { title: "主页", icon: <Home className={iconClass} />, onClick: () => window.dispatchEvent(new CustomEvent("dock:go-home")) },
    // 全局搜索入口：经 dock:open-search 事件由页面装配点打开搜索面板（替代原 AI 助手入口）
    { title: "全局搜索", icon: <Search className={iconClass} />, onClick: () => window.dispatchEvent(new CustomEvent("dock:open-search")) },
    { title: mounted ? themeUtils.getAction() : "主题切换", icon: mounted ? themeUtils.getIcon(iconClass) : <MoonIcon className={iconClass} />, onClick: handleThemeToggle },
    { title: "通知中心", icon: <Bell className={iconClass} />, badge: unreadCount, onClick: () => window.dispatchEvent(new CustomEvent("dock:request-notification")) },
    { title: "QQ群", icon: <MessageCircle className={iconClass} />, onClick: () => window.dispatchEvent(new CustomEvent("dock:request-support")) },
  ];

  return (
    <div className={DOCK_POSITION_CLASS}>
      <FloatingDock
        items={dockItems}
        variant={variant}
        // 柔和小扩散阴影(--dock-shadow)：向下延伸约 5px，远离主容器底边(间距 16px)，
        // 避免矩形阴影被容器 overflow-hidden 在圆角处硬裁出方形暗块
        desktopClassName="pointer-events-auto border border-border/80 bg-background shadow-[var(--dock-shadow)]"
        mobileClassName="pointer-events-auto"
        mobileButtonClassName="border border-border/80 bg-background shadow-[var(--dock-shadow)]"
        open={mobileExpanded}
        onOpenChange={setMobileExpanded}
      />
    </div>
  );
}
