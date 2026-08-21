"use client";

import { PageTransition } from "@/components/motion/page-transition";
import { useState, useRef, useCallback, useEffect, useMemo } from "react";
import type { ChangeEvent, ReactNode, Dispatch, SetStateAction, DragEvent, KeyboardEvent } from "react";
import { toast } from "sonner";
import {
  Plus,
  Upload,
  Download,
  MoreHorizontal,
  Search,
  UserMinus,
  CalendarPlus,
  CalendarMinus,
  Settings,
  Dice6,
  Trash2,
  X,
  Printer,
  FileText,
  PiggyBank,
  Eye,
  ShieldCheck,
  FileSpreadsheet,
  Loader2,
  Unlock,
} from "lucide-react";
import * as XLSX from "xlsx";
import "xlsx/dist/cpexcel";

import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger, DropdownMenuSeparator } from "@/components/ui/dropdown-menu";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Badge } from "@/components/ui/badge";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { ScrollArea, ScrollBar } from "@/components/ui/scroll-area";
import { Textarea } from "@/components/ui/textarea";
import {
  fetchEmployees,
  importEmployees as importEmployeesApi,
  downloadEmployeeTemplate,
  downloadResignedEmployeeTemplate,
  exportEmployees,
  deleteEmployeesApi,
  resignEmployeeApi,
  downloadResignProof,
  restoreEmployees,
  fetchSocialInsuranceChanges,
  importSocialInsuranceChanges,
  deleteSocialInsuranceChanges,
  fetchSocialInsuranceOptions,
  createSocialInsuranceChange,
  updateSocialInsuranceChange,
  importResignedEmployees as importResignedEmployeesApi,
  EmployeeImportConflictError,
  EmployeeResponse,
  EmployeeImportMode,
  type SocialInsuranceRecordDTO,
  type SocialInsuranceImportPayload,
  type SocialInsuranceImportRecordPayload,
  type SocialInsuranceManualPayload,
  type SocialInsuranceChangeType as ApiSocialInsuranceChangeType,
  fetchProvidentRecords,
  createProvidentRecord,
  updateProvidentRecord,
  sealProvidentRecord,
  unsealProvidentRecord,
  getProvidentSettings,
  updateProvidentSettings,
  generateProvidentBill,
  fetchProvidentBills,
  fetchProvidentBillDetail,
  deleteProvidentBill,
  downloadInsuranceTemplate,
  fetchUserPreferences,
  updateUserPreferences,
} from "@/lib/api";
import { createReportPdf } from "@/lib/pdf";
import { useAuth } from "@/lib/auth";
import { parseListPreference, sanitizeSortPreference, type TableSortState } from "@/lib/preferences";
import { cn } from "@/lib/utils";
import { DIALOG_SIZES } from "@/lib/dialog-sizes";
import { DataTableWrapper } from "@/components/common/data-table-wrapper";
import { RequirePermission } from "@/components/auth/RequirePermission";
import type {
  SocialInsuranceTemplateOptions,
  EmployeeImportConflict,
  ProvidentFundRecord,
  ProvidentFundSettings,
  ProvidentFundBill,
} from "@/lib/types";

// 定义所有可用的字段
interface FieldOption {
  key: keyof Employee;
  label: string;
  width?: string;
}

type PrintOrientation = "auto" | "portrait" | "landscape";
type InsuranceView = "increase" | "decrease" | "provident";
type RosterTab =
  | "active"
  | "resigned"
  | "insurance-increase"
  | "insurance-decrease"
  | "provident";

interface PrintSettings {
  title: string;
  watermark: string;
  orientation: PrintOrientation;
}

interface UnitInfo {
  socialCode: string;
  unitName: string;
}

const EMPLOYEE_VISIBLE_FIELDS_STORAGE_KEY = "employee-visible-fields-v2";
const LEGACY_VISIBLE_FIELD_STORAGE_KEYS = ["employee-visible-fields", "employee-visible-fields-v1"];
const PRINT_SETTINGS_STORAGE_KEY = "employee-print-settings-v1";
const UNIT_INFO_STORAGE_KEY = "insurance-unit-info";
const DEFAULT_UNIT_INFO: UnitInfo = {
  socialCode: "20302685",
  unitName: "重庆星达铜业有限公司",
};

const AVAILABLE_FIELDS: FieldOption[] = [
  { key: "name", label: "姓名", width: "100px" },
  { key: "employeeId", label: "工号", width: "80px" },
  { key: "insuranceStatus", label: "参保状态", width: "100px" },
  { key: "department", label: "部门", width: "120px" },
  { key: "position", label: "岗位", width: "120px" },
  { key: "gender", label: "性别", width: "60px" },
  { key: "age", label: "年龄", width: "60px" },
  { key: "workYears", label: "工龄", width: "60px" },
  { key: "hireDate", label: "入职时间", width: "120px" },
  { key: "birthMonth", label: "出生月份", width: "80px" },
  { key: "education", label: "文化程度", width: "100px" },
  { key: "politicalStatus", label: "政治面貌", width: "100px" },
  { key: "workClothingSize", label: "工作服", width: "80px" },
  { key: "safetyShoeSize", label: "劳保鞋", width: "80px" },
  { key: "householdType", label: "户口性质", width: "80px" },
  { key: "ethnicity", label: "民族", width: "80px" },
  { key: "nativePlace", label: "籍贯", width: "100px" },
  { key: "idAddress", label: "身份证地址", width: "250px" },
  { key: "idNumber", label: "身份证号码", width: "200px" },
  { key: "maritalStatus", label: "婚姻状况", width: "80px" },
  { key: "hasBirth", label: "是否生育", width: "80px" },
  { key: "phone", label: "联系电话", width: "120px" },
  { key: "emergencyContact", label: "紧急联系人", width: "100px" },
  { key: "emergencyPhone", label: "紧急联系电话", width: "200px" },
  { key: "currentAddress", label: "现居住地址", width: "250px" },
  { key: "graduateSchool", label: "毕业院校", width: "150px" },
  { key: "major", label: "专业", width: "120px" },
  { key: "graduationTime", label: "毕业时间", width: "100px" },
  { key: "socialInsuranceNumber", label: "社保编号", width: "140px" },
  { key: "providentFundNumber", label: "公积金编号", width: "140px" },
];

// 默认显示的字段
const DEFAULT_VISIBLE_FIELDS: (keyof Employee)[] = [
  "name",
  "employeeId",
  "insuranceStatus",
  "department",
  "position",
  "gender",
  "age",
  "workYears",
  "hireDate",
  "idNumber",
  "phone",
];

interface InsuranceStatusOption {
  value: string;
  label: string;
}

const ACTIVE_INSURANCE_STATUS_OPTIONS: InsuranceStatusOption[] = [
  { value: "已参保", label: "已参保" },
  { value: "未参保", label: "未参保" },
];

const RESIGNED_INSURANCE_STATUS_OPTIONS: InsuranceStatusOption[] = [
  { value: "已退保", label: "已退保" },
  { value: "未退保", label: "未退保" },
];
const ALIGNMENT_CLASS: Record<"left" | "center" | "right", string> = {
  left: "text-left",
  center: "text-center",
  right: "text-right",
};

const IMPORT_MODE_OPTIONS: Array<{ value: EmployeeImportMode; label: string; description: string }> = [
  {
    value: "merge",
    label: "更新 + 插入",
    description: "已存在的员工将更新资料，新的员工自动加入",
  },
  {
    value: "update",
    label: "仅更新",
    description: "只更新已存在的员工，忽略模板中的新增人员",
  },
  {
    value: "insert",
    label: "仅插入",
    description: "只导入模板中的新员工，已存在的人员将跳过",
  },
];

const RESIGN_REASON_OPTIONS = [
  {
    value: "management",
    label: "直接上级的管理方式",
    description: "缺乏尊重、沟通不畅、无法提供支持等",
  },
  {
    value: "compensation",
    label: "薪酬福利缺乏竞争力",
    description: "感觉收入、奖金和福利与个人贡献或市场水平不匹配",
  },
  {
    value: "growth",
    label: "职业发展空间受限",
    description: "看不到晋升通道或技能成长机会",
  },
  {
    value: "worklife",
    label: "工作与生活严重失衡",
    description: "长期加班或高强度压力导致身心疲惫",
  },
  {
    value: "culture",
    label: "公司文化与价值观不符",
    description: "对公司文化氛围或价值观不认同，缺乏归属感",
  },
  {
    value: "personal",
    label: "个人原因",
    description: "个人发展规划、家庭或其他主观因素",
  },
];

const RESIGNED_EXTRA_COLUMNS = ["resignDate", "resignReasons"] as const;
const RESIGNED_COLUMN_ORDER_STORAGE_KEY = "resigned-column-order";
const INSURANCE_COLUMN_ORDER_STORAGE_KEY = "insurance-column-order";
const INSURANCE_VISIBLE_COLUMNS_STORAGE_KEY = "insurance-visible-columns";
const INSURANCE_SORT_STORAGE_KEY = "insurance-sort";
const EMPLOYEE_SORT_STORAGE_KEY = "employee-sort";
const RESIGNED_SORT_STORAGE_KEY = "resigned-sort";
const PROVIDENT_VISIBLE_COLUMNS_STORAGE_KEY = "provident-visible-columns";
const PROVIDENT_COLUMN_ORDER_STORAGE_KEY = "provident-column-order";
const PROVIDENT_SORT_STORAGE_KEY = "provident-sort";

type ProvidentColumnId =
  | "personal_account"
  | "name"
  | "identity_number"
  | "personal_base"
  | "contribution_ratio"
  | "personal_amount"
  | "company_amount"
  | "total_amount"
  | "status"
  | "sealed_at"
  | "notes"
  | "created_at";

interface ProvidentColumnConfig extends ColumnConfig<ProvidentFundRecord> {
  id: ProvidentColumnId;
  numeric?: boolean;
  defaultVisible?: boolean;
}

const PROVIDENT_COLUMN_CONFIGS: ProvidentColumnConfig[] = [
  { id: "personal_account", label: "个人账号", width: "140px" },
  { id: "name", label: "姓名", width: "120px" },
  { id: "identity_number", label: "证件号码", width: "170px" },
  { id: "personal_base", label: "缴存基数", numeric: true, defaultVisible: true },
  {
    id: "contribution_ratio",
    label: "缴存比例",
    width: "110px",
    defaultVisible: true,
    getValue: (record) => formatRatioDisplay(record.contribution_ratio),
  },
  { id: "personal_amount", label: "月应缴额（个人）", numeric: true, width: "160px" },
  { id: "company_amount", label: "月应缴额（单位）", numeric: true, width: "160px" },
  { id: "total_amount", label: "合计", numeric: true, width: "130px" },
  { id: "status", label: "状态", width: "110px" },
  { id: "sealed_at", label: "封存时间", width: "140px" },
  { id: "notes", label: "备注", width: "180px" },
  { id: "created_at", label: "创建时间", width: "140px" },
];

const DEFAULT_PROVIDENT_COLUMN_ORDER: ProvidentColumnId[] = PROVIDENT_COLUMN_CONFIGS.map((column) => column.id);
const DEFAULT_PROVIDENT_VISIBLE_COLUMNS: ProvidentColumnId[] = [
  "personal_account",
  "name",
  "identity_number",
  "personal_base",
  "contribution_ratio",
  "personal_amount",
  "company_amount",
  "total_amount",
  "status",
];

const PROVIDENT_TEMPLATE_HEADERS = [
  "个人账号",
  "姓名",
  "证件号码",
  "个人缴存基数",
  "缴存比例（%）",
  "月应缴额（个人）",
  "月应缴额（单位）",
  "备注",
];

const PROVIDENT_TEMPLATE_SHEET_NAME = "公积金导入模板";
const DEFAULT_PROVIDENT_UNIT_NAME = "重庆星达铜业有限公司";
const DEFAULT_PROVIDENT_UNIT_ACCOUNT = "201005128130";
const DEFAULT_RESIGNED_ORDER_BASE: string[] = [...DEFAULT_VISIBLE_FIELDS.map((field) => field as string), ...RESIGNED_EXTRA_COLUMNS];
const PREF_KEY_EMPLOYEE_SORT = "employeeActiveSort";
const PREF_KEY_RESIGNED_SORT = "employeeResignedSort";
const PREF_KEY_INSURANCE_SORT = "employeeInsuranceSort";
const PREF_KEY_PROVIDENT_SORT = "employeeProvidentSort";
const PREF_KEY_EMPLOYEE_FILTERS = "employeeActiveFilters";
const PREF_KEY_INSURANCE_FILTERS = "employeeInsuranceFilters";
const PREF_KEY_PROVIDENT_FILTERS = "employeeProvidentFilters";
const PREF_KEY_RESIGNED_FILTERS = "employeeResignedFilters";

const formatFileSize = (bytes: number): string => {
  if (!Number.isFinite(bytes) || bytes < 0) {
    return "未知大小";
  }
  if (bytes === 0) {
    return "0 B";
  }
  const units = ["B", "KB", "MB", "GB"];
  let size = bytes;
  let unitIndex = 0;
  while (size >= 1024 && unitIndex < units.length - 1) {
    size /= 1024;
    unitIndex += 1;
  }
  return `${size.toFixed(size >= 10 || unitIndex === 0 ? 0 : 1)} ${units[unitIndex]}`;
};

const RESIGN_PROOF_ALLOWED_MIME_PREFIXES = ["application/pdf", "image/"];
const RESIGN_PROOF_ALLOWED_EXTENSIONS = new Set(["pdf", "png", "jpg", "jpeg", "gif", "bmp", "webp"]);
const MAX_RESIGN_PROOF_SIZE_BYTES = 20 * 1024 * 1024; // 20 MB，与后端 multipart 限制保持一致

type SortDirection = "asc" | "desc";

interface ColumnConfig<Row> {
  id: string;
  label: string;
  sortable?: boolean;
  width?: string;
  getValue?: (row: Row) => unknown;
  renderCell?: (row: Row) => ReactNode;
}

type EmployeeColumnId = keyof Employee | typeof RESIGNED_EXTRA_COLUMNS[number];
type ResignedColumnType = "base" | "resignDate" | "resignReasons" | "resignProof";

interface ResignedColumnDefinition extends ColumnConfig<Employee> {
  type: ResignedColumnType;
}

const readLocalStorageJSON = <T,>(key: string): T | null => {
  if (typeof window === "undefined") {
    return null;
  }
  try {
    const raw = window.localStorage.getItem(key);
    if (!raw) {
      return null;
    }
    return JSON.parse(raw) as T;
  } catch (error) {
    console.warn(`[localStorage] 读取 ${key} 失败`, error);
    return null;
  }
};

const writeLocalStorageJSON = (key: string, value: unknown) => {
  if (typeof window === "undefined") {
    return;
  }
  try {
    window.localStorage.setItem(key, JSON.stringify(value));
  } catch (error) {
    console.warn(`[localStorage] 写入 ${key} 失败`, error);
  }
};

const arraysShallowEqual = <T,>(a: T[], b: T[]) => {
  if (a === b) return true;
  if (a.length !== b.length) return false;
  for (let i = 0; i < a.length; i += 1) {
    if (a[i] !== b[i]) {
      return false;
    }
  }
  return true;
};

const sanitizeVisibleFieldKeys = (keys: (keyof Employee)[] | null | undefined) => {
  const allowed = new Set<keyof Employee>(AVAILABLE_FIELDS.map((field) => field.key));
  if (!keys || keys.length === 0) {
    return [...DEFAULT_VISIBLE_FIELDS];
  }

  const unique: (keyof Employee)[] = [];
  keys.forEach((key) => {
    if (allowed.has(key) && !unique.includes(key)) {
      unique.push(key);
    }
  });

  if (unique.length === 0) {
    return [...DEFAULT_VISIBLE_FIELDS];
  }

  return unique;
};

const reorderList = <T,>(list: T[], source: T, target: T) => {
  const next = [...list];
  const sourceIndex = next.indexOf(source);
  const targetIndex = next.indexOf(target);
  if (sourceIndex === -1 || targetIndex === -1 || sourceIndex === targetIndex) {
    return next;
  }
  next.splice(sourceIndex, 1);
  next.splice(targetIndex, 0, source);
  return next;
};

const toSortableValue = (value: unknown): number | string => {
  if (value === null || value === undefined) {
    return "";
  }
  if (value instanceof Date) {
    return value.getTime();
  }
  if (typeof value === "number") {
    return value;
  }
  const text = String(value).trim();
  if (!text) {
    return "";
  }
  const numeric = Number(text);
  if (!Number.isNaN(numeric)) {
    return numeric;
  }
  const normalized = normalizeDateInput(text);
  if (normalized && /^\d{4}-\d{2}-\d{2}/.test(normalized)) {
    const timestamp = new Date(normalized.replace(/-/g, "/")).getTime();
    if (!Number.isNaN(timestamp)) {
      return timestamp;
    }
  }
  return text;
};

const compareSortableValues = (a: unknown, b: unknown, direction: SortDirection) => {
  const left = toSortableValue(a);
  const right = toSortableValue(b);
  let result = 0;
  if (typeof left === "number" && typeof right === "number") {
    result = left - right;
  } else if (typeof left === "number") {
    result = 1;
  } else if (typeof right === "number") {
    result = -1;
  } else {
    result = String(left).localeCompare(String(right), "zh-CN", { numeric: true });
  }
  return direction === "asc" ? result : -result;
};

const applySort = <Row, T extends string>(
  rows: Row[],
  sortState: TableSortState<T>,
  getValue: (row: Row, columnId: T) => unknown,
) => {
  if (!sortState.key) {
    return rows;
  }
  const key = sortState.key as T;
  return [...rows].sort((a, b) => compareSortableValues(getValue(a, key), getValue(b, key), sortState.direction));
};

const cycleSort = <T extends string>(columnId: T, setter: Dispatch<SetStateAction<TableSortState<T>>>) => {
  setter((prev) => {
    if (prev.key !== columnId) {
      return { key: columnId, direction: "asc" };
    }
    if (prev.direction === "asc") {
      return { key: columnId, direction: "desc" };
    }
    return { key: null, direction: "asc" };
  });
};

const isAllowedResignProofType = (mime: string, extension: string) => {
  if (mime) {
    const normalizedMime = mime.toLowerCase();
    if (RESIGN_PROOF_ALLOWED_MIME_PREFIXES.some((prefix) => normalizedMime.startsWith(prefix))) {
      return true;
    }
  }
  if (extension) {
    return RESIGN_PROOF_ALLOWED_EXTENSIONS.has(extension.toLowerCase());
  }
  return false;
};

interface SearchableSelectProps {
  id: string;
  value: string;
  onChange: (value: string) => void;
  options: string[];
  placeholder: string;
}

const SearchableSelect = ({ id, value, onChange, options, placeholder }: SearchableSelectProps) => {
  const sanitizedOptions = options
    .map((option) => option.trim())
    .filter((option) => option.length > 0);

  if (sanitizedOptions.length === 0) {
    return (
      <Input
        id={id}
        value={value}
        onChange={(event) => onChange(event.target.value)}
        placeholder={placeholder}
      />
    );
  }

  if (sanitizedOptions.length > 3) {
    const dataListId = `${id}-options`;
    return (
      <>
        <Input
          id={id}
          list={dataListId}
          value={value}
          onChange={(event) => onChange(event.target.value)}
          placeholder={placeholder}
        />
        <datalist id={dataListId}>
          {sanitizedOptions.map((option) => (
            <option key={option} value={option} />
          ))}
        </datalist>
      </>
    );
  }

  return (
    <Select value={value} onValueChange={onChange}>
      <SelectTrigger id={id}>
        <SelectValue placeholder={placeholder} />
      </SelectTrigger>
      <SelectContent>
        {sanitizedOptions.map((option) => (
          <SelectItem key={option} value={option}>
            {option}
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
  );
};

interface Employee {
  id: string;
  // 基本信息 - 对应Excel模板字段
  employeeId?: string;        // 工号
  name: string;               // 姓名
  department: string;         // 部门
  position: string;           // 岗位
  gender?: string;            // 性别
  hireDate: string;           // 入职时间
  age?: string;               // 年龄
  workYears?: string;         // 工龄
  birthMonth?: string;        // 出生月份
  education?: string;         // 文化程度
  politicalStatus?: string;   // 政治面貌
  workClothingSize?: string;  // 工作服
  safetyShoeSize?: string;    // 劳保鞋
  householdType?: string;     // 户口性质
  ethnicity?: string;         // 民族
  nativePlace?: string;       // 籍贯
  idAddress?: string;         // 身份证地址
  idNumber: string;           // 身份证号码
  maritalStatus?: string;     // 婚姻状况
  insuranceStatus?: string;   // 参保状态（手动覆盖）
  hasBirth?: string;          // 是否生育
  phone?: string;             // 联系电话
  emergencyContact?: string;  // 紧急联系人
  emergencyPhone?: string;    // 家庭电话/紧急情况联系电话
  currentAddress?: string;    // 现居住地址
  graduateSchool?: string;    // 毕业院校
  major?: string;             // 专业
  graduationTime?: string;    // 毕业时间
  socialInsuranceNumber?: string; // 社保编号
  providentFundNumber?: string;   // 公积金编号

  // 系统字段
  email?: string;
  remarks?: string;
  status: 'active' | 'resigned';
  resignDate?: string;
  resignProofName?: string;
  resignProofUrl?: string;
  resignReasons?: string[];
}

const normalizeApiString = (value?: string | null) => (value ?? "").trim();

const generateClientId = () => {
  if (typeof crypto !== "undefined" && "randomUUID" in crypto) {
    return crypto.randomUUID();
  }
  return `${Date.now()}-${Math.random().toString(16).slice(2)}`;
};

const MS_PER_DAY = 1000 * 60 * 60 * 24;
const MS_PER_YEAR = MS_PER_DAY * 365.25;

const formatDateToInput = (date: Date) => {
  const year = date.getFullYear();
  const month = String(date.getMonth() + 1).padStart(2, "0");
  const day = String(date.getDate()).padStart(2, "0");
  return `${year}-${month}-${day}`;
};

const normalizeDateInput = (value?: string | null) => {
  if (!value) return "";
  let cleaned = value.trim();
  if (!cleaned) return "";
  cleaned = cleaned
    .replace(/[年\.\/]/g, "-")
    .replace(/月/g, "-")
    .replace(/日/g, "")
    .replace(/\s+/g, "");
  cleaned = cleaned.replace(/--+/g, "-");
  cleaned = cleaned.replace(/^-/, "");
  cleaned = cleaned.replace(/-$/, "");
  if (/^\d{8}$/.test(cleaned)) {
    return `${cleaned.slice(0, 4)}-${cleaned.slice(4, 6)}-${cleaned.slice(6, 8)}`;
  }

  const yearMonthMatch = cleaned.match(/^(\d{4})-(\d{1,2})$/);
  if (yearMonthMatch) {
    const [, y, m] = yearMonthMatch;
    return `${y}-${m.padStart(2, "0")}-01`;
  }

  const monthDayMatch = cleaned.match(/^(\d{1,2})-(\d{1,2})$/);
  if (monthDayMatch) {
    const today = new Date();
    const [, m, d] = monthDayMatch;
    return `${today.getFullYear()}-${m.padStart(2, "0")}-${d.padStart(2, "0")}`;
  }

  const explicitYearMatch = cleaned.match(/^(\d{4})-(\d{1,2})-(\d{1,2})/);
  if (explicitYearMatch) {
    const [, y, m, d] = explicitYearMatch;
    return `${y}-${m.padStart(2, "0")}-${d.padStart(2, "0")}`;
  }

  const ambiguousMatch = cleaned.match(/^(\d{1,2})-(\d{1,2})-(\d{2,4})$/);
  if (ambiguousMatch) {
    let [monthPart, dayPart, yearPart] = ambiguousMatch.slice(1);
    if (yearPart.length === 2) {
      const numericYear = Number(yearPart);
      yearPart = (numericYear >= 70 ? 1900 + numericYear : 2000 + numericYear).toString();
    }

    if (Number(monthPart) > 12 && Number(dayPart) <= 12) {
      [monthPart, dayPart] = [dayPart, monthPart];
    }

    return `${yearPart.padStart(4, "0")}-${monthPart.padStart(2, "0")}-${dayPart.padStart(2, "0")}`;
  }

  const parsed = new Date(cleaned.replace(/-/g, "/"));
  if (!Number.isNaN(parsed.getTime())) {
    return formatDateToInput(parsed);
  }

  return cleaned.slice(0, 10);
};

const normalizeIdKey = (value?: string | null) => {
  if (!value) {
    return "";
  }
  return value.replace(/\s+/g, "").toUpperCase();
};

const formatAgeValue = (value?: string | null) => {
  if (!value) return "";
  const num = parseInt(value, 10);
  if (Number.isNaN(num) || num <= 0) {
    return "";
  }
  return String(num);
};

const formatWorkYearsValue = (value?: string | null) => {
  if (!value) return "";
  const num = Number(value);
  if (Number.isNaN(num) || num < 0) {
    return "";
  }
  return num.toFixed(1);
};

const normalizeBirthMonth = (value?: string | null) => {
  if (!value) return "";
  const trimmed = value.trim();
  if (!trimmed) return "";
  const normalized = normalizeDateInput(trimmed);
  if (normalized) {
    return normalized;
  }
  if (/^\d{6}$/.test(trimmed)) {
    return `${trimmed.slice(0, 4)}-${trimmed.slice(4, 6)}-01`;
  }
  if (/^\d{4}$/.test(trimmed)) {
    return `${trimmed}-01-01`;
  }
  if (/^(\d{1,2})月(\d{1,2})日$/.test(trimmed)) {
    const match = trimmed.match(/^(\d{1,2})月(\d{1,2})日$/);
    if (match) {
      const [, month, day] = match;
      const year = new Date().getFullYear();
      return `${year}-${month.padStart(2, "0")}-${day.padStart(2, "0")}`;
    }
  }
  if (/^\d{1,2}月$/.test(trimmed)) {
    const month = trimmed.replace("月", "");
    const year = new Date().getFullYear();
    return `${year}-${month.padStart(2, "0")}-01`;
  }
  return trimmed;
};

const displayDate = (value?: string | null) => {
  if (!value) {
    return "-";
  }
  const normalized = normalizeDateInput(value);
  return normalized || "-";
};

const downloadBlob = (blob: Blob, filename: string) => {
  const blobUrl = URL.createObjectURL(blob);
  const link = document.createElement("a");
  link.href = blobUrl;
  link.download = filename;
  document.body.appendChild(link);
  link.click();
  document.body.removeChild(link);
  URL.revokeObjectURL(blobUrl);
};

const formatBirthDateDisplay = (value?: string | null) => {
  const normalized = normalizeDateInput(value ?? "");
  if (!normalized) {
    return value ?? "-";
  }
  return normalized;
};

const getEmployeeSortValue = (employee: Employee, columnId: string): unknown => {
  switch (columnId) {
    case "resignDate":
      return normalizeDateInput(employee.resignDate ?? "");
    case "resignReasons":
      return (employee.resignReasons && employee.resignReasons.length > 0)
        ? employee.resignReasons.join("，")
        : "";
    case "resignProof":
      return employee.resignProofName ?? "";
    default: {
      if (columnId in employee) {
        const value = employee[columnId as keyof Employee];
        if (columnId === "hireDate" || columnId === "graduationTime" || columnId === "birthMonth") {
          return normalizeDateInput(typeof value === "string" ? value : String(value ?? ""));
        }
        return value ?? "";
      }
      return "";
    }
  }
};

const getProvidentSortValue = (record: ProvidentFundRecord, columnId: ProvidentColumnId): unknown => {
  switch (columnId) {
    case "personal_account":
      return record.personal_account ?? "";
    case "name":
      return record.name ?? "";
    case "identity_number":
      return record.identity_number ?? "";
    case "personal_base":
      return record.personal_base ?? 0;
    case "contribution_ratio":
      return record.contribution_ratio ?? DEFAULT_PROVIDENT_RATIO;
    case "personal_amount":
      return record.personal_amount ?? 0;
    case "company_amount":
      return record.company_amount ?? 0;
    case "total_amount":
      return record.total_amount ?? 0;
    case "status":
      return record.status;
    case "sealed_at":
      return normalizeDateInput(record.sealed_at ?? "") ?? "";
    case "created_at":
      return normalizeDateInput(record.created_at ?? "") ?? "";
    case "notes":
      return record.notes ?? "";
    default:
      return "";
  }
};

const renderSortIndicator = <T extends string>(sortState: TableSortState<T>, columnId: T) => {
  if (sortState.key !== columnId) {
    return null;
  }
  return (
    <span className="text-xs text-muted-foreground">
      {sortState.direction === "asc" ? "↑" : "↓"}
    </span>
  );
};

const getEmployeeCellClass = (fieldKey: keyof Employee) => {
  if (fieldKey === "name") {
    return "font-medium";
  }
  if (fieldKey === "idNumber") {
    return "font-mono text-sm";
  }
  return "text-sm";
};



const formatFieldDisplayValue = (employee: Employee, fieldKey: keyof Employee) => {
  const raw = employee[fieldKey];
  if (!raw) {
    return "-";
  }
  if (fieldKey === "hireDate") {
    return displayDate(raw as string);
  }
  if (fieldKey === "graduationTime" || fieldKey === "resignDate") {
    return displayDate(raw as string);
  }
  if (fieldKey === "birthMonth") {
    return formatBirthDateDisplay(raw as string);
  }
  if (fieldKey === "age") {
    const normalizedAge = formatAgeValue(raw as string);
    return normalizedAge || "-";
  }
  if (fieldKey === "workYears") {
    return formatWorkYearsValue(raw as string) || "-";
  }
  return raw;
};

const getResignReasonLabel = (value: string) => {
  const match = RESIGN_REASON_OPTIONS.find((option) => option.value === value);
  return match ? match.label : value;
};

const renderInsuranceStatusBadge = (label: string) => {
  if (!label || label === "-") {
    return "-";
  }
  let colorClass = "border-amber-200 bg-amber-50 text-amber-700";
  if (label.startsWith("已")) {
    colorClass = "border-emerald-200 bg-emerald-50 text-emerald-700";
  } else if (label.startsWith("未")) {
    colorClass = "border-rose-200 bg-rose-50 text-rose-700";
  }
  return (
    <Badge variant="outline" className={`px-2 py-0.5 text-xs font-semibold ${colorClass}`}>
      {label}
    </Badge>
  );
};

const formatAmountValue = (value?: number | null) => {
  if (value === null || value === undefined || Number.isNaN(value)) {
    return "-";
  }
  return value.toLocaleString("zh-CN", {
    minimumFractionDigits: 2,
    maximumFractionDigits: 2,
  });
};

const getCurrentMonthLabel = () => {
  const now = new Date();
  const year = now.getFullYear();
  const month = String(now.getMonth() + 1).padStart(2, "0");
  return `${year}-${month}`;
};

const DEFAULT_PROVIDENT_RATIO = 6;

const formatRatioDisplay = (value?: number | string, fallback = DEFAULT_PROVIDENT_RATIO) => {
  const numericValue = typeof value === "string" ? Number.parseFloat(value) : value;
  const ratio = Number.isFinite(numericValue) ? Number(numericValue) : fallback;
  return `${ratio}%`;
};

interface ProvidentFormState {
  personal_account: string;
  name: string;
  identity_number: string;
  personal_base: string;
  personal_amount: string;
  company_amount: string;
  contribution_ratio: string;
  notes: string;
}

const createEmptyProvidentForm = (): ProvidentFormState => ({
  personal_account: "",
  name: "",
  identity_number: "",
  personal_base: "",
  personal_amount: "",
  company_amount: "",
  contribution_ratio: DEFAULT_PROVIDENT_RATIO.toString(),
  notes: "",
});

const mapRecordToFormState = (record: ProvidentFundRecord): ProvidentFormState => ({
  personal_account: record.personal_account ?? "",
  name: record.name ?? "",
  identity_number: record.identity_number ?? "",
  personal_base: record.personal_base?.toString() ?? "",
  personal_amount: record.personal_amount?.toString() ?? "",
  company_amount: record.company_amount?.toString() ?? "",
  contribution_ratio: (record.contribution_ratio ?? DEFAULT_PROVIDENT_RATIO).toString(),
  notes: record.notes ?? "",
});

const renderProvidentStatusBadge = (status: ProvidentFundRecord["status"]) => {
  const label = status === "sealed" ? "已封存" : "在缴";
  const colorClass = status === "sealed"
    ? "border-rose-200 bg-rose-50 text-rose-700"
    : "border-emerald-200 bg-emerald-50 text-emerald-700";
  return (
    <Badge variant="outline" className={`px-2 py-0.5 text-xs font-semibold ${colorClass}`}>
      {label}
    </Badge>
  );
};

const getProvidentCellValue = (record: ProvidentFundRecord, columnId: ProvidentColumnId) => {
  switch (columnId) {
    case "personal_account":
      return record.personal_account || "-";
    case "name":
      return record.name || "-";
    case "identity_number":
      return record.identity_number || "-";
    case "personal_base":
      return formatAmountValue(record.personal_base);
    case "contribution_ratio":
      return formatRatioDisplay(record.contribution_ratio);
    case "personal_amount":
      return formatAmountValue(record.personal_amount);
    case "company_amount":
      return formatAmountValue(record.company_amount);
    case "total_amount":
      return formatAmountValue(record.total_amount);
    case "contribution_ratio":
      return formatRatioDisplay(record.contribution_ratio);
    case "status":
      return renderProvidentStatusBadge(record.status);
    case "sealed_at":
      return record.sealed_at ? displayDate(record.sealed_at) : "-";
    case "created_at":
      return record.created_at ? displayDate(record.created_at) : "-";
    case "notes":
      return record.notes?.trim() || "-";
    default:
      return "-";
  }
};

const getProvidentPrintableValue = (record: ProvidentFundRecord, columnId: ProvidentColumnId) => {
  switch (columnId) {
    case "personal_base":
      return formatAmountValue(record.personal_base);
    case "personal_amount":
      return formatAmountValue(record.personal_amount);
    case "company_amount":
      return formatAmountValue(record.company_amount);
    case "total_amount":
      return formatAmountValue(record.total_amount);
    case "status":
      return record.status === "sealed" ? "已封存" : "在缴";
    case "sealed_at":
      return record.sealed_at ? displayDate(record.sealed_at) : "-";
    case "created_at":
      return record.created_at ? displayDate(record.created_at) : "-";
    case "notes":
      return record.notes?.trim() || "-";
    default:
      return getProvidentSortValue(record, columnId) ?? "-";
  }
};

type PrintDataset = {
  type: "active" | "resigned" | "insurance" | "provident";
  columns: string[];
  rows: string[][];
  defaultTitle: string;
};

const mapEmployeeFromApi = (employee: EmployeeResponse): Employee => ({
  id: employee.id ? String(employee.id) : generateClientId(),
  employeeId: normalizeApiString(employee.employee_id),
  name: normalizeApiString(employee.name),
  department: normalizeApiString(employee.department),
  position: normalizeApiString(employee.position),
  gender: normalizeApiString(employee.gender),
  hireDate: normalizeDateInput(employee.hire_date),
  age: formatAgeValue(employee.age),
  workYears: formatWorkYearsValue(employee.work_years),
  birthMonth: normalizeBirthMonth(employee.birth_month),
  education: normalizeApiString(employee.education),
  politicalStatus: normalizeApiString(employee.political_status),
  workClothingSize: normalizeApiString(employee.work_clothing_size),
  safetyShoeSize: normalizeApiString(employee.safety_shoe_size),
  householdType: normalizeApiString(employee.household_type),
  ethnicity: normalizeApiString(employee.ethnicity),
  nativePlace: normalizeApiString(employee.native_place),
  idAddress: normalizeApiString(employee.id_address),
  idNumber: normalizeApiString(employee.id_number),
  maritalStatus: normalizeApiString(employee.marital_status),
  insuranceStatus: normalizeApiString(employee.social_insurance),
  hasBirth: normalizeApiString(employee.has_birth),
  phone: normalizeApiString(employee.phone),
  emergencyContact: normalizeApiString(employee.emergency_contact),
  emergencyPhone: normalizeApiString(employee.emergency_phone),
  currentAddress: normalizeApiString(employee.current_address),
  graduateSchool: normalizeApiString(employee.graduate_school),
  major: normalizeApiString(employee.major),
  graduationTime: normalizeApiString(employee.graduation_time),
  socialInsuranceNumber: normalizeApiString(employee.social_insurance_number ?? undefined),
  providentFundNumber: normalizeApiString(employee.provident_fund_number ?? undefined) || "无",
  email: normalizeApiString(employee.email),
  remarks: normalizeApiString(employee.remarks),
  status: employee.status === "resigned" ? "resigned" : "active",
  resignDate: normalizeDateInput(employee.resign_date),
  resignProofName: normalizeApiString(employee.resign_proof_name ?? undefined),
  resignProofUrl: normalizeApiString(employee.resign_proof_url ?? undefined),
  resignReasons: (() => {
    if (!employee.resign_reasons) {
      return [];
    }
    try {
      const parsed = JSON.parse(employee.resign_reasons);
      if (Array.isArray(parsed)) {
        return parsed.filter((item): item is string => typeof item === "string" && item.trim().length > 0);
      }
    } catch (error) {
      console.error("[EmployeeManagement] failed to parse resign reasons", error);
    }
    return [];
  })(),
});

const applyDefaultInsuranceStatus = (employee: Employee): Employee => {
  const trimmed = employee.insuranceStatus?.trim();
  if (trimmed) {
    return { ...employee, insuranceStatus: trimmed };
  }
  const pendingLabel = employee.status === "resigned" ? "需办理退保" : "需办理参保";
  return { ...employee, insuranceStatus: pendingLabel };
};

const mapSocialInsuranceRecordFromApi = (record: SocialInsuranceRecordDTO): SocialInsuranceRecord => ({
  id: String(record.id),
  numericId: record.id,
  batchId: record.batch_id,
  changeType: record.change_type,
  employeeName: record.employee_name ?? "",
  department: record.department ?? "",
  identityNumber: record.identity_number ?? "",
  personalNumber: record.personal_number ?? "",
  effectiveDate: record.effective_date ?? "",
  reason: record.reason ?? "",
  templateValues: record.template_values ?? {},
  originalFileName: record.original_file_name ?? "",
  createdAt: record.created_at,
  updatedAt: record.updated_at,
});

const buildEmployeeStubFromRecord = (record: SocialInsuranceRecord): Employee => ({
  id: record.id,
  employeeId: record.personalNumber || "",
  name: record.employeeName,
  department: record.department,
  position: record.templateValues?.job ?? "",
  gender: "",
  hireDate: "",
  age: "",
  workYears: "",
  birthMonth: "",
  education: record.templateValues?.education ?? "",
  politicalStatus: "",
  workClothingSize: "",
  safetyShoeSize: "",
  householdType: record.templateValues?.householdType ?? "",
  ethnicity: record.templateValues?.nationality ?? "",
  nativePlace: "",
  idAddress: "",
  idNumber: record.identityNumber,
  maritalStatus: "",
  insuranceStatus: "",
  hasBirth: "",
  phone: record.templateValues?.phone ?? "",
  emergencyContact: "",
  emergencyPhone: "",
  currentAddress: "",
  graduateSchool: "",
  major: "",
  graduationTime: "",
  socialInsuranceNumber: record.personalNumber || "",
  providentFundNumber: "无",
  email: "",
  remarks: record.templateValues?.remark ?? "",
  status: "active",
  resignDate: "",
  resignProofName: "",
  resignProofUrl: "",
  resignReasons: [],
});

const createEmptyEmployee = (): Partial<Employee> => ({
  employeeId: "",
  name: "",
  department: "",
  position: "",
  gender: "",
  hireDate: formatDateToInput(new Date()),
  age: "",
  workYears: "",
  birthMonth: "",
  education: "",
  politicalStatus: "",
  workClothingSize: "",
  safetyShoeSize: "",
  householdType: "",
  ethnicity: "",
  nativePlace: "",
  idAddress: "",
  idNumber: "",
  maritalStatus: "",
  insuranceStatus: "",
  hasBirth: "",
  phone: "",
  emergencyContact: "",
  emergencyPhone: "",
  currentAddress: "",
  graduateSchool: "",
  major: "",
  graduationTime: "",
  socialInsuranceNumber: "",
  providentFundNumber: "无",
  email: "",
  remarks: "",
  resignReasons: [],
});

const sanitizeTemplateCell = (value: unknown): string => {
  if (value === null || value === undefined) {
    return "";
  }
  if (value instanceof Date) {
    return formatDateToInput(value);
  }
  if (typeof value === "number") {
    if (!Number.isFinite(value)) {
      return "";
    }
    if (Number.isInteger(value)) {
      return String(value);
    }
    return value.toString();
  }
  return String(value).trim();
};

const convertExcelSerialToDateString = (numeric: number): string => {
  if (!Number.isFinite(numeric)) {
    return "";
  }
  const decoded = XLSX.SSF.parse_date_code(numeric);
  if (!decoded || !decoded.y || !decoded.m || !decoded.d) {
    return "";
  }
  const year = String(decoded.y);
  const month = String(decoded.m).padStart(2, "0");
  const day = String(decoded.d).padStart(2, "0");
  return `${year}-${month}-${day}`;
};

const normalizeTemplateDateValue = (value: string): string => {
  const trimmed = value.trim();
  if (!trimmed) {
    return "";
  }
  if (/^\d+$/.test(trimmed)) {
    const numeric = Number(trimmed);
    const converted = convertExcelSerialToDateString(numeric);
    if (converted) {
      return converted;
    }
  }
  const replaced = trimmed.replace(/[\.\/年]/g, "-").replace(/月/g, "-").replace(/日/g, "");
  const normalized = normalizeDateInput(replaced);
  return normalized || replaced;
};


const HEADER_SENTINELS: Record<SocialInsuranceChangeType, string[]> = {
  increase: ["证件号码", "姓名"],
  decrease: ["证件号码", "姓名"],
};

const findHeaderRowIndex = (rows: string[][], changeType: SocialInsuranceChangeType) => {
  const sentinels = HEADER_SENTINELS[changeType];
  return rows.findIndex((row) => {
    if (!row || row.length === 0) {
      return false;
    }
    const trimmed = row.map((cell) => sanitizeTemplateCell(cell));
    return sentinels.every((item) => trimmed.includes(item));
  });
};

const parseSocialInsuranceTemplate = async (
  file: File,
  changeType: SocialInsuranceChangeType,
): Promise<SocialInsuranceImportRecordPayload[]> => {
  const buffer = await file.arrayBuffer();
  const workbook = XLSX.read(buffer, { type: "array", raw: false, cellDates: true });
  const sheetName = workbook.SheetNames[0];
  if (!sheetName) {
    throw new Error("未在模板中找到工作表");
  }
  const sheet = workbook.Sheets[sheetName];
  if (!sheet) {
    throw new Error("未在模板中找到工作表数据");
  }

  const rows = XLSX.utils.sheet_to_json<string[]>(sheet, {
    header: 1,
    blankrows: false,
    defval: "",
  });

  const headerRowIndex = findHeaderRowIndex(rows, changeType);
  if (headerRowIndex === -1) {
    throw new Error("未找到模板表头，请确认文件是否符合要求");
  }

  const fieldDefinitions = changeType === "increase" ? INCREASE_TEMPLATE_FIELDS : DECREASE_TEMPLATE_FIELDS;
  const headerRow = rows[headerRowIndex].map((cell) => sanitizeTemplateCell(cell));

  const records: SocialInsuranceImportRecordPayload[] = [];

  for (let rowIndex = headerRowIndex + 1; rowIndex < rows.length; rowIndex += 1) {
    const row = rows[rowIndex] ?? [];
    const sanitizedRow = row.map((cell) => sanitizeTemplateCell(cell));
    const containsInstructionRow = sanitizedRow.some((cell) => cell.includes("≤25个汉字"));
    if (containsInstructionRow) {
      continue;
    }
    const valueByHeader = new Map<string, string>();
    headerRow.forEach((header, columnIndex) => {
      if (!header) {
        return;
      }
      const rawValue = columnIndex < sanitizedRow.length ? sanitizedRow[columnIndex] : "";
      valueByHeader.set(header, rawValue);
    });

    const nameValue = valueByHeader.get("姓名") ?? "";
    const idValue = valueByHeader.get("证件号码") ?? "";
    if (!nameValue.trim() && !idValue.trim()) {
      continue;
    }

    const templateValues: Record<string, string> = {};
    fieldDefinitions.forEach((field) => {
      const raw = valueByHeader.get(field.header) ?? "";
      const normalized = field.isDate ? normalizeTemplateDateValue(raw) : raw;
      templateValues[field.key] = normalized;
    });

    const record: SocialInsuranceImportRecordPayload = {
      employee_name: templateValues.name ?? nameValue,
      department: "",
      identity_number: templateValues.idNumber ?? idValue,
      personal_number: templateValues.personalNumber ?? "",
      effective_date:
        changeType === "increase"
          ? templateValues.pensionStartDate ?? ""
          : templateValues.decreaseDate ?? "",
      reason:
        changeType === "increase"
          ? templateValues.remark ?? ""
          : templateValues.decreaseReason ?? "",
      template_values: templateValues,
    };

    records.push(record);
  }

  if (records.length === 0) {
    throw new Error("未解析到任何有效数据，请确认模板中包含有效的员工记录");
  }

  return records;
};

type SocialInsuranceChangeType = ApiSocialInsuranceChangeType;

interface SocialInsuranceRecord {
  id: string;
  numericId: number;
  batchId: number;
  changeType: SocialInsuranceChangeType;
  employeeName: string;
  department: string;
  identityNumber: string;
  personalNumber: string;
  effectiveDate: string;
  reason: string;
  templateValues: Record<string, string>;
  originalFileName?: string;
  createdAt: string;
  updatedAt: string;
}

interface TemplateFieldDefinition {
  key: string;
  header: string;
  label: string;
  appliesTo: SocialInsuranceChangeType[];
  includeInTable?: boolean;
  isDate?: boolean;
}

interface SocialInsuranceIncreaseForm {
  nationality: string;
  firstWorkDate: string;
  baseSalary: string;
  job: string;
  personalIdentity: string;
  householdType: string;
  phone: string;
  education: string;
  pensionStartDate: string;
  unemploymentStartDate: string;
  medicalStartDate: string;
  injuryStartDate: string;
  maternityStartDate: string;
  remark: string;
  specialSkill: string;
  skillLevel: string;
  personalNumber: string;
}

interface SocialInsuranceDecreaseForm {
  personalNumber: string;
  decreaseDate: string;
  decreaseReason: string;
  pensionFlag: string;
  unemploymentFlag: string;
  medicalFlag: string;
  injuryFlag: string;
  maternityFlag: string;
  unemploymentReason: string;
}


const INCREASE_TEMPLATE_FIELDS: TemplateFieldDefinition[] = [
  { key: "idNumber", header: "证件号码", label: "证件号码", appliesTo: ["increase"], includeInTable: false },
  { key: "name", header: "姓名", label: "姓名", appliesTo: ["increase"], includeInTable: false },
  { key: "nationality", header: "民族", label: "民族", appliesTo: ["increase"] },
  { key: "firstWorkDate", header: "首次参加工作日期", label: "首次参加工作日期", appliesTo: ["increase"], isDate: true },
  { key: "baseSalary", header: "月基本工资额", label: "月基本工资额", appliesTo: ["increase"] },
  { key: "job", header: "工种", label: "工种", appliesTo: ["increase"] },
  { key: "personalIdentity", header: "个人身份", label: "个人身份", appliesTo: ["increase"] },
  { key: "householdType", header: "户口性质", label: "户口性质", appliesTo: ["increase"] },
  { key: "phone", header: "联系电话", label: "联系电话", appliesTo: ["increase"] },
  { key: "education", header: "文化程度", label: "文化程度", appliesTo: ["increase"] },
  { key: "pensionStartDate", header: "养老保险参保时间", label: "养老保险参保时间", appliesTo: ["increase"], isDate: true },
  { key: "unemploymentStartDate", header: "失业保险参保时间", label: "失业保险参保时间", appliesTo: ["increase"], isDate: true },
  { key: "medicalStartDate", header: "医疗保险参保时间", label: "医疗保险参保时间", appliesTo: ["increase"], isDate: true },
  { key: "injuryStartDate", header: "工伤保险参保时间", label: "工伤保险参保时间", appliesTo: ["increase"], isDate: true },
  { key: "maternityStartDate", header: "生育保险参保时间", label: "生育保险参保时间", appliesTo: ["increase"], isDate: true },
  { key: "remark", header: "备注", label: "备注", appliesTo: ["increase"] },
  { key: "specialSkill", header: "专职技能", label: "专职技能", appliesTo: ["increase"] },
  { key: "skillLevel", header: "技能等级", label: "技能等级", appliesTo: ["increase"] },
];

const DECREASE_TEMPLATE_FIELDS: TemplateFieldDefinition[] = [
  { key: "personalNumber", header: "个人编号", label: "个人编号", appliesTo: ["decrease"], includeInTable: false },
  { key: "idNumber", header: "证件号码", label: "证件号码", appliesTo: ["decrease"], includeInTable: false },
  { key: "name", header: "姓名", label: "姓名", appliesTo: ["decrease"], includeInTable: false },
  { key: "decreaseDate", header: "减少时间", label: "减少时间", appliesTo: ["decrease"], isDate: true },
  { key: "decreaseReason", header: "减少原因", label: "减少原因", appliesTo: ["decrease"], includeInTable: true },
  { key: "pensionFlag", header: "养老保险减少标志", label: "养老保险减少标志", appliesTo: ["decrease"] },
  { key: "unemploymentFlag", header: "失业保险减少标志", label: "失业保险减少标志", appliesTo: ["decrease"] },
  { key: "medicalFlag", header: "医疗保险减少标志", label: "医疗保险减少标志", appliesTo: ["decrease"] },
  { key: "injuryFlag", header: "工伤保险减少标志", label: "工伤保险减少标志", appliesTo: ["decrease"] },
  { key: "maternityFlag", header: "生育保险减少标志", label: "生育保险减少标志", appliesTo: ["decrease"] },
  { key: "unemploymentReason", header: "失业原因", label: "失业原因", appliesTo: ["decrease"] },
];

const ALL_TEMPLATE_FIELDS: TemplateFieldDefinition[] = [...INCREASE_TEMPLATE_FIELDS, ...DECREASE_TEMPLATE_FIELDS];

const BASE_INSURANCE_COLUMNS = [
  { id: "changeType", label: "类型" },
  { id: "employeeName", label: "姓名" },
  { id: "identityNumber", label: "证件号码" },
  { id: "personalNumber", label: "个人编号" },
  { id: "department", label: "部门" },
  { id: "effectiveDate", label: "生效日期" },
  { id: "reason", label: "变动原因" },
  { id: "createdAt", label: "录入时间" },
  { id: "originalFileName", label: "来源文件" },
] as const;

type InsuranceBaseColumnId = typeof BASE_INSURANCE_COLUMNS[number]["id"];

const TEMPLATE_INSURANCE_COLUMNS = ALL_TEMPLATE_FIELDS
  .filter((field) => field.includeInTable !== false)
  .map((field) => ({
    id: `template:${field.key}` as const,
    field,
  }));

type InsuranceTemplateColumnId = typeof TEMPLATE_INSURANCE_COLUMNS[number]["id"];

type InsuranceColumnId = InsuranceBaseColumnId | InsuranceTemplateColumnId;

type InsuranceViewKey = Extract<InsuranceView, "increase" | "decrease">;
const REQUIRED_INSURANCE_COLUMNS: InsuranceColumnId[] = ["changeType", "employeeName", "identityNumber"];

const TEMPLATE_ENTRIES_BY_VIEW: Record<InsuranceViewKey, typeof TEMPLATE_INSURANCE_COLUMNS> = {
  increase: TEMPLATE_INSURANCE_COLUMNS.filter((entry) => entry.field.appliesTo.includes("increase")),
  decrease: TEMPLATE_INSURANCE_COLUMNS.filter((entry) => entry.field.appliesTo.includes("decrease")),
};

const ALLOWED_INSURANCE_COLUMNS_BY_VIEW: Record<InsuranceViewKey, InsuranceColumnId[]> = {
  increase: [
    ...BASE_INSURANCE_COLUMNS.map((column) => column.id),
    ...TEMPLATE_ENTRIES_BY_VIEW.increase.map((entry) => entry.id),
  ],
  decrease: [
    ...BASE_INSURANCE_COLUMNS.map((column) => column.id),
    ...TEMPLATE_ENTRIES_BY_VIEW.decrease.map((entry) => entry.id),
  ],
};

const DEFAULT_INSURANCE_COLUMN_ORDER: InsuranceColumnId[] = [
  ...BASE_INSURANCE_COLUMNS.map((column) => column.id),
  ...TEMPLATE_INSURANCE_COLUMNS.map((column) => column.id),
];

const INCREASE_LIST_COLUMN_ORDER: InsuranceColumnId[] = [
  "identityNumber",
  "employeeName",
  "template:nationality",
  "template:firstWorkDate",
  "template:baseSalary",
  "template:job",
  "template:personalIdentity",
  "template:householdType",
  "template:phone",
  "template:education",
  "template:pensionStartDate",
  "template:unemploymentStartDate",
  "template:medicalStartDate",
  "template:injuryStartDate",
  "template:maternityStartDate",
  "template:specialSkill",
  "template:skillLevel",
  "template:remark",
];

const DECREASE_LIST_COLUMN_ORDER: InsuranceColumnId[] = [
  "identityNumber",
  "employeeName",
  "personalNumber",
  "template:decreaseDate",
  "template:decreaseReason",
  "template:pensionFlag",
  "template:unemploymentFlag",
  "template:medicalFlag",
  "template:injuryFlag",
  "template:maternityFlag",
  "template:unemploymentReason",
];

const REDUCTION_FLAG_LABELS: Record<"pensionFlag" | "unemploymentFlag" | "medicalFlag" | "injuryFlag" | "maternityFlag", string> = {
  pensionFlag: "养老保险减少标志",
  unemploymentFlag: "失业保险减少标志",
  medicalFlag: "医疗保险减少标志",
  injuryFlag: "工伤保险减少标志",
  maternityFlag: "生育保险减少标志",
};

const DEFAULT_INSURANCE_VISIBLE_MAP: Record<InsuranceViewKey, InsuranceColumnId[]> = {
  increase: [...INCREASE_LIST_COLUMN_ORDER],
  decrease: [...DECREASE_LIST_COLUMN_ORDER],
};

const sanitizeInsuranceVisibleList = (list: InsuranceColumnId[] | undefined, view: InsuranceViewKey): InsuranceColumnId[] => {
  const allowedSet = new Set(ALLOWED_INSURANCE_COLUMNS_BY_VIEW[view]);
  const fallback = DEFAULT_INSURANCE_VISIBLE_MAP[view];
  const preferred = Array.isArray(list) && list.length ? list : fallback;
  const normalized = preferred.filter((id) => allowedSet.has(id));
  const ensured = normalized.length > 0 ? normalized : fallback.filter((id) => allowedSet.has(id));
  const combined = Array.from(new Set([...REQUIRED_INSURANCE_COLUMNS, ...ensured]));
  return combined.filter((id) => allowedSet.has(id));
};

const sanitizeInsuranceVisibleMap = (
  source?: Partial<Record<InsuranceViewKey, InsuranceColumnId[]>>,
): Record<InsuranceViewKey, InsuranceColumnId[]> => ({
  increase: sanitizeInsuranceVisibleList(source?.increase, "increase"),
  decrease: sanitizeInsuranceVisibleList(source?.decrease, "decrease"),
});

const parseInsuranceVisiblePreference = (raw: unknown): Record<InsuranceViewKey, InsuranceColumnId[]> | null => {
  if (!raw || typeof raw !== "object") {
    return null;
  }
  const payload = raw as { visible?: unknown };
  const visibleValue = payload.visible;
  if (Array.isArray(visibleValue)) {
    return sanitizeInsuranceVisibleMap({
      increase: visibleValue as InsuranceColumnId[],
      decrease: visibleValue as InsuranceColumnId[],
    });
  }
  if (visibleValue && typeof visibleValue === "object") {
    const record = visibleValue as Record<string, unknown>;
    const next: Partial<Record<InsuranceViewKey, InsuranceColumnId[]>> = {};
    if (Array.isArray(record.increase)) {
      next.increase = record.increase as InsuranceColumnId[];
    }
    if (Array.isArray(record.decrease)) {
      next.decrease = record.decrease as InsuranceColumnId[];
    }
    return sanitizeInsuranceVisibleMap(next);
  }
  return null;
};

interface EmployeeManagementProps {
  className?: string;
  initialTab?: RosterTab;
}

const formatDateForTemplate = (value: string) => {
  const normalized = normalizeDateInput(value);
  if (!normalized) {
    return "";
  }
  const [year, month, day] = normalized.split("-");
  return `${Number(year)}/${Number(month)}/${Number(day)}`;
};

const normalizeTemplateCellValue = (value: unknown): string => {
  if (value === null || value === undefined) {
    return "";
  }
  if (typeof value === "string") {
    return value;
  }
  if (typeof value === "number") {
    return Number.isFinite(value) ? `${value}` : "";
  }
  if (typeof value === "boolean") {
    return value ? "是" : "否";
  }
  if (value instanceof Date) {
    return formatDateForTemplate(formatDateToInput(value));
  }
  if (typeof value === "object") {
    const maybeValue = (value as { value?: unknown }).value ?? (value as { text?: unknown }).text;
    if (maybeValue !== undefined) {
      return normalizeTemplateCellValue(maybeValue);
    }
    try {
      return JSON.stringify(value);
    } catch {
      return String(value);
    }
  }
  return String(value);
};

const getNextMonthFirstDay = () => {
  const now = new Date();
  const nextMonthFirstDay = new Date(now.getFullYear(), now.getMonth() + 1, 1);
  return formatDateToInput(nextMonthFirstDay);
};

const createEmptyIncreaseForm = (): SocialInsuranceIncreaseForm => ({
  nationality: "",
  firstWorkDate: "",
  baseSalary: "",
  job: "",
  personalIdentity: "",
  householdType: "",
  phone: "",
  education: "71.初级中学",
  pensionStartDate: "",
  unemploymentStartDate: "",
  medicalStartDate: "",
  injuryStartDate: "",
  maternityStartDate: getNextMonthFirstDay(),
  remark: "",
  specialSkill: "",
  skillLevel: "",
  personalNumber: "",
});

const createEmptyDecreaseForm = (): SocialInsuranceDecreaseForm => ({
  personalNumber: "",
  decreaseDate: "",
  decreaseReason: "",
  pensionFlag: "",
  unemploymentFlag: "",
  medicalFlag: "",
  injuryFlag: "",
  maternityFlag: "",
  unemploymentReason: "",
});

const DetailField = ({ label, value }: { label: string; value: ReactNode }) => (
  <div className="flex flex-col gap-1 rounded-md border border-dashed border-muted-foreground/30 p-3">
    <span className="text-xs text-muted-foreground">{label}</span>
    <span className="text-sm font-medium break-words">{value && String(value).trim() ? value : "-"}</span>
  </div>
);

type SocialOptionSelectableKey =
  | "personal_identity"
  | "household_type"
  | "education_level"
  | "special_skill"
  | "skill_level"
  | "decrease_reason"
  | "unemployment_reason"
  | "reduction_flag";

type IncreaseFieldKey = keyof SocialInsuranceIncreaseForm;
type DecreaseFieldKey = keyof SocialInsuranceDecreaseForm;

type InsuranceFieldInputType = "text" | "number" | "date" | "textarea" | "select";

interface BaseInsuranceFieldConfig<Key> {
  key: Key;
  label: string;
  input: InsuranceFieldInputType;
  placeholder?: string;
  required?: boolean;
  fullWidth?: boolean;
  selectKey?: SocialOptionSelectableKey;
  inputMode?: "text" | "decimal" | "tel";
}

const INCREASE_FORM_FIELD_CONFIGS: BaseInsuranceFieldConfig<IncreaseFieldKey>[] = [
  { key: "nationality", label: "民族", input: "text", placeholder: "例如：01.汉族" },
  { key: "firstWorkDate", label: "首次参加工作日期", input: "date", placeholder: "选择日期" },
  { key: "baseSalary", label: "月基本工资额", input: "number", placeholder: "例如：2330", required: true, inputMode: "decimal" },
  { key: "job", label: "工种", input: "text", placeholder: "如：操作工" },
  { key: "personalIdentity", label: "个人身份", input: "select", selectKey: "personal_identity", required: true },
  { key: "householdType", label: "户口性质", input: "select", selectKey: "household_type", required: true },
  { key: "phone", label: "联系电话", input: "text", placeholder: "11位手机号", inputMode: "tel" },
  { key: "education", label: "文化程度", input: "select", selectKey: "education_level", required: true },
  { key: "pensionStartDate", label: "养老保险参保时间", input: "date", required: true },
  { key: "unemploymentStartDate", label: "失业保险参保时间", input: "date" },
  { key: "medicalStartDate", label: "医疗保险参保时间", input: "date" },
  { key: "injuryStartDate", label: "工伤保险参保时间", input: "date" },
  { key: "maternityStartDate", label: "生育保险参保时间", input: "date" },
  { key: "specialSkill", label: "专职技能", input: "select", selectKey: "special_skill" },
  { key: "skillLevel", label: "技能等级", input: "select", selectKey: "skill_level" },
  { key: "personalNumber", label: "个人编号", input: "text", placeholder: "便于对账" },
  { key: "remark", label: "备注", input: "textarea", placeholder: "可填写参保说明", fullWidth: true },
];

const DECREASE_FORM_FIELD_CONFIGS: BaseInsuranceFieldConfig<DecreaseFieldKey>[] = [
  { key: "personalNumber", label: "个人编号", input: "text", placeholder: "请输入个人编号", required: true },
  { key: "decreaseDate", label: "减少时间", input: "date", required: true },
  { key: "decreaseReason", label: "减少原因", input: "select", selectKey: "decrease_reason", required: true },
  { key: "pensionFlag", label: REDUCTION_FLAG_LABELS.pensionFlag, input: "select", selectKey: "reduction_flag" },
  { key: "unemploymentFlag", label: REDUCTION_FLAG_LABELS.unemploymentFlag, input: "select", selectKey: "reduction_flag" },
  { key: "medicalFlag", label: REDUCTION_FLAG_LABELS.medicalFlag, input: "select", selectKey: "reduction_flag" },
  { key: "injuryFlag", label: REDUCTION_FLAG_LABELS.injuryFlag, input: "select", selectKey: "reduction_flag" },
  { key: "maternityFlag", label: REDUCTION_FLAG_LABELS.maternityFlag, input: "select", selectKey: "reduction_flag" },
  { key: "unemploymentReason", label: "失业原因", input: "select", selectKey: "unemployment_reason", fullWidth: true },
];

const RESPONSIVE_DIALOG_CLASS =
  DIALOG_SIZES.full + " overflow-y-auto px-4 py-4 sm:px-6 sm:py-6";
const RESPONSIVE_FIELD_GRID_CLASS =
  "grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3 2xl:grid-cols-4 [&>div]:min-w-0 [&>div]:w-full sm:[&>div]:min-w-[260px]";
const CALLBACK_PERSONAL_MAP_STORAGE_KEY = "insurance-callback-map";

export function EmployeeManagement({ initialTab = "active" }: EmployeeManagementProps) {
  // 状态管理
  const [employees, setEmployees] = useState<Employee[]>([]);
  const [resignedEmployees, setResignedEmployees] = useState<Employee[]>([]);
  const [socialInsuranceChanges, setSocialInsuranceChanges] = useState<SocialInsuranceRecord[]>([]);
  const [loadingInsurance, setLoadingInsurance] = useState(false);
  const [isLoading, setIsLoading] = useState(false);
  const [importMode, setImportMode] = useState<EmployeeImportMode>("merge");
  const [exporting, setExporting] = useState(false);
  const [downloadingTemplate, setDownloadingTemplate] = useState(false);
  const [selectedEmployeeIds, setSelectedEmployeeIds] = useState<string[]>([]);
  const [selectedResignedIds, setSelectedResignedIds] = useState<string[]>([]);
  const [selectedInsuranceChangeIds, setSelectedInsuranceChangeIds] = useState<string[]>([]);
  const [resignedExporting, setResignedExporting] = useState(false);
  const [insuranceExporting, setInsuranceExporting] = useState(false);
  const [insuranceSearch, setInsuranceSearch] = useState("");
  const [insuranceView, setInsuranceView] = useState<InsuranceView>("increase");
  const [activeRosterTab, setActiveRosterTab] = useState<RosterTab>(initialTab);
  const [insuranceFormMode, setInsuranceFormMode] = useState<"create" | "edit">("create");
  const [socialOptions, setSocialOptions] = useState<SocialInsuranceTemplateOptions | null>(null);
  const [socialOptionsLoading, setSocialOptionsLoading] = useState(false);
  const [socialOptionsError, setSocialOptionsError] = useState("");
  const [showInsuranceFormDialog, setShowInsuranceFormDialog] = useState(false);
  const [insuranceFormType, setInsuranceFormType] = useState<SocialInsuranceChangeType>("increase");
  const [insuranceFormEmployee, setInsuranceFormEmployee] = useState<Employee | null>(null);
  const [editingInsuranceRecord, setEditingInsuranceRecord] = useState<SocialInsuranceRecord | null>(null);
  const [insuranceIncreaseForm, setInsuranceIncreaseForm] = useState<SocialInsuranceIncreaseForm>(() => createEmptyIncreaseForm());
  const [insuranceDecreaseForm, setInsuranceDecreaseForm] = useState<SocialInsuranceDecreaseForm>(() => createEmptyDecreaseForm());
  const [insuranceFormSubmitting, setInsuranceFormSubmitting] = useState(false);
  const [insuranceFormError, setInsuranceFormError] = useState("");
  const [duplicateWarning, setDuplicateWarning] = useState<{ type: SocialInsuranceChangeType; name: string } | null>(null);
  const [callbackPersonalMap, setCallbackPersonalMap] = useState<Record<string, string>>({});
  const [providentRecords, setProvidentRecords] = useState<ProvidentFundRecord[]>([]);
  const [providentBills, setProvidentBills] = useState<ProvidentFundBill[]>([]);
  const [providentLoading, setProvidentLoading] = useState(false);
  const [providentBillsLoading, setProvidentBillsLoading] = useState(false);
  const [providentImporting, setProvidentImporting] = useState(false);
  const [importedProvidentFileName, setImportedProvidentFileName] = useState<string>("");
  const [providentSearch, setProvidentSearch] = useState("");
  const [providentStatusFilter, setProvidentStatusFilter] = useState<"all" | "active" | "sealed">("all");
  const [selectedProvidentIds, setSelectedProvidentIds] = useState<number[]>([]);
  const [providentSort, setProvidentSort] = useState<TableSortState<ProvidentColumnId>>(() => {
    return readLocalStorageJSON<TableSortState<ProvidentColumnId>>(PROVIDENT_SORT_STORAGE_KEY) ?? {
      key: "personal_account",
      direction: "asc",
    };
  });
  const [providentVisibleColumns, setProvidentVisibleColumns] = useState<ProvidentColumnId[]>(() => {
    return readLocalStorageJSON<ProvidentColumnId[]>(PROVIDENT_VISIBLE_COLUMNS_STORAGE_KEY) ?? [...DEFAULT_PROVIDENT_VISIBLE_COLUMNS];
  });
  const [providentColumnOrder, setProvidentColumnOrder] = useState<ProvidentColumnId[]>(() => {
    return readLocalStorageJSON<ProvidentColumnId[]>(PROVIDENT_COLUMN_ORDER_STORAGE_KEY) ?? [...DEFAULT_PROVIDENT_COLUMN_ORDER];
  });
  const [showProvidentFieldSelector, setShowProvidentFieldSelector] = useState(false);
  const [showProvidentDialog, setShowProvidentDialog] = useState(false);
  const [providentFormMode, setProvidentFormMode] = useState<"create" | "edit">("create");
  const [providentForm, setProvidentForm] = useState<ProvidentFormState>(() => createEmptyProvidentForm());
  const [editingProvidentRecord, setEditingProvidentRecord] = useState<ProvidentFundRecord | null>(null);
  const [savingProvidentRecord, setSavingProvidentRecord] = useState(false);
  const [providentSettings, setProvidentSettings] = useState<ProvidentFundSettings | null>(null);
  const [providentSettingsDraft, setProvidentSettingsDraft] = useState<ProvidentFundSettings>({
    unit_name: DEFAULT_PROVIDENT_UNIT_NAME,
    unit_account: DEFAULT_PROVIDENT_UNIT_ACCOUNT,
  });
  const [showProvidentSettingsDialog, setShowProvidentSettingsDialog] = useState(false);
  const [providentBillMonth, setProvidentBillMonth] = useState(() => getCurrentMonthLabel());
  const [billGenerating, setBillGenerating] = useState(false);
  const [showBillDetailDialog, setShowBillDetailDialog] = useState(false);
  const [activeBill, setActiveBill] = useState<ProvidentFundBill | null>(null);
  const [selectedBillId, setSelectedBillId] = useState<number | null>(null);
  const [billOverwriteTarget, setBillOverwriteTarget] = useState<ProvidentFundBill | null>(null);
  const [pendingBillMonth, setPendingBillMonth] = useState("");
  const [showBillOverwriteDialog, setShowBillOverwriteDialog] = useState(false);
  const [billDeleteTarget, setBillDeleteTarget] = useState<ProvidentFundBill | null>(null);
  const [showBillPrecheckDialog, setShowBillPrecheckDialog] = useState(false);
  const [billDetailSearch, setBillDetailSearch] = useState("");
  const [providentImportDialogOpen, setProvidentImportDialogOpen] = useState(false);
  const [selectedProvidentFile, setSelectedProvidentFile] = useState<File | null>(null);
  const [providentImportError, setProvidentImportError] = useState("");
  const [sealDialogRecord, setSealDialogRecord] = useState<ProvidentFundRecord | null>(null);
  const [sealDate, setSealDate] = useState(formatDateToInput(new Date()));
  const [sealSubmitting, setSealSubmitting] = useState(false);
  const [unsealDialogRecord, setUnsealDialogRecord] = useState<ProvidentFundRecord | null>(null);
  const [unsealDate, setUnsealDate] = useState(formatDateToInput(new Date()));
  const [unsealSubmitting, setUnsealSubmitting] = useState(false);
  const { token, isAuthenticated, isLoading: authLoading } = useAuth();

  useEffect(() => {
    setActiveRosterTab(initialTab);
  }, [initialTab]);

  useEffect(() => {
    if (!showBillDetailDialog) {
      setBillDetailSearch("");
    }
  }, [showBillDetailDialog]);

  useEffect(() => {
    if (showBillDetailDialog && activeBill) {
      setBillDetailSearch("");
    }
  }, [activeBill, showBillDetailDialog]);

  const selectedBill = useMemo(() => {
    if (providentBills.length === 0) {
      return null;
    }
    if (selectedBillId && providentBills.some((bill) => bill.id === selectedBillId)) {
      return providentBills.find((bill) => bill.id === selectedBillId) ?? providentBills[0];
    }
    return providentBills[0];
  }, [providentBills, selectedBillId]);

  const filteredBillItems = useMemo(() => {
    const items = activeBill?.items ?? [];
    const keyword = billDetailSearch.trim().toLowerCase();
    if (!keyword) {
      return items;
    }
    return items.filter((item) =>
      [item.personal_account, item.name, item.identity_number]
        .map((value) => (value ?? "").toLowerCase())
        .some((value) => value.includes(keyword)),
    );
  }, [activeBill, billDetailSearch]);

  const handleExportBillItems = useCallback(() => {
    if (!activeBill) {
      toast.error("请选择账单后再导出");
      return;
    }
    const rows = filteredBillItems;
    if (rows.length === 0) {
      toast.info("暂无账单明细可导出");
      return;
    }
    const sheetData = [
      ["个人账号", "姓名", "证件号码", "月应缴额（个人）", "月应缴额（单位）", "合计"],
      ...rows.map((item) => [
        item.personal_account,
        item.name,
        item.identity_number,
        Number(item.personal_amount ?? 0),
        Number(item.company_amount ?? 0),
        Number(item.total_amount ?? 0),
      ]),
    ];
    const worksheet = XLSX.utils.aoa_to_sheet(sheetData);
    const workbook = XLSX.utils.book_new();
    XLSX.utils.book_append_sheet(workbook, worksheet, "账单明细");
    const buffer = XLSX.write(workbook, { type: "array", bookType: "xlsx" });
    const filename = `公积金账单-${activeBill.month_label}-${new Date().toISOString().slice(0, 10)}.xlsx`;
    const blob = new Blob([buffer], {
      type: "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
    });
    downloadBlob(blob, filename);
  }, [activeBill, filteredBillItems]);

  const mergeEmployeesWithCallbackMap = useCallback(
    (list: Employee[]) => {
      if (!list.length) {
        return list;
      }
      const hasMap = Object.keys(callbackPersonalMap).length > 0;
      if (!hasMap) {
        return list;
      }
      let changed = false;
      const next = list.map((employee) => {
        const current = employee.socialInsuranceNumber?.trim();
        if (current) {
          return employee;
        }
        const mappedNumber = callbackPersonalMap[normalizeIdKey(employee.idNumber)];
        if (!mappedNumber) {
          return employee;
        }
        changed = true;
        return { ...employee, socialInsuranceNumber: mappedNumber };
      });
      return changed ? next : list;
    },
    [callbackPersonalMap],
  );

  const applyEmployeeData = useCallback((list: EmployeeResponse[]) => {
    const mapped = list.map(mapEmployeeFromApi).map(applyDefaultInsuranceStatus);
    const withCallback = mergeEmployeesWithCallbackMap(mapped);
    const active = withCallback.filter((emp) => emp.status !== "resigned");
    const resigned = withCallback.filter((emp) => emp.status === "resigned");
    setEmployees(active);
    setResignedEmployees(resigned);
    setSelectedEmployeeIds((prev) => {
      const activeSet = new Set(active.map((emp) => emp.id));
      return prev.filter((id) => activeSet.has(id));
    });
  }, [mergeEmployeesWithCallbackMap, setEmployees, setResignedEmployees, setSelectedEmployeeIds]);

  const matchEmployeeByIdNumber = useCallback((idNumber?: string | null) => {
    const normalized = normalizeIdKey(idNumber);
    if (!normalized) {
      return null;
    }
    const combined = [...employees, ...resignedEmployees];
    const match = combined.find((emp) => normalizeIdKey(emp.idNumber) === normalized);
    return match ? { ...match } : null;
  }, [employees, resignedEmployees]);


  const loadEmployees = useCallback(async () => {
    if (!token) {
      return;
    }
    try {
      const data = await fetchEmployees(token);
      applyEmployeeData(data);
    } catch (error) {
      console.error("[EmployeeManagement] failed to load employees", error);
      toast.error(
        error instanceof Error ? error.message : "获取员工列表失败，请稍后重试",
      );
    }
  }, [applyEmployeeData, token]);

  const loadSocialInsuranceData = useCallback(async () => {
    if (!token) {
      return;
    }
    setLoadingInsurance(true);
    try {
      const data = await fetchSocialInsuranceChanges({ token });
      setSocialInsuranceChanges(data.map(mapSocialInsuranceRecordFromApi));
    } catch (error) {
      console.error("[EmployeeManagement] failed to load social insurance changes", error);
      toast.error(error instanceof Error ? error.message : "获取社保增减记录失败，请稍后重试");
    } finally {
      setLoadingInsurance(false);
    }
  }, [token]);

  const loadSocialOptions = useCallback(async () => {
    if (!token) {
      return;
    }
    setSocialOptionsLoading(true);
    try {
      const data = await fetchSocialInsuranceOptions(token);
      setSocialOptions(data);
      setSocialOptionsError("");
    } catch (error) {
      console.error("[EmployeeManagement] failed to load social template options", error);
      const message = error instanceof Error ? error.message : "社保模板选项加载失败，请稍后重试";
      setSocialOptionsError(message);
      toast.error(message);
    } finally {
      setSocialOptionsLoading(false);
    }
  }, [token]);

  const loadProvidentRecords = useCallback(async () => {
    if (!token) {
      return;
    }
    setProvidentLoading(true);
    try {
      const data = await fetchProvidentRecords();
      setProvidentRecords(data);
    } catch (error) {
      console.error("[EmployeeManagement] failed to load provident records", error);
      toast.error(error instanceof Error ? error.message : "获取公积金列表失败，请稍后重试");
    } finally {
      setProvidentLoading(false);
    }
  }, [token]);

  const loadProvidentSettings = useCallback(async () => {
    if (!token) {
      return;
    }
    try {
      const settings = await getProvidentSettings();
      setProvidentSettings(settings);
      setProvidentSettingsDraft({
        unit_name: settings.unit_name || DEFAULT_PROVIDENT_UNIT_NAME,
        unit_account: settings.unit_account || DEFAULT_PROVIDENT_UNIT_ACCOUNT,
      });
    } catch (error) {
      console.error("[EmployeeManagement] failed to load provident settings", error);
      toast.error(error instanceof Error ? error.message : "加载公积金单位设置失败");
    }
  }, [token]);

  const loadProvidentBills = useCallback(async () => {
    if (!token) {
      return;
    }
    setProvidentBillsLoading(true);
    try {
      const bills = await fetchProvidentBills();
      setProvidentBills(bills);
    } catch (error) {
      console.error("[EmployeeManagement] failed to load provident bills", error);
      toast.error(error instanceof Error ? error.message : "加载公积金账单失败");
    } finally {
      setProvidentBillsLoading(false);
    }
  }, [token]);

  useEffect(() => {
    if (!authLoading && isAuthenticated) {
      loadEmployees();
      loadSocialInsuranceData();
      loadSocialOptions();
      loadProvidentRecords();
      loadProvidentSettings();
      loadProvidentBills();
    }
  }, [
    authLoading,
    isAuthenticated,
    loadEmployees,
    loadSocialInsuranceData,
    loadSocialOptions,
    loadProvidentRecords,
    loadProvidentSettings,
    loadProvidentBills,
  ]);

  useEffect(() => {
    setSelectedResignedIds((prev) => prev.filter((id) => resignedEmployees.some((emp) => emp.id === id)));
  }, [resignedEmployees]);

  useEffect(() => {
    setSelectedProvidentIds((prev) => prev.filter((id) => providentRecords.some((record) => record.id === id)));
  }, [providentRecords]);

  useEffect(() => {
    setSelectedBillId((prev) => {
      if (providentBills.length === 0) {
        return prev === null ? prev : null;
      }
      if (prev && providentBills.some((bill) => bill.id === prev)) {
        return prev;
      }
      return providentBills[0].id;
    });
  }, [providentBills]);

  useEffect(() => {
    if (typeof window === "undefined") {
      return;
    }
    try {
      const stored = window.localStorage.getItem(UNIT_INFO_STORAGE_KEY);
      if (stored) {
        const parsed = JSON.parse(stored) as Partial<UnitInfo>;
        const next: UnitInfo = {
          socialCode: parsed?.socialCode?.trim() || DEFAULT_UNIT_INFO.socialCode,
          unitName: parsed?.unitName?.trim() || DEFAULT_UNIT_INFO.unitName,
        };
        setUnitInfo(next);
      }
    } catch (error) {
      console.warn("[UnitInfo] failed to read settings", error);
    }
  }, []);

  useEffect(() => {
    setSelectedInsuranceChangeIds((prev) => prev.filter((id) => socialInsuranceChanges.some((change) => change.id === id)));
  }, [socialInsuranceChanges]);

  useEffect(() => {
    if (insuranceView === "provident") {
      if (selectedInsuranceChangeIds.length > 0) {
        setSelectedInsuranceChangeIds([]);
      }
      return;
    }
    setSelectedInsuranceChangeIds((prev) => {
      const allowedIds = new Set(
        socialInsuranceChanges
          .filter((change) => change.changeType === insuranceView)
          .map((change) => change.id),
      );
      const next = prev.filter((id) => allowedIds.has(id));
      return next.length === prev.length ? prev : next;
    });
  }, [insuranceView, selectedInsuranceChangeIds.length, socialInsuranceChanges]);

  // 对话框状态
  const [showAddEmployee, setShowAddEmployee] = useState(false);
  const [showBatchImport, setShowBatchImport] = useState(false);
  const [showResignedImport, setShowResignedImport] = useState(false);
  const [showResignDialog, setShowResignDialog] = useState(false);
  const [showInsuranceUploadDialog, setShowInsuranceUploadDialog] = useState(false);
  const [showEditEmployee, setShowEditEmployee] = useState(false);
  const [showBatchDeleteConfirm, setShowBatchDeleteConfirm] = useState(false);
  const [deletingEmployees, setDeletingEmployees] = useState(false);
  const [showBatchRestoreConfirm, setShowBatchRestoreConfirm] = useState(false);
  const [showBatchInsuranceConfirm, setShowBatchInsuranceConfirm] = useState(false);
  const [resignSubmitting, setResignSubmitting] = useState(false);
  const [restoringResigned, setRestoringResigned] = useState(false);
  const [showResignedDetail, setShowResignedDetail] = useState(false);
  const [resignedDetailEmployee, setResignedDetailEmployee] = useState<Employee | null>(null);
  const [unitInfo, setUnitInfo] = useState<UnitInfo>(() => ({ ...DEFAULT_UNIT_INFO }));
  const [unitInfoDialogOpen, setUnitInfoDialogOpen] = useState(false);
  const [unitInfoDraft, setUnitInfoDraft] = useState<UnitInfo>(() => ({ ...DEFAULT_UNIT_INFO }));
  const [resignProofPreviewUrl, setResignProofPreviewUrl] = useState<string>("");
  const [resignProofPreviewType, setResignProofPreviewType] = useState<"image" | "pdf" | "other" | "">("");
  const [resignProofPreviewError, setResignProofPreviewError] = useState<string>("");
  const [resignProofPreviewLoading, setResignProofPreviewLoading] = useState(false);
const [resignProofPreviewFilename, setResignProofPreviewFilename] = useState<string>("");
const [resignProofPreviewSize, setResignProofPreviewSize] = useState<number | null>(null);
const [resignProofPreviewContentType, setResignProofPreviewContentType] = useState<string>("");
const [resignReasons, setResignReasons] = useState<string[]>([]);
  const resignedImportMode: EmployeeImportMode = "merge";
const [resignedImporting, setResignedImporting] = useState(false);
const [downloadingResignedTemplate, setDownloadingResignedTemplate] = useState(false);
const [pendingResignedImport, setPendingResignedImport] = useState<{ file: File; mode: EmployeeImportMode } | null>(null);
const [resignedConflicts, setResignedConflicts] = useState<EmployeeImportConflict[]>([]);
const [showResignedConflictDialog, setShowResignedConflictDialog] = useState(false);
const [resignedConflictImporting, setResignedConflictImporting] = useState(false);
const [insuranceUploadType, setInsuranceUploadType] = useState<SocialInsuranceChangeType>("increase");
const [insuranceUploadFile, setInsuranceUploadFile] = useState<File | null>(null);
const [insuranceUploadError, setInsuranceUploadError] = useState("");
const [insuranceUploadPreview, setInsuranceUploadPreview] = useState<SocialInsuranceImportRecordPayload[]>([]);
  const [insuranceImporting, setInsuranceImporting] = useState(false);
  const [showPrintDialog, setShowPrintDialog] = useState(false);
  const [printContext, setPrintContext] = useState<
    | { type: "active"; rows: Employee[] }
    | { type: "resigned"; rows: Employee[] }
    | { type: "insurance"; rows: SocialInsuranceRecord[] }
    | { type: "provident"; rows: ProvidentFundRecord[] }
    | null
  >(null);
  const [printTitle, setPrintTitle] = useState(() => {
    const saved = readLocalStorageJSON<PrintSettings>(PRINT_SETTINGS_STORAGE_KEY);
    return saved?.title ?? "";
  });
  const [printSuggestedTitle, setPrintSuggestedTitle] = useState("");
  const [printWatermark, setPrintWatermark] = useState(() => {
    const saved = readLocalStorageJSON<PrintSettings>(PRINT_SETTINGS_STORAGE_KEY);
    return saved?.watermark ?? "内部资料 请勿外传";
  });
  const [printOrientation, setPrintOrientation] = useState<PrintOrientation>(() => {
    const saved = readLocalStorageJSON<PrintSettings>(PRINT_SETTINGS_STORAGE_KEY);
    if (saved?.orientation === "portrait" || saved?.orientation === "landscape") {
      return saved.orientation;
    }
    return "auto";
  });
  const [userPreferenceReady, setUserPreferenceReady] = useState(false);
  const [employeeSort, setEmployeeSort] = useState<TableSortState<EmployeeColumnId>>(() => {
    return readLocalStorageJSON<TableSortState<EmployeeColumnId>>(EMPLOYEE_SORT_STORAGE_KEY) ?? {
      key: null,
      direction: "asc",
    };
  });
  const [resignedSort, setResignedSort] = useState<TableSortState<EmployeeColumnId>>(() => {
    return readLocalStorageJSON<TableSortState<EmployeeColumnId>>(RESIGNED_SORT_STORAGE_KEY) ?? {
      key: null,
      direction: "asc",
    };
  });
  const [insuranceSort, setInsuranceSort] = useState<TableSortState<InsuranceColumnId>>(() => {
    return readLocalStorageJSON<TableSortState<InsuranceColumnId>>(INSURANCE_SORT_STORAGE_KEY) ?? {
      key: null,
      direction: "asc",
    };
  });
  const [resignedColumnOrder, setResignedColumnOrder] = useState<string[]>(() => {
    return readLocalStorageJSON<string[]>(RESIGNED_COLUMN_ORDER_STORAGE_KEY) ?? [];
  });
  const [insuranceColumnOrder, setInsuranceColumnOrder] = useState<InsuranceColumnId[]>(() => {
    const saved = readLocalStorageJSON<InsuranceColumnId[]>(INSURANCE_COLUMN_ORDER_STORAGE_KEY) ?? [];
    const allowed = new Set(DEFAULT_INSURANCE_COLUMN_ORDER);
    const filtered = saved.filter((id) => allowed.has(id));
    const appended = DEFAULT_INSURANCE_COLUMN_ORDER.filter((id) => !filtered.includes(id));
    if (filtered.length === 0 && appended.length === DEFAULT_INSURANCE_COLUMN_ORDER.length) {
      return [...DEFAULT_INSURANCE_COLUMN_ORDER];
    }
    return [...filtered, ...appended];
  });
  const [insuranceVisibleColumns, setInsuranceVisibleColumns] = useState<Record<InsuranceViewKey, InsuranceColumnId[]>>(() => {
    const saved = readLocalStorageJSON<Record<InsuranceViewKey, InsuranceColumnId[]>>(INSURANCE_VISIBLE_COLUMNS_STORAGE_KEY);
    return sanitizeInsuranceVisibleMap(saved ?? undefined);
  });
  const [insuranceFieldDialogFor, setInsuranceFieldDialogFor] = useState<InsuranceViewKey | null>(null);

  // 当前操作的员工
  const [selectedEmployee, setSelectedEmployee] = useState<Employee | null>(null);

  // 编辑员工表单数据
  const [editEmployee, setEditEmployee] = useState<Partial<Employee>>({});
  const updateEditEmployeeField = useCallback((key: keyof Employee, value: string) => {
    setEditEmployee((prev) => ({ ...prev, [key]: value }));
  }, []);

  // 表单数据（包含默认值）
const [newEmployee, setNewEmployee] = useState<Partial<Employee>>(() => ({
    ...createEmptyEmployee(),
    education: "初中",
    politicalStatus: "群众",
    workClothingSize: "L",
    safetyShoeSize: "40",
    insuranceStatus: "",
    providentFundNumber: "无",
  }));
  const [resignDate, setResignDate] = useState(() => formatDateToInput(new Date()));
  const [resignProofFile, setResignProofFile] = useState<File | null>(null);
  const [resignProofError, setResignProofError] = useState("");

  // 搜索和筛选
  const [searchTerm, setSearchTerm] = useState("");
  const [departmentFilter, setDepartmentFilter] = useState("all");
  const [resignedSearchTerm, setResignedSearchTerm] = useState("");
  const [resignedDepartmentFilter, setResignedDepartmentFilter] = useState("all");
  const [insuranceDepartmentFilter, setInsuranceDepartmentFilter] = useState("all");
  const [insuranceReasonFilter, setInsuranceReasonFilter] = useState("all");

  // 字段显示控制
  const [visibleFields, setVisibleFields] = useState<(keyof Employee)[]>(() => {
    if (typeof window !== "undefined") {
      const saved = window.localStorage.getItem(EMPLOYEE_VISIBLE_FIELDS_STORAGE_KEY);
      if (saved) {
        try {
          const parsed = JSON.parse(saved) as (keyof Employee)[];
          const sanitized = sanitizeVisibleFieldKeys(parsed);
          if (!arraysShallowEqual(parsed, sanitized)) {
            writeLocalStorageJSON(EMPLOYEE_VISIBLE_FIELDS_STORAGE_KEY, sanitized);
          }
          return sanitized;
        } catch (error) {
          console.warn("[employee-visible-fields] 解析失败, 使用默认值", error);
        }
      }

      for (const legacyKey of LEGACY_VISIBLE_FIELD_STORAGE_KEYS) {
        const legacySaved = window.localStorage.getItem(legacyKey);
        if (legacySaved) {
          try {
            const parsed = JSON.parse(legacySaved) as (keyof Employee)[];
            const sanitized = sanitizeVisibleFieldKeys(parsed);
            writeLocalStorageJSON(EMPLOYEE_VISIBLE_FIELDS_STORAGE_KEY, sanitized);
            window.localStorage.removeItem(legacyKey);
            return sanitized;
          } catch (error) {
            console.warn(`[employee-visible-fields] 解析旧键 ${legacyKey} 失败`, error);
            window.localStorage.removeItem(legacyKey);
          }
        }
      }
    }
    return [...DEFAULT_VISIBLE_FIELDS];
  });
  const [showFieldSelector, setShowFieldSelector] = useState(false);

  // 文件上传引用
  const fileInputRef = useRef<HTMLInputElement>(null);
  const resignedFileInputRef = useRef<HTMLInputElement>(null);
  const selectAllRef = useRef<HTMLInputElement>(null);
  const insuranceUploadInputRef = useRef<HTMLInputElement>(null);
  const providentSelectAllRef = useRef<HTMLInputElement>(null);
  const providentImportInputRef = useRef<HTMLInputElement>(null);

  const hasActiveSelection = selectedEmployeeIds.length > 0;

  useEffect(() => {
    if (showResignDialog) {
      setResignDate(formatDateToInput(new Date()));
      setResignProofFile(null);
      setResignProofError("");
      setResignSubmitting(false);
      setResignReasons(selectedEmployee?.resignReasons ?? []);
    } else {
      setResignProofFile(null);
      setResignProofError("");
      setResignSubmitting(false);
      setResignReasons([]);
    }
  }, [selectedEmployee, showResignDialog]);

  useEffect(() => {
    writeLocalStorageJSON(EMPLOYEE_SORT_STORAGE_KEY, employeeSort);
  }, [employeeSort]);

  useEffect(() => {
    writeLocalStorageJSON(RESIGNED_SORT_STORAGE_KEY, resignedSort);
  }, [resignedSort]);

  useEffect(() => {
    writeLocalStorageJSON(INSURANCE_SORT_STORAGE_KEY, insuranceSort);
  }, [insuranceSort]);

  useEffect(() => {
    writeLocalStorageJSON(PROVIDENT_SORT_STORAGE_KEY, providentSort);
  }, [providentSort]);

  useEffect(() => {
    writeLocalStorageJSON(PROVIDENT_VISIBLE_COLUMNS_STORAGE_KEY, providentVisibleColumns);
  }, [providentVisibleColumns]);

  useEffect(() => {
    if (providentColumnOrder.length > 0) {
      writeLocalStorageJSON(PROVIDENT_COLUMN_ORDER_STORAGE_KEY, providentColumnOrder);
    }
  }, [providentColumnOrder]);

  useEffect(() => {
    const payload: PrintSettings = {
      title: printTitle.trim(),
      watermark: printWatermark.trim(),
      orientation: printOrientation,
    };
    writeLocalStorageJSON(PRINT_SETTINGS_STORAGE_KEY, payload);
  }, [printTitle, printWatermark, printOrientation]);

  useEffect(() => {
    let cancelled = false;
    const hydratePreferences = async () => {
      try {
        const prefs = await fetchUserPreferences();
        const activePref = parseListPreference<keyof Employee>(
          prefs?.[PREF_KEY_EMPLOYEE_COLUMNS] ?? prefs?.[PREF_KEY_EMPLOYEE_VISIBLE_FIELDS],
          AVAILABLE_FIELDS.map((field) => field.key),
          [...DEFAULT_VISIBLE_FIELDS],
        );
        const nextVisibleFields =
          activePref?.visible && activePref.visible.length
            ? sanitizeVisibleFieldKeys(activePref.visible as (keyof Employee)[])
            : activePref?.order && activePref.order.length
              ? sanitizeVisibleFieldKeys(activePref.order as (keyof Employee)[])
              : null;
        if (nextVisibleFields && nextVisibleFields.length > 0) {
          setVisibleFields(nextVisibleFields);
        }

        const resignedPref = parseListPreference<string>(prefs?.[PREF_KEY_RESIGNED_COLUMNS], DEFAULT_RESIGNED_ORDER_BASE);
        if (resignedPref?.order?.length) {
          setResignedColumnOrder(resignedPref.order);
        }

        const insurancePref = parseListPreference<InsuranceColumnId>(
          prefs?.[PREF_KEY_INSURANCE_COLUMNS],
          DEFAULT_INSURANCE_COLUMN_ORDER,
          DEFAULT_INSURANCE_COLUMN_ORDER,
        );
        if (insurancePref?.order?.length) {
          setInsuranceColumnOrder(insurancePref.order);
        }
        const insuranceVisiblePref = parseInsuranceVisiblePreference(prefs?.[PREF_KEY_INSURANCE_COLUMNS]);
        if (insuranceVisiblePref) {
          setInsuranceVisibleColumns(insuranceVisiblePref);
        }

        const providentPref = parseListPreference<ProvidentColumnId>(
          prefs?.[PREF_KEY_PROVIDENT_COLUMNS],
          DEFAULT_PROVIDENT_COLUMN_ORDER,
          DEFAULT_PROVIDENT_VISIBLE_COLUMNS,
        );
        if (providentPref?.visible?.length) {
          setProvidentVisibleColumns(providentPref.visible);
        }
        if (providentPref?.order?.length) {
          setProvidentColumnOrder(providentPref.order);
        }

        const employeeSortPref = prefs?.[PREF_KEY_EMPLOYEE_SORT];
        if (employeeSortPref) {
          const allowed = AVAILABLE_FIELDS.map((field) => field.key as EmployeeColumnId);
          setEmployeeSort((prev) => sanitizeSortPreference(employeeSortPref, allowed, prev));
        }

        const resignedSortPref = prefs?.[PREF_KEY_RESIGNED_SORT];
        if (resignedSortPref) {
          setResignedSort((prev) => sanitizeSortPreference(resignedSortPref, DEFAULT_RESIGNED_ORDER_BASE as EmployeeColumnId[], prev));
        }

        const insuranceSortPref = prefs?.[PREF_KEY_INSURANCE_SORT];
        if (insuranceSortPref) {
          setInsuranceSort((prev) => sanitizeSortPreference(insuranceSortPref, DEFAULT_INSURANCE_COLUMN_ORDER, prev));
        }

        const providentSortPref = prefs?.[PREF_KEY_PROVIDENT_SORT];
        if (providentSortPref) {
          setProvidentSort((prev) => sanitizeSortPreference(providentSortPref, DEFAULT_PROVIDENT_COLUMN_ORDER, prev));
        }

        const employeeFiltersPref = prefs?.[PREF_KEY_EMPLOYEE_FILTERS];
        if (employeeFiltersPref && typeof employeeFiltersPref === "object") {
          const { searchTerm: prefSearch, department } = employeeFiltersPref as Record<string, unknown>;
          if (typeof prefSearch === "string") {
            setSearchTerm(prefSearch);
          }
          if (typeof department === "string" && department.trim()) {
            setDepartmentFilter(department);
          }
        }

        const insuranceFiltersPref = prefs?.[PREF_KEY_INSURANCE_FILTERS];
        if (insuranceFiltersPref && typeof insuranceFiltersPref === "object") {
          const { search, view, department, reason } = insuranceFiltersPref as Record<string, unknown>;
          if (typeof search === "string") {
            setInsuranceSearch(search);
          }
          if (view === "increase" || view === "decrease" || view === "provident") {
            setInsuranceView(view);
          }
          if (typeof department === "string" && department.trim()) {
            setInsuranceDepartmentFilter(department);
          }
          if (typeof reason === "string" && reason.trim()) {
            setInsuranceReasonFilter(reason);
          }
        }

        const providentFiltersPref = prefs?.[PREF_KEY_PROVIDENT_FILTERS];
        if (providentFiltersPref && typeof providentFiltersPref === "object") {
          const { search, status } = providentFiltersPref as Record<string, unknown>;
          if (typeof search === "string") {
            setProvidentSearch(search);
          }
          if (status === "all" || status === "active" || status === "sealed") {
            setProvidentStatusFilter(status);
          }
        }

        const resignedFiltersPref = prefs?.[PREF_KEY_RESIGNED_FILTERS];
        if (resignedFiltersPref && typeof resignedFiltersPref === "object") {
          const { search, department } = resignedFiltersPref as Record<string, unknown>;
          if (typeof search === "string") {
            setResignedSearchTerm(search);
          }
          if (typeof department === "string" && department.trim()) {
            setResignedDepartmentFilter(department);
          }
        }
      } catch (error) {
        console.error("[EmployeeManagement] 加载用户偏好失败", error);
      } finally {
        if (!cancelled) {
          setUserPreferenceReady(true);
        }
      }
    };
    hydratePreferences();
    return () => {
      cancelled = true;
    };
  }, []);

  useEffect(() => {
    if (!userPreferenceReady) return;
    updateUserPreferences({
      [PREF_KEY_EMPLOYEE_COLUMNS]: { order: visibleFields, visible: visibleFields },
      [PREF_KEY_EMPLOYEE_VISIBLE_FIELDS]: { visible: visibleFields },
      [PREF_KEY_RESIGNED_COLUMNS]: { order: resignedColumnOrder },
      [PREF_KEY_INSURANCE_COLUMNS]: { order: insuranceColumnOrder, visible: insuranceVisibleColumns },
      [PREF_KEY_PROVIDENT_COLUMNS]: { order: providentColumnOrder, visible: providentVisibleColumns },
      [PREF_KEY_EMPLOYEE_SORT]: { key: employeeSort.key, direction: employeeSort.direction },
      [PREF_KEY_RESIGNED_SORT]: { key: resignedSort.key, direction: resignedSort.direction },
      [PREF_KEY_INSURANCE_SORT]: { key: insuranceSort.key, direction: insuranceSort.direction },
      [PREF_KEY_PROVIDENT_SORT]: { key: providentSort.key, direction: providentSort.direction },
      [PREF_KEY_EMPLOYEE_FILTERS]: { searchTerm, department: departmentFilter },
      [PREF_KEY_INSURANCE_FILTERS]: { search: insuranceSearch, view: insuranceView, department: insuranceDepartmentFilter, reason: insuranceReasonFilter },
      [PREF_KEY_PROVIDENT_FILTERS]: { search: providentSearch, status: providentStatusFilter },
      [PREF_KEY_RESIGNED_FILTERS]: { search: resignedSearchTerm, department: resignedDepartmentFilter },
    }).catch((error) => {
      console.error("[EmployeeManagement] 保存用户偏好失败", error);
    });
  }, [
    departmentFilter,
    employeeSort,
    insuranceColumnOrder,
    insuranceSearch,
    insuranceSort,
    insuranceView,
    insuranceReasonFilter,
    insuranceVisibleColumns,
    insuranceDepartmentFilter,
    providentColumnOrder,
    providentVisibleColumns,
    providentSearch,
    providentSort,
    providentStatusFilter,
    resignedDepartmentFilter,
    resignedColumnOrder,
    resignedSort,
    resignedSearchTerm,
    searchTerm,
    userPreferenceReady,
    visibleFields,
  ]);

  useEffect(() => {
    const baseOrder = visibleFields.map((field) => field as string);
    const extra = Array.from(RESIGNED_EXTRA_COLUMNS);
    setResignedColumnOrder((prev) => {
      const allowed = new Set([...baseOrder, ...extra]);
      const filtered = prev.filter((id) => allowed.has(id));
      const appendedBase = baseOrder.filter((id) => !filtered.includes(id));
      const appendedExtra = extra.filter((id) => !filtered.includes(id));
      const next = [...filtered, ...appendedBase, ...appendedExtra];
      if (arraysShallowEqual(next, prev)) {
        return prev;
      }
      return next;
    });
  }, [visibleFields]);

  useEffect(() => {
    if (resignedColumnOrder.length > 0) {
      writeLocalStorageJSON(RESIGNED_COLUMN_ORDER_STORAGE_KEY, resignedColumnOrder);
    }
  }, [resignedColumnOrder]);

  useEffect(() => {
    setInsuranceColumnOrder((prev) => {
      const allowed = new Set(DEFAULT_INSURANCE_COLUMN_ORDER);
      const filtered = prev.filter((id) => allowed.has(id));
      const appended = DEFAULT_INSURANCE_COLUMN_ORDER.filter((id) => !filtered.includes(id));
      const next = [...filtered, ...appended];
      if (arraysShallowEqual(next, prev)) {
        return prev;
      }
      return next;
    });
  }, []);

  useEffect(() => {
    if (insuranceColumnOrder.length > 0) {
      writeLocalStorageJSON(INSURANCE_COLUMN_ORDER_STORAGE_KEY, insuranceColumnOrder);
    }
  }, [insuranceColumnOrder]);
  useEffect(() => {
    writeLocalStorageJSON(INSURANCE_VISIBLE_COLUMNS_STORAGE_KEY, insuranceVisibleColumns);
  }, [insuranceVisibleColumns]);

  useEffect(() => {
    setProvidentColumnOrder((prev) => {
      const allowed = new Set(DEFAULT_PROVIDENT_COLUMN_ORDER);
      const filtered = prev.filter((id) => allowed.has(id));
      const appended = DEFAULT_PROVIDENT_COLUMN_ORDER.filter((id) => !filtered.includes(id));
      const next = [...filtered, ...appended];
      if (arraysShallowEqual(next, prev)) {
        return prev;
      }
      return next;
    });
  }, []);

  useEffect(() => {
    setProvidentVisibleColumns((prev) => {
      const allowed = new Set(DEFAULT_PROVIDENT_COLUMN_ORDER);
      const filtered = prev.filter((id) => allowed.has(id)) as ProvidentColumnId[];
      if (filtered.length === 0) {
        return [...DEFAULT_PROVIDENT_VISIBLE_COLUMNS];
      }
      if (arraysShallowEqual(filtered, prev)) {
        return prev;
      }
      return filtered;
    });
  }, []);

  useEffect(() => {
    if (typeof window === "undefined") {
      return;
    }
    try {
      const stored = window.localStorage.getItem(CALLBACK_PERSONAL_MAP_STORAGE_KEY);
      if (stored) {
        const parsed = JSON.parse(stored);
        if (parsed && typeof parsed === "object") {
          setCallbackPersonalMap(parsed as Record<string, string>);
        }
      }
    } catch (error) {
      console.warn("[EmployeeManagement] 读取回盘映射失败", error);
    }
  }, []);

  useEffect(() => {
    if (typeof window === "undefined") {
      return;
    }
    const handleStorage = (event: StorageEvent) => {
      if (event.key !== CALLBACK_PERSONAL_MAP_STORAGE_KEY) {
        return;
      }
      try {
        if (event.newValue) {
          const parsed = JSON.parse(event.newValue);
          setCallbackPersonalMap(parsed as Record<string, string>);
        }
      } catch (error) {
        console.warn("[EmployeeManagement] 同步回盘映射失败", error);
      }
    };
    window.addEventListener("storage", handleStorage);
    return () => window.removeEventListener("storage", handleStorage);
  }, []);

  useEffect(() => {
    if (Object.keys(callbackPersonalMap).length === 0) {
      return;
    }
    setEmployees((prev) => mergeEmployeesWithCallbackMap(prev));
    setResignedEmployees((prev) => mergeEmployeesWithCallbackMap(prev));
  }, [callbackPersonalMap, mergeEmployeesWithCallbackMap]);

  useEffect(() => {
    if (employeeSort.key && !visibleFields.includes(employeeSort.key as keyof Employee)) {
      setEmployeeSort({ key: null, direction: "asc" });
    }
  }, [employeeSort, visibleFields]);

  useEffect(() => {
    if (resignedSort.key && !resignedColumnOrder.includes(resignedSort.key as string)) {
      setResignedSort({ key: null, direction: "asc" });
    }
  }, [resignedSort, resignedColumnOrder]);

  useEffect(() => {
    if (insuranceSort.key && !insuranceColumnOrder.includes(insuranceSort.key as InsuranceColumnId)) {
      setInsuranceSort({ key: null, direction: "asc" });
    }
  }, [insuranceSort, insuranceColumnOrder]);

  const resetResignProofPreview = useCallback(() => {
    setResignProofPreviewUrl((prev) => {
      if (prev) {
        URL.revokeObjectURL(prev);
      }
      return "";
    });
    setResignProofPreviewType("");
    setResignProofPreviewFilename("");
    setResignProofPreviewError("");
    setResignProofPreviewLoading(false);
    setResignProofPreviewSize(null);
    setResignProofPreviewContentType("");
  }, []);

  useEffect(() => {
    let aborted = false;

    const clearPreviewState = () => {
      resetResignProofPreview();
      setResignProofPreviewSize(null);
      setResignProofPreviewContentType("");
    };

    clearPreviewState();

    if (!showResignedDetail || !resignedDetailEmployee) {
      return () => {
        aborted = true;
      };
    }

    if (!resignedDetailEmployee.resignProofUrl) {
      setResignProofPreviewError("该员工尚未上传离职证明");
      return () => {
        aborted = true;
      };
    }

    if (!token) {
      setResignProofPreviewError("登录状态失效，无法加载离职证明");
      return () => {
        aborted = true;
      };
    }

    const employeeId = Number(resignedDetailEmployee.id);
    if (!Number.isFinite(employeeId) || employeeId <= 0) {
      setResignProofPreviewError("当前离职记录缺少有效ID，无法加载离职证明");
      return () => {
        aborted = true;
      };
    }

    const loadPreview = async () => {
      setResignProofPreviewLoading(true);
      try {
        const { blob, filename, contentType, size } = await downloadResignProof(employeeId, token ?? undefined);
        if (aborted) {
          return;
        }
        const objectUrl = URL.createObjectURL(blob);
        if (aborted) {
          URL.revokeObjectURL(objectUrl);
          return;
        }
        const normalizedFilename = filename || resignedDetailEmployee.resignProofName || "离职证明";
        const normalizedContentType = contentType.toLowerCase();
        let type: "image" | "pdf" | "other" = "other";
        if (normalizedContentType.startsWith("image/")) {
          type = "image";
        } else if (normalizedContentType === "application/pdf" || normalizedFilename.toLowerCase().endsWith(".pdf")) {
          type = "pdf";
        }

        setResignProofPreviewUrl(objectUrl);
        setResignProofPreviewFilename(normalizedFilename);
        setResignProofPreviewType(type);
        setResignProofPreviewSize(Number.isFinite(size) && size >= 0 ? size : blob.size);
        setResignProofPreviewContentType(contentType);
        setResignProofPreviewError("");
      } catch (error) {
        console.error("[EmployeeManagement] preview resign proof failed", error);
        if (!aborted) {
          const message = error instanceof Error ? error.message : "预览离职证明失败，请稍后重试";
          if (/不存在|未找到/.test(message)) {
            setResignProofPreviewError("离职证明文件不存在或已被删除，请重新上传");
          } else {
            setResignProofPreviewError(message);
          }
        }
      } finally {
        if (!aborted) {
          setResignProofPreviewLoading(false);
        }
      }
    };

    loadPreview();

    return () => {
      aborted = true;
      clearPreviewState();
    };
  }, [resetResignProofPreview, resignedDetailEmployee, showResignedDetail, token]);

  // 获取所有部门列表
  const getDepartments = () => {
    const depts = new Set<string>();
    Object.keys(DEPARTMENT_CODES).forEach((dept) => depts.add(dept));
    employees.forEach(emp => {
      const dept = emp.department?.trim();
      if (dept) depts.add(dept);
    });
    resignedEmployees.forEach(emp => {
      const dept = emp.department?.trim();
      if (dept) depts.add(dept);
    });
    return Array.from(depts).sort((a, b) => a.localeCompare(b, "zh-CN"));
  };

  const positionOptions = useMemo(() => {
    const set = new Set<string>();
    [...employees, ...resignedEmployees].forEach((emp) => {
      const value = emp.position?.trim();
      if (value) set.add(value);
    });
    return Array.from(set).sort((a, b) => a.localeCompare(b, "zh-CN"));
  }, [employees, resignedEmployees]);

  const insuranceStatusLookup = useMemo(() => {
    const increase = new Set<string>();
    const decrease = new Set<string>();
    socialInsuranceChanges.forEach((change) => {
      const key = change.identityNumber?.trim().toUpperCase();
      if (!key) {
        return;
      }
      if (change.changeType === "increase") {
        increase.add(key);
      } else if (change.changeType === "decrease") {
        decrease.add(key);
      }
    });
    return { increase, decrease };
  }, [socialInsuranceChanges]);

  const deriveInsuranceStatus = useCallback(
    (idNumber: string | undefined, context: "active" | "resigned") => {
      const key = idNumber?.trim().toUpperCase();
      if (!key) {
        return context === "active" ? "需办理参保" : "需办理退保";
      }
      const hasIncrease = insuranceStatusLookup.increase.has(key);
      const hasDecrease = insuranceStatusLookup.decrease.has(key);
      if (context === "active") {
        if (hasIncrease) {
          return "已参保";
        }
        return "需办理参保";
      }
      if (hasDecrease) {
        return "已退保";
      }
      return "需办理退保";
    },
    [insuranceStatusLookup],
  );

  const getInsuranceStatusLabel = useCallback(
    (employee: Pick<Employee, "insuranceStatus" | "idNumber">, context: "active" | "resigned") => {
      const derived = deriveInsuranceStatus(employee.idNumber, context);
      if (derived) {
        return derived;
      }
      const manual = employee.insuranceStatus?.trim();
      if (manual) {
        return manual;
      }
      return context === "active" ? "未参保" : "未退保";
    },
    [deriveInsuranceStatus],
  );

  const derivedActiveStatusForNew = useMemo(
    () => deriveInsuranceStatus(newEmployee.idNumber, "active"),
    [newEmployee.idNumber, deriveInsuranceStatus],
  );

  const editDepartmentOptions = useMemo(() => {
    const depts = new Set<string>();
    Object.keys(DEPARTMENT_CODES).forEach((dept) => depts.add(dept));
    employees.forEach((emp) => {
      const dept = emp.department?.trim();
      if (dept) {
        depts.add(dept);
      }
    });
    resignedEmployees.forEach((emp) => {
      const dept = emp.department?.trim();
      if (dept) {
        depts.add(dept);
      }
    });
    if (editEmployee.department?.trim()) {
      depts.add(editEmployee.department.trim() as string);
    }
    return Array.from(depts).sort((a, b) => a.localeCompare(b, "zh-CN"));
  }, [editEmployee.department, employees, resignedEmployees]);

  const editPositionOptions = useMemo(() => {
    const base = [...positionOptions];
    if (editEmployee.position && !base.includes(editEmployee.position)) {
      return [editEmployee.position, ...base];
    }
    return base;
  }, [positionOptions, editEmployee.position]);

  const editInsuranceContext: "active" | "resigned" =
    editEmployee.status === "resigned" ? "resigned" : "active";
  const derivedInsuranceStatusForEdit = useMemo(
    () => deriveInsuranceStatus(editEmployee.idNumber, editInsuranceContext),
    [deriveInsuranceStatus, editEmployee.idNumber, editInsuranceContext],
  );
  const editInsuranceOptions = editInsuranceContext === "resigned" ? RESIGNED_INSURANCE_STATUS_OPTIONS : ACTIVE_INSURANCE_STATUS_OPTIONS;
  const insuranceStatusHint =
    editInsuranceContext === "active"
      ? "默认将依据社保增加记录自动判断，可手动调整。"
      : "默认将依据社保减少记录自动判断，可手动调整。";

  const handleRosterTabChange = useCallback(
    (next: RosterTab) => {
      setActiveRosterTab(next);
      if (next === "insurance-increase") {
        setInsuranceView("increase");
      } else if (next === "insurance-decrease") {
        setInsuranceView("decrease");
      }
    },
    [setInsuranceView],
  );

  const nativePlaceOptions = useMemo(() => {
    const set = new Set<string>();
    [...employees, ...resignedEmployees].forEach((emp) => {
      const value = emp.nativePlace?.trim();
      if (value) set.add(value);
    });
    return Array.from(set).sort((a, b) => a.localeCompare(b, "zh-CN"));
  }, [employees, resignedEmployees]);

  const educationOptions = useMemo(() => {
    const defaults = ["小学", "初中", "高中", "中专", "大专", "本科", "硕士", "博士"];
    const set = new Set<string>(defaults);
    [...employees, ...resignedEmployees].forEach((emp) => {
      const value = emp.education?.trim();
      if (value) set.add(value);
    });
    return Array.from(set).sort((a, b) => a.localeCompare(b, "zh-CN"));
  }, [employees, resignedEmployees]);

  const graduateSchoolOptions = useMemo(() => {
    const set = new Set<string>();
    [...employees, ...resignedEmployees].forEach((emp) => {
      const value = emp.graduateSchool?.trim();
      if (value) set.add(value);
    });
    return Array.from(set).sort((a, b) => a.localeCompare(b, "zh-CN"));
  }, [employees, resignedEmployees]);

  const majorOptions = useMemo(() => {
    const set = new Set<string>();
    [...employees, ...resignedEmployees].forEach((emp) => {
      const value = emp.major?.trim();
      if (value) set.add(value);
    });
    return Array.from(set).sort((a, b) => a.localeCompare(b, "zh-CN"));
  }, [employees, resignedEmployees]);

  // 字段管理函数
  const handleFieldToggle = (fieldKey: keyof Employee) => {
    const newVisibleFields = visibleFields.includes(fieldKey)
      ? visibleFields.filter((key) => key !== fieldKey)
      : [...visibleFields, fieldKey];

    const sanitized = sanitizeVisibleFieldKeys(newVisibleFields);
    setVisibleFields(sanitized);

    writeLocalStorageJSON(EMPLOYEE_VISIBLE_FIELDS_STORAGE_KEY, sanitized);
  };

  const resetFieldsToDefault = () => {
    const sanitizedDefaults = sanitizeVisibleFieldKeys(DEFAULT_VISIBLE_FIELDS);
    setVisibleFields(sanitizedDefaults);
    const defaultResignedOrder = [
      ...sanitizedDefaults.map((key) => key as string),
      ...RESIGNED_EXTRA_COLUMNS,
    ];
    setResignedColumnOrder(defaultResignedOrder);
    writeLocalStorageJSON(RESIGNED_COLUMN_ORDER_STORAGE_KEY, defaultResignedOrder);
    try {
      if (typeof window !== "undefined" && window.localStorage) {
        window.localStorage.removeItem(EMPLOYEE_VISIBLE_FIELDS_STORAGE_KEY);
        LEGACY_VISIBLE_FIELD_STORAGE_KEYS.forEach((key) => window.localStorage.removeItem(key));
      }
    } catch (error) {
      console.warn("[employee-visible-fields] 清理默认字段缓存失败", error);
    }
    writeLocalStorageJSON(EMPLOYEE_VISIBLE_FIELDS_STORAGE_KEY, sanitizedDefaults);
  };

  const handleProvidentFieldToggle = (columnId: ProvidentColumnId) => {
    setProvidentVisibleColumns((prev) => {
      const exists = prev.includes(columnId);
      let next: ProvidentColumnId[];
      if (exists) {
        next = prev.filter((id) => id !== columnId);
      } else {
        next = [...prev, columnId];
      }
      if (next.length === 0) {
        next = [...DEFAULT_PROVIDENT_VISIBLE_COLUMNS];
      }
      return next;
    });
  };

  const resetProvidentFieldsToDefault = () => {
    setProvidentVisibleColumns([...DEFAULT_PROVIDENT_VISIBLE_COLUMNS]);
    setProvidentColumnOrder([...DEFAULT_PROVIDENT_COLUMN_ORDER]);
  };

  const fieldMap = useMemo(() => new Map(AVAILABLE_FIELDS.map((field) => [field.key, field])), []);
  const visibleFieldConfigs = useMemo(() => {
    return visibleFields
      .map((key) => fieldMap.get(key))
      .filter((field): field is FieldOption => Boolean(field));
  }, [fieldMap, visibleFields]);

  const providentColumnMap = useMemo(
    () => new Map(PROVIDENT_COLUMN_CONFIGS.map((column) => [column.id, column])),
    [],
  );

  const providentColumnsForRender = useMemo(() => {
    const visibleSet = new Set(providentVisibleColumns);
    const ordered = providentColumnOrder
      .map((id) => providentColumnMap.get(id))
      .filter((cfg): cfg is ProvidentColumnConfig => Boolean(cfg && visibleSet.has(cfg.id)));
    if (ordered.length === 0) {
      return DEFAULT_PROVIDENT_VISIBLE_COLUMNS.map((id) => providentColumnMap.get(id))
        .filter((cfg): cfg is ProvidentColumnConfig => Boolean(cfg));
    }
    return ordered;
  }, [providentColumnMap, providentColumnOrder, providentVisibleColumns]);

  const draggingColumnIdRef = useRef<string | null>(null);

  const handleColumnDragOver = useCallback((event: DragEvent<HTMLTableCellElement>) => {
    event.preventDefault();
    if (event.dataTransfer) {
      event.dataTransfer.dropEffect = "move";
    }
  }, []);

  const handleColumnDragStart = useCallback((event: DragEvent<HTMLTableCellElement>, columnId: string) => {
    draggingColumnIdRef.current = columnId;
    if (event.dataTransfer) {
      event.dataTransfer.effectAllowed = "move";
      event.dataTransfer.setData("text/plain", columnId);
    }
  }, []);

  const handleColumnDragEnd = useCallback(() => {
    draggingColumnIdRef.current = null;
  }, []);

  const handleActiveColumnDrop = useCallback((event: DragEvent<HTMLTableCellElement>, targetId: keyof Employee) => {
    event.preventDefault();
    const sourceId = draggingColumnIdRef.current as keyof Employee | null;
    if (!sourceId || sourceId === targetId) {
      draggingColumnIdRef.current = null;
      return;
    }
    setVisibleFields((prev) => {
      const next = reorderList(prev, sourceId, targetId);
      const sanitized = sanitizeVisibleFieldKeys(next);
      writeLocalStorageJSON(EMPLOYEE_VISIBLE_FIELDS_STORAGE_KEY, sanitized);
      return sanitized;
    });
    draggingColumnIdRef.current = null;
  }, [setVisibleFields]);

  const handleResignedColumnDrop = useCallback((event: DragEvent<HTMLTableCellElement>, targetId: string) => {
    event.preventDefault();
    const sourceId = draggingColumnIdRef.current;
    if (!sourceId || sourceId === targetId) {
      draggingColumnIdRef.current = null;
      return;
    }
    setResignedColumnOrder((prev) => reorderList(prev, sourceId, targetId));
    draggingColumnIdRef.current = null;
  }, [setResignedColumnOrder]);

  const handleInsuranceColumnDrop = useCallback((event: DragEvent<HTMLTableCellElement>, targetId: InsuranceColumnId) => {
    event.preventDefault();
    const sourceId = draggingColumnIdRef.current as InsuranceColumnId | null;
    if (!sourceId || sourceId === targetId) {
      draggingColumnIdRef.current = null;
      return;
    }
    setInsuranceColumnOrder((prev) => reorderList(prev, sourceId, targetId));
    draggingColumnIdRef.current = null;
  }, [setInsuranceColumnOrder]);

  const handleProvidentColumnDrop = useCallback((event: DragEvent<HTMLTableCellElement>, targetId: ProvidentColumnId) => {
    event.preventDefault();
    const sourceId = draggingColumnIdRef.current as ProvidentColumnId | null;
    if (!sourceId || sourceId === targetId) {
      draggingColumnIdRef.current = null;
      return;
    }
    setProvidentColumnOrder((prev) => reorderList(prev, sourceId, targetId));
    draggingColumnIdRef.current = null;
  }, []);

  const handleEmployeeSortClick = useCallback((columnId: EmployeeColumnId) => {
    if (draggingColumnIdRef.current) {
      return;
    }
    cycleSort(columnId, setEmployeeSort);
  }, [setEmployeeSort]);

  const handleResignedSortClick = useCallback((columnId: EmployeeColumnId) => {
    if (draggingColumnIdRef.current) {
      return;
    }
    cycleSort(columnId, setResignedSort);
  }, [setResignedSort]);

  const handleInsuranceSortClick = useCallback((columnId: InsuranceColumnId) => {
    if (draggingColumnIdRef.current) {
      return;
    }
    cycleSort(columnId, setInsuranceSort);
  }, [setInsuranceSort]);

  const handleProvidentSortClick = useCallback((columnId: ProvidentColumnId) => {
    if (draggingColumnIdRef.current) {
      return;
    }
    cycleSort(columnId, setProvidentSort);
  }, []);

  const filteredEmployees = employees.filter(emp => {
    const term = searchTerm.toLowerCase();
    const matchesSearch = !term ||
      emp.name.toLowerCase().includes(term) ||
      emp.idNumber.includes(searchTerm) ||
      emp.department?.toLowerCase().includes(term);
    const matchesDepartment = !departmentFilter || departmentFilter === "all" || emp.department === departmentFilter;
    return matchesSearch && matchesDepartment;
  });

  const sortedEmployees = useMemo(
    () =>
      applySort(filteredEmployees, employeeSort, (row, columnId) => {
        if (columnId === "insuranceStatus") {
          return getInsuranceStatusLabel(row, "active");
        }
        return getEmployeeSortValue(row, columnId);
      }),
    [filteredEmployees, employeeSort, getInsuranceStatusLabel],
  );

  const allFilteredSelected = sortedEmployees.length > 0 && sortedEmployees.every((emp) => selectedEmployeeIds.includes(emp.id));
  const hasFilteredSelection = sortedEmployees.some((emp) => selectedEmployeeIds.includes(emp.id));

  const filteredResignedEmployees = resignedEmployees.filter((emp) => {
    const term = resignedSearchTerm.toLowerCase();
    const matchesSearch =
      !term ||
      emp.name.toLowerCase().includes(term) ||
      emp.idNumber.includes(resignedSearchTerm) ||
      emp.department?.toLowerCase().includes(term);
    const matchesDepartment = !resignedDepartmentFilter || resignedDepartmentFilter === "all" || emp.department === resignedDepartmentFilter;
    return matchesSearch && matchesDepartment;
  });

  const sortedResignedEmployees = useMemo(
    () =>
      applySort(filteredResignedEmployees, resignedSort, (row, columnId) => {
        if (columnId === "insuranceStatus") {
          return getInsuranceStatusLabel(row, "resigned");
        }
        return getEmployeeSortValue(row, columnId);
      }),
    [filteredResignedEmployees, resignedSort, getInsuranceStatusLabel],
  );

  const resignedColumnsForRender = useMemo<ResignedColumnDefinition[]>(() => {
    const baseConfigs: ResignedColumnDefinition[] = visibleFieldConfigs.map((field) => ({
      id: field.key,
      label: field.label,
      sortable: true,
      type: "base",
      getValue: (row: Employee) =>
        field.key === "insuranceStatus" ? getInsuranceStatusLabel(row, "resigned") : getEmployeeSortValue(row, field.key),
    }));
    const extraConfigs: ResignedColumnDefinition[] = [
      {
        id: "resignDate",
        label: "离职日期",
        sortable: true,
        type: "resignDate",
        width: "120px",
        getValue: (row: Employee) => row.resignDate ?? "",
      },
      {
        id: "resignReasons",
        label: "离职原因",
        sortable: false,
        type: "resignReasons",
        width: "160px",
        getValue: (row: Employee) => (row.resignReasons && row.resignReasons.length > 0 ? row.resignReasons.join("，") : ""),
      },
      {
        id: "resignProof",
        label: "离职证明",
        sortable: false,
        type: "resignProof",
        width: "160px",
      },
    ];
    const configMap = new Map<string, ResignedColumnDefinition>([...baseConfigs, ...extraConfigs].map((cfg) => [cfg.id, cfg]));
    const fallbackOrder = [...visibleFields.map((field) => field as string), ...RESIGNED_EXTRA_COLUMNS];
    const order = resignedColumnOrder.length > 0 ? resignedColumnOrder : fallbackOrder;
    return order
      .map((id) => configMap.get(id))
      .filter((cfg): cfg is ResignedColumnDefinition => Boolean(cfg));
  }, [visibleFieldConfigs, resignedColumnOrder, visibleFields, getInsuranceStatusLabel]);

  useEffect(() => {
    if (selectAllRef.current) {
      selectAllRef.current.indeterminate = hasFilteredSelection && !allFilteredSelected;
    }
  }, [hasFilteredSelection, allFilteredSelected, sortedEmployees.length]);

  const toggleEmployeeSelection = (employeeId: string, checked: boolean) => {
    setSelectedEmployeeIds((prev) => {
      if (checked) {
        if (prev.includes(employeeId)) {
          return prev;
        }
        return [...prev, employeeId];
      }
      return prev.filter((id) => id !== employeeId);
    });
  };

  const toggleSelectAllFiltered = (checked: boolean) => {
    if (checked) {
      const idsToAdd = sortedEmployees.map((emp) => emp.id);
      setSelectedEmployeeIds(idsToAdd);
    } else {
      const filteredIdSet = new Set(sortedEmployees.map((emp) => emp.id));
      setSelectedEmployeeIds((prev) => prev.filter((id) => !filteredIdSet.has(id)));
    }
  };

  const toggleResignedSelection = (employeeId: string, checked: boolean) => {
    setSelectedResignedIds((prev) => {
      if (checked) {
        if (prev.includes(employeeId)) {
          return prev;
        }
        return [...prev, employeeId];
      }
      return prev.filter((id) => id !== employeeId);
    });
  };

  const toggleAllResigned = (checked: boolean) => {
    if (checked) {
      setSelectedResignedIds(sortedResignedEmployees.map((emp) => emp.id));
    } else {
      setSelectedResignedIds([]);
    }
  };

  const toggleInsuranceChangeSelection = (changeId: string, checked: boolean) => {
    setSelectedInsuranceChangeIds((prev) => {
      if (checked) {
        if (prev.includes(changeId)) {
          return prev;
        }
        return [...prev, changeId];
      }
      return prev.filter((id) => id !== changeId);
    });
  };

  const toggleAllInsuranceChanges = (checked: boolean) => {
    if (checked) {
      setSelectedInsuranceChangeIds(sortedInsuranceChanges.map((item) => item.id));
    } else {
      setSelectedInsuranceChangeIds([]);
    }
  };

  const handleInsuranceFieldToggle = useCallback(
    (view: InsuranceViewKey, columnId: InsuranceColumnId, checked: boolean) => {
      const allowedSet = new Set(ALLOWED_INSURANCE_COLUMNS_BY_VIEW[view]);
      if (!allowedSet.has(columnId)) {
        return;
      }
      setInsuranceVisibleColumns((prev) => {
        const current = prev[view] ?? DEFAULT_INSURANCE_VISIBLE_MAP[view];
        const next = new Set(current);
        if (checked) {
          next.add(columnId);
        } else if (!REQUIRED_INSURANCE_COLUMNS.includes(columnId)) {
          next.delete(columnId);
        }
        // 确保必选列始终存在
        REQUIRED_INSURANCE_COLUMNS.forEach((required) => {
          if (allowedSet.has(required)) {
            next.add(required);
          }
        });
        const normalized = Array.from(next).filter((id) => allowedSet.has(id));
        return {
          ...prev,
          [view]: normalized.length > 0 ? normalized : DEFAULT_INSURANCE_VISIBLE_MAP[view],
        };
      });
    },
    [],
  );

  const resetInsuranceFieldsToDefault = useCallback(
    (view: InsuranceViewKey) => {
      setInsuranceVisibleColumns((prev) => ({
        ...prev,
        [view]: DEFAULT_INSURANCE_VISIBLE_MAP[view],
      }));
    },
    [],
  );

  const toggleProvidentSelection = (recordId: number, checked: boolean) => {
    setSelectedProvidentIds((prev) => {
      if (checked) {
        if (prev.includes(recordId)) {
          return prev;
        }
        return [...prev, recordId];
      }
      return prev.filter((id) => id !== recordId);
    });
  };

  const toggleAllProvidentRecords = (checked: boolean) => {
    if (checked) {
      setSelectedProvidentIds(sortedProvidentRecords.map((record) => record.id));
    } else {
      setSelectedProvidentIds([]);
    }
  };

  const filteredInsuranceChanges = useMemo(() => {
    if (insuranceView === "provident") {
      return [];
    }
    const normalizedSearch = insuranceSearch.trim().toLowerCase();
    return socialInsuranceChanges
      .filter((change) => change.changeType === insuranceView)
      .filter((change) => {
        if (insuranceDepartmentFilter === "all") {
          return true;
        }
        return change.department === insuranceDepartmentFilter;
      })
      .filter((change) => {
        if (insuranceReasonFilter === "all") {
          return true;
        }
        return change.reason === insuranceReasonFilter;
      })
      .filter((change) => {
        if (!normalizedSearch) {
          return true;
        }
        const candidates: string[] = [
          change.employeeName,
          change.identityNumber,
          change.personalNumber,
          change.department,
          change.reason,
          change.effectiveDate,
          change.originalFileName ?? "",
        ];
        Object.values(change.templateValues ?? {}).forEach((value) => {
          if (typeof value === "string") {
            candidates.push(value);
          }
        });
        return candidates.some((value) => value && value.toLowerCase().includes(normalizedSearch));
      });
  }, [insuranceDepartmentFilter, insuranceReasonFilter, insuranceSearch, insuranceView, socialInsuranceChanges]);

  const insuranceCounts = useMemo(() => {
    const summary: Record<Extract<InsuranceView, "increase" | "decrease">, number> = {
      increase: 0,
      decrease: 0,
    };
    socialInsuranceChanges.forEach((change) => {
      if (change.changeType === "increase") {
        summary.increase += 1;
      } else if (change.changeType === "decrease") {
        summary.decrease += 1;
      }
    });
    return summary;
  }, [socialInsuranceChanges]);

  const insuranceDepartmentOptions = useMemo(() => {
    const set = new Set<string>();
    socialInsuranceChanges.forEach((item) => {
      if (item.department) {
        set.add(item.department);
      }
    });
    return Array.from(set).sort();
  }, [socialInsuranceChanges]);

  const insuranceReasonOptions = useMemo(() => {
    const set = new Set<string>();
    socialInsuranceChanges.forEach((item) => {
      if (item.reason) {
        set.add(item.reason);
      }
    });
    return Array.from(set).sort();
  }, [socialInsuranceChanges]);

  const insuranceColumnsForRender = useMemo<ColumnConfig<SocialInsuranceRecord>[]>(() => {
    if (insuranceView === "provident") {
      return [];
    }
    const baseConfigs: ColumnConfig<SocialInsuranceRecord>[] = BASE_INSURANCE_COLUMNS.map((column) => {
      let getter: (row: SocialInsuranceRecord) => unknown;
      switch (column.id) {
        case "changeType":
          getter = (row) => (row.changeType === "increase" ? "增加" : "减少");
          break;
        case "employeeName":
          getter = (row) => row.employeeName ?? "";
          break;
        case "identityNumber":
          getter = (row) => row.identityNumber ?? "";
          break;
        case "personalNumber":
          getter = (row) => row.personalNumber ?? "";
          break;
        case "department":
          getter = (row) => row.department ?? "";
          break;
        case "effectiveDate":
          getter = (row) => displayDate(row.effectiveDate);
          break;
        case "reason":
          getter = (row) => row.reason ?? "";
          break;
        case "createdAt":
          getter = (row) => displayDate(row.createdAt);
          break;
        case "originalFileName":
          getter = (row) => row.originalFileName ?? "";
          break;
        default:
          getter = () => "";
      }
      return {
        id: column.id,
        label: column.label,
        sortable: true,
        getValue: getter,
      };
    });

    const viewKey: InsuranceViewKey = insuranceView === "increase" ? "increase" : "decrease";
    const allowedTemplateConfigs: ColumnConfig<SocialInsuranceRecord>[] = TEMPLATE_ENTRIES_BY_VIEW[viewKey].map((entry) => ({
      id: entry.id,
      label: entry.field.label,
      sortable: true,
      getValue: (row) => {
        const raw = row.templateValues?.[entry.field.key] ?? "";
        if (!raw) {
          return "";
        }
        if (entry.field.isDate) {
          return displayDate(raw);
        }
        return raw;
      },
    }));

    const configEntries = [...baseConfigs, ...allowedTemplateConfigs];
    const configMap = new Map<string, ColumnConfig<SocialInsuranceRecord>>(configEntries.map((cfg) => [cfg.id, cfg]));
    const allowedIds = new Set(ALLOWED_INSURANCE_COLUMNS_BY_VIEW[viewKey]);

    const preferredVisible = insuranceVisibleColumns[viewKey] ?? DEFAULT_INSURANCE_VISIBLE_MAP[viewKey];
    const filteredVisible = preferredVisible.filter((id) => allowedIds.has(id));
    const resolvedVisible =
      filteredVisible.length > 0
        ? filteredVisible
        : DEFAULT_INSURANCE_VISIBLE_MAP[viewKey].filter((id) => allowedIds.has(id));
    const visibleSet = new Set<InsuranceColumnId>(
      resolvedVisible.length > 0 ? resolvedVisible : ALLOWED_INSURANCE_COLUMNS_BY_VIEW[viewKey],
    );

    const baseOrder = insuranceColumnOrder.length > 0 ? insuranceColumnOrder : DEFAULT_INSURANCE_COLUMN_ORDER;
    const orderedColumns = baseOrder.filter(
      (id) => allowedIds.has(id as InsuranceColumnId) && visibleSet.has(id as InsuranceColumnId),
    ) as InsuranceColumnId[];
    const finalOrder = orderedColumns.length > 0 ? orderedColumns : Array.from(visibleSet);

    return finalOrder
      .map((id) => configMap.get(id))
      .filter((cfg): cfg is ColumnConfig<SocialInsuranceRecord> => Boolean(cfg));
  }, [insuranceColumnOrder, insuranceVisibleColumns, insuranceView]);

  const insuranceColumnGetterMap = useMemo(() => {
    const map = new Map<string, (row: SocialInsuranceRecord) => unknown>();
    insuranceColumnsForRender.forEach((column) => {
      if (column.getValue) {
        map.set(column.id, column.getValue);
      }
    });
    return map;
  }, [insuranceColumnsForRender]);

  const sortedInsuranceChanges = useMemo(
    () =>
      applySort(filteredInsuranceChanges, insuranceSort, (row, columnId) => {
        const getter = insuranceColumnGetterMap.get(columnId);
        return getter ? getter(row) : "";
      }),
    [filteredInsuranceChanges, insuranceSort, insuranceColumnGetterMap],
  );

  const filteredProvidentRecords = useMemo(() => {
    const normalizedSearch = providentSearch.trim().toLowerCase();
    return providentRecords.filter((record) => {
      const matchesStatus = providentStatusFilter === "all" || record.status === providentStatusFilter;
      if (!matchesStatus) {
        return false;
      }
      if (!normalizedSearch) {
        return true;
      }
      const candidates = [
        record.personal_account,
        record.name,
        record.identity_number,
        record.notes,
      ]
        .filter(Boolean)
        .map((value) => value!.toLowerCase());
      return candidates.some((value) => value.includes(normalizedSearch));
    });
  }, [providentRecords, providentSearch, providentStatusFilter]);

  const sortedProvidentRecords = useMemo(
    () => applySort(filteredProvidentRecords, providentSort, (row, columnId) => getProvidentSortValue(row, columnId)),
    [filteredProvidentRecords, providentSort],
  );

  const selectedProvidentRecords = useMemo(
    () => sortedProvidentRecords.filter((record) => selectedProvidentIds.includes(record.id)),
    [sortedProvidentRecords, selectedProvidentIds],
  );

  const providentSummary = useMemo(() => {
    return filteredProvidentRecords.reduce(
      (acc, record) => {
        acc.personal += record.personal_amount ?? 0;
        acc.company += record.company_amount ?? 0;
        acc.total += record.total_amount ?? 0;
        return acc;
      },
      { personal: 0, company: 0, total: 0 },
    );
  }, [filteredProvidentRecords]);

  const allProvidentFilteredSelected =
    sortedProvidentRecords.length > 0 && sortedProvidentRecords.every((record) => selectedProvidentIds.includes(record.id));
  const hasProvidentFilteredSelection = sortedProvidentRecords.some((record) => selectedProvidentIds.includes(record.id));
  const hasProvidentSelection = selectedProvidentIds.length > 0;

  useEffect(() => {
    if (providentSelectAllRef.current) {
      providentSelectAllRef.current.indeterminate =
        hasProvidentFilteredSelection && !allProvidentFilteredSelected;
    }
  }, [hasProvidentFilteredSelection, allProvidentFilteredSelected, sortedProvidentRecords.length]);

  // 添加员工
  const handleAddEmployee = () => {
    // 必填字段验证
    if (!newEmployee.name?.trim()) {
      toast.error("请填写姓名");
      return;
    }
    if (!newEmployee.idNumber?.trim()) {
      toast.error("请填写身份证号");
      return;
    }
    if (!newEmployee.department?.trim()) {
      toast.error("请选择部门");
      return;
    }

    // 身份证号格式验证
    const idRegex = /^[1-9]\d{5}(18|19|([23]\d))\d{2}((0[1-9])|(10|11|12))(([0-2][1-9])|10|20|30|31)\d{3}[0-9Xx]$/;
    if (!idRegex.test(newEmployee.idNumber)) {
      toast.error("身份证号格式不正确");
      return;
    }

    // 手机号格式验证（如果填写了）
    if (newEmployee.phone && !/^1[3-9]\d{9}$/.test(newEmployee.phone)) {
      toast.error("手机号格式不正确");
      return;
    }

    // 邮箱格式验证（如果填写了）
    if (newEmployee.email && !/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(newEmployee.email)) {
      toast.error("邮箱格式不正确");
      return;
    }

    // 年龄验证（如果填写了）
    if (newEmployee.age && (isNaN(Number(newEmployee.age)) || Number(newEmployee.age) < 16 || Number(newEmployee.age) > 100)) {
      toast.error("年龄应为16-100之间的数字");
      return;
    }

    // 工龄验证（如果填写了）
    if (newEmployee.workYears && (isNaN(Number(newEmployee.workYears)) || Number(newEmployee.workYears) < 0)) {
      toast.error("工龄应为非负数字");
      return;
    }

    // 检查身份证号是否重复
    const sanitizedIdNumber = newEmployee.idNumber.trim().toUpperCase();
    const existingEmployee = employees.find(emp => emp.idNumber === sanitizedIdNumber);
    if (existingEmployee) {
      toast.error("该身份证号已存在");
      return;
    }

    const autoInfo = calculateAgeFromIdNumber(sanitizedIdNumber);
    const normalizedHireDate = normalizeDateInput(newEmployee.hireDate || formatDateToInput(new Date()));
    const normalizedAge = formatAgeValue(newEmployee.age || autoInfo.age);
    const normalizedWorkYears = formatWorkYearsValue(newEmployee.workYears || calculateWorkYears(normalizedHireDate));
    const normalizedBirthMonth = normalizeBirthMonth(newEmployee.birthMonth || autoInfo.birthMonth);

    const employee: Employee = {
      id: Date.now().toString(),
      employeeId: newEmployee.employeeId || Date.now().toString(),
      name: newEmployee.name.trim(),
      department: newEmployee.department.trim(),
      position: newEmployee.position?.trim() || "",
      gender: newEmployee.gender || "",
      hireDate: normalizedHireDate,
      age: normalizedAge,
      workYears: normalizedWorkYears,
  birthMonth: normalizedBirthMonth,
      education: newEmployee.education?.trim() || "",
      politicalStatus: newEmployee.politicalStatus?.trim() || "",
      workClothingSize: newEmployee.workClothingSize?.trim() || "",
      safetyShoeSize: newEmployee.safetyShoeSize?.trim() || "",
      householdType: newEmployee.householdType?.trim() || "",
      ethnicity: newEmployee.ethnicity?.trim() || "",
      nativePlace: newEmployee.nativePlace?.trim() || "",
      idAddress: newEmployee.idAddress?.trim() || "",
      idNumber: sanitizedIdNumber,
      maritalStatus: newEmployee.maritalStatus?.trim() || "",
      insuranceStatus: newEmployee.insuranceStatus?.trim() || "",
      hasBirth: newEmployee.hasBirth?.trim() || "",
      phone: newEmployee.phone?.trim() || "",
      emergencyContact: newEmployee.emergencyContact?.trim() || "",
      emergencyPhone: newEmployee.emergencyPhone?.trim() || "",
      currentAddress: newEmployee.currentAddress?.trim() || "",
      graduateSchool: newEmployee.graduateSchool?.trim() || "",
      major: newEmployee.major?.trim() || "",
      graduationTime: normalizeDateInput(newEmployee.graduationTime),
      socialInsuranceNumber: newEmployee.socialInsuranceNumber?.trim() || "",
      providentFundNumber: newEmployee.providentFundNumber?.trim() || "无",
      email: newEmployee.email?.trim() || "",
      remarks: newEmployee.remarks?.trim() || "",
      status: 'active',
    };

    setEmployees(prev => [...prev, employee]);
    setNewEmployee({
      ...createEmptyEmployee(),
      education: "初中",
      politicalStatus: "群众",
      workClothingSize: "L",
      safetyShoeSize: "40",
      insuranceStatus: "",
    });
    setShowAddEmployee(false);
    toast.success("员工添加成功");
  };

  // 批量导入员工
  const handleBatchImport = async () => {
    const file = fileInputRef.current?.files?.[0];
    if (!file) {
      toast.error("请选择文件");
      return;
    }

    setIsLoading(true);
    try {
    const token = localStorage.getItem("token");
    if (!token) {
      toast.error("登录状态失效，请重新登录后再导入");
      setIsLoading(false);
      return;
    }

      if (!token) {
        toast.error("登录状态失效，请重新登录后再导入");
        return;
      }

      const result = await importEmployeesApi(file, token, importMode);
      applyEmployeeData(result.employees);
      setShowBatchImport(false);
      const summary: string[] = [];
      if (result.inserted > 0) {
        summary.push(`新增 ${result.inserted} 人`);
      }
      if (result.updated > 0) {
        summary.push(`更新 ${result.updated} 人`);
      }
      if (result.skipped > 0) {
        summary.push(`跳过 ${result.skipped} 行`);
      }
      toast.success(`导入完成：${summary.length > 0 ? summary.join("，") : "未导入任何数据"}`);
    } catch (error) {
      console.error("[EmployeeImport] failed", error);
      toast.error(
        error instanceof Error
          ? error.message
          : "导入失败，请确认文件符合模板要求",
      );
    } finally {
      setIsLoading(false);
      if (fileInputRef.current) {
        fileInputRef.current.value = "";
      }
    }
  };

const handleResignedImport = async () => {
    const file = resignedFileInputRef.current?.files?.[0];
    if (!file) {
      toast.error("请选择文件");
      return;
    }

    const token = localStorage.getItem("token");
    if (!token) {
      toast.error("登录状态失效，请重新登录后再导入");
      return;
    }

    setResignedImporting(true);
    let conflictHandled = false;
    try {
      const result = await importResignedEmployeesApi(file, token, { mode: resignedImportMode, forceTransition: false });
      applyEmployeeData(result.employees);
      setShowResignedImport(false);
      const summary: string[] = [];
      if (result.inserted > 0) summary.push(`新增 ${result.inserted} 人`);
      if (result.updated > 0) summary.push(`更新 ${result.updated} 人`);
      if (result.skipped > 0) summary.push(`跳过 ${result.skipped} 行`);
      toast.success(`离职员工导入完成：${summary.length > 0 ? summary.join("，") : "未导入任何数据"}`);
    } catch (error) {
      console.error("[ResignedImport] failed", error);
      if (error instanceof EmployeeImportConflictError) {
        conflictHandled = true;
        setPendingResignedImport({ file, mode: resignedImportMode });
        setResignedConflicts(error.conflicts || []);
        setShowResignedConflictDialog(true);
        toast.info("检测到在职员工与导入记录身份证重复，请确认是否转为离职状态");
      } else {
        toast.error(error instanceof Error ? error.message : "离职员工导入失败，请确认文件符合模板要求");
      }
    } finally {
      setResignedImporting(false);
      if (resignedFileInputRef.current) {
        resignedFileInputRef.current.value = "";
      }
      if (conflictHandled) {
        return;
      }
      setPendingResignedImport(null);
      setResignedConflicts([]);
      setShowResignedConflictDialog(false);
    }
  };

  const handleConfirmResignedConflicts = async () => {
    if (!pendingResignedImport) {
      setShowResignedConflictDialog(false);
      return;
    }
    const token = localStorage.getItem("token");
    if (!token) {
      toast.error("登录状态失效，请重新登录后再导入");
      return;
    }
    setResignedConflictImporting(true);
    try {
      const result = await importResignedEmployeesApi(pendingResignedImport.file, token, {
        mode: pendingResignedImport.mode,
        forceTransition: true,
      });
      applyEmployeeData(result.employees);
      toast.success("已将冲突的在职员工转为离职状态并完成导入");
      setShowResignedImport(false);
      setShowResignedConflictDialog(false);
      setPendingResignedImport(null);
      setResignedConflicts([]);
    } catch (error) {
      console.error("[ResignedImport] force failed", error);
      toast.error(error instanceof Error ? error.message : "导入失败，请稍后重试");
    } finally {
      setResignedConflictImporting(false);
    }
  };

  const handleCancelResignedConflicts = () => {
    setShowResignedConflictDialog(false);
    setPendingResignedImport(null);
    setResignedConflicts([]);
  };

  const handleBatchDelete = async () => {
    if (selectedEmployeeIds.length === 0) {
      toast.error("请先选择需要删除的在职员工");
      setShowBatchDeleteConfirm(false);
      return;
    }

    const numericIds = selectedEmployeeIds
      .map((id) => Number(id))
      .filter((value) => Number.isFinite(value) && value > 0)
      .map((value) => Math.trunc(value));

    if (numericIds.length === 0) {
      toast.error("所选记录中没有可删除的员工");
      setShowBatchDeleteConfirm(false);
      return;
    }

    setDeletingEmployees(true);
    try {
      const result = await deleteEmployeesApi(numericIds);
      const removedIds = new Set(result.ids.map((id) => String(id)));
      setEmployees((prev) => prev.filter((emp) => !removedIds.has(emp.id)));
      setSelectedEmployeeIds([]);
      toast.success(`成功删除 ${result.deleted} 名在职员工`);
    } catch (error) {
      console.error("[EmployeeManagement] delete employees failed", error);
      toast.error(error instanceof Error ? error.message : "删除失败，请稍后重试");
    } finally {
      setDeletingEmployees(false);
      setShowBatchDeleteConfirm(false);
    }
  };

  const handleExportResigned = async (scope: "selected" | "filtered" | "all") => {
    if (resignedExporting) {
      return;
    }

    if (scope === "selected" && selectedResignedIds.length === 0) {
      toast.error("请先勾选需要导出的离职员工");
      return;
    }

    const trimmedSearch = resignedSearchTerm.trim();
    const numericIds = selectedResignedIds
      .map((id) => Number(id))
      .filter((value) => Number.isFinite(value) && value > 0)
      .map((value) => Math.trunc(value));
    const fallbackIdNumbers = selectedResignedIds
      .filter((id) => Number.isNaN(Number(id)))
      .map((id) => id.trim())
      .filter(Boolean);
    const idsParam = scope === "selected" && numericIds.length > 0 ? numericIds : undefined;
    const idNumbersParam = scope === "selected" && fallbackIdNumbers.length > 0 ? fallbackIdNumbers : undefined;

    setResignedExporting(true);
    try {
      const blob = await exportEmployees({
        scope,
        status: "resigned",
        department: scope === "filtered" && resignedDepartmentFilter !== "all" ? resignedDepartmentFilter : undefined,
        search: scope === "filtered" && trimmedSearch ? trimmedSearch : undefined,
        ids: idsParam,
        idNumbers: idNumbersParam,
      });

      downloadBlob(blob, `离职员工-${scope}-${new Date().toISOString().slice(0, 10)}.xlsx`);
      toast.success("离职员工导出完成");
    } catch (error) {
      console.error("[ResignedExport] failed", error);
      toast.error(error instanceof Error ? error.message : "导出失败，请稍后再试");
    } finally {
      setResignedExporting(false);
    }
  };

  const handleBatchRestore = async () => {
    const ids = new Set(selectedResignedIds);
    if (ids.size === 0) {
      setShowBatchRestoreConfirm(false);
      return;
    }

    const targetEmployees = resignedEmployees.filter((emp) => ids.has(emp.id));
    if (targetEmployees.length === 0) {
      setShowBatchRestoreConfirm(false);
      return;
    }

    const numericIds = targetEmployees
      .map((emp) => Number(emp.id))
      .filter((value) => Number.isFinite(value) && value > 0)
      .map((value) => Math.trunc(value));
    const idNumbers = Array.from(
      new Set(
        targetEmployees
          .map((emp) => emp.idNumber?.trim())
          .filter((value): value is string => Boolean(value)),
      ),
    );

    if (numericIds.length === 0 && idNumbers.length === 0) {
      toast.error("当前所选员工缺少有效标识，无法撤销离职");
      return;
    }

    if (!token) {
      toast.error("登录状态失效，请重新登录后再执行撤销");
      return;
    }

    setRestoringResigned(true);
    try {
      const result = await restoreEmployees(
        {
          ids: numericIds.length > 0 ? numericIds : undefined,
          idNumbers: idNumbers.length > 0 ? idNumbers : undefined,
        },
        token ?? undefined,
      );

      const restoredCount = typeof result?.restored === "number" ? result.restored : targetEmployees.length;
      toast.success(`已恢复 ${restoredCount} 名员工至在职状态`);
      setShowBatchRestoreConfirm(false);
      setSelectedResignedIds([]);
      if (printContext?.type === "resigned") {
        handleClosePrintDialog();
      }
      await loadEmployees();
    } catch (error) {
      console.error("[EmployeeManagement] restore failed", error);
      toast.error(error instanceof Error ? error.message : "撤销离职失败，请稍后重试");
    } finally {
      setRestoringResigned(false);
    }
  };

const buildIncreaseRow = (record: SocialInsuranceRecord) => {
  const values = record.templateValues ?? {};
  return [
    values.idNumber ?? record.identityNumber ?? "",
    values.name ?? record.employeeName ?? "",
    values.nationality ?? "",
    formatDateForTemplate(values.firstWorkDate ?? ""),
    values.baseSalary ?? "",
    values.job ?? "",
    values.personalIdentity ?? "",
    values.householdType ?? "",
    values.phone ?? "",
    values.education ?? "",
    formatDateForTemplate(values.pensionStartDate ?? record.effectiveDate ?? ""),
    formatDateForTemplate(values.unemploymentStartDate ?? ""),
    formatDateForTemplate(values.medicalStartDate ?? ""),
    formatDateForTemplate(values.injuryStartDate ?? ""),
    formatDateForTemplate(values.maternityStartDate ?? ""),
    values.specialSkill ?? "",
    values.skillLevel ?? "",
    values.remark ?? record.reason ?? "",
  ].map(normalizeTemplateCellValue);
};

const buildDecreaseRow = (record: SocialInsuranceRecord) => {
  const values = record.templateValues ?? {};
  return [
    values.personalNumber ?? record.personalNumber ?? "",
    values.idNumber ?? record.identityNumber ?? "",
    values.name ?? record.employeeName ?? "",
    formatDateForTemplate(values.decreaseDate ?? record.effectiveDate ?? ""),
    values.decreaseReason ?? record.reason ?? "",
    values.pensionFlag ?? "",
    values.unemploymentFlag ?? "",
    values.medicalFlag ?? "",
    values.injuryFlag ?? "",
    values.maternityFlag ?? "",
    values.unemploymentReason ?? "",
  ].map(normalizeTemplateCellValue);
};

const INCREASE_EXPORT_HEADERS = [
  "证件号码",
  "姓名",
  "民族",
  "首次参加工作日期",
  "月基本工资额",
  "工种",
  "个人身份",
  "户口性质",
  "联系电话",
  "文化程度",
  "养老保险参保时间",
  "失业保险参保时间",
  "医疗保险参保时间",
  "工伤保险参保时间",
  "生育保险参保时间",
  "专职技能",
  "技能等级",
  "备注",
];

const DECREASE_EXPORT_HEADERS = [
  "个人编号",
  "证件号码",
  "姓名",
  "减少时间",
  "减少原因",
  "养老保险减少标志",
  "失业保险减少标志",
  "医疗保险减少标志",
  "工伤保险减少标志",
  "生育保险减少标志",
  "失业原因",
];

const exportInsuranceChanges = async (
  type: SocialInsuranceChangeType,
  dataset: SocialInsuranceRecord[],
  unitInfo: UnitInfo,
) => {
  const templateBlob = await downloadInsuranceTemplate(type);
  const buffer = await templateBlob.arrayBuffer();
  const workbook = XLSX.read(buffer, { type: "array", raw: false, cellDates: true });
  if (workbook.Custprops) {
    delete workbook.Custprops;
  }
  if (workbook.Props) {
    delete workbook.Props;
  }
  if (workbook.Workbook && (workbook.Workbook as Record<string, unknown>).Custprops) {
    delete (workbook.Workbook as Record<string, unknown>).Custprops;
  }
  const sheetName = workbook.SheetNames[0];
  if (!sheetName) {
    throw new Error("模板缺少工作表");
  }
  const sheet = workbook.Sheets[sheetName];
  if (!sheet) {
    throw new Error("模板数据不可用");
  }

  const writeCell = (address: string, value: string) => {
    XLSX.utils.sheet_add_aoa(sheet, [[value]], { origin: address });
  };

  const trimmedCode = unitInfo.socialCode.trim();
  const trimmedName = unitInfo.unitName.trim();
  if (trimmedCode) {
    writeCell("B1", trimmedCode);
  }
  if (trimmedName) {
    writeCell("D1", trimmedName);
  }
  // 部分模板会在 C1 写入默认名称，保持提示语一致
  writeCell("C1", "单位名称：\n（D1栏填写）");

  const clearDataRows = () => {
    const ref = sheet["!ref"];
    if (!ref) {
      return;
    }
    const range = XLSX.utils.decode_range(ref);
    for (let row = 2; row <= range.e.r; row += 1) {
      for (let col = range.s.c; col <= range.e.c; col += 1) {
        const cellAddress = XLSX.utils.encode_cell({ r: row, c: col });
        if (sheet[cellAddress]) {
          delete sheet[cellAddress];
        }
      }
    }
  };
  clearDataRows();

  const paddedRows =
    type === "increase"
      ? dataset.map((record) => ["", ...buildIncreaseRow(record)])
      : dataset.map((record) => ["", ...buildDecreaseRow(record)]);

  const normalizedRows = paddedRows.map((row) => row.map((cell) => normalizeTemplateCellValue(cell)));

  const origin = { r: 2, c: 0 };
  XLSX.utils.sheet_add_aoa(sheet, normalizedRows, { origin });

  const columnCount =
    (normalizedRows[0]?.length ??
      (type === "increase" ? INCREASE_EXPORT_HEADERS.length + 1 : DECREASE_EXPORT_HEADERS.length + 1)) || 1;
  const totalRows = Math.max(2 + normalizedRows.length, 2);
  sheet["!ref"] = XLSX.utils.encode_range({ r: 0, c: 0 }, { r: totalRows - 1, c: columnCount - 1 });

  const workbookArray = XLSX.write(workbook, { bookType: "xls", type: "array" });
  const blob = new Blob([workbookArray], { type: "application/vnd.ms-excel" });
  const suffix = type === "increase" ? "增加" : "减少";
  downloadBlob(blob, `社保${suffix}-${new Date().toISOString().slice(0, 10)}.xls`);
};

  const handleExportInsurance = async (scope: "selected" | "filtered" | "all") => {
    if (insuranceView === "provident") {
      toast.error("公积金模块暂未开放导出功能");
      return;
    }
    if (insuranceExporting) {
      return;
    }

    if (scope === "selected" && selectedInsuranceChangeIds.length === 0) {
      toast.error("请先勾选需要导出的社保变动记录");
      return;
    }

    const dataset = (() => {
      switch (scope) {
        case "selected":
          return filteredInsuranceChanges.filter((change) => selectedInsuranceChangeIds.includes(change.id));
        case "filtered":
          return filteredInsuranceChanges;
        default:
          return socialInsuranceChanges.filter((change) => change.changeType === insuranceView);
      }
    })();

    if (dataset.length === 0) {
      toast.error(insuranceView === "increase" ? "没有可导出的社保增加记录" : "没有可导出的社保减少记录");
      return;
    }

    const expectedType = dataset[0].changeType;
    const hasMismatch = dataset.some((item) => item.changeType !== expectedType);
    if (hasMismatch) {
      toast.error("请选择同类型的社保变动记录再导出");
      return;
    }

    setInsuranceExporting(true);
    try {
      await exportInsuranceChanges(expectedType, dataset, unitInfo);
      toast.success("社保变动导出完成");
    } catch (error) {
      console.error("[InsuranceExport] failed", error);
      toast.error(error instanceof Error ? error.message : "导出失败，请稍后再试");
    } finally {
      setInsuranceExporting(false);
    }
  };

  const handleBatchInsuranceDelete = async () => {
    if (insuranceView === "provident") {
      toast.error("公积金模块暂未开放撤销功能");
      return;
    }
    if (!token) {
      toast.error("登录状态失效，请重新登录后再试");
      return;
    }
    const numericIds = selectedInsuranceChangeIds
      .map((id) => Number(id))
      .filter((value) => Number.isFinite(value) && value > 0)
      .map((value) => Math.trunc(value));

    if (numericIds.length === 0) {
      toast.error("请选择包含有效编号的社保变动记录");
      return;
    }

    try {
      await deleteSocialInsuranceChanges(numericIds, token);
      toast.success(`已撤销 ${numericIds.length} 条社保变动记录`);
      setSelectedInsuranceChangeIds([]);
      setShowBatchInsuranceConfirm(false);
      await loadSocialInsuranceData();
    } catch (error) {
      console.error("[InsuranceDelete] failed", error);
      toast.error(error instanceof Error ? error.message : "撤销失败，请稍后再试");
    }
  };

  const handleResignProofChange = (event: ChangeEvent<HTMLInputElement>) => {
    const file = event.target.files?.[0] ?? null;
    if (!file) {
      setResignProofFile(null);
      setResignProofError("");
      return;
    }

    const extension = file.name.split(".").pop()?.toLowerCase() ?? "";
    const isValidType = isAllowedResignProofType(file.type, extension);

    if (!isValidType) {
      setResignProofFile(null);
      setResignProofError("离职证明仅支持 PDF 或图片格式");
      event.target.value = "";
      return;
    }

    if (file.size > MAX_RESIGN_PROOF_SIZE_BYTES) {
      setResignProofFile(null);
      setResignProofError(`文件体积超出限制，单个文件不得超过 ${formatFileSize(MAX_RESIGN_PROOF_SIZE_BYTES)}`);
      event.target.value = "";
      return;
    }

    setResignProofFile(file);
    setResignProofError("");
  };

  const openProvidentDialog = useCallback(
    (mode: "create" | "edit", options?: { record?: ProvidentFundRecord; employee?: Employee }) => {
      if (mode === "edit" && options?.record) {
        setProvidentForm(mapRecordToFormState(options.record));
        setEditingProvidentRecord(options.record);
      } else if (options?.employee) {
        setProvidentForm({
          personal_account:
            options.employee.providentFundNumber && options.employee.providentFundNumber !== "无"
              ? options.employee.providentFundNumber
              : "",
          name: options.employee.name ?? "",
          identity_number: options.employee.idNumber ?? "",
          personal_base: "",
          personal_amount: "",
          company_amount: "",
          contribution_ratio: DEFAULT_PROVIDENT_RATIO.toString(),
          notes: "",
        });
        setEditingProvidentRecord(null);
      } else {
        setProvidentForm(createEmptyProvidentForm());
        setEditingProvidentRecord(null);
      }
      setProvidentFormMode(mode);
      setShowProvidentDialog(true);
    },
    [],
  );

  const closeProvidentDialog = () => {
    setShowProvidentDialog(false);
    setProvidentForm(createEmptyProvidentForm());
    setEditingProvidentRecord(null);
  };

  const handleProvidentFormChange = (key: keyof ProvidentFormState, value: string) => {
    setProvidentForm((prev) => ({ ...prev, [key]: value }));
  };

  const handleSaveProvidentRecord = async () => {
    const personalAccount = providentForm.personal_account.trim();
    const name = providentForm.name.trim();
    const identityNumber = providentForm.identity_number.trim().toUpperCase();
    if (!personalAccount) {
      toast.error("请填写个人账号");
      return;
    }
    if (!name) {
      toast.error("请填写姓名");
      return;
    }
    if (!identityNumber) {
      toast.error("请填写证件号码");
      return;
    }

    const personalBase = Number.parseFloat(providentForm.personal_base || "0");
    const personalAmount = Number.parseFloat(providentForm.personal_amount || "0");
    const companyAmount = Number.parseFloat(providentForm.company_amount || "0");
    const contributionRatio = Number.parseFloat(providentForm.contribution_ratio || DEFAULT_PROVIDENT_RATIO.toString());
    if ([personalBase, personalAmount, companyAmount].some((value) => Number.isNaN(value) || value < 0)) {
      toast.error("缴存基数和金额必须为非负数字");
      return;
    }
    if (!Number.isFinite(contributionRatio) || contributionRatio < 0) {
      toast.error("缴存比例必须为非负数字");
      return;
    }

    const payload = {
      personal_account: personalAccount,
      name,
      identity_number: identityNumber,
      personal_base: personalBase,
      personal_amount: personalAmount,
      company_amount: companyAmount,
      contribution_ratio: contributionRatio,
      notes: providentForm.notes.trim(),
    };

    setSavingProvidentRecord(true);
    try {
      if (providentFormMode === "edit" && editingProvidentRecord) {
        await updateProvidentRecord(editingProvidentRecord.id, payload);
        toast.success("公积金记录已更新");
      } else {
        await createProvidentRecord(payload);
        toast.success("已新增公积金记录");
      }
      closeProvidentDialog();
      await loadProvidentRecords();
    } catch (error) {
      console.error("[EmployeeManagement] save provident record failed", error);
      toast.error(error instanceof Error ? error.message : "保存公积金记录失败");
    } finally {
      setSavingProvidentRecord(false);
    }
  };

  const openSealDialog = (record: ProvidentFundRecord) => {
    setSealDialogRecord(record);
    setSealDate(formatDateToInput(new Date()));
    setSealSubmitting(false);
  };

  const openUnsealDialog = (record: ProvidentFundRecord) => {
    setUnsealDialogRecord(record);
    setUnsealDate(formatDateToInput(new Date()));
    setUnsealSubmitting(false);
  };

  const handleConfirmSeal = async () => {
    if (!sealDialogRecord) {
      return;
    }
    if (!sealDate) {
      toast.error("请选择封存日期");
      return;
    }
    setSealSubmitting(true);
    try {
      await sealProvidentRecord(sealDialogRecord.id, { date: sealDate });
      toast.success(`已封存 ${sealDialogRecord.name} 的公积金记录`);
      setSealDialogRecord(null);
      await loadProvidentRecords();
    } catch (error) {
      console.error("[EmployeeManagement] seal provident record failed", error);
      toast.error(error instanceof Error ? error.message : "封存失败，请稍后重试");
    } finally {
      setSealSubmitting(false);
    }
  };

  const handleConfirmUnseal = async () => {
    if (!unsealDialogRecord) {
      return;
    }
    if (!unsealDate) {
      toast.error("请选择启封日期");
      return;
    }
    setUnsealSubmitting(true);
    try {
      await unsealProvidentRecord(unsealDialogRecord.id, { date: unsealDate });
      toast.success(`已启封 ${unsealDialogRecord.name} 的公积金记录`);
      setUnsealDialogRecord(null);
      await loadProvidentRecords();
    } catch (error) {
      console.error("[EmployeeManagement] unseal provident record failed", error);
      toast.error(error instanceof Error ? error.message : "启封失败，请稍后重试");
    } finally {
      setUnsealSubmitting(false);
    }
  };

  const handleProvidentTemplateDownload = () => {
    const workbook = XLSX.utils.book_new();
    const sheetData = [PROVIDENT_TEMPLATE_HEADERS];
    const worksheet = XLSX.utils.aoa_to_sheet(sheetData);
    XLSX.utils.book_append_sheet(workbook, worksheet, PROVIDENT_TEMPLATE_SHEET_NAME);
    XLSX.writeFile(workbook, `公积金导入模板-${new Date().toISOString().slice(0, 10)}.xlsx`);
  };

  const handleProvidentFileChange = (event: ChangeEvent<HTMLInputElement>) => {
    const file = event.target.files?.[0] ?? null;
    setSelectedProvidentFile(file);
    setProvidentImportError("");
    setImportedProvidentFileName(file?.name ?? "");
  };

  const handleExecuteProvidentImport = async () => {
    if (!selectedProvidentFile) {
      setProvidentImportError("请选择需要导入的模板文件");
      return;
    }
    setProvidentImporting(true);
    try {
      const buffer = await selectedProvidentFile.arrayBuffer();
      const workbook = XLSX.read(buffer, { type: "array", cellDates: true });
      const sheetName = workbook.SheetNames[0];
      if (!sheetName) {
        throw new Error("模板中未找到工作表");
      }
      const worksheet = workbook.Sheets[sheetName];
      const rows = XLSX.utils.sheet_to_json<Record<string, unknown>>(worksheet, { defval: "" });
      if (rows.length === 0) {
        throw new Error("导入文件中没有可用数据");
      }
      let success = 0;
      const failed: Array<{ name: string; reason: string }> = [];
      for (const row of rows) {
        const account = String(row["个人账号"] ?? "").trim();
        const name = String(row["姓名"] ?? "").trim();
        const identity = String(row["证件号码"] ?? "").trim();
        if (!account || !identity) {
          continue;
        }
        const personalBase = Number.parseFloat(String(row["个人缴存基数"] ?? "0"));
        const ratioCell = row["缴存比例（%）"] ?? row["缴存比例(%)"] ?? row["缴存比例"] ?? DEFAULT_PROVIDENT_RATIO;
        const contributionRatio = Number.parseFloat(String(ratioCell ?? DEFAULT_PROVIDENT_RATIO));
        const personalAmount = Number.parseFloat(String(row["月应缴额（个人）"] ?? "0"));
        const companyAmount = Number.parseFloat(String(row["月应缴额（单位）"] ?? "0"));
        try {
          await createProvidentRecord({
            personal_account: account,
            name,
            identity_number: identity,
            personal_base: Number.isFinite(personalBase) ? personalBase : 0,
            personal_amount: Number.isFinite(personalAmount) ? personalAmount : 0,
            company_amount: Number.isFinite(companyAmount) ? companyAmount : 0,
            contribution_ratio: Number.isFinite(contributionRatio) ? contributionRatio : DEFAULT_PROVIDENT_RATIO,
            notes: String(row["备注"] ?? ""),
          });
          success += 1;
        } catch (error) {
          failed.push({
            name: name || identity,
            reason: error instanceof Error ? error.message : "未知错误",
          });
        }
      }
      await loadProvidentRecords();
      if (success > 0) {
        toast.success(`成功导入 ${success} 条公积金记录`);
      }
      if (failed.length > 0) {
        toast.error(`有 ${failed.length} 条记录导入失败，请检查日志`);
        console.warn("[EmployeeManagement] provident import failures", failed);
      }
      setSelectedProvidentFile(null);
      setImportedProvidentFileName("");
      setProvidentImportDialogOpen(false);
      if (providentImportInputRef.current) {
        providentImportInputRef.current.value = "";
      }
    } catch (error) {
      console.error("[EmployeeManagement] provident import failed", error);
      toast.error(error instanceof Error ? error.message : "导入失败，请检查模板格式");
    } finally {
      setProvidentImporting(false);
    }
  };

  const handleProvidentExport = (scope: "selected" | "all") => {
    const dataset = scope === "selected" ? selectedProvidentRecords : filteredProvidentRecords;
    if (dataset.length === 0) {
      toast.error("没有可导出的公积金数据");
      return;
    }
    const rows = dataset.map((record) => [
      record.personal_account,
      record.name,
      record.identity_number,
      record.personal_base ?? 0,
      record.contribution_ratio ?? DEFAULT_PROVIDENT_RATIO,
      record.personal_amount ?? 0,
      record.company_amount ?? 0,
      record.total_amount ?? 0,
      record.status === "sealed" ? "已封存" : "在缴",
      displayDate(record.sealed_at),
      record.notes ?? "",
    ]);
    const header = [
      "个人账号",
      "姓名",
      "证件号码",
      "缴存基数",
      "缴存比例（%）",
      "月应缴额（个人）",
      "月应缴额（单位）",
      "合计",
      "状态",
      "封存时间",
      "备注",
    ];
    const workbook = XLSX.utils.book_new();
    const worksheet = XLSX.utils.aoa_to_sheet([header, ...rows]);
    XLSX.utils.book_append_sheet(workbook, worksheet, "公积金");
    XLSX.writeFile(workbook, `公积金记录-${scope === "selected" ? "选中" : "全部"}-${new Date().toISOString().slice(0, 10)}.xlsx`);
  };

  const handleProvidentSettingsSave = async () => {
    const unitName = providentSettingsDraft.unit_name.trim() || DEFAULT_PROVIDENT_UNIT_NAME;
    const unitAccount = providentSettingsDraft.unit_account.trim() || DEFAULT_PROVIDENT_UNIT_ACCOUNT;
    try {
      const saved = await updateProvidentSettings({ unit_name: unitName, unit_account: unitAccount, month: "" });
      setProvidentSettings(saved);
      setProvidentSettingsDraft({ unit_name: saved.unit_name, unit_account: saved.unit_account });
      setShowProvidentSettingsDialog(false);
      toast.success("公积金单位设置已保存");
    } catch (error) {
      console.error("[EmployeeManagement] update provident settings failed", error);
      toast.error(error instanceof Error ? error.message : "保存设置失败");
    }
  };

  const generateBillInternal = async (targetMonth: string, overwrite = false) => {
    setBillGenerating(true);
    try {
      const bill = await generateProvidentBill({ month: targetMonth, overwrite });
      toast.success(`已生成 ${bill.month_label} 公积金账单`);
      await loadProvidentBills();
      setPendingBillMonth("");
      setBillOverwriteTarget(null);
      setShowBillOverwriteDialog(false);
    } catch (error) {
      console.error("[EmployeeManagement] generate provident bill failed", error);
      const message = error instanceof Error ? error.message : "生成账单失败";
      if (message.includes("no active records")) {
        toast.error("当前没有处于在缴状态的公积金记录，无法生成账单");
      } else {
        toast.error(message);
      }
    } finally {
      setBillGenerating(false);
    }
  };

  const proceedBillGeneration = (targetMonth: string) => {
    const duplicatedBill = providentBills.find((bill) => bill.month_label === targetMonth);
    if (duplicatedBill) {
      setBillOverwriteTarget(duplicatedBill);
      setPendingBillMonth(targetMonth);
      setShowBillOverwriteDialog(true);
      return;
    }
    void generateBillInternal(targetMonth);
  };

  const handleGenerateProvidentBill = () => {
    if (!/^[0-9]{4}-[0-9]{2}$/.test(providentBillMonth)) {
      toast.error("请输入正确的汇缴月份（YYYY-MM）");
      return;
    }
    if (!providentRecords.some((record) => record.status === "active")) {
      toast.error("当前没有处于在缴状态的公积金记录，无法生成账单");
      return;
    }
    setPendingBillMonth(providentBillMonth);
    setShowBillPrecheckDialog(true);
  };

  const handleOpenBillDetail = async (bill: ProvidentFundBill) => {
    try {
      const detail = await fetchProvidentBillDetail(bill.id);
      setActiveBill(detail);
      setShowBillDetailDialog(true);
    } catch (error) {
      console.error("[EmployeeManagement] load bill detail failed", error);
      toast.error(error instanceof Error ? error.message : "加载账单详情失败");
    }
  };

  const handleConfirmOverwriteBill = () => {
    if (!pendingBillMonth) {
      handleCancelOverwriteBill();
      return;
    }
    void generateBillInternal(pendingBillMonth, true);
  };

  const handleCancelOverwriteBill = () => {
    setShowBillOverwriteDialog(false);
    setBillOverwriteTarget(null);
    setPendingBillMonth("");
  };

  const handleConfirmPrecheckBill = () => {
    if (!pendingBillMonth) {
      setShowBillPrecheckDialog(false);
      return;
    }
    setShowBillPrecheckDialog(false);
    proceedBillGeneration(pendingBillMonth);
  };

  const handleCancelPrecheckBill = () => {
    setShowBillPrecheckDialog(false);
    setPendingBillMonth("");
  };

  const handleCloseBillDetail = () => {
    setShowBillDetailDialog(false);
    setActiveBill(null);
  };

  const handleBillDeleteRequest = (bill: ProvidentFundBill) => {
    setBillDeleteTarget(bill);
  };

  const handleConfirmDeleteBill = async () => {
    if (!billDeleteTarget) {
      return;
    }
    try {
      await deleteProvidentBill(billDeleteTarget.id);
      toast.success(`已删除 ${billDeleteTarget.month_label} 公积金账单`);
      if (activeBill?.id === billDeleteTarget.id) {
        setActiveBill(null);
        setShowBillDetailDialog(false);
      }
      setBillDeleteTarget(null);
      await loadProvidentBills();
    } catch (error) {
      console.error("[EmployeeManagement] delete bill failed", error);
      toast.error(error instanceof Error ? error.message : "删除账单失败");
    }
  };

  const openUnitInfoDialog = useCallback(() => {
    setUnitInfoDraft({ ...unitInfo });
    setUnitInfoDialogOpen(true);
  }, [unitInfo]);

  const handleUnitInfoSave = useCallback(() => {
    const trimmed: UnitInfo = {
      socialCode: unitInfoDraft.socialCode.trim(),
      unitName: unitInfoDraft.unitName.trim(),
    };
    setUnitInfo(trimmed);
    if (typeof window !== "undefined") {
      window.localStorage.setItem(UNIT_INFO_STORAGE_KEY, JSON.stringify(trimmed));
    }
    setUnitInfoDialogOpen(false);
    toast.success("单位信息已保存");
  }, [unitInfoDraft]);

  const handleResignReasonToggle = (value: string, checked: boolean) => {
    setResignReasons((prev) => {
      const next = new Set(prev);
      if (checked) {
        next.add(value);
      } else {
        next.delete(value);
      }
      return Array.from(next);
    });
  };

  const buildPrintDataset = (): PrintDataset | null => {
    if (!printContext || printContext.rows.length === 0) {
      return null;
    }

    if (printContext.type === "active") {
      const rows = printContext.rows as Employee[];
      const columnDefs = visibleFieldConfigs.map((field) => ({
        header: field.label,
        getter: (emp: Employee) =>
          field.key === "insuranceStatus"
            ? getInsuranceStatusLabel(emp, "active")
            : formatFieldDisplayValue(emp, field.key),
      }));

      return {
        type: "active",
        columns: columnDefs.map((col) => col.header),
        rows: rows.map((row) => columnDefs.map((col) => {
          const value = col.getter(row);
          return value === undefined || value === null ? "" : String(value);
        })),
        defaultTitle: `在职员工打印清单（共 ${rows.length} 人）`,
      };
    }

    if (printContext.type === "resigned") {
      const rows = printContext.rows as Employee[];
      const columnDefs = resignedColumnsForRender.map((column) => {
        if (column.type === "base") {
          return {
            header: column.label,
            getter: (emp: Employee) =>
              (column.id as keyof Employee) === "insuranceStatus"
                ? getInsuranceStatusLabel(emp, "resigned")
                : formatFieldDisplayValue(emp, column.id as keyof Employee),
          };
        }
        if (column.id === "resignDate") {
          return {
            header: column.label,
            getter: (emp: Employee) => displayDate(emp.resignDate),
          };
        }
        if (column.id === "resignReasons") {
          return {
            header: column.label,
            getter: (emp: Employee) =>
              emp.resignReasons && emp.resignReasons.length > 0
                ? emp.resignReasons.map((reason) => getResignReasonLabel(reason)).join("，")
                : "-",
          };
        }
        return {
          header: column.label,
          getter: (emp: Employee) => emp.resignProofName || (emp.resignProofUrl ? "已上传" : "未上传"),
        };
      });

      return {
        type: "resigned",
        columns: columnDefs.map((col) => col.header),
        rows: rows.map((row) => columnDefs.map((col) => {
          const value = col.getter(row);
          return value === undefined || value === null ? "" : String(value);
        })),
        defaultTitle: `离职员工打印清单（共 ${rows.length} 人）`,
      };
    }

    if (printContext.type === "provident") {
      const rows = printContext.rows as ProvidentFundRecord[];
      const columnDefs = (providentColumnsForRender.length > 0 ? providentColumnsForRender : PROVIDENT_COLUMN_CONFIGS).map((column) => ({
        header: column.label,
        getter: (record: ProvidentFundRecord) => getProvidentPrintableValue(record, column.id),
      }));
      return {
        type: "provident",
        columns: columnDefs.map((col) => col.header),
        rows: rows.map((row) =>
          columnDefs.map((col) => {
            const value = col.getter(row);
            return value === undefined || value === null ? "" : String(value);
          }),
        ),
        defaultTitle: `公积金打印清单（共 ${rows.length} 条）`,
      };
    }

    const rows = printContext.rows as SocialInsuranceRecord[];
    const columnDefs = insuranceColumnsForRender.map((column) => ({
      header: column.label,
      getter: (row: SocialInsuranceRecord) => {
        const value = column.getValue ? column.getValue(row) : row[column.id as keyof SocialInsuranceRecord];
        if (value === undefined || value === null) {
          return "";
        }
        return typeof value === "string" ? value : String(value);
      },
    }));

    return {
      type: "insurance",
      columns: columnDefs.map((col) => col.header),
      rows: rows.map((row) => columnDefs.map((col) => col.getter(row) || "")),
      defaultTitle: `社保增减打印清单（共 ${rows.length} 条）`,
    };
  };

  const openPrintDialog = (type: "active" | "resigned" | "insurance" | "provident") => {
    let rows: Employee[] | SocialInsuranceRecord[] | ProvidentFundRecord[] = [];
    let suggestedTitle = "";
    if (type === "active") {
      rows = sortedEmployees.filter((emp) => selectedEmployeeIds.includes(emp.id));
      if (rows.length === 0) {
        toast.error("请先勾选需要打印的在职员工");
        return;
      }
      suggestedTitle = `在职员工打印清单（共 ${rows.length} 人）`;
      setPrintContext({ type, rows });
    } else if (type === "resigned") {
      rows = sortedResignedEmployees.filter((emp) => selectedResignedIds.includes(emp.id));
      if (rows.length === 0) {
        toast.error("请先勾选需要打印的离职员工");
        return;
      }
      suggestedTitle = `离职员工打印清单（共 ${rows.length} 人）`;
      setPrintContext({ type, rows });
    } else if (type === "provident") {
      rows = sortedProvidentRecords.filter((record) => selectedProvidentIds.includes(record.id));
      if (rows.length === 0) {
        toast.error("请先勾选需要打印的公积金记录");
        return;
      }
      suggestedTitle = `公积金打印清单（共 ${rows.length} 条）`;
      setPrintContext({ type, rows });
    } else {
      rows = sortedInsuranceChanges.filter((change) => selectedInsuranceChangeIds.includes(change.id));
      if (rows.length === 0) {
        toast.error("请先勾选需要打印的社保变动记录");
        return;
      }
      suggestedTitle = `社保增减打印清单（共 ${rows.length} 条）`;
      setPrintContext({ type, rows });
    }
    setPrintSuggestedTitle(suggestedTitle);
    setShowPrintDialog(true);
  };

  const handleClosePrintDialog = () => {
    setShowPrintDialog(false);
    setPrintContext(null);
    setPrintSuggestedTitle("");
  };

  const handleGeneratePrint = async () => {
    if (!printContext || printContext.rows.length === 0) {
      toast.error("暂无需要打印的数据");
      return;
    }

    if (typeof window === "undefined" || typeof document === "undefined") {
      toast.error("当前环境不支持打印预览");
      return;
    }

    const dataset = buildPrintDataset();
    if (!dataset) {
      toast.error("未找到可打印的数据");
      return;
    }

    const title = (printTitle.trim() || dataset.defaultTitle).trim();
    const watermark = (printWatermark.trim() || "内部资料 请勿外传").trim();

    const loadingToastId = toast.loading("正在生成打印预览，请稍候...");
    try {
      const blob = await createReportPdf({
        title,
        watermark,
        columns: dataset.columns,
        rows: dataset.rows,
        orientation: printOrientation,
      });
      const url = URL.createObjectURL(blob);
      const previewWindow = window.open(url);
      if (!previewWindow) {
        toast.error("浏览器阻止了打印预览窗口，请允许弹窗后重试");
        URL.revokeObjectURL(url);
      } else {
        previewWindow.onload = () => previewWindow.focus();
        const cleanup = () => URL.revokeObjectURL(url);
        previewWindow.addEventListener("beforeunload", cleanup, { once: true });
        setTimeout(cleanup, 60_000);
      }
      handleClosePrintDialog();
    } catch (error) {
      console.error("[EmployeeManagement] generate pdf failed", error);
      toast.error("生成打印预览失败，请稍后重试");
    } finally {
      toast.dismiss(loadingToastId);
    }
  };

  const handleDownloadResignProof = async (employee: Employee) => {
    const numericId = Number(employee.id);
    if (!Number.isFinite(numericId) || numericId <= 0) {
      toast.error("当前离职记录缺少有效ID，无法下载离职证明");
      return;
    }

    try {
      const { blob, filename } = await downloadResignProof(numericId, token ?? undefined);
      downloadBlob(blob, filename || employee.resignProofName || `离职证明-${employee.name}`);
    } catch (error) {
      console.error("[EmployeeManagement] download resign proof failed", error);
      toast.error(error instanceof Error ? error.message : "离职证明下载失败，请稍后重试");
    }
  };

  const resetInsuranceUploadState = useCallback(() => {
    setInsuranceUploadFile(null);
    setInsuranceUploadPreview([]);
    setInsuranceUploadError("");
    setInsuranceImporting(false);
    if (insuranceUploadInputRef.current) {
      insuranceUploadInputRef.current.value = "";
    }
  }, []);

  const closeInsuranceUploadDialog = useCallback(() => {
    setShowInsuranceUploadDialog(false);
    resetInsuranceUploadState();
  }, [resetInsuranceUploadState]);

  const openInsuranceUpload = useCallback((type: SocialInsuranceChangeType) => {
    setInsuranceUploadType(type);
    setShowInsuranceUploadDialog(true);
    setInsuranceUploadError("");
    setInsuranceUploadPreview([]);
    setInsuranceUploadFile(null);
    if (insuranceUploadInputRef.current) {
      insuranceUploadInputRef.current.value = "";
    }
  }, []);

  const handleInsuranceFileChange = useCallback(async (event: ChangeEvent<HTMLInputElement>) => {
    const file = event.target.files?.[0] ?? null;
    if (!file) {
      setInsuranceUploadFile(null);
      setInsuranceUploadPreview([]);
      setInsuranceUploadError("");
      return;
    }

    setInsuranceUploadError("");
    try {
      const parsed = await parseSocialInsuranceTemplate(file, insuranceUploadType);
      setInsuranceUploadFile(file);
      setInsuranceUploadPreview(parsed);
    } catch (error) {
      console.error("[EmployeeManagement] parse insurance template failed", error);
      setInsuranceUploadFile(null);
      setInsuranceUploadPreview([]);
      setInsuranceUploadError(error instanceof Error ? error.message : "解析模板失败，请确认文件格式是否正确");
    }
  }, [insuranceUploadType]);

  const handleConfirmInsuranceUpload = useCallback(async () => {
    if (!insuranceUploadFile) {
      setInsuranceUploadError("请先选择模板文件");
      return;
    }

    if (insuranceUploadPreview.length === 0) {
      setInsuranceUploadError("模板中未解析到有效数据");
      return;
    }

    setInsuranceImporting(true);
    try {
      const payload: SocialInsuranceImportPayload = {
        records: insuranceUploadPreview,
      };
      await importSocialInsuranceChanges(insuranceUploadType, insuranceUploadFile, payload, token ?? undefined);
      toast.success("社保变动记录导入成功");
      closeInsuranceUploadDialog();
      await loadSocialInsuranceData();
    } catch (error) {
      console.error("[EmployeeManagement] import social insurance failed", error);
      setInsuranceUploadError(error instanceof Error ? error.message : "社保变动导入失败，请稍后重试");
    } finally {
      setInsuranceImporting(false);
    }
  }, [insuranceUploadFile, insuranceUploadPreview, insuranceUploadType, token, closeInsuranceUploadDialog, loadSocialInsuranceData]);

  const resetInsuranceFormState = useCallback(() => {
    setInsuranceIncreaseForm(createEmptyIncreaseForm());
    setInsuranceDecreaseForm(createEmptyDecreaseForm());
    setInsuranceFormEmployee(null);
    setInsuranceFormMode("create");
    setEditingInsuranceRecord(null);
  }, []);

  const closeInsuranceFormDialog = useCallback(() => {
    setShowInsuranceFormDialog(false);
    setInsuranceFormError("");
    setInsuranceFormSubmitting(false);
    setDuplicateWarning(null);
    resetInsuranceFormState();
  }, [resetInsuranceFormState]);

  const hydrateIncreaseForm = useCallback((employee?: Employee | null, record?: SocialInsuranceRecord | null) => {
    const base = createEmptyIncreaseForm();
    const templateValues = record?.templateValues ?? {};
    const getTemplateValue = (key: string) => (templateValues[key] ?? "").trim();
    const sourceHireDate = getTemplateValue("firstWorkDate") || employee?.hireDate || "";
    const hireDate = normalizeDateInput(sourceHireDate);
    const today = formatDateToInput(new Date());
    const defaultDate = hireDate || today;
    const nextMonthFirst = getNextMonthFirstDay();
    const defaultEducation = socialOptions?.education_level.default || "71.初级中学";
    base.nationality = getTemplateValue("nationality") || employee?.ethnicity?.trim() || "";
    base.firstWorkDate = getTemplateValue("firstWorkDate") || hireDate;
    base.job = getTemplateValue("job") || employee?.position?.trim() || "";
    base.householdType = getTemplateValue("householdType") || employee?.householdType?.trim() || socialOptions?.household_type.default || "";
    base.personalIdentity = getTemplateValue("personalIdentity") || socialOptions?.personal_identity.default || "";
    base.phone = getTemplateValue("phone") || employee?.phone?.trim() || "";
    base.education = getTemplateValue("education") || employee?.education?.trim() || defaultEducation;
    base.baseSalary = getTemplateValue("baseSalary") || "";
    base.pensionStartDate = getTemplateValue("pensionStartDate") || defaultDate;
    base.unemploymentStartDate = getTemplateValue("unemploymentStartDate") || defaultDate;
    base.medicalStartDate = getTemplateValue("medicalStartDate") || defaultDate;
    base.injuryStartDate = getTemplateValue("injuryStartDate") || defaultDate;
    base.maternityStartDate = getTemplateValue("maternityStartDate") || nextMonthFirst;
    base.specialSkill = getTemplateValue("specialSkill") || socialOptions?.special_skill.default || "";
    base.skillLevel = getTemplateValue("skillLevel") || socialOptions?.skill_level.default || "";
    const fallbackPersonalNumber = employee?.socialInsuranceNumber?.trim() || employee?.employeeId || "";
    base.personalNumber = record?.personalNumber || fallbackPersonalNumber;
    base.remark = getTemplateValue("remark");
    setInsuranceIncreaseForm(base);
  }, [socialOptions]);

  const hydrateDecreaseForm = useCallback((record?: SocialInsuranceRecord | null, employee?: Employee | null) => {
    const base = createEmptyDecreaseForm();
    const templateValues = record?.templateValues ?? {};
    const getTemplateValue = (key: string) => (templateValues[key] ?? "").trim();
    const today = formatDateToInput(new Date());
    const flagDefault = socialOptions?.reduction_flag.default || "";
    const fallbackPersonalNumber = employee?.socialInsuranceNumber?.trim() || employee?.employeeId || "";
    base.personalNumber = record?.personalNumber || getTemplateValue("personalNumber") || fallbackPersonalNumber;
    base.decreaseDate = getTemplateValue("decreaseDate") || today;
    base.decreaseReason = getTemplateValue("decreaseReason") || socialOptions?.decrease_reason.default || "";
    base.pensionFlag = getTemplateValue("pensionFlag") || flagDefault;
    base.unemploymentFlag = getTemplateValue("unemploymentFlag") || flagDefault;
    base.medicalFlag = getTemplateValue("medicalFlag") || flagDefault;
    base.injuryFlag = getTemplateValue("injuryFlag") || flagDefault;
    base.maternityFlag = getTemplateValue("maternityFlag") || flagDefault;
    base.unemploymentReason = getTemplateValue("unemploymentReason") || socialOptions?.unemployment_reason.default || "";
    setInsuranceDecreaseForm(base);
  }, [socialOptions]);

  const openInsuranceForm = useCallback((
    type: SocialInsuranceChangeType,
    options?: { employee?: Employee; record?: SocialInsuranceRecord },
  ) => {
    if (!socialOptions) {
      toast.error("社保模板选项仍在加载，请稍后重试");
      if (!socialOptionsLoading) {
        loadSocialOptions();
      }
      return;
    }

    setInsuranceFormType(type);
    setInsuranceFormError("");

    if (options?.record) {
      setInsuranceFormMode("edit");
      setEditingInsuranceRecord(options.record);
      const matchedEmployee = matchEmployeeByIdNumber(options.record.identityNumber);
      const stub = matchedEmployee ? { ...matchedEmployee } : buildEmployeeStubFromRecord(options.record);
      setInsuranceFormEmployee(stub);
      if (type === "increase") {
        hydrateIncreaseForm(stub, options.record);
      } else {
        hydrateDecreaseForm(options.record, stub);
      }
    } else if (options?.employee) {
      setInsuranceFormMode("create");
      setEditingInsuranceRecord(null);
      setInsuranceFormEmployee(options.employee);
      if (type === "increase") {
        hydrateIncreaseForm(options.employee);
      } else {
        hydrateDecreaseForm(undefined, options.employee);
      }
    } else {
      toast.error("请选择需要操作的员工");
      return;
    }

    setShowInsuranceFormDialog(true);
  }, [hydrateDecreaseForm, hydrateIncreaseForm, loadSocialOptions, matchEmployeeByIdNumber, socialOptions, socialOptionsLoading]);

  const handleSubmitInsuranceForm = useCallback(async () => {
    if (!insuranceFormEmployee) {
      toast.error("请选择需要操作的员工");
      return;
    }
    if (!socialOptions) {
      toast.error("社保模板选项尚未加载完成");
      return;
    }

    const errors: string[] = [];
    if (insuranceFormType === "increase") {
      if (!insuranceIncreaseForm.personalIdentity.trim()) {
        errors.push("请选择个人身份");
      }
      if (!insuranceIncreaseForm.householdType.trim()) {
        errors.push("请选择户口性质");
      }
      if (!insuranceIncreaseForm.baseSalary.trim()) {
        errors.push("请填写月基本工资额");
      }
      const normalizedPensionDate = normalizeDateInput(insuranceIncreaseForm.pensionStartDate);
      if (!normalizedPensionDate) {
        errors.push("请填写有效的养老参保时间");
      }
    } else {
      if (!insuranceDecreaseForm.personalNumber.trim()) {
        errors.push("请填写个人编号");
      }
      const normalizedDecreaseDate = normalizeDateInput(insuranceDecreaseForm.decreaseDate);
      if (!normalizedDecreaseDate) {
        errors.push("请填写有效的减少时间");
      }
      if (!insuranceDecreaseForm.decreaseReason.trim()) {
        errors.push("请选择减少原因");
      }
    }

    if (errors.length > 0) {
      const message = errors.join("，");
      setInsuranceFormError(message);
      toast.error(message);
      return;
    }

    const normalizedEmployeeName = insuranceFormEmployee.name.trim();
    const duplicateRecord = socialInsuranceChanges.find((change) => {
      if (!normalizedEmployeeName) {
        return false;
      }
      if (change.changeType !== insuranceFormType) {
        return false;
      }
      if (insuranceFormMode === "edit" && editingInsuranceRecord && change.id === editingInsuranceRecord.id) {
        return false;
      }
      return change.employeeName.trim() === normalizedEmployeeName;
    });
    if (duplicateRecord) {
      setDuplicateWarning({ type: insuranceFormType, name: normalizedEmployeeName });
      return;
    }

    const templateValues: Record<string, string> = {
      idNumber: insuranceFormEmployee.idNumber,
      name: insuranceFormEmployee.name,
    };

    let effectiveDate = "";
    let reason = "";
    let personalNumber = "";

    if (insuranceFormType === "increase") {
      const normalizedPensionDate = normalizeDateInput(insuranceIncreaseForm.pensionStartDate);
      templateValues.nationality = insuranceIncreaseForm.nationality.trim();
      templateValues.firstWorkDate = normalizeDateInput(insuranceIncreaseForm.firstWorkDate);
      templateValues.baseSalary = insuranceIncreaseForm.baseSalary.trim();
      templateValues.job = insuranceIncreaseForm.job.trim();
      templateValues.personalIdentity = insuranceIncreaseForm.personalIdentity.trim();
      templateValues.householdType = insuranceIncreaseForm.householdType.trim();
      templateValues.phone = insuranceIncreaseForm.phone.trim();
      templateValues.education = insuranceIncreaseForm.education.trim();
      templateValues.pensionStartDate = normalizedPensionDate;
      templateValues.unemploymentStartDate = normalizeDateInput(insuranceIncreaseForm.unemploymentStartDate);
      templateValues.medicalStartDate = normalizeDateInput(insuranceIncreaseForm.medicalStartDate);
      templateValues.injuryStartDate = normalizeDateInput(insuranceIncreaseForm.injuryStartDate);
      templateValues.maternityStartDate = normalizeDateInput(insuranceIncreaseForm.maternityStartDate);
      templateValues.remark = insuranceIncreaseForm.remark.trim();
      templateValues.specialSkill = insuranceIncreaseForm.specialSkill.trim();
      templateValues.skillLevel = insuranceIncreaseForm.skillLevel.trim();
      templateValues.personalNumber = insuranceIncreaseForm.personalNumber.trim();
      effectiveDate = normalizedPensionDate;
      reason = insuranceIncreaseForm.remark.trim();
      personalNumber = insuranceIncreaseForm.personalNumber.trim();
    } else {
      const normalizedDecreaseDate = normalizeDateInput(insuranceDecreaseForm.decreaseDate);
      templateValues.personalNumber = insuranceDecreaseForm.personalNumber.trim();
      templateValues.decreaseDate = normalizedDecreaseDate;
      templateValues.decreaseReason = insuranceDecreaseForm.decreaseReason.trim();
      templateValues.pensionFlag = insuranceDecreaseForm.pensionFlag.trim();
      templateValues.unemploymentFlag = insuranceDecreaseForm.unemploymentFlag.trim();
      templateValues.medicalFlag = insuranceDecreaseForm.medicalFlag.trim();
      templateValues.injuryFlag = insuranceDecreaseForm.injuryFlag.trim();
      templateValues.maternityFlag = insuranceDecreaseForm.maternityFlag.trim();
      templateValues.unemploymentReason = insuranceDecreaseForm.unemploymentReason.trim();
      effectiveDate = normalizedDecreaseDate;
      reason = insuranceDecreaseForm.decreaseReason.trim();
      personalNumber = insuranceDecreaseForm.personalNumber.trim();
    }

    const requestPayload: SocialInsuranceManualPayload = {
      change_type: insuranceFormType,
      employee_name: insuranceFormEmployee.name,
      department: insuranceFormEmployee.department,
      identity_number: insuranceFormEmployee.idNumber,
      personal_number: personalNumber,
      effective_date: effectiveDate,
      reason,
      template_values: templateValues,
    };

    setInsuranceFormSubmitting(true);
    setInsuranceFormError("");
    try {
      if (insuranceFormMode === "edit" && editingInsuranceRecord) {
        await updateSocialInsuranceChange(editingInsuranceRecord.numericId, requestPayload, token ?? undefined);
        toast.success("社保变动记录已更新");
      } else {
        await createSocialInsuranceChange(requestPayload, token ?? undefined);
        toast.success(insuranceFormType === "increase" ? "社保增加记录创建成功" : "社保减少记录创建成功");
      }
      setInsuranceView(insuranceFormType);
      closeInsuranceFormDialog();
      await loadSocialInsuranceData();
    } catch (error) {
      console.error("[EmployeeManagement] save social insurance change failed", error);
      const message = error instanceof Error ? error.message : "保存社保变更失败，请稍后重试";
      setInsuranceFormError(message);
      toast.error(message);
    } finally {
      setInsuranceFormSubmitting(false);
    }
  }, [
    closeInsuranceFormDialog,
    editingInsuranceRecord,
    insuranceDecreaseForm,
    insuranceFormEmployee,
    insuranceFormMode,
    insuranceFormType,
    insuranceIncreaseForm,
    loadSocialInsuranceData,
    socialInsuranceChanges,
    socialOptions,
    token,
  ]);

  const renderInsuranceManagementCard = (view: Extract<InsuranceView, "increase" | "decrease">) => {
    const isIncrease = view === "increase";
    const viewTitle = isIncrease ? "社保增加明细" : "社保减少明细";
    const viewDescription = isIncrease ? "记录员工社保增加业务明细" : "记录员工社保减少业务明细";
    const viewTotalCount = insuranceCounts[view];
    const filteredCount = filteredInsuranceChanges.length;
    const selectedChangesInView = sortedInsuranceChanges.filter((change) => selectedInsuranceChangeIds.includes(change.id));
    const selectedCount = selectedChangesInView.length;
    const hasSelection = selectedCount > 0;
    const allVisibleSelected =
      sortedInsuranceChanges.length > 0 &&
      sortedInsuranceChanges.every((change) => selectedInsuranceChangeIds.includes(change.id));
    const exportAllLabel = isIncrease ? "导出全部增加" : "导出全部减少";
    const isFilterActive =
      filteredCount !== viewTotalCount ||
      insuranceSearch.trim().length > 0 ||
      insuranceDepartmentFilter !== "all" ||
      insuranceReasonFilter !== "all";

    const viewKey: InsuranceViewKey = isIncrease ? "increase" : "decrease";
    const visibleForView = insuranceVisibleColumns[viewKey] ?? DEFAULT_INSURANCE_VISIBLE_MAP[viewKey];
    const visibleSetForView = new Set(visibleForView);
    const columnOptions = [
      ...BASE_INSURANCE_COLUMNS.map((column) => ({
        id: column.id as InsuranceColumnId,
        label: column.label,
        required: REQUIRED_INSURANCE_COLUMNS.includes(column.id),
      })),
      ...TEMPLATE_ENTRIES_BY_VIEW[viewKey].map((entry) => ({
        id: entry.id as InsuranceColumnId,
        label: entry.field.label,
        required: REQUIRED_INSURANCE_COLUMNS.includes(entry.id as InsuranceColumnId),
      })),
    ];
    const isFieldSelectorOpen = insuranceFieldDialogFor === viewKey;

    return (
      <Card>
        <CardHeader>
          <div className="flex flex-wrap items-center justify-between gap-4">
            <div>
              <CardTitle>{viewTitle}</CardTitle>
              <CardDescription>{viewDescription}</CardDescription>
              <div className="mt-2 text-xs text-muted-foreground leading-relaxed">
                <div>单位社保编号：{unitInfo.socialCode || "未配置"}</div>
                <div>单位名称：{unitInfo.unitName || "未配置"}</div>
              </div>
            </div>
            <div className="flex flex-wrap items-center gap-3 text-sm">
              <span className="text-sm text-muted-foreground">
                {loadingInsurance ? "记录加载中..." : `记录总数 ${viewTotalCount}`}
              </span>
              {isFilterActive && (
                <Badge variant="outline">筛选 {filteredCount}</Badge>
              )}
              {hasSelection && <Badge variant="default">已选择 {selectedCount}</Badge>}
              <div className="flex flex-wrap items-center gap-2">
                {hasSelection ? (
                  <>
                    <DropdownMenu>
                      {/* P7.1：社保记录导出需 insurance.view 权限 */}
                      <RequirePermission resource="insurance" action="view">
                        <DropdownMenuTrigger asChild>
                          <Button
                            variant="outline"
                            size="sm"
                            disabled={insuranceExporting || viewTotalCount === 0}
                          >
                            <Download className="h-4 w-4 mr-2" />
                            导出
                          </Button>
                        </DropdownMenuTrigger>
                      </RequirePermission>
                      <DropdownMenuContent align="end" className="w-44">
                        <DropdownMenuItem
                          disabled={insuranceExporting}
                          onClick={() => handleExportInsurance("selected")}
                        >
                          导出选中数据
                        </DropdownMenuItem>
                        <DropdownMenuItem
                          disabled={filteredCount === 0 || insuranceExporting}
                          onClick={() => handleExportInsurance("filtered")}
                        >
                          导出当前筛选
                        </DropdownMenuItem>
                        <DropdownMenuSeparator />
                        <DropdownMenuItem
                          disabled={viewTotalCount === 0 || insuranceExporting}
                          onClick={() => handleExportInsurance("all")}
                        >
                          {exportAllLabel}
                        </DropdownMenuItem>
                      </DropdownMenuContent>
                    </DropdownMenu>
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={() => openPrintDialog("insurance")}
                    >
                      <Printer className="h-4 w-4 mr-2" /> 打印
                    </Button>
                    {/* P7.1：批量撤销社保记录为删除操作，需 insurance.delete 权限 */}
                    <RequirePermission resource="insurance" action="delete">
                      <Button
                        variant="destructive"
                        size="sm"
                        onClick={() => setShowBatchInsuranceConfirm(true)}
                      >
                        <Trash2 className="h-4 w-4 mr-2" /> 撤销
                      </Button>
                    </RequirePermission>
                  </>
                ) : (
                  /* P7.1：导入社保变动记录需 insurance.create 权限 */
                  <RequirePermission resource="insurance" action="create">
                    <Button
                      variant="outline"
                      size="sm"
                      disabled={insuranceImporting}
                      onClick={() => openInsuranceUpload(view)}
                    >
                      <Upload className="h-4 w-4 mr-2" />
                      导入
                    </Button>
                  </RequirePermission>
                )}
                {/* P7.1：单位信息配置需 insurance.edit 权限 */}
                <RequirePermission resource="insurance" action="edit">
                  <Button variant="ghost" size="sm" onClick={openUnitInfoDialog}>
                    <Settings className="mr-2 h-4 w-4" />
                    单位信息
                  </Button>
                </RequirePermission>
              </div>
            </div>
          </div>
        </CardHeader>
        <CardContent>
          <div className="mb-4 flex flex-wrap items-center gap-4">
            <div className="relative flex-1 min-w-[220px]">
              <Search className="absolute left-3 top-3 h-4 w-4 text-muted-foreground" />
              <Input
                placeholder={isIncrease ? "搜索增加记录：姓名、证件号、原因..." : "搜索减少记录：姓名、证件号、原因..."}
                value={insuranceSearch}
                onChange={(event) => setInsuranceSearch(event.target.value)}
                className="pl-10 pr-10"
              />
              {insuranceSearch && (
                <button
                  type="button"
                  onClick={() => setInsuranceSearch("")}
                  className="absolute right-2 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground"
                  aria-label="清除搜索"
                >
                  <X className="h-4 w-4" />
                </button>
              )}
            </div>
            {insuranceDepartmentOptions.length > 0 && (
              <Select value={insuranceDepartmentFilter} onValueChange={setInsuranceDepartmentFilter}>
                <SelectTrigger className="w-40">
                  <SelectValue placeholder="选择部门" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="all">全部部门</SelectItem>
                  {insuranceDepartmentOptions.map((dept) => (
                    <SelectItem key={dept} value={dept}>
                      {dept}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            )}
            {insuranceReasonOptions.length > 0 && (
              <Select value={insuranceReasonFilter} onValueChange={setInsuranceReasonFilter}>
                <SelectTrigger className="w-44">
                  <SelectValue placeholder="选择变动原因" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="all">全部原因</SelectItem>
                  {insuranceReasonOptions.map((reason) => (
                    <SelectItem key={reason} value={reason}>
                      {reason}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            )}
            <Dialog open={isFieldSelectorOpen} onOpenChange={(open) => setInsuranceFieldDialogFor(open ? viewKey : null)}>
              <DialogTrigger asChild>
                <Button variant="outline" size="sm">
                  <Settings className="h-4 w-4 mr-2" />
                  显示字段
                </Button>
              </DialogTrigger>
              <DialogContent className="max-w-md">
                <DialogHeader>
                  <DialogTitle>自定义显示字段</DialogTitle>
                  <DialogDescription>选择要在表格中显示的字段</DialogDescription>
                </DialogHeader>
                <div className="max-h-[60vh] space-y-3 overflow-y-auto py-2">
                  {columnOptions.map((option) => (
                    <label key={`${viewKey}-${option.id}`} className="flex items-center justify-between rounded-md border px-3 py-2 text-sm">
                      <div className="flex items-center gap-2">
                        <input
                          type="checkbox"
                          className="h-4 w-4 rounded border-muted-foreground"
                          checked={visibleSetForView.has(option.id)}
                          disabled={option.required}
                          onChange={(event) => handleInsuranceFieldToggle(viewKey, option.id, event.target.checked)}
                        />
                        <span>{option.label}</span>
                      </div>
                      {option.required && <Badge variant="outline">必选</Badge>}
                    </label>
                  ))}
                </div>
                <DialogFooter className="flex flex-col gap-2 sm:flex-row sm:justify-end">
                  <Button variant="outline" onClick={() => resetInsuranceFieldsToDefault(viewKey)}>
                    恢复默认
                  </Button>
                  <Button onClick={() => setInsuranceFieldDialogFor(null)}>完成</Button>
                </DialogFooter>
              </DialogContent>
            </Dialog>
          </div>
          <DataTableWrapper height="h-[60vh]">
              <Table className="min-w-full table-auto text-sm">
                <TableHeader>
                  <TableRow className="text-muted-foreground">
                    <TableHead className={cn(ALIGNMENT_CLASS.center, "w-12")}>
                      <input
                        type="checkbox"
                        className="h-4 w-4 rounded border-muted-foreground"
                        checked={sortedInsuranceChanges.length > 0 && allVisibleSelected}
                        onChange={(event) => toggleAllInsuranceChanges(event.target.checked)}
                        aria-label="选择全部社保记录"
                      />
                    </TableHead>
                    {insuranceColumnsForRender.map((column) => {
                      const headClass = cn("cursor-move select-none whitespace-nowrap", ALIGNMENT_CLASS.left);
                      return (
                        <TableHead
                          key={column.id}
                          className={headClass}
                          draggable
                          onDragStart={(event) => handleColumnDragStart(event, column.id)}
                          onDragOver={handleColumnDragOver}
                          onDrop={(event) => handleInsuranceColumnDrop(event, column.id as InsuranceColumnId)}
                          onDragEnd={handleColumnDragEnd}
                          onClick={column.sortable === false ? undefined : () => handleInsuranceSortClick(column.id as InsuranceColumnId)}
                        >
                          <span className="flex items-center gap-1">
                            {column.label}
                            {column.sortable !== false && renderSortIndicator(insuranceSort, column.id as InsuranceColumnId)}
                          </span>
                        </TableHead>
                      );
                    })}
                    <TableHead className={cn("w-16", ALIGNMENT_CLASS.center)}>操作</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {sortedInsuranceChanges.length === 0 ? (
                    <TableRow>
                      <TableCell colSpan={insuranceColumnsForRender.length + 2} className="py-8 text-center text-muted-foreground">
                        {isIncrease ? "暂无社保增加记录" : "暂无社保减少记录"}
                      </TableCell>
                    </TableRow>
                  ) : (
                    sortedInsuranceChanges.map((change) => (
                      <TableRow
                        key={change.id}
                        className="hover:bg-muted/60"
                        onDoubleClick={() => openInsuranceForm(change.changeType, { record: change })}
                      >
                        <TableCell className={ALIGNMENT_CLASS.center}>
                          <input
                            type="checkbox"
                            className="h-4 w-4 rounded border-muted-foreground"
                            checked={selectedInsuranceChangeIds.includes(change.id)}
                            onChange={(event) => toggleInsuranceChangeSelection(change.id, event.target.checked)}
                          />
                        </TableCell>
                        {insuranceColumnsForRender.map((column) => {
                          const value = column.getValue ? column.getValue(change) : "";
                          if (column.id === "changeType") {
                            return (
                              <TableCell key={column.id} className={ALIGNMENT_CLASS.left}>
                                <Badge variant={change.changeType === "increase" ? "default" : "destructive"}>
                                  {change.changeType === "increase" ? "增加" : "减少"}
                                </Badge>
                              </TableCell>
                            );
                          }
                          if (column.id === "effectiveDate" || column.id === "createdAt") {
                            return (
                              <TableCell key={column.id} className={ALIGNMENT_CLASS.left}>
                                {displayDate(String(value ?? ""))}
                              </TableCell>
                            );
                          }
                          const text = value === null || value === undefined ? "" : String(value);
                          return (
                            <TableCell key={column.id} className={cn("text-sm", ALIGNMENT_CLASS.left, { "font-medium": column.id === "employeeName" })}>
                              {text || "-"}
                            </TableCell>
                          );
                        })}
                        <TableCell className={cn(ALIGNMENT_CLASS.center, "w-16")}>
                          <DropdownMenu>
                            <DropdownMenuTrigger asChild>
                              <Button variant="ghost" size="icon" className="h-8 w-8" aria-label="社保操作">
                                <MoreHorizontal className="h-4 w-4" />
                              </Button>
                            </DropdownMenuTrigger>
                            <DropdownMenuContent align="end">
                              <DropdownMenuItem
                                onClick={(event) => {
                                  event.stopPropagation();
                                  openInsuranceForm(change.changeType, { record: change });
                                }}
                                className="gap-2"
                              >
                                <Eye className="h-4 w-4" /> 查看详情
                              </DropdownMenuItem>
                              {/* P7.1：行内撤销社保记录需 insurance.delete 权限 */}
                              <RequirePermission resource="insurance" action="delete">
                                <DropdownMenuItem
                                  onClick={(event) => {
                                    event.stopPropagation();
                                    setSelectedInsuranceChangeIds([change.id]);
                                    setShowBatchInsuranceConfirm(true);
                                  }}
                                  className="gap-2 text-destructive"
                                >
                                  <Trash2 className="h-4 w-4" /> 撤销记录
                                </DropdownMenuItem>
                              </RequirePermission>
                            </DropdownMenuContent>
                          </DropdownMenu>
                        </TableCell>
                      </TableRow>
                    ))
                  )}
                </TableBody>
              </Table>
              <ScrollBar orientation="horizontal" />
              </DataTableWrapper>
        </CardContent>
      </Card>
    );
  };

  const renderProvidentManagementCard = () => {
    const recordCount = providentRecords.length;
    const filteredCount = filteredProvidentRecords.length;
    const handleBillCardKeyDown = (event: KeyboardEvent<HTMLDivElement>, bill: ProvidentFundBill) => {
      if (event.key === "Enter" || event.key === " ") {
        event.preventDefault();
        void handleOpenBillDetail(bill);
      }
    };

    const ProvidentBillCard = ({ bill }: { bill: ProvidentFundBill }) => (
      <div
        role="button"
        tabIndex={0}
        aria-label={`账期 ${bill.month_label}，双击查看详情`}
        title="双击查看账单详情"
        className="group flex w-full flex-wrap items-center justify-between gap-4 rounded-xl border px-4 py-3 text-sm transition duration-200 hover:-translate-y-0.5 hover:border-primary/40 hover:bg-primary/5 hover:shadow-lg focus:outline-none focus-visible:ring-2 focus-visible:ring-primary"
        onDoubleClick={() => {
          void handleOpenBillDetail(bill);
        }}
        onKeyDown={(event) => handleBillCardKeyDown(event, bill)}
      >
        <div className="space-y-1">
          <div className="flex items-center gap-2 text-base font-semibold text-foreground">
            <PiggyBank className="h-5 w-5 text-primary" />
            <span>{bill.month_label}</span>
          </div>
          <p className="text-xs text-muted-foreground">双击或按 Enter 查看详情，右侧图标可删除当前账单。</p>
        </div>
                <div className="flex items-center gap-2">
                  <span className="hidden text-[11px] text-muted-foreground sm:inline">双击查看</span>
                  {/* P7.1：删除账单需 insurance.delete 权限（行内按钮，隐藏破坏布局用 disable） */}
                  <RequirePermission resource="insurance" action="delete" mode="disable">
                    <Button
                      variant="ghost"
                      size="icon"
                      className="text-destructive hover:text-destructive"
                      onClick={(event) => {
                        event.stopPropagation();
                        handleBillDeleteRequest(bill);
                      }}
                    >
                      <span className="sr-only">删除账单</span>
                      <Trash2 className="h-4 w-4" />
                    </Button>
                  </RequirePermission>
                </div>
      </div>
    );

    return (
      <>
        <Card className="">
          <CardHeader className="space-y-4">
            <div className="flex flex-wrap items-start justify-between gap-4">
              <div>
                <CardTitle>公积金账单</CardTitle>
                <CardDescription>快速生成并查看账期，双击卡片即可展开详情。</CardDescription>
              </div>
              <div className="flex flex-wrap items-center gap-3">
                <div className="flex items-center gap-2 text-sm text-muted-foreground">
                  <Label className="text-sm">汇缴月份</Label>
                  <Input type="month" value={providentBillMonth} onChange={(event) => setProvidentBillMonth(event.target.value)} className="w-36" />
                </div>
                {/* P7.1：生成公积金账单为创建操作，需 insurance.create 权限 */}
                <RequirePermission resource="insurance" action="create">
                  <Button onClick={handleGenerateProvidentBill} disabled={billGenerating || providentLoading}>
                    {billGenerating ? <Loader2 className="mr-2 h-4 w-4 animate-spin" /> : <FileSpreadsheet className="mr-2 h-4 w-4" />}
                    生成账单
                  </Button>
                </RequirePermission>
              </div>
            </div>
            {providentBills.length > 0 && (
              <div className="flex flex-wrap items-center gap-2 text-sm text-muted-foreground">
                <Label className="text-sm">选择账期</Label>
                <Select value={selectedBill ? String(selectedBill.id) : ""} onValueChange={(value) => setSelectedBillId(Number(value))}>
                  <SelectTrigger className="w-48">
                    <SelectValue placeholder="请选择账单" />
                  </SelectTrigger>
                  <SelectContent className="max-h-64">
                    {providentBills.map((bill) => (
                      <SelectItem key={bill.id} value={String(bill.id)}>
                        {bill.month_label}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
                {!selectedBill && <span className="text-xs">请选择账单以查看详情</span>}
              </div>
            )}
          </CardHeader>
          <CardContent>
            {providentBillsLoading ? (
              <div className="py-6 text-center text-sm text-muted-foreground">账单加载中...</div>
            ) : providentBills.length === 0 ? (
              <div className="py-6 text-center text-sm text-muted-foreground">暂无历史账单，生成后将展示在此。</div>
            ) : selectedBill ? (
              <ProvidentBillCard bill={selectedBill} />
            ) : null}
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <div className="flex flex-wrap items-center justify-between gap-4">
              <div>
                <CardTitle>公积金明细</CardTitle>
                <CardDescription>支持按状态筛选、批量导入、下载与打印报表。</CardDescription>
                <div className="mt-2 text-xs text-muted-foreground leading-relaxed space-y-1">
                  <div>记录总数：{recordCount} 条，当前筛选：{filteredCount} 条</div>
                  <div>
                    单位：{(providentSettings?.unit_name || providentSettingsDraft.unit_name) ?? DEFAULT_PROVIDENT_UNIT_NAME}（账号：
                    {providentSettings?.unit_account || providentSettingsDraft.unit_account || DEFAULT_PROVIDENT_UNIT_ACCOUNT}）
                  </div>
                </div>
              </div>
              <div className="flex flex-wrap items-center gap-2 text-sm">
                {hasProvidentSelection ? (
                  <>
                    <DropdownMenu>
                      {/* P7.1：公积金导出需 insurance.view 权限 */}
                      <RequirePermission resource="insurance" action="view">
                        <DropdownMenuTrigger asChild>
                          <Button variant="outline" size="sm">
                            <Download className="mr-2 h-4 w-4" /> 导出
                          </Button>
                        </DropdownMenuTrigger>
                      </RequirePermission>
                      <DropdownMenuContent align="end" className="w-40">
                        <DropdownMenuItem onClick={() => handleProvidentExport("selected")}>
                          导出选中
                        </DropdownMenuItem>
                        <DropdownMenuItem onClick={() => handleProvidentExport("all")}>
                          导出全部
                        </DropdownMenuItem>
                      </DropdownMenuContent>
                    </DropdownMenu>
                    <Button variant="outline" size="sm" onClick={() => openPrintDialog("provident")}>
                      <Printer className="mr-2 h-4 w-4" /> 打印
                    </Button>
                  </>
                ) : (
                  <>
                    {/* P7.1：新增公积金记录需 insurance.create 权限 */}
                    <RequirePermission resource="insurance" action="create">
                      <Button size="sm" onClick={() => openProvidentDialog("create")}>
                        <Plus className="mr-2 h-4 w-4" /> 新增记录
                      </Button>
                    </RequirePermission>
                    {/* P7.1：导入公积金记录需 insurance.create 权限 */}
                    <RequirePermission resource="insurance" action="create">
                      <Button
                        variant="outline"
                        size="sm"
                        onClick={() => {
                          setProvidentImportDialogOpen(true);
                          if (providentImportInputRef.current) {
                            providentImportInputRef.current.value = "";
                          }
                        }}
                      >
                        <Upload className="mr-2 h-4 w-4" /> 导入
                      </Button>
                    </RequirePermission>
                  </>
                )}
                {hasProvidentSelection && (
                  <Badge variant="secondary">已选 {selectedProvidentIds.length}</Badge>
                )}
                {/* P7.1：公积金单位设置需 insurance.edit 权限 */}
                <RequirePermission resource="insurance" action="edit">
                  <Button variant="ghost" size="sm" onClick={() => setShowProvidentSettingsDialog(true)}>
                    <Settings className="mr-2 h-4 w-4" /> 单位设置
                  </Button>
                </RequirePermission>
              </div>
            </div>
          </CardHeader>
          <CardContent>
            <div className="flex flex-wrap items-center gap-4 mb-4">
              <div className="relative flex-1 min-w-[220px]">
                <Search className="absolute left-3 top-3 h-4 w-4 text-muted-foreground" />
                <Input
                  placeholder="搜索个人账号、姓名或证件号码"
                  value={providentSearch}
                  onChange={(event) => setProvidentSearch(event.target.value)}
                  className="pl-9 pr-10"
                />
                {providentSearch && (
                  <button
                    type="button"
                    onClick={() => setProvidentSearch("")}
                    className="absolute right-2 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground"
                    aria-label="清除搜索"
                  >
                    <X className="h-4 w-4" />
                  </button>
                )}
              </div>
              <Select value={providentStatusFilter} onValueChange={(value) => setProvidentStatusFilter(value as typeof providentStatusFilter)}>
                <SelectTrigger className="w-36">
                  <SelectValue placeholder="选择状态" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="all">全部状态</SelectItem>
                  <SelectItem value="active">在缴</SelectItem>
                  <SelectItem value="sealed">已封存</SelectItem>
                </SelectContent>
              </Select>
              <Dialog open={showProvidentFieldSelector} onOpenChange={setShowProvidentFieldSelector}>
                <DialogTrigger asChild>
                  <Button variant="outline" size="sm">
                    <Settings className="mr-2 h-4 w-4" /> 显示字段
                  </Button>
                </DialogTrigger>
                <DialogContent className="max-w-md">
                  <DialogHeader>
                    <DialogTitle>自定义显示字段</DialogTitle>
                    <DialogDescription>勾选后即可在表格中显示对应列。</DialogDescription>
                  </DialogHeader>
                  <div className="max-h-80 overflow-y-auto space-y-2">
                    {PROVIDENT_COLUMN_CONFIGS.map((column) => (
                      <label key={column.id} className="flex items-center gap-2 text-sm">
                        <input
                          type="checkbox"
                          className="rounded border-gray-300"
                          checked={providentVisibleColumns.includes(column.id)}
                          onChange={() => handleProvidentFieldToggle(column.id)}
                        />
                        {column.label}
                      </label>
                    ))}
                  </div>
                  <DialogFooter className="flex flex-col gap-2 sm:flex-row sm:justify-end">
                    <Button variant="ghost" onClick={resetProvidentFieldsToDefault}>恢复默认</Button>
                    <Button onClick={() => setShowProvidentFieldSelector(false)}>完成</Button>
                  </DialogFooter>
                </DialogContent>
              </Dialog>
            </div>
            <div className="mb-4 flex flex-wrap gap-4 rounded-md border border-dashed border-muted-foreground/40 bg-muted/20 p-3 text-xs text-muted-foreground">
              <span>个人金额合计：{formatAmountValue(providentSummary.personal)}</span>
              <span>单位金额合计：{formatAmountValue(providentSummary.company)}</span>
              <span>合计：{formatAmountValue(providentSummary.total)}</span>
            </div>
            <DataTableWrapper height="h-[60vh]">
                <Table className="min-w-full table-auto text-sm">
                  <TableHeader>
                    <TableRow className="text-muted-foreground">
                      <TableHead className={cn(ALIGNMENT_CLASS.center, "w-12")}>
                        <input
                          ref={providentSelectAllRef}
                          type="checkbox"
                          className="h-4 w-4 rounded border-muted-foreground"
                          checked={sortedProvidentRecords.length > 0 && allProvidentFilteredSelected}
                          onChange={(event) => toggleAllProvidentRecords(event.target.checked)}
                        />
                      </TableHead>
                      {providentColumnsForRender.map((column) => {
                        const headClass = cn("select-none whitespace-nowrap", ALIGNMENT_CLASS.left);
                        return (
                          <TableHead
                            key={column.id}
                            draggable
                            onDragStart={(event) => handleColumnDragStart(event, column.id)}
                            onDragOver={handleColumnDragOver}
                            onDrop={(event) => handleProvidentColumnDrop(event, column.id)}
                            onDragEnd={handleColumnDragEnd}
                            onClick={() => handleProvidentSortClick(column.id)}
                            className={headClass}
                          >
                            <span className="flex items-center gap-1">
                              {column.label}
                              {providentSort.key === column.id && renderSortIndicator(providentSort, column.id)}
                            </span>
                          </TableHead>
                        );
                      })}
                      <TableHead className={cn("w-16", ALIGNMENT_CLASS.center)}>操作</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {sortedProvidentRecords.length === 0 ? (
                      <TableRow>
                        <TableCell colSpan={providentColumnsForRender.length + 2} className="py-10 text-center text-sm text-muted-foreground">
                          {providentLoading ? "公积金数据加载中..." : "暂无公积金记录，请先导入或新增"}
                        </TableCell>
                      </TableRow>
                    ) : (
                      sortedProvidentRecords.map((record) => (
                        <TableRow
                          key={record.id}
                          className="cursor-pointer hover:bg-muted/50"
                          onDoubleClick={() => openProvidentDialog("edit", { record })}
                        >
                        <TableCell className={ALIGNMENT_CLASS.center}>
                          <input
                            type="checkbox"
                            className="h-4 w-4 rounded border-muted-foreground"
                            checked={selectedProvidentIds.includes(record.id)}
                            onChange={(event) => toggleProvidentSelection(record.id, event.target.checked)}
                          />
                        </TableCell>
                        {providentColumnsForRender.map((column) => {
                          const value = getProvidentCellValue(record, column.id);
                          const alignClass = cn("align-top", ALIGNMENT_CLASS.left, {
                            "tabular-nums whitespace-nowrap": column.numeric,
                            "whitespace-normal break-words": !column.numeric,
                          });
                          return (
                            <TableCell key={column.id} className={alignClass}>
                              {value}
                            </TableCell>
                          );
                        })}
                        <TableCell className={cn(ALIGNMENT_CLASS.center, "w-16")}>
                          <DropdownMenu>
                            <DropdownMenuTrigger asChild>
                              <Button variant="ghost" size="icon" className="h-8 w-8" aria-label="公积金操作">
                                  <MoreHorizontal className="h-4 w-4" />
                                </Button>
                              </DropdownMenuTrigger>
                              <DropdownMenuContent align="end" className="w-40">
                                <DropdownMenuItem
                                  onClick={(event) => {
                                    event.stopPropagation();
                                    openProvidentDialog("edit", { record });
                                  }}
                                  className="gap-2"
                                >
                                  <Eye className="h-4 w-4" /> 查看详情
                                </DropdownMenuItem>
                                {record.status !== "sealed" ? (
                                  /* P7.1：办理封存为状态变更，需 insurance.edit 权限 */
                                  <RequirePermission resource="insurance" action="edit">
                                    <DropdownMenuItem
                                      onClick={(event) => {
                                        event.stopPropagation();
                                        openSealDialog(record);
                                      }}
                                      className="gap-2"
                                    >
                                      <ShieldCheck className="h-4 w-4" /> 办理封存
                                    </DropdownMenuItem>
                                  </RequirePermission>
                                ) : (
                                  /* P7.1：办理启封为状态变更，需 insurance.edit 权限 */
                                  <RequirePermission resource="insurance" action="edit">
                                    <DropdownMenuItem
                                      onClick={(event) => {
                                        event.stopPropagation();
                                        openUnsealDialog(record);
                                      }}
                                      className="gap-2"
                                    >
                                      <Unlock className="h-4 w-4" /> 办理启封
                                    </DropdownMenuItem>
                                  </RequirePermission>
                                )}
                              </DropdownMenuContent>
                            </DropdownMenu>
                          </TableCell>
                        </TableRow>
                      ))
                    )}
                  </TableBody>
                </Table>
                <ScrollBar orientation="horizontal" />
            </DataTableWrapper>
            <div className="mt-4 flex flex-wrap items-center justify-between gap-3 text-xs text-muted-foreground">
              <span>共 {recordCount} 条记录，当前筛选 {filteredCount} 条</span>
              <span className="flex items-center gap-2">
                <Badge variant="outline">导入文件：{importedProvidentFileName || "未导入"}</Badge>
                <Badge variant="outline">模板版本：v1.2</Badge>
              </span>
            </div>
          </CardContent>
        </Card>

      </>
    );
  };
  const getTemplateOptions = (key?: SocialOptionSelectableKey) => {
    if (!key || !socialOptions) {
      return [];
    }
    const optionSet = socialOptions[key];
    if (!optionSet) {
      return [];
    }
    return optionSet.options;
  };

  const handleIncreaseFieldChange = (key: IncreaseFieldKey, value: string) => {
    setInsuranceIncreaseForm((prev) => ({ ...prev, [key]: value }));
  };

  const handleDecreaseFieldChange = (key: DecreaseFieldKey, value: string) => {
    setInsuranceDecreaseForm((prev) => ({ ...prev, [key]: value }));
  };

  const renderIncreaseField = (config: BaseInsuranceFieldConfig<IncreaseFieldKey>) => {
    const fieldValue = insuranceIncreaseForm[config.key] ?? "";
    const baseClass = `flex flex-col gap-2 min-w-[260px] ${config.fullWidth ? "md:col-span-2 xl:col-span-3" : ""}`;
    const commonProps = {
      value: fieldValue,
      placeholder: config.placeholder,
      onChange: (event: ChangeEvent<HTMLInputElement | HTMLTextAreaElement>) =>
        handleIncreaseFieldChange(config.key, event.target.value),
    };
    let inputNode: ReactNode;
    switch (config.input) {
      case "number":
        inputNode = (
          <Input
            type="number"
            min="0"
            inputMode={config.inputMode ?? "decimal"}
            {...commonProps}
          />
        );
        break;
      case "date":
        inputNode = (
          <Input
            type="date"
            {...commonProps}
          />
        );
        break;
      case "textarea":
        inputNode = (
          <Textarea
            rows={config.fullWidth ? 4 : 3}
            {...commonProps}
          />
        );
        break;
      case "select": {
        const options = getTemplateOptions(config.selectKey);
        const disabled = options.length === 0;
        inputNode = (
          <Select
            value={fieldValue}
            onValueChange={(value) => handleIncreaseFieldChange(config.key, value)}
            disabled={disabled}
          >
            <SelectTrigger className="w-full" aria-disabled={disabled}>
              <SelectValue placeholder={config.placeholder ?? `选择${config.label}`} />
            </SelectTrigger>
            <SelectContent>
              {options.map((option) => (
                <SelectItem key={option} value={option}>
                  {option}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        );
        break;
      }
      default:
        inputNode = (
          <Input
            type="text"
            inputMode={config.inputMode ?? "text"}
            {...commonProps}
          />
        );
        break;
    }
    return (
      <div key={config.key} className={baseClass}>
        <Label>
          {config.label}
          {config.required ? " *" : ""}
        </Label>
        {inputNode}
      </div>
    );
  };

  const renderDecreaseField = (config: BaseInsuranceFieldConfig<DecreaseFieldKey>) => {
    const fieldValue = insuranceDecreaseForm[config.key] ?? "";
    const baseClass = `flex flex-col gap-2 min-w-[260px] ${config.fullWidth ? "md:col-span-2 xl:col-span-3" : ""}`;
    const commonProps = {
      value: fieldValue,
      placeholder: config.placeholder,
      onChange: (event: ChangeEvent<HTMLInputElement | HTMLTextAreaElement>) =>
        handleDecreaseFieldChange(config.key, event.target.value),
    };
    let inputNode: ReactNode;
    switch (config.input) {
      case "date":
        inputNode = (
          <Input
            type="date"
            {...commonProps}
          />
        );
        break;
      case "select": {
        const options = getTemplateOptions(config.selectKey);
        const disabled = options.length === 0;
        inputNode = (
          <Select
            value={fieldValue}
            onValueChange={(value) => handleDecreaseFieldChange(config.key, value)}
            disabled={disabled}
          >
            <SelectTrigger className="w-full" aria-disabled={disabled}>
              <SelectValue placeholder={config.placeholder ?? `选择${config.label}`} />
            </SelectTrigger>
            <SelectContent>
              {options.map((option) => (
                <SelectItem key={option} value={option}>
                  {option}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        );
        break;
      }
      default:
        inputNode = (
          <Input
            type="text"
            inputMode={config.inputMode ?? "text"}
            {...commonProps}
          />
        );
        break;
    }
    return (
      <div key={config.key} className={baseClass}>
        <Label>
          {config.label}
          {config.required ? " *" : ""}
        </Label>
        {inputNode}
      </div>
    );
  };

  const renderResignedExtraCell = (employee: Employee, columnId: typeof RESIGNED_EXTRA_COLUMNS[number]) => {
    if (columnId === "resignDate") {
      return displayDate(employee.resignDate);
    }
    if (columnId === "resignReasons") {
      return employee.resignReasons && employee.resignReasons.length > 0
        ? employee.resignReasons.map((reason) => getResignReasonLabel(reason)).join("，")
        : "-";
    }
    if (columnId === "resignProof") {
      return employee.resignProofUrl ? (
        <Button
          variant="link"
          size="sm"
          className="px-0"
          onClick={() => handleDownloadResignProof(employee)}
        >
          <Download className="h-4 w-4 mr-1" />
          {employee.resignProofName || "下载证明"}
        </Button>
      ) : (
        <span className="text-sm text-muted-foreground">-</span>
      );
    }
    return "-";
  };

  // 员工离职
  const handleResignEmployee = async () => {
    if (!selectedEmployee) {
      toast.error("请选择需要办理离职的员工");
      return;
    }

    const numericId = Number(selectedEmployee.id);
    if (!Number.isFinite(numericId) || numericId <= 0) {
      toast.error("当前员工数据尚未同步至后台，无法办理离职");
      return;
    }

    if (!resignDate) {
      toast.error("请选择离职日期");
      return;
    }

    const normalizedResignDate = normalizeDateInput(resignDate);
    if (!normalizedResignDate) {
      toast.error("离职日期格式不正确，请重新选择");
      return;
    }

    if (resignProofFile && resignProofFile.size > MAX_RESIGN_PROOF_SIZE_BYTES) {
      const message = `离职证明文件过大，请上传不超过 ${formatFileSize(MAX_RESIGN_PROOF_SIZE_BYTES)} 的文件`;
      setResignProofError(message);
      toast.error(message);
      return;
    }

    setResignProofError("");

    if (!token) {
      toast.error("登录状态失效，请重新登录后再办理离职");
      return;
    }

    setResignSubmitting(true);
    try {
      await resignEmployeeApi(
        numericId,
        normalizedResignDate,
        resignProofFile ?? null,
        token ?? undefined,
        resignReasons,
      );
      toast.success("员工离职处理成功");

      setShowResignDialog(false);
      setSelectedEmployee(null);
      setResignProofFile(null);
      setResignProofError("");
      setResignReasons([]);
      setSelectedEmployeeIds((prev) => prev.filter((id) => id !== selectedEmployee.id));
      await loadEmployees();
    } catch (error) {
      console.error("[EmployeeManagement] resign failed", error);
      toast.error(error instanceof Error ? error.message : "离职办理失败，请稍后重试");
    } finally {
      setResignSubmitting(false);
    }
  };


  // 下载模板
  const downloadTemplate = async () => {
    if (downloadingTemplate) {
      return;
    }
    setDownloadingTemplate(true);
    try {
      const blob = await downloadEmployeeTemplate();
      const url = URL.createObjectURL(blob);
      const link = document.createElement("a");
      link.href = url;
      link.download = `员工导入模板-${new Date().toISOString().slice(0, 10)}.xlsx`;
      document.body.appendChild(link);
      link.click();
      document.body.removeChild(link);
      URL.revokeObjectURL(url);
      toast.success("模板下载成功");
    } catch (error) {
      console.error("[EmployeeTemplate] download failed", error);
      toast.error(error instanceof Error ? error.message : "模板下载失败，请稍后重试");
    } finally {
      setDownloadingTemplate(false);
    }
  };

  const downloadResignedTemplate = async () => {
    if (downloadingResignedTemplate) {
      return;
    }
    setDownloadingResignedTemplate(true);
    try {
      const blob = await downloadResignedEmployeeTemplate();
      const url = URL.createObjectURL(blob);
      const link = document.createElement("a");
      link.href = url;
      link.download = `离职员工导入模板-${new Date().toISOString().slice(0, 10)}.xlsx`;
      document.body.appendChild(link);
      link.click();
      document.body.removeChild(link);
      URL.revokeObjectURL(url);
      toast.success("离职模板下载成功");
    } catch (error) {
      console.error("[ResignedTemplate] download failed", error);
      toast.error(error instanceof Error ? error.message : "离职模板下载失败，请稍后重试");
    } finally {
      setDownloadingResignedTemplate(false);
    }
  };

  const handleExportActive = async (scope: "selected" | "filtered" | "all") => {
    if (exporting) {
      return;
    }

    if (scope === "selected" && selectedEmployeeIds.length === 0) {
      toast.error("请先勾选需要导出的员工");
      return;
    }

    const numericIds = selectedEmployeeIds
      .map((id) => Number(id))
      .filter((value) => Number.isFinite(value) && value > 0)
      .map((value) => Math.trunc(value));
    const fallbackIdNumbers = selectedEmployeeIds
      .filter((id) => Number.isNaN(Number(id)))
      .map((id) => id.trim())
      .filter(Boolean);
    const idsParam = scope === "selected" && numericIds.length > 0 ? numericIds : undefined;
    const idNumbersParam = scope === "selected" && fallbackIdNumbers.length > 0 ? fallbackIdNumbers : undefined;

    setExporting(true);
    try {
      const payloadStatus = scope === "all" ? undefined : "active";
      const payloadDepartment = scope === "filtered" && departmentFilter !== "all" ? departmentFilter : undefined;
      const payloadSearch = scope === "filtered" && searchTerm.trim() ? searchTerm.trim() : undefined;

      const blob = await exportEmployees({
        scope,
        status: payloadStatus,
        department: payloadDepartment,
        search: payloadSearch,
        ids: idsParam,
        idNumbers: idNumbersParam,
      });

      const labelMap: Record<typeof scope, string> = {
        selected: "选中",
        filtered: "筛选",
        all: "全部",
      };
      downloadBlob(blob, `在职员工-${labelMap[scope]}-${new Date().toISOString().slice(0, 10)}.xlsx`);
      toast.success("导出完成，文件已保存");
    } catch (error) {
      console.error("[EmployeeExport] failed", error);
      toast.error(error instanceof Error ? error.message : "导出失败，请稍后再试");
    } finally {
      setExporting(false);
    }
  };

  // 从身份证号码计算年龄和出生月份
  const calculateAgeFromIdNumber = (idNumber: string) => {
    const normalized = idNumber.replace(/\s+/g, "").toUpperCase();
    if (!normalized || normalized.length !== 18) return { age: "", birthMonth: "" };

    const birth = normalized.substring(6, 14);
    const birthYear = parseInt(birth.substring(0, 4));
    const birthMonth = parseInt(birth.substring(4, 6));
    const birthDay = parseInt(birth.substring(6, 8));

    const today = new Date();
    const currentYear = today.getFullYear();
    const currentMonth = today.getMonth() + 1;
    const currentDay = today.getDate();

    let age = currentYear - birthYear;
    if (currentMonth < birthMonth || (currentMonth === birthMonth && currentDay < birthDay)) {
      age--;
    }

    return {
      age: age.toString(),
      birthMonth: `${birthYear}-${String(birthMonth).padStart(2, "0")}-${String(birthDay).padStart(2, "0")}`,
    };
  };

  // 从入职时间计算工龄
  const calculateWorkYears = (hireDate: string) => {
    const normalized = normalizeDateInput(hireDate);
    if (!normalized) return "";

    const hire = new Date(normalized);
    const today = new Date();
    const diffTime = Math.abs(today.getTime() - hire.getTime());
    if (!Number.isFinite(diffTime)) return "";
    const diffYears = diffTime / MS_PER_YEAR;
    return formatWorkYearsValue(diffYears.toString());
  };

  // 生成工号
  const generateEmployeeId = (department: string) => {
    const departmentCode = DEPARTMENT_CODES[department as keyof typeof DEPARTMENT_CODES];
    if (!departmentCode) return "";

    // 查找该部门最大的工号
    const departmentEmployees = [...employees, ...resignedEmployees]
      .filter(emp => emp.employeeId && emp.employeeId.startsWith(departmentCode))
      .map(emp => emp.employeeId!)
      .sort();

    if (departmentEmployees.length === 0) {
      return `${departmentCode}001`;
    }

    const lastId = departmentEmployees[departmentEmployees.length - 1];
    if (!lastId) {
      return `${departmentCode}001`;
    }
    
    const lastNumber = parseInt(lastId.substring(departmentCode.length));
    const nextNumber = lastNumber + 1;

    return `${departmentCode}${nextNumber.toString().padStart(3, '0')}`;
  };

  // 自动生成工号
  const handleGenerateEmployeeId = () => {
    if (!newEmployee.department) {
      toast.error("请先选择部门");
      return;
    }

    const generatedId = generateEmployeeId(newEmployee.department);
    setNewEmployee(prev => ({ ...prev, employeeId: generatedId }));
    toast.success(`已生成工号：${generatedId}`);
  };

  // 处理身份证号变化
  const handleIdNumberChange = (idNumber: string, isEdit = false) => {
    const normalized = idNumber.replace(/\s+/g, "").toUpperCase();
    const { age, birthMonth } = calculateAgeFromIdNumber(normalized);

    if (isEdit) {
      setEditEmployee(prev => ({
        ...prev,
        idNumber: normalized,
        age: age || prev.age,
        birthMonth: birthMonth || prev.birthMonth
      }));
    } else {
      setNewEmployee(prev => ({
        ...prev,
        idNumber: normalized,
        age: age || prev.age,
        birthMonth: birthMonth || prev.birthMonth
      }));
    }
  };

  // 处理入职时间变化
  const handleHireDateChange = (hireDate: string, isEdit = false) => {
    const normalized = normalizeDateInput(hireDate);
    const workYears = calculateWorkYears(normalized);

    if (isEdit) {
      setEditEmployee(prev => ({
        ...prev,
        hireDate: normalized,
        workYears: workYears || prev.workYears
      }));
    } else {
      setNewEmployee(prev => ({
        ...prev,
        hireDate: normalized,
        workYears: workYears || prev.workYears
      }));
    }
  };

  const handleAgeBlur = (value: string, isEdit = false) => {
    const normalized = formatAgeValue(value);
    if (isEdit) {
      setEditEmployee(prev => ({ ...prev, age: normalized }));
    } else {
      setNewEmployee(prev => ({ ...prev, age: normalized }));
    }
  };

  const handleWorkYearsBlur = (value: string, isEdit = false) => {
    const normalized = formatWorkYearsValue(value);
    if (isEdit) {
      setEditEmployee(prev => ({ ...prev, workYears: normalized }));
    } else {
      setNewEmployee(prev => ({ ...prev, workYears: normalized }));
    }
  };

  // 编辑员工
  const handleEditEmployee = (employee: Employee) => {
    const context: "active" | "resigned" = employee.status === "resigned" ? "resigned" : "active";
    const derivedStatus = deriveInsuranceStatus(employee.idNumber, context);
    setEditEmployee({
      ...employee,
      insuranceStatus: employee.insuranceStatus?.trim() || derivedStatus,
    });
    setShowEditEmployee(true);
  };

  // 保存编辑的员工信息
  const handleSaveEmployee = () => {
    // 必填字段验证
    if (!editEmployee.name?.trim()) {
      toast.error("请填写姓名");
      return;
    }
    if (!editEmployee.idNumber?.trim()) {
      toast.error("请填写身份证号");
      return;
    }
    if (!editEmployee.department?.trim()) {
      toast.error("请选择部门");
      return;
    }

    // 身份证号格式验证
    const idRegex = /^[1-9]\d{5}(18|19|([23]\d))\d{2}((0[1-9])|(10|11|12))(([0-2][1-9])|10|20|30|31)\d{3}[0-9Xx]$/;
    if (!idRegex.test(editEmployee.idNumber)) {
      toast.error("身份证号格式不正确");
      return;
    }

    // 手机号格式验证（如果填写了）
    if (editEmployee.phone && !/^1[3-9]\d{9}$/.test(editEmployee.phone)) {
      toast.error("手机号格式不正确");
      return;
    }

    // 邮箱格式验证（如果填写了）
    if (editEmployee.email && !/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(editEmployee.email)) {
      toast.error("邮箱格式不正确");
      return;
    }

    // 年龄验证（如果填写了）
    if (editEmployee.age && (isNaN(Number(editEmployee.age)) || Number(editEmployee.age) < 16 || Number(editEmployee.age) > 100)) {
      toast.error("年龄应为16-100之间的数字");
      return;
    }

    // 工龄验证（如果填写了）
    if (editEmployee.workYears && (isNaN(Number(editEmployee.workYears)) || Number(editEmployee.workYears) < 0)) {
      toast.error("工龄应为非负数字");
      return;
    }

    // 检查身份证号是否重复（排除当前员工）
    const sanitizedIdNumber = editEmployee.idNumber.trim().toUpperCase();
    const existingEmployee = [...employees, ...resignedEmployees].find(
      emp => emp.idNumber === sanitizedIdNumber && emp.id !== editEmployee.id
    );
    if (existingEmployee) {
      toast.error("该身份证号已存在");
      return;
    }

    const autoInfo = calculateAgeFromIdNumber(sanitizedIdNumber);
    const normalizedHireDate = normalizeDateInput(editEmployee.hireDate || "");
    const normalizedAge = formatAgeValue(editEmployee.age || autoInfo.age);
    const normalizedWorkYears = formatWorkYearsValue(editEmployee.workYears || calculateWorkYears(normalizedHireDate));
    const normalizedBirthMonth = normalizeBirthMonth(editEmployee.birthMonth || autoInfo.birthMonth);

    const updatedEmployee: Employee = {
      ...editEmployee,
      name: editEmployee.name!.trim(),
      idNumber: sanitizedIdNumber,
      department: editEmployee.department!.trim(),
      employeeId: editEmployee.employeeId?.trim() || editEmployee.id || "",
      position: editEmployee.position?.trim() || "",
      gender: editEmployee.gender?.trim() || "",
      hireDate: normalizedHireDate,
      age: normalizedAge,
      workYears: normalizedWorkYears,
      birthMonth: normalizedBirthMonth,
      education: editEmployee.education?.trim() || "",
      politicalStatus: editEmployee.politicalStatus?.trim() || "",
      workClothingSize: editEmployee.workClothingSize?.trim() || "",
      safetyShoeSize: editEmployee.safetyShoeSize?.trim() || "",
      householdType: editEmployee.householdType?.trim() || "",
      ethnicity: editEmployee.ethnicity?.trim() || "",
      nativePlace: editEmployee.nativePlace?.trim() || "",
      idAddress: editEmployee.idAddress?.trim() || "",
      maritalStatus: editEmployee.maritalStatus?.trim() || "",
      insuranceStatus: editEmployee.insuranceStatus?.trim() || "",
      hasBirth: editEmployee.hasBirth?.trim() || "",
      phone: editEmployee.phone?.trim() || "",
      emergencyContact: editEmployee.emergencyContact?.trim() || "",
      emergencyPhone: editEmployee.emergencyPhone?.trim() || "",
      currentAddress: editEmployee.currentAddress?.trim() || "",
      graduateSchool: editEmployee.graduateSchool?.trim() || "",
      major: editEmployee.major?.trim() || "",
      graduationTime: normalizeDateInput(editEmployee.graduationTime),
      socialInsuranceNumber: editEmployee.socialInsuranceNumber?.trim() || "",
      providentFundNumber: editEmployee.providentFundNumber?.trim() || "无",
      email: editEmployee.email?.trim() || "",
      remarks: editEmployee.remarks?.trim() || "",
    } as Employee;

    // 根据员工状态更新对应的列表
    if (updatedEmployee.status === 'active') {
      setEmployees(prev => prev.map(emp => emp.id === updatedEmployee.id ? updatedEmployee : emp));
    } else if (updatedEmployee.status === 'resigned') {
      setResignedEmployees(prev => prev.map(emp => emp.id === updatedEmployee.id ? updatedEmployee : emp));
    }

    setShowEditEmployee(false);
    setEditEmployee({});
    toast.success("员工信息更新成功");
  };

  const handleOpenResignedDetail = (employee: Employee, options?: { mode?: "detail" | "edit" }) => {
    if (options?.mode === "edit") {
      handleEditEmployee(employee);
      return;
    }
    resetResignProofPreview();
    setResignedDetailEmployee(employee);
    setShowResignedDetail(true);
  };

  const handleCloseResignedDetail = () => {
    setShowResignedDetail(false);
    setResignedDetailEmployee(null);
    resetResignProofPreview();
  };

  return (
    <PageTransition className="mx-auto flex w-full max-w-none flex-col gap-6 p-6 pb-16">
      {/* 页面标题 */}
      <header className="flex flex-col gap-2">
        <div className="flex flex-wrap items-center justify-between gap-4">
          <div>
            <h1 className="text-3xl font-bold tracking-tight">员工花名册</h1>
            <p className="text-muted-foreground">
              管理员工信息、离职记录和社保增减变动
            </p>
          </div>
        </div>
      </header>

      {/* 主要内容 */}
      <Tabs
        value={activeRosterTab}
        onValueChange={(value) => handleRosterTabChange(value as RosterTab)}
        className="w-full"
      >
        <TabsList className="grid w-full grid-cols-5">
          <TabsTrigger value="active">在职员工</TabsTrigger>
          <TabsTrigger value="resigned">离职员工</TabsTrigger>
          <TabsTrigger value="insurance-increase">社保增加</TabsTrigger>
          <TabsTrigger value="insurance-decrease">社保减少</TabsTrigger>
          <TabsTrigger value="provident">公积金</TabsTrigger>
        </TabsList>

        {/* 在职员工标签页 */}
        <TabsContent value="active" className="space-y-6">
          <Card>
            <CardHeader>
              <div className="flex items-center justify-between">
                <div>
                  <CardTitle>在职员工管理</CardTitle>
                  <CardDescription>
                    管理在职员工信息，支持单个添加和批量导入
                  </CardDescription>
                </div>
                <div className="flex gap-2">
                  {!hasActiveSelection ? (
                    <>
                      {/* P7.1：新增员工需 employee.create 权限（页面级按钮，无权限隐藏）。
                          注意：RequirePermission 必须包裹在 Dialog 外层，若放在 DialogTrigger asChild
                          内部会返回 Fragment，导致 Radix 无法注入 ref/onClick，点击新增无法打开弹窗 */}
                      <RequirePermission resource="employee" action="create">
                        <Dialog open={showAddEmployee} onOpenChange={setShowAddEmployee}>
                          <DialogTrigger asChild>
                            <Button>
                              <Plus className="h-4 w-4 mr-2" />
                              新增
                            </Button>
                          </DialogTrigger>
                        <DialogContent className={RESPONSIVE_DIALOG_CLASS}>
                          <DialogHeader>
                            <DialogTitle>新增员工</DialogTitle>
                            <DialogDescription>
                              填写员工详细信息，按分类录入
                            </DialogDescription>
                          </DialogHeader>
                       <ScrollArea className="max-h-[60vh] pr-2">
                          <Tabs defaultValue="basic" className="w-full space-y-4">
                            <TabsList className="grid w-full grid-cols-4">
                              <TabsTrigger value="basic">基本信息</TabsTrigger>
                              <TabsTrigger value="personal">个人信息</TabsTrigger>
                              <TabsTrigger value="contact">联系信息</TabsTrigger>
                              <TabsTrigger value="other">其他信息</TabsTrigger>
                            </TabsList>

                        {/* 基本信息 */}
                        <TabsContent value="basic" className="space-y-4">
                          <div className="rounded-2xl border bg-muted/10 p-3 sm:p-4 shadow-sm">
                            <div className={RESPONSIVE_FIELD_GRID_CLASS}>
                            <div className="space-y-2">
                              <Label htmlFor="employeeId">工号</Label>
                              <div className="flex gap-2">
                                <Input
                                  id="employeeId"
                                  value={newEmployee.employeeId}
                                  onChange={(e) => setNewEmployee(prev => ({ ...prev, employeeId: e.target.value }))}
                                  placeholder="选择部门后可自动生成"
                                />
                                <Button
                                  type="button"
                                  variant="outline"
                                  size="sm"
                                  onClick={handleGenerateEmployeeId}
                                  disabled={!newEmployee.department}
                                  title="根据部门规则生成工号"
                                >
                                  <Dice6 className="h-4 w-4" />
                                </Button>
                              </div>
                            </div>
                            <div className="space-y-2">
                              <Label htmlFor="name">姓名 *</Label>
                              <Input
                                id="name"
                                value={newEmployee.name}
                                onChange={(e) => setNewEmployee(prev => ({ ...prev, name: e.target.value }))}
                                required
                              />
                            </div>
                            <div className="space-y-2">
                              <Label htmlFor="department">部门 *</Label>
                              <SearchableSelect
                                id="department"
                                value={newEmployee.department || ""}
                                onChange={(value) => setNewEmployee(prev => ({ ...prev, department: value }))}
                                options={getDepartments()}
                                placeholder="选择或搜索部门"
                              />
                            </div>
                            <div className="space-y-2">
                              <Label htmlFor="position">岗位</Label>
                              <SearchableSelect
                                id="position"
                                value={newEmployee.position || ""}
                                onChange={(value) => setNewEmployee(prev => ({ ...prev, position: value }))}
                                options={positionOptions}
                                placeholder="选择或输入岗位"
                              />
                            </div>
                            <div className="space-y-2">
                              <Label htmlFor="gender">性别</Label>
                              <Select value={newEmployee.gender} onValueChange={(value) => setNewEmployee(prev => ({ ...prev, gender: value }))}>
                                <SelectTrigger>
                                  <SelectValue placeholder="选择性别" />
                                </SelectTrigger>
                                <SelectContent>
                                  <SelectItem value="男">男</SelectItem>
                                  <SelectItem value="女">女</SelectItem>
                                </SelectContent>
                              </Select>
                            </div>
                            <div className="space-y-2">
                              <Label htmlFor="hireDate">入职时间</Label>
                              <Input
                                id="hireDate"
                                type="date"
                                value={newEmployee.hireDate}
                                onChange={(e) => handleHireDateChange(e.target.value)}
                              />
                            </div>
                            <div className="space-y-2">
                              <Label htmlFor="age">年龄</Label>
                              <Input
                                id="age"
                                type="number"
                                value={newEmployee.age}
                                onChange={(e) => setNewEmployee(prev => ({ ...prev, age: e.target.value }))}
                                onBlur={(e) => handleAgeBlur(e.target.value)}
                              />
                            </div>
                            <div className="space-y-2">
                              <Label htmlFor="workYears">工龄</Label>
                              <Input
                                id="workYears"
                                inputMode="decimal"
                                value={newEmployee.workYears}
                                onChange={(e) => setNewEmployee(prev => ({ ...prev, workYears: e.target.value }))}
                                onBlur={(e) => handleWorkYearsBlur(e.target.value)}
                              />
                            </div>
                            </div>
                          </div>
                        </TabsContent>

                        {/* 个人信息 */}
                        <TabsContent value="personal" className="space-y-4">
                          <div className="rounded-2xl border bg-muted/10 p-3 sm:p-4 space-y-4 shadow-sm">
                          <div className={RESPONSIVE_FIELD_GRID_CLASS}>
                            <div className="space-y-2">
                              <Label htmlFor="birthMonth">出生月份</Label>
                              <Input
                                id="birthMonth"
                                type="date"
                                value={newEmployee.birthMonth}
                                onChange={(e) => setNewEmployee((prev) => ({ ...prev, birthMonth: e.target.value }))}
                                onBlur={(e) => setNewEmployee((prev) => ({ ...prev, birthMonth: normalizeBirthMonth(e.target.value) }))}
                              />
                            </div>
                            <div className="space-y-2">
                              <Label htmlFor="education">文化程度</Label>
                              <SearchableSelect
                                id="education"
                                value={newEmployee.education || ""}
                                onChange={(value) => setNewEmployee(prev => ({ ...prev, education: value }))}
                                options={educationOptions}
                                placeholder="选择或输入文化程度"
                              />
                            </div>
                            <div className="space-y-2">
                              <Label htmlFor="politicalStatus">政治面貌</Label>
                              <Select value={newEmployee.politicalStatus} onValueChange={(value) => setNewEmployee(prev => ({ ...prev, politicalStatus: value }))}>
                                <SelectTrigger>
                                  <SelectValue placeholder="选择政治面貌" />
                                </SelectTrigger>
                                <SelectContent>
                                  <SelectItem value="群众">群众</SelectItem>
                                  <SelectItem value="团员">团员</SelectItem>
                                  <SelectItem value="党员">党员</SelectItem>
                                  <SelectItem value="民主党派">民主党派</SelectItem>
                                </SelectContent>
                              </Select>
                            </div>
                            <div className="space-y-2">
                              <Label htmlFor="ethnicity">民族</Label>
                              <Input
                                id="ethnicity"
                                value={newEmployee.ethnicity}
                                onChange={(e) => setNewEmployee(prev => ({ ...prev, ethnicity: e.target.value }))}
                              />
                            </div>
                            <div className="space-y-2">
                              <Label htmlFor="nativePlace">籍贯</Label>
                              <SearchableSelect
                                id="nativePlace"
                                value={newEmployee.nativePlace || ""}
                                onChange={(value) => setNewEmployee(prev => ({ ...prev, nativePlace: value }))}
                                options={nativePlaceOptions}
                                placeholder="选择或输入籍贯"
                              />
                            </div>
                            <div className="space-y-2">
                              <Label htmlFor="maritalStatus">婚姻状况</Label>
                              <Select value={newEmployee.maritalStatus} onValueChange={(value) => setNewEmployee(prev => ({ ...prev, maritalStatus: value }))}>
                                <SelectTrigger>
                                  <SelectValue placeholder="选择婚姻状况" />
                                </SelectTrigger>
                                <SelectContent>
                                  <SelectItem value="未婚">未婚</SelectItem>
                                  <SelectItem value="已婚">已婚</SelectItem>
                                  <SelectItem value="离异">离异</SelectItem>
                                  <SelectItem value="丧偶">丧偶</SelectItem>
                                </SelectContent>
                              </Select>
                            </div>
                            <div className="space-y-2">
                              <Label htmlFor="householdType">户口性质</Label>
                              <Select value={newEmployee.householdType} onValueChange={(value) => setNewEmployee(prev => ({ ...prev, householdType: value }))}>
                                <SelectTrigger>
                                  <SelectValue placeholder="选择户口性质" />
                                </SelectTrigger>
                                <SelectContent>
                                  <SelectItem value="城镇">城镇</SelectItem>
                                  <SelectItem value="农村">农村</SelectItem>
                                </SelectContent>
                              </Select>
                            </div>
                            <div className="space-y-2">
                              <Label htmlFor="hasBirth">是否生育</Label>
                              <Select value={newEmployee.hasBirth} onValueChange={(value) => setNewEmployee(prev => ({ ...prev, hasBirth: value }))}>
                                <SelectTrigger>
                                  <SelectValue placeholder="选择是否生育" />
                                </SelectTrigger>
                                <SelectContent>
                                  <SelectItem value="是">是</SelectItem>
                                  <SelectItem value="否">否</SelectItem>
                                </SelectContent>
                              </Select>
                            </div>
                          </div>
                          <div className={RESPONSIVE_FIELD_GRID_CLASS}>
                            <div className="space-y-2">
                              <Label htmlFor="idAddress">身份证地址</Label>
                              <Input
                                id="idAddress"
                                value={newEmployee.idAddress}
                                onChange={(e) => setNewEmployee(prev => ({ ...prev, idAddress: e.target.value }))}
                              />
                            </div>
                            <div className="space-y-2">
                              <Label htmlFor="idNumber">身份证号码 *</Label>
                              <Input
                                id="idNumber"
                                value={newEmployee.idNumber}
                                onChange={(e) => handleIdNumberChange(e.target.value)}
                                required
                                placeholder="自动计算年龄和出生月份"
                              />
                            </div>
                          </div>
                          </div>
                        </TabsContent>

                        {/* 联系信息 */}
                        <TabsContent value="contact" className="space-y-4">
                          <div className="rounded-2xl border bg-muted/10 p-3 sm:p-4 space-y-4 shadow-sm">
                          <div className={RESPONSIVE_FIELD_GRID_CLASS}>
                            <div className="space-y-2">
                              <Label htmlFor="phone">联系电话</Label>
                              <Input
                                id="phone"
                                value={newEmployee.phone}
                                onChange={(e) => setNewEmployee(prev => ({ ...prev, phone: e.target.value }))}
                              />
                            </div>
                            <div className="space-y-2">
                              <Label htmlFor="emergencyContact">紧急联系人</Label>
                              <Input
                                id="emergencyContact"
                                value={newEmployee.emergencyContact}
                                onChange={(e) => setNewEmployee(prev => ({ ...prev, emergencyContact: e.target.value }))}
                              />
                            </div>
                            <div className="space-y-2">
                              <Label htmlFor="emergencyPhone">紧急联系电话</Label>
                              <Input
                                id="emergencyPhone"
                                value={newEmployee.emergencyPhone}
                                onChange={(e) => setNewEmployee(prev => ({ ...prev, emergencyPhone: e.target.value }))}
                              />
                            </div>
                            <div className="space-y-2">
                              <Label htmlFor="email">邮箱</Label>
                              <Input
                                id="email"
                                type="email"
                                value={newEmployee.email}
                                onChange={(e) => setNewEmployee(prev => ({ ...prev, email: e.target.value }))}
                              />
                            </div>
                          </div>
                          <div className="space-y-2">
                            <Label htmlFor="currentAddress">现居住地址</Label>
                            <Input
                              id="currentAddress"
                              value={newEmployee.currentAddress}
                              onChange={(e) => setNewEmployee(prev => ({ ...prev, currentAddress: e.target.value }))}
                            />
                          </div>
                          </div>
                        </TabsContent>

                        {/* 其他信息 */}
                        <TabsContent value="other" className="space-y-4">
                          <div className="rounded-2xl border bg-muted/10 p-3 sm:p-4 space-y-4 shadow-sm">
                          <div className={RESPONSIVE_FIELD_GRID_CLASS}>
                            <div className="space-y-2">
                              <Label htmlFor="workClothingSize">工作服</Label>
                              <Select value={newEmployee.workClothingSize} onValueChange={(value) => setNewEmployee(prev => ({ ...prev, workClothingSize: value }))}>
                                <SelectTrigger>
                                  <SelectValue placeholder="选择工作服尺码" />
                                </SelectTrigger>
                                <SelectContent>
                                  <SelectItem value="S">S</SelectItem>
                                  <SelectItem value="M">M</SelectItem>
                                  <SelectItem value="L">L</SelectItem>
                                  <SelectItem value="XL">XL</SelectItem>
                                  <SelectItem value="XXL">XXL</SelectItem>
                                </SelectContent>
                              </Select>
                            </div>
                            <div className="space-y-2">
                              <Label htmlFor="safetyShoeSize">劳保鞋</Label>
                              <Select value={newEmployee.safetyShoeSize} onValueChange={(value) => setNewEmployee(prev => ({ ...prev, safetyShoeSize: value }))}>
                                <SelectTrigger>
                                  <SelectValue placeholder="选择劳保鞋尺码" />
                                </SelectTrigger>
                                <SelectContent>
                                  {Array.from({length: 15}, (_, i) => 35 + i).map(size => (
                                    <SelectItem key={size} value={size.toString()}>{size}</SelectItem>
                                  ))}
                                </SelectContent>
                              </Select>
                            </div>
                            <div className="space-y-2">
                            <Label htmlFor="insuranceStatus">参保状态</Label>
                            <Select
                              value={newEmployee.insuranceStatus?.trim() || derivedActiveStatusForNew}
                              onValueChange={(value) =>
                                setNewEmployee((prev) => ({
                                  ...prev,
                                  insuranceStatus: value,
                                }))
                              }
                            >
                              <SelectTrigger>
                                <SelectValue />
                              </SelectTrigger>
                              <SelectContent>
                                {ACTIVE_INSURANCE_STATUS_OPTIONS.map((option) => (
                                  <SelectItem key={option.value} value={option.value}>
                                    {option.label}
                                  </SelectItem>
                                ))}
                              </SelectContent>
                            </Select>
                            <p className="text-xs text-muted-foreground">默认将依据社保增加记录自动判断，可手动调整。</p>
                          </div>
                            <div className="space-y-2">
                              <Label htmlFor="graduateSchool">毕业院校</Label>
                              <SearchableSelect
                                id="graduateSchool"
                                value={newEmployee.graduateSchool || ""}
                                onChange={(value) => setNewEmployee(prev => ({ ...prev, graduateSchool: value }))}
                                options={graduateSchoolOptions}
                                placeholder="选择或输入毕业院校"
                              />
                            </div>
                            <div className="space-y-2">
                              <Label htmlFor="major">专业</Label>
                              <SearchableSelect
                                id="major"
                                value={newEmployee.major || ""}
                                onChange={(value) => setNewEmployee(prev => ({ ...prev, major: value }))}
                                options={majorOptions}
                                placeholder="选择或输入专业"
                              />
                            </div>
                            <div className="space-y-2">
                              <Label htmlFor="graduationTime">毕业时间</Label>
                              <Input
                                id="graduationTime"
                                value={newEmployee.graduationTime}
                                onChange={(e) => setNewEmployee(prev => ({ ...prev, graduationTime: e.target.value }))}
                              />
                            </div>
                            <div className="space-y-2">
                              <Label htmlFor="socialInsuranceNumber">社保编号</Label>
                              <Input
                                id="socialInsuranceNumber"
                                value={newEmployee.socialInsuranceNumber || ""}
                                onChange={(e) =>
                                  setNewEmployee((prev) => ({
                                    ...prev,
                                    socialInsuranceNumber: e.target.value,
                                  }))
                                }
                                placeholder="用于填充社保增减个人编号"
                              />
                            </div>
                            <div className="space-y-2">
                              <Label htmlFor="providentFundNumber">公积金编号</Label>
                              <Input
                                id="providentFundNumber"
                                value={newEmployee.providentFundNumber || ""}
                                onChange={(e) =>
                                  setNewEmployee((prev) => ({
                                    ...prev,
                                    providentFundNumber: e.target.value || "无",
                                  }))
                                }
                                placeholder="默认填写“无”"
                              />
                            </div>
                          </div>
                            <div className="space-y-2">
                              <Label htmlFor="remarks">备注</Label>
                              <Input
                                id="remarks"
                                value={newEmployee.remarks}
                                onChange={(e) => setNewEmployee(prev => ({ ...prev, remarks: e.target.value }))}
                              />
                            </div>
                          </div>
                        </TabsContent>
                      </Tabs>
                      <ScrollBar orientation="vertical" />
              </ScrollArea>
                          <DialogFooter className="flex flex-col gap-2 sm:flex-row sm:justify-end">
                            <Button variant="outline" onClick={() => setShowAddEmployee(false)}>
                              取消
                            </Button>
                            {/* P7.1：新增员工提交按钮与入口一致需 employee.create 权限 */}
                            <RequirePermission resource="employee" action="create">
                              <Button onClick={handleAddEmployee}>添加员工</Button>
                            </RequirePermission>
                          </DialogFooter>
                        </DialogContent>
                      </Dialog>
                      </RequirePermission>

                      {/* P7.1：批量导入员工需 employee.create 权限（RequirePermission 必须包裹在 Dialog 外层，
                          若放在 DialogTrigger asChild 内部会返回 Fragment，导致 Radix 无法注入 ref/onClick，点击导入无法打开弹窗） */}
                      <RequirePermission resource="employee" action="create">
                      <Dialog open={showBatchImport} onOpenChange={setShowBatchImport}>
                        <DialogTrigger asChild>
                          <Button variant="outline">
                            <Upload className="h-4 w-4 mr-2" />
                            导入
                          </Button>
                        </DialogTrigger>
                        <DialogContent>
                          <DialogHeader>
                            <DialogTitle>批量导入员工</DialogTitle>
                            <DialogDescription>
                              上传Excel文件批量导入员工信息
                            </DialogDescription>
                          </DialogHeader>
                          <div className="grid gap-4 py-4">
                            <div className="space-y-2">
                              <Label>选择文件</Label>
                              <Input
                                ref={fileInputRef}
                                type="file"
                                accept=".xlsx,.xls"
                              />
                              <p className="text-sm text-muted-foreground">
                                支持 .xlsx 和 .xls 格式文件
                              </p>
                            </div>
                            <div className="space-y-2">
                              <Label>导入模式</Label>
                              <div className="flex flex-col gap-2">
                                {IMPORT_MODE_OPTIONS.map((option) => {
                                  const active = importMode === option.value;
                                  return (
                                    <button
                                      key={option.value}
                                      type="button"
                                      onClick={() => setImportMode(option.value)}
                                      className={`rounded-md border px-3 py-2 text-left transition-colors ${
                                        active
                                          ? "border-primary bg-primary/10 text-primary"
                                          : "border-muted hover:border-primary/60"
                                      }`}
                                    >
                                      <div className="text-sm font-medium">{option.label}</div>
                                      <div className="text-xs text-muted-foreground">{option.description}</div>
                                    </button>
                                  );
                                })}
                              </div>
                            </div>
                            <Button
                              variant="outline"
                              onClick={downloadTemplate}
                              className="w-full"
                              disabled={downloadingTemplate}
                            >
                              <Download className="h-4 w-4 mr-2" />
                              {downloadingTemplate ? "下载中..." : "下载导入模板"}
                            </Button>
                          </div>
                          <DialogFooter className="flex flex-col gap-2 sm:flex-row sm:justify-end">
                            <Button variant="outline" onClick={() => setShowBatchImport(false)}>
                              取消
                            </Button>
                            <Button onClick={handleBatchImport} disabled={isLoading}>
                              {isLoading ? "导入中..." : "开始导入"}
                            </Button>
                          </DialogFooter>
                        </DialogContent>
                      </Dialog>
                      </RequirePermission>
                    </>
                  ) : (
                    <>
                      <DropdownMenu>
                        {/* P7.1：导出需 employee.view 权限（所有角色可见，仅作规范统一） */}
                        <RequirePermission resource="employee" action="view">
                          <DropdownMenuTrigger asChild>
                            <Button variant="outline" disabled={exporting}>
                              <Download className="h-4 w-4 mr-2" />
                              导出
                            </Button>
                          </DropdownMenuTrigger>
                        </RequirePermission>
                        <DropdownMenuContent align="end" className="w-44">
                          <DropdownMenuItem disabled={exporting} onClick={() => handleExportActive("selected")}>
                            导出选中数据
                          </DropdownMenuItem>
                          <DropdownMenuItem disabled={exporting} onClick={() => handleExportActive("filtered")}>
                            导出当前筛选
                          </DropdownMenuItem>
                          <DropdownMenuSeparator />
                          <DropdownMenuItem disabled={exporting} onClick={() => handleExportActive("all")}>
                            导出全部在职
                          </DropdownMenuItem>
                        </DropdownMenuContent>
                      </DropdownMenu>
                      <Button variant="outline" onClick={() => openPrintDialog("active")}>
                        <Printer className="h-4 w-4 mr-2" />
                        打印
                      </Button>
                      {/* P7.1：批量删除需 employee.delete 权限（页面级按钮，无权限隐藏） */}
                      <RequirePermission resource="employee" action="delete">
                        <Button
                          variant="destructive"
                          disabled={selectedEmployeeIds.length === 0 || deletingEmployees}
                          onClick={() => setShowBatchDeleteConfirm(true)}
                        >
                          <Trash2 className="h-4 w-4 mr-2" />
                          {deletingEmployees ? "删除中..." : "删除"}
                        </Button>
                      </RequirePermission>
                    </>
                  )}
                </div>
              </div>
            </CardHeader>
            <CardContent>
              {/* 搜索和筛选 */}
              <div className="flex flex-wrap items-center justify-between gap-4 mb-4">
                <div className="flex flex-1 flex-wrap items-center gap-4">
                  <div className="flex-1 min-w-[220px]">
                    <div className="relative">
                      <Search className="absolute left-3 top-3 h-4 w-4 text-muted-foreground" />
                      <Input
                        placeholder="搜索员工姓名、身份证号或部门..."
                        value={searchTerm}
                        onChange={(e) => setSearchTerm(e.target.value)}
                        className="pl-10 pr-10"
                      />
                      {searchTerm && (
                        <button
                          type="button"
                          onClick={() => setSearchTerm("")}
                          className="absolute right-2 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground"
                          aria-label="清除搜索"
                        >
                          <X className="h-4 w-4" />
                        </button>
                      )}
                    </div>
                  </div>
                  <Select value={departmentFilter} onValueChange={setDepartmentFilter}>
                    <SelectTrigger className="w-40">
                      <SelectValue placeholder="选择部门" />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="all">所有部门</SelectItem>
                      {getDepartments().map((dept) => (
                        <SelectItem key={dept} value={dept}>
                          {dept}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                  <Dialog open={showFieldSelector} onOpenChange={setShowFieldSelector}>
                    <DialogTrigger asChild>
                      <Button variant="outline" size="sm">
                        <Settings className="h-4 w-4 mr-2" />
                        显示字段
                      </Button>
                    </DialogTrigger>
                    <DialogContent className="max-w-md">
                      <DialogHeader>
                        <DialogTitle>自定义显示字段</DialogTitle>
                        <DialogDescription>
                          选择要在表格中显示的字段
                        </DialogDescription>
                      </DialogHeader>
                      <div className="space-y-4">
                        <div className="max-h-80 overflow-y-auto space-y-2">
                          {AVAILABLE_FIELDS.map((field) => (
                            <div key={field.key} className="flex items-center space-x-2">
                              <input
                                type="checkbox"
                                id={`field-${field.key}`}
                                checked={visibleFields.includes(field.key)}
                                onChange={() => handleFieldToggle(field.key)}
                                className="rounded border-gray-300"
                              />
                              <label
                                htmlFor={`field-${field.key}`}
                                className="text-sm font-medium leading-none peer-disabled:cursor-not-allowed peer-disabled:opacity-70"
                              >
                                {field.label}
                              </label>
                            </div>
                          ))}
                        </div>
                      </div>
                      <DialogFooter className="flex flex-col gap-2 sm:flex-row sm:justify-end">
                        <Button variant="outline" onClick={resetFieldsToDefault}>
                          恢复默认
                        </Button>
                        <Button onClick={() => setShowFieldSelector(false)}>
                          确定
                        </Button>
                      </DialogFooter>
                    </DialogContent>
                  </Dialog>
                </div>
                <div className="flex items-center gap-3 text-sm">
                  <span className="text-sm text-muted-foreground">
                    记录总数 {employees.length}
                  </span>
                  {filteredEmployees.length !== employees.length && (
                    <Badge variant="outline">当前筛选 {filteredEmployees.length}</Badge>
                  )}
                  {selectedEmployeeIds.length > 0 && (
                    <Badge variant="default">已选择 {selectedEmployeeIds.length}</Badge>
                  )}
                </div>
              </div>

              {/* 员工列表 */}
              <DataTableWrapper height="h-[65vh]">
                <Table className="min-w-full table-auto">
                  <TableHeader>
                    <TableRow>
                      <TableHead className="w-10">
                        <input
                          ref={selectAllRef}
                          type="checkbox"
                          className="rounded border-gray-300"
                          checked={sortedEmployees.length > 0 && allFilteredSelected}
                          onChange={(event) => toggleSelectAllFiltered(event.target.checked)}
                        />
                      </TableHead>
                      {visibleFieldConfigs.map((field) => {
                        const isSorted = employeeSort.key === field.key;
                        return (
                          <TableHead
                            key={field.key}
                            className="cursor-move select-none"
                            draggable
                            onDragStart={(event) => handleColumnDragStart(event, field.key)}
                            onDragOver={handleColumnDragOver}
                            onDrop={(event) => handleActiveColumnDrop(event, field.key)}
                            onDragEnd={handleColumnDragEnd}
                            onClick={() => handleEmployeeSortClick(field.key)}
                          >
                            <span className="flex items-center gap-1">
                              {field.label}
                              {isSorted && renderSortIndicator(employeeSort, field.key)}
                            </span>
                          </TableHead>
                        );
                      })}
                      <TableHead className="w-20 text-center">操作</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {sortedEmployees.length === 0 ? (
                      <TableRow>
                        <TableCell colSpan={visibleFields.length + 2} className="text-center py-8 text-muted-foreground">
                          暂无员工数据
                        </TableCell>
                      </TableRow>
                    ) : (
                      sortedEmployees.map((employee) => (
                          <TableRow
                            key={employee.id}
                            onDoubleClick={() => handleEditEmployee(employee)}
                            className="cursor-pointer hover:bg-muted/50"
                        >
                          <TableCell className="w-12">
                            <input
                              type="checkbox"
                              checked={selectedEmployeeIds.includes(employee.id)}
                              onChange={(event) => toggleEmployeeSelection(employee.id, event.target.checked)}
                              className="h-4 w-4 rounded border-muted-foreground"
                            />
                          </TableCell>
                          {visibleFieldConfigs.map((field) => {
                            const isInsuranceField = field.key === "insuranceStatus";
                            const node = isInsuranceField
                              ? renderInsuranceStatusBadge(getInsuranceStatusLabel(employee, "active"))
                              : formatFieldDisplayValue(employee, field.key);
                            return (
                              <TableCell key={field.key} className={getEmployeeCellClass(field.key)}>
                                {node}
                              </TableCell>
                            );
                          })}
                          <TableCell className="text-center">
                            <DropdownMenu>
                              <DropdownMenuTrigger asChild>
                                <Button variant="ghost" size="icon" className="h-8 w-8" aria-label="在职操作">
                                  <MoreHorizontal className="h-4 w-4" />
                                </Button>
                              </DropdownMenuTrigger>
                              <DropdownMenuContent align="end" className="w-40">
                                {/* P7.1：办理参保属于创建社保记录，需 insurance.create 权限 */}
                                <RequirePermission resource="insurance" action="create">
                                  <DropdownMenuItem
                                    onClick={(event) => {
                                      event.stopPropagation();
                                      openInsuranceForm("increase", { employee });
                                    }}
                                  >
                                    <CalendarPlus className="h-4 w-4 mr-2" />
                                    办理参保
                                  </DropdownMenuItem>
                                </RequirePermission>
                                {/* P7.1：办理公积金属于创建公积金记录，需 insurance.create 权限 */}
                                <RequirePermission resource="insurance" action="create">
                                  <DropdownMenuItem
                                    onClick={(event) => {
                                      event.stopPropagation();
                                      openProvidentDialog("create", { employee });
                                    }}
                                  >
                                    <PiggyBank className="h-4 w-4 mr-2" />
                                    办理公积金
                                  </DropdownMenuItem>
                                </RequirePermission>
                                {/* P7.1：办理离职为修改员工状态，需 employee.edit 权限 */}
                                <RequirePermission resource="employee" action="edit">
                                  <DropdownMenuItem
                                    onClick={(event) => {
                                      event.stopPropagation();
                                      setSelectedEmployee(employee);
                                      setShowResignDialog(true);
                                    }}
                                  >
                                    <UserMinus className="h-4 w-4 mr-2" />
                                    办理离职
                                  </DropdownMenuItem>
                                </RequirePermission>
                              </DropdownMenuContent>
                            </DropdownMenu>
                          </TableCell>
                        </TableRow>
                      ))
                    )}
                  </TableBody>
                </Table>
                <ScrollBar orientation="horizontal" />
              </DataTableWrapper>
            </CardContent>
          </Card>

          <Dialog open={showBatchDeleteConfirm} onOpenChange={setShowBatchDeleteConfirm}>
            <DialogContent>
              <DialogHeader>
                <DialogTitle>确认批量删除</DialogTitle>
                <DialogDescription>
                  将永久删除选中的 {selectedEmployeeIds.length} 名在职员工，操作不可恢复，是否继续？
                </DialogDescription>
              </DialogHeader>
              <DialogFooter className="flex flex-col gap-2 sm:flex-row sm:justify-end">
                <Button variant="outline" onClick={() => setShowBatchDeleteConfirm(false)}>
                  取消
                </Button>
                {/* P7.1：确认批量删除需 employee.delete 权限 */}
                <RequirePermission resource="employee" action="delete">
                  <Button variant="destructive" onClick={handleBatchDelete} disabled={deletingEmployees}>
                    {deletingEmployees ? "删除中..." : "确认删除"}
                  </Button>
                </RequirePermission>
              </DialogFooter>
            </DialogContent>
          </Dialog>
        </TabsContent>

        {/* 离职员工标签页 */}
        <TabsContent value="resigned" className="space-y-6">
          <Card>
            <CardHeader>
              <div className="flex flex-wrap items-center justify-between gap-4">
                <div>
                  <CardTitle>离职员工管理</CardTitle>
                  <CardDescription>
                    管理离职员工信息、退保状态与离职资料
                  </CardDescription>
                </div>
                <div className="flex flex-wrap items-center gap-3 text-sm">
                  <span className="text-sm text-muted-foreground">
                    记录总数 {resignedEmployees.length}
                  </span>
                  {filteredResignedEmployees.length !== resignedEmployees.length && (
                    <Badge variant="outline">筛选 {filteredResignedEmployees.length}</Badge>
                  )}
                  {selectedResignedIds.length > 0 && (
                    <Badge variant="default">已选择 {selectedResignedIds.length}</Badge>
                  )}
                  {selectedResignedIds.length === 0 ? (
                    <RequirePermission resource="employee" action="create">
                    {/* P7.1：批量导入离职员工需 employee.create 权限（RequirePermission 必须包裹在 Dialog 外层，
                        若放在 DialogTrigger asChild 内部会返回 Fragment，导致 Radix 无法注入 ref/onClick，点击导入无法打开弹窗） */}
                    <Dialog open={showResignedImport} onOpenChange={setShowResignedImport}>
                      <DialogTrigger asChild>
                        <Button variant="outline" size="sm">
                          <Upload className="h-4 w-4 mr-2" /> 导入
                        </Button>
                      </DialogTrigger>
                      <DialogContent>
                        <DialogHeader>
                          <DialogTitle>批量导入离职员工</DialogTitle>
                          <DialogDescription>上传 Excel 文件批量导入离职员工资料</DialogDescription>
                        </DialogHeader>
                        <div className="grid gap-4 py-4">
                          <div className="space-y-2">
                            <Label>选择文件</Label>
                            <Input ref={resignedFileInputRef} type="file" accept=".xlsx,.xls" />
                            <p className="text-sm text-muted-foreground">支持 .xlsx 和 .xls 格式文件</p>
                          </div>
                          <div className="space-y-2">
                            <Button
                              variant="outline"
                              className="w-full"
                              onClick={downloadResignedTemplate}
                              disabled={downloadingResignedTemplate}
                            >
                              <Download className="h-4 w-4 mr-2" />
                              {downloadingResignedTemplate ? "下载中..." : "下载导入模板"}
                            </Button>
                            <p className="text-xs text-muted-foreground">
                              如检测到身份证重复，会提示是否将对应在职员工迁移为离职状态。
                            </p>
                          </div>
                        </div>
                        <DialogFooter className="flex flex-col gap-2 sm:flex-row sm:justify-end">
                          <Button variant="outline" onClick={() => setShowResignedImport(false)}>
                            取消
                          </Button>
                          <Button onClick={handleResignedImport} disabled={resignedImporting}>
                            {resignedImporting ? "导入中..." : "开始导入"}
                          </Button>
                        </DialogFooter>
                      </DialogContent>
                    </Dialog>
                    </RequirePermission>
                  ) : (
                    <div className="flex items-center gap-2">
                      <DropdownMenu>
                        {/* P7.1：导出需 employee.view 权限 */}
                        <RequirePermission resource="employee" action="view">
                          <DropdownMenuTrigger asChild>
                            <Button variant="outline" size="sm" disabled={resignedExporting}>
                              <Download className="h-4 w-4 mr-2" /> 导出
                            </Button>
                          </DropdownMenuTrigger>
                        </RequirePermission>
                        <DropdownMenuContent align="end" className="w-44">
                          <DropdownMenuItem disabled={resignedExporting} onClick={() => handleExportResigned("selected")}>
                            导出选中数据
                          </DropdownMenuItem>
                          <DropdownMenuItem disabled={resignedExporting} onClick={() => handleExportResigned("filtered")}>
                            导出当前筛选
                          </DropdownMenuItem>
                          <DropdownMenuSeparator />
                          <DropdownMenuItem disabled={resignedExporting} onClick={() => handleExportResigned("all")}>
                            导出全部离职
                          </DropdownMenuItem>
                        </DropdownMenuContent>
                      </DropdownMenu>
                      <Button variant="outline" size="sm" onClick={() => openPrintDialog("resigned")}>
                        <Printer className="h-4 w-4 mr-2" /> 打印
                      </Button>
                      {/* P7.1：撤销离职（恢复在职）为修改员工状态，需 employee.edit 权限 */}
                      <RequirePermission resource="employee" action="edit">
                        <Button variant="destructive" size="sm" onClick={() => setShowBatchRestoreConfirm(true)}>
                          <CalendarPlus className="h-4 w-4 mr-2" /> 撤销
                        </Button>
                      </RequirePermission>
                    </div>
                  )}
                </div>
              </div>
            </CardHeader>
            <CardContent>
              <div className="mb-4 flex flex-wrap items-center gap-4">
                <div className="relative flex-1 min-w-[220px]">
                  <Search className="absolute left-3 top-3 h-4 w-4 text-muted-foreground" />
                  <Input
                    placeholder="搜索员工姓名、身份证号或部门..."
                    value={resignedSearchTerm}
                    onChange={(e) => setResignedSearchTerm(e.target.value)}
                    className="pl-10 pr-10"
                  />
                  {resignedSearchTerm && (
                    <button
                      type="button"
                      onClick={() => setResignedSearchTerm("")}
                      className="absolute right-2 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground"
                      aria-label="清除搜索"
                    >
                      <X className="h-4 w-4" />
                    </button>
                  )}
                </div>
                <Select value={resignedDepartmentFilter} onValueChange={setResignedDepartmentFilter}>
                  <SelectTrigger className="w-40">
                    <SelectValue placeholder="选择部门" />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="all">所有部门</SelectItem>
                    {getDepartments().map((dept) => (
                      <SelectItem key={dept} value={dept}>
                        {dept}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
                <Dialog open={showFieldSelector} onOpenChange={setShowFieldSelector}>
                  <DialogTrigger asChild>
                    <Button variant="outline" size="sm">
                      <Settings className="h-4 w-4 mr-2" />
                      显示字段
                    </Button>
                  </DialogTrigger>
                      <DialogContent className="max-w-md">
                    <DialogHeader>
                      <DialogTitle>自定义显示字段</DialogTitle>
                      <DialogDescription>
                        选择要在表格中显示的字段
                      </DialogDescription>
                    </DialogHeader>
                    <div className="space-y-4">
                      <div className="max-h-80 overflow-y-auto space-y-2">
                        {AVAILABLE_FIELDS.map((field) => (
                          <div key={field.key} className="flex items-center space-x-2">
                            <input
                              type="checkbox"
                              id={`resigned-field-${field.key}`}
                              checked={visibleFields.includes(field.key)}
                              onChange={() => handleFieldToggle(field.key)}
                              className="rounded border-gray-300"
                            />
                            <label
                              htmlFor={`resigned-field-${field.key}`}
                              className="text-sm font-medium leading-none peer-disabled:cursor-not-allowed peer-disabled:opacity-70"
                            >
                              {field.label}
                            </label>
                          </div>
                        ))}
                      </div>
                    </div>
                  <DialogFooter className="flex flex-col gap-2 sm:flex-row sm:justify-end">
                      <Button variant="outline" onClick={resetFieldsToDefault}>
                        恢复默认
                      </Button>
                      <Button onClick={() => setShowFieldSelector(false)}>
                        确定
                      </Button>
                    </DialogFooter>
                  </DialogContent>
                </Dialog>
              </div>

              {/* 离职员工列表 */}
              <DataTableWrapper height="h-[65vh]">
                <Table className="min-w-full table-auto text-sm">
                    <TableHeader>
                      <TableRow className="text-muted-foreground">
                        <TableHead className="w-12">
                          <input
                            type="checkbox"
                            className="h-4 w-4 rounded border-muted-foreground"
                            checked={sortedResignedEmployees.length > 0 && sortedResignedEmployees.every((emp) => selectedResignedIds.includes(emp.id))}
                            onChange={(event) => toggleAllResigned(event.target.checked)}
                            aria-label="选择全部离职员工"
                          />
                        </TableHead>
                        {resignedColumnsForRender.map((column) => (
                          <TableHead
                            key={column.id}
                            className="cursor-move select-none"
                            draggable
                            onDragStart={(event) => handleColumnDragStart(event, column.id)}
                            onDragOver={handleColumnDragOver}
                            onDrop={(event) => handleResignedColumnDrop(event, column.id)}
                            onDragEnd={handleColumnDragEnd}
                            onClick={column.sortable === false ? undefined : () => handleResignedSortClick(column.id as EmployeeColumnId)}
                          >
                            <span className="flex items-center gap-1">
                              {column.label}
                              {column.sortable !== false && renderSortIndicator(resignedSort, column.id as EmployeeColumnId)}
                            </span>
                          </TableHead>
                        ))}
                        <TableHead className="w-20 text-center">操作</TableHead>
                      </TableRow>
                    </TableHeader>
                    <TableBody>
                      {sortedResignedEmployees.length === 0 ? (
                        <TableRow>
                          <TableCell colSpan={resignedColumnsForRender.length + 2} className="py-8 text-center text-muted-foreground">
                            暂无离职员工数据
                          </TableCell>
                        </TableRow>
                      ) : (
                        sortedResignedEmployees.map((employee) => (
                          <TableRow
                            key={employee.id}
                            className="cursor-pointer hover:bg-muted/50"
                            onDoubleClick={() => handleOpenResignedDetail(employee, { mode: "edit" })}
                          >
                            <TableCell>
                              <input
                                type="checkbox"
                                className="h-4 w-4 rounded border-muted-foreground"
                                checked={selectedResignedIds.includes(employee.id)}
                                onChange={(event) => toggleResignedSelection(employee.id, event.target.checked)}
                              />
                            </TableCell>
                            {resignedColumnsForRender.map((column) => {
                              if (column.type === "base") {
                                const isInsuranceField = column.id === "insuranceStatus";
                                const displayValue = isInsuranceField
                                  ? renderInsuranceStatusBadge(getInsuranceStatusLabel(employee, "resigned"))
                                  : formatFieldDisplayValue(employee, column.id as keyof Employee);
                                return (
                                  <TableCell key={column.id} className={getEmployeeCellClass(column.id as keyof Employee)}>
                                    {displayValue}
                                  </TableCell>
                                );
                              }
                              return (
                                <TableCell key={column.id} className="text-sm">
                                  {renderResignedExtraCell(employee, column.id as typeof RESIGNED_EXTRA_COLUMNS[number])}
                                </TableCell>
                              );
                            })}
                          <TableCell className={ALIGNMENT_CLASS.center}>
                            <DropdownMenu>
                              <DropdownMenuTrigger asChild>
                                <Button
                                    variant="ghost"
                                    size="icon"
                                    className="h-8 w-8"
                                    onClick={(event) => event.stopPropagation()}
                                    aria-label="离职操作"
                                  >
                                    <MoreHorizontal className="h-4 w-4" />
                                  </Button>
                                </DropdownMenuTrigger>
                                <DropdownMenuContent align="end" className="w-40">
                                  {/* P7.1：办理退保为创建社保减少记录，需 insurance.create 权限 */}
                                  <RequirePermission resource="insurance" action="create">
                                    <DropdownMenuItem
                                      onClick={(event) => {
                                        event.stopPropagation();
                                        openInsuranceForm("decrease", { employee });
                                      }}
                                      className="gap-2"
                                    >
                                      <CalendarMinus className="h-4 w-4" />
                                      办理退保
                                    </DropdownMenuItem>
                                  </RequirePermission>
                                  <DropdownMenuItem
                                    onClick={(event) => {
                                      event.stopPropagation();
                                      handleOpenResignedDetail(employee, { mode: "detail" });
                                    }}
                                    className="gap-2"
                                  >
                                    <FileText className="h-4 w-4" />
                                    离职证明
                                  </DropdownMenuItem>
                                </DropdownMenuContent>
                              </DropdownMenu>
                            </TableCell>
                          </TableRow>
                        ))
                      )}
                    </TableBody>
                  </Table>
                  <ScrollBar orientation="horizontal" />
              </DataTableWrapper>
            </CardContent>
          </Card>

          <AlertDialog open={showResignedConflictDialog} onOpenChange={(open) => {
            if (resignedConflictImporting) return;
            setShowResignedConflictDialog(open);
            if (!open) {
              setPendingResignedImport(null);
              setResignedConflicts([]);
            }
          }}>
            <AlertDialogContent>
              <AlertDialogHeader>
                <AlertDialogTitle>发现 {resignedConflicts.length} 条在职数据与导入记录重复</AlertDialogTitle>
                <AlertDialogDescription>
                  这些员工当前仍在“在职员工”列表中，是否将其状态更新为离职并覆盖相关信息？
                </AlertDialogDescription>
              </AlertDialogHeader>
              <div className="max-h-64 overflow-y-auto rounded-md border">
                <table className="w-full text-sm">
                  <thead className="bg-muted text-xs uppercase text-muted-foreground">
                    <tr>
                      <th className="px-3 py-2 text-left">姓名</th>
                      <th className="px-3 py-2 text-left">证件号码</th>
                      <th className="px-3 py-2 text-left">部门</th>
                      <th className="px-3 py-2 text-left">状态</th>
                    </tr>
                  </thead>
                  <tbody>
                    {resignedConflicts.map((item) => (
                      <tr key={`${item.identity_number}-${item.id ?? item.name}`} className="border-t">
                        <td className="px-3 py-2">{item.name || "-"}</td>
                        <td className="px-3 py-2 text-xs text-muted-foreground">{item.identity_number}</td>
                        <td className="px-3 py-2">{item.department || "-"}</td>
                        <td className="px-3 py-2">{item.status || "active"}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
              <AlertDialogFooter className="flex flex-col gap-2 sm:flex-row sm:justify-end">
                <AlertDialogCancel
                  disabled={resignedConflictImporting}
                  onClick={handleCancelResignedConflicts}
                  className="h-9 px-4 text-sm"
                >
                  暂不处理
                </AlertDialogCancel>
                <AlertDialogAction
                  onClick={handleConfirmResignedConflicts}
                  disabled={resignedConflictImporting}
                  className="h-9 px-4 text-sm"
                >
                  {resignedConflictImporting ? "处理中..." : "继续并迁移"}
                </AlertDialogAction>
              </AlertDialogFooter>
            </AlertDialogContent>
          </AlertDialog>
        </TabsContent>

        <TabsContent value="insurance-increase" className="space-y-6">
          {renderInsuranceManagementCard("increase")}
        </TabsContent>

        <TabsContent value="insurance-decrease" className="space-y-6">
          {renderInsuranceManagementCard("decrease")}
        </TabsContent>

        <TabsContent value="provident" className="space-y-6">
          {renderProvidentManagementCard()}
        </TabsContent>
      </Tabs>

      <Dialog
        open={showEditEmployee}
        onOpenChange={(open) => {
          if (!open) {
            setShowEditEmployee(false);
            setEditEmployee({});
          }
        }}
      >
        <DialogContent className={RESPONSIVE_DIALOG_CLASS}>
          <DialogHeader>
            <DialogTitle>编辑员工信息</DialogTitle>
            <DialogDescription>
              参照新增员工的分组结构查看并修改所有字段，保存后立即生效。
            </DialogDescription>
          </DialogHeader>
          {!editEmployee.id ? (
            <p className="text-sm text-muted-foreground">请选择需要编辑的员工。</p>
          ) : (
            <ScrollArea className="max-h-[60vh] pr-2">
              <Tabs key={editEmployee.id} defaultValue="basic" className="w-full space-y-4">
                <TabsList className="grid w-full grid-cols-4">
                  <TabsTrigger value="basic">基本信息</TabsTrigger>
                  <TabsTrigger value="personal">个人信息</TabsTrigger>
                  <TabsTrigger value="contact">联系信息</TabsTrigger>
                  <TabsTrigger value="other">其他信息</TabsTrigger>
                </TabsList>

            <TabsContent value="basic" className="space-y-4">
              <div className="rounded-2xl border bg-muted/10 p-3 sm:p-4 shadow-sm">
                  <div className={RESPONSIVE_FIELD_GRID_CLASS}>
                    <div className="space-y-2">
                      <Label htmlFor="edit-employee-id">工号</Label>
                      <Input
                        id="edit-employee-id"
                        value={editEmployee.employeeId ?? ""}
                        onChange={(event) => updateEditEmployeeField("employeeId", event.target.value)}
                        placeholder="可填写或保持为空"
                      />
                    </div>
                    <div className="space-y-2">
                      <Label htmlFor="edit-name">姓名 *</Label>
                      <Input
                        id="edit-name"
                        value={editEmployee.name ?? ""}
                        onChange={(event) => updateEditEmployeeField("name", event.target.value)}
                        required
                      />
                    </div>
                    <div className="space-y-2">
                      <Label htmlFor="edit-department">部门 *</Label>
                      <SearchableSelect
                        id="edit-department"
                        value={editEmployee.department ?? ""}
                        onChange={(value) => updateEditEmployeeField("department", value)}
                        options={editDepartmentOptions}
                        placeholder="选择或输入部门"
                      />
                    </div>
                    <div className="space-y-2">
                      <Label htmlFor="edit-position">岗位</Label>
                      <SearchableSelect
                        id="edit-position"
                        value={editEmployee.position ?? ""}
                        onChange={(value) => updateEditEmployeeField("position", value)}
                        options={editPositionOptions}
                        placeholder="选择或输入岗位"
                      />
                    </div>
                    <div className="space-y-2">
                      <Label htmlFor="edit-gender">性别</Label>
                      <Select
                        value={editEmployee.gender ?? ""}
                        onValueChange={(value) => updateEditEmployeeField("gender", value)}
                      >
                        <SelectTrigger id="edit-gender">
                          <SelectValue placeholder="选择性别" />
                        </SelectTrigger>
                        <SelectContent>
                          <SelectItem value="男">男</SelectItem>
                          <SelectItem value="女">女</SelectItem>
                        </SelectContent>
                      </Select>
                    </div>
                    <div className="space-y-2">
                      <Label htmlFor="edit-hire-date">入职时间</Label>
                      <Input
                        id="edit-hire-date"
                        type="date"
                        value={editEmployee.hireDate ?? ""}
                        onChange={(event) => handleHireDateChange(event.target.value, true)}
                      />
                    </div>
                    <div className="space-y-2">
                      <Label htmlFor="edit-age">年龄</Label>
                      <Input
                        id="edit-age"
                        type="number"
                        value={editEmployee.age ?? ""}
                        onChange={(event) => updateEditEmployeeField("age", event.target.value)}
                        onBlur={(event) => handleAgeBlur(event.target.value, true)}
                      />
                    </div>
                    <div className="space-y-2">
                      <Label htmlFor="edit-work-years">工龄</Label>
                      <Input
                        id="edit-work-years"
                        inputMode="decimal"
                        value={editEmployee.workYears ?? ""}
                        onChange={(event) => updateEditEmployeeField("workYears", event.target.value)}
                        onBlur={(event) => handleWorkYearsBlur(event.target.value, true)}
                      />
                    </div>
                  </div>
                </div>
              </TabsContent>

            <TabsContent value="personal" className="space-y-4">
              <div className="rounded-2xl border bg-muted/10 p-3 sm:p-4 space-y-4 shadow-sm">
                  <div className={RESPONSIVE_FIELD_GRID_CLASS}>
                    <div className="space-y-2">
                      <Label htmlFor="edit-birth-month">出生月份</Label>
                      <Input
                        id="edit-birth-month"
                        type="date"
                        value={editEmployee.birthMonth ?? ""}
                        onChange={(event) => updateEditEmployeeField("birthMonth", event.target.value)}
                        onBlur={(event) =>
                          updateEditEmployeeField("birthMonth", normalizeBirthMonth(event.target.value))
                        }
                      />
                    </div>
                    <div className="space-y-2">
                      <Label htmlFor="edit-education">文化程度</Label>
                      <SearchableSelect
                        id="edit-education"
                        value={editEmployee.education ?? ""}
                        onChange={(value) => updateEditEmployeeField("education", value)}
                        options={educationOptions}
                        placeholder="选择或输入文化程度"
                      />
                    </div>
                    <div className="space-y-2">
                      <Label htmlFor="edit-political">政治面貌</Label>
                      <Select
                        value={editEmployee.politicalStatus ?? ""}
                        onValueChange={(value) => updateEditEmployeeField("politicalStatus", value)}
                      >
                        <SelectTrigger id="edit-political">
                          <SelectValue placeholder="选择政治面貌" />
                        </SelectTrigger>
                        <SelectContent>
                          <SelectItem value="群众">群众</SelectItem>
                          <SelectItem value="团员">团员</SelectItem>
                          <SelectItem value="党员">党员</SelectItem>
                          <SelectItem value="民主党派">民主党派</SelectItem>
                        </SelectContent>
                      </Select>
                    </div>
                    <div className="space-y-2">
                      <Label htmlFor="edit-ethnicity">民族</Label>
                      <Input
                        id="edit-ethnicity"
                        value={editEmployee.ethnicity ?? ""}
                        onChange={(event) => updateEditEmployeeField("ethnicity", event.target.value)}
                      />
                    </div>
                    <div className="space-y-2">
                      <Label htmlFor="edit-native-place">籍贯</Label>
                      <SearchableSelect
                        id="edit-native-place"
                        value={editEmployee.nativePlace ?? ""}
                        onChange={(value) => updateEditEmployeeField("nativePlace", value)}
                        options={nativePlaceOptions}
                        placeholder="选择或输入籍贯"
                      />
                    </div>
                    <div className="space-y-2">
                      <Label htmlFor="edit-marital-status">婚姻状况</Label>
                      <Select
                        value={editEmployee.maritalStatus ?? ""}
                        onValueChange={(value) => updateEditEmployeeField("maritalStatus", value)}
                      >
                        <SelectTrigger id="edit-marital-status">
                          <SelectValue placeholder="选择婚姻状况" />
                        </SelectTrigger>
                        <SelectContent>
                          <SelectItem value="未婚">未婚</SelectItem>
                          <SelectItem value="已婚">已婚</SelectItem>
                          <SelectItem value="离异">离异</SelectItem>
                          <SelectItem value="丧偶">丧偶</SelectItem>
                        </SelectContent>
                      </Select>
                    </div>
                    <div className="space-y-2">
                      <Label htmlFor="edit-household-type">户口性质</Label>
                      <Select
                        value={editEmployee.householdType ?? ""}
                        onValueChange={(value) => updateEditEmployeeField("householdType", value)}
                      >
                        <SelectTrigger id="edit-household-type">
                          <SelectValue placeholder="选择户口性质" />
                        </SelectTrigger>
                        <SelectContent>
                          <SelectItem value="城镇">城镇</SelectItem>
                          <SelectItem value="农村">农村</SelectItem>
                        </SelectContent>
                      </Select>
                    </div>
                    <div className="space-y-2">
                      <Label htmlFor="edit-has-birth">是否生育</Label>
                      <Select
                        value={editEmployee.hasBirth ?? ""}
                        onValueChange={(value) => updateEditEmployeeField("hasBirth", value)}
                      >
                        <SelectTrigger id="edit-has-birth">
                          <SelectValue placeholder="选择是否生育" />
                        </SelectTrigger>
                        <SelectContent>
                          <SelectItem value="是">是</SelectItem>
                          <SelectItem value="否">否</SelectItem>
                        </SelectContent>
                      </Select>
                    </div>
                  </div>
                  <div className="grid gap-4">
                    <div className="space-y-2">
                      <Label htmlFor="edit-id-address">身份证地址</Label>
                      <Input
                        id="edit-id-address"
                        value={editEmployee.idAddress ?? ""}
                        onChange={(event) => updateEditEmployeeField("idAddress", event.target.value)}
                      />
                    </div>
                    <div className="space-y-2">
                      <Label htmlFor="edit-id-number">身份证号码 *</Label>
                      <Input
                        id="edit-id-number"
                        value={editEmployee.idNumber ?? ""}
                        onChange={(event) => handleIdNumberChange(event.target.value, true)}
                        placeholder="自动计算年龄和出生月份"
                        required
                      />
                    </div>
                  </div>
                </div>
              </TabsContent>

            <TabsContent value="contact" className="space-y-4">
              <div className="rounded-2xl border bg-muted/10 p-3 sm:p-4 space-y-4 shadow-sm">
                  <div className={RESPONSIVE_FIELD_GRID_CLASS}>
                    <div className="space-y-2">
                      <Label htmlFor="edit-phone">联系电话</Label>
                      <Input
                        id="edit-phone"
                        value={editEmployee.phone ?? ""}
                        onChange={(event) => updateEditEmployeeField("phone", event.target.value)}
                      />
                    </div>
                    <div className="space-y-2">
                      <Label htmlFor="edit-emergency-contact">紧急联系人</Label>
                      <Input
                        id="edit-emergency-contact"
                        value={editEmployee.emergencyContact ?? ""}
                        onChange={(event) => updateEditEmployeeField("emergencyContact", event.target.value)}
                      />
                    </div>
                    <div className="space-y-2">
                      <Label htmlFor="edit-emergency-phone">紧急联系电话</Label>
                      <Input
                        id="edit-emergency-phone"
                        value={editEmployee.emergencyPhone ?? ""}
                        onChange={(event) => updateEditEmployeeField("emergencyPhone", event.target.value)}
                      />
                    </div>
                    <div className="space-y-2">
                      <Label htmlFor="edit-email">邮箱</Label>
                      <Input
                        id="edit-email"
                        type="email"
                        value={editEmployee.email ?? ""}
                        onChange={(event) => updateEditEmployeeField("email", event.target.value)}
                      />
                    </div>
                  </div>
                  <div className="space-y-2">
                    <Label htmlFor="edit-current-address">现居住地址</Label>
                    <Input
                      id="edit-current-address"
                      value={editEmployee.currentAddress ?? ""}
                      onChange={(event) => updateEditEmployeeField("currentAddress", event.target.value)}
                    />
                  </div>
                </div>
              </TabsContent>

            <TabsContent value="other" className="space-y-4">
              <div className="rounded-2xl border bg-muted/10 p-3 sm:p-4 space-y-4 shadow-sm">
                  <div className={RESPONSIVE_FIELD_GRID_CLASS}>
                    <div className="space-y-2">
                      <Label htmlFor="edit-work-clothing-size">工作服</Label>
                      <Select
                        value={editEmployee.workClothingSize ?? ""}
                        onValueChange={(value) => updateEditEmployeeField("workClothingSize", value)}
                      >
                        <SelectTrigger id="edit-work-clothing-size">
                          <SelectValue placeholder="选择工作服尺码" />
                        </SelectTrigger>
                        <SelectContent>
                          <SelectItem value="S">S</SelectItem>
                          <SelectItem value="M">M</SelectItem>
                          <SelectItem value="L">L</SelectItem>
                          <SelectItem value="XL">XL</SelectItem>
                          <SelectItem value="XXL">XXL</SelectItem>
                        </SelectContent>
                      </Select>
                    </div>
                    <div className="space-y-2">
                      <Label htmlFor="edit-safety-shoe-size">劳保鞋</Label>
                      <Select
                        value={editEmployee.safetyShoeSize ?? ""}
                        onValueChange={(value) => updateEditEmployeeField("safetyShoeSize", value)}
                      >
                        <SelectTrigger id="edit-safety-shoe-size">
                          <SelectValue placeholder="选择劳保鞋尺码" />
                        </SelectTrigger>
                        <SelectContent>
                          {Array.from({ length: 15 }, (_, index) => (35 + index).toString()).map((size) => (
                            <SelectItem key={size} value={size}>
                              {size}
                            </SelectItem>
                          ))}
                        </SelectContent>
                      </Select>
                    </div>
                    <div className="space-y-2">
                      <Label htmlFor="edit-insurance-status">参保状态</Label>
                      <Select
                        value={editEmployee.insuranceStatus?.trim() || derivedInsuranceStatusForEdit}
                        onValueChange={(value) => updateEditEmployeeField("insuranceStatus", value)}
                      >
                        <SelectTrigger id="edit-insurance-status">
                          <SelectValue placeholder="选择参保状态" />
                        </SelectTrigger>
                        <SelectContent>
                          {editInsuranceOptions.map((option) => (
                            <SelectItem key={option.value} value={option.value}>
                              {option.label}
                            </SelectItem>
                          ))}
                        </SelectContent>
                      </Select>
                      <p className="text-xs text-muted-foreground">{insuranceStatusHint}</p>
                    </div>
                    <div className="space-y-2">
                      <Label htmlFor="edit-graduate-school">毕业院校</Label>
                      <SearchableSelect
                        id="edit-graduate-school"
                        value={editEmployee.graduateSchool ?? ""}
                        onChange={(value) => updateEditEmployeeField("graduateSchool", value)}
                        options={graduateSchoolOptions}
                        placeholder="选择或输入毕业院校"
                      />
                    </div>
                    <div className="space-y-2">
                      <Label htmlFor="edit-major">专业</Label>
                      <SearchableSelect
                        id="edit-major"
                        value={editEmployee.major ?? ""}
                        onChange={(value) => updateEditEmployeeField("major", value)}
                        options={majorOptions}
                        placeholder="选择或输入专业"
                      />
                    </div>
                    <div className="space-y-2">
                      <Label htmlFor="edit-graduation-time">毕业时间</Label>
                      <Input
                        id="edit-graduation-time"
                        value={editEmployee.graduationTime ?? ""}
                        onChange={(event) => updateEditEmployeeField("graduationTime", event.target.value)}
                      />
                    </div>
                    <div className="space-y-2">
                      <Label htmlFor="edit-social-insurance-number">社保编号</Label>
                      <Input
                        id="edit-social-insurance-number"
                        value={editEmployee.socialInsuranceNumber ?? ""}
                        onChange={(event) => updateEditEmployeeField("socialInsuranceNumber", event.target.value)}
                        placeholder="与社保个人编号保持一致"
                      />
                    </div>
                    <div className="space-y-2">
                      <Label htmlFor="edit-provident-number">公积金编号</Label>
                      <Input
                        id="edit-provident-number"
                        value={editEmployee.providentFundNumber ?? "无"}
                        onChange={(event) =>
                          updateEditEmployeeField("providentFundNumber", event.target.value || "无")
                        }
                      />
                    </div>
                  </div>
                  <div className="space-y-2">
                    <Label htmlFor="edit-remarks">备注</Label>
                    <Textarea
                      id="edit-remarks"
                      value={editEmployee.remarks ?? ""}
                      onChange={(event) => updateEditEmployeeField("remarks", event.target.value)}
                      placeholder="可记录特殊说明、参保备注等"
                    />
                  </div>
                </div>
            </TabsContent>
          </Tabs>
          <ScrollBar orientation="vertical" />
        </ScrollArea>
          )}
          <DialogFooter className="flex flex-col gap-2 sm:flex-row sm:justify-end">
            <Button
              variant="outline"
              onClick={() => {
                setShowEditEmployee(false);
                setEditEmployee({});
              }}
            >
              取消
            </Button>
            {/* P7.1：编辑员工保存需 employee.edit 权限 */}
            <RequirePermission resource="employee" action="edit">
              <Button onClick={handleSaveEmployee} disabled={!editEmployee.id}>
                保存
              </Button>
            </RequirePermission>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Dialog
        open={showProvidentDialog}
        onOpenChange={(open) => {
          if (!open) {
            closeProvidentDialog();
          } else {
            setShowProvidentDialog(true);
          }
        }}
      >
        <DialogContent className={RESPONSIVE_DIALOG_CLASS}>
          <DialogHeader>
            <DialogTitle>{providentFormMode === "create" ? "新增公积金记录" : "公积金详情"}</DialogTitle>
            <DialogDescription>
              完整填写个人账号、证件号码等信息；金额字段支持小数，将自动保留两位小数。
            </DialogDescription>
          </DialogHeader>
          <div className={RESPONSIVE_FIELD_GRID_CLASS}>
            <div className="space-y-2">
              <Label>个人账号 *</Label>
              <Input
                value={providentForm.personal_account}
                onChange={(event) => handleProvidentFormChange("personal_account", event.target.value)}
                placeholder="请输入个人账号"
              />
            </div>
            <div className="space-y-2">
              <Label>姓名 *</Label>
              <Input value={providentForm.name} onChange={(event) => handleProvidentFormChange("name", event.target.value)} />
            </div>
            <div className="space-y-2">
              <Label>证件号码 *</Label>
              <Input
                value={providentForm.identity_number}
                onChange={(event) => handleProvidentFormChange("identity_number", event.target.value)}
                placeholder="请输入18位身份证号"
              />
            </div>
            <div className="space-y-2">
              <Label>缴存基数</Label>
              <Input
                type="number"
                inputMode="decimal"
                value={providentForm.personal_base}
                onChange={(event) => handleProvidentFormChange("personal_base", event.target.value)}
                placeholder="0.00"
              />
            </div>
            <div className="space-y-2">
              <Label>缴存比例（%）</Label>
              <Input
                type="number"
                inputMode="decimal"
                min="0"
                step="0.1"
                value={providentForm.contribution_ratio}
                onChange={(event) => handleProvidentFormChange("contribution_ratio", event.target.value)}
                placeholder={DEFAULT_PROVIDENT_RATIO.toString()}
              />
            </div>
            <div className="space-y-2">
              <Label>月应缴额（个人）</Label>
              <Input
                type="number"
                inputMode="decimal"
                value={providentForm.personal_amount}
                onChange={(event) => handleProvidentFormChange("personal_amount", event.target.value)}
                placeholder="0.00"
              />
            </div>
            <div className="space-y-2">
              <Label>月应缴额（单位）</Label>
              <Input
                type="number"
                inputMode="decimal"
                value={providentForm.company_amount}
                onChange={(event) => handleProvidentFormChange("company_amount", event.target.value)}
                placeholder="0.00"
              />
            </div>
            <div className="space-y-2 col-span-full">
              <Label>备注</Label>
              <Textarea
                value={providentForm.notes}
                onChange={(event) => handleProvidentFormChange("notes", event.target.value)}
                placeholder="可填写缴存比例、特殊说明等"
              />
            </div>
          </div>
          <DialogFooter className="flex flex-col gap-2 sm:flex-row sm:justify-end">
            <Button variant="outline" onClick={closeProvidentDialog} disabled={savingProvidentRecord}>
              取消
            </Button>
            {/* P7.1：公积金记录保存按模式区分：新增需 create、编辑需 edit */}
            <RequirePermission
              resource="insurance"
              action={providentFormMode === "create" ? "create" : "edit"}
            >
              <Button onClick={handleSaveProvidentRecord} disabled={savingProvidentRecord}>
                {savingProvidentRecord && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
                保存
              </Button>
            </RequirePermission>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Dialog open={!!sealDialogRecord} onOpenChange={(open) => (open ? undefined : setSealDialogRecord(null))}>
        <DialogContent className="max-w-lg">
          <DialogHeader>
            <DialogTitle>办理封存</DialogTitle>
            <DialogDescription>
              请选择封存日期，确认后该员工将无法参与后续账单汇缴。
            </DialogDescription>
          </DialogHeader>
          {sealDialogRecord && (
            <div className="space-y-4">
              <div className="rounded-lg border bg-muted/20 p-3 text-sm">
                <div>姓名：{sealDialogRecord.name}</div>
                <div>个人账号：{sealDialogRecord.personal_account}</div>
                <div>证件号码：{sealDialogRecord.identity_number}</div>
              </div>
              <div className="space-y-2">
                <Label>封存日期 *</Label>
                <Input type="date" value={sealDate} onChange={(event) => setSealDate(event.target.value)} />
              </div>
            </div>
          )}
          <DialogFooter className="flex flex-col gap-2 sm:flex-row sm:justify-end">
            <Button variant="outline" onClick={() => setSealDialogRecord(null)} disabled={sealSubmitting}>
              取消
            </Button>
            {/* P7.1：确认封存需 insurance.edit 权限 */}
            <RequirePermission resource="insurance" action="edit">
              <Button onClick={handleConfirmSeal} disabled={sealSubmitting}>
                {sealSubmitting && <Loader2 className="mr-2 h-4 w-4 animate-spin" />} 确认封存
              </Button>
            </RequirePermission>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Dialog open={!!unsealDialogRecord} onOpenChange={(open) => (open ? undefined : setUnsealDialogRecord(null))}>
        <DialogContent className="max-w-lg">
          <DialogHeader>
            <DialogTitle>办理启封</DialogTitle>
            <DialogDescription>确认启封后，该员工将重新进入在缴状态并参与账单汇缴。</DialogDescription>
          </DialogHeader>
          {unsealDialogRecord && (
            <div className="space-y-4">
              <div className="rounded-lg border bg-muted/20 p-3 text-sm">
                <div>姓名：{unsealDialogRecord.name}</div>
                <div>个人账号：{unsealDialogRecord.personal_account}</div>
                <div>证件号码：{unsealDialogRecord.identity_number}</div>
              </div>
              <div className="space-y-2">
                <Label>启封日期 *</Label>
                <Input type="date" value={unsealDate} onChange={(event) => setUnsealDate(event.target.value)} />
              </div>
            </div>
          )}
          <DialogFooter className="flex flex-col gap-2 sm:flex-row sm:justify-end">
            <Button variant="outline" onClick={() => setUnsealDialogRecord(null)} disabled={unsealSubmitting}>
              取消
            </Button>
            {/* P7.1：确认启封需 insurance.edit 权限 */}
            <RequirePermission resource="insurance" action="edit">
              <Button onClick={handleConfirmUnseal} disabled={unsealSubmitting}>
                {unsealSubmitting && <Loader2 className="mr-2 h-4 w-4 animate-spin" />} 确认启封
              </Button>
            </RequirePermission>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Dialog
        open={providentImportDialogOpen}
        onOpenChange={(open) => {
          setProvidentImportDialogOpen(open);
          if (!open && providentImportInputRef.current) {
            providentImportInputRef.current.value = "";
          }
          if (!open) {
            setSelectedProvidentFile(null);
            setProvidentImportError("");
            setImportedProvidentFileName("");
          }
        }}
      >
        <DialogContent className="max-w-lg">
          <DialogHeader>
            <DialogTitle>导入公积金记录</DialogTitle>
            <DialogDescription>请先下载模板并按列填写，导入前确认金额单位为元。</DialogDescription>
          </DialogHeader>
          <div className="space-y-4">
            <div className="space-y-2">
              <Label>选择导入文件（.xls/.xlsx）</Label>
              <Input
                ref={providentImportInputRef}
                type="file"
                accept=".xls,.xlsx"
                disabled={providentImporting}
                onChange={handleProvidentFileChange}
              />
              {selectedProvidentFile && (
                <p className="text-xs text-muted-foreground">已选择：{selectedProvidentFile.name}</p>
              )}
              {providentImportError && <p className="text-xs text-red-500">{providentImportError}</p>}
              <p className="text-xs text-muted-foreground">模板第一行必须包含：{PROVIDENT_TEMPLATE_HEADERS.join("、")}。</p>
            </div>
            <Button variant="outline" onClick={handleProvidentTemplateDownload} className="w-full">
              <Download className="mr-2 h-4 w-4" /> 下载导入模板
            </Button>
          </div>
          <DialogFooter className="flex flex-col gap-2 sm:flex-row sm:justify-end">
            <Button variant="outline" onClick={() => setProvidentImportDialogOpen(false)} disabled={providentImporting}>
              取消
            </Button>
            {/* P7.1：执行公积金导入需 insurance.create 权限 */}
            <RequirePermission resource="insurance" action="create">
              <Button onClick={handleExecuteProvidentImport} disabled={providentImporting}>
                {providentImporting && <Loader2 className="mr-2 h-4 w-4 animate-spin" />} 导入
              </Button>
            </RequirePermission>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Dialog
        open={showProvidentSettingsDialog}
        onOpenChange={(open) => {
          setShowProvidentSettingsDialog(open);
          if (open) {
            setProvidentSettingsDraft({
              unit_name: providentSettings?.unit_name || DEFAULT_PROVIDENT_UNIT_NAME,
              unit_account: providentSettings?.unit_account || DEFAULT_PROVIDENT_UNIT_ACCOUNT,
            });
          }
        }}
      >
        <DialogContent className="max-w-lg">
          <DialogHeader>
            <DialogTitle>单位设置</DialogTitle>
            <DialogDescription>单位信息将体现在导出的报表和账单抬头。</DialogDescription>
          </DialogHeader>
          <div className="space-y-4">
            <div className="space-y-2">
              <Label>单位名称</Label>
              <Input
                value={providentSettingsDraft.unit_name}
                onChange={(event) => setProvidentSettingsDraft((prev) => ({ ...prev, unit_name: event.target.value }))}
              />
            </div>
            <div className="space-y-2">
              <Label>单位公积金账号</Label>
              <Input
                value={providentSettingsDraft.unit_account}
                onChange={(event) => setProvidentSettingsDraft((prev) => ({ ...prev, unit_account: event.target.value }))}
              />
            </div>
          </div>
          <DialogFooter className="flex flex-col gap-2 sm:flex-row sm:justify-end">
            <Button variant="outline" onClick={() => setShowProvidentSettingsDialog(false)}>
              取消
            </Button>
            {/* P7.1：保存公积金单位设置需 insurance.edit 权限 */}
            <RequirePermission resource="insurance" action="edit">
              <Button onClick={handleProvidentSettingsSave}>保存</Button>
            </RequirePermission>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Dialog open={showBillDetailDialog} onOpenChange={(open) => {
        if (!open) {
          handleCloseBillDetail();
        } else {
          setShowBillDetailDialog(true);
        }
      }}>
        <DialogContent className={cn(DIALOG_SIZES.lg, "overflow-y-auto")}>
          <DialogHeader>
            <DialogTitle>账单详情</DialogTitle>
            <DialogDescription>查看该月份的人员名单与金额明细。</DialogDescription>
          </DialogHeader>
          {activeBill ? (
            <div className="space-y-4">
              <div className="grid gap-3 rounded-lg border bg-muted/20 p-3 text-xs text-muted-foreground sm:grid-cols-3 sm:text-sm">
                <div className="text-foreground font-semibold">账期：{activeBill.month_label}</div>
                <div>生成时间：{displayDate(activeBill.created_at)}</div>
                <div>人数：{activeBill.record_count}</div>
                <div>个人合计：{formatAmountValue(activeBill.personal_amount_total)}</div>
                <div>单位合计：{formatAmountValue(activeBill.company_amount_total)}</div>
                <div className="text-primary font-semibold">总计：{formatAmountValue(activeBill.combined_amount_total)}</div>
              </div>
              <div className="rounded-lg border">
                <div className="flex flex-wrap items-center justify-between gap-3 px-4 py-3 text-sm text-muted-foreground">
                  <span>
                    共 {filteredBillItems.length} 人
                    {activeBill.items && filteredBillItems.length !== activeBill.items.length && (
                      <span className="text-xs text-muted-foreground/80">（已筛选自 {activeBill.items.length} 人）</span>
                    )}
                  </span>
                  <Button
                    variant="default"
                    size="sm"
                    className="gap-1 bg-black text-white hover:bg-black/90"
                    onClick={handleExportBillItems}
                  >
                    <Download className="h-4 w-4" />
                    导出 Excel
                  </Button>
                </div>
                <div className="border-b bg-muted/20 px-4 py-3 text-xs text-muted-foreground">
                  显示 {filteredBillItems.length} 条（共 {activeBill.items?.length ?? 0} 条）
                </div>
                <div className="flex flex-wrap items-center gap-3 px-4 py-3">
                  <div className="relative flex-1 min-w-[220px]">
                    <Search className="pointer-events-none absolute left-3 top-3 h-4 w-4 text-muted-foreground" />
                    <Input
                      value={billDetailSearch}
                      onChange={(event) => setBillDetailSearch(event.target.value)}
                      placeholder="搜索个人账号、姓名或证件号"
                      className="pl-9"
                    />
                    {billDetailSearch && (
                      <button
                        type="button"
                        onClick={() => setBillDetailSearch("")}
                        className="absolute right-3 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground"
                        aria-label="清除搜索"
                      >
                        <X className="h-4 w-4" />
                      </button>
                    )}
                  </div>
                </div>
                <ScrollArea className="h-[40vh] min-h-[300px] max-h-[420px] rounded-b-lg">
                  <Table className="table-fixed">
                    <TableHeader>
                      <TableRow>
                        <TableHead className="w-[140px]">个人账号</TableHead>
                        <TableHead className="w-[120px]">姓名</TableHead>
                        <TableHead className="w-[180px]">证件号码</TableHead>
                        <TableHead className="text-right w-[100px]">个人</TableHead>
                        <TableHead className="text-right w-[100px]">单位</TableHead>
                        <TableHead className="text-right w-[110px]">合计</TableHead>
                      </TableRow>
                    </TableHeader>
                    <TableBody>
                      {filteredBillItems.length === 0 ? (
                        <TableRow>
                          <TableCell colSpan={6} className="py-6 text-center text-sm text-muted-foreground">
                            暂无账单明细
                          </TableCell>
                        </TableRow>
                      ) : (
                        <>
                          {filteredBillItems.map((item) => (
                            <TableRow key={item.id}>
                              <TableCell className="font-mono text-xs whitespace-nowrap">{item.personal_account}</TableCell>
                              <TableCell className="whitespace-nowrap">{item.name}</TableCell>
                              <TableCell className="font-mono text-xs whitespace-nowrap">{item.identity_number}</TableCell>
                              <TableCell className="text-right font-mono tabular-nums text-xs">{formatAmountValue(item.personal_amount)}</TableCell>
                              <TableCell className="text-right font-mono tabular-nums text-xs">{formatAmountValue(item.company_amount)}</TableCell>
                              <TableCell className="text-right font-mono tabular-nums text-xs">{formatAmountValue(item.total_amount)}</TableCell>
                            </TableRow>
                          ))}
                          {filteredBillItems.length > 0 && (
                            <TableRow className="border-t-2 border-primary bg-muted/50 font-semibold">
                              <TableCell>-</TableCell>
                              <TableCell>合计</TableCell>
                              <TableCell>-</TableCell>
                              <TableCell className="text-right font-mono tabular-nums text-xs">
                                {formatAmountValue(filteredBillItems.reduce((sum, item) => sum + (item.personal_amount ?? 0), 0))}
                              </TableCell>
                              <TableCell className="text-right font-mono tabular-nums text-xs">
                                {formatAmountValue(filteredBillItems.reduce((sum, item) => sum + (item.company_amount ?? 0), 0))}
                              </TableCell>
                              <TableCell className="text-right font-mono tabular-nums text-xs">
                                {formatAmountValue(filteredBillItems.reduce((sum, item) => sum + (item.total_amount ?? 0), 0))}
                              </TableCell>
                            </TableRow>
                          )}
                        </>
                      )}
                    </TableBody>
                  </Table>
                  <ScrollBar orientation="horizontal" />
                </ScrollArea>
              </div>
            </div>
          ) : (
            <p className="py-6 text-center text-sm text-muted-foreground">正在加载账单详情...</p>
          )}
          <DialogFooter className="flex flex-col gap-2 sm:flex-row sm:justify-end">
            <Button variant="outline" onClick={handleCloseBillDetail}>
              关闭
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Dialog
        open={showResignDialog}
        onOpenChange={(open) => {
          if (resignSubmitting) {
            return;
          }
          setShowResignDialog(open);
        }}
      >
        <DialogContent className={RESPONSIVE_DIALOG_CLASS}>
          <DialogHeader>
            <DialogTitle>员工离职</DialogTitle>
            <DialogDescription>确认将员工 {selectedEmployee?.name} 设置为离职状态。</DialogDescription>
          </DialogHeader>
          <div className="space-y-6 py-2">
            <div className="space-y-2">
              <Label htmlFor="resignDate">离职日期</Label>
              <Input
                id="resignDate"
                type="date"
                value={resignDate}
                onChange={(e) => setResignDate(e.target.value)}
                className="w-full sm:max-w-xs"
                disabled={resignSubmitting}
              />
            </div>
            <div className="space-y-3">
              <Label>离职原因</Label>
              <p className="text-xs text-muted-foreground">可多选，用于记录离职原因分析。</p>
              <div className="grid gap-2 sm:grid-cols-2">
                {RESIGN_REASON_OPTIONS.map((option) => (
                  <label
                    key={option.value}
                    className="flex items-start gap-2 rounded-md border border-border p-3 text-sm"
                  >
                    <input
                      type="checkbox"
                      className="mt-1 h-4 w-4 rounded border-gray-300"
                      checked={resignReasons.includes(option.value)}
                      onChange={(event) => handleResignReasonToggle(option.value, event.target.checked)}
                      disabled={resignSubmitting}
                    />
                    <div className="space-y-1">
                      <div className="font-medium text-foreground">{option.label}</div>
                      <p className="text-xs text-muted-foreground leading-relaxed">{option.description}</p>
                    </div>
                  </label>
                ))}
              </div>
            </div>
            <div className="space-y-2">
              <Label htmlFor="resignProof">离职证明</Label>
              <Input
                id="resignProof"
                type="file"
                accept=".pdf,image/*"
                onChange={handleResignProofChange}
                disabled={resignSubmitting}
              />
                {resignProofError && <p className="text-xs text-destructive">{resignProofError}</p>}
                <p className="text-xs text-muted-foreground">
                  （选填）支持 PDF 或图片格式，文件大小不超过 20MB。
                </p>
            </div>
          </div>
          <DialogFooter className="flex flex-col gap-2 sm:flex-row sm:justify-end">
            <Button variant="outline" onClick={() => setShowResignDialog(false)} disabled={resignSubmitting}>
              取消
            </Button>
            {/* P7.1：确认离职为修改员工状态，需 employee.edit 权限 */}
            <RequirePermission resource="employee" action="edit">
              <Button onClick={handleResignEmployee} disabled={resignSubmitting}>
                {resignSubmitting ? "提交中..." : "确认离职"}
              </Button>
            </RequirePermission>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Dialog
        open={showResignedDetail}
        onOpenChange={(open) => {
          if (!open) {
            handleCloseResignedDetail();
          }
        }}
      >
        <DialogContent className={RESPONSIVE_DIALOG_CLASS}>
          <DialogHeader>
            <DialogTitle>离职员工详情</DialogTitle>
            <DialogDescription>查看离职员工的详细信息与离职证明预览。</DialogDescription>
          </DialogHeader>
          {resignedDetailEmployee ? (
            <div className="space-y-6 py-2">
              <div className="grid gap-3 md:grid-cols-3">
                <DetailField label="姓名" value={resignedDetailEmployee.name} />
                <DetailField label="身份证号码" value={resignedDetailEmployee.idNumber} />
                <DetailField label="部门" value={resignedDetailEmployee.department || "-"} />
                <DetailField label="岗位" value={resignedDetailEmployee.position || "-"} />
                <DetailField label="离职日期" value={displayDate(resignedDetailEmployee.resignDate)} />
                <DetailField label="联系方式" value={resignedDetailEmployee.phone || "-"} />
              </div>
              <div className="space-y-3">
                <h4 className="text-sm font-semibold text-muted-foreground">离职证明</h4>
                <div className="rounded-lg border bg-muted/20 p-4 space-y-3">
                  {resignProofPreviewError && (
                    <p className="text-xs text-destructive">{resignProofPreviewError}</p>
                  )}
                  {resignProofPreviewLoading ? (
                    <div className="text-sm text-muted-foreground">加载预览中...</div>
                  ) : resignProofPreviewUrl ? (
                    resignProofPreviewType === "image" ? (
                      <div className="max-h-[360px] w-full overflow-hidden rounded-md border bg-card">
                        {/* eslint-disable-next-line @next/next/no-img-element */}
                        <img
                          src={resignProofPreviewUrl}
                          alt={resignProofPreviewFilename || "离职证明预览"}
                          className="h-full w-full object-contain"
                        />
                      </div>
                    ) : resignProofPreviewType === "pdf" ? (
                      <iframe
                        src={resignProofPreviewUrl}
                        title="离职证明预览"
                        className="h-[360px] w-full rounded-md border"
                      />
                    ) : (
                      <div className="text-center text-sm text-muted-foreground space-y-2">
                        <p>暂不支持直接预览该格式，请使用下载入口查看原文件。</p>
                        {resignProofPreviewFilename && (
                          <p className="text-xs">文件：{resignProofPreviewFilename}</p>
                        )}
                      </div>
                    )
                  ) : (
                    <span className="text-sm text-muted-foreground">尚未上传离职证明</span>
                  )}
                  {(resignProofPreviewFilename || resignProofPreviewSize !== null || resignProofPreviewContentType) && (
                    <div className="text-xs text-muted-foreground space-y-1">
                      {resignProofPreviewFilename && <div>文件名：{resignProofPreviewFilename}</div>}
                      {resignProofPreviewSize !== null && <div>文件大小：{formatFileSize(resignProofPreviewSize)}</div>}
                      {resignProofPreviewContentType && <div>文件类型：{resignProofPreviewContentType}</div>}
                    </div>
                  )}
                </div>
              </div>
            </div>
          ) : (
            <div className="py-6 text-center text-sm text-muted-foreground">
              未找到离职员工详情，请重新选择后再试。
            </div>
          )}
          <DialogFooter className="flex flex-col gap-2 sm:flex-row sm:justify-end">
            <Button variant="outline" onClick={handleCloseResignedDetail}>
              关闭
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Dialog
        open={showPrintDialog}
        onOpenChange={(open) => {
          if (!open) {
            handleClosePrintDialog();
          }
        }}
      >
        <DialogContent className="max-w-md">
          <DialogHeader>
            <DialogTitle>打印设置</DialogTitle>
            <DialogDescription>配置打印标题与水印后生成 PDF 预览。</DialogDescription>
          </DialogHeader>
          <div className="space-y-4 py-2">
            <div className="space-y-2">
              <Label htmlFor="printTitle">打印标题</Label>
              <Input
                id="printTitle"
                value={printTitle}
                onChange={(event) => setPrintTitle(event.target.value)}
                placeholder={printSuggestedTitle || "例如：在职员工打印清单"}
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="printWatermark">水印</Label>
              <Input
                id="printWatermark"
                value={printWatermark}
                onChange={(event) => setPrintWatermark(event.target.value)}
                placeholder="内部资料 请勿外传"
              />
              <p className="text-xs text-muted-foreground">为空将使用默认水印「内部资料 请勿外传」。</p>
            </div>
            <div className="space-y-2">
              <Label htmlFor="printOrientation">排版方向</Label>
              <Select value={printOrientation} onValueChange={(value) => setPrintOrientation(value as PrintOrientation)}>
                <SelectTrigger id="printOrientation">
                  <SelectValue placeholder="自动适配" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="auto">自动适配</SelectItem>
                  <SelectItem value="portrait">纵向（A4）</SelectItem>
                  <SelectItem value="landscape">横向（A4）</SelectItem>
                </SelectContent>
              </Select>
              <p className="text-xs text-muted-foreground">字段较多时建议选择横向排版，避免表格被裁切。</p>
            </div>
            <div className="rounded-md border border-dashed border-muted-foreground/50 bg-muted/20 p-3 text-xs leading-relaxed text-muted-foreground">
              {printContext ? (
                <>
                  <div>
                    打印范围：
                    {printContext.type === "active"
                      ? "在职员工"
                      : printContext.type === "resigned"
                        ? "离职员工"
                        : printContext.type === "provident"
                          ? "公积金记录"
                          : "社保增减记录"}
                  </div>
                  <div>数据量：{printContext.rows.length} 条</div>
                  <div>提示：生成的 PDF 将在新窗口打开，可直接打印或保存。</div>
                </>
              ) : (
                <div>未找到打印数据，请重新选择。</div>
              )}
            </div>
          </div>
          <DialogFooter className="flex flex-col gap-2 sm:flex-row sm:justify-end">
            <Button variant="outline" onClick={handleClosePrintDialog}>
              取消
            </Button>
            <Button onClick={handleGeneratePrint} disabled={!printContext || printContext.rows.length === 0}>
              生成预览
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Dialog open={showBatchRestoreConfirm} onOpenChange={setShowBatchRestoreConfirm}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>确认批量撤销离职</DialogTitle>
            <DialogDescription>将恢复选中的 {selectedResignedIds.length} 名员工至在职状态，是否继续？</DialogDescription>
          </DialogHeader>
          <DialogFooter className="flex flex-col gap-2 sm:flex-row sm:justify-end">
            <Button variant="outline" onClick={() => setShowBatchRestoreConfirm(false)}>
              取消
            </Button>
            {/* P7.1：批量恢复在职为修改员工状态，需 employee.edit 权限 */}
            <RequirePermission resource="employee" action="edit">
              <Button onClick={handleBatchRestore} disabled={restoringResigned}>
                {restoringResigned ? "处理中..." : "确认撤销"}
              </Button>
            </RequirePermission>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Dialog open={showBatchInsuranceConfirm} onOpenChange={setShowBatchInsuranceConfirm}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>确认批量撤销社保变动</DialogTitle>
            <DialogDescription>
              将删除选中的 {selectedInsuranceChangeIds.length} 条社保增减记录，操作不可恢复，是否继续？
            </DialogDescription>
          </DialogHeader>
          <DialogFooter className="flex flex-col gap-2 sm:flex-row sm:justify-end">
            <Button variant="outline" onClick={() => setShowBatchInsuranceConfirm(false)}>
              取消
            </Button>
            {/* P7.1：确认撤销社保记录需 insurance.delete 权限 */}
            <RequirePermission resource="insurance" action="delete">
              <Button variant="destructive" onClick={handleBatchInsuranceDelete}>确认撤销</Button>
            </RequirePermission>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Dialog open={unitInfoDialogOpen} onOpenChange={setUnitInfoDialogOpen}>
        <DialogContent className="max-w-md">
          <DialogHeader>
            <DialogTitle>配置单位信息</DialogTitle>
            <DialogDescription>设置导出模板中的单位社保编号与单位名称，便于批量导入申报表。</DialogDescription>
          </DialogHeader>
          <div className="space-y-4 py-2">
            <div className="space-y-2">
              <Label htmlFor="unit-social-code">单位社保编号</Label>
              <Input
                id="unit-social-code"
                value={unitInfoDraft.socialCode}
                onChange={(event) =>
                  setUnitInfoDraft((prev) => ({
                    ...prev,
                    socialCode: event.target.value,
                  }))
                }
                placeholder="例如：20302685"
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="unit-name">单位名称</Label>
              <Input
                id="unit-name"
                value={unitInfoDraft.unitName}
                onChange={(event) =>
                  setUnitInfoDraft((prev) => ({
                    ...prev,
                    unitName: event.target.value,
                  }))
                }
                placeholder="请输入单位全称"
              />
            </div>
          </div>
          <DialogFooter className="flex flex-col gap-2 sm:flex-row sm:justify-end">
            <Button variant="outline" onClick={() => setUnitInfoDialogOpen(false)}>
              取消
            </Button>
            <Button onClick={handleUnitInfoSave}>保存</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Dialog
        open={showInsuranceUploadDialog}
        onOpenChange={(open) => {
          if (open) {
            setShowInsuranceUploadDialog(true);
          } else {
            closeInsuranceUploadDialog();
          }
        }}
      >
        <DialogContent className={RESPONSIVE_DIALOG_CLASS}>
          <DialogHeader>
            <DialogTitle>导入名单</DialogTitle>
            <DialogDescription>
              {insuranceUploadType === "increase"
                ? "上传企业职工批量新参保人员名单"
                : "上传企业职工批量减少参保人员名单"}
            </DialogDescription>
          </DialogHeader>
          <div className="space-y-6 py-2">
            <div className="space-y-2">
              <Label>模板文件</Label>
              <Input
                ref={insuranceUploadInputRef}
                type="file"
                accept=".xls,.xlsx"
                onChange={handleInsuranceFileChange}
              />
              <p className="text-xs text-muted-foreground">支持官方模板（xls/xlsx），请确保字段填写完整并与模板列顺序一致。</p>
              {insuranceUploadError && <p className="text-xs text-destructive">{insuranceUploadError}</p>}
            </div>
            {insuranceUploadPreview.length > 0 && (
              <div className="space-y-3 rounded-lg border bg-muted/20 p-3 text-xs">
                <div className="flex items-center justify-between">
                  <span>已解析 {insuranceUploadPreview.length} 条记录，提交后将持久化保存。</span>
                  <span className="text-muted-foreground">仅展示前 5 条预览</span>
                </div>
                <div className="max-h-48 overflow-auto rounded border bg-card">
                  <table className="min-w-full text-left text-xs">
                    <thead className="bg-muted">
                      <tr>
                        <th className="px-2 py-1">姓名</th>
                        <th className="px-2 py-1">证件号码</th>
                        <th className="px-2 py-1">生效日期</th>
                        <th className="px-2 py-1">变动原因</th>
                      </tr>
                    </thead>
                    <tbody>
                      {insuranceUploadPreview.slice(0, 5).map((item, index) => (
                        <tr key={`${item.identity_number}-${index}`} className="border-t">
                          <td className="px-2 py-1">{item.employee_name || "-"}</td>
                          <td className="px-2 py-1">{item.identity_number || "-"}</td>
                          <td className="px-2 py-1">{displayDate(item.effective_date)}</td>
                          <td className="px-2 py-1">{item.reason || "-"}</td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              </div>
            )}
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={closeInsuranceUploadDialog}>
              取消
            </Button>
            <Button onClick={handleConfirmInsuranceUpload} disabled={insuranceImporting || !insuranceUploadFile}>
              {insuranceImporting ? "导入中..." : "确认导入"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Dialog
        open={showInsuranceFormDialog}
        onOpenChange={(open) => {
          if (!open) {
            closeInsuranceFormDialog();
          }
        }}
      >
        <DialogContent className={RESPONSIVE_DIALOG_CLASS}>
          <DialogHeader>
            <DialogTitle>{insuranceFormType === "increase" ? "新增社保" : "减少社保"}</DialogTitle>
            <DialogDescription>
              {insuranceFormEmployee ? `自动带入员工 ${insuranceFormEmployee.name} 的基础信息，补充必填字段后提交。` : "请选择需要操作的员工"}
            </DialogDescription>
          </DialogHeader>

          {insuranceFormError && (
            <p className="text-sm text-destructive">{insuranceFormError}</p>
          )}
          {socialOptionsError && (
            <p className="text-sm text-destructive">{socialOptionsError}</p>
          )}

          <ScrollArea className="max-h-[60vh] pr-2">
            <div className="space-y-8">
              <section className="space-y-3">
                <div className="flex flex-wrap items-center justify-between gap-2">
                  <p className="text-base font-semibold">员工基础信息</p>
                <p className="text-sm text-muted-foreground">
                  {insuranceFormEmployee ? "以下字段来自员工花名册，仅用于参照" : "请选择需要操作的员工"}
                </p>
              </div>
              <div className={RESPONSIVE_FIELD_GRID_CLASS}>
                <div className="flex flex-col gap-2 min-w-0">
                  <Label>姓名</Label>
                  <Input value={insuranceFormEmployee?.name ?? ""} disabled />
                </div>
                <div className="flex flex-col gap-2 min-w-0">
                  <Label>部门</Label>
                  <Input value={insuranceFormEmployee?.department ?? ""} disabled />
                </div>
                <div className="flex flex-col gap-2 min-w-0">
                  <Label>证件号码</Label>
                  <Input value={insuranceFormEmployee?.idNumber ?? ""} disabled />
                </div>
                <div className="flex flex-col gap-2 min-w-0">
                  <Label>岗位</Label>
                  <Input value={insuranceFormEmployee?.position ?? ""} disabled />
                </div>
              </div>
            </section>

              {insuranceFormType === "increase" ? (
                <div className="rounded-lg border bg-muted/10 p-3 sm:p-4">
                  <div className={RESPONSIVE_FIELD_GRID_CLASS}>
                    {INCREASE_FORM_FIELD_CONFIGS.map((config) => renderIncreaseField(config))}
                  </div>
                </div>
              ) : (
                <div className="rounded-lg border bg-muted/10 p-3 sm:p-4">
                  <div className={RESPONSIVE_FIELD_GRID_CLASS}>
                    {DECREASE_FORM_FIELD_CONFIGS.map((config) => renderDecreaseField(config))}
                  </div>
                </div>
              )}
            </div>
            <ScrollBar orientation="vertical" />
          </ScrollArea>

          <DialogFooter className="flex flex-col gap-2 sm:flex-row sm:justify-end">
            <Button variant="outline" onClick={closeInsuranceFormDialog} disabled={insuranceFormSubmitting}>
              取消
            </Button>
            <Button onClick={handleSubmitInsuranceForm} disabled={insuranceFormSubmitting || socialOptionsLoading}>
              {insuranceFormSubmitting ? "提交中..." : "提交"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <AlertDialog open={duplicateWarning !== null} onOpenChange={(open) => !open && setDuplicateWarning(null)}>
        <AlertDialogContent className={DIALOG_SIZES.sm}>
          <AlertDialogHeader>
            <AlertDialogTitle className="text-base font-semibold">
              {duplicateWarning?.type === "increase" ? "检测到重复的社保增加记录" : "检测到重复的社保减少记录"}
            </AlertDialogTitle>
            <AlertDialogDescription className="text-sm text-muted-foreground">
              {duplicateWarning
                ? `员工 ${duplicateWarning.name} 已存在对应记录，请前往社保${duplicateWarning.type === "increase" ? "增加" : "减少"}明细中进行编辑或撤销。`
                : ""}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter className="flex flex-row justify-end gap-2 sm:space-x-2">
            <AlertDialogCancel className="h-9 rounded-md px-4 text-sm" onClick={() => setDuplicateWarning(null)}>
              知道了
            </AlertDialogCancel>
            <AlertDialogAction
              className="h-9 rounded-md px-4 text-sm"
              onClick={() => {
                if (duplicateWarning) {
                  const targetTab = duplicateWarning.type === "increase" ? "insurance-increase" : "insurance-decrease";
                  handleRosterTabChange(targetTab);
                }
                setDuplicateWarning(null);
                setShowInsuranceFormDialog(false);
              }}
            >
              前往明细
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      <AlertDialog
        open={showBillPrecheckDialog}
        onOpenChange={(open) => {
          if (!open) {
            handleCancelPrecheckBill();
          }
        }}
      >
        <AlertDialogContent className={DIALOG_SIZES.sm}>
          <AlertDialogHeader>
            <AlertDialogTitle>生成账单前请确认封存/启封</AlertDialogTitle>
            <AlertDialogDescription className="text-sm text-muted-foreground">
              若公积金汇缴人员有变动，请先完成相应的封存或启封操作。系统只会依据当前处于“在缴”状态的人员生成账单。
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel onClick={handleCancelPrecheckBill}>取消</AlertDialogCancel>
            <AlertDialogAction onClick={handleConfirmPrecheckBill}>继续生成</AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      <AlertDialog
        open={showBillOverwriteDialog}
        onOpenChange={(open) => {
          if (!open) {
            handleCancelOverwriteBill();
          }
        }}
      >
        <AlertDialogContent className={DIALOG_SIZES.sm}>
          <AlertDialogHeader>
            <AlertDialogTitle>覆盖已生成的账单？</AlertDialogTitle>
            <AlertDialogDescription className="text-sm text-muted-foreground">
              {billOverwriteTarget
                ? `账期 ${billOverwriteTarget.month_label} 已存在账单，继续操作将删除旧数据并重新生成。`
                : "该账期已存在账单，继续操作将删除旧数据并重新生成。"}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel onClick={handleCancelOverwriteBill}>取消</AlertDialogCancel>
            <AlertDialogAction onClick={handleConfirmOverwriteBill}>覆盖生成</AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      <AlertDialog
        open={billDeleteTarget !== null}
        onOpenChange={(open) => {
          if (!open) {
            setBillDeleteTarget(null);
          }
        }}
      >
        <AlertDialogContent className={DIALOG_SIZES.sm}>
          <AlertDialogHeader>
            <AlertDialogTitle>删除公积金账单</AlertDialogTitle>
            <AlertDialogDescription className="text-sm text-muted-foreground">
              {billDeleteTarget
                ? `确认删除 ${billDeleteTarget.month_label} 账期的公积金账单？操作不可恢复。`
                : "确认删除所选账单？操作不可恢复。"}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel onClick={() => setBillDeleteTarget(null)}>取消</AlertDialogCancel>
            <AlertDialogAction className="bg-destructive text-destructive-foreground hover:bg-destructive/90" onClick={handleConfirmDeleteBill}>
              删除
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

    </PageTransition>
  );
}
const DEPARTMENT_CODES: Record<string, string> = {
  "总经办": "C020100",
  "财务部": "C020200",
  "销售部": "C020300",
  "仓库": "C020400",
  "生产部": "C020500",
  "机电部": "C020510",
  "技术质量部": "C020600",
  "人事行政部": "C020700",
};
const PREF_KEY_EMPLOYEE_COLUMNS = "employeeActiveColumns";
const PREF_KEY_RESIGNED_COLUMNS = "employeeResignedColumns";
const PREF_KEY_INSURANCE_COLUMNS = "employeeInsuranceColumns";
const PREF_KEY_PROVIDENT_COLUMNS = "employeeProvidentColumns";
const PREF_KEY_EMPLOYEE_VISIBLE_FIELDS = "employeeVisibleFields";
