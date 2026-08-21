"use client";

import { useCallback, useEffect, useState } from "react";
import { CheckCircle2, HeartPulse, Pencil, Trash2, XCircle } from "lucide-react";
import { toast } from "sonner";

import { fetchEmployees, type EmployeeResponse } from "@/lib/api";
import {
  completeOccupationalHealthCheck, createOccupationalHealthCheck, deleteOccupationalHealthCheck,
  fetchOccupationalHealthChecks, updateOccupationalHealthCheck, voidOccupationalHealthCheck,
  type OccupationalHealthCheck, type OccupationalHealthCheckPayload, type OccupationalHealthCheckStatus,
} from "@/lib/api-occupational-health-checks";
import { useAuth } from "@/lib/auth";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";

const STATUS_LABELS: Record<OccupationalHealthCheckStatus, string> = { draft: "草稿", completed: "已完成", voided: "已作废" };
const STATUS_OPTIONS: Array<"all" | OccupationalHealthCheckStatus> = ["all", "draft", "completed", "voided"];

export function OccupationalHealthCheckManagement() {
  const { user, hasPermission } = useAuth();
  const canView = hasPermission("occupational_health", "view");
  const canCreate = hasPermission("occupational_health", "create");
  const canEdit = hasPermission("occupational_health", "edit");
  const canDelete = hasPermission("occupational_health", "delete");
  const [records, setRecords] = useState<OccupationalHealthCheck[]>([]);
  const [employees, setEmployees] = useState<EmployeeResponse[]>([]);
  const [status, setStatus] = useState<"all" | OccupationalHealthCheckStatus>("all");
  const [editing, setEditing] = useState<OccupationalHealthCheck | null | false>(false);
  const [voiding, setVoiding] = useState<OccupationalHealthCheck | null>(null);
  const [loading, setLoading] = useState(true);
  const [busy, setBusy] = useState(false);

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const [items, staff] = await Promise.all([fetchOccupationalHealthChecks(status === "all" ? undefined : status), fetchEmployees()]);
      setRecords(items);
      setEmployees(staff.filter((employee) => employee.status === "active"));
    } catch (error) {
      toast.error(error instanceof Error ? error.message : "职业健康检查台账加载失败");
    } finally { setLoading(false); }
  }, [status]);

  useEffect(() => { if (user && canView) void load(); }, [canView, load, user]);

  const runAction = async (action: () => Promise<unknown>, success: string, failure: string) => {
    setBusy(true);
    try { await action(); toast.success(success); await load(); return true; }
    catch (error) { toast.error(error instanceof Error ? error.message : failure); return false; }
    finally { setBusy(false); }
  };

  if (!user) return null;
  if (!canView) return <NoPermission />;
  return <div className="mx-auto flex w-full max-w-7xl flex-col gap-6">
    <header className="relative overflow-hidden rounded-3xl bg-gradient-to-br from-teal-950 via-emerald-900 to-cyan-900 px-6 py-7 text-white shadow-xl">
      <div className="absolute -right-8 -top-12 h-40 w-40 rounded-full bg-emerald-200/20 blur-3xl" />
      <div className="relative flex flex-wrap items-end justify-between gap-4"><div><p className="text-sm text-emerald-100">行政管理 / 职业卫生</p><h1 className="mt-1 text-3xl font-semibold tracking-tight">职业健康检查</h1><p className="mt-2 max-w-xl text-sm text-emerald-50/80">独立记录职业健康检查结果与后续检查安排，不联动员工资料。</p></div>
        {canCreate && <Button className="bg-white text-emerald-950 hover:bg-emerald-50" onClick={() => setEditing(null)}><HeartPulse className="mr-2 h-4 w-4" />新建健康检查</Button>}
      </div>
    </header>
    <Card><CardHeader className="gap-4 md:flex-row md:items-center md:justify-between"><div><CardTitle>职业健康检查台账</CardTitle><CardDescription>草稿可编辑与完成；草稿和已完成记录均可填写原因作废，作废后不可恢复。</CardDescription></div><div className="flex flex-wrap gap-2" aria-label="职业健康检查状态筛选">{STATUS_OPTIONS.map((item) => <Button key={item} size="sm" variant={status === item ? "default" : "outline"} onClick={() => setStatus(item)}>{item === "all" ? "全部" : STATUS_LABELS[item]}</Button>)}</div></CardHeader>
      <CardContent>{loading ? <p className="py-12 text-center text-sm text-muted-foreground">正在加载职业健康检查台账…</p> : <CheckTable records={records} canEdit={canEdit} canDelete={canDelete} onEdit={setEditing} onComplete={(id) => void runAction(() => completeOccupationalHealthCheck(id), "健康检查已完成", "健康检查完成失败")} onDelete={(id) => void runAction(() => deleteOccupationalHealthCheck(id), "健康检查草稿已删除", "健康检查草稿删除失败")} onVoid={setVoiding} />}</CardContent>
    </Card>
    <CheckDialog record={editing} employees={employees} busy={busy} onClose={() => setEditing(false)} onSubmit={async (payload) => { const succeeded = await runAction(() => editing ? updateOccupationalHealthCheck(editing.id, payload) : createOccupationalHealthCheck(payload), editing ? "健康检查草稿已更新" : "健康检查草稿已创建", "健康检查保存失败"); if (succeeded) setEditing(false); }} />
    <VoidDialog record={voiding} busy={busy} onClose={() => setVoiding(null)} onSubmit={async (reason) => { if (!voiding) return; const succeeded = await runAction(() => voidOccupationalHealthCheck(voiding.id, reason), "健康检查已作废", "健康检查作废失败"); if (succeeded) setVoiding(null); }} />
  </div>;
}

function NoPermission() { return <Card className="mx-auto mt-16 max-w-lg"><CardHeader><CardTitle>无职业健康检查查看权限</CardTitle><CardDescription>请联系管理员申请 occupational_health.view 权限。</CardDescription></CardHeader></Card>; }

function CheckTable({ records, canEdit, canDelete, onEdit, onComplete, onDelete, onVoid }: { records: OccupationalHealthCheck[]; canEdit: boolean; canDelete: boolean; onEdit: (record: OccupationalHealthCheck) => void; onComplete: (id: number) => void; onDelete: (id: number) => void; onVoid: (record: OccupationalHealthCheck) => void }) {
  if (!records.length) return <div className="py-12 text-center"><HeartPulse className="mx-auto mb-3 h-9 w-9 text-muted-foreground" /><p className="font-medium">暂无职业健康检查记录</p><p className="mt-1 text-sm text-muted-foreground">从新建一条健康检查记录开始。</p></div>;
  return <div className="overflow-x-auto"><table className="w-full min-w-[1080px] text-sm"><thead><tr className="border-b text-left text-muted-foreground"><th className="p-3">员工</th><th className="p-3">检查日期 / 类别</th><th className="p-3">医疗机构</th><th className="p-3">检查结论</th><th className="p-3">下次检查</th><th className="p-3">状态</th><th className="p-3 text-right">操作</th></tr></thead><tbody>{records.map((record) => <tr key={record.id} className="border-b last:border-0"><td className="p-3"><p className="font-medium">{record.employee_name}</p><p className="text-xs text-muted-foreground">{[record.employee_department, record.employee_position].filter(Boolean).join(" / ") || "员工快照未提供"}</p></td><td className="p-3"><p className="font-mono text-xs">{record.check_date}</p><p>{record.check_category}</p></td><td className="p-3">{record.medical_institution}</td><td className="max-w-xs p-3"><p className="line-clamp-2">{record.check_conclusion || "-"}</p></td><td className="p-3 font-mono text-xs">{record.next_check_date || "-"}</td><td className="p-3"><span className="rounded-full bg-secondary px-2.5 py-1 text-xs font-medium">{STATUS_LABELS[record.status]}</span></td><td className="p-3"><div className="flex justify-end gap-1">{record.status === "draft" && canEdit && <><Button size="sm" variant="ghost" onClick={() => onEdit(record)}><Pencil className="mr-1 h-4 w-4" />编辑</Button><Button size="sm" variant="ghost" onClick={() => onComplete(record.id)}><CheckCircle2 className="mr-1 h-4 w-4" />完成</Button></>}{record.status === "draft" && canDelete && <Button size="sm" variant="ghost" onClick={() => onDelete(record.id)}><Trash2 className="mr-1 h-4 w-4" />删除</Button>}{record.status !== "voided" && canDelete && <Button size="sm" variant="ghost" onClick={() => onVoid(record)}><XCircle className="mr-1 h-4 w-4" />作废</Button>}</div></td></tr>)}</tbody></table></div>;
}

function CheckDialog({ record, employees, busy, onClose, onSubmit }: { record: OccupationalHealthCheck | null | false; employees: EmployeeResponse[]; busy: boolean; onClose: () => void; onSubmit: (payload: OccupationalHealthCheckPayload) => Promise<void> }) {
  const [form, setForm] = useState<OccupationalHealthCheckPayload>(emptyPayload());
  useEffect(() => setForm(record ? { employee_id: record.employee_id, check_date: record.check_date, medical_institution: record.medical_institution, check_category: record.check_category, check_conclusion: record.check_conclusion, next_check_date: record.next_check_date, remarks: record.remarks } : emptyPayload()), [record]);
  const update = (key: keyof OccupationalHealthCheckPayload, value: string | number) => setForm((current) => ({ ...current, [key]: value }));
  const submit = () => { if (!form.employee_id || !form.check_date || !form.medical_institution.trim() || !form.check_category.trim()) return toast.error("请填写员工、检查日期、医疗机构和检查类别"); return onSubmit({ ...form, medical_institution: form.medical_institution.trim(), check_category: form.check_category.trim() }); };
  return <Dialog open={record !== false} onOpenChange={(open) => !open && onClose()}><DialogContent className="sm:max-w-2xl"><DialogHeader><DialogTitle>{record ? "编辑健康检查草稿" : "新建职业健康检查"}</DialogTitle><DialogDescription>员工信息仅作为本次检查快照留存；不包含附件、异常干预、提醒或审批流程。</DialogDescription></DialogHeader><div className="grid gap-4 sm:grid-cols-2"><Label>员工 *<select aria-label="员工" className="mt-1 h-9 w-full rounded-md border bg-background px-2" value={form.employee_id || ""} onChange={(event) => update("employee_id", Number(event.target.value))}><option value="">请选择在职员工</option>{employees.map((employee) => <option key={employee.id} value={employee.id}>{employee.name} / {employee.department || "未分配"} / {employee.position || "未填写"}</option>)}</select></Label><Field label="检查日期 *" type="date" value={form.check_date} onChange={(value) => update("check_date", value)} /><Field label="医疗机构 *" value={form.medical_institution} onChange={(value) => update("medical_institution", value)} /><Field label="检查类别 *" value={form.check_category} onChange={(value) => update("check_category", value)} /><Field label="检查结论" value={form.check_conclusion || ""} onChange={(value) => update("check_conclusion", value)} /><Field label="下次检查日期" type="date" value={form.next_check_date || ""} onChange={(value) => update("next_check_date", value)} /><Label className="sm:col-span-2">备注<Textarea className="mt-1" value={form.remarks || ""} onChange={(event) => update("remarks", event.target.value)} /></Label></div><DialogFooter><Button variant="outline" onClick={onClose}>取消</Button><Button disabled={busy} onClick={submit}>保存草稿</Button></DialogFooter></DialogContent></Dialog>;
}

function Field({ label, value, type = "text", onChange }: { label: string; value: string; type?: string; onChange: (value: string) => void }) { return <Label>{label}<Input className="mt-1" type={type} value={value} onChange={(event) => onChange(event.target.value)} /></Label>; }
function emptyPayload(): OccupationalHealthCheckPayload { return { employee_id: 0, check_date: "", medical_institution: "", check_category: "", check_conclusion: "", next_check_date: "", remarks: "" }; }
function VoidDialog({ record, busy, onClose, onSubmit }: { record: OccupationalHealthCheck | null; busy: boolean; onClose: () => void; onSubmit: (reason: string) => Promise<void> }) { const [reason, setReason] = useState(""); useEffect(() => setReason(""), [record]); return <Dialog open={record !== null} onOpenChange={(open) => !open && onClose()}><DialogContent><DialogHeader><DialogTitle>作废职业健康检查</DialogTitle><DialogDescription>作废后记录终态保留，仅影响职业健康检查台账。</DialogDescription></DialogHeader><Label>作废原因 *<Textarea className="mt-1" value={reason} onChange={(event) => setReason(event.target.value)} /></Label><DialogFooter><Button variant="outline" onClick={onClose}>取消</Button><Button variant="destructive" disabled={busy} onClick={() => reason.trim() ? onSubmit(reason.trim()) : toast.error("作废原因必填")}>确认作废</Button></DialogFooter></DialogContent></Dialog>; }
