"use client";

import { fetchWithAuth, request } from "./api";

/**
 * 头像元数据，与后端 avatarMetadataResponse 对齐。
 * 头像二进制本身通过 GET /api/user/avatar 获取，不写入 User 类型。
 */
export interface AvatarMetadata {
  type: string;
  seed: string;
  custom_file_id?: number;
  custom_content_type?: string;
  content_type: string;
}

/**
 * 拉取当前登录用户头像二进制（默认 SVG 或自定义图片）。
 * 复用 fetchWithAuth：自动携带 Bearer token，401 时自动刷新 token 后重试。
 * 注意：token 只进请求头，绝不拼进 URL。
 */
export async function fetchUserAvatar(): Promise<Blob> {
  const res = await fetchWithAuth("/api/user/avatar", { cache: "no-store" });
  if (!res.ok) {
    throw new Error(await extractAvatarError(res, "头像加载失败"));
  }
  return res.blob();
}

/** 上传自定义头像（multipart 字段 file），服务端仍是最终校验方。 */
export async function uploadUserAvatar(file: File): Promise<AvatarMetadata> {
  const formData = new FormData();
  formData.append("file", file);
  return request<AvatarMetadata>("/user/avatar", {
    method: "POST",
    body: formData,
  });
}

/** 恢复默认头像。 */
export async function resetUserAvatar(): Promise<AvatarMetadata> {
  return request<AvatarMetadata>("/user/avatar", { method: "DELETE" });
}

/** 从非 2xx 响应中提取后端错误文案（JSON error 字段优先）。 */
async function extractAvatarError(res: Response, fallback: string): Promise<string> {
  try {
    const data = await res.json();
    if (data && typeof data.error === "string" && data.error) {
      return data.error;
    }
  } catch {
    // 响应体不是 JSON（如网关错误页），忽略
  }
  return fallback;
}