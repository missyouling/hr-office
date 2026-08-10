package service

import (
	"fmt"
	"math/rand"
	"time"

	"gorm.io/gorm"
)

// GenerateOrderNo 生成采购单号 BG-YYYYMMDD-NN
// 在事务中使用计数实现递增序号
func GenerateOrderNo(tx *gorm.DB) (string, error) {
	now := time.Now()
	dateKey := now.Format("20060102")
	prefix := fmt.Sprintf("BG-%s-", dateKey)

	var count int64
	if err := tx.Table("office_purchases").
		Where("order_no LIKE ?", prefix+"%").
		Count(&count).Error; err != nil {
		return "", err
	}
	seq := count + 1
	return fmt.Sprintf("%s%02d", prefix, seq), nil
}

// GenerateRequestNo 生成请款单号 PR-YYYYMMDD-XXXX
func GenerateRequestNo() string {
	now := time.Now()
	dateStr := now.Format("20060102")
	randBytes := make([]byte, 4)
	for i := range randBytes {
		randBytes[i] = byte(65 + rand.Intn(26)) // A-Z
	}
	return fmt.Sprintf("PR-%s-%s", dateStr, string(randBytes))
}
