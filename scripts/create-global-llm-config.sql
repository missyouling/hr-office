-- =====================================================
-- 创建全局 LLM 配置
-- 说明：此脚本从现有的 LLM 配置复制一份，user_id 设为 NULL
--       全局配置对所有系统用户都可用
-- =====================================================

-- 步骤 1：检查现有的 LLM 配置
SELECT id, user_id, model_name, provider, enabled, is_default 
FROM model_configs 
WHERE config_type = 'llm' AND enabled = true 
LIMIT 5;

-- 步骤 2：创建全局配置（从第一个启用的 LLM 配置复制）
-- 注意：如果现有配置在 (user_id, config_type, model_name) 上有唯一约束，
--       创建 user_id=NULL 的全局配置可能失败。此时需要调整约束或使用 ON CONFLICT
INSERT INTO model_configs (
    user_id,
    config_type,
    provider,
    model_name,
    api_key,
    api_endpoint,
    extra_params,
    enabled,
    is_default,
    role,
    priority,
    is_built_in,
    context_length,
    capabilities,
    rate_limit_rpm,
    rate_limit_tpm,
    created_at,
    updated_at
)
SELECT 
    NULL,  -- 全局配置标记
    config_type,
    provider,
    model_name,
    api_key,
    api_endpoint,
    extra_params,
    true,  -- 启用全局配置
    true,  -- 设为默认
    role,
    0,     -- 重置优先级
    is_built_in,
    context_length,
    capabilities,
    rate_limit_rpm,
    rate_limit_tpm,
    NOW(),
    NOW()
FROM model_configs
WHERE config_type = 'llm' AND enabled = true AND user_id IS NOT NULL
LIMIT 1;

-- 步骤 3：验证全局配置已创建
SELECT id, user_id, model_name, provider, enabled, is_default 
FROM model_configs 
WHERE config_type = 'llm' AND user_id IS NULL;
