package api

import (
	"context"
	"errors"
	"log"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"siapp/internal/models"
	"siapp/internal/service/storage"
)

const (
	invoicePendingCleanupStatus = "pending_cleanup"
	cleanupTaskPending          = "pending"
	cleanupTaskRunning          = "running"
	cleanupLeaseDuration        = 5 * time.Minute
)

func (h *Handler) queueInvoiceSysFileCleanup(ctx context.Context, file *models.SysFile) bool {
	if file.StorageConfigID == nil {
		return false
	}
	return h.handleInvoiceUploadFailure(*file.StorageConfigID, file.Path, &file.ID, nil, errors.New("invoice creation failed"))
}

func (h *Handler) queueInvoiceCleanup(configID uint, path string, fileID *uint, cause error) (uint, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	task := models.InvoiceFileCleanupTask{StorageConfigID: configID, ObjectPath: path, SysFileID: fileID, Status: cleanupTaskPending, LastError: cause.Error()}
	if err := h.db.WithContext(ctx).Create(&task).Error; err != nil {
		return 0, err
	}
	return task.ID, nil
}

func (h *Handler) handleInvoiceUploadFailure(configID uint, path string, fileID *uint, driver storage.Driver, cause error) bool {
	taskID, err := h.queueInvoiceCleanup(configID, path, fileID, cause)
	if err == nil {
		return true
	}
	if driver == nil && storage.GlobalManager != nil {
		driver, _ = storage.GlobalManager.GetDriver(configID)
	}
	if driver != nil {
		if deleteErr := driver.Delete(context.Background(), path); deleteErr == nil {
			h.deleteOrLogOrphanedSysFile(fileID, configID)
			return false
		} else {
			log.Printf("发票附件清理入队失败: task_id=0 storage_id=%d file_id=%v delete_failed=%v db_error=%v", configID, fileID, deleteErr, err)
			return false
		}
	}
	log.Printf("发票附件清理入队失败: task_id=%d storage_id=%d file_id=%v storage_unavailable db_error=%v", taskID, configID, fileID, err)
	return false
}

func (h *Handler) deleteOrLogOrphanedSysFile(fileID *uint, configID uint) {
	if fileID == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	result := h.db.WithContext(ctx).Delete(&models.SysFile{}, *fileID)
	if result.Error != nil || result.RowsAffected != 1 {
		log.Printf("发票附件物理删除后元数据清理失败: storage_id=%d file_id=%d err=%v", configID, *fileID, result.Error)
	}
}

// StartInvoiceFileCleanup 启动时执行有限批次，并以可取消定时器低频重试。
func (h *Handler) StartInvoiceFileCleanup(ctx context.Context, limit int, interval time.Duration) {
	h.runInvoiceFileCleanup(ctx, limit)
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				h.runInvoiceFileCleanup(ctx, limit)
			}
		}
	}()
}

func (h *Handler) runInvoiceFileCleanup(ctx context.Context, limit int) {
	if limit < 1 {
		limit = 100
	}
	h.migrateLegacyInvoiceCleanup(ctx, limit)
	for i := 0; i < limit; i++ {
		task, ok := h.claimInvoiceCleanupTask(ctx)
		if !ok {
			return
		}
		h.finishInvoiceCleanupTask(ctx, task)
	}
}

func (h *Handler) migrateLegacyInvoiceCleanup(ctx context.Context, limit int) {
	var files []models.SysFile
	h.db.WithContext(ctx).Where("migration_status = ?", invoicePendingCleanupStatus).Limit(limit).Find(&files)
	for i := range files {
		if h.queueInvoiceSysFileCleanup(ctx, &files[i]) {
			h.db.WithContext(ctx).Model(&files[i]).Update("migration_status", "cleanup_queued")
		}
	}
}

func (h *Handler) claimInvoiceCleanupTask(ctx context.Context) (*models.InvoiceFileCleanupTask, bool) {
	now := time.Now()
	var candidates []models.InvoiceFileCleanupTask
	if h.db.WithContext(ctx).Where("status = ? OR (status = ? AND locked_until < ?)", cleanupTaskPending, cleanupTaskRunning, now).Order("id").Limit(10).Find(&candidates).Error != nil {
		return nil, false
	}
	for i := range candidates {
		until := now.Add(cleanupLeaseDuration)
		token := uuid.NewString()
		result := h.db.WithContext(ctx).Model(&models.InvoiceFileCleanupTask{}).Where("id = ? AND (status = ? OR (status = ? AND locked_until < ?))", candidates[i].ID, cleanupTaskPending, cleanupTaskRunning, now).Updates(map[string]any{"status": cleanupTaskRunning, "owner_token": token, "locked_until": until, "attempts": gorm.Expr("attempts + ?", 1)})
		if result.Error == nil && result.RowsAffected == 1 {
			candidates[i].LockedUntil = &until
			candidates[i].OwnerToken = token
			return &candidates[i], true
		}
	}
	return nil, false
}

func (h *Handler) finishInvoiceCleanupTask(ctx context.Context, task *models.InvoiceFileCleanupTask) {
	err := h.deleteInvoiceCleanupObject(ctx, task)
	if err == nil {
		result := h.db.WithContext(ctx).Where("id = ? AND owner_token = ? AND status = ?", task.ID, task.OwnerToken, cleanupTaskRunning).Delete(&models.InvoiceFileCleanupTask{})
		if result.Error != nil || result.RowsAffected != 1 {
			log.Printf("发票清理任务完成更新失败: task_id=%d storage_id=%d err=%v", task.ID, task.StorageConfigID, result.Error)
		}
		return
	}
	result := h.db.WithContext(ctx).Model(&models.InvoiceFileCleanupTask{}).Where("id = ? AND owner_token = ? AND status = ?", task.ID, task.OwnerToken, cleanupTaskRunning).Updates(map[string]any{"status": cleanupTaskPending, "owner_token": "", "locked_until": nil, "last_error": err.Error()})
	if result.Error != nil || result.RowsAffected != 1 {
		log.Printf("发票清理任务失败更新失败: task_id=%d storage_id=%d err=%v", task.ID, task.StorageConfigID, result.Error)
	}
}

func (h *Handler) deleteInvoiceCleanupObject(ctx context.Context, task *models.InvoiceFileCleanupTask) error {
	if storage.GlobalManager == nil {
		return errors.New("storage unavailable")
	}
	driver, err := storage.GlobalManager.GetDriver(task.StorageConfigID)
	if err != nil {
		return err
	}
	exists, err := driver.Exists(ctx, task.ObjectPath)
	if err != nil {
		return err
	}
	if exists {
		if err := driver.Delete(ctx, task.ObjectPath); err != nil {
			return err
		}
	}
	if task.SysFileID != nil {
		return h.db.WithContext(ctx).Delete(&models.SysFile{}, *task.SysFileID).Error
	}
	return nil
}
