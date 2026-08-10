/* eslint-disable @typescript-eslint/no-explicit-any */
"use client";

import { useState, useEffect, useCallback } from "react";
import { toast } from "sonner";
import {
  BarChart, Bar, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer,
  ComposedChart, Line,
} from "recharts";
import { TrendingUp, TrendingDown, DollarSign, FileText, AlertTriangle, Lightbulb } from "lucide-react";

import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import {
  Table, TableBody, TableCell, TableHead, TableHeader, TableRow,
} from "@/components/ui/table";
import { Badge } from "@/components/ui/badge";
import { ScrollArea } from "@/components/ui/scroll-area";

import { officeApi } from "@/lib/api-office";
import { formatCurrency } from "../utils";

type TimeDimension = "monthly" | "half-yearly" | "yearly";

/** 数据分析 Tab：时间维度筛选 + KPI + 图表 + 价格异常 + 优化建议 */
export default function AnalyticsTab() {
  const now = new Date();
  const [dim, setDim] = useState<TimeDimension>("monthly");
  const [date, setDate] = useState(now.toISOString().substring(0, 7));
  const [loading, setLoading] = useState(false);
  const [data, setData] = useState<any>(null);

  const loadData = useCallback(async () => {
    setLoading(true);
    try {
      const params = { type: dim, date };
      const [summary, catTrend, topItems, anomaly, suggestions] = await Promise.all([
        officeApi.analytics.summary(params),
        officeApi.analytics.categoryTrend(params),
        officeApi.analytics.topItems(params),
        officeApi.analytics.priceAnomaly(params),
        officeApi.analytics.suggestions(params),
      ]);
      const s = summary.data || {};
      const totalAmount = Number(s.totalAmount) || 0;
      const totalPurchases = Number(s.totalPurchases) || 0;
      const avgOrderAmount = Number(s.avgOrderAmount) || 0;
      const yoyChange = Number(s.yoyChange) || 0;
      const currentTotal = Number(s.currentTotal) || 0;
      const prevTotal = Number(s.prevTotal) || 0;

      // 分类趋势
      const catData = (catTrend.data as Record<string, unknown>)?.categoryStats as any[] | undefined || [];
      const categoryStats = catData.map((c: any) => ({
        category: c.category || "其他",
        amount: Number(c.amount) || 0,
        quantity: Number(c.quantity) || 0,
      }));

      // Top 用品
      const topData = (topItems.data as Record<string, unknown>)?.topSupplies as any[] | undefined || [];
      const topSupplies = topData.map((t: any) => ({
        name: t.name,
        totalAmount: Number(t.total_amount) || 0,
        totalQuantity: Number(t.total_qty) || 0,
        avgPrice: Number(t.avg_price) || 0,
      })).slice(0, 10);

      // 价格异常
      const anomalyData = (anomaly.data as Record<string, unknown>)?.priceAnomalies as any[] | undefined || [];
      const priceAnomalies = anomalyData.map((a: any) => ({
        supplyName: a.supplyName,
        spec: a.spec || "",
        lastUnitPrice: Number(a.lastUnitPrice) || 0,
        prevUnitPrice: Number(a.prevUnitPrice) || 0,
        changePercent: Number(a.changePercent) || 0,
        lastPurchaseDate: a.lastPurchaseDate || "",
      }));

      // 优化建议
      const sugData = (suggestions.data as Record<string, unknown>)?.suggestions as any[] | undefined || [];
      const sugList = sugData.map((sg: any, i: number) => ({
        id: `sug-${i + 1}`,
        type: sg.type || "info",
        title: sg.title || "",
        description: sg.description || "",
        impact: sg.impact,
      }));

      setData({
        kpi: { totalAmount, totalPurchases, avgOrderAmount, yoyChange, currentTotal, prevTotal },
        categoryStats, topSupplies, priceAnomalies, suggestions: sugList,
      });
    } catch (e: any) {
      toast.error("数据分析加载失败", { description: e.message });
    } finally {
      setLoading(false);
    }
  }, [dim, date]);

  useEffect(() => { loadData(); }, [loadData]);

  return (
    <div className="space-y-4">
      {/* 时间维度选择 */}
      <div className="flex items-center gap-3">
        <Select value={dim} onValueChange={v => setDim(v as TimeDimension)}>
          <SelectTrigger className="w-[140px]"><SelectValue /></SelectTrigger>
          <SelectContent>
            <SelectItem value="monthly">月度</SelectItem>
            <SelectItem value="half-yearly">半年度</SelectItem>
            <SelectItem value="yearly">年度</SelectItem>
          </SelectContent>
        </Select>
        <input type={dim === "yearly" ? "text" : dim === "half-yearly" ? "text" : "month"} className="h-9 w-[180px] rounded-lg border border-input bg-background px-3 text-sm"
          value={date} onChange={e => setDate(e.target.value)} placeholder={dim === "monthly" ? "YYYY-MM" : "YYYY"} />
        <Button variant="outline" size="sm" onClick={loadData} disabled={loading}>查询</Button>
      </div>

      {loading ? (
        <div className="space-y-4">
          <div className="grid grid-cols-2 lg:grid-cols-4 gap-3">
            {[1, 2, 3, 4].map(i => <Skeleton key={i} className="h-24 rounded-xl" />)}
          </div>
          <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
            {[1, 2, 3, 4].map(i => <Skeleton key={i} className="h-72 rounded-xl" />)}
          </div>
          <Skeleton className="h-48 rounded-xl" />
        </div>
      ) : !data ? (
        <div className="text-center py-16 text-muted-foreground">
          <p className="text-lg mb-2">暂无数据</p>
          <p className="text-sm">所选条件下没有采购记录，请先录入采购单</p>
        </div>
      ) : (
        <div className="space-y-4">
          {/* KPI 卡片 */}
          <div className="grid grid-cols-2 gap-3 lg:grid-cols-4">
            <KpiCard title="总采购金额" value={formatCurrency(data.kpi.totalAmount)} icon={<DollarSign className="h-4 w-4" />}
              trend={`采购 ${data.kpi.totalPurchases} 单`} />
            <KpiCard title="客单价" value={formatCurrency(data.kpi.avgOrderAmount)} icon={<FileText className="h-4 w-4" />}
              trend="" />
            <KpiCard title="同比变化" value={`${data.kpi.yoyChange > 0 ? "+" : ""}${data.kpi.yoyChange.toFixed(1)}%`}
              icon={data.kpi.yoyChange >= 0 ? <TrendingUp className="h-4 w-4 text-green-500" /> : <TrendingDown className="h-4 w-4 text-red-500" />}
              trend="较上一期" />
            <KpiCard title="当前总额" value={formatCurrency(data.kpi.currentTotal)} icon={<DollarSign className="h-4 w-4" />}
              trend={data.kpi.prevTotal ? `上期 ${formatCurrency(data.kpi.prevTotal)}` : ""} />
          </div>

          {/* 图表行 */}
          <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
            {/* 分类趋势柱线组合 */}
            <Card>
              <CardHeader className="pb-2"><CardTitle className="text-sm">分类金额分布</CardTitle></CardHeader>
              <CardContent className="h-72">
                {data.categoryStats.length === 0 ? (
                  <p className="text-sm text-muted-foreground text-center pt-24">暂无分类数据</p>
                ) : (
                  <ResponsiveContainer width="100%" height="100%">
                    <ComposedChart data={data.categoryStats}>
                      <CartesianGrid strokeDasharray="3 3" />
                      <XAxis dataKey="category" tick={{ fontSize: 10 }} />
                      <YAxis tick={{ fontSize: 10 }} tickFormatter={v => `¥${(v / 1000).toFixed(0)}k`} />
                      <Tooltip formatter={(v) => formatCurrency(Number(v))} />
                      <Bar dataKey="amount" fill="var(--chart-1)" name="金额" />
                      <Line type="monotone" dataKey="quantity" stroke="var(--chart-3)" name="数量" strokeWidth={2} />
                    </ComposedChart>
                  </ResponsiveContainer>
                )}
              </CardContent>
            </Card>

            {/* Top 用品柱图 */}
            <Card>
              <CardHeader className="pb-2"><CardTitle className="text-sm">Top 用品（金额）</CardTitle></CardHeader>
              <CardContent className="h-72">
                {data.topSupplies.length === 0 ? (
                  <p className="text-sm text-muted-foreground text-center pt-24">暂无用品数据</p>
                ) : (
                  <ResponsiveContainer width="100%" height="100%">
                    <BarChart data={data.topSupplies.slice(0, 8)} layout="vertical">
                      <CartesianGrid strokeDasharray="3 3" />
                      <XAxis type="number" tick={{ fontSize: 10 }} tickFormatter={v => `¥${(v / 1000).toFixed(0)}k`} />
                      <YAxis type="category" dataKey="name" tick={{ fontSize: 10 }} width={80} />
                      <Tooltip formatter={(v) => formatCurrency(Number(v))} />
                      <Bar dataKey="totalAmount" fill="var(--chart-2)" name="金额" radius={[0, 4, 4, 0]} />
                    </BarChart>
                  </ResponsiveContainer>
                )}
              </CardContent>
            </Card>
          </div>

          {/* 价格异常表 */}
          {data.priceAnomalies.length > 0 && (
            <Card>
              <CardHeader className="pb-2">
                <CardTitle className="text-sm flex items-center gap-2">
                  <AlertTriangle className="h-4 w-4 text-amber-500" />价格异常监控
                </CardTitle>
              </CardHeader>
              <CardContent className="p-0">
                <ScrollArea className="max-h-64 rounded-md border">
                  <Table>
                    <TableHeader className="sticky top-0 bg-muted">
                      <TableRow>
                        <TableHead className="text-xs">用品</TableHead>
                        <TableHead className="text-xs">规格</TableHead>
                        <TableHead className="text-xs text-right">上次单价</TableHead>
                        <TableHead className="text-xs text-right">此前单价</TableHead>
                        <TableHead className="text-xs text-center">变动</TableHead>
                        <TableHead className="text-xs">采购日期</TableHead>
                      </TableRow>
                    </TableHeader>
                    <TableBody>
                      {data.priceAnomalies.map((a: any, i: number) => (
                        <TableRow key={i}>
                          <TableCell className="text-sm font-medium">{a.supplyName}</TableCell>
                          <TableCell className="text-xs text-muted-foreground">{a.spec || "-"}</TableCell>
                          <TableCell className="text-xs text-right font-mono">{formatCurrency(a.lastUnitPrice)}</TableCell>
                          <TableCell className="text-xs text-right font-mono">{formatCurrency(a.prevUnitPrice)}</TableCell>
                          <TableCell className="text-center">
                            <Badge variant={a.changePercent > 10 ? "destructive" : a.changePercent > 0 ? "default" : "secondary"} className="text-xs">
                              {a.changePercent > 0 ? "+" : ""}{a.changePercent.toFixed(1)}%
                            </Badge>
                          </TableCell>
                          <TableCell className="text-xs">{a.lastPurchaseDate}</TableCell>
                        </TableRow>
                      ))}
                    </TableBody>
                  </Table>
                </ScrollArea>
              </CardContent>
            </Card>
          )}

          {/* 优化建议 */}
          {data.suggestions.length > 0 && (
            <Card>
              <CardHeader className="pb-2">
                <CardTitle className="text-sm flex items-center gap-2">
                  <Lightbulb className="h-4 w-4 text-amber-500" />优化建议
                </CardTitle>
              </CardHeader>
              <CardContent>
                <div className="space-y-2">
                  {data.suggestions.map((sg: any) => (
                    <div key={sg.id} className="flex items-start gap-2 p-2 rounded-lg border bg-muted/30">
                      <div className={`mt-0.5 h-2 w-2 rounded-full flex-shrink-0 ${sg.type === "warning" ? "bg-amber-500" : sg.type === "danger" ? "bg-red-500" : "bg-blue-500"}`} />
                      <div>
                        <p className="text-sm font-medium">{sg.title}</p>
                        <p className="text-xs text-muted-foreground">{sg.description}</p>
                        {sg.impact && <p className="text-xs text-muted-foreground mt-0.5">预估影响：{sg.impact}</p>}
                      </div>
                    </div>
                  ))}
                </div>
              </CardContent>
            </Card>
          )}
        </div>
      )}
    </div>
  );
}

/** KPI 卡片组件 */
function KpiCard({ title, value, icon, trend }: {
  title: string; value: string; icon: React.ReactNode; trend: string;
}) {
  return (
    <Card>
      <CardContent className="p-4">
        <div className="flex items-center justify-between mb-2">
          <span className="text-xs text-muted-foreground">{title}</span>
          {icon}
        </div>
        <p className="text-2xl font-bold">{value}</p>
        {trend && <p className="text-xs text-muted-foreground mt-1">{trend}</p>}
      </CardContent>
    </Card>
  );
}
