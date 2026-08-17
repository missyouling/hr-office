import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { WorkbenchMemos } from "@/components/workbench-memos";
import * as memoApi from "@/lib/api";

vi.mock("@/lib/api", () => ({
  getUserMemos: vi.fn(),
  createUserMemo: vi.fn(),
  updateUserMemo: vi.fn(),
  deleteUserMemo: vi.fn(),
}));

const firstMemo = { id: 1, title: "整理会议纪要", content: "周五前发出", pinned: false, completed: false, created_at: "2026-08-17T08:00:00Z", updated_at: "2026-08-17T08:00:00Z" };
const pinnedMemo = { ...firstMemo, id: 2, title: "优先事项", pinned: true, updated_at: "2026-08-18T08:00:00Z" };
const getMemosMock = vi.mocked(memoApi.getUserMemos);
const createMemoMock = vi.mocked(memoApi.createUserMemo);
const updateMemoMock = vi.mocked(memoApi.updateUserMemo);
const deleteMemoMock = vi.mocked(memoApi.deleteUserMemo);

describe("WorkbenchMemos", () => {
  beforeEach(() => { vi.clearAllMocks(); });

  it("加载时展示骨架，随后按置顶优先展示近期备忘录", async () => {
    getMemosMock.mockResolvedValue({ memos: [firstMemo, pinnedMemo] });
    render(<WorkbenchMemos />);
    expect(document.querySelector('[data-slot="skeleton"]')).toBeInTheDocument();
    const list = await screen.findByRole("list", { name: "近期备忘录列表" });
    expect(list.textContent).toMatch(/优先事项[\s\S]*整理会议纪要/);
    expect(screen.getAllByText("周五前发出")).toHaveLength(2);
  });

  it("请求失败时可重新加载", async () => {
    getMemosMock.mockRejectedValueOnce(new Error("网络异常")).mockResolvedValueOnce({ memos: [] });
    render(<WorkbenchMemos />);
    expect(await screen.findByText("网络异常")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "重新加载" }));
    expect(await screen.findByText("暂无备忘录，记下第一件重要的事吧。")).toBeInTheDocument();
    expect(getMemosMock).toHaveBeenCalledTimes(2);
  });

  it("无数据时展示空状态", async () => {
    getMemosMock.mockResolvedValue({ memos: [] });
    render(<WorkbenchMemos />);
    expect(await screen.findByText("暂无备忘录，记下第一件重要的事吧。")).toBeInTheDocument();
  });

  it("置顶和完成切换使用包含其他字段的完整 PUT 载荷", async () => {
    getMemosMock.mockResolvedValue({ memos: [firstMemo] });
    updateMemoMock.mockImplementation(async (_id, payload) => ({ ...firstMemo, ...payload }));
    render(<WorkbenchMemos />);
    await screen.findByText("整理会议纪要");
    fireEvent.click(screen.getByRole("button", { name: "置顶 整理会议纪要" }));
    await waitFor(() => expect(updateMemoMock).toHaveBeenLastCalledWith(1, { title: "整理会议纪要", content: "周五前发出", pinned: true, completed: false }));
    fireEvent.click(screen.getByRole("button", { name: "标为已完成 整理会议纪要" }));
    await waitFor(() => expect(updateMemoMock).toHaveBeenLastCalledWith(1, { title: "整理会议纪要", content: "周五前发出", pinned: true, completed: true }));
  });

  it("保存中禁用表单，保存失败后保留输入并提示错误", async () => {
    getMemosMock.mockResolvedValue({ memos: [] });
    let rejectSave: (error: Error) => void = () => undefined;
    createMemoMock.mockReturnValue(new Promise((_, reject) => { rejectSave = reject; }));
    render(<WorkbenchMemos />);
    await screen.findByText("暂无备忘录，记下第一件重要的事吧。");
    fireEvent.click(screen.getByRole("button", { name: "新增" }));
    fireEvent.change(screen.getByLabelText("标题"), { target: { value: "待确认" } });
    fireEvent.click(screen.getByRole("button", { name: "保存" }));
    expect(screen.getByRole("button", { name: "保存中…" })).toBeDisabled();
    rejectSave(new Error("保存异常"));
    expect(await screen.findByRole("alert")).toHaveTextContent("保存异常");
    expect(screen.getByDisplayValue("待确认")).toBeInTheDocument();
  });

  it("删除失败时保留备忘录并显示错误", async () => {
    getMemosMock.mockResolvedValue({ memos: [firstMemo] });
    deleteMemoMock.mockRejectedValue(new Error("删除异常"));
    render(<WorkbenchMemos />);
    await screen.findByText("整理会议纪要");
    fireEvent.click(screen.getByRole("button", { name: "删除 整理会议纪要" }));
    expect(await screen.findByText("删除异常")).toBeInTheDocument();
    expect(screen.getByText("整理会议纪要")).toBeInTheDocument();
  });
});
