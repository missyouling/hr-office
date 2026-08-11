"use client";

import { useEffect, useState, useCallback } from "react";
import { toast } from "sonner";
import { Check, Loader2, Eye } from "lucide-react";

import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { invoiceApi, type Invoice } from "@/lib/api-invoice";
import { ApprovalDialog } from "../dialogs/ApprovalDialog";
import { InvoiceDetailDialog } from "../dialogs/InvoiceDetailDialog";
import {
  getStatusLabel,
  getStatusBadgeClass,
  getSourceTypeLabel,
  formatCurrency,
  formatDate,
} from "../utils";

interface PendingApprovalTabProps {
  refreshKey: number;
  onRefresh: () => void;
}

export default function PendingApprovalTab({ refreshKey, onRefresh }: PendingApprovalTabProps) {
  const [invoices, setInvoices] = useState<Invoice[]>([]);
  const [loading, setLoading] = useState(false);
  const [approvalTarget, setApprovalTarget] = useState<Invoice | null>(null);
  const [detailTarget, setDetailTarget] = useState<Invoice | null>(null);

  /** 加载已提交状态的发票 */
  const loadList = useCallback(async () => {
    setLoading(true);
    try {
      const data = await invoiceApi.list({ status: "submitted", page_size: 50 });
      setInvoices(data.items ?? []);
    } catch (err) {
      const message = err instanceof Error ? err.message : "加载待审批列表失败";
      toast.error(message);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    loadList();
  }, [loadList, refreshKey]);

  /** 审批后刷新 */
  const handleApprovalSuccess = () => {
    setApprovalTarget(null);
    onRefresh();
  };

  return (
    <>
      <Card>
        <CardHeader>
          <CardTitle>待审批发票</CardTitle>
        </CardHeader>
        <CardContent className="p-0">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>发票号</TableHead>
                <TableHead>状态</TableHead>
                <TableHead>销售方</TableHead>
                <TableHead className="text-right">金额</TableHead>
                <TableHead className="text-right">含税总额</TableHead>
                <TableHead>来源</TableHead>
                <TableHead>开票日期</TableHead>
                <TableHead className="w-[160px]">操作</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {loading && invoices.length === 0 && (
                <TableRow>
                  <TableCell colSpan={8} className="h-32 text-center text-muted-foreground">
                    <Loader2 className="mx-auto h-6 w-6 animate-spin" />
                  </TableCell>
                </TableRow>
              )}
              {!loading && invoices.length === 0 && (
                <TableRow>
                  <TableCell colSpan={8} className="h-32 text-center text-muted-foreground">
                    暂无待审批发票
                  </TableCell>
                </TableRow>
              )}
              {invoices.map((inv) => (
                <TableRow key={inv.id}>
                  <TableCell className="font-mono text-sm">{inv.invoice_no}</TableCell>
                  <TableCell>
                    <Badge variant="outline" className={getStatusBadgeClass(inv.status)}>
                      {getStatusLabel(inv.status)}
                    </Badge>
                  </TableCell>
                  <TableCell className="max-w-[140px] truncate">{inv.seller}</TableCell>
                  <TableCell className="text-right font-medium">{formatCurrency(inv.amount)}</TableCell>
                  <TableCell className="text-right">{formatCurrency(inv.total_amount)}</TableCell>
                  <TableCell>{getSourceTypeLabel(inv.source_type)}</TableCell>
                  <TableCell className="text-sm text-muted-foreground">{formatDate(inv.invoice_date)}</TableCell>
                  <TableCell>
                    <div className="flex gap-1">
                      <Button variant="ghost" size="sm" onClick={() => setDetailTarget(inv)}>
                        <Eye className="mr-1 h-3.5 w-3.5" />
                        详情
                      </Button>
                      <Button variant="ghost" size="sm" className="text-emerald-600 hover:text-emerald-700"
                        onClick={() => setApprovalTarget(inv)}>
                        <Check className="mr-1 h-3.5 w-3.5" />
                        审批
                      </Button>
                    </div>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </CardContent>
      </Card>

      {/* 审批 Dialog */}
      <ApprovalDialog
        open={!!approvalTarget}
        onOpenChange={(v) => { if (!v) setApprovalTarget(null); }}
        invoice={approvalTarget}
        onSuccess={handleApprovalSuccess}
      />

      {/* 详情 Dialog */}
      <InvoiceDetailDialog
        open={!!detailTarget}
        onOpenChange={(v) => { if (!v) setDetailTarget(null); }}
        invoice={detailTarget}
        onSuccess={handleApprovalSuccess}
      />
    </>
  );
}
