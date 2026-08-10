# Issue Tracker

本仓库的问题追踪器使用 **GitHub Issues**。

- 仓库：`github.com/missyouling/hr-office`
- CLI 工具：`gh`（GitHub 官方 CLI）
- 工作流：所有任务、缺陷、特性通过 `gh issue create` 创建，三角化的工作在 `gh issue list` 中按状态筛选。

## 使用约束

- 创建 Issue 时必须使用 `triage` 技能内置的 5 个角色标签（详见 `triage-labels.md`）。
- 标题使用中文，遵循 `类型: 简明描述` 格式，与提交规范保持一致。
- PR 与 Issue 通过 `gh pr create --issue <id>` 关联时，编号必须与 Issue 编号一致。

## PRs 作为请求面（默认关闭）

`use_prs_as_request_surface: false`

本项目默认不把外部 PR 拉入 triage 队列。如果未来需要，可在本文件中将此项改为 `true`，并由 `triage` 技能自动识别。
