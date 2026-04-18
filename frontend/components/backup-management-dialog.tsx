"use client";

import { useState, useEffect } from "react";
import { Dialog, DialogContent, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";

interface BackupManagementDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

export function BackupManagementDialog({ open, onOpenChange }: BackupManagementDialogProps) {
  const [backupLocation, setBackupLocation] = useState("./data/logs-backup");
  const [retentionCount, setRetentionCount] = useState("30");
  const [cronExpression, setCronExpression] = useState("0 2 * * *");
  const [backups, setBackups] = useState<Array<{id: number; filename: string; created_at: string; file_size: number}>>([]);

  const handleManualBackup = async () => {
    try {
      const res = await fetch("/api/logs/backup", { method: "POST" });
      if (res.ok) {
        alert("备份已开始");
      }
    } catch (e) {
      alert("备份失败");
    }
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-md">
        <DialogHeader>
          <DialogTitle>备份管理</DialogTitle>
        </DialogHeader>
        <div className="space-y-4">
          <div className="grid gap-2">
            <Label>备份位置</Label>
            <Input value={backupLocation} onChange={(e) => setBackupLocation(e.target.value)} />
          </div>
          <div className="grid gap-2">
            <Label>保留份数</Label>
            <Input type="number" value={retentionCount} onChange={(e) => setRetentionCount(e.target.value)} />
          </div>
          <div className="grid gap-2">
            <Label>定时备份 (Cron)</Label>
            <Input value={cronExpression} onChange={(e) => setCronExpression(e.target.value)} placeholder="0 2 * * *" />
          </div>
          
          <div className="border rounded p-3">
            <h4 className="font-medium mb-2">历史备份</h4>
            {backups.length === 0 ? (
              <p className="text-sm text-muted-foreground">暂无备份记录</p>
            ) : (
              <ul className="space-y-2">
                {backups.map((b) => (
                  <li key={b.id} className="flex justify-between text-sm">
                    <span>{b.created_at} - {b.filename}</span>
                    <div>
                      <Button variant="ghost" size="sm">下载</Button>
                      <Button variant="ghost" size="sm">删除</Button>
                    </div>
                  </li>
                ))}
              </ul>
            )}
          </div>
          
          <div className="flex gap-2">
            <Button onClick={handleManualBackup}>手动备份</Button>
            <Button variant="outline">删除过期</Button>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  );
}
