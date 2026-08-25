import { render, screen } from "@testing-library/react";
import { afterEach, describe, expect, test, vi } from "vitest";
import { RollingText } from "@/components/layout/rolling-text";

/** 构造可编程的 matchMedia 桩，控制"减少动态"偏好 */
function stubMatchMedia(matches: boolean) {
  Object.defineProperty(window, "matchMedia", {
    writable: true,
    configurable: true,
    value: (query: string) => ({
      matches: query.includes("prefers-reduced-motion") && matches,
      media: query,
      onchange: null,
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
      addListener: vi.fn(),
      removeListener: vi.fn(),
      dispatchEvent: vi.fn(),
    }),
  });
}

afterEach(() => {
  Reflect.deleteProperty(window, "matchMedia");
});

describe("RollingText", () => {
  test("逐字渲染动画字符并保留 sr-only 完整可访问名称", () => {
    stubMatchMedia(false);
    const { container } = render(<RollingText text="人事行政管理" />);

    expect(screen.getByText("人事行政管理")).toHaveClass("sr-only");
    // 动画字符对辅助技术隐藏，由 sr-only 提供可访问名称
    const visual = container.querySelector('[aria-hidden="true"]');
    expect(visual).not.toBeNull();
    expect(visual?.textContent).toBe("人事行政管理");
    expect(container.querySelector('[data-slot="rolling-text"]')).not.toHaveAttribute("data-static");
  });

  test("系统减少动态时降级为静态标题且不再渲染翻转字符", () => {
    stubMatchMedia(true);
    const { container } = render(<RollingText text="人事行政管理" />);

    const root = container.querySelector('[data-slot="rolling-text"]');
    expect(root).toHaveAttribute("data-static", "true");
    expect(root).toHaveTextContent("人事行政管理");
    // 静态模式下不应存在任何 aria-hidden 动画层与 sr-only 重复文本
    expect(container.querySelector('[aria-hidden="true"]')).toBeNull();
    expect(container.querySelector(".sr-only")).toBeNull();
  });

  test("空格字符替换为不换行空格避免折行", () => {
    stubMatchMedia(false);
    const { container } = render(<RollingText text="a b" />);
    expect(container.querySelector('[aria-hidden="true"]')?.textContent).toBe("a\u00A0b");
  });
});
