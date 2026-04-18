"use client";

import { useState, useEffect, useCallback } from "react";
import { format } from "date-fns";
import { toast } from "sonner";
import { 
  fetchSystemLogs, 
  exportSystemLogs, 
  type SystemLog, 
  type SystemLogParams 
} from "@/lib/api";
import { 
  Card, 
  CardContent, 
  CardHeader, 
  CardTitle, 
  CardDescription 
} from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { 
  Select, 
  SelectContent, 
  SelectItem, 
  SelectTrigger, 
  SelectValue 
} from "@/components/ui/select";
import { Badge } from "@/components/ui/badge";
import { 
  Table, 
  TableBody, 
  TableCell, 
  TableHead, 
  TableHeader, 
  TableRow 
} from "@/components/ui/table";
import { Skeleton } from "@/components/ui/skeleton";
import { BackupManagementDialog } from "./backup-management-dialog";
import { AlertSettingsDialog } from "./alert-settings-dialog";

type LogType = "operation" | "login" | "system";

export function SystemLogs() {
  const [activeTab, setActiveTab] = useState<LogType>("operation");
  const [loading, setLoading] = useState(false);
  const [exporting, setExporting] = useState(false);
  const [logs, setLogs] = useState<SystemLog[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const pageSize = 20;

  const [showBackupDialog, setShowBackupDialog] = useState(false);
  const [showAlertDialog, setShowAlertDialog] = useState(false);

  const [startDate, setStartDate] = useState("");
  const [endDate, setEndDate] = useState("");
  const [action, setAction] = useState("all");
  const [status, setStatus] = useState("all");
  const [level, setLevel] = useState("all");
  const [search, setSearch] = useState("");

  const loadLogs = useCallback(async (reset: boolean = false) => {
    setLoading(true);
    const currentPage = reset ? 1 : page;
    try {
      const params: SystemLogParams = {
        log_type: activeTab,
        page: currentPage,
        size: pageSize,
      };

      if (startDate) params.start_date = startDate;
      if (endDate) params.end_date = endDate;
      if (action !== "all") params.action = action;
      if (status !== "all") params.status = status;
      if (level !== "all") params.level = level;
      if (search) params.search = search;

      const response = await fetchSystemLogs(params);
      if (reset) {
        setLogs(response.data);
      } else {
        setLogs((prev) => [...prev, ...response.data]);
      }
      setTotal(response.total);
      if (reset) setPage(1);
    } catch (error) {
      console.error("加载日志失败:", error);
      toast.error("加载日志失败");
    } finally {
      setLoading(false);
    }
  }, [activeTab, page, startDate, endDate, action, status, level, search]);

  useEffect(() => {
    loadLogs(true);
  }, [activeTab]);

  const handleFilter = () => {
    loadLogs(true);
  };

  const handleLoadMore = () => {
    setPage((prev) => prev + 1);
  };

  useEffect(() => {
    if (page > 1) {
      loadLogs(false);
    }
  }, [page]);

  const handleExport = async () => {
    setExporting(true);
    try {
      const params: SystemLogParams = {
        log_type: activeTab,
      };
      if (startDate) params.start_date = startDate;
      if (endDate) params.end_date = endDate;
      if (action !== "all") params.action = action;
      if (status !== "all") params.status = status;
      if (level !== "all") params.level = level;
      if (search) params.search = search;

      const blob = await exportSystemLogs(params);
      const url = window.URL.createObjectURL(blob);
      const a = document.createElement("a");
      a.href = url;
      a.download = `system_logs_${activeTab}_${format(new Date(), "yyyyMMdd")}.xlsx`;
      document.body.appendChild(a);
      a.click();
      window.URL.revokeObjectURL(url);
      document.body.removeChild(a);
      toast.success("导出成功");
    } catch (error) {
      console.error("导出失败:", error);
      toast.error("导出失败");
    } finally {
      setExporting(false);
    }
  };

  const renderStatusBadge = (status: string) => {
    const isSuccess = status.toLowerCase() === "success" || status.toLowerCase() === "成功" || status === "200";
    return (
      <Badge variant={isSuccess ? "default" : "destructive"} className={isSuccess ? "bg-green-500 hover:bg-green-600" : ""}>
        {status === "success" ? "成功" : status === "failed" ? "失败" : status}
      </Badge>
    );
  };

  const renderTableHeaders = () => {
    if (activeTab === "system") {
      return (
        <TableRow>
          <TableHead>时间</TableHead>
          <TableHead>级别</TableHead>
          <TableHead>来源</TableHead>
          <TableHead>消息</TableHead>
          <TableHead>Trace ID</TableHead>
        </TableRow>
      );
    }
    if (activeTab === "login") {
      return (
        <TableRow>
          <TableHead>时间</TableHead>
          <TableHead>操作人</TableHead>
          <TableHead>操作</TableHead>
          <TableHead>IP</TableHead>
          <TableHead>状态</TableHead>
          <TableHead>状态码</TableHead>
        </TableRow>
      );
    }
    return (
      <TableRow>
        <TableHead>时间</TableHead>
        <TableHead>操作人</TableHead>
        <TableHead>操作类型</TableHead>
        <TableHead>目标</TableHead>
        <TableHead>状态</TableHead>
        <TableHead>IP</TableHead>
      </TableRow>
    );
  };

  const renderTableRows = () => {
    if (logs.length === 0 && !loading) {
      return (
        <TableRow>
          <TableCell colSpan={activeTab === "system" ? 5 : 6} className="h-24 text-center text-muted-foreground">
            暂无数据
          </TableCell>
        </TableRow>
      );
    }

    return logs.map((log) => {
      const dateStr = log.created_at ? format(new Date(log.created_at), "yyyy-MM-dd HH:mm:ss") : "-";
      
      if (activeTab === "system") {
        return (
          <TableRow key={log.id}>
            <TableCell className="whitespace-nowrap">{dateStr}</TableCell>
            <TableCell>
              <Badge variant={log.level === "ERROR" ? "destructive" : "outline"}>
                {log.level}
              </Badge>
            </TableCell>
            <TableCell>{log.source}</TableCell>
            <TableCell className="max-w-md truncate" title={log.message}>{log.message}</TableCell>
            <TableCell className="font-mono text-xs">{log.trace_id}</TableCell>
          </TableRow>
        );
      }

      if (activeTab === "login") {
        return (
          <TableRow key={log.id}>
            <TableCell className="whitespace-nowrap">{dateStr}</TableCell>
            <TableCell>{log.user_id || "系统"}</TableCell>
            <TableCell>{log.action}</TableCell>
            <TableCell>{log.ip_address}</TableCell>
            <TableCell>{renderStatusBadge(log.status)}</TableCell>
            <TableCell>{log.status_code}</TableCell>
          </TableRow>
        );
      }

      return (
        <TableRow key={log.id}>
          <TableCell className="whitespace-nowrap">{dateStr}</TableCell>
          <TableCell>{log.user_id}</TableCell>
          <TableCell>{log.action}</TableCell>
          <TableCell>{log.resource}</TableCell>
          <TableCell>{renderStatusBadge(log.status)}</TableCell>
          <TableCell>{log.ip_address}</TableCell>
        </TableRow>
      );
    });
  };

  const tabs = [
    { value: "operation" as const, label: "用户操作" },
    { value: "login" as const, label: "登录记录" },
    { value: "system" as const, label: "系统日志" },
  ];

  return (
    <Card>
      <CardHeader>
        <div className="flex items-center justify-between">
          <div>
            <CardTitle>系统日志</CardTitle>
            <CardDescription>管理系统操作、登录记录和系统运行日志</CardDescription>
          </div>
          <div className="flex gap-2">
            <Button variant="outline" onClick={handleExport} disabled={exporting}>
              {exporting ? "导出中..." : "导出"}
            </Button>
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

        <div className="flex flex-wrap gap-2 items-end">
          <div className="space-y-1">
            <label className="text-xs text-muted-foreground">开始日期</label>
            <Input 
              type="date" 
              className="w-40" 
              value={startDate} 
              onChange={(e) => setStartDate(e.target.value)} 
            />
          </div>
          <div className="flex items-center h-10 px-1">至</div>
          <div className="space-y-1">
            <label className="text-xs text-muted-foreground">结束日期</label>
            <Input 
              type="date" 
              className="w-40" 
              value={endDate} 
              onChange={(e) => setEndDate(e.target.value)} 
            />
          </div>

          <div className="space-y-1">
            <label className="text-xs text-muted-foreground">状态</label>
            <Select value={status} onValueChange={setStatus}>
              <SelectTrigger className="w-32">
                <SelectValue placeholder="所有状态" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">所有状态</SelectItem>
                <SelectItem value="success">成功</SelectItem>
                <SelectItem value="failed">失败</SelectItem>
              </SelectContent>
            </Select>
          </div>

          {activeTab === "system" ? (
            <div className="space-y-1">
              <label className="text-xs text-muted-foreground">日志级别</label>
              <Select value={level} onValueChange={setLevel}>
                <SelectTrigger className="w-32">
                  <SelectValue placeholder="级别" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="all">所有级别</SelectItem>
                  <SelectItem value="INFO">INFO</SelectItem>
                  <SelectItem value="WARN">WARN</SelectItem>
                  <SelectItem value="ERROR">ERROR</SelectItem>
                  <SelectItem value="DEBUG">DEBUG</SelectItem>
                </SelectContent>
              </Select>
            </div>
          ) : (
            <div className="space-y-1">
              <label className="text-xs text-muted-foreground">操作类型</label>
              <Input 
                placeholder="搜索操作..." 
                className="w-40" 
                value={action === "all" ? "" : action} 
                onChange={(e) => setAction(e.target.value || "all")} 
              />
            </div>
          )}

          <div className="space-y-1">
            <label className="text-xs text-muted-foreground">关键词</label>
            <Input 
              placeholder="搜索..." 
              className="w-48" 
              value={search} 
              onChange={(e) => setSearch(e.target.value)} 
            />
          </div>

          <Button onClick={handleFilter}>筛选</Button>
        </div>

        <div className="border rounded-md">
          <Table>
            <TableHeader className="bg-muted/50">
              {renderTableHeaders()}
            </TableHeader>
            <TableBody>
              {renderTableRows()}
              {loading && page === 1 && (
                Array.from({ length: 5 }).map((_, i) => (
                  <TableRow key={i}>
                    {Array.from({ length: activeTab === "system" ? 5 : 6 }).map((_, j) => (
                      <TableCell key={j}>
                        <Skeleton className="h-4 w-full" />
                      </TableCell>
                    ))}
                  </TableRow>
                ))
              )}
            </TableBody>
          </Table>
        </div>

        <div className="flex flex-col items-center gap-2 pt-2">
          <div className="text-sm text-muted-foreground">
            共 {total} 条记录 {logs.length > 0 && `(已加载 ${logs.length})`}
          </div>
          {logs.length < total && (
            <Button 
              variant="outline" 
              onClick={handleLoadMore} 
              disabled={loading}
              className="min-w-[120px]"
            >
              {loading ? "加载中..." : "加载更多"}
            </Button>
          )}
        </div>
      </CardContent>

      <BackupManagementDialog 
        open={showBackupDialog} 
        onOpenChange={setShowBackupDialog} 
      />
      <AlertSettingsDialog 
        open={showAlertDialog} 
        onOpenChange={setShowAlertDialog} 
      />
    </Card>
  );
}
