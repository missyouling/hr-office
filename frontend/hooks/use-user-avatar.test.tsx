import { describe, it, expect, vi, beforeEach } from "vitest";
import React from "react";
import { render, screen, waitFor, fireEvent } from "@testing-library/react";
import { AuthProvider, useAuth } from "@/lib/auth";
import { useUserAvatar } from "@/hooks/use-user-avatar";
import { fetchUserAvatar } from "@/lib/avatar-api";
import type { User } from "@/lib/types";

// mock 头像 API，避免真实网络请求
vi.mock("@/lib/avatar-api", () => ({
  fetchUserAvatar: vi.fn(),
}));

const fetchUserAvatarMock = vi.mocked(fetchUserAvatar);

// jsdom 未实现 URL.createObjectURL / revokeObjectURL，这里补齐
let createObjectURLMock: ReturnType<typeof vi.fn>;
let revokeObjectURLMock: ReturnType<typeof vi.fn>;

beforeEach(() => {
  localStorage.clear();
  vi.clearAllMocks();
  createObjectURLMock = vi.fn(() => "blob:mock-url");
  revokeObjectURLMock = vi.fn();
  Object.defineProperty(URL, "createObjectURL", { value: createObjectURLMock, configurable: true });
  Object.defineProperty(URL, "revokeObjectURL", { value: revokeObjectURLMock, configurable: true });
});

function createUser(): User {
  return {
    id: 1,
    username: "testuser",
    email: "test@example.com",
    full_name: "测试用户",
    active: true,
    permissions: [],
    created_at: "2024-01-01T00:00:00Z",
    updated_at: "2024-01-01T00:00:00Z",
  };
}

/** 注入已登录用户的包裹组件（参考 RequirePermission.test.tsx 模式） */
function renderWithUser() {
  const TestConsumer = ({ children }: { children: React.ReactNode }) => {
    const { login } = useAuth();
    const [done, setDone] = React.useState(false);
    React.useEffect(() => {
      if (!done) {
        login("fake-token", createUser());
        setDone(true);
      }
    }, [done, login]);
    return done ? <>{children}</> : null;
  };

  return {
    wrapper: ({ children }: { children: React.ReactNode }) => (
      <AuthProvider>
        <TestConsumer>{children}</TestConsumer>
      </AuthProvider>
    ),
  };
}

function Probe() {
  const avatar = useUserAvatar();
  return (
    <div>
      <span data-testid="status">{avatar.status}</span>
      <span data-testid="src">{avatar.src ?? "null"}</span>
      <button type="button" onClick={avatar.refresh}>
        refresh
      </button>
    </div>
  );
}

describe("useUserAvatar", () => {
  it("登录后加载成功：生成 object URL 并进入 ready", async () => {
    fetchUserAvatarMock.mockResolvedValue(new Blob(["svg"], { type: "image/svg+xml" }));
    const { wrapper } = renderWithUser();

    render(<Probe />, { wrapper });

    await waitFor(() => expect(screen.getByTestId("status").textContent).toBe("ready"));
    expect(screen.getByTestId("src").textContent).toBe("blob:mock-url");
    expect(fetchUserAvatarMock).toHaveBeenCalledTimes(1);
  });

  it("加载失败：进入 error 且不生成 URL", async () => {
    fetchUserAvatarMock.mockRejectedValue(new Error("network error"));
    const { wrapper } = renderWithUser();

    render(<Probe />, { wrapper });

    await waitFor(() => expect(screen.getByTestId("status").textContent).toBe("error"));
    expect(screen.getByTestId("src").textContent).toBe("null");
    expect(createObjectURLMock).not.toHaveBeenCalled();
  });

  it("组件卸载时释放 object URL", async () => {
    fetchUserAvatarMock.mockResolvedValue(new Blob(["svg"], { type: "image/svg+xml" }));
    const { wrapper } = renderWithUser();

    const { unmount } = render(<Probe />, { wrapper });
    await waitFor(() => expect(screen.getByTestId("status").textContent).toBe("ready"));

    unmount();
    expect(revokeObjectURLMock).toHaveBeenCalledWith("blob:mock-url");
  });

  it("refresh 后重新请求并释放旧 URL", async () => {
    let urlCounter = 0;
    createObjectURLMock.mockImplementation(() => `blob:mock-${++urlCounter}`);
    fetchUserAvatarMock
      .mockResolvedValueOnce(new Blob(["svg-1"], { type: "image/svg+xml" }))
      .mockResolvedValueOnce(new Blob(["svg-2"], { type: "image/svg+xml" }));
    const { wrapper } = renderWithUser();

    render(<Probe />, { wrapper });
    await waitFor(() => expect(screen.getByTestId("src").textContent).toBe("blob:mock-1"));

    fireEvent.click(screen.getByText("refresh"));

    await waitFor(() => expect(screen.getByTestId("src").textContent).toBe("blob:mock-2"));
    expect(fetchUserAvatarMock).toHaveBeenCalledTimes(2);
    // 旧 URL 被释放，新 URL 未被释放
    expect(revokeObjectURLMock).toHaveBeenCalledWith("blob:mock-1");
    expect(revokeObjectURLMock).not.toHaveBeenCalledWith("blob:mock-2");
  });

  it("未登录时不发起请求并保持 idle", async () => {
    render(
      <AuthProvider>
        <Probe />
      </AuthProvider>,
    );

    await waitFor(() => expect(screen.getByTestId("status").textContent).toBe("idle"));
    expect(fetchUserAvatarMock).not.toHaveBeenCalled();
  });
});