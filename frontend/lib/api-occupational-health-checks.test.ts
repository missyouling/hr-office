import { beforeEach, describe, expect, test, vi } from "vitest";

import {
  completeOccupationalHealthCheck, createOccupationalHealthCheck, deleteOccupationalHealthCheck,
  fetchOccupationalHealthChecks, getOccupationalHealthCheck, updateOccupationalHealthCheck,
  voidOccupationalHealthCheck, type OccupationalHealthCheckPayload,
} from "@/lib/api-occupational-health-checks";

const mocks = vi.hoisted(() => ({ request: vi.fn() }));
vi.mock("@/lib/api", () => ({ request: mocks.request }));

const payload: OccupationalHealthCheckPayload = {
  employee_id: 2,
  check_date: "2026-09-01",
  medical_institution: "市职业病防治院",
  check_category: "上岗前检查",
  check_conclusion: "未见异常",
  next_check_date: "2027-09-01",
  remarks: "复查时携带既往报告",
};

describe("职业健康检查 REST 客户端", () => {
  beforeEach(() => mocks.request.mockReset());

  test("列表支持状态筛选并可读取详情", async () => {
    await fetchOccupationalHealthChecks(); await fetchOccupationalHealthChecks("completed"); await getOccupationalHealthCheck(3);
    expect(mocks.request).toHaveBeenNthCalledWith(1, "/occupational-health-checks");
    expect(mocks.request).toHaveBeenNthCalledWith(2, "/occupational-health-checks?status=completed");
    expect(mocks.request).toHaveBeenNthCalledWith(3, "/occupational-health-checks/3");
  });

  test("创建、编辑和删除严格使用约定接口", async () => {
    await createOccupationalHealthCheck(payload); await updateOccupationalHealthCheck(3, payload); await deleteOccupationalHealthCheck(3);
    expect(mocks.request).toHaveBeenNthCalledWith(1, "/occupational-health-checks", expect.objectContaining({ method: "POST", body: JSON.stringify(payload) }));
    expect(mocks.request).toHaveBeenNthCalledWith(2, "/occupational-health-checks/3", expect.objectContaining({ method: "PUT", body: JSON.stringify(payload) }));
    expect(mocks.request).toHaveBeenNthCalledWith(3, "/occupational-health-checks/3", { method: "DELETE" });
  });

  test("完成与带原因作废调用限定状态接口", async () => {
    await completeOccupationalHealthCheck(3); await voidOccupationalHealthCheck(3, "员工信息录入错误");
    expect(mocks.request).toHaveBeenNthCalledWith(1, "/occupational-health-checks/3/complete", expect.objectContaining({ method: "POST", body: undefined }));
    expect(mocks.request).toHaveBeenNthCalledWith(2, "/occupational-health-checks/3/void", expect.objectContaining({ method: "POST", body: JSON.stringify({ reason: "员工信息录入错误" }) }));
  });
});
