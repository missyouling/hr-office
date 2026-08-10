# Triage Labels

`triage` 技能已安装，使用 **默认五大角色标签**：

| 标签 | 含义 | 触发动作 |
|------|------|----------|
| `needs-triage` | 新建 Issue 默认标签，等待分诊 | 分诊员认领并归类 |
| `needs-info` | 缺少必要信息，无法继续 | 等提交者补充后移除 |
| `ready-for-agent` | 已具备行动指南，等待 Agent 实施 | Agent 接管并跟踪 |
| `ready-for-human` | 需人工审查或决策 | 项目维护者介入 |
| `wontfix` | 经评审不再处理 | 关闭并归档 |

## 使用约束

- Issue 创建后必须立即打 `needs-triage` 标签。
- 分诊完成后由分诊员更换为 `ready-for-agent` / `ready-for-human` / `wontfix` 之一。
- 信息补充后从 `needs-info` 切回 `needs-triage`。
- 标签字符串与名称完全一致，禁止本地化或重命名；如需覆盖，单独建映射清单。

## 与提交规范的协同

标签流转不与 `feat/fix/refactor` 提交类型冲突，可叠加使用。
