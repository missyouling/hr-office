"use client";

import { getRuntimeConfig } from "./runtime-config";

// ========== Base URL 解析 ==========

/** 从运行时配置获取 API 基础地址，去除尾部斜杠 */
function getApiBase(): string {
  const config = getRuntimeConfig();
  const base = config.API_BASE ?? process.env.NEXT_PUBLIC_API_BASE_URL;
  if (base) return base.replace(/\/+$/, "");
  if (typeof window !== "undefined") {
    return `${window.location.origin}/api`;
  }
  return "http://localhost:8081/api";
}

const API_BASE = getApiBase();

// ========== 通用响应类型 ==========

export interface ApiResponse<T = unknown> {
  data: T;
  message?: string;
  code?: number;
}

// ========== 业务类型定义 ==========

/** 办公用品分类 */
export interface OfficeCategory {
  id: number;
  name: string;
  description?: string;
  created_at?: string;
  updated_at?: string;
}

/** 供应商 */
export interface OfficeSupplier {
  id: number;
  name: string;
  contact_person?: string;
  phone?: string;
  email?: string;
  address?: string;
  notes?: string;
  created_at?: string;
  updated_at?: string;
}

/** 办公用品 */
export interface OfficeSupply {
  id: number;
  name: string;
  category_id?: number;
  unit?: string;
  current_stock?: number;
  min_stock?: number;
  price?: number;
  supplier_id?: number;
  description?: string;
  created_at?: string;
  updated_at?: string;
}

/** 采购单明细行 */
export interface OfficePurchaseItem {
  id?: number;
  purchase_id?: number;
  supply_id: number;
  supply_name?: string;
  quantity: number;
  unit_price: number;
  total_price?: number;
  unit?: string;
}

/** 采购单 */
export interface OfficePurchase {
  id: number;
  supplier_id?: number;
  supplier_name?: string;
  purchase_date?: string;
  total_amount?: number;
  status?: string;
  notes?: string;
  items?: OfficePurchaseItem[];
  created_at?: string;
  updated_at?: string;
}

/** 请款单 */
export interface OfficePaymentRequest {
  id: number;
  purchase_id?: number;
  amount: number;
  reason?: string;
  payee?: string;
  status?: string;
  payment_date?: string;
  notes?: string;
  created_at?: string;
  updated_at?: string;
}

// ========== 内部请求工具 ==========

/** JSON 请求：自动注入 JWT，失败抛 Error */
async function request<T>(path: string, init?: RequestInit, expectJson = true): Promise<T> {
  const url = `${API_BASE}${path}`;
  const token = localStorage.getItem("token");
  const authHeaders: Record<string, string> = token ? { Authorization: `Bearer ${token}` } : {};

  const res = await fetch(url, {
    ...init,
    headers: { ...authHeaders, ...((init?.headers as Record<string, string>) || {}) },
    cache: "no-store",
  });

  if (!res.ok) {
    let detail = "";
    try {
      const body = await res.json();
      detail = body?.error || JSON.stringify(body);
    } catch {
      detail = res.statusText;
    }
    throw new Error(`[${res.status}] ${detail || "请求失败"}`);
  }

  if (!expectJson) return undefined as T;
  return (await res.json()) as T;
}

/** 二进制下载：返回 Blob */
async function downloadBlob(path: string, init?: RequestInit): Promise<Blob> {
  const url = `${API_BASE}${path}`;
  const token = localStorage.getItem("token");
  const headers = new Headers((init?.headers as Record<string, string>) || {});
  if (token && !headers.has("Authorization")) {
    headers.set("Authorization", `Bearer ${token}`);
  }

  const res = await fetch(url, { ...init, headers });
  if (!res.ok) {
    let detail = res.statusText;
    try {
      const body = await res.json();
      detail = body?.error || body?.details || detail;
    } catch {
      // 忽略 JSON 解析失败
    }
    throw new Error(detail || "下载失败");
  }
  return res.blob();
}

/** 将原始响应包装为 ApiResponse */
function wrap<T>(promise: Promise<T>): Promise<ApiResponse<T>> {
  return promise.then((data) => ({ data }));
}

// ========== 办公用品 API ==========

export const officeApi = {
  /** 分类管理 */
  categories: {
    list: () => wrap(request<OfficeCategory[]>("/office/categories")),
    create: (payload: Partial<OfficeCategory>) =>
      wrap(request<OfficeCategory>("/office/categories", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(payload),
      })),
    update: (id: number, payload: Partial<OfficeCategory>) =>
      wrap(request<OfficeCategory>(`/office/categories/${id}`, {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(payload),
      })),
    remove: (id: number) =>
      wrap(request<void>(`/office/categories/${id}`, { method: "DELETE" }, false)),
  },

  /** 供应商管理 */
  suppliers: {
    list: () => wrap(request<OfficeSupplier[]>("/office/suppliers")),
    create: (payload: Partial<OfficeSupplier>) =>
      wrap(request<OfficeSupplier>("/office/suppliers", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(payload),
      })),
    update: (id: number, payload: Partial<OfficeSupplier>) =>
      wrap(request<OfficeSupplier>(`/office/suppliers/${id}`, {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(payload),
      })),
    remove: (id: number) =>
      wrap(request<void>(`/office/suppliers/${id}`, { method: "DELETE" }, false)),
  },

  /** 用品管理 */
  supplies: {
    list: (params?: Record<string, string>) => {
      const qs = params ? `?${new URLSearchParams(params).toString()}` : "";
      return wrap(request<OfficeSupply[]>(`/office/supplies${qs}`));
    },
    get: (id: number) => wrap(request<OfficeSupply>(`/office/supplies/${id}`)),
    create: (payload: Partial<OfficeSupply>) =>
      wrap(request<OfficeSupply>("/office/supplies", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(payload),
      })),
    update: (id: number, payload: Partial<OfficeSupply>) =>
      wrap(request<OfficeSupply>(`/office/supplies/${id}`, {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(payload),
      })),
    remove: (id: number) =>
      wrap(request<void>(`/office/supplies/${id}`, { method: "DELETE" }, false)),
    /** 获取所有计量单位 */
    getUnits: () => wrap(request<string[]>("/office/supplies/units")),
    /** 导出用品列表 (CSV) */
    export: () => downloadBlob("/office/supplies/export"),
    /** 从 CSV 文件批量导入用品 */
    import: (file: File) => {
      const formData = new FormData();
      formData.append("file", file);
      return wrap(request<{ imported: number }>("/office/supplies/import", {
        method: "POST",
        body: formData,
      }));
    },
  },

  /** 采购单管理 */
  purchases: {
    list: (params?: Record<string, string>) => {
      const qs = params ? `?${new URLSearchParams(params).toString()}` : "";
      return wrap(request<OfficePurchase[]>(`/office/purchases${qs}`));
    },
    get: (id: number) => wrap(request<OfficePurchase>(`/office/purchases/${id}`)),
    create: (payload: Partial<OfficePurchase>) =>
      wrap(request<OfficePurchase>("/office/purchases", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(payload),
      })),
    update: (id: number, payload: Partial<OfficePurchase>) =>
      wrap(request<OfficePurchase>(`/office/purchases/${id}`, {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(payload),
      })),
    remove: (id: number) =>
      wrap(request<void>(`/office/purchases/${id}`, { method: "DELETE" }, false)),
    /** 获取未付款采购单列表 */
    unpaid: () => wrap(request<OfficePurchase[]>("/office/purchases/unpaid")),
    /** 导出采购单 (CSV) */
    export: () => downloadBlob("/office/purchases/export"),
    /** 复制采购单 */
    copy: (id: number) =>
      wrap(request<OfficePurchase>(`/office/purchases/${id}/copy`, { method: "POST" })),
    /** 下载采购单 PDF */
    pdf: (id: number) => downloadBlob(`/office/purchases/${id}/pdf`),
    /** 下载采购单 Excel */
    excel: (id: number) => downloadBlob(`/office/purchases/${id}/excel`),
  },

  /** 请款单管理 */
  paymentRequests: {
    list: (params?: Record<string, string>) => {
      const qs = params ? `?${new URLSearchParams(params).toString()}` : "";
      return wrap(request<OfficePaymentRequest[]>(`/office/payment-requests${qs}`));
    },
    get: (id: number) => wrap(request<OfficePaymentRequest>(`/office/payment-requests/${id}`)),
    create: (payload: Partial<OfficePaymentRequest>) =>
      wrap(request<OfficePaymentRequest>("/office/payment-requests", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(payload),
      })),
    update: (id: number, payload: Partial<OfficePaymentRequest>) =>
      wrap(request<OfficePaymentRequest>(`/office/payment-requests/${id}`, {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(payload),
      })),
    remove: (id: number) =>
      wrap(request<void>(`/office/payment-requests/${id}`, { method: "DELETE" }, false)),
  },

  /** 分析报表 */
  analytics: {
    summary: (params?: Record<string, string>) => {
      const qs = params ? `?${new URLSearchParams(params).toString()}` : "";
      return wrap(request<Record<string, unknown>>(`/office/analytics/summary${qs}`));
    },
    categoryTrend: (params?: Record<string, string>) => {
      const qs = params ? `?${new URLSearchParams(params).toString()}` : "";
      return wrap(request<Record<string, unknown>>(`/office/analytics/category-trend${qs}`));
    },
    frequency: (params?: Record<string, string>) => {
      const qs = params ? `?${new URLSearchParams(params).toString()}` : "";
      return wrap(request<Record<string, unknown>>(`/office/analytics/frequency${qs}`));
    },
    topItems: (params?: Record<string, string>) => {
      const qs = params ? `?${new URLSearchParams(params).toString()}` : "";
      return wrap(request<Record<string, unknown>>(`/office/analytics/top-items${qs}`));
    },
    priceAnomaly: (params?: Record<string, string>) => {
      const qs = params ? `?${new URLSearchParams(params).toString()}` : "";
      return wrap(request<Record<string, unknown>>(`/office/analytics/price-anomaly${qs}`));
    },
    suggestions: (params?: Record<string, string>) => {
      const qs = params ? `?${new URLSearchParams(params).toString()}` : "";
      return wrap(request<Record<string, unknown>>(`/office/analytics/suggestions${qs}`));
    },
    trend: (params?: Record<string, string>) => {
      const qs = params ? `?${new URLSearchParams(params).toString()}` : "";
      return wrap(request<Record<string, unknown>>(`/office/analytics/trend${qs}`));
    },
    /** 生成分析报告 PDF */
    reportPdf: (payload?: Record<string, unknown>) =>
      downloadBlob("/office/analytics/report-pdf", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: payload ? JSON.stringify(payload) : undefined,
      }),
  },

  /** 系统工具 */
  system: {
    /** 重置办公用品模块数据 */
    reset: () =>
      wrap(request<{ message: string }>("/office/system/reset", { method: "POST" })),
    /** 获取备份列表 */
    getBackups: () => wrap(request<{ id: number; created_at: string; filename: string }[]>("/office/system/backups")),
    /** 创建备份 */
    createBackup: () =>
      wrap(request<{ id: number; filename: string }>("/office/system/backups", { method: "POST" })),
    /** 从备份恢复 */
    restoreBackup: (id: number) =>
      wrap(request<{ message: string }>(`/office/system/backups/${id}/restore`, { method: "POST" })),
    /** 删除备份 */
    removeBackup: (id: number) =>
      wrap(request<void>(`/office/system/backups/${id}`, { method: "DELETE" }, false)),
  },
};
