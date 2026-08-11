"use client";

import { useState, useEffect, useCallback } from "react";
import { Search, Plus, BookOpen, MoreHorizontal, Trash2, Edit, Eye, Shield, FilterX } from "lucide-react";
import { toast } from "sonner";

import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { AlertDialog, AlertDialogAction, AlertDialogCancel, AlertDialogContent, AlertDialogDescription, AlertDialogFooter, AlertDialogHeader, AlertDialogTitle } from "@/components/ui/alert-dialog";
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger, DropdownMenuSeparator } from "@/components/ui/dropdown-menu";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { PermissionGate } from "@/components/permission-gate";
import { useAuth } from "@/lib/auth";
import { normalizeRole } from "@/lib/permissions";
import { knowledgeApi, type KnowledgeBase, type KBStats } from "@/lib/api-knowledge";
import { getSourceModuleLabel, getVisibilityLabel, formatChunkingConfig } from "../utils";

// ========== 源模块图标映射 ==========

/** 根据 source_module 返回对应图标（emoji 兜底） */
function getModuleIcon(module: string): string {
  const icons: Record<string, string> = {
    employee: "👤",
    insurance: "🏥",
    canteen: "🍽️",
    invoice: "🧾",
    dormitory: "🏠",
    archives: "📁",
    "office-supply": "📦",
    announcement: "📢",
    custom: "📚",
  };
  return icons[module] ?? "📚";
}

/** cursor: default 占位，防止卡片上按钮事件冒泡 */
function stopPropagation(e: React.MouseEvent) {
  e.stopPropagation();
}

// ========== KPI 卡片骨架 ==========
function StatsSkeleton() {
  return (
    <div className="grid grid-cols-2 gap-4 lg:grid-cols-4">
      {Array.from({ length: 4 }).map((_, i) => (
        <Card key={i}>
          <CardContent className="p-4">
            <Skeleton className="h-4 w-20" />
            <Skeleton className="mt-2 h-8 w-12" />
          </CardContent>
        </Card>
      ))}
    </div>
  );
}

// ========== 空状态 ==========
function EmptyKnowledgeBases({ isAdmin }: { isAdmin: boolean }) {
  return (
    <div className="flex h-64 flex-col items-center justify-center text-muted-foreground">
      <BookOpen className="mb-3 h-12 w-12 opacity-40" />
      <p className="text-base">暂无知识库</p>
      <p className="mt-1 text-sm">{isAdmin ? "点击「新建知识库」开始创建" : "请联系管理员创建知识库"}</p>
    </div>
  );
}

// ========== 知识库表单 Dialog ==========
interface KBFormDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  existing?: KnowledgeBase; // undefined → 新建
  onSaved: () => void;
}

function KBFormDialog({ open, onOpenChange, existing, onSaved }: KBFormDialogProps) {
  const [name, setName] = useState("");
  const [description, setDescription] = useState("");
  const [sourceModule, setSourceModule] = useState("custom");
  const [visibility, setVisibility] = useState("restricted");
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    if (open) {
      if (existing) {
        setName(existing.name);
        setDescription(existing.description ?? "");
        setSourceModule(existing.source_module);
        setVisibility(existing.visibility);
      } else {
        setName("");
        setDescription("");
        setSourceModule("custom");
        setVisibility("restricted");
      }
    }
  }, [open, existing]);

  const handleSave = async () => {
    if (!name.trim()) {
      toast.error("请输入知识库名称");
      return;
    }
    setSaving(true);
    try {
      if (existing) {
        await knowledgeApi.update(existing.id, {
          name: name.trim(),
          description: description.trim(),
          source_module: sourceModule,
          visibility,
        });
        toast.success("知识库已更新");
      } else {
        await knowledgeApi.create({
          name: name.trim(),
          description: description.trim(),
          source_module: sourceModule,
          visibility,
        });
        toast.success("知识库已创建");
      }
      onSaved();
      onOpenChange(false);
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : "操作失败";
      toast.error(existing ? "更新失败" : "创建失败", { description: msg });
    } finally {
      setSaving(false);
    }
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-[560px]">
        <DialogHeader>
          <DialogTitle>{existing ? "编辑知识库" : "新建知识库"}</DialogTitle>
          <DialogDescription>
            {existing ? "修改知识库的基本信息" : "创建新的知识库，用于存储和管理智能问答的底层知识"}
          </DialogDescription>
        </DialogHeader>
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
          <div className="space-y-1.5 sm:col-span-2">
            <Label>
              名称 <span className="text-red-500">*</span>
            </Label>
            <Input value={name} onChange={(e) => setName(e.target.value)} placeholder="知识库名称" />
          </div>
          <div className="space-y-1.5 sm:col-span-2">
            <Label>描述</Label>
            <Textarea value={description} onChange={(e) => setDescription(e.target.value)} placeholder="知识库用途说明（可选）" rows={3} />
          </div>
          <div className="space-y-1.5">
            <Label>来源模块</Label>
            <Select value={sourceModule} onValueChange={setSourceModule}>
              <SelectTrigger><SelectValue /></SelectTrigger>
              <SelectContent>
                <SelectItem value="custom">自定义</SelectItem>
                <SelectItem value="employee">员工花名册</SelectItem>
                <SelectItem value="insurance">社保业务</SelectItem>
                <SelectItem value="canteen">食堂管理</SelectItem>
                <SelectItem value="invoice">发票管理</SelectItem>
                <SelectItem value="dormitory">宿舍管理</SelectItem>
                <SelectItem value="archives">档案管理</SelectItem>
                <SelectItem value="office-supply">办公劳保</SelectItem>
                <SelectItem value="announcement">公告通知</SelectItem>
              </SelectContent>
            </Select>
          </div>
          <div className="space-y-1.5">
            <Label>可见性</Label>
            <Select value={visibility} onValueChange={setVisibility}>
              <SelectTrigger><SelectValue /></SelectTrigger>
              <SelectContent>
                <SelectItem value="public">公开</SelectItem>
                <SelectItem value="restricted">受限</SelectItem>
                <SelectItem value="private">私有</SelectItem>
              </SelectContent>
            </Select>
          </div>
        </div>
        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>取消</Button>
          <Button onClick={handleSave} disabled={saving}>
            {saving ? "保存中…" : "保存"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

// ======== 主组件 ========

export default function KnowledgeListTab() {
  const { user } = useAuth();
  const role = normalizeRole(user?.role ?? "viewer");
  const isAdmin = role === "admin" || role === "super_admin";

  const [kbs, setKbs] = useState<KnowledgeBase[]>([]);
  const [stats, setStats] = useState<KBStats | null>(null);
  const [loading, setLoading] = useState(true);
  const [statsLoading, setStatsLoading] = useState(isAdmin);
  const [error, setError] = useState<string | null>(null);

  // 筛选状态
  const [keyword, setKeyword] = useState("");
  const [sourceFilter, setSourceFilter] = useState("all");

  // Dialog 状态
  const [formOpen, setFormOpen] = useState(false);
  const [editingKB, setEditingKB] = useState<KnowledgeBase | undefined>(undefined);
  const [deleteTarget, setDeleteTarget] = useState<KnowledgeBase | null>(null);

  const fetchKBs = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const res = await knowledgeApi.list();
      setKbs(res.items ?? []);
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : "加载失败";
      setError(msg);
      toast.error("加载知识库列表失败", { description: msg });
    } finally {
      setLoading(false);
    }
  }, []);

  const fetchStats = useCallback(async () => {
    if (!isAdmin) return;
    setStatsLoading(true);
    try {
      const res = await knowledgeApi.stats();
      setStats(res);
    } catch {
      // 静默失败，stats 非关键数据
    } finally {
      setStatsLoading(false);
    }
  }, [isAdmin]);

  useEffect(() => {
    fetchKBs();
    fetchStats();
  }, [fetchKBs, fetchStats]);

  // 筛选
  const filtered = kbs.filter((kb) => {
    if (keyword && !kb.name.toLowerCase().includes(keyword.toLowerCase())) return false;
    if (sourceFilter !== "all" && kb.source_module !== sourceFilter) return false;
    return true;
  });

  const distinctModules = [...new Set(kbs.map((k) => k.source_module))];

  const handleEdit = (kb: KnowledgeBase) => {
    setEditingKB(kb);
    setFormOpen(true);
  };

  const handleCreate = () => {
    setEditingKB(undefined);
    setFormOpen(true);
  };

  const handleDeleteConfirm = async () => {
    if (!deleteTarget) return;
    try {
      await knowledgeApi.remove(deleteTarget.id);
      toast.success("知识库已删除");
      setDeleteTarget(null);
      fetchKBs();
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : "删除失败";
      toast.error("删除失败", { description: msg });
    }
  };

  return (
    <div className="space-y-4">
      {/* KPI 概览卡片行（仅 admin 可见） */}
      {isAdmin && (
        <>
          {statsLoading ? (
            <StatsSkeleton />
          ) : stats ? (
            <div className="grid grid-cols-2 gap-4 lg:grid-cols-4">
              <Card>
                <CardContent className="p-4">
                  <p className="text-xs text-muted-foreground">知识库总数</p>
                  <p className="mt-1 text-2xl font-bold font-mono">{stats.total_count}</p>
                </CardContent>
              </Card>
              <Card>
                <CardContent className="p-4">
                  <p className="text-xs text-muted-foreground">系统模板</p>
                  <p className="mt-1 text-2xl font-bold font-mono">{stats.system_count}</p>
                </CardContent>
              </Card>
              <Card>
                <CardContent className="p-4">
                  <p className="text-xs text-muted-foreground">自定义库</p>
                  <p className="mt-1 text-2xl font-bold font-mono">{stats.custom_count}</p>
                </CardContent>
              </Card>
              <Card>
                <CardContent className="p-4">
                  <p className="text-xs text-muted-foreground">来源模块</p>
                  <p className="mt-1 text-2xl font-bold font-mono">{stats.by_source_module?.length ?? 0}</p>
                </CardContent>
              </Card>
            </div>
          ) : null}
        </>
      )}

      {/* 筛选工具栏 */}
      <Card>
        <CardContent className="pt-4">
          <div className="flex flex-wrap items-center gap-3">
            <div className="relative flex-1 min-w-[200px]">
              <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
              <Input placeholder="搜索知识库名称…" className="pl-9" value={keyword} onChange={(e) => setKeyword(e.target.value)} />
            </div>
            <Select value={sourceFilter} onValueChange={setSourceFilter}>
              <SelectTrigger className="w-[160px]">
                <SelectValue placeholder="全部来源模块" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">全部来源模块</SelectItem>
                {distinctModules.map((m) => (
                  <SelectItem key={m} value={m}>{getSourceModuleLabel(m)}</SelectItem>
                ))}
              </SelectContent>
            </Select>
            {(keyword || sourceFilter !== "all") && (
              <Button variant="outline" size="icon" onClick={() => { setKeyword(""); setSourceFilter("all"); }} aria-label="重置筛选">
                <FilterX className="h-4 w-4" />
              </Button>
            )}
            <PermissionGate resource="knowledge_base" action="create">
              <Button onClick={handleCreate}>
                <Plus className="mr-2 h-4 w-4" />新建知识库
              </Button>
            </PermissionGate>
          </div>
        </CardContent>
      </Card>

      {/* 加载态 */}
      {loading && (
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3 2xl:grid-cols-4">
          {Array.from({ length: 4 }).map((_, i) => (
            <Card key={i}>
              <CardContent className="p-4">
                <Skeleton className="h-5 w-3/4" />
                <Skeleton className="mt-2 h-4 w-full" />
                <Skeleton className="mt-2 h-4 w-2/3" />
                <Skeleton className="mt-4 h-3 w-1/2" />
              </CardContent>
            </Card>
          ))}
        </div>
      )}

      {/* 错误态 */}
      {!loading && error && (
        <Card>
          <CardContent className="flex h-32 flex-col items-center justify-center gap-2">
            <p className="text-muted-foreground">{error}</p>
            <Button variant="outline" onClick={fetchKBs}>重新加载</Button>
          </CardContent>
        </Card>
      )}

      {/* 空态 */}
      {!loading && !error && filtered.length === 0 && (
        <EmptyKnowledgeBases isAdmin={isAdmin} />
      )}

      {/* 知识库卡片网格 */}
      {!loading && !error && filtered.length > 0 && (
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3 2xl:grid-cols-4">
          {filtered.map((kb) => (
            <Card key={kb.id} className="group relative flex flex-col">
              <CardContent className="flex flex-1 flex-col p-4">
                <div className="flex items-start justify-between gap-2">
                  <div className="flex items-center gap-2 min-w-0">
                    <span className="text-lg shrink-0">{getModuleIcon(kb.source_module)}</span>
                    <h3 className="text-base font-semibold truncate">{kb.name}</h3>
                  </div>
                  <div className="flex items-center gap-1 shrink-0">
                    <Badge variant={kb.visibility === "public" ? "default" : "secondary"} className="text-xs">
                      {getVisibilityLabel(kb.visibility)}
                    </Badge>
                    {kb.is_system && (
                      <Badge variant="outline" className="text-xs">系统</Badge>
                    )}
                  </div>
                </div>
                <p className="mt-1 line-clamp-2 text-sm text-muted-foreground min-h-[2.5rem]">
                  {kb.description || "暂无描述"}
                </p>
                <div className="mt-3 flex items-center gap-3 text-xs text-muted-foreground">
                  <Badge variant="outline" className="text-xs font-normal">
                    {getSourceModuleLabel(kb.source_module)}
                  </Badge>
                  <span className="font-mono">{formatChunkingConfig(kb.chunking_config)}</span>
                </div>
                <div className="mt-auto pt-3 flex items-center justify-between border-t border-border/50">
                  <span className="text-xs text-muted-foreground">
                    更新于 {new Date(kb.updated_at).toLocaleDateString("zh-CN")}
                  </span>
                  <div onClick={stopPropagation}>
                    <DropdownMenu>
                      <DropdownMenuTrigger asChild>
                        <Button variant="ghost" size="icon" className="h-8 w-8">
                          <MoreHorizontal className="h-4 w-4" />
                        </Button>
                      </DropdownMenuTrigger>
                      <DropdownMenuContent align="end">
                        <DropdownMenuItem onClick={() => handleEdit(kb)}>
                          <Edit className="mr-2 h-4 w-4" />编辑
                        </DropdownMenuItem>
                        <PermissionGate resource="knowledge_base" action="view">
                          <DropdownMenuItem onClick={() => handleEdit(kb)}>
                            <Eye className="mr-2 h-4 w-4" />详情
                          </DropdownMenuItem>
                        </PermissionGate>
                        <PermissionGate resource="knowledge_base" action="edit">
                          <DropdownMenuItem onClick={() => handleEdit(kb)}>
                            <Shield className="mr-2 h-4 w-4" />管理
                          </DropdownMenuItem>
                        </PermissionGate>
                        <DropdownMenuSeparator />
                        <PermissionGate resource="knowledge_base" action="delete">
                          <DropdownMenuItem
                            className="text-destructive focus:text-destructive"
                            disabled={kb.is_system}
                            onClick={() => setDeleteTarget(kb)}
                          >
                            <Trash2 className="mr-2 h-4 w-4" />
                            {kb.is_system ? "系统模板不可删" : "删除"}
                          </DropdownMenuItem>
                        </PermissionGate>
                      </DropdownMenuContent>
                    </DropdownMenu>
                  </div>
                </div>
              </CardContent>
            </Card>
          ))}
        </div>
      )}

      {/* 知识库表单 Dialog */}
      <KBFormDialog open={formOpen} onOpenChange={setFormOpen} existing={editingKB} onSaved={fetchKBs} />

      {/* 删除确认 AlertDialog */}
      <AlertDialog open={!!deleteTarget} onOpenChange={(v) => { if (!v) setDeleteTarget(null); }}>
        <AlertDialogContent className="sm:max-w-[400px]">
          <AlertDialogHeader>
            <AlertDialogTitle>确认删除</AlertDialogTitle>
            <AlertDialogDescription>
              确定要删除知识库「{deleteTarget?.name}」吗？此操作不可撤销。
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>取消</AlertDialogCancel>
            <AlertDialogAction className="bg-destructive text-destructive-foreground hover:bg-destructive/90" onClick={handleDeleteConfirm}>确认删除</AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
}
