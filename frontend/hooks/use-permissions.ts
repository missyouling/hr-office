"use client";

import { useAuth } from "@/lib/supabase/auth-context";
import {
  canPerform,
  normalizeRole,
  type Role,
  type ResourceType,
  type ActionType,
} from "@/lib/permissions";

/** 权限上下文简化版（被 Hook 自动注入 role 和 departmentId） */
export interface PermissionCheckContext {
  resourceDepartmentId?: number | null;
  isOwnResource?: boolean;
}

/**
 * 权限判断 Hook，封装当前用户信息与 canPerform 调用。
 *
 * 用法：
 *   const { can, role, departmentId } = usePermissions();
 *   if (can("employee", "delete")) { ... }
 */
export function usePermissions() {
  const { user } = useAuth();
  const role: Role = normalizeRole(user?.role ?? "viewer");
  const departmentId: number | null = user?.department_id ?? null;

  const can = (
    resource: ResourceType,
    action: ActionType,
    ctx?: PermissionCheckContext,
  ): boolean => {
    return canPerform(role, resource, action, {
      role,
      departmentId,
      ...ctx,
    });
  };

  return { can, role, departmentId };
}
