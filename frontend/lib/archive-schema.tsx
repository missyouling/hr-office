import React from 'react';
import { Badge } from '@/components/ui/badge';
import type { ArchiveFieldDefinition, Document, ConditionConfig } from './api';

export interface FormFieldSchema {
  field_name: string;
  field_label: string;
  field_type: string;
  required: boolean;
  options?: string[];
  placeholder?: string;
  default_value?: string;
  condition_config?: ConditionConfig;
  sort_order: number;
  visible: boolean;
  editable: boolean;
}

export interface TableColumnSchema {
  key: string;
  label: string;
  render?: (value: unknown, record: Document) => React.ReactNode;
  sortable?: boolean;
  width?: number;
  visible: boolean;
  sort_order: number;
}

export function generateFormSchema(fields: ArchiveFieldDefinition[]): FormFieldSchema[] {
  return fields
    .filter(f => f.visible && f.editable)
    .sort((a, b) => a.sort_order - b.sort_order)
    .map(field => ({
      field_name: field.field_name,
      field_label: field.field_label,
      field_type: field.field_type,
      required: field.required,
      options: field.options ? field.options.split(',').map(opt => opt.trim()) : [],
      placeholder: field.placeholder,
      default_value: field.default_value,
      condition_config: field.condition_config,
      sort_order: field.sort_order,
      visible: field.visible,
      editable: field.editable,
    }));
}

export function generateTableSchema(fields: ArchiveFieldDefinition[]): TableColumnSchema[] {
  return fields
    .filter(f => f.visible)
    .sort((a, b) => a.sort_order - b.sort_order)
    .map(field => ({
      key: field.field_name,
      label: field.field_label,
      render: getRenderFunction(field.field_type),
      sortable: true,
      visible: field.visible,
      sort_order: field.sort_order,
    }));
}

export function getRenderFunction(fieldType: string): (value: unknown, record: Document) => React.ReactNode {
  switch (fieldType) {
    case 'number':
      return function NumberRenderer(value: unknown) {
        if (value === undefined || value === null || value === '') return '-';
        const num = Number(value);
        return isNaN(num) ? String(value) : `¥${num.toLocaleString()}`;
      };
    case 'date':
      return function DateRenderer(value: unknown) {
        if (!value) return '-';
        try {
          return new Date(value as string).toLocaleDateString('zh-CN');
        } catch {
          return String(value);
        }
      };
    case 'select':
      return function SelectRenderer(value: unknown) {
        return value ? <Badge variant="outline">{String(value)}</Badge> : '-';
      };
    case 'checkbox':
      return function CheckboxRenderer(value: unknown) {
        return value ? '是' : '否';
      };
    case 'multiselect':
      return function MultiSelectRenderer(value: unknown) {
        if (!value) return '-';
        const vals = Array.isArray(value) ? value : String(value).split(',');
        return (
          <div className="flex flex-wrap gap-1">
            {vals.map((v, i) => (
              <Badge key={i} variant="secondary">{String(v).trim()}</Badge>
            ))}
          </div>
        );
      };
    default:
      return function DefaultRenderer(value: unknown) {
        return (value === undefined || value === null || value === '') ? '-' : String(value);
      };
  }
}


export function shouldShowField(field: FormFieldSchema | ArchiveFieldDefinition, formData: Document): boolean {
  if (!field.condition_config) return true;

  const { field_name, operator, value: expectedValue } = field.condition_config;
  const formDataObj = formData as unknown as Record<string, unknown>;
  const fieldValue = formDataObj[field_name];

  switch (operator) {
    case 'equals':
      return String(fieldValue) === String(expectedValue);
    case 'contains':
      return String(fieldValue).includes(String(expectedValue));
    case 'gt':
      return Number(fieldValue) > Number(expectedValue);
    case 'lt':
      return Number(fieldValue) < Number(expectedValue);
    case 'in':
      const opts = expectedValue.split(',').map(s => s.trim());
      return opts.includes(String(fieldValue));
    case 'not_empty':
      return fieldValue !== undefined && fieldValue !== null && fieldValue !== '';
    default:
      return true;
  }
}



