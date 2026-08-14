"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import { Bell, MessageCircle, MoonIcon, Home, AlignJustify, Sparkles } from "lucide-react";

import { useSidebar } from "@/components/ui/sidebar";
import { FloatingDock } from "@/components/ui/floating-dock";
import { useThemeUtils } from "@/hooks/use-theme-utils";
import { getSiteNotificationCount } from "@/lib/dorm-notifications";
import { fetchUserPreferences, updateUserPreferences } from "@/lib/api";
import { clampDockPosition, parseDockPosition, type DockPosition } from "@/lib/preferences";

const iconClass = "h-4 w-4";
const DOCK_LEFT_MOBILE = "1rem";
const DOCK_HEIGHT = 48;
const DOCK_BOTTOM = 32;
// 位置持久化防抖窗口（毫秒）：拖动过程中仅本地视觉实时更新，
// 停止拖动一段时间后才写入后端偏好，避免 pointermove 触发大量请求。
const DOCK_PERSIST_DEBOUNCE_MS = 300;

export function ManagementBar() {
  const themeUtils = useThemeUtils();
  const { toggleSidebar, state, isMobile } = useSidebar();
  const [mounted, setMounted] = useState(false);
  const [unreadCount, setUnreadCount] = useState(0);
  const [dockPosition, setDockPosition] = useState<DockPosition | null>(null);
  const [preferencesLoaded, setPreferencesLoaded] = useState(false);
  const persistTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  const defaultDockPosition = useCallback((): DockPosition => ({
    left: isMobile ? 16 : state === "collapsed" ? 80 : 208,
    top: Math.max(8, window.innerHeight - DOCK_BOTTOM - DOCK_HEIGHT),
  }), [isMobile, state]);

  const handleDockPositionChange = useCallback((position: DockPosition) => {
    const element = document.querySelector<HTMLElement>("[data-floating-dock]");
    const dockSize = element ? { width: element.offsetWidth, height: element.offsetHeight } : { width: 240, height: DOCK_HEIGHT };
    const next = clampDockPosition(position, { width: window.innerWidth, height: window.innerHeight }, dockSize);
    // 视觉状态实时更新
    setDockPosition(next);
    // 持久化做 trailing debounce，避免拖动过程中大量请求
    if (persistTimerRef.current) clearTimeout(persistTimerRef.current);
    persistTimerRef.current = setTimeout(() => {
      void updateUserPreferences({ dock_position: next });
    }, DOCK_PERSIST_DEBOUNCE_MS);
  }, []);

  // 组件卸载时清理未触发的持久化定时器
  useEffect(() => () => {
    if (persistTimerRef.current) clearTimeout(persistTimerRef.current);
  }, []);

  useEffect(() => {
    setMounted(true);
    void fetchUserPreferences().then((preferences) => {
      setDockPosition(parseDockPosition(preferences.dock_position));
      setPreferencesLoaded(true);
    }).catch(() => setPreferencesLoaded(true));
    
    // Fetch unread notification count periodically
    const fetchUnread = async () => {
      setUnreadCount(getSiteNotificationCount());
    };
    
    fetchUnread();
    
    // Refresh every 30 seconds
    const interval = setInterval(fetchUnread, 30000);
    
    // Listen for notification count refresh events
    const handleRefresh = () => fetchUnread();
    window.addEventListener("notification:refresh", handleRefresh);
    
    return () => {
      clearInterval(interval);
      window.removeEventListener("notification:refresh", handleRefresh);
    };
  }, []);

  useEffect(() => {
    if (!preferencesLoaded || dockPosition) return;
    setDockPosition(defaultDockPosition());
  }, [defaultDockPosition, dockPosition, preferencesLoaded]);

  useEffect(() => {
    if (!dockPosition) return;
    const handleResize = () => {
      const element = document.querySelector<HTMLElement>("[data-floating-dock]");
      const dockSize = element ? { width: element.offsetWidth, height: element.offsetHeight } : { width: 240, height: DOCK_HEIGHT };
      setDockPosition(clampDockPosition(dockPosition, { width: window.innerWidth, height: window.innerHeight }, dockSize));
    };
    window.addEventListener("resize", handleResize);
    return () => window.removeEventListener("resize", handleResize);
  }, [dockPosition]);

  const dockItems = [
    {
      title: "侧边栏",
      icon: <AlignJustify className={iconClass} />,
      onClick: () => {
        toggleSidebar();
      },
    },
    {
      title: "主页",
      icon: <Home className={iconClass} />,
      onClick: () => window.dispatchEvent(new CustomEvent("dock:go-home")),
    },
    {
      title: "AI 助手",
      icon: <Sparkles className={iconClass} />,
      onClick: () => window.dispatchEvent(new CustomEvent("dock:open-chat")),
    },
    {
      title: mounted ? themeUtils.getAction() : "主题切换",
      icon: mounted ? themeUtils.getIcon(iconClass) : <MoonIcon className={iconClass} />,
      onClick: themeUtils.toggle,
    },
    {
      title: "通知中心",
      icon: <Bell className={iconClass} />,
      badge: unreadCount,
      onClick: () => window.dispatchEvent(new CustomEvent("dock:request-notification")),
    },
    {
      title: "QQ群",
      icon: <MessageCircle className={iconClass} />,
      onClick: () => window.dispatchEvent(new CustomEvent("dock:request-support")),
    },
  ];

  return (
    <div
      className="pointer-events-none fixed bottom-8 z-50 flex flex-col items-start gap-2"
      style={
        isMobile
          ? { left: DOCK_LEFT_MOBILE }
          : dockPosition
            ? undefined
            : { left: state === "collapsed" ? "5rem" : "13rem" }
      }
    >
      <FloatingDock
        items={dockItems}
        desktopClassName="pointer-events-auto bg-background/70 backdrop-blur-md border border-border/60 shadow-lg shadow-black/10 dark:shadow-white/5"
        mobileClassName="pointer-events-auto"
        mobileButtonClassName="bg-background/80 backdrop-blur"
        desktopPosition={!isMobile ? dockPosition ?? undefined : undefined}
        onDesktopPositionChange={handleDockPositionChange}
      />
    </div>
  );
}
