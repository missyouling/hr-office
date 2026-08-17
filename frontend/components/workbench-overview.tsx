"use client";

import { useEffect, useState } from "react";
import { BellRing, BookOpen, CloudSun, MessageSquare, Newspaper, Settings2, Sparkles } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Skeleton } from "@/components/ui/skeleton";
import { WorkbenchCalendar } from "@/components/workbench-calendar";
import { WorkbenchMemos } from "@/components/workbench-memos";
import {
  getWorkbenchConfig,
  getWorkbenchReminders,
  updateWorkbenchConfig,
  type WorkbenchConfig,
  type WorkbenchReminder,
  type WorkbenchReminderType,
} from "@/lib/api";

type CalendarInfo = { date: string; lunar: string };

const EMPTY_CONFIG: WorkbenchConfig = { weather: null, news: null };

const REMINDER_TYPE_LABELS: Record<WorkbenchReminderType, string> = {
  document_expiration: "档案到期",
  dorm_bill_due: "宿舍账单到期",
  invoice_pending: "发票待处理",
  payment_request_pending: "请款待处理",
};

function getGreeting(): string {
  const hour = new Date().getHours();
  if (hour < 12) return "上午好";
  if (hour < 18) return "下午好";
  return "晚上好";
}

function getCalendarInfo(): CalendarInfo {
  const date = new Date();
  const solar = new Intl.DateTimeFormat("zh-CN", {
    year: "numeric",
    month: "long",
    day: "numeric",
    weekday: "long",
  }).format(date);
  const lunar = new Intl.DateTimeFormat("zh-CN-u-ca-chinese", {
    month: "long",
    day: "numeric",
  }).format(date);
  return { date: solar, lunar: `农历 ${lunar}` };
}

function normalizeConfig(config: WorkbenchConfig): WorkbenchConfig {
  return {
    weather: config.weather && typeof config.weather.city === "string"
      ? { enabled: Boolean(config.weather.enabled), city: config.weather.city }
      : null,
    news: config.news && Array.isArray(config.news.categories)
      ? { enabled: Boolean(config.news.enabled), categories: config.news.categories.filter((item) => typeof item === "string") }
      : null,
  };
}

function normalizeReminders(items: WorkbenchReminder[]): WorkbenchReminder[] {
  const uniqueItems = new Map<string, WorkbenchReminder>();
  items.forEach((item) => {
    const key = `${item.reminder_type}:${item.id}`;
    if (!uniqueItems.has(key)) uniqueItems.set(key, item);
  });
  return [...uniqueItems.values()].sort((first, second) => {
    const firstDueAt = first.due_at ? new Date(first.due_at).getTime() : Number.POSITIVE_INFINITY;
    const secondDueAt = second.due_at ? new Date(second.due_at).getTime() : Number.POSITIVE_INFINITY;
    const dueDifference = firstDueAt - secondDueAt;
    if (dueDifference) return dueDifference;
    return first.reminder_type.localeCompare(second.reminder_type) || first.id - second.id;
  });
}

function formatReminderDueAt(dueAt: string | null): string {
  if (!dueAt) return "待处理";
  const date = new Date(dueAt);
  if (Number.isNaN(date.getTime())) return "待处理";
  return `到期：${new Intl.DateTimeFormat("zh-CN", { year: "numeric", month: "long", day: "numeric" }).format(date)}`;
}

function ReminderCard({
  reminders,
  isLoading,
  hasError,
  onReload,
}: {
  reminders: WorkbenchReminder[];
  isLoading: boolean;
  hasError: boolean;
  onReload: () => void;
}) {
  return (
    <Card className="border-amber-200/70 bg-gradient-to-br from-amber-50/80 via-card to-card shadow-sm dark:border-amber-900/50 dark:from-amber-950/20">
      <CardContent className="p-5">
        <div className="flex items-start gap-3">
          <span className="rounded-2xl bg-amber-500/10 p-3 text-amber-700 dark:text-amber-300"><BellRing className="h-5 w-5" /></span>
          <div><h3 className="font-semibold">待办提醒</h3><p className="mt-1 text-sm text-muted-foreground">集中查看近期到期与待处理事项</p></div>
        </div>
        {isLoading ? <Skeleton className="mt-4 h-24 rounded-xl" /> : hasError ? (
          <div className="mt-4 rounded-xl border border-destructive/20 bg-destructive/5 p-3 text-sm text-destructive">
            <p>提醒暂时无法加载，请稍后刷新重试。</p>
            <Button className="mt-2" size="sm" variant="outline" onClick={onReload}>重新加载</Button>
          </div>
        ) : reminders.length === 0 ? <p className="mt-4 text-sm text-muted-foreground">暂无待处理提醒。</p> : (
          <ul className="mt-4 divide-y divide-border/70" aria-label="工作台提醒列表">
            {reminders.map((reminder) => <li className="flex items-center justify-between gap-3 py-3 first:pt-0 last:pb-0" key={`${reminder.reminder_type}:${reminder.id}`}>
              <div className="min-w-0"><p className="text-xs font-medium text-amber-700 dark:text-amber-300">{REMINDER_TYPE_LABELS[reminder.reminder_type]}</p><p className="truncate text-sm font-medium">{reminder.title}</p><p className="mt-0.5 text-xs text-muted-foreground">状态：{reminder.status}</p></div>
              <span className="shrink-0 text-xs text-muted-foreground">{formatReminderDueAt(reminder.due_at)}</span>
            </li>)}
          </ul>
        )}
      </CardContent>
    </Card>
  );
}

function ConfigCard({
  type,
  config,
  onSave,
}: {
  type: "weather" | "news";
  config: WorkbenchConfig;
  onSave: (config: WorkbenchConfig) => Promise<void>;
}) {
  const isWeather = type === "weather";
  const current = config[type];
  const weatherConfig = config.weather;
  const newsConfig = config.news;
  const [isEditing, setIsEditing] = useState(false);
  const [value, setValue] = useState("");
  const [isSaving, setIsSaving] = useState(false);
  const title = isWeather ? "天气" : "新闻";
  const Icon = isWeather ? CloudSun : Newspaper;

  useEffect(() => {
    setValue(isWeather ? (config.weather?.city ?? "") : (config.news?.categories.join("、") ?? ""));
  }, [config, isWeather]);

  const save = async () => {
    const normalizedValue = value.trim();
    if (!normalizedValue) return;
    const nextConfig = isWeather
      ? { ...config, weather: { enabled: true, city: normalizedValue } }
      : { ...config, news: { enabled: true, categories: normalizedValue.split(/[、,，]/).map((item) => item.trim()).filter(Boolean) } };
    setIsSaving(true);
    try {
      await onSave(nextConfig);
      setIsEditing(false);
    } finally {
      setIsSaving(false);
    }
  };

  const status = current?.enabled ? "已配置" : current ? "已暂停" : "尚未配置";
  const description = isWeather
    ? weatherConfig?.city ? `城市：${weatherConfig.city}` : "设置城市后可在这里查看状态"
    : newsConfig?.categories.length ? `关注：${newsConfig.categories.join("、")}` : "设置关注分类后可在这里查看状态";

  return (
    <Card className="border-border/80 bg-card/85 shadow-sm transition-shadow hover:shadow-md">
      <CardContent className="p-5">
        <div className="flex items-start gap-3">
          <span className={`rounded-2xl p-3 ${isWeather ? "bg-sky-500/10 text-sky-600 dark:text-sky-400" : "bg-violet-500/10 text-violet-600 dark:text-violet-400"}`}><Icon className="h-5 w-5" /></span>
          <div className="min-w-0 flex-1">
            <div className="flex items-center justify-between gap-2"><h3 className="font-semibold">{title}</h3><span className="text-xs text-muted-foreground">{status}</span></div>
            <p className="mt-1 text-sm text-muted-foreground">{description}</p>
          </div>
        </div>
        {isEditing ? (
          <div className="mt-4 space-y-3">
            <Label htmlFor={`${type}-config`}>{isWeather ? "所在城市" : "新闻分类"}</Label>
            <Input id={`${type}-config`} value={value} onChange={(event) => setValue(event.target.value)} placeholder={isWeather ? "例如：杭州" : "例如：财经、科技"} />
            <div className="flex gap-2"><Button size="sm" onClick={save} disabled={!value.trim() || isSaving}>{isSaving ? "保存中…" : "保存并启用"}</Button><Button size="sm" variant="ghost" onClick={() => setIsEditing(false)} disabled={isSaving}>取消</Button></div>
          </div>
        ) : (
          <Button className="mt-4" size="sm" variant="outline" onClick={() => setIsEditing(true)}><Settings2 className="h-3.5 w-3.5" />{current ? "调整配置" : "立即配置"}</Button>
        )}
      </CardContent>
    </Card>
  );
}

export function WorkbenchOverview({ name }: { name?: string | null }) {
  const [calendar, setCalendar] = useState<CalendarInfo | null>(null);
  const [config, setConfig] = useState<WorkbenchConfig>(EMPTY_CONFIG);
  const [isConfigLoading, setIsConfigLoading] = useState(true);
  const [hasConfigError, setHasConfigError] = useState(false);
  const [reminders, setReminders] = useState<WorkbenchReminder[]>([]);
  const [isRemindersLoading, setIsRemindersLoading] = useState(true);
  const [hasRemindersError, setHasRemindersError] = useState(false);

  const loadConfig = async () => {
    setIsConfigLoading(true);
    setHasConfigError(false);
    try {
      setConfig(normalizeConfig(await getWorkbenchConfig()));
    } catch {
      setHasConfigError(true);
    } finally {
      setIsConfigLoading(false);
    }
  };

  const loadReminders = async () => {
    setIsRemindersLoading(true);
    setHasRemindersError(false);
    try {
      const response = await getWorkbenchReminders();
      setReminders(normalizeReminders(response.items));
    } catch {
      setHasRemindersError(true);
    } finally {
      setIsRemindersLoading(false);
    }
  };

  useEffect(() => {
    setCalendar(getCalendarInfo());
    void loadConfig();
    void loadReminders();
  }, []);

  const saveConfig = async (nextConfig: WorkbenchConfig) => {
    const saved = await updateWorkbenchConfig(nextConfig);
    setConfig(normalizeConfig(saved));
  };

  // 打开全局知识库问答面板（由 app/page.tsx 监听 dock:open-chat 并渲染 ChatPanel）
  const openKnowledgeChat = () => {
    window.dispatchEvent(new CustomEvent("dock:open-chat"));
  };

  return (
    <section className="grid gap-4 lg:grid-cols-[1.1fr_1.9fr]" aria-label="工作台概览">
      <Card className="relative overflow-hidden border-0 bg-gradient-to-br from-primary via-blue-600 to-indigo-700 text-primary-foreground shadow-[0_14px_34px_-18px_rgba(37,99,235,0.8)]">
        <div className="absolute -right-10 -top-10 h-36 w-36 rounded-full bg-white/10 blur-2xl" />
        <CardContent className="relative p-5"><Sparkles className="h-5 w-5 text-white/80" /><p className="mt-5 text-xl font-semibold">{getGreeting()}，{name || "同事"}</p><p className="mt-2 text-sm text-white/80">{calendar?.date ?? "正在同步日期…"}</p><p className="mt-1 text-sm text-white/80">{calendar?.lunar ?? "农历加载中…"}</p></CardContent>
      </Card>
      <Card className="border-border/80 bg-card/85 shadow-sm transition-shadow hover:shadow-md">
        <CardContent className="p-5">
          <div className="flex items-start gap-3">
            <span className="rounded-2xl p-3 bg-emerald-500/10 text-emerald-600 dark:text-emerald-400"><BookOpen className="h-5 w-5" /></span>
            <div className="min-w-0 flex-1">
              <h3 className="font-semibold">知识库问答</h3>
              <p className="mt-1 text-sm text-muted-foreground">基于已上架知识文档智能问答，可查询公司制度、流程与资料</p>
            </div>
          </div>
          <Button className="mt-4" size="sm" variant="outline" onClick={openKnowledgeChat}><MessageSquare className="h-3.5 w-3.5" />开始提问</Button>
        </CardContent>
      </Card>
      <ReminderCard reminders={reminders} isLoading={isRemindersLoading} hasError={hasRemindersError} onReload={() => void loadReminders()} />
      <WorkbenchCalendar />
      <WorkbenchMemos />
      {isConfigLoading ? <div className="grid gap-4 sm:grid-cols-2"><Skeleton className="h-48 rounded-xl" /><Skeleton className="h-48 rounded-xl" /></div> : hasConfigError ? <Card className="sm:col-span-2"><CardContent className="p-5"><div className="rounded-xl border border-destructive/20 bg-destructive/5 p-3 text-sm text-destructive"><p>工作台配置暂时无法加载，请稍后刷新重试。</p><Button className="mt-2" size="sm" variant="outline" onClick={() => void loadConfig()}>重新加载</Button></div></CardContent></Card> : <div className="grid gap-4 sm:grid-cols-2"><ConfigCard type="weather" config={config} onSave={saveConfig} /><ConfigCard type="news" config={config} onSave={saveConfig} /></div>}
    </section>
  );
}
