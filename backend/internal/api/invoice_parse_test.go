package api

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"gorm.io/gorm"
	"siapp/internal/models"
)

type fakeInvoiceExtractor struct {
	text  string
	err   error
	calls int
}

func (f *fakeInvoiceExtractor) Extract(context.Context, string) (string, error) {
	f.calls++
	return f.text, f.err
}

func TestParseInvoiceFields_ParsesInvoiceNumberAndAmount(t *testing.T) {
	parsed := parseInvoiceFields("增值税发票\n发票号码：ABC123456\n价税合计：￥1,234.56")
	if parsed.invoiceNo != "ABC123456" || parsed.total != 1234.56 {
		t.Fatalf("字段解析错误: %+v", parsed)
	}
}

func TestParseInvoiceFields_CompleteVATAndItems(t *testing.T) {
	text := "发票代码：123456789012\n发票号码：ABC123456\n开票日期：2026年08月13日\n发票类型：增值税专用发票\n销售方名称：销售公司\n销售方纳税人识别号：SELLER123\n购买方名称：购买公司\n购买方纳税人识别号：BUYER123\n不含税金额：1,000.00\n税额：130.00\n价税合计：1,130.01\n备注：测试备注\n货物|规格|件|2|500.00|1000.00|13%|130.00"
	p := parseInvoiceFields(text)
	if p.code == "" || p.invoiceNo != "ABC123456" || p.date.IsZero() || p.sellerTaxNo != "SELLER123" || p.total != 1130.01 || len(p.items) != 1 || p.items[0].TaxRate != 13 {
		t.Fatalf("完整样本解析失败: %+v", p)
	}
	if !strings.Contains(string(p.confidence), `"amount_anomaly":false`) {
		t.Fatalf("金额容差判断错误: %s", p.confidence)
	}
}

func TestParseInvoiceFields_RedAndMissingFieldsRemainSuccessful(t *testing.T) {
	p := parseInvoiceFields("发票号码：RED123456\n价税合计：-1,130.00\n不含税金额：-1,000.00\n税额：-130.00")
	if p.total != -1130 || p.invoiceNo == "" || !strings.Contains(string(p.confidence), "missing_fields") {
		t.Fatalf("红字或缺失字段解析错误: %+v", p)
	}
}

func TestParseInvoiceFields_ElectronicNumberAndInvalidDate(t *testing.T) {
	parsed := parseInvoiceFields("电子票号：ELECTRONIC123\n发票号码：NORMAL123456\n开票日期：2026-02-30")
	if parsed.electronicNo != "ELECTRONIC123" || parsed.invoiceNo != "NORMAL123456" {
		t.Fatalf("电子票号必须独立解析: %+v", parsed)
	}
	if !parsed.date.IsZero() || !strings.Contains(string(parsed.confidence), `"invoice_date"`) {
		t.Fatalf("无效日期必须不写入且标记低置信度: %+v", parsed)
	}
}

func TestParseInvoiceDate_NormalizesSlashes(t *testing.T) {
	date := parseInvoiceDate("2026/08/13")
	if date.IsZero() || date.Format("2006-01-02") != "2026-08-13" {
		t.Fatalf("斜杠日期归一化失败: %v", date)
	}
}

func TestInvoiceTextQuality_RecognizesSupportedVouchersAndRejectsGarbage(t *testing.T) {
	for _, text := range []string{
		"收据\n收款金额：100.00\n收款单位：测试公司，附加说明文字用于满足长度要求",
		"付款凭证\n付款金额：100.00\n付款方：测试公司，附加说明文字用于满足长度要求",
		"电子行程单\n行程金额：100.00\n乘客：测试用户，附加说明文字用于满足长度要求",
	} {
		if !invoiceTextQuality(text) {
			t.Fatalf("凭证文本应可用: %q", text)
		}
	}
	if invoiceTextQuality(strings.Repeat("\uFFFD", 30) + "发票") {
		t.Fatal("高比例乱码不得视为可用文本")
	}
}

func TestAmountAnomaly_ToleranceAndRedInvoice(t *testing.T) {
	for _, test := range []struct {
		total float64
		want  bool
	}{{110.019, false}, {110.020, false}, {110.021, true}, {-110.021, true}} {
		if got := amountAnomaly(-100, -10, -test.total); got != test.want {
			t.Fatalf("金额容差 total=%v got=%v want=%v", test.total, got, test.want)
		}
	}
}

func TestParseInvoiceFields_AmountAnomalyIsLowConfidence(t *testing.T) {
	parsed := parseInvoiceFields("发票号码：ABC123456\n不含税金额：100.00\n税额：10.00\n价税合计：120.00")
	if !strings.Contains(string(parsed.confidence), `"amount_anomaly":true`) || !strings.Contains(string(parsed.confidence), `"amount"`) {
		t.Fatalf("金额异常必须标记低置信度: %s", parsed.confidence)
	}
}

func TestInvoiceParsingClaim_LeaseAndAvailableAt(t *testing.T) {
	handler, _, db := setupInvoiceAttachmentTest(t)
	invoiceID := createParseTestInvoice(t, db)
	future := time.Now().Add(time.Minute)
	tasks := []models.InvoiceParsingTask{{InvoiceID: invoiceID, AvailableAt: &future}, {InvoiceID: createParseTestInvoice(t, db)}}
	for i := range tasks {
		if err := db.Create(&tasks[i]).Error; err != nil {
			t.Fatal(err)
		}
	}
	claimed, ok := handler.claimInvoiceParsingTask(context.Background())
	if !ok || claimed.InvoiceID != tasks[1].InvoiceID || claimed.LockedBy == "" {
		t.Fatalf("领取或延迟判断错误: %+v", claimed)
	}
	if _, ok := handler.claimInvoiceParsingTask(context.Background()); ok {
		t.Fatal("有效租约不应重复领取")
	}
}

func TestInvoiceParsingFailure_BackoffAndNonRetryable(t *testing.T) {
	handler, _, db := setupInvoiceAttachmentTest(t)
	task := models.InvoiceParsingTask{InvoiceID: createParseTestInvoice(t, db), Status: models.InvoiceParsingTaskRunning, AttemptCount: 1, LockedBy: "token", LockedUntil: timePtr(time.Now().Add(time.Minute))}
	if err := db.Create(&task).Error; err != nil {
		t.Fatal(err)
	}
	handler.finishInvoiceParseFailure(context.Background(), &task, invoiceParseError{"temporary", true})
	var stored models.InvoiceParsingTask
	if err := db.First(&stored, task.ID).Error; err != nil || stored.Status != models.InvoiceParsingTaskPending || stored.AvailableAt == nil {
		t.Fatalf("重试退避错误: %+v %v", stored, err)
	}
	locked := models.InvoiceParsingTask{InvoiceID: createParseTestInvoice(t, db), Status: models.InvoiceParsingTaskRunning, AttemptCount: 1, LockedBy: "token2", LockedUntil: timePtr(time.Now().Add(time.Minute))}
	if err := db.Create(&locked).Error; err != nil {
		t.Fatal(err)
	}
	handler.finishInvoiceParseFailure(context.Background(), &locked, invoiceParseError{"no_recognizable_content", false})
	stored = models.InvoiceParsingTask{}
	if err := db.First(&stored, locked.ID).Error; err != nil || stored.Status != models.InvoiceParsingTaskFailed {
		t.Fatalf("不可重试错误未终态: %+v %v", stored, err)
	}
}

func TestInvoiceParsingExpiredThirdAttemptFails(t *testing.T) {
	handler, _, db := setupInvoiceAttachmentTest(t)
	task := models.InvoiceParsingTask{InvoiceID: createParseTestInvoice(t, db), Status: models.InvoiceParsingTaskRunning, AttemptCount: 3, MaxAttempts: 3, LockedUntil: timePtr(time.Now().Add(-time.Minute))}
	if err := db.Create(&task).Error; err != nil {
		t.Fatal(err)
	}
	handler.failExpiredInvoiceTasks(context.Background(), time.Now())
	var stored models.InvoiceParsingTask
	if err := db.First(&stored, task.ID).Error; err != nil || stored.Status != models.InvoiceParsingTaskFailed || stored.ErrorCode != "worker_lease_expired" || stored.LockedBy != "" || stored.LockedUntil != nil || stored.AvailableAt != nil {
		t.Fatalf("过期任务未收敛: %+v %v", stored, err)
	}
}

func TestInvoiceParsing_DocreaderFallbackAndTransactionWrite(t *testing.T) {
	handler, _, db := setupInvoiceAttachmentTest(t)
	user := createInvoiceTestUser(t, db, "parse-write", "解析写入")
	fileID := createInvoiceAttachmentFile(t, db, user.ID, "parse.pdf")
	invoice := models.Invoice{UserID: &user.ID, ApplicantID: &user.ID, AttachmentFileID: &fileID, InvoiceDate: time.Now(), Amount: 1, Seller: "待识别", Status: models.InvoiceStatusDraft, ArchiveStatus: models.InvoiceArchiveStatusPending}
	if err := db.Create(&invoice).Error; err != nil {
		t.Fatal(err)
	}
	task := models.InvoiceParsingTask{InvoiceID: invoice.ID}
	if err := db.Create(&task).Error; err != nil {
		t.Fatal(err)
	}
	doc := &fakeInvoiceExtractor{text: "发票号码：DOC123456\n发票代码：123456789012\n开票日期：2026-08-13\n销售方名称：销售公司\n销售方纳税人识别号：SELLER\n购买方名称：购买公司\n购买方纳税人识别号：BUYER\n不含税金额：1,000.00\n税额：234.56\n价税合计：1,234.56\n项目|规格|件|2|500|1000|13%|234.56"}
	ocr := &fakeInvoiceExtractor{text: "不应调用"}
	handler.invoicePDFText, handler.invoiceOCRText = doc, ocr
	claimed, ok := handler.claimInvoiceParsingTask(context.Background())
	if !ok {
		t.Fatal("领取失败")
	}
	handler.processInvoiceParsingTask(context.Background(), claimed)
	if err := db.First(&invoice, invoice.ID).Error; err != nil || invoice.InvoiceNo != "DOC123456" || invoice.TotalAmount != 1234.56 {
		var current models.InvoiceParsingTask
		_ = db.First(&current, task.ID).Error
		t.Fatalf("事务写入失败: %+v task=%+v %v", invoice, current, err)
	}
	if invoice.RecognitionSource != "docreader" || invoice.SellerTaxNo != "SELLER" || invoice.BuyerTaxNo != "BUYER" {
		t.Fatalf("来源或主字段未持久化: %+v", invoice)
	}
	var items []models.InvoiceItem
	if err := db.Where("invoice_id = ?", invoice.ID).Find(&items).Error; err != nil || len(items) != 1 || items[0].LineNo != 1 {
		t.Fatalf("明细未写入: %+v %v", items, err)
	}
	if doc.calls != 1 || ocr.calls != 0 {
		t.Fatalf("有效 docreader 文本不应调用 OCR: doc=%d ocr=%d", doc.calls, ocr.calls)
	}
	if err := db.First(&task, task.ID).Error; err != nil || task.Status != models.InvoiceParsingTaskSucceeded {
		t.Fatalf("任务未成功: %+v %v", task, err)
	}
}

func TestInvoiceParsing_LowQualityUsesOCRAndEmptyFails(t *testing.T) {
	handler, _, db := setupInvoiceAttachmentTest(t)
	user := createInvoiceTestUser(t, db, "parse-ocr", "解析 OCR")
	fileID := createInvoiceAttachmentFile(t, db, user.ID, "ocr.pdf")
	invoice := models.Invoice{UserID: &user.ID, ApplicantID: &user.ID, AttachmentFileID: &fileID, InvoiceDate: time.Now(), Amount: 1, Seller: "待识别", Status: models.InvoiceStatusDraft, ArchiveStatus: models.InvoiceArchiveStatusPending}
	if err := db.Create(&invoice).Error; err != nil {
		t.Fatal(err)
	}
	task := models.InvoiceParsingTask{InvoiceID: invoice.ID}
	if err := db.Create(&task).Error; err != nil {
		t.Fatal(err)
	}
	doc := &fakeInvoiceExtractor{text: "短文本"}
	ocr := &fakeInvoiceExtractor{text: ""}
	handler.invoicePDFText, handler.invoiceOCRText = doc, ocr
	claimed, ok := handler.claimInvoiceParsingTask(context.Background())
	if !ok {
		t.Fatal("领取失败")
	}
	handler.processInvoiceParsingTask(context.Background(), claimed)
	if doc.calls != 1 || ocr.calls != 1 {
		t.Fatalf("低质量文本未走 OCR: doc=%d ocr=%d", doc.calls, ocr.calls)
	}
	if err := db.First(&task, task.ID).Error; err != nil || task.Status != models.InvoiceParsingTaskFailed || task.ErrorCode != "no_recognizable_content" {
		t.Fatalf("OCR 空文本未终态失败: %+v %v", task, err)
	}
}

func TestInvoiceParsing_ExtractionClassifiesOCRErrors(t *testing.T) {
	handler, _, db := setupInvoiceAttachmentTest(t)
	user := createInvoiceTestUser(t, db, "parse-extract", "解析提取")
	fileID := createInvoiceAttachmentFile(t, db, user.ID, "extract.pdf")
	snapshot := invoiceSnapshot{attachmentID: fileID}
	handler.invoicePDFText = &fakeInvoiceExtractor{text: "这是没有任何业务信号的普通长文本内容用于触发 OCR"}
	handler.invoiceOCRText = &fakeInvoiceExtractor{err: errors.New("ocr failed")}
	_, _, err := handler.extractInvoiceText(context.Background(), snapshot)
	parseErr, ok := err.(invoiceParseError)
	if !ok || parseErr.Code != "extractor_unavailable" || !parseErr.Retryable {
		t.Fatalf("OCR 错误必须可重试: %#v", err)
	}
	handler.invoiceOCRText = &fakeInvoiceExtractor{text: "没有可识别的业务内容，只是普通文本用于测试"}
	_, _, err = handler.extractInvoiceText(context.Background(), snapshot)
	parseErr, ok = err.(invoiceParseError)
	if !ok || parseErr.Code != "no_recognizable_content" || parseErr.Retryable {
		t.Fatalf("OCR 无错误但低质量文本必须不可重试: %#v", err)
	}
}

func TestInvoiceParsing_DownloadUsesPrivateTemporaryFile(t *testing.T) {
	handler, _, db := setupInvoiceAttachmentTest(t)
	user := createInvoiceTestUser(t, db, "parse-temp", "解析临时文件")
	fileID := createInvoiceAttachmentFile(t, db, user.ID, "temporary.pdf")
	path, err := handler.downloadInvoiceParseFile(context.Background(), fileID)
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(path)
	info, err := os.Stat(path)
	if err != nil || info.Mode().Perm() != 0600 {
		t.Fatalf("临时文件权限必须为 0600: mode=%v err=%v", info.Mode(), err)
	}
}

func TestInvoiceParsing_SnapshotConflictPreservesManualChanges(t *testing.T) {
	handler, _, db := setupInvoiceAttachmentTest(t)
	user := createInvoiceTestUser(t, db, "parse-snapshot", "解析快照")
	fileID := createInvoiceAttachmentFile(t, db, user.ID, "snapshot.pdf")
	invoice := models.Invoice{UserID: &user.ID, ApplicantID: &user.ID, AttachmentFileID: &fileID, InvoiceDate: time.Now(), Amount: 1, Seller: "原始销售方", Status: models.InvoiceStatusDraft, ArchiveStatus: models.InvoiceArchiveStatusPending}
	if err := db.Create(&invoice).Error; err != nil {
		t.Fatal(err)
	}
	task := models.InvoiceParsingTask{InvoiceID: invoice.ID}
	if err := db.Create(&task).Error; err != nil {
		t.Fatal(err)
	}
	claimed, ok := handler.claimInvoiceParsingTask(context.Background())
	if !ok {
		t.Fatal("领取失败")
	}
	snapshot, err := handler.loadInvoiceSnapshot(context.Background(), invoice.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&invoice).Updates(map[string]any{"seller": "人工更正", "remark": "保留人工修改"}).Error; err != nil {
		t.Fatal(err)
	}
	err = handler.saveParsedInvoice(context.Background(), claimed, snapshot, parseInvoiceFields("发票号码：NEW123456\n价税合计：100.00"), "发票号码：NEW123456", "docreader")
	parseErr, ok := err.(invoiceParseError)
	if !ok || parseErr.Code != "invoice_changed" {
		t.Fatalf("快照冲突必须终止写入: %#v", err)
	}
	if err := db.First(&invoice, invoice.ID).Error; err != nil || invoice.Seller != "人工更正" || invoice.InvoiceNo != "" {
		t.Fatalf("不得覆盖人工修改: %+v %v", invoice, err)
	}
}

func timePtr(value time.Time) *time.Time { return &value }

func createParseTestInvoice(t *testing.T, db *gorm.DB) uint {
	t.Helper()
	user := createInvoiceTestUser(t, db, fmt.Sprintf("parse-%d", time.Now().UnixNano()), "解析用户")
	invoice := models.Invoice{UserID: &user.ID, ApplicantID: &user.ID, InvoiceDate: time.Now(), Amount: 1, Seller: "测试"}
	if err := db.Create(&invoice).Error; err != nil {
		t.Fatal(err)
	}
	return invoice.ID
}
