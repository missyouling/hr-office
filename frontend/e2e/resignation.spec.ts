import { test, expect, type Locator, type Page } from "@playwright/test";
import { ACCOUNTS, login } from "./helpers/auth";

/**
 * P12.3.1.3 离职管理 E2E
 *
 * 前置依赖（由后端 seed-e2e 幂等准备）：
 * - 稳定名称「E2E离职测试员工」的真实在职员工，属于 admin 账号（GET /employees 按 user_id 隔离）。
 *
 * 壳层覆盖（离职独立页面与入口固定新壳提供）：
 * - 新壳：侧边栏「员工管理」分组下存在独立「离职管理」入口（employee.edit 权限）。
 *   覆盖 admin 完整离职流程（办理离职→列表验证→下载证明→恢复在职→返回），
 *   以及 viewer 无 employee.edit 时入口必须隐藏的权限边界。
 * - 旧壳：侧边栏无「离职管理」入口（产品现状）。
 *   - admin 流程无法执行 → 显式 test.skip 并注明阻塞；
 *   - viewer 不尝试定位 admin 员工行（user_id 隔离下 viewer 看不到），仅收敛断言「无离职管理新入口」。
 *
 * admin 流程（新壳）：
 * 1. 从「离职管理」入口打开页面；
 * 2. 办理离职：选择种子员工、填离职日期/原因、上传内存生成的最小 PDF 证明；
 * 3. 验证离职列表出现该员工（含证明文件名）；
 * 4. 通过 waitForEvent('download') 下载证明并验证内容为最小 PDF；
 * 5. 恢复在职并确认列表移除（状态回滚）；
 * 6. 返回工作台确认导航正常。
 *
 * viewer：验证无办理离职入口（新壳无 employee.edit 时「离职管理」隐藏；旧壳无「离职管理」新入口）。
 */

const SEED_EMPLOYEE_NAME = "E2E离职测试员工";
const RESIGN_REASON = "E2E 自动化测试离职";
const PROOF_FILENAME = "e2e-resign-proof.pdf";

/** 生成最小合法 PDF 内容（内存 buffer，不落仓库）。 */
function minimalPdfBuffer(): Buffer {
  const content = [
    "%PDF-1.4",
    "1 0 obj<</Type/Catalog/Pages 2 0 R>>endobj",
    "2 0 obj<</Type/Pages/Kids[3 0 R]/Count 1>>endobj",
    "3 0 obj<</Type/Page/Parent 2 0 R/MediaBox[0 0 200 200]>>endobj",
    "xref",
    "0 4",
    "0000000000 65535 f ",
    "0000000009 00000 n ",
    "0000000052 00000 n ",
    "0000000101 00000 n ",
    "trailer<</Size 4/Root 1 0 R>>",
    "startxref",
    "149",
    "%%EOF",
  ].join("\n");
  return Buffer.from(content, "utf-8");
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

/** 新壳：从侧边栏「员工管理」分组进入「离职管理」页面。 */
async function openResignationManagement(page: Page) {
  await expandEmployeeGroup(page);
  await page.getByRole("button", { name: "离职管理", exact: true }).click();
  await expect(page.getByRole("heading", { name: "离职管理", exact: true })).toBeVisible();
}

/** 在办理离职对话框中按精确种子名选中员工 option。 */
async function selectSeedEmployee(dialog: Locator) {
  const select = dialog.locator("#resignation-employee");
  const option = select.locator("option").filter({ hasText: SEED_EMPLOYEE_NAME }).first();
  await expect(option).toHaveCount(1);
  const value = await option.getAttribute("value");
  if (!value) throw new Error("未找到种子员工的 option value");
  await select.selectOption(value);
}

test.describe("P12.3.1.3 离职管理 E2E", () => {
  // 离职流程会变更种子员工状态，串行执行避免并发互相影响。
  test.describe.configure({ mode: "serial" });

  test("admin：从离职管理入口完成带最小 PDF 证明的离职，验证列表/下载证明，恢复在职并返回", async ({ page }) => {
    await login(page, ACCOUNTS.admin.username, ACCOUNTS.admin.password);

    // 旧壳侧边栏无「离职管理」入口（产品现状），新独立入口流程无法执行 → 阻塞，跳过并注明。
    test.skip(!(await isNewShell(page)), "旧壳侧边栏无「离职管理」入口（产品现状），新独立入口流程无法执行，阻塞待主协调者决策");

    await openResignationManagement(page);

    // 1) 办理离职（带最小 PDF 证明）
    await page.getByRole("button", { name: "办理离职", exact: true }).click();
    const dialog = page.getByRole("dialog", { name: "办理离职" });
    await expect(dialog).toBeVisible();
    await selectSeedEmployee(dialog);
    const today = new Date().toISOString().slice(0, 10);
    await dialog.locator("#resignation-date").fill(today);
    await dialog.locator("#resignation-reasons").fill(RESIGN_REASON);
    await dialog.locator("#resignation-proof").setInputFiles({
      name: PROOF_FILENAME,
      mimeType: "application/pdf",
      buffer: minimalPdfBuffer(),
    });
    await dialog.getByRole("button", { name: "确认离职", exact: true }).click();

    // 2) 验证离职列表出现种子员工（含证明文件名）
    const resignedRow = page.locator("tr").filter({ hasText: SEED_EMPLOYEE_NAME });
    await expect(resignedRow).toHaveCount(1, { timeout: 15_000 });
    await expect(resignedRow.getByText(PROOF_FILENAME, { exact: true })).toBeVisible();

    // 3) 下载证明（waitForEvent('download')）：
    //    resignation-management 用 fetch→blob→URL.createObjectURL→a[download] 触发下载，
    //    Playwright 对 blob 下载的 suggestedFilename 固定为 "download"，不可作为断言依据。
    //    改为验证下载事件已发生且下载内容为最小 PDF（读取流前几个字节断言 %PDF-）。
    const downloadPromise = page.waitForEvent("download");
    await resignedRow.getByRole("button", { name: "下载", exact: true }).click();
    const download = await downloadPromise;
    const stream = await download.createReadStream();
    if (!stream) throw new Error("下载内容不可读");
    const chunks: Buffer[] = [];
    for await (const chunk of stream) {
      chunks.push(chunk as Buffer);
    }
    const pdfHead = Buffer.concat(chunks).subarray(0, 5).toString("utf-8");
    expect(pdfHead).toBe("%PDF-");

    // 4) 恢复在职并确认列表移除（状态回滚）
    await resignedRow.getByRole("button", { name: "恢复在职", exact: true }).click();
    const alertDialog = page.getByRole("alertdialog");
    await expect(alertDialog.getByText("确认恢复在职", { exact: true })).toBeVisible();
    await alertDialog.getByRole("button", { name: "确认恢复", exact: true }).click();
    await expect(page.locator("tr").filter({ hasText: SEED_EMPLOYEE_NAME })).toHaveCount(0, { timeout: 15_000 });

    // 5) 返回工作台确认导航正常
    await page.getByRole("button", { name: "工作台", exact: true }).click();
    await expect(page.getByRole("heading", { name: "离职管理", exact: true })).toHaveCount(0);
    await expect(page.getByRole("button", { name: "工作台", exact: true })).toHaveAttribute("data-active", "true");
  });

  test("viewer：无办理离职操作或入口", async ({ page }) => {
    await login(page, ACCOUNTS.viewer.username, ACCOUNTS.viewer.password);

    if (await isNewShell(page)) {
      // 新壳：展开「员工管理」分组，断言无「离职管理」入口（无 employee.edit 权限）
      await expandEmployeeGroup(page);
      await expect(page.getByRole("button", { name: "离职管理", exact: true })).toHaveCount(0);
    } else {
      // 旧壳：离职独立页面与入口仅存在于新壳；且种子员工属于 admin（GET /employees 按 user_id
      // 隔离，viewer 看不到 admin 员工），因此不尝试定位员工行，仅收敛断言「无离职管理新入口」。
      await expect(page.getByRole("button", { name: "离职管理", exact: true })).toHaveCount(0);
    }
  });
});