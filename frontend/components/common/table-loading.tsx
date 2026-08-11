"use client";
import { Skeleton } from "@/components/ui/skeleton";

interface TableLoadingProps {
  /** 骨架行数，默认 5 */
  rows?: number;
  /** 骨架列数，默认 4 */
  columns?: number;
  /** 是否显示表头骨架，默认 true */
  showHeader?: boolean;
}

/** 表格加载态骨架屏组件 —— 禁止出现英文 "Loading" 文案 */
export function TableLoading({ rows = 5, columns = 4, showHeader = true }: TableLoadingProps) {
  return (
    <div className="space-y-3">
      {showHeader && (
        <div className="flex gap-2">
          {Array.from({ length: columns }).map((_, i) => (
            <Skeleton key={i} className="h-4 flex-1" />
          ))}
        </div>
      )}
      {Array.from({ length: rows }).map((_, rowIdx) => (
        <div key={rowIdx} className="flex gap-2">
          {Array.from({ length: columns }).map((_, colIdx) => (
            <Skeleton key={colIdx} className="h-8 flex-1" />
          ))}
        </div>
      ))}
    </div>
  );
}
