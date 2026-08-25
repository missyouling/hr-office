"use client";

import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { PersonalSettings } from "@/components/personal-settings";

/**
 * 新壳"个人资料"模态对话框：复用 PersonalSettings（账户资料 / 外观偏好两个标签页）。
 * 点击头像菜单"个人资料"直接弹窗展示，关闭即返回原视图，
 * 不再切换侧栏到 personal-settings 替换模式（该视图注册保留供其他入口使用）。
 * 不传 onBack：弹窗场景无"返回"按钮，由右上角关闭图标承担退出。
 */
export function ProfileDialog({ open, onOpenChange }: { open: boolean; onOpenChange: (open: boolean) => void }) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      {/* sm:max-w-2xl 覆盖 DialogContent 默认 sm:max-w-lg；限高内部滚动，避免小屏溢出 */}
      <DialogContent className="max-h-[85dvh] overflow-y-auto sm:max-w-2xl">
        <DialogHeader>
          <DialogTitle>个人资料</DialogTitle>
          <DialogDescription>管理你的公开资料与登录安全。</DialogDescription>
        </DialogHeader>
        <PersonalSettings />
      </DialogContent>
    </Dialog>
  );
}
