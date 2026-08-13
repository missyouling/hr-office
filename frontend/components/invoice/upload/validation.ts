"use client";

import type { ParsedInvoiceField } from "@/lib/api-invoice";

// ========== 上传限制常量 ==========

/** 单批最多文件数 */
export const MAX_UPLOAD_FILES = 50;
/** 单文件最大体积（20MB） */
export const MAX_FILE_SIZE = 20 << 20; // 20971520 字节
/** 允许的文件扩展名 */
export const ALLOWED_EXTENSION = ".pdf";
/** 低置信度阈值：confidence < 该值时视为识别置信度低 */
export const LOW_CONFIDENCE_THRESHOLD = 0.8;

// ========== 前端文件校验 ==========

export type FileRejectCode = "too_many" | "invalid_extension" | "file_too_large";

/** 单个文件校验失败信息 */
export interface FileValidationError {
  name: string;
  code: FileRejectCode;
  message: string;
}

/** 文件校验结果：accepted 为可上传文件，rejected 为被拒绝文件 */
export interface UploadValidationResult {
  accepted: File[];
  rejected: FileValidationError[];
}

/** 判断文件名是否为允许的 PDF 扩展名（不区分大小写） */
export function isPdfFileName(name: string): boolean {
  return name.toLowerCase().endsWith(ALLOWED_EXTENSION);
}

/**
 * 前端文件校验：数量（≤50）、扩展名（.pdf）、大小（≤20MB）。
 * 数量超限时保留前 50 个文件，超出部分标记为 too_many。
 */
export function validateUploadFiles(files: File[]): UploadValidationResult {
  const accepted: File[] = [];
  const rejected: FileValidationError[] = [];

  if (files.length > MAX_UPLOAD_FILES) {
    files.slice(MAX_UPLOAD_FILES).forEach((file) => {
      rejected.push({
        name: file.name,
        code: "too_many",
        message: `每批最多 ${MAX_UPLOAD_FILES} 份文件`,
      });
    });
  }

  const candidates = files.slice(0, MAX_UPLOAD_FILES);
  candidates.forEach((file) => {
    if (!isPdfFileName(file.name)) {
      rejected.push({
        name: file.name,
        code: "invalid_extension",
        message: "仅支持 PDF 文件",
      });
      return;
    }
    if (file.size > MAX_FILE_SIZE) {
      rejected.push({
        name: file.name,
        code: "file_too_large",
        message: "单个文件不能超过 20MB",
      });
      return;
    }
    accepted.push(file);
  });

  return { accepted, rejected };
}

// ========== 识别字段展示与高亮 ==========

/** 识别字段配置（与发票识别结果对应） */
export interface ParseFieldDef {
  key: string;
  label: string;
  required: boolean;
}

/** 识别字段列表：用于编辑表单与缺失/低置信度高亮 */
export const PARSE_FIELD_DEFS: ParseFieldDef[] = [
  { key: "invoice_no", label: "发票号", required: true },
  { key: "invoice_date", label: "开票日期", required: true },
  { key: "invoice_type", label: "发票类型", required: false },
  { key: "amount", label: "金额", required: true },
  { key: "tax_amount", label: "税额", required: false },
  { key: "total_amount", label: "含税总额", required: false },
  { key: "seller", label: "销售方", required: true },
  { key: "seller_tax_no", label: "销售方税号", required: false },
  { key: "buyer", label: "购方", required: false },
  { key: "purpose", label: "用途", required: false },
  { key: "remark", label: "备注", required: false },
];

/** 判断识别字段值是否缺失（空串 / null / undefined） */
export function isFieldMissing(value: unknown): boolean {
  return value === null || value === undefined || String(value).trim() === "";
}

/** 判断字段是否低置信度（task.fields 中 confidence 低于阈值） */
export function isFieldLowConfidence(fields: ParsedInvoiceField[] | Record<string, unknown> | null | undefined, key: string): boolean {
  if (!fields) return false;
  if (Array.isArray(fields)) {
    const field = fields.find((item) => item.key === key);
    return field?.confidence != null && field.confidence < LOW_CONFIDENCE_THRESHOLD;
  }
  const lowConfidence = fields.low_confidence_fields;
  return Array.isArray(lowConfidence) && lowConfidence.includes(key);
}

/**
 * 识别字段是否允许编辑保存：
 * 仅草稿状态且当前用户有发票 edit 权限时可编辑，否则只读。
 */
export function canEditParsedFields(status: string | undefined, canEdit: boolean): boolean {
  return status === "draft" && canEdit;
}
