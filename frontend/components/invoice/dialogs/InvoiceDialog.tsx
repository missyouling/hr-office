"use client";

import { useState, useEffect } from "react";
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
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { invoiceApi, type Invoice } from "@/lib/api-invoice";
import { INVOICE_TYPE_OPTIONS, SOURCE_TYPE_OPTIONS } from "../utils";

interface InvoiceDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  mode: "create" | "edit";
  invoice?: Invoice | null;
  onSuccess: () => void;
}

/** 表单初始值 */
const initialForm = {
  invoice_no: "",
  invoice_date: new Date().toISOString().slice(0, 10),
  invoice_type: "",
  amount: "",
  tax_amount: "",
  total_amount: "",
  seller: "",
  seller_tax_no: "",
  buyer: "",
  purpose: "",
  remark: "",
  source_type: "independent",
  source_id: "",
};

export function InvoiceDialog({ open, onOpenChange, mode, invoice, onSuccess }: InvoiceDialogProps) {
  const [form, setForm] = useState(initialForm);
  const [loading, setLoading] = useState(false);

  // 编辑模式时填充表单
  useEffect(() => {
    if (mode === "edit" && invoice) {
      setForm({
        invoice_no: invoice.invoice_no || "",
        invoice_date: invoice.invoice_date?.slice(0, 10) || new Date().toISOString().slice(0, 10),
        invoice_type: invoice.invoice_type || "",
        amount: invoice.amount ? String(invoice.amount) : "",
        tax_amount: invoice.tax_amount ? String(invoice.tax_amount) : "",
        total_amount: invoice.total_amount ? String(invoice.total_amount) : "",
        seller: invoice.seller || "",
        seller_tax_no: invoice.seller_tax_no || "",
        buyer: invoice.buyer || "",
        purpose: invoice.purpose || "",
        remark: invoice.remark || "",
        source_type: invoice.source_type || "independent",
        source_id: invoice.source_id ? String(invoice.source_id) : "",
      });
    } else {
      setForm(initialForm);
    }
  }, [mode, invoice, open]);

  /** 更新表单字段 */
  const updateField = (field: string, value: string) => {
    setForm((prev) => ({ ...prev, [field]: value }));
  };

  /** 校验必填字段 */
  const validate = (): string | null => {
    if (!form.invoice_no.trim()) return "发票号不能为空";
    if (!form.invoice_date) return "开票日期不能为空";
    const amount = parseFloat(form.amount);
    if (Number.isNaN(amount) || amount <= 0) return "金额必须大于 0";
    if (!form.seller.trim()) return "销售方不能为空";
    return null;
  };

  /** 提交 */
  const handleSubmit = async () => {
    const error = validate();
    if (error) {
      toast.error(error);
      return;
    }
    setLoading(true);
    try {
      const payload = {
        invoice_no: form.invoice_no.trim(),
        invoice_date: form.invoice_date,
        invoice_type: form.invoice_type || undefined,
        amount: parseFloat(form.amount),
        tax_amount: form.tax_amount ? parseFloat(form.tax_amount) : 0,
        total_amount: form.total_amount ? parseFloat(form.total_amount) : 0,
        seller: form.seller.trim(),
        seller_tax_no: form.seller_tax_no.trim() || undefined,
        buyer: form.buyer.trim() || undefined,
        purpose: form.purpose.trim() || undefined,
        remark: form.remark.trim() || undefined,
        source_type: form.source_type || "independent",
        source_id: form.source_id ? parseInt(form.source_id, 10) : undefined,
      };

      if (mode === "edit" && invoice) {
        await invoiceApi.update(invoice.id, payload);
        toast.success("发票已更新");
      } else {
        await invoiceApi.create(payload);
        toast.success("发票草稿已创建");
      }
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
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-h-[90vh] max-w-lg overflow-y-auto">
        <DialogHeader>
          <DialogTitle>{mode === "create" ? "新建发票" : "编辑发票"}</DialogTitle>
          <DialogDescription>
            {mode === "create" ? "创建一张新的发票记录" : "修改发票信息（仅草稿状态可编辑）"}
          </DialogDescription>
        </DialogHeader>

        <div className="grid gap-4 py-4">
          {/* 发票号 */}
          <div className="grid gap-2">
            <Label htmlFor="invoice_no">发票号 *</Label>
            <Input
              id="invoice_no"
              value={form.invoice_no}
              onChange={(e) => updateField("invoice_no", e.target.value)}
              placeholder="输入发票号码"
            />
          </div>

          {/* 开票日期 + 发票类型 */}
          <div className="grid grid-cols-2 gap-4">
            <div className="grid gap-2">
              <Label htmlFor="invoice_date">开票日期 *</Label>
              <Input
                id="invoice_date"
                type="date"
                value={form.invoice_date}
                onChange={(e) => updateField("invoice_date", e.target.value)}
              />
            </div>
            <div className="grid gap-2">
              <Label htmlFor="invoice_type">发票类型</Label>
              <Select value={form.invoice_type} onValueChange={(v) => updateField("invoice_type", v)}>
                <SelectTrigger id="invoice_type">
                  <SelectValue placeholder="选择类型" />
                </SelectTrigger>
                <SelectContent>
                  {INVOICE_TYPE_OPTIONS.map((opt) => (
                    <SelectItem key={opt.value} value={opt.value}>
                      {opt.label}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
          </div>

          {/* 金额 / 税额 / 含税总额 */}
          <div className="grid grid-cols-3 gap-4">
            <div className="grid gap-2">
              <Label htmlFor="amount">金额 *</Label>
              <Input
                id="amount"
                type="number"
                step="0.01"
                min="0"
                value={form.amount}
                onChange={(e) => updateField("amount", e.target.value)}
                placeholder="0.00"
              />
            </div>
            <div className="grid gap-2">
              <Label htmlFor="tax_amount">税额</Label>
              <Input
                id="tax_amount"
                type="number"
                step="0.01"
                min="0"
                value={form.tax_amount}
                onChange={(e) => updateField("tax_amount", e.target.value)}
                placeholder="0.00"
              />
            </div>
            <div className="grid gap-2">
              <Label htmlFor="total_amount">含税总额</Label>
              <Input
                id="total_amount"
                type="number"
                step="0.01"
                min="0"
                value={form.total_amount}
                onChange={(e) => updateField("total_amount", e.target.value)}
                placeholder="0.00"
              />
            </div>
          </div>

          {/* 销售方 / 销售方税号 */}
          <div className="grid grid-cols-2 gap-4">
            <div className="grid gap-2">
              <Label htmlFor="seller">销售方 *</Label>
              <Input
                id="seller"
                value={form.seller}
                onChange={(e) => updateField("seller", e.target.value)}
                placeholder="销售方名称"
              />
            </div>
            <div className="grid gap-2">
              <Label htmlFor="seller_tax_no">销售方税号</Label>
              <Input
                id="seller_tax_no"
                value={form.seller_tax_no}
                onChange={(e) => updateField("seller_tax_no", e.target.value)}
                placeholder="税号"
              />
            </div>
          </div>

          {/* 购方 */}
          <div className="grid gap-2">
            <Label htmlFor="buyer">购方</Label>
            <Input
              id="buyer"
              value={form.buyer}
              onChange={(e) => updateField("buyer", e.target.value)}
              placeholder="购方名称（默认本公司）"
            />
          </div>

          {/* 用途 */}
          <div className="grid gap-2">
            <Label htmlFor="purpose">用途说明</Label>
            <Input
              id="purpose"
              value={form.purpose}
              onChange={(e) => updateField("purpose", e.target.value)}
              placeholder="发票用途"
            />
          </div>

          {/* 关联业务 */}
          <div className="grid grid-cols-2 gap-4">
            <div className="grid gap-2">
              <Label htmlFor="source_type">关联业务类型</Label>
              <Select value={form.source_type} onValueChange={(v) => updateField("source_type", v)}>
                <SelectTrigger id="source_type">
                  <SelectValue placeholder="选择关联类型" />
                </SelectTrigger>
                <SelectContent>
                  {SOURCE_TYPE_OPTIONS.map((opt) => (
                    <SelectItem key={opt.value} value={opt.value}>
                      {opt.label}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
            <div className="grid gap-2">
              <Label htmlFor="source_id">关联业务 ID</Label>
              <Input
                id="source_id"
                type="number"
                value={form.source_id}
                onChange={(e) => updateField("source_id", e.target.value)}
                placeholder="选填"
              />
            </div>
          </div>

          {/* 备注 */}
          <div className="grid gap-2">
            <Label htmlFor="remark">备注</Label>
            <Textarea
              id="remark"
              value={form.remark}
              onChange={(e) => updateField("remark", e.target.value)}
              placeholder="备注信息"
              rows={2}
            />
          </div>
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)} disabled={loading}>
            取消
          </Button>
          <Button onClick={handleSubmit} disabled={loading}>
            {loading ? "保存中..." : mode === "create" ? "创建草稿" : "保存修改"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
