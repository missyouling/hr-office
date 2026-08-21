import { expect, test, type APIRequestContext, type Page } from "@playwright/test";

import { ACCOUNTS, login } from "./helpers/auth";

const API_BASE = "http://127.0.0.1:8080/api";
// 唯一车牌：时间戳保证每次运行不冲突（plate_number 长度上限 20，E2E- + 13 位时间戳 = 17 字符）
const PLATE = `E2E-${Date.now()}`;
const MODEL = `E2E测试车型-${Date.now()}`;
const MODEL_EDITED = `${MODEL}-改`;
const BRAND = "E2E品牌";

function today(): string {
  const date = new Date();
  const month = String(date.getMonth() + 1).padStart(2, "0");
  const day = String(date.getDate()).padStart(2, "0");
  return `${date.getFullYear()}-${month}-${day}`;
}

async function isNewShell(page: Page): Promise<boolean> {
  return (await page.locator('[data-shell="new"]').count()) > 0;
}

async function expandDailyGroup(page: Page) {
  const groupToggle = page.getByRole("button", { name: "日常事务", exact: true });
  await expect(groupToggle).toBeVisible();
  if ((await groupToggle.getAttribute("aria-expanded")) !== "true") {
    await groupToggle.click();
  }
}

function vehicleRow(page: Page) {
  return page.locator("tbody tr").filter({ hasText: PLATE });
}

async function apiLogin(request: APIRequestContext): Promise<string> {
  const response = await request.post(`${API_BASE}/auth/login`, { data: ACCOUNTS.admin });
  expect(response.ok()).toBeTruthy();
  return ((await response.json()) as { token: string }).token;
}

async function fetchVehicles(request: APIRequestContext, token: string) {
  const response = await request.get(`${API_BASE}/fleet-vehicles`, {
    headers: { Authorization: `Bearer ${token}` },
  });
  expect(response.ok()).toBeTruthy();
  return (await response.json()) as Array<{ plate_number: string; status: string; vehicle_model: string }>;
}

test.describe("P12 车辆档案隔离 E2E", () => {
  test.describe.configure({ mode: "serial" });

  test("admin（新壳）：新增 active 车辆→编辑→停用→编辑/删除不可用→恢复→删除并 API 核验", async ({ page, request }) => {
    await login(page, ACCOUNTS.admin.username, ACCOUNTS.admin.password);
    test.skip(!(await isNewShell(page)), "旧壳没有车辆档案入口，无法执行新壳验收流程");

    // 进入车辆档案页面
    await expandDailyGroup(page);
    await expect(page.getByRole("button", { name: "车辆档案", exact: true })).toBeVisible();
    await page.getByRole("button", { name: "车辆档案", exact: true }).click();
    await expect(page.getByRole("heading", { name: "车辆档案", exact: true })).toBeVisible();

    // 1. 新增 active 车辆
    await page.getByRole("button", { name: "新增车辆", exact: true }).click();
    const dialog = page.getByRole("dialog", { name: "新增车辆" });
    await expect(dialog).toBeVisible();
    await dialog.locator("label", { hasText: "车牌号" }).locator("input").fill(PLATE);
    await dialog.locator("label", { hasText: "车型" }).locator("input").fill(MODEL);
    await dialog.locator("label", { hasText: "品牌" }).locator("input").fill(BRAND);
    await dialog.locator("label", { hasText: "座位数" }).locator("input").fill("7");
    await dialog.locator("label", { hasText: "购置日期" }).locator("input").fill(today());
    await dialog.getByRole("button", { name: "保存", exact: true }).click();
    await expect(dialog).toHaveCount(0, { timeout: 10_000 });
    await expect(page.getByText("车辆档案已创建", { exact: false })).toBeVisible({ timeout: 10_000 });
    await expect(vehicleRow(page).getByText("启用中", { exact: true })).toBeVisible();

    // 2. 编辑（active 可编辑，修改车型）
    await vehicleRow(page).getByRole("button", { name: "编辑", exact: true }).click();
    const editDialog = page.getByRole("dialog", { name: "编辑车辆档案" });
    await expect(editDialog).toBeVisible();
    await editDialog.locator("label", { hasText: "车型" }).locator("input").fill(MODEL_EDITED);
    await editDialog.getByRole("button", { name: "保存", exact: true }).click();
    await expect(editDialog).toHaveCount(0, { timeout: 10_000 });
    await expect(vehicleRow(page).getByText(MODEL_EDITED, { exact: true })).toBeVisible();

    // 3. 停用（inactive）
    await vehicleRow(page).getByRole("button", { name: "编辑", exact: true }).click();
    const deactivateDialog = page.getByRole("dialog", { name: "编辑车辆档案" });
    await expect(deactivateDialog).toBeVisible();
    await deactivateDialog.getByLabel("状态", { exact: true }).selectOption("inactive");
    await deactivateDialog.getByRole("button", { name: "保存", exact: true }).click();
    await expect(deactivateDialog).toHaveCount(0, { timeout: 10_000 });
    await expect(vehicleRow(page).getByText("已停用", { exact: true })).toBeVisible();

    // 4. 断言编辑/删除不可用，仅保留恢复启用
    await expect(vehicleRow(page).getByRole("button", { name: "编辑", exact: true })).toHaveCount(0);
    await expect(vehicleRow(page).getByRole("button", { name: "删除", exact: true })).toHaveCount(0);
    await expect(vehicleRow(page).getByRole("button", { name: "恢复启用", exact: true })).toBeVisible();

    // 5. 恢复为 active（UI + API 双重核验）
    await vehicleRow(page).getByRole("button", { name: "恢复启用", exact: true }).click();
    await expect(vehicleRow(page).getByText("启用中", { exact: true })).toBeVisible();
    const token = await apiLogin(request);
    await expect.poll(() => fetchVehicles(request, token).then((list) => list.find((v) => v.plate_number === PLATE))).toMatchObject({
      status: "active",
      vehicle_model: MODEL_EDITED,
    });

    // 6. 删除并 API 核验（列表中不再存在该车牌）
    await vehicleRow(page).getByRole("button", { name: "删除", exact: true }).click();
    const deleteDialog = page.getByRole("dialog", { name: "删除车辆档案" });
    await expect(deleteDialog).toBeVisible();
    await deleteDialog.getByRole("button", { name: "确认删除", exact: true }).click();
    await expect(deleteDialog).toHaveCount(0, { timeout: 10_000 });
    await expect(vehicleRow(page)).toHaveCount(0);
    await expect.poll(() => fetchVehicles(request, token).then((list) => list.some((v) => v.plate_number === PLATE))).toBe(false);
  });

  test("viewer：无 fleet.view 时侧栏不显示车辆档案", async ({ page }) => {
    await login(page, ACCOUNTS.viewer.username, ACCOUNTS.viewer.password);
    if (await isNewShell(page)) {
      await expandDailyGroup(page);
    }
    await expect(page.getByRole("button", { name: "车辆档案", exact: true })).toHaveCount(0);
  });
});