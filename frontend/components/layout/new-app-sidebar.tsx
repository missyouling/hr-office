"use client";

import Link from "next/link";
import { useState } from "react";
import {
  BedDouble, BookOpen, BriefcaseBusiness, Building2, CarFront, ChevronLeft, ChevronRight, FileSignature, ClipboardCheck, ScrollText, FileText, Award, ArrowRightLeft, GraduationCap, ShieldCheck, HeartPulse, Zap,
  FolderOpen, HardHat, Home, Landmark, Shield, UserMinus, UserPlus, Users, MessageSquareText,
  Megaphone, Cpu, Mail, Database, Activity, UserRound,
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
import { RollingText } from "@/components/layout/rolling-text";
import type { SystemSettingsPanel } from "@/components/system-settings";

/** 系统侧栏标题：滚动文字动效与 sr-only 可访问名称共用此文本 */
export const APP_SIDEBAR_TITLE = "人事行政管理";

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
  // 知识库入口指向 AI 问答页（knowledge-qa）；管理视图 knowledge 经 QA 页内按钮或全局搜索可达
  { id: "knowledge-qa", label: "知识库", icon: BookOpen, resource: "knowledge_base" },
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
    // 反馈管理从顶级移入日常事务组末尾；权限判断（users.view）保持不变
    { id: "feedback", label: "反馈管理", icon: MessageSquareText, resource: "users" },
  ] },
];

/** 系统设置模式侧栏菜单（已批准映射）：panel: 前缀表示打开观察面板而非切换 tab */
const SYSTEM_SETTINGS_ITEMS: NavigationItem[] = [
  { id: "announcements", label: "系统设置概览", icon: Megaphone },
  { id: "roles", label: "用户与权限", icon: Users },
  { id: "organization", label: "组织与部门", icon: Building2 },
  { id: "ai", label: "模型配置", icon: Cpu },
  { id: "notification", label: "通知与邮件", icon: Mail },
  { id: "storage", label: "存储与备份", icon: Database },
];

/** 审计/监控面板入口：沿用 users.view 权限过滤 */
const SYSTEM_OBSERVABILITY_NAV: NavigationItem[] = [
  { id: "panel:audit", label: "审计日志", icon: ScrollText },
  { id: "panel:monitoring", label: "系统监控", icon: Activity },
];

interface NewAppSidebarProps extends React.ComponentProps<typeof Sidebar> {
  currentView: string;
  onViewChange: (id: string) => void;
  onOpenSettings: (mode: SettingsMode) => void;
  /** 受控：系统设置子页 tab（与 SystemSettings.activeSubTab 同源） */
  settingsTab?: string;
  onSettingsTabChange?: (tab: string) => void;
  /** 受控：当前打开的系统观察面板；null 表示未打开 */
  settingsPanel?: SystemSettingsPanel | null;
  onOpenSettingsPanel?: (panel: SystemSettingsPanel | null) => void;
  /** 替换模式下"返回主界面"回调（复用页面级返回来源语义） */
  onReturnHome?: () => void;
}

export function NewAppSidebar({ currentView, onViewChange, onOpenSettings, settingsTab = "announcements", onSettingsTabChange, settingsPanel = null, onOpenSettingsPanel, onReturnHome, ...props }: NewAppSidebarProps) {
  const { user, hasPermission } = useAuth();
  // state：侧栏折叠态（"expanded" | "collapsed"），折叠时主菜单切换为纯图标平铺
  const { isMobile, setOpenMobile, state } = useSidebar();
  const [expandedGroup, setExpandedGroup] = useState<string | null>(null);
  // 替换模式判定：系统设置 / 个人设置 / 业务分组子页（顶级视图保持主菜单）
  const isSystemMode = currentView === "system";
  const isPersonalMode = currentView === "personal-settings";
  const activeGroup = NAVIGATION_GROUPS.find((group) => group.items.some((item) => item.id === currentView)) ?? null;
  const isGroupMode = !isSystemMode && !isPersonalMode && activeGroup !== null;
  const isReplacedNav = isSystemMode || isPersonalMode || isGroupMode;

  const handleViewChange = (id: string) => {
    onViewChange(id);
    if (isMobile) setOpenMobile(false);
  };

  // 系统设置菜单分发：panel: 前缀打开观察面板，organization 跳业务页，其余切换设置子页
  const handleSettingsSelect = (id: string) => {
    if (id.startsWith("panel:")) {
      onOpenSettingsPanel?.(id.slice("panel:".length) as SystemSettingsPanel);
    } else if (id === "organization") {
      handleViewChange("organization");
    } else {
      onSettingsTabChange?.(id);
    }
    if (isMobile) setOpenMobile(false);
  };

  return (
    // border-transparent 覆盖 sidebar 容器变体的 border-r（颜色继承全局 border-border），
    // 消除侧栏与主容器之间的真实竖线；底层与壳层同色，主容器圆角浮于其上
    <Sidebar {...props} collapsible="icon" mobileWidth="85vw" mobileMaxWidth="24rem" className="bg-sidebar border-transparent" style={{ "--sidebar-width": "15rem", "--sidebar-width-icon": "4rem" } as React.CSSProperties}>
      {isReplacedNav ? <ReturnHomeHeader onReturnHome={onReturnHome} /> : <SidebarTitleHeader />}
      <SidebarContent className="flex-1 overflow-x-hidden overflow-y-auto px-2">
        {isSystemMode && (
          <SystemSettingsNav
            activeId={settingsPanel ? `panel:${settingsPanel}` : settingsTab}
            showObservability={hasPermission("users", "view")}
            onSelect={handleSettingsSelect}
          />
        )}
        {isPersonalMode && (
          <SettingsNavigation items={[{ id: "personal-settings", label: "个人设置", icon: UserRound }]} activeId="personal-settings" onSelect={() => {}} />
        )}
        {isGroupMode && activeGroup && (
          <GroupSubPageNav group={activeGroup} currentView={currentView} hasPermission={hasPermission} role={user?.role} username={user?.username} onSelect={handleViewChange} />
        )}
        {!isReplacedNav && (
          <MainMenuContent currentView={currentView} hasPermission={hasPermission} role={user?.role} username={user?.username} expandedGroup={expandedGroup} onToggleGroup={setExpandedGroup} onSelect={handleViewChange} collapsed={state === "collapsed"} />
        )}
      </SidebarContent>
      <SidebarUserFooter displayName={user?.full_name || user?.email || "未登录"} subLine={user?.email} onOpenSettings={onOpenSettings} />
    </Sidebar>
  );
}

/** 主菜单头部：图标 + 滚动文字标题；
 * py-3 + 按钮 py-2 仅用上下间距把标题区加高到约 h-16 观感（60px），
 * 与内容区拉开呼吸感；禁止分隔线与底色块（契约） */
function SidebarTitleHeader() {
  return (
    <SidebarHeader className="shrink-0 py-3">
      <SidebarMenu><SidebarMenuItem><SidebarMenuButton asChild tooltip={APP_SIDEBAR_TITLE} className="h-auto items-center gap-2 rounded-lg px-2 py-2">
        <Link href="/"><BriefcaseBusiness className="h-5 w-5 shrink-0" /><RollingText text={APP_SIDEBAR_TITLE} className="truncate text-base font-semibold" /></Link>
      </SidebarMenuButton></SidebarMenuItem></SidebarMenu>
    </SidebarHeader>
  );
}

/** 替换模式共用头部：返回主界面（复用页面级返回来源语义）；未提供回调时不渲染，避免无效入口；
 * 与 SidebarTitleHeader 同用 py-3 加高节奏，保持两种头部观感协调 */
function ReturnHomeHeader({ onReturnHome }: { onReturnHome?: () => void }) {
  if (!onReturnHome) return null;
  return (
    <SidebarHeader className="shrink-0 py-3">
      <SidebarMenu><SidebarMenuItem>
        <SidebarMenuButton tooltip="返回主界面" className={cn(MENU_BUTTON_MOTION_CLASS, "w-full justify-start text-muted-foreground")} onClick={onReturnHome}>
          <ChevronLeft className="h-4 w-4" /><span>返回主界面</span>
        </SidebarMenuButton>
      </SidebarMenuItem></SidebarMenu>
    </SidebarHeader>
  );
}

/** 系统设置模式内容：已批准映射菜单 + 观察面板区（users.view 门控） */
function SystemSettingsNav({ activeId, showObservability, onSelect }: { activeId: string; showObservability: boolean; onSelect: (id: string) => void }) {
  return (
    <>
      <SettingsNavigation items={SYSTEM_SETTINGS_ITEMS} activeId={activeId} onSelect={onSelect} />
      {showObservability && (
        <section className="mt-3" aria-label="系统观察">
          <p className="px-2 py-1 text-[12.5px] font-medium text-foreground">系统观察</p>
          <SettingsNavigation items={SYSTEM_OBSERVABILITY_NAV} activeId={activeId} onSelect={onSelect} />
        </section>
      )}
    </>
  );
}

/** 业务分组子页内容：分组名标题 + 该组全部权限过滤后的扁平项 */
function GroupSubPageNav({ group, currentView, hasPermission, role, username, onSelect }: { group: NavigationGroup; currentView: string; hasPermission: (resource: string, action: string) => boolean; role?: string; username?: string; onSelect: (id: string) => void }) {
  return (
    <section className="mt-1" aria-label={group.label}>
      <p className="px-2 py-1 text-[12.5px] font-medium text-foreground">{group.label}</p>
      <NavigationItems items={filterVisibleItems(group.items, hasPermission, role, username)} currentView={currentView} onSelect={onSelect} />
    </section>
  );
}

/** 主菜单内容：顶级项 + 可折叠分组（仅非替换模式渲染）；折叠态跳过分组头，各组成员以图标平铺 */
function MainMenuContent({ currentView, hasPermission, role, username, expandedGroup, onToggleGroup, onSelect, collapsed }: { currentView: string; hasPermission: (resource: string, action: string) => boolean; role?: string; username?: string; expandedGroup: string | null; onToggleGroup: React.Dispatch<React.SetStateAction<string | null>>; onSelect: (id: string) => void; collapsed: boolean }) {
  const showDepartments = hasPermission("department", "view") || role === "admin" || role === "super_admin";
  // 顶级项与分组成员共用同一套可见性规则（含 organization/knowledge-qa 管理员兜底），避免两处过滤逻辑漂移
  const visibleTopLevelItems = TOP_LEVEL_ITEMS.filter((item) => {
    if (item.id === "departments") return showDepartments;
    return filterVisibleItems([item], hasPermission, role, username).length > 0;
  });
  const visibleGroups = NAVIGATION_GROUPS.map((group) => ({
    ...group,
    items: filterVisibleItems(group.items, hasPermission, role, username),
  })).filter((group) => group.items.length > 0);

  // 折叠态：不渲染分组头按钮，顶级项与各可见分组 items 扁平平铺（tooltip 由 SidebarMenuButton 提供）
  if (collapsed) {
    return (
      <>
        <NavigationItems items={visibleTopLevelItems} currentView={currentView} onSelect={onSelect} />
        {visibleGroups.map((group) => <NavigationItems key={group.label} items={group.items} currentView={currentView} onSelect={onSelect} />)}
      </>
    );
  }

  return (
    <>
      <NavigationItems items={visibleTopLevelItems} currentView={currentView} onSelect={onSelect} />
      {visibleGroups.map((group) => <NavigationGroupView key={group.label} group={group} currentView={currentView} expanded={expandedGroup === group.label} onToggle={onToggleGroup} onSelect={onSelect} />)}
    </>
  );
}

/** 底部账户区：所有模式共用，反馈对话框状态自持 */
function SidebarUserFooter({ displayName, subLine, onOpenSettings }: { displayName: string; subLine?: string; onOpenSettings: (mode: SettingsMode) => void }) {
  const { isMobile, setOpenMobile } = useSidebar();
  const [isMyFeedbackOpen, setIsMyFeedbackOpen] = useState(false);

  return (
    <SidebarFooter className="mt-auto shrink-0 pb-6">
      <SidebarSeparator />
      <NavUser
        displayName={displayName}
        subLine={subLine}
        variant="new"
        onOpenSettings={onOpenSettings}
        onOpenFeedback={() => {
          setIsMyFeedbackOpen(true);
          if (isMobile) setOpenMobile(false);
        }}
      />
      <MyFeedbackDialog open={isMyFeedbackOpen} onOpenChange={setIsMyFeedbackOpen} />
    </SidebarFooter>
  );
}

function filterVisibleItems(items: NavigationItem[], hasPermission: (resource: string, action: string) => boolean, role?: string, username?: string) {
  return items.filter((item) => {
    if (!item.resource) return true;
    // 管理员角色兜底：organization 与 knowledge-qa 同样存在后端未种子化权限导致全员不可见的问题
    if (item.id === "organization" && (role === "admin" || role === "super_admin" || username === "admin")) return true;
    if (item.id === "knowledge-qa" && (role === "admin" || role === "super_admin" || username === "admin")) return true;
    return hasPermission(item.resource, item.action ?? "view");
  });
}

function NavigationGroupView({ group, currentView, expanded, onToggle, onSelect }: { group: NavigationGroup; currentView: string; expanded: boolean; onToggle: React.Dispatch<React.SetStateAction<string | null>>; onSelect: (id: string) => void }) {
  return <section className="mt-3" aria-label={group.label}>
    <button type="button" aria-expanded={expanded} className="group flex h-9 w-full items-center gap-1.5 rounded-lg px-2 text-left text-[12.5px] font-medium text-foreground transition-colors hover:bg-sidebar-accent hover:text-sidebar-accent-foreground" onClick={() => onToggle((current) => current === group.label ? null : group.label)}><span className="flex-1">{group.label}</span><ChevronRight className={cn("h-3 w-3 transition-transform duration-200", expanded && "rotate-90")} aria-hidden /></button>
    {expanded && <NavigationItems items={group.items} currentView={currentView} onSelect={onSelect} nested />}
  </section>;
}

function NavigationItems({ items, currentView, onSelect, nested = false }: { items: NavigationItem[]; currentView: string; onSelect: (id: string) => void; nested?: boolean }) {
  return <SidebarMenu className={cn("space-y-0.5", nested && "mt-1")}>
    {items.map((item) => {
      const Icon = item.icon;
      const isActive = item.id === currentView;
      return <SidebarMenuItem key={item.label}><SidebarMenuButton isActive={isActive} tooltip={item.label} className={cn(MENU_BUTTON_MOTION_CLASS, nested && "pl-4", isActive && MENU_BUTTON_ACTIVE_CLASS)} onClick={() => onSelect(item.id)}><Icon className="h-4 w-4" /><span>{item.label}</span></SidebarMenuButton></SidebarMenuItem>;
    })}
  </SidebarMenu>;
}

/** 按 activeId 匹配激活态的扁平导航（复用既有悬停/选中样式契约） */
function SettingsNavigation({ items, activeId, onSelect }: { items: NavigationItem[]; activeId: string; onSelect: (id: string) => void }) {
  return (
    <SidebarMenu className="space-y-0.5">
      {items.map((item) => {
        const Icon = item.icon;
        const isActive = item.id === activeId;
        return <SidebarMenuItem key={item.label}><SidebarMenuButton isActive={isActive} tooltip={item.label} className={cn(MENU_BUTTON_MOTION_CLASS, isActive && MENU_BUTTON_ACTIVE_CLASS)} onClick={() => onSelect(item.id)}><Icon className="h-4 w-4" /><span>{item.label}</span></SidebarMenuButton></SidebarMenuItem>;
      })}
    </SidebarMenu>
  );
}

/**
 * 菜单态样式契约：
 * - 悬停 #CDCDCF（--sidebar-accent）圆角矩形，150ms 平滑过渡背景/文字/阴影，无位移无缩放；
 * - 选中浅灰 #F0F0F3（--muted）圆角矩形轻阴影（data-[active=true] 变体确保覆盖通用基类的 active 背景；
 *   #F9F9FB 底层上白色不可辨，改用 --muted 与底层拉开对比）。
 */
const MENU_BUTTON_MOTION_CLASS = "h-9 gap-3 rounded-lg px-2.5 text-[14px] font-normal transition-[background-color,color,box-shadow] duration-150 hover:bg-sidebar-accent hover:text-sidebar-accent-foreground";
const MENU_BUTTON_ACTIVE_CLASS = "data-[active=true]:bg-muted data-[active=true]:font-medium data-[active=true]:text-sidebar-foreground data-[active=true]:shadow-sm";
