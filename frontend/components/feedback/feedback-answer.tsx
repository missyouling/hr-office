import { ArchiveX, FileText } from "lucide-react";
import { MarkdownContent } from "@/lib/markdown";
import type { ChatFeedback } from "@/lib/api";

export function FeedbackAnswer({ item }: { item: ChatFeedback }) {
  if (item.answer_unavailable) {
    return (
      <div className="flex items-start gap-3 rounded-xl border border-dashed bg-muted/40 p-4 text-muted-foreground">
        <ArchiveX className="mt-0.5 h-5 w-5 shrink-0" aria-hidden />
        <div>
          <p className="font-medium text-foreground">原回答不可用</p>
          <p className="mt-1 text-xs">这是一条旧反馈，关联的历史回答无法恢复，反馈记录仍为您保留。</p>
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-3">
      <div className="rounded-xl border bg-card p-4"><MarkdownContent content={item.answer || "暂无回答内容"} /></div>
      {item.sources?.length > 0 && (
        <div className="grid gap-2 sm:grid-cols-2">
          {item.sources.map((source, index) => (
            <div key={`${source.doc_id}-${index}`} className="rounded-lg border bg-muted/30 p-3">
              <p className="flex items-center gap-2 text-xs font-medium"><FileText className="h-3.5 w-3.5 text-primary" />{source.title}</p>
              <p className="mt-1 line-clamp-2 text-xs leading-relaxed text-muted-foreground">{source.snippet}</p>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
