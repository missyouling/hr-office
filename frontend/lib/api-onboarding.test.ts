import { beforeEach, describe, expect, test, vi } from "vitest";

import {
  abandonOnboardingRecord, confirmOnboardingRecord, createOnboardingRecord,
  fetchOnboardingRecords, quickOnboardingRecord, restoreOnboardingRecord,
  updateOnboardingRecord, type OnboardingPayload,
} from "@/lib/api-onboarding";

const mocks = vi.hoisted(() => ({ request: vi.fn() }));
vi.mock("@/lib/api", () => ({ request: mocks.request }));

const payload: OnboardingPayload = {
  name: "张三", id_number: "110101199001011234", phone: "13800000000",
  department: "人事部", position: "专员", planned_hire_date: "2026-09-01",
  remarks: "备注", offer_id: "OFF-1", offer_source: "招聘网站",
};

describe("入职 REST 客户端", () => {
  beforeEach(() => mocks.request.mockReset());

  test("列表与状态筛选使用既定路径", async () => {
    await fetchOnboardingRecords();
    await fetchOnboardingRecords("pending");
    expect(mocks.request).toHaveBeenNthCalledWith(1, "/onboarding-records");
    expect(mocks.request).toHaveBeenNthCalledWith(2, "/onboarding-records?status=pending");
  });

  test("创建、编辑与快速入职发送正确请求", async () => {
    await createOnboardingRecord(payload);
    await updateOnboardingRecord(8, payload);
    await quickOnboardingRecord({ ...payload, employment_status: "formal" });
    expect(mocks.request).toHaveBeenNthCalledWith(1, "/onboarding-records", expect.objectContaining({ method: "POST", body: JSON.stringify(payload) }));
    expect(mocks.request).toHaveBeenNthCalledWith(2, "/onboarding-records/8", expect.objectContaining({ method: "PUT", body: JSON.stringify(payload) }));
    expect(mocks.request).toHaveBeenNthCalledWith(3, "/onboarding-records/quick", expect.objectContaining({ method: "POST", body: JSON.stringify({ ...payload, employment_status: "formal" }) }));
  });

  test("三条状态流转路径与请求体保持契约一致", async () => {
    await confirmOnboardingRecord(8, "trial");
    await abandonOnboardingRecord(8, "候选人放弃", "已电话确认");
    await restoreOnboardingRecord(8);
    expect(mocks.request).toHaveBeenNthCalledWith(1, "/onboarding-records/8/confirm", expect.objectContaining({ body: JSON.stringify({ employment_status: "trial" }) }));
    expect(mocks.request).toHaveBeenNthCalledWith(2, "/onboarding-records/8/abandon", expect.objectContaining({ body: JSON.stringify({ reason: "候选人放弃", remarks: "已电话确认" }) }));
    expect(mocks.request).toHaveBeenNthCalledWith(3, "/onboarding-records/8/restore", expect.objectContaining({ method: "POST", body: undefined }));
  });
});
