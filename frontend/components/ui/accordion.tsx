"use client";

import * as React from "react";
import { ChevronDown } from "lucide-react";

import { cn } from "@/lib/utils";

type AccordionContextValue = {
  type: "single" | "multiple";
};

const AccordionContext = React.createContext<AccordionContextValue | null>(null);

type AccordionItemContextValue = {
  open: boolean;
  toggle: () => void;
};

const AccordionItemContext = React.createContext<AccordionItemContextValue | null>(null);

type AccordionProps = React.HTMLAttributes<HTMLDivElement> & {
  type?: "single" | "multiple";
};

const Accordion = React.forwardRef<HTMLDivElement, AccordionProps>(({ className, type = "single", ...props }, ref) => (
  <AccordionContext.Provider value={{ type }}>
    <div ref={ref} className={cn(className)} {...props} />
  </AccordionContext.Provider>
));
Accordion.displayName = "Accordion";

type AccordionItemProps = React.HTMLAttributes<HTMLDivElement> & {
  value: string;
  defaultOpen?: boolean;
};

const AccordionItem = React.forwardRef<HTMLDivElement, AccordionItemProps>(({ className, defaultOpen = false, ...props }, ref) => {
  const [open, setOpen] = React.useState(defaultOpen);
  const toggle = React.useCallback(() => setOpen((prev) => !prev), []);
  return (
    <AccordionItemContext.Provider value={{ open, toggle }}>
      <div ref={ref} data-state={open ? "open" : "closed"} className={cn("border border-border/50 rounded-xl", className)} {...props} />
    </AccordionItemContext.Provider>
  );
});
AccordionItem.displayName = "AccordionItem";

const AccordionTrigger = React.forwardRef<
  HTMLButtonElement,
  React.ButtonHTMLAttributes<HTMLButtonElement>
>(({ className, children, ...props }, ref) => {
  const ctx = React.useContext(AccordionItemContext);
  if (!ctx) throw new Error("AccordionTrigger must be used within AccordionItem");
  return (
    <button
      ref={ref}
      type="button"
      className={cn(
        "flex w-full items-center justify-between rounded-t-xl px-4 py-3 text-left text-sm font-semibold transition hover:bg-muted/50",
        className,
      )}
      data-state={ctx.open ? "open" : "closed"}
      onClick={(event) => {
        ctx.toggle();
        props.onClick?.(event);
      }}
      {...props}
    >
      {children}
      <ChevronDown className="h-4 w-4 shrink-0 transition-transform duration-200 data-[state=open]:rotate-180" data-state={ctx.open ? "open" : "closed"} />
    </button>
  );
});
AccordionTrigger.displayName = "AccordionTrigger";

const AccordionContent = React.forwardRef<HTMLDivElement, React.HTMLAttributes<HTMLDivElement>>(({ className, children, ...props }, ref) => {
  const ctx = React.useContext(AccordionItemContext);
  if (!ctx) throw new Error("AccordionContent must be used within AccordionItem");
  if (!ctx.open) return null;
  return (
    <div ref={ref} className={cn("px-4 pb-4 text-sm", className)} {...props}>
      {children}
    </div>
  );
});
AccordionContent.displayName = "AccordionContent";

export { Accordion, AccordionItem, AccordionTrigger, AccordionContent };
