import { beforeEach, describe, expect, test, vi } from "vitest";

import { completeTrainingRecord, createTrainingRecord, deleteTrainingRecord, fetchTrainingRecords, getTrainingRecord, updateTrainingRecord, voidTrainingRecord, type TrainingPayload } from "@/lib/api-training-records";

const mocks = vi.hoisted(() => ({ request: vi.fn() }));
vi.mock("@/lib/api", () => ({ request: mocks.request }));
const payload: TrainingPayload = { topic: "安全生产培训", training_type: "internal", training_date: "2026-09-01", trainer_or_institution: "安全部", employee_id: 8, result: "通过", remarks: "年度必修" };

describe("培训记录 REST 客户端", () => {
  beforeEach(() => mocks.request.mockReset());
  test("列表按可选状态请求路径并支持详情", async () => { await fetchTrainingRecords(); await fetchTrainingRecords("completed"); await getTrainingRecord(3); expect(mocks.request).toHaveBeenNthCalledWith(1, "/training-records"); expect(mocks.request).toHaveBeenNthCalledWith(2, "/training-records?status=completed"); expect(mocks.request).toHaveBeenNthCalledWith(3, "/training-records/3"); });
  test("创建、编辑和删除使用约定路径与方法", async () => { await createTrainingRecord(payload); await updateTrainingRecord(3, payload); await deleteTrainingRecord(3); expect(mocks.request).toHaveBeenNthCalledWith(1, "/training-records", expect.objectContaining({ method: "POST", body: JSON.stringify(payload) })); expect(mocks.request).toHaveBeenNthCalledWith(2, "/training-records/3", expect.objectContaining({ method: "PUT", body: JSON.stringify(payload) })); expect(mocks.request).toHaveBeenNthCalledWith(3, "/training-records/3", { method: "DELETE" }); });
  test("完成与原因作废调用限定状态接口", async () => { await completeTrainingRecord(3); await voidTrainingRecord(3, "计划取消"); expect(mocks.request).toHaveBeenNthCalledWith(1, "/training-records/3/complete", expect.objectContaining({ method: "POST", body: undefined })); expect(mocks.request).toHaveBeenNthCalledWith(2, "/training-records/3/void", expect.objectContaining({ method: "POST", body: JSON.stringify({ reason: "计划取消" }) })); });
});
