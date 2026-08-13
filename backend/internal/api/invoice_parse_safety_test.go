package api

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"gorm.io/gorm"

	"siapp/internal/models"
)

const safetyInvoiceText = "发票号码：SAFE123456\n不含税金额：100.00\n税额：13.00\n价税合计：113.00\n项目|规格|件|1|100|100|13%|13"

type blockingInvoiceExtractor struct {
	started chan<- struct{}
	release <-chan struct{}
	text    string
}

func (e blockingInvoiceExtractor) Extract(context.Context, string) (string, error) {
	e.started <- struct{}{}
	<-e.release
	return e.text, nil
}

type observingInvoiceExtractor struct {
	mu     sync.Mutex
	paths  []string
	perms  []os.FileMode
	data   [][]byte
	text   string
	err    error
	cancel context.CancelFunc
}

func (e *observingInvoiceExtractor) Extract(_ context.Context, path string) (string, error) {
	info, statErr := os.Stat(path)
	data, readErr := os.ReadFile(path)
	e.mu.Lock()
	e.paths, e.perms, e.data = append(e.paths, path), append(e.perms, info.Mode().Perm()), append(e.data, data)
	e.mu.Unlock()
	if statErr != nil || readErr != nil {
		return "", errors.New("临时文件不可读")
	}
	if e.cancel != nil {
		e.cancel()
	}
	return e.text, e.err
}

func safetyInvoice(t *testing.T, db *gorm.DB) (models.Invoice, models.InvoiceParsingTask, uint) {
	t.Helper()
	user := createInvoiceTestUser(t, db, fmt.Sprintf("safety-%d", time.Now().UnixNano()), "安全测试")
	fileID := createInvoiceAttachmentFile(t, db, user.ID, "safety.pdf")
	invoice := models.Invoice{UserID: &user.ID, ApplicantID: &user.ID, AttachmentFileID: &fileID, InvoiceDate: time.Now(), Amount: 1, Seller: "人工原值", Status: models.InvoiceStatusDraft, ArchiveStatus: models.InvoiceArchiveStatusPending}
	if err := db.Create(&invoice).Error; err != nil {
		t.Fatal(err)
	}
	item := models.InvoiceItem{InvoiceID: invoice.ID, LineNo: 1, Name: "人工明细", Amount: 1}
	if err := db.Create(&item).Error; err != nil {
		t.Fatal(err)
	}
	task := models.InvoiceParsingTask{InvoiceID: invoice.ID}
	if err := db.Create(&task).Error; err != nil {
		t.Fatal(err)
	}
	return invoice, task, fileID
}

func TestInvoiceParsingSafety_WorkerSnapshotConflicts(t *testing.T) {
	for _, test := range []struct {
		name   string
		change func(*testing.T, *gorm.DB, *models.Invoice, uint)
	}{
		{"人工编辑", func(_ *testing.T, db *gorm.DB, invoice *models.Invoice, _ uint) {
			db.Model(invoice).Updates(map[string]any{"seller": "人工更正", "remark": "不得覆盖"})
		}},
		{"已确认", func(_ *testing.T, db *gorm.DB, invoice *models.Invoice, _ uint) {
			db.Model(invoice).Update("archive_status", models.InvoiceArchiveStatusConfirmed)
		}},
		{"已作废", func(_ *testing.T, db *gorm.DB, invoice *models.Invoice, _ uint) {
			db.Model(invoice).Update("archive_status", models.InvoiceArchiveStatusVoided)
		}},
		{"逻辑删除", func(_ *testing.T, db *gorm.DB, invoice *models.Invoice, _ uint) { db.Delete(invoice) }},
		{"替换附件", func(t *testing.T, db *gorm.DB, invoice *models.Invoice, _ uint) {
			userID := *invoice.UserID
			id := createInvoiceAttachmentFile(t, db, userID, "replacement.pdf")
			db.Model(invoice).Update("attachment_file_id", id)
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			handler, _, db := setupInvoiceAttachmentTest(t)
			invoice, task, fileID := safetyInvoice(t, db)
			started, release := make(chan struct{}, 1), make(chan struct{})
			handler.invoicePDFText = blockingInvoiceExtractor{started: started, release: release, text: safetyInvoiceText}
			handler.invoiceOCRText = &fakeInvoiceExtractor{}
			claimed, ok := handler.claimInvoiceParsingTask(context.Background())
			if !ok {
				t.Fatal("领取失败")
			}
			done := make(chan struct{})
			go func() { handler.processInvoiceParsingTask(context.Background(), claimed); close(done) }()
			<-started
			test.change(t, db, &invoice, fileID)
			close(release)
			<-done
			var storedTask models.InvoiceParsingTask
			if err := db.First(&storedTask, task.ID).Error; err != nil || storedTask.Status != models.InvoiceParsingTaskFailed || storedTask.ErrorCode != "invoice_changed" {
				t.Fatalf("任务未以冲突失败: %+v %v", storedTask, err)
			}
			var items []models.InvoiceItem
			if err := db.Where("invoice_id = ?", invoice.ID).Find(&items).Error; err != nil || len(items) != 1 || items[0].Name != "人工明细" {
				t.Fatalf("原明细被改变: %+v %v", items, err)
			}
			var current models.Invoice
			if err := db.Unscoped().First(&current, invoice.ID).Error; err != nil || current.InvoiceNo != "" || current.Seller == "" {
				t.Fatalf("快照冲突覆盖了发票: %+v %v", current, err)
			}
		})
	}
}

func TestInvoiceParsingSafety_ConcurrentClaimAndExpiredTokens(t *testing.T) {
	handler, _, db := setupInvoiceAttachmentTest(t)
	if sqlDB, err := db.DB(); err == nil {
		sqlDB.SetMaxOpenConns(1)
	}
	_, task, _ := safetyInvoice(t, db)
	start := make(chan struct{})
	results := make(chan *models.InvoiceParsingTask, 2)
	for range 2 {
		go func() {
			<-start
			claimed, ok := handler.claimInvoiceParsingTask(context.Background())
			if ok {
				results <- claimed
			} else {
				results <- nil
			}
		}()
	}
	close(start)
	first, second := <-results, <-results
	if (first == nil) == (second == nil) {
		t.Fatalf("并发领取结果错误: %+v %+v", first, second)
	}
	claimed := first
	if claimed == nil {
		claimed = second
	}
	var stored models.InvoiceParsingTask
	if err := db.First(&stored, task.ID).Error; err != nil || stored.AttemptCount != 1 {
		t.Fatalf("领取次数错误: %+v %v", stored, err)
	}
	db.Model(&stored).Update("locked_until", time.Now().Add(-time.Second))
	if handler.updateInvoiceParseTask(claimed, map[string]any{"status": models.InvoiceParsingTaskSucceeded}) {
		t.Fatal("过期旧令牌不得完成")
	}
	newClaim, ok := handler.claimInvoiceParsingTask(context.Background())
	if !ok || newClaim.LockedBy == claimed.LockedBy {
		t.Fatalf("过期任务未由新令牌接管: %+v", newClaim)
	}
	if handler.updateInvoiceParseTask(claimed, map[string]any{"status": models.InvoiceParsingTaskFailed}) {
		t.Fatal("被接管旧令牌不得失败任务")
	}
}

func TestInvoiceParsingSafety_CancelPersistsRetryAndCleansTemporaryFile(t *testing.T) {
	handler, _, db := setupInvoiceAttachmentTest(t)
	invoice, task, _ := safetyInvoice(t, db)
	ctx, cancel := context.WithCancel(context.Background())
	observed := &observingInvoiceExtractor{text: "短文本", cancel: cancel}
	handler.invoicePDFText, handler.invoiceOCRText = observed, observed
	claimed, ok := handler.claimInvoiceParsingTask(context.Background())
	if !ok {
		t.Fatal("领取失败")
	}
	handler.processInvoiceParsingTask(ctx, claimed)
	var stored models.InvoiceParsingTask
	if err := db.First(&stored, task.ID).Error; err != nil || stored.Status != models.InvoiceParsingTaskPending || stored.AvailableAt == nil || stored.ErrorCode != "execution_cancelled" {
		t.Fatalf("取消后未持久化重试: %+v %v", stored, err)
	}
	assertObservedTemporaryFiles(t, observed.paths, observed.perms, observed.data)
	if handler.updateInvoiceParseTask(claimed, map[string]any{"status": models.InvoiceParsingTaskSucceeded}) {
		t.Fatal("旧令牌不得在重试后更新")
	}
	if invoice.AttachmentFileID == nil {
		t.Fatal("附件丢失")
	}
}

func TestInvoiceParsingSafety_CancelAfterExtractionRollsBackInvoice(t *testing.T) {
	handler, _, db := setupInvoiceAttachmentTest(t)
	invoice, task, _ := safetyInvoice(t, db)
	parent, cancel := context.WithCancel(context.Background())
	handler.invoicePDFText = &fakeInvoiceExtractor{text: safetyInvoiceText}
	handler.invoiceOCRText = &fakeInvoiceExtractor{}
	handler.invoiceParseBeforeSave = cancel
	claimed, ok := handler.claimInvoiceParsingTask(context.Background())
	if !ok {
		t.Fatal("领取失败")
	}
	handler.processInvoiceParsingTask(parent, claimed)
	var current models.Invoice
	var items []models.InvoiceItem
	var stored models.InvoiceParsingTask
	if db.First(&current, invoice.ID).Error != nil || db.Where("invoice_id = ?", invoice.ID).Find(&items).Error != nil || db.First(&stored, task.ID).Error != nil || current.InvoiceNo != "" || current.Seller != "人工原值" || len(items) != 1 || items[0].Name != "人工明细" || stored.Status != models.InvoiceParsingTaskPending || stored.ErrorCode != "execution_cancelled" || stored.AvailableAt == nil {
		t.Fatalf("取消后出现部分提交: invoice=%+v items=%+v task=%+v", current, items, stored)
	}
}
