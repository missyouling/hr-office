import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import {
  AVATAR_MAX_BYTES,
  AVATAR_SIZE,
  computeCropRect,
  cropImageToWebP,
  getInitial,
  getInitialColorClass,
  validateAvatarFile,
} from "./avatar";

describe("validateAvatarFile 文件校验", () => {
  it("允许 JPG/PNG/WebP 格式", () => {
    expect(validateAvatarFile(new File(["x"], "a.jpg", { type: "image/jpeg" })).ok).toBe(true);
    expect(validateAvatarFile(new File(["x"], "a.png", { type: "image/png" })).ok).toBe(true);
    expect(validateAvatarFile(new File(["x"], "a.webp", { type: "image/webp" })).ok).toBe(true);
  });

  it("拒绝不支持格式并给出中文提示", () => {
    const result = validateAvatarFile(new File(["x"], "a.gif", { type: "image/gif" }));
    expect(result.ok).toBe(false);
    if (!result.ok) {
      expect(result.error).toContain("仅支持 JPG、PNG、WebP");
    }
  });

  it("拒绝超过 2 MiB 的图片并给出中文提示", () => {
    const big = new File([new Uint8Array(AVATAR_MAX_BYTES + 1)], "big.png", { type: "image/png" });
    const result = validateAvatarFile(big);
    expect(result.ok).toBe(false);
    if (!result.ok) {
      expect(result.error).toContain("2 MB");
    }
  });

  it("恰好 2 MiB 允许通过（边界）", () => {
    const exact = new File([new Uint8Array(AVATAR_MAX_BYTES)], "exact.png", { type: "image/png" });
    expect(validateAvatarFile(exact).ok).toBe(true);
  });
});

describe("computeCropRect 裁剪矩形计算", () => {
  const VIEWPORT = 256;

  it("1:1 图片 cover 缩放后裁剪全图", () => {
    // 1000x1000，scale=0.256 → 绘制 256x256 恰好铺满视口
    const rect = computeCropRect(1000, 1000, VIEWPORT, 0.256, { x: 0, y: 0 });
    expect(rect).not.toBeNull();
    expect(rect!.x).toBeCloseTo(0);
    expect(rect!.y).toBeCloseTo(0);
    expect(rect!.width).toBeCloseTo(1000);
    expect(rect!.height).toBeCloseTo(1000);
  });

  it("宽图 cover 后裁剪中间区域", () => {
    // 2000x1000，scale=0.256 → 绘制 512x256，水平居中 → 裁剪 x=500 起 1000 宽
    const rect = computeCropRect(2000, 1000, VIEWPORT, 0.256, { x: 0, y: 0 });
    expect(rect).not.toBeNull();
    expect(rect!.x).toBeCloseTo(500);
    expect(rect!.y).toBeCloseTo(0);
    expect(rect!.width).toBeCloseTo(1000);
    expect(rect!.height).toBeCloseTo(1000);
  });

  it("高图 cover 后裁剪中间区域", () => {
    // 1000x2000，scale=0.256 → 绘制 256x512，垂直居中 → 裁剪 y=500 起 1000 高
    const rect = computeCropRect(1000, 2000, VIEWPORT, 0.256, { x: 0, y: 0 });
    expect(rect).not.toBeNull();
    expect(rect!.x).toBeCloseTo(0);
    expect(rect!.y).toBeCloseTo(500);
    expect(rect!.width).toBeCloseTo(1000);
    expect(rect!.height).toBeCloseTo(1000);
  });

  it("水平平移 offset 改变裁剪起点", () => {
    // 2000x1000 宽图，scale=0.256 → 绘制 512x256 水平居中(left=-128)。
    // 向右平移 100px → left=-28 → 裁剪起点 x = -left/scale = 28/0.256 ≈ 109.375
    const rect = computeCropRect(2000, 1000, VIEWPORT, 0.256, { x: 100, y: 0 });
    expect(rect).not.toBeNull();
    expect(rect!.x).toBeCloseTo(109.375);
    expect(rect!.y).toBeCloseTo(0);
  });

  it("平移越界时 clamp 到图片边界", () => {
    // 大幅向右平移 → 裁剪起点被限制在图片右边界内
    const rect = computeCropRect(2000, 1000, VIEWPORT, 0.256, { x: 5000, y: 0 });
    expect(rect).not.toBeNull();
    // 裁剪矩形右边缘不能超过图片宽度
    expect(rect!.x + rect!.width).toBeLessThanOrEqual(2000 + 1e-6);
    expect(rect!.x).toBeGreaterThanOrEqual(0);
  });

  it("非法参数返回 null", () => {
    expect(computeCropRect(0, 100, VIEWPORT, 1, { x: 0, y: 0 })).toBeNull();
    expect(computeCropRect(100, 100, VIEWPORT, 0, { x: 0, y: 0 })).toBeNull();
    expect(computeCropRect(-1, 100, VIEWPORT, 1, { x: 0, y: 0 })).toBeNull();
  });
});

describe("cropImageToWebP Canvas 导出", () => {
  // toBlob 签名：回调 + 可选 MIME 类型与质量参数
  type ToBlobImpl = (callback: BlobCallback, type?: string, quality?: number) => void;
  let drawImageMock: ReturnType<typeof vi.fn>;
  let toBlobMock: ReturnType<typeof vi.fn<ToBlobImpl>>;
  let capturedCanvas: HTMLCanvasElement | null;

  beforeEach(() => {
    capturedCanvas = null;
    drawImageMock = vi.fn();
    toBlobMock = vi.fn<ToBlobImpl>((callback) => {
      callback(new Blob(["fake-webp"], { type: "image/webp" }));
    });
    // jsdom 未实现 Canvas 绘制，这里 mock 原型方法并捕获创建的 canvas 实例
    const originalCreateElement = document.createElement.bind(document);
    vi.spyOn(document, "createElement").mockImplementation((tagName: string, options?: ElementCreationOptions) => {
      const el = originalCreateElement(tagName, options);
      if (tagName === "canvas") {
        capturedCanvas = el as HTMLCanvasElement;
      }
      return el;
    });
    vi.spyOn(HTMLCanvasElement.prototype, "getContext").mockReturnValue({
      drawImage: drawImageMock,
    } as unknown as CanvasRenderingContext2D);
    if (typeof HTMLCanvasElement.prototype.toBlob !== "function") {
      HTMLCanvasElement.prototype.toBlob = toBlobMock as unknown as typeof HTMLCanvasElement.prototype.toBlob;
    } else {
      vi.spyOn(HTMLCanvasElement.prototype, "toBlob").mockImplementation(toBlobMock);
    }
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("导出 256x256 WebP 并绘制裁剪区域", async () => {
    const img = new Image();
    const rect = { x: 10, y: 20, width: 100, height: 100 };
    const blob = await cropImageToWebP(img, rect);

    expect(blob.type).toBe("image/webp");
    expect(capturedCanvas).not.toBeNull();
    expect(capturedCanvas!.width).toBe(AVATAR_SIZE);
    expect(capturedCanvas!.height).toBe(AVATAR_SIZE);
    // drawImage 收到裁剪矩形与目标尺寸
    expect(drawImageMock).toHaveBeenCalledWith(img, 10, 20, 100, 100, 0, 0, AVATAR_SIZE, AVATAR_SIZE);
    // toBlob 以 WebP 格式导出
    expect(toBlobMock).toHaveBeenCalledWith(expect.any(Function), "image/webp", 0.9);
  });

  it("Canvas 上下文不可用时拒绝", async () => {
    vi.spyOn(HTMLCanvasElement.prototype, "getContext").mockReturnValue(null);
    await expect(cropImageToWebP(new Image(), { x: 0, y: 0, width: 10, height: 10 })).rejects.toThrow(
      "当前环境不支持图片裁剪",
    );
  });
});

describe("getInitial 首字母回退", () => {
  it("中文取首字", () => {
    expect(getInitial("张三")).toBe("张");
  });

  it("拉丁字母取首字母大写", () => {
    expect(getInitial("alice")).toBe("A");
    expect(getInitial("Bob")).toBe("B");
  });

  it("空串与空白返回问号", () => {
    expect(getInitial("")).toBe("?");
    expect(getInitial("   ")).toBe("?");
  });
});

describe("getInitialColorClass 稳定配色", () => {
  it("同一名字配色恒定", () => {
    expect(getInitialColorClass("张三")).toBe(getInitialColorClass("张三"));
  });

  it("返回合法 Tailwind 类名", () => {
    const cls = getInitialColorClass("李四");
    expect(cls).toMatch(/^bg-/);
    expect(cls).toContain("text-");
  });
});