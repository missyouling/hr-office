import { expect, test, type APIRequestContext, type Page } from "@playwright/test";

import { ACCOUNTS, login } from "./helpers/auth";

const API_BASE = "http://127.0.0.1:8080/api";
const LOCATION = `E2E例行检查-${Date.now()}`;
const VOID_REASON = "E2E 自动化测试取消检查";

function today(): string {
  const date = new Date();
  const month = String(date.getMonth() + 1).padStart(2, "0");
  const day = String(date.getDate()).padStart(2, "0");
  return `${date.getFullYear()}-${month}-${day}`;
}

async function isNewShell(page: Page): Promise<boolean> {
  return (await page.locator('[data-shell="new"]').count()) > 0;
}

async function expandAdminGroup(page: Page) {
  const groupToggle = page.getByRole("button", { name: "行政管理", exact: true });
  await expect(groupToggle).toBeVisible();
  if ((await groupToggle.getAttribute("aria-expanded")) !== "true") {
    await groupToggle.click();
  }
}

async function createRoutineInspectionViaUI(page: Page) {
  await page.getByRole("button", { name: "新建安全检查", exact: true }).click();
  const dialog = page.getByRole("dialog", { name: "新建安全检查" });
  await expect(dialog).toBeVisible();
  await dialog.getByLabel("检查类型", { exact: true }).selectOption("routine");
  await dialog.locator("label", { hasText: "检查日期" }).locator("input").fill(today());
  await dialog.locator("label", { hasText: "检查地点" }).locator("input").fill(LOCATION);
  await dialog.locator("label", { hasText: "负责人" }).locator("input").fill("E2E安全员");
  await dialog.getByLabel("问题描述", { exact: true }).fill("发现消防通道堆放杂物");
  await dialog.getByLabel("整改要求", { exact: true }).fill("立即清理并复查");
  await dialog.getByRole("button", { name: "保存草稿", exact: true }).click();
  await expect(dialog).toHaveCount(0, { timeout: 10_000 });
}

function inspectionRow(page: Page) {
  return page.locator("tbody tr").filter({ hasText: LOCATION });
}

async function apiLogin(request: APIRequestContext): Promise<string> {
  const response = await request.post(`${API_BASE}/auth/login`, { data: ACCOUNTS.admin });
  expect(response.ok()).toBeTruthy();
  return ((await response.json()) as { token: string }).token;
}

async function fetchInspection(request: APIRequestContext, token: string) {
  const response = await request.get(`${API_BASE}/safety-inspections`, {
    headers: { Authorization: `Bearer ${token}` },
  });
  expect(response.ok()).toBeTruthy();
  const records = (await response.json()) as Array<{ location: string; status: string; void_reason: string }>;
  const record = records.find((item) => item.location === LOCATION);
  if (!record) throw new Error(`未找到检查地点 ${LOCATION}`);
  return record;
}

test.describe("P12.3.9 安全管理隔离 E2E", () => {
  test.describe.configure({ mode: "serial" });

  test("admin（新壳）：入口→UI例行检查草稿→完成→填写原因作废→API核对状态与原因", async ({ page, request }) => {
    await login(page, ACCOUNTS.admin.username, ACCOUNTS.admin.password);
    test.skip(!(await isNewShell(page)), "旧壳没有安全管理入口，无法执行新壳验收流程");

    await expandAdminGroup(page);
    await expect(page.getByRole("button", { name: "安全管理", exact: true })).toBeVisible();
    await page.getByRole("button", { name: "安全管理", exact: true }).click();
    await expect(page.getByRole("heading", { name: "安全管理", exact: true })).toBeVisible();

    await createRoutineInspectionViaUI(page);
    await expect(inspectionRow(page).getByText("草稿", { exact: true })).toBeVisible();
    await inspectionRow(page).getByRole("button", { name: "完成", exact: true }).click();
    await expect(page.getByText("安全检查已完成", { exact: false })).toBeVisible({ timeout: 10_000 });
    await expect(inspectionRow(page).getByText("已完成", { exact: true })).toBeVisible();

    await inspectionRow(page).getByRole("button", { name: "作废", exact: true }).click();
    const dialog = page.getByRole("dialog", { name: "作废安全检查" });
    await expect(dialog).toBeVisible();
    await dialog.locator("textarea").fill(VOID_REASON);
    await dialog.getByRole("button", { name: "确认作废", exact: true }).click();
    await expect(dialog).toHaveCount(0, { timeout: 10_000 });
    await expect(page.getByText("安全检查已作废", { exact: false })).toBeVisible({ timeout: 10_000 });

    const token = await apiLogin(request);
    await expect.poll(() => fetchInspection(request, token)).toMatchObject({
      status: "voided",
      void_reason: VOID_REASON,
    });
  });

  test("viewer：无 safety.view 时隐藏安全管理入口", async ({ page }) => {
    await login(page, ACCOUNTS.viewer.username, ACCOUNTS.viewer.password);
    if (await isNewShell(page)) {
      await expandAdminGroup(page);
    }
    await expect(page.getByRole("button", { name: "安全管理", exact: true })).toHaveCount(0);
  });
});
