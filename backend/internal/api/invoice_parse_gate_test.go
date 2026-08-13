package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
	"siapp/internal/models"
	"siapp/internal/service/storage"
)

func TestInvoiceParsingGate_TemporaryFileLifecycle(t *testing.T) {
	for _, test := range []struct {
		name, doc, ocr string
		docErr         error
		wantStatus     models.InvoiceParsingTaskStatus
		wantCode       string
	}{
		{"文本识别成功", safetyInvoiceText, "", nil, models.InvoiceParsingTaskSucceeded, ""},
		{"低质文本由OCR成功", "短文本", safetyInvoiceText, nil, models.InvoiceParsingTaskSucceeded, ""},
		{"OCR失败", "短文本", "", errors.New("OCR失败"), models.InvoiceParsingTaskPending, "extractor_unavailable"},
		{"取消", "", "", context.Canceled, models.InvoiceParsingTaskPending, "execution_cancelled"},
	} {
		t.Run(test.name, func(t *testing.T) {
			handler, _, db := setupInvoiceAttachmentTest(t)
			_, task, fileID := safetyInvoice(t, db)
			storagePath, objectPath, objectData := invoiceObjectPath(t, db, fileID)
			doc := &observingInvoiceExtractor{text: test.doc, err: test.docErr}
			ocr := &observingInvoiceExtractor{text: test.ocr, err: test.docErr}
			if test.name == "取消" {
				doc.text, ocr.err = "短文本", context.Canceled
			}
			handler.invoicePDFText, handler.invoiceOCRText = doc, ocr
			claimed, ok := handler.claimInvoiceParsingTask(context.Background())
			if !ok {
				t.Fatal("领取失败")
			}
			handler.processInvoiceParsingTask(context.Background(), claimed)
			assertObservedTemporaryFiles(t, append(doc.paths, ocr.paths...), append(doc.perms, ocr.perms...), append(doc.data, ocr.data...))
			for _, path := range append(doc.paths, ocr.paths...) {
				if path == storagePath || path == objectPath {
					t.Fatal("解析器不得接收原始对象路径")
				}
			}
			if data, err := os.ReadFile(objectPath); err != nil || !bytes.Equal(data, objectData) {
				t.Fatalf("原始对象被修改: %v", err)
			}
			var stored models.InvoiceParsingTask
			if err := db.First(&stored, task.ID).Error; err != nil || stored.Status != test.wantStatus || stored.ErrorCode != test.wantCode {
				t.Fatalf("任务状态错误: %+v %v", stored, err)
			}
		})
	}
}

func invoiceObjectPath(t *testing.T, db *gorm.DB, fileID uint) (string, string, []byte) {
	t.Helper()
	var file models.SysFile
	if err := db.First(&file, fileID).Error; err != nil {
		t.Fatal(err)
	}
	var config models.StorageConfig
	if err := db.First(&config, *file.StorageConfigID).Error; err != nil {
		t.Fatal(err)
	}
	var settings struct {
		RootPath string `json:"root_path"`
	}
	if err := json.Unmarshal(config.Config, &settings); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(settings.RootPath, file.Path)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return file.Path, path, data
}

func assertObservedTemporaryFiles(t *testing.T, paths []string, perms []os.FileMode, data [][]byte) {
	t.Helper()
	if len(paths) == 0 || len(paths) != len(perms) || len(paths) != len(data) {
		t.Fatalf("临时文件未被完整观察: paths=%d perms=%d data=%d", len(paths), len(perms), len(data))
	}
	for i, path := range paths {
		if perms[i] != 0600 || !bytes.Equal(data[i], []byte("%PDF-1.4\n附件")) {
			t.Fatalf("临时文件隔离、权限或内容错误: path=%q perm=%o", path, perms[i])
		}
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatalf("临时文件未自动删除: %q err=%v", path, err)
		}
	}
}

func TestInvoiceParsingGate_RejectsActualOversizedObject(t *testing.T) {
	t.Setenv("TMPDIR", t.TempDir())
	if matches, err := filepath.Glob(filepath.Join(os.TempDir(), "invoice-parse-*.pdf")); err != nil || len(matches) != 0 {
		t.Fatalf("测试前临时目录不应有解析文件: %v %v", matches, err)
	}
	handler, _, db := setupInvoiceAttachmentTest(t)
	user := createInvoiceTestUser(t, db, "oversized-object", "超限对象")
	config := models.StorageConfig{Name: "超限假存储", Type: "parse-oversized", Enabled: true, Config: datatypes.JSON([]byte(`{}`))}
	if err := db.Create(&config).Error; err != nil {
		t.Fatal(err)
	}
	file := models.SysFile{StorageType: "parse-oversized", Path: "small.pdf", Size: 1, StorageConfigID: &config.ID, CreatedBy: &user.ID}
	if err := db.Create(&file).Error; err != nil {
		t.Fatal(err)
	}
	registry := storage.NewRegistry()
	registry.Register("parse-oversized", func() storage.Driver { return oversizedInvoiceDriver{} })
	previous := storage.GlobalManager
	storage.GlobalManager = storage.NewStorageManager(db, registry)
	t.Cleanup(func() { storage.GlobalManager = previous })
	invoice := models.Invoice{UserID: &user.ID, ApplicantID: &user.ID, AttachmentFileID: &file.ID, InvoiceDate: time.Now(), Amount: 1, Seller: "超限", ArchiveStatus: models.InvoiceArchiveStatusPending}
	if err := db.Create(&invoice).Error; err != nil {
		t.Fatal(err)
	}
	task := models.InvoiceParsingTask{InvoiceID: invoice.ID}
	if err := db.Create(&task).Error; err != nil {
		t.Fatal(err)
	}
	doc, ocr := &fakeInvoiceExtractor{}, &fakeInvoiceExtractor{}
	handler.invoicePDFText, handler.invoiceOCRText = doc, ocr
	claimed, ok := handler.claimInvoiceParsingTask(context.Background())
	if !ok {
		t.Fatal("领取失败")
	}
	handler.processInvoiceParsingTask(context.Background(), claimed)
	if doc.calls != 0 || ocr.calls != 0 {
		t.Fatalf("超限对象不得调用提取器: doc=%d ocr=%d", doc.calls, ocr.calls)
	}
	if matches, err := filepath.Glob(filepath.Join(os.TempDir(), "invoice-parse-*.pdf")); err != nil || len(matches) != 0 {
		t.Fatalf("超限错误后残留临时文件: %v %v", matches, err)
	}
	if err := db.First(&task, task.ID).Error; err != nil || task.Status != models.InvoiceParsingTaskPending {
		t.Fatal("对象实际大小超过 20MB 必须被拒绝")
	}
}

type oversizedInvoiceDriver struct{}

func (oversizedInvoiceDriver) Type() string                                           { return "parse-oversized" }
func (oversizedInvoiceDriver) Init([]byte) error                                      { return nil }
func (oversizedInvoiceDriver) Test(context.Context) (*storage.HealthStatus, error)    { return nil, nil }
func (oversizedInvoiceDriver) Upload(context.Context, string, io.Reader, int64) error { return nil }
func (oversizedInvoiceDriver) Download(context.Context, string) (io.ReadCloser, error) {
	return io.NopCloser(io.LimitReader(zeroInvoiceReader{}, int64(maxInvoicePDFSize)+1)), nil
}
func (oversizedInvoiceDriver) Delete(context.Context, string) error { return nil }
func (oversizedInvoiceDriver) List(context.Context, string) ([]storage.FileInfo, error) {
	return nil, nil
}
func (oversizedInvoiceDriver) Exists(context.Context, string) (bool, error) { return true, nil }

type zeroInvoiceReader struct{}

func (zeroInvoiceReader) Read(data []byte) (int, error) {
	for i := range data {
		data[i] = 0
	}
	return len(data), nil
}

func TestInvoiceParsingGate_BatchContinuesAfterFailure(t *testing.T) {
	handler, _, db := setupInvoiceAttachmentTest(t)
	firstID := createParseTestInvoice(t, db)
	if err := db.Create(&models.InvoiceParsingTask{InvoiceID: firstID}).Error; err != nil {
		t.Fatal(err)
	}
	_, second, _ := safetyInvoice(t, db)
	handler.invoicePDFText = &fakeInvoiceExtractor{text: safetyInvoiceText}
	handler.invoiceOCRText = &fakeInvoiceExtractor{}
	handler.runInvoiceParsingBatch(context.Background(), 2)
	var tasks []models.InvoiceParsingTask
	if err := db.Order("id").Find(&tasks).Error; err != nil || len(tasks) != 2 || tasks[0].Status != models.InvoiceParsingTaskFailed || tasks[1].ID != second.ID || tasks[1].Status != models.InvoiceParsingTaskSucceeded {
		t.Fatalf("批次隔离错误: %+v %v", tasks, err)
	}
}

func TestInvoiceParsingGate_LeaseLossRollsBackTransaction(t *testing.T) {
	for _, replaceLease := range []bool{false, true} {
		t.Run(map[bool]string{false: "过期", true: "令牌更换"}[replaceLease], func(t *testing.T) {
			handler, _, db := setupInvoiceAttachmentTest(t)
			invoice, task, _ := safetyInvoice(t, db)
			claimed, ok := handler.claimInvoiceParsingTask(context.Background())
			if !ok {
				t.Fatal("领取失败")
			}
			snapshot, err := handler.loadInvoiceSnapshot(context.Background(), invoice.ID)
			if err != nil {
				t.Fatal(err)
			}
			updates := map[string]any{"locked_until": time.Now().Add(-time.Second)}
			if replaceLease {
				updates = map[string]any{"locked_by": "new-token"}
			}
			if err := db.Model(&models.InvoiceParsingTask{}).Where("id = ?", task.ID).Updates(updates).Error; err != nil {
				t.Fatal(err)
			}
			err = handler.saveParsedInvoice(context.Background(), claimed, snapshot, parseInvoiceFields(safetyInvoiceText), safetyInvoiceText, "docreader")
			if parseErr, ok := err.(invoiceParseError); !ok || parseErr.Code != "worker_lease_expired" {
				t.Fatalf("租约失效必须拒绝提交: %#v", err)
			}
			var current models.Invoice
			var items []models.InvoiceItem
			var stored models.InvoiceParsingTask
			if db.First(&current, invoice.ID).Error != nil || db.Where("invoice_id = ?", invoice.ID).Find(&items).Error != nil || db.First(&stored, task.ID).Error != nil || current.Seller != "人工原值" || current.InvoiceNo != "" || len(items) != 1 || items[0].Name != "人工明细" || stored.Status != models.InvoiceParsingTaskRunning {
				t.Fatalf("事务未完整回滚: invoice=%+v items=%+v task=%+v", current, items, stored)
			}
		})
	}
}
