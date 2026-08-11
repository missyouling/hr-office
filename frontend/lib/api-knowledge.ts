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

/** 知识库 */
export interface KnowledgeBase {
  id: number;
  user_id?: number | null;
  name: string;
  description: string;
  source_module: string;
  embedding_model_id?: number | null;
  chunking_config?: unknown;
  visibility: string;
  is_system: boolean;
  owner_id?: number | null;
  created_at: string;
  updated_at: string;
}

/** 知识库访问规则（角色 + 部门 + 用户三维；多条规则 OR 组合） */
export interface KBAccessRule {
  id: number;
  knowledge_base_id: number;
  role_level?: string | null;
  department_id?: number | null;
  user_id?: number | null;
  created_at: string;
}

/** 知识库字段脱敏规则 */
export interface KBFieldMask {
  id: number;
  knowledge_base_id: number;
  field_name: string;
  mask_pattern: string;
  exempt_role?: string | null;
  created_at: string;
  updated_at: string;
}

/** 知识库统计数据 */
export interface KBStats {
  total_count: number;
  system_count: number;
  custom_count: number;
  by_visibility: { visibility: string; count: number }[];
  by_source_module: { source_module: string; count: number }[];
}

/** 知识库列表响应 */
export interface KBListResponse {
  items: KnowledgeBase[];
  total: number;
}

/** 知识库详情响应 */
export interface KBDetailResponse {
  item: KnowledgeBase;
  rules: KBAccessRule[];
  masks: KBFieldMask[];
}

/** 通用单项响应 */
export interface KBItemResponse<T = KnowledgeBase> {
  item: T;
}

/** 列表响应 */
export interface KBListItemResponse<T> {
  items: T[];
  total: number;
}

/** 入库操作响应 */
export interface KBIngestResponse {
  message: string;
  source_module: string;
  kb_id: number;
  since?: string;
  scanned: number;
  ingested: number;
  skipped: number;
}

/** 删除响应 */
export interface KBDeleteResponse {
  ok: boolean;
}

/** 新增/修改访问规则的请求体 */
export interface KBAccessRulePayload {
  role_level?: string | null;
  department_id?: number | null;
  user_id?: number | null;
}

/** 新增/修改脱敏规则的请求体 */
export interface KBFieldMaskPayload {
  field_name: string;
  mask_pattern: string;
  exempt_role?: string | null;
}

/** 创建知识库的请求体 */
export interface KBCreatePayload {
  name: string;
  description?: string;
  source_module?: string;
  visibility?: string;
  chunking_config?: unknown;
  embedding_model_id?: number | null;
}

/** 更新知识库的请求体 */
export type KBUpdatePayload = Partial<KBCreatePayload>;

/** 入库请求体 */
export interface KBIngestPayload {
  since?: string;
  module?: string;
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

// ========== 知识库 API ==========

export const knowledgeApi = {
  /** 获取知识库列表（登录用户可见） */
  list: (): Promise<KBListResponse> =>
    request<KBListResponse>("/knowledge-bases"),

  /** 获取单个知识库详情（含规则和脱敏列表） */
  get: (id: number): Promise<KBDetailResponse> =>
    request<KBDetailResponse>(`/knowledge-bases/${id}`),

  /** 创建知识库（admin 专属） */
  create: (payload: KBCreatePayload): Promise<KBItemResponse> =>
    request<KBItemResponse>("/knowledge-bases", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(payload),
    }),

  /** 更新知识库（admin 专属） */
  update: (id: number, payload: KBUpdatePayload): Promise<KBItemResponse> =>
    request<KBItemResponse>(`/knowledge-bases/${id}`, {
      method: "PUT",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(payload),
    }),

  /** 删除知识库（admin 专属，系统模板不可删） */
  remove: (id: number): Promise<KBDeleteResponse> =>
    request<KBDeleteResponse>(`/knowledge-bases/${id}`, { method: "DELETE" }, false),

  /** 获取知识库的访问规则列表 */
  listRules: (id: number): Promise<KBListItemResponse<KBAccessRule>> =>
    request<KBListItemResponse<KBAccessRule>>(`/knowledge-bases/${id}/rules`),

  /** 添加访问规则（admin 专属） */
  addRule: (id: number, payload: KBAccessRulePayload): Promise<KBItemResponse<KBAccessRule>> =>
    request<KBItemResponse<KBAccessRule>>(`/knowledge-bases/${id}/rules`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(payload),
    }),

  /** 删除访问规则（admin 专属） */
  removeRule: (id: number, ruleId: number): Promise<KBDeleteResponse> =>
    request<KBDeleteResponse>(`/knowledge-bases/${id}/rules/${ruleId}`, { method: "DELETE" }, false),

  /** 获取知识库的脱敏规则列表 */
  listMasks: (id: number): Promise<KBListItemResponse<KBFieldMask>> =>
    request<KBListItemResponse<KBFieldMask>>(`/knowledge-bases/${id}/masks`),

  /** 添加脱敏规则（admin 专属） */
  addMask: (id: number, payload: KBFieldMaskPayload): Promise<KBItemResponse<KBFieldMask>> =>
    request<KBItemResponse<KBFieldMask>>(`/knowledge-bases/${id}/masks`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(payload),
    }),

  /** 删除脱敏规则（admin 专属） */
  removeMask: (id: number, maskId: number): Promise<KBDeleteResponse> =>
    request<KBDeleteResponse>(`/knowledge-bases/${id}/masks/${maskId}`, { method: "DELETE" }, false),

  /** 触发出入库（登录用户可操作，占位实现） */
  ingest: (id: number, payload?: KBIngestPayload): Promise<KBIngestResponse> =>
    request<KBIngestResponse>(`/knowledge-bases/${id}/ingest`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(payload || {}),
    }),

  /** 获取知识库统计（admin 专属） */
  stats: (): Promise<KBStats> =>
    request<KBStats>("/knowledge-bases/stats"),
};
