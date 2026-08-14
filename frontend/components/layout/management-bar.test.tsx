import { act, fireEvent, render, screen } from "@testing-library/react";
import { beforeEach, describe, expect, test, vi } from "vitest";
import { ManagementBar } from "@/components/layout/management-bar";

const mocks = vi.hoisted(() => ({
  getSiteNotificationCount: vi.fn(),
  toggleSidebar: vi.fn(),
  fetchUserPreferences: vi.fn(),
  updateUserPreferences: vi.fn(),
}));

vi.mock("@/lib/dorm-notifications", () => ({
  getSiteNotificationCount: mocks.getSiteNotificationCount,
}));

vi.mock("@/lib/api", () => ({
  fetchUserPreferences: mocks.fetchUserPreferences,
  updateUserPreferences: mocks.updateUserPreferences,
}));

vi.mock("@/components/ui/sidebar", () => ({
  useSidebar: () => ({ toggleSidebar: mocks.toggleSidebar, state: "expanded", isMobile: false }),
}));

vi.mock("@/hooks/use-theme-utils", () => ({
  useThemeUtils: () => ({
    toggle: vi.fn(),
    getIcon: () => null,
    getAction: () => "主题切换",
  }),
}));

// 替换 FloatingDock 为普通按钮列表，直接触发 item.onClick / onDesktopPositionChange，
// 避免 motion 依赖；同时暴露 desktopPosition 供断言视觉状态实时更新
vi.mock("@/components/ui/floating-dock", () => ({
  FloatingDock: ({
    items,
    onDesktopPositionChange,
    desktopPosition,
  }: {
    items: Array<{ title: string; onClick?: () => void }>;
    onDesktopPositionChange?: (position: { left: number; top: number }) => void;
    desktopPosition?: { left: number; top: number } | null;
  }) => (
    <div>
      <div data-testid="dock-position">{desktopPosition ? JSON.stringify(desktopPosition) : "null"}</div>
      <button
        type="button"
        data-testid="dock-position-change"
        onClick={() => onDesktopPositionChange?.({ left: 300, top: 120 })}
      >
        变更位置
      </button>
      {items.map((item) => (
        <button key={item.title} type="button" onClick={item.onClick} aria-label={item.title}>
          {item.title}
        </button>
      ))}
    </div>
  ),
}));

describe("ManagementBar", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mocks.getSiteNotificationCount.mockReturnValue(0);
    mocks.fetchUserPreferences.mockResolvedValue({ dock_position: { left: 100, top: 100 } });
    mocks.updateUserPreferences.mockResolvedValue({ dock_position: { left: 100, top: 100 } });
  });

  test("渲染 AI 助手入口", () => {
    render(<ManagementBar />);
    expect(screen.getByRole("button", { name: "AI 助手" })).toBeInTheDocument();
  });

  test("点击 AI 助手派发 dock:open-chat 事件", () => {
    const handler = vi.fn();
    window.addEventListener("dock:open-chat", handler);
    render(<ManagementBar />);
    fireEvent.click(screen.getByRole("button", { name: "AI 助手" }));
    expect(handler).toHaveBeenCalledTimes(1);
    window.removeEventListener("dock:open-chat", handler);
  });

  test("拖动 Dock 变更位置后视觉状态实时更新", async () => {
    render(<ManagementBar />);
    // 等待 fetchUserPreferences 完成，获得初始位置
    await act(async () => {
      await Promise.resolve();
    });
    expect(screen.getByTestId("dock-position").textContent).toBe('{"left":100,"top":100}');

    fireEvent.click(screen.getByTestId("dock-position-change"));
    // clamp 后应为 (300, 120)，无需等待持久化 debounce 即已更新
    expect(screen.getByTestId("dock-position").textContent).toBe('{"left":300,"top":120}');
  });

  test("拖动 Dock 结束后经 debounce 持久化到偏好 API", async () => {
    vi.useFakeTimers();
    const { unmount } = render(<ManagementBar />);
    // 等待 fetchUserPreferences 完成
    await act(async () => {
      await Promise.resolve();
      await Promise.resolve();
    });

    fireEvent.click(screen.getByTestId("dock-position-change"));
    // debounce 窗口内不立即写后端
    expect(mocks.updateUserPreferences).not.toHaveBeenCalled();

    await act(async () => {
      vi.advanceTimersByTime(400);
    });
    expect(mocks.updateUserPreferences).toHaveBeenCalledTimes(1);
    expect(mocks.updateUserPreferences).toHaveBeenCalledWith(
      expect.objectContaining({ dock_position: expect.any(Object) }),
    );

    vi.useRealTimers();
    unmount();
  });

  test("连续变更位置只持久化一次，避免拖动过程大量请求", async () => {
    vi.useFakeTimers();
    const { unmount } = render(<ManagementBar />);
    await act(async () => {
      await Promise.resolve();
      await Promise.resolve();
    });

    fireEvent.click(screen.getByTestId("dock-position-change"));
    fireEvent.click(screen.getByTestId("dock-position-change"));
    fireEvent.click(screen.getByTestId("dock-position-change"));
    expect(mocks.updateUserPreferences).not.toHaveBeenCalled();

    await act(async () => {
      vi.advanceTimersByTime(400);
    });
    expect(mocks.updateUserPreferences).toHaveBeenCalledTimes(1);

    vi.useRealTimers();
    unmount();
  });
});