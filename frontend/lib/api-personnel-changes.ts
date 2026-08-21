"use client";

import { request } from "@/lib/api";

export type PersonnelChangeType = "transfer" | "promotion" | "demotion";
export type PersonnelChangeStatus = "draft" | "effective" | "voided";

export interface PersonnelChangeRecord {
  id: number;
  employee_id: number;
  before_department: string;
  before_position: string;
  before_job_level: string;
  change_type: PersonnelChangeType;
  effective_date: string;
  reason: string;
  after_department: string;
  after_position: string;
  after_job_level: string;
  status: PersonnelChangeStatus;
  effective_at: string | null;
  voided_at: string | null;
  void_reason: string;
  created_at: string;
  updated_at: string;
}

export interface PersonnelChangePayload {
  employee_id: number;
  change_type: PersonnelChangeType;
  effective_date: string;
  reason: string;
  after_department_id?: number | null;
  after_position?: string;
  after_job_level?: string;
}

function post<T>(path: string, body?: object): Promise<T> { return request<T>(path, { method: "POST", headers: { "Content-Type": "application/json" }, body: body ? JSON.stringify(body) : undefined }); }
export function fetchPersonnelChanges(status?: PersonnelChangeStatus): Promise<PersonnelChangeRecord[]> { const query = status ? `?status=${encodeURIComponent(status)}` : ""; return request<PersonnelChangeRecord[]>(`/personnel-changes${query}`); }
export function getPersonnelChange(id: number): Promise<PersonnelChangeRecord> { return request<PersonnelChangeRecord>(`/personnel-changes/${id}`); }
export function createPersonnelChange(payload: PersonnelChangePayload): Promise<PersonnelChangeRecord> { return post<PersonnelChangeRecord>("/personnel-changes", payload); }
export function updatePersonnelChange(id: number, payload: PersonnelChangePayload): Promise<PersonnelChangeRecord> { return request<PersonnelChangeRecord>(`/personnel-changes/${id}`, { method: "PUT", headers: { "Content-Type": "application/json" }, body: JSON.stringify(payload) }); }
export function deletePersonnelChange(id: number): Promise<void> { return request<void>(`/personnel-changes/${id}`, { method: "DELETE" }); }
export function activatePersonnelChange(id: number): Promise<PersonnelChangeRecord> { return post<PersonnelChangeRecord>(`/personnel-changes/${id}/activate`); }
export function voidPersonnelChange(id: number, reason: string): Promise<PersonnelChangeRecord> { return post<PersonnelChangeRecord>(`/personnel-changes/${id}/void`, { reason }); }
