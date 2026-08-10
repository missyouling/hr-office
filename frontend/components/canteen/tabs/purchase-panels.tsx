"use client";

import { useState, useEffect, useCallback } from "react";
import { Card, CardContent } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogClose } from "@/components/ui/dialog";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { toast } from "sonner";
import { canteenApi } from "@/lib/api-canteen";
import type { CanteenPurchase, CanteenPurchaseItem, CanteenSupply } from "@/lib/api-canteen";
import { fmt } from "@/components/canteen/utils";
import { Plus, Pencil, Trash2, X, Save } from "lucide-react";

const CHANNELS = ["电商平台", "个体经营", "自购"];

// ---------- 食材采购面板 ----------
export function PurchasePanel() {
  const [list, setList] = useState<CanteenPurchase[]>([]);
  const [supplies, setSupplies] = useState<CanteenSupply[]>([]);
  const [month, setMonth] = useState(new Date().toISOString().slice(0, 7));
  const [open, setOpen] = useState(false);
  const [editId, setEditId] = useState<number | null>(null);
  const [form, setForm] = useState({
    purchase_date: new Date().toISOString().slice(0, 10), supplier_name: "", channel: CHANNELS[0], notes: "",
    items: [] as (CanteenPurchaseItem & { key?: string })[],
  });
  const [rowKeyword, setRowKeyword] = useState("");
  const [rowOpen, setRowOpen] = useState(false);
  const [confirmId, setConfirmId] = useState<number | null>(null);

  const load = useCallback(async () => {
    try { setList((await canteenApi.purchases.list({ month })).data); } catch (e: unknown) { toast.error("加载失败", { description: (e as Error).message }); }
  }, [month]);
  useEffect(() => { load(); }, [load]);

  const loadSupplies = async (kw = "") => {
    try { setSupplies((await canteenApi.supplies.list({ keyword: kw, limit: "30" })).data); } catch { /* ignore */ }
  };

  const openNew = () => {
    setEditId(null);
    setForm({ purchase_date: new Date().toISOString().slice(0, 10), supplier_name: "", channel: CHANNELS[0], notes: "", items: [] });
    setOpen(true); loadSupplies();
  };
  const openEdit = async (p: CanteenPurchase) => {
    setEditId(p.id);
    try {
      const d = (await canteenApi.purchases.get(p.id)).data;
      const raw = d as unknown as Record<string, unknown>;
      setForm({
        purchase_date: (raw.purchase_date as string) || new Date().toISOString().slice(0, 10),
        supplier_name: (raw.supplier_name as string) || "",
        channel: (raw.channel as string) || CHANNELS[0],
        notes: (raw.notes as string) || "",
        items: ((raw.items || []) as (CanteenPurchaseItem & { subtotal?: number })[]).map((it) => ({
          supply_id: it.supply_id, supply_name: it.supply_name, unit: it.unit,
          quantity: it.quantity, unit_price: it.unit_price, total_price: it.subtotal ?? it.total_price,
          key: `${it.supply_id}-${Math.random()}`,
        })),
      });
      setOpen(true); loadSupplies();
    } catch (e: unknown) { toast.error("加载失败", { description: (e as Error).message }); }
  };

  const totalAmount = form.items.reduce((s, i) => s + (Number((i as unknown as Record<string, unknown>).subtotal ?? i.total_price) || Number(i.quantity) * Number(i.unit_price) || 0), 0);

  const addRow = (s: CanteenSupply) => {
    const price = s.unit_price || 0;
    setForm((f) => ({ ...f, items: [...f.items, { supply_id: s.id, supply_name: s.name, unit: s.unit, quantity: 1, unit_price: price, total_price: price, key: `${s.id}-${Date.now()}` }] }));
    setRowOpen(false); setRowKeyword("");
  };
  const removeRow = (idx: number) => setForm((f) => ({ ...f, items: f.items.filter((_, i) => i !== idx) }));

  const save = async () => {
    if (!form.items.length) { toast.error("校验失败", { description: "请至少添加一行采购明细" }); return; }
    try {
      // eslint-disable-next-line @typescript-eslint/no-unused-vars
      const payload = { ...form, total_amount: totalAmount, items: form.items.map(({ key, ...rest }) => rest) } as unknown as Record<string, unknown>;
      if (editId) { await canteenApi.purchases.update(editId, payload); toast.success("已更新"); }
      else { await canteenApi.purchases.create(payload); toast.success("已保存"); }
      setOpen(false); load();
    } catch (e: unknown) { toast.error("保存失败", { description: (e as Error).message }); }
  };
  const del = async () => {
    if (!confirmId) return;
    try { await canteenApi.purchases.remove(confirmId); toast.success("已删除"); load(); }
    catch (e: unknown) { toast.error("删除失败", { description: (e as Error).message }); }
    finally { setConfirmId(null); }
  };

  const filteredSupplies = supplies.filter((s) => !form.items.some((i) => i.supply_id === s.id));

  return (
    <Card>
      <CardContent className="p-4 space-y-3">
        <div className="flex items-center justify-between">
          <h3 className="text-sm font-semibold">食材采购</h3>
          <div className="flex gap-2 items-center">
            <Input type="month" className="h-8 w-36" value={month} onChange={(e) => setMonth(e.target.value)} />
            <Button size="sm" onClick={openNew}><Plus className="mr-1 h-4 w-4" />新建</Button>
          </div>
        </div>
        <div className="relative overflow-y-auto max-h-[50vh] rounded-md border">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead className="w-12 text-center">序号</TableHead><TableHead className="text-center">日期</TableHead>
                <TableHead className="text-center">供应商</TableHead><TableHead className="text-center">明细</TableHead>
                <TableHead className="text-center">总金额</TableHead><TableHead className="w-[100px] text-center">操作</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {list.length === 0 ? (
                <TableRow><TableCell colSpan={6} className="h-16 text-center text-muted-foreground">暂无采购单</TableCell></TableRow>
              ) : (<>
                {list.map((p) => (
                  <TableRow key={p.id} className="hover:bg-muted/50">
                    <TableCell className="text-center text-muted-foreground">{p.id}</TableCell>
                    <TableCell className="text-center">{p.purchase_date}</TableCell>
                    <TableCell className="text-center">{(p as unknown as Record<string, unknown>).supplier_name as string || "-"}</TableCell>
                    <TableCell className="text-center">{p.items?.length || 0} 项</TableCell>
                    <TableCell className="text-center font-medium">{fmt(p.total_amount)}</TableCell>
                    <TableCell className="text-center">
                      <Button variant="ghost" size="icon" onClick={() => openEdit(p)}><Pencil className="h-4 w-4" /></Button>
                      <Button variant="ghost" size="icon" onClick={() => setConfirmId(p.id)}><Trash2 className="h-4 w-4 text-red-500" /></Button>
                    </TableCell>
                  </TableRow>
                ))}
                <TableRow className="bg-muted/50 font-semibold">
                  <TableCell colSpan={4} className="text-center">合计</TableCell>
                  <TableCell className="text-center">{fmt(list.reduce((s, p) => s + Number(p.total_amount || 0), 0))}</TableCell>
                  <TableCell />
                </TableRow>
              </>)}
            </TableBody>
          </Table>
        </div>
      </CardContent>

      {/* 采购单编辑弹窗 */}
      <Dialog open={open} onOpenChange={setOpen}>
        <DialogContent className="sm:max-w-[800px] max-h-[85vh] flex flex-col">
          <DialogHeader><DialogTitle>{editId ? `编辑采购单 #${editId}` : "新建采购单"}</DialogTitle></DialogHeader>
          <div className="grid grid-cols-2 md:grid-cols-4 gap-2">
            <Input type="date" value={form.purchase_date} onChange={(e) => setForm({ ...form, purchase_date: e.target.value })} />
            <Input placeholder="供应商名称" value={form.supplier_name} onChange={(e) => setForm({ ...form, supplier_name: e.target.value })} />
            <select className="h-9 rounded-md border px-2 text-sm bg-background" value={form.channel} onChange={(e) => setForm({ ...form, channel: e.target.value })}>
              {CHANNELS.map((c) => <option key={c} value={c}>{c}</option>)}
            </select>
            <Input placeholder="备注" value={form.notes} onChange={(e) => setForm({ ...form, notes: e.target.value })} />
          </div>
          <div className="relative">
            <Input placeholder="搜索食材添加…" value={rowKeyword}
              onFocus={() => { setRowOpen(true); loadSupplies(rowKeyword); }}
              onChange={(e) => { setRowKeyword(e.target.value); setRowOpen(true); loadSupplies(e.target.value); }} />
            {rowOpen && (
              <div className="absolute z-20 left-0 right-0 top-full mt-1 max-h-40 overflow-y-auto border rounded-md bg-background shadow-lg p-1">
                {filteredSupplies.length === 0 ? <p className="text-xs text-muted-foreground p-2">无匹配食材</p> : filteredSupplies.map((s) => (
                  <div key={s.id} className="flex items-center justify-between px-2 py-1.5 hover:bg-muted rounded cursor-pointer" onClick={() => addRow(s)}>
                    <span className="text-sm">{s.name} <span className="text-muted-foreground text-xs">{s.unit}</span></span>
                    <span className="text-xs text-primary">{fmt(s.unit_price)}</span>
                  </div>
                ))}
              </div>
            )}
          </div>
          <div className="flex-1 min-h-0 overflow-y-auto">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead className="w-12 text-center">序号</TableHead><TableHead className="text-center">品名</TableHead>
                  <TableHead className="w-20 text-center">单位</TableHead><TableHead className="w-24 text-center">数量</TableHead>
                  <TableHead className="w-24 text-center">单价</TableHead><TableHead className="w-24 text-center">小计</TableHead>
                  <TableHead className="w-14" />
                </TableRow>
              </TableHeader>
              <TableBody>
                {form.items.length === 0 ? (
                  <TableRow><TableCell colSpan={7} className="h-16 text-center text-muted-foreground">搜索上方添加食材</TableCell></TableRow>
                ) : form.items.map((it, idx) => (
                  <TableRow key={it.key || idx}>
                    <TableCell className="text-center text-muted-foreground">{idx + 1}</TableCell>
                    <TableCell className="text-center">{it.supply_name}</TableCell>
                    <TableCell className="text-center">{it.unit}</TableCell>
                    <TableCell className="text-center"><Input className="h-7 w-20 text-center" type="number" value={it.quantity} onChange={(e) => {
                      const q = parseFloat(e.target.value) || 0; const items = [...form.items]; items[idx] = { ...items[idx], quantity: q, total_price: q * Number(items[idx].unit_price) }; setForm({ ...form, items });
                    }} /></TableCell>
                    <TableCell className="text-center"><Input className="h-7 w-24 text-center" type="number" value={it.unit_price} onChange={(e) => {
                      const p = parseFloat(e.target.value) || 0; const items = [...form.items]; items[idx] = { ...items[idx], unit_price: p, total_price: Number(items[idx].quantity) * p }; setForm({ ...form, items });
                    }} /></TableCell>
                    <TableCell className="font-medium text-center">{fmt(it.total_price)}</TableCell>
                    <TableCell><Button variant="ghost" size="icon" onClick={() => removeRow(idx)}><X className="h-4 w-4 text-red-500" /></Button></TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </div>
          <div className="flex items-center justify-between pt-2 border-t">
            <div className="text-sm">合计：<span className="font-bold text-red-600 text-base">{fmt(totalAmount)}</span></div>
            <div className="flex gap-2">
              <DialogClose asChild><Button variant="outline">取消</Button></DialogClose>
              <Button onClick={save}><Save className="mr-1 h-4 w-4" />保存</Button>
            </div>
          </div>
        </DialogContent>
      </Dialog>

      <Dialog open={confirmId !== null} onOpenChange={() => setConfirmId(null)}>
        <DialogContent className="sm:max-w-[400px]">
          <DialogHeader><DialogTitle>确认操作</DialogTitle></DialogHeader>
          <p className="text-sm text-muted-foreground py-2">删除此采购单？明细将一并删除。</p>
          <div className="flex justify-end gap-2">
            <DialogClose asChild><Button variant="outline">取消</Button></DialogClose>
            <Button variant="destructive" onClick={del}>确认删除</Button>
          </div>
        </DialogContent>
      </Dialog>
    </Card>
  );
}

// ---------- 其他费用面板 ----------
export function ExpensePanel() {
  const [month, setMonth] = useState(new Date().toISOString().slice(0, 7));
  const [people, setPeople] = useState(0);
  const [params, setParams] = useState({ water_per_capita: 25, water_price: 5.22, elec_usage: 50, gas_usage: 10, gas_price: 3.55 });
  const [actual, setActual] = useState<Record<string, number>>({ water: 0, elec: 0, gas: 0, labor: 0, maintenance: 0 });
  const [saving, setSaving] = useState(false);

  const [y, mo] = month.split("-").map(Number);
  const dim = new Date(y, mo, 0).getDate();
  const now = new Date();
  const isCur = now.getFullYear() === y && now.getMonth() + 1 === mo;
  const days = isCur ? Math.min(now.getDate(), dim) : dim;

  const waterEst = people * (params.water_per_capita / 1000) * params.water_price;
  const elecEst = params.elec_usage * days;
  const gasEst = params.gas_usage * days * params.gas_price;
  const A = (key: string, est: number) => (Number(actual[key]) > 0 ? Number(actual[key]) : est);
  const totalEst = waterEst + elecEst + gasEst;
  const totalAct = A("water", waterEst) + A("elec", elecEst) + A("gas", gasEst) + A("labor", 0) + A("maintenance", 0);

  const load = useCallback(async () => {
    try {
      const inc = await canteenApi.income.list({ month });
      const p = (inc.data || []).reduce((acc: number, d) => acc + (Number((d as unknown as Record<string, unknown>).lunch_count) || 0) + (Number((d as unknown as Record<string, unknown>).dinner_count) || 0), 0);
      setPeople(p);
    } catch { /* ignore */ }
  }, [month]);
  useEffect(() => { load(); }, [load]);

  const save = async () => {
    setSaving(true);
    try {
      const items = [
        { category: "水费", amount: waterEst, actual_amount: actual.water || 0 },
        { category: "电费", amount: elecEst, actual_amount: actual.elec || 0 },
        { category: "燃气费", amount: gasEst, actual_amount: actual.gas || 0 },
        { category: "工资", amount: 0, actual_amount: actual.labor || 0 },
        { category: "设备维护费", amount: 0, actual_amount: actual.maintenance || 0 },
      ];
      for (const item of items) {
        await canteenApi.expenses.upsert({ ...item, expense_date: month + "-01", category_name: item.category } as unknown as Record<string, unknown>);
      }
      toast.success("已保存");
    } catch (e: unknown) { toast.error("保存失败", { description: (e as Error).message }); }
    finally { setSaving(false); }
  };

  const numCls = "h-7 w-24 text-center [appearance:textfield] [&::-webkit-outer-spin-button]:appearance-none [&::-webkit-inner-spin-button]:appearance-none";

  return (
    <Card>
      <CardContent className="p-4 space-y-3">
        <div className="flex items-center justify-between">
          <h3 className="text-sm font-semibold">其他费用（水电气、工资）</h3>
          <div className="flex gap-2 items-center">
            <Input type="month" className="h-8 w-36" value={month} onChange={(e) => setMonth(e.target.value)} />
            <Button size="sm" onClick={save} disabled={saving}>{saving ? "保存中…" : "保存"}</Button>
          </div>
        </div>
        <div className="flex flex-wrap gap-2 text-xs">
          <span className="bg-blue-50 text-blue-700 rounded px-2 py-1">分析合计 <b>{fmt(totalAct)}</b></span>
          <span className="bg-muted rounded px-2 py-1">估算合计 {fmt(totalEst)}</span>
          <span className="bg-muted rounded px-2 py-1">用餐人次（午+晚）{people}</span>
          <span className="bg-muted rounded px-2 py-1">计费天数 {days}</span>
        </div>
        <div className="relative overflow-y-auto max-h-[45vh] rounded-md border">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead className="w-16 text-center">科目</TableHead><TableHead className="text-center">估算参数</TableHead>
                <TableHead className="w-24 text-center">估算金额</TableHead><TableHead className="w-24 text-center">实际金额</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              <TableRow>
                <TableCell className="font-medium text-center">水费</TableCell>
                <TableCell className="text-center">
                  <div className="inline-flex items-center justify-center gap-1">
                    <Input type="number" className={numCls} value={params.water_per_capita || ""} onChange={(e) => setParams({ ...params, water_per_capita: parseFloat(e.target.value) || 0 })} />
                    <span className="text-xs text-muted-foreground">L/人 ×</span>
                    <Input type="number" className={numCls} value={params.water_price || ""} onChange={(e) => setParams({ ...params, water_price: parseFloat(e.target.value) || 0 })} />
                    <span className="text-xs text-muted-foreground">元/吨</span>
                  </div>
                </TableCell>
                <TableCell className="font-medium text-center">{fmt(waterEst)}</TableCell>
                <TableCell className="text-center"><Input type="number" className={numCls} placeholder="实际" value={actual.water > 0 ? actual.water : ""} onChange={(e) => setActual({ ...actual, water: parseFloat(e.target.value) || 0 })} /></TableCell>
              </TableRow>
              <TableRow>
                <TableCell className="font-medium text-center">电费</TableCell>
                <TableCell className="text-center"><div className="inline-flex items-center justify-center gap-1"><Input type="number" className={numCls} value={params.elec_usage || ""} onChange={(e) => setParams({ ...params, elec_usage: parseFloat(e.target.value) || 0 })} /><span className="text-xs text-muted-foreground">度/天 × {days}天</span></div></TableCell>
                <TableCell className="font-medium text-center">{fmt(elecEst)}</TableCell>
                <TableCell className="text-center"><Input type="number" className={numCls} placeholder="实际" value={actual.elec > 0 ? actual.elec : ""} onChange={(e) => setActual({ ...actual, elec: parseFloat(e.target.value) || 0 })} /></TableCell>
              </TableRow>
              <TableRow>
                <TableCell className="font-medium text-center">气费</TableCell>
                <TableCell className="text-center"><div className="inline-flex items-center justify-center gap-1"><Input type="number" className={numCls} value={params.gas_usage || ""} onChange={(e) => setParams({ ...params, gas_usage: parseFloat(e.target.value) || 0 })} /><span className="text-xs text-muted-foreground">m³/天 × {days}天 ×</span><Input type="number" className={numCls} value={params.gas_price || ""} onChange={(e) => setParams({ ...params, gas_price: parseFloat(e.target.value) || 0 })} /><span className="text-xs text-muted-foreground">元/m³</span></div></TableCell>
                <TableCell className="font-medium text-center">{fmt(gasEst)}</TableCell>
                <TableCell className="text-center"><Input type="number" className={numCls} placeholder="实际" value={actual.gas > 0 ? actual.gas : ""} onChange={(e) => setActual({ ...actual, gas: parseFloat(e.target.value) || 0 })} /></TableCell>
              </TableRow>
              <TableRow>
                <TableCell className="font-medium text-center">工资</TableCell><TableCell className="text-center text-muted-foreground">只记实际金额</TableCell>
                <TableCell className="text-center">{fmt(0)}</TableCell>
                <TableCell className="text-center"><Input type="number" className={numCls} placeholder="实际" value={actual.labor > 0 ? actual.labor : ""} onChange={(e) => setActual({ ...actual, labor: parseFloat(e.target.value) || 0 })} /></TableCell>
              </TableRow>
              <TableRow>
                <TableCell className="font-medium text-center">设备维护</TableCell><TableCell className="text-center text-muted-foreground">只记实际金额</TableCell>
                <TableCell className="text-center">{fmt(0)}</TableCell>
                <TableCell className="text-center"><Input type="number" className={numCls} placeholder="实际" value={actual.maintenance > 0 ? actual.maintenance : ""} onChange={(e) => setActual({ ...actual, maintenance: parseFloat(e.target.value) || 0 })} /></TableCell>
              </TableRow>
              <TableRow className="bg-muted/50">
                <TableCell className="text-center" colSpan={2}>合计</TableCell>
                <TableCell className="font-medium text-center">{fmt(totalEst)}</TableCell>
                <TableCell className="font-bold text-red-600 text-center">{fmt(totalAct)}</TableCell>
              </TableRow>
            </TableBody>
          </Table>
        </div>
        <p className="text-xs text-muted-foreground">实际金额手填后数据分析以实际为准，未填时采用估算金额。</p>
      </CardContent>
    </Card>
  );
}
