import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import type { ComponentProps, ReactNode } from "react";
import { InvoiceDetailDialog } from "./InvoiceDetailDialog";
import { invoiceApi } from "@/lib/api-invoice";
import type { Invoice } from "@/lib/api-invoice";
import { toast } from "sonner";

/** 可变的当前用户角色（在用例中切换以验证角色门控） */
const authState = vi.hoisted(() => ({ role: "admin" }));

// ========== 依赖 Mock ==========

vi.mock("@/components/permission-gate", () => ({
  PermissionGate: ({ children }: { children: ReactNode }) => <>{children}</>,
}));

vi.mock("@/lib/auth", () => ({
  useAuth: () => ({ user: { role: authState.role } }),
}));

vi.mock("@/lib/permissions", () => ({
  normalizeRole: (role: string) => role,
}));

vi.mock("@/lib/api-invoice", () => ({
  invoiceApi: {
    getAttachment: vi.fn(),
    downloadAttachment: vi.fn(),
    confirm: vi.fn(),
    void: vi.fn(),
    submit: vi.fn(),
    approve: vi.fn(),
    reject: vi.fn(),
    remove: vi.fn(),
    reimburse: vi.fn(),
  },
}));

vi.mock("sonner", () => ({
  toast: { success: vi.fn(), error: vi.fn(), warning: vi.fn(), loading: vi.fn(), dismiss: vi.fn() },
}));

// ========== 辅助函数 ==========

/** 构造测试用发票（默认：admin 可操作的 已审批+待确认+有附件） */
function makeInvoice(overrides: Partial<Invoice> = {}): Invoice {
  return {
    id: 1,
    invoice_no: "FP2026001",
    invoice_date: "2026-06-01",
    amount: 100,
    tax_amount: 13,
    total_amount: 113,
    seller: "测试销售方有限公司",
    reimburse_amount: 0,
    status: "approved",
    archive_status: "pending",
    attachment_file_id: 99,
    created_at: "2026-06-01T00:00:00Z",
    updated_at: "2026-06-01T00:00:00Z",
    ...overrides,
  };
}

function renderDialog(
  invoice: Invoice | null,
  props?: Partial<ComponentProps<typeof InvoiceDetailDialog>>,
) {
  const onOpenChange = vi.fn();
  const onSuccess = vi.fn();
  render(
    <InvoiceDetailDialog
      open
      onOpenChange={onOpenChange}
      invoice={invoice}
      onSuccess={onSuccess}
      {...props}
    />,
  );
  return { onOpenChange, onSuccess };
}

/** 模拟新窗口对象（jsdom 无真实 window.open 行为） */
function mockPopupWindow(extra: Partial<Window> = {}) {
  const popup = {
    focus: vi.fn(),
    print: vi.fn(),
    addEventListener: vi.fn(),
    onload: null,
    ...extra,
  } as unknown as Window;
  const openSpy = vi.spyOn(window, "open").mockReturnValue(popup);
  return { openSpy, popup };
}

beforeEach(() => {
  vi.clearAllMocks();
  authState.role = "admin";
  // jsdom 未实现 Blob URL API，需打桩
  Object.defineProperty(URL, "createObjectURL", {
    writable: true,
    configurable: true,
    value: vi.fn(() => "blob:mock-url"),
  });
  Object.defineProperty(URL, "revokeObjectURL", {
    writable: true,
    configurable: true,
    value: vi.fn(),
  });
});

afterEach(() => {
  vi.restoreAllMocks();
  delete (URL as unknown as { createObjectURL?: unknown }).createObjectURL;
  delete (URL as unknown as { revokeObjectURL?: unknown }).revokeObjectURL;
});

// ========== 测试用例 ==========

describe("InvoiceDetailDialog 渲染", () => {
  it("展示发票详情字段", () => {
    renderDialog(makeInvoice());

    expect(screen.getByText("发票号")).toBeInTheDocument();
    expect(screen.getByText("FP2026001")).toBeInTheDocument();
    expect(screen.getByText("测试销售方有限公司")).toBeInTheDocument();
    expect(screen.getByText("¥100.00")).toBeInTheDocument();
  });

  it("invoice 为 null 时不渲染内容", () => {
    renderDialog(null);
    expect(screen.queryByText("发票详情")).not.toBeInTheDocument();
  });
});

describe("附件操作按钮显示控制", () => {
  it("待确认且有附件（attachment_file_id）时显示预览/下载/打印", () => {
    renderDialog(makeInvoice());

    expect(screen.getByRole("button", { name: /原件预览/ })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /下载附件/ })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /打印/ })).toBeInTheDocument();
  });

  it("待确认且仅 attachment_url（遗留数据）时仍显示附件按钮", () => {
    renderDialog(makeInvoice({ attachment_file_id: null, attachment_url: "https://legacy.example/invoice.pdf" }));

    expect(screen.getByRole("button", { name: /原件预览/ })).toBeInTheDocument();
  });

  it("已确认（confirmed）后附件锁定，不显示附件按钮", () => {
    renderDialog(makeInvoice({ archive_status: "confirmed" }));

    expect(screen.queryByRole("button", { name: /原件预览/ })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /下载附件/ })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /打印/ })).not.toBeInTheDocument();
  });

  it("待确认但无附件时不显示附件按钮", () => {
    renderDialog(makeInvoice({ attachment_file_id: null, attachment_url: undefined }));

    expect(screen.queryByRole("button", { name: /原件预览/ })).not.toBeInTheDocument();
  });
});

describe("确认归档", () => {
  it("点击后调用 confirm，无预警时提示成功并刷新关闭", async () => {
    vi.mocked(invoiceApi.confirm).mockResolvedValue({
      item: makeInvoice({ archive_status: "confirmed" }),
      warnings: [],
    });
    const { onSuccess, onOpenChange } = renderDialog(makeInvoice());

    fireEvent.click(screen.getByRole("button", { name: /确认归档/ }));

    await waitFor(() => expect(invoiceApi.confirm).toHaveBeenCalledWith(1));
    expect(toast.success).toHaveBeenCalledWith("发票已确认归档");
    expect(onSuccess).toHaveBeenCalled();
    expect(onOpenChange).toHaveBeenCalledWith(false);
  });

  it("有后端关联预警时展示预警弹窗", async () => {
    vi.mocked(invoiceApi.confirm).mockResolvedValue({
      item: makeInvoice({ archive_status: "confirmed" }),
      warnings: ["发票金额与采购金额不一致", "购方名称与税号不匹配"],
    });
    renderDialog(makeInvoice());

    fireEvent.click(screen.getByRole("button", { name: /确认归档/ }));

    expect(await screen.findByText("发票金额与采购金额不一致")).toBeInTheDocument();
    expect(screen.getByText("购方名称与税号不匹配")).toBeInTheDocument();
    expect(toast.success).toHaveBeenCalledWith("发票已确认归档，存在关联预警");
  });

  it("后端拒绝时提示错误", async () => {
    vi.mocked(invoiceApi.confirm).mockRejectedValue(new Error("[403] 仅已审批且待确认的发票可确认"));
    renderDialog(makeInvoice());

    fireEvent.click(screen.getByRole("button", { name: /确认归档/ }));

    await waitFor(() =>
      expect(toast.error).toHaveBeenCalledWith("[403] 仅已审批且待确认的发票可确认"),
    );
  });

  it("非 admin 角色不显示确认归档与作废按钮", () => {
    authState.role = "manager";
    renderDialog(makeInvoice());

    expect(screen.queryByRole("button", { name: /确认归档/ })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /作废/ })).not.toBeInTheDocument();
  });
});

describe("作废归档", () => {
  it("输入原因后调用 void", async () => {
    const promptSpy = vi.spyOn(window, "prompt").mockReturnValue("开票有误");
    vi.mocked(invoiceApi.void).mockResolvedValue({
      item: makeInvoice({ archive_status: "voided" }),
    });
    const { onSuccess } = renderDialog(makeInvoice());

    fireEvent.click(screen.getByRole("button", { name: /作废/ }));

    await waitFor(() => expect(invoiceApi.void).toHaveBeenCalledWith(1, "开票有误"));
    expect(promptSpy).toHaveBeenCalledWith("请输入作废原因：");
    expect(toast.success).toHaveBeenCalledWith("发票已作废");
    expect(onSuccess).toHaveBeenCalled();
  });

  it("取消 prompt 时不调用 void", () => {
    vi.spyOn(window, "prompt").mockReturnValue(null);
    renderDialog(makeInvoice());

    fireEvent.click(screen.getByRole("button", { name: /作废/ }));

    expect(invoiceApi.void).not.toHaveBeenCalled();
  });
});

describe("受控原件预览", () => {
  it("获取 Blob 后新窗口打开，窗口关闭时清理 Blob URL", async () => {
    vi.mocked(invoiceApi.getAttachment).mockResolvedValue(
      new Blob(["%PDF-1.4"], { type: "application/pdf" }),
    );
    const addEventListenerMock = vi.fn();
    const { openSpy } = mockPopupWindow({ addEventListener: addEventListenerMock });

    renderDialog(makeInvoice());
    fireEvent.click(screen.getByRole("button", { name: /原件预览/ }));

    await waitFor(() => expect(invoiceApi.getAttachment).toHaveBeenCalledWith(1));
    expect(URL.createObjectURL).toHaveBeenCalled();
    expect(openSpy).toHaveBeenCalledWith("blob:mock-url", "_blank");

    // 触发新窗口关闭时的清理回调
    const cleanup = addEventListenerMock.mock.calls[0][1] as () => void;
    cleanup();
    expect(URL.revokeObjectURL).toHaveBeenCalledWith("blob:mock-url");
  });

  it("弹窗被浏览器拦截时提示并立即清理 URL", async () => {
    vi.mocked(invoiceApi.getAttachment).mockResolvedValue(
      new Blob(["%PDF-1.4"], { type: "application/pdf" }),
    );
    vi.spyOn(window, "open").mockReturnValue(null);

    renderDialog(makeInvoice());
    fireEvent.click(screen.getByRole("button", { name: /原件预览/ }));

    await waitFor(() =>
      expect(toast.error).toHaveBeenCalledWith("浏览器阻止了预览窗口，请允许弹窗后重试"),
    );
    expect(URL.revokeObjectURL).toHaveBeenCalledWith("blob:mock-url");
  });
});

describe("附件下载", () => {
  it("Blob + 临时 a 标签触发下载并清理 URL", async () => {
    vi.mocked(invoiceApi.downloadAttachment).mockResolvedValue(
      new Blob(["%PDF-1.4"], { type: "application/pdf" }),
    );
    const createElementSpy = vi.spyOn(document, "createElement");
    const clickSpy = vi.spyOn(HTMLAnchorElement.prototype, "click");

    renderDialog(makeInvoice());
    fireEvent.click(screen.getByRole("button", { name: /下载附件/ }));

    await waitFor(() => expect(invoiceApi.downloadAttachment).toHaveBeenCalledWith(1));
    const link = createElementSpy.mock.results
      .map((result) => result.value)
      .find((el): el is HTMLAnchorElement => el instanceof HTMLAnchorElement);
    expect(link).toBeDefined();
    expect(link?.download).toBe("发票附件_FP2026001.pdf");
    expect(clickSpy).toHaveBeenCalled();
    expect(URL.revokeObjectURL).toHaveBeenCalledWith("blob:mock-url");
    expect(toast.success).toHaveBeenCalledWith("附件已开始下载");
  });
});

describe("浏览器打印", () => {
  it("打开 PDF 后自动调用 print", async () => {
    vi.mocked(invoiceApi.getAttachment).mockResolvedValue(
      new Blob(["%PDF-1.4"], { type: "application/pdf" }),
    );
    const printMock = vi.fn();
    const focusMock = vi.fn();
    const { openSpy } = mockPopupWindow({ print: printMock, focus: focusMock });

    renderDialog(makeInvoice());
    fireEvent.click(screen.getByRole("button", { name: /打印/ }));

    await waitFor(() => expect(openSpy).toHaveBeenCalledWith("blob:mock-url", "_blank"));
    const popup = openSpy.mock.results[0].value as unknown as { onload: (() => void) | null };
    popup.onload?.();

    expect(focusMock).toHaveBeenCalled();
    expect(printMock).toHaveBeenCalled();
    expect(URL.revokeObjectURL).toHaveBeenCalledWith("blob:mock-url");
  });

  it("print 抛错时提示手动打印", async () => {
    vi.mocked(invoiceApi.getAttachment).mockResolvedValue(
      new Blob(["%PDF-1.4"], { type: "application/pdf" }),
    );
    const printMock = vi.fn(() => {
      throw new Error("print denied");
    });
    const { openSpy } = mockPopupWindow({ print: printMock });

    renderDialog(makeInvoice());
    fireEvent.click(screen.getByRole("button", { name: /打印/ }));

    await waitFor(() => expect(openSpy).toHaveBeenCalled());
    const popup = openSpy.mock.results[0].value as unknown as { onload: (() => void) | null };
    popup.onload?.();

    expect(printMock).toHaveBeenCalled();
    expect(toast.error).toHaveBeenCalledWith("自动打印失败，请在新窗口中手动打印");
  });
});
