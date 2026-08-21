import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, test, vi } from "vitest";

import { ResignationManagement } from "@/components/resignation-management";

const mocks = vi.hoisted(() => ({
  fetchEmployees: vi.fn(),
  resignEmployeeApi: vi.fn(),
  restoreEmployees: vi.fn(),
  downloadResignProof: vi.fn(),
  hasPermission: vi.fn(),
}));

vi.mock("@/lib/auth", () => ({
  useAuth: () => ({ user: { id: 1, username: "editor", full_name: "编辑员" }, token: "token", hasPermission: mocks.hasPermission }),
}));

vi.mock("@/lib/api", () => ({
  fetchEmployees: mocks.fetchEmployees,
  resignEmployeeApi: mocks.resignEmployeeApi,
  restoreEmployees: mocks.restoreEmployees,
  downloadResignProof: mocks.downloadResignProof,
}));

const activeEmployee = { id: 1, name: "张三", department: "行政部", status: "active", resign_date: null, resign_proof_name: null, resign_proof_url: null };
const resignedEmployee = { id: 2, name: "李四", department: "财务部", status: "resigned", resign_date: "2026-08-01", resign_proof_name: "离职证明.pdf", resign_proof_url: "/proof" };

describe("ResignationManagement", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mocks.hasPermission.mockReturnValue(true);
    mocks.fetchEmployees.mockResolvedValue([activeEmployee, resignedEmployee]);
    mocks.resignEmployeeApi.mockResolvedValue(resignedEmployee);
    mocks.restoreEmployees.mockResolvedValue({ restored: 1, employees: [activeEmployee] });
  });

  test("按 employee.edit 显示离职员工并可正常提交离职", async () => {
    render(<ResignationManagement />);
    await waitFor(() => expect(screen.getByText("李四")).toBeInTheDocument());
    fireEvent.click(screen.getByRole("button", { name: /办理离职/ }));
    fireEvent.change(screen.getByLabelText("离职原因"), { target: { value: "个人发展" } });
    fireEvent.click(screen.getByRole("button", { name: "确认离职" }));
    await waitFor(() => expect(mocks.resignEmployeeApi).toHaveBeenCalledWith(1, expect.any(String), null, "token", ["个人发展"]));
  });

  test("离职日期缺失时阻止提交", async () => {
    render(<ResignationManagement />);
    await waitFor(() => expect(screen.getByText("李四")).toBeInTheDocument());
    fireEvent.click(screen.getByRole("button", { name: /办理离职/ }));
    fireEvent.change(screen.getByLabelText("离职日期"), { target: { value: "" } });
    fireEvent.click(screen.getByRole("button", { name: "确认离职" }));
    expect(mocks.resignEmployeeApi).not.toHaveBeenCalled();
  });

  test("恢复在职需要确认，并调用现有恢复接口", async () => {
    render(<ResignationManagement />);
    await waitFor(() => expect(screen.getByText("李四")).toBeInTheDocument());
    fireEvent.click(screen.getByRole("button", { name: /恢复在职/ }));
    expect(screen.getByText(/清空离职日期、原因和证明关联/)).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "确认恢复" }));
    await waitFor(() => expect(mocks.restoreEmployees).toHaveBeenCalledWith({ ids: [2] }, "token"));
  });

  test("无 employee.edit 权限时隐藏页面", () => {
    mocks.hasPermission.mockReturnValue(false);
    render(<ResignationManagement />);
    expect(screen.queryByRole("heading", { name: "离职管理" })).not.toBeInTheDocument();
  });
});
