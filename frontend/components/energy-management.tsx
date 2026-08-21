"use client";

import { useEffect, useMemo, useState } from "react";
import { Building2, Droplets, Lightbulb, LoaderCircle, ChevronRight } from "lucide-react";

import { fetchDormitoryEnergySummary, type DormitoryEnergyBuilding, type DormitoryEnergySummary } from "@/lib/api-dormitory-energy";
import { useAuth } from "@/lib/supabase/auth-context";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Label } from "@/components/ui/label";

const EMPTY_SUMMARY: DormitoryEnergySummary = {
  overall: { electric: { usage: 0, amount: 0, count: 0 }, water: { usage: 0, amount: 0, count: 0 }, total_amount: 0 },
  by_building: [],
  rooms: [],
};

function currentMonth() {
  return new Date().toISOString().slice(0, 7);
}

export function EnergyManagement() {
  const { user, hasPermission } = useAuth();
  const canView = hasPermission("dormitory", "view");
  const [month, setMonth] = useState(currentMonth);
  const [buildingId, setBuildingId] = useState<number | null>(null);
  const [summary, setSummary] = useState<DormitoryEnergySummary>(EMPTY_SUMMARY);
  const [buildingOptions, setBuildingOptions] = useState<DormitoryEnergyBuilding[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState("");

  useEffect(() => {
    if (!user || !canView) return;
    let active = true;
    setIsLoading(true);
    setError("");
    fetchDormitoryEnergySummary({ month, buildingId: buildingId ?? undefined })
      .then((data) => {
        if (!active) return;
        setSummary(data);
        if (!buildingId) setBuildingOptions(data.by_building);
      })
      .catch(() => active && setError("能耗数据加载失败，请稍后重试。"))
      .finally(() => active && setIsLoading(false));
    return () => { active = false; };
  }, [buildingId, canView, month, user]);

  const selectedBuilding = useMemo(
    () => summary.by_building.find((item) => item.building_id === buildingId) ?? buildingOptions.find((item) => item.building_id === buildingId),
    [buildingId, buildingOptions, summary.by_building],
  );

  if (!user) return null;
  if (!canView) return <NoPermission />;

  return <div className="mx-auto flex w-full max-w-7xl flex-col gap-6 pb-6">
    <header className="relative overflow-hidden rounded-3xl bg-gradient-to-br from-slate-950 via-emerald-950 to-teal-800 px-6 py-7 text-white shadow-xl">
      <div className="absolute -right-10 -top-14 h-44 w-44 rounded-full bg-emerald-300/20 blur-3xl" />
      <div className="relative"><p className="text-sm text-emerald-100">日常事务 / 宿舍能耗</p><h1 className="mt-1 text-3xl font-semibold tracking-tight">能耗管理</h1><p className="mt-2 max-w-2xl text-sm text-emerald-50/80">按月查看宿舍电、水用量与费用，点击楼栋可查看房间明细。</p></div>
    </header>
    <Filters month={month} buildingId={buildingId} buildings={buildingOptions} onMonthChange={setMonth} onBuildingChange={setBuildingId} />
    {isLoading ? <LoadingState /> : error ? <ErrorState message={error} /> : <EnergyContent summary={summary} selectedBuilding={selectedBuilding} onSelectBuilding={setBuildingId} />}
  </div>;
}

function Filters({ month, buildingId, buildings, onMonthChange, onBuildingChange }: { month: string; buildingId: number | null; buildings: DormitoryEnergyBuilding[]; onMonthChange: (value: string) => void; onBuildingChange: (value: number | null) => void }) {
  return <Card><CardContent className="flex flex-col gap-4 pt-6 sm:flex-row sm:items-end"><Label className="grid gap-1.5 text-sm font-medium">月份<input aria-label="月份" type="month" value={month} onChange={(event) => onMonthChange(event.target.value)} className="h-9 rounded-md border border-input bg-background px-3 text-sm" /></Label><Label className="grid gap-1.5 text-sm font-medium">楼栋<select aria-label="楼栋筛选" value={buildingId ?? "all"} onChange={(event) => onBuildingChange(event.target.value === "all" ? null : Number(event.target.value))} className="h-9 min-w-44 rounded-md border border-input bg-background px-3 text-sm"><option value="all">全部楼栋</option>{buildings.map((item) => <option key={item.building_id} value={item.building_id}>{item.building_name}</option>)}</select></Label></CardContent></Card>;
}

function EnergyContent({ summary, selectedBuilding, onSelectBuilding }: { summary: DormitoryEnergySummary; selectedBuilding?: DormitoryEnergyBuilding; onSelectBuilding: (id: number | null) => void }) {
  if (!summary.by_building.length && !summary.rooms.length) return <EmptyState />;
  return <><section className="grid gap-4 md:grid-cols-3"><MetricCard label="电力用量" value={`${formatNumber(summary.overall.electric.usage)} kWh`} amount={summary.overall.electric.amount} icon={Lightbulb} tone="amber" /><MetricCard label="用水量" value={`${formatNumber(summary.overall.water.usage)} m³`} amount={summary.overall.water.amount} icon={Droplets} tone="sky" /><MetricCard label="电水费用合计" value={formatMoney(summary.overall.total_amount)} amount={null} icon={Building2} tone="emerald" /></section><BuildingTable buildings={summary.by_building} selectedBuildingId={selectedBuilding?.building_id} onSelectBuilding={onSelectBuilding} />{selectedBuilding && <RoomTable summary={summary} building={selectedBuilding} onBack={() => onSelectBuilding(null)} />}</>;
}

function MetricCard({ label, value, amount, icon: Icon, tone }: { label: string; value: string; amount: number | null; icon: typeof Lightbulb; tone: "amber" | "sky" | "emerald" }) { const colors = { amber: "bg-amber-100 text-amber-700", sky: "bg-sky-100 text-sky-700", emerald: "bg-emerald-100 text-emerald-700" }; return <Card><CardContent className="flex items-start justify-between p-5"><div><p className="text-sm text-muted-foreground">{label}</p><p className="mt-2 text-2xl font-semibold tracking-tight">{value}</p>{amount !== null && <p className="mt-1 text-sm text-muted-foreground">费用 {formatMoney(amount)}</p>}</div><span className={`rounded-2xl p-3 ${colors[tone]}`}><Icon className="h-5 w-5" /></span></CardContent></Card>; }

function BuildingTable({ buildings, selectedBuildingId, onSelectBuilding }: { buildings: DormitoryEnergyBuilding[]; selectedBuildingId?: number; onSelectBuilding: (id: number) => void }) { return <Card><CardHeader><CardTitle>楼栋汇总</CardTitle><CardDescription>选择一栋楼查看对应房间的电、水明细。</CardDescription></CardHeader><CardContent className="overflow-x-auto"><table className="w-full min-w-[700px] text-sm"><thead><tr className="border-b text-left text-muted-foreground"><th className="p-3">楼栋</th><th className="p-3">电力用量</th><th className="p-3">电费</th><th className="p-3">用水量</th><th className="p-3">水费</th><th className="p-3 text-right">合计</th></tr></thead><tbody>{buildings.map((item) => <tr key={item.building_id} className={`cursor-pointer border-b transition-colors last:border-0 hover:bg-muted/60 ${selectedBuildingId === item.building_id ? "bg-emerald-50/70" : ""}`} onClick={() => onSelectBuilding(item.building_id)}><td className="p-3 font-medium">{item.building_name}<ChevronRight className="ml-1 inline h-4 w-4 text-muted-foreground" /></td><td className="p-3">{formatNumber(item.electric.usage)} kWh</td><td className="p-3">{formatMoney(item.electric.amount)}</td><td className="p-3">{formatNumber(item.water.usage)} m³</td><td className="p-3">{formatMoney(item.water.amount)}</td><td className="p-3 text-right font-medium">{formatMoney(item.total_amount)}</td></tr>)}</tbody></table></CardContent></Card>; }

function RoomTable({ summary, building, onBack }: { summary: DormitoryEnergySummary; building: DormitoryEnergyBuilding; onBack: () => void }) { const rooms = summary.rooms.filter((room) => room.building_id === building.building_id); return <Card><CardHeader className="flex-row items-start justify-between gap-4"><div><CardTitle>{building.building_name} · 房间明细</CardTitle><CardDescription>只读展示当前筛选条件下的房间能耗。</CardDescription></div><button type="button" className="text-sm font-medium text-emerald-700 hover:underline" onClick={onBack}>返回全部楼栋</button></CardHeader><CardContent>{rooms.length ? <div className="overflow-x-auto"><table className="w-full min-w-[620px] text-sm"><thead><tr className="border-b text-left text-muted-foreground"><th className="p-3">房间</th><th className="p-3">电力用量 / 费用</th><th className="p-3">用水量 / 费用</th><th className="p-3 text-right">合计</th></tr></thead><tbody>{rooms.map((room) => <tr key={room.room_id} className="border-b last:border-0"><td className="p-3 font-medium">{room.room_number}</td><td className="p-3">{formatNumber(room.electric.usage)} kWh · {formatMoney(room.electric.amount)}</td><td className="p-3">{formatNumber(room.water.usage)} m³ · {formatMoney(room.water.amount)}</td><td className="p-3 text-right font-medium">{formatMoney(room.total_amount)}</td></tr>)}</tbody></table></div> : <p className="py-8 text-center text-sm text-muted-foreground">该楼栋暂无房间能耗数据。</p>}</CardContent></Card>; }

function NoPermission() { return <Card className="mx-auto mt-16 max-w-lg"><CardHeader><CardTitle>无能耗管理查看权限</CardTitle><CardDescription>请联系管理员申请 dormitory.view 权限后再查看宿舍能耗数据。</CardDescription></CardHeader></Card>; }
function LoadingState() { return <div className="flex min-h-64 items-center justify-center text-sm text-muted-foreground"><LoaderCircle className="mr-2 h-4 w-4 animate-spin" />正在加载能耗数据…</div>; }
function ErrorState({ message }: { message: string }) { return <Card><CardContent className="py-12 text-center text-sm text-muted-foreground">{message}</CardContent></Card>; }
function EmptyState() { return <Card><CardContent className="py-16 text-center"><Building2 className="mx-auto mb-3 h-9 w-9 text-muted-foreground" /><p className="font-medium">暂无能耗数据</p><p className="mt-1 text-sm text-muted-foreground">调整月份或楼栋筛选后再查看。</p></CardContent></Card>; }
function formatNumber(value: number) { return new Intl.NumberFormat("zh-CN", { maximumFractionDigits: 2 }).format(value); }
function formatMoney(value: number) { return `¥${formatNumber(value)}`; }
