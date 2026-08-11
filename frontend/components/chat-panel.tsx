"use client";

import {
  useState,
  useRef,
  useEffect,
  useCallback,
  useMemo,
} from "react";
import {
  MessageSquare,
  Send,
  Bot,
  User,
  X,
  Plus,
  Trash2,
  Square,
  FileText,
  Search,
  ChevronRight,
  PanelLeftClose,
  PanelLeftOpen,
  Clock,
  Sparkles,
  ThumbsUp,
  ThumbsDown,
} from "lucide-react";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Card } from "@/components/ui/card";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Textarea } from "@/components/ui/textarea";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from "@/components/ui/dialog";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Skeleton } from "@/components/ui/skeleton";
import { MarkdownContent } from "@/lib/markdown";
import {
  chatKnowledgeStream,
  fetchSessions,
  deleteChatSession,
  submitFeedback,
} from "@/lib/api";
import type { SearchResult, ChatSession } from "@/lib/api";
import { knowledgeApi } from "@/lib/api-knowledge";
import type { KnowledgeBase } from "@/lib/api-knowledge";
import { useAuth } from "@/lib/auth";
import { getSourceModuleLabel } from "@/components/knowledge/utils";

// ─── 类型定义 ────────────────────────────────────────────

interface Message {
  id: string;
  role: "user" | "assistant";
  content: string;
  sources?: SearchResult[];
  timestamp: Date;
}

// ─── 工具函数 ────────────────────────────────────────────

function getTimeLabel(dateStr: string): string {
  const d = new Date(dateStr);
  const now = new Date();
  const diffMs = now.getTime() - d.getTime();
  const diffMin = Math.floor(diffMs / 60000);
  if (diffMin < 1) return "刚刚";
  if (diffMin < 60) return `${diffMin} 分钟前`;
  const diffHr = Math.floor(diffMin / 60);
  if (diffHr < 24) return `${diffHr} 小时前`;
  const diffDay = Math.floor(diffHr / 24);
  if (diffDay < 7) return `${diffDay} 天前`;
  return d.toLocaleDateString("zh-CN");
}

// ─── 主组件 ──────────────────────────────────────────────

export function ChatPanel() {
  const [isOpen, setIsOpen] = useState(false);
  const [messages, setMessages] = useState<Message[]>([]);
  const [inputValue, setInputValue] = useState("");
  const [isLoading, setIsLoading] = useState(false);
  const [sessionId, setSessionId] = useState<string>("");
  const [sessions, setSessions] = useState<ChatSession[]>([]);
  const [activeSessionId, setActiveSessionId] = useState<number | null>(null);
  const [showSessions, setShowSessions] = useState(true);
  const [sessionsLoading, setSessionsLoading] = useState(false);
  const [ratedMap, setRatedMap] = useState<Record<string, "positive" | "negative">>({});
  const [feedbackDialog, setFeedbackDialog] = useState<{
    open: boolean;
    messageId: string;
    rating: "positive" | "negative";
    comment: string;
  }>({ open: false, messageId: "", rating: "positive", comment: "" });

  // ─── 知识库范围选择器 ──────────────────────────────────
  const [selectedKbId, setSelectedKbId] = useState<number | null>(null); // null=全部可见KB
  const [kbs, setKbs] = useState<KnowledgeBase[]>([]);
  const [kbsLoading, setKbsLoading] = useState(false);
  const { user } = useAuth();
  const abortRef = useRef<AbortController | null>(null);
  const scrollRef = useRef<HTMLDivElement>(null);
  const inputRef = useRef<HTMLInputElement>(null);

  // ─── 加载会话列表 ──────────────────────────────────────

  const loadSessions = useCallback(async () => {
    setSessionsLoading(true);
    try {
      const list = await fetchSessions();
      setSessions(list);
    } catch (error) {
      console.error("加载会话列表失败:", error);
    } finally {
      setSessionsLoading(false);
    }
  }, []);

  // 打开面板时加载会话列表
  useEffect(() => {
    if (isOpen) {
      loadSessions();
      // 自动聚焦输入框
      setTimeout(() => inputRef.current?.focus(), 100);
    }
  }, [isOpen, loadSessions]);

  // 拉取当前用户可见的知识库列表
  useEffect(() => {
    if (!user) return;
    setKbsLoading(true);
    knowledgeApi
      .list()
      .then((data) => setKbs(data.items))
      .catch(() => {
        toast.error("加载知识库列表失败");
      })
      .finally(() => setKbsLoading(false));
  }, [user]);

  // ─── 自动滚动到底部 ────────────────────────────────────

  useEffect(() => {
    if (scrollRef.current) {
      scrollRef.current.scrollIntoView({ behavior: "smooth" });
    }
  }, [messages]);

  // ─── 新建会话 ──────────────────────────────────────────

  const handleNewSession = useCallback(() => {
    setMessages([]);
    setSessionId("");
    setActiveSessionId(null);
    setInputValue("");
    inputRef.current?.focus();
  }, []);

  // ─── 切换会话 ──────────────────────────────────────────

  const handleSelectSession = useCallback(
    (session: ChatSession) => {
      setActiveSessionId(session.id);
      setSessionId(session.session_id);
      setMessages([]);
      setInputValue("");
      inputRef.current?.focus();
    },
    [],
  );

  // ─── 删除会话 ──────────────────────────────────────────

  const handleDeleteSession = useCallback(
    async (e: React.MouseEvent, sess: ChatSession) => {
      e.stopPropagation();
      try {
        await deleteChatSession(sess.id);
        toast.success("会话已删除");
        if (activeSessionId === sess.id) {
          handleNewSession();
        }
        loadSessions();
      } catch (error) {
        console.error("删除会话失败:", error);
        toast.error("删除会话失败");
      }
    },
    [activeSessionId, handleNewSession, loadSessions],
  );

  // ─── 发送消息 ──────────────────────────────────────────

  const handleSendMessage = useCallback(async () => {
    if (!inputValue.trim() || isLoading) return;

    const question = inputValue.trim();
    const userMessage: Message = {
      id: `msg-${Date.now()}`,
      role: "user",
      content: question,
      timestamp: new Date(),
    };

    setMessages((prev) => [...prev, userMessage]);
    setInputValue("");
    setIsLoading(true);

    // 创建 AbortController 用于停止生成
    const abortController = new AbortController();
    abortRef.current = abortController;

    // 预创建 AI 消息占位，SSE 流式填充内容
    const assistantId = `msg-${Date.now()}-ai`;
    const assistantMessage: Message = {
      id: assistantId,
      role: "assistant",
      content: "",
      timestamp: new Date(),
    };
    setMessages((prev) => [...prev, assistantMessage]);

    await chatKnowledgeStream(
      question,
      sessionId,
      // onToken — 逐词追加到 AI 消息的 content 中
      (token) => {
        setMessages((prev) =>
          prev.map((m) =>
            m.id === assistantId
              ? { ...m, content: m.content + token }
              : m,
          ),
        );
      },
      // onDone — 流式完成
      () => {
        setSessionsLoading(false);
        setIsLoading(false);
        abortRef.current = null;
        // 流式结束后刷新会话列表以获取新会话
        loadSessions();
        // 如果还没有 session_id，从列表中推断最新的
        if (!sessionId) {
          fetchSessions().then((list) => {
            if (list.length > 0) {
              const latest = list[0];
              setSessionId(latest.session_id);
              setActiveSessionId(latest.id);
            }
          }).catch(() => {});
        }
      },
      // onError — 错误处理
      (error) => {
        console.error("流式聊天出错:", error);
        toast.error(error);
        setMessages((prev) =>
          prev.map((m) =>
            m.id === assistantId
              ? { ...m, content: m.content || `错误: ${error}` }
              : m,
          ),
        );
        setIsLoading(false);
        abortRef.current = null;
      },
      abortController.signal,
      selectedKbId,
    );
  }, [inputValue, isLoading, sessionId, loadSessions, selectedKbId]);

  // ─── 停止生成 ──────────────────────────────────────────

  const handleStopGeneration = useCallback(() => {
    if (abortRef.current) {
      abortRef.current.abort();
      abortRef.current = null;
      setIsLoading(false);
    }
  }, []);

  // ─── 反馈 ──────────────────────────────────────────────

  const openFeedbackDialog = useCallback((messageId: string, rating: "positive" | "negative") => {
    setFeedbackDialog({ open: true, messageId, rating, comment: "" });
  }, []);

  const closeFeedbackDialog = useCallback(() => {
    setFeedbackDialog((prev) => ({ ...prev, open: false }));
  }, []);

  const handleSubmitFeedback = useCallback(async () => {
    if (!feedbackDialog.messageId) return;
    try {
      await submitFeedback({
        message_id: feedbackDialog.messageId,
        session_id: sessionId || undefined,
        rating: feedbackDialog.rating,
        comment: feedbackDialog.comment.trim() || undefined,
      });
      setRatedMap((prev) => ({ ...prev, [feedbackDialog.messageId]: feedbackDialog.rating }));
      toast.success("反馈已提交，感谢您的建议");
      closeFeedbackDialog();
    } catch (error) {
      console.error("提交反馈失败:", error);
      toast.error(error instanceof Error ? error.message : "提交反馈失败");
    }
  }, [feedbackDialog, sessionId, closeFeedbackDialog]);

  // ─── 键盘事件 ──────────────────────────────────────────

  const handleKeyDown = useCallback(
    (e: React.KeyboardEvent<HTMLInputElement>) => {
      if (e.key === "Enter" && !e.shiftKey) {
        e.preventDefault();
        handleSendMessage();
      }
    },
    [handleSendMessage],
  );

  // ─── 会话列表组件（内嵌） ──────────────────────────────

  const sortedSessions = useMemo(() => {
    return [...sessions].sort((a, b) => {
      if (a.is_pinned !== b.is_pinned) return a.is_pinned ? -1 : 1;
      return new Date(b.updated_at).getTime() - new Date(a.updated_at).getTime();
    });
  }, [sessions]);

  // ─── 浮动触发按钮 ──────────────────────────────────────

  if (!isOpen) {
    return (
      <div className="fixed bottom-6 right-6 z-40">
        <Button
          onClick={() => setIsOpen(true)}
          size="lg"
          className="rounded-full w-14 h-14 shadow-lg bg-primary hover:bg-primary/90 transition-all hover:scale-105 active:scale-95"
        >
          <MessageSquare className="w-6 h-6" />
        </Button>
      </div>
    );
  }

  return (
    <>
      {/* 背景遮罩 */}
      <div
        className="fixed inset-0 bg-black/20 z-40 transition-opacity"
        onClick={() => setIsOpen(false)}
      />

      {/* 侧滑面板 */}
      <Card className="fixed inset-y-0 right-0 z-50 w-[880px] max-w-[95vw] flex flex-col shadow-2xl bg-card rounded-l-2xl border-l border-border animate-in slide-in-from-right duration-300">
        {/* ─── 顶栏 ──────────────────────────────────────── */}
        <div className="flex items-center justify-between px-5 py-3 border-b bg-gradient-to-r from-muted to-background">
          <div className="flex items-center gap-3">
            <div className="w-8 h-8 rounded-lg bg-primary flex items-center justify-center">
              <Sparkles className="w-4 h-4 text-primary-foreground" />
            </div>
            <div>
              <h2 className="font-semibold text-sm text-foreground">AI 知识库问答</h2>
              <p className="text-xs text-muted-foreground">基于档案与规章制度检索回答</p>
            </div>
          </div>
          <div className="flex items-center gap-1">
            <Button
              variant="ghost"
              size="sm"
              onClick={() => setShowSessions(!showSessions)}
              className="h-8 px-2 text-muted-foreground hover:text-foreground"
              title={showSessions ? "隐藏会话列表" : "显示会话列表"}
            >
              {showSessions ? (
                <PanelLeftClose className="w-4 h-4" />
              ) : (
                <PanelLeftOpen className="w-4 h-4" />
              )}
            </Button>
            <Button
              variant="ghost"
              size="sm"
              onClick={() => setIsOpen(false)}
              className="h-8 w-8 p-0 text-muted-foreground hover:text-foreground"
            >
              <X className="w-4 h-4" />
            </Button>
          </div>
        </div>

        {/* ─── 主体区域 ──────────────────────────────────── */}
        <div className="flex flex-1 min-h-0">
          {/* ── 会话列表侧栏 ──────────────────────────────── */}
          {showSessions && (
            <div className="w-60 border-r flex flex-col bg-muted/30 shrink-0">
              {/* 顶部操作栏 */}
              <div className="p-3 border-b flex items-center justify-between">
                <span className="text-xs font-medium text-muted-foreground flex items-center gap-1.5">
                  <Clock className="w-3 h-3" />
                  历史会话
                </span>
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={handleNewSession}
                  className="h-7 w-7 p-0 text-primary hover:text-primary hover:bg-accent"
                  title="新建会话"
                >
                  <Plus className="w-4 h-4" />
                </Button>
              </div>

              {/* 会话列表 */}
              <ScrollArea className="flex-1">
                <div className="p-2 space-y-0.5">
                  {/* 当前新会话入口 */}
                  {!activeSessionId && (
                    <div className="flex items-center gap-2 px-3 py-2 rounded-lg bg-accent border border-border cursor-default">
                      <div className="w-6 h-6 rounded-full bg-primary flex items-center justify-center shrink-0">
                        <Sparkles className="w-3 h-3 text-primary-foreground" />
                      </div>
                      <div className="min-w-0 flex-1">
                        <p className="text-xs font-medium text-primary truncate">
                          新对话
                        </p>
                        <p className="text-[10px] text-muted-foreground">当前</p>
                      </div>
                    </div>
                  )}

                  {sessionsLoading && sortedSessions.length === 0 && (
                    <div className="px-3 py-6 text-center">
                      <div className="flex justify-center gap-1">
                        <div className="w-1.5 h-1.5 bg-muted-foreground rounded-full animate-bounce" />
                        <div className="w-1.5 h-1.5 bg-muted-foreground rounded-full animate-bounce delay-100" />
                        <div className="w-1.5 h-1.5 bg-muted-foreground rounded-full animate-bounce delay-200" />
                      </div>
                    </div>
                  )}

                  {!sessionsLoading && sortedSessions.length === 0 && (
                    <div className="px-3 py-8 text-center">
                      <Search className="w-8 h-8 mx-auto mb-2 text-muted-foreground/30" />
                      <p className="text-xs text-muted-foreground">暂无会话记录</p>
                      <p className="text-[10px] text-muted-foreground/50 mt-1">
                        开始提问即可创建
                      </p>
                    </div>
                  )}

                  {sortedSessions.map((sess) => (
                    <div
                      key={sess.id}
                      onClick={() => handleSelectSession(sess)}
                      className={`group flex items-center gap-2 px-3 py-2 rounded-lg cursor-pointer transition-all
                        ${
                          activeSessionId === sess.id
                            ? "bg-accent border border-border"
                            : "hover:bg-accent/50 border border-transparent"
                        }`}
                    >
                      <div
                        className={`w-6 h-6 rounded-full flex items-center justify-center shrink-0
                          ${
                            activeSessionId === sess.id
                              ? "bg-primary text-primary-foreground"
                              : "bg-muted text-muted-foreground"
                          }`}
                      >
                        <MessageSquare className="w-3 h-3" />
                      </div>
                      <div className="min-w-0 flex-1">
                        <p
                          className={`text-xs font-medium truncate
                            ${activeSessionId === sess.id ? "text-primary" : "text-foreground"}`}
                        >
                          {sess.title || "未命名会话"}
                        </p>
                        <p className="text-[10px] text-muted-foreground">
                          {getTimeLabel(sess.updated_at)}
                        </p>
                      </div>
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={(e) => handleDeleteSession(e, sess)}
                        className="h-6 w-6 p-0 opacity-0 group-hover:opacity-100 text-muted-foreground hover:text-destructive hover:bg-accent shrink-0 transition-all"
                        title="删除会话"
                      >
                        <Trash2 className="w-3 h-3" />
                      </Button>
                    </div>
                  ))}
                </div>
              </ScrollArea>

              {/* 底部快捷操作 */}
              <div className="p-3 border-t">
                <Button
                  variant="outline"
                  size="sm"
                  onClick={handleNewSession}
                  className="w-full text-xs h-8 border-primary/20 text-primary hover:bg-accent hover:text-primary"
                >
                  <Plus className="w-3.5 h-3.5 mr-1.5" />
                  新建对话
                </Button>
              </div>
            </div>
          )}

          {/* ── 聊天主体区域 ──────────────────────────────── */}
          <div className="flex-1 flex flex-col min-w-0">
            <ScrollArea className="flex-1 px-5">
              <div className="max-w-3xl mx-auto py-5 space-y-5">
                {/* 空状态 */}
                {messages.length === 0 && (
                  <div className="flex flex-col items-center justify-center py-16 text-center">
                    <div className="w-16 h-16 rounded-2xl bg-gradient-to-br from-muted to-muted/50 flex items-center justify-center mb-4 shadow-sm">
                      <Sparkles className="w-8 h-8 text-primary" />
                    </div>
                    <h3 className="text-base font-semibold text-foreground mb-1">
                      知识库 AI 问答
                    </h3>
                    <p className="text-sm text-muted-foreground max-w-xs">
                      基于档案文档和规章制度，为您提供精准回答
                    </p>
                    <div className="flex flex-wrap gap-2 mt-5 justify-center">
                      {[
                        "本月社保缴费标准是什么？",
                        "员工入职需要哪些档案材料？",
                        "公积金提取流程是怎样的？",
                      ].map((q) => (
                        <button
                          key={q}
                          onClick={() => {
                            setInputValue(q);
                            inputRef.current?.focus();
                          }}
                          className="px-3 py-1.5 text-xs rounded-full border border-border text-muted-foreground hover:border-primary/30 hover:text-primary hover:bg-accent transition-all"
                        >
                          {q}
                        </button>
                      ))}
                    </div>
                  </div>
                )}

                {/* 消息列表 */}
                {messages.map((message) => (
                  <div
                    key={message.id}
                    className={`flex gap-3 ${
                      message.role === "user" ? "justify-end" : "justify-start"
                    }`}
                  >
                    {/* AI 头像 */}
                    {message.role === "assistant" && (
                      <div className="flex-shrink-0 w-8 h-8 rounded-full bg-gradient-to-br from-primary to-primary/80 flex items-center justify-center mt-0.5 shadow-sm">
                        <Bot className="w-4 h-4 text-primary-foreground" />
                      </div>
                    )}

                    <div
                      className={`max-w-[80%] ${
                        message.role === "user"
                          ? "order-first"
                          : ""
                      }`}
                    >
                      {/* 用户消息气泡 */}
                      {message.role === "user" ? (
                        <div className="px-4 py-2.5 rounded-2xl rounded-br-md bg-primary text-primary-foreground shadow-sm">
                          <p className="text-sm whitespace-pre-wrap break-words leading-relaxed">
                            {message.content}
                          </p>
                        </div>
                      ) : (
                        /* AI 消息 */
                        <div className="space-y-2">
                          <div className="px-4 py-3 rounded-2xl rounded-bl-md bg-card border border-border shadow-sm">
                            {message.content ? (
                              <MarkdownContent content={message.content} />
                            ) : (
                              <div className="flex gap-1.5 py-1">
                                <div className="w-2 h-2 bg-primary rounded-full animate-bounce" />
                                <div className="w-2 h-2 bg-primary rounded-full animate-bounce delay-100" />
                                <div className="w-2 h-2 bg-primary rounded-full animate-bounce delay-200" />
                              </div>
                            )}
                          </div>

                          {/* 点赞/点踩 */}
                          {message.content && (
                            <div className="flex items-center gap-1 px-1">
                              <button
                                onClick={() => openFeedbackDialog(message.id, "positive")}
                                title="有帮助"
                                className={`p-1.5 rounded-md transition-colors ${
                                  ratedMap[message.id] === "positive"
                                    ? "text-primary bg-accent"
                                    : "text-muted-foreground hover:text-primary hover:bg-accent"
                                }`}
                              >
                                <ThumbsUp className="w-3.5 h-3.5" />
                              </button>
                              <button
                                onClick={() => openFeedbackDialog(message.id, "negative")}
                                title="没有帮助"
                                className={`p-1.5 rounded-md transition-colors ${
                                  ratedMap[message.id] === "negative"
                                    ? "text-primary bg-accent"
                                    : "text-muted-foreground hover:text-primary hover:bg-accent"
                                }`}
                              >
                                <ThumbsDown className="w-3.5 h-3.5" />
                              </button>
                            </div>
                          )}

                          {/* 引用溯源 */}
                          {message.sources &&
                            message.sources.length > 0 && (
                              <SourcesCard sources={message.sources} />
                            )}
                        </div>
                      )}

                      {/* 时间戳 */}
                      <p className="text-[10px] text-muted-foreground mt-1 px-1">
                        {message.timestamp.toLocaleTimeString("zh-CN", {
                          hour: "2-digit",
                          minute: "2-digit",
                        })}
                      </p>
                    </div>

                    {/* 用户头像 */}
                    {message.role === "user" && (
                      <div className="flex-shrink-0 w-8 h-8 rounded-full bg-muted flex items-center justify-center mt-0.5">
                        <User className="w-4 h-4 text-muted-foreground" />
                      </div>
                    )}
                  </div>
                ))}

                {/* 滚动锚点 */}
                <div ref={scrollRef} />
              </div>
            </ScrollArea>

            {/* ─── 底部输入栏 ─────────────────────────────── */}
            <div className="border-t p-4 bg-card">
              <div className="max-w-3xl mx-auto">
                {/* 知识库范围选择器 */}
                <div className="flex items-center gap-2 mb-2">
                  <span className="text-xs text-muted-foreground shrink-0">搜索范围</span>
                  {kbsLoading ? (
                    <Skeleton className="h-8 w-48 rounded-md" />
                  ) : (
                    <Select
                      value={selectedKbId?.toString() ?? "all"}
                      onValueChange={(v) =>
                        setSelectedKbId(v === "all" ? null : Number(v))
                      }
                    >
                      <SelectTrigger className="h-8 text-xs w-52 bg-muted border-border">
                        <SelectValue placeholder="全部知识库" />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectItem value="all">全部知识库</SelectItem>
                        {kbs.map((kb) => (
                          <SelectItem key={kb.id} value={kb.id.toString()}>
                            {kb.name}（{getSourceModuleLabel(kb.source_module)}）
                          </SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
                  )}
                </div>
                <div className="flex gap-2 items-end">
                  <div className="flex-1 relative">
                    <Input
                      ref={inputRef}
                      placeholder="输入问题，基于知识库检索回答..."
                      value={inputValue}
                      onChange={(e) => setInputValue(e.target.value)}
                      onKeyDown={handleKeyDown}
                      disabled={isLoading}
                      className="flex-1 h-11 pl-4 pr-12 rounded-xl border-border bg-muted focus:bg-card focus:border-ring focus:ring-1 focus:ring-ring transition-all text-sm"
                    />
                  </div>

                  {isLoading ? (
                    <Button
                      onClick={handleStopGeneration}
                      variant="outline"
                      size="sm"
                      className="h-11 px-4 rounded-xl border-destructive/30 text-destructive hover:bg-destructive/10 hover:text-destructive transition-all"
                    >
                      <Square className="w-4 h-4 mr-1.5 fill-current" />
                      停止
                    </Button>
                  ) : (
                    <Button
                      onClick={handleSendMessage}
                      disabled={!inputValue.trim()}
                      size="sm"
                      className="h-11 px-4 rounded-xl bg-primary hover:bg-primary/90 transition-all disabled:bg-muted"
                    >
                      <Send className="w-4 h-4" />
                    </Button>
                  )}
                </div>
                <p className="text-[10px] text-muted-foreground mt-2 text-center">
                  AI 回答基于知识库内容生成，请以原始文档为准
                </p>
              </div>
            </div>
          </div>
        </div>
      </Card>

      {/* 反馈输入弹窗 */}
      <Dialog open={feedbackDialog.open} onOpenChange={(open) => { if (!open) closeFeedbackDialog(); }}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle>
              {feedbackDialog.rating === "positive" ? "这个回答对您有帮助吗？" : "这个回答哪里不够好？"}
            </DialogTitle>
          </DialogHeader>
          <Textarea
            placeholder="请输入您的建议或问题详情（选填）"
            value={feedbackDialog.comment}
            onChange={(e) => setFeedbackDialog((prev) => ({ ...prev, comment: e.target.value }))}
            className="min-h-[100px] resize-none"
          />
          <DialogFooter className="gap-2 sm:gap-0">
            <Button variant="outline" onClick={closeFeedbackDialog}>
              取消
            </Button>
            <Button onClick={handleSubmitFeedback} className="bg-primary hover:bg-primary/90">
              提交反馈
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  );
}

// ─── 引用溯源卡片子组件 ──────────────────────────────────

function SourcesCard({ sources }: { sources: SearchResult[] }) {
  const [expanded, setExpanded] = useState(false);

  return (
    <div className="border border-border rounded-xl bg-muted/30 overflow-hidden">
      <button
        onClick={() => setExpanded(!expanded)}
        className="w-full flex items-center justify-between px-3 py-2 hover:bg-accent/50 transition-colors"
      >
        <div className="flex items-center gap-1.5">
          <FileText className="w-3.5 h-3.5 text-primary" />
          <span className="text-xs font-medium text-foreground">
            引用 {sources.length} 篇文档
          </span>
        </div>
        {expanded ? (
          <ChevronRight className="w-3.5 h-3.5 text-muted-foreground rotate-90 transition-transform" />
        ) : (
          <ChevronRight className="w-3.5 h-3.5 text-muted-foreground transition-transform" />
        )}
      </button>

      {expanded && (
        <div className="px-3 pb-3 space-y-2 animate-in slide-in-from-top-1 duration-200">
          {sources.map((source, idx) => (
            <div
              key={idx}
              className="bg-card border border-border rounded-lg p-2.5"
            >
              <div className="flex items-start gap-2">
                <FileText className="w-3 h-3 mt-0.5 flex-shrink-0 text-muted-foreground" />
                <div className="flex-1 min-w-0">
                  <p className="text-xs font-medium text-foreground truncate">
                    {source.title}
                  </p>
                  <p className="text-xs text-muted-foreground mt-1 line-clamp-3 leading-relaxed">
                    {source.snippet}
                  </p>
                  <div className="flex items-center gap-2 mt-1.5">
                    <span className="text-[10px] text-primary bg-accent px-1.5 py-0.5 rounded font-medium">
                      相关度 {(source.score * 100).toFixed(0)}%
                    </span>
                  </div>
                </div>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
