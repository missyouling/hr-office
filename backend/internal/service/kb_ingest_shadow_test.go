package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"siapp/internal/models"
	"siapp/internal/service/docreader"
)

// ============================================================
// P9.1 集成测试：非 archives 源记录 → 影子 Document → 可检索闭环
// 覆盖：影子文档创建/复用、chunk 关联真实 document、重复 ingest 幂等、
//      全文命中、embedding JSON 降级写入、统计与错误隔离
// ============================================================

// mockParserByContent 按输入文本内容返回解析结果/错误（用于错误隔离测试）
type mockParserByContent struct {
	parseFunc func(text string) (*docreader.ParseResult, error)
}

func (m *mockParserByContent) Parse(_ context.Context, req docreader.ParseRequest) (*docreader.ParseResult, error) {
	data, err := os.ReadFile(req.FilePath)
	if err != nil {
		return nil, err
	}
	return m.parseFunc(string(data))
}

// setupP9DB 创建文件型 SQLite（多连接共享同一库，支持并发场景验证）
// - busy_timeout 通过 DSN _pragma 生效（连接级，每个连接自动应用）
// - _txlock=immediate：写事务立即获取写锁，避免 WAL 下"读后写"快照过期报 locked
// - WAL 为数据库级，需打开连接后 Exec 一次并持久化到 db 文件
// 并发事务串行化而非直接报锁，配合唯一索引验证并发兜底
func setupP9DB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := filepath.Join(t.TempDir(), "p9.db") +
		"?_pragma=busy_timeout(5000)&_txlock=immediate"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开文件 SQLite 失败: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("获取底层连接失败: %v", err)
	}
	sqlDB.SetMaxOpenConns(10) // 并发场景需要多连接
	if err := db.Exec("PRAGMA journal_mode = WAL").Error; err != nil {
		t.Fatalf("启用 WAL 失败: %v", err)
	}

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

	// 与 main.go ensureShadowDocumentUniqueIndex 保持一致：影子文档联合唯一部分索引
	if err := db.Exec(`
		CREATE UNIQUE INDEX IF NOT EXISTS idx_documents_shadow_unique
		ON documents (source_type, source_id, source_kb_id)
		WHERE source_type IS NOT NULL AND source_type != ''
	`).Error; err != nil {
		t.Fatalf("创建影子文档唯一索引失败: %v", err)
	}
	return db
}

// seedP9KB 插入 employee 来源知识库（public + employee = 全员可见，kbID=0 可解析到）
func seedP9KB(t *testing.T, db *gorm.DB) models.KnowledgeBase {
	t.Helper()
	kb := models.KnowledgeBase{
		Name:         "P9测试知识库",
		SourceModule: "employee",
		Visibility:   "public",
	}
	if err := db.Create(&kb).Error; err != nil {
		t.Fatalf("创建知识库失败: %v", err)
	}
	return kb
}

// seedP9User 创建测试用户（AccessibleKBIDs 解析依赖 users 表）
func seedP9User(t *testing.T, db *gorm.DB, id uint) {
	t.Helper()
	user := models.User{
		ID:       id,
		Username: fmt.Sprintf("user%d", id),
		Email:    fmt.Sprintf("user%d@test.local", id),
		Password: "hash-placeholder",
	}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("创建用户失败: %v", err)
	}
}

// assignP9Role 为用户分配角色（user_roles 关联表）
func assignP9Role(t *testing.T, db *gorm.DB, userID uint, roleName string) {
	t.Helper()
	var role models.Role
	if err := db.Where("name = ?", roleName).First(&role).Error; err != nil {
		role = models.Role{Name: roleName, Label: roleName, IsSystem: true}
		if err := db.Create(&role).Error; err != nil {
			t.Fatalf("创建角色失败: %v", err)
		}
	}
	ur := models.UserRole{UserID: userID, RoleID: role.ID}
	if err := db.Create(&ur).Error; err != nil {
		t.Fatalf("分配角色失败: %v", err)
	}
}

// p9EmpSeq 员工 IDNumber 全局递增计数器（多次 seed 调用不冲突）
var p9EmpSeq uint64

// seedP9Employees 插入指定姓名的员工（IDNumber 全局唯一以满足联合唯一约束）
func seedP9Employees(t *testing.T, db *gorm.DB, names []string) []models.Employee {
	t.Helper()
	emps := make([]models.Employee, 0, len(names))
	for _, name := range names {
		seq := atomic.AddUint64(&p9EmpSeq, 1)
		emp := models.Employee{
			UserID:     1,
			IDNumber:   fmt.Sprintf("P9EMP%04d", seq),
			Name:       name,
			Department: "研发部",
		}
		if err := db.Create(&emp).Error; err != nil {
			t.Fatalf("创建员工失败: %v", err)
		}
		emps = append(emps, emp)
	}
	return emps
}

// newP9Service 构造测试用 KBIngestService
func newP9Service(t *testing.T, db *gorm.DB, parser docParser) *KBIngestService {
	t.Helper()
	return &KBIngestService{
		db:           db,
		parser:       parser,
		embeddingSvc: NewEmbeddingService(db),
	}
}

// p9FixedParser 固定返回两章节解析结果
func p9FixedParser() *mockParserByContent {
	return &mockParserByContent{
		parseFunc: func(text string) (*docreader.ParseResult, error) {
			return &docreader.ParseResult{
				FullText: text,
				Sections: []docreader.ParseSection{
					{Title: "第一章", Content: text, Level: 1},
				},
			}, nil
		},
	}
}

// countShadowDocs 统计指定来源的影子文档数量
func countShadowDocs(t *testing.T, db *gorm.DB, sourceType string) int64 {
	t.Helper()
	var n int64
	if err := db.Model(&models.Document{}).Where("source_type = ?", sourceType).Count(&n).Error; err != nil {
		t.Fatalf("统计影子文档失败: %v", err)
	}
	return n
}

// ============================================================
// 1. 影子文档创建 + chunk 关联真实 document
// ============================================================

func TestP9ShadowDocument_CreatedAndChunkLinked(t *testing.T) {
	db := setupP9DB(t)
	seedP9User(t, db, 1)
	kb := seedP9KB(t, db)
	seedP9Employees(t, db, []string{"张三", "李四"})

	svc := newP9Service(t, db, p9FixedParser())
	result, err := svc.Ingest(context.Background(), 1, IngestRequest{KBID: kb.ID, SourceModule: "employee"})
	if err != nil {
		t.Fatalf("Ingest 返回错误: %v", err)
	}
	if result.Ingested != 2 || result.Skipped != 0 {
		t.Fatalf("统计异常: ingested=%d skipped=%d", result.Ingested, result.Skipped)
	}

	// 影子文档：2 条 employee 来源记录
	if n := countShadowDocs(t, db, "employee"); n != 2 {
		t.Fatalf("影子文档数量期望 2，实际 %d", n)
	}

	// 每个 chunk 的 doc_id 必须指向真实 documents.id，且来源元数据正确
	var chunks []models.DocumentChunk
	if err := db.Find(&chunks).Error; err != nil {
		t.Fatalf("查询 chunks 失败: %v", err)
	}
	if len(chunks) == 0 {
		t.Fatal("期望存在 chunks，实际为空")
	}
	for _, c := range chunks {
		var doc models.Document
		if err := db.First(&doc, c.DocID).Error; err != nil {
			t.Fatalf("chunk doc_id=%d 指向不存在的 documents 记录（孤儿引用）: %v", c.DocID, err)
		}
		if doc.SourceType != "employee" || doc.SourceID == 0 {
			t.Errorf("影子文档来源元数据缺失: source_type=%q source_id=%d", doc.SourceType, doc.SourceID)
		}
		if doc.SourceKBID == nil || *doc.SourceKBID != kb.ID {
			t.Errorf("影子文档 KB 元数据错误: %v", doc.SourceKBID)
		}
		if doc.SourceDept != "研发部" {
			t.Errorf("影子文档部门元数据错误: %q", doc.SourceDept)
		}
		if doc.ContentText == "" {
			t.Error("影子文档 content_text 为空，无法被全文检索")
		}
	}
}

// ============================================================
// 2. 重复 ingest 幂等：影子文档复用、chunk 不翻倍
// ============================================================

func TestP9ShadowDocument_ReingestIdempotent(t *testing.T) {
	db := setupP9DB(t)
	seedP9User(t, db, 1)
	kb := seedP9KB(t, db)
	seedP9Employees(t, db, []string{"王五"})

	svc := newP9Service(t, db, p9FixedParser())
	req := IngestRequest{KBID: kb.ID, SourceModule: "employee"}

	if _, err := svc.Ingest(context.Background(), 1, req); err != nil {
		t.Fatalf("首次 Ingest 失败: %v", err)
	}
	var chunkCount1 int64
	db.Model(&models.DocumentChunk{}).Count(&chunkCount1)

	// 重复 ingest
	if _, err := svc.Ingest(context.Background(), 1, req); err != nil {
		t.Fatalf("重复 Ingest 失败: %v", err)
	}

	// 影子文档仍只有 1 条
	if n := countShadowDocs(t, db, "employee"); n != 1 {
		t.Fatalf("重复 ingest 后影子文档数量期望 1，实际 %d", n)
	}
	// chunks 不翻倍（先删后建）
	var chunkCount2 int64
	db.Model(&models.DocumentChunk{}).Count(&chunkCount2)
	if chunkCount2 != chunkCount1 {
		t.Fatalf("重复 ingest 后 chunk 数量变化: 首次 %d，二次 %d", chunkCount1, chunkCount2)
	}
}

// ============================================================
// 3. 全文命中：影子文档可被检索
// ============================================================

func TestP9ShadowDocument_FullTextHit(t *testing.T) {
	db := setupP9DB(t)
	seedP9User(t, db, 1)
	kb := seedP9KB(t, db)
	seedP9Employees(t, db, []string{"欧阳锋"})

	svc := newP9Service(t, db, p9FixedParser())
	if _, err := svc.Ingest(context.Background(), 1, IngestRequest{KBID: kb.ID, SourceModule: "employee"}); err != nil {
		t.Fatalf("Ingest 失败: %v", err)
	}

	// SQLite 无 tsvector，FullTextSearch 自动降级到 LIKE 检索 documents
	retrieval := NewRetrievalService(db, NewEmbeddingService(db))
	results, err := retrieval.FullTextSearch(1, "欧阳锋", 10, 0)
	if err != nil {
		t.Fatalf("FullTextSearch 失败: %v", err)
	}
	if len(results) == 0 {
		t.Fatal("期望全文检索命中影子文档，实际无结果")
	}
	var doc models.Document
	if err := db.First(&doc, results[0].DocID).Error; err != nil {
		t.Fatalf("检索结果 doc_id=%d 不存在: %v", results[0].DocID, err)
	}
	if doc.SourceType != "employee" {
		t.Errorf("检索命中的不是影子文档: source_type=%q", doc.SourceType)
	}
}

// ============================================================
// 4. embedding 写入：SQLite 降级写 embedding_json（不触碰 vector 列）
// ============================================================

func TestP9ShadowDocument_EmbeddingJSONWritten(t *testing.T) {
	db := setupP9DB(t)
	seedP9User(t, db, 1)
	kb := seedP9KB(t, db)
	seedP9Employees(t, db, []string{"洪七公"})

	// 本地 mock embedding API
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[{"embedding":[0.1,0.2,0.3,0.4]}]}`))
	}))
	defer srv.Close()

	userID := uint(1)
	cfg := models.ModelConfig{
		UserID:      &userID,
		ConfigType:  "embedding",
		Provider:    "test",
		ModelName:   "test-embed",
		APIEndpoint: srv.URL,
		Enabled:     true,
		IsDefault:   true,
	}
	if err := db.Create(&cfg).Error; err != nil {
		t.Fatalf("创建 embedding 配置失败: %v", err)
	}

	svc := newP9Service(t, db, p9FixedParser())
	if _, err := svc.Ingest(context.Background(), userID, IngestRequest{KBID: kb.ID, SourceModule: "employee"}); err != nil {
		t.Fatalf("Ingest 失败: %v", err)
	}

	var chunks []models.DocumentChunk
	if err := db.Find(&chunks).Error; err != nil {
		t.Fatalf("查询 chunks 失败: %v", err)
	}
	if len(chunks) == 0 {
		t.Fatal("期望存在 chunks")
	}
	for _, c := range chunks {
		if c.IndexStatus != models.IndexStatusReady {
			t.Errorf("chunk %d index_status 期望 ready，实际 %q", c.ID, c.IndexStatus)
		}
		if len(c.EmbeddingJSON) == 0 {
			t.Errorf("chunk %d embedding_json 为空（SQLite 降级写入失败）", c.ID)
		}
		var vec []float64
		if err := json.Unmarshal(c.EmbeddingJSON, &vec); err != nil || len(vec) == 0 {
			t.Errorf("chunk %d embedding_json 不是有效向量: %v", c.ID, err)
		}
	}
}

// ============================================================
// 5. 统计保留 + 错误隔离：失败记录不污染已成功记录
// ============================================================

func TestP9ShadowDocument_ErrorIsolation(t *testing.T) {
	db := setupP9DB(t)
	seedP9User(t, db, 1)
	kb := seedP9KB(t, db)
	seedP9Employees(t, db, []string{"黄药师", "段智兴"})

	// 文本含 "FAIL" 的记录解析失败
	parser := &mockParserByContent{
		parseFunc: func(text string) (*docreader.ParseResult, error) {
			if strings.Contains(text, "FAIL") {
				return nil, fmt.Errorf("模拟解析失败")
			}
			return &docreader.ParseResult{
				FullText: text,
				Sections: []docreader.ParseSection{{Title: "正文", Content: text, Level: 1}},
			}, nil
		},
	}
	// 让第二条员工记录文本包含 FAIL
	db.Model(&models.Employee{}).Where("name = ?", "段智兴").Update("remarks", "FAIL")

	svc := newP9Service(t, db, parser)
	result, err := svc.Ingest(context.Background(), 1, IngestRequest{KBID: kb.ID, SourceModule: "employee"})
	if err != nil {
		t.Fatalf("Ingest 不应整体失败: %v", err)
	}

	// 统计：1 成功 + 1 失败
	if result.Scanned != 2 {
		t.Errorf("Scanned 期望 2，实际 %d", result.Scanned)
	}
	if result.Ingested != 1 {
		t.Errorf("Ingested 期望 1，实际 %d", result.Ingested)
	}
	if result.Skipped != 1 {
		t.Errorf("Skipped 期望 1，实际 %d", result.Skipped)
	}
	if len(result.Errors) != 1 {
		t.Errorf("Errors 期望 1 条，实际 %d 条", len(result.Errors))
	}

	// 失败记录不产生任何 chunk（无孤儿）
	var failedChunks int64
	db.Model(&models.DocumentChunk{}).
		Joins("JOIN documents ON documents.id = document_chunks.doc_id").
		Where("documents.source_id = ? AND documents.source_type = ?", 2, "employee").
		Count(&failedChunks)
	if failedChunks != 0 {
		t.Errorf("失败记录不应产生 chunk，实际 %d 条", failedChunks)
	}

	// 成功记录仍可检索（错误不污染已成功记录）
	retrieval := NewRetrievalService(db, NewEmbeddingService(db))
	results, err := retrieval.FullTextSearch(1, "黄药师", 10, 0)
	if err != nil {
		t.Fatalf("FullTextSearch 失败: %v", err)
	}
	if len(results) == 0 {
		t.Fatal("成功记录应可被检索，实际无结果")
	}
}

// ============================================================
// 6. KB 范围过滤：kbID>0 限定 KB；kbID=0 仅检索有权限 KB
// ============================================================

func TestP9ShadowDocument_KBScopeFilter(t *testing.T) {
	db := setupP9DB(t)
	seedP9User(t, db, 1)
	kb1 := seedP9KB(t, db)
	kb2 := models.KnowledgeBase{Name: "P9测试知识库2", SourceModule: "employee", Visibility: "public"}
	if err := db.Create(&kb2).Error; err != nil {
		t.Fatalf("创建知识库2失败: %v", err)
	}
	// 无权限 KB：restricted 且无任何访问规则
	kb3 := models.KnowledgeBase{Name: "P9无权限知识库", SourceModule: "employee", Visibility: "restricted"}
	if err := db.Create(&kb3).Error; err != nil {
		t.Fatalf("创建无权限知识库失败: %v", err)
	}

	// 手动构造影子文档（模拟 ingest 产物，聚焦检索过滤语义）：
	// kb1 含郭靖/黄蓉，kb2 含杨过，kb3 含小龙女；另加一条 archives 档案文档（source_kb_id 为空）
	now := time.Now()
	docs := []models.Document{
		{UserID: 1, DocumentCode: "T-1", FileName: "员工-郭靖", ContentText: "郭靖 降龙十八掌", SourceType: "employee", SourceID: 1, SourceKBID: &kb1.ID, SourceDept: "研发部", Status: "active", OCRStatus: "completed", CreatedAt: now, UpdatedAt: now},
		{UserID: 1, DocumentCode: "T-2", FileName: "员工-黄蓉", ContentText: "黄蓉 打狗棒法", SourceType: "employee", SourceID: 2, SourceKBID: &kb1.ID, SourceDept: "研发部", Status: "active", OCRStatus: "completed", CreatedAt: now, UpdatedAt: now},
		{UserID: 1, DocumentCode: "T-3", FileName: "员工-杨过", ContentText: "杨过 黯然销魂掌", SourceType: "employee", SourceID: 3, SourceKBID: &kb2.ID, SourceDept: "研发部", Status: "active", OCRStatus: "completed", CreatedAt: now, UpdatedAt: now},
		{UserID: 1, DocumentCode: "T-4", FileName: "员工-小龙女", ContentText: "小龙女 玉女心经", SourceType: "employee", SourceID: 4, SourceKBID: &kb3.ID, SourceDept: "研发部", Status: "active", OCRStatus: "completed", CreatedAt: now, UpdatedAt: now},
		{UserID: 1, DocumentCode: "T-5", FileName: "档案-入职协议", ContentText: "档案内容 保密协议", Status: "active", OCRStatus: "completed", CreatedAt: now, UpdatedAt: now},
	}
	if err := db.Create(&docs).Error; err != nil {
		t.Fatalf("构造影子文档失败: %v", err)
	}

	retrieval := NewRetrievalService(db, NewEmbeddingService(db))

	// kbID>0：只命中该 KB 的影子文档
	results, err := retrieval.FullTextSearch(1, "郭靖", 10, kb1.ID)
	if err != nil {
		t.Fatalf("FullTextSearch(kb1) 失败: %v", err)
	}
	if len(results) == 0 {
		t.Fatal("kb1 内应命中郭靖")
	}
	results, err = retrieval.FullTextSearch(1, "郭靖", 10, kb2.ID)
	if err != nil {
		t.Fatalf("FullTextSearch(kb2) 失败: %v", err)
	}
	if len(results) != 0 {
		t.Fatalf("kb2 内不应命中郭靖（跨 KB 泄漏），实际 %d 条", len(results))
	}

	// kbID=0：有权限 KB（kb1/kb2）命中，无权限 KB（kb3）不命中
	results, err = retrieval.FullTextSearch(1, "黄蓉", 10, 0)
	if err != nil {
		t.Fatalf("FullTextSearch(0) 失败: %v", err)
	}
	if len(results) == 0 {
		t.Fatal("kbID=0 应命中有权限 KB 内的黄蓉")
	}
	results, err = retrieval.FullTextSearch(1, "杨过", 10, 0)
	if err != nil {
		t.Fatalf("FullTextSearch(0) 失败: %v", err)
	}
	if len(results) == 0 {
		t.Fatal("kbID=0 应命中有权限 KB 内的杨过")
	}
	results, err = retrieval.FullTextSearch(1, "小龙女", 10, 0)
	if err != nil {
		t.Fatalf("FullTextSearch(0) 失败: %v", err)
	}
	if len(results) != 0 {
		t.Fatalf("kbID=0 不应命中无权限 KB 内的小龙女，实际 %d 条", len(results))
	}

	// 档案文档（source_kb_id 为空）在 kbID=0 时仍可检索（保持既有权限语义）
	results, err = retrieval.FullTextSearch(1, "保密协议", 10, 0)
	if err != nil {
		t.Fatalf("FullTextSearch(0) 失败: %v", err)
	}
	if len(results) == 0 {
		t.Fatal("kbID=0 应仍可检索用户自己的档案文档")
	}
}

// ============================================================
// 7. SQLite 向量降级：应用层余弦命中（MatchType=vector）
// ============================================================

// p9Vec768 构造 768 维测试向量（与 embeddingDim 一致）
func p9Vec768() []float64 {
	vec := make([]float64, 768)
	for i := range vec {
		vec[i] = 0.1
	}
	return vec
}

func TestP9ShadowDocument_SQLiteVectorHit(t *testing.T) {
	db := setupP9DB(t)
	seedP9User(t, db, 1)
	kb := seedP9KB(t, db)
	seedP9Employees(t, db, []string{"小龙女"})

	// mock embedding API：返回 768 维固定向量（查询与入库同源 → 余弦相似度≈1）
	vecJSON, _ := json.Marshal(p9Vec768())
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[{"embedding":` + string(vecJSON) + `}]}`))
	}))
	defer srv.Close()

	userID := uint(1)
	cfg := models.ModelConfig{
		UserID:      &userID,
		ConfigType:  "embedding",
		Provider:    "test",
		ModelName:   "test-embed",
		APIEndpoint: srv.URL,
		Enabled:     true,
		IsDefault:   true,
	}
	if err := db.Create(&cfg).Error; err != nil {
		t.Fatalf("创建 embedding 配置失败: %v", err)
	}

	svc := newP9Service(t, db, p9FixedParser())
	if _, err := svc.Ingest(context.Background(), userID, IngestRequest{KBID: kb.ID, SourceModule: "employee"}); err != nil {
		t.Fatalf("Ingest 失败: %v", err)
	}

	retrieval := NewRetrievalService(db, NewEmbeddingService(db))
	results, err := retrieval.vectorSearch(userID, "小龙女", 10, kb.ID)
	if err != nil {
		t.Fatalf("vectorSearch 失败: %v", err)
	}
	if len(results) == 0 {
		t.Fatal("SQLite 向量降级应命中影子文档，实际无结果")
	}
	if results[0].MatchType != "vector" {
		t.Errorf("MatchType 期望 vector，实际 %q", results[0].MatchType)
	}
	if results[0].Score < 0.9 {
		t.Errorf("同源向量余弦相似度应接近 1，实际 %f", results[0].Score)
	}
}

// ============================================================
// 8. 并发 ingest：同源同 KB 唯一；同源不同 KB 各自独立
// ============================================================

func TestP9ShadowDocument_ConcurrentIngest(t *testing.T) {
	db := setupP9DB(t)
	seedP9User(t, db, 1)
	kb := seedP9KB(t, db)
	seedP9Employees(t, db, []string{"周伯通"})

	svc := newP9Service(t, db, p9FixedParser())
	req := IngestRequest{KBID: kb.ID, SourceModule: "employee"}

	// 两个 goroutine 并发 ingest 同源同 KB
	var wg sync.WaitGroup
	results := make([]*IngestResult, 2)
	errs := make([]error, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			results[idx], errs[idx] = svc.Ingest(context.Background(), 1, req)
		}(i)
	}
	wg.Wait()
	for i, err := range errs {
		if err != nil {
			t.Fatalf("并发 ingest #%d 失败: %v", i, err)
		}
		// 冲突后必须复用成功：两个 goroutine 都计入 ingested=1（而非一个失败 Skipped）
		if results[i].Ingested != 1 {
			t.Fatalf("并发 ingest #%d 期望 Ingested=1（冲突后复用），实际 %d（Skipped=%d）",
				i, results[i].Ingested, results[i].Skipped)
		}
	}

	// 唯一键兜底：最终只有 1 个影子文档
	if n := countShadowDocs(t, db, "employee"); n != 1 {
		t.Fatalf("并发同源同 KB 后影子文档期望 1，实际 %d", n)
	}
	// chunks 不翻倍：1 个 section → 1 个 chunk
	var chunkCount int64
	db.Model(&models.DocumentChunk{}).Count(&chunkCount)
	if chunkCount != 1 {
		t.Fatalf("并发后 chunk 期望 1，实际 %d", chunkCount)
	}
}

func TestP9ShadowDocument_ConcurrentDifferentKB(t *testing.T) {
	db := setupP9DB(t)
	seedP9User(t, db, 1)
	kb1 := seedP9KB(t, db)
	kb2 := models.KnowledgeBase{Name: "P9测试知识库2", SourceModule: "employee", Visibility: "public"}
	if err := db.Create(&kb2).Error; err != nil {
		t.Fatalf("创建知识库2失败: %v", err)
	}
	seedP9Employees(t, db, []string{"一灯大师"})

	svc := newP9Service(t, db, p9FixedParser())

	var wg sync.WaitGroup
	errs := make([]error, 2)
	for i, kbID := range []uint{kb1.ID, kb2.ID} {
		wg.Add(1)
		go func(idx int, id uint) {
			defer wg.Done()
			_, errs[idx] = svc.Ingest(context.Background(), 1, IngestRequest{KBID: id, SourceModule: "employee"})
		}(i, kbID)
	}
	wg.Wait()
	for i, err := range errs {
		if err != nil {
			t.Fatalf("并发 ingest #%d 失败: %v", i, err)
		}
	}

	// 同源不同 KB：各自独立影子文档（source_kb_id 不同，不互相覆盖）
	if n := countShadowDocs(t, db, "employee"); n != 2 {
		t.Fatalf("同源不同 KB 影子文档期望 2，实际 %d", n)
	}
	var kbIDs []uint
	db.Model(&models.Document{}).Where("source_type = ?", "employee").Pluck("source_kb_id", &kbIDs)
	if len(kbIDs) != 2 {
		t.Fatalf("source_kb_id 期望 2 个不同值，实际 %d 个", len(kbIDs))
	}
}

// ============================================================
// 9. 向量维度校验（PostgreSQL 原生向量列固定 768 维）
// ============================================================

func TestP9ValidateVectorDim(t *testing.T) {
	if err := validateVectorDim(p9Vec768()); err != nil {
		t.Errorf("768 维向量不应报错: %v", err)
	}
	if err := validateVectorDim(make([]float64, 4)); err == nil {
		t.Error("4 维向量应报维度不符")
	}
	if err := validateVectorDim(nil); err == nil {
		t.Error("空向量应报维度不符")
	}
}

// ============================================================
// 10. 事务回滚：解析结果为空时影子文档/旧分块不残留
// ============================================================

func TestP9ShadowDocument_TransactionRollback(t *testing.T) {
	db := setupP9DB(t)
	seedP9User(t, db, 1)
	kb := seedP9KB(t, db)
	seedP9Employees(t, db, []string{"金轮法王"})

	// 场景 A：首次 ingest 解析返回 (nil, nil) → 事务回滚，影子文档不残留
	// 注意：Ingest 对单条失败不整体失败（Skipped++），需断言统计而非 err
	emptyParser := &mockParserByContent{
		parseFunc: func(_ string) (*docreader.ParseResult, error) {
			return nil, nil
		},
	}
	svc := newP9Service(t, db, emptyParser)
	result, err := svc.Ingest(context.Background(), 1, IngestRequest{KBID: kb.ID, SourceModule: "employee"})
	if err != nil {
		t.Fatalf("Ingest 不应整体失败: %v", err)
	}
	if result.Skipped != 1 || len(result.Errors) != 1 {
		t.Fatalf("解析为空应计为 1 条失败: skipped=%d errors=%d", result.Skipped, len(result.Errors))
	}
	if n := countShadowDocs(t, db, "employee"); n != 0 {
		t.Fatalf("回滚后影子文档应不存在，实际 %d 条", n)
	}
	var chunkCount int64
	db.Model(&models.DocumentChunk{}).Count(&chunkCount)
	if chunkCount != 0 {
		t.Fatalf("回滚后 chunk 应不存在，实际 %d 条", chunkCount)
	}

	// 场景 B：已有影子文档 + chunks，二次 ingest 解析为空 → 旧数据保留
	svc2 := newP9Service(t, db, p9FixedParser())
	if _, err := svc2.Ingest(context.Background(), 1, IngestRequest{KBID: kb.ID, SourceModule: "employee"}); err != nil {
		t.Fatalf("首次正常 ingest 失败: %v", err)
	}
	result, err = svc.Ingest(context.Background(), 1, IngestRequest{KBID: kb.ID, SourceModule: "employee"})
	if err != nil {
		t.Fatalf("二次 Ingest 不应整体失败: %v", err)
	}
	if result.Skipped != 1 {
		t.Fatalf("二次解析为空应计为 1 条失败，实际 %d", result.Skipped)
	}
	if n := countShadowDocs(t, db, "employee"); n != 1 {
		t.Fatalf("二次回滚后影子文档应保留 1 条，实际 %d", n)
	}
	db.Model(&models.DocumentChunk{}).Count(&chunkCount)
	if chunkCount != 1 {
		t.Fatalf("二次回滚后旧 chunk 应保留 1 条，实际 %d", chunkCount)
	}
}

// ============================================================
// 11. EmbeddingFailed 统计：向量化失败计入但不影响 ingested
// ============================================================

func TestP9ShadowDocument_EmbeddingFailed(t *testing.T) {
	db := setupP9DB(t)
	seedP9User(t, db, 1)
	kb := seedP9KB(t, db)
	seedP9Employees(t, db, []string{"裘千仞"})

	// mock embedding API：返回 500，向量化全部失败
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "embedding service down", http.StatusInternalServerError)
	}))
	defer srv.Close()

	userID := uint(1)
	cfg := models.ModelConfig{
		UserID:      &userID,
		ConfigType:  "embedding",
		Provider:    "test",
		ModelName:   "test-embed",
		APIEndpoint: srv.URL,
		Enabled:     true,
		IsDefault:   true,
	}
	if err := db.Create(&cfg).Error; err != nil {
		t.Fatalf("创建 embedding 配置失败: %v", err)
	}

	svc := newP9Service(t, db, p9FixedParser())
	result, err := svc.Ingest(context.Background(), userID, IngestRequest{KBID: kb.ID, SourceModule: "employee"})
	if err != nil {
		t.Fatalf("Ingest 不应整体失败: %v", err)
	}
	// 1 个 section → 1 个 chunk → 向量化失败 1
	if result.Ingested != 1 {
		t.Errorf("Ingested 期望 1，实际 %d", result.Ingested)
	}
	if result.EmbeddingFailed != 1 {
		t.Errorf("EmbeddingFailed 期望 1，实际 %d", result.EmbeddingFailed)
	}
	// 失败 chunk 标记 pending 待重试，不误报 ready
	var pending int64
	db.Model(&models.DocumentChunk{}).Where("index_status = ?", "pending").Count(&pending)
	if pending != 1 {
		t.Errorf("pending chunk 期望 1，实际 %d", pending)
	}
}

// ============================================================
// 12. 跨用户授权 KB：kbID=0 时影子文档不限制 user_id（授权即可命中）
// ============================================================

func TestP9ShadowDocument_CrossUserKBHit(t *testing.T) {
	db := setupP9DB(t)
	seedP9User(t, db, 1) // 被授权用户
	seedP9User(t, db, 2) // KB 所有者/源记录所属用户
	seedP9User(t, db, 3) // 无权限用户

	// restricted KB：所有者 user 2，通过 KBAccessRule 授权 user 1
	kb := models.KnowledgeBase{
		Name:         "P9跨用户授权KB",
		SourceModule: "employee",
		Visibility:   "restricted",
		OwnerID:      uintPtr(2),
	}
	if err := db.Create(&kb).Error; err != nil {
		t.Fatalf("创建知识库失败: %v", err)
	}
	rule := models.KBAccessRule{KnowledgeBaseID: kb.ID, UserID: uintPtr(1)}
	if err := db.Create(&rule).Error; err != nil {
		t.Fatalf("创建访问规则失败: %v", err)
	}

	// 影子文档 user_id=2（源记录属于所有者），source_kb_id=kb
	now := time.Now()
	doc := models.Document{
		UserID:       2,
		DocumentCode: "XU-1",
		FileName:     "员工-跨用户",
		ContentText:  "跨用户授权内容 降龙掌",
		SourceType:   "employee",
		SourceID:     1,
		SourceKBID:   &kb.ID,
		Status:       "active",
		OCRStatus:    "completed",
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if err := db.Create(&doc).Error; err != nil {
		t.Fatalf("构造影子文档失败: %v", err)
	}

	retrieval := NewRetrievalService(db, NewEmbeddingService(db))

	// 被授权用户 1：kbID=0 可命中（KB 有权限，不限制 user_id）
	results, err := retrieval.FullTextSearch(1, "降龙掌", 10, 0)
	if err != nil {
		t.Fatalf("FullTextSearch 失败: %v", err)
	}
	if len(results) == 0 {
		t.Fatal("跨用户授权 KB 的影子文档应可被授权用户检索到")
	}

	// 无权限用户 3：kbID=0 不命中
	results, err = retrieval.FullTextSearch(3, "降龙掌", 10, 0)
	if err != nil {
		t.Fatalf("FullTextSearch 失败: %v", err)
	}
	if len(results) != 0 {
		t.Fatalf("无权限用户不应命中，实际 %d 条", len(results))
	}
}

// uintPtr 返回 *uint（测试辅助）
func uintPtr(v uint) *uint { return &v }

// ============================================================
// 13. 管理员 kbID=0 全量可见（含无权限 KB）
// ============================================================

func TestP9ShadowDocument_AdminFullVisibility(t *testing.T) {
	db := setupP9DB(t)
	seedP9User(t, db, 1) // 普通用户
	seedP9User(t, db, 2) // 管理员
	assignP9Role(t, db, 2, models.RoleAdmin)

	// restricted 无任何规则的 KB（普通用户无权限）
	kb := models.KnowledgeBase{
		Name:         "P9无权限KB",
		SourceModule: "employee",
		Visibility:   "restricted",
		OwnerID:      uintPtr(1),
	}
	if err := db.Create(&kb).Error; err != nil {
		t.Fatalf("创建知识库失败: %v", err)
	}

	// 影子文档 + 普通用户自己的档案文档
	now := time.Now()
	docs := []models.Document{
		{UserID: 1, DocumentCode: "AD-1", FileName: "员工-机密", ContentText: "机密情报 葵花宝典", SourceType: "employee", SourceID: 1, SourceKBID: &kb.ID, Status: "active", OCRStatus: "completed", CreatedAt: now, UpdatedAt: now},
		{UserID: 1, DocumentCode: "AD-2", FileName: "档案-普通", ContentText: "普通档案 九阳神功", Status: "active", OCRStatus: "completed", CreatedAt: now, UpdatedAt: now},
	}
	if err := db.Create(&docs).Error; err != nil {
		t.Fatalf("构造文档失败: %v", err)
	}

	retrieval := NewRetrievalService(db, NewEmbeddingService(db))

	// 普通用户 1：无权限 KB 不命中（但自己的档案文档命中）
	results, err := retrieval.FullTextSearch(1, "葵花宝典", 10, 0)
	if err != nil {
		t.Fatalf("FullTextSearch 失败: %v", err)
	}
	if len(results) != 0 {
		t.Fatalf("普通用户不应命中无权限 KB，实际 %d 条", len(results))
	}
	results, err = retrieval.FullTextSearch(1, "九阳神功", 10, 0)
	if err != nil {
		t.Fatalf("FullTextSearch 失败: %v", err)
	}
	if len(results) == 0 {
		t.Fatal("普通用户应命中自己的档案文档")
	}

	// 管理员 2：kbID=0 全量可见（无权限 KB 也命中）
	results, err = retrieval.FullTextSearch(2, "葵花宝典", 10, 0)
	if err != nil {
		t.Fatalf("FullTextSearch 失败: %v", err)
	}
	if len(results) == 0 {
		t.Fatal("管理员 kbID=0 应全量可见（命中无权限 KB）")
	}
}

// ============================================================
// 14. SavePoint 恢复（实现级）：事务内唯一冲突后 RollbackTo 可继续使用
//     PostgreSQL 中失败事务后续命令会被拒绝（aborted），
//     findOrCreateShadowDocument 依赖 SavePoint 回滚恢复。SQLite 同样支持。
// ============================================================

func TestP9ShadowDocument_SavePointRecovery(t *testing.T) {
	db := setupP9DB(t)

	// 预插入一条影子文档（占用 document_code 唯一键，模拟并发方已提交）
	now := time.Now()
	kbID := uint(1)
	conflict := models.Document{
		UserID:       1,
		DocumentCode: "INGEST-employee-1-1",
		FileName:     "占位",
		SourceType:   "other",
		SourceID:     999,
		Status:       "active",
		OCRStatus:    "completed",
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if err := db.Create(&conflict).Error; err != nil {
		t.Fatalf("预插入冲突记录失败: %v", err)
	}

	// 模拟 findOrCreateShadowDocument 的 SavePoint 模式：
	// 事务内 Create 触发 document_code 唯一冲突 → RollbackTo 恢复 → 事务可继续
	tx := db.Begin()
	if tx.Error != nil {
		t.Fatalf("开启事务失败: %v", tx.Error)
	}
	doc := models.Document{
		UserID:       1,
		DocumentCode: "INGEST-employee-1-1", // 与占位记录冲突
		FileName:     "触发冲突",
		SourceType:   "employee",
		SourceID:     1,
		SourceKBID:   &kbID,
		Status:       "active",
		OCRStatus:    "completed",
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	tx.SavePoint("sp_shadow_create")
	createErr := tx.Create(&doc).Error
	if createErr == nil {
		tx.Rollback()
		t.Fatal("预期 Create 触发唯一冲突，实际成功")
	}
	if !isUniqueViolation(createErr) {
		tx.Rollback()
		t.Fatalf("预期唯一冲突错误，实际: %v", createErr)
	}
	// RollbackTo 恢复后事务可继续执行其他操作
	if rb := tx.RollbackTo("sp_shadow_create"); rb.Error != nil {
		tx.Rollback()
		t.Fatalf("RollbackTo 失败: %v", rb.Error)
	}
	if err := tx.Exec("INSERT INTO documents (user_id, document_code, file_name, status, ocr_status, created_at, updated_at) VALUES (1, 'AFTER-SP-1', '恢复后写入', 'active', 'completed', ?, ?)", now, now).Error; err != nil {
		tx.Rollback()
		t.Fatalf("RollbackTo 后事务应可继续写入，实际失败: %v", err)
	}
	if err := tx.Commit().Error; err != nil {
		t.Fatalf("提交失败: %v", err)
	}

	// 验证：恢复后写入的记录存在
	var after int64
	db.Model(&models.Document{}).Where("document_code = ?", "AFTER-SP-1").Count(&after)
	if after != 1 {
		t.Fatalf("恢复后写入的记录期望 1，实际 %d", after)
	}
}

// ============================================================
// 15. 真正数据库写入失败 → 事务回滚（document_code 唯一约束注入）
// ============================================================

func TestP9ShadowDocument_DBWriteFailureRollback(t *testing.T) {
	db := setupP9DB(t)
	seedP9User(t, db, 1)
	kb := seedP9KB(t, db)
	seedP9Employees(t, db, []string{"洪七公"})

	// 注入约束冲突：预插入一条 document_code 与 ingest 将生成的影子文档相同的记录
	// （source_type 不同，避开影子唯一键，确保走 Create 而非复用路径）
	now := time.Now()
	conflict := models.Document{
		UserID:       1,
		DocumentCode: "INGEST-employee-1-1", // kb.ID=1, rec.ID=1
		FileName:     "占位文档",
		SourceType:   "other",
		SourceID:     999,
		Status:       "active",
		OCRStatus:    "completed",
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if err := db.Create(&conflict).Error; err != nil {
		t.Fatalf("预插入冲突记录失败: %v", err)
	}

	svc := newP9Service(t, db, p9FixedParser())
	result, err := svc.Ingest(context.Background(), 1, IngestRequest{KBID: kb.ID, SourceModule: "employee"})
	if err != nil {
		t.Fatalf("Ingest 不应整体失败: %v", err)
	}
	// 该记录因 DB 写入失败被跳过
	if result.Ingested != 0 || result.Skipped != 1 {
		t.Fatalf("DB 写入失败应计 1 条 Skipped: ingested=%d skipped=%d", result.Ingested, result.Skipped)
	}

	// 事务回滚：无影子文档残留、无 chunks 残留
	if n := countShadowDocs(t, db, "employee"); n != 0 {
		t.Fatalf("回滚后 employee 影子文档应不存在，实际 %d 条", n)
	}
	var chunkCount int64
	db.Model(&models.DocumentChunk{}).Count(&chunkCount)
	if chunkCount != 0 {
		t.Fatalf("回滚后 chunk 应不存在，实际 %d 条", chunkCount)
	}
}
