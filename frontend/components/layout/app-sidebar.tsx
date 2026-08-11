"use client";

import Link from "next/link";
import { Calculator, Home, Users, SquareArrowUpRight, BedDouble, FolderOpen, Settings, MessageSquareText, Building2 } from "lucide-react";

import { useAuth } from "@/lib/supabase/auth-context";
import { toast } from "sonner";

import {
  Sidebar,
  SidebarContent,
  SidebarFooter,
  SidebarHeader,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
  SidebarSeparator,
} from "@/components/ui/sidebar";
import { NavMain, type NavMainItem } from "@/components/layout/nav-main";
import { NavDocuments } from "@/components/layout/nav-documents";
import { NavUser } from "@/components/layout/nav-user";

const NAV_ITEMS: NavMainItem[] = [
  { id: "landing", label: "工作台", icon: Home },
  { id: "employee", label: "员工管理", icon: Users },
  { id: "insurance", label: "社保管理", icon: Calculator },
  { id: "dormitory", label: "宿舍管理", icon: BedDouble },
  { id: "daily-affairs", label: "日常事务", icon: FolderOpen },
];

const SYSTEM_SETTINGS_ITEM: NavMainItem = { id: "system", label: "系统设置", icon: Settings };
const FEEDBACK_ITEM: NavMainItem = { id: "feedback", label: "反馈管理", icon: MessageSquareText };
const DEPARTMENTS_ITEM: NavMainItem = { id: "departments", label: "部门管理", icon: Building2 };

const HELP_LINKS: [] = [];

interface AppSidebarProps extends React.ComponentProps<typeof Sidebar> {
  currentView: string;
  onViewChange: (id: string) => void;
}

export function AppSidebar({ currentView, onViewChange, ...props }: AppSidebarProps) {
  const { user, logout } = useAuth();

  const availableNavItems = NAV_ITEMS.filter((item) => {
    if (item.id === "organization" || item.id === "audit" || item.id === "monitoring") {
      return user?.email === "admin@example.com";
    }
    return true;
  });

  // 系统设置仅 admin/super_admin 可见，合并到主菜单
  const showSystemSettings = user?.role === "admin" || user?.role === "super_admin";
  const showFeedback = user?.role === "admin" || user?.role === "super_admin";
  const showDepartments = user?.role === "admin" || user?.role === "super_admin";

  const allNavItems = [
    ...availableNavItems,
    ...(showDepartments ? [DEPARTMENTS_ITEM] : []),
    ...(showSystemSettings ? [SYSTEM_SETTINGS_ITEM] : []),
    ...(showFeedback ? [FEEDBACK_ITEM] : []),
  ];

  const handleLogout = async () => {
    try {
      logout();
      toast.success("已退出登录");
    } catch (error) {
      console.error("Logout error", error);
      toast.error("退出登录失败");
    }
  };

  return (
    <Sidebar collapsible="icon" variant="inset" className="relative" style={{ "--sidebar-width": "16rem", "--sidebar-width-icon": "3.5rem" } as React.CSSProperties} {...props}>
      <div className="flex h-full flex-col">
        <SidebarHeader>
          <SidebarMenu>
            <SidebarMenuItem>
              <SidebarMenuButton asChild className="h-auto items-center gap-2 rounded-xl px-2 py-1.5">
                <Link href="/">
                  <SquareArrowUpRight className="h-5 w-5" />
                  <span className="rolling-text text-base font-semibold">
                    <span>人事行政管理系统</span>
                    <span aria-hidden>人事行政管理系统</span>
                  </span>
                  <span className="text-[11px] text-muted-foreground">v 1.0.1</span>
                </Link>
              </SidebarMenuButton>
            </SidebarMenuItem>
          </SidebarMenu>
        </SidebarHeader>
        <SidebarContent className="flex-1 space-y-2">
          <NavMain items={allNavItems} activeId={currentView} onSelect={onViewChange} />
        </SidebarContent>
        <SidebarFooter className="mt-auto space-y-3 pb-6">
          <SidebarSeparator />
          <NavDocuments items={HELP_LINKS} />
          <NavUser
            displayName={user?.full_name || user?.email || "未登录"}
            subLine={user?.email}
          />
        </SidebarFooter>
      </div>
    </Sidebar>
  );
}
