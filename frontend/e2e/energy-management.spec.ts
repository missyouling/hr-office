import { expect, test, type APIRequestContext, type Page } from "@playwright/test";

import { ACCOUNTS, login } from "./helpers/auth";

/**
 * P12.3.13 能耗管理 E2E（最小只读水、电统计）
 *
 * 覆盖验收：
 * 1. admin 强制新壳：经「日常事务」分组进入「能耗管理」，页面标题可见，
 *    且无燃气文案、无任何新增/编辑/删除/保存等写操作入口；
 * 2. admin 通过既有受认证宿舍 API 准备独立园区/楼栋/房间与水电抄表数据
 *    （charge_details 含 electric/water/gas），页面显示电/水单位（kWh、m³）
 *    与金额（¥），点击楼栋行下钻房间明细；API 核验汇总仅统计电/水、忽略燃气；
 * 3. viewer 保留 dormitory.view：侧栏可见「能耗管理」入口并可进入只读页面，
 *    直接调用能耗汇总 GET 返回 200（只读验证，不调用任何写入接口）。
 *
 * 运行前提：后端已按 PostgreSQL 完整环境变量启动并监听 :8080；
 * seed-e2e 已初始化 4 个固定账号（admin 全量权限含 dormitory.view/create/edit/delete；
 * viewer 含 dormitory.view）。
 */

const API_BASE = "http://127.0.0.1:8080/api";

// 唯一测试标识：时间戳后缀，保证每次运行创建的数据唯一、可精确核验
const UNIQUE_SUFFIX = Date.now();
const SITE_NAME = `E2E能耗园区-${UNIQUE_SUFFIX}`;
const BUILDING_NAME = `E2E能耗楼栋-${UNIQUE_SUFFIX}`;
const ROOM_NUMBER = `E2E-${UNIQUE_SUFFIX}`;
// 页面默认月份与后端 month 筛选均按 UTC 自然月（YYYY-MM），抄表日期落在当前月内
const CURRENT_MONTH = new Date().toISOString().slice(0, 7);
const METER_DATE = `${CURRENT_MONTH}-05`;
// 抄表明细：电 100/50、水 10/20、燃气 5/3（燃气应被汇总忽略）
const ELECTRIC_USAGE = 100;
const ELECTRIC_AMOUNT = 50;
const WATER_USAGE = 10;
const WATER_AMOUNT = 20;
const GAS_USAGE = 5;
const GAS_AMOUNT = 3;

/** 固定新壳页面渲染 [data-shell="new"] 标记（app/page.tsx NewShell 包装器） */
async function isNewShell(page: Page): Promise<boolean> {
  return (await page.locator('[data-shell="new"]').count()) > 0;
}

/** 展开「日常事务」分组（新壳侧栏分组默认折叠，点击切换展开） */
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

/** 通过既有受认证宿舍 API 准备独立园区/楼栋/房间与水电抄表数据，返回各实体 ID */
async function createEnergyFixture(request: APIRequestContext, token: string) {
  // 1. 园区
  const siteResponse = await request.post(`${API_BASE}/dormitories/sites`, {
    headers: { Authorization: `Bearer ${token}` },
    data: { name: SITE_NAME },
  });
  expect(siteResponse.status()).toBe(201);
  const site = (await siteResponse.json()) as { id: number };

  // 2. 楼栋
  const buildingResponse = await request.post(`${API_BASE}/dormitories/buildings`, {
    headers: { Authorization: `Bearer ${token}` },
    data: { site_id: site.id, name: BUILDING_NAME, floors: 1 },
  });
  expect(buildingResponse.status()).toBe(201);
  const building = (await buildingResponse.json()) as { id: number };

  // 3. 房间
  const roomResponse = await request.post(`${API_BASE}/dormitories/rooms`, {
    headers: { Authorization: `Bearer ${token}` },
    data: { site_id: site.id, building_id: building.id, room_number: ROOM_NUMBER, status: "available" },
  });
  expect(roomResponse.status()).toBe(201);
  const room = (await roomResponse.json()) as { id: number };

  // 4. 水电抄表（含燃气明细，用于证明汇总忽略燃气）
  const readingResponse = await request.post(`${API_BASE}/dormitories/meter-readings`, {
    headers: { Authorization: `Bearer ${token}` },
    data: {
      room_id: room.id,
      meter_date: METER_DATE,
      billing_start: METER_DATE,
      billing_end: METER_DATE,
      charge_details: [
        { key: "electric", usage: ELECTRIC_USAGE, amount: ELECTRIC_AMOUNT },
        { key: "water", usage: WATER_USAGE, amount: WATER_AMOUNT },
        { key: "gas", usage: GAS_USAGE, amount: GAS_AMOUNT },
      ],
    },
  });
  expect(readingResponse.status()).toBe(201);

  return { siteId: site.id, buildingId: building.id, roomId: room.id };
}

test.describe("P12.3.13 能耗管理 E2E", () => {
  // 两个 admin 用例会写入同一认证账号的会话状态，串行执行避免并发登录竞争。
  test.describe.configure({ mode: "serial" });

  test("admin（新壳）：日常事务进入能耗管理，页面无燃气及写操作入口", async ({ page }) => {
    await login(page, ACCOUNTS.admin.username, ACCOUNTS.admin.password);
    test.skip(!(await isNewShell(page)), "旧壳无能耗管理入口，无法执行新壳验收流程");

    // 侧栏「日常事务」分组下可见「能耗管理」入口
    await expandDailyAffairsGroup(page);
    await expect(page.getByRole("button", { name: "能耗管理", exact: true })).toBeVisible();

    // 进入能耗管理真实页面，确认真实标题可见
    await page.getByRole("button", { name: "能耗管理", exact: true }).click();
    await expect(page.getByRole("heading", { name: "能耗管理", exact: true })).toBeVisible({ timeout: 15_000 });

    // 只读页面：无燃气文案、无任何写操作入口（新增/编辑/删除/保存）
    await expect(page.getByText("燃气", { exact: false })).toHaveCount(0);
    await expect(page.getByRole("button", { name: /新增|编辑|删除|保存/ })).toHaveCount(0);
  });

  test("admin：API 准备独立楼栋/房间/水电抄表，页面显示单位金额与房间下钻，汇总忽略燃气", async ({ page, request }) => {
    // 1. 通过既有受认证宿舍 API 准备独立数据（园区→楼栋→房间→抄表）
    const token = await apiLogin(request, ACCOUNTS.admin.username, ACCOUNTS.admin.password);
    const { buildingId, roomId } = await createEnergyFixture(request, token);

    // 2. API 核验能耗汇总：仅统计电/水，忽略燃气，按楼栋/房间聚合
    const summaryResponse = await request.get(`${API_BASE}/dormitories/energy/summary?month=${CURRENT_MONTH}&building_id=${buildingId}`, {
      headers: { Authorization: `Bearer ${token}` },
    });
    expect(summaryResponse.status()).toBe(200);
    const summary = (await summaryResponse.json()) as {
      overall: {
        electric: { usage: number; amount: number };
        water: { usage: number; amount: number };
        total_amount: number;
      };
      by_building: Array<{ building_id: number; building_name: string }>;
      rooms: Array<{ room_id: number; room_number: string }>;
    };
    expect(summary.overall.electric.usage).toBe(ELECTRIC_USAGE);
    expect(summary.overall.electric.amount).toBe(ELECTRIC_AMOUNT);
    expect(summary.overall.water.usage).toBe(WATER_USAGE);
    expect(summary.overall.water.amount).toBe(WATER_AMOUNT);
    expect(summary.overall.total_amount).toBe(ELECTRIC_AMOUNT + WATER_AMOUNT);
    // 汇总响应无燃气字段（gas 明细被忽略）
    expect(summary.overall).not.toHaveProperty("gas");
    const building = summary.by_building.find((item) => item.building_id === buildingId);
    expect(building).toBeTruthy();
    expect(building!.building_name).toBe(BUILDING_NAME);
    const room = summary.rooms.find((item) => item.room_id === roomId);
    expect(room).toBeTruthy();
    expect(room!.room_number).toBe(ROOM_NUMBER);

    // 3. 页面进入能耗管理，断言电/水单位与金额、楼栋汇总与房间下钻
    await login(page, ACCOUNTS.admin.username, ACCOUNTS.admin.password);
    test.skip(!(await isNewShell(page)), "旧壳无能耗管理入口，无法执行新壳验收流程");
    await expandDailyAffairsGroup(page);
    await page.getByRole("button", { name: "能耗管理", exact: true }).click();
    await expect(page.getByRole("heading", { name: "能耗管理", exact: true })).toBeVisible({ timeout: 15_000 });

    // 电/水单位与金额（卡片 + 楼栋汇总表）
    await expect(page.getByText("电力用量", { exact: true }).first()).toBeVisible({ timeout: 15_000 });
    await expect(page.getByText("用水量", { exact: true }).first()).toBeVisible();
    await expect(page.getByText("100 kWh", { exact: true }).first()).toBeVisible();
    await expect(page.getByText("10 m³", { exact: true }).first()).toBeVisible();
    await expect(page.getByText("¥70", { exact: true }).first()).toBeVisible();

    // 楼栋汇总表显示独立楼栋
    const buildingRow = page.locator("tbody tr").filter({ hasText: BUILDING_NAME });
    await expect(buildingRow).toBeVisible({ timeout: 15_000 });

    // 点击楼栋行 → 房间下钻
    await buildingRow.click();
    await expect(page.getByText(`${BUILDING_NAME} · 房间明细`, { exact: true })).toBeVisible({ timeout: 15_000 });
    await expect(page.getByText(ROOM_NUMBER, { exact: true })).toBeVisible();

    // 页面无燃气文案、无写操作入口
    await expect(page.getByText("燃气", { exact: false })).toHaveCount(0);
    await expect(page.getByRole("button", { name: /新增|编辑|删除|保存/ })).toHaveCount(0);
  });

  test("viewer（新壳）：保留 dormitory.view 可见能耗入口与只读页面，GET 汇总正常", async ({ page, request }) => {
    await login(page, ACCOUNTS.viewer.username, ACCOUNTS.viewer.password);
    test.skip(!(await isNewShell(page)), "旧壳无能耗管理入口，无法执行新壳验收流程");

    // viewer 有 dormitory.view：侧栏可见「能耗管理」入口并可进入只读页面
    await expandDailyAffairsGroup(page);
    await expect(page.getByRole("button", { name: "能耗管理", exact: true })).toBeVisible();
    await page.getByRole("button", { name: "能耗管理", exact: true }).click();
    await expect(page.getByRole("heading", { name: "能耗管理", exact: true })).toBeVisible({ timeout: 15_000 });

    // 直接调用能耗汇总 GET 正常（只读验证；viewer 不调用任何写入接口）
    const token = await apiLogin(request, ACCOUNTS.viewer.username, ACCOUNTS.viewer.password);
    const summaryResponse = await request.get(`${API_BASE}/dormitories/energy/summary`, {
      headers: { Authorization: `Bearer ${token}` },
    });
    expect(summaryResponse.status()).toBe(200);
  });
});
