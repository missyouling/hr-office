/* eslint-disable @typescript-eslint/no-explicit-any */
"use client";

import { useState, useEffect, useCallback, useRef } from "react";
import { toast } from "sonner";
import {
  Search, Plus, FileText, Trash2, Eye, Pencil, Save, X, RotateCcw, Loader2, Printer,
} from "lucide-react";

import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Badge } from "@/components/ui/badge";
import { ScrollArea } from "@/components/ui/scroll-area";
import {
  Table, TableBody, TableCell, TableHead, TableHeader, TableRow,
} from "@/components/ui/table";
import {
  Dialog, DialogContent, DialogHeader, DialogTitle,
} from "@/components/ui/dialog";
import { Label } from "@/components/ui/label";

import { officeApi } from "@/lib/api-office";
import { formatCurrency, formatShortDate, formatPurchaseDate, todayStr } from "../utils";

interface FormItem {
  supply_id: number;
  supply_name: string;
  supply_spec: string;
  unit: string;
  reference_price: number;
  date: string;
  quantity: number;
  unit_price: number;
  unit_price_str: string;
  subtotal: number;
}

type DialogType = "new" | "view" | "edit" | null;

/** 采购单 Tab：列表 + 新建/编辑/查看 + 打印预览 + 复制草稿 */
export default function PurchasesTab() {
  const [purchases, setPurchases] = useState<any[]>([]);
  const [filters, setFilters] = useState({ keyword: "", date_from: "", date_to: "" });
  const [page, setPage] = useState(1);
  const [hasMore, setHasMore] = useState(true);
  const [loading, setLoading] = useState(false);
  const [totalSum, setTotalSum] = useState(0);
  const observerRef = useRef<IntersectionObserver | null>(null);
  const sentinelRef = useRef<HTMLDivElement>(null);

  const [dialogType, setDialogType] = useState<DialogType>(null);
  const [dialogPurchase, setDialogPurchase] = useState<any>(null);
  const [dialogItems, setDialogItems] = useState<any[]>([]);
  const [saving, setSaving] = useState(false);

  const [suppliers, setSuppliers] = useState<any[]>([]);
  const [supplies, setSupplies] = useState<any[]>([]);
  const [formDateFrom, setFormDateFrom] = useState(todayStr());
  const [formDateTo, setFormDateTo] = useState(todayStr());
  const [formItems, setFormItems] = useState<FormItem[]>([]);
  const [formSupplierId, setFormSupplierId] = useState<number | null>(null);
  const [formRemark, setFormRemark] = useState("");
  const [searchQuery, setSearchQuery] = useState("");
  const [searchOpen, setSearchOpen] = useState(false);
  const searchRef = useRef<HTMLInputElement>(null);
  const itemsScrollRef = useRef<HTMLDivElement>(null);

  const [confirmOpen, setConfirmOpen] = useState(false);
  const [confirmMsg, setConfirmMsg] = useState("");
  const [confirmAction, setConfirmAction] = useState<() => void>(() => {});

  // loading 是防重入的守卫状态，加入 dep 会导致循环更新
  const loadList = useCallback(async (reset = false) => {
    if (loading) return;
    setLoading(true);
    try {
      const p = reset ? 1 : page;
      const params: Record<string, string> = { page: String(p), limit: "20", ...filters };
      const r = await officeApi.purchases.list(params);
      const items = r.data || [];
      if (reset) { setPurchases(items); setPage(2); }
      else { setPurchases(prev => [...prev, ...items]); setPage(p + 1); }
      setHasMore(items.length >= 20);
      setTotalSum(items.reduce((s: number, it: any) => s + (it.total_amount || 0), 0));
    } catch (e: any) { toast.error("加载失败", { description: e.message }); }
    finally { setLoading(false); }
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [page, filters]);

  useEffect(() => { setPage(1); setHasMore(true); loadList(true); }, [filters]); // eslint-disable-line react-hooks/exhaustive-deps
  useEffect(() => {
    if (observerRef.current) observerRef.current.disconnect();
    if (!sentinelRef.current) return;
    observerRef.current = new IntersectionObserver(entries => {
      if (entries[0].isIntersecting && hasMore && !loading) loadList();
    }, { rootMargin: "200px" });
    observerRef.current.observe(sentinelRef.current);
    return () => observerRef.current?.disconnect();
  }, [hasMore, loading]); // eslint-disable-line react-hooks/exhaustive-deps

  useEffect(() => {
    officeApi.suppliers.list().then(r => {
      const list = r.data || [];
      setSuppliers(list);
      if (list.length) setFormSupplierId(prev => prev ?? list[0].id);
    }).catch(() => {});
  }, []);

  useEffect(() => {
    if (!searchQuery.trim()) { setSupplies([]); return; }
    const t = setTimeout(async () => {
      try {
        const r = await officeApi.supplies.list({ keyword: searchQuery, limit: "10", status: "active" });
        setSupplies(r.data || []);
      } catch { setSupplies([]); }
    }, 250);
    return () => clearTimeout(t);
  }, [searchQuery]);

  const addItem = (sup: any) => {
    const lastDate = formItems.length ? formItems[formItems.length - 1].date : todayStr();
    setFormItems(prev => [...prev, {
      supply_id: sup.id, supply_name: sup.name, supply_spec: sup.spec || "", unit: sup.unit || "个",
      reference_price: sup.reference_price || 0, date: lastDate,
      quantity: 1, unit_price: sup.reference_price || 0,
      unit_price_str: (sup.reference_price || 0).toFixed(2), subtotal: sup.reference_price || 0,
    }]);
    setSearchQuery(""); setSearchOpen(false);
    setTimeout(() => { if (itemsScrollRef.current) itemsScrollRef.current.scrollTop = itemsScrollRef.current.scrollHeight; }, 60);
    setTimeout(() => searchRef.current?.focus(), 50);
  };

  const updateQty = (idx: number, qty: number) =>
    setFormItems(prev => prev.map((item, i) => i === idx ? { ...item, quantity: Math.max(1, qty), subtotal: Math.max(1, qty) * item.unit_price } : item));

  const updatePrice = (idx: number, raw: string) => {
    const price = parseFloat(raw);
    const valid = isNaN(price) ? 0 : Math.max(0, price);
    setFormItems(prev => prev.map((item, i) => i === idx ? { ...item, unit_price_str: raw, unit_price: valid, subtotal: item.quantity * valid } : item));
  };

  const blurPrice = (idx: number) =>
    setFormItems(prev => prev.map((item, i) => i === idx ? { ...item, unit_price_str: item.unit_price.toFixed(2) } : item));

  const removeItem = (idx: number) => setFormItems(prev => prev.filter((_, i) => i !== idx));

  const openNew = () => {
    setFormDateFrom(todayStr()); setFormDateTo(todayStr()); setFormItems([]);
    setFormSupplierId(suppliers.length ? suppliers[0].id : null); setFormRemark("");
    setDialogPurchase(null); setDialogType("new");
  };

  const openView = async (id: number) => {
    try {
      const r = await officeApi.purchases.get(id);
      setDialogPurchase(r.data); setDialogItems((r.data as any)?.items || []); setDialogType("view");
    } catch (e: any) { toast.error("加载失败", { description: e.message }); }
  };

  const openEdit = async (id: number) => {
    try {
      const r = await officeApi.purchases.get(id);
      const d = r.data as any;
      setDialogPurchase(d);
      const ds = d.purchase_date || "";
      if (ds.includes("~")) { const [f, t] = ds.split("~"); setFormDateFrom(f); setFormDateTo(t); }
      else { setFormDateFrom(ds); setFormDateTo(ds); }
      setFormSupplierId(d.supplier_id || null); setFormRemark(d.remark || "");
      setFormItems((d.items || []).map((i: any) => ({
        supply_id: i.supply_id, supply_name: i.supply_name, supply_spec: i.supply_spec || "",
        unit: i.unit || "个", reference_price: i.reference_price || 0,
        date: i.date || d.purchase_date, quantity: i.quantity, unit_price: i.unit_price,
        unit_price_str: Number(i.unit_price).toFixed(2), subtotal: i.subtotal,
      })));
      setDialogType("edit");
    } catch (e: any) { toast.error("加载失败", { description: e.message }); }
  };

  const handleSave = async () => {
    if (formItems.length === 0) { toast.error("请至少添加一项用品"); return; }
    setSaving(true);
    try {
      const dates = formItems.map(i => i.date).filter(Boolean);
      let purchaseDate: string;
      if (dates.length) {
        const minD = dates.reduce((a, b) => (a < b ? a : b));
        const maxD = dates.reduce((a, b) => (a > b ? a : b));
        purchaseDate = minD === maxD ? minD : `${minD}~${maxD}`;
      } else {
        purchaseDate = formDateFrom === formDateTo ? formDateFrom : `${formDateFrom}~${formDateTo}`;
      }
      const body = {
        purchase_date: purchaseDate,
        items: formItems.map(i => ({ supply_id: i.supply_id, quantity: i.quantity, unit_price: i.unit_price, date: i.date })),
        supplier_id: formSupplierId ?? undefined, remark: formRemark.trim(),
      };
      if (dialogType === "edit" && dialogPurchase) {
        await officeApi.purchases.update(dialogPurchase.id, body);
        toast.success("采购单已更新");
      } else {
        await officeApi.purchases.create(body);
        toast.success("采购单已保存");
      }
      setDialogType(null);
      setFilters({ keyword: "", date_from: "", date_to: "" });
    } catch (e: any) { toast.error("保存失败", { description: e.message }); }
    finally { setSaving(false); }
  };

  const handleCopy = async (id: number) => {
    try { await officeApi.purchases.copy(id); toast.success("已复制采购单"); loadList(true); }
    catch (e: any) { toast.error("复制失败", { description: e.message }); }
  };

  const handlePrint = (p: any) => {
    const items = (p.items || []).map((it: any, i: number) =>
      `<tr><td>${i + 1}</td><td>${it.supply_name || ""}</td><td>${it.supply_spec || ""}</td><td>${it.unit || ""}</td><td>¥${Number(it.unit_price).toFixed(2)}</td><td>${it.quantity}</td><td>¥${Number(it.subtotal).toFixed(2)}</td></tr>`).join("");
    const html = `<!doctype html><html><head><meta charset="utf-8"><title>采购单 ${p.order_no || ""}</title>
<style>*{margin:0;padding:0}body{font-family:"Microsoft YaHei",sans-serif;padding:40px 50px;color:#333;font-size:13px}
h1{font-size:22px;margin-bottom:12px}.meta{color:#666;font-size:12px;margin-bottom:12px}
table{width:100%;border-collapse:collapse;margin-bottom:16px}
th{background:#1e40af;color:#fff;padding:7px 6px;text-align:center;font-size:12px;border:1px solid #1e40af}
td{padding:6px;border:1px solid #d1d5db;text-align:center;font-size:12px}
.total{font-size:18px;font-weight:bold;text-align:right;color:#dc2626}
@media print{body{padding:20px 30px}th{background:#1e40af!important;-webkit-print-color-adjust:exact;print-color-adjust:exact}}</style></head><body>
<h1>采购单</h1><div class="meta"><span>单号：${p.order_no || ""}</span><span>日期：${formatPurchaseDate(p.purchase_date)}</span><span>供应商：${p.supplier_name || ""}</span></div>
<table><thead><tr><th>序号</th><th>品名</th><th>规格</th><th>单位</th><th>单价</th><th>数量</th><th>小计</th></tr></thead><tbody>${items}</tbody></table>
<div class="total">合计：¥${Number(p.total_amount).toFixed(2)}</div>
<script>setTimeout(()=>window.print(),300)</script></body></html>`;
    const w = window.open("", "_blank");
    if (w) { w.document.write(html); w.document.close(); }
  };

  const handleDelete = (id: number, orderNo: string) => {
    setConfirmMsg(`确认删除采购单「${orderNo}」？`);
    setConfirmAction(() => async () => {
      try { await officeApi.purchases.remove(id); toast.success("已删除"); loadList(true); setConfirmOpen(false); }
      catch (e: any) { toast.error("删除失败", { description: e.message }); }
    });
    setConfirmOpen(true);
  };

  const totalAmount = formItems.reduce((s, i) => s + i.subtotal, 0);
  const isFormOpen = dialogType === "new" || dialogType === "edit";

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between flex-wrap gap-2">
        <div />
        <Button onClick={openNew} size="sm"><Plus className="mr-1 h-4 w-4" />新建采购单</Button>
      </div>

      <Card>
        <CardContent className="pt-4">
          <div className="flex flex-wrap items-center gap-2">
            <Input placeholder="搜索单号..." className="w-[180px] h-9 text-sm" value={filters.keyword}
              onChange={e => setFilters(f => ({ ...f, keyword: e.target.value }))} />
            <input type="date" className="h-9 rounded-lg border border-input bg-background px-3 text-sm font-mono"
              value={filters.date_from} onChange={e => setFilters(f => ({ ...f, date_from: e.target.value }))} />
            <span className="text-muted-foreground text-xs">~</span>
            <input type="date" className="h-9 rounded-lg border border-input bg-background px-3 text-sm font-mono"
              value={filters.date_to} onChange={e => setFilters(f => ({ ...f, date_to: e.target.value }))} />
            <Button variant="outline" size="sm" onClick={() => { setFilters({ keyword: "", date_from: "", date_to: "" }); }}>
              <RotateCcw className="h-3.5 w-3.5 mr-1" />重置
            </Button>
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardContent className="p-0">
          <ScrollArea className="h-[calc(100vh-400px)] rounded-md border">
            <Table>
              <TableHeader className="sticky top-0 bg-muted">
                <TableRow>
                  <TableHead className="text-xs">采购单号</TableHead><TableHead className="text-xs">日期范围</TableHead>
                  <TableHead className="text-xs text-center">品项</TableHead><TableHead className="text-xs">供应商</TableHead>
                  <TableHead className="text-xs text-right">金额</TableHead><TableHead className="text-xs text-center">付款状态</TableHead>
                  <TableHead className="text-xs">付款日期</TableHead><TableHead className="text-xs">备注</TableHead>
                  <TableHead className="text-xs text-center w-[120px]">操作</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {loading ? (
                  <TableRow><TableCell colSpan={9} className="h-24 text-center text-muted-foreground">加载中...</TableCell></TableRow>
                ) : purchases.length === 0 ? (
                  <TableRow><TableCell colSpan={9} className="h-24 text-center text-muted-foreground">暂无采购单</TableCell></TableRow>
                ) : purchases.map(p => (
                  <TableRow key={p.id}>
                    <TableCell className="font-mono text-xs font-medium whitespace-nowrap">{p.order_no}</TableCell>
                    <TableCell className="text-xs whitespace-nowrap">{formatPurchaseDate(p.purchase_date)}</TableCell>
                    <TableCell className="text-center text-xs">{p.item_count || 0} 项</TableCell>
                    <TableCell className="text-xs text-muted-foreground">{p.supplier_name || "-"}</TableCell>
                    <TableCell className="text-sm text-right font-mono font-bold">{formatCurrency(p.total_amount)}</TableCell>
                    <TableCell className="text-center">
                      <span className={`inline-block px-2 py-0.5 rounded-full text-xs font-medium ${p.payment_status === "已付款" ? "bg-green-100 text-green-700" : "bg-orange-100 text-orange-700"}`}>
                        {p.payment_status || "未付款"}
                      </span>
                    </TableCell>
                    <TableCell className="text-xs">{p.payment_date ? formatShortDate(p.payment_date) : "-"}</TableCell>
                    <TableCell className="text-xs text-muted-foreground truncate max-w-[100px]">{p.remark || "-"}</TableCell>
                    <TableCell className="text-center">
                      <Button variant="ghost" size="icon" className="h-7 w-7" onClick={() => openView(p.id)}><Eye className="h-3.5 w-3.5" /></Button>
                      <Button variant="ghost" size="icon" className="h-7 w-7" onClick={() => openEdit(p.id)}><Pencil className="h-3.5 w-3.5" /></Button>
                      <Button variant="ghost" size="icon" className="h-7 w-7" onClick={() => handleCopy(p.id)}><FileText className="h-3.5 w-3.5" /></Button>
                      <Button variant="ghost" size="icon" className="h-7 w-7" onClick={() => handleDelete(p.id, p.order_no)}><Trash2 className="h-3.5 w-3.5 text-red-500" /></Button>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </ScrollArea>
          {purchases.length > 0 && (
            <div className="flex justify-end items-center gap-2 border-t px-4 py-3 bg-muted/50">
              <span className="text-sm text-muted-foreground">共 {purchases.length} 单</span>
              <span className="text-sm font-medium">汇总金额：</span>
              <span className="text-lg font-bold text-red-600">{formatCurrency(totalSum)}</span>
            </div>
          )}
        </CardContent>
      </Card>
      <div ref={sentinelRef} className="py-3 text-center text-sm text-muted-foreground">
        {loading && <span>加载中...</span>}
        {!hasMore && purchases.length > 0 && <span>已显示所有采购单</span>}
      </div>

      {/* 新建/编辑弹窗 */}
      <Dialog open={isFormOpen} onOpenChange={v => { if (!v) setDialogType(null); }}>
        <DialogContent className="sm:max-w-[900px] max-h-[90vh] flex flex-col overflow-hidden">
          <DialogHeader>
            <DialogTitle>
              {dialogType === "edit" ? "编辑采购单" : "新建采购单"}
              {dialogType === "edit" && dialogPurchase && <span className="text-sm font-normal text-muted-foreground ml-2">· {dialogPurchase.order_no}</span>}
            </DialogTitle>
          </DialogHeader>
          <div className="flex flex-col gap-3 flex-1 min-h-0 overflow-y-auto">
            <div className="flex items-center gap-3 p-3 bg-muted/30 rounded-lg">
              <Label className="text-sm font-medium w-20">供应商：</Label>
              <select className="flex-1 max-w-xs h-9 rounded-lg border border-input bg-background px-3 text-sm"
                value={formSupplierId ?? ""} onChange={e => setFormSupplierId(e.target.value ? Number(e.target.value) : null)}>
                <option value="">请选择供应商</option>
                {suppliers.map(s => <option key={s.id} value={s.id}>{s.name}</option>)}
              </select>
            </div>

            <div className="relative">
              <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
              <input ref={searchRef} className="flex h-10 w-full rounded-lg border border-input bg-background pl-10 pr-4 text-sm"
                placeholder="搜索用品添加到清单..." value={searchQuery} onChange={e => setSearchQuery(e.target.value)}
                onFocus={() => setSearchOpen(true)} onBlur={() => setTimeout(() => setSearchOpen(false), 200)}
                onKeyDown={e => { if (e.key === "ArrowDown" && supplies.length > 0) { e.preventDefault(); addItem(supplies[0]); } }} />
              {searchOpen && searchQuery.trim() && (
                <div className="absolute left-0 right-0 top-full mt-1 z-50 bg-background border rounded-lg shadow-lg max-h-[260px] overflow-y-auto">
                  {supplies.length === 0 ? (
                    <div className="p-4 text-center text-sm text-muted-foreground">未找到匹配用品</div>
                  ) : supplies.map(sup => (
                    <div key={sup.id} className="flex items-center justify-between px-3 py-2 hover:bg-accent cursor-pointer border-b last:border-0"
                      onMouseDown={() => addItem(sup)}>
                      <div><span className="font-medium text-sm">{sup.name}</span>
                        <span className="text-xs text-muted-foreground ml-2">{sup.spec} · {sup.unit}</span></div>
                      <span className="text-sm font-semibold text-primary">¥{(sup.reference_price || 0).toFixed(2)}</span>
                    </div>
                  ))}
                </div>
              )}
            </div>

            <div ref={itemsScrollRef} className="overflow-auto border rounded-lg flex-1 min-h-0 max-h-[50vh]">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead className="w-8 text-center text-xs">序号</TableHead><TableHead className="text-xs">品名</TableHead>
                    <TableHead className="text-xs">规格</TableHead><TableHead className="w-14 text-center text-xs">单位</TableHead>
                    <TableHead className="w-20 text-center text-xs">参考价</TableHead><TableHead className="w-24 text-center text-xs">单价</TableHead>
                    <TableHead className="w-20 text-center text-xs">数量</TableHead><TableHead className="w-32 text-center text-xs">日期</TableHead>
                    <TableHead className="w-20 text-xs text-right">小计</TableHead><TableHead className="w-8" />
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {formItems.length === 0 ? (
                    <TableRow><TableCell colSpan={10} className="h-20 text-center text-sm text-muted-foreground">搜索添加用品</TableCell></TableRow>
                  ) : formItems.map((item, idx) => (
                    <TableRow key={idx}>
                      <TableCell className="text-center text-xs text-muted-foreground">{idx + 1}</TableCell>
                      <TableCell className="text-sm font-medium">{item.supply_name}</TableCell>
                      <TableCell className="text-xs text-muted-foreground">{item.supply_spec}</TableCell>
                      <TableCell className="text-center text-xs text-muted-foreground">{item.unit}</TableCell>
                      <TableCell className="text-center text-xs text-muted-foreground font-mono">¥{item.reference_price?.toFixed(2)}</TableCell>
                      <TableCell className="text-center">
                        <input type="text" inputMode="decimal" className={`h-7 w-20 text-center font-mono text-xs rounded border px-1 ${item.reference_price > 0 && item.unit_price > item.reference_price ? "border-amber-500 text-amber-700" : "border-input"}`}
                          value={item.unit_price_str} onChange={e => updatePrice(idx, e.target.value)} onBlur={() => blurPrice(idx)} />
                      </TableCell>
                      <TableCell className="text-center">
                        <input type="text" inputMode="numeric" className="h-7 w-20 text-center font-mono text-xs rounded border border-input px-1"
                          value={item.quantity} onChange={e => updateQty(idx, parseInt(e.target.value) || 1)} />
                      </TableCell>
                      <TableCell className="text-center">
                        <input type="date" className="h-7 w-32 rounded border border-input px-1 text-xs font-mono"
                          value={item.date} onChange={e => setFormItems(prev => prev.map((f, i) => i === idx ? { ...f, date: e.target.value } : f))} />
                      </TableCell>
                      <TableCell className="text-sm font-mono font-medium text-right">¥{item.subtotal.toFixed(2)}</TableCell>
                      <TableCell><Button variant="ghost" size="icon" className="h-6 w-6" onClick={() => removeItem(idx)}><X className="h-3 w-3 text-red-500" /></Button></TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </div>

            <div className="flex items-center gap-3">
              <Label className="text-sm font-medium w-20 shrink-0">备注：</Label>
              <Input placeholder="备注（可选）" value={formRemark} onChange={e => setFormRemark(e.target.value)} />
            </div>

            <div className="flex items-center justify-between pt-1">
              <label className="text-sm text-muted-foreground">
                采购日期：
                <input type="date" className="ml-1 h-8 w-[140px] rounded border border-input px-1 text-sm font-mono"
                  value={formDateFrom} onChange={e => setFormDateFrom(e.target.value)} />
                <span className="mx-1">~</span>
                <input type="date" className="h-8 w-[140px] rounded border border-input px-1 text-sm font-mono"
                  value={formDateTo} onChange={e => setFormDateTo(e.target.value)} />
              </label>
              <span className="text-xl font-bold text-red-600">{formatCurrency(totalAmount)}</span>
            </div>
          </div>
          <div className="flex justify-end gap-2 pt-2 border-t">
            <Button variant="outline" size="sm" onClick={() => setDialogType(null)}>取消</Button>
            <Button onClick={handleSave} disabled={saving} size="sm">
              {saving ? <Loader2 className="mr-1 h-3.5 w-3.5 animate-spin" /> : <Save className="mr-1 h-3.5 w-3.5" />}
              {saving ? "保存中..." : "保存"}
            </Button>
          </div>
        </DialogContent>
      </Dialog>

      {/* 查看弹窗 */}
      <Dialog open={dialogType === "view"} onOpenChange={v => { if (!v) setDialogType(null); }}>
        <DialogContent className="sm:max-w-[780px] max-h-[85vh] overflow-y-auto">
          <DialogHeader><DialogTitle>采购单详情 · {dialogPurchase?.order_no || ""}</DialogTitle></DialogHeader>
          {dialogPurchase && (
            <div className="space-y-3">
              <div className="flex flex-wrap gap-4 text-sm">
                <span><strong>单号：</strong>{dialogPurchase.order_no}</span>
                <span><strong>日期：</strong>{formatShortDate(dialogPurchase.purchase_date)}</span>
                <span><strong>供应商：</strong>{dialogPurchase.supplier_name || "-"}</span>
                <span><strong>状态：</strong>
                  <Badge variant={dialogPurchase.status === "completed" ? "default" : "secondary"}>
                    {dialogPurchase.status === "completed" ? "已完成" : "草稿"}
                  </Badge>
                </span>
              </div>
              <div className="overflow-x-auto border rounded-lg">
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead className="w-8 text-center text-xs">序号</TableHead><TableHead className="text-xs">品名</TableHead>
                      <TableHead className="text-xs">规格</TableHead><TableHead className="w-14 text-center text-xs">单位</TableHead>
                      <TableHead className="w-20 text-center text-xs">单价</TableHead><TableHead className="w-16 text-center text-xs">数量</TableHead>
                      <TableHead className="w-28 text-center text-xs">日期</TableHead><TableHead className="w-20 text-xs text-right">小计</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {dialogItems.map((item: any, idx: number) => (
                      <TableRow key={idx}>
                        <TableCell className="text-center text-xs">{idx + 1}</TableCell>
                        <TableCell className="text-sm font-medium">{item.supply_name}</TableCell>
                        <TableCell className="text-xs text-muted-foreground">{item.supply_spec || "-"}</TableCell>
                        <TableCell className="text-center text-xs">{item.unit || "-"}</TableCell>
                        <TableCell className="text-xs font-mono">¥{Number(item.unit_price).toFixed(2)}</TableCell>
                        <TableCell className="text-xs font-mono text-center">{item.quantity}</TableCell>
                        <TableCell className="text-center font-mono text-xs">{item.date ? formatShortDate(item.date) : "-"}</TableCell>
                        <TableCell className="text-xs font-mono font-medium text-right">¥{Number(item.subtotal).toFixed(2)}</TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </div>
              <div className="flex items-center justify-between pt-1">
                <span className="text-xl font-bold text-red-600">合计：{formatCurrency(dialogPurchase.total_amount)}</span>
                <Button size="sm" variant="outline" onClick={() => handlePrint(dialogPurchase)}><Printer className="mr-1.5 h-4 w-4" />打印预览</Button>
              </div>
            </div>
          )}
        </DialogContent>
      </Dialog>

      <Dialog open={confirmOpen} onOpenChange={setConfirmOpen}>
        <DialogContent className="sm:max-w-[400px]">
          <DialogHeader><DialogTitle>确认操作</DialogTitle></DialogHeader>
          <p className="text-sm text-muted-foreground py-2">{confirmMsg}</p>
          <div className="flex justify-end gap-2">
            <Button variant="outline" onClick={() => setConfirmOpen(false)}>取消</Button>
            <Button variant="destructive" onClick={() => confirmAction()}>确认</Button>
          </div>
        </DialogContent>
      </Dialog>
    </div>
  );
}
