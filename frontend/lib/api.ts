"use client";

import type {
  AuditLog,
  AuditStats,
  BatchUploadItem,
  CallbackRecordsResponse,
  EmployeeImportConflict,
  DatabaseStatus,
  SocialInsuranceTemplateOptions,
  Part,
  Period,
  PeriodSummary,
  PersonalCharge,
  ProvidentFundBill,
  ProvidentFundRecord,
  ProvidentFundSettings,
  RosterEntry,
  Scheme,
  SourceFile,
  SystemInfo,
  SystemMetrics,
  UnitCharge,
  User,
  DormSite,
  DormBuilding,
  DormRoom,
  DormBed,
  DormContract,
  DormBill,
  DormMeterRecord,
  DormChargeDetail,
  UserPreferences,
  StorageConfig,
  StorageRule,
  StorageTestResult,
  SysFile,
  StorageModuleConfig,
} from "./types";
import { getRuntimeConfig } from "./runtime-config";

const runtimeConfig = getRuntimeConfig();

const PUBLIC_API_BASE = sanitizeBase(runtimeConfig.API_BASE ?? process.env.NEXT_PUBLIC_API_BASE_URL);
const PUBLIC_API_BASE_DOMAIN = sanitizeBase(runtimeConfig.API_BASE_DOMAIN ?? process.env.NEXT_PUBLIC_API_BASE_URL_DOMAIN);
const PUBLIC_API_BASE_IP = sanitizeBase(runtimeConfig.API_BASE_IP ?? process.env.NEXT_PUBLIC_API_BASE_URL_IP);
const PUBLIC_API_IPV4_FALLBACK_PORT = runtimeConfig.API_IPV4_FALLBACK_PORT ?? process.env.NEXT_PUBLIC_API_IPV4_FALLBACK_PORT;
const INTERNAL_API_BASE = sanitizeBase(process.env.INTERNAL_API_BASE_URL);
const IPV4_REG = /^(?:\d{1,3}\.){3}\d{1,3}$/;
const DEFAULT_LOCAL_API = "http://localhost:8081/api";

function sanitizeBase(value?: string): string | undefined {
  if (!value) {
    return undefined;
  }
  return value.replace(/\/+$/, "");
}

function extractHostname(value?: string): string | undefined {
  if (!value) {
    return undefined;
  }
  try {
    return new URL(value).hostname;
  } catch (error) {
    console.warn("[API检测] 环境变量解析失败:", value, error);
    return undefined;
  }
}

function composeApiBase(protocol: string, hostname: string, port?: string | null): string {
  const safeProtocol = protocol || (isLocalhost(hostname) ? "http:" : "https:");
  const fallbackPort = port || (IPV4_REG.test(hostname) ? "8081" : "");
  const portSegment = fallbackPort ? `:${fallbackPort}` : "";
  return sanitizeBase(`${safeProtocol}//${hostname}${portSegment}/api`) ?? DEFAULT_LOCAL_API;
}

function isLocalhost(hostname: string): boolean {
  return hostname === "localhost" || hostname === "127.0.0.1";
}

// 根据当前执行环境解析 API Base，客户端与服务端分开处理
function resolveApiBase(): string {
  if (typeof window !== "undefined") {
    const { hostname, protocol, port } = window.location;
    console.log(`[API检测] 客户端检测 hostname=${hostname}, protocol=${protocol}, port=${port}`);

    const resolved = resolvePublicBase(hostname, protocol, port);
    console.log(`[API检测] 客户端解析基础地址: ${resolved}`);
    return resolved;
  }

  if (INTERNAL_API_BASE) {
    console.log(`[API检测] 服务端命中内部地址: ${INTERNAL_API_BASE}`);
    return INTERNAL_API_BASE;
  }

  if (PUBLIC_API_BASE) {
    console.log(`[API检测] 服务端回退至公开地址: ${PUBLIC_API_BASE}`);
    return PUBLIC_API_BASE;
  }

  console.log(`[API检测] 环境变量缺失，回退默认地址: ${DEFAULT_LOCAL_API}`);
  return DEFAULT_LOCAL_API;
}

function resolvePublicBase(hostname: string, protocol: string, port?: string | null): string {
  if (isLocalhost(hostname)) {
    const localBase = PUBLIC_API_BASE ?? PUBLIC_API_BASE_IP ?? PUBLIC_API_BASE_DOMAIN ?? DEFAULT_LOCAL_API;
    console.log(`[API检测] 本地环境命中: ${localBase}`);
    return localBase;
  }

  if (IPV4_REG.test(hostname)) {
    if (PUBLIC_API_BASE_IP) {
      console.log(`[API检测] 使用公网 IP 地址变量: ${PUBLIC_API_BASE_IP}`);
      return PUBLIC_API_BASE_IP;
    }

    const publicHost = extractHostname(PUBLIC_API_BASE);
    if (PUBLIC_API_BASE && publicHost && IPV4_REG.test(publicHost)) {
      console.log(`[API检测] 使用旧配置中的 IP 地址: ${PUBLIC_API_BASE}`);
      return PUBLIC_API_BASE;
    }

    if (PUBLIC_API_IPV4_FALLBACK_PORT) {
      const fallbackWithPort = composeApiBase(protocol, hostname, PUBLIC_API_IPV4_FALLBACK_PORT);
      console.log(`[API检测] 使用内网端口变量: ${fallbackWithPort}`);
      return fallbackWithPort;
    }

    const fallback = composeApiBase(protocol, hostname, port);
    console.log(`[API检测] 未配置公网 IP，回退拼接地址: ${fallback}`);
    return fallback;
  }

  if (PUBLIC_API_BASE_DOMAIN) {
    console.log(`[API检测] 使用域名地址变量: ${PUBLIC_API_BASE_DOMAIN}`);
    return PUBLIC_API_BASE_DOMAIN;
  }

  if (PUBLIC_API_BASE) {
    console.log(`[API检测] 使用旧配置中的公开地址: ${PUBLIC_API_BASE}`);
    return PUBLIC_API_BASE;
  }

  const fallback = composeApiBase(protocol, hostname, port);
  console.log(`[API检测] 未命中任何配置，使用组合域名: ${fallback}`);
  return fallback;
}

let cachedApiBase: string | null = null;

function getApiBase(): string {
  if (cachedApiBase) {
    return cachedApiBase;
  }

  cachedApiBase = resolveApiBase();
  return cachedApiBase;
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
    // 包含状态码便于前端判断
    const errorMsg = detail || "请求失败";
    throw new Error(`[${res.status}] ${errorMsg}`);
  }

  if (!expectJson) {
    return undefined as T;
  }

  return (await res.json()) as T;
}

export async function listPeriods(): Promise<Period[]> {
  return request<Period[]>("/periods");
}

export async function getCallbackRecords(): Promise<CallbackRecordsResponse> {
  return request<CallbackRecordsResponse>("/callback-records");
}

export async function uploadCallbackRecords(file: File): Promise<CallbackRecordsResponse> {
  const formData = new FormData();
  formData.append("file", file);
  return request<CallbackRecordsResponse>("/callback-records/upload", {
    method: "POST",
    body: formData,
  });
}

export async function clearCallbackRecords(): Promise<CallbackRecordsResponse> {
  return request<CallbackRecordsResponse>("/callback-records", {
    method: "DELETE",
  });
}

export async function checkAccountAvailability(payload: {
  email?: string;
  username?: string;
}): Promise<{ email_available: boolean; username_available: boolean }> {
  return request<{ email_available: boolean; username_available: boolean }>(
    "/auth/check-availability",
    {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(payload),
    },
  );
}

export async function resendVerificationEmail(email: string): Promise<{ message: string }> {
  return request<{ message: string }>(
    "/auth/resend-verification",
    {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ email }),
    },
  );
}

export async function createPeriod(yearMonth: string, allowAdjustments = false): Promise<Period> {
  return request<Period>("/periods", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ year_month: yearMonth, allow_adjustments: allowAdjustments }),
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

export interface EmployeeResponse {
  id: number;
  user_id: number;
  employee_id: string | null;
  name: string;
  department: string | null;
  position: string | null;
  gender: string | null;
  hire_date: string | null;
  age: string | null;
  work_years: string | null;
  birth_month: string | null;
  education: string | null;
  political_status: string | null;
  work_clothing_size: string | null;
  safety_shoe_size: string | null;
  household_type: string | null;
  ethnicity: string | null;
  native_place: string | null;
  id_address: string | null;
  id_number: string;
  marital_status: string | null;
  social_insurance: string | null;
  has_birth: string | null;
  phone: string | null;
  emergency_contact: string | null;
  emergency_phone: string | null;
  current_address: string | null;
  graduate_school: string | null;
  major: string | null;
  graduation_time: string | null;
  email: string | null;
  remarks: string | null;
  social_insurance_number?: string | null;
  provident_fund_number?: string | null;
  status: "active" | "resigned" | string;
  resign_date: string | null;
  resign_proof_name?: string | null;
  resign_proof_url?: string | null;
  resign_reasons?: string | null;
  created_at: string;
  updated_at: string;
}

export type EmployeeImportMode = "merge" | "insert" | "update";

export interface EmployeeImportResponse {
  mode: EmployeeImportMode;
  inserted: number;
  updated: number;
  skipped: number;
  employees: EmployeeResponse[];
}

export interface DeleteEmployeesResponse {
  deleted: number;
  ids: number[];
}

export type DormMeterRecordPayload = {
  room_id: number;
  meter_date: string;
  billing_start: string;
  billing_end: string;
  inspector?: string;
  charge_details: DormChargeDetail[];
  notes?: string;
};


export class EmployeeImportConflictError extends Error {
  code = 409;
  conflicts: EmployeeImportConflict[];

  constructor(message: string, conflicts: EmployeeImportConflict[]) {
    super(message);
    this.name = "EmployeeImportConflictError";
    this.conflicts = conflicts;
  }
}

// 用户偏好
export async function fetchUserPreferences(): Promise<UserPreferences> {
  return request("/user/preferences");
}

export async function updateUserPreferences(payload: UserPreferences): Promise<UserPreferences> {
  return request("/user/preferences", {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(payload),
  });
}

function buildAuthHeaders(token?: string) {
  const reqToken = token ?? (typeof window !== "undefined" ? localStorage.getItem("token") ?? undefined : undefined);
  return reqToken
    ? {
        Authorization: `Bearer ${reqToken}`,
      }
    : undefined;
}

export async function fetchEmployees(token?: string): Promise<EmployeeResponse[]> {
  return request("/employees", {
    headers: buildAuthHeaders(token),
  });
}

export async function deleteEmployeesApi(ids: number[], token?: string): Promise<DeleteEmployeesResponse> {
  if (!Array.isArray(ids) || ids.length === 0) {
    throw new Error("请选择需要删除的在职员工");
  }

  const validIds = ids.filter((value) => Number.isFinite(value) && value > 0).map((value) => Math.trunc(value));
  if (validIds.length === 0) {
    throw new Error("没有可删除的员工记录");
  }

  const res = await fetch(`${API_BASE}/employees/delete`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      ...buildAuthHeaders(token),
    },
    body: JSON.stringify({ ids: validIds }),
  });

  if (!res.ok) {
    const text = await res.text();
    let detail = text;
    try {
      const data = JSON.parse(text);
      detail = data?.details || data?.error || text;
    } catch {
      // ignore
    }
    throw new Error(detail || "删除员工失败");
  }

  return (await res.json()) as DeleteEmployeesResponse;
}

// Dormitory APIs

export async function fetchDormSites(): Promise<DormSite[]> {
  return request("/dormitories/sites");
}

export async function createDormSite(payload: Partial<DormSite>): Promise<DormSite> {
  return request("/dormitories/sites", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(payload),
  });
}

export async function updateDormSite(siteId: number, payload: Partial<DormSite>): Promise<DormSite> {
  return request(`/dormitories/sites/${siteId}`, {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(payload),
  });
}

export async function deleteDormSite(siteId: number): Promise<{ deleted: number }> {
  return request(`/dormitories/sites/${siteId}`, { method: "DELETE" });
}

export async function fetchDormBuildings(params?: { siteId?: number }): Promise<DormBuilding[]> {
  const query = params?.siteId ? `?site_id=${params.siteId}` : "";
  return request(`/dormitories/buildings${query}`);
}

export async function createDormBuilding(payload: Partial<DormBuilding>): Promise<DormBuilding> {
  return request("/dormitories/buildings", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(payload),
  });
}

export async function updateDormBuilding(buildingId: number, payload: Partial<DormBuilding>): Promise<DormBuilding> {
  return request(`/dormitories/buildings/${buildingId}`, {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(payload),
  });
}

export async function fetchDormRooms(params?: { buildingId?: number; siteId?: number }): Promise<DormRoom[]> {
  const searchParams = new URLSearchParams();
  if (params?.buildingId) searchParams.set("building_id", String(params.buildingId));
  if (params?.siteId) searchParams.set("site_id", String(params.siteId));
  const query = searchParams.toString() ? `?${searchParams.toString()}` : "";
  return request(`/dormitories/rooms${query}`);
}

export async function createDormRoom(payload: Record<string, unknown>): Promise<DormRoom> {
  return request("/dormitories/rooms", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(payload),
  });
}

export async function updateDormRoom(roomId: number, payload: Record<string, unknown>): Promise<DormRoom> {
  return request(`/dormitories/rooms/${roomId}`, {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(payload),
  });
}

export async function deleteDormRoom(roomId: number): Promise<void> {
  await request(`/dormitories/rooms/${roomId}`, {
    method: "DELETE",
  });
}

export async function createDormBed(payload: { room_id: number; bed_number: string; status?: string }): Promise<DormBed> {
  return request("/dormitories/beds", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(payload),
  });
}

export async function fetchDormContracts(params?: { status?: string }): Promise<DormContract[]> {
  const query = params?.status ? `?status=${params.status}` : "";
  return request(`/dormitories/contracts${query}`);
}

export async function createDormContract(payload: Record<string, unknown>): Promise<DormContract> {
  return request("/dormitories/contracts", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(payload),
  });
}

export async function updateDormContract(contractId: number, payload: Record<string, unknown>): Promise<DormContract> {
  return request(`/dormitories/contracts/${contractId}`, {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(payload),
  });
}

export async function deleteDormContract(contractId: number): Promise<void> {
  await request(`/dormitories/contracts/${contractId}`, {
    method: "DELETE",
  });
}

export async function createDormCheckout(contractId: number, payload: Record<string, unknown>) {
  return request(`/dormitories/contracts/${contractId}/checkout`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(payload),
  });
}

export async function fetchDormBills(params?: { status?: string }): Promise<DormBill[]> {
  const query = params?.status ? `?status=${params.status}` : "";
  return request(`/dormitories/bills${query}`);
}

export interface DormBillItemPayload {
  item_type?: string;
  label: string;
  quantity?: number;
  unit_price?: number;
  amount?: number;
}

export interface DormBillPayload {
  bill_code?: string;
  room_id?: number;
  contract_id?: number;
  employee_id?: number;
  employee_name?: string;
  period_label?: string;
  due_date?: string;
  status?: string;
  items: DormBillItemPayload[];
  metadata?: Record<string, unknown>;
}

export async function createDormBill(payload: DormBillPayload): Promise<DormBill> {
  return request(`/dormitories/bills`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(payload),
  });
}

export async function fetchDormMeterRecords(params?: { building_id?: number }): Promise<DormMeterRecord[]> {
  const query = params?.building_id ? `?building_id=${params.building_id}` : "";
  return request(`/dormitories/meter-readings${query}`);
}

export async function createDormMeterRecord(payload: DormMeterRecordPayload): Promise<DormMeterRecord> {
  return request(`/dormitories/meter-readings`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(payload),
  });
}

export async function updateDormMeterRecord(recordId: number, payload: DormMeterRecordPayload): Promise<DormMeterRecord> {
  return request(`/dormitories/meter-readings/${recordId}`, {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(payload),
  });
}

export async function deleteDormMeterRecord(recordId: number): Promise<void> {
  return request(`/dormitories/meter-readings/${recordId}`, { method: "DELETE" });
}

export async function resignEmployeeApi(
  employeeId: number,
  resignDate: string,
  proofFile?: File | null,
  token?: string,
  reasons?: string[],
): Promise<EmployeeResponse> {
  if (!Number.isFinite(employeeId) || employeeId <= 0) {
    throw new Error("无效的员工编号");
  }

  const formData = new FormData();
  formData.append("resign_date", resignDate);
  if (proofFile) {
    formData.append("resign_proof", proofFile);
  }
  if (Array.isArray(reasons) && reasons.length > 0) {
    formData.append("resign_reasons", JSON.stringify(reasons));
  }

  const res = await fetch(`${API_BASE}/employees/${employeeId}/resign`, {
    method: "POST",
    headers: buildAuthHeaders(token),
    body: formData,
  });

  if (!res.ok) {
    const text = await res.text();
    let detail = text;
    try {
      const data = JSON.parse(text);
      detail = data?.details || data?.error || text;
    } catch {
      // ignore
    }
    throw new Error(detail || "离职办理失败");
  }

  return (await res.json()) as EmployeeResponse;
}

// Provident fund APIs
export interface ProvidentRecordPayload {
  personal_account: string;
  name: string;
  identity_number: string;
  personal_base: number;
  personal_amount: number;
  company_amount: number;
  contribution_ratio?: number;
  notes?: string;
}

export interface ProvidentBillPayload {
  month: string;
  overwrite?: boolean;
}

export interface ProvidentSealPayload {
  date?: string;
}

export async function fetchProvidentRecords(params?: { status?: string; q?: string }): Promise<ProvidentFundRecord[]> {
  const searchParams = new URLSearchParams();
  if (params?.status && params.status !== "all") {
    searchParams.set("status", params.status);
  }
  if (params?.q) {
    searchParams.set("q", params.q);
  }
  const query = searchParams.toString() ? `?${searchParams.toString()}` : "";
  return request(`/provident-fund/records${query}`);
}

export async function createProvidentRecord(payload: ProvidentRecordPayload): Promise<ProvidentFundRecord> {
  return request("/provident-fund/records", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(payload),
  });
}

export async function updateProvidentRecord(recordId: number, payload: ProvidentRecordPayload): Promise<ProvidentFundRecord> {
  return request(`/provident-fund/records/${recordId}`, {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(payload),
  });
}

export async function sealProvidentRecord(recordId: number, payload?: ProvidentSealPayload): Promise<ProvidentFundRecord> {
  return request(`/provident-fund/records/${recordId}/seal`, {
    method: "POST",
    headers: payload ? { "Content-Type": "application/json" } : undefined,
    body: payload ? JSON.stringify(payload) : undefined,
  });
}

export async function unsealProvidentRecord(recordId: number, payload?: ProvidentSealPayload): Promise<ProvidentFundRecord> {
  return request(`/provident-fund/records/${recordId}/unseal`, {
    method: "POST",
    headers: payload ? { "Content-Type": "application/json" } : undefined,
    body: payload ? JSON.stringify(payload) : undefined,
  });
}

export async function getProvidentSettings(): Promise<ProvidentFundSettings> {
  return request(`/provident-fund/settings`);
}

export async function updateProvidentSettings(payload: ProvidentBillPayload & { unit_name: string; unit_account: string }): Promise<ProvidentFundSettings> {
  const { unit_name, unit_account } = payload;
  return request(`/provident-fund/settings`, {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ unit_name, unit_account }),
  });
}

export async function generateProvidentBill(payload: ProvidentBillPayload): Promise<ProvidentFundBill> {
  return request(`/provident-fund/bills`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(payload),
  });
}

export async function fetchProvidentBills(params?: { month?: string; withItems?: boolean }): Promise<ProvidentFundBill[]> {
  const searchParams = new URLSearchParams();
  if (params?.month) {
    searchParams.set("month", params.month);
  }
  if (params?.withItems) {
    searchParams.set("with_items", "true");
  }
  const query = searchParams.toString() ? `?${searchParams.toString()}` : "";
  return request(`/provident-fund/bills${query}`);
}

export async function fetchProvidentBillDetail(id: number): Promise<ProvidentFundBill> {
  return request(`/provident-fund/bills/${id}`);
}

export async function deleteProvidentBill(id: number): Promise<void> {
  await request(`/provident-fund/bills/${id}`, {
    method: "DELETE",
  }, false);
}

export async function downloadInsuranceTemplate(type: "increase" | "decrease"): Promise<Blob> {
  const url = `${getApiBase()}/insurance-template?type=${type}`;
  const res = await fetch(url, {
    method: "GET",
    cache: "no-store",
  });
  if (!res.ok) {
    const message = (await res.text()) || "模板下载失败";
    throw new Error(message);
  }
  return res.blob();
}

export interface ResignProofDownloadResult {
  blob: Blob;
  filename: string;
  contentType: string;
  size: number;
}

export async function downloadResignProof(
  employeeId: number,
  token?: string,
): Promise<ResignProofDownloadResult> {
  if (!Number.isFinite(employeeId) || employeeId <= 0) {
    throw new Error("无效的员工编号");
  }

  const res = await fetch(`${API_BASE}/employees/${employeeId}/resign-proof`, {
    headers: buildAuthHeaders(token),
  });

  if (!res.ok) {
    const text = await res.text();
    let detail = text;
    try {
      const data = JSON.parse(text);
      detail = data?.details || data?.error || text;
    } catch {
      // ignore
    }
    throw new Error(detail || "离职证明下载失败");
  }

  const blob = await res.blob();
  const disposition = res.headers.get("content-disposition") || "";
  let filename = `离职证明-${employeeId}`;

  const match = disposition.match(/filename\*=UTF-8''([^;]+)|filename="?([^";]+)"?/i);
  if (match) {
    const encoded = match[1] || match[2];
    try {
      filename = decodeURIComponent(encoded);
    } catch {
      filename = encoded;
    }
  }

  const contentType = res.headers.get("content-type") || blob.type || "application/octet-stream";
  const contentLengthHeader = res.headers.get("content-length");
  const parsedLength = contentLengthHeader ? Number(contentLengthHeader) : Number.NaN;
  const size = Number.isFinite(parsedLength) && parsedLength >= 0 ? parsedLength : blob.size;

  return { blob, filename, contentType, size };
}

export type SocialInsuranceChangeType = "increase" | "decrease";

export interface SocialInsuranceRecordDTO {
  id: number;
  batch_id: number;
  change_type: SocialInsuranceChangeType;
  employee_name: string;
  department: string;
  identity_number: string;
  personal_number: string;
  effective_date: string;
  reason: string;
  template_values: Record<string, string>;
  created_at: string;
  updated_at: string;
  original_file_name?: string;
}

export interface SocialInsuranceImportRecordPayload {
  employee_name: string;
  department?: string;
  identity_number: string;
  personal_number?: string;
  effective_date?: string;
  reason?: string;
  template_values: Record<string, string>;
}

export interface SocialInsuranceImportPayload {
  records: SocialInsuranceImportRecordPayload[];
}

export interface SocialInsuranceImportResponse {
  batch: {
    id: number;
    change_type: SocialInsuranceChangeType;
    original_file_name: string;
    stored_file_name: string;
    stored_file_path: string;
    created_at: string;
  };
  records: SocialInsuranceRecordDTO[];
}

export async function fetchSocialInsuranceChanges(options: {
  changeType?: SocialInsuranceChangeType;
  token?: string;
} = {}): Promise<SocialInsuranceRecordDTO[]> {
  const { changeType, token } = options;
  const params = new URLSearchParams();
  if (changeType) {
    params.set("change_type", changeType);
  }
  const query = params.toString();
  const path = `/social-insurance/changes${query ? `?${query}` : ""}`;
  const result = await request<{ records: SocialInsuranceRecordDTO[] }>(path, {
    headers: buildAuthHeaders(token),
  });
  return result.records ?? [];
}

export async function importSocialInsuranceChanges(
  changeType: SocialInsuranceChangeType,
  file: File,
  payload: SocialInsuranceImportPayload,
  token?: string,
): Promise<SocialInsuranceImportResponse> {
  const formData = new FormData();
  formData.append("change_type", changeType);
  formData.append("payload", JSON.stringify(payload));
  formData.append("file", file);

  const res = await fetch(`${API_BASE}/social-insurance/changes/import`, {
    method: "POST",
    headers: buildAuthHeaders(token),
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
    throw new Error(detail || "社保变动导入失败");
  }

  return (await res.json()) as SocialInsuranceImportResponse;
}

export async function deleteSocialInsuranceChanges(ids: number[], token?: string): Promise<number> {
  const res = await fetch(`${API_BASE}/social-insurance/changes/delete`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      ...(buildAuthHeaders(token) ?? {}),
    },
    body: JSON.stringify({ ids }),
  });

  if (!res.ok) {
    let detail = await res.text();
    try {
      const data = JSON.parse(detail);
      detail = data?.error || detail;
    } catch {
      // ignore
    }
    throw new Error(detail || "社保变动撤销失败");
  }

  const data = (await res.json()) as { deleted?: number };
  return data.deleted ?? 0;
}

export interface SocialInsuranceManualPayload {
  change_type: SocialInsuranceChangeType;
  employee_name: string;
  department?: string;
  identity_number: string;
  personal_number?: string;
  effective_date: string;
  reason?: string;
  template_values: Record<string, string>;
}

export async function createSocialInsuranceChange(
  payload: SocialInsuranceManualPayload,
  token?: string,
): Promise<SocialInsuranceRecordDTO> {
  return request<SocialInsuranceRecordDTO>("/social-insurance/changes", {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      ...(buildAuthHeaders(token) ?? {}),
    },
    body: JSON.stringify(payload),
  });
}

export async function updateSocialInsuranceChange(
  id: number,
  payload: SocialInsuranceManualPayload,
  token?: string,
): Promise<SocialInsuranceRecordDTO> {
  return request<SocialInsuranceRecordDTO>(`/social-insurance/changes/${id}`, {
    method: "PUT",
    headers: {
      "Content-Type": "application/json",
      ...(buildAuthHeaders(token) ?? {}),
    },
    body: JSON.stringify(payload),
  });
}

export async function fetchSocialInsuranceOptions(token?: string): Promise<SocialInsuranceTemplateOptions> {
  const data = await request<{ options: SocialInsuranceTemplateOptions }>("/social-insurance/options", {
    headers: buildAuthHeaders(token),
  });
  return data.options;
}

export async function restoreEmployees(
  payload: { ids?: number[]; idNumbers?: string[] },
  token?: string,
): Promise<{ restored: number; employees: EmployeeResponse[] }> {
  const body: Record<string, unknown> = {};
  if (payload.ids && payload.ids.length > 0) {
    body.ids = payload.ids;
  }
  if (payload.idNumbers && payload.idNumbers.length > 0) {
    body.id_numbers = payload.idNumbers;
  }

  if (!body.ids && !body.id_numbers) {
    throw new Error("请选择需要撤销离职的员工");
  }

  const res = await fetch(`${API_BASE}/employees/restore`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      ...(buildAuthHeaders(token) ?? {}),
    },
    body: JSON.stringify(body),
  });

  if (!res.ok) {
    const text = await res.text();
    let detail = text;
    try {
      const data = JSON.parse(text);
      detail = data?.details || data?.error || text;
    } catch {
      // ignore
    }
    throw new Error(detail || "撤销离职失败");
  }

  return (await res.json()) as { restored: number; employees: EmployeeResponse[] };
}

export async function importEmployees(file: File, token: string, mode: EmployeeImportMode = "merge"): Promise<EmployeeImportResponse> {
  const formData = new FormData();
  formData.append("file", file);
  formData.append("mode", mode);

  const headers: Record<string, string> = { Authorization: `Bearer ${token}` };

  const res = await fetch(`${API_BASE}/employees/import`, {
    method: "POST",
    headers,
    body: formData,
  });

  if (!res.ok) {
    const text = await res.text();
    let detail = text;
    try {
      const data = JSON.parse(text);
      detail = data?.details || data?.error || text;
    } catch {
      // ignore json parse failure
    }
    throw new Error(detail || "员工导入失败");
  }

  return (await res.json()) as EmployeeImportResponse;
}

export async function importResignedEmployees(
  file: File,
  token: string,
  options?: { mode?: EmployeeImportMode; forceTransition?: boolean },
): Promise<EmployeeImportResponse> {
  const mode = options?.mode ?? "merge";
  const force = options?.forceTransition ? "true" : "false";

  const formData = new FormData();
  formData.append("file", file);
  formData.append("mode", mode);
  formData.append("force_transition", force);

  const headers: Record<string, string> = { Authorization: `Bearer ${token}` };

  const res = await fetch(`${API_BASE}/employees/resigned/import`, {
    method: "POST",
    headers,
    body: formData,
  });

  if (res.status === 409) {
    const data = await res.json().catch(() => ({}));
    throw new EmployeeImportConflictError(
      data?.message || data?.error || "存在与在职员工身份证重复的记录",
      (data?.conflicts as EmployeeImportConflict[]) ?? [],
    );
  }

  if (!res.ok) {
    const text = await res.text();
    let detail = text;
    try {
      const data = JSON.parse(text);
      detail = data?.details || data?.error || text;
    } catch {
      // ignore json parse failure
    }
    throw new Error(detail || "离职员工导入失败");
  }

  return (await res.json()) as EmployeeImportResponse;
}

interface ExportEmployeesOptions {
  scope: "filtered" | "all" | "selected";
  status?: string;
  search?: string;
  department?: string;
  ids?: number[];
  idNumbers?: string[];
}

async function downloadBinary(path: string, init?: RequestInit): Promise<Blob> {
  const url = `${API_BASE}${path}`;
  const token = typeof window !== "undefined" ? localStorage.getItem("token") : null;
  const headers = new Headers(init?.headers || {});
  if (token && !headers.has("Authorization")) {
    headers.set("Authorization", `Bearer ${token}`);
  }
  const response = await fetch(url, {
    ...init,
    headers,
  });
  if (!response.ok) {
    let detail = response.statusText;
    try {
      const data = await response.json();
      detail = (data && (data.error || data.details)) || detail;
    } catch {
      // ignore JSON parse error
    }
    throw new Error(detail || "下载失败");
  }
  return await response.blob();
}

export async function downloadEmployeeTemplate(): Promise<Blob> {
  return downloadBinary("/employees/template");
}

export async function downloadResignedEmployeeTemplate(): Promise<Blob> {
  return downloadBinary("/employees/resigned/template");
}

export async function exportEmployees(options: ExportEmployeesOptions): Promise<Blob> {
  const params = new URLSearchParams();
  if (options.status && options.status !== "all") {
    params.set("status", options.status);
  }
  if (options.department && options.department !== "all") {
    params.set("department", options.department);
  }
  if (options.search) {
    params.set("search", options.search);
  }

  const body = {
    scope: options.scope,
    ids: options.ids,
    id_numbers: options.idNumbers,
  };

  return downloadBinary(`/employees/export${params.toString() ? `?${params}` : ""}`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
  });
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

export async function requestPasswordReset(payload: { email: string; username?: string }): Promise<{ message: string }> {
  return request('/auth/request-password-reset', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      email: payload.email,
      username: payload.username,
    }),
  });
}

export interface PasswordResetTokenValidation {
  valid: boolean;
  email?: string;
}

export async function validatePasswordResetToken(token: string): Promise<PasswordResetTokenValidation> {
  const encoded = encodeURIComponent(token);
  return request(`/auth/validate-reset-token?token=${encoded}`);
}

export async function resetPassword(payload: { token: string; newPassword: string }): Promise<{ message: string }> {
  return request('/auth/reset-password', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      token: payload.token,
      new_password: payload.newPassword,
    }),
  });
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

// ============ 档案管理 API ============

export interface DocumentCategory {
  id: number;
  code: string;
  name: string;
  description?: string;
  sub_categories?: DocumentSubCategory[];
  sort_order: number;
}

export interface DocumentSubCategory {
  id: number;
  category_id: number;
  code: string;
  name: string;
  description?: string;
  sort_order?: number;
  fields_config?: Record<string, unknown>;
}

export interface Document {
  id: number;
  user_id: number;
  document_code: string;
  category_code: string;
  sub_category_code: string;
  year: number;
  sequence: number;
  file_name: string;
  document_type: string;
  sub_type: string;
  summary?: string;
  tags?: string | string[]; // 标签数组，存储为 JSON 字符串或数组
  // 前端临时字段
  tagInput?: string; // 标签输入
  signed_date?: string;
  expiration_date?: string;
  retention_period?: string;
  custom_fields?: Record<string, unknown>; // 自定义字段值
  party_a?: string;
  party_b?: string;
  amount?: number;
  payment_progress?: string;
  project_name?: string;
  design_unit?: string;
  designer?: string;
  project_leader?: string;
  equipment_name?: string;
  equipment_model?: string;
  content_description?: string;
  capture_date?: string;
  capturer?: string;
  activity_name?: string;
  carrier_type?: string;
  storage_location?: string; // 存放位置
  file_path?: string;
  file_name_original?: string;
  file_size?: number;
  file_type?: string; // MIME type like "application/pdf"
  file_format?: string; // 文件格式后缀如 pdf/txt
  remarks?: string;
  status: string;
  created_at: string;
  updated_at: string;
}

export interface DocumentListResponse {
  items: Document[];
  total: number;
  page: number;
  page_size: number;
}

// ============ 档案字段定义类型 ============

export interface ConditionConfig {
  field_name: string;   // 触发条件的字段名
  operator: "equals" | "contains" | "gt" | "lt" | "in" | "not_empty"; // 操作符
  value: string;       // 期望值
}

export interface ArchiveSharedField {
  id: number;
  field_name: string;
  field_label: string;
  field_type: string;  // text/number/date/select/file/textarea/multiselect/user
  is_required: boolean;
  is_ocr_related: boolean;
  options: string;
  default_value: string;
  placeholder: string;
  sort_order: number;
}

export interface ArchiveFieldDefinition {
  id: number;
  sub_category_id: number;
  group_id: number | null;
  group?: ArchiveFieldGroup;
  field_name: string;      // 字段英文名（唯一标识）
  field_label: string;     // 显示名称
  field_type: "text" | "textarea" | "number" | "date" | "select" | "multiselect" | "checkbox";
  required: boolean;
  default_value: string;
  options: string;         // 下拉选项（逗号分隔）
  placeholder: string;
  sort_order: number;
  visible: boolean;
  editable: boolean;
  condition_config?: ConditionConfig; // 条件显示配置
  help_text: string;
  created_at: string;
  updated_at: string;
}

export interface ArchiveFieldGroup {
  id: number;
  sub_category_id: number;
  name: string;
  description: string;
  sort_order: number;
  fields?: ArchiveFieldDefinition[];
  created_at: string;
  updated_at: string;
}

export interface SubCategoryFieldsResponse {
  groups: Array<{
    id: number;
    name: string;
    fields: ArchiveFieldDefinition[];
  }>;
  ungrouped: ArchiveFieldDefinition[];
}

export async function fetchDocumentCategories(): Promise<DocumentCategory[]> {
  return request("/archives/categories");
}

export async function fetchDocuments(params?: {
  category_code?: string;
  sub_category_code?: string;
  keyword?: string;
  retention_period?: string;
  status?: string;
  sort_field?: string;
  sort_direction?: "asc" | "desc";
  page?: number;
  page_size?: number;
}): Promise<DocumentListResponse> {
  const query = new URLSearchParams();
  if (params?.category_code) query.set("category_code", params.category_code);
  if (params?.sub_category_code) query.set("sub_category_code", params.sub_category_code);
  if (params?.keyword) query.set("keyword", params.keyword);
  if (params?.retention_period) query.set("retention_period", params.retention_period);
  if (params?.status) query.set("status", params.status);
  if (params?.sort_field) query.set("sort_field", params.sort_field);
  if (params?.sort_direction) query.set("sort_direction", params.sort_direction);
  if (params?.page) query.set("page", String(params.page));
  if (params?.page_size) query.set("page_size", String(params.page_size));

  return request(`/archives/documents?${query}`);
}

export async function createDocument(data: Partial<Document>): Promise<Document> {
  return request("/archives/documents", {
    method: "POST",
    body: JSON.stringify(data),
    headers: { "Content-Type": "application/json" },
  });
}

export async function updateDocument(docId: number, data: Partial<Document>): Promise<Document> {
  return request(`/archives/documents/${docId}`, {
    method: "PUT",
    body: JSON.stringify(data),
    headers: { "Content-Type": "application/json" },
  });
}

export async function deleteDocument(docId: number): Promise<void> {
  await request(`/archives/documents/${docId}`, { method: "DELETE" });
}

export async function uploadDocumentFile(docId: number, file: File): Promise<{ file_path: string; file_name: string; file_size: number }> {
  const formData = new FormData();
  formData.append("file", file);
  
  return request(`/archives/documents/${docId}/upload`, {
    method: "POST",
    body: formData,
  });
}

// ============ 档案 OCR 提取 API ============

export interface OCRExtractResult {
  ocr_status: string;          // success/failed/skipped
  ocr_text: string;            // OCR提取的原文
  shared_fields: Record<string, unknown>;       // 共用字段预填充值
  proprietary_fields: Record<string, unknown>;  // 专用字段预填充值
  error_message: string;       // 错误信息
}

export async function uploadWithOCR(file: File, subCategoryCode: string): Promise<OCRExtractResult> {
  const formData = new FormData();
  formData.append("file", file);
  formData.append("sub_category_code", subCategoryCode);
  
  const token = localStorage.getItem("token");
  const headers: Record<string, string> = token ? { Authorization: `Bearer ${token}` } : {};
  
  const res = await fetch(`${API_BASE}/archives/documents/upload-with-ocr`, {
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
    throw new Error(detail || "OCR识别失败");
  }
  
  return (await res.json()) as OCRExtractResult;
}

export async function getExpiringDocuments(days: number = 30): Promise<Document[]> {
  return request(`/archives/documents/expiring?days=${days}`);
}

// ============ 档案批量操作 API ============

export interface BatchUploadResponse {
  items: Array<{
    id: number;
    document_code: string;
    file_name: string;
    status: 'success' | 'error';
    error?: string;
  }>;
  total: number;
  success: number;
  failed: number;
}

export async function batchUploadDocuments(
  files: File[],
  categoryCode: string,
  subCategoryCode: string,
  metadata?: Record<string, unknown>
): Promise<BatchUploadResponse> {
  const formData = new FormData();
  files.forEach((file) => {
    formData.append("files", file);
  });
  formData.append("category_code", categoryCode);
  formData.append("sub_category_code", subCategoryCode);
  if (metadata) {
    formData.append("metadata", JSON.stringify(metadata));
  }

  const token = localStorage.getItem("token");
  const headers: Record<string, string> = token ? { Authorization: `Bearer ${token}` } : {};

  const res = await fetch(`${API_BASE}/archives/documents/batch-upload`, {
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

  return (await res.json()) as BatchUploadResponse;
}

export async function batchDownloadDocuments(docIds: number[]): Promise<Blob> {
  const token = localStorage.getItem("token");
  const headers: Record<string, string> = token ? { Authorization: `Bearer ${token}` } : {};

  const res = await fetch(`${API_BASE}/archives/documents/batch-download`, {
    method: "POST",
    headers: {
      ...headers,
      "Content-Type": "application/json",
    },
    body: JSON.stringify({ ids: docIds }),
  });

  if (!res.ok) {
    let detail = await res.text();
    try {
      const data = JSON.parse(detail);
      detail = data?.error || detail;
    } catch {
      // ignore
    }
    throw new Error(detail || "批量下载失败");
  }

  return res.blob();
}

export interface ShareLinkResult {
  id: number;
  file_name: string;
  link: string;
  expires_at: string;
}

export async function generateShareLink(
  docIds: number[],
  expiryHours: number = 24
): Promise<ShareLinkResult[]> {
  const token = localStorage.getItem("token");
  const headers: Record<string, string> = token ? { Authorization: `Bearer ${token}` } : {};

  const res = await fetch(`${API_BASE}/archives/documents/share`, {
    method: "POST",
    headers: {
      ...headers,
      "Content-Type": "application/json",
    },
    body: JSON.stringify({ ids: docIds, expiry_hours: expiryHours }),
  });

  if (!res.ok) {
    let detail = await res.text();
    try {
      const data = JSON.parse(detail);
      detail = data?.error || detail;
    } catch {
      // ignore
    }
    throw new Error(detail || "生成分享链接失败");
  }

  const data = await res.json();
  return data.links as ShareLinkResult[];
}

// ============ 档案字段定义 API ============

export async function fetchFieldGroups(subCategoryId?: number): Promise<ArchiveFieldGroup[]> {
  const query = subCategoryId ? `?sub_category_id=${subCategoryId}` : "";
  return request(`/archives/field-groups${query}`);
}

export async function createFieldGroup(data: Partial<ArchiveFieldGroup>): Promise<ArchiveFieldGroup> {
  return request("/archives/field-groups", {
    method: "POST",
    body: JSON.stringify(data),
    headers: { "Content-Type": "application/json" },
  });
}

export async function updateFieldGroup(groupId: number, data: Partial<ArchiveFieldGroup>): Promise<ArchiveFieldGroup> {
  return request(`/archives/field-groups/${groupId}`, {
    method: "PUT",
    body: JSON.stringify(data),
    headers: { "Content-Type": "application/json" },
  });
}

export async function deleteFieldGroup(groupId: number): Promise<void> {
  await request(`/archives/field-groups/${groupId}`, { method: "DELETE" });
}

export async function fetchFieldDefinitions(subCategoryId?: number): Promise<ArchiveFieldDefinition[]> {
  const query = subCategoryId ? `?sub_category_id=${subCategoryId}` : "";
  return request(`/archives/field-definitions${query}`);
}

export async function createFieldDefinition(data: Partial<ArchiveFieldDefinition>): Promise<ArchiveFieldDefinition> {
  return request("/archives/field-definitions", {
    method: "POST",
    body: JSON.stringify(data),
    headers: { "Content-Type": "application/json" },
  });
}

export async function updateFieldDefinition(fieldId: number, data: Partial<ArchiveFieldDefinition>): Promise<ArchiveFieldDefinition> {
  return request(`/archives/field-definitions/${fieldId}`, {
    method: "PUT",
    body: JSON.stringify(data),
    headers: { "Content-Type": "application/json" },
  });
}

export async function deleteFieldDefinition(fieldId: number): Promise<void> {
  await request(`/archives/field-definitions/${fieldId}`, { method: "DELETE" });
}

export async function fetchSharedFields(): Promise<ArchiveSharedField[]> {
  return request("/archives/shared-fields");
}

export async function fetchFieldsBySubCategory(subCategoryId: number): Promise<SubCategoryFieldsResponse> {
  return request(`/archives/sub-categories/${subCategoryId}/fields`);
}

// ============ 档案配置 API ============

export interface RetentionPeriod {
  id: number;
  user_id?: number;
  name: string;
  years: number;       // 0 表示永久
  sort_order: number;
  is_default: boolean;
  created_at: string;
  updated_at: string;
}

export interface StorageLocation {
  id: number;
  user_id?: number;
  name: string;
  description?: string;
  sort_order: number;
  created_at: string;
  updated_at: string;
}

export interface CodeRule {
  id: number;
  user_id?: number;
  category_id?: number;
  sub_category_id?: number;
  name: string;
  pattern: string;
  separator: string;
  description?: string;
  created_at: string;
  updated_at: string;
}

export interface CodeRulePreview {
  sample_code: string;
  next_sequence: number;
  year: number;
  format: string;
}

export interface ArchiveConfig {
  id: number;
  user_id?: number;
  default_code_rule_id?: number;
  auto_generate_code: boolean;
  require_code_prefix: boolean;
  created_at: string;
  updated_at: string;
}

// 保管期限 API
export async function fetchRetentionPeriods(): Promise<RetentionPeriod[]> {
  return request("/archives/retention-periods");
}

export async function createRetentionPeriod(data: Partial<RetentionPeriod>): Promise<RetentionPeriod> {
  return request("/archives/retention-periods", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(data),
  });
}

export async function updateRetentionPeriod(id: number, data: Partial<RetentionPeriod>): Promise<RetentionPeriod> {
  return request(`/archives/retention-periods/${id}`, {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(data),
  });
}

export async function deleteRetentionPeriod(id: number): Promise<void> {
  await request(`/archives/retention-periods/${id}`, { method: "DELETE" });
}

// 存档地点 API
export async function fetchStorageLocations(): Promise<StorageLocation[]> {
  return request("/archives/storage-locations");
}

export async function createStorageLocation(data: Partial<StorageLocation>): Promise<StorageLocation> {
  return request("/archives/storage-locations", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(data),
  });
}

export async function updateStorageLocation(id: number, data: Partial<StorageLocation>): Promise<StorageLocation> {
  return request(`/archives/storage-locations/${id}`, {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(data),
  });
}

export async function deleteStorageLocation(id: number): Promise<void> {
  await request(`/archives/storage-locations/${id}`, { method: "DELETE" });
}

// 编码规则 API
export async function fetchCodeRules(params?: { category_id?: number; sub_category_id?: number }): Promise<CodeRule[]> {
  const query = new URLSearchParams();
  if (params?.category_id) query.set("category_id", String(params.category_id));
  if (params?.sub_category_id) query.set("sub_category_id", String(params.sub_category_id));
  return request(`/archives/code-rules?${query}`);
}

export async function createCodeRule(data: Partial<CodeRule>): Promise<CodeRule> {
  return request("/archives/code-rules", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(data),
  });
}

export async function updateCodeRule(id: number, data: Partial<CodeRule>): Promise<CodeRule> {
  return request(`/archives/code-rules/${id}`, {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(data),
  });
}

export async function deleteCodeRule(id: number): Promise<void> {
  await request(`/archives/code-rules/${id}`, { method: "DELETE" });
}

export async function getCodeRulePreview(categoryCode: string, subCategoryCode: string, year?: number): Promise<CodeRulePreview> {
  const query = new URLSearchParams({ category_code: categoryCode, sub_category_code: subCategoryCode });
  if (year) query.set("year", String(year));
  return request(`/archives/code-rules/preview?${query}`);
}

// 档案全局配置 API
export async function fetchArchiveConfig(): Promise<ArchiveConfig> {
  return request("/archives/config");
}

export async function updateArchiveConfig(data: Partial<ArchiveConfig>): Promise<ArchiveConfig> {
  return request("/archives/config", {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(data),
  });
}

// 一级/二级分类编码 API
export async function createCategoryCode(data: { code: string; name: string; description?: string; sort_order?: number }): Promise<DocumentCategory> {
  return request("/archives/categories", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(data),
  });
}

export async function updateCategoryCode(categoryId: number, data: { code: string; name?: string; description?: string; sort_order?: number }): Promise<DocumentCategory> {
  return request(`/archives/categories/${categoryId}`, {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(data),
  });
}

export async function deleteCategory(categoryId: number): Promise<void> {
  await request(`/archives/categories/${categoryId}`, { method: "DELETE" });
}

export async function createSubCategory(data: { category_id: number; code: string; name: string; description?: string; sort_order?: number }): Promise<DocumentSubCategory> {
  return request("/archives/sub-categories", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(data),
  });
}

export async function updateSubCategoryCode(subCategoryId: number, data: { code: string; name?: string; description?: string; sort_order?: number }): Promise<DocumentSubCategory> {
  return request(`/archives/sub-categories/${subCategoryId}`, {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(data),
  });
}

export async function deleteSubCategory(subCategoryId: number): Promise<void> {
  await request(`/archives/sub-categories/${subCategoryId}`, { method: "DELETE" });
}

export interface Announcement {
  id: number;
  title: string;
  content: string;
  is_top: boolean;
  status: string;
  published_at: string | null;
  created_by: number;
  created_at: string;
  updated_at: string;
}

export async function fetchAnnouncements(status?: string): Promise<Announcement[]> {
  const query = status ? `?status=${status}` : "";
  return request(`/announcements${query}`);
}

export async function createAnnouncement(data: Partial<Announcement>): Promise<Announcement> {
  return request("/announcements", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(data),
  });
}

export async function updateAnnouncement(id: number, data: Partial<Announcement>): Promise<Announcement> {
  return request(`/announcements/${id}`, {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(data),
  });
}

export async function deleteAnnouncement(id: number): Promise<void> {
  await request(`/announcements/${id}`, { method: "DELETE" });
}

export interface Role {
  id: number;
  name: string;
  label: string;
  description: string;
  is_system: boolean;
  user_count: number;
  created_at: string;
  updated_at: string;
}

export interface Permission {
  id: number;
  module: string;
  action: string;
  label: string;
  sort_order: number;
}

export async function listStorageDirectoriesEnhanced(req: { type: string; config: Record<string, unknown> }): Promise<{ directories: string[] }> {
  return request<{ directories: string[] }>("/admin/storage/directories", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(req),
  });
}

export async function fetchRoles(): Promise<Role[]> {
  return request("/rbac/roles");
}

export async function createRole(data: { name: string; label: string; description: string }): Promise<Role> {
  return request("/rbac/roles", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(data),
  });
}

export async function updateRole(id: number, data: { label: string; description: string }): Promise<Role> {
  return request(`/rbac/roles/${id}`, {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(data),
  });
}

export async function deleteRole(id: number): Promise<void> {
  await request(`/rbac/roles/${id}`, { method: "DELETE" });
}

export async function fetchRolePermissions(roleId: number): Promise<Permission[]> {
  return request(`/rbac/roles/${roleId}/permissions`);
}

export async function updateRolePermissions(roleId: number, permissionIds: number[]): Promise<void> {
  await request(`/rbac/roles/${roleId}/permissions`, {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ permission_ids: permissionIds }),
  });
}

export async function fetchPermissions(): Promise<Permission[]> {
  return request("/rbac/permissions");
}

// 存储配置 API
// Storage Config API (new CRUD endpoints)
export async function listStorageConfigs(): Promise<StorageConfig[]> {
  return request("/admin/storage");
}

export async function createStorageConfig(config: Partial<StorageConfig>): Promise<StorageConfig> {
  return request("/admin/storage", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(config),
  });
}

export async function updateStorageConfig(id: number, config: Partial<StorageConfig>): Promise<StorageConfig> {
  return request(`/admin/storage/${id}`, {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(config),
  });
}

export async function deleteStorageConfig(id: number): Promise<void> {
  await request(`/admin/storage/${id}`, { method: "DELETE" });
}

export async function testStorageConnection(config: { type: string; config: Record<string, unknown> }): Promise<StorageTestResult> {
  return request("/admin/storage/test", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(config),
  });
}

export async function listStorageRules(): Promise<StorageRule[]> {
  return request("/admin/storage/rules");
}

export async function updateStorageRules(rules: Partial<StorageRule>[]): Promise<StorageRule[]> {
  return request("/admin/storage/rules/batch", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ rules }),
  });
}

// Storage - New endpoints
export async function getStorageStatus(id: number): Promise<{
  id: number;
  name: string;
  type: string;
  status: string;
  enabled: boolean;
  is_default: boolean;
  last_health_check: string | null;
  fail_count: number;
  max_fail_count: number;
}> {
  return request(`/admin/storage/${id}/status`);
}

export async function setStoragePrimary(id: number): Promise<{ message: string }> {
  return request(`/admin/storage/${id}/set-primary`, { method: "POST" });
}

export interface StorageCapacityInfo {
  total_bytes: number;
  used_bytes: number;
  free_bytes: number;
  usage_percent: number;
  available: boolean;
  message?: string;
}

export async function getStorageCapacity(id: number): Promise<StorageCapacityInfo> {
  return request(`/admin/storage/${id}/capacity`);
}

export async function createStorageRule(rule: Partial<StorageRule>): Promise<StorageRule> {
  return request("/admin/storage/rules", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(rule),
  });
}

export async function updateStorageRule(id: number, rule: Partial<StorageRule>): Promise<StorageRule> {
  return request(`/admin/storage/rules/${id}`, {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(rule),
  });
}

export async function deleteStorageRule(id: number): Promise<{ message: string }> {
  return request(`/admin/storage/rules/${id}`, { method: "DELETE" });
}

export interface StorageDirectory {
  key: string;
  label: string;
  path: string;
  description: string;
  children?: StorageDirectory[];
}

export async function listStorageDirectories(): Promise<StorageDirectory[]> {
  return request("/admin/storage/directories");
}

export async function saveStorageConfig(config: {

  storages: Array<{
    type: string;
    root_path?: string;
    s3_endpoint?: string;
    s3_bucket?: string;
    s3_access_key?: string;
    s3_secret_key?: string;
    s3_region?: string;
    webdav_url?: string;
    webdav_username?: string;
    webdav_password?: string;
  }>;
}): Promise<void> {
  await request("/admin/storage", {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(config),
  });
}

// Storage File API
export async function uploadStorageFile(file: File, storageConfigId: number): Promise<SysFile> {
  const formData = new FormData();
  formData.append("file", file);
  formData.append("storage_config_id", String(storageConfigId));
  return request("/admin/storage/files", {
    method: "POST",
    body: formData,
    // Do NOT set Content-Type header - browser sets it with boundary for multipart
  });
}

export interface StorageFileListResponse {
  files: SysFile[];
  total: number;
  limit: number;
  offset: number;
}

export async function listStorageFiles(params?: { storage_config_id?: number; limit?: number; offset?: number }): Promise<StorageFileListResponse> {
  const searchParams = new URLSearchParams();
  if (params?.storage_config_id) searchParams.set("storage_config_id", String(params.storage_config_id));
  if (params?.limit) searchParams.set("limit", String(params.limit));
  if (params?.offset) searchParams.set("offset", String(params.offset));
  const query = searchParams.toString();
  return request(`/admin/storage/files${query ? `?${query}` : ""}`);
}

export async function deleteStorageFile(fileId: number): Promise<{ message: string }> {
  return request(`/admin/storage/files/${fileId}`, { method: "DELETE" });
}

export function getStorageFileDownloadUrl(fileId: number): string {
  // Returns the URL for direct download - caller handles auth header
  const base = PUBLIC_API_BASE || DEFAULT_LOCAL_API;
  return `${base}/admin/storage/files/${fileId}`;
}

// SMTP 配置 API
export async function testSMTPConnection(config: {
  host: string;
  port: string;
  username: string;
  password: string;
  from: string;
  from_name: string;
  use_tls: boolean;
}): Promise<{ success: boolean; message: string }> {
  return request("/admin/smtp/test", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(config),
  });
}

export async function saveSMTPConfig(config: {
  host: string;
  port: string;
  username: string;
  password: string;
  from: string;
  from_name: string;
  use_tls: boolean;
}): Promise<void> {
  await request("/admin/smtp", {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(config),
  });
}

export async function getSMTPConfig(): Promise<{
  configured: boolean;
  host?: string;
  port?: string;
  username?: string;
  from?: string;
  from_name?: string;
  use_tls?: boolean;
}> {
  return request("/admin/smtp", {
    method: "GET",
  });
}

export interface NotificationConfig {
  id?: number;
  channel: string;
  name: string;
  enabled: boolean;
  config?: Record<string, unknown>;
  status?: string;
  remark?: string;
}

export async function listNotificationConfigs(): Promise<NotificationConfig[]> {
  return request("/notifications/configs", { method: "GET" });
}

export async function createNotificationConfig(config: NotificationConfig): Promise<NotificationConfig> {
  return request("/notifications/configs", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(config),
  });
}

export async function updateNotificationConfig(id: number, config: Partial<NotificationConfig>): Promise<NotificationConfig> {
  return request(`/notifications/configs/${id}`, {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(config),
  });
}

export async function deleteNotificationConfig(id: number): Promise<void> {
  return request(`/notifications/configs/${id}`, { method: "DELETE" });
}

export async function sendSMTPNotification(to: string, subject: string, content: string, config?: Record<string, unknown>): Promise<void> {
  return request("/notifications/smtp/send", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ to, subject, content, config }),
  });
}

export async function sendSMSNotification(phone: string, template: Record<string, string>, config?: Record<string, unknown>): Promise<void> {
  return request("/notifications/sms/send", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ phone, template: template, config }),
  });
}

export async function sendTelegramNotification(chatId: string, text: string, config?: Record<string, unknown>): Promise<void> {
  return request("/notifications/telegram/send", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ chat_id: chatId, text, config }),
  });
}

export async function sendWebhookNotification(method: string, body: string, headers?: Record<string, string>, config?: Record<string, unknown>): Promise<void> {
  return request("/notifications/webhook/send", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ method, body, headers, config }),
  });
}

export async function testNotification(channel: string, to: string, config?: Record<string, unknown>): Promise<void> {
  return request("/notifications/test", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ channel, to, config }),
  });
}

// ============ 知识库 API ============

// 文档入知识库
export async function ingestDocument(documentId: number): Promise<{ message: string }> {
  return request("/knowledge/ingest", { method: "POST", body: JSON.stringify({ document_id: documentId }) });
}

// 知识库搜索
export async function searchKnowledge(query: string, limit?: number): Promise<{ results: SearchResult[] }> {
  const params = new URLSearchParams({ q: query });
  if (limit) params.set("limit", String(limit));
  return request(`/knowledge/search?${params}`);
}

// AI 问答
export async function chatWithKnowledge(question: string, sessionId?: string): Promise<ChatResponse> {
  return request("/knowledge/chat", { method: "POST", body: JSON.stringify({ question, session_id: sessionId }) });
}

// 知识库统计
export async function getKnowledgeStats(): Promise<{ documents: number; embeddings: number; messages: number }> {
  return request("/knowledge/stats");
}

// 全局搜索
export async function globalSearch(query: string, limit?: number): Promise<{ results: GlobalSearchResult[] }> {
  const params = new URLSearchParams({ q: query });
  if (limit) params.set("limit", String(limit));
  return request(`/search/global?${params}`);
}

// ============ 档案列配置 API ============

export async function fetchColumnConfig(subCategoryCode: string): Promise<{ column_keys: string[] }> {
  return request(`/archives/column-config?sub_category_code=${subCategoryCode}`);
}

export async function saveColumnConfig(subCategoryCode: string, columnKeys: string[]): Promise<{ message: string }> {
  return request('/archives/column-config', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ sub_category_code: subCategoryCode, column_keys: JSON.stringify(columnKeys) }),
  });
}

// ============ 模型配置 API ============

export async function listModelConfigs(configType?: string): Promise<ModelConfig[]> {
  const params = configType ? `?config_type=${configType}` : "";
  return request(`/settings/models${params}`);
}

export async function createModelConfig(config: Partial<ModelConfig>): Promise<ModelConfig> {
  return request("/settings/models", { method: "POST", body: JSON.stringify(config) });
}

export async function updateModelConfig(id: number, config: Partial<ModelConfig>): Promise<ModelConfig> {
  return request(`/settings/models/${id}`, { method: "PUT", body: JSON.stringify(config) });
}

export async function deleteModelConfig(id: number): Promise<{ message: string }> {
  return request(`/settings/models/${id}`, { method: "DELETE" });
}

export async function testModelConfig(id: number): Promise<{ success: boolean; message: string; latency?: number }> {
  return request(`/settings/models/${id}/test`, { method: "POST" });
}

export async function listBuiltInProviders(): Promise<Record<string, { name: string; endpoint: string; models_endpoint: string; auth_type: string }>> {
  return request("/settings/models/providers");
}

export async function listAvailableModels(configId: number): Promise<Array<{ id: string; name: string; disabled: boolean }>> {
  return request(`/settings/models/${configId}/available-models`);
}

export async function fetchModelsByEndpoint(endpoint: string, apiKey: string): Promise<Array<{ id: string; name: string }>> {
  const params = new URLSearchParams({ endpoint, api_key: apiKey });
  return request(`/settings/models/fetch-models?${params.toString()}`);
}

// ============ 类型定义 ============

export interface SearchResult {
  doc_id: number;
  score: number;
  snippet: string;
  title: string;
}

export interface GlobalSearchResult {
  module: string;
  id: number;
  title: string;
  snippet: string;
  score: number;
}

export interface ChatResponse {
  answer: string;
  sources: SearchResult[];
  session_id: string;
}

export interface ModelConfig {
  id: number;
  user_id: number;
  config_type: string;
  provider: string;
  model_name: string;
  api_key: string;
  api_endpoint: string;
  extra_params?: Record<string, unknown>;
  enabled: boolean;
  is_default: boolean;
  role?: string;
  priority?: number;
  is_built_in?: boolean;
  context_length?: string; // "256K", "128K", "8K"
  capabilities?: string; // "vision,tool_call"
  rate_limit_rpm?: number;
  rate_limit_tpm?: number;
  status?: string; // idle/success/error
  latency?: number; // ms
  available_models?: Array<{ id: string; name: string; disabled: boolean }>;
  created_at: string;
  updated_at: string;
}

// ============ 系统日志 API ============

export interface SystemLog {
  id: number;
  user_id?: number;
  user?: {
    id: number;
    username: string;
    full_name?: string;
  };
  action: string;
  resource?: string;
  resource_id?: string;
  method?: string;
  path?: string;
  ip_address: string;
  user_agent?: string;
  status: string;
  status_code?: number;
  error_msg?: string;
  duration?: number;
  details?: string;
  created_at: string;
  level?: string;
  trace_id?: string;
  source?: string;
  message?: string;
}

export interface SystemLogParams {
  log_type?: string;
  start_date?: string;
  end_date?: string;
  action?: string;
  user_id?: string;
  status?: string;
  level?: string;
  search?: string;
  page?: number;
  size?: number;
}

export interface SystemLogsResponse {
  data: SystemLog[];
  total: number;
  page: number;
  size: number;
}

export interface LogBackup {
  id: number;
  filename: string;
  file_path: string;
  file_size: number;
  record_count: number;
  backup_type: string;
  status: string;
  created_by: number;
  created_at: string;
}

export interface AlertRule {
  id: number;
  name: string;
  keywords: string[];
  threshold: number;
  time_window: number;
  enabled: boolean;
  notification_channel: string;
  created_by: number;
  created_at: string;
  updated_at: string;
}

export interface Notification {
  id: number;
  user_id: number;
  title: string;
  content: string;
  type: string;
  read: boolean;
  source: string;
  created_at: string;
}

export interface NotificationsResponse {
  data: Notification[];
  total: number;
  unread: number;
}

export async function fetchSystemLogs(params: SystemLogParams): Promise<SystemLogsResponse> {
  const searchParams = new URLSearchParams();
  if (params.log_type) searchParams.append("log_type", params.log_type);
  if (params.start_date) searchParams.append("start_date", params.start_date);
  if (params.end_date) searchParams.append("end_date", params.end_date);
  if (params.action) searchParams.append("action", params.action);
  if (params.user_id) searchParams.append("user_id", params.user_id);
  if (params.status) searchParams.append("status", params.status);
  if (params.level) searchParams.append("level", params.level);
  if (params.search) searchParams.append("search", params.search);
  if (params.page) searchParams.append("page", params.page.toString());
  if (params.size) searchParams.append("size", params.size.toString());
  
  return request<SystemLogsResponse>(`/logs?${searchParams.toString()}`);
}

export async function exportSystemLogs(params: SystemLogParams): Promise<Blob> {
  const searchParams = new URLSearchParams();
  if (params.log_type) searchParams.append("log_type", params.log_type);
  if (params.start_date) searchParams.append("start_date", params.start_date);
  if (params.end_date) searchParams.append("end_date", params.end_date);
  if (params.action) searchParams.append("action", params.action);
  if (params.user_id) searchParams.append("user_id", params.user_id);
  if (params.status) searchParams.append("status", params.status);
  if (params.level) searchParams.append("level", params.level);
  if (params.search) searchParams.append("search", params.search);
  
  return downloadBinary(`/logs/export?${searchParams.toString()}`);
}

export async function createLogBackup(): Promise<{ message: string }> {
  return request<{ message: string }>("/logs/backup", { method: "POST" });
}

export async function fetchLogBackups(): Promise<LogBackup[]> {
  return request<LogBackup[]>("/logs/backups");
}

export async function deleteLogBackup(id: number): Promise<{ message: string }> {
  return request<{ message: string }>(`/logs/backups/${id}`, { method: "DELETE" });
}

export async function cleanExpiredLogs(): Promise<{ deleted: number; message: string }> {
  return request<{ deleted: number; message: string }>("/logs/cleanup", { method: "POST" });
}

export async function updateBackupSettings(settings: {
  retention_days: number;
  auto_backup_enabled: boolean;
  backup_cron: string;
}): Promise<{ message: string }> {
  return request<{ message: string }>("/logs/backup-settings", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(settings),
  });
}

export async function getBackupSettings(): Promise<{
  retention_days: number;
  auto_backup_enabled: boolean;
  backup_cron: string;
}> {
  return request<{
    retention_days: number;
    auto_backup_enabled: boolean;
    backup_cron: string;
  }>("/logs/backup-settings");
}

export async function fetchAlertRules(): Promise<AlertRule[]> {
  return request<AlertRule[]>("/logs/alert-rules");
}

export async function createAlertRule(rule: Omit<AlertRule, "id" | "created_by" | "created_at" | "updated_at">): Promise<AlertRule> {
  return request<AlertRule>("/logs/alert-rules", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(rule),
  });
}

export async function updateAlertRule(id: number, rule: Partial<Omit<AlertRule, "id" | "created_by" | "created_at" | "updated_at">>): Promise<AlertRule> {
  return request<AlertRule>(`/logs/alert-rules/${id}`, {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(rule),
  });
}

export async function deleteAlertRule(id: number): Promise<{ message: string }> {
  return request<{ message: string }>(`/logs/alert-rules/${id}`, { method: "DELETE" });
}

export async function fetchNotifications(page: number = 1, size: number = 20): Promise<NotificationsResponse> {
  const params = new URLSearchParams({ page: page.toString(), size: size.toString() });
  return request<NotificationsResponse>(`/notifications?${params.toString()}`);
}

export async function getUnreadNotificationCount(): Promise<{ unread: number }> {
  return request<{ unread: number }>("/notifications/unread-count");
}

export async function markNotificationAsRead(id: number): Promise<{ status: string }> {
  return request<{ status: string }>(`/notifications/${id}/read`, { method: "PUT" });
}

export async function markAllNotificationsAsRead(): Promise<{ status: string }> {
  return request<{ status: string }>("/notifications/read-all", { method: "PUT" });
}

// ============ Model Usage Statistics API ============
export interface ModelUsageStatsResponse {
  total_calls: number;
  success_calls: number;
  failed_calls: number;
  success_rate: number;
  total_tokens: number;
  input_tokens: number;
  output_tokens: number;
  total_cost: number;
  avg_duration_ms: number;
  today_calls: number;
  today_cost: number;
  today_input_tokens: number;
  today_output_tokens: number;
  rpm: number;
  tpm: number;
}

export interface ModelUsageTrendItem {
  date: string;
  total_calls: number;
  success_calls: number;
  failed_calls: number;
  total_tokens: number;
  input_tokens: number;
  output_tokens: number;
  total_cost: number;
}

export interface ModelUsageByModelItem {
  model_name: string;
  config_type: string;
  provider: string;
  total_calls: number;
  success_calls: number;
  failed_calls: number;
  total_tokens: number;
  input_tokens: number;
  output_tokens: number;
  total_cost: number;
  avg_duration_ms: number;
  success_rate: number;
}

export async function fetchModelUsageStats(params?: {
  config_type?: string;
  model_name?: string;
  start_date?: string;
  end_date?: string;
}): Promise<ModelUsageStatsResponse> {
  const searchParams = new URLSearchParams();
  if (params?.config_type) searchParams.append("config_type", params.config_type);
  if (params?.model_name) searchParams.append("model_name", params.model_name);
  if (params?.start_date) searchParams.append("start_date", params.start_date);
  if (params?.end_date) searchParams.append("end_date", params.end_date);
  const qs = searchParams.toString();
  return request<ModelUsageStatsResponse>(`/settings/models/usage${qs ? `?${qs}` : ""}`);
}

export async function fetchModelUsageTrend(params?: {
  period?: string;
  config_type?: string;
  model_name?: string;
  start_date?: string;
  end_date?: string;
}): Promise<ModelUsageTrendItem[]> {
  const searchParams = new URLSearchParams();
  if (params?.period) searchParams.append("period", params.period);
  if (params?.config_type) searchParams.append("config_type", params.config_type);
  if (params?.model_name) searchParams.append("model_name", params.model_name);
  if (params?.start_date) searchParams.append("start_date", params.start_date);
  if (params?.end_date) searchParams.append("end_date", params.end_date);
  const qs = searchParams.toString();
  return request<ModelUsageTrendItem[]>(`/settings/models/usage/trend${qs ? `?${qs}` : ""}`);
}

export async function fetchModelUsageByModel(params?: {
  config_type?: string;
  model_name?: string;
  start_date?: string;
  end_date?: string;
}): Promise<ModelUsageByModelItem[]> {
  const searchParams = new URLSearchParams();
  if (params?.config_type) searchParams.append("config_type", params.config_type);
  if (params?.model_name) searchParams.append("model_name", params.model_name);
  if (params?.start_date) searchParams.append("start_date", params.start_date);
  if (params?.end_date) searchParams.append("end_date", params.end_date);
  const qs = searchParams.toString();
  return request<ModelUsageByModelItem[]>(`/settings/models/usage/by-model${qs ? `?${qs}` : ""}`);
}

// ============ 存储模块配置 API ============

export async function listStorageModules(): Promise<StorageModuleConfig[]> {
  return request<StorageModuleConfig[]>("/admin/storage/modules");
}

export async function createStorageModule(data: Partial<StorageModuleConfig>): Promise<StorageModuleConfig> {
  return request<StorageModuleConfig>("/admin/storage/modules", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(data),
  });
}

export async function updateStorageModule(id: number, data: Partial<StorageModuleConfig>): Promise<StorageModuleConfig> {
  return request<StorageModuleConfig>(`/admin/storage/modules/${id}`, {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(data),
  });
}

export async function deleteStorageModule(id: number): Promise<void> {
  await request(`/admin/storage/modules/${id}`, {
    method: "DELETE",
  }, false);
}

// ============ 存储规则 API（增强版） ============

export async function listStorageRulesEnhanced(params?: { module_code?: string; resource_type?: string }): Promise<StorageRule[]> {
  const searchParams = new URLSearchParams();
  if (params?.module_code) searchParams.set("module_code", params.module_code);
  if (params?.resource_type) searchParams.set("resource_type", params.resource_type);
  const query = searchParams.toString() ? `?${searchParams.toString()}` : "";
  return request<StorageRule[]>(`/admin/storage/rules${query}`);
}

export async function createStorageRuleEnhanced(data: Partial<StorageRule>): Promise<StorageRule> {
  return request<StorageRule>("/admin/storage/rules", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(data),
  });
}

export async function updateStorageRuleEnhanced(id: number, data: Partial<StorageRule>): Promise<StorageRule> {
  return request<StorageRule>(`/admin/storage/rules/${id}`, {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(data),
  });
}

export async function deleteStorageRuleEnhanced(id: number): Promise<void> {
  await request(`/admin/storage/rules/${id}`, {
    method: "DELETE",
  }, false);
}
