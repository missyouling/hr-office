"use client";

import { useCallback, useEffect, useMemo, useRef, useState, type Dispatch, type SetStateAction } from "react";

/**
 * SmartTable - 通用表格交互相应 Hook
 *
 * 统一排序、列拖拽、显隐、持久化，驱动 employee/archives/dormitory 等表格组件。
 *
 * 用法：
 * ```tsx
 * const {
 *   sortState, cycleSort, visibleColumns, setVisibleColumns, visibleColumnDefs,
 *   handleColumnDragStart, handleColumnDragOver, handleColumnDrop, handleColumnDragEnd,
 * } = useSmartTable({
 *   storageKey: "my-table-v1",
 *   defaultColumnOrder: ["name", "age", "department"],
 *   defaultVisible: ["name", "age"],
 * });
 * ```
 */

// -------------------- 类型 --------------------

export type SortDirection = "asc" | "desc";

export interface TableSortState<T extends string = string> {
  key: T | null;
  direction: SortDirection;
}

export interface ColumnConfig<T = unknown> {
  id: string;
  label: string;
  sortable?: boolean;
  width?: string;
  getValue?: (row: T) => unknown;
  renderCell?: (row: T) => React.ReactNode;
}

export interface UseSmartTableOptions<T extends string = string> {
  /** localStorage 键名，需唯一（如 "employee-table-v1"） */
  storageKey: string;
  /** 所有列的默认顺序 */
  defaultColumnOrder: T[];
  /** 默认显示的列 ID 列表 */
  defaultVisible: T[];
  /** 是否禁用持久化（仅内存） */
  disablePersistence?: boolean;
}

// -------------------- 工具函数 --------------------

const normalizeDateInput = (text: string): string | null => {
  if (!text) return null;
  const cleaned = text.trim();
  const iso = cleaned.replace(/^(\d{4})[\/-](\d{1,2})[\/-](\d{1,2})$/, (_, y, m, d) => {
    const month = m.padStart(2, "0");
    const day = d.padStart(2, "0");
    return `${y}-${month}-${day}`;
  });
  return /^\d{4}-\d{2}-\d{2}$/.test(iso) ? iso : null;
};

const toSortableValue = (value: unknown): number | string => {
  if (value === null || value === undefined) return "";
  if (value instanceof Date) return value.getTime();
  if (typeof value === "number") return value;
  const text = String(value).trim();
  if (!text) return "";
  const numeric = Number(text);
  if (!Number.isNaN(numeric)) return numeric;
  const normalized = normalizeDateInput(text);
  if (normalized) {
    const timestamp = new Date(normalized.replace(/-/g, "/")).getTime();
    if (!Number.isNaN(timestamp)) return timestamp;
  }
  return text;
};

const compareSortableValues = (a: unknown, b: unknown, direction: SortDirection): number => {
  const left = toSortableValue(a);
  const right = toSortableValue(b);
  let result = 0;
  if (typeof left === "number" && typeof right === "number") {
    result = left - right;
  } else if (typeof left === "number") {
    result = 1;
  } else if (typeof right === "number") {
    result = -1;
  } else {
    result = String(left).localeCompare(String(right), "zh-CN", { numeric: true });
  }
  return direction === "asc" ? result : -result;
};

const arraysShallowEqual = <T,>(a: T[], b: T[]): boolean => {
  if (a === b) return true;
  if (a.length !== b.length) return false;
  for (let i = 0; i < a.length; i += 1) {
    if (a[i] !== b[i]) return false;
  }
  return true;
};

const reorderList = <T,>(list: T[], source: T, target: T): T[] => {
  const next = [...list];
  const sourceIndex = next.indexOf(source);
  const targetIndex = next.indexOf(target);
  if (sourceIndex === -1 || targetIndex === -1 || sourceIndex === targetIndex) return next;
  next.splice(sourceIndex, 1);
  next.splice(targetIndex, 0, source);
  return next;
};

// -------------------- 持久化 helpers --------------------

const readLocalStorageJSON = <T,>(key: string): T | null => {
  if (typeof window === "undefined") return null;
  try {
    const raw = window.localStorage.getItem(key);
    if (!raw) return null;
    return JSON.parse(raw) as T;
  } catch {
    console.warn(`[SmartTable] 读取 ${key} 失败`);
    return null;
  }
};

const writeLocalStorageJSON = (key: string, value: unknown): void => {
  if (typeof window === "undefined") return;
  try {
    window.localStorage.setItem(key, JSON.stringify(value));
  } catch {
    console.warn(`[SmartTable] 写入 ${key} 失败`);
  }
};

// -------------------- 主 Hook --------------------

export function useSmartTable<T extends string = string>(options: UseSmartTableOptions<T>) {
  const { storageKey, defaultColumnOrder, defaultVisible, disablePersistence = false } = options;

  // ----- 排序状态 -----
  const [sortState, setSortState] = useState<TableSortState<T>>({ key: null, direction: "asc" });

  // ----- 列拖拽 -----
  const draggingColumnKeyRef = useRef<T | null>(null);

  // ----- 可见列 -----
  const [visibleColumns, setVisibleColumnsState] = useState<T[]>([]);

  // ----- 初始化（仅一次） -----
  const initRef = useRef(false);
  useEffect(() => {
    if (initRef.current) return;
    initRef.current = true;

    // 列顺序
    let order: T[] | null = null;
    if (!disablePersistence) {
      order = readLocalStorageJSON<T[]>(`${storageKey}_order`);
    }
    if (!order || order.length === 0) {
      order = [...defaultColumnOrder];
    } else {
      // 兼容：新列出现时追加到末尾
      const existing = new Set(order);
      defaultColumnOrder.forEach((col) => {
        if (!existing.has(col)) order!.push(col);
      });
    }

    // 可见性
    let visible: T[] | null = null;
    if (!disablePersistence) {
      visible = readLocalStorageJSON<T[]>(`${storageKey}_visible`);
    }
    if (!visible || visible.length === 0) {
      visible = [...defaultVisible];
    } else {
      // 过滤已移除的列
      visible = visible.filter((col) => order!.includes(col));
      if (visible.length === 0) {
        visible = [...defaultVisible];
      }
    }

    setVisibleColumnsState(order);
    setVisibleColumnsState(visible);
  }, [storageKey, defaultColumnOrder, defaultVisible, disablePersistence]);

  // ----- 写回到 localStorage -----
  const setVisibleColumns = useCallback<Dispatch<SetStateAction<T[]>>>((next) => {
    setVisibleColumnsState((prev) => {
      const nextValue = typeof next === "function" ? (next as (prev: T[]) => T[])(prev) : next;
      if (!disablePersistence && !arraysShallowEqual(prev, nextValue)) {
        writeLocalStorageJSON(`${storageKey}_visible`, nextValue);
      }
      return nextValue;
    });
  }, [storageKey, disablePersistence]);

  const setColumnOrder = useCallback((order: T[]) => {
    if (!disablePersistence) {
      writeLocalStorageJSON(`${storageKey}_order`, order);
    }
  }, [storageKey, disablePersistence]);

  // ----- 排序 -----
  const cycleSort = useCallback((columnId: T) => {
    setSortState((prev) => {
      if (prev.key !== columnId) {
        return { key: columnId, direction: "asc" };
      }
      if (prev.direction === "asc") {
        return { key: columnId, direction: "desc" };
      }
      return { key: null, direction: "asc" };
    });
  }, []);

  const applySort = useCallback(
    <Row,>(rows: Row[], getValue: (row: Row, columnId: T) => unknown): Row[] => {
      if (!sortState.key) return rows;
      return [...rows].sort((a, b) =>
        compareSortableValues(getValue(a, sortState.key!), getValue(b, sortState.key!), sortState.direction)
      );
    },
    [sortState]
  );

  // ----- 拖拽 -----
  const moveColumn = useCallback((sourceId: T, targetId: T) => {
    setVisibleColumnsState((prev) => {
      const next = reorderList(prev, sourceId, targetId);
      setColumnOrder(next);
      return next;
    });
  }, [setColumnOrder]);

  const handleColumnDragOver = useCallback((event: React.DragEvent<HTMLTableCellElement>) => {
    event.preventDefault();
    if (event.dataTransfer) {
      event.dataTransfer.dropEffect = "move";
    }
  }, []);

  const handleColumnDragStart = useCallback((event: React.DragEvent<HTMLTableCellElement>, columnId: T) => {
    draggingColumnKeyRef.current = columnId;
    if (event.dataTransfer) {
      event.dataTransfer.effectAllowed = "move";
      event.dataTransfer.setData("text/plain", columnId);
    }
  }, []);

  const handleColumnDragEnd = useCallback(() => {
    draggingColumnKeyRef.current = null;
  }, []);

  const handleColumnDrop = useCallback((event: React.DragEvent<HTMLTableCellElement>, targetId: T) => {
    event.preventDefault();
    const sourceId = draggingColumnKeyRef.current;
    if (!sourceId || sourceId === targetId) {
      draggingColumnKeyRef.current = null;
      return;
    }
    moveColumn(sourceId, targetId);
    draggingColumnKeyRef.current = null;
  }, [moveColumn]);

  // ----- 清除持久化 -----
  const clearPersistence = useCallback(() => {
    if (!disablePersistence) {
      window.localStorage.removeItem(`${storageKey}_order`);
      window.localStorage.removeItem(`${storageKey}_visible`);
    }
    setSortState({ key: null, direction: "asc" });
    setVisibleColumnsState([...defaultVisible]);
    setColumnOrder([...defaultColumnOrder]);
  }, [storageKey, defaultColumnOrder, defaultVisible, disablePersistence, setColumnOrder]);

  // ----- 返回 -----
  return {
    // 排序
    sortState,
    cycleSort,
    applySort,
    // 列可见性
    visibleColumns,
    setVisibleColumns,
    // 拖拽
    handleColumnDragStart,
    handleColumnDragOver,
    handleColumnDrop,
    handleColumnDragEnd,
    // 持久化
    clearPersistence,
  };
}