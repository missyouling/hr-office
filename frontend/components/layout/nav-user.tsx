"use client";

import { useState } from "react";
import { SidebarGroup, SidebarMenu, SidebarMenuItem, useSidebar } from "@/components/ui/sidebar";
import { useAuth } from "@/lib/auth";
import { UserPreferencesDialog } from "@/components/user-preferences-dialog";

interface NavUserProps {
  displayName: string;
  subLine?: string;
}

export function NavUser({ displayName, subLine }: NavUserProps) {
  const { state } = useSidebar();
  const { user } = useAuth();
  const isCollapsed = state === "collapsed";
  const [isPreferencesOpen, setIsPreferencesOpen] = useState(false);

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
              onClick={() => setIsPreferencesOpen(true)}
              aria-label="用户设置"
              title={subLine ? `${displayName} | ${subLine}` : displayName}
              className="flex h-10 w-10 shrink-0 items-center justify-center rounded-full bg-muted shadow-sm transition hover:opacity-90"
            >
              <img
                src={user?.full_name ? getDefaultAvatar(user.full_name) : getDefaultAvatar(displayName)}
                alt="avatar"
                className="h-8 w-8 rounded-full bg-muted object-cover"
              />
            </button>
          </SidebarMenuItem>
        </SidebarMenu>
      </SidebarGroup>
    );
  }

  return (
    <SidebarGroup>
      <SidebarMenu>
        <SidebarMenuItem>
          <button
            type="button"
            onClick={() => setIsPreferencesOpen(true)}
            className="flex h-[46px] w-full items-center gap-2 rounded-md bg-muted px-3 text-foreground transition hover:bg-muted/80"
          >
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
    </SidebarGroup>
  );
}
