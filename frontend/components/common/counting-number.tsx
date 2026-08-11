"use client";

import { useEffect, useState } from "react";
import { useMotionValue, useTransform, animate } from "motion/react";

interface CountingNumberProps {
  value: number;
  /** 动画时长（秒），默认 1.2 */
  duration?: number;
  /** 小数位数，默认 0（整数） */
  decimals?: number;
  /** 前缀字符串（如 ¥、+） */
  prefix?: string;
  /** 后缀字符串（如 张、次） */
  suffix?: string;
  className?: string;
}

/**
 * 数字滚动动画组件
 * 当 value 变化时从旧值平滑滚动到新值，首次进入从 0 滚动到 value。
 * 自动添加千位分隔符（逗号）。
 */
export function CountingNumber({
  value,
  duration = 1.2,
  decimals = 0,
  prefix = "",
  suffix = "",
  className,
}: CountingNumberProps) {
  const count = useMotionValue(0);
  const rounded = useTransform(count, (latest: number) => {
    return (
      prefix +
      latest.toFixed(decimals).replace(/\B(?=(\d{3})+(?!\d))/g, ",") +
      suffix
    );
  });

  // 初始值：动画启动前短暂展示的占位数值
  const initialFormatted =
    decimals > 0 ? "0." + "0".repeat(decimals) : "0";
  const [displayValue, setDisplayValue] = useState(
    prefix + initialFormatted + suffix
  );

  useEffect(() => {
    const controls = animate(count, value, {
      duration,
      ease: "easeOut",
    });
    const unsubscribe = rounded.on("change", (v: string) =>
      setDisplayValue(v)
    );
    return () => {
      controls.stop();
      unsubscribe();
    };
  }, [value, duration, count, rounded]);

  return <span className={className}>{displayValue}</span>;
}
