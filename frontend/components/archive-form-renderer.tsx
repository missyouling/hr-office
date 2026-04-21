"use client";

import React from "react";
import { FormFieldSchema } from "@/lib/archive-schema";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Checkbox } from "@/components/ui/checkbox";
import { Label } from "@/components/ui/label";
import { cn } from "@/lib/utils";

interface ArchiveFormRendererProps {
  schema: FormFieldSchema[];
  data: Record<string, unknown>;
  onChange: (data: Record<string, unknown>) => void;
  storageLocations?: unknown[];
  retentionPeriods?: unknown[];
}

export const ArchiveFormRenderer: React.FC<ArchiveFormRendererProps> = ({
  schema,
  data,
  onChange,
  storageLocations = [],
  retentionPeriods = [],
}) => {
  const handleFieldChange = (name: string, value: unknown) => {
    onChange({
      ...data,
      [name]: value,
    });
  };

  return (
    <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
      {schema
        .filter((field) => field.visible)
        .map((field) => (
          <DynamicFormField
            key={field.name}
            field={field}
            value={data[field.name]}
            onChange={(val) => handleFieldChange(field.name, val)}
            storageLocations={storageLocations.map(loc => typeof loc === 'string' ? loc : (loc as { name?: string; value?: string }).name || (loc as { name?: string; value?: string }).value || "")}
            retentionPeriods={retentionPeriods.map(rp => typeof rp === 'string' ? rp : (rp as { name?: string; value?: string }).name || (rp as { name?: string; value?: string }).value || "")}
          />
        ))}
    </div>
  );
};

interface DynamicFormFieldProps {
  field: FormFieldSchema;
  value: unknown;
  onChange: (value: unknown) => void;
  storageLocations: string[];
  retentionPeriods: string[];
}

const DynamicFormField: React.FC<DynamicFormFieldProps> = ({
  field,
  value,
  onChange,
  storageLocations,
  retentionPeriods,
}) => {
  const id = `field-${field.name}`;

  const renderInput = () => {
    // Special handling for predefined system lists if applicable
    let options = field.options;
    if (field.name === "storage_location" && storageLocations.length > 0) {
      options = storageLocations;
    } else if (field.name === "retention_period" && retentionPeriods.length > 0) {
      options = retentionPeriods;
    }

    switch (field.type) {
      case "textarea":
        return (
          <Textarea
            id={id}
            placeholder={field.placeholder}
            value={value || ""}
            onChange={(e) => onChange(e.target.value)}
            disabled={!field.editable}
            className="min-h-[100px]"
          />
        );

      case "number":
        return (
          <Input
            id={id}
            type="number"
            placeholder={field.placeholder}
            value={value || ""}
            onChange={(e) => onChange(e.target.value === "" ? null : Number(e.target.value))}
            disabled={!field.editable}
          />
        );

      case "date":
        return (
          <Input
            id={id}
            type="date"
            value={value || ""}
            onChange={(e) => onChange(e.target.value)}
            disabled={!field.editable}
          />
        );

      case "select":
        return (
          <Select
            value={value || ""}
            onValueChange={onChange}
            disabled={!field.editable}
          >
            <SelectTrigger id={id}>
              <SelectValue placeholder={field.placeholder || "请选择"} />
            </SelectTrigger>
            <SelectContent>
              {options?.map((opt) => (
                <SelectItem key={opt} value={opt}>
                  {opt}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        );

      case "checkbox":
        return (
          <div className="flex items-center space-x-2 pt-2">
            <Checkbox
              id={id}
              checked={!!value}
              onCheckedChange={onChange}
              disabled={!field.editable}
            />
            <label
              htmlFor={id}
              className="text-sm font-medium leading-none peer-disabled:cursor-not-allowed peer-disabled:opacity-70"
            >
              {field.label}
            </label>
          </div>
        );

      case "multiselect":
        return (
          <Input
            id={id}
            placeholder={field.placeholder || "多个选项以逗号分隔"}
            value={value || ""}
            onChange={(e) => onChange(e.target.value)}
            disabled={!field.editable}
          />
        );

      case "text":
      default:
        return (
          <Input
            id={id}
            type="text"
            placeholder={field.placeholder}
            value={value || ""}
            onChange={(e) => onChange(e.target.value)}
            disabled={!field.editable}
          />
        );
    }
  };

  return (
    <div className={cn("space-y-2", field.type === "textarea" ? "md:col-span-2" : "md:col-span-1")}>
      {field.type !== "checkbox" && (
        <Label htmlFor={id} className="text-sm font-semibold">
          {field.label}
          {field.required && <span className="text-destructive ml-1">*</span>}
        </Label>
      )}
      {renderInput()}
      {field.helpText && (
        <p className="text-xs text-muted-foreground">{field.helpText}</p>
      )}
    </div>
  );
};
