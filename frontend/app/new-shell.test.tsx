import { fireEvent, render, screen } from "@testing-library/react";
import { readFileSync } from "node:fs";
import { join } from "node:path";
import { beforeEach, describe, expect, test, vi } from "vitest";

// app-shell 与新壳都在 @/app/page 中导出；为避免拉起 HomePage 渲染链，这里只测导出的壳层契约
import { APP_MAIN_CONTAINER_SLOT, APP_SHELL_CLASS, APP_SIDEBAR_WIDTH_VARS, AppMainContainer, MAIN_SCROLL_CLASS, MAIN_SURFACE_CLASS, NewShell } from "@/components/layout/app-shell";
import { DOCK_POSITION_CLASS, ManagementBar } from "@/components/layout/management-bar";

// ManagementBar 依赖隔离：FloatingDock 退化为平铺按钮，其余外部依赖给最小桩实现
vi.mock("@/components/ui/sidebar", () => ({ useSidebar: () => ({ toggleSidebar: vi.fn() }) }));
vi.mock("@/components/ui/floating-dock", () => ({
  FloatingDock: ({ items }: { items: Array<{ title: string; onClick?: () => void }> }) => (
    <nav aria-label="快捷操作">
      {items.map((item) => <button key={item.title} type="button" aria-label={item.title} onClick={item.onClick}>{item.title}</button>)}
    </nav>
  ),
}));
vi.mock("@/hooks/use-theme-utils", () => ({
  useThemeUtils: () => ({ toggle: () => "dark", getAction: () => "切换主题", getIcon: () => null }),
}));
vi.mock("@/lib/dorm-notifications", () => ({ getSiteNotificationCount: () => 0 }));
vi.mock("@/lib/api", () => ({ updateUserPreferences: vi.fn().mockResolvedValue(undefined) }));

describe("NewShell", () => {
  test("新壳仅保留标记节点，不再提供右上角固定通知按钮", () => {
    const { container } = render(<NewShell><main>应用内容</main></NewShell>);

    // 通知入口已收敛至控制坞：壳内不允许残留任何工具栏按钮
    expect(container.querySelector('[data-shell="new"]')).not.toBeNull();
    expect(screen.queryByRole("button", { name: "打开通知中心" })).not.toBeInTheDocument();
    expect(container.querySelector("button")).toBeNull();
    expect(screen.getByText("应用内容")).toBeInTheDocument();
  });

  test("新壳不提供顶部工具栏搜索/AI 按钮入口", () => {
    render(<NewShell><div>占位</div></NewShell>);
    expect(screen.queryByRole("button", { name: "打开全局搜索" })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "打开 AI 助手" })).not.toBeInTheDocument();
  });
});

describe("壳层滚动职责与留白节奏", () => {
  test("外层壳 h-[100dvh] 且窗口外不滚动，留白遵循 p-3 gap-3、桌面左侧贴边", () => {
    expect(APP_SHELL_CLASS).toContain("app-shell");
    expect(APP_SHELL_CLASS).toContain("h-[100dvh]");
    expect(APP_SHELL_CLASS).toContain("overflow-hidden");
    expect(APP_SHELL_CLASS).toContain("gap-3");
    expect(APP_SHELL_CLASS).toContain("p-3");
    // 桌面端取消左内边距，使固定侧栏直角贴齐视口左边
    expect(APP_SHELL_CLASS).toContain("md:pl-0");
  });

  test("业务内容仅在主容器内部滚动层滚动，外层容器自身不滚动", () => {
    expect(MAIN_SCROLL_CLASS).toContain("overflow-y-auto");
    expect(MAIN_SCROLL_CLASS).toContain("min-h-0");
    expect(MAIN_SURFACE_CLASS).toContain("overflow-hidden");
    expect(MAIN_SURFACE_CLASS).not.toContain("overflow-y-auto");
  });

  test("壳层底层与侧栏统一 #F9F9FB(bg-sidebar)：消除色差竖带，--muted 保留供表头", () => {
    // 壳层不再用 bg-muted：与侧栏(#F9F9FB)的色差会被读作"竖线分隔"（用户截图反馈）
    expect(APP_SHELL_CLASS).toContain("bg-sidebar");
    expect(APP_SHELL_CLASS).not.toContain("bg-muted");
    // --muted 仍为 #F0F0F3，供表头 thead 等主题化场景；深色主题见"深色主题配色"用例
    const css = readFileSync(join(process.cwd(), "app", "globals.css"), "utf-8");
    const rootBlock = css.match(/:root\s*\{[\s\S]*?\n\}/)?.[0] ?? "";
    expect(rootBlock).toContain("--muted: #F0F0F3");
  });

  test("侧栏宽度契约在 wrapper 层统一定义：占位层与 fixed 可见层同宽，防止左缘圆角被遮", () => {
    expect(APP_SIDEBAR_WIDTH_VARS["--sidebar-width"]).toBe("15rem");
    expect(APP_SIDEBAR_WIDTH_VARS["--sidebar-width-icon"]).toBe("4rem");
  });
});

describe("右侧纯白 #FFFFFF 圆角内容容器契约", () => {
  test("AppMainContainer 渲染为纯白固定圆角裁剪容器并承载滚动层", () => {
    const { container } = render(
      <AppMainContainer>
        <div data-slot="app-main-content" className={MAIN_SCROLL_CLASS}>业务内容</div>
      </AppMainContainer>,
    );

    const surface = container.querySelector(`[data-slot="${APP_MAIN_CONTAINER_SLOT}"]`);
    expect(surface).not.toBeNull();
    expect(surface).toHaveClass("bg-background"); // --background 浅色主题下即纯白 #FFFFFF
    expect(surface).toHaveClass("rounded-2xl");
    expect(surface).toHaveClass("overflow-hidden");
    // 主容器描边 + 多层柔影：大模糊低透明度环境影 + 近距接触影（--main-surface-shadow 主题变量）
    expect(surface).toHaveClass("border");
    expect(surface).toHaveClass("border-border/70");
    expect(surface).toHaveClass("shadow-[var(--main-surface-shadow)]");
    expect(surface).not.toHaveClass("shadow-sm");

    const scrollLayer = container.querySelector('[data-slot="app-main-content"]');
    expect(scrollLayer).toHaveClass("overflow-y-auto");
    expect(scrollLayer?.parentElement).toBe(surface);
    expect(screen.getByText("业务内容")).toBeInTheDocument();
  });

  test("主容器多层柔影与控制坞柔影由主题变量定义，深浅主题各自协调", () => {
    const css = readFileSync(join(process.cwd(), "app", "globals.css"), "utf-8");
    const rootBlock = css.match(/:root\s*\{[\s\S]*?\n\}/)?.[0] ?? "";
    const darkBlock = css.match(/\.dark\s*\{[\s\S]*?\n\}/)?.[0] ?? "";
    // 浅色：多层柔影（环境影 + 接触影）；深色：纯黑高透明度同节奏
    expect(rootBlock).toContain("--main-surface-shadow:");
    expect(darkBlock).toContain("--main-surface-shadow:");
    // 控制坞柔影：小扩散，避免矩形阴影在主容器圆角处被硬裁成方形暗块
    expect(rootBlock).toContain("--dock-shadow:");
    expect(darkBlock).toContain("--dock-shadow:");
    // dock 不再使用大扩散 shadow-lg（左下角方形阴影残留根因）
    const bar = readFileSync(join(process.cwd(), "components", "layout", "management-bar.tsx"), "utf-8");
    expect(bar).toContain("shadow-[var(--dock-shadow)]");
    expect(bar).not.toContain("shadow-lg");
  });

  test("主内容滚动层使用细滚动条：thumb 用 --border 系、track 透明，不影响其他滚动区", () => {
    const css = readFileSync(join(process.cwd(), "app", "globals.css"), "utf-8");
    // Firefox：thin + thumb 色 --border / track 透明
    expect(css).toMatch(/\[data-slot="app-main-content"\]\s*\{[^}]*scrollbar-width:\s*thin[^}]*\}/);
    expect(css).toMatch(/\[data-slot="app-main-content"\]\s*\{[^}]*scrollbar-color:\s*var\(--border\)\s+transparent[^}]*\}/);
    // WebKit：6px 轨道、胶囊圆角 thumb、track 透明
    expect(css).toMatch(/\[data-slot="app-main-content"\]::-webkit-scrollbar\s*\{[^}]*width:\s*6px/);
    expect(css).toMatch(/\[data-slot="app-main-content"\]::-webkit-scrollbar-thumb\s*\{[^}]*border-radius:\s*9999px[^}]*background-color:\s*var\(--border\)[^}]*\}/);
    expect(css).toMatch(/\[data-slot="app-main-content"\]::-webkit-scrollbar-track\s*\{[^}]*transparent/);
  });

  test("控制坞相对主容器左下角 16px 定位且位于主容器内部", () => {
    expect(DOCK_POSITION_CLASS).toContain("absolute");
    expect(DOCK_POSITION_CLASS).toContain("bottom-4");
    expect(DOCK_POSITION_CLASS).toContain("left-4");
    expect(DOCK_POSITION_CLASS).not.toContain("fixed");
  });
});

describe("深色主题配色契约（仅 .dark 区块，浅色一字不动）", () => {
  const readBlocks = () => {
    const css = readFileSync(join(process.cwd(), "app", "globals.css"), "utf-8");
    return {
      root: css.match(/:root\s*\{[\s\S]*?\n\}/)?.[0] ?? "",
      dark: css.match(/\.dark\s*\{[\s\S]*?\n\}/)?.[0] ?? "",
    };
  };

  test("深色基线：#101012 底 / #19191B 侧栏 / #222225 选中 / #444446 悬停 / #101012 弹层", () => {
    const { dark } = readBlocks();
    expect(dark).toContain("--background: #101012");
    expect(dark).toContain("--popover: #101012");
    expect(dark).toContain("--sidebar: #19191B");
    expect(dark).toContain("--muted: #222225"); // 菜单选中态 bg-muted 自动命中
    expect(dark).toContain("--sidebar-accent: #444446"); // 菜单悬停态自动命中
  });

  test("深色层级和谐：卡片略亮于底色形成分层，描边介于选中与悬停之间", () => {
    const { dark } = readBlocks();
    expect(dark).toContain("--card: #18181B");
    expect(dark).toContain("--border: #303034");
  });

  test("浅色主题值保持不动（#F9F9FB 侧栏 / #F0F0F3 muted / #CDCDCF 悬停）", () => {
    const { root } = readBlocks();
    expect(root).toContain("--sidebar: #F9F9FB");
    expect(root).toContain("--muted: #F0F0F3");
    expect(root).toContain("--sidebar-accent: #CDCDCF");
  });
});

describe("控制坞入口（新壳 dock）", () => {
  beforeEach(() => vi.clearAllMocks());

  test("dock 提供全局搜索入口并派发 dock:open-search，不再包含 AI 助手", () => {
    const handler = vi.fn();
    window.addEventListener("dock:open-search", handler);
    try {
      render(<ManagementBar variant="new" />);
      // AI 助手已从 dock 移除（浮动聊天面板不再有 dock 入口属预期）
      expect(screen.queryByRole("button", { name: "AI 助手" })).not.toBeInTheDocument();
      fireEvent.click(screen.getByRole("button", { name: "全局搜索" }));
      expect(handler).toHaveBeenCalledTimes(1);
    } finally {
      window.removeEventListener("dock:open-search", handler);
    }
  });

  test("既有 dock 常驻入口保留：侧边栏/主页/主题切换/通知中心/QQ群", () => {
    render(<ManagementBar variant="new" />);
    for (const name of ["侧边栏", "主页", "通知中心", "QQ群"]) {
      expect(screen.getByRole("button", { name })).toBeInTheDocument();
    }
  });
});
