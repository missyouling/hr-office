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

/** 食材分类 */
export interface CanteenCategory {
  id: number;
  name: string;
  description?: string;
  created_at?: string;
  updated_at?: string;
}

/** 食材字典 */
export interface CanteenSupply {
  id: number;
  name: string;
  category_id?: number;
  category_name?: string;
  unit?: string;
  unit_price?: number;
  description?: string;
  created_at?: string;
  updated_at?: string;
}

/** 费用科目 */
export interface CanteenExpenseCategory {
  id: number;
  name: string;
  description?: string;
  created_at?: string;
  updated_at?: string;
}

/** 食堂采购单明细行 */
export interface CanteenPurchaseItem {
  id?: number;
  purchase_id?: number;
  supply_id: number;
  supply_name?: string;
  quantity: number;
  unit_price: number;
  total_price?: number;
  unit?: string;
}

/** 食堂采购单 */
export interface CanteenPurchase {
  id: number;
  purchase_date?: string;
  total_amount?: number;
  status?: string;
  notes?: string;
  items?: CanteenPurchaseItem[];
  created_at?: string;
  updated_at?: string;
}

/** 其他费用 */
export interface CanteenOtherExpense {
  id: number;
  expense_category_id?: number;
  category_name?: string;
  amount: number;
  expense_date?: string;
  description?: string;
  notes?: string;
  created_at?: string;
  updated_at?: string;
}

/** 每日收入 */
export interface CanteenDailyIncome {
  id: number;
  income_date: string;
  amount: number;
  note?: string;
  created_at?: string;
  updated_at?: string;
}

/** 资源占用费 */
export interface CanteenResourceFee {
  id: number;
  month: string;
  amount: number;
  description?: string;
  created_at?: string;
  updated_at?: string;
}

/** 每周菜单（一天的菜单项） */
export interface CanteenMenuDay {
  day_of_week: number;
  breakfast?: string;
  lunch?: string;
  dinner?: string;
}

/** 每周菜单 */
export interface CanteenWeeklyMenu {
  id?: number;
  week_start: string;
  days: CanteenMenuDay[];
  created_at?: string;
  updated_at?: string;
}

/** 菜单模板 */
export interface CanteenMenuTemplate {
  id: number;
  name: string;
  days: CanteenMenuDay[];
  created_at?: string;
  updated_at?: string;
}

/** 饭卡充值记录 */
export interface CanteenCardRecharge {
  id: number;
  employee_name?: string;
  employee_id?: string;
  amount: number;
  recharge_date?: string;
  note?: string;
  created_at?: string;
}

/** 饭卡退费记录 */
export interface CanteenCardRefund {
  id: number;
  employee_name?: string;
  employee_id?: string;
  amount: number;
  refund_date?: string;
  note?: string;
  created_at?: string;
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

/** 拼接查询参数 */
function withQs(base: string, params?: Record<string, string>): string {
  if (!params) return base;
  const qs = new URLSearchParams(params).toString();
  return qs ? `${base}?${qs}` : base;
}

// ========== 食堂管理 API ==========

export const canteenApi = {
  /** 食材分类管理 */
  categories: {
    list: () => wrap(request<CanteenCategory[]>("/canteen/categories")),
    create: (payload: Partial<CanteenCategory>) =>
      wrap(request<CanteenCategory>("/canteen/categories", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(payload),
      })),
    update: (id: number, payload: Partial<CanteenCategory>) =>
      wrap(request<CanteenCategory>(`/canteen/categories/${id}`, {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(payload),
      })),
    remove: (id: number) =>
      wrap(request<void>(`/canteen/categories/${id}`, { method: "DELETE" }, false)),
  },

  /** 食材字典管理 */
  supplies: {
    list: (params?: Record<string, string>) =>
      wrap(request<CanteenSupply[]>(withQs("/canteen/supplies", params))),
    /** 获取全部食材（不分页） */
    all: () => wrap(request<CanteenSupply[]>("/canteen/supplies/all")),
    get: (id: number) => wrap(request<CanteenSupply>(`/canteen/supplies/${id}`)),
    create: (payload: Partial<CanteenSupply>) =>
      wrap(request<CanteenSupply>("/canteen/supplies", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(payload),
      })),
    update: (id: number, payload: Partial<CanteenSupply>) =>
      wrap(request<CanteenSupply>(`/canteen/supplies/${id}`, {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(payload),
      })),
    remove: (id: number) =>
      wrap(request<void>(`/canteen/supplies/${id}`, { method: "DELETE" }, false)),
  },

  /** 费用科目管理 */
  expenseCategories: {
    list: () => wrap(request<CanteenExpenseCategory[]>("/canteen/expense-categories")),
    create: (payload: Partial<CanteenExpenseCategory>) =>
      wrap(request<CanteenExpenseCategory>("/canteen/expense-categories", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(payload),
      })),
    update: (id: number, payload: Partial<CanteenExpenseCategory>) =>
      wrap(request<CanteenExpenseCategory>(`/canteen/expense-categories/${id}`, {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(payload),
      })),
    remove: (id: number) =>
      wrap(request<void>(`/canteen/expense-categories/${id}`, { method: "DELETE" }, false)),
  },

  /** 采购单管理 */
  purchases: {
    list: (params?: Record<string, string>) =>
      wrap(request<CanteenPurchase[]>(withQs("/canteen/purchases", params))),
    get: (id: number) => wrap(request<CanteenPurchase>(`/canteen/purchases/${id}`)),
    create: (payload: Partial<CanteenPurchase>) =>
      wrap(request<CanteenPurchase>("/canteen/purchases", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(payload),
      })),
    update: (id: number, payload: Partial<CanteenPurchase>) =>
      wrap(request<CanteenPurchase>(`/canteen/purchases/${id}`, {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(payload),
      })),
    remove: (id: number) =>
      wrap(request<void>(`/canteen/purchases/${id}`, { method: "DELETE" }, false)),
    /** 导出采购单 CSV */
    exportCsv: () => downloadBlob("/canteen/purchases/export/csv"),
  },

  /** 其他费用管理 */
  expenses: {
    list: (params?: Record<string, string>) =>
      wrap(request<CanteenOtherExpense[]>(withQs("/canteen/expenses", params))),
    create: (payload: Partial<CanteenOtherExpense>) =>
      wrap(request<CanteenOtherExpense>("/canteen/expenses", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(payload),
      })),
    update: (id: number, payload: Partial<CanteenOtherExpense>) =>
      wrap(request<CanteenOtherExpense>(`/canteen/expenses/${id}`, {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(payload),
      })),
    remove: (id: number) =>
      wrap(request<void>(`/canteen/expenses/${id}`, { method: "DELETE" }, false)),
    /** 更新或创建一条费用记录（有 id 则更新，无则新建） */
    upsert: (payload: Partial<CanteenOtherExpense>) =>
      wrap(request<CanteenOtherExpense>("/canteen/expenses/upsert", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(payload),
      })),
  },

  /** 每日收入管理 */
  income: {
    list: (params?: Record<string, string>) =>
      wrap(request<CanteenDailyIncome[]>(withQs("/canteen/income", params))),
    create: (payload: Partial<CanteenDailyIncome>) =>
      wrap(request<CanteenDailyIncome>("/canteen/income", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(payload),
      })),
    update: (id: number, payload: Partial<CanteenDailyIncome>) =>
      wrap(request<CanteenDailyIncome>(`/canteen/income/${id}`, {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(payload),
      })),
    remove: (id: number) =>
      wrap(request<void>(`/canteen/income/${id}`, { method: "DELETE" }, false)),
  },

  /** 资源占用费管理 */
  resourceFees: {
    list: (params?: Record<string, string>) =>
      wrap(request<CanteenResourceFee[]>(withQs("/canteen/resource-fees", params))),
    create: (payload: Partial<CanteenResourceFee>) =>
      wrap(request<CanteenResourceFee>("/canteen/resource-fees", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(payload),
      })),
    update: (id: number, payload: Partial<CanteenResourceFee>) =>
      wrap(request<CanteenResourceFee>(`/canteen/resource-fees/${id}`, {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(payload),
      })),
    remove: (id: number) =>
      wrap(request<void>(`/canteen/resource-fees/${id}`, { method: "DELETE" }, false)),
    /** 获取指定月份的资源占用费汇总 */
    summary: (month: string) =>
      wrap(request<Record<string, unknown>>(`/canteen/resource-fees/summary/${month}`)),
  },

  /** 每周菜单管理 */
  menus: {
    /** 按周获取菜单，需传入 week 参数（如 "2025-01-06"） */
    getByWeek: (week: string) =>
      wrap(request<CanteenWeeklyMenu>(`/canteen/menus?week=${encodeURIComponent(week)}`)),
    create: (payload: Partial<CanteenWeeklyMenu>) =>
      wrap(request<CanteenWeeklyMenu>("/canteen/menus", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(payload),
      })),
    /** 从指定周复制菜单到目标周 */
    copy: (payload: { source_week: string; target_week: string }) =>
      wrap(request<CanteenWeeklyMenu>("/canteen/menus/copy", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(payload),
      })),
  },

  /** 菜单模板管理 */
  menuTemplates: {
    list: () => wrap(request<CanteenMenuTemplate[]>("/canteen/menu-templates")),
    create: (payload: Partial<CanteenMenuTemplate>) =>
      wrap(request<CanteenMenuTemplate>("/canteen/menu-templates", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(payload),
      })),
    remove: (id: number) =>
      wrap(request<void>(`/canteen/menu-templates/${id}`, { method: "DELETE" }, false)),
  },

  /** 饭卡充值管理 */
  recharges: {
    list: (params?: Record<string, string>) =>
      wrap(request<CanteenCardRecharge[]>(withQs("/canteen/recharges", params))),
    remove: (id: number) =>
      wrap(request<void>(`/canteen/recharges/${id}`, { method: "DELETE" }, false)),
    /** 充值汇总统计 */
    summary: (params?: Record<string, string>) =>
      wrap(request<Record<string, unknown>>(withQs("/canteen/recharges/summary", params))),
    /** 从 Excel/CSV 文件批量导入充值记录 */
    import: (file: File) => {
      const formData = new FormData();
      formData.append("file", file);
      return wrap(request<{ imported: number }>("/canteen/recharges/import", {
        method: "POST",
        body: formData,
      }));
    },
  },

  /** 饭卡退费管理 */
  refunds: {
    list: (params?: Record<string, string>) =>
      wrap(request<CanteenCardRefund[]>(withQs("/canteen/refunds", params))),
    remove: (id: number) =>
      wrap(request<void>(`/canteen/refunds/${id}`, { method: "DELETE" }, false)),
    /** 退费汇总统计 */
    summary: (params?: Record<string, string>) =>
      wrap(request<Record<string, unknown>>(withQs("/canteen/refunds/summary", params))),
    /** 从 Excel/CSV 文件批量导入退费记录 */
    import: (file: File) => {
      const formData = new FormData();
      formData.append("file", file);
      return wrap(request<{ imported: number }>("/canteen/refunds/import", {
        method: "POST",
        body: formData,
      }));
    },
  },

  /** 分析报表 */
  analytics: {
    summary: (params?: Record<string, string>) =>
      wrap(request<Record<string, unknown>>(withQs("/canteen/analytics/summary", params))),
    dailyTrend: (params?: Record<string, string>) =>
      wrap(request<Record<string, unknown>>(withQs("/canteen/analytics/daily-trend", params))),
    expenseBreakdown: (params?: Record<string, string>) =>
      wrap(request<Record<string, unknown>>(withQs("/canteen/analytics/expense-breakdown", params))),
    foodShare: (params?: Record<string, string>) =>
      wrap(request<Record<string, unknown>>(withQs("/canteen/analytics/food-share", params))),
    topSupplies: (params?: Record<string, string>) =>
      wrap(request<Record<string, unknown>>(withQs("/canteen/analytics/top-supplies", params))),
    monthlyCompare: (params?: Record<string, string>) =>
      wrap(request<Record<string, unknown>>(withQs("/canteen/analytics/monthly-compare", params))),
    suggestions: (params?: Record<string, string>) =>
      wrap(request<Record<string, unknown>>(withQs("/canteen/analytics/suggestions", params))),
    costSummary: (params?: Record<string, string>) =>
      wrap(request<Record<string, unknown>>(withQs("/canteen/analytics/cost-summary", params))),
    /** 按日期范围查询成本汇总 */
    costSummaryRange: (params: { start_date: string; end_date: string }) =>
      wrap(request<Record<string, unknown>>(
        withQs("/canteen/analytics/cost-summary-range", params),
      )),
  },
};
