import { clsx, type ClassValue } from "clsx"
import { twMerge } from "tailwind-merge"

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs))
}

interface FormatDateOptions {
  includeTime?: boolean;
}

export function formatDisplayDate(
  value?: string | number | Date | null,
  options?: FormatDateOptions,
): string {
  if (value === null || value === undefined || value === "") {
    return "-";
  }

  const { includeTime = false } = options ?? {};

  const normalizeDatePart = (year: string | number, month: string | number, day: string | number) => {
    const y = Number(year);
    const m = Number(month);
    const d = Number(day);
    if (!Number.isFinite(y) || !Number.isFinite(m) || !Number.isFinite(d)) {
      return null;
    }
    return `${y}-${m}-${d}`;
  };

  if (typeof value === "string") {
    const trimmed = value.trim();
    if (!trimmed) {
      return "-";
    }

    const simpleDateMatch = trimmed.match(/^(\d{4})-(\d{1,2})-(\d{1,2})$/);
    if (simpleDateMatch) {
      const [, y, m, d] = simpleDateMatch;
      return normalizeDatePart(y, m, d) ?? "-";
    }

    const dateTimeMatch = trimmed.match(/^(\d{4})-(\d{1,2})-(\d{1,2})(?:[ T](\d{1,2}):(\d{1,2})(?::(\d{1,2}))?)?/);
    if (dateTimeMatch) {
      const [, y, m, d, hh, mm, ss] = dateTimeMatch;
      const base = normalizeDatePart(y, m, d);
      if (!base) {
        return "-";
      }
      if (!includeTime || hh === undefined || mm === undefined) {
        return base;
      }
      const hours = Number(hh);
      const minutes = Number(mm);
      const seconds = Number(ss ?? "0");
      if ([hours, minutes, seconds].some((component) => Number.isNaN(component))) {
        return base;
      }
      const time = `${String(hours).padStart(2, "0")}:${String(minutes).padStart(2, "0")}:${String(seconds).padStart(2, "0")}`;
      return `${base} ${time}`;
    }
  }

  const source = value instanceof Date ? value : new Date(value);
  if (Number.isNaN(source.getTime())) {
    return "-";
  }
  const base = `${source.getFullYear()}-${source.getMonth() + 1}-${source.getDate()}`;
  if (!includeTime) {
    return base;
  }
  const hours = String(source.getHours()).padStart(2, "0");
  const minutes = String(source.getMinutes()).padStart(2, "0");
  const seconds = String(source.getSeconds()).padStart(2, "0");
  return `${base} ${hours}:${minutes}:${seconds}`;
}
