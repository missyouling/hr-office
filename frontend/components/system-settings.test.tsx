import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
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

  it("不渲染未接入的高级字段预留占位内容", () => {
    render(<SystemSettings />);

    expect(screen.queryByText("字段分组配置 (待开发)")).not.toBeInTheDocument();
    expect(screen.queryByText("条件显示规则 (待开发)")).not.toBeInTheDocument();
    expect(screen.queryByText("该功能正在内测中，敬请期待")).not.toBeInTheDocument();
    expect(screen.queryByText("需要配合高级字段引擎使用")).not.toBeInTheDocument();
  });

  it("无系统设置权限时保留权限门控（触发跳转）", async () => {
    mocks.hasPermission.mockReturnValue(false);
    render(<SystemSettings />);

    // 权限不足时 useEffect 触发 toast + router.push("/")
    await waitFor(() => expect(mocks.hasPermission).toHaveBeenCalledWith("settings", "view"));
  });

  it("仅 users.view 用户可见审计日志和系统监控内部入口", () => {
    mocks.hasPermission.mockImplementation((resource: string) => resource === "settings" || resource === "users");
    render(<SystemSettings />);
    expect(screen.getByRole("button", { name: "审计日志" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "系统监控" })).toBeInTheDocument();

    cleanup();
    mocks.hasPermission.mockImplementation((resource: string) => resource === "settings");
    render(<SystemSettings />);
    expect(screen.queryByRole("button", { name: "审计日志" })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "系统监控" })).not.toBeInTheDocument();
  });

  it("进入系统监控后可返回系统设置", () => {
    mocks.hasPermission.mockReturnValue(true);
    render(<SystemSettings />);
    fireEvent.click(screen.getByRole("button", { name: "系统监控" }));
    expect(screen.getByRole("heading", { name: "系统监控", level: 1 })).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "返回系统设置" }));
    expect(screen.getByRole("heading", { name: "系统设置" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "审计日志" })).toBeInTheDocument();
  });
});
