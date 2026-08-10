"use client";

import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { fmt } from "@/components/canteen/utils";
import {
  ResponsiveContainer, Line, BarChart, Bar, PieChart, Pie, Cell,
  XAxis, YAxis, CartesianGrid, Tooltip, Legend, ComposedChart,
} from "recharts";
import { Download } from "lucide-react";

const CHART_COLORS = [
  "hsl(var(--chart-1))", "hsl(var(--chart-2))", "hsl(var(--chart-3))",
  "hsl(var(--chart-4))", "hsl(var(--chart-5))", "#06b6d4", "#ec4899", "#84cc16",
];

// ---------- KPI 卡片 ----------
export function KpiCards({ summary }: { summary: Record<string, unknown> | null }) {
  if (!summary) return null;
  const income = (summary.income as Record<string, number>) || {};
  const expense = (summary.expense as Record<string, number>) || {};
  const profit = Number(summary.profit || 0);
  const count = Number((summary.income as Record<string, number>)?.count || 0);

  return (
    <div className="grid grid-cols-2 md:grid-cols-4 gap-3">
      <Card><CardContent className="p-4"><p className="text-xs text-muted-foreground">总收入</p><p className="text-2xl font-bold text-green-600">{fmt(income.total)}</p></CardContent></Card>
      <Card><CardContent className="p-4"><p className="text-xs text-muted-foreground">总支出</p><p className="text-2xl font-bold text-red-600">{fmt(expense.total)}</p></CardContent></Card>
      <Card><CardContent className="p-4"><p className="text-xs text-muted-foreground">盈亏</p><p className={`text-2xl font-bold ${profit >= 0 ? "text-blue-600" : "text-red-600"}`}>{fmt(profit)}</p></CardContent></Card>
      <Card><CardContent className="p-4"><p className="text-xs text-muted-foreground">人均成本</p><p className="text-2xl font-bold">{count > 0 ? fmt((expense.total || 0) / count) : "-"}</p></CardContent></Card>
    </div>
  );
}

// ---------- 每日收支趋势图 ----------
export function DailyTrendChart({ data }: { data: Record<string, unknown>[] }) {
  if (data.length === 0) return <p className="text-sm text-muted-foreground text-center pt-20">暂无数据</p>;
  return (
    <ResponsiveContainer width="100%" height="100%">
      <ComposedChart data={data}>
        <CartesianGrid strokeDasharray="3 3" /><XAxis dataKey="date" tick={{ fontSize: 10 }} /><YAxis tick={{ fontSize: 10 }} />
        <Tooltip formatter={(v: unknown) => fmt(Number(v))} /><Legend />
        <Bar dataKey="income" name="收入" fill="hsl(var(--chart-2))" />
        <Bar dataKey="expense" name="支出" fill="hsl(var(--chart-1))" />
        <Line type="monotone" dataKey="profit" name="盈亏" stroke="hsl(var(--chart-3))" strokeWidth={2} dot={false} />
      </ComposedChart>
    </ResponsiveContainer>
  );
}

// ---------- 支出构成饼图 ----------
export function ExpensePieChart({ data }: { data: { name: string; value: number }[] }) {
  if (data.length === 0) return <p className="text-sm text-muted-foreground text-center pt-20">暂无数据</p>;
  return (
    <ResponsiveContainer width="100%" height="100%">
      <PieChart>
        <Pie data={data} dataKey="value" nameKey="name" innerRadius={50} outerRadius={80} label={({ name, value }) => `${name} ${((value / data.reduce((s, d) => s + d.value, 0)) * 100).toFixed(0)}%`}>
          {data.map((_, i) => <Cell key={i} fill={CHART_COLORS[i % CHART_COLORS.length]} />)}
        </Pie>
        <Tooltip formatter={(v: unknown) => fmt(Number(v))} />
      </PieChart>
    </ResponsiveContainer>
  );
}

// ---------- 食材占比饼图 ----------
export function FoodShareChart({ data }: { data: { name: string; value: number }[] }) {
  if (data.length === 0) return <p className="text-sm text-muted-foreground text-center pt-20">暂无数据</p>;
  return (
    <ResponsiveContainer width="100%" height="100%">
      <PieChart>
        <Pie data={data} dataKey="value" nameKey="name" outerRadius={80} label={({ name }) => name}>
          {data.map((_, i) => <Cell key={i} fill={CHART_COLORS[i % CHART_COLORS.length]} />)}
        </Pie>
        <Tooltip formatter={(v: unknown) => fmt(Number(v))} />
      </PieChart>
    </ResponsiveContainer>
  );
}

// ---------- Top10 食材横向柱状图 ----------
export function TopSuppliesChart({ data }: { data: Record<string, unknown>[] }) {
  if (data.length === 0) return <p className="text-sm text-muted-foreground text-center pt-20">暂无数据</p>;
  return (
    <ResponsiveContainer width="100%" height="100%">
      <BarChart data={data} layout="vertical">
        <CartesianGrid strokeDasharray="3 3" /><XAxis type="number" tick={{ fontSize: 10 }} /><YAxis type="category" dataKey="name" width={80} tick={{ fontSize: 11 }} />
        <Tooltip formatter={(v: unknown) => fmt(Number(v))} /><Bar dataKey="amount" name="采购金额" fill="hsl(var(--primary))" radius={[0, 3, 3, 0]} />
      </BarChart>
    </ResponsiveContainer>
  );
}

// ---------- 月度对比图 ----------
export function MonthlyCompareChart({ data }: { data: Record<string, unknown>[] }) {
  if (data.length === 0) return <p className="text-sm text-muted-foreground text-center pt-24">暂无数据</p>;
  return (
    <ResponsiveContainer width="100%" height="100%">
      <ComposedChart data={data}>
        <CartesianGrid strokeDasharray="3 3" /><XAxis dataKey="month" tick={{ fontSize: 10 }} /><YAxis tick={{ fontSize: 10 }} />
        <Tooltip formatter={(v: unknown) => fmt(Number(v))} /><Legend />
        <Bar dataKey="totalIncome" name="收入" fill="hsl(var(--chart-2))" /><Bar dataKey="expense" name="支出" fill="hsl(var(--chart-1))" />
        <Line type="monotone" dataKey="profit" name="盈亏" stroke="hsl(var(--chart-3))" strokeWidth={2} />
      </ComposedChart>
    </ResponsiveContainer>
  );
}

// ---------- 人次/人均双Y轴图 ----------
export function PeoplePerCapitaChart({ data }: { data: Record<string, unknown>[] }) {
  if (data.length === 0) return <p className="text-sm text-muted-foreground text-center pt-24">暂无数据</p>;
  return (
    <ResponsiveContainer width="100%" height="100%">
      <ComposedChart data={data}>
        <CartesianGrid strokeDasharray="3 3" /><XAxis dataKey="month" tick={{ fontSize: 10 }} />
        <YAxis yAxisId="l" tick={{ fontSize: 10 }} /><YAxis yAxisId="r" orientation="right" tick={{ fontSize: 10 }} />
        <Tooltip /><Legend />
        <Bar yAxisId="l" dataKey="count" name="人次" fill="hsl(var(--chart-4))" />
        <Line yAxisId="r" type="monotone" dataKey="perCapita" name="人均" stroke="hsl(var(--chart-3))" strokeWidth={2} />
      </ComposedChart>
    </ResponsiveContainer>
  );
}

// ---------- 每日盈亏明细表 ----------
export function DailyDetailTable({ dailyTable, onExport }: {
  dailyTable: { 序号: number; 日期: string; 收入: number; 采购支出: number; 分摊支出: number; 盈亏: number; 人次: number; 人均成本: string }[];
  onExport: () => void;
}) {
  return (
    <Card>
      <CardHeader className="flex flex-row items-center justify-between pb-2">
        <CardTitle className="text-sm">每日盈亏明细</CardTitle>
        <Button size="sm" variant="outline" onClick={onExport}><Download className="mr-1 h-4 w-4" />导出</Button>
      </CardHeader>
      <CardContent className="p-0">
        <div className="relative max-h-[40vh] overflow-y-auto rounded-md border">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead className="w-12 text-center">序号</TableHead><TableHead className="w-28 text-center">日期</TableHead>
                <TableHead className="w-28 text-center">收入</TableHead><TableHead className="w-28 text-center">采购支出</TableHead>
                <TableHead className="w-28 text-center">分摊支出</TableHead><TableHead className="w-28 text-center">盈亏</TableHead>
                <TableHead className="w-20 text-center">人次</TableHead><TableHead className="w-24 text-center">人均成本</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {dailyTable.length === 0 ? (
                <TableRow><TableCell colSpan={8} className="h-16 text-center text-muted-foreground">本月无记录</TableCell></TableRow>
              ) : (<>
                {dailyTable.map((r) => (
                  <TableRow key={r.序号}>
                    <TableCell className="text-center text-muted-foreground">{r.序号}</TableCell><TableCell className="text-center">{r.日期}</TableCell>
                    <TableCell className="text-green-600 text-center">{fmt(r.收入)}</TableCell>
                    <TableCell className="text-center">{r.采购支出 ? fmt(r.采购支出) : "-"}</TableCell>
                    <TableCell className="text-center">{r.分摊支出 ? fmt(r.分摊支出) : "-"}</TableCell>
                    <TableCell className={`font-medium text-center ${r.盈亏 >= 0 ? "text-blue-600" : "text-red-600"}`}>{fmt(r.盈亏)}</TableCell>
                    <TableCell className="text-center">{r.人次 || "-"}</TableCell><TableCell className="text-center">{r.人均成本}</TableCell>
                  </TableRow>
                ))}
                <TableRow className="bg-muted/50 font-semibold">
                  <TableCell className="text-center" colSpan={2}>合计</TableCell>
                  <TableCell className="text-green-700 text-center">{fmt(dailyTable.reduce((s, r) => s + r.收入, 0))}</TableCell>
                  <TableCell className="text-center">{fmt(dailyTable.reduce((s, r) => s + r.采购支出, 0))}</TableCell>
                  <TableCell className="text-center">{fmt(dailyTable.reduce((s, r) => s + r.分摊支出, 0))}</TableCell>
                  <TableCell className={`text-center ${dailyTable.reduce((s, r) => s + r.盈亏, 0) >= 0 ? "text-blue-700" : "text-red-700"}`}>{fmt(dailyTable.reduce((s, r) => s + r.盈亏, 0))}</TableCell>
                  <TableCell className="text-center">{dailyTable.reduce((s, r) => s + r.人次, 0)}</TableCell><TableCell className="text-center">-</TableCell>
                </TableRow>
              </>)}
            </TableBody>
          </Table>
        </div>
      </CardContent>
    </Card>
  );
}

// ---------- 月度对比明细表 ----------
export function CompareTable({ data }: { data: { month: string; totalIncome: number; expense: number; profit: number; count: number; perCapita: number }[] }) {
  return (
    <Card className="lg:col-span-2">
      <CardHeader className="pb-2"><CardTitle className="text-sm">月度对比明细</CardTitle></CardHeader>
      <CardContent className="p-0">
        <div className="relative max-h-[40vh] overflow-y-auto rounded-md border">
          <Table>
            <TableHeader><TableRow><TableHead className="w-20 text-center">月份</TableHead><TableHead className="text-center">收入</TableHead><TableHead className="text-center">支出</TableHead><TableHead className="text-center">盈亏</TableHead><TableHead className="text-center">人次</TableHead><TableHead className="text-center">人均</TableHead></TableRow></TableHeader>
            <TableBody>
              {data.length === 0 ? <TableRow><TableCell colSpan={6} className="h-16 text-center text-muted-foreground">暂无数据</TableCell></TableRow> : data.map((c) => (
                <TableRow key={c.month}>
                  <TableCell className="text-center">{c.month}</TableCell><TableCell className="text-green-600 text-center">{fmt(c.totalIncome)}</TableCell>
                  <TableCell className="text-center">{fmt(c.expense)}</TableCell><TableCell className={`font-medium text-center ${c.profit >= 0 ? "text-blue-600" : "text-red-600"}`}>{fmt(c.profit)}</TableCell>
                  <TableCell className="text-center">{c.count || "-"}</TableCell><TableCell className="text-center">{c.perCapita || "-"}</TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </div>
      </CardContent>
    </Card>
  );
}
