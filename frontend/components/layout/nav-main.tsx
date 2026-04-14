"use client";

import { SidebarGroup, SidebarGroupContent, SidebarMenu, SidebarMenuButton, SidebarMenuItem } from "@/components/ui/sidebar";
import { cn } from "@/lib/utils";
import type { LucideIcon } from "lucide-react";

export type NavMainItem = {
  id: string;
  label: string;
  icon: LucideIcon;
};

interface NavMainProps {
  items: NavMainItem[];
  activeId: string;
  onSelect: (id: string) => void;
}

export function NavMain({ items, activeId, onSelect }: NavMainProps) {
  return (
    <SidebarGroup>
      <SidebarGroupContent className="space-y-1">
        <SidebarMenu>
          {items.map((item) => {
            const Icon = item.icon;
            const active = activeId === item.id;
            return (
              <SidebarMenuItem key={item.id}>
                <SidebarMenuButton
                  className={cn(
                    "h-10 w-full items-center gap-3 rounded-lg px-3",
                    active ? "bg-black text-white shadow-sm dark:bg-white dark:text-black" : "hover:bg-sidebar-accent"
                  )}
                  isActive={active}
                  onClick={() => onSelect(item.id)}
                >
                  <Icon className="h-4 w-4" />
                  <span className="text-sm font-medium">{item.label}</span>
                </SidebarMenuButton>
              </SidebarMenuItem>
            );
          })}
        </SidebarMenu>
      </SidebarGroupContent>
    </SidebarGroup>
  );
}
