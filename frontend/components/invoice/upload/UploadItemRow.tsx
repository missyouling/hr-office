"use client";
import { FileText, Pencil, RotateCcw, X } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import type { InvoiceParsingTaskStatus } from "@/lib/api-invoice";
import { type FileStatus, type UploadItemState } from "./upload-state";
export type { FileStatus, UploadItemState } from "./upload-state";
interface UploadItemRowProps { item: UploadItemState; uploading: boolean; onRetry: (key: string) => void; onRemove: (key: string) => void; onEdit: (invoiceId: number) => void; }
const STATUS_LABEL: Record<FileStatus, string> = { pending: "待上传", running: "上传中", succeeded: "解析成功", failed: "失败" };
const TASK_STATUS_LABEL: Record<InvoiceParsingTaskStatus, string> = { pending: "待解析", running: "解析中", succeeded: "解析成功", failed: "解析失败" };
const STATUS_CLASS: Record<FileStatus, string> = { pending: "border-gray-200 bg-gray-50 text-gray-700", running: "border-blue-200 bg-blue-50 text-blue-700", succeeded: "border-emerald-200 bg-emerald-50 text-emerald-700", failed: "border-rose-200 bg-rose-50 text-rose-700" };
const TASK_STATUS_CLASS: Record<InvoiceParsingTaskStatus, string> = { pending: STATUS_CLASS.pending, running: STATUS_CLASS.running, succeeded: STATUS_CLASS.succeeded, failed: STATUS_CLASS.failed };
export function UploadItemRow({ item, uploading, onRetry, onRemove, onEdit }: UploadItemRowProps) {
  const label = item.taskStatus ? TASK_STATUS_LABEL[item.taskStatus] : STATUS_LABEL[item.status];
  const className = item.taskStatus ? TASK_STATUS_CLASS[item.taskStatus] : STATUS_CLASS[item.status];
  return <li className="flex items-center gap-3 px-3 py-2"><FileText className="h-4 w-4 shrink-0 text-muted-foreground" /><span className="min-w-0 flex-1 truncate text-sm">{item.file.name}</span><span className="shrink-0 text-xs text-muted-foreground">{(item.file.size / (1024 * 1024)).toFixed(2)} MB</span>{item.status === "running" && <span className="h-4 w-4 shrink-0 animate-spin rounded-full border-2 border-blue-500 border-t-transparent" aria-label="处理中" />}<Badge variant="outline" className={`shrink-0 ${className}`}>{label}</Badge>{item.duplicateWarning && <Badge variant="outline" className="shrink-0 border-amber-300 bg-amber-50 text-amber-700">重复预警</Badge>}{item.status === "failed" && item.error && <span className="hidden max-w-[200px] truncate text-xs text-rose-600 md:inline">{item.error}</span>}{item.status === "succeeded" && item.invoiceId && <Button size="sm" variant="outline" className="shrink-0" disabled={uploading} onClick={() => onEdit(item.invoiceId as number)}><Pencil className="mr-1 h-3.5 w-3.5" />编辑字段</Button>}{item.status === "failed" && <Button size="sm" variant="outline" className="shrink-0" disabled={uploading} onClick={() => onRetry(item.key)}><RotateCcw className="mr-1 h-3.5 w-3.5" />重试</Button>}{item.status === "pending" && <Button size="sm" variant="ghost" className="shrink-0" aria-label={`移除 ${item.file.name}`} disabled={uploading} onClick={() => onRemove(item.key)}><X className="h-4 w-4" /></Button>}</li>;
}
