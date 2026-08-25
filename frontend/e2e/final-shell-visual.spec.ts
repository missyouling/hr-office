import { expect, test } from "@playwright/test";

import { ACCOUNTS, login } from "./helpers/auth";

const SYSTEM_TITLE = "人事行政管理";
const SCREENSHOT_DIRECTORY = "test-results/screenshots";

test("管理员新壳工作台保留浅色与深色视觉取证", async ({ page }) => {
  await page.addInitScript(() => {
    window.localStorage.setItem("theme", "light");
  });

  await login(page, ACCOUNTS.admin.username, ACCOUNTS.admin.password);

  const systemTitle = page.getByText(SYSTEM_TITLE, { exact: true });
  const floatingDock = page.locator("[data-floating-dock]");

  await expect(page.locator('[data-shell="new"]')).toHaveCount(1);
  await expect(systemTitle).toBeVisible();
  await expect(systemTitle).not.toHaveClass(/rolling-text/);
  await expect(page.locator('[data-slot="app-main-content"]')).toBeVisible();
  await expect(floatingDock).toBeVisible();

  await page.screenshot({ path: `${SCREENSHOT_DIRECTORY}/final-shell-light.png` });

  await page.getByRole("button", { name: "切换深色模式", exact: true }).click();
  await expect(page.locator("html")).toHaveClass(/dark/);
  await expect(floatingDock).toBeVisible();

  await page.screenshot({ path: `${SCREENSHOT_DIRECTORY}/final-shell-dark.png` });
});
