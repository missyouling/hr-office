# 前端 UI 外壳迁移验证计划

> 文档类型：验证计划（仅规划验证活动，不修改任何业务代码）
> 适用范围：已确认的前端 UI 外壳（应用外壳）迁移——侧边栏、导航菜单、顶部用户区、底部管理条（ManagementBar）、浮动 Dock、用户头像、通知中心、个人设置等外壳组件的重构升级
> 验证责任：主协调者（Orchestrator）负责组织执行、记录证据并签署各阶段门槛；各层验证可由对应 specialist 执行，结果统一汇总至主协调者
> 安全约定：本计划不包含任何密码、令牌等敏感参数；测试账号仅引用代码中的常量，不展开具体值

---

## 1. 验收主张（Acceptance Claims）

每条主张必须被至少一条证据路径（见第 3 节）支撑，全部通过视为迁移验收合格。

| 编号 | 验收主张 | 证据路径 |
| --- | --- | --- |
| AC-1 | 登录后应用外壳完整渲染：侧边栏、内容区、底部管理条、浮动 Dock 均可见且布局正常 | L2、L3 |
| AC-2 | 导航可用：点击侧边栏菜单可切换视图，激活态高亮正确 | L2 |
| AC-3 | 用户区能力可用：头像展示与编辑、个人设置、通知中心可正常打开与交互 | L1、L2 |
| AC-4 | 外壳权限联动不回归：菜单/按钮按角色显隐的行为与迁移前一致 | L1、L2 |
| AC-5 | 深浅双模式视觉一致：无硬编码色破相，主题变量（`--background` / `--sidebar-*` 等）生效 | L3 |
| AC-6 | 外壳改动未破坏既有主流程：RBAC、反馈闭环、发票管理 3 个既有 E2E 全部通过 | L2 |
| AC-7 | 全量质量门禁通过：前端 lint / tsc / build / Vitest 与后端 go test / vet / build 全绿 | L4 |
| AC-8 | 迁移可逆：迁移前外壳在临时回退分支可正常构建运行（演练级，不做代码恢复） | L5 |
| AC-9 | 旧功能全量接入与占位清零：迁移矩阵中 ⏳ / 🛑 状态全部转为 ✅，全部旧功能真实接入基线导航，无占位页面或「功能开发中」等占位文案残留 | L4（矩阵状态核对 + 残留检测） |

---

## 2. 风险清单

| 风险 | 影响 | 缓解措施 |
| --- | --- | --- |
| `.next` 目录竞争 | dev server 与 `next build` 并行执行会互相覆盖产物，导致 E2E 或 build 随机失败 | 第 5 节串行约束（硬性） |
| 陈旧 `.next` 缓存 | E2E 实际验证的是旧产物，误判迁移结果 | 验证前清理 `frontend/.next` |
| jsdom 能力有限 | Vitest 无法验证布局、动画、滚动等真实渲染行为 | 布局类断言必须走 L2/L3，不依赖 L1 |
| 无截图基线 | 视觉回归只能人工比对，无自动 diff | 首阶段采用人工核对，不引入新依赖（后续可选基线化） |
| 外壳影响全站布局 | 外壳容器尺寸改动可能波及所有模块页面 | 回归既有 3 个 E2E spec + 登录冒烟 |
| 深色模式易漏测 | 自动化默认浅色，深色破相难以被发现 | L3 强制浅/深各截图一张 |
| 权限逻辑与外壳耦合 | 外壳重构可能连带改动权限显隐分支 | 以 `rbac.spec.ts` 作为外壳回归主入口 |
| 环境依赖 | E2E 需后端 :8080 与种子账号，环境不一致易误报 | seed-e2e 幂等创建 + 独立测试库 |

---

## 3. 分层证据路径

证据按成本从低到高组织为五层，逐层叠加，**高层通过不豁免低层**：

| 层级 | 内容 | 现有设施 | 覆盖主张 |
| --- | --- | --- | --- |
| **L1 单元/组件集成** | 外壳组件的状态分支、权限分支、交互回调、hooks 与工具函数 | Vitest + jsdom（`frontend/vitest.config.ts`，已排除 e2e 目录） | AC-3、AC-4 |
| **L2 E2E** | 真实浏览器下的登录、外壳渲染、导航、权限显隐、主流程回归 | Playwright（`frontend/playwright.config.ts`，webServer 自动拉起 dev server） | AC-1、AC-2、AC-3、AC-4、AC-6 |
| **L3 视觉截图** | 关键视图的浅色/深色截图，人工比对 | Playwright 内置 `page.screenshot`，输出到 `frontend/test-results/screenshots/`（已 gitignore） | AC-1、AC-5 |
| **L4 全量门禁** | 静态检查、类型检查、构建、后端测试 | `npm run lint` / `npx tsc --noEmit` / `npm run build` / `go test ./...` + CI（`.github/workflows/ci.yml`） | AC-7、AC-9 |
| **L5 回退验证** | 迁移前外壳可构建运行，确认迁移可逆 | git 临时分支 checkout 迁移前 commit | AC-8 |

**证据记录要求**：每层执行后在文档末尾的验证记录表追加一行（日期、执行人、命令、结果、截图路径），由主协调者确认归档。

---

## 4. 验证环境与命令（仓库现状）

### 4.1 前置环境

- 前端工作目录 `frontend/`，后端工作目录 `backend/`。
- Playwright 配置已就绪：`baseURL: http://localhost:3000`，webServer 自动执行 `npm run dev`（`NEXT_PUBLIC_API_BASE_URL=http://localhost:8080/api`），无需额外安装依赖。
- 测试账号由 `backend/cmd/seed-e2e` 幂等创建（账号清单见 `frontend/e2e/helpers/auth.ts` 的 `ACCOUNTS` 常量；具体值属敏感参数，本计划不展开）。
- 数据库隔离：主后端强制 PostgreSQL——`SIAPP_DATABASE_TYPE=postgres`，连接参数为 `SIAPP_DB_HOST` / `SIAPP_DB_PORT` / `SIAPP_DB_USER` / `SIAPP_DB_PASSWORD` / `SIAPP_DB_NAME` / `SIAPP_DB_SSLMODE`（SQLite 已禁用，`SIAPP_DATABASE_PATH` 不再是主程序数据库连接变量）。E2E 使用独立的 PostgreSQL 测试数据库（示例库名 `siapp_e2e`）与开发/生产库分离；本计划不写入任何密码或实际连接串。

### 4.2 命令速查

| 步骤 | 命令（在对应目录执行） |
| --- | --- |
| 创建种子账号（幂等） | `cd backend && SIAPP_DATABASE_PATH=<测试库 DSN> go run ./cmd/seed-e2e`（该变量仅供 seed-e2e 以 DSN 直连，非主程序连接变量） |
| 启动后端（监听 :8080） | `cd backend && SIAPP_DATABASE_TYPE=postgres SIAPP_DB_HOST=<主机> SIAPP_DB_PORT=<端口> SIAPP_DB_USER=<用户> SIAPP_DB_PASSWORD=<密码> SIAPP_DB_NAME=<库名> SIAPP_DB_SSLMODE=<sslmode> JWT_SECRET_KEY=<本地密钥> go run .` |
| L1 单元/组件测试 | `cd frontend && npm run test` |
| L2 全量 E2E | `cd frontend && npm run test:e2e` |
| L2 单跑外壳回归 | `cd frontend && npx playwright test e2e/rbac.spec.ts` |
| L4 前端门禁 | `cd frontend && npm run lint && npx tsc --noEmit && npm run build` |
| L4 后端门禁 | `cd backend && go test ./... && go vet ./... && go build ./...` |
| L4 CI | 推送 / 提 PR 触发 `.github/workflows/ci.yml`（前端 lint+build、后端 test+build、Docker 构建） |
| L5 回退演练 | 见第 7 节 G5 |

**环境变量约定（E2E 验证环境）**：

- 主后端必须设置 `SIAPP_DATABASE_TYPE=postgres` 及全套 `SIAPP_DB_*` 连接参数（`SIAPP_DB_HOST` / `SIAPP_DB_PORT` / `SIAPP_DB_USER` / `SIAPP_DB_PASSWORD` / `SIAPP_DB_NAME` / `SIAPP_DB_SSLMODE`）；SQLite 已禁用，`SIAPP_DATABASE_PATH` 仅可供 seed-e2e 以 DSN 直连测试库，不再作为主程序数据库连接变量。
- 不得设置任何 `SUPABASE_*` 环境变量（`SUPABASE_URL`、`SUPABASE_SERVICE_ROLE_KEY`、`SUPABASE_JWT_SECRET`、`SUPABASE_DB_PASSWORD` 等），确保验证环境走本地自建认证与本地 PostgreSQL，而非云端 Supabase。
- 必须设置本地 `JWT_SECRET_KEY`：未配置 `SUPABASE_JWT_SECRET` 时后端启动会强制要求（自建认证模式），开发环境使用本地任意密钥即可。
- 上表所有 `<...>` 仅为占位标记，本计划不写入具体密码、令牌或连接串。

---

## 5. 环境隔离与 `.next` 串行约束（硬性）

**背景**：`npm run test:e2e` 的 webServer 会执行 `npm run dev`（Next dev server），`npm run build` 执行 `next build --turbopack`；二者都读写 `frontend/.next/`，同一工作区并行会互相覆盖产物。

1. **dev（含 E2E）与 build 严格串行**：同一份前端工作区上，禁止在 dev server 运行期间执行 `npm run build`，反之亦然。E2E 全部结束后（webServer 随测试进程退出），再执行 build。
2. **不依赖 `reuseExistingServer` 的空窗**：若开发者手动启动了 :3000 dev server，Playwright 会复用；此状态下同样禁止执行 build。
3. **验证前清理缓存**：怀疑产物陈旧时执行 `rm -rf frontend/.next` 后再跑 E2E 或 build，保证验证对象是当前代码。
4. **Vitest 与构建互不冲突**：Vitest 使用 jsdom，不触碰 `.next`；但为避免磁盘/CPU 争用，建议按 L1 → L2 → L4 顺序执行。
5. **测试库隔离**：E2E 使用独立 PostgreSQL 测试数据库（示例库名 `siapp_e2e`，经 `SIAPP_DATABASE_TYPE=postgres` 与 `SIAPP_DB_*` 参数指向）；seed-e2e 的 `SIAPP_DATABASE_PATH` 仅作 DSN 直连，不代表主程序数据库连接变量。`frontend/test-results/`、`frontend/.next/` 均已 gitignore，截图与产物不入库。
6. **不并行两套 E2E**：同一数据库上同时跑两批 Playwright 会互相干扰种子数据，E2E 只串行跑。

---

## 6. 首阶段最小用例集

首阶段（G1-G3，见第 7 节）只需满足以下最小集合即可放行进入全量门禁，后续可增量补充：

**L1（Vitest，全部必须通过）**——外壳迁移配套测试文件：

- `frontend/components/layout/management-bar.test.tsx`、`nav-user.test.tsx`、`app-sidebar.test.tsx`
- `frontend/components/ui/sidebar.test.tsx`、`floating-dock.test.tsx`
- `frontend/components/personal-settings.test.tsx`、`notification-center.test.tsx`
- `frontend/components/avatar/user-avatar.test.tsx`、`frontend/hooks/use-user-avatar.test.tsx`
- `frontend/lib/avatar.test.ts`、`avatar-api.test.ts`、`preferences.test.ts`
- `frontend/components/chat-panel.test.tsx`（外壳内嵌面板回归）

**L2（Playwright，单跑）**：`e2e/rbac.spec.ts` 全量——覆盖登录、侧边栏渲染、菜单按角色显隐、按钮显隐，是外壳权限联动（AC-4）与主流程（AC-6）的回归主入口。

**L3（截图，人工核对）**：admin 登录后首页两张截图——浅色一张、经现有主题切换入口（next-themes）切深色后再一张；截图输出到 `frontend/test-results/screenshots/`。

**L4（门禁）**：前端 `lint` + `tsc` + `build`；后端 `go test ./...`（迁移不涉及后端代码时至少跑一次全量确认无连带影响）。

**L5（可选，首次迁移必做）**：回退演练，见 G5。

---

## 7. 每阶段质量门槛

| 门槛 | 执行内容 | 通过标准 | 责任 |
| --- | --- | --- | --- |
| **G0 前置** | 确认后端 :8080 可启动、seed-e2e 幂等执行成功、`frontend/.next` 已清理 | 种子账号创建成功、后端健康、无残留 dev server；主协调者以启动配置确认 DB 名称/地址（`SIAPP_DB_NAME` / `SIAPP_DB_HOST` 等实际指向独立测试库），禁止仅凭 /health 返回即认定隔离 | 主协调者 |
| **G1 组件层** | 运行第 6 节 L1 全部测试 | Vitest 全部通过，无跳过失败 | 前端 specialist |
| **G2 E2E 回归** | 串行运行 L2 既有 3 个 spec（rbac / feedback-closure / invoice-p73） | 3 个 spec 全部通过，无 flaky 重试依赖 | 前端 specialist |
| **G3 视觉截图** | 执行 L3 浅/深截图并人工比对（对照迁移前截图或设计规范 `docs/ui-design-p9.md`） | 深浅双模式无硬编码色破相、无布局错位，截图归档 | 主协调者 |
| **G4 全量门禁** | 前端 lint/tsc/build + 后端 go test/vet/build；推送触发 CI | 全部命令退出码 0，CI 流水线绿 | 主协调者 |
| **G5 回退演练** | `git stash`（若有未提交改动）→ 临时分支 checkout 迁移前 commit（`df4278b^`）→ `npm run build` 通过 → 切回原分支 | 旧外壳可构建，确认回退路径存在 | 主协调者 |

> 说明：G5 仅验证可回退性，不恢复任何代码；演练完成后必须回到当前分支并清理临时分支。

---

## 8. 最终切换门槛

"切换"指主协调者签署——以新外壳为准、迁移视为完成且不再回退。必须**同时满足**：

1. G0-G4 全部通过，G5（回退演练）完成且有记录。
2. **旧功能全量真实接入（硬性）**：迁移矩阵中 ⏳ 与 🛑 状态全部转为 ✅，全部旧功能真实接入基线导航；无任何占位页面或「功能开发中，敬请期待」等占位文案残留（残留检测零命中）。分阶段期间的"已登记占位 / 已登记下批次计划"不作为保留占位的理由。
3. AC-1 至 AC-9 全部主张有对应证据，无未解释偏差。
4. 无已知严重 BUG：E2E 无控制台异常、无 5xx 接口错误、无深色模式破相。
5. 视觉截图人工确认通过（浅/深各至少一张关键视图）；系统头部标题字体与动效经核对未变。
6. 验证记录表归档完整，结果同步至 TODO.md 与 agentmemory【任务规划】/【测试记录】。

任一条件未满足时，切换挂起，由主协调者发起偏差处置（修复后重跑受影响门槛），禁止带失败项签署。

---

## 9. 禁止事项

- 禁止在本计划或任何验证脚本中写入密码、令牌、API 密钥等敏感参数；测试账号一律引用 `e2e/helpers/auth.ts` 常量。
- 禁止引入新的验证依赖（如截图对比库、测试框架）；视觉比对用 Playwright 内置 `page.screenshot` + 人工核对。
- 禁止 dev 与 build 并行（见第 5 节）。
- 禁止在验证过程中修改业务代码；发现缺陷时先记录，交由实施方修复后再重跑对应门槛。
- 禁止以"全部 E2E 通过"豁免 L1 或 L4；各层独立通过。

---

## 10. 完成核对（Markdown 自检）

- [x] 标题层级从 H2 起连续，无跳级（H2 → H3 顺序）。
- [x] 表格均有表头行，`| --- |` 分隔行完整。
- [x] 若含代码块，均标注语言（bash / ts）；本计划以表格承载命令，未使用未标注代码块。
- [x] 全文无敏感参数；数据库连接一律以占位表达（如 `<测试库 DSN>`、`<密码>`），无密码或实际连接串。
- [x] 命令均来自仓库现有配置（package.json / playwright.config.ts / ci.yml / main.go 与 `.env.postgresql.example` 的环境变量 / TODO.md 记录）。

---

## 11. 验证记录表

| 日期 | 门槛 | 执行人 | 命令 | 结果 | 截图/证据路径 |
| --- | --- | --- | --- | --- | --- |
| 2026-08-17 | G0（P12.0.2 准备证据） | 主协调者 | seed-e2e 幂等 + 后端启动 + 健康检查 + `.next` 清理 | 准备证据就绪（随 P12.0.3 一并由主协调者签署） | 见 11.1 |
| 2026-08-17 | P12.0.3 环境变量隔离验证 | 主协调者 | 同镜像 Docker 验证（未设置 UI_SHELL → 回退旧壳 / 显式 new → 新壳）+ runtime-config 产物核对 + 临时容器清理 + `.next` 清理 | **通过** | 见 11.2 |
| 2026-08-17 | **G0 前置（正式签署）** | 主协调者 | 汇总 P12.0.1-0.4 证据（映射表定稿 / 基准 commit / 隔离验证 / G0 门槛）并签署 | **通过（已签署）** | 见 11.1、11.2 |
| 2026-08-17 | P12.1.5 L1 / L2 / L3 阶段验收 | 主协调者 | 全量 Vitest、lint、tsc、build；old/new 壳 RBAC；浅/深/移动截图 | **P12.1 阶段通过**；不等同于 G2-G5 或最终切换 | 见 11.3 |
| 2026-08-17 | P12.2.2 阶段验收 | 主协调者 | 专项 Vitest、new 壳 RBAC 筛选 E2E、lint、tsc、diff check | **P12.2.2 阶段通过**；不等同于 P12.2.3 或 G2-G4 | 见 11.5 |
| 2026-08-17 | P12.2.3 阶段验收 | 主协调者 | 隐藏策略决策登记 + 新壳侧栏隐藏断言（Vitest）+ `git diff --check` | **P12.2.3 阶段通过**；不等同于 G2-G4 或最终切换 | 见 11.6 |
| 2026-08-17 | P12.2.4 阶段验收 | 主协调者 | Dock 遮挡修复专项 Vitest、old/new 两轮 E2E、lint/tsc/build、后端 go test/vet/build、G3 截图视觉复核 | **P12.2.4 通过（G2/G3/G4 已签署）**；G5、AC-6~AC-9 与最终切换仍未签署 | 见 11.7 |
| 2026-08-17 | P12.3.1-1 离职管理子批次签署 | 主协调者 | 新壳/旧壳 E2E、后端 go test -count=1/vet/build、前端专项 Vitest/lint/tsc/build、隔离库与清理核对 | **批次签署通过**；生产切换未签署，⏳/🛑 尚未归零 | 见 11.9 |
| 2026-08-18 | P12.3.2 入职管理子批次签署 | 主协调者 | 新壳 E2E、后端专项/全量 go test/vet/build、前端专项 Vitest/lint/tsc/build、隔离库与清理核对 | **批次签署通过**；生产切换未签署，⏳/🛑 尚未归零 | 见 11.10 |
| 2026-08-20 | G1 | 主协调者 | `npm run test` | **通过：58 个测试文件、373 项测试** | Vitest 输出 |
| 2026-08-20 | G2 | 主协调者 | 隔离库 `npm run test:e2e` | **通过：37 项通过、4 项条件跳过、0 项失败**；反馈闭环使用用户批准的临时本地 OpenAI 兼容 SSE 模拟服务，结束后已还原 | Playwright 输出 |
| 2026-08-20 | G3 | 主协调者 | `page.screenshot` ×2 + 人工复核 | **通过**；浅深主题、侧栏、内容区、底部 Dock 可见；标题动效运行时断言通过 | `test-results/screenshots/final-shell-light.png`、`final-shell-dark.png` |
| 2026-08-20 | G4 | 主协调者 | lint/tsc/build + go test/vet/build | **通过**；前端仅 1 条既有 warning、无 error | 本轮门禁输出 |
| 2026-08-20 | G5 | 主协调者 | 隔离 worktree 回退构建 | **通过**；`omos/g5-rollback-3c18d29` 固定 `3c18d29`，`npm ci && npm run build` 成功 | 临时 worktree |
| 2026-08-20 | 占位清零核对（AC-9） | 主协调者 | 矩阵状态核对 + 运行时代码残留检测 | **通过**；矩阵无 ⏳/🛑，`components`/`app` 运行时代码零命中 | 测试反向断言不计入运行时残留 |
| 2026-08-20 | 切换 | 主协调者 | AC-1 至 AC-9 证据汇总与签署 | **迁移验收签署通过**；P12.3.4 生产移除开关尚未执行 | 见 11.23 |

### 11.1 G0 准备证据明细（P12.0.2，验证责任：主协调者）

- **回退基准**：`72f0243`（迁移前 commit，回退演练以该 commit 为基准）。
- **本地隔离后端**：明确连接 `127.0.0.1:55432` 的 `siapp_e2e` 独立测试库，与开发/生产库分离；不以 /health 返回即认定隔离。
- **seed-e2e**：已幂等执行成功（不记录账号/密码/密钥等敏感信息）。
- **健康检查**：`/health`、`/health/readiness`、`/health/liveness`、`/version` 全部通过。
- **缓存清理**：`frontend/.next` 已在验证前清理。
- **签署状态**：G0 已由主协调者正式签署通过（P12.0.3 单点切换隔离验证完成，证据见 11.2），不再处于待签署状态。

### 11.2 P12.0.3 环境变量隔离方案验证证据（验证责任：主协调者）

- **方案落地**：迁移开关采用运行时环境变量 `UI_SHELL`（取值 `old` / `new`），由 `frontend/docker-entrypoint.sh` 在容器启动时写入 `frontend/public/runtime-config.js`（`window.__RUNTIME_CONFIG__.UI_SHELL`），浏览器运行时经 `frontend/lib/runtime-config.ts` 的 `getUiShell()` 读取；`app/page.tsx` 客户端挂载后再切换，初始统一旧壳避免 SSR 与客户端 hydration 不一致。该机制复用仓库已验证的 runtime-config 链路，未引入新依赖。
- **回退安全**：`UI_SHELL` 未设置/为空或任意非法值时，`parseUiShell` 一律安全回退旧壳 `old`（含 SSR 端 window 未定义场景）；仅显式 `UI_SHELL=new` 时选择新壳。默认与非法输入均不误伤生产固定形态。
- **同镜像 Docker 验证**：使用同一前端镜像分别以「未设置 UI_SHELL」与「显式 UI_SHELL=new」启动容器，核对运行时生成的 runtime-config 中 `UI_SHELL` 取值与页面实际渲染壳层一致；验证通过后临时容器已清理，`frontend/.next` 已在验证前清理。
- **测试与门禁**：`frontend/lib/runtime-config.test.ts` 专项 11 项全部通过（未设置回退 / 显式 old / 显式 new / 非法值 / SSR 全分支）；全量 Vitest 30 个测试文件 225 项全部通过；`npm run lint` 0 errors（仅 1 条既有 `dormitory-management.tsx` 的 react-hooks/exhaustive-deps warning，非本次迁移引入）；`npx tsc --noEmit` 通过；`npm run build` 通过。
- **说明**：本小节仅记录环境变量隔离方案（P12.0.3）与 G0 前置门槛证据，不代表 P12.1 新壳或任何真实新壳已完成；不记录任何密码、令牌或实际连接串等敏感参数。

### 11.3 P12.1.5 阶段验收证据（验证责任：主协调者）

- **L1 与前端门禁**：全量 Vitest 33 个文件、255 项全部通过；`npm run lint` 为 0 error，仅保留未修改的 `dormitory-management.tsx` Hooks 依赖 warning；`npx tsc --noEmit` 与 `npm run build` 通过。
- **L2 最小集**：`rbac.spec.ts` 在 old 壳与 `UI_SHELL=new` 新壳下均为 4/4 通过。old 壳下 `rbac.spec.ts`、`feedback-closure.spec.ts`、`invoice-p73.spec.ts` 合计 9/9 通过。
- **L2 延后项**：new 壳下 `feedback-closure.spec.ts` 与 `invoice-p73.spec.ts` 未通过，不能计为通过或豁免。前者因“反馈管理”尚未接入新侧栏、且“我的反馈”仍使用旧 E2E 访问路径；后者因旧“日常事务 + 卡片”路径已变为新壳折叠分组，当前入口仍映射至 `daily-affairs`。两项归入 P12.2.1 导航接入完成后的 G2 重跑项。
- **L3 截图**：`/tmp/opencode/p12-1-5-screenshots/01-desktop-light-workbench.png`（1440×900 浅色）、`/tmp/opencode/p12-1-5-screenshots/02-desktop-dark-workbench.png`（1440×900 深色）、`/tmp/opencode/p12-1-5-screenshots/03-mobile-drawer-open.png`（390×844 移动抽屉）。自动化事实核对：`data-shell="new"` 可见；桌面侧栏 190px；移动抽屉 332px（85vw）；`html.dark` 已生效；系统标题与主内容区均可见。
- **阶段结论**：P12.1 范围内的 L1、最小 L2（RBAC）与 L3 已通过，可进入 P12.2.1。G2、G3、G4、G5、AC-6 至 AC-9 与最终切换仍未签署。

### 11.4 P12.2.1 阶段验收证据（验证责任：主协调者）

- **范围结论**：P12.2.1 仅验证“已实现旧功能”的新壳可达性与归属，不扩展为 G2-G4 或最终切换签署。
- **新壳直达已有功能**：公积金、档案、食堂、发票、办公劳保均已在新壳导航中直达既有页面；反馈管理按 `users.view`、`department.view` 控制，并沿用 `admin` / `super_admin` 兜底；系统设置保持账户菜单；审计、监控、组织明确留给 P12.2.2。
- **专项验证**：Vitest 4 个文件共 25 项通过；相关 `lint`、`tsc`、`git diff --check` 通过。
- **E2E 复核**：old 壳 `feedback + invoice` 已此前 5/5 通过；本次 `UI_SHELL=new` 下 `feedback-closure + invoice-p73` 共 5/5 通过，其中反馈 1/1、发票 4/4。
- **环境清理**：临时 `runtime-config` 与 `frontend/.next` 已清理。
- **阶段结论**：P12.2.1 已完成，但不代表 P12.2.4、G2、G3、G4 通过；下一步进入 P12.2.2。

### 11.5 P12.2.2 阶段验收证据（验证责任：主协调者）

- **专项 Vitest**：`npx vitest run components/layout/new-app-sidebar.test.tsx components/system-settings.test.tsx lib/view-mapping.test.tsx`，3 个文件共 30 项通过。
- **new 壳 E2E**：`npx playwright test e2e/rbac.spec.ts --grep "新壳"`，2/2 通过。
- **静态与差异检查**：`npm run lint` 为 0 error，仅保留既有 `dormitory-management.tsx` 的 Hooks 依赖 warning 1 条；`npx tsc --noEmit` 与 `git diff --check` 通过。
- **阶段结论**：P12.2.2 已完成；P12.2.3、G2、G3、G4 仍未完成或签署，本次证据不构成其通过结论。

### 11.6 P12.2.3 阶段验收证据（验证责任：主协调者）

- **本阶段决策（用户已确认）**：新壳只显示真实 ✅ 功能；⏳ 后续接入项（入职管理、转正管理、离职管理、人事异动共 4 项）与 🛑 临时占位项（劳动合同、奖惩记录、培训管理、合同管理、安全管理、能耗管理、车队管理、职业卫生、社保业务共 9 项）在新壳一律隐藏——不显示入口、不可点击、不展示「开发中」或 PlaceholderView；仅在迁移矩阵与 TODO 登记 P12.3 独立归零批次与升级条件（矩阵 5.2 归零五条件）。
- **验收边界**：本阶段仅作前端迁移文档与必要测试收敛——不改生产 TSX、不改 you.com 基线、不改业务逻辑 / 后端 / API / 权限定义 / 页面装配或导航生产代码；未运行无关 E2E。
- **测试收敛**：`new-app-sidebar.test.tsx` 补充断言——员工管理组 4 项 ⏳（入职 / 转正 / 离职 / 人事异动）与 3 项 🛑（劳动合同 / 奖惩记录 / 培训管理）、行政管理组 2 项 🛑（合同管理 / 安全管理）、日常事务组 4 项 🛑（能耗 / 车队 / 职业卫生 / 社保业务）均不渲染；同时断言真实 ✅ 入口正常显示（区分「社保管理」真实项与「社保业务」占位项）。
- **核查结果**：`npx vitest run components/layout/new-app-sidebar.test.tsx` 通过；`git diff --check` 通过。
- **阶段结论**：P12.2.3 已完成；本证据不代表 P12.2.4、G2、G3、G4 或最终切换通过。

### 11.7 P12.2.4 阶段验收证据（验证责任：主协调者）

- **Dock 遮挡修复专项**：专项 Vitest 8/8 通过。
- **前端静态与差异检查**：`npm run lint` 0 error（仅既有 `dormitory-management.tsx` Hooks warning 1 条）；`npx tsc --noEmit` 通过；`git diff --check` 通过；`npm run build` 通过。
- **G2 E2E（old/new 两轮）**：old 壳与 `UI_SHELL=new` 新壳两轮 E2E 均 11/11 通过（rbac / feedback-closure / invoice-p73 三个 spec）。old 壳首次执行时 feedback admin 登录超时一次（并发登录偶发超时），单 spec 重跑及完整重跑 11/11 成功；失败证据归档于 `/tmp/opencode/p12-2-4-failure-evidence/`。该偶发超时不阻塞本门槛，但 **P12.3 开始前需跟踪测试稳定性**。
- **G4 后端门禁**：`/home/hr-office/backend` 下 `go test ./...`、`go vet ./...`、`go build ./...` 全部通过。
- **G3 截图（主协调者已签署通过）**：5 张截图位于 `/tmp/opencode/p12-2-4-screenshots/`——`light-workbench.png`、`dark-workbench.png`、`light-admin-group-expanded.png`、`light-system-settings-observability.png`、`dark-system-settings-observability.png`。主协调者视觉复核结论：5 张截图全部通过——深浅主题正常、约 190px 侧栏与内容区布局正常、行政管理展开可见组织管理、系统设置内系统观察入口可见、Dock 不遮挡账户菜单、无明显溢出回归。
- **阶段结论**：P12.2.4 的 G2 / G3 / G4 已通过并签署；G5、AC-6 至 AC-9 与最终切换仍未签署。

### 11.8 P12.3.1 首批边界确认（离职管理，验证责任：主协调者）

- **用户已确认首批边界（离职管理）**：
  - 独立新壳入口与页面；
  - 复用既有 API；
  - 无审批或跨模块自动联动；
  - 恢复在职清空离职关联；
  - 未完成前矩阵离职保持 ⏳，新壳入口由实施任务接入；
  - 验收需组件、后端 handler 边界/集成与隔离 E2E。
- **状态说明**：本小节仅登记边界与计划，P12.3.1 未完成、未签署；不改变其他项状态，不涉及业务实现细节。

### 11.9 P12.3.1 离职批次验收签署（验证责任：主协调者）

- **范围**：离职管理独立新壳页面与入口已完成；后端 resign/restore 强制 `employee.edit` 鉴权；显式 seed-e2e 幂等创建 E2E 员工与 archives/resign_proof 本地存储三件套。
- **E2E**：新壳 2/2 通过（admin 完整离职→PDF 下载魔数校验→恢复；viewer 无入口）；旧壳 1 passed + 1 预期 skip（独立入口仅新壳）。
- **门禁**：后端 `go test ./... -count=1`、`go vet ./...`、`go build ./...` 全通过；前端专项 Vitest 3 文件 33/33、lint 0 error（既有 dormitory warning 1 条）、tsc、build、`git diff --check` 通过。
- **环境与清理**：隔离库仅使用 `127.0.0.1:55432/siapp_e2e`；后端进程、临时配置、`.next`、`test-results`、存储上传物已清理；测试员工已恢复 active。
- **签署状态**：本批次（离职管理）已签署通过；**生产切换未签署**，矩阵 ⏳/🛑 尚未归零（入职/转正/人事异动 3 项 ⏳ 与 9 项 🛑 仍待 P12.3 后续批次）；**G5、AC-6 至 AC-9 与最终切换均未签署**。

### 11.10 P12.3.2 入职批次验收签署（验证责任：主协调者）

- **用户已确认 P12.3.2 规则**：
  - 独立 onboarding_records / work_todos 表（不混入 employees）；
  - 入职三态与 trial/formal 正交（独立维度）；
  - 来源分手动/批量，Offer 仅字段与内部契约；
  - 定时任务 Asia/Shanghai 每日 02:00、仅扫描 planned_hire_date 等于上海当日、错过不补跑、失败不重试；
  - 身份证命中 employees 全量统一拒绝；
  - 放弃恢复保留原日期与历史字段；
  - 通用待办唯一去重且租户管理员归属；
  - 不创建账号、不跨模块自动化。
- **实施分批（5 项，全部完成）**：P12.3.2-1 模型（onboarding_records / work_todos 独立 GORM 模型 + AutoMigrate）→ P12.3.2-2 状态 API（七个：列表/创建/编辑/快速入职/确认/放弃/恢复，入职三态与 trial/formal 正交 + 权限鉴权）→ P12.3.2-3 导入/任务（手动/批量 JSON/Excel 全文件导入与模板；身份证命中 employees 全量统一拒绝；定时任务仅扫描当日；失败置 pending+日志+去重异常待办；部门 Code+三位工号生成与唯一冲突重试；同公司最早 admin/super_admin 归属）→ P12.3.2-4 页面（独立新壳页与入口）→ P12.3.2-5 验收。
- **E2E**：新壳 `e2e/onboarding.spec.ts` 2/2 通过（admin 完整入职流程；viewer 入口隐藏）；E2E 所需部门已由 seed-e2e 显式创建；E2E 仅使用 `127.0.0.1:55432/siapp_e2e` 隔离库。
- **门禁**：后端专项与全量 `go test`、`go vet ./...`、`go build ./...` 全通过；前端专项 Vitest 37/37、lint 0 error（仅既有无关 warning 1 条）、`npx tsc --noEmit`、`npm run build` 通过。
- **环境与清理**：临时 `runtime-config`、`frontend/.next`、`test-results` 与后端进程均已清理。
- **签署状态**：本批次（入职管理）已签署通过，矩阵入职管理已转 ✅；**生产切换未签署**，矩阵 ⏳/🛑 尚未归零（转正管理/人事异动 2 项 ⏳ 与 9 项 🛑 仍待 P12.3 后续批次）；**G5、AC-6 至 AC-9 与最终切换均未签署**。

### 11.11 P12.3.3 转正批次验收签署（验证责任：主协调者）

- **范围**：员工在职状态与用工状态分离；新增 `employment_status`、`probation_end_date`、`actual_regular_date`，历史 active 员工幂等回填 formal、历史 resigned 保持空值。独立转正记录保存不可变员工快照，支持手工三级审批、HR 一次延期、拒绝离职引导待办、作废后新建、Excel 批量导入及人工/自动生效。
- **定时任务与导入**：Excel 采用整文件原子校验，工号与身份证双重匹配；当日/过去日期直接生效，未来日期免审批进入排期。自动任务使用 `Asia/Shanghai` 每日 02:00，仅扫描计划日等于上海当日的记录；失败转 `effect_failed`、创建唯一异常待办并不自动重试；运行记录按业务日期唯一。
- **前端与权限**：新壳转正管理入口与页面已接入；服务端与入口统一以 `employee.edit` 鉴权；无权限用户入口隐藏且直接页面显示无权限状态。页面覆盖申请、列表/详情时间线、审批、延期、作废、人工生效、模板下载与导入警告/行错误。
- **E2E**：隔离环境仅使用 PostgreSQL `127.0.0.1:55432/siapp_e2e` 与后端 `127.0.0.1:8080`；`e2e/regularization.spec.ts` 新壳 3/3 通过：admin UI 发起→上级/HR 审批→员工 formal、HR 拒绝后员工仍 trial 且存在离职待办、viewer 无入口。测试不依赖真实等待 02:00。
- **门禁**：后端专项及全量 `go test ./... -count=1`、`go vet ./...`、`go build ./...` 通过；前端专项 Vitest 4 文件 37/37、`npx tsc --noEmit`、`npm run lint`、`npm run build`、`git diff --check` 通过。lint/build 仅保留既有未修改 `dormitory-management.tsx` Hooks warning，无错误。
- **环境与清理**：E2E 后端和开发服务器已停止；`frontend/public/runtime-config.js`、`frontend/.next`、`frontend/test-results` 已清理，未连接 5432、生产或 Supabase。
- **签署状态**：本批次（转正管理）已签署通过，矩阵转正管理已转 ✅；**生产切换未签署**，人事异动 1 项 ⏳ 与 9 项 🛑 仍待 P12.3 后续批次；**G5、AC-6 至 AC-9 与最终切换均未签署**。

### 11.12 P12.3.4 劳动合同批次验收签署（验证责任：主协调者）

- **范围**：首批仅支持固定期限劳动合同。独立 `labor_contracts` 保存员工/部门快照与可选档案 `document_id`；状态为 `draft`、`active`、`expired`、`cancelled`。草稿可编辑，生效合同不可编辑或删除，只能由 `contract.delete` 作废后新建；不改变员工状态。
- **权限与隔离**：新增 `contract.view/create/edit/delete`；入口以 `contract.view` 显示，创建、编辑/生效、作废分别按对应权限控制。后端由登录态注入租户，按合同创建时部门快照隔离；请求体不接受租户归属。
- **到期处理与附件**：复用档案文档关联，不新增合同直传；`Asia/Shanghai` 每日 02:00 扫描 active 且结束日早于当日的合同置为 expired，幂等且不修改员工状态。合同起始日不自动生效。
- **E2E**：隔离环境仅使用 PostgreSQL `127.0.0.1:55432/siapp_e2e` 与后端 `127.0.0.1:8080`；`e2e/contract.spec.ts` 2/2 通过：admin 新壳创建草稿→手动生效→作废（含作废原因必填）且数据库验证状态/原因/时间戳；viewer 无 `contract.view` 时入口隐藏。自动到期由固定时间的后端 `TestContractExpiryWorker` 覆盖，不等待真实 02:00。
- **门禁**：后端全量 `go test ./... -count=1`、`go vet ./...`、`go build ./...` 通过；前端专项 Vitest 4 文件 39/39、`npx tsc --noEmit`、`npm run lint`、`npm run build`、`git diff --check` 通过。lint/build 仅保留既有未修改 `dormitory-management.tsx` Hooks warning，无错误。
- **环境与清理**：E2E 后端和开发服务器已停止；`frontend/public/runtime-config.js`、`frontend/.next`、`frontend/test-results` 已清理，未连接 5432、生产或 Supabase。
- **签署状态**：本批次（劳动合同）已签署通过，矩阵劳动合同已转 ✅；**生产切换未签署**，人事异动 1 项 ⏳ 与 8 项 🛑 仍待 P12.3 后续批次；**G5、AC-6 至 AC-9 与最终切换均未签署**。

### 11.13 P12.3.5 行政合同批次验收签署（验证责任：主协调者）

- **范围**：独立于劳动合同和宿舍合同，覆盖全部外部主体。`admin_contracts` 必填合同编号、名称、相对方、类型、起止日期；选填含税金额、币种、负责人、备注与档案 `document_id`。状态为 `draft`、`active`、`expired`、`cancelled`；草稿可编辑，draft/active 均可填写原因后作废并新建替代合同。
- **权限与隔离**：新增 `admin_contract.view/create/edit/delete`；入口按 view，创建、编辑/生效、作废按对应权限控制。viewer 不持有 `admin_contract.view`，历史角色关联由生产 seed 与 E2E seed 幂等清理；后端由登录态注入租户，不接受请求体中的归属字段。
- **到期处理、附件与工作台**：复用档案文档关联，不新增直传；`Asia/Shanghai` 每日 02:00 扫描 active 且结束日早于当日的合同置为 expired，幂等且不联动其他业务。工作台统一提醒新增 `admin_contract_expiring`，仅展示 30 日内到期的 active 合同；合同起始日不自动生效。
- **E2E**：隔离环境仅使用 PostgreSQL `127.0.0.1:55432/siapp_e2e` 与后端 `127.0.0.1:8080`；后端以显式 `SIAPP_DB_*` 参数连接该隔离库，避免 seed 的 `SIAPP_DATABASE_PATH` 与主服务配置不一致。`e2e/admin-contract.spec.ts` 2/2 通过：admin 新壳创建草稿→手动生效→工作台显示“行政合同到期”→作废原因必填→作废落库；viewer 无 `admin_contract.view` 时入口隐藏。自动到期由固定时间的 `TestAdminContractExpiryWorker` 覆盖，不等待真实 02:00。
- **门禁**：后端全量 `go test ./... -count=1`、`go vet ./...`、`go build ./...` 通过；前端专项 Vitest 5 文件 52/52、`npx tsc --noEmit`、`npm run lint`、`npm run build`、`git diff --check` 通过。lint/build 仅保留既有未修改 `dormitory-management.tsx` Hooks warning，无错误。
- **环境与清理**：E2E 后端和开发服务器已停止；`frontend/public/runtime-config.js`、`frontend/.next`、`frontend/test-results` 已清理，未连接 5432、生产或 Supabase。
- **签署状态**：本批次（行政合同）已签署通过，矩阵合同管理已转 ✅；**生产切换未签署**，人事异动 1 项 ⏳ 与 7 项 🛑 仍待 P12.3 后续批次；**G5、AC-6 至 AC-9 与最终切换均未签署**。

### 11.14 P12.3.6 奖惩记录批次验收签署（验证责任：主协调者）

- **范围**：独立 `reward_records` 台账，支持奖励（`reward`）与惩罚（`punishment`）两类。必填员工、记录类型、发生日期、事由、等级；选填分值、金额、负责人、档案 `document_id`、备注。创建时冻结员工姓名、部门和岗位快照；不改变员工状态或薪资。
- **状态与权限**：状态为 `draft`、`effective`、`voided`；草稿可编辑，`reward.edit` 手动生效，draft/effective 均可由 `reward.delete` 填写原因后作废。新增 `reward.view/create/edit/delete`，入口与写操作按对应权限控制；查询/写入由登录态注入租户并按部门快照隔离。viewer 不持有 `reward.view`，生产 seed 和 E2E seed 均会幂等清理历史残留。
- **附件**：复用档案文档关联并验证文档归属当前租户，不新增奖惩直传附件能力。
- **E2E**：隔离环境仅使用 PostgreSQL `127.0.0.1:55432/siapp_e2e` 与后端 `127.0.0.1:8080`；后端以显式 `SIAPP_DB_*` 参数连接隔离库。`e2e/reward.spec.ts` 2/2 通过：admin 新壳创建奖励草稿→手动生效→作废原因必填→作废落库，并确认员工仍为 active；viewer 无 `reward.view` 时入口隐藏。
- **门禁**：后端全量 `go test ./... -count=1`、`go vet ./...`、`go build ./...` 通过；前端专项 Vitest 4 文件 40/40、`npx tsc --noEmit`、`npm run lint`、`npm run build`、`git diff --check` 通过。lint/build 仅保留既有未修改 `dormitory-management.tsx` Hooks warning，无错误。
- **环境与清理**：E2E 后端和开发服务器已停止；`frontend/public/runtime-config.js`、`frontend/.next`、`frontend/test-results` 已清理，未连接 5432、生产或 Supabase。
- **签署状态**：本批次（奖惩记录）已签署通过，矩阵奖惩记录已转 ✅；**生产切换未签署**，人事异动 1 项 ⏳ 与 6 项 🛑 仍待 P12.3 后续批次；**G5、AC-6 至 AC-9 与最终切换均未签署**。

### 11.15 P12.3.7 人事异动批次验收签署（验证责任：主协调者）

- **范围**：独立 `personnel_changes` 台账，支持调岗（`transfer`）、晋升（`promotion`）、降级（`demotion`）。新增员工可选职级 `job_level`；记录员工部门、岗位、职级的变更前后快照。必填员工、类型、生效日期、事由，异动后部门、岗位、职级至少变更一项。
- **状态与权限**：状态为 `draft`、`effective`、`voided`；草稿可编辑，具备 `employee.edit` 权限的用户手动生效，生效时才同步员工部门、岗位、职级；草稿和已生效记录均可填写原因作废。所有接口均按登录态租户和部门范围隔离，不引入审批流、薪资或员工状态联动。
- **一致性与作废**：生效在数据库事务内重新读取员工并校验其当前部门、岗位、职级与异动前快照一致，否则返回冲突且不更新；已生效记录作废仅作废台账，不回滚员工资料，避免覆盖后续异动。
- **E2E**：隔离环境仅使用 PostgreSQL `127.0.0.1:55432/siapp_e2e` 与后端 `127.0.0.1:8080`；后端以显式 `SIAPP_DB_*` 参数连接隔离库。`e2e/personnel-change.spec.ts` 2/2 通过：admin 新壳创建晋升草稿→手动生效→API 验证员工部门/岗位/职级更新→作废并验证资料不回滚；viewer 无 `employee.edit` 时入口隐藏。
- **门禁**：后端全量 `go test ./... -count=1`、`go vet ./...`、`go build ./...` 通过；前端专项 Vitest 4 文件 42/42、`npx tsc --noEmit`、`npm run lint`、`npm run build`、`git diff --check` 通过。lint/build 仅保留既有未修改 `dormitory-management.tsx` Hooks warning，无错误。
- **环境与清理**：E2E 后端和开发服务器已停止；`frontend/public/runtime-config.js`、`frontend/.next`、`frontend/test-results` 已清理，未连接 5432、生产或 Supabase。
- **签署状态**：本批次（人事异动）已签署通过，矩阵人事异动已转 ✅；**生产切换未签署**，六项 🛑 占位仍待 P12.3 后续批次；**G5、AC-6 至 AC-9 与最终切换均未签署**。

### 11.16 P12.3.8 培训管理批次验收签署（验证责任：主协调者）

- **范围**：独立 `training_records` 台账，支持内部（`internal`）、外部（`external`）、线上（`online`）培训；状态为 `draft`、`completed`、`voided`。必填培训主题、类型、日期；选填讲师或机构、关联员工、结果、备注。关联员工时冻结姓名、部门、岗位快照，不更新员工资料。
- **权限与隔离**：新增 `training.view/create/edit/delete`；入口按 view，创建按 create，草稿编辑/手动完成按 edit，草稿删除与草稿/已完成作废按 delete。后端由登录态注入租户，并按员工部门快照隔离；viewer 不持有 `training.view`，生产与 E2E seed 幂等清理历史关联。
- **不在范围**：不实现报名、审批、签到、证书、附件、批量导入、定时任务或培训学分/员工字段联动。
- **E2E**：隔离环境仅使用 PostgreSQL `127.0.0.1:55432/siapp_e2e` 与后端 `127.0.0.1:8080`；后端以显式 `SIAPP_DB_*` 参数连接隔离库。`e2e/training.spec.ts` 2/2 通过：admin 新壳创建内部培训草稿→手动完成→填写原因作废，并通过 API 验证关联员工部门、岗位、职级、状态均未变化；viewer 无 `training.view` 时入口隐藏。
- **门禁**：后端全量 `go test ./... -count=1`、`go vet ./...`、`go build ./...` 通过；前端专项 Vitest 4 文件 43/43、`npx tsc --noEmit`、`npm run lint`、`npm run build`、`git diff --check` 通过。lint/build 仅保留既有未修改 `dormitory-management.tsx` Hooks warning，无错误。
- **环境与清理**：E2E 后端和开发服务器已停止；`frontend/public/runtime-config.js`、`frontend/.next`、`frontend/test-results` 已清理，未连接 5432、生产或 Supabase。
- **签署状态**：本批次（培训管理）已签署通过，矩阵培训管理已转 ✅；**生产切换未签署**，五项 🛑 占位仍待 P12.3 后续批次；**G5、AC-6 至 AC-9 与最终切换均未签署**。

### 11.17 P12.3.9 安全管理批次验收签署（验证责任：主协调者）

- **范围**：独立 `safety_inspections` 台账，检查类型仅例行（`routine`）与专项（`special`）；状态为 `draft`、`completed`、`voided`。必填检查类型、检查日期、检查地点、负责人、问题描述、整改要求。
- **权限与隔离**：新增 `safety.view/create/edit/delete`；入口与列表/详情按 view，创建按 create，草稿编辑/手动完成按 edit，草稿删除与草稿/已完成作废按 delete。后端由登录态注入租户；viewer 不持有 `safety.view`，生产与 E2E seed 幂等清理历史关联。
- **不在范围**：不实现隐患分派、审批、附件、定时任务、批量导入或员工联动。
- **契约修复**：初次隔离 E2E 复现前端将“例行检查”中文值提交给后端、而后端仅接受 `routine/special` 的 400 问题。已将前端改为下拉选择并提交标准值，台账仍显示中文；保存失败时对话框保持打开，避免误导用户。
- **E2E**：隔离环境仅使用 PostgreSQL `127.0.0.1:55432/siapp_e2e` 与后端 `127.0.0.1:8080`；后端以显式 `SIAPP_DB_*` 参数连接隔离库。`e2e/safety.spec.ts` 2/2 通过：admin 新壳创建例行检查草稿→手动完成→填写原因作废，并通过 API 核验状态与作废原因；viewer 无 `safety.view` 时入口隐藏。
- **门禁**：后端全量 `go test ./... -count=1`、`go vet ./...`、`go build ./...` 通过；前端专项 Vitest 4 文件 45/45、`npx tsc --noEmit`、`npm run lint`、`npm run build`、`git diff --check` 通过。lint/build 仅保留既有未修改 `dormitory-management.tsx` Hooks warning，无错误。
- **环境与清理**：E2E 后端和开发服务器已停止；`frontend/public/runtime-config.js`、`frontend/.next`、`frontend/test-results` 已清理，未连接 5432、生产或 Supabase。
- **签署状态**：本批次（安全管理）已签署通过，矩阵安全管理已转 ✅；**生产切换未签署**，四项 🛑 占位仍待 P12.3 后续批次；**G5、AC-6 至 AC-9 与最终切换均未签署**。

### 11.18 P12.3.10 车队管理批次验收签署（验证责任：主协调者）

- **范围**：仅独立 `fleet_vehicles` 车辆档案；必填车牌号（租户内唯一）、车型与状态，选填品牌、座位数、购置日期与备注。状态为 `active`（启用）和 `inactive`（停用）；启用车辆可编辑或删除，停用车辆不可编辑或删除，仅能恢复为启用。
- **权限与隔离**：新增 `fleet.view/create/edit/delete`；入口与列表/详情按 view，创建按 create，编辑与恢复按 edit，删除按 delete。后端由登录态注入租户；viewer 不持有 `fleet.view`，生产与 E2E seed 幂等清理历史关联。
- **不在范围**：不实现调度、加油、维修、审批、轨迹、附件、定时任务或员工联动。
- **E2E**：隔离环境仅使用 PostgreSQL `127.0.0.1:55432/siapp_e2e` 与后端 `127.0.0.1:8080`；后端以显式 `SIAPP_DB_*` 参数连接隔离库。`e2e/fleet-vehicles.spec.ts` 2/2 通过：admin 新壳新增启用车辆→编辑→停用，验证编辑/删除不可用→恢复启用→删除，并通过 API 核验；viewer 无 `fleet.view` 时入口隐藏。
- **门禁**：后端全量 `go test ./... -count=1`、`go vet ./...`、`go build ./...` 通过；前端专项 Vitest 4 文件 44/44、`npx tsc --noEmit`、`npm run lint`、`npm run build`、`git diff --check` 通过。lint/build 仅保留既有未修改 `dormitory-management.tsx` Hooks warning，无错误。
- **环境与清理**：E2E 后端和开发服务器已停止；`frontend/public/runtime-config.js`、`frontend/.next`、`frontend/test-results`、`frontend/playwright-report` 已清理，未连接 5432、生产或 Supabase。
- **签署状态**：本批次（车队管理）已签署通过，矩阵车队管理已转 ✅；**生产切换未签署**，三项 🛑 占位仍待 P12.3 后续批次；**G5、AC-6 至 AC-9 与最终切换均未签署**。

### 11.19 P12.3.11 职业卫生批次验收签署（验证责任：主协调者）

- **范围**：独立 `occupational_health_checks` 职业健康检查台账；必填员工、检查日期、医疗机构、检查类别，选填检查结论、下次检查日期和备注。创建或编辑时冻结员工姓名、部门、岗位快照，不更新员工资料。状态为 `draft`、`completed`、`voided`；草稿可编辑、完成或删除，草稿和已完成记录均可填写原因作废。
- **权限与隔离**：新增 `occupational_health.view/create/edit/delete`；入口与列表/详情按 view，创建按 create，草稿编辑/手动完成按 edit，草稿删除及草稿/已完成作废按 delete。后端由登录态注入租户，并按员工部门快照隔离；viewer 不持有 `occupational_health.view`，生产与 E2E seed 幂等清理历史关联。权限排序为 114–117，不与安全管理和车队管理权限冲突。
- **不在范围**：不实现附件、异常干预、复查提醒、审批、批量导入、定时任务或员工资料联动。
- **契约修复**：前后端统一 JSON 字段为 `medical_institution`、`check_conclusion` 与 `employee_name/employee_department/employee_position` 快照，避免后端旧字段名导致创建失败或页面快照为空。
- **E2E**：隔离环境仅使用 PostgreSQL `127.0.0.1:55432/siapp_e2e` 与后端 `127.0.0.1:8080`；后端以显式 `SIAPP_DB_*` 参数连接隔离库。`e2e/occupational-health.spec.ts` 2/2 通过：admin 新壳选在职员工创建职业健康检查草稿→手动完成→填写原因作废，并通过 API 核验员工快照、状态和作废原因；viewer 无 `occupational_health.view` 时入口隐藏。
- **门禁**：后端全量 `go test ./... -count=1`、`go vet ./...`、`go build ./...` 通过；前端专项 Vitest 4 文件 47/47、`npx tsc --noEmit`、`npm run lint`、`npm run build`、`git diff --check` 通过。lint/build 仅保留既有未修改 `dormitory-management.tsx` Hooks warning，无错误。
- **环境与清理**：E2E 后端和开发服务器已停止；`frontend/public/runtime-config.js`、`frontend/.next`、`frontend/test-results`、`frontend/playwright-report` 已清理，未连接 5432、生产或 Supabase。
- **签署状态**：本批次（职业卫生）已签署通过，矩阵职业卫生已转 ✅；**生产切换未签署**，两项 🛑 占位仍待 P12.3 后续批次；**G5、AC-6 至 AC-9 与最终切换均未签署**。

### 11.20 P12.3.12 社保业务批次验收签署（验证责任：主协调者）

- **范围**：不新增重复的“社保业务”模块，直接复用既有 `insurance` 社保管理能力；移除 `daily-affairs-hub.tsx` 中无真实挂载的 `social` 占位卡片，保留新壳行政管理 → 社保管理真实入口与 `InsuranceManagement` 多 tab 页面。
- **权限与隔离**：现有社保变更、选项、导入、模板、回盘列表/上传/清空接口统一补齐服务端 `insurance.view/create/edit/delete` 鉴权：读取使用 view，创建/导入/上传使用 create，更新/处理使用 edit，删除/清空使用 delete；保留 JWT 与 `user_id` 租户隔离。
- **E2E**：隔离环境仅使用 PostgreSQL `127.0.0.1:55432/siapp_e2e` 与后端 `127.0.0.1:8080`；后端以显式 `SIAPP_DB_*` 参数连接隔离库。`e2e/insurance-management.spec.ts` 3/3 通过：admin 新壳可见并进入真实社保管理、页面及日常事务无“社保业务”占位；admin API 创建增员并 GET 列表核验；viewer 保留 `insurance.view` 可查看真实页面，但创建接口返回 403。
- **门禁**：后端专项 `insurance_rbac_test.go` 覆盖管理员正常路径、viewer 无写权限、所有社保路由权限拒绝及跨租户隔离；后端全量 `go test ./... -count=1`、`go vet ./...`、`go build ./...`、`git diff --check` 通过。前端相关回归 Vitest 2 文件 27/27、`npx tsc --noEmit`、`npm run lint`、串行 `npm run build`、`git diff --check` 通过；lint 仅保留既有未修改 `dormitory-management.tsx` Hooks warning，无错误。
- **环境与清理**：E2E 后端和开发服务器已停止；`frontend/public/runtime-config.js`、`frontend/.next`、`frontend/test-results`、`frontend/playwright-report` 已清理，未连接 5432、生产或 Supabase。一次并行构建曾因 E2E 清理 `.next` 产生 `/_document` 缺失，串行重跑构建通过，确认属于环境竞争而非源码错误。
- **签署状态**：本批次（社保业务）已签署通过，已合并至矩阵社保管理 ✅；**生产切换未签署**，仅能耗管理 🛑 占位仍按用户决策隐藏；**G5、AC-6 至 AC-9 与最终切换均未签署**。

### 11.21 P12.3.13 能耗管理批次验收签署（验证责任：主协调者）

- **范围**：仅实现宿舍水、电能耗只读统计；按抄表日期所属自然月与楼栋筛选，展示整体、楼栋和房间三级用量/费用。复用 `DormMeterReading` 与既有 `dormitory.view`，不统计燃气，不新增录入、账单、预测、告警、模型或权限资源。
- **权限与隔离**：新增 `GET /api/dormitories/energy/summary`，由 `dormitory.view` 鉴权；按 JWT 注入 `user_id` 严格隔离。仅聚合 `charge_details` 中 electric/water 的合法非负 `usage` 与 `amount`；燃气、缺失值、NaN/Inf、负数与不可解析明细忽略。支持 `month=YYYY-MM` 和 `building_id` 查询参数，参数非法返回 400。
- **前端**：新壳日常事务新增受 `dormitory.view` 控制的真实“能耗管理”入口。独立只读页面支持月份/楼栋筛选、整体水电指标、楼栋汇总和房间下钻；无新增、编辑、删除、保存、账单、预测、告警或燃气界面。
- **E2E**：隔离环境仅使用 PostgreSQL `127.0.0.1:55432/siapp_e2e` 与后端 `127.0.0.1:8080`；后端以显式 `SIAPP_DB_*` 与本地 `JWT_SECRET_KEY` 参数连接隔离库。`e2e/energy-management.spec.ts` 3/3 通过：admin 可经日常事务进入真实页面且无写操作/燃气；admin 经既有宿舍 API 创建唯一园区、楼栋、房间和含燃气明细的抄表数据，API 核验只聚合水电并在页面完成房间下钻；viewer 保留 `dormitory.view`，入口可见、页面可达、GET 汇总返回 200。
- **门禁**：后端专项 `TestDormEnergySummary` 7/7、全量 `go test ./... -count=1`、`go vet ./...`、`go build ./...`、`git diff --check` 通过。前端专项 Vitest 4 文件 47/47、`npx tsc --noEmit`、`npm run lint`、串行 `npm run build`、`git diff --check` 通过；lint/build 仅保留既有未修改 `dormitory-management.tsx` Hooks warning，无错误。
- **环境与清理**：E2E 后端与开发服务器已停止；`frontend/public/runtime-config.js`、`frontend/.next`、`frontend/test-results`、`frontend/playwright-report` 已清理，未连接生产或 Supabase。
- **签署状态**：本批次（能耗管理）已签署通过，矩阵能耗管理已转 ✅；P12.3 全部占位已归零。**G5、AC-1 至 AC-9、最终切换与生产切换均未签署**。

### 11.22 最终切换前占位残留处置（验证责任：主协调者）

- **范围**：移除系统设置中无调用的 `AdvancedOptions` 预留组件，不改任何已实现的系统设置入口、权限门控或真实字段条件显示功能。
- **处置**：已移除“字段分组配置（待开发）”“条件显示规则（待开发）”及“该功能正在内测中，敬请期待”等未接入占位 UI；对应系统设置回归测试保留反向断言，防止占位复发。
- **门禁**：`components/system-settings.test.tsx` 6/6、`npx tsc --noEmit`、`npm run lint`、`npm run build`、`git diff --check` 通过。lint/build 仅保留既有未修改 `dormitory-management.tsx` Hooks warning，无错误。
- **残留检测**：`frontend/components` 与 `frontend/app` 中“功能开发中|敬请期待”源码文本零命中；测试中的反向断言不构成运行时占位残留。
- **签署状态**：占位残留阻塞已消除；**G5、AC-1 至 AC-9、最终切换与生产切换仍未签署**。

### 11.23 P12.3.3 最终切换门槛核对与签署（验证责任：主协调者）

- **环境与隔离**：后端明确连接独立 PostgreSQL 测试库；全量 E2E 前执行并等待 `seed-e2e` 成功，确保人事异动等可写测试数据恢复基线。全量 Playwright 结果为 **37 项通过、4 项条件跳过、0 项失败**。
- **AC-1 / AC-2**：全量 E2E 与视觉取证确认新壳侧栏、内容区、底部管理条和浮动 Dock 正常渲染；导航可进入真实页面并保持激活态。
- **AC-3 / AC-4**：Vitest **58 个测试文件、373 项测试**通过；全量 RBAC E2E 覆盖 admin、manager、editor、viewer 及新壳组织管理可见性。
- **AC-5**：视觉取证 `test-results/screenshots/final-shell-light.png` 与 `test-results/screenshots/final-shell-dark.png` 经人工复核，浅深主题区分明确、文本可读、无布局破相。标题字体外观一致；运行时断言标题第一个子节点计算样式 `animation-name` 为 `rolling-text`，确认既有动效生效。
- **AC-6**：既有 RBAC、反馈闭环、发票管理链路均在全量 E2E 中通过。反馈闭环因外部模型网络不可达，使用用户批准的临时本地 OpenAI 兼容 SSE 模拟服务验证真实聊天消息→差评→管理员回复→关闭→用户端状态同步的系统内闭环；执行结束后隔离库模型配置已还原，模拟服务已停止。
- **AC-7**：前端 `npm run lint`、`npx tsc --noEmit`、`npm run build` 通过；lint/build 仅保留既有未修改 `dormitory-management.tsx` Hooks warning，无 error。后端 `go test ./... -count=1`、`go vet ./...`、`go build ./...` 通过。
- **AC-8 / G5**：在隔离 worktree `omos/g5-rollback-3c18d29` 中，以 `3c18d29` 执行 `npm ci && npm run build` 成功。验证计划 G5 条目的 `df4278b^` 解析为该提交；11.1 的 `72f0243` 与此存在历史记录矛盾，最终按更具体的 G5 条目执行真正迁移前基准。主工作区未切换或回滚。
- **AC-9**：迁移矩阵所有状态均为 ✅，无 ⏳ / 🛑。`frontend/components` 与 `frontend/app` 运行时代码中“功能开发中|敬请期待”零命中；测试中的反向断言不属于运行时残留。
- **签署结论**：P12.3.3 迁移验收已签署通过。**本次签署不包含 P12.3.4**：生产固定新壳、移除 `UI_SHELL` 迁移开关、提交与推送均尚未执行，必须在用户后续明确授权后另行实施。

### 11.24 P12.3.4 生产固定新壳与本地提交（验证责任：主协调者）

- **授权范围**：用户明确授权固定新壳、完成验证并创建本地 Git 提交；不推送，也不清理 G5 临时 worktree。
- **生产形态**：`app/page.tsx` 已无条件装配 `NewAppSidebar`、`ManagementBar` 的 `new` 变体、受控全局搜索、内容区工具条、嵌入式 `ChatPanel` 与 `NewShell`。旧壳装配分支已移除。
- **运行时配置**：`runtime-config.ts` 与 `docker-entrypoint.sh` 已删除 `UI_SHELL`，保留 API 地址配置与读取能力。前端 TypeScript、JavaScript、Shell 源码中 `UI_SHELL`、`UiShell`、`getUiShell`、`parseUiShell` 零命中；E2E 不再注入迁移开关。
- **前端门禁**：`npm run test` 为 **58 个测试文件、365 项测试通过**；`npm run lint` 为 0 errors，仅保留既有 `dormitory-management.tsx` Hooks warning；`npx tsc --noEmit`、`npm run build` 与 `git diff --check` 通过。
- **固定新壳 E2E**：隔离后端 `127.0.0.1:8080` 连接 E2E PostgreSQL，`final-shell-visual.spec.ts` 与 `rbac.spec.ts` 串行执行 **7/7 通过**，覆盖浅深主题、标题动效、四角色权限、组织管理可达性与隐藏性。
- **后端门禁**：`go test ./... -count=1`、`go vet ./...`、`go build ./...` 通过。
- **签署结论**：P12.3.4 已完成，生产最终形态固定为新壳。本地提交创建后不推送；G5 回退 worktree 保留为既有证据。
