package service

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"gorm.io/gorm"

	"siapp/internal/models"
)

// ============================================================
// P9 PostgreSQL 集成测试辅助函数（seed / mock / 解析）
// 与 kb_ingest_pg_test.go 同包共享，仅测试使用。
// ============================================================

// seedP9PGUser 创建前缀隔离的测试用户，返回完整记录（含数据库分配的 ID）
func seedP9PGUser(t *testing.T, db *gorm.DB, prefix string, id uint) models.User {
	t.Helper()
	user := models.User{
		Username: fmt.Sprintf("%suser%d", prefix, id),
		Email:    fmt.Sprintf("%suser%d@test.local", prefix, id),
		Password: "hash-placeholder",
	}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("创建用户失败: %v", err)
	}
	return user
}

// seedP9PGKB 创建前缀隔离的 employee 来源知识库（public + employee = 全员可见）
func seedP9PGKB(t *testing.T, db *gorm.DB, prefix string) models.KnowledgeBase {
	t.Helper()
	kb := models.KnowledgeBase{
		Name:         prefix + "知识库",
		SourceModule: "employee",
		Visibility:   "public",
	}
	if err := db.Create(&kb).Error; err != nil {
		t.Fatalf("创建知识库失败: %v", err)
	}
	return kb
}

// seedP9PGEmployee 创建前缀隔离的员工（IDNumber 前缀唯一，满足联合唯一约束）
func seedP9PGEmployee(t *testing.T, db *gorm.DB, userID uint, prefix, name string) models.Employee {
	t.Helper()
	emp := models.Employee{
		UserID:     userID,
		IDNumber:   prefix + "EMP" + name,
		Name:       name,
		Department: "研发部",
	}
	if err := db.Create(&emp).Error; err != nil {
		t.Fatalf("创建员工失败: %v", err)
	}
	return emp
}

// newP9PGService 构造带方言识别的 KBIngestService。
// 必须显式设置 dialect：PG 下 embedChunks 依赖 isPostgres() 走原生向量列写入，
// 否则会误走 SQLite JSON 降级分支（不写 embedding 列）。
func newP9PGService(t *testing.T, db *gorm.DB, parser docParser) *KBIngestService {
	t.Helper()
	return &KBIngestService{
		db:           db,
		parser:       parser,
		embeddingSvc: NewEmbeddingService(db),
		dialect:      newDBDialect(db),
	}
}

// p9PGEmbeddingServer 返回固定向量的 mock embedding API（入库与查询同源）
func p9PGEmbeddingServer(t *testing.T, vec []float64) *httptest.Server {
	t.Helper()
	vecJSON, _ := json.Marshal(vec)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[{"embedding":` + string(vecJSON) + `}]}`))
	}))
	t.Cleanup(srv.Close)
	return srv
}

// seedP9PGEmbeddingConfig 创建前缀隔离的 embedding 模型配置（指向 mock API）
func seedP9PGEmbeddingConfig(t *testing.T, db *gorm.DB, prefix string, userID uint, srvURL string) {
	t.Helper()
	cfg := models.ModelConfig{
		UserID:      &userID,
		ConfigType:  "embedding",
		Provider:    "test",
		ModelName:   prefix + "-embed",
		APIEndpoint: srvURL,
		Enabled:     true,
		IsDefault:   true,
	}
	if err := db.Create(&cfg).Error; err != nil {
		t.Fatalf("创建 embedding 配置失败: %v", err)
	}
}

// parsePGVector 解析 pgvector 文本输出 "[0.1,0.2,...]" 为 []float64
func parsePGVector(text string) ([]float64, error) {
	trimmed := strings.TrimSpace(text)
	if len(trimmed) < 2 || trimmed[0] != '[' || trimmed[len(trimmed)-1] != ']' {
		return nil, fmt.Errorf("非法 pgvector 文本: %q", text)
	}
	parts := strings.Split(trimmed[1:len(trimmed)-1], ",")
	vec := make([]float64, 0, len(parts))
	for _, p := range parts {
		v, err := strconv.ParseFloat(strings.TrimSpace(p), 64)
		if err != nil {
			return nil, err
		}
		vec = append(vec, v)
	}
	return vec, nil
}
