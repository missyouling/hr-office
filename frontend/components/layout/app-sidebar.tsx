"use client";

import Link from "next/link";
import { useState } from "react";
import { Calculator, Home, Users, SquareArrowUpRight, BedDouble, FolderOpen, Settings, MessageSquareText, Building2, BookOpen } from "lucide-react";

import { useAuth } from "@/lib/supabase/auth-context";

import {
  Sidebar,
  SidebarContent,
  SidebarFooter,
  SidebarHeader,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
  SidebarSeparator,
  useSidebar,
} from "@/components/ui/sidebar";
import { NavMain, type NavMainItem } from "@/components/layout/nav-main";
import { NavDocuments } from "@/components/layout/nav-documents";
import { NavUser, type SettingsMode } from "@/components/layout/nav-user";
import { MyFeedbackDialog } from "@/components/feedback/my-feedback-dialog";
import { Button } from "@/components/ui/button";

const NAV_ITEMS: NavMainItem[] = [
  { id: "landing", label: "工作台", icon: Home },
  { id: "employee", label: "员工管理", icon: Users },
  { id: "insurance", label: "社保管理", icon: Calculator },
  { id: "dormitory", label: "宿舍管理", icon: BedDouble },
  { id: "daily-affairs", label: "日常事务", icon: FolderOpen },
  { id: "knowledge", label: "知识库", icon: BookOpen },
];

const SYSTEM_SETTINGS_ITEM: NavMainItem = { id: "system", label: "系统设置", icon: Settings };
const FEEDBACK_ITEM: NavMainItem = { id: "feedback", label: "反馈管理", icon: MessageSquareText };
const DEPARTMENTS_ITEM: NavMainItem = { id: "departments", label: "部门管理", icon: Building2 };

const HELP_LINKS: [] = [];

interface AppSidebarProps extends React.ComponentProps<typeof Sidebar> {
  currentView: string;
  onViewChange: (id: string) => void;
  onOpenSettings: (mode: SettingsMode) => void;
}

export function AppSidebar({ currentView, onViewChange, onOpenSettings, ...props }: AppSidebarProps) {
  const { user, hasPermission } = useAuth();
  const { isMobile, setOpenMobile } = useSidebar();
  const [isMyFeedbackOpen, setIsMyFeedbackOpen] = useState(false);

  const availableNavItems = NAV_ITEMS.filter((item) => {
    if (item.id === "organization" || item.id === "audit" || item.id === "monitoring") {
      return user?.email === "admin@example.com";
    }
    return true;
  });

  // ===== P7.1：管理类菜单按权限过滤 =====
  // 基础模块（员工/保险/宿舍/档案/公告）所有角色均有 view 权限，无需过滤；
  // 仅管理类菜单按权限显隐：
  // - 「系统设置」需要 settings.view
  // - 「部门管理」需要 department.view（仅 admin 有该权限，带角色兜底防止后端未下发权限时回归）
  // - 「反馈管理」无独立权限资源，沿用角色判断
  // - 备份/用户管理菜单当前未在侧边栏注册，若后续新增需分别用 backups.view / users.view 过滤
  const showSystemSettings = hasPermission("settings", "view");
  const showFeedback = hasPermission("users", "view");
  const showDepartments =
    hasPermission("department", "view") || user?.role === "admin" || user?.role === "super_admin";

  const allNavItems = [
    ...availableNavItems,
    ...(showDepartments ? [DEPARTMENTS_ITEM] : []),
    ...(showSystemSettings ? [SYSTEM_SETTINGS_ITEM] : []),
    ...(showFeedback ? [FEEDBACK_ITEM] : []),
  ];

  const handleViewChange = (id: string) => {
    onViewChange(id);
    if (isMobile) {
      setOpenMobile(false);
    }
  };

  return (
    <Sidebar
      collapsible="icon"
      className="border-r border-sidebar-border"
      style={{ "--sidebar-width": "11.875rem", "--sidebar-width-icon": "4rem" } as React.CSSProperties}
      {...props}
    >
      <div className="flex h-full flex-col">
        <SidebarHeader>
          <SidebarMenu>
            <SidebarMenuItem>
              <SidebarMenuButton asChild tooltip="人事行政管理系统" className="h-auto items-center gap-2 rounded-md px-2 py-1.5">
                <Link href="/">
                  <SquareArrowUpRight className="h-5 w-5" />
                  <span className="truncate text-base font-semibold">人事行政管理系统</span>
                  <span className="text-[11px] text-muted-foreground">v 1.0.1</span>
                </Link>
              </SidebarMenuButton>
            </SidebarMenuItem>
          </SidebarMenu>
        </SidebarHeader>
        <SidebarContent className="flex-1 space-y-2">
          <NavMain items={allNavItems} activeId={currentView} onSelect={handleViewChange} />
        </SidebarContent>
        <SidebarFooter className="mt-auto space-y-3 pb-6">
          <SidebarSeparator />
          <NavDocuments items={HELP_LINKS} />
          <Button
            type="button"
            variant="ghost"
            className="w-full justify-start"
            onClick={() => {
              setIsMyFeedbackOpen(true);
              if (isMobile) setOpenMobile(false);
            }}
          >
            <MessageSquareText className="h-4 w-4" />
            我的反馈
          </Button>
          <NavUser
            displayName={user?.full_name || user?.email || "未登录"}
            subLine={user?.email}
            onOpenSettings={(mode) => {
              onOpenSettings(mode);
              if (isMobile) setOpenMobile(false);
            }}
          />
          <MyFeedbackDialog open={isMyFeedbackOpen} onOpenChange={setIsMyFeedbackOpen} />
        </SidebarFooter>
      </div>
    </Sidebar>
  );
}
