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
import { Textarea } from "@/components/ui/textarea";
import { Label } from "@/components/ui/label";
import { invoiceApi, type Invoice } from "@/lib/api-invoice";
import { formatCurrency, formatDate, getStatusLabel } from "../utils";

interface ApprovalDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  invoice: Invoice | null;
  onSuccess: () => void;
}

export function ApprovalDialog({ open, onOpenChange, invoice, onSuccess }: ApprovalDialogProps) {
  const [remark, setRemark] = useState("");
  const [loading, setLoading] = useState(false);

  if (!invoice) return null;

  const handleAction = async (action: () => Promise<Invoice>, successMsg: string) => {
    setLoading(true);
    try {
      await action();
      toast.success(successMsg);
      setRemark("");
      onSuccess();
      onOpenChange(false);
    } catch (err) {
      const message = err instanceof Error ? err.message : "操作失败";
      toast.error(message);
    } finally {
      setLoading(false);
    }
  };

  return (
    <Dialog open={open} onOpenChange={(v) => { if (!v) setRemark(""); onOpenChange(v); }}>
      <DialogContent className="max-w-md">
        <DialogHeader>
          <DialogTitle>审批发票</DialogTitle>
          <DialogDescription>
            请审核以下发票信息并做出决定
          </DialogDescription>
        </DialogHeader>

        {/* 发票关键信息 */}
        <div className="space-y-2 rounded-md border bg-muted/30 p-3 text-sm">
          <div className="flex justify-between">
            <span className="text-muted-foreground">发票号：</span>
            <span className="font-medium">{invoice.invoice_no}</span>
          </div>
          <div className="flex justify-between">
            <span className="text-muted-foreground">销售方：</span>
            <span>{invoice.seller}</span>
          </div>
          <div className="flex justify-between">
            <span className="text-muted-foreground">金额：</span>
            <span className="font-medium">{formatCurrency(invoice.amount)}</span>
          </div>
          <div className="flex justify-between">
            <span className="text-muted-foreground">含税总额：</span>
            <span>{formatCurrency(invoice.total_amount)}</span>
          </div>
          <div className="flex justify-between">
            <span className="text-muted-foreground">开票日期：</span>
            <span>{formatDate(invoice.invoice_date)}</span>
          </div>
          <div className="flex justify-between">
            <span className="text-muted-foreground">状态：</span>
            <span>{getStatusLabel(invoice.status)}</span>
          </div>
          {invoice.purpose && (
            <div className="flex justify-between">
              <span className="text-muted-foreground">用途：</span>
              <span>{invoice.purpose}</span>
            </div>
          )}
        </div>

        {/* 审批意见 */}
        <div className="grid gap-2">
          <Label htmlFor="approval-remark">审批意见</Label>
          <Textarea
            id="approval-remark"
            value={remark}
            onChange={(e) => setRemark(e.target.value)}
            placeholder="请输入审批意见（可选）"
            rows={3}
          />
        </div>

        <DialogFooter className="flex gap-2 sm:justify-between">
          <Button variant="outline" onClick={() => onOpenChange(false)} disabled={loading}>
            取消
          </Button>
          <div className="flex gap-2">
            <Button
              variant="destructive"
              disabled={loading || !remark.trim()}
              onClick={() => handleAction(
                () => invoiceApi.reject(invoice.id, remark.trim()),
                "发票已驳回"
              )}
            >
              驳回
            </Button>
            <Button
              disabled={loading}
              onClick={() => handleAction(
                () => invoiceApi.approve(invoice.id, remark.trim() || undefined),
                "发票已审批通过"
              )}
            >
              审批通过
            </Button>
          </div>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
