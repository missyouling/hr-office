"use client";

import { useState, useEffect, useCallback } from "react";
import { Folder, FolderOpen, ChevronRight, ChevronDown, FileText, Loader2 } from "lucide-react";
import { ScrollArea } from "@/components/ui/scroll-area";
import { cn } from "@/lib/utils";
import {
  fetchArchivesFolders,
  type FolderNode,
  type FolderTreeResult,
} from "@/lib/api";

interface FolderTreeProps {
  categoryCode: string;
  onSelect: (path: string | null) => void;
}

/** 递归渲染单个文件夹节点 */
function TreeNode({
  node,
  depth,
  selectedPath,
  onSelect,
}: {
  node: FolderNode;
  depth: number;
  selectedPath: string | null;
  onSelect: (path: string | null) => void;
}) {
  const [expanded, setExpanded] = useState(
    depth < 1 || (node.children && node.children.length <= 2)
  );
  const hasChildren = node.children && node.children.length > 0;
  const isSelected = selectedPath === node.path;

  return (
    <div>
      <button
        type="button"
        onClick={() => {
          if (hasChildren) {
            setExpanded((prev) => !prev);
          }
          onSelect(isSelected ? null : node.path);
        }}
        className={cn(
          "flex w-full items-center gap-1.5 rounded-md px-2 py-1.5 text-left text-sm transition-colors",
          "hover:bg-accent hover:text-accent-foreground",
          isSelected && "bg-primary/10 text-primary font-medium",
          !isSelected && "text-muted-foreground"
        )}
        style={{ paddingLeft: `${depth * 16 + 8}px` }}
        title={node.path}
      >
        {hasChildren ? (
          expanded ? (
            <ChevronDown className="h-3.5 w-3.5 shrink-0 text-muted-foreground/70" />
          ) : (
            <ChevronRight className="h-3.5 w-3.5 shrink-0 text-muted-foreground/70" />
          )
        ) : (
          <FileText className="h-3.5 w-3.5 shrink-0 text-muted-foreground/50" />
        )}
        {expanded ? (
          <FolderOpen className="h-4 w-4 shrink-0 text-amber-500" />
        ) : (
          <Folder className="h-4 w-4 shrink-0 text-amber-500/70" />
        )}
        <span className="min-w-0 flex-1 truncate">{node.name}</span>
        <span className="shrink-0 text-xs tabular-nums text-muted-foreground/60">
          {node.total_count ?? node.document_count}
        </span>
      </button>
      {hasChildren && expanded && (
        <div className="animate-in fade-in slide-in-from-top-1 duration-150">
          {node.children!.map((child) => (
            <TreeNode
              key={child.path}
              node={child}
              depth={depth + 1}
              selectedPath={selectedPath}
              onSelect={onSelect}
            />
          ))}
        </div>
      )}
    </div>
  );
}

export function FolderTree({ categoryCode, onSelect }: FolderTreeProps) {
  const [tree, setTree] = useState<FolderTreeResult | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");
  const [selectedPath, setSelectedPath] = useState<string | null>(null);

  const loadTree = useCallback(async () => {
    if (!categoryCode) return;
    setLoading(true);
    setError("");
    try {
      const result = await fetchArchivesFolders(categoryCode);
      setTree(result);
    } catch (err) {
      console.error("加载文件夹树失败:", err);
      setError("加载失败");
    } finally {
      setLoading(false);
    }
  }, [categoryCode]);

  useEffect(() => {
    loadTree();
  }, [loadTree]);

  // 当 categoryCode 变化时重置选中状态
  useEffect(() => {
    setSelectedPath(null);
    onSelect(null);
  }, [categoryCode, onSelect]);

  const handleSelect = (path: string | null) => {
    setSelectedPath(path);
    onSelect(path);
  };

  return (
    <div className="flex h-full flex-col">
      {/* 标题栏 */}
      <div className="shrink-0 border-b px-3 py-2.5">
        <h3 className="text-xs font-semibold uppercase tracking-wider text-muted-foreground">
          文件夹
        </h3>
      </div>

      {/* 加载中 */}
      {loading && (
        <div className="flex items-center justify-center py-8">
          <Loader2 className="h-5 w-5 animate-spin text-muted-foreground" />
        </div>
      )}

      {/* 加载失败 */}
      {!loading && error && (
        <div className="flex flex-col items-center gap-2 py-6">
          <p className="text-xs text-muted-foreground">{error}</p>
          <button
            type="button"
            onClick={loadTree}
            className="text-xs text-primary underline-offset-2 hover:underline"
          >
            重试
          </button>
        </div>
      )}

      {/* 文件夹树 */}
      {!loading && !error && tree && (
        <ScrollArea className="flex-1">
          <div className="py-1.5">
            {/* 全部文档 */}
            <button
              type="button"
              onClick={() => handleSelect(null)}
              className={cn(
                "flex w-full items-center gap-1.5 rounded-md px-3 py-1.5 text-left text-sm transition-colors",
                "hover:bg-accent hover:text-accent-foreground",
                !selectedPath && "bg-primary/10 text-primary font-medium",
                selectedPath && "text-muted-foreground"
              )}
            >
              <FileText className="h-4 w-4 shrink-0" />
              <span className="flex-1">全部文档</span>
              <span className="shrink-0 text-xs tabular-nums text-muted-foreground/60">
                {tree.total_document_count}
              </span>
            </button>

            {/* 根目录 */}
            {tree.root_document_count > 0 && (
              <button
                type="button"
                onClick={() => handleSelect("")}
                className={cn(
                  "flex w-full items-center gap-1.5 rounded-md px-3 py-1.5 text-left text-sm transition-colors",
                  "hover:bg-accent hover:text-accent-foreground",
                  selectedPath === "" && "bg-primary/10 text-primary font-medium",
                  selectedPath !== "" && "text-muted-foreground"
                )}
              >
                <Folder className="h-4 w-4 shrink-0 text-muted-foreground/50" />
                <span className="flex-1 truncate">根目录</span>
                <span className="shrink-0 text-xs tabular-nums text-muted-foreground/60">
                  {tree.root_document_count}
                </span>
              </button>
            )}

            {/* 文件夹节点 */}
            {tree.folders.map((folder) => (
              <TreeNode
                key={folder.path}
                node={folder}
                depth={0}
                selectedPath={selectedPath}
                onSelect={handleSelect}
              />
            ))}

            {tree.folders.length === 0 && tree.root_document_count === 0 && (
              <p className="px-3 py-4 text-center text-xs text-muted-foreground">
                暂无文件夹
              </p>
            )}
          </div>
        </ScrollArea>
      )}
    </div>
  );
}
