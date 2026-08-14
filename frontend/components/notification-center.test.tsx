import { fireEvent, render, screen } from "@testing-library/react";
import { beforeEach, describe, expect, test, vi } from "vitest";
import { NotificationCenter } from "@/components/notification-center";
import { SITE_MEMO_STORAGE_KEY } from "@/lib/dorm-notifications";

const mocks = vi.hoisted(() => ({
  fetchDormSites: vi.fn(),
}));

vi.mock("@/lib/api", () => ({
  fetchDormSites: mocks.fetchDormSites,
}));

describe("NotificationCenter", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    window.localStorage.clear();
  });

  test("open=false 时不渲染", () => {
    render(<NotificationCenter open={false} onOpenChange={vi.fn()} />);
    expect(screen.queryByText("通知中心")).not.toBeInTheDocument();
  });

  test("空态：无站点时显示暂无待处理提醒", async () => {
    mocks.fetchDormSites.mockResolvedValue([]);
    render(<NotificationCenter open onOpenChange={vi.fn()} />);
    expect(await screen.findByText("暂无待处理提醒。")).toBeInTheDocument();
  });

  test("有 localStorage 备忘录且站点 API 返回站点时渲染列表", async () => {
    mocks.fetchDormSites.mockResolvedValue([{ id: 1, name: "一号宿舍" }]);
    window.localStorage.setItem(
      SITE_MEMO_STORAGE_KEY,
      JSON.stringify({ "1": [{ id: "m1", content: "检查消防设施", completed: false }] }),
    );
    render(<NotificationCenter open onOpenChange={vi.fn()} />);
    expect(await screen.findByText("一号宿舍")).toBeInTheDocument();
    expect(screen.getByText("检查消防设施")).toBeInTheDocument();
  });

  test("点击查看派发 dock:open-site-memo 并携带 siteId，同时关闭面板", async () => {
    mocks.fetchDormSites.mockResolvedValue([{ id: 1, name: "一号宿舍" }]);
    window.localStorage.setItem(
      SITE_MEMO_STORAGE_KEY,
      JSON.stringify({ "1": [{ id: "m1", content: "检查消防设施", completed: false }] }),
    );
    const handler = vi.fn();
    const onOpenChange = vi.fn();
    window.addEventListener("dock:open-site-memo", handler);
    render(<NotificationCenter open onOpenChange={onOpenChange} />);
    fireEvent.click(await screen.findByRole("button", { name: "查看一号宿舍备忘录" }));
    const event = handler.mock.calls[0][0] as CustomEvent<{ siteId: number }>;
    expect(event.detail.siteId).toBe(1);
    expect(onOpenChange).toHaveBeenCalledWith(false);
    window.removeEventListener("dock:open-site-memo", handler);
  });
});