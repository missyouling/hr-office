import { expect, test, type APIRequestContext, type Page } from "@playwright/test";

import { ACCOUNTS, login } from "./helpers/auth";

const API_BASE = "http://127.0.0.1:8080/api";
const EMPLOYEE_NAME = "E2E培训测试员工";
const EMPLOYEE_DEPARTMENT = "E2E测试部";
const TOPIC = `E2E内部培训-${Date.now()}`;
const VOID_REASON = "E2E 自动化测试取消培训";

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

async function createInternalTrainingViaUI(page: Page) {
  await page.getByRole("button", { name: "新建培训记录", exact: true }).click();
  const dialog = page.getByRole("dialog", { name: "新建培训记录" });
  await expect(dialog).toBeVisible();
  await dialog.locator("label", { hasText: "培训主题" }).locator("input").fill(TOPIC);
  await dialog.getByLabel("培训类型", { exact: true }).selectOption({ label: "内部培训" });
  await dialog.locator("label", { hasText: "培训日期" }).locator("input").fill(today());
  await dialog.locator("label", { hasText: "讲师或机构" }).locator("input").fill("E2E人力资源部");
  await dialog.getByLabel("关联员工", { exact: true }).selectOption({ label: `${EMPLOYEE_NAME} · ${EMPLOYEE_DEPARTMENT}` });
  await dialog.getByRole("button", { name: "保存草稿", exact: true }).click();
  await expect(dialog).toHaveCount(0, { timeout: 10_000 });
  await expect(page.getByText("培训草稿已创建", { exact: false })).toBeVisible({ timeout: 10_000 });
}

function trainingRow(page: Page) {
  return page.locator("tr").filter({ hasText: TOPIC }).first();
}

async function apiLogin(request: APIRequestContext): Promise<string> {
  const response = await request.post(`${API_BASE}/auth/login`, { data: ACCOUNTS.admin });
  expect(response.ok()).toBeTruthy();
  return ((await response.json()) as { token: string }).token;
}

async function fetchTrainingRecord(request: APIRequestContext, token: string) {
  const response = await request.get(`${API_BASE}/training-records`, {
    headers: { Authorization: `Bearer ${token}` },
  });
  expect(response.ok()).toBeTruthy();
  const records = (await response.json()) as Array<{ topic: string; status: string; void_reason: string }>;
  const record = records.find((item) => item.topic === TOPIC);
  if (!record) throw new Error(`未找到培训主题 ${TOPIC}`);
  return record;
}

async function fetchEmployeeProfile(request: APIRequestContext, token: string) {
  const response = await request.get(`${API_BASE}/employees`, {
    headers: { Authorization: `Bearer ${token}` },
  });
  expect(response.ok()).toBeTruthy();
  const employees = (await response.json()) as Array<{ name: string; department: string; position: string; job_level: string; status: string }>;
  const employee = employees.find((item) => item.name === EMPLOYEE_NAME);
  if (!employee) throw new Error(`未找到员工 ${EMPLOYEE_NAME}`);
  return employee;
}

test.describe("P12.3.8 培训管理隔离 E2E", () => {
  test.describe.configure({ mode: "serial" });

  test("admin（新壳）：入口→UI内部培训草稿→完成→填写原因作废→员工资料不变", async ({ page, request }) => {
    await login(page, ACCOUNTS.admin.username, ACCOUNTS.admin.password);
    test.skip(!(await isNewShell(page)), "旧壳没有培训管理入口，无法执行新壳验收流程");

    const token = await apiLogin(request);
    const employeeBefore = await fetchEmployeeProfile(request, token);

    await expandEmployeeGroup(page);
    await expect(page.getByRole("button", { name: "培训管理", exact: true })).toBeVisible();
    await page.getByRole("button", { name: "培训管理", exact: true }).click();
    await expect(page.getByRole("heading", { name: "培训管理", exact: true })).toBeVisible();

    await createInternalTrainingViaUI(page);
    await expect(trainingRow(page).getByText("草稿", { exact: true })).toBeVisible();
    await trainingRow(page).getByRole("button", { name: "完成", exact: true }).click();
    await expect(page.getByText("培训记录已完成", { exact: false })).toBeVisible({ timeout: 10_000 });
    await expect(trainingRow(page).getByText("已完成", { exact: true })).toBeVisible();

    await trainingRow(page).getByRole("button", { name: "作废", exact: true }).click();
    const dialog = page.getByRole("dialog", { name: "作废培训记录" });
    await expect(dialog).toBeVisible();
    await dialog.locator("textarea").fill(VOID_REASON);
    await dialog.getByRole("button", { name: "确认作废", exact: true }).click();
    await expect(dialog).toHaveCount(0, { timeout: 10_000 });
    await expect(page.getByText("培训记录已作废", { exact: false })).toBeVisible({ timeout: 10_000 });

    await expect.poll(() => fetchTrainingRecord(request, token)).toMatchObject({
      status: "voided",
      void_reason: VOID_REASON,
    });
    await expect.poll(() => fetchEmployeeProfile(request, token)).toEqual(employeeBefore);
  });

  test("viewer：无 training.view 时隐藏培训管理入口", async ({ page }) => {
    await login(page, ACCOUNTS.viewer.username, ACCOUNTS.viewer.password);
    if (await isNewShell(page)) {
      await expandEmployeeGroup(page);
    }
    await expect(page.getByRole("button", { name: "培训管理", exact: true })).toHaveCount(0);
  });
});
