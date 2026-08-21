package service

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"siapp/internal/models"
)

// TestImportOnboardingRecordsRechecksEmployeeConflict 验证事务内复查员工冲突并整批回滚。
func TestImportOnboardingRecordsRechecksEmployeeConflict(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&models.Employee{}, &models.OnboardingRecord{}))
	require.NoError(t, db.Create(&models.Employee{UserID: 1, Name: "已有员工", IDNumber: "110101199001011234", Status: "active"}).Error)

	rows := []OnboardingImportRow{
		{Name: "合法行", IDNumber: "110101199002022345", PlannedHireDate: "2026-09-01"},
		{Name: "冲突行", IDNumber: "110101199001011234", PlannedHireDate: "2026-09-01"},
	}
	_, err = ImportOnboardingRecords(db, 1, rows)
	assert.True(t, errors.Is(err, ErrOnboardingEmployeeConflict))

	var count int64
	require.NoError(t, db.Model(&models.OnboardingRecord{}).Count(&count).Error)
	assert.Zero(t, count, "任一冲突应整批回滚")
}
