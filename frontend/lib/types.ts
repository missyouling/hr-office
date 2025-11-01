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

export interface SchemeChargeDetail {
  name: string;
  id_number: string;
  department: string;
  base: number;
  amount: number;
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
