import { fireEvent, render, screen } from "@testing-library/react";
import { beforeEach, describe, expect, test, vi } from "vitest";
import { NewAppSidebar } from "@/components/layout/new-app-sidebar";

const mocks = vi.hoisted(() => ({
  hasPermission: vi.fn(),
  setOpenMobile: vi.fn(),
  sidebar: { isMobile: false },
  authState: { user: { full_name: "普通用户", email: "viewer@example.com" } as { full_name: string; email: string; role?: string } },
}));

vi.mock("@/components/ui/sidebar", () => ({
  Sidebar: ({ children, mobileWidth, mobileMaxWidth, style }: { children: React.ReactNode; mobileWidth?: string; mobileMaxWidth?: string; style?: React.CSSProperties }) => <aside data-mobile-width={mobileWidth} data-mobile-max-width={mobileMaxWidth} style={style}>{children}</aside>,
  SidebarContent: ({ children, className }: { children: React.ReactNode; className?: string }) => <div className={className}>{children}</div>,
  SidebarHeader: ({ children, className }: { children: React.ReactNode; className?: string }) => <header className={className}>{children}</header>,
  SidebarFooter: ({ children, className }: { children: React.ReactNode; className?: string }) => <footer className={className}>{children}</footer>,
  SidebarMenu: ({ children, className }: { children: React.ReactNode; className?: string }) => <div className={className}>{children}</div>,
  SidebarMenuButton: ({ children, onClick, isActive }: { children: React.ReactNode; onClick?: () => void; isActive?: boolean }) => <button type="button" data-active={isActive} onClick={onClick}>{children}</button>,
  SidebarMenuItem: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  SidebarSeparator: () => <hr />,
  useSidebar: () => ({ ...mocks.sidebar, setOpenMobile: mocks.setOpenMobile }),
}));
vi.mock("@/lib/supabase/auth-context", () => ({ useAuth: () => ({ ...mocks.authState, hasPermission: mocks.hasPermission }) }));
vi.mock("@/components/layout/nav-user", () => ({ NavUser: ({ variant, onOpenFeedback }: { variant?: string; onOpenFeedback?: () => void }) => <button type="button" data-variant={variant} onClick={onOpenFeedback}>账户菜单</button> }));
vi.mock("@/components/feedback/my-feedback-dialog", () => ({ MyFeedbackDialog: ({ open }: { open: boolean }) => open ? <div role="dialog">我的反馈内容</div> : null }));

describe("NewAppSidebar", () => {
  const onViewChange = vi.fn();
  const onOpenSettings = vi.fn();
  const renderSidebar = (currentView = "landing") => render(<NewAppSidebar currentView={currentView} onViewChange={onViewChange} onOpenSettings={onOpenSettings} />);

  beforeEach(() => { vi.clearAllMocks(); mocks.sidebar.isMobile = false; mocks.hasPermission.mockReturnValue(true); });
  beforeEach(() => { mocks.authState.user = { full_name: "普通用户", email: "viewer@example.com" }; });

  test("保留系统标题既有滚动文字类", () => {
    renderSidebar();
    expect(screen.getAllByText("人事行政管理系统")[0].parentElement).toHaveClass("rolling-text", "text-base", "font-semibold");
  });

  test("使用190px桌面栏、64px折叠栏与受限85vw移动栏", () => {
    const { container } = renderSidebar();
    const sidebar = container.querySelector("aside");
    expect(sidebar).toHaveAttribute("data-mobile-width", "85vw");
    expect(sidebar).toHaveAttribute("data-mobile-max-width", "24rem");
    expect(sidebar).toHaveStyle({ "--sidebar-width": "11.875rem", "--sidebar-width-icon": "4rem" });
  });

  test("标题与用户区固定，只有侧栏内容可垂直滚动", () => {
    const { container } = renderSidebar();
    expect(container.querySelector("header")).toHaveClass("shrink-0");
    expect(container.querySelector("footer")).toHaveClass("shrink-0");
    expect(container.querySelector("header")?.nextElementSibling).toHaveClass("overflow-y-auto");
  });

  test("底部账户区使用新壳菜单并可打开既有我的反馈对话框", () => {
    renderSidebar();
    const accountMenu = screen.getByRole("button", { name: "账户菜单" });
    expect(accountMenu).toHaveAttribute("data-variant", "new");
    fireEvent.click(accountMenu);
    expect(screen.getByRole("dialog")).toHaveTextContent("我的反馈内容");
  });

  test("仅显示矩阵中已完成的真实入口，不显示后续或占位项", () => {
    renderSidebar();
    fireEvent.click(screen.getByRole("button", { name: "员工管理" }));
    expect(screen.getByRole("button", { name: "员工花名册" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "公积金管理" })).toBeInTheDocument();
    expect(screen.getAllByRole("button", { name: "公积金管理" })).toHaveLength(1);
    // 入职管理已完成真实接入；其余后续项继续隐藏
    expect(screen.getByRole("button", { name: "入职管理" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "转正管理" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "人事异动" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "劳动合同" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "奖惩记录" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "培训管理" })).toBeInTheDocument();
  });

  test("离职管理使用 employee.edit 权限显示，并可导航", () => {
    mocks.hasPermission.mockImplementation((resource: string, action: string) => resource === "employee" && action === "edit");
    renderSidebar();
    fireEvent.click(screen.getByRole("button", { name: "员工管理" }));
    expect(screen.getByRole("button", { name: "离职管理" })).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "离职管理" }));
    expect(onViewChange).toHaveBeenCalledWith("resignation");
  });

  test("人事异动使用 employee.edit 权限显示，并可导航", () => {
    mocks.hasPermission.mockImplementation((resource: string, action: string) => resource === "employee" && action === "edit");
    renderSidebar();
    fireEvent.click(screen.getByRole("button", { name: "员工管理" }));
    fireEvent.click(screen.getByRole("button", { name: "人事异动" }));
    expect(onViewChange).toHaveBeenCalledWith("personnel-changes");
  });

  test("入职管理使用 employee.create 权限显示，并可导航", () => {
    mocks.hasPermission.mockImplementation((resource: string, action: string) => resource === "employee" && action === "create");
    renderSidebar();
    fireEvent.click(screen.getByRole("button", { name: "员工管理" }));
    fireEvent.click(screen.getByRole("button", { name: "入职管理" }));
    expect(onViewChange).toHaveBeenCalledWith("onboarding");
    expect(screen.queryByRole("button", { name: "离职管理" })).not.toBeInTheDocument();
  });

  test("无 employee.edit 权限时隐藏离职管理", () => {
    mocks.hasPermission.mockImplementation((resource: string, action: string) => resource === "employee" && action === "view");
    renderSidebar();
    fireEvent.click(screen.getByRole("button", { name: "员工管理" }));
    expect(screen.queryByRole("button", { name: "离职管理" })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "转正管理" })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "劳动合同" })).not.toBeInTheDocument();
  });

  test("劳动合同沿用 contract.view 权限并可导航", () => {
    mocks.hasPermission.mockImplementation((resource: string, action: string) => resource === "contract" && action === "view");
    renderSidebar();
    fireEvent.click(screen.getByRole("button", { name: "员工管理" }));
    fireEvent.click(screen.getByRole("button", { name: "劳动合同" }));
    expect(onViewChange).toHaveBeenCalledWith("labor-contracts");
  });

  test("奖惩记录按 reward.view 权限展示并可导航", () => {
    mocks.hasPermission.mockImplementation((resource: string, action: string) => resource === "reward" && action === "view");
    renderSidebar();
    fireEvent.click(screen.getByRole("button", { name: "员工管理" }));
    fireEvent.click(screen.getByRole("button", { name: "奖惩记录" }));
    expect(onViewChange).toHaveBeenCalledWith("rewards");
  });

  test("培训管理按 training.view 权限展示并可导航", () => {
    mocks.hasPermission.mockImplementation((resource: string, action: string) => resource === "training" && action === "view");
    renderSidebar();
    fireEvent.click(screen.getByRole("button", { name: "员工管理" }));
    fireEvent.click(screen.getByRole("button", { name: "培训管理" }));
    expect(onViewChange).toHaveBeenCalledWith("training");
  });

  test("行政管理分组仅显示真实入口，合同/安全管理占位项隐藏", () => {
    renderSidebar();
    fireEvent.click(screen.getByRole("button", { name: "行政管理" }));
    expect(screen.getByRole("button", { name: "组织管理" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "社保管理" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "档案管理" })).toBeInTheDocument();
    // 🛑 占位项 2 项全部隐藏；"社保业务"占位与真实"社保管理"区分
    expect(screen.queryByRole("button", { name: "合同管理" })).not.toBeInTheDocument();
    expect(screen.getByRole("button", { name: "安全管理" })).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "社保业务" })).not.toBeInTheDocument();
  });

  test("社保管理仅按 insurance.view 权限显示并导航至真实视图", () => {
    mocks.hasPermission.mockImplementation((resource: string, action: string) => resource === "insurance" && action === "view");
    renderSidebar();
    fireEvent.click(screen.getByRole("button", { name: "行政管理" }));
    fireEvent.click(screen.getByRole("button", { name: "社保管理" }));
    expect(onViewChange).toHaveBeenCalledWith("insurance");
    expect(screen.queryByRole("button", { name: "社保业务" })).not.toBeInTheDocument();
  });

  test("行政合同按 admin_contract.view 权限展示并可导航", () => {
    mocks.hasPermission.mockImplementation((resource: string, action: string) => resource === "admin_contract" && action === "view");
    renderSidebar();
    fireEvent.click(screen.getByRole("button", { name: "行政管理" }));
    fireEvent.click(screen.getByRole("button", { name: "行政合同" }));
    expect(onViewChange).toHaveBeenCalledWith("admin-contracts");
  });

  test("安全管理按 safety.view 权限展示并可导航", () => {
    mocks.hasPermission.mockImplementation((resource: string, action: string) => resource === "safety" && action === "view");
    renderSidebar();
    fireEvent.click(screen.getByRole("button", { name: "行政管理" }));
    fireEvent.click(screen.getByRole("button", { name: "安全管理" }));
    expect(onViewChange).toHaveBeenCalledWith("safety");
  });

  test("职业健康检查按 occupational_health.view 权限展示并可导航", () => {
    mocks.hasPermission.mockImplementation((resource: string, action: string) => resource === "occupational_health" && action === "view");
    renderSidebar();
    fireEvent.click(screen.getByRole("button", { name: "行政管理" }));
    fireEvent.click(screen.getByRole("button", { name: "职业健康检查" }));
    expect(onViewChange).toHaveBeenCalledWith("occupational-health");
  });

  test("日常事务分组展示有 dormitory.view 权限的能耗管理和 fleet.view 权限的车辆档案", () => {
    renderSidebar();
    fireEvent.click(screen.getByRole("button", { name: "日常事务" }));
    expect(screen.getByRole("button", { name: "宿舍管理" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "能耗管理" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "食堂管理" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "发票管理" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "办公劳保" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "车辆档案" })).toBeInTheDocument();
    // 历史占位名称不应作为重复入口出现
    expect(screen.queryByRole("button", { name: "车队管理" })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "职业卫生" })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "社保业务" })).not.toBeInTheDocument();
  });

  test("能耗管理仅按 dormitory.view 权限显示并导航", () => {
    mocks.hasPermission.mockImplementation((resource: string, action: string) => resource === "dormitory" && action === "view");
    renderSidebar();
    fireEvent.click(screen.getByRole("button", { name: "日常事务" }));
    fireEvent.click(screen.getByRole("button", { name: "能耗管理" }));
    expect(onViewChange).toHaveBeenCalledWith("energy");
  });

  test("无 dormitory.view 权限时隐藏能耗管理入口", () => {
    mocks.hasPermission.mockImplementation((resource: string) => resource !== "dormitory");
    renderSidebar();
    fireEvent.click(screen.getByRole("button", { name: "日常事务" }));
    expect(screen.queryByRole("button", { name: "能耗管理" })).not.toBeInTheDocument();
  });

  test("车辆档案仅按 fleet.view 显示并导航至独立视图", () => {
    mocks.hasPermission.mockImplementation((resource: string, action: string) => resource === "fleet" && action === "view");
    renderSidebar();
    fireEvent.click(screen.getByRole("button", { name: "日常事务" }));
    fireEvent.click(screen.getByRole("button", { name: "车辆档案" }));
    expect(onViewChange).toHaveBeenCalledWith("fleet-vehicles");
  });

  test("按既有资源查看权限过滤顶级和分组入口，并隐藏空分组", () => {
    mocks.hasPermission.mockImplementation((resource: string) => ["employee", "users", "department"].includes(resource));
    renderSidebar();
    expect(screen.getByRole("button", { name: "工作台" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "偏好设置" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "反馈管理" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "部门管理" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "员工管理" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "行政管理" })).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "日常事务" })).not.toBeInTheDocument();
    expect(mocks.hasPermission).toHaveBeenCalledWith("users", "view");
    expect(mocks.hasPermission).toHaveBeenCalledWith("department", "view");
    expect(mocks.hasPermission).toHaveBeenCalledWith("employee", "view");
  });

  test("部门管理仅对旧壳角色进行可见性兜底，普通用户无权限时隐藏", () => {
    mocks.hasPermission.mockImplementation((resource: string) => resource !== "department");
    renderSidebar();
    expect(screen.queryByRole("button", { name: "部门管理" })).not.toBeInTheDocument();
  });

  test("部门管理在旧壳管理员无部门权限时仍可见", () => {
    mocks.authState.user = { full_name: "管理员", email: "admin@example.com", role: "admin" };
    mocks.hasPermission.mockImplementation((resource: string) => resource !== "department");
    renderSidebar();
    expect(screen.getByRole("button", { name: "部门管理" })).toBeInTheDocument();
  });

  test("组织管理归入行政管理，并沿用管理员角色兜底", () => {
    mocks.authState.user = { full_name: "管理员", email: "admin@example.com", username: "admin", role: "admin" } as typeof mocks.authState.user;
    mocks.hasPermission.mockImplementation((resource: string) => resource !== "department");
    renderSidebar();
    fireEvent.click(screen.getByRole("button", { name: "行政管理" }));
    expect(screen.getByRole("button", { name: "组织管理" })).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "组织管理" }));
    expect(onViewChange).toHaveBeenCalledWith("organization");
  });

  test("无部门权限的普通用户隐藏组织管理和行政管理空分组", () => {
    mocks.hasPermission.mockImplementation((resource: string) => resource !== "department" && resource !== "insurance");
    renderSidebar();
    expect(screen.queryByRole("button", { name: "组织管理" })).not.toBeInTheDocument();
  });

  test("分组互斥展开，激活入口可导航且移动端关闭抽屉", () => {
    mocks.sidebar.isMobile = true;
    renderSidebar("employee");
    fireEvent.click(screen.getByRole("button", { name: "员工管理" }));
    expect(screen.getByRole("button", { name: "员工花名册" })).toHaveAttribute("data-active", "true");
    fireEvent.click(screen.getByRole("button", { name: "员工花名册" }));
    expect(onViewChange).toHaveBeenCalledWith("employee");
    expect(mocks.setOpenMobile).toHaveBeenCalledWith(false);
    fireEvent.click(screen.getByRole("button", { name: "行政管理" }));
    expect(screen.queryByRole("button", { name: "员工花名册" })).not.toBeInTheDocument();
  });
});
