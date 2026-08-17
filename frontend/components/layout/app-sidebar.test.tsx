import { fireEvent, render, screen } from "@testing-library/react";
import { beforeEach, describe, expect, test, vi } from "vitest";
import { AppSidebar } from "@/components/layout/app-sidebar";

const mocks = vi.hoisted(() => ({
  hasPermission: vi.fn(),
  setOpenMobile: vi.fn(),
}));

vi.mock("@/lib/supabase/auth-context", () => ({
  useAuth: () => ({
    user: { full_name: "普通用户", email: "viewer@example.com", role: "viewer" },
    hasPermission: mocks.hasPermission,
  }),
}));

vi.mock("@/components/ui/sidebar", () => ({
  Sidebar: ({ children }: { children: React.ReactNode }) => <aside>{children}</aside>,
  SidebarContent: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  SidebarFooter: ({ children }: { children: React.ReactNode }) => <footer>{children}</footer>,
  SidebarHeader: ({ children }: { children: React.ReactNode }) => <header>{children}</header>,
  SidebarMenu: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  SidebarMenuButton: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  SidebarMenuItem: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  SidebarSeparator: () => <hr />,
  useSidebar: () => ({ isMobile: false, setOpenMobile: mocks.setOpenMobile }),
}));

vi.mock("@/components/layout/nav-main", () => ({
  NavMain: ({ items }: { items: Array<{ label: string }> }) => <nav>{items.map((item) => <span key={item.label}>{item.label}</span>)}</nav>,
}));

vi.mock("@/components/layout/nav-documents", () => ({
  NavDocuments: () => null,
}));

vi.mock("@/components/layout/nav-user", () => ({
  NavUser: () => <div>账户菜单</div>,
}));

vi.mock("@/components/feedback/my-feedback-dialog", () => ({
  MyFeedbackDialog: ({ open, onOpenChange }: { open: boolean; onOpenChange: (open: boolean) => void }) => (
    open ? <div role="dialog"><button type="button" onClick={() => onOpenChange(false)}>关闭我的反馈</button></div> : null
  ),
}));

describe("AppSidebar", () => {
  const onViewChange = vi.fn();
  const onOpenSettings = vi.fn();

  beforeEach(() => {
    vi.clearAllMocks();
    mocks.hasPermission.mockReturnValue(false);
  });

  function renderSidebar() {
    return render(<AppSidebar currentView="landing" onViewChange={onViewChange} onOpenSettings={onOpenSettings} />);
  }

  test("普通 viewer 可打开我的反馈，但看不到反馈管理", () => {
    renderSidebar();

    expect(screen.getByRole("button", { name: "我的反馈" })).toBeInTheDocument();
    expect(screen.queryByText("反馈管理")).not.toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: "我的反馈" }));
    expect(screen.getByRole("dialog")).toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: "关闭我的反馈" }));
    expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
  });

  test("反馈管理仍仅向拥有 users.view 权限的用户显示", () => {
    mocks.hasPermission.mockImplementation((resource: string, action: string) => resource === "users" && action === "view");

    renderSidebar();

    expect(screen.getByText("反馈管理")).toBeInTheDocument();
  });
});
