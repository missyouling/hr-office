import { expect, test, type APIRequestContext, type Page } from "@playwright/test";

import { ACCOUNTS, login } from "./helpers/auth";

const API_BASE = "http://127.0.0.1:8080/api";
const EMPLOYEE_NAME = "E2E人事异动测试员工";
const BEFORE_DEPARTMENT = "E2E异动原部门";
const AFTER_DEPARTMENT = "E2E异动目标部门";
const AFTER_POSITION = "高级专员";
const AFTER_JOB_LEVEL = "P4";
const REASON = `E2E人事异动晋升-${Date.now()}`;
const VOID_REASON = "E2E 自动化测试作废晋升记录";

function today(): string {
  const date = new Date();
  const month = String(date.getMonth() + 1).padStart(2, "0");
  const day = String(date.getDate()).padStart(2, "0");
  return `${date.getFullYear()}-${month}-${day}`;
}

async function isNewShell(page: Page): Promise<boolean> {
  return (await page.locator('[data-shell="new"]').count()) > 0;
}

async function expandEmployeeGroup(page: Page) {
  const groupToggle = page.getByRole("button", { name: "员工管理", exact: true });
  await expect(groupToggle).toBeVisible();
  if ((await groupToggle.getAttribute("aria-expanded")) !== "true") {
    await groupToggle.click();
  }
}

async function createPromotionDraftViaUI(page: Page) {
  await page.getByRole("button", { name: "新建人事异动", exact: true }).click();
  const dialog = page.getByRole("dialog", { name: "新建人事异动" });
  await expect(dialog).toBeVisible();
  await dialog.getByLabel("员工", { exact: true }).selectOption({ label: `${EMPLOYEE_NAME} · ${BEFORE_DEPARTMENT}` });
  await dialog.getByLabel("异动类型", { exact: true }).selectOption({ label: "晋升" });
  await dialog.locator("label", { hasText: "生效日期" }).locator("input").fill(today());
  await dialog.getByLabel("事由", { exact: true }).fill(REASON);
  await dialog.getByLabel("异动后部门", { exact: true }).selectOption({ label: AFTER_DEPARTMENT });
  await dialog.locator("label", { hasText: "异动后岗位" }).locator("input").fill(AFTER_POSITION);
  await dialog.locator("label", { hasText: "异动后职级" }).locator("input").fill(AFTER_JOB_LEVEL);
  await dialog.getByRole("button", { name: "保存草稿", exact: true }).click();
  await expect(dialog).toHaveCount(0, { timeout: 10_000 });
  await expect(page.getByText("人事异动草稿已创建", { exact: false })).toBeVisible({ timeout: 10_000 });
}

function changeRow(page: Page) {
  return page.locator("tr").filter({ hasText: REASON }).first();
}

async function apiLogin(request: APIRequestContext): Promise<string> {
  const response = await request.post(`${API_BASE}/auth/login`, {
    data: ACCOUNTS.admin,
  });
  expect(response.ok()).toBeTruthy();
  return ((await response.json()) as { token: string }).token;
}

async function fetchEmployee(request: APIRequestContext, token: string) {
  const response = await request.get(`${API_BASE}/employees`, {
    headers: { Authorization: `Bearer ${token}` },
  });
  expect(response.ok()).toBeTruthy();
  const employees = (await response.json()) as Array<{ name: string; department: string; position: string; job_level: string }>;
  const employee = employees.find((item) => item.name === EMPLOYEE_NAME);
  if (!employee) throw new Error(`未找到员工 ${EMPLOYEE_NAME}`);
  return employee;
}

test.describe("P12.3.7 人事异动隔离 E2E", () => {
  test.describe.configure({ mode: "serial" });

  test("admin（新壳）：入口→UI晋升草稿→手动生效→员工资料更新→作废不回滚", async ({ page, request }) => {
    await login(page, ACCOUNTS.admin.username, ACCOUNTS.admin.password);
    test.skip(!(await isNewShell(page)), "旧壳没有人事异动入口，无法执行新壳验收流程");

    await expandEmployeeGroup(page);
    await expect(page.getByRole("button", { name: "人事异动", exact: true })).toBeVisible();
    await page.getByRole("button", { name: "人事异动", exact: true }).click();
    await expect(page.getByRole("heading", { name: "人事异动", exact: true })).toBeVisible();

    await createPromotionDraftViaUI(page);
    await expect(changeRow(page).getByText("草稿", { exact: true })).toBeVisible();
    await changeRow(page).getByRole("button", { name: "生效", exact: true }).click();
    await expect(page.getByText("人事异动已手动生效", { exact: false })).toBeVisible();

    const token = await apiLogin(request);
    await expect.poll(() => fetchEmployee(request, token)).toMatchObject({
      department: AFTER_DEPARTMENT,
      position: AFTER_POSITION,
      job_level: AFTER_JOB_LEVEL,
    });

    await changeRow(page).getByRole("button", { name: "作废", exact: true }).click();
    const dialog = page.getByRole("dialog", { name: "作废人事异动" });
    await dialog.locator("textarea").fill(VOID_REASON);
    await dialog.getByRole("button", { name: "确认作废", exact: true }).click();
    await expect(page.getByText("人事异动已作废，不会回滚员工资料", { exact: false })).toBeVisible();
    await expect.poll(() => fetchEmployee(request, token)).toMatchObject({
      department: AFTER_DEPARTMENT,
      position: AFTER_POSITION,
      job_level: AFTER_JOB_LEVEL,
    });
  });

  test("viewer：无 employee.edit 时隐藏人事异动入口", async ({ page }) => {
    await login(page, ACCOUNTS.viewer.username, ACCOUNTS.viewer.password);
    if (await isNewShell(page)) {
      await expandEmployeeGroup(page);
    }
    await expect(page.getByRole("button", { name: "人事异动", exact: true })).toHaveCount(0);
  });
});
