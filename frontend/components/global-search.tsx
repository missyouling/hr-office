"use client";

import { useState, useEffect, useCallback } from "react";
import { Search, FileText, Users, Building2 } from "lucide-react";
import * as DialogPrimitive from "@radix-ui/react-dialog";
import { Input } from "@/components/ui/input";
import { Badge } from "@/components/ui/badge";
import { globalSearch } from "@/lib/api";
import type { GlobalSearchResult } from "@/lib/api";

interface GlobalSearchProps {
  onNavigate: (module: string, id?: number) => void;
  open?: boolean;
  onOpenChange?: (open: boolean) => void;
  hideTrigger?: boolean;
}

const MODULE_ICONS: Record<string, React.ReactNode> = {
  档案: <FileText className="w-4 h-4" />,
  员工: <Users className="w-4 h-4" />,
  宿舍: <Building2 className="w-4 h-4" />,
};

const MODULE_COLORS: Record<string, string> = {
  档案: "bg-blue-100 text-blue-800",
  员工: "bg-green-100 text-green-800",
  宿舍: "bg-purple-100 text-purple-800",
};

export function GlobalSearch({ onNavigate, open, onOpenChange, hideTrigger = false }: GlobalSearchProps) {
  const [internalOpen, setInternalOpen] = useState(false);
  const isOpen = open ?? internalOpen;
  const setIsOpen = onOpenChange ?? setInternalOpen;
  const [searchQuery, setSearchQuery] = useState("");
  const [results, setResults] = useState<GlobalSearchResult[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [groupedResults, setGroupedResults] = useState<
    Record<string, GlobalSearchResult[]>
  >({});

  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if ((e.metaKey || e.ctrlKey) && e.key === "k") {
        e.preventDefault();
        setIsOpen(true);
      }
    };

    window.addEventListener("keydown", handleKeyDown);
    return () => window.removeEventListener("keydown", handleKeyDown);
  }, [setIsOpen]);

  const performSearch = useCallback(
    async (query: string) => {
      if (!query.trim()) {
        setResults([]);
        setGroupedResults({});
        setError(null);
        return;
      }

      setIsLoading(true);
      setError(null);
      try {
        const response = await globalSearch(query, 20);
        setResults(response.results);

        const grouped = response.results.reduce(
          (acc, result) => {
            if (!acc[result.module]) {
              acc[result.module] = [];
            }
            acc[result.module].push(result);
            return acc;
          },
          {} as Record<string, GlobalSearchResult[]>
        );

        setGroupedResults(grouped);
      } catch (err) {
        console.error("Search error:", err);
        setResults([]);
        setGroupedResults({});
        setError(
          err instanceof Error && err.message
            ? err.message
            : "搜索失败，请稍后重试"
        );
      } finally {
        setIsLoading(false);
      }
    },
    []
  );

  const handleSearch = useCallback(
    (value: string) => {
      setSearchQuery(value);

      const debounceTimer = setTimeout(() => {
        performSearch(value);
      }, 300);

      return () => clearTimeout(debounceTimer);
    },
    [performSearch]
  );

  const handleResultClick = (module: string, id: number) => {
    onNavigate(module, id);
    setIsOpen(false);
    setSearchQuery("");
    setResults([]);
    setGroupedResults({});
    setError(null);
  };

  return (
    <>
      {/* 直接组装 Radix Dialog 原语：遮罩模糊加深 + 面板居中偏上，样式不依赖 ui/dialog 的通用弹窗契约 */}
      <DialogPrimitive.Root open={isOpen} onOpenChange={setIsOpen}>
        <DialogPrimitive.Portal>
          {/* 遮罩：半透明加深 + 背景模糊，聚焦搜索内容 */}
          <DialogPrimitive.Overlay className="fixed inset-0 z-50 bg-black/40 backdrop-blur-sm data-[state=open]:animate-in data-[state=closed]:animate-out data-[state=closed]:fade-out-0 data-[state=open]:fade-in-0" />
          <DialogPrimitive.Content
            className="fixed left-1/2 top-[20%] z-50 w-full max-w-2xl -translate-x-1/2 rounded-2xl border border-border/70 bg-card p-6 text-card-foreground shadow-[0_24px_60px_-24px_rgba(0,0,0,0.65)] outline-none data-[state=open]:animate-in data-[state=closed]:animate-out data-[state=closed]:fade-out-0 data-[state=open]:fade-in-0 data-[state=closed]:zoom-out-95 data-[state=open]:zoom-in-95"
            aria-describedby={undefined}
          >
            <DialogPrimitive.Title className="flex items-center gap-2 text-lg font-semibold leading-none">
              <Search className="w-5 h-5" />
              全局搜索
            </DialogPrimitive.Title>

            <div className="mt-4 space-y-4">
              <div className="relative">
                <Search className="absolute left-3 top-3 w-4 h-4 text-gray-400" />
                <Input
                  placeholder="搜索档案、员工、宿舍... (Cmd+K)"
                  value={searchQuery}
                  onChange={(e) => handleSearch(e.target.value)}
                  className="pl-10"
                  autoFocus
                />
              </div>

            <div className="max-h-96 overflow-y-auto">
              {isLoading && (
                <div className="flex items-center justify-center py-8">
                  <div className="text-sm text-gray-500">搜索中...</div>
                </div>
              )}

              {!isLoading && error && (
                <div className="flex items-center justify-center py-8">
                  <div
                    className="text-sm text-red-500"
                    role="alert"
                    data-testid="global-search-error"
                  >
                    搜索失败：{error}
                  </div>
                </div>
              )}

              {!isLoading && !error && results.length === 0 && searchQuery && (
                <div className="flex items-center justify-center py-8">
                  <div className="text-sm text-gray-500">未找到相关结果</div>
                </div>
              )}

              {!isLoading && !error && results.length === 0 && !searchQuery && (
                <div className="flex items-center justify-center py-8">
                  <div className="text-sm text-gray-500">
                    输入关键词开始搜索
                  </div>
                </div>
              )}

              {Object.entries(groupedResults).map(([module, items]) => (
                <div key={module} className="mb-6">
                  <div className="flex items-center gap-2 mb-3">
                    {MODULE_ICONS[module] || <FileText className="w-4 h-4" />}
                    <h3 className="font-semibold text-sm">{module}</h3>
                    <Badge variant="secondary" className="text-xs">
                      {items.length}
                    </Badge>
                  </div>

                  <div className="space-y-2">
                    {items.map((result) => (
                      <div
                        key={`${result.module}-${result.id}`}
                        onClick={() => handleResultClick(result.module, result.id)}
                        className="p-3 rounded-lg border border-gray-200 hover:border-blue-400 hover:bg-blue-50 cursor-pointer transition"
                      >
                        <div className="flex items-start justify-between gap-2">
                          <div className="flex-1 min-w-0">
                            <p className="font-medium text-sm text-gray-900 truncate">
                              {result.title}
                            </p>
                            <p className="text-xs text-gray-600 line-clamp-2 mt-1">
                              {result.snippet}
                            </p>
                          </div>
                          <Badge
                            className={`flex-shrink-0 text-xs ${
                              MODULE_COLORS[result.module] ||
                              "bg-gray-100 text-gray-800"
                            }`}
                          >
                            {(result.score * 100).toFixed(0)}%
                          </Badge>
                        </div>
                      </div>
                    ))}
                  </div>
                </div>
              ))}
            </div>

            {searchQuery && !error && (
              <div className="text-xs text-gray-500 text-center py-2">
                找到 {results.length} 个结果
              </div>
            )}
            </div>
          </DialogPrimitive.Content>
        </DialogPrimitive.Portal>
      </DialogPrimitive.Root>

      {!hideTrigger && <div className="fixed top-4 right-4 z-30">
        <button
          onClick={() => setIsOpen(true)}
          className="flex items-center gap-2 px-3 py-2 rounded-lg border border-gray-300 bg-white hover:bg-gray-50 text-sm text-gray-600 shadow-sm transition"
        >
          <Search className="w-4 h-4" />
          <span className="hidden sm:inline">搜索</span>
          <kbd className="hidden sm:inline ml-2 px-2 py-1 text-xs font-semibold text-gray-800 bg-gray-100 border border-gray-200 rounded">
            ⌘K
          </kbd>
        </button>
      </div>}
    </>
  );
}
