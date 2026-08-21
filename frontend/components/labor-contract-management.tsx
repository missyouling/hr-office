"use client";

import { useCallback, useEffect, useMemo, useState } from "react";
import { AlertTriangle, FilePlus2, FileText, Pencil, ShieldCheck, XCircle } from "lucide-react";
import { toast } from "sonner";

import { fetchDocuments, fetchEmployees, type Document, type EmployeeResponse } from "@/lib/api";
import { activateLaborContract, cancelLaborContract, createLaborContract, fetchLaborContracts, updateLaborContract, type LaborContractPayload, type LaborContractRecord, type LaborContractStatus } from "@/lib/api-labor-contracts";
import { useAuth } from "@/lib/auth";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";

const STATUS_LABELS: Record<LaborContractStatus, string> = { draft: "草稿", active: "履行中", expired: "已到期", cancelled: "已作废" };
const STATUS_OPTIONS: Array<"all" | LaborContractStatus> = ["all", "draft", "active", "expired", "cancelled"];
const REMINDER_DAYS = 30;
type FormState = { record?: LaborContractRecord } | null;
type CancelState = LaborContractRecord | null;

export function LaborContractManagement() {
  const { user, hasPermission } = useAuth();
  // 页面入口与台账查看基于 contract.view；创建/编辑/作废分别按 contract.create/edit/delete 控制
  const canView = hasPermission("contract", "view");
  const canCreate = hasPermission("contract", "create");
  const canEdit = hasPermission("contract", "edit");
  const canDelete = hasPermission("contract", "delete");
  const [records, setRecords] = useState<LaborContractRecord[]>([]);
  const [employees, setEmployees] = useState<EmployeeResponse[]>([]);
  const [documents, setDocuments] = useState<Document[]>([]);
  const [status, setStatus] = useState<"all" | LaborContractStatus>("all");
  const [loading, setLoading] = useState(true);
  const [submitting, setSubmitting] = useState(false);
  const [form, setForm] = useState<FormState>(null);
  const [cancelRecord, setCancelRecord] = useState<CancelState>(null);

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const [contractList, employeeList, documentList] = await Promise.all([
        fetchLaborContracts(status === "all" ? undefined : status), fetchEmployees(), fetchDocuments({ page_size: 100 }),
      ]);
      setRecords(contractList);
      setEmployees(employeeList.filter((item) => item.status === "active"));
      setDocuments(documentList.items);
    } catch (error) { toast.error(error instanceof Error ? error.message : "劳动合同加载失败"); }
    finally { setLoading(false); }
  }, [status]);

  useEffect(() => { if (user && canView) void load(); }, [canView, load, user]);
  const reminderCount = useMemo(() => records.filter((item) => isExpiringSoon(item)).length, [records]);

  const activate = async (record: LaborContractRecord) => {
    setSubmitting(true);
    try { await activateLaborContract(record.id); toast.success("合同已生效，生效后不可编辑"); await load(); }
    catch (error) { toast.error(error instanceof Error ? error.message : "合同生效失败"); }
    finally { setSubmitting(false); }
  };

  if (!user) return null;
  if (!canView) return <NoPermission />;
  return <div className="mx-auto flex w-full max-w-7xl flex-col gap-6">
    <header className="flex flex-wrap items-end justify-between gap-4"><div><p className="text-sm text-muted-foreground">员工管理 / 劳动合同</p><h1 className="text-3xl font-bold tracking-tight">劳动合同</h1><p className="mt-1 text-muted-foreground">固定期限合同台账；到期仅提醒，不改变履行中的合同。</p></div>{canCreate && <Button onClick={() => setForm({})}><FilePlus2 className="mr-2 h-4 w-4" />新建合同</Button>}</header>
    <div className="grid gap-3 sm:grid-cols-2"><Metric label="履行中合同" value={records.filter((item) => item.status === "active").length} icon={<ShieldCheck />} /><Metric label={`${REMINDER_DAYS} 日内到期提醒`} value={reminderCount} icon={<AlertTriangle />} warn /></div>
    <Card><CardHeader className="gap-4 md:flex-row md:items-center md:justify-between"><div><CardTitle>合同台账</CardTitle><CardDescription>草稿可编辑或生效；生效后仅能作废并新建替代合同。</CardDescription></div><div className="flex flex-wrap gap-2" aria-label="合同状态筛选">{STATUS_OPTIONS.map((value) => <Button key={value} size="sm" variant={status === value ? "default" : "outline"} onClick={() => setStatus(value)}>{value === "all" ? "全部" : STATUS_LABELS[value]}</Button>)}</div></CardHeader><CardContent>{loading ? <p className="py-12 text-center text-sm text-muted-foreground">正在加载合同台账…</p> : <ContractTable records={records} submitting={submitting} canCreate={canCreate} canEdit={canEdit} canDelete={canDelete} onEdit={(record) => setForm({ record })} onActivate={activate} onCancel={setCancelRecord} onCreate={() => setForm({})} />}</CardContent></Card>
    <ContractDialog state={form} employees={employees} documents={documents} submitting={submitting} onClose={() => setForm(null)} onSubmit={async (payload) => { setSubmitting(true); try { if (form?.record) await updateLaborContract(form.record.id, payload); else await createLaborContract(payload); toast.success(form?.record ? "草稿已更新" : "合同草稿已创建"); setForm(null); await load(); } catch (error) { toast.error(error instanceof Error ? error.message : "合同保存失败"); } finally { setSubmitting(false); } }} />
    <CancelDialog record={cancelRecord} submitting={submitting} onClose={() => setCancelRecord(null)} onSubmit={async (reason) => { if (!cancelRecord) return; setSubmitting(true); try { await cancelLaborContract(cancelRecord.id, reason); toast.success("合同已作废，请新建替代合同"); setCancelRecord(null); await load(); } catch (error) { toast.error(error instanceof Error ? error.message : "合同作废失败"); } finally { setSubmitting(false); } }} />
  </div>;
}

function isExpiringSoon(record: LaborContractRecord) { const days = Math.ceil((new Date(`${record.end_date}T00:00:00`).getTime() - Date.now()) / 86_400_000); return record.status === "active" && days >= 0 && days <= REMINDER_DAYS; }
function NoPermission() { return <div className="mx-auto flex w-full max-w-4xl justify-center py-20"><Card className="w-full max-w-lg"><CardHeader><CardTitle>无劳动合同管理权限</CardTitle><CardDescription>仅拥有合同查看权限（contract.view）的用户可以查看合同台账。</CardDescription></CardHeader></Card></div>; }
function Metric({ label, value, icon, warn = false }: { label: string; value: number; icon: React.ReactNode; warn?: boolean }) { return <Card><CardContent className="flex items-center justify-between p-5"><div><p className="text-sm text-muted-foreground">{label}</p><p className="mt-1 font-mono text-3xl font-semibold tabular-nums">{value}</p></div><div className={`rounded-xl p-3 [&_svg]:h-5 [&_svg]:w-5 ${warn ? "bg-amber-500/10 text-amber-700" : "bg-primary/10 text-primary"}`}>{icon}</div></CardContent></Card>; }
function ContractTable({ records, submitting, canCreate, canEdit, canDelete, onEdit, onActivate, onCancel, onCreate }: { records: LaborContractRecord[]; submitting: boolean; canCreate: boolean; canEdit: boolean; canDelete: boolean; onEdit: (record: LaborContractRecord) => void; onActivate: (record: LaborContractRecord) => void; onCancel: (record: LaborContractRecord) => void; onCreate: () => void }) { if (!records.length) return <div className="py-12 text-center"><FileText className="mx-auto mb-3 h-9 w-9 text-muted-foreground" /><p className="font-medium">暂无合同记录</p><p className="mt-1 text-sm text-muted-foreground">从新建固定期限合同开始建立台账。</p></div>; return <div className="overflow-x-auto"><table className="w-full min-w-[920px] text-sm"><thead><tr className="border-b text-left text-muted-foreground"><th className="p-3">员工 / 合同编号</th><th className="p-3">部门 / 岗位</th><th className="p-3">合同期限</th><th className="p-3">附件</th><th className="p-3">状态</th><th className="p-3 text-right">操作</th></tr></thead><tbody>{records.map((record) => <tr key={record.id} className="border-b last:border-0"><td className="p-3"><p className="font-medium">{record.snapshot_name}</p><p className="font-mono text-xs text-muted-foreground">{record.contract_no}</p></td><td className="p-3">{record.snapshot_department || "未分配"} / {record.snapshot_position || "未填写"}</td><td className="p-3"><p className="font-mono">{record.start_date} 至 {record.end_date}</p>{isExpiringSoon(record) && <p className="mt-1 flex items-center gap-1 text-xs text-amber-700"><AlertTriangle className="h-3.5 w-3.5" />即将到期，仅提醒</p>}</td><td className="p-3">{record.document_id ? "1 份档案" : "-"}</td><td className="p-3"><span className="rounded-full bg-secondary px-2.5 py-1 text-xs font-medium">{STATUS_LABELS[record.status]}</span></td><td className="p-3"><div className="flex justify-end gap-1">{record.status === "draft" && canEdit && <><Button size="sm" variant="ghost" onClick={() => onEdit(record)}><Pencil className="mr-1 h-4 w-4" />编辑</Button><Button size="sm" disabled={submitting} onClick={() => onActivate(record)}><ShieldCheck className="mr-1 h-4 w-4" />生效</Button></>}{record.status === "active" && canDelete && <Button size="sm" variant="outline" disabled={submitting} onClick={() => onCancel(record)}><XCircle className="mr-1 h-4 w-4" />作废</Button>}{record.status === "cancelled" && canCreate && <Button size="sm" variant="outline" onClick={onCreate}>新建替代合同</Button>}</div></td></tr>)}</tbody></table></div>; }
function ContractDialog({ state, employees, documents, submitting, onClose, onSubmit }: { state: FormState; employees: EmployeeResponse[]; documents: Document[]; submitting: boolean; onClose: () => void; onSubmit: (payload: LaborContractPayload) => Promise<void> }) { const record = state?.record; const [employeeId, setEmployeeId] = useState(0); const [startDate, setStartDate] = useState(""); const [endDate, setEndDate] = useState(""); const [documentId, setDocumentId] = useState(0); const [remarks, setRemarks] = useState(""); useEffect(() => { setEmployeeId(record?.employee_id ?? 0); setStartDate(record?.start_date ?? ""); setEndDate(record?.end_date ?? ""); setDocumentId(record?.document_id ?? 0); setRemarks(record?.remarks ?? ""); }, [record, state]); const submit = () => { if (!employeeId || !startDate || !endDate) return toast.error("请填写员工和合同起止日期"); if (endDate <= startDate) return toast.error("合同结束日期须晚于开始日期"); return onSubmit({ employee_id: employeeId, start_date: startDate, end_date: endDate, term_months: calcTermMonths(startDate, endDate), document_id: documentId || null, remarks }); }; return <Dialog open={state !== null} onOpenChange={(open) => !open && onClose()}><DialogContent className="sm:max-w-xl"><DialogHeader><DialogTitle>{record ? "编辑合同草稿" : "新建固定期限合同"}</DialogTitle><DialogDescription>合同附件选择既有档案文档，合同生效后内容不可编辑。</DialogDescription></DialogHeader><div className="grid gap-4 sm:grid-cols-2"><Label>员工 *<select aria-label="员工" className="mt-1 h-9 w-full rounded-md border bg-background px-2" value={employeeId} onChange={(event) => setEmployeeId(Number(event.target.value))}><option value="0">请选择在职员工</option>{employees.map((item) => <option key={item.id} value={item.id}>{item.name} · {item.department || "未分配"}</option>)}</select></Label><Label>合同类型<Input className="mt-1" value="固定期限" disabled /></Label><Label>开始日期 *<Input className="mt-1" aria-label="开始日期" type="date" value={startDate} onChange={(event) => setStartDate(event.target.value)} /></Label><Label>结束日期 *<Input className="mt-1" aria-label="结束日期" type="date" value={endDate} onChange={(event) => setEndDate(event.target.value)} /></Label></div><div className="space-y-2"><Label>关联档案附件</Label><div className="max-h-32 space-y-2 overflow-y-auto rounded-md border p-3">{documents.length ? documents.map((document) => <label key={document.id} className="flex cursor-pointer items-center gap-2 text-sm"><input type="radio" name="contract-document" checked={documentId === document.id} onChange={() => setDocumentId(document.id)} />{document.file_name || document.document_code}</label>) : <p className="text-sm text-muted-foreground">暂无可关联的档案文档</p>}</div></div><Label>备注<Textarea className="mt-1" value={remarks} onChange={(event) => setRemarks(event.target.value)} /></Label><DialogFooter><Button variant="outline" onClick={onClose}>取消</Button><Button disabled={submitting} onClick={submit}>{submitting ? "保存中…" : "保存草稿"}</Button></DialogFooter></DialogContent></Dialog>; }

// calcTermMonths 按起止日期计算合同期限月数（含开始月，后端要求 term_months 为正整数）。
function calcTermMonths(start: string, end: string): number {
  const [startYear, startMonth] = start.split("-").map(Number);
  const [endYear, endMonth] = end.split("-").map(Number);
  return (endYear - startYear) * 12 + (endMonth - startMonth) + 1;
}
function CancelDialog({ record, submitting, onClose, onSubmit }: { record: CancelState; submitting: boolean; onClose: () => void; onSubmit: (reason: string) => Promise<void> }) { const [reason, setReason] = useState(""); useEffect(() => setReason(""), [record]); const submit = () => { if (!reason.trim()) return toast.error("作废原因必填"); return onSubmit(reason); }; return <Dialog open={record !== null} onOpenChange={(open) => !open && onClose()}><DialogContent><DialogHeader><DialogTitle>作废生效合同</DialogTitle><DialogDescription>作废后原合同保留台账记录，不可恢复或编辑；请按需新建替代合同。</DialogDescription></DialogHeader><Label>作废原因 *<Textarea className="mt-1" value={reason} onChange={(event) => setReason(event.target.value)} /></Label><DialogFooter><Button variant="outline" onClick={onClose}>取消</Button><Button variant="destructive" disabled={submitting} onClick={submit}>确认作废</Button></DialogFooter></DialogContent></Dialog>; }
