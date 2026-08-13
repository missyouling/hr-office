"use client";

import { useCallback, useEffect, useState } from "react";
import { ChevronLeft, ChevronRight, Download, Eye, FileArchive, Loader2, Search } from "lucide-react";
import { toast } from "sonner";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { invoiceApi, type Invoice } from "@/lib/api-invoice";
import { getRuntimeConfig } from "@/lib/runtime-config";
import { formatCurrency, formatDate, getSourceTypeLabel, SOURCE_TYPE_OPTIONS } from "../utils";
import { ARCHIVE_PAGE_SIZE, buildArchiveQuery, clampArchivePage, normalizeArchiveSummary, type ArchiveFilters } from "./invoice-archive-logic";

type ArchiveStatus = "pending" | "confirmed" | "voided";
interface ArchiveInvoice extends Invoice { archive_status?: ArchiveStatus; }
interface ArchiveListResponse { items: ArchiveInvoice[]; total: number; }
interface InvoiceArchiveTabProps { onViewDetail: (invoice: Invoice) => void; refreshKey: number; }

const INITIAL_FILTERS: ArchiveFilters = { archiveStatus: "", sourceType: "", keyword: "" };
const STATUS_OPTIONS: Array<{ value: ArchiveStatus; label: string }> = [
  { value: "pending", label: "待确认" }, { value: "confirmed", label: "已确认" }, { value: "voided", label: "已作废" },
];
const STATUS_CLASSES: Record<ArchiveStatus, string> = {
  pending: "border-amber-200 bg-amber-50 text-amber-700", confirmed: "border-emerald-200 bg-emerald-50 text-emerald-700", voided: "border-slate-200 bg-slate-50 text-slate-600",
};

function getApiBase(): string {
  const base = getRuntimeConfig().API_BASE ?? process.env.NEXT_PUBLIC_API_BASE_URL;
  return base ? base.replace(/\/+$/, "") : `${window.location.origin}/api`;
}

async function requestArchive<T>(path: string): Promise<T> {
  const token = localStorage.getItem("token");
  const response = await fetch(`${getApiBase()}${path}`, { headers: token ? { Authorization: `Bearer ${token}` } : {}, cache: "no-store" });
  if (!response.ok) throw new Error(`请求归档数据失败（${response.status}）`);
  return response.json() as Promise<T>;
}

function getStatusLabel(status?: ArchiveStatus): string {
  return STATUS_OPTIONS.find((option) => option.value === status)?.label ?? "待确认";
}

export default function InvoiceArchiveTab({ onViewDetail, refreshKey }: InvoiceArchiveTabProps) {
  const [filters, setFilters] = useState(INITIAL_FILTERS);
  const [invoices, setInvoices] = useState<ArchiveInvoice[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [summary, setSummary] = useState(() => normalizeArchiveSummary(0));
  const [isLoading, setIsLoading] = useState(false);
  const [isExporting, setIsExporting] = useState(false);

  const loadData = useCallback(async () => {
    setIsLoading(true);
    try {
      const query = buildArchiveQuery(filters, page).toString();
      const [list, stats] = await Promise.all([requestArchive<ArchiveListResponse>(`/invoices?${query}`), invoiceApi.stats()]);
      setInvoices(list.items ?? []);
      setTotal(list.total ?? 0);
      setSummary(normalizeArchiveSummary(stats.total, stats.by_status));
    } catch (error) {
      toast.error(error instanceof Error ? error.message : "加载归档列表失败");
    } finally {
      setIsLoading(false);
    }
  }, [filters, page]);

  useEffect(() => { loadData(); }, [loadData, refreshKey]);

  const updateFilter = (key: keyof ArchiveFilters, value: string) => {
    setFilters((current) => ({ ...current, [key]: value }));
    setPage(1);
  };

  const handleExport = async () => {
    setIsExporting(true);
    try {
      const query = buildArchiveQuery(filters).toString();
      const token = localStorage.getItem("token");
      const response = await fetch(`${getApiBase()}/invoices/export${query ? `?${query}` : ""}`, { headers: token ? { Authorization: `Bearer ${token}` } : {} });
      if (!response.ok) throw new Error("导出归档 CSV 失败");
      const url = URL.createObjectURL(await response.blob());
      const link = document.createElement("a");
      link.href = url;
      link.download = "发票归档.csv";
      link.click();
      URL.revokeObjectURL(url);
      toast.success("归档 CSV 已开始下载");
    } catch (error) {
      toast.error(error instanceof Error ? error.message : "导出归档 CSV 失败");
    } finally {
      setIsExporting(false);
    }
  };

  const totalPages = Math.max(1, Math.ceil(total / ARCHIVE_PAGE_SIZE));
  const summaries = [["归档总数", summary.total, "text-foreground"], ["待确认", summary.pending, "text-amber-700"], ["已确认", summary.confirmed, "text-emerald-700"], ["已作废", summary.voided, "text-slate-600"]] as const;

  return <div className="space-y-4">
    <div className="grid grid-cols-2 gap-3 lg:grid-cols-4">{summaries.map(([label, value, className]) => <Card key={label} className="border-muted shadow-sm"><CardContent className="p-4"><p className="text-xs text-muted-foreground">{label}</p><p className={`mt-1 text-2xl font-semibold tabular-nums ${className}`}>{value}</p></CardContent></Card>)}</div>
    <Card><CardHeader className="pb-3"><div className="flex flex-wrap items-center justify-between gap-3"><div><CardTitle className="flex items-center gap-2"><FileArchive className="h-5 w-5 text-primary" />归档管理</CardTitle><p className="mt-1 text-sm font-normal text-muted-foreground">查询、核对并导出已归档的发票凭证</p></div><Button size="sm" variant="outline" onClick={handleExport} disabled={isExporting}><Download className="mr-1.5 h-4 w-4" />{isExporting ? "正在导出" : "导出 CSV"}</Button></div></CardHeader>
      <CardContent className="p-0"><div className="flex flex-wrap gap-3 px-4 pb-4">
        <Select value={filters.archiveStatus} onValueChange={(value) => updateFilter("archiveStatus", value === "all" ? "" : value)}><SelectTrigger className="w-[132px]"><SelectValue placeholder="归档状态" /></SelectTrigger><SelectContent><SelectItem value="all">全部归档状态</SelectItem>{STATUS_OPTIONS.map((option) => <SelectItem key={option.value} value={option.value}>{option.label}</SelectItem>)}</SelectContent></Select>
        <Select value={filters.sourceType} onValueChange={(value) => updateFilter("sourceType", value === "all" ? "" : value)}><SelectTrigger className="w-[132px]"><SelectValue placeholder="全部来源" /></SelectTrigger><SelectContent><SelectItem value="all">全部来源</SelectItem>{SOURCE_TYPE_OPTIONS.map((option) => <SelectItem key={option.value} value={option.value}>{option.label}</SelectItem>)}</SelectContent></Select>
        <div className="relative min-w-[200px] flex-1"><Search className="absolute left-2.5 top-2.5 h-4 w-4 text-muted-foreground" /><Input value={filters.keyword} onChange={(event) => updateFilter("keyword", event.target.value)} className="pl-8" placeholder="搜索票号、用途..." /></div>
      </div><div className="overflow-x-auto"><Table><TableHeader><TableRow><TableHead>票号 / 凭证</TableHead><TableHead>归档状态</TableHead><TableHead>销售方</TableHead><TableHead className="text-right">价税合计</TableHead><TableHead>来源</TableHead><TableHead>开票日期</TableHead><TableHead>操作</TableHead></TableRow></TableHeader><TableBody>
        {isLoading && invoices.length === 0 && <TableRow><TableCell colSpan={7} className="h-32 text-center"><Loader2 className="mx-auto h-6 w-6 animate-spin text-muted-foreground" /></TableCell></TableRow>}
        {!isLoading && invoices.length === 0 && <TableRow><TableCell colSpan={7} className="h-32 text-center text-muted-foreground">暂无符合条件的归档发票</TableCell></TableRow>}
        {invoices.map((invoice) => <TableRow key={invoice.id} className="cursor-pointer hover:bg-muted/50" onClick={() => onViewDetail(invoice)}><TableCell><p className="font-mono text-sm">{invoice.invoice_no || "-"}</p><p className="mt-0.5 text-xs text-muted-foreground">{invoice.voucher_type || "发票凭证"}</p></TableCell><TableCell><Badge variant="outline" className={STATUS_CLASSES[invoice.archive_status ?? "pending"]}>{getStatusLabel(invoice.archive_status)}</Badge></TableCell><TableCell className="max-w-[180px] truncate">{invoice.seller}</TableCell><TableCell className="text-right font-medium tabular-nums">{formatCurrency(invoice.total_amount)}</TableCell><TableCell>{getSourceTypeLabel(invoice.source_type)}</TableCell><TableCell className="whitespace-nowrap text-sm text-muted-foreground">{formatDate(invoice.invoice_date)}</TableCell><TableCell><Button variant="ghost" size="sm" onClick={(event) => { event.stopPropagation(); onViewDetail(invoice); }}><Eye className="mr-1 h-3.5 w-3.5" />详情</Button></TableCell></TableRow>)}
      </TableBody></Table></div>{total > 0 && <div className="flex items-center justify-between border-t px-4 py-3"><p className="text-sm text-muted-foreground">共 {total} 条，第 {page} / {totalPages} 页</p><div className="flex gap-2"><Button variant="outline" size="sm" aria-label="上一页" disabled={page <= 1} onClick={() => setPage((current) => clampArchivePage(current - 1, total))}><ChevronLeft className="h-4 w-4" /></Button><Button variant="outline" size="sm" aria-label="下一页" disabled={page >= totalPages} onClick={() => setPage((current) => clampArchivePage(current + 1, total))}><ChevronRight className="h-4 w-4" /></Button></div></div>}</CardContent>
    </Card>
  </div>;
}
