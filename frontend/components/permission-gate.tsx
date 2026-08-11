"use client";

import { usePermissions } from "@/hooks/use-permissions";
import type { ResourceType, ActionType } from "@/lib/permissions";

export interface PermissionGateProps {
  /** 目标资源 */
  resource: ResourceType;
  /** 目标操作 */
  action: ActionType;
  /** 资源所属部门 ID（用于跨部门访问判断） */
  resourceDepartmentId?: number | null;
  /** 是否为当前用户自己的资源 */
  isOwnResource?: boolean;
  /** 无权限时显示的替代内容，默认隐藏（null） */
  fallback?: React.ReactNode;
  children: React.ReactNode;
}

/**
 * 按钮级权限门控组件。
 * 根据当前用户角色和部门上下文自动显示/隐藏子内容。
 *
 * 用法：
 *   <PermissionGate resource="employee" action="delete">
 *     <Button>删除</Button>
 *   </PermissionGate>
 */
export function PermissionGate({
  resource,
  action,
  resourceDepartmentId,
  isOwnResource,
  fallback = null,
  children,
}: PermissionGateProps) {
  const { can } = usePermissions();
  const allowed = can(resource, action, {
    resourceDepartmentId,
    isOwnResource,
  });

  return allowed ? <>{children}</> : <>{fallback}</>;
}
