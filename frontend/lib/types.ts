export type Part = "personal" | "unit";

export type Scheme =
  | "pension"
  | "medical"
  | "serious_illness"
  | "unemployment"
  | "injury";

export interface Period {
  id: number;
  year_month: string;
  status: string;
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
