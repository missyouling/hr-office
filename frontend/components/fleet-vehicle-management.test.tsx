import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, test, vi } from "vitest";

import { FleetVehicleManagement } from "@/components/fleet-vehicle-management";

const mocks = vi.hoisted(() => ({ hasPermission: vi.fn(), fetchVehicles: vi.fn(), create: vi.fn(), update: vi.fn(), remove: vi.fn() }));
vi.mock("@/lib/auth", () => ({ useAuth: () => ({ user: { id: 1 }, hasPermission: mocks.hasPermission }) }));
vi.mock("@/lib/api-fleet-vehicles", () => ({ fetchFleetVehicles: mocks.fetchVehicles, createFleetVehicle: mocks.create, updateFleetVehicle: mocks.update, deleteFleetVehicle: mocks.remove }));

const active = { id: 1, plate_number: "粤A12345", vehicle_model: "商务车", status: "active" as const, brand: "比亚迪", seat_count: 7, purchase_date: "2026-01-01", remarks: "" };
const inactive = { ...active, id: 2, plate_number: "粤B54321", status: "inactive" as const };

describe("FleetVehicleManagement", () => {
  beforeEach(() => { vi.clearAllMocks(); mocks.hasPermission.mockReturnValue(true); mocks.fetchVehicles.mockResolvedValue([active, inactive]); });

  test("无 fleet.view 权限时显示明确提示", () => {
    mocks.hasPermission.mockReturnValue(false);
    render(<FleetVehicleManagement />);
    expect(screen.getByText("无车辆档案查看权限")).toBeInTheDocument();
  });

  test("入口和车辆操作按各自权限显隐", async () => {
    mocks.hasPermission.mockImplementation((_: string, action: string) => action === "view" || action === "create");
    render(<FleetVehicleManagement />);
    await waitFor(() => expect(screen.getByText("粤A12345")).toBeInTheDocument());
    expect(screen.getByRole("button", { name: "新增车辆" })).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "编辑" })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "删除" })).not.toBeInTheDocument();
  });

  test("已停用车辆不显示编辑和删除，只允许 fleet.edit 恢复启用", async () => {
    mocks.hasPermission.mockImplementation((_: string, action: string) => action === "view" || action === "edit" || action === "delete");
    render(<FleetVehicleManagement />);
    await waitFor(() => expect(screen.getByText("粤B54321")).toBeInTheDocument());
    expect(screen.getAllByRole("button", { name: "编辑" })).toHaveLength(1);
    expect(screen.getAllByRole("button", { name: "删除" })).toHaveLength(1);
    fireEvent.click(screen.getByRole("button", { name: "恢复启用" }));
    await waitFor(() => expect(mocks.update).toHaveBeenCalledWith(2, expect.objectContaining({ status: "active" })));
  });

  test("保存失败后新增对话框保持打开", async () => {
    mocks.create.mockRejectedValueOnce(new Error("保存失败"));
    render(<FleetVehicleManagement />);
    await waitFor(() => expect(screen.getByText("粤A12345")).toBeInTheDocument());
    fireEvent.click(screen.getByRole("button", { name: "新增车辆" }));
    fireEvent.change(screen.getByLabelText("车牌号 *"), { target: { value: "粤C88888" } });
    fireEvent.change(screen.getByLabelText("车型 *"), { target: { value: "轿车" } });
    fireEvent.click(screen.getByRole("button", { name: "保存" }));
    await waitFor(() => expect(mocks.create).toHaveBeenCalled());
    expect(screen.getByRole("dialog")).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "新增车辆" })).toBeInTheDocument();
  });

  test("删除失败后确认对话框保持打开", async () => {
    mocks.remove.mockRejectedValueOnce(new Error("删除失败"));
    render(<FleetVehicleManagement />);
    await waitFor(() => expect(screen.getByText("粤A12345")).toBeInTheDocument());
    fireEvent.click(screen.getByRole("button", { name: "删除" }));
    fireEvent.click(screen.getByRole("button", { name: "确认删除" }));
    await waitFor(() => expect(mocks.remove).toHaveBeenCalledWith(1));
    expect(screen.getByRole("dialog")).toBeInTheDocument();
    expect(screen.getByText("删除车辆档案")).toBeInTheDocument();
  });
});
