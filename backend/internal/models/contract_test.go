package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestIsValidContractStatus 状态常量校验：合法四态通过，非法状态拒绝。
func TestIsValidContractStatus(t *testing.T) {
	assert.True(t, IsValidContractStatus(ContractStatusDraft))
	assert.True(t, IsValidContractStatus(ContractStatusActive))
	assert.True(t, IsValidContractStatus(ContractStatusExpired))
	assert.True(t, IsValidContractStatus(ContractStatusCancelled))
	assert.False(t, IsValidContractStatus(""))
	assert.False(t, IsValidContractStatus("pending"))
	assert.False(t, IsValidContractStatus("voided"))
}

// TestIsValidContractType 类型校验：仅固定期限通过，其他类型拒绝。
func TestIsValidContractType(t *testing.T) {
	assert.True(t, IsValidContractType(ContractTypeFixedTerm))
	assert.False(t, IsValidContractType(""))
	assert.False(t, IsValidContractType("open_ended"))
	assert.False(t, IsValidContractType("probation"))
}

// TestLaborContractValidate 字段约束校验：正常通过，异常边界逐项拒绝。
func TestLaborContractValidate(t *testing.T) {
	valid := LaborContract{
		ContractType: ContractTypeFixedTerm,
		Status:       ContractStatusDraft,
		StartDate:    "2026-01-01",
		EndDate:      "2028-12-31",
		TermMonths:   36,
	}
	require.NoError(t, valid.Validate(), "合法合同应通过校验")

	cases := []struct {
		name   string
		mutate func(*LaborContract)
	}{
		{"状态非法", func(c *LaborContract) { c.Status = "pending" }},
		{"类型非法", func(c *LaborContract) { c.ContractType = "open_ended" }},
		{"起始日缺失", func(c *LaborContract) { c.StartDate = "" }},
		{"到期日缺失", func(c *LaborContract) { c.EndDate = "" }},
		{"起始日格式错误", func(c *LaborContract) { c.StartDate = "2026/01/01" }},
		{"到期日格式错误", func(c *LaborContract) { c.EndDate = "2028-13-01" }},
		{"期限非正数", func(c *LaborContract) { c.TermMonths = 0 }},
		{"期限负数", func(c *LaborContract) { c.TermMonths = -1 }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := valid
			tc.mutate(&c)
			assert.Error(t, c.Validate(), "非法合同应校验失败")
		})
	}
}
