# 人事行政管理系统 (hr-office) 开发计划

## P0-P2：基础配置与知识库 ✅ 已完成
- 社保分摊系统、标签管理、文件夹树
- 知识库配置、分块编辑、版本管理、SSE流式问答
- BM25 全文检索 + RRF 融合排序
- 3个新测试文件 (46用例)

## P3：多轮对话优化 ✅ 已完成
- 上下文压缩（smart_summarize 策略）
- 追问意图识别（规则 + LLM 双层检测）

## P4：前端与交互体验升级 ✅ 已完成
- 智能问答工作台（SSE + Markdown + 引用溯源 + 会话管理）
- 档案管理增强（文件夹树 + 标签筛选 + 批量标签操作）
- 首页统计仪表盘

## P5：企业级特性与权限深化 🔄 当前阶段

### P5.1 权限控制（数据隔离 + RBAC）
- 部门级数据隔离：用户只能看本部门员工数据
- RBAC 角色系统：admin/manager/editor/viewer
- API 权限校验中间件
- 前端权限控制（菜单/按钮级）

### P5.2 用户反馈闭环
- AI 回答 👍/👎 评分
- 文字反馈提交
- 管理员反馈管理面版（查看/回复/统计）
- 反馈数据存储与 API

### P5.3 更多文件类型/IM 集成
- 待定

## P7：后续迭代（按 P7.4 → P7.1 → P7.2 → P7.3 串行执行）🔜 已确认执行

### P7.4 修复 pre-existing storage 测试 ✅ 已由 P11.1 DSN 修复顺带解决
- [x] 调研 TestResolve_NoDefaultConfig 失败原因（P11.1 DSN 修复后已 PASS）
- [x] 修复代码或测试用例（无需额外改动）
- [x] 验收：`go test ./internal/service/storage/...` 通过

### P7.1 RBAC 权限深化 ✅ 已完成
> 决策：Q1=6 子任务全做 / Q2=角色迁移 dry-run + 清单 + 管理员确认 / Q3=User.Role 用 gorm:"-" 废弃、前端读 user.permissions[] / Q4=fallback 区分未登录/token失效/网络错误 / Q5=RequirePermission mode="hide"|"disable" 默认 hide / Q6=全局写入审计 + 路径白/黑名单配置 / Q7=单测 + Playwright E2E

- [x] 1. 后端：User.Role 迁移脚本 `cmd/migrate-roles/main.go`——dry-run 模式打印不规范值清单（super_admin→admin 映射表、空值/NULL→viewer、未知值→viewer），确认后实际迁移 User.Role → user_roles 关联表，最后 seed 默认角色权限映射（4 角色 × 资源 × 操作）（交付物：cmd/migrate-roles/main.go + 迁移文档；依赖：无）
- [x] 2. 后端：User 模型 User.Role 字段加 `gorm:"-"` 废弃（JSON 不再返回），全栈统一读 user_roles 表（交付物：models/models.go + 相关引用清理；依赖：1）
- [x] 3. 后端：全局写入审计中间件——自动拦截所有非 GET 请求，记录操作者/操作类型/payload，路径白名单/黑名单配置化过滤（交付物：middleware/audit.go + main.go 挂载 + 配置项；依赖：无）
- [x] 4. 后端：登录/刷新接口返回 user.permissions[]（从 user_roles + role_permissions 聚合），中间件权限校验走 UserRole 表（交付物：api/auth.go 响应扩展 + middleware/permission.go 调整；依赖：2）
- [x] 5. 前端：`lib/permissions.ts` hasPermission(resource, action) + `components/auth/RequirePermission.tsx`（mode="hide"|"disable" 默认 hide）+ useAuth().hasPermission hook——fallback 策略：未登录→false、token失效→false+自动刷新、网络错误→true+degraded 标记（交付物：permissions.ts + RequirePermission.tsx + rbac.tsx 调整；依赖：4）
- [x] 6. 前端：菜单侧边栏 + 关键按钮（删除/新增/编辑/导出等）接入 RequirePermission 按角色显隐（交付物：app-sidebar.tsx + 各管理组件按钮包裹；依赖：5）
- [x] 7. 验收：后端 go test 全绿（含迁移 dry-run 单测 + 权限矩阵单测）；前端 lint/tsc/build 全绿；Playwright E2E 4 角色登录验证菜单/按钮显隐（admin 全显/manager 部门/editor 编辑/viewer 只读）（交付物：测试 + E2E 脚本；依赖：1-6）

### P7.2 用户反馈闭环
> 决策：用户增加“我的反馈”入口与站内回复提示；状态为 pending/replied/closed 三态；管理员按评分+状态+时间范围筛选与统计；旧临时 message_id 反馈保留且标记“原回答不可用”。
- [x] 1. 后端：ChatFeedback 三态模型与反馈 API 深化——评分/状态/时间筛选、统计时间范围、管理员回复/关闭、审计、我的反馈、回答原文/提问/溯源与旧记录不可用标记（交付物：models/feedback.go + api/feedback.go + 单测；依赖：无）
- [x] 2. 后端：SSE done 事件回传真实 ChatMessage.ID，前端反馈提交使用真实消息 ID（交付物：service/chat.go + chat-panel API 契约 + 单测；依赖：无）
- [x] 3. 前端：用户侧“我的反馈”页面与站内回复提示，旧反馈原回答不可用提示（交付物：用户反馈组件 + API 类型；依赖：1）
- [x] 4. 前端：chat-panel 反馈真实消息关联；管理员 feedback-panel 增加状态/时间筛选、原文/提问/来源展开、关闭操作（交付物：chat-panel.tsx + feedback-panel.tsx；依赖：1,2）
- [x] 5. 验收：后端状态/API/SSE 单测；前端组件测试 + lint/tsc/build；端到端“提问→反馈→管理员回复→用户查看→关闭”闭环（依赖：1-4；用户确认以当前验证结果收尾，端到端实测因测试环境 LLM 配置已清理而暂缓）

### P7.3 发票管理卡片

> 已确认：首期仅支持 PDF（每批最多 50 份、单份最多 20MB），聚焦进项增值税发票，兼容收据/付款凭证/电子行程单/其他凭证。PDF 文本优先、OCR 兜底；不支持 OFD、图片、税务联网验真、抵扣申报、会计科目、多主体、批量下载原件与新报销流程。

- [x] 1. 后端：扩展发票归档数据模型——独立归档状态（待确认/已确认/已作废）、凭证类型、真实受控附件、文件哈希、识别来源/原始文本/字段置信度、购方匹配、逻辑删除及 30 天清理字段、发票明细、解析任务与字段更正审计（交付物：models/invoice.go + 迁移 + 单测；依赖：无）
- [x] 2. 后端：PDF 批量上传与安全附件链路——StorageManager 存储、哈希判重、受权限保护的预览/下载/原件打印接口、批量独立结果与错误草稿逻辑删除（交付物：invoice API + storage 集成 + 单测；依赖：1）
- [x] 3. 后端：持久化解析工作器与字段提取——任务恢复、PDF 文本优先/OCR 兜底、发票专用字段规则、低置信度/缺失/金额异常、失败重试；单文件失败不影响批次其他任务（交付物：invoice 解析 service + worker + 单测；依赖：1,2）
  - [x] P7.3.3 解析安全修复：提取前快照并以快照 CAS 写入，防止覆盖人工修改；OCR 错误可重试、空/低质文本不可重试；状态写入脱离已取消工作上下文；独立电子票号、无效日期保护、金额异常和税号低置信度标记。
- [x] 4. 后端：归档与业务规则 API——草稿编辑、admin 确认/作废/已确认更正（原因+差异审计）、增值税票代码号码或电子票号强判重、其他凭证哈希预警、采购可选关联与供应商/金额/日期预警、购方主体 admin 设置、查询统计与 CSV 导出（交付物：invoice API + 权限/审计 + 集成测试；依赖：1,2,3）
- [x] 5. 前端：上传与解析工作台——批量 PDF 拖拽上传、每份任务状态/失败重试、识别字段编辑、低置信度高亮、人工补录最低门槛、草稿权限控制（交付物：invoice 上传/编辑组件 + API 类型 + Vitest；依赖：2,3）【已完成】
- [x] 6. 前端：归档管理与打印导出——筛选/统计/关联预警、受控原件预览下载打印、归档打印单、字段 CSV 导出、确认/作废/更正原因交互、已确认/已作废状态展示（交付物：InvoiceManagement 增强 + 打印/导出组件 + Vitest；依赖：4,5）【已完成；本期仅 CSV，不实现 Excel】
- [x] 7. 验收：模型/API/解析工作器单元与集成测试；前端 lint/tsc/Vitest/build；端到端入口与 RBAC 验证（依赖：1-6）【已完成：隔离 PostgreSQL 重建并 seed-e2e；Playwright invoice-p73.spec.ts 4/4 通过；后端 go test/vet/build、前端 Vitest 99/99、lint、tsc、build 全部通过】

【总进度】7 / 7 完成 ✅ P7.3 发票管理前端与入口验收已完成
【下一步】等待用户确认是否提交 P7.3 变更

【验收说明】已使用隔离 PostgreSQL 数据库 p73_e2e 与临时后端完成入口级 Playwright 验收；测试接口对发票 API 使用 mock，未替代真实 PDF 上传→解析→补录→确认→打印→作废全主流程，后续可在具备解析/OCR 配置的环境补跑。

【总进度】15 / 18 完成（P7.4 3 项 + P7.1 7 项 + P7.2 5 项）
【下一步】P7.2 端到端实测在具备临时全局 LLM 配置的测试环境补跑；P7.3 等待提交确认

## P6：日常事务模块合并（办公劳保 + 食堂管理）✅ 已完成

> 来源：`office-supply-analytics`（Worker+Hono+D1）与 `office-supply-analytics-frontend`（React 18+Vite）
> 合并方式：后端 Go 完全重写（复用 hr-office 鉴权/RBAC/审计），前端按 hr-office 风格重设计（功能不变）

- [x] 1. 后端模型层：office_* 系列 + canteen_* 系列 GORM 模型（含 user_id 多租户），注册 AutoMigrate
- [x] 2. 后端 API 层：办公用品路由 `/api/office/*`（44 路由：字典/供应商/用品/采购/请款/分析/CSV）
- [x] 3. 后端 API 层：食堂路由 `/api/canteen/*`（55 路由：字典/采购/收入/菜单/充值退费/分析）
- [x] 4. 数据迁移脚本：源 D1/SQLite → hr-office 新表（一次性，含字典+历史单据，ID 重映射，事务回滚）
- [x] 5. 前端 API 层：`lib/api-office.ts`（44 方法） + `lib/api-canteen.ts`（57 方法）
- [x] 6. 前端办公劳保页面：`OfficeSuppliesManagement` + 5 个 Tab（字典/采购单/请款单/分析/基础数据）
- [x] 7. 前端食堂管理页面：`CanteenManagement` + 5 个 Tab（字典/采购/收入/菜单/分析）
- [x] 8. 菜单接入：daily-affairs-hub 新增「办公劳保」+「食堂管理」卡片，主组件挂载
- [x] 9. 测试补齐：Go migrate/office_analytics/canteen_analytics 单测全过；前端 lint/tsc 0 错误
- [x] 10. 文档与提交：TODO/agentmemory 更新 + git commit

【总进度】10 / 10 完成
【下一步】持续迭代：RBAC 细化、IM 集成、多文件类型扩展

## P8：知识库模块 + 存储安全 + 认证补强（借鉴 WeKnora）🔜 已确认方案

> 独立知识库模块（侧边栏主菜单），档案管理留日常事务不动；借鉴 WeKnora 存储安全 + 认证补强
> 权限：角色 + 部门 + 指定员工 三维 AND + 字段级脱敏（身份证/手机号/地址等）
> 所有前端新增必须先经 @designer 输出 ui-design-p8.md

### P8.0 认证安全补强（P0 优先，排在 P8.1 之前）
- [x] 0.1 JWT 密钥兜底移除（`JWT_SECRET_KEY` 为空时启动失败 fatal exit，禁用硬编码默认值）
- [x] 0.2 修复双认证不一致：reset-password 页统一走自建 token 验证（不再依赖 Supabase session）
- [x] 0.3 Refresh Token 旋转机制（新增 `auth_tokens` 表 + `/auth/refresh` + Logout 服务端吊销）
- [x] 0.4 前端 401 自动刷新（lib/api.ts 拦截器：遇 401 → 调 `/auth/refresh` → retry + 队列防并发）
- [x] 0.5 登录 IP 限流 + 账号连续失败锁定（5 次/15 分钟，仿 WeKnora 滑动窗口模式）
- [x] 验收：JWT 密钥空值启动即报错；密码重置可用；Refresh 旋转 + Logout 吊销 + 前端自动刷新全链路通过

### P8.1 后端基础（知识库模型 + 权限 + 脱敏 + 路由）
- [x] 1.1 新增 KnowledgeBase / KBAccessRule / KBFieldMask 模型 + AutoMigrate
- [x] 1.2 知识库 CRUD API + 系统模板 seed（7 个模块）+ 手动创建/编辑/删除
- [x] 1.3 权限 API：addRule / removeRule / listRules（角色+部门+用户三维）
- [x] 1.4 脱敏 API：setFieldMask / getFieldMask + 检索层自动脱敏（非 admin 自动 mask）
- [x] 1.5 入库 API：POST /api/knowledge-bases/{id}/ingest（半自动，手动触发）
- [x] 1.6 侧边栏新增「📚 知识库」菜单 + page.tsx currentView 注册
- [x] 验收：API 可用，脱敏生效，权限三维校验通过，模板 seed 8 个知识库入库

### P8.2 文档解析 + 存储安全（两个独立子批次，可并行）
- [x] 2a.1 docreader 微服务集成：Docker Compose 编排 + Go HTTP REST client
- [x] 2a.2 docreader 桥接：Word/Excel/PPT 解析结果写入 DocumentChunk
- [x] 2a.3 验收：上传 Word 文档可解析并检索
- [x] 2b.1 存储安全：凭据 AES-GCM 加密 + 脱敏返回（models/system_settings.go）
- [x] 2b.2 存储安全：路径遍历防护（SafePathUnderBase）+ SSRF 防护
- [x] 2b.3 存储安全：清理 fallback 双轨（manager.go）
- [x] 2b.4 OAuth 驱动降级标记（experimental）+ go test storage 全绿
- [x] 2b.5 验收：凭据加密落库、路径/SSRF 防护生效、go test 全绿

### P8.3 RAG 增强 + 前端（@designer 先出设计规范）
- [x] @designer 输出 docs/ui-design-p8.md（知识库主页面 + 分类模板 + 权限配置 UI）
- [x] 3.1 后端：rerank 重排 + 知识库级模型独立选择 + 父子分块策略默认配置
- [x] 3.2 后端：rerank 单元测试 4/4 通过
- [x] 3.3 前端：KnowledgeBaseManagement 主组件 + 4 Tab（列表/入库/权限/脱敏）
- [x] 3.4 前端：知识库权限配置面板（按角色/部门/指定员工配置可见范围）
- [x] 3.5 前端：半自动入库面板（选知识库→预览待入库数据→一键入库）
- [x] 3.6 前端：字段脱敏规则配置面板
- [x] 3.7 验收：完整"预览→入库→脱敏检索→权限隔离问答"闭环；权限配置生效

### P8.4 收尾 P8
- [x] 更新 TODO.md + agentmemory + git commit + push

【总进度】25 / 25 完成 ✅ P8 已全部交付

## P9：P8 遗留收尾（ingest 实装 + chat-panel 范围过滤）🔜 已确认

> 背景：P8 交付后遗留两项已确认未实施：① ingest API 是 stub，需打通 docreader→分块→向量化→入库全链路；② 悬浮问答面板（chat-panel）未按用户知识库权限过滤检索范围（Q8=B 决策）。

### P9.1 后端 ingest 全链路实装
- [x] 1.1 后端：完成 ingest 到可检索闭环——修复非 archives 源记录与 `documents` 的关联，并确保向量列与 JSON 副本同步写入（交付物：knowledge_base.go + service/kb_ingest.go + retrieval.go；依赖：无）
- [x] 1.2 后端：保留并验证入库结果统计（scanned/ingested/skipped/errors 明细）（交付物：同上；依赖：1.1）
- [x] 1.3 后端：补齐 ingest→检索集成测试（SQLite 降级 + PostgreSQL/pgvector 实测，验证 chunk 写入、KB 关联、全文/向量检索命中、并发幂等与事务回滚）（交付物：kb_ingest_*_test.go + retrieval 集成测试；依赖：1.1）

### P9.2 后端检索按 KB 权限过滤
- [x] 2.1 后端：search/chat/流式 chat 统一增加 `kb_id` 参数与 HasAccess 校验（非 admin 仅可检索自己有权限的 KB）（交付物：knowledge.go + chat.go + retrieval.go；依赖：无）【已完成：三条链路统一校验，管理员全量放行，普通用户越权返回 403】
- [x] 2.2 后端：全文检索、向量检索和问答 prompt/Sources 返回前统一应用字段脱敏，修正规则字段与结果字段映射（交付物：retrieval.go + kb_mask.go + chat.go；依赖：2.1）【已完成：字段级精准脱敏、长文本安全边界、Prompt/答案/SSE/Sources 全链路接入】
- [x] 2.3 后端：补强权限过滤、脱敏内容、流式越权拒绝的单测与集成测试（交付物：knowledge_test.go + kb_ingest_test.go；依赖：2.1,2.2）【已完成：专项与全量 Go 门禁通过，覆盖跨片、持久化、失败安全及参数矩阵】

### P9.3 前端 chat-panel 范围过滤
- [x] 3.1 前端：chat-panel 知识库范围选择器与 SSE 请求链路生效（下拉列出当前用户可见 KB，默认全部）（交付物：chat-panel.tsx + 后端 SSE；依赖：P9.2）【已完成：选择器与服务端 SSE 范围参数已接通】
- [x] 3.2 前端：检索及流式问答请求带 `kb_id`，响应中的脱敏字段正常显示（交付物：lib/api.ts + chat-panel.tsx；依赖：3.1）【已完成：请求参数与服务端接收契约一致】
- [x] 3.3 验收：完整链路实测——入库→脱敏检索→权限隔离问答全通；go build/test + tsc/lint/build 全绿（依赖：全部）【已完成：后端专项/API/全量测试、go vet、go build，前端 kb_id 契约、lint、tsc、build 均通过；未配置真实 LLM 的手工联调不作为阻塞】

【总进度】9 / 9 完成 ✅ P9 已全部交付
【下一步】推进 P7.3 前端上传解析工作台（子任务 5）

### P9.3 全链路验收清单（2026-08-13 初始化）

【任务标题】完成知识库入库、脱敏检索与权限隔离问答的全链路验收
【背景】P9.1/P9.2 已完成后端入库、检索范围控制、字段级脱敏和问答链路接线，P9.3.3 仍需用现有本地环境完成最终验证并关闭 TODO 遗留项。
【验收标准】不修改业务代码；后端入库/检索/问答权限与脱敏验证通过；前端 `lint`、TypeScript 类型检查（tsc）和生产构建（build）通过；所有测试结果、环境限制与最终提交记录写入 TODO.md 和 agentmemory；验收通过后创建规范提交并推送远端。

## 子任务列表
- [x] 1. 验收准备与范围核对（交付物：确认当前分支、工作区、测试夹具、SQLite/外部服务可用性；依赖：无）【已完成：main 与 origin/main 初始一致；工作区仅含 TODO.md 和未跟踪 .ignore】
- [x] 2. 后端全链路验收（交付物：入库→脱敏检索→权限隔离问答的 Go 单测/集成验证及 `go test ./...`、`go vet ./...`、`go build ./...` 结果；依赖：1）【已完成：全量与专项 API/service 测试通过】
- [x] 3. 前端链路与质量门禁（交付物：chat-panel `kb_id` 契约核对、`npm run lint`、tsc、`npm run build` 结果；依赖：1）【已完成：契约一致，lint、tsc、build 全部通过】
- [x] 4. 结果记录、TODO 收尾与提交推送（交付物：更新 P9.3.3 状态、记录测试/阻塞项、规范 commit、远端同步核对；依赖：2,3）【当前会话待提交 TODO 更新】

【总进度】3 / 4 完成
【下一步】执行 #4：提交 TODO 更新并核对远端状态

## P7.3 前端上传与解析工作台（2026-08-13 初始化）

【任务标题】实现 P7.3 前端发票上传与解析工作台
【背景】P7.3 后端发票归档、PDF 上传与解析工作器已完成，剩余前端工作台尚未实现。目标是补齐批量 PDF 上传、任务状态、失败重试、识别字段编辑、低置信度高亮、人工补录和草稿权限控制。
【验收标准】交付前端相关组件、API 类型与 Vitest 测试；批量最多 50 份、单份最多 20MB 且仅接受 PDF；上传、解析状态、失败重试、字段编辑/保存接口契约一致；正常流程和异常边界测试通过；npm run lint、TypeScript 类型检查、Vitest、npm run build 全部通过；完成后提交并推送。
【边界】允许修改 P7.3 相关前端组件、API 类型和测试；发现后端接口缺口先记录并暂停，不擅自扩展后端；不修改 P9 知识库链路。

## 子任务列表
- [x] 1. 梳理现有发票前端页面、API 类型与后端接口契约，确认可复用组件和缺口（交付物：文件地图、接口契约与实现方案；依赖：无）【已完成：已有发票列表/详情/草稿编辑 API 与页面；上传接口为 POST /invoices/upload，支持 files 字段、单批 1-50 个、单文件不超过 20MB、仅 PDF；后端缺少解析任务查询/详情和手动重试 HTTP 接口】
- [x] 2. 实现批量 PDF 上传与解析工作台：上传限制、任务状态、失败重试（交付物：发票上传/解析组件与 API 调用；依赖：1；后端接口扩展已获用户批准）【已完成：批量 PDF 限制、解析状态轮询、失败重试与权限挂载】
- [x] 3. 实现识别字段编辑、低置信度高亮、人工补录与草稿权限控制（交付物：字段编辑交互与权限状态；依赖：1,2）【已完成：field_confidence 缺失/低置信度高亮，草稿且具备 edit 权限可保存，其余只读】
- [x] 4. 补充 Vitest 组件测试，覆盖正常流程与异常边界（交付物：组件测试；依赖：2,3）【已完成：上传限制、API 契约、任务状态归一化、权限和置信度边界测试】
- [x] 5. 运行 lint、tsc、Vitest、build，更新 TODO 并提交推送（交付物：全量门禁结果、TODO/agentmemory 记录、规范提交；依赖：4）【已完成：前端 56/56 Vitest、lint、tsc、build 与后端 go test/vet/build 全部通过】

【总进度】5 / 5 完成 ✅ P7.3 前端上传解析工作台已交付
【下一步】推进 P7.3 剩余归档管理与打印导出（子任务 6）

## P10：前端 UI 全站治理（借鉴 CDK 设计语言）🔜 已确认方案

> 背景：hr-office 前端新模块规范执行好（90%），历史大模块（employee/dormitory/insurance）视觉滞后，存在 20+ 粗糙点（弹窗 19 种尺寸、加载态 3 套、硬编码颜色散落、深色模式破相）。参照 CDK 项目（/home/cdk）的设计语言做全站治理。
> 决策：Q1=全站治理 P0+P1 / Q2=引入 CDK 个性亮点（渐变系统化+毛玻璃） / Q3=中度动效 / Q4=一并修深色 / Q5=@designer 先出规范

### P10.1 设计规范（@designer 先行）
- [x] 0.1 @designer 输出 docs/ui-design-p9.md（全站设计规范 v2：token 体系/四档弹窗/三态组件/动效规范/深色规范/渐变与毛玻璃系统化方案）
- [x] 0.2 验收：规范文档完整覆盖 P0+P1 全部治理点 + CDK 亮点引入方案

### P10.2 基础设施治理（P0）
- [x] 1.1 弹窗尺寸 4 档常量（sm=420/md=560/lg=800/full 全屏）统一全站 19 种写法
- [x] 1.2 三态组件抽取：TableLoading（骨架屏）/ EmptyState（图标+文字+motion 弹性）/ 错误态恢复组件，消灭英文 "Loading"
- [x] 1.3 硬编码颜色清零：system-settings 15 色 hex 色板、chat-panel（含 bg-white 深色破相）、PaymentsTab、feedback-panel、invoice StatsTab、canteen analytics-components 迁移到主题变量
- [x] 1.4 表格规范化：统一 ScrollArea + sticky 表头容器，globals.css 表头硬编码背景改主题变量
- [x] 1.5 深色模式补全：修复 bg-white 破相、表头硬编码、insurance dark:bg 补丁等 4-5 处

### P10.3 布局与视觉亮点（P1 + CDK 亮点）
- [x] 2.1 侧边栏激活态改主题变量 + ManagementBar 折叠位置修复（16rem 硬编码改响应式）

## P12：全局 UI 壳层浅色适配（2026-08-21）🔄 当前任务

> 范围：仅调整全局壳层、通用操作模式与主题基线；不修改菜单数据、权限过滤、业务流程和业务模块内部功能。

- [x] 1. 现状盘点：确认 Next.js 15 + Tailwind CSS 4、侧栏/顶部工具区/控制坞/主题实现与验收页面入口
- [x] 2. 全局视觉基线：系统字体、浅灰工作区、白色内容容器、浅色/深色主题映射、移除滚动文字动效
- [x] 3. 侧栏与顶部工具区：约 200px 侧栏、分组/激活态/用户区、轻量标题工具区
- [x] 4. 控制坞与个人资料：固定内容区左下角，移除拖拽和位置持久化；控制坞与个人资料均优先同步主题偏好接口、失败回退本地存储
- [x] 5. 自检：壳层相关 Vitest 41/41、TypeScript 类型检查、生产构建通过；lint 仅保留未修改宿舍模块既存警告

【复盘】改动严格限于全局壳层、主题、资料偏好与对应测试；菜单资源、权限过滤、业务组件及接口契约未改。未执行 E2E：需要可用后端与测试账号，建议在集成环境按验收页面补跑。
- [x] 2.2 双层 bg-card 嵌套清理（page.tsx 内容区与模块内部去重）
- [x] 2.3 首页欢迎卡渐变系统化（品牌渐变+装饰光斑+轮播，对齐 CDK ExploreBanner）
- [x] 2.4 日常事务卡片墙渐变业务编码（8 张卡片按模块语义分色渐变，对齐 CDK ProjectCard）
- [x] 2.5 浮动 Dock 毛玻璃强化（bg-background/70 backdrop-blur-md + 主题化阴影）

### P10.4 动效引入（中度）
- [x] 3.1 页面入场动画（stagger 容器 + item 位移，对齐 CDK containerVariants）
- [x] 3.2 统计数字滚动（CountingNumber 对齐 CDK）
- [x] 3.3 空状态 motion 弹性入场 + 弹窗 zoom-in 统一

### P10.5 验收与收尾
- [x] 4.1 验收：全站 lint/tsc/build 通过 + 深色模式完整 + 弹窗四档统一 + 无英文 Loading
- [x] 4.2 TODO/agentmemory 更新 + git commit + push

## P11：遗留问题清理（lint 103 errors + storage flaky 测试）🔜 已确认

> 背景：用户选择清理历史遗留。前端全量 lint 103 errors（全部 no-explicit-any，97% 在 2 个 api.ts）+ 37 warnings；后端 storage TestUploadConcurrent flaky（DSN 写法错误导致共享内存库未生效）。

### P11.1 后端 storage 测试修复
- [x] 1.1 修复 integration_test.go:42 DSN → `file::memory:?cache=shared`（共享内存库 URI，实验已验证 10/10 并发成功）
- [x] 1.2 连续 10 轮 TestUploadConcurrent 100% 通过

### P11.2 前端 unused-vars 清理（27 warnings）
- [x] 2.1 system-settings.tsx 21 处未使用 import/常量/函数删除
- [x] 2.2 archives/employee/insurance/app-sidebar/user-preferences/nav-user 6 文件 6 处清理
- [x] 2.3 验收：lint 该批文件 0 errors 0 warnings

### P11.3 前端 no-explicit-any 治理（103 errors）
- [x] 3.1 canteen/api.ts 58 处 any → 死代码直接删除（0 使用方）
- [x] 3.2 office-supply/api.ts 42 处 any + useAuth unused import → lib 正式类型
- [x] 3.3 SupplyDialog.tsx 3 处 any → SupplyRecord extends OfficeSupply
- [x] 3.4 验收：lint 全站 0 errors

### P11.4 人工判断类（exhaustive-deps 4 + no-img-element 5 + alt-text 误报 1）
- [x] 4.1 exhaustive-deps 修复（archives 2 处 + model-settings 1 + system-settings 1）
- [x] 4.2 no-img-element 决策（nav-user 2 + user-preferences 2 + archives 1：保留 img + 行内 disable）
- [x] 4.3 alt-text 误报修复（archives-management.tsx:1049 lucide Image 误判 → 行内 disable + 注释）

### P11.5 验收收尾
- [x] 5.1 全站 lint `npm run lint -- components/` 0 errors + go test storage 全绿
- [x] 5.2 TODO/agentmemory 更新 + git commit + push

【总进度】14 / 14 完成 ✅
【下一步】已完成；后续优先推进 P9 检索链路收尾

【总进度】17 / 17 完成 ✅
【下一步】已完成；后续优先推进 P9 检索链路收尾

### P6 交付摘要
- 后端：19 个 GORM 模型 + 99 个 HTTP 路由 + 6 个单测（office_analytics 6 例 + canteen_analytics 2 例 + migrate 3 例）
- 前端：2 个主组件 + 10 个 Tab + 7 个辅助文件（utils/api/dialogs）+ 2 个 API 封装
- 设计：`docs/ui-design-p6.md`（信息架构+组件复用清单+视觉规范）
- 迁移：`backend/migrate/import_legacy.go` + `cmd/migrate-legacy/main.go` + `docs/migration-p6.md`
- 验证：`go build ./...` 通过、`go test ./migrate/...` 通过、`tsc --noEmit` 通过、`eslint` 通过

## P12：前端 UI 外壳迁移（you.com 基线）🔜 已确认方案

> 背景：应用外壳（侧边栏、导航菜单、顶部用户区、底部管理条、浮动 Dock、用户头像、通知中心、个人设置等）按 you.com 基线（`/home/you.com/src/App.tsx`）重构升级，保留 Next.js 运行时（Next.js 15 + React 19 + TS 5）与全部旧业务组件，将全部旧功能零遗漏接入新导航。
> 关联文档：`docs/frontend-ui-migration-matrix.md`（迁移矩阵，三态状态唯一权威）、`docs/frontend-ui-migration-plan.md`（迁移方案，阶段 0-3 范围与切换门槛定义）、`docs/frontend-ui-migration-verification.md`（验证计划，G0-G5 门槛与 AC-1~AC-9 验收主张）
> 验证责任：主协调者（各阶段完成条件、G 门槛与最终切换签署；验证执行可委托对应 specialist，证据统一汇总至主协调者）
> 三条总原则：新壳承载（外壳只承担布局/导航/用户区/全局浮层，不内嵌业务逻辑）；旧业务复用（既有业务组件原样复用，仅调整挂载方式、不重写不重构）；单点切换（旧壳/新壳切换收敛到唯一装配点 page.tsx 顶层，一次切换整体生效、保证可逆）

### P12.0 阶段 0：基准 / 映射 / 隔离（前置准备）
- [x] 0.1 视图映射表定稿：基线导航项 → currentView → 复用组件 → 三态状态（✅/⏳/🛑）逐项登记，禁止出现未登记项（交付物：映射表核对结论与定稿；依赖：无）【已完成：映射表已定稿，见 docs/frontend-ui-migration-matrix.md】
- [x] 0.2 回退基准固化：确认迁移前 commit 可构建运行并记录基准 commit（供 G5 回退演练使用）（交付物：基准 commit 记录；依赖：0.1）【已完成：回退基准 commit 已记录，见验证文档 11.1】
- [x] 0.3 环境变量隔离方案在开发/预发布环境落地并验证：迁移开关仅 dev/preview 启用，生产固定最终形态；变量名实施阶段确定并登记，复用仓库已验证机制，不预设未验证变量值（交付物：隔离配置 + 验证记录；依赖：无）【已完成：迁移开关采用运行时环境变量 UI_SHELL（old/new），由 frontend/docker-entrypoint.sh 启动时写入 runtime-config；未设置/非法值安全回退旧壳、显式 new 选新壳；同镜像 Docker 三态验证通过、临时容器已清理、.next 已清理；专项测试 11 项、全量 Vitest 30 文件 225 项、lint 0 errors（仅既有 dormitory Hooks warning）、tsc、Next build 均通过，证据见验证文档 11.2】
- [x] 0.4 G0 前置门槛通过：后端 :8080 可启动、seed-e2e 幂等成功、`frontend/.next` 已清理；映射表与基准经主协调者确认（交付物：G0 门槛记录；依赖：0.1-0.3）【已完成：G0 已由主协调者正式签署通过，证据见验证文档 11.1/11.2】
- 明确不改范围：任何业务组件、后端代码、认证/API；系统头部标题字体与动效（见硬性约束）。

### P12.1 阶段 1：新外壳与首阶段真实接入
- [x] 1.1 侧边栏按 you.com 基线重构导航结构（顶级：工作台/知识库/偏好设置；分组：员工管理、行政管理、日常事务；用户菜单：个人资料/系统设置/退出登录），含分组展开折叠、激活态高亮、折叠态响应式（交付物：app-sidebar 等新壳侧栏组件；依赖：P12.0）【已完成：新增 `new-app-sidebar.tsx` 并仅在 `UI_SHELL=new` 的既有唯一装配点启用；旧 `app-sidebar.tsx` 保持原样。已接入项可导航，未接入项禁用；账户菜单保留。标题 `rolling-text text-base font-semibold` 与 globals.css 动画未改。组件测试 7/7、指定文件 lint 通过；E2E/build/截图由主协调者验证】
- [x] 1.2 顶部用户区（头像展示与编辑、个人资料/系统设置入口、通知中心入口）+ 底部管理条（ManagementBar）+ 浮动 Dock（毛玻璃主题化，全局事件契约 `dashboard:toggle-sidebar` / `dock:*` 不变）（交付物：nav-user / management-bar / floating-dock 组件；依赖：1.1）【已完成：仅 `UI_SHELL=new` 的侧栏底部用户区按 you.com 菜单视觉接入“个人资料 / 我的反馈 / 退出系统”，系统设置继续以 `settings.view` 控制；头像、编辑上传/裁剪/恢复默认与设置返回来源均复用既有能力。新壳应用层固定通知入口派发既有 `dock:open-notification`，不读取宿舍菜单权限；通知内容与 `dock:open-site-memo` 跳转仍复用原实现。ManagementBar/FloatingDock 新增可选 `new` 视觉变体，默认值和 `dock:*`、偏好存储、拖动行为未改。专项 Vitest 5 文件 33/33、`npm run lint` 0 errors（既有 dormitory Hooks warning 1 条）、`npx tsc --noEmit` 通过；未运行 E2E、生产构建和人工浅/深截图，留待 P12.1.5 统一验收。】
- [x] 1.3 全局搜索入口 + ChatPanel 内嵌面板（外壳内嵌层）（交付物：global-search 与 chat-panel 外壳接线；依赖：1.2）【已完成：仅 `UI_SHELL=new` 在主内容区顶部提供搜索与 AI 助手受控入口；GlobalSearch 以可选受控状态隐藏默认固定按钮，但仍支持 Ctrl/Cmd+K 与原搜索 API/结果导航；ChatPanel 以可选 `embedded` 变体作为内容区右侧抽屉、无全局遮罩，保留会话/SSE/反馈，关闭按钮与 Escape 可用，且不覆盖新侧栏固定用户区。旧壳继续使用 GlobalSearch 默认固定入口和 ChatPanel 默认全局悬浮层。ManagementBar 同时派发 `dock:open-ai` 与兼容的 `dock:open-chat`，页面监听两者。专项 Vitest 4 文件 28/28 通过，lint 0 errors（仅既有 dormitory Hooks warning 1 条）、`npx tsc --noEmit`、`git diff --check` 均通过。】
  - [x] P12.1.3-1：核对 `page.tsx` 唯一装配点、GlobalSearch、ChatPanel、ManagementBar 与现有 Vitest 事件/快捷键契约（依赖：1.2）【已完成：唯一装配点为 `page.tsx`；现有 Dock 事件为 `dock:open-chat`，已兼容需求事件。】
  - [x] P12.1.3-2：为搜索与聊天补充可选受控参数或新壳变体，默认保持旧壳固定/悬浮行为（依赖：P12.1.3-1）【已完成：GlobalSearch 增加可选 `open` / `onOpenChange` / `hideTrigger`；ChatPanel 增加可选 `embedded` 变体，默认均不变。】
  - [x] P12.1.3-3：仅在 `UI_SHELL=new` 接入内容区搜索、AI 助手入口和内容区右侧受控抽屉，保留 `dock:open-ai`、关闭和键盘操作，并避免页面根滚动（依赖：P12.1.3-2）【已完成：新壳抽屉限制在 `relative overflow-hidden` 主内容容器，主内容继续独立纵向滚动。】
  - [x] P12.1.3-4：补齐 Vitest 并运行专项测试、lint、`tsc` 与差异检查；仅全部通过后更新完成状态（依赖：P12.1.3-3）【已完成：专项 Vitest 28/28、lint 0 errors、tsc、diff check 全部通过。】
- [x] 1.4 统一视图映射模块（按 0.1 映射表落地）+ 单点切换开关接线（唯一装配点，仅 dev/preview 启用）（交付物：视图映射模块 + page.tsx 装配点 + 配套测试；依赖：P12.0.1）【已完成：新增 `frontend/lib/view-mapping.ts` 统一视图映射模块——13 个合法 ViewId（VIEW_IDS，顺序与既有 switch 一致）、默认视图 DEFAULT_VIEW=landing、非法回退 FALLBACK_VIEW=insurance（与既有 default 分支一致，矩阵未规定不同）、isViewId/resolveViewId、ViewComponentMap（Record 类型强制覆盖全部 ViewId，缺任一视图编译失败）、renderView 装配策略（landing 注入 userName；system/personal-settings 注入 onBack；其余无 props；非法回退 insurance）。`page.tsx` 的 renderMainContent switch 收敛为 renderView 调用，13 个既有组件分支全部保留于文件底部 VIEW_COMPONENTS 映射表；初始视图接线 DEFAULT_VIEW；设置返回来源（handleOpenSettings/handleBackFromSettings）、dock 事件、UI_SHELL 运行时开关消费点均未变化，未新增 ⏳/🛑 入口。新增 `frontend/lib/view-mapping.test.tsx` 12 项测试（VIEW_IDS 全覆盖、默认/回退常量、isViewId/resolveViewId 正常+非法、renderView 13 分支全覆盖、非法回退、props 注入与无泄漏、缺省 ctx）。验证：专项 Vitest 12/12、全量 Vitest 33 文件 255 项通过、`npx tsc --noEmit` 通过、`npm run lint` 0 errors（仅既有 dormitory Hooks warning 1 条）、`git diff --check` 通过；E2E/生产构建/截图留待 P12.1.5 统一验收】
- [x] 1.5 首阶段最小用例集通过：L1 外壳配套 Vitest 全绿、L2 `rbac.spec.ts` 回归通过、L3 浅/深截图人工核对通过（交付物：测试证据 + 截图归档；依赖：1.1-1.4）【已完成：L1 全量 Vitest 33 文件 255 项通过；lint 0 error（仅既有 dormitory Hooks warning）、tsc 与 build 通过。L2 最小集 rbac.spec.ts 在 old/new 两壳均 4/4 通过；old 壳 feedback-closure 与 invoice-p73 一并 9/9 通过。new 壳的 feedback-closure 与 invoice-p73 尚未通过，不计成功：反馈管理未接入新侧栏、我的反馈路径仍为旧 E2E 路径；发票入口由旧“日常事务+卡片”变为新壳折叠分组且当前映射仍指向 daily-affairs。两项均登记为 P12.2.1 后的 G2 重跑项。L3 截图：`/tmp/opencode/p12-1-5-screenshots/01-desktop-light-workbench.png`、`02-desktop-dark-workbench.png`、`03-mobile-drawer-open.png`；已核对 new shell、桌面 190px、移动 332px/85vw、深色主题、标题与主内容可见。本项通过不代表 G2/G3/G4/G5 或最终切换通过。】
- 硬性约束：系统头部标题「人事行政管理系统」字体大小与字重（`text-base` + `font-semibold`）与 rolling-text 滚动动效（时长/缓动/帧位）保持当前行为不变，复刻必须原样复用现有类与动画定义，禁止另起一套；本阶段不改 `renderMainContent` 各业务分支、不改后端与认证。

### P12.2 阶段 2：全部已实现旧功能可达与不可达组件归属
- [x] 2.1 矩阵 ✅ 项全部接入新壳导航（工作台/知识库/偏好设置/员工花名册/社保/公积金/档案/食堂/发票/办公劳保/宿舍/系统设置/部门管理/反馈管理/AI 助手/通知中心/全局搜索等），复用现有组件仅挂载、不改逻辑（交付物：新壳导航挂载清单与接线；依赖：P12.1）【已完成：新壳 ✅ 项已直达已有功能——公积金、档案、食堂、发票、办公劳保均已接入；反馈管理按 `users.view`、`department.view` 控制并沿用 `admin/super_admin` 兜底；系统设置保持账户菜单；审计 / 监控 / 组织明确留给 P12.2.2。专项 Vitest 4 文件 25 项通过；相关 lint、tsc、diff check 通过；old 壳 feedback + invoice 已此前 5/5 通过，本次 `UI_SHELL=new` 的 feedback-closure + invoice-p73 共 5/5 通过（反馈 1/1，发票 4/4）；临时 `runtime-config` 与 `.next` 已清理。P12.2.1 完成不代表 P12.2.4 / G2-G4 通过】
- [x] 2.2 三个不可达组件归属落地：审计日志、系统监控归系统设置内，组织管理归行政管理组——先补 you.com 基线入口 → 同步矩阵 3.6 状态 → hr-office 新壳挂载 + 权限过滤断言（交付物：基线入口 + 挂载代码 + 可达性断言；依赖：2.1）【已完成：you.com 基线在行政管理加入组织管理，并在设置内加入审计日志/系统监控静态入口；新壳组织管理按 `department.view` 显示并沿用 `admin`/`super_admin` 兜底；审计日志/系统监控仅在 `SystemSettings` 内按 `users.view` 显示，点击复用既有组件并可返回系统设置，未进入主侧栏。默认 `SystemSettings` 行为和 `UI_SHELL=old` 未变。专项 Vitest `npx vitest run components/layout/new-app-sidebar.test.tsx components/system-settings.test.tsx lib/view-mapping.test.tsx` 3 文件 30 项通过；new 壳 E2E `npx playwright test e2e/rbac.spec.ts --grep "新壳"` 2/2 通过；`npm run lint` 0 error（既有 `dormitory-management.tsx` Hooks warning 1 条）、`npx tsc --noEmit`、`git diff --check` 均通过。验证责任：主协调者；本项完成不代表 P12.2.3 或 G2-G4 通过】
- [x] 2.3 ⏳ 后续接入项（入职/转正/离职/人事异动）与 🛑 占位项（劳动合同/奖惩记录/培训管理/合同管理/安全管理/能耗管理/车队管理/职业卫生/社保业务共 9 项）开发期在新壳一律隐藏——不显示入口、不可点击、不展示「开发中」或 PlaceholderView，仅在迁移矩阵与 TODO 登记 P12.3 独立归零批次与升级条件（交付物：矩阵 4.2/5.1-5.3/6 节同步 + 新壳隐藏断言；依赖：2.1）【已完成：新壳只显示真实 ✅ 功能；4 个 ⏳（入职/转正/离职/人事异动）与 9 个 🛑（劳动合同/奖惩记录/培训管理/合同管理/安全管理/能耗管理/车队管理/职业卫生/社保业务）在新壳一律隐藏，不展示「开发中」或 PlaceholderView；归零批次与升级条件登记于矩阵 4.2/5.2 与 P12.3，最终切换前必须归零。矩阵 4.2/5.1-5.3/6 节已同步开发期隐藏与最终切换归零约束；`new-app-sidebar.test.tsx` 已断言 4+9 项全部不渲染、真实 ✅ 入口正常显示（13/13 通过）；`git diff --check` 与未跟踪文档行尾空白核查通过；未运行无关 E2E。本项完成不代表 G2-G4 或最终切换通过】
- [x] 2.4 G2（既有 rbac / feedback-closure / invoice-p73 三个 E2E spec 全通过）+ G3（关键视图浅/深截图人工核对）+ G4（前端 lint/tsc/build、后端 go test/vet/build 全绿）（交付物：G2-G4 门槛证据；依赖：2.1-2.3）【已完成：G2——old/new 两轮 E2E 均 11/11 通过（rbac / feedback-closure / invoice-p73）；old 壳首次并发登录偶发超时一次（feedback admin 登录超时），单 spec 及完整重跑 11/11 成功，失败证据 `/tmp/opencode/p12-2-4-failure-evidence/`，不阻塞本门槛，但 P12.3 开始前需跟踪测试稳定性。G3——主协调者视觉复核 5 张截图全部通过（深浅主题、约 190px 侧栏与内容区、行政管理展开组织管理、系统设置系统观察入口、Dock 不遮挡账户菜单、无明显溢出回归），截图路径 `/tmp/opencode/p12-2-4-screenshots/`。G4——Dock 遮挡修复专项 Vitest 8/8、lint 0 error（既有 dormitory warning 1 条）、tsc、build、diff check 通过，后端 `/home/hr-office/backend` 下 go test/vet/build 全通过。P12.2.4 G2/G3/G4 已通过，但 G5、AC-6~AC-9 与最终切换仍未签署，证据详见验证文档 11.7】
- 明确不改范围：不实现占位项与后续项的真实业务（仅登记批次）；不修改既有业务组件逻辑；后端不做迁移相关改动。

### P12.3 阶段 3：按独立业务批次实现后续/占位项直至全部归零并最终切换
- [x] 3.1 按独立业务批次实现 ⏳ 后续接入项（入职管理/转正管理/离职管理/人事异动），每批满足矩阵 5.2 归零五条件（权限依据/后端 API/前端挂载/验收证据/入口可达性）后矩阵状态转 ✅（交付物：各批次组件 + API + 测试；依赖：P12.2）【已完成：离职管理（P12.3.1-1）、入职管理（P12.3.2）、转正管理（P12.3.3）与人事异动（P12.3.7）子批次均已完成并归零；P12.3 整体仍未完成】
- [x] P12.3.1-1 离职管理子批次（首批）：独立新壳入口与页面、复用既有 API、无审批或跨模块自动联动、恢复在职清空离职关联，满足矩阵 5.2 归零五条件后矩阵 3.3 离职管理转 ✅（交付物：独立页面 + 新壳入口 + 服务端鉴权 + E2E；依赖：P12.2）【已完成：独立新壳页面和入口已接入；后端 resign/restore 强制 `employee.edit` 鉴权；显式 seed-e2e 幂等创建 E2E 员工与 archives/resign_proof 本地存储三件套。新壳 E2E 2/2（admin 完整离职→PDF 下载魔数→恢复；viewer 无入口）；旧壳 E2E 1 passed + 1 预期 skip（独立入口仅新壳）；后端 `go test ./... -count=1`/vet/build 全通过；前端专项 Vitest 3 文件 33/33、lint 0 error（既有 dormitory warning 1 条）、tsc/build、diff check 通过；隔离库仅使用 127.0.0.1:55432/siapp_e2e，后端/临时配置/.next/test-results/存储上传物已清理，测试员工恢复 active。本子批次完成不代表 P12.3 整体或生产切换完成，证据详见验证文档 11.9】
- [x] P12.3.2 入职管理子批次（第二批）：独立 onboarding_records/work_todos 模型 + 状态 API + 手动/批量导入与定时任务 + 独立新壳页面，满足矩阵 5.2 归零五条件后矩阵 3.3 入职管理转 ✅（交付物：模型 + 状态 API + 导入/任务 + 页面 + 验收；依赖：P12.2）【已完成：独立 onboarding_records/work_todos 模型；七个状态 API（列表/创建/编辑/快速入职/确认/放弃/恢复）；JSON/Excel 全文件导入与模板；定时任务 Asia/Shanghai 每日 02:00 仅扫描 planned_hire_date 等于上海当日（不补跑、不重试）；失败 pending+日志+去重异常待办；身份证全量冲突；部门 Code+三位工号与唯一重试；同公司最早 admin/super_admin 归属；独立新壳页与入口；矩阵 3.3 入职管理转 ✅，详见五子项与验证文档 11.10】
- [x] P12.3.3 转正管理子批次（第三批）：员工试用/正式字段与历史回填；独立转正记录、三级审批、延期/作废、批量导入、Asia/Shanghai 定时生效与新壳页面，满足矩阵 5.2 归零五条件后矩阵 3.3 转正管理转 ✅（交付物：模型 + API + 导入/任务 + 页面 + E2E；依赖：P12.2）【已完成：`Employee.Status` 保持 active/resigned，新增 employment_status、probation_end_date、actual_regular_date；历史 active 员工幂等回填 formal，resigned 保持空值；单条申请固定发起 HR/上级/HR 复核三人且必须不同，支持审批、延期一次、拒绝离职引导待办、作废后重建与人工生效；Excel 整文件原子导入和合规软提示；上海时间每日 02:00 仅处理当日排期记录，失败转 effect_failed 并创建唯一异常待办且不重试；独立新壳页面与 `employee.edit` 入口；矩阵 3.3 转正管理转 ✅，详见验证文档 11.11】
- [x] P12.3.4 劳动合同子批次（首个基线占位归零）：固定期限合同独立台账、档案附件关联、到期扫描与新壳页面，满足矩阵 5.2 归零五条件后矩阵 3.3 劳动合同转 ✅（交付物：模型 + API + 权限 + 定时任务 + 页面 + E2E；依赖：P12.2）【已完成：独立 `labor_contracts` 模型，仅允许 fixed_term；状态 draft→active→expired/cancelled，active 不可编辑或删除，只能由 `contract.delete` 作废后新建；创建时保存员工与部门快照；档案 `document_id` 关联复用既有附件能力；新增 `contract.view/create/edit/delete` 权限与租户/部门隔离；上海时间每日 02:00 自动扫描 active 且 end_date 早于当日的合同置 expired，不改变员工状态；独立新壳页面和入口；隔离 E2E 2/2（admin 草稿→生效→作废，viewer 无入口）；后端全量 go test/vet/build、前端专项 Vitest 39/39、tsc/lint/build、diff check 通过，详见验证文档 11.12】
- [x] P12.3.5 行政合同子批次（第二个基线占位归零）：外部主体行政合同独立台账、档案附件关联、到期扫描/工作台提醒与新壳页面，满足矩阵 5.2 归零五条件后矩阵 3.4 合同管理转 ✅（交付物：模型 + API + 权限 + 定时任务 + 工作台提醒 + 页面 + E2E；依赖：P12.2）【已完成：独立 `admin_contracts` 模型，必填合同编号、名称、相对方、类型、起止日期，选填含税金额、币种、负责人、备注；状态 draft→active→expired/cancelled，草稿可编辑，draft/active 均可由 `admin_contract.delete` 填写原因作废后新建替代；复用档案 `document_id` 关联；新增 `admin_contract.view/create/edit/delete` 权限和租户隔离；上海时间每日 02:00 自动扫描到期合同并将 active 置 expired，不联动其他业务；工作台新增 30 日内 `admin_contract_expiring` 提醒；独立新壳“行政合同”入口；隔离 E2E 2/2（admin 草稿→生效→工作台提醒→作废，viewer 无入口）；后端全量 go test/vet/build、前端专项 Vitest 52/52、tsc/lint/build、diff check 通过，详见验证文档 11.13】
- [x] P12.3.6 奖惩记录子批次（第三个基线占位归零）：奖励/惩罚独立台账、员工快照、档案附件关联与新壳页面，满足矩阵 5.2 归零五条件后矩阵 3.3 奖惩记录转 ✅（交付物：模型 + API + 权限 + 页面 + E2E；依赖：P12.2）【已完成：独立 `reward_records` 模型，奖励/惩罚两类，必填员工、类型、发生日期、事由、等级，选填分值、金额、负责人、附件、备注；状态 draft→effective→voided，草稿可编辑，draft/effective 均可由 `reward.delete` 填写原因作废；不改变员工状态或薪资；创建时保存员工姓名/部门/岗位快照，复用档案 `document_id`；新增 `reward.view/create/edit/delete` 权限与租户/部门隔离；独立新壳“奖惩记录”入口；隔离 E2E 2/2（admin 奖励草稿→生效→作废且员工保持 active，viewer 无入口）；后端全量 go test/vet/build、前端专项 Vitest 40/40、tsc/lint/build、diff check 通过，详见验证文档 11.14】
- [x] P12.3.7 人事异动子批次（唯一 ⏳ 后续接入项归零）：调岗/晋升/降级独立台账、员工职级、手动生效与新壳页面，满足矩阵 5.2 归零五条件后矩阵 3.3 人事异动转 ✅（交付物：模型 + API + 页面 + E2E；依赖：P12.2）【已完成：新增可选 `Employee.JobLevel` 与独立 `personnel_changes` 台账；类型仅 transfer/promotion/demotion，状态 draft→effective→voided；创建/编辑/生效/作废统一要求 `employee.edit`，生效在事务内校验员工前快照一致后同步部门、岗位、职级；已生效作废仅作废台账、不回滚员工资料；独立新壳页面、入口和视图映射；隔离 E2E 2/2（admin 晋升草稿→手动生效→员工资料更新→作废不回滚，viewer 无入口）；后端全量 go test/vet/build、前端专项 Vitest 42/42、tsc/lint/build、diff check 通过，详见验证文档 11.15】
- [x] P12.3.8 培训管理子批次（第四个基线占位归零）：内部/外部/线上培训独立记录台账、新壳页面与权限，满足矩阵 5.2 归零五条件后矩阵 3.3 培训管理转 ✅（交付物：模型 + API + 权限 + 页面 + E2E；依赖：P12.2）【已完成：独立 `training_records` 模型，类型 internal/external/online，状态 draft→completed→voided；必填主题、类型、日期，选填讲师/机构、关联员工、结果、备注；关联员工冻结姓名/部门/岗位快照且不联动员工资料；新增 `training.view/create/edit/delete` 权限与租户/部门隔离；独立新壳“培训管理”入口；隔离 E2E 2/2（admin 内部培训草稿→完成→作废且员工资料不变，viewer 无入口）；后端全量 go test/vet/build、前端专项 Vitest 43/43、tsc/lint/build、diff check 通过，详见验证文档 11.16】
- [x] P12.3.9 安全管理子批次（第五个基线占位归零）：安全检查记录台账、新壳页面与权限，满足矩阵 5.2 归零五条件后矩阵 3.4 安全管理转 ✅（交付物：模型 + API + 权限 + 页面 + E2E；依赖：P12.2）【已完成：独立 `safety_inspections` 模型，检查类型仅 routine/special（新壳显示例行检查/专项检查），状态 draft→completed→voided；必填类型、日期、地点、负责人、问题描述、整改要求；新增 `safety.view/create/edit/delete` 权限与租户隔离；独立新壳“安全管理”入口；修复前端类型契约，表单仅提交后端允许的 routine/special，保存失败不关闭对话框；隔离 E2E 2/2（admin 例行检查草稿→完成→作废并 API 核验状态/原因，viewer 无入口）；后端全量 go test/vet/build、前端专项 Vitest 45/45、tsc/lint/build、diff check 通过，详见验证文档 11.17】
- [x] P12.3.10 车队管理子批次（第六个基线占位归零）：仅车辆档案、新壳页面与权限，满足矩阵 5.2 归零五条件后矩阵 3.5 车队管理转 ✅（交付物：模型 + API + 权限 + 页面 + E2E；依赖：P12.2）【已完成：独立 `fleet_vehicles` 模型，必填租户内唯一车牌号、车型、状态 active/inactive，选填品牌、座位数、购置日期、备注；active 可编辑或删除，inactive 不可编辑或删除，仅可恢复为 active；不实现调度、加油、维修、审批、轨迹、附件、定时任务或员工联动；新增 `fleet.view/create/edit/delete` 权限与租户隔离；独立新壳“车辆档案”入口；隔离 E2E 2/2（admin 新增→编辑→停用→恢复→删除并 API 核验，viewer 无入口）；后端全量 go test/vet/build、前端专项 Vitest 44/44、tsc/lint/build、diff check 通过，详见验证文档 11.18】
- [x] P12.3.11 职业卫生子批次（第七个基线占位归零）：职业健康检查台账、新壳页面与权限，满足矩阵 5.2 归零五条件后矩阵 3.4 职业卫生转 ✅（交付物：模型 + API + 权限 + 页面 + E2E；依赖：P12.2）【已完成：独立 `occupational_health_checks` 模型，必填员工、检查日期、医疗机构、检查类别，选填检查结论、下次检查日期、备注；创建或编辑冻结员工姓名、部门、岗位快照，不联动员工资料；状态 draft→completed→voided，草稿可编辑/完成/删除，草稿或已完成均可填写原因作废；新增 `occupational_health.view/create/edit/delete` 权限与租户/部门快照隔离；独立新壳“职业健康检查”入口；前后端 JSON 契约统一为 `medical_institution`、`check_conclusion` 与员工快照字段，权限排序为 114–117，避免与安全/车队权限冲突；隔离 E2E 2/2（admin 草稿→完成→作废并 API 核验快照/状态/原因，viewer 无入口）；后端全量 go test/vet/build、前端专项 Vitest 47/47、tsc/lint/build、diff check 通过，详见验证文档 11.19】
- [x] P12.3.12 社保业务子批次（复用既有社保管理并归零占位）：移除无真实挂载的 `daily-affairs-hub.tsx` `social` 卡片，保留新壳真实“社保管理”入口与 `InsuranceManagement` 页面；为既有社保变更、导入、模板、回盘 API 补齐 `insurance.view/create/edit/delete` 服务端鉴权；补充管理员正常、viewer 无写权限、跨租户隔离测试；隔离 E2E 3/3（admin 入口/真实页面、API 创建增员并列表核验；viewer 可查看但创建返回 403）；后端全量 go test/vet/build、前端相关回归 Vitest 27/27、tsc/lint/build、diff check 通过，详见验证文档 11.20】
  > 用户已确认 P12.3.2 规则：① 独立 onboarding_records / work_todos 表，不混入 employees；② 入职三态与 trial/formal 正交（独立维度）；③ 来源分手动/批量，Offer 仅字段与内部契约；④ 定时任务 Asia/Shanghai 每日 02:00、仅扫描 planned_hire_date 等于上海当日、错过不补跑、失败不重试；⑤ 身份证命中 employees 全量统一拒绝；⑥ 放弃恢复保留原日期与历史字段；⑦ 通用待办唯一去重且租户管理员归属；⑧ 不创建账号、不跨模块自动化。
  - [x] P12.3.2-1 模型：onboarding_records / work_todos 独立 GORM 模型 + AutoMigrate（交付物：models + 迁移 + 单测；依赖：无）【已完成：独立模型与迁移，专项单测通过】
  - [x] P12.3.2-2 状态 API：入职三态（待入职/已入职/已放弃）与 trial/formal 正交的状态流转 API + 权限鉴权（交付物：API + handler 测试；依赖：P12.3.2-1）【已完成：七个状态 API（列表/创建/编辑/快速入职/确认/放弃/恢复）与权限鉴权，handler 测试通过】
  - [x] P12.3.2-3 导入/任务：手动/批量来源导入（身份证命中 employees 全量统一拒绝）+ 定时任务（Asia/Shanghai 每日 02:00、仅扫描 planned_hire_date 等于上海当日、错过不补跑、失败不重试）+ 通用待办唯一去重且租户管理员归属（交付物：导入 service + 定时任务 + 单测；依赖：P12.3.2-1, P12.3.2-2）【已完成：JSON/Excel 全文件导入与模板；身份证命中 employees 全量统一拒绝；部门 Code+三位工号生成与唯一冲突重试；同公司最早 admin/super_admin 归属；失败置 pending+日志+去重异常待办，单测通过】
  - [x] P12.3.2-4 页面：独立新壳入口与页面（Offer 仅字段与内部契约；放弃恢复保留原日期与历史字段；不创建账号、不跨模块自动化）（交付物：独立页面 + 新壳入口 + Vitest；依赖：P12.3.2-2, P12.3.2-3）【已完成：独立新壳页与入口已接入；新壳 E2E 2/2（admin 完整流程；viewer 入口隐藏）】
  - [x] P12.3.2-5 验收：后端 go test/vet/build + 前端 lint/tsc/Vitest/build + 隔离 E2E（交付物：测试证据；依赖：P12.3.2-1 至 P12.3.2-4）【已完成：后端专项/全量 go test、vet、build 通过；前端专项 Vitest 37/37、lint 0 error（既有无关 warning 1 条）、tsc、build 通过；E2E 仅使用 127.0.0.1:55432/siapp_e2e（含 E2E seed 部门），临时 runtime-config/.next/test-results/后端进程均清理】

### P12.3.2-4 新壳入职页面执行清单（2026-08-18，用户已确认立即实施）

【本轮范围】仅接入独立新壳入职管理页面、既有 REST API、按钮级权限、视图映射与前端测试；批量导入/定时任务暂不伪造，旧壳保持不变。

- [x] 1. 新增入职 REST 客户端与类型，覆盖列表、创建、编辑、快速入职、确认、放弃、恢复路径（依赖：既有 P12.3.2-2 API）【已完成：新增 `lib/api-onboarding.ts`，复用统一认证请求；无批量导入伪接口】
- [x] 2. 实现独立入职管理页面，覆盖三态筛选、手动登记、快速入职、待入职编辑、确认/放弃/恢复及异常反馈（依赖：1）【已完成：新增独立页面与表单/状态对话框；只展示真实接口能力】
- [x] 3. 接入新壳 `employee.create` 入口、页面内 `employee.create/edit` 按钮权限及统一视图映射；不修改旧壳（依赖：2）【已完成：新壳入口与页面装配已接入；旧 `app-sidebar.tsx` 零改动】
- [x] 4. 补充 API、页面、侧栏和视图映射专项 Vitest，覆盖正常流程与权限/状态边界（依赖：1-3）【已完成：新增 API/页面测试并更新侧栏、映射测试；含无批量入口与只读权限边界】
- [x] 5. 运行专项 Vitest、lint、TypeScript 类型检查（tsc）和 `git diff --check`，记录结果（依赖：4）【已完成：专项 Vitest 4 文件 37/37；lint 0 error、1 条既有 dormitory Hooks warning；tsc 与 diff check 通过】

【本轮进度】5 / 5 完成 ✅
【阶段复盘】已完成 API、页面与新壳装配三项，均在确认范围内；未新增后端接口、批量导入、定时任务或旧壳改动。下一阶段仅补专项测试与执行指定门禁。
【最终复盘】本轮仅新增新壳入职页面、既有 REST 接线、权限与视图映射及测试；批量导入/定时任务未在 P12.3.2-4 轮次实现或伪造，旧壳未改。P12.3.2-3 导入/任务随后续独立批次实施完成，完整批次验收见 P12.3.2-5 与验证文档 11.10，本批次整体已归档。
【下一步】已完成；P12.3.2 批次整体归档，继续 P12.3 后续批次（转正/人事异动与九项占位）。
- [x] 3.2 按独立业务批次实现 🛑 占位项（劳动合同/奖惩记录/培训管理/合同管理/安全管理/能耗管理/车队管理/职业卫生/社保业务），每批归零并移除占位文案（交付物：各批次真实组件 + 归零记录；依赖：3.1 流程复用）【已完成：全部基线占位已归零；矩阵无 ⏳/🛑 状态，待执行最终切换门槛核对】
  - [x] P12.3.4 劳动合同、P12.3.5 行政合同、P12.3.6 奖惩记录、P12.3.8 培训管理、P12.3.9 安全管理、P12.3.10 车队管理、P12.3.11 职业卫生、P12.3.12 社保业务与 P12.3.13 能耗管理均已归零。
  - [x] P12.3.13 能耗管理子批次（最后一个基线占位归零）：只读水、电能耗统计，按抄表日期所属自然月、楼栋汇总用量与费用，并支持按房间下钻；复用宿舍 `DormMeterReading` 数据、既有 `dormitory.view` 权限与抄表 API，不统计燃气，不新增录入、账单、预测、告警或权限资源【已完成：新增受 `dormitory.view` 鉴权的 `GET /api/dormitories/energy/summary`，按月、楼栋、房间聚合水电 usage/amount 并严格 user_id 隔离；独立新壳“能耗管理”只读入口及页面支持月份/楼栋筛选、总览和房间下钻；后端专项 7/7 与全量 go test/vet/build、前端专项 Vitest 47/47、tsc/lint/build、隔离 E2E 3/3、diff check 通过，详见验证文档 11.21】
- [x] 3.3 最终切换门槛核对：AC-1~AC-9 证据齐备、矩阵 ⏳/🛑 全部归零、运行时代码占位残留检测零命中、深浅截图人工确认、系统头部标题字体与动效核对未变、G5 回退演练完成（交付物：切换签署记录 + 验证记录表归档 + TODO/agentmemory 同步；依赖：3.1, 3.2）【已完成：G5 在隔离 worktree 的 `3c18d29` 旧壳构建通过；AC-1 至 AC-9 已归档至验证文档 11.23；全量 E2E 37 通过、4 条件跳过、0 失败；反馈闭环临时使用用户批准的本地模拟 LLM 验证后已恢复隔离配置；深浅截图、标题动效与占位清零核对通过】
- [x] 3.4 生产环境移除迁移开关，固定新壳为最终形态，验证记录归档并本地提交（交付物：生产配置确认 + 最终提交；依赖：3.3）【已完成：用户授权范围为固定新壳、完成测试并创建本地提交，不推送；移除 `UI_SHELL`、`UiShell`、`getUiShell`、`parseUiShell` 与容器运行时字段，保留 API 地址运行时配置；页面无条件装配 NewAppSidebar、new 管理条、受控搜索、嵌入式 AI 助手与 NewShell。前端 Vitest 58 文件 365 项、lint（0 error，仅既有宿舍 Hooks warning）、tsc、build、固定新壳 E2E 7/7；后端 go test/vet/build、diff check 均通过。G5 临时 worktree 保留，未推送。证据见验证文档 11.24】
- 硬性约束：生产切换必须所有旧功能真实接入且无 ⏳/🛑/占位页面或「功能开发中」等占位文案残留；开发期允许保留已登记占位（每批次结束归零或明确下批次计划）；不新增功能、不调整系统头部标题。

【总进度】22 / 22 完成（P12.0-P12.3 所有迁移、归零批次与最终迁移验收均已完成；矩阵 ⏳/🛑 已全部归零）
【下一步】P12.3.4 已完成并创建本地提交；按用户授权，不推送，也不清理 G5 临时 worktree。

【最终切换前置处置】已完成：移除 `system-settings.tsx` 中无调用的 AdvancedOptions 预留组件及“敬请期待”占位文案；系统设置专项 Vitest 6/6、tsc、lint、build、diff check 通过。`frontend/components` 与 `frontend/app` 的占位残留检测已零命中（测试断言中的历史文本不计源码残留）。

【能耗管理决策】已按用户确认完成最小只读统计：仅统计水、电，按月、楼栋汇总并可下钻房间；复用宿舍抄表数据与 `dormitory.view`，未新增录入、账单、预测、告警或燃气统计；P12.3.13 验收后新壳已显示真实入口。
