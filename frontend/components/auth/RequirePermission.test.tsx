import { describe, it, expect, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";
import { RequirePermission } from "@/components/auth/RequirePermission";
import { AuthProvider, useAuth, usePermissionDegraded } from "@/lib/auth";
import { hasPermission } from "@/lib/permissions";
import type { User } from "@/lib/types";

// localStorage mock 为模块级单例，跨测试共享，需在每个用例前清空，
// 避免前序 login 写入的缓存用户泄漏到「未登录」场景。
beforeEach(() => {
  localStorage.clear();
});

// ========== 辅助函数 ==========

/** 创建测试用 User 对象 */
function createUser(permissions: string[]): User {
  return {
    id: 1,
    username: "testuser",
    email: "test@example.com",
    full_name: "测试用户",
    active: true,
    role: "admin",
    permissions,
    created_at: "2024-01-01T00:00:00Z",
    updated_at: "2024-01-01T00:00:00Z",
  };
}

/** 注入已登录用户的包裹组件 + 手动触发 login */
function renderWithUser(user: User) {
  const TestConsumer = ({ children }: { children: React.ReactNode }) => {
    const { login } = useAuth();
    // 在 effect 外不会被调用，需要用个标记
    const [done, setDone] = React.useState(false);
    React.useEffect(() => {
      if (!done) {
        login("fake-token", user);
        setDone(true);
      }
      // user 为测试注入的固定引用，仅在首次挂载时登录一次
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

import React from "react";

// ========== 纯函数 hasPermission 测试 ==========

describe("hasPermission 纯函数", () => {
  it("permissions 数组包含对应权限应返回 true", () => {
    expect(hasPermission(["employee.view", "employee.create"], "employee", "view")).toBe(true);
  });

  it("permissions 数组不包含对应权限应返回 false", () => {
    expect(hasPermission(["employee.view"], "employee", "delete")).toBe(false);
  });

  it("空数组应返回 false", () => {
    expect(hasPermission([], "employee", "view")).toBe(false);
  });

  it("null/undefined 应返回 false", () => {
    expect(hasPermission(null, "employee", "view")).toBe(false);
    expect(hasPermission(undefined, "employee", "view")).toBe(false);
  });

  it("跨资源权限不互相干扰", () => {
    expect(hasPermission(["employee.view"], "insurance", "view")).toBe(false);
  });
});

// ========== RequirePermission 组件测试 ==========

describe("RequirePermission 组件", () => {
  describe("hide 模式（默认）", () => {
    it("有权限时渲染 children", () => {
      const user = createUser(["employee.view", "employee.delete"]);
      const { wrapper } = renderWithUser(user);

      render(
        <RequirePermission resource="employee" action="delete">
          <button>删除按钮</button>
        </RequirePermission>,
        { wrapper },
      );

      expect(screen.getByText("删除按钮")).toBeInTheDocument();
    });

    it("无权限时不渲染 children（返回 null）", () => {
      const user = createUser(["employee.view"]);
      const { wrapper } = renderWithUser(user);

      render(
        <RequirePermission resource="employee" action="delete">
          <button>删除按钮</button>
        </RequirePermission>,
        { wrapper },
      );

      expect(screen.queryByText("删除按钮")).not.toBeInTheDocument();
    });

    it("无权限时渲染 fallback", () => {
      const user = createUser(["employee.view"]);
      const { wrapper } = renderWithUser(user);

      render(
        <RequirePermission
          resource="employee"
          action="delete"
          fallback={<span>无权限提示</span>}
        >
          <button>删除按钮</button>
        </RequirePermission>,
        { wrapper },
      );

      expect(screen.queryByText("删除按钮")).not.toBeInTheDocument();
      expect(screen.getByText("无权限提示")).toBeInTheDocument();
    });
  });

  describe("disable 模式", () => {
    it("有权限时正常渲染 children", () => {
      const user = createUser(["employee.view", "employee.delete"]);
      const { wrapper } = renderWithUser(user);

      render(
        <RequirePermission resource="employee" action="delete" mode="disable">
          <button>删除按钮</button>
        </RequirePermission>,
        { wrapper },
      );

      expect(screen.getByText("删除按钮")).toBeInTheDocument();
    });

    it("无权限时渲染但按钮被 disabled", () => {
      const user = createUser(["employee.view"]);
      const { wrapper } = renderWithUser(user);

      render(
        <RequirePermission resource="employee" action="delete" mode="disable">
          <button>删除按钮</button>
        </RequirePermission>,
        { wrapper },
      );

      // 按钮仍渲染
      const btn = screen.getByText("删除按钮");
      expect(btn).toBeInTheDocument();
      // 按钮被禁用
      expect(btn).toBeDisabled();
    });

    it("无权限时显示 Tooltip 提示", async () => {
      const user = createUser(["employee.view"]);
      const { wrapper } = renderWithUser(user);

      render(
        <RequirePermission resource="employee" action="delete" mode="disable">
          <button>删除按钮</button>
        </RequirePermission>,
        { wrapper },
      );

      // Tooltip 的内容应该存在于 DOM 中
      // 注意：Radix Tooltip 渲染方式，内容在 opened 后才可见
      // 但 TooltipContent 的内容 "您没有此权限" 应该在 DOM 内
      const tooltipContent = document.querySelector('[data-state]');
      expect(tooltipContent).toBeTruthy();
    });
  });
});

// ========== fallback 三场景测试 ==========

describe("fallback 三场景策略", () => {
  describe("场景 1：未登录（user === null）", () => {
    it("未登录时 hasPermission 返回 false", () => {
      // 不传用户 = 未登录；但 AuthProvider 内需要消费 useAuth
      // hasPermission 在未登录时由 useAuth 内部逻辑处理
      // 这里直接测试 useAuth 的 hasPermission 行为
      let capturedHasPermission: ((r: string, a: string) => boolean) | null = null;

      function TestComponent() {
        const auth = useAuth();
        capturedHasPermission = auth.hasPermission;
        return null;
      }

      render(
        <AuthProvider>
          <TestComponent />
        </AuthProvider>,
      );

      // 未登录时 user 为 null → hasPermission 返回 false
      expect(capturedHasPermission).not.toBeNull();
      expect(capturedHasPermission!("employee", "view")).toBe(false);
    });
  });

  describe("场景 3：网络错误降级模式", () => {
    it("usePermissionDegraded 初始为 false", () => {
      let degraded: boolean | null = null;

      function TestComponent() {
        degraded = usePermissionDegraded();
        return null;
      }

      render(
        <AuthProvider>
          <TestComponent />
        </AuthProvider>,
      );

      expect(degraded).toBe(false);
    });
  });
});

// ========== useAuth().hasPermission 集成测试 ==========

describe("useAuth().hasPermission 集成", () => {
  it("已登录用户有权限返回 true", () => {
    const user = createUser(["employee.view", "employee.create"]);
    const { wrapper } = renderWithUser(user);

    let capturedHasPermission: ((r: string, a: string) => boolean) | null = null;

    function TestComponent() {
      const auth = useAuth();
      capturedHasPermission = auth.hasPermission;
      return <div>test</div>;
    }

    render(<TestComponent />, { wrapper });

    expect(capturedHasPermission!).not.toBeNull();
    expect(capturedHasPermission!("employee", "view")).toBe(true);
    expect(capturedHasPermission!("employee", "create")).toBe(true);
  });

  it("已登录用户无权限返回 false", () => {
    const user = createUser(["employee.view"]);
    const { wrapper } = renderWithUser(user);

    let capturedHasPermission: ((r: string, a: string) => boolean) | null = null;

    function TestComponent() {
      const auth = useAuth();
      capturedHasPermission = auth.hasPermission;
      return <div>test</div>;
    }

    render(<TestComponent />, { wrapper });

    expect(capturedHasPermission!("employee", "delete")).toBe(false);
  });

  it("permissions 为空数组时返回 false", () => {
    const user = createUser([]);
    const { wrapper } = renderWithUser(user);

    let capturedHasPermission: ((r: string, a: string) => boolean) | null = null;

    function TestComponent() {
      const auth = useAuth();
      capturedHasPermission = auth.hasPermission;
      return <div>test</div>;
    }

    render(<TestComponent />, { wrapper });

    expect(capturedHasPermission!("employee", "view")).toBe(false);
  });
});
