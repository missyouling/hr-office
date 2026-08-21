import { beforeEach, describe, expect, test, vi } from "vitest";

import { createFleetVehicle, deleteFleetVehicle, fetchFleetVehicles, getFleetVehicle, updateFleetVehicle, type FleetVehiclePayload } from "@/lib/api-fleet-vehicles";

const mocks = vi.hoisted(() => ({ request: vi.fn() }));
vi.mock("@/lib/api", () => ({ request: mocks.request }));

const payload: FleetVehiclePayload = { plate_number: "粤A12345", vehicle_model: "商务车", status: "active", brand: "比亚迪", seat_count: 7, purchase_date: "2026-01-01", remarks: "接待用车" };

describe("车辆档案 REST 客户端", () => {
  beforeEach(() => mocks.request.mockReset());

  test("列表与详情使用车辆档案接口", async () => {
    await fetchFleetVehicles();
    await getFleetVehicle(8);
    expect(mocks.request).toHaveBeenNthCalledWith(1, "/fleet-vehicles");
    expect(mocks.request).toHaveBeenNthCalledWith(2, "/fleet-vehicles/8");
  });

  test("创建、更新和删除使用约定方法与载荷", async () => {
    await createFleetVehicle(payload);
    await updateFleetVehicle(8, payload);
    await deleteFleetVehicle(8);
    expect(mocks.request).toHaveBeenNthCalledWith(1, "/fleet-vehicles", expect.objectContaining({ method: "POST", body: JSON.stringify(payload) }));
    expect(mocks.request).toHaveBeenNthCalledWith(2, "/fleet-vehicles/8", expect.objectContaining({ method: "PUT", body: JSON.stringify(payload) }));
    expect(mocks.request).toHaveBeenNthCalledWith(3, "/fleet-vehicles/8", { method: "DELETE" });
  });
});
