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
 * 权限判断 Hook，封装当前用户信息与权限判断调用（P7.1 升级版）。
 *
 * 优先级（Q3=A）：
 *   1. 优先检查 user.permissions 扁平数组（后端实时权限）
 *   2. 若 permissions 为空，回退到角色矩阵 canPerform（兼容旧版）
 *
 * 用法：
 *   const { can, role, departmentId } = usePermissions();
 *   if (can("employee", "delete")) { ... }
 */
export function usePermissions() {
  const { user, hasPermission } = useAuth();
  const role: Role = normalizeRole(user?.role ?? "viewer");
  const departmentId: number | null = user?.department_id ?? null;

  const can = (
    resource: ResourceType,
    action: ActionType,
    ctx?: PermissionCheckContext,
  ): boolean => {
    // P7.1: 优先使用扁平权限数组
    if (user?.permissions && user.permissions.length > 0) {
      return hasPermission(resource, action);
    }

    // 回退：使用角色矩阵（向后兼容无 permissions 字段的旧版响应）
    return canPerform(role, resource, action, {
      role,
      departmentId,
      ...ctx,
    });
  };

  return { can, role, departmentId };
}
