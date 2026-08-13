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
- [ ] 5. 前端：上传与解析工作台——批量 PDF 拖拽上传、每份任务状态/失败重试、识别字段编辑、低置信度高亮、人工补录最低门槛、草稿权限控制（交付物：invoice 上传/编辑组件 + API 类型 + Vitest；依赖：2,3）
- [ ] 6. 前端：归档管理与打印导出——筛选/统计/关联预警、受控原件预览下载打印、归档打印单、字段 CSV/Excel 导出、确认/作废/更正原因交互、已确认/已作废状态展示（交付物：InvoiceManagement 增强 + 打印/导出组件 + Vitest；依赖：4,5）
- [ ] 7. 验收：模型/API/解析工作器单元与集成测试；前端 lint/tsc/Vitest/build；端到端“上传→解析→补录→确认→查询/打印→作废”主流程；30 天清理任务、判重、附件鉴权与 RBAC 验证（依赖：1-6）

【总进度】3 / 7 完成
【下一步】P7.3.4：归档与业务规则 API

【总进度】15 / 18 完成（P7.4 3 项 + P7.1 7 项 + P7.2 5 项）
【下一步】P7.3：发票管理卡片；P7.2 端到端实测在具备临时全局 LLM 配置的测试环境补跑

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
- [ ] 1.1 后端：ingest 实装——docreader.Parse → IngestToChunks → embedding 向量化 → 写入 DocumentChunk + 关联 KB（交付物：api/knowledge_base.go ingest handler 替换 stub + service/kb_ingest.go；依赖：无）
- [ ] 1.2 后端：入库结果统计返回（scanned/ingested/skipped/errors 明细）（交付物：同上；依赖：1.1）
- [ ] 1.3 后端：单测覆盖 ingest 链路（mock docreader + 内存 SQLite 验证 chunk 写入与 KB 关联）（交付物：service/kb_ingest_test.go；依赖：1.1）

### P9.2 后端检索按 KB 权限过滤
- [ ] 2.1 后端：/api/knowledge/search 与 /api/knowledge/chat 增加 kb_id 参数 + HasAccess 校验（非 admin 仅可检索自己有权限的 KB）（交付物：api/knowledge.go 修改 + service/retrieval.go 扩展；依赖：无）
- [ ] 2.2 后端：脱敏在检索结果返回前应用（复用 kb_mask.ApplyFieldMask）（交付物：service/retrieval.go + service/kb_mask.go 集成；依赖：2.1）
- [ ] 2.3 后端：单测覆盖权限过滤与脱敏（交付物：api/knowledge_test.go 扩展；依赖：2.1,2.2）

### P9.3 前端 chat-panel 范围过滤
- [ ] 3.1 前端：chat-panel 增加知识库范围选择器（下拉列出当前用户可见 KB，默认全部）（交付物：components/chat-panel.tsx 修改；依赖：P9.2）
- [ ] 3.2 前端：检索请求带 kb_id 参数，响应中脱敏字段正常显示（交付物：lib/api.ts chat 相关函数 + chat-panel.tsx；依赖：3.1）
- [ ] 3.3 验收：完整链路实测——入库→脱敏检索→权限隔离问答全通；go build/test + tsc/lint 全绿（依赖：全部）

【总进度】0 / 9 完成
【下一步】P9.1.1：ingest 全链路实装

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

【总进度】0 / 13 完成
【下一步】P11.1.1：修复 storage DSN

【总进度】0 / 19 完成
【下一步】P10.0.1：@designer 输出 ui-design-p9.md 规范

### P6 交付摘要
- 后端：19 个 GORM 模型 + 99 个 HTTP 路由 + 6 个单测（office_analytics 6 例 + canteen_analytics 2 例 + migrate 3 例）
- 前端：2 个主组件 + 10 个 Tab + 7 个辅助文件（utils/api/dialogs）+ 2 个 API 封装
- 设计：`docs/ui-design-p6.md`（信息架构+组件复用清单+视觉规范）
- 迁移：`backend/migrate/import_legacy.go` + `cmd/migrate-legacy/main.go` + `docs/migration-p6.md`
- 验证：`go build ./...` 通过、`go test ./migrate/...` 通过、`tsc --noEmit` 通过、`eslint` 通过
