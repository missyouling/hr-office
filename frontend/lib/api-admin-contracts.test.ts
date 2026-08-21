import { beforeEach, describe, expect, test, vi } from "vitest";

import { activateAdminContract, cancelAdminContract, createAdminContract, fetchAdminContracts, updateAdminContract, type AdminContractPayload } from "@/lib/api-admin-contracts";

const mocks = vi.hoisted(() => ({ request: vi.fn() }));
vi.mock("@/lib/api", () => ({ request: mocks.request }));

const payload: AdminContractPayload = { contract_no: "AC-001", name: "物业服务合同", counterparty: "安心物业", contract_type: "服务合同", start_date: "2026-09-01", end_date: "2027-08-31", amount_incl_tax: 120000, currency: "CNY", owner: "张三", document_id: 9, remarks: "年度续签" };

describe("行政合同 REST 客户端", () => {
  beforeEach(() => mocks.request.mockReset());
  test("列表按可选状态请求行政合同路径", async () => {
    await fetchAdminContracts(); await fetchAdminContracts("active");
    expect(mocks.request).toHaveBeenNthCalledWith(1, "/admin-contracts");
    expect(mocks.request).toHaveBeenNthCalledWith(2, "/admin-contracts?status=active");
  });
  test("创建和编辑发送完整行政合同字段", async () => {
    await createAdminContract(payload); await updateAdminContract(3, payload);
    expect(mocks.request).toHaveBeenNthCalledWith(1, "/admin-contracts", expect.objectContaining({ method: "POST", body: JSON.stringify(payload) }));
    expect(mocks.request).toHaveBeenNthCalledWith(2, "/admin-contracts/3", expect.objectContaining({ method: "PUT", body: JSON.stringify(payload) }));
  });
  test("手动生效和作废使用限定状态接口", async () => {
    await activateAdminContract(3); await cancelAdminContract(3, "供应商变更");
    expect(mocks.request).toHaveBeenNthCalledWith(1, "/admin-contracts/3/activate", expect.objectContaining({ method: "POST", body: undefined }));
    expect(mocks.request).toHaveBeenNthCalledWith(2, "/admin-contracts/3/cancel", expect.objectContaining({ method: "POST", body: JSON.stringify({ reason: "供应商变更" }) }));
  });
});
