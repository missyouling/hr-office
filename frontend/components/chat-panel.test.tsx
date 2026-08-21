import { fireEvent, render, screen } from "@testing-library/react";
import { beforeEach, describe, expect, test, vi } from "vitest";
import { ChatPanel } from "@/components/chat-panel";

// jsdom 未实现 scrollIntoView，ChatPanel 挂载时会调用，需补齐
Element.prototype.scrollIntoView = vi.fn();

// ========== 依赖 Mock（避免网络请求与复杂子组件） ==========

vi.mock("@/lib/api", () => ({
  chatKnowledgeStream: vi.fn(),
  fetchSessions: vi.fn().mockResolvedValue([]),
  deleteChatSession: vi.fn().mockResolvedValue(undefined),
  submitFeedback: vi.fn().mockResolvedValue(undefined),
}));

vi.mock("@/lib/api-knowledge", () => ({
  knowledgeApi: { list: vi.fn().mockResolvedValue({ items: [] }) },
}));

vi.mock("@/lib/auth", () => ({
  useAuth: () => ({ user: { id: 1, full_name: "测试用户" } }),
}));

vi.mock("sonner", () => ({
  toast: { success: vi.fn(), error: vi.fn(), info: vi.fn() },
}));

// Radix ScrollArea 依赖 ResizeObserver，jsdom 未实现，替换为普通容器
vi.mock("@/components/ui/scroll-area", () => ({
  ScrollArea: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
}));

describe("ChatPanel", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  test("open=false 时不渲染面板", () => {
    render(<ChatPanel open={false} onOpenChange={vi.fn()} />);
    expect(screen.queryByText("AI 知识库问答")).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "关闭 AI 助手" })).not.toBeInTheDocument();
  });

  test("open=true 时渲染面板与关闭按钮", () => {
    render(<ChatPanel open onOpenChange={vi.fn()} />);
    expect(screen.getByText("AI 知识库问答")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "关闭 AI 助手" })).toBeInTheDocument();
  });

  test("点击关闭按钮调用 onOpenChange(false)", () => {
    const onOpenChange = vi.fn();
    render(<ChatPanel open onOpenChange={onOpenChange} />);
    fireEvent.click(screen.getByRole("button", { name: "关闭 AI 助手" }));
    expect(onOpenChange).toHaveBeenCalledWith(false);
  });

  test("内嵌变体位于受控容器内，不渲染全局遮罩", () => {
    const { container } = render(<ChatPanel open onOpenChange={vi.fn()} variant="embedded" />);
    expect(container.querySelector('[data-variant="embedded"]')).toHaveClass("absolute");
    expect(container.querySelector(".bg-black\\/20")).not.toBeInTheDocument();
  });

  test("默认变体保留全局悬浮面板，按 Escape 可关闭", () => {
    const onOpenChange = vi.fn();
    const { container } = render(<ChatPanel open onOpenChange={onOpenChange} />);
    expect(container.querySelector('[data-variant="floating"]')).toHaveClass("fixed");

    fireEvent.keyDown(window, { key: "Escape" });
    expect(onOpenChange).toHaveBeenCalledWith(false);
  });
});
