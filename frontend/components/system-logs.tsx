"use client";

import { useState } from "react";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import { Button } from "@/components/ui/button";

type LogType = "operation" | "login" | "system";

export function SystemLogs() {
  const [activeTab, setActiveTab] = useState<LogType>("operation");
  const [showBackupDialog, setShowBackupDialog] = useState(false);
  const [showAlertDialog, setShowAlertDialog] = useState(false);

  const tabs = [
    { value: "operation" as const, label: "用户操作" },
    { value: "login" as const, label: "登录记录" },
    { value: "system" as const, label: "系统日志" },
  ];

  const handleExport = () => {
    const params = new URLSearchParams({ type: activeTab });
    window.open("/api/logs/export?" + params);
  };

  return (
    <Card>
      <CardHeader>
        <div className="flex items-center justify-between">
          <div>
            <CardTitle>系统日志</CardTitle>
            <CardDescription>管理系统操作、登录记录和系统运行日志</CardDescription>
          </div>
          <div className="flex gap-2">
            <Button variant="outline" onClick={handleExport}>导出</Button>
            <Button variant="outline" onClick={() => setShowBackupDialog(true)}>备份管理</Button>
            <Button variant="outline" onClick={() => setShowAlertDialog(true)}>告警设置</Button>
          </div>
        </div>
      </CardHeader>
      <CardContent className="space-y-4">
        <div className="flex gap-2">
          {tabs.map((tab) => (
            <Button
              key={tab.value}
              variant={activeTab === tab.value ? "default" : "ghost"}
              onClick={() => setActiveTab(tab.value)}
            >
              {tab.label}
            </Button>
          ))}
        </div>

        <div className="flex gap-2 items-center flex-wrap">
          <input type="date" className="border rounded px-2 py-1" placeholder="开始日期" />
          <span>至</span>
          <input type="date" className="border rounded px-2 py-1" placeholder="结束日期" />
          <select className="border rounded px-2 py-1">
            <option value="">操作类型</option>
          </select>
          <select className="border rounded px-2 py-1">
            <option value="">操作人</option>
          </select>
          <select className="border rounded px-2 py-1">
            <option value="">状态</option>
            <option value="success">成功</option>
            <option value="failed">失败</option>
          </select>
          <Button variant="default">筛选</Button>
        </div>

        <div className="border rounded">
          <table className="w-full text-sm">
            <thead className="bg-muted">
              <tr>
                <th className="px-3 py-2 text-left">时间</th>
                <th className="px-3 py-2 text-left">操作人</th>
                <th className="px-3 py-2 text-left">操作类型</th>
                <th className="px-3 py-2 text-left">目标</th>
                <th className="px-3 py-2 text-left">状态</th>
                <th className="px-3 py-2 text-left">IP</th>
              </tr>
            </thead>
            <tbody>
              <tr>
                <td colSpan={6} className="px-3 py-8 text-center text-muted-foreground">
                  暂无数据
                </td>
              </tr>
            </tbody>
          </table>
        </div>

        <div className="flex justify-center">
          <Button variant="outline">加载更多</Button>
        </div>
      </CardContent>
    </Card>
  );
}
