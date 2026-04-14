import { jsPDF } from "jspdf";

type JsPdfFontRegistrar = jsPDF & {
  addFileToVFS?: (filename: string, data: string) => void;
  addFont?: (postScriptName: string, fontName: string, fontStyle: string) => void;
};

const FONT_FILE_NAME = "LXGWWenKai-Regular.ttf";
const FONT_NAME = "LXGWWenKai";
export const REGISTERED_FONT_NAME = FONT_NAME;

let fontDataBase64: string | null = null;
let fontDataPromise: Promise<string> | null = null;
let fontRegisteredGlobally = false;

const fetchFontBase64 = async (): Promise<string> => {
  if (fontDataBase64) {
    return fontDataBase64;
  }
  if (fontDataPromise) {
    return fontDataPromise;
  }

  fontDataPromise = (async () => {
    if (typeof window === "undefined") {
      throw new Error("当前环境不支持字体注册");
    }

    const response = await fetch(`/fonts/${FONT_FILE_NAME}`);
    if (!response.ok) {
      throw new Error(`字体文件加载失败：${response.status}`);
    }

    const buffer = await response.arrayBuffer();
    const bytes = new Uint8Array(buffer);
    let binary = "";
    const chunkSize = 0x8000;
    for (let offset = 0; offset < bytes.length; offset += chunkSize) {
      const chunk = bytes.subarray(offset, offset + chunkSize);
      binary += String.fromCharCode(...chunk);
    }
    fontDataBase64 = btoa(binary);
    return fontDataBase64;
  })().catch((error) => {
    fontDataPromise = null;
    throw error;
  });

  return fontDataPromise;
};

const injectFont = (target: JsPdfFontRegistrar, base64: string) => {
  if (typeof target.addFileToVFS !== "function" || typeof target.addFont !== "function") {
    return false;
  }
  target.addFileToVFS(FONT_FILE_NAME, base64);
  target.addFont(FONT_FILE_NAME, FONT_NAME, "normal");
  return true;
};

export const ensureFont = async (doc: jsPDF) => {
  const base64 = await fetchFontBase64();
  if (!fontRegisteredGlobally) {
    const api = (jsPDF as unknown as { API?: JsPdfFontRegistrar }).API;
    if (api && injectFont(api, base64)) {
      fontRegisteredGlobally = true;
    }
  }

  const registrar = doc as JsPdfFontRegistrar;
  if (!injectFont(registrar, base64) && !fontRegisteredGlobally) {
    throw new Error("当前 jsPDF 实例缺少 addFileToVFS 或 addFont 方法");
  }

  const fontList = typeof registrar.getFontList === "function" ? registrar.getFontList() : undefined;
  if (fontList && !Object.prototype.hasOwnProperty.call(fontList, FONT_NAME)) {
    // 再次尝试注册一次，若仍失败则抛出错误
    if (!injectFont(registrar, base64)) {
      throw new Error("字体注册失败，jsPDF 无法识别指定字体");
    }
  }

  doc.setFont(FONT_NAME, "normal");
};
