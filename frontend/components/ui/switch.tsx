"use client";

import * as React from "react";

import { cn } from "@/lib/utils";

export interface SwitchProps extends React.ButtonHTMLAttributes<HTMLButtonElement> {
  checked?: boolean;
  onCheckedChange?: (checked: boolean) => void;
}

export const Switch = React.forwardRef<HTMLButtonElement, SwitchProps>(
  ({ checked = false, disabled, className, onCheckedChange, ...props }, ref) => {
    const handleClick = (event: React.MouseEvent<HTMLButtonElement>) => {
      if (disabled) {
        event.preventDefault();
        return;
      }
      onCheckedChange?.(!checked);
    };

    return (
      <button
        type="button"
        role="switch"
        aria-checked={checked}
        data-state={checked ? "checked" : "unchecked"}
        data-disabled={disabled ? "true" : undefined}
        ref={ref}
        onClick={handleClick}
        disabled={disabled}
        className={cn(
          "relative inline-flex h-6 w-11 shrink-0 cursor-pointer rounded-full border border-input transition-colors",
          disabled
            ? "cursor-not-allowed bg-muted"
            : checked
              ? "bg-primary"
              : "bg-muted",
          className,
        )}
        {...props}
      >
        <span
          aria-hidden="true"
          className={cn(
            "pointer-events-none absolute top-1/2 h-5 w-5 -translate-y-1/2 rounded-full bg-background shadow transition-transform",
            checked ? "translate-x-5" : "translate-x-1",
          )}
        />
      </button>
    );
  },
);

Switch.displayName = "Switch";

