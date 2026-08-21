import { beforeEach, describe, expect, test, vi } from "vitest";
import { fetchDormitoryEnergySummary } from "@/lib/api-dormitory-energy";

const mocks = vi.hoisted(() => ({ request: vi.fn() }));
vi.mock("@/lib/api", () => ({ request: mocks.request }));

describe("宿舍能耗汇总客户端", () => {
  beforeEach(() => mocks.request.mockReset());

  test("仅以 GET 请求获取能耗汇总，并传递月份与楼栋筛选", async () => {
    await fetchDormitoryEnergySummary({ month: "2026-08", buildingId: 12 });
    expect(mocks.request).toHaveBeenCalledWith("/dormitories/energy/summary?month=2026-08&building_id=12");
  });

  test("无筛选条件时不附加查询参数", async () => {
    await fetchDormitoryEnergySummary();
    expect(mocks.request).toHaveBeenCalledWith("/dormitories/energy/summary");
  });
});
