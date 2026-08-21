"use client";

import { request } from "@/lib/api";

export type FleetVehicleStatus = "active" | "inactive";

export interface FleetVehicle {
  id: number;
  plate_number: string;
  vehicle_model: string;
  status: FleetVehicleStatus;
  brand?: string | null;
  seat_count?: number | null;
  purchase_date?: string | null;
  remarks?: string | null;
  created_at?: string;
  updated_at?: string;
}

export interface FleetVehiclePayload {
  plate_number: string;
  vehicle_model: string;
  status: FleetVehicleStatus;
  brand?: string | null;
  seat_count?: number | null;
  purchase_date?: string | null;
  remarks?: string | null;
}

const JSON_HEADERS = { "Content-Type": "application/json" };

export function fetchFleetVehicles(): Promise<FleetVehicle[]> {
  return request<FleetVehicle[]>("/fleet-vehicles");
}

export function createFleetVehicle(payload: FleetVehiclePayload): Promise<FleetVehicle> {
  return request<FleetVehicle>("/fleet-vehicles", { method: "POST", headers: JSON_HEADERS, body: JSON.stringify(payload) });
}

export function getFleetVehicle(id: number): Promise<FleetVehicle> {
  return request<FleetVehicle>(`/fleet-vehicles/${id}`);
}

export function updateFleetVehicle(id: number, payload: FleetVehiclePayload): Promise<FleetVehicle> {
  return request<FleetVehicle>(`/fleet-vehicles/${id}`, { method: "PUT", headers: JSON_HEADERS, body: JSON.stringify(payload) });
}

export function deleteFleetVehicle(id: number): Promise<void> {
  return request<void>(`/fleet-vehicles/${id}`, { method: "DELETE" });
}
