import { afterEach, describe, expect, it, vi } from "vitest";
import { getPersonalCalendar, type PersonalCalendarEvent } from "./api";

describe("个人日历 API", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("getPersonalCalendar 解包后端 { events } 响应并返回事件数组", async () => {
    const events: PersonalCalendarEvent[] = [
      {
        id: 1,
        title: "休假",
        start_at: "2026-08-10T09:00:00Z",
        end_at: "2026-08-10T10:00:00Z",
        location: "会议室",
        notes: "年中休假",
        all_day: false,
      },
    ];
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue({ ok: true, json: async () => ({ events }) }),
    );

    const result = await getPersonalCalendar("2026-08-01", "2026-08-31");

    expect(result).toEqual(events);
    expect(result).toHaveLength(1);
    expect(result[0].title).toBe("休假");
  });

  it("getPersonalCalendar 携带 from/to 查询参数", async () => {
    const mockFetch = vi.fn().mockResolvedValue({
      ok: true,
      json: async () => ({ events: [] }),
    });
    vi.stubGlobal("fetch", mockFetch);

    const result = await getPersonalCalendar("2026-08-01", "2026-08-31");

    expect(result).toEqual([]);
    const [url] = mockFetch.mock.calls[0] as [string];
    // api.ts 的 API_BASE 在测试环境按 jsdom 回退解析，仅断言路径与查询参数
    expect(url).toContain("/user/calendar?from=2026-08-01&to=2026-08-31");
  });
});
