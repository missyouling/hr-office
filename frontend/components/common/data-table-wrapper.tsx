"use client";

import { ScrollArea } from "@/components/ui/scroll-area";

interface DataTableWrapperProps {
  /** 表格容器高度，默认 "h-[calc(100vh-340px)]" */
  height?: string;
  children: React.ReactNode;
}

/** 表格统一容器：ScrollArea + 圆角边框，搭配 sticky 表头使用 */
export function DataTableWrapper({ height = "h-[calc(100vh-340px)]", children }: DataTableWrapperProps) {
  return (
    <ScrollArea className={`${height} rounded-md border`}>
      {children}
    </ScrollArea>
  );
}
