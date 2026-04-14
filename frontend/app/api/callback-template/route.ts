import { promises as fs } from "fs";
import path from "path";
import { NextResponse } from "next/server";

export const runtime = "nodejs";
export const dynamic = "force-dynamic";

const DEFAULT_FILE = "人员增减申请回盘表2025.07.23.xlsx";

const resolveTemplateDir = () => {
  if (process.env.TEMPLATE_BASE_DIR) {
    return process.env.TEMPLATE_BASE_DIR;
  }
  return path.resolve(process.cwd(), "templates");
};

export async function GET() {
  const baseDir = resolveTemplateDir();
  const filePath = path.resolve(baseDir, DEFAULT_FILE);
  try {
    await fs.access(filePath);
    const data = await fs.readFile(filePath);
    const payload = new Uint8Array(data); // ensure BodyInit type is ArrayBufferView
    return new NextResponse(payload, {
      status: 200,
      headers: {
        "Content-Type": "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
        "Content-Disposition": `attachment; filename="${encodeURIComponent(DEFAULT_FILE)}"`,
        "Cache-Control": "no-store",
      },
    });
  } catch (error) {
    console.error("[callback-template] failed to load template", { filePath, error });
    return NextResponse.json({ error: "模板读取失败" }, { status: 500 });
  }
}
