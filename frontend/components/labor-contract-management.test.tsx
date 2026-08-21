import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, test, vi } from "vitest";

import { LaborContractManagement } from "@/components/labor-contract-management";

const mocks = vi.hoisted(() => ({ hasPermission: vi.fn(), fetchContracts: vi.fn(), fetchEmployees: vi.fn(), fetchDocuments: vi.fn(), create: vi.fn(), update: vi.fn(), activate: vi.fn(), cancel: vi.fn() }));
vi.mock("@/lib/auth", () => ({ useAuth: () => ({ user: { id: 1 }, hasPermission: mocks.hasPermission }) }));
vi.mock("@/lib/api", () => ({ fetchEmployees: mocks.fetchEmployees, fetchDocuments: mocks.fetchDocuments }));
vi.mock("@/lib/api-labor-contracts", () => ({ fetchLaborContracts: mocks.fetchContracts, createLaborContract: mocks.create, updateLaborContract: mocks.update, activateLaborContract: mocks.activate, cancelLaborContract: mocks.cancel }));

const draft = { id: 1, contract_no: "LC-001", employee_id: 2, snapshot_name: "张三", snapshot_department: "人事部", snapshot_position: "专员", snapshot_id_number: "", contract_type: "fixed_term", start_date: "2026-09-01", end_date: "2029-08-31", term_months: 36, document_id: 9, document: { id: 9, document_code: "DOC-9", file_name: "张三劳动合同.pdf" }, status: "draft", cancel_reason: "", cancelled_at: null, activated_at: null, expired_at: null, remarks: "", created_at: "", updated_at: "" } as const;
const active = { ...draft, id: 2, contract_no: "LC-002", status: "active" as const, end_date: "2026-09-10" };

describe("LaborContractManagement", () => {
  beforeEach(() => { vi.clearAllMocks(); mocks.hasPermission.mockImplementation((resource: string) => resource === "contract"); mocks.fetchContracts.mockResolvedValue([draft, active]); mocks.fetchEmployees.mockResolvedValue([{ id: 2, name: "张三", department: "人事部", status: "active" }]); mocks.fetchDocuments.mockResolvedValue({ items: [{ id: 9, file_name: "张三劳动合同.pdf", document_code: "DOC-9" }] }); });

  test("只展示固定期限台账、到期提醒和允许的草稿操作", async () => {
    render(<LaborContractManagement />);
    await waitFor(() => expect(screen.getByText("LC-001")).toBeInTheDocument());
    expect(screen.getByText("固定期限合同台账；到期仅提醒，不改变履行中的合同。")).toBeInTheDocument();
    expect(screen.getByText("即将到期，仅提醒")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "编辑" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "生效" })).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "手动到期" })).not.toBeInTheDocument();
  });

  test("草稿生效调用状态接口，生效合同只提供作废", async () => {
    render(<LaborContractManagement />);
    await waitFor(() => expect(screen.getByText("LC-001")).toBeInTheDocument());
    fireEvent.click(screen.getByRole("button", { name: "生效" }));
    await waitFor(() => expect(mocks.activate).toHaveBeenCalledWith(1));
    expect(screen.getAllByRole("button", { name: "作废" })).toHaveLength(1);
  });

  test("作废原因必填，且不允许绕过确认直接提交", async () => {
    render(<LaborContractManagement />);
    await waitFor(() => expect(screen.getByText("LC-002")).toBeInTheDocument());
    fireEvent.click(screen.getByRole("button", { name: "作废" }));
    fireEvent.click(screen.getByRole("button", { name: "确认作废" }));
    expect(mocks.cancel).not.toHaveBeenCalled();
  });

  test("无 contract.view 权限时显示明确的权限状态", () => {
    mocks.hasPermission.mockReturnValue(false);
    render(<LaborContractManagement />);
    expect(screen.getByText("无劳动合同管理权限")).toBeInTheDocument();
  });

  test("无 contract.create 权限时隐藏新建入口", async () => {
    mocks.hasPermission.mockImplementation((resource: string, action: string) => resource === "contract" && action !== "create");
    render(<LaborContractManagement />);
    await waitFor(() => expect(screen.getByText("LC-001")).toBeInTheDocument());
    expect(screen.queryByRole("button", { name: "新建合同" })).not.toBeInTheDocument();
  });
});