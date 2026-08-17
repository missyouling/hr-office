import { describe, expect, test } from "vitest";

import { clampDockPosition, parseDockPosition, parseMobileExpanded } from "./preferences";

describe("Dock 偏好", () => {
  test("将拖动位置限制在视口安全范围内", () => {
    expect(clampDockPosition({ left: -20, top: 900 }, { width: 800, height: 600 }, { width: 200, height: 48 })).toEqual({ left: 8, top: 544 });
  });

  test("在 Dock 大于视口时仍返回安全位置", () => {
    expect(clampDockPosition({ left: 100, top: 100 }, { width: 100, height: 80 }, { width: 240, height: 48 })).toEqual({ left: 8, top: 24 });
  });

  test("拒绝无效的持久化位置", () => {
    expect(parseDockPosition({ left: "10", top: 20 })).toBeNull();
    expect(parseDockPosition({ left: 10, top: Number.NaN })).toBeNull();
    expect(parseDockPosition({ left: 10, top: 20 })).toEqual({ left: 10, top: 20 });
  });

  test("仅接受布尔类型的移动端展开偏好", () => {
    expect(parseMobileExpanded(true)).toBe(true);
    expect(parseMobileExpanded(false)).toBe(false);
    expect(parseMobileExpanded("true")).toBeNull();
    expect(parseMobileExpanded(1)).toBeNull();
    expect(parseMobileExpanded(null)).toBeNull();
  });
});
