import { render, screen } from "@testing-library/react";
import { describe, expect, test, vi } from "vitest";
import type { ReactNode } from "react";

import { EmployeeManagement } from "@/components/employee-management";

vi.mock("@/lib/api", () => ({
  fetchEmployees: vi.fn().mockResolvedValue([]),
  fetchSocialInsuranceOptions: vi.fn().mockResolvedValue({}),
  fetchSocialInsuranceChanges: vi.fn().mockResolvedValue([]),
  fetchProvidentRecords: vi.fn().mockResolvedValue([]),
  fetchProvidentBills: vi.fn().mockResolvedValue([]),
  fetchUserPreferences: vi.fn().mockResolvedValue({}),
}));
vi.mock("@/lib/pdf", () => ({ createReportPdf: vi.fn() }));
vi.mock("@/lib/preferences", () => ({
  parseListPreference: vi.fn(() => []),
  sanitizeSortPreference: vi.fn((value) => value),
}));
vi.mock("@/lib/utils", () => ({ cn: (...classes: Array<string | false | null | undefined>) => classes.filter(Boolean).join(" ") }));
vi.mock("@/components/auth/RequirePermission", () => ({ RequirePermission: ({ children }: { children: ReactNode }) => <>{children}</> }));
vi.mock("@/components/common/data-table-wrapper", () => ({ DataTableWrapper: ({ children }: { children: ReactNode }) => <div>{children}</div> }));
vi.mock("@/components/ui/scroll-area", () => ({
  ScrollArea: ({ children }: { children: ReactNode }) => <div>{children}</div>,
  ScrollBar: () => null,
}));
vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

vi.mock("@/components/motion/page-transition", () => ({
  PageTransition: ({ children }: { children: ReactNode }) => <div>{children}</div>,
}));
vi.mock("@/lib/auth", () => ({
  useAuth: () => ({ token: "token", isAuthenticated: true, isLoading: false }),
}));
vi.mock("@/lib/supabase/auth-context", () => ({
  useAuth: () => ({ token: "token", isAuthenticated: true, isLoading: false }),
}));

describe("EmployeeManagement", () => {
  test("可按最小参数直接打开公积金页签", () => {
    render(<EmployeeManagement initialTab="provident" />);
    expect(screen.getByRole("tab", { name: "公积金" })).toHaveAttribute("data-state", "active");
  });
});
