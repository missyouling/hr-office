package api

import (
	"context"
	"log"
	"time"

	"siapp/internal/models"
)

// adminContractWorkerTimeZone 行政合同到期扫描定时任务时区（Asia/Shanghai）。
const adminContractWorkerTimeZone = "Asia/Shanghai"

// StartAdminContractExpiryWorker 启动行政合同到期扫描定时任务：
// 每日 Asia/Shanghai 02:00 将 active 且 end_date 早于当日的合同标记为 expired；
// 不联动其他模块；批量更新天然幂等，重复运行无副作用。
// 复用既有 onboarding/regularization/contract worker 的启动模式（main 中由 Handler 启动）。
func (h *Handler) StartAdminContractExpiryWorker(ctx context.Context) {
	go func() {
		loc, err := time.LoadLocation(adminContractWorkerTimeZone)
		if err != nil {
			log.Printf("[admin-contract-worker] 加载时区失败: %v", err)
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
				h.runAdminContractExpiryOnce(loc, time.Now())
			}
		}
	}()
}

// runAdminContractExpiryOnce 执行一次全租户到期扫描（幂等，可安全重复执行）。
// now 由调用方注入以便确定性测试；时区统一转为 Asia/Shanghai 取"当日"。
func (h *Handler) runAdminContractExpiryOnce(loc *time.Location, now time.Time) {
	today := now.In(loc).Format("2006-01-02")
	affected, err := h.markAllExpiredAdminContracts(today)
	if err != nil {
		log.Printf("[admin-contract-worker] %s 到期扫描失败: %v", today, err)
		return
	}
	log.Printf("[admin-contract-worker] %s 到期扫描完成: %d 份合同标记为 expired", today, affected)
}

// markAllExpiredAdminContracts 跨租户标记到期行政合同（worker 专用）：
// active 且 end_date < today → expired，记录到期标记时间；不联动其他模块。
// 返回本次受影响行数（仅 active → expired 的合同）。
func (h *Handler) markAllExpiredAdminContracts(today string) (int64, error) {
	result := h.db.Model(&models.AdminContract{}).
		Where("status = ? AND end_date < ?", models.AdminContractStatusActive, today).
		Updates(map[string]any{
			"status":     models.AdminContractStatusExpired,
			"expired_at": time.Now(),
		})
	if result.Error != nil {
		return 0, result.Error
	}
	return result.RowsAffected, nil
}
