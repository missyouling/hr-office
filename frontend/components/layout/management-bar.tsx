"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import { Bell, MessageCircle, MoonIcon, Home, AlignJustify, Sparkles } from "lucide-react";

import { useSidebar } from "@/components/ui/sidebar";
import { FloatingDock } from "@/components/ui/floating-dock";
import { useThemeUtils } from "@/hooks/use-theme-utils";
import { getSiteNotificationCount } from "@/lib/dorm-notifications";
import * as dockApi from "@/lib/api";
import { clampDockPosition, parseDockPosition, parseMobileExpanded, type DockPosition } from "@/lib/preferences";

const iconClass = "h-4 w-4";
const DOCK_LEFT_MOBILE = "1rem";
const DOCK_HEIGHT = 48;
const DOCK_BOTTOM = 32;
// 位置持久化防抖窗口（毫秒）：拖动过程中仅本地视觉实时更新，
// 停止拖动一段时间后才写入后端偏好，避免 pointermove 触发大量请求。
const DOCK_PERSIST_DEBOUNCE_MS = 300;
const DOCK_PREFERENCES_STORAGE_KEY = "dock_preferences_v1";

function openAiAssistant(): void {
  window.dispatchEvent(new CustomEvent("dock:open-ai"));
  window.dispatchEvent(new CustomEvent("dock:open-chat"));
}

type StoredDockPreferences = {
  desktop_position: DockPosition | null;
  mobile_expanded: boolean;
};

function parseStoredDockPreferences(raw: unknown): StoredDockPreferences | null {
  if (!raw || typeof raw !== "object") return null;
  const candidate = raw as { desktop_position?: unknown; mobile_expanded?: unknown };
  const desktopPosition = candidate.desktop_position === null ? null : parseDockPosition(candidate.desktop_position);
  const mobileExpanded = parseMobileExpanded(candidate.mobile_expanded);
  if (desktopPosition === null && candidate.desktop_position !== null) return null;
  if (mobileExpanded === null) return null;
  return { desktop_position: desktopPosition, mobile_expanded: mobileExpanded };
}

function readStoredDockPreferences(): StoredDockPreferences | null {
  try {
    const value = window.localStorage.getItem(DOCK_PREFERENCES_STORAGE_KEY);
    return value ? parseStoredDockPreferences(JSON.parse(value)) : null;
  } catch {
    return null;
  }
}

function writeStoredDockPreferences(preferences: StoredDockPreferences): void {
  try {
    window.localStorage.setItem(DOCK_PREFERENCES_STORAGE_KEY, JSON.stringify(preferences));
  } catch {
    // 本地存储不可用时仍继续使用 Dock，服务端保存不受影响。
  }
}

export function ManagementBar({ variant = "default" }: { variant?: "default" | "new" }) {
  const themeUtils = useThemeUtils();
  const { toggleSidebar, state, isMobile } = useSidebar();
  const [mounted, setMounted] = useState(false);
  const [unreadCount, setUnreadCount] = useState(0);
  const [dockPosition, setDockPosition] = useState<DockPosition | null>(null);
  const [mobileExpanded, setMobileExpanded] = useState(false);
  const [preferencesLoaded, setPreferencesLoaded] = useState(false);
  const persistTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const mobileExpandedRef = useRef(false);

  const hasDockPreferencesApi = () => Object.keys(dockApi).includes("getDockPreferences") && Object.keys(dockApi).includes("updateDockPreferences");

  const defaultDockPosition = useCallback((): DockPosition => ({
    left: isMobile ? 16 : state === "collapsed" ? 80 : 208,
    top: Math.max(8, window.innerHeight - DOCK_BOTTOM - DOCK_HEIGHT),
  }), [isMobile, state]);

  const clampPosition = useCallback((position: DockPosition) => {
    const element = document.querySelector<HTMLElement>("[data-floating-dock]");
    const dockSize = element ? { width: element.offsetWidth, height: element.offsetHeight } : { width: 240, height: DOCK_HEIGHT };
    return clampDockPosition(position, { width: window.innerWidth, height: window.innerHeight }, dockSize);
  }, []);

  const applyDockPreferences = useCallback((preferences: StoredDockPreferences) => {
    const position = preferences.desktop_position ? clampPosition(preferences.desktop_position) : null;
    mobileExpandedRef.current = preferences.mobile_expanded;
    setDockPosition(position);
    setMobileExpanded(preferences.mobile_expanded);
    return { desktop_position: position, mobile_expanded: preferences.mobile_expanded };
  }, [clampPosition]);

  const handleDockPositionChange = useCallback((position: DockPosition) => {
    const next = clampPosition(position);
    // 视觉状态实时更新
    setDockPosition(next);
    // 持久化做 trailing debounce，避免拖动过程中大量请求
    if (persistTimerRef.current) clearTimeout(persistTimerRef.current);
    persistTimerRef.current = setTimeout(() => {
      const preferences = { desktop_position: next, mobile_expanded: mobileExpandedRef.current };
      writeStoredDockPreferences(preferences);
      if (hasDockPreferencesApi()) void dockApi.updateDockPreferences(preferences);
    }, DOCK_PERSIST_DEBOUNCE_MS);
  }, [clampPosition]);

  // 组件卸载时清理未触发的持久化定时器
  useEffect(() => () => {
    if (persistTimerRef.current) clearTimeout(persistTimerRef.current);
  }, []);

  useEffect(() => {
    setMounted(true);
    const applyStoredOrDefault = () => {
      const stored = readStoredDockPreferences();
      if (stored) applyDockPreferences(stored);
      setPreferencesLoaded(true);
    };
    if (!hasDockPreferencesApi()) {
      applyStoredOrDefault();
    } else void dockApi.getDockPreferences().then((preferences) => {
      const parsed = parseStoredDockPreferences(preferences);
      if (!parsed) {
        applyStoredOrDefault();
        return;
      }
      writeStoredDockPreferences(applyDockPreferences(parsed));
      setPreferencesLoaded(true);
    }).catch(applyStoredOrDefault);
    
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
  }, [applyDockPreferences]);

  const handleMobileOpenChange = useCallback((open: boolean) => {
    mobileExpandedRef.current = open;
    setMobileExpanded(open);
    const preferences = { desktop_position: dockPosition, mobile_expanded: open };
    writeStoredDockPreferences(preferences);
    if (hasDockPreferencesApi()) void dockApi.updateDockPreferences(preferences);
  }, [dockPosition]);

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
      onClick: openAiAssistant,
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
        variant={variant}
        desktopClassName={`pointer-events-auto bg-background/70 backdrop-blur-md border border-border/60 shadow-lg shadow-black/10 dark:shadow-white/5 ${variant === "new" ? "rounded-2xl" : ""} ${preferencesLoaded ? "" : "opacity-0"}`}
        mobileClassName={`pointer-events-auto ${preferencesLoaded ? "" : "opacity-0"}`}
        mobileButtonClassName={`bg-background/80 backdrop-blur ${variant === "new" ? "ring-1 ring-border/70" : ""}`}
        desktopPosition={!isMobile ? dockPosition ?? undefined : undefined}
        onDesktopPositionChange={handleDockPositionChange}
        open={mobileExpanded}
        onOpenChange={handleMobileOpenChange}
      />
    </div>
  );
}
