import { render, screen } from "@testing-library/react";
import { readFileSync } from "node:fs";
import { join } from "node:path";
import { beforeEach, describe, expect, test, vi } from "vitest";
import { GlobalSearch } from "@/components/global-search";

vi.mock("@/lib/api", () => ({
  globalSearch: vi.fn().mockResolvedValue({ results: [] }),
}));

describe("GlobalSearch（全局搜索体验）", () => {
  const onNavigate = vi.fn();
  const onOpenChange = vi.fn();

  beforeEach(() => {
    vi.clearAllMocks();
    document.body.innerHTML = "";
  });

  test("遮罩半透明加深且背景模糊：bg-black/40 + backdrop-blur-sm", () => {
    render(<GlobalSearch onNavigate={onNavigate} open onOpenChange={onOpenChange} hideTrigger />);
    // Radix Portal 渲染到 document.body：直接按类查找遮罩层
    const overlay = document.querySelector(".backdrop-blur-sm");
    expect(overlay).not.toBeNull();
    expect(overlay).toHaveClass("bg-black/40");
    expect(overlay).toHaveClass("backdrop-blur-sm");
  });

  test("内容面板居中偏上且为圆角描边阴影卡片：top-[20%] + rounded-2xl", () => {
    render(<GlobalSearch onNavigate={onNavigate} open onOpenChange={onOpenChange} hideTrigger />);
    const content = screen.getByText("全局搜索").closest('[data-slot="dialog-content"], .top-\\[20\\%\\]');
    expect(content).not.toBeNull();
    expect(content).toHaveClass("top-[20%]");
    expect(content).toHaveClass("rounded-2xl");
    expect(content).toHaveClass("border-border/70");
    expect(content).toHaveClass("shadow-[0_24px_60px_-24px_rgba(0,0,0,0.65)]");
  });

  test("打开即自动聚焦搜索输入框（光标就位可直接输入）", () => {
    render(<GlobalSearch onNavigate={onNavigate} open onOpenChange={onOpenChange} hideTrigger />);
    const input = screen.getByPlaceholderText("搜索档案、员工、宿舍... (Cmd+K)");
    expect(document.activeElement).toBe(input);
  });

  test("源码契约防回归：遮罩模糊与居中偏上类不被误删", () => {
    const source = readFileSync(join(process.cwd(), "components", "global-search.tsx"), "utf-8");
    expect(source).toContain("bg-black/40");
    expect(source).toContain("backdrop-blur-sm");
    expect(source).toContain("top-[20%]");
    expect(source).toContain("rounded-2xl");
    expect(source).toContain("autoFocus");
  });
});
