"use client";

import { useCallback, useEffect, useState } from "react";
import { toast } from "sonner";
import {
  Server,
  Database,
  HardDrive,
  Cpu,
  Activity,
  RefreshCw,
  CheckCircle,
  AlertTriangle,
  XCircle
} from "lucide-react";

import {
  getSystemMetrics,
  getDatabaseStatus,
  getSystemInfo,
  runMaintenance
} from "@/lib/api";
import type { SystemMetrics, DatabaseStatus, SystemInfo } from "@/lib/types";

import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Progress } from "@/components/ui/progress";
import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";

interface SystemMonitoringProps {
  className?: string;
}

export function SystemMonitoring({ className }: SystemMonitoringProps) {
  const [metrics, setMetrics] = useState<SystemMetrics | null>(null);
  const [dbStatus, setDbStatus] = useState<DatabaseStatus | null>(null);
  const [systemInfo, setSystemInfo] = useState<SystemInfo | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [isMaintenanceRunning, setIsMaintenanceRunning] = useState(false);
  const [showMaintenanceDialog, setShowMaintenanceDialog] = useState(false);

  const loadData = useCallback(async () => {
    setIsLoading(true);
    try {
      const [metricsData, dbData, infoData] = await Promise.all([
        getSystemMetrics(),
        getDatabaseStatus(),
        getSystemInfo(),
      ]);

      setMetrics(metricsData);
      setDbStatus(dbData);
      setSystemInfo(infoData);
    } catch (error) {
      console.error("Failed to load system data:", error);
      toast.error("加载系统监控数据失败");
    } finally {
      setIsLoading(false);
    }
  }, []);

  useEffect(() => {
    loadData();
    // 每30秒自动刷新一次
    const interval = setInterval(loadData, 30000);
    return () => clearInterval(interval);
  }, [loadData]);

  const handleMaintenance = async () => {
    setIsMaintenanceRunning(true);
    try {
      await runMaintenance();
      toast.success("维护任务执行成功");
      setShowMaintenanceDialog(false);
      // 重新加载数据
      await loadData();
    } catch (error) {
      console.error("Maintenance failed:", error);
      toast.error("维护任务执行失败");
    } finally {
      setIsMaintenanceRunning(false);
    }
  };

  const formatUptime = (uptime: string) => {
    const match = uptime.match(/([0-9.]+)([a-z]+)/);
    if (!match) return uptime;

    const value = parseFloat(match[1]);
    const unit = match[2];

    if (unit.includes('h')) {
      const hours = Math.floor(value);
      const minutes = Math.floor((value - hours) * 60);
      return `${hours}小时${minutes}分钟`;
    } else if (unit.includes('m')) {
      return `${Math.floor(value)}分钟`;
    } else if (unit.includes('s')) {
      return `${Math.floor(value)}秒`;
    }
    return uptime;
  };

  const formatBytes = (bytes: number) => {
    if (bytes === 0) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
  };

  const getStatusIcon = (status: string) => {
    switch (status.toLowerCase()) {
      case 'healthy':
      case 'connected':
        return <CheckCircle className="h-4 w-4 text-green-500" />;
      case 'warning':
        return <AlertTriangle className="h-4 w-4 text-yellow-500" />;
      case 'error':
      case 'disconnected':
        return <XCircle className="h-4 w-4 text-red-500" />;
      default:
        return <Activity className="h-4 w-4 text-blue-500" />;
    }
  };

  const getStatusBadge = (status: string | undefined) => {
    if (!status) {
      return <Badge variant="outline">未知</Badge>;
    }

    switch (status.toLowerCase()) {
      case 'healthy':
      case 'connected':
        return <Badge className="bg-green-100 text-green-800">正常</Badge>;
      case 'warning':
        return <Badge className="bg-yellow-100 text-yellow-800">警告</Badge>;
      case 'error':
      case 'disconnected':
        return <Badge className="bg-red-100 text-red-800">错误</Badge>;
      default:
        return <Badge variant="outline">{status}</Badge>;
    }
  };

  return (
    <div className={`space-y-6 ${className}`}>
      {/* 系统概览 */}
      <div className="flex justify-between items-center">
        <div>
          <h2 className="text-2xl font-bold">系统监控</h2>
          <p className="text-muted-foreground">
            实时监控系统状态和性能指标
          </p>
        </div>
        <div className="flex space-x-2">
          <Dialog open={showMaintenanceDialog} onOpenChange={setShowMaintenanceDialog}>
            <DialogTrigger asChild>
              <Button variant="outline">
                <Server className="h-4 w-4 mr-2" />
                维护任务
              </Button>
            </DialogTrigger>
            <DialogContent>
              <DialogHeader>
                <DialogTitle>执行系统维护</DialogTitle>
                <DialogDescription>
                  清理过期的令牌和临时数据，优化系统性能。
                </DialogDescription>
              </DialogHeader>
              <DialogFooter>
                <Button
                  variant="outline"
                  onClick={() => setShowMaintenanceDialog(false)}
                >
                  取消
                </Button>
                <Button
                  onClick={handleMaintenance}
                  disabled={isMaintenanceRunning}
                >
                  {isMaintenanceRunning ? "执行中..." : "执行维护"}
                </Button>
              </DialogFooter>
            </DialogContent>
          </Dialog>

          <Button variant="outline" size="icon" onClick={loadData}>
            <RefreshCw className="h-4 w-4" />
          </Button>
        </div>
      </div>

      {/* 系统信息卡片 */}
      {isLoading ? (
        <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
          {Array.from({ length: 3 }).map((_, i) => (
            <Card key={i}>
              <CardHeader>
                <Skeleton className="h-4 w-24" />
              </CardHeader>
              <CardContent>
                <Skeleton className="h-8 w-16" />
              </CardContent>
            </Card>
          ))}
        </div>
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
          {/* 系统状态 */}
          <Card>
            <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
              <CardTitle className="text-sm font-medium">系统状态</CardTitle>
              {getStatusIcon(systemInfo?.health_status || 'unknown')}
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold">
                {getStatusBadge(systemInfo?.health_status || 'unknown')}
              </div>
              <p className="text-xs text-muted-foreground">
                运行时间: {systemInfo?.uptime ? formatUptime(systemInfo.uptime) : 'N/A'}
              </p>
            </CardContent>
          </Card>

          {/* 数据库状态 */}
          <Card>
            <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
              <CardTitle className="text-sm font-medium">数据库</CardTitle>
              <Database className="h-4 w-4 text-muted-foreground" />
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold">
                {getStatusBadge(dbStatus?.status || 'unknown')}
              </div>
              <p className="text-xs text-muted-foreground">
                类型: {dbStatus?.database_type || 'N/A'}
              </p>
            </CardContent>
          </Card>

          {/* 活跃连接 */}
          <Card>
            <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
              <CardTitle className="text-sm font-medium">活跃连接</CardTitle>
              <Activity className="h-4 w-4 text-muted-foreground" />
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold">
                {dbStatus?.active_connections || 0}
              </div>
              <p className="text-xs text-muted-foreground">
                最大连接: {dbStatus?.max_connections || 'N/A'}
              </p>
            </CardContent>
          </Card>
        </div>
      )}

      {/* 性能指标 */}
      {metrics && (
        <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
          {/* 内存使用情况 */}
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center">
                <HardDrive className="h-4 w-4 mr-2" />
                内存使用情况
              </CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <div>
                <div className="flex justify-between mb-2">
                  <span className="text-sm">堆内存</span>
                  <span className="text-sm">
                    {formatBytes(metrics.memory_heap_inuse)} / {formatBytes(metrics.memory_heap_sys)}
                  </span>
                </div>
                <Progress
                  value={(metrics.memory_heap_inuse / metrics.memory_heap_sys) * 100}
                  className="h-2"
                />
              </div>
              <div>
                <div className="flex justify-between mb-2">
                  <span className="text-sm">系统内存</span>
                  <span className="text-sm">{formatBytes(metrics.memory_sys)}</span>
                </div>
              </div>
              <div className="text-xs text-muted-foreground">
                GC次数: {metrics.memory_gc_count}
              </div>
            </CardContent>
          </Card>

          {/* Goroutines */}
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center">
                <Cpu className="h-4 w-4 mr-2" />
                并发处理
              </CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <div>
                <div className="flex justify-between mb-2">
                  <span className="text-sm">活跃Goroutines</span>
                  <span className="text-2xl font-bold">{metrics.goroutines}</span>
                </div>
              </div>
              <div className="text-xs text-muted-foreground">
                用于处理并发请求和后台任务
              </div>
            </CardContent>
          </Card>
        </div>
      )}

      {/* 数据库详细信息 */}
      {dbStatus && (
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center">
              <Database className="h-4 w-4 mr-2" />
              数据库详细信息
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
              <div>
                <div className="text-sm text-muted-foreground">数据库类型</div>
                <div className="font-medium">{dbStatus.database_type}</div>
              </div>
              <div>
                <div className="text-sm text-muted-foreground">数据库版本</div>
                <div className="font-medium">{dbStatus.database_version || 'N/A'}</div>
              </div>
              <div>
                <div className="text-sm text-muted-foreground">活跃连接数</div>
                <div className="font-medium">{dbStatus.active_connections}</div>
              </div>
              <div>
                <div className="text-sm text-muted-foreground">最大连接数</div>
                <div className="font-medium">{dbStatus.max_connections || 'N/A'}</div>
              </div>
              {dbStatus.total_tables && (
                <div>
                  <div className="text-sm text-muted-foreground">数据表总数</div>
                  <div className="font-medium">{dbStatus.total_tables}</div>
                </div>
              )}
              {dbStatus.total_size && (
                <div>
                  <div className="text-sm text-muted-foreground">数据库大小</div>
                  <div className="font-medium">{dbStatus.total_size}</div>
                </div>
              )}
            </div>
          </CardContent>
        </Card>
      )}

      {/* 系统信息 */}
      {systemInfo && (
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center">
              <Server className="h-4 w-4 mr-2" />
              系统信息
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
              <div>
                <div className="text-sm text-muted-foreground">系统版本</div>
                <div className="font-medium">{systemInfo?.version || 'N/A'}</div>
              </div>
              <div>
                <div className="text-sm text-muted-foreground">Go版本</div>
                <div className="font-medium">{systemInfo?.go_version || 'N/A'}</div>
              </div>
              <div>
                <div className="text-sm text-muted-foreground">运行环境</div>
                <div className="font-medium">{systemInfo?.environment || 'N/A'}</div>
              </div>
              <div>
                <div className="text-sm text-muted-foreground">启动时间</div>
                <div className="font-medium">
                  {systemInfo?.start_time ? new Date(systemInfo.start_time).toLocaleString("zh-CN") : 'N/A'}
                </div>
              </div>
              <div>
                <div className="text-sm text-muted-foreground">运行时间</div>
                <div className="font-medium">{systemInfo?.uptime ? formatUptime(systemInfo.uptime) : 'N/A'}</div>
              </div>
              <div>
                <div className="text-sm text-muted-foreground">健康状态</div>
                <div className="font-medium">
                  {getStatusBadge(systemInfo?.health_status)}
                </div>
              </div>
            </div>
          </CardContent>
        </Card>
      )}
    </div>
  );
}
