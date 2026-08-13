import { describe, expect, it } from "vitest";
import { buildFeedbackQuery, mapFeedbackPayload } from "./api";
import { buildAdminFeedbackParams, getViewedReplies, isReplyUnread, markReplyViewed, normalizeFeedbackStatus } from "./feedback";
import type { ChatFeedback } from "./api";

const repliedFeedback = {
  id: 7,
  status: "replied",
  reply: "已调整回答逻辑",
  replied_at: "2026-08-12T08:00:00Z",
  updated_at: "2026-08-12T08:00:00Z",
} as ChatFeedback;

describe("反馈 API 查询映射", () => {
  it("映射评分、状态、时间与用户筛选", () => {
    const query = buildFeedbackQuery({ page: 2, rating: "negative", status: "pending", start_at: "2026-08-01T00:00:00Z", end_at: "2026-08-12T23:59:59Z", user_id: 9 });
    const params = new URLSearchParams(query.slice(1));
    expect(Object.fromEntries(params)).toEqual({ page: "2", rating: "negative", status: "pending", start_at: "2026-08-01T00:00:00Z", end_at: "2026-08-12T23:59:59Z", user_id: "9" });
  });

  it("忽略未设置的筛选条件", () => {
    expect(buildFeedbackQuery({ page: 1 })).toBe("?page=1");
    expect(buildFeedbackQuery()).toBe("");
  });

  it("将 SSE 返回的数字消息 ID 映射为后端存储字段", () => {
    expect(mapFeedbackPayload({ message_id: 123, rating: "positive" })).toEqual({
      message_id: "123",
      rating: "positive",
    });
  });
});

describe("用户反馈回复状态", () => {
  it("兼容旧记录按回复内容归一为已回复", () => {
    expect(normalizeFeedbackStatus({ status: "pending", reply: "已有回复" })).toBe("replied");
  });

  it("记录查看版本后不再标记为新回复", () => {
    expect(isReplyUnread(repliedFeedback, {})).toBe(true);
    const viewed = markReplyViewed(repliedFeedback, {});
    expect(isReplyUnread(repliedFeedback, viewed)).toBe(false);
  });

  it("损坏的本地记录安全回退为空对象", () => {
    expect(getViewedReplies("{broken")).toEqual({});
  });
});

describe("管理员反馈状态筛选", () => {
  it("全部选项不发送评分与状态参数", () => {
    expect(buildAdminFeedbackParams({ page: 1, rating: "all", status: "all", startDate: "", endDate: "" })).toEqual({
      page: 1, rating: undefined, status: undefined, start_at: undefined, end_at: undefined,
    });
  });

  it("将状态和日期边界映射为接口参数", () => {
    const result = buildAdminFeedbackParams({ page: 3, rating: "negative", status: "closed", startDate: "2026-08-01", endDate: "2026-08-12" });
    expect(result.page).toBe(3);
    expect(result.rating).toBe("negative");
    expect(result.status).toBe("closed");
    expect(result.start_at).toBe(new Date("2026-08-01T00:00:00").toISOString());
    expect(result.end_at).toBe(new Date("2026-08-12T23:59:59.999").toISOString());
  });
});
