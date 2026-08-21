import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, test, vi } from "vitest";
import { NavUser } from "@/components/layout/nav-user";
import { TooltipProvider } from "@/components/ui/tooltip";

const mocks = vi.hoisted(() => ({
  hasPermission: vi.fn(),
  logout: vi.fn(),
  setOpenMobile: vi.fn(),
  sidebar: { state: "expanded" as "expanded" | "collapsed", isMobile: false },
}));

vi.mock("@/lib/auth", () => ({
  useAuth: () => ({
    user: { full_name: "张三" },
    hasPermission: mocks.hasPermission,
    logout: mocks.logout,
  }),
}));

vi.mock("@/components/ui/sidebar", () => ({
  SidebarGroup: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  SidebarMenu: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  SidebarMenuItem: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  useSidebar: () => ({ ...mocks.sidebar, setOpenMobile: mocks.setOpenMobile }),
}));

vi.mock("@/components/avatar/user-avatar", () => ({
  UserAvatar: ({ name }: { name: string }) => <span data-testid="avatar">{name}</span>,
}));

describe("NavUser", () => {
  const onOpenSettings = vi.fn();

  beforeEach(() => {
    vi.clearAllMocks();
    mocks.hasPermission.mockReturnValue(true);
    mocks.sidebar.state = "expanded";
    mocks.sidebar.isMobile = false;
  });

  function renderNavUser() {
    return render(
      <TooltipProvider>
        <NavUser displayName="张三" subLine="zhangsan@example.com" onOpenSettings={onOpenSettings} />
      </TooltipProvider>,
    );
  }

  function openMenu(label = "打开账户菜单：个人信息") {
    const trigger = screen.getByRole("button", { name: label });
    fireEvent.pointerDown(trigger, { button: 0 });
  }

  test("展开态展示姓名邮箱和账户菜单三项", async () => {
    renderNavUser();
    expect(screen.getAllByText("张三")).toHaveLength(2);
    expect(screen.getByText("zhangsan@example.com")).toBeInTheDocument();

    openMenu();
    expect(await screen.findByRole("menuitem", { name: "个人信息" })).toBeInTheDocument();
    expect(screen.getByRole("menuitem", { name: "系统设置" })).toBeInTheDocument();
    expect(screen.getByRole("menuitem", { name: "退出系统" })).toBeInTheDocument();
  });

  test("无系统设置权限时隐藏该菜单项", async () => {
    mocks.hasPermission.mockReturnValue(false);
    renderNavUser();
    openMenu();
    expect(await screen.findByRole("menuitem", { name: "个人信息" })).toBeInTheDocument();
    expect(screen.queryByRole("menuitem", { name: "系统设置" })).not.toBeInTheDocument();
  });

  test("折叠态仅显示头像，并可通过提示打开菜单", async () => {
    mocks.sidebar.state = "collapsed";
    renderNavUser();
    expect(screen.getByTestId("avatar")).toBeInTheDocument();
    expect(screen.queryByText("zhangsan@example.com")).not.toBeInTheDocument();

    openMenu("打开账户菜单");
    expect(await screen.findByRole("menuitem", { name: "个人信息" })).toBeInTheDocument();
  });

  test("Escape 和外部点击关闭菜单", async () => {
    renderNavUser();
    openMenu();
    expect(await screen.findByRole("menuitem", { name: "个人信息" })).toBeInTheDocument();
    fireEvent.keyDown(document, { key: "Escape" });
    await waitFor(() => expect(screen.queryByRole("menuitem", { name: "个人信息" })).not.toBeInTheDocument());

    openMenu();
    expect(await screen.findByRole("menuitem", { name: "个人信息" })).toBeInTheDocument();
    fireEvent.pointerDown(document.body);
    await waitFor(() => expect(screen.queryByRole("menuitem", { name: "个人信息" })).not.toBeInTheDocument());
  });

  test("菜单操作触发设置回调、退出登录并在移动端关闭抽屉", async () => {
    mocks.sidebar.isMobile = true;
    renderNavUser();
    openMenu();
    fireEvent.click(await screen.findByRole("menuitem", { name: "个人信息" }));
    expect(onOpenSettings).toHaveBeenCalledWith("personal");
    expect(mocks.setOpenMobile).toHaveBeenCalledWith(false);

    openMenu();
    fireEvent.click(await screen.findByRole("menuitem", { name: "系统设置" }));
    expect(onOpenSettings).toHaveBeenCalledWith("system");

    openMenu();
    fireEvent.click(await screen.findByRole("menuitem", { name: "退出系统" }));
    expect(mocks.logout).toHaveBeenCalledOnce();
  });

  test("新壳菜单展示个人资料和我的反馈，系统设置仍受权限控制", async () => {
    const onOpenFeedback = vi.fn();
    render(
      <TooltipProvider>
        <NavUser displayName="张三" subLine="zhangsan@example.com" variant="new" onOpenSettings={onOpenSettings} onOpenFeedback={onOpenFeedback} />
      </TooltipProvider>,
    );
    openMenu();
    fireEvent.click(await screen.findByRole("menuitem", { name: "个人资料" }));
    expect(onOpenSettings).toHaveBeenCalledWith("personal");

    openMenu();
    fireEvent.click(await screen.findByRole("menuitem", { name: "我的反馈" }));
    expect(onOpenFeedback).toHaveBeenCalledOnce();

    fireEvent.keyDown(document, { key: "Escape" });
    mocks.hasPermission.mockReturnValue(false);
    render(
      <TooltipProvider>
        <NavUser displayName="张三" subLine="zhangsan@example.com" variant="new" onOpenSettings={onOpenSettings} onOpenFeedback={onOpenFeedback} />
      </TooltipProvider>,
    );
    const triggers = screen.getAllByRole("button", { name: "打开账户菜单：个人信息" });
    fireEvent.pointerDown(triggers[triggers.length - 1], { button: 0 });
    await screen.findByRole("menuitem", { name: "个人资料" });
    expect(screen.queryByRole("menuitem", { name: "系统设置" })).not.toBeInTheDocument();
  });
});
