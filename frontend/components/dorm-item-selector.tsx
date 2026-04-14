import * as React from "react";
import { Accordion, AccordionContent, AccordionItem, AccordionTrigger } from "@/components/ui/accordion";
import { Checkbox } from "@/components/ui/checkbox";
import { Input } from "@/components/ui/input";
import { cn } from "@/lib/utils";

export type DormItemSelectorValue = Record<
  string,
  {
    checked: boolean;
    count: number;
  }
>;

export type DormItemSelectorProps = {
  categories: {
    name: string;
    items: { key: string; label: string; unit: string }[];
  }[];
  value: DormItemSelectorValue;
  onChange: (nextValue: DormItemSelectorValue) => void;
  className?: string;
};

export const DormItemSelector: React.FC<DormItemSelectorProps> = ({ categories, value, onChange, className }) => {
  const handleToggle = (key: string, checked: boolean) => {
    const next: DormItemSelectorValue = {
      ...value,
      [key]: {
        checked,
        count: checked ? Math.max(1, value[key]?.count ?? 1) : value[key]?.count ?? 1,
      },
    };
    onChange(next);
  };

  const handleCountChange = (key: string, raw: string) => {
    const parsed = Number.parseInt(raw, 10);
    const nextCount = Number.isFinite(parsed) && parsed >= 1 ? parsed : 1;
    const next: DormItemSelectorValue = {
      ...value,
      [key]: {
        checked: true,
        count: nextCount,
      },
    };
    onChange(next);
  };

  return (
    <Accordion
      type="multiple"
      className={cn("grid grid-cols-2 gap-4 md:grid-cols-3 xl:grid-cols-4 max-[640px]:grid-cols-2 max-[480px]:grid-cols-1", className)}
    >
      {categories.map((category) => (
        <AccordionItem key={category.name} value={category.name} className="border-none">
          <div className="w-full rounded-2xl border bg-card/70 shadow-sm">
            <AccordionTrigger className="px-4 py-3 text-sm font-semibold">
              {category.name}
            </AccordionTrigger>
            <AccordionContent>
              <div className="max-h-[260px] overflow-y-auto pr-1">
                <div className="divide-y">
                  {category.items.map((item) => {
                    const state = value[item.key] || { checked: false, count: 1 };
                    const disabled = !state.checked;
                    return (
                      <div key={item.key} className="flex items-center justify-between gap-3 px-4 py-2">
                      <button
                        type="button"
                        className="flex flex-1 items-center gap-2 text-left"
                        onClick={() => handleToggle(item.key, !state.checked)}
                      >
                        <Checkbox checked={state.checked} onCheckedChange={(checked) => handleToggle(item.key, checked === true)} />
                        <span className="truncate text-sm text-foreground">{item.label}</span>
                      </button>
                      <div className="flex items-center gap-2">
                        <Input
                          className="h-9 w-16 text-center text-sm"
                          type="number"
                          min={1}
                          step={1}
                          inputMode="numeric"
                          disabled={disabled}
                          value={disabled ? "" : state.count}
                          onChange={(event) => handleCountChange(item.key, event.target.value)}
                        />
                        <span className="text-xs text-muted-foreground">{item.unit}</span>
                      </div>
                    </div>
                    );
                  })}
                </div>
              </div>
            </AccordionContent>
          </div>
        </AccordionItem>
      ))}
    </Accordion>
  );
};

export const stringifyDormItems = (
  value: DormItemSelectorValue,
  categories: DormItemSelectorProps["categories"],
) => {
  const labelMap = new Map<string, { label: string; unit: string }>();
  categories.forEach((category) => {
    category.items.forEach((item) => labelMap.set(item.key, { label: item.label, unit: item.unit }));
  });
  return Object.entries(value)
    .filter(([, record]) => record.checked && record.count >= 1)
    .map(([key, record]) => {
      const meta = labelMap.get(key);
      return meta ? `${meta.label}${record.count}${meta.unit}` : "";
    })
    .filter(Boolean)
    .join(" | ");
};
