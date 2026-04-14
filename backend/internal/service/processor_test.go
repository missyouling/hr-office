package service

import (
	"testing"
	"time"

	"gorm.io/gorm"

	"siapp/internal/models"
)

func createProcessorTempTables(t *testing.T, tx *gorm.DB) {
	t.Helper()

	statements := []string{
		`CREATE TEMP TABLE periods (
			id BIGSERIAL PRIMARY KEY,
			user_id BIGINT,
			year_month TEXT,
			status TEXT,
			created_at TIMESTAMPTZ DEFAULT NOW(),
			updated_at TIMESTAMPTZ DEFAULT NOW()
		) ON COMMIT DROP`,
		`CREATE TEMP TABLE raw_records (
			id BIGSERIAL PRIMARY KEY,
			user_id BIGINT,
			period_id BIGINT NOT NULL,
			source_file_id BIGINT,
			sequence INTEGER,
			name TEXT,
			id_type TEXT,
			id_number TEXT,
			department TEXT,
			pay_salary DOUBLE PRECISION,
			pay_base DOUBLE PRECISION,
			rate_text TEXT,
			amount_due DOUBLE PRECISION,
			amount_adjust DOUBLE PRECISION,
			person_code TEXT,
			scheme TEXT,
			part TEXT,
			file_type TEXT,
			created_at TIMESTAMPTZ DEFAULT NOW(),
			updated_at TIMESTAMPTZ DEFAULT NOW()
		) ON COMMIT DROP`,
		`CREATE TEMP TABLE roster_entries (
			id BIGSERIAL PRIMARY KEY,
			user_id BIGINT,
			period_id BIGINT NOT NULL,
			name TEXT,
			id_number TEXT,
			department TEXT,
			title TEXT,
			remarks TEXT,
			created_at TIMESTAMPTZ DEFAULT NOW(),
			updated_at TIMESTAMPTZ DEFAULT NOW()
		) ON COMMIT DROP`,
		`CREATE TEMP TABLE period_summaries (
			id BIGSERIAL PRIMARY KEY,
			user_id BIGINT,
			period_id BIGINT NOT NULL,
			scheme TEXT,
			part TEXT,
			headcount INTEGER,
			base_total DOUBLE PRECISION,
			amount_total DOUBLE PRECISION,
			is_adjustment BOOLEAN DEFAULT FALSE,
			created_at TIMESTAMPTZ DEFAULT NOW(),
			updated_at TIMESTAMPTZ DEFAULT NOW()
		) ON COMMIT DROP`,
		`CREATE TEMP TABLE personal_charges (
			id BIGSERIAL PRIMARY KEY,
			user_id BIGINT,
			period_id BIGINT NOT NULL,
			name TEXT,
			id_number TEXT,
			department TEXT,
			base DOUBLE PRECISION,
			pension DOUBLE PRECISION,
			medical_maternity DOUBLE PRECISION,
			serious_illness DOUBLE PRECISION,
			unemployment DOUBLE PRECISION,
			subtotal DOUBLE PRECISION,
			is_adjustment BOOLEAN DEFAULT FALSE,
			created_at TIMESTAMPTZ DEFAULT NOW(),
			updated_at TIMESTAMPTZ DEFAULT NOW()
		) ON COMMIT DROP`,
		`CREATE TEMP TABLE unit_charges (
			id BIGSERIAL PRIMARY KEY,
			user_id BIGINT,
			period_id BIGINT NOT NULL,
			name TEXT,
			id_number TEXT,
			department TEXT,
			base DOUBLE PRECISION,
			pension DOUBLE PRECISION,
			medical_maternity DOUBLE PRECISION,
			serious_illness DOUBLE PRECISION,
			injury DOUBLE PRECISION,
			unemployment DOUBLE PRECISION,
			subtotal DOUBLE PRECISION,
			is_adjustment BOOLEAN DEFAULT FALSE,
			created_at TIMESTAMPTZ DEFAULT NOW(),
			updated_at TIMESTAMPTZ DEFAULT NOW()
		) ON COMMIT DROP`,
		`CREATE TEMP TABLE source_files (
			id BIGSERIAL PRIMARY KEY,
			user_id BIGINT,
			period_id BIGINT NOT NULL,
			file_name TEXT,
			stored_path TEXT,
			scheme TEXT,
			part TEXT,
			file_type TEXT,
			rows INTEGER,
			status TEXT,
			uploaded_at TIMESTAMPTZ DEFAULT NOW(),
			original_name TEXT,
			created_at TIMESTAMPTZ DEFAULT NOW(),
			updated_at TIMESTAMPTZ DEFAULT NOW()
		) ON COMMIT DROP`,
	}

	for _, stmt := range statements {
		if err := tx.Exec(stmt).Error; err != nil {
			_ = tx.Rollback()
			t.Fatalf("创建临时处理流程表失败: %v", err)
		}
	}
}

func TestProcessor_ProcessPeriod_Success(t *testing.T) {
	db := setupTestDB(t)
	tx := newTestTransaction(t, db)
	createProcessorTempTables(t, tx)

	now := time.Now()
	period := models.Period{
		YearMonth: "2024-01",
		Status:    "draft",
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := tx.Create(&period).Error; err != nil {
		t.Fatalf("创建期间失败: %v", err)
	}

	var records []models.RawRecord
	records = append(records,
		models.RawRecord{
			PeriodID:   period.ID,
			Sequence:   1,
			Name:       "张三",
			IDNumber:   "ID123",
			Department: "人事部",
			PayBase:    5000,
			AmountDue:  500,
			Scheme:     models.SchemePension,
			Part:       models.PartPersonal,
			FileType:   models.FileTypeNormal,
			CreatedAt:  now,
			UpdatedAt:  now,
		},
		models.RawRecord{
			PeriodID:   period.ID,
			Sequence:   2,
			Name:       "张三",
			IDNumber:   "ID123",
			Department: "人事部",
			PayBase:    5000,
			AmountDue:  400,
			Scheme:     models.SchemeMedical,
			Part:       models.PartPersonal,
			FileType:   models.FileTypeNormal,
			CreatedAt:  now,
			UpdatedAt:  now,
		},
		models.RawRecord{
			PeriodID:   period.ID,
			Sequence:   3,
			Name:       "张三",
			IDNumber:   "ID123",
			Department: "人事部",
			PayBase:    5000,
			AmountDue:  50,
			Scheme:     models.SchemeSeriousIllness,
			Part:       models.PartPersonal,
			FileType:   models.FileTypeNormal,
			CreatedAt:  now,
			UpdatedAt:  now,
		},
		models.RawRecord{
			PeriodID:   period.ID,
			Sequence:   4,
			Name:       "张三",
			IDNumber:   "ID123",
			Department: "人事部",
			PayBase:    5000,
			AmountDue:  30,
			Scheme:     models.SchemeUnemployment,
			Part:       models.PartPersonal,
			FileType:   models.FileTypeNormal,
			CreatedAt:  now,
			UpdatedAt:  now,
		},
		models.RawRecord{
			PeriodID:   period.ID,
			Sequence:   5,
			Name:       "张三",
			IDNumber:   "ID123",
			Department: "人事部",
			PayBase:    6000,
			AmountDue:  1000,
			Scheme:     models.SchemePension,
			Part:       models.PartUnit,
			FileType:   models.FileTypeNormal,
			CreatedAt:  now,
			UpdatedAt:  now,
		},
		models.RawRecord{
			PeriodID:   period.ID,
			Sequence:   6,
			Name:       "张三",
			IDNumber:   "ID123",
			Department: "人事部",
			PayBase:    6000,
			AmountDue:  800,
			Scheme:     models.SchemeMedical,
			Part:       models.PartUnit,
			FileType:   models.FileTypeNormal,
			CreatedAt:  now,
			UpdatedAt:  now,
		},
		models.RawRecord{
			PeriodID:   period.ID,
			Sequence:   7,
			Name:       "张三",
			IDNumber:   "ID123",
			Department: "人事部",
			PayBase:    6000,
			AmountDue:  100,
			Scheme:     models.SchemeSeriousIllness,
			Part:       models.PartUnit,
			FileType:   models.FileTypeNormal,
			CreatedAt:  now,
			UpdatedAt:  now,
		},
		models.RawRecord{
			PeriodID:   period.ID,
			Sequence:   8,
			Name:       "张三",
			IDNumber:   "ID123",
			Department: "人事部",
			PayBase:    6000,
			AmountDue:  60,
			Scheme:     models.SchemeUnemployment,
			Part:       models.PartUnit,
			FileType:   models.FileTypeNormal,
			CreatedAt:  now,
			UpdatedAt:  now,
		},
		models.RawRecord{
			PeriodID:   period.ID,
			Sequence:   9,
			Name:       "张三",
			IDNumber:   "ID123",
			Department: "人事部",
			PayBase:    6000,
			AmountDue:  70,
			Scheme:     models.SchemeInjury,
			Part:       models.PartUnit,
			FileType:   models.FileTypeNormal,
			CreatedAt:  now,
			UpdatedAt:  now,
		},
	)

	if err := tx.Create(&records).Error; err != nil {
		t.Fatalf("插入原始记录失败: %v", err)
	}

	roster := models.RosterEntry{
		PeriodID:   period.ID,
		Name:       "张三",
		IDNumber:   "ID123",
		Department: "人事部",
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	if err := tx.Create(&roster).Error; err != nil {
		t.Fatalf("插入花名册失败: %v", err)
	}

	processor := NewProcessor(tx)
	output, err := processor.ProcessPeriod(period.ID)
	if err != nil {
		t.Fatalf("处理期间失败: %v", err)
	}

	if len(output.Personal) != 1 {
		t.Fatalf("个人扣款条目数量不符，期望 1，实际 %d", len(output.Personal))
	}
	personal := output.Personal[0]
	if personal.Subtotal != 980 {
		t.Errorf("个人扣款汇总不符，期望 980，实际 %.2f", personal.Subtotal)
	}
	if personal.Base != 5000 {
		t.Errorf("个人缴费基数不符，期望 5000，实际 %.2f", personal.Base)
	}

	if len(output.Unit) != 1 {
		t.Fatalf("单位扣款条目数量不符，期望 1，实际 %d", len(output.Unit))
	}
	unit := output.Unit[0]
	if unit.Subtotal != 2030 {
		t.Errorf("单位扣款汇总不符，期望 2030，实际 %.2f", unit.Subtotal)
	}
	if unit.Base != 6000 {
		t.Errorf("单位缴费基数不符，期望 6000，实际 %.2f", unit.Base)
	}

	var updatedPeriod models.Period
	if err := tx.First(&updatedPeriod, period.ID).Error; err != nil {
		t.Fatalf("查询更新后的期间失败: %v", err)
	}
	if updatedPeriod.Status != "processed" {
		t.Errorf("期间状态未更新为 processed，实际为 %s", updatedPeriod.Status)
	}
}

func TestProcessor_ProcessPeriod_MissingScheme(t *testing.T) {
	db := setupTestDB(t)
	tx := newTestTransaction(t, db)
	createProcessorTempTables(t, tx)

	now := time.Now()
	period := models.Period{
		YearMonth: "2024-02",
		Status:    "draft",
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := tx.Create(&period).Error; err != nil {
		t.Fatalf("创建期间失败: %v", err)
	}

	records := []models.RawRecord{
		{
			PeriodID:   period.ID,
			Sequence:   1,
			Name:       "李四",
			IDNumber:   "ID999",
			Department: "财务部",
			PayBase:    4800,
			AmountDue:  480,
			Scheme:     models.SchemePension,
			Part:       models.PartPersonal,
			FileType:   models.FileTypeNormal,
			CreatedAt:  now,
			UpdatedAt:  now,
		},
		{
			PeriodID:   period.ID,
			Sequence:   2,
			Name:       "李四",
			IDNumber:   "ID999",
			Department: "财务部",
			PayBase:    4800,
			AmountDue:  380,
			Scheme:     models.SchemeMedical,
			Part:       models.PartPersonal,
			FileType:   models.FileTypeNormal,
			CreatedAt:  now,
			UpdatedAt:  now,
		},
		{
			PeriodID:   period.ID,
			Sequence:   3,
			Name:       "李四",
			IDNumber:   "ID999",
			Department: "财务部",
			PayBase:    4800,
			AmountDue:  25,
			Scheme:     models.SchemeUnemployment,
			Part:       models.PartPersonal,
			FileType:   models.FileTypeNormal,
			CreatedAt:  now,
			UpdatedAt:  now,
		},
		{
			PeriodID:   period.ID,
			Sequence:   4,
			Name:       "李四",
			IDNumber:   "ID999",
			Department: "财务部",
			PayBase:    5800,
			AmountDue:  900,
			Scheme:     models.SchemePension,
			Part:       models.PartUnit,
			FileType:   models.FileTypeNormal,
			CreatedAt:  now,
			UpdatedAt:  now,
		},
		{
			PeriodID:   period.ID,
			Sequence:   5,
			Name:       "李四",
			IDNumber:   "ID999",
			Department: "财务部",
			PayBase:    5800,
			AmountDue:  700,
			Scheme:     models.SchemeMedical,
			Part:       models.PartUnit,
			FileType:   models.FileTypeNormal,
			CreatedAt:  now,
			UpdatedAt:  now,
		},
		{
			PeriodID:   period.ID,
			Sequence:   6,
			Name:       "李四",
			IDNumber:   "ID999",
			Department: "财务部",
			PayBase:    5800,
			AmountDue:  55,
			Scheme:     models.SchemeUnemployment,
			Part:       models.PartUnit,
			FileType:   models.FileTypeNormal,
			CreatedAt:  now,
			UpdatedAt:  now,
		},
		{
			PeriodID:   period.ID,
			Sequence:   7,
			Name:       "李四",
			IDNumber:   "ID999",
			Department: "财务部",
			PayBase:    5800,
			AmountDue:  65,
			Scheme:     models.SchemeInjury,
			Part:       models.PartUnit,
			FileType:   models.FileTypeNormal,
			CreatedAt:  now,
			UpdatedAt:  now,
		},
	}

	if err := tx.Create(&records).Error; err != nil {
		t.Fatalf("插入原始记录失败: %v", err)
	}

	processor := NewProcessor(tx)
	if _, err := processor.ProcessPeriod(period.ID); err == nil {
		t.Fatalf("缺少险种时应返回错误")
	}
}
