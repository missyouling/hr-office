import type { InvoiceParsingTaskStatus, InvoiceUploadItem } from "@/lib/api-invoice";

export type FileStatus = "pending" | "running" | "succeeded" | "failed";
export interface UploadItemState { key: string; file: File; status: FileStatus; error?: string; duplicateWarning?: boolean; invoiceId?: number; taskId?: number; taskStatus?: InvoiceParsingTaskStatus; }
export function mapTaskStatusToFileStatus(status: InvoiceParsingTaskStatus): FileStatus { return status === "succeeded" ? "succeeded" : status === "failed" ? "failed" : "running"; }
export function shouldRetryParsingTask(item: Pick<UploadItemState, "invoiceId">): boolean { return item.invoiceId !== undefined; }
export function applyUploadResults(items: UploadItemState[], results: InvoiceUploadItem[]): UploadItemState[] {
  return items.map((item) => { const result = results.find((candidate) => candidate.original_name === item.file.name); if (!result) return item; if (result.status === "failed") return { ...item, status: "failed", taskStatus: undefined, error: result.error || result.error_code || "上传失败" }; return { ...item, status: "running", taskStatus: "pending", error: undefined, invoiceId: result.invoice_id, taskId: result.task_id, duplicateWarning: result.duplicate_warning }; });
}
