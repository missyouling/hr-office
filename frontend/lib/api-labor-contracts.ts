"use client";

import { request } from "@/lib/api";

export type LaborContractStatus = "draft" | "active" | "expired" | "cancelled";

/**
 * 劳动合同记录（对齐后端 models.LaborContract JSON 契约，见 backend/internal/models/contract.go）。
 * 快照字段（snapshot_*）创建时从员工主表拷贝后冻结；附件为单个 document_id + document 对象。
 */
export interface LaborContractRecord {
  id: number;
  employee_id: number | null;
  snapshot_name: string;
  snapshot_department: string;
  snapshot_position: string;
  snapshot_id_number: string;
  contract_no: string;
  contract_type: "fixed_term";
  start_date: string;
  end_date: string;
  term_months: number;
  document_id: number | null;
  document?: { id: number; document_code: string; file_name: string } | null;
  status: LaborContractStatus;
  activated_at: string | null;
  expired_at: string | null;
  cancelled_at: string | null;
  cancel_reason: string;
  remarks: string;
  created_at: string;
  updated_at: string;
}

/**
 * 创建/更新劳动合同请求体（对齐后端 contractPayload）。
 * 关联员工时快照从员工主表拷贝；未关联员工时需提供 name/department/position/id_number。
 */
export interface LaborContractPayload {
  employee_id: number | null;
  contract_no?: string;
  start_date: string;
  end_date: string;
  term_months: number;
  document_id?: number | null;
  remarks?: string;
  name?: string;
  department?: string;
  position?: string;
  id_number?: string;
}

function post<T>(path: string, body?: object): Promise<T> {
  return request<T>(path, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: body ? JSON.stringify(body) : undefined,
  });
}

export function fetchLaborContracts(status?: LaborContractStatus): Promise<LaborContractRecord[]> {
  const query = status ? `?status=${encodeURIComponent(status)}` : "";
  return request<LaborContractRecord[]>(`/contracts${query}`);
}

export function createLaborContract(payload: LaborContractPayload): Promise<LaborContractRecord> {
  return post<LaborContractRecord>("/contracts", payload);
}

export function updateLaborContract(id: number, payload: LaborContractPayload): Promise<LaborContractRecord> {
  return request<LaborContractRecord>(`/contracts/${id}`, {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(payload),
  });
}

export function activateLaborContract(id: number): Promise<LaborContractRecord> {
  return post<LaborContractRecord>(`/contracts/${id}/activate`);
}

export function cancelLaborContract(id: number, reason: string): Promise<LaborContractRecord> {
  return post<LaborContractRecord>(`/contracts/${id}/cancel`, { reason });
}