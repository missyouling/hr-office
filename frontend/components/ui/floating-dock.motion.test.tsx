import { fireEvent, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, test, vi } from "vitest";
import { FloatingDock } from "@/components/ui/floating-dock";

const items = [
  { title: "主页", icon: <span>i</span>, onClick: vi.fn() },
  { title: "通知中心", icon: <span>n</span>, badge: 3 },
];

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

describe("FloatingDock 弹性缩放与减少动态降级", () => {
  test("桌面坞默认启用鼠标距离弹簧缩放并渲染悬停提示", () => {
    stubMatchMedia(false);
    render(<FloatingDock items={items} />);

    const dock = document.querySelector('[data-floating-dock]');
    expect(dock).toHaveAttribute("data-motion", "spring");

    // 悬停前无提示文本（按钮可访问名来自 aria-label）
    expect(screen.queryByText("主页")).toBeNull();
    // 悬停单个图标后浮现提示（React 合成 onMouseEnter 由 mouseover 驱动）
    const trigger = screen.getByRole("button", { name: "主页" });
    const iconBox = trigger.querySelector("div")!;
    fireEvent.mouseEnter(iconBox);
    fireEvent.mouseOver(iconBox);
    expect(screen.getByText("主页")).toHaveClass("pointer-events-none");
    fireEvent.mouseLeave(iconBox);
  });

  test("系统减少动态时桌面坞退化为静态尺寸且不启用弹簧", () => {
    stubMatchMedia(true);
    render(<FloatingDock items={items} />);

    expect(document.querySelector('[data-floating-dock]')).toHaveAttribute("data-motion", "off");
    // 静态动作按钮保持固定尺寸，且与弹簧模式共用圆形常驻底契约
    expect(screen.getByRole("button", { name: "主页" })).toHaveClass("h-9", "w-9", "rounded-full", "bg-muted", "hover:bg-border");
  });

  test("通知角标在弹簧与静态模式下均正常渲染并可点击触发回调", () => {
    stubMatchMedia(false);
    render(<FloatingDock items={items} />);
    expect(screen.getByText("3")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "主页" }));
    expect(items[0].onClick).toHaveBeenCalledOnce();

    stubMatchMedia(true);
    render(<FloatingDock items={items} />);
    expect(screen.getAllByText("3").length).toBeGreaterThan(0);
  });
});
