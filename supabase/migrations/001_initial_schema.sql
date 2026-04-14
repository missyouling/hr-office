-- =====================================================
-- Supabase 初始数据库模式迁移
-- 版本: 001
-- 创建日期: 2024-01-XX
-- 说明: 创建所有业务表和RLS策略
-- =====================================================

-- =====================================================
-- 1. 创建 profiles 扩展表
-- =====================================================
CREATE TABLE public.profiles (
  id uuid PRIMARY KEY REFERENCES auth.users(id) ON DELETE CASCADE,
  username text UNIQUE NOT NULL,
  full_name text,
  company_id text,
  active boolean DEFAULT true,
  created_at timestamptz DEFAULT now(),
  updated_at timestamptz DEFAULT now()
);

-- 添加索引
CREATE INDEX idx_profiles_username ON public.profiles(username);
CREATE INDEX idx_profiles_company_id ON public.profiles(company_id);

-- 启用RLS
ALTER TABLE public.profiles ENABLE ROW LEVEL SECURITY;

-- RLS策略
CREATE POLICY "Users can view own profile"
  ON public.profiles FOR SELECT
  USING (auth.uid() = id);

CREATE POLICY "Users can update own profile"
  ON public.profiles FOR UPDATE
  USING (auth.uid() = id);

-- 注释
COMMENT ON TABLE public.profiles IS '用户扩展信息表';
COMMENT ON COLUMN public.profiles.username IS '用户名（唯一）';
COMMENT ON COLUMN public.profiles.company_id IS '所属公司ID';

-- =====================================================
-- 2. 创建 periods 表（期间）
-- =====================================================
CREATE TABLE public.periods (
  id bigserial PRIMARY KEY,
  user_id uuid REFERENCES auth.users(id) ON DELETE SET NULL,
  year_month text NOT NULL,
  status text NOT NULL,
  created_at timestamptz DEFAULT now(),
  updated_at timestamptz DEFAULT now()
);

-- 添加索引
CREATE INDEX idx_periods_user_id ON public.periods(user_id);
CREATE INDEX idx_periods_year_month ON public.periods(year_month);
CREATE INDEX idx_periods_status ON public.periods(status);

-- 启用RLS
ALTER TABLE public.periods ENABLE ROW LEVEL SECURITY;

-- RLS策略
CREATE POLICY "Users can manage own periods"
  ON public.periods
  FOR ALL
  USING (auth.uid() = user_id);

-- 注释
COMMENT ON TABLE public.periods IS '社保期间表';
COMMENT ON COLUMN public.periods.year_month IS '期间（格式：YYYY-MM）';
COMMENT ON COLUMN public.periods.status IS '状态（draft/processing/completed）';

-- =====================================================
-- 3. 创建 source_files 表（源文件）
-- =====================================================
CREATE TABLE public.source_files (
  id bigserial PRIMARY KEY,
  user_id uuid REFERENCES auth.users(id) ON DELETE SET NULL,
  period_id bigint REFERENCES public.periods(id) ON DELETE CASCADE,
  file_name text NOT NULL,
  stored_path text NOT NULL,
  scheme text NOT NULL,
  part text NOT NULL,
  file_type text DEFAULT 'normal',
  rows int NOT NULL,
  status text NOT NULL,
  uploaded_at timestamptz NOT NULL,
  original_name text,
  notes text,
  created_at timestamptz DEFAULT now(),
  updated_at timestamptz DEFAULT now()
);

-- 添加索引
CREATE INDEX idx_source_files_user_id ON public.source_files(user_id);
CREATE INDEX idx_source_files_period_id ON public.source_files(period_id);
CREATE INDEX idx_source_files_scheme ON public.source_files(scheme);
CREATE INDEX idx_source_files_part ON public.source_files(part);
CREATE INDEX idx_source_files_file_type ON public.source_files(file_type);

-- 启用RLS
ALTER TABLE public.source_files ENABLE ROW LEVEL SECURITY;

-- RLS策略
CREATE POLICY "Users can manage own source files"
  ON public.source_files
  FOR ALL
  USING (auth.uid() = user_id);

-- 注释
COMMENT ON TABLE public.source_files IS '源文件表';
COMMENT ON COLUMN public.source_files.scheme IS '险种（pension/medical/unemployment等）';
COMMENT ON COLUMN public.source_files.part IS '部分（personal/unit）';
COMMENT ON COLUMN public.source_files.file_type IS '文件类型（normal/adjustment）';

-- =====================================================
-- 4. 创建 raw_records 表（原始记录）
-- =====================================================
CREATE TABLE public.raw_records (
  id bigserial PRIMARY KEY,
  user_id uuid REFERENCES auth.users(id) ON DELETE SET NULL,
  period_id bigint REFERENCES public.periods(id) ON DELETE CASCADE,
  source_file_id bigint REFERENCES public.source_files(id) ON DELETE CASCADE,
  sequence int NOT NULL,
  name text,
  id_type text,
  id_number text,
  department text,
  pay_salary numeric(15,2),
  pay_base numeric(15,2),
  rate_text text,
  amount_due numeric(15,2),
  amount_adjust numeric(15,2),
  person_code text,
  scheme text,
  part text,
  file_type text DEFAULT 'normal',
  created_at timestamptz DEFAULT now(),
  updated_at timestamptz DEFAULT now()
);

-- 添加索引
CREATE INDEX idx_raw_records_user_id ON public.raw_records(user_id);
CREATE INDEX idx_raw_records_period_id ON public.raw_records(period_id);
CREATE INDEX idx_raw_records_source_file_id ON public.raw_records(source_file_id);
CREATE INDEX idx_raw_records_id_number ON public.raw_records(id_number);
CREATE INDEX idx_raw_records_scheme ON public.raw_records(scheme);
CREATE INDEX idx_raw_records_part ON public.raw_records(part);

-- 启用RLS
ALTER TABLE public.raw_records ENABLE ROW LEVEL SECURITY;

-- RLS策略
CREATE POLICY "Users can manage own raw records"
  ON public.raw_records
  FOR ALL
  USING (auth.uid() = user_id);

-- 注释
COMMENT ON TABLE public.raw_records IS '原始数据记录表';

-- =====================================================
-- 5. 创建 period_summaries 表（期间汇总）
-- =====================================================
CREATE TABLE public.period_summaries (
  id bigserial PRIMARY KEY,
  user_id uuid REFERENCES auth.users(id) ON DELETE SET NULL,
  period_id bigint REFERENCES public.periods(id) ON DELETE CASCADE,
  scheme text NOT NULL,
  part text NOT NULL,
  headcount int NOT NULL,
  base_total numeric(15,2) NOT NULL,
  amount_total numeric(15,2) NOT NULL,
  is_adjustment boolean DEFAULT false,
  created_at timestamptz DEFAULT now(),
  updated_at timestamptz DEFAULT now()
);

-- 添加索引
CREATE INDEX idx_period_summaries_user_id ON public.period_summaries(user_id);
CREATE INDEX idx_period_summaries_period_id ON public.period_summaries(period_id);
CREATE INDEX idx_period_summaries_is_adjustment ON public.period_summaries(is_adjustment);

-- 启用RLS
ALTER TABLE public.period_summaries ENABLE ROW LEVEL SECURITY;

-- RLS策略
CREATE POLICY "Users can manage own period summaries"
  ON public.period_summaries
  FOR ALL
  USING (auth.uid() = user_id);

-- 注释
COMMENT ON TABLE public.period_summaries IS '期间汇总表';

-- =====================================================
-- 6. 创建 personal_charges 表（个人费用）
-- =====================================================
CREATE TABLE public.personal_charges (
  id bigserial PRIMARY KEY,
  user_id uuid REFERENCES auth.users(id) ON DELETE SET NULL,
  period_id bigint REFERENCES public.periods(id) ON DELETE CASCADE,
  name text NOT NULL,
  id_number text NOT NULL,
  department text,
  base numeric(15,2) NOT NULL,
  pension numeric(15,2) DEFAULT 0,
  medical_maternity numeric(15,2) DEFAULT 0,
  serious_illness numeric(15,2) DEFAULT 0,
  unemployment numeric(15,2) DEFAULT 0,
  subtotal numeric(15,2) NOT NULL,
  is_adjustment boolean DEFAULT false,
  created_at timestamptz DEFAULT now(),
  updated_at timestamptz DEFAULT now()
);

-- 添加索引
CREATE INDEX idx_personal_charges_user_id ON public.personal_charges(user_id);
CREATE INDEX idx_personal_charges_period_id ON public.personal_charges(period_id);
CREATE INDEX idx_personal_charges_id_number ON public.personal_charges(id_number);
CREATE INDEX idx_personal_charges_is_adjustment ON public.personal_charges(is_adjustment);

-- 启用RLS
ALTER TABLE public.personal_charges ENABLE ROW LEVEL SECURITY;

-- RLS策略
CREATE POLICY "Users can manage own personal charges"
  ON public.personal_charges
  FOR ALL
  USING (auth.uid() = user_id);

-- 注释
COMMENT ON TABLE public.personal_charges IS '个人费用明细表';

-- =====================================================
-- 7. 创建 unit_charges 表（单位费用）
-- =====================================================
CREATE TABLE public.unit_charges (
  id bigserial PRIMARY KEY,
  user_id uuid REFERENCES auth.users(id) ON DELETE SET NULL,
  period_id bigint REFERENCES public.periods(id) ON DELETE CASCADE,
  name text NOT NULL,
  id_number text NOT NULL,
  department text,
  base numeric(15,2) NOT NULL,
  pension numeric(15,2) DEFAULT 0,
  medical_maternity numeric(15,2) DEFAULT 0,
  serious_illness numeric(15,2) DEFAULT 0,
  injury numeric(15,2) DEFAULT 0,
  unemployment numeric(15,2) DEFAULT 0,
  subtotal numeric(15,2) NOT NULL,
  is_adjustment boolean DEFAULT false,
  created_at timestamptz DEFAULT now(),
  updated_at timestamptz DEFAULT now()
);

-- 添加索引
CREATE INDEX idx_unit_charges_user_id ON public.unit_charges(user_id);
CREATE INDEX idx_unit_charges_period_id ON public.unit_charges(period_id);
CREATE INDEX idx_unit_charges_id_number ON public.unit_charges(id_number);
CREATE INDEX idx_unit_charges_is_adjustment ON public.unit_charges(is_adjustment);

-- 启用RLS
ALTER TABLE public.unit_charges ENABLE ROW LEVEL SECURITY;

-- RLS策略
CREATE POLICY "Users can manage own unit charges"
  ON public.unit_charges
  FOR ALL
  USING (auth.uid() = user_id);

-- 注释
COMMENT ON TABLE public.unit_charges IS '单位费用明细表';

-- =====================================================
-- 8. 创建 roster_entries 表（花名册）
-- =====================================================
CREATE TABLE public.roster_entries (
  id bigserial PRIMARY KEY,
  user_id uuid REFERENCES auth.users(id) ON DELETE SET NULL,
  period_id bigint REFERENCES public.periods(id) ON DELETE CASCADE,
  name text NOT NULL,
  id_number text NOT NULL,
  department text,
  title text,
  remarks text,
  created_at timestamptz DEFAULT now(),
  updated_at timestamptz DEFAULT now()
);

-- 添加索引
CREATE INDEX idx_roster_entries_user_id ON public.roster_entries(user_id);
CREATE INDEX idx_roster_entries_period_id ON public.roster_entries(period_id);
CREATE INDEX idx_roster_entries_id_number ON public.roster_entries(id_number);

-- 启用RLS
ALTER TABLE public.roster_entries ENABLE ROW LEVEL SECURITY;

-- RLS策略
CREATE POLICY "Users can manage own roster entries"
  ON public.roster_entries
  FOR ALL
  USING (auth.uid() = user_id);

-- 注释
COMMENT ON TABLE public.roster_entries IS '花名册表';

-- =====================================================
-- 9. 创建 audit_logs 表（审计日志）
-- =====================================================
CREATE TABLE public.audit_logs (
  id bigserial PRIMARY KEY,
  user_id uuid REFERENCES auth.users(id) ON DELETE SET NULL,
  username text,
  action text NOT NULL,
  resource_type text,
  resource_id text,
  status text NOT NULL,
  ip_address text,
  user_agent text,
  details jsonb,
  created_at timestamptz DEFAULT now()
);

-- 添加索引
CREATE INDEX idx_audit_logs_user_id ON public.audit_logs(user_id);
CREATE INDEX idx_audit_logs_action ON public.audit_logs(action);
CREATE INDEX idx_audit_logs_status ON public.audit_logs(status);
CREATE INDEX idx_audit_logs_created_at ON public.audit_logs(created_at);

-- 启用RLS
ALTER TABLE public.audit_logs ENABLE ROW LEVEL SECURITY;

-- RLS策略（审计日志可能需要管理员查看所有）
CREATE POLICY "Users can view own audit logs"
  ON public.audit_logs
  FOR SELECT
  USING (auth.uid() = user_id);

-- 注释
COMMENT ON TABLE public.audit_logs IS '审计日志表';

-- =====================================================
-- 10. 创建触发器（自动更新 updated_at）
-- =====================================================
CREATE OR REPLACE FUNCTION public.handle_updated_at()
RETURNS TRIGGER AS $$
BEGIN
  NEW.updated_at = now();
  RETURN NEW;
END;
$$ LANGUAGE plpgsql SECURITY DEFINER;

-- 为所有表添加 updated_at 触发器
CREATE TRIGGER set_updated_at_profiles
  BEFORE UPDATE ON public.profiles
  FOR EACH ROW
  EXECUTE FUNCTION public.handle_updated_at();

CREATE TRIGGER set_updated_at_periods
  BEFORE UPDATE ON public.periods
  FOR EACH ROW
  EXECUTE FUNCTION public.handle_updated_at();

CREATE TRIGGER set_updated_at_source_files
  BEFORE UPDATE ON public.source_files
  FOR EACH ROW
  EXECUTE FUNCTION public.handle_updated_at();

CREATE TRIGGER set_updated_at_raw_records
  BEFORE UPDATE ON public.raw_records
  FOR EACH ROW
  EXECUTE FUNCTION public.handle_updated_at();

CREATE TRIGGER set_updated_at_period_summaries
  BEFORE UPDATE ON public.period_summaries
  FOR EACH ROW
  EXECUTE FUNCTION public.handle_updated_at();

CREATE TRIGGER set_updated_at_personal_charges
  BEFORE UPDATE ON public.personal_charges
  FOR EACH ROW
  EXECUTE FUNCTION public.handle_updated_at();

CREATE TRIGGER set_updated_at_unit_charges
  BEFORE UPDATE ON public.unit_charges
  FOR EACH ROW
  EXECUTE FUNCTION public.handle_updated_at();

CREATE TRIGGER set_updated_at_roster_entries
  BEFORE UPDATE ON public.roster_entries
  FOR EACH ROW
  EXECUTE FUNCTION public.handle_updated_at();

-- =====================================================
-- 11. 创建自动创建profile的触发器
-- =====================================================
CREATE OR REPLACE FUNCTION public.handle_new_user()
RETURNS TRIGGER AS $$
BEGIN
  INSERT INTO public.profiles (id, username, full_name, company_id)
  VALUES (
    NEW.id,
    COALESCE(NEW.raw_user_meta_data->>'username', split_part(NEW.email, '@', 1)),
    NEW.raw_user_meta_data->>'full_name',
    NEW.raw_user_meta_data->>'company_id'
  );
  RETURN NEW;
END;
$$ LANGUAGE plpgsql SECURITY DEFINER;

-- 当新用户注册时自动创建profile
CREATE TRIGGER on_auth_user_created
  AFTER INSERT ON auth.users
  FOR EACH ROW
  EXECUTE FUNCTION public.handle_new_user();

-- =====================================================
-- 12. 创建辅助函数
-- =====================================================

-- 获取用户的所有期间
CREATE OR REPLACE FUNCTION public.get_user_periods(user_uuid uuid)
RETURNS TABLE (
  id bigint,
  year_month text,
  status text,
  created_at timestamptz
) AS $$
BEGIN
  RETURN QUERY
  SELECT p.id, p.year_month, p.status, p.created_at
  FROM public.periods p
  WHERE p.user_id = user_uuid
  ORDER BY p.year_month DESC;
END;
$$ LANGUAGE plpgsql SECURITY DEFINER;

-- 获取期间统计
CREATE OR REPLACE FUNCTION public.get_period_stats(period_id_param bigint)
RETURNS json AS $$
DECLARE
  result json;
BEGIN
  SELECT json_build_object(
    'period_id', period_id_param,
    'files_count', (SELECT COUNT(*) FROM public.source_files WHERE period_id = period_id_param),
    'raw_records_count', (SELECT COUNT(*) FROM public.raw_records WHERE period_id = period_id_param),
    'personal_charges_count', (SELECT COUNT(*) FROM public.personal_charges WHERE period_id = period_id_param),
    'unit_charges_count', (SELECT COUNT(*) FROM public.unit_charges WHERE period_id = period_id_param),
    'roster_count', (SELECT COUNT(*) FROM public.roster_entries WHERE period_id = period_id_param)
  ) INTO result;
  
  RETURN result;
END;
$$ LANGUAGE plpgsql SECURITY DEFINER;

-- =====================================================
-- 完成
-- =====================================================
-- 迁移完成！
-- 下一步：使用 Supabase CLI 或 Dashboard 执行此脚本