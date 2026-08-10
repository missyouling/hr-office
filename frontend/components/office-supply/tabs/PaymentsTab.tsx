/* eslint-disable @typescript-eslint/no-explicit-any */
"use client";

import { useState, useEffect, useCallback, useRef } from "react";
import { toast } from "sonner";
import {
  Plus, Eye, Pencil, Trash2, Printer, Save, Loader2,
} from "lucide-react";

import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import {
  Dialog, DialogContent, DialogHeader, DialogTitle,
} from "@/components/ui/dialog";
import { Label } from "@/components/ui/label";

import { officeApi } from "@/lib/api-office";
import { amountToCn, formatShortDate, formatPurchaseDate, todayStr } from "../utils";

type DialogType = "new" | "view" | "edit" | null;
const PAYMENT_METHODS = ["现金", "现支", "转支", "电汇", "其它"];

/** 请款单 Tab：滚动加载列表 + 新建/编辑/查看 + 金额大写 + 打印预览 */
export default function PaymentsTab() {
  const [items, setItems] = useState<any[]>([]);
  const [loading, setLoading] = useState(false);
  const [listPage, setListPage] = useState(1);
  const [hasMore, setHasMore] = useState(true);
  const [loadingMore, setLoadingMore] = useState(false);
  const scrollRef = useRef<HTMLDivElement>(null);

  const [dialogType, setDialogType] = useState<DialogType>(null);
  const [editId, setEditId] = useState<number | null>(null);
  const [viewItem, setViewItem] = useState<any>(null);
  const [viewPurchases, setViewPurchases] = useState<any[]>([]);

  const [form, setForm] = useState({
    payment_unit: "", department: "", applicant: "", request_date: todayStr(),
    content: "", payee: "", payee_supplier_id: "", bank_name: "", bank_account: "",
    amount: 0, payment_method: "转支", remark: "",
    company_head: "", finance_head: "", dept_head: "", handler: "", purchase_ids: "",
  });
  const [saving, setSaving] = useState(false);

  const [unpaidPurchases, setUnpaidPurchases] = useState<any[]>([]);
  const [suppliers, setSuppliers] = useState<any[]>([]);

  const [confirmOpen, setConfirmOpen] = useState(false);
  const [deleteTarget, setDeleteTarget] = useState<number | null>(null);
  const [confirmMsg, setConfirmMsg] = useState("");

  const load = useCallback(async (reset = true) => {
    setLoading(true);
    try {
      const p = reset ? 1 : listPage;
      const r = await officeApi.paymentRequests.list({ limit: "50", page: String(p) });
      const data = r.data || [];
      setItems(reset ? data : prev => [...prev, ...data]);
      setHasMore(data.length >= 50);
      if (reset) { setListPage(2); if (scrollRef.current) scrollRef.current.scrollTop = 0; }
      else setListPage(p + 1);
    } catch (e: any) { toast.error("加载失败", { description: e.message }); }
    finally { setLoading(false); }
  }, [listPage]);

  // eslint-disable-next-line react-hooks/exhaustive-deps
  useEffect(() => { load(true); }, []);

  const loadMore = async () => {
    if (loadingMore || loading || !hasMore) return;
    setLoadingMore(true);
    try {
      const r = await officeApi.paymentRequests.list({ limit: "50", page: String(listPage) });
      setItems(prev => [...prev, ...(r.data || [])]);
      setHasMore((r.data || []).length >= 50);
      setListPage(listPage + 1);
    } catch (e: any) { toast.error("加载失败", { description: e.message }); }
    finally { setLoadingMore(false); }
  };

  useEffect(() => {
    officeApi.purchases.unpaid().then(r => setUnpaidPurchases(r.data || [])).catch(() => {});
    officeApi.suppliers.list().then(r => setSuppliers(r.data || [])).catch(() => {});
  }, []);
  useEffect(() => { if (dialogType === "new" || dialogType === "edit") { officeApi.purchases.unpaid().then(r => setUnpaidPurchases(r.data || [])).catch(() => {}); } }, [dialogType]);

  const togglePurchase = (id: number) => {
    setForm(f => {
      const ids = f.purchase_ids ? f.purchase_ids.split(",").map(s => s.trim()).filter(Boolean) : [];
      const has = ids.includes(String(id));
      const newIds = has ? ids.filter(x => x !== String(id)) : [...ids, String(id)];
      const total = unpaidPurchases.filter(p => newIds.includes(String(p.id))).reduce((s, p) => s + (Number(p.total_amount) || 0), 0);
      return { ...f, purchase_ids: newIds.join(","), amount: Math.round(total * 100) / 100 };
    });
  };

  const handlePayeeChange = (value: string) => {
    setForm(f => ({ ...f, payee: value, payee_supplier_id: "" }));
    const sup = suppliers.find(s => s.name === value);
    if (sup) setForm(f => ({ ...f, payee: sup.name, payee_supplier_id: String(sup.id), bank_name: sup.bank_name || "", bank_account: sup.bank_account || "" }));
  };

  const amountCn = amountToCn(form.amount);

  const openNew = () => {
    setEditId(null);
    setForm({
      payment_unit: "", department: "", applicant: "", request_date: todayStr(),
      content: "", payee: "", payee_supplier_id: "", bank_name: "", bank_account: "",
      amount: 0, payment_method: "转支", remark: "",
      company_head: "", finance_head: "", dept_head: "", handler: "", purchase_ids: "",
    });
    setDialogType("new");
  };

  const openEdit = async (id: number) => {
    try {
      const r = await officeApi.paymentRequests.get(id);
      const d = r.data as any;
      setEditId(id);
      setForm({
        payment_unit: d.payment_unit || "", department: d.department || "", applicant: d.applicant || "",
        request_date: d.request_date || todayStr(), content: d.content || "",
        payee: d.payee || "", payee_supplier_id: d.payee_supplier_id ? String(d.payee_supplier_id) : "",
        bank_name: d.bank_name || "", bank_account: d.bank_account || "", amount: d.amount || 0,
        payment_method: d.payment_method || "转支", remark: d.remark || "",
        company_head: d.company_head || "", finance_head: d.finance_head || "",
        dept_head: d.dept_head || "", handler: d.handler || "", purchase_ids: d.purchase_ids || "",
      });
      setDialogType("edit");
    } catch (e: any) { toast.error("加载失败", { description: e.message }); }
  };

  const openView = async (id: number) => {
    try {
      const r = await officeApi.paymentRequests.get(id);
      const d = r.data as any;
      setViewItem(d);
      setViewPurchases(await fetchLinkedPurchases(d));
      setDialogType("view");
    } catch (e: any) { toast.error("加载失败", { description: e.message }); }
  };

  const fetchLinkedPurchases = async (d: any): Promise<any[]> => {
    const ids = (d.purchase_ids || "").split(",").map((s: string) => s.trim()).filter(Boolean);
    if (!ids.length) return [];
    const results: any[] = [];
    for (const id of ids) {
      try { const r = await officeApi.purchases.get(Number(id)); if (r.data) results.push(r.data); }
      catch { /* 忽略 */ }
    }
    return results;
  };

  const handleSave = async () => {
    if (!form.request_date) { toast.error("请选择申请日期"); return; }
    if (form.amount <= 0) { toast.error("金额必须大于0"); return; }
    setSaving(true);
    try {
      const body = { ...form, payee_supplier_id: form.payee_supplier_id ? Number(form.payee_supplier_id) : null, amount_cn: amountCn };
      if (editId) { await officeApi.paymentRequests.update(editId, body); toast.success("请款单已更新"); }
      else { await officeApi.paymentRequests.create(body); toast.success("请款单已保存"); }
      setDialogType(null); load();
    } catch (e: any) { toast.error("保存失败", { description: e.message }); }
    finally { setSaving(false); }
  };

  const handleDelete = (id: number, no: string) => { setDeleteTarget(id); setConfirmMsg(`确认删除请款单「${no}」？`); setConfirmOpen(true); };
  const confirmDelete = async () => {
    if (!deleteTarget) return;
    try { await officeApi.paymentRequests.remove(deleteTarget); toast.success("已删除"); load(); }
    catch (e: any) { toast.error("删除失败", { description: e.message }); }
    finally { setConfirmOpen(false); setDeleteTarget(null); }
  };

  const handlePrint = async (d: any) => {
    const purchases = await fetchLinkedPurchases(d);
    const cn = d.amount_cn || amountToCn(d.amount);
    let itemLines = ""; let rowNo = 0; let grandTotal = 0;
    purchases.forEach(p => { (p.items || []).forEach((it: any) => { rowNo++; grandTotal += Number(it.subtotal) || 0; itemLines += `<div class="line">${rowNo}. ${it.supply_name || ""} ${it.supply_spec || ""} ${it.unit || ""} ×${it.quantity}　¥${Number(it.subtotal).toFixed(2)}</div>`; }); });
    const attach = purchases.length ? `<div class="attach"><h2>附件：采购单明细</h2>${itemLines}<div class="attach-total">合计：¥${grandTotal.toFixed(2)}</div></div>` : "";
    const html = `<!doctype html><html><head><meta charset="utf-8"><title>请款单 ${d.request_no || ""}</title>
<style>*{margin:0;padding:0}body{font-family:"Microsoft YaHei",sans-serif;padding:50px 60px;color:#333;font-size:14px;width:210mm;margin:0 auto}
h1{font-size:28px;text-align:center;letter-spacing:12px;margin-bottom:24px}table{width:100%;border-collapse:collapse;margin-bottom:20px}
th,td{border:1px solid #333;padding:8px 10px;font-size:13px}th{background:#f0f4ff;font-weight:600;text-align:center;width:110px;color:#1e40af}
.cn{font-size:15px;letter-spacing:2px;color:#1e40af;font-weight:600}.amt{font-family:'Courier New',monospace;font-weight:bold;font-size:16px;color:#dc2626}
.attach{margin-top:30px;padding-top:14px}.attach h2{font-size:16px;color:#1e40af;margin-bottom:8px}
.line{font-size:13px;line-height:1.9}.attach-total{margin-top:6px;font-size:14px;font-weight:bold;color:#dc2626;border-top:1px dashed #999;padding-top:6px}
@media print{body{padding:30px 40px}th{background:#f0f4ff!important;-webkit-print-color-adjust:exact;print-color-adjust:exact}}</style></head><body>
<h1>请 款 单</h1>
<div style="display:flex;justify-content:space-between;font-size:13px;color:#555;margin-bottom:16px"><span><strong>单号：</strong>${d.request_no || ""}</span><span><strong>日期：</strong>${d.request_date || ""}</span></div>
<table><tbody>
<tr><th>付款单位</th><td>${d.payment_unit || ""}</td><th>使用部门</th><td>${d.department || ""}</td></tr>
<tr><th>申请人</th><td>${d.applicant || ""}</td><th>申请日期</th><td>${d.request_date || ""}</td></tr>
<tr><th>请款内容</th><td colspan="3">${d.content || ""}</td></tr>
<tr><th>收款人</th><td>${d.payee || ""}</td><th>支付方式</th><td>${d.payment_method || ""}</td></tr>
<tr><th>开户行</th><td>${d.bank_name || ""}</td><th>银行账号</th><td style="font-family:monospace">${d.bank_account || ""}</td></tr>
<tr><th>金额（小写）</th><td class="amt" colspan="3">¥${Number(d.amount).toFixed(2)}</td></tr>
<tr><th>金额（大写）</th><td class="cn" colspan="3">${cn}</td></tr>
<tr><th>备注</th><td colspan="3">${d.remark || ""}</td></tr>
<tr><th>公司负责人</th><td>${d.company_head || ""}</td><th>财务负责人</th><td>${d.finance_head || ""}</td></tr>
<tr><th>部门负责人</th><td>${d.dept_head || ""}</td><th>经办人</th><td>${d.handler || ""}</td></tr></tbody></table>
${attach}
<script>setTimeout(()=>window.print(),300)</script></body></html>`;
    const w = window.open("", "_blank");
    if (w) { w.document.write(html); w.document.close(); }
  };

  const isFormOpen = dialogType === "new" || dialogType === "edit";

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <div />
        <Button onClick={openNew} size="sm"><Plus className="mr-1 h-4 w-4" />新建请款单</Button>
      </div>

      <Card>
        <CardContent className="p-0">
          <div ref={scrollRef} className="relative overflow-y-auto max-h-[calc(100vh-240px)] rounded-md border"
            onScroll={e => { const el = e.currentTarget; if (el.scrollTop + el.clientHeight >= el.scrollHeight - 40) loadMore(); }}>
            <table className="w-full text-sm">
              <thead className="sticky top-0 z-10 bg-muted">
                <tr className="border-b">
                  <th className="px-2 py-2 text-center font-medium text-xs w-10">序号</th>
                  <th className="px-2 py-2 text-center font-medium text-xs">单号</th>
                  <th className="px-2 py-2 text-center font-medium text-xs">申请日期</th>
                  <th className="px-2 py-2 text-center font-medium text-xs">请款内容</th>
                  <th className="px-2 py-2 text-center font-medium text-xs">收款人</th>
                  <th className="px-2 py-2 text-center font-medium text-xs">金额</th>
                  <th className="px-2 py-2 text-center font-medium text-xs">状态</th>
                  <th className="px-2 py-2 text-center font-medium text-xs w-[120px]">操作</th>
                </tr>
              </thead>
              <tbody>
                {loading && items.length === 0 ? (
                  <tr><td colSpan={8} className="h-24 text-center text-muted-foreground">加载中...</td></tr>
                ) : items.length === 0 ? (
                  <tr><td colSpan={8} className="h-24 text-center text-muted-foreground">暂无请款单</td></tr>
                ) : items.map((p, idx) => (
                  <tr key={p.id} className="border-b hover:bg-muted/50">
                    <td className="px-2 py-1.5 text-center text-xs text-muted-foreground">{idx + 1}</td>
                    <td className="px-2 py-1.5 text-center font-mono text-xs font-medium">{p.request_no}</td>
                    <td className="px-2 py-1.5 text-center text-xs">{formatShortDate(p.request_date)}</td>
                    <td className="px-2 py-1.5 text-center text-xs truncate max-w-[200px]">{p.content || "-"}</td>
                    <td className="px-2 py-1.5 text-center text-xs">{p.payee || "-"}</td>
                    <td className="px-2 py-1.5 text-center font-mono text-sm font-bold">¥{Number(p.amount).toFixed(2)}</td>
                    <td className="px-2 py-1.5 text-center">
                      <span className={`inline-block px-2 py-0.5 rounded-full text-xs font-medium ${p.status === "submitted" ? "bg-green-100 text-green-700" : "bg-yellow-100 text-yellow-700"}`}>
                        {p.status === "submitted" ? "已提交" : "草稿"}
                      </span>
                    </td>
                    <td className="px-2 py-1.5 text-center">
                      <Button variant="ghost" size="icon" className="h-7 w-7" onClick={() => openView(p.id)}><Eye className="h-3.5 w-3.5" /></Button>
                      <Button variant="ghost" size="icon" className="h-7 w-7" onClick={() => openEdit(p.id)}><Pencil className="h-3.5 w-3.5" /></Button>
                      <Button variant="ghost" size="icon" className="h-7 w-7" onClick={() => handleDelete(p.id, p.request_no)}><Trash2 className="h-3.5 w-3.5 text-red-500" /></Button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
            {loadingMore && <div className="py-2 text-center text-xs text-muted-foreground">加载中…</div>}
            {!loadingMore && !hasMore && items.length > 0 && <div className="py-2 text-center text-xs text-muted-foreground">已加载全部 {items.length} 条</div>}
          </div>
        </CardContent>
      </Card>

      {/* 新建/编辑弹窗 */}
      <Dialog open={isFormOpen} onOpenChange={v => { if (!v) setDialogType(null); }}>
        <DialogContent className="sm:max-w-[700px] max-h-[90vh] overflow-y-auto">
          <DialogHeader><DialogTitle>{editId ? "编辑请款单" : "新建请款单"}</DialogTitle></DialogHeader>
          <div className="space-y-3 py-2">
            <div className="grid grid-cols-2 gap-3">
              <div className="space-y-1.5"><Label>付款单位</Label><Input value={form.payment_unit} onChange={e => setForm(f => ({ ...f, payment_unit: e.target.value }))} placeholder="付款单位名称" /></div>
              <div className="space-y-1.5"><Label>使用部门</Label><Input value={form.department} onChange={e => setForm(f => ({ ...f, department: e.target.value }))} placeholder="部门名称" /></div>
            </div>
            <div className="grid grid-cols-2 gap-3">
              <div className="space-y-1.5"><Label>申请人</Label><Input value={form.applicant} onChange={e => setForm(f => ({ ...f, applicant: e.target.value }))} /></div>
              <div className="space-y-1.5"><Label>申请日期</Label><Input type="date" value={form.request_date} onChange={e => setForm(f => ({ ...f, request_date: e.target.value }))} /></div>
            </div>
            <div className="space-y-1.5"><Label>请款内容</Label><Input value={form.content} onChange={e => setForm(f => ({ ...f, content: e.target.value }))} placeholder="请款事由及内容" /></div>
            {unpaidPurchases.length > 0 && (
              <div className="border rounded-lg p-3">
                <Label className="mb-1 block">关联采购单（未付款，勾选自动汇总金额）</Label>
                <div className="max-h-[140px] overflow-y-auto space-y-1">
                  {unpaidPurchases.map(p => {
                    const checked = form.purchase_ids.split(",").includes(String(p.id));
                    return (
                      <label key={p.id} className={`flex items-center gap-2 px-2 py-1.5 rounded cursor-pointer text-sm border ${checked ? "bg-blue-50 border-blue-300" : "hover:bg-muted/50"}`}>
                        <input type="checkbox" checked={checked} onChange={() => togglePurchase(p.id)} />
                        <span className="font-mono text-xs flex-1 truncate">{p.order_no}</span>
                        <span className="text-xs text-muted-foreground flex-1 truncate">{p.supplier_name || ""} · {formatPurchaseDate(p.purchase_date)}</span>
                        <span className="font-mono text-xs font-bold">¥{Number(p.total_amount).toFixed(2)}</span>
                      </label>
                    );
                  })}
                </div>
              </div>
            )}
            <div className="space-y-1.5"><Label>收款人</Label>
              <Input value={form.payee} onChange={e => handlePayeeChange(e.target.value)} placeholder="输入或从供应商列表选择" list="payee-list" />
              <datalist id="payee-list">{suppliers.map(s => <option key={s.id} value={s.name} />)}</datalist>
            </div>
            <div className="grid grid-cols-2 gap-3">
              <div className="space-y-1.5"><Label>开户行</Label><Input value={form.bank_name} onChange={e => setForm(f => ({ ...f, bank_name: e.target.value }))} /></div>
              <div className="space-y-1.5"><Label>银行账号</Label><Input value={form.bank_account} onChange={e => setForm(f => ({ ...f, bank_account: e.target.value }))} /></div>
            </div>
            <div className="grid grid-cols-3 gap-3">
              <div className="space-y-1.5"><Label>金额（元）</Label><Input type="number" step="0.01" min="0" value={form.amount || ""} onChange={e => setForm(f => ({ ...f, amount: parseFloat(e.target.value) || 0 }))} /></div>
              <div className="col-span-2 space-y-1.5"><Label>金额大写</Label><div className="flex h-10 w-full items-center rounded-lg border border-blue-200 bg-blue-50 px-3 text-sm font-bold text-blue-700">{amountCn}</div></div>
            </div>
            <div className="space-y-1.5"><Label>支付方式</Label>
              <div className="flex gap-3 mt-1">
                {PAYMENT_METHODS.map(m => (
                  <label key={m} className="flex items-center gap-1.5 cursor-pointer">
                    <input type="radio" name="pm" checked={form.payment_method === m} onChange={() => setForm(f => ({ ...f, payment_method: m }))} />
                    <span className="text-sm">{m}</span>
                  </label>
                ))}
              </div>
            </div>
            <div className="space-y-1.5"><Label>备注</Label><textarea className="flex h-16 w-full rounded-md border border-input bg-transparent px-3 py-2 text-sm" value={form.remark} onChange={e => setForm(f => ({ ...f, remark: e.target.value }))} /></div>
            <div className="grid grid-cols-2 gap-3">
              <div className="space-y-1.5"><Label>部门负责人</Label><Input value={form.dept_head} onChange={e => setForm(f => ({ ...f, dept_head: e.target.value }))} /></div>
              <div className="space-y-1.5"><Label>经办人</Label><Input value={form.handler} onChange={e => setForm(f => ({ ...f, handler: e.target.value }))} /></div>
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
      {viewItem && (
        <Dialog open={dialogType === "view"} onOpenChange={v => { if (!v) { setDialogType(null); setViewItem(null); setViewPurchases([]); } }}>
          <DialogContent className="sm:max-w-[800px] max-h-[90vh] overflow-y-auto">
            <DialogHeader><DialogTitle>请款单详情 · {viewItem.request_no}</DialogTitle></DialogHeader>
            <div className="border rounded-lg p-4 bg-background">
              <h2 style={{ textAlign: "center", fontSize: "22px", letterSpacing: "10px", marginBottom: "16px" }}>请 款 单</h2>
              <div style={{ display: "flex", justifyContent: "space-between", fontSize: "12px", color: "#555", marginBottom: "12px" }}>
                <span><strong>单号：</strong>{viewItem.request_no}</span>
                <span><strong>日期：</strong>{viewItem.request_date}</span>
              </div>
              <table className="w-full border-collapse mb-3 text-sm">
                <tbody>
                  {[
                    ["付款单位", viewItem.payment_unit || "", "使用部门", viewItem.department || ""],
                    ["申请人", viewItem.applicant || "", "申请日期", viewItem.request_date || ""],
                    ["请款内容", viewItem.content || "", "", ""],
                    ["收款人", viewItem.payee || "", "支付方式", viewItem.payment_method || ""],
                    ["开户行", viewItem.bank_name || "", "银行账号", viewItem.bank_account || ""],
                    ["金额(小写)", `¥${Number(viewItem.amount).toFixed(2)}`, "金额(大写)", viewItem.amount_cn || amountToCn(viewItem.amount)],
                    ["备注", viewItem.remark || "", "", ""],
                    ["公司负责人", viewItem.company_head || "", "财务负责人", viewItem.finance_head || ""],
                    ["部门负责人", viewItem.dept_head || "", "经办人", viewItem.handler || ""],
                  ].map(([l1, v1, l2, v2], i) => (
                    <tr key={i}>
                      <th className="border border-gray-400 p-1.5 px-2 text-center bg-muted w-24 text-xs">{l1}</th>
                      <td colSpan={!l2 ? 3 : 1} className={`border border-gray-400 p-1.5 px-2 ${l1 === "金额(小写)" ? "font-bold text-red-600 font-mono" : ""}`}>
                        {v1}
                      </td>
                      {l2 && <th className="border border-gray-400 p-1.5 px-2 text-center bg-muted w-24 text-xs">{l2}</th>}
                      {l2 && <td className={`border border-gray-400 p-1.5 px-2 ${l2 === "金额(大写)" ? "text-blue-700 font-bold tracking-wider" : ""}`}>{v2}</td>}
                    </tr>
                  ))}
                </tbody>
              </table>
              {viewPurchases.length > 0 && (
                <div className="mt-3 pt-2">
                  <div className="font-bold text-sm text-primary mb-1">附件：采购单明细</div>
                  {(() => {
                    let no = 0, total = 0;
                    const lines = viewPurchases.flatMap(p => (p.items || []).map((it: any) => { no++; total += Number(it.subtotal) || 0; return <div key={`${p.id}-${it.supply_id}`} className="text-[13px] leading-7">{no}. {it.supply_name} {it.supply_spec || ""} {it.unit} ×{it.quantity}　¥{Number(it.subtotal).toFixed(2)}</div>; }));
                    return <>{lines}<div className="mt-1 text-sm font-bold text-red-600 border-t border-dashed pt-1">合计：¥{total.toFixed(2)}</div></>;
                  })()}
                </div>
              )}
            </div>
            <div className="flex justify-end gap-2 pt-1">
              <Button variant="outline" size="sm" onClick={() => setDialogType(null)}>关闭</Button>
              <Button size="sm" onClick={() => handlePrint(viewItem)}><Printer className="mr-1.5 h-4 w-4" />打印</Button>
            </div>
          </DialogContent>
        </Dialog>
      )}

      <Dialog open={confirmOpen} onOpenChange={setConfirmOpen}>
        <DialogContent className="sm:max-w-[400px]">
          <DialogHeader><DialogTitle>确认操作</DialogTitle></DialogHeader>
          <p className="text-sm text-muted-foreground py-2">{confirmMsg}</p>
          <div className="flex justify-end gap-2">
            <Button variant="outline" onClick={() => setConfirmOpen(false)}>取消</Button>
            <Button variant="destructive" onClick={confirmDelete}>确认删除</Button>
          </div>
        </DialogContent>
      </Dialog>
    </div>
  );
}
