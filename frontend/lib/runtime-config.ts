export interface RuntimeConfig {
  /** 完整的 API 基础地址，例如 https://example.com/api */
  API_BASE?: string;
  /** 域名形式的 API 地址，例如 https://example.com/api */
  API_BASE_DOMAIN?: string;
  /** IPv4 形式的 API 地址，例如 http://1.2.3.4:8081/api */
  API_BASE_IP?: string;
  /** 当命中裸 IP 时用于拼接端口的兜底值 */
  API_IPV4_FALLBACK_PORT?: string;
}

declare global {
  interface Window {
    __RUNTIME_CONFIG__?: RuntimeConfig;
  }

  var __RUNTIME_CONFIG__: RuntimeConfig | undefined;
}

export function getRuntimeConfig(): RuntimeConfig {
  if (typeof window !== "undefined") {
    return window.__RUNTIME_CONFIG__ ?? {};
  }

  return (globalThis as typeof globalThis & { __RUNTIME_CONFIG__?: RuntimeConfig }).__RUNTIME_CONFIG__ ?? {};
}
