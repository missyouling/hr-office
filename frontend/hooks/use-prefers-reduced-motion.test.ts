import { act, renderHook } from "@testing-library/react";
import { afterEach, describe, expect, test, vi } from "vitest";
import { usePrefersReducedMotion } from "@/hooks/use-prefers-reduced-motion";

type MediaListener = (event: { matches: boolean }) => void;

/** 构造可编程的 matchMedia 桩：记录 change 监听器并维护动态 matches，便于模拟系统偏好切换 */
function stubMatchMedia(initialMatches: boolean) {
  let currentMatches = initialMatches;
  const listeners = new Set<MediaListener>();
  Object.defineProperty(window, "matchMedia", {
    writable: true,
    configurable: true,
    value: (query: string) => ({
      // 动态读取当前偏好，模拟真实 MediaQueryList 行为
      get matches() {
        return query.includes("prefers-reduced-motion") && currentMatches;
      },
      media: query,
      onchange: null,
      addEventListener: (_: string, listener: MediaListener) => listeners.add(listener),
      removeEventListener: (_: string, listener: MediaListener) => listeners.delete(listener),
      addListener: vi.fn(),
      removeListener: vi.fn(),
      dispatchEvent: vi.fn(),
    }),
  });
  return (matches: boolean) => {
    currentMatches = matches;
    listeners.forEach((listener) => listener({ matches }));
  };
}

afterEach(() => {
  Reflect.deleteProperty(window, "matchMedia");
});

describe("usePrefersReducedMotion", () => {
  test("无 matchMedia 的环境（jsdom/SSR）安全降级为 false 且不抛错", () => {
    const { result } = renderHook(() => usePrefersReducedMotion());
    expect(result.current).toBe(false);
  });

  test("系统开启减少动态时返回 true", () => {
    stubMatchMedia(true);
    const { result } = renderHook(() => usePrefersReducedMotion());
    expect(result.current).toBe(true);
  });

  test("系统关闭减少动态时返回 false", () => {
    stubMatchMedia(false);
    const { result } = renderHook(() => usePrefersReducedMotion());
    expect(result.current).toBe(false);
  });

  test("运行中切换系统偏好会同步更新", () => {
    const emitChange = stubMatchMedia(false);
    const { result } = renderHook(() => usePrefersReducedMotion());
    expect(result.current).toBe(false);

    act(() => emitChange(true));
    expect(result.current).toBe(true);

    act(() => emitChange(false));
    expect(result.current).toBe(false);
  });
});
