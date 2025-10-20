"use client";

import type {
  BatchUploadItem,
  Part,
  Period,
  PeriodSummary,
  PersonalCharge,
  RosterEntry,
  Scheme,
  SourceFile,
  UnitCharge,
} from "./types";

const API_BASE =
  process.env.NEXT_PUBLIC_API_BASE_URL?.replace(/\/+$/, "") ||
  "http://localhost:8080/api";

async function request<T>(
  path: string,
  init?: RequestInit,
  expectJson = true,
): Promise<T> {
  const url = `${API_BASE}${path}`;
  console.log(`Making request to: ${url}`);
  console.log(`API_BASE is: ${API_BASE}`);

  const res = await fetch(url, {
    ...init,
    headers: {
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

  const res = await fetch(`${API_BASE}/periods/${periodId}/files`, {
    method: "POST",
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

  const res = await fetch(`${API_BASE}/periods/${periodId}/roster`, {
    method: "POST",
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
  const res = await fetch(`${API_BASE}/periods/${periodId}/roster/import`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
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

  const res = await fetch(`${API_BASE}/periods/${periodId}/files/batch`, {
    method: "POST",
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
  const res = await fetch(
    `${API_BASE}/periods/${periodId}/charges/export?part=${part}`,
    {
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
  const res = await fetch(`${API_BASE}/roster-template`, {
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
  const res = await fetch(
    `${API_BASE}/periods/${periodId}/charges/scheme/export?scheme=${scheme}&part=${part}`,
    {
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

  const res = await fetch(`${API_BASE}/periods/${periodId}/adjustments/batch`, {
    method: "POST",
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
