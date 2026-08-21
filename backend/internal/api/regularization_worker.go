package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strconv"
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"

	"siapp/internal/models"
)

// regularizationWorkerTimeZone 转正 worker 时区。
const regularizationWorkerTimeZone = "Asia/Shanghai"

// ErrRegularizationEffectFailed 表示转正生效失败。
var ErrRegularizationEffectFailed = errors.New("转正生效失败")

// StartRegularizationWorker 启动转正定时任务（每日 Asia/Shanghai 02:00）。
func (h *Handler) StartRegularizationWorker(ctx context.Context) {
	go func() {
		loc, err := time.LoadLocation(regularizationWorkerTimeZone)
		if err != nil {
			log.Printf("[regularization-worker] 加载时区失败: %v", err)
			return
		}
		for {
			now := time.Now().In(loc)
			next := time.Date(now.Year(), now.Month(), now.Day(), 2, 0, 0, 0, loc)
			if !now.Before(next) {
				next = next.AddDate(0, 0, 1)
			}
			timer := time.NewTimer(next.Sub(now))
			select {
			case <-ctx.Done():
				timer.Stop()
				return
			case <-timer.C:
				h.runRegularizationWorkerOnce(loc, time.Now())
			}
		}
	}()
}

// runRegularizationWorkerOnce 执行一次转正扫描（同日幂等，失败不重试）。
func (h *Handler) runRegularizationWorkerOnce(loc *time.Location, now time.Time) {
	today := now.In(loc).Format("2006-01-02")
	var existing models.RegularizationEffectRun
	if err := h.db.Where("run_date = ?", today).First(&existing).Error; err == nil {
		log.Printf("[regularization-worker] %s 已运行，跳过", today)
		return
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		log.Printf("[regularization-worker] 检查运行记录失败: %v", err)
		return
	}
	run := models.RegularizationEffectRun{RunDate: today, Status: models.RegularizationRunStatusSuccess}
	if err := h.db.Create(&run).Error; err != nil {
		log.Printf("[regularization-worker] 创建运行记录失败: %v", err)
		return
	}
	var records []models.RegularizationRecord
	if err := h.db.Where("status IN ? AND planned_regular_date = ? AND effect_attempted_at IS NULL",
		[]string{models.RegularizationStatusScheduled, models.RegularizationStatusPostponedScheduled}, today).Find(&records).Error; err != nil {
		run.Status = models.RegularizationRunStatusFailed
		run.ErrorMsg = "扫描转正记录失败: " + err.Error()
		_ = h.db.Save(&run).Error
		h.writeRegularizationWorkerLog("error", "扫描转正记录失败", err.Error())
		return
	}
	processed, failed := 0, 0
	for i := range records {
		if err := h.effectiveOneRegularization(&records[i]); err != nil {
			failed++
			h.writeRegularizationWorkerLog("error", "转正处理失败", "record_id="+strconv.FormatUint(uint64(records[i].ID), 10)+" 原因="+err.Error())
		} else {
			processed++
		}
	}
	run.Processed = processed
	run.Failed = failed
	if failed > 0 {
		run.Status = models.RegularizationRunStatusFailed
		run.ErrorMsg = strconv.Itoa(failed) + " 条记录处理失败"
	}
	if err := h.db.Save(&run).Error; err != nil {
		log.Printf("[regularization-worker] 更新运行记录失败: %v", err)
	}
	log.Printf("[regularization-worker] %s 运行完成: 成功 %d 失败 %d", today, processed, failed)
}

// effectiveOneRegularization 单条记录生效（同事务；成功 effective，失败 effect_failed + 唯一待办）。
// 失败时 effect_failed 与待办必须随事务提交，故用事务外标志变量记录失败，
// 事务提交后再返回 ErrRegularizationEffectFailed（绝不能在事务回调内返回，否则 GORM 会回滚）。
func (h *Handler) effectiveOneRegularization(record *models.RegularizationRecord) error {
	failed := false
	var failReason string
	err := h.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&models.RegularizationRecord{}).
			Where("id = ? AND status IN ? AND effect_attempted_at IS NULL", record.ID,
				[]string{models.RegularizationStatusScheduled, models.RegularizationStatusPostponedScheduled}).
			Updates(map[string]any{"effect_attempted_at": time.Now()}).Error; err != nil {
			return err
		}
		if err := h.applyRegularizationEffect(tx, record, record.PlannedRegularDate); err != nil {
			if updateErr := tx.Model(&models.RegularizationRecord{}).Where("id = ?", record.ID).Updates(map[string]any{
				"status":               models.RegularizationStatusEffectFailed,
				"effect_failed_reason": err.Error(),
			}).Error; updateErr != nil {
				return updateErr
			}
			if err := createRegularizationEffectTodo(tx, record, err.Error()); err != nil {
				return err
			}
			failReason = err.Error()
			failed = true
			return nil
		}
		return tx.Model(&models.RegularizationRecord{}).Where("id = ?", record.ID).Updates(map[string]any{
			"status":              models.RegularizationStatusEffective,
			"actual_regular_date": record.PlannedRegularDate,
		}).Error
	})
	if err != nil {
		return err
	}
	if failed {
		return fmt.Errorf("%w: %s", ErrRegularizationEffectFailed, failReason)
	}
	return nil
}

// createRegularizationEffectTodo 创建唯一生效异常待办。
func createRegularizationEffectTodo(tx *gorm.DB, record *models.RegularizationRecord, reason string) error {
	assigneeID := record.UserID
	if record.InitiatorHRUserID != nil {
		assigneeID = *record.InitiatorHRUserID
	}
	todo := models.WorkTodo{
		UserID:       record.UserID,
		BusinessType: "regularization_effect_exception",
		BusinessID:   record.ID,
		Title:        "转正自动生效失败：" + record.SnapshotName,
		Description:  "请人工排查并处理转正生效失败，原因：" + reason,
		Status:       models.WorkTodoStatusPending,
		AssigneeID:   &assigneeID,
	}
	if err := tx.Create(&todo).Error; err != nil && !isUniqueViolation(err) {
		return err
	}
	return nil
}

// writeRegularizationWorkerLog 写入系统日志。
func (h *Handler) writeRegularizationWorkerLog(level, message, detail string) {
	entry := SystemLog{Level: level, Source: "regularization-worker", Message: message}
	if detail != "" {
		if data, err := json.Marshal(map[string]string{"detail": detail}); err == nil {
			entry.Details = datatypes.JSON(data)
		}
	}
	if err := h.db.Create(&entry).Error; err != nil {
		log.Printf("[regularization-worker] 写日志失败: %v", err)
	}
}
