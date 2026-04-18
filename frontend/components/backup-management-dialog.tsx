"use client";

import { useState, useEffect } from "react";
import { toast } from "sonner";
import { Dialog, DialogContent, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Checkbox } from "@/components/ui/checkbox";
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

  const handleDeleteBackup = async (id: number) => {
    if (!confirm("确定要删除此备份吗？")) return;
    try {
      await deleteLogBackup(id);
      toast.success("备份已删除");
      loadBackups();
    } catch (error) {
      console.error("Delete failed:", error);
      toast.error("删除失败");
    }
  };

  const handleCleanExpired = async () => {
    if (!confirm(`确定要删除超过 ${retentionDays} 天的日志吗？此操作不可恢复。`)) return;
    try {
      const result = await cleanExpiredLogs();
      toast.success(`已清理 ${result.deleted} 条日志`);
    } catch (error) {
      console.error("Cleanup failed:", error);
      toast.error("清理失败");
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
      toast.success("备份设置已保存");
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
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-lg">
        <DialogHeader>
          <DialogTitle>备份管理</DialogTitle>
        </DialogHeader>
        <div className="space-y-4">
          <div className="space-y-3 border rounded-lg p-4">
            <div className="flex items-center gap-2">
              <Checkbox 
                id="auto-backup"
                checked={autoBackupEnabled}
                onCheckedChange={(checked) => setAutoBackupEnabled(checked as boolean)}
              />
              <Label htmlFor="auto-backup" className="cursor-pointer">启用自动备份</Label>
            </div>

            <div className="grid grid-cols-2 gap-4">
              <div className="space-y-2">
                <Label>自动保留天数</Label>
                <Input 
                  type="number" 
                  value={retentionDays} 
                  onChange={(e) => setRetentionDays(e.target.value)}
                  placeholder="30"
                />
              </div>
              <div className="space-y-2">
                <Label>定时备份 Cron</Label>
                <Input 
                  value={backupCron} 
                  onChange={(e) => setBackupCron(e.target.value)}
                  placeholder="0 2 * * *"
                  disabled={!autoBackupEnabled}
                />
              </div>
            </div>

            <p className="text-xs text-muted-foreground">
              示例：0 2 * * * (每天凌晨2点)、0 */6 * * * (每6小时)
            </p>

            <Button 
              variant="outline" 
              size="sm" 
              onClick={handleSaveSettings}
              disabled={saving}
            >
              {saving ? "保存中..." : "保存设置"}
            </Button>
          </div>
          
          <div className="border rounded-lg">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>文件名</TableHead>
                  <TableHead>大小</TableHead>
                  <TableHead>创建时间</TableHead>
                  <TableHead>操作</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {loading ? (
                  <TableRow>
                    <TableCell colSpan={4} className="text-center">加载中...</TableCell>
                  </TableRow>
                ) : backups.length === 0 ? (
                  <TableRow>
                    <TableCell colSpan={4} className="text-center text-muted-foreground">暂无备份记录</TableCell>
                  </TableRow>
                ) : (
                  backups.map((backup) => (
                    <TableRow key={backup.id}>
                      <TableCell className="font-mono text-sm">{backup.filename}</TableCell>
                      <TableCell>{formatFileSize(backup.file_size)}</TableCell>
                      <TableCell>{new Date(backup.created_at).toLocaleString("zh-CN")}</TableCell>
                      <TableCell>
                        <Button 
                          variant="ghost" 
                          size="sm"
                          onClick={() => handleDeleteBackup(backup.id)}
                          className="text-red-500 hover:text-red-600"
                        >
                          删除
                        </Button>
                      </TableCell>
                    </TableRow>
                  ))
                )}
              </TableBody>
            </Table>
          </div>
          
          <div className="flex gap-2">
            <Button onClick={handleManualBackup} disabled={creating}>
              {creating ? "创建中..." : "手动备份"}
            </Button>
            <Button variant="outline" onClick={handleCleanExpired}>
              清理过期日志
            </Button>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  );
}