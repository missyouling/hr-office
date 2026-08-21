"use client";

import { request } from "@/lib/api";

export type RewardType = "reward" | "punishment";
export type RewardStatus = "draft" | "effective" | "voided";

export interface RewardRecord {
  id: number;
  employee_id: number;
  snapshot_name: string;
  snapshot_department: string;
  snapshot_position: string;
  record_type: RewardType;
  occurred_date: string;
  reason: string;
  level: string;
  score: number | null;
  amount: number | null;
  owner: string;
  document_id: number | null;
  document?: { id: number; document_code: string; file_name: string } | null;
  remarks: string;
  status: RewardStatus;
  effective_at: string | null;
  voided_at: string | null;
  void_reason: string;
  created_at: string;
  updated_at: string;
}

export interface RewardPayload {
  employee_id: number;
  record_type: RewardType;
  occurred_date: string;
  reason: string;
  level: string;
  score?: number | null;
  amount?: number | null;
  owner?: string;
  document_id?: number | null;
  remarks?: string;
}

function post<T>(path: string, body?: object): Promise<T> {
  return request<T>(path, { method: "POST", headers: { "Content-Type": "application/json" }, body: body ? JSON.stringify(body) : undefined });
}

export function fetchRewards(status?: RewardStatus): Promise<RewardRecord[]> {
  const query = status ? `?status=${encodeURIComponent(status)}` : "";
  return request<RewardRecord[]>(`/rewards${query}`);
}

export function createReward(payload: RewardPayload): Promise<RewardRecord> { return post<RewardRecord>("/rewards", payload); }
export function updateReward(id: number, payload: RewardPayload): Promise<RewardRecord> {
  return request<RewardRecord>(`/rewards/${id}`, { method: "PUT", headers: { "Content-Type": "application/json" }, body: JSON.stringify(payload) });
}
export function makeRewardEffective(id: number): Promise<RewardRecord> { return post<RewardRecord>(`/rewards/${id}/activate`); }
export function voidReward(id: number, reason: string): Promise<RewardRecord> { return post<RewardRecord>(`/rewards/${id}/void`, { reason }); }
