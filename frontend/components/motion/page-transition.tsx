"use client";

import { motion, type Variants, type HTMLMotionProps } from "motion/react";
import type { ReactNode } from "react";

/** 容器变体：控制子项 stagger 编排（参考 CDK DashboardMain 模式） */
const containerVariants: Variants = {
  hidden: { opacity: 0 },
  show: {
    opacity: 1,
    transition: {
      staggerChildren: 0.1,
      delayChildren: 0.05,
    },
  },
};

/** 子项变体：从下方 16px 淡入上移 */
const itemVariants: Variants = {
  hidden: { opacity: 0, y: 16 },
  show: {
    opacity: 1,
    y: 0,
    transition: {
      duration: 0.4,
      ease: [0.25, 0.1, 0.25, 1], // ease-out cubic
    },
  },
};

interface PageTransitionProps extends HTMLMotionProps<"div"> {
  children: ReactNode;
  /** 是否启用 stagger 子项动画，默认关闭（仅淡入容器） */
  stagger?: boolean;
}

/**
 * 页面入场动画包装组件
 * - 浅层使用：<PageTransition>...</PageTransition> 仅整体淡入
 * - stagger 模式：<PageTransition stagger>...</PageTransition> 容器 + 子项级联动画
 *   子项需用 <PageItem> 包裹以接收 itemVariants
 */
export function PageTransition({ children, stagger = false, ...rest }: PageTransitionProps) {
  return (
    <motion.div
      initial="hidden"
      animate="show"
      variants={stagger ? containerVariants : undefined}
      transition={!stagger ? { duration: 0.3 } : undefined}
      {...rest}
    >
      {children}
    </motion.div>
  );
}

/**
 * 子项动画包装（与 PageTransition stagger 配套使用）
 */
export function PageItem({ children, ...rest }: HTMLMotionProps<"div">) {
  return (
    <motion.div variants={itemVariants} {...rest}>
      {children}
    </motion.div>
  );
}
