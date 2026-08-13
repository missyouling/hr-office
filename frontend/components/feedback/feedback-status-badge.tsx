import { Badge } from "@/components/ui/badge";
import { FEEDBACK_STATUS_LABELS, normalizeFeedbackStatus } from "@/lib/feedback";
import type { ChatFeedback } from "@/lib/api";

const STATUS_STYLES = {
  pending: "border-amber-500/30 bg-amber-500/10 text-amber-700 dark:text-amber-300",
  replied: "border-primary/30 bg-primary/10 text-primary",
  closed: "border-border bg-muted text-muted-foreground",
};

export function FeedbackStatusBadge({ item }: { item: Pick<ChatFeedback, "status" | "reply"> }) {
  const status = normalizeFeedbackStatus(item);
  return (
    <Badge variant="outline" className={STATUS_STYLES[status]}>
      {FEEDBACK_STATUS_LABELS[status]}
    </Badge>
  );
}
