export type ColumnMapPreference<T extends string> = {
  order?: T[];
  visibility?: Record<T, boolean>;
};

export type ColumnListPreference<T extends string> = {
  order?: T[];
  visible?: T[];
};

export function normalizeOrder<T extends string>(order: readonly T[] | undefined | null, allowed: readonly T[]): T[] {
  const fallback = [...allowed];
  if (!order || order.length === 0) {
    return fallback;
  }
  const allowedSet = new Set(allowed);
  const visited = new Set<T>();
  const next: T[] = [];
  order.forEach((item) => {
    const typed = item as T;
    if (allowedSet.has(typed) && !visited.has(typed)) {
      next.push(typed);
      visited.add(typed);
    }
  });
  allowed.forEach((item) => {
    if (!visited.has(item as T)) {
      next.push(item as T);
      visited.add(item as T);
    }
  });
  return next;
}

export function normalizeVisibility<T extends string>(
  visibility: Record<string, boolean> | undefined | null,
  defaults: Record<T, boolean>,
): Record<T, boolean> {
  const next: Record<T, boolean> = { ...defaults };
  if (!visibility) {
    return next;
  }
  (Object.keys(defaults) as T[]).forEach((key) => {
    if (typeof visibility[key] === "boolean") {
      next[key] = visibility[key];
    }
  });
  return next;
}

export function parseColumnPreferenceValue<T extends string>(
  raw: unknown,
  allowedOrder: readonly T[],
  defaultVisibility: Record<T, boolean>,
): ColumnMapPreference<T> | null {
  if (!raw || typeof raw !== "object") {
    return null;
  }
  const { order, visibility } = raw as { order?: unknown; visibility?: unknown };
  const preference: ColumnMapPreference<T> = {};
  if (Array.isArray(order)) {
    preference.order = normalizeOrder(order as T[], allowedOrder);
  }
  if (visibility && typeof visibility === "object") {
    preference.visibility = normalizeVisibility(visibility as Record<string, boolean>, defaultVisibility);
  }
  return Object.keys(preference).length ? preference : null;
}

export function parseListPreference<T extends string>(
  raw: unknown,
  allowedOrder: readonly T[],
  defaultVisible?: readonly T[],
): ColumnListPreference<T> | null {
  if (!raw || typeof raw !== "object") {
    return null;
  }
  const payload = raw as { order?: unknown; visible?: unknown; visibility?: unknown };
  const result: ColumnListPreference<T> = {};
  if (Array.isArray(payload.order)) {
    result.order = normalizeOrder(payload.order as T[], allowedOrder);
  }
  const visibleValue = (Array.isArray(payload.visible) ? payload.visible : payload.visibility) as T[] | undefined;
  if (visibleValue && visibleValue.length > 0 && defaultVisible) {
    result.visible = normalizeOrder(visibleValue as T[], defaultVisible);
  }
  return Object.keys(result).length ? result : null;
}

export type TableSortState<T extends string> = { key: T | null; direction: "asc" | "desc" };

export function sanitizeSortPreference<T extends string>(
  raw: unknown,
  allowed: readonly T[],
  fallback: TableSortState<T>,
): TableSortState<T> {
  if (!raw || typeof raw !== "object") {
    return fallback;
  }
  const maybe = raw as Partial<TableSortState<T>>;
  const allowedSet = new Set(allowed);
  const nextKey = allowedSet.has((maybe.key as T) ?? ("" as T)) ? (maybe.key as T) : fallback.key;
  const nextDirection = maybe.direction === "desc" ? "desc" : "asc";
  if (nextKey === fallback.key && nextDirection === fallback.direction) {
    return fallback;
  }
  return { key: nextKey, direction: nextDirection };
}
