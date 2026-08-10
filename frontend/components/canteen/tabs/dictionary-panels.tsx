"use client";

import { useState, useEffect, useCallback } from "react";
import { Card, CardContent } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogClose } from "@/components/ui/dialog";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { toast } from "sonner";
import { canteenApi } from "@/lib/api-canteen";
import type { CanteenCategory, CanteenSupply, CanteenExpenseCategory } from "@/lib/api-canteen";
import { Plus, Pencil, Trash2, Link2 } from "lucide-react";

// ---------- 通用编辑弹窗 ----------
export function EditDialog({
  open, onOpenChange, title, fields, values, setValues, onSave,
}: {
  open: boolean; onOpenChange: (v: boolean) => void; title: string;
  fields: { key: string; label: string; type?: string; placeholder?: string }[];
  values: Record<string, unknown>; setValues: (v: Record<string, unknown>) => void; onSave: () => void;
}) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-[420px]">
        <DialogHeader><DialogTitle>{title}</DialogTitle></DialogHeader>
        <div className="space-y-3 py-2">
          {fields.map((f) => (
            <Input
              key={f.key} type={f.type || "text"} placeholder={f.placeholder || f.label}
              value={(values[f.key] ?? "") as string}
              onChange={(e) => setValues({ ...values, [f.key]: f.type === "number" ? (parseFloat(e.target.value) || 0) : e.target.value })}
            />
          ))}
        </div>
        <div className="flex justify-end gap-2">
          <DialogClose asChild><Button variant="outline">取消</Button></DialogClose>
          <Button onClick={onSave}>保存</Button>
        </div>
      </DialogContent>
    </Dialog>
  );
}

// ---------- 食材分类面板 ----------
export function CategoryPanel() {
  const [cats, setCats] = useState<CanteenCategory[]>([]);
  const [open, setOpen] = useState(false);
  const [edit, setEdit] = useState<CanteenCategory | null>(null);
  const [form, setForm] = useState<Record<string, unknown>>({ name: "" });
  const [confirmId, setConfirmId] = useState<number | null>(null);

  const load = useCallback(async () => {
    try { setCats((await canteenApi.categories.list()).data); } catch (e: unknown) { toast.error("加载失败", { description: (e as Error).message }); }
  }, []);
  useEffect(() => { load(); }, [load]);

  const save = async () => {
    if (!(form.name as string)?.trim()) { toast.error("校验失败", { description: "名称不能为空" }); return; }
    try {
      if (edit) { await canteenApi.categories.update(edit.id, form as Partial<CanteenCategory>); toast.success("已更新"); }
      else { await canteenApi.categories.create(form as Partial<CanteenCategory>); toast.success("已新增"); }
      setOpen(false); load();
    } catch (e: unknown) { toast.error("保存失败", { description: (e as Error).message }); }
  };
  const del = async () => {
    if (!confirmId) return;
    try { await canteenApi.categories.remove(confirmId); toast.success("已删除"); load(); }
    catch (e: unknown) { toast.error("删除失败", { description: (e as Error).message }); }
    finally { setConfirmId(null); }
  };

  return (
    <Card>
      <CardContent className="p-4 space-y-3">
        <div className="flex items-center justify-between">
          <h3 className="text-sm font-semibold">食材分类</h3>
          <Button size="sm" onClick={() => { setEdit(null); setForm({ name: "" }); setOpen(true); }}>
            <Plus className="mr-1 h-4 w-4" />新增
          </Button>
        </div>
        <div className="relative max-h-[50vh] overflow-y-auto rounded-md border">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead className="w-12 text-center">序号</TableHead>
                <TableHead className="text-center">名称</TableHead>
                <TableHead className="w-[100px] text-center">操作</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {cats.length === 0 ? (
                <TableRow><TableCell colSpan={3} className="h-16 text-center text-muted-foreground">暂无分类</TableCell></TableRow>
              ) : cats.map((c) => (
                <TableRow key={c.id}>
                  <TableCell className="text-center text-muted-foreground">{c.id}</TableCell>
                  <TableCell className="font-medium text-center">{c.name}</TableCell>
                  <TableCell className="text-center">
                    <Button variant="ghost" size="icon" onClick={() => { setEdit(c); setForm({ name: c.name }); setOpen(true); }}><Pencil className="h-4 w-4" /></Button>
                    <Button variant="ghost" size="icon" onClick={() => setConfirmId(c.id)}><Trash2 className="h-4 w-4 text-red-500" /></Button>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </div>
      </CardContent>
      <EditDialog open={open} onOpenChange={setOpen} title={edit ? "编辑分类" : "新增分类"} fields={[{ key: "name", label: "分类名称" }]} values={form} setValues={setForm} onSave={save} />
      <Dialog open={confirmId !== null} onOpenChange={() => setConfirmId(null)}>
        <DialogContent className="sm:max-w-[400px]">
          <DialogHeader><DialogTitle>确认操作</DialogTitle></DialogHeader>
          <p className="text-sm text-muted-foreground py-2">删除此分类？如被食材引用则无法删除。</p>
          <div className="flex justify-end gap-2">
            <DialogClose asChild><Button variant="outline">取消</Button></DialogClose>
            <Button variant="destructive" onClick={del}>确认删除</Button>
          </div>
        </DialogContent>
      </Dialog>
    </Card>
  );
}

// ---------- 食材字典面板 ----------
export function SupplyPanel() {
  const [items, setItems] = useState<CanteenSupply[]>([]);
  const [keyword, setKeyword] = useState("");
  const [open, setOpen] = useState(false);
  const [edit, setEdit] = useState<CanteenSupply | null>(null);
  const [form, setForm] = useState<Record<string, unknown>>({ name: "", unit: "斤", unit_price: 0, category_id: "" });
  const [confirmId, setConfirmId] = useState<number | null>(null);

  const load = useCallback(async () => {
    try { setItems((await canteenApi.supplies.list({ keyword, limit: "100" })).data); } catch (e: unknown) { toast.error("加载失败", { description: (e as Error).message }); }
  }, [keyword]);
  useEffect(() => { const t = setTimeout(load, 300); return () => clearTimeout(t); }, [load]);

  const save = async () => {
    if (!(form.name as string)?.trim()) { toast.error("校验失败", { description: "品名不能为空" }); return; }
    try {
      if (edit) { await canteenApi.supplies.update(edit.id, form as Partial<CanteenSupply>); toast.success("已更新"); }
      else { await canteenApi.supplies.create(form as Partial<CanteenSupply>); toast.success("已新增"); }
      setOpen(false); load();
    } catch (e: unknown) { toast.error("保存失败", { description: (e as Error).message }); }
  };
  const del = async () => {
    if (!confirmId) return;
    try { await canteenApi.supplies.remove(confirmId); toast.success("已删除"); load(); }
    catch (e: unknown) { toast.error("删除失败", { description: (e as Error).message }); }
    finally { setConfirmId(null); }
  };

  return (
    <Card>
      <CardContent className="p-4 space-y-3">
        <div className="flex items-center justify-between gap-2 flex-wrap">
          <h3 className="text-sm font-semibold">食材字典</h3>
          <div className="flex items-center gap-2">
            <Input className="h-8 w-40" placeholder="搜索品名" value={keyword} onChange={(e) => setKeyword(e.target.value)} />
            <Button size="sm" onClick={() => { setEdit(null); setForm({ name: "", unit: "斤", unit_price: 0, category_id: "" }); setOpen(true); }}>
              <Plus className="mr-1 h-4 w-4" />新增
            </Button>
          </div>
        </div>
        <div className="relative max-h-[50vh] overflow-y-auto rounded-md border">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead className="w-12 text-center">序号</TableHead>
                <TableHead className="text-center">品名</TableHead>
                <TableHead className="w-16 text-center">单位</TableHead>
                <TableHead className="w-24 text-center">参考单价</TableHead>
                <TableHead className="w-24 text-center">分类</TableHead>
                <TableHead className="w-[100px] text-center">操作</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {items.length === 0 ? (
                <TableRow><TableCell colSpan={6} className="h-16 text-center text-muted-foreground">暂无食材</TableCell></TableRow>
              ) : items.map((s) => (
                <TableRow key={s.id}>
                  <TableCell className="text-center text-muted-foreground">{s.id}</TableCell>
                  <TableCell className="font-medium text-center">{s.name}</TableCell>
                  <TableCell className="text-center">{s.unit || "-"}</TableCell>
                  <TableCell className="text-center font-mono">¥{(s.unit_price ?? 0).toFixed(2)}</TableCell>
                  <TableCell className="text-center">{s.category_name || "-"}</TableCell>
                  <TableCell className="text-center">
                    <Button variant="ghost" size="icon" onClick={() => { setEdit(s); setForm({ name: s.name, unit: s.unit, unit_price: s.unit_price, category_id: s.category_id || "" }); setOpen(true); }}><Pencil className="h-4 w-4" /></Button>
                    <Button variant="ghost" size="icon" onClick={() => setConfirmId(s.id)}><Trash2 className="h-4 w-4 text-red-500" /></Button>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </div>
      </CardContent>
      <Dialog open={open} onOpenChange={setOpen}>
        <DialogContent className="sm:max-w-[420px]">
          <DialogHeader><DialogTitle>{edit ? "编辑食材" : "新增食材"}</DialogTitle></DialogHeader>
          <div className="space-y-3 py-2">
            <div className="space-y-1.5"><Label>品名</Label><Input value={(form.name as string) || ""} onChange={(e) => setForm({ ...form, name: e.target.value })} /></div>
            <div className="grid grid-cols-2 gap-3">
              <div className="space-y-1.5"><Label>单位</Label><Input value={(form.unit as string) || ""} onChange={(e) => setForm({ ...form, unit: e.target.value })} /></div>
              <div className="space-y-1.5"><Label>参考单价</Label><Input type="number" value={form.unit_price as number || ""} onChange={(e) => setForm({ ...form, unit_price: parseFloat(e.target.value) || 0 })} /></div>
            </div>
          </div>
          <div className="flex justify-end gap-2">
            <DialogClose asChild><Button variant="outline">取消</Button></DialogClose>
            <Button onClick={save}>保存</Button>
          </div>
        </DialogContent>
      </Dialog>
      <Dialog open={confirmId !== null} onOpenChange={() => setConfirmId(null)}>
        <DialogContent className="sm:max-w-[400px]">
          <DialogHeader><DialogTitle>确认操作</DialogTitle></DialogHeader>
          <p className="text-sm text-muted-foreground py-2">删除此食材？如被采购记录引用则无法删除。</p>
          <div className="flex justify-end gap-2">
            <DialogClose asChild><Button variant="outline">取消</Button></DialogClose>
            <Button variant="destructive" onClick={del}>确认删除</Button>
          </div>
        </DialogContent>
      </Dialog>
    </Card>
  );
}

// ---------- 费用科目面板 ----------
export function ExpenseCategoryPanel() {
  const [cats, setCats] = useState<CanteenExpenseCategory[]>([]);
  const [open, setOpen] = useState(false);
  const [edit, setEdit] = useState<CanteenExpenseCategory | null>(null);
  const [form, setForm] = useState<Record<string, unknown>>({ name: "" });
  const [confirmId, setConfirmId] = useState<number | null>(null);

  const load = useCallback(async () => {
    try { setCats((await canteenApi.expenseCategories.list()).data); } catch (e: unknown) { toast.error("加载失败", { description: (e as Error).message }); }
  }, []);
  useEffect(() => { load(); }, [load]);

  const save = async () => {
    if (!(form.name as string)?.trim()) { toast.error("校验失败", { description: "名称不能为空" }); return; }
    try {
      if (edit) { await canteenApi.expenseCategories.update(edit.id, form as Partial<CanteenExpenseCategory>); toast.success("已更新"); }
      else { await canteenApi.expenseCategories.create(form as Partial<CanteenExpenseCategory>); toast.success("已新增"); }
      setOpen(false); load();
    } catch (e: unknown) { toast.error("保存失败", { description: (e as Error).message }); }
  };
  const del = async () => {
    if (!confirmId) return;
    try { await canteenApi.expenseCategories.remove(confirmId); toast.success("已删除"); load(); }
    catch (e: unknown) { toast.error("删除失败", { description: (e as Error).message }); }
    finally { setConfirmId(null); }
  };

  return (
    <Card>
      <CardContent className="p-4 space-y-3">
        <div className="flex items-center justify-between">
          <h3 className="text-sm font-semibold">费用科目</h3>
          <Button size="sm" onClick={() => { setEdit(null); setForm({ name: "" }); setOpen(true); }}><Plus className="mr-1 h-4 w-4" />新增</Button>
        </div>
        <div className="relative max-h-[40vh] overflow-y-auto rounded-md border">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead className="w-12 text-center">序号</TableHead>
                <TableHead className="text-center">名称</TableHead>
                <TableHead className="w-[100px] text-center">操作</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {cats.length === 0 ? (
                <TableRow><TableCell colSpan={3} className="h-16 text-center text-muted-foreground">暂无科目</TableCell></TableRow>
              ) : cats.map((c) => (
                <TableRow key={c.id}>
                  <TableCell className="text-center text-muted-foreground">{c.id}</TableCell>
                  <TableCell className="font-medium text-center">{c.name}</TableCell>
                  <TableCell className="text-center">
                    <Button variant="ghost" size="icon" onClick={() => { setEdit(c); setForm({ name: c.name }); setOpen(true); }}><Pencil className="h-4 w-4" /></Button>
                    <Button variant="ghost" size="icon" onClick={() => setConfirmId(c.id)}><Trash2 className="h-4 w-4 text-red-500" /></Button>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </div>
      </CardContent>
      <EditDialog open={open} onOpenChange={setOpen} title={edit ? "编辑科目" : "新增科目"} fields={[{ key: "name", label: "科目名称" }]} values={form} setValues={setForm} onSave={save} />
      <Dialog open={confirmId !== null} onOpenChange={() => setConfirmId(null)}>
        <DialogContent className="sm:max-w-[400px]">
          <DialogHeader><DialogTitle>确认操作</DialogTitle></DialogHeader>
          <p className="text-sm text-muted-foreground py-2">删除此科目？如被费用记录引用则无法删除。</p>
          <div className="flex justify-end gap-2">
            <DialogClose asChild><Button variant="outline">取消</Button></DialogClose>
            <Button variant="destructive" onClick={del}>确认删除</Button>
          </div>
        </DialogContent>
      </Dialog>
    </Card>
  );
}

// ---------- 供应商入口 ----------
export function SupplierPanel() {
  return (
    <Card>
      <CardContent className="p-4 space-y-3">
        <div className="flex items-center justify-between"><h3 className="text-sm font-semibold">供应商</h3></div>
        <p className="text-sm text-muted-foreground">食堂采购可复用办公劳保模块的供应商数据（含联系人、电话、结算账户）。请在「日常事务 → 办公劳保 → 基础数据」中管理供应商信息。</p>
        <Button variant="outline" size="sm" disabled><Link2 className="mr-1 h-4 w-4" />前往供应商管理（办公劳保模块已收录）</Button>
      </CardContent>
    </Card>
  );
}
