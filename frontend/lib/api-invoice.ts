"use client";

import { getRuntimeConfig } from "./runtime-config";

// ========== Base URL 解析 ==========

/** 从运行时配置获取 API 基础地址 */
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

// ========== 业务类型定义 ==========

/** 发票 */
export interface Invoice {
  id: number;
  user_id?: number | null;
  invoice_no: string;
  invoice_date: string;
  invoice_type?: string;
  amount: number;
  tax_amount: number;
  total_amount: number;
  seller: string;
  seller_tax_no?: string;
  buyer?: string;
  purpose?: string;
  remark?: string;
  attachment_url?: string;
  source_type?: string;
  source_id?: number | null;
  applicant_id?: number | null;
  reimburse_amount: number;
  status: "draft" | "submitted" | "approved" | "rejected" | "reimbursed";
  approver_id?: number | null;
  approved_at?: string | null;
  approval_remark?: string;
  created_at: string;
  updated_at: string;
}

/** 发票列表响应 */
export interface InvoiceListResponse {
  items: Invoice[];
  total: number;
  page: number;
  page_size: number;
}

/** 发票统计 */
export interface InvoiceStats {
  total: number;
  total_amount: number;
  by_status: Record<string, number>;
  by_source: Record<string, number>;
}

/** 发票列表查询参数 */
export interface InvoiceListParams {
  status?: string;
  source_type?: string;
  keyword?: string;
  start_date?: string;
  end_date?: string;
  applicant_id?: number;
  page?: number;
  page_size?: number;
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

// ========== 发票 API ==========

export const invoiceApi = {
  /** 分页查询发票列表（manager+） */
  list: (params?: InvoiceListParams): Promise<InvoiceListResponse> => {
    const searchParams = new URLSearchParams();
    if (params) {
      Object.entries(params).forEach(([key, value]) => {
        if (value !== undefined && value !== null) {
          searchParams.set(key, String(value));
        }
      });
    }
    const qs = searchParams.toString() ? `?${searchParams.toString()}` : "";
    return request<InvoiceListResponse>(`/invoices${qs}`);
  },

  /** 获取单张发票详情 */
  get: (id: number): Promise<Invoice> => {
    return request<Invoice>(`/invoices/${id}`);
  },

  /** 创建发票草稿 */
  create: (payload: Partial<Invoice>): Promise<Invoice> => {
    return request<Invoice>("/invoices", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(payload),
    });
  },

  /** 更新发票（仅草稿） */
  update: (id: number, payload: Partial<Invoice>): Promise<Invoice> => {
    return request<Invoice>(`/invoices/${id}`, {
      method: "PUT",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(payload),
    });
  },

  /** 删除发票（仅草稿） */
  remove: (id: number): Promise<void> => {
    return request<void>(`/invoices/${id}`, { method: "DELETE" }, false);
  },

  /** 提交发票审批 */
  submit: (id: number): Promise<Invoice> => {
    return request<Invoice>(`/invoices/${id}/submit`, { method: "POST" });
  },

  /** 审批通过 */
  approve: (id: number, remark?: string): Promise<Invoice> => {
    return request<Invoice>(`/invoices/${id}/approve`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ remark }),
    });
  },

  /** 驳回 */
  reject: (id: number, remark: string): Promise<Invoice> => {
    return request<Invoice>(`/invoices/${id}/reject`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ remark }),
    });
  },

  /** 确认报销（manager+） */
  reimburse: (id: number, amount: number): Promise<Invoice> => {
    return request<Invoice>(`/invoices/${id}/reimburse`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ amount }),
    });
  },

  /** 获取发票统计 */
  stats: (): Promise<InvoiceStats> => {
    return request<InvoiceStats>("/invoices/stats");
  },
};
