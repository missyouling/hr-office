import { describe, expect, it } from "vitest";
import { buildSiteNotifications, getSiteNotificationCount, readSiteMemos } from "./dorm-notifications";

describe("宿舍通知本地数据源", () => {
  it("没有本地数据时返回空态", () => {
    const storage = { getItem: () => null } as unknown as Storage;
    expect(readSiteMemos(storage)).toEqual({});
    expect(getSiteNotificationCount(storage)).toBe(0);
  });

  it("按地点构建未完成提醒并过滤无效数据", () => {
    const storage = {
      getItem: () => JSON.stringify({ "1": [{ id: "m1", content: "缴费", completed: false }, { id: "m2", content: "完成", completed: true }, { content: "无编号" }] }),
    } as unknown as Storage;
    const sites = [{ id: 1, name: "一号楼" }] as Parameters<typeof buildSiteNotifications>[0];
    expect(buildSiteNotifications(sites, storage)).toMatchObject([{ siteId: 1, siteName: "一号楼", memo: { id: "m1" } }]);
    expect(getSiteNotificationCount(storage)).toBe(1);
  });
});
