import type { Metadata } from "next";
import "./globals.css";
import { AppProviders } from "./providers";

// Force dynamic rendering for all pages to avoid SSR issues with client-side auth
export const dynamic = 'force-dynamic';
export const dynamicParams = true;

// Use system fonts to avoid Google Fonts network issues during Docker build
const fontVariables = {
  "--font-geist-sans": "system-ui, -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif",
  "--font-geist-mono": "ui-monospace, 'SF Mono', Monaco, 'Cascadia Code', 'Roboto Mono', Consolas, 'Courier New', monospace"
} as React.CSSProperties;

export const metadata: Metadata = {
  title: "社保数据整合平台",
  description: "上传社保险种明细并生成汇总、扣款表",
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="zh-CN">
      <body
        className="antialiased bg-background text-foreground"
        style={fontVariables}
      >
        <AppProviders>{children}</AppProviders>
      </body>
    </html>
  );
}
