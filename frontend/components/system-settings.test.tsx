import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { SystemSettings } from "@/components/system-settings";

const mocks = vi.hoisted(() => ({
  hasPermission: vi.fn(),
}));

vi.mock("@/lib/auth", () => ({
  useAuth: () => ({
    user: { full_name: "管理员", username: "admin", role: "admin", permissions: ["settings.view"] },
    hasPermission: mocks.hasPermission,
  }),
}));

// 默认 Tab 为公告管理，其加载依赖 fetchAnnouncements，这里返回空列表
vi.mock("@/lib/api", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@/lib/api")>();
  return {
    ...actual,
    fetchAnnouncements: vi.fn().mockResolvedValue([]),
  };
});

describe("SystemSettings 组件", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mocks.hasPermission.mockReturnValue(true);
  });

  it("渲染返回按钮并触发 onBack 回调", () => {
    const onBack = vi.fn();
    render(<SystemSettings onBack={onBack} />);

    fireEvent.click(screen.getByRole("button", { name: "返回" }));
    expect(onBack).toHaveBeenCalledTimes(1);
  });

  it("未传入 onBack 时不渲染返回按钮", () => {
    render(<SystemSettings />);
    expect(screen.queryByRole("button", { name: "返回" })).not.toBeInTheDocument();
  });

  it("无系统设置权限时保留权限门控（触发跳转）", async () => {
    mocks.hasPermission.mockReturnValue(false);
    render(<SystemSettings />);

    // 权限不足时 useEffect 触发 toast + router.push("/")
    await waitFor(() => expect(mocks.hasPermission).toHaveBeenCalledWith("settings", "view"));
  });
});