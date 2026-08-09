# 流程优先级（最高准则）
当多个规则或技能对同一流程有不同定义时，始终以 `dev-workflow-strict` 技能的定义为准。其他技能仅作为该流程中某个步骤的具体执行方法。

# 项目规则

## 代码风格 [ECC: coding-style]

### 不可变性（CRITICAL）
- 始终创建新对象，**绝不**修改已有对象。不可变数据防止隐藏的副作用，便于调试和支持安全并发。
- 示例：`update(original, field, value)` 返回新副本，而非 `modify(original, field, value)` 原地修改。

### KISS / DRY / YAGNI
- **KISS**：优先选择能正常工作的最简单方案，避免过早优化，清晰性优于巧妙性。
- **DRY**：将重复逻辑提取为共享函数或工具，避免复制粘贴导致的实现漂移。
- **YAGNI**：不在需要之前构建功能或抽象，避免推测性的通用设计。

### 文件组织
- 多小文件优于少大文件：高内聚、低耦合。
- 典型 200-400 行，最大 800 行。
- 按功能/领域组织，而非按类型。

### 输入验证
- 在系统边界（用户输入、API 响应、文件内容）始终进行验证。
- 使用基于 Schema 的验证（如可行）。
- 以清晰的错误信息快速失败，永不信任外部数据。

### 命名规范
- 变量/函数：`camelCase`，描述性命名。
- 布尔值：优先使用 `is`、`has`、`should`、`can` 前缀。
- 接口/类型/组件：`PascalCase`。
- 常量：`UPPER_SNAKE_CASE`。
- 自定义 Hook：`camelCase` + `use` 前缀。

### 代码坏味道
- **深层嵌套**：超过 4 层 → 使用提前返回（early return）替代嵌套条件。
- **魔法数字**：使用命名常量替代有意义的阈值、延迟和限制值。
- **长函数**：将大函数拆分为职责明确的小函数（<50 行）。

## 安全检查 [ECC: security]

### 每次提交前必检清单
- [ ] 无硬编码密钥（API 密钥、密码、令牌）
- [ ] 所有用户输入已验证
- [ ] SQL 注入防护（参数化查询）
- [ ] XSS 防护（HTML 已清理）
- [ ] CSRF 防护已启用
- [ ] 认证/授权已验证
- [ ] 所有端点已配置速率限制
- [ ] 错误消息不泄露敏感数据

### 密钥管理
- **绝不**在源码中硬编码密钥。
- **始终**使用环境变量或密钥管理器。
- 在启动时验证必要的密钥是否存在。
- 轮换任何可能已暴露的密钥。

### 安全响应协议
发现安全问题时：
1. 立即停止当前工作。
2. 修复关键问题后才能继续。
3. 轮换已暴露的密钥。
4. 审查整个代码库中的类似问题。

## 代码审查 [ECC: code-review]

### 审查严重级别
| 级别 | 含义 | 操作 |
|------|------|------|
| CRITICAL | 安全漏洞或数据丢失风险 | **阻塞** — 合并前必须修复 |
| HIGH | Bug 或重大质量问题 | **警告** — 合并前应修复 |
| MEDIUM | 可维护性担忧 | **提示** — 建议修复 |
| LOW | 样式或次要建议 | **备注** — 可选 |

### 提交前审查要求
- 所有自动化检查（CI/CD）已通过。
- 合并冲突已解决。
- 分支已与目标分支同步。
- 无 `console.log` 或调试语句残留。

## 测试要求 [ECC: testing]

### AAA 测试模式
- **Arrange**（准备）：设置测试数据和前置条件。
- **Act**（执行）：调用被测试的代码。
- **Assert**（断言）：验证结果是否符合预期。

### 测试命名规范
使用描述性名称说明被测行为：
- `test('returns empty array when no items match query', () => {})`
- `test('throws error when required API key is missing', () => {})`
- `test('falls back to default when service is unavailable', () => {})`

## Git 提交规范 [ECC: git-workflow]

### 提交消息格式
```
<type>: <description>

<optional body>
```

类型：`feat`、`fix`、`refactor`、`docs`、`test`、`chore`、`perf`、`ci`

## 开发工作流 [ECC: development-workflow]

### 研究优先（编码前必须）
1. 先搜索现有实现和开源方案，确认没有可复用的轮子。
2. 查阅官方文档确认 API 行为和版本特性。
3. 检查包注册表，优先使用经过验证的库而非手写工具代码。
4. 优先采用或移植已验证的方案，而非从头编写新代码。

## 通用架构模式 [ECC: patterns]

### 仓库模式（Repository Pattern）
将数据访问封装在一致的接口后：
- 定义标准操作：`findAll`、`findById`、`create`、`update`、`delete`。
- 具体实现处理存储细节（数据库、API、文件等）。
- 业务逻辑依赖抽象接口，而非存储机制。

### API 响应格式
所有 API 响应使用统一信封：
- 包含成功/状态指示器。
- 包含数据负载（错误时为 null）。
- 包含错误消息字段（成功时为 null）。
- 分页响应包含元数据（total、page、limit）。

---

## 规则来源说明
- 标注 `[ECC: xxx]` 的规则来自 ECC 通用规则库的精简整合。
- 已跳过的 ECC 规则：
  - `agents.md` — ECC 代理体系为 Claude Code 专属，OpenCode 使用自有代理系统（@explorer / @oracle / @fixer 等），编排逻辑已内建于编排器提示词中。
  - `hooks.md` — Claude Code 专属的 Hook 系统（PreToolUse / PostToolUse / Stop），OpenCode 无对应机制。
  - `performance.md` — 模型选择（Haiku/Sonnet/Opus）为 Claude 专属，上下文窗口管理已内建于编排器，构建排障由 `debug-assistant` 技能覆盖。

## 任务状态持久化规则 (最高优先级，强制执行)
- 每次会话开始时，你必须首先读取 `TODO.md` 获取当前进度。
- 制定或更新开发计划后，**必须立即**写入 `TODO.md`。
- 每完成一个阶段的任务，**必须立即**更新 `TODO.md` 中的状态。
- 如果 `TODO.md` 不存在或进度不明确，必须停止并询问用户，**绝对禁止猜测任务**。
