"use client";

import { useCallback, useEffect, useMemo, useState } from "react";
import { CalendarClock, Pencil, RotateCcw, UserCheck, UserPlus, UserX } from "lucide-react";
import { toast } from "sonner";

import { useAuth } from "@/lib/auth";
import {
  abandonOnboardingRecord, confirmOnboardingRecord, createOnboardingRecord,
  fetchOnboardingRecords, quickOnboardingRecord, restoreOnboardingRecord,
  updateOnboardingRecord, type EmploymentStatus, type OnboardingRecord,
  type OnboardingStatus,
} from "@/lib/api-onboarding";
import { OnboardingFormDialog, type OnboardingFormSubmit } from "@/components/onboarding/onboarding-form-dialog";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";

const STATUS_OPTIONS: Array<{ value: "all" | OnboardingStatus; label: string }> = [
  { value: "all", label: "全部" }, { value: "pending", label: "待入职" },
  { value: "onboarded", label: "已入职" }, { value: "abandoned", label: "已放弃" },
];

const STATUS_LABELS: Record<OnboardingStatus, string> = { pending: "待入职", onboarded: "已入职", abandoned: "已放弃" };

type FormMode = "create" | "quick" | "edit";
type ActionDialog = { type: "confirm" | "abandon"; record: OnboardingRecord } | null;

export function OnboardingManagement() {
  const { user, hasPermission } = useAuth();
  const [records, setRecords] = useState<OnboardingRecord[]>([]);
  const [filter, setFilter] = useState<"all" | OnboardingStatus>("all");
  const [loading, setLoading] = useState(true);
  const [submitting, setSubmitting] = useState(false);
  const [formMode, setFormMode] = useState<FormMode | null>(null);
  const [editingRecord, setEditingRecord] = useState<OnboardingRecord | null>(null);
  const [actionDialog, setActionDialog] = useState<ActionDialog>(null);
  const [employmentStatus, setEmploymentStatus] = useState<EmploymentStatus>("trial");
  const [abandonReason, setAbandonReason] = useState("");
  const [abandonRemarks, setAbandonRemarks] = useState("");

  const canCreate = hasPermission("employee", "create");
  const canEdit = hasPermission("employee", "edit");
  const counts = useMemo(() => STATUS_OPTIONS.slice(1).map(({ value }) => records.filter((item) => item.status === value).length), [records]);

  const loadRecords = useCallback(async () => {
    setLoading(true);
    try { setRecords(await fetchOnboardingRecords()); }
    catch (error) { toast.error(error instanceof Error ? error.message : "入职记录加载失败"); }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { if (user) void loadRecords(); }, [loadRecords, user]);

  const visibleRecords = filter === "all" ? records : records.filter((item) => item.status === filter);
  const closeForm = () => { setFormMode(null); setEditingRecord(null); };

  const submitForm = async ({ payload, employmentStatus: status }: OnboardingFormSubmit) => {
    if (!payload.name.trim() || !payload.id_number.trim() || !payload.planned_hire_date) {
      toast.error("请填写姓名、身份证号和计划入职日期"); return;
    }
    setSubmitting(true);
    try {
      if (formMode === "edit" && editingRecord) await updateOnboardingRecord(editingRecord.id, payload);
      else if (formMode === "quick") await quickOnboardingRecord({ ...payload, employment_status: status });
      else await createOnboardingRecord(payload);
      toast.success(formMode === "edit" ? "入职信息已更新" : formMode === "quick" ? "快速入职成功" : "待入职记录已创建");
      closeForm(); await loadRecords();
    } catch (error) { toast.error(error instanceof Error ? error.message : "入职信息保存失败"); }
    finally { setSubmitting(false); }
  };

  const runAction = async () => {
    if (!actionDialog) return;
    if (actionDialog.type === "abandon" && (!abandonReason.trim() || !abandonRemarks.trim())) { toast.error("放弃原因和备注必填"); return; }
    setSubmitting(true);
    try {
      if (actionDialog.type === "confirm") await confirmOnboardingRecord(actionDialog.record.id, employmentStatus);
      else await abandonOnboardingRecord(actionDialog.record.id, abandonReason, abandonRemarks);
      toast.success(actionDialog.type === "confirm" ? "已确认入职" : "已标记为放弃");
      setActionDialog(null); setAbandonReason(""); setAbandonRemarks(""); await loadRecords();
    } catch (error) { toast.error(error instanceof Error ? error.message : "状态更新失败"); }
    finally { setSubmitting(false); }
  };

  const restore = async (record: OnboardingRecord) => {
    setSubmitting(true);
    try { await restoreOnboardingRecord(record.id); toast.success("已恢复为待入职，历史信息保持不变"); await loadRecords(); }
    catch (error) { toast.error(error instanceof Error ? error.message : "恢复失败"); }
    finally { setSubmitting(false); }
  };

  if (!user) return null;

  return <div className="mx-auto flex w-full max-w-7xl flex-col gap-6">
    <header className="flex flex-wrap items-end justify-between gap-4"><div><p className="text-sm text-muted-foreground">员工管理 / 入职管理</p><h1 className="text-3xl font-bold tracking-tight">入职管理</h1><p className="mt-1 text-muted-foreground">从候选人计划到确认到岗，清晰管理每一步。</p></div>{canCreate && <div className="flex gap-2"><Button variant="outline" onClick={() => setFormMode("quick")}><UserCheck className="mr-2 h-4 w-4" />快速入职</Button><Button onClick={() => setFormMode("create")}><UserPlus className="mr-2 h-4 w-4" />登记待入职</Button></div>}</header>
    <div className="grid gap-3 sm:grid-cols-3"><StatCard label="待入职" value={counts[0]} icon={<CalendarClock />} /><StatCard label="已入职" value={counts[1]} icon={<UserCheck />} /><StatCard label="已放弃" value={counts[2]} icon={<UserX />} /></div>
    <Card><CardHeader className="gap-4 md:flex-row md:items-center md:justify-between"><div><CardTitle>入职记录</CardTitle><CardDescription>共 {records.length} 条；当前显示 {visibleRecords.length} 条</CardDescription></div><div className="flex flex-wrap gap-2" aria-label="入职状态筛选">{STATUS_OPTIONS.map((option) => <Button key={option.value} size="sm" variant={filter === option.value ? "default" : "outline"} onClick={() => setFilter(option.value)}>{option.label}</Button>)}</div></CardHeader><CardContent>{loading ? <p className="py-12 text-center text-sm text-muted-foreground">正在加载入职记录…</p> : visibleRecords.length === 0 ? <div className="py-12 text-center"><CalendarClock className="mx-auto mb-3 h-9 w-9 text-muted-foreground" /><p className="font-medium">暂无相关入职记录</p><p className="text-sm text-muted-foreground">可通过“登记待入职”开始。</p></div> : <OnboardingTable records={visibleRecords} canEdit={canEdit} submitting={submitting} onEdit={(record) => { setEditingRecord(record); setFormMode("edit"); }} onConfirm={(record) => setActionDialog({ type: "confirm", record })} onAbandon={(record) => setActionDialog({ type: "abandon", record })} onRestore={restore} />}</CardContent></Card>
    <OnboardingFormDialog open={formMode !== null} record={editingRecord} isQuick={formMode === "quick"} submitting={submitting} onClose={closeForm} onSubmit={submitForm} />
    <ActionDialog value={actionDialog} employmentStatus={employmentStatus} reason={abandonReason} remarks={abandonRemarks} submitting={submitting} onStatusChange={setEmploymentStatus} onReasonChange={setAbandonReason} onRemarksChange={setAbandonRemarks} onClose={() => setActionDialog(null)} onSubmit={runAction} />
  </div>;
}

function StatCard({ label, value, icon }: { label: string; value: number; icon: React.ReactNode }) { return <Card className="overflow-hidden"><CardContent className="flex items-center justify-between p-5"><div><p className="text-sm text-muted-foreground">{label}</p><p className="mt-1 font-mono text-3xl font-semibold tabular-nums">{value}</p></div><div className="rounded-xl bg-primary/10 p-3 text-primary [&_svg]:h-5 [&_svg]:w-5">{icon}</div></CardContent></Card>; }

function OnboardingTable({ records, canEdit, submitting, onEdit, onConfirm, onAbandon, onRestore }: { records: OnboardingRecord[]; canEdit: boolean; submitting: boolean; onEdit: (record: OnboardingRecord) => void; onConfirm: (record: OnboardingRecord) => void; onAbandon: (record: OnboardingRecord) => void; onRestore: (record: OnboardingRecord) => void }) {
  return <div className="overflow-x-auto"><table className="w-full min-w-[820px] text-sm"><thead><tr className="border-b text-left text-muted-foreground"><th className="p-3">候选人</th><th className="p-3">部门 / 岗位</th><th className="p-3">计划日期</th><th className="p-3">状态</th><th className="p-3">Offer</th><th className="p-3 text-right">操作</th></tr></thead><tbody>{records.map((record) => <tr key={record.id} className="border-b last:border-0"><td className="p-3"><p className="font-medium">{record.name}</p><p className="text-xs text-muted-foreground">{record.phone || "未留电话"}</p></td><td className="p-3">{record.department || "未分配"} / {record.position || "未填写"}</td><td className="p-3 font-mono">{record.planned_hire_date}</td><td className="p-3"><span className="rounded-full bg-secondary px-2.5 py-1 text-xs font-medium">{STATUS_LABELS[record.status]}</span></td><td className="p-3">{record.offer_id || "-"}</td><td className="p-3"><div className="flex justify-end gap-1">{canEdit && record.status === "pending" && <><Button size="sm" variant="ghost" onClick={() => onEdit(record)}><Pencil className="mr-1 h-4 w-4" />编辑</Button><Button size="sm" variant="outline" onClick={() => onAbandon(record)}><UserX className="mr-1 h-4 w-4" />放弃</Button><Button size="sm" onClick={() => onConfirm(record)}><UserCheck className="mr-1 h-4 w-4" />确认入职</Button></>}{canEdit && record.status === "abandoned" && <Button size="sm" variant="outline" disabled={submitting} onClick={() => onRestore(record)}><RotateCcw className="mr-1 h-4 w-4" />恢复待入职</Button>}</div></td></tr>)}</tbody></table></div>;
}

function ActionDialog({ value, employmentStatus, reason, remarks, submitting, onStatusChange, onReasonChange, onRemarksChange, onClose, onSubmit }: { value: ActionDialog; employmentStatus: EmploymentStatus; reason: string; remarks: string; submitting: boolean; onStatusChange: (value: EmploymentStatus) => void; onReasonChange: (value: string) => void; onRemarksChange: (value: string) => void; onClose: () => void; onSubmit: () => void }) {
  const isConfirm = value?.type === "confirm";
  return <Dialog open={value !== null} onOpenChange={(open) => !open && onClose()}><DialogContent className="sm:max-w-[560px]"><DialogHeader><DialogTitle>{isConfirm ? "确认员工入职" : "放弃本次入职"}</DialogTitle><DialogDescription>{isConfirm ? `将为 ${value?.record.name ?? "候选人"} 创建在职员工记录。` : "放弃后可恢复为待入职，原计划日期与历史信息会保留。"}</DialogDescription></DialogHeader>{isConfirm ? <div className="space-y-2"><Label htmlFor="confirm-employment">用工状态</Label><select id="confirm-employment" className="h-9 w-full rounded-md border border-input bg-background px-3 text-sm" value={employmentStatus} onChange={(event) => onStatusChange(event.target.value as EmploymentStatus)}><option value="trial">试用期</option><option value="formal">正式</option></select></div> : <div className="space-y-4"><div className="space-y-2"><Label htmlFor="abandon-reason">放弃原因 *</Label><Input id="abandon-reason" value={reason} onChange={(event) => onReasonChange(event.target.value)} /></div><div className="space-y-2"><Label htmlFor="abandon-remarks">备注 *</Label><Textarea id="abandon-remarks" value={remarks} onChange={(event) => onRemarksChange(event.target.value)} /></div></div>}<DialogFooter><Button variant="outline" onClick={onClose}>取消</Button><Button variant={isConfirm ? "default" : "destructive"} disabled={submitting} onClick={onSubmit}>{submitting ? "处理中…" : isConfirm ? "确认入职" : "确认放弃"}</Button></DialogFooter></DialogContent></Dialog>;
}
