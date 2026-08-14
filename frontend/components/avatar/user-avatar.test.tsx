import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";
import { UserAvatar } from "@/components/avatar/user-avatar";
import { useUserAvatar } from "@/hooks/use-user-avatar";

// mock hook，聚焦组件渲染状态
vi.mock("@/hooks/use-user-avatar", () => ({
  useUserAvatar: vi.fn(),
}));

const useUserAvatarMock = vi.mocked(useUserAvatar);

beforeEach(() => {
  vi.clearAllMocks();
});

describe("UserAvatar 组件", () => {
  it("ready 状态渲染服务端头像 img", () => {
    useUserAvatarMock.mockReturnValue({
      src: "blob:avatar-url",
      status: "ready",
      refresh: vi.fn(),
    });

    render(<UserAvatar name="张三" className="h-8 w-8 rounded-full" />);

    const img = screen.getByAltText("avatar");
    expect(img).toHaveAttribute("src", "blob:avatar-url");
    expect(img).toHaveClass("h-8", "w-8", "rounded-full");
  });

  it("loading 状态渲染占位元素", () => {
    useUserAvatarMock.mockReturnValue({
      src: null,
      status: "loading",
      refresh: vi.fn(),
    });

    render(<UserAvatar name="张三" />);

    expect(screen.getByLabelText("头像加载中")).toBeInTheDocument();
  });

  it("加载失败回退本地首字母（中文取首字）", () => {
    useUserAvatarMock.mockReturnValue({
      src: null,
      status: "error",
      refresh: vi.fn(),
    });

    render(<UserAvatar name="张三" />);

    expect(screen.getByText("张")).toBeInTheDocument();
  });

  it("未登录（idle）回退本地首字母（拉丁取大写）", () => {
    useUserAvatarMock.mockReturnValue({
      src: null,
      status: "idle",
      refresh: vi.fn(),
    });

    render(<UserAvatar name="alice" />);

    expect(screen.getByText("A")).toBeInTheDocument();
  });
});