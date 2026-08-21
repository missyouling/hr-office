"use client";

import { request } from "@/lib/api";

export type AdminContractStatus = "draft" | "active" | "expired" | "cancelled";

export interface AdminContractRecord {
  id: number;
  contract_no: string;
  name: string;
  counterparty: string;
  contract_type: string;
  start_date: string;
  end_date: string;
  amount_incl_tax: number | null;
  currency: string;
  owner: string;
  document_id: number | null;
  document?: { id: number; document_code: string; file_name: string } | null;
  status: AdminContractStatus;
  activated_at: string | null;
  expired_at: string | null;
  cancelled_at: string | null;
  cancel_reason: string;
  remarks: string;
  created_at: string;
  updated_at: string;
}

export interface AdminContractPayload {
  contract_no: string;
  name: string;
  counterparty: string;
  contract_type: string;
  start_date: string;
  end_date: string;
  amount_incl_tax?: number | null;
  currency?: string;
  owner?: string;
  document_id?: number | null;
  remarks?: string;
}

function post<T>(path: string, body?: object): Promise<T> {
  return request<T>(path, { method: "POST", headers: { "Content-Type": "application/json" }, body: body ? JSON.stringify(body) : undefined });
}

export function fetchAdminContracts(status?: AdminContractStatus): Promise<AdminContractRecord[]> {
  const query = status ? `?status=${encodeURIComponent(status)}` : "";
  return request<AdminContractRecord[]>(`/admin-contracts${query}`);
}

export function createAdminContract(payload: AdminContractPayload): Promise<AdminContractRecord> {
  return post<AdminContractRecord>("/admin-contracts", payload);
}

export function updateAdminContract(id: number, payload: AdminContractPayload): Promise<AdminContractRecord> {
  return request<AdminContractRecord>(`/admin-contracts/${id}`, { method: "PUT", headers: { "Content-Type": "application/json" }, body: JSON.stringify(payload) });
}

export function activateAdminContract(id: number): Promise<AdminContractRecord> {
  return post<AdminContractRecord>(`/admin-contracts/${id}/activate`);
}

export function cancelAdminContract(id: number, reason: string): Promise<AdminContractRecord> {
  return post<AdminContractRecord>(`/admin-contracts/${id}/cancel`, { reason });
}
