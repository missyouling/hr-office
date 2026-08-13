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
  /** 后端解析字段置信度与缺失字段摘要 */
  field_confidence?: Record<string, unknown> | null;
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

// ========== P7.3 上传与解析任务类型 ==========

/** 解析任务状态 */
export type InvoiceParsingTaskStatus = "pending" | "running" | "succeeded" | "failed";

/** 批量上传中单个文件的处理结果 */
export interface InvoiceUploadItem {
  original_name: string;
  invoice_id?: number;
  task_id?: number;
  /** "pending" 表示已受理待解析；"failed" 表示上传/创建失败 */
  status: "pending" | "failed";
  error_code?: string;
  error?: string;
  /** 检测到内容相同的既有发票时为 true（重复预警） */
  duplicate_warning?: boolean;
}

/** 批量上传接口响应 */
export interface InvoiceUploadResult {
  items: InvoiceUploadItem[];
}

/** 识别字段（带置信度，契约可选扩展；后端未返回时前端按字段缺失处理） */
export interface ParsedInvoiceField {
  key: string;
  label: string;
  value?: string | number | null;
  /** 0~1 置信度，低于阈值时前端高亮 */
  confidence?: number;
}

/** 发票解析任务详情（GET /invoices/{id}/parsing-task 与 retry 接口返回） */
export interface ParsingTaskDetail {
  id: number;
  invoice_id: number;
  status: InvoiceParsingTaskStatus;
  attempt_count?: number;
  max_attempts?: number;
  error_code?: string;
  last_error?: string;
  started_at?: string | null;
  completed_at?: string | null;
  created_at?: string;
  updated_at?: string;
  /** 识别字段明细（可选，后端未实现时缺省） */
  fields?: ParsedInvoiceField[];
}

/** 上传可选参数（关联采购来源） */
export interface InvoiceUploadOptions {
  source_type?: string;
  source_id?: number;
}

/**
 * 归一化解析任务详情响应：
 * 兼容后端两种返回形态——直接返回任务对象，或包装为 { task: {...} }。
 */
export function normalizeParsingTaskDetail(payload: unknown): ParsingTaskDetail {
  const raw = (payload && typeof payload === "object" && ("task" in payload || "item" in payload)
    ? (payload as { task?: unknown; item?: unknown }).task ?? (payload as { item?: unknown }).item
    : payload) as Partial<ParsingTaskDetail> | null;
  if (!raw || typeof raw !== "object" || typeof raw.id !== "number") {
    throw new Error("解析任务详情格式错误");
  }
  return {
    id: raw.id,
    invoice_id: raw.invoice_id ?? 0,
    status: raw.status ?? "pending",
    attempt_count: raw.attempt_count,
    max_attempts: raw.max_attempts,
    error_code: raw.error_code,
    last_error: raw.last_error,
    started_at: raw.started_at,
    completed_at: raw.completed_at,
    created_at: raw.created_at,
    updated_at: raw.updated_at,
    fields: raw.fields,
  };
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

  // ========== P7.3 批量上传与解析任务 ==========

  /**
   * 批量上传 PDF 发票（multipart/form-data）。
   * 后端逐文件返回结果：status 为 "pending"（已受理待解析）或 "failed"（失败）。
   */
  upload: (files: File[], options?: InvoiceUploadOptions): Promise<InvoiceUploadResult> => {
    const formData = new FormData();
    files.forEach((file) => formData.append("files", file));
    if (options?.source_type) formData.append("source_type", options.source_type);
    if (options?.source_id) formData.append("source_id", String(options.source_id));
    return request<InvoiceUploadResult>("/invoices/upload", {
      method: "POST",
      body: formData,
    });
  },

  /** 获取发票解析任务详情 */
  getParsingTask: async (id: number): Promise<ParsingTaskDetail> => {
    const payload = await request<unknown>(`/invoices/${id}/parsing-task`);
    return normalizeParsingTaskDetail(payload);
  },

  /** 触发失败解析任务重试，返回最新任务详情 */
  retryParsingTask: async (id: number): Promise<ParsingTaskDetail> => {
    const payload = await request<unknown>(`/invoices/${id}/parsing-task/retry`, {
      method: "POST",
    });
    return normalizeParsingTaskDetail(payload);
  },
};
