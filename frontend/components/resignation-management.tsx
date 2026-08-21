"use client";

import { useCallback, useEffect, useMemo, useState } from "react";
import { toast } from "sonner";
import { UserMinus, Upload, RotateCcw, Download, Eye } from "lucide-react";

import { useAuth } from "@/lib/auth";
import {
  downloadResignProof,
  fetchEmployees,
  resignEmployeeApi,
  restoreEmployees,
  type EmployeeResponse,
} from "@/lib/api";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { AlertDialog, AlertDialogAction, AlertDialogCancel, AlertDialogContent, AlertDialogDescription, AlertDialogFooter, AlertDialogHeader, AlertDialogTitle } from "@/components/ui/alert-dialog";

const MAX_PROOF_SIZE = 20 * 1024 * 1024;

type ResignationForm = { employeeId: number | null; date: string; reasons: string; proof: File | null };

const EMPTY_FORM: ResignationForm = { employeeId: null, date: "", reasons: "", proof: null };

function isResigned(employee: EmployeeResponse) {
  return employee.status === "resigned";
}

function displayDate(value: string | null) {
  return value ? value.slice(0, 10) : "-";
}

export function ResignationManagement() {
  const { user, token, hasPermission } = useAuth();
  const [employees, setEmployees] = useState<EmployeeResponse[]>([]);
  const [loading, setLoading] = useState(true);
  const [form, setForm] = useState<ResignationForm>(EMPTY_FORM);
  const [restoreTarget, setRestoreTarget] = useState<EmployeeResponse | null>(null);
  const [submitting, setSubmitting] = useState(false);
  const [proofLoading, setProofLoading] = useState<number | null>(null);

  const canEdit = hasPermission("employee", "edit");
  const resignedEmployees = useMemo(() => employees.filter(isResigned), [employees]);

  const loadEmployees = useCallback(async () => {
    setLoading(true);
    try {
      setEmployees(await fetchEmployees(token ?? undefined));
    } catch (error) {
      toast.error(error instanceof Error ? error.message : "离职员工加载失败");
    } finally {
      setLoading(false);
    }
  }, [token]);

  useEffect(() => {
    if (user) void loadEmployees();
  }, [loadEmployees, user]);

  const openResignForm = () => {
    const firstEmployee = employees.find((employee) => !isResigned(employee));
    setForm({ ...EMPTY_FORM, employeeId: firstEmployee?.id ?? null, date: new Date().toISOString().slice(0, 10) });
  };

  const submitResignation = async () => {
    if (!form.employeeId || !form.date) {
      toast.error("请选择员工和离职日期");
      return;
    }
    if (form.proof && form.proof.size > MAX_PROOF_SIZE) {
      toast.error("离职证明文件不得超过 20MB");
      return;
    }
    setSubmitting(true);
    try {
      await resignEmployeeApi(form.employeeId, form.date, form.proof, token ?? undefined, form.reasons ? [form.reasons] : []);
      toast.success("员工离职处理成功");
      setForm(EMPTY_FORM);
      await loadEmployees();
    } catch (error) {
      toast.error(error instanceof Error ? error.message : "离职办理失败");
    } finally {
      setSubmitting(false);
    }
  };

  const confirmRestore = async () => {
    if (!restoreTarget) return;
    setSubmitting(true);
    try {
      await restoreEmployees({ ids: [restoreTarget.id] }, token ?? undefined);
      toast.success("员工已恢复在职");
      setRestoreTarget(null);
      await loadEmployees();
    } catch (error) {
      toast.error(error instanceof Error ? error.message : "恢复在职失败");
    } finally {
      setSubmitting(false);
    }
  };

  const handleProof = async (employee: EmployeeResponse, download: boolean) => {
    setProofLoading(employee.id);
    try {
      const result = await downloadResignProof(employee.id, token ?? undefined);
      const url = URL.createObjectURL(result.blob);
      if (download) {
        const link = document.createElement("a");
        link.href = url;
        link.download = result.filename;
        link.click();
      } else {
        window.open(url, "_blank", "noopener,noreferrer");
      }
      window.setTimeout(() => URL.revokeObjectURL(url), 1000);
    } catch (error) {
      toast.error(error instanceof Error ? error.message : "离职证明处理失败");
    } finally {
      setProofLoading(null);
    }
  };

  if (!user || !canEdit) return null;

  return (
    <div className="mx-auto flex w-full max-w-6xl flex-col gap-6">
      <header className="flex flex-wrap items-center justify-between gap-4">
        <div>
          <p className="text-sm text-muted-foreground">员工管理 / 离职管理</p>
          <h1 className="text-3xl font-bold tracking-tight">离职管理</h1>
          <p className="text-muted-foreground">办理离职、查看证明，并按现有规则恢复在职状态。</p>
        </div>
        <Button onClick={openResignForm}><UserMinus className="mr-2 h-4 w-4" />办理离职</Button>
      </header>

      <Card>
        <CardHeader><CardTitle>离职员工</CardTitle><CardDescription>共 {resignedEmployees.length} 名离职员工</CardDescription></CardHeader>
        <CardContent>
          {loading ? <p className="py-8 text-center text-sm text-muted-foreground">正在加载离职员工...</p> : resignedEmployees.length === 0 ? <p className="py-8 text-center text-sm text-muted-foreground">暂无离职员工</p> : <ResignedTable employees={resignedEmployees} onRestore={setRestoreTarget} onProof={handleProof} proofLoading={proofLoading} />}
        </CardContent>
      </Card>

      <ResignationFormDialog form={form} employees={employees.filter((employee) => !isResigned(employee))} submitting={submitting} onChange={setForm} onSubmit={submitResignation} onClose={() => setForm(EMPTY_FORM)} />
      <AlertDialog open={!!restoreTarget} onOpenChange={(open) => !open && setRestoreTarget(null)}>
        <AlertDialogContent>
          <AlertDialogHeader><AlertDialogTitle>确认恢复在职</AlertDialogTitle><AlertDialogDescription>将恢复 {restoreTarget?.name} 的在职状态，并沿用系统规则清空离职日期、原因和证明关联。</AlertDialogDescription></AlertDialogHeader>
          <AlertDialogFooter><AlertDialogCancel disabled={submitting}>取消</AlertDialogCancel><AlertDialogAction onClick={confirmRestore} disabled={submitting}>确认恢复</AlertDialogAction></AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
}

function ResignedTable({ employees, onRestore, onProof, proofLoading }: { employees: EmployeeResponse[]; onRestore: (employee: EmployeeResponse) => void; onProof: (employee: EmployeeResponse, download: boolean) => void; proofLoading: number | null }) {
  return <div className="overflow-x-auto"><table className="w-full text-sm"><thead><tr className="border-b text-left text-muted-foreground"><th className="p-3">姓名</th><th className="p-3">部门</th><th className="p-3">离职日期</th><th className="p-3">证明</th><th className="p-3 text-right">操作</th></tr></thead><tbody>{employees.map((employee) => <tr key={employee.id} className="border-b last:border-0"><td className="p-3 font-medium">{employee.name}</td><td className="p-3">{employee.department || "-"}</td><td className="p-3">{displayDate(employee.resign_date)}</td><td className="p-3">{employee.resign_proof_name || "未上传"}</td><td className="p-3"><div className="flex justify-end gap-2">{employee.resign_proof_url && <><Button variant="ghost" size="sm" onClick={() => onProof(employee, false)} disabled={proofLoading === employee.id}><Eye className="mr-1 h-4 w-4" />预览</Button><Button variant="ghost" size="sm" onClick={() => onProof(employee, true)} disabled={proofLoading === employee.id}><Download className="mr-1 h-4 w-4" />下载</Button></>}<Button variant="outline" size="sm" onClick={() => onRestore(employee)}><RotateCcw className="mr-1 h-4 w-4" />恢复在职</Button></div></td></tr>)}</tbody></table></div>;
}

function ResignationFormDialog({ form, employees, submitting, onChange, onSubmit, onClose }: { form: ResignationForm; employees: EmployeeResponse[]; submitting: boolean; onChange: (form: ResignationForm) => void; onSubmit: () => void; onClose: () => void }) {
  if (form.employeeId === null) return null;
  return <div role="dialog" aria-label="办理离职" className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4"><div className="w-full max-w-lg rounded-xl border bg-background p-6 shadow-xl"><div className="mb-5"><h2 className="text-xl font-semibold">办理离职</h2><p className="text-sm text-muted-foreground">填写离职日期、原因和可选证明。</p></div><div className="space-y-4"><div><Label htmlFor="resignation-employee">员工</Label><select id="resignation-employee" className="mt-2 w-full rounded-md border bg-background p-2" value={form.employeeId} onChange={(event) => onChange({ ...form, employeeId: Number(event.target.value) })}>{employees.map((employee) => <option key={employee.id} value={employee.id}>{employee.name} · {employee.department || "未分配部门"}</option>)}</select></div><div><Label htmlFor="resignation-date">离职日期</Label><Input id="resignation-date" type="date" value={form.date} onChange={(event) => onChange({ ...form, date: event.target.value })} /></div><div><Label htmlFor="resignation-reasons">离职原因</Label><Textarea id="resignation-reasons" value={form.reasons} onChange={(event) => onChange({ ...form, reasons: event.target.value })} placeholder="可填写离职原因" /></div><div><Label htmlFor="resignation-proof"><Upload className="mr-1 inline h-4 w-4" />离职证明（选填）</Label><Input id="resignation-proof" type="file" accept=".pdf,image/*" className="mt-2" onChange={(event) => onChange({ ...form, proof: event.target.files?.[0] ?? null })} /></div></div><div className="mt-6 flex justify-end gap-2"><Button variant="outline" onClick={onClose} disabled={submitting}>取消</Button><Button onClick={onSubmit} disabled={submitting}>{submitting ? "提交中..." : "确认离职"}</Button></div></div></div>;
}
