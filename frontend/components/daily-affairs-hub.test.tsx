import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, test, vi } from "vitest";

import { DailyAffairsHub } from "@/components/daily-affairs-hub";

vi.mock("./archives-management", () => ({
  ArchivesManagement: () => <div data-testid="archives-management">档案管理内容</div>,
}));
vi.mock("./office-supply/OfficeSuppliesManagement", () => ({
  default: ({ onBack }: { onBack?: () => void }) => <div data-testid="office-supplies-management"><button type="button" onClick={onBack}>返回</button>办公劳保内容</div>,
}));
vi.mock("./canteen/CanteenManagement", () => ({
  default: ({ onBack }: { onBack?: () => void }) => <div data-testid="canteen-management"><button type="button" onClick={onBack}>返回</button>食堂内容</div>,
}));
vi.mock("./invoice/InvoiceManagement", () => ({
  default: ({ onBack }: { onBack?: () => void }) => <div data-testid="invoice-management"><button type="button" onClick={onBack}>返回</button>发票内容</div>,
}));

describe("DailyAffairsHub", () => {
  test("旧壳默认显示卡片墙，新壳精确入口可直接打开对应子功能", () => {
    const { rerender } = render(<DailyAffairsHub />);
    expect(screen.getByRole("heading", { name: "日常事务" })).toBeVisible();
    expect(screen.getByRole("heading", { name: "发票管理" })).toBeVisible();
    expect(screen.queryByRole("heading", { name: "社保业务" })).not.toBeInTheDocument();

    rerender(<DailyAffairsHub defaultModule="invoice" />);
    expect(screen.getByTestId("invoice-management")).toBeInTheDocument();
  });

  test("点击卡片后会进入对应模块，并通知外层导航", () => {
    const onNavigate = vi.fn();
    render(<DailyAffairsHub onNavigate={onNavigate} />);

    fireEvent.click(screen.getByRole("heading", { name: "档案管理" }));
    expect(onNavigate).toHaveBeenCalledWith("archives");
    expect(screen.getByTestId("archives-management")).toBeInTheDocument();
  });
});
