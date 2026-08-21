import { createElement, type ComponentType, type ReactNode } from "react";
import { describe, expect, test } from "vitest";
import { render, screen } from "@testing-library/react";

import {
  DEFAULT_VIEW,
  FALLBACK_VIEW,
  VIEW_IDS,
  isViewId,
  renderView,
  resolveViewId,
  type ViewComponentMap,
  type ViewRenderContext,
} from "@/lib/view-mapping";

/** 每个视图的桩组件：渲染带视图标记的节点，并透出收到的上下文 props 便于断言 */
function makeStub(id: string): ComponentType {
  return function Stub({ userName, onBack, initialTab, defaultModule }: { userName?: string | null; onBack?: () => void; initialTab?: string; defaultModule?: string | null }): ReactNode {
    return createElement("div", {
      "data-testid": `view-${id}`,
      "data-user-name": userName ?? "",
      "data-has-on-back": onBack ? "yes" : "no",
      "data-initial-tab": initialTab ?? "",
      "data-default-module": defaultModule ?? "",
    });
  };
}

/** 覆盖全部 17 个合法视图的桩映射表 */
const stubMap: ViewComponentMap = {
  landing: makeStub("landing"),
  employee: makeStub("employee"),
  "employee-provident": makeStub("employee-provident"),
  onboarding: makeStub("onboarding"),
  resignation: makeStub("resignation"),
  regularization: makeStub("regularization"),
  "labor-contracts": makeStub("labor-contracts"),
  rewards: makeStub("rewards"),
  "personnel-changes": makeStub("personnel-changes"),
  training: makeStub("training"),
  "admin-contracts": makeStub("admin-contracts"),
  safety: makeStub("safety"),
  "occupational-health": makeStub("occupational-health"),
  insurance: makeStub("insurance"),
  dormitory: makeStub("dormitory"),
  energy: makeStub("energy"),
  organization: makeStub("organization"),
  audit: makeStub("audit"),
  monitoring: makeStub("monitoring"),
  "daily-affairs": makeStub("daily-affairs"),
  "daily-affairs-archives": makeStub("daily-affairs-archives"),
  "daily-affairs-office-supplies": makeStub("daily-affairs-office-supplies"),
  "daily-affairs-canteen": makeStub("daily-affairs-canteen"),
  "daily-affairs-invoice": makeStub("daily-affairs-invoice"),
  "fleet-vehicles": makeStub("fleet-vehicles"),
  system: makeStub("system"),
  "personal-settings": makeStub("personal-settings"),
  feedback: makeStub("feedback"),
  departments: makeStub("departments"),
  knowledge: makeStub("knowledge"),
};

const ctx: ViewRenderContext = { userName: "张三", onBackFromSettings: () => {} };

describe("视图映射常量", () => {
  test("VIEW_IDS 覆盖全部 17 个合法视图分支，顺序与 page.tsx 既有装配一致", () => {
    expect(VIEW_IDS).toEqual([
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
    ]);
  });

  test("默认视图为工作台 landing，非法回退视图为 insurance", () => {
    expect(DEFAULT_VIEW).toBe("landing");
    expect(FALLBACK_VIEW).toBe("insurance");
  });
});

describe("isViewId / resolveViewId", () => {
  test("isViewId 对全部 17 个合法视图返回 true", () => {
    for (const view of VIEW_IDS) {
      expect(isViewId(view)).toBe(true);
    }
  });

  test("isViewId 对非法输入返回 false", () => {
    for (const bad of [undefined, null, "", "unknown", "settings", "home", 42, {}, []]) {
      expect(isViewId(bad)).toBe(false);
    }
  });

  test("resolveViewId 对合法视图原样返回", () => {
    for (const view of VIEW_IDS) {
      expect(resolveViewId(view)).toBe(view);
    }
  });

  test("resolveViewId 对非法输入一律回退 insurance", () => {
    expect(resolveViewId("unknown")).toBe("insurance");
    expect(resolveViewId("")).toBe("insurance");
    expect(resolveViewId(undefined)).toBe("insurance");
    expect(resolveViewId(null)).toBe("insurance");
  });
});

describe("renderView 装配策略", () => {
  test("正常全覆盖：13 个合法视图各自渲染对应组件", () => {
    for (const view of VIEW_IDS) {
      render(<>{renderView(view, stubMap, ctx)}</>);
      expect(screen.getByTestId(`view-${view}`)).toBeInTheDocument();
    }
  });

  test("非法视图回退渲染 insurance 组件，不渲染其他组件", () => {
    render(<>{renderView("unknown", stubMap, ctx)}</>);
    expect(screen.getByTestId("view-insurance")).toBeInTheDocument();
    expect(screen.queryByTestId("view-landing")).not.toBeInTheDocument();
  });

  test("landing 视图注入 userName，且不注入返回回调", () => {
    render(<>{renderView("landing", stubMap, ctx)}</>);
    expect(screen.getByTestId("view-landing")).toHaveAttribute("data-user-name", "张三");
    expect(screen.getByTestId("view-landing")).toHaveAttribute("data-has-on-back", "no");
  });

  test("员工公积金与日常事务精确入口注入最小装配参数", () => {
    render(<>{renderView("employee-provident", stubMap, ctx)}</>);
    expect(screen.getByTestId("view-employee-provident")).toHaveAttribute("data-initial-tab", "provident");

    render(<>{renderView("daily-affairs-invoice", stubMap, ctx)}</>);
    expect(screen.getByTestId("view-daily-affairs-invoice")).toHaveAttribute("data-default-module", "invoice");
  });

  test("system / personal-settings 视图注入返回回调", () => {
    const onBackFromSettings = () => {};
    render(<>{renderView("system", stubMap, { onBackFromSettings })}</>);
    expect(screen.getByTestId("view-system")).toHaveAttribute("data-has-on-back", "yes");
    render(<>{renderView("personal-settings", stubMap, { onBackFromSettings })}</>);
    expect(screen.getByTestId("view-personal-settings")).toHaveAttribute("data-has-on-back", "yes");
  });

  test("system 视图仅注入可选的系统设置内部面板", () => {
    render(<>{renderView("system", stubMap, { systemSettingsPanel: "audit" })}</>);
    expect(screen.getByTestId("view-system")).toBeInTheDocument();
  });

  test("其余视图不泄漏上下文 props（无 userName / onBack）", () => {
    render(<>{renderView("employee", stubMap, ctx)}</>);
    expect(screen.getByTestId("view-employee")).toHaveAttribute("data-user-name", "");
    expect(screen.getByTestId("view-employee")).toHaveAttribute("data-has-on-back", "no");
  });

  test("缺省 ctx 时安全渲染，landing 的用户名为空", () => {
    render(<>{renderView("landing", stubMap)}</>);
    expect(screen.getByTestId("view-landing")).toHaveAttribute("data-user-name", "");
  });
});
