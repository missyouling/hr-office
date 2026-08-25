"use client";

import { useCallback, useRef, useState } from "react";
import { PanelLeft } from "lucide-react";
import {
  AnimatePresence,
  motion,
  useMotionValue,
  useSpring,
  useTransform,
  type MotionValue,
} from "motion/react";

import { cn } from "@/lib/utils";
import { usePrefersReducedMotion } from "@/hooks/use-prefers-reduced-motion";

type DockItem = {
  title: string;
  icon: React.ReactNode;
  href?: string;
  onClick?: () => void;
  badge?: number;
};

type FloatingDockProps = {
  items: DockItem[];
  desktopClassName?: string;
  mobileClassName?: string;
  mobileButtonClassName?: string;
  open?: boolean;
  onOpenChange?: (open: boolean) => void;
  variant?: "default" | "new";
};

/** 弹性缩放参数：鼠标距离映射区间与弹簧配置（参照 floating-dock 参考实现） */
const MAGNET_RANGE = 150;
/** 距离 -150/0/150 三点的容器尺寸映射（显式 number 元组，避免字面量类型污染 MotionValue） */
const DOCK_SIZE_RANGE: [number, number, number] = [32, 56, 32];
/** 距离 -150/0/150 三点的图标尺寸映射 */
const ICON_SIZE_RANGE: [number, number, number] = [16, 28, 16];
const SPRING_CONFIG = { mass: 0.1, stiffness: 150, damping: 12 } as const;

export function FloatingDock({ items, desktopClassName, mobileClassName, mobileButtonClassName, open: controlledOpen, onOpenChange, variant = "default" }: FloatingDockProps) {
  const prefersReducedMotion = usePrefersReducedMotion();
  const [uncontrolledOpen, setUncontrolledOpen] = useState(false);
  const open = controlledOpen ?? uncontrolledOpen;
  const toggleOpen = useCallback(() => {
    const nextOpen = !open;
    if (controlledOpen === undefined) setUncontrolledOpen(nextOpen);
    onOpenChange?.(nextOpen);
  }, [controlledOpen, onOpenChange, open]);

  return (
    <>
      <DesktopDock items={items} staticMode={prefersReducedMotion} className={desktopClassName} variant={variant} />
      <div className={cn("relative block md:hidden", mobileClassName)}>
        {open && <div className="absolute bottom-full mb-2 flex flex-col gap-1 rounded-2xl border border-border/80 bg-background p-1.5 shadow-lg">{items.map((item) => <DockAction key={item.title} item={item} />)}</div>}
        <button type="button" onClick={toggleOpen} className={cn("flex h-10 w-10 items-center justify-center rounded-full text-foreground transition-colors duration-150", mobileButtonClassName)} aria-expanded={open} aria-label="展开管理 Dock">
          <PanelLeft className="h-4 w-4" />
        </button>
      </div>
    </>
  );
}

/** 桌面控制坞：默认按鼠标距离弹性缩放；减少动态时退化为静态尺寸按钮 */
function DesktopDock({ items, staticMode, className, variant }: { items: DockItem[]; staticMode: boolean; className?: string; variant: "default" | "new" }) {
  const mouseX = useMotionValue(Infinity);
  const handleMouseMove = useCallback((event: React.MouseEvent) => mouseX.set(event.clientX), [mouseX]);
  const handleMouseLeave = useCallback(() => mouseX.set(Infinity), [mouseX]);

  return (
    <nav
      data-floating-dock
      data-motion={staticMode ? "off" : "spring"}
      onMouseMove={handleMouseMove}
      onMouseLeave={handleMouseLeave}
      className={cn("hidden h-12 items-end gap-1 rounded-2xl px-2 pb-2 md:flex", variant === "default" && "rounded-xl", className)}
      aria-label="快捷操作"
    >
      {items.map((item) =>
        staticMode
          ? <DockAction key={item.title} item={item} />
          : <DockIconContainer key={item.title} mouseX={mouseX} item={item} />,
      )}
    </nav>
  );
}

/** 单个坞图标容器：随鼠标距离弹性放大并显示悬停提示；常驻圆形底色对齐 cdk IconContainer 方案 */
function DockIconContainer({ mouseX, item }: { mouseX: MotionValue<number>; item: DockItem }) {
  const ref = useRef<HTMLDivElement>(null);
  const [hovered, setHovered] = useState(false);

  const distance = useTransform(mouseX, (value) => {
    const bounds = ref.current?.getBoundingClientRect() ?? { x: 0, width: 0 };
    return value - bounds.x - bounds.width / 2;
  });
  const size = useSpring(useTransform(distance, [-MAGNET_RANGE, 0, MAGNET_RANGE], [...DOCK_SIZE_RANGE]), SPRING_CONFIG);
  const iconSize = useSpring(useTransform(distance, [-MAGNET_RANGE, 0, MAGNET_RANGE], [...ICON_SIZE_RANGE]), SPRING_CONFIG);
  const content = (
    <motion.div
      ref={ref}
      style={{ width: size, height: size }}
      onMouseEnter={() => setHovered(true)}
      onMouseLeave={() => setHovered(false)}
      // 常驻圆形底（浅色 --muted / 深色协调），悬停仅加深一档至 --border，不再"仅悬停出现方圆角底"
      className="relative flex cursor-pointer items-center justify-center rounded-full bg-muted text-muted-foreground transition-colors duration-150 hover:bg-border hover:text-foreground"
    >
      <DockTooltip title={item.title} visible={hovered} />
      <motion.div style={{ width: iconSize, height: iconSize }} className="flex items-center justify-center [&_svg]:h-full [&_svg]:w-full">
        {item.icon}
      </motion.div>
      <BadgeDot badge={item.badge} />
    </motion.div>
  );

  if (item.href) return <a href={item.href} aria-label={item.title}>{content}</a>;
  return <button type="button" onClick={item.onClick} aria-label={item.title}>{content}</button>;
}

/** 悬停提示：跟随图标上方浮现，不拦截鼠标事件 */
function DockTooltip({ title, visible }: { title: string; visible: boolean }) {
  return (
    <AnimatePresence>
      {visible && (
        <motion.div
          initial={{ opacity: 0, y: 6, x: "-50%" }}
          animate={{ opacity: 1, y: 0, x: "-50%" }}
          exit={{ opacity: 0, y: 4, x: "-50%" }}
          transition={{ duration: 0.15 }}
          className="pointer-events-none absolute -top-9 left-1/2 z-10 whitespace-pre rounded-md border border-border bg-popover px-2 py-0.5 text-xs text-popover-foreground shadow-md"
        >
          {title}
        </motion.div>
      )}
    </AnimatePresence>
  );
}

/** 通知角标：大于 9 显示 9+ */
function BadgeDot({ badge }: { badge?: number }) {
  if (badge === undefined || badge <= 0) return null;
  return <span className="absolute -right-1 -top-1 flex h-4 min-w-4 items-center justify-center rounded-full bg-destructive px-1 text-[10px] text-destructive-foreground">{badge > 9 ? "9+" : badge}</span>;
}

/** 静态尺寸动作按钮：移动端面板与减少动态时的桌面坞共用；圆形常驻底与 DockIconContainer 视觉一致 */
function DockAction({ item }: { item: DockItem }) {
  const className = "relative flex h-9 w-9 items-center justify-center rounded-full bg-muted text-muted-foreground transition-colors duration-150 hover:bg-border hover:text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring";
  const content = <>{item.icon}<BadgeDot badge={item.badge} /></>;
  if (item.href) return <a href={item.href} className={className} aria-label={item.title}>{content}</a>;
  return <button type="button" onClick={item.onClick} className={className} aria-label={item.title}>{content}</button>;
}
