"use client";

import { useEffect, useState, useCallback } from "react";
import { toast } from "sonner";
import { Plus, Search, Loader2, ChevronLeft, ChevronRight } from "lucide-react";

import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Badge } from "@/components/ui/badge";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { PermissionGate } from "@/components/permission-gate";
import { invoiceApi, type Invoice, type InvoiceListParams } from "@/lib/api-invoice";
import {
  getStatusLabel,
  getStatusBadgeClass,
  getSourceTypeLabel,
  formatCurrency,
  formatDate,
  STATUS_OPTIONS,
  SOURCE_TYPE_OPTIONS,
} from "../utils";

interface InvoicesTabProps {
  onViewDetail: (invoice: Invoice) => void;
  onCreateNew: () => void;
  refreshKey: number;
}

const PAGE_SIZE = 15;

export default function InvoicesTab({ onViewDetail, onCreateNew, refreshKey }: InvoicesTabProps) {
  const [invoices, setInvoices] = useState<Invoice[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [loading, setLoading] = useState(false);

  // 筛选条件
  const [status, setStatus] = useState("");
  const [sourceType, setSourceType] = useState("");
  const [keyword, setKeyword] = useState("");

  /** 加载列表 */
  const loadList = useCallback(async () => {
    setLoading(true);
    try {
      const params: InvoiceListParams = {
        page,
        page_size: PAGE_SIZE,
      };
      if (status) params.status = status;
      if (sourceType) params.source_type = sourceType;
      if (keyword.trim()) params.keyword = keyword.trim();

      const data = await invoiceApi.list(params);
      setInvoices(data.items ?? []);
      setTotal(data.total ?? 0);
    } catch (err) {
      const message = err instanceof Error ? err.message : "加载发票列表失败";
      toast.error(message);
    } finally {
      setLoading(false);
    }
  }, [page, status, sourceType, keyword]);

  useEffect(() => {
    loadList();
  }, [loadList, refreshKey]);

  /** 筛选条件变更时重置到第 1 页 */
  const handleFilterChange = (setter: (v: string) => void, value: string) => {
    setter(value);
    setPage(1);
  };

  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE));

  return (
    <Card>
      <CardHeader className="pb-3">
        <div className="flex flex-wrap items-center justify-between gap-4">
          <CardTitle>发票列表</CardTitle>
          <div className="flex flex-wrap gap-2">
            <PermissionGate resource="invoice" action="create">
              <Button size="sm" onClick={onCreateNew}>
                <Plus className="mr-1 h-4 w-4" />
                新建发票
              </Button>
            </PermissionGate>
          </div>
        </div>
      </CardHeader>

      <CardContent className="p-0">
        {/* 筛选条 */}
        <div className="flex flex-wrap items-center gap-3 px-4 pb-3">
          <Select value={status} onValueChange={(v) => handleFilterChange(setStatus, v === "all" ? "" : v)}>
            <SelectTrigger className="w-[120px]">
              <SelectValue placeholder="全部状态" />
            </SelectTrigger>
            <SelectContent>
              {STATUS_OPTIONS.map((opt) => (
                <SelectItem key={opt.value || "all"} value={opt.value || "all"}>
                  {opt.label}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>

          <Select value={sourceType} onValueChange={(v) => handleFilterChange(setSourceType, v === "all" ? "" : v)}>
            <SelectTrigger className="w-[140px]">
              <SelectValue placeholder="全部来源" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">全部来源</SelectItem>
              {SOURCE_TYPE_OPTIONS.map((opt) => (
                <SelectItem key={opt.value} value={opt.value}>
                  {opt.label}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>

          <div className="relative flex-1 min-w-[180px]">
            <Search className="absolute left-2.5 top-2.5 h-4 w-4 text-muted-foreground" />
            <Input
              placeholder="搜索发票号/销售方..."
              value={keyword}
              onChange={(e) => handleFilterChange(setKeyword, e.target.value)}
              className="pl-8"
            />
          </div>
        </div>

        {/* 表格 */}
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead className="w-[130px]">发票号</TableHead>
              <TableHead className="w-[80px]">状态</TableHead>
              <TableHead>销售方</TableHead>
              <TableHead className="text-right">金额</TableHead>
              <TableHead className="text-right">含税总额</TableHead>
              <TableHead>来源</TableHead>
              <TableHead className="w-[100px]">开票日期</TableHead>
              <TableHead className="w-[80px]">操作</TableHead>
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
                  暂无发票数据
                </TableCell>
              </TableRow>
            )}
            {invoices.map((inv) => (
              <TableRow key={inv.id} className="cursor-pointer hover:bg-muted/50" onClick={() => onViewDetail(inv)}>
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
                  <Button variant="ghost" size="sm" onClick={(e) => { e.stopPropagation(); onViewDetail(inv); }}>
                    详情
                  </Button>
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>

        {/* 分页 */}
        {totalPages > 1 && (
          <div className="flex items-center justify-between border-t px-4 py-3">
            <p className="text-sm text-muted-foreground">
              共 {total} 条，第 {page} / {totalPages} 页
            </p>
            <div className="flex gap-2">
              <Button variant="outline" size="sm" onClick={() => setPage((p) => Math.max(1, p - 1))} disabled={page <= 1}>
                <ChevronLeft className="h-4 w-4" />
              </Button>
              <Button variant="outline" size="sm" onClick={() => setPage((p) => Math.min(totalPages, p + 1))} disabled={page >= totalPages}>
                <ChevronRight className="h-4 w-4" />
              </Button>
            </div>
          </div>
        )}
      </CardContent>
    </Card>
  );
}
