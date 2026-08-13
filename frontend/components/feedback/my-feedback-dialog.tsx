"use client";

import { useCallback, useEffect, useState } from "react";
import { MessageCircleReply, MessageSquareText, ThumbsDown, ThumbsUp } from "lucide-react";
import { toast } from "sonner";
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { EmptyState } from "@/components/common/empty-state";
import { Accordion, AccordionContent, AccordionItem, AccordionTrigger } from "@/components/ui/accordion";
import { FeedbackAnswer } from "./feedback-answer";
import { FeedbackStatusBadge } from "./feedback-status-badge";
import { listMyFeedback, type ChatFeedback } from "@/lib/api";
import { FEEDBACK_VIEWED_STORAGE_KEY, getViewedReplies, isReplyUnread, markReplyViewed } from "@/lib/feedback";

const PAGE_SIZE = 20;

function formatDate(value: string) {
  return new Date(value).toLocaleString("zh-CN", { month: "2-digit", day: "2-digit", hour: "2-digit", minute: "2-digit" });
}

export function MyFeedbackDialog({ open, onOpenChange }: { open: boolean; onOpenChange: (open: boolean) => void }) {
  const [items, setItems] = useState<ChatFeedback[]>([]);
  const [page, setPage] = useState(1);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(false);
  const [viewed, setViewed] = useState<Record<string, string>>({});

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const data = await listMyFeedback(page);
      setItems(data.items ?? []);
      setTotal(data.total ?? 0);
    } catch (error) {
      toast.error(error instanceof Error ? error.message : "加载我的反馈失败");
    } finally {
      setLoading(false);
    }
  }, [page]);

  useEffect(() => {
    if (!open) return;
    setViewed(getViewedReplies(localStorage.getItem(FEEDBACK_VIEWED_STORAGE_KEY)));
    load();
  }, [open, load]);

  const viewReply = (item: ChatFeedback) => {
    const next = markReplyViewed(item, viewed);
    setViewed(next);
    localStorage.setItem(FEEDBACK_VIEWED_STORAGE_KEY, JSON.stringify(next));
    window.dispatchEvent(new CustomEvent("feedback:reply-viewed"));
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-h-[88vh] overflow-y-auto sm:max-w-[800px]">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2"><MessageSquareText className="h-5 w-5 text-primary" />我的反馈</DialogTitle>
          <DialogDescription>查看您对 AI 回答的评价、处理进度与管理员回复。</DialogDescription>
        </DialogHeader>

        {loading && items.length === 0 && <div className="py-16 text-center text-sm text-muted-foreground">正在加载反馈…</div>}
        {!loading && items.length === 0 && <EmptyState icon={<MessageSquareText />} title="还没有反馈" description="在 AI 问答中评价回答后，处理进度会显示在这里。" height="h-72" />}
        <Accordion type="multiple" className="space-y-3">
          {items.map((item) => {
            const unread = isReplyUnread(item, viewed);
            return (
              <AccordionItem key={item.id} value={String(item.id)} className={unread ? "border-primary/40 bg-primary/[0.03] shadow-sm" : "bg-card"}>
                <AccordionTrigger onClick={() => viewReply(item)} aria-label={`查看反馈：${item.question || item.comment || "反馈详情"}`}>
                  <div className="min-w-0 pr-3">
                    <div className="flex flex-wrap items-center gap-2">
                      {item.rating === "positive" ? <ThumbsUp className="h-4 w-4 text-primary" /> : <ThumbsDown className="h-4 w-4 text-destructive" />}
                      <FeedbackStatusBadge item={item} />
                      {unread && <Badge className="gap-1"><span className="h-1.5 w-1.5 rounded-full bg-primary-foreground" />新回复</Badge>}
                      <span className="text-xs font-normal text-muted-foreground">{formatDate(item.created_at)}</span>
                    </div>
                    <p className="mt-2 truncate text-sm font-medium">{item.question || item.comment || "AI 回答反馈"}</p>
                  </div>
                </AccordionTrigger>
                <AccordionContent className="space-y-4 border-t pt-4">
                  {item.comment && <section><p className="mb-1 text-xs font-medium text-muted-foreground">我的留言</p><p className="rounded-lg bg-muted/50 p-3">{item.comment}</p></section>}
                  <section><p className="mb-2 text-xs font-medium text-muted-foreground">原回答</p><FeedbackAnswer item={item} /></section>
                  {item.reply && <section className="rounded-xl border border-primary/20 bg-primary/[0.06] p-4"><p className="flex items-center gap-2 text-sm font-semibold text-primary"><MessageCircleReply className="h-4 w-4" />管理员回复</p><p className="mt-2 whitespace-pre-wrap text-sm leading-relaxed">{item.reply}</p>{item.replied_at && <p className="mt-2 text-xs text-muted-foreground">回复于 {formatDate(item.replied_at)}</p>}</section>}
                </AccordionContent>
              </AccordionItem>
            );
          })}
        </Accordion>

        {total > PAGE_SIZE && <div className="flex items-center justify-between border-t pt-4 text-sm text-muted-foreground"><span>共 {total} 条</span><div className="flex gap-2"><Button variant="outline" size="sm" disabled={page === 1} onClick={() => setPage((value) => value - 1)}>上一页</Button><Button variant="outline" size="sm" disabled={page * PAGE_SIZE >= total} onClick={() => setPage((value) => value + 1)}>下一页</Button></div></div>}
      </DialogContent>
    </Dialog>
  );
}
