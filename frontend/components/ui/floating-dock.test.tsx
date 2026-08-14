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
});
