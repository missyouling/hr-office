/* eslint-disable @typescript-eslint/no-explicit-any */
"use client";

import { useState, useEffect, useCallback } from "react";
import { toast } from "sonner";
import {
  Search, Plus, Pencil, Trash2, PackageOpen, Upload, Download, FileDown, Loader2,
} from "lucide-react";

import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Badge } from "@/components/ui/badge";
import { ScrollArea } from "@/components/ui/scroll-area";
import {
  Table, TableBody, TableCell, TableHead, TableHeader, TableRow,
} from "@/components/ui/table";
import {
  Dialog, DialogContent, DialogHeader, DialogTitle,
} from "@/components/ui/dialog";
import { Textarea } from "@/components/ui/textarea";

import { officeApi } from "@/lib/api-office";
import { formatCurrency, exportCsv } from "../utils";
import SupplyDialog from "../dialogs/SupplyDialog";

/** 用品字典 Tab：搜索/筛选、表格、新增/编辑/删除、CSV 导入/导出 */
export default function DictionaryTab() {
  const [supplies, setSupplies] = useState<any[]>([]);
  const [categories, setCategories] = useState<any[]>([]);
  const [loading, setLoading] = useState(false);
  const [keyword, setKeyword] = useState("");
  const [category, setCategory] = useState("all");
  const [statusFilter, setStatusFilter] = useState("all");

  const [dialogOpen, setDialogOpen] = useState(false);
  const [editingSupply, setEditingSupply] = useState<any>(null);
  const [delOpen, setDelOpen] = useState(false);
  const [delTarget, setDelTarget] = useState<any>(null);
  const [importOpen, setImportOpen] = useState(false);
  const [importText, setImportText] = useState("");
  const [importing, setImporting] = useState(false);
  const [importResult, setImportResult] = useState<{ ok: number; err: number } | null>(null);

  useEffect(() => {
    officeApi.categories.list().then(r => setCategories(r.data || [])).catch(() => {});
  }, []);

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const params: Record<string, string> = { limit: "500" };
      if (keyword) params.keyword = keyword;
      if (category !== "all") params.category_id = category;
      if (statusFilter !== "all") params.status = statusFilter;
      const r = await officeApi.supplies.list(params);
      setSupplies(r.data || []);
    } catch (e: any) {
      toast.error("加载失败", { description: e.message });
    } finally {
      setLoading(false);
    }
  }, [keyword, category, statusFilter]);

  useEffect(() => { load(); }, [load]);

  const handleExport = async () => {
    try {
      const blob = await officeApi.supplies.export();
      const a = document.createElement("a");
      a.href = URL.createObjectURL(blob);
      a.download = `用品字典_${new Date().toISOString().substring(0, 10)}.csv`;
      a.click();
      URL.revokeObjectURL(a.href);
      toast.success("已导出");
    } catch (e: any) {
      toast.error("导出失败", { description: e.message });
    }
  };

  const handleExportFiltered = () => {
    if (supplies.length === 0) { toast.error("无数据可导出"); return; }
    const headers = ["品名", "规格", "单位", "参考单价", "分类", "状态", "备注"];
    const rows = supplies.map(s => [
      s.name, s.spec || "", s.unit || "", s.reference_price ?? "",
      categories.find(c => c.id === s.category_id)?.name || "", s.status === "active" ? "启用" : "停用", s.remark || "",
    ]);
    exportCsv(`用品字典_筛选结果_${new Date().toISOString().substring(0, 10)}.csv`, [headers, ...rows]);
    toast.success("已导出筛选结果");
  };

  const handleFileSelect = (file: File) => {
    const reader = new FileReader();
    reader.onload = (e) => {
      const buf = e.target?.result as ArrayBuffer;
      if (!buf) return;
      let text = new TextDecoder("utf-8", { fatal: false }).decode(buf);
      if (text.indexOf("\uFFFD") >= 0) {
        try {
          const gbk = new TextDecoder("gbk", { fatal: false }).decode(buf);
          if (gbk.indexOf("\uFFFD") < 0) text = gbk;
        } catch { /* 保留 UTF-8 */ }
      }
      if (text.charCodeAt(0) === 0xFEFF) text = text.slice(1);
      setImportText(text);
    };
    reader.readAsArrayBuffer(file);
  };

  const handleImport = async () => {
    if (!importText.trim()) { toast.error("请选择 CSV 文件"); return; }
    setImporting(true);
    setImportResult(null);
    try {
      const blob = new Blob([importText], { type: "text/csv;charset=utf-8" });
      const r = await officeApi.supplies.import(new File([blob], "import.csv"));
      const imported = (r.data as any)?.imported ?? 0;
      setImportResult({ ok: imported, err: 0 });
      toast.success(`导入完成，成功 ${imported} 条`);
      load();
    } catch (e: any) {
      toast.error("导入失败", { description: e.message });
    } finally {
      setImporting(false);
    }
  };

  const confirmDelete = async () => {
    if (!delTarget) return;
    try {
      await officeApi.supplies.remove(delTarget.id);
      toast.success("已删除");
      setDelOpen(false);
      setDelTarget(null);
      load();
    } catch (e: any) {
      toast.error("删除失败", { description: e.message });
    }
  };

  const handleDialogClose = (refresh: boolean) => {
    setDialogOpen(false);
    setEditingSupply(null);
    if (refresh) load();
  };

  const getCategoryName = (catId: number) =>
    categories.find(c => c.id === catId)?.name || "-";

  return (
    <div className="space-y-4">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <div className="flex gap-2">
          <Button onClick={() => { setEditingSupply(null); setDialogOpen(true); }}>
            <Plus className="mr-1 h-4 w-4" />新增用品
          </Button>
          <Button variant="outline" onClick={handleExport}>
            <Download className="mr-1 h-4 w-4" />导出全部
          </Button>
          <Button variant="outline" onClick={handleExportFiltered}>
            <FileDown className="mr-1 h-4 w-4" />导出筛选
          </Button>
          <Button variant="outline" onClick={() => { setImportText(""); setImportResult(null); setImportOpen(true); }}>
            <Upload className="mr-1 h-4 w-4" />导入
          </Button>
        </div>
      </div>

      <Card>
        <CardContent className="pt-4">
          <div className="flex flex-wrap gap-3">
            <div className="relative flex-1 min-w-[200px]">
              <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
              <Input placeholder="搜索品名 / 规格..." value={keyword}
                onChange={e => setKeyword(e.target.value)} className="pl-9" />
            </div>
            <Select value={category} onValueChange={setCategory}>
              <SelectTrigger className="w-[140px]"><SelectValue placeholder="全部分类" /></SelectTrigger>
              <SelectContent>
                <SelectItem value="all">全部分类</SelectItem>
                {categories.map(c => <SelectItem key={c.id} value={String(c.id)}>{c.name}</SelectItem>)}
              </SelectContent>
            </Select>
            <Select value={statusFilter} onValueChange={setStatusFilter}>
              <SelectTrigger className="w-[120px]"><SelectValue placeholder="全部状态" /></SelectTrigger>
              <SelectContent>
                <SelectItem value="all">全部状态</SelectItem>
                <SelectItem value="active">启用</SelectItem>
                <SelectItem value="inactive">停用</SelectItem>
              </SelectContent>
            </Select>
            <Button variant="outline" onClick={() => { setKeyword(""); setCategory("all"); setStatusFilter("all"); }}>
              重置
            </Button>
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardContent className="p-0">
          <ScrollArea className="h-[calc(100vh-340px)] rounded-md border">
            <Table>
              <TableHeader className="sticky top-0 bg-muted">
                <TableRow>
                  <TableHead className="text-xs font-medium text-muted-foreground">品名</TableHead>
                  <TableHead className="text-xs font-medium text-muted-foreground">规格</TableHead>
                  <TableHead className="text-xs font-medium text-muted-foreground">单位</TableHead>
                  <TableHead className="text-xs font-medium text-muted-foreground text-right">参考单价</TableHead>
                  <TableHead className="text-xs font-medium text-muted-foreground">分类</TableHead>
                  <TableHead className="text-xs font-medium text-muted-foreground text-center">状态</TableHead>
                  <TableHead className="text-xs font-medium text-muted-foreground">备注</TableHead>
                  <TableHead className="text-xs font-medium text-muted-foreground text-center w-[100px]">操作</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {loading ? (
                  <TableRow><TableCell colSpan={8} className="h-32 text-center text-muted-foreground">加载中...</TableCell></TableRow>
                ) : supplies.length === 0 ? (
                  <TableRow><TableCell colSpan={8} className="h-32 text-center text-muted-foreground">
                    <PackageOpen className="mx-auto h-10 w-10 mb-2 opacity-40" /><p>暂无用品数据</p>
                  </TableCell></TableRow>
                ) : supplies.map(s => (
                  <TableRow key={s.id}>
                    <TableCell className="text-sm font-medium">{s.name}</TableCell>
                    <TableCell className="text-xs text-muted-foreground max-w-[120px] truncate">{s.spec || "-"}</TableCell>
                    <TableCell className="text-sm">{s.unit || "-"}</TableCell>
                    <TableCell className="text-sm text-right font-mono">{formatCurrency(s.reference_price)}</TableCell>
                    <TableCell><Badge variant="secondary" className="text-xs">{getCategoryName(s.category_id)}</Badge></TableCell>
                    <TableCell className="text-center">
                      <Badge variant={s.status === "active" ? "default" : "secondary"} className="text-xs">
                        {s.status === "active" ? "启用" : "停用"}
                      </Badge>
                    </TableCell>
                    <TableCell className="text-xs text-muted-foreground max-w-[160px] truncate">{s.remark || "-"}</TableCell>
                    <TableCell className="text-center">
                      <Button variant="ghost" size="icon" className="h-8 w-8" onClick={() => { setEditingSupply(s); setDialogOpen(true); }}><Pencil className="h-4 w-4" /></Button>
                      <Button variant="ghost" size="icon" className="h-8 w-8" onClick={() => { setDelTarget(s); setDelOpen(true); }}><Trash2 className="h-4 w-4 text-red-500" /></Button>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </ScrollArea>
        </CardContent>
      </Card>

      <SupplyDialog open={dialogOpen} onClose={handleDialogClose} supply={editingSupply} />

      <Dialog open={delOpen} onOpenChange={setDelOpen}>
        <DialogContent className="sm:max-w-[400px]">
          <DialogHeader><DialogTitle>确认删除</DialogTitle></DialogHeader>
          <p className="text-sm text-muted-foreground py-2">确认删除用品「{delTarget?.name}」？</p>
          <div className="flex justify-end gap-2">
            <Button variant="outline" onClick={() => setDelOpen(false)}>取消</Button>
            <Button variant="destructive" onClick={confirmDelete}>确认删除</Button>
          </div>
        </DialogContent>
      </Dialog>

      <Dialog open={importOpen} onOpenChange={setImportOpen}>
        <DialogContent className="sm:max-w-[560px]">
          <DialogHeader><DialogTitle>导入用品 CSV</DialogTitle></DialogHeader>
          <div className="space-y-3 py-2">
            <Input type="file" accept=".csv" onChange={e => { const f = e.target.files?.[0]; if (f) handleFileSelect(f); }} />
            <Textarea className="h-44 text-xs font-mono" placeholder="选择 CSV 文件后此处预览内容..."
              value={importText} onChange={e => setImportText(e.target.value)} />
            {importResult && <p className="text-sm text-muted-foreground">成功 {importResult.ok} 条，失败 {importResult.err} 条</p>}
          </div>
          <div className="flex justify-end gap-2">
            <Button variant="outline" onClick={() => setImportOpen(false)}>取消</Button>
            <Button onClick={handleImport} disabled={importing || !importText.trim()}>
              {importing ? <Loader2 className="mr-2 h-4 w-4 animate-spin" /> : null}开始导入
            </Button>
          </div>
        </DialogContent>
      </Dialog>
    </div>
  );
}
