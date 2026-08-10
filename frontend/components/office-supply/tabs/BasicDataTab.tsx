/* eslint-disable @typescript-eslint/no-explicit-any */
"use client";

import { useState, useEffect, useCallback } from "react";
import { toast } from "sonner";
import {
  Plus, Pencil, Trash2, FolderTree, Building2, Star, Loader2,
} from "lucide-react";

import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Switch } from "@/components/ui/switch";
import { ScrollArea } from "@/components/ui/scroll-area";
import {
  Table, TableBody, TableCell, TableHead, TableHeader, TableRow,
} from "@/components/ui/table";
import {
  Dialog, DialogContent, DialogHeader, DialogTitle, DialogClose,
} from "@/components/ui/dialog";

import { officeApi } from "@/lib/api-office";

/** 基础数据 Tab：分类管理 + 供应商管理（两个子区域） */
export default function BasicDataTab() {
  return (
    <div className="space-y-6">
      <CategoriesSection />
      <SuppliersSection />
    </div>
  );
}

/* ========== 分类管理 ========== */
function CategoriesSection() {
  const [cats, setCats] = useState<any[]>([]);
  const [loading, setLoading] = useState(false);
  const [dialogOpen, setDialogOpen] = useState(false);
  const [editItem, setEditItem] = useState<any>(null);
  const [saving, setSaving] = useState(false);
  const [form, setForm] = useState({ name: "", description: "" });
  const [confirmOpen, setConfirmOpen] = useState(false);
  const [deleteTarget, setDeleteTarget] = useState<any>(null);

  const load = useCallback(async () => {
    setLoading(true);
    try { const r = await officeApi.categories.list(); setCats(r.data || []); }
    catch (e: any) { toast.error("加载分类失败", { description: e.message }); }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { load(); }, [load]);

  const openNew = () => { setEditItem(null); setForm({ name: "", description: "" }); setDialogOpen(true); };
  const openEdit = (c: any) => { setEditItem(c); setForm({ name: c.name, description: c.description || "" }); setDialogOpen(true); };

  const handleSave = async () => {
    if (!form.name.trim()) { toast.error("分类名称不能为空"); return; }
    setSaving(true);
    try {
      const body = { name: form.name.trim(), description: form.description.trim() };
      if (editItem) { await officeApi.categories.update(editItem.id, body); toast.success("分类已更新"); }
      else { await officeApi.categories.create(body); toast.success("分类已添加"); }
      setDialogOpen(false); load();
    } catch (e: any) { toast.error("保存失败", { description: e.message }); }
    finally { setSaving(false); }
  };

  const confirmDelete = async () => {
    if (!deleteTarget) return;
    try { await officeApi.categories.remove(deleteTarget.id); toast.success("已删除"); setConfirmOpen(false); setDeleteTarget(null); load(); }
    catch (e: any) { toast.error("删除失败", { description: e.message }); }
  };

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h3 className="text-lg font-semibold flex items-center gap-2"><FolderTree className="h-5 w-5" />分类管理</h3>
        <Button size="sm" onClick={openNew}><Plus className="mr-1 h-4 w-4" />新增分类</Button>
      </div>

      <Card>
        <CardContent className="p-0">
          <ScrollArea className="max-h-[300px] rounded-md border">
            <Table>
              <TableHeader className="sticky top-0 bg-muted">
                <TableRow>
                  <TableHead className="w-12 text-center text-xs">序号</TableHead>
                  <TableHead className="text-xs">名称</TableHead>
                  <TableHead className="text-xs">描述</TableHead>
                  <TableHead className="w-[100px] text-center text-xs">操作</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {loading ? (
                  <TableRow><TableCell colSpan={4} className="h-24 text-center text-muted-foreground">加载中...</TableCell></TableRow>
                ) : cats.length === 0 ? (
                  <TableRow><TableCell colSpan={4} className="h-24 text-center text-muted-foreground">暂无分类</TableCell></TableRow>
                ) : cats.map(c => (
                  <TableRow key={c.id}>
                    <TableCell className="text-center text-xs text-muted-foreground">{c.id}</TableCell>
                    <TableCell className="text-sm font-medium">{c.name}</TableCell>
                    <TableCell className="text-xs text-muted-foreground">{c.description || "-"}</TableCell>
                    <TableCell className="text-center">
                      <Button variant="ghost" size="icon" className="h-8 w-8" onClick={() => openEdit(c)}><Pencil className="h-4 w-4" /></Button>
                      <Button variant="ghost" size="icon" className="h-8 w-8" onClick={() => { setDeleteTarget(c); setConfirmOpen(true); }}><Trash2 className="h-4 w-4 text-red-500" /></Button>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </ScrollArea>
        </CardContent>
      </Card>

      <Dialog open={dialogOpen} onOpenChange={setDialogOpen}>
        <DialogContent className="sm:max-w-[420px]">
          <DialogHeader><DialogTitle>{editItem ? "编辑分类" : "新增分类"}</DialogTitle></DialogHeader>
          <div className="space-y-4 py-2">
            <div className="space-y-1.5"><Label>分类名称</Label><Input value={form.name} onChange={e => setForm(f => ({ ...f, name: e.target.value }))} placeholder="输入分类名称" /></div>
            <div className="space-y-1.5"><Label>描述</Label><Input value={form.description} onChange={e => setForm(f => ({ ...f, description: e.target.value }))} placeholder="可选的描述信息" /></div>
          </div>
          <div className="flex justify-end gap-2">
            <DialogClose asChild><Button variant="outline">取消</Button></DialogClose>
            <Button onClick={handleSave} disabled={saving}>{saving ? <Loader2 className="mr-2 h-4 w-4 animate-spin" /> : null}保存</Button>
          </div>
        </DialogContent>
      </Dialog>

      <Dialog open={confirmOpen} onOpenChange={setConfirmOpen}>
        <DialogContent className="sm:max-w-[400px]">
          <DialogHeader><DialogTitle>确认操作</DialogTitle></DialogHeader>
          <p className="text-sm text-muted-foreground py-2">确认删除分类「{deleteTarget?.name}」？</p>
          <div className="flex justify-end gap-2">
            <DialogClose asChild><Button variant="outline">取消</Button></DialogClose>
            <Button variant="destructive" onClick={confirmDelete}>确认删除</Button>
          </div>
        </DialogContent>
      </Dialog>
    </div>
  );
}

/* ========== 供应商管理 ========== */
function SuppliersSection() {
  const [items, setItems] = useState<any[]>([]);
  const [loading, setLoading] = useState(false);
  const [dialogOpen, setDialogOpen] = useState(false);
  const [edit, setEdit] = useState<any>(null);
  const [form, setForm] = useState({
    name: "", contact_person: "", phone: "", email: "", address: "",
    bank_name: "", bank_account: "", is_default: false, notes: "",
  });
  const [saving, setSaving] = useState(false);
  const [confirmOpen, setConfirmOpen] = useState(false);
  const [deleteTarget, setDeleteTarget] = useState<any>(null);

  const load = useCallback(async () => {
    setLoading(true);
    try { const r = await officeApi.suppliers.list(); setItems(r.data || []); }
    catch (e: any) { toast.error("加载供应商失败", { description: e.message }); }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { load(); }, [load]);

  const openNew = () => {
    setEdit(null);
    setForm({ name: "", contact_person: "", phone: "", email: "", address: "", bank_name: "", bank_account: "", is_default: false, notes: "" });
    setDialogOpen(true);
  };
  const openEdit = (s: any) => {
    setEdit(s);
    setForm({
      name: s.name || "", contact_person: s.contact_person || "", phone: s.phone || "",
      email: s.email || "", address: s.address || "", bank_name: s.bank_name || "",
      bank_account: s.bank_account || "", is_default: !!s.is_default, notes: s.notes || "",
    });
    setDialogOpen(true);
  };

  const handleSave = async () => {
    if (!form.name.trim()) { toast.error("供应商名称不能为空"); return; }
    setSaving(true);
    try {
      const body = { ...form, name: form.name.trim() };
      if (edit) { await officeApi.suppliers.update(edit.id, body); toast.success("供应商已更新"); }
      else { await officeApi.suppliers.create(body); toast.success("供应商已添加"); }
      setDialogOpen(false); load();
    } catch (e: any) { toast.error("保存失败", { description: e.message }); }
    finally { setSaving(false); }
  };

  const confirmDelete = async () => {
    if (!deleteTarget) return;
    try { await officeApi.suppliers.remove(deleteTarget.id); toast.success("已删除"); setConfirmOpen(false); setDeleteTarget(null); load(); }
    catch (e: any) { toast.error("删除失败", { description: e.message }); }
  };

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h3 className="text-lg font-semibold flex items-center gap-2"><Building2 className="h-5 w-5" />供应商管理</h3>
        <Button size="sm" onClick={openNew}><Plus className="mr-1 h-4 w-4" />新增供应商</Button>
      </div>

      <Card>
        <CardContent className="p-0">
          <ScrollArea className="max-h-[400px] rounded-md border">
            <Table>
              <TableHeader className="sticky top-0 bg-muted">
                <TableRow>
                  <TableHead className="text-xs">名称</TableHead><TableHead className="text-xs">联系人</TableHead>
                  <TableHead className="text-xs">电话</TableHead><TableHead className="text-xs">开户行</TableHead>
                  <TableHead className="text-xs">账号</TableHead><TableHead className="text-xs">备注</TableHead>
                  <TableHead className="w-[100px] text-center text-xs">操作</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {loading ? (
                  <TableRow><TableCell colSpan={7} className="h-24 text-center text-muted-foreground">加载中...</TableCell></TableRow>
                ) : items.length === 0 ? (
                  <TableRow><TableCell colSpan={7} className="h-24 text-center text-muted-foreground">暂无供应商</TableCell></TableRow>
                ) : items.map(s => (
                  <TableRow key={s.id}>
                    <TableCell className="text-sm font-medium">
                      {s.name}
                      {!!s.is_default && (
                        <span className="ml-2 inline-flex items-center gap-0.5 text-xs text-amber-600 bg-amber-50 border border-amber-200 rounded px-1.5 py-0.5">
                          <Star className="h-3 w-3 fill-amber-500 text-amber-500" />默认
                        </span>
                      )}
                    </TableCell>
                    <TableCell className="text-xs">{s.contact_person || "-"}</TableCell>
                    <TableCell className="text-xs">{s.phone || "-"}</TableCell>
                    <TableCell className="text-xs">{s.bank_name || "-"}</TableCell>
                    <TableCell className="text-xs font-mono">{s.bank_account || "-"}</TableCell>
                    <TableCell className="text-xs text-muted-foreground truncate max-w-[120px]">{s.notes || "-"}</TableCell>
                    <TableCell className="text-center">
                      <Button variant="ghost" size="icon" className="h-8 w-8" onClick={() => openEdit(s)}><Pencil className="h-4 w-4" /></Button>
                      <Button variant="ghost" size="icon" className="h-8 w-8" onClick={() => { setDeleteTarget(s); setConfirmOpen(true); }}><Trash2 className="h-4 w-4 text-red-500" /></Button>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </ScrollArea>
        </CardContent>
      </Card>

      <Dialog open={dialogOpen} onOpenChange={setDialogOpen}>
        <DialogContent className="sm:max-w-[560px]">
          <DialogHeader><DialogTitle>{edit ? "编辑供应商" : "新增供应商"}</DialogTitle></DialogHeader>
          <div className="space-y-3 py-2 max-h-[70vh] overflow-y-auto">
            <div className="space-y-1.5"><Label>名称 <span className="text-red-500">*</span></Label><Input value={form.name} onChange={e => setForm(f => ({ ...f, name: e.target.value }))} placeholder="供应商名称" /></div>
            <div className="grid grid-cols-2 gap-3">
              <div className="space-y-1.5"><Label>联系人</Label><Input value={form.contact_person} onChange={e => setForm(f => ({ ...f, contact_person: e.target.value }))} /></div>
              <div className="space-y-1.5"><Label>电话</Label><Input value={form.phone} onChange={e => setForm(f => ({ ...f, phone: e.target.value }))} /></div>
            </div>
            <div className="grid grid-cols-2 gap-3">
              <div className="space-y-1.5"><Label>邮箱</Label><Input type="email" value={form.email} onChange={e => setForm(f => ({ ...f, email: e.target.value }))} /></div>
              <div className="space-y-1.5"><Label>地址</Label><Input value={form.address} onChange={e => setForm(f => ({ ...f, address: e.target.value }))} /></div>
            </div>
            <div className="grid grid-cols-2 gap-3">
              <div className="space-y-1.5"><Label>开户行</Label><Input value={form.bank_name} onChange={e => setForm(f => ({ ...f, bank_name: e.target.value }))} placeholder="银行名称" /></div>
              <div className="space-y-1.5"><Label>银行账号</Label><Input value={form.bank_account} onChange={e => setForm(f => ({ ...f, bank_account: e.target.value }))} placeholder="账号" /></div>
            </div>
            <div className="space-y-1.5"><Label>备注</Label><textarea className="flex h-16 w-full rounded-md border border-input bg-transparent px-3 py-2 text-sm" value={form.notes} onChange={e => setForm(f => ({ ...f, notes: e.target.value }))} placeholder="可选备注" /></div>
            <div className="flex items-center justify-between rounded-lg border p-3">
              <div><Label className="text-sm font-medium cursor-pointer">默认供货商</Label><p className="text-xs text-muted-foreground mt-0.5">设为默认后，新建采购单将自动加载该供应商</p></div>
              <Switch checked={form.is_default} onCheckedChange={v => setForm(f => ({ ...f, is_default: v }))} />
            </div>
          </div>
          <div className="flex justify-end gap-2">
            <DialogClose asChild><Button variant="outline">取消</Button></DialogClose>
            <Button onClick={handleSave} disabled={saving}>{saving ? <Loader2 className="mr-2 h-4 w-4 animate-spin" /> : null}保存</Button>
          </div>
        </DialogContent>
      </Dialog>

      <Dialog open={confirmOpen} onOpenChange={setConfirmOpen}>
        <DialogContent className="sm:max-w-[400px]">
          <DialogHeader><DialogTitle>确认操作</DialogTitle></DialogHeader>
          <p className="text-sm text-muted-foreground py-2">确认删除供应商「{deleteTarget?.name}」？</p>
          <div className="flex justify-end gap-2">
            <DialogClose asChild><Button variant="outline">取消</Button></DialogClose>
            <Button variant="destructive" onClick={confirmDelete}>确认删除</Button>
          </div>
        </DialogContent>
      </Dialog>
    </div>
  );
}
