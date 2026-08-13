package api

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"gorm.io/datatypes"
	"gorm.io/gorm"

	"siapp/internal/models"
	"siapp/internal/service/storage"
)

func newInvoiceAttachmentRouter(handler *Handler) chi.Router {
	r := chi.NewRouter()
	r.Post("/invoices/upload", handler.uploadInvoicePDFs)
	r.Get("/invoices/{id}/attachment", handler.previewInvoiceAttachment)
	r.Get("/invoices/{id}/attachment/download", handler.downloadInvoiceAttachment)
	return r
}

func setupInvoiceAttachmentTest(t *testing.T) (*Handler, chi.Router, *gorm.DB) {
	t.Helper()
	db := setupTestDB(t)
	migrateInvoiceTables(t, db)
	dir := t.TempDir()
	config := models.StorageConfig{Name: "测试本地存储", Type: "local", Enabled: true, IsDefault: true, Config: datatypes.JSON([]byte(`{"root_path":"` + dir + `"}`))}
	if err := db.Create(&config).Error; err != nil {
		t.Fatalf("创建存储配置失败: %v", err)
	}
	previous := storage.GlobalManager
	storage.GlobalManager = storage.NewStorageManager(db, storage.DefaultRegistry)
	t.Cleanup(func() { storage.GlobalManager = previous })
	handler := NewHandler(db)
	return handler, newInvoiceAttachmentRouter(handler), db
}

type invoiceTestFile struct {
	name    string
	content []byte
}

func invoiceMultipartRequest(t *testing.T, files []invoiceTestFile) *http.Request {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	for _, file := range files {
		part, err := writer.CreateFormFile("files", file.name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := part.Write(file.content); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/invoices/upload", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	return req
}

func TestInvoiceUpload_ValidatesBatchAndCreatesTasks(t *testing.T) {
	_, router, db := setupInvoiceAttachmentTest(t)
	user := createInvoiceTestUser(t, db, "upload-user", "上传用户")
	pdf := []byte("%PDF-1.7\n测试")
	req := setAuthContext(invoiceMultipartRequest(t, []invoiceTestFile{{"ok.pdf", pdf}, {"bad.pdf", []byte("不是 PDF")}}), user.ID)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("上传失败: %d %s", w.Code, w.Body.String())
	}
	if strings.Contains(w.Body.String(), "path") {
		t.Fatalf("响应泄露内部路径: %s", w.Body.String())
	}
	var response struct {
		Items []invoiceUploadResult `json:"items"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if len(response.Items) != 2 || response.Items[0].Status != "pending" || response.Items[1].ErrorCode != "invalid_pdf" {
		t.Fatalf("批量结果不正确: %+v", response.Items)
	}
	var invoice models.Invoice
	if err := db.First(&invoice, response.Items[0].InvoiceID).Error; err != nil {
		t.Fatal(err)
	}
	wantHash := sha256.Sum256(pdf)
	if invoice.FileSHA256 != hex.EncodeToString(wantHash[:]) {
		t.Fatalf("哈希不一致: %s", invoice.FileSHA256)
	}
	var task models.InvoiceParsingTask
	if err := db.First(&task, response.Items[0].TaskID).Error; err != nil || task.InvoiceID != invoice.ID || task.Status != models.InvoiceParsingTaskPending {
		t.Fatalf("解析任务错误: %+v, %v", task, err)
	}
}

func TestInvoiceUpload_RejectsOverLimitAnd51Files(t *testing.T) {
	_, router, db := setupInvoiceAttachmentTest(t)
	user := createInvoiceTestUser(t, db, "limit-user", "限制用户")
	overstated := append([]byte("%PDF-"), bytes.Repeat([]byte("x"), maxInvoicePDFSize)...)
	req := setAuthContext(invoiceMultipartRequest(t, []invoiceTestFile{{"large.pdf", overstated}}), user.ID)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if !strings.Contains(w.Body.String(), "file_too_large") {
		t.Fatalf("未拒绝实际超限文件: %s", w.Body.String())
	}
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	for i := 0; i < 51; i++ {
		part, _ := writer.CreateFormFile("files", "many.pdf")
		_, _ = part.Write([]byte("%PDF-"))
	}
	_ = writer.Close()
	req = setAuthContext(httptest.NewRequest(http.MethodPost, "/invoices/upload", &body), user.ID)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("51 份文件应被拒绝: %d", w.Code)
	}
}

func TestInvoiceUpload_Allows50PDFsAndUsesFixedMemoryThreshold(t *testing.T) {
	_, router, db := setupInvoiceAttachmentTest(t)
	user := createInvoiceTestUser(t, db, "fifty-user", "五十份用户")
	if invoiceMultipartMemorySize != 1<<20 || invoiceMultipartMemorySize >= maxInvoicePDFSize {
		t.Fatalf("multipart 内存阈值必须固定且小于单文件限制: %d", invoiceMultipartMemorySize)
	}
	files := make([]invoiceTestFile, maxInvoiceUploadFiles)
	for i := range files {
		files[i] = invoiceTestFile{name: "file" + strconv.Itoa(i) + ".pdf", content: []byte("%PDF-1.4\n")}
	}
	req := setAuthContext(invoiceMultipartRequest(t, files), user.ID)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("50 份 PDF 上传失败: %d %s", w.Code, w.Body.String())
	}
	var response struct {
		Items []invoiceUploadResult `json:"items"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil || len(response.Items) != maxInvoiceUploadFiles {
		t.Fatalf("50 份响应错误: %v, %+v", err, response.Items)
	}
	for _, item := range response.Items {
		if item.Status != "pending" || item.InvoiceID == 0 || item.TaskID == 0 {
			t.Fatalf("存在未成功项: %+v", item)
		}
	}
}

func TestInvoiceUpload_ReturnsDuplicateWarning(t *testing.T) {
	_, router, db := setupInvoiceAttachmentTest(t)
	user := createInvoiceTestUser(t, db, "duplicate-user", "重复用户")
	pdf := []byte("%PDF-1.4\n相同内容")
	for attempt := 0; attempt < 2; attempt++ {
		req := setAuthContext(invoiceMultipartRequest(t, []invoiceTestFile{{"same.pdf", pdf}}), user.ID)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if attempt == 1 && !strings.Contains(w.Body.String(), `"duplicate_warning":true`) {
			t.Fatalf("未返回重复预警: %s", w.Body.String())
		}
	}
}

func TestInvoiceAttachment_AuthorizationAndDisposition(t *testing.T) {
	_, router, db := setupInvoiceAttachmentTest(t)
	owner := createInvoiceTestUser(t, db, "owner", "本人")
	other := createInvoiceTestUser(t, db, "other", "他人")
	manager := createInvoiceTestUser(t, db, "manager", "经理")
	role := createInvoiceTestRole(t, db, models.RoleManager)
	assignRole(t, db, manager.ID, role.ID)
	// manager 仅本部门：与 owner 同部门才可访问其附件
	db.Model(&owner).Update("department", "财务部")
	db.Model(&manager).Update("department", "财务部")
	root := t.TempDir()
	content := []byte("%PDF-1.4\n附件")
	if err := os.WriteFile(filepath.Join(root, "invoice.pdf"), content, 0600); err != nil {
		t.Fatal(err)
	}
	config := models.StorageConfig{Name: "附件存储", Type: "local", Enabled: true, IsDefault: false, Config: datatypes.JSON([]byte(`{"root_path":"` + root + `"}`))}
	if err := db.Create(&config).Error; err != nil {
		t.Fatal(err)
	}
	file := models.SysFile{StorageType: "local", Path: "invoice.pdf", OriginalName: "evil\r\n.pdf", Size: int64(len(content)), StorageConfigID: &config.ID, CreatedBy: &owner.ID}
	if err := db.Create(&file).Error; err != nil {
		t.Fatal(err)
	}
	invoice := models.Invoice{UserID: &owner.ID, ApplicantID: &owner.ID, AttachmentFileID: &file.ID, InvoiceDate: time.Now(), Amount: 1, Seller: "测试", Status: models.InvoiceStatusDraft}
	if err := db.Create(&invoice).Error; err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		user        uint
		path        string
		want        int
		disposition string
	}{{other.ID, "/invoices/" + strconv.Itoa(int(invoice.ID)) + "/attachment", http.StatusNotFound, ""}, {owner.ID, "/invoices/" + strconv.Itoa(int(invoice.ID)) + "/attachment", http.StatusOK, "inline"}, {manager.ID, "/invoices/" + strconv.Itoa(int(invoice.ID)) + "/attachment/download", http.StatusOK, "attachment"}} {
		req := setAuthContext(httptest.NewRequest(http.MethodGet, test.path, nil), test.user)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != test.want || (test.disposition != "" && !strings.HasPrefix(w.Header().Get("Content-Disposition"), test.disposition)) {
			t.Fatalf("访问结果错误: code=%d disposition=%q", w.Code, w.Header().Get("Content-Disposition"))
		}
		if test.disposition != "" && w.Header().Get("X-Content-Type-Options") != "nosniff" {
			t.Fatalf("缺少 nosniff 安全响应头: %q", w.Header().Get("X-Content-Type-Options"))
		}
		if test.disposition != "" && w.Header().Get("Cache-Control") != "private, no-store" {
			t.Fatalf("缺少私有禁缓存响应头: %q", w.Header().Get("Cache-Control"))
		}
		if strings.Contains(w.Header().Get("Content-Disposition"), "\r") || strings.Contains(w.Header().Get("Content-Disposition"), "\n") {
			t.Fatalf("文件名注入响应头: %q", w.Header().Get("Content-Disposition"))
		}
	}
}

func TestInvoiceUpload_SameNamesUseSeparateObjectPaths(t *testing.T) {
	_, router, db := setupInvoiceAttachmentTest(t)
	user := createInvoiceTestUser(t, db, "path-user", "路径用户")
	contents := [][]byte{[]byte("%PDF-1.4\n甲"), []byte("%PDF-1.4\n乙")}
	for _, files := range [][]invoiceTestFile{{{"same.pdf", contents[0]}, {"same.pdf", contents[1]}}, {{"same.pdf", contents[0]}}} {
		w := httptest.NewRecorder()
		router.ServeHTTP(w, setAuthContext(invoiceMultipartRequest(t, files), user.ID))
		if w.Code != http.StatusCreated {
			t.Fatalf("上传失败: %d %s", w.Code, w.Body.String())
		}
	}
	var files []models.SysFile
	if err := db.Order("id").Find(&files).Error; err != nil || len(files) != 3 {
		t.Fatalf("读取附件失败: %v, %d", err, len(files))
	}
	seen := map[string]bool{}
	for _, file := range files {
		if seen[file.Path] || !strings.Contains(file.Path, "invoice/pdf/") || file.OriginalName != "same.pdf" {
			t.Fatalf("路径未隔离或原始名错误: %+v", file)
		}
		seen[file.Path] = true
	}
	var invoices []models.Invoice
	if err := db.Order("id").Find(&invoices).Error; err != nil {
		t.Fatal(err)
	}
	for i, invoice := range invoices {
		path := "/invoices/" + strconv.Itoa(int(invoice.ID)) + "/attachment"
		w := httptest.NewRecorder()
		router.ServeHTTP(w, setAuthContext(httptest.NewRequest(http.MethodGet, path, nil), user.ID))
		if w.Code != http.StatusOK || !bytes.Equal(w.Body.Bytes(), contents[i%2]) {
			t.Fatalf("附件内容错误: invoice=%d", invoice.ID)
		}
	}
}

func TestInvoiceCleanup_RetriesObjectAndClaimsLeaseOnce(t *testing.T) {
	handler, _, db := setupInvoiceAttachmentTest(t)
	root := t.TempDir()
	config := models.StorageConfig{Name: "清理存储", Type: "local", Enabled: true, Config: datatypes.JSON([]byte(`{"root_path":"` + root + `"}`))}
	if err := db.Create(&config).Error; err != nil {
		t.Fatal(err)
	}
	path := "invoice/pdf/orphan.pdf"
	fullPath := filepath.Join(root, path)
	if err := os.MkdirAll(filepath.Dir(fullPath), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(fullPath, []byte("orphan"), 0600); err != nil {
		t.Fatal(err)
	}
	task := models.InvoiceFileCleanupTask{StorageConfigID: config.ID, ObjectPath: path, Status: cleanupTaskPending}
	if err := db.Create(&task).Error; err != nil {
		t.Fatal(err)
	}
	first, ok := handler.claimInvoiceCleanupTask(context.Background())
	if !ok || first.ID != task.ID {
		t.Fatalf("首次领取失败: %+v", first)
	}
	if _, ok := handler.claimInvoiceCleanupTask(context.Background()); ok {
		t.Fatal("有效租约不应被重复领取")
	}
	handler.finishInvoiceCleanupTask(context.Background(), first)
	if _, err := os.Stat(fullPath); !os.IsNotExist(err) {
		t.Fatalf("物理对象未清理: %v", err)
	}
	if err := db.First(&models.InvoiceFileCleanupTask{}, task.ID).Error; err == nil {
		t.Fatal("清理成功后任务未删除")
	}
}

func TestInvoiceCleanup_PreservesFailedTask(t *testing.T) {
	handler, _, db := setupInvoiceAttachmentTest(t)
	task := models.InvoiceFileCleanupTask{StorageConfigID: 99999, ObjectPath: "invoice/pdf/missing.pdf", Status: cleanupTaskPending}
	if err := db.Create(&task).Error; err != nil {
		t.Fatal(err)
	}
	handler.runInvoiceFileCleanup(context.Background(), 1)
	var stored models.InvoiceFileCleanupTask
	if err := db.First(&stored, task.ID).Error; err != nil || stored.Status != cleanupTaskPending || stored.Attempts != 1 {
		t.Fatalf("失败任务未保留追踪: %+v, %v", stored, err)
	}
}

func TestInvoiceCleanup_UsesIndependentContextAndProtectsNewLease(t *testing.T) {
	handler, _, db := setupInvoiceAttachmentTest(t)
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	if taskID, err := handler.queueInvoiceCleanup(99999, "invoice/pdf/canceled.pdf", nil, context.Canceled); err != nil || taskID == 0 {
		t.Fatalf("取消请求上下文后仍应入队: task=%d err=%v", taskID, err)
	}
	first, ok := handler.claimInvoiceCleanupTask(context.Background())
	if !ok {
		t.Fatal("首次租约领取失败")
	}
	if err := db.Model(first).Update("locked_until", time.Now().Add(-time.Second)).Error; err != nil {
		t.Fatal(err)
	}
	second, ok := handler.claimInvoiceCleanupTask(context.Background())
	if !ok || second.OwnerToken == first.OwnerToken {
		t.Fatalf("过期租约未被新令牌接管: first=%+v second=%+v", first, second)
	}
	handler.finishInvoiceCleanupTask(canceled, first)
	var stored models.InvoiceFileCleanupTask
	if err := db.First(&stored, first.ID).Error; err != nil || stored.OwnerToken != second.OwnerToken || stored.Status != cleanupTaskRunning {
		t.Fatalf("旧令牌覆盖了新租约: %+v, %v", stored, err)
	}
}

func TestInvoiceUpload_FailedDriverUploadQueuesCleanup(t *testing.T) {
	_, router, db := setupInvoiceAttachmentTest(t)
	user := createInvoiceTestUser(t, db, "failed-upload", "失败上传")
	config := models.StorageConfig{Name: "失败驱动", Type: "invoice-fake", Enabled: true, IsDefault: false, Config: datatypes.JSON([]byte(`{}`))}
	if err := db.Create(&config).Error; err != nil {
		t.Fatal(err)
	}
	rule := models.StorageRule{StorageID: config.ID, ModuleCode: "invoice", ResourceType: "pdf", Enabled: true}
	if err := db.Create(&rule).Error; err != nil {
		t.Fatal(err)
	}
	fake := &invoiceFakeDriver{uploadErr: errors.New("上传后失败")}
	registry := storage.NewRegistry()
	registry.Register("invoice-fake", func() storage.Driver { return fake })
	previous := storage.GlobalManager
	storage.GlobalManager = storage.NewStorageManager(db, registry)
	t.Cleanup(func() { storage.GlobalManager = previous })
	w := httptest.NewRecorder()
	router.ServeHTTP(w, setAuthContext(invoiceMultipartRequest(t, []invoiceTestFile{{"failed.pdf", []byte("%PDF-1.4\n")}}), user.ID))
	var task models.InvoiceFileCleanupTask
	if err := db.First(&task).Error; err != nil || task.StorageConfigID != config.ID || fake.deleteCalls != 0 {
		t.Fatalf("上传失败未持久追踪: task=%+v err=%v delete=%d", task, err, fake.deleteCalls)
	}
}

func TestInvoiceCleanup_QueueFailureDeletesExistingSysFile(t *testing.T) {
	handler, _, db := setupInvoiceAttachmentTest(t)
	user := createInvoiceTestUser(t, db, "cleanup-user", "清理用户")
	config := models.StorageConfig{Name: "删除驱动", Type: "invoice-fake-delete", Enabled: true, Config: datatypes.JSON([]byte(`{}`))}
	if err := db.Create(&config).Error; err != nil {
		t.Fatal(err)
	}
	fake := &invoiceFakeDriver{}
	registry := storage.NewRegistry()
	registry.Register("invoice-fake-delete", func() storage.Driver { return fake })
	previous := storage.GlobalManager
	storage.GlobalManager = storage.NewStorageManager(db, registry)
	t.Cleanup(func() { storage.GlobalManager = previous })
	file := models.SysFile{StorageType: "invoice-fake-delete", Path: "invoice/pdf/orphan.pdf", StorageConfigID: &config.ID, CreatedBy: &user.ID}
	if err := db.Create(&file).Error; err != nil {
		t.Fatal(err)
	}
	db.Exec("DROP TABLE invoice_file_cleanup_tasks")
	handler.queueInvoiceSysFileCleanup(context.Background(), &file)
	if fake.deleteCalls != 1 {
		t.Fatalf("入队失败未同步删除对象: %d", fake.deleteCalls)
	}
	if err := db.First(&models.SysFile{}, file.ID).Error; err == nil {
		t.Fatal("物理删除成功后 SysFile 元数据残留")
	}
}

func TestInvoiceUpload_MetadataFailureQueuesCleanup(t *testing.T) {
	_, router, db := setupInvoiceAttachmentTest(t)
	user := createInvoiceTestUser(t, db, "metadata-fail", "元数据失败")
	db.Callback().Create().Before("gorm:create").Register("invoice_test_fail_sysfile", func(tx *gorm.DB) {
		if tx.Statement.Table == "sys_files" {
			tx.AddError(errors.New("模拟元数据失败"))
		}
	})
	t.Cleanup(func() { db.Callback().Create().Remove("invoice_test_fail_sysfile") })
	w := httptest.NewRecorder()
	router.ServeHTTP(w, setAuthContext(invoiceMultipartRequest(t, []invoiceTestFile{{"metadata.pdf", []byte("%PDF-1.4\n")}}), user.ID))
	var task models.InvoiceFileCleanupTask
	if err := db.First(&task).Error; err != nil || task.SysFileID != nil {
		t.Fatalf("元数据失败未建立对象清理任务: %+v, %v", task, err)
	}
}

func TestInvoiceUpload_TaskTransactionFailureQueuesSysFileCleanup(t *testing.T) {
	_, router, db := setupInvoiceAttachmentTest(t)
	user := createInvoiceTestUser(t, db, "task-fail", "任务失败")
	db.Callback().Create().Before("gorm:create").Register("invoice_test_fail_task", func(tx *gorm.DB) {
		if tx.Statement.Table == "invoice_parsing_tasks" {
			tx.AddError(errors.New("模拟任务失败"))
		}
	})
	t.Cleanup(func() { db.Callback().Create().Remove("invoice_test_fail_task") })
	w := httptest.NewRecorder()
	router.ServeHTTP(w, setAuthContext(invoiceMultipartRequest(t, []invoiceTestFile{{"task.pdf", []byte("%PDF-1.4\n")}}), user.ID))
	var task models.InvoiceFileCleanupTask
	if err := db.First(&task).Error; err != nil || task.SysFileID == nil {
		t.Fatalf("事务失败未建立带 SysFileID 的清理任务: %+v, %v", task, err)
	}
	var invoices int64
	db.Model(&models.Invoice{}).Count(&invoices)
	if invoices != 0 {
		t.Fatalf("事务失败后存在发票: %d", invoices)
	}
}

type invoiceFakeDriver struct {
	uploadErr   error
	deleteErr   error
	deleteCalls int
}

func (d *invoiceFakeDriver) Type() string                                        { return "invoice-fake" }
func (d *invoiceFakeDriver) Init([]byte) error                                   { return nil }
func (d *invoiceFakeDriver) Test(context.Context) (*storage.HealthStatus, error) { return nil, nil }
func (d *invoiceFakeDriver) Upload(context.Context, string, io.Reader, int64) error {
	return d.uploadErr
}
func (d *invoiceFakeDriver) Download(context.Context, string) (io.ReadCloser, error) {
	return nil, errors.New("not implemented")
}
func (d *invoiceFakeDriver) Delete(context.Context, string) error {
	d.deleteCalls++
	return d.deleteErr
}
func (d *invoiceFakeDriver) List(context.Context, string) ([]storage.FileInfo, error) {
	return nil, nil
}
func (d *invoiceFakeDriver) Exists(context.Context, string) (bool, error) { return false, nil }

func TestInvoiceAttachment_RejectsOwnerAfterArchiveOrFileDeletion(t *testing.T) {
	_, router, db := setupInvoiceAttachmentTest(t)
	owner := createInvoiceTestUser(t, db, "archive-owner", "归档本人")
	fileID := createInvoiceAttachmentFile(t, db, owner.ID, "locked.pdf")
	invoice := models.Invoice{UserID: &owner.ID, ApplicantID: &owner.ID, AttachmentFileID: &fileID, InvoiceDate: time.Now(), Amount: 1, Seller: "测试", Status: models.InvoiceStatusDraft, ArchiveStatus: models.InvoiceArchiveStatusConfirmed}
	if err := db.Create(&invoice).Error; err != nil {
		t.Fatal(err)
	}
	path := "/invoices/" + strconv.Itoa(int(invoice.ID)) + "/attachment"
	request := setAuthContext(httptest.NewRequest(http.MethodGet, path, nil), owner.ID)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, request)
	if w.Code != http.StatusNotFound {
		t.Fatalf("已归档草稿不应允许本人访问: %d", w.Code)
	}
	if err := db.Model(&invoice).Update("archive_status", models.InvoiceArchiveStatusPending).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Delete(&models.SysFile{}, fileID).Error; err != nil {
		t.Fatal(err)
	}
	w = httptest.NewRecorder()
	router.ServeHTTP(w, request)
	if w.Code != http.StatusNotFound {
		t.Fatalf("已删除附件不应允许访问: %d", w.Code)
	}
}

func TestInvoiceAttachment_AllowsSuperAdmin(t *testing.T) {
	_, router, db := setupInvoiceAttachmentTest(t)
	owner := createInvoiceTestUser(t, db, "admin-owner", "附件本人")
	admin := createInvoiceTestUser(t, db, "super-admin", "超级管理员")
	role := createInvoiceTestRole(t, db, "super_admin")
	assignRole(t, db, admin.ID, role.ID)
	fileID := createInvoiceAttachmentFile(t, db, owner.ID, "admin.pdf")
	invoice := models.Invoice{UserID: &owner.ID, ApplicantID: &owner.ID, AttachmentFileID: &fileID, InvoiceDate: time.Now(), Amount: 1, Seller: "测试", Status: models.InvoiceStatusSubmitted}
	if err := db.Create(&invoice).Error; err != nil {
		t.Fatal(err)
	}
	path := "/invoices/" + strconv.Itoa(int(invoice.ID)) + "/attachment"
	w := httptest.NewRecorder()
	router.ServeHTTP(w, setAuthContext(httptest.NewRequest(http.MethodGet, path, nil), admin.ID))
	if w.Code != http.StatusOK {
		t.Fatalf("super_admin 应按 admin 权限访问: %d", w.Code)
	}
}

func createInvoiceAttachmentFile(t *testing.T, db *gorm.DB, userID uint, name string) uint {
	t.Helper()
	root := t.TempDir()
	content := []byte("%PDF-1.4\n附件")
	if err := os.WriteFile(filepath.Join(root, name), content, 0600); err != nil {
		t.Fatal(err)
	}
	config := models.StorageConfig{Name: name, Type: "local", Enabled: true, Config: datatypes.JSON([]byte(`{"root_path":"` + root + `"}`))}
	if err := db.Create(&config).Error; err != nil {
		t.Fatal(err)
	}
	file := models.SysFile{StorageType: "local", Path: name, OriginalName: name, Size: int64(len(content)), StorageConfigID: &config.ID, CreatedBy: &userID}
	if err := db.Create(&file).Error; err != nil {
		t.Fatal(err)
	}
	return file.ID
}
