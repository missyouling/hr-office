"use client";

import { useEffect, useState, useCallback } from "react";
import { toast } from "sonner";
import { Loader2 } from "lucide-react";
import { PieChart, Pie, Cell, ResponsiveContainer, Tooltip, Legend } from "recharts";

import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { invoiceApi, type InvoiceStats } from "@/lib/api-invoice";
import { getStatusLabel, formatCurrency } from "../utils";

// 饼图颜色
const COLORS = ["#6b7280", "#3b82f6", "#10b981", "#ef4444", "#8b5cf6"];

interface StatsTabProps {
  refreshKey: number;
}

/** 将 by_status 转换为 recharts 数据 */
function buildPieData(stats: InvoiceStats) {
  return Object.entries(stats.by_status).map(([status, count]) => ({
    name: getStatusLabel(status),
    value: count,
    status,
  }));
}

/** 状态对应的颜色索引 */
function getColorIndex(status: string): string {
  const map: Record<string, string> = {
    draft: COLORS[0],
    submitted: COLORS[1],
    approved: COLORS[2],
    rejected: COLORS[3],
    reimbursed: COLORS[4],
  };
  return map[status] || COLORS[0];
}

export default function StatsTab({ refreshKey }: StatsTabProps) {
  const [stats, setStats] = useState<InvoiceStats | null>(null);
  const [loading, setLoading] = useState(false);

  const loadStats = useCallback(async () => {
    setLoading(true);
    try {
      const data = await invoiceApi.stats();
      setStats(data);
    } catch (err) {
      const message = err instanceof Error ? err.message : "加载统计数据失败";
      toast.error(message);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    loadStats();
  }, [loadStats, refreshKey]);

  if (loading && !stats) {
    return (
      <div className="flex min-h-[300px] items-center justify-center">
        <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
      </div>
    );
  }

  if (!stats) {
    return (
      <div className="flex min-h-[300px] items-center justify-center text-muted-foreground">
        暂无统计数据
      </div>
    );
  }

  const pieData = buildPieData(stats);

  return (
    <div className="space-y-6">
      {/* KPI 卡片 */}
      <div className="grid gap-4 md:grid-cols-4">
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium text-muted-foreground">发票总数</CardTitle>
          </CardHeader>
          <CardContent>
            <p className="text-3xl font-bold">{stats.total}</p>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium text-muted-foreground">总金额</CardTitle>
          </CardHeader>
          <CardContent>
            <p className="text-3xl font-bold text-blue-600">{formatCurrency(stats.total_amount)}</p>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium text-muted-foreground">草稿数</CardTitle>
          </CardHeader>
          <CardContent>
            <p className="text-3xl font-bold text-gray-600">{stats.by_status?.draft ?? 0}</p>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium text-muted-foreground">已报销数</CardTitle>
          </CardHeader>
          <CardContent>
            <p className="text-3xl font-bold text-violet-600">{stats.by_status?.reimbursed ?? 0}</p>
          </CardContent>
        </Card>
      </div>

      {/* 饼图 */}
      <div className="grid gap-6 md:grid-cols-2">
        <Card>
          <CardHeader>
            <CardTitle>状态分布</CardTitle>
          </CardHeader>
          <CardContent>
            {pieData.length === 0 ? (
              <p className="text-center text-muted-foreground">暂无数据</p>
            ) : (
              <ResponsiveContainer width="100%" height={300}>
                <PieChart>
                  <Pie
                    data={pieData}
                    cx="50%"
                    cy="50%"
                    innerRadius={60}
                    outerRadius={100}
                    paddingAngle={3}
                    dataKey="value"
                    label={({ name, value }) => `${name}: ${value}`}
                  >
                    {pieData.map((entry) => (
                      <Cell key={entry.status} fill={getColorIndex(entry.status)} />
                    ))}
                  </Pie>
                  <Tooltip formatter={(v: unknown) => `${v} 张`} />
                  <Legend />
                </PieChart>
              </ResponsiveContainer>
            )}
          </CardContent>
        </Card>

        {/* 状态明细卡片 */}
        <Card>
          <CardHeader>
            <CardTitle>各状态明细</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="space-y-3">
              {pieData.map((item) => (
                <div key={item.status} className="flex items-center justify-between">
                  <div className="flex items-center gap-2">
                    <span className="inline-block h-3 w-3 rounded-full" style={{ backgroundColor: getColorIndex(item.status) }} />
                    <span className="text-sm">{item.name}</span>
                  </div>
                  <span className="text-sm font-medium">{item.value} 张</span>
                </div>
              ))}
            </div>
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
