"use client";

import { request } from "@/lib/api";

export interface EnergyMetric {
  usage: number;
  amount: number;
  count: number;
}

export interface DormitoryEnergyOverall {
  electric: EnergyMetric;
  water: EnergyMetric;
  total_amount: number;
}

export interface DormitoryEnergyBuilding extends DormitoryEnergyOverall {
  building_id: number;
  building_name: string;
}

export interface DormitoryEnergyRoom extends DormitoryEnergyOverall {
  room_id: number;
  room_number: string;
  building_id: number;
}

export interface DormitoryEnergySummary {
  overall: DormitoryEnergyOverall;
  by_building: DormitoryEnergyBuilding[];
  rooms: DormitoryEnergyRoom[];
}

export interface DormitoryEnergyFilters {
  month?: string;
  buildingId?: number;
}

export function fetchDormitoryEnergySummary(filters: DormitoryEnergyFilters = {}): Promise<DormitoryEnergySummary> {
  const params = new URLSearchParams();
  if (filters.month) params.set("month", filters.month);
  if (filters.buildingId) params.set("building_id", String(filters.buildingId));
  const query = params.toString();
  return request<DormitoryEnergySummary>(`/dormitories/energy/summary${query ? `?${query}` : ""}`);
}
