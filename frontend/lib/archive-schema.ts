import { ArchiveFieldDefinition } from "./api";

export interface FormFieldSchema {
  name: string;
  label: string;
  type: "text" | "textarea" | "number" | "date" | "select" | "multiselect" | "checkbox";
  required: boolean;
  options?: string[];
  placeholder?: string;
  defaultValue?: string;
  helpText?: string;
  editable: boolean;
  visible: boolean;
  condition?: {
    field: string;
    operator: "equals" | "contains" | "gt" | "lt" | "in" | "not_empty";
    value: string;
  };
}

export interface TableColumnSchema {
  key: string;
  label: string;
  type: string;
  sortable: boolean;
}

/**
 * Generates a form schema from archive field definitions.
 */
export function generateFormSchema(fields: ArchiveFieldDefinition[]): FormFieldSchema[] {
  return fields.map((field) => ({
    name: field.field_name,
    label: field.field_label,
    type: field.field_type,
    required: field.required,
    options: field.options ? field.options.split(",").map((o) => o.trim()) : undefined,
    placeholder: field.placeholder,
    defaultValue: field.default_value,
    helpText: field.help_text,
    editable: field.editable,
    visible: field.visible,
    condition: field.condition_config
      ? {
          field: field.condition_config.field_name,
          operator: field.condition_config.operator,
          value: field.condition_config.value,
        }
      : undefined,
  }));
}

/**
 * Generates a table schema from archive field definitions.
 */
export function generateTableSchema(fields: ArchiveFieldDefinition[]): TableColumnSchema[] {
  // Always include standard fields first
  const standardColumns: TableColumnSchema[] = [
    { key: "document_code", label: "档案编号", type: "text", sortable: true },
    { key: "category_code", label: "分类", type: "text", sortable: true },
    { key: "year", label: "年度", type: "number", sortable: true },
    { key: "file_name", label: "文件名", type: "text", sortable: true },
  ];

  const dynamicColumns: TableColumnSchema[] = fields
    .filter((f) => f.visible)
    .map((field) => ({
      key: field.field_name,
      label: field.field_label,
      type: field.field_type,
      sortable: true,
    }));

  const endColumns: TableColumnSchema[] = [
    { key: "status", label: "状态", type: "text", sortable: true },
    { key: "updated_at", label: "更新时间", type: "date", sortable: true },
  ];

  return [...standardColumns, ...dynamicColumns, ...endColumns];
}

/**
 * Determines if a field should be shown based on its condition config and current form data.
 */
export function shouldShowField(field: ArchiveFieldDefinition, formData: Record<string, unknown>): boolean {
  if (!field.visible) return false;
  if (!field.condition_config) return true;

  const { field_name, operator, value } = field.condition_config;
  const targetValue = formData[field_name];

  switch (operator) {
    case "equals":
      return String(targetValue) === String(value);
    case "contains":
      return String(targetValue).includes(String(value));
    case "gt":
      return Number(targetValue) > Number(value);
    case "lt":
      return Number(targetValue) < Number(value);
    case "in":
      const options = value.split(",").map((v) => v.trim());
      return options.includes(String(targetValue));
    case "not_empty":
      return targetValue !== undefined && targetValue !== null && targetValue !== "";
    default:
      return true;
  }
}
