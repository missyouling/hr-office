"use client";

import { useState, useEffect, useCallback } from "react";
import { Card, CardContent } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogClose } from "@/components/ui/dialog";
import { toast } from "sonner";
import { canteenApi } from "@/lib/api-canteen";
import type { CanteenMenuTemplate, CanteenMenuDay } from "@/lib/api-canteen";
import { mondayOf, addDays } from "@/components/canteen/utils";
import { ChevronLeft, ChevronRight, Copy, Save, BookmarkPlus, FolderOpen, Trash2, Printer } from "lucide-react";

const MEALS = ["早餐", "午餐", "晚餐"] as const;
const DAY_NAMES = ["周一", "周二", "周三", "周四", "周五", "周六", "周日"];

/** 生成空菜单 */
function emptyDays(): CanteenMenuDay[] {
  return Array.from({ length: 7 }, (_, i) => ({ day_of_week: i + 1, breakfast: "", lunch: "", dinner: "" }));
}

export default function MenuTab() {
  const [weekStart, setWeekStart] = useState(() => mondayOf(new Date()));
  const [days, setDays] = useState<CanteenMenuDay[]>(emptyDays());
  const [templates, setTemplates] = useState<CanteenMenuTemplate[]>([]);
  const [tmplOpen, setTmplOpen] = useState(false);
  const [tmplName, setTmplName] = useState("");
  const [applyOpen, setApplyOpen] = useState(false);
  const [saved, setSaved] = useState(false);

  const load = useCallback(async () => {
    try {
      const r = (await canteenApi.menus.getByWeek(weekStart)).data;
      const loaded = (r as unknown as Record<string, unknown>).days as CanteenMenuDay[] || [];
      if (loaded.length) {
        setDays(loaded.map((d) => ({ ...emptyDays()[d.day_of_week - 1], ...d })));
        setSaved(true);
      } else {
        setDays(emptyDays());
        setSaved(false);
      }
    } catch { setDays(emptyDays()); setSaved(false); }
  }, [weekStart]);
  useEffect(() => { load(); }, [load]);

  const loadTemplates = useCallback(async () => {
    try { setTemplates((await canteenApi.menuTemplates.list()).data); } catch { /* ignore */ }
  }, []);
  useEffect(() => { loadTemplates(); }, [loadTemplates]);

  const updateDay = (idx: number, meal: string, val: string) => {
    setDays((ds) => ds.map((d, i) => (i === idx ? { ...d, [meal]: val } : d)));
  };

  const save = async () => {
    try {
      await canteenApi.menus.create({ week_start: weekStart, days });
      toast.success("菜单已保存");
      setSaved(true);
    } catch (e: unknown) { toast.error("保存失败", { description: (e as Error).message }); }
  };

  const copyPrev = async () => {
    const prev = addDays(weekStart, -7);
    try {
      await canteenApi.menus.copy({ source_week: prev, target_week: weekStart });
      toast.success("已复制上周菜单");
      load();
    } catch (e: unknown) { toast.error("复制失败", { description: (e as Error).message }); }
  };

  const saveTemplate = async () => {
    if (!tmplName.trim()) { toast.error("校验失败", { description: "模板名称不能为空" }); return; }
    try {
      await canteenApi.menuTemplates.create({ name: tmplName.trim(), days });
      toast.success("模板已保存");
      setTmplOpen(false); setTmplName(""); loadTemplates();
    } catch (e: unknown) { toast.error("保存失败", { description: (e as Error).message }); }
  };

  const applyTemplate = async (t: CanteenMenuTemplate) => {
    const tDays = (t as unknown as Record<string, unknown>).days as CanteenMenuDay[] || [];
    setDays(tDays.length ? tDays.map((d) => ({ ...emptyDays()[d.day_of_week - 1], ...d })) : emptyDays());
    toast.success(`已套用「${t.name}」`);
    setApplyOpen(false);
  };

  const deleteTemplate = async (id: number) => {
    try { await canteenApi.menuTemplates.remove(id); toast.success("已删除"); loadTemplates(); }
    catch (e: unknown) { toast.error("删除失败", { description: (e as Error).message }); }
  };

  const printPreview = () => {
    const rows = days.map((d) => `
      <tr>
        <td class="day">${DAY_NAMES[d.day_of_week - 1]}<br><span class="date">${addDays(weekStart, d.day_of_week - 1)}</span></td>
        <td>${d.breakfast || "&nbsp;"}</td><td>${d.lunch || "&nbsp;"}</td><td>${d.dinner || "&nbsp;"}</td>
      </tr>`).join("");
    const html = `<!doctype html>
<html><head><meta charset="utf-8"><title>每周菜单 ${weekStart}</title>
<style>
*{margin:0;padding:0;box-sizing:border-box}
body{font-family:"Microsoft YaHei","PingFang SC","Noto Sans SC",sans-serif;padding:40px 50px;color:#333;font-size:14px}
h1{font-size:24px;text-align:center;margin-bottom:6px}
.meta{text-align:center;color:#666;font-size:13px;margin-bottom:24px}
table{width:100%;border-collapse:collapse;margin-bottom:20px}
th{background:#1e40af;color:#fff;padding:10px 8px;text-align:center;font-size:14px}
td{padding:12px 8px;border:1px solid #d1d5db;text-align:center;font-size:14px;vertical-align:middle}
td.day{font-weight:bold;background:#f8fafc}
td .date{font-size:11px;color:#9ca3af;font-weight:normal}
@media print{body{padding:20px 30px}th{background:#1e40af!important;-webkit-print-color-adjust:exact;print-color-adjust:exact}}
</style></head><body>
<h1>食堂每周菜单</h1>
<div class="meta">日期：${weekStart} 至 ${addDays(weekStart, 6)}</div>
<table><thead><tr><th style="width:110px">星期</th><th>早餐</th><th>午餐</th><th>晚餐</th></tr></thead><tbody>${rows}</tbody></table>
<script>setTimeout(()=>window.print(),300)</script>
</body></html>`;
    const w = window.open("", "_blank");
    if (!w) { toast.error("浏览器拦截了打印窗口"); return; }
    w.document.write(html); w.document.close();
  };

  return (
    <div className="space-y-4">
      <Card>
        <CardContent className="p-4 space-y-3">
          <div className="flex items-center justify-between flex-wrap gap-2">
            <div className="flex items-center gap-1">
              <Button size="sm" variant="outline" onClick={() => setWeekStart(addDays(weekStart, -7))}><ChevronLeft className="h-4 w-4" /></Button>
              <Input type="date" className="h-8 w-40" value={weekStart} onChange={(e) => setWeekStart(e.target.value || weekStart)} />
              <Button size="sm" variant="outline" onClick={() => setWeekStart(addDays(weekStart, 7))}><ChevronRight className="h-4 w-4" /></Button>
              <Button size="sm" variant="ghost" onClick={() => setWeekStart(mondayOf(new Date()))}>本周</Button>
            </div>
            <div className="flex gap-2">
              <Button size="sm" variant="outline" onClick={copyPrev}><Copy className="mr-1 h-4 w-4" />复制上周</Button>
              <Button size="sm" variant="outline" onClick={printPreview}><Printer className="mr-1 h-4 w-4" />打印</Button>
              <Button size="sm" variant="outline" onClick={() => { setTmplName(""); setTmplOpen(true); }}><BookmarkPlus className="mr-1 h-4 w-4" />存为模板</Button>
              <Button size="sm" variant="outline" onClick={() => setApplyOpen(true)} disabled={templates.length === 0}><FolderOpen className="mr-1 h-4 w-4" />套用模板</Button>
              <Button size="sm" onClick={save}><Save className="mr-1 h-4 w-4" />保存</Button>
            </div>
          </div>
          <div className="text-xs text-muted-foreground">{saved ? "✅ 本周菜单已保存" : "⚠️ 本周菜单尚未保存"}</div>
        </CardContent>
      </Card>

      <Card>
        <CardContent className="p-4">
          <table className="w-full border-collapse text-sm">
            <thead>
              <tr>
                <th className="border p-2 bg-muted w-28">星期</th>
                {MEALS.map((m) => <th key={m} className="border p-2 bg-muted">{m}</th>)}
              </tr>
            </thead>
            <tbody>
              {days.map((d, idx) => (
                <tr key={d.day_of_week}>
                  <td className="border p-2 font-medium bg-muted/50 text-center">
                    {DAY_NAMES[idx]}<div className="text-[10px] text-muted-foreground font-normal">{addDays(weekStart, idx)}</div>
                  </td>
                  {MEALS.map((m) => (
                    <td key={m} className="border p-1">
                      <Input className="h-8 border-transparent hover:border-input focus:border-primary" placeholder={`${DAY_NAMES[idx]}${m}`}
                        value={(d as unknown as Record<string, string>)[m] || ""} onChange={(e) => updateDay(idx, m, e.target.value)} />
                    </td>
                  ))}
                </tr>
              ))}
            </tbody>
          </table>
        </CardContent>
      </Card>

      {/* 存为模板弹窗 */}
      <Dialog open={tmplOpen} onOpenChange={setTmplOpen}>
        <DialogContent className="sm:max-w-[380px]">
          <DialogHeader><DialogTitle>保存为菜单模板</DialogTitle></DialogHeader>
          <div className="py-2">
            <Input placeholder="模板名称（如：标准周菜单）" value={tmplName} onChange={(e) => setTmplName(e.target.value)} />
          </div>
          <div className="flex justify-end gap-2">
            <DialogClose asChild><Button variant="outline">取消</Button></DialogClose>
            <Button onClick={saveTemplate}>保存</Button>
          </div>
        </DialogContent>
      </Dialog>

      {/* 套用模板弹窗 */}
      <Dialog open={applyOpen} onOpenChange={setApplyOpen}>
        <DialogContent className="sm:max-w-[420px]">
          <DialogHeader><DialogTitle>套用菜单模板</DialogTitle></DialogHeader>
          <div className="space-y-2 py-2">
            {templates.length === 0 ? <p className="text-sm text-muted-foreground">暂无模板</p> :
              templates.map((t) => (
                <div key={t.id} className="flex items-center justify-between border rounded-md px-3 py-2">
                  <button className="text-sm hover:text-primary cursor-pointer text-left flex-1" onClick={() => applyTemplate(t)}>{t.name}</button>
                  <Button variant="ghost" size="icon" onClick={() => deleteTemplate(t.id)}><Trash2 className="h-4 w-4 text-red-500" /></Button>
                </div>
              ))}
          </div>
          <div className="flex justify-end">
            <DialogClose asChild><Button variant="outline">关闭</Button></DialogClose>
          </div>
        </DialogContent>
      </Dialog>
    </div>
  );
}
