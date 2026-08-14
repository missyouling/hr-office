"use client";

import { useEffect, useMemo, useState } from "react";
import { Bell, ExternalLink } from "lucide-react";
import { fetchDormSites } from "@/lib/api";
import type { DormSite } from "@/lib/types";
import { buildSiteNotifications, type SiteNotification } from "@/lib/dorm-notifications";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";

type NotificationCenterProps = { open: boolean; onOpenChange: (open: boolean) => void };

export function NotificationCenter({ open, onOpenChange }: NotificationCenterProps) {
  const [sites, setSites] = useState<DormSite[]>([]);
  const [notifications, setNotifications] = useState<SiteNotification[]>([]);
  const [siteFilter, setSiteFilter] = useState("all");

  useEffect(() => {
    if (!open) return;
    const load = async () => {
      try {
        const nextSites = await fetchDormSites();
        setSites(nextSites);
        setNotifications(buildSiteNotifications(nextSites));
      } catch {
        setSites([]);
        setNotifications([]);
      }
    };
    load();
  }, [open]);

  useEffect(() => {
    const refresh = () => setNotifications(buildSiteNotifications(sites));
    window.addEventListener("storage", refresh);
    window.addEventListener("notification:refresh", refresh);
    return () => {
      window.removeEventListener("storage", refresh);
      window.removeEventListener("notification:refresh", refresh);
    };
  }, [sites]);

  const siteOptions = useMemo(() => sites.filter((site) => notifications.some((item) => item.siteId === site.id)), [sites, notifications]);
  const visibleNotifications = siteFilter === "all" ? notifications : notifications.filter((item) => String(item.siteId) === siteFilter);

  if (!open) return null;

  const openMemo = (item: SiteNotification) => {
    window.dispatchEvent(new CustomEvent("dock:open-site-memo", { detail: { siteId: item.siteId } }));
    onOpenChange(false);
  };

  return (
    <div className="fixed inset-0 z-40 flex items-start justify-end bg-black/20 p-4 pt-20" onClick={() => onOpenChange(false)}>
      <Card className="max-h-[min(680px,calc(100vh-6rem))] w-full max-w-xl overflow-hidden shadow-2xl" onClick={(event) => event.stopPropagation()}>
        <CardHeader className="border-b">
          <CardTitle className="flex items-center gap-2 text-base"><Bell className="h-4 w-4" />通知中心</CardTitle>
          <p className="text-sm text-muted-foreground">集中查看宿舍地点的待处理备忘录。</p>
        </CardHeader>
        <CardContent className="space-y-4 overflow-y-auto p-4">
          {notifications.length === 0 ? <p className="py-8 text-center text-sm text-muted-foreground">暂无待处理提醒。</p> : (
            <>
              <div className="flex items-center justify-between gap-3 text-sm text-muted-foreground">
                <span>共 {notifications.length} 条提醒</span>
                <Select value={siteFilter} onValueChange={setSiteFilter}>
                  <SelectTrigger className="h-8 w-44" aria-label="按地点筛选"><SelectValue placeholder="全部地点" /></SelectTrigger>
                  <SelectContent><SelectItem value="all">全部地点</SelectItem>{siteOptions.map((site) => <SelectItem key={site.id} value={String(site.id)}>{site.name}</SelectItem>)}</SelectContent>
                </Select>
              </div>
              {visibleNotifications.length === 0 ? <p className="py-6 text-center text-sm text-muted-foreground">当前筛选下暂无提醒。</p> : (
                <div className="space-y-2">{visibleNotifications.map((item) => (
                  <div key={item.id} className="flex items-start gap-3 rounded-lg border p-3">
                    <div className="min-w-0 flex-1"><p className="text-sm font-medium">{item.siteName}</p><p className="mt-1 whitespace-pre-wrap break-words text-xs text-muted-foreground">{item.memo.content}</p></div>
                    <Button variant="ghost" size="sm" onClick={() => openMemo(item)} aria-label={`查看${item.siteName}备忘录`}>查看<ExternalLink className="ml-1 h-3.5 w-3.5" /></Button>
                  </div>
                ))}</div>
              )}
            </>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
