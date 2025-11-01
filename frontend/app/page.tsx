"use client";

import { useState } from "react";
import { useAuth } from "@/lib/supabase/auth-context";
import { useRouter } from "next/navigation";
import { useEffect } from "react";

// Disable static generation for this page
export const dynamic = 'force-dynamic';

import { Sidebar } from "@/components/sidebar";
import { EmployeeManagement } from "@/components/employee-management";
import { InsuranceManagement } from "@/components/insurance-management";
import { AuditLogs } from "@/components/audit-logs";
import { SystemMonitoring } from "@/components/system-monitoring";
import { OrganizationManagement } from "@/components/organization-management";
import { Skeleton } from "@/components/ui/skeleton";

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
  const [currentView, setCurrentView] = useState("employee");

  // Show loading spinner while authentication is being checked
  if (isLoading) {
    return (
      <div className="flex min-h-screen bg-background">
        <div className="w-64 border-r bg-muted/40 p-4">
          <Skeleton className="h-8 w-full mb-4" />
          <div className="space-y-2">
            <Skeleton className="h-10 w-full" />
            <Skeleton className="h-10 w-full" />
            <Skeleton className="h-10 w-full" />
          </div>
        </div>
        <main className="flex-1 overflow-auto">
          <div className="p-6">
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
      case "employee":
        return <EmployeeManagement />;
      case "insurance":
        return <InsuranceManagement />;
      case "organization":
        return <OrganizationManagement />;
      case "audit":
        return <AuditLogs />;
      case "monitoring":
        return <SystemMonitoring />;
      default:
        return <InsuranceManagement />;
    }
  };

  return (
    <div className="h-screen bg-white overflow-hidden">
      {/* Fixed Sidebar */}
      <Sidebar currentView={currentView} onViewChange={setCurrentView} />

      {/* Scrollable Main Content Area */}
      <main className="fixed top-0 left-64 right-0 bottom-0 overflow-auto">
        <div className="p-6">
          {renderMainContent()}
        </div>
      </main>
    </div>
  );
}
