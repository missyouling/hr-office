"use client";

import { useState, useEffect, useCallback } from "react";
import { Download, Loader2 } from "lucide-react";
import { toast } from "sonner";

import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Skeleton } from "@/components/ui/skeleton";
import { Badge } from "@/components/ui/badge";
import { ScrollArea } from "@/components/ui/scroll-area";
import { PermissionGate } from "@/components/permission-gate";

import { knowledgeApi, type KnowledgeBase, type KBIngestResponse } from "@/lib/api-knowledge";
import { getSourceModuleLabel } from "../utils";

interface IngestRecord {
  kbId: number;
  kbName: string;
  sourceModule: string;
  result?: KBIngestResponse;
  loading: boolean;
}

/** 空状态（未选择知识库时） */
function NoKBSelected() {
  return (
    <div className="flex h-64 flex-col items-center justify-center text-muted-foreground">
      <Download className="mb-3 h-12 w-12 opacity-40" />
      <p className="text-base">请先选择知识库</p>
      <p className="mt-1 text-sm">从上方的下拉框中选择要入库的知识库</p>
    </div>
  );
}

/** 空状态（已选择知识库但无入库记录） */
function NoIngestRecords() {
  return (
    <div className="flex h-32 flex-col items-center justify-center text-muted-foreground">
      <p className="text-sm">暂无入库记录</p>
      <p className="mt-1 text-xs">点击「执行入库」触发数据入库</p>
    </div>
  );
}

export default function IngestTab() {
  const [kbs, setKbs] = useState<KnowledgeBase[]>([]);
  const [kbsLoading, setKbsLoading] = useState(true);
  const [selectedKBId, setSelectedKBId] = useState<string>("");
  const [since, setSince] = useState("");
  const [records, setRecords] = useState<IngestRecord[]>([]);
  const [todayCount, setTodayCount] = useState(0);
  const [totalCount, setTotalCount] = useState(0);

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

  useEffect(() => {
    fetchKBs();
  }, [fetchKBs]);

  const selectedKB = kbs.find((k) => String(k.id) === selectedKBId);

  const handleIngest = async () => {
    const id = Number(selectedKBId);
    if (!id) {
      toast.error("请先选择知识库");
      return;
    }

    const record: IngestRecord = {
      kbId: id,
      kbName: selectedKB?.name ?? `KB #${id}`,
      sourceModule: selectedKB?.source_module ?? "",
      loading: true,
    };

    setRecords((prev) => [record, ...prev]);
    try {
      const res = await knowledgeApi.ingest(id, since ? { since } : undefined);
      setRecords((prev) =>
        prev.map((r) => (r.kbId === id && r.loading ? { ...r, result: res, loading: false } : r))
      );
      toast.success(`入库完成：${res.ingested} 条`, { description: `扫描 ${res.scanned}，跳过 ${res.skipped}` });
      setTodayCount((c) => c + (res.ingested ?? 0));
      setTotalCount((c) => c + (res.ingested ?? 0));
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : "入库失败";
      toast.error("入库失败", { description: msg });
      setRecords((prev) =>
        prev.map((r) => (r.kbId === id && r.loading ? { ...r, loading: false } : r))
      );
    }
  };

  return (
    <div className="space-y-4">
      {/* 统计行 */}
      <div className="grid grid-cols-2 gap-4 lg:grid-cols-2">
        <Card>
          <CardContent className="p-4">
            <p className="text-xs text-muted-foreground">今日入库数</p>
            <p className="mt-1 text-2xl font-bold font-mono">{todayCount}</p>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="p-4">
            <p className="text-xs text-muted-foreground">累计入库数</p>
            <p className="mt-1 text-2xl font-bold font-mono">{totalCount}</p>
          </CardContent>
        </Card>
      </div>

      {/* 上下文选择 + 入库参数 */}
      <Card>
        <CardContent className="pt-4">
          <div className="flex flex-wrap items-end gap-3">
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
                        {kb.name} ({getSourceModuleLabel(kb.source_module)})
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              )}
            </div>
            <div className="space-y-1.5">
              <Label>起始时间（可选）</Label>
              <Input
                type="date"
                value={since}
                onChange={(e) => setSince(e.target.value)}
                className="w-[180px]"
              />
            </div>
            <PermissionGate resource="knowledge_base" action="edit">
              <Button onClick={handleIngest} disabled={!selectedKBId}>
                {records.find((r) => r.loading) ? (
                  <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                ) : (
                  <Download className="mr-2 h-4 w-4" />
                )}
                执行入库
              </Button>
            </PermissionGate>
          </div>
        </CardContent>
      </Card>

      {/* 入库记录表格 */}
      <Card>
        <CardContent className="p-0">
          <ScrollArea className="h-[300px] rounded-md border">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead className="text-xs font-medium text-muted-foreground">知识库</TableHead>
                  <TableHead className="text-xs font-medium text-muted-foreground">来源模块</TableHead>
                  <TableHead className="text-xs font-medium text-muted-foreground text-center">扫描</TableHead>
                  <TableHead className="text-xs font-medium text-muted-foreground text-center">入库</TableHead>
                  <TableHead className="text-xs font-medium text-muted-foreground text-center">跳过</TableHead>
                  <TableHead className="text-xs font-medium text-muted-foreground">状态</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {records.length === 0 ? (
                  <TableRow>
                    <TableCell colSpan={6} className="h-32">
                      {selectedKBId ? <NoIngestRecords /> : <NoKBSelected />}
                    </TableCell>
                  </TableRow>
                ) : (
                  records.map((r) => (
                    <TableRow key={`${r.kbId}-${records.indexOf(r)}`}>
                      <TableCell className="text-sm">{r.kbName}</TableCell>
                      <TableCell className="text-sm">{getSourceModuleLabel(r.sourceModule)}</TableCell>
                      <TableCell className="text-sm text-center font-mono">
                        {r.loading ? "—" : r.result?.scanned ?? "—"}
                      </TableCell>
                      <TableCell className="text-sm text-center font-mono">
                        {r.loading ? "—" : r.result?.ingested ?? "—"}
                      </TableCell>
                      <TableCell className="text-sm text-center font-mono">
                        {r.loading ? "—" : r.result?.skipped ?? "—"}
                      </TableCell>
                      <TableCell className="text-sm">
                        {r.loading ? (
                          <Badge variant="secondary">
                            <Loader2 className="mr-1 h-3 w-3 animate-spin" />入库中
                          </Badge>
                        ) : (
                          <Badge variant="default">已完成</Badge>
                        )}
                      </TableCell>
                    </TableRow>
                  ))
                )}
              </TableBody>
            </Table>
          </ScrollArea>
        </CardContent>
      </Card>
    </div>
  );
}
