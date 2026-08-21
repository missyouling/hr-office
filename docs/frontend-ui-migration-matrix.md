# 前端 UI 迁移矩阵（you.com 基线）

> 状态：草稿 v0.1（首版录入）
> 维护责任人：前端负责人（矩阵变更必须同步 code review）
> 关联文档：`/home/hr-office/docs/ui-design-p9.md`、`/home/hr-office/TODO.md`（进度计划，本矩阵不复制其细节，仅引用）

---

## 1. 目标与原则

1. **保留 Next.js 运行时**：`/home/hr-office/frontend`（Next.js 15 + React 19 + TS 5）为唯一业务实现载体，不迁移框架。
2. **you.com 为唯一 UI 基线**：`/home/you.com/src/App.tsx`（React + Vite 原型）定义的导航结构与信息架构，是 hr-office 侧栏与页面组织的唯一权威来源；新增功能必须先入基线，再谈实现。
3. **旧功能逐步接入、零遗漏**：hr-office 全部既有功能域，必须在 you.com 基线中找到归属；基线中每一项导航也必须登记真实接入 / 后续接入 / 临时占位三态之一，禁止未登记项。
4. **占位可清零**：任何"临时占位"都必须登记归零条件与计划批次，禁止无限期"开发中"（见第 5 节归零规则）；分阶段开发期间可保留已登记占位，但**最终切换 / 生产正式入口前必须全部清零**——全部旧功能真实接入、矩阵状态无 ⏳ 与 🛑、无占位页面或"功能开发中"文案（硬性条件，见计划第 8 节与验证计划第 8 节）。

---

## 2. 基线导航结构（you.com 唯一基线）

> 注：本节为 **P12.1 目标基线**（最终切换后的导航形态）；当前实现仍为旧扁平导航（`app-sidebar.tsx` 现行条目），迁移完成前以旧导航为准。

来源：`/home/you.com/src/App.tsx` 的 `ALL_GROUPS` / 顶级视图 / 用户菜单。

| 层级 | 分组 / 入口 | 包含项 |
| --- | --- | --- |
| 顶级 | 工作台（home） | — |
| 顶级 | 知识库（library） | — |
| 顶级 | 偏好设置（settings） | 外观（跟随系统 / 浅色 / 深色） |
| 分组 | 员工管理 | 员工花名册、入职管理、转正管理、离职管理、人事异动、劳动合同、奖惩记录、培训管理 |
| 分组 | 行政管理 | 组织管理、社保管理、公积金管理、档案管理、合同管理、安全管理 |
| 分组 | 日常事务 | 宿舍管理、食堂管理、能耗管理、车队管理、发票管理、办公劳保 |
| 用户菜单 | 个人资料 / 系统设置 / 退出登录 | — |

---

## 3. 迁移矩阵

### 3.1 状态图例

| 标记 | 含义 | 判定标准 |
| --- | --- | --- |
| ✅ 首阶段真实接入 | 旧功能已有真实组件，可直接挂到基线导航 | 有组件 + 有 API + 有测试 |
| ⏳ 后续接入 | 有基础数据或子能力，但未达完整功能形态 | 有部分实现，缺组件/API/测试其一 |
| 🛑 临时占位 | 无组件、无 API，仅占位卡片或文案 | 缺完整实现，登记归零条件 |

### 3.2 顶级视图

| 旧功能域 | 目标新导航入口 | 权限依据 | 状态 | 关键现有入口 / 文件 | 验收证据 |
| --- | --- | --- | --- | --- | --- |
| 工作台（问候、公告轮播、知识库统计、日历、备忘） | 工作台 | 登录即可（viewer 起） | ✅ | `/home/hr-office/frontend/app/page.tsx`（landing 分支）；`components/workbench-overview.tsx`、`workbench-calendar.tsx`、`workbench-memos.tsx`、`knowledge-stats.tsx` | `workbench-overview.test.tsx`、`workbench-calendar.test.tsx`、`workbench-memos.test.tsx` |
| 知识库（文档列表、入库、权限、脱敏） | 知识库 | `knowledge_base.view`（`lib/permissions.ts`） | ✅ | `components/knowledge/KnowledgeBaseManagement.tsx`、`lib/api-knowledge.ts` | `knowledge/` 内 tab 组件；后端 `internal/service/kb_*.go` 测试 |
| 个人偏好（外观主题） | 偏好设置 | 登录即可 | ✅ | `components/personal-settings.tsx`（`personal-settings` 视图） | `personal-settings.test.tsx` |

### 3.3 员工管理组（you.com：员工管理）

| 旧功能域 | 目标新导航入口 | 权限依据 | 状态 | 关键现有入口 / 文件 | 验收证据 |
| --- | --- | --- | --- | --- | --- |
| 员工花名册（在职/离职列表、社保增减、公积金） | 员工管理 → 员工花名册 | `employee.view` | ✅ | `components/employee-management.tsx`（在职员工 / 离职员工 / 社保增加 / 社保减少 / 公积金 tab） | 组件内 tab 分支；e2e `rbac.spec.ts` 覆盖员工资源 |
| 入职管理 | 员工管理 → 入职管理 | `employee.create` | ✅ | 独立新壳页面与入口（新壳侧栏员工管理组）：`components/onboarding-management.tsx`、`components/onboarding/onboarding-form-dialog.tsx`、`lib/api-onboarding.ts`；独立 `onboarding_records` / `work_todos` 模型；七个状态 API（列表/创建/编辑/快速入职/确认/放弃/恢复）；JSON/Excel 全文件导入与模板；定时任务 Asia/Shanghai 每日 02:00 仅扫描 planned_hire_date 等于上海当日（不补跑、不重试）；失败 pending+日志+去重异常待办；身份证全量冲突；部门 Code+三位工号与唯一重试；同公司最早 admin/super_admin 归属 | 新壳 E2E 2/2（admin 完整流程；viewer 入口隐藏，见 `e2e/onboarding.spec.ts`）；后端专项/全量 go test、vet、build 通过；前端专项 Vitest 37/37、lint（仅既有无关 warning）、tsc、build 通过；归零条件见 5.2 |
| 转正管理 | 员工管理 → 转正管理 | `employee.edit` | ✅ | 独立新壳页面与入口：`components/regularization-management.tsx`、`lib/api-regularization.ts`；独立 `regularization_records` / `regularization_effect_runs` 模型；单条三级审批、延期/作废、Excel 全文件导入、Asia/Shanghai 每日 02:00 生效任务与异常待办 | 新壳 E2E 3/3（UI 创建→上级/HR 审批→生效；HR 拒绝待办；viewer 入口隐藏，见 `e2e/regularization.spec.ts`）；后端专项/全量 go test、vet、build 通过；前端专项 Vitest 37/37、lint（仅既有无关 warning）、tsc、build 通过；归零条件见 5.2 |
| 离职管理 | 员工管理 → 离职管理 | `employee.edit` | ✅ | 独立新壳页面与入口（新壳侧栏员工管理组）；后端 resign/restore 强制 `employee.edit` 鉴权；显式 seed-e2e 幂等创建 E2E 员工与 archives/resign_proof 本地存储三件套 | 新壳 E2E 2/2（admin 完整离职→PDF 下载魔数→恢复；viewer 无入口）；旧壳 E2E 1 passed + 1 预期 skip（独立入口仅新壳）；后端 go test/vet/build 全通过；前端专项 Vitest 33/33、lint/tsc/build 通过 |
| 人事异动 | 员工管理 → 人事异动 | `employee.edit` | ✅ | 独立新壳页面与入口：`components/personnel-change-management.tsx`、`lib/api-personnel-changes.ts`；独立 `personnel_changes` 模型；调岗/晋升/降级草稿/生效/作废状态，员工部门/岗位/职级前后快照，生效时事务性更新员工资料 | 新壳 E2E 2/2（admin 晋升草稿→手动生效→员工资料更新→作废不回滚；viewer 无入口，见 `e2e/personnel-change.spec.ts`）；后端专项/全量 go test、vet、build 通过；前端专项 Vitest 42/42、lint（仅既有无关 warning）、tsc、build 通过；归零条件见 5.2 |
| 劳动合同 | 员工管理 → 劳动合同 | `contract.view`（写操作分别为 `contract.create/edit/delete`） | ✅ | 独立新壳页面与入口：`components/labor-contract-management.tsx`、`lib/api-labor-contracts.ts`；独立 `labor_contracts` 模型；固定期限合同草稿/生效/到期/作废状态、部门与员工快照、档案文档关联、Asia/Shanghai 每日 02:00 到期扫描 | 新壳 E2E 2/2（admin 草稿创建→生效→作废；viewer 无入口，见 `e2e/contract.spec.ts`）；后端专项/全量 go test、vet、build 通过；前端专项 Vitest 39/39、lint（仅既有无关 warning）、tsc、build 通过；归零条件见 5.2 |
| 奖惩记录 | 员工管理 → 奖惩记录 | `reward.view`（写操作分别为 `reward.create/edit/delete`） | ✅ | 独立新壳页面与入口：`components/reward-management.tsx`、`lib/api-rewards.ts`；独立 `reward_records` 模型；奖励/惩罚草稿/生效/作废状态、员工快照与档案文档关联 | 新壳 E2E 2/2（admin 奖励草稿创建→生效→作废且员工保持 active；viewer 无入口，见 `e2e/reward.spec.ts`）；后端专项/全量 go test、vet、build 通过；前端专项 Vitest 40/40、lint（仅既有无关 warning）、tsc、build 通过；归零条件见 5.2 |
| 培训管理 | 员工管理 → 培训管理 | `training.view`（写操作分别为 `training.create/edit/delete`） | ✅ | 独立新壳页面与入口：`components/training-management.tsx`、`lib/api-training-records.ts`；独立 `training_records` 模型；内部/外部/线上培训草稿/完成/作废状态，关联员工快照 | 新壳 E2E 2/2（admin 内部培训草稿→完成→作废且员工资料不变；viewer 无入口，见 `e2e/training.spec.ts`）；后端专项/全量 go test、vet、build 通过；前端专项 Vitest 43/43、lint（仅既有无关 warning）、tsc、build 通过；归零条件见 5.2 |

### 3.4 行政管理组（you.com：行政管理）

| 旧功能域 | 目标新导航入口 | 权限依据 | 状态 | 关键现有入口 / 文件 | 验收证据 |
| --- | --- | --- | --- | --- | --- |
| 社保管理（社保业务：增减员、缴纳记录、回盘） | 行政管理 → 社保管理 | `insurance.view`（写操作分别为 `insurance.create/edit/delete`） | ✅ | 复用真实 `components/insurance-management.tsx`（insurance 视图）及既有社保变更/回盘 API；已移除无挂载的日常事务 `social` 占位卡片 | 新壳 E2E 3/3（admin 入口与真实页面、API 创建增员并列表核验；viewer 可查看但创建接口 403，见 `e2e/insurance-management.spec.ts`）；后端 `insurance_rbac_test.go` 覆盖权限/租户隔离；前端相关回归 Vitest 27/27、tsc、lint、build 通过 |
| 公积金管理（缴存/提取） | 行政管理 → 公积金管理 | `insurance.view` | ✅ | `components/employee-management.tsx` provident tab | 公积金 tab 已实现并有测试路径 |
| 档案管理（四类档案全生命周期） | 行政管理 → 档案管理 | `archives.view` | ✅ | `components/archives-management.tsx`（经 `daily-affairs-hub.tsx` archives 分支挂载） | 后端 `internal/api/archives.go`（分类/上传/批量）；e2e `invoice-p73.spec.ts` 链路 |
| 合同管理（行政合同签订归档） | 行政管理 → 行政合同 | `admin_contract.view`（写操作分别为 `admin_contract.create/edit/delete`） | ✅ | 独立新壳页面与入口：`components/admin-contract-management.tsx`、`lib/api-admin-contracts.ts`；独立 `admin_contracts` 模型；外部主体合同草稿/生效/到期/作废状态、档案文档关联、Asia/Shanghai 每日 02:00 到期扫描与工作台到期提醒 | 新壳 E2E 2/2（admin 草稿创建→生效→工作台提醒→作废；viewer 无入口，见 `e2e/admin-contract.spec.ts`）；后端专项/全量 go test、vet、build 通过；前端专项 Vitest 52/52、lint（仅既有无关 warning）、tsc、build 通过；归零条件见 5.2 |
| 安全管理（安全检查） | 行政管理 → 安全管理 | `safety.view`（写操作分别为 `safety.create/edit/delete`） | ✅ | 独立新壳页面与入口：`components/safety-management.tsx`、`lib/api-safety-inspections.ts`；独立 `safety_inspections` 模型；例行/专项检查草稿/完成/作废状态 | 新壳 E2E 2/2（admin 例行检查草稿→完成→作废；viewer 无入口，见 `e2e/safety.spec.ts`）；后端专项/全量 go test、vet、build 通过；前端专项 Vitest 45/45、lint（仅既有无关 warning）、tsc、build 通过；归零条件见 5.2 |
| 职业卫生（职业健康检查） | 行政管理 → 职业健康检查 | `occupational_health.view`（写操作分别为 `occupational_health.create/edit/delete`） | ✅ | 独立新壳页面与入口：`components/occupational-health-check-management.tsx`、`lib/api-occupational-health-checks.ts`；独立 `occupational_health_checks` 模型；员工快照草稿/完成/作废状态 | 新壳 E2E 2/2（admin 草稿创建→完成→作废并 API 核验快照；viewer 无入口，见 `e2e/occupational-health.spec.ts`）；后端专项/全量 go test、vet、build 通过；前端专项 Vitest 47/47、lint（仅既有无关 warning）、tsc、build 通过；归零条件见 5.2 |

### 3.5 日常事务组（you.com：日常事务）

| 旧功能域 | 目标新导航入口 | 权限依据 | 状态 | 关键现有入口 / 文件 | 验收证据 |
| --- | --- | --- | --- | --- | --- |
| 宿舍管理（房屋/入住/抄表/账单/备忘） | 日常事务 → 宿舍管理 | `dormitory.view` | ✅ | `components/dormitory-management.tsx`（dormitory 视图）；`lib/dorm-notifications.ts` | `dorm-notifications.test.ts`；e2e 覆盖宿舍链路 |
| 食堂管理（字典/采购/收入/菜单/分析） | 日常事务 → 食堂管理 | `canteen.view` | ✅ | `components/canteen/CanteenManagement.tsx`（经 `daily-affairs-hub.tsx` canteen 分支） | `canteen/tabs/` 各 tab 组件；后端 `api-canteen.ts` |
| 能耗管理（水电监控统计） | 日常事务 → 能耗管理 | `dormitory.view` | ✅ | 独立新壳只读页面：`components/energy-management.tsx`；`GET /api/dormitories/energy/summary` 复用 `DormMeterReading` | 新壳 E2E 3/3（admin 入口、真实水电汇总与房间下钻；viewer 可见并可只读查询，见 `e2e/energy-management.spec.ts`）；后端专项 7/7 与全量 go test/vet/build、前端专项 Vitest 47/47、tsc/lint/build 通过。仅水、电；无燃气、录入、账单、预测或告警 |
| 车队管理（车辆档案） | 日常事务 → 车辆档案 | `fleet.view`（写操作分别为 `fleet.create/edit/delete`） | ✅ | 独立新壳页面与入口：`components/fleet-vehicle-management.tsx`、`lib/api-fleet-vehicles.ts`；独立 `fleet_vehicles` 模型；启用/停用车辆档案 | 新壳 E2E 2/2（admin 新增→编辑→停用→恢复→删除并 API 核验；viewer 无入口，见 `e2e/fleet-vehicles.spec.ts`）；后端专项/全量 go test、vet、build 通过；前端专项 Vitest 44/44、lint（仅既有无关 warning）、tsc、build 通过；归零条件见 5.2 |
| 发票管理（录入/报销/审批/归档/统计） | 日常事务 → 发票管理 | `invoice.view`（submit/approve/reject 分级） | ✅ | `components/invoice/InvoiceManagement.tsx`（经 `daily-affairs-hub.tsx` invoice 分支）；`lib/api-invoice.ts` | `invoice/InvoiceManagement.test.tsx`、`lib/api-invoice.test.ts`；e2e `invoice-p73.spec.ts` |
| 办公劳保（字典/采购/请款/分析） | 日常事务 → 办公劳保 | `office-supply.view` | ✅ | `components/office-supply/OfficeSuppliesManagement.tsx`（经 `daily-affairs-hub.tsx` office-supplies 分支） | `office-supply/tabs/` 组件；后端 `api-office.ts` |

### 3.6 用户菜单 / 系统级

| 旧功能域 | 目标新导航入口 | 权限依据 | 状态 | 关键现有入口 / 文件 | 验收证据 |
| --- | --- | --- | --- | --- | --- |
| 个人资料（个人设置） | 用户菜单 → 个人资料 | 登录即可 | ✅ | `components/personal-settings.tsx` | `personal-settings.test.tsx` |
| 系统设置（站点配置、存储、模型、备份、告警） | 用户菜单 → 系统设置 | `settings.view`（`app-sidebar.tsx` 按 `hasPermission("settings","view")` 过滤） | ✅ | `components/system-settings.tsx`；`lib/runtime-config.ts`；`alert-settings-dialog.tsx`、`backup-management-dialog.tsx`、`model-settings.tsx` | `system-settings.test.tsx`；后端 `internal/models/system_settings.go` |
| 审计日志 | 系统设置内（系统设置 → 审计日志） | `users.view` | ✅ | `components/system-settings.tsx` 内部入口 → `components/audit-logs.tsx` | `system-settings.test.tsx`；不进入主侧栏 |
| 系统监控 | 系统设置内（系统设置 → 系统监控） | `users.view` | ✅ | `components/system-settings.tsx` 内部入口 → `components/system-monitoring.tsx` | `system-settings.test.tsx`；不进入主侧栏 |
| 组织管理 | 行政管理 → 组织管理 | `department.view`（`admin` / `super_admin` 兜底） | ✅ | `components/layout/new-app-sidebar.tsx` → `organization` → `components/organization-management.tsx` | `new-app-sidebar.test.tsx`；`rbac.spec.ts` |

### 3.7 hr-office 独有能力（基线外，纳入零遗漏清单）

| 旧功能域 | 目标新导航入口 | 权限依据 | 状态 | 关键现有入口 / 文件 | 验收证据 |
| --- | --- | --- | --- | --- | --- |
| 部门管理 | 侧栏独立项（现有 `DEPARTMENTS_ITEM`） | `department.view`（`app-sidebar.tsx` 带角色兜底） | ✅ | `components/admin/department-management.tsx`；`lib/api-department.ts` | 组件已挂 `departments` 视图 |
| 反馈管理（用户反馈闭环） | 侧栏独立项（现有 `FEEDBACK_ITEM`） | `users.view` | ✅ | `components/feedback-panel.tsx`；`components/feedback/`（`feedback-answer.tsx`、`feedback-status-badge.tsx`、`my-feedback-dialog.tsx`）；`lib/feedback.ts` | `lib/feedback.test.ts`；e2e `feedback-closure.spec.ts` |
| AI 助手 | 全局浮动 Dock（现有 `dock:open-chat` 事件） | 登录即可 | ✅ | `components/chat-panel.tsx`；`components/layout/management-bar.tsx` | `chat-panel.test.tsx` |
| 通知中心 | 全局浮动 Dock（现有 `dock:open-notification` 事件） | 登录即可 | ✅ | `components/notification-center.tsx` | `notification-center.test.tsx` |
| 全局搜索 | 全局（`GlobalSearch` 覆盖全部视图） | 登录即可 | ✅ | `components/global-search.tsx`；后端 `internal/service/retrieval.go` | `global-search.test.tsx`；后端 `internal/api/global_search_test.go` |
| 日常事务中心 | 工作台（入口卡片，非独立导航） | 登录即可 | ✅ | `components/daily-affairs-hub.tsx`（含 8 张卡片导航） | 各真实模块见 3.4 / 3.5；占位卡片见 5.2 |

---

## 4. 零遗漏核对

### 4.1 核对方法

1. **基线 → 实现**：遍历 `/home/you.com/src/App.tsx` 全部导航项（第 3 节 3.2–3.6），每项必须存在且状态非空。
2. **实现 → 基线**：遍历 `/home/hr-office/frontend/components/` 与 `app/page.tsx` 的 `renderMainContent` 分支，每个视图必须能在基线中找到归属（含 3.7 独有能力清单）。
3. 检查结果登记在每批次 `/task-review` 时复核本文件。

### 4.2 当前缺失登记（必须在批次计划中消项）

| 缺失类型 | 内容 | 处置要求 |
| --- | --- | --- |
| 基线缺入口 | 已处理：审计日志、系统监控归入系统设置内；组织管理归入行政管理组 | P12.2.2 完成后由 hr-office 新壳验证可达性与权限隐藏 |
| 基线占位项 | 无（能耗管理已于 P12.3.13 归零；社保业务已于 P12.3.12 合并至既有社保管理真实入口并归零；其余历史占位均已在 P12.3.4–P12.3.11 归零） | 新壳仅显示真实 ✅ 功能；无占位入口、不可点击入口或 PlaceholderView |
| 后续接入项 | 无（离职管理见 3.3 ✅ 已于 P12.3.1-1 归零，入职管理见 3.3 ✅ 已于 P12.3.2 归零，转正管理见 3.3 ✅ 已于 P12.3.3 归零，人事异动见 3.3 ✅ 已于 P12.3.7 归零） | 无待接入项；P12.3 全部占位已归零，待执行最终切换门槛核对 |

> **权限覆盖核对注记**：食堂管理、办公劳保、知识库、部门管理四域当前矩阵状态为 ✅，但其**前端静态权限（`lib/permissions.ts`）与后端权限种子（`rbac_seed.go`）的覆盖差异**须在 P12.3 归零前单独逐一核对，确认无缺口后更新本矩阵；在此之前不得声称该四域权限已完整覆盖。

---

## 5. 占位归零规则

### 5.1 占位定义

满足以下任一即视为占位：

- 页面渲染"功能开发中，敬请期待"或等价文案；
- 导航 / 卡片可点击但无真实组件挂载（如 `daily-affairs-hub.tsx` 中 `fleet`、`training`、`occupational`、`social` 卡片）；
- 有组件但基线导航无入口、用户不可达（如审计日志 / 系统监控 / 组织管理）。

**开发期隐藏策略（P12.2.3 已确认）**：⏳ 后续接入项与 🛑 临时占位项在新壳导航中一律不可见——不显示入口、不可点击、不渲染「开发中」或 PlaceholderView；可见性以状态列为唯一依据（仅 ✅ 项进入新壳导航）。隐藏 ≠ 归零，归零仍须按 5.2 / 5.3 执行，最终切换前全部归零。

### 5.2 归零条件（每项占位必须同时满足）

| # | 条件 | 验收手段 |
| --- | --- | --- |
| 1 | 权限依据就绪：后端 `permissions` 种子新增对应 `module` 权限代码；前端 `lib/permissions.ts` 的 `ResourceType` 扩展并写入权限矩阵 | `backend/rbac_seed.go` 与 `cmd/migrate-roles/main.go` 同步；`go test ./backend/cmd/...` 通过 |
| 2 | 后端 API 完整（列表/详情/写操作/审批等业务动作） | 对应 handler 测试通过，遵循 `backend/internal/middleware/permission.go` 鉴权 |
| 3 | 前端真实组件挂载到基线导航对应入口（禁用占位文案） | `npm run lint`、`npm run build` 通过 |
| 4 | 验收证据齐备：e2e（`frontend/e2e/`，Playwright）或组件测试覆盖正常 + 边界 | 测试通过后更新本矩阵状态列为 ✅ |
| 5 | 入口可达性复核：真实用户可经基线导航到达，无权限用户按矩阵隐藏 | `app-sidebar.tsx` 权限过滤断言 |

> **开发期注记**：最终切换前，⏳ / 🛑 项在新壳一律隐藏（见 5.1 开发期隐藏策略）；隐藏不影响归零条件的执行，归零完成后状态转 ✅ 并接入新壳导航。

### 5.3 归零流程

0. **开发期新壳隐藏**：⏳ / 🛑 项不进入新壳导航（不显示入口、不可点击、不展示「开发中」/PlaceholderView）；归零批次仅登记于矩阵 4.2 与 TODO，最终切换前按第 4 条全部归零。
1. 在批次计划（`/home/hr-office/TODO.md`）中登记占位项与计划批次。
2. 完成 5.2 全部条件后，将本矩阵状态由 🛑 改为 ✅，并填写验收证据列。
3. 执行批次 `task-review` 时核对占位计数：**每批次结束时"临时占位"必须归零，或已在 TODO 明确下批次计划**（该节奏仅适用于分阶段开发期间，允许保留已登记占位）；连续两批次未动的占位项视为优先级缺陷，须上报主协调者。
4. **最终切换硬性门槛**：最终切换 / 生产正式入口时，⏳ 与 🛑 状态必须全部转为 ✅（全部旧功能真实接入），无任何占位页面或「功能开发中」等占位文案残留；不得以"已登记批次 / 已明确下批次计划"为由保留占位（切换门槛见计划第 8 节与验证计划第 8 节）。
5. 占位残留检测（可选自动化）：CI 对 `frontend/` 执行 `rg "功能开发中|敬请期待" components app`，命中即失败；最终切换前必须启用且零命中。

---

## 6. 维护机制

1. **单一事实来源**：本文件是 UI 迁移状态的唯一权威登记，任何状态变更必须同步本文件。
2. **基线变更流程**：新增功能先改 `/home/you.com/src/App.tsx`（基线定义导航），再在本文件第 3 节新增行（登记三态），最后在 hr-office 实现组件——顺序不可颠倒。
3. **批次联动**：每个 P 阶段初始化（`/task-init`）前，主协调者复核本文件占位清单；完成时 `/task-review` 对照 5.3 计数。
4. **权限联动**：新功能权限一律先入 `lib/permissions.ts` 与后端种子，侧栏入口用 `hasPermission` / `PermissionGate` 过滤（`components/permission-gate.tsx`、`components/layout/app-sidebar.tsx`）。
5. **文档引用**：本文件与 `/home/hr-office/docs/ui-design-p9.md`（视觉规范）、`/home/hr-office/TODO.md`（进度计划）互引；实施细节以各自文档为准，不在此重复。
6. **新壳可见性约束**：新壳导航只渲染状态为 ✅ 的项；⏳ / 🛑 项开发期一律隐藏（不显示入口、不可点击、不展示「开发中」/PlaceholderView），仅登记于矩阵与 TODO；任何新增导航项必须先在基线（you.com）与本矩阵登记三态，禁止绕过矩阵直接加显。

---

## 7. 自核记录

- [x] Markdown 表格列宽与分隔符核对（各表头 `---` 对齐）。
- [x] 无 API 密钥、无个人数据（基线 `App.tsx` 中的示例用户"寇江"未写入本文件）。
- [x] 未复制 TODO 细节，仅引用 `/home/hr-office/TODO.md` 路径。
- [x] 每项功能域均有：目标入口、权限依据、三态状态、关键文件、验收证据五要素。
- [x] 占位归零规则（第 5 节）与维护机制（第 6 节）齐备。
- [x] 全文简体中文，专业术语首次出现附解释（见 5.1 占位定义、6.1 单一事实来源）。
