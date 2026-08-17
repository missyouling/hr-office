"use client";

import { useEffect, useState } from "react";
import { CalendarDays, MapPin, Pencil, Plus, Trash2 } from "lucide-react";
import * as calendarApi from "@/lib/api";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { Checkbox } from "@/components/ui/checkbox";
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Skeleton } from "@/components/ui/skeleton";
import { Textarea } from "@/components/ui/textarea";

type CalendarForm = { title: string; startAt: string; endAt: string; location: string; notes: string; allDay: boolean };

const CALENDAR_RANGE_DAYS = 30;
const EMPTY_FORM: CalendarForm = { title: "", startAt: "", endAt: "", location: "", notes: "", allDay: false };

function getCalendarRange(): { from: string; to: string } {
  const from = new Date();
  const to = new Date(from);
  to.setDate(to.getDate() + CALENDAR_RANGE_DAYS);
  return { from: from.toISOString(), to: to.toISOString() };
}

function toLocalDateTime(value: string): string {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return "";
  return new Date(date.getTime() - date.getTimezoneOffset() * 60_000).toISOString().slice(0, 16);
}

function toForm(event: calendarApi.PersonalCalendarEvent): CalendarForm {
  return { title: event.title, startAt: toLocalDateTime(event.start_at), endAt: toLocalDateTime(event.end_at), location: event.location ?? "", notes: event.notes ?? "", allDay: event.all_day };
}

function formatEventTime(event: calendarApi.PersonalCalendarEvent): string {
  const start = new Date(event.start_at);
  const end = new Date(event.end_at);
  if (Number.isNaN(start.getTime()) || Number.isNaN(end.getTime())) return "时间待确认";
  const dateFormat = new Intl.DateTimeFormat("zh-CN", { month: "long", day: "numeric" });
  if (event.all_day) return `${dateFormat.format(start)} · 全天`;
  const timeFormat = new Intl.DateTimeFormat("zh-CN", { hour: "2-digit", minute: "2-digit", hour12: false });
  return `${dateFormat.format(start)} ${timeFormat.format(start)} · ${timeFormat.format(end)}`;
}

function sortEvents(events: calendarApi.PersonalCalendarEvent[]): calendarApi.PersonalCalendarEvent[] {
  return [...events].sort((first, second) => new Date(first.start_at).getTime() - new Date(second.start_at).getTime());
}

function getErrorMessage(error: unknown, fallback: string): string {
  return error instanceof Error && error.message ? error.message : fallback;
}

function hasCalendarApi(): boolean {
  return Object.keys(calendarApi).includes("getPersonalCalendar");
}

export function WorkbenchCalendar() {
  const [events, setEvents] = useState<calendarApi.PersonalCalendarEvent[]>([]);
  const [isLoading, setIsLoading] = useState(hasCalendarApi);
  const [loadError, setLoadError] = useState("");
  const [formError, setFormError] = useState("");
  const [isDialogOpen, setIsDialogOpen] = useState(false);
  const [editingEvent, setEditingEvent] = useState<calendarApi.PersonalCalendarEvent | null>(null);
  const [form, setForm] = useState<CalendarForm>(EMPTY_FORM);
  const [isSaving, setIsSaving] = useState(false);
  const [deletingId, setDeletingId] = useState<number | null>(null);

  const loadEvents = async () => {
    if (!hasCalendarApi()) return;
    setIsLoading(true);
    setLoadError("");
    try {
      const range = getCalendarRange();
      setEvents(sortEvents(await calendarApi.getPersonalCalendar(range.from, range.to)));
    } catch (error) {
      setLoadError(getErrorMessage(error, "日历暂时无法加载，请稍后刷新重试。"));
    } finally {
      setIsLoading(false);
    }
  };

  useEffect(() => { void loadEvents(); }, []);

  const openCreateDialog = () => {
    setEditingEvent(null);
    setForm(EMPTY_FORM);
    setFormError("");
    setIsDialogOpen(true);
  };

  const openEditDialog = (event: calendarApi.PersonalCalendarEvent) => {
    setEditingEvent(event);
    setForm(toForm(event));
    setFormError("");
    setIsDialogOpen(true);
  };

  const saveEvent = async () => {
    if (!form.title.trim() || !form.startAt || !form.endAt) return setFormError("请填写标题、开始时间和结束时间。");
    const startAt = new Date(form.startAt);
    const endAt = new Date(form.endAt);
    if (Number.isNaN(startAt.getTime()) || Number.isNaN(endAt.getTime()) || endAt < startAt) return setFormError("结束时间不能早于开始时间。");
    const payload: calendarApi.PersonalCalendarEventPayload = { title: form.title.trim(), start_at: startAt.toISOString(), end_at: endAt.toISOString(), location: form.location.trim(), notes: form.notes.trim(), all_day: form.allDay };
    setIsSaving(true);
    setFormError("");
    try {
      const saved = editingEvent ? await calendarApi.updatePersonalCalendarEvent(editingEvent.id, payload) : await calendarApi.createPersonalCalendarEvent(payload);
      setEvents((current) => sortEvents(editingEvent ? current.map((item) => item.id === saved.id ? saved : item) : [...current, saved]));
      setIsDialogOpen(false);
    } catch (error) {
      setFormError(getErrorMessage(error, "保存失败，请稍后重试。"));
    } finally {
      setIsSaving(false);
    }
  };

  const deleteEvent = async (id: number) => {
    setDeletingId(id);
    setLoadError("");
    try {
      await calendarApi.deletePersonalCalendarEvent(id);
      setEvents((current) => current.filter((event) => event.id !== id));
    } catch (error) {
      setLoadError(getErrorMessage(error, "删除失败，请稍后重试。"));
    } finally {
      setDeletingId(null);
    }
  };

  return <Card className="border-sky-200/70 bg-gradient-to-br from-sky-50/80 via-card to-card shadow-sm dark:border-sky-900/50 dark:from-sky-950/20"><CardContent className="p-5">
    <div className="flex items-start justify-between gap-3"><div className="flex items-start gap-3"><span className="rounded-2xl bg-sky-500/10 p-3 text-sky-700 dark:text-sky-300"><CalendarDays className="h-5 w-5" /></span><div><h3 className="font-semibold">我的日历</h3><p className="mt-1 text-sm text-muted-foreground">未来 30 天的个人安排</p></div></div><Button size="sm" onClick={openCreateDialog}><Plus />新增日程</Button></div>
    {isLoading ? <Skeleton className="mt-4 h-28 rounded-xl" /> : <><>{loadError && <div className="mt-4 rounded-xl border border-destructive/20 bg-destructive/5 p-3 text-sm text-destructive"><p>{loadError}</p><Button className="mt-2" size="sm" variant="outline" onClick={() => void loadEvents()}>重新加载</Button></div>}</>{events.length === 0 ? <p className="mt-4 rounded-xl border border-dashed border-sky-200/80 bg-background/50 p-4 text-sm text-muted-foreground dark:border-sky-900/60">未来 30 天暂无日程，安排一件重要的事吧。</p> : <ul className="mt-4 divide-y divide-border/70" aria-label="近期日程列表">{events.map((event) => <li className="flex items-start justify-between gap-3 py-3 first:pt-0 last:pb-0" key={event.id}><div className="min-w-0"><p className="truncate text-sm font-medium">{event.title}</p><p className="mt-1 text-xs text-sky-700 dark:text-sky-300">{formatEventTime(event)}</p>{event.location && <p className="mt-1 flex items-center gap-1 text-xs text-muted-foreground"><MapPin className="h-3 w-3" />{event.location}</p>}</div><div className="flex shrink-0 gap-1"><Button aria-label={`编辑 ${event.title}`} size="icon-sm" variant="ghost" onClick={() => openEditDialog(event)} disabled={deletingId === event.id}><Pencil /></Button><Button aria-label={`删除 ${event.title}`} size="icon-sm" variant="ghost" onClick={() => void deleteEvent(event.id)} disabled={deletingId === event.id}>{deletingId === event.id ? <span className="text-xs">删除中…</span> : <Trash2 className="text-destructive" />}</Button></div></li>)}</ul>}</>}
  </CardContent><Dialog open={isDialogOpen} onOpenChange={(open) => !isSaving && setIsDialogOpen(open)}><DialogContent><DialogHeader><DialogTitle>{editingEvent ? "编辑日程" : "新增日程"}</DialogTitle><DialogDescription>日程仅对当前登录用户可见。</DialogDescription></DialogHeader><div className="grid gap-3"><div className="grid gap-2"><Label htmlFor="calendar-title">标题</Label><Input id="calendar-title" value={form.title} onChange={(event) => setForm({ ...form, title: event.target.value })} /></div><div className="grid gap-2 sm:grid-cols-2"><div className="grid gap-2"><Label htmlFor="calendar-start">开始时间</Label><Input id="calendar-start" type="datetime-local" value={form.startAt} onChange={(event) => setForm({ ...form, startAt: event.target.value })} /></div><div className="grid gap-2"><Label htmlFor="calendar-end">结束时间</Label><Input id="calendar-end" type="datetime-local" value={form.endAt} onChange={(event) => setForm({ ...form, endAt: event.target.value })} /></div></div><div className="flex items-center gap-2"><Checkbox id="calendar-all-day" checked={form.allDay} onCheckedChange={(checked) => setForm({ ...form, allDay: checked === true })} /><Label htmlFor="calendar-all-day">全天</Label></div><div className="grid gap-2"><Label htmlFor="calendar-location">地点</Label><Input id="calendar-location" value={form.location} onChange={(event) => setForm({ ...form, location: event.target.value })} /></div><div className="grid gap-2"><Label htmlFor="calendar-notes">备注</Label><Textarea id="calendar-notes" value={form.notes} onChange={(event) => setForm({ ...form, notes: event.target.value })} /></div>{formError && <p role="alert" className="text-sm text-destructive">{formError}</p>}</div><DialogFooter><Button variant="outline" onClick={() => setIsDialogOpen(false)} disabled={isSaving}>取消</Button><Button onClick={() => void saveEvent()} disabled={isSaving}>{isSaving ? "保存中…" : "保存"}</Button></DialogFooter></DialogContent></Dialog></Card>;
}
