package service

import (
	"strings"

	"gorm.io/gorm"
)

// dbDialect 数据库方言能力集中判断（P9.1）
// 所有涉及方言差异的逻辑（向量写入、模糊匹配、全文检索降级）统一走此处，
// 避免散落各处判断导致行为漂移。
type dbDialect struct {
	name string
}

// newDBDialect 从 GORM 连接提取方言名（小写）
func newDBDialect(db *gorm.DB) dbDialect {
	name := ""
	if db != nil {
		name = db.Dialector.Name()
	}
	return dbDialect{name: strings.ToLower(name)}
}

// isPostgres 是否为 PostgreSQL（支持 pgvector 原生向量列）
func (d dbDialect) isPostgres() bool {
	return d.name == "postgres"
}

// isSQLite 是否为 SQLite（无原生向量类型，向量降级为 JSON + 应用层计算）
func (d dbDialect) isSQLite() bool {
	return d.name == "sqlite"
}

// likeExpr 返回方言兼容的模糊匹配表达式（列名 + 一个 ? 参数）
// SQLite 不支持 ILIKE 关键字，用 LOWER 包裹保持与 ILIKE 一致的大小写语义；
// 其余数据库（PostgreSQL）使用 ILIKE。
func (d dbDialect) likeExpr(column string) string {
	if d.isSQLite() {
		return "LOWER(" + column + ") LIKE LOWER(?)"
	}
	return column + " ILIKE ?"
}
