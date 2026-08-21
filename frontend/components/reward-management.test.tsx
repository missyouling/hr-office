import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, test, vi } from "vitest";

import { RewardManagement } from "@/components/reward-management";

const mocks = vi.hoisted(() => ({ hasPermission: vi.fn(), fetchRewards: vi.fn(), fetchEmployees: vi.fn(), fetchDocuments: vi.fn(), create: vi.fn(), update: vi.fn(), effective: vi.fn(), void: vi.fn() }));
vi.mock("@/lib/auth", () => ({ useAuth: () => ({ user: { id: 1 }, hasPermission: mocks.hasPermission }) }));
vi.mock("@/lib/api", () => ({ fetchEmployees: mocks.fetchEmployees, fetchDocuments: mocks.fetchDocuments }));
vi.mock("@/lib/api-rewards", () => ({ fetchRewards: mocks.fetchRewards, createReward: mocks.create, updateReward: mocks.update, makeRewardEffective: mocks.effective, voidReward: mocks.void }));

const draft = { id: 1, employee_id: 2, snapshot_name: "张三", snapshot_department: "人事部", snapshot_position: "专员", record_type: "reward", occurred_date: "2026-09-01", reason: "完成重点项目", level: "嘉奖", score: 10, amount: 500, owner: "李四", document_id: 9, status: "draft", effective_at: null, voided_at: null, void_reason: "", remarks: "", created_at: "", updated_at: "" } as const;
const effective = { ...draft, id: 2, record_type: "punishment" as const, status: "effective" as const, reason: "违反考勤规定" };

describe("RewardManagement", () => {
  beforeEach(() => { vi.clearAllMocks(); mocks.hasPermission.mockImplementation((resource: string) => resource === "reward"); mocks.fetchRewards.mockResolvedValue([draft, effective]); mocks.fetchEmployees.mockResolvedValue([{ id: 2, name: "张三", department: "人事部", status: "active" }]); mocks.fetchDocuments.mockResolvedValue({ items: [{ id: 9, file_name: "奖惩依据.pdf", document_code: "DOC-9" }] }); });

  test("展示两类台账及限定的草稿操作，不提示员工或薪资联动", async () => {
    render(<RewardManagement />);
    await waitFor(() => expect(screen.getByText("完成重点项目")).toBeInTheDocument());
    expect(screen.getByText("奖励")).toBeInTheDocument();
    expect(screen.getByText("惩罚")).toBeInTheDocument();
    expect(screen.getByText(/不联动员工状态或薪资/)).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "编辑" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "生效" })).toBeInTheDocument();
  });

  test("草稿手动生效，草稿和已生效记录均可作废", async () => {
    render(<RewardManagement />);
    await waitFor(() => expect(screen.getByText("完成重点项目")).toBeInTheDocument());
    fireEvent.click(screen.getByRole("button", { name: "生效" }));
    await waitFor(() => expect(mocks.effective).toHaveBeenCalledWith(1));
    expect(screen.getAllByRole("button", { name: "作废" })).toHaveLength(2);
  });

  test("作废原因必填，不能绕过确认提交", async () => {
    render(<RewardManagement />);
    await waitFor(() => expect(screen.getByText("完成重点项目")).toBeInTheDocument());
    fireEvent.click(screen.getAllByRole("button", { name: "作废" })[0]);
    fireEvent.click(screen.getByRole("button", { name: "确认作废" }));
    expect(mocks.void).not.toHaveBeenCalled();
  });

  test("无 reward.view 权限时显示明确权限状态", () => {
    mocks.hasPermission.mockReturnValue(false);
    render(<RewardManagement />);
    expect(screen.getByText("无奖惩记录查看权限")).toBeInTheDocument();
  });
});
