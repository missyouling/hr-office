// 食堂管理模块纯工具函数

/**
 * 格式化金额为人民币字符串
 */
export function fmt(n: number | string | undefined): string {
  const val = typeof n === "string" ? parseFloat(n) : typeof n === "number" ? n : 0;
  if (Number.isNaN(val)) return "¥0.00";
  return `¥${val.toLocaleString("zh-CN", { minimumFractionDigits: 2, maximumFractionDigits: 2 })}`;
}

/**
 * 格式化数字为千分位
 */
export function fmtNum(n: number | string | undefined): string {
  const val = typeof n === "string" ? parseFloat(n) : typeof n === "number" ? n : 0;
  return Number.isNaN(val) ? "0" : val.toLocaleString("zh-CN");
}

/**
 * 返回今天的 YYYY-MM-DD
 */
export function todayStr(): string {
  return new Date().toISOString().slice(0, 10);
}

/**
 * 返回当前月份 YYYY-MM
 */
export function currentMonth(): string {
  return new Date().toISOString().slice(0, 7);
}

/**
 * 根据日期获取所在周的周一
 */
export function mondayOf(date: Date): string {
  const d = new Date(date);
  const day = d.getDay() || 7;
  d.setDate(d.getDate() - day + 1);
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, "0")}-${String(d.getDate()).padStart(2, "0")}`;
}

/**
 * 日期加减天数
 */
export function addDays(dateStr: string, days: number): string {
  const d = new Date(`${dateStr}T00:00:00`);
  d.setDate(d.getDate() + days);
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, "0")}-${String(d.getDate()).padStart(2, "0")}`;
}

/**
 * 读取 CSV 文件并自动检测 UTF-8 / GBK 编码
 */
export async function readCsvFile(file: File): Promise<string> {
  const buf = await file.arrayBuffer();
  try {
    return new TextDecoder("utf-8", { fatal: true }).decode(buf);
  } catch {
    try {
      return new TextDecoder("gbk").decode(buf);
    } catch {
      return new TextDecoder("utf-8", { fatal: false }).decode(buf);
    }
  }
}

/**
 * 解析简单 CSV 行（支持双引号包裹）
 */
export function parseCsvLine(line: string): string[] {
  const out: string[] = [];
  let cur = "";
  let inQ = false;
  for (let i = 0; i < line.length; i++) {
    const ch = line[i];
    if (ch === '"') {
      if (inQ && line[i + 1] === '"') {
        cur += '"';
        i++;
      } else {
        inQ = !inQ;
      }
    } else if (ch === "," && !inQ) {
      out.push(cur);
      cur = "";
    } else {
      cur += ch;
    }
  }
  out.push(cur);
  return out.map((s) => s.trim().replace(/^"|"$/g, ""));
}

/**
 * 导出 CSV
 */
export function exportCsv(filename: string, rows: (string | number)[][]): void {
  const escape = (v: string | number): string => {
    const s = String(v ?? "");
    if (s.includes(",") || s.includes('"') || s.includes("\n")) {
      return `"${s.replace(/"/g, '""')}"`;
    }
    return s;
  };
  const csv = "\uFEFF" + rows.map((row) => row.map(escape).join(",")).join("\n");
  const blob = new Blob([csv], { type: "text/csv;charset=utf-8;" });
  const url = URL.createObjectURL(blob);
  const a = document.createElement("a");
  a.href = url;
  a.download = filename;
  a.click();
  URL.revokeObjectURL(url);
}

/**
 * 计算月份天数
 */
export function daysInMonth(year: number, month: number): number {
  return new Date(year, month, 0).getDate();
}
