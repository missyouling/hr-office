// 弹窗尺寸四档规范（P10.2.1）
export const DIALOG_SIZES = {
  sm: "sm:max-w-[420px]", // 简单表单/确认
  md: "sm:max-w-[560px]", // 标准表单
  lg: "sm:max-w-[800px] max-h-[85vh]", // 复杂表单/详情
  full: "w-full max-w-[95vw] sm:max-w-5xl 2xl:max-w-6xl h-[90vh] p-0 flex flex-col", // 全屏大弹窗
} as const;

// 内部滚动模式（避免内容被挤压）
export const DIALOG_SCROLL_PATTERN = "flex flex-col gap-4 flex-1 min-h-0 overflow-y-auto";

// DialogContent 通用扩展基类
export const DIALOG_CONTENT_BASE = "gap-4 flex flex-col";
