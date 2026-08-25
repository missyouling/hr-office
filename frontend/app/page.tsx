"use client";

import { useState, useEffect } from "react";
import { useAuth } from "@/lib/supabase/auth-context";
import { useRouter } from "next/navigation";
import { DEFAULT_VIEW, renderView, type ViewComponentMap } from "@/lib/view-mapping";

// Disable static generation for this page
export const dynamic = 'force-dynamic';

import { EmployeeManagement } from "@/components/employee-management";
import { OnboardingManagement } from "@/components/onboarding-management";
import { ResignationManagement } from "@/components/resignation-management";
import { RegularizationManagement } from "@/components/regularization-management";
import { LaborContractManagement } from "@/components/labor-contract-management";
import { RewardManagement } from "@/components/reward-management";
import { PersonnelChangeManagement } from "@/components/personnel-change-management";
import { TrainingManagement } from "@/components/training-management";
import { SafetyManagement } from "@/components/safety-management";
import { OccupationalHealthCheckManagement } from "@/components/occupational-health-check-management";
import { AdminContractManagement } from "@/components/admin-contract-management";
import { InsuranceManagement } from "@/components/insurance-management";
import { DormitoryManagement } from "@/components/dormitory-management";
import { EnergyManagement } from "@/components/energy-management";
import { AuditLogs } from "@/components/audit-logs";
import { SystemMonitoring } from "@/components/system-monitoring";
import { OrganizationManagement } from "@/components/organization-management";
import { DailyAffairsHub } from "@/components/daily-affairs-hub";
import { SystemSettings, type SystemSettingsPanel } from "@/components/system-settings";
import { Skeleton } from "@/components/ui/skeleton";
import { SidebarProvider, SidebarInset } from "@/components/ui/sidebar";
import { NewAppSidebar } from "@/components/layout/new-app-sidebar";
import type { SettingsMode } from "@/components/layout/nav-user";
import { ManagementBar } from "@/components/layout/management-bar";
import { APP_SHELL_CLASS, APP_SIDEBAR_WIDTH_VARS, MAIN_SCROLL_CLASS, AppMainContainer, NewShell } from "@/components/layout/app-shell";
import { Button } from "@/components/ui/button";
import { ArrowLeft, ArrowRight } from "lucide-react";
import { fetchAnnouncements, type Announcement } from "@/lib/api";
import { ChatPanel } from "@/components/chat-panel";
import { NotificationCenter } from "@/components/notification-center";
import { GlobalSearch } from "@/components/global-search";
import { KnowledgeStats } from "@/components/knowledge-stats";
import { FeedbackPanel } from "@/components/feedback-panel";
import { DepartmentManagement } from "@/components/admin/department-management";
import KnowledgeBaseManagement from "@/components/knowledge/KnowledgeBaseManagement";
import { KnowledgeQaPage } from "@/components/knowledge/knowledge-qa-page";
import { PersonalSettings } from "@/components/personal-settings";
import { WorkbenchOverview } from "@/components/workbench-overview";
import { FleetVehicleManagement } from "@/components/fleet-vehicle-management";

export default function HomePage() {
  const { user, isLoading: loading } = useAuth();
  const router = useRouter();
  const isAuthenticated = !!user;
  const isLoading = loading;

  // Redirect to auth page if not authenticated
  useEffect(() => {
    if (!loading && !user) {
      router.push('/auth');
    }
  }, [loading, user, router]);
  // 初始视图沿用默认视图常量（工作台 landing），后续切换由侧栏 / dock 事件驱动
  const [currentView, setCurrentView] = useState<string>(DEFAULT_VIEW);
  // 记录进入设置前的来源视图，供“返回”按钮恢复
  const [settingsReturnView, setSettingsReturnView] = useState<string | null>(null);
  // 系统设置受控状态：侧栏与内容区共享同一子页 tab 与观察面板
  const [settingsTab, setSettingsTab] = useState("announcements");
  const [settingsPanel, setSettingsPanel] = useState<SystemSettingsPanel | null>(null);
  const [chatOpen, setChatOpen] = useState(false);
  const [searchOpen, setSearchOpen] = useState(false);
  const [notificationOpen, setNotificationOpen] = useState(false);
  const [pendingMemoSiteId, setPendingMemoSiteId] = useState<number | null>(null);

  const handleOpenSettings = (mode: SettingsMode) => {
    // 仅在当前不是设置视图时记录来源，避免设置内互相跳转覆盖来源
    if (currentView !== "system" && currentView !== "personal-settings") {
      setSettingsReturnView(currentView);
    }
    // 重新进入系统设置时关闭遗留观察面板，避免直接落在审计/监控页
    setSettingsPanel(null);
    setCurrentView(mode === "personal" ? "personal-settings" : "system");
  };

  const handleBackFromSettings = () => {
    // 来源无效（未记录或本身是设置视图）时回退 landing
    const isValidReturn =
      settingsReturnView !== null &&
      settingsReturnView !== "system" &&
      settingsReturnView !== "personal-settings";
    setCurrentView(isValidReturn ? settingsReturnView : "landing");
    setSettingsReturnView(null);
  };

  useEffect(() => {
    const handler = () => {
      const trigger = document.querySelector('[data-sidebar="trigger"]') as HTMLButtonElement | null;
      trigger?.click();
    };
    window.addEventListener("dashboard:toggle-sidebar", handler as EventListener);
    return () => window.removeEventListener("dashboard:toggle-sidebar", handler as EventListener);
  }, []);

  useEffect(() => {
    const handleNotification = () => {
      setNotificationOpen(true);
    };
    const handleSiteMemo = (event: Event) => {
      const siteId = (event as CustomEvent<{ siteId?: number }>).detail?.siteId;
      if (typeof siteId !== "number") return;
      if (currentView === "dormitory") return;
      setPendingMemoSiteId(siteId);
      setCurrentView("dormitory");
    };
    const handleChat = () => setChatOpen(true);
    const handleOpenSearch = () => setSearchOpen(true);
    const handleGoHome = () => setCurrentView("landing");
    const handleSupport = () => {
      if (currentView !== "dormitory") {
        setCurrentView("dormitory");
        window.dispatchEvent(new CustomEvent("dock:open-support"));
      } else {
        window.dispatchEvent(new CustomEvent("dock:open-support"));
      }
    };
    window.addEventListener("dock:request-notification", handleNotification as EventListener);
    window.addEventListener("dock:open-notification", handleNotification as EventListener);
    window.addEventListener("dock:open-chat", handleChat as EventListener);
    window.addEventListener("dock:open-ai", handleChat as EventListener);
    // dock 全局搜索入口：打开 GlobalSearch 面板（AI 助手已从 dock 移除，浮动面板事件监听保留兼容）
    window.addEventListener("dock:open-search", handleOpenSearch as EventListener);
    window.addEventListener("dock:open-site-memo", handleSiteMemo);
    window.addEventListener("dock:request-support", handleSupport as EventListener);
    window.addEventListener("dock:go-home", handleGoHome as EventListener);
    return () => {
      window.removeEventListener("dock:request-notification", handleNotification as EventListener);
      window.removeEventListener("dock:open-notification", handleNotification as EventListener);
      window.removeEventListener("dock:open-chat", handleChat as EventListener);
      window.removeEventListener("dock:open-ai", handleChat as EventListener);
      window.removeEventListener("dock:open-search", handleOpenSearch as EventListener);
      window.removeEventListener("dock:open-site-memo", handleSiteMemo);
      window.removeEventListener("dock:request-support", handleSupport as EventListener);
      window.removeEventListener("dock:go-home", handleGoHome as EventListener);
    };
  }, [currentView]);

  useEffect(() => {
    if (currentView !== "dormitory" || pendingMemoSiteId === null) return;
    const siteId = pendingMemoSiteId;
    setPendingMemoSiteId(null);
    queueMicrotask(() => window.dispatchEvent(new CustomEvent("dock:open-site-memo", { detail: { siteId } })));
  }, [currentView, pendingMemoSiteId]);

  // Show loading spinner while authentication is being checked
  if (isLoading) {
    return (
      <div className={cnShell("flex")}>
        {/* 侧栏骨架：桌面直角贴边 #F9F9FB，宽度与正式侧栏(--sidebar-width 15rem)一致；移动端隐藏（正式侧栏移动端为抽屉，不占文档流） */}
        <div className="hidden w-[15rem] shrink-0 bg-sidebar p-4 md:block">
          <Skeleton className="h-8 w-full mb-4" />
          <div className="space-y-2">
            <Skeleton className="h-10 w-full" />
            <Skeleton className="h-10 w-full" />
            <Skeleton className="h-10 w-full" />
          </div>
        </div>
        {/* 主区：白色圆角容器，外层不滚动，内部内容滚动，与正式壳层一致 */}
        <main className="min-h-0 flex-1 overflow-hidden rounded-2xl bg-background shadow-sm">
          <div className="h-full overflow-y-auto p-4 md:p-6">
            <div className="space-y-4">
              <Skeleton className="h-8 w-64" />
              <Skeleton className="h-32 w-full" />
              <Skeleton className="h-64 w-full" />
            </div>
          </div>
        </main>
      </div>
    );
  }

  // Redirect to auth page if not authenticated (handled by router)
  if (!isAuthenticated) {
    // The auth context will handle the redirect
    return null;
  }

  // 视图装配已收敛至 lib/view-mapping.ts：currentView → 既有组件映射见文件底部 VIEW_COMPONENTS，
  // 非法视图回退 insurance 组件（与既有 default 分支一致），此处仅注入页面级上下文 props
  const renderMainContent = () =>
    renderView(currentView, VIEW_COMPONENTS, {
      userName: user.full_name || user.username,
      onBackFromSettings: handleBackFromSettings,
      systemSettingsTab: settingsTab,
      onSystemSettingsTabChange: setSettingsTab,
      systemSettingsPanelState: settingsPanel,
      onSystemSettingsPanelChange: setSettingsPanel,
      onViewChange: setCurrentView,
    });

  const appShell = (
    /* --sidebar-width 提升至 wrapper 层：Sidebar 组件的 style 只作用于 fixed 可见层，
       若仅在 NewAppSidebar 传 15rem，占位层(sidebar-gap)仍是全局 11.875rem(190px)，
       fixed 层(240px)会右溢 50px 压住主容器左缘，遮盖左缘上下圆角（用户反馈"看似直角"的根因） */
    <SidebarProvider className={APP_SHELL_CLASS} style={APP_SIDEBAR_WIDTH_VARS as React.CSSProperties}>
      <NewAppSidebar
        currentView={currentView}
        onViewChange={setCurrentView}
        onOpenSettings={handleOpenSettings}
        settingsTab={settingsTab}
        onSettingsTabChange={setSettingsTab}
        settingsPanel={settingsPanel}
        onOpenSettingsPanel={setSettingsPanel}
        onReturnHome={handleBackFromSettings}
      />
      <SidebarInset className="relative h-full min-h-0 bg-transparent">
        <AppMainContainer>
          <ManagementBar variant="new" />
          <div data-slot="app-main-content" className={MAIN_SCROLL_CLASS}>{renderMainContent()}</div>
          <ChatPanel open={chatOpen} onOpenChange={setChatOpen} variant="embedded" />
        </AppMainContainer>
        <GlobalSearch
          onNavigate={(module) => { setCurrentView(module); }}
          open={searchOpen}
          onOpenChange={setSearchOpen}
          hideTrigger
        />
        <NotificationCenter open={notificationOpen} onOpenChange={setNotificationOpen} />
      </SidebarInset>
    </SidebarProvider>
  );

  return <NewShell>{appShell}</NewShell>;
}

/** 组合壳层类名的小工具：骨架屏与正式壳共用同一套留白节奏（契约常量见 components/layout/app-shell） */
function cnShell(extra: string): string {
  return `${APP_SHELL_CLASS} ${extra}`;
}

function LandingContent({ userName }: { userName?: string | null }) {
  const [announcements, setAnnouncements] = useState<Announcement[]>([]);
  const [currentIndex, setCurrentIndex] = useState(0);

  useEffect(() => {
    fetchAnnouncements("published")
      .then(setAnnouncements)
      .catch(() => {});
  }, []);

  // 自动轮播
  useEffect(() => {
    if (announcements.length <= 1) return;
    const interval = setInterval(() => {
      setCurrentIndex((prev) => (prev + 1) % announcements.length);
    }, 5000);
    return () => clearInterval(interval);
  }, [announcements.length]);

  const handlePrev = () => {
    setCurrentIndex((prev) => (prev - 1 + announcements.length) % announcements.length);
  };

  const handleNext = () => {
    setCurrentIndex((prev) => (prev + 1) % announcements.length);
  };

  const currentAnnouncement = announcements[currentIndex];
  const hasAnnouncements = announcements.length > 0;

  return (
    <div className="mx-auto flex w-full max-w-[1180px] flex-col gap-8 pb-6">
      {/* 知识库统计卡片 */}
      <KnowledgeStats />

      {/* 当前用户的问候、日期与个性化工作台配置 */}
      <WorkbenchOverview name={userName} />

      {/* 欢迎卡 + 公告轮播 */}
      <div className="relative overflow-hidden rounded-[32px] bg-gradient-to-r from-blue-600 via-blue-700 to-purple-800 p-6 shadow-[0_12px_40px_-16px_rgba(59,130,246,0.5)]">
        {/* 装饰光斑 */}
        <div className="pointer-events-none absolute -right-20 -top-20 h-64 w-64 rounded-full bg-white/10 blur-3xl" />
        <div className="pointer-events-none absolute -bottom-16 -left-16 h-48 w-48 rounded-full bg-white/5 blur-2xl" />
        <div className="relative flex items-center justify-between">
          <Button
            variant="ghost"
            size="icon"
            className="h-10 w-10 rounded-full bg-white/20 text-white backdrop-blur hover:bg-white/30 disabled:opacity-50 transition-all duration-300"
            onClick={handlePrev}
            disabled={!hasAnnouncements}
          >
            <ArrowLeft className="h-5 w-5" />
          </Button>

          <div className="text-center text-white flex-1 px-4">
            {hasAnnouncements ? (
              <div key={currentAnnouncement.id} className="transition-all duration-300">
                <h2 className="text-2xl font-bold tracking-tight">{currentAnnouncement.title}</h2>
                {currentAnnouncement.content && (
                  <p className="mt-2 text-base text-white/90 line-clamp-2">{currentAnnouncement.content}</p>
                )}
                {announcements.length > 1 && (
                  <div className="flex justify-center gap-1.5 mt-4">
                    {announcements.map((_, idx) => (
                      <button
                        key={idx}
                        onClick={(e) => { e.stopPropagation(); setCurrentIndex(idx); }}
                        className={`h-1.5 rounded-full transition-all ${
                          idx === currentIndex ? "w-6 bg-white" : "w-1.5 bg-white/40"
                        }`}
                      />
                    ))}
                  </div>
                )}
              </div>
            ) : (
              <div>
                <h1 className="text-3xl font-bold tracking-tight">欢迎使用 人事行政管理系统</h1>
                <p className="mt-2 text-base text-white/90">高效办公省时省力，所见即所得快人一步</p>
              </div>
            )}
          </div>

          <Button
            variant="ghost"
            size="icon"
            className="h-10 w-10 rounded-full bg-white/20 text-white backdrop-blur hover:bg-white/30 disabled:opacity-50 transition-all duration-300"
            onClick={handleNext}
            disabled={!hasAnnouncements}
          >
            <ArrowRight className="h-5 w-5" />
          </Button>
        </div>
      </div>
    </div>
  );
}

/**
 * 视图装配映射表：合法视图 → React 组件（装配规则见 lib/view-mapping.ts）。
 * 声明在 LandingContent 之后以规避 TDZ（模块初始化时引用尚未定义）与循环依赖。
 */
const VIEW_COMPONENTS: ViewComponentMap = {
  landing: LandingContent,
  employee: EmployeeManagement,
  "employee-provident": EmployeeManagement,
  onboarding: OnboardingManagement,
  resignation: ResignationManagement,
  regularization: RegularizationManagement,
  "labor-contracts": LaborContractManagement,
  rewards: RewardManagement,
  "personnel-changes": PersonnelChangeManagement,
  training: TrainingManagement,
  "admin-contracts": AdminContractManagement,
  safety: SafetyManagement,
  "occupational-health": OccupationalHealthCheckManagement,
  insurance: InsuranceManagement,
  dormitory: DormitoryManagement,
  energy: EnergyManagement,
  organization: OrganizationManagement,
  audit: AuditLogs,
  monitoring: SystemMonitoring,
  "daily-affairs": DailyAffairsHub,
  "daily-affairs-archives": DailyAffairsHub,
  "daily-affairs-office-supplies": DailyAffairsHub,
  "daily-affairs-canteen": DailyAffairsHub,
  "daily-affairs-invoice": DailyAffairsHub,
  "fleet-vehicles": FleetVehicleManagement,
  system: SystemSettings,
  "personal-settings": PersonalSettings,
  feedback: FeedbackPanel,
  departments: DepartmentManagement,
  knowledge: KnowledgeBaseManagement,
  "knowledge-qa": KnowledgeQaPage,
};
