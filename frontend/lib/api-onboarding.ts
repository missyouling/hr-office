"use client";

import { request } from "@/lib/api";

export type OnboardingStatus = "pending" | "onboarded" | "abandoned";
export type EmploymentStatus = "trial" | "formal";

export interface OnboardingRecord {
  id: number;
  name: string;
  id_number: string;
  phone: string;
  department: string;
  position: string;
  planned_hire_date: string;
  actual_hire_date: string | null;
  status: OnboardingStatus;
  employment_status: EmploymentStatus | "";
  employee_id: number | null;
  abandon_reason: string;
  abandoned_at: string | null;
  remarks: string;
  offer_id: string;
  offer_source: string;
  created_at: string;
  updated_at: string;
}

export interface OnboardingPayload {
  name: string;
  id_number: string;
  phone: string;
  department: string;
  position: string;
  planned_hire_date: string;
  remarks: string;
  offer_id: string;
  offer_source: string;
}

const jsonOptions = (method: "POST" | "PUT", body?: object): RequestInit => ({
  method,
  headers: { "Content-Type": "application/json" },
  body: body ? JSON.stringify(body) : undefined,
});

export function fetchOnboardingRecords(status?: OnboardingStatus): Promise<OnboardingRecord[]> {
  const query = status ? `?status=${encodeURIComponent(status)}` : "";
  return request(`/onboarding-records${query}`);
}

export function createOnboardingRecord(payload: OnboardingPayload): Promise<OnboardingRecord> {
  return request("/onboarding-records", jsonOptions("POST", payload));
}

export function quickOnboardingRecord(payload: OnboardingPayload & { employment_status: EmploymentStatus }): Promise<OnboardingRecord> {
  return request("/onboarding-records/quick", jsonOptions("POST", payload));
}

export function updateOnboardingRecord(id: number, payload: OnboardingPayload): Promise<OnboardingRecord> {
  return request(`/onboarding-records/${id}`, jsonOptions("PUT", payload));
}

export function confirmOnboardingRecord(id: number, employmentStatus: EmploymentStatus): Promise<OnboardingRecord> {
  return request(`/onboarding-records/${id}/confirm`, jsonOptions("POST", { employment_status: employmentStatus }));
}

export function abandonOnboardingRecord(id: number, reason: string, remarks: string): Promise<OnboardingRecord> {
  return request(`/onboarding-records/${id}/abandon`, jsonOptions("POST", { reason, remarks }));
}

export function restoreOnboardingRecord(id: number): Promise<OnboardingRecord> {
  return request(`/onboarding-records/${id}/restore`, jsonOptions("POST"));
}
