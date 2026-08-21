package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestIsValidAdminContractStatus 状态常量校验：合法四态通过，非法状态拒绝。
func TestIsValidAdminContractStatus(t *testing.T) {
	assert.True(t, IsValidAdminContractStatus(AdminContractStatusDraft))
	assert.True(t, IsValidAdminContractStatus(AdminContractStatusActive))
	assert.True(t, IsValidAdminContractStatus(AdminContractStatusExpired))
	assert.True(t, IsValidAdminContractStatus(AdminContractStatusCancelled))
	assert.False(t, IsValidAdminContractStatus(""))
	assert.False(t, IsValidAdminContractStatus("pending"))
	assert.False(t, IsValidAdminContractStatus("voided"))
}

// TestAdminContractValidate 字段约束校验：正常通过，异常边界逐项拒绝。
func TestAdminContractValidate(t *testing.T) {
	valid := AdminContract{
		ContractNo:   "XZ-2026-001",
		Name:         "保洁服务合同",
		Counterparty: "某某保洁公司",
		ContractType: "服务合同",
		StartDate:    "2026-01-01",
		EndDate:      "2026-12-31",
		Status:       AdminContractStatusDraft,
	}
	require.NoError(t, valid.Validate(), "合法行政合同应通过校验")

	cases := []struct {
		name   string
		mutate func(*AdminContract)
	}{
		{"状态非法", func(c *AdminContract) { c.Status = "pending" }},
		{"合同编号缺失", func(c *AdminContract) { c.ContractNo = "" }},
		{"合同名称缺失", func(c *AdminContract) { c.Name = " " }},
		{"相对方缺失", func(c *AdminContract) { c.Counterparty = "" }},
		{"合同类型缺失", func(c *AdminContract) { c.ContractType = "" }},
		{"起始日缺失", func(c *AdminContract) { c.StartDate = "" }},
		{"到期日缺失", func(c *AdminContract) { c.EndDate = "" }},
		{"起始日格式错误", func(c *AdminContract) { c.StartDate = "2026/01/01" }},
		{"到期日格式错误", func(c *AdminContract) { c.EndDate = "2026-13-01" }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := valid
			tc.mutate(&c)
			assert.Error(t, c.Validate(), "非法行政合同应校验失败")
		})
	}
}

// TestAdminContractOptionalFields 选填字段（含税金额/币种/负责人/备注）不影响校验通过。
func TestAdminContractOptionalFields(t *testing.T) {
	amount := 12345.67
	c := AdminContract{
		ContractNo:    "XZ-2026-002",
		Name:          "设备采购合同",
		Counterparty:  "某某设备公司",
		ContractType:  "采购合同",
		StartDate:     "2026-01-01",
		EndDate:       "2026-06-30",
		AmountInclTax: &amount,
		Currency:      "CNY",
		Owner:         "张三",
		Remarks:       "分批付款",
		Status:        AdminContractStatusDraft,
	}
	require.NoError(t, c.Validate(), "含选填字段的合法合同应通过校验")
}
