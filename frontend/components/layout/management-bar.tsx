"use client";

import { useEffect, useState } from "react";
import { Bell, MessageCircle, MoonIcon, Home, AlignJustify } from "lucide-react";

import { useSidebar } from "@/components/ui/sidebar";
import { FloatingDock } from "@/components/ui/floating-dock";
import { useThemeUtils } from "@/hooks/use-theme-utils";
import { getUnreadNotificationCount } from "@/lib/api";

const iconClass = "h-4 w-4";

export function ManagementBar() {
  const themeUtils = useThemeUtils();
  const { toggleSidebar } = useSidebar();
  const [mounted, setMounted] = useState(false);
  const [unreadCount, setUnreadCount] = useState(0);

  useEffect(() => {
    setMounted(true);
    
    // Fetch unread notification count periodically
    const fetchUnread = async () => {
      try {
        const result = await getUnreadNotificationCount();
        setUnreadCount(result.unread || 0);
      } catch (error) {
        console.error("Failed to fetch unread count:", error);
      }
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
      style={{ left: "calc(var(--sidebar-width) + 1rem)" }}
    >
      <FloatingDock
        items={dockItems}
        desktopClassName="pointer-events-auto bg-background/70 backdrop-blur-md border border-border/60 shadow-lg shadow-black/10 dark:shadow-white/5"
        mobileClassName="pointer-events-auto"
        mobileButtonClassName="bg-background/80 backdrop-blur"
      />
    </div>
  );
}
