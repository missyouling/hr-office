import type { DormSite } from "@/lib/types";

export const SITE_MEMO_STORAGE_KEY = "dorm_site_memos_v1";

export type SiteMemo = {
  id: string;
  date?: string;
  targetDate?: string;
  time?: string;
  content: string;
  priority?: "normal" | "urgent" | "low";
  createdAt?: string;
  completed?: boolean;
};

export type SiteNotification = {
  id: string;
  siteId: number;
  siteName: string;
  memo: SiteMemo;
};

export function readSiteMemos(storage: Storage | undefined = typeof window === "undefined" ? undefined : window.localStorage) {
  if (!storage) return {} as Record<string, SiteMemo[]>;
  try {
    const value: unknown = JSON.parse(storage.getItem(SITE_MEMO_STORAGE_KEY) ?? "{}");
    if (!value || typeof value !== "object" || Array.isArray(value)) return {};
    return Object.fromEntries(
      Object.entries(value).map(([siteId, memos]) => [
        siteId,
        Array.isArray(memos) ? memos.filter(isSiteMemo) : [],
      ]),
    );
  } catch {
    return {};
  }
}

function isSiteMemo(value: unknown): value is SiteMemo {
  if (!value || typeof value !== "object") return false;
  const memo = value as Partial<SiteMemo>;
  return typeof memo.id === "string" && typeof memo.content === "string";
}

export function getSiteNotificationCount(storage?: Storage) {
  return Object.values(readSiteMemos(storage)).reduce(
    (count, memos) => count + memos.filter((memo) => !memo.completed).length,
    0,
  );
}

export function buildSiteNotifications(sites: DormSite[], storage?: Storage) {
  const memos = readSiteMemos(storage);
  return sites.flatMap((site) =>
    (memos[String(site.id)] ?? [])
      .filter((memo) => !memo.completed)
      .map((memo) => ({
        id: `${site.id}-${memo.id}`,
        siteId: site.id,
        siteName: site.name,
        memo,
      })),
  );
}
