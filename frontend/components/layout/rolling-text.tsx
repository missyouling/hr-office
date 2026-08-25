"use client";

import { useEffect, useMemo, useState } from "react";
import { motion } from "motion/react";

import { cn } from "@/lib/utils";
import { usePrefersReducedMotion } from "@/hooks/use-prefers-reduced-motion";

/** 整轮动画播完后到下一轮的间隔 */
const LOOP_INTERVAL_MS = 6000;
/** 相邻字符的翻转错峰延迟 */
const PER_CHAR_DELAY_S = 0.06;
/** 单个字符翻转时长 */
const CHAR_DURATION_S = 0.5;

type RollingTextProps = React.ComponentProps<"span"> & {
  text: string;
};

/**
 * 逐字 rotateX 翻转的循环滚动标题（参照 animate-ui/text/rolling 复刻）。
 * 尊重系统"减少动态"偏好：开启时渲染静态标题，且始终保留 sr-only 完整可访问名称。
 */
export function RollingText({ text, className, ...props }: RollingTextProps) {
  const prefersReducedMotion = usePrefersReducedMotion();
  const [cycle, setCycle] = useState(0);

  useEffect(() => {
    if (prefersReducedMotion) return;
    const timer = window.setInterval(() => setCycle((value) => value + 1), LOOP_INTERVAL_MS);
    return () => window.clearInterval(timer);
  }, [prefersReducedMotion]);

  const characters = useMemo(() => splitCharacters(text), [text]);

  // 减少动态：静态完整标题，视觉与屏幕阅读器读到一致内容
  if (prefersReducedMotion) {
    return (
      <span data-slot="rolling-text" data-static="true" className={className} {...props}>
        {text}
      </span>
    );
  }

  return (
    <span data-slot="rolling-text" className={cn("inline-block", className)} {...props}>
      <span aria-hidden="true" className="inline-flex whitespace-pre [perspective:999999px]">
        {characters.map((char, index) => (
          <RollingCharacter key={`${cycle}-${index}`} char={char} index={index} />
        ))}
      </span>
      <span className="sr-only">{text}</span>
    </span>
  );
}

/** 单个字符：从下方绕 X 轴翻入；key 变化时重新触发入场实现 loop */
function RollingCharacter({ char, index }: { char: string; index: number }) {
  return (
    <motion.span
      className="inline-block"
      style={{ transformOrigin: "50% 100%" }}
      initial={{ rotateX: -90, opacity: 0 }}
      animate={{ rotateX: 0, opacity: 1 }}
      transition={{ duration: CHAR_DURATION_S, delay: index * PER_CHAR_DELAY_S, ease: "easeOut" }}
    >
      {char}
    </motion.span>
  );
}

/** 拆分字符并把空格替换为不换行空格，避免翻转过程中折行 */
function splitCharacters(text: string): string[] {
  return Array.from(text).map((char) => (char === " " ? "\u00A0" : char));
}
