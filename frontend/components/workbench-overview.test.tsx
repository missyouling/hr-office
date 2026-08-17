import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { WorkbenchOverview } from "@/components/workbench-overview";
import { getWorkbenchConfig, getWorkbenchReminders } from "@/lib/api";

vi.mock("@/lib/api", () => ({
  getWorkbenchConfig: vi.fn(),
  getWorkbenchReminders: vi.fn(),
  updateWorkbenchConfig: vi.fn(),
}));

const getWorkbenchConfigMock = vi.mocked(getWorkbenchConfig);
const getWorkbenchRemindersMock = vi.mocked(getWorkbenchReminders);

describe("WorkbenchOverview", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    getWorkbenchRemindersMock.mockResolvedValue({ days: 30, items: [] });
  });

  it("展示欢迎文案、日期和农历", async () => {
    getWorkbenchConfigMock.mockResolvedValue({ weather: null, news: null });
    render(<WorkbenchOverview name="王小明" />);
    expect(screen.getByText(/好，王小明/)).toBeInTheDocument();
    await waitFor(() => expect(screen.getByText(/星期/)).toBeInTheDocument());
    expect(screen.getByText(/农历/)).toBeInTheDocument();
  });

  it("未配置时展示天气和新闻的配置引导", async () => {
    getWorkbenchConfigMock.mockResolvedValue({ weather: null, news: null });
    render(<WorkbenchOverview />);
    await waitFor(() => expect(screen.getAllByText("尚未配置")).toHaveLength(2));
    expect(screen.getAllByRole("button", { name: "立即配置" })).toHaveLength(2);
  });

  it("已配置时展示天气城市和新闻分类", async () => {
    getWorkbenchConfigMock.mockResolvedValue({ weather: { enabled: true, city: "杭州" }, news: { enabled: true, categories: ["财经", "科技"] } });
    render(<WorkbenchOverview />);
    await waitFor(() => expect(screen.getByText("城市：杭州")).toBeInTheDocument());
    expect(screen.getByText("关注：财经、科技")).toBeInTheDocument();
    expect(screen.getAllByText("已配置")).toHaveLength(2);
  });

  it("展示知识库问答入口", async () => {
    getWorkbenchConfigMock.mockResolvedValue({ weather: null, news: null });
    render(<WorkbenchOverview />);
    expect(screen.getByText("知识库问答")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "开始提问" })).toBeInTheDocument();
  });

  it("点击知识库问答入口派发 dock:open-chat 事件", async () => {
    getWorkbenchConfigMock.mockResolvedValue({ weather: null, news: null });
    const handler = vi.fn();
    window.addEventListener("dock:open-chat", handler);
    render(<WorkbenchOverview />);
    fireEvent.click(screen.getByRole("button", { name: "开始提问" }));
    expect(handler).toHaveBeenCalledTimes(1);
    window.removeEventListener("dock:open-chat", handler);
  });

  it("加载时展示骨架，失败时展示恢复提示", async () => {
    let rejectRequest: (error: Error) => void = () => undefined;
    getWorkbenchConfigMock.mockReturnValue(new Promise((_, reject) => { rejectRequest = reject; }));
    render(<WorkbenchOverview />);
    expect(document.querySelectorAll('[data-slot="skeleton"]')).toHaveLength(3);
    rejectRequest(new Error("网络异常"));
    await waitFor(() => expect(screen.getByText("工作台配置暂时无法加载，请稍后刷新重试。")).toBeInTheDocument());
  });

  it("提醒加载时展示独立骨架", () => {
    getWorkbenchConfigMock.mockResolvedValue({ weather: null, news: null });
    getWorkbenchRemindersMock.mockReturnValue(new Promise(() => undefined));
    render(<WorkbenchOverview />);
    expect(document.querySelectorAll('[data-slot="skeleton"]')).toHaveLength(3);
  });

  it("提醒为空时展示空状态", async () => {
    getWorkbenchConfigMock.mockResolvedValue({ weather: null, news: null });
    render(<WorkbenchOverview />);
    expect(await screen.findByText("暂无待处理提醒。")).toBeInTheDocument();
  });

  it("展示四类提醒标签，并按到期时间去重排序", async () => {
    getWorkbenchConfigMock.mockResolvedValue({ weather: null, news: null });
    getWorkbenchRemindersMock.mockResolvedValue({ days: 30, items: [
      { id: 4, reminder_type: "payment_request_pending", title: "请款单 PR-4", status: "submitted", due_at: null },
      { id: 2, reminder_type: "dorm_bill_due", title: "宿舍账单 B-2", status: "unpaid", due_at: "2026-09-20T00:00:00Z" },
      { id: 1, reminder_type: "document_expiration", title: "档案 D-1", status: "active", due_at: "2026-09-10T00:00:00Z" },
      { id: 3, reminder_type: "invoice_pending", title: "发票 I-3", status: "draft", due_at: null },
      { id: 1, reminder_type: "document_expiration", title: "重复档案", status: "active", due_at: "2026-09-10T00:00:00Z" },
    ] });
    render(<WorkbenchOverview />);
    const list = await screen.findByRole("list", { name: "工作台提醒列表" });
    expect(screen.getByText("档案到期")).toBeInTheDocument();
    expect(screen.getByText("宿舍账单到期")).toBeInTheDocument();
    expect(screen.getByText("发票待处理")).toBeInTheDocument();
    expect(screen.getByText("请款待处理")).toBeInTheDocument();
    expect(screen.getAllByText("待处理")).toHaveLength(2);
    expect(list.textContent).toMatch(/档案 D-1[\s\S]*宿舍账单 B-2[\s\S]*发票 I-3[\s\S]*请款单 PR-4/);
    expect(screen.queryByText("重复档案")).not.toBeInTheDocument();
  });

  it("提醒请求失败时不影响配置展示", async () => {
    getWorkbenchConfigMock.mockResolvedValue({ weather: { enabled: true, city: "杭州" }, news: null });
    getWorkbenchRemindersMock.mockRejectedValue(new Error("网络异常"));
    render(<WorkbenchOverview />);
    expect(await screen.findByText("提醒暂时无法加载，请稍后刷新重试。")).toBeInTheDocument();
    expect(screen.getByText("城市：杭州")).toBeInTheDocument();
  });

  it("提醒失败后点击重新加载会重新请求并恢复", async () => {
    getWorkbenchConfigMock.mockResolvedValue({ weather: null, news: null });
    getWorkbenchRemindersMock
      .mockRejectedValueOnce(new Error("网络异常"))
      .mockResolvedValueOnce({ days: 30, items: [{ id: 1, reminder_type: "document_expiration", title: "档案 D-1", status: "active", due_at: null }] });
    render(<WorkbenchOverview />);
    await screen.findByText("提醒暂时无法加载，请稍后刷新重试。");
    fireEvent.click(screen.getByRole("button", { name: "重新加载" }));
    await waitFor(() => expect(getWorkbenchRemindersMock).toHaveBeenCalledTimes(2));
    expect(await screen.findByText("档案 D-1")).toBeInTheDocument();
  });

  it("配置失败后点击重新加载会重新请求并恢复", async () => {
    getWorkbenchConfigMock
      .mockRejectedValueOnce(new Error("网络异常"))
      .mockResolvedValueOnce({ weather: { enabled: true, city: "杭州" }, news: null });
    render(<WorkbenchOverview />);
    await screen.findByText("工作台配置暂时无法加载，请稍后刷新重试。");
    fireEvent.click(screen.getByRole("button", { name: "重新加载" }));
    await waitFor(() => expect(getWorkbenchConfigMock).toHaveBeenCalledTimes(2));
    expect(await screen.findByText("城市：杭州")).toBeInTheDocument();
  });
});
