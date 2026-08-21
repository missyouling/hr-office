"use client";

import Link from "next/link";
import { useState } from "react";
import {
  BedDouble, BookOpen, BriefcaseBusiness, Building2, CarFront, ChevronRight, FileSignature, ClipboardCheck, ScrollText, FileText, Award, ArrowRightLeft, GraduationCap, ShieldCheck, HeartPulse, Zap,
  FolderOpen, HardHat, Home, Landmark, Settings2, Shield, UserMinus, UserPlus, Users, MessageSquareText,
} from "lucide-react";
import type { LucideIcon } from "lucide-react";

import { cn } from "@/lib/utils";
import { useAuth } from "@/lib/supabase/auth-context";
import {
  Sidebar, SidebarContent, SidebarFooter, SidebarHeader, SidebarMenu,
  SidebarMenuButton, SidebarMenuItem, SidebarSeparator, useSidebar,
} from "@/components/ui/sidebar";
import { NavUser, type SettingsMode } from "@/components/layout/nav-user";
import { MyFeedbackDialog } from "@/components/feedback/my-feedback-dialog";

type NavigationItem = {
  id: string;
  label: string;
  icon: LucideIcon;
  resource?: string;
  action?: string;
};

type NavigationGroup = {
  label: string;
  items: NavigationItem[];
};

const TOP_LEVEL_ITEMS: NavigationItem[] = [
  { id: "landing", label: "工作台", icon: Home },
  { id: "knowledge", label: "知识库", icon: BookOpen, resource: "knowledge_base" },
  { id: "personal-settings", label: "偏好设置", icon: Settings2 },
  { id: "feedback", label: "反馈管理", icon: MessageSquareText, resource: "users" },
  { id: "departments", label: "部门管理", icon: Shield, resource: "department" },
];

const NAVIGATION_GROUPS: NavigationGroup[] = [
  { label: "员工管理", items: [
    { id: "employee", label: "员工花名册", icon: Users, resource: "employee" },
    { id: "employee-provident", label: "公积金管理", icon: Landmark, resource: "employee" },
    { id: "onboarding", label: "入职管理", icon: UserPlus, resource: "employee", action: "create" },
    { id: "resignation", label: "离职管理", icon: UserMinus, resource: "employee", action: "edit" },
    { id: "regularization", label: "转正管理", icon: ClipboardCheck, resource: "employee", action: "edit" },
    { id: "labor-contracts", label: "劳动合同", icon: ScrollText, resource: "contract", action: "view" },
    { id: "rewards", label: "奖惩记录", icon: Award, resource: "reward", action: "view" },
    { id: "personnel-changes", label: "人事异动", icon: ArrowRightLeft, resource: "employee", action: "edit" },
    { id: "training", label: "培训管理", icon: GraduationCap, resource: "training", action: "view" },
  ] },
  { label: "行政管理", items: [
    { id: "organization", label: "组织管理", icon: Building2, resource: "department" },
    { id: "insurance", label: "社保管理", icon: Shield, resource: "insurance" },
    { id: "daily-affairs-archives", label: "档案管理", icon: FolderOpen, resource: "archives" },
    { id: "admin-contracts", label: "行政合同", icon: FileText, resource: "admin_contract", action: "view" },
    { id: "safety", label: "安全管理", icon: ShieldCheck, resource: "safety", action: "view" },
    { id: "occupational-health", label: "职业健康检查", icon: HeartPulse, resource: "occupational_health", action: "view" },
  ] },
  { label: "日常事务", items: [
    { id: "dormitory", label: "宿舍管理", icon: BedDouble, resource: "dormitory" },
    { id: "energy", label: "能耗管理", icon: Zap, resource: "dormitory" },
    { id: "daily-affairs-canteen", label: "食堂管理", icon: BriefcaseBusiness, resource: "canteen" },
    { id: "daily-affairs-invoice", label: "发票管理", icon: FileSignature, resource: "invoice" },
    { id: "daily-affairs-office-supplies", label: "办公劳保", icon: HardHat, resource: "office-supply" },
    { id: "fleet-vehicles", label: "车辆档案", icon: CarFront, resource: "fleet" },
  ] },
];

interface NewAppSidebarProps extends React.ComponentProps<typeof Sidebar> {
  currentView: string;
  onViewChange: (id: string) => void;
  onOpenSettings: (mode: SettingsMode) => void;
}

export function NewAppSidebar({ currentView, onViewChange, onOpenSettings, ...props }: NewAppSidebarProps) {
  const { user, hasPermission } = useAuth();
  const { isMobile, setOpenMobile } = useSidebar();
  const [expandedGroup, setExpandedGroup] = useState<string | null>(null);
  const [isMyFeedbackOpen, setIsMyFeedbackOpen] = useState(false);
  const showDepartments = hasPermission("department", "view") || user?.role === "admin" || user?.role === "super_admin";
  const visibleTopLevelItems = TOP_LEVEL_ITEMS.filter((item) => {
    if (item.id === "departments") {
      return showDepartments;
    }
    return !item.resource || hasPermission(item.resource, "view");
  });
  const visibleGroups = NAVIGATION_GROUPS.map((group) => ({
    ...group,
    items: filterVisibleItems(group.items, hasPermission, user?.role, user?.username),
  })).filter((group) => group.items.length > 0);

  const handleViewChange = (id: string) => {
    onViewChange(id);
    if (isMobile) setOpenMobile(false);
  };

  return (
    <Sidebar {...props} collapsible="icon" mobileWidth="85vw" mobileMaxWidth="24rem" className="border-r border-sidebar-border" style={{ "--sidebar-width": "11.875rem", "--sidebar-width-icon": "4rem" } as React.CSSProperties}>
      <SidebarHeader className="shrink-0">
        <SidebarMenu><SidebarMenuItem><SidebarMenuButton asChild tooltip="人事行政管理系统" className="h-auto items-center gap-2 rounded-md px-2 py-1.5">
          <Link href="/"><BriefcaseBusiness className="h-5 w-5" /><span className="rolling-text text-base font-semibold"><span>人事行政管理系统</span><span aria-hidden>人事行政管理系统</span></span><span className="text-[11px] text-muted-foreground">v 1.0.1</span></Link>
        </SidebarMenuButton></SidebarMenuItem></SidebarMenu>
      </SidebarHeader>
      <SidebarContent className="flex-1 overflow-x-hidden overflow-y-auto px-2">
        <NavigationItems items={visibleTopLevelItems} currentView={currentView} onSelect={handleViewChange} />
        {visibleGroups.map((group) => <NavigationGroupView key={group.label} group={group} currentView={currentView} expanded={expandedGroup === group.label} onToggle={setExpandedGroup} onSelect={handleViewChange} />)}
      </SidebarContent>
      <SidebarFooter className="mt-auto shrink-0 pb-6">
        <SidebarSeparator />
        <NavUser
          displayName={user?.full_name || user?.email || "未登录"}
          subLine={user?.email}
          variant="new"
          onOpenSettings={onOpenSettings}
          onOpenFeedback={() => {
            setIsMyFeedbackOpen(true);
            if (isMobile) setOpenMobile(false);
          }}
        />
        <MyFeedbackDialog open={isMyFeedbackOpen} onOpenChange={setIsMyFeedbackOpen} />
      </SidebarFooter>
    </Sidebar>
  );
}

function filterVisibleItems(items: NavigationItem[], hasPermission: (resource: string, action: string) => boolean, role?: string, username?: string) {
  return items.filter((item) => {
    if (!item.resource) return true;
    if (item.id === "organization" && (role === "admin" || role === "super_admin" || username === "admin")) return true;
    return hasPermission(item.resource, item.action ?? "view");
  });
}

function NavigationGroupView({ group, currentView, expanded, onToggle, onSelect }: { group: NavigationGroup; currentView: string; expanded: boolean; onToggle: React.Dispatch<React.SetStateAction<string | null>>; onSelect: (id: string) => void }) {
  return <section className="mt-3" aria-label={group.label}>
    <button type="button" aria-expanded={expanded} className="group flex w-full items-center gap-1.5 rounded-lg px-2 py-1 text-left text-[12.5px] font-medium text-muted-foreground transition-colors hover:bg-sidebar-accent hover:text-sidebar-accent-foreground" onClick={() => onToggle((current) => current === group.label ? null : group.label)}><span className="flex-1">{group.label}</span><ChevronRight className={cn("h-3 w-3 transition-transform duration-200", expanded && "rotate-90")} aria-hidden /></button>
    {expanded && <NavigationItems items={group.items} currentView={currentView} onSelect={onSelect} nested />}
  </section>;
}

function NavigationItems({ items, currentView, onSelect, nested = false }: { items: NavigationItem[]; currentView: string; onSelect: (id: string) => void; nested?: boolean }) {
  return <SidebarMenu className={cn("space-y-0.5", nested && "mt-1")}>
    {items.map((item) => {
      const Icon = item.icon;
      const isActive = item.id === currentView;
      return <SidebarMenuItem key={item.label}><SidebarMenuButton isActive={isActive} tooltip={item.label} className={cn("h-9 gap-3 rounded-lg px-2.5 transition-colors", nested && "pl-4", isActive ? "bg-sidebar-primary text-sidebar-primary-foreground" : "hover:bg-sidebar-accent hover:text-sidebar-accent-foreground")} onClick={() => onSelect(item.id)}><Icon className="h-4 w-4" /><span className="text-sm font-medium">{item.label}</span></SidebarMenuButton></SidebarMenuItem>;
    })}
  </SidebarMenu>;
}
