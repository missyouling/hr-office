import { describe, expect, it } from "vitest";
import { mapTaskStatusToFileStatus, shouldRetryParsingTask } from "./upload-state";
import {
  MAX_FILE_SIZE,
  MAX_UPLOAD_FILES,
  canEditParsedFields,
  isFieldLowConfidence,
  isFieldMissing,
  isPdfFileName,
  validateUploadFiles,
} from "./validation";

function makeFile(name: string, size = 1024): File {
  return new File([new Uint8Array(size)], name, { type: "application/pdf" });
}

describe("发票上传文件校验", () => {
  it("接受合法 PDF 文件", () => {
    const file = makeFile("invoice.pdf");
    const { accepted, rejected } = validateUploadFiles([file]);
    expect(accepted).toEqual([file]);
    expect(rejected).toEqual([]);
  });

  it("扩展名校验不区分大小写", () => {
    expect(isPdfFileName("INVOICE.PDF")).toBe(true);
    expect(isPdfFileName("发票.Pdf")).toBe(true);
    expect(isPdfFileName("invoice.jpg")).toBe(false);
    expect(isPdfFileName("invoice.pdf.exe")).toBe(false);
  });

  it("拒绝非 PDF 扩展名文件", () => {
    const file = makeFile("invoice.jpg");
    const { accepted, rejected } = validateUploadFiles([file]);
    expect(accepted).toEqual([]);
    expect(rejected).toEqual([
      { name: "invoice.jpg", code: "invalid_extension", message: "仅支持 PDF 文件" },
    ]);
  });

  it("拒绝超过 20MB 的文件", () => {
    const file = makeFile("big.pdf", MAX_FILE_SIZE + 1);
    const { accepted, rejected } = validateUploadFiles([file]);
    expect(accepted).toEqual([]);
    expect(rejected[0]).toMatchObject({ code: "file_too_large" });
  });

  it("恰好 20MB 的文件可通过校验", () => {
    const file = makeFile("edge.pdf", MAX_FILE_SIZE);
    const { accepted, rejected } = validateUploadFiles([file]);
    expect(accepted).toHaveLength(1);
    expect(rejected).toEqual([]);
  });

  it("数量超过上限时只保留前 50 个，超出部分标记 too_many", () => {
    const files = Array.from({ length: MAX_UPLOAD_FILES + 3 }, (_, i) => makeFile(`f${i}.pdf`));
    const { accepted, rejected } = validateUploadFiles(files);
    expect(accepted).toHaveLength(MAX_UPLOAD_FILES);
    expect(rejected).toHaveLength(3);
    rejected.forEach((err) => expect(err.code).toBe("too_many"));
    expect(rejected[0].name).toBe("f50.pdf");
  });

  it("空文件列表返回空结果", () => {
    expect(validateUploadFiles([])).toEqual({ accepted: [], rejected: [] });
  });
});

describe("发票解析任务状态", () => {
  it("映射任务状态", () => {
    expect(mapTaskStatusToFileStatus("pending")).toBe("running");
    expect(mapTaskStatusToFileStatus("running")).toBe("running");
    expect(mapTaskStatusToFileStatus("succeeded")).toBe("succeeded");
    expect(mapTaskStatusToFileStatus("failed")).toBe("failed");
  });
  it("按 invoiceId 选择解析重试或重新上传", () => {
    expect(shouldRetryParsingTask({ invoiceId: 1 })).toBe(true);
    expect(shouldRetryParsingTask({})).toBe(false);
  });
});

describe("识别字段编辑权限边界", () => {
  it("草稿状态且具备编辑权限时可编辑", () => {
    expect(canEditParsedFields("draft", true)).toBe(true);
  });

  it("非草稿状态即使有权限也只读", () => {
    expect(canEditParsedFields("submitted", true)).toBe(false);
    expect(canEditParsedFields("approved", true)).toBe(false);
    expect(canEditParsedFields("rejected", true)).toBe(false);
    expect(canEditParsedFields("reimbursed", true)).toBe(false);
  });

  it("草稿但无编辑权限只读", () => {
    expect(canEditParsedFields("draft", false)).toBe(false);
  });

  it("未定义状态默认只读", () => {
    expect(canEditParsedFields(undefined, true)).toBe(false);
  });
});

describe("识别字段缺失与置信度高亮", () => {
  it("空串 / null / undefined 视为缺失", () => {
    expect(isFieldMissing("")).toBe(true);
    expect(isFieldMissing(null)).toBe(true);
    expect(isFieldMissing(undefined)).toBe(true);
    expect(isFieldMissing("  ")).toBe(true);
  });

  it("非空值不视为缺失", () => {
    expect(isFieldMissing("abc")).toBe(false);
    expect(isFieldMissing(0)).toBe(false);
  });

  it("confidence 低于阈值视为低置信度", () => {
    const fields = [{ key: "seller", label: "销售方", value: "某公司", confidence: 0.79 }];
    expect(isFieldLowConfidence(fields, "seller")).toBe(true);
  });

  it("confidence 等于阈值不视为低置信度", () => {
    const fields = [{ key: "seller", label: "销售方", value: "某公司", confidence: 0.8 }];
    expect(isFieldLowConfidence(fields, "seller")).toBe(false);
  });

  it("无字段明细或未提供 confidence 时不视为低置信度", () => {
    expect(isFieldLowConfidence(undefined, "seller")).toBe(false);
    expect(isFieldLowConfidence([], "seller")).toBe(false);
    expect(isFieldLowConfidence([{ key: "seller", label: "销售方" }], "seller")).toBe(false);
  });

  it("未命中字段 key 不视为低置信度", () => {
    const fields = [{ key: "amount", label: "金额", value: 100, confidence: 0.5 }];
    expect(isFieldLowConfidence(fields, "seller")).toBe(false);
  });

  it("兼容后端 field_confidence 对象中的低置信度字段", () => {
    expect(isFieldLowConfidence({ low_confidence_fields: ["seller_tax_no"] }, "seller_tax_no")).toBe(true);
    expect(isFieldLowConfidence({ low_confidence_fields: ["seller_tax_no"] }, "seller")).toBe(false);
  });
});
