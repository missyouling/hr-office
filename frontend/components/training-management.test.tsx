import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, test, vi } from "vitest";

import { TrainingManagement } from "@/components/training-management";

const mocks = vi.hoisted(() => ({ hasPermission: vi.fn(), fetchRecords: vi.fn(), fetchEmployees: vi.fn(), create: vi.fn(), update: vi.fn(), complete: vi.fn(), remove: vi.fn(), void: vi.fn() }));
vi.mock("@/lib/auth", () => ({ useAuth: () => ({ user: { id: 1 }, hasPermission: mocks.hasPermission }) }));
vi.mock("@/lib/api", () => ({ fetchEmployees: mocks.fetchEmployees }));
vi.mock("@/lib/api-training-records", () => ({ fetchTrainingRecords: mocks.fetchRecords, createTrainingRecord: mocks.create, updateTrainingRecord: mocks.update, completeTrainingRecord: mocks.complete, deleteTrainingRecord: mocks.remove, voidTrainingRecord: mocks.void }));

const draft = { id: 1, topic: "安全生产培训", training_type: "internal", training_date: "2026-09-01", trainer_or_institution: "安全部", employee_id: 2, snapshot_name: "张三", snapshot_department: "人事部", snapshot_position: "专员", result: "通过", remarks: "", status: "draft", void_reason: "", created_at: "", updated_at: "" } as const;
const completed = { ...draft, id: 2, training_type: "online" as const, topic: "合规线上培训", status: "completed" as const };

describe("TrainingManagement", () => {
  beforeEach(() => { vi.clearAllMocks(); mocks.hasPermission.mockImplementation((resource: string) => resource === "training"); mocks.fetchRecords.mockResolvedValue([draft, completed]); mocks.fetchEmployees.mockResolvedValue([{ id: 2, name: "张三", department: "人事部", position: "专员", status: "active" }]); });
  test("展示已确认培训类型，已完成记录不显示编辑", async () => { render(<TrainingManagement />); await waitFor(() => expect(screen.getByText("安全生产培训")).toBeInTheDocument()); expect(screen.getByText("内部培训")).toBeInTheDocument(); expect(screen.getByText("线上培训")).toBeInTheDocument(); expect(screen.getAllByRole("button", { name: "编辑" })).toHaveLength(1); expect(screen.getAllByRole("button", { name: "完成" })).toHaveLength(1); });
  test("草稿完成调用状态接口，草稿和已完成记录均可作废", async () => { render(<TrainingManagement />); await waitFor(() => expect(screen.getByText("安全生产培训")).toBeInTheDocument()); fireEvent.click(screen.getByRole("button", { name: "完成" })); await waitFor(() => expect(mocks.complete).toHaveBeenCalledWith(1)); expect(screen.getAllByRole("button", { name: "作废" })).toHaveLength(2); });
  test("作废原因必填，不能直接提交", async () => { render(<TrainingManagement />); await waitFor(() => expect(screen.getByText("安全生产培训")).toBeInTheDocument()); fireEvent.click(screen.getAllByRole("button", { name: "作废" })[1]); fireEvent.click(screen.getByRole("button", { name: "确认作废" })); expect(mocks.void).not.toHaveBeenCalled(); });
  test("按 training 操作权限隐藏创建和编辑按钮", async () => { mocks.hasPermission.mockImplementation((resource: string, action: string) => resource === "training" && action === "view"); render(<TrainingManagement />); await waitFor(() => expect(screen.getByText("安全生产培训")).toBeInTheDocument()); expect(screen.queryByRole("button", { name: "新建培训记录" })).not.toBeInTheDocument(); expect(screen.queryByRole("button", { name: "编辑" })).not.toBeInTheDocument(); expect(screen.queryByRole("button", { name: "完成" })).not.toBeInTheDocument(); expect(screen.queryByRole("button", { name: "作废" })).not.toBeInTheDocument(); });
  test("无 training.view 权限时显示明确权限状态", () => { mocks.hasPermission.mockReturnValue(false); render(<TrainingManagement />); expect(screen.getByText("无培训查看权限")).toBeInTheDocument(); });
});
