"use client";

import { useState, useEffect, useCallback } from "react";
import { toast } from "sonner";
import { Plus, Trash2, Pencil, UserPlus, Loader2, Users } from "lucide-react";

import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { Dialog, DialogContent, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Badge } from "@/components/ui/badge";

import { departmentApi, type Department, type DepartmentMember } from "@/lib/api-department";

// ---- 内部对象类型 ----

interface DeptForm { name: string; code: string; parent_id: number | null; }
interface AssignForm { user_id: string; role: string; }

const EMPTY_DEPT: DeptForm = { name: "", code: "", parent_id: null };
const EMPTY_ASSIGN: AssignForm = { user_id: "", role: "member" };

// ---- 主组件 ----

export function DepartmentManagement() {
  const [list, setList] = useState<Department[]>([]);
  const [loading, setLoading] = useState(true);

  const [showForm, setShowForm] = useState(false);
  const [edit, setEdit] = useState<Department | null>(null);
  const [form, setForm] = useState<DeptForm>(EMPTY_DEPT);
  const [saving, setSaving] = useState(false);

  const [delTarget, setDelTarget] = useState<Department | null>(null);
  const [deleting, setDeleting] = useState(false);

  const [showAssign, setShowAssign] = useState(false);
  const [assignDept, setAssignDept] = useState<Department | null>(null);
  const [assignForm, setAssignForm] = useState<AssignForm>(EMPTY_ASSIGN);
  const [assigning, setAssigning] = useState(false);

  const [memDept, setMemDept] = useState<Department | null>(null);
  const [members, setMembers] = useState<DepartmentMember[]>([]);
  const [loadingMem, setLoadingMem] = useState(false);

  // ---- 加载 ----

  const reload = useCallback(async () => {
    setLoading(true);
    try {
      const res = await departmentApi.list();
      setList(res.data);
    } catch {
      toast.error("加载部门列表失败");
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => { reload(); }, [reload]);

  // ---- 打开新建/编辑 ----

  const openNew = () => { setEdit(null); setForm(EMPTY_DEPT); setShowForm(true); };
  const openEdit = (d: Department) => { setEdit(d); setForm({ name: d.name, code: d.code, parent_id: d.parent_id ?? null }); setShowForm(true); };

  // ---- 保存 ----

  const save = async () => {
    if (!form.name.trim()) { toast.error("请输入部门名称"); return; }
    setSaving(true);
    try {
      const p = { name: form.name.trim(), code: form.code.trim(), parent_id: form.parent_id };
      if (edit) { await departmentApi.update(edit.id, p); toast.success("部门已更新"); }
      else { await departmentApi.create(p); toast.success("部门已创建"); }
      setShowForm(false);
      reload();
    } catch (e) { toast.error(e instanceof Error ? e.message : "保存失败"); }
    finally { setSaving(false); }
  };

  // ---- 删除 ----

  const doDelete = async () => {
    if (!delTarget) return;
    setDeleting(true);
    try {
      await departmentApi.remove(delTarget.id);
      toast.success("部门已删除"); setDelTarget(null); reload();
    } catch (e) { toast.error(e instanceof Error ? e.message : "删除失败"); }
    finally { setDeleting(false); }
  };

  // ---- 分配用户 ----

  const doAssign = async () => {
    if (!assignDept) return;
    const uid = Number(assignForm.user_id.trim());
    if (!uid || isNaN(uid)) { toast.error("请输入有效的用户 ID"); return; }
    setAssigning(true);
    try {
      await departmentApi.assignUser(assignDept.id, { user_id: uid, role: assignForm.role });
      toast.success("用户已分配到部门"); setShowAssign(false);
    } catch (e) { toast.error(e instanceof Error ? e.message : "分配用户失败"); }
    finally { setAssigning(false); }
  };

  const openAssign = (d: Department) => { setAssignDept(d); setAssignForm(EMPTY_ASSIGN); setShowAssign(true); };

  // ---- 查看成员 ----

  const viewMembers = async (d: Department) => {
    setMemDept(d); setLoadingMem(true);
    try {
      const res = await departmentApi.listMembers(d.id);
      setMembers(res.data);
    } catch { toast.error("加载成员失败"); }
    finally { setLoadingMem(false); }
  };

  const parentName = (pid?: number | null) => pid ? list.find((d) => d.id === pid)?.name ?? "-" : "-";

  // ---- 渲染 ----

  if (loading) return <div className="p-6 text-muted-foreground">加载中...</div>;

  return (
    <div className="space-y-6 p-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold tracking-tight">部门管理</h1>
          <p className="text-muted-foreground">管理公司组织架构与部门成员分配</p>
        </div>
        <Button onClick={openNew}><Plus className="mr-2 h-4 w-4" />新增部门</Button>
      </div>

      {/* 部门表格 */}
      <Card>
        <CardHeader><CardTitle>部门列表（共 {list.length} 个）</CardTitle></CardHeader>
        <CardContent>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>部门名称</TableHead><TableHead>编码</TableHead><TableHead>上级部门</TableHead>
                <TableHead>创建时间</TableHead><TableHead className="w-[180px]">操作</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {list.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={5} className="py-8 text-center text-muted-foreground">暂无部门数据，点击「新增部门」添加</TableCell>
                </TableRow>
              ) : (
                list.map((d) => (
                  <TableRow key={d.id}>
                    <TableCell className="font-medium">{d.name}</TableCell>
                    <TableCell>{d.code ? <Badge variant="outline">{d.code}</Badge> : "-"}</TableCell>
                    <TableCell>{parentName(d.parent_id)}</TableCell>
                    <TableCell className="text-sm text-muted-foreground">{d.created_at ? new Date(d.created_at).toLocaleDateString("zh-CN") : "-"}</TableCell>
                    <TableCell>
                      <div className="flex gap-1">
                        <Button variant="ghost" size="sm" onClick={() => viewMembers(d)} title="查看成员"><Users className="h-4 w-4" /></Button>
                        <Button variant="ghost" size="sm" onClick={() => openAssign(d)} title="分配用户"><UserPlus className="h-4 w-4" /></Button>
                        <Button variant="ghost" size="sm" onClick={() => openEdit(d)} title="编辑"><Pencil className="h-4 w-4" /></Button>
                        <Button variant="ghost" size="sm" onClick={() => setDelTarget(d)} title="删除"><Trash2 className="h-4 w-4 text-destructive" /></Button>
                      </div>
                    </TableCell>
                  </TableRow>
                ))
              )}
            </TableBody>
          </Table>
        </CardContent>
      </Card>

      {/* 新建/编辑 */}
      <Dialog open={showForm} onOpenChange={setShowForm}>
        <DialogContent>
          <DialogHeader><DialogTitle>{edit ? "编辑部门" : "新增部门"}</DialogTitle></DialogHeader>
          <div className="space-y-4">
            <div className="space-y-2">
              <Label>部门名称 *</Label>
              <Input value={form.name} onChange={(e) => setForm((p) => ({ ...p, name: e.target.value }))} placeholder="如：研发部" />
            </div>
            <div className="space-y-2">
              <Label>部门编码</Label>
              <Input value={form.code} onChange={(e) => setForm((p) => ({ ...p, code: e.target.value }))} placeholder="如：RD" />
            </div>
            <div className="space-y-2">
              <Label>上级部门</Label>
              <Select value={form.parent_id != null ? String(form.parent_id) : ""} onValueChange={(v) => setForm((p) => ({ ...p, parent_id: v ? Number(v) : null }))}>
                <SelectTrigger><SelectValue placeholder="无（顶级部门）" /></SelectTrigger>
                <SelectContent>
                  <SelectItem value="">无（顶级部门）</SelectItem>
                  {list.filter((d) => !edit || d.id !== edit.id).map((d) => (
                    <SelectItem key={d.id} value={String(d.id)}>{d.name}</SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setShowForm(false)}>取消</Button>
            <Button onClick={save} disabled={saving}>{saving ? "保存中..." : "保存"}</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* 确认删除 */}
      <Dialog open={!!delTarget} onOpenChange={(o) => !o && setDelTarget(null)}>
        <DialogContent>
          <DialogHeader><DialogTitle>确认删除</DialogTitle></DialogHeader>
          <p>将永久删除部门「{delTarget?.name}」，不可恢复。</p>
          <DialogFooter>
            <Button variant="outline" onClick={() => setDelTarget(null)}>取消</Button>
            <Button variant="destructive" onClick={doDelete} disabled={deleting}>{deleting ? "删除中..." : "确认删除"}</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* 分配用户 */}
      <Dialog open={showAssign} onOpenChange={setShowAssign}>
        <DialogContent>
          <DialogHeader><DialogTitle>分配用户到「{assignDept?.name}」</DialogTitle></DialogHeader>
          <div className="space-y-4">
            <div className="space-y-2">
              <Label>用户 ID *</Label>
              <Input type="number" value={assignForm.user_id} onChange={(e) => setAssignForm((p) => ({ ...p, user_id: e.target.value }))} placeholder="输入用户数字 ID" />
            </div>
            <div className="space-y-2">
              <Label>部门角色</Label>
              <Select value={assignForm.role} onValueChange={(v) => setAssignForm((p) => ({ ...p, role: v }))}>
                <SelectTrigger><SelectValue /></SelectTrigger>
                <SelectContent>
                  <SelectItem value="member">成员</SelectItem>
                  <SelectItem value="leader">部门负责人</SelectItem>
                </SelectContent>
              </Select>
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setShowAssign(false)}>取消</Button>
            <Button onClick={doAssign} disabled={assigning}>{assigning ? "分配中..." : "确认分配"}</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* 查看成员 */}
      <Dialog open={!!memDept} onOpenChange={(o) => { if (!o) setMemDept(null); }}>
        <DialogContent>
          <DialogHeader><DialogTitle>{memDept?.name} - 部门成员</DialogTitle></DialogHeader>
          {loadingMem ? (
            <div className="py-8 text-center"><Loader2 className="mx-auto h-6 w-6 animate-spin" /></div>
          ) : members.length === 0 ? (
            <p className="py-8 text-center text-muted-foreground">暂无成员</p>
          ) : (
            <div className="max-h-64 space-y-2 overflow-y-auto">
              {members.map((m) => (
                <div key={m.id} className="flex items-center justify-between rounded-lg border p-3">
                  <span className="font-medium">用户 #{m.user_id}</span>
                  <Badge variant={m.role === "leader" ? "default" : "secondary"}>{m.role === "leader" ? "负责人" : "成员"}</Badge>
                </div>
              ))}
            </div>
          )}
        </DialogContent>
      </Dialog>
    </div>
  );
}
