"use client";

import { useCallback, useEffect, useMemo, useState } from "react";
import { AlertTriangle, FilePlus2, FileSignature, Pencil, ShieldCheck, XCircle } from "lucide-react";
import { toast } from "sonner";

import { fetchDocuments, type Document } from "@/lib/api";
import { activateAdminContract, cancelAdminContract, createAdminContract, fetchAdminContracts, updateAdminContract, type AdminContractPayload, type AdminContractRecord, type AdminContractStatus } from "@/lib/api-admin-contracts";
import { useAuth } from "@/lib/auth";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";

const STATUS_LABELS: Record<AdminContractStatus, string> = { draft: "草稿", active: "履行中", expired: "已到期", cancelled: "已作废" };
const STATUS_OPTIONS: Array<"all" | AdminContractStatus> = ["all", "draft", "active", "expired", "cancelled"];
const REMINDER_DAYS = 30;

export function AdminContractManagement() {
  const { user, hasPermission } = useAuth();
  const canView = hasPermission("admin_contract", "view");
  const canCreate = hasPermission("admin_contract", "create");
  const canEdit = hasPermission("admin_contract", "edit");
  const canDelete = hasPermission("admin_contract", "delete");
  const [records, setRecords] = useState<AdminContractRecord[]>([]);
  const [documents, setDocuments] = useState<Document[]>([]);
  const [status, setStatus] = useState<"all" | AdminContractStatus>("all");
  const [editing, setEditing] = useState<AdminContractRecord | null | false>(false);
  const [cancelling, setCancelling] = useState<AdminContractRecord | null>(null);
  const [loading, setLoading] = useState(true);
  const [busy, setBusy] = useState(false);

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const [items, docs] = await Promise.all([fetchAdminContracts(status === "all" ? undefined : status), fetchDocuments({ page_size: 100 })]);
      setRecords(items); setDocuments(docs.items);
    } catch (error) { toast.error(error instanceof Error ? error.message : "行政合同加载失败"); }
    finally { setLoading(false); }
  }, [status]);
  useEffect(() => { if (user && canView) void load(); }, [canView, load, user]);
  const reminderCount = useMemo(() => records.filter(isExpiringSoon).length, [records]);

  const runAction = async (action: () => Promise<unknown>, success: string, failure: string) => {
    setBusy(true); try { await action(); toast.success(success); await load(); } catch (error) { toast.error(error instanceof Error ? error.message : failure); } finally { setBusy(false); }
  };
  if (!user) return null;
  if (!canView) return <NoPermission />;
  return <div className="mx-auto flex w-full max-w-7xl flex-col gap-6">
    <header className="relative overflow-hidden rounded-3xl bg-gradient-to-br from-slate-950 via-indigo-950 to-violet-900 px-6 py-7 text-white shadow-xl"><div className="absolute -right-8 -top-12 h-40 w-40 rounded-full bg-violet-400/20 blur-3xl" /><div className="relative flex flex-wrap items-end justify-between gap-4"><div><p className="text-sm text-violet-200">行政管理 / 合同台账</p><h1 className="mt-1 text-3xl font-semibold tracking-tight">行政合同</h1><p className="mt-2 max-w-xl text-sm text-slate-300">集中管理供应商、服务及其他外部主体合同。到期只提醒，不自动改变状态。</p></div>{canCreate && <Button className="bg-white text-slate-950 hover:bg-violet-100" onClick={() => setEditing(null)}><FilePlus2 className="mr-2 h-4 w-4" />新建行政合同</Button>}</div></header>
    <div className="grid gap-3 sm:grid-cols-2"><Metric label="履行中合同" value={records.filter((item) => item.status === "active").length} icon={<ShieldCheck />} /><Metric label={`${REMINDER_DAYS} 日内到期提醒`} value={reminderCount} icon={<AlertTriangle />} warn /></div>
    <Card><CardHeader className="gap-4 md:flex-row md:items-center md:justify-between"><div><CardTitle>合同台账</CardTitle><CardDescription>草稿可编辑或手动生效；草稿和履行中合同均可填写原因作废。</CardDescription></div><div className="flex flex-wrap gap-2" aria-label="行政合同状态筛选">{STATUS_OPTIONS.map((item) => <Button key={item} size="sm" variant={status === item ? "default" : "outline"} onClick={() => setStatus(item)}>{item === "all" ? "全部" : STATUS_LABELS[item]}</Button>)}</div></CardHeader><CardContent>{loading ? <p className="py-12 text-center text-sm text-muted-foreground">正在加载行政合同…</p> : <ContractTable records={records} canEdit={canEdit} canDelete={canDelete} onEdit={setEditing} onActivate={(id) => void runAction(() => activateAdminContract(id), "合同已手动生效", "合同生效失败")} onCancel={setCancelling} />}</CardContent></Card>
    <ContractDialog record={editing} documents={documents} busy={busy} onClose={() => setEditing(false)} onSubmit={async (payload) => { await runAction(() => editing ? updateAdminContract(editing.id, payload) : createAdminContract(payload), editing ? "合同草稿已更新" : "行政合同草稿已创建", "合同保存失败"); setEditing(false); }} />
    <CancelDialog record={cancelling} busy={busy} onClose={() => setCancelling(null)} onSubmit={async (reason) => { if (!cancelling) return; await runAction(() => cancelAdminContract(cancelling.id, reason), "合同已作废，可新建替代合同", "合同作废失败"); setCancelling(null); }} />
  </div>;
}

function isExpiringSoon(record: AdminContractRecord) { const days = Math.ceil((new Date(`${record.end_date}T00:00:00`).getTime() - Date.now()) / 86_400_000); return record.status === "active" && days >= 0 && days <= REMINDER_DAYS; }
function NoPermission() { return <Card className="mx-auto mt-16 max-w-lg"><CardHeader><CardTitle>无行政合同查看权限</CardTitle><CardDescription>请联系管理员申请 admin_contract.view 权限。</CardDescription></CardHeader></Card>; }
function Metric({ label, value, icon, warn = false }: { label: string; value: number; icon: React.ReactNode; warn?: boolean }) { return <Card><CardContent className="flex items-center justify-between p-5"><div><p className="text-sm text-muted-foreground">{label}</p><p className="mt-1 font-mono text-3xl font-semibold tabular-nums">{value}</p></div><div className={`rounded-xl p-3 [&_svg]:h-5 [&_svg]:w-5 ${warn ? "bg-amber-500/10 text-amber-700" : "bg-primary/10 text-primary"}`}>{icon}</div></CardContent></Card>; }

function ContractTable({ records, canEdit, canDelete, onEdit, onActivate, onCancel }: { records: AdminContractRecord[]; canEdit: boolean; canDelete: boolean; onEdit: (record: AdminContractRecord | null) => void; onActivate: (id: number) => void; onCancel: (record: AdminContractRecord) => void }) {
  if (!records.length) return <div className="py-12 text-center"><FileSignature className="mx-auto mb-3 h-9 w-9 text-muted-foreground" /><p className="font-medium">暂无行政合同</p><p className="mt-1 text-sm text-muted-foreground">从新建一份外部主体合同开始。</p></div>;
  return <div className="overflow-x-auto"><table className="w-full min-w-[980px] text-sm"><thead><tr className="border-b text-left text-muted-foreground"><th className="p-3">合同 / 编号</th><th className="p-3">相对方</th><th className="p-3">合同类型</th><th className="p-3">期限</th><th className="p-3">状态</th><th className="p-3 text-right">操作</th></tr></thead><tbody>{records.map((record) => <tr key={record.id} className="border-b last:border-0"><td className="p-3"><p className="font-medium">{record.name}</p><p className="font-mono text-xs text-muted-foreground">{record.contract_no}</p></td><td className="p-3">{record.counterparty}</td><td className="p-3">{record.contract_type}</td><td className="p-3"><p className="font-mono">{record.start_date} 至 {record.end_date}</p>{isExpiringSoon(record) && <p className="mt-1 flex items-center gap-1 text-xs text-amber-700"><AlertTriangle className="h-3.5 w-3.5" />即将到期，仅提醒</p>}</td><td className="p-3"><span className="rounded-full bg-secondary px-2.5 py-1 text-xs font-medium">{STATUS_LABELS[record.status]}</span></td><td className="p-3"><div className="flex justify-end gap-1">{record.status === "draft" && canEdit && <><Button size="sm" variant="ghost" onClick={() => onEdit(record)}><Pencil className="mr-1 h-4 w-4" />编辑</Button><Button size="sm" variant="ghost" onClick={() => onActivate(record.id)}>生效</Button></>}{(record.status === "draft" || record.status === "active") && canDelete && <Button size="sm" variant="ghost" className="text-destructive" onClick={() => onCancel(record)}><XCircle className="mr-1 h-4 w-4" />作废</Button>}</div></td></tr>)}</tbody></table></div>;
}

function ContractDialog({ record, documents, busy, onClose, onSubmit }: { record: AdminContractRecord | null | false; documents: Document[]; busy: boolean; onClose: () => void; onSubmit: (payload: AdminContractPayload) => Promise<void> }) {
  const [form, setForm] = useState<AdminContractPayload>(emptyPayload());
  useEffect(() => setForm(record ? { contract_no: record.contract_no, name: record.name, counterparty: record.counterparty, contract_type: record.contract_type, start_date: record.start_date, end_date: record.end_date, amount_incl_tax: record.amount_incl_tax, currency: record.currency, owner: record.owner, document_id: record.document_id, remarks: record.remarks } : emptyPayload()), [record]);
  const update = (key: keyof AdminContractPayload, value: string | number | null) => setForm((current) => ({ ...current, [key]: value }));
  const submit = () => { if (!form.contract_no.trim() || !form.name.trim() || !form.counterparty.trim() || !form.contract_type.trim() || !form.start_date || !form.end_date) return toast.error("请填写全部必填项"); if (form.end_date <= form.start_date) return toast.error("结束日期须晚于开始日期"); return onSubmit({ ...form, document_id: form.document_id || null }); };
  return <Dialog open={record !== false} onOpenChange={(open) => !open && onClose()}><DialogContent className="sm:max-w-2xl"><DialogHeader><DialogTitle>{record ? "编辑行政合同草稿" : "新建行政合同"}</DialogTitle><DialogDescription>外部主体合同单关联一份档案文档；生效由管理员手动触发。</DialogDescription></DialogHeader><div className="grid gap-4 sm:grid-cols-2"><Field label="合同编号 *" value={form.contract_no} onChange={(value) => update("contract_no", value)} /><Field label="合同名称 *" value={form.name} onChange={(value) => update("name", value)} /><Field label="相对方名称 *" value={form.counterparty} onChange={(value) => update("counterparty", value)} /><Field label="合同类型 *" value={form.contract_type} onChange={(value) => update("contract_type", value)} /><Field label="开始日期 *" type="date" value={form.start_date} onChange={(value) => update("start_date", value)} /><Field label="结束日期 *" type="date" value={form.end_date} onChange={(value) => update("end_date", value)} /><Field label="含税金额" type="number" value={form.amount_incl_tax ?? ""} onChange={(value) => update("amount_incl_tax", value === "" ? null : Number(value))} /><Field label="币种" value={form.currency ?? "CNY"} onChange={(value) => update("currency", value)} /><Label>档案文档（单关联）<select className="mt-1 h-9 w-full rounded-md border bg-background px-2" value={form.document_id ?? ""} onChange={(event) => update("document_id", event.target.value ? Number(event.target.value) : null)}><option value="">不关联</option>{documents.map((doc) => <option key={doc.id} value={doc.id}>{doc.document_code} · {doc.file_name}</option>)}</select></Label><Field label="负责人" value={form.owner ?? ""} onChange={(value) => update("owner", value)} /><Label className="sm:col-span-2">备注<Textarea className="mt-1" value={form.remarks ?? ""} onChange={(event) => update("remarks", event.target.value)} /></Label></div><DialogFooter><Button variant="outline" onClick={onClose}>取消</Button><Button disabled={busy} onClick={submit}>保存草稿</Button></DialogFooter></DialogContent></Dialog>;
}
function Field({ label, value, type = "text", onChange }: { label: string; value: string | number; type?: string; onChange: (value: string) => void }) { return <Label>{label}<Input className="mt-1" type={type} value={value} onChange={(event) => onChange(event.target.value)} /></Label>; }
function emptyPayload(): AdminContractPayload { return { contract_no: "", name: "", counterparty: "", contract_type: "", start_date: "", end_date: "", amount_incl_tax: null, currency: "CNY", owner: "", document_id: null, remarks: "" }; }
function CancelDialog({ record, busy, onClose, onSubmit }: { record: AdminContractRecord | null; busy: boolean; onClose: () => void; onSubmit: (reason: string) => Promise<void> }) { const [reason, setReason] = useState(""); useEffect(() => setReason(""), [record]); return <Dialog open={record !== null} onOpenChange={(open) => !open && onClose()}><DialogContent><DialogHeader><DialogTitle>作废行政合同</DialogTitle><DialogDescription>原记录会保留在台账中；确认后可新建替代合同。</DialogDescription></DialogHeader><Label>作废原因 *<Textarea className="mt-1" value={reason} onChange={(event) => setReason(event.target.value)} /></Label><DialogFooter><Button variant="outline" onClick={onClose}>取消</Button><Button variant="destructive" disabled={busy} onClick={() => reason.trim() ? onSubmit(reason) : toast.error("作废原因必填")}>确认作废</Button></DialogFooter></DialogContent></Dialog>; }
