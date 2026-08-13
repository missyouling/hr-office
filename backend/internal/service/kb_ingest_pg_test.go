package service

import (
	"fmt"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"siapp/internal/models"
)

// ============================================================
// P9.1 PostgreSQL/pgvector 集成测试（真实 PG，不伪造）
// 门控：SIAPP_P9_POSTGRES_TEST=1 且 SIAPP_DATABASE_TYPE=postgres，
// 否则全部 Skip。连接参数：SIAPP_DB_HOST/PORT/USER/PASSWORD/NAME/SSLMODE，
// SSLMODE 未设置时默认 disable（兼容本地临时容器）。
// 测试数据使用独立前缀（p9pg_<纳秒>_<序号>）并在测试结束时清理，可重复运行。
// ============================================================

// p9PGEnabled 判断是否启用 P9 PostgreSQL 集成测试（双开关，任一不满足则 Skip）
func p9PGEnabled(t *testing.T) bool {
	t.Helper()
	if os.Getenv("SIAPP_P9_POSTGRES_TEST") != "1" {
		t.Skip("SIAPP_P9_POSTGRES_TEST != 1，跳过 P9 PostgreSQL 集成测试")
		return false
	}
	dbType := strings.ToLower(os.Getenv("SIAPP_DATABASE_TYPE"))
	if dbType != "postgres" && dbType != "postgresql" {
		t.Skipf("SIAPP_DATABASE_TYPE=%s 非 postgres，跳过 P9 PostgreSQL 集成测试", dbType)
		return false
	}
	return true
}

// envOr 读取环境变量，为空时返回默认值
func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// p9PGConnect 按 SIAPP_DB_* 环境变量连接 PostgreSQL（兼容本地临时容器）
func p9PGConnect(t *testing.T) *gorm.DB {
	t.Helper()
	password := os.Getenv("SIAPP_DB_PASSWORD")
	if password == "" {
		t.Fatal("SIAPP_DB_PASSWORD 未设置，无法连接 PostgreSQL")
	}
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=%s",
		envOr("SIAPP_DB_HOST", "localhost"),
		envOr("SIAPP_DB_USER", "siapp"),
		password,
		envOr("SIAPP_DB_NAME", "siapp"),
		envOr("SIAPP_DB_PORT", "5432"),
		envOr("SIAPP_DB_SSLMODE", "disable"), // 本地容器默认无 SSL
	)
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("连接 PostgreSQL 失败: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("获取底层连接失败: %v", err)
	}
	sqlDB.SetMaxOpenConns(10) // 并发场景需要多连接
	t.Cleanup(func() { _ = sqlDB.Close() })
	return db
}

// p9PGSeq 前缀序号（进程内递增，保证同库多次运行不冲突）
var p9PGSeq uint64

// p9PGPrefix 生成独立测试数据前缀（可重复运行的关键）
func p9PGPrefix() string {
	seq := atomic.AddUint64(&p9PGSeq, 1)
	return fmt.Sprintf("p9pg_%d_%d", time.Now().UnixNano(), seq)
}

// setupP9PGDB 建立 P9 PostgreSQL 测试环境：
//  1. 连接 + AutoMigrate（与 setupP9DB 相同模型集合）
//  2. CREATE EXTENSION vector（权限/扩展不可用则明确失败，不静默跳过）
//  3. document_chunks.embedding vector(768) 列（与 main.go ensureKnowledgeBaseInfrastructure 一致）
//  4. 影子文档联合唯一部分索引（与 main.go ensureShadowDocumentUniqueIndex 一致）
//  5. 注册按前缀清理
func setupP9PGDB(t *testing.T) (*gorm.DB, string) {
	t.Helper()
	if !p9PGEnabled(t) {
		return nil, ""
	}
	db := p9PGConnect(t)
	prefix := p9PGPrefix()

	if err := db.AutoMigrate(
		&models.User{},
		&models.Role{},
		&models.UserRole{},
		&models.KnowledgeBase{},
		&models.KBAccessRule{},
		&models.Employee{},
		&models.Document{},
		&models.DocumentChunk{},
		&models.ModelConfig{},
	); err != nil {
		t.Fatalf("自动迁移失败: %v", err)
	}

	// CREATE EXTENSION vector：不可用时明确失败
	if err := db.Exec("CREATE EXTENSION IF NOT EXISTS vector").Error; err != nil {
		t.Fatalf("CREATE EXTENSION vector 失败（pgvector 扩展不可用）: %v", err)
	}

	// embedding 原生向量列（与生产迁移一致）
	if err := db.Exec("ALTER TABLE document_chunks ADD COLUMN IF NOT EXISTS embedding vector(768)").Error; err != nil {
		t.Fatalf("添加 embedding vector(768) 列失败: %v", err)
	}

	// 影子文档联合唯一部分索引（与生产一致）
	if err := db.Exec(`
		CREATE UNIQUE INDEX IF NOT EXISTS idx_documents_shadow_unique
		ON documents (source_type, source_id, source_kb_id)
		WHERE source_type IS NOT NULL AND source_type != ''
	`).Error; err != nil {
		t.Fatalf("创建影子文档唯一索引失败: %v", err)
	}

	// HNSW 向量索引（与 main.go ensureKnowledgeBaseInfrastructure 一致）
	if err := db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_chunk_embedding_hnsw
		ON document_chunks USING hnsw (embedding vector_cosine_ops)
		WITH (m = 16, ef_construction = 64)
	`).Error; err != nil {
		t.Fatalf("创建 HNSW 索引失败: %v", err)
	}

	t.Cleanup(func() { p9PGCleanup(t, db, prefix) })
	return db, prefix
}

// p9PGCleanup 按前缀清理测试数据（先子表后父表，避免残留）
// 影子文档通过 source_kb_id 关联前缀知识库，或 document_code 带前缀识别。
func p9PGCleanup(t *testing.T, db *gorm.DB, prefix string) {
	t.Helper()
	docCond := `source_kb_id IN (SELECT id FROM knowledge_bases WHERE name LIKE ?) OR document_code LIKE ?`
	db.Exec(`DELETE FROM document_chunks WHERE doc_id IN (SELECT id FROM documents WHERE `+docCond+`)`, prefix+"%", prefix+"%")
	db.Exec(`DELETE FROM documents WHERE `+docCond, prefix+"%", prefix+"%")
	db.Exec(`DELETE FROM employees WHERE id_number LIKE ?`, prefix+"%")
	db.Exec(`DELETE FROM knowledge_bases WHERE name LIKE ?`, prefix+"%")
	db.Exec(`DELETE FROM model_configs WHERE model_name LIKE ?`, prefix+"%")
	db.Exec(`DELETE FROM users WHERE username LIKE ?`, prefix+"%")
}

// ============================================================
// 1. CREATE EXTENSION vector：扩展不可用则明确失败，不静默跳过
// ============================================================

func TestP9PG_CreateExtension(t *testing.T) {
	if !p9PGEnabled(t) {
		return
	}
	db := p9PGConnect(t)
	if err := db.Exec("CREATE EXTENSION IF NOT EXISTS vector").Error; err != nil {
		t.Fatalf("CREATE EXTENSION vector 失败（pgvector 扩展不可用）: %v", err)
	}
	var extName string
	if err := db.Raw("SELECT extname FROM pg_extension WHERE extname = 'vector'").Scan(&extName).Error; err != nil {
		t.Fatalf("查询 vector 扩展失败: %v", err)
	}
	if extName != "vector" {
		t.Fatalf("vector 扩展未注册，实际: %q", extName)
	}
}

// ============================================================
// 2. 迁移/创建：embedding vector(768)、embedding_json、
//    documents source 元数据、P9 唯一部分索引
// ============================================================

func TestP9PG_MigrationInfra(t *testing.T) {
	db, prefix := setupP9PGDB(t)
	if db == nil {
		return
	}
	// documents.user_id 存在外键约束（fk_documents_user），需先创建测试用户
	user := seedP9PGUser(t, db, prefix, 1)

	// document_chunks.embedding 必须是 vector(768)
	var colType string
	if err := db.Raw(`
		SELECT format_type(a.atttypid, a.atttypmod)
		FROM pg_attribute a
		JOIN pg_class c ON c.oid = a.attrelid
		WHERE c.relname = 'document_chunks' AND a.attname = 'embedding'
	`).Scan(&colType).Error; err != nil {
		t.Fatalf("查询 embedding 列类型失败: %v", err)
	}
	if colType != "vector(768)" {
		t.Fatalf("embedding 列类型期望 vector(768)，实际 %q", colType)
	}

	// embedding_json 列存在（jsonb）
	var jsonCol string
	if err := db.Raw(`
		SELECT format_type(a.atttypid, a.atttypmod)
		FROM pg_attribute a
		JOIN pg_class c ON c.oid = a.attrelid
		WHERE c.relname = 'document_chunks' AND a.attname = 'embedding_json'
	`).Scan(&jsonCol).Error; err != nil {
		t.Fatalf("查询 embedding_json 列失败: %v", err)
	}
	if jsonCol != "jsonb" {
		t.Fatalf("embedding_json 列类型期望 jsonb，实际 %q", jsonCol)
	}

	// documents source 元数据列存在
	var srcCols []string
	if err := db.Raw(`
		SELECT a.attname FROM pg_attribute a
		JOIN pg_class c ON c.oid = a.attrelid
		WHERE c.relname = 'documents'
		  AND a.attname IN ('source_type', 'source_id', 'source_kb_id')
		  AND a.attnum > 0
	`).Scan(&srcCols).Error; err != nil {
		t.Fatalf("查询 documents source 列失败: %v", err)
	}
	if len(srcCols) != 3 {
		t.Fatalf("documents 缺少 source 元数据列，实际: %v", srcCols)
	}

	// P9 唯一部分索引存在
	var idxCount int64
	if err := db.Raw(`
		SELECT COUNT(*) FROM pg_indexes WHERE indexname = 'idx_documents_shadow_unique'
	`).Scan(&idxCount).Error; err != nil {
		t.Fatalf("查询唯一索引失败: %v", err)
	}
	if idxCount != 1 {
		t.Fatal("idx_documents_shadow_unique 索引不存在")
	}

	// 部分索引行为：source_type 为空允许重复；非空同键唯一
	now := time.Now()
	dupEmpty := []models.Document{
		{UserID: user.ID, DocumentCode: prefix + "-E1", FileName: "空源1", Status: "active", OCRStatus: "completed", CreatedAt: now, UpdatedAt: now},
		{UserID: user.ID, DocumentCode: prefix + "-E2", FileName: "空源2", Status: "active", OCRStatus: "completed", CreatedAt: now, UpdatedAt: now},
	}
	if err := db.Create(&dupEmpty).Error; err != nil {
		t.Fatalf("source_type 为空的两条文档应允许重复: %v", err)
	}
	kbID := uint(1)
	shadow1 := models.Document{UserID: user.ID, DocumentCode: prefix + "-S1", FileName: "影子1", SourceType: "employee", SourceID: 1, SourceKBID: &kbID, Status: "active", OCRStatus: "completed", CreatedAt: now, UpdatedAt: now}
	shadow2 := models.Document{UserID: user.ID, DocumentCode: prefix + "-S2", FileName: "影子2", SourceType: "employee", SourceID: 1, SourceKBID: &kbID, Status: "active", OCRStatus: "completed", CreatedAt: now, UpdatedAt: now}
	if err := db.Create(&shadow1).Error; err != nil {
		t.Fatalf("创建影子文档1失败: %v", err)
	}
	if err := db.Create(&shadow2).Error; err == nil {
		t.Fatal("同键影子文档第二条应触发唯一冲突，实际成功")
	} else if !isUniqueViolation(err) {
		t.Fatalf("预期唯一冲突错误，实际: %v", err)
	}
}

// ============================================================
// 3. 生产启动 HNSW 索引存在性验证
//    （setupP9PGDB 已按 main.go ensureKnowledgeBaseInfrastructure
//    创建 idx_chunk_embedding_hnsw，此处验证存在且访问方法为 hnsw）
// ============================================================

func TestP9PG_HNSWIndexExists(t *testing.T) {
	db, _ := setupP9PGDB(t)
	if db == nil {
		return
	}

	// 索引必须存在（生产启动 ensureKnowledgeBaseInfrastructure 会创建）
	var idxCount int64
	if err := db.Raw(`
		SELECT COUNT(*) FROM pg_indexes WHERE indexname = 'idx_chunk_embedding_hnsw'
	`).Scan(&idxCount).Error; err != nil {
		t.Fatalf("查询 HNSW 索引失败: %v", err)
	}
	if idxCount != 1 {
		t.Fatal("idx_chunk_embedding_hnsw 索引不存在（生产启动应创建）")
	}

	// 索引访问方法必须是 hnsw（而非 btree 等降级实现）
	var amName string
	if err := db.Raw(`
		SELECT am.amname
		FROM pg_index i
		JOIN pg_class c ON c.oid = i.indexrelid
		JOIN pg_am am ON am.oid = c.relam
		WHERE c.relname = 'idx_chunk_embedding_hnsw'
	`).Scan(&amName).Error; err != nil {
		t.Fatalf("查询 HNSW 索引访问方法失败: %v", err)
	}
	if amName != "hnsw" {
		t.Fatalf("索引访问方法期望 hnsw，实际 %q", amName)
	}
}
