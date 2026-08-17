"use client";

import { useEffect, useState } from "react";
import { Check, Circle, ClipboardList, Pencil, Pin, PinOff, Plus, Trash2 } from "lucide-react";
import * as memoApi from "@/lib/api";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Skeleton } from "@/components/ui/skeleton";
import { Textarea } from "@/components/ui/textarea";

type MemoForm = { title: string; content: string };

const EMPTY_FORM: MemoForm = { title: "", content: "" };
const RECENT_MEMO_COUNT = 5;

function getErrorMessage(error: unknown, fallback: string): string {
  return error instanceof Error && error.message ? error.message : fallback;
}

function sortMemos(memos: memoApi.Memo[]): memoApi.Memo[] {
  return [...memos].sort((first, second) => Number(second.pinned) - Number(first.pinned) || new Date(second.updated_at).getTime() - new Date(first.updated_at).getTime());
}

function toPayload(memo: memoApi.Memo, changes: Partial<memoApi.MemoPayload> = {}): memoApi.MemoPayload {
  return { title: memo.title, content: memo.content, pinned: memo.pinned, completed: memo.completed, ...changes };
}

function hasMemoApi(): boolean {
  return Object.keys(memoApi).includes("getUserMemos");
}

export function WorkbenchMemos() {
  const [memos, setMemos] = useState<memoApi.Memo[]>([]);
  const [isLoading, setIsLoading] = useState(hasMemoApi);
  const [loadError, setLoadError] = useState("");
  const [formError, setFormError] = useState("");
  const [isDialogOpen, setIsDialogOpen] = useState(false);
  const [editingMemo, setEditingMemo] = useState<memoApi.Memo | null>(null);
  const [form, setForm] = useState<MemoForm>(EMPTY_FORM);
  const [isSaving, setIsSaving] = useState(false);
  const [deletingId, setDeletingId] = useState<number | null>(null);
  const [updatingId, setUpdatingId] = useState<number | null>(null);

  const loadMemos = async () => {
    if (!hasMemoApi()) return;
    setIsLoading(true);
    setLoadError("");
    try {
      const response = await memoApi.getUserMemos();
      setMemos(sortMemos(response.memos));
    } catch (error) {
      setLoadError(getErrorMessage(error, "备忘录暂时无法加载，请稍后重试。"));
    } finally {
      setIsLoading(false);
    }
  };

  useEffect(() => { void loadMemos(); }, []);

  const openForm = (memo?: memoApi.Memo) => {
    setEditingMemo(memo ?? null);
    setForm(memo ? { title: memo.title, content: memo.content } : EMPTY_FORM);
    setFormError("");
    setIsDialogOpen(true);
  };

  const saveMemo = async () => {
    if (!form.title.trim()) return setFormError("请填写标题。");
    setIsSaving(true);
    setFormError("");
    const payload = editingMemo ? toPayload(editingMemo, { title: form.title.trim(), content: form.content.trim() }) : { title: form.title.trim(), content: form.content.trim(), pinned: false, completed: false };
    try {
      const saved = editingMemo ? await memoApi.updateUserMemo(editingMemo.id, payload) : await memoApi.createUserMemo(payload);
      setMemos((current) => sortMemos(editingMemo ? current.map((memo) => memo.id === saved.id ? saved : memo) : [...current, saved]));
      setIsDialogOpen(false);
    } catch (error) {
      setFormError(getErrorMessage(error, "保存失败，请稍后重试。"));
    } finally {
      setIsSaving(false);
    }
  };

  const updateMemo = async (memo: memoApi.Memo, changes: Partial<memoApi.MemoPayload>) => {
    setUpdatingId(memo.id);
    setLoadError("");
    try {
      const saved = await memoApi.updateUserMemo(memo.id, toPayload(memo, changes));
      setMemos((current) => sortMemos(current.map((item) => item.id === saved.id ? saved : item)));
    } catch (error) {
      setLoadError(getErrorMessage(error, "更新失败，请稍后重试。"));
    } finally {
      setUpdatingId(null);
    }
  };

  const deleteMemo = async (id: number) => {
    setDeletingId(id);
    setLoadError("");
    try {
      await memoApi.deleteUserMemo(id);
      setMemos((current) => current.filter((memo) => memo.id !== id));
    } catch (error) {
      setLoadError(getErrorMessage(error, "删除失败，请稍后重试。"));
    } finally {
      setDeletingId(null);
    }
  };

  const recentMemos = sortMemos(memos).slice(0, RECENT_MEMO_COUNT);
  return <Card className="border-rose-200/70 bg-gradient-to-br from-rose-50/80 via-card to-card shadow-sm dark:border-rose-900/50 dark:from-rose-950/20"><CardContent className="p-5">
    <div className="flex items-start justify-between gap-3"><div className="flex items-start gap-3"><span className="rounded-2xl bg-rose-500/10 p-3 text-rose-700 dark:text-rose-300"><ClipboardList className="h-5 w-5" /></span><div><h3 className="font-semibold">我的备忘录</h3><p className="mt-1 text-sm text-muted-foreground">记录当下，优先处理重要事项</p></div></div><Button size="sm" onClick={() => openForm()}><Plus />新增</Button></div>
    {isLoading ? <Skeleton className="mt-4 h-28 rounded-xl" /> : <>{loadError && <div className="mt-4 rounded-xl border border-destructive/20 bg-destructive/5 p-3 text-sm text-destructive"><p>{loadError}</p><Button className="mt-2" size="sm" variant="outline" onClick={() => void loadMemos()}>重新加载</Button></div>}{recentMemos.length === 0 ? <p className="mt-4 rounded-xl border border-dashed border-rose-200/80 bg-background/50 p-4 text-sm text-muted-foreground dark:border-rose-900/60">暂无备忘录，记下第一件重要的事吧。</p> : <ul className="mt-4 divide-y divide-border/70" aria-label="近期备忘录列表">{recentMemos.map((memo) => <li className="flex items-start gap-2 py-3 first:pt-0 last:pb-0" key={memo.id}><Button aria-label={`${memo.completed ? "标为未完成" : "标为已完成"} ${memo.title}`} className="mt-0.5 shrink-0" size="icon-sm" variant="ghost" disabled={updatingId === memo.id || deletingId === memo.id} onClick={() => void updateMemo(memo, { completed: !memo.completed })}>{memo.completed ? <Check className="text-emerald-600" /> : <Circle />}</Button><div className="min-w-0 flex-1"><p className={`truncate text-sm font-medium ${memo.completed ? "text-muted-foreground line-through" : ""}`}>{memo.title}</p>{memo.content && <p className="mt-1 line-clamp-2 text-xs text-muted-foreground">{memo.content}</p>}</div><div className="flex shrink-0 gap-0.5"><Button aria-label={`${memo.pinned ? "取消置顶" : "置顶"} ${memo.title}`} size="icon-sm" variant="ghost" disabled={updatingId === memo.id || deletingId === memo.id} onClick={() => void updateMemo(memo, { pinned: !memo.pinned })}>{memo.pinned ? <Pin className="text-rose-600" /> : <PinOff />}</Button><Button aria-label={`编辑 ${memo.title}`} size="icon-sm" variant="ghost" disabled={updatingId === memo.id || deletingId === memo.id} onClick={() => openForm(memo)}><Pencil /></Button><Button aria-label={`删除 ${memo.title}`} size="icon-sm" variant="ghost" disabled={updatingId === memo.id || deletingId === memo.id} onClick={() => void deleteMemo(memo.id)}>{deletingId === memo.id ? <span className="text-xs">删除中…</span> : <Trash2 className="text-destructive" />}</Button></div></li>)}</ul>}</>}
  </CardContent><Dialog open={isDialogOpen} onOpenChange={(open) => !isSaving && setIsDialogOpen(open)}><DialogContent><DialogHeader><DialogTitle>{editingMemo ? "编辑备忘录" : "新增备忘录"}</DialogTitle><DialogDescription>备忘录仅对当前登录用户可见。</DialogDescription></DialogHeader><div className="grid gap-3"><div className="grid gap-2"><Label htmlFor="memo-title">标题</Label><Input id="memo-title" value={form.title} onChange={(event) => setForm({ ...form, title: event.target.value })} /></div><div className="grid gap-2"><Label htmlFor="memo-content">内容</Label><Textarea id="memo-content" value={form.content} onChange={(event) => setForm({ ...form, content: event.target.value })} /></div>{formError && <p role="alert" className="text-sm text-destructive">{formError}</p>}</div><DialogFooter><Button variant="outline" onClick={() => setIsDialogOpen(false)} disabled={isSaving}>取消</Button><Button onClick={() => void saveMemo()} disabled={isSaving}>{isSaving ? "保存中…" : "保存"}</Button></DialogFooter></DialogContent></Dialog></Card>;
}
