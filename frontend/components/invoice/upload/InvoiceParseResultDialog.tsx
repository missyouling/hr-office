"use client";

import { useEffect, useState } from "react";
import { toast } from "sonner";
import { Loader2, Save } from "lucide-react";

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
import { Badge } from "@/components/ui/badge";
import { usePermissions } from "@/hooks/use-permissions";
import { invoiceApi, type Invoice, type ParsingTaskDetail } from "@/lib/api-invoice";
import {
  PARSE_FIELD_DEFS,
  canEditParsedFields,
  isFieldLowConfidence,
  isFieldMissing,
} from "./validation";

interface InvoiceParseResultDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  invoiceId: number;
  /** 保存成功后回调（用于刷新列表） */
  onSaved?: () => void;
}

/** 任务状态中文名 */
const TASK_STATUS_LABEL: Record<string, string> = {
  pending: "待解析",
  running: "解析中",
  succeeded: "解析成功",
  failed: "解析失败",
};

/** 任务状态 Badge 颜色 */
const TASK_STATUS_CLASS: Record<string, string> = {
  pending: "border-gray-200 bg-gray-50 text-gray-700",
  running: "border-blue-200 bg-blue-50 text-blue-700",
  succeeded: "border-emerald-200 bg-emerald-50 text-emerald-700",
  failed: "border-rose-200 bg-rose-50 text-rose-700",
};

/** 将发票字段映射为表单字符串值（与 InvoiceDialog 口径一致） */
function buildFormFromInvoice(invoice: Invoice): Record<string, string> {
  return {
    invoice_no: invoice.invoice_no || "",
    invoice_date: invoice.invoice_date?.slice(0, 10) || "",
    invoice_type: invoice.invoice_type || "",
    amount: invoice.amount ? String(invoice.amount) : "",
    tax_amount: invoice.tax_amount ? String(invoice.tax_amount) : "",
    total_amount: invoice.total_amount ? String(invoice.total_amount) : "",
    seller: invoice.seller || "",
    seller_tax_no: invoice.seller_tax_no || "",
    buyer: invoice.buyer || "",
    purpose: invoice.purpose || "",
    remark: invoice.remark || "",
  };
}

export function InvoiceParseResultDialog({
  open,
  onOpenChange,
  invoiceId,
  onSaved,
}: InvoiceParseResultDialogProps) {
  const { can } = usePermissions();
  const [invoice, setInvoice] = useState<Invoice | null>(null);
  const [task, setTask] = useState<ParsingTaskDetail | null>(null);
  const [form, setForm] = useState<Record<string, string>>({});
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);

  // 加载发票与解析任务详情
  useEffect(() => {
    if (!open || !invoiceId) return;
    let cancelled = false;
    setLoading(true);
    (async () => {
      try {
        const [inv, taskDetail] = await Promise.all([
          invoiceApi.get(invoiceId),
          // 解析任务详情为可选增强：接口失败不阻断字段编辑入口
          invoiceApi.getParsingTask(invoiceId).catch(() => null),
        ]);
        if (cancelled) return;
        setInvoice(inv);
        setTask(taskDetail);
        setForm(buildFormFromInvoice(inv));
      } catch (err) {
        if (!cancelled) {
          toast.error(err instanceof Error ? err.message : "加载发票详情失败");
          onOpenChange(false);
        }
      } finally {
        if (!cancelled) setLoading(false);
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [open, invoiceId, onOpenChange]);

  const editable = canEditParsedFields(invoice?.status, can("invoice", "edit"));

  /** 更新表单字段 */
  const updateField = (key: string, value: string) => {
    setForm((prev) => ({ ...prev, [key]: value }));
  };

  /** 保存识别字段（仅草稿 + 有编辑权限） */
  const handleSave = async () => {
    if (!invoice) return;
    const missingRequired = PARSE_FIELD_DEFS.find(
      (def) => def.required && isFieldMissing(form[def.key]),
    );
    if (missingRequired) {
      toast.error(`「${missingRequired.label}」不能为空`);
      return;
    }
    setSaving(true);
    try {
      await invoiceApi.update(invoice.id, {
        invoice_no: form.invoice_no.trim(),
        invoice_date: form.invoice_date,
        invoice_type: form.invoice_type || undefined,
        amount: parseFloat(form.amount) || 0,
        tax_amount: form.tax_amount ? parseFloat(form.tax_amount) : 0,
        total_amount: form.total_amount ? parseFloat(form.total_amount) : 0,
        seller: form.seller.trim(),
        seller_tax_no: form.seller_tax_no.trim() || undefined,
        buyer: form.buyer.trim() || undefined,
        purpose: form.purpose.trim() || undefined,
        remark: form.remark.trim() || undefined,
      });
      toast.success("识别字段已保存");
      onSaved?.();
      onOpenChange(false);
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "保存失败");
    } finally {
      setSaving(false);
    }
  };

  /** 字段高亮样式：低置信度优先于缺失 */
  const highlightClass = (key: string): { className: string; hint: string } => {
    const lowConf = isFieldLowConfidence(invoice?.field_confidence, key);
    const missing = isFieldMissing(form[key]);
    if (lowConf) return { className: "border-amber-400", hint: "低置信度" };
    if (missing) return { className: "border-amber-300", hint: "缺失" };
    return { className: "", hint: "" };
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-h-[90vh] max-w-lg overflow-y-auto">
        <DialogHeader>
          <DialogTitle>发票识别字段</DialogTitle>
          <DialogDescription>
            {invoice && (
              <span className="flex items-center gap-2">
                发票编号：{invoice.invoice_no || "-"}
                {task && (
                  <Badge variant="outline" className={TASK_STATUS_CLASS[task.status] ?? ""}>
                    {TASK_STATUS_LABEL[task.status] ?? task.status}
                  </Badge>
                )}
              </span>
            )}
          </DialogDescription>
        </DialogHeader>

        {loading && (
          <div className="flex justify-center py-10">
            <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
          </div>
        )}

        {!loading && invoice && (
          <div className="grid gap-4 py-4">
            {!editable && (
              <p className="rounded-md bg-muted px-3 py-2 text-sm text-muted-foreground">
                仅草稿状态且具备编辑权限时可修改识别字段，当前为只读展示。
              </p>
            )}

            {PARSE_FIELD_DEFS.map((def) => {
              const { className, hint } = highlightClass(def.key);
              const isTextarea = def.key === "remark";
              return (
                <div key={def.key} className="grid gap-2">
                  <Label htmlFor={`parse_${def.key}`}>
                    {def.label}
                    {def.required && " *"}
                    {hint && (
                      <Badge variant="outline" className="ml-2 border-amber-300 bg-amber-50 text-amber-700">
                        {hint}
                      </Badge>
                    )}
                  </Label>
                  {isTextarea ? (
                    <Textarea
                      id={`parse_${def.key}`}
                      rows={2}
                      value={form[def.key] ?? ""}
                      disabled={!editable}
                      onChange={(e) => updateField(def.key, e.target.value)}
                      className={className}
                    />
                  ) : (
                    <Input
                      id={`parse_${def.key}`}
                      type={def.key === "invoice_date" ? "date" : def.key.includes("amount") || def.key === "tax_amount" || def.key === "total_amount" ? "number" : "text"}
                      step={def.key.includes("amount") || def.key === "tax_amount" || def.key === "total_amount" ? "0.01" : undefined}
                      value={form[def.key] ?? ""}
                      disabled={!editable}
                      onChange={(e) => updateField(def.key, e.target.value)}
                      className={className}
                    />
                  )}
                </div>
              );
            })}
          </div>
        )}

        {!loading && !invoice && (
          <p className="py-8 text-center text-sm text-muted-foreground">未找到发票记录</p>
        )}

        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)} disabled={saving}>
            关闭
          </Button>
          {editable && (
            <Button onClick={handleSave} disabled={saving}>
              {saving ? <Loader2 className="mr-1 h-4 w-4 animate-spin" /> : <Save className="mr-1 h-4 w-4" />}
              保存字段
            </Button>
          )}
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
