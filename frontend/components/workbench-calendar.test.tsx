import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { WorkbenchCalendar } from "@/components/workbench-calendar";
import * as calendarApi from "@/lib/api";

vi.mock("@/lib/api", () => ({
  getPersonalCalendar: vi.fn(),
  createPersonalCalendarEvent: vi.fn(),
  updatePersonalCalendarEvent: vi.fn(),
  deletePersonalCalendarEvent: vi.fn(),
}));

const event = { id: 7, title: "项目复盘", start_at: "2026-08-20T01:00:00Z", end_at: "2026-08-20T02:00:00Z", location: "会议室 A", notes: "请带资料", all_day: false };
const getCalendarMock = vi.mocked(calendarApi.getPersonalCalendar);
const createEventMock = vi.mocked(calendarApi.createPersonalCalendarEvent);
const updateEventMock = vi.mocked(calendarApi.updatePersonalCalendarEvent);
const deleteEventMock = vi.mocked(calendarApi.deletePersonalCalendarEvent);

describe("WorkbenchCalendar", () => {
  beforeEach(() => { vi.clearAllMocks(); });

  it("加载并展示近期日程和地点", async () => {
    getCalendarMock.mockResolvedValue([event]);
    render(<WorkbenchCalendar />);
    expect(document.querySelector('[data-slot="skeleton"]')).toBeInTheDocument();
    expect(await screen.findByText("项目复盘")).toBeInTheDocument();
    expect(screen.getByText("会议室 A")).toBeInTheDocument();
    expect(getCalendarMock).toHaveBeenCalledWith(expect.stringMatching(/Z$/), expect.stringMatching(/Z$/));
  });

  it("请求失败时展示错误并允许重新加载", async () => {
    getCalendarMock.mockRejectedValueOnce(new Error("网络异常")).mockResolvedValueOnce([]);
    render(<WorkbenchCalendar />);
    expect(await screen.findByText("网络异常")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "重新加载" }));
    expect(await screen.findByText("未来 30 天暂无日程，安排一件重要的事吧。")).toBeInTheDocument();
  });

  it("校验时间边界并以 RFC3339 创建日程", async () => {
    getCalendarMock.mockResolvedValue([]);
    createEventMock.mockResolvedValue(event);
    render(<WorkbenchCalendar />);
    await screen.findByText("未来 30 天暂无日程，安排一件重要的事吧。");
    fireEvent.click(screen.getByRole("button", { name: "新增日程" }));
    fireEvent.change(screen.getByLabelText("标题"), { target: { value: "项目复盘" } });
    fireEvent.change(screen.getByLabelText("开始时间"), { target: { value: "2026-08-20T10:00" } });
    fireEvent.change(screen.getByLabelText("结束时间"), { target: { value: "2026-08-20T09:00" } });
    fireEvent.click(screen.getByRole("button", { name: "保存" }));
    expect(await screen.findByRole("alert")).toHaveTextContent("结束时间不能早于开始时间。");
    fireEvent.change(screen.getByLabelText("结束时间"), { target: { value: "2026-08-20T11:00" } });
    fireEvent.click(screen.getByLabelText("全天"));
    fireEvent.click(screen.getByRole("button", { name: "保存" }));
    await waitFor(() => expect(createEventMock).toHaveBeenCalledWith(expect.objectContaining({ title: "项目复盘", all_day: true, start_at: expect.stringMatching(/Z$/), end_at: expect.stringMatching(/Z$/) })));
    expect(await screen.findByText("项目复盘")).toBeInTheDocument();
  });

  it("编辑和删除失败时保留日程并显示操作状态", async () => {
    getCalendarMock.mockResolvedValue([event]);
    updateEventMock.mockResolvedValue({ ...event, title: "复盘更新" });
    deleteEventMock.mockRejectedValue(new Error("删除失败"));
    render(<WorkbenchCalendar />);
    await screen.findByText("项目复盘");
    fireEvent.click(screen.getByRole("button", { name: "编辑 项目复盘" }));
    fireEvent.change(screen.getByLabelText("标题"), { target: { value: "复盘更新" } });
    fireEvent.click(screen.getByRole("button", { name: "保存" }));
    expect(await screen.findByText("复盘更新")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "删除 复盘更新" }));
    expect(await screen.findByText("删除失败")).toBeInTheDocument();
    expect(screen.getByText("复盘更新")).toBeInTheDocument();
  });
});
