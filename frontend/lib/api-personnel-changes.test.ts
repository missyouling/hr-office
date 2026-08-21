import { beforeEach, describe, expect, test, vi } from "vitest";

import { activatePersonnelChange, createPersonnelChange, deletePersonnelChange, fetchPersonnelChanges, getPersonnelChange, updatePersonnelChange, voidPersonnelChange, type PersonnelChangePayload } from "@/lib/api-personnel-changes";

const mocks = vi.hoisted(() => ({ request: vi.fn() }));
vi.mock("@/lib/api", () => ({ request: mocks.request }));
const payload: PersonnelChangePayload = { employee_id: 8, change_type: "promotion", effective_date: "2026-09-01", reason: "年度考核优秀", after_position: "高级专员", after_job_level: "P5", after_department_id: 12 };

describe("人事异动 REST 客户端", () => {
  beforeEach(() => mocks.request.mockReset());
  test("列表按可选状态请求既定路径，并支持单条查询", async () => { await fetchPersonnelChanges(); await fetchPersonnelChanges("draft"); await getPersonnelChange(3); expect(mocks.request).toHaveBeenNthCalledWith(1, "/personnel-changes"); expect(mocks.request).toHaveBeenNthCalledWith(2, "/personnel-changes?status=draft"); expect(mocks.request).toHaveBeenNthCalledWith(3, "/personnel-changes/3"); });
  test("创建、草稿编辑和删除使用约定方法", async () => { await createPersonnelChange(payload); await updatePersonnelChange(3, payload); await deletePersonnelChange(3); expect(mocks.request).toHaveBeenNthCalledWith(1, "/personnel-changes", expect.objectContaining({ method: "POST", body: JSON.stringify(payload) })); expect(mocks.request).toHaveBeenNthCalledWith(2, "/personnel-changes/3", expect.objectContaining({ method: "PUT", body: JSON.stringify(payload) })); expect(mocks.request).toHaveBeenNthCalledWith(3, "/personnel-changes/3", { method: "DELETE" }); });
  test("手动生效与原因作废调用限定状态接口", async () => { await activatePersonnelChange(3); await voidPersonnelChange(3, "业务调整"); expect(mocks.request).toHaveBeenNthCalledWith(1, "/personnel-changes/3/activate", expect.objectContaining({ method: "POST", body: undefined })); expect(mocks.request).toHaveBeenNthCalledWith(2, "/personnel-changes/3/void", expect.objectContaining({ method: "POST", body: JSON.stringify({ reason: "业务调整" }) })); });
});
