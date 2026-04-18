-- 新增模块与目录映射表
CREATE TABLE IF NOT EXISTS storage_module_configs (
    id SERIAL PRIMARY KEY,
    user_id INT REFERENCES users(id),
    module_code VARCHAR(50) NOT NULL,
    module_name VARCHAR(100) NOT NULL,
    base_directory VARCHAR(255) NOT NULL,
    description VARCHAR(500),
    enabled BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_storage_module_configs_user_module 
    ON storage_module_configs(user_id, module_code);

-- 扩展存储规则表
ALTER TABLE storage_rules ADD COLUMN IF NOT EXISTS module_code VARCHAR(50);
ALTER TABLE storage_rules ADD COLUMN IF NOT EXISTS resource_type VARCHAR(100);

CREATE INDEX IF NOT EXISTS idx_storage_rules_module_code ON storage_rules(module_code);
CREATE INDEX IF NOT EXISTS idx_storage_rules_resource_type ON storage_rules(resource_type);
