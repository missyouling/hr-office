"use client";

// ========== 角色与权限类型定义 ==========

/** 系统角色 */
export type Role = "admin" | "manager" | "editor" | "viewer" | "super_admin";

/** 资源类型 */
export type ResourceType =
  | "employee"
  | "insurance"
  | "dormitory"
  | "archives"
  | "announcement"
  | "user"
  | "department"
  | "office-supply"
  | "canteen"
  | "invoice"
  | "knowledge_base";

/** 操作类型 */
export type ActionType = "view" | "create" | "edit" | "delete" | "approve" | "submit" | "reject";

/** 权限判断上下文（用于跨部门访问控制） */
export interface PermissionContext {
  role: Role;
  departmentId?: number | null;
  resourceDepartmentId?: number | null;
  isOwnResource?: boolean;
}

// ========== 角色层级（数值越大权限越高）==========

export const ROLE_HIERARCHY: Record<Role, number> = {
  super_admin: 4, // 兼容映射，等同于 admin
  admin: 4,
  manager: 3,
  editor: 2,
  viewer: 1,
};

// ========== 权限矩阵 ==========
// 定义每个角色在各资源上允许的操作列表

const FULL: ActionType[] = ["view", "create", "edit", "delete"];
const VIEW_EDIT: ActionType[] = ["view", "edit"];
const VIEW_ONLY: ActionType[] = ["view"];

const PERMISSION_MATRIX: Record<Role, Partial<Record<ResourceType, ActionType[]>>> = {
  super_admin: {
    employee: FULL,
    insurance: FULL,
    dormitory: FULL,
    archives: FULL,
    announcement: FULL,
    user: FULL,
    department: FULL,
    "office-supply": FULL,
    canteen: FULL,
    invoice: FULL,
    knowledge_base: FULL,
  },
  admin: {
    employee: FULL,
    insurance: FULL,
    dormitory: FULL,
    archives: FULL,
    announcement: FULL,
    user: FULL,
    department: FULL,
    "office-supply": FULL,
    canteen: FULL,
    invoice: FULL,
    knowledge_base: FULL,
  },
  manager: {
    employee: VIEW_EDIT,
    insurance: VIEW_EDIT,
    dormitory: VIEW_EDIT,
    archives: VIEW_EDIT,
    announcement: ["view", "create", "edit"],
    // manager 不能管理用户和部门
    "office-supply": VIEW_EDIT,
    canteen: VIEW_EDIT,
    invoice: ["view", "create", "edit", "delete", "submit", "approve", "reject"],
    knowledge_base: VIEW_EDIT,
  },
  editor: {
    employee: VIEW_EDIT,
    insurance: VIEW_EDIT,
    dormitory: VIEW_EDIT,
    archives: VIEW_EDIT,
    announcement: VIEW_ONLY,
    "office-supply": VIEW_EDIT,
    canteen: VIEW_EDIT,
    invoice: ["view", "create", "edit", "delete", "submit"],
    knowledge_base: VIEW_EDIT,
  },
  viewer: {
    employee: VIEW_ONLY,
    insurance: VIEW_ONLY,
    dormitory: VIEW_ONLY,
    archives: VIEW_ONLY,
    announcement: VIEW_ONLY,
    "office-supply": VIEW_ONLY,
    canteen: VIEW_ONLY,
    invoice: VIEW_ONLY,
    knowledge_base: VIEW_ONLY,
  },
};

// ========== 角色归一化 ==========

/** 将后端角色字符串映射为标准 Role 类型，兼容旧角色名 */
export function normalizeRole(role: string): Role {
  switch (role) {
    case "super_admin":
      return "super_admin";
    case "admin":
      return "admin";
    case "manager":
      return "manager";
    case "editor":
      return "editor";
    case "viewer":
      return "viewer";
    // 兼容旧角色名
    case "user":
      return "viewer";
    default:
      return "viewer"; // 安全兜底：未知角色按最低权限处理
  }
}

// ========== 权限判断函数 ==========

/**
 * 判断指定角色是否有权对某资源执行某操作。
 *
 * @param role          - 当前用户角色
 * @param resource      - 目标资源
 * @param action        - 目标操作
 * @param ctx           - 可选的部门上下文（用于跨部门判断）
 * @returns 是否允许执行
 */
export function canPerform(
  role: Role,
  resource: ResourceType,
  action: ActionType,
  ctx?: PermissionContext,
): boolean {
  const actions = PERMISSION_MATRIX[role]?.[resource];
  if (!actions || !actions.includes(action)) {
    return false;
  }

  // 跨部门访问控制：admin/super_admin 可跨部门，其他角色仅限本部门
  if (ctx && isCrossDepartment(ctx)) {
    const hierarchy = ROLE_HIERARCHY[ctx.role] ?? 0;
    if (hierarchy < ROLE_HIERARCHY.admin) {
      return false;
    }
  }

  return true;
}

/**
 * 判断当前上下文是否涉及跨部门访问。
 * 规则：部门 ID 都存在且不相等时视为跨部门。
 */
export function isCrossDepartment(ctx: PermissionContext): boolean {
  if (ctx.departmentId == null || ctx.resourceDepartmentId == null) {
    // 任一为空时不限制（向后兼容无部门数据的资源）
    return false;
  }
  return ctx.departmentId !== ctx.resourceDepartmentId;
}
