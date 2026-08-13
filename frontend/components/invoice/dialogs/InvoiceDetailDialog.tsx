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
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Input } from "@/components/ui/input";
import { Archive, Download, Eye, FileX, Printer } from "lucide-react";
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
  /** 确认归档返回的关联预警 */
  const [confirmWarnings, setConfirmWarnings] = useState<string[]>([]);
  const [warningsOpen, setWarningsOpen] = useState(false);

  if (!invoice) return null;

  /** 是否有受控附件（attachment_file_id 为主，attachment_url 兼容遗留数据） */
  const hasAttachment = invoice.attachment_file_id != null || Boolean(invoice.attachment_url);

  /** 附件是否可访问：后端仅允许待确认（pending）状态读取附件，已确认/作废后锁定 */
  const attachmentAccessible = invoice.archive_status === "pending" && hasAttachment;

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

  /** 受控原件预览：Blob URL 新窗口打开，关闭/超时后清理 URL */
  const handlePreviewAttachment = async () => {
    setLoading(true);
    try {
      const blob = await invoiceApi.getAttachment(invoice.id);
      const url = URL.createObjectURL(blob);
      const previewWindow = window.open(url, "_blank");
      if (!previewWindow) {
        toast.error("浏览器阻止了预览窗口，请允许弹窗后重试");
        URL.revokeObjectURL(url);
        return;
      }
      previewWindow.onload = () => previewWindow.focus();
      const cleanup = () => URL.revokeObjectURL(url);
      previewWindow.addEventListener("beforeunload", cleanup, { once: true });
      // 兜底：onload/关闭事件未触发时也释放 Blob URL，避免内存泄漏
      setTimeout(cleanup, 60_000);
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "原件预览失败");
    } finally {
      setLoading(false);
    }
  };

  /** 附件下载：Blob + 临时 a 标签触发浏览器下载 */
  const handleDownloadAttachment = async () => {
    setLoading(true);
    try {
      const blob = await invoiceApi.downloadAttachment(invoice.id);
      const url = URL.createObjectURL(blob);
      const link = document.createElement("a");
      link.href = url;
      link.download = `发票附件_${invoice.invoice_no || invoice.id}.pdf`;
      document.body.appendChild(link);
      link.click();
      link.remove();
      URL.revokeObjectURL(url);
      toast.success("附件已开始下载");
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "附件下载失败");
    } finally {
      setLoading(false);
    }
  };

  /** 浏览器打印：打开 PDF 后调用 print，失败提示 */
  const handlePrintAttachment = async () => {
    setLoading(true);
    try {
      const blob = await invoiceApi.getAttachment(invoice.id);
      const url = URL.createObjectURL(blob);
      const printWindow = window.open(url, "_blank");
      if (!printWindow) {
        toast.error("浏览器阻止了打印窗口，请允许弹窗后重试");
        URL.revokeObjectURL(url);
        return;
      }
      printWindow.onload = () => {
        try {
          printWindow.focus();
          printWindow.print();
        } catch {
          toast.error("自动打印失败，请在新窗口中手动打印");
        } finally {
          URL.revokeObjectURL(url);
        }
      };
      // 兜底：onload 未触发时也释放 Blob URL
      setTimeout(() => URL.revokeObjectURL(url), 60_000);
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "打印失败");
    } finally {
      setLoading(false);
    }
  };

  /** 管理员确认归档：成功后有后端关联预警则弹窗展示 */
  const handleConfirmArchive = async () => {
    setLoading(true);
    try {
      const result = await invoiceApi.confirm(invoice.id);
      const warnings = result.warnings ?? [];
      if (warnings.length > 0) {
        setConfirmWarnings(warnings);
        setWarningsOpen(true);
        toast.success("发票已确认归档，存在关联预警");
      } else {
        toast.success("发票已确认归档");
      }
      onSuccess();
      onOpenChange(false);
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "确认归档失败");
    } finally {
      setLoading(false);
    }
  };

  /** 管理员作废归档：作废原因必填（window.prompt），取消或空原因不执行 */
  const handleVoidArchive = async () => {
    const reason = window.prompt("请输入作废原因：");
    if (!reason) return;
    setLoading(true);
    try {
      await invoiceApi.void(invoice.id, reason);
      toast.success("发票已作废");
      onSuccess();
      onOpenChange(false);
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "作废失败");
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

    // 附件操作：仅待确认状态且有附件时可用（后端锁定 confirmed/voided 附件）
    if (attachmentAccessible) {
      buttons.push(
        <Button key="preview" variant="outline" size="sm" disabled={loading} onClick={handlePreviewAttachment}>
          <Eye className="mr-1.5 h-4 w-4" />原件预览
        </Button>,
        <Button key="download" variant="outline" size="sm" disabled={loading} onClick={handleDownloadAttachment}>
          <Download className="mr-1.5 h-4 w-4" />下载附件
        </Button>,
        <Button key="print" variant="outline" size="sm" disabled={loading} onClick={handlePrintAttachment}>
          <Printer className="mr-1.5 h-4 w-4" />打印
        </Button>,
      );
    }

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

    // 归档操作：admin 可确认归档（已审批且待确认）与作废（待确认/已确认）
    if (["admin", "super_admin"].includes(role)) {
      if (invoice.status === "approved" && invoice.archive_status === "pending") {
        buttons.push(
          <Button key="confirm-archive" variant="default" size="sm" disabled={loading} onClick={handleConfirmArchive}>
            <Archive className="mr-1.5 h-4 w-4" />确认归档
          </Button>,
        );
      }
      if (invoice.archive_status === "pending" || invoice.archive_status === "confirmed") {
        buttons.push(
          <Button key="void-archive" variant="destructive" size="sm" disabled={loading} onClick={handleVoidArchive}>
            <FileX className="mr-1.5 h-4 w-4" />作废
          </Button>,
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

      {/* 确认归档关联预警 */}
      <AlertDialog open={warningsOpen} onOpenChange={setWarningsOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>发票已确认归档，存在关联预警</AlertDialogTitle>
            <AlertDialogDescription>
              发票 {invoice.invoice_no || `#${invoice.id}`} 已归档，但检测到以下关联预警，请留意：
            </AlertDialogDescription>
          </AlertDialogHeader>
          <ul className="space-y-2">
            {confirmWarnings.map((warning, index) => (
              <li key={index} className="rounded-md border border-amber-200 bg-amber-50 px-3 py-2 text-sm text-amber-800">
                {warning}
              </li>
            ))}
          </ul>
          <AlertDialogFooter>
            <AlertDialogAction onClick={() => setWarningsOpen(false)}>知道了</AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </Dialog>
  );
}
