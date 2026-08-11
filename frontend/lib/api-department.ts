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

/** 部门 */
export interface Department {
  id: number;
  user_id?: number | null;
  name: string;
  parent_id?: number | null;
  code: string;
  created_at?: string;
  updated_at?: string;
}

/** 部门成员 */
export interface DepartmentMember {
  id: number;
  department_id: number;
  user_id: number;
  role: string; // leader / member
  joined_at?: string;
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

/** 将原始响应包装为 ApiResponse */
function wrap<T>(promise: Promise<T>): Promise<ApiResponse<T>> {
  return promise.then((data) => ({ data }));
}

// ========== 部门 API ==========

export const departmentApi = {
  /** 获取部门列表 */
  list: () => wrap(request<Department[]>("/departments")),

  /** 创建部门 */
  create: (payload: { name: string; parent_id?: number | null; code?: string }) =>
    wrap(
      request<Department>("/departments", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(payload),
      }),
    ),

  /** 更新部门信息 */
  update: (id: number, payload: { name: string; parent_id?: number | null; code?: string }) =>
    wrap(
      request<Department>(`/departments/${id}`, {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(payload),
      }),
    ),

  /** 删除部门 */
  remove: (id: number) =>
    wrap(request<void>(`/departments/${id}`, { method: "DELETE" }, false)),

  /** 将用户分配到部门 */
  assignUser: (deptId: number, payload: { user_id: number; role?: string }) =>
    wrap(
      request<{ message: string; department_id: number; user_id: number }>(
        `/departments/${deptId}/members`,
        {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify(payload),
        },
      ),
    ),

  /** 获取部门成员列表 */
  listMembers: (deptId: number) =>
    wrap(request<DepartmentMember[]>(`/departments/${deptId}/members`)),
};
