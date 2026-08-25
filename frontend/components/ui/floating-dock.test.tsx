import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, test, vi } from "vitest";
import { FloatingDock } from "@/components/ui/floating-dock";

const items = [{ title: "测试项", icon: <span>i</span>, onClick: vi.fn() }];

describe("FloatingDock", () => {
  test("桌面端提供固定快捷操作，不渲染拖动手柄", () => {
    render(<FloatingDock items={items} variant="new" />);
    expect(document.querySelector("[data-floating-dock]")).toHaveClass("rounded-2xl");
    expect(screen.getByRole("button", { name: "测试项" })).toBeInTheDocument();
    expect(screen.queryByLabelText("拖动 Dock")).not.toBeInTheDocument();
  });

  test("坞图标为常驻圆形底色（bg-muted），悬停加深一档而非悬停才出现底色", () => {
    render(<FloatingDock items={items} />);
    // 弹簧模式下按钮内首个 div 即图标容器
    const iconBox = screen.getByRole("button", { name: "测试项" }).querySelector("div")!;
    expect(iconBox).toHaveClass("rounded-full", "bg-muted");
    expect(iconBox).not.toHaveClass("rounded-xl");
    // 悬停仅切换更深一档（--border），不再依赖悬停才出现底
    expect(iconBox).toHaveClass("hover:bg-border");
    expect(iconBox).not.toHaveClass("hover:bg-muted");
  });

  test("移动端由展开按钮控制快捷操作", () => {
    render(<FloatingDock items={items} />);
    fireEvent.click(screen.getByRole("button", { name: "展开管理 Dock" }));
    expect(screen.getAllByRole("button", { name: "测试项" })).toHaveLength(2);
  });

  test("受控移动端展开状态变化通知调用方", () => {
    const onOpenChange = vi.fn();
    render(<FloatingDock items={items} open onOpenChange={onOpenChange} />);
    fireEvent.click(screen.getByRole("button", { name: "展开管理 Dock" }));
    expect(onOpenChange).toHaveBeenCalledWith(false);
  });
});
