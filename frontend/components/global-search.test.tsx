import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, test, vi } from "vitest";
import { GlobalSearch } from "@/components/global-search";

const mocks = vi.hoisted(() => ({
  globalSearch: vi.fn(),
}));

vi.mock("@/lib/api", () => ({
  globalSearch: mocks.globalSearch,
}));

// 等待组件内 300ms 防抖触发
const waitDebounce = () => new Promise((resolve) => setTimeout(resolve, 350));

describe("GlobalSearch 组件", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  const openDialog = () => {
    render(<GlobalSearch onNavigate={vi.fn()} />);
    fireEvent.click(screen.getByRole("button", { name: /搜索/ }));
    return screen.getByPlaceholderText(/搜索档案、员工、宿舍/);
  };

  test("未输入状态：打开对话框显示输入关键词开始搜索", () => {
    openDialog();
    expect(screen.getByText("输入关键词开始搜索")).toBeInTheDocument();
  });

  test("加载状态：输入关键词后显示搜索中", async () => {
    // 永不 resolve 的请求，模拟加载中
    mocks.globalSearch.mockImplementation(() => new Promise(() => {}));
    const input = openDialog();

    fireEvent.change(input, { target: { value: "张三" } });
    await waitDebounce();

    expect(await screen.findByText("搜索中...")).toBeInTheDocument();
  });

  test("错误状态：请求失败时显示明确错误信息而非空结果", async () => {
    mocks.globalSearch.mockRejectedValue(new Error("服务器内部错误"));
    const input = openDialog();

    fireEvent.change(input, { target: { value: "张三" } });
    await waitDebounce();

    expect(
      await screen.findByText("搜索失败：服务器内部错误")
    ).toBeInTheDocument();
    expect(screen.queryByText("未找到相关结果")).not.toBeInTheDocument();
    expect(mocks.globalSearch).toHaveBeenCalledWith("张三", 20);
  });

  test("无结果状态：有输入但无匹配时显示未找到相关结果", async () => {
    mocks.globalSearch.mockResolvedValue({ results: [] });
    const input = openDialog();

    fireEvent.change(input, { target: { value: "不存在的关键字" } });
    await waitDebounce();

    expect(await screen.findByText("未找到相关结果")).toBeInTheDocument();
  });

  test("有结果时按模块分组渲染，点击结果触发 onNavigate 并关闭", async () => {
    const onNavigate = vi.fn();
    render(<GlobalSearch onNavigate={onNavigate} />);
    fireEvent.click(screen.getByRole("button", { name: /搜索/ }));

    mocks.globalSearch.mockResolvedValue({
      results: [
        {
          module: "档案",
          id: 1,
          title: "张三的劳动合同",
          snippet: "劳动合同内容",
          score: 0.9,
        },
        {
          module: "员工",
          id: 2,
          title: "张三",
          snippet: "部门: 技术部",
          score: 0.8,
        },
      ],
    });

    const input = screen.getByPlaceholderText(/搜索档案、员工、宿舍/);
    fireEvent.change(input, { target: { value: "张三" } });

    expect(await screen.findByText("张三的劳动合同")).toBeInTheDocument();
    expect(screen.getByText("张三")).toBeInTheDocument();
    expect(screen.getByText("找到 2 个结果")).toBeInTheDocument();

    fireEvent.click(screen.getByText("张三的劳动合同"));
    expect(onNavigate).toHaveBeenCalledWith("档案", 1);
    // 点击后对话框关闭，回到未输入状态
    await waitFor(() => {
      expect(screen.queryByText("张三的劳动合同")).not.toBeInTheDocument();
    });
  });

  test("键盘入口：Ctrl/Cmd+K 打开对话框", () => {
    render(<GlobalSearch onNavigate={vi.fn()} />);
    fireEvent.keyDown(window, { key: "k", ctrlKey: true });
    expect(
      screen.getByPlaceholderText(/搜索档案、员工、宿舍/)
    ).toBeInTheDocument();
  });

  test("受控模式隐藏默认悬浮入口，但仍由 Ctrl/Cmd+K 请求打开", () => {
    const onOpenChange = vi.fn();
    render(<GlobalSearch onNavigate={vi.fn()} open={false} onOpenChange={onOpenChange} hideTrigger />);

    expect(screen.queryByRole("button", { name: /搜索/ })).not.toBeInTheDocument();
    fireEvent.keyDown(window, { key: "k", metaKey: true });

    expect(onOpenChange).toHaveBeenCalledWith(true);
  });

  test("默认模式保留固定悬浮搜索入口", () => {
    const { container } = render(<GlobalSearch onNavigate={vi.fn()} />);
    expect(container.querySelector(".fixed.top-4.right-4")).toBeInTheDocument();
  });
});
