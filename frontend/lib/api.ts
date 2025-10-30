"use client";

import type {
  AuditLog,
  AuditStats,
  BatchUploadItem,
  DatabaseStatus,
  Part,
  Period,
  PeriodSummary,
  PersonalCharge,
  RosterEntry,
  Scheme,
  SourceFile,
  SystemInfo,
  SystemMetrics,
  UnitCharge,
  User,
} from "./types";

// 动态检测API地址：优先使用环境变量，再根据访问域名自动选择后端地址
function getApiBase(): string {
  // 优先使用环境变量
  if (typeof window !== "undefined" && process.env.NEXT_PUBLIC_API_BASE_URL) {
    console.log(`[API检测] 使用环境变量: ${process.env.NEXT_PUBLIC_API_BASE_URL}`);
    return process.env.NEXT_PUBLIC_API_BASE_URL;
  }

  // 运行时动态检测
  if (typeof window !== "undefined") {
    const hostname = window.location.hostname;
    console.log(`[API检测] 当前hostname: ${hostname}`);

    // 支持任何域名访问，自动使用HTTPS + 相同域名
    if (hostname !== "localhost" && hostname !== "127.0.0.1" && !hostname.match(/^\d+\.\d+\.\d+\.\d+$/)) {
      const apiUrl = `https://${hostname}/api`;
      console.log(`[API检测] 域名访问，使用: ${apiUrl}`);
      return apiUrl;
    }

    // 本地访问，使用localhost:8081
    const localUrl = "http://localhost:8081/api";
    console.log(`[API检测] 本地访问，使用: ${localUrl}`);
    return localUrl;
  }

  // SSR时优先使用环境变量
  if (process.env.NEXT_PUBLIC_API_BASE_URL) {
    return process.env.NEXT_PUBLIC_API_BASE_URL;
  }

  // SSR时的默认值（仅用于服务端渲染，不影响客户端）
  return "http://localhost:8081/api";
}

const API_BASE = getApiBase();

async function request<T>(
  path: string,
  init?: RequestInit,
  expectJson = true,
): Promise<T> {
  const url = `${API_BASE}${path}`;
  console.log(`Making request to: ${url}`);
  console.log(`API_BASE is: ${API_BASE}`);

  // Get token from localStorage for authenticated requests
  const token = localStorage.getItem("token");
  const authHeaders: Record<string, string> = token ? { Authorization: `Bearer ${token}` } : {};

  const res = await fetch(url, {
    ...init,
    headers: {
      ...authHeaders,
      ...(init?.headers || {}),
    },
    cache: "no-store",
  });

  if (!res.ok) {
    let detail = "";
    try {
      const data = await res.json();
      detail = data?.error || JSON.stringify(data);
    } catch {
      detail = res.statusText;
    }
    throw new Error(detail || "请求失败");
  }

  if (!expectJson) {
    return undefined as T;
  }

  return (await res.json()) as T;
}

export async function listPeriods(): Promise<Period[]> {
  return request<Period[]>("/periods");
}

export async function createPeriod(yearMonth: string): Promise<Period> {
  return request<Period>("/periods", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ year_month: yearMonth }),
  });
}

export async function listFiles(periodId: number): Promise<SourceFile[]> {
  return request<SourceFile[]>(`/periods/${periodId}/files`);
}

export async function getRoster(periodId: number): Promise<RosterEntry[]> {
  return request<RosterEntry[]>(`/periods/${periodId}/roster`);
}

interface UploadFileParams {
  periodId: number;
  scheme: Scheme;
  part: Part;
  file: File;
}

export async function uploadSourceFile({
  periodId,
  scheme,
  part,
  file,
}: UploadFileParams): Promise<{ file: SourceFile; imported: number }> {
  const formData = new FormData();
  formData.append("scheme", scheme);
  formData.append("part", part);
  formData.append("file", file);

  const token = localStorage.getItem("token");
  const headers: Record<string, string> = token ? { Authorization: `Bearer ${token}` } : {};

  const res = await fetch(`${API_BASE}/periods/${periodId}/files`, {
    method: "POST",
    headers,
    body: formData,
  });

  if (!res.ok) {
    let detail = await res.text();
    try {
      const data = JSON.parse(detail);
      detail = data?.error || detail;
    } catch {
      // ignore
    }
    throw new Error(detail || "上传失败");
  }

  return (await res.json()) as { file: SourceFile; imported: number };
}

export async function uploadRoster(
  periodId: number,
  file: File,
): Promise<{ imported: number }> {
  const formData = new FormData();
  formData.append("file", file);

  const token = localStorage.getItem("token");
  const headers: Record<string, string> = token ? { Authorization: `Bearer ${token}` } : {};

  const res = await fetch(`${API_BASE}/periods/${periodId}/roster`, {
    method: "POST",
    headers,
    body: formData,
  });

  if (!res.ok) {
    let detail = await res.text();
    try {
      const data = JSON.parse(detail);
      detail = data?.error || detail;
    } catch {
      // ignore
    }
    throw new Error(detail || "花名册上传失败");
  }

  return (await res.json()) as { imported: number };
}

export async function importLatestRoster(
  periodId: number,
): Promise<{ imported: number; message: string }> {
  const token = localStorage.getItem("token");
  const headers = {
    "Content-Type": "application/json",
    ...(token ? { Authorization: `Bearer ${token}` } : {}),
  };

  const res = await fetch(`${API_BASE}/periods/${periodId}/roster/import`, {
    method: "POST",
    headers,
  });

  if (!res.ok) {
    let detail = await res.text();
    try {
      const data = JSON.parse(detail);
      detail = data?.error || detail;
    } catch {
      // ignore
    }
    throw new Error(detail || "一键导入失败");
  }

  return (await res.json()) as { imported: number; message: string };
}

interface BatchUploadParams {
  periodId: number;
  items: Array<{ scheme: Scheme; part: Part; file: File }>;
}

export async function uploadSourceFilesBatch({
  periodId,
  items,
}: BatchUploadParams): Promise<{ items: BatchUploadItem[] }> {
  const formData = new FormData();
  items.forEach((item) => {
    formData.append("scheme", item.scheme);
    formData.append("part", item.part);
    formData.append("files", item.file);
  });

  const token = localStorage.getItem("token");
  const headers: Record<string, string> = token ? { Authorization: `Bearer ${token}` } : {};

  const res = await fetch(`${API_BASE}/periods/${periodId}/files/batch`, {
    method: "POST",
    headers,
    body: formData,
  });

  if (!res.ok) {
    let detail = await res.text();
    try {
      const data = JSON.parse(detail);
      detail = data?.error || detail;
    } catch {
      // ignore
    }
    throw new Error(detail || "批量上传失败");
  }

  return (await res.json()) as { items: BatchUploadItem[] };
}

export async function processPeriod(periodId: number): Promise<{
  period_id: number;
  summary: PeriodSummary[];
  personal: PersonalCharge[];
  unit: UnitCharge[];
}> {
  return request(`/periods/${periodId}/process`, { method: "POST" });
}

export async function getSummary(
  periodId: number,
): Promise<PeriodSummary[]> {
  return request(`/periods/${periodId}/summary`);
}

export async function getCharges(
  periodId: number,
  part: Part,
): Promise<PersonalCharge[] | UnitCharge[]> {
  return request(`/periods/${periodId}/charges?part=${part}`);
}

export async function downloadChargesExcel(
  periodId: number,
  part: Part,
): Promise<Blob> {
  const token = localStorage.getItem("token");
  const headers: Record<string, string> = token ? { Authorization: `Bearer ${token}` } : {};

  const res = await fetch(
    `${API_BASE}/periods/${periodId}/charges/export?part=${part}`,
    {
      headers,
      cache: "no-store",
    },
  );

  if (!res.ok) {
    let detail = await res.text();
    try {
      const data = JSON.parse(detail);
      detail = data?.error || detail;
    } catch {
      // ignore
    }
    throw new Error(detail || "导出失败");
  }

  return res.blob();
}

export async function downloadRosterTemplate(): Promise<Blob> {
  const token = localStorage.getItem("token");
  const headers: Record<string, string> = token ? { Authorization: `Bearer ${token}` } : {};

  const res = await fetch(`${API_BASE}/roster-template`, {
    headers,
    cache: "no-store",
  });

  if (!res.ok) {
    let detail = await res.text();
    try {
      const data = JSON.parse(detail);
      detail = data?.error || detail;
    } catch {
      // ignore
    }
    throw new Error(detail || "模板下载失败");
  }

  return res.blob();
}

export interface SchemeChargeDetail {
  name: string;
  id_number: string;
  department: string;
  base: number;
  amount: number;
}

export async function getSchemeCharges(
  periodId: number,
  scheme: Scheme,
  part: Part,
  isAdjustment?: boolean,
): Promise<SchemeChargeDetail[]> {
  let url = `/periods/${periodId}/charges/scheme?scheme=${scheme}&part=${part}`;
  if (isAdjustment !== undefined) {
    url += `&is_adjustment=${isAdjustment}`;
  }
  return request<SchemeChargeDetail[]>(url);
}

export async function downloadSchemeChargesExcel(
  periodId: number,
  scheme: Scheme,
  part: Part,
): Promise<Blob> {
  const token = localStorage.getItem("token");
  const headers: Record<string, string> = token ? { Authorization: `Bearer ${token}` } : {};

  const res = await fetch(
    `${API_BASE}/periods/${periodId}/charges/scheme/export?scheme=${scheme}&part=${part}`,
    {
      headers,
      cache: "no-store",
    },
  );

  if (!res.ok) {
    let detail = await res.text();
    try {
      const data = JSON.parse(detail);
      detail = data?.error || detail;
    } catch {
      // ignore
    }
    throw new Error(detail || "导出失败");
  }

  return res.blob();
}

export async function resetPeriod(periodId: number): Promise<{ message: string }> {
  return request<{ message: string }>(`/periods/${periodId}/reset`, {
    method: "POST",
  });
}

export async function deletePeriod(periodId: number): Promise<{ message: string }> {
  return request<{ message: string }>(`/periods/${periodId}`, {
    method: "DELETE",
  });
}

// 补退文件批量上传
export async function uploadAdjustmentsBatch(
  periodId: number,
  files: File[],
): Promise<{ items: BatchUploadItem[] }> {
  const formData = new FormData();
  files.forEach((file) => {
    formData.append("files", file);
  });

  const token = localStorage.getItem("token");
  const headers: Record<string, string> = token ? { Authorization: `Bearer ${token}` } : {};

  const res = await fetch(`${API_BASE}/periods/${periodId}/adjustments/batch`, {
    method: "POST",
    headers,
    body: formData,
  });

  if (!res.ok) {
    let detail = await res.text();
    try {
      const data = JSON.parse(detail);
      detail = data?.error || detail;
    } catch {
      // ignore
    }
    throw new Error(detail || "补退文件批量上传失败");
  }

  return (await res.json()) as { items: BatchUploadItem[] };
}

// 处理补退数据
export async function processAdjustments(periodId: number): Promise<{
  period_id: number;
  summary: PeriodSummary[];
  personal: PersonalCharge[];
  unit: UnitCharge[];
}> {
  return request(`/periods/${periodId}/adjustments/process`, { method: "POST" });
}

// 清空社保文件
export async function clearFiles(periodId: number): Promise<{ message: string; cleared: string }> {
  return request<{ message: string; cleared: string }>(`/periods/${periodId}/files/clear`, {
    method: "POST",
  });
}

// 清空补退文件
export async function clearAdjustments(periodId: number): Promise<{ message: string; cleared: string }> {
  return request<{ message: string; cleared: string }>(`/periods/${periodId}/adjustments/clear`, {
    method: "POST",
  });
}

// 认证相关API函数
export async function changePassword(currentPassword: string, newPassword: string): Promise<{ message: string }> {
  return request<{ message: string }>("/auth/change-password", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({
      current_password: currentPassword,
      new_password: newPassword
    }),
  });
}

// 审计相关API函数
export async function getAuditLogs(params?: {
  user_id?: number;
  limit?: number;
  offset?: number;
  action?: string;
  status?: string;
  start_date?: string;
  end_date?: string;
}): Promise<{ logs: AuditLog[]; total: number; limit: number; offset: number }> {
  const searchParams = new URLSearchParams();
  if (params) {
    Object.entries(params).forEach(([key, value]) => {
      if (value !== undefined) {
        searchParams.append(key, value.toString());
      }
    });
  }
  const queryString = searchParams.toString();
  const url = queryString ? `/audit/logs?${queryString}` : '/audit/logs';
  return request(url);
}

export async function getAuditStats(days?: number): Promise<AuditStats> {
  const url = days ? `/audit/stats/system?days=${days}` : '/audit/stats/system';
  return request(url);
}

// 监控相关API函数
export async function getSystemMetrics(): Promise<SystemMetrics> {
  return request('/monitoring/metrics');
}

export async function getDatabaseStatus(): Promise<DatabaseStatus> {
  return request('/monitoring/database');
}

export async function getSystemInfo(): Promise<SystemInfo> {
  return request('/monitoring/info');
}

export async function runMaintenance(): Promise<{ message: string; tasks_completed: string[] }> {
  return request<{ message: string; tasks_completed: string[] }>('/monitoring/maintenance', {
    method: 'POST',
  });
}

// 认证相关API函数（供其他组件使用）
export async function getUserProfile(token: string): Promise<User> {
  return request('/auth/profile', {
    headers: { Authorization: `Bearer ${token}` },
  });
}

export async function login(credentials: { username: string; password: string }): Promise<{ token: string; user: unknown }> {
  return request('/auth/login', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(credentials),
  });
}

export async function register(userData: { username: string; email: string; password: string; fullName?: string; companyId: string }): Promise<{ email: string; message: string }> {
  return request('/auth/register', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      username: userData.username,
      email: userData.email,
      password: userData.password,
      full_name: userData.fullName,
      companyId: userData.companyId,
    }),
  });
}

export async function verifyEmail(token: string): Promise<{ message: string }> {
  return request(`/auth/verify-email?token=${token}`);
}

// 获取组织机构选项（用于注册时选择所属公司）
export interface CompanyOption {
  id: string;
  name: string;
  type: "group" | "subsidiary";
}

export async function getCompanyOptions(): Promise<CompanyOption[]> {
  // 暂时返回硬编码数据，后续可连接到组织机构API
  // TODO: 连接到真实的组织机构API
  return [
    {
      id: "1",
      name: "某某集团有限公司",
      type: "group",
    },
    {
      id: "2",
      name: "生产子公司",
      type: "subsidiary",
    },
    {
      id: "11",
      name: "营销子公司",
      type: "subsidiary",
    },
  ];
}
