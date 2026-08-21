import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, test, vi } from "vitest";

import { PersonnelChangeManagement } from "@/components/personnel-change-management";

const mocks = vi.hoisted(() => ({ hasPermission: vi.fn(), fetchChanges: vi.fn(), fetchEmployees: vi.fn(), listDepartments: vi.fn(), create: vi.fn(), update: vi.fn(), activate: vi.fn(), void: vi.fn() }));
vi.mock("@/lib/auth", () => ({ useAuth: () => ({ user: { id: 1 }, hasPermission: mocks.hasPermission }) }));
vi.mock("@/lib/api", () => ({ fetchEmployees: mocks.fetchEmployees }));
vi.mock("@/lib/api-department", () => ({ departmentApi: { list: mocks.listDepartments } }));
vi.mock("@/lib/api-personnel-changes", () => ({ fetchPersonnelChanges: mocks.fetchChanges, createPersonnelChange: mocks.create, updatePersonnelChange: mocks.update, activatePersonnelChange: mocks.activate, voidPersonnelChange: mocks.void }));

const draft = { id: 1, employee_id: 2, before_department: "人事部", before_position: "专员", before_job_level: "P3", change_type: "promotion", effective_date: "2026-09-01", reason: "年度考核优秀", after_department: "人事部", after_position: "高级专员", after_job_level: "P5", status: "draft", effective_at: null, voided_at: null, void_reason: "", created_at: "", updated_at: "" } as const;
const effective = { ...draft, id: 2, status: "effective" as const, change_type: "transfer" as const, reason: "组织调整", after_department: "行政部" };

describe("PersonnelChangeManagement", () => {
  beforeEach(() => { vi.clearAllMocks(); mocks.hasPermission.mockImplementation((resource: string, action: string) => resource === "employee" && action === "edit"); mocks.fetchChanges.mockResolvedValue([draft, effective]); mocks.fetchEmployees.mockResolvedValue([{ id: 2, name: "张三", department: "人事部", position: "专员", job_level: "P3", status: "active" }]); mocks.listDepartments.mockResolvedValue({ data: [{ id: 10, name: "人事部", code: "HR" }, { id: 12, name: "行政部", code: "ADM" }] }); });
  test("已生效记录不显示编辑，仅草稿显示编辑与生效", async () => { render(<PersonnelChangeManagement />); await waitFor(() => expect(screen.getByText("年度考核优秀")).toBeInTheDocument()); expect(screen.getAllByRole("button", { name: "编辑" })).toHaveLength(1); expect(screen.getAllByRole("button", { name: "生效" })).toHaveLength(1); expect(screen.getAllByRole("button", { name: "作废" })).toHaveLength(2); });
  test("异动后信息至少填写一项", async () => { render(<PersonnelChangeManagement />); await waitFor(() => expect(screen.getByText("新建人事异动")).toBeInTheDocument()); fireEvent.click(screen.getByRole("button", { name: "新建人事异动" })); fireEvent.change(screen.getByLabelText("员工"), { target: { value: "2" } }); fireEvent.change(screen.getByLabelText("异动类型"), { target: { value: "transfer" } }); fireEvent.change(screen.getByLabelText(/生效日期/), { target: { value: "2026-09-01" } }); fireEvent.change(screen.getByLabelText("事由"), { target: { value: "组织调整" } }); fireEvent.click(screen.getByRole("button", { name: "保存草稿" })); expect(mocks.create).not.toHaveBeenCalled(); });
  test("异动后部门从同公司部门列表选择并提交 ID", async () => { render(<PersonnelChangeManagement />); await waitFor(() => expect(screen.getByText("新建人事异动")).toBeInTheDocument()); fireEvent.click(screen.getByRole("button", { name: "新建人事异动" })); fireEvent.change(screen.getByLabelText("员工"), { target: { value: "2" } }); fireEvent.change(screen.getByLabelText(/生效日期/), { target: { value: "2026-09-01" } }); fireEvent.change(screen.getByLabelText("事由"), { target: { value: "组织调整" } }); fireEvent.change(screen.getByLabelText("异动后部门"), { target: { value: "12" } }); fireEvent.click(screen.getByRole("button", { name: "保存草稿" })); await waitFor(() => expect(mocks.create).toHaveBeenCalledWith(expect.objectContaining({ after_department_id: 12 }))); });
  test("作废原因必填，且已生效作废提示不回滚员工资料", async () => { render(<PersonnelChangeManagement />); await waitFor(() => expect(screen.getByText("组织调整")).toBeInTheDocument()); fireEvent.click(screen.getAllByRole("button", { name: "作废" })[1]); expect(screen.getByText("已生效记录作废不会回滚员工资料。")).toBeInTheDocument(); fireEvent.click(screen.getByRole("button", { name: "确认作废" })); expect(mocks.void).not.toHaveBeenCalled(); });
  test("无 employee.edit 权限时显示明确权限状态", () => { mocks.hasPermission.mockReturnValue(false); render(<PersonnelChangeManagement />); expect(screen.getByText("无人事异动管理权限")).toBeInTheDocument(); });
});
