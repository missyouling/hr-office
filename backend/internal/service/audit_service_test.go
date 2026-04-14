package service

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"siapp/internal/models"
)

// setupTestDB 根据环境变量建立数据库连接，确保测试使用 Supabase PostgreSQL。
func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dbType := strings.ToLower(os.Getenv("SIAPP_DATABASE_TYPE"))
	if dbType == "" {
		t.Skip("未配置 SIAPP_DATABASE_TYPE，跳过需要数据库的测试")
	}

	if dbType != "postgres" && dbType != "postgresql" {
		t.Skipf("当前数据库类型 %s 不受支持，跳过", dbType)
	}

	dbHost := os.Getenv("SIAPP_DB_HOST")
	dbPort := os.Getenv("SIAPP_DB_PORT")
	dbUser := os.Getenv("SIAPP_DB_USER")
	dbPassword := os.Getenv("SIAPP_DB_PASSWORD")
	dbName := os.Getenv("SIAPP_DB_NAME")
	sslMode := os.Getenv("SIAPP_DB_SSLMODE")

	if dbHost == "" || dbPort == "" || dbUser == "" || dbPassword == "" || dbName == "" {
		t.Skip("Supabase PostgreSQL 环境变量不完整，跳过")
	}
	if sslMode == "" {
		sslMode = "require"
	}

	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=%s",
		dbHost, dbUser, dbPassword, dbName, dbPort, sslMode)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("连接 Supabase PostgreSQL 失败: %v", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("获取数据库连接对象失败: %v", err)
	}

	sqlDB.SetMaxIdleConns(1)
	sqlDB.SetMaxOpenConns(2)
	sqlDB.SetConnMaxLifetime(5 * time.Minute)

	t.Cleanup(func() {
		_ = sqlDB.Close()
	})

	return db
}

// newTestTransaction 返回一个会在测试结束时回滚的事务，避免污染实际数据。
func newTestTransaction(t *testing.T, db *gorm.DB) *gorm.DB {
	t.Helper()
	tx := db.Begin()
	if tx.Error != nil {
		t.Fatalf("开启事务失败: %v", tx.Error)
	}

	createTables := []string{
		`CREATE TEMP TABLE audit_logs (
			id BIGSERIAL PRIMARY KEY,
			user_id BIGINT,
			action TEXT NOT NULL,
			resource TEXT,
			resource_id TEXT,
			method TEXT,
			path TEXT,
			ip_address TEXT,
			user_agent TEXT,
			status TEXT,
			status_code INTEGER,
			error_msg TEXT,
			duration BIGINT,
			details TEXT,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		) ON COMMIT DROP`,
		`CREATE TEMP TABLE users (
			id BIGSERIAL PRIMARY KEY,
			username TEXT,
			email TEXT,
			password TEXT,
			full_name TEXT,
			company_id TEXT,
			active BOOLEAN,
			email_verified BOOLEAN,
			email_verified_at TIMESTAMPTZ,
			created_at TIMESTAMPTZ DEFAULT NOW(),
			updated_at TIMESTAMPTZ DEFAULT NOW()
		) ON COMMIT DROP`,
	}

	for _, stmt := range createTables {
		if err := tx.Exec(stmt).Error; err != nil {
			_ = tx.Rollback()
			t.Fatalf("创建临时测试表失败: %v", err)
		}
	}

	t.Cleanup(func() {
		_ = tx.Rollback()
	})

	return tx
}

func TestAuditService_LogActionAndGet(t *testing.T) {
	db := setupTestDB(t)
	tx := newTestTransaction(t, db)
	service := NewAuditService(tx)

	resourceID := fmt.Sprintf("test-resource-%d", time.Now().UnixNano())
	ctx := context.Background()

	err := service.LogAction(ctx, models.CreateAuditLogParams{
		Action:     models.ActionSystemStart,
		Resource:   "system",
		ResourceID: &resourceID,
		Method:     "POST",
		Path:       "/internal/test",
		IPAddress:  "127.0.0.1",
		UserAgent:  "go-test-suite",
		Status:     models.StatusSuccess,
		StatusCode: 201,
		Duration:   42,
		Details: &models.LogDetails{
			Custom: map[string]interface{}{
				"test_case": "TestAuditService_LogActionAndGet",
			},
		},
	})
	if err != nil {
		t.Fatalf("写入审计日志失败: %v", err)
	}

	logs, total, err := service.GetAuditLogs(nil, 10, 0, string(models.ActionSystemStart), "", nil, nil)
	if err != nil {
		t.Fatalf("查询审计日志失败: %v", err)
	}
	if total == 0 {
		t.Fatalf("预期至少检索到 1 条日志，实际为 %d", total)
	}

	found := false
	for _, logEntry := range logs {
		if logEntry.ResourceID != nil && *logEntry.ResourceID == resourceID {
			found = true
			if logEntry.Action != string(models.ActionSystemStart) {
				t.Errorf("日志 Action 不符，期望 %s，实际 %s", models.ActionSystemStart, logEntry.Action)
			}
			if logEntry.Status != string(models.StatusSuccess) {
				t.Errorf("日志 Status 不符，期望 SUCCESS，实际 %s", logEntry.Status)
			}
			details := logEntry.GetParsedDetails()
			if details == nil || details.Custom["test_case"] != "TestAuditService_LogActionAndGet" {
				t.Errorf("日志详情未包含预期字段: %+v", details)
			}
			break
		}
	}

	if !found {
		t.Fatalf("未在最新日志中找到资源 ID 为 %s 的记录", resourceID)
	}
}

func TestAuditService_GetUserActivityStats(t *testing.T) {
	db := setupTestDB(t)
	tx := newTestTransaction(t, db)
	service := NewAuditService(tx)

	usernameSuffix := time.Now().UnixNano()
	user := &models.User{
		Username: fmt.Sprintf("audit_user_%d", usernameSuffix),
		Email:    fmt.Sprintf("audit_user_%d@example.com", usernameSuffix),
		Active:   true,
	}
	if err := user.SetPassword("SupabaseTest123!"); err != nil {
		t.Fatalf("设置用户密码失败: %v", err)
	}

	if err := tx.Create(user).Error; err != nil {
		t.Fatalf("创建测试用户失败: %v", err)
	}

	makeLog := func(action models.ActionType) {
		if err := service.LogAction(context.Background(), models.CreateAuditLogParams{
			UserID:   &user.ID,
			Action:   action,
			Resource: "integration-test",
			Path:     "/integration",
			Method:   "GET",
			Status:   models.StatusSuccess,
		}); err != nil {
			t.Fatalf("写入审计日志失败: %v", err)
		}
	}

	makeLog(models.ActionLogin)
	makeLog(models.ActionUploadFile)

	stats, err := service.GetUserActivityStats(user.ID, 7)
	if err != nil {
		t.Fatalf("获取用户行为统计失败: %v", err)
	}

	if stats[string(models.ActionLogin)] != 1 {
		t.Errorf("Action LOGIN 计数不正确，期望 1，实际 %d", stats[string(models.ActionLogin)])
	}
	if stats[string(models.ActionUploadFile)] != 1 {
		t.Errorf("Action UPLOAD_FILE 计数不正确，期望 1，实际 %d", stats[string(models.ActionUploadFile)])
	}
}

func TestAuditService_CleanupOldLogs(t *testing.T) {
	db := setupTestDB(t)
	tx := newTestTransaction(t, db)
	service := NewAuditService(tx)

	oldLog := &models.AuditLog{
		Action:     string(models.ActionSystemInfo),
		Resource:   "system",
		Status:     string(models.StatusSuccess),
		StatusCode: 200,
		IPAddress:  "127.0.0.1",
		CreatedAt:  time.Now().AddDate(0, 0, -45),
	}
	if err := tx.Create(oldLog).Error; err != nil {
		t.Fatalf("创建历史日志失败: %v", err)
	}
	if err := tx.Model(oldLog).Update("created_at", oldLog.CreatedAt).Error; err != nil {
		t.Fatalf("更新历史日志时间失败: %v", err)
	}

	recentLog := &models.AuditLog{
		Action:     string(models.ActionSystemMetrics),
		Resource:   "system",
		Status:     string(models.StatusSuccess),
		StatusCode: 200,
		IPAddress:  "127.0.0.1",
		CreatedAt:  time.Now().Add(-12 * time.Hour),
	}
	if err := tx.Create(recentLog).Error; err != nil {
		t.Fatalf("创建近期日志失败: %v", err)
	}
	if err := tx.Model(recentLog).Update("created_at", recentLog.CreatedAt).Error; err != nil {
		t.Fatalf("更新近期日志时间失败: %v", err)
	}

	if err := service.CleanupOldLogs(30); err != nil {
		t.Fatalf("清理旧日志失败: %v", err)
	}

	var oldCount int64
	if err := tx.Model(&models.AuditLog{}).Where("id = ?", oldLog.ID).Count(&oldCount).Error; err != nil {
		t.Fatalf("统计旧日志状态失败: %v", err)
	}
	if oldCount != 0 {
		t.Errorf("旧日志未被删除，剩余计数: %d", oldCount)
	}

	var recentCount int64
	if err := tx.Model(&models.AuditLog{}).Where("id = ?", recentLog.ID).Count(&recentCount).Error; err != nil {
		t.Fatalf("统计近期日志状态失败: %v", err)
	}
	if recentCount != 1 {
		t.Errorf("近期日志被误删，剩余计数: %d", recentCount)
	}
}
