# 人事行政管理系统前端 (hr-office)

基于 Next.js + Tailwind CSS + shadcn/ui 的单页应用，负责对接 Go 后端，完成账期管理、险种明细上传、花名册导入及汇总结果展示/导出。

## 环境要求

- Node.js 18+
- npm（项目使用 npm 作为包管理器）

## 快速开始

```bash
cd frontend
npm install                    # 如已执行 create-next-app 可跳过
cp .env.local.example .env.local
npm run dev
```

默认在 `http://localhost:3000` 启动开发服务器。若后端运行在其他地址，请修改 `.env.local` 中的 `NEXT_PUBLIC_API_BASE_URL`。

## 主要功能

- **账期管理**：创建、切换社保账期。
- **险种上传**：单文件及批量上传（选择多文件后逐一确认险种、扣款部分），实时显示缺失险种。
- **花名册导入**：上传花名册文件，为明细补全部门等信息，同时提供导入预览。
- **数据处理与展示**：触发汇总，展示社保总表与个人/单位扣款明细。
- **导出能力**：个人、单位扣款明细支持一键导出 Excel。
- **UI 组件**：基于 shadcn/ui 组件库，配合 Sonner 进行状态提示。

## 目录结构

```
frontend/
  app/
    layout.tsx        # 全局布局，注入 UI Provider
    page.tsx          # 主界面与交互逻辑
    providers.tsx     # 全局 Toast 等 Provider
  components/ui/      # shadcn 生成的 UI 组件
  lib/
    api.ts            # 封装后端 API 调用
    types.ts          # 接口数据类型定义
  public/             # 静态资源
  components.json     # shadcn 配置
  ...
```

## 可用脚本

| 命令 | 说明 |
| --- | --- |
| `npm run dev` | 启动开发服务器 |
| `npm run lint` | 运行 ESLint 检查 |
| `npm run build` | 生成生产构建 |
| `npm run start` | 启动生产构建（需先执行 `build`） |

## 现阶段功能亮点

- 多文件批量上传，自动识别并允许手动修正险种/扣款部分。
- 花名册导入后即时在列表中预览，汇总时自动补全部门信息。
- 扣款明细支持直接导出 Excel，方便与后端结果核对。

## 后续建议

- 在前端加入高级筛选、搜索等功能，提升数据查阅体验。
- 引入 React Query/SWR 等数据缓存方案，减少重复请求。
- 增加 Excel 导出时的样式模板选择。
