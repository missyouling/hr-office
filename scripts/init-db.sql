-- PostgreSQL initialization script for 人事行政管理系统 (hr-office)
-- This script is automatically executed when the PostgreSQL container starts for the first time

-- Create extensions if needed
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Set timezone to Asia/Shanghai
SET timezone = 'Asia/Shanghai';

-- Create additional indexes for performance (GORM will create the basic ones)
-- These will be created after GORM migration, so we use IF NOT EXISTS

-- Create function to add indexes after tables are created
CREATE OR REPLACE FUNCTION create_performance_indexes()
RETURNS void AS $$
BEGIN
    -- Index for audit logs performance
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'audit_logs') THEN
        CREATE INDEX IF NOT EXISTS idx_audit_logs_created_at_desc ON audit_logs (created_at DESC);
        CREATE INDEX IF NOT EXISTS idx_audit_logs_user_action ON audit_logs (user_id, action);
        CREATE INDEX IF NOT EXISTS idx_audit_logs_resource_action ON audit_logs (resource, action);
    END IF;

    -- Index for users
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'users') THEN
        CREATE INDEX IF NOT EXISTS idx_users_email_verified ON users (email_verified);
        CREATE INDEX IF NOT EXISTS idx_users_active ON users (active);
    END IF;

    -- Index for periods
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'periods') THEN
        CREATE INDEX IF NOT EXISTS idx_periods_year_month ON periods (year_month);
        CREATE INDEX IF NOT EXISTS idx_periods_status ON periods (status);
    END IF;

    -- Index for raw records
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'raw_records') THEN
        CREATE INDEX IF NOT EXISTS idx_raw_records_period_scheme ON raw_records (period_id, scheme);
        CREATE INDEX IF NOT EXISTS idx_raw_records_id_number ON raw_records (id_number);
    END IF;

    -- Index for personal charges
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'personal_charges') THEN
        CREATE INDEX IF NOT EXISTS idx_personal_charges_period ON personal_charges (period_id);
        CREATE INDEX IF NOT EXISTS idx_personal_charges_id_number ON personal_charges (id_number);
    END IF;

    -- Index for unit charges
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'unit_charges') THEN
        CREATE INDEX IF NOT EXISTS idx_unit_charges_period ON unit_charges (period_id);
        CREATE INDEX IF NOT EXISTS idx_unit_charges_id_number ON unit_charges (id_number);
    END IF;

    RAISE NOTICE 'Performance indexes created successfully';
END;
$$ LANGUAGE plpgsql;

-- Note: The create_performance_indexes() function can be called manually after GORM migration
-- or you can create a startup script to call it automatically

-- Database configuration optimizations for social insurance workload
ALTER DATABASE siapp SET timezone = 'Asia/Shanghai';
ALTER DATABASE siapp SET default_text_search_config = 'simple';

-- Log successful initialization
INSERT INTO pg_stat_statements_info (dealloc) VALUES (0) ON CONFLICT DO NOTHING;

-- Create a view for audit log summary (will be created after migration)
-- This is a template that can be executed later
/*
CREATE OR REPLACE VIEW audit_summary AS
SELECT
    DATE(created_at) as date,
    action,
    resource,
    COUNT(*) as count,
    COUNT(DISTINCT user_id) as unique_users
FROM audit_logs
WHERE created_at >= CURRENT_DATE - INTERVAL '30 days'
GROUP BY DATE(created_at), action, resource
ORDER BY date DESC, count DESC;
*/
