"use client";

import { useState, useCallback } from "react";
import { ArrowLeft } from "lucide-react";

import { PageTransition } from "@/components/motion/page-transition";
import { Card, CardContent } from "@/components/ui/card";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { useAuth } from "@/lib/auth";
import { normalizeRole } from "@/lib/permissions";
import { usePermissions } from "@/hooks/use-permissions";
import type { Invoice } from "@/lib/api-invoice";

import InvoicesTab from "./tabs/InvoicesTab";
import PendingApprovalTab from "./tabs/PendingApprovalTab";
import StatsTab from "./tabs/StatsTab";
import { InvoiceDialog } from "./dialogs/InvoiceDialog";
import { InvoiceDetailDialog } from "./dialogs/InvoiceDetailDialog";
import { InvoiceUploadWorkbench } from "./upload/InvoiceUploadWorkbench";

interface InvoiceManagementProps {
  onBack?: () => void;
}

export default function InvoiceManagement({ onBack }: InvoiceManagementProps) {
  const { user } = useAuth();
  const role = normalizeRole(user?.role ?? "viewer");
  const { can } = usePermissions();

  const [activeTab, setActiveTab] = useState("list");

  // Dialog 状态
  const [dialogMode, setDialogMode] = useState<"create" | "edit">("create");
  const [editingInvoice, setEditingInvoice] = useState<Invoice | null>(null);
  const [showFormDialog, setShowFormDialog] = useState(false);
  const [detailInvoice, setDetailInvoice] = useState<Invoice | null>(null);

  // 刷新计数器（用于触发子组件重载）
  const [refreshKey, setRefreshKey] = useState(0);

  const userInitial = user?.full_name?.[0] || user?.username?.[0] || "U";

  /** 操作成功后刷新 */
  const handleRefresh = useCallback(() => {
    setRefreshKey((k) => k + 1);
  }, []);

  /** 打开新建 Dialog */
  const handleCreate = useCallback(() => {
    setDialogMode("create");
    setEditingInvoice(null);
    setShowFormDialog(true);
  }, []);

  /** 打开编辑 Dialog */
  const handleEdit = useCallback((invoice: Invoice) => {
    setDialogMode("edit");
    setEditingInvoice(invoice);
    setShowFormDialog(true);
  }, []);

  /** 查看详情 */
  const handleViewDetail = useCallback((invoice: Invoice) => {
    setDetailInvoice(invoice);
  }, []);

  /** manager+ 可查看统计和待审批 */
  const canManage = ["admin", "super_admin", "manager"].includes(role);
  /** admin 可审批 */
  const isAdmin = ["admin", "super_admin"].includes(role);
  /** 可上传解析（创建发票草稿） */
  const canUpload = can("invoice", "create");

  return (
    <PageTransition className="mx-auto flex w-full flex-col gap-4 p-2 md:p-4">
      {/* 顶部标题栏 */}
      <div className="flex flex-col gap-2">
        <div className="flex flex-wrap items-center justify-between gap-4">
          <div className="flex items-center gap-3">
            {onBack && (
              <button
                onClick={onBack}
                className="flex h-8 w-8 items-center justify-center rounded-full hover:bg-muted"
                aria-label="返回"
              >
                <ArrowLeft className="h-5 w-5" />
              </button>
            )}
            <div>
              <h1 className="text-3xl font-bold tracking-tight">发票管理</h1>
              <p className="text-muted-foreground">发票录入、审批与报销管理</p>
            </div>
          </div>
          <Card className="border-0 shadow-none">
            <CardContent className="flex items-center gap-2 p-2">
              <div className="flex h-8 w-8 items-center justify-center rounded-full bg-primary text-primary-foreground text-xs font-medium">
                {userInitial}
              </div>
              <span className="text-sm text-muted-foreground">
                {user?.full_name || user?.username || "当前用户"}
              </span>
            </CardContent>
          </Card>
        </div>
      </div>

      {/* Tab 架构 */}
      <Tabs value={activeTab} onValueChange={setActiveTab} className="space-y-4">
        <TabsList className="flex w-full justify-start">
          <TabsTrigger value="list">发票列表</TabsTrigger>
          {canUpload && <TabsTrigger value="upload">上传解析</TabsTrigger>}
          {isAdmin && <TabsTrigger value="pending">待审批</TabsTrigger>}
          {canManage && <TabsTrigger value="stats">统计分析</TabsTrigger>}
        </TabsList>

        <TabsContent value="list" className="space-y-4">
          <InvoicesTab
            onViewDetail={handleViewDetail}
            onCreateNew={handleCreate}
            refreshKey={refreshKey}
          />
        </TabsContent>

        {canUpload && (
          <TabsContent value="upload" className="space-y-4">
            <InvoiceUploadWorkbench onDone={handleRefresh} />
          </TabsContent>
        )}

        {isAdmin && (
          <TabsContent value="pending" className="space-y-4">
            <PendingApprovalTab refreshKey={refreshKey} onRefresh={handleRefresh} />
          </TabsContent>
        )}

        {canManage && (
          <TabsContent value="stats" className="space-y-4">
            <StatsTab refreshKey={refreshKey} />
          </TabsContent>
        )}
      </Tabs>

      {/* 新建/编辑 Dialog */}
      <InvoiceDialog
        open={showFormDialog}
        onOpenChange={setShowFormDialog}
        mode={dialogMode}
        invoice={editingInvoice}
        onSuccess={handleRefresh}
      />

      {/* 详情 Dialog */}
      <InvoiceDetailDialog
        open={!!detailInvoice}
        onOpenChange={(v) => { if (!v) setDetailInvoice(null); }}
        invoice={detailInvoice}
        onSuccess={handleRefresh}
        onEdit={handleEdit}
      />
    </PageTransition>
  );
}
