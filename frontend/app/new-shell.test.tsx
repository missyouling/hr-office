import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, test, vi } from "vitest";

import { NewShell, NewShellContentTools } from "@/app/page";

describe("NewShell", () => {
  test("通知入口在应用壳独立可达并沿用既有打开事件", () => {
    const handler = vi.fn();
    window.addEventListener("dock:open-notification", handler);
    render(<NewShell><main>应用内容</main></NewShell>);

    fireEvent.click(screen.getByRole("button", { name: "打开通知中心" }));

    expect(handler).toHaveBeenCalledOnce();
    expect(screen.getByText("应用内容")).toBeInTheDocument();
    window.removeEventListener("dock:open-notification", handler);
  });
});

describe("NewShellContentTools", () => {
  test("新壳在内容区提供受控搜索和 AI 助手入口", () => {
    const onOpenSearch = vi.fn();
    const onOpenChat = vi.fn();
    render(<NewShellContentTools onOpenSearch={onOpenSearch} onOpenChat={onOpenChat} />);

    fireEvent.click(screen.getByRole("button", { name: "打开全局搜索" }));
    fireEvent.click(screen.getByRole("button", { name: "打开 AI 助手" }));

    expect(onOpenSearch).toHaveBeenCalledOnce();
    expect(onOpenChat).toHaveBeenCalledOnce();
  });
});
