"use client";

import { useCallback, useRef, useState } from "react";
import { toast } from "sonner";
import { Camera, RotateCcw, X } from "lucide-react";
import { Button } from "@/components/ui/button";
import { UserAvatar } from "@/components/avatar/user-avatar";
import { useUserAvatar } from "@/hooks/use-user-avatar";
import {
  AVATAR_MAX_BYTES,
  AVATAR_SIZE,
  computeCropRect,
  cropImageToWebP,
  validateAvatarFile,
  type Point,
} from "@/lib/avatar";
import { uploadUserAvatar, resetUserAvatar } from "@/lib/avatar-api";

interface AvatarEditorProps {
  /** 用于首字母回退与稳定配色的显示名 */
  name: string;
}

interface CropImageState {
  /** 原图 object URL（裁剪完成后 revoke） */
  url: string;
  img: HTMLImageElement;
  width: number;
  height: number;
}

interface FeedbackState {
  type: "success" | "error";
  text: string;
}

/** 裁剪视口边长（px），与导出尺寸一致，保证所见即所得 */
const CROP_VIEWPORT = AVATAR_SIZE;
const SCALE_MIN = 1;
const SCALE_MAX = 3;

/**
 * 头像编辑器：选择 JPG/PNG/WebP → 1:1 方形裁剪（拖动平移 + 滑块缩放）→
 * 导出 256×256 WebP 上传；并提供恢复默认头像。
 * 服务端仍是最终校验，前端校验仅用于提前给出友好中文提示。
 */
export function AvatarEditor({ name }: AvatarEditorProps) {
  const { refresh } = useUserAvatar();

  const fileInputRef = useRef<HTMLInputElement>(null);
  const dragRef = useRef<{ startX: number; startY: number; startOffset: Point } | null>(null);

  const [isUploading, setIsUploading] = useState(false);
  const [isResetting, setIsResetting] = useState(false);
  const [cropImage, setCropImage] = useState<CropImageState | null>(null);
  const [scale, setScale] = useState(1);
  const [offset, setOffset] = useState<Point>({ x: 0, y: 0 });
  const [feedback, setFeedback] = useState<FeedbackState | null>(null);

  /** 限制平移范围：图片必须始终覆盖整个视口（cover） */
  const clampOffset = useCallback((drawWidth: number, drawHeight: number, next: Point): Point => {
    return {
      x: Math.min(Math.max(next.x, CROP_VIEWPORT - drawWidth), 0),
      y: Math.min(Math.max(next.y, CROP_VIEWPORT - drawHeight), 0),
    };
  }, []);

  const pickFile = () => {
    setFeedback(null);
    fileInputRef.current?.click();
  };

  const handleFileChange = (event: React.ChangeEvent<HTMLInputElement>) => {
    const file = event.target.files?.[0];
    // 重置 input，保证同一文件再次选择也能触发 change
    event.target.value = "";
    if (!file) return;

    const validation = validateAvatarFile(file);
    if (!validation.ok) {
      setFeedback({ type: "error", text: validation.error });
      toast.error(validation.error);
      return;
    }

    const url = URL.createObjectURL(file);
    const img = new Image();
    img.onload = () => {
      // 过小图片放大后仍模糊，提前给出友好提示
      if (img.naturalWidth < CROP_VIEWPORT || img.naturalHeight < CROP_VIEWPORT) {
        URL.revokeObjectURL(url);
        setFeedback({ type: "error", text: "图片尺寸过小，建议至少 256×256 像素" });
        toast.error("图片尺寸过小，建议至少 256×256 像素");
        return;
      }
      const coverScale = Math.max(CROP_VIEWPORT / img.naturalWidth, CROP_VIEWPORT / img.naturalHeight);
      setCropImage({ url, img, width: img.naturalWidth, height: img.naturalHeight });
      setScale(Math.max(coverScale, SCALE_MIN));
      setOffset({ x: 0, y: 0 });
      setFeedback(null);
    };
    img.onerror = () => {
      URL.revokeObjectURL(url);
      toast.error("图片读取失败，请更换图片");
    };
    img.src = url;
  };

  const cancelCrop = () => {
    if (cropImage) {
      URL.revokeObjectURL(cropImage.url);
    }
    setCropImage(null);
  };

  const confirmUpload = async () => {
    if (!cropImage) return;
    setIsUploading(true);
    try {
      const rect = computeCropRect(cropImage.width, cropImage.height, CROP_VIEWPORT, scale, offset);
      if (!rect) {
        throw new Error("图片尺寸无效，请重新选择");
      }
      const blob = await cropImageToWebP(cropImage.img, rect);
      if (blob.size > AVATAR_MAX_BYTES) {
        throw new Error("裁剪后的图片超过 2 MB，请缩小图片后重试");
      }
      const file = new File([blob], "avatar.webp", { type: "image/webp" });
      await uploadUserAvatar(file);
      refresh();
      toast.success("头像已更新");
      cancelCrop();
    } catch (error) {
      const message = error instanceof Error ? error.message : "头像上传失败，请重试";
      setFeedback({ type: "error", text: message });
      toast.error(message);
    } finally {
      setIsUploading(false);
    }
  };

  const handleReset = async () => {
    setIsResetting(true);
    setFeedback(null);
    try {
      await resetUserAvatar();
      refresh();
      toast.success("已恢复默认头像");
    } catch (error) {
      const message = error instanceof Error ? error.message : "恢复默认头像失败，请重试";
      setFeedback({ type: "error", text: message });
      toast.error(message);
    } finally {
      setIsResetting(false);
    }
  };

  // ========== 拖动平移 ==========
  const handlePointerDown = (event: React.PointerEvent<HTMLDivElement>) => {
    if (!cropImage) return;
    dragRef.current = {
      startX: event.clientX,
      startY: event.clientY,
      startOffset: { ...offset },
    };
    event.currentTarget.setPointerCapture(event.pointerId);
  };

  const handlePointerMove = (event: React.PointerEvent<HTMLDivElement>) => {
    if (!dragRef.current || !cropImage) return;
    const drawWidth = cropImage.width * scale;
    const drawHeight = cropImage.height * scale;
    setOffset(
      clampOffset(drawWidth, drawHeight, {
        x: dragRef.current.startOffset.x + (event.clientX - dragRef.current.startX),
        y: dragRef.current.startOffset.y + (event.clientY - dragRef.current.startY),
      }),
    );
  };

  const handlePointerUp = () => {
    dragRef.current = null;
  };

  const drawWidth = cropImage ? cropImage.width * scale : 0;
  const drawHeight = cropImage ? cropImage.height * scale : 0;

  return (
    <div className="space-y-3">
      <input
        ref={fileInputRef}
        type="file"
        accept="image/jpeg,image/png,image/webp"
        className="hidden"
        onChange={handleFileChange}
        aria-label="选择头像图片"
      />

      <div className="flex items-center gap-4">
        <div className="relative">
          <UserAvatar name={name} className="h-20 w-20 rounded-full" />
          <Button
            type="button"
            size="icon"
            variant="secondary"
            className="absolute -bottom-1 -right-1 h-8 w-8 rounded-full"
            onClick={pickFile}
            disabled={isUploading || isResetting}
            aria-label="上传新头像"
            title="上传新头像"
          >
            <Camera className="h-4 w-4" />
          </Button>
        </div>
        <div className="text-sm text-muted-foreground">
          <p>点击相机图标上传新头像</p>
          <p>支持 JPG、PNG、WebP 格式，1:1 方形裁剪</p>
        </div>
      </div>

      {feedback && (
        <p
          role="alert"
          className={`text-sm ${feedback.type === "error" ? "text-destructive" : "text-emerald-600"}`}
        >
          {feedback.text}
        </p>
      )}

      {cropImage && (
        <div className="space-y-3 rounded-lg border bg-muted/40 p-3">
          <p className="text-sm font-medium">裁剪头像</p>
          {/* 裁剪视口：固定 256px 方形，拖动图片平移，滑块缩放 */}
          <div
            className="relative mx-auto h-64 w-64 overflow-hidden rounded-full border-2 border-primary"
            style={{ touchAction: "none" }}
            onPointerDown={handlePointerDown}
            onPointerMove={handlePointerMove}
            onPointerUp={handlePointerUp}
            onPointerCancel={handlePointerUp}
            role="img"
            aria-label="拖动图片调整裁剪位置"
          >
            {/* eslint-disable-next-line @next/next/no-img-element -- 本地待裁剪原图，无需 next/image 优化 */}
            <img
              src={cropImage.url}
              alt="待裁剪图片"
              draggable={false}
              className="pointer-events-none absolute max-w-none select-none"
              style={{
                width: drawWidth,
                height: drawHeight,
                left: (CROP_VIEWPORT - drawWidth) / 2 + offset.x,
                top: (CROP_VIEWPORT - drawHeight) / 2 + offset.y,
              }}
            />
          </div>

          <div className="flex items-center gap-3">
            <span className="text-xs text-muted-foreground">缩放</span>
            <input
              type="range"
              min={SCALE_MIN}
              max={SCALE_MAX}
              step={0.05}
              value={scale}
              onChange={(event) => setScale(Number(event.target.value))}
              className="flex-1"
              aria-label="缩放比例"
            />
          </div>

          <div className="flex gap-2">
            <Button type="button" size="sm" className="flex-1" onClick={confirmUpload} disabled={isUploading}>
              {isUploading ? "上传中..." : "确认上传"}
            </Button>
            <Button type="button" size="sm" variant="outline" onClick={cancelCrop} disabled={isUploading}>
              <X className="mr-1 h-4 w-4" />
              取消
            </Button>
          </div>
        </div>
      )}

      <Button type="button" variant="outline" size="sm" onClick={handleReset} disabled={isResetting || isUploading}>
        <RotateCcw className="mr-1 h-4 w-4" />
        {isResetting ? "恢复中..." : "恢复默认头像"}
      </Button>
    </div>
  );
}