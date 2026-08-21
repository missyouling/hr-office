import { test, expect, type APIRequestContext, type Page } from "@playwright/test";
import { execSync } from "child_process";
import { ACCOUNTS, login } from "./helpers/auth";

/**
 * P12.3.3 转正管理 E2E
 *
 * 前置依赖（由后端 seed-e2e 幂等准备）：
 * - 固定账号 admin/manager/editor/viewer（helpers/auth.ts 的 ACCOUNTS 与 seed-e2e 一致）；
 * - admin/manager/editor 三名审批用户同租户（company_id=E2E-TENANT-001，seed-e2e 幂等设置）；
 * - 两名稳定试用期员工（seed-e2e 常量 e2eTrialEmployees）：
 *   「E2E转正测试员工」（工号 E2E-REG-001，主流程：通过 → effective → formal）、
 *   「E2E转正拒绝员工」（工号 E2E-REG-002，拒绝路径：HR 拒绝 → rejected，保持 trial）。
 *
 * 壳层覆盖（转正管理独立页面与入口固定新壳提供）：
 * - 新壳：侧边栏「员工管理」分组下存在独立「转正管理」入口（employee.edit 权限）。
 *   覆盖 admin 完整转正流程（UI 创建申请 → API 辅助推进 → UI 展示 effective → API 验证员工 formal），
 *   以及 HR 拒绝路径（员工保持 trial + 离职待办存在），
 *   以及 viewer 无 employee.edit 时入口必须隐藏的权限边界。
 * - 旧壳：侧边栏无「转正管理」入口（产品现状）。
 *   - admin 流程无法执行 → 显式 test.skip 并注明阻塞；
 *   - viewer 仅收敛断言「无转正管理新入口」。
 *
 * 说明与前置条件：
 * - 本测试在隔离 E2E 测试库执行（127.0.0.1:55432/siapp_e2e，禁止 5432/生产/Supabase）；
 * - 创建申请必须走真实 UI；上级/HR 审批通过 API 辅助推进（任务允许）；
 * - 计划转正日期填「今天」，HR 通过后立即生效（effective），不依赖真实等候 02:00；
 * - 离职待办无查询 API，通过 docker exec psql 直查隔离库验证（仅本任务 E2E 环境使用）。
 */

/** seed-e2e 两名稳定试用期员工（与 backend/cmd/seed-e2e/regularization.go 常量一致）。 */
const MAIN_EMPLOYEE_NAME = "E2E转正测试员工";
const REJECT_EMPLOYEE_NAME = "E2E转正拒绝员工";
const E2E_DEPARTMENT = "E2E测试部";

/** 三名审批用户 ID（seed-e2e 固定账号，与数据库 users 表一致）。 */
const INITIATOR_USER_ID = 1; // admin（发起 HR）
const SUPERVISOR_USER_ID = 2; // manager（直属上级）
const HR_USER_ID = 3; // editor（HR 复核）

/** 后端 API 地址（隔离 E2E 后端固定监听 127.0.0.1:8080）。 */
const API_BASE = "http://127.0.0.1:8080/api";

/** 计算今天本地日期（YYYY-MM-DD），作为计划转正日期（当天 HR 通过即生效）。 */
function today(): string {
  const date = new Date();
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

/** 新壳：从侧边栏「员工管理」分组进入「转正管理」页面。 */
async function openRegularizationManagement(page: Page) {
  await expandEmployeeGroup(page);
  await page.getByRole("button", { name: "转正管理", exact: true }).click();
  await expect(page.getByRole("heading", { name: "转正管理", exact: true })).toBeVisible();
}

/** 切换状态筛选（全部状态/待上级审批/已生效/已拒绝…），切换会触发列表重新加载。 */
async function switchStatusFilter(page: Page, label: string) {
  await page.getByLabel("状态筛选").selectOption({ label });
}

/** 在指定状态筛选下断言该员工记录至少一条且状态标签正确（重复运行会积累同名记录，取最新一条）。 */
async function expectRowInStatus(page: Page, employeeName: string, statusLabel: string) {
  await switchStatusFilter(page, statusLabel);
  const row = page.locator("tr").filter({ hasText: employeeName });
  await expect(row.first()).toBeVisible({ timeout: 15_000 });
  await expect(row.first().getByText(statusLabel, { exact: true })).toBeVisible();
}

/** 通过 UI 发起转正申请（员工/计划日期/上级/HR 必填，合同期限默认 12 个月）。 */
async function createApplicationViaUI(page: Page, employeeName: string) {
  await page.getByRole("button", { name: "新申请", exact: true }).click();
  const dialog = page.getByRole("dialog", { name: "发起转正申请" });
  await expect(dialog).toBeVisible();
  await dialog.getByLabel("员工", { exact: true }).selectOption({ label: `${employeeName} · ${E2E_DEPARTMENT}` });
  await dialog.locator("label", { hasText: "计划转正日期" }).locator("input").fill(today());
  await dialog.locator("label", { hasText: "直属上级用户 ID" }).locator("input").fill(String(SUPERVISOR_USER_ID));
  await dialog.locator("label", { hasText: "HR 复核用户 ID" }).locator("input").fill(String(HR_USER_ID));
  await dialog.getByRole("button", { name: "提交申请", exact: true }).click();
  await expect(dialog).toHaveCount(0, { timeout: 10_000 });
  await expect(page.getByText("转正申请已提交", { exact: false })).toBeVisible({ timeout: 10_000 });
}

// ───────────────────── API 辅助（上级/HR 审批推进 + 数据验证） ─────────────────────

/** 通过登录接口获取指定账号的 Bearer token。 */
async function apiLogin(request: APIRequestContext, username: string, password: string): Promise<string> {
  const res = await request.post(`${API_BASE}/auth/login`, { data: { username, password } });
  expect(res.ok()).toBeTruthy();
  const body = (await res.json()) as { token: string };
  return body.token;
}

/** 查询该员工最新一条转正记录 ID（列表按 created_at DESC，取第一条）。 */
async function findLatestRecordId(request: APIRequestContext, token: string, employeeName: string): Promise<number> {
  const res = await request.get(`${API_BASE}/regularization-records`, {
    headers: { Authorization: `Bearer ${token}` },
  });
  expect(res.ok()).toBeTruthy();
  const records = (await res.json()) as { id: number; snapshot_name: string }[];
  const match = records.find((r) => r.snapshot_name === employeeName);
  if (!match) throw new Error(`未找到员工 ${employeeName} 的转正记录`);
  return match.id;
}

/** 上级（manager）审批通过：pending_supervisor → pending_hr_review。 */
async function apiSupervisorApprove(request: APIRequestContext, recordId: number) {
  const token = await apiLogin(request, ACCOUNTS.manager.username, ACCOUNTS.manager.password);
  const res = await request.post(`${API_BASE}/regularization-records/${recordId}/supervisor-approve`, {
    headers: { Authorization: `Bearer ${token}` },
    data: { comment: "E2E 上级审批通过" },
  });
  expect(res.ok()).toBeTruthy();
}

/** HR（editor）复核通过：pending_hr_review → effective（计划日期<=今天）。 */
async function apiHRApprove(request: APIRequestContext, recordId: number) {
  const token = await apiLogin(request, ACCOUNTS.editor.username, ACCOUNTS.editor.password);
  const res = await request.post(`${API_BASE}/regularization-records/${recordId}/hr-approve`, {
    headers: { Authorization: `Bearer ${token}` },
    data: { comment: "E2E HR 复核通过" },
  });
  expect(res.ok()).toBeTruthy();
}

/** HR（editor）复核拒绝：pending_hr_review → rejected（原因必填，同事务创建离职待办）。 */
async function apiHRReject(request: APIRequestContext, recordId: number) {
  const token = await apiLogin(request, ACCOUNTS.editor.username, ACCOUNTS.editor.password);
  const res = await request.post(`${API_BASE}/regularization-records/${recordId}/hr-reject`, {
    headers: { Authorization: `Bearer ${token}` },
    data: { reason: "E2E 自动化测试驳回", comment: "试用期表现不达标" },
  });
  expect(res.ok()).toBeTruthy();
}

/** 通过 API 查询员工用工状态（employment_status: trial/formal）。复用已获取的 admin token，避免同秒重复登录触发 token_hash 唯一冲突。 */
async function fetchEmploymentStatus(request: APIRequestContext, adminToken: string, employeeName: string): Promise<string> {
  const res = await request.get(`${API_BASE}/employees`, {
    headers: { Authorization: `Bearer ${adminToken}` },
  });
  expect(res.ok()).toBeTruthy();
  const employees = (await res.json()) as { name: string; employment_status?: string }[];
  const emp = employees.find((e) => e.name === employeeName);
  if (!emp) throw new Error(`未找到员工 ${employeeName}`);
  return emp.employment_status ?? "";
}

/** 直查隔离 E2E 库（docker exec psql，仅本任务 E2E 环境使用），返回去空格结果。 */
function queryDB(sql: string): string {
  const escaped = sql.replace(/"/g, '\\"');
  return execSync(
    `docker exec siapp-e2e-postgres psql -U siapp_e2e -d siapp_e2e -t -A -c "${escaped}"`,
    { encoding: "utf-8" },
  ).trim();
}

test.describe("P12.3.3 转正管理 E2E", () => {
  // 转正流程会变更员工用工状态与记录状态，串行执行避免并发互相影响。
  test.describe.configure({ mode: "serial" });

  test("admin（新壳）：入口可见→进入页面→UI创建申请→API推进→UI展示已生效→员工formal", async ({ page, request }) => {
    await login(page, ACCOUNTS.admin.username, ACCOUNTS.admin.password);

    // 旧壳侧边栏无「转正管理」入口（产品现状），新壳流程无法执行 → 阻塞，跳过并注明。
    test.skip(!(await isNewShell(page)), "旧壳侧边栏无「转正管理」入口（产品现状），新壳流程无法执行，阻塞待主协调者决策");

    // 1) 新壳侧边栏「员工管理」分组下存在「转正管理」入口
    await expandEmployeeGroup(page);
    await expect(page.getByRole("button", { name: "转正管理", exact: true })).toBeVisible();

    // 2) 进入页面：标题 + 转正流程说明 + 新申请按钮
    await page.getByRole("button", { name: "转正管理", exact: true }).click();
    await expect(page.getByRole("heading", { name: "转正管理", exact: true })).toBeVisible();
    await expect(page.getByText("审批过程与员工快照均只读留存，不可覆盖。", { exact: true })).toBeVisible();
    await expect(page.getByRole("button", { name: "新申请", exact: true })).toBeVisible();

    // 3) UI 创建申请 → 列表出现「待上级审批」
    await createApplicationViaUI(page, MAIN_EMPLOYEE_NAME);
    await expectRowInStatus(page, MAIN_EMPLOYEE_NAME, "待上级审批");

    // 4) API 辅助推进：manager 上级通过 → editor HR 通过（计划日期=今天 → 立即生效）
    const adminToken = await apiLogin(request, ACCOUNTS.admin.username, ACCOUNTS.admin.password);
    const recordId = await findLatestRecordId(request, adminToken, MAIN_EMPLOYEE_NAME);
    await apiSupervisorApprove(request, recordId);
    await apiHRApprove(request, recordId);

    // 5) UI 展示「已生效」（切换状态筛选触发重新加载）
    await expectRowInStatus(page, MAIN_EMPLOYEE_NAME, "已生效");

    // 6) API 准确验证员工用工状态已转正（formal）
    expect(await fetchEmploymentStatus(request, adminToken, MAIN_EMPLOYEE_NAME)).toBe("formal");
  });

  test("admin（新壳）：HR 拒绝路径 → 员工仍 trial 且离职待办存在", async ({ page, request }) => {
    await login(page, ACCOUNTS.admin.username, ACCOUNTS.admin.password);

    test.skip(!(await isNewShell(page)), "旧壳侧边栏无「转正管理」入口（产品现状），新壳流程无法执行，阻塞待主协调者决策");

    // 1) 进入转正管理页面
    await openRegularizationManagement(page);

    // 2) UI 创建第二条申请（拒绝路径员工）→ 待上级审批
    await createApplicationViaUI(page, REJECT_EMPLOYEE_NAME);
    await expectRowInStatus(page, REJECT_EMPLOYEE_NAME, "待上级审批");

    // 3) API 辅助推进：manager 上级通过 → editor HR 拒绝（原因必填）
    const adminToken = await apiLogin(request, ACCOUNTS.admin.username, ACCOUNTS.admin.password);
    const recordId = await findLatestRecordId(request, adminToken, REJECT_EMPLOYEE_NAME);
    await apiSupervisorApprove(request, recordId);
    await apiHRReject(request, recordId);

    // 4) UI 展示「已拒绝」
    await expectRowInStatus(page, REJECT_EMPLOYEE_NAME, "已拒绝");

    // 5) API 验证员工仍为试用期（trial）
    expect(await fetchEmploymentStatus(request, adminToken, REJECT_EMPLOYEE_NAME)).toBe("trial");

    // 6) DB 验证离职办理待办存在（business_type=regularization_rejection，同事务创建）
    const todoCount = queryDB(
      `SELECT COUNT(*) FROM work_todos WHERE user_id = ${INITIATOR_USER_ID} AND business_type = 'regularization_rejection' AND business_id = ${recordId} AND status = 'pending';`,
    );
    expect(todoCount).toBe("1");
  });

  test("viewer：无「转正管理」入口（无 employee.edit 权限）", async ({ page }) => {
    await login(page, ACCOUNTS.viewer.username, ACCOUNTS.viewer.password);

    if (await isNewShell(page)) {
      // 新壳：展开「员工管理」分组，断言无「转正管理」入口（无 employee.edit 权限）
      await expandEmployeeGroup(page);
      await expect(page.getByRole("button", { name: "转正管理", exact: true })).toHaveCount(0);
    } else {
      // 旧壳：转正管理独立页面与入口仅存在于新壳，收敛断言「无转正管理新入口」
      await expect(page.getByRole("button", { name: "转正管理", exact: true })).toHaveCount(0);
    }
  });
});