"use client";

import { useEffect, useState } from "react";
import { toast } from "sonner";
import { ThumbsUp, ThumbsDown, MessageSquareReply, Loader2, ChevronLeft, ChevronRight } from "lucide-react";

import { usePermission } from "@/lib/rbac";
import {
  listFeedback,
  replyFeedback,
  fetchFeedbackStats,
  type ChatFeedback,
  type ChatFeedbackStats,
} from "@/lib/api";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Textarea } from "@/components/ui/textarea";
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs";

const pageSize = 20;

function formatDateTime(value?: string): string {
  if (!value) return "-";
  return new Date(value).toLocaleString("zh-CN", {
    year: "numeric",
    month: "2-digit",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
  });
}

export function FeedbackPanel() {
  const { isAdmin } = usePermission();
  const [rating, setRating] = useState<string>("all");
  const [page, setPage] = useState(1);
  const [items, setItems] = useState<ChatFeedback[]>([]);
  const [total, setTotal] = useState(0);
  const [stats, setStats] = useState<ChatFeedbackStats | null>(null);
  const [loading, setLoading] = useState(false);
  const [replyingId, setReplyingId] = useState<number | null>(null);
  const [replyText, setReplyText] = useState("");

  const loadList = async () => {
    setLoading(true);
    try {
      const params: { rating?: string; page?: number } = { page };
      if (rating !== "all") params.rating = rating;
      const data = await listFeedback(params);
      setItems(data.items ?? []);
      setTotal(data.total ?? 0);
    } catch (error) {
      console.error("加载反馈列表失败:", error);
      toast.error(error instanceof Error ? error.message : "加载反馈列表失败");
    } finally {
      setLoading(false);
    }
  };

  const loadStats = async () => {
    try {
      const data = await fetchFeedbackStats();
      setStats(data);
    } catch (error) {
      console.error("加载反馈统计失败:", error);
    }
  };

  useEffect(() => {
    loadList();
    loadStats();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [rating, page]);

  const handleStartReply = (item: ChatFeedback) => {
    setReplyingId(item.id);
    setReplyText(item.reply ?? "");
  };

  const handleCancelReply = () => {
    setReplyingId(null);
    setReplyText("");
  };

  const handleSubmitReply = async (id: number) => {
    try {
      await replyFeedback(id, replyText.trim());
      toast.success("回复已保存");
      setReplyingId(null);
      setReplyText("");
      loadList();
      loadStats();
    } catch (error) {
      console.error("保存回复失败:", error);
      toast.error(error instanceof Error ? error.message : "保存回复失败");
    }
  };

  if (!isAdmin) {
    return (
      <div className="flex min-h-[400px] items-center justify-center text-muted-foreground">
        暂无权限查看该页面
      </div>
    );
  }

  const totalPages = Math.max(1, Math.ceil(total / pageSize));

  return (
    <div className="mx-auto w-full max-w-[1180px] space-y-6 pb-6">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-bold tracking-tight">AI 反馈闭环</h1>
      </div>

      {/* 统计概览 */}
      <div className="grid gap-4 md:grid-cols-4">
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium text-muted-foreground">反馈总数</CardTitle>
          </CardHeader>
          <CardContent>
            <p className="text-3xl font-bold">{stats?.total ?? 0}</p>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium text-muted-foreground">好评数</CardTitle>
          </CardHeader>
          <CardContent>
            <p className="text-3xl font-bold text-emerald-600">{stats?.positive ?? 0}</p>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium text-muted-foreground">差评数</CardTitle>
          </CardHeader>
          <CardContent>
            <p className="text-3xl font-bold text-rose-600">{stats?.negative ?? 0}</p>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium text-muted-foreground">好评率</CardTitle>
          </CardHeader>
          <CardContent>
            <p className="text-3xl font-bold text-blue-600">
              {stats ? `${(stats.positive_rate * 100).toFixed(1)}%` : "0%"}
            </p>
          </CardContent>
        </Card>
      </div>

      {/* 筛选 */}
      <Tabs value={rating} onValueChange={(value) => { setRating(value); setPage(1); }}>
        <TabsList>
          <TabsTrigger value="all">全部</TabsTrigger>
          <TabsTrigger value="positive">好评</TabsTrigger>
          <TabsTrigger value="negative">差评</TabsTrigger>
        </TabsList>
      </Tabs>

      {/* 反馈列表 */}
      <Card>
        <CardHeader>
          <CardTitle>反馈列表</CardTitle>
        </CardHeader>
        <CardContent className="p-0">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead className="w-[120px]">用户</TableHead>
                <TableHead className="w-[80px]">评分</TableHead>
                <TableHead>内容</TableHead>
                <TableHead className="w-[100px]">状态</TableHead>
                <TableHead className="w-[160px]">时间</TableHead>
                <TableHead className="w-[120px]">操作</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {loading && items.length === 0 && (
                <TableRow>
                  <TableCell colSpan={6} className="h-32 text-center text-muted-foreground">
                    <Loader2 className="mx-auto h-6 w-6 animate-spin" />
                  </TableCell>
                </TableRow>
              )}
              {!loading && items.length === 0 && (
                <TableRow>
                  <TableCell colSpan={6} className="h-32 text-center text-muted-foreground">
                    暂无反馈数据
                  </TableCell>
                </TableRow>
              )}
              {items.map((item) => (
                <>
                  <TableRow key={item.id}>
                    <TableCell className="font-medium">
                      {item.full_name || item.username || `用户 #${item.user_id}`}
                    </TableCell>
                    <TableCell>
                      {item.rating === "positive" ? (
                        <Badge variant="outline" className="gap-1 border-emerald-200 text-emerald-700">
                          <ThumbsUp className="h-3 w-3" />
                          好评
                        </Badge>
                      ) : (
                        <Badge variant="outline" className="gap-1 border-rose-200 text-rose-700">
                          <ThumbsDown className="h-3 w-3" />
                          差评
                        </Badge>
                      )}
                    </TableCell>
                    <TableCell className="max-w-md">
                      <div className="space-y-1">
                        <p className="truncate text-sm">{item.comment || "无文字反馈"}</p>
                        {item.reply && (
                          <p className="text-xs text-muted-foreground">
                            <span className="font-medium">回复：</span>
                            {item.reply}
                          </p>
                        )}
                      </div>
                    </TableCell>
                    <TableCell>
                      {item.reply ? (
                        <Badge className="bg-blue-100 text-blue-700 hover:bg-blue-100">已回复</Badge>
                      ) : (
                        <Badge variant="secondary">待回复</Badge>
                      )}
                    </TableCell>
                    <TableCell className="text-xs text-muted-foreground">
                      {formatDateTime(item.created_at)}
                    </TableCell>
                    <TableCell>
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={() => handleStartReply(item)}
                        className="h-8 px-2 text-blue-600 hover:bg-blue-50 hover:text-blue-700"
                      >
                        <MessageSquareReply className="mr-1 h-3.5 w-3.5" />
                        {item.reply ? "编辑回复" : "回复"}
                      </Button>
                    </TableCell>
                  </TableRow>
                  {replyingId === item.id && (
                    <TableRow className="bg-blue-50/50">
                      <TableCell colSpan={6} className="py-4">
                        <div className="space-y-3">
                          <Textarea
                            placeholder="请输入管理员回复..."
                            value={replyText}
                            onChange={(e) => setReplyText(e.target.value)}
                            className="min-h-[80px] resize-none bg-white"
                          />
                          <div className="flex justify-end gap-2">
                            <Button variant="outline" size="sm" onClick={handleCancelReply}>
                              取消
                            </Button>
                            <Button
                              size="sm"
                              onClick={() => handleSubmitReply(item.id)}
                              disabled={!replyText.trim()}
                              className="bg-blue-600 hover:bg-blue-700"
                            >
                              保存回复
                            </Button>
                          </div>
                        </div>
                      </TableCell>
                    </TableRow>
                  )}
                </>
              ))}
            </TableBody>
          </Table>

          {/* 分页 */}
          {totalPages > 1 && (
            <div className="flex items-center justify-between border-t px-4 py-3">
              <p className="text-sm text-muted-foreground">
                共 {total} 条，第 {page} / {totalPages} 页
              </p>
              <div className="flex gap-2">
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => setPage((p) => Math.max(1, p - 1))}
                  disabled={page <= 1}
                >
                  <ChevronLeft className="h-4 w-4" />
                </Button>
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => setPage((p) => Math.min(totalPages, p + 1))}
                  disabled={page >= totalPages}
                >
                  <ChevronRight className="h-4 w-4" />
                </Button>
              </div>
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
