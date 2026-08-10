// 食堂管理模块 API 封装（直接 fetch，后续可迁移到 lib/api-canteen.ts）
"use client";

const BASE = "/api/canteen";

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

function toQuery(params?: Record<string, unknown>): string {
  if (!params) return "";
  const sp = new URLSearchParams();
  Object.entries(params).forEach(([k, v]) => {
    if (v !== undefined && v !== null && v !== "") sp.set(k, String(v));
  });
  return sp.toString();
}

export const canteenCategoriesApi = {
  list: () => request<{ items: any[] }>("/categories"),
  create: (data: any) => request<any>("/categories", { method: "POST", body: JSON.stringify(data) }),
  update: (id: number, data: any) => request<any>(`/categories/${id}`, { method: "PUT", body: JSON.stringify(data) }),
  delete: (id: number) => request<void>(`/categories/${id}`, { method: "DELETE" }),
};

export const canteenSuppliesApi = {
  list: (params?: Record<string, unknown>) => request<{ items: any[] }>(`/supplies?${toQuery(params)}`),
  create: (data: any) => request<any>("/supplies", { method: "POST", body: JSON.stringify(data) }),
  update: (id: number, data: any) => request<any>(`/supplies/${id}`, { method: "PUT", body: JSON.stringify(data) }),
  delete: (id: number) => request<void>(`/supplies/${id}`, { method: "DELETE" }),
};

export const expenseCategoriesApi = {
  list: () => request<{ items: any[] }>("/expense-categories"),
  create: (data: any) => request<any>("/expense-categories", { method: "POST", body: JSON.stringify(data) }),
  update: (id: number, data: any) => request<any>(`/expense-categories/${id}`, { method: "PUT", body: JSON.stringify(data) }),
  delete: (id: number) => request<void>(`/expense-categories/${id}`, { method: "DELETE" }),
};

export const canteenPurchasesApi = {
  list: (params?: Record<string, unknown>) => request<{ items: any[]; total?: number }>(`/purchases?${toQuery(params)}`),
  get: (id: number) => request<any>(`/purchases/${id}`),
  create: (data: any) => request<any>("/purchases", { method: "POST", body: JSON.stringify(data) }),
  update: (id: number, data: any) => request<any>(`/purchases/${id}`, { method: "PUT", body: JSON.stringify(data) }),
  delete: (id: number) => request<void>(`/purchases/${id}`, { method: "DELETE" }),
};

export const canteenExpensesApi = {
  list: (params?: Record<string, unknown>) => request<{ items: any[] }>(`/expenses?${toQuery(params)}`),
  upsert: (data: any) => request<any>("/expenses/upsert", { method: "POST", body: JSON.stringify(data) }),
};

export const canteenIncomeApi = {
  list: (params?: Record<string, unknown>) => request<{ items: any[] }>(`/income?${toQuery(params)}`),
  save: (data: any) => request<any>("/income", { method: "POST", body: JSON.stringify(data) }),
  delete: (id: number) => request<void>(`/income/${id}`, { method: "DELETE" }),
};

export const resourceFeesApi = {
  list: (params?: Record<string, unknown>) => request<{ items: any[] }>(`/resource-fees?${toQuery(params)}`),
  summary: (month: string) => request<any>(`/resource-fees/summary?month=${encodeURIComponent(month)}`),
  create: (data: any) => request<any>("/resource-fees", { method: "POST", body: JSON.stringify(data) }),
  update: (id: number, data: any) => request<any>(`/resource-fees/${id}`, { method: "PUT", body: JSON.stringify(data) }),
  delete: (id: number) => request<void>(`/resource-fees/${id}`, { method: "DELETE" }),
};

export const rechargesApi = {
  list: (params?: Record<string, unknown>) => request<{ items: any[]; total?: number }>(`/recharges?${toQuery(params)}`),
  summary: (month: string) => request<any>(`/recharges/summary?month=${encodeURIComponent(month)}`),
  importCsv: (data: any) => request<any>("/recharges/import", { method: "POST", body: JSON.stringify(data) }),
  delete: (id: number) => request<void>(`/recharges/${id}`, { method: "DELETE" }),
};

export const refundsApi = {
  list: (params?: Record<string, unknown>) => request<{ items: any[]; total?: number }>(`/refunds?${toQuery(params)}`),
  summary: (month: string) => request<any>(`/refunds/summary?month=${encodeURIComponent(month)}`),
  create: (data: any) => request<any>("/refunds", { method: "POST", body: JSON.stringify(data) }),
  update: (id: number, data: any) => request<any>(`/refunds/${id}`, { method: "PUT", body: JSON.stringify(data) }),
  delete: (id: number) => request<void>(`/refunds/${id}`, { method: "DELETE" }),
};

export const menusApi = {
  get: (weekStart: string) => request<any>(`/menus?week_start=${encodeURIComponent(weekStart)}`),
  save: (data: any) => request<any>("/menus", { method: "POST", body: JSON.stringify(data) }),
  copy: (from: string, to: string) => request<any>(`/menus/copy?from=${encodeURIComponent(from)}&to=${encodeURIComponent(to)}`, { method: "POST" }),
};

export const menuTemplatesApi = {
  list: () => request<{ items: any[] }>("/menu-templates"),
  create: (data: any) => request<any>("/menu-templates", { method: "POST", body: JSON.stringify(data) }),
  delete: (id: number) => request<void>(`/menu-templates/${id}`, { method: "DELETE" }),
};

export const canteenAnalyticsApi = {
  summary: (month: string) => request<any>(`/analytics/summary?month=${encodeURIComponent(month)}`),
  dailyTrend: (month: string) => request<any>(`/analytics/daily-trend?month=${encodeURIComponent(month)}`),
  expenseBreakdown: (month: string) => request<any>(`/analytics/expense-breakdown?month=${encodeURIComponent(month)}`),
  foodShare: (month: string) => request<any>(`/analytics/food-share?month=${encodeURIComponent(month)}`),
  topSupplies: (month: string, limit = 10) => request<any>(`/analytics/top-supplies?month=${encodeURIComponent(month)}&limit=${limit}`),
  costSummary: (params?: Record<string, unknown>) => request<any>(`/analytics/cost-summary?${toQuery(params)}`),
  monthlyCompare: (params: any) => request<any>(`/analytics/monthly-compare?${toQuery(params)}`),
};
