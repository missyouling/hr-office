"use client";

import { useState, useEffect } from "react";
import { X, Loader2 } from "lucide-react";
import { cn } from "@/lib/utils";
import { fetchArchivesTags, type TagWithCount } from "@/lib/api";

interface TagFilterProps {
  selectedTags: string[];
  onFilter: (tagNames: string[]) => void;
}

/** 将 hex 颜色转换为浅色背景 + 同色文字的组合 */
function tagColorStyle(hex: string): { bg: string; text: string; border: string } {
  // 默认蓝色
  const color = hex || "#3b82f6";
  // 生产高透明度的背景和边框
  return {
    bg: `${color}18`,
    text: color,
    border: `${color}40`,
  };
}

export function TagFilter({ selectedTags, onFilter }: TagFilterProps) {
  const [tags, setTags] = useState<TagWithCount[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");

  useEffect(() => {
    let cancelled = false;
    setLoading(true);
    setError("");
    fetchArchivesTags()
      .then((data) => {
        if (!cancelled) {
          setTags(data);
        }
      })
      .catch((err) => {
        console.error("加载标签失败:", err);
        if (!cancelled) setError("加载标签失败");
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, []);

  const toggleTag = (tagName: string) => {
    if (selectedTags.includes(tagName)) {
      onFilter(selectedTags.filter((t) => t !== tagName));
    } else {
      onFilter([...selectedTags, tagName]);
    }
  };

  const removeTag = (tagName: string) => {
    onFilter(selectedTags.filter((t) => t !== tagName));
  };

  const hasSelected = selectedTags.length > 0;

  return (
    <div className="space-y-2">
      <div className="flex items-center gap-2">
        <h4 className="text-xs font-semibold uppercase tracking-wider text-muted-foreground">
          标签筛选
        </h4>
        {loading && <Loader2 className="h-3 w-3 animate-spin text-muted-foreground" />}
        {error && <span className="text-xs text-destructive">{error}</span>}
        {hasSelected && (
          <button
            type="button"
            onClick={() => onFilter([])}
            className="text-xs text-muted-foreground underline-offset-2 hover:text-foreground hover:underline"
          >
            清除 ({selectedTags.length})
          </button>
        )}
      </div>

      {/* 已选中的标签（置顶） */}
      {hasSelected && (
        <div className="flex flex-wrap gap-1.5">
          {selectedTags.map((tagName) => {
            const tag = tags.find((t) => t.name === tagName);
            const style = tagColorStyle(tag?.color || "");
            return (
              <span
                key={tagName}
                className={cn(
                  "inline-flex items-center gap-1 rounded-full border px-2.5 py-0.5 text-xs font-medium cursor-pointer transition-colors hover:opacity-80"
                )}
                style={{
                  backgroundColor: style.bg,
                  color: style.text,
                  borderColor: style.text,
                }}
                onClick={() => removeTag(tagName)}
              >
                {tagName}
                <X className="h-3 w-3 shrink-0" />
              </span>
            );
          })}
        </div>
      )}

      {/* 所有可选标签 */}
      <div className="flex flex-wrap gap-1.5">
        {tags.map((tag) => {
          const isSelected = selectedTags.includes(tag.name);
          const style = tagColorStyle(tag.color);
          return (
            <button
              key={tag.id}
              type="button"
              onClick={() => toggleTag(tag.name)}
              className={cn(
                "inline-flex items-center gap-1 rounded-full border px-2.5 py-0.5 text-xs font-medium transition-all duration-150",
                isSelected
                  ? "shadow-sm"
                  : "border-transparent opacity-70 hover:opacity-100"
              )}
              style={{
                backgroundColor: isSelected ? style.bg : `${tag.color}10`,
                color: isSelected ? style.text : `${tag.color}cc`,
                borderColor: isSelected ? style.border : "transparent",
              }}
            >
              {tag.name}
              <span className="tabular-nums opacity-60">
                {tag.document_count}
              </span>
            </button>
          );
        })}
        {!loading && tags.length === 0 && (
          <span className="text-xs text-muted-foreground">暂无标签</span>
        )}
      </div>
    </div>
  );
}
