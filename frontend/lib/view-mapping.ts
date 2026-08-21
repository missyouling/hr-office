/**
 * 统一视图映射模块（P12.1.4）
 *
 * 依据 docs/frontend-ui-migration-matrix.md（P12.0.1 定稿的视图映射表）落地：
 * - 合法视图 ID（ViewId）与 page.tsx 的 renderMainContent 分支一一对应；
 * - 默认视图为工作台 landing；非法/未知视图安全回退 insurance（与既有 default 分支一致）；
 * - 装配策略：由页面装配点提供「视图 → 既有 React 组件」映射表（类型强制覆盖全部 ViewId），
 *   本模块负责解析视图 ID、按视图注入上下文 props 并渲染，page.tsx 不再维护 switch。
 */
import { createElement, type ComponentType, type ReactNode } from "react";

/** 全部合法视图 ID，顺序与 page.tsx 装配保持一致。 */
export const VIEW_IDS = [
  "landing",
  "employee",
  "employee-provident",
  "onboarding",
  "resignation",
  "regularization",
  "labor-contracts",
  "rewards",
  "personnel-changes",
  "training",
  "admin-contracts",
  "safety",
  "occupational-health",
  "insurance",
  "dormitory",
  "energy",
  "organization",
  "audit",
  "monitoring",
  "daily-affairs",
  "daily-affairs-archives",
  "daily-affairs-office-supplies",
  "daily-affairs-canteen",
  "daily-affairs-invoice",
  "fleet-vehicles",
  "system",
  "personal-settings",
  "feedback",
  "departments",
  "knowledge",
] as const;

/** 合法视图 ID 联合类型 */
export type ViewId = (typeof VIEW_IDS)[number];

/** 默认视图：应用初始进入的工作台 */
export const DEFAULT_VIEW: ViewId = "landing";

/** 非法/未知视图的回退视图：沿用既有 default 分支的 InsuranceManagement（迁移矩阵未规定不同） */
export const FALLBACK_VIEW: ViewId = "insurance";

/** 类型守卫：判断任意输入是否为合法 ViewId */
export function isViewId(value: unknown): value is ViewId {
  return typeof value === "string" && (VIEW_IDS as readonly string[]).includes(value);
}

/** 解析任意输入为合法 ViewId，非法值一律回退 FALLBACK_VIEW */
export function resolveViewId(value: unknown): ViewId {
  return isViewId(value) ? value : FALLBACK_VIEW;
}

/** 视图装配上下文：页面装配点注入给个别视图的 props */
export interface ViewRenderContext {
  /** landing 工作台问候用户名 */
  userName?: string | null;
  /** system / personal-settings 的“返回”按钮回调 */
  onBackFromSettings?: () => void;
  /** 系统设置内部初始面板；默认不指定，保持旧行为 */
  systemSettingsPanel?: "audit" | "monitoring";
}

/** 视图 → 既有 React 组件的映射表（Record 类型强制要求覆盖全部 ViewId，缺任一视图编译失败） */
export type ViewComponentMap = Record<ViewId, ComponentType>;

/**
 * 装配策略：解析视图 ID 后按视图注入上下文 props 并渲染既有组件。
 * - landing：注入 userName；
 * - employee-provident：注入公积金页默认标签；
 * - daily-affairs-*：注入日常事务页默认模块；
 * - system / personal-settings：注入 onBack（设置返回来源回调）；
 * - 其余视图：无 props 渲染。
 * 组件运行时会忽略多余 props、缺省 props 取 undefined，与既有 switch 各分支行为一致。
 */
export function renderView(view: string, components: ViewComponentMap, ctx: ViewRenderContext = {}): ReactNode {
  const viewId = resolveViewId(view);
  const Component = components[viewId];
  // 按视图注入上下文 props；组件运行时会忽略多余 props、缺省 props 取 undefined，与既有 switch 各分支行为一致
  if (viewId === "landing") {
    return createElement(Component as ComponentType<{ userName?: string | null }>, { userName: ctx.userName });
  }
  if (viewId === "employee-provident") {
    return createElement(Component as ComponentType<{ initialTab?: "active" | "resigned" | "insurance-increase" | "insurance-decrease" | "provident" }>, { initialTab: "provident" });
  }
  if (viewId === "onboarding" || viewId === "resignation" || viewId === "regularization" || viewId === "labor-contracts" || viewId === "rewards" || viewId === "personnel-changes" || viewId === "training" || viewId === "admin-contracts" || viewId === "safety" || viewId === "occupational-health") {
    return createElement(Component);
  }
  if (viewId === "daily-affairs-archives") {
    return createElement(Component as ComponentType<{ defaultModule?: string | null }>, { defaultModule: "archives" });
  }
  if (viewId === "daily-affairs-office-supplies") {
    return createElement(Component as ComponentType<{ defaultModule?: string | null }>, { defaultModule: "office-supplies" });
  }
  if (viewId === "daily-affairs-canteen") {
    return createElement(Component as ComponentType<{ defaultModule?: string | null }>, { defaultModule: "canteen" });
  }
  if (viewId === "daily-affairs-invoice") {
    return createElement(Component as ComponentType<{ defaultModule?: string | null }>, { defaultModule: "invoice" });
  }
  if (viewId === "system") {
    return createElement(Component as ComponentType<{ onBack?: () => void; initialPanel?: "audit" | "monitoring" }>, {
      onBack: ctx.onBackFromSettings,
      initialPanel: ctx.systemSettingsPanel,
    });
  }
  if (viewId === "personal-settings") {
    return createElement(Component as ComponentType<{ onBack?: () => void }>, { onBack: ctx.onBackFromSettings });
  }
  return createElement(Component);
}
