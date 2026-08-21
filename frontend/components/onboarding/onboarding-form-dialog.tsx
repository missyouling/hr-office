"use client";

import { useEffect, useState } from "react";

import type { EmploymentStatus, OnboardingPayload, OnboardingRecord } from "@/lib/api-onboarding";
import { Button } from "@/components/ui/button";
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";

const EMPTY_PAYLOAD: OnboardingPayload = {
  name: "", id_number: "", phone: "", department: "", position: "",
  planned_hire_date: "", remarks: "", offer_id: "", offer_source: "",
};

export interface OnboardingFormSubmit {
  payload: OnboardingPayload;
  employmentStatus: EmploymentStatus;
}

interface Props {
  open: boolean;
  record: OnboardingRecord | null;
  isQuick: boolean;
  submitting: boolean;
  onClose: () => void;
  onSubmit: (value: OnboardingFormSubmit) => void;
}

function payloadFromRecord(record: OnboardingRecord | null): OnboardingPayload {
  if (!record) return EMPTY_PAYLOAD;
  return {
    name: record.name, id_number: record.id_number, phone: record.phone,
    department: record.department, position: record.position,
    planned_hire_date: record.planned_hire_date, remarks: record.remarks,
    offer_id: record.offer_id, offer_source: record.offer_source,
  };
}

export function OnboardingFormDialog({ open, record, isQuick, submitting, onClose, onSubmit }: Props) {
  const [payload, setPayload] = useState<OnboardingPayload>(EMPTY_PAYLOAD);
  const [employmentStatus, setEmploymentStatus] = useState<EmploymentStatus>("trial");

  useEffect(() => {
    if (open) setPayload(payloadFromRecord(record));
  }, [open, record]);

  const update = (field: keyof OnboardingPayload, value: string) => {
    setPayload((current) => ({ ...current, [field]: value }));
  };

  return (
    <Dialog open={open} onOpenChange={(next) => !next && onClose()}>
      <DialogContent className="max-h-[90vh] overflow-y-auto sm:max-w-[800px]">
        <DialogHeader>
          <DialogTitle>{record ? "编辑待入职信息" : isQuick ? "快速入职" : "登记待入职"}</DialogTitle>
          <DialogDescription>{isQuick ? "保存后直接创建在职员工。" : "先登记计划，后续再确认入职。"}</DialogDescription>
        </DialogHeader>
        <div className="grid gap-4 py-2 md:grid-cols-2">
          <FormField id="onboarding-name" label="姓名 *" value={payload.name} onChange={(value) => update("name", value)} />
          <FormField id="onboarding-id-number" label="身份证号 *" value={payload.id_number} onChange={(value) => update("id_number", value)} />
          <FormField id="onboarding-phone" label="联系电话" value={payload.phone} onChange={(value) => update("phone", value)} />
          <FormField id="onboarding-date" label="计划入职日期 *" type="date" value={payload.planned_hire_date} onChange={(value) => update("planned_hire_date", value)} />
          <FormField id="onboarding-department" label="拟入职部门" value={payload.department} onChange={(value) => update("department", value)} />
          <FormField id="onboarding-position" label="拟入职岗位" value={payload.position} onChange={(value) => update("position", value)} />
          <FormField id="onboarding-offer-id" label="Offer 编号" value={payload.offer_id} onChange={(value) => update("offer_id", value)} />
          <FormField id="onboarding-offer-source" label="Offer 来源" value={payload.offer_source} onChange={(value) => update("offer_source", value)} />
          {isQuick && <div className="space-y-2"><Label htmlFor="onboarding-employment">用工状态</Label><select id="onboarding-employment" className="h-9 w-full rounded-md border border-input bg-background px-3 text-sm" value={employmentStatus} onChange={(event) => setEmploymentStatus(event.target.value as EmploymentStatus)}><option value="trial">试用期</option><option value="formal">正式</option></select></div>}
          <div className="space-y-2 md:col-span-2"><Label htmlFor="onboarding-remarks">备注</Label><Textarea id="onboarding-remarks" value={payload.remarks} onChange={(event) => update("remarks", event.target.value)} placeholder="补充候选人沟通信息" /></div>
        </div>
        <DialogFooter><Button variant="outline" onClick={onClose} disabled={submitting}>取消</Button><Button onClick={() => onSubmit({ payload, employmentStatus })} disabled={submitting}>{submitting ? "保存中…" : record ? "保存修改" : isQuick ? "确认快速入职" : "保存登记"}</Button></DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

function FormField({ id, label, value, type = "text", onChange }: { id: string; label: string; value: string; type?: string; onChange: (value: string) => void }) {
  return <div className="space-y-2"><Label htmlFor={id}>{label}</Label><Input id={id} type={type} value={value} onChange={(event) => onChange(event.target.value)} /></div>;
}
