"use client";

import { useEffect, useState } from "react";

const REDUCED_MOTION_QUERY = "(prefers-reduced-motion: reduce)";

/**
 * 同步读取系统"减少动态"偏好。
 * 注意：jsdom / SSR 环境没有 matchMedia，必须防御性降级为 false，
 * 否则单元测试与服务器渲染会直接抛 TypeError。
 */
function readReducedMotionPreference(): boolean {
  if (typeof window === "undefined" || typeof window.matchMedia !== "function") {
    return false;
  }
  return window.matchMedia(REDUCED_MOTION_QUERY).matches;
}

/** 订阅系统 prefers-reduced-motion 变化；不支持的环境安全降级为 false */
export function usePrefersReducedMotion(): boolean {
  const [reduced, setReduced] = useState(readReducedMotionPreference);

  useEffect(() => {
    if (typeof window.matchMedia !== "function") return;
    const mediaQuery = window.matchMedia(REDUCED_MOTION_QUERY);
    const syncPreference = () => setReduced(mediaQuery.matches);
    syncPreference();
    mediaQuery.addEventListener("change", syncPreference);
    return () => mediaQuery.removeEventListener("change", syncPreference);
  }, []);

  return reduced;
}
