import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, test, vi } from "vitest";

import { OnboardingManagement } from "@/components/onboarding-management";

const mocks = vi.hoisted(() => ({
  hasPermission: vi.fn(), fetchRecords: vi.fn(), createRecord: vi.fn(), quickRecord: vi.fn(),
  updateRecord: vi.fn(), confirmRecord: vi.fn(), abandonRecord: vi.fn(), restoreRecord: vi.fn(),
}));

vi.mock("@/lib/auth", () => ({ useAuth: () => ({ user: { id: 1 }, hasPermission: mocks.hasPermission }) }));
vi.mock("@/lib/api-onboarding", () => ({
  fetchOnboardingRecords: mocks.fetchRecords, createOnboardingRecord: mocks.createRecord,
  quickOnboardingRecord: mocks.quickRecord, updateOnboardingRecord: mocks.updateRecord,
  confirmOnboardingRecord: mocks.confirmRecord, abandonOnboardingRecord: mocks.abandonRecord,
  restoreOnboardingRecord: mocks.restoreRecord,
}));

const pending = { id: 1, name: "张三", id_number: "1101", phone: "138", department: "人事部", position: "专员", planned_hire_date: "2026-09-01", actual_hire_date: null, status: "pending", employment_status: "", employee_id: null, abandon_reason: "", abandoned_at: null, remarks: "", offer_id: "OFF-1", offer_source: "", created_at: "", updated_at: "" };
const abandoned = { ...pending, id: 2, name: "李四", status: "abandoned", abandon_reason: "个人原因" };

describe("OnboardingManagement", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mocks.hasPermission.mockReturnValue(true);
    mocks.fetchRecords.mockResolvedValue([pending, abandoned]);
    mocks.createRecord.mockResolvedValue(pending);
    mocks.confirmRecord.mockResolvedValue({ ...pending, status: "onboarded" });
    mocks.restoreRecord.mockResolvedValue({ ...abandoned, status: "pending" });
  });

  test("展示三态记录和真实操作，不伪造批量导入", async () => {
    render(<OnboardingManagement />);
    await waitFor(() => expect(screen.getByText("张三")).toBeInTheDocument());
    expect(screen.getByText("李四")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /登记待入职/ })).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /批量导入/ })).not.toBeInTheDocument();
  });

  test("确认入职选择用工状态后调用状态接口", async () => {
    render(<OnboardingManagement />);
    await waitFor(() => expect(screen.getByText("张三")).toBeInTheDocument());
    fireEvent.click(screen.getByRole("button", { name: "确认入职" }));
    fireEvent.change(screen.getByLabelText("用工状态"), { target: { value: "formal" } });
    fireEvent.click(screen.getByRole("button", { name: "确认入职" }));
    await waitFor(() => expect(mocks.confirmRecord).toHaveBeenCalledWith(1, "formal"));
  });

  test("恢复已放弃记录并保留后端历史语义", async () => {
    render(<OnboardingManagement />);
    await waitFor(() => expect(screen.getByText("李四")).toBeInTheDocument());
    fireEvent.click(screen.getByRole("button", { name: "恢复待入职" }));
    await waitFor(() => expect(mocks.restoreRecord).toHaveBeenCalledWith(2));
  });

  test("无 create/edit 权限时保留只读列表并隐藏全部操作", async () => {
    mocks.hasPermission.mockReturnValue(false);
    render(<OnboardingManagement />);
    await waitFor(() => expect(screen.getByText("张三")).toBeInTheDocument());
    expect(screen.queryByRole("button", { name: /登记待入职/ })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "编辑" })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "确认入职" })).not.toBeInTheDocument();
  });
});
