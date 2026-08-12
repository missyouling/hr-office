"use client";

import React from "react";
import { useAuth } from "@/lib/auth";
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
  TooltipProvider,
} from "@/components/ui/tooltip";

// ========== RequirePermission 组件类型定义 ==========

export interface RequirePermissionProps {
  /** 目标资源名，如 "employee" */
  resource: string;
  /** 目标操作名，如 "delete" */
  action: string;
  /**
   * 无权限时的行为模式：
   * - "hide": 不渲染 children（默认）
   * - "disable": 渲染 children 但禁用交互，并显示 Tooltip 提示
   */
  mode?: "hide" | "disable";
  /** 需要保护的内容 */
  children: React.ReactNode;
  /**
   * hide 模式下无权限时展示的替代内容。
   * 仅在 mode="hide" 时生效，默认返回 null。
   */
  fallback?: React.ReactNode;
}

// ========== 禁用模式子组件 ==========

interface DisableWrapperProps {
  children: React.ReactNode;
}

/**
 * 禁用包装组件。
 * 使用 cloneElement 给子元素注入 disabled 属性和空 onClick，
 * 避免与 Button 或其他组件强耦合。
 */
function DisableWrapper({ children }: DisableWrapperProps) {
  // 仅处理单个 React 元素的场景（多子元素 = Fragment，直接包裹）
  const child = React.Children.only(children) as React.ReactElement;

  // cloneElement 注入禁用属性
  //  注意：React 19 对 cloneElement 的类型约束更严格，
  //  使用 Record 类型绕过对特定组件的依赖
  const disabledChild = React.cloneElement(child, {
    disabled: true,
    "aria-disabled": true,
    onClick: (e: React.MouseEvent) => {
      e.preventDefault();
      e.stopPropagation();
    },
    style: {
      ...(child.props as Record<string, unknown>).style as Record<string, unknown> || {},
      cursor: "not-allowed",
      pointerEvents: "auto" as React.CSSProperties["pointerEvents"],
    },
  } as Record<string, unknown>);

  return (
    <TooltipProvider>
      <Tooltip>
        <TooltipTrigger asChild>
          <span style={{ display: "inline-block", cursor: "not-allowed" }}>
            {disabledChild}
          </span>
        </TooltipTrigger>
        <TooltipContent>
          <p>您没有此权限</p>
        </TooltipContent>
      </Tooltip>
    </TooltipProvider>
  );
}

// ========== RequirePermission 主组件 ==========

/**
 * 权限门控组件（P7.1）。
 * 根据当前用户的扁平权限数组决定是否渲染子内容。
 *
 * 使用示例：
 *   // hide 模式（默认）
 *   <RequirePermission resource="employee" action="delete">
 *     <Button>删除</Button>
 *   </RequirePermission>
 *
 *   // disable 模式
 *   <RequirePermission resource="employee" action="delete" mode="disable">
 *     <Button>删除</Button>
 *   </RequirePermission>
 *
 *   // hide 模式 + 自定义 fallback
 *   <RequirePermission resource="employee" action="create" fallback={<span>无权限</span>}>
 *     <Button>创建</Button>
 *   </RequirePermission>
 */
export function RequirePermission({
  resource,
  action,
  mode = "hide",
  children,
  fallback = null,
}: RequirePermissionProps) {
  const { hasPermission } = useAuth();
  const allowed = hasPermission(resource, action);

  // 有权限：直接渲染
  if (allowed) {
    return <>{children}</>;
  }

  // 无权限 + hide 模式
  if (mode === "hide") {
    return <>{fallback}</>;
  }

  // 无权限 + disable 模式
  return <DisableWrapper>{children}</DisableWrapper>;
}