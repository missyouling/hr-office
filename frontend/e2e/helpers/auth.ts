import { expect, type Page } from "@playwright/test";

export const ACCOUNTS = {
  admin: { username: "admin", password: "Admin@123456" },
  manager: { username: "manager", password: "Manager@123456" },
  editor: { username: "editor", password: "Editor@123456" },
  viewer: { username: "viewer", password: "Viewer@123456" },
} as const;

const USERNAME_SELECTOR = "#login-username, #login-username-simple";
const PASSWORD_SELECTOR = "#login-password, #login-password-simple";

/** 登录指定账号并等待首页侧边栏完成渲染。 */
export async function login(page: Page, username: string, password: string) {
  await page.goto("/auth");
  await expect(page.locator(USERNAME_SELECTOR).first()).toBeVisible({ timeout: 15_000 });
  await page.locator(USERNAME_SELECTOR).first().fill(username);
  await page.locator(PASSWORD_SELECTOR).first().fill(password);
  const loginForm = page.locator(USERNAME_SELECTOR).first().locator("xpath=ancestor::form");
  await loginForm.getByRole("button", { name: "登录", exact: true }).click();
  await page.waitForURL(/\/$/, { timeout: 60_000 });
  await expect(page.getByRole("button", { name: "员工管理", exact: true })).toBeVisible({
    timeout: 30_000,
  });
}
