import { promises as fs } from "fs";
import path from "path";
import type { NextRequest } from "next/server";
import { NextResponse } from "next/server";

export const runtime = "nodejs";
export const dynamic = "force-dynamic";

const TEMPLATE_FILES = {
  increase: "社保缴费人员增加申报（企业职工批量新参保）模版.xls",
  decrease: "社保缴费人员减少申报（企业职工批量减少参保）模板.xls",
} as const;

const resolveTemplateDir = () => {
  if (process.env.TEMPLATE_BASE_DIR) {
    return process.env.TEMPLATE_BASE_DIR;
  }
  // 优先使用容器/构建产物中的 templates 目录，若不存在则退回仓库上级目录
  const bundledDir = path.resolve(process.cwd(), "templates");
  return bundledDir;
};

export async function GET(request: NextRequest) {
  const url = new URL(request.url);
  const typeParam = url.searchParams.get("type");
  if (typeParam !== "increase" && typeParam !== "decrease") {
    return NextResponse.json({ error: "type 参数无效" }, { status: 400 });
  }

  const relativePath = TEMPLATE_FILES[typeParam];
  const baseDir = resolveTemplateDir();
  const filePath = path.resolve(baseDir, relativePath);
  try {
    await fs.access(filePath);
    const data = await fs.readFile(filePath);
    const payload = new Uint8Array(data);
    return new NextResponse(payload, {
      status: 200,
      headers: {
        "Content-Type": "application/vnd.ms-excel",
        "Content-Disposition": `attachment; filename="${encodeURIComponent(relativePath)}"`,
        "Cache-Control": "no-store",
      },
    });
  } catch (error) {
    console.error("[insurance-template] failed to load template", { filePath, error });
    return NextResponse.json({ error: "模板读取失败" }, { status: 500 });
  }
}
