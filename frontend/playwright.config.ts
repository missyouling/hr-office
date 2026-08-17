import { defineConfig, devices } from "@playwright/test";

/**
 * Playwright E2E 配置（P7.1 RBAC 权限矩阵验证）
 *
 * 运行前提：后端需先启动并监听 :8080；Playwright 会自动拉起前端开发服务。
 *   1. 初始化测试账号（幂等，可重复执行）：
 *      cd backend && SIAPP_DATABASE_PATH=<数据库路径> go run ./cmd/seed-e2e
 *      seed-e2e 同时会创建一条全局启用的 LLM 配置（user_id IS NULL），
 *      解除 viewer 等无自有配置账号走 /api/knowledge/chat/stream 时的
 *      "未找到可用的 LLM 配置" 阻塞。
 *   2. 启动后端（监听 :8080）：
 *      cd backend && CGO_ENABLED=1 go run .
 *   3. 运行 E2E：
 *      cd frontend && npm run test:e2e
 *
 * 可选环境变量（覆盖 seed-e2e 创建的默认 Siliconflow 测试 LLM）：
 *   E2E_LLM_MODEL / E2E_LLM_ENDPOINT / E2E_LLM_API_KEY
 *   默认模型 Qwen/Qwen3-8B，端点 https://api.siliconflow.cn/v1。
 */
export default defineConfig({
  // 测试文件目录
  testDir: "./e2e",
  // 单个用例超时时间
  timeout: 90_000,
  // 用例之间并行执行（每个用例独立登录，不共享状态）
  fullyParallel: true,
  // 失败不重试（便于暴露真实问题）
  retries: 0,
  // 控制台输出简洁列表
  reporter: "list",
  use: {
    // 前端 dev server 地址
    baseURL: "http://localhost:3000",
    // 首次失败时保留 trace 便于排查
    trace: "on-first-retry",
  },
  projects: [
    {
      name: "chromium",
      use: { ...devices["Desktop Chrome"] },
    },
  ],
  webServer: {
    command: "NEXT_PUBLIC_API_BASE_URL=http://localhost:8080/api npm run dev",
    url: "http://localhost:3000/",
    reuseExistingServer: true,
    timeout: 60_000,
  },
});
