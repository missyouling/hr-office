"use client";

import Link from "next/link";
import { ExternalLink } from "lucide-react";
import { SidebarGroup, SidebarGroupLabel, SidebarMenu, SidebarMenuButton, SidebarMenuItem } from "@/components/ui/sidebar";

type NavDocumentItem = {
  label: string;
  url: string;
};

interface NavDocumentsProps {
  items: NavDocumentItem[];
}

export function NavDocuments({ items }: NavDocumentsProps) {
  if (!items.length) return null;
  return (
    <SidebarGroup>
      <SidebarGroupLabel>帮助与链接</SidebarGroupLabel>
      <SidebarMenu>
        {items.map((item) => (
          <SidebarMenuItem key={item.url}>
            <SidebarMenuButton asChild>
              <Link href={item.url} target="_blank" rel="noreferrer" className="flex items-center gap-2">
                <ExternalLink className="h-4 w-4" />
                <span>{item.label}</span>
              </Link>
            </SidebarMenuButton>
          </SidebarMenuItem>
        ))}
      </SidebarMenu>
    </SidebarGroup>
  );
}
