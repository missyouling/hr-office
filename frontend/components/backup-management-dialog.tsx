"use client";

import { useState, useEffect } from "react";
import { toast } from "sonner";
import { Dialog, DialogContent, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { 
  AlertDialog, 
  AlertDialogAction, 
  AlertDialogCancel, 
  AlertDialogContent, 
  AlertDialogDescription, 
  AlertDialogFooter, 
  AlertDialogHeader, 
  AlertDialogTitle 
} from "@/components/ui/alert-dialog";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Checkbox } from "@/components/ui/checkbox";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Trash2 } from "lucide-react";
import { 
  Table, 
  TableBody, 
  TableCell, 
  TableHead, 
  TableHeader, 
  TableRow 
} from "@/components/ui/table";
import { 
  createLogBackup, 
  fetchLogBackups, 
  deleteLogBackup,
  cleanExpiredLogs,
  updateBackupSettings,
  getBackupSettings,
  type LogBackup
} from "@/lib/api";

interface BackupManagementDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

export function BackupManagementDialog({ open, onOpenChange }: BackupManagementDialogProps) {
  const [retentionDays, setRetentionDays] = useState("30");
  const [backupCron, setBackupCron] = useState("0 2 * * *");
  const [autoBackupEnabled, setAutoBackupEnabled] = useState(false);
  const [backups, setBackups] = useState<LogBackup[]>([]);
  const [loading, setLoading] = useState(false);
  const [creating, setCreating] = useState(false);
  const [saving, setSaving] = useState(false);
  const [deleteConfirmId, setDeleteConfirmId] = useState<number | null>(null);
  const [cleanConfirmOpen, setCleanConfirmOpen] = useState(false);

  useEffect(() => {
    if (open) {
      loadBackups();
      loadSettings();
    }
  }, [open]);

  const loadBackups = async () => {
    setLoading(true);
    try {
      const data = await fetchLogBackups();
      // Handle both direct array and { data: [...] } response
      const backupList = Array.isArray(data) ? data : ((data as unknown) as { data?: LogBackup[] })?.data || [];
      setBackups(backupList);
    } catch (error) {
      console.error("Failed to load backups:", error);
      setBackups([]);
    } finally {
      setLoading(false);
    }
  };

  const loadSettings = async () => {
    try {
      const settings = await getBackupSettings();
      setRetentionDays(settings.retention_days.toString());
      setBackupCron(settings.backup_cron);
      setAutoBackupEnabled(settings.auto_backup_enabled);
    } catch (error) {
      console.error("Failed to load backup settings:", error);
    }
  };

  const handleManualBackup = async () => {
    setCreating(true);
    try {
      await createLogBackup();
      toast.success("备份创建成功");
      loadBackups();
    } catch (error) {
      console.error("Backup failed:", error);
      toast.error("备份失败");
    } finally {
      setCreating(false);
    }
  };

  const handleDeleteBackup = (id: number) => {
    setDeleteConfirmId(id);
  };

  const executeDeleteBackup = async () => {
    if (deleteConfirmId === null) return;
    try {
      await deleteLogBackup(deleteConfirmId);
      toast.success("备份已删除");
      loadBackups();
    } catch (error) {
      console.error("Delete failed:", error);
      toast.error("删除失败");
    } finally {
      setDeleteConfirmId(null);
    }
  };

  const handleCleanExpired = () => {
    setCleanConfirmOpen(true);
  };

  const executeCleanExpired = async () => {
    try {
      const result = await cleanExpiredLogs();
      toast.success(`已清理 ${result.deleted} 条日志`);
    } catch (error) {
      console.error("Cleanup failed:", error);
      toast.error("清理失败");
    } finally {
      setCleanConfirmOpen(false);
    }
  };

  const handleSaveSettings = async () => {
    setSaving(true);
    try {
      await updateBackupSettings({
        retention_days: parseInt(retentionDays) || 30,
        auto_backup_enabled: autoBackupEnabled,
        backup_cron: backupCron,
      });
      if (autoBackupEnabled) {
        toast.success("自动备份设置已保存并启用");
      } else {
        toast.success("备份设置已保存", { 
          description: "注意：当前未启用自动备份" 
        });
      }
    } catch (error) {
      console.error("Save settings failed:", error);
      toast.error("保存设置失败");
    } finally {
      setSaving(false);
    }
  };

  const formatFileSize = (bytes: number) => {
    if (bytes < 1024) return bytes + " B";
    if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + " KB";
    return (bytes / (1024 * 1024)).toFixed(1) + " MB";
  };

  return (
    <>
      <Dialog open={open} onOpenChange={onOpenChange}>
        <DialogContent 
          className="max-w-none sm:max-w-[1000px] w-[90vw] max-h-[90vh] flex flex-col overflow-hidden p-0 bg-transparent border-none shadow-none"
          style={{ maxWidth: 'min(90vw, 1000px)' }}
        >
          <DialogHeader className="sr-only">
            <DialogTitle>备份设置</DialogTitle>
          </DialogHeader>
          <Card className="flex-1 flex flex-col overflow-hidden">
            <CardHeader className="border-b">
              <CardTitle>备份管理</CardTitle>
            </CardHeader>
            <CardContent className="flex-1 flex p-0 overflow-hidden">
              <div className="w-[350px] border-r p-6 space-y-6 flex flex-col bg-muted/10">
                <div className="space-y-4">
                  <div className="flex items-center gap-2">
                    <Checkbox 
                      id="auto-backup"
                      checked={autoBackupEnabled}
                      onCheckedChange={(checked) => setAutoBackupEnabled(checked as boolean)}
                    />
                    <Label htmlFor="auto-backup" className="cursor-pointer font-medium">启用自动备份</Label>
                  </div>

                  <div className="space-y-4 pt-2">
                    <div className="space-y-2">
                      <Label>自动保留天数</Label>
                      <Input 
                        type="number" 
                        value={retentionDays} 
                        onChange={(e) => setRetentionDays(e.target.value)}
                        placeholder="30"
                      />
                      <p className="text-xs text-muted-foreground">超过天数的日志将被自动清理</p>
                    </div>
                    
                    <div className="space-y-2">
                      <Label>定时备份 Cron</Label>
                      <Input 
                        value={backupCron} 
                        onChange={(e) => setBackupCron(e.target.value)}
                        placeholder="0 2 * * *"
                        disabled={!autoBackupEnabled}
                      />
                      <p className="text-xs text-muted-foreground">
                        示例：0 2 * * * (凌晨2点)、0 */6 * * * (每6小时)
                      </p>
                    </div>
                  </div>
                </div>

                <div className="pt-4 mt-auto space-y-2">
                  <Button 
                    className="w-full"
                    onClick={handleSaveSettings}
                    disabled={saving}
                  >
                    {saving ? "保存中..." : "保存"}
                  </Button>
                  <div className="grid grid-cols-2 gap-2">
                    <Button variant="outline" onClick={handleManualBackup} disabled={creating}>
                      {creating ? "备份中..." : "备份"}
                    </Button>
                    <Button variant="outline" onClick={handleCleanExpired}>
                      清理过期
                    </Button>
                  </div>
                </div>
              </div>

              <div className="flex-1 flex flex-col overflow-hidden">
                <ScrollArea className="flex-1">
                  <Table>
                    <TableHeader className="sticky top-0 bg-background z-10">
                      <TableRow>
                        <TableHead className="w-[45%]">文件名</TableHead>
                        <TableHead>大小</TableHead>
                        <TableHead>创建时间</TableHead>
                        <TableHead className="text-right">操作</TableHead>
                      </TableRow>
                    </TableHeader>
                    <TableBody>
                      {loading ? (
                        <TableRow>
                          <TableCell colSpan={4} className="h-32 text-center">加载中...</TableCell>
                        </TableRow>
                      ) : backups.length === 0 ? (
                        <TableRow>
                          <TableCell colSpan={4} className="h-32 text-center text-muted-foreground">暂无备份记录</TableCell>
                        </TableRow>
                      ) : (
                        backups.map((backup) => (
                          <TableRow key={backup.id}>
                            <TableCell className="font-mono text-xs break-all">{backup.filename}</TableCell>
                            <TableCell className="text-xs">{formatFileSize(backup.file_size)}</TableCell>
                            <TableCell className="text-xs whitespace-nowrap">{new Date(backup.created_at).toLocaleString("zh-CN")}</TableCell>
                            <TableCell className="text-right">
                              <Button 
                                variant="ghost" 
                                size="icon"
                                onClick={() => handleDeleteBackup(backup.id)}
                                className="h-8 w-8 text-destructive hover:text-destructive/80"
                              >
                                <Trash2 className="h-4 w-4" />
                              </Button>
                            </TableCell>
                          </TableRow>
                        ))
                      )}
                    </TableBody>
                  </Table>
                </ScrollArea>
              </div>
            </CardContent>
          </Card>
        </DialogContent>
      </Dialog>

      <AlertDialog open={deleteConfirmId !== null} onOpenChange={(open) => !open && setDeleteConfirmId(null)}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>确定删除备份？</AlertDialogTitle>
            <AlertDialogDescription>
              此操作将永久删除该备份文件，不可恢复。
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>取消</AlertDialogCancel>
            <AlertDialogAction onClick={executeDeleteBackup} className="bg-destructive text-destructive-foreground hover:bg-destructive/90">
              确认删除
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      <AlertDialog open={cleanConfirmOpen} onOpenChange={setCleanConfirmOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>确认清理过期日志？</AlertDialogTitle>
            <AlertDialogDescription>
              确定要删除超过 {retentionDays} 天的日志吗？此操作不可恢复。
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>取消</AlertDialogCancel>
            <AlertDialogAction onClick={executeCleanExpired} className="bg-destructive text-destructive-foreground hover:bg-destructive/90">
              确认清理
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </>
  );
}