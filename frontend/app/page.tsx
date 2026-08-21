"use client";

import { useState, useEffect, type ReactNode } from "react";
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
import { SystemSettings } from "@/components/system-settings";
import { Skeleton } from "@/components/ui/skeleton";
import { SidebarProvider, SidebarInset } from "@/components/ui/sidebar";
import { NewAppSidebar } from "@/components/layout/new-app-sidebar";
import type { SettingsMode } from "@/components/layout/nav-user";
import { ManagementBar } from "@/components/layout/management-bar";
import { Button } from "@/components/ui/button";
import { ArrowLeft, ArrowRight, Bell, MessageSquare, Search } from "lucide-react";
import { fetchAnnouncements, type Announcement } from "@/lib/api";
import { ChatPanel } from "@/components/chat-panel";
import { NotificationCenter } from "@/components/notification-center";
import { GlobalSearch } from "@/components/global-search";
import { KnowledgeStats } from "@/components/knowledge-stats";
import { FeedbackPanel } from "@/components/feedback-panel";
import { DepartmentManagement } from "@/components/admin/department-management";
import KnowledgeBaseManagement from "@/components/knowledge/KnowledgeBaseManagement";
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
  const [chatOpen, setChatOpen] = useState(false);
  const [searchOpen, setSearchOpen] = useState(false);
  const [notificationOpen, setNotificationOpen] = useState(false);
  const [pendingMemoSiteId, setPendingMemoSiteId] = useState<number | null>(null);

  const handleOpenSettings = (mode: SettingsMode) => {
    // 仅在当前不是设置视图时记录来源，避免设置内互相跳转覆盖来源
    if (currentView !== "system" && currentView !== "personal-settings") {
      setSettingsReturnView(currentView);
    }
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
    window.addEventListener("dock:open-site-memo", handleSiteMemo);
    window.addEventListener("dock:request-support", handleSupport as EventListener);
    window.addEventListener("dock:go-home", handleGoHome as EventListener);
    return () => {
      window.removeEventListener("dock:request-notification", handleNotification as EventListener);
      window.removeEventListener("dock:open-notification", handleNotification as EventListener);
      window.removeEventListener("dock:open-chat", handleChat as EventListener);
      window.removeEventListener("dock:open-ai", handleChat as EventListener);
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
      <div className="app-shell flex h-[100dvh] min-h-0 overflow-hidden bg-muted">
        {/* 侧栏骨架：桌面 190px 白底边框；移动端隐藏（正式侧栏移动端为抽屉，不占文档流，避免页面滚动） */}
        <div className="hidden w-[11.875rem] shrink-0 border-r bg-background p-4 md:block">
          <Skeleton className="h-8 w-full mb-4" />
          <div className="space-y-2">
            <Skeleton className="h-10 w-full" />
            <Skeleton className="h-10 w-full" />
            <Skeleton className="h-10 w-full" />
          </div>
        </div>
        {/* 主区：外层不滚动，内部内容容器滚动，与正式 app-main-content 一致 */}
        <main className="min-h-0 flex-1 overflow-hidden">
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
    });

  const appShell = (
    <SidebarProvider className="app-shell h-[100dvh] overflow-hidden bg-muted">
      <NewAppSidebar currentView={currentView} onViewChange={setCurrentView} onOpenSettings={handleOpenSettings} />
      <SidebarInset className="relative h-full min-h-0 bg-muted">
        <ManagementBar variant="new" />
        <GlobalSearch
          onNavigate={(module) => { setCurrentView(module); }}
          open={searchOpen}
          onOpenChange={setSearchOpen}
          hideTrigger
        />
        <div className="relative flex min-h-0 flex-1 flex-col overflow-hidden">
          <NewShellContentTools onOpenSearch={() => setSearchOpen(true)} onOpenChat={() => setChatOpen(true)} />
          <div data-slot="app-main-content" className="min-h-0 flex-1 overflow-x-hidden overflow-y-auto bg-muted p-4 md:p-6">
            {renderMainContent()}
          </div>
          <ChatPanel open={chatOpen} onOpenChange={setChatOpen} variant="embedded" />
        </div>
        <NotificationCenter open={notificationOpen} onOpenChange={setNotificationOpen} />
      </SidebarInset>
    </SidebarProvider>
  );

  return <NewShell>{appShell}</NewShell>;
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
};

/**
 * P12.0.3 新壳占位包装器：仅以 data-shell 属性标记开关路径，
 * display:contents 保证无任何可见 UI 变化；后续 P12.1 以真实新壳实现替换本占位。
 */
export function NewShell({ children }: { children: ReactNode }) {
  const requestNotification = () => window.dispatchEvent(new CustomEvent("dock:open-notification"));

  return (
    <div data-shell="new" className="contents">
      <button
        type="button"
        aria-label="打开通知中心"
        onClick={requestNotification}
        className="fixed right-4 top-4 z-30 flex h-10 w-10 items-center justify-center rounded-full border border-border/70 bg-background/80 text-muted-foreground shadow-lg backdrop-blur transition-colors hover:bg-sidebar-accent hover:text-sidebar-accent-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring md:right-6 md:top-6"
      >
        <Bell className="h-4 w-4" aria-hidden />
      </button>
      {children}
    </div>
  );
}

export function NewShellContentTools({ onOpenSearch, onOpenChat }: { onOpenSearch: () => void; onOpenChat: () => void }) {
  return (
    <div className="flex shrink-0 items-center justify-between border-b border-border/70 bg-background px-4 py-3 md:px-6">
      <p className="text-sm text-muted-foreground">在当前工作区内搜索与使用 AI 助手</p>
      <div className="flex items-center gap-2">
        <Button variant="outline" size="sm" onClick={onOpenSearch} aria-label="打开全局搜索">
          <Search className="mr-2 h-4 w-4" />搜索 <kbd className="ml-2 hidden rounded border bg-muted px-1.5 text-[10px] sm:inline">⌘K</kbd>
        </Button>
        <Button size="sm" onClick={onOpenChat} aria-label="打开 AI 助手">
          <MessageSquare className="mr-2 h-4 w-4" />AI 助手
        </Button>
      </div>
    </div>
  );
}
