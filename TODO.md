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

### P7.4 修复 pre-existing storage 测试
- [ ] 调研 TestResolve_NoDefaultConfig 失败原因（依赖：1）
- [ ] 修复代码或测试用例（依赖：1）
- [ ] 验收：`go test ./internal/service/storage/...` 通过

### P7.1 RBAC 权限深化
- [ ] 后端：扩展 4 角色枚举 admin/manager/editor/viewer（依赖：P7.4）
- [ ] 后端：部门级数据隔离中间件（用户只能看本部门数据）（依赖：上一项）
- [ ] 后端：API 权限校验中间件扩展（基于角色 + 资源 + 操作）（依赖：上一项）
- [ ] 后端：种子数据补充默认角色权限映射（依赖：上一项）
- [ ] 前端：按钮级权限组件 + 菜单/操作按角色隐藏（依赖：后端 API）
- [ ] 验收：4 角色登录后菜单/按钮按预期显隐；部门隔离生效；CI 全绿

### P7.2 用户反馈闭环
- [ ] 后端：Feedback 模型（用户/AI回答/评分/文字反馈/管理员回复/状态）
- [ ] 后端：API（提交反馈、列表、回复、统计）
- [ ] 前端：chat-panel 集成 👍/👎 评分按钮 + 文字反馈弹窗
- [ ] 前端：管理员反馈面板（feedback-panel 已有雏形，完善列表/回复/统计）
- [ ] 验收：用户可评分/反馈；管理员可查看/回复/统计

### P7.3 发票管理卡片
- [ ] 后端：发票模型 + 基础 CRUD API（与 P6 采购数据可选联动）
- [ ] 前端：发票管理主组件 + 卡片接入 daily-affairs-hub
- [ ] 验收：日常事务菜单可进入发票页，CRUD 正常

【总进度】0 / 13 完成
【下一步】P7.4：调研 TestResolve_NoDefaultConfig 失败原因

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
- [ ] 更新 TODO.md + agentmemory + git commit + push

【总进度】0 / 25 完成
【下一步】P8.1.1：新增 KnowledgeBase/KBAccessRule/KBFieldMask 模型

### P6 交付摘要
- 后端：19 个 GORM 模型 + 99 个 HTTP 路由 + 6 个单测（office_analytics 6 例 + canteen_analytics 2 例 + migrate 3 例）
- 前端：2 个主组件 + 10 个 Tab + 7 个辅助文件（utils/api/dialogs）+ 2 个 API 封装
- 设计：`docs/ui-design-p6.md`（信息架构+组件复用清单+视觉规范）
- 迁移：`backend/migrate/import_legacy.go` + `cmd/migrate-legacy/main.go` + `docs/migration-p6.md`
- 验证：`go build ./...` 通过、`go test ./migrate/...` 通过、`tsc --noEmit` 通过、`eslint` 通过
