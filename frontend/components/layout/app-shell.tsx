import type { ReactNode } from "react";

/**
 * 新壳壳层契约模块：从 app/page.tsx 拆出，避免页面模块导出非约定成员
 * （Next.js 页面仅允许 default/metadata 等导出，其余命名导出会触发 .next/types 校验失败）。
 */

/** 壳层留白契约：外层不滚动，四周 p-3、侧栏与主容器之间 gap-3；桌面左侧不留白使侧栏贴齐视口左边。
 * 底层背景与侧栏统一用 bg-sidebar(#F9F9FB)：侧栏位于底层、白色主容器浮于其上，
 * 消除 bg-muted(#F0F0F3) 与侧栏色差形成的"竖线分隔感"（用户截图反馈）；
 * --muted 保留 #F0F0F3 仅供表头 thead 等主题化场景使用。
 * overflow-hidden 裁切边界=视口边缘（无 border），兜底防页面级滚动；
 * 主容器柔影的实际裁切元凶是 SidebarInset 基类（同尺寸 overflow-hidden），
 * 已在 globals.css 末尾按 .app-shell 作用域放开 */
export const APP_SHELL_CLASS = "app-shell flex h-[100dvh] min-h-0 overflow-hidden bg-sidebar gap-3 p-3 md:pl-0";

/**
 * 新壳侧栏宽度契约（wrapper 层统一定义）：占位层(sidebar-gap)与 fixed 可见层(sidebar-container)
 * 必须同宽，否则 fixed 层右溢会压住主容器左缘、遮盖左缘上下圆角（P15 视觉修复）。
 */
export const APP_SIDEBAR_WIDTH_VARS = { "--sidebar-width": "15rem", "--sidebar-width-icon": "4rem" } as const;

/** 主容器内滚动层契约：业务内容仅在白色圆角主容器内部的这一层滚动 */
export const MAIN_SCROLL_CLASS = "min-h-0 flex-1 overflow-y-auto p-4 md:p-6";

/** 右侧纯白 #FFFFFF（--background）固定圆角内容容器：窗口级唯一可见业务区域；
 * 细描边(border-border/70) + 三层 0 扩展柔影(--main-surface-shadow，globals.css 按主题定义)：
 * 0 扩展(spread)确保阴影轮廓与容器圆角重合，不在圆角角隙沉积方形楔角暗块；
 * 阴影向下延伸经 .app-shell 内放开的 SidebarInset 裁切（globals.css 末尾规则），
 * 在壳层 12px 留白内自然衰减，四角圆弧干净无方形残留 */
export const MAIN_SURFACE_CLASS = "relative flex min-h-0 flex-1 flex-col overflow-hidden rounded-2xl border border-border/70 bg-background shadow-[var(--main-surface-shadow)]";

/** 主容器的 data-slot 标记值，供壳层契约测试与视觉定位使用 */
export const APP_MAIN_CONTAINER_SLOT = "app-main-container";

/**
 * 新壳右侧纯白圆角主容器：控制坞、聊天面板与业务滚动层都挂在它内部，
 * 保证 dock 相对主容器左下角定位且业务内容只在容器内滚动。
 */
export function AppMainContainer({ children }: { children: ReactNode }) {
  return (
    <div data-slot={APP_MAIN_CONTAINER_SLOT} className={MAIN_SURFACE_CLASS}>
      {children}
    </div>
  );
}

/**
 * P13 新壳标记节点：右上角固定通知按钮已移除，通知入口统一收敛至控制坞；
 * display:contents 保证本节点不影响任何布局。
 */
export function NewShell({ children }: { children: ReactNode }) {
  return <div data-shell="new" className="contents">{children}</div>;
}
