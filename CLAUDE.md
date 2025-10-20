# CLAUDE.md

此文件为 Claude Code (claude.ai/code) 在本代码仓库中工作时提供指导。

## 项目概述

这是一个社保整合系统，采用 Go 后端和 Next.js 前端架构。系统负责处理社保账期管理、多险种文件上传、花名册管理、数据处理和 Excel 报表生成。

## 系统架构

- **后端**: 使用 Chi 路由器的 Go 应用，GORM 配合 SQLite，excelize 处理 Excel
- **前端**: Next.js 15 配合 React 19、TypeScript、Tailwind CSS、shadcn/ui 组件
- **数据库**: SQLite 配合 GORM 自动迁移
- **API**: 支持 CORS 的 RESTful API，用于前后端通信

### 核心数据模型 (backend/internal/models/models.go)

- `Period`: 社保账期（如 "2025-08"）
- `SourceFile`: 上传的险种文件
- `RawRecord`: 从上传文件中解析的个人保险记录
- `PeriodSummary`: 按险种/缴费部分聚合的统计数据
- `PersonalCharge`/`UnitCharge`: 个人/单位最终计算的扣款明细
- `RosterEntry`: 员工花名册数据，用于补充部门信息

### 险种和缴费部分

- **险种**: `pension`（养老）、`medical`（基本医疗）、`serious_illness`（大额/生育）、`unemployment`（失业）、`injury`（工伤）
- **缴费部分**: `personal`（个人缴费）、`unit`（单位缴费）

## 开发命令

### 前端 (frontend/)
```bash
npm install                    # 安装依赖
npm run dev                    # 启动开发服务器 (http://localhost:3000)
npm run build                  # 生产环境构建
npm run lint                   # 运行 ESLint 检查
npm run start                  # 启动生产构建
```

### 后端 (backend/)
```bash
go run .                       # 启动开发服务器 (http://localhost:8080)
```

### 环境配置

- 前端：复制 `.env.local.example` 为 `.env.local` 并配置 `NEXT_PUBLIC_API_BASE_URL`
- 后端：设置环境变量：
  - `SIAPP_ADDR`: HTTP 监听地址（默认 `:8080`）
  - `SIAPP_DATABASE_PATH`: SQLite 文件路径（默认 `./data/siapp.db`）

## 核心工作流程

1. 通过 `POST /api/periods` 创建社保账期
2. 通过 `POST /api/periods/{id}/files` 上传险种文件（单个或批量）
3. 可选通过 `POST /api/periods/{id}/roster` 上传员工花名册
4. 通过 `POST /api/periods/{id}/process` 处理数据生成汇总和扣款明细
5. 通过 `GET /api/periods/{id}/charges/export?part=personal|unit` 导出结果

## 主要 API 接口

- `GET/POST /api/periods` - 账期管理
- `POST /api/periods/{id}/files` - 单文件上传
- `POST /api/periods/{id}/files/batch` - 批量文件上传
- `POST /api/periods/{id}/roster` - 花名册上传（Excel/CSV 需包含 姓名、证件号码、部门 列）
- `POST /api/periods/{id}/process` - 数据处理和聚合
- `GET /api/periods/{id}/summary` - 查看汇总统计
- `GET /api/periods/{id}/charges` - 查看扣款明细
- `GET /api/periods/{id}/charges/export` - 导出 Excel

## 文件结构

```
frontend/
  app/
    page.tsx              # 主界面和交互逻辑
    layout.tsx            # 全局布局和 Provider
    providers.tsx         # Toast 和主题 Provider
  components/ui/          # shadcn/ui 组件
  lib/
    api.ts               # 后端 API 客户端函数
    types.ts             # TypeScript 类型定义

backend/
  main.go                # HTTP 服务器设置和数据库初始化
  internal/
    api/                 # 路由处理器和 HTTP 逻辑
    models/              # GORM 模型定义
    service/             # Excel 解析和业务逻辑
```

## 重要说明

- 系统支持中文文件名和数据处理
- Excel 文件必须遵循特定的列结构才能正确解析
- 医疗保险分为 `medical`（8.5%）和 `serious_illness`（1.5%）两个文件
- 花名册数据用于补充保险文件中缺失的部门信息
- 所有文件上传支持 Excel (.xls/.xlsx) 和 CSV 格式
- 前端使用 shadcn/ui 组件和 Sonner 进行通知提示