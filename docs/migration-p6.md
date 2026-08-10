# P6 旧系统数据迁移手册

## 功能说明

将 `office-supply-analytics` 旧系统（SQLite 文件）中的字典数据与历史业务单据迁移到 hr-office 新表。

## 前置条件

1. 旧系统 SQLite 数据库文件可访问（需包含完整表结构，见 `office-supply-analytics/schema.sql` 和 `canteen-schema.sql`）
2. 目标库已创建（SQLite / PostgreSQL 均可），表结构由 GORM AutoMigrate 自动生成
3. Go 1.25+，项目依赖已安装（`go mod download`）

## 运行命令

```bash
# 仅迁移字典（安全先行）
go run ./cmd/migrate-legacy \
  --source /path/to/source.db \
  --target file:./data/target.db \
  --only-dictionaries

# 试运行：查看将要执行的语句但不写入
go run ./cmd/migrate-legacy \
  --source /path/to/source.db \
  --target file:./data/target.db \
  --dry-run

# 全量迁移（字典 + 业务单据）
go run ./cmd/migrate-legacy \
  --source /path/to/source.db \
  --target file:./data/target.db
```

如果未指定 `--target`，将从环境变量 `SIAPP_DATABASE_PATH` 或 `DATABASE_URL` 读取。

## 迁移范围

| 类型 | 源表 → 目标表 | 去重策略 |
|------|---------------|----------|
| 字典 | categories → office_categories | 按 name |
| 字典 | suppliers → office_suppliers | 按 name |
| 字典 | supplies → office_supplies | 按 name |
| 字典 | canteen_categories → canteen_categories | 按 name |
| 字典 | canteen_supplies → canteen_supplies | 按 name |
| 字典 | canteen_expense_categories → canteen_expense_categories | 按 name |
| 业务 | purchases → office_purchases | 按 order_no |
| 业务 | purchase_items → office_purchase_items | 外键重映射 |
| 业务 | payment_requests → office_payment_requests | 按 request_no |
| 业务 | canteen_purchases → canteen_purchases | 按 order_no |
| 业务 | canteen_purchase_items → canteen_purchase_items | 外键重映射 |
| 业务 | canteen_other_expenses → canteen_other_expenses | 无 |
| 业务 | canteen_daily_income → canteen_daily_income | 按 income_date |
| 业务 | canteen_resource_fees → canteen_resource_fees | 无 |
| 业务 | canteen_weekly_menu → canteen_weekly_menu | 无 |
| 业务 | canteen_menu_templates → canteen_menu_templates | 无 |
| 业务 | canteen_card_recharges → canteen_card_recharges | 按 external_sn |
| 业务 | canteen_card_refunds → canteen_card_refunds | 按 external_sn |

## 验证步骤

1. 先 `--dry-run` 确认输出无异常
2. 再 `--only-dictionaries` 迁移字典，检查目标库记录数
3. 全量迁移后对比源库与目标库各表记录数（排除因重复跳过的部分）

## 已知限制

1. **不支持 Cloudflare D1**：仅支持 SQLite 文件 DSN
2. **数据库类型校验**：运行时禁用（直接 execute），不做 DB 类型校验
3. **外键映射失败**：当源记录的外键在目标库中找不到对应新 ID 时，明细记录会被跳过并打印日志
4. **菜单模板 JSON**：源库 data 为 TEXT，直接转为 datatypes.JSON 存入目标库，不做格式校验
5. **时区**：所有时间统一按 `2006-01-02 15:04:05` 格式解析，不做时区转换
