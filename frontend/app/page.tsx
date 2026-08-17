"use client";

import { useState, useEffect } from "react";
import { useAuth } from "@/lib/supabase/auth-context";
import { useRouter } from "next/navigation";

// Disable static generation for this page
export const dynamic = 'force-dynamic';

import { EmployeeManagement } from "@/components/employee-management";
import { InsuranceManagement } from "@/components/insurance-management";
import { DormitoryManagement } from "@/components/dormitory-management";
import { AuditLogs } from "@/components/audit-logs";
import { SystemMonitoring } from "@/components/system-monitoring";
import { OrganizationManagement } from "@/components/organization-management";
import { DailyAffairsHub } from "@/components/daily-affairs-hub";
import { SystemSettings } from "@/components/system-settings";
import { Skeleton } from "@/components/ui/skeleton";
import { SidebarProvider, SidebarInset } from "@/components/ui/sidebar";
import { AppSidebar } from "@/components/layout/app-sidebar";
import type { SettingsMode } from "@/components/layout/nav-user";
import { ManagementBar } from "@/components/layout/management-bar";
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
import { PersonalSettings } from "@/components/personal-settings";
import { WorkbenchOverview } from "@/components/workbench-overview";

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
  const [currentView, setCurrentView] = useState("landing");
  // 记录进入设置前的来源视图，供“返回”按钮恢复
  const [settingsReturnView, setSettingsReturnView] = useState<string | null>(null);
  const [chatOpen, setChatOpen] = useState(false);
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
    window.addEventListener("dock:open-site-memo", handleSiteMemo);
    window.addEventListener("dock:request-support", handleSupport as EventListener);
    window.addEventListener("dock:go-home", handleGoHome as EventListener);
    return () => {
      window.removeEventListener("dock:request-notification", handleNotification as EventListener);
      window.removeEventListener("dock:open-notification", handleNotification as EventListener);
      window.removeEventListener("dock:open-chat", handleChat as EventListener);
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

  const renderMainContent = () => {
    switch (currentView) {
      case "landing":
        return <LandingContent userName={user.full_name || user.username} />;
      case "employee":
        return <EmployeeManagement />;
      case "insurance":
        return <InsuranceManagement />;
      case "dormitory":
        return <DormitoryManagement />;
      case "organization":
        return <OrganizationManagement />;
      case "audit":
        return <AuditLogs />;
      case "monitoring":
        return <SystemMonitoring />;
      case "daily-affairs":
        return <DailyAffairsHub />;
      case "system":
        return <SystemSettings onBack={handleBackFromSettings} />;
      case "personal-settings":
        return <PersonalSettings onBack={handleBackFromSettings} />;
      case "feedback":
        return <FeedbackPanel />;
      case "departments":
        return <DepartmentManagement />;
      case "knowledge":
        return <KnowledgeBaseManagement />;
      default:
        return <InsuranceManagement />;
    }
  };

  return (
    <SidebarProvider className="app-shell bg-muted">
      <AppSidebar currentView={currentView} onViewChange={setCurrentView} onOpenSettings={handleOpenSettings} />
      <SidebarInset className="relative h-full min-h-0 bg-muted">
        <ManagementBar />
        <GlobalSearch onNavigate={(module) => { setCurrentView(module); }} />
        <div data-slot="app-main-content" className="min-h-0 flex-1 overflow-x-hidden overflow-y-auto bg-muted p-4 md:p-6">
          {renderMainContent()}
        </div>
        <NotificationCenter open={notificationOpen} onOpenChange={setNotificationOpen} />
        <ChatPanel open={chatOpen} onOpenChange={setChatOpen} />
      </SidebarInset>
    </SidebarProvider>
  );
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
