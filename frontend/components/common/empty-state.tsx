"use client";
import { motion } from "motion/react";
import { Card, CardContent } from "@/components/ui/card";
import { Button } from "@/components/ui/button";

interface EmptyStateProps {
  /** 空状态图标，48px 灰色底圆内展示 */
  icon?: React.ReactNode;
  /** 空状态标题 */
  title: string;
  /** 空状态描述文字 */
  description?: string;
  /** 操作按钮：label + onClick */
  action?: { label: string; onClick: () => void };
  /** 容器最小高度，默认 h-64 */
  height?: string;
}

/** 全站统一空状态组件，带 motion 弹性入场动效 */
export function EmptyState({ icon, title, description, action, height = "h-64" }: EmptyStateProps) {
  return (
    <motion.div
      initial={{ opacity: 0, scale: 0.95 }}
      animate={{ opacity: 1, scale: 1 }}
      transition={{ duration: 0.3, ease: [0.34, 1.56, 0.64, 1] }}
      className={`flex items-center justify-center ${height}`}
    >
      <Card className="border-dashed border-2 max-w-md w-full">
        <CardContent className="flex flex-col items-center gap-3 p-8 text-center">
          {icon && (
            <motion.div
              initial={{ scale: 0 }}
              animate={{ scale: 1 }}
              transition={{ delay: 0.1, type: "spring", stiffness: 300 }}
              className="h-12 w-12 rounded-full bg-muted flex items-center justify-center opacity-60"
            >
              {icon}
            </motion.div>
          )}
          <h3 className="text-lg font-semibold">{title}</h3>
          {description && <p className="text-sm text-muted-foreground">{description}</p>}
          {action && <Button onClick={action.onClick}>{action.label}</Button>}
        </CardContent>
      </Card>
    </motion.div>
  );
}
