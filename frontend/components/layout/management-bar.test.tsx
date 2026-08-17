import { act, fireEvent, render, screen } from "@testing-library/react";
import { beforeEach, describe, expect, test, vi } from "vitest";
import { ManagementBar } from "@/components/layout/management-bar";

const mocks = vi.hoisted(() => ({
  getSiteNotificationCount: vi.fn(),
  toggleSidebar: vi.fn(),
  getDockPreferences: vi.fn(),
  updateDockPreferences: vi.fn(),
}));

const DOCK_PREFERENCES_STORAGE_KEY = "dock_preferences_v1";

vi.mock("@/lib/dorm-notifications", () => ({
  getSiteNotificationCount: mocks.getSiteNotificationCount,
}));

vi.mock("@/lib/api", () => ({
  getDockPreferences: mocks.getDockPreferences,
  updateDockPreferences: mocks.updateDockPreferences,
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
    open,
    onOpenChange,
  }: {
    items: Array<{ title: string; onClick?: () => void }>;
    onDesktopPositionChange?: (position: { left: number; top: number }) => void;
    desktopPosition?: { left: number; top: number } | null;
    open?: boolean;
    onOpenChange?: (open: boolean) => void;
  }) => (
    <div>
      <div data-testid="dock-position">{desktopPosition ? JSON.stringify(desktopPosition) : "null"}</div>
      <div data-testid="dock-mobile-open">{String(open)}</div>
      <button
        type="button"
        data-testid="dock-position-change"
        onClick={() => onDesktopPositionChange?.({ left: 300, top: 120 })}
      >
        变更位置
      </button>
      <button type="button" data-testid="dock-mobile-toggle" onClick={() => onOpenChange?.(!open)}>切换移动展开</button>
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
    window.localStorage.clear();
    mocks.getSiteNotificationCount.mockReturnValue(0);
    mocks.getDockPreferences.mockResolvedValue({ desktop_position: { left: 100, top: 100 }, mobile_expanded: false });
    mocks.updateDockPreferences.mockResolvedValue({ desktop_position: { left: 100, top: 100 }, mobile_expanded: false });
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
    // 等待服务端偏好完成，获得初始位置
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
    // 等待服务端偏好完成
    await act(async () => {
      await Promise.resolve();
      await Promise.resolve();
    });

    fireEvent.click(screen.getByTestId("dock-position-change"));
    // debounce 窗口内不立即写后端
    expect(mocks.updateDockPreferences).not.toHaveBeenCalled();

    await act(async () => {
      vi.advanceTimersByTime(400);
    });
    expect(mocks.updateDockPreferences).toHaveBeenCalledTimes(1);
    expect(mocks.updateDockPreferences).toHaveBeenCalledWith({ desktop_position: { left: 300, top: 120 }, mobile_expanded: false });
    expect(JSON.parse(window.localStorage.getItem(DOCK_PREFERENCES_STORAGE_KEY) ?? "")).toEqual({ desktop_position: { left: 300, top: 120 }, mobile_expanded: false });

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
    expect(mocks.updateDockPreferences).not.toHaveBeenCalled();

    await act(async () => {
      vi.advanceTimersByTime(400);
    });
    expect(mocks.updateDockPreferences).toHaveBeenCalledTimes(1);

    vi.useRealTimers();
    unmount();
  });

  test("服务端位置加载后钳制到当前视口安全范围", async () => {
    mocks.getDockPreferences.mockResolvedValue({ desktop_position: { left: -20, top: 99999 }, mobile_expanded: true });
    render(<ManagementBar />);
    await act(async () => { await Promise.resolve(); });
    expect(screen.getByTestId("dock-position").textContent).toBe(JSON.stringify({ left: 8, top: window.innerHeight - 56 }));
    expect(screen.getByTestId("dock-mobile-open")).toHaveTextContent("true");
  });

  test("服务端请求失败时回退默认位置且不阻断 Dock", async () => {
    mocks.getDockPreferences.mockRejectedValue(new Error("网络异常"));
    render(<ManagementBar />);
    await act(async () => { await Promise.resolve(); await Promise.resolve(); });
    expect(screen.getByTestId("dock-position").textContent).toBe(JSON.stringify({ left: 208, top: window.innerHeight - 80 }));
  });

  test("服务端请求失败时使用本地缓存并钳制位置", async () => {
    window.localStorage.setItem(DOCK_PREFERENCES_STORAGE_KEY, JSON.stringify({ desktop_position: { left: -10, top: 99999 }, mobile_expanded: true }));
    mocks.getDockPreferences.mockRejectedValue(new Error("网络异常"));
    render(<ManagementBar />);
    await act(async () => { await Promise.resolve(); await Promise.resolve(); });
    expect(screen.getByTestId("dock-position").textContent).toBe(JSON.stringify({ left: 8, top: window.innerHeight - 56 }));
    expect(screen.getByTestId("dock-mobile-open")).toHaveTextContent("true");
  });

  test("非法本地缓存回退默认位置和未展开状态", async () => {
    window.localStorage.setItem(DOCK_PREFERENCES_STORAGE_KEY, JSON.stringify({ desktop_position: { left: "错误", top: 20 }, mobile_expanded: "true" }));
    mocks.getDockPreferences.mockRejectedValue(new Error("网络异常"));
    render(<ManagementBar />);
    await act(async () => { await Promise.resolve(); await Promise.resolve(); });
    expect(screen.getByTestId("dock-position").textContent).toBe(JSON.stringify({ left: 208, top: window.innerHeight - 80 }));
    expect(screen.getByTestId("dock-mobile-open")).toHaveTextContent("false");
  });

  test("非法服务端位置安全回退默认位置", async () => {
    mocks.getDockPreferences.mockResolvedValue({ desktop_position: { left: "错误", top: 20 }, mobile_expanded: "true" });
    render(<ManagementBar />);
    await act(async () => { await Promise.resolve(); });
    expect(screen.getByTestId("dock-position").textContent).toBe(JSON.stringify({ left: 208, top: window.innerHeight - 80 }));
    expect(screen.getByTestId("dock-mobile-open")).toHaveTextContent("false");
  });

  test("移动展开切换保存当前位置和展开状态", async () => {
    render(<ManagementBar />);
    await act(async () => { await Promise.resolve(); });
    fireEvent.click(screen.getByTestId("dock-mobile-toggle"));
    expect(mocks.updateDockPreferences).toHaveBeenCalledWith({ desktop_position: { left: 100, top: 100 }, mobile_expanded: true });
    expect(JSON.parse(window.localStorage.getItem(DOCK_PREFERENCES_STORAGE_KEY) ?? "")).toEqual({ desktop_position: { left: 100, top: 100 }, mobile_expanded: true });
  });

  test("服务端成功值优先于旧缓存并覆盖本地缓存", async () => {
    window.localStorage.setItem(DOCK_PREFERENCES_STORAGE_KEY, JSON.stringify({ desktop_position: { left: 20, top: 20 }, mobile_expanded: true }));
    mocks.getDockPreferences.mockResolvedValue({ desktop_position: { left: 360, top: 180 }, mobile_expanded: false });
    render(<ManagementBar />);
    await act(async () => { await Promise.resolve(); });
    expect(screen.getByTestId("dock-position").textContent).toBe('{"left":360,"top":180}');
    expect(screen.getByTestId("dock-mobile-open")).toHaveTextContent("false");
    expect(JSON.parse(window.localStorage.getItem(DOCK_PREFERENCES_STORAGE_KEY) ?? "")).toEqual({ desktop_position: { left: 360, top: 180 }, mobile_expanded: false });
  });
});
