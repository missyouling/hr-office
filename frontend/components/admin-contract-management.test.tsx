import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, test, vi } from "vitest";

import { AdminContractManagement } from "@/components/admin-contract-management";

const mocks = vi.hoisted(() => ({ hasPermission: vi.fn(), fetchContracts: vi.fn(), fetchDocuments: vi.fn(), activate: vi.fn(), cancel: vi.fn() }));
vi.mock("@/lib/auth", () => ({ useAuth: () => ({ user: { id: 1 }, hasPermission: mocks.hasPermission }) }));
vi.mock("@/lib/api", () => ({ fetchDocuments: mocks.fetchDocuments }));
vi.mock("@/lib/api-admin-contracts", () => ({ fetchAdminContracts: mocks.fetchContracts, activateAdminContract: mocks.activate, cancelAdminContract: mocks.cancel, createAdminContract: vi.fn(), updateAdminContract: vi.fn() }));

const draft = { id: 1, contract_no: "AC-001", name: "物业服务合同", counterparty: "安心物业", contract_type: "服务合同", start_date: "2026-09-01", end_date: "2027-08-31", amount_incl_tax: null, currency: "CNY", owner: "", document_id: null, status: "draft", activated_at: null, expired_at: null, cancelled_at: null, cancel_reason: "", remarks: "", created_at: "", updated_at: "" } as const;

describe("AdminContractManagement", () => {
  beforeEach(() => { vi.clearAllMocks(); mocks.hasPermission.mockImplementation((resource: string) => resource === "admin_contract"); mocks.fetchContracts.mockResolvedValue([draft, { ...draft, id: 2, status: "active" as const, end_date: "2026-09-10" }]); mocks.fetchDocuments.mockResolvedValue({ items: [] }); });
  test("展示外部行政合同、到期提醒且不会提供手动到期操作", async () => {
    render(<AdminContractManagement />); await waitFor(() => expect(screen.getAllByText("AC-001")).toHaveLength(2));
    expect(screen.getByText("集中管理供应商、服务及其他外部主体合同。到期只提醒，不自动改变状态。")).toBeInTheDocument();
    expect(screen.getByText("即将到期，仅提醒")).toBeInTheDocument(); expect(screen.queryByRole("button", { name: "手动到期" })).not.toBeInTheDocument();
  });
  test("草稿可以手动生效，草稿和履行中合同都允许作废", async () => {
    render(<AdminContractManagement />); await waitFor(() => expect(screen.getAllByText("AC-001")).toHaveLength(2));
    fireEvent.click(screen.getByRole("button", { name: "生效" })); await waitFor(() => expect(mocks.activate).toHaveBeenCalledWith(1));
    expect(screen.getAllByRole("button", { name: "作废" })).toHaveLength(2);
  });
  test("作废必须填写原因", async () => {
    render(<AdminContractManagement />); await waitFor(() => expect(screen.getAllByText("AC-001")).toHaveLength(2));
    fireEvent.click(screen.getAllByRole("button", { name: "作废" })[0]); fireEvent.click(screen.getByRole("button", { name: "确认作废" }));
    expect(mocks.cancel).not.toHaveBeenCalled();
  });
  test("无查看权限时显示权限提示", () => {
    mocks.hasPermission.mockReturnValue(false); render(<AdminContractManagement />); expect(screen.getByText("无行政合同查看权限")).toBeInTheDocument();
  });
});
