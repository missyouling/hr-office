package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"siapp/internal/models"
	"siapp/internal/service"
)

// ---------------------------------------------------------------------------
// 全局搜索（/api/search/global）测试
// 覆盖：空查询 400、未登录 401、用户隔离、limit 边界/非法值、DB 错误传播
// ---------------------------------------------------------------------------

// newGlobalSearchTestRouter 挂载全局搜索路由（无鉴权中间件，测试用 setAuthContext 模拟）
func newGlobalSearchTestRouter(handler *Handler) chi.Router {
	r := chi.NewRouter()
	r.Get("/api/search/global", handler.globalSearch)
	return r
}

// migrateGlobalSearchTables 迁移全局搜索所需表
func migrateGlobalSearchTables(t *testing.T, tx *gorm.DB) {
	t.Helper()
	err := tx.AutoMigrate(
		&models.User{},
		&models.Document{},
		&models.Employee{},
		&models.DormSite{},
		&models.DormBuilding{},
		&models.DormRoom{},
	)
	if err != nil {
		t.Fatalf("自动迁移全局搜索表失败: %v", err)
	}
}

// seedGlobalSearchData 为指定用户创建文档/员工/宿舍各一条（tag 用于隔离关键字）
func seedGlobalSearchData(t *testing.T, tx *gorm.DB, user models.User, tag string) {
	t.Helper()

	doc := models.Document{
		UserID:       user.ID,
		DocumentCode: fmt.Sprintf("%s-DOC-001", tag),
		FileName:     tag + "的劳动合同",
		ContentText:  "这是" + tag + "专属档案内容",
	}
	require.NoError(t, tx.Create(&doc).Error, "创建测试文档失败")

	emp := models.Employee{
		UserID:     user.ID,
		Name:       tag + "员工",
		Department: tag + "部门",
		IDNumber:   fmt.Sprintf("11000019900101%s", tag),
	}
	require.NoError(t, tx.Create(&emp).Error, "创建测试员工失败")

	// 宿舍房间依赖宿舍园区/楼栋（DormRoom.BuildingID 外键）
	site := models.DormSite{Name: tag + "园区"}
	require.NoError(t, tx.Create(&site).Error, "创建测试园区失败")
	building := models.DormBuilding{SiteID: site.ID, Name: tag + "楼"}
	require.NoError(t, tx.Create(&building).Error, "创建测试楼栋失败")
	room := models.DormRoom{
		UserID:     &user.ID,
		BuildingID: building.ID,
		RoomNumber: tag + "101",
		RoomType:   "单人间",
	}
	require.NoError(t, tx.Create(&room).Error, "创建测试宿舍失败")
}

// doGlobalSearchRequest 执行全局搜索请求并返回响应
func doGlobalSearchRequest(t *testing.T, router chi.Router, url string, userID uint, authenticated bool) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, url, nil)
	if authenticated {
		req = setAuthContext(req, userID)
	}
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

// ---------------------------------------------------------------------------
// 接口测试
// ---------------------------------------------------------------------------

// TestGlobalSearchAPI_EmptyQuery 空 q 返回 400
func TestGlobalSearchAPI_EmptyQuery(t *testing.T) {
	db := setupTestDB(t)
	tx := newTestTransaction(t, db)
	migrateGlobalSearchTables(t, tx)

	user := createTestUser(t, tx, "gsempty", "空查询用户")
	handler := NewHandler(tx)
	router := newGlobalSearchTestRouter(handler)

	// 缺省 q
	w := doGlobalSearchRequest(t, router, "/api/search/global", user.ID, true)
	assert.Equal(t, http.StatusBadRequest, w.Code, "缺省 q 应返回 400: %s", w.Body.String())

	// 纯空白 q
	w2 := doGlobalSearchRequest(t, router, "/api/search/global?q=%20%20", user.ID, true)
	assert.Equal(t, http.StatusBadRequest, w2.Code, "空白 q 应返回 400: %s", w2.Body.String())
}

// TestGlobalSearchAPI_Unauthorized 未登录返回 401
func TestGlobalSearchAPI_Unauthorized(t *testing.T) {
	db := setupTestDB(t)
	tx := newTestTransaction(t, db)
	migrateGlobalSearchTables(t, tx)

	handler := NewHandler(tx)
	router := newGlobalSearchTestRouter(handler)

	w := doGlobalSearchRequest(t, router, "/api/search/global?q=张三", 0, false)
	assert.Equal(t, http.StatusUnauthorized, w.Code, "未登录应返回 401: %s", w.Body.String())
}

// TestGlobalSearchAPI_UserIsolation A 用户搜不到 B 用户的档案/员工/宿舍
func TestGlobalSearchAPI_UserIsolation(t *testing.T) {
	db := setupTestDB(t)
	tx := newTestTransaction(t, db)
	migrateGlobalSearchTables(t, tx)

	userA := createTestUser(t, tx, "gsa", "甲方用户")
	userB := createTestUser(t, tx, "gsb", "乙方用户")
	seedGlobalSearchData(t, tx, userA, "甲方")
	seedGlobalSearchData(t, tx, userB, "乙方")

	handler := NewHandler(tx)
	router := newGlobalSearchTestRouter(handler)

	w := doGlobalSearchRequest(t, router, "/api/search/global?q=%E7%94%B2%E6%96%B9", userA.ID, true)
	require.Equal(t, http.StatusOK, w.Code, "搜索应成功: %s", w.Body.String())

	var resp struct {
		Results []service.GlobalSearchResult `json:"results"`
		Count   int                          `json:"count"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))

	// A 用户命中自己的三个模块结果
	modules := map[string]bool{}
	for _, r := range resp.Results {
		modules[r.Module] = true
		assert.NotEqual(t, "乙方", r.Title, "A 用户不应看到 B 用户的结果")
	}
	assert.Equal(t, 3, len(resp.Results), "A 用户应命中三个模块，实际 %d: %s", resp.Count, w.Body.String())
	assert.True(t, modules["archives"], "应包含档案模块结果")
	assert.True(t, modules["employee"], "应包含员工模块结果")
	assert.True(t, modules["dormitory"], "应包含宿舍模块结果")

	// B 用户反向搜索同样隔离
	wB := doGlobalSearchRequest(t, router, "/api/search/global?q=%E4%B9%99%E6%96%B9", userB.ID, true)
	require.Equal(t, http.StatusOK, wB.Code)
	var respB struct {
		Results []service.GlobalSearchResult `json:"results"`
	}
	require.NoError(t, json.Unmarshal(wB.Body.Bytes(), &respB))
	assert.Equal(t, 3, len(respB.Results), "B 用户应只命中自己的结果")
}

// TestGlobalSearchAPI_LimitParam limit=1 截断结果，非法 limit 回退默认值
func TestGlobalSearchAPI_LimitParam(t *testing.T) {
	db := setupTestDB(t)
	tx := newTestTransaction(t, db)
	migrateGlobalSearchTables(t, tx)

	user := createTestUser(t, tx, "gslimit", "limit测试用户")
	// 造 3 条文档便于验证截断
	for i := 1; i <= 3; i++ {
		doc := models.Document{
			UserID:       user.ID,
			DocumentCode: fmt.Sprintf("LMT-%d", i),
			FileName:     fmt.Sprintf("limit文档%d", i),
			ContentText:  "limit 截断测试内容",
		}
		require.NoError(t, tx.Create(&doc).Error)
	}

	handler := NewHandler(tx)
	router := newGlobalSearchTestRouter(handler)

	// limit=1 → 结果不超过 1 条
	w := doGlobalSearchRequest(t, router, "/api/search/global?q=limit&limit=1", user.ID, true)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var resp struct {
		Results []service.GlobalSearchResult `json:"results"`
		Count   int                          `json:"count"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.LessOrEqual(t, resp.Count, 1, "limit=1 时结果应不超过 1 条: %s", w.Body.String())

	// 非法 limit（非数字）→ 不报错，回退默认值
	w2 := doGlobalSearchRequest(t, router, "/api/search/global?q=limit&limit=abc", user.ID, true)
	assert.Equal(t, http.StatusOK, w2.Code, "非法 limit 不应报错: %s", w2.Body.String())

	// limit=0 / 负数 → 回退默认值不报错
	w3 := doGlobalSearchRequest(t, router, "/api/search/global?q=limit&limit=-5", user.ID, true)
	assert.Equal(t, http.StatusOK, w3.Code, "负 limit 不应报错: %s", w3.Body.String())
}

// ---------------------------------------------------------------------------
// 服务层测试
// ---------------------------------------------------------------------------

// TestGlobalSearchService_EmptyQuery 空查询返回错误
func TestGlobalSearchService_EmptyQuery(t *testing.T) {
	db := setupTestDB(t)
	tx := newTestTransaction(t, db)
	migrateGlobalSearchTables(t, tx)

	retSvc := service.NewRetrievalService(tx, service.NewEmbeddingService(tx))

	_, err := retSvc.GlobalSearch(1, "   ", 10)
	require.Error(t, err, "空查询应返回错误")
}

// TestGlobalSearchService_UserIsolation 服务层用户隔离
func TestGlobalSearchService_UserIsolation(t *testing.T) {
	db := setupTestDB(t)
	tx := newTestTransaction(t, db)
	migrateGlobalSearchTables(t, tx)

	userA := createTestUser(t, tx, "gsvcA", "服务层甲方")
	userB := createTestUser(t, tx, "gsvcB", "服务层乙方")
	seedGlobalSearchData(t, tx, userA, "服务层甲方")
	seedGlobalSearchData(t, tx, userB, "服务层乙方")

	retSvc := service.NewRetrievalService(tx, service.NewEmbeddingService(tx))

	results, err := retSvc.GlobalSearch(userA.ID, "服务层甲方", 20)
	require.NoError(t, err)
	assert.Len(t, results, 3, "应命中三个模块")
	for _, r := range results {
		assert.NotContains(t, r.Title, "服务层乙方", "A 用户不应看到 B 用户的数据")
	}
}

// TestGlobalSearchService_LimitUpperBound limit 超过上限时截断到 MaxGlobalSearchLimit
func TestGlobalSearchService_LimitUpperBound(t *testing.T) {
	db := setupTestDB(t)
	tx := newTestTransaction(t, db)
	migrateGlobalSearchTables(t, tx)

	user := createTestUser(t, tx, "gsupper", "上限测试用户")
	// 造 55 条文档（> 上限 50），验证单模块截断
	for i := 1; i <= 55; i++ {
		doc := models.Document{
			UserID:       user.ID,
			DocumentCode: fmt.Sprintf("UPPER-%d", i),
			FileName:     fmt.Sprintf("上限文档%d", i),
			ContentText:  "上限截断测试内容",
		}
		require.NoError(t, tx.Create(&doc).Error)
	}

	retSvc := service.NewRetrievalService(tx, service.NewEmbeddingService(tx))

	results, err := retSvc.GlobalSearch(user.ID, "上限截断", 9999)
	require.NoError(t, err)
	assert.LessOrEqual(t, len(results), service.MaxGlobalSearchLimit,
		"limit 应被截断到 %d，实际 %d", service.MaxGlobalSearchLimit, len(results))
}

// TestGlobalSearchService_DBError 数据库查询异常必须向上传播，不能静默吞掉
func TestGlobalSearchService_DBError(t *testing.T) {
	db := setupTestDB(t)
	tx := newTestTransaction(t, db)
	migrateGlobalSearchTables(t, tx)

	user := createTestUser(t, tx, "gserr", "错误传播用户")
	seedGlobalSearchData(t, tx, user, "错误传播")

	// 删除 documents 表，模拟数据库查询异常
	require.NoError(t, tx.Migrator().DropTable(&models.Document{}))

	retSvc := service.NewRetrievalService(tx, service.NewEmbeddingService(tx))

	_, err := retSvc.GlobalSearch(user.ID, "错误传播", 20)
	require.Error(t, err, "数据库异常应返回错误而非静默忽略")
	assert.True(t, strings.Contains(err.Error(), "global search"),
		"错误信息应包含模块上下文: %v", err)
}
