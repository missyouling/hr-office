import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { PersonalSettings } from "@/components/personal-settings";
import { changePassword, updateUserProfile } from "@/lib/api";

const mocks = vi.hoisted(() => ({
  logout: vi.fn(),
  refreshUser: vi.fn(),
}));

vi.mock("@/lib/auth", () => ({
  useAuth: () => ({
    user: { full_name: "张三", username: "zhangsan", email: "zhangsan@example.com" },
    refreshUser: mocks.refreshUser,
    logout: mocks.logout,
  }),
}));

// 仅替换 changePassword 为 mock，其余 API（如 updateUserProfile）保留真实实现
vi.mock("@/lib/api", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@/lib/api")>();
  return {
    ...actual,
    changePassword: vi.fn(),
  };
});

const changePasswordMock = vi.mocked(changePassword);

describe("个人资料接口", () => {
  beforeEach(() => {
    localStorage.clear();
  });

  it("仅以固定契约提交姓名", async () => {
    const fetchMock = vi.spyOn(globalThis, "fetch").mockResolvedValueOnce(
      new Response(JSON.stringify({ user: {}, permissions: [] }), { status: 200 }),
    );

    await updateUserProfile("王小明");

    expect(fetchMock).toHaveBeenCalledWith(expect.stringContaining("/auth/profile"), expect.objectContaining({
      method: "PATCH",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ full_name: "王小明" }),
    }));
    fetchMock.mockRestore();
  });
});

describe("PersonalSettings 组件", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    changePasswordMock.mockReset();
    changePasswordMock.mockResolvedValue({ message: "ok" });
  });

  it("密码修改成功后清空表单并调用统一 logout", async () => {
    render(<PersonalSettings />);

    fireEvent.change(screen.getByLabelText("当前密码"), { target: { value: "old-pass" } });
    fireEvent.change(screen.getByLabelText("新密码"), { target: { value: "new-pass-123" } });
    fireEvent.change(screen.getByLabelText("确认新密码"), { target: { value: "new-pass-123" } });
    fireEvent.click(screen.getByRole("button", { name: "修改密码" }));

    await waitFor(() => expect(mocks.logout).toHaveBeenCalledTimes(1));
    expect(changePasswordMock).toHaveBeenCalledWith("old-pass", "new-pass-123");
    // 表单已清空
    expect((screen.getByLabelText("当前密码") as HTMLInputElement).value).toBe("");
    expect((screen.getByLabelText("新密码") as HTMLInputElement).value).toBe("");
    expect((screen.getByLabelText("确认新密码") as HTMLInputElement).value).toBe("");
  });

  it("密码修改失败时不调用 logout", async () => {
    changePasswordMock.mockRejectedValueOnce(new Error("密码错误"));
    render(<PersonalSettings />);

    fireEvent.change(screen.getByLabelText("当前密码"), { target: { value: "wrong-pass" } });
    fireEvent.change(screen.getByLabelText("新密码"), { target: { value: "new-pass-123" } });
    fireEvent.change(screen.getByLabelText("确认新密码"), { target: { value: "new-pass-123" } });
    fireEvent.click(screen.getByRole("button", { name: "修改密码" }));

    await waitFor(() => expect(changePasswordMock).toHaveBeenCalledTimes(1));
    expect(mocks.logout).not.toHaveBeenCalled();
  });

  it("点击返回按钮触发 onBack 回调", () => {
    const onBack = vi.fn();
    render(<PersonalSettings onBack={onBack} />);

    fireEvent.click(screen.getByRole("button", { name: "返回" }));
    expect(onBack).toHaveBeenCalledTimes(1);
  });

  it("未传入 onBack 时不渲染返回按钮", () => {
    render(<PersonalSettings />);
    expect(screen.queryByRole("button", { name: "返回" })).not.toBeInTheDocument();
  });
});