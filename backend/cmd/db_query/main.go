package main

import (
	"fmt"
	"log"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	dsn := "host=localhost port=5432 user=koujiang password=koujiang dbname=siapp sslmode=disable"
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("连接数据库失败: %v", err)
	}

	fmt.Println("=== 修复空 IP 地址 ===\n")

	// 修复所有空 IP
	result := db.Table("audit_logs").
		Where("ip_address IS NULL OR ip_address = ''").
		Update("ip_address", "127.0.0.1")

	if result.Error != nil {
		log.Fatalf("修复失败: %v", result.Error)
	}

	fmt.Printf("✅ 已修复 %d 条空 IP 的审计日志\n\n", result.RowsAffected)

	// 验证修复
	var emptyIPCount int64
	db.Table("audit_logs").
		Where("ip_address IS NULL OR ip_address = ''").
		Count(&emptyIPCount)
	fmt.Printf("修复后剩余空 IP 数量: %d\n\n", emptyIPCount)

	// 查看修复后的样本
	type AuditLog struct {
		ID        uint
		Action    string
		IPAddress string
	}
	var logs []AuditLog
	db.Table("audit_logs").
		Select("id, action, ip_address").
		Where("action = 'SYSTEM_START'").
		Limit(5).
		Scan(&logs)

	fmt.Printf("修复后的 SYSTEM_START 日志样本:\n")
	for i, log := range logs {
		fmt.Printf("  %d. ID=%d IP=%s\n", i+1, log.ID, log.IPAddress)
	}
}
