"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import { useAuth } from "@/lib/auth";
import { fetchUserAvatar } from "@/lib/avatar-api";

/** 头像更新广播事件：上传/恢复默认成功后派发，所有头像实例统一刷新。 */
export const AVATAR_UPDATED_EVENT = "avatar:updated";

export type AvatarStatus = "idle" | "loading" | "ready" | "error";

export interface UseUserAvatarResult {
  /** 头像 object URL；未就绪时为 null */
  src: string | null;
  /** idle=未登录不加载；loading=请求中；ready=已就绪；error=加载失败（回退首字母） */
  status: AvatarStatus;
  /** 重新拉取头像并广播，供上传/恢复默认后调用 */
  refresh: () => void;
}

/**
 * 用当前登录 token 拉取头像 Blob 并生成 object URL。
 * - 组件卸载 / 头像切换时 revokeObjectURL，避免内存泄漏；
 * - 未登录（idle）不发起请求；
 * - 加载失败进入 error，由展示组件回退本地首字母；
 * - 监听 AVATAR_UPDATED_EVENT，上传/恢复默认后自动刷新。
 */
export function useUserAvatar(): UseUserAvatarResult {
  const { isAuthenticated } = useAuth();
  const [src, setSrc] = useState<string | null>(null);
  const [status, setStatus] = useState<AvatarStatus>("idle");

  // 当前 object URL 引用，用于切换/卸载时统一 revoke
  const urlRef = useRef<string | null>(null);
  // 请求序号：丢弃过期响应，避免快速切换时旧结果覆盖新结果
  const requestIdRef = useRef(0);

  const replaceUrl = useCallback((url: string | null) => {
    if (urlRef.current) {
      URL.revokeObjectURL(urlRef.current);
    }
    urlRef.current = url;
    setSrc(url);
  }, []);

  const load = useCallback(async () => {
    const requestId = ++requestIdRef.current;
    setStatus("loading");
    try {
      const blob = await fetchUserAvatar();
      if (requestId !== requestIdRef.current) return; // 已有更新的请求，丢弃本次结果
      replaceUrl(URL.createObjectURL(blob));
      setStatus("ready");
    } catch (error) {
      if (requestId !== requestIdRef.current) return;
      console.error("头像加载失败:", error);
      replaceUrl(null);
      setStatus("error");
    }
  }, [replaceUrl]);

  // 登录状态变化时加载/清空
  useEffect(() => {
    if (!isAuthenticated) {
      requestIdRef.current += 1; // 作废进行中的请求
      replaceUrl(null);
      setStatus("idle");
      return;
    }
    load();
  }, [isAuthenticated, load, replaceUrl]);

  // 监听头像更新事件（上传/恢复默认后由 AvatarEditor 派发）
  useEffect(() => {
    const handler = () => load();
    window.addEventListener(AVATAR_UPDATED_EVENT, handler);
    return () => window.removeEventListener(AVATAR_UPDATED_EVENT, handler);
  }, [load]);

  // 组件卸载时释放 object URL
  useEffect(() => {
    return () => {
      requestIdRef.current += 1;
      if (urlRef.current) {
        URL.revokeObjectURL(urlRef.current);
        urlRef.current = null;
      }
    };
  }, []);

  const refresh = useCallback(() => {
    window.dispatchEvent(new Event(AVATAR_UPDATED_EVENT));
  }, []);

  return { src, status, refresh };
}