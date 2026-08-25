import { fireEvent, render, screen } from "@testing-library/react";
import { beforeEach, describe, expect, test, vi } from "vitest";
import { KnowledgeQaPage } from "@/components/knowledge/knowledge-qa-page";
import { VIEW_IDS } from "@/lib/view-mapping";

const mocks = vi.hoisted(() => ({
  authState: { user: null as null | { role?: string; username?: string } },
}));

vi.mock("@/lib/supabase/auth-context", () => ({ useAuth: () => ({ user: mocks.authState.user }) }));
// ChatPanel 依赖较重（SSE/会话列表/知识库选择器），此处只验证装配契约：page 变体挂载
vi.mock("@/components/chat-panel", () => ({
  ChatPanel: ({ variant }: { variant?: string }) => <div data-testid="chat-panel" data-variant={variant}>问答面板</div>,
}));

describe("KnowledgeQaPage（AI 知识库问答页）", () => {
  const onViewChange = vi.fn();

  beforeEach(() => {
    vi.clearAllMocks();
    mocks.authState.user = null;
  });

  test("视图映射包含 knowledge-qa 且既有 knowledge 管理视图保留注册", () => {
    expect(VIEW_IDS).toContain("knowledge-qa");
    expect(VIEW_IDS).toContain("knowledge");
  });

  test("占满内容区高度：标题窄条 + page 变体 ChatPanel 铺满", () => {
    render(<KnowledgeQaPage onViewChange={onViewChange} />);
    expect(screen.getByText("AI 知识库问答")).toBeInTheDocument();
    const panel = screen.getByTestId("chat-panel");
    expect(panel).toHaveAttribute("data-variant", "page");
    // 容器契约：relative h-full，ChatPanel 以 absolute inset-0 铺满
    expect(panel.parentElement).toHaveClass("relative");
    expect(panel.parentElement).toHaveClass("min-h-0");
    expect(panel.parentElement).toHaveClass("flex-1");
  });

  test.each([
    { label: "role=admin", user: { role: "admin" } },
    { label: "role=super_admin", user: { role: "super_admin" } },
    { label: "username=admin 兜底", user: { username: "admin" } },
  ])("管理员兜底（$label）可见知识库管理按钮并跳转 knowledge 管理页", ({ user }) => {
    mocks.authState.user = user;
    render(<KnowledgeQaPage onViewChange={onViewChange} />);
    fireEvent.click(screen.getByRole("button", { name: "知识库管理" }));
    expect(onViewChange).toHaveBeenCalledWith("knowledge");
  });

  test("普通用户不显示知识库管理按钮，管理功能不经此入口暴露", () => {
    mocks.authState.user = { role: "viewer" };
    render(<KnowledgeQaPage onViewChange={onViewChange} />);
    expect(screen.queryByRole("button", { name: "知识库管理" })).not.toBeInTheDocument();
  });
});
