// 办公劳保模块 API 封装（直接 fetch，后续可迁移到 lib/api-office.ts）
"use client";

import { useAuth } from "@/lib/auth";

const BASE = "/api/office";

export class ApiError extends Error {
  constructor(public status: number, message: string) {
    super(message);
    this.name = "ApiError";
  }
}

async function getToken(): Promise<string | null> {
  if (typeof window === "undefined") return null;
  return localStorage.getItem("token");
}

async function request<T>(path: string, options: RequestInit = {}): Promise<T> {
  const token = await getToken();
  const headers: Record<string, string> = {
    "Content-Type": "application/json",
    ...(token ? { Authorization: `Bearer ${token}` } : {}),
    ...(options.headers as Record<string, string> || {}),
  };
  const res = await fetch(`${BASE}${path}`, { ...options, headers });
  if (!res.ok) {
    const text = await res.text().catch(() => "请求失败");
    throw new ApiError(res.status, text || `HTTP ${res.status}`);
  }
  if (res.status === 204) return undefined as T;
  return res.json() as Promise<T>;
}

// 用品字典
export const suppliesApi = {
  list: (params?: Record<string, unknown>) => request<{ items: any[]; total?: number }>(`/supplies?${toQuery(params)}`),
  get: (id: number) => request<any>(`/supplies/${id}`),
  create: (data: any) => request<any>("/supplies", { method: "POST", body: JSON.stringify(data) }),
  update: (id: number, data: any) => request<any>(`/supplies/${id}`, { method: "PUT", body: JSON.stringify(data) }),
  delete: (id: number) => request<void>(`/supplies/${id}`, { method: "DELETE" }),
  listUnits: () => request<{ units: string[] }>("/supplies/units"),
  importCsv: (csvText: string) => request<{ ok: number; err: number }>("/supplies/import", { method: "POST", body: JSON.stringify({ csv: csvText }) }),
  exportCsv: async () => {
    const token = await getToken();
    const res = await fetch(`${BASE}/supplies/export`, { headers: { Authorization: `Bearer ${token || ""}` } });
    if (!res.ok) throw new ApiError(res.status, "导出失败");
    return res.blob();
  },
};

// 用品分类
export const categoriesApi = {
  list: () => request<{ items: any[] }>("/categories"),
  create: (data: any) => request<any>("/categories", { method: "POST", body: JSON.stringify(data) }),
  update: (id: number, data: any) => request<any>(`/categories/${id}`, { method: "PUT", body: JSON.stringify(data) }),
  delete: (id: number) => request<void>(`/categories/${id}`, { method: "DELETE" }),
};

// 供应商
export const suppliersApi = {
  list: () => request<{ items: any[] }>("/suppliers"),
  create: (data: any) => request<any>("/suppliers", { method: "POST", body: JSON.stringify(data) }),
  update: (id: number, data: any) => request<any>(`/suppliers/${id}`, { method: "PUT", body: JSON.stringify(data) }),
  delete: (id: number) => request<void>(`/suppliers/${id}`, { method: "DELETE" }),
};

// 采购单
export const purchasesApi = {
  list: (params?: Record<string, unknown>) => request<{ items: any[]; total?: number; total_sum?: number; min_date?: string; max_date?: string }>(`/purchases?${toQuery(params)}`),
  get: (id: number) => request<any>(`/purchases/${id}`),
  create: (data: any) => request<any>("/purchases", { method: "POST", body: JSON.stringify(data) }),
  update: (id: number, data: any) => request<any>(`/purchases/${id}`, { method: "PUT", body: JSON.stringify(data) }),
  delete: (id: number) => request<void>(`/purchases/${id}`, { method: "DELETE" }),
  unpaid: () => request<{ items: any[] }>("/purchases/unpaid"),
  searchBySupply: (keyword: string) => request<{ items: any[] }>(`/purchases/search-by-supply?keyword=${encodeURIComponent(keyword)}`),
};

// 请款单
export const paymentRequestsApi = {
  list: (params?: Record<string, unknown>) => request<{ items: any[]; total?: number }>(`/payment-requests?${toQuery(params)}`),
  get: (id: number) => request<any>(`/payment-requests/${id}`),
  create: (data: any) => request<any>("/payment-requests", { method: "POST", body: JSON.stringify(data) }),
  update: (id: number, data: any) => request<any>(`/payment-requests/${id}`, { method: "PUT", body: JSON.stringify(data) }),
  delete: (id: number) => request<void>(`/payment-requests/${id}`, { method: "DELETE" }),
};

// 数据分析
export const analyticsApi = {
  summary: (params: any) => request<any>(`/analytics/summary?${toQuery(params)}`),
  categoryTrend: (params: any) => request<any>(`/analytics/category-trend?${toQuery(params)}`),
  trend: (params: any) => request<any>(`/analytics/trend?${toQuery(params)}`),
  topItems: (params: any) => request<any>(`/analytics/top-items?${toQuery(params)}`),
  priceAnomaly: (params: any) => request<any>(`/analytics/price-anomaly?${toQuery(params)}`),
  suggestions: (params: any) => request<any>(`/analytics/suggestions?${toQuery(params)}`),
};

function toQuery(params?: Record<string, unknown>): string {
  if (!params) return "";
  const sp = new URLSearchParams();
  Object.entries(params).forEach(([k, v]) => {
    if (v !== undefined && v !== null && v !== "") sp.set(k, String(v));
  });
  return sp.toString();
}
