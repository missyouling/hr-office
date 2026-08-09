"use client";

import React from "react";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { TableColumnSchema } from "@/lib/archive-schema";

import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import { Edit2, Trash2, Eye } from "lucide-react";

interface ArchiveTableRendererProps {
  schema: TableColumnSchema[];
  data: Array<Record<string, unknown> & { id: number }>;
  visibleColumns: string[];
  onEdit: (row: Record<string, unknown> & { id: number }) => void;
  onDelete: (id: number) => void;
  onView?: (row: Record<string, unknown> & { id: number }) => void;
  selectedIds?: number[];
  onSelect?: (id: number) => void;
  onSelectAll?: () => void;
  selectAll?: boolean;
}

export const ArchiveTableRenderer: React.FC<ArchiveTableRendererProps> = ({
  schema,
  data,
  visibleColumns,
  onEdit,
  onDelete,
  onView,
  selectedIds,
  onSelect,
  onSelectAll,
  selectAll,
}) => {
  const columns = schema.filter((col) => visibleColumns.includes(col.key));

  return (
    <div className="rounded-md border">
      <Table>
        <TableHeader>
          <TableRow>
            {(onSelect || onSelectAll) && (
              <TableHead className="w-[40px]">
                <Checkbox
                  checked={selectAll}
                  onCheckedChange={() => onSelectAll?.()}
                />
              </TableHead>
            )}
            {columns.map((col) => (
              <TableHead key={col.key} className="whitespace-nowrap">
                {col.label}
              </TableHead>
            ))}
            <TableHead className="text-right">操作</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {data.length > 0 ? (
            data.map((row) => (
              <TableRow
                key={row.id}
                className={selectedIds?.includes(row.id) ? "bg-muted/50" : ""}
              >
                {(onSelect || onSelectAll) && (
                  <TableCell>
                    <Checkbox
                      checked={selectedIds?.includes(row.id)}
                      onCheckedChange={() => onSelect?.(row.id)}
                    />
                  </TableCell>
                )}
                {columns.map((col) => (
                  <TableCell key={col.key}>
                    {renderCellValue(row[col.key], col.type)}
                  </TableCell>
                ))}
                <TableCell className="text-right">
                  <div className="flex justify-end gap-2">
                    {onView && (
                      <Button
                        variant="ghost"
                        size="icon"
                        onClick={() => onView(row)}
                      >
                        <Eye className="h-4 w-4" />
                      </Button>
                    )}
                    <Button
                      variant="ghost"
                      size="icon"
                      onClick={() => onEdit(row)}
                    >
                      <Edit2 className="h-4 w-4" />
                    </Button>
                    <Button
                      variant="ghost"
                      size="icon"
                      onClick={() => onDelete(row.id)}
                    >
                      <Trash2 className="h-4 w-4" />
                    </Button>
                  </div>
                </TableCell>
              </TableRow>
            ))
          ) : (
            <TableRow>
              <TableCell colSpan={columns.length + (onSelect ? 2 : 1)} className="h-24 text-center">
                暂无数据
              </TableCell>
            </TableRow>
          )}
        </TableBody>
      </Table>
    </div>
  );
};

function renderCellValue(value: unknown, type: string) {
  if (value === null || value === undefined) return "-";

  if (type === "date") {
    try {
      return new Date(String(value)).toLocaleDateString("zh-CN");
    } catch {
      return String(value);
    }
  }

  if (typeof value === "boolean") {
    return value ? "是" : "否";
  }

  if (Array.isArray(value)) {
    return value.join(", ");
  }

  return String(value);
}
