import { beforeEach, describe, expect, test, vi } from "vitest";

import { createReward, fetchRewards, makeRewardEffective, updateReward, voidReward, type RewardPayload } from "@/lib/api-rewards";

const mocks = vi.hoisted(() => ({ request: vi.fn() }));
vi.mock("@/lib/api", () => ({ request: mocks.request }));

const payload: RewardPayload = { employee_id: 8, record_type: "reward", occurred_date: "2026-09-01", reason: "完成重点项目", level: "嘉奖", score: 10, amount: 500, owner: "张三", document_id: 12, remarks: "季度表彰" };

describe("奖惩记录 REST 客户端", () => {
  beforeEach(() => mocks.request.mockReset());

  test("列表按可选状态请求既定路径", async () => {
    await fetchRewards(); await fetchRewards("effective");
    expect(mocks.request).toHaveBeenNthCalledWith(1, "/rewards");
    expect(mocks.request).toHaveBeenNthCalledWith(2, "/rewards?status=effective");
  });

  test("草稿创建和编辑提交已确认字段", async () => {
    await createReward(payload); await updateReward(3, payload);
    expect(mocks.request).toHaveBeenNthCalledWith(1, "/rewards", expect.objectContaining({ method: "POST", body: JSON.stringify(payload) }));
    expect(mocks.request).toHaveBeenNthCalledWith(2, "/rewards/3", expect.objectContaining({ method: "PUT", body: JSON.stringify(payload) }));
  });

  test("仅提供手动生效与填写原因作废状态接口", async () => {
    await makeRewardEffective(3); await voidReward(3, "事实认定有误");
    expect(mocks.request).toHaveBeenNthCalledWith(1, "/rewards/3/activate", expect.objectContaining({ method: "POST", body: undefined }));
    expect(mocks.request).toHaveBeenNthCalledWith(2, "/rewards/3/void", expect.objectContaining({ method: "POST", body: JSON.stringify({ reason: "事实认定有误" }) }));
  });
});
