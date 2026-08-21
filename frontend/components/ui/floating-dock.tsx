"use client";

import { useCallback, useRef, useState, memo } from "react";
import { GripVertical, PanelLeft } from "lucide-react";
import { AnimatePresence, MotionValue, motion, useMotionValue, useSpring, useTransform } from "motion/react";

import { cn } from "@/lib/utils";
import type { DockPosition } from "@/lib/preferences";

const NEW_SHELL_DESKTOP_DOCK_MIN_LEFT = 208;

type DockItem = {
  title: string;
  icon: React.ReactNode;
  href?: string;
  onClick?: () => void;
  tooltip?: string;
  badge?: number;
};

export function FloatingDock({
  items,
  desktopClassName,
  mobileClassName,
  mobileButtonClassName,
  desktopPosition,
  onDesktopPositionChange,
  open,
  onOpenChange,
  variant = "default",
}: {
  items: DockItem[];
  desktopClassName?: string;
  mobileClassName?: string;
  mobileButtonClassName?: string;
  desktopPosition?: DockPosition;
  onDesktopPositionChange?: (position: DockPosition) => void;
  open?: boolean;
  onOpenChange?: (open: boolean) => void;
  variant?: "default" | "new";
}) {
  return (
    <>
      <FloatingDockDesktop
        items={items}
        className={desktopClassName}
        position={desktopPosition}
        onPositionChange={onDesktopPositionChange}
        variant={variant}
      />
      <FloatingDockMobile items={items} className={mobileClassName} buttonClassName={mobileButtonClassName} open={open} onOpenChange={onOpenChange} />
    </>
  );
}

const FloatingDockMobile = memo(
  ({ items, className, buttonClassName, open: controlledOpen, onOpenChange }: { items: DockItem[]; className?: string; buttonClassName?: string; open?: boolean; onOpenChange?: (open: boolean) => void }) => {
    const [uncontrolledOpen, setUncontrolledOpen] = useState(false);
    const open = controlledOpen ?? uncontrolledOpen;
    const toggleOpen = useCallback(() => {
      const nextOpen = !open;
      if (controlledOpen === undefined) setUncontrolledOpen(nextOpen);
      onOpenChange?.(nextOpen);
    }, [controlledOpen, onOpenChange, open]);

    return (
      <div className={cn("relative block md:hidden", className)}>
        <AnimatePresence>
          {open && (
            <motion.div layoutId="dock" className="absolute inset-x-0 bottom-full mb-2 flex flex-col gap-1">
              {items.map((item, idx) => (
                <motion.div
                  key={item.title}
                  initial={{ opacity: 0, y: 10 }}
                  animate={{ opacity: 1, y: 0 }}
                  exit={{ opacity: 0, y: 10, transition: { delay: idx * 0.05 } }}
                  transition={{ delay: (items.length - 1 - idx) * 0.05 }}
                >
                  {item.href ? (
                    <a
                      href={item.href}
                      className={cn(
                        "flex h-8 w-8 items-center justify-center rounded-full bg-muted text-foreground shadow",
                        buttonClassName,
                      )}
                      {...(item.href.startsWith("http") ? { target: "_blank", rel: "noreferrer" } : {})}
                    >
                      {item.icon}
                    </a>
                  ) : (
                    <button
                      type="button"
                      onClick={item.onClick}
                      aria-label={item.title}
                      className={cn(
                        "flex h-8 w-8 items-center justify-center rounded-full bg-muted text-foreground shadow relative",
                        buttonClassName,
                      )}
                    >
                      {item.icon}
                      {item.badge !== undefined && item.badge > 0 && (
                        <span className="absolute -top-1 -right-1 flex h-4 w-4 items-center justify-center rounded-full bg-red-500 text-[10px] text-white">
                          {item.badge > 9 ? "9+" : item.badge}
                        </span>
                      )}
                    </button>
                  )}
                </motion.div>
              ))}
            </motion.div>
          )}
        </AnimatePresence>
        <button
          type="button"
          onClick={toggleOpen}
          className={cn("flex h-8 w-8 items-center justify-center rounded-full bg-muted shadow", buttonClassName)}
          aria-expanded={open}
          aria-label="展开管理 Dock"
        >
          <motion.div animate={{ rotate: open ? 180 : 0 }} transition={{ duration: 0.3, ease: "easeInOut" }}>
            <PanelLeft className="h-4 w-4" />
          </motion.div>
        </button>
      </div>
    );
  },
);
FloatingDockMobile.displayName = "FloatingDockMobile";

const FloatingDockDesktop = memo(({
  items,
  className,
  position,
  onPositionChange,
  variant,
}: {
  items: DockItem[];
  className?: string;
  position?: DockPosition;
  onPositionChange?: (position: DockPosition) => void;
  variant: "default" | "new";
}) => {
  const mouseX = useMotionValue(Infinity);
  const dockRef = useRef<HTMLDivElement>(null);
  const dragRef = useRef<{ x: number; y: number; left: number; top: number } | null>(null);
  const handleMouseMove = useCallback((event: React.MouseEvent) => {
    mouseX.set(event.pageX);
  }, [mouseX]);
  const handleMouseLeave = useCallback(() => mouseX.set(Infinity), [mouseX]);
  const handlePointerDown = useCallback((event: React.PointerEvent<HTMLDivElement>) => {
    if (!position || !onPositionChange) return;
    const element = dockRef.current;
    if (!element) return;
    dragRef.current = { x: event.clientX, y: event.clientY, left: position.left, top: position.top };
    event.currentTarget.setPointerCapture(event.pointerId);
  }, [onPositionChange, position]);
  const handlePointerMove = useCallback((event: React.PointerEvent<HTMLDivElement>) => {
    const drag = dragRef.current;
    const element = dockRef.current;
    if (!drag || !element || !onPositionChange) return;
    const minLeft = variant === "new" ? NEW_SHELL_DESKTOP_DOCK_MIN_LEFT : 8;
    const maxLeft = Math.max(minLeft, window.innerWidth - element.offsetWidth - 8);
    const maxTop = Math.max(8, window.innerHeight - element.offsetHeight - 8);
    onPositionChange({
      left: Math.min(Math.max(drag.left + event.clientX - drag.x, minLeft), maxLeft),
      top: Math.min(Math.max(drag.top + event.clientY - drag.y, 8), maxTop),
    });
  }, [onPositionChange, variant]);
  const handlePointerUp = useCallback(() => { dragRef.current = null; }, []);
  // 拖动手柄：仅手柄启动拖动，按钮点击保持原行为；移动端隐藏（md:flex）
  const dragHandle = position && onPositionChange ? (
    <div
      data-dock-drag-handle
      aria-label="拖动 Dock"
      role="separator"
      title="拖动 Dock"
      onPointerDown={handlePointerDown}
      onPointerMove={handlePointerMove}
      onPointerUp={handlePointerUp}
      className="mr-1 hidden touch-none cursor-grab select-none items-center justify-center self-stretch rounded-md px-1 text-muted-foreground/70 transition-colors hover:text-foreground active:cursor-grabbing md:flex"
    >
      <GripVertical className="h-4 w-4" />
    </div>
  ) : null;

  return (
    <motion.div
      onMouseMove={handleMouseMove}
      onMouseLeave={handleMouseLeave}
      ref={dockRef}
      data-floating-dock
      style={position ? { left: variant === "new" ? Math.max(position.left, NEW_SHELL_DESKTOP_DOCK_MIN_LEFT) : position.left, top: position.top } : undefined}
      className={cn("hidden h-12 items-end gap-2 rounded-xl bg-background/70 backdrop-blur-md border border-border/60 px-2 pb-2 md:flex shadow-lg shadow-black/10 dark:shadow-white/5", variant === "new" && "rounded-2xl", position && "fixed", className)}
    >
      {dragHandle}
      {items.map((item) => (
        <IconContainer mouseX={mouseX} key={item.title} {...item} />
      ))}
    </motion.div>
  );
});
FloatingDockDesktop.displayName = "FloatingDockDesktop";

const IconContainer = memo(({ mouseX, title, icon, href, onClick, badge }: DockItem & { mouseX: MotionValue }) => {
  const ref = useRef<HTMLDivElement>(null);
  const distance = useTransform(mouseX, (value) => {
    const bounds = ref.current?.getBoundingClientRect() ?? { x: 0, width: 0 };
    return value - bounds.x - bounds.width / 2;
  });
  const widthTransform = useTransform(distance, [-150, 0, 150], [32, 56, 32]);
  const heightTransform = useTransform(distance, [-150, 0, 150], [32, 56, 32]);
  const widthIconTransform = useTransform(distance, [-150, 0, 150], [16, 28, 16]);
  const heightIconTransform = useTransform(distance, [-150, 0, 150], [16, 28, 16]);

  const width = useSpring(widthTransform, { mass: 0.1, stiffness: 150, damping: 12 });
  const height = useSpring(heightTransform, { mass: 0.1, stiffness: 150, damping: 12 });
  const widthIcon = useSpring(widthIconTransform, { mass: 0.1, stiffness: 150, damping: 12 });
  const heightIcon = useSpring(heightIconTransform, { mass: 0.1, stiffness: 150, damping: 12 });

  const [hovered, setHovered] = useState(false);

  const content = (
    <motion.div
      ref={ref}
      style={{ width, height }}
      onMouseEnter={() => setHovered(true)}
      onMouseLeave={() => setHovered(false)}
      className="relative flex aspect-square items-center justify-center rounded-full bg-muted text-foreground"
    >
      <AnimatePresence>
        {hovered && (
          <motion.div
            initial={{ opacity: 0, y: 10, x: "-50%" }}
            animate={{ opacity: 1, y: 0, x: "-50%" }}
            exit={{ opacity: 0, y: 2, x: "-50%" }}
            className="absolute -top-8 left-1/2 whitespace-nowrap rounded-md border border-border bg-background px-2 py-0.5 text-xs"
          >
            {title}
          </motion.div>
        )}
      </AnimatePresence>
      <motion.div style={{ width: widthIcon, height: heightIcon }} className="flex items-center justify-center">
        {icon}
      </motion.div>
      {badge !== undefined && badge > 0 && (
        <span className="absolute -top-1 -right-1 flex h-4 w-4 items-center justify-center rounded-full bg-red-500 text-[10px] text-white">
          {badge > 9 ? "9+" : badge}
        </span>
      )}
    </motion.div>
  );

  if (href) {
    return (
      <a href={href} {...(href.startsWith("http") ? { target: "_blank", rel: "noreferrer" } : {})} className="relative">
        {content}
      </a>
    );
  }

  return (
    <button type="button" onClick={onClick} className="relative" aria-label={title}>
      {content}
    </button>
  );
});
IconContainer.displayName = "IconContainer";
