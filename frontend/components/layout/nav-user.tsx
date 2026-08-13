"use client";

import { useCallback, useEffect, useState } from "react";
import { MessageSquareText } from "lucide-react";
import { SidebarGroup, SidebarMenu, SidebarMenuItem, useSidebar } from "@/components/ui/sidebar";
import { useAuth } from "@/lib/auth";
import { UserPreferencesDialog } from "@/components/user-preferences-dialog";
import { MyFeedbackDialog } from "@/components/feedback/my-feedback-dialog";
import { listMyFeedback } from "@/lib/api";
import { FEEDBACK_VIEWED_STORAGE_KEY, getViewedReplies, isReplyUnread } from "@/lib/feedback";

interface NavUserProps {
  displayName: string;
  subLine?: string;
}

export function NavUser({ displayName, subLine }: NavUserProps) {
  const { state } = useSidebar();
  const { user } = useAuth();
  const isCollapsed = state === "collapsed";
  const [isPreferencesOpen, setIsPreferencesOpen] = useState(false);
  const [isFeedbackOpen, setIsFeedbackOpen] = useState(false);
  const [hasNewReply, setHasNewReply] = useState(false);

  const checkReplies = useCallback(async () => {
    try {
      const data = await listMyFeedback(1);
      const viewed = getViewedReplies(localStorage.getItem(FEEDBACK_VIEWED_STORAGE_KEY));
      setHasNewReply(data.items.some((item) => isReplyUnread(item, viewed)));
    } catch {
      setHasNewReply(false);
    }
  }, []);

  useEffect(() => {
    checkReplies();
    window.addEventListener("feedback:reply-viewed", checkReplies);
    return () => window.removeEventListener("feedback:reply-viewed", checkReplies);
  }, [checkReplies]);

  const getDefaultAvatar = (username: string) => {
    return `https://api.dicebear.com/7.x/initials/png?name=${encodeURIComponent(username)}&backgroundColor=random`;
  };

  if (isCollapsed) {
    return (
      <SidebarGroup>
        <SidebarMenu>
          <SidebarMenuItem>
            <button
              type="button"
              onClick={() => setIsFeedbackOpen(true)}
              aria-label={hasNewReply ? "我的反馈，有新回复" : "我的反馈"}
              className="relative flex h-10 w-10 items-center justify-center rounded-md transition hover:bg-muted"
            >
              <MessageSquareText className="h-4 w-4 text-primary" />
              {hasNewReply && <span className="absolute right-1 top-1 h-2 w-2 rounded-full bg-primary ring-2 ring-background" aria-hidden />}
            </button>
          </SidebarMenuItem>
          <SidebarMenuItem>
            <button
              type="button"
              onClick={() => setIsPreferencesOpen(true)}
              aria-label="用户设置"
              title={subLine ? `${displayName} | ${subLine}` : displayName}
              className="relative flex h-10 w-10 shrink-0 items-center justify-center rounded-full bg-muted shadow-sm transition hover:opacity-90"
            >
              {/* eslint-disable-next-line @next/next/no-img-element -- 头像URL动态生成（DiceBear API），无需 next/image 优化 */}
              <img
                src={user?.full_name ? getDefaultAvatar(user.full_name) : getDefaultAvatar(displayName)}
                alt="avatar"
                className="h-8 w-8 rounded-full bg-muted object-cover"
              />
            </button>
          </SidebarMenuItem>
        </SidebarMenu>
        <UserPreferencesDialog open={isPreferencesOpen} onOpenChange={setIsPreferencesOpen} />
        <MyFeedbackDialog open={isFeedbackOpen} onOpenChange={setIsFeedbackOpen} />
      </SidebarGroup>
    );
  }

  return (
    <SidebarGroup>
      <SidebarMenu>
        <SidebarMenuItem>
          <button type="button" onClick={() => setIsFeedbackOpen(true)} className="flex w-full items-center gap-2 rounded-md px-3 py-2 text-sm transition hover:bg-muted" aria-label={hasNewReply ? "我的反馈，有新回复" : "我的反馈"}>
            <span className="relative"><MessageSquareText className="h-4 w-4 text-primary" />{hasNewReply && <span className="absolute -right-1 -top-1 h-2 w-2 rounded-full bg-primary ring-2 ring-background" />}</span>
            <span>我的反馈</span>{hasNewReply && <span className="ml-auto text-xs font-medium text-primary">新回复</span>}
          </button>
        </SidebarMenuItem>
        <SidebarMenuItem>
          <button
            type="button"
            onClick={() => setIsPreferencesOpen(true)}
            className="flex h-[46px] w-full items-center gap-2 rounded-md bg-muted px-3 text-foreground transition hover:bg-muted/80"
          >
            {/* eslint-disable-next-line @next/next/no-img-element -- 头像URL动态生成（DiceBear API），无需 next/image 优化 */}
            <img
              src={user?.full_name ? getDefaultAvatar(user.full_name) : getDefaultAvatar(displayName)}
              alt="avatar"
              className="h-8 w-8 shrink-0 rounded-full bg-muted object-cover"
            />
            <div className="min-w-0 flex-1 text-left">
              <div className="truncate text-sm font-medium leading-tight">{displayName}</div>
              {subLine && <div className="truncate text-[11px] text-muted-foreground leading-tight">{subLine}</div>}
            </div>
          </button>
        </SidebarMenuItem>
      </SidebarMenu>

      <UserPreferencesDialog open={isPreferencesOpen} onOpenChange={setIsPreferencesOpen} />
      <MyFeedbackDialog open={isFeedbackOpen} onOpenChange={setIsFeedbackOpen} />
    </SidebarGroup>
  );
}
