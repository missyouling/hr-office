export const ARCHIVE_PAGE_SIZE = 15;

export interface ArchiveFilters {
  archiveStatus: string;
  sourceType: string;
  keyword: string;
}

export interface ArchiveSummary {
  total: number;
  pending: number;
  confirmed: number;
  voided: number;
}

/** 生成归档列表筛选参数。 */
export function buildArchiveQuery(filters: ArchiveFilters, page?: number): URLSearchParams {
  const params = new URLSearchParams();
  if (filters.archiveStatus) params.set("archive_status", filters.archiveStatus);
  if (filters.sourceType) params.set("source_type", filters.sourceType);
  if (filters.keyword.trim()) params.set("keyword", filters.keyword.trim());
  if (page !== undefined) {
    params.set("page", String(Math.max(1, page)));
    params.set("page_size", String(ARCHIVE_PAGE_SIZE));
  }
  return params;
}

/** 归一化统计字段，接口缺失状态按零展示。 */
export function normalizeArchiveSummary(total: number, byStatus?: Record<string, number>): ArchiveSummary {
  return { total: Math.max(0, total), pending: byStatus?.pending ?? 0, confirmed: byStatus?.confirmed ?? 0, voided: byStatus?.voided ?? 0 };
}

/** 将页码限制在结果总页数内。 */
export function clampArchivePage(page: number, total: number): number {
  const totalPages = Math.max(1, Math.ceil(Math.max(0, total) / ARCHIVE_PAGE_SIZE));
  return Math.min(Math.max(1, page), totalPages);
}
