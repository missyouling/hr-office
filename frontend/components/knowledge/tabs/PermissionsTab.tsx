"use client";

import { useState, useEffect, useCallback } from "react";
import { Plus, Trash2, Shield, Loader2 } from "lucide-react";
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
import { PermissionGate } from "@/components/permission-gate";

import { knowledgeApi, type KnowledgeBase, type KBAccessRule, type KBAccessRulePayload } from "@/lib/api-knowledge";
import { getSourceModuleLabel, getVisibilityLabel, getRoleLabel } from "../utils";

/** 空状态 */
function EmptyRules() {
  return (
    <div className="flex h-32 flex-col items-center justify-center text-muted-foreground">
      <Shield className="mb-2 h-8 w-8 opacity-40" />
      <p className="text-sm">暂无访问规则</p>
      <p className="mt-1 text-xs">点击「添加规则」为知识库配置访问权限</p>
    </div>
  );
}

/** 规则表单 Dialog */
interface RuleFormDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  kbId: number;
  onSaved: () => void;
}

function RuleFormDialog({ open, onOpenChange, kbId, onSaved }: RuleFormDialogProps) {
  const [roleLevel, setRoleLevel] = useState("");
  const [departmentId, setDepartmentId] = useState("");
  const [userId, setUserId] = useState("");
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    if (!open) {
      setRoleLevel("");
      setDepartmentId("");
      setUserId("");
    }
  }, [open]);

  const handleSave = async () => {
    // 至少填写一项
    if (!roleLevel && !departmentId && !userId) {
      toast.error("至少需要填写角色、部门或用户之一");
      return;
    }

    const payload: KBAccessRulePayload = {};
    if (roleLevel) payload.role_level = roleLevel;
    if (departmentId) {
      const id = Number(departmentId);
      if (isNaN(id)) { toast.error("部门 ID 格式错误"); return; }
      payload.department_id = id;
    }
    if (userId) {
      const id = Number(userId);
      if (isNaN(id)) { toast.error("用户 ID 格式错误"); return; }
      payload.user_id = id;
    }

    setSaving(true);
    try {
      await knowledgeApi.addRule(kbId, payload);
      toast.success("访问规则已添加");
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
          <DialogTitle>添加访问规则</DialogTitle>
          <DialogDescription>至少指定角色、部门或用户之一，多条规则之间为 OR 关系</DialogDescription>
        </DialogHeader>
        <div className="grid grid-cols-1 gap-4">
          <div className="space-y-1.5">
            <Label>角色</Label>
            <Select value={roleLevel} onValueChange={setRoleLevel}>
              <SelectTrigger><SelectValue placeholder="不限角色" /></SelectTrigger>
              <SelectContent>
                <SelectItem value="admin">管理员</SelectItem>
                <SelectItem value="manager">经理</SelectItem>
                <SelectItem value="editor">编辑者</SelectItem>
                <SelectItem value="viewer">查看者</SelectItem>
              </SelectContent>
            </Select>
          </div>
          <div className="space-y-1.5">
            <Label>部门 ID</Label>
            <Input
              value={departmentId}
              onChange={(e) => setDepartmentId(e.target.value)}
              placeholder="输入部门数字 ID（可选）"
              type="number"
            />
          </div>
          <div className="space-y-1.5">
            <Label>用户 ID</Label>
            <Input
              value={userId}
              onChange={(e) => setUserId(e.target.value)}
              placeholder="输入用户数字 ID（可选）"
              type="number"
            />
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

export default function PermissionsTab() {
  const [kbs, setKbs] = useState<KnowledgeBase[]>([]);
  const [kbsLoading, setKbsLoading] = useState(true);
  const [selectedKBId, setSelectedKBId] = useState<string>("");
  const [rules, setRules] = useState<KBAccessRule[]>([]);
  const [rulesLoading, setRulesLoading] = useState(false);
  const [formOpen, setFormOpen] = useState(false);
  const [deleteTarget, setDeleteTarget] = useState<KBAccessRule | null>(null);

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

  const fetchRules = useCallback(async () => {
    const id = Number(selectedKBId);
    if (!id) {
      setRules([]);
      return;
    }
    setRulesLoading(true);
    try {
      const res = await knowledgeApi.listRules(id);
      setRules(res.items ?? []);
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : "加载失败";
      toast.error("加载访问规则失败", { description: msg });
    } finally {
      setRulesLoading(false);
    }
  }, [selectedKBId]);

  useEffect(() => {
    fetchKBs();
  }, [fetchKBs]);

  useEffect(() => {
    fetchRules();
  }, [fetchRules]);

  const handleDelete = async () => {
    const kbId = Number(selectedKBId);
    if (!deleteTarget) return;
    try {
      await knowledgeApi.removeRule(kbId, deleteTarget.id);
      toast.success("访问规则已删除");
      setDeleteTarget(null);
      fetchRules();
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

      {/* 规则表格 */}
      <Card>
        <CardContent className="p-0">
          <ScrollArea className="h-[calc(100vh-380px)] rounded-md border">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead className="text-xs font-medium text-muted-foreground">角色</TableHead>
                  <TableHead className="text-xs font-medium text-muted-foreground">部门 ID</TableHead>
                  <TableHead className="text-xs font-medium text-muted-foreground">用户 ID</TableHead>
                  <TableHead className="text-xs font-medium text-muted-foreground">创建时间</TableHead>
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
                ) : rulesLoading ? (
                  <TableRow>
                    <TableCell colSpan={5} className="h-32 text-center text-muted-foreground">
                      加载中…
                    </TableCell>
                  </TableRow>
                ) : rules.length === 0 ? (
                  <TableRow>
                    <TableCell colSpan={5} className="h-32">
                      <EmptyRules />
                    </TableCell>
                  </TableRow>
                ) : (
                  rules.map((rule) => (
                    <TableRow key={rule.id}>
                      <TableCell className="text-sm">
                        {rule.role_level ? getRoleLabel(rule.role_level) : "—"}
                      </TableCell>
                      <TableCell className="text-sm font-mono">
                        {rule.department_id ?? "—"}
                      </TableCell>
                      <TableCell className="text-sm font-mono">
                        {rule.user_id ?? "—"}
                      </TableCell>
                      <TableCell className="text-sm text-muted-foreground">
                        {new Date(rule.created_at).toLocaleString("zh-CN")}
                      </TableCell>
                      <TableCell className="text-center">
                        <PermissionGate resource="knowledge_base" action="edit">
                          <Button
                            variant="ghost"
                            size="icon"
                            className="h-8 w-8 text-destructive hover:text-destructive"
                            onClick={() => setDeleteTarget(rule)}
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

      {/* 规则表单 Dialog */}
      <RuleFormDialog
        open={formOpen}
        onOpenChange={setFormOpen}
        kbId={Number(selectedKBId)}
        onSaved={fetchRules}
      />

      {/* 删除确认 */}
      <AlertDialog open={!!deleteTarget} onOpenChange={(v) => { if (!v) setDeleteTarget(null); }}>
        <AlertDialogContent className="sm:max-w-[400px]">
          <AlertDialogHeader>
            <AlertDialogTitle>确认删除</AlertDialogTitle>
            <AlertDialogDescription>
              确定要删除此访问规则吗？此操作不可撤销。
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
