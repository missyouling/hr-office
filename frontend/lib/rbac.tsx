"use client";

import { useAuth } from "@/lib/auth";
import { Button } from "@/components/ui/button";

type Permission = "view" | "create" | "edit" | "delete";

type Module = 
  | "employee" 
  | "insurance" 
  | "dormitory" 
  | "archives" 
  | "settings" 
  | "announcements" 
  | "backups" 
  | "users";

const ROLE_PERMISSIONS: Record<string, Partial<Record<Module, Permission[]>>> = {
  user: {
    employee: ["view"],
    insurance: ["view"],
    dormitory: ["view"],
    archives: ["view", "create"],
    announcements: ["view"],
  },
  admin: {
    employee: ["view", "create", "edit", "delete"],
    insurance: ["view", "create", "edit"],
    dormitory: ["view", "create", "edit", "delete"],
    archives: ["view", "create", "edit", "delete"],
    settings: ["view", "edit"],
    announcements: ["view", "create", "edit"],
    backups: ["view", "create"],
    users: ["view"],
  },
  super_admin: {
    employee: ["view", "create", "edit", "delete"],
    insurance: ["view", "create", "edit", "delete"],
    dormitory: ["view", "create", "edit", "delete"],
    archives: ["view", "create", "edit", "delete"],
    settings: ["view", "create", "edit", "delete"],
    announcements: ["view", "create", "edit", "delete"],
    backups: ["view", "create", "edit", "delete"],
    users: ["view", "create", "edit", "delete"],
  },
};

export function usePermission() {
  const { user } = useAuth();
  
  const role = user?.role || "user";
  const permissions = ROLE_PERMISSIONS[role] || ROLE_PERMISSIONS.user;

  const can = (module: Module, action: Permission): boolean => {
    const modulePerms = permissions[module];
    if (!modulePerms) return false;
    return modulePerms.includes(action);
  };

  const canAny = (module: Module, actions: Permission[]): boolean => {
    return actions.some((action) => can(module, action));
  };

  const canAll = (module: Module, actions: Permission[]): boolean => {
    return actions.every((action) => can(module, action));
  };

  return {
    can,
    canAny,
    canAll,
    role,
    isAdmin: role === "admin" || role === "super_admin",
    isSuperAdmin: role === "super_admin",
  };
}

interface PermissionGuardProps {
  module: Module;
  action: Permission;
  children: React.ReactNode;
  fallback?: React.ReactNode;
}

export function PermissionGuard({ module, action, children, fallback = null }: PermissionGuardProps) {
  const { can } = usePermission();
  
  if (!can(module, action)) {
    return <>{fallback}</>;
  }
  
  return <>{children}</>;
}

interface PermissionButtonProps extends PermissionGuardProps {
  onClick?: () => void;
  disabled?: boolean;
  variant?: "default" | "destructive" | "outline" | "secondary" | "ghost" | "link";
  size?: "default" | "sm" | "lg" | "icon";
}

export function PermissionButton({ 
  module, 
  action, 
  children, 
  fallback = null,
  onClick,
  disabled = false,
  variant = "default",
  size = "default",
  ...props 
}: PermissionButtonProps) {
  const { can } = usePermission();
  
  if (!can(module, action)) {
    return <>{fallback}</>;
  }
  
  return (
    <Button onClick={onClick} disabled={disabled} variant={variant} size={size} {...props}>
      {children}
    </Button>
  );
}
