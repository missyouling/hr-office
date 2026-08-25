"use client";

import { BookOpen, Sparkles } from "lucide-react";

import { Button } from "@/components/ui/button";
import { ChatPanel } from "@/components/chat-panel";
import { useAuth } from "@/lib/supabase/auth-context";

/** ChatPanel page 变体常开不关：onOpenChange 提供空实现，避免 Escape 关闭整页问答 */
const noop = () => {};

interface KnowledgeQaPageProps {
  /** 切换视图回调：知识库管理按钮跳转既有 knowledge 管理页 */
  onViewChange?: (view: string) => void;
}

/**
 * AI 知识库问答页：占满内容区高度，复用 ChatPanel(page 变体)。
 * 顶部窄条提供标题与管理入口；管理入口沿用管理员兜底
 * （后端 rbac_seed 未种子化 knowledge_base 权限，admin/super_admin/username=admin 放行）。
 */
export function KnowledgeQaPage({ onViewChange }: KnowledgeQaPageProps) {
  const { user } = useAuth();
  const canManage =
    user?.role === "admin" || user?.role === "super_admin" || user?.username === "admin";

  return (
    <div className="relative flex h-full min-h-0 flex-col">
      {/* 顶部窄条：标题 + 知识库管理入口 */}
      <header className="flex h-12 shrink-0 items-center justify-between border-b px-1">
        <div className="flex items-center gap-2">
          <Sparkles className="h-4 w-4 text-primary" aria-hidden />
          <h1 className="text-base font-semibold">AI 知识库问答</h1>
        </div>
        {canManage && (
          <Button
            variant="ghost"
            size="sm"
            className="h-8 gap-1.5 px-2.5 text-muted-foreground hover:text-foreground"
            onClick={() => onViewChange?.("knowledge")}
          >
            <BookOpen className="h-4 w-4" aria-hidden />
            知识库管理
          </Button>
        )}
      </header>
      {/* 问答主体：page 变体以 absolute inset-0 铺满本容器 */}
      <div className="relative min-h-0 flex-1">
        <ChatPanel variant="page" open onOpenChange={noop} />
      </div>
    </div>
  );
}
