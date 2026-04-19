"use client";

import React from 'react';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Textarea } from '@/components/ui/textarea';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { Checkbox } from '@/components/ui/checkbox';
import { FormFieldSchema, shouldShowField } from '@/lib/archive-schema';
import type { Document } from '@/lib/api';

interface ArchiveFormRendererProps {
  schema: FormFieldSchema[];
  data: Document;
  onChange: (data: Document) => void;
  storageLocations?: Array<Record<string, unknown>>;
  retentionPeriods?: Array<Record<string, unknown>>;
}

export function ArchiveFormRenderer({
  schema,
  data,
  onChange,
  storageLocations,
  retentionPeriods,
}: ArchiveFormRendererProps) {
  const visibleSchema = schema.filter(field => shouldShowField(field, data));

  const handleFieldChange = (fieldName: string, value: unknown) => {
    onChange({
      ...data,
      [fieldName]: value,
    });
  };

  return (
    <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
      {visibleSchema.map((field) => (
        <div
          key={field.field_name}
          className={field.field_type === 'textarea' ? 'col-span-1 md:col-span-2' : ''}
        >
          <Label className="mb-1 block">
            {field.field_label}
            {field.required && <span className="text-destructive ml-1">*</span>}
          </Label>
          <DynamicFormField
            key={field.field_name}
            field={field}
            value={((data as unknown) as Record<string, unknown>)[field.field_name]}
            onChange={(val) => handleFieldChange(field.field_name, val)}
            storageLocations={storageLocations}
            retentionPeriods={retentionPeriods}
          />
        </div>
      ))}
    </div>
  );
}

interface DynamicFormFieldProps {
  field: FormFieldSchema;
  value: unknown;
  onChange: (value: unknown) => void;
  storageLocations?: Array<Record<string, unknown>>;
  retentionPeriods?: Array<Record<string, unknown>>;
}

function DynamicFormField({
  field,
  value,
  onChange,
  storageLocations,
  retentionPeriods,
}: DynamicFormFieldProps) {
  const placeholder = field.placeholder || `请输入${field.field_label}`;
  const defaultValue = value !== undefined && value !== null ? String(value) : (field.default_value || "");

  switch (field.field_type) {
    case 'textarea':
      return (
        <Textarea
          value={defaultValue}
          onChange={(e) => onChange(e.target.value)}
          placeholder={placeholder}
          disabled={!field.editable}
          rows={3}
        />
      );
    case 'number':
      return (
        <Input
          type="number"
          value={defaultValue}
          onChange={(e) => onChange(e.target.value === '' ? null : Number(e.target.value))}
          placeholder={placeholder}
          disabled={!field.editable}
        />
      );
    case 'date':
      return (
        <Input
          type="date"
          value={defaultValue}
          onChange={(e) => onChange(e.target.value)}
          placeholder={placeholder}
          disabled={!field.editable}
        />
      );
    case 'select':
      let options = field.options || [];
      
      if (field.field_name === 'storage_location' && storageLocations) {
        options = storageLocations.map(loc => loc.name as string);
      } else if (field.field_name === 'retention_period' && retentionPeriods) {
        options = retentionPeriods.map(p => p.name as string);
      }

      return (
        <Select
          value={defaultValue}
          onValueChange={onChange}
          disabled={!field.editable}
        >
          <SelectTrigger>
            <SelectValue placeholder={placeholder} />
          </SelectTrigger>
          <SelectContent>
            {options.map((opt) => (
              <SelectItem key={opt} value={opt}>
                {opt}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      );
    case 'checkbox':
      return (
        <div className="flex items-center space-x-2 pt-2">
          <Checkbox
            id={field.field_name}
            checked={Boolean(value)}
            onCheckedChange={onChange}
            disabled={!field.editable}
          />
          <label
            htmlFor={field.field_name}
            className="text-sm font-medium leading-none peer-disabled:cursor-not-allowed peer-disabled:opacity-70"
          >
            {field.field_label}
          </label>
        </div>
      );
    case 'multiselect':
      return (
        <Input
          value={Array.isArray(value) ? value.join(', ') : (value ? String(value) : "")}
          onChange={(e) => onChange(e.target.value.split(',').map(s => s.trim()))}
          placeholder={`${placeholder} (用逗号分隔)`}
          disabled={!field.editable}
        />
      );
    default:
      return (
        <Input
          value={defaultValue}
          onChange={(e) => onChange(e.target.value)}
          placeholder={placeholder}
          disabled={!field.editable}
        />
      );
  }
}


