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

const statusLabels: Record<string, string> = {
  SUCCESS: "成功",
  FAILED: "失败",
  ERROR: "错误",
  WARNING: "警告",
  INFO: "信息",
  DEBUG: "调试",
  UNKNOWN: "未知",
};

const levelLabels: Record<string, string> = {
  ERROR: "错误",
  WARN: "警告",
  WARNING: "警告",
  INFO: "信息",
  DEBUG: "调试",
  TRACE: "跟踪",
};

const getStatusLabel = (status: string): string => {
  return statusLabels[status] || status;
};

const getLevelLabel = (level: string): string => {
  return levelLabels[level] || level;
};

export function SystemLogs() {
  const [activeTab, setActiveTab] = useState<LogType>("operation");
  const [loading, setLoading] = useState(false);
  const [exporting, setExporting] = useState(false);
  const [logs, setLogs] = useState<SystemLog[]>([]);
  const [total, setTotal] = useState(0);
  const [currentPage, setCurrentPage] = useState(1);
  const pageSize = 20;

  const [showBackupDialog, setShowBackupDialog] = useState(false);
  const [showAlertDialog, setShowAlertDialog] = useState(false);

  const [startDate, setStartDate] = useState("");
  const [endDate, setEndDate] = useState("");
  const [action, setAction] = useState("all");
  const [status, setStatus] = useState("all");
  const [knownStatuses, setKnownStatuses] = useState<string[]>([]);
  const [knownActions, setKnownActions] = useState<string[]>([]);
  const [knownLevels, setKnownLevels] = useState<string[]>([]);
  const [level, setLevel] = useState("all");
  const [search, setSearch] = useState("");

  const loadLogs = useCallback(async (pageNum: number = 1) => {
    setLoading(true);
    try {
      const params: SystemLogParams = {
        log_type: activeTab,
        page: pageNum,
        size: pageSize,
      };

      if (startDate) params.start_date = startDate;
      if (endDate) params.end_date = endDate;
      if (action !== "all") params.action = action;
      if (status !== "all") params.status = status;
      if (level !== "all") params.level = level;
      if (search) params.search = search;

      const response = await fetchSystemLogs(params);
      const data = response.data || [];
      
      if (pageNum === 1) {
        setLogs(data);
        setCurrentPage(1);
      } else {
        setLogs((prev) => [...prev, ...data]);
        setCurrentPage(pageNum);
      }
      setTotal(response.total);
    } catch (error) {
      console.error("加载日志失败:", error);
      toast.error("加载日志失败");
    } finally {
      setLoading(false);
    }
  }, [activeTab, startDate, endDate, action, status, level, search]);

  useEffect(() => {
    setStatus("all");
    setAction("all");
    setLevel("all");
    setKnownStatuses([]);
    setKnownActions([]);
    setKnownLevels([]);
    setStartDate("");
    setEndDate("");
    setSearch("");
  }, [activeTab]);

  useEffect(() => {
    loadLogs(1);
  }, [activeTab, action, status, level, startDate, endDate, search, loadLogs]);

  useEffect(() => {
    if (logs && logs.length > 0) {
      const statuses = new Set<string>();
      const actions = new Set<string>();
      const levels = new Set<string>();
      logs.forEach(log => {
        if (log.status) statuses.add(log.status);
        if (log.action) actions.add(log.action);
        if ((log.level as string)) levels.add(log.level as string);
      });
      setKnownStatuses(Array.from(statuses));
      setKnownActions(Array.from(actions));
      setKnownLevels(Array.from(levels));
    }
  }, [logs]);

  const handleFilter = () => {
    loadLogs(1);
  };

  const handleTableScroll = (e: React.UIEvent<HTMLDivElement>) => {
    const target = e.currentTarget;
    const isNearBottom = target.scrollHeight - target.scrollTop - target.clientHeight < 100;
    
    if (isNearBottom && !loading && logs.length < total && logs.length > 0) {
      loadLogs(currentPage + 1);
    }
  };

  const handleExport = async () => {
    if (!logs || logs.length === 0) { toast.warning("没有日志数据", { description: "当前暂无可导出的系统日志数据" }); return; }
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
    const displayStatus = getStatusLabel(status);
    const isSuccess = status.toUpperCase() === "SUCCESS" || status === "200";
    const isWarning = status.toUpperCase().includes("WARN");
    const isError = status.toUpperCase() === "FAILED" || status.toUpperCase() === "ERROR" || status.startsWith("4") || status.startsWith("5");
    
    let variant: "default" | "destructive" | "outline" | "secondary" = "outline";
    if (isSuccess) variant = "default";
    else if (isError) variant = "destructive";
    else if (isWarning) variant = "secondary";
    
    return (
      <Badge variant={variant}>
        {displayStatus}
      </Badge>
    );
  };

  const renderTableHeaders = () => {
    if (activeTab === "system") {
      return (
        <TableRow>
          <TableHead className="text-center">时间</TableHead>
          <TableHead className="text-center">级别</TableHead>
          <TableHead className="text-center">来源</TableHead>
          <TableHead className="text-center">消息</TableHead>
          <TableHead className="text-center">Trace ID</TableHead>
        </TableRow>
      );
    }
    if (activeTab === "login") {
      return (
        <TableRow>
          <TableHead className="text-center">时间</TableHead>
          <TableHead className="text-center">操作人</TableHead>
          <TableHead className="text-center">操作</TableHead>
          <TableHead className="text-center">IP</TableHead>
          <TableHead className="text-center">状态</TableHead>
          <TableHead className="text-center">状态码</TableHead>
        </TableRow>
      );
    }
    return (
      <TableRow>
        <TableHead className="text-center">时间</TableHead>
        <TableHead className="text-center">操作人</TableHead>
        <TableHead className="text-center">操作类型</TableHead>
        <TableHead className="text-center">目标</TableHead>
        <TableHead className="text-center">状态</TableHead>
        <TableHead className="text-center">IP</TableHead>
      </TableRow>
    );
  };

  const renderTableRows = () => {
    if (!logs || logs.length === 0) {
      if (!loading) {
        return (
          <TableRow>
            <TableCell colSpan={activeTab === "system" ? 5 : 6} className="h-24 text-center text-muted-foreground">
              暂无数据
            </TableCell>
          </TableRow>
        );
      }
      return null;
    }

    return logs.map((log, index) => {
      const dateStr = log.created_at ? format(new Date(log.created_at), "yyyy-MM-dd HH:mm:ss") : "-";
      
      if (activeTab === "system") {
        const levelDisplay = getLevelLabel(log.level || '');
        const isError = log.level === "ERROR";
        const isWarn = log.level === "WARN" || log.level === "WARNING";
        return (
          <TableRow key={`${log.id}-${index}`}>
            <TableCell className="whitespace-nowrap text-center">{dateStr}</TableCell>
            <TableCell className="text-center">
              <Badge variant={isError ? "destructive" : isWarn ? "secondary" : "outline"}>
                {levelDisplay}
              </Badge>
            </TableCell>
            <TableCell className="truncate text-center">{log.source}</TableCell>
            <TableCell className="truncate text-center" title={log.message}>{log.message}</TableCell>
            <TableCell className="font-mono text-xs truncate text-center">{log.trace_id}</TableCell>
          </TableRow>
        );
      }

       if (activeTab === "login") {
         return (
           <TableRow key={`${log.id}-${index}`}>
             <TableCell className="whitespace-nowrap text-center">{dateStr}</TableCell>
             <TableCell className="truncate text-center">{log.user?.username || log.user_id || "系统"}</TableCell>
             <TableCell className="truncate text-center">{log.action}</TableCell>
             <TableCell className="truncate text-center">{log.ip_address || "-"}</TableCell>
             <TableCell className="text-center">{renderStatusBadge(log.status)}</TableCell>
             <TableCell className="text-center">{log.status_code}</TableCell>
           </TableRow>
         );
       }

       return (
         <TableRow key={`${log.id}-${index}`}>
           <TableCell className="whitespace-nowrap text-center">{dateStr}</TableCell>
           <TableCell className="truncate text-center">{log.user?.username || log.user_id || "系统"}</TableCell>
           <TableCell className="truncate text-center">{log.action}</TableCell>
           <TableCell className="truncate text-center">{log.resource}</TableCell>
           <TableCell className="text-center">{renderStatusBadge(log.status)}</TableCell>
           <TableCell className="truncate text-center">{log.ip_address || "-"}</TableCell>
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
            <Button variant="outline" onClick={() => setShowBackupDialog(true)}>备份设置</Button>
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
                <SelectItem value="all">所有状态</SelectItem>{knownStatuses.map(s => <SelectItem key={s} value={s}>{getStatusLabel(s)}</SelectItem>)}
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
                  <SelectItem value="all">所有级别</SelectItem>{knownLevels.map(l => <SelectItem key={l} value={l}>{getLevelLabel(l)}</SelectItem>)}
                </SelectContent>
              </Select>
            </div>
          ) : (
            <div className="space-y-1">
              <label className="text-xs text-muted-foreground">操作类型</label>
              <Select value={action} onValueChange={setAction}>
                <SelectTrigger className="w-40">
                  <SelectValue placeholder="所有操作" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="all">所有操作</SelectItem>
                  {knownActions.map(a => <SelectItem key={a} value={a}>{a}</SelectItem>)}
                </SelectContent>
              </Select>
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

        <div className="border rounded-md overflow-hidden h-[600px] flex flex-col">
          <div className="overflow-y-auto flex-1" id="scroll-container" onScroll={handleTableScroll}>
            <Table className="w-full">
              <colgroup>
                {activeTab === "system" ? (
                  <>
                    <col style={{ width: "180px" }} />
                    <col style={{ width: "100px" }} />
                    <col style={{ width: "120px" }} />
                    <col style={{ flex: 1, minWidth: "200px" }} />
                    <col style={{ width: "150px" }} />
                  </>
                ) : (
                  <>
                    <col style={{ width: "180px" }} />
                    <col style={{ width: "120px" }} />
                    <col style={{ width: "120px" }} />
                    <col style={{ width: "120px" }} />
                    <col style={{ width: "80px" }} />
                    <col style={{ width: "80px" }} />
                  </>
                )}
              </colgroup>
              <TableHeader className="bg-muted/50 sticky top-0 z-10">
                {renderTableHeaders()}
              </TableHeader>
              <TableBody>
                {renderTableRows()}
                {loading && currentPage === 1 && (
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
        </div>

        <div className="flex flex-col items-center gap-2 pt-2">
          <div className="text-sm text-muted-foreground">
            共 {total} 条记录 {logs && logs.length > 0 && `(已加载 ${logs.length})`}
          </div>
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
