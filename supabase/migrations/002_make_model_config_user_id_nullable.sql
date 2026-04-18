-- =====================================================
-- Migration 002: 支持全局 LLM 模型配置
-- 日期: 2026-04-18
-- 说明: 将 model_configs.user_id 改为可空，支持 user_id=NULL 表示全局配置
-- =====================================================

-- 修改 model_configs 表，使 user_id 可空
ALTER TABLE IF EXISTS public.model_configs
ALTER COLUMN user_id DROP NOT NULL;

-- 添加注释说明
COMMENT ON COLUMN public.model_configs.user_id IS '用户 ID（NULL 表示全局配置，所有用户可用）';
