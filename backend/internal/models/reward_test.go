package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestIsValidRewardType 类型常量校验：reward/punishment 通过，非法类型拒绝。
func TestIsValidRewardType(t *testing.T) {
	assert.True(t, IsValidRewardType(RewardTypeReward))
	assert.True(t, IsValidRewardType(RewardTypePunishment))
	assert.False(t, IsValidRewardType(""))
	assert.False(t, IsValidRewardType("bonus"))
	assert.False(t, IsValidRewardType("penalty"))
}

// TestIsValidRewardStatus 状态常量校验：合法三态通过，非法状态拒绝。
func TestIsValidRewardStatus(t *testing.T) {
	assert.True(t, IsValidRewardStatus(RewardStatusDraft))
	assert.True(t, IsValidRewardStatus(RewardStatusEffective))
	assert.True(t, IsValidRewardStatus(RewardStatusVoided))
	assert.False(t, IsValidRewardStatus(""))
	assert.False(t, IsValidRewardStatus("pending"))
	assert.False(t, IsValidRewardStatus("active"))
}

// TestRewardRecordValidate 字段约束校验：正常通过，异常边界逐项拒绝。
func TestRewardRecordValidate(t *testing.T) {
	valid := RewardRecord{
		EmployeeID:   1,
		RecordType:   RewardTypeReward,
		OccurredDate: "2026-08-01",
		Reason:       "季度优秀员工",
		Level:        "嘉奖",
		Status:       RewardStatusDraft,
	}
	require.NoError(t, valid.Validate(), "合法奖惩记录应通过校验")

	cases := []struct {
		name   string
		mutate func(*RewardRecord)
	}{
		{"状态非法", func(r *RewardRecord) { r.Status = "pending" }},
		{"类型非法", func(r *RewardRecord) { r.RecordType = "bonus" }},
		{"员工缺失", func(r *RewardRecord) { r.EmployeeID = 0 }},
		{"发生日期缺失", func(r *RewardRecord) { r.OccurredDate = "" }},
		{"发生日期格式错误", func(r *RewardRecord) { r.OccurredDate = "2026/08/01" }},
		{"发生日期非法值", func(r *RewardRecord) { r.OccurredDate = "2026-13-01" }},
		{"事由缺失", func(r *RewardRecord) { r.Reason = " " }},
		{"等级缺失", func(r *RewardRecord) { r.Level = "" }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := valid
			tc.mutate(&r)
			assert.Error(t, r.Validate(), "非法奖惩记录应校验失败")
		})
	}
}

// TestRewardRecordOptionalFields 选填字段（分值/金额/经办人/备注/文档）不影响校验通过。
func TestRewardRecordOptionalFields(t *testing.T) {
	score := 5.0
	amount := 1000.0
	docID := uint(9)
	r := RewardRecord{
		EmployeeID:   2,
		RecordType:   RewardTypePunishment,
		OccurredDate: "2026-08-02",
		Reason:       "迟到三次",
		Level:        "警告",
		Score:        &score,
		Amount:       &amount,
		Owner:        "张三",
		DocumentID:   &docID,
		Remarks:      "已面谈",
		Status:       RewardStatusDraft,
	}
	require.NoError(t, r.Validate(), "含选填字段的合法记录应通过校验")
}
