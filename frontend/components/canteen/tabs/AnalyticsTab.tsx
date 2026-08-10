"use client";

import { useState, useEffect, useCallback } from "react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { toast } from "sonner";
import { canteenApi } from "@/lib/api-canteen";
import {
  KpiCards, DailyTrendChart, ExpensePieChart, FoodShareChart, TopSuppliesChart,
  MonthlyCompareChart, PeoplePerCapitaChart, DailyDetailTable, CompareTable,
} from "./analytics-components";

type Period = "month" | "half" | "year";

export default function AnalyticsTab() {
  const [period, setPeriod] = useState<Period>("month");
  const [month, setMonth] = useState(new Date().toISOString().slice(0, 7));
  const [year, setYear] = useState(String(new Date().getFullYear()));
  const [half, setHalf] = useState<"h1" | "h2">(() => {
    const mo = Number(new Date().toISOString().slice(5, 7)); return mo <= 6 ? "h1" : "h2";
  });

  const [summary, setSummary] = useState<Record<string, unknown> | null>(null);
  const [dailyTrend, setDailyTrend] = useState<Record<string, unknown>[]>([]);
  const [expenseBreakdown, setExpenseBreakdown] = useState<Record<string, unknown>>({ food: 0, others: [] });
  const [foodShare, setFoodShare] = useState<Record<string, unknown>[]>([]);
  const [topSupplies, setTopSupplies] = useState<Record<string, unknown>[]>([]);
  const [compare, setCompare] = useState<Record<string, unknown>[]>([]);
  const [loading, setLoading] = useState(false);

  const loadMonth = useCallback(async (m: string) => {
    setLoading(true);
    try {
      const [s, t, b, f, top] = await Promise.all([
        canteenApi.analytics.summary({ month: m }), canteenApi.analytics.dailyTrend({ month: m }),
        canteenApi.analytics.expenseBreakdown({ month: m }), canteenApi.analytics.foodShare({ month: m }),
        canteenApi.analytics.topSupplies({ month: m, limit: "10" }),
      ]);
      setSummary(s.data as Record<string, unknown>);
      setDailyTrend((t.data as Record<string, unknown>).items as Record<string, unknown>[] || []);
      setExpenseBreakdown(b.data as Record<string, unknown>);
      setFoodShare((f.data as Record<string, unknown>).items as Record<string, unknown>[] || []);
      setTopSupplies((top.data as Record<string, unknown>).items as Record<string, unknown>[] || []);
    } catch (e: unknown) { toast.error("加载失败", { description: (e as Error).message }); }
    finally { setLoading(false); }
  }, []);

  const loadCompare = useCallback(async (params: Record<string, string>) => {
    setLoading(true);
    try { setCompare(((await canteenApi.analytics.monthlyCompare(params)).data as Record<string, unknown>).items as Record<string, unknown>[] || []); }
    catch (e: unknown) { toast.error("加载失败", { description: (e as Error).message }); }
    finally { setLoading(false); }
  }, []);

  useEffect(() => {
    if (period === "month") loadMonth(month);
    else if (period === "half") loadCompare({ from: half === "h1" ? `${year}-01` : `${year}-07`, to: half === "h1" ? `${year}-06` : `${year}-12` });
    else loadCompare({ year });
  }, [period, month, year, half, loadMonth, loadCompare]);

  // 趋势图数据
  const trendChartData = dailyTrend.map((d) => ({ ...d, expense: (Number(d.expense) || 0) + (Number(d.share_expense) || 0) }));

  // 支出构成饼图数据
  const expensePieData = [
    { name: "食材采购", value: Number(expenseBreakdown.food) || 0 },
    ...((expenseBreakdown.others as Record<string, unknown>[]) || []).map((o) => ({ name: o.category as string, value: Number(o.amount) || 0 })),
  ].filter((d) => d.value > 0);

  // 食材分类饼图
  const foodPieData = foodShare.map((f) => ({ name: (f.category as string) || "未分类", value: Number(f.amount) || 0 }));

  // 月度对比数据
  const compareData = compare.map((c) => ({
    month: c.month as string, totalIncome: (Number(c.income) || 0) + (Number(c.resource) || 0),
    expense: (Number(c.food) || 0) + (Number(c.other) || 0),
    profit: (Number(c.income) || 0) + (Number(c.resource) || 0) - (Number(c.food) || 0) - (Number(c.other) || 0),
    count: Number(c.count) || 0, perCapita: Number(c.perCapita) || 0,
  }));

  // 每日盈亏明细
  const dailyTable = dailyTrend.map((d, i) => {
    const share = Number(d.share_expense) || 0, purchase = Number(d.expense) || 0;
    const income = Number(d.income) || 0, breakfast = Number(d.breakfast) || 0, resource = Number(d.resource) || 0;
    const totalExpense = purchase + share, costPerCapita = Number(d.count) ? ((totalExpense - breakfast - resource) / Number(d.count)).toFixed(2) : "-";
    return { 序号: i + 1, 日期: d.date as string, 收入: income, 采购支出: purchase, 分摊支出: share, 盈亏: income - totalExpense, 人次: Number(d.count) || 0, 人均成本: costPerCapita };
  });

  // 导出CSV
  const exportCsv = () => {
    if (!dailyTable.length) return;
    const header = "序号,日期,收入,采购支出,分摊支出,盈亏,人次,人均成本";
    const lines = dailyTable.map((r) => [r.序号, r.日期, r.收入.toFixed(2), r.采购支出.toFixed(2), r.分摊支出.toFixed(2), r.盈亏.toFixed(2), r.人次, r.人均成本].join(","));
    const csv = "\uFEFF" + header + "\n" + lines.join("\n");
    const blob = new Blob([csv], { type: "text/csv;charset=utf-8" }); const a = document.createElement("a"); a.href = URL.createObjectURL(blob); a.download = `食堂收支明细_${month}.csv`; a.click(); URL.revokeObjectURL(a.href);
    toast.success("导出成功");
  };

  return (
    <div className="space-y-4">
      <Card>
        <CardContent className="p-4 flex items-center justify-between flex-wrap gap-2">
          <div className="flex gap-1">
            {(["month", "half", "year"] as Period[]).map((p) => (
              <Button key={p} size="sm" variant={period === p ? "default" : "outline"} onClick={() => setPeriod(p)}>{p === "month" ? "月度" : p === "half" ? "半年度" : "年度"}</Button>
            ))}
          </div>
          <div className="flex items-center gap-2">
            {period === "month" && <Input type="month" className="h-8 w-40" value={month} onChange={(e) => setMonth(e.target.value)} />}
            {period === "half" && <><Input className="h-8 w-20" value={year} onChange={(e) => setYear(e.target.value)} /><select className="h-8 rounded-md border px-2 text-sm bg-background" value={half} onChange={(e) => setHalf(e.target.value as "h1" | "h2")}><option value="h1">上半年</option><option value="h2">下半年</option></select></>}
            {period === "year" && <><Input className="h-8 w-24" value={year} onChange={(e) => setYear(e.target.value)} /><Button size="sm" variant="outline" onClick={() => loadCompare({ year })}>查询</Button></>}
          </div>
        </CardContent>
      </Card>

      {loading && <p className="text-sm text-muted-foreground text-center py-4">加载中…</p>}

      {!loading && period === "month" && (
        <>
          <KpiCards summary={summary} />
          <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
            <Card><CardHeader className="pb-2"><CardTitle className="text-sm">每日收支趋势</CardTitle></CardHeader><CardContent className="h-64"><DailyTrendChart data={trendChartData} /></CardContent></Card>
            <Card><CardHeader className="pb-2"><CardTitle className="text-sm">支出构成</CardTitle></CardHeader><CardContent className="h-64"><ExpensePieChart data={expensePieData} /></CardContent></Card>
            <Card><CardHeader className="pb-2"><CardTitle className="text-sm">食材采购分类占比</CardTitle></CardHeader><CardContent className="h-64"><FoodShareChart data={foodPieData} /></CardContent></Card>
            <Card><CardHeader className="pb-2"><CardTitle className="text-sm">采购金额 Top 10 食材</CardTitle></CardHeader><CardContent className="h-64"><TopSuppliesChart data={topSupplies} /></CardContent></Card>
          </div>
          <DailyDetailTable dailyTable={dailyTable} onExport={exportCsv} />
        </>
      )}

      {!loading && period !== "month" && (
        <>
          <KpiCards summary={compare.length > 0 ? {
            income: { total: compareData.reduce((s: number, c) => s + c.totalIncome, 0), count: compareData.reduce((s: number, c) => s + c.count, 0) },
            expense: { total: compareData.reduce((s: number, c) => s + c.expense, 0) }, profit: compareData.reduce((s: number, c) => s + c.profit, 0),
          } as Record<string, unknown> : null} />
          <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
            <Card><CardHeader className="pb-2"><CardTitle className="text-sm">月度收支对比</CardTitle></CardHeader><CardContent className="h-72"><MonthlyCompareChart data={compareData as unknown as Record<string, unknown>[]} /></CardContent></Card>
            <Card><CardHeader className="pb-2"><CardTitle className="text-sm">各月人次与人均</CardTitle></CardHeader><CardContent className="h-72"><PeoplePerCapitaChart data={compareData as unknown as Record<string, unknown>[]} /></CardContent></Card>
            <CompareTable data={compareData} />
          </div>
        </>
      )}
    </div>
  );
}
