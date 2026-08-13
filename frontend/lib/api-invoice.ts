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

/** 发票归档状态（与审批/报销状态独立） */
export type InvoiceArchiveStatus = "pending" | "confirmed" | "voided";

/** 发票/凭证归档类型 */
export type InvoiceVoucherType = "vat_input" | "receipt" | "payment_proof" | "e_itinerary" | "other";

/** 发票 */
export interface Invoice {
  id: number;
  user_id?: number | null;
  invoice_no: string;
  /** 增值税发票代码（仅增值税票） */
  invoice_code?: string;
  /** 电子发票号码（数电票） */
  electronic_invoice_no?: string;
  invoice_date: string;
  invoice_type?: string;
  amount: number;
  tax_amount: number;
  total_amount: number;
  seller: string;
  seller_tax_no?: string;
  buyer?: string;
  buyer_tax_no?: string;
  purpose?: string;
  remark?: string;
  attachment_url?: string;
  /** 受控附件文件 ID（新附件经 StorageManager 存管） */
  attachment_file_id?: number | null;
  /** 附件 SHA-256 哈希 */
  file_sha256?: string;
  archive_status?: InvoiceArchiveStatus;
  voucher_type?: InvoiceVoucherType;
  /** 购方主体匹配结果（确认时评估） */
  buyer_matched?: boolean;
  buyer_match_note?: string;
  recognition_source?: string;
  /** 逻辑删除后的清理时间（软删发票 30 天清理） */
  purge_after?: string | null;
  source_type?: string;
  source_id?: number | null;
  applicant_id?: number | null;
  reimburse_amount: number;
  status: "draft" | "submitted" | "approved" | "rejected" | "reimbursed";
  approver_id?: number | null;
  approved_at?: string | null;
  approval_remark?: string;
  confirmed_by?: number | null;
  confirmed_at?: string | null;
  voided_by?: number | null;
  voided_at?: string | null;
  voided_reason?: string;
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

/** 后端发票统计原始响应（GET /invoices/stats） */
export interface InvoiceStatsResponse {
  total_count: number;
  total_amount: number;
  /** 各状态统计（含计数与金额） */
  by_status: { status: string; count: number; amount: number }[];
  /** 各来源统计 */
  by_source: { source_type: string; count: number }[];
}

/** 发票统计（前端消费形态：by_status/by_source 归一化为 status->count 映射） */
export interface InvoiceStats {
  total: number;
  total_amount: number;
  by_status: Record<string, number>;
  by_source: Record<string, number>;
}

/**
 * 归一化后端统计响应：数组 -> Record，缺失或非法字段给安全默认值。
 * 后端 by_status/by_source 为数组，组件按 status->count 映射消费。
 */
export function normalizeInvoiceStats(payload: unknown): InvoiceStats {
  const raw = (payload && typeof payload === "object" ? payload : {}) as Partial<InvoiceStatsResponse>;
  const byStatus: Record<string, number> = {};
  const bySource: Record<string, number> = {};
  if (Array.isArray(raw.by_status)) {
    raw.by_status.forEach((row) => {
      if (row?.status) byStatus[row.status] = row.count ?? 0;
    });
  }
  if (Array.isArray(raw.by_source)) {
    raw.by_source.forEach((row) => {
      if (row?.source_type) bySource[row.source_type] = row.count ?? 0;
    });
  }
  return {
    total: raw.total_count ?? 0,
    total_amount: raw.total_amount ?? 0,
    by_status: byStatus,
    by_source: bySource,
  };
}

/** 发票列表查询参数（list 与 CSV 导出共享，与后端 buildInvoiceListQuery 一致） */
export interface InvoiceListParams {
  status?: string;
  archive_status?: InvoiceArchiveStatus;
  source_type?: string;
  keyword?: string;
  seller?: string;
  date_from?: string;
  date_to?: string;
  applicant_id?: number;
  page?: number;
  page_size?: number;
}

/** 后端通用单条操作响应（confirm/void/correct 均包装为 { item }） */
export interface InvoiceActionResponse {
  item: Invoice;
}

/** 确认归档响应（附带关联采购/购方预警） */
export interface InvoiceConfirmResponse extends InvoiceActionResponse {
  warnings: string[];
}

/** 已确认发票更正载荷（白名单字段，未提供的字段不更新；reason 必填） */
export interface InvoiceCorrectPayload {
  reason: string;
  invoice_no?: string;
  invoice_code?: string;
  electronic_invoice_no?: string;
  invoice_date?: string;
  invoice_type?: string;
  amount?: number;
  tax_amount?: number;
  total_amount?: number;
  seller?: string;
  seller_tax_no?: string;
  buyer?: string;
  buyer_tax_no?: string;
  purpose?: string;
  remark?: string;
  voucher_type?: InvoiceVoucherType;
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

/** 构建发票列表/导出共享的查询字符串（空参数返回空串） */
function buildInvoiceQuery(params?: InvoiceListParams): string {
  if (!params) return "";
  const searchParams = new URLSearchParams();
  Object.entries(params).forEach(([key, value]) => {
    if (value !== undefined && value !== null) {
      searchParams.set(key, String(value));
    }
  });
  const qs = searchParams.toString();
  return qs ? `?${qs}` : "";
}

/** 二进制/文件下载请求：自动注入 JWT，失败抛 Error（错误消息格式与 request 一致） */
async function downloadBlob(path: string, init?: RequestInit): Promise<Blob> {
  const url = `${API_BASE}${path}`;
  const token = localStorage.getItem("token");
  const headers = new Headers((init?.headers as Record<string, string>) || {});
  if (token && !headers.has("Authorization")) {
    headers.set("Authorization", `Bearer ${token}`);
  }
  const res = await fetch(url, { ...init, headers, cache: "no-store" });
  if (!res.ok) {
    let detail = res.statusText;
    try {
      const body = await res.json();
      detail = body?.error || JSON.stringify(body);
    } catch {
      // 响应体非 JSON，保持 statusText
    }
    throw new Error(`[${res.status}] ${detail || "下载失败"}`);
  }
  return res.blob();
}

// ========== 发票 API ==========

export const invoiceApi = {
  /** 分页查询发票列表（manager+） */
  list: (params?: InvoiceListParams): Promise<InvoiceListResponse> => {
    return request<InvoiceListResponse>(`/invoices${buildInvoiceQuery(params)}`);
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

  /** 获取发票统计（manager+；归一化为前端消费形态） */
  stats: async (): Promise<InvoiceStats> => {
    const payload = await request<unknown>("/invoices/stats");
    return normalizeInvoiceStats(payload);
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

  // ========== P7.3 子任务6：归档管理与导出 ==========

  /** 导出发票 CSV（与 list 共享筛选参数，manager+；返回 UTF-8 BOM 文本 Blob） */
  exportCSV: (params?: InvoiceListParams): Promise<Blob> => {
    return downloadBlob(`/invoices/export${buildInvoiceQuery(params)}`);
  },

  /** 预览发票受控附件（inline PDF Blob，仅待确认状态可访问） */
  getAttachment: (id: number): Promise<Blob> => {
    return downloadBlob(`/invoices/${id}/attachment`);
  },

  /** 下载发票受控附件（attachment PDF Blob） */
  downloadAttachment: (id: number): Promise<Blob> => {
    return downloadBlob(`/invoices/${id}/attachment/download`);
  },

  /** 确认归档（仅 admin、仅已审批且待确认；返回发票与关联预警） */
  confirm: (id: number): Promise<InvoiceConfirmResponse> => {
    return request<InvoiceConfirmResponse>(`/invoices/${id}/confirm`, { method: "POST" });
  },

  /** 作废归档（仅 admin；pending/confirmed -> voided，作废原因必填） */
  void: (id: number, reason: string): Promise<InvoiceActionResponse> => {
    return request<InvoiceActionResponse>(`/invoices/${id}/void`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ reason }),
    });
  },

  /** 更正已确认发票（仅 admin、仅 confirmed；白名单字段 + 更正原因必填） */
  correct: (id: number, payload: InvoiceCorrectPayload): Promise<InvoiceActionResponse> => {
    return request<InvoiceActionResponse>(`/invoices/${id}/correct`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(payload),
    });
  },
};
