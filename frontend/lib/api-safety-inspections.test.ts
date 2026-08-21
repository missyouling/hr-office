import { beforeEach, describe, expect, test, vi } from "vitest";

import { completeSafetyInspection, createSafetyInspection, deleteSafetyInspection, fetchSafetyInspections, getSafetyInspection, updateSafetyInspection, voidSafetyInspection, type SafetyInspectionPayload } from "@/lib/api-safety-inspections";

const mocks = vi.hoisted(() => ({ request: vi.fn() }));
vi.mock("@/lib/api", () => ({ request: mocks.request }));
const payload: SafetyInspectionPayload = { inspection_type: "routine", inspection_date: "2026-09-01", location: "仓库", responsible_person: "张三", issue_description: "通道堆放物料", rectification_requirement: "立即清理并保持畅通" };

describe("安全检查 REST 客户端", () => {
  beforeEach(() => mocks.request.mockReset());
  test("列表按可选状态请求路径并支持详情", async () => { await fetchSafetyInspections(); await fetchSafetyInspections("completed"); await getSafetyInspection(3); expect(mocks.request).toHaveBeenNthCalledWith(1, "/safety-inspections"); expect(mocks.request).toHaveBeenNthCalledWith(2, "/safety-inspections?status=completed"); expect(mocks.request).toHaveBeenNthCalledWith(3, "/safety-inspections/3"); });
  test("创建、编辑和删除使用约定路径与方法", async () => { await createSafetyInspection(payload); await updateSafetyInspection(3, payload); await deleteSafetyInspection(3); expect(mocks.request).toHaveBeenNthCalledWith(1, "/safety-inspections", expect.objectContaining({ method: "POST", body: JSON.stringify(payload) })); expect(mocks.request).toHaveBeenNthCalledWith(2, "/safety-inspections/3", expect.objectContaining({ method: "PUT", body: JSON.stringify(payload) })); expect(mocks.request).toHaveBeenNthCalledWith(3, "/safety-inspections/3", { method: "DELETE" }); });
  test("完成与原因作废调用限定状态接口", async () => { await completeSafetyInspection(3); await voidSafetyInspection(3, "检查计划取消"); expect(mocks.request).toHaveBeenNthCalledWith(1, "/safety-inspections/3/complete", expect.objectContaining({ method: "POST", body: undefined })); expect(mocks.request).toHaveBeenNthCalledWith(2, "/safety-inspections/3/void", expect.objectContaining({ method: "POST", body: JSON.stringify({ reason: "检查计划取消" }) })); });
});
