import { expect, test, type APIRequestContext, type Page } from "@playwright/test";

import { ACCOUNTS, login } from "./helpers/auth";

/**
 * P12 社保管理 E2E（归零范围：复用既有社保管理，无"社保业务"占位卡片）
 *
 * 覆盖验收：
 * 1. admin 强制新壳：侧栏可见并进入「社保管理」，真实页面标题可见，
 *    且页面/日常事务范围无「社保业务」占位卡片；
 * 2. admin 通过 API 创建一条唯一的低风险社保增员变更
 *    （payload 参考 backend/internal/api/insurance_rbac_test.go 的 insuranceIncreasePayload
 *    与 frontend/lib/api.ts 的 SocialInsuranceManualPayload），
 *    再 GET 列表核验归属（user_id 隔离）与记录字段；
 * 3. viewer 保留 insurance.view：能看社保管理入口及真实页面，
 *    但直接 POST /api/social-insurance/changes 得到 403（无 insurance.create）。
 *
 * 运行前提：后端已按 PostgreSQL 完整环境变量启动并监听 :8080；
 * seed-e2e 已初始化 4 个固定账号（viewer 含 insurance.view、不含 insurance.create）。
 */

const API_BASE = "http://127.0.0.1:8080/api";

// 唯一测试标识：时间戳后缀，保证每次运行创建的数据唯一、可精确核验
const UNIQUE_SUFFIX = Date.now();
const EMPLOYEE_NAME = `E2E社保增员-${UNIQUE_SUFFIX}`;
const VIEWER_EMPLOYEE_NAME = `E2E社保增员viewer-${UNIQUE_SUFFIX}`;
// 虚构测试身份证号（非真实个人信息），仅用于幂等定位与记录核验
const IDENTITY_NUMBER = "110101199001011234";
const EFFECTIVE_DATE = "2026-01-01";

/** 构造合法的 increase（增员）变更记录请求体，对齐 insurance_rbac_test.go 的 insuranceIncreasePayload */
function insuranceIncreasePayload(employeeName: string, identityNumber: string) {
  return {
    change_type: "increase",
    employee_name: employeeName,
    identity_number: identityNumber,
    effective_date: EFFECTIVE_DATE,
    template_values: {
      personalIdentity: identityNumber,
      householdType: "城镇",
      education: "本科",
      pensionStartDate: EFFECTIVE_DATE,
      baseSalary: "5000",
    },
  };
}

/** 固定新壳页面渲染 [data-shell="new"] 标记（app/page.tsx NewShell 包装器） */
async function isNewShell(page: Page): Promise<boolean> {
  return (await page.locator('[data-shell="new"]').count()) > 0;
}

/** 展开「行政管理」分组（新壳侧栏分组默认折叠，点击切换展开） */
async function expandAdminGroup(page: Page) {
  const groupToggle = page.getByRole("button", { name: "行政管理", exact: true });
  await expect(groupToggle).toBeVisible();
  if ((await groupToggle.getAttribute("aria-expanded")) !== "true") {
    await groupToggle.click();
  }
}

/** 展开「日常事务」分组（用于断言该范围无「社保业务」占位入口） */
async function expandDailyAffairsGroup(page: Page) {
  const groupToggle = page.getByRole("button", { name: "日常事务", exact: true });
  await expect(groupToggle).toBeVisible();
  if ((await groupToggle.getAttribute("aria-expanded")) !== "true") {
    await groupToggle.click();
  }
}

/** 通过 API 登录指定账号并返回 Bearer token */
async function apiLogin(request: APIRequestContext, username: string, password: string): Promise<string> {
  const response = await request.post(`${API_BASE}/auth/login`, {
    data: { username, password },
  });
  expect(response.ok()).toBeTruthy();
  return ((await response.json()) as { token: string }).token;
}

test.describe("P12 社保管理 E2E", () => {
  test("admin（新壳）：侧栏进入社保管理，真实标题可见且无社保业务占位卡片", async ({ page }) => {
    await login(page, ACCOUNTS.admin.username, ACCOUNTS.admin.password);
    test.skip(!(await isNewShell(page)), "旧壳无社保管理入口，无法执行新壳验收流程");

    // 侧栏「行政管理」分组下可见「社保管理」入口
    await expandAdminGroup(page);
    await expect(page.getByRole("button", { name: "社保管理", exact: true })).toBeVisible();
    // 侧栏无「社保业务」占位入口（占位卡片已移除）
    await expect(page.getByRole("button", { name: "社保业务", exact: true })).toHaveCount(0);

    // 进入社保管理真实页面，确认真实标题可见
    await page.getByRole("button", { name: "社保管理", exact: true }).click();
    await expect(page.getByRole("heading", { name: "社保管理", exact: true })).toBeVisible({ timeout: 15_000 });

    // 页面范围内无「社保业务」占位卡片
    await expect(page.getByText("社保业务", { exact: true })).toHaveCount(0);

    // 日常事务范围无「社保业务」占位入口
    await expandDailyAffairsGroup(page);
    await expect(page.getByRole("button", { name: "社保业务", exact: true })).toHaveCount(0);
  });

  test("admin：API 创建唯一低风险社保增员变更，GET 列表核验归属与记录", async ({ request }) => {
    const token = await apiLogin(request, ACCOUNTS.admin.username, ACCOUNTS.admin.password);
    const payload = insuranceIncreasePayload(EMPLOYEE_NAME, IDENTITY_NUMBER);

    // 创建一条唯一的低风险社保增员变更
    const createResponse = await request.post(`${API_BASE}/social-insurance/changes`, {
      headers: { Authorization: `Bearer ${token}` },
      data: payload,
    });
    expect(createResponse.status()).toBe(201);
    const created = (await createResponse.json()) as {
      id: number;
      change_type: string;
      employee_name: string;
      identity_number: string;
      effective_date: string;
    };
    expect(created.employee_name).toBe(EMPLOYEE_NAME);
    expect(created.change_type).toBe("increase");
    expect(created.identity_number).toBe(IDENTITY_NUMBER);

    // GET 列表核验：记录存在且归属当前用户（列表接口按 user_id 隔离，能查到即归属正确）
    const listResponse = await request.get(`${API_BASE}/social-insurance/changes`, {
      headers: { Authorization: `Bearer ${token}` },
    });
    expect(listResponse.ok()).toBeTruthy();
    const list = (await listResponse.json()) as {
      records: Array<{
        id: number;
        change_type: string;
        employee_name: string;
        identity_number: string;
        effective_date: string;
        template_values: Record<string, string>;
      }>;
    };
    const record = list.records.find((item) => item.id === created.id);
    expect(record).toBeTruthy();
    expect(record!.employee_name).toBe(EMPLOYEE_NAME);
    expect(record!.change_type).toBe("increase");
    expect(record!.identity_number).toBe(IDENTITY_NUMBER);
    expect(record!.effective_date).toBe(EFFECTIVE_DATE);
    expect(record!.template_values?.baseSalary).toBe("5000");
  });

  test("viewer（新壳）：保留 insurance.view 可看社保管理入口与真实页面，POST 变更 403", async ({ page, request }) => {
    await login(page, ACCOUNTS.viewer.username, ACCOUNTS.viewer.password);
    test.skip(!(await isNewShell(page)), "旧壳无社保管理入口，无法执行新壳验收流程");

    // viewer 有 insurance.view：侧栏可见「社保管理」入口并可进入真实页面
    await expandAdminGroup(page);
    await expect(page.getByRole("button", { name: "社保管理", exact: true })).toBeVisible();
    await page.getByRole("button", { name: "社保管理", exact: true }).click();
    await expect(page.getByRole("heading", { name: "社保管理", exact: true })).toBeVisible({ timeout: 15_000 });

    // viewer 无 insurance.create：直接 POST 变更必须 403
    const token = await apiLogin(request, ACCOUNTS.viewer.username, ACCOUNTS.viewer.password);
    const postResponse = await request.post(`${API_BASE}/social-insurance/changes`, {
      headers: { Authorization: `Bearer ${token}` },
      data: insuranceIncreasePayload(VIEWER_EMPLOYEE_NAME, IDENTITY_NUMBER),
    });
    expect(postResponse.status()).toBe(403);
  });
});