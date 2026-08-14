"use client";

import { useUserAvatar } from "@/hooks/use-user-avatar";
import { getInitial, getInitialColorClass } from "@/lib/avatar";
import { cn } from "@/lib/utils";

interface UserAvatarProps {
  /** 用于首字母回退与稳定配色；优先传 user.full_name，其次 username */
  name: string;
  /** 尺寸与圆角样式，如 "h-8 w-8 rounded-full" */
  className?: string;
  alt?: string;
}

/**
 * 鉴权头像展示组件：通过 token 拉取头像 Blob 并显示。
 * 状态覆盖：
 * - loading：灰色脉冲占位
 * - error / 未登录（idle）：本地首字母回退（稳定配色，不依赖外部服务）
 * - ready：显示服务端头像
 */
export function UserAvatar({ name, className, alt = "avatar" }: UserAvatarProps) {
  const { src, status } = useUserAvatar();

  if (status === "loading") {
    return <div aria-label="头像加载中" className={cn("animate-pulse bg-muted", className)} />;
  }

  if (status === "ready" && src) {
    // eslint-disable-next-line @next/next/no-img-element -- 头像为鉴权 Blob object URL，无法走 next/image 优化
    return <img src={src} alt={alt} className={cn("bg-muted object-cover", className)} />;
  }

  // 加载失败 / 未登录：本地首字母回退
  return (
    <div
      aria-label={`${name || "用户"}的头像`}
      className={cn(
        "flex select-none items-center justify-center font-medium",
        getInitialColorClass(name),
        className,
      )}
    >
      {getInitial(name)}
    </div>
  );
}