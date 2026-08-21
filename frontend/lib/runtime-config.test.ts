import { afterEach, describe, expect, test, vi } from "vitest";

import { getRuntimeConfig, type RuntimeConfig } from "./runtime-config";

/**
 * 测试间全局运行时配置清理：
 * 清除 window 与 globalThis 上的运行时配置，避免测试间相互污染。
 */
afterEach(() => {
  // 先恢复可能被 stub 的 window（SSR 用例会 stub 为 undefined），再清理运行时配置
  vi.unstubAllGlobals();
  delete window.__RUNTIME_CONFIG__;
  delete (globalThis as typeof globalThis & { __RUNTIME_CONFIG__?: RuntimeConfig }).__RUNTIME_CONFIG__;
});

describe("getRuntimeConfig 运行时配置读取", () => {
  test("读取浏览器运行时 API 配置", () => {
    window.__RUNTIME_CONFIG__ = { API_BASE: "https://example.test/api" };
    expect(getRuntimeConfig()).toEqual({ API_BASE: "https://example.test/api" });
  });

  test("浏览器端未注入配置时返回空对象", () => {
    expect(getRuntimeConfig()).toEqual({});
  });

  test("SSR 端读取 globalThis 配置", () => {
    (globalThis as typeof globalThis & { __RUNTIME_CONFIG__?: RuntimeConfig }).__RUNTIME_CONFIG__ = {
      API_BASE_IP: "http://127.0.0.1:8080/api",
    };
    vi.stubGlobal("window", undefined);
    expect(getRuntimeConfig()).toEqual({ API_BASE_IP: "http://127.0.0.1:8080/api" });
  });
});
