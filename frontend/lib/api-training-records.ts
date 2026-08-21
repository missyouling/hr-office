"use client";

import { request } from "@/lib/api";

export type TrainingType = "internal" | "external" | "online";
export type TrainingStatus = "draft" | "completed" | "voided";

export interface TrainingRecord {
  id: number;
  topic: string;
  training_type: TrainingType;
  training_date: string;
  trainer_or_institution: string;
  employee_id: number | null;
  snapshot_name: string;
  snapshot_department: string;
  snapshot_position: string;
  result: string;
  remarks: string;
  status: TrainingStatus;
  void_reason: string;
  created_at: string;
  updated_at: string;
}

export interface TrainingPayload {
  topic: string;
  training_type: TrainingType;
  training_date: string;
  trainer_or_institution?: string;
  employee_id?: number | null;
  result?: string;
  remarks?: string;
}

function post<T>(path: string, body?: object): Promise<T> {
  return request<T>(path, { method: "POST", headers: { "Content-Type": "application/json" }, body: body ? JSON.stringify(body) : undefined });
}

export function fetchTrainingRecords(status?: TrainingStatus): Promise<TrainingRecord[]> {
  const query = status ? `?status=${encodeURIComponent(status)}` : "";
  return request<TrainingRecord[]>(`/training-records${query}`);
}
export function getTrainingRecord(id: number): Promise<TrainingRecord> { return request<TrainingRecord>(`/training-records/${id}`); }
export function createTrainingRecord(payload: TrainingPayload): Promise<TrainingRecord> { return post<TrainingRecord>("/training-records", payload); }
export function updateTrainingRecord(id: number, payload: TrainingPayload): Promise<TrainingRecord> { return request<TrainingRecord>(`/training-records/${id}`, { method: "PUT", headers: { "Content-Type": "application/json" }, body: JSON.stringify(payload) }); }
export function deleteTrainingRecord(id: number): Promise<void> { return request<void>(`/training-records/${id}`, { method: "DELETE" }); }
export function completeTrainingRecord(id: number): Promise<TrainingRecord> { return post<TrainingRecord>(`/training-records/${id}/complete`); }
export function voidTrainingRecord(id: number, reason: string): Promise<TrainingRecord> { return post<TrainingRecord>(`/training-records/${id}/void`, { reason }); }
