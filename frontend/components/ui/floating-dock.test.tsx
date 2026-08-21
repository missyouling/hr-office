import { fireEvent, render, screen } from "@testing-library/react";
import type { ReactNode } from "react";
import { beforeEach, describe, expect, test, vi } from "vitest";

import { FloatingDock } from "@/components/ui/floating-dock";

// 用轻量 mock 替换 motion，避免 jsdom 中动画 / ResizeObserver 等复杂依赖
vi.mock("motion/react", () => ({
  motion: {
    div: ({ children, ...rest }: Record<string, unknown>) => {
      // 剥离动画专属 props，避免 React 将其渲染为未知 DOM 属性
      delete rest.initial;
      delete rest.animate;
      delete rest.exit;
      delete rest.transition;
      delete rest.layoutId;
      return <div {...rest}>{children as ReactNode}</div>;
    },
  },
  AnimatePresence: ({ children }: { children?: ReactNode }) => <>{children}</>,
  useMotionValue: () => ({ set: () => {} }),
  useSpring: (value: unknown) => value,
  useTransform: () => 0,
}));

const items = [{ title: "测试项", icon: <span>i</span> }];

describe("FloatingDock 拖动手柄", () => {
  beforeEach(() => {
    // jsdom 缺少 pointer capture 相关 API，测试中桩掉
    Element.prototype.setPointerCapture = vi.fn();
    Element.prototype.releasePointerCapture = vi.fn();
  });

  test("桌面端渲染可访问的拖动手柄", () => {
    render(
      <FloatingDock
        items={items}
        desktopPosition={{ left: 100, top: 100 }}
        onDesktopPositionChange={vi.fn()}
      />,
    );
    const handle = screen.getByLabelText("拖动 Dock");
    expect(handle).toHaveAttribute("data-dock-drag-handle");
    expect(handle).toHaveAttribute("role", "separator");
    // 移动端隐藏：手柄随桌面 Dock 一起通过 CSS 控制显示（hidden md:flex）
    expect(handle.className).toContain("hidden");
    expect(handle.className).toContain("md:flex");
  });

  test("未提供位置或回调时不渲染拖动手柄", () => {
    render(<FloatingDock items={items} />);
    expect(screen.queryByLabelText("拖动 Dock")).not.toBeInTheDocument();
  });

  test("pointerDown / pointerMove / pointerUp 触发 onDesktopPositionChange", () => {
    const onChange = vi.fn();
    render(
      <FloatingDock
        items={items}
        desktopPosition={{ left: 100, top: 100 }}
        onDesktopPositionChange={onChange}
      />,
    );
    const handle = screen.getByLabelText("拖动 Dock");

    fireEvent.pointerDown(handle, { pointerId: 1, clientX: 200, clientY: 150 });
    fireEvent.pointerMove(handle, { pointerId: 1, clientX: 220, clientY: 170 });
    fireEvent.pointerUp(handle, { pointerId: 1 });

    // 位移量 (20, 20) 叠加到初始位置 (100, 100)
    expect(onChange).toHaveBeenCalledTimes(1);
    expect(onChange).toHaveBeenCalledWith(expect.objectContaining({ left: 120, top: 120 }));
  });

  test("在按钮上按下并移动不会启动拖动", () => {
    const onChange = vi.fn();
    render(
      <FloatingDock
        items={items}
        desktopPosition={{ left: 100, top: 100 }}
        onDesktopPositionChange={onChange}
      />,
    );
    const button = screen.getByRole("button", { name: "测试项" });

    fireEvent.pointerDown(button, { pointerId: 1, clientX: 200, clientY: 150 });
    fireEvent.pointerMove(button, { pointerId: 1, clientX: 220, clientY: 170 });
    fireEvent.pointerUp(button, { pointerId: 1 });

    expect(onChange).not.toHaveBeenCalled();
  });

  test("移动端可由外部受控展开并通知状态切换", () => {
    const onOpenChange = vi.fn();
    render(<FloatingDock items={items} open onOpenChange={onOpenChange} />);
    expect(screen.getAllByRole("button", { name: "测试项" })).toHaveLength(2);
    fireEvent.click(screen.getByRole("button", { name: "展开管理 Dock" }));
    expect(onOpenChange).toHaveBeenCalledWith(false);
  });

  test("新壳变体仅增强桌面 Dock 圆角，不影响现有操作入口", () => {
    render(<FloatingDock items={items} variant="new" />);
    expect(document.querySelector("[data-floating-dock]")).toHaveClass("rounded-2xl");
    expect(screen.getByRole("button", { name: "测试项" })).toBeInTheDocument();
  });

  test("新壳桌面 Dock 保持在190px侧栏外，默认变体仍使用原位置", () => {
    const { rerender } = render(<FloatingDock items={items} desktopPosition={{ left: 100, top: 100 }} variant="new" />);
    expect(document.querySelector("[data-floating-dock]")).toHaveStyle({ left: "208px", top: "100px" });

    rerender(<FloatingDock items={items} desktopPosition={{ left: 100, top: 100 }} />);
    expect(document.querySelector("[data-floating-dock]")).toHaveStyle({ left: "100px", top: "100px" });
  });

  test("新壳拖动 Dock 时不允许进入侧栏账户区", () => {
    const onChange = vi.fn();
    render(<FloatingDock items={items} desktopPosition={{ left: 240, top: 100 }} onDesktopPositionChange={onChange} variant="new" />);
    const handle = screen.getByLabelText("拖动 Dock");

    fireEvent.pointerDown(handle, { pointerId: 1, clientX: 240, clientY: 100 });
    fireEvent.pointerMove(handle, { pointerId: 1, clientX: 0, clientY: 100 });

    expect(onChange).toHaveBeenCalledWith(expect.objectContaining({ left: 208, top: 100 }));
  });
});
