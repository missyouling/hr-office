"use client";

import { request } from "@/lib/api";

export type SafetyInspectionStatus = "draft" | "completed" | "voided";
export type SafetyInspectionType = "routine" | "special";

export interface SafetyInspectionRecord {
  id: number;
  inspection_type: SafetyInspectionType;
  inspection_date: string;
  location: string;
  responsible_person: string;
  issue_description: string;
  rectification_requirement: string;
  status: SafetyInspectionStatus;
  void_reason: string;
  created_at: string;
  updated_at: string;
}

export interface SafetyInspectionPayload {
  inspection_type: SafetyInspectionType;
  inspection_date: string;
  location: string;
  responsible_person: string;
  issue_description: string;
  rectification_requirement: string;
}

function post<T>(path: string, body?: object): Promise<T> {
  return request<T>(path, { method: "POST", headers: { "Content-Type": "application/json" }, body: body ? JSON.stringify(body) : undefined });
}

export function fetchSafetyInspections(status?: SafetyInspectionStatus): Promise<SafetyInspectionRecord[]> {
  const query = status ? `?status=${encodeURIComponent(status)}` : "";
  return request<SafetyInspectionRecord[]>(`/safety-inspections${query}`);
}
export function getSafetyInspection(id: number): Promise<SafetyInspectionRecord> { return request<SafetyInspectionRecord>(`/safety-inspections/${id}`); }
export function createSafetyInspection(payload: SafetyInspectionPayload): Promise<SafetyInspectionRecord> { return post<SafetyInspectionRecord>("/safety-inspections", payload); }
export function updateSafetyInspection(id: number, payload: SafetyInspectionPayload): Promise<SafetyInspectionRecord> { return request<SafetyInspectionRecord>(`/safety-inspections/${id}`, { method: "PUT", headers: { "Content-Type": "application/json" }, body: JSON.stringify(payload) }); }
export function deleteSafetyInspection(id: number): Promise<void> { return request<void>(`/safety-inspections/${id}`, { method: "DELETE" }); }
export function completeSafetyInspection(id: number): Promise<SafetyInspectionRecord> { return post<SafetyInspectionRecord>(`/safety-inspections/${id}/complete`); }
export function voidSafetyInspection(id: number, reason: string): Promise<SafetyInspectionRecord> { return post<SafetyInspectionRecord>(`/safety-inspections/${id}/void`, { reason }); }
