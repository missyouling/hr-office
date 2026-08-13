import { afterEach, describe, expect, it, vi } from "vitest";
import {
  invoiceApi,
  normalizeParsingTaskDetail,
  normalizeInvoiceStats,
  type ParsingTaskDetail,
} from "./api-invoice";

const API_BASE = "http://localhost:3000/api";

function makePdfFile(name: string): File {
  return new File(["%PDF-1.4 test"], name, { type: "application/pdf" });
}

describe("发票批量上传 API", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("upload 构造 multipart 请求并返回逐文件结果", async () => {
    const items = [
      { original_name: "a.pdf", invoice_id: 1, task_id: 10, status: "pending" },
      { original_name: "b.pdf", status: "failed", error_code: "invalid_pdf", error: "文件不是有效的 PDF" },
    ];
    const mockFetch = vi.fn().mockResolvedValue({
      ok: true,
      json: async () => ({ items }),
    });
    vi.stubGlobal("fetch", mockFetch);

    const result = await invoiceApi.upload([makePdfFile("a.pdf"), makePdfFile("b.pdf")]);

    expect(result.items).toHaveLength(2);
    expect(result.items[0]).toMatchObject({ original_name: "a.pdf", invoice_id: 1, status: "pending" });
    expect(result.items[1].error_code).toBe("invalid_pdf");

    const [url, init] = mockFetch.mock.calls[0] as [string, RequestInit];
    expect(url).toBe(`${API_BASE}/invoices/upload`);
    expect(init.method).toBe("POST");
    expect(init.body).toBeInstanceOf(FormData);
    const form = init.body as FormData;
    const files = form.getAll("files");
    expect(files).toHaveLength(2);
    expect((files[0] as File).name).toBe("a.pdf");
    // multipart 请求不应手动设置 Content-Type（由浏览器自动带 boundary）
    expect(init.headers).not.toHaveProperty("Content-Type");
  });

  it("upload 携带 source_type / source_id 表单字段", async () => {
    const mockFetch = vi.fn().mockResolvedValue({
      ok: true,
      json: async () => ({ items: [] }),
    });
    vi.stubGlobal("fetch", mockFetch);

    await invoiceApi.upload([makePdfFile("a.pdf")], { source_type: "office", source_id: 7 });

    const [, init] = mockFetch.mock.calls[0] as [string, RequestInit];
    const form = init.body as FormData;
    expect(form.get("source_type")).toBe("office");
    expect(form.get("source_id")).toBe("7");
  });

  it("upload 后端返回错误状态时抛出 Error", async () => {
    const mockFetch = vi.fn().mockResolvedValue({
      ok: false,
      status: 413,
      json: async () => ({ error: "请求体过大" }),
    });
    vi.stubGlobal("fetch", mockFetch);

    await expect(invoiceApi.upload([makePdfFile("a.pdf")])).rejects.toThrow(
      "[413] 请求体过大",
    );
  });

  it("upload 网络异常时抛出 Error", async () => {
    vi.stubGlobal("fetch", vi.fn().mockRejectedValue(new TypeError("Failed to fetch")));
    await expect(invoiceApi.upload([makePdfFile("a.pdf")])).rejects.toThrow();
  });
});

describe("发票解析任务 API", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("getParsingTask 兼容 { task } 包装返回", async () => {
    const task = {
      id: 10,
      invoice_id: 3,
      status: "running",
      attempt_count: 1,
      max_attempts: 3,
    };
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue({ ok: true, json: async () => ({ task }) }),
    );

    const detail = await invoiceApi.getParsingTask(3);

    expect(detail.id).toBe(10);
    expect(detail.invoice_id).toBe(3);
    expect(detail.status).toBe("running");

    const [url] = mockFetchUrl();
    expect(url).toBe(`${API_BASE}/invoices/3/parsing-task`);
  });

  it("getParsingTask 兼容后端 { item } 包装返回", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue({
      ok: true,
      json: async () => ({ item: { id: 13, invoice_id: 6, status: "pending" } }),
    }));

    await expect(invoiceApi.getParsingTask(6)).resolves.toMatchObject({ id: 13, status: "pending" });
  });

  it("getParsingTask 兼容直接返回任务对象", async () => {
    const task = { id: 11, invoice_id: 4, status: "succeeded", fields: [{ key: "amount", label: "金额", value: 100, confidence: 0.99 }] };
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue({ ok: true, json: async () => task }));

    const detail = await invoiceApi.getParsingTask(4);

    expect(detail.status).toBe("succeeded");
    expect(detail.fields?.[0]).toMatchObject({ key: "amount", confidence: 0.99 });
  });

  it("retryParsingTask 发送 POST 请求并返回任务详情", async () => {
    const task = { id: 12, invoice_id: 5, status: "pending", attempt_count: 0 };
    const mockFetch = vi.fn().mockResolvedValue({ ok: true, json: async () => task });
    vi.stubGlobal("fetch", mockFetch);

    const detail = await invoiceApi.retryParsingTask(5);

    expect(detail.status).toBe("pending");
    const [url, init] = mockFetch.mock.calls[0] as [string, RequestInit];
    expect(url).toBe(`${API_BASE}/invoices/5/parsing-task/retry`);
    expect(init.method).toBe("POST");
  });

  it("normalizeParsingTaskDetail 对非法响应抛错", () => {
    expect(() => normalizeParsingTaskDetail(null)).toThrow("解析任务详情格式错误");
    expect(() => normalizeParsingTaskDetail({ foo: 1 })).toThrow("解析任务详情格式错误");
  });

  it("normalizeParsingTaskDetail 补齐缺省字段", () => {
    const detail: ParsingTaskDetail = normalizeParsingTaskDetail({ id: 1, invoice_id: 2 });
    expect(detail.status).toBe("pending");
    expect(detail.fields).toBeUndefined();
  });
});

/** 读取最近一次 fetch 调用的 URL */
function mockFetchUrl(): [string] {
  const calls = vi.mocked(fetch).mock.calls;
  return [calls[calls.length - 1][0] as string];
}

describe("发票统计 API", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("stats 归一化后端数组响应为 status->count 映射", async () => {
    const backendResponse = {
      total_count: 3,
      total_amount: 1500.5,
      by_status: [
        { status: "draft", count: 1, amount: 100 },
        { status: "approved", count: 2, amount: 1400.5 },
      ],
      by_source: [
        { source_type: "office", count: 2 },
        { source_type: "independent", count: 1 },
      ],
    };
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue({ ok: true, json: async () => backendResponse }),
    );

    const stats = await invoiceApi.stats();

    expect(stats.total).toBe(3);
    expect(stats.total_amount).toBe(1500.5);
    expect(stats.by_status).toEqual({ draft: 1, approved: 2 });
    expect(stats.by_source).toEqual({ office: 2, independent: 1 });

    const [url] = mockFetchUrl();
    expect(url).toBe(`${API_BASE}/invoices/stats`);
  });

  it("stats 后端返回空统计时给安全默认值", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue({
        ok: true,
        json: async () => ({ total_count: 0, total_amount: 0, by_status: [], by_source: [] }),
      }),
    );

    const stats = await invoiceApi.stats();

    expect(stats.total).toBe(0);
    expect(stats.by_status).toEqual({});
    expect(stats.by_source).toEqual({});
  });

  it("normalizeInvoiceStats 对非对象/非法行容错", () => {
    expect(normalizeInvoiceStats(null)).toEqual({
      total: 0,
      total_amount: 0,
      by_status: {},
      by_source: {},
    });
    const stats = normalizeInvoiceStats({
      total_count: 5,
      by_status: [
        { status: "draft", count: 3 },
        { status: "", count: 2 },
        { status: "rejected" },
      ],
      by_source: "bad",
    });
    // 空 status 跳过，缺 count 按 0 处理，by_source 非数组置空
    expect(stats.by_status).toEqual({ draft: 3, rejected: 0 });
    expect(stats.by_source).toEqual({});
    expect(stats.total_amount).toBe(0);
  });

  it("stats 后端返回错误状态时抛出 Error", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue({
        ok: false,
        status: 403,
        json: async () => ({ error: "无权限查看统计" }),
      }),
    );

    await expect(invoiceApi.stats()).rejects.toThrow("[403] 无权限查看统计");
  });
});

describe("发票 CSV 导出 API", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("exportCSV 携带筛选参数并返回 CSV Blob", async () => {
    const mockFetch = vi.fn().mockResolvedValue({
      ok: true,
      blob: async () => new Blob(["\uFEFFID,发票号码"], { type: "text/csv" }),
    });
    vi.stubGlobal("fetch", mockFetch);

    const blob = await invoiceApi.exportCSV({
      archive_status: "confirmed",
      date_from: "2026-01-01",
      page_size: 50,
    });

    expect(blob).toBeInstanceOf(Blob);
    expect(blob.type).toBe("text/csv");
    const [url] = mockFetch.mock.calls[0] as [string];
    expect(url).toBe(
      `${API_BASE}/invoices/export?archive_status=confirmed&date_from=2026-01-01&page_size=50`,
    );
  });

  it("exportCSV 无参数时不带查询字符串", async () => {
    const mockFetch = vi.fn().mockResolvedValue({ ok: true, blob: async () => new Blob([""]) });
    vi.stubGlobal("fetch", mockFetch);

    await invoiceApi.exportCSV();

    const [url] = mockFetch.mock.calls[0] as [string];
    expect(url).toBe(`${API_BASE}/invoices/export`);
  });

  it("exportCSV 数据量过大时抛出后端错误", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue({
        ok: false,
        status: 400,
        json: async () => ({ error: "导出数据量过大，请缩小筛选范围" }),
      }),
    );

    await expect(invoiceApi.exportCSV()).rejects.toThrow(
      "[400] 导出数据量过大，请缩小筛选范围",
    );
  });

  it("exportCSV 网络异常时抛出 Error", async () => {
    vi.stubGlobal("fetch", vi.fn().mockRejectedValue(new TypeError("Failed to fetch")));
    await expect(invoiceApi.exportCSV()).rejects.toThrow();
  });
});

describe("发票附件预览/下载 API", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("getAttachment 请求预览路径并返回 PDF Blob", async () => {
    const mockFetch = vi.fn().mockResolvedValue({
      ok: true,
      blob: async () => new Blob(["%PDF-1.4"], { type: "application/pdf" }),
    });
    vi.stubGlobal("fetch", mockFetch);

    const blob = await invoiceApi.getAttachment(9);

    expect(blob.type).toBe("application/pdf");
    const [url] = mockFetch.mock.calls[0] as [string];
    expect(url).toBe(`${API_BASE}/invoices/9/attachment`);
  });

  it("downloadAttachment 请求下载路径并返回 Blob", async () => {
    const mockFetch = vi.fn().mockResolvedValue({
      ok: true,
      blob: async () => new Blob(["%PDF-1.4"]),
    });
    vi.stubGlobal("fetch", mockFetch);

    await invoiceApi.downloadAttachment(9);

    const [url] = mockFetch.mock.calls[0] as [string];
    expect(url).toBe(`${API_BASE}/invoices/9/attachment/download`);
  });

  it("附件不存在（已确认/作废锁定或越权）时抛出 Error", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue({
        ok: false,
        status: 404,
        json: async () => ({ error: "附件不存在" }),
      }),
    );

    await expect(invoiceApi.getAttachment(9)).rejects.toThrow("[404] 附件不存在");
  });
});

describe("发票归档操作 API（confirm/void/correct）", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  const confirmedInvoice = {
    id: 1,
    invoice_no: "FP001",
    status: "approved",
    archive_status: "confirmed",
  };

  it("confirm 发送 POST 并返回发票与关联预警", async () => {
    const mockFetch = vi.fn().mockResolvedValue({
      ok: true,
      json: async () => ({ item: confirmedInvoice, warnings: ["发票金额与采购金额不一致"] }),
    });
    vi.stubGlobal("fetch", mockFetch);

    const result = await invoiceApi.confirm(1);

    expect(result.item.id).toBe(1);
    expect(result.item.archive_status).toBe("confirmed");
    expect(result.warnings).toEqual(["发票金额与采购金额不一致"]);
    const [url, init] = mockFetch.mock.calls[0] as [string, RequestInit];
    expect(url).toBe(`${API_BASE}/invoices/1/confirm`);
    expect(init.method).toBe("POST");
  });

  it("confirm 状态不满足（未审批/非待确认）时抛出 Error", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue({
        ok: false,
        status: 403,
        json: async () => ({ error: "仅已审批且待确认的发票可确认" }),
      }),
    );

    await expect(invoiceApi.confirm(1)).rejects.toThrow("[403] 仅已审批且待确认的发票可确认");
  });

  it("void 发送作废原因并返回 item", async () => {
    const mockFetch = vi.fn().mockResolvedValue({
      ok: true,
      json: async () => ({
        item: { ...confirmedInvoice, archive_status: "voided", voided_reason: "开票有误" },
      }),
    });
    vi.stubGlobal("fetch", mockFetch);

    const result = await invoiceApi.void(1, "开票有误");

    expect(result.item.archive_status).toBe("voided");
    expect(result.item.voided_reason).toBe("开票有误");
    const [url, init] = mockFetch.mock.calls[0] as [string, RequestInit];
    expect(url).toBe(`${API_BASE}/invoices/1/void`);
    expect(init.method).toBe("POST");
    expect(JSON.parse(String(init.body))).toEqual({ reason: "开票有误" });
  });

  it("void 缺原因时抛出后端校验错误", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue({
        ok: false,
        status: 400,
        json: async () => ({ error: "作废原因必填" }),
      }),
    );

    await expect(invoiceApi.void(1, "")).rejects.toThrow("[400] 作废原因必填");
  });

  it("correct 发送白名单字段与原因并返回 item", async () => {
    const mockFetch = vi.fn().mockResolvedValue({
      ok: true,
      json: async () => ({ item: { ...confirmedInvoice, amount: 200 } }),
    });
    vi.stubGlobal("fetch", mockFetch);

    const result = await invoiceApi.correct(1, {
      reason: "金额录入有误",
      amount: 200,
      invoice_no: "FP002",
    });

    expect(result.item.amount).toBe(200);
    const [url, init] = mockFetch.mock.calls[0] as [string, RequestInit];
    expect(url).toBe(`${API_BASE}/invoices/1/correct`);
    expect(JSON.parse(String(init.body))).toEqual({
      reason: "金额录入有误",
      amount: 200,
      invoice_no: "FP002",
    });
  });

  it("correct 身份键冲突时抛出后端冲突错误", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue({
        ok: false,
        status: 409,
        json: async () => ({ error: "发票身份键与活动记录冲突" }),
      }),
    );

    await expect(invoiceApi.correct(1, { reason: "改票号" })).rejects.toThrow(
      "[409] 发票身份键与活动记录冲突",
    );
  });
});
