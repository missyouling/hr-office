import { describe, expect, it } from "vitest";
import { ARCHIVE_PAGE_SIZE, buildArchiveQuery, clampArchivePage, normalizeArchiveSummary } from "./invoice-archive-logic";

describe("归档筛选参数", () => {
  it("忽略空筛选并裁剪关键词", () => {
    expect(buildArchiveQuery({ archiveStatus: "", sourceType: "", keyword: "  差旅  " }, 0).toString()).toBe(`keyword=%E5%B7%AE%E6%97%85&page=1&page_size=${ARCHIVE_PAGE_SIZE}`);
  });
  it("保留状态、来源和关键词组合", () => {
    expect(buildArchiveQuery({ archiveStatus: "confirmed", sourceType: "office", keyword: "A-001" }, 2).toString()).toBe(`archive_status=confirmed&source_type=office&keyword=A-001&page=2&page_size=${ARCHIVE_PAGE_SIZE}`);
  });
});

describe("归档摘要与分页", () => {
  it("缺少的状态统计按零补齐", () => {
    expect(normalizeArchiveSummary(5, { confirmed: 3 })).toEqual({ total: 5, pending: 0, confirmed: 3, voided: 0 });
  });
  it("页码不会超出有效范围", () => {
    expect(clampArchivePage(-1, 31)).toBe(1);
    expect(clampArchivePage(5, 31)).toBe(3);
    expect(clampArchivePage(2, 0)).toBe(1);
  });
});
