"use client";

import { useState } from "react";
import { toast } from "sonner";

import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Input } from "@/components/ui/input";
import { PermissionGate } from "@/components/permission-gate";
import { useAuth } from "@/lib/auth";
import { normalizeRole } from "@/lib/permissions";
import { invoiceApi, type Invoice } from "@/lib/api-invoice";
import {
  getStatusLabel,
  getStatusBadgeClass,
  getSourceTypeLabel,
  formatCurrency,
  formatDate,
} from "../utils";

interface InvoiceDetailDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  invoice: Invoice | null;
  onSuccess: () => void;
  onEdit?: (invoice: Invoice) => void;
}

/** 显示字段配置 */
interface FieldConfig {
  label: string;
  value: React.ReactNode;
}

export function InvoiceDetailDialog({ open, onOpenChange, invoice, onSuccess, onEdit }: InvoiceDetailDialogProps) {
  const { user } = useAuth();
  const role = normalizeRole(user?.role ?? "viewer");
  const [loading, setLoading] = useState(false);
  const [reimburseAmount, setReimburseAmount] = useState("");
  const [showReimburseInput, setShowReimburseInput] = useState(false);

  if (!invoice) return null;

  /** 执行操作 */
  const handleAction = async (action: () => Promise<unknown>, successMsg: string) => {
    setLoading(true);
    try {
      await action();
      toast.success(successMsg);
      onSuccess();
      onOpenChange(false);
    } catch (err) {
      const message = err instanceof Error ? err.message : "操作失败";
      toast.error(message);
    } finally {
      setLoading(false);
    }
  };

  /** 构建显示字段列表 */
  const fields: FieldConfig[] = [
    { label: "发票号", value: invoice.invoice_no || "-" },
    { label: "状态", value: <Badge variant="outline" className={getStatusBadgeClass(invoice.status)}>{getStatusLabel(invoice.status)}</Badge> },
    { label: "开票日期", value: formatDate(invoice.invoice_date) },
    { label: "发票类型", value: invoice.invoice_type || "-" },
    { label: "金额", value: formatCurrency(invoice.amount) },
    { label: "税额", value: formatCurrency(invoice.tax_amount) },
    { label: "含税总额", value: formatCurrency(invoice.total_amount) },
    { label: "销售方", value: invoice.seller || "-" },
    { label: "销售方税号", value: invoice.seller_tax_no || "-" },
    { label: "购方", value: invoice.buyer || "-" },
    { label: "用途", value: invoice.purpose || "-" },
    { label: "关联业务", value: invoice.source_type ? `${getSourceTypeLabel(invoice.source_type)}${invoice.source_id ? ` (ID: ${invoice.source_id})` : ""}` : "-" },
    { label: "实报销金额", value: formatCurrency(invoice.reimburse_amount) },
    { label: "审批意见", value: invoice.approval_remark || "-" },
    { label: "备注", value: invoice.remark || "-" },
    { label: "创建时间", value: formatDate(invoice.created_at) },
    { label: "更新时间", value: formatDate(invoice.updated_at) },
  ];

  /** 渲染操作按钮 */
  const renderActions = () => {
    const buttons: React.ReactNode[] = [];

    // 草稿状态：editor+ 可提交/编辑/删除
    if (invoice.status === "draft") {
      buttons.push(
        <PermissionGate key="submit" resource="invoice" action="submit">
          <Button variant="default" size="sm" disabled={loading}
            onClick={() => handleAction(() => invoiceApi.submit(invoice.id), "发票已提交审批")}>
            提交审批
          </Button>
        </PermissionGate>,
      );
      buttons.push(
        <PermissionGate key="edit" resource="invoice" action="edit">
          <Button variant="outline" size="sm" disabled={loading}
            onClick={() => { onOpenChange(false); onEdit?.(invoice); }}>
            编辑
          </Button>
        </PermissionGate>,
      );
      buttons.push(
        <PermissionGate key="delete" resource="invoice" action="delete">
          <Button variant="outline" size="sm" disabled={loading}
            onClick={() => handleAction(() => invoiceApi.remove(invoice.id), "发票已删除")}>
            删除
          </Button>
        </PermissionGate>,
      );
    }

    // 已提交状态：admin 可审批/驳回
    if (invoice.status === "submitted" && ["admin", "super_admin"].includes(role)) {
      buttons.push(
        <Button key="approve" variant="default" size="sm" disabled={loading}
          onClick={() => handleAction(() => invoiceApi.approve(invoice.id), "发票已审批通过")}>
          审批通过
        </Button>,
      );
      buttons.push(
        <Button key="reject" variant="destructive" size="sm" disabled={loading}
          onClick={() => {
            const remark = window.prompt("请输入驳回原因：");
            if (remark) handleAction(() => invoiceApi.reject(invoice.id, remark), "发票已驳回");
          }}>
          驳回
        </Button>,
      );
    }

    // 已审批状态：manager+ 可确认报销
    if (invoice.status === "approved" && ["admin", "super_admin", "manager"].includes(role)) {
      if (!showReimburseInput) {
        buttons.push(
          <Button key="reimburse" variant="default" size="sm" disabled={loading}
            onClick={() => setShowReimburseInput(true)}>
            确认报销
          </Button>,
        );
      } else {
        buttons.push(
          <div key="reimburse-row" className="flex items-center gap-2">
            <Input
              type="number"
              step="0.01"
              min="0"
              value={reimburseAmount}
              onChange={(e) => setReimburseAmount(e.target.value)}
              placeholder="报销金额"
              className="w-32"
            />
            <Button size="sm" disabled={loading || !reimburseAmount}
              onClick={() => handleAction(
                () => invoiceApi.reimburse(invoice.id, parseFloat(reimburseAmount)),
                "发票已确认报销"
              )}>
              确认
            </Button>
            <Button variant="ghost" size="sm" onClick={() => setShowReimburseInput(false)}>
              取消
            </Button>
          </div>,
        );
      }
    }

    return buttons;
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-h-[90vh] max-w-lg overflow-y-auto">
        <DialogHeader>
          <DialogTitle>
            <span className="inline-flex items-center gap-2">
              发票详情
              <Badge variant="outline" className={getStatusBadgeClass(invoice.status)}>
                {getStatusLabel(invoice.status)}
              </Badge>
            </span>
          </DialogTitle>
          <DialogDescription>发票编号：{invoice.invoice_no}</DialogDescription>
        </DialogHeader>

        {/* 字段列表 */}
        <div className="space-y-3 py-2">
          {fields.map((f) => (
            <div key={f.label} className="grid grid-cols-[100px_1fr] items-start gap-2">
              <span className="text-sm font-medium text-muted-foreground">{f.label}</span>
              <span className="text-sm">{f.value}</span>
            </div>
          ))}
        </div>

        {/* 操作按钮区 */}
        {renderActions().length > 0 && (
          <DialogFooter className="flex flex-wrap gap-2 sm:justify-start">
            {renderActions()}
          </DialogFooter>
        )}
      </DialogContent>
    </Dialog>
  );
}
