import { test, expect, type Page } from "@playwright/test";
import { ACCOUNTS, login } from "./helpers/auth";

/**
 * P7.3 发票管理主流程 UI 验收 E2E
 *
 * 覆盖场景（避免依赖真实发票数据，仅用 route mocking 模拟空列表/空统计响应）：
 * 1. admin 登录后发票管理入口可见，进入「上传解析」，断言批量 PDF 上传工作台关键文案与控件；
 * 2. admin / manager 进入「归档管理」，断言归档统计卡片、筛选控件与 CSV 导出按钮；
 * 3. 归档管理入口权限对比：admin 可见，viewer 不可见（viewer 仅 invoice.view）。
 *
 * 说明：
 * - 登录请求（/api/auth/login 等认证接口）始终保持真实通过，仅拦截 /api/invoices* 发票数据接口；
 * - 通过 route mocking 返回空列表 / 空统计，保证用例不依赖后端真实发票数据。
 */

/** 模拟空发票数据：仅拦截发票列表/统计/导出接口，认证请求不受影响。 */
async function mockEmptyInvoiceData(page: Page) {
  await page.route("**/api/invoices*", async (route) => {
    const url = route.request().url();
    if (url.includes("/stats")) {
      // 归档统计：总数与各状态均为 0
      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify({ total: 0, by_status: {} }),
      });
    } else if (url.includes("/export")) {
      // CSV 导出：仅返回表头，避免真实下载
      await route.fulfill({ status: 200, contentType: "text/csv", body: "票号,归档状态\n" });
    } else {
      // 发票列表：空数据
      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify({ items: [], total: 0 }),
      });
    }
  });
}

/** 从侧边栏进入「日常事务」并打开「发票管理」页面。 */
async function openInvoiceManagement(page: Page) {
  await page.getByRole("button", { name: "日常事务", exact: true }).click();
  await expect(page.getByRole("heading", { name: "日常事务", exact: true })).toBeVisible();

  // 「发票管理」入口是可点击卡片，内部标题仅用于展示，不承担点击语义。
  const invoiceModule = page.locator("div.cursor-pointer").filter({ hasText: "发票管理" }).first();
  await expect(invoiceModule).toBeVisible();
  await invoiceModule.click();
  await expect(page.getByRole("heading", { name: "发票管理", exact: true })).toBeVisible();
}

test.describe("P7.3 发票管理主流程 UI 验收", () => {
  // 固定 E2E 账号不能并发登录，避免登录态或后端会话互相影响。
  test.describe.configure({ mode: "serial" });

  test("admin：发票管理入口可见，上传解析工作台关键文案与批量 PDF 控件齐全", async ({ page }) => {
    await login(page, ACCOUNTS.admin.username, ACCOUNTS.admin.password);
    await mockEmptyInvoiceData(page);

    // 1) 发票管理入口可见
    await openInvoiceManagement(page);

    // 2) 进入「上传解析」并断言批量 PDF 上传工作台
    await page.getByRole("tab", { name: "上传解析", exact: true }).click();
    await expect(page.getByText("发票上传与解析", { exact: true })).toBeVisible();
    // 关键文案：拖拽提示 + 批量/大小限制
    await expect(page.getByText("拖拽 PDF 文件到此处，或点击选择文件", { exact: true })).toBeVisible();
    await expect(page.getByText("最多 50 份 / 单文件 ≤ 20MB / 仅支持 PDF", { exact: true })).toBeVisible();
    // 关键控件：上传区（拖拽/点击）与隐藏的批量 PDF 文件输入
    await expect(page.getByRole("button", { name: "选择或拖拽 PDF 文件上传" })).toBeVisible();
    const fileInput = page.locator('input[type="file"]');
    await expect(fileInput).toBeAttached();
    await expect(fileInput).toHaveAttribute("accept", ".pdf");
    await expect(fileInput).toHaveAttribute("multiple", "");
  });

  for (const account of [ACCOUNTS.admin, ACCOUNTS.manager]) {
    test(`归档管理：${account.username} 可见统计卡片、筛选控件与 CSV 导出按钮`, async ({ page }) => {
      await login(page, account.username, account.password);
      await mockEmptyInvoiceData(page);
      await openInvoiceManagement(page);

      // 进入「归档管理」
      await page.getByRole("tab", { name: "归档管理", exact: true }).click();
      // 归档统计卡片（总数 + 三个状态）
      for (const label of ["归档总数", "待确认", "已确认", "已作废"]) {
        await expect(page.getByText(label, { exact: true })).toBeVisible();
      }
      // CSV 导出按钮
      await expect(page.getByRole("button", { name: "导出 CSV", exact: true })).toBeVisible();
      // 筛选控件：归档状态、全部来源下拉框 + 关键字搜索框
      await expect(page.getByRole("combobox").filter({ hasText: "归档状态" })).toBeVisible();
      await expect(page.getByRole("combobox").filter({ hasText: "全部来源" })).toBeVisible();
      await expect(page.getByPlaceholder("搜索票号、用途...")).toBeVisible();
      // route mock 生效：空列表兜底文案可见
      await expect(page.getByText("暂无符合条件的归档发票", { exact: true })).toBeVisible();
    });
  }

  test("归档管理入口权限：admin 可见，viewer 不可见", async ({ browser }) => {
    const adminPage = await browser.newPage();
    const viewerPage = await browser.newPage();

    // admin：可看到归档管理入口
    await login(adminPage, ACCOUNTS.admin.username, ACCOUNTS.admin.password);
    await mockEmptyInvoiceData(adminPage);
    await openInvoiceManagement(adminPage);
    await expect(adminPage.getByRole("tab", { name: "归档管理", exact: true })).toBeVisible();

    // viewer：无归档管理入口（viewer 不在 canManage 角色范围内）
    await login(viewerPage, ACCOUNTS.viewer.username, ACCOUNTS.viewer.password);
    await mockEmptyInvoiceData(viewerPage);
    await openInvoiceManagement(viewerPage);
    await expect(viewerPage.getByRole("tab", { name: "归档管理", exact: true })).toHaveCount(0);
    // viewer 仅 invoice.view，无 invoice.create，「上传解析」同样不可见
    await expect(viewerPage.getByRole("tab", { name: "上传解析", exact: true })).toHaveCount(0);
    // 发票列表入口仍可见（viewer 有 invoice.view）
    await expect(viewerPage.getByRole("tab", { name: "发票列表", exact: true })).toBeVisible();
  });
});
