import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, test, vi } from "vitest";

import { OccupationalHealthCheckManagement } from "@/components/occupational-health-check-management";

const mocks = vi.hoisted(() => ({ hasPermission: vi.fn(), fetchRecords: vi.fn(), fetchEmployees: vi.fn(), create: vi.fn(), complete: vi.fn(), remove: vi.fn(), void: vi.fn() }));
vi.mock("@/lib/auth", () => ({ useAuth: () => ({ user: { id: 1 }, hasPermission: mocks.hasPermission }) }));
vi.mock("@/lib/api", () => ({ fetchEmployees: mocks.fetchEmployees }));
vi.mock("@/lib/api-occupational-health-checks", () => ({ fetchOccupationalHealthChecks: mocks.fetchRecords, createOccupationalHealthCheck: mocks.create, updateOccupationalHealthCheck: vi.fn(), completeOccupationalHealthCheck: mocks.complete, deleteOccupationalHealthCheck: mocks.remove, voidOccupationalHealthCheck: mocks.void }));

const draft = { id: 1, employee_id: 2, employee_name: "张三", employee_department: "生产部", employee_position: "操作员", check_date: "2026-09-01", medical_institution: "市职业病防治院", check_category: "在岗期间检查", check_conclusion: "未见异常", next_check_date: "2027-09-01", remarks: "", status: "draft", void_reason: "", created_at: "", updated_at: "" } as const;
const completed = { ...draft, id: 2, employee_name: "李四", status: "completed" as const };
const voided = { ...draft, id: 3, employee_name: "王五", status: "voided" as const, void_reason: "信息重复" };

describe("OccupationalHealthCheckManagement", () => {
  beforeEach(() => { vi.clearAllMocks(); mocks.hasPermission.mockImplementation((resource: string) => resource === "occupational_health"); mocks.fetchRecords.mockResolvedValue([draft, completed, voided]); mocks.fetchEmployees.mockResolvedValue([{ id: 2, name: "张三", department: "生产部", position: "操作员", status: "active" }]); });

  test("展示员工快照、检查字段与三种状态，已作废记录为终态", async () => {
    render(<OccupationalHealthCheckManagement />); await waitFor(() => expect(screen.getByText("张三")).toBeInTheDocument());
    expect(screen.getAllByText("生产部 / 操作员")).toHaveLength(3); expect(screen.getAllByText("在岗期间检查")).toHaveLength(3);
    expect(screen.getAllByText("草稿")).toHaveLength(2); expect(screen.getAllByText("已完成")).toHaveLength(2); expect(screen.getAllByText("已作废")).toHaveLength(2);
    expect(screen.getAllByRole("button", { name: "编辑" })).toHaveLength(1); expect(screen.getAllByRole("button", { name: "完成" })).toHaveLength(1); expect(screen.getAllByRole("button", { name: "作废" })).toHaveLength(2);
  });

  test("草稿可完成，草稿和已完成记录均可作废", async () => {
    render(<OccupationalHealthCheckManagement />); await waitFor(() => expect(screen.getByText("张三")).toBeInTheDocument());
    fireEvent.click(screen.getByRole("button", { name: "完成" })); await waitFor(() => expect(mocks.complete).toHaveBeenCalledWith(1));
    fireEvent.click(screen.getAllByRole("button", { name: "作废" })[1]); fireEvent.change(screen.getByLabelText("作废原因 *"), { target: { value: "员工离职" } }); fireEvent.click(screen.getByRole("button", { name: "确认作废" }));
    await waitFor(() => expect(mocks.void).toHaveBeenCalledWith(2, "员工离职"));
  });

  test("作废原因必填，不能直接提交", async () => {
    render(<OccupationalHealthCheckManagement />); await waitFor(() => expect(screen.getByText("张三")).toBeInTheDocument()); fireEvent.click(screen.getAllByRole("button", { name: "作废" })[0]); fireEvent.click(screen.getByRole("button", { name: "确认作废" })); expect(mocks.void).not.toHaveBeenCalled();
  });

  test("保存失败时保留对话框和已填写字段", async () => {
    mocks.create.mockRejectedValueOnce(new Error("保存失败")); render(<OccupationalHealthCheckManagement />); await waitFor(() => expect(screen.getByText("张三")).toBeInTheDocument()); fireEvent.click(screen.getByRole("button", { name: "新建健康检查" }));
    const dialog = screen.getByRole("dialog", { name: "新建职业健康检查" }); fireEvent.change(screen.getByLabelText("员工"), { target: { value: "2" } }); fireEvent.change(screen.getByLabelText("检查日期 *"), { target: { value: "2026-09-01" } }); fireEvent.change(screen.getByLabelText("医疗机构 *"), { target: { value: "市职业病防治院" } }); fireEvent.change(screen.getByLabelText("检查类别 *"), { target: { value: "上岗前检查" } }); fireEvent.click(screen.getByRole("button", { name: "保存草稿" }));
    await waitFor(() => expect(mocks.create).toHaveBeenCalledWith(expect.objectContaining({ employee_id: 2, check_category: "上岗前检查" }))); expect(dialog).toBeInTheDocument(); expect(screen.getByLabelText("医疗机构 *")).toHaveValue("市职业病防治院");
  });

  test("按 occupational_health 操作权限隐藏创建与台账操作", async () => {
    mocks.hasPermission.mockImplementation((resource: string, action: string) => resource === "occupational_health" && action === "view"); render(<OccupationalHealthCheckManagement />); await waitFor(() => expect(screen.getByText("张三")).toBeInTheDocument());
    expect(screen.queryByRole("button", { name: "新建健康检查" })).not.toBeInTheDocument(); expect(screen.queryByRole("button", { name: "编辑" })).not.toBeInTheDocument(); expect(screen.queryByRole("button", { name: "完成" })).not.toBeInTheDocument(); expect(screen.queryByRole("button", { name: "作废" })).not.toBeInTheDocument();
  });

  test("无 occupational_health.view 权限时展示明确提示", () => { mocks.hasPermission.mockReturnValue(false); render(<OccupationalHealthCheckManagement />); expect(screen.getByText("无职业健康检查查看权限")).toBeInTheDocument(); });
});
