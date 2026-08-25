import { fireEvent, render, screen } from "@testing-library/react";
import { beforeEach, describe, expect, test, vi } from "vitest";
import { ManagementBar } from "@/components/layout/management-bar";

const mocks = vi.hoisted(() => ({ getSiteNotificationCount: vi.fn(), toggleSidebar: vi.fn() }));

vi.mock("@/lib/dorm-notifications", () => ({ getSiteNotificationCount: mocks.getSiteNotificationCount }));
vi.mock("@/components/ui/sidebar", () => ({ useSidebar: () => ({ toggleSidebar: mocks.toggleSidebar }) }));
vi.mock("@/hooks/use-theme-utils", () => ({ useThemeUtils: () => ({ toggle: vi.fn(), getIcon: () => null, getAction: () => "主题切换" }) }));

describe("ManagementBar", () => {
  beforeEach(() => { vi.clearAllMocks(); mocks.getSiteNotificationCount.mockReturnValue(0); });

  test("保留全局搜索与通知事件入口", () => {
    const searchHandler = vi.fn();
    const notificationHandler = vi.fn();
    window.addEventListener("dock:open-search", searchHandler);
    window.addEventListener("dock:request-notification", notificationHandler);
    render(<ManagementBar variant="new" />);
    // AI 助手已从 dock 移除（浮动聊天面板不再有 dock 入口属预期），搜索为新增入口
    expect(screen.queryByRole("button", { name: "AI 助手" })).not.toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "全局搜索" }));
    fireEvent.click(screen.getByRole("button", { name: "通知中心" }));
    expect(searchHandler).toHaveBeenCalledOnce();
    expect(notificationHandler).toHaveBeenCalledOnce();
    window.removeEventListener("dock:open-search", searchHandler);
    window.removeEventListener("dock:request-notification", notificationHandler);
  });

  test("桌面控制坞固定在侧栏右侧且不传递拖拽位置", () => {
    render(<ManagementBar variant="new" />);
    expect(document.querySelector("[data-floating-dock]")).toHaveClass("rounded-2xl");
    expect(document.querySelector("[data-dock-drag-handle]")).not.toBeInTheDocument();
  });

  test("移动端展开状态仅保存在组件内", () => {
    render(<ManagementBar />);
    fireEvent.click(screen.getByRole("button", { name: "展开管理 Dock" }));
    expect(screen.getAllByRole("button", { name: "全局搜索" })).toHaveLength(2);
    expect(window.localStorage.getItem("dock_preferences_v1")).toBeNull();
  });
});
