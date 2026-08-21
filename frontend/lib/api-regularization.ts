"use client";

import { request } from "@/lib/api";

export type RegularizationStatus = "pending_supervisor" | "pending_hr_review" | "scheduled" | "postponed_scheduled" | "effective" | "rejected" | "effect_failed" | "cancelled_by_resignation" | "voided";
export type RegularizationSource = "manual" | "excel_direct";

export interface RegularizationRecord {
  id: number; approval_no: string; employee_id: number | null; snapshot_name: string;
  snapshot_department: string; snapshot_position: string; snapshot_employment_status: string;
  snapshot_probation_end_date: string; contract_term_months: number; employee_self_review: string;
  initiator_hr_user_id: number | null; supervisor_approver_user_id: number | null; hr_reviewer_user_id: number | null;
  planned_regular_date: string; original_planned_regular_date: string; actual_regular_date: string;
  status: RegularizationStatus; source: RegularizationSource; extension_count: number;
  rejection_reason: string; postponed_reason: string; void_reason: string;
  initiator_submitted_at: string | null; supervisor_approved_at: string | null; supervisor_rejected_at: string | null;
  supervisor_approval_comment: string; hr_reviewed_at: string | null; hr_review_comment: string; voided_at: string | null;
  created_at: string; updated_at: string;
}

export interface CreateRegularizationPayload {
  employee_id: number; contract_term_months: number; employee_self_review: string;
  supervisor_approver_user_id: number; hr_reviewer_user_id: number; planned_regular_date: string; probation_end_date: string;
}
export interface ImportFeedback { imported?: number; records?: RegularizationRecord[]; warnings: ImportIssue[]; errors: ImportIssue[]; }
export interface ImportIssue { row: number; reason: string; }
const post = <T>(path: string, body?: object) => request<T>(path, { method: "POST", headers: { "Content-Type": "application/json" }, body: body ? JSON.stringify(body) : undefined });

export function fetchRegularizationRecords(status?: RegularizationStatus, source?: RegularizationSource) {
  const query = new URLSearchParams(); if (status) query.set("status", status); if (source) query.set("source", source);
  return request<RegularizationRecord[]>(`/regularization-records${query.size ? `?${query}` : ""}`);
}
export const fetchRegularizationRecord = (id: number) => request<RegularizationRecord>(`/regularization-records/${id}`);
export const createRegularizationRecord = (payload: CreateRegularizationPayload) => post<RegularizationRecord>("/regularization-records", payload);
export const supervisorApprove = (id: number, comment: string) => post<RegularizationRecord>(`/regularization-records/${id}/supervisor-approve`, { comment });
export const supervisorReject = (id: number, reason: string, comment: string) => post<RegularizationRecord>(`/regularization-records/${id}/supervisor-reject`, { reason, comment });
export const hrApprove = (id: number, comment: string) => post<RegularizationRecord>(`/regularization-records/${id}/hr-approve`, { comment });
export const hrReject = (id: number, reason: string, comment: string) => post<RegularizationRecord>(`/regularization-records/${id}/hr-reject`, { reason, comment });
export const postponeRegularization = (id: number, newDate: string, reason: string, comment: string) => post<RegularizationRecord>(`/regularization-records/${id}/postpone`, { new_planned_regular_date: newDate, reason, comment });
export const voidRegularization = (id: number, reason: string) => post<RegularizationRecord>(`/regularization-records/${id}/void`, { reason });
export const effectRegularization = (id: number) => post<RegularizationRecord>(`/regularization-records/${id}/effect`);

export async function importRegularization(file: File): Promise<ImportFeedback> {
  const form = new FormData(); form.append("file", file);
  try { return await request<ImportFeedback>("/regularization-records/import", { method: "POST", body: form }); }
  catch (error) {
    const message = error instanceof Error ? error.message : "导入失败";
    const json = message.match(/\{.*\}/)?.[0];
    if (json) { try { const data = JSON.parse(json) as ImportFeedback; if (data.errors || data.warnings) return { warnings: data.warnings ?? [], errors: data.errors ?? [] }; } catch { /* 保留服务端原始错误 */ } }
    throw error;
  }
}
export function downloadRegularizationTemplate() { window.open("/regularization-records/template", "_blank", "noopener,noreferrer"); }
