import type { ChatFeedback, FeedbackListParams, FeedbackRating, FeedbackStatus } from "./api";

export const FEEDBACK_VIEWED_STORAGE_KEY = "feedback-replies-viewed";

export const FEEDBACK_STATUS_LABELS: Record<FeedbackStatus, string> = {
  pending: "待处理",
  replied: "已回复",
  closed: "已关闭",
};

export function normalizeFeedbackStatus(item: Pick<ChatFeedback, "status" | "reply">): FeedbackStatus {
  if (item.status === "replied" || item.status === "closed") return item.status;
  return item.reply ? "replied" : "pending";
}

export function getViewedReplies(storageValue: string | null): Record<string, string> {
  if (!storageValue) return {};
  try {
    const parsed = JSON.parse(storageValue);
    return parsed && typeof parsed === "object" ? parsed as Record<string, string> : {};
  } catch {
    return {};
  }
}

export function isReplyUnread(item: ChatFeedback, viewed: Record<string, string>): boolean {
  if (!item.reply || normalizeFeedbackStatus(item) !== "replied") return false;
  return viewed[String(item.id)] !== (item.replied_at || item.updated_at);
}

export function markReplyViewed(item: ChatFeedback, viewed: Record<string, string>): Record<string, string> {
  if (!item.reply) return viewed;
  return { ...viewed, [String(item.id)]: item.replied_at || item.updated_at };
}

export function toFeedbackDateRange(startDate: string, endDate: string) {
  return {
    start_at: startDate ? new Date(`${startDate}T00:00:00`).toISOString() : undefined,
    end_at: endDate ? new Date(`${endDate}T23:59:59.999`).toISOString() : undefined,
  };
}

export function buildAdminFeedbackParams(filters: {
  page: number;
  rating: FeedbackRating | "all";
  status: FeedbackStatus | "all";
  startDate: string;
  endDate: string;
}): FeedbackListParams {
  return {
    page: filters.page,
    rating: filters.rating === "all" ? undefined : filters.rating,
    status: filters.status === "all" ? undefined : filters.status,
    ...toFeedbackDateRange(filters.startDate, filters.endDate),
  };
}
