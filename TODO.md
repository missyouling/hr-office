# 人事行政管理系统 (hr-office) 开发计划

## P0-P2：基础配置与知识库 ✅ 已完成
- 社保分摊系统、标签管理、文件夹树
- 知识库配置、分块编辑、版本管理
- SSE流式问答、会话管理、Chunk搜索
- 3个新测试文件 (46用例)
- BM25 全文检索（tsvector + ts_rank + GIN 索引）— 已在 P2 阶段完成
- RRF 检后重排融合 — 已在 P2 阶段完成

## P3：多轮对话优化 ✅ 已完成
- 上下文压缩：历史消息超 token 限制时自动摘要（smart_summarize 策略）
  - `SummarizeHistory`：LLM 摘要生成
  - `CompressContext`：旧消息摘要化 + 最近消息保留
  - ChatSession 新增 `summary` 字段持久化
- 追问意图识别：检测代词/省略句，LLM 融合上下文改写 query
  - `RewriteQuery`：规则 + LLM 双层检测
  - 集成到 `StreamChat`，在检索前自动改写追问

## P4：前端与交互体验升级 📝
- 智能问答工作台 (Markdown渲染、引用溯源)
- 知识库管理UI优化
- 管理后台 (数据统计仪表盘)

## P5：企业级特性与权限深化 📝
- 部门/角色维度权限控制
- 用户反馈闭环
- 更多文件类型/IM集成
