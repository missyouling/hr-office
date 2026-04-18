
-- Migration: Add System Logs Tables
-- Date: 2026-04-18

-- 1. Extend audit_logs table with new columns
ALTER TABLE audit_logs ADD COLUMN IF NOT EXISTS level text DEFAULT 'INFO';
ALTER TABLE audit_logs ADD COLUMN IF NOT EXISTS trace_id text;
ALTER TABLE audit_logs ADD COLUMN IF NOT EXISTS source text;
ALTER TABLE audit_logs ADD COLUMN IF NOT EXISTS details jsonb;

-- 2. Create system_logs table
CREATE TABLE IF NOT EXISTS system_logs (
  id bigserial PRIMARY KEY,
  level text NOT NULL,
  trace_id text,
  source text,
  message text NOT NULL,
  details jsonb,
  created_at timestamptz DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_system_logs_level ON system_logs(level);
CREATE INDEX IF NOT EXISTS idx_system_logs_created_at ON system_logs(created_at);
CREATE INDEX IF NOT EXISTS idx_system_logs_trace_id ON system_logs(trace_id);

-- 3. Create log_backups table
CREATE TABLE IF NOT EXISTS log_backups (
  id bigserial PRIMARY KEY,
  filename text NOT NULL,
  file_path text NOT NULL,
  file_size bigint,
  record_count int,
  backup_type text,
  status text,
  created_by text,
  created_at timestamptz DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_log_backups_created_at ON log_backups(created_at);

-- 4. Create alert_rules table
CREATE TABLE IF NOT EXISTS alert_rules (
  id bigserial PRIMARY KEY,
  name text NOT NULL,
  keywords text[] NOT NULL,
  threshold int DEFAULT 10,
  time_window int DEFAULT 5,
  enabled boolean DEFAULT true,
  notification_channel text DEFAULT 'in-app',
  created_by text,
  created_at timestamptz DEFAULT now(),
  updated_at timestamptz
);
CREATE INDEX IF NOT EXISTS idx_alert_rules_enabled ON alert_rules(enabled);

-- 5. Create notifications table (local PostgreSQL - no auth.users)
CREATE TABLE IF NOT EXISTS notifications (
  id bigserial PRIMARY KEY,
  user_id bigint,
  title text NOT NULL,
  content text NOT NULL,
  type text DEFAULT 'info',
  read boolean DEFAULT false,
  source text,
  created_at timestamptz DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_notifications_user_id ON notifications(user_id);
CREATE INDEX IF NOT EXISTS idx_notifications_read ON notifications(read);
