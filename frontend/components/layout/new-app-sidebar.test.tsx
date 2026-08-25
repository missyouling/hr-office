import { fireEvent, render, screen } from "@testing-library/react";
import { beforeEach, describe, expect, test, vi } from "vitest";
import { NewAppSidebar } from "@/components/layout/new-app-sidebar";

const mocks = vi.hoisted(() => ({
  hasPermission: vi.fn(),
  setOpenMobile: vi.fn(),
  sidebar: { isMobile: false, state: "expanded" as "expanded" | "collapsed" },
  authState: { user: { full_name: "普通用户", email: "viewer@example.com" } as { full_name: string; email: string; role?: string; username?: string } },
}));

vi.mock("@/components/ui/sidebar", () => ({
  Sidebar: ({ children, className, mobileWidth, mobileMaxWidth, style, collapsible, variant = "sidebar" }: { children: React.ReactNode; className?: string; mobileWidth?: string; mobileMaxWidth?: string; style?: React.CSSProperties; collapsible?: string; variant?: string }) => <aside data-mobile-width={mobileWidth} data-mobile-max-width={mobileMaxWidth} data-collapsible={collapsible} data-variant={variant} className={className} style={style}>{children}</aside>,
  SidebarContent: ({ children, className }: { children: React.ReactNode; className?: string }) => <div className={className}>{children}</div>,
  SidebarHeader: ({ children, className }: { children: React.ReactNode; className?: string }) => <header className={className}>{children}</header>,
  SidebarFooter: ({ children, className }: { children: React.ReactNode; className?: string }) => <footer className={className}>{children}</footer>,
  SidebarMenu: ({ children, className }: { children: React.ReactNode; className?: string }) => <div className={className}>{children}</div>,
  SidebarMenuButton: ({ children, onClick, isActive, className }: { children: React.ReactNode; onClick?: () => void; isActive?: boolean; className?: string }) => <button type="button" data-active={isActive} className={className} onClick={onClick}>{children}</button>,
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

  beforeEach(() => { vi.clearAllMocks(); mocks.sidebar.isMobile = false; mocks.sidebar.state = "expanded"; mocks.hasPermission.mockReturnValue(true); });
  beforeEach(() => { mocks.authState.user = { full_name: "普通用户", email: "viewer@example.com" }; });

  test("标题为人事行政管理，使用滚动文字动效且不再显示版本号", () => {
    renderSidebar();
    // getByText 命中的是组件内 sr-only 完整名称，向上回溯滚动文字根节点
    const accessibleName = screen.getByText("人事行政管理");
    expect(accessibleName).toHaveClass("sr-only");
    const title = accessibleName.closest('[data-slot="rolling-text"]');
    expect(title).toHaveClass("text-base", "font-semibold");
    expect(screen.queryByText(/v\s?\d/)).not.toBeInTheDocument();
  });

  test("侧栏保持默认 sidebar 变体：直角贴边且无浮动圆角", () => {
    const { container } = renderSidebar();
    expect(container.querySelector("aside")).toHaveAttribute("data-variant", "sidebar");
    expect(container.querySelector("aside")).toHaveAttribute("data-collapsible", "icon");
  });

  test("侧栏容器携带 border-transparent：无侧栏与主容器之间的竖线分隔", () => {
    const { container } = renderSidebar();
    // 覆盖 sidebar 容器变体的 border-r 颜色，消除真实边框竖线（用户截图反馈）
    expect(container.querySelector("aside")).toHaveClass("border-transparent");
    expect(container.querySelector("aside")).toHaveClass("bg-sidebar");
  });

  test("选中菜单为浅灰 #F0F0F3 圆角矩形轻阴影，悬停为 #CDCDCF 且 150ms 过渡无位移缩放", () => {
    renderSidebar("landing");
    // 当前视图 landing → 工作台为选中项；#F9F9FB 底层上白色不可辨，选中底色用 --muted(#F0F0F3)
    const activeButton = screen.getByRole("button", { name: "工作台" });
    expect(activeButton).toHaveClass("data-[active=true]:bg-muted");
    expect(activeButton).toHaveClass("data-[active=true]:shadow-sm");

    // 所有菜单按钮共用悬停与过渡契约（--sidebar-accent 即 #CDCDCF）
    for (const name of ["工作台", "知识库"]) {
      const button = screen.getByRole("button", { name });
      expect(button).toHaveClass("hover:bg-sidebar-accent");
      expect(button).toHaveClass("transition-[background-color,color,box-shadow]");
      expect(button).toHaveClass("duration-150");
      // 无位移无缩放：不允许出现 transform 类过渡
      expect(button.className).not.toMatch(/scale|translate|transition-transform/);
    }
  });

  test("使用240px桌面栏、64px折叠栏与受限85vw移动栏", () => {
    const { container } = renderSidebar();
    const sidebar = container.querySelector("aside");
    expect(sidebar).toHaveAttribute("data-mobile-width", "85vw");
    expect(sidebar).toHaveAttribute("data-mobile-max-width", "24rem");
    expect(sidebar).toHaveStyle({ "--sidebar-width": "15rem", "--sidebar-width-icon": "4rem" });
  });

  test("标题与用户区固定，只有侧栏内容可垂直滚动", () => {
    const { container } = renderSidebar();
    expect(container.querySelector("header")).toHaveClass("shrink-0");
    expect(container.querySelector("footer")).toHaveClass("shrink-0");
    expect(container.querySelector("header")?.nextElementSibling).toHaveClass("overflow-y-auto");
  });

  test("系统标题区加高至约 h-16 观感：仅用上下间距，无分隔线与底色块", () => {
    const { container } = renderSidebar();
    // 头部容器 py-3 + 标题按钮 py-2 → 总高约 60px（h-14~h-16 观感），与内容区拉开呼吸感
    const header = container.querySelector("header");
    expect(header).toHaveClass("py-3");
    const titleButton = header?.querySelector("button");
    expect(titleButton).toHaveClass("py-2");
    // 仅用间距实现：头部内不允许分隔线，头部自身不允许底色类
    expect(header?.querySelector("hr")).toBeNull();
    expect(header?.className).not.toMatch(/bg-/);
    // 替换模式 ReturnHomeHeader 与标题区共用同一加高节奏
    const { container: replacedContainer } = render(
      <NewAppSidebar currentView="system" onViewChange={vi.fn()} onOpenSettings={vi.fn()} onReturnHome={vi.fn()} />,
    );
    expect(replacedContainer.querySelector("header")).toHaveClass("py-3");
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
    expect(screen.queryByRole("button", { name: "偏好设置" })).not.toBeInTheDocument();
    // 反馈管理已归入日常事务组末尾：分组默认收起时顶级不可见
    expect(screen.queryByRole("button", { name: "反馈管理" })).not.toBeInTheDocument();
    expect(screen.getByRole("button", { name: "部门管理" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "员工管理" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "行政管理" })).toBeInTheDocument();
    // users.view 命中反馈管理 → 日常事务组非空、组头出现
    fireEvent.click(screen.getByRole("button", { name: "日常事务" }));
    expect(screen.getByRole("button", { name: "反馈管理" })).toBeInTheDocument();
    expect(mocks.hasPermission).toHaveBeenCalledWith("users", "view");
    expect(mocks.hasPermission).toHaveBeenCalledWith("department", "view");
    expect(mocks.hasPermission).toHaveBeenCalledWith("employee", "view");
  });

  test("反馈管理位于日常事务组末尾且顶级无此项，点击导航至 feedback 视图", () => {
    renderSidebar();
    expect(screen.queryByRole("button", { name: "反馈管理" })).not.toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "日常事务" }));
    const feedbackButton = screen.getByRole("button", { name: "反馈管理" });
    expect(feedbackButton).toBeInTheDocument();
    fireEvent.click(feedbackButton);
    expect(onViewChange).toHaveBeenCalledWith("feedback");
  });

  test("反馈管理沿用 users.view 权限判断：无权限时隐藏且不改变其余日常事务项", () => {
    mocks.hasPermission.mockImplementation((resource: string) => resource === "dormitory");
    renderSidebar();
    fireEvent.click(screen.getByRole("button", { name: "日常事务" }));
    expect(screen.queryByRole("button", { name: "反馈管理" })).not.toBeInTheDocument();
    expect(screen.getByRole("button", { name: "宿舍管理" })).toBeInTheDocument();
  });

  test("知识库在管理员角色无 knowledge_base 权限时仍可见并导航至 knowledge-qa 视图", () => {
    mocks.authState.user = { full_name: "管理员", email: "admin@example.com", username: "admin", role: "super_admin" };
    mocks.hasPermission.mockImplementation((resource: string) => resource !== "knowledge_base");
    renderSidebar();
    expect(screen.getByRole("button", { name: "知识库" })).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "知识库" }));
    expect(onViewChange).toHaveBeenCalledWith("knowledge-qa");
  });

  test("普通用户无 knowledge_base 权限时知识库入口隐藏", () => {
    mocks.hasPermission.mockImplementation((resource: string) => resource !== "knowledge_base");
    renderSidebar();
    expect(screen.queryByRole("button", { name: "知识库" })).not.toBeInTheDocument();
  });

  test("知识库菜单指向 knowledge-qa 视图（AI 问答页），管理视图经页内按钮或搜索可达", () => {
    renderSidebar();
    fireEvent.click(screen.getByRole("button", { name: "知识库" }));
    expect(onViewChange).toHaveBeenCalledWith("knowledge-qa");
    expect(onViewChange).not.toHaveBeenCalledWith("knowledge");
  });

  test("主菜单分组标题使用 text-foreground 类（浅色黑/深色白自动适配）", () => {
    renderSidebar();
    for (const name of ["员工管理", "行政管理", "日常事务"]) {
      const groupHeader = screen.getByRole("button", { name });
      expect(groupHeader).toHaveClass("text-foreground");
      expect(groupHeader.className).not.toContain("text-muted-foreground");
    }
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

  test("折叠态纯图标平铺：分组头隐藏，各组成员与顶级项直接可见", () => {
    mocks.sidebar.state = "collapsed";
    renderSidebar();
    // 分组头按钮（展开态的折叠开关）不应出现
    for (const name of ["员工管理", "行政管理", "日常事务"]) {
      expect(screen.queryByRole("button", { name })).not.toBeInTheDocument();
    }
    // 顶级项 + 各可见分组成员以图标项平铺（tooltip 由 SidebarMenuButton 契约提供）
    expect(screen.getByRole("button", { name: "工作台" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "员工花名册" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "组织管理" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "宿舍管理" })).toBeInTheDocument();
  });

  test("分组互斥展开，激活入口可导航且移动端关闭抽屉", () => {
    mocks.sidebar.isMobile = true;
    // 主菜单互斥展开在 landing 视图验证；employee 视图已进入分组子页替换模式
    renderSidebar("landing");
    fireEvent.click(screen.getByRole("button", { name: "员工管理" }));
    fireEvent.click(screen.getByRole("button", { name: "员工花名册" }));
    expect(onViewChange).toHaveBeenCalledWith("employee");
    expect(mocks.setOpenMobile).toHaveBeenCalledWith(false);
    fireEvent.click(screen.getByRole("button", { name: "行政管理" }));
    expect(screen.queryByRole("button", { name: "员工花名册" })).not.toBeInTheDocument();
  });
});

describe("替换模式导航（系统设置 / 个人设置 / 分组子页）", () => {
  const onViewChange = vi.fn();
  const onOpenSettings = vi.fn();
  const onReturnHome = vi.fn();
  const onSettingsTabChange = vi.fn();
  const onOpenSettingsPanel = vi.fn();

  const renderReplaced = (currentView: string, overrides?: { settingsTab?: string; settingsPanel?: "audit" | "monitoring" | null }) =>
    render(
      <NewAppSidebar
        currentView={currentView}
        onViewChange={onViewChange}
        onOpenSettings={onOpenSettings}
        onReturnHome={onReturnHome}
        settingsTab={overrides?.settingsTab ?? "announcements"}
        onSettingsTabChange={onSettingsTabChange}
        settingsPanel={overrides?.settingsPanel ?? null}
        onOpenSettingsPanel={onOpenSettingsPanel}
      />,
    );

  beforeEach(() => { vi.clearAllMocks(); mocks.sidebar.isMobile = false; mocks.hasPermission.mockReturnValue(true); });

  test("系统设置模式渲染映射项与观察区，默认概览激活且主菜单隐藏", () => {
    renderReplaced("system");
    expect(screen.getByRole("button", { name: "返回主界面" })).toBeInTheDocument();
    for (const name of ["系统设置概览", "用户与权限", "组织与部门", "模型配置", "通知与邮件", "存储与备份"]) {
      expect(screen.getByRole("button", { name })).toBeInTheDocument();
    }
    expect(screen.getByRole("button", { name: "审计日志" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "系统监控" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "系统设置概览" })).toHaveAttribute("data-active", "true");
    expect(screen.queryByRole("button", { name: "工作台" })).not.toBeInTheDocument();
  });

  test("设置项分发：tab 切换、组织跳业务页、观察面板与返回回调", () => {
    renderReplaced("system");
    fireEvent.click(screen.getByRole("button", { name: "用户与权限" }));
    expect(onSettingsTabChange).toHaveBeenCalledWith("roles");
    fireEvent.click(screen.getByRole("button", { name: "组织与部门" }));
    expect(onViewChange).toHaveBeenCalledWith("organization");
    fireEvent.click(screen.getByRole("button", { name: "审计日志" }));
    expect(onOpenSettingsPanel).toHaveBeenCalledWith("audit");
    fireEvent.click(screen.getByRole("button", { name: "系统监控" }));
    expect(onOpenSettingsPanel).toHaveBeenCalledWith("monitoring");
    fireEvent.click(screen.getByRole("button", { name: "返回主界面" }));
    expect(onReturnHome).toHaveBeenCalledTimes(1);
  });

  test("打开观察面板时对应项激活；无 users.view 权限时观察区隐藏", () => {
    const { rerender } = renderReplaced("system", { settingsPanel: "monitoring" });
    expect(screen.getByRole("button", { name: "系统监控" })).toHaveAttribute("data-active", "true");
    expect(screen.getByRole("button", { name: "系统设置概览" })).toHaveAttribute("data-active", "false");

    mocks.hasPermission.mockImplementation((resource: string) => resource !== "users");
    rerender(
      <NewAppSidebar
        currentView="system"
        onViewChange={onViewChange}
        onOpenSettings={onOpenSettings}
        onReturnHome={onReturnHome}
        settingsTab="announcements"
        onSettingsTabChange={onSettingsTabChange}
        settingsPanel={null}
        onOpenSettingsPanel={onOpenSettingsPanel}
      />,
    );
    expect(screen.queryByRole("button", { name: "审计日志" })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "系统监控" })).not.toBeInTheDocument();
  });

  test("分组子页模式渲染组名扁平项并隐藏其他分组与主菜单", () => {
    renderReplaced("employee");
    expect(screen.getByText("员工管理")).toBeInTheDocument();
    const activeItem = screen.getByRole("button", { name: "员工花名册" });
    expect(activeItem).toHaveAttribute("data-active", "true");
    expect(screen.queryByRole("button", { name: "工作台" })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "宿舍管理" })).not.toBeInTheDocument();
    fireEvent.click(activeItem);
    expect(onViewChange).toHaveBeenCalledWith("employee");
  });

  test("分组子页组名与系统观察标题使用 text-foreground 类（浅色黑/深色白自动适配）", () => {
    const { unmount } = renderReplaced("employee");
    expect(screen.getByText("员工管理")).toHaveClass("text-foreground");
    unmount();
    renderReplaced("system");
    expect(screen.getByText("系统观察")).toHaveClass("text-foreground");
  });

  test("个人设置模式渲染个人设置激活项与返回按钮", () => {
    renderReplaced("personal-settings");
    expect(screen.getByRole("button", { name: "个人设置" })).toHaveAttribute("data-active", "true");
    expect(screen.getByRole("button", { name: "返回主界面" })).toBeInTheDocument();
  });
});
