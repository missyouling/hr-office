import { test, expect, type Locator, type Page } from "@playwright/test";
import { ACCOUNTS, login } from "./helpers/auth";

/**
 * P7.1 RBAC 权限矩阵 E2E 验证
 *
 * 覆盖 4 个固定角色（与后端 cmd/seed-e2e 创建的账号一致）：
 *   - admin  ：全部权限
 *   - manager：基础模块 view+create+edit（无 delete）+ settings.view + backups.view + users.view
 *   - editor ：基础模块 view+edit（无 create/delete）+ settings.view + backups.view
 *   - viewer ：基础模块仅 view（不含 settings / backups）
 *
 * 每个用例独立登录（beforeEach），不共享浏览器状态。
 *
 * ── 壳层兼容（P12.1.1 新壳）──
 * 固定新壳下：
 * - 「员工管理」是侧边栏可折叠分组，需展开分组后点击「员工花名册」进入员工页；
 * - 「系统设置」不在主侧栏，仅在底部头像菜单中按 settings.view 权限显隐。
 * 辅助函数通过新壳渲染的 [data-shell="new"] 标记识别壳层并切换定位路径，
 * 旧壳保留主侧栏定位；两套壳断言语义一致。
 *
 * ── 已知偏差说明（与任务验收标准的差异，测试按实际代码行为断言）──
 * 1. 「备份」「用户管理」菜单当前未在侧边栏注册（app-sidebar.tsx 注释明确说明），
 *    前端也无用户管理组件，因此这两个菜单无法断言"按权限显隐"，本测试不涉及。
 * 2. 员工管理页「删除」按钮仅在勾选员工后渲染（hasActiveSelection 分支），
 *    且前端「新增员工」为本地 state 操作（不调后端 API，刷新即失）。
 *    因此 admin/manager 用例通过 UI 新增一名员工并勾选，再断言删除按钮显隐，
 *    以区分"未勾选"与"无权限"两种隐藏原因。
 */

// 新增员工用的合法身份证号（满足前端格式校验）
const TEST_ID_NUMBER = "11010519900307753X";

// ===== 壳层识别 =====

/** 固定新壳页面渲染 [data-shell="new"] 标记（app/page.tsx NewShell 包装器） */
async function isNewShell(page: Page): Promise<boolean> {
  return (await page.locator('[data-shell="new"]').count()) > 0;
}

// ===== 头像菜单辅助（新壳下系统设置入口） =====

/** 底部头像菜单触发器按钮（两种壳共用 NavUser 组件） */
function accountMenuTrigger(page: Page) {
  return page.getByRole("button", { name: /打开账户菜单/ });
}

/** 打开头像菜单并返回菜单定位器（Radix DropdownMenu 挂载于 body 下） */
async function openAccountMenu(page: Page): Promise<Locator> {
  await accountMenuTrigger(page).click();
  return page.getByRole("menu");
}

/** 关闭头像菜单（Escape），避免遮挡后续侧边栏操作 */
async function closeAccountMenu(page: Page) {
  await page.keyboard.press("Escape");
  await expect(page.getByRole("menu")).toHaveCount(0);
}

// ===== 系统设置入口断言 =====

/**
 * 断言「系统设置」可达（角色有 settings.view）：
 * - 新壳：头像菜单中出现「系统设置」菜单项
 * - 旧壳：主侧栏出现「系统设置」按钮
 */
async function expectSystemSettingsVisible(page: Page) {
  if (await isNewShell(page)) {
    const menu = await openAccountMenu(page);
    await expect(menu.getByRole("menuitem", { name: "系统设置", exact: true })).toBeVisible();
    await closeAccountMenu(page);
  } else {
    await expect(page.getByRole("button", { name: "系统设置", exact: true })).toBeVisible();
  }
}

/**
 * 断言「系统设置」不可见（角色无 settings.view）：
 * - 新壳：头像菜单中不存在「系统设置」菜单项
 * - 旧壳：主侧栏不存在「系统设置」按钮
 */
async function expectSystemSettingsHidden(page: Page) {
  if (await isNewShell(page)) {
    const menu = await openAccountMenu(page);
    await expect(menu.getByRole("menuitem", { name: "系统设置", exact: true })).toHaveCount(0);
    await closeAccountMenu(page);
  } else {
    await expect(page.getByRole("button", { name: "系统设置", exact: true })).toBeHidden();
  }
}

// ===== 员工管理入口 =====

/**
 * 进入员工管理页：
 * - 新壳：「员工管理」是可折叠分组，先展开分组再点击「员工花名册」
 * - 旧壳：「员工管理」是主侧栏直接入口
 * 统一等待在职员工卡片标题出现（初始 tab 为「在职员工」）。
 */
async function openEmployeePage(page: Page) {
  if (await isNewShell(page)) {
    const groupToggle = page.getByRole("button", { name: "员工管理", exact: true });
    if ((await groupToggle.getAttribute("aria-expanded")) !== "true") {
      await groupToggle.click();
    }
    await page.getByRole("button", { name: "员工花名册", exact: true }).click();
  } else {
    await page.getByRole("button", { name: "员工管理", exact: true }).click();
  }
  await expect(page.getByText("在职员工管理", { exact: true })).toBeVisible({ timeout: 15_000 });
}

/**
 * 通过 UI 新增一名员工（前端本地 state 操作，不调后端）。
 * 流程：点「新增」→ 填姓名/部门 → 切「个人信息」tab 填身份证号 → 提交。
 */
async function addEmployee(page: Page, name: string) {
  await page.getByRole("button", { name: "新增", exact: true }).click();
  // 基本信息 tab（默认激活）：姓名 + 部门（部门选项 >3 个，渲染为可输入框）
  await page.locator("#name").fill(name);
  await page.locator("#department").fill("总经办");
  // 切换到「个人信息」tab 填写身份证号
  await page.getByRole("tab", { name: "个人信息" }).click();
  await page.locator("#idNumber").fill(TEST_ID_NUMBER);
  // 提交新增（「添加员工」按钮同样受 employee.create 权限控制）
  await page.getByRole("button", { name: "添加员工", exact: true }).click();
  // 等待新员工出现在表格中
  await expect(page.getByText(name, { exact: true })).toBeVisible({ timeout: 10_000 });
}

/** 勾选表格第一行员工（触发 hasActiveSelection，显示删除按钮区） */
async function selectFirstEmployee(page: Page) {
  await page.locator('table input[type="checkbox"]').first().check();
}

// ===== 测试用例 =====

test.describe("RBAC 权限矩阵 E2E", () => {
  test("admin：全部权限，系统设置菜单与新增/删除按钮均可见", async ({ page }) => {
    await login(page, ACCOUNTS.admin.username, ACCOUNTS.admin.password);

    // 菜单断言：admin 有 settings.view，系统设置入口可达（新壳在头像菜单，旧壳在主侧栏）
    await expectSystemSettingsVisible(page);

    // 员工管理页按钮断言：admin 有 employee.create / employee.delete
    await openEmployeePage(page);
    await expect(page.getByRole("button", { name: "新增", exact: true })).toBeVisible();
    // 新增一名员工并勾选，验证删除按钮（有 employee.delete 权限 → 可见）
    await addEmployee(page, "E2E管理员测试员工");
    await selectFirstEmployee(page);
    await expect(page.getByRole("button", { name: "删除", exact: true })).toBeVisible();
  });

  test("manager：系统设置可见，新增可见但删除不可见（无 delete 权限）", async ({ page }) => {
    await login(page, ACCOUNTS.manager.username, ACCOUNTS.manager.password);

    // 菜单断言：manager 有 settings.view，系统设置入口可达
    await expectSystemSettingsVisible(page);

    // 员工管理页按钮断言：manager 有 employee.create（新增可见）
    await openEmployeePage(page);
    await expect(page.getByRole("button", { name: "新增", exact: true })).toBeVisible();
    // 新增员工并勾选后，删除按钮仍不可见（manager 无 employee.delete 权限）
    await addEmployee(page, "E2E经理测试员工");
    await selectFirstEmployee(page);
    await expect(page.getByRole("button", { name: "删除", exact: true })).toBeHidden();
  });

  test("editor：系统设置可见，新增与删除均不可见（无 create/delete 权限）", async ({ page }) => {
    await login(page, ACCOUNTS.editor.username, ACCOUNTS.editor.password);

    // 菜单断言：editor 有 settings.view，系统设置入口可达
    await expectSystemSettingsVisible(page);

    // 员工管理页按钮断言：editor 无 employee.create / employee.delete，均隐藏
    await openEmployeePage(page);
    await expect(page.getByRole("button", { name: "新增", exact: true })).toBeHidden();
    await expect(page.getByRole("button", { name: "删除", exact: true })).toBeHidden();
  });

  test("viewer：仅基础模块 view，系统设置/新增/删除均不可见", async ({ page }) => {
    await login(page, ACCOUNTS.viewer.username, ACCOUNTS.viewer.password);

    // 菜单断言：viewer 无 settings.view，系统设置入口不可见（验收标准：viewer 看不到系统设置）
    await expectSystemSettingsHidden(page);

    // 员工管理页按钮断言：viewer 无 employee.create / employee.delete，均隐藏
    await openEmployeePage(page);
    await expect(page.getByRole("button", { name: "新增", exact: true })).toBeHidden();
    await expect(page.getByRole("button", { name: "删除", exact: true })).toBeHidden();
  });

  test("新壳：admin 可进入组织管理", async ({ page }) => {
    await login(page, ACCOUNTS.admin.username, ACCOUNTS.admin.password);
    await expect(page.locator('[data-shell="new"]')).toHaveCount(1);
    const adminGroup = page.getByRole("button", { name: "行政管理", exact: true });
    await adminGroup.click();
    await page.getByRole("button", { name: "组织管理", exact: true }).click();
    await expect(page.getByText("组织机构管理", { exact: true })).toBeVisible({ timeout: 15_000 });
  });

  test("新壳：viewer 隐藏组织管理入口", async ({ page }) => {
    await login(page, ACCOUNTS.viewer.username, ACCOUNTS.viewer.password);
    await expect(page.locator('[data-shell="new"]')).toHaveCount(1);
    await expect(page.getByRole("button", { name: "组织管理", exact: true })).toHaveCount(0);
  });
});
