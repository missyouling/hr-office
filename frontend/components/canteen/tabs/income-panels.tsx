"use client";

import { useState, useEffect, useCallback, useRef } from "react";
import { Card, CardContent } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogClose } from "@/components/ui/dialog";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { toast } from "sonner";
import { canteenApi } from "@/lib/api-canteen";
import type { CanteenDailyIncome, CanteenCardRecharge, CanteenCardRefund, CanteenResourceFee } from "@/lib/api-canteen";
import { fmt } from "@/components/canteen/utils";
import { Plus, Pencil, Trash2, Upload, Download } from "lucide-react";

// ---------- CSV 导入弹窗 ----------
export function CsvImportDialog({
  open, onOpenChange, title, description,
  onImport, onDownloadTemplate,
}: {
  open: boolean; onOpenChange: (v: boolean) => void; title: string; description: string;
  onImport: (file: File) => Promise<void>; onDownloadTemplate?: () => void;
}) {
  const [fileName, setFileName] = useState("");
  const [importing, setImporting] = useState(false);
  const fileRef = useRef<HTMLInputElement>(null);

  const handleImport = async (f: File | null) => {
    if (!f) return;
    setFileName(f.name); setImporting(true);
    try { await onImport(f); toast.success("导入成功"); setFileName(""); onOpenChange(false); }
    catch (e: unknown) { toast.error("导入失败", { description: (e as Error).message }); }
    finally { setImporting(false); }
  };

  return (
    <Dialog open={open} onOpenChange={(v) => { if (!importing) { setFileName(""); onOpenChange(v); } }}>
      <DialogContent className="sm:max-w-[520px]">
        <DialogHeader><DialogTitle>{title}</DialogTitle></DialogHeader>
        <div className="space-y-3 py-2">
          <div className="rounded-md bg-blue-50 border border-blue-200 px-3 py-2.5 text-xs leading-5 text-blue-900">
            <p className="font-semibold mb-1">导入说明</p>
            <p>{description}</p>
          </div>
          {onDownloadTemplate && <Button size="sm" variant="outline" onClick={onDownloadTemplate}><Download className="mr-1 h-4 w-4" />下载模板</Button>}
          <div className="space-y-1">
            <Label className="text-xs text-muted-foreground">选择 CSV 文件</Label>
            <input ref={fileRef} type="file" accept=".csv" className="hidden" onChange={(e) => handleImport(e.target.files?.[0] || null)} />
            <div className="flex items-center gap-2">
              <Button size="sm" variant="outline" onClick={() => fileRef.current?.click()} disabled={importing}><Upload className="mr-1 h-4 w-4" />选择文件</Button>
              {fileName && <span className="text-xs text-muted-foreground">{fileName}</span>}
            </div>
          </div>
        </div>
        <div className="flex justify-end gap-2"><DialogClose asChild><Button variant="outline">取消</Button></DialogClose></div>
      </DialogContent>
    </Dialog>
  );
}

// ---------- 每日收入面板 ----------
export function IncomePanel() {
  const [list, setList] = useState<CanteenDailyIncome[]>([]);
  const [month, setMonth] = useState(new Date().toISOString().slice(0, 7));
  const [open, setOpen] = useState(false);
  const [editId, setEditId] = useState<number | null>(null);
  const [form, setForm] = useState({ income_date: new Date().toISOString().slice(0, 10), amount: 0, note: "" });
  const [confirmId, setConfirmId] = useState<number | null>(null);

  const load = useCallback(async () => {
    try { setList((await canteenApi.income.list({ month })).data); } catch (e: unknown) { toast.error("加载失败", { description: (e as Error).message }); }
  }, [month]);
  useEffect(() => { load(); }, [load]);

  const totals = list.reduce((acc, d) => ({ amount: acc.amount + Number(d.amount || 0) }), { amount: 0 });

  const save = async () => {
    if (!form.income_date) { toast.error("校验失败", { description: "日期不能为空" }); return; }
    try {
      if (editId) { await canteenApi.income.update(editId, form as Partial<CanteenDailyIncome>); toast.success("已更新"); }
      else { await canteenApi.income.create(form as Partial<CanteenDailyIncome>); toast.success("已新增"); }
      setOpen(false); load();
    } catch (e: unknown) { toast.error("保存失败", { description: (e as Error).message }); }
  };
  const del = async () => {
    if (!confirmId) return;
    try { await canteenApi.income.remove(confirmId); toast.success("已删除"); load(); }
    catch (e: unknown) { toast.error("删除失败", { description: (e as Error).message }); }
    finally { setConfirmId(null); }
  };

  return (
    <Card>
      <CardContent className="p-4 space-y-3">
        <div className="flex items-center justify-between">
          <h3 className="text-sm font-semibold">每日收入</h3>
          <div className="flex gap-2 items-center">
            <Input type="month" className="h-8 w-36" value={month} onChange={(e) => setMonth(e.target.value)} />
            <Button size="sm" onClick={() => { setEditId(null); setForm({ income_date: new Date().toISOString().slice(0, 10), amount: 0, note: "" }); setOpen(true); }}><Plus className="mr-1 h-4 w-4" />新增</Button>
          </div>
        </div>
        {list.length > 0 && <div className="flex flex-wrap gap-2 text-xs"><span className="bg-blue-50 text-blue-700 rounded px-2 py-1">月收入 <b>{fmt(totals.amount)}</b></span></div>}
        <div className="relative overflow-y-auto max-h-[45vh] rounded-md border">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead className="w-12 text-center">序号</TableHead><TableHead className="text-center">日期</TableHead>
                <TableHead className="text-center">金额</TableHead><TableHead className="text-center">备注</TableHead>
                <TableHead className="w-[100px] text-center">操作</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {list.length === 0 ? (
                <TableRow><TableCell colSpan={5} className="h-16 text-center text-muted-foreground">本月暂无收入记录</TableCell></TableRow>
              ) : list.map((d) => (
                <TableRow key={d.id}>
                  <TableCell className="text-center text-muted-foreground">{d.id}</TableCell>
                  <TableCell className="text-center">{d.income_date}</TableCell>
                  <TableCell className="font-medium text-red-600 text-center">{fmt(d.amount)}</TableCell>
                  <TableCell className="text-center text-sm text-muted-foreground">{d.note || "-"}</TableCell>
                  <TableCell className="text-center">
                    <Button variant="ghost" size="icon" onClick={() => { setEditId(d.id); setForm({ income_date: d.income_date, amount: d.amount, note: d.note || "" }); setOpen(true); }}><Pencil className="h-4 w-4" /></Button>
                    <Button variant="ghost" size="icon" onClick={() => setConfirmId(d.id)}><Trash2 className="h-4 w-4 text-red-500" /></Button>
                  </TableCell>
                </TableRow>
              ))}
              {list.length > 0 && (
                <TableRow className="bg-muted/50 font-semibold">
                  <TableCell colSpan={2} className="text-center">合计</TableCell>
                  <TableCell className="text-red-700 text-center">{fmt(totals.amount)}</TableCell>
                  <TableCell colSpan={2} />
                </TableRow>
              )}
            </TableBody>
          </Table>
        </div>
      </CardContent>
      <Dialog open={open} onOpenChange={setOpen}>
        <DialogContent className="sm:max-w-[420px]">
          <DialogHeader><DialogTitle>{editId ? "编辑收入" : "新增收入"}</DialogTitle></DialogHeader>
          <div className="space-y-3 py-2">
            <div className="space-y-1.5"><Label>日期</Label><Input type="date" value={form.income_date} onChange={(e) => setForm({ ...form, income_date: e.target.value })} /></div>
            <div className="space-y-1.5"><Label>金额</Label><Input type="number" value={form.amount || ""} onChange={(e) => setForm({ ...form, amount: parseFloat(e.target.value) || 0 })} /></div>
            <div className="space-y-1.5"><Label>备注</Label><Input value={form.note} onChange={(e) => setForm({ ...form, note: e.target.value })} /></div>
          </div>
          <div className="flex justify-end gap-2"><DialogClose asChild><Button variant="outline">取消</Button></DialogClose><Button onClick={save}>保存</Button></div>
        </DialogContent>
      </Dialog>
      <Dialog open={confirmId !== null} onOpenChange={() => setConfirmId(null)}>
        <DialogContent className="sm:max-w-[400px]"><DialogHeader><DialogTitle>确认操作</DialogTitle></DialogHeader><p className="text-sm text-muted-foreground py-2">删除此收入记录？</p><div className="flex justify-end gap-2"><DialogClose asChild><Button variant="outline">取消</Button></DialogClose><Button variant="destructive" onClick={del}>确认删除</Button></div></DialogContent>
      </Dialog>
    </Card>
  );
}

// ---------- 饭卡充值面板 ----------
export function RechargePanel() {
  const [list, setList] = useState<CanteenCardRecharge[]>([]);
  const [month, setMonth] = useState(new Date().toISOString().slice(0, 7));
  const [importOpen, setImportOpen] = useState(false);
  const [confirmId, setConfirmId] = useState<number | null>(null);

  const load = useCallback(async () => {
    try { setList((await canteenApi.recharges.list({ month })).data); } catch (e: unknown) { toast.error("加载失败", { description: (e as Error).message }); }
  }, [month]);
  useEffect(() => { load(); }, [load]);

  const del = async () => { if (!confirmId) return; try { await canteenApi.recharges.remove(confirmId); toast.success("已删除"); load(); } catch (e: unknown) { toast.error("删除失败", { description: (e as Error).message }); } finally { setConfirmId(null); } };
  const doImport = async (file: File) => { await canteenApi.recharges.import(file); load(); };
  const downloadTemplate = () => {
    const tpl = "姓名,工号,金额,日期\n张三,1001,100,2025-01-15\n";
    const blob = new Blob(["\uFEFF" + tpl], { type: "text/csv;charset=utf-8" }); const a = document.createElement("a"); a.href = URL.createObjectURL(blob); a.download = "饭卡充值导入模板.csv"; a.click(); URL.revokeObjectURL(a.href);
  };

  const totals = list.reduce((acc, d) => ({ amount: acc.amount + Number(d.amount || 0) }), { amount: 0 });

  return (
    <Card>
      <CardContent className="p-4 space-y-3">
        <div className="flex items-center justify-between">
          <h3 className="text-sm font-semibold">饭卡充值</h3>
          <div className="flex gap-2 items-center">
            <Input type="month" className="h-8 w-36" value={month} onChange={(e) => setMonth(e.target.value)} />
            <Button size="sm" variant="outline" onClick={() => setImportOpen(true)}><Upload className="mr-1 h-4 w-4" />导入</Button>
          </div>
        </div>
        <div className="relative overflow-y-auto max-h-[45vh] rounded-md border">
          <Table>
            <TableHeader><TableRow><TableHead className="w-12 text-center">序号</TableHead><TableHead className="text-center">姓名</TableHead><TableHead className="text-center">工号</TableHead><TableHead className="text-center">金额</TableHead><TableHead className="text-center">日期</TableHead><TableHead className="w-[80px] text-center">操作</TableHead></TableRow></TableHeader>
            <TableBody>
              {list.length === 0 ? <TableRow><TableCell colSpan={6} className="h-16 text-center text-muted-foreground">暂无充值记录</TableCell></TableRow> : list.map((r) => (
                <TableRow key={r.id}>
                  <TableCell className="text-center text-muted-foreground">{r.id}</TableCell><TableCell className="text-center">{r.employee_name || "-"}</TableCell><TableCell className="text-center">{r.employee_id || "-"}</TableCell>
                  <TableCell className="text-green-600 font-medium text-center">{fmt(r.amount)}</TableCell><TableCell className="text-center">{r.recharge_date || "-"}</TableCell>
                  <TableCell className="text-center"><Button variant="ghost" size="icon" onClick={() => setConfirmId(r.id)}><Trash2 className="h-4 w-4 text-red-500" /></Button></TableCell>
                </TableRow>
              ))}
              {list.length > 0 && <TableRow className="bg-muted/50 font-semibold"><TableCell colSpan={3} className="text-center">合计</TableCell><TableCell className="text-green-700 text-center">{fmt(totals.amount)}</TableCell><TableCell colSpan={2} /></TableRow>}
            </TableBody>
          </Table>
        </div>
      </CardContent>
      <CsvImportDialog open={importOpen} onOpenChange={setImportOpen} title="导入饭卡充值" description="CSV 文件需含：姓名、工号、金额、日期四列，支持 UTF-8/GBK 编码。" onImport={doImport} onDownloadTemplate={downloadTemplate} />
      <Dialog open={confirmId !== null} onOpenChange={() => setConfirmId(null)}><DialogContent className="sm:max-w-[400px]"><DialogHeader><DialogTitle>确认操作</DialogTitle></DialogHeader><p className="text-sm text-muted-foreground py-2">删除此充值记录？</p><div className="flex justify-end gap-2"><DialogClose asChild><Button variant="outline">取消</Button></DialogClose><Button variant="destructive" onClick={del}>确认删除</Button></div></DialogContent></Dialog>
    </Card>
  );
}

// ---------- 饭卡退费面板 ----------
export function RefundPanel() {
  const [list, setList] = useState<CanteenCardRefund[]>([]);
  const [month, setMonth] = useState(new Date().toISOString().slice(0, 7));
  const [importOpen, setImportOpen] = useState(false);
  const [confirmId, setConfirmId] = useState<number | null>(null);

  const load = useCallback(async () => {
    try { setList((await canteenApi.refunds.list({ month })).data); } catch (e: unknown) { toast.error("加载失败", { description: (e as Error).message }); }
  }, [month]);
  useEffect(() => { load(); }, [load]);

  const del = async () => { if (!confirmId) return; try { await canteenApi.refunds.remove(confirmId); toast.success("已删除"); load(); } catch (e: unknown) { toast.error("删除失败", { description: (e as Error).message }); } finally { setConfirmId(null); } };
  const doImport = async (file: File) => { await canteenApi.refunds.import(file); load(); };
  const downloadTemplate = () => {
    const tpl = "姓名,工号,金额,日期\n张三,1001,50,2025-01-15\n";
    const blob = new Blob(["\uFEFF" + tpl], { type: "text/csv;charset=utf-8" }); const a = document.createElement("a"); a.href = URL.createObjectURL(blob); a.download = "饭卡退费导入模板.csv"; a.click(); URL.revokeObjectURL(a.href);
  };

  const totals = list.reduce((acc, d) => ({ amount: acc.amount + Number(d.amount || 0) }), { amount: 0 });

  return (
    <Card>
      <CardContent className="p-4 space-y-3">
        <div className="flex items-center justify-between">
          <h3 className="text-sm font-semibold">饭卡退费</h3>
          <div className="flex gap-2 items-center">
            <Input type="month" className="h-8 w-36" value={month} onChange={(e) => setMonth(e.target.value)} />
            <Button size="sm" variant="outline" onClick={() => setImportOpen(true)}><Upload className="mr-1 h-4 w-4" />导入</Button>
          </div>
        </div>
        <div className="relative overflow-y-auto max-h-[45vh] rounded-md border">
          <Table>
            <TableHeader><TableRow><TableHead className="w-12 text-center">序号</TableHead><TableHead className="text-center">姓名</TableHead><TableHead className="text-center">工号</TableHead><TableHead className="text-center">金额</TableHead><TableHead className="text-center">日期</TableHead><TableHead className="w-[80px] text-center">操作</TableHead></TableRow></TableHeader>
            <TableBody>
              {list.length === 0 ? <TableRow><TableCell colSpan={6} className="h-16 text-center text-muted-foreground">暂无退费记录</TableCell></TableRow> : list.map((r) => (
                <TableRow key={r.id}>
                  <TableCell className="text-center text-muted-foreground">{r.id}</TableCell><TableCell className="text-center">{r.employee_name || "-"}</TableCell><TableCell className="text-center">{r.employee_id || "-"}</TableCell>
                  <TableCell className="text-red-600 font-medium text-center">{fmt(r.amount)}</TableCell><TableCell className="text-center">{r.refund_date || "-"}</TableCell>
                  <TableCell className="text-center"><Button variant="ghost" size="icon" onClick={() => setConfirmId(r.id)}><Trash2 className="h-4 w-4 text-red-500" /></Button></TableCell>
                </TableRow>
              ))}
              {list.length > 0 && <TableRow className="bg-muted/50 font-semibold"><TableCell colSpan={3} className="text-center">合计</TableCell><TableCell className="text-red-700 text-center">{fmt(totals.amount)}</TableCell><TableCell colSpan={2} /></TableRow>}
            </TableBody>
          </Table>
        </div>
      </CardContent>
      <CsvImportDialog open={importOpen} onOpenChange={setImportOpen} title="导入饭卡退费" description="CSV 文件需含：姓名、工号、金额、日期四列，支持 UTF-8/GBK 编码。" onImport={doImport} onDownloadTemplate={downloadTemplate} />
      <Dialog open={confirmId !== null} onOpenChange={() => setConfirmId(null)}><DialogContent className="sm:max-w-[400px]"><DialogHeader><DialogTitle>确认操作</DialogTitle></DialogHeader><p className="text-sm text-muted-foreground py-2">删除此退费记录？</p><div className="flex justify-end gap-2"><DialogClose asChild><Button variant="outline">取消</Button></DialogClose><Button variant="destructive" onClick={del}>确认删除</Button></div></DialogContent></Dialog>
    </Card>
  );
}

// ---------- 资源占用费面板 ----------
export function ResourceFeePanel() {
  const [list, setList] = useState<CanteenResourceFee[]>([]);
  const [month, setMonth] = useState(new Date().toISOString().slice(0, 7));
  const [open, setOpen] = useState(false);
  const [edit, setEdit] = useState<CanteenResourceFee | null>(null);
  const [form, setForm] = useState({ month: new Date().toISOString().slice(0, 7), amount: 0, description: "" });
  const [confirmId, setConfirmId] = useState<number | null>(null);

  const load = useCallback(async () => {
    try { setList((await canteenApi.resourceFees.list({ month })).data); } catch (e: unknown) { toast.error("加载失败", { description: (e as Error).message }); }
  }, [month]);
  useEffect(() => { load(); }, [load]);

  const save = async () => {
    try {
      if (edit) { await canteenApi.resourceFees.update(edit.id, form as Partial<CanteenResourceFee>); toast.success("已更新"); }
      else { await canteenApi.resourceFees.create(form as Partial<CanteenResourceFee>); toast.success("已新增"); }
      setOpen(false); load();
    } catch (e: unknown) { toast.error("保存失败", { description: (e as Error).message }); }
  };
  const del = async () => { if (!confirmId) return; try { await canteenApi.resourceFees.remove(confirmId); toast.success("已删除"); load(); } catch (e: unknown) { toast.error("删除失败", { description: (e as Error).message }); } finally { setConfirmId(null); } };

  const totals = list.reduce((acc, d) => ({ amount: acc.amount + Number(d.amount || 0) }), { amount: 0 });

  return (
    <Card>
      <CardContent className="p-4 space-y-3">
        <div className="flex items-center justify-between">
          <h3 className="text-sm font-semibold">资源占用费</h3>
          <div className="flex gap-2 items-center">
            <Input type="month" className="h-8 w-36" value={month} onChange={(e) => setMonth(e.target.value)} />
            <Button size="sm" onClick={() => { setEdit(null); setForm({ month, amount: 0, description: "" }); setOpen(true); }}><Plus className="mr-1 h-4 w-4" />新增</Button>
          </div>
        </div>
        {list.length > 0 && <div className="flex flex-wrap gap-2 text-xs"><span className="bg-blue-50 text-blue-700 rounded px-2 py-1">月度合计 <b>{fmt(totals.amount)}</b></span></div>}
        <div className="relative overflow-y-auto max-h-[45vh] rounded-md border">
          <Table>
            <TableHeader><TableRow><TableHead className="w-12 text-center">序号</TableHead><TableHead className="text-center">月份</TableHead><TableHead className="text-center">金额</TableHead><TableHead className="text-center">说明</TableHead><TableHead className="w-[100px] text-center">操作</TableHead></TableRow></TableHeader>
            <TableBody>
              {list.length === 0 ? <TableRow><TableCell colSpan={5} className="h-16 text-center text-muted-foreground">暂无记录</TableCell></TableRow> : list.map((f) => (
                <TableRow key={f.id}>
                  <TableCell className="text-center text-muted-foreground">{f.id}</TableCell><TableCell className="text-center">{f.month}</TableCell><TableCell className="text-center font-medium">{fmt(f.amount)}</TableCell><TableCell className="text-center text-sm">{f.description || "-"}</TableCell>
                  <TableCell className="text-center">
                    <Button variant="ghost" size="icon" onClick={() => { setEdit(f); setForm({ month: f.month, amount: f.amount, description: f.description || "" }); setOpen(true); }}><Pencil className="h-4 w-4" /></Button>
                    <Button variant="ghost" size="icon" onClick={() => setConfirmId(f.id)}><Trash2 className="h-4 w-4 text-red-500" /></Button>
                  </TableCell>
                </TableRow>
              ))}
              {list.length > 0 && <TableRow className="bg-muted/50 font-semibold"><TableCell colSpan={2} className="text-center">合计</TableCell><TableCell className="text-center">{fmt(totals.amount)}</TableCell><TableCell colSpan={2} /></TableRow>}
            </TableBody>
          </Table>
        </div>
      </CardContent>
      <Dialog open={open} onOpenChange={setOpen}>
        <DialogContent className="sm:max-w-[420px]"><DialogHeader><DialogTitle>{edit ? "编辑资源占用费" : "新增资源占用费"}</DialogTitle></DialogHeader>
          <div className="space-y-3 py-2">
            <div className="space-y-1.5"><Label>月份</Label><Input type="month" value={form.month} onChange={(e) => setForm({ ...form, month: e.target.value })} /></div>
            <div className="space-y-1.5"><Label>金额</Label><Input type="number" value={form.amount || ""} onChange={(e) => setForm({ ...form, amount: parseFloat(e.target.value) || 0 })} /></div>
            <div className="space-y-1.5"><Label>说明</Label><Input value={form.description} onChange={(e) => setForm({ ...form, description: e.target.value })} /></div>
          </div>
          <div className="flex justify-end gap-2"><DialogClose asChild><Button variant="outline">取消</Button></DialogClose><Button onClick={save}>保存</Button></div>
        </DialogContent>
      </Dialog>
      <Dialog open={confirmId !== null} onOpenChange={() => setConfirmId(null)}><DialogContent className="sm:max-w-[400px]"><DialogHeader><DialogTitle>确认操作</DialogTitle></DialogHeader><p className="text-sm text-muted-foreground py-2">删除此资源占用费记录？</p><div className="flex justify-end gap-2"><DialogClose asChild><Button variant="outline">取消</Button></DialogClose><Button variant="destructive" onClick={del}>确认删除</Button></div></DialogContent></Dialog>
    </Card>
  );
}
