# 人事行政管理系统后端 (hr-office)

基于 Go + SQLite 的 REST API，用于管理社保账期、导入社保局导出的各险种明细、花名册数据，并生成汇总及扣款明细结果。

## 环境要求

- Go 1.24 及以上（当前使用 1.24.5）
- SQLite 随框架内嵌，无需额外安装

## 目录结构

```
backend/
  main.go                 // 程序入口，启动 HTTP 服务
  internal/
    api/                  // 路由与 HTTP 处理逻辑
    models/               // GORM 模型定义
    service/              // Excel 解析与业务处理
```

## 启动

```bash
cd backend
go run .
```

默认监听 `:8080`，数据库文件保存在 `data/siapp.db`。可通过环境变量覆盖：

- `SIAPP_ADDR`：HTTP 监听地址（默认 `:8080`）
- `SIAPP_DATABASE_PATH`：SQLite 文件路径（默认 `./data/siapp.db`）

## API 概览

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| `GET /api/periods` | 查询所有账期 |
| `POST /api/periods` | 创建账期，JSON `{ "year_month": "2025-08" }` |
| `GET /api/periods/{id}` | 查看单个账期 |
| `GET /api/periods/{id}/files` | 查看已上传的险种明细文件 |
| `POST /api/periods/{id}/files` | 单文件上传（`multipart/form-data`），字段：`scheme`、`part`、`file` |
| `POST /api/periods/{id}/files/batch` | 批量上传险种明细，表单需包含多组 `scheme`、`part`、`files` |
| `GET /api/periods/{id}/roster` | 查看花名册条目 |
| `POST /api/periods/{id}/roster` | 上传花名册（支持 xls/xlsx/csv），需含“姓名”“证件号码”“部门”列 |
| `POST /api/periods/{id}/process` | 执行数据处理，生成社保总表与单位/个人扣款明细 |
| `GET /api/periods/{id}/summary` | 获取各险种汇总（人数、基数合计、金额合计） |
| `GET /api/periods/{id}/charges?part=personal|unit` | 获取个人或单位扣款明细（JSON） |
| `GET /api/periods/{id}/charges/export?part=personal|unit` | 导出个人/单位扣款明细 Excel |

### scheme / part 取值

- `scheme`: `pension`（养老）、`medical`（基本医疗）、`serious_illness`（大额/生育）、`unemployment`、`injury`
- `part`: `personal`（个人）、`unit`（单位）

> 注意：单位医疗 10% 由 `medical`（8.5%）与 `serious_illness`（1.5%）两份文件构成，系统会在处理时自动合并。

## 处理流程

1. `POST /periods` 创建账期。
2. 依次上传花名册（可选）及各险种明细。支持单文件或 `POST /files/batch` 批量上传。
3. `POST /periods/{id}/process` 执行数据清洗与汇总。
4. 使用 `GET /summary`、`GET /charges`、`GET /charges/export` 查询或导出结果。

## 花名册导入说明

花名册用于补全原始险种明细中缺失的部门信息。花名册文件需至少包含：

- 姓名（`姓名`）
- 证件号码（`证件号码`）
- 部门（`部门`）

系统按证件号码与明细匹配；导入花名册后再次处理数据即可看到部门字段被填充。

## 下一步建议

- 根据实际模板扩展 Excel 导出样式（套用格式、公式等）。
- 增加更多校验（如金额/基数比对、人员缺失预警）。
- 编写集成测试覆盖批量上传与导出流程。
