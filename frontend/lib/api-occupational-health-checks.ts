"use client";

import { request } from "@/lib/api";

export type OccupationalHealthCheckStatus = "draft" | "completed" | "voided";

export interface OccupationalHealthCheck {
  id: number;
  employee_id: number;
  employee_name: string;
  employee_department: string;
  employee_position: string;
  check_date: string;
  medical_institution: string;
  check_category: string;
  check_conclusion: string;
  next_check_date: string;
  remarks: string;
  status: OccupationalHealthCheckStatus;
  void_reason: string;
  created_at: string;
  updated_at: string;
}

export interface OccupationalHealthCheckPayload {
  employee_id: number;
  check_date: string;
  medical_institution: string;
  check_category: string;
  check_conclusion?: string;
  next_check_date?: string;
  remarks?: string;
}

function post<T>(path: string, body?: object): Promise<T> {
  return request<T>(path, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: body ? JSON.stringify(body) : undefined,
  });
}

export function fetchOccupationalHealthChecks(status?: OccupationalHealthCheckStatus): Promise<OccupationalHealthCheck[]> {
  const query = status ? `?status=${encodeURIComponent(status)}` : "";
  return request<OccupationalHealthCheck[]>(`/occupational-health-checks${query}`);
}

export function getOccupationalHealthCheck(id: number): Promise<OccupationalHealthCheck> {
  return request<OccupationalHealthCheck>(`/occupational-health-checks/${id}`);
}

export function createOccupationalHealthCheck(payload: OccupationalHealthCheckPayload): Promise<OccupationalHealthCheck> {
  return post<OccupationalHealthCheck>("/occupational-health-checks", payload);
}

export function updateOccupationalHealthCheck(id: number, payload: OccupationalHealthCheckPayload): Promise<OccupationalHealthCheck> {
  return request<OccupationalHealthCheck>(`/occupational-health-checks/${id}`, {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(payload),
  });
}

export function deleteOccupationalHealthCheck(id: number): Promise<void> {
  return request<void>(`/occupational-health-checks/${id}`, { method: "DELETE" });
}

export function completeOccupationalHealthCheck(id: number): Promise<OccupationalHealthCheck> {
  return post<OccupationalHealthCheck>(`/occupational-health-checks/${id}/complete`);
}

export function voidOccupationalHealthCheck(id: number, reason: string): Promise<OccupationalHealthCheck> {
  return post<OccupationalHealthCheck>(`/occupational-health-checks/${id}/void`, { reason });
}
