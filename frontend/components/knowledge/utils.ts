"use client";

// ========== 源模块中文映射 ==========

/** 预定义的源模块名称 → 中文标题映射 */
const SOURCE_MODULE_MAP: Record<string, string> = {
  employee: "员工花名册",
  insurance: "社保业务",
  canteen: "食堂管理",
  invoice: "发票管理",
  dormitory: "宿舍管理",
  archives: "档案管理",
  "office-supply": "办公劳保",
  announcement: "公告通知",
  custom: "自定义",
};

/**
 * 将源模块标识符转为中文标题。
 * @param module 源模块标识（如 employee、custom）
 * @returns 中文名称，未映射时返回原文
 */
export function getSourceModuleLabel(module: string): string {
  return SOURCE_MODULE_MAP[module] ?? module;
}

// ========== 可见性中文映射 ==========

const VISIBILITY_MAP: Record<string, string> = {
  public: "公开",
  restricted: "受限",
  private: "私有",
};

/**
 * 将可见性标识转为中文。
 * @param visibility 可见性（public/restricted/private）
 * @returns 中文标签
 */
export function getVisibilityLabel(visibility: string): string {
  return VISIBILITY_MAP[visibility] ?? visibility;
}

// ========== 脱敏模式中文映射 ==========

const MASK_PATTERN_MAP: Record<string, string> = {
  front3back4: "前 3 后 4",
  all_star: "全星号",
};

/**
 * 将脱敏模式转为中文说明。
 * @param pattern 脱敏模式（front3back4 / all_star）
 * @returns 中文标签
 */
export function getMaskPatternLabel(pattern: string): string {
  return MASK_PATTERN_MAP[pattern] ?? pattern;
}

// ========== ChunkingConfig 格式化 ==========

/**
 * 将 JSON 分块配置格式化为简短可读字符串。
 * 后端 DefaultChunkingConfig 包含 strategy / chunk_size / chunk_overlap 等字段。
 * @param config 分块配置（可为 JSON 对象、字符串或 null）
 * @returns 格式化的中文描述，null/空对象时返回 "默认"
 */
export function formatChunkingConfig(config: unknown): string {
  if (config === null || config === undefined) return "默认";

  let cfg: Record<string, unknown>;
  if (typeof config === "string") {
    try {
      cfg = JSON.parse(config);
    } catch {
      return "默认";
    }
  } else if (typeof config === "object") {
    cfg = config as Record<string, unknown>;
  } else {
    return "默认";
  }

  const strategy = cfg.strategy ?? "auto";
  const chunkSize = cfg.chunk_size ?? 512;
  const overlap = cfg.chunk_overlap ?? 80;

  const strategyLabel = strategy === "auto" ? "自动" : String(strategy);
  return `${strategyLabel} · ${chunkSize}字块 · ${overlap}字重叠`;
}

// ========== 角色中文映射 ==========

const ROLE_LABEL_MAP: Record<string, string> = {
  admin: "管理员",
  super_admin: "超级管理员",
  manager: "经理",
  editor: "编辑者",
  viewer: "查看者",
};

/**
 * 将角色标识转为中文标签。
 * @param role 角色标识
 * @returns 中文标签
 */
export function getRoleLabel(role: string): string {
  return ROLE_LABEL_MAP[role] ?? role;
}
