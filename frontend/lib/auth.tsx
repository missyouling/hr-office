"use client";

import { createContext, useContext, useEffect, useState, useCallback, useMemo } from "react";
import { useRouter } from "next/navigation";
import { getUserProfile, normalizeAuthUser } from "./api";
import type { User } from "./types";

export type { User };

// ========== 权限降级状态（P7.1 fallback 场景 3） ==========
// 使用独立 context，避免 hasPermission 变动引发全局重渲染
interface PermissionDegradedContextType {
  /** 当前是否处于权限降级模式（网络错误导致） */
  degraded: boolean;
  setDegraded: (v: boolean) => void;
}

const PermissionDegradedContext = createContext<PermissionDegradedContextType>({
  degraded: false,
  setDegraded: () => {},
});

interface AuthContextType {
  user: User | null;
  token: string | null;
  isLoading: boolean;
  isAuthenticated: boolean;
  /** P7.1: 当前用户扁平权限列表（从 user.permissions 派生） */
  permissions: string[];
  /** P7.1: 检查当前用户是否有指定 resource.action 权限 */
  hasPermission: (resource: string, action: string) => boolean;
  login: (token: string, user: User, refreshToken?: string) => void;
  logout: () => void;
  refreshUser: () => Promise<void>;
}

const AuthContext = createContext<AuthContextType | undefined>(undefined);

/**
 * 判断错误是否为网络错误（非 401/403 认证类错误）。
 * 网络错误特征：fetch 抛出的 TypeError、AbortError、超时等。
 */
function isNetworkError(error: unknown): boolean {
  if (error instanceof TypeError) return true;
  if (error instanceof DOMException && error.name === "AbortError") return true;
  if (error instanceof Error) {
    const msg = error.message.toLowerCase();
    // fetch 网络错误会抛出 "Failed to fetch" 等
    if (msg.includes("network") || msg.includes("fetch") || msg.includes("timeout")) return true;
  }
  return false;
}

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const [user, setUser] = useState<User | null>(null);
  const [token, setToken] = useState<string | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [degraded, setDegraded] = useState(false);
  const router = useRouter();

  const logout = useCallback(() => {
    localStorage.removeItem("token");
    localStorage.removeItem("refresh_token");
    localStorage.removeItem("user");
    setToken(null);
    setUser(null);
    setDegraded(false);
    router.push("/auth");
  }, [router]);

  const validateToken = useCallback(async (authToken: string) => {
    try {
      const userData = normalizeAuthUser(await getUserProfile(authToken));
      setUser(userData);
      setDegraded(false); // 验证成功则清除降级标记
    } catch (error) {
      console.error("Token validation error:", error);
      // P7.1 场景区分：网络错误 → 降级(保留缓存用户)；认证错误 → 登出
      if (isNetworkError(error)) {
        setDegraded(true);
        // 保留 localStorage 中的缓存用户，不做登出
        return;
      }
      logout();
      throw error;
    }
  }, [logout]);

  // 初始化认证状态
  useEffect(() => {
    const initAuth = async () => {
      try {
        const storedToken = localStorage.getItem("token");
        const storedUser = localStorage.getItem("user");

        if (storedToken && storedUser) {
          setToken(storedToken);
          setUser(JSON.parse(storedUser));

          // 验证 token 有效性
          await validateToken(storedToken);
        } else {
          // 无凭证，跳转登录页
          if (typeof window !== 'undefined' && window.location.pathname !== '/auth') {
            router.push('/auth');
          }
        }
      } catch (error) {
        console.error("Auth initialization error:", error);
        // P7.1 场景区分
        if (isNetworkError(error)) {
          setDegraded(true);
          // 保留 localStorage 缓存，不清除
        } else {
          localStorage.removeItem("token");
          localStorage.removeItem("refresh_token");
          localStorage.removeItem("user");
          setToken(null);
          setUser(null);
          if (typeof window !== 'undefined') {
            router.push('/auth');
          }
        }
      } finally {
        setIsLoading(false);
      }
    };

    initAuth();
  }, [validateToken, router]);

  const login = (newToken: string, newUser: User, refreshToken?: string) => {
    localStorage.setItem("token", newToken);
    localStorage.setItem("user", JSON.stringify(newUser));
    if (refreshToken) {
      localStorage.setItem("refresh_token", refreshToken);
    }
    setToken(newToken);
    setUser(newUser);
    setDegraded(false);
  };

  const refreshUser = useCallback(async () => {
    if (!token) return;

    try {
      const userData = normalizeAuthUser(await getUserProfile(token));
      setUser(userData);
      localStorage.setItem("user", JSON.stringify(userData));
      setDegraded(false);
    } catch (error) {
      console.error("User refresh error:", error);
      // P7.1 场景区分
      if (isNetworkError(error)) {
        setDegraded(true);
        // 保留现有用户数据，不做登出
        return;
      }
      logout();
    }
  }, [token, logout]);

  // P7.1: 从 user 或 localStorage 缓存计算 permissions
  const permissions = useMemo<string[]>(() => {
    if (user?.permissions) return user.permissions;
    // 降级时从 localStorage 缓存读取
    try {
      const cached = localStorage.getItem("user");
      if (cached) {
        const parsed = JSON.parse(cached) as User;
        return parsed.permissions ?? [];
      }
    } catch {
      // ignore
    }
    return [];
  }, [user]);

  // P7.1: 权限判断核心函数
  // 场景 1: 未登录 → false
  // 场景 2: token 失效（已被 401 拦截器处理，user 为 null） → false
  // 场景 3: 网络错误（degraded=true 且 user 不为 null） → true（乐观允许）
  const hasPermission = useCallback((resource: string, action: string): boolean => {
    // 场景 1: 未登录
    if (!user) return false;
    // 场景 3: 网络错误降级模式 → 乐观允许
    if (degraded) return true;
    // 正常模式：检查扁平权限数组
    const required = `${resource}.${action}`;
    return permissions.includes(required);
  }, [user, degraded, permissions]);

  const value: AuthContextType = {
    user,
    token,
    isLoading,
    isAuthenticated: !!user && !!token,
    permissions,
    hasPermission,
    login,
    logout,
    refreshUser,
  };

  return (
    <PermissionDegradedContext.Provider value={{ degraded, setDegraded }}>
      <AuthContext.Provider value={value}>{children}</AuthContext.Provider>
    </PermissionDegradedContext.Provider>
  );
}

export function useAuth() {
  const context = useContext(AuthContext);
  if (context === undefined) {
    throw new Error("useAuth must be used within an AuthProvider");
  }
  return context;
}

/**
 * P7.1: 权限降级状态 Hook。
 * 当网络错误导致无法验证 token 时，系统进入降级模式，
 * 所有权限检查乐观返回 true，UI 可借此展示警告横幅。
 */
export function usePermissionDegraded(): boolean {
  const { degraded } = useContext(PermissionDegradedContext);
  return degraded;
}

// Hook for protected routes
export function useRequireAuth() {
  const { isAuthenticated, isLoading } = useAuth();
  const router = useRouter();

  useEffect(() => {
    if (!isLoading && !isAuthenticated) {
      router.push("/auth");
    }
  }, [isAuthenticated, isLoading, router]);

  return { isAuthenticated, isLoading };
}
