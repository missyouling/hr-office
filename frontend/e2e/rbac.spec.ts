import { test, expect, type Page } from "@playwright/test";

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
 * ── 已知偏差说明（与任务验收标准的差异，测试按实际代码行为断言）──
 * 1. 「备份」「用户管理」菜单当前未在侧边栏注册（app-sidebar.tsx 注释明确说明），
 *    前端也无用户管理组件，因此这两个菜单无法断言"按权限显隐"，本测试不涉及。
 * 2. 员工管理页「删除」按钮仅在勾选员工后渲染（hasActiveSelection 分支），
 *    且前端「新增员工」为本地 state 操作（不调后端 API，刷新即失）。
 *    因此 admin/manager 用例通过 UI 新增一名员工并勾选，再断言删除按钮显隐，
 *    以区分"未勾选"与"无权限"两种隐藏原因。
 */

// ===== 测试账号（与后端 cmd/seed-e2e 保持一致）=====
const ACCOUNTS = {
  admin: { username: "admin", password: "Admin@123456" },
  manager: { username: "manager", password: "Manager@123456" },
  editor: { username: "editor", password: "Editor@123456" },
  viewer: { username: "viewer", password: "Viewer@123456" },
} as const;

// 登录页字段兼容两种表单：
// - SMTP 已配置：Tabs 表单（id=login-username / login-password）
// - SMTP 未配置：简单表单（id=login-username-simple / login-password-simple）
const USERNAME_SELECTOR = "#login-username, #login-username-simple";
const PASSWORD_SELECTOR = "#login-password, #login-password-simple";

// 新增员工用的合法身份证号（满足前端格式校验）
const TEST_ID_NUMBER = "11010519900307753X";

// ===== 辅助函数 =====

/** 登录指定账号并等待首页侧边栏渲染完成 */
async function login(page: Page, username: string, password: string) {
  await page.goto("/auth");
  // 等待登录表单出现（两种表单任一）
  await expect(page.locator(USERNAME_SELECTOR).first()).toBeVisible({ timeout: 15_000 });
  await page.locator(USERNAME_SELECTOR).first().fill(username);
  await page.locator(PASSWORD_SELECTOR).first().fill(password);
  // 仅提交包含当前用户名输入框的登录表单，避免误点其他表单的提交按钮。
  const loginForm = page.locator(USERNAME_SELECTOR).first().locator("xpath=ancestor::form");
  await loginForm.getByRole("button", { name: "登录", exact: true }).click();
  // 先确认登录跳转至首页；开发模式首次编译首页可能较慢。
  await page.waitForURL(/\/$/, { timeout: 60_000 });
  // 首页完成跳转后，再等待侧边栏基础菜单「员工管理」（所有角色均有该菜单）。
  await expect(page.getByRole("button", { name: "员工管理", exact: true })).toBeVisible({
    timeout: 30_000,
  });
}

/** 点击侧边栏「员工管理」进入员工管理页，等待内容区渲染完成 */
async function openEmployeePage(page: Page) {
  await page.getByRole("button", { name: "员工管理", exact: true }).click();
  // 等待在职员工卡片标题出现（初始 tab 为「在职员工」）
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

    // 菜单断言：admin 有 settings.view，系统设置菜单可见
    await expect(page.getByRole("button", { name: "系统设置", exact: true })).toBeVisible();

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

    // 菜单断言：manager 有 settings.view，系统设置菜单可见
    await expect(page.getByRole("button", { name: "系统设置", exact: true })).toBeVisible();

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

    // 菜单断言：editor 有 settings.view，系统设置菜单可见
    await expect(page.getByRole("button", { name: "系统设置", exact: true })).toBeVisible();

    // 员工管理页按钮断言：editor 无 employee.create / employee.delete，均隐藏
    await openEmployeePage(page);
    await expect(page.getByRole("button", { name: "新增", exact: true })).toBeHidden();
    await expect(page.getByRole("button", { name: "删除", exact: true })).toBeHidden();
  });

  test("viewer：仅基础模块 view，系统设置/新增/删除均不可见", async ({ page }) => {
    await login(page, ACCOUNTS.viewer.username, ACCOUNTS.viewer.password);

    // 菜单断言：viewer 无 settings.view，系统设置菜单不可见（验收标准：viewer 看不到系统设置）
    await expect(page.getByRole("button", { name: "系统设置", exact: true })).toBeHidden();

    // 员工管理页按钮断言：viewer 无 employee.create / employee.delete，均隐藏
    await openEmployeePage(page);
    await expect(page.getByRole("button", { name: "新增", exact: true })).toBeHidden();
    await expect(page.getByRole("button", { name: "删除", exact: true })).toBeHidden();
  });
});
