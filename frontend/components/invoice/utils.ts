"use client";

// ========== 状态映射 ==========

/** 发票状态中文映射 */
export const INVOICE_STATUS_MAP: Record<string, string> = {
  draft: "草稿",
  submitted: "已提交",
  approved: "已审批",
  rejected: "已驳回",
  reimbursed: "已报销",
};

/** 获取状态中文名 */
export function getStatusLabel(status: string): string {
  return INVOICE_STATUS_MAP[status] || status;
}

/** 状态对应的 Badge 颜色类 */
export function getStatusBadgeClass(status: string): string {
  switch (status) {
    case "draft":
      return "border-gray-200 bg-gray-50 text-gray-700";
    case "submitted":
      return "border-blue-200 bg-blue-50 text-blue-700";
    case "approved":
      return "border-emerald-200 bg-emerald-50 text-emerald-700";
    case "rejected":
      return "border-rose-200 bg-rose-50 text-rose-700";
    case "reimbursed":
      return "border-violet-200 bg-violet-50 text-violet-700";
    default:
      return "border-gray-200 bg-gray-50 text-gray-700";
  }
}

// ========== 来源类型映射 ==========

/** 发票来源类型中文映射 */
export const SOURCE_TYPE_MAP: Record<string, string> = {
  office: "办公采购",
  canteen: "食堂支出",
  payment_request: "请款单",
  independent: "独立发票",
};

/** 获取来源类型中文名 */
export function getSourceTypeLabel(sourceType?: string): string {
  if (!sourceType) return "-";
  return SOURCE_TYPE_MAP[sourceType] || sourceType;
}

// ========== 金额格式化 ==========

/** 格式化金额为带千分位分隔符的 2 位小数 */
export function formatAmount(amount?: number | null): string {
  if (amount === null || amount === undefined || Number.isNaN(amount)) return "-";
  return amount.toLocaleString("zh-CN", {
    minimumFractionDigits: 2,
    maximumFractionDigits: 2,
  });
}

/** 格式化金额，前缀 ¥ 符号 */
export function formatCurrency(amount?: number | null): string {
  if (amount === null || amount === undefined || Number.isNaN(amount)) return "-";
  return `¥${formatAmount(amount)}`;
}

// ========== 日期格式化 ==========

/** 格式化日期（yyyy-MM-dd） */
export function formatDate(value?: string | null): string {
  if (!value) return "-";
  try {
    const d = new Date(value);
    if (Number.isNaN(d.getTime())) return value.slice(0, 10);
    return d.toLocaleDateString("zh-CN", {
      year: "numeric",
      month: "2-digit",
      day: "2-digit",
    });
  } catch {
    return value.slice(0, 10);
  }
}

// ========== 发票类型选项 ==========

/** 发票类型选项（用于下拉选择） */
export const INVOICE_TYPE_OPTIONS = [
  { value: "增值税普通发票", label: "增值税普通发票" },
  { value: "增值税专用发票", label: "增值税专用发票" },
  { value: "增值税电子普通发票", label: "增值税电子普通发票" },
  { value: "增值税电子专用发票", label: "增值税电子专用发票" },
  { value: "其他", label: "其他" },
];

/** 关联业务来源选项（用于下拉选择） */
export const SOURCE_TYPE_OPTIONS = [
  { value: "independent", label: "独立发票（无关联）" },
  { value: "office", label: "办公采购" },
  { value: "canteen", label: "食堂支出" },
  { value: "payment_request", label: "请款单" },
];

/** 状态选项（用于筛选下拉） */
export const STATUS_OPTIONS = [
  { value: "", label: "全部状态" },
  { value: "draft", label: "草稿" },
  { value: "submitted", label: "已提交" },
  { value: "approved", label: "已审批" },
  { value: "rejected", label: "已驳回" },
  { value: "reimbursed", label: "已报销" },
];
