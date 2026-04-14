# SmartTable 通用表格 Hook

> 统一 employee、archives、dormitory 等表格的排序、列拖拽、显隐与持久化。

## 位置

```
frontend/hooks/useSmartTable.ts
```

## 使用方法

```tsx
import { useSmartTable } from "@/hooks/useSmartTable";

type MyColumnId = "name" | "age" | "department" | "position";

const COLUMN_ORDER: MyColumnId[] = ["name", "age", "department", "position"];
const DEFAULT_VISIBLE: MyColumnId[] = ["name", "age", "department"];

export default function MyTable() {
  const {
    sortState,           // { key, direction }
    cycleSort,           // (columnId) => void
    applySort,          // (rows, getValue) => sortedRows
    visibleColumns,     // string[]
    setVisibleColumns,  // (prev) => void | (new) => void
    handleColumnDragStart,   // (event, columnId) => void
    handleColumnDragOver,   // (event) => void
    handleColumnDrop,     // (event, targetId) => void
    handleColumnDragEnd,   // () => void
    clearPersistence,    // () => void
  } = useSmartTable<MyColumnId>({
    storageKey: "my-table-v1",          // 唯一键名
    defaultColumnOrder: COLUMN_ORDER,  // 默认列顺序
    defaultVisible: DEFAULT_VISIBLE,   // 默认显示列
  });

  // 数据源
  const rows = useMemo(() => applySort(data, (row, col) => row[col]), [data, sortState]);

  // 表格渲染...
}
```

## Persistence Keys

| Key | 作用 |
|-----|------|
| `{storageKey}_order` | 列顺序 |
| `{storageKey}_visible` | 显隐状态 |

## 已有接入（里程碑）

- **archives-management.tsx**: 当前仍使用内联代码（见 `visibleColumns` state 块）
- **dormitory-management.tsx**: 使用独立 localStorage helpers（已接入，暂不迁移）
- **employee-management.tsx**: 使用独立 localStorage helpers（已接入，暂不迁移）

## 迁移指南

新模块或现有模块重构时：

1. 导入 `useSmartTable`
2. 删除本地 `visibleColumns`、`sortField`、`sortDirection` 状态
3. 替换拖拽 handler 调用
4. 验证 persistence keys 唯一且兼容旧数据

## 扩展

如需更多功能（如远程排序字段、分页状态），可在 hook 中添加工具函数后复用。