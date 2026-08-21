import { test, expect, type Locator, type Page } from "@playwright/test";
import { ACCOUNTS, login } from "./helpers/auth";

/**
 * P12.3.2 入职管理 E2E
 *
 * 前置依赖（由后端 seed-e2e 幂等准备）：
 * - 固定账号 admin/viewer（helpers/auth.ts 的 ACCOUNTS 与 seed-e2e 一致）；
 * - 稳定 E2E 测试部门「E2E测试部」（seed-e2e 常量 e2eEmployeeDepartment）；
 * - 在职员工「E2E离职测试员工」（身份证 110101199001011234），用于身份证冲突用例。
 *
 * 壳层覆盖（入职管理独立页面与入口固定新壳提供）：
 * - 新壳：侧边栏「员工管理」分组下存在独立「入职管理」入口（employee.create 权限）。
 *   覆盖 admin 完整入职流程（登记待入职→确认入职→快速入职→放弃→恢复→身份证冲突拒绝→
 *   花名册可见），以及 viewer 无 employee.create 时入口必须隐藏的权限边界。
 * - 旧壳：侧边栏无独立「入职管理」入口（产品现状，app-sidebar.tsx 无该菜单）。
 *   - admin 流程无法执行 → 显式 test.skip 并注明阻塞；
 *   - viewer 仅收敛断言「无入职管理新入口」。
 *
 * admin 流程（新壳，串行）：
 * 1. 登记待入职（固定唯一测试姓名/身份证/计划日期/部门 E2E测试部）→ 验证待入职行；
 * 2. 确认入职（用工状态默认试用期）→ 「已入职」筛选验证状态流转；
 * 3. 快速入职（显式选择正式 formal）→ 「已入职」筛选验证；
 * 4. 另建一条待入职 → 放弃（原因+备注必填）→ 「已放弃」验证 → 恢复 → 「待入职」验证；
 * 5. 身份证冲突（对 seed 在职员工同证件登记）→ 后端 409 → toast 失败提示、对话框不关闭、
 *    列表不新增；
 * 6. 切到员工花名册（在职员工 tab）验证确认入职/快速入职的员工可见（真实 UI 数据闭环）。
 *
 * viewer：新壳展开「员工管理」组断言无「入职管理」入口；旧壳断言无「入职管理」新入口。
 *
 * 说明与前置条件：
 * - 本测试在隔离 E2E 测试库执行（docs/frontend-ui-migration-verification.md 约定独立库）；
 *   入职记录无删除入口，重复运行会积累同名已入职记录，建议库级重置后运行。
 *   为降低重复运行风险，登记步骤会复用「全部」下已存在的同名待入职记录（UI 数据复用）。
 * - 确认入职/快速入职要求「E2E测试部」在 departments 表存在且已配置部门编码，
 *   否则后端返回 422「部门未配置编码，无法生成工号」（功能设计行为，非测试缺陷）。
 * - 全程仅通过真实 UI 操作，禁止直接调用 API 或数据库。
 */

/** seed-e2e 在职员工「E2E离职测试员工」的固定身份证（用于冲突用例，避免与离职种子冲突的是本文件主流程身份证）。 */
const SEED_EMPLOYEE_ID_NUMBER = "110101199001011234";

/** 本文件唯一稳定的测试身份证（虚构测试号码，非真实个人信息）。 */
const MAIN_ID_NUMBER = "110101199506074321";
const QUICK_ID_NUMBER = "110101199708154321";
const ABANDON_ID_NUMBER = "110101199211234321";

/** 固定唯一测试姓名。 */
const MAIN_NAME = "E2E入职测试-登记确认";
const QUICK_NAME = "E2E入职测试-快速正式";
const ABANDON_NAME = "E2E入职测试-放弃恢复";
const CONFLICT_NAME = "E2E入职测试-身份证冲突";

/** 后端 seed 提供的稳定 E2E 测试部门与岗位。 */
const E2E_DEPARTMENT = "E2E测试部";
const E2E_POSITION = "测试专员";

const ABANDON_REASON = "E2E 自动化测试放弃入职";
const ABANDON_REMARKS = "候选人放弃，已电话确认";

/** 计算未来第 days 天的本地日期（YYYY-MM-DD），作为计划入职日期。 */
function plusDays(days: number): string {
  const date = new Date();
  date.setDate(date.getDate() + days);
  const month = String(date.getMonth() + 1).padStart(2, "0");
  const day = String(date.getDate()).padStart(2, "0");
  return `${date.getFullYear()}-${month}-${day}`;
}

/** 固定新壳页面渲染 [data-shell="new"] 标记。 */
async function isNewShell(page: Page): Promise<boolean> {
  return (await page.locator('[data-shell="new"]').count()) > 0;
}

/** 展开侧边栏「员工管理」分组（新壳分组按钮带 aria-expanded）。 */
async function expandEmployeeGroup(page: Page) {
  const groupToggle = page.getByRole("button", { name: "员工管理", exact: true });
  await expect(groupToggle).toBeVisible();
  if ((await groupToggle.getAttribute("aria-expanded")) !== "true") {
    await groupToggle.click();
  }
}

/** 新壳：从侧边栏「员工管理」分组进入「入职管理」页面。 */
async function openOnboardingManagement(page: Page) {
  await expandEmployeeGroup(page);
  await page.getByRole("button", { name: "入职管理", exact: true }).click();
  await expect(page.getByRole("heading", { name: "入职管理", exact: true })).toBeVisible();
}

/** 点击入职状态筛选按钮（全部/待入职/已入职/已放弃）。 */
async function switchFilter(page: Page, label: string) {
  await page.getByRole("button", { name: label, exact: true }).click();
}

/** 登记待入职；「全部」下已存在同名待入职记录时复用（UI 数据复用，避免重复运行堆积）。 */
async function registerPending(page: Page, name: string, idNumber: string): Promise<Locator> {
  await switchFilter(page, "全部");
  const existingPending = page
    .locator("tr")
    .filter({ hasText: name })
    .filter({ has: page.getByText("待入职", { exact: true }) });
  if ((await existingPending.count()) > 0) {
    return existingPending.first();
  }
  await page.getByRole("button", { name: "登记待入职", exact: true }).click();
  const dialog = page.getByRole("dialog", { name: "登记待入职" });
  await expect(dialog).toBeVisible();
  await dialog.locator("#onboarding-name").fill(name);
  await dialog.locator("#onboarding-id-number").fill(idNumber);
  await dialog.locator("#onboarding-date").fill(plusDays(30));
  await dialog.locator("#onboarding-department").fill(E2E_DEPARTMENT);
  await dialog.locator("#onboarding-position").fill(E2E_POSITION);
  await dialog.getByRole("button", { name: "保存登记", exact: true }).click();
  await expect(dialog).toHaveCount(0, { timeout: 10_000 });
  return page.locator("tr").filter({ hasText: name });
}

/** 在指定状态筛选下断言该姓名记录恰好一条且状态标签正确。 */
async function expectRowInFilter(page: Page, name: string, statusLabel: string) {
  await switchFilter(page, statusLabel);
  const row = page.locator("tr").filter({ hasText: name });
  await expect(row).toHaveCount(1, { timeout: 15_000 });
  await expect(row.getByText(statusLabel, { exact: true })).toBeVisible();
}

/** 确认入职（用工状态保持默认试用期）。 */
async function confirmOnboarding(page: Page, row: Locator) {
  await row.getByRole("button", { name: "确认入职", exact: true }).click();
  const dialog = page.getByRole("dialog", { name: "确认员工入职" });
  await expect(dialog).toBeVisible();
  await dialog.getByRole("button", { name: "确认入职", exact: true }).click();
  await expect(dialog).toHaveCount(0, { timeout: 10_000 });
}

/** 快速入职：登记+确认一步到位，显式选择用工状态为正式 formal。 */
async function quickOnboard(page: Page, name: string, idNumber: string) {
  await page.getByRole("button", { name: "快速入职", exact: true }).click();
  const dialog = page.getByRole("dialog", { name: "快速入职" });
  await expect(dialog).toBeVisible();
  await dialog.locator("#onboarding-name").fill(name);
  await dialog.locator("#onboarding-id-number").fill(idNumber);
  await dialog.locator("#onboarding-date").fill(plusDays(30));
  await dialog.locator("#onboarding-department").fill(E2E_DEPARTMENT);
  await dialog.locator("#onboarding-position").fill(E2E_POSITION);
  await dialog.locator("#onboarding-employment").selectOption("formal");
  await dialog.getByRole("button", { name: "确认快速入职", exact: true }).click();
  await expect(dialog).toHaveCount(0, { timeout: 10_000 });
}

/** 放弃入职（原因+备注必填）。 */
async function abandonOnboarding(page: Page, row: Locator) {
  await row.getByRole("button", { name: "放弃", exact: true }).click();
  const dialog = page.getByRole("dialog", { name: "放弃本次入职" });
  await expect(dialog).toBeVisible();
  await dialog.locator("#abandon-reason").fill(ABANDON_REASON);
  await dialog.locator("#abandon-remarks").fill(ABANDON_REMARKS);
  await dialog.getByRole("button", { name: "确认放弃", exact: true }).click();
  await expect(dialog).toHaveCount(0, { timeout: 10_000 });
}

/** 对「已放弃」筛选下的记录恢复为待入职（恢复后该行从当前筛选消失）。 */
async function restoreAbandoned(page: Page, name: string) {
  const row = page.locator("tr").filter({ hasText: name });
  await row.getByRole("button", { name: "恢复待入职", exact: true }).click();
  await expect(row).toHaveCount(0, { timeout: 15_000 });
}

/** 身份证冲突：对 seed 在职员工同证件登记，断言失败提示且列表不新增。 */
async function rejectConflictingRegistration(page: Page) {
  await switchFilter(page, "全部");
  await page.getByRole("button", { name: "登记待入职", exact: true }).click();
  const dialog = page.getByRole("dialog", { name: "登记待入职" });
  await expect(dialog).toBeVisible();
  await dialog.locator("#onboarding-name").fill(CONFLICT_NAME);
  await dialog.locator("#onboarding-id-number").fill(SEED_EMPLOYEE_ID_NUMBER);
  await dialog.locator("#onboarding-date").fill(plusDays(30));
  await dialog.locator("#onboarding-department").fill(E2E_DEPARTMENT);
  await dialog.locator("#onboarding-position").fill(E2E_POSITION);
  await dialog.getByRole("button", { name: "保存登记", exact: true }).click();
  // 后端 409 → 前端 toast 展示错误信息（toast 文本形如 [409] 该身份证号已存在员工记录…）
  await expect(page.getByText("该身份证号已存在员工记录", { exact: false })).toBeVisible({ timeout: 10_000 });
  // 保存失败对话框保持打开，取消后列表无新增
  await expect(dialog).toBeVisible();
  await dialog.getByRole("button", { name: "取消", exact: true }).click();
  await expect(dialog).toHaveCount(0);
  await expect(page.locator("tr").filter({ hasText: CONFLICT_NAME })).toHaveCount(0);
}

/** 新壳：从侧边栏进入员工花名册（在职员工 tab 为默认）。 */
async function openRoster(page: Page) {
  await expandEmployeeGroup(page);
  await page.getByRole("button", { name: "员工花名册", exact: true }).click();
  await expect(page.getByRole("heading", { name: "员工花名册", exact: true })).toBeVisible();
  await expect(page.getByText("在职员工管理", { exact: true })).toBeVisible();
}

test.describe("P12.3.2 入职管理 E2E", () => {
  // 入职流程会创建真实员工与记录，串行执行避免并发互相影响。
  test.describe.configure({ mode: "serial" });

  test("admin（新壳）：登记→确认入职→快速入职(formal)→放弃→恢复→冲突拒绝→花名册可见", async ({ page }) => {
    await login(page, ACCOUNTS.admin.username, ACCOUNTS.admin.password);

    // 旧壳侧边栏无独立「入职管理」入口（产品现状），新壳流程无法执行 → 阻塞，跳过并注明。
    test.skip(!(await isNewShell(page)), "旧壳侧边栏无独立「入职管理」入口（产品现状），新壳流程无法执行，阻塞待主协调者决策");

    await openOnboardingManagement(page);

    // 1) 登记待入职 → 确认入职（默认试用期）
    const mainRow = await registerPending(page, MAIN_NAME, MAIN_ID_NUMBER);
    await expect(mainRow.getByText("待入职", { exact: true })).toBeVisible();
    await confirmOnboarding(page, mainRow);
    await expectRowInFilter(page, MAIN_NAME, "已入职");

    // 2) 快速入职（选择 formal）→ 已入职
    await quickOnboard(page, QUICK_NAME, QUICK_ID_NUMBER);
    await expectRowInFilter(page, QUICK_NAME, "已入职");

    // 3) 另建待入职 → 放弃（原因+备注）→ 已放弃 → 恢复 → 待入职
    const abandonRow = await registerPending(page, ABANDON_NAME, ABANDON_ID_NUMBER);
    await expect(abandonRow.getByText("待入职", { exact: true })).toBeVisible();
    await abandonOnboarding(page, abandonRow);
    await expectRowInFilter(page, ABANDON_NAME, "已放弃");
    await restoreAbandoned(page, ABANDON_NAME);
    await expectRowInFilter(page, ABANDON_NAME, "待入职");

    // 4) 身份证冲突（对 seed 在职员工同证件登记）→ 失败且不新增
    await rejectConflictingRegistration(page);

    // 5) 已入职员工落花名册（在职员工 tab）可见
    await openRoster(page);
    await expect(page.getByText(MAIN_NAME, { exact: true }).first()).toBeVisible({ timeout: 20_000 });
    await expect(page.getByText(QUICK_NAME, { exact: true }).first()).toBeVisible({ timeout: 20_000 });
  });

  test("viewer：无「入职管理」入口（无 employee.create 权限）", async ({ page }) => {
    await login(page, ACCOUNTS.viewer.username, ACCOUNTS.viewer.password);

    if (await isNewShell(page)) {
      // 新壳：展开「员工管理」分组，断言无「入职管理」入口（无 employee.create 权限）
      await expandEmployeeGroup(page);
      await expect(page.getByRole("button", { name: "入职管理", exact: true })).toHaveCount(0);
    } else {
      // 旧壳：入职管理独立页面与入口仅存在于新壳，收敛断言「无入职管理新入口」
      await expect(page.getByRole("button", { name: "入职管理", exact: true })).toHaveCount(0);
    }
  });
});
