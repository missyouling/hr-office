import { jsPDF } from "jspdf";
import { ensureFont, REGISTERED_FONT_NAME } from "./font";

type PdfOrientation = "portrait" | "landscape" | "auto";

type CreateReportPdfOptions = {
  title: string;
  watermark?: string;
  columns: string[];
  rows: string[][];
  generatedAt?: string;
  orientation?: PdfOrientation;
};

const PAGE_MARGIN = 15;
const BASE_HEADER_FONT_SIZE = 16;
const BASE_META_FONT_SIZE = 9;
const BASE_TEXT_FONT_SIZE = 10;
const BASE_HEADER_GAP = 8;
const BASE_LINE_HEIGHT = 5;
const PORTRAIT_PAGE_WIDTH = 210; // A4 portrait width (mm)

type JsPdfWithGState = jsPDF & {
  saveGraphicsState?: () => void;
  restoreGraphicsState?: () => void;
  setGState?: (state: unknown) => void;
  GState?: new (config: Record<string, unknown>) => unknown;
};

const pickOrientation = (requested: PdfOrientation | undefined, columnCount: number) => {
  if (requested && requested !== "auto") {
    return requested;
  }
  const portraitAvailable = PORTRAIT_PAGE_WIDTH - PAGE_MARGIN * 2;
  const estimatedMinWidth = columnCount * 22; // 估算每列最小 22mm
  if (columnCount >= 8 || estimatedMinWidth > portraitAvailable) {
    return "landscape";
  }
  return "portrait";
};

const deriveTypography = (
  orientation: "portrait" | "landscape",
  columnCount: number,
  rowCount: number,
  availableWidth: number,
) => {
  let textFontSize = orientation === "landscape" ? 9 : BASE_TEXT_FONT_SIZE;
  if (columnCount >= 18) {
    textFontSize = 6;
  } else if (columnCount >= 14) {
    textFontSize = 7;
  } else if (columnCount >= 10) {
    textFontSize = 8;
  }
  if (rowCount >= 500) {
    textFontSize = Math.max(5.5, textFontSize - 2);
  } else if (rowCount >= 250) {
    textFontSize = Math.max(6, textFontSize - 1);
  }

  const headerFontSize = Math.max(12, orientation === "landscape" ? 14 : BASE_HEADER_FONT_SIZE - Math.max(0, columnCount - 10));
  const metaFontSize = Math.max(7, BASE_META_FONT_SIZE - Math.floor(columnCount / 6));
  const baseLineHeight = BASE_LINE_HEIGHT * (textFontSize / BASE_TEXT_FONT_SIZE);
  const lineHeight = Math.max(2.6, baseLineHeight);
  const headerGap = BASE_HEADER_GAP * (textFontSize / BASE_TEXT_FONT_SIZE);
  const columnPadding = Math.max(1, orientation === "landscape" ? 1.2 : 1.6);
  const dynamicMin = availableWidth / Math.max(columnCount, 1) - columnPadding * 2;
  const minColumnWidth = Math.max(10, Math.min(orientation === "landscape" ? 20 : 24, dynamicMin));
  const rowPadding = Math.max(1.5, Math.min(3.5, lineHeight * 0.75));

  return {
    textFontSize,
    headerFontSize,
    metaFontSize,
    lineHeight,
    headerGap,
    columnPadding,
    minColumnWidth,
    rowPadding,
  };
};

export async function createReportPdf({
  title,
  watermark,
  columns,
  rows,
  generatedAt,
  orientation = "auto",
}: CreateReportPdfOptions): Promise<Blob> {
  const finalOrientation = pickOrientation(orientation, columns.length);
  const doc = new jsPDF({ unit: "mm", format: "a4", orientation: finalOrientation });
  const pdfWithState = doc as JsPdfWithGState;
  const pageWidth = doc.internal.pageSize.getWidth();
  const pageHeight = doc.internal.pageSize.getHeight();
  const availableWidth = pageWidth - PAGE_MARGIN * 2;
  const {
    textFontSize,
    headerFontSize,
    metaFontSize,
    lineHeight,
    headerGap,
    columnPadding,
    minColumnWidth,
    rowPadding,
  } = deriveTypography(finalOrientation, columns.length, rows.length, availableWidth);
  await ensureFont(doc);
  doc.setFont(REGISTERED_FONT_NAME, "normal");
  doc.setFontSize(headerFontSize);
  doc.setTextColor(34, 34, 34);
  doc.text(title || "数据报表", pageWidth / 2, PAGE_MARGIN, { align: "center" });
  let cursorY = PAGE_MARGIN + headerGap;

  doc.setFontSize(metaFontSize);
  doc.setTextColor(100, 116, 139);
  doc.text(`生成时间：${generatedAt ?? new Date().toLocaleString()}`, PAGE_MARGIN, cursorY);
  cursorY += Math.max(5, lineHeight);
  doc.setTextColor(15, 23, 42);
  doc.setFontSize(textFontSize);

  if (watermark) {
    doc.setFontSize(finalOrientation === "landscape" ? 42 : 48);
    doc.setTextColor(226, 232, 240);
    if (typeof pdfWithState.saveGraphicsState === "function") {
      pdfWithState.saveGraphicsState();
      if (
        typeof pdfWithState.setGState === "function" &&
        typeof pdfWithState.GState === "function"
      ) {
        pdfWithState.setGState(new pdfWithState.GState({ opacity: 0.2 }));
      }
      doc.text(watermark, pageWidth / 2, pageHeight / 2, { align: "center", angle: 45 });
      if (typeof pdfWithState.restoreGraphicsState === "function") {
        pdfWithState.restoreGraphicsState();
      }
    } else {
      doc.text(watermark, pageWidth / 2, pageHeight / 2, { align: "center", angle: 45 });
    }
    doc.setTextColor(15, 23, 42);
    doc.setFontSize(textFontSize);
  }

  const computeColumnWidths = () => {
    const baseWidths = columns.map((columnName, index) => {
      const texts = [columnName, ...rows.map((row) => (row[index] ?? "-").toString())];
      let maxWidth = 0;
      texts.forEach((text) => {
        const width = doc.getTextWidth(text) + columnPadding * 2;
        maxWidth = Math.max(maxWidth, width);
      });
      return Math.max(minColumnWidth, maxWidth);
    });

    const totalBase = baseWidths.reduce((sum, width) => sum + width, 0);
    if (totalBase <= 0) {
      return new Array(columns.length).fill(availableWidth / Math.max(columns.length, 1));
    }

    let scaled = baseWidths.map((width) => (width / totalBase) * availableWidth);
    scaled = scaled.map((width) => Math.max(minColumnWidth, width));

    let scaledTotal = scaled.reduce((sum, width) => sum + width, 0);
    if (scaledTotal > availableWidth) {
      const overflow = scaledTotal - availableWidth;
      const flexibleTotal = scaled.reduce((sum, width) => sum + (width - minColumnWidth), 0);
      if (flexibleTotal > 0) {
        scaled = scaled.map((width) => {
          const flexiblePart = Math.max(0, width - minColumnWidth);
          const reduction = (overflow * flexiblePart) / flexibleTotal;
          return Math.max(minColumnWidth, width - reduction);
        });
      } else {
        const equalWidth = availableWidth / Math.max(columns.length, 1);
        scaled = scaled.map(() => equalWidth);
      }
      scaledTotal = scaled.reduce((sum, width) => sum + width, 0);
    }

    if (scaledTotal < availableWidth) {
      const remaining = availableWidth - scaledTotal;
      const increment = remaining / Math.max(columns.length, 1);
      scaled = scaled.map((width) => width + increment);
    }

    const finalTotal = scaled.reduce((sum, width) => sum + width, 0);
    const diff = availableWidth - finalTotal;
    if (Math.abs(diff) > 0.01 && scaled.length > 0) {
      scaled[scaled.length - 1] += diff;
    }

    return scaled;
  };

  const columnWidths = computeColumnWidths();

  const drawHeader = () => {
    doc.setFont(REGISTERED_FONT_NAME, "normal");
    doc.setFillColor(241, 245, 249);
    doc.setDrawColor(226, 232, 240);
    const headerHeight = lineHeight + rowPadding * 2;
    doc.rect(PAGE_MARGIN, cursorY, availableWidth, headerHeight, "F");

    let x = PAGE_MARGIN;
    columns.forEach((columnName, index) => {
      doc.rect(x, cursorY, columnWidths[index], headerHeight, "S");
      doc.text(columnName, x + columnPadding, cursorY + rowPadding + lineHeight);
      x += columnWidths[index];
    });

    cursorY += headerHeight;
  };

  const ensureSpace = (rowHeight: number) => {
    if (cursorY + rowHeight > pageHeight - PAGE_MARGIN) {
      doc.addPage();
      cursorY = PAGE_MARGIN;
      drawHeader();
    }
  };

  drawHeader();

  rows.forEach((row, rowIndex) => {
    const cellLines = columns.map((_, columnIndex) => {
      const value = (row[columnIndex] ?? "-").toString();
      const maxWidth = Math.max(columnWidths[columnIndex] - columnPadding * 2, minColumnWidth - columnPadding * 2);
      return doc.splitTextToSize(value, maxWidth);
    });

    const lineCount = Math.max(...cellLines.map((lines) => lines.length));
    const rowHeight = lineCount * lineHeight + rowPadding * 2;
    ensureSpace(rowHeight + 1);

    let x = PAGE_MARGIN;
    cellLines.forEach((lines: string[], columnIndex) => {
      doc.setDrawColor(226, 232, 240);
      doc.rect(x, cursorY, columnWidths[columnIndex], rowHeight, "S");
      lines.forEach((line: string, lineIndex: number) => {
        const textY = cursorY + rowPadding + lineIndex * lineHeight + lineHeight * 0.8;
        doc.text(line, x + columnPadding, textY);
      });
      x += columnWidths[columnIndex];
    });

    cursorY += rowHeight;

    if (rowIndex === rows.length - 1) {
      doc.setDrawColor(226, 232, 240);
      doc.line(PAGE_MARGIN, cursorY, PAGE_MARGIN + availableWidth, cursorY);
    }
  });

  return doc.output("blob");
}
