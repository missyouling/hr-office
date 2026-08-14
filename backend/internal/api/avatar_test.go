package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"

	"siapp/internal/models"
	"siapp/internal/service/avatar"
	"siapp/internal/service/storage"
)

// ---------------------------------------------------------------------------
// 测试辅助
// ---------------------------------------------------------------------------

// newAvatarTestHandler 构造仅含头像所需依赖的 Handler（不依赖远程服务）
func newAvatarTestHandler(t *testing.T, tx *gorm.DB) (*Handler, string) {
	t.Helper()
	dir := t.TempDir()
	store, err := storage.NewAvatarStore(dir)
	if err != nil {
		t.Fatalf("创建头像存储失败: %v", err)
	}
	return &Handler{db: tx, avatarStore: store}, dir
}

// newAvatarTestRouter 注册头像路由（无 JWT 中间件，测试用 setAuthContext 模拟登录）
func newAvatarTestRouter(t *testing.T, handler *Handler) chi.Router {
	t.Helper()
	r := chi.NewRouter()
	r.Route("/api/user", func(ur chi.Router) {
		ur.Get("/avatar", handler.getAvatar)
		ur.Post("/avatar", handler.uploadAvatar)
		ur.Delete("/avatar", handler.resetAvatar)
	})
	return r
}

// migrateAvatarTables 迁移头像测试所需表
func migrateAvatarTables(t *testing.T, tx *gorm.DB) {
	t.Helper()
	if err := tx.AutoMigrate(&models.User{}, &models.UserPreference{}, &models.SysFile{}); err != nil {
		t.Fatalf("自动迁移表结构失败: %v", err)
	}
}

// buildAvatarUploadRequest 构造 multipart 上传请求
func buildAvatarUploadRequest(t *testing.T, filename string, data []byte) *http.Request {
	t.Helper()
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	fw, err := mw.CreateFormFile("file", filename)
	if err != nil {
		t.Fatalf("创建 multipart 字段失败: %v", err)
	}
	if _, err := fw.Write(data); err != nil {
		t.Fatalf("写入 multipart 数据失败: %v", err)
	}
	if err := mw.Close(); err != nil {
		t.Fatalf("关闭 multipart writer 失败: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/user/avatar", &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	return req
}

// 测试用图像数据：魔数正确且长度 ≥512 字节，确保 http.DetectContentType 可识别
var (
	testPNGData  = append([]byte("\x89PNG\r\n\x1a\n"), bytes.Repeat([]byte("A"), 600)...)
	testJPEGData = append([]byte("\xff\xd8\xff\xe0"), bytes.Repeat([]byte("B"), 600)...)
	testWebPData = append([]byte("RIFF\x00\x00\x00\x00WEBPVP8 "), bytes.Repeat([]byte("C"), 600)...)
	testTXTData  = []byte("hello world, this is not an image")
)

// countAvatarFiles 统计头像目录中的文件数
func countAvatarFiles(t *testing.T, dir string) int {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("读取头像目录失败: %v", err)
	}
	return len(entries)
}

// ---------------------------------------------------------------------------
// 纯函数测试：种子稳定性 / 首字母边界 / SVG 稳定性
// ---------------------------------------------------------------------------

func TestGenerateSeedStable(t *testing.T) {
	seed1 := avatar.GenerateSeed(42, "alice")
	seed2 := avatar.GenerateSeed(42, "alice")
	if seed1 != seed2 {
		t.Fatalf("同一用户种子应稳定: %q != %q", seed1, seed2)
	}
	if seed1 == "" {
		t.Fatal("种子不应为空")
	}
	// 不同用户种子应不同
	if avatar.GenerateSeed(43, "alice") == seed1 {
		t.Fatal("不同用户 ID 种子不应相同")
	}
	if avatar.GenerateSeed(42, "bob") == seed1 {
		t.Fatal("不同用户名种子不应相同")
	}
}

func TestInitialBoundaries(t *testing.T) {
	cases := []struct {
		name string
		want string
	}{
		{"", "?"},        // 空串兜底
		{"   ", "?"},     // 纯空白兜底
		{"张三", "张"},      // 中文首字
		{"Alice", "A"},   // 英文首字母
		{"😀happy", "😀"},  // emoji 首字符（UTF-8 安全）
		{"  Bob  ", "B"}, // 首尾空白裁剪
	}
	for _, c := range cases {
		if got := avatar.Initial(c.name); got != c.want {
			t.Errorf("Initial(%q) = %q, want %q", c.name, got, c.want)
		}
	}
}

func TestDefaultSVGStable(t *testing.T) {
	svg1 := avatar.DefaultSVG("seed-a", "张")
	svg2 := avatar.DefaultSVG("seed-a", "张")
	if svg1 != svg2 {
		t.Fatal("同一 seed+initial 应输出完全一致的 SVG")
	}
	if !strings.Contains(svg1, "张") {
		t.Fatal("SVG 应包含首字母")
	}
	if !strings.Contains(svg1, "<svg") || !strings.Contains(svg1, "</svg>") {
		t.Fatal("SVG 结构不完整")
	}
	// 不同 seed 配色应不同
	svg3 := avatar.DefaultSVG("seed-b", "张")
	if svg1 == svg3 {
		t.Fatal("不同 seed 应产生不同配色")
	}
	// 特殊字符应被转义，防止 SVG 注入
	svg4 := avatar.DefaultSVG("seed-c", "<&")
	if strings.Contains(svg4, "<&") {
		t.Fatal("特殊字符应被 HTML 转义")
	}
}

// ---------------------------------------------------------------------------
// 未登录 401
// ---------------------------------------------------------------------------

func TestAvatarUnauthorized(t *testing.T) {
	db := setupTestDB(t)
	tx := newTestTransaction(t, db)
	migrateAvatarTables(t, tx)
	handler, _ := newAvatarTestHandler(t, tx)
	router := newAvatarTestRouter(t, handler)

	// GET 未登录
	req := httptest.NewRequest(http.MethodGet, "/api/user/avatar", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("GET 未登录应返回 401, got %d", rec.Code)
	}

	// POST 未登录
	req = buildAvatarUploadRequest(t, "a.png", testPNGData)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("POST 未登录应返回 401, got %d", rec.Code)
	}

	// DELETE 未登录
	req = httptest.NewRequest(http.MethodDelete, "/api/user/avatar", nil)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("DELETE 未登录应返回 401, got %d", rec.Code)
	}
}

// ---------------------------------------------------------------------------
// 默认头像 GET
// ---------------------------------------------------------------------------

func TestGetAvatarDefault(t *testing.T) {
	db := setupTestDB(t)
	tx := newTestTransaction(t, db)
	migrateAvatarTables(t, tx)
	user := createTestUser(t, tx, "alice", "张三")
	handler, _ := newAvatarTestHandler(t, tx)
	router := newAvatarTestRouter(t, handler)

	req := httptest.NewRequest(http.MethodGet, "/api/user/avatar", nil)
	req = setAuthContext(req, user.ID)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("GET 默认头像应返回 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	if ct := rec.Header().Get("Content-Type"); ct != "image/svg+xml" {
		t.Fatalf("Content-Type 应为 image/svg+xml, got %q", ct)
	}
	if !strings.Contains(rec.Body.String(), "张") {
		t.Fatal("默认 SVG 应包含用户首字母")
	}
	// 种子应已持久化到 UserPreference
	var pref models.UserPreference
	if err := tx.Where("user_id = ? AND pref_key = ?", user.ID, models.AvatarPrefKey).First(&pref).Error; err != nil {
		t.Fatalf("种子应持久化到 UserPreference: %v", err)
	}
	var ap models.AvatarPreference
	if err := json.Unmarshal(pref.Value, &ap); err != nil {
		t.Fatalf("解析偏好失败: %v", err)
	}
	if ap.Seed == "" || ap.Type != models.AvatarTypeDefault {
		t.Fatalf("偏好应含种子且为 default: %+v", ap)
	}
}

// ---------------------------------------------------------------------------
// 上传：类型 / 大小拒绝
// ---------------------------------------------------------------------------

func TestUploadAvatarRejectsBadType(t *testing.T) {
	db := setupTestDB(t)
	tx := newTestTransaction(t, db)
	migrateAvatarTables(t, tx)
	user := createTestUser(t, tx, "bob", "Bob")
	handler, dir := newAvatarTestHandler(t, tx)
	router := newAvatarTestRouter(t, handler)

	// 伪装成 .png 的文本文件：服务端必须按真实 MIME 拒绝
	req := buildAvatarUploadRequest(t, "fake.png", testTXTData)
	req = setAuthContext(req, user.ID)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("非图片应返回 400, got %d body=%s", rec.Code, rec.Body.String())
	}
	if countAvatarFiles(t, dir) != 0 {
		t.Fatal("被拒绝的上传不应留下文件")
	}
}

func TestUploadAvatarRejectsTooLarge(t *testing.T) {
	db := setupTestDB(t)
	tx := newTestTransaction(t, db)
	migrateAvatarTables(t, tx)
	user := createTestUser(t, tx, "carol", "Carol")
	handler, dir := newAvatarTestHandler(t, tx)
	router := newAvatarTestRouter(t, handler)

	// 2 MiB + 1 字节，超过限制
	bigData := append([]byte("\x89PNG\r\n\x1a\n"), bytes.Repeat([]byte("A"), (2<<20)+1)...)
	req := buildAvatarUploadRequest(t, "big.png", bigData)
	req = setAuthContext(req, user.ID)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("超大文件应返回 413, got %d body=%s", rec.Code, rec.Body.String())
	}
	if countAvatarFiles(t, dir) != 0 {
		t.Fatal("被拒绝的上传不应留下文件")
	}
}

// ---------------------------------------------------------------------------
// 上传成功 + 替换清理 + 恢复默认
// ---------------------------------------------------------------------------

func TestUploadAvatarSuccess(t *testing.T) {
	db := setupTestDB(t)
	tx := newTestTransaction(t, db)
	migrateAvatarTables(t, tx)
	user := createTestUser(t, tx, "dave", "Dave")
	handler, dir := newAvatarTestHandler(t, tx)
	router := newAvatarTestRouter(t, handler)

	req := buildAvatarUploadRequest(t, "avatar.png", testPNGData)
	req = setAuthContext(req, user.ID)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("上传应返回 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	var meta avatarMetadataResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &meta); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}
	if meta.Type != models.AvatarTypeCustom || meta.CustomFileID == nil {
		t.Fatalf("响应应为 custom 且含 file id: %+v", meta)
	}
	if meta.ContentType != "image/png" {
		t.Fatalf("ContentType 应为 image/png, got %q", meta.ContentType)
	}
	if countAvatarFiles(t, dir) != 1 {
		t.Fatalf("应恰好 1 个头像文件, got %d", countAvatarFiles(t, dir))
	}

	// SysFile 元数据应记录 CreatedBy
	var sf models.SysFile
	if err := tx.First(&sf, *meta.CustomFileID).Error; err != nil {
		t.Fatalf("SysFile 应存在: %v", err)
	}
	if sf.CreatedBy == nil || *sf.CreatedBy != user.ID {
		t.Fatalf("SysFile.CreatedBy 应为 %d, got %v", user.ID, sf.CreatedBy)
	}

	// GET 应返回自定义头像二进制
	req = httptest.NewRequest(http.MethodGet, "/api/user/avatar", nil)
	req = setAuthContext(req, user.ID)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET 自定义头像应返回 200, got %d", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "image/png" {
		t.Fatalf("Content-Type 应为 image/png, got %q", ct)
	}
	if !bytes.Equal(rec.Body.Bytes(), testPNGData) {
		t.Fatal("GET 应返回上传的原始二进制")
	}
}

func TestUploadAvatarReplacesOld(t *testing.T) {
	db := setupTestDB(t)
	tx := newTestTransaction(t, db)
	migrateAvatarTables(t, tx)
	user := createTestUser(t, tx, "erin", "Erin")
	handler, dir := newAvatarTestHandler(t, tx)
	router := newAvatarTestRouter(t, handler)

	// 第一次上传 PNG
	req := buildAvatarUploadRequest(t, "first.png", testPNGData)
	req = setAuthContext(req, user.ID)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("第一次上传失败: %d %s", rec.Code, rec.Body.String())
	}
	var firstMeta avatarMetadataResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &firstMeta)

	// 第二次上传 WebP（替换）
	req = buildAvatarUploadRequest(t, "second.webp", testWebPData)
	req = setAuthContext(req, user.ID)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("第二次上传失败: %d %s", rec.Code, rec.Body.String())
	}
	var secondMeta avatarMetadataResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &secondMeta)
	if secondMeta.CustomFileID == nil || *secondMeta.CustomFileID == *firstMeta.CustomFileID {
		t.Fatal("替换后应指向新的 SysFile")
	}

	// 旧 SysFile 应被软删除
	var oldSF models.SysFile
	err := tx.Unscoped().First(&oldSF, *firstMeta.CustomFileID).Error
	if err != nil {
		t.Fatalf("旧 SysFile 应存在（软删除）: %v", err)
	}
	if !oldSF.DeletedAt.Valid {
		t.Fatal("旧 SysFile 应被软删除")
	}
	// 目录中应只剩 1 个文件（旧文件已清理）
	if n := countAvatarFiles(t, dir); n != 1 {
		t.Fatalf("替换后应只剩 1 个文件, got %d", n)
	}

	// GET 应返回新的 WebP
	req = httptest.NewRequest(http.MethodGet, "/api/user/avatar", nil)
	req = setAuthContext(req, user.ID)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if ct := rec.Header().Get("Content-Type"); ct != "image/webp" {
		t.Fatalf("替换后 Content-Type 应为 image/webp, got %q", ct)
	}
}

func TestResetAvatar(t *testing.T) {
	db := setupTestDB(t)
	tx := newTestTransaction(t, db)
	migrateAvatarTables(t, tx)
	user := createTestUser(t, tx, "frank", "Frank")
	handler, dir := newAvatarTestHandler(t, tx)
	router := newAvatarTestRouter(t, handler)

	// 先上传
	req := buildAvatarUploadRequest(t, "a.png", testPNGData)
	req = setAuthContext(req, user.ID)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	var meta avatarMetadataResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &meta)

	// 恢复默认
	req = httptest.NewRequest(http.MethodDelete, "/api/user/avatar", nil)
	req = setAuthContext(req, user.ID)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("恢复默认应返回 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	var resetMeta avatarMetadataResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resetMeta); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}
	if resetMeta.Type != models.AvatarTypeDefault || resetMeta.CustomFileID != nil {
		t.Fatalf("恢复后应为 default 且无 file id: %+v", resetMeta)
	}

	// 旧文件与元数据应被清理
	if n := countAvatarFiles(t, dir); n != 0 {
		t.Fatalf("恢复默认后应无头像文件, got %d", n)
	}
	var oldSF models.SysFile
	if err := tx.Unscoped().First(&oldSF, *meta.CustomFileID).Error; err != nil {
		t.Fatalf("旧 SysFile 应存在（软删除）: %v", err)
	}
	if !oldSF.DeletedAt.Valid {
		t.Fatal("旧 SysFile 应被软删除")
	}

	// GET 应返回默认 SVG
	req = httptest.NewRequest(http.MethodGet, "/api/user/avatar", nil)
	req = setAuthContext(req, user.ID)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if ct := rec.Header().Get("Content-Type"); ct != "image/svg+xml" {
		t.Fatalf("恢复后 Content-Type 应为 image/svg+xml, got %q", ct)
	}
	if !strings.Contains(rec.Body.String(), "F") {
		t.Fatal("默认 SVG 应包含首字母")
	}
}

// ---------------------------------------------------------------------------
// 用户隔离
// ---------------------------------------------------------------------------

func TestAvatarUserIsolation(t *testing.T) {
	db := setupTestDB(t)
	tx := newTestTransaction(t, db)
	migrateAvatarTables(t, tx)
	userA := createTestUser(t, tx, "alice2", "Alice")
	userB := createTestUser(t, tx, "bob2", "Bob")
	handler, _ := newAvatarTestHandler(t, tx)
	router := newAvatarTestRouter(t, handler)

	// A 上传自定义头像
	req := buildAvatarUploadRequest(t, "a.png", testPNGData)
	req = setAuthContext(req, userA.ID)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("A 上传失败: %d %s", rec.Code, rec.Body.String())
	}

	// B 的 GET 应返回 B 自己的默认 SVG，绝不能拿到 A 的 PNG
	req = httptest.NewRequest(http.MethodGet, "/api/user/avatar", nil)
	req = setAuthContext(req, userB.ID)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("B GET 应返回 200, got %d", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "image/svg+xml" {
		t.Fatalf("B 应得到默认 SVG, got %q", ct)
	}
	if bytes.Contains(rec.Body.Bytes(), testPNGData) {
		t.Fatal("B 不应拿到 A 的自定义头像内容")
	}
	if !strings.Contains(rec.Body.String(), "B") {
		t.Fatal("B 的默认 SVG 应包含 B 的首字母")
	}

	// A 的 GET 仍应返回自己的 PNG
	req = httptest.NewRequest(http.MethodGet, "/api/user/avatar", nil)
	req = setAuthContext(req, userA.ID)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if ct := rec.Header().Get("Content-Type"); ct != "image/png" {
		t.Fatalf("A 应仍得到自己的 PNG, got %q", ct)
	}
}

// ---------------------------------------------------------------------------
// 二进制响应头
// ---------------------------------------------------------------------------

func TestAvatarBinaryResponseHeaders(t *testing.T) {
	db := setupTestDB(t)
	tx := newTestTransaction(t, db)
	migrateAvatarTables(t, tx)
	user := createTestUser(t, tx, "grace", "Grace")
	handler, _ := newAvatarTestHandler(t, tx)
	router := newAvatarTestRouter(t, handler)

	// 上传 JPEG
	req := buildAvatarUploadRequest(t, "g.jpg", testJPEGData)
	req = setAuthContext(req, user.ID)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("上传失败: %d %s", rec.Code, rec.Body.String())
	}

	// GET 校验响应头
	req = httptest.NewRequest(http.MethodGet, "/api/user/avatar", nil)
	req = setAuthContext(req, user.ID)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if ct := rec.Header().Get("Content-Type"); ct != "image/jpeg" {
		t.Fatalf("Content-Type 应为 image/jpeg, got %q", ct)
	}
	if cc := rec.Header().Get("Cache-Control"); cc != "private, no-store" {
		t.Fatalf("Cache-Control 应为 private, no-store, got %q", cc)
	}
	if cl := rec.Header().Get("Content-Length"); cl != fmt.Sprintf("%d", len(testJPEGData)) {
		t.Fatalf("Content-Length 应为 %d, got %q", len(testJPEGData), cl)
	}
	if !bytes.Equal(rec.Body.Bytes(), testJPEGData) {
		t.Fatal("响应体应与上传内容一致")
	}
}

// ---------------------------------------------------------------------------
// 元数据响应不泄露绝对路径
// ---------------------------------------------------------------------------

func TestAvatarMetadataNoAbsolutePath(t *testing.T) {
	db := setupTestDB(t)
	tx := newTestTransaction(t, db)
	migrateAvatarTables(t, tx)
	user := createTestUser(t, tx, "henry", "Henry")
	handler, dir := newAvatarTestHandler(t, tx)
	router := newAvatarTestRouter(t, handler)

	req := buildAvatarUploadRequest(t, "h.png", testPNGData)
	req = setAuthContext(req, user.ID)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	body := rec.Body.String()
	if strings.Contains(body, dir) || strings.Contains(body, filepath.Base(dir)) {
		t.Fatalf("响应不应泄露宿主机路径: %s", body)
	}
	if strings.Contains(body, "avatars") && strings.Contains(body, "/") {
		t.Fatalf("响应不应包含存储路径: %s", body)
	}
}

// ---------------------------------------------------------------------------
// 偏好重复 / 软删除边界
// ---------------------------------------------------------------------------

// TestAvatarConcurrentFirstGetNoDuplicatePreference 并发首次 GET 不应创建重复偏好记录。
// 使用独立数据库（非事务）以便并发请求共享同一连接池。
func TestAvatarConcurrentFirstGetNoDuplicatePreference(t *testing.T) {
	db := setupTestDB(t)
	migrateAvatarTables(t, db)
	user := createTestUser(t, db, "concurrent", "并发用户")
	store, err := storage.NewAvatarStore(t.TempDir())
	if err != nil {
		t.Fatalf("创建头像存储失败: %v", err)
	}
	handler := &Handler{db: db, avatarStore: store}
	router := newAvatarTestRouter(t, handler)

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			req := httptest.NewRequest(http.MethodGet, "/api/user/avatar", nil)
			req = setAuthContext(req, user.ID)
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)
			if rec.Code != http.StatusOK {
				t.Errorf("并发 GET 应返回 200, got %d", rec.Code)
			}
		}()
	}
	wg.Wait()

	var count int64
	if err := db.Model(&models.UserPreference{}).
		Where("user_id = ? AND pref_key = ?", user.ID, models.AvatarPrefKey).
		Count(&count).Error; err != nil {
		t.Fatalf("统计偏好失败: %v", err)
	}
	if count != 1 {
		t.Fatalf("并发首次 GET 后应恰好 1 条偏好记录, got %d", count)
	}
}

// TestAvatarSoftDeletedFileFallsBackToDefault 偏好指向已软删除的 SysFile 时应回退默认。
func TestAvatarSoftDeletedFileFallsBackToDefault(t *testing.T) {
	db := setupTestDB(t)
	tx := newTestTransaction(t, db)
	migrateAvatarTables(t, tx)
	user := createTestUser(t, tx, "softdel", "软删用户")
	handler, _ := newAvatarTestHandler(t, tx)
	router := newAvatarTestRouter(t, handler)

	// 上传自定义头像
	req := buildAvatarUploadRequest(t, "a.png", testPNGData)
	req = setAuthContext(req, user.ID)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("上传失败: %d %s", rec.Code, rec.Body.String())
	}
	var meta avatarMetadataResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &meta)

	// 模拟 SysFile 元数据被软删除（如其他模块误删）
	if err := tx.Delete(&models.SysFile{}, *meta.CustomFileID).Error; err != nil {
		t.Fatalf("软删除 SysFile 失败: %v", err)
	}

	// GET 应回退默认 SVG，且偏好被重置为 default
	req = httptest.NewRequest(http.MethodGet, "/api/user/avatar", nil)
	req = setAuthContext(req, user.ID)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET 应返回 200, got %d", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "image/svg+xml" {
		t.Fatalf("软删除后应回退默认 SVG, got %q", ct)
	}
	var pref models.UserPreference
	if err := tx.Where("user_id = ? AND pref_key = ?", user.ID, models.AvatarPrefKey).First(&pref).Error; err != nil {
		t.Fatalf("偏好应存在: %v", err)
	}
	var ap models.AvatarPreference
	if err := json.Unmarshal(pref.Value, &ap); err != nil {
		t.Fatalf("解析偏好失败: %v", err)
	}
	if ap.Type != models.AvatarTypeDefault || ap.CustomFileID != nil {
		t.Fatalf("偏好应重置为 default 且无 file id: %+v", ap)
	}
}

// TestAvatarMissingFileFallsBackToDefault 物理文件缺失时应回退默认并清理失效引用。
func TestAvatarMissingFileFallsBackToDefault(t *testing.T) {
	db := setupTestDB(t)
	tx := newTestTransaction(t, db)
	migrateAvatarTables(t, tx)
	user := createTestUser(t, tx, "missing", "缺失用户")
	handler, dir := newAvatarTestHandler(t, tx)
	router := newAvatarTestRouter(t, handler)

	// 上传自定义头像
	req := buildAvatarUploadRequest(t, "a.png", testPNGData)
	req = setAuthContext(req, user.ID)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("上传失败: %d %s", rec.Code, rec.Body.String())
	}
	var meta avatarMetadataResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &meta)

	// 删除物理文件（模拟磁盘文件丢失）
	entries, err := os.ReadDir(dir)
	if err != nil || len(entries) != 1 {
		t.Fatalf("头像目录应恰好 1 个文件: %v", err)
	}
	if err := os.Remove(filepath.Join(dir, entries[0].Name())); err != nil {
		t.Fatalf("删除物理文件失败: %v", err)
	}

	// GET 应回退默认 SVG，偏好重置为 default，失效 SysFile 被软删除
	req = httptest.NewRequest(http.MethodGet, "/api/user/avatar", nil)
	req = setAuthContext(req, user.ID)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET 应返回 200, got %d", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "image/svg+xml" {
		t.Fatalf("文件缺失应回退默认 SVG, got %q", ct)
	}
	var pref models.UserPreference
	if err := tx.Where("user_id = ? AND pref_key = ?", user.ID, models.AvatarPrefKey).First(&pref).Error; err != nil {
		t.Fatalf("偏好应存在: %v", err)
	}
	var ap models.AvatarPreference
	if err := json.Unmarshal(pref.Value, &ap); err != nil {
		t.Fatalf("解析偏好失败: %v", err)
	}
	if ap.Type != models.AvatarTypeDefault || ap.CustomFileID != nil {
		t.Fatalf("偏好应重置为 default 且无 file id: %+v", ap)
	}
	var sf models.SysFile
	if err := tx.Unscoped().First(&sf, *meta.CustomFileID).Error; err != nil {
		t.Fatalf("SysFile 应存在（软删除）: %v", err)
	}
	if !sf.DeletedAt.Valid {
		t.Fatal("失效 SysFile 应被软删除")
	}
}
