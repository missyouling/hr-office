import { beforeEach, describe, expect, test, vi } from "vitest";

import { activateLaborContract, cancelLaborContract, createLaborContract, fetchLaborContracts, updateLaborContract, type LaborContractPayload } from "@/lib/api-labor-contracts";

const mocks = vi.hoisted(() => ({ request: vi.fn() }));
vi.mock("@/lib/api", () => ({ request: mocks.request }));

const payload: LaborContractPayload = { employee_id: 8, start_date: "2026-09-01", end_date: "2029-08-31", term_months: 36, document_id: 12, remarks: "续签" };

describe("劳动合同 REST 客户端", () => {
  beforeEach(() => mocks.request.mockReset());

  test("列表按可选状态请求既定路径", async () => {
    await fetchLaborContracts();
    await fetchLaborContracts("active");
    expect(mocks.request).toHaveBeenNthCalledWith(1, "/contracts");
    expect(mocks.request).toHaveBeenNthCalledWith(2, "/contracts?status=active");
  });

  test("草稿创建和编辑提交固定期限及档案附件关联", async () => {
    await createLaborContract(payload);
    await updateLaborContract(3, payload);
    expect(mocks.request).toHaveBeenNthCalledWith(1, "/contracts", expect.objectContaining({ method: "POST", body: JSON.stringify(payload) }));
    expect(mocks.request).toHaveBeenNthCalledWith(2, "/contracts/3", expect.objectContaining({ method: "PUT", body: JSON.stringify(payload) }));
  });

  test("生效与作废只调用允许的状态流转接口", async () => {
    await activateLaborContract(3);
    await cancelLaborContract(3, "双方协商解除");
    expect(mocks.request).toHaveBeenNthCalledWith(1, "/contracts/3/activate", expect.objectContaining({ method: "POST", body: undefined }));
    expect(mocks.request).toHaveBeenNthCalledWith(2, "/contracts/3/cancel", expect.objectContaining({ method: "POST", body: JSON.stringify({ reason: "双方协商解除" }) }));
  });
});