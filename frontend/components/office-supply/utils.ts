// 办公劳保模块纯工具函数

/**
 * 将数字格式化为人民币字符串
 * @param n 金额
 * @returns 例如 "¥1,234.56"
 */
export function formatCurrency(n: number | string | undefined): string {
  const val = typeof n === "string" ? parseFloat(n) : typeof n === "number" ? n : 0;
  if (Number.isNaN(val)) return "¥0.00";
  return `¥${val.toLocaleString("zh-CN", { minimumFractionDigits: 2, maximumFractionDigits: 2 })}`;
}

/**
 * 将数字金额转换为大写人民币
 * @param n 金额
 * @returns 例如 "壹仟贰佰叁拾肆元伍角陆分"
 */
export function amountToCn(n: number | string | undefined): string {
  const val = typeof n === "string" ? parseFloat(n) : typeof n === "number" ? n : 0;
  if (Number.isNaN(val) || val === 0) return "零元整";
  const cnNums = ["零", "壹", "贰", "叁", "肆", "伍", "陆", "柒", "捌", "玖"];
  const cnUnits = ["", "拾", "佰", "仟"];
  const cnBigUnits = ["", "万", "亿", "万亿"];
  const integerPart = Math.floor(Math.abs(val));
  const decimalPart = Math.round((Math.abs(val) - integerPart) * 100);

  function intToCn(num: number): string {
    if (num === 0) return "";
    let str = "";
    let zero = false;
    const groups: string[] = [];
    let temp = num;
    while (temp > 0) {
      groups.push(String(temp % 10000).padStart(4, "0"));
      temp = Math.floor(temp / 10000);
    }
    for (let gi = groups.length - 1; gi >= 0; gi--) {
      const g = groups[gi];
      let gStr = "";
      let hasValue = false;
      for (let i = 0; i < 4; i++) {
        const d = parseInt(g[i], 10);
        if (d !== 0) {
          if (zero) gStr += cnNums[0];
          gStr += cnNums[d] + cnUnits[3 - i];
          zero = false;
          hasValue = true;
        } else {
          zero = true;
        }
      }
      if (hasValue) {
        str += gStr + cnBigUnits[gi];
      } else if (gi > 0 && str.length > 0 && !str.endsWith(cnNums[0])) {
        str += cnNums[0];
      }
    }
    return str;
  }

  let result = "";
  if (integerPart > 0) {
    result += intToCn(integerPart) + "元";
  } else {
    result += "零元";
  }
  const jiao = Math.floor(decimalPart / 10);
  const fen = decimalPart % 10;
  if (jiao === 0 && fen === 0) {
    result += "整";
  } else {
    if (jiao > 0) result += cnNums[jiao] + "角";
    if (fen > 0) result += cnNums[fen] + "分";
  }
  return val < 0 ? "负" + result : result;
}

/**
 * 导出 CSV 文件
 * @param filename 文件名
 * @param rows 二维数组数据（已处理表头）
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
 * 格式化日期为 YYYY-MM-DD
 */
export function formatDate(d: Date | string | number): string {
  const date = typeof d === "string" || typeof d === "number" ? new Date(d) : d;
  const y = date.getFullYear();
  const m = String(date.getMonth() + 1).padStart(2, "0");
  const day = String(date.getDate()).padStart(2, "0");
  return `${y}-${m}-${day}`;
}

/**
 * 返回今天的 YYYY-MM-DD
 */
export function todayStr(): string {
  return formatDate(new Date());
}

/**
 * 格式化采购单日期范围显示
 */
export function formatPurchaseDate(dateStr: string | undefined): string {
  if (!dateStr) return "-";
  if (dateStr.includes("~")) {
    const [from, to] = dateStr.split("~");
    return `${from} ~ ${to}`;
  }
  return dateStr;
}

/**
 * 简化日期显示
 */
export function formatShortDate(dateStr: string | undefined): string {
  if (!dateStr) return "-";
  const d = dateStr.split("T")[0];
  return d.slice(5);
}
