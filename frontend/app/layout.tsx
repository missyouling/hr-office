import Script from "next/script";
import type { Metadata } from "next";
import "./globals.css";
import { AppProviders } from "./providers";

// Force dynamic rendering for all pages to avoid SSR issues with client-side auth
export const dynamic = 'force-dynamic';
export const dynamicParams = true;

// Use system fonts to avoid Google Fonts network issues during Docker build
const fontVariables = {
  "--app-font-sans": "Inter, 'PingFang SC', 'Hiragino Sans GB', 'Microsoft YaHei', 'Source Han Sans SC', 'Noto Sans SC', 'Segoe UI', system-ui, sans-serif",
  "--font-geist-mono": "ui-monospace, 'SF Mono', Monaco, 'Cascadia Code', 'Roboto Mono', Consolas, 'Courier New', monospace"
} as React.CSSProperties;

export const metadata: Metadata = {
  title: "人事行政管理系统",
  description: "覆盖员工花名册、社保数据、账期处理与报表导出",
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="zh-CN" suppressHydrationWarning>
      <body
        className="antialiased bg-background text-foreground"
        style={fontVariables}
      >
        <Script src="/runtime-config.js" strategy="beforeInteractive" />
        <AppProviders>{children}</AppProviders>
      </body>
    </html>
  );
}
