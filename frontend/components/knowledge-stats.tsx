"use client";

import { useState, useEffect } from "react";
import { Card } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { FileText, Layers, MessageSquare, Tag } from "lucide-react";
import { getKnowledgeStats, fetchKnowledgeSessions, fetchArchiveTags } from "@/lib/api";
import { CountingNumber } from "@/components/common/counting-number";

interface StatsData {
  totalDocuments: number;
  totalEmbeddings: number;
  sessionCount: number;
  topTags: string[];
}

export function KnowledgeStats() {
  const [stats, setStats] = useState<StatsData | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    let cancelled = false;

    async function loadStats() {
      try {
        const [knowledgeStats, sessions, tags] = await Promise.all([
          getKnowledgeStats(),
          fetchKnowledgeSessions(),
          fetchArchiveTags(),
        ]);

        if (cancelled) return;

        // 按文档数量排序，取前 3 个标签
        const sortedTags = [...(tags || [])]
          .sort((a, b) => (b.document_count ?? 0) - (a.document_count ?? 0))
          .slice(0, 3)
          .map((t) => t.name);

        setStats({
          totalDocuments: knowledgeStats.documents ?? 0,
          totalEmbeddings: knowledgeStats.embeddings ?? 0,
          sessionCount: (sessions || []).length,
          topTags: sortedTags,
        });
      } catch {
        // 静默处理，卡片留空
      } finally {
        if (!cancelled) setLoading(false);
      }
    }

    loadStats();
    return () => { cancelled = true; };
  }, []);

  if (loading) {
    return (
      <div className="grid grid-cols-2 lg:grid-cols-4 gap-4 mb-6">
        {[...Array(4)].map((_, i) => (
          <Card key={i} className="p-4">
            <div className="flex items-center gap-3">
              <Skeleton className="w-10 h-10 rounded-lg" />
              <div className="space-y-1.5">
                <Skeleton className="h-7 w-12" />
                <Skeleton className="h-4 w-20" />
              </div>
            </div>
          </Card>
        ))}
      </div>
    );
  }

  const cards = [
    {
      label: "知识库文档",
      value: stats?.totalDocuments ?? "--",
      icon: FileText,
      bg: "bg-blue-100",
      color: "text-blue-600",
    },
    {
      label: "向量分块数",
      value: stats?.totalEmbeddings ?? "--",
      icon: Layers,
      bg: "bg-indigo-100",
      color: "text-indigo-600",
    },
    {
      label: "问答会话数",
      value: stats?.sessionCount ?? "--",
      icon: MessageSquare,
      bg: "bg-emerald-100",
      color: "text-emerald-600",
    },
    {
      label: "热门标签",
      value: stats?.topTags?.length ? stats.topTags.join(" · ") : "--",
      icon: Tag,
      bg: "bg-amber-100",
      color: "text-amber-600",
      isTags: true,
    },
  ];

  return (
    <div className="grid grid-cols-2 lg:grid-cols-4 gap-4 mb-6">
      {cards.map((card) => (
        <Card key={card.label} className="p-4 hover:shadow-md transition-shadow">
          <div className="flex items-center gap-3">
            <div className={`p-2 rounded-lg ${card.bg}`}>
              <card.icon className={`w-5 h-5 ${card.color}`} />
            </div>
            <div className="min-w-0">
              <p className={`font-bold ${card.isTags ? "text-sm" : "text-2xl"} truncate`}>
                {card.isTags
                  ? card.value
                  : typeof card.value === "number"
                    ? <CountingNumber value={card.value} />
                    : card.value}
              </p>
              <p className="text-sm text-gray-500">{card.label}</p>
            </div>
          </div>
        </Card>
      ))}
    </div>
  );
}
