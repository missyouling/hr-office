import { expect, test, type APIRequestContext, type Page } from "@playwright/test";

import { ACCOUNTS, login } from "./helpers/auth";

const API_BASE = "http://127.0.0.1:8080/api";
// 唯一标识：医疗机构名带时间戳，保证每次运行互不冲突，且表格/API 均可稳定定位
const MEDICAL_INSTITUTION = `E2E职业卫生中心-${Date.now()}`;
const CHECK_CATEGORY = "岗中职业健康检查";
const VOID_REASON = "E2E 自动化测试取消检查";
// 在职员工快照基线（seed-e2e 幂等创建/恢复的 E2E培训测试员工，状态 active）
const EMPLOYEE_NAME = "E2E培训测试员工";
const EMPLOYEE_DEPARTMENT = "E2E测试部";
const EMPLOYEE_POSITION = "测试专员";

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

async function createHealthCheckViaUI(page: Page) {
  await page.getByRole("button", { name: "新建健康检查", exact: true }).click();
  const dialog = page.getByRole("dialog", { name: "新建职业健康检查" });
  await expect(dialog).toBeVisible();
  await dialog.getByLabel("员工", { exact: true }).selectOption({ label: `${EMPLOYEE_NAME} / ${EMPLOYEE_DEPARTMENT} / ${EMPLOYEE_POSITION}` });
  await dialog.locator("label", { hasText: "检查日期 *" }).locator("input").fill(today());
  await dialog.locator("label", { hasText: "医疗机构 *" }).locator("input").fill(MEDICAL_INSTITUTION);
  await dialog.locator("label", { hasText: "检查类别 *" }).locator("input").fill(CHECK_CATEGORY);
  await dialog.getByRole("button", { name: "保存草稿", exact: true }).click();
  await expect(dialog).toHaveCount(0, { timeout: 10_000 });
  await expect(page.getByText("健康检查草稿已创建", { exact: false })).toBeVisible({ timeout: 10_000 });
}

function checkRow(page: Page) {
  return page.locator("tbody tr").filter({ hasText: MEDICAL_INSTITUTION });
}

async function apiLogin(request: APIRequestContext): Promise<string> {
  const response = await request.post(`${API_BASE}/auth/login`, { data: ACCOUNTS.admin });
  expect(response.ok()).toBeTruthy();
  return ((await response.json()) as { token: string }).token;
}

async function fetchCheck(request: APIRequestContext, token: string) {
  const response = await request.get(`${API_BASE}/occupational-health-checks`, {
    headers: { Authorization: `Bearer ${token}` },
  });
  expect(response.ok()).toBeTruthy();
  const records = (await response.json()) as Array<{
    medical_institution: string;
    employee_name: string;
    employee_department: string;
    employee_position: string;
    status: string;
    void_reason: string;
  }>;
  const record = records.find((item) => item.medical_institution === MEDICAL_INSTITUTION);
  if (!record) throw new Error(`未找到医疗机构 ${MEDICAL_INSTITUTION}`);
  return record;
}

test.describe("P12 职业卫生隔离 E2E", () => {
  test.describe.configure({ mode: "serial" });

  test("admin（新壳）：入口→UI新建职业健康检查草稿→完成→填写原因作废→API核对员工快照/状态/原因", async ({ page, request }) => {
    await login(page, ACCOUNTS.admin.username, ACCOUNTS.admin.password);
    test.skip(!(await isNewShell(page)), "旧壳没有职业健康检查入口，无法执行新壳验收流程");

    await expandAdminGroup(page);
    await expect(page.getByRole("button", { name: "职业健康检查", exact: true })).toBeVisible();
    await page.getByRole("button", { name: "职业健康检查", exact: true }).click();
    await expect(page.getByRole("heading", { name: "职业健康检查", exact: true })).toBeVisible();

    await createHealthCheckViaUI(page);
    await expect(checkRow(page).getByText("草稿", { exact: true })).toBeVisible();
    await checkRow(page).getByRole("button", { name: "完成", exact: true }).click();
    await expect(page.getByText("健康检查已完成", { exact: false })).toBeVisible({ timeout: 10_000 });
    await expect(checkRow(page).getByText("已完成", { exact: true })).toBeVisible();

    await checkRow(page).getByRole("button", { name: "作废", exact: true }).click();
    const dialog = page.getByRole("dialog", { name: "作废职业健康检查" });
    await expect(dialog).toBeVisible();
    await dialog.locator("textarea").fill(VOID_REASON);
    await dialog.getByRole("button", { name: "确认作废", exact: true }).click();
    await expect(dialog).toHaveCount(0, { timeout: 10_000 });
    await expect(page.getByText("健康检查已作废", { exact: false })).toBeVisible({ timeout: 10_000 });

    const token = await apiLogin(request);
    await expect.poll(() => fetchCheck(request, token)).toMatchObject({
      employee_name: EMPLOYEE_NAME,
      employee_department: EMPLOYEE_DEPARTMENT,
      employee_position: EMPLOYEE_POSITION,
      status: "voided",
      void_reason: VOID_REASON,
    });
  });

  test("viewer：无 occupational_health.view 时隐藏职业健康检查入口", async ({ page }) => {
    await login(page, ACCOUNTS.viewer.username, ACCOUNTS.viewer.password);
    if (await isNewShell(page)) {
      await expandAdminGroup(page);
    }
    await expect(page.getByRole("button", { name: "职业健康检查", exact: true })).toHaveCount(0);
  });
});