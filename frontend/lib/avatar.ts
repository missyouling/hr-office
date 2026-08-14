"use client";

/**
 * 头像相关纯函数：文件校验、1:1 裁剪矩形计算、Canvas 导出 WebP、首字母回退。
 * 全部为无副作用纯函数（cropImageToWebP 除外，它操作 Canvas），便于单元测试。
 */

/** 导出头像边长（1:1 方形） */
export const AVATAR_SIZE = 256;

/** 服务端头像文件上限 2 MiB（与后端 avatarMaxFileBytes 一致） */
export const AVATAR_MAX_BYTES = 2 * 1024 * 1024;

/** 允许上传的图片 MIME 类型（与后端 avatarExtByContentType 一致） */
export const ALLOWED_AVATAR_TYPES = ["image/jpeg", "image/png", "image/webp"] as const;

/** 裁剪矩形（原图坐标系） */
export interface CropRect {
  x: number;
  y: number;
  width: number;
  height: number;
}

export interface Point {
  x: number;
  y: number;
}

export type AvatarFileValidation =
  | { ok: true; file: File }
  | { ok: false; error: string };

/**
 * 上传前校验：仅允许 JPG/PNG/WebP，且不超过 2 MiB。
 * 服务端仍会做最终校验，这里只是提前给出友好中文提示。
 */
export function validateAvatarFile(file: File): AvatarFileValidation {
  if (!ALLOWED_AVATAR_TYPES.includes(file.type as (typeof ALLOWED_AVATAR_TYPES)[number])) {
    return { ok: false, error: "仅支持 JPG、PNG、WebP 格式的图片" };
  }
  if (file.size > AVATAR_MAX_BYTES) {
    return { ok: false, error: "图片大小不能超过 2 MB" };
  }
  return { ok: true, file };
}

/**
 * 计算 1:1 裁剪矩形（原图坐标系）。
 *
 * 视口内图片以 scale 缩放、offset 平移后 cover 显示：
 *   绘制尺寸 = 原图尺寸 × scale
 *   图片左上角 = (视口 - 绘制尺寸) / 2 + offset
 * 裁剪矩形即视口在原图中的投影，并 clamp 到图片边界，保证不越界。
 *
 * @param imageWidth  原图宽度（px）
 * @param imageHeight 原图高度（px）
 * @param viewportSize 方形视口边长（px）
 * @param scale       缩放比例（>= 1，cover 基准）
 * @param offset      平移偏移（px，相对居中位置）
 * @returns 原图坐标系下的裁剪矩形；参数非法时返回 null
 */
export function computeCropRect(
  imageWidth: number,
  imageHeight: number,
  viewportSize: number,
  scale: number,
  offset: Point,
): CropRect | null {
  if (imageWidth <= 0 || imageHeight <= 0 || viewportSize <= 0 || scale <= 0) {
    return null;
  }

  const drawWidth = imageWidth * scale;
  const drawHeight = imageHeight * scale;

  // 图片无法覆盖整个视口时不存在有效 1:1 裁剪（cover 语义）
  if (drawWidth < viewportSize || drawHeight < viewportSize) {
    return null;
  }

  // 图片左上角相对视口的位置（居中 + 用户平移）
  const left = (viewportSize - drawWidth) / 2 + offset.x;
  const top = (viewportSize - drawHeight) / 2 + offset.y;

  // 裁剪矩形在原图中的起点与尺寸
  const cropWidth = viewportSize / scale;
  const cropHeight = viewportSize / scale;

  // clamp：裁剪矩形必须完全落在图片内
  const minLeft = viewportSize - drawWidth; // 图片右边缘贴视口右边缘
  const maxLeft = 0; // 图片左边缘贴视口左边缘
  const minTop = viewportSize - drawHeight;
  const maxTop = 0;

  const clampedLeft = Math.min(Math.max(left, minLeft), maxLeft);
  const clampedTop = Math.min(Math.max(top, minTop), maxTop);

  return {
    x: -clampedLeft / scale,
    y: -clampedTop / scale,
    width: cropWidth,
    height: cropHeight,
  };
}

/**
 * 将裁剪矩形绘制到 256×256 Canvas 并导出 WebP Blob。
 * 调用方需保证 image 已加载完成（complete 且 naturalWidth > 0）。
 */
export function cropImageToWebP(
  image: HTMLImageElement,
  rect: CropRect,
  targetSize: number = AVATAR_SIZE,
  quality = 0.9,
): Promise<Blob> {
  return new Promise((resolve, reject) => {
    const canvas = document.createElement("canvas");
    canvas.width = targetSize;
    canvas.height = targetSize;
    const ctx = canvas.getContext("2d");
    if (!ctx) {
      reject(new Error("当前环境不支持图片裁剪"));
      return;
    }
    ctx.drawImage(image, rect.x, rect.y, rect.width, rect.height, 0, 0, targetSize, targetSize);
    canvas.toBlob(
      (blob) => {
        if (blob) {
          resolve(blob);
        } else {
          reject(new Error("头像导出失败，请重试"));
        }
      },
      "image/webp",
      quality,
    );
  });
}

/**
 * 取显示名首字符用于回退头像：中文取首字，拉丁取首字母大写，空串返回 "?"。
 */
export function getInitial(name: string): string {
  const trimmed = (name ?? "").trim();
  if (!trimmed) return "?";
  const first = Array.from(trimmed)[0];
  // 拉丁字母转大写，其余字符（中文等）原样保留
  return /[a-zA-Z]/.test(first) ? first.toUpperCase() : first;
}

/** 首字母回退头像的稳定配色（按名字哈希选取，同一用户颜色恒定） */
const INITIAL_COLOR_CLASSES = [
  "bg-rose-500/20 text-rose-700 dark:text-rose-300",
  "bg-blue-500/20 text-blue-700 dark:text-blue-300",
  "bg-emerald-500/20 text-emerald-700 dark:text-emerald-300",
  "bg-amber-500/20 text-amber-700 dark:text-amber-300",
  "bg-violet-500/20 text-violet-700 dark:text-violet-300",
  "bg-cyan-500/20 text-cyan-700 dark:text-cyan-300",
] as const;

/** 按名字哈希稳定选取配色类名（不依赖外部服务）。 */
export function getInitialColorClass(name: string): string {
  const trimmed = name ?? "";
  let hash = 0;
  for (let i = 0; i < trimmed.length; i += 1) {
    hash = (hash * 31 + trimmed.charCodeAt(i)) >>> 0;
  }
  return INITIAL_COLOR_CLASSES[hash % INITIAL_COLOR_CLASSES.length];
}