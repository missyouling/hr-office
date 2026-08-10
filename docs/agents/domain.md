# Domain Docs

本仓库采用 **单上下文（single-context）** 布局。

- 根目录 `CONTEXT.md`：领域模型、核心概念、术语表、所有技能共享的唯一根文档。
- `docs/adr/`：架构决策记录（Architecture Decision Records），按 `NNNN-标题.md` 命名。

## 读取规则

任何 Agent / 技能在开始工作前，必须：

1. 读取根 `CONTEXT.md` 获取项目领域模型。
2. 如涉及架构决策，查阅 `docs/adr/` 目录内已有 ADR，避免重复决策。
3. 若发现 `CONTEXT.md` 与代码实际状态不一致，提交 Issue 并打 `needs-info` 标签。

## 写作规则

- `CONTEXT.md` 由 setup-matt-pocock-skills 与 domain-modeling 共同维护。
- ADR 一旦发布不可修改；如需变更，发布新 ADR 引用旧 ADR。
- 中文优先，技术术语首次出现附通俗解释。

## 复用全局规则

工程纪律、命名规范、测试要求、提交规范统一遵循：

- `/root/.config/opencode/AGENTS.md`（系统级全局规则）
- `.opencode/rules.md`（本项目规则精简整合）
- `/home/hr-office/AGENTS.md`（仓库级入口，含 ## Agent skills 块）
