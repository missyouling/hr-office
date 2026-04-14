"use client";

import { useCallback, useRef, useState, memo } from "react";
import { PanelLeft } from "lucide-react";
import { AnimatePresence, MotionValue, motion, useMotionValue, useSpring, useTransform } from "motion/react";

import { cn } from "@/lib/utils";

type DockItem = {
  title: string;
  icon: React.ReactNode;
  href?: string;
  onClick?: () => void;
  tooltip?: string;
};

export function FloatingDock({
  items,
  desktopClassName,
  mobileClassName,
  mobileButtonClassName,
}: {
  items: DockItem[];
  desktopClassName?: string;
  mobileClassName?: string;
  mobileButtonClassName?: string;
}) {
  return (
    <>
      <FloatingDockDesktop items={items} className={desktopClassName} />
      <FloatingDockMobile items={items} className={mobileClassName} buttonClassName={mobileButtonClassName} />
    </>
  );
}

const FloatingDockMobile = memo(
  ({ items, className, buttonClassName }: { items: DockItem[]; className?: string; buttonClassName?: string }) => {
    const [open, setOpen] = useState(false);
    const toggleOpen = useCallback(() => setOpen((prev) => !prev), []);

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
                        "flex h-8 w-8 items-center justify-center rounded-full bg-muted text-foreground shadow dark:bg-neutral-900",
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
                      className={cn(
                        "flex h-8 w-8 items-center justify-center rounded-full bg-muted text-foreground shadow dark:bg-neutral-900",
                        buttonClassName,
                      )}
                    >
                      {item.icon}
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
          className={cn("flex h-8 w-8 items-center justify-center rounded-full bg-muted shadow dark:bg-neutral-800", buttonClassName)}
          aria-expanded={open}
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

const FloatingDockDesktop = memo(({ items, className }: { items: DockItem[]; className?: string }) => {
  const mouseX = useMotionValue(Infinity);
  const handleMouseMove = useCallback((event: React.MouseEvent) => {
    mouseX.set(event.pageX);
  }, [mouseX]);
  const handleMouseLeave = useCallback(() => mouseX.set(Infinity), [mouseX]);

  return (
    <motion.div
      onMouseMove={handleMouseMove}
      onMouseLeave={handleMouseLeave}
      className={cn("mx-auto hidden h-12 items-end gap-2 rounded-xl bg-[#E5E7EB] px-2 pb-2 md:flex shadow-sm dark:bg-[#292929]", className)}
    >
      {items.map((item) => (
        <IconContainer mouseX={mouseX} key={item.title} {...item} />
      ))}
    </motion.div>
  );
});
FloatingDockDesktop.displayName = "FloatingDockDesktop";

const IconContainer = memo(({ mouseX, title, icon, href, onClick }: DockItem & { mouseX: MotionValue }) => {
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
      className="relative flex aspect-square items-center justify-center rounded-full bg-muted text-foreground dark:bg-[#292929]"
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
    </motion.div>
  );

  if (href) {
    return (
      <a href={href} {...(href.startsWith("http") ? { target: "_blank", rel: "noreferrer" } : {})}>
        {content}
      </a>
    );
  }

  return (
    <button type="button" onClick={onClick}>
      {content}
    </button>
  );
});
IconContainer.displayName = "IconContainer";
