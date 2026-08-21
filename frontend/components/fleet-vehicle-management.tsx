"use client";

import { useCallback, useEffect, useState } from "react";
import { CarFront, Pencil, Plus, RotateCcw, Trash2 } from "lucide-react";
import { toast } from "sonner";

import { createFleetVehicle, deleteFleetVehicle, fetchFleetVehicles, updateFleetVehicle, type FleetVehicle, type FleetVehiclePayload } from "@/lib/api-fleet-vehicles";
import { useAuth } from "@/lib/auth";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";

const STATUS_LABELS = { active: "启用中", inactive: "已停用" } as const;

export function FleetVehicleManagement() {
  const { user, hasPermission } = useAuth();
  const canView = hasPermission("fleet", "view");
  const canCreate = hasPermission("fleet", "create");
  const canEdit = hasPermission("fleet", "edit");
  const canDelete = hasPermission("fleet", "delete");
  const [records, setRecords] = useState<FleetVehicle[]>([]);
  const [editing, setEditing] = useState<FleetVehicle | null | false>(false);
  const [deleting, setDeleting] = useState<FleetVehicle | null>(null);
  const [loading, setLoading] = useState(true);
  const [busy, setBusy] = useState(false);

  const load = useCallback(async () => {
    setLoading(true);
    try { setRecords(await fetchFleetVehicles()); }
    catch (error) { toast.error(error instanceof Error ? error.message : "车辆档案加载失败"); }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { if (user && canView) void load(); }, [canView, load, user]);

  const save = async (payload: FleetVehiclePayload) => {
    setBusy(true);
    try {
      await (editing ? updateFleetVehicle(editing.id, payload) : createFleetVehicle(payload));
      toast.success(editing ? "车辆档案已更新" : "车辆档案已创建");
      await load();
      setEditing(false);
    } catch (error) { toast.error(error instanceof Error ? error.message : "车辆档案保存失败"); }
    finally { setBusy(false); }
  };

  const restore = async (record: FleetVehicle) => {
    setBusy(true);
    try {
      await updateFleetVehicle(record.id, { ...toPayload(record), status: "active" });
      toast.success("车辆已恢复启用");
      await load();
    } catch (error) { toast.error(error instanceof Error ? error.message : "恢复启用失败"); }
    finally { setBusy(false); }
  };

  const remove = async () => {
    if (!deleting) return;
    setBusy(true);
    try {
      await deleteFleetVehicle(deleting.id);
      toast.success("车辆档案已删除");
      await load();
      setDeleting(null);
    } catch (error) { toast.error(error instanceof Error ? error.message : "车辆档案删除失败"); }
    finally { setBusy(false); }
  };

  if (!user) return null;
  if (!canView) return <NoPermission />;
  return <div className="mx-auto flex w-full max-w-7xl flex-col gap-6">
    <header className="relative overflow-hidden rounded-3xl bg-gradient-to-br from-slate-950 via-cyan-950 to-teal-800 px-6 py-7 text-white shadow-xl">
      <div className="absolute -right-10 -top-14 h-44 w-44 rounded-full bg-cyan-300/20 blur-3xl" />
      <div className="relative flex flex-wrap items-end justify-between gap-4"><div><p className="text-sm text-cyan-100">日常事务 / 车队管理</p><h1 className="mt-1 text-3xl font-semibold tracking-tight">车辆档案</h1><p className="mt-2 text-sm text-slate-200">维护车辆基础信息与启停状态，不包含调度、加油或维修业务。</p></div>{canCreate && <Button className="bg-white text-slate-950 hover:bg-cyan-100" onClick={() => setEditing(null)}><Plus className="mr-2 h-4 w-4" />新增车辆</Button>}</div>
    </header>
    <Card><CardHeader><CardTitle>车辆清单</CardTitle><CardDescription>已停用车辆不可编辑或删除；需要重新使用时可直接恢复为启用中。</CardDescription></CardHeader><CardContent>{loading ? <p className="py-12 text-center text-sm text-muted-foreground">正在加载车辆档案…</p> : <VehicleTable records={records} canEdit={canEdit} canDelete={canDelete} busy={busy} onEdit={setEditing} onRestore={restore} onDelete={setDeleting} />}</CardContent></Card>
    <VehicleDialog record={editing} busy={busy} onClose={() => setEditing(false)} onSubmit={save} />
    <DeleteDialog record={deleting} busy={busy} onClose={() => setDeleting(null)} onSubmit={remove} />
  </div>;
}

function NoPermission() { return <Card className="mx-auto mt-16 max-w-lg"><CardHeader><CardTitle>无车辆档案查看权限</CardTitle><CardDescription>请联系管理员申请 fleet.view 权限。</CardDescription></CardHeader></Card>; }

function VehicleTable({ records, canEdit, canDelete, busy, onEdit, onRestore, onDelete }: { records: FleetVehicle[]; canEdit: boolean; canDelete: boolean; busy: boolean; onEdit: (record: FleetVehicle) => void; onRestore: (record: FleetVehicle) => void; onDelete: (record: FleetVehicle) => void }) {
  if (!records.length) return <div className="py-12 text-center"><CarFront className="mx-auto mb-3 h-9 w-9 text-muted-foreground" /><p className="font-medium">暂无车辆档案</p><p className="mt-1 text-sm text-muted-foreground">从新增一辆车开始建立基础台账。</p></div>;
  return <div className="overflow-x-auto"><table className="w-full min-w-[760px] text-sm"><thead><tr className="border-b text-left text-muted-foreground"><th className="p-3">车牌号</th><th className="p-3">品牌 / 车型</th><th className="p-3">座位</th><th className="p-3">购置日期</th><th className="p-3">状态</th><th className="p-3 text-right">操作</th></tr></thead><tbody>{records.map((record) => <tr key={record.id} className="border-b last:border-0"><td className="p-3 font-mono font-medium">{record.plate_number}</td><td className="p-3"><p>{record.vehicle_model}</p><p className="text-xs text-muted-foreground">{record.brand || "未填写品牌"}</p></td><td className="p-3">{record.seat_count ?? "—"}</td><td className="p-3 font-mono">{record.purchase_date || "—"}</td><td className="p-3"><span className={record.status === "active" ? "rounded-full bg-emerald-500/10 px-2.5 py-1 text-xs font-medium text-emerald-700" : "rounded-full bg-slate-500/10 px-2.5 py-1 text-xs font-medium text-slate-600"}>{STATUS_LABELS[record.status]}</span></td><td className="p-3"><div className="flex justify-end gap-1">{record.status === "active" && canEdit && <Button size="sm" variant="ghost" onClick={() => onEdit(record)}><Pencil className="mr-1 h-4 w-4" />编辑</Button>}{record.status === "inactive" && canEdit && <Button size="sm" variant="ghost" disabled={busy} onClick={() => void onRestore(record)}><RotateCcw className="mr-1 h-4 w-4" />恢复启用</Button>}{record.status === "active" && canDelete && <Button size="sm" variant="ghost" className="text-destructive" onClick={() => onDelete(record)}><Trash2 className="mr-1 h-4 w-4" />删除</Button>}</div></td></tr>)}</tbody></table></div>;
}

function VehicleDialog({ record, busy, onClose, onSubmit }: { record: FleetVehicle | null | false; busy: boolean; onClose: () => void; onSubmit: (payload: FleetVehiclePayload) => Promise<void> }) {
  const [form, setForm] = useState<FleetVehiclePayload>(emptyPayload());
  useEffect(() => setForm(record ? toPayload(record) : emptyPayload()), [record]);
  const update = (key: keyof FleetVehiclePayload, value: string | number | null) => setForm((current) => ({ ...current, [key]: value }));
  const submit = () => { if (!form.plate_number.trim() || !form.vehicle_model.trim() || !form.status) return toast.error("请填写车牌号、车型和状态"); void onSubmit({ ...form, plate_number: form.plate_number.trim(), vehicle_model: form.vehicle_model.trim() }); };
  return <Dialog open={record !== false} onOpenChange={(open) => !open && !busy && onClose()}><DialogContent showCloseButton={!busy}><DialogHeader><DialogTitle>{record ? "编辑车辆档案" : "新增车辆"}</DialogTitle><DialogDescription>状态使用清晰中文选项；停用后的车辆仅能通过列表中的“恢复启用”操作恢复。</DialogDescription></DialogHeader><div className="grid gap-4 sm:grid-cols-2"><Field label="车牌号 *" value={form.plate_number} onChange={(value) => update("plate_number", value)} /><Field label="车型 *" value={form.vehicle_model} onChange={(value) => update("vehicle_model", value)} /><Label>状态 *<select aria-label="状态" className="mt-1 h-9 w-full rounded-md border bg-background px-2" value={form.status} onChange={(event) => update("status", event.target.value as FleetVehiclePayload["status"])}><option value="active">启用中</option><option value="inactive">已停用</option></select></Label><Field label="品牌" value={form.brand ?? ""} onChange={(value) => update("brand", value || null)} /><Field label="座位数" type="number" value={form.seat_count ?? ""} onChange={(value) => update("seat_count", value === "" ? null : Number(value))} /><Field label="购置日期" type="date" value={form.purchase_date ?? ""} onChange={(value) => update("purchase_date", value || null)} /><Label className="sm:col-span-2">备注<Textarea className="mt-1" value={form.remarks ?? ""} onChange={(event) => update("remarks", event.target.value || null)} /></Label></div><DialogFooter><Button variant="outline" disabled={busy} onClick={onClose}>取消</Button><Button disabled={busy} onClick={submit}>保存</Button></DialogFooter></DialogContent></Dialog>;
}

function DeleteDialog({ record, busy, onClose, onSubmit }: { record: FleetVehicle | null; busy: boolean; onClose: () => void; onSubmit: () => Promise<void> }) { return <Dialog open={record !== null} onOpenChange={(open) => !open && !busy && onClose()}><DialogContent showCloseButton={!busy}><DialogHeader><DialogTitle>删除车辆档案</DialogTitle><DialogDescription>确认删除 {record?.plate_number}？该操作不可撤销。</DialogDescription></DialogHeader><DialogFooter><Button variant="outline" disabled={busy} onClick={onClose}>取消</Button><Button variant="destructive" disabled={busy} onClick={() => void onSubmit()}>确认删除</Button></DialogFooter></DialogContent></Dialog>; }
function Field({ label, value, type = "text", onChange }: { label: string; value: string | number; type?: string; onChange: (value: string) => void }) { return <Label>{label}<Input className="mt-1" type={type} value={value} onChange={(event) => onChange(event.target.value)} /></Label>; }
function emptyPayload(): FleetVehiclePayload { return { plate_number: "", vehicle_model: "", status: "active", brand: null, seat_count: null, purchase_date: null, remarks: null }; }
function toPayload(record: FleetVehicle): FleetVehiclePayload { return { plate_number: record.plate_number, vehicle_model: record.vehicle_model, status: record.status, brand: record.brand ?? null, seat_count: record.seat_count ?? null, purchase_date: record.purchase_date ?? null, remarks: record.remarks ?? null }; }
