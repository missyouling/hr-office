import { describe, it, expect, beforeEach, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import React from "react";
import { AuthProvider, useAuth } from "@/lib/auth";
import type { User } from "@/lib/types";
import InvoiceManagement from "./InvoiceManagement";

// ========== 子组件 Mock（聚焦页签门控逻辑，避免真实渲染副作用） ==========

vi.mock("./tabs/InvoicesTab", () => ({ default: () => <div>InvoicesTab</div> }));
vi.mock("./tabs/PendingApprovalTab", () => ({ default: () => <div>PendingApprovalTab</div> }));
vi.mock("./tabs/StatsTab", () => ({ default: () => <div>StatsTab</div> }));
vi.mock("./tabs/InvoiceArchiveTab", () => ({ default: () => <div>InvoiceArchiveTab</div> }));
vi.mock("./dialogs/InvoiceDialog", () => ({ InvoiceDialog: () => <div>InvoiceDialog</div> }));
vi.mock("./dialogs/InvoiceDetailDialog", () => ({ InvoiceDetailDialog: () => <div>InvoiceDetailDialog</div> }));
vi.mock("./upload/InvoiceUploadWorkbench", () => ({
  InvoiceUploadWorkbench: () => <div>InvoiceUploadWorkbench</div>,
}));

beforeEach(() => {
  localStorage.clear();
});

// ========== 辅助函数 ==========

/**
 * 创建测试用 User。
 * 后端认证接口（AuthUserResponse）不再返回 role 字段，故默认省略 role 以模拟真实 payload；
 * 需要验证 role 兜底逻辑时显式传入 role。
 */
function createUser(permissions: string[], role?: User["role"]): User {
  return {
    id: 1,
    username: "testuser",
    email: "test@example.com",
    full_name: "测试用户",
    active: true,
    ...(role ? { role } : {}),
    permissions,
    created_at: "2024-01-01T00:00:00Z",
    updated_at: "2024-01-01T00:00:00Z",
  };
}

/** 注入已登录用户的包裹组件 + 手动触发 login（与 RequirePermission.test.tsx 同模式） */
function renderWithUser(user: User) {
  const TestConsumer = ({ children }: { children: React.ReactNode }) => {
    const { login } = useAuth();
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

  const wrapper = ({ children }: { children: React.ReactNode }) => (
    <AuthProvider>
      <TestConsumer>{children}</TestConsumer>
    </AuthProvider>
  );

  render(<InvoiceManagement />, { wrapper });
}

/** 管理类页签的角色（tab 按钮）查询 */
const tabByRole = (name: string) => screen.queryByRole("tab", { name });

// ========== 测试用例 ==========

describe("InvoiceManagement 页签门控（后端无 user.role，仅扁平 permissions）", () => {
  it("admin：role 缺失但 permissions 含 invoice.create/approve 及 admin 专属权限时，归档管理等页签可见", () => {
    renderWithUser(
      createUser([
        "invoice.view",
        "invoice.create",
        "invoice.approve",
        "invoice.reject",
        "users.delete",
      ]),
    );

    expect(tabByRole("发票列表")).toBeInTheDocument();
    expect(tabByRole("归档管理")).toBeInTheDocument();
    expect(tabByRole("统计分析")).toBeInTheDocument();
    expect(tabByRole("待审批")).toBeInTheDocument();
    expect(tabByRole("上传解析")).toBeInTheDocument();
  });

  it("manager：role 缺失但有 invoice.approve、无 admin 专属权限时，归档管理可见、待审批不可见", () => {
    renderWithUser(
      createUser(["invoice.view", "invoice.create", "invoice.approve", "users.view"]),
    );

    expect(tabByRole("归档管理")).toBeInTheDocument();
    expect(tabByRole("统计分析")).toBeInTheDocument();
    expect(tabByRole("待审批")).not.toBeInTheDocument();
    expect(tabByRole("上传解析")).toBeInTheDocument();
  });

  it("editor：有 invoice.create、无 invoice.approve 时，归档管理与待审批均不可见", () => {
    renderWithUser(
      createUser(["invoice.view", "invoice.create", "invoice.edit", "invoice.delete", "invoice.submit"]),
    );

    expect(tabByRole("归档管理")).not.toBeInTheDocument();
    expect(tabByRole("统计分析")).not.toBeInTheDocument();
    expect(tabByRole("待审批")).not.toBeInTheDocument();
    expect(tabByRole("上传解析")).toBeInTheDocument();
  });

  it("viewer：仅 invoice.view 时，管理类与上传页签均不可见", () => {
    renderWithUser(createUser(["invoice.view"]));

    expect(tabByRole("归档管理")).not.toBeInTheDocument();
    expect(tabByRole("统计分析")).not.toBeInTheDocument();
    expect(tabByRole("待审批")).not.toBeInTheDocument();
    expect(tabByRole("上传解析")).not.toBeInTheDocument();
  });

  it("兜底：role=manager 但 permissions 为空（旧缓存/降级）时，归档管理可见、待审批不可见", () => {
    renderWithUser(createUser([], "manager"));

    expect(tabByRole("归档管理")).toBeInTheDocument();
    expect(tabByRole("统计分析")).toBeInTheDocument();
    expect(tabByRole("待审批")).not.toBeInTheDocument();
  });
});
