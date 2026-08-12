export type Part = "personal" | "unit";

export type Scheme =
  | "pension"
  | "medical"
  | "serious_illness"
  | "unemployment"
  | "injury";

export interface User {
  id: number;
  username: string;
  email: string;
  full_name: string;
  active: boolean;
  /** 已废弃角色字段；认证接口不再返回，旧调用方应使用 permissions。 */
  role?: "user" | "admin" | "super_admin" | "manager" | "editor" | "viewer";
  /** 扁平权限数组，如 ["employee.view", "employee.create"]。 */
  permissions: string[];
  department_id?: number | null;  // P7.1 部门级权限：关联 Department 表
  created_at: string;
  updated_at: string;
}

export interface LoginRequest {
  username: string;
  password: string;
}

export interface RegisterRequest {
  username: string;
  email: string;
  password: string;
  full_name?: string;
}

// 通用 JSON 类型
export type UnknownJSON = Record<string, unknown> | unknown[] | null;

export interface AuthResponse {
  token: string;
  user: User;
}

export interface Period {
  id: number;
  year_month: string;
  status: string;
  allow_adjustments: boolean;
  created_at: string;
  updated_at: string;
}

export interface SourceFile {
  id: number;
  period_id: number;
  file_name: string;
  stored_path: string;
  scheme: Scheme;
  part: Part;
  file_type?: string;
  rows: number;
  status: string;
  original_name: string;
  uploaded_at: string;
}

export interface PeriodSummary {
  id: number;
  period_id: number;
  scheme: Scheme;
  part: Part;
  headcount: number;
  base_total: number;
  amount_total: number;
  is_adjustment?: boolean;
}

export interface PersonalCharge {
  id: number;
  period_id: number;
  name: string;
  id_number: string;
  department: string;
  base: number;
  pension: number;
  medical_maternity: number;
  serious_illness: number;
  unemployment: number;
  subtotal: number;
  is_adjustment?: boolean;
}

export interface UnitCharge {
  id: number;
  period_id: number;
  name: string;
  id_number: string;
  department: string;
  base: number;
  pension: number;
  medical_maternity: number;
  serious_illness: number;
  injury: number;
  unemployment: number;
  subtotal: number;
  is_adjustment?: boolean;
}

export interface RosterEntry {
  id: number;
  period_id: number;
  name: string;
  id_number: string;
  department: string;
  title: string;
  remarks: string;
}

export interface BatchUploadItem {
  file_name: string;
  original_name: string;
  scheme: Scheme;
  part: Part;
  imported: number;
  error?: string;
}

export interface SocialInsuranceOptionSet {
  options: string[];
  default: string;
}

export interface SocialInsuranceTemplateOptions {
  generated_at: string;
  personal_identity: SocialInsuranceOptionSet;
  household_type: SocialInsuranceOptionSet;
  education_level: SocialInsuranceOptionSet;
  special_skill: SocialInsuranceOptionSet;
  skill_level: SocialInsuranceOptionSet;
  decrease_reason: SocialInsuranceOptionSet;
  unemployment_reason: SocialInsuranceOptionSet;
  reduction_flag: SocialInsuranceOptionSet;
}

export interface SchemeChargeDetail {
  name: string;
  id_number: string;
  department: string;
  base: number;
  amount: number;
}

export interface CallbackRecord {
  id: number;
  personal_number: string;
  identity_number: string;
  name: string;
  change_type?: string;
  phone?: string;
  remark?: string;
  sequence: number;
  updated_at: string;
}

export interface CallbackRecordsResponse {
  records: CallbackRecord[];
  last_uploaded_at?: string;
  last_file_name?: string;
  personal_map: Record<string, string>;
}

export interface EmployeeImportConflict {
  id?: number;
  name: string;
  identity_number: string;
  status: string;
  department?: string | null;
  position?: string | null;
}

// =============================================================================
// Audit and Monitoring Types
// =============================================================================

export interface AuditLog {
  id: number;
  user_id?: number;
  username?: string;
  action: string;
  resource_type?: string;
  resource_id?: string;
  status: "SUCCESS" | "FAILURE";
  ip_address?: string;
  user_agent?: string;
  timestamp: string;
  duration_ms?: number;
  details?: Record<string, unknown>;
}

export interface AuditStats {
  days: number;
  stats: {
    total_events: number;
    by_status: {
      [key: string]: number;
    };
    active_users: number;
    top_actions: Array<{
      action: string;
      count: number;
    }>;
  };
}

export interface SystemMetrics {
  cpu_usage: number;
  memory_usage: number;
  memory_heap_inuse: number;
  memory_heap_sys: number;
  memory_sys: number;
  memory_gc_count: number;
  disk_usage: number;
  active_connections: number;
  uptime_seconds: number;
  goroutines: number;
  go_version: string;
  database_connections: number;
}

export interface DatabaseStatus {
  status: string;
  database_type: string;
  database_version?: string;
  connection_count?: number;
  active_connections?: number;
  max_connections?: number;
  database_size?: string;
  total_tables?: number;
  total_size?: string;
  last_backup?: string;
  tables?: Array<{
    name: string;
    rows: number;
  }>;
}

export interface SystemInfo {
  hostname?: string;
  platform?: string;
  cpu_cores?: number;
  total_memory?: number;
  go_version?: string;
  build_time?: string;
  version?: string;
  environment?: string;
  start_time?: string;
  uptime?: string;
  health_status?: string;
}

export interface ProvidentFundRecord {
  id: number;
  personal_account: string;
  name: string;
  identity_number: string;
  personal_base: number;
  personal_amount: number;
  company_amount: number;
  total_amount: number;
  contribution_ratio?: number;
  status: "active" | "sealed";
  notes?: string;
  sealed_at?: string | null;
  unsealed_at?: string | null;
  created_at: string;
  updated_at: string;
}

export interface ProvidentFundSettings {
  unit_name: string;
  unit_account: string;
  created_at?: string;
  updated_at?: string;
}

export interface ProvidentFundBillItem {
  id: number;
  bill_id: number;
  record_id: number;
  personal_account: string;
  name: string;
  identity_number: string;
  personal_amount: number;
  company_amount: number;
  total_amount: number;
  created_at: string;
}


export interface ProvidentFundBill {
  id: number;
  month_label: string;
  record_count: number;
  personal_amount_total: number;
  company_amount_total: number;
  combined_amount_total: number;
  created_at: string;
  updated_at: string;
  items?: ProvidentFundBillItem[];
}

export interface DormSite {
  id: number;
  name: string;
  address?: string;
  contact_name?: string;
  contact_phone?: string;
  building_number?: string;
  property_company?: string;
  property_contact?: string;
  support_wechat?: string;
  description?: string;
  charge_config?: DormChargeConfig;
}

export interface DormBuilding {
  id: number;
  site_id: number;
  name: string;
  floors?: number;
  description?: string;
}
export interface DormBed {
  id: number;
  room_id: number;
  bed_number: string;
  status?: string;
}

export type DormCostBearingMode = "personal" | "company";

export interface DormRoom {
  id: number;
  site_id?: number | null;
  building_id: number;
  room_number: string;
  room_type?: string;
  room_category?: string;
  house_layout?: string;
  bed_count?: number;
  area_square?: number;
  first_month_fee?: number;
  status?: string;
  monthly_rent?: number;
  quarterly_rent?: number;
  property_fee?: number;
  guarantee_fee?: number;
  deposit_fee?: number;
  electric_base?: number;
  water_base?: number;
  gas_base?: number;
  trash_fee?: number;
  water_supply_fee?: number;
  sewage_fee?: number;
  inventory_note?: string;
  charge_rates?: DormChargeRates;
  notes?: string;
  beds?: DormBed[];
  cost_bearing_mode?: DormCostBearingMode;
  company_name?: string | null;
}

export interface DormContract {
  id: number;
  employee_id?: number | null;
  room_id: number;
  bed_id?: number | null;
  employee_name?: string;
  employee_department?: string;
  employee_phone?: string;
  employee_id_number?: string;
  employee_residence?: string;
  start_date: string;
  end_date?: string | null;
  rent_amount?: number;
  deposit_amount?: number;
  payment_method?: string;
  status?: string;
  notes?: string;
  attachments?: string[];
  room?: DormRoom;
  bed?: DormBed;
}

export interface DormBill {
  id: number;
  bill_code?: string;
  room_id?: number;
  contract_id?: number;
  employee_name?: string;
  period_label?: string;
  due_date?: string;
  status?: string;
  amount_due?: number;
  amount_paid?: number;
}

export interface DormMeterRecord {
  id: number;
  room_id: number;
  room?: DormRoom;
  meter_date: string;
  billing_start: string;
  billing_end: string;
  billing_month?: string | null;
  inspector?: string;
  charge_details?: DormChargeDetail[];
}

export type ChargeMode = "meter" | "fixed";

export interface DormChargeItemConfig {
  key: string;
  label: string;
  enabled: boolean;
  unit_price?: number;
  unit_label?: string;
  mode?: ChargeMode;
  default_enabled?: boolean;
}

export interface DormChargeConfig {
  items: DormChargeItemConfig[];
}

export interface DormChargeRateEntry {
  key: string;
  unit_price?: number;
  unit_label?: string;
  mode?: ChargeMode;
  start?: number | null;
  end?: number | null;
}

export interface DormChargeRates {
  items: DormChargeRateEntry[];
}

export interface DormChargeDetail {
  key: string;
  label?: string;
  start?: number | null;
  end?: number | null;
  usage?: number | null;
  unit_price?: number | null;
  amount?: number | null;
  unit_label?: string;
  mode?: ChargeMode;
  participants?: Array<{
    name: string;
    contract_id?: number;
    amount?: number;
    ratio?: number;
  }>;
}

export interface UserPreferences {
  user_theme?: string;
  notification?: {
    email_notification?: boolean;
    system_notification?: boolean;
    announcement_popup?: boolean;
    duty_reminder?: boolean;
    reminder_time?: string;
  };
  display?: {
    table_density?: "compact" | "default" | "comfortable";
    default_page_size?: number;
    date_format?: string;
    compact_sidebar?: boolean;
    show_animations?: boolean;
  };
  [key: string]: unknown;
}

// 存储模块配置
export interface StorageModuleConfig {
  id: number;
  user_id: number | null;
  module_code: string;
  module_name: string;
  base_directory: string;
  description: string;
  enabled: boolean;
  created_at: string;
  updated_at: string;
}

// 存储配置
export interface StorageConfig {
  id: number;
  user_id: number;
  name: string;
  type: "local" | "s3" | "webdav";
  enabled: boolean;
  is_default: boolean;
  is_backup: boolean;
  priority: number;
  config: Record<string, unknown>;
  resource_types: string[];
  status: "active" | "inactive" | "error" | "checking";
  description: string;
  health_check_enabled: boolean;
  health_check_interval: number;
  last_health_check: string | null;
  fail_count: number;
  max_fail_count: number;
  created_at: string;
  updated_at: string;
}

export interface StorageRule {
  id: number;
  user_id: number;
  storage_id: number;
  module_code: string;
  resource_type: string;
  category_code: string;
  priority: number;
  enabled: boolean;
  name: string;
  target_type: string;
  target_pattern: string;
  size_min: number | null;
  size_max: number | null;
  fallback_storage_id: number | null;
  base_path?: string;
  created_at: string;
  updated_at: string;
}

export interface StorageTestResult {
  success: boolean;
  message: string;
  latency_ms: number;
}

export interface SysFile {
  id: number;
  storage_type: string;
  path: string;
  original_name: string;
  size: number;
  content_type: string;
  etag: string;
  storage_config_id: number | null;
  created_by: number | null;
  created_at: string;
  updated_at: string;
}
