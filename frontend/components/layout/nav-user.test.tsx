import { fireEvent, render, screen, within } from "@testing-library/react";
import { beforeEach, describe, expect, test, vi } from "vitest";
import { NavUser } from "@/components/layout/nav-user";

const mocks = vi.hoisted(() => ({
  hasPermission: vi.fn(),
  logout: vi.fn(),
  setOpenMobile: vi.fn(),
  sidebar: { state: "expanded", isMobile: false },
  authUser: { full_name: "张三", email: "zhang@example.com" } as { full_name: string; email: string },
}));

// sidebar mock：SidebarGroup 记录 className 以断言双层内边距收敛
vi.mock("@/components/ui/sidebar", () => ({
  SidebarGroup: ({ children, className }: { children: React.ReactNode; className?: string }) => <div data-slot="sidebar-group" className={className}>{children}</div>,
  SidebarMenu: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  SidebarMenuItem: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  useSidebar: () => ({ ...mocks.sidebar, setOpenMobile: mocks.setOpenMobile }),
}));

vi.mock("@/lib/auth", () => ({
  useAuth: () => ({ user: mocks.authUser, hasPermission: mocks.hasPermission, logout: mocks.logout }),
}));

// 头像 mock：真实组件为图形头像，这里仅保留结构占位、不渲染文本
vi.mock("@/components/avatar/user-avatar", () => ({
  UserAvatar: ({ className }: { className?: string }) => <div data-slot="avatar" className={className} />,
}));

// tooltip mock：折叠态分支透传即可
vi.mock("@/components/ui/tooltip", () => ({
  Tooltip: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  TooltipTrigger: ({ children }: { children: React.ReactNode }) => <>{children}</>,
  TooltipContent: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
}));

// dropdown-menu mock：Content 直渲染 children，便于断言分组结构与类名契约；
// Item 将 radix onSelect 映射到 onClick，保持回调语义可测
vi.mock("@/components/ui/dropdown-menu", () => ({
  DropdownMenu: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  DropdownMenuTrigger: ({ children }: { children: React.ReactElement }) => children,
  DropdownMenuContent: ({ children, className }: { children: React.ReactNode; className?: string }) => <div role="menu" className={className}>{children}</div>,
  DropdownMenuItem: ({ children, className, onSelect }: { children: React.ReactNode; className?: string; onSelect?: (event: Event) => void }) => (
    <button type="button" role="menuitem" className={className} onClick={() => onSelect?.(new Event("select"))}>{children}</button>
  ),
  DropdownMenuSeparator: () => <div role="separator" />,
}));

const baseProps = { displayName: "张三", subLine: "zhang@example.com", onOpenSettings: vi.fn() };

function renderNavUser(overrides?: Partial<Parameters<typeof NavUser>[0]>) {
  const props = { ...baseProps, ...overrides };
  return render(<NavUser {...props} />);
}

beforeEach(() => {
  vi.clearAllMocks();
  mocks.sidebar.state = "expanded";
  mocks.sidebar.isMobile = false;
  mocks.hasPermission.mockReturnValue(true);
});

describe("NavUser 弹出菜单分组结构（参照新壳视觉规范）", () => {
  test("顶部信息头渲染姓名(font-semibold)与 subLine(muted)，且非交互区块", () => {
    renderNavUser({ variant: "new" });
    const info = document.querySelector('[data-slot="nav-user-info"]');
    expect(info).not.toBeNull();
    const nameEl = within(info as HTMLElement).getByText("张三");
    expect(nameEl).toHaveClass("font-semibold");
    const subEl = within(info as HTMLElement).getByText("zhang@example.com");
    expect(subEl).toHaveClass("text-muted-foreground");
    // 信息头不是可点击菜单项
    expect(screen.queryByRole("menuitem", { name: /张三/ })).not.toBeInTheDocument();
  });

  test("分组顺序：信息头 → 分隔线 → 功能组 → 分隔线 → 危险项，共两条分隔线", () => {
    renderNavUser({ variant: "new", onOpenFeedback: vi.fn() });
    const menu = screen.getByRole("menu");
    const children = Array.from(menu.children) as HTMLElement[];
    const separators = within(menu).getAllByRole("separator");
    expect(separators).toHaveLength(2);
    // 结构契约：信息头在最前，其后第一条分隔线，退出系统在最后一条分隔线之后
    const indexOf = (el: Element) => children.indexOf(el as HTMLElement);
    expect(indexOf(children[0].querySelector('[data-slot="nav-user-info"]') ?? children[0])).toBeLessThan(indexOf(separators[0]));
    expect(within(menu).getByRole("menuitem", { name: "个人资料" })).toBeInTheDocument();
    expect(within(menu).getByRole("menuitem", { name: "我的反馈" })).toBeInTheDocument();
    expect(within(menu).getByRole("menuitem", { name: "系统设置" })).toBeInTheDocument();
    const logout = within(menu).getByRole("menuitem", { name: "退出系统" });
    expect(indexOf(separators[1])).toBeLessThan(indexOf(logout));
    expect(logout).toBe(children[children.length - 1]);
  });

  test("退出系统为危险项：携带 text-destructive 类契约", () => {
    renderNavUser({ variant: "new" });
    expect(screen.getByRole("menuitem", { name: "退出系统" })).toHaveClass("text-destructive");
  });

  test("new 变体菜单加宽至 w-56 匹配分组布局", () => {
    renderNavUser({ variant: "new" });
    expect(screen.getByRole("menu")).toHaveClass("w-56");
  });

  test("功能组条件渲染：无反馈入口或无 settings.view 权限时对应项隐藏", () => {
    const first = renderNavUser({ variant: "new" });
    expect(screen.queryByRole("menuitem", { name: "我的反馈" })).not.toBeInTheDocument();
    expect(screen.getByRole("menuitem", { name: "系统设置" })).toBeInTheDocument();
    first.unmount();

    mocks.hasPermission.mockReturnValue(false);
    renderNavUser({ variant: "new", onOpenFeedback: vi.fn() });
    expect(screen.getByRole("menuitem", { name: "个人资料" })).toBeInTheDocument();
    expect(screen.queryByRole("menuitem", { name: "系统设置" })).not.toBeInTheDocument();
  });

  test("既有 onSelect 行为保留：反馈回调、设置分发与退出登录；个人资料改走弹窗", () => {
    const onOpenSettings = vi.fn();
    const onOpenFeedback = vi.fn();
    renderNavUser({ variant: "new", onOpenSettings, onOpenFeedback });
    fireEvent.click(screen.getByRole("menuitem", { name: "我的反馈" }));
    expect(onOpenFeedback).toHaveBeenCalledTimes(1);
    fireEvent.click(screen.getByRole("menuitem", { name: "系统设置" }));
    expect(onOpenSettings).toHaveBeenCalledWith("system");
    fireEvent.click(screen.getByRole("menuitem", { name: "退出系统" }));
    expect(mocks.logout).toHaveBeenCalledTimes(1);
    // 桌面展开态不触发移动抽屉关闭
    expect(mocks.setOpenMobile).not.toHaveBeenCalled();
    // 新壳"个人资料"不再派发视图切换，直接打开个人资料模态弹窗
    fireEvent.click(screen.getByRole("menuitem", { name: "个人资料" }));
    expect(onOpenSettings).toHaveBeenCalledTimes(1);
    expect(screen.getByRole("dialog", { name: "个人资料" })).toBeInTheDocument();
  });

  test("default 变体成立：个人信息文案、w-52 宽度且信息头同样渲染", () => {
    const onOpenSettings = vi.fn();
    renderNavUser({ onOpenSettings });
    expect(screen.getByRole("menuitem", { name: "个人信息" })).toBeInTheDocument();
    expect(screen.queryByRole("menuitem", { name: "个人资料" })).not.toBeInTheDocument();
    expect(screen.getByRole("menu")).toHaveClass("w-52");
    expect(document.querySelector('[data-slot="nav-user-info"]')).not.toBeNull();
    // 旧壳保留个人信息视图切换契约
    fireEvent.click(screen.getByRole("menuitem", { name: "个人信息" }));
    expect(onOpenSettings).toHaveBeenCalledWith("personal");
  });
});

describe("头像区内边距收敛", () => {
  test("SidebarGroup 收敛为 p-0 消除双层内边距，触发按钮 px-3 加宽点击域", () => {
    renderNavUser({ variant: "new" });
    expect(document.querySelector('[data-slot="sidebar-group"]')).toHaveClass("p-0");
    expect(screen.getByRole("button", { name: "打开账户菜单：个人信息" })).toHaveClass("px-3");
  });

  test("折叠态行为不变：仅头像触发，触发按钮不渲染文字行", () => {
    mocks.sidebar.state = "collapsed";
    renderNavUser({ variant: "new" });
    const trigger = screen.getByRole("button", { name: "打开账户菜单" });
    // 折叠态触发行内不应出现姓名/邮箱文字（菜单信息头除外）
    expect(within(trigger).queryByText("张三")).not.toBeInTheDocument();
    expect(within(trigger).queryByText("zhang@example.com")).not.toBeInTheDocument();
  });
});
