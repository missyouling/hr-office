"use client";

import { useCallback, useEffect, useMemo, useState } from "react";
import { toast } from "sonner";
import { Search, RefreshCw, ChevronLeft, ChevronRight } from "lucide-react";

import { getAuditLogs, getAuditStats } from "@/lib/api";
import type { AuditLog, AuditStats } from "@/lib/types";
import { formatDisplayDate } from "@/lib/utils";

import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";

interface AuditLogsProps {
  className?: string;
}

const ACTION_LABELS: Record<string, string> = {
  AUTH_LOGIN: "用户登录",
  AUTH_LOGOUT: "用户登出",
  AUTH_REGISTER: "用户注册",
  AUTH_PASSWORD_CHANGE: "修改密码",
  AUTH_PASSWORD_RESET: "重置密码",
  PERIOD_CREATE: "创建账期",
  PERIOD_DELETE: "删除账期",
  PERIOD_RESET: "重置账期",
  FILE_UPLOAD: "文件上传",
  FILE_DELETE: "文件删除",
  DATA_PROCESS: "数据处理",
  DATA_EXPORT: "数据导出",
  SYSTEM_START: "系统启动",
  SYSTEM_SHUTDOWN: "系统关闭",
};

const STATUS_LABELS: Record<string, { label: string; variant: "default" | "secondary" | "destructive" | "outline" }> = {
  SUCCESS: { label: "成功", variant: "default" },
  FAILURE: { label: "失败", variant: "destructive" },
  WARNING: { label: "警告", variant: "secondary" },
};

export function AuditLogs({ className }: AuditLogsProps) {
  const [logs, setLogs] = useState<AuditLog[]>([]);
  const [stats, setStats] = useState<AuditStats | null>(null);
  const [todayLogs, setTodayLogs] = useState<number>(0);
  const [isLoading, setIsLoading] = useState(true);
  const [searchTerm, setSearchTerm] = useState("");
  const [actionFilter, setActionFilter] = useState("all");
  const [statusFilter, setStatusFilter] = useState("all");
  const [userFilter, setUserFilter] = useState("all");

  // 分页相关状态
  const [currentPage, setCurrentPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [totalCount, setTotalCount] = useState(0);
  const [totalPages, setTotalPages] = useState(0);

  const loadData = useCallback(async () => {
    setIsLoading(true);
    try {
      const offset = (currentPage - 1) * pageSize;
      const [logsData, statsData, todayStatsData] = await Promise.all([
        getAuditLogs({
          limit: pageSize,
          offset: offset,
          action: actionFilter === "all" ? undefined : actionFilter,
          status: statusFilter === "all" ? undefined : statusFilter,
          user_id: userFilter === "all" ? undefined : parseInt(userFilter),
        }),
        getAuditStats(7), // 7天统计
        getAuditStats(1), // 1天统计用于今日操作
      ]);

      setLogs(logsData.logs || []);
      setTotalCount(logsData.total || 0);
      setTotalPages(Math.ceil((logsData.total || 0) / pageSize));
      setStats(statsData);
      setTodayLogs(todayStatsData?.stats?.total_events || 0);
    } catch (error) {
      console.error("Failed to load audit data:", error);
      toast.error("加载审计数据失败");
    } finally {
      setIsLoading(false);
    }
  }, [actionFilter, statusFilter, userFilter, currentPage, pageSize]);

  useEffect(() => {
    loadData();
  }, [loadData]);

  // 筛选条件变化时重置到第一页
  useEffect(() => {
    if (currentPage !== 1) {
      setCurrentPage(1);
    }
  }, [actionFilter, statusFilter, userFilter, currentPage]);

  // 注意：现在在服务端进行分页，所以前端搜索只在当前页面内进行
  const filteredLogs = logs.filter((log) => {
    if (!searchTerm) return true;
    const searchLower = searchTerm.toLowerCase();
    return (
      log.user_id?.toString().includes(searchLower) ||
      log.action.toLowerCase().includes(searchLower) ||
      log.resource_type?.toLowerCase().includes(searchLower) ||
      log.resource_id?.toLowerCase().includes(searchLower)
    );
  });

  // 分页控制函数
  const goToPage = (page: number) => {
    if (page >= 1 && page <= totalPages) {
      setCurrentPage(page);
    }
  };

  const goToPrevPage = () => {
    if (currentPage > 1) {
      setCurrentPage(currentPage - 1);
    }
  };

  const goToNextPage = () => {
    if (currentPage < totalPages) {
      setCurrentPage(currentPage + 1);
    }
  };

  // 生成页码数组
  const getPageNumbers = () => {
    const pages = [];
    const maxVisible = 5;
    let start = Math.max(1, currentPage - Math.floor(maxVisible / 2));
    const end = Math.min(totalPages, start + maxVisible - 1);

    if (end - start < maxVisible - 1) {
      start = Math.max(1, end - maxVisible + 1);
    }

    for (let i = start; i <= end; i++) {
      pages.push(i);
    }
    return pages;
  };

  const formatDateTime = (dateTime: string) => formatDisplayDate(dateTime, { includeTime: true });

  const uniqueUsers = useMemo(() => {
    const users = new Set<string>();
    logs.forEach((log) => {
      if (log.user_id) {
        users.add(log.user_id.toString());
      }
    });
    return Array.from(users);
  }, [logs]);

  const uniqueActions = useMemo(() => {
    const actions = new Set<string>();
    logs.forEach((log) => {
      actions.add(log.action);
    });
    return Array.from(actions);
  }, [logs]);

  return (
    <div className={`space-y-6 ${className}`}>
      {/* 统计概览 */}
      {stats && (
        <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
          <Card>
            <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
              <CardTitle className="text-sm font-medium">总操作数</CardTitle>
              <CardDescription className="text-xs text-muted-foreground">
                最近 {stats.days} 天
              </CardDescription>
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold">{stats.stats.total_events}</div>
            </CardContent>
          </Card>
          <Card>
            <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
              <CardTitle className="text-sm font-medium">今日操作</CardTitle>
              <CardDescription className="text-xs text-muted-foreground">
                过去 24 小时
              </CardDescription>
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold">{todayLogs}</div>
            </CardContent>
          </Card>
          <Card>
            <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
              <CardTitle className="text-sm font-medium">失败操作</CardTitle>
              <CardDescription className="text-xs text-muted-foreground">
                最近 {stats.days} 天
              </CardDescription>
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold text-destructive">
                {stats.stats.by_status?.FAILURE || stats.stats.by_status?.FAILED || 0}
              </div>
            </CardContent>
          </Card>
        </div>
      )}

      {/* 操作工具栏 */}
      <Card>
        <CardHeader>
          <CardTitle>审计日志</CardTitle>
          <CardDescription>
            查看系统操作的详细记录和审计信息
          </CardDescription>
        </CardHeader>
        <CardContent>
          <div className="flex flex-col space-y-4 sm:flex-row sm:space-y-0 sm:space-x-4">
            {/* 搜索框 */}
            <div className="flex-1">
              <div className="relative">
                <Search className="absolute left-3 top-3 h-4 w-4 text-muted-foreground" />
                <Input
                  placeholder="搜索用户ID、操作类型、资源或描述..."
                  value={searchTerm}
                  onChange={(e) => setSearchTerm(e.target.value)}
                  className="pl-10"
                />
              </div>
            </div>

            {/* 筛选器 */}
            <div className="flex space-x-2">
              <Select value={actionFilter} onValueChange={setActionFilter}>
                <SelectTrigger className="w-40">
                  <SelectValue placeholder="操作类型" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="all">所有操作</SelectItem>
                  {uniqueActions.map((action) => (
                    <SelectItem key={action} value={action}>
                      {ACTION_LABELS[action] || action}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>

              <Select value={statusFilter} onValueChange={setStatusFilter}>
                <SelectTrigger className="w-32">
                  <SelectValue placeholder="状态" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="all">所有状态</SelectItem>
                  <SelectItem value="SUCCESS">成功</SelectItem>
                  <SelectItem value="FAILURE">失败</SelectItem>
                  <SelectItem value="WARNING">警告</SelectItem>
                </SelectContent>
              </Select>

              <Select value={userFilter} onValueChange={setUserFilter}>
                <SelectTrigger className="w-32">
                  <SelectValue placeholder="用户" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="all">所有用户</SelectItem>
                  {uniqueUsers.map((userId) => (
                    <SelectItem key={userId} value={userId}>
                      用户 {userId}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>

              <Select value={pageSize.toString()} onValueChange={(value) => {
                setPageSize(parseInt(value));
                setCurrentPage(1);
              }}>
                <SelectTrigger className="w-24">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="10">10条</SelectItem>
                  <SelectItem value="20">20条</SelectItem>
                  <SelectItem value="50">50条</SelectItem>
                  <SelectItem value="100">100条</SelectItem>
                </SelectContent>
              </Select>

              <Button variant="outline" size="icon" onClick={loadData}>
                <RefreshCw className="h-4 w-4" />
              </Button>
            </div>
          </div>

          {/* 分页信息 */}
          <div className="mt-4 flex items-center justify-between">
            <div className="text-sm text-muted-foreground">
              显示第 {totalCount === 0 ? 0 : (currentPage - 1) * pageSize + 1} - {Math.min(currentPage * pageSize, totalCount)} 条，共 {totalCount} 条记录
            </div>

            {totalPages > 1 && (
              <div className="flex items-center space-x-2">
                <Button
                  variant="outline"
                  size="sm"
                  onClick={goToPrevPage}
                  disabled={currentPage === 1}
                >
                  <ChevronLeft className="h-4 w-4" />
                  上一页
                </Button>

                {getPageNumbers().map((page) => (
                  <Button
                    key={page}
                    variant={page === currentPage ? "default" : "outline"}
                    size="sm"
                    onClick={() => goToPage(page)}
                    className="min-w-[2rem]"
                  >
                    {page}
                  </Button>
                ))}

                <Button
                  variant="outline"
                  size="sm"
                  onClick={goToNextPage}
                  disabled={currentPage === totalPages}
                >
                  下一页
                  <ChevronRight className="h-4 w-4" />
                </Button>
              </div>
            )}
          </div>
        </CardContent>
      </Card>

      {/* 审计日志表格 */}
      <Card>
        <CardContent className="p-0">
          {isLoading ? (
            <div className="p-6 space-y-4">
              {Array.from({ length: 5 }).map((_, i) => (
                <Skeleton key={i} className="h-12 w-full" />
              ))}
            </div>
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>时间</TableHead>
                  <TableHead>用户</TableHead>
                  <TableHead>操作</TableHead>
                  <TableHead>资源</TableHead>
                  <TableHead>状态</TableHead>
                  <TableHead>描述</TableHead>
                  <TableHead>IP地址</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {filteredLogs.length === 0 ? (
                  <TableRow>
                    <TableCell colSpan={7} className="text-center py-8 text-muted-foreground">
                      没有找到匹配的审计日志
                    </TableCell>
                  </TableRow>
                ) : (
                  filteredLogs.map((log) => (
                    <TableRow key={log.id}>
                      <TableCell className="font-mono text-xs">
                        {formatDateTime(log.timestamp)}
                      </TableCell>
                      <TableCell>
                        {log.username || (log.user_id ? `用户 ${log.user_id}` : "系统")}
                      </TableCell>
                      <TableCell>
                        <Badge variant="outline">
                          {ACTION_LABELS[log.action] || log.action}
                        </Badge>
                      </TableCell>
                      <TableCell className="max-w-32 truncate">
                        {log.resource_type ? `${log.resource_type}${log.resource_id ? `:${log.resource_id}` : ''}` : "-"}
                      </TableCell>
                      <TableCell>
                        <Badge
                          variant={STATUS_LABELS[log.status]?.variant || "outline"}
                        >
                          {STATUS_LABELS[log.status]?.label || log.status}
                        </Badge>
                      </TableCell>
                      <TableCell className="max-w-48 truncate">
                        {log.details ? JSON.stringify(log.details) : "-"}
                      </TableCell>
                      <TableCell className="font-mono text-xs">
                        {log.ip_address || "-"}
                      </TableCell>
                    </TableRow>
                  ))
                )}
              </TableBody>
            </Table>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
