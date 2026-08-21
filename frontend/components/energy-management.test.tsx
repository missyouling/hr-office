import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, test, vi } from "vitest";
import { EnergyManagement } from "@/components/energy-management";

const mocks = vi.hoisted(() => ({ fetchSummary: vi.fn(), hasPermission: vi.fn() }));
vi.mock("@/lib/api-dormitory-energy", () => ({ fetchDormitoryEnergySummary: mocks.fetchSummary }));
vi.mock("@/lib/supabase/auth-context", () => ({ useAuth: () => ({ user: { id: 1 }, hasPermission: mocks.hasPermission }) }));

const summary = {
  overall: { electric: { usage: 128.5, amount: 96.4, count: 2 }, water: { usage: 18, amount: 36, count: 2 }, total_amount: 132.4 },
  by_building: [{ building_id: 1, building_name: "A 栋", electric: { usage: 128.5, amount: 96.4, count: 2 }, water: { usage: 18, amount: 36, count: 2 }, total_amount: 132.4 }],
  rooms: [{ room_id: 11, room_number: "101", building_id: 1, electric: { usage: 64, amount: 48, count: 1 }, water: { usage: 9, amount: 18, count: 1 }, total_amount: 66 }],
};

describe("EnergyManagement", () => {
  beforeEach(() => { vi.clearAllMocks(); mocks.hasPermission.mockReturnValue(true); mocks.fetchSummary.mockResolvedValue(summary); });

  test("展示整体指标、楼栋汇总，点击楼栋可下钻房间且没有写入或燃气功能", async () => {
    render(<EnergyManagement />);
    expect(await screen.findByRole("cell", { name: /128\.5 kWh/ })).toBeInTheDocument();
    fireEvent.click(screen.getAllByText("A 栋")[1]);
    expect(await screen.findByText("A 栋 · 房间明细")).toBeInTheDocument();
    expect(screen.getByText("101")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /新增|编辑|删除|保存|账单|预测|告警|燃气/ })).not.toBeInTheDocument();
    expect(mocks.fetchSummary).toHaveBeenCalledWith(expect.objectContaining({ buildingId: 1 }));
  });

  test("月份与楼栋筛选会重新请求对应数据", async () => {
    render(<EnergyManagement />);
    await screen.findByRole("cell", { name: "A 栋" });
    fireEvent.change(screen.getByLabelText("月份"), { target: { value: "2026-07" } });
    await waitFor(() => expect(mocks.fetchSummary).toHaveBeenCalledWith({ month: "2026-07", buildingId: undefined }));
    fireEvent.change(screen.getByLabelText("楼栋筛选"), { target: { value: "1" } });
    await waitFor(() => expect(mocks.fetchSummary).toHaveBeenCalledWith({ month: "2026-07", buildingId: 1 }));
  });

  test("空数据展示清晰空态", async () => {
    mocks.fetchSummary.mockResolvedValue({ overall: summary.overall, by_building: [], rooms: [] });
    render(<EnergyManagement />);
    expect(await screen.findByText("暂无能耗数据")).toBeInTheDocument();
  });

  test("无 dormitory.view 权限时不请求接口并显示无权限提示", () => {
    mocks.hasPermission.mockReturnValue(false);
    render(<EnergyManagement />);
    expect(screen.getByText("无能耗管理查看权限")).toBeInTheDocument();
    expect(mocks.fetchSummary).not.toHaveBeenCalled();
  });
});
