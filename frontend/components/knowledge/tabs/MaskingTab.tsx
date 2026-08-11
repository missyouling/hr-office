"use client";

import { useState, useEffect, useCallback } from "react";
import { Plus, Trash2, EyeOff, Loader2 } from "lucide-react";
import { toast } from "sonner";

import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { AlertDialog, AlertDialogAction, AlertDialogCancel, AlertDialogContent, AlertDialogDescription, AlertDialogFooter, AlertDialogHeader, AlertDialogTitle } from "@/components/ui/alert-dialog";
import { Skeleton } from "@/components/ui/skeleton";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Badge } from "@/components/ui/badge";
import { Separator } from "@/components/ui/separator";
import { Textarea } from "@/components/ui/textarea";
import { PermissionGate } from "@/components/permission-gate";

import { knowledgeApi, type KnowledgeBase, type KBFieldMask } from "@/lib/api-knowledge";
import { getSourceModuleLabel, getVisibilityLabel, getMaskPatternLabel, getRoleLabel } from "../utils";

/** 空状态 */
function EmptyMasks() {
  return (
    <div className="flex h-32 flex-col items-center justify-center text-muted-foreground">
      <EyeOff className="mb-2 h-8 w-8 opacity-40" />
      <p className="text-sm">暂无脱敏规则</p>
      <p className="mt-1 text-xs">点击「添加规则」配置敏感字段脱敏</p>
    </div>
  );
}

/** 脱敏规则表单 Dialog */
interface MaskFormDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  kbId: number;
  onSaved: () => void;
}

function MaskFormDialog({ open, onOpenChange, kbId, onSaved }: MaskFormDialogProps) {
  const [fieldName, setFieldName] = useState("");
  const [maskPattern, setMaskPattern] = useState("front3back4");
  const [exemptRole, setExemptRole] = useState("");
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    if (!open) {
      setFieldName("");
      setMaskPattern("front3back4");
      setExemptRole("");
    }
  }, [open]);

  const handleSave = async () => {
    const trimmed = fieldName.trim();
    if (!trimmed) {
      toast.error("请输入字段名");
      return;
    }
    setSaving(true);
    try {
      await knowledgeApi.addMask(kbId, {
        field_name: trimmed,
        mask_pattern: maskPattern,
        exempt_role: exemptRole || null,
      });
      toast.success("脱敏规则已添加");
      onSaved();
      onOpenChange(false);
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : "操作失败";
      toast.error("添加规则失败", { description: msg });
    } finally {
      setSaving(false);
    }
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-[560px]">
        <DialogHeader>
          <DialogTitle>添加脱敏规则</DialogTitle>
          <DialogDescription>为知识库中的敏感字段配置脱敏处理</DialogDescription>
        </DialogHeader>
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
          <div className="space-y-1.5">
            <Label>
              字段名 <span className="text-red-500">*</span>
            </Label>
            <Input
              value={fieldName}
              onChange={(e) => setFieldName(e.target.value)}
              placeholder="如：身份证号、手机号"
            />
          </div>
          <div className="space-y-1.5">
            <Label>脱敏模式</Label>
            <Select value={maskPattern} onValueChange={setMaskPattern}>
              <SelectTrigger><SelectValue /></SelectTrigger>
              <SelectContent>
                <SelectItem value="front3back4">前 3 后 4（如 138****1234）</SelectItem>
                <SelectItem value="all_star">全星号（如 ****）</SelectItem>
              </SelectContent>
            </Select>
          </div>
          <div className="space-y-1.5 sm:col-span-2">
            <Label>豁免角色</Label>
            <Select value={exemptRole} onValueChange={setExemptRole}>
              <SelectTrigger><SelectValue placeholder="无豁免（所有用户可见脱敏结果）" /></SelectTrigger>
              <SelectContent>
                <SelectItem value="admin">管理员（脱敏豁免）</SelectItem>
                <SelectItem value="manager">经理（脱敏豁免）</SelectItem>
              </SelectContent>
            </Select>
          </div>
        </div>
        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>取消</Button>
          <Button onClick={handleSave} disabled={saving}>
            {saving ? <Loader2 className="mr-2 h-4 w-4 animate-spin" /> : null}
            确认添加
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

export default function MaskingTab() {
  const [kbs, setKbs] = useState<KnowledgeBase[]>([]);
  const [kbsLoading, setKbsLoading] = useState(true);
  const [selectedKBId, setSelectedKBId] = useState<string>("");
  const [masks, setMasks] = useState<KBFieldMask[]>([]);
  const [masksLoading, setMasksLoading] = useState(false);
  const [formOpen, setFormOpen] = useState(false);
  const [deleteTarget, setDeleteTarget] = useState<KBFieldMask | null>(null);

  const fetchKBs = useCallback(async () => {
    setKbsLoading(true);
    try {
      const res = await knowledgeApi.list();
      setKbs(res.items ?? []);
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : "加载失败";
      toast.error("加载知识库列表失败", { description: msg });
    } finally {
      setKbsLoading(false);
    }
  }, []);

  const fetchMasks = useCallback(async () => {
    const id = Number(selectedKBId);
    if (!id) {
      setMasks([]);
      return;
    }
    setMasksLoading(true);
    try {
      const res = await knowledgeApi.listMasks(id);
      setMasks(res.items ?? []);
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : "加载失败";
      toast.error("加载脱敏规则失败", { description: msg });
    } finally {
      setMasksLoading(false);
    }
  }, [selectedKBId]);

  useEffect(() => {
    fetchKBs();
  }, [fetchKBs]);

  useEffect(() => {
    fetchMasks();
  }, [fetchMasks]);

  const handleDelete = async () => {
    const kbId = Number(selectedKBId);
    if (!deleteTarget) return;
    try {
      await knowledgeApi.removeMask(kbId, deleteTarget.id);
      toast.success("脱敏规则已删除");
      setDeleteTarget(null);
      fetchMasks();
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : "删除失败";
      toast.error("删除规则失败", { description: msg });
    }
  };

  const selectedKB = kbs.find((k) => String(k.id) === selectedKBId);

  return (
    <div className="space-y-4">
      {/* 上下文选择 */}
      <Card>
        <CardContent className="pt-4">
          <div className="flex flex-wrap items-center gap-3">
            <div className="space-y-1.5 min-w-[240px]">
              <Label>选择知识库</Label>
              {kbsLoading ? (
                <Skeleton className="h-10 w-full" />
              ) : (
                <Select value={selectedKBId} onValueChange={setSelectedKBId}>
                  <SelectTrigger>
                    <SelectValue placeholder="请选择知识库…" />
                  </SelectTrigger>
                  <SelectContent>
                    {kbs.map((kb) => (
                      <SelectItem key={kb.id} value={String(kb.id)}>
                        {kb.name}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              )}
            </div>
            {selectedKB && (
              <div className="flex items-center gap-2 text-sm text-muted-foreground">
                <Badge variant="outline">{getSourceModuleLabel(selectedKB.source_module)}</Badge>
                <Badge variant="secondary">{getVisibilityLabel(selectedKB.visibility)}</Badge>
              </div>
            )}
            <div className="ml-auto">
              <PermissionGate resource="knowledge_base" action="edit">
                <Button onClick={() => setFormOpen(true)} disabled={!selectedKBId}>
                  <Plus className="mr-2 h-4 w-4" />添加规则
                </Button>
              </PermissionGate>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* 脱敏规则表格 */}
      <Card>
        <CardContent className="p-0">
          <ScrollArea className="h-[calc(100vh-500px)] rounded-md border">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead className="text-xs font-medium text-muted-foreground">字段名</TableHead>
                  <TableHead className="text-xs font-medium text-muted-foreground">脱敏模式</TableHead>
                  <TableHead className="text-xs font-medium text-muted-foreground">豁免角色</TableHead>
                  <TableHead className="text-xs font-medium text-muted-foreground">更新时间</TableHead>
                  <TableHead className="text-xs font-medium text-muted-foreground text-center w-[80px]">操作</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {!selectedKBId ? (
                  <TableRow>
                    <TableCell colSpan={5} className="h-32 text-center text-muted-foreground">
                      请先选择知识库
                    </TableCell>
                  </TableRow>
                ) : masksLoading ? (
                  <TableRow>
                    <TableCell colSpan={5} className="h-32 text-center text-muted-foreground">
                      加载中…
                    </TableCell>
                  </TableRow>
                ) : masks.length === 0 ? (
                  <TableRow>
                    <TableCell colSpan={5} className="h-32">
                      <EmptyMasks />
                    </TableCell>
                  </TableRow>
                ) : (
                  masks.map((mask) => (
                    <TableRow key={mask.id}>
                      <TableCell className="text-sm font-mono">{mask.field_name}</TableCell>
                      <TableCell className="text-sm">
                        <Badge variant="outline">{getMaskPatternLabel(mask.mask_pattern)}</Badge>
                      </TableCell>
                      <TableCell className="text-sm">
                        {mask.exempt_role ? getRoleLabel(mask.exempt_role) : "—"}
                      </TableCell>
                      <TableCell className="text-sm text-muted-foreground">
                        {new Date(mask.updated_at).toLocaleString("zh-CN")}
                      </TableCell>
                      <TableCell className="text-center">
                        <PermissionGate resource="knowledge_base" action="edit">
                          <Button
                            variant="ghost"
                            size="icon"
                            className="h-8 w-8 text-destructive hover:text-destructive"
                            onClick={() => setDeleteTarget(mask)}
                          >
                            <Trash2 className="h-4 w-4" />
                          </Button>
                        </PermissionGate>
                      </TableCell>
                    </TableRow>
                  ))
                )}
              </TableBody>
            </Table>
          </ScrollArea>
        </CardContent>
      </Card>

      {/* 脱敏规则测试区 */}
      <Card>
        <CardContent className="pt-4">
          <div className="space-y-4">
            <div>
              <h3 className="text-sm font-medium">规则效果测试</h3>
              <p className="text-xs text-muted-foreground mt-1">输入样本文本查看当前知识库脱敏规则的处理效果</p>
            </div>
            <Separator />
            <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
              <div className="space-y-1.5">
                <Label>原始文本</Label>
                <Textarea placeholder="输入测试文本，如包含：张三的身份证号是 110101199001011234…" rows={4} />
              </div>
              <div className="space-y-1.5">
                <Label>脱敏后结果</Label>
                <div className="rounded-md border border-input bg-muted/50 px-3 py-2 min-h-[100px] text-sm text-muted-foreground">
                  {masks.length === 0
                    ? "当前知识库暂无脱敏规则，请先添加规则后再测试"
                    : "输入原始文本后，此处将显示脱敏后的结果（前端占位展示）"}
                </div>
              </div>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* 规则表单 Dialog */}
      <MaskFormDialog
        open={formOpen}
        onOpenChange={setFormOpen}
        kbId={Number(selectedKBId)}
        onSaved={fetchMasks}
      />

      {/* 删除确认 */}
      <AlertDialog open={!!deleteTarget} onOpenChange={(v) => { if (!v) setDeleteTarget(null); }}>
        <AlertDialogContent className="sm:max-w-[400px]">
          <AlertDialogHeader>
            <AlertDialogTitle>确认删除</AlertDialogTitle>
            <AlertDialogDescription>
              确定要删除字段「{deleteTarget?.field_name}」的脱敏规则吗？此操作不可撤销。
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>取消</AlertDialogCancel>
            <AlertDialogAction className="bg-destructive text-destructive-foreground hover:bg-destructive/90" onClick={handleDelete}>
              确认删除
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
}
