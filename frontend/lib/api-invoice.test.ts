import { afterEach, describe, expect, it, vi } from "vitest";
import {
  invoiceApi,
  normalizeParsingTaskDetail,
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
