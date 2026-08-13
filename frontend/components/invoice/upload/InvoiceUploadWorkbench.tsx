"use client";

import { useEffect, useMemo, useRef, useState } from "react";
import { toast } from "sonner";
import { Loader2, RotateCcw, Trash2, UploadCloud } from "lucide-react";

import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { invoiceApi } from "@/lib/api-invoice";
import { MAX_FILE_SIZE, MAX_UPLOAD_FILES, validateUploadFiles } from "./validation";
import { UploadItemRow } from "./UploadItemRow";
import { InvoiceParseResultDialog } from "./InvoiceParseResultDialog";
import { applyUploadResults, mapTaskStatusToFileStatus, shouldRetryParsingTask, type UploadItemState } from "./upload-state";

interface InvoiceUploadWorkbenchProps {
  /** 上传成功后回调（用于刷新发票列表） */
  onDone?: (count: number) => void;
}

export function InvoiceUploadWorkbench({ onDone }: InvoiceUploadWorkbenchProps) {
  const [items, setItems] = useState<UploadItemState[]>([]);
  const [dragActive, setDragActive] = useState(false);
  const [uploading, setUploading] = useState(false);
  const [editInvoiceId, setEditInvoiceId] = useState<number | null>(null);
  const fileInputRef = useRef<HTMLInputElement>(null);
  const mountedRef = useRef(true);
  useEffect(() => () => { mountedRef.current = false; }, []);

  /** 当前需要轮询解析进度的任务（提取为稳定值，避免将整个 items 数组作为 effect 依赖） */
  const pollingTargets = useMemo(
    () => items.filter((item) => item.invoiceId !== undefined && (item.taskStatus === "pending" || item.taskStatus === "running")),
    [items],
  );

  useEffect(() => {
    if (pollingTargets.length === 0) return;
    let cancelled = false;
    const timer = setInterval(async () => {
      await Promise.all(pollingTargets.map(async (target) => {
        try {
          const task = await invoiceApi.getParsingTask(target.invoiceId as number);
          if (cancelled || !mountedRef.current) return;
          setItems((prev) => prev.map((item) => item.key !== target.key ? item : {
            ...item,
            status: mapTaskStatusToFileStatus(task.status),
            taskStatus: task.status,
            error: task.status === "failed" ? task.last_error || task.error_code || "解析失败" : undefined,
          }));
        } catch { /* 查询失败时保留状态，下一轮继续 */ }
      }));
    }, 2000);
    return () => { cancelled = true; clearInterval(timer); };
  }, [pollingTargets]);

  const hasFailed = items.some((item) => item.status === "failed");

  /** 前端校验并追加文件（总量 ≤50） */
  const addFiles = (files: File[]) => {
    const { accepted, rejected } = validateUploadFiles(files);
    const remaining = MAX_UPLOAD_FILES - items.length;
    if (remaining < accepted.length) {
      accepted.slice(remaining).forEach((file) => {
        rejected.push({
          name: file.name,
          code: "too_many",
          message: `每批最多 ${MAX_UPLOAD_FILES} 份文件`,
        });
      });
    }
    const acceptedFinal = accepted.slice(0, Math.max(0, remaining));

    if (rejected.length > 0) {
      const first = rejected[0];
      toast.error(`已拒绝 ${rejected.length} 个文件：${first.message}${rejected.length > 1 ? " 等" : ""}`);
    }
    if (acceptedFinal.length > 0) {
      setItems((prev) => [
        ...prev,
        ...acceptedFinal.map((file, idx) => ({
          key: `${Date.now()}-${idx}-${file.name}`,
          file,
          status: "pending" as const,
        })),
      ]);
    }
  };

  /** 选择文件（点击上传区） */
  const handleSelect = (e: React.ChangeEvent<HTMLInputElement>) => {
    if (e.target.files) addFiles(Array.from(e.target.files));
    e.target.value = "";
  };

  /** 拖拽文件 */
  const handleDrop = (e: React.DragEvent) => {
    e.preventDefault();
    setDragActive(false);
    if (e.dataTransfer.files) addFiles(Array.from(e.dataTransfer.files));
  };

  /** 批量上传（一次 multipart 请求，后端逐文件返回结果） */
  const handleUploadAll = async () => {
    const pending = items.filter((item) => item.status === "pending");
    if (pending.length === 0) return;
    setUploading(true);
    setItems((prev) => prev.map((item) => (item.status === "pending" ? { ...item, status: "running" } : item)));
    try {
      const result = await invoiceApi.upload(pending.map((item) => item.file));
      setItems((prev) => applyUploadResults(prev, result.items ?? []));
      const okCount = (result.items ?? []).filter((r) => r.status === "pending").length;
      const failCount = (result.items ?? []).length - okCount;
      if (failCount > 0) toast.error(`${failCount} 个文件上传失败，可重试`);
      else toast.success(`${okCount} 个文件已上传`);
      onDone?.(okCount);
    } catch (err) {
      const message = err instanceof Error ? err.message : "上传失败";
      setItems((prev) => prev.map((item) => (item.status === "running" ? { ...item, status: "failed", error: message } : item)));
      toast.error(message);
    } finally {
      setUploading(false);
    }
  };

  /** 上传一组文件（重试用），按响应合并状态 */
  const uploadFiles = async (targets: UploadItemState[]) => {
    const result = await invoiceApi.upload(targets.map((item) => item.file));
    setItems((prev) => applyUploadResults(prev, result.items ?? []));
  };

  const retryParsing = async (item: UploadItemState) => {
    const task = await invoiceApi.retryParsingTask(item.invoiceId as number);
    if (mountedRef.current) setItems((prev) => prev.map((current) => current.key === item.key ? { ...current, status: mapTaskStatusToFileStatus(task.status), taskStatus: task.status, error: undefined } : current));
  };

  /** 重试单个失败文件 */
  const handleRetryOne = async (key: string) => {
    const item = items.find((it) => it.key === key);
    if (!item) return;
    setItems((prev) => prev.map((it) => (it.key === key ? { ...it, status: "running", error: undefined } : it)));
    try {
      if (shouldRetryParsingTask(item)) await retryParsing(item);
      else await uploadFiles([item]);
    } catch (err) {
      const message = err instanceof Error ? err.message : "重试失败";
      setItems((prev) => prev.map((it) => (it.key === key ? { ...it, status: "failed", error: message } : it)));
      toast.error(message);
    }
  };

  /** 重试全部失败文件 */
  const handleRetryFailed = async () => {
    const failed = items.filter((item) => item.status === "failed");
    setItems((prev) => prev.map((item) => (item.status === "failed" ? { ...item, status: "running", error: undefined } : item)));
    try {
      await Promise.all(failed.map((item) => shouldRetryParsingTask(item) ? retryParsing(item) : uploadFiles([item])));
    } catch (err) {
      const message = err instanceof Error ? err.message : "重试失败";
      setItems((prev) => prev.map((item) => (item.status === "running" ? { ...item, status: "failed", error: message } : item)));
      toast.error(message);
    }
  };

  /** 移除单个文件（仅未上传成功项） */
  const handleRemove = (key: string) => {
    setItems((prev) => prev.filter((item) => item.key !== key));
  };

  const pendingCount = items.filter((item) => item.status === "pending").length;
  const succeededCount = items.filter((item) => item.status === "succeeded").length;

  return (
    <Card>
      <CardHeader className="pb-3">
        <div className="flex flex-wrap items-center justify-between gap-4">
          <CardTitle>发票上传与解析</CardTitle>
          <p className="text-sm text-muted-foreground">
            最多 {MAX_UPLOAD_FILES} 份 / 单文件 ≤ {MAX_FILE_SIZE / (1024 * 1024)}MB / 仅支持 PDF
          </p>
        </div>
      </CardHeader>

      <CardContent className="space-y-4">
        {/* 拖拽 / 选择上传区 */}
        <div
          role="button"
          tabIndex={0}
          aria-label="选择或拖拽 PDF 文件上传"
          onClick={() => fileInputRef.current?.click()}
          onKeyDown={(e) => {
            if (e.key === "Enter" || e.key === " ") fileInputRef.current?.click();
          }}
          onDragOver={(e) => {
            e.preventDefault();
            setDragActive(true);
          }}
          onDragLeave={() => setDragActive(false)}
          onDrop={handleDrop}
          className={`flex cursor-pointer flex-col items-center justify-center gap-2 rounded-lg border-2 border-dashed px-4 py-8 text-center transition-colors ${
            dragActive ? "border-primary bg-primary/5" : "border-border hover:border-primary/50"
          }`}
        >
          <UploadCloud className="h-8 w-8 text-muted-foreground" />
          <p className="text-sm font-medium">拖拽 PDF 文件到此处，或点击选择文件</p>
          <p className="text-xs text-muted-foreground">
            支持批量选择，每批最多 {MAX_UPLOAD_FILES} 份、单份不超过 20MB
          </p>
          <input
            ref={fileInputRef}
            type="file"
            accept=".pdf"
            multiple
            className="hidden"
            onChange={handleSelect}
          />
        </div>

        {/* 文件列表 */}
        {items.length > 0 && (
          <div className="overflow-hidden rounded-lg border">
            <div className="flex items-center justify-between border-b bg-muted/50 px-3 py-2">
              <span className="text-sm font-medium">
                共 {items.length} 个文件（待上传 {pendingCount} / 已受理 {succeededCount}）
              </span>
              <div className="flex gap-2">
                {hasFailed && (
                  <Button size="sm" variant="outline" onClick={handleRetryFailed} disabled={uploading}>
                    <RotateCcw className="mr-1 h-3.5 w-3.5" />
                    重试失败项
                  </Button>
                )}
                <Button size="sm" variant="ghost" onClick={() => setItems([])} disabled={uploading}>
                  <Trash2 className="mr-1 h-3.5 w-3.5" />
                  清空
                </Button>
              </div>
            </div>

            <ul className="max-h-72 divide-y overflow-y-auto">
              {items.map((item) => (
                <UploadItemRow
                  key={item.key}
                  item={item}
                  uploading={uploading}
                  onRetry={handleRetryOne}
                  onRemove={handleRemove}
                  onEdit={(invoiceId) => setEditInvoiceId(invoiceId)}
                />
              ))}
            </ul>
          </div>
        )}

        {/* 上传操作 */}
        {pendingCount > 0 && (
          <div className="flex justify-end">
            <Button onClick={handleUploadAll} disabled={uploading}>
              {uploading ? <Loader2 className="mr-1 h-4 w-4 animate-spin" /> : <UploadCloud className="mr-1 h-4 w-4" />}
              {uploading ? "上传中..." : `上传 ${pendingCount} 个文件`}
            </Button>
          </div>
        )}
      </CardContent>

      {/* 识别字段编辑 Dialog */}
      <InvoiceParseResultDialog
        open={editInvoiceId !== null}
        onOpenChange={(v) => {
          if (!v) setEditInvoiceId(null);
        }}
        invoiceId={editInvoiceId ?? 0}
        onSaved={() => onDone?.(succeededCount)}
      />
    </Card>
  );
}
