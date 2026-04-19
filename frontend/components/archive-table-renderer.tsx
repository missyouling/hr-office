"use client";

import React from 'react';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import { Button } from '@/components/ui/button';
import { TableColumnSchema } from '@/lib/archive-schema';
import type { Document } from '@/lib/api';

interface ArchiveTableRendererProps {
  schema: TableColumnSchema[];
  data: Document[];
  visibleColumns?: string[];
  onEdit?: (doc: Document) => void;
  onDelete?: (id: number) => void;
  onView?: (doc: Document) => void;
}

export function ArchiveTableRenderer({
  schema,
  data,
  visibleColumns,
  onEdit,
  onDelete,
  onView,
}: ArchiveTableRendererProps) {
  const columns = schema.filter(
    col => col.visible && (!visibleColumns || visibleColumns.includes(col.key))
  );

  return (
    <div className="rounded-md border">
      <Table>
        <TableHeader>
          <TableRow>
            {columns.map((col) => (
              <TableHead key={col.key}>{col.label}</TableHead>
            ))}
            <TableHead className="text-right">操作</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {data.length > 0 ? (
            data.map((row) => (
              <TableRow key={row.id}>
                {columns.map((col) => (
                  <TableCell key={col.key}>
                    {col.render ? col.render((row as Record<string, unknown>)[col.key], row) : String((row as Record<string, unknown>)[col.key] || '')}
                  </TableCell>
                ))}
                <TableCell className="text-right">
                  <div className="flex justify-end gap-2">
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={() => onView?.(row)}
                    >
                      查看
                    </Button>
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={() => onEdit?.(row)}
                    >
                      编辑
                    </Button>
                    <Button
                      variant="ghost"
                      size="sm"
                      className="text-destructive hover:text-destructive"
                      onClick={() => onDelete?.(row.id)}
                    >
                      删除
                    </Button>
                  </div>
                </TableCell>
              </TableRow>
            ))
          ) : (
            <TableRow>
              <TableCell colSpan={columns.length + 1} className="h-24 text-center">
                暂无数据
              </TableCell>
            </TableRow>
          )}
        </TableBody>
      </Table>
    </div>
  );
}

