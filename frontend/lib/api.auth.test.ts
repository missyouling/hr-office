import { describe, expect, it } from "vitest";
import { normalizeAuthUser } from "./api";

describe("normalizeAuthUser", () => {
  it("将认证响应的顶层权限写入用户对象", () => {
    const user = normalizeAuthUser({
      user: {
        id: 1,
        username: "viewer",
        email: "viewer@example.com",
        full_name: "只读用户",
        active: true,
        created_at: "2026-01-01T00:00:00Z",
        updated_at: "2026-01-01T00:00:00Z",
      },
      permissions: ["employee.view"],
    });

    expect(user.permissions).toEqual(["employee.view"]);
    expect(user.username).toBe("viewer");
  });

  it("保留空权限数组，避免使用废弃角色字段推断权限", () => {
    const user = normalizeAuthUser({
      user: {
        id: 2,
        username: "no-role",
        email: "no-role@example.com",
        full_name: "无权限用户",
        active: true,
        created_at: "2026-01-01T00:00:00Z",
        updated_at: "2026-01-01T00:00:00Z",
      },
      permissions: [],
    });

    expect(user.permissions).toEqual([]);
    expect(user.role).toBeUndefined();
  });
});
