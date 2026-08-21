import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, test, vi } from "vitest";

const { getIsMobile, setIsMobile } = vi.hoisted(() => {
  let isMobile = false;
  return {
    getIsMobile: () => isMobile,
    setIsMobile: (value: boolean) => {
      isMobile = value;
    },
  };
});

vi.mock("@/hooks/use-mobile", () => ({
  useIsMobile: getIsMobile,
}));

import {
  Sidebar,
  SidebarContent,
  SidebarInset,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
  SidebarProvider,
  useSidebar,
} from "@/components/ui/sidebar";

function TestSidebar({ mobileWidth, mobileMaxWidth }: { mobileWidth?: string; mobileMaxWidth?: string }) {
  const { toggleSidebar } = useSidebar();

  return (
    <>
      <button onClick={toggleSidebar}>切换</button>
      <Sidebar collapsible="icon" mobileWidth={mobileWidth} mobileMaxWidth={mobileMaxWidth}>
        <SidebarContent>
          <SidebarMenu>
            <SidebarMenuItem>
              <SidebarMenuButton tooltip="工作台">工作台</SidebarMenuButton>
            </SidebarMenuItem>
          </SidebarMenu>
        </SidebarContent>
      </Sidebar>
    </>
  );
}

describe("SidebarProvider", () => {
  test("桌面外壳锁定视口高度并使用紧凑侧栏宽度", () => {
    const { container } = render(
      <SidebarProvider>
        <TestSidebar />
      </SidebarProvider>,
    );

    expect(container.querySelector('[data-slot="sidebar-wrapper"]')).toHaveClass("h-[100dvh]", "overflow-hidden");
    expect(container.querySelector('[data-slot="sidebar-gap"]')).toHaveClass("w-[var(--sidebar-width)]");
  });

  test("桌面侧栏使用显式 var() 宽度与 offcanvas 位移，保持几何一致", () => {
    const { container } = render(
      <SidebarProvider>
        <TestSidebar />
      </SidebarProvider>,
    );

    expect(container.querySelector('[data-slot="sidebar-gap"]')).toHaveClass("w-[var(--sidebar-width)]");
    expect(container.querySelector('[data-slot="sidebar-container"]')).toHaveClass(
      "fixed",
      "z-10",
      "w-[var(--sidebar-width)]",
      "left-0",
      "group-data-[collapsible=offcanvas]:left-[calc(var(--sidebar-width)*-1)]",
      "group-data-[collapsible=icon]:w-[var(--sidebar-width-icon)]",
    );
  });

  test("桌面侧栏保留自身宽度，内容区可收缩到剩余空间", () => {
    const { container } = render(
      <SidebarProvider>
        <TestSidebar />
        <SidebarInset>内容卡片</SidebarInset>
      </SidebarProvider>,
    );

    expect(container.querySelector('[data-slot="sidebar"]')).toHaveClass("shrink-0");
    expect(container.querySelector('[data-slot="sidebar-inset"]')).toHaveClass("min-w-0", "flex-1", "overflow-hidden");
  });

  test("折叠时保留图标栏并显示导航提示", async () => {
    const { container } = render(
      <SidebarProvider>
        <TestSidebar />
      </SidebarProvider>,
    );

    fireEvent.click(screen.getByRole("button", { name: "切换" }));
    expect(container.querySelector('[data-slot="sidebar"]')).toHaveAttribute("data-state", "collapsed");
    expect(screen.getByRole("button", { name: "工作台" })).toBeInTheDocument();
  });

  test("移动端使用带遮罩的抽屉而非桌面侧栏", () => {
    setIsMobile(true);
    render(
      <SidebarProvider>
        <TestSidebar />
      </SidebarProvider>,
    );

    fireEvent.click(screen.getByRole("button", { name: "切换" }));
    expect(screen.getByRole("dialog")).toHaveAttribute("data-mobile", "true");
    setIsMobile(false);
  });

  test("未传移动尺寸时保持旧壳默认宽度", () => {
    setIsMobile(true);
    render(<SidebarProvider><TestSidebar /></SidebarProvider>);

    fireEvent.click(screen.getByRole("button", { name: "切换" }));
    const dialog = screen.getByRole("dialog");
    expect(dialog).toHaveStyle({ "--sidebar-mobile-width": "18rem", "--sidebar-mobile-max-width": "none" });
    setIsMobile(false);
  });

  test("可选移动尺寸覆盖默认宽度且支持最大宽度", () => {
    setIsMobile(true);
    render(<SidebarProvider><TestSidebar mobileWidth="85vw" mobileMaxWidth="24rem" /></SidebarProvider>);

    fireEvent.click(screen.getByRole("button", { name: "切换" }));
    const dialog = screen.getByRole("dialog");
    expect(dialog).toHaveStyle({ "--sidebar-mobile-width": "85vw", "--sidebar-mobile-max-width": "24rem" });
    setIsMobile(false);
  });
});
