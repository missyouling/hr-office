"use client";

import { Fragment, useCallback, useEffect, useState } from "react";
import { ChevronDown, ChevronLeft, ChevronRight, Loader2, Lock, MessageSquareReply, ThumbsDown, ThumbsUp } from "lucide-react";
import { toast } from "sonner";
import { listFeedback, replyFeedback, closeFeedback, fetchFeedbackStats, type ChatFeedback, type ChatFeedbackStats, type FeedbackRating, type FeedbackStatus } from "@/lib/api";
import { buildAdminFeedbackParams, toFeedbackDateRange } from "@/lib/feedback";
import { useAuth } from "@/lib/auth";
import { FeedbackStatusBadge } from "@/components/feedback/feedback-status-badge";
import { FeedbackAnswer } from "@/components/feedback/feedback-answer";
import { EmptyState } from "@/components/common/empty-state";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { Textarea } from "@/components/ui/textarea";
import { AlertDialog, AlertDialogAction, AlertDialogCancel, AlertDialogContent, AlertDialogDescription, AlertDialogFooter, AlertDialogHeader, AlertDialogTitle } from "@/components/ui/alert-dialog";

const PAGE_SIZE = 20;

function formatDate(value?: string) {
  return value ? new Date(value).toLocaleString("zh-CN", { month: "2-digit", day: "2-digit", hour: "2-digit", minute: "2-digit" }) : "-";
}

function Stats({ stats }: { stats: ChatFeedbackStats | null }) {
  const cards = [
    ["反馈总数", stats?.total ?? 0], ["好评率", `${((stats?.positive_rate ?? 0) * 100).toFixed(1)}%`],
    ["待处理", stats?.pending ?? 0], ["已回复", stats?.replied ?? 0], ["已关闭", stats?.closed ?? 0],
  ];
  return <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-5">{cards.map(([label, value]) => <Card key={label}><CardContent className="p-4"><p className="text-xs text-muted-foreground">{label}</p><p className="mt-1 text-2xl font-semibold tracking-tight">{value}</p></CardContent></Card>)}</div>;
}

export function FeedbackPanel() {
  const { hasPermission } = useAuth();
  const canView = hasPermission("users", "view");
  const canAdmin = hasPermission("users", "delete");
  const [rating, setRating] = useState<FeedbackRating | "all">("all");
  const [status, setStatus] = useState<FeedbackStatus | "all">("all");
  const [startDate, setStartDate] = useState("");
  const [endDate, setEndDate] = useState("");
  const [page, setPage] = useState(1);
  const [items, setItems] = useState<ChatFeedback[]>([]);
  const [total, setTotal] = useState(0);
  const [stats, setStats] = useState<ChatFeedbackStats | null>(null);
  const [loading, setLoading] = useState(false);
  const [expandedId, setExpandedId] = useState<number | null>(null);
  const [replyingId, setReplyingId] = useState<number | null>(null);
  const [replyText, setReplyText] = useState("");
  const [closingItem, setClosingItem] = useState<ChatFeedback | null>(null);

  const load = useCallback(async () => {
    if (!canView) return;
    setLoading(true);
    try {
      const params = buildAdminFeedbackParams({ page, rating, status, startDate, endDate });
      const [list, summary] = await Promise.all([listFeedback(params), fetchFeedbackStats(toFeedbackDateRange(startDate, endDate))]);
      setItems(list.items ?? []); setTotal(list.total ?? 0); setStats(summary);
    } catch (error) {
      toast.error(error instanceof Error ? error.message : "加载反馈失败");
    } finally { setLoading(false); }
  }, [canView, page, rating, status, startDate, endDate]);

  useEffect(() => { load(); }, [load]);
  const changeFilter = (change: () => void) => { change(); setPage(1); };

  const saveReply = async (id: number) => {
    try { await replyFeedback(id, replyText.trim()); toast.success("回复已保存，用户将在“我的反馈”中看到"); setReplyingId(null); setReplyText(""); load(); }
    catch (error) { toast.error(error instanceof Error ? error.message : "保存回复失败"); }
  };

  const confirmClose = async () => {
    if (!closingItem) return;
    try { await closeFeedback(closingItem.id); toast.success("反馈已关闭"); setClosingItem(null); load(); }
    catch (error) { toast.error(error instanceof Error ? error.message : "关闭反馈失败"); }
  };

  if (!canView) return <EmptyState icon={<Lock />} title="暂无查看权限" description="反馈管理仅对经理及以上角色开放。" />;
  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE));

  return <div className="mx-auto w-full max-w-[1180px] space-y-5 pb-6">
    <div><h1 className="text-2xl font-bold tracking-tight">AI 反馈闭环</h1><p className="mt-1 text-sm text-muted-foreground">跟进回答质量、用户留言与处理状态。</p></div>
    <Stats stats={stats} />
    <Card><CardContent className="grid gap-3 p-4 sm:grid-cols-2 lg:grid-cols-4">
      <Select value={rating} onValueChange={(value) => changeFilter(() => setRating(value as FeedbackRating | "all"))}><SelectTrigger aria-label="按评分筛选"><SelectValue /></SelectTrigger><SelectContent><SelectItem value="all">全部评分</SelectItem><SelectItem value="positive">好评</SelectItem><SelectItem value="negative">差评</SelectItem></SelectContent></Select>
      <Select value={status} onValueChange={(value) => changeFilter(() => setStatus(value as FeedbackStatus | "all"))}><SelectTrigger aria-label="按状态筛选"><SelectValue /></SelectTrigger><SelectContent><SelectItem value="all">全部状态</SelectItem><SelectItem value="pending">待处理</SelectItem><SelectItem value="replied">已回复</SelectItem><SelectItem value="closed">已关闭</SelectItem></SelectContent></Select>
      <Input type="date" value={startDate} max={endDate || undefined} onChange={(event) => changeFilter(() => setStartDate(event.target.value))} aria-label="开始日期" />
      <Input type="date" value={endDate} min={startDate || undefined} onChange={(event) => changeFilter(() => setEndDate(event.target.value))} aria-label="结束日期" />
    </CardContent></Card>
    <Card><CardHeader><CardTitle className="text-base">反馈列表 <span className="font-normal text-muted-foreground">· {total} 条</span></CardTitle></CardHeader><CardContent className="overflow-x-auto p-0">
      <Table><TableHeader><TableRow><TableHead>用户</TableHead><TableHead>评分</TableHead><TableHead>反馈</TableHead><TableHead>状态</TableHead><TableHead>时间</TableHead><TableHead className="text-right">操作</TableHead></TableRow></TableHeader>
      <TableBody>{loading && items.length === 0 && <TableRow><TableCell colSpan={6} className="h-32 text-center"><Loader2 className="mx-auto h-5 w-5 animate-spin" aria-label="正在加载" /></TableCell></TableRow>}
      {!loading && items.length === 0 && <TableRow><TableCell colSpan={6}><EmptyState title="没有符合条件的反馈" description="调整评分、状态或时间范围后再试。" height="h-48" /></TableCell></TableRow>}
      {items.map((item) => <Fragment key={item.id}><TableRow>
        <TableCell className="font-medium">{item.full_name || item.username || `用户 #${item.user_id}`}</TableCell>
        <TableCell>{item.rating === "positive" ? <Badge variant="outline" className="gap-1 text-primary"><ThumbsUp className="h-3 w-3" />好评</Badge> : <Badge variant="outline" className="gap-1 text-destructive"><ThumbsDown className="h-3 w-3" />差评</Badge>}</TableCell>
        <TableCell className="max-w-[320px]"><p className="truncate">{item.comment || "无文字反馈"}</p></TableCell><TableCell><FeedbackStatusBadge item={item} /></TableCell><TableCell className="text-xs text-muted-foreground">{formatDate(item.created_at)}</TableCell>
        <TableCell><div className="flex justify-end gap-1"><Button variant="ghost" size="sm" onClick={() => setExpandedId(expandedId === item.id ? null : item.id)} aria-expanded={expandedId === item.id}><ChevronDown className={`mr-1 h-4 w-4 transition-transform ${expandedId === item.id ? "rotate-180" : ""}`} />详情</Button>{canAdmin && <><Button variant="ghost" size="sm" onClick={() => { setExpandedId(item.id); setReplyingId(item.id); setReplyText(item.reply || ""); }}><MessageSquareReply className="mr-1 h-4 w-4" />回复</Button>{item.status !== "closed" && <Button variant="ghost" size="sm" className="text-destructive" onClick={() => setClosingItem(item)}>关闭</Button>}</>}</div></TableCell>
      </TableRow>{expandedId === item.id && <TableRow className="bg-muted/20"><TableCell colSpan={6}><div className="grid gap-5 p-3 lg:grid-cols-[minmax(0,1fr)_minmax(0,1.5fr)]"><section><p className="text-xs font-medium text-muted-foreground">用户提问</p><p className="mt-2 whitespace-pre-wrap">{item.question || "提问内容不可用"}</p>{item.comment && <><p className="mt-4 text-xs font-medium text-muted-foreground">反馈留言</p><p className="mt-2 whitespace-pre-wrap">{item.comment}</p></>}</section><section><p className="mb-2 text-xs font-medium text-muted-foreground">AI 原回答与来源</p><FeedbackAnswer item={item} /></section></div>{replyingId === item.id && canAdmin && <div className="mx-3 mb-3 space-y-2 rounded-xl border bg-card p-4"><Textarea value={replyText} onChange={(event) => setReplyText(event.target.value)} placeholder="输入给用户的回复…" aria-label="管理员回复" /><div className="flex justify-end gap-2"><Button variant="outline" size="sm" onClick={() => setReplyingId(null)}>取消</Button><Button size="sm" disabled={!replyText.trim()} onClick={() => saveReply(item.id)}>保存回复</Button></div></div>}</TableCell></TableRow>}</Fragment>)}</TableBody></Table>
      {totalPages > 1 && <div className="flex items-center justify-between border-t p-4 text-sm text-muted-foreground"><span>第 {page} / {totalPages} 页</span><div className="flex gap-2"><Button variant="outline" size="icon" disabled={page <= 1} onClick={() => setPage((value) => value - 1)} aria-label="上一页"><ChevronLeft className="h-4 w-4" /></Button><Button variant="outline" size="icon" disabled={page >= totalPages} onClick={() => setPage((value) => value + 1)} aria-label="下一页"><ChevronRight className="h-4 w-4" /></Button></div></div>}
    </CardContent></Card>
    <AlertDialog open={Boolean(closingItem)} onOpenChange={(open) => !open && setClosingItem(null)}><AlertDialogContent><AlertDialogHeader><AlertDialogTitle>确认关闭这条反馈？</AlertDialogTitle><AlertDialogDescription>关闭后状态将变为“已关闭”。该记录与管理员回复仍会保留，不会删除。</AlertDialogDescription></AlertDialogHeader><AlertDialogFooter><AlertDialogCancel>取消</AlertDialogCancel><AlertDialogAction className="bg-destructive text-destructive-foreground hover:bg-destructive/90" onClick={confirmClose}>确认关闭</AlertDialogAction></AlertDialogFooter></AlertDialogContent></AlertDialog>
  </div>;
}
