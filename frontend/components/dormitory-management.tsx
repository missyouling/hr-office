"use client";

import { PageTransition } from "@/components/motion/page-transition";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import type { ChangeEvent } from "react";
import Image from "next/image";
import {
  Plus,
  Building2,
  BedDouble,
  FileText,
  Printer,
  Download,
  Search,
  X,
  Settings,
  MoreHorizontal,
  Upload,
  Eye,
  LogOut,
  Trash2,
  Home,
  MapPin,
  Gauge,
  PenSquare,
  RotateCcw,
  MessageCircle,
} from "lucide-react";
import { toast } from "sonner";
import * as XLSX from "xlsx";
import "xlsx/dist/cpexcel";

import {
  fetchDormSites,
  createDormSite,
  updateDormSite,
  deleteDormSite,
  fetchDormBuildings,
  createDormBuilding,
  fetchDormRooms,
  createDormRoom,
  updateDormRoom,
  deleteDormRoom,
  fetchDormContracts,
  createDormContract,
  updateDormContract,
  deleteDormContract,
  createDormCheckout,
  fetchDormBills,
  createDormBill,
  createDormBed,
  updateDormBuilding,
  fetchEmployees,
  fetchDormMeterRecords,
  createDormMeterRecord,
  updateDormMeterRecord,
  deleteDormMeterRecord,
  fetchUserPreferences,
  updateUserPreferences,
  type EmployeeResponse,
  type DormMeterRecordPayload,
} from "@/lib/api";
import {
  normalizeOrder,
  normalizeVisibility,
  parseColumnPreferenceValue,
  sanitizeSortPreference,
  type TableSortState,
} from "@/lib/preferences";
import type { DormSite, DormBuilding, DormRoom, DormBed, DormContract, DormBill, DormMeterRecord, DormChargeDetail, ChargeMode, DormChargeConfig, DormChargeRates, DormChargeRateEntry, DormCostBearingMode } from "@/lib/types";
import { createReportPdf } from "@/lib/pdf";
import { DormItemSelector, type DormItemSelectorValue } from "@/components/dorm-item-selector";

import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { RequirePermission } from "@/components/auth/RequirePermission";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle, DialogTrigger } from "@/components/ui/dialog";
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
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { ScrollArea, ScrollBar } from "@/components/ui/scroll-area";
import { Badge } from "@/components/ui/badge";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger } from "@/components/ui/dropdown-menu";
import { Checkbox } from "@/components/ui/checkbox";
import { RadioGroup, RadioGroupItem } from "@/components/ui/radio-group";
import { Switch } from "@/components/ui/switch";
import { cn } from "@/lib/utils";
import { DIALOG_SIZES } from "@/lib/dialog-sizes";
import { DataTableWrapper } from "@/components/common/data-table-wrapper";

const RESPONSIVE_DIALOG_CLASS = DIALOG_SIZES.full;
const RESPONSIVE_FIELD_GRID_CLASS = "grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3 2xl:grid-cols-4 [&>div]:min-w-0";
const DIALOG_SCROLL_CLASS = "flex-1 min-h-0 overflow-y-auto px-1";
const SITE_CARD_GRID_CLASS = "grid grid-cols-1 gap-4 sm:grid-cols-2 md:grid-cols-3 lg:grid-cols-4 2xl:grid-cols-5 auto-cols-fr";
const SITE_CARD_BG_CLASS = "bg-card";
const SITE_MEMO_STORAGE_KEY = "dorm_site_memos_v1";
const SITE_ORDER_STORAGE_KEY = "dorm_site_order_v1";
const SITE_HOUSE_STORAGE_KEY = "dorm_site_house_v1";
const SITE_CONTRACT_STORAGE_KEY = "dorm_site_contract_v1";
const ROOM_PRINT_SETTINGS_KEY = "dorm_room_print_settings_v1";
const CONTRACT_PRINT_SETTINGS_KEY = "dorm_contract_print_settings_v1";
const METER_PRINT_SETTINGS_KEY = "dorm_meter_print_settings_v1";
const ROOM_COLUMN_ORDER_STORAGE_KEY = "dorm_room_column_order_v1";
const ROOM_COLUMN_VISIBILITY_STORAGE_KEY = "dorm_room_column_visibility_v1";
const CONTRACT_COLUMN_ORDER_STORAGE_KEY = "dorm_contract_column_order_v1";
const CONTRACT_COLUMN_VISIBILITY_STORAGE_KEY = "dorm_contract_column_visibility_v1";
const METER_COLUMN_ORDER_STORAGE_KEY = "dorm_meter_column_order_v1";
const METER_COLUMN_VISIBILITY_STORAGE_KEY = "dorm_meter_column_visibility_v1";
const PREF_KEY_ROOM_COLUMNS = "dormRoomColumns";
const PREF_KEY_CONTRACT_COLUMNS = "dormContractColumns";
const PREF_KEY_METER_COLUMNS = "dormMeterColumns";
const PREF_KEY_ROOM_SORT = "dormRoomSort";
const PREF_KEY_CONTRACT_SORT = "dormContractSort";
const PREF_KEY_METER_SORT = "dormMeterSort";
const PREF_KEY_ROOM_FILTERS = "dormRoomFilters";
const PREF_KEY_CONTRACT_FILTERS = "dormContractFilters";
const PREF_KEY_METER_FILTERS = "dormMeterFilters";
const PREF_KEY_CHARGE_ITEMS = "dormChargeItems";
const PAYMENT_REMINDER_MEMO_PREFIX = "auto-payment-reminder";
const ROOM_CHARGE_REMINDER_PREFIX = "auto-room-charge";
const WATER_DEPENDENT_KEYS = ["trash", "water_supply", "sewage"];
const RENT_LIKE_CHARGE_LABELS: Record<string, string> = {
  rent: "房租费",
  property: "物业费",
};
const DEFAULT_EXTRA_CHARGE_TOGGLES: Record<string, boolean> = { water_supply: true };
const MEMBER_SHARE_CHARGE_KEYS = ["electric", "water", "gas", "trash", "water_supply", "sewage"] as const;
const MEMBER_CHARGE_LABELS: Record<(typeof MEMBER_SHARE_CHARGE_KEYS)[number], string> = {
  electric: "电费",
  water: "水费",
  gas: "气费",
  trash: "垃圾费",
  water_supply: "二次供水费",
  sewage: "污水处理费",
};
const SHARE_MODE_LABELS: Record<DormCostBearingMode, string> = {
  personal: "个人",
  company: "公司",
};
type ShareMode = DormCostBearingMode;

type MeterTableRow = {
  key: string;
  display: MeterRecord;
  source: MeterRecord;
};

const PAYMENT_REMINDER_SCHEDULE: Array<{ days: number; priority: SiteMemoPriority }> = [
  { days: 30, priority: "low" },
  { days: 20, priority: "normal" },
  { days: 10, priority: "urgent" },
];
const CONTRACT_REMINDER_MEMO_PREFIX = "auto-contract-reminder";
const CONTRACT_REMINDER_TARGETS: Array<{ field: "contractStartDate" | "contractEndDate"; label: string; key: string }> = [
  { field: "contractStartDate", label: "合同开始", key: "start" },
  { field: "contractEndDate", label: "合同结束", key: "end" },
];
const RENT_MEMO_RECURRENCE_BY_CYCLE: Record<PaymentCycle, SiteMemoRecurrence> = {
  monthly: "monthly",
  quarterly: "quarterly",
  semiannual: "semiannual",
  yearly: "annual",
};

type PrintOrientation = "auto" | "portrait" | "landscape";

type RoomOccupancyMember = {
  contractId?: number;
  name: string;
};

type RoomOccupancyMeta = {
  names: string[];
  count: number;
  bedAssignments: Record<number, string>;
  members: RoomOccupancyMember[];
};

type AttachmentEntry = {
  name: string;
  data: string;
};

type RoomPrintSettings = {
  title?: string;
  watermark?: string;
  orientation?: PrintOrientation;
};

type ContractPrintSettings = {
  title?: string;
  watermark?: string;
  orientation?: PrintOrientation;
};

type MeterPrintSettings = {
  title?: string;
  watermark?: string;
  orientation?: PrintOrientation;
};

type SiteHouseExtra = {
  propertyCompany: string;
  propertyContact: string;
  buildingNumber?: string;
  buildingCodeSnapshot?: string;
  inventoryItems?: SiteInventoryStoredItem[];
};

type PaymentCycle = "monthly" | "quarterly" | "semiannual" | "yearly";

type SiteContractExtra = {
  partyA: string;
  partyB: string;
  agentA: string;
  agentB: string;
  signingDate: string;
  contractStartDate: string;
  contractEndDate: string;
  paymentCycle?: PaymentCycle | "";
  lastPaymentDate: string;
  nextPaymentDate: string;
  notes?: string;
  paymentReminderEnabled?: boolean;
  contractReminderEnabled?: boolean;
  // legacy字段兼容
  contractTerm?: string;
  termStart?: string;
  termEnd?: string;
  firstPaymentDate?: string;
  reminderEnabled?: boolean; // 兼容旧字段
};

const PAYMENT_CYCLE_LABELS: Record<PaymentCycle, string> = {
  monthly: "月付",
  quarterly: "季付",
  semiannual: "半年付",
  yearly: "年付",
};
const PAYMENT_CYCLE_LABEL_TO_KEY = Object.entries(PAYMENT_CYCLE_LABELS).reduce<Record<string, PaymentCycle>>((acc, [key, label]) => {
  acc[label] = key as PaymentCycle;
  return acc;
}, {});
const PAYMENT_CYCLE_VALUES: PaymentCycle[] = ["monthly", "quarterly", "semiannual", "yearly"];

type SiteMemoPriority = "normal" | "urgent" | "low";

type SiteMemoRecurrence = "none" | "daily" | "weekly" | "monthly" | "quarterly" | "semiannual" | "annual";

const SITE_MEMO_RECURRENCE_LABELS: Record<SiteMemoRecurrence, string> = {
  none: "不重复",
  daily: "每天",
  weekly: "每周",
  monthly: "每月",
  quarterly: "每季度",
  semiannual: "每半年",
  annual: "每年",
};

type SiteMemoEntry = {
  id: string;
  date: string;
  time?: string;
  content: string;
  priority: SiteMemoPriority;
  createdAt: string;
  recurrence?: SiteMemoRecurrence;
  targetDate?: string;
  completed?: boolean;
  completedAt?: string;
};

const getMemoBaseDate = (memo: SiteMemoEntry) => {
  const baseDate = memo.date || memo.targetDate || memo.createdAt;
  if (!baseDate) return null;
  const timestamp = new Date(`${baseDate}T${memo.time || "00:00"}`);
  if (Number.isNaN(timestamp.getTime())) return null;
  return timestamp;
};

const advanceMemoDate = (date: Date, recurrence: SiteMemoRecurrence) => {
  const next = new Date(date);
  switch (recurrence) {
    case "daily":
      next.setDate(next.getDate() + 1);
      break;
    case "weekly":
      next.setDate(next.getDate() + 7);
      break;
    case "monthly":
      next.setMonth(next.getMonth() + 1);
      break;
    case "quarterly":
      next.setMonth(next.getMonth() + 3);
      break;
    case "semiannual":
      next.setMonth(next.getMonth() + 6);
      break;
    case "annual":
      next.setFullYear(next.getFullYear() + 1);
      break;
    default:
      next.setDate(next.getDate() + 1);
      break;
  }
  return next;
};

const getMemoTargetDate = (memo: SiteMemoEntry) => memo.targetDate || memo.date || memo.createdAt;

const getNextMemoOccurrence = (memo: SiteMemoEntry, fromDate = new Date()) => {
  const base = getMemoBaseDate(memo);
  if (!base) return null;
  const recurrence = memo.recurrence ?? "none";
  if (recurrence === "none") {
    return base;
  }
  const target = new Date(fromDate);
  const candidate = new Date(base);
  let guard = 0;
  while (candidate.getTime() < target.getTime() && guard < 200) {
    candidate.setTime(advanceMemoDate(candidate, recurrence).getTime());
    guard += 1;
  }
  return candidate;
};

const isMemoExpired = (memo: SiteMemoEntry) => {
  const nextOccurrence = getNextMemoOccurrence(memo);
  if (!nextOccurrence) return false;
  const today = new Date();
  today.setHours(0, 0, 0, 0);
  return nextOccurrence.getTime() < today.getTime();
};

const getMemoDisplayDate = (memo: SiteMemoEntry) => {
  const occurrence = getNextMemoOccurrence(memo);
  if (!occurrence) {
    return { date: memo.date || memo.targetDate || "", time: memo.time || "" };
  }
  return {
    date: formatLocalDateString(occurrence),
    time: formatLocalTimeString(occurrence),
  };
};

const AUTO_MEMO_PREFIXES = [PAYMENT_REMINDER_MEMO_PREFIX, CONTRACT_REMINDER_MEMO_PREFIX, ROOM_CHARGE_REMINDER_PREFIX];

const isAutoGeneratedMemo = (memo: SiteMemoEntry) => AUTO_MEMO_PREFIXES.some((prefix) => memo.id.startsWith(prefix));

const extractAutoMemoThreshold = (memo: SiteMemoEntry) => {
  if (!isAutoGeneratedMemo(memo)) return null;
  const segments = memo.id.split("-");
  if (segments.length === 0) return null;
  const candidate = Number(segments[segments.length - 1]);
  return Number.isFinite(candidate) ? candidate : null;
};

const deriveAutoMemoGroupKey = (memo: SiteMemoEntry) => {
  if (!isAutoGeneratedMemo(memo)) return memo.id;
  const segments = memo.id.split("-");
  if (segments.length <= 1) return memo.id;
  segments.pop();
  return segments.join("-");
};

const selectPrimaryAutoMemos = (memos: SiteMemoEntry[]) => {
  const manual: SiteMemoEntry[] = [];
  const grouped = new Map<string, SiteMemoEntry[]>();
  memos.forEach((memo) => {
    if (!memo) return;
    if (!isAutoGeneratedMemo(memo)) {
      manual.push(memo);
      return;
    }
    const key = deriveAutoMemoGroupKey(memo);
    const list = grouped.get(key) ?? [];
    list.push(memo);
    grouped.set(key, list);
  });
  grouped.forEach((list) => {
    const targetDate = getMemoTargetDate(list[0]);
    const diffDays = targetDate ? calculateDaysUntil(targetDate) : null;
    if (diffDays == null || diffDays < 0 || diffDays > 30) {
      return;
    }
    const sortedByThreshold = list
      .map((memo) => ({
        memo,
        threshold: extractAutoMemoThreshold(memo) ?? Number.MAX_SAFE_INTEGER,
      }))
      .sort((a, b) => a.threshold - b.threshold);
    const candidate = sortedByThreshold.find((entry) => diffDays <= entry.threshold) ?? sortedByThreshold[sortedByThreshold.length - 1];
    if (candidate?.memo) {
      manual.push(candidate.memo);
    }
  });
  return manual;
};

const shouldDisplayMemoNow = (memo: SiteMemoEntry) => {
  if (!isAutoGeneratedMemo(memo)) return true;
  const targetDate = getMemoTargetDate(memo);
  const diffDays = targetDate ? calculateDaysUntil(targetDate) : null;
  return diffDays != null && diffDays >= 0 && diffDays <= 30;
};

const formatRemainingDaysLabel = (targetDate?: string | null) => {
  if (!targetDate) return "剩余 -- 天";
  const today = new Date();
  today.setHours(0, 0, 0, 0);
  const target = new Date(targetDate);
  if (Number.isNaN(target.getTime())) return "剩余 -- 天";
  target.setHours(0, 0, 0, 0);
  const diffMs = target.getTime() - today.getTime();
  const diffDays = Math.max(0, Math.round(diffMs / (1000 * 60 * 60 * 24)));
  return `剩余 ${diffDays.toString().padStart(2, "0")} 天`;
};


type RoomColumnHelpers = { building?: DormBuilding; site?: DormSite; occupancy: RoomOccupancyMeta; chargeRates: Map<string, RoomChargeRateMeta> };

type Alignment = "left" | "center" | "right";

const ALIGNMENT_CLASS: Record<Alignment, string> = {
  left: "text-left",
  center: "text-center",
  right: "text-right",
};

type RoomColumnConfig = {
  id: string;
  label: string;
  sortable?: boolean;
  defaultVisible?: boolean;
  numeric?: boolean;
  align?: Alignment;
  width?: string;
  render: (room: DormRoom, helpers: RoomColumnHelpers) => React.ReactNode;
  getValue?: (room: DormRoom, helpers: RoomColumnHelpers) => string | number | null | undefined;
};

type ContractColumnHelpers = {
  getBuildingName: (contract: DormContract) => string;
  getNoteMeta: (contract: DormContract) => ContractNoteMeta;
};

type ContractColumnConfig = {
  id: string;
  label: string;
  sortable?: boolean;
  defaultVisible?: boolean;
  numeric?: boolean;
  align?: Alignment;
  width?: string;
  render: (contract: DormContract, helpers?: ContractColumnHelpers) => React.ReactNode;
  getValue?: (contract: DormContract, helpers?: ContractColumnHelpers) => string | number | null | undefined;
};

type ContractNoteMeta = {
  jobNumber?: string;
  position?: string;
  emergencyContact?: string;
  waterStart?: string;
  electricStart?: string;
  gasStart?: string;
  depositPlanMonth?: string;
  rentPlanMonth?: string;
  depositShareMode?: "personal" | "company";
  pledgePlanMonth?: string;
  pledgeShareMode?: "personal" | "company";
  pledgeAmount?: string;
  additionalNotes?: string;
};

const CONTRACT_NOTE_SEGMENTS: Array<{ key: keyof ContractNoteMeta; label: string }> = [
  { key: "jobNumber", label: "工号：" },
  { key: "position", label: "岗位：" },
  { key: "emergencyContact", label: "紧急联系人：" },
  { key: "waterStart", label: "水表起始：" },
  { key: "electricStart", label: "电表起始：" },
  { key: "gasStart", label: "气表起始：" },
  { key: "depositPlanMonth", label: "押金月份：" },
  { key: "rentPlanMonth", label: "租金月份：" },
  { key: "depositShareMode", label: "押金承担：" },
  { key: "pledgePlanMonth", label: "保证金月份：" },
  { key: "pledgeShareMode", label: "保证金承担：" },
  { key: "pledgeAmount", label: "保证金金额：" },
];

const parseContractNoteMeta = (raw?: string): ContractNoteMeta => {
  if (!raw?.trim()) {
    return { additionalNotes: "" };
  }
  const meta: ContractNoteMeta = { additionalNotes: "" };
  const rest: string[] = [];
  raw
    .split("|")
    .map((segment) => segment.trim())
    .filter(Boolean)
    .forEach((segment) => {
      const matched = CONTRACT_NOTE_SEGMENTS.find((item) => segment.startsWith(item.label));
      if (matched) {
        const value = segment.slice(matched.label.length).trim();
        if (value) {
          if (matched.key === "depositShareMode" || matched.key === "pledgeShareMode") {
            (meta as Record<string, ShareMode>)[matched.key] = value === SHARE_MODE_LABELS.company ? "company" : "personal";
          } else {
            (meta as Record<string, string>)[matched.key] = value;
          }
        }
      } else {
        rest.push(segment);
      }
    });
  meta.additionalNotes = rest.join(" | ");
  return meta;
};

const formatMonthDisplay = (value?: string, emptyLabel = "—") => {
  if (!value) return emptyLabel;
  const [year, month] = value.split("-");
  if (year && month) {
    return `${year}年${month}月`;
  }
  return value;
};

const ROOM_STATUS_OPTIONS = ["空闲", "缺人", "满员", "维护中"];
const SELECT_EMPTY_VALUE = "__empty__";

const formatPeriodLabel = (key: string) => {
  if (!key || key === "all") return "全部账期";
  const [year, month] = key.split("-");
  if (!year || !month) return key;
  return `${year}年${month}月`;
};

const derivePeriodKeyFromDate = (value?: string | null) => {
  if (!value) return "";
  return value.slice(0, 7);
};

const derivePeriodRangeFromKey = (key?: string | null) => {
  if (!key) return null;
  const [yearPart, monthPart] = key.split("-");
  const year = Number(yearPart);
  const month = Number(monthPart);
  if (!Number.isFinite(year) || !Number.isFinite(month) || month < 1 || month > 12) {
    return null;
  }
  const startDate = new Date(Date.UTC(year, month - 1, 1));
  const endDate = new Date(Date.UTC(year, month, 0));
  const formatDate = (value: Date) => value.toISOString().slice(0, 10);
  return {
    start: formatDate(startDate),
    end: formatDate(endDate),
  };
};

type SiteInventoryStoredItem = {
  key: string;
  quantity: number;
};

type SiteInventoryConfigItem = {
  key: string;
  label: string;
  unit: string;
  enabled: boolean;
  quantity: string;
};

const SITE_INVENTORY_OPTIONS: Array<{ key: string; label: string; unit: string }> = [
  { key: "clothes_rod", label: "晾衣杆", unit: "根" },
  { key: "towel_rod", label: "毛巾杆", unit: "根" },
  { key: "wardrobe", label: "衣柜", unit: "个" },
  { key: "ac_remote", label: "空调遥控器", unit: "个" },
  { key: "bedside_table", label: "床头柜", unit: "个" },
  { key: "bed_cabinet", label: "床柜", unit: "个" },
  { key: "iron_cabinet", label: "铁皮柜", unit: "个" },
  { key: "air_conditioner", label: "空调", unit: "台" },
  { key: "water_heater", label: "电热水器", unit: "台" },
  { key: "gas_water_heater", label: "燃气热水器", unit: "台" },
  { key: "single_bed", label: "单人床", unit: "张" },
  { key: "bunk_bed", label: "高低床", unit: "张" },
  { key: "mattress", label: "棕垫", unit: "张" },
  { key: "keys", label: "钥匙", unit: "把" },
  { key: "curtain", label: "窗帘", unit: "套" },
];

const SITE_INVENTORY_CATEGORY_PRESETS: Array<{ name: string; keys: string[] }> = [
  {
    name: "家具类",
    keys: ["clothes_rod", "towel_rod", "wardrobe", "bedside_table", "bed_cabinet", "iron_cabinet", "single_bed", "bunk_bed"],
  },
  {
    name: "电器类",
    keys: ["ac_remote", "air_conditioner", "water_heater", "gas_water_heater"],
  },
  {
    name: "床上用品类",
    keys: ["mattress", "keys", "curtain"],
  },
];

const buildInventoryCategories = () => {
  const optionMap = new Map(SITE_INVENTORY_OPTIONS.map((item) => [item.key, item]));
  const usedKeys = new Set<string>();
  const categories = SITE_INVENTORY_CATEGORY_PRESETS.map((preset) => {
    const items = preset.keys
      .map((key) => {
        const option = optionMap.get(key);
        if (option) {
          usedKeys.add(key);
        }
        return option || null;
      })
      .filter(Boolean) as Array<{ key: string; label: string; unit: string }>;
    return { name: preset.name, items };
  }).filter((category) => category.items.length > 0);
  const leftovers = SITE_INVENTORY_OPTIONS.filter((item) => !usedKeys.has(item.key));
  if (leftovers.length > 0) {
    categories.push({ name: "其他物品", items: leftovers });
  }
  return categories;
};

const SITE_INVENTORY_CATEGORY_DATA = buildInventoryCategories();
const CHARGE_GROUPS: Array<{ title: string; keys: string[] }> = [
  { title: "基础费用", keys: ["electric", "water", "gas", "rent", "property"] },
  { title: "附加费用", keys: ["deposit", "pledge", "trash", "water_supply", "sewage", "bonus", "penalty"] },
  { title: "付款周期", keys: ["cycle_monthly", "cycle_quarterly", "cycle_semiannual", "cycle_yearly"] },
];

const buildMeterTemplateHeaders = () => {
  const headers = ["楼栋", "房号", "抄表日期", "账单起始", "账单截止"];
  const meterItems = AVAILABLE_CHARGE_ITEMS.filter((item) => isBaseChargeKey(item.key as LegacyChargeKey));
  meterItems.forEach((item) => {
    if (item.key === "gas") {
      headers.push(`${item.label}金额`);
    } else {
      headers.push(`${item.label}起度`, `${item.label}止度`);
    }
  });
  headers.push("抄表人", "入住人员");
  AVAILABLE_CHARGE_ITEMS.filter((item) => !isBaseChargeKey(item.key as LegacyChargeKey)).forEach((item) => {
    headers.push(`${item.label}金额`);
  });
  return headers;
};

type LegacyChargeKey = "electric" | "water" | "gas";

const LEGACY_CHARGE_LABELS: Record<LegacyChargeKey, string> = {
  electric: "电费",
  water: "水费",
  gas: "气费",
};
const BASE_METER_KEYS: LegacyChargeKey[] = ["electric", "water", "gas"];
const isBaseChargeKey = (key: string): key is LegacyChargeKey => BASE_METER_KEYS.includes(key as LegacyChargeKey);
const MANDATORY_CHARGE_KEYS: LegacyChargeKey[] = ["electric", "water"];
const isMandatoryChargeKey = (key: string) => MANDATORY_CHARGE_KEYS.includes(key as LegacyChargeKey);

type ChargeDefinition = {
  key: string;
  label: string;
  unitLabel: string;
  defaultEnabled?: boolean;
  defaultUnitPrice?: number;
  mode: ChargeMode;
  cycleValue?: PaymentCycle;
};

const AVAILABLE_CHARGE_ITEMS: ChargeDefinition[] = [
  { key: "electric", label: "电费", unitLabel: "元/kWh", defaultEnabled: true, mode: "meter" },
  { key: "water", label: "水费", unitLabel: "元/m³", defaultEnabled: true, mode: "meter" },
  { key: "gas", label: "气费", unitLabel: "元/m³", defaultEnabled: false, mode: "meter" },
  { key: "rent", label: "房租费", unitLabel: "元", mode: "fixed" },
  { key: "property", label: "物业费", unitLabel: "元", mode: "fixed" },
  { key: "deposit", label: "保证金", unitLabel: "元", mode: "fixed" },
  { key: "pledge", label: "押金", unitLabel: "元", mode: "fixed" },
  { key: "trash", label: "垃圾处理费", unitLabel: "元/套", mode: "fixed" },
  { key: "water_supply", label: "二次供水费", unitLabel: "元/m³", mode: "fixed" },
  { key: "sewage", label: "污水处理费", unitLabel: "元/m³", mode: "fixed" },
  { key: "cycle_monthly", label: "月付", unitLabel: "元/月", mode: "fixed", cycleValue: "monthly" },
  { key: "cycle_quarterly", label: "季付", unitLabel: "元/季度", mode: "fixed", cycleValue: "quarterly" },
  { key: "cycle_semiannual", label: "半年付", unitLabel: "元/半年", mode: "fixed", cycleValue: "semiannual" },
  { key: "cycle_yearly", label: "年付", unitLabel: "元/年", mode: "fixed", cycleValue: "yearly" },
  { key: "bonus", label: "奖金", unitLabel: "元", mode: "fixed" },
  { key: "penalty", label: "罚款", unitLabel: "元", mode: "fixed" },
];

const DEFAULT_ENABLED_CHARGE_KEYS = AVAILABLE_CHARGE_ITEMS.filter((item) => item.defaultEnabled).map((item) => item.key);

const PAYMENT_CYCLE_MONTHS: Record<PaymentCycle, number> = {
  monthly: 1,
  quarterly: 3,
  semiannual: 6,
  yearly: 12,
};


const createDefaultInventorySettings = (): SiteInventoryConfigItem[] =>
  SITE_INVENTORY_OPTIONS.map((option) => ({
    ...option,
    enabled: false,
    quantity: "1",
  }));

const parseSiteInventorySettings = (stored?: SiteInventoryStoredItem[] | null): SiteInventoryConfigItem[] => {
  const base = createDefaultInventorySettings();
  if (!stored || stored.length === 0) {
    return base;
  }
  const map = new Map<string, SiteInventoryConfigItem>();
  base.forEach((item) => map.set(item.key, { ...item }));
  stored.forEach((entry) => {
    const current = map.get(entry.key);
    if (current) {
      current.enabled = entry.quantity > 0;
      current.quantity = entry.quantity > 0 ? String(entry.quantity) : "1";
    }
  });
  return Array.from(map.values());
};

const serializeSiteInventoryItems = (items: SiteInventoryConfigItem[]): SiteInventoryStoredItem[] =>
  items
    .filter((item) => item.enabled && item.quantity.trim())
    .map((item) => ({ key: item.key, quantity: Math.max(0, Number(item.quantity) || 0) }))
    .filter((item) => item.quantity > 0);

const INVENTORY_SUMMARY_SEPARATOR = " | ";

const formatInventorySummary = (items: SiteInventoryStoredItem[]) => {
  if (!items || items.length === 0) return "";
  return items
    .map((item) => {
      const option = SITE_INVENTORY_OPTIONS.find((opt) => opt.key === item.key);
      const label = option?.label || item.key;
      const unit = option?.unit || "件";
      return `${label}${item.quantity}${unit}`;
    })
    .join(INVENTORY_SUMMARY_SEPARATOR);
};

const buildInventorySelectorValue = (items: SiteInventoryConfigItem[]): DormItemSelectorValue => {
  return items.reduce<DormItemSelectorValue>((acc, item) => {
    acc[item.key] = {
      checked: item.enabled,
      count: Math.max(1, Number.parseInt(item.quantity, 10) || 1),
    };
    return acc;
  }, {});
};

const applyInventorySelectorValue = (items: SiteInventoryConfigItem[], value: DormItemSelectorValue) =>
  items.map((item) => {
    const nextState = value[item.key];
    if (!nextState) return item;
    return {
      ...item,
      enabled: Boolean(nextState.checked),
      quantity: String(Math.max(1, nextState.count || 1)),
    };
  });

const mergeInventorySummaryWithNote = (summary?: string, existing?: string) => {
  const trimmedSummary = summary?.trim() ?? "";
  const trimmedExisting = existing?.trim() ?? "";
  if (!trimmedSummary) {
    return trimmedExisting;
  }
  if (!trimmedExisting) {
    return trimmedSummary;
  }
  if (trimmedExisting.startsWith(trimmedSummary)) {
    return trimmedExisting;
  }
  const needsSeparator = !trimmedExisting.startsWith("|") && !trimmedSummary.endsWith("|");
  return `${trimmedSummary}${needsSeparator ? INVENTORY_SUMMARY_SEPARATOR : ""}${trimmedExisting}`;
};

const formatDateToInputString = (date: Date) => {
  const year = date.getFullYear();
  const month = String(date.getMonth() + 1).padStart(2, "0");
  const day = String(date.getDate()).padStart(2, "0");
  return `${year}-${month}-${day}`;
};

const buildNaturalMonthRange = (monthValue?: string | null) => {
  if (!monthValue) return null;
  const [yearStr, monthStr] = monthValue.split("-");
  if (!yearStr || !monthStr) return null;
  const year = Number(yearStr);
  const monthIndex = Number(monthStr) - 1;
  if (!Number.isFinite(year) || !Number.isFinite(monthIndex) || monthIndex < 0 || monthIndex > 11) {
    return null;
  }
  const start = new Date(Date.UTC(year, monthIndex, 1));
  const end = new Date(Date.UTC(year, monthIndex + 1, 0));
  return {
    start: formatDateToInputString(start),
    end: formatDateToInputString(end),
  };
};

type MeterColumnId =
  | "site"
  | "building"
  | "room"
  | "roomType"
  | "occupants"
  | "shareDetails"
  | "meterDate"
  | "inspector"
  | "billingRange"
  | "electricStart"
  | "electricEnd"
  | "electricFee"
  | "waterStart"
  | "waterEnd"
  | "waterFee"
  | "gasStart"
  | "gasEnd"
  | "gasFee";

const DEFAULT_ROOM_SORT: ColumnSortState = { columnId: "roomNumber", direction: "asc" };
const DEFAULT_CONTRACT_SORT: ColumnSortState = { columnId: "startDate", direction: "desc" };
const DEFAULT_METER_SORT: TableSortState<MeterColumnId> = { key: "meterDate", direction: "desc" };

interface MeterRecord {
  id: number;
  room_id: number;
  site_id: number | null;
  site_name: string;
  building_name: string;
  room_number: string;
  room_type: string;
  occupants: string[];
  meter_date: string;
  inspector: string;
  billing_month: string;
  billing_range: string;
  billing_start: string;
  billing_end: string;
  electric_start: number | null;
  electric_end: number | null;
  electric_fee: number | null;
  water_start: number | null;
  water_end: number | null;
  water_fee: number | null;
  gas_start: number | null;
  gas_end: number | null;
  gas_fee: number | null;
  charges: MeterChargeView[];
}

interface MeterMemberEntry {
  key: string;
  name: string;
  contractId?: number;
}

type MeterMemberShareRow = {
  key: string;
  name: string;
  contractId?: number;
  ratioWeight: number;
  ratioNormalized: number;
  autoCharges: Record<string, number | null>;
  resolvedCharges: Record<string, number | null>;
  totalAmount: number | null;
};

interface MeterFormState {
  room_id: string;
  billing_month?: string;
  billing_auto?: string;
  meter_date: string;
  inspector: string;
  billing_start: string;
  billing_end: string;
  electric_start: string;
  electric_end: string;
  water_start: string;
  water_end: string;
  gas_start: string;
  gas_end: string;
}

interface MeterChargeView {
  key: string;
  label: string;
  mode: ChargeMode;
  unitLabel?: string;
  unitPrice?: number | null;
  start?: number | null;
  end?: number | null;
  usage?: number | null;
  amount?: number | null;
  participants?: MeterChargeParticipantView[];
}

type MeterChargeParticipantView = {
  name: string;
  contractId?: number;
  amount?: number | null;
  ratio?: number | null;
};

const describeMeterChargeParticipant = (participant: MeterChargeParticipantView) => {
  const tokens: string[] = [];
  tokens.push(participant.name || "未命名");
  if (participant.amount != null) {
    tokens.push(formatCurrencyValue(participant.amount));
  }
  if (participant.ratio != null && Number.isFinite(participant.ratio)) {
    tokens.push(String(Math.round(participant.ratio * 100)) + "%");
  }
  return tokens.join(" ");
};

const buildChargeShareSummary = (charge: MeterChargeView) => {
  if (!charge.participants || charge.participants.length === 0) return "";
  const content = charge.participants.map(describeMeterChargeParticipant).join(" / ");
  return (charge.label || charge.key) + "：" + content;
};

const buildChargesShareSummary = (charges: MeterChargeView[]) =>
  charges
    .map(buildChargeShareSummary)
    .filter((text) => Boolean(text))
    .join("；");

const createBlankMeterForm = (): MeterFormState => {
  const today = new Date();
  const monthStart = new Date(today.getFullYear(), today.getMonth(), 1).toISOString().slice(0, 10);
  const nextMonth = new Date(today.getFullYear(), today.getMonth() + 1, 1);
  const monthEnd = new Date(nextMonth.getTime() - 86400000).toISOString().slice(0, 10);
  return {
    room_id: "",
    billing_month: today.toISOString().slice(0, 7),
    billing_auto: monthStart,
    meter_date: today.toISOString().slice(0, 10),
    inspector: "李永娇",
    billing_start: monthStart,
    billing_end: monthEnd,
    electric_start: "",
    electric_end: "",
    water_start: "",
    water_end: "",
    gas_start: "",
    gas_end: "",
  };
};

const toNullableNumber = (value?: number | null) => {
  return typeof value === "number" && Number.isFinite(value) ? value : null;
};

const METER_COLUMN_CONFIGS: { id: MeterColumnId; label: string; numeric?: boolean; width?: string }[] = [
  { id: "site", label: "地点", width: "140px" },
  { id: "building", label: "楼栋", width: "120px" },
  { id: "room", label: "房号", width: "100px" },
  { id: "roomType", label: "房间类型", width: "110px" },
  { id: "occupants", label: "入住人员", width: "220px" },
  { id: "shareDetails", label: "分摊详情", width: "260px" },
  { id: "meterDate", label: "抄表时间", width: "120px" },
  { id: "inspector", label: "抄表人", width: "110px" },
  { id: "billingRange", label: "账期月份", width: "140px" },
  { id: "electricStart", label: "电表起度", numeric: true },
  { id: "electricEnd", label: "电表止度", numeric: true },
  { id: "electricFee", label: "电费", numeric: true },
  { id: "waterStart", label: "水表起度", numeric: true },
  { id: "waterEnd", label: "水表止度", numeric: true },
  { id: "waterFee", label: "水费", numeric: true },
  { id: "gasStart", label: "气表起度", numeric: true },
  { id: "gasEnd", label: "气表止度", numeric: true },
  { id: "gasFee", label: "气费", numeric: true },
];

const METER_COLUMN_CHARGE_DEPENDENCIES: Partial<Record<MeterColumnId, string>> = {
  electricStart: "electric",
  electricEnd: "electric",
  electricFee: "electric",
  waterStart: "water",
  waterEnd: "water",
  waterFee: "water",
  gasStart: "gas",
  gasEnd: "gas",
  gasFee: "gas",
};

const METER_SORTABLE_COLUMN_IDS = METER_COLUMN_CONFIGS.map((column) => column.id);

type MeterColumnConfig = (typeof METER_COLUMN_CONFIGS)[number];

type SortDirection = "asc" | "desc";

type ColumnSortState = {
  columnId: string;
  direction: SortDirection;
};

const parseAllOrNumber = (value: unknown): "all" | number | null => {
  if (value === "all") {
    return "all";
  }
  if (typeof value === "number" && Number.isFinite(value)) {
    return value;
  }
  return null;
};

const isRoomStatusOption = (value: unknown): value is (typeof ROOM_STATUS_OPTIONS)[number] => {
  if (typeof value !== "string") {
    return false;
  }
  return ROOM_STATUS_OPTIONS.includes(value as (typeof ROOM_STATUS_OPTIONS)[number]);
};

const isContractStatusOption = (value: unknown): value is "all" | "active" | "completed" => {
  return value === "all" || value === "active" || value === "completed";
};

const MEMO_PRIORITY_TEXT_CLASS: Record<SiteMemoPriority, string> = {
  low: "text-emerald-600",
  normal: "text-amber-500",
  urgent: "text-red-500",
};

const ATTACHMENT_SEPARATOR = ":::";

const encodeAttachmentEntries = (entries: AttachmentEntry[]) =>
  entries
    .filter((entry) => entry.data)
    .map((entry, index) => {
      const fallbackName = `附件${index + 1}`;
      const safeName = (entry.name?.trim() || fallbackName).replace(/[|]/g, "-");
      return `${safeName}${ATTACHMENT_SEPARATOR}${entry.data}`;
    });

const formatCurrencyValue = (value?: number | string) => {
  if (value === null || value === undefined || value === "") return "";
  const numeric = typeof value === "number" ? value : Number(value);
  if (Number.isFinite(numeric)) {
    return numeric.toFixed(2);
  }
  return String(value);
};

const parseDecimalInput = (value?: string | null) => {
  if (value == null) return null;
  const trimmed = value.trim();
  if (!trimmed) return null;
  const parsed = Number(trimmed);
  return Number.isFinite(parsed) ? parsed : null;
};

const formatUnitValue = (value?: number | string, unit?: string) => {
  if (value === null || value === undefined || value === "") return "";
  return `${value}${unit ?? ""}`;
};

const formatBooleanLabel = (state: boolean) => (state ? "是" : "否");

const DECIMAL_INPUT_PATTERN = /^\d+(?:\.\d{0,4})?$/;

const formatDateInShanghai = (value: string) => {
  const parsed = new Date(value);
  if (Number.isNaN(parsed.getTime())) {
    return value.slice(0, 10);
  }
  const formatter = new Intl.DateTimeFormat("zh-CN", {
    timeZone: "Asia/Shanghai",
    year: "numeric",
    month: "2-digit",
    day: "2-digit",
  });
  const parts = formatter.formatToParts(parsed);
  const year = parts.find((part) => part.type === "year")?.value ?? "";
  const month = parts.find((part) => part.type === "month")?.value ?? "";
  const day = parts.find((part) => part.type === "day")?.value ?? "";
  return `${year}-${month}-${day}`;
};

const formatDateLabel = (value?: string | null) => (value ? formatDateInShanghai(value) : "--");
const formatDateInputValue = (value?: string | null) => (value ? formatDateInShanghai(value) : "");

const subtractDaysFromDate = (dateString: string, days: number) => {
  if (!dateString || !Number.isFinite(days)) return "";
  const baseDate = new Date(dateString);
  if (Number.isNaN(baseDate.getTime())) return "";
  const target = new Date(baseDate);
  target.setDate(target.getDate() - days);
  return Number.isNaN(target.getTime()) ? "" : target.toISOString().slice(0, 10);
};

const calculateDaysUntil = (dateString: string) => {
  if (!dateString) return null;
  const target = new Date(`${dateString}T00:00:00`);
  if (Number.isNaN(target.getTime())) return null;
  const today = new Date();
  today.setHours(0, 0, 0, 0);
  const diffMs = target.getTime() - today.getTime();
  return Math.ceil(diffMs / 86400000);
};

const computeNextPaymentDate = (lastPaymentDate: string, cycle?: PaymentCycle | "") => {
  if (!lastPaymentDate || !cycle) return "";
  const baseDate = new Date(lastPaymentDate);
  if (Number.isNaN(baseDate.getTime())) return "";
  const monthsToAdd = PAYMENT_CYCLE_MONTHS[cycle] ?? 0;
  if (!monthsToAdd) return "";
  const nextDate = new Date(baseDate);
  nextDate.setMonth(nextDate.getMonth() + monthsToAdd);
  return nextDate.toISOString().slice(0, 10);
};

const rollPaymentDatesForward = (lastPaymentDate: string, cycle?: PaymentCycle | "") => {
  if (!lastPaymentDate || !cycle) {
    return { lastDate: lastPaymentDate, nextDate: lastPaymentDate && cycle ? computeNextPaymentDate(lastPaymentDate, cycle) : "" };
  }
  let currentLast = lastPaymentDate;
  let currentNext = computeNextPaymentDate(currentLast, cycle) || "";
  if (!currentNext) {
    return { lastDate: currentLast, nextDate: "" };
  }
  const today = new Date();
  today.setHours(0, 0, 0, 0);
  let guard = 0;
  while (currentNext) {
    const nextDateObj = new Date(`${currentNext}T00:00:00`);
    if (Number.isNaN(nextDateObj.getTime())) {
      break;
    }
    if (nextDateObj.getTime() >= today.getTime() || guard > 120) {
      break;
    }
    currentLast = currentNext;
    currentNext = computeNextPaymentDate(currentLast, cycle) || "";
    guard += 1;
  }
  return { lastDate: currentLast, nextDate: currentNext || "" };
};

const isValidPaymentCycleValue = (value?: string | null): value is PaymentCycle =>
  Boolean(value && PAYMENT_CYCLE_VALUES.includes(value as PaymentCycle));

const createBlankSiteContractForm = (): SiteContractExtra => ({
  partyA: "",
  partyB: "",
  agentA: "",
  agentB: "",
  signingDate: "",
  contractStartDate: "",
  contractEndDate: "",
  paymentCycle: "" as PaymentCycle | "",
  lastPaymentDate: "",
  nextPaymentDate: "",
  notes: "",
  paymentReminderEnabled: false,
  contractReminderEnabled: false,
});

const normalizeSiteContractForm = (extra?: SiteContractExtra | null): SiteContractExtra => {
  const base = createBlankSiteContractForm();
  if (!extra) return base;
  const normalizedCycle = isValidPaymentCycleValue(extra.paymentCycle) ? extra.paymentCycle : ("" as PaymentCycle | "");
  const startDate = extra.contractStartDate || extra.termStart || "";
  const endDate = extra.contractEndDate || extra.termEnd || "";
  const lastPaymentDate = extra.lastPaymentDate || extra.firstPaymentDate || "";
  const nextPaymentDate = extra.nextPaymentDate || computeNextPaymentDate(lastPaymentDate, normalizedCycle);
  return {
    ...base,
    ...extra,
    contractStartDate: startDate,
    contractEndDate: endDate,
    lastPaymentDate,
    paymentCycle: normalizedCycle,
    nextPaymentDate,
    paymentReminderEnabled: extra.paymentReminderEnabled ?? extra.reminderEnabled ?? false,
    contractReminderEnabled: extra.contractReminderEnabled ?? false,
    notes: extra.notes ?? "",
  };
};

const ROOM_IMPORT_HEADERS = [
  { key: "site", label: "宿舍地点 *" },
  { key: "building", label: "楼栋名称 *" },
  { key: "room_number", label: "房号 *" },
  { key: "house_layout", label: "户型" },
  { key: "room_category", label: "房间类型" },
  { key: "bed_count", label: "床位数量" },
  { key: "area_square", label: "建筑面积(m²)" },
  { key: "first_month_fee", label: "首月费用(元)" },
  { key: "monthly_rent", label: "月租金(元/月)" },
  { key: "quarterly_rent", label: "季租金(元/季)" },
  { key: "property_fee", label: "物业费(元/月)" },
  { key: "deposit_fee", label: "押金(元)" },
  { key: "guarantee_fee", label: "保证金(元)" },
  { key: "electric_base", label: "电费基准(元/kWh)" },
  { key: "water_base", label: "水费基准(元/m³)" },
  { key: "gas_base", label: "气费基准(元/m³)" },
  { key: "trash_fee", label: "垃圾处置费(元/套)" },
  { key: "water_supply_fee", label: "二次供水费(元/m³)" },
  { key: "sewage_fee", label: "污水处理费(元/m³)" },
  { key: "inventory", label: "物品清单（以顿号“、”分隔）" },
];

const ROOM_IMPORT_ACCESSORS: Record<string, string[]> = ROOM_IMPORT_HEADERS.reduce((acc, header) => {
  const sanitized = header.label
    .replace(/[\*]/g, "")
    .replace(/[（(].*?[)）]/g, "")
    .trim();
  acc[header.key] = [header.label, sanitized, header.key];
  return acc;
}, {} as Record<string, string[]>);

const CONTRACT_IMPORT_HEADERS = [
  { key: "employee_name", label: "姓名 *" },
  { key: "employee_department", label: "部门" },
  { key: "employee_phone", label: "联系电话" },
  { key: "employee_id_number", label: "身份证号码" },
  { key: "employee_residence", label: "户籍地址" },
  { key: "room_number", label: "房号 *" },
  { key: "bed_number", label: "床位" },
  { key: "start_date", label: "入住日期 *" },
  { key: "end_date", label: "退宿日期" },
  { key: "rent_amount", label: "租金(元)" },
  { key: "deposit_amount", label: "押金(元)" },
  { key: "payment_method", label: "支付方式" },
  { key: "notes", label: "备注" },
];

const CONTRACT_IMPORT_ACCESSORS: Record<string, string[]> = CONTRACT_IMPORT_HEADERS.reduce((acc, header) => {
  const sanitized = header.label
    .replace(/[\*]/g, "")
    .replace(/[（(].*?[)）]/g, "")
    .trim();
  acc[header.key] = [header.label, sanitized, header.key];
  return acc;
}, {} as Record<string, string[]>);

const ROOM_STATUS_BADGE_CLASS: Record<string, string> = {
  空闲: "bg-emerald-50 text-emerald-700 border-emerald-200",
  缺人: "bg-amber-50 text-amber-700 border-amber-200",
  满员: "bg-rose-50 text-rose-700 border-rose-200",
  维护中: "bg-slate-200 text-slate-700 border-slate-300",
};

const BED_STATUS_BADGE_CLASS: Record<string, string> = {
  未分配: "bg-emerald-50 text-emerald-700 border-emerald-200",
  部分入住: "bg-sky-50 text-sky-700 border-sky-200",
  全部入住: "bg-rose-50 text-rose-700 border-rose-200",
};

const EMPTY_OCCUPANCY: RoomOccupancyMeta = { names: [], count: 0, bedAssignments: {}, members: [] };

const deriveRoomStatus = (room: DormRoom, occupancy: RoomOccupancyMeta) => {
  const manualStatus = room.status?.trim();
  if (manualStatus === "维护中") {
    return "维护中";
  }
  const capacity = room.bed_count || room.beds?.length || 0;
  if (capacity <= 1) {
    return occupancy.count >= 1 ? "满员" : "空闲";
  }
  if (occupancy.count === 0) return "空闲";
  if (capacity > 0 && occupancy.count >= capacity) {
    return "满员";
  }
  if (occupancy.count >= 2) {
    return "满员";
  }
  return "缺人";
};

const deriveBedStatus = (room: DormRoom, occupancy: RoomOccupancyMeta) => {
  const capacity = room.bed_count || room.beds?.length || 0;
  if (capacity <= 0) {
    return occupancy.count === 0 ? "未分配" : "部分入住";
  }
  if (occupancy.count === 0) return "未分配";
  if (occupancy.count >= capacity) return "全部入住";
  return "部分入住";
};

const formatBillingRangeLabel = (start?: string, end?: string) => {
  const normalize = (value?: string) => (value ? value.replaceAll("-", "") : "");
  if (!start && !end) return "";
  return `${normalize(start)}-${normalize(end)}`;
};

const computeShareRatio = (room?: DormRoom, occupancy?: RoomOccupancyMeta) => {
  if (!room) return 1;
  const capacity = room.bed_count || room.beds?.length || 1;
  if (capacity <= 1) return 1;
  const occupantCount = occupancy?.count && occupancy.count > 0 ? occupancy.count : capacity || 1;
  return 1 / occupantCount;
};

type ChargeSetting = {
  key: string;
  label: string;
  unitLabel: string;
  mode: ChargeMode;
  enabled: boolean;
  unitPrice: number;
};

type RoomChargeRateMeta = {
  unitPrice?: number;
  unitLabel?: string;
  mode?: ChargeMode;
};

type RoomChargeRecordType = "meter" | "rent" | "deposit" | "bonus" | "penalty";
type RoomChargeParticipant = {
  id: string;
  name: string;
  contractId?: number;
  amount?: string;
  collectDate?: string;
  refunded?: boolean;
  refundDate?: string;
};

const createLocalId = () => `${Date.now().toString(36)}-${Math.random().toString(36).slice(2)}`;

const createRoomChargeParticipant = (defaults: Partial<RoomChargeParticipant> = {}): RoomChargeParticipant => ({
  id: defaults.id ?? createLocalId(),
  name: defaults.name ?? "",
  contractId: defaults.contractId,
  amount: defaults.amount ?? "",
  collectDate: defaults.collectDate ?? "",
  refunded: defaults.refunded ?? false,
  refundDate: defaults.refundDate ?? "",
});

type RoomChargeRecordEntry = {
  type: RoomChargeRecordType;
  start?: string;
  end?: string;
  usage?: string;
  amount?: string;
  paymentDate?: string;
  paymentCycle?: string;
  nextPaymentDate?: string;
  paymentAmount?: string;
  addMemo?: boolean;
  collectDate?: string;
  refundDate?: string;
  refunded?: boolean;
  eventDate?: string;
  reason?: string;
  payerName?: string;
  participants?: RoomChargeParticipant[];
};
type RoomChargeRecordState = Record<string, RoomChargeRecordEntry>;

const participantHasData = (participant?: RoomChargeParticipant | null) => {
  if (!participant) return false;
  return Boolean(
    participant.name?.trim() ||
      (typeof participant.amount === "string" && participant.amount.trim() !== "") ||
      participant.collectDate ||
      participant.refundDate ||
      participant.refunded,
  );
};

const normalizeRoomChargeParticipant = (raw: unknown): RoomChargeParticipant | null => {
  if (!raw || typeof raw !== "object") return null;
  const source = raw as Record<string, unknown>;
  const id = typeof source.id === "string" && source.id.trim() ? source.id : createLocalId();
  const name = typeof source.name === "string" ? source.name.trim() : "";
  let amount = "";
  if (typeof source.amount === "string") {
    amount = source.amount.trim();
  } else if (typeof source.amount === "number" && Number.isFinite(source.amount)) {
    amount = source.amount.toString();
  }
  const collectDate = typeof source.collectDate === "string" ? source.collectDate : "";
  const refundDate = typeof source.refundDate === "string" ? source.refundDate : "";
  const contractId = typeof source.contractId === "number"
    ? source.contractId
    : typeof source.contract_id === "number"
      ? (source.contract_id as number)
      : undefined;
  const refunded = source.refunded === true || source.refunded === "true";
  const participant: RoomChargeParticipant = {
    id,
    name,
    amount,
    collectDate,
    refundDate,
    refunded,
    contractId,
  };
  return participantHasData(participant) ? participant : null;
};

const sanitizeRoomChargeParticipants = (value?: unknown): RoomChargeParticipant[] | undefined => {
  if (!value || !Array.isArray(value)) return undefined;
  const normalized = value
    .map((participant) => normalizeRoomChargeParticipant(participant))
    .filter((participant): participant is RoomChargeParticipant => Boolean(participant));
  return normalized.length > 0 ? normalized : undefined;
};

const enforceMandatoryChargeSelection = (items: ChargeSetting[]): ChargeSetting[] =>
  items.map((item) => (isMandatoryChargeKey(item.key) ? { ...item, enabled: true } : item));

const createDefaultChargeSettings = (): ChargeSetting[] =>
  enforceMandatoryChargeSelection(
    AVAILABLE_CHARGE_ITEMS.map((item) => ({
      key: item.key,
      label: item.label,
      unitLabel: item.unitLabel,
      mode: item.mode,
      enabled: item.defaultEnabled ?? false,
      unitPrice: item.defaultUnitPrice ?? 0,
    })),
  );

const cloneChargeSettings = (items: ChargeSetting[]) => items.map((item) => ({ ...item }));

const getChargeDefinition = (key: string) => AVAILABLE_CHARGE_ITEMS.find((item) => item.key === key);

const derivePaymentCycleOptions = (items: ChargeSetting[]) => {
  const cycles = new Set<PaymentCycle>();
  items.forEach((item) => {
    if (!item.enabled) return;
    const definition = getChargeDefinition(item.key);
    if (definition?.cycleValue) {
      cycles.add(definition.cycleValue);
    }
  });
  return Array.from(cycles);
};

const parseChargeConfigItems = (config?: DormChargeConfig | null): ChargeSetting[] => {
  const base = createDefaultChargeSettings();
  const map = new Map<string, ChargeSetting>();
  base.forEach((item) => map.set(item.key, item));
  const items = (config as DormChargeConfig | undefined)?.items ?? [];
  items.forEach((item) => {
    const def = getChargeDefinition(item.key);
    map.set(item.key, {
      key: item.key,
      label: item.label || def?.label || item.key,
      unitLabel: item.unit_label || def?.unitLabel || "",
      mode: item.mode || def?.mode || "meter",
      enabled: item.enabled ?? Boolean(def?.defaultEnabled),
      unitPrice: item.unit_price ?? def?.defaultUnitPrice ?? 0,
    });
  });
  return enforceMandatoryChargeSelection(Array.from(map.values()));
};

const serializeChargeConfig = (items: ChargeSetting[]): DormChargeConfig => ({
  items: items.map((item) => ({
    key: item.key,
    label: item.label,
    enabled: item.enabled,
    unit_price: Number.isFinite(item.unitPrice) ? item.unitPrice : 0,
    unit_label: item.unitLabel,
    mode: item.mode,
  })),
});

const parseChargeRates = (rates?: DormChargeRates | null) => {
  const map = new Map<string, RoomChargeRateMeta>();
  const items = (rates as DormChargeRates | undefined)?.items ?? [];
  items.forEach((item) => {
    map.set(item.key, {
      unitPrice: item.unit_price,
      unitLabel: item.unit_label,
      mode: item.mode,
    });
  });
  return map;
};

const ensureActiveCharges = (items: ChargeSetting[]): ChargeSetting[] => {
  const enabled = enforceMandatoryChargeSelection(items).filter((item) => item.enabled);
  if (enabled.length > 0) return enabled;
  return items.filter((item) => DEFAULT_ENABLED_CHARGE_KEYS.includes(item.key));
};

const mergeSiteRoomCharges = (siteItems: ChargeSetting[], rates?: DormChargeRates | null): ChargeSetting[] => {
  const overrides = parseChargeRates(rates);
  const merged = ensureActiveCharges(siteItems).map((item) => {
    const override = overrides.get(item.key);
    return {
      ...item,
      unitPrice: override?.unitPrice ?? item.unitPrice,
      unitLabel: override?.unitLabel ?? item.unitLabel,
      mode: override?.mode ?? item.mode,
    };
  });
  return merged.length ? merged : ensureActiveCharges(createDefaultChargeSettings());
};

const ROOM_RECORD_TYPE_BY_KEY: Record<string, RoomChargeRecordType> = {
  electric: "meter",
  water: "meter",
  gas: "meter",
  rent: "rent",
  property: "rent",
  deposit: "deposit",
  pledge: "deposit",
  bonus: "bonus",
  penalty: "penalty",
};

const ROOM_RECORD_KEY_ORDER = ["rent", "property", "deposit", "pledge", "bonus", "penalty"];

const shouldRenderRoomRecord = (key: string, costMode: ShareMode) => {
  if (key === "deposit" && costMode !== "company") return false;
  return true;
};

const getRoomRecordOrder = (key: string) => {
  const index = ROOM_RECORD_KEY_ORDER.indexOf(key);
  return index === -1 ? ROOM_RECORD_KEY_ORDER.length + 1 : index;
};

const determineRecordTypeForCharge = (key: string, mode?: ChargeMode): RoomChargeRecordType | null => {
  if (ROOM_RECORD_TYPE_BY_KEY[key]) return ROOM_RECORD_TYPE_BY_KEY[key];
  if (mode === "meter") return "meter";
  return null;
};

const resolveRecordUnitPrice = (sourceItem?: ChargeSetting, overrideValue?: string) => {
  if (overrideValue != null && overrideValue.trim() !== "") {
    const parsed = Number(overrideValue);
    if (!Number.isNaN(parsed)) {
      return parsed;
    }
  }
  if (typeof sourceItem?.unitPrice === "number" && Number.isFinite(sourceItem.unitPrice)) {
    return sourceItem.unitPrice;
  }
  return undefined;
};

const normalizeRoomRecordEntry = (
  entry: RoomChargeRecordEntry,
  sourceItem?: ChargeSetting,
  overrideValue?: string,
): RoomChargeRecordEntry => {
  if (!sourceItem) return entry;
  let modified = false;
  const next = { ...entry };
  const setField = <K extends keyof RoomChargeRecordEntry>(field: K, value: RoomChargeRecordEntry[K]) => {
    const current = next[field];
    const isEqual = current === value || (current == null && value == null);
    if (isEqual) return;
    next[field] = value;
    modified = true;
  };

  const resolvedUnitPrice = resolveRecordUnitPrice(sourceItem, overrideValue);
  if (entry.type === "meter") {
    const startNum = next.start != null && next.start !== "" ? Number(next.start) : NaN;
    const endNum = next.end != null && next.end !== "" ? Number(next.end) : NaN;
    if (!Number.isNaN(startNum) && !Number.isNaN(endNum)) {
      setField("usage", (endNum - startNum).toString());
    } else if (next.usage) {
      setField("usage", "");
    }
    const usageNum = next.usage != null && next.usage !== "" ? Number(next.usage) : NaN;
    if (!Number.isNaN(usageNum) && resolvedUnitPrice != null) {
      setField("amount", (usageNum * resolvedUnitPrice).toFixed(2));
    } else if (next.amount) {
      setField("amount", "");
    }
  } else if (entry.type === "rent") {
    const cycle = isValidPaymentCycleValue(next.paymentCycle) ? (next.paymentCycle as PaymentCycle) : undefined;
    if (!cycle && next.paymentCycle) {
      setField("paymentCycle", "");
    }
    if (next.paymentDate && cycle) {
      const rolled = rollPaymentDatesForward(next.paymentDate, cycle);
      setField("paymentDate", rolled.lastDate);
      setField("nextPaymentDate", rolled.nextDate || "");
    } else {
      const nextDate = next.paymentDate && cycle ? computeNextPaymentDate(next.paymentDate, cycle) : "";
      setField("nextPaymentDate", nextDate || "");
    }
    if (resolvedUnitPrice != null && cycle) {
      const months = PAYMENT_CYCLE_MONTHS[cycle] ?? 0;
      const multiplier = months > 0 ? months : 1;
      setField("paymentAmount", (resolvedUnitPrice * multiplier).toFixed(2));
    } else if (next.paymentAmount) {
      setField("paymentAmount", "");
    }
  } else if (entry.type === "deposit") {
    const participants = Array.isArray(next.participants)
      ? next.participants
      : [];
    const nextParticipants =
      participants.length > 0
        ? participants
        : next.collectDate || next.amount || next.refunded
          ? [createRoomChargeParticipant({ collectDate: next.collectDate, amount: next.amount, refunded: next.refunded, refundDate: next.refundDate })]
          : [];
    const normalizedParticipants = nextParticipants.map((participant) =>
      createRoomChargeParticipant({
        ...participant,
        id: participant.id,
        collectDate: participant.collectDate,
        amount: participant.amount,
        refunded: participant.refunded,
        refundDate: participant.refundDate,
        name: participant.name,
        contractId: participant.contractId,
      }),
    );
    setField("participants", normalizedParticipants);
    if (normalizedParticipants.length > 0) {
      const allRefunded = normalizedParticipants.every((participant) => participant.refunded);
      setField("refunded", allRefunded);
      if (!allRefunded) {
        setField("refundDate", "");
      }
    }
  }
  return modified ? next : entry;
};

const areRoomRecordEntriesEqual = (a: RoomChargeRecordEntry, b: RoomChargeRecordEntry) => {
  if (a === b) return true;
  if (!a || !b) return false;
  const keys = new Set([...Object.keys(a), ...Object.keys(b)]);
  for (const key of keys) {
    if ((a as Record<string, unknown>)[key] !== (b as Record<string, unknown>)[key]) {
      return false;
    }
  }
  return true;
};

const createDefaultRecordEntry = (type: RoomChargeRecordType): RoomChargeRecordEntry => ({ type });

const hasRecordData = (entry: RoomChargeRecordEntry) =>
  Boolean(
    entry.start ||
      entry.end ||
      entry.usage ||
      entry.paymentDate ||
      entry.paymentCycle ||
      entry.nextPaymentDate ||
      entry.paymentAmount ||
      entry.addMemo ||
      entry.collectDate ||
      entry.refundDate ||
      entry.refunded ||
      entry.eventDate ||
      entry.reason ||
      entry.payerName?.trim() ||
      entry.amount ||
      entry.participants?.some((participant) => participantHasData(participant)),
  );

const parseRoomRecordNotes = (notes?: string | null): RoomChargeRecordState => {
  if (!notes) return {};
  try {
    const parsed = JSON.parse(notes) as Record<string, unknown>;
    if (!parsed || typeof parsed !== "object") return {};
    const normalizedEntries: RoomChargeRecordState = {};
    Object.entries(parsed).forEach(([key, value]) => {
      if (!value || typeof value !== "object") return;
      const entry = value as RoomChargeRecordEntry;
      const normalized: RoomChargeRecordEntry = { ...entry };
      if (normalized.paymentCycle) {
        const cycleValue = normalized.paymentCycle as PaymentCycle;
        if (!PAYMENT_CYCLE_VALUES.includes(cycleValue)) {
          const fix = PAYMENT_CYCLE_LABEL_TO_KEY[normalized.paymentCycle as keyof typeof PAYMENT_CYCLE_LABEL_TO_KEY];
          if (fix) {
            normalized.paymentCycle = fix;
          }
        }
      }
      if (typeof normalized.refunded !== "boolean") {
        normalized.refunded = normalized.refunded === true || normalized.refunded === "true";
      }
      if (normalized.paymentDate && isValidPaymentCycleValue(normalized.paymentCycle)) {
        const rolled = rollPaymentDatesForward(normalized.paymentDate, normalized.paymentCycle as PaymentCycle);
        normalized.paymentDate = rolled.lastDate;
        normalized.nextPaymentDate = rolled.nextDate;
      }
      const participants = sanitizeRoomChargeParticipants(normalized.participants);
      if (participants) {
        normalized.participants = participants;
      } else {
        delete normalized.participants;
      }
      normalizedEntries[key] = normalized;
    });
    return normalizedEntries;
  } catch {
    return {};
  }
};

const serializeRoomRecordNotes = (records: RoomChargeRecordState) => {
  const cleaned = Object.entries(records).reduce<Record<string, RoomChargeRecordEntry>>((acc, [key, entry]) => {
    if (!entry || !hasRecordData(entry)) return acc;
    const normalized: RoomChargeRecordEntry = { ...entry };
    if (normalized.participants) {
      const sanitizedParticipants = sanitizeRoomChargeParticipants(normalized.participants);
      if (sanitizedParticipants) {
        normalized.participants = sanitizedParticipants;
      } else {
        delete normalized.participants;
      }
    }
    acc[key] = normalized;
    return acc;
  }, {});
  return Object.keys(cleaned).length > 0 ? JSON.stringify(cleaned) : "";
};

const buildRoomChargeOverridesFromRoom = (room?: DormRoom | null): Record<string, string> => {
  const overrides = parseChargeRates(room?.charge_rates as DormChargeRates | undefined);
  const result: Record<string, string> = {};
  overrides.forEach((value, key) => {
    if (value.unitPrice != null) {
      result[key] = String(value.unitPrice);
    }
  });
  return result;
};

const PRIORITY_LABELS: Record<SiteMemoPriority, string> = {
  low: "普通",
  normal: "重要",
  urgent: "紧急",
};

const PRIORITY_BADGE_CLASS: Record<SiteMemoPriority, string> = {
  low: "bg-muted text-muted-foreground border-muted",
  normal: "bg-amber-50 text-amber-700 border-amber-200",
  urgent: "bg-rose-50 text-rose-700 border-rose-200",
};

const ROOM_COLUMN_CONFIG: RoomColumnConfig[] = [
  {
    id: "site",
    label: "宿舍地点",
    sortable: true,
    defaultVisible: true,
    width: "150px",
    render: (_room, { site }) => site?.name || "-",
    getValue: (_room, { site }) => site?.name || "",
  },
  {
    id: "building",
    label: "楼栋",
    sortable: false,
    defaultVisible: true,
    width: "130px",
    render: (_room, { building }) => building?.name || "未命名楼栋",
    getValue: (_room, { building }) => building?.name || "",
  },
  {
    id: "roomNumber",
    label: "房号",
    sortable: true,
    defaultVisible: true,
    width: "100px",
    render: (room) => room.room_number || "-",
    getValue: (room) => room.room_number || "",
  },
  {
    id: "houseLayout",
    label: "户型",
    sortable: false,
    defaultVisible: true,
    width: "110px",
    render: (room) => room.house_layout || room.room_type || "-",
    getValue: (room) => room.house_layout || room.room_type || "",
  },
  {
    id: "single",
    label: "单间",
    sortable: false,
    defaultVisible: true,
    width: "80px",
    render: (room) => {
      const capacity = room.bed_count || room.beds?.length || 0;
      const isSingle = room.room_category === "单间" || capacity <= 1;
      return isSingle ? "是" : "否";
    },
    getValue: (room) => {
      const capacity = room.bed_count || room.beds?.length || 0;
      const isSingle = room.room_category === "单间" || capacity <= 1;
      return formatBooleanLabel(isSingle);
    },
  },
  {
    id: "area",
    label: "面积(㎡)",
    sortable: true,
    defaultVisible: true,
    width: "110px",
    numeric: true,
    align: "center",
    render: (room) => (room.area_square ? `${room.area_square}` : "-"),
    getValue: (room) => (room.area_square ? String(room.area_square) : ""),
  },
  {
    id: "monthlyRent",
    label: "月租金",
    sortable: true,
    defaultVisible: false,
    width: "110px",
    numeric: true,
    align: "center",
    render: (room) => (room.monthly_rent ? `¥${room.monthly_rent}` : "-"),
    getValue: (room) => formatCurrencyValue(room.monthly_rent),
  },
  {
    id: "propertyFee",
    label: "物业费",
    sortable: true,
    defaultVisible: false,
    width: "110px",
    numeric: true,
    align: "center",
    render: (room) => (room.property_fee ? `¥${room.property_fee}` : "-"),
    getValue: (room) => formatCurrencyValue(room.property_fee),
  },
  {
    id: "bedCount",
    label: "床位数量",
    sortable: true,
    defaultVisible: true,
    width: "130px",
    render: (room, { occupancy }) => {
      const capacity = room.bed_count || room.beds?.length || 0;
      if (!capacity && !occupancy.count) return "-";
      return `床位${capacity || 0}/入住${occupancy.count}`;
    },
    getValue: (room, { occupancy }) => {
      const capacity = room.bed_count || room.beds?.length || 0;
      if (!capacity && !occupancy.count) return "";
      return `床位${capacity || 0}/入住${occupancy.count}`;
    },
  },
  {
    id: "occupants",
    label: "入住人员",
    sortable: false,
    defaultVisible: true,
    width: "160px",
    render: (_room, { occupancy }) =>
      occupancy.names.length ? (
        occupancy.names.join("、")
      ) : (
        <Badge variant="outline" className="border bg-emerald-50 text-emerald-700 border-emerald-200">
          未分配
        </Badge>
      ),
    getValue: (_room, { occupancy }) => (occupancy.names.length ? occupancy.names.join("、") : "未分配"),
  },
  {
    id: "bedStatus",
    label: "床位状态",
    sortable: false,
    defaultVisible: true,
    width: "110px",
    render: (room, { occupancy }) => {
      const statusLabel = deriveBedStatus(room, occupancy);
      const badgeClass = BED_STATUS_BADGE_CLASS[statusLabel] ?? "bg-muted text-foreground border-muted";
      return (
        <Badge variant="outline" className={`border ${badgeClass}`}>
          {statusLabel}
        </Badge>
      );
    },
    getValue: (room, { occupancy }) => deriveBedStatus(room, occupancy),
  },
  {
    id: "roomStatus",
    label: "房间状态",
    sortable: true,
    defaultVisible: true,
    width: "110px",
    render: (room, { occupancy }) => {
      const statusLabel = deriveRoomStatus(room, occupancy);
      const badgeClass = ROOM_STATUS_BADGE_CLASS[statusLabel] ?? "bg-muted text-foreground border-muted";
      return (
        <Badge variant="outline" className={`border ${badgeClass}`}>
          {statusLabel}
        </Badge>
      );
    },
    getValue: (room, { occupancy }) => deriveRoomStatus(room, occupancy),
  },
  {
    id: "firstMonthFee",
    label: "首月费用",
    sortable: false,
    defaultVisible: false,
    width: "120px",
    numeric: true,
    align: "center",
    render: (room) => (room.first_month_fee ? `¥${room.first_month_fee}` : "-"),
    getValue: (room) => formatCurrencyValue(room.first_month_fee),
  },
  {
    id: "quarterlyRent",
    label: "季租金",
    sortable: false,
    defaultVisible: false,
    width: "120px",
    numeric: true,
    align: "center",
    render: (room) => (room.quarterly_rent ? `¥${room.quarterly_rent}` : "-"),
    getValue: (room) => formatCurrencyValue(room.quarterly_rent),
  },
  {
    id: "guaranteeFee",
    label: "保证金",
    sortable: false,
    defaultVisible: false,
    width: "120px",
    numeric: true,
    align: "center",
    render: (room) => (room.guarantee_fee ? `¥${room.guarantee_fee}` : "-"),
    getValue: (room) => formatCurrencyValue(room.guarantee_fee),
  },
  {
    id: "depositFee",
    label: "押金",
    sortable: false,
    defaultVisible: false,
    width: "120px",
    numeric: true,
    align: "center",
    render: (room) => (room.deposit_fee ? `¥${room.deposit_fee}` : "-"),
    getValue: (room) => formatCurrencyValue(room.deposit_fee),
  },
  {
    id: "electricBase",
    label: "电费基准",
    sortable: false,
    defaultVisible: false,
    width: "140px",
    render: (room) => (room.electric_base ? `${room.electric_base}/kWh` : "-"),
    getValue: (room) => formatUnitValue(room.electric_base, "/kWh"),
  },
  {
    id: "waterBase",
    label: "水费基准",
    sortable: false,
    defaultVisible: false,
    width: "140px",
    render: (room) => (room.water_base ? `${room.water_base}/m³` : "-"),
    getValue: (room) => formatUnitValue(room.water_base, "/m³"),
  },
  {
    id: "gasBase",
    label: "气费基准",
    sortable: false,
    defaultVisible: false,
    width: "120px",
    render: (room) => (room.gas_base ? `${room.gas_base}/m³` : "-"),
    getValue: (room) => formatUnitValue(room.gas_base, "/m³"),
  },
  {
    id: "trashFee",
    label: "垃圾处理费",
    sortable: false,
    defaultVisible: false,
    width: "120px",
    numeric: true,
    render: (room) => (room.trash_fee ? `¥${room.trash_fee}` : "-"),
    getValue: (room) => formatCurrencyValue(room.trash_fee),
  },
  {
    id: "waterSupplyFee",
    label: "二次供水费",
    sortable: false,
    defaultVisible: false,
    width: "140px",
    render: (room) => (room.water_supply_fee ? `${room.water_supply_fee}/m³` : "-"),
    getValue: (room) => formatUnitValue(room.water_supply_fee, "/m³"),
  },
  {
    id: "sewageFee",
    label: "污水处理费",
    sortable: false,
    defaultVisible: false,
    width: "140px",
    render: (room) => (room.sewage_fee ? `${room.sewage_fee}/m³` : "-"),
    getValue: (room) => formatUnitValue(room.sewage_fee, "/m³"),
  },
  {
    id: "notes",
    label: "物品备注",
    sortable: false,
    defaultVisible: false,
    width: "220px",
    render: (room) => <span className="line-clamp-2 text-xs text-muted-foreground">{room.inventory_note || "-"}</span>,
    getValue: (room) => room.inventory_note || "",
  },
];

const roomDynamicChargeDependencies: Array<{ columnId: string; key: string }> = [];

AVAILABLE_CHARGE_ITEMS.forEach((item) => {
  const unitLabel = item.unitLabel || "元";
  const columnId = `${item.key}Charge`;
  ROOM_COLUMN_CONFIG.push({
    id: columnId,
    label: `${item.label}${item.mode === "meter" ? "单价" : "金额"}`,
    sortable: false,
    defaultVisible: false,
    width: "130px",
    render: (_room, helpers) => {
      const value = helpers.chargeRates.get(item.key)?.unitPrice;
      if (value == null) return "-";
      return `${value}${unitLabel}`;
    },
    getValue: (_room, helpers) => {
      const value = helpers.chargeRates.get(item.key)?.unitPrice;
      return value == null ? "" : value.toString();
    },
  });
  roomDynamicChargeDependencies.push({ columnId, key: item.key });
});

const ROOM_COLUMN_CHARGE_DEPENDENCIES: Record<string, string> = {
  electricBase: "electric",
  waterBase: "water",
  gasBase: "gas",
  trashFee: "trash",
  waterSupplyFee: "water_supply",
  sewageFee: "sewage",
};

roomDynamicChargeDependencies.forEach(({ columnId, key }) => {
  ROOM_COLUMN_CHARGE_DEPENDENCIES[columnId] = key;
});

const CONTRACT_COLUMN_CHARGE_DEPENDENCIES: Record<string, string> = {
  rent: "rent",
  rentPlanMonth: "rent",
  deposit: "deposit",
  depositPlanMonth: "deposit",
};

const ROOM_SORTABLE_COLUMN_IDS = ROOM_COLUMN_CONFIG.filter((column) => column.sortable !== false).map((column) => column.id);

const CONTRACT_STATUS_META: Record<string, { label: string; badge: string }> = {
  active: { label: "已入住", badge: "bg-emerald-50 text-emerald-700 border-emerald-200" },
  completed: { label: "已退宿", badge: "bg-rose-50 text-rose-700 border-rose-200" },
};

const CONTRACT_COLUMN_CONFIG: ContractColumnConfig[] = [
  {
    id: "employee",
    label: "姓名",
    sortable: true,
    defaultVisible: true,
    width: "120px",
    render: (contract) => contract.employee_name || "-",
    getValue: (contract) => contract.employee_name || "",
  },
  {
    id: "department",
    label: "部门",
    sortable: true,
    defaultVisible: true,
    width: "140px",
    render: (contract) => contract.employee_department || "-",
    getValue: (contract) => contract.employee_department || "",
  },
  {
    id: "phone",
    label: "联系电话",
    sortable: false,
    defaultVisible: true,
    width: "140px",
    render: (contract) => contract.employee_phone || "-",
    getValue: (contract) => contract.employee_phone || "",
  },
  {
    id: "building",
    label: "楼栋",
    sortable: true,
    defaultVisible: true,
    width: "130px",
    render: (_contract, helpers) => helpers?.getBuildingName?.(_contract) ?? "-",
    getValue: (_contract, helpers) => helpers?.getBuildingName?.(_contract) ?? "",
  },
  {
    id: "room",
    label: "房间",
    sortable: true,
    defaultVisible: true,
    width: "110px",
    render: (contract) => contract.room?.room_number || "-",
    getValue: (contract) => contract.room?.room_number || "",
  },
  {
    id: "bed",
    label: "床位",
    sortable: false,
    defaultVisible: true,
    width: "100px",
    render: (contract) => contract.bed?.bed_number || "-",
    getValue: (contract) => contract.bed?.bed_number || "",
  },
  {
    id: "startDate",
    label: "入住日期",
    sortable: true,
    defaultVisible: true,
    width: "130px",
    render: (contract) => formatDateLabel(contract.start_date),
    getValue: (contract) => formatDateInputValue(contract.start_date),
  },
  {
    id: "endDate",
    label: "退宿日期",
    sortable: true,
    defaultVisible: true,
    width: "130px",
    render: (contract) => {
      if ((contract.status || "active") !== "completed" || !contract.end_date) {
        return "-";
      }
      return formatDateLabel(contract.end_date);
    },
    getValue: (contract) => ((contract.status || "active") !== "completed" ? "" : formatDateInputValue(contract.end_date)),
  },
  {
    id: "status",
    label: "状态",
    sortable: true,
    defaultVisible: true,
    width: "110px",
    render: (contract) => {
      const meta = CONTRACT_STATUS_META[contract.status || "active"] ?? CONTRACT_STATUS_META.active;
      return (
        <Badge variant="outline" className={`border ${meta.badge}`}>
          {meta.label}
        </Badge>
      );
    },
    getValue: (contract) => (CONTRACT_STATUS_META[contract.status || "active"] ?? CONTRACT_STATUS_META.active).label,
  },
  {
    id: "rent",
    label: "租金",
    sortable: true,
    defaultVisible: false,
    width: "110px",
    numeric: true,
    align: "center",
    render: (contract) => (contract.rent_amount ? `¥${contract.rent_amount}` : "-"),
    getValue: (contract) => formatCurrencyValue(contract.rent_amount),
  },
  {
    id: "deposit",
    label: "押金",
    sortable: true,
    defaultVisible: false,
    width: "110px",
    numeric: true,
    align: "center",
    render: (contract) => (contract.deposit_amount ? `¥${contract.deposit_amount}` : "-"),
    getValue: (contract) => formatCurrencyValue(contract.deposit_amount),
  },
  {
    id: "depositPlanMonth",
    label: "押金月份",
    sortable: false,
    defaultVisible: false,
    width: "140px",
    render: (contract, helpers) => formatMonthDisplay(helpers?.getNoteMeta?.(contract).depositPlanMonth),
    getValue: (contract, helpers) => helpers?.getNoteMeta?.(contract).depositPlanMonth || "",
  },
  {
    id: "rentPlanMonth",
    label: "租金月份",
    sortable: false,
    defaultVisible: false,
    width: "140px",
    render: (contract, helpers) => formatMonthDisplay(helpers?.getNoteMeta?.(contract).rentPlanMonth),
    getValue: (contract, helpers) => helpers?.getNoteMeta?.(contract).rentPlanMonth || "",
  },
  {
    id: "payment",
    label: "支付方式",
    sortable: false,
    defaultVisible: false,
    width: "120px",
    render: (contract) => contract.payment_method || "-",
    getValue: (contract) => contract.payment_method || "",
  },
  {
    id: "attachments",
    label: "附件",
    sortable: false,
    defaultVisible: false,
    width: "100px",
    render: (contract) => (contract.attachments?.length ? `${contract.attachments.length} 个` : "-"),
    getValue: (contract) => (contract.attachments?.length ? `${contract.attachments.length}` : ""),
  },
];

const CONTRACT_SORTABLE_COLUMN_IDS = CONTRACT_COLUMN_CONFIG.filter((column) => column.sortable !== false).map((column) => column.id);

const buildDefaultContractVisibility = (config: ContractColumnConfig[] = CONTRACT_COLUMN_CONFIG) =>
  config.reduce((acc, column) => {
    acc[column.id] = column.defaultVisible !== false;
    return acc;
  }, {} as Record<string, boolean>);

const buildDefaultRoomVisibility = (config: RoomColumnConfig[] = ROOM_COLUMN_CONFIG) =>
  config.reduce((acc, column) => {
    acc[column.id] = column.defaultVisible !== false;
    return acc;
  }, {} as Record<string, boolean>);

const buildDefaultMeterVisibility = (config: MeterColumnConfig[] = METER_COLUMN_CONFIGS) =>
  config.reduce((acc, column) => {
    acc[column.id] = true;
    return acc;
  }, {} as Record<MeterColumnId, boolean>);

const safeParseJSON = <T,>(value: string | null, fallback: T): T => {
  if (!value) return fallback;
  try {
    return JSON.parse(value) as T;
  } catch {
    return fallback;
  }
};

const writeLocalStorageJSON = (key: string, value: unknown) => {
  if (typeof window === "undefined") return;
  try {
    localStorage.setItem(key, JSON.stringify(value));
  } catch (error) {
    console.error(`[Dormitory] 保存 ${key} 到本地存储失败`, error);
  }
};

const loadRoomPrintSettings = (): RoomPrintSettings => {
  if (typeof window === "undefined") return {};
  return safeParseJSON<RoomPrintSettings>(localStorage.getItem(ROOM_PRINT_SETTINGS_KEY), {});
};

const saveRoomPrintSettings = (settings: RoomPrintSettings) => {
  if (typeof window === "undefined") return;
  localStorage.setItem(ROOM_PRINT_SETTINGS_KEY, JSON.stringify(settings));
};

const loadContractPrintSettings = (): ContractPrintSettings => {
  if (typeof window === "undefined") return {};
  return safeParseJSON<ContractPrintSettings>(localStorage.getItem(CONTRACT_PRINT_SETTINGS_KEY), {});
};

const saveContractPrintSettings = (settings: ContractPrintSettings) => {
  if (typeof window === "undefined") return;
  localStorage.setItem(CONTRACT_PRINT_SETTINGS_KEY, JSON.stringify(settings));
};

const loadMeterPrintSettings = (): MeterPrintSettings => {
  if (typeof window === "undefined") return {};
  return safeParseJSON<MeterPrintSettings>(localStorage.getItem(METER_PRINT_SETTINGS_KEY), {});
};

const saveMeterPrintSettings = (settings: MeterPrintSettings) => {
  if (typeof window === "undefined") return;
  localStorage.setItem(METER_PRINT_SETTINGS_KEY, JSON.stringify(settings));
};

const loadSiteOrder = () => {
  if (typeof window === "undefined") return [] as number[];
  const data = safeParseJSON<number[]>(localStorage.getItem(SITE_ORDER_STORAGE_KEY), []);
  return Array.isArray(data) ? data : [];
};

const loadSiteMemos = () => {
  if (typeof window === "undefined") return {} as Record<string, SiteMemoEntry[]>;
  const data = safeParseJSON<Record<string, SiteMemoEntry[]>>(localStorage.getItem(SITE_MEMO_STORAGE_KEY), {});
  return data || {};
};

const createMemoId = () =>
  typeof crypto !== "undefined" && typeof crypto.randomUUID === "function" ? crypto.randomUUID() : `memo-${Date.now()}`;

const padTimeUnit = (value: number) => value.toString().padStart(2, "0");

const formatLocalDateString = (value: Date) => `${value.getFullYear()}-${padTimeUnit(value.getMonth() + 1)}-${padTimeUnit(value.getDate())}`;

const formatLocalTimeString = (value: Date) => `${padTimeUnit(value.getHours())}:${padTimeUnit(value.getMinutes())}`;

const createBlankMemoForm = () => {
  const now = new Date();
  return {
    date: formatLocalDateString(now),
    time: formatLocalTimeString(now),
    startDate: formatLocalDateString(now),
    startTime: formatLocalTimeString(now),
    endDate: "",
    endTime: "",
    content: "",
    priority: "low" as SiteMemoPriority,
    recurrence: "none" as SiteMemoRecurrence,
  };
};

const readRoomImportValue = (row: Record<string, unknown>, key: string) => {
  const candidates = ROOM_IMPORT_ACCESSORS[key] ?? [key];
  for (const candidate of candidates) {
    const value = row[candidate];
    if (value !== undefined) {
      return value;
    }
  }
  return row[key];
};

const readContractImportValue = (row: Record<string, unknown>, key: string) => {
  const candidates = CONTRACT_IMPORT_ACCESSORS[key] ?? [key];
  for (const candidate of candidates) {
    const value = row[candidate];
    if (value !== undefined) {
      return value;
    }
  }
  return row[key];
};

const toNumberValue = (value: unknown) => {
  const num = Number(value);
  return Number.isFinite(num) ? num : 0;
};

const parseInventoryItems = (value: string) =>
  value
    .split(/[、，,]/)
    .map((item) => item.trim())
    .filter(Boolean);

type XLSXWithSSF = typeof XLSX & {
  SSF?: {
    parse_date_code?: (input: number) => { y: number; m: number; d: number } | null;
  };
};

const normalizeExcelDate = (value: unknown) => {
  const parseDateCode = (XLSX as XLSXWithSSF).SSF?.parse_date_code;
  if (typeof value === "number" && typeof parseDateCode === "function") {
    const parsed = parseDateCode(value);
    if (parsed) {
      const date = new Date(Date.UTC(parsed.y, parsed.m - 1, parsed.d));
      return date.toISOString().slice(0, 10);
    }
  }
  if (value instanceof Date) {
    return Number.isNaN(value.getTime()) ? "" : value.toISOString().slice(0, 10);
  }
  if (typeof value === "string") {
    const trimmed = value.trim();
    if (!trimmed) return "";
    const date = new Date(trimmed);
    if (!Number.isNaN(date.getTime())) {
      return date.toISOString().slice(0, 10);
    }
    return trimmed.slice(0, 10);
  }
  return "";
};

const fileToDataUrl = (file: File) =>
  new Promise<string>((resolve, reject) => {
    const reader = new FileReader();
    reader.onload = () => resolve(typeof reader.result === "string" ? reader.result : "");
    reader.onerror = () => reject(new Error("文件读取失败"));
    reader.readAsDataURL(file);
  });

const loadSiteHouseExtras = () => {
  if (typeof window === "undefined") return {} as Record<string, SiteHouseExtra>;
  return safeParseJSON<Record<string, SiteHouseExtra>>(localStorage.getItem(SITE_HOUSE_STORAGE_KEY), {});
};

const loadSiteContractExtras = () => {
  if (typeof window === "undefined") return {} as Record<string, SiteContractExtra>;
  return safeParseJSON<Record<string, SiteContractExtra>>(localStorage.getItem(SITE_CONTRACT_STORAGE_KEY), {});
};

const loadRoomColumnOrder = () => {
  const fallback = ROOM_COLUMN_CONFIG.map((column) => column.id);
  if (typeof window === "undefined") return [...fallback];
  const cached = safeParseJSON<string[] | null>(localStorage.getItem(ROOM_COLUMN_ORDER_STORAGE_KEY), null);
  return normalizeOrder(cached ?? undefined, fallback);
};

const loadRoomColumnVisibility = () => {
  const defaults = ROOM_COLUMN_CONFIG.reduce((acc, column) => {
    acc[column.id] = column.defaultVisible !== false;
    return acc;
  }, {} as Record<string, boolean>);
  if (typeof window === "undefined") return defaults;
  const cached = safeParseJSON<Record<string, boolean> | null>(localStorage.getItem(ROOM_COLUMN_VISIBILITY_STORAGE_KEY), null);
  return normalizeVisibility(cached ?? undefined, defaults);
};

const loadContractColumnOrder = () => {
  const fallback = CONTRACT_COLUMN_CONFIG.map((column) => column.id);
  if (typeof window === "undefined") return [...fallback];
  const cached = safeParseJSON<string[] | null>(localStorage.getItem(CONTRACT_COLUMN_ORDER_STORAGE_KEY), null);
  return normalizeOrder(cached ?? undefined, fallback);
};

const loadContractColumnVisibility = () => {
  const defaults = CONTRACT_COLUMN_CONFIG.reduce((acc, column) => {
    acc[column.id] = column.defaultVisible !== false;
    return acc;
  }, {} as Record<string, boolean>);
  if (typeof window === "undefined") return defaults;
  const cached = safeParseJSON<Record<string, boolean> | null>(localStorage.getItem(CONTRACT_COLUMN_VISIBILITY_STORAGE_KEY), null);
  return normalizeVisibility(cached ?? undefined, defaults);
};

const loadMeterColumnOrder = () => {
  const fallback = METER_COLUMN_CONFIGS.map((column) => column.id);
  if (typeof window === "undefined") return [...fallback];
  const cached = safeParseJSON<MeterColumnId[] | null>(localStorage.getItem(METER_COLUMN_ORDER_STORAGE_KEY), null);
  return normalizeOrder(cached ?? undefined, fallback);
};

const loadMeterColumnVisibility = () => {
  const defaults = buildDefaultMeterVisibility();
  if (typeof window === "undefined") return defaults;
  const cached = safeParseJSON<Record<string, boolean> | null>(localStorage.getItem(METER_COLUMN_VISIBILITY_STORAGE_KEY), null);
  return normalizeVisibility(cached ?? undefined, defaults);
};

export function DormitoryManagement() {
  const [activeTab, setActiveTab] = useState("overview");
  const [sites, setSites] = useState<DormSite[]>([]);
  const [buildings, setBuildings] = useState<DormBuilding[]>([]);
  const [rooms, setRooms] = useState<DormRoom[]>([]);
  const [contracts, setContracts] = useState<DormContract[]>([]);
  const [bills, setBills] = useState<DormBill[]>([]);
  const [loading, setLoading] = useState(true);
  const [siteOrder, setSiteOrder] = useState<number[]>(() => (typeof window === "undefined" ? [] : loadSiteOrder()));
  const [draggingSiteId, setDraggingSiteId] = useState<number | null>(null);
  const [siteMemos, setSiteMemos] = useState<Record<string, SiteMemoEntry[]>>(() => (typeof window === "undefined" ? {} : loadSiteMemos()));
  const siteCacheRef = useRef<Map<string, DormSite>>(new Map());
  const buildingCacheRef = useRef<Map<string, DormBuilding>>(new Map());

  const [activeSiteId, setActiveSiteId] = useState<number | null>(null);
  const [siteDialogOpen, setSiteDialogOpen] = useState(false);
  const [editingSite, setEditingSite] = useState<DormSite | null>(null);
  const blankSiteForm = {
    name: "",
    address: "",
    description: "",
    building_name: "",
    building_number: "",
    building_code: "",
    support_wechat: "",
  };
  const blankHouseForm: SiteHouseExtra = { propertyCompany: "", propertyContact: "", buildingNumber: "", buildingCodeSnapshot: "" };
  const [defaultChargePreference, setDefaultChargePreference] = useState<ChargeSetting[]>(() => createDefaultChargeSettings());
  const [siteForm, setSiteForm] = useState(blankSiteForm);
  const [siteHouseForm, setSiteHouseForm] = useState(blankHouseForm);
  const [siteContractForm, setSiteContractForm] = useState<SiteContractExtra>(() => createBlankSiteContractForm());
  const [siteInventoryItems, setSiteInventoryItems] = useState<SiteInventoryConfigItem[]>(() => createDefaultInventorySettings());
  const siteInventorySelectorValue = useMemo(() => buildInventorySelectorValue(siteInventoryItems), [siteInventoryItems]);
  const handleInventorySelectorChange = (nextValue: DormItemSelectorValue) => {
    setSiteInventoryItems((prev) => applyInventorySelectorValue(prev, nextValue));
  };
  const handleSiteChargeToggle = useCallback(
    (key: string, enabled: boolean) => {
      setSiteChargeItems((prev) => {
        const next = prev.map((item) => ({ ...item }));
        const getItem = (targetKey: string) => next.find((item) => item.key === targetKey);
        const applyEnabled = (targetKey: string, value: boolean) => {
          const target = getItem(targetKey);
          if (target) {
            target.enabled = value;
          }
        };
        if (WATER_DEPENDENT_KEYS.includes(key)) {
          const waterItem = getItem("water");
          if (enabled && !waterItem?.enabled) {
            toast.error("请先勾选水费，再启用垃圾/二次供水/污水费用");
            return prev;
          }
          WATER_DEPENDENT_KEYS.forEach((depKey) => applyEnabled(depKey, enabled));
        } else if (key === "water") {
          if (!enabled) {
            const extrasActive = WATER_DEPENDENT_KEYS.some((depKey) => getItem(depKey)?.enabled);
            if (extrasActive) {
              toast.error("关闭水费前请先取消垃圾/二次供水/污水费用");
              return prev;
            }
          }
          applyEnabled("water", enabled);
        } else {
          applyEnabled(key, enabled);
        }
        return enforceMandatoryChargeSelection(next);
      });
    },
    [],
  );
  const [siteDialogTab, setSiteDialogTab] = useState("house");
  const [siteHouseExtras, setSiteHouseExtras] = useState<Record<string, SiteHouseExtra>>(() => (typeof window === "undefined" ? {} : loadSiteHouseExtras()));
  const [siteContractExtras, setSiteContractExtras] = useState<Record<string, SiteContractExtra>>(() => (typeof window === "undefined" ? {} : loadSiteContractExtras()));
  const [siteChargeItems, setSiteChargeItems] = useState<ChargeSetting[]>(() => createDefaultChargeSettings());
  const memoKey = useCallback((siteId: number) => String(siteId), []);
  const siteInventoryConfigMap = useMemo(() => {
    const map = new Map<number, SiteInventoryStoredItem[]>();
    sites.forEach((site) => {
      const extra = siteHouseExtras[memoKey(site.id)];
      map.set(site.id, extra?.inventoryItems ?? []);
    });
    return map;
  }, [sites, siteHouseExtras, memoKey]);
  const siteInventorySummaryMap = useMemo(() => {
    const map = new Map<number, string>();
    siteInventoryConfigMap.forEach((items, siteId) => {
      map.set(siteId, formatInventorySummary(items));
    });
    return map;
  }, [siteInventoryConfigMap]);
  const [memoForm, setMemoForm] = useState(createBlankMemoForm);
  const [siteDeleteTarget, setSiteDeleteTarget] = useState<DormSite | null>(null);
  const [roomDeleteTarget, setRoomDeleteTarget] = useState<DormRoom | null>(null);
  const [siteDeleting, setSiteDeleting] = useState(false);
  const [roomDeleting, setRoomDeleting] = useState(false);
  const [contractDeleteTarget, setContractDeleteTarget] = useState<DormContract | null>(null);
  const [contractDeleting, setContractDeleting] = useState(false);
  const [wechatDialogOpen, setWechatDialogOpen] = useState(false);
  const [activeWechatSiteId, setActiveWechatSiteId] = useState<number | null>(null);

const initialRoomForm = {
  site_id: "",
  building_id: "",
  room_number: "",
  room_category: "单间",
  house_layout: "一室一厅",
  bed_count: 1,
  area_square: "",
  first_month_fee: "",
  monthly_rent: "",
  property_fee: "",
  quarterly_rent: "",
  guarantee_fee: "",
  deposit_fee: "",
  water_base: "",
  electric_base: "",
  gas_base: "",
  trash_fee: "",
  water_supply_fee: "",
  sewage_fee: "",
  inventory_note: "",
  status: "",
  cost_bearing_mode: "company" as ShareMode,
  company_name: "",
};
  const [roomDialogOpen, setRoomDialogOpen] = useState(false);
  const [roomDialogTab, setRoomDialogTab] = useState<"detail" | "records" | "history">("detail");
  const [roomForm, setRoomForm] = useState(initialRoomForm);
  const [roomChargeOverrides, setRoomChargeOverrides] = useState<Record<string, string>>({});
  const [roomChargeRecords, setRoomChargeRecords] = useState<RoomChargeRecordState>({});
  const [selectedRoomIds, setSelectedRoomIds] = useState<number[]>([]);
  const [roomSearch, setRoomSearch] = useState("");
  const [selectedSiteCardId, setSelectedSiteCardId] = useState<"all" | number>("all");
  const [roomSiteFilter, setRoomSiteFilter] = useState<"all" | number>("all");
  const [roomStatusFilter, setRoomStatusFilter] = useState<"all" | (typeof ROOM_STATUS_OPTIONS)[number]>("all");
  const [roomTypeFilter, setRoomTypeFilter] = useState<"all" | "company" | "personal">("all");
  const [contractSiteFilter, setContractSiteFilter] = useState<"all" | number>("all");
  const [roomColumnOrder, setRoomColumnOrder] = useState<string[]>(() => (typeof window === "undefined" ? [] : loadRoomColumnOrder()));
  const [roomColumnVisibility, setRoomColumnVisibility] = useState<Record<string, boolean>>(() =>
    typeof window === "undefined" ? {} : loadRoomColumnVisibility(),
  );
  const [showRoomFieldSelector, setShowRoomFieldSelector] = useState(false);
  const [draggingColumn, setDraggingColumn] = useState<string | null>(null);
  const [roomSort, setRoomSort] = useState<ColumnSortState>(DEFAULT_ROOM_SORT);
  const [editingRoomId, setEditingRoomId] = useState<number | null>(null);
  const [allEmployees, setAllEmployees] = useState<EmployeeResponse[]>([]);
  const [employeeLookupLoading, setEmployeeLookupLoading] = useState(false);
  const [employeeSearchTerm, setEmployeeSearchTerm] = useState("");
  const [employeeSuggestions, setEmployeeSuggestions] = useState<EmployeeResponse[]>([]);
  const [employeeFetchError, setEmployeeFetchError] = useState<string | null>(null);
  const [roomImportDialogOpen, setRoomImportDialogOpen] = useState(false);
  const [roomImporting, setRoomImporting] = useState(false);
  const [roomImportResult, setRoomImportResult] = useState<{ inserted: number; skipped: number } | null>(null);
  const [roomImportFile, setRoomImportFile] = useState<File | null>(null);
  const [roomImportError, setRoomImportError] = useState("");
  const roomImportFileInputRef = useRef<HTMLInputElement | null>(null);
  const contractImportFileInputRef = useRef<HTMLInputElement | null>(null);
  const [roomPrintDialogOpen, setRoomPrintDialogOpen] = useState(false);
  const [roomPrintContext, setRoomPrintContext] = useState<DormRoom[]>([]);
  const [roomPrintSuggestedTitle, setRoomPrintSuggestedTitle] = useState("");
  const [roomPrintTitle, setRoomPrintTitle] = useState(() => loadRoomPrintSettings().title ?? "");
  const [roomPrintWatermark, setRoomPrintWatermark] = useState(() => loadRoomPrintSettings().watermark ?? "内部资料 请勿外传");
  const [roomPrintOrientation, setRoomPrintOrientation] = useState<PrintOrientation>(() => loadRoomPrintSettings().orientation ?? "auto");
  const [contractPrintDialogOpen, setContractPrintDialogOpen] = useState(false);
  const [contractPrintContext, setContractPrintContext] = useState<DormContract[]>([]);
  const [contractPrintSuggestedTitle, setContractPrintSuggestedTitle] = useState("");
  const [contractPrintTitle, setContractPrintTitle] = useState(() => loadContractPrintSettings().title ?? "");
  const [contractPrintWatermark, setContractPrintWatermark] = useState(() => loadContractPrintSettings().watermark ?? "内部资料 请勿外传");
  const [contractPrintOrientation, setContractPrintOrientation] = useState<PrintOrientation>(() => loadContractPrintSettings().orientation ?? "landscape");
  const [meterForm, setMeterForm] = useState(createBlankMeterForm);
  const [meterFormMode, setMeterFormMode] = useState<"create" | "edit">("create");
  const [meterExtraChargeInputs, setMeterExtraChargeInputs] = useState<Record<string, string>>({});
  const [meterMemberRatios, setMeterMemberRatios] = useState<Record<string, string>>({});
  const [meterMemberChargeOverrides, setMeterMemberChargeOverrides] = useState<Record<string, Record<string, string>>>({});
  const [meterExtraChargeToggles, setMeterExtraChargeToggles] = useState<Record<string, boolean>>({ ...DEFAULT_EXTRA_CHARGE_TOGGLES });
  const [editingMeterId, setEditingMeterId] = useState<number | null>(null);
  const [meterSourceRecords, setMeterSourceRecords] = useState<DormMeterRecord[]>([]);
  const [meterRecords, setMeterRecords] = useState<MeterRecord[]>([]);
  const [meterListTab, setMeterListTab] = useState<"personal" | "company">("personal");
  const [meterSearch, setMeterSearch] = useState("");
  const [meterSiteFilter, setMeterSiteFilter] = useState<"all" | number>("all");
  const [meterBuildingFilter, setMeterBuildingFilter] = useState<"all" | number>("all");
  const [meterPeriodFilter, setMeterPeriodFilter] = useState<string>("all");
  const [meterPeriodInitialized, setMeterPeriodInitialized] = useState(false);
  const [meterSelectedIds, setMeterSelectedIds] = useState<number[]>([]);
  const [meterSort, setMeterSort] = useState<TableSortState<MeterColumnId>>(DEFAULT_METER_SORT);
  const [meterExtraColumnVisibility, setMeterExtraColumnVisibility] = useState<Record<string, boolean>>({});
  const [meterFormBuildingId, setMeterFormBuildingId] = useState<"all" | number>("all");
  const [meterFormSiteId, setMeterFormSiteId] = useState<string>("");
  const [meterColumnOrder, setMeterColumnOrder] = useState<MeterColumnId[]>(() => {
    if (typeof window === "undefined") {
      return METER_COLUMN_CONFIGS.map((column) => column.id);
    }
    const cached = loadMeterColumnOrder();
    return cached.length ? cached : METER_COLUMN_CONFIGS.map((column) => column.id);
  });
  const [meterColumnVisibility, setMeterColumnVisibility] = useState<Record<MeterColumnId, boolean>>(() => loadMeterColumnVisibility());
  const [draggingMeterColumn, setDraggingMeterColumn] = useState<MeterColumnId | null>(null);
  const [userPreferenceReady, setUserPreferenceReady] = useState(false);
  const [showMeterFieldSelector, setShowMeterFieldSelector] = useState(false);
  const [meterImportDialogOpen, setMeterImportDialogOpen] = useState(false);
  const [meterDialogOpen, setMeterDialogOpen] = useState(false);
  const meterImportFileInputRef = useRef<HTMLInputElement | null>(null);
  const [meterImportFile, setMeterImportFile] = useState<File | null>(null);
  const [meterImporting, setMeterImporting] = useState(false);
  const [meterImportError, setMeterImportError] = useState("");
  const [meterImportResult, setMeterImportResult] = useState<{ inserted: number; skipped: number } | null>(null);
  const [meterDeleteContext, setMeterDeleteContext] = useState<{ mode: "single"; record: MeterRecord } | { mode: "bulk" } | null>(null);
  const [meterPrintDialogOpen, setMeterPrintDialogOpen] = useState(false);
  const [meterPrintContext, setMeterPrintContext] = useState<MeterRecord[]>([]);
  const [meterPrintSuggestedTitle, setMeterPrintSuggestedTitle] = useState("");
  const [meterPrintTitle, setMeterPrintTitle] = useState(() => loadMeterPrintSettings().title ?? "");
  const [meterPrintWatermark, setMeterPrintWatermark] = useState(() => loadMeterPrintSettings().watermark ?? "内部资料 请勿外传");
  const [meterPrintOrientation, setMeterPrintOrientation] = useState<PrintOrientation>(() => loadMeterPrintSettings().orientation ?? "landscape");
  const [meterSequentialRooms, setMeterSequentialRooms] = useState<number[]>([]);
  const [meterSequentialIndex, setMeterSequentialIndex] = useState(0);
  const [roomContractSyncing, setRoomContractSyncing] = useState(false);
  const [billGenerating, setBillGenerating] = useState(false);

  const siteById = useMemo(() => {
    const map = new Map<number, DormSite>();
    sites.forEach((site) => map.set(site.id, site));
    return map;
  }, [sites]);

  const siteChargeConfigMap = useMemo(() => {
    const map = new Map<number, ChargeSetting[]>();
    sites.forEach((site) => {
      map.set(site.id, ensureActiveCharges(parseChargeConfigItems(site.charge_config)));
    });
    return map;
  }, [sites]);

  const fallbackChargeSettings = useMemo(() => ensureActiveCharges(cloneChargeSettings(defaultChargePreference)), [defaultChargePreference]);

  const buildingById = useMemo(() => {
    const map = new Map<number, DormBuilding>();
    buildings.forEach((building) => map.set(building.id, building));
    return map;
  }, [buildings]);

  const getRoomSiteId = useCallback(
    (room?: DormRoom | null) => {
      if (!room) return null;
      if (room.site_id) return room.site_id;
      const building = buildingById.get(room.building_id);
      return building?.site_id ?? null;
    },
    [buildingById],
  );

  const meterFilterBuildings = useMemo(() => {
    if (meterSiteFilter === "all") {
      return buildings;
    }
    return buildings.filter((building) => building.site_id === meterSiteFilter);
  }, [meterSiteFilter, buildings]);

  useEffect(() => {
    if (meterBuildingFilter === "all") return;
    if (!meterFilterBuildings.some((building) => building.id === meterBuildingFilter)) {
      setMeterBuildingFilter("all");
    }
  }, [meterFilterBuildings, meterBuildingFilter]);

  useEffect(() => {
    if (!meterFormSiteId && sites.length > 0) {
      setMeterFormSiteId(String(sites[0].id));
    }
  }, [meterFormSiteId, sites]);


  const enabledChargeKeysForTables = useMemo(() => {
    const gatherKeys = (siteIds?: number[]) => {
      const enabled = new Set<string>();
      if (siteIds && siteIds.length > 0) {
        siteIds.forEach((siteId) => {
          const items = siteChargeConfigMap.get(siteId) ?? [];
          items.forEach((item) => {
            if (item.enabled) enabled.add(item.key);
          });
        });
      } else {
        sites.forEach((site) => {
          const items = siteChargeConfigMap.get(site.id) ?? [];
          items.forEach((item) => {
            if (item.enabled) enabled.add(item.key);
          });
        });
      }
      if (enabled.size === 0) {
        fallbackChargeSettings.forEach((item) => {
          if (item.enabled) enabled.add(item.key);
        });
      }
      if (enabled.size === 0) {
        DEFAULT_ENABLED_CHARGE_KEYS.forEach((key) => enabled.add(key));
      }
      return enabled;
    };

    const targetSites = new Set<number>();
    if (typeof selectedSiteCardId === "number") {
      targetSites.add(selectedSiteCardId);
    }
    if (roomSiteFilter !== "all") {
      targetSites.add(roomSiteFilter as number);
    }
    if (contractSiteFilter !== "all") {
      targetSites.add(contractSiteFilter as number);
    }
    if (meterBuildingFilter !== "all") {
      const building = buildingById.get(meterBuildingFilter as number);
      if (building?.site_id) targetSites.add(building.site_id);
    }

    return gatherKeys(targetSites.size > 0 ? Array.from(targetSites) : undefined);
  }, [
    sites,
    fallbackChargeSettings,
    siteChargeConfigMap,
    selectedSiteCardId,
    roomSiteFilter,
    contractSiteFilter,
    meterBuildingFilter,
    buildingById,
  ]);

  const effectiveRoomColumnConfig = useMemo(
    () => ROOM_COLUMN_CONFIG.filter((column) => {
      const dependency = ROOM_COLUMN_CHARGE_DEPENDENCIES[column.id];
      if (!dependency) return true;
      return enabledChargeKeysForTables.has(dependency);
    }),
    [enabledChargeKeysForTables],
  );

  const roomColumnMap = useMemo(() => {
    const map = new Map<string, RoomColumnConfig>();
    effectiveRoomColumnConfig.forEach((column) => map.set(column.id, column));
    return map;
  }, [effectiveRoomColumnConfig]);

  const effectiveMeterColumnConfig = useMemo(
    () => METER_COLUMN_CONFIGS.filter((column) => {
      const dependency = METER_COLUMN_CHARGE_DEPENDENCIES[column.id];
      if (!dependency) return true;
      return enabledChargeKeysForTables.has(dependency);
    }),
    [enabledChargeKeysForTables],
  );

  const meterColumnMap = useMemo(() => {
    const map = new Map<MeterColumnId, MeterColumnConfig>();
    effectiveMeterColumnConfig.forEach((column) => map.set(column.id, column));
    return map;
  }, [effectiveMeterColumnConfig]);

  const effectiveContractColumnConfig = useMemo(
    () =>
      CONTRACT_COLUMN_CONFIG.filter((column) => {
        const dependency = CONTRACT_COLUMN_CHARGE_DEPENDENCIES[column.id];
        if (!dependency) return true;
        return enabledChargeKeysForTables.has(dependency);
      }),
    [enabledChargeKeysForTables],
  );

  const contractColumnMap = useMemo(() => {
    const map = new Map<string, ContractColumnConfig>();
    effectiveContractColumnConfig.forEach((column) => map.set(column.id, column));
    return map;
  }, [effectiveContractColumnConfig]);

  const paymentCycleOptions = useMemo(() => derivePaymentCycleOptions(siteChargeItems), [siteChargeItems]);

  const activeRoomChargeItems = useMemo(() => {
    const siteId = roomForm.site_id ? Number(roomForm.site_id) : undefined;
    if (siteId && siteChargeConfigMap.get(siteId)?.length) {
      return siteChargeConfigMap.get(siteId)!;
    }
    return fallbackChargeSettings;
  }, [roomForm.site_id, siteChargeConfigMap, fallbackChargeSettings]);

  const roomRecordCycleOptions = useMemo(() => derivePaymentCycleOptions(activeRoomChargeItems), [activeRoomChargeItems]);

  useEffect(() => {
    setRoomChargeOverrides((prev) => {
      const next: Record<string, string> = {};
      activeRoomChargeItems.forEach((item) => {
        if (prev[item.key]) {
          next[item.key] = prev[item.key];
        }
      });
      return next;
    });
  }, [activeRoomChargeItems]);

  useEffect(() => {
    if (!roomDialogOpen) return;
    setRoomChargeRecords((prev) => {
      let changed = false;
      const next: RoomChargeRecordState = {};
      activeRoomChargeItems.forEach((item) => {
        const recordType = determineRecordTypeForCharge(item.key, item.mode);
        if (!recordType) return;
        const previous = prev[item.key] ?? createDefaultRecordEntry(recordType);
        let ensured = previous;
        if (previous.type !== recordType) {
          ensured = { ...previous, type: recordType };
          changed = true;
        }
        const normalized = normalizeRoomRecordEntry(ensured, item, roomChargeOverrides[item.key]);
        if (areRoomRecordEntriesEqual(ensured, normalized)) {
          next[item.key] = ensured;
        } else {
          next[item.key] = normalized;
          changed = true;
        }
        if (!prev[item.key]) {
          changed = true;
        }
      });
      Object.keys(prev).forEach((key) => {
        if (!next[key]) {
          changed = true;
        }
      });
      return changed ? next : prev;
    });
  }, [activeRoomChargeItems, roomChargeOverrides, roomDialogOpen]);

  useEffect(() => {
    if (!roomDialogOpen) return;
    if (roomForm.cost_bearing_mode !== "personal") return;
    setRoomChargeRecords((prev) => {
      if (!prev.deposit) return prev;
      const next = { ...prev };
      delete next.deposit;
      return next;
    });
  }, [roomDialogOpen, roomForm.cost_bearing_mode]);

  useEffect(() => {
    const siteId = roomForm.site_id ? Number(roomForm.site_id) : null;
    if (!siteId) return;
    const summary = siteInventorySummaryMap.get(siteId) ?? "";
    if (!summary) return;
    setRoomForm((prev) => {
      const merged = mergeInventorySummaryWithNote(summary, prev.inventory_note);
      if (merged === prev.inventory_note) {
        return prev;
      }
      return { ...prev, inventory_note: merged };
    });
  }, [roomForm.site_id, siteInventorySummaryMap]);


  useEffect(() => {
    if (sites.length === 0) {
      setSiteOrder([]);
      return;
    }
    setSiteOrder((prev) => {
      const filtered = prev.filter((id) => sites.some((site) => site.id === id));
      const missing = sites.map((site) => site.id).filter((id) => !filtered.includes(id));
      return [...filtered, ...missing];
    });
  }, [sites]);

  useEffect(() => {
    if (typeof window === "undefined") return;
    if (siteOrder.length === 0) {
      localStorage.removeItem(SITE_ORDER_STORAGE_KEY);
      return;
    }
    localStorage.setItem(SITE_ORDER_STORAGE_KEY, JSON.stringify(siteOrder));
  }, [siteOrder]);

  useEffect(() => {
    if (typeof window === "undefined") return;
    localStorage.setItem(SITE_MEMO_STORAGE_KEY, JSON.stringify(siteMemos));
  }, [siteMemos]);

  useEffect(() => {
    if (typeof window === "undefined") return;
    localStorage.setItem(SITE_HOUSE_STORAGE_KEY, JSON.stringify(siteHouseExtras));
  }, [siteHouseExtras]);

  useEffect(() => {
    if (typeof window === "undefined") return;
    localStorage.setItem(SITE_CONTRACT_STORAGE_KEY, JSON.stringify(siteContractExtras));
  }, [siteContractExtras]);

  useEffect(() => {
    saveRoomPrintSettings({
      title: roomPrintTitle.trim(),
      watermark: roomPrintWatermark.trim(),
      orientation: roomPrintOrientation,
    });
  }, [roomPrintTitle, roomPrintWatermark, roomPrintOrientation]);

  useEffect(() => {
    saveContractPrintSettings({
      title: contractPrintTitle.trim(),
      watermark: contractPrintWatermark.trim(),
      orientation: contractPrintOrientation,
    });
  }, [contractPrintTitle, contractPrintWatermark, contractPrintOrientation]);

  useEffect(() => {
    saveMeterPrintSettings({
      title: meterPrintTitle.trim(),
      watermark: meterPrintWatermark.trim(),
      orientation: meterPrintOrientation,
    });
  }, [meterPrintTitle, meterPrintWatermark, meterPrintOrientation]);

  useEffect(() => {
    const cache = siteCacheRef.current;
    cache.clear();
    sites.forEach((site) => {
      if (!site.name) return;
      cache.set(site.name.trim(), site);
    });
  }, [sites]);

  useEffect(() => {
    const cache = buildingCacheRef.current;
    cache.clear();
    buildings.forEach((building) => {
      cache.set(`${building.site_id}-${building.name?.trim() ?? ""}`, building);
    });
  }, [buildings]);

  useEffect(() => {
    let cancelled = false;
    const hydratePreferences = async () => {
      try {
    const prefs = await fetchUserPreferences();
    const roomPref = parseColumnPreferenceValue<string>(
      prefs[PREF_KEY_ROOM_COLUMNS],
      ROOM_COLUMN_CONFIG.map((column) => column.id),
      buildDefaultRoomVisibility(),
    );
    if (roomPref?.order) setRoomColumnOrder(roomPref.order);
    if (roomPref?.visibility) setRoomColumnVisibility(roomPref.visibility);

    const contractPref = parseColumnPreferenceValue<string>(
      prefs[PREF_KEY_CONTRACT_COLUMNS],
      CONTRACT_COLUMN_CONFIG.map((column) => column.id),
      buildDefaultContractVisibility(),
    );
    if (contractPref?.order) setContractColumnOrder(contractPref.order);
    if (contractPref?.visibility) setContractColumnVisibility(contractPref.visibility);

    const meterPref = parseColumnPreferenceValue<MeterColumnId>(
      prefs[PREF_KEY_METER_COLUMNS],
      METER_COLUMN_CONFIGS.map((column) => column.id),
      buildDefaultMeterVisibility(),
    );
    if (meterPref?.order) setMeterColumnOrder(meterPref.order);
    if (meterPref?.visibility) setMeterColumnVisibility(meterPref.visibility);

    const chargePrefRaw = prefs[PREF_KEY_CHARGE_ITEMS];
    if (chargePrefRaw && typeof chargePrefRaw === "object") {
      const rawConfig = chargePrefRaw as unknown as Partial<DormChargeConfig>;
      const parsedCharges = parseChargeConfigItems({
        items: Array.isArray(rawConfig.items) ? rawConfig.items : [],
      });
      setDefaultChargePreference(parsedCharges);
      if (!editingSite) {
        setSiteChargeItems(cloneChargeSettings(parsedCharges));
      }
    }

    const roomSortPref = prefs[PREF_KEY_ROOM_SORT];
    if (roomSortPref) {
      const next = sanitizeSortPreference(roomSortPref, ROOM_SORTABLE_COLUMN_IDS, {
        key: DEFAULT_ROOM_SORT.columnId,
        direction: DEFAULT_ROOM_SORT.direction,
      });
      if (next.key) {
        setRoomSort({ columnId: next.key, direction: next.direction });
      }
    }

    const contractSortPref = prefs[PREF_KEY_CONTRACT_SORT];
    if (contractSortPref) {
      const next = sanitizeSortPreference(contractSortPref, CONTRACT_SORTABLE_COLUMN_IDS, {
        key: DEFAULT_CONTRACT_SORT.columnId,
        direction: DEFAULT_CONTRACT_SORT.direction,
      });
      if (next.key) {
        setContractSort({ columnId: next.key, direction: next.direction });
      }
    }

    const meterSortPref = prefs[PREF_KEY_METER_SORT];
    if (meterSortPref) {
      const next = sanitizeSortPreference<MeterColumnId>(meterSortPref, METER_SORTABLE_COLUMN_IDS, DEFAULT_METER_SORT);
      setMeterSort(next);
    }

    const roomFilterPref = prefs[PREF_KEY_ROOM_FILTERS];
    if (roomFilterPref && typeof roomFilterPref === "object") {
      const { search, status, site, selectedSite, type } = roomFilterPref as Record<string, unknown>;
      if (typeof search === "string") setRoomSearch(search);
      if (isRoomStatusOption(status)) setRoomStatusFilter(status);
      const parsedSiteFilter = parseAllOrNumber(site);
      if (parsedSiteFilter !== null) {
        setRoomSiteFilter(parsedSiteFilter);
      }
      const parsedCardSelection = parseAllOrNumber(selectedSite);
      if (parsedCardSelection !== null) {
        setSelectedSiteCardId(parsedCardSelection);
      }
      if (type === "company" || type === "personal" || type === "all") {
        setRoomTypeFilter(type as "all" | "company" | "personal");
      }
    }

    const contractFilterPref = prefs[PREF_KEY_CONTRACT_FILTERS];
    if (contractFilterPref && typeof contractFilterPref === "object") {
      const { search, status, site, building } = contractFilterPref as Record<string, unknown>;
      if (typeof search === "string") setContractSearch(search);
      if (isContractStatusOption(status)) setContractStatusFilter(status);
      const parsedSiteFilter = parseAllOrNumber(site ?? building);
      if (parsedSiteFilter !== null) setContractSiteFilter(parsedSiteFilter);
    }

    const meterFilterPref = prefs[PREF_KEY_METER_FILTERS];
    if (meterFilterPref && typeof meterFilterPref === "object") {
      const { search, building } = meterFilterPref as Record<string, unknown>;
      if (typeof search === "string") setMeterSearch(search);
      const parsedBuilding = parseAllOrNumber(building);
      if (parsedBuilding !== null) setMeterBuildingFilter(parsedBuilding);
    }
  } catch (error) {
    console.error("[Dormitory] 加载用户偏好失败", error);
  } finally {
    if (!cancelled) setUserPreferenceReady(true);
  }
    };
    hydratePreferences();
    return () => {
      cancelled = true;
    };
  }, [editingSite]);

  useEffect(() => {
    if (Object.keys(roomColumnVisibility).length === 0 && effectiveRoomColumnConfig.length > 0) {
      setRoomColumnVisibility(buildDefaultRoomVisibility(effectiveRoomColumnConfig));
    }
  }, [roomColumnVisibility, effectiveRoomColumnConfig]);

  useEffect(() => {
    if (Object.keys(roomColumnVisibility).length === 0) return;
    const validIds = new Set(effectiveRoomColumnConfig.map((column) => column.id));
    setRoomColumnVisibility((prev) => {
      const next = { ...prev };
      let changed = false;
      Object.keys(next).forEach((key) => {
        if (!validIds.has(key)) {
          delete next[key];
          changed = true;
        }
      });
      effectiveRoomColumnConfig.forEach((column) => {
        if (!(column.id in next)) {
          next[column.id] = column.defaultVisible !== false;
          changed = true;
        }
      });
      return changed ? next : prev;
    });
  }, [effectiveRoomColumnConfig, roomColumnVisibility]);

  useEffect(() => {
    if (typeof window === "undefined") return;
    if (Object.keys(roomColumnVisibility).length === 0) {
      localStorage.removeItem(ROOM_COLUMN_VISIBILITY_STORAGE_KEY);
      return;
    }
    writeLocalStorageJSON(ROOM_COLUMN_VISIBILITY_STORAGE_KEY, roomColumnVisibility);
  }, [roomColumnVisibility]);

  useEffect(() => {
    if (Object.keys(meterColumnVisibility).length === 0 && effectiveMeterColumnConfig.length > 0) {
      setMeterColumnVisibility(buildDefaultMeterVisibility(effectiveMeterColumnConfig));
    }
  }, [meterColumnVisibility, effectiveMeterColumnConfig]);

  useEffect(() => {
    if (Object.keys(meterColumnVisibility).length === 0) return;
    const validIds = new Set(effectiveMeterColumnConfig.map((column) => column.id));
    setMeterColumnVisibility((prev) => {
      const next = { ...prev };
      let changed = false;
      Object.keys(next).forEach((key) => {
        const typedKey = key as MeterColumnId;
        if (!validIds.has(typedKey)) {
          delete next[typedKey];
          changed = true;
        }
      });
      effectiveMeterColumnConfig.forEach((column) => {
        if (!(column.id in next)) {
          next[column.id] = true;
          changed = true;
        }
      });
      return changed ? next : prev;
    });
  }, [effectiveMeterColumnConfig, meterColumnVisibility]);

  useEffect(() => {
    if (typeof window === "undefined") return;
    if (Object.keys(meterColumnVisibility).length === 0) {
      localStorage.removeItem(METER_COLUMN_VISIBILITY_STORAGE_KEY);
      return;
    }
    writeLocalStorageJSON(METER_COLUMN_VISIBILITY_STORAGE_KEY, meterColumnVisibility);
  }, [meterColumnVisibility]);

  const roomColumnsForRender = useMemo(
    () =>
      roomColumnOrder
        .map((id) => roomColumnMap.get(id))
        .filter((column): column is RoomColumnConfig => Boolean(column)),
    [roomColumnOrder, roomColumnMap],
  );

  const visibleRoomColumns = useMemo(
    () => roomColumnsForRender.filter((column) => roomColumnVisibility[column.id] !== false),
    [roomColumnsForRender, roomColumnVisibility],
  );

  const meterColumnsForRender = useMemo(
    () =>
      meterColumnOrder
        .map((id) => meterColumnMap.get(id))
        .filter((column): column is MeterColumnConfig => Boolean(column)),
    [meterColumnOrder, meterColumnMap],
  );

  const visibleMeterColumns = useMemo(
    () => meterColumnsForRender.filter((column) => meterColumnVisibility[column.id] !== false),
    [meterColumnsForRender, meterColumnVisibility],
  );

  const handleRoomFieldToggle = (columnId: string) => {
    setRoomColumnVisibility((prev) => ({ ...prev, [columnId]: prev[columnId] === false }));
  };

  const resetRoomFieldVisibility = () => {
    setRoomColumnVisibility(buildDefaultRoomVisibility(effectiveRoomColumnConfig));
    setRoomColumnOrder(effectiveRoomColumnConfig.map((column) => column.id));
  };

  const handleMeterFieldToggle = (columnId: MeterColumnId) => {
    setMeterColumnVisibility((prev) => ({ ...prev, [columnId]: prev[columnId] === false }));
  };

  const handleMeterExtraFieldToggle = (key: string) => {
    setMeterExtraColumnVisibility((prev) => {
      const next = { ...prev };
      if (prev[key] === false) {
        delete next[key];
      } else {
        next[key] = false;
      }
      return next;
    });
  };

  const resetMeterFieldVisibility = () => {
    setMeterColumnVisibility(buildDefaultMeterVisibility(effectiveMeterColumnConfig));
    setMeterColumnOrder(effectiveMeterColumnConfig.map((column) => column.id));
    setMeterExtraColumnVisibility({});
  };

  const refreshRoomsAndContracts = useCallback(async () => {
    if (roomContractSyncing) {
      return;
    }
    setRoomContractSyncing(true);
    try {
      const [roomData, contractData] = await Promise.all([fetchDormRooms(), fetchDormContracts()]);
      setRooms(roomData);
      setContracts(contractData);
    } catch (error) {
      console.error("[Dormitory] refresh rooms/contracts failed", error);
      toast.error("刷新房间与入住数据失败，请稍后重试");
    } finally {
      setRoomContractSyncing(false);
    }
  }, [roomContractSyncing]);

  const createBlankContractForm = () => {
    const today = formatLocalDateString(new Date());
    return {
      employee_id: "",
      employee_name: "",
      employee_department: "",
      employee_position: "",
      employee_job_number: "",
      employee_phone: "",
      employee_id_number: "",
      residence_address: "",
      emergency_contact: "",
      room_id: "",
      bed_id: "",
      start_date: today,
      end_date: "",
      rent_amount: "",
      deposit_amount: "",
      deposit_share_mode: "personal" as ShareMode,
      payment_method: "按月",
      deposit_plan_date: "",
      rent_plan_date: "",
      water_start: "",
      electric_start: "",
      gas_start: "",
      pledge_amount: "",
      pledge_plan_date: "",
      pledge_share_mode: "personal" as ShareMode,
      notes: "",
    };
  };
  type ContractFormState = ReturnType<typeof createBlankContractForm>;

  const [contractDialogOpen, setContractDialogOpen] = useState(false);
  const [contractDialogTitle, setContractDialogTitle] = useState("办理入住");
  const [contractDialogTab, setContractDialogTab] = useState<"detail" | "history">("detail");
  const [contractForm, setContractForm] = useState<ContractFormState>(createBlankContractForm);
  const [selectedContractIds, setSelectedContractIds] = useState<number[]>([]);
  const [editingContractId, setEditingContractId] = useState<number | null>(null);
  const [editingContractOriginalBedId, setEditingContractOriginalBedId] = useState<number | null>(null);
  const [contractSaving, setContractSaving] = useState(false);
  const [contractSearch, setContractSearch] = useState("");
  const [contractStatusFilter, setContractStatusFilter] = useState<"all" | "active" | "completed">("active");
  const [contractColumnOrder, setContractColumnOrder] = useState<string[]>(() => (typeof window === "undefined" ? [] : loadContractColumnOrder()));
  const [contractColumnVisibility, setContractColumnVisibility] = useState<Record<string, boolean>>(() =>
    typeof window === "undefined" ? {} : loadContractColumnVisibility(),
  );
  const [contractSort, setContractSort] = useState<ColumnSortState>(DEFAULT_CONTRACT_SORT);
  useEffect(() => {
    if (!userPreferenceReady) return;
    const chargePreferencePayload = serializeChargeConfig(defaultChargePreference) as unknown as Record<string, unknown>;
    updateUserPreferences({
      [PREF_KEY_ROOM_COLUMNS]: {
        order: roomColumnOrder,
        visibility: roomColumnVisibility,
      },
      [PREF_KEY_CONTRACT_COLUMNS]: {
        order: contractColumnOrder,
        visibility: contractColumnVisibility,
      },
      [PREF_KEY_METER_COLUMNS]: {
        order: meterColumnOrder,
        visibility: meterColumnVisibility,
      },
      [PREF_KEY_ROOM_SORT]: {
        key: roomSort.columnId,
        direction: roomSort.direction,
      },
      [PREF_KEY_CONTRACT_SORT]: {
        key: contractSort.columnId,
        direction: contractSort.direction,
      },
      [PREF_KEY_METER_SORT]: {
        key: meterSort.key,
        direction: meterSort.direction,
      },
      [PREF_KEY_ROOM_FILTERS]: {
        search: roomSearch,
        status: roomStatusFilter,
        site: roomSiteFilter,
        selectedSite: selectedSiteCardId,
        type: roomTypeFilter,
      },
      [PREF_KEY_CONTRACT_FILTERS]: {
        search: contractSearch,
        status: contractStatusFilter,
        site: contractSiteFilter,
      },
      [PREF_KEY_METER_FILTERS]: {
        search: meterSearch,
        building: meterBuildingFilter,
      },
      [PREF_KEY_CHARGE_ITEMS]: chargePreferencePayload,
    }).catch((error) => {
      console.error("[Dormitory] 保存用户偏好失败", error);
    });
  }, [
    contractColumnOrder,
    contractColumnVisibility,
    contractSearch,
    contractStatusFilter,
    contractSiteFilter,
    contractSort.columnId,
    contractSort.direction,
    meterColumnOrder,
    meterColumnVisibility,
    meterSearch,
    meterBuildingFilter,
    meterSort.direction,
    meterSort.key,
    roomColumnOrder,
    roomColumnVisibility,
    roomSearch,
    roomSiteFilter,
    roomSort.columnId,
    roomSort.direction,
    roomStatusFilter,
    roomTypeFilter,
    selectedSiteCardId,
    userPreferenceReady,
    defaultChargePreference,
  ]);
  const [draggingContractColumn, setDraggingContractColumn] = useState<string | null>(null);
  const [showContractFieldSelector, setShowContractFieldSelector] = useState(false);
  const [contractImportDialogOpen, setContractImportDialogOpen] = useState(false);
  const [contractImporting, setContractImporting] = useState(false);
  const [contractImportResult, setContractImportResult] = useState<{ inserted: number; skipped: number } | null>(null);
const createBlankCheckoutForm = () => ({
  contract_id: "",
  checkout_date: new Date().toISOString().slice(0, 10),
  inspector: "李永娇",
  water_end: "",
  electric_end: "",
  damage_report: "",
  items_status: "",
  fee_summary: "",
  deposit_collected: "",
  deposit_deduct: "",
  deposit_return: "",
  deposit_return_date: "",
  guarantee_collected: "",
  guarantee_deduct: "",
  guarantee_return: "",
  guarantee_return_date: "",
  attachments: [] as AttachmentEntry[],
});
  const [checkoutDialogOpen, setCheckoutDialogOpen] = useState(false);
  const [checkoutForm, setCheckoutForm] = useState(createBlankCheckoutForm);
  const [checkoutSubmitting, setCheckoutSubmitting] = useState(false);
  const bedPreview = useMemo(() => {
    const count = Number(roomForm.bed_count || 0);
    if (!Number.isFinite(count) || count <= 0) {
      return [] as string[];
    }
    return Array.from({ length: count }, (_, index) => `床位${index + 1}`);
  }, [roomForm.bed_count]);

  const enforceBedCountRule = useCallback((roomCategory: string, desiredCount: number) => {
    if (roomCategory === "单间") {
      if (desiredCount !== 1) {
        toast.info("单人间默认 1 个床位");
      }
      return 1;
    }
    if (!Number.isFinite(desiredCount) || desiredCount < 2) {
      toast.info("多人间床位数量不得少于 2");
      return 2;
    }
    return Math.floor(desiredCount);
  }, []);

  const handleRoomCategoryChange = (value: string) => {
    setRoomForm((prev) => ({
      ...prev,
      room_category: value,
      bed_count: enforceBedCountRule(value, Number(prev.bed_count || 0)),
    }));
  };

  const handleRoomBedCountChange = (rawValue: string) => {
    const numericValue = Number(rawValue);
    setRoomForm((prev) => ({
      ...prev,
      bed_count: enforceBedCountRule(prev.room_category, numericValue),
    }));
  };

  const contractColumnsForRender = useMemo(
    () =>
      contractColumnOrder
        .map((id) => contractColumnMap.get(id))
        .filter((column): column is ContractColumnConfig => Boolean(column)),
    [contractColumnOrder, contractColumnMap],
  );

  const visibleContractColumns = useMemo(
    () => contractColumnsForRender.filter((column) => contractColumnVisibility[column.id] !== false),
    [contractColumnsForRender, contractColumnVisibility],
  );

  useEffect(() => {
    if (!contractDialogOpen) {
      setEmployeeSearchTerm("");
      setEmployeeSuggestions([]);
      return;
    }
    if (allEmployees.length === 0 && !employeeLookupLoading) {
      setEmployeeLookupLoading(true);
      fetchEmployees()
        .then((data) => {
          setAllEmployees(data);
          setEmployeeFetchError(null);
        })
        .catch((error) => {
          console.error("[Dormitory] fetch employees failed", error);
          const message = error instanceof Error ? error.message : "加载员工列表失败";
          setEmployeeFetchError(message);
          toast.error(message);
        })
        .finally(() => setEmployeeLookupLoading(false));
    }
  }, [contractDialogOpen, allEmployees.length, employeeLookupLoading]);

  useEffect(() => {
    if (!employeeSearchTerm.trim()) {
      setEmployeeSuggestions([]);
      return;
    }
    const keyword = employeeSearchTerm.trim().toLowerCase();
    const matches = allEmployees
      .filter((employee) => {
        const name = (employee.name || "").toLowerCase();
        const idNumber = employee.id_number?.toLowerCase() ?? "";
        const employeeNo = employee.employee_id?.toLowerCase() ?? "";
        const phone = employee.phone?.toLowerCase() ?? "";
        return name.includes(keyword) || idNumber.includes(keyword) || employeeNo.includes(keyword) || phone.includes(keyword);
      })
      .slice(0, 6);
    setEmployeeSuggestions(matches);
  }, [employeeSearchTerm, allEmployees]);

  const getLastMeterReading = useCallback(
    (roomId: number, type: "electric" | "water") => {
      if (!roomId) return null;
      const sorted = meterRecords
        .filter((record) => record.room_id === roomId)
        .sort((a, b) => new Date(b.meter_date).getTime() - new Date(a.meter_date).getTime());
      if (sorted.length === 0) return null;
      const latest = sorted[0];
      return type === "electric" ? latest.electric_end : latest.water_end;
    },
    [meterRecords],
  );

  const handleMeterRoomChange = useCallback(
    (roomId: string) => {
      setMeterForm((prev) => {
        if (!roomId) {
          return { ...prev, room_id: "", electric_start: "", water_start: "" };
        }
        const numericRoomId = Number(roomId);
        const electricStart = getLastMeterReading(numericRoomId, "electric");
        const waterStart = getLastMeterReading(numericRoomId, "water");
        return {
          ...prev,
          room_id: roomId,
          electric_start: electricStart != null ? String(electricStart) : "",
          water_start: waterStart != null ? String(waterStart) : "",
        };
      });
    },
    [getLastMeterReading],
  );

  const handleMeterFormChange = (key: keyof MeterFormState, value: string) => {
    if (key === "room_id") {
      const normalized = value === SELECT_EMPTY_VALUE ? "" : value;
      handleMeterRoomChange(normalized);
      return;
    }
    if (key === "billing_month") {
      const monthValue = value;
      const range = buildNaturalMonthRange(monthValue);
      setMeterForm((prev) => ({
        ...prev,
        billing_month: monthValue,
        billing_start: range?.start ?? prev.billing_start,
        billing_end: range?.end ?? prev.billing_end,
      }));
      return;
    }
    setMeterForm((prev) => ({ ...prev, [key]: value }));
  };

  const handleResetMeterForm = (options?: {
    preserveBuilding?: boolean;
    preserveContext?: boolean;
    preserveSite?: boolean;
    nextRoomId?: string;
    nextBillingEnd?: string | null;
    overrideBillingMonth?: string | null;
  }) => {
    setMeterForm((prev) => {
      const overrideMonth = options?.overrideBillingMonth && options.overrideBillingMonth !== "all" ? options.overrideBillingMonth : "";
      const overrideRange = overrideMonth ? buildNaturalMonthRange(overrideMonth) : null;
      if (options?.preserveContext) {
        const nextForm: MeterFormState = {
          ...createBlankMeterForm(),
          billing_month: prev.billing_month,
          room_id: options?.nextRoomId ?? "",
          meter_date: prev.meter_date,
          inspector: prev.inspector,
          billing_start: prev.billing_start,
          billing_end: options?.nextBillingEnd ?? prev.billing_end,
        };
        if (overrideMonth) {
          return {
            ...nextForm,
            billing_month: overrideMonth,
            billing_start: overrideRange?.start ?? nextForm.billing_start,
            billing_end: overrideRange?.end ?? nextForm.billing_end,
          };
        }
        return nextForm;
      }
      const blank = createBlankMeterForm();
      if (!overrideMonth) {
        return blank;
      }
      return {
        ...blank,
        billing_month: overrideMonth,
        billing_start: overrideRange?.start ?? blank.billing_start,
        billing_end: overrideRange?.end ?? blank.billing_end,
      };
    });
    if (!options?.preserveContext) {
      setMeterExtraChargeInputs({});
    }
    setMeterFormMode("create");
    setEditingMeterId(null);
    if (!options?.preserveSite) {
      const defaultSiteId = sites.length > 0 ? String(sites[0].id) : "";
      setMeterFormSiteId(defaultSiteId);
    }
    if (!options?.preserveBuilding) {
      setMeterFormBuildingId("all");
    }
    setMeterMemberRatios({});
    setMeterMemberChargeOverrides({});
    setMeterExtraChargeToggles({ ...DEFAULT_EXTRA_CHARGE_TOGGLES });
  };

  const handleSubmitMeterForm = async (options?: { stayOpen?: boolean }) => {
    if (!meterForm.room_id) {
      toast.error("请选择房间");
      return;
    }
    const roomId = Number(meterForm.room_id);
    if (!roomById.get(roomId)) {
      toast.error("未找到对应房间");
      return;
    }

    const parseOptionalNumber = (value: string) => {
      if (!value.trim()) return null;
      const parsed = Number(value);
      return Number.isFinite(parsed) ? parsed : null;
    };

    const electricStart = parseOptionalNumber(meterForm.electric_start);
    const electricEnd = parseOptionalNumber(meterForm.electric_end);
    const waterStart = parseOptionalNumber(meterForm.water_start);
    const waterEnd = parseOptionalNumber(meterForm.water_end);
    const gasStart = parseOptionalNumber(meterForm.gas_start);
    const gasEnd = parseOptionalNumber(meterForm.gas_end);
    const chargeDetails: DormChargeDetail[] = [];
    const useMemberDistribution = selectedMeterRoom?.cost_bearing_mode === "personal" && meterMemberSharePreview.length > 0;
    const participantsForCharge = (chargeKey: string, totalAmount: number | null) => {
      if (!useMemberDistribution || totalAmount == null || meterMemberSharePreview.length === 0) return undefined;
      const participants = meterMemberSharePreview
        .map((row) => {
          const value = row.resolvedCharges[chargeKey] ?? null;
          if (value == null) return null;
          return {
            name: row.name,
            contract_id: row.contractId,
            amount: Number(value.toFixed(2)),
            ratio: row.ratioNormalized,
          };
        })
        .filter(Boolean) as Array<{ name: string; contract_id?: number; amount: number; ratio: number }>;
      return participants.length > 0 ? participants : undefined;
    };

    const addMeterCharge = (key: LegacyChargeKey, label: string, start: number | null, end: number | null) => {
      if (!meterChargeSettingMap.has(key)) return;
      const setting = meterChargeSettingMap.get(key);
      const unitPrice = setting?.unitPrice ?? 0;
      const usage = start != null && end != null ? Math.max(0, end - start) : null;
      const amount = usage != null && unitPrice > 0 ? Number((usage * unitPrice).toFixed(2)) : null;
      chargeDetails.push({
        key,
        label: setting?.label || label,
        start: start ?? undefined,
        end: end ?? undefined,
        usage: usage ?? undefined,
        unit_price: unitPrice || undefined,
        unit_label: setting?.unitLabel,
        amount: amount ?? undefined,
        participants: participantsForCharge(key, amount ?? null),
        mode: setting?.mode ?? "meter",
      });
    };

    addMeterCharge("electric", LEGACY_CHARGE_LABELS.electric, electricStart, electricEnd);
    addMeterCharge("water", LEGACY_CHARGE_LABELS.water, waterStart, waterEnd);
    addMeterCharge("gas", LEGACY_CHARGE_LABELS.gas, gasStart, gasEnd);
    supplementalChargesForCurrentRoom.forEach((item) => {
      const amount = getResolvedSupplementalAmount(item.key);
      if (amount == null) return;
      chargeDetails.push({
        key: item.key,
        label: item.label,
        unit_price: item.unitPrice,
        unit_label: item.unitLabel,
        amount,
        participants: participantsForCharge(item.key, amount),
        mode: item.mode,
      });
    });

    const payload: DormMeterRecordPayload = {
      room_id: roomId,
      meter_date: meterForm.meter_date,
      billing_start: meterForm.billing_start,
      billing_end: meterForm.billing_end,
      inspector: meterForm.inspector || "李永娇",
      charge_details: chargeDetails,
    };

    try {
      let saved: DormMeterRecord;
      if (meterFormMode === "edit" && editingMeterId != null) {
        saved = await updateDormMeterRecord(editingMeterId, payload);
      } else {
        saved = await createDormMeterRecord(payload);
      }

      let recordedRoomIdsAfterSave = new Set<number>();
      setMeterSourceRecords((prev) => {
        let next: DormMeterRecord[];
        if (meterFormMode === "edit" && editingMeterId != null) {
          next = prev.map((record) => (record.id === saved.id ? saved : record));
        } else {
          next = [saved, ...prev];
        }
        const currentPeriodKey =
          meterForm.billing_month && meterForm.billing_month !== "all"
            ? meterForm.billing_month
            : derivePeriodKeyFromDate(saved.billing_end || saved.billing_start || meterForm.billing_month);
        const periodFiltered = currentPeriodKey
          ? next.filter((record) => {
              const recordPeriod = record.billing_month || derivePeriodKeyFromDate(record.billing_end || record.billing_start || record.meter_date);
              return recordPeriod === currentPeriodKey;
            })
          : next;
        recordedRoomIdsAfterSave = new Set(periodFiltered.map((record) => record.room_id));
        return next;
      });

      toast.success(meterFormMode === "edit" ? "抄表记录已更新" : "抄表记录已保存");
      if (options?.stayOpen) {
        const nextRoomId = getNextSequentialRoomId(roomId, recordedRoomIdsAfterSave);
        if (nextRoomId) {
          setMeterSequentialIndex(meterSequentialRooms.indexOf(nextRoomId));
          handleResetMeterForm({
            preserveBuilding: true,
            preserveContext: true,
            preserveSite: true,
            nextRoomId: String(nextRoomId),
            nextBillingEnd: meterForm.billing_end,
          });
          return;
        }
        toast.info("当前楼栋已完成全部房间抄表");
      }
      handleResetMeterForm();
      setMeterDialogOpen(false);
    } catch (error) {
      console.error("[Dormitory] save meter record failed", error);
      toast.error(error instanceof Error ? error.message : "保存抄表记录失败");
    }
  };

  const handleEditMeterRecord = (record: MeterRecord) => {
    setMeterFormMode("edit");
    setEditingMeterId(record.id);
    setMeterForm({
      room_id: String(record.room_id),
      billing_month: derivePeriodKeyFromDate(record.billing_end || record.billing_start || meterForm.billing_month) || meterForm.billing_month,
      meter_date: record.meter_date,
      inspector: record.inspector,
      billing_start: record.billing_start || meterForm.billing_start,
      billing_end: record.billing_end || meterForm.billing_end,
      electric_start: record.electric_start != null ? String(record.electric_start) : "",
      electric_end: record.electric_end != null ? String(record.electric_end) : "",
      water_start: record.water_start != null ? String(record.water_start) : "",
      water_end: record.water_end != null ? String(record.water_end) : "",
      gas_start: record.gas_start != null ? String(record.gas_start) : "",
      gas_end: record.gas_end != null ? String(record.gas_end) : "",
    });
    const extras: Record<string, string> = {};
    record.charges.forEach((charge) => {
      if (!isBaseChargeKey(charge.key) && charge.amount != null) {
        extras[charge.key] = String(charge.amount);
      }
    });
    setMeterExtraChargeInputs(extras);
    const matchedRoom = roomById.get(record.room_id);
    const siteId = getRoomSiteId(matchedRoom);
    setMeterFormSiteId(siteId ? String(siteId) : "");
    setMeterFormBuildingId(matchedRoom?.building_id ?? "all");
    setMeterDialogOpen(true);
  };

  const deleteMeterRecords = async (recordIds: number[]) => {
    await Promise.all(recordIds.map((id) => deleteDormMeterRecord(id)));
    setMeterSourceRecords((prev) => prev.filter((record) => !recordIds.includes(record.id)));
    setMeterSelectedIds((prev) => prev.filter((id) => !recordIds.includes(id)));
  };

  const handleMeterSelectAll = (checked: boolean, rows: MeterTableRow[]) => {
    const ids = rows.map((row) => row.source.id);
    setMeterSelectedIds((prev) => {
      if (checked) {
        return Array.from(new Set([...prev, ...ids]));
      }
      return prev.filter((id) => !ids.includes(id));
    });
  };

  const toggleMeterSelection = (recordId: number) => {
    setMeterSelectedIds((prev) => (prev.includes(recordId) ? prev.filter((id) => id !== recordId) : [...prev, recordId]));
  };

  const handleMeterSortClick = (columnId: MeterColumnId) => {
    setMeterSort((prev) => {
      if (prev.key === columnId) {
        return { key: columnId, direction: prev.direction === "asc" ? "desc" : "asc" };
      }
      return { key: columnId, direction: "asc" };
    });
  };

  const requestDeleteMeterRecord = (record: MeterRecord) => {
    setMeterDeleteContext({ mode: "single", record });
  };

  const requestBulkDeleteMeterRecords = () => {
    if (meterSelectedIds.length === 0) {
      toast.error("请先勾选需要删除的抄表记录");
      return;
    }
    setMeterDeleteContext({ mode: "bulk" });
  };

  const handleConfirmDeleteMeterRecords = async () => {
    if (!meterDeleteContext) return;
    try {
      if (meterDeleteContext.mode === "single" && meterDeleteContext.record) {
        await deleteMeterRecords([meterDeleteContext.record.id]);
        toast.success("抄表记录已删除");
      } else if (meterDeleteContext.mode === "bulk") {
        await deleteMeterRecords(meterSelectedIds);
        toast.success("已删除选中的抄表记录");
      }
    } catch (error) {
      console.error("[Dormitory] delete meter record failed", error);
      toast.error(error instanceof Error ? error.message : "删除抄表记录失败");
    } finally {
      setMeterDeleteContext(null);
    }
  };

  const handleOpenRoomDialog = () => {
    setEditingRoomId(null);
    const summary = activeSiteId ? siteInventorySummaryMap.get(activeSiteId) ?? "" : "";
    setRoomForm(() => ({
      ...initialRoomForm,
      site_id: activeSiteId ? String(activeSiteId) : "",
      inventory_note: mergeInventorySummaryWithNote(summary, initialRoomForm.inventory_note),
    }));
    setRoomChargeOverrides({});
    setRoomChargeRecords({});
    setRoomDialogTab("detail");
    setRoomDialogOpen(true);
  };

  const handleEditRoom = (room: DormRoom) => {
    const building = buildingById.get(room.building_id);
    const derivedSiteId = room.site_id ?? building?.site_id ?? null;
    const inventorySummary = derivedSiteId ? siteInventorySummaryMap.get(derivedSiteId) ?? "" : "";
    setEditingRoomId(room.id);
    setRoomForm({
      site_id: String(room.site_id ?? building?.site_id ?? ""),
      building_id: String(room.building_id || ""),
      room_number: room.room_number || "",
      room_category: room.room_category || "",
      house_layout: room.house_layout || "一室一厅",
      bed_count: Number(room.bed_count || room.beds?.length || 1),
      area_square: room.area_square ? String(room.area_square) : "",
      first_month_fee: room.first_month_fee ? String(room.first_month_fee) : "",
      monthly_rent: room.monthly_rent ? String(room.monthly_rent) : "",
      property_fee: room.property_fee ? String(room.property_fee) : "",
      quarterly_rent: room.quarterly_rent ? String(room.quarterly_rent) : "",
      guarantee_fee: room.guarantee_fee ? String(room.guarantee_fee) : "",
      deposit_fee: room.deposit_fee ? String(room.deposit_fee) : "",
      water_base: room.water_base ? String(room.water_base) : "",
      electric_base: room.electric_base ? String(room.electric_base) : "",
      gas_base: room.gas_base ? String(room.gas_base) : "",
      trash_fee: room.trash_fee ? String(room.trash_fee) : "",
      water_supply_fee: room.water_supply_fee ? String(room.water_supply_fee) : "",
      sewage_fee: room.sewage_fee ? String(room.sewage_fee) : "",
      inventory_note: mergeInventorySummaryWithNote(inventorySummary, room.inventory_note || ""),
      status: room.status === "维护中" ? "维护中" : "",
      cost_bearing_mode: (room.cost_bearing_mode as ShareMode) || "company",
      company_name: room.company_name || "",
    });
    setRoomChargeOverrides(buildRoomChargeOverridesFromRoom(room));
    setRoomChargeRecords(parseRoomRecordNotes(room.notes));
    setRoomDialogTab("detail");
    setRoomDialogOpen(true);
  };

  useEffect(() => {
    const load = async () => {
      try {
        setLoading(true);
        const [siteData, buildingData, roomData, contractData, billData, meterData] = await Promise.all([
          fetchDormSites(),
          fetchDormBuildings(),
          fetchDormRooms(),
          fetchDormContracts(),
          fetchDormBills({ status: "pending" }),
          fetchDormMeterRecords(),
        ]);
        setSites(siteData);
        setBuildings(buildingData);
        setRooms(roomData);
        setContracts(contractData);
        setBills(billData);
        setMeterSourceRecords(meterData);
        if (siteData.length > 0) {
          setActiveSiteId(siteData[0].id);
        }
      } catch (error) {
        console.error("[Dormitory] load failed", error);
        toast.error(error instanceof Error ? error.message : "加载宿舍信息失败");
      } finally {
        setLoading(false);
      }
    };
    load();
  }, []);

  const siteBuildingMap = useMemo(() => {
    const map: Record<number, DormBuilding[]> = {};
    for (const building of buildings) {
      if (!map[building.site_id]) {
        map[building.site_id] = [];
      }
      map[building.site_id].push(building);
    }
    return map;
  }, [buildings]);

  const roomSiteLookupRef = useRef<Map<number, number>>(new Map());
  const siteCardClickTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const siteCardRefs = useRef<Map<number, HTMLDivElement | null>>(new Map());
  useEffect(() => () => {
    if (siteCardClickTimerRef.current) {
      clearTimeout(siteCardClickTimerRef.current);
    }
  }, []);

  const roomById = useMemo(() => {
    const map = new Map<number, DormRoom>();
    rooms.forEach((room) => map.set(room.id, room));
    return map;
  }, [rooms]);
  useEffect(() => {
    const lookup = new Map<number, number>();
    rooms.forEach((room) => {
      const siteId = room.site_id ?? buildingById.get(room.building_id)?.site_id ?? null;
      if (siteId) {
        lookup.set(room.id, siteId);
      }
    });
    roomSiteLookupRef.current = lookup;
  }, [rooms, buildingById]);


  useEffect(() => {
    if (roomColumnOrder.length === 0 && effectiveRoomColumnConfig.length > 0) {
      setRoomColumnOrder(effectiveRoomColumnConfig.map((column) => column.id));
    }
  }, [roomColumnOrder.length, effectiveRoomColumnConfig]);

  useEffect(() => {
    if (roomColumnOrder.length === 0) return;
    const validIds = new Set(effectiveRoomColumnConfig.map((column) => column.id));
    setRoomColumnOrder((prev) => {
      const filtered = prev.filter((id) => validIds.has(id));
      const missing = effectiveRoomColumnConfig.map((column) => column.id).filter((id) => !filtered.includes(id));
      if (missing.length === 0 && filtered.length === prev.length) {
        return prev;
      }
      return [...filtered, ...missing];
    });
  }, [effectiveRoomColumnConfig, roomColumnOrder.length]);

  useEffect(() => {
    if (typeof window === "undefined") return;
    if (roomColumnOrder.length === 0) {
      localStorage.removeItem(ROOM_COLUMN_ORDER_STORAGE_KEY);
      return;
    }
    writeLocalStorageJSON(ROOM_COLUMN_ORDER_STORAGE_KEY, roomColumnOrder);
  }, [roomColumnOrder]);

  useEffect(() => {
    if (meterColumnOrder.length === 0 && effectiveMeterColumnConfig.length > 0) {
      setMeterColumnOrder(effectiveMeterColumnConfig.map((column) => column.id));
    }
  }, [meterColumnOrder.length, effectiveMeterColumnConfig]);

  useEffect(() => {
    if (meterColumnOrder.length === 0) return;
    const validIds = new Set(effectiveMeterColumnConfig.map((column) => column.id));
    setMeterColumnOrder((prev) => {
      const filtered = prev.filter((id) => validIds.has(id));
      const missing = effectiveMeterColumnConfig.map((column) => column.id).filter((id) => !filtered.includes(id));
      if (missing.length === 0 && filtered.length === prev.length) {
        return prev;
      }
      return [...filtered, ...missing];
    });
  }, [effectiveMeterColumnConfig, meterColumnOrder.length]);

  useEffect(() => {
    if (typeof window === "undefined") return;
    if (meterColumnOrder.length === 0) {
      localStorage.removeItem(METER_COLUMN_ORDER_STORAGE_KEY);
      return;
    }
    writeLocalStorageJSON(METER_COLUMN_ORDER_STORAGE_KEY, meterColumnOrder);
  }, [meterColumnOrder]);

  const meterFormBuildingOptions = useMemo(() => {
    if (!meterFormSiteId) {
      return [] as DormBuilding[];
    }
    const siteId = Number(meterFormSiteId);
    if (!Number.isFinite(siteId)) return [];
    return buildings.filter((building) => building.site_id === siteId);
  }, [meterFormSiteId, buildings]);

  const roomOccupancyMap = useMemo(() => {
    const map = new Map<number, RoomOccupancyMeta>();
    contracts
      .filter((contract) => contract.status === "active" && contract.room_id)
      .forEach((contract) => {
        const existing = map.get(contract.room_id!) ?? { names: [], count: 0, bedAssignments: {}, members: [] };
        const memberName = contract.employee_name || "未命名";
        const next: RoomOccupancyMeta = {
          names: [...existing.names, memberName],
          count: existing.count + 1,
          bedAssignments: { ...existing.bedAssignments },
          members: [...existing.members, { contractId: contract.id, name: memberName }],
        };
        if (contract.bed_id) {
          next.bedAssignments[contract.bed_id] = contract.employee_name || "未命名";
        }
        map.set(contract.room_id!, next);
      });
    return map;
  }, [contracts]);

  const roomContractPeriodsMap = useMemo(() => {
    const map = new Map<number, Array<{ start: string; end: string | null }>>();
    contracts.forEach((contract) => {
      if (!contract.room_id || !contract.start_date) return;
      const status = (contract.status || "active").toLowerCase();
      if (status === "cancelled") return;
      const normalizedStart = contract.start_date.slice(0, 10);
      const normalizedEnd = contract.end_date ? contract.end_date.slice(0, 10) : null;
      const existing = map.get(contract.room_id) ?? [];
      existing.push({ start: normalizedStart, end: normalizedEnd });
      map.set(contract.room_id, existing);
    });
    return map;
  }, [contracts]);

  const meterRecordedRoomsByPeriod = useMemo(() => {
    const map = new Map<string, Set<number>>();
    meterRecords.forEach((record) => {
      const key = record.billing_month || derivePeriodKeyFromDate(record.billing_end || record.billing_start || record.meter_date);
      if (!key) return;
      if (!map.has(key)) {
        map.set(key, new Set());
      }
      map.get(key)!.add(record.room_id);
    });
    return map;
  }, [meterRecords]);

  useEffect(() => {
    if (!meterFormSiteId) {
      if (meterFormBuildingId !== "all") {
        setMeterFormBuildingId("all");
      }
      handleMeterRoomChange("");
      return;
    }
    if (meterFormBuildingOptions.length === 0) {
      if (meterFormBuildingId !== "all") {
        setMeterFormBuildingId("all");
      }
      handleMeterRoomChange("");
      return;
    }
    if (meterFormBuildingId === "all" || !meterFormBuildingOptions.some((building) => building.id === meterFormBuildingId)) {
      setMeterFormBuildingId(meterFormBuildingOptions[0].id);
      handleMeterRoomChange("");
    }
  }, [meterFormSiteId, meterFormBuildingId, meterFormBuildingOptions, handleMeterRoomChange]);

  const meterFormRoomOptions = useMemo(() => {
    if (!meterFormSiteId || meterFormBuildingId === "all") {
      return [] as DormRoom[];
    }
    const siteId = Number(meterFormSiteId);
    if (!Number.isFinite(siteId)) return [];
    const recordedInPeriod = meterRecordedRoomsByPeriod.get(meterForm.billing_month ?? "") ?? null;
    const periodRange = derivePeriodRangeFromKey(meterForm.billing_month ?? "");
    const currentRoomId = meterForm.room_id ? Number(meterForm.room_id) : null;
    const allowEditingRoom = meterFormMode === "edit" && editingMeterId != null && currentRoomId != null;
    const scopedRooms = rooms.filter((room) => {
      if (room.building_id !== meterFormBuildingId) return false;
      if (getRoomSiteId(room) !== siteId) return false;
      if (allowEditingRoom && room.id === currentRoomId) {
        return true;
      }
      if (room.status === "维护中") return false;
      const hasEligibleContracts = (() => {
        const segments = roomContractPeriodsMap.get(room.id) ?? [];
        if (periodRange) {
          const overlap = segments.some((segment) => {
            if (!segment.start) return false;
            const endBoundary = segment.end ?? "9999-12-31";
            return segment.start <= periodRange.end && endBoundary >= periodRange.start;
          });
          if (overlap) return true;
          const occupancy = roomOccupancyMap.get(room.id);
          return (occupancy?.count ?? 0) > 0;
        }
        if (segments.length > 0) return true;
        const occupancy = roomOccupancyMap.get(room.id);
        return (occupancy?.count ?? 0) > 0;
      })();
      if (!hasEligibleContracts) return false;
      if (recordedInPeriod?.has(room.id)) return false;
      return true;
    });
    return [...scopedRooms].sort((a, b) => (a.room_number || "").localeCompare(b.room_number || "", "zh-CN"));
  }, [
    rooms,
    meterFormBuildingId,
    meterFormSiteId,
    getRoomSiteId,
    meterForm.billing_month,
    meterRecordedRoomsByPeriod,
    meterFormMode,
    editingMeterId,
    meterForm.room_id,
    roomOccupancyMap,
    roomContractPeriodsMap,
  ]);

  useEffect(() => {
    if (meterFormBuildingId === "all") {
      if (meterForm.room_id) {
        handleMeterRoomChange("");
      }
      return;
    }
    if (meterFormRoomOptions.length === 0) {
      if (meterForm.room_id) {
        handleMeterRoomChange("");
      }
      return;
    }
    if (!meterFormRoomOptions.some((room) => String(room.id) === meterForm.room_id)) {
      handleMeterRoomChange(String(meterFormRoomOptions[0].id));
    }
  }, [meterFormBuildingId, meterFormRoomOptions, meterForm.room_id, handleMeterRoomChange]);

  useEffect(() => {
    if (meterFormBuildingId === "all" || !meterFormSiteId) {
      setMeterSequentialRooms([]);
      setMeterSequentialIndex(0);
      return;
    }
    setMeterSequentialRooms(meterFormRoomOptions.map((room) => room.id));
    setMeterSequentialIndex(0);
  }, [meterFormRoomOptions, meterFormBuildingId, meterFormSiteId]);

  const getNextSequentialRoomId = useCallback(
    (currentRoomId: number, recordedIds: Set<number>) => {
      if (!meterSequentialRooms.length) return null;
      let startIndex = meterSequentialRooms.indexOf(currentRoomId);
      if (startIndex === -1) {
        startIndex = meterSequentialIndex;
      }
      for (let offset = 1; offset <= meterSequentialRooms.length; offset += 1) {
        const idx = (startIndex + offset) % meterSequentialRooms.length;
        const candidateId = meterSequentialRooms[idx];
        if (!recordedIds.has(candidateId)) {
          return candidateId;
        }
      }
      return null;
    },
    [meterSequentialRooms, meterSequentialIndex],
  );


  const siteStats = useMemo(() => {
    return sites.map((site) => {
      const buildingList = siteBuildingMap[site.id] || [];
      const roomList = rooms.filter(
        (room) => room.site_id === site.id || buildingList.some((building) => building.id === room.building_id),
      );
      const roomIds = new Set(roomList.map((room) => room.id));
      const activeContractsForSite = contracts.filter(
        (contract) => contract.status === "active" && roomIds.has(contract.room_id),
      );
      const occupiedRoomIds = new Set(activeContractsForSite.map((contract) => contract.room_id));
      const totalBeds = roomList.reduce((sum, room) => sum + (room.beds?.length || room.bed_count || 0), 0);
      return {
        ...site,
        totalRooms: roomList.length,
        freeRooms: Math.max(roomList.length - occupiedRoomIds.size, 0),
        totalBeds,
        freeBeds: Math.max(totalBeds - activeContractsForSite.length, 0),
        tenants: activeContractsForSite.length,
      };
    });
  }, [sites, rooms, contracts, siteBuildingMap]);

  const orderedSiteStats = useMemo(() => {
    if (siteStats.length === 0) return [] as (DormSite & { totalRooms: number; freeRooms: number; totalBeds: number; freeBeds: number; tenants: number })[];
    if (siteOrder.length === 0) return siteStats;
    const rank = new Map(siteOrder.map((id, index) => [id, index]));
    return [...siteStats].sort((a, b) => (rank.get(a.id) ?? Number.MAX_SAFE_INTEGER) - (rank.get(b.id) ?? Number.MAX_SAFE_INTEGER));
  }, [siteStats, siteOrder]);

  const supportWechatConfigs = useMemo(() => {
    return orderedSiteStats
      .filter((item) => item.support_wechat && item.support_wechat.trim().length > 0)
      .map((item) => ({ siteId: item.id, siteName: item.name, value: item.support_wechat!.trim() }));
  }, [orderedSiteStats]);

  const primaryWechatConfig = supportWechatConfigs[0] ?? null;

  const activeWechatConfig = useMemo(() => {
    if (supportWechatConfigs.length === 0) return null;
    if (activeWechatSiteId == null) return supportWechatConfigs[0];
    return supportWechatConfigs.find((config) => config.siteId === activeWechatSiteId) ?? supportWechatConfigs[0];
  }, [supportWechatConfigs, activeWechatSiteId]);

  useEffect(() => {
    setSelectedSiteCardId(roomSiteFilter);
  }, [roomSiteFilter]);

  useEffect(() => {
    if (!wechatDialogOpen) {
      setActiveWechatSiteId(null);
      return;
    }
    if (supportWechatConfigs.length > 0 && activeWechatSiteId == null) {
      setActiveWechatSiteId(supportWechatConfigs[0].siteId);
    }
  }, [wechatDialogOpen, supportWechatConfigs, activeWechatSiteId]);

  const enrichContractRelations = useCallback(
    (contract: DormContract) => {
      const room = contract.room_id ? roomById.get(contract.room_id) ?? contract.room : contract.room;
      const bed =
        contract.bed_id && room?.beds?.length
          ? room.beds.find((bedItem: DormBed) => bedItem.id === contract.bed_id) ?? contract.bed
          : contract.bed;
      if (room === contract.room && bed === contract.bed) {
        return contract;
      }
      return { ...contract, room, bed };
    },
    [roomById],
  );


  const cloneRecordForParticipant = useCallback(
    (record: MeterRecord, participantName: string, chargeAmounts: Record<string, number | null>) => {
      const cloned: MeterRecord = {
        ...record,
        occupants: [participantName],
        electric_fee: chargeAmounts.electric ?? null,
        water_fee: chargeAmounts.water ?? null,
        gas_fee: chargeAmounts.gas ?? null,
        charges: record.charges.map((charge) => ({
          ...charge,
          amount: chargeAmounts[charge.key] ?? null,
        })),
      };
      return cloned;
    },
    [],
  );

  const buildPersonalMeterRows = useCallback(
    (records: MeterRecord[]): MeterTableRow[] => {
      const rows: MeterTableRow[] = [];
      records.forEach((record) => {
        const occupancy = roomOccupancyMap.get(record.room_id);
        const participantMap = new Map<string, { name: string; contractId?: number; amounts: Record<string, number | null> }>();
        record.charges.forEach((charge) => {
          if (!charge.participants?.length) return;
          charge.participants.forEach((participant) => {
            if (!participant) return;
            const key = participant.contractId ? `contract-${participant.contractId}` : `${participant.name || "member"}-${record.id}`;
            const existing = participantMap.get(key) || { name: participant.name || "未命名", contractId: participant.contractId, amounts: {} };
            existing.amounts[charge.key] = participant.amount ?? null;
            participantMap.set(key, existing);
          });
        });
        if (participantMap.size === 0) {
          const occupantNames = occupancy?.names?.length ? occupancy.names : record.occupants;
          const count = occupantNames.length || 1;
          occupantNames.forEach((name, index) => {
            const amounts: Record<string, number | null> = {};
            record.charges.forEach((charge) => {
              const shared = charge.amount != null ? Number((charge.amount / count).toFixed(2)) : null;
              amounts[charge.key] = shared;
            });
            participantMap.set(`fallback-${record.id}-${index}`, { name, contractId: undefined, amounts });
          });
        }
        Array.from(participantMap.values()).forEach((participant, index) => {
          const display = cloneRecordForParticipant(record, participant.name, participant.amounts);
          rows.push({
            key: `personal-${record.id}-${participant.contractId ?? index}`,
            display,
            source: record,
          });
        });
      });
      return rows;
    },
    [roomOccupancyMap, cloneRecordForParticipant],
  );

  const buildCompanyMeterRows = useCallback(
    (records: MeterRecord[]): MeterTableRow[] => records.map((record) => ({ key: `company-${record.id}`, display: record, source: record })),
    [],
  );

  useEffect(() => {
    setContracts((prev) => {
      if (prev.length === 0) return prev;
      let changed = false;
      const next = prev.map((contract) => {
        const enriched = enrichContractRelations(contract);
        if (enriched !== contract) {
          changed = true;
        }
        return enriched;
      });
      return changed ? next : prev;
    });
  }, [enrichContractRelations]);

  const getChargeSettingsForRoom = useCallback(
    (room?: DormRoom | null) => {
      if (!room) {
        return fallbackChargeSettings;
      }
      const building = buildingById.get(room.building_id);
      const siteId = room.site_id ?? building?.site_id;
      const siteCharges = siteId ? siteChargeConfigMap.get(siteId) : undefined;
      const base = siteCharges && siteCharges.length > 0 ? siteCharges : fallbackChargeSettings;
      return mergeSiteRoomCharges(base, room.charge_rates as DormChargeRates | undefined);
    },
    [buildingById, siteChargeConfigMap, fallbackChargeSettings],
  );

  const selectedMeterRoom = useMemo(() => {
    if (!meterForm.room_id) return null;
    const numericId = Number(meterForm.room_id);
    if (!Number.isFinite(numericId)) return null;
    return roomById.get(numericId) ?? null;
  }, [meterForm.room_id, roomById]);

  const currentMeterChargeSettings = useMemo(() => {
    if (selectedMeterRoom) {
      return getChargeSettingsForRoom(selectedMeterRoom);
    }
    return fallbackChargeSettings;
  }, [selectedMeterRoom, getChargeSettingsForRoom, fallbackChargeSettings]);

  const supplementalChargesForCurrentRoom = useMemo(
    () => currentMeterChargeSettings.filter((item) => !BASE_METER_KEYS.includes(item.key as LegacyChargeKey)),
    [currentMeterChargeSettings],
  );

  const meterChargeSettingMap = useMemo(() => {
    const map = new Map<string, ChargeSetting>();
    currentMeterChargeSettings.forEach((item) => map.set(item.key, item));
    return map;
  }, [currentMeterChargeSettings]);

  const editingSiteMemoEntries = useMemo(() => {
    if (!editingSite) return [] as SiteMemoEntry[];
    const entries = (siteMemos[memoKey(editingSite.id)] ?? []).filter(
      (memo): memo is SiteMemoEntry => Boolean(memo && (memo.date || memo.targetDate || memo.createdAt)),
    );
    return selectPrimaryAutoMemos(entries);
  }, [editingSite, siteMemos, memoKey]);

  const editingSiteMemoSections = useMemo(() => {
    const todo: SiteMemoEntry[] = [];
    const completed: SiteMemoEntry[] = [];
    editingSiteMemoEntries.forEach((memo) => {
      if (!memo) return;
      if (memo.completed) {
        completed.push(memo);
        return;
      }
      if (!isAutoGeneratedMemo(memo) || shouldDisplayMemoNow(memo)) {
        todo.push(memo);
      }
    });
    const sortByOccurrenceDesc = (list: SiteMemoEntry[]) =>
      [...list].sort((a, b) => {
        const aTime = getNextMemoOccurrence(a)?.getTime() ?? 0;
        const bTime = getNextMemoOccurrence(b)?.getTime() ?? 0;
        return bTime - aTime;
      });
    const sortByCompletedDesc = (list: SiteMemoEntry[]) =>
      [...list].sort((a, b) => {
        const aTime = new Date(a.completedAt || a.date).getTime();
        const bTime = new Date(b.completedAt || b.date).getTime();
        return bTime - aTime;
      });
    return { todo: sortByOccurrenceDesc(todo), completed: sortByCompletedDesc(completed) };
  }, [editingSiteMemoEntries]);

  const selectedMeterOccupancy = useMemo(
    () => (selectedMeterRoom ? roomOccupancyMap.get(selectedMeterRoom.id) ?? EMPTY_OCCUPANCY : undefined),
    [selectedMeterRoom, roomOccupancyMap],
  );

  useEffect(() => {
    const extras = currentMeterChargeSettings.filter((item) => !BASE_METER_KEYS.includes(item.key as LegacyChargeKey));
    setMeterExtraChargeInputs((prev) => {
      const next: Record<string, string> = {};
      extras.forEach((item) => {
        next[item.key] = prev[item.key] ?? "";
      });
      return next;
    });
  }, [currentMeterChargeSettings]);

  useEffect(() => {
    if (meterFormBuildingId === "all") return;
    if (!meterForm.room_id) return;
    const numericId = Number(meterForm.room_id);
    if (!Number.isFinite(numericId)) return;
    const matchedRoom = roomById.get(numericId);
    if (matchedRoom && matchedRoom.building_id !== meterFormBuildingId) {
      setMeterForm((prev) => ({ ...prev, room_id: "" }));
    }
  }, [meterFormBuildingId, meterForm.room_id, roomById]);

  const meterFormPreview = useMemo(() => {
    const parseValue = (value: string) => {
      if (!value.trim()) return null;
      const parsed = Number(value);
      return Number.isFinite(parsed) ? parsed : null;
    };
    const electricStart = parseValue(meterForm.electric_start);
    const electricEnd = parseValue(meterForm.electric_end);
    const waterStart = parseValue(meterForm.water_start);
    const waterEnd = parseValue(meterForm.water_end);
    const gasStart = parseValue(meterForm.gas_start);
    const gasEnd = parseValue(meterForm.gas_end);
    const occupancy = selectedMeterOccupancy ?? EMPTY_OCCUPANCY;
    const shareRatio = computeShareRatio(selectedMeterRoom ?? undefined, occupancy);
    const electricUsage = electricStart != null && electricEnd != null ? Math.max(0, electricEnd - electricStart) : null;
    const waterUsage = waterStart != null && waterEnd != null ? Math.max(0, waterEnd - waterStart) : null;
    const gasUsage = gasStart != null && gasEnd != null ? Math.max(0, gasEnd - gasStart) : null;
    const electricRate =
      meterChargeSettingMap.get("electric")?.unitPrice ??
      (selectedMeterRoom?.electric_base ? Number(selectedMeterRoom.electric_base) : 0);
    const waterRate =
      meterChargeSettingMap.get("water")?.unitPrice ??
      (selectedMeterRoom?.water_base ? Number(selectedMeterRoom.water_base) : 0);
    const gasRate =
      meterChargeSettingMap.get("gas")?.unitPrice ??
      (selectedMeterRoom?.gas_base ? Number(selectedMeterRoom.gas_base) : 0);
    const electricFee = electricUsage != null && electricRate > 0 ? Number((electricUsage * electricRate).toFixed(2)) : null;
    const waterFee = waterUsage != null && waterRate > 0 ? Number((waterUsage * waterRate).toFixed(2)) : null;
    const gasFee = gasUsage != null && gasRate > 0 ? Number((gasUsage * gasRate).toFixed(2)) : null;
    return {
      shareRatio,
      occupantCount: occupancy.count,
      electricUsage,
      waterUsage,
      electricFee,
      waterFee,
      gasFee,
      gasUsage,
      billingLabel: formatBillingRangeLabel(meterForm.billing_start, meterForm.billing_end),
    };
  }, [meterForm, selectedMeterRoom, selectedMeterOccupancy, meterChargeSettingMap]);

  const meterMembers = useMemo(() => selectedMeterOccupancy?.members ?? [], [selectedMeterOccupancy]);
  const meterMemberEntries = useMemo<MeterMemberEntry[]>(
    () =>
      meterMembers.map((member, index) => ({
        key: member.contractId ? `contract-${member.contractId}` : `member-${index}`,
        name: member.name || `成员${index + 1}`,
        contractId: member.contractId,
      })),
    [meterMembers],
  );
  const meterMemberSignature = meterMemberEntries.map((entry) => entry.key).join("|");

  useEffect(() => {
    setMeterMemberRatios({});
    setMeterMemberChargeOverrides({});
  }, [meterMemberSignature]);

  const meterOccupantDisplay = selectedMeterOccupancy?.names?.length ? selectedMeterOccupancy.names.join("、") : "暂无入住人员";

  const autoChargeAmounts = useMemo(() => {
    const map = new Map<string, number>();
    const addAmount = (key: string, amount?: number | null) => {
      if (amount == null || Number.isNaN(amount)) return;
      map.set(key, Number(amount.toFixed(2)));
    };
    const trashSetting = meterChargeSettingMap.get("trash");
    if (trashSetting?.unitPrice != null) {
      addAmount("trash", trashSetting.unitPrice);
    }
    if (meterExtraChargeToggles.water_supply !== false) {
      const waterSupplySetting = meterChargeSettingMap.get("water_supply");
      if (waterSupplySetting?.unitPrice != null && meterFormPreview.waterUsage != null) {
        addAmount("water_supply", meterFormPreview.waterUsage * waterSupplySetting.unitPrice);
      }
    }
    const sewageSetting = meterChargeSettingMap.get("sewage");
    if (sewageSetting?.unitPrice != null && meterFormPreview.waterUsage != null) {
      addAmount("sewage", meterFormPreview.waterUsage * sewageSetting.unitPrice);
    }
    return map;
  }, [meterChargeSettingMap, meterFormPreview.waterUsage, meterExtraChargeToggles]);

  const getResolvedSupplementalAmount = useCallback(
    (key: string) => {
      if (meterExtraChargeToggles[key] === false) return null;
      const manualValue = parseDecimalInput(meterExtraChargeInputs[key]);
      if (manualValue != null) return manualValue;
      return autoChargeAmounts.get(key) ?? null;
    },
    [meterExtraChargeInputs, autoChargeAmounts, meterExtraChargeToggles],
  );

  type MemberChargeKey = (typeof MEMBER_SHARE_CHARGE_KEYS)[number];

  const resolvedChargeTotals = useMemo<Record<MemberChargeKey, number | null>>(
    () => ({
      electric: meterFormPreview.electricFee ?? null,
      water: meterFormPreview.waterFee ?? null,
      gas: meterFormPreview.gasFee ?? null,
      trash: getResolvedSupplementalAmount("trash"),
      water_supply: getResolvedSupplementalAmount("water_supply"),
      sewage: getResolvedSupplementalAmount("sewage"),
    }),
    [meterFormPreview, getResolvedSupplementalAmount],
  );

  const meterMemberSharePreview = useMemo<MeterMemberShareRow[]>(() => {
    if (meterMemberEntries.length === 0) return [];
    if (selectedMeterRoom?.cost_bearing_mode !== "personal") return [];
    const weights = meterMemberEntries.map((entry) => {
      const ratioValue = parseDecimalInput(meterMemberRatios[entry.key]);
      return ratioValue != null && ratioValue > 0 ? ratioValue : 1;
    });
    const totalWeight = weights.reduce((sum, weight) => sum + weight, 0) || meterMemberEntries.length;
    return meterMemberEntries.map((entry, index) => {
      const normalized = totalWeight > 0 ? weights[index] / totalWeight : 1 / meterMemberEntries.length;
      const autoCharges: Record<string, number | null> = {};
      const resolvedCharges: Record<string, number | null> = {};
      MEMBER_SHARE_CHARGE_KEYS.forEach((chargeKey) => {
        const total = resolvedChargeTotals[chargeKey];
        if (total == null) {
          autoCharges[chargeKey] = null;
          resolvedCharges[chargeKey] = null;
          return;
        }
        const autoValue = Number((total * normalized).toFixed(2));
        autoCharges[chargeKey] = autoValue;
        const overrideValue = parseDecimalInput(meterMemberChargeOverrides[entry.key]?.[chargeKey]);
        resolvedCharges[chargeKey] = overrideValue != null ? overrideValue : autoValue;
      });
      const totalAmount = Object.values(resolvedCharges).reduce<number>((sum, value) => (value != null ? sum + value : sum), 0);
      return {
        key: entry.key,
        name: entry.name,
        contractId: entry.contractId,
        ratioWeight: weights[index],
        ratioNormalized: normalized,
        autoCharges,
        resolvedCharges,
        totalAmount,
      };
    });
  }, [meterMemberEntries, meterMemberRatios, meterMemberChargeOverrides, resolvedChargeTotals, selectedMeterRoom?.cost_bearing_mode]);

  const getRoomStatusLabel = useCallback(
    (room: DormRoom) => deriveRoomStatus(room, roomOccupancyMap.get(room.id) ?? EMPTY_OCCUPANCY),
    [roomOccupancyMap],
  );

  const roomByNumber = useMemo(() => {
    const map = new Map<string, DormRoom[]>();
    rooms.forEach((room) => {
      const key = room.room_number?.trim();
      if (!key) return;
      const bucket = map.get(key) ?? [];
      bucket.push(room);
      map.set(key, bucket);
    });
    return map;
  }, [rooms]);

  const getContractBuildingName = useCallback(
    (contract: DormContract) => {
      const room = contract.room || roomById.get(contract.room_id);
      if (!room) return "-";
      const building = buildingById.get(room.building_id);
      return building?.name || "-";
    },
    [roomById, buildingById],
  );

  const getContractNoteMeta = useCallback((contract: DormContract) => parseContractNoteMeta(contract.notes), []);

  const contractColumnHelpers = useMemo<ContractColumnHelpers>(
    () => ({
      getBuildingName: getContractBuildingName,
      getNoteMeta: getContractNoteMeta,
    }),
    [getContractBuildingName, getContractNoteMeta],
  );

  const contractById = useMemo(() => {
    const map = new Map<number, DormContract>();
    contracts.forEach((contract) => {
      map.set(contract.id, contract);
    });
    return map;
  }, [contracts]);

  const selectedCheckoutContract = useMemo(() => {
    if (!checkoutForm.contract_id) return null;
    return contractById.get(Number(checkoutForm.contract_id)) ?? null;
  }, [checkoutForm.contract_id, contractById]);

  const checkoutChargeKeys = useMemo(() => {
    const keys = new Set<string>();
    if (!selectedCheckoutContract) return keys;
    const room = (selectedCheckoutContract.room_id ? roomById.get(selectedCheckoutContract.room_id) : undefined) || selectedCheckoutContract.room;
    const siteId = room?.site_id ?? (room?.building_id ? buildingById.get(room.building_id)?.site_id : undefined);
    if (!siteId) return keys;
    const chargeItems = siteChargeConfigMap.get(siteId) ?? [];
    chargeItems.forEach((item) => {
      if (item.enabled) {
        keys.add(item.key);
      }
    });
    return keys;
  }, [selectedCheckoutContract, roomById, buildingById, siteChargeConfigMap]);
  const showDepositFields = checkoutChargeKeys.has("deposit");
  const showGuaranteeFields = checkoutChargeKeys.has("pledge");

  const roomHistoryMap = useMemo(() => {
    const map = new Map<number, DormContract[]>();
    contracts.forEach((contract) => {
      if (!contract.room_id) return;
      const bucket = map.get(contract.room_id) ?? [];
      bucket.push(contract);
      map.set(contract.room_id, bucket);
    });
    map.forEach((list) => {
      list.sort((a, b) => {
        const aTime = new Date(a.start_date || 0).getTime();
        const bTime = new Date(b.start_date || 0).getTime();
        return bTime - aTime;
      });
    });
    return map;
  }, [contracts]);

const getRoomContext = useCallback(
  (room: DormRoom): RoomColumnHelpers => {
    const building = buildingById.get(room.building_id);
    const site = room.site_id
      ? siteById.get(room.site_id)
      : building?.site_id
        ? siteById.get(building.site_id)
        : undefined;
    const occupancy = roomOccupancyMap.get(room.id) ?? EMPTY_OCCUPANCY;
    const resolvedCharges = getChargeSettingsForRoom(room);
    const chargeRates = new Map<string, RoomChargeRateMeta>();
    resolvedCharges.forEach((charge) => {
      chargeRates.set(charge.key, { unitPrice: charge.unitPrice, unitLabel: charge.unitLabel, mode: charge.mode });
    });
    return { building, site, occupancy, chargeRates };
  },
  [buildingById, siteById, roomOccupancyMap, getChargeSettingsForRoom],
);

  const composeMeterRecord = useCallback(
    (params: {
      room: DormRoom;
      id: number;
      meterDate: string;
      inspector: string;
      billingStart: string;
      billingEnd: string;
      chargeDetails: DormChargeDetail[];
      occupantsOverride?: string[];
    }): MeterRecord => {
      const { room } = params;
      const building = buildingById.get(room.building_id);
      const siteId = room.site_id ?? building?.site_id ?? null;
      const siteName = siteId ? siteById.get(siteId)?.name || "" : "";
      const occupancy = roomOccupancyMap.get(room.id);
      const fallbackOccupants = contracts
        .filter((contract) => contract.room_id === room.id)
        .map((contract) => contract.employee_name || "")
        .filter(Boolean);
      const occupantNames = params.occupantsOverride?.length
        ? params.occupantsOverride
        : occupancy?.names?.length
          ? occupancy.names
          : fallbackOccupants;
    const charges: MeterChargeView[] = (params.chargeDetails ?? []).map((detail) => {
      const definition = getChargeDefinition(detail.key);
      const participants = Array.isArray(detail.participants)
        ? detail.participants
            .map((participant) => {
              const normalizedAmount = toNullableNumber(participant.amount ?? null);
              const normalizedRatio = typeof participant.ratio === "number" && Number.isFinite(participant.ratio) ? participant.ratio : null;
              const contractId = (participant as { contract_id?: number }).contract_id ?? (participant as { contractId?: number }).contractId;
              const name = participant.name || "";
              if (!name && normalizedAmount == null && normalizedRatio == null && !contractId) {
                return null;
              }
              return {
                name: name || "未命名",
                contractId: contractId ?? undefined,
                amount: normalizedAmount,
                ratio: normalizedRatio,
              } as MeterChargeParticipantView;
            })
            .filter((participant): participant is MeterChargeParticipantView => Boolean(participant))
        : undefined;
      return {
        key: detail.key,
        label: detail.label || definition?.label || detail.key,
        mode: detail.mode || definition?.mode || "fixed",
        unitLabel: detail.unit_label || definition?.unitLabel,
        unitPrice: detail.unit_price ?? definition?.defaultUnitPrice ?? null,
        start: toNullableNumber(detail.start),
        end: toNullableNumber(detail.end),
        usage: toNullableNumber(detail.usage),
        amount: toNullableNumber(detail.amount),
        participants,
      };
    });
      const chargeMap = new Map(charges.map((item) => [item.key, item]));
      const electric = chargeMap.get("electric");
      const water = chargeMap.get("water");
      const gas = chargeMap.get("gas");
      return {
        id: params.id,
        room_id: room.id,
        site_id: siteId,
        site_name: siteName || "",
        building_name: building?.name || "未命名楼栋",
        room_number: room.room_number || "--",
        room_type: room.room_category || ((room.bed_count || 0) <= 1 ? "单间" : "多人间"),
        occupants: occupantNames,
        meter_date: params.meterDate,
        inspector: params.inspector || "李永娇",
        billing_month: derivePeriodKeyFromDate(params.billingEnd || params.billingStart || params.meterDate) || "",
        billing_range: formatBillingRangeLabel(params.billingStart, params.billingEnd),
        billing_start: params.billingStart,
        billing_end: params.billingEnd,
        electric_start: electric?.start ?? null,
        electric_end: electric?.end ?? null,
        electric_fee: electric?.amount ?? null,
        water_start: water?.start ?? null,
        water_end: water?.end ?? null,
        water_fee: water?.amount ?? null,
        gas_start: gas?.start ?? null,
        gas_end: gas?.end ?? null,
        gas_fee: gas?.amount ?? null,
        charges,
      };
    },
    [buildingById, roomOccupancyMap, contracts, siteById],
  );


  const buildChargeDetailsFromValues = useCallback(
    (roomId: number, values: { electricStart?: number | null; electricEnd?: number | null; waterStart?: number | null; waterEnd?: number | null; gasStart?: number | null; gasEnd?: number | null; gasAmount?: number | null }) => {
      const room = roomById.get(roomId);
      const chargeSettings = getChargeSettingsForRoom(room ?? undefined);
      const chargeMap = new Map<string, ChargeSetting>();
      chargeSettings.forEach((item) => chargeMap.set(item.key, item));
      const details: DormChargeDetail[] = [];

      const addMeterDetail = (key: LegacyChargeKey, start?: number | null, end?: number | null) => {
        const setting = chargeMap.get(key);
        if (!setting) return;
        const unitPrice = setting.unitPrice ?? 0;
        const usage = start != null && end != null ? Math.max(0, end - start) : null;
        const amount = usage != null && unitPrice > 0 ? Number((usage * unitPrice).toFixed(2)) : null;
        details.push({
          key,
          label: setting.label,
          start: start ?? undefined,
          end: end ?? undefined,
          usage: usage ?? undefined,
          unit_price: unitPrice || undefined,
          unit_label: setting.unitLabel,
          amount: amount ?? undefined,
          mode: setting.mode,
        });
      };

      const addManualAmount = (key: string, amount?: number | null) => {
        const setting = chargeMap.get(key);
        if (!setting || amount == null) return;
        details.push({
          key,
          label: setting.label,
          amount,
          unit_price: setting.unitPrice,
          unit_label: setting.unitLabel,
          mode: setting.mode,
        });
      };

      addMeterDetail("electric", values.electricStart ?? null, values.electricEnd ?? null);
      addMeterDetail("water", values.waterStart ?? null, values.waterEnd ?? null);
      if (values.gasStart != null && values.gasEnd != null) {
        addMeterDetail("gas", values.gasStart, values.gasEnd);
      } else {
        addManualAmount("gas", values.gasAmount);
      }
      return details;
    },
    [roomById, getChargeSettingsForRoom],
  );

  const buildRoomPrintDataset = useCallback(
    (targetRooms: DormRoom[]) => {
      if (targetRooms.length === 0) {
        return null;
      }
      const columns = visibleRoomColumns;
      return {
        columns: columns.map((column) => column.label),
        rows: targetRooms.map((room) => {
          const context = getRoomContext(room);
          return columns.map((column) => {
            const value = column.getValue ? column.getValue(room, context) : undefined;
            if (value === null || value === undefined) {
              return "";
            }
            return typeof value === "number" ? String(value) : String(value);
          });
        }),
        defaultTitle: `房间列表打印（共 ${targetRooms.length} 条）`,
      };
    },
    [visibleRoomColumns, getRoomContext],
  );

  useEffect(() => {
    setMeterRecords(() => {
      const mapped = meterSourceRecords
        .map((record) => {
          const room = roomById.get(record.room_id) ?? record.room;
          if (!room) return null;
          const chargeDetails = Array.isArray(record.charge_details) ? record.charge_details : [];
          return composeMeterRecord({
            room,
            id: record.id,
            meterDate: formatDateInputValue(record.meter_date) || record.meter_date,
            inspector: record.inspector || "李永娇",
            billingStart: formatDateInputValue(record.billing_start) || record.billing_start,
            billingEnd: formatDateInputValue(record.billing_end) || record.billing_end,
            chargeDetails,
          });
        })
        .filter((item): item is MeterRecord => Boolean(item));
      return mapped;
    });
  }, [meterSourceRecords, roomById, composeMeterRecord]);

  const supplementalChargeKeys = useMemo(() => {
    const keys = new Set<string>();
    siteChargeConfigMap.forEach((items) => {
      items.forEach((item) => {
        if (!isBaseChargeKey(item.key)) {
          keys.add(item.key);
        }
      });
    });
    meterRecords.forEach((record) => {
      record.charges.forEach((charge) => {
        if (!isBaseChargeKey(charge.key)) {
          keys.add(charge.key);
        }
      });
    });
    return Array.from(keys);
  }, [siteChargeConfigMap, meterRecords]);

  const visibleSupplementalKeys = useMemo(
    () => supplementalChargeKeys.filter((key) => meterExtraColumnVisibility[key] !== false),
    [supplementalChargeKeys, meterExtraColumnVisibility],
  );

  const meterPeriodOptions = useMemo(() => {
    const map = new Map<string, string>();
    meterRecords.forEach((record) => {
      const key = derivePeriodKeyFromDate(record.billing_end || record.billing_start || record.meter_date);
      if (!key) return;
      if (!map.has(key)) {
        map.set(key, formatPeriodLabel(key));
      }
    });
    return Array.from(map.entries())
      .sort((a, b) => b[0].localeCompare(a[0]))
      .map(([key, label]) => ({ key, label }));
  }, [meterRecords]);

  useEffect(() => {
    if (meterPeriodOptions.length === 0) {
      if (meterPeriodInitialized) {
        setMeterPeriodFilter("all");
        setMeterPeriodInitialized(false);
      }
      return;
    }
    if (!meterPeriodInitialized) {
      setMeterPeriodFilter(meterPeriodOptions[0].key);
      setMeterPeriodInitialized(true);
      return;
    }
    if (meterPeriodFilter !== "all" && !meterPeriodOptions.some((option) => option.key === meterPeriodFilter)) {
      setMeterPeriodFilter(meterPeriodOptions[0].key);
    }
  }, [meterPeriodOptions, meterPeriodFilter, meterPeriodInitialized]);

  const buildContractPrintDataset = useCallback(
    (targetContracts: DormContract[]) => {
      if (targetContracts.length === 0) {
        return null;
      }
      const columns = visibleContractColumns;
      return {
        columns: columns.map((column) => column.label),
        rows: targetContracts.map((contract) =>
          columns.map((column) => {
            const fallbackRecord = contract as unknown as Record<string, unknown>;
            const value = column.getValue ? column.getValue(contract, contractColumnHelpers) : fallbackRecord[column.id];
            if (value === undefined || value === null) {
              return "";
            }
            return typeof value === "number" ? String(value) : String(value);
          }),
        ),
        defaultTitle: `入住列表打印（共 ${targetContracts.length} 条）`,
      };
    },
    [visibleContractColumns, contractColumnHelpers],
  );

  const getMeterSortValue = useCallback((record: MeterRecord, columnId: MeterColumnId) => {
    switch (columnId) {
      case "site":
        return record.site_name || "";
      case "building":
        return record.building_name || "";
      case "room":
        return record.room_number || "";
      case "roomType":
        return record.room_type || "";
      case "occupants":
        return record.occupants.join("、");
      case "shareDetails":
        return buildChargesShareSummary(record.charges);
      case "meterDate":
        return record.meter_date || "";
      case "inspector":
        return record.inspector || "";
      case "billingRange":
        return record.billing_month || record.billing_range || "";
      case "electricStart":
        return record.electric_start ?? 0;
      case "electricEnd":
        return record.electric_end ?? 0;
      case "electricFee":
        return record.electric_fee ?? 0;
      case "waterStart":
        return record.water_start ?? 0;
      case "waterEnd":
        return record.water_end ?? 0;
      case "waterFee":
        return record.water_fee ?? 0;
      case "gasFee":
        return record.gas_fee ?? 0;
      default:
        return record.meter_date || "";
    }
  }, []);

  const filteredMeterRecords = useMemo(() => {
    const keyword = meterSearch.trim().toLowerCase();
    return meterRecords.filter((record) => {
      const room = roomById.get(record.room_id);
      const roomSiteId = record.site_id ?? getRoomSiteId(room);
      if (meterSiteFilter !== "all" && roomSiteId !== meterSiteFilter) {
        return false;
      }
      if (meterBuildingFilter !== "all") {
        if (!room || room.building_id !== meterBuildingFilter) {
          return false;
        }
      }
      if (meterPeriodFilter !== "all") {
        const periodKey = record.billing_month || derivePeriodKeyFromDate(record.billing_end || record.billing_start || record.meter_date);
        if (periodKey !== meterPeriodFilter) {
          return false;
        }
      }
      if (!keyword) {
        return true;
      }
      const occupantLabel = record.occupants.join("、").toLowerCase();
      const billingLabel = (record.billing_month || record.billing_range || "").toLowerCase();
      const roomLabel = (record.room_number || "").toLowerCase();
      const buildingLabel = (record.building_name || "").toLowerCase();
      return occupantLabel.includes(keyword) || billingLabel.includes(keyword) || roomLabel.includes(keyword) || buildingLabel.includes(keyword);
    });
  }, [meterRecords, meterSearch, meterBuildingFilter, meterSiteFilter, meterPeriodFilter, roomById, getRoomSiteId]);

  const meterRecordsByBearing = useMemo(() => {
    const personal: MeterRecord[] = [];
    const company: MeterRecord[] = [];
    filteredMeterRecords.forEach((record) => {
      const room = roomById.get(record.room_id);
      if (room?.cost_bearing_mode === "personal") {
        personal.push(record);
      } else {
        company.push(record);
      }
    });
    return { personal, company };
  }, [filteredMeterRecords, roomById]);

  const meterCompanyRows = useMemo(() => buildCompanyMeterRows(meterRecordsByBearing.company), [meterRecordsByBearing.company, buildCompanyMeterRows]);
  const meterPersonalRows = useMemo(
    () => buildPersonalMeterRows(meterRecordsByBearing.personal),
    [meterRecordsByBearing.personal, buildPersonalMeterRows],
  );

  const sortMeterRows = useCallback(
    (rows: MeterTableRow[]) => {
      if (!meterSort.key) return rows;
      const sorted = [...rows];
      sorted.sort((a, b) => {
        const valueA = getMeterSortValue(a.display, meterSort.key!);
        const valueB = getMeterSortValue(b.display, meterSort.key!);
        if (typeof valueA === "number" && typeof valueB === "number") {
          return meterSort.direction === "asc" ? valueA - valueB : valueB - valueA;
        }
        return meterSort.direction === "asc"
          ? String(valueA).localeCompare(String(valueB), "zh-CN")
          : String(valueB).localeCompare(String(valueA), "zh-CN");
      });
      return sorted;
    },
    [meterSort, getMeterSortValue],
  );

  useEffect(() => {
    if (!meterSort.key) return;
    const sortable = meterColumnsForRender.some((column) => column.id === meterSort.key);
    if (!sortable) {
      const fallback = meterColumnsForRender[0];
      if (fallback) {
        setMeterSort({ key: fallback.id, direction: DEFAULT_METER_SORT.direction });
      }
    }
  }, [meterColumnsForRender, meterSort.key]);

  const sortedCompanyRows = useMemo(() => sortMeterRows(meterCompanyRows), [meterCompanyRows, sortMeterRows]);
  const sortedPersonalRows = useMemo(() => sortMeterRows(meterPersonalRows), [meterPersonalRows, sortMeterRows]);

  const summarizeCompanyRecords = useCallback(
    (records: MeterRecord[]) => {
      const extras: Record<string, number> = {};
      supplementalChargeKeys.forEach((key) => {
        extras[key] = 0;
      });
      const summary = records.reduce(
        (acc, record) => {
          if (record.electric_fee != null) acc.electric += record.electric_fee;
          if (record.water_fee != null) acc.water += record.water_fee;
          if (record.gas_fee != null) acc.gas += record.gas_fee;
          record.charges.forEach((charge) => {
            if (!isBaseChargeKey(charge.key) && charge.amount != null) {
              acc.extras[charge.key] = (acc.extras[charge.key] || 0) + charge.amount;
            }
          });
          return acc;
        },
        { electric: 0, water: 0, gas: 0, extras },
      );
      return { count: records.length, ...summary };
    },
    [supplementalChargeKeys],
  );

  const summarizePersonalRows = useCallback(
    (rows: MeterTableRow[]) => {
      const extras: Record<string, number> = {};
      supplementalChargeKeys.forEach((key) => {
        extras[key] = 0;
      });
      const summary = rows.reduce(
        (acc, row) => {
          const record = row.display;
          if (record.electric_fee != null) acc.electric += record.electric_fee;
          if (record.water_fee != null) acc.water += record.water_fee;
          if (record.gas_fee != null) acc.gas += record.gas_fee;
          record.charges.forEach((charge) => {
            if (!isBaseChargeKey(charge.key) && charge.amount != null) {
              acc.extras[charge.key] = (acc.extras[charge.key] || 0) + charge.amount;
            }
          });
          return acc;
        },
        { electric: 0, water: 0, gas: 0, extras },
      );
      return { count: rows.length, ...summary };
    },
    [supplementalChargeKeys],
  );

  const meterCompanySummary = useMemo(() => summarizeCompanyRecords(meterRecordsByBearing.company), [summarizeCompanyRecords, meterRecordsByBearing.company]);
  const meterPersonalSummary = useMemo(() => summarizePersonalRows(sortedPersonalRows), [summarizePersonalRows, sortedPersonalRows]);

  const buildRoomExportRecord = useCallback(
    (room: DormRoom) => {
      const { building, site, occupancy } = getRoomContext(room);
      const capacity = room.bed_count || room.beds?.length || 0;
      const bedStatus = deriveBedStatus(room, occupancy);
      const roomStatus = deriveRoomStatus(room, occupancy);
      return {
        宿舍地点: site?.name ?? "",
        楼栋: building?.name ?? "",
        房号: room.room_number || "",
        房间类型: room.room_category || "",
        户型: room.house_layout || room.room_type || "",
        单间: formatBooleanLabel(room.room_category === "单间" || capacity <= 1),
        面积: room.area_square ?? "",
        月租金: formatCurrencyValue(room.monthly_rent),
        物业费: formatCurrencyValue(room.property_fee),
        季租金: formatCurrencyValue(room.quarterly_rent),
        押金: formatCurrencyValue(room.deposit_fee),
        保证金: formatCurrencyValue(room.guarantee_fee),
        床位数量: capacity,
        入住人数: occupancy.count,
        床位状态: bedStatus,
        房间状态: roomStatus,
        入住人员: occupancy.names.join("、"),
        首月费用: formatCurrencyValue(room.first_month_fee),
        电费基准: formatUnitValue(room.electric_base, "/度"),
        水费基准: formatUnitValue(room.water_base, "/m³"),
        燃气费: formatUnitValue(room.gas_base, "/m³"),
        垃圾费: formatCurrencyValue(room.trash_fee),
        二次供水费: formatUnitValue(room.water_supply_fee, "/m³"),
        污水处理费: formatUnitValue(room.sewage_fee, "/m³"),
        物品清单: room.inventory_note || "",
      };
    },
    [getRoomContext],
  );

  const exportRoomsToWorkbook = useCallback(
    (roomsToExport: DormRoom[], filename: string) => {
      if (roomsToExport.length === 0) {
        toast.error("没有可导出的房间");
        return;
      }
      const rows = roomsToExport.map((room) => buildRoomExportRecord(room));
      const worksheet = XLSX.utils.json_to_sheet(rows);
      const workbook = XLSX.utils.book_new();
      XLSX.utils.book_append_sheet(workbook, worksheet, "房间列表");
      XLSX.writeFile(workbook, `${filename}-${new Date().toISOString().slice(0, 10)}.xlsx`);
    },
    [buildRoomExportRecord],
  );

  const getMeterColumnDisplay = useCallback(
    (record: MeterRecord, columnId: MeterColumnId) => {
      let value: string | number;
      switch (columnId) {
        case "site":
          value = record.site_name || "--";
          break;
        case "building":
          value = record.building_name || "--";
          break;
        case "room":
          value = record.room_number || "--";
          break;
      case "roomType":
        value = record.room_type || "--";
        break;
      case "occupants":
        value = record.occupants.length ? record.occupants.join("、") : "--";
        break;
      case "shareDetails":
        value = buildChargesShareSummary(record.charges) || "--";
        break;
      case "meterDate":
        value = formatDateLabel(record.meter_date);
        break;
        case "inspector":
          value = record.inspector || "--";
          break;
      case "billingRange":
          value = formatMonthDisplay(record.billing_month || undefined) || record.billing_range || "--";
          break;
        case "electricStart":
          value = record.electric_start ?? "--";
          break;
        case "electricEnd":
          value = record.electric_end ?? "--";
          break;
        case "electricFee":
          value = record.electric_fee != null ? formatCurrencyValue(record.electric_fee) : "--";
          break;
        case "waterStart":
          value = record.water_start ?? "--";
          break;
        case "waterEnd":
          value = record.water_end ?? "--";
          break;
        case "waterFee":
          value = record.water_fee != null ? formatCurrencyValue(record.water_fee) : "--";
          break;
        case "gasStart":
          value = record.gas_start ?? "--";
          break;
        case "gasEnd":
          value = record.gas_end ?? "--";
          break;
        case "gasFee":
          value = record.gas_fee != null ? formatCurrencyValue(record.gas_fee) : "--";
          break;
        default:
          value = "";
          break;
      }
      return typeof value === "number" ? String(value) : value;
    },
    [],
  );

  const buildMeterExportRecord = useCallback(
    (record: MeterRecord) => {
      const base: Record<string, string> = {
        地点: record.site_name || "",
        楼栋: record.building_name,
        房号: record.room_number,
        房间类型: record.room_type,
        入住人员: record.occupants.join("、") || "--",
        分摊详情: buildChargesShareSummary(record.charges) || "",
        抄表时间: formatDateLabel(record.meter_date),
        抄表人: record.inspector,
        账期月份: formatMonthDisplay(record.billing_month || undefined, record.billing_range || ""),
        电表起度: record.electric_start != null ? String(record.electric_start) : "",
        电表止度: record.electric_end != null ? String(record.electric_end) : "",
        电费: record.electric_fee != null ? formatCurrencyValue(record.electric_fee) : "",
        水表起度: record.water_start != null ? String(record.water_start) : "",
        水表止度: record.water_end != null ? String(record.water_end) : "",
        水费: record.water_fee != null ? formatCurrencyValue(record.water_fee) : "",
        气表起度: record.gas_start != null ? String(record.gas_start) : "",
        气表止度: record.gas_end != null ? String(record.gas_end) : "",
        气费: record.gas_fee != null ? formatCurrencyValue(record.gas_fee) : "",
      };
      supplementalChargeKeys.forEach((key) => {
        const label = getChargeDefinition(key)?.label || key;
        const amount = record.charges.find((charge) => charge.key === key)?.amount ?? null;
        base[label] = amount != null ? formatCurrencyValue(amount) : "";
      });
      return base;
    },
    [supplementalChargeKeys],
  );

  const exportMeterRecords = useCallback(
    (records: MeterRecord[], filename: string) => {
      if (records.length === 0) {
        toast.error("暂无可导出的抄表记录");
        return;
      }
      const rows = records.map((record) => buildMeterExportRecord(record));
      const worksheet = XLSX.utils.json_to_sheet(rows);
      const workbook = XLSX.utils.book_new();
      XLSX.utils.book_append_sheet(workbook, worksheet, "抄表记录");
      XLSX.writeFile(workbook, `${filename}-${new Date().toISOString().slice(0, 10)}.xlsx`);
    },
    [buildMeterExportRecord],
  );


  const buildMeterPrintDataset = useCallback(
    (records: MeterRecord[]) => {
      if (records.length === 0) return null;
      const extraColumns = supplementalChargeKeys.map((key) => getChargeDefinition(key)?.label || key);
      const columns = [...visibleMeterColumns.map((column) => column.label), ...extraColumns];
      const rows = records.map((record) => {
        const baseValues = visibleMeterColumns.map((column) => getMeterColumnDisplay(record, column.id));
        const extraValues = supplementalChargeKeys.map((key) => {
          const amount = record.charges.find((charge) => charge.key === key)?.amount ?? null;
          return amount != null ? formatCurrencyValue(amount) : "--";
        });
        return [...baseValues, ...extraValues];
      });
      return {
        columns,
        rows,
        defaultTitle: `抄表记录（共 ${records.length} 条）`,
      };
    },
    [visibleMeterColumns, getMeterColumnDisplay, supplementalChargeKeys],
  );

  const composeBillPayloadFromMeterRecord = useCallback(
    (record: MeterRecord) => {
      const items = record.charges
        .filter((charge) => charge.amount != null)
        .map((charge) => {
          const quantity = charge.usage ?? 1;
          const unitPrice = charge.unitPrice ?? (charge.amount ?? 0);
          const amount = charge.amount ?? unitPrice * quantity;
          return {
            item_type: charge.key,
            label: charge.label,
            quantity,
            unit_price: unitPrice,
            amount,
          };
        });
      if (items.length === 0) return null;
      return {
        room_id: record.room_id,
        employee_name: record.occupants.join("、") || undefined,
        period_label: formatMonthDisplay(record.billing_month || undefined, record.billing_range || record.billing_start),
        due_date: record.billing_end || record.meter_date,
        status: "pending",
        items,
      };
    },
    [],
  );

  const closeMeterPrintDialog = () => {
    setMeterPrintDialogOpen(false);
    setMeterPrintContext([]);
    setMeterPrintSuggestedTitle("");
  };

  const closeRoomPrintDialog = () => {
    setRoomPrintDialogOpen(false);
    setRoomPrintContext([]);
    setRoomPrintSuggestedTitle("");
  };

  const closeContractPrintDialog = () => {
    setContractPrintDialogOpen(false);
    setContractPrintContext([]);
    setContractPrintSuggestedTitle("");
  };

  const handleGenerateMeterPrint = async () => {
    if (meterPrintContext.length === 0) {
      toast.error("请选择需要打印的抄表记录");
      return;
    }
    const dataset = buildMeterPrintDataset(meterPrintContext);
    if (!dataset) {
      toast.error("未找到可打印的数据");
      return;
    }
    const title = (meterPrintTitle.trim() || meterPrintSuggestedTitle || dataset.defaultTitle).trim();
    const watermark = (meterPrintWatermark.trim() || "内部资料 请勿外传").trim();
    const loadingId = toast.loading("正在生成打印预览...");
    try {
      const blob = await createReportPdf({
        title,
        watermark,
        columns: dataset.columns,
        rows: dataset.rows,
        orientation: meterPrintOrientation,
      });
      const url = URL.createObjectURL(blob);
      const previewWindow = window.open(url);
      if (!previewWindow) {
        toast.error("浏览器阻止了打印窗口，请允许弹窗后重试");
        URL.revokeObjectURL(url);
      } else {
        previewWindow.onload = () => previewWindow.focus();
        const cleanup = () => URL.revokeObjectURL(url);
        previewWindow.addEventListener("beforeunload", cleanup, { once: true });
        setTimeout(cleanup, 60_000);
      }
      closeMeterPrintDialog();
    } catch (error) {
      console.error("[Dormitory] generate meter pdf failed", error);
      toast.error("生成打印预览失败，请稍后重试");
    } finally {
      toast.dismiss(loadingId);
    }
  };

  const handleGenerateRoomPrint = async () => {
    if (roomPrintContext.length === 0) {
      toast.error("请选择需要打印的房间");
      return;
    }
    if (typeof window === "undefined") {
      toast.error("当前环境不支持打印");
      return;
    }
    const dataset = buildRoomPrintDataset(roomPrintContext);
    if (!dataset) {
      toast.error("未找到可打印的数据");
      return;
    }
    const title = (roomPrintTitle.trim() || roomPrintSuggestedTitle || dataset.defaultTitle).trim();
    const watermark = (roomPrintWatermark.trim() || "内部资料 请勿外传").trim();
    const loadingId = toast.loading("正在生成打印预览，请稍候...");
    try {
      const blob = await createReportPdf({
        title,
        watermark,
        columns: dataset.columns,
        rows: dataset.rows,
        orientation: roomPrintOrientation,
      });
      const url = URL.createObjectURL(blob);
      const previewWindow = window.open(url);
      if (!previewWindow) {
        toast.error("浏览器阻止了打印窗口，请允许弹窗后重试");
        URL.revokeObjectURL(url);
      } else {
        previewWindow.onload = () => previewWindow.focus();
        const cleanup = () => URL.revokeObjectURL(url);
        previewWindow.addEventListener("beforeunload", cleanup, { once: true });
        setTimeout(cleanup, 60_000);
      }
      closeRoomPrintDialog();
    } catch (error) {
      console.error("[Dormitory] generate room pdf failed", error);
      toast.error("生成打印预览失败，请稍后重试");
    } finally {
      toast.dismiss(loadingId);
    }
  };

  const handleGenerateContractPrint = async () => {
    if (contractPrintContext.length === 0) {
      toast.error("请选择需要打印的入住记录");
      return;
    }
    const title = (contractPrintTitle.trim() || contractPrintSuggestedTitle || `入住列表（共 ${contractPrintContext.length} 条）`).trim();
    const watermark = (contractPrintWatermark.trim() || "内部资料 请勿外传").trim();
    await printContracts(contractPrintContext, {
      title,
      watermark,
      orientation: contractPrintOrientation,
    });
    closeContractPrintDialog();
  };

  const availableRoomsForContracts = useMemo(() => {
    const currentRoomId = Number(contractForm.room_id || 0);
    return rooms.filter((room) => {
      const isEditingCurrentRoom = currentRoomId && room.id === currentRoomId;
      if (room.status === "维护中" && !isEditingCurrentRoom) {
        return false;
      }
      const occupancy = roomOccupancyMap.get(room.id) ?? EMPTY_OCCUPANCY;
      const capacity = room.bed_count || room.beds?.length || 0;
      if (capacity <= 0) return true;
      if (occupancy.count >= capacity) {
        return isEditingCurrentRoom;
      }
      return true;
    });
  }, [rooms, contractForm.room_id, roomOccupancyMap]);

  const getContractSortValue = useCallback(
    (contract: DormContract, columnId: string) => {
      switch (columnId) {
        case "employee":
          return contract.employee_name || "";
        case "department":
          return contract.employee_department || "";
        case "building":
          return getContractBuildingName(contract);
        case "room":
          return contract.room?.room_number || "";
        case "startDate":
          return contract.start_date || "";
        case "endDate":
          return contract.end_date || "";
        case "rent":
          return contract.rent_amount || 0;
        case "deposit":
          return contract.deposit_amount || 0;
        case "status":
          return contract.status || "active";
        default:
          return contract.employee_name || "";
      }
    },
    [getContractBuildingName],
  );

  const filteredContracts = useMemo(() => {
    const keyword = contractSearch.trim().toLowerCase();
    return contracts.filter((contract) => {
      const status = contract.status || "active";
      if (contractStatusFilter !== "all" && status !== contractStatusFilter) {
        return false;
      }
      if (contractSiteFilter !== "all") {
        const siteId =
          contract.room?.site_id ?? (contract.room_id ? getRoomSiteId(roomById.get(contract.room_id)) ?? null : null);
        if (siteId !== contractSiteFilter) {
          return false;
        }
      }
      if (!keyword) {
        return true;
      }
      const roomNumber = contract.room?.room_number?.toLowerCase() ?? "";
      const dept = contract.employee_department?.toLowerCase() ?? "";
      const phone = contract.employee_phone?.toLowerCase() ?? "";
      return (
        contract.employee_name?.toLowerCase().includes(keyword) ||
        roomNumber.includes(keyword) ||
        dept.includes(keyword) ||
        phone.includes(keyword)
      );
    });
  }, [contracts, contractSearch, contractStatusFilter, contractSiteFilter, roomById, getRoomSiteId]);

const sortedContracts = useMemo(() => {
  const sortableColumn = contractColumnsForRender.find(
    (column) => column.id === contractSort.columnId && column.sortable !== false,
  );
  if (!sortableColumn) {
    return [...filteredContracts];
  }
    return [...filteredContracts].sort((a, b) => {
      const valueA = getContractSortValue(a, contractSort.columnId);
      const valueB = getContractSortValue(b, contractSort.columnId);
      if (valueA === valueB) return 0;
      if (typeof valueA === "number" && typeof valueB === "number") {
        return contractSort.direction === "asc" ? valueA - valueB : valueB - valueA;
      }
      return contractSort.direction === "asc"
        ? String(valueA).localeCompare(String(valueB))
        : String(valueB).localeCompare(String(valueA));
    });
  }, [filteredContracts, contractColumnsForRender, contractSort, getContractSortValue]);

  const contractNavigationState = useMemo(() => {
    if (!editingContractId) {
      return { hasPrev: false, hasNext: false };
    }
    const index = sortedContracts.findIndex((contract) => contract.id === editingContractId);
    if (index === -1) {
      return { hasPrev: false, hasNext: false };
    }
    return {
      hasPrev: index > 0,
      hasNext: index < sortedContracts.length - 1,
    };
  }, [editingContractId, sortedContracts]);

  const handleContractNavigate = (direction: "prev" | "next") => {
    if (!editingContractId) return;
    const index = sortedContracts.findIndex((contract) => contract.id === editingContractId);
    if (index === -1) return;
    const offset = direction === "next" ? 1 : -1;
    const target = sortedContracts[index + offset];
    if (target) {
      handleEditContract(target);
    }
  };

  useEffect(() => {
    const sortableColumn = contractColumnsForRender.find(
      (column) => column.id === contractSort.columnId && column.sortable !== false,
    );
    if (sortableColumn) return;
    const fallback = contractColumnsForRender.find((column) => column.sortable !== false);
    if (fallback && contractSort.columnId !== fallback.id) {
      setContractSort({ columnId: fallback.id, direction: DEFAULT_CONTRACT_SORT.direction });
    }
  }, [contractColumnsForRender, contractSort.columnId]);

  const activeContracts = useMemo(
    () => contracts.filter((contract) => (contract.status || "active") === "active"),
    [contracts],
  );

  const contractMetaById = useMemo(() => {
    const map = new Map<number, { status: string; endDate?: string | null }>();
    contracts.forEach((contract) => {
      if (!contract.id) return;
      map.set(contract.id, { status: contract.status || "active", endDate: contract.end_date || null });
    });
    return map;
  }, [contracts]);

  const roomContractStatusMap = useMemo(() => {
    const map = new Map<number, { hasActive: boolean; lastCheckoutDate?: string }>();
    contracts.forEach((contract) => {
      if (!contract.room_id) return;
      const status = (contract.status || "active").toLowerCase();
      const next = map.get(contract.room_id) ?? { hasActive: false, lastCheckoutDate: undefined as string | undefined };
      if (status === "active") {
        next.hasActive = true;
      } else if (contract.end_date) {
        if (!next.lastCheckoutDate || contract.end_date > next.lastCheckoutDate) {
          next.lastCheckoutDate = contract.end_date;
        }
      }
      map.set(contract.room_id, next);
    });
    return map;
  }, [contracts]);

  const selectedRoom = useMemo(() => {
    const roomId = Number(contractForm.room_id);
    if (!roomId) return null;
    return roomById.get(roomId) ?? null;
  }, [roomById, contractForm.room_id]);

  const selectedRoomChargeSettings = useMemo(() => {
    if (!selectedRoom) return null;
    return getChargeSettingsForRoom(selectedRoom);
  }, [selectedRoom, getChargeSettingsForRoom]);

  useEffect(() => {
    const targetMode: ShareMode = selectedRoom?.cost_bearing_mode === "personal" ? "personal" : "company";
    setContractForm((prev) => {
      if (prev.deposit_share_mode === targetMode && prev.pledge_share_mode === targetMode) {
        return prev;
      }
      return { ...prev, deposit_share_mode: targetMode, pledge_share_mode: targetMode };
    });
  }, [selectedRoom?.cost_bearing_mode]);

  useEffect(() => {
    if (!selectedRoomChargeSettings) return;
    setContractForm((prev) => {
      let changed = false;
      const next = { ...prev };
      const depositSetting = selectedRoomChargeSettings.find((item) => item.key === "deposit");
      if (depositSetting?.unitPrice != null && next.deposit_amount.trim() === "") {
        next.deposit_amount = String(depositSetting.unitPrice);
        changed = true;
      }
      const pledgeSetting = selectedRoomChargeSettings.find((item) => item.key === "pledge");
      if (pledgeSetting?.unitPrice != null && next.pledge_amount.trim() === "") {
        next.pledge_amount = String(pledgeSetting.unitPrice);
        changed = true;
      }
      return changed ? next : prev;
    });
  }, [selectedRoomChargeSettings]);

  const ensuringRoomBedsRef = useRef<Set<number>>(new Set());

  const ensureRoomBeds = useCallback(
    async (room: DormRoom) => {
      const capacity = room.bed_count || 0;
      const currentBeds = room.beds?.length || 0;
      if (!capacity || currentBeds >= capacity) {
        return;
      }
      if (ensuringRoomBedsRef.current.has(room.id)) {
        return;
      }
      ensuringRoomBedsRef.current.add(room.id);
      const startIndex = currentBeds;
      try {
        const missingCount = capacity - currentBeds;
        const requests = Array.from({ length: missingCount }).map((_, index) =>
          createDormBed({ room_id: room.id, bed_number: `床位${startIndex + index + 1}`, status: "空闲" }),
        );
        const createdBeds = await Promise.all(requests);
        setRooms((prev) =>
          prev.map((item) =>
            item.id === room.id ? { ...item, beds: [...(item.beds ?? []), ...createdBeds] } : item,
          ),
        );
      } catch (error) {
        console.error("[Dormitory] ensure room beds failed", error);
        toast.error("自动生成床位失败，请稍后重试");
      } finally {
        ensuringRoomBedsRef.current.delete(room.id);
      }
    },
    [setRooms],
  );

  useEffect(() => {
    if (!selectedRoom) return;
    const capacity = selectedRoom.bed_count || 0;
    const currentBeds = selectedRoom.beds?.length || 0;
    if (capacity > 0 && currentBeds < capacity) {
      void ensureRoomBeds(selectedRoom);
    }
  }, [selectedRoom, ensureRoomBeds]);

  const bedOptions = useMemo(() => {
    if (!selectedRoom) return [] as Array<{ id: number; bed_number: string; occupiedBy?: string }>;
    const occupancy = roomOccupancyMap.get(selectedRoom.id) ?? EMPTY_OCCUPANCY;
    const seen = new Set<string>();
    const sortedBeds = [...(selectedRoom.beds ?? [])].sort((a, b) =>
      (a.bed_number || "").localeCompare(b.bed_number || "", "zh-CN"),
    );
    const limit = selectedRoom.bed_count && selectedRoom.bed_count > 0 ? selectedRoom.bed_count : sortedBeds.length;
    const deduped = sortedBeds.reduce<Array<{ id: number; bed_number: string; occupiedBy?: string }>>((acc, bed) => {
      const key = bed.bed_number?.trim() || `bed-${bed.id}`;
      if (seen.has(key)) return acc;
      seen.add(key);
      acc.push({ ...bed, occupiedBy: occupancy.bedAssignments[bed.id] });
      return acc;
    }, []);
    return deduped.slice(0, limit);
  }, [selectedRoom, roomOccupancyMap]);

  const getRoomSortValue = useCallback(
    (room: DormRoom, columnId: string) => {
      switch (columnId) {
        case "roomNumber":
          return room.room_number || "";
        case "bedCount":
          return room.bed_count || room.beds?.length || 0;
        case "roomStatus":
          return getRoomStatusLabel(room);
        case "site": {
          const siteId = room.site_id ?? buildingById.get(room.building_id)?.site_id ?? null;
          return siteId ? siteById.get(siteId)?.name ?? "" : "";
        }
        case "building":
          return buildingById.get(room.building_id)?.name ?? "";
        case "area":
          return Number(room.area_square || 0);
        default:
          return room.room_number || "";
      }
    },
    [buildingById, siteById, getRoomStatusLabel],
  );

  const activeRooms = useMemo(() => {
    const keyword = roomSearch.trim().toLowerCase();
    return rooms.filter((room) => {
      const building = buildingById.get(room.building_id);
      const siteId = room.site_id ?? building?.site_id ?? null;
      const occupancy = roomOccupancyMap.get(room.id) ?? EMPTY_OCCUPANCY;
      const derivedStatus = getRoomStatusLabel(room);
      const bearing = (room.cost_bearing_mode as "company" | "personal" | undefined) || "company";
      if (roomSiteFilter !== "all" && siteId !== roomSiteFilter) {
        return false;
      }
      if (roomStatusFilter !== "all" && derivedStatus !== roomStatusFilter) {
        return false;
      }
      if (roomTypeFilter !== "all" && bearing !== roomTypeFilter) {
        return false;
      }
      if (!keyword) {
        return true;
      }
      const siteName = siteId ? siteById.get(siteId)?.name ?? "" : "";
      return (
        room.room_number?.toLowerCase().includes(keyword) ||
        (building?.name || "").toLowerCase().includes(keyword) ||
        siteName.toLowerCase().includes(keyword) ||
        (room.inventory_note || "").toLowerCase().includes(keyword) ||
        occupancy.names.join("、").toLowerCase().includes(keyword)
      );
    });
  }, [rooms, buildingById, siteById, roomSiteFilter, roomStatusFilter, roomTypeFilter, roomSearch, roomOccupancyMap, getRoomStatusLabel]);

const sortedRooms = useMemo(() => {
  const sortableColumn = roomColumnsForRender.find((column) => column.id === roomSort.columnId && column.sortable !== false);
  if (!sortableColumn) {
    return [...activeRooms];
  }
    return [...activeRooms].sort((a, b) => {
      const valueA = getRoomSortValue(a, roomSort.columnId);
      const valueB = getRoomSortValue(b, roomSort.columnId);
      if (valueA === valueB) return 0;
      if (typeof valueA === "number" && typeof valueB === "number") {
        return roomSort.direction === "asc" ? valueA - valueB : valueB - valueA;
      }
      return roomSort.direction === "asc"
        ? String(valueA).localeCompare(String(valueB))
        : String(valueB).localeCompare(String(valueA));
    });
  }, [activeRooms, roomColumnsForRender, roomSort, getRoomSortValue]);

  const roomNavigationState = useMemo(() => {
    if (!editingRoomId) {
      return { hasPrev: false, hasNext: false };
    }
    const index = sortedRooms.findIndex((room) => room.id === editingRoomId);
    if (index === -1) {
      return { hasPrev: false, hasNext: false };
    }
    return {
      hasPrev: index > 0,
      hasNext: index < sortedRooms.length - 1,
    };
  }, [editingRoomId, sortedRooms]);

  const handleRoomNavigate = (direction: "prev" | "next") => {
    if (!editingRoomId) return;
    const index = sortedRooms.findIndex((room) => room.id === editingRoomId);
    if (index === -1) return;
    const offset = direction === "next" ? 1 : -1;
    const target = sortedRooms[index + offset];
    if (target) {
      handleEditRoom(target);
    }
  };

  useEffect(() => {
    if (!roomColumnsForRender.some((column) => column.id === roomSort.columnId && column.sortable !== false)) {
      const fallback = roomColumnsForRender.find((column) => column.sortable !== false);
      if (fallback) {
        setRoomSort({ columnId: fallback.id, direction: DEFAULT_ROOM_SORT.direction });
      }
    }
  }, [roomColumnsForRender, roomSort.columnId]);

  useEffect(() => {
    setSelectedRoomIds((prev) => prev.filter((roomId) => activeRooms.some((room) => room.id === roomId)));
  }, [activeRooms]);

  useEffect(() => {
    if (contractColumnOrder.length === 0 && effectiveContractColumnConfig.length > 0) {
      setContractColumnOrder(effectiveContractColumnConfig.map((column) => column.id));
    }
  }, [contractColumnOrder.length, effectiveContractColumnConfig]);

  useEffect(() => {
    if (contractColumnOrder.length === 0) return;
    const validIds = new Set(effectiveContractColumnConfig.map((column) => column.id));
    setContractColumnOrder((prev) => {
      const filtered = prev.filter((id) => validIds.has(id));
      const missing = effectiveContractColumnConfig.map((column) => column.id).filter((id) => !filtered.includes(id));
      if (missing.length === 0 && filtered.length === prev.length) {
        return prev;
      }
      return [...filtered, ...missing];
    });
  }, [effectiveContractColumnConfig, contractColumnOrder.length]);

  useEffect(() => {
    if (typeof window === "undefined") return;
    if (contractColumnOrder.length === 0) {
      localStorage.removeItem(CONTRACT_COLUMN_ORDER_STORAGE_KEY);
      return;
    }
    writeLocalStorageJSON(CONTRACT_COLUMN_ORDER_STORAGE_KEY, contractColumnOrder);
  }, [contractColumnOrder]);

  useEffect(() => {
    setSiteContractForm((prev) => {
      const computedNext = prev.lastPaymentDate && prev.paymentCycle ? computeNextPaymentDate(prev.lastPaymentDate, prev.paymentCycle) : "";
      if (prev.nextPaymentDate === computedNext) return prev;
      return { ...prev, nextPaymentDate: computedNext };
    });
  }, [siteContractForm.lastPaymentDate, siteContractForm.paymentCycle]);

  useEffect(() => {
    setSiteContractForm((prev) => {
      if (paymentCycleOptions.length === 0) {
        return prev.paymentCycle ? { ...prev, paymentCycle: "" as PaymentCycle | "" } : prev;
      }
      if (prev.paymentCycle && paymentCycleOptions.includes(prev.paymentCycle as PaymentCycle)) {
        return prev;
      }
      return { ...prev, paymentCycle: paymentCycleOptions[0] ?? ("" as PaymentCycle | "") };
    });
  }, [paymentCycleOptions]);

  useEffect(() => {
    setSelectedContractIds((prev) => prev.filter((contractId) => sortedContracts.some((contract) => contract.id === contractId)));
  }, [sortedContracts]);

  useEffect(() => {
    if (Object.keys(contractColumnVisibility).length === 0 && effectiveContractColumnConfig.length > 0) {
      setContractColumnVisibility(buildDefaultContractVisibility(effectiveContractColumnConfig));
    }
  }, [contractColumnVisibility, effectiveContractColumnConfig]);

  useEffect(() => {
    if (Object.keys(contractColumnVisibility).length === 0) return;
    const validIds = new Set(effectiveContractColumnConfig.map((column) => column.id));
    setContractColumnVisibility((prev) => {
      const next = { ...prev };
      let changed = false;
      Object.keys(next).forEach((key) => {
        if (!validIds.has(key)) {
          delete next[key];
          changed = true;
        }
      });
      effectiveContractColumnConfig.forEach((column) => {
        if (!(column.id in next)) {
          next[column.id] = column.defaultVisible !== false;
          changed = true;
        }
      });
      return changed ? next : prev;
    });
  }, [effectiveContractColumnConfig, contractColumnVisibility]);

  useEffect(() => {
    if (typeof window === "undefined") return;
    if (Object.keys(contractColumnVisibility).length === 0) {
      localStorage.removeItem(CONTRACT_COLUMN_VISIBILITY_STORAGE_KEY);
      return;
    }
    writeLocalStorageJSON(CONTRACT_COLUMN_VISIBILITY_STORAGE_KEY, contractColumnVisibility);
  }, [contractColumnVisibility]);

  useEffect(() => {
    if (!checkoutDialogOpen) {
      setCheckoutForm(createBlankCheckoutForm());
    }
  }, [checkoutDialogOpen]);

  const syncSiteReminderMemos = useCallback(
    (siteId: number, siteLabel: string, contractForm: SiteContractExtra) => {
      setSiteMemos((prev) => {
        const key = memoKey(siteId);
        const currentList = prev[key] ?? [];
        const withoutAuto = currentList.filter(
          (memo) =>
            !memo.id.startsWith(`${PAYMENT_REMINDER_MEMO_PREFIX}-${siteId}`) &&
            !memo.id.startsWith(`${CONTRACT_REMINDER_MEMO_PREFIX}-${siteId}`),
        );
        const additions: SiteMemoEntry[] = [];

        const appendReminder = (
          idSuffix: string,
          date: string,
          content: string,
          priority: SiteMemoPriority,
          target?: string,
        ) => {
          additions.push({
            id: idSuffix,
            date,
            time: "09:00",
            content,
            priority,
            createdAt: new Date().toISOString(),
            targetDate: target,
            completed: false,
          });
        };

        const maybeAddSchedule = (prefix: string, label: string, targetDate: string) => {
          const diffDays = calculateDaysUntil(targetDate);
          if (diffDays == null || diffDays > 30 || diffDays < 0) {
            return;
          }
          PAYMENT_REMINDER_SCHEDULE.forEach(({ days, priority }) => {
            if (diffDays < days) return;
            const reminderDate = subtractDaysFromDate(targetDate, days);
            if (!reminderDate) return;
            appendReminder(
              `${prefix}-${siteId}-${label}-${days}`,
              reminderDate,
              `【自动】${siteLabel || "该地点"} ${label}提醒：目标 ${formatDateLabel(targetDate)}，剩余 ${days} 天`,
              priority,
              targetDate,
            );
          });
        };

        if (contractForm.paymentReminderEnabled && contractForm.paymentCycle && contractForm.nextPaymentDate) {
          maybeAddSchedule(
            PAYMENT_REMINDER_MEMO_PREFIX,
            `${PAYMENT_CYCLE_LABELS[contractForm.paymentCycle]}缴费`,
            contractForm.nextPaymentDate,
          );
        }

        if (contractForm.contractReminderEnabled) {
          CONTRACT_REMINDER_TARGETS.forEach(({ field, label, key: targetKey }) => {
            const targetDate = contractForm[field];
            if (!targetDate) return;
            const diffDays = calculateDaysUntil(targetDate);
            if (diffDays == null || diffDays > 30 || diffDays < 0) {
              return;
            }
            PAYMENT_REMINDER_SCHEDULE.forEach(({ days, priority }) => {
              if (diffDays < days) return;
              const reminderDate = subtractDaysFromDate(targetDate, days);
              if (!reminderDate) return;
              appendReminder(
                `${CONTRACT_REMINDER_MEMO_PREFIX}-${siteId}-${targetKey}-${days}`,
                reminderDate,
                `【自动】${siteLabel || "该地点"} ${label}提醒：目标 ${formatDateLabel(targetDate)}，剩余 ${days} 天`,
                priority,
                targetDate,
              );
            });
          });
        }

        const nextList = [...withoutAuto, ...additions];
        nextList.sort((a, b) => {
          const aTime = new Date(`${a.date}T${a.time || "00:00"}`).getTime();
          const bTime = new Date(`${b.date}T${b.time || "00:00"}`).getTime();
          return bTime - aTime;
        });
        return { ...prev, [key]: nextList };
      });
    },
    [memoKey],
  );

  const syncRoomChargeReminderMemos = useCallback(
    (room: DormRoom, chargeKey: string, entry?: RoomChargeRecordEntry) => {
      if (!room || !RENT_LIKE_CHARGE_LABELS[chargeKey]) return;
      const building = buildingById.get(room.building_id);
      const siteId = room.site_id ?? building?.site_id;
      if (!siteId) return;
      setSiteMemos((prev) => {
        const key = memoKey(siteId);
        const currentList = prev[key] ?? [];
        const prefix = `${ROOM_CHARGE_REMINDER_PREFIX}-${room.id}-${chargeKey}`;
        let nextList = currentList.filter((memo) => !memo.id.startsWith(prefix));
        let mutated = nextList.length !== currentList.length;
        const siteLabel = siteById.get(siteId)?.name || "";
        const normalizedCycle = entry && isValidPaymentCycleValue(entry.paymentCycle) ? (entry.paymentCycle as PaymentCycle) : null;
        const nextPaymentDate = entry?.nextPaymentDate ?? "";
        const shouldAdd = Boolean(entry && entry.addMemo && normalizedCycle && nextPaymentDate);
        if (!shouldAdd || !entry || !normalizedCycle || !nextPaymentDate) {
          if (!mutated) {
            return prev;
          }
          if (nextList.length === 0) {
            const nextState = { ...prev };
            delete nextState[key];
            return nextState;
          }
          return { ...prev, [key]: nextList };
        }
        const cycle = normalizedCycle;
        const recurrence = RENT_MEMO_RECURRENCE_BY_CYCLE[cycle] ?? "none";
        const diffDays = calculateDaysUntil(nextPaymentDate);
        if (diffDays == null || diffDays > 30 || diffDays < 0) {
          if (!mutated) {
            return prev;
          }
          if (nextList.length === 0) {
            const nextState = { ...prev };
            delete nextState[key];
            return nextState;
          }
          return { ...prev, [key]: nextList };
        }
        const additions: SiteMemoEntry[] = [];
        PAYMENT_REMINDER_SCHEDULE.forEach(({ days, priority }) => {
          if (diffDays < days) return;
          const reminderDate = subtractDaysFromDate(nextPaymentDate, days);
          if (!reminderDate) return;
          additions.push({
            id: `${prefix}-${days}`,
            date: reminderDate,
            time: "09:00",
            content: `【自动】${siteLabel || "该地点"} ${room.room_number || `房间${room.id}`} ${RENT_LIKE_CHARGE_LABELS[chargeKey]}提醒：${PAYMENT_CYCLE_LABELS[cycle]}应收¥${
              entry?.paymentAmount || "--"
            }，目标日 ${formatDateLabel(nextPaymentDate)}（剩余 ${days} 天）`,
            priority,
            createdAt: new Date().toISOString(),
            recurrence,
            targetDate: nextPaymentDate,
            completed: false,
          });
        });
        if (additions.length > 0) {
          nextList = [...nextList, ...additions];
          mutated = true;
        }
        if (!mutated) {
          return prev;
        }
        nextList.sort((a, b) => {
          const aTime = new Date(`${a.date}T${a.time || "00:00"}`).getTime();
          const bTime = new Date(`${b.date}T${b.time || "00:00"}`).getTime();
          return bTime - aTime;
        });
        if (nextList.length === 0) {
          const nextState = { ...prev };
          delete nextState[key];
          return nextState;
        }
        return { ...prev, [key]: nextList };
      });
    },
    [buildingById, memoKey, siteById],
  );

  const handleOpenSiteDialog = (site?: DormSite, initialTab: "house" | "contract" | "memo" = "house") => {
    if (site) {
      setEditingSite(site);
      const primaryBuilding = siteBuildingMap[site.id]?.[0];
      const memoKeyValue = memoKey(site.id);
      const houseExtra = siteHouseExtras[memoKeyValue] ?? blankHouseForm;
      const resolvedBuildingNumber = site.building_number || houseExtra.buildingNumber || "";
      const resolvedPropertyCompany = site.property_company || houseExtra.propertyCompany || "";
      const resolvedPropertyContact = site.property_contact || houseExtra.propertyContact || "";
      setSiteChargeItems(parseChargeConfigItems(site.charge_config));
      setSiteForm({
        name: site.name || "",
        address: site.address || "",
        description: site.description || "",
        building_name: primaryBuilding?.name || "",
        building_number: resolvedBuildingNumber,
        building_code: houseExtra.buildingCodeSnapshot || primaryBuilding?.description || "",
        support_wechat: site.support_wechat || "",
      });
      setSiteHouseForm({
        propertyCompany: resolvedPropertyCompany,
        propertyContact: resolvedPropertyContact,
        buildingNumber: resolvedBuildingNumber,
        buildingCodeSnapshot: houseExtra.buildingCodeSnapshot || primaryBuilding?.description || "",
      });
      setSiteInventoryItems(parseSiteInventorySettings(houseExtra.inventoryItems));
      const contractExtra = siteContractExtras[memoKeyValue];
      setSiteContractForm(normalizeSiteContractForm(contractExtra));
    } else {
      setEditingSite(null);
      setSiteForm(blankSiteForm);
      setSiteHouseForm(blankHouseForm);
      setSiteContractForm(createBlankSiteContractForm());
      setSiteChargeItems(cloneChargeSettings(defaultChargePreference));
      setSiteInventoryItems(createDefaultInventorySettings());
    }
    setMemoForm(createBlankMemoForm());
    setSiteDialogTab(initialTab);
    setSiteDialogOpen(true);
  };

  const handleSaveSite = async () => {
    try {
      if (!siteForm.name.trim()) {
        toast.error("请填写地点名称");
        return;
      }
      const chargeConfigPayload = serializeChargeConfig(siteChargeItems);
      const propertyCompanyInput = siteHouseForm.propertyCompany.trim();
      const propertyContactInput = siteHouseForm.propertyContact.trim();
      const payload = {
        name: siteForm.name.trim(),
        address: siteForm.address.trim(),
        description: siteForm.description?.trim() || "",
        building_number: siteForm.building_number.trim(),
        property_company: propertyCompanyInput,
        property_contact: propertyContactInput,
        support_wechat: siteForm.support_wechat.trim(),
        charge_config: chargeConfigPayload,
      };
      const buildingNameInput = siteForm.building_name.trim();
      const buildingNumberInput = siteForm.building_number.trim();
      const buildingCodeInput = siteForm.building_code.trim();
      const composedBuildingDescription = [buildingNumberInput, buildingCodeInput].filter(Boolean).join(" / ");
      let savedSite: DormSite;
      if (editingSite) {
        savedSite = await updateDormSite(editingSite.id, payload);
        savedSite = {
          ...savedSite,
          charge_config: chargeConfigPayload,
          building_number: buildingNumberInput,
          property_company: propertyCompanyInput,
          property_contact: propertyContactInput,
          support_wechat: siteForm.support_wechat.trim(),
        };
        setSites((prev) => prev.map((site) => (site.id === savedSite.id ? savedSite : site)));
        toast.success("地点信息已更新");
      } else {
        savedSite = await createDormSite(payload);
        savedSite = {
          ...savedSite,
          charge_config: chargeConfigPayload,
          building_number: buildingNumberInput,
          property_company: propertyCompanyInput,
          property_contact: propertyContactInput,
          support_wechat: siteForm.support_wechat.trim(),
        };
        setSites((prev) => [savedSite, ...prev]);
        toast.success("已新增宿舍地点");
      }
      if (buildingNameInput || buildingCodeInput || buildingNumberInput) {
        const existingBuilding = siteBuildingMap[savedSite.id]?.[0];
        try {
          if (existingBuilding) {
            const updatedBuilding = await updateDormBuilding(existingBuilding.id, {
              name: buildingNameInput || existingBuilding.name,
              description: composedBuildingDescription || existingBuilding.description,
            });
            setBuildings((prev) => prev.map((item) => (item.id === updatedBuilding.id ? updatedBuilding : item)));
          } else {
            const createdBuilding = await createDormBuilding({
              site_id: savedSite.id,
              name: buildingNameInput || `${savedSite.name}楼栋`,
              description: composedBuildingDescription || undefined,
            });
            setBuildings((prev) => [createdBuilding, ...prev]);
          }
        } catch (buildingError) {
          console.warn("[Dormitory] building sync failed", buildingError);
          toast.warning("地点已保存，但楼栋信息同步失败，请稍后在楼栋管理中补齐");
        }
      }
      setDefaultChargePreference(cloneChargeSettings(siteChargeItems));
      const memoKeyValue = memoKey(savedSite.id);
      setSiteHouseExtras((prev) => ({
        ...prev,
        [memoKeyValue]: {
          propertyCompany: propertyCompanyInput,
          propertyContact: propertyContactInput,
          buildingNumber: buildingNumberInput,
          buildingCodeSnapshot: siteForm.building_code || "",
          inventoryItems: serializeSiteInventoryItems(siteInventoryItems),
        },
      }));
      const normalizedContract = normalizeSiteContractForm(siteContractForm);
      setSiteContractExtras((prev) => ({
        ...prev,
        [memoKeyValue]: normalizedContract,
      }));
      syncSiteReminderMemos(savedSite.id, siteForm.name.trim(), normalizedContract);
      setSiteDialogOpen(false);
      setEditingSite(null);
      setSiteForm(blankSiteForm);
      setSiteHouseForm(blankHouseForm);
      setSiteContractForm(createBlankSiteContractForm());
      setSiteChargeItems(createDefaultChargeSettings());
    } catch (error) {
      console.error("[Dormitory] save site failed", error);
      toast.error(error instanceof Error ? error.message : "保存地点失败");
    }
  };

  const handleSaveRoom = async (options?: { keepOpen?: boolean }) => {
    try {
      const keepDialogOpen = options?.keepOpen === true;
      if (!roomForm.site_id || !roomForm.building_id || !roomForm.room_number.trim()) {
        toast.error("请选择宿舍地点、楼栋并填写房号");
        return;
      }
      const numericBedCount = Number(roomForm.bed_count || 0);
      if (roomForm.room_category === "单间" && numericBedCount !== 1) {
        toast.error("单人间床位数量必须为 1");
        return;
      }
      if (roomForm.room_category === "多人间" && (!Number.isFinite(numericBedCount) || numericBedCount < 2)) {
        toast.error("多人间床位数量不得少于 2");
        return;
      }
      const trimmedCompanyName = roomForm.company_name?.trim() || "";
      const payload: Record<string, unknown> = {
        site_id: Number(roomForm.site_id),
        building_id: Number(roomForm.building_id),
        room_number: roomForm.room_number,
        room_category: roomForm.room_category,
        house_layout: roomForm.house_layout,
        bed_count: numericBedCount,
        area_square: Number(roomForm.area_square || 0),
        first_month_fee: Number(roomForm.first_month_fee || 0),
        monthly_rent: Number(roomForm.monthly_rent || 0),
        property_fee: Number(roomForm.property_fee || 0),
        quarterly_rent: Number(roomForm.quarterly_rent || 0),
        guarantee_fee: Number(roomForm.guarantee_fee || 0),
        deposit_fee: Number(roomForm.deposit_fee || 0),
        water_base: Number(roomForm.water_base || 0),
        electric_base: Number(roomForm.electric_base || 0),
        gas_base: Number(roomForm.gas_base || 0),
        trash_fee: Number(roomForm.trash_fee || 0),
        water_supply_fee: Number(roomForm.water_supply_fee || 0),
        sewage_fee: Number(roomForm.sewage_fee || 0),
        inventory_note: roomForm.inventory_note?.trim() || "",
        status: roomForm.status === "维护中" ? "维护中" : "",
        cost_bearing_mode: roomForm.cost_bearing_mode,
        company_name: trimmedCompanyName,
      };
      const serializedRecords = serializeRoomRecordNotes(roomChargeRecords);
      if (serializedRecords) {
        payload.notes = serializedRecords;
      } else if (editingRoomId) {
        payload.notes = "";
      }
      const chargeRateEntries: DormChargeRateEntry[] = [];
      activeRoomChargeItems.forEach((item) => {
        const override = roomChargeOverrides[item.key];
        if (!override?.trim()) return;
        const parsed = Number(override);
        if (!Number.isFinite(parsed)) return;
        chargeRateEntries.push({
          key: item.key,
          unit_price: parsed,
          unit_label: item.unitLabel,
          mode: item.mode,
        });
      });
      let normalizedChargeRates: DormChargeRates | undefined;
      if (chargeRateEntries.length > 0) {
        normalizedChargeRates = { items: chargeRateEntries };
        payload.charge_rates = normalizedChargeRates;
      } else if (editingRoomId) {
        normalizedChargeRates = { items: [] };
        payload.charge_rates = normalizedChargeRates;
      }
      const rentRecordEntry = roomChargeRecords.rent;
      const propertyRecordEntry = roomChargeRecords.property;
      if (editingRoomId) {
        const updated = await updateDormRoom(editingRoomId, payload);
        const patched = {
          ...updated,
          cost_bearing_mode: roomForm.cost_bearing_mode,
          company_name: trimmedCompanyName,
          ...(normalizedChargeRates ? { charge_rates: normalizedChargeRates } : {}),
        };
        setRooms((prev) => prev.map((room) => (room.id === patched.id ? patched : room)));
        syncRoomChargeReminderMemos(patched, "rent", rentRecordEntry);
        syncRoomChargeReminderMemos(patched, "property", propertyRecordEntry);
        toast.success("房间信息已更新");
        if (keepDialogOpen) {
          handleEditRoom(patched);
          return;
        }
      } else {
        const created = await createDormRoom(payload);
        const enrichedRoom = {
          ...created,
          cost_bearing_mode: roomForm.cost_bearing_mode,
          company_name: trimmedCompanyName,
          ...(normalizedChargeRates ? { charge_rates: normalizedChargeRates } : {}),
        };
        if (roomForm.bed_count && roomForm.bed_count > 0) {
          try {
            const requests = Array.from({ length: Number(roomForm.bed_count) }).map((_, index) =>
              createDormBed({ room_id: created.id, bed_number: `床位${index + 1}`, status: "空闲" }),
            );
            await Promise.all(requests);
          } catch (bedError) {
            console.warn("[Dormitory] auto bed creation failed", bedError);
          }
        }
        setRooms((prev) => [enrichedRoom, ...prev]);
        syncRoomChargeReminderMemos(enrichedRoom, "rent", rentRecordEntry);
        syncRoomChargeReminderMemos(enrichedRoom, "property", propertyRecordEntry);
        toast.success("已新增房间");
        if (keepDialogOpen) {
          handleEditRoom(enrichedRoom);
          return;
        }
      }
      setRoomForm({ ...initialRoomForm });
      setRoomChargeOverrides({});
      setEditingRoomId(null);
      setRoomDialogOpen(false);
    } catch (error) {
      console.error("[Dormitory] create room failed", error);
      toast.error(error instanceof Error ? error.message : "保存房间失败");
    }
  };

  const handleDeleteRoom = async () => {
    if (!roomDeleteTarget) return;
    setRoomDeleting(true);
    try {
      await deleteDormRoom(roomDeleteTarget.id);
      setRooms((prev) => prev.filter((item) => item.id !== roomDeleteTarget.id));
      setSelectedRoomIds((prev) => prev.filter((id) => id !== roomDeleteTarget.id));
      toast.success("房间已删除");
    } catch (error) {
      console.error("[Dormitory] delete room failed", error);
      toast.error(error instanceof Error ? error.message : "删除房间失败");
    } finally {
      setRoomDeleting(false);
      setRoomDeleteTarget(null);
    }
  };

  const handleAddSiteMemo = () => {
    if (!editingSite) {
      toast.error("请选择需要编辑的地点");
      return;
    }
    if (!memoForm.startDate || !memoForm.content.trim()) {
      toast.error("请填写开始时间与待办内容");
      return;
    }
    if (memoForm.endDate && memoForm.recurrence && memoForm.recurrence !== "none") {
      toast.error("选择结束时间后不能开启循环提醒");
      return;
    }
    const entry: SiteMemoEntry = {
      id: createMemoId(),
      date: memoForm.startDate,
      time: memoForm.startTime,
      content: memoForm.content.trim(),
      priority: memoForm.priority,
      createdAt: new Date().toISOString(),
      recurrence: memoForm.endDate ? "none" : memoForm.recurrence,
      targetDate: memoForm.endDate || memoForm.startDate,
      completed: false,
    };
    setSiteMemos((prev) => {
      const key = memoKey(editingSite.id);
      const nextList = prev[key] ? [...prev[key]] : [];
      nextList.push(entry);
      nextList.sort((a, b) => {
        const aTime = getNextMemoOccurrence(a)?.getTime() ?? 0;
        const bTime = getNextMemoOccurrence(b)?.getTime() ?? 0;
        return bTime - aTime;
      });
      return { ...prev, [key]: nextList };
    });
    setMemoForm(createBlankMemoForm());
    toast.success("备忘已添加");
  };

  const handleRemoveSiteMemo = (siteId: number, memoId: string) => {
    setSiteMemos((prev) => {
      const key = memoKey(siteId);
      const list = prev[key]?.filter((memo) => memo.id !== memoId) ?? [];
      const next = { ...prev };
      if (list.length === 0) {
        delete next[key];
      } else {
        next[key] = list;
      }
      return next;
    });
  };

  const handleToggleSiteMemoCompletion = (siteId: number, memoId: string, completed: boolean) => {
    setSiteMemos((prev) => {
      const key = memoKey(siteId);
      const list = prev[key];
      if (!list) return prev;
      const now = new Date().toISOString();
      const nextList = list.map((memo) =>
        memo.id === memoId
          ? {
              ...memo,
              completed,
              completedAt: completed ? now : undefined,
            }
          : memo,
      );
      return { ...prev, [key]: nextList };
    });
  };

  const handleCheckoutAttachmentChange = async (event: ChangeEvent<HTMLInputElement>) => {
    const files = event.target.files;
    if (!files || files.length === 0) return;
    const entries: AttachmentEntry[] = [];
    for (const file of Array.from(files)) {
      try {
        const data = await fileToDataUrl(file);
        entries.push({ name: file.name, data });
      } catch (error) {
        console.error("[Dormitory] checkout attachment failed", error);
        toast.error(`附件 ${file.name} 读取失败`);
      }
    }
    if (entries.length > 0) {
      setCheckoutForm((prev) => ({ ...prev, attachments: [...prev.attachments, ...entries] }));
    }
    event.target.value = "";
  };

  const handleRemoveCheckoutAttachment = (name: string) => {
    setCheckoutForm((prev) => ({
      ...prev,
      attachments: prev.attachments.filter((entry) => entry.name !== name),
    }));
  };

  const handleContractSelectAll = (checked: boolean, list: DormContract[]) => {
    setSelectedContractIds(checked ? list.map((contract) => contract.id) : []);
  };

  const toggleContractSelection = (contractId: number) => {
    setSelectedContractIds((prev) => (prev.includes(contractId) ? prev.filter((id) => id !== contractId) : [...prev, contractId]));
  };

  const exportContracts = (rows: DormContract[], filename: string) => {
    if (rows.length === 0) {
      toast.error("没有可导出的数据");
      return;
    }
    const sheetData = rows.map((contract) => {
      const noteMeta = parseContractNoteMeta(contract.notes);
      return {
        姓名: contract.employee_name,
        部门: contract.employee_department,
        联系电话: contract.employee_phone,
        房间: contract.room?.room_number,
        床位: contract.bed?.bed_number,
        入住日期: formatDateInputValue(contract.start_date),
        退宿日期: formatDateInputValue(contract.end_date),
        状态: (CONTRACT_STATUS_META[contract.status || "active"] ?? CONTRACT_STATUS_META.active).label,
        租金: contract.rent_amount ?? "",
        押金: contract.deposit_amount ?? "",
        押金月份: noteMeta.depositPlanMonth ?? "",
        租金月份: noteMeta.rentPlanMonth ?? "",
      };
    });
    const worksheet = XLSX.utils.json_to_sheet(sheetData);
    const workbook = XLSX.utils.book_new();
    XLSX.utils.book_append_sheet(workbook, worksheet, "入住列表");
    XLSX.writeFile(workbook, `${filename}-${new Date().toISOString().slice(0, 10)}.xlsx`);
  };

  const handleContractExport = (scope: "selected" | "filtered") => {
    const target = scope === "selected"
      ? sortedContracts.filter((contract) => selectedContractIds.includes(contract.id))
      : sortedContracts;
    const filename = scope === "selected" ? "入住列表-选中" : "入住列表-当前";
    exportContracts(target, filename);
  };

  const printContracts = async (
    rows: DormContract[],
    options: { title: string; watermark: string; orientation: PrintOrientation },
  ) => {
    if (rows.length === 0) {
      toast.error("请选择需要打印的记录");
      return;
    }
    if (typeof window === "undefined") {
      toast.error("当前环境不支持打印预览");
      return;
    }
    const dataset = buildContractPrintDataset(rows);
    if (!dataset) {
      toast.error("未找到可打印的数据");
      return;
    }
    const loadingId = toast.loading("正在生成打印预览，请稍候...");
    try {
      const blob = await createReportPdf({
        title: options.title || dataset.defaultTitle,
        watermark: options.watermark || "内部资料 请勿外传",
        columns: dataset.columns,
        rows: dataset.rows,
        orientation: options.orientation,
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
    } catch (error) {
      console.error("[Dormitory] generate contract pdf failed", error);
      toast.error("生成打印预览失败，请稍后重试");
    } finally {
      toast.dismiss(loadingId);
    }
  };

  const handleContractPrintRequest = (scope: "selected" | "filtered") => {
    const target =
      scope === "selected"
        ? sortedContracts.filter((contract) => selectedContractIds.includes(contract.id))
        : sortedContracts;
    if (target.length === 0) {
      toast.error(scope === "selected" ? "请先勾选需要打印的入住记录" : "暂无可打印的入住记录");
      return;
    }
    const title = scope === "selected" ? `入住列表（选中 ${target.length} 条）` : `入住列表（当前 ${target.length} 条）`;
    setContractPrintContext(target);
    setContractPrintSuggestedTitle(title);
    setContractPrintDialogOpen(true);
  };

  const handleMeterExport = (scope: "selected" | "filtered", rows: MeterTableRow[]) => {
    const targetRows =
      scope === "selected"
        ? rows.filter((row) => meterSelectedIds.includes(row.source.id))
        : rows;
    if (targetRows.length === 0) {
      toast.error(scope === "selected" ? "请先勾选需要导出的抄表记录" : "暂无可导出的抄表记录");
      return;
    }
    exportMeterRecords(targetRows.map((row) => row.display), scope === "selected" ? "抄表记录-选中" : "抄表记录-当前");
  };

  const handleMeterPrintRequest = (scope: "selected" | "filtered", rows: MeterTableRow[]) => {
    const targetRows =
      scope === "selected"
        ? rows.filter((row) => meterSelectedIds.includes(row.source.id))
        : rows;
    if (targetRows.length === 0) {
      toast.error(scope === "selected" ? "请先勾选需要打印的抄表记录" : "暂无可打印的抄表记录");
      return;
    }
    const title = scope === "selected" ? `抄表记录（选中 ${targetRows.length} 条）` : `抄表记录（当前 ${targetRows.length} 条）`;
    setMeterPrintContext(targetRows.map((row) => row.display));
    setMeterPrintSuggestedTitle(title);
    setMeterPrintDialogOpen(true);
  };

  const handleGenerateBillsFromSelection = async () => {
    const target =
      meterSelectedIds.length > 0
        ? meterRecords.filter((record) => meterSelectedIds.includes(record.id))
        : [];
    if (target.length === 0) {
      toast.error("请先勾选需要生成账单的抄表记录");
      return;
    }
    const payloads = target
      .map((record) => composeBillPayloadFromMeterRecord(record))
      .filter((item): item is NonNullable<ReturnType<typeof composeBillPayloadFromMeterRecord>> => Boolean(item));
    if (payloads.length === 0) {
      toast.error("选中记录缺少可计费项，无法生成账单");
      return;
    }
    const loadingId = toast.loading("正在生成账单...");
    setBillGenerating(true);
    try {
      await Promise.all(payloads.map((payload) => createDormBill(payload)));
      toast.success(`账单已生成（${payloads.length} 条）`);
    } catch (error) {
      console.error("[Dormitory] generate bills failed", error);
      toast.error("生成账单失败，请稍后重试");
    } finally {
      setBillGenerating(false);
      toast.dismiss(loadingId);
    }
  };

  const handleOpenCheckoutDialog = (contract?: DormContract) => {
    if (contract) {
      const blank = createBlankCheckoutForm();
      setCheckoutForm({
        ...blank,
        contract_id: String(contract.id),
        deposit_collected: contract.deposit_amount ? String(contract.deposit_amount) : blank.deposit_collected,
        deposit_return: contract.deposit_amount ? String(contract.deposit_amount) : blank.deposit_return,
      });
    } else {
      setCheckoutForm(createBlankCheckoutForm());
    }
    setCheckoutDialogOpen(true);
  };

  const handleRevokeCheckout = async (contract: DormContract) => {
    if (!contract.id) {
      toast.error("无效的入住记录");
      return;
    }
    const loadingId = toast.loading("正在撤销退宿...");
    try {
      const updated = await updateDormContract(contract.id, { status: "active", end_date: null });
      setContracts((prev) => prev.map((item) => (item.id === updated.id ? { ...item, ...updated } : item)));
      toast.success("退宿已撤销");
    } catch (error) {
      console.error("[Dormitory] revoke checkout failed", error);
      toast.error(error instanceof Error ? error.message : "撤销退宿失败");
    } finally {
      toast.dismiss(loadingId);
    }
  };

  const handleCheckoutSubmit = async () => {
    if (!checkoutForm.contract_id) {
      toast.error("请选择需要办理退宿的记录");
      return;
    }
    setCheckoutSubmitting(true);
    try {
      const contractId = Number(checkoutForm.contract_id);
      const payload = {
        checkout_date: checkoutForm.checkout_date || new Date().toISOString().slice(0, 10),
        inspector: checkoutForm.inspector,
        water_end: checkoutForm.water_end,
        electric_end: checkoutForm.electric_end,
        damage_report: checkoutForm.damage_report,
        items_status: checkoutForm.items_status,
        fee_summary: checkoutForm.fee_summary,
        deposit_collected: Number(checkoutForm.deposit_collected || 0),
        deposit_deduct: Number(checkoutForm.deposit_deduct || 0),
        deposit_return: Number(checkoutForm.deposit_return || 0),
        deposit_return_date: checkoutForm.deposit_return_date,
        guarantee_collected: Number(checkoutForm.guarantee_collected || 0),
        guarantee_deduct: Number(checkoutForm.guarantee_deduct || 0),
        guarantee_return: Number(checkoutForm.guarantee_return || 0),
        guarantee_return_date: checkoutForm.guarantee_return_date,
        attachments: encodeAttachmentEntries(checkoutForm.attachments),
      };
      await createDormCheckout(contractId, payload);
      setContracts((prev) =>
        prev.map((contract) =>
          contract.id === contractId
            ? { ...contract, status: "completed", end_date: payload.checkout_date }
            : contract,
        ),
      );
      toast.success("退宿办理成功");
      await refreshRoomsAndContracts();
      setCheckoutDialogOpen(false);
      setCheckoutForm(createBlankCheckoutForm());
    } catch (error) {
      console.error("[Dormitory] checkout failed", error);
      toast.error(error instanceof Error ? error.message : "退宿办理失败");
    } finally {
      setCheckoutSubmitting(false);
    }
  };

  const handleGenerateCheckoutPdf = async () => {
    if (!checkoutForm.contract_id) {
      toast.error("请选择入住记录后再生成表单");
      return;
    }
    const contractId = Number(checkoutForm.contract_id);
    const target = contractById.get(contractId);
    if (!target) {
      toast.error("未找到对应的入住记录");
      return;
    }
    if (typeof window === "undefined") {
      toast.error("当前环境不支持生成表单");
      return;
    }
    const rows: string[][] = [
      ["入住人", target.employee_name || "-"],
      ["部门", target.employee_department || "-"],
      ["联系方式", target.employee_phone || "-"],
      ["房间/床位", `${target.room?.room_number || "-"} / ${target.bed?.bed_number || "未分配"}`],
      ["入住日期", formatDateLabel(target.start_date)],
      ["退宿日期", formatDateLabel(checkoutForm.checkout_date)],
      ["水表退宿度数", checkoutForm.water_end || "--"],
      ["电表退宿度数", checkoutForm.electric_end || "--"],
      ["押金收取金额", checkoutForm.deposit_collected || "--"],
      ["押金扣除金额", checkoutForm.deposit_deduct || "--"],
      ["押金退还金额", checkoutForm.deposit_return || "--"],
      ["押金退还日期", checkoutForm.deposit_return_date || "--"],
      ...(showGuaranteeFields
        ? ([
            ["保证金收取金额", checkoutForm.guarantee_collected || "--"],
            ["保证金扣除金额", checkoutForm.guarantee_deduct || "--"],
            ["保证金退还金额", checkoutForm.guarantee_return || "--"],
            ["保证金退还日期", checkoutForm.guarantee_return_date || "--"],
          ] as string[][])
        : []),
      ["设施情况", checkoutForm.items_status || "--"],
      ["损坏说明", checkoutForm.damage_report || "--"],
      ["费用说明", checkoutForm.fee_summary || "--"],
      ["办理人", checkoutForm.inspector || "--"],
    ];
    const loadingId = toast.loading("正在生成退宿表单...");
    try {
      const blob = await createReportPdf({
        title: `退宿确认表 - ${target.employee_name || ""}`,
        watermark: "内部资料 请勿外传",
        columns: ["字段", "内容"],
        rows,
        orientation: "portrait",
      });
      const url = URL.createObjectURL(blob);
      const previewWindow = window.open(url);
      if (!previewWindow) {
        toast.error("浏览器阻止了打印窗口，请允许弹窗后重试");
        URL.revokeObjectURL(url);
      } else {
        previewWindow.onload = () => previewWindow.focus();
        const cleanup = () => URL.revokeObjectURL(url);
        previewWindow.addEventListener("beforeunload", cleanup, { once: true });
        setTimeout(cleanup, 60_000);
      }
    } catch (error) {
      console.error("[Dormitory] generate checkout pdf failed", error);
      toast.error("生成退宿表单失败，请稍后再试");
    } finally {
      toast.dismiss(loadingId);
    }
  };

  const requestDeleteContract = (contract: DormContract) => {
    setContractDeleteTarget(contract);
  };

  const handleDeleteContract = async () => {
    if (!contractDeleteTarget) return;
    setContractDeleting(true);
    try {
      await deleteDormContract(contractDeleteTarget.id);
      setContracts((prev) => prev.filter((item) => item.id !== contractDeleteTarget.id));
      setSelectedContractIds((prev) => prev.filter((id) => id !== contractDeleteTarget.id));
      toast.success("入住记录已删除");
    } catch (error) {
      console.error("[Dormitory] delete contract failed", error);
      toast.error(error instanceof Error ? error.message : "删除失败");
    } finally {
      setContractDeleting(false);
      setContractDeleteTarget(null);
    }
  };

  const handleContractFieldToggle = (columnId: string) => {
    setContractColumnVisibility((prev) => ({ ...prev, [columnId]: prev[columnId] === false }));
  };

  const resetContractFieldVisibility = () => {
    setContractColumnVisibility(buildDefaultContractVisibility(effectiveContractColumnConfig));
    setContractColumnOrder(effectiveContractColumnConfig.map((column) => column.id));
  };

  const handleDownloadRoomTemplate = () => {
    const workbook = XLSX.utils.book_new();
    const headerRow = ROOM_IMPORT_HEADERS.map((header) => header.label);
    const sampleRow = [
      "观音桥宿舍点",
      "1号楼",
      "101",
      "一室一厅",
      "单间",
      2,
      60,
      800,
      1200,
      1500,
      200,
      500,
      1.2,
      0.8,
      1.5,
      30,
      0.6,
      0.4,
      "空调、衣柜、单人床",
    ];
    const worksheet = XLSX.utils.aoa_to_sheet([headerRow, sampleRow]);
    XLSX.utils.book_append_sheet(workbook, worksheet, "房间导入模板");
    XLSX.writeFile(workbook, `宿舍房间导入模板-${new Date().toISOString().slice(0, 10)}.xlsx`);
  };

  const handleImportRooms = async () => {
    if (!roomImportFile) {
      setRoomImportError("请选择需要导入的模板文件");
      return;
    }
    setRoomImporting(true);
    setRoomImportResult(null);
    try {
      setRoomImportError("");
      const buffer = await roomImportFile.arrayBuffer();
      const workbook = XLSX.read(buffer, { type: "array" });
      const sheetName = workbook.SheetNames[0];
      if (!sheetName) {
        throw new Error("未检测到工作表，请检查模板");
      }
      const rows = XLSX.utils.sheet_to_json<Record<string, unknown>>(workbook.Sheets[sheetName], { defval: "" });
      if (rows.length === 0) {
        throw new Error("模板中没有数据");
      }
      let inserted = 0;
      let skipped = 0;
      for (const row of rows) {
        const siteName = String(readRoomImportValue(row, "site") ?? "").trim();
        const buildingName = String(readRoomImportValue(row, "building") ?? "").trim();
        const roomNumber = String(readRoomImportValue(row, "room_number") ?? "").trim();
        if (!siteName || !buildingName || !roomNumber) {
          skipped += 1;
          continue;
        }
        try {
          const site = await ensureSiteByName(siteName);
          const building = await ensureBuildingByName(site.id, buildingName);
          const houseLayout = String(readRoomImportValue(row, "house_layout") ?? "").trim() || "一室一厅";
          const roomCategory = String(readRoomImportValue(row, "room_category") ?? "").trim() || "单间";
          const bedCount = toNumberValue(readRoomImportValue(row, "bed_count"));
          const areaSquare = toNumberValue(readRoomImportValue(row, "area_square"));
          const firstMonthFee = toNumberValue(readRoomImportValue(row, "first_month_fee"));
          const monthlyRent = toNumberValue(readRoomImportValue(row, "monthly_rent"));
          const quarterlyRent = toNumberValue(readRoomImportValue(row, "quarterly_rent"));
          const propertyFee = toNumberValue(readRoomImportValue(row, "property_fee"));
          const depositFee = toNumberValue(readRoomImportValue(row, "deposit_fee"));
          const guaranteeFee = toNumberValue(readRoomImportValue(row, "guarantee_fee"));
          const electricBase = toNumberValue(readRoomImportValue(row, "electric_base"));
          const waterBaseValue = toNumberValue(readRoomImportValue(row, "water_base"));
          const gasBase = toNumberValue(readRoomImportValue(row, "gas_base"));
          const trashFee = toNumberValue(readRoomImportValue(row, "trash_fee"));
          const waterSupplyFee = toNumberValue(readRoomImportValue(row, "water_supply_fee"));
          const sewageFee = toNumberValue(readRoomImportValue(row, "sewage_fee"));
          const inventoryRaw = String(readRoomImportValue(row, "inventory") ?? "");
          const inventoryItems = parseInventoryItems(inventoryRaw);
          const payload = {
            site_id: site.id,
            building_id: building.id,
            room_number: roomNumber,
            room_type: roomCategory,
            room_category: roomCategory,
            house_layout: houseLayout,
            bed_count: bedCount || 0,
            area_square: areaSquare,
            first_month_fee: firstMonthFee,
            monthly_rent: monthlyRent,
            quarterly_rent: quarterlyRent,
            property_fee: propertyFee,
            deposit_fee: depositFee,
            guarantee_fee: guaranteeFee,
            electric_base: electricBase,
            water_base: waterBaseValue,
            gas_base: gasBase,
            trash_fee: trashFee,
            water_supply_fee: waterSupplyFee,
            sewage_fee: sewageFee,
            inventory_note: inventoryItems.join("、"),
            status: "空闲",
          };
          const created = await createDormRoom(payload);
          setRooms((prev) => [created, ...prev]);
          inserted += 1;
        } catch (error) {
          console.error("[Dormitory] room import row failed", error);
          skipped += 1;
        }
      }
      setRoomImportResult({ inserted, skipped });
      toast.success(`导入完成：成功 ${inserted} 条，跳过 ${skipped} 条`);
      if (roomImportFileInputRef.current) {
        roomImportFileInputRef.current.value = "";
      }
    } catch (error) {
      console.error("[Dormitory] import rooms failed", error);
      toast.error(error instanceof Error ? error.message : "导入失败，请检查模板");
    } finally {
      setRoomImporting(false);
    }
  };

  const handleRoomImportFileChange = (event: ChangeEvent<HTMLInputElement>) => {
    const file = event.target.files?.[0] ?? null;
    setRoomImportFile(file);
    setRoomImportError("");
  };

  const handleRoomImportDialogToggle = (open: boolean) => {
    setRoomImportDialogOpen(open);
    if (!open) {
      setRoomImportResult(null);
      setRoomImportFile(null);
      setRoomImportError("");
      if (roomImportFileInputRef.current) {
        roomImportFileInputRef.current.value = "";
      }
      setRoomImportFile(null);
    }
  };

  const handleMeterImportDialogToggle = (open: boolean) => {
    setMeterImportDialogOpen(open);
    if (!open) {
      setMeterImportResult(null);
      setMeterImportFile(null);
      setMeterImportError("");
      if (meterImportFileInputRef.current) {
        meterImportFileInputRef.current.value = "";
      }
    }
  };

  const handleMeterImportFileChange = (event: ChangeEvent<HTMLInputElement>) => {
    const file = event.target.files?.[0] ?? null;
    setMeterImportFile(file);
    setMeterImportError("");
  };

  const handleDownloadMeterTemplate = () => {
    const headers = buildMeterTemplateHeaders();
    const sampleRow = headers.map((header) => {
      if (header.includes("起度")) return 100;
      if (header.includes("止度")) return 120;
      if (header.includes("金额")) return 80;
      switch (header) {
        case "楼栋":
          return "一号楼";
        case "房号":
          return "101";
        case "抄表日期":
          return new Date().toISOString().slice(0, 10);
        case "账单起始":
          return "2025-01-01";
        case "账单截止":
          return "2025-01-31";
        case "抄表人":
          return "李永娇";
        case "入住人员":
          return "张三";
        default:
          return "";
      }
    });
    const workbook = XLSX.utils.book_new();
    const worksheet = XLSX.utils.aoa_to_sheet([headers, sampleRow]);
    XLSX.utils.book_append_sheet(workbook, worksheet, "抄表模板");
    XLSX.writeFile(workbook, `抄表导入模板-${new Date().toISOString().slice(0, 10)}.xlsx`);
  };

  const handleImportMeterRecords = async () => {
    if (!meterImportFile) {
      setMeterImportError("请选择需要导入的表格文件");
      return;
    }
    setMeterImporting(true);
    try {
      const buffer = await meterImportFile.arrayBuffer();
      const workbook = XLSX.read(buffer, { type: "array" });
      const sheet = workbook.Sheets[workbook.SheetNames[0]];
      if (!sheet) {
        throw new Error("未找到工作表");
      }
      const rows = XLSX.utils.sheet_to_json<Record<string, unknown>>(sheet, { defval: "" });
      let inserted = 0;
      let skipped = 0;
      const newRecords: DormMeterRecord[] = [];
      for (const row of rows) {
        const roomNumber = String(row["房号"] ?? row["room"] ?? "").trim();
        if (!roomNumber) {
          skipped += 1;
          continue;
        }
        const buildingName = String(row["楼栋"] ?? row["building"] ?? "").trim();
        const candidates = roomByNumber.get(roomNumber) ?? [];
        const matchedRoom = candidates.find((room) => {
          if (!buildingName) return true;
          const building = buildingById.get(room.building_id);
          return building?.name?.trim() === buildingName;
        });
        if (!matchedRoom) {
          skipped += 1;
          continue;
        }
        const meterDate = normalizeExcelDate(row["抄表日期"] ?? row["meter_date"] ?? row["抄表时间"]);
        const billingStart = normalizeExcelDate(row["账单起始"] ?? row["账单起"] ?? row["billing_start"]);
        const billingEnd = normalizeExcelDate(row["账单截止"] ?? row["账单止"] ?? row["billing_end"]);
        const toNullable = (value: unknown) => {
          if (value === "" || value === null || value === undefined) return null;
          const parsed = Number(value);
          return Number.isFinite(parsed) ? parsed : null;
        };
        const electricStart = toNullable(row["电表起度"] ?? row["electric_start"]);
        const electricEnd = toNullable(row["电表止度"] ?? row["electric_end"]);
        const waterStart = toNullable(row["水表起度"] ?? row["water_start"]);
        const waterEnd = toNullable(row["水表止度"] ?? row["water_end"]);
        const gasStart = toNullable(row["气表起度"] ?? row["gas_start"]);
        const gasEnd = toNullable(row["气表止度"] ?? row["gas_end"]);
        const gasAmount = toNullable(row["气费"] ?? row["gas_fee"]);
        const chargeDetails = buildChargeDetailsFromValues(matchedRoom.id, {
          electricStart,
          electricEnd,
          waterStart,
          waterEnd,
          gasStart,
          gasEnd,
          gasAmount,
        });
        const chargeSettingsForImport = getChargeSettingsForRoom(matchedRoom);
        chargeSettingsForImport.forEach((setting) => {
          if (isBaseChargeKey(setting.key as LegacyChargeKey)) return;
          const amountValue = toNullable(row[`${setting.label}金额`] ?? row[setting.label] ?? row[setting.key]);
          if (amountValue == null) return;
          chargeDetails.push({
            key: setting.key,
            label: setting.label,
            amount: amountValue,
            unit_price: setting.unitPrice,
            unit_label: setting.unitLabel,
            mode: setting.mode,
          });
        });
        const payload: DormMeterRecordPayload = {
          room_id: matchedRoom.id,
          meter_date: meterDate || new Date().toISOString().slice(0, 10),
          billing_start: billingStart || meterForm.billing_start,
          billing_end: billingEnd || meterForm.billing_end,
          inspector: String(row["抄表人"] ?? row["inspector"] ?? "李永娇").trim() || "李永娇",
          charge_details: chargeDetails,
        };
        try {
          const saved = await createDormMeterRecord(payload);
          newRecords.push(saved);
          inserted += 1;
        } catch (error) {
          console.error("[Dormitory] import meter row failed", error);
          skipped += 1;
        }
      }
      if (newRecords.length > 0) {
        setMeterSourceRecords((prev) => [...newRecords, ...prev]);
      }
      setMeterImportResult({ inserted, skipped });
      toast.success(`抄表导入完成：成功 ${inserted} 条，跳过 ${skipped} 条`);
      setMeterImportError(inserted === 0 ? "请检查模板内容是否正确" : "");
      if (meterImportFileInputRef.current) {
        meterImportFileInputRef.current.value = "";
      }
    } catch (error) {
      console.error("[Dormitory] import meter records failed", error);
      toast.error(error instanceof Error ? error.message : "抄表导入失败，请稍后重试");
      setMeterImportError("导入失败，请检查文件");
    } finally {
      setMeterImporting(false);
    }
  };

  const handleContractImportDialogToggle = (open: boolean) => {
    setContractImportDialogOpen(open);
    if (!open) {
      setContractImportResult(null);
      if (contractImportFileInputRef.current) {
        contractImportFileInputRef.current.value = "";
      }
    }
  };

  const handleDownloadContractTemplate = () => {
    const workbook = XLSX.utils.book_new();
    const headerRow = CONTRACT_IMPORT_HEADERS.map((header) => header.label);
    const sampleRow = [
      "张三",
      "生产部",
      "138****8888",
      "110105199001010011",
      "北京市海淀区",
      "A1-302",
      "床位1",
      new Date().toISOString().slice(0, 10),
      "",
      1200,
      300,
      "按月",
      "试用期入住"
    ];
    const worksheet = XLSX.utils.aoa_to_sheet([headerRow, sampleRow]);
    XLSX.utils.book_append_sheet(workbook, worksheet, "入住导入模板");
    XLSX.writeFile(workbook, `入住列表导入模板-${new Date().toISOString().slice(0, 10)}.xlsx`);
  };

  const handleImportContracts = async () => {
    const file = contractImportFileInputRef.current?.files?.[0];
    if (!file) {
      toast.error("请选择需要导入的文件");
      return;
    }
    setContractImporting(true);
    setContractImportResult(null);
    const roomBedLocks = new Map<number, Set<number>>();
    rooms.forEach((room) => {
      const lock = new Set<number>();
      const occupancy = roomOccupancyMap.get(room.id);
      room.beds?.forEach((bed) => {
        if (occupancy?.bedAssignments[bed.id]) {
          lock.add(bed.id);
        }
      });
      roomBedLocks.set(room.id, lock);
    });
    try {
      const buffer = await file.arrayBuffer();
      const workbook = XLSX.read(buffer, { type: "array" });
      const sheetName = workbook.SheetNames[0];
      if (!sheetName) {
        throw new Error("未检测到工作表，请检查模板");
      }
      const rows = XLSX.utils.sheet_to_json<Record<string, unknown>>(workbook.Sheets[sheetName], { defval: "" });
      if (rows.length === 0) {
        throw new Error("模板中没有数据");
      }
      let inserted = 0;
      let skipped = 0;
      for (const row of rows) {
        try {
          const employeeName = String(readContractImportValue(row, "employee_name") ?? "").trim();
          const roomNumber = String(readContractImportValue(row, "room_number") ?? "").trim();
          const startDate = normalizeExcelDate(readContractImportValue(row, "start_date"));
          if (!employeeName || !roomNumber || !startDate) {
            skipped += 1;
            continue;
          }
          const roomCandidates = roomByNumber.get(roomNumber);
          if (!roomCandidates || roomCandidates.length === 0) {
            skipped += 1;
            continue;
          }
          const targetRoom = roomCandidates[0];
          const bedNumber = String(readContractImportValue(row, "bed_number") ?? "").trim();
          let bedId: number | undefined;
          if (bedNumber) {
            const targetBed = targetRoom.beds?.find((bed) => bed.bed_number === bedNumber);
            if (!targetBed) {
              skipped += 1;
              continue;
            }
            const lock = roomBedLocks.get(targetRoom.id) ?? new Set<number>();
            if (lock.has(targetBed.id)) {
              skipped += 1;
              continue;
            }
            bedId = targetBed.id;
            lock.add(targetBed.id);
            roomBedLocks.set(targetRoom.id, lock);
          } else if ((targetRoom.beds?.length || 0) > 0) {
            const lock = roomBedLocks.get(targetRoom.id) ?? new Set<number>();
            const freeBed = targetRoom.beds?.find((bed) => !lock.has(bed.id));
            if (freeBed) {
              bedId = freeBed.id;
              lock.add(freeBed.id);
              roomBedLocks.set(targetRoom.id, lock);
            }
          }
          const payload = {
            employee_name: employeeName,
            employee_department: String(readContractImportValue(row, "employee_department") ?? "").trim(),
            employee_phone: String(readContractImportValue(row, "employee_phone") ?? "").trim(),
            employee_id_number: String(readContractImportValue(row, "employee_id_number") ?? "").trim(),
            employee_residence: String(readContractImportValue(row, "employee_residence") ?? "").trim(),
            room_id: targetRoom.id,
            bed_id: bedId,
            start_date: startDate,
            end_date: normalizeExcelDate(readContractImportValue(row, "end_date")) || startDate,
            rent_amount: toNumberValue(readContractImportValue(row, "rent_amount")),
            deposit_amount: toNumberValue(readContractImportValue(row, "deposit_amount")),
            payment_method: String(readContractImportValue(row, "payment_method") ?? "按月") || "按月",
            status: "active",
            notes: String(readContractImportValue(row, "notes") ?? ""),
          };
          const created = await createDormContract(payload);
          const enriched = enrichContractRelations(created);
          setContracts((prev) => [enriched, ...prev]);
          inserted += 1;
        } catch (rowError) {
          console.error("[Dormitory] contract import row failed", rowError);
          skipped += 1;
        }
      }
      setContractImportResult({ inserted, skipped });
      toast.success(`导入完成：成功 ${inserted} 条，跳过 ${skipped} 条`);
      if (contractImportFileInputRef.current) {
        contractImportFileInputRef.current.value = "";
      }
    } catch (error) {
      console.error("[Dormitory] import contracts failed", error);
      toast.error(error instanceof Error ? error.message : "导入失败，请检查模板");
    } finally {
      setContractImporting(false);
    }
  };

  const ensureSiteByName = async (name: string) => {
    const trimmed = name.trim();
    if (!trimmed) {
      throw new Error("模板中存在未填写地点名称的记录");
    }
    const cached = siteCacheRef.current.get(trimmed);
    if (cached) return cached;
    const created = await createDormSite({ name: trimmed });
    siteCacheRef.current.set(trimmed, created);
    setSites((prev) => [created, ...prev]);
    setSiteOrder((prev) => [created.id, ...prev.filter((id) => id !== created.id)]);
    return created;
  };

  const ensureBuildingByName = async (siteId: number, buildingName: string) => {
    const trimmed = buildingName.trim() || "默认楼栋";
    const key = `${siteId}-${trimmed}`;
    const cached = buildingCacheRef.current.get(key);
    if (cached) return cached;
    const created = await createDormBuilding({ site_id: siteId, name: trimmed });
    buildingCacheRef.current.set(key, created);
    setBuildings((prev) => [created, ...prev]);
    return created;
  };


  const handleRoomRecordValueChange = useCallback(
    (key: string, updates: Partial<RoomChargeRecordEntry>) => {
      const sourceItem = activeRoomChargeItems.find((item) => item.key === key);
      const recordType = determineRecordTypeForCharge(key, sourceItem?.mode);
      if (!recordType) return;
      setRoomChargeRecords((prev) => {
        const base = prev[key] ?? createDefaultRecordEntry(recordType);
        const merged: RoomChargeRecordEntry = { ...base, ...updates, type: recordType };
        const nextEntry = normalizeRoomRecordEntry(merged, sourceItem, roomChargeOverrides[key]);
        if (areRoomRecordEntriesEqual(base, nextEntry)) {
          return prev;
        }
        return { ...prev, [key]: nextEntry };
      });
    },
    [activeRoomChargeItems, roomChargeOverrides],
  );

  useEffect(() => {
    if (!roomDialogOpen) return;
    setRoomChargeRecords((prev) => {
      let changed = false;
      const nextRecords: RoomChargeRecordState = {};
      Object.entries(prev).forEach(([key, entry]) => {
        const sourceItem = activeRoomChargeItems.find((item) => item.key === key);
        if (!sourceItem) {
          nextRecords[key] = entry;
          return;
        }
        const normalized = normalizeRoomRecordEntry(entry, sourceItem, roomChargeOverrides[key]);
        if (!areRoomRecordEntriesEqual(entry, normalized)) {
          changed = true;
        }
        nextRecords[key] = normalized;
      });
      return changed ? nextRecords : prev;
    });
  }, [roomDialogOpen, activeRoomChargeItems, roomChargeOverrides]);

  const handleContractDialogToggle = (open: boolean) => {
    setContractDialogOpen(open);
    if (!open) {
      setContractForm(createBlankContractForm());
      setEmployeeSearchTerm("");
      setEmployeeSuggestions([]);
      setEditingContractId(null);
      setEditingContractOriginalBedId(null);
      setContractSaving(false);
      setContractDialogTitle("办理入住");
      setContractDialogTab("detail");
    }
  };

  const handleOpenContractDialog = () => {
    setEditingContractId(null);
    setEditingContractOriginalBedId(null);
    const blankForm = createBlankContractForm();
    setContractForm(blankForm);
    setEmployeeSearchTerm("");
    setEmployeeSuggestions([]);
    setContractDialogTitle("办理入住");
    setContractDialogOpen(true);
  };

  const handleEditContract = (contract: DormContract) => {
    setEditingContractId(contract.id);
    setEditingContractOriginalBedId(contract.bed_id ?? null);
    const noteMeta = parseContractNoteMeta(contract.notes);
    const nextForm: ContractFormState = {
      employee_id: contract.employee_id ? String(contract.employee_id) : "",
      employee_name: contract.employee_name || "",
      employee_department: contract.employee_department || "",
      employee_position: noteMeta.position || "",
      employee_job_number: noteMeta.jobNumber || "",
      employee_phone: contract.employee_phone || "",
      employee_id_number: contract.employee_id_number || "",
      residence_address: contract.employee_residence || "",
      emergency_contact: noteMeta.emergencyContact || "",
      room_id: String(contract.room_id || ""),
      bed_id: contract.bed_id ? String(contract.bed_id) : "",
      start_date: formatDateInputValue(contract.start_date),
      end_date: formatDateInputValue(contract.end_date),
      rent_amount: contract.rent_amount ? String(contract.rent_amount) : "",
      deposit_amount: contract.deposit_amount ? String(contract.deposit_amount) : "",
      deposit_share_mode: noteMeta.depositShareMode || "personal",
      payment_method: contract.payment_method || "按月",
      deposit_plan_date: noteMeta.depositPlanMonth || "",
      rent_plan_date: noteMeta.rentPlanMonth || "",
      water_start: noteMeta.waterStart || "",
      electric_start: noteMeta.electricStart || "",
      gas_start: noteMeta.gasStart || "",
      pledge_amount: noteMeta.pledgeAmount || "",
      pledge_plan_date: noteMeta.pledgePlanMonth || "",
      pledge_share_mode: noteMeta.pledgeShareMode || "personal",
      notes: noteMeta.additionalNotes || "",
    };
    setContractForm(nextForm);
    setEmployeeSuggestions([]);
    setContractDialogTitle("入住详情");
    setContractDialogOpen(true);
  };

  const handleSelectEmployee = (employee: EmployeeResponse) => {
    setContractForm((prev) => ({
      ...prev,
      employee_id: String(employee.id),
      employee_name: employee.name || prev.employee_name,
      employee_department: employee.department || prev.employee_department,
       employee_position: employee.position || prev.employee_position,
       employee_job_number: employee.employee_id || prev.employee_job_number,
      employee_phone: employee.phone || prev.employee_phone,
      employee_id_number: employee.id_number || prev.employee_id_number,
      residence_address: employee.id_address || prev.residence_address,
       emergency_contact:
        employee.emergency_contact
          ? employee.emergency_phone
            ? `${employee.emergency_contact}（${employee.emergency_phone}）`
            : employee.emergency_contact
          : prev.emergency_contact,
    }));
    setEmployeeSearchTerm("");
    setEmployeeSuggestions([]);
  };

  const buildContractNotes = (form: ReturnType<typeof createBlankContractForm>) => {
    const segments: string[] = [];
    if (form.employee_job_number?.trim()) segments.push(`工号：${form.employee_job_number.trim()}`);
    if (form.employee_position?.trim()) segments.push(`岗位：${form.employee_position.trim()}`);
    if (form.emergency_contact?.trim()) segments.push(`紧急联系人：${form.emergency_contact.trim()}`);
    if (form.water_start) segments.push(`水表起始：${form.water_start}`);
    if (form.electric_start) segments.push(`电表起始：${form.electric_start}`);
    if (form.gas_start) segments.push(`气表起始：${form.gas_start}`);
    if (form.deposit_plan_date) segments.push(`押金月份：${form.deposit_plan_date}`);
    if (form.rent_plan_date) segments.push(`租金月份：${form.rent_plan_date}`);
    if (form.deposit_share_mode) segments.push(`押金承担：${SHARE_MODE_LABELS[form.deposit_share_mode]}`);
    if (form.pledge_plan_date) segments.push(`保证金月份：${form.pledge_plan_date}`);
    if (form.pledge_amount.trim()) segments.push(`保证金金额：${form.pledge_amount.trim()}`);
    if (form.pledge_share_mode) segments.push(`保证金承担：${SHARE_MODE_LABELS[form.pledge_share_mode]}`);
    if (form.notes.trim()) segments.push(form.notes.trim());
    return segments.join(" | ");
  };

  const handleSaveContract = async (options?: { keepOpen?: boolean }) => {
    try {
      if (contractSaving) {
        return;
      }
      const keepDialogOpen = options?.keepOpen === true;
      if (!contractForm.employee_name.trim() || !contractForm.room_id || !contractForm.start_date) {
        toast.error("请填写完整入住信息");
        return;
      }
      const startDate = contractForm.start_date?.trim();
      if (!startDate || Number.isNaN(new Date(startDate).getTime())) {
        toast.error("请选择有效的入住日期");
        return;
      }

      const normalizedEnd = contractForm.end_date?.trim() || "";
      let endDateToSend: string | undefined;
      if (normalizedEnd && !Number.isNaN(new Date(normalizedEnd).getTime())) {
        endDateToSend = normalizedEnd;
      } else {
        endDateToSend = startDate;
      }
      const roomId = Number(contractForm.room_id);
      const targetRoom = roomById.get(roomId);
      const hasBeds = (targetRoom?.beds?.length || 0) > 0;
      if (hasBeds && !contractForm.bed_id) {
        toast.error("请选择床位");
        return;
      }
      const selectedBedId = contractForm.bed_id ? Number(contractForm.bed_id) : null;
      const isOriginalBedSelection = Boolean(
        editingContractId && editingContractOriginalBedId && selectedBedId && selectedBedId === editingContractOriginalBedId,
      );
      if (hasBeds && selectedBedId) {
        const occupancy = roomOccupancyMap.get(roomId) ?? EMPTY_OCCUPANCY;
        if (!isOriginalBedSelection && occupancy.bedAssignments[selectedBedId]) {
          toast.error("该床位已被占用，请重新选择");
          return;
        }
      }
      const payload = {
        employee_id: contractForm.employee_id ? Number(contractForm.employee_id) : undefined,
        employee_name: contractForm.employee_name.trim(),
        employee_department: contractForm.employee_department,
        employee_phone: contractForm.employee_phone,
        employee_id_number: contractForm.employee_id_number?.trim(),
        employee_residence: contractForm.residence_address?.trim(),
        room_id: roomId,
        bed_id: selectedBedId ?? undefined,
        start_date: startDate,
        end_date: endDateToSend,
        rent_amount: Number(contractForm.rent_amount || 0),
        deposit_amount: Number(contractForm.deposit_amount || 0),
        payment_method: contractForm.payment_method,
        status: "active",
        notes: buildContractNotes(contractForm),
      };
      setContractSaving(true);
      console.debug("[Dormitory] contract payload", payload);
      if (editingContractId) {
        const updated = await updateDormContract(editingContractId, payload);
        const enriched = enrichContractRelations(updated);
        setContracts((prev) => prev.map((item) => (item.id === enriched.id ? enriched : item)));
        toast.success("入住信息已更新");
        if (keepDialogOpen) {
          await refreshRoomsAndContracts();
          handleEditContract(enriched);
          return;
        }
      } else {
        const created = await createDormContract(payload);
        const enriched = enrichContractRelations(created);
        setContracts((prev) => [enriched, ...prev]);
        toast.success("已安排入住");
      }
      await refreshRoomsAndContracts();
      handleContractDialogToggle(false);
    } catch (error) {
      console.error("[Dormitory] create contract failed", error);
      toast.error(error instanceof Error ? error.message : "保存入住信息失败");
    } finally {
      setContractSaving(false);
    }
  };

  const handleSiteDragStart = (siteId: number) => (event: React.DragEvent<HTMLDivElement>) => {
    setDraggingSiteId(siteId);
    event.dataTransfer.effectAllowed = "move";
    event.dataTransfer.setData("text/plain", String(siteId));
  };

  const handleSiteDragOver = (targetId: number) => (event: React.DragEvent<HTMLDivElement>) => {
    event.preventDefault();
    if (draggingSiteId === null || draggingSiteId === targetId) return;
    setSiteOrder((prev) => {
      const from = prev.indexOf(draggingSiteId);
      const to = prev.indexOf(targetId);
      if (from === -1 || to === -1) {
        return prev;
      }
      const next = [...prev];
      next.splice(from, 1);
      next.splice(to, 0, draggingSiteId);
      return next;
    });
  };

  const handleSiteDragEnd = () => {
    setDraggingSiteId(null);
  };

  const handleSiteCardClick = (siteId: number) => {
    if (siteCardClickTimerRef.current) {
      clearTimeout(siteCardClickTimerRef.current);
      siteCardClickTimerRef.current = null;
    }
    siteCardClickTimerRef.current = setTimeout(() => {
      setRoomSiteFilter((prev) => (prev === siteId ? "all" : siteId));
      siteCardClickTimerRef.current = null;
    }, 200);
  };

  const handleSiteCardDoubleClick = (site: DormSite) => {
    if (siteCardClickTimerRef.current) {
      clearTimeout(siteCardClickTimerRef.current);
      siteCardClickTimerRef.current = null;
    }
    handleOpenSiteDialog(site);
  };

  const focusSiteCard = useCallback(
    (siteId: number) => {
      setActiveTab("overview");
      setRoomSiteFilter(siteId);
      window.setTimeout(() => {
        const card = siteCardRefs.current.get(siteId);
        card?.scrollIntoView({ behavior: "smooth", block: "center" });
      }, 180);
    },
    [setActiveTab, setRoomSiteFilter],
  );

  const handleWechatClick = useCallback(() => {
    if (!primaryWechatConfig?.value) {
      toast.info("管理员尚未配置客服微信");
      return;
    }
    if (primaryWechatConfig.value.startsWith("weixin://")) {
      window.location.href = primaryWechatConfig.value;
      return;
    }
    setWechatDialogOpen(true);
  }, [primaryWechatConfig]);

  useEffect(() => {
    const handleSupport = () => handleWechatClick();
    window.addEventListener("dock:open-support", handleSupport as EventListener);
    const handleOpenMemo = (event: Event) => {
      const siteId = (event as CustomEvent<{ siteId?: number }>).detail?.siteId;
      if (typeof siteId !== "number") return;
      const targetSite = siteById.get(siteId);
      if (!targetSite) return;
      focusSiteCard(siteId);
      handleOpenSiteDialog(targetSite, "memo");
    };
    window.addEventListener("dock:open-site-memo", handleOpenMemo);
    return () => {
      window.removeEventListener("dock:open-support", handleSupport as EventListener);
      window.removeEventListener("dock:open-site-memo", handleOpenMemo);
    };
  }, [focusSiteCard, handleOpenSiteDialog, handleWechatClick, siteById]);

  const renderSiteCards = () => {
    if (orderedSiteStats.length === 0) {
      return (
        <Card className="border-dashed">
          <CardContent className="flex h-32 flex-col items-center justify-center text-sm text-muted-foreground">
            暂无宿舍地点，请先点击“新增地点”完成录入。
          </CardContent>
        </Card>
      );
    }
    return (
      <div className={SITE_CARD_GRID_CLASS}>
        {orderedSiteStats.map((site) => {
          const houseExtra = siteHouseExtras[memoKey(site.id)];
          const propertyCompany = site.property_company || houseExtra?.propertyCompany || "";
          const propertyContact = site.property_contact || houseExtra?.propertyContact || "";
          return (
            <Card
              key={site.id}
              ref={(node) => {
                if (node) {
                  siteCardRefs.current.set(site.id, node);
                } else {
                  siteCardRefs.current.delete(site.id);
                }
              }}
              draggable
              onClick={() => handleSiteCardClick(site.id)}
              onDragStart={handleSiteDragStart(site.id)}
              onDragOver={handleSiteDragOver(site.id)}
              onDragEnd={handleSiteDragEnd}
              onDrop={(event) => {
                event.preventDefault();
                handleSiteDragEnd();
              }}
              onDoubleClick={(event) => {
                event.preventDefault();
                event.stopPropagation();
                handleSiteCardDoubleClick(site);
              }}
              className={`rounded-xl border ${SITE_CARD_BG_CLASS} shadow-sm transition hover:-translate-y-1 hover:shadow-md ${draggingSiteId === site.id ? "ring-2 ring-primary/40" : ""} ${selectedSiteCardId === site.id ? "ring-2 ring-primary" : ""}`}
            >
              <CardContent className="space-y-3 p-4">
                <div className="flex items-start justify-between gap-2">
                  <div className="space-y-1">
                    <p className="text-base font-semibold text-foreground">{site.name}</p>
                    <p className="text-xs text-muted-foreground line-clamp-2">{site.address || "未填写地址"}</p>
                    {propertyCompany && (
                      <p className="text-[11px] text-muted-foreground">
                        物业：{propertyCompany}（{propertyContact || "未填写"}）
                      </p>
                    )}
                  </div>
                  <DropdownMenu>
                    <DropdownMenuTrigger asChild>
                      <Button variant="ghost" size="icon" className="h-8 w-8" aria-label="地点操作">
                        <MoreHorizontal className="h-4 w-4" />
                      </Button>
                    </DropdownMenuTrigger>
                    <DropdownMenuContent align="end">
                      <DropdownMenuItem onClick={() => handleOpenSiteDialog(site)} className="gap-2">
                        <Eye className="h-4 w-4" />
                        查看详情
                      </DropdownMenuItem>
                      {/* P7.1：删除宿舍地点需 dormitory.delete 权限 */}
                      <RequirePermission resource="dormitory" action="delete">
                        <DropdownMenuItem className="gap-2 text-destructive" onClick={() => setSiteDeleteTarget(site)}>
                          <Trash2 className="h-4 w-4" />
                          删除
                        </DropdownMenuItem>
                      </RequirePermission>
                    </DropdownMenuContent>
                  </DropdownMenu>
                </div>
                <div className="grid grid-cols-3 gap-2 text-center text-xs">
                  <div className="rounded-xl border border-muted-foreground/10 bg-background/40 px-2 py-1.5">
                    <p className="text-muted-foreground">房间</p>
                    <p className="text-lg font-semibold text-foreground">{site.totalRooms}</p>
                    <p className="text-muted-foreground">空闲 {site.freeRooms}</p>
                  </div>
                  <div className="rounded-xl border border-muted-foreground/10 bg-background/40 px-2 py-1.5">
                    <p className="text-muted-foreground">床位</p>
                    <p className="text-lg font-semibold text-foreground">{site.totalBeds}</p>
                    <p className="text-muted-foreground">空闲 {site.freeBeds}</p>
                  </div>
                  <div className="rounded-xl border border-muted-foreground/10 bg-background/40 px-2 py-1.5">
                    <p className="text-muted-foreground">入住</p>
                    <p className="text-lg font-semibold text-foreground">{site.tenants}</p>
                    <p className="text-muted-foreground">占用房 {Math.max(site.totalRooms - site.freeRooms, 0)}</p>
                  </div>
                </div>

              </CardContent>
            </Card>
          );
        })}
      </div>
    );
  };

  const renderRoomTable = () => {
    const roomAllSelected = sortedRooms.length > 0 && sortedRooms.every((room) => selectedRoomIds.includes(room.id));
    const selectionCount = selectedRoomIds.length;
    const displayedBedCount = sortedRooms.reduce((sum, room) => sum + (room.bed_count || room.beds?.length || 0), 0);

    const handleRoomSelectAll = (checked: boolean) => {
      setSelectedRoomIds(checked ? sortedRooms.map((room) => room.id) : []);
    };

    const toggleRoomSelection = (roomId: number) => {
      setSelectedRoomIds((prev) => (prev.includes(roomId) ? prev.filter((id) => id !== roomId) : [...prev, roomId]));
    };

    const handleRoomExportSelected = () => {
      const targetRooms = sortedRooms.filter((room) => selectedRoomIds.includes(room.id));
      if (targetRooms.length === 0) {
        toast.error("请先选择需要导出的房间");
        return;
      }
      exportRoomsToWorkbook(targetRooms, "房间列表-选中");
    };

  const handleRoomPrintSelected = () => {
    const targetRooms = sortedRooms.filter((room) => selectedRoomIds.includes(room.id));
    if (targetRooms.length === 0) {
      toast.error("请先勾选需要打印的房间");
      return;
      }
      setRoomPrintContext(targetRooms);
      setRoomPrintSuggestedTitle(`房间列表打印（共 ${targetRooms.length} 条）`);
    setRoomPrintDialogOpen(true);
  };

    const renderToolbarActions = () => {
      if (selectionCount > 0) {
        return (
          <div className="flex gap-2">
            {/* P7.1：房间导出需 dormitory.view 权限 */}
            <RequirePermission resource="dormitory" action="view">
              <Button variant="outline" size="sm" className="gap-1" onClick={handleRoomExportSelected}>
                <Download className="h-4 w-4" />
                导出
              </Button>
            </RequirePermission>
            {/* P7.1：房间打印为查看类操作，需 dormitory.view 权限 */}
            <RequirePermission resource="dormitory" action="view">
              <Button variant="outline" size="sm" className="gap-1" onClick={handleRoomPrintSelected}>
                <Printer className="h-4 w-4" />
                打印
              </Button>
            </RequirePermission>
          </div>
        );
      }
      return (
        <div className="flex flex-wrap gap-2">
          {/* P7.1：新增宿舍地点需 dormitory.create 权限 */}
          <RequirePermission resource="dormitory" action="create">
            <Button variant="outline" size="sm" className="gap-1" onClick={() => handleOpenSiteDialog()}>
              <Plus className="h-4 w-4" />
              新增地点
            </Button>
          </RequirePermission>
          {/* P7.1：新增房间需 dormitory.create 权限 */}
          <RequirePermission resource="dormitory" action="create">
            <Button size="sm" className="gap-1" onClick={handleOpenRoomDialog}>
              <Plus className="h-4 w-4" />
              新增房间
            </Button>
          </RequirePermission>
          {/* P7.1：导入房间需 dormitory.create 权限 */}
          <RequirePermission resource="dormitory" action="create">
            <Button variant="outline" size="sm" className="gap-1" onClick={() => handleRoomImportDialogToggle(true)}>
              <Upload className="h-4 w-4" />
              导入
            </Button>
          </RequirePermission>
        </div>
      );
    };

    return (
      <Card className="shadow-sm border overflow-hidden">
        <CardHeader className="space-y-4">
          <div className="flex flex-wrap items-start justify-between gap-4">
            <div className="space-y-1">
              <CardTitle className="flex items-center gap-2 text-base">
                <Building2 className="h-4 w-4" />
                房间列表
              </CardTitle>
              <p className="text-xs text-muted-foreground">房间 {sortedRooms.length} 间 · 床位 {displayedBedCount} 个</p>
            </div>
            {renderToolbarActions()}
          </div>
        </CardHeader>
        <CardContent>
          <div className="mb-4 flex flex-wrap items-center gap-3">
            <div className="relative min-w-[220px] flex-1">
              <Search className="pointer-events-none absolute left-3 top-3 h-4 w-4 text-muted-foreground" />
              <Input
                placeholder="搜索房号、楼栋、地点或备注..."
                value={roomSearch}
                onChange={(event) => setRoomSearch(event.target.value)}
                className="pl-9"
              />
              {roomSearch && (
                <button
                  type="button"
                  onClick={() => setRoomSearch("")}
                  className="absolute right-3 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground"
                  aria-label="清除搜索"
                >
                  <X className="h-4 w-4" />
                </button>
              )}
            </div>
            <Select value={roomStatusFilter} onValueChange={(value) => setRoomStatusFilter(value)}>
              <SelectTrigger className="w-36">
                <SelectValue placeholder="房间状态" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">全部状态</SelectItem>
                {ROOM_STATUS_OPTIONS.map((status) => (
                  <SelectItem key={status} value={status}>
                    {status}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
            <Select value={roomTypeFilter} onValueChange={(value) => setRoomTypeFilter(value as "all" | "company" | "personal")}>
              <SelectTrigger className="w-36">
                <SelectValue placeholder="宿舍类型" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">全部宿舍</SelectItem>
                <SelectItem value="company">公司宿舍</SelectItem>
                <SelectItem value="personal">个人宿舍</SelectItem>
              </SelectContent>
            </Select>
            <Dialog open={showRoomFieldSelector} onOpenChange={setShowRoomFieldSelector}>
              <DialogTrigger asChild>
                <Button variant="outline" size="sm" className="gap-1">
                  <Settings className="h-4 w-4" /> 显示字段
                </Button>
              </DialogTrigger>
              <DialogContent className={DIALOG_SIZES.sm}>
                <DialogHeader>
                  <DialogTitle className="flex items-center gap-2">
                    <Settings className="h-4 w-4" /> 自定义显示字段
                  </DialogTitle>
                  <DialogDescription>勾选需要在房间列表中展示的字段。</DialogDescription>
                </DialogHeader>
                <div className="max-h-80 overflow-y-auto space-y-2">
                  {effectiveRoomColumnConfig.map((column) => (
                    <label key={column.id} className="flex items-center gap-2 text-sm font-medium text-foreground">
                      <input
                        type="checkbox"
                        className="h-4 w-4 rounded border-muted-foreground"
                        checked={roomColumnVisibility[column.id] !== false}
                        onChange={() => handleRoomFieldToggle(column.id)}
                      />
                      <span>{column.label}</span>
                    </label>
                  ))}
                </div>
                <DialogFooter className="flex flex-col gap-2 sm:flex-row sm:justify-end">
                  <Button variant="outline" onClick={resetRoomFieldVisibility}>
                    恢复默认
                  </Button>
                  <Button onClick={() => setShowRoomFieldSelector(false)}>完成</Button>
                </DialogFooter>
              </DialogContent>
            </Dialog>
          </div>
          <DataTableWrapper height="h-[65vh]">
              <Table className="min-w-full table-auto text-sm">
                <TableHeader>
                  <TableRow className="text-muted-foreground">
                    <TableHead className={cn("w-12", ALIGNMENT_CLASS.center)}>
                      <input
                        type="checkbox"
                        className="h-4 w-4 rounded border-muted-foreground"
                        checked={roomAllSelected}
                        onChange={(event) => handleRoomSelectAll(event.target.checked)}
                        aria-label="选择全部房间"
                      />
                    </TableHead>
                    {visibleRoomColumns.map((column) => (
                      <TableHead
                        key={column.id}
                        draggable
                        onDragStart={(event) => handleRoomColumnDragStart(event, column.id)}
                        onDragOver={handleRoomColumnDragOver}
                        onDrop={(event) => handleRoomColumnDrop(event, column.id)}
                        onDragEnd={handleRoomColumnDragEnd}
                        onClick={column.sortable === false ? undefined : () => handleRoomSortClick(column.id)}
                        className={cn("select-none whitespace-normal break-words", ALIGNMENT_CLASS.left)}
                      >
                        <span className="flex items-center gap-1">
                          {column.label}
                          {column.sortable !== false && roomSort.columnId === column.id && (
                            <span>{roomSort.direction === "asc" ? "↑" : "↓"}</span>
                          )}
                        </span>
                      </TableHead>
                    ))}
                    <TableHead className={cn("w-14", ALIGNMENT_CLASS.center)}>操作</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {sortedRooms.length === 0 ? (
                    <TableRow>
                      <TableCell colSpan={visibleRoomColumns.length + 2} className="py-6 text-center text-muted-foreground">
                        暂无符合条件的房间
                      </TableCell>
                    </TableRow>
                  ) : (
                    sortedRooms.map((room) => {
                      const context = getRoomContext(room);
                      const checked = selectedRoomIds.includes(room.id);
                      return (
                        <TableRow key={room.id} className="text-sm hover:bg-muted/50" onDoubleClick={() => handleEditRoom(room)}>
                          <TableCell className={ALIGNMENT_CLASS.center}>
                            <input
                              type="checkbox"
                              className="h-4 w-4 rounded border-muted-foreground"
                              checked={checked}
                              onChange={() => toggleRoomSelection(room.id)}
                              aria-label={`选择房间 ${room.room_number}`}
                            />
                          </TableCell>
                          {visibleRoomColumns.map((column) => (
                            <TableCell key={column.id} className={cn("align-top whitespace-normal break-words", ALIGNMENT_CLASS.left)}>
                              {column.render(room, context)}
                            </TableCell>
                          ))}
                          <TableCell className={cn("w-14", ALIGNMENT_CLASS.center)}>
                            <DropdownMenu>
                              <DropdownMenuTrigger asChild>
                                <Button variant="ghost" size="icon" className="h-8 w-8" aria-label="房间操作">
                                  <MoreHorizontal className="h-4 w-4" />
                                </Button>
                              </DropdownMenuTrigger>
                              <DropdownMenuContent align="end">
                                <DropdownMenuItem onClick={() => handleEditRoom(room)} className="gap-2">
                                  <Eye className="h-4 w-4" />
                                  查看详情
                                </DropdownMenuItem>
                                {/* P7.1：删除房间需 dormitory.delete 权限 */}
                                <RequirePermission resource="dormitory" action="delete">
                                  <DropdownMenuItem className="gap-2 text-destructive" onClick={() => setRoomDeleteTarget(room)}>
                                    <Trash2 className="h-4 w-4" />
                                    删除房间
                                  </DropdownMenuItem>
                                </RequirePermission>
                              </DropdownMenuContent>
                            </DropdownMenu>
                          </TableCell>
                        </TableRow>
                      );
                    })
                  )}
                </TableBody>
              </Table>
              <ScrollBar orientation="horizontal" />
            </DataTableWrapper>

        </CardContent>
      </Card>
    );
  };

const renderRoomDetailSections = () => (
  <>
      <section className="space-y-4">
        <h4 className="text-sm font-semibold text-foreground">基础信息</h4>
        <div className={RESPONSIVE_FIELD_GRID_CLASS}>
          {roomForm.cost_bearing_mode === "company" && (
            <div className="sm:col-span-2 lg:col-span-1 xl:max-w-md">
              <Label className="text-xs font-medium text-muted-foreground">公司名称</Label>
              <Input
                className="mt-1"
                placeholder="例如：总部行政部"
                value={roomForm.company_name}
                onChange={(event) => setRoomForm((prev) => ({ ...prev, company_name: event.target.value }))}
              />
            </div>
          )}
          <div>
            <Label className="text-xs font-medium text-muted-foreground">宿舍地点</Label>
            <select
              className="mt-1 w-full rounded-md border border-input bg-background px-3 py-2 text-sm"
              value={roomForm.site_id}
              onChange={(event) => {
                const nextSite = event.target.value;
                setRoomForm((prev) => ({ ...prev, site_id: nextSite, building_id: "" }));
              }}
            >
              <option value="">请选择地点</option>
              {sites.map((site) => (
                <option key={site.id} value={site.id}>
                  {site.name}
                </option>
              ))}
            </select>
          </div>
          <div>
            <Label className="text-xs font-medium text-muted-foreground">楼栋</Label>
            <select
              className="mt-1 w-full rounded-md border border-input bg-background px-3 py-2 text-sm"
              value={roomForm.building_id}
              onChange={(event) => setRoomForm((prev) => ({ ...prev, building_id: event.target.value }))}
              disabled={!roomForm.site_id}
            >
              <option value="">请选择楼栋</option>
              {(roomForm.site_id ? siteBuildingMap[Number(roomForm.site_id)] || [] : buildings).map((building) => (
                <option key={building.id} value={building.id}>
                  {building.name}
                </option>
              ))}
            </select>
          </div>
          <div>
            <Label className="text-xs font-medium text-muted-foreground">房号</Label>
            <Input className="mt-1" value={roomForm.room_number} onChange={(event) => setRoomForm((prev) => ({ ...prev, room_number: event.target.value }))} />
          </div>
          <div>
            <Label className="text-xs font-medium text-muted-foreground">户型</Label>
            <select
              className="mt-1 w-full rounded-md border border-input bg-background px-3 py-2 text-sm"
              value={roomForm.house_layout}
              onChange={(event) => setRoomForm((prev) => ({ ...prev, house_layout: event.target.value }))}
            >
              <option value="一室一厅">一室一厅</option>
              <option value="二室一厅">二室一厅</option>
              <option value="三室一厅">三室一厅</option>
            </select>
          </div>
          <div>
            <Label className="text-xs font-medium text-muted-foreground">房间类型</Label>
            <select
              className="mt-1 w-full rounded-md border border-input bg-background px-3 py-2 text-sm"
              value={roomForm.room_category}
              onChange={(event) => handleRoomCategoryChange(event.target.value)}
            >
              <option value="单间">单间</option>
              <option value="多人间">多人间</option>
            </select>
          </div>
          <div>
            <Label className="text-xs font-medium text-muted-foreground">建筑面积 (m²)</Label>
            <Input className="mt-1" value={roomForm.area_square} onChange={(event) => setRoomForm((prev) => ({ ...prev, area_square: event.target.value }))} />
          </div>
          <div className="space-y-2">
            <div>
              <Label className="text-xs font-medium text-muted-foreground">床位数量 (个)</Label>
              <Input
                className="mt-1"
                type="number"
                min={roomForm.room_category === "单间" ? 1 : 2}
                value={roomForm.bed_count}
                onChange={(event) => handleRoomBedCountChange(event.target.value)}
              />
            </div>
            {bedPreview.length > 0 && (
              <p className="text-xs text-muted-foreground">将自动生成：{bedPreview.join("、")}</p>
            )}
          </div>
        </div>
      </section>

      <section className="space-y-3">
        <div className="flex items-center justify-between">
          <h4 className="text-sm font-semibold text-foreground">扣费标准</h4>
        </div>
        {roomForm.site_id ? (
          <div className={RESPONSIVE_FIELD_GRID_CLASS}>
            {activeRoomChargeItems.map((item) => {
              const overrideValue = roomChargeOverrides[item.key] ?? "";
              const placeholder = item.unitPrice ? `默认 ${item.unitPrice}${item.unitLabel || "元"}` : "未设置";
              return (
                <div key={item.key}>
                  <Label className="flex items-center justify-between text-xs font-medium text-muted-foreground">
                    <span>
                      {item.label}
                      <span className="text-[11px] text-muted-foreground">（{item.unitLabel || "元"}）</span>
                    </span>
                  </Label>
                  <Input
                    inputMode="decimal"
                    className="mt-1"
                    placeholder={placeholder}
                    value={overrideValue}
                    onChange={(event) => {
                      const value = event.target.value;
                      if (value && !DECIMAL_INPUT_PATTERN.test(value)) {
                        return;
                      }
                      setRoomChargeOverrides((prev) => {
                        const next = { ...prev };
                        if (!value.trim()) {
                          delete next[item.key];
                        } else {
                          next[item.key] = value;
                        }
                        return next;
                      });
                    }}
                  />
                </div>
              );
            })}
          </div>
        ) : (
          <Card className="border-dashed bg-muted/40">
            <CardContent className="py-4 text-center text-xs text-muted-foreground">请选择地点后配置扣费项</CardContent>
          </Card>
        )}
      </section>

      <section className="space-y-3">
        <div className="flex items-center justify-between">
          <h4 className="text-sm font-semibold text-foreground">物品备注</h4>
        </div>
        <Textarea
          rows={3}
          placeholder="例如：单人床2张、衣柜2个、空调1台"
          value={roomForm.inventory_note}
          onChange={(event) => setRoomForm((prev) => ({ ...prev, inventory_note: event.target.value }))}
        />
      </section>
    </>
);

const renderRoomRecordSections = (
  recordableItems: ChargeSetting[],
  roomChargeRecords: RoomChargeRecordState,
  handleRecordChange: (key: string, updates: Partial<RoomChargeRecordEntry>) => void,
  paymentCycles: PaymentCycle[],
  costMode: ShareMode,
  companyName?: string,
) => {
  const cycleOptions = paymentCycles.map((cycle) => ({
    key: cycle,
    label: PAYMENT_CYCLE_LABELS[cycle] ?? cycle,
  }));
  const visibleItems = recordableItems
    .filter((item) => item.enabled !== false && shouldRenderRoomRecord(item.key, costMode))
    .sort((a, b) => getRoomRecordOrder(a.key) - getRoomRecordOrder(b.key));
  if (visibleItems.length === 0) {
    return (
      <Card className="bg-muted/30">
        <CardContent className="py-6 text-center text-sm text-muted-foreground">当前扣费标准未启用可记录的项目。</CardContent>
      </Card>
    );
  }
  return (
    <div className="space-y-4">
      {visibleItems.map((item) => {
        const chargeKey = item.key;
        const recordType = determineRecordTypeForCharge(chargeKey, item.mode);
        if (!recordType) return null;
        const entry = roomChargeRecords[chargeKey] ?? createDefaultRecordEntry(recordType);
        if (recordType === "meter") {
          return null;
        }
        if (recordType === "rent") {
          const isCompanyRent = costMode === "company" && (chargeKey === "rent" || chargeKey === "property");
          return (
            <section key={chargeKey} className="space-y-3">
              <div className="flex items-center justify-between">
                <h4 className="text-sm font-semibold text-foreground">{item.label}记录</h4>
                {isCompanyRent && <span className="text-xs text-muted-foreground">公司承担，不分摊至入住人员</span>}
              </div>
              <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
                <div>
                  <Label className="text-xs font-medium text-muted-foreground">最近收取日期</Label>
                  <Input
                    className="mt-1"
                    type="date"
                    value={entry.paymentDate || ""}
                    onChange={(event) => handleRecordChange(chargeKey, { paymentDate: event.target.value })}
                  />
                </div>
                <div>
                  <Label className="text-xs font-medium text-muted-foreground">下一次收取日期</Label>
                  <Input className="mt-1" type="date" value={entry.nextPaymentDate || ""} readOnly placeholder="根据周期自动计算" />
                </div>
                <div>
                  <Label className="text-xs font-medium text-muted-foreground">付款周期</Label>
                  <select
                    className="mt-1 w-full rounded-md border border-input bg-background px-3 py-2 text-sm"
                    value={entry.paymentCycle || ""}
                    onChange={(event) => handleRecordChange(chargeKey, { paymentCycle: event.target.value })}
                  >
                    <option value="">请选择</option>
                    {cycleOptions.map((option) => (
                      <option key={option.key} value={option.key}>
                        {option.label}
                      </option>
                    ))}
                  </select>
                </div>
                <div>
                  <Label className="text-xs font-medium text-muted-foreground">付款金额（元）</Label>
                  <Input className="mt-1" readOnly value={entry.paymentAmount || ""} placeholder="根据单价与周期自动计算" />
                </div>
              </div>
              <div className="flex items-center gap-2 pt-1">
                <Checkbox checked={entry.addMemo ?? false} onCheckedChange={(checked) => handleRecordChange(chargeKey, { addMemo: checked === true })} />
                <span className="text-xs text-muted-foreground">自动添加提醒（提前 30/20/10 天）</span>
              </div>
            </section>
          );
        }
        if (recordType === "deposit") {
          const isGuarantee = chargeKey === "deposit";
          const treatAsCompany = costMode === "company" && isGuarantee;
          const participants = entry.participants ?? [];
          const occupancyForRoom = editingRoomId ? roomOccupancyMap.get(editingRoomId) : undefined;
          const occupantCandidates = occupancyForRoom?.members ?? [];
          const depositDefaultAmount = (() => {
            const overrideValue = roomChargeOverrides[chargeKey];
            if (overrideValue && overrideValue.trim()) return overrideValue.trim();
            if (typeof item.unitPrice === "number" && Number.isFinite(item.unitPrice)) {
              return String(item.unitPrice);
            }
            return "";
          })();
          const applyDepositDefaults = (defaults?: Partial<RoomChargeParticipant>) => {
            if (!depositDefaultAmount) return defaults;
            if (defaults?.amount && defaults.amount.trim()) return defaults;
            return { ...defaults, amount: depositDefaultAmount };
          };
          const resolveDepositRefundInfo = (participant: RoomChargeParticipant) => {
            if (participant.contractId && contractMetaById.has(participant.contractId)) {
              const meta = contractMetaById.get(participant.contractId)!;
              return { refunded: (meta.status || "active") === "completed", date: meta.endDate || "", derived: true };
            }
            return {
              refunded: participant.refunded ?? false,
              date: participant.refundDate || "",
              derived: false,
            };
          };
          const updateParticipant = (participantId: string, updates: Partial<RoomChargeParticipant>) => {
            const nextList = participants.map((participant) =>
              participant.id === participantId ? { ...participant, ...updates } : participant,
            );
            handleRecordChange(chargeKey, { participants: nextList });
          };
          const removeParticipant = (participantId: string) => {
            const nextList = participants.filter((participant) => participant.id !== participantId);
            handleRecordChange(chargeKey, { participants: nextList });
          };
          const syncParticipants = () => {
            if (!occupantCandidates.length) return;
            const existingContractIds = participants.map((participant) => participant.contractId).filter(Boolean);
            const additions = occupantCandidates.filter(
              (member) => member.contractId && !existingContractIds.includes(member.contractId),
            );
            if (additions.length === 0) return;
            const merged = [
              ...participants,
              ...additions.map((member) =>
                createRoomChargeParticipant(applyDepositDefaults({ name: member.name, contractId: member.contractId })),
              ),
            ];
            handleRecordChange(chargeKey, { participants: merged });
          };
          if (treatAsCompany) {
            const payerLabel = companyName?.trim() || "缴纳单位";
            const derivedCompanyRefund = (() => {
              if (editingRoomId) {
                const roomContractStatus = roomContractStatusMap.get(editingRoomId);
                if (roomContractStatus) {
                  if (roomContractStatus.hasActive) {
                    return { refunded: false, date: "", derived: true };
                  }
                  return {
                    refunded: true,
                    date: roomContractStatus.lastCheckoutDate || entry.refundDate || "",
                    derived: true,
                  };
                }
              }
              return { refunded: entry.refunded === true, date: entry.refundDate || "", derived: false };
            })();
            return (
              <section key={chargeKey} className="space-y-3">
                <div className="flex items-center justify-between">
                  <h4 className="text-sm font-semibold text-foreground">{item.label}记录</h4>
                  <p className="text-xs text-muted-foreground">公司宿舍由单位承担，状态自动根据入住记录判断</p>
                </div>
                <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
                  <div>
                    <Label className="text-xs font-medium text-muted-foreground">缴纳单位</Label>
                    <Input
                      className="mt-1"
                      placeholder="例如：总部行政部"
                      value={entry.payerName ?? payerLabel}
                      onChange={(event) => handleRecordChange(chargeKey, { payerName: event.target.value })}
                    />
                  </div>
                  <div>
                    <Label className="text-xs font-medium text-muted-foreground">缴纳日期</Label>
                    <Input
                      className="mt-1"
                      type="date"
                      value={entry.collectDate || ""}
                      onChange={(event) => handleRecordChange(chargeKey, { collectDate: event.target.value })}
                    />
                  </div>
                  <div>
                    <Label className="text-xs font-medium text-muted-foreground">金额（元）</Label>
                    <Input
                      className="mt-1"
                      type="number"
                      inputMode="decimal"
                      placeholder={depositDefaultAmount ? `默认 ${depositDefaultAmount}` : "请输入金额"}
                      value={entry.amount || ""}
                      onChange={(event) => handleRecordChange(chargeKey, { amount: event.target.value })}
                    />
                  </div>
                  <div className="sm:col-span-2">
                    <Label className="text-xs font-medium text-muted-foreground">退还状态</Label>
                    <div className="mt-1 flex flex-wrap items-center gap-2 rounded-md border bg-muted/30 px-3 py-2 text-xs">
                      <span className={cn("h-2 w-2 rounded-full", derivedCompanyRefund.refunded ? "bg-emerald-500" : "bg-red-500")} />
                      <span className="font-medium text-foreground">{derivedCompanyRefund.refunded ? "已退还" : "未退还"}</span>
                      {derivedCompanyRefund.date && (
                        <span className="text-muted-foreground">退还日期：{formatDateLabel(derivedCompanyRefund.date)}</span>
                      )}
                      <span className="text-[11px] text-muted-foreground">
                        {derivedCompanyRefund.derived ? "来源：入住记录" : "来源：手动记录"}
                      </span>
                    </div>
                  </div>
                  <div>
                    <Label className="text-xs font-medium text-muted-foreground">退还日期（可补充）</Label>
                    <Input
                      className="mt-1"
                      type="date"
                      value={entry.refundDate || ""}
                      onChange={(event) => handleRecordChange(chargeKey, { refundDate: event.target.value })}
                      placeholder="自动判断，可手动填写"
                    />
                  </div>
                </div>
              </section>
            );
          }
          return (
            <section key={chargeKey} className="space-y-3">
              <div className="flex flex-wrap items-center justify-between gap-2">
                <h4 className="text-sm font-semibold text-foreground">{item.label}记录</h4>
                {occupantCandidates.length > 0 && (
                  <Button
                    size="sm"
                    className="bg-foreground text-background hover:bg-foreground/90"
                    onClick={syncParticipants}
                  >
                    同步入住人员
                  </Button>
                )}
              </div>
              {participants.length === 0 ? (
                <div className="rounded-lg border border-dashed bg-muted/40 px-4 py-6 text-center text-xs text-muted-foreground">
                  尚未记录{item.label}，请先点击“同步入住人员”导入在住成员后再填写。
                </div>
              ) : (
                <div className="space-y-3">
                  {participants.map((participant, index) => {
                    const participantId = participant.id || `${chargeKey}-${index}`;
                    const refundInfo = resolveDepositRefundInfo(participant);
                    const displayValue = (field: "amount" | "collectDate" | "refundDate") => participant[field] ?? "";
                    return (
                      <div key={participantId} className="flex flex-wrap items-end gap-3 rounded-lg border px-4 py-3">
                        <div className="flex min-w-[140px] flex-1 flex-col">
                          <Label className="text-xs font-medium text-muted-foreground">人员姓名</Label>
                          <Input
                            className="mt-1"
                            value={participant.name}
                            placeholder="请输入姓名"
                            onChange={(event) => updateParticipant(participantId, { name: event.target.value })}
                          />
                        </div>
                        <div className="flex min-w-[120px] flex-1 flex-col">
                          <Label className="text-xs font-medium text-muted-foreground">金额（元）</Label>
                          <Input
                            className="mt-1"
                            type="number"
                            inputMode="decimal"
                            value={displayValue("amount")}
                            onChange={(event) => updateParticipant(participantId, { amount: event.target.value })}
                            placeholder="请输入金额"
                          />
                        </div>
                        <div className="flex min-w-[150px] flex-1 flex-col">
                          <Label className="text-xs font-medium text-muted-foreground">{item.label}收取日期</Label>
                          <Input
                            className="mt-1"
                            type="date"
                            value={displayValue("collectDate")}
                            onChange={(event) => updateParticipant(participantId, { collectDate: event.target.value })}
                          />
                        </div>
                        <div className="flex min-w-[220px] flex-1 flex-col">
                          <Label className="text-xs font-medium text-muted-foreground">押金状态</Label>
                          <div className="mt-1 flex flex-wrap items-center gap-2 rounded-md border bg-muted/30 px-3 py-2 text-xs">
                            <span className={cn("h-2 w-2 rounded-full", refundInfo.refunded ? "bg-emerald-500" : "bg-red-500")} />
                            <span className="font-medium text-foreground">{refundInfo.refunded ? "已退还" : "未退还"}</span>
                            {refundInfo.refunded && refundInfo.date && (
                              <span className="text-muted-foreground">退还日期：{formatDateLabel(refundInfo.date)}</span>
                            )}
                            <span className="text-[11px] text-muted-foreground">
                              {refundInfo.derived ? "来源：入住状态" : "来源：手动记录"}
                            </span>
                          </div>
                        </div>
                        <div className="flex items-center">
                          <Button variant="ghost" size="sm" className="text-destructive" onClick={() => removeParticipant(participantId)}>
                            删除
                          </Button>
                        </div>
                      </div>
                    );
                  })}
                </div>
              )}
            </section>
          );
        }
        if (recordType === "bonus" || recordType === "penalty") {
          const labelPrefix = recordType === "bonus" ? "获奖" : "处罚";
          return (
            <section key={item.key} className="space-y-3">
              <div className="flex items-center justify-between">
                <h4 className="text-sm font-semibold text-foreground">{item.label}记录</h4>
              </div>
              <div className="grid gap-3 sm:grid-cols-2">
                <div>
                  <Label className="text-xs font-medium text-muted-foreground">{labelPrefix}日期</Label>
                  <Input
                    className="mt-1"
                    type="date"
                    value={entry.eventDate || ""}
                    onChange={(event) => handleRecordChange(item.key, { eventDate: event.target.value })}
                  />
                </div>
                <div>
                  <Label className="text-xs font-medium text-muted-foreground">{labelPrefix}理由</Label>
                  <Textarea
                    className="mt-1"
                    rows={3}
                    value={entry.reason || ""}
                    onChange={(event) => handleRecordChange(item.key, { reason: event.target.value })}
                  />
                </div>
              </div>
            </section>
          );
        }
        return null;
      })}
    </div>
  );
};
const renderRoomHistoryTimeline = () => {
  if (!editingRoomId) {
    return <p className="text-sm text-muted-foreground">保存房间信息后可查看入住历史。</p>;
  }
    const history = roomHistoryMap.get(editingRoomId) ?? [];
    if (history.length === 0) {
      return (
        <div className="rounded-lg border border-dashed bg-muted/30 px-4 py-6 text-sm text-muted-foreground">
          暂无历史记录，办理入住/退宿后将自动沉淀。
        </div>
);
    }
    return (
      <ol className="relative space-y-6 border-l border-dashed border-muted-foreground/40 pl-6">
        {history.map((record) => {
          const statusKey = record.status || "active";
          const statusMeta = CONTRACT_STATUS_META[statusKey] ?? {
            label: statusKey,
            badge: "bg-muted text-muted-foreground border-muted",
          };
          const noteMeta = parseContractNoteMeta(record.notes);
          return (
            <li key={`${record.id}-${record.start_date}`} className="relative rounded-lg bg-background/80 p-4 shadow-sm ring-1 ring-border">
              <span className="absolute -left-3 top-5 h-3 w-3 rounded-full bg-primary shadow" />
              <div className="flex flex-wrap items-center justify-between gap-2">
                <div>
                  <p className="text-sm font-semibold text-foreground">{record.employee_name || "未命名员工"}</p>
                  <p className="text-xs text-muted-foreground">
                    入住：{formatDateLabel(record.start_date)} · 床位：{record.bed?.bed_number || "未指定"}
                  </p>
                </div>
                <Badge variant="outline" className={`border ${statusMeta.badge}`}>
                  {statusMeta.label}
                </Badge>
              </div>
              <div className="mt-3 grid gap-2 text-xs text-muted-foreground sm:grid-cols-2">
                <p>退宿：{record.end_date ? formatDateLabel(record.end_date) : "—"}</p>
                <p>租金：{record.rent_amount ? `¥${record.rent_amount}` : "未填写"}</p>
                <p>押金：{record.deposit_amount ? `¥${record.deposit_amount}` : "未填写"}</p>
                <p>支付方式：{record.payment_method || "—"}</p>
                <p>押金月份：{formatMonthDisplay(noteMeta.depositPlanMonth)}</p>
                <p>租金月份：{formatMonthDisplay(noteMeta.rentPlanMonth)}</p>
              </div>
              {noteMeta.additionalNotes && (
                <p className="mt-2 rounded bg-muted/50 px-3 py-2 text-xs text-muted-foreground">备注：{noteMeta.additionalNotes}</p>
              )}
            </li>
          );
        })}
      </ol>
  );
};

  const renderContractDetailSections = () => (
    <>
      <section className="space-y-4">
        <div className="flex items-center justify-between">
          <h4 className="text-sm font-semibold text-foreground">人员信息</h4>
        </div>
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
          <div className="space-y-1 sm:col-span-2 lg:col-span-3 xl:col-span-4">
            <Label className="text-xs font-medium text-muted-foreground">
              员工姓名 <span className="text-destructive">*</span>
            </Label>
            <Input
              className="mt-1"
              value={contractForm.employee_name}
              placeholder="输入姓名、身份证号或工号搜索，支持手动录入"
              onChange={(event) => {
                const nextValue = event.target.value;
                setContractForm((prev) => ({ ...prev, employee_name: nextValue }));
                setEmployeeSearchTerm(nextValue);
              }}
            />
            <p className="text-[11px] text-muted-foreground">匹配花名册后可一键导入人员信息，未匹配时可直接填写。</p>
            {employeeLookupLoading && <p className="text-[11px] text-muted-foreground">员工信息加载中...</p>}
            {!employeeLookupLoading && employeeFetchError && allEmployees.length === 0 && (
              <p className="text-[11px] text-destructive">{employeeFetchError}</p>
            )}
            {employeeSuggestions.length > 0 && (
              <div className="max-h-40 space-y-1 overflow-y-auto rounded-md border bg-muted/40 p-2">
                {employeeSuggestions.map((employee) => (
                  <button
                    type="button"
                    key={employee.id}
                    className="w-full rounded-md px-2 py-1 text-left hover:bg-background"
                    onClick={() => handleSelectEmployee(employee)}
                  >
                    <p className="text-sm font-medium text-foreground">{employee.name}</p>
                    <p className="text-xs text-muted-foreground">
                      {(employee.department || "未分配部门") + " · " + (employee.id_number || "无身份证号")}
                    </p>
                  </button>
                ))}
              </div>
            )}
          </div>
          <div>
            <Label className="text-xs font-medium text-muted-foreground">部门</Label>
            <Input className="mt-1" value={contractForm.employee_department} onChange={(event) => setContractForm((prev) => ({ ...prev, employee_department: event.target.value }))} />
          </div>
          <div>
            <Label className="text-xs font-medium text-muted-foreground">岗位</Label>
            <Input className="mt-1" value={contractForm.employee_position} onChange={(event) => setContractForm((prev) => ({ ...prev, employee_position: event.target.value }))} />
          </div>
          <div>
            <Label className="text-xs font-medium text-muted-foreground">工号</Label>
            <Input className="mt-1" value={contractForm.employee_job_number} onChange={(event) => setContractForm((prev) => ({ ...prev, employee_job_number: event.target.value }))} />
          </div>
          <div>
            <Label className="text-xs font-medium text-muted-foreground">身份证号码</Label>
            <Input className="mt-1" value={contractForm.employee_id_number} onChange={(event) => setContractForm((prev) => ({ ...prev, employee_id_number: event.target.value }))} />
          </div>
          <div>
            <Label className="text-xs font-medium text-muted-foreground">户籍地址</Label>
            <Input className="mt-1" value={contractForm.residence_address} onChange={(event) => setContractForm((prev) => ({ ...prev, residence_address: event.target.value }))} />
          </div>
          <div>
            <Label className="text-xs font-medium text-muted-foreground">联系电话</Label>
            <Input className="mt-1" value={contractForm.employee_phone} onChange={(event) => setContractForm((prev) => ({ ...prev, employee_phone: event.target.value }))} />
          </div>
          <div>
            <Label className="text-xs font-medium text-muted-foreground">紧急联系人</Label>
            <Input className="mt-1" value={contractForm.emergency_contact} onChange={(event) => setContractForm((prev) => ({ ...prev, emergency_contact: event.target.value }))} />
          </div>
        </div>
      </section>

      <section className="space-y-4">
        <div className="flex items-center justify-between">
          <h4 className="text-sm font-semibold text-foreground">床位信息</h4>
        </div>
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
          <div>
            <Label className="text-xs font-medium text-muted-foreground">入住日期 *</Label>
            <Input className="mt-1" type="date" value={contractForm.start_date} onChange={(event) => setContractForm((prev) => ({ ...prev, start_date: event.target.value }))} />
          </div>
          <div className="sm:col-span-2 lg:col-span-1">
            <Label className="text-xs font-medium text-muted-foreground">
              房间 <span className="text-destructive">*</span>
            </Label>
            <select
              className="mt-1 w-full rounded-md border border-input bg-background px-3 py-2 text-sm"
              value={contractForm.room_id}
              onChange={(event) => setContractForm((prev) => ({ ...prev, room_id: event.target.value, bed_id: "" }))}
            >
              <option value="">请选择房间</option>
              {availableRoomsForContracts.map((room) => {
                const building = buildingById.get(room.building_id);
                const site = room.site_id ? siteById.get(room.site_id) : building?.site_id ? siteById.get(building.site_id) : undefined;
                return (
                  <option key={room.id} value={String(room.id)}>
                    {room.room_number} · {(building?.name || "未命名楼栋") + (site ? `（${site.name}）` : "")}
                  </option>
                );
              })}
            </select>
          </div>
          <div>
            <Label className="text-xs font-medium text-muted-foreground">床位选择</Label>
            {selectedRoom ? (
              (selectedRoom.beds?.length || 0) > 0 ? (
                <>
                  <select
                    className="mt-1 w-full rounded-md border border-input bg-background px-3 py-2 text-sm"
                    value={contractForm.bed_id}
                    onChange={(event) => setContractForm((prev) => ({ ...prev, bed_id: event.target.value }))}
                  >
                    <option value="">请选择床位</option>
                    {bedOptions.map((bed) => {
                      const isLocked = Boolean(
                        bed.occupiedBy && (!editingContractOriginalBedId || bed.id !== editingContractOriginalBedId),
                      );
                      return (
                        <option key={bed.id} value={bed.id} disabled={isLocked}>
                          {bed.bed_number}
                          {isLocked ? `（${bed.occupiedBy ? `${bed.occupiedBy}已入住` : "已占用"}）` : ""}
                        </option>
                      );
                    })}
                  </select>
                  {contractDialogTitle === "入住详情" && contractForm.bed_id && editingContractOriginalBedId && (
                    <p className="text-[11px] text-muted-foreground">编辑状态下将优先保留原床位。</p>
                  )}
                </>
              ) : (
                <p className="text-xs text-muted-foreground">该房间暂无床位信息，请先在房间详情中配置。</p>
              )
            ) : (
              <p className="text-xs text-muted-foreground">请选择房间后再选择床位。</p>
            )}
          </div>
        </div>
      </section>

      <section className="space-y-4">
        <div className="flex items-center justify-between">
          <h4 className="text-sm font-semibold text-foreground">基础信息</h4>
        </div>
        <div className={RESPONSIVE_FIELD_GRID_CLASS}>
          {(["water", "electric", "gas"] as const)
            .filter((key) => {
              if (!selectedRoomChargeSettings) return false;
              return selectedRoomChargeSettings.some((item) => item.key === key);
            })
            .map((key) => {
              const label = key === "water" ? "入住时水表度数" : key === "electric" ? "入住时电表度数" : "入住时气表度数";
              const fieldKey = (key === "water" ? "water_start" : key === "electric" ? "electric_start" : "gas_start") as
                | "water_start"
                | "electric_start"
                | "gas_start";
              const value = contractForm[fieldKey];
              const chargeDefinition = selectedRoomChargeSettings?.find((item) => item.key === key);
              const placeholder = chargeDefinition?.unitPrice != null
                ? `当前单价：${chargeDefinition.unitPrice}${chargeDefinition.unitLabel || "元"}`
                : undefined;
              return (
                <div key={key}>
                  <Label className="text-xs font-medium text-muted-foreground">{label}</Label>
                  <Input
                    className="mt-1"
                    type="number"
                    inputMode="decimal"
                    placeholder={placeholder}
                    value={value}
                    onChange={(event) => setContractForm((prev) => ({ ...prev, [fieldKey]: event.target.value }))}
                  />
                </div>
              );
            })}
        </div>
        <div className="rounded-lg border bg-muted/40 px-3 py-2 text-xs text-muted-foreground">
          {selectedRoom ? `该房间类型：${selectedRoom.cost_bearing_mode === "personal" ? "个人宿舍" : "公司宿舍"}。${selectedRoom.cost_bearing_mode === "personal" ? "房租及相关费用将计入入住人员账单" : "房租及相关费用由公司承担"}` : "请选择房间以查看费用承担方式。"}
        </div>
      </section>

      <section className="space-y-4">
        <div className="flex items-center justify-between">
          <h4 className="text-sm font-semibold text-foreground">备注</h4>
        </div>
        <Textarea
          rows={3}
          placeholder="补充费用分摊、入住说明等"
          value={contractForm.notes}
          onChange={(event) => setContractForm((prev) => ({ ...prev, notes: event.target.value }))}
        />
      </section>

    </>
  );

  const renderContractHistoryTimeline = () => {
    if (!editingContractId) {
      return <p className="text-sm text-muted-foreground">保存入住信息后可查看该员工的入住历史。</p>;
    }
    const currentName = contractForm.employee_name?.trim();
    const history = contracts
      .filter((record) => record.employee_name?.trim() === currentName)
      .sort((a, b) => new Date(b.start_date).getTime() - new Date(a.start_date).getTime());
    if (history.length === 0) {
      return (
        <div className="rounded-lg border border-dashed bg-muted/30 px-4 py-6 text-sm text-muted-foreground">
          暂无历史记录，办理入住/退宿后将自动沉淀。
        </div>
      );
    }
    return (
      <ol className="relative space-y-6 border-l border-dashed border-muted-foreground/40 pl-6">
        {history.map((record) => {
          const statusKey = record.status || "active";
          const statusMeta = CONTRACT_STATUS_META[statusKey] ?? {
            label: statusKey,
            badge: "bg-muted text-muted-foreground border-muted",
          };
          const noteMeta = parseContractNoteMeta(record.notes);
          return (
            <li key={`${record.id}-${record.start_date}`} className="relative rounded-lg bg-background/80 p-4 shadow-sm ring-1 ring-border">
              <span className="absolute -left-3 top-5 h-3 w-3 rounded-full bg-primary shadow" />
              <div className="flex flex-wrap items-center justify-between gap-2">
                <div>
                  <p className="text-sm font-semibold text-foreground">{record.room?.room_number || "未分配房间"}</p>
                  <p className="text-xs text-muted-foreground">
                    床位：{record.bed?.bed_number || "未指定"} · 入住：{formatDateLabel(record.start_date)}
                  </p>
                </div>
                <Badge variant="outline" className={`border ${statusMeta.badge}`}>
                  {statusMeta.label}
                </Badge>
              </div>
              <div className="mt-3 grid gap-2 text-xs text-muted-foreground sm:grid-cols-2">
                <p>退宿：{record.end_date ? formatDateLabel(record.end_date) : "—"}</p>
                <p>租金：{record.rent_amount ? `¥${record.rent_amount}` : "未填写"}</p>
                <p>押金：{record.deposit_amount ? `¥${record.deposit_amount}` : "未填写"}</p>
                <p>支付方式：{record.payment_method || "—"}</p>
                <p>押金月份：{formatMonthDisplay(noteMeta.depositPlanMonth)}</p>
                <p>租金月份：{formatMonthDisplay(noteMeta.rentPlanMonth)}</p>
              </div>
              {noteMeta.additionalNotes && (
                <p className="mt-2 rounded bg-muted/50 px-3 py-2 text-xs text-muted-foreground">备注：{noteMeta.additionalNotes}</p>
              )}
            </li>
          );
        })}
      </ol>
    );
  };
  const renderContractTable = () => {
    const contractAllSelected =
      sortedContracts.length > 0 && sortedContracts.every((contract) => selectedContractIds.includes(contract.id));
    const selectionCount = selectedContractIds.length;

    const renderToolbar = () => {
      if (selectionCount > 0) {
        return (
          <div className="flex flex-wrap gap-2">
            {/* P7.1：入住记录导出需 dormitory.view 权限 */}
            <RequirePermission resource="dormitory" action="view">
              <Button variant="outline" size="sm" className="gap-1" onClick={() => handleContractExport("selected")}>
                <Download className="h-4 w-4" /> 导出
              </Button>
            </RequirePermission>
            {/* P7.1：入住记录打印为查看类操作，需 dormitory.view 权限 */}
            <RequirePermission resource="dormitory" action="view">
              <Button variant="outline" size="sm" className="gap-1" onClick={() => handleContractPrintRequest("selected")}>
                <Printer className="h-4 w-4" /> 打印
              </Button>
            </RequirePermission>
          </div>
        );
      }
      return (
        <div className="flex flex-wrap gap-2">
          {/* P7.1：办理入住为创建合同，需 dormitory.create 权限 */}
          <RequirePermission resource="dormitory" action="create">
            <Button size="sm" className="gap-1" onClick={handleOpenContractDialog}>
              <Plus className="h-4 w-4" /> 办理入住
            </Button>
          </RequirePermission>
          {/* P7.1：导入入住记录需 dormitory.create 权限 */}
          <RequirePermission resource="dormitory" action="create">
            <Button variant="outline" size="sm" className="gap-1" onClick={() => handleContractImportDialogToggle(true)}>
              <Upload className="h-4 w-4" /> 导入
            </Button>
          </RequirePermission>
        </div>
      );
    };

    return (
      <Card className="shadow-sm border overflow-hidden">
        <CardHeader className="space-y-4">
          <div className="flex flex-wrap items-start justify-between gap-4">
            <div className="space-y-1">
              <CardTitle className="flex items-center gap-2 text-base">
                <BedDouble className="h-4 w-4" /> 入住列表
              </CardTitle>
              <p className="text-xs text-muted-foreground">已入住 {filteredContracts.filter((c) => (c.status || "active") === "active").length} 人</p>
            </div>
            {renderToolbar()}
          </div>
          <p className="text-xs text-muted-foreground">提示：双击数据行可编辑“入住”信息，更多操作请使用行尾菜单。</p>
        </CardHeader>
        <CardContent>
          <div className="mb-4 flex flex-wrap items-center gap-3">
            <div className="relative min-w-[220px] flex-1">
              <Search className="pointer-events-none absolute left-3 top-3 h-4 w-4 text-muted-foreground" />
              <Input
                placeholder="搜索姓名、部门或房间..."
                value={contractSearch}
                onChange={(event) => setContractSearch(event.target.value)}
                className="pl-9"
              />
              {contractSearch && (
                <button
                  type="button"
                  onClick={() => setContractSearch("")}
                  className="absolute right-3 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground"
                  aria-label="清除搜索"
                >
                  <X className="h-4 w-4" />
                </button>
              )}
            </div>
            <Select
              value={contractSiteFilter === "all" ? "all" : String(contractSiteFilter)}
              onValueChange={(value) => setContractSiteFilter(value === "all" ? "all" : Number(value))}
            >
              <SelectTrigger className="w-40">
                <SelectValue placeholder="地点筛选" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">全部地点</SelectItem>
                {sites.map((site) => (
                  <SelectItem key={site.id} value={String(site.id)}>
                    {site.name || `地点${site.id}`}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
            <Select value={contractStatusFilter} onValueChange={(value) => setContractStatusFilter(value as "all" | "active" | "completed")}>
              <SelectTrigger className="w-36">
                <SelectValue placeholder="状态筛选" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">全部状态</SelectItem>
                <SelectItem value="active">已入住</SelectItem>
                <SelectItem value="completed">已退宿</SelectItem>
              </SelectContent>
            </Select>
            <Dialog open={showContractFieldSelector} onOpenChange={setShowContractFieldSelector}>
              <DialogTrigger asChild>
                <Button variant="outline" size="sm" className="gap-1">
                  <Settings className="h-4 w-4" /> 显示字段
                </Button>
              </DialogTrigger>
              <DialogContent className={DIALOG_SIZES.sm}>
                <DialogHeader>
                  <DialogTitle className="flex items-center gap-2">
                    <Settings className="h-4 w-4" />
                    自定义显示字段
                  </DialogTitle>
                  <DialogDescription>选择需要在入住列表中显示的字段。</DialogDescription>
                </DialogHeader>
                <div className="max-h-80 overflow-y-auto space-y-2">
                  {effectiveContractColumnConfig.map((column) => (
                    <label key={column.id} className="flex items-center gap-2 text-sm font-medium text-foreground">
                      <input
                        type="checkbox"
                        className="h-4 w-4 rounded border-muted-foreground"
                        checked={contractColumnVisibility[column.id] !== false}
                        onChange={() => handleContractFieldToggle(column.id)}
                      />
                      <span>{column.label}</span>
                    </label>
                  ))}
                </div>
                <DialogFooter className="flex flex-col gap-2 sm:flex-row sm:justify-end">
                  <Button variant="outline" onClick={resetContractFieldVisibility}>
                    恢复默认
                  </Button>
                  <Button onClick={() => setShowContractFieldSelector(false)}>完成</Button>
                </DialogFooter>
              </DialogContent>
            </Dialog>
          </div>
          <DataTableWrapper height="h-[65vh]">
              <Table className="min-w-full table-auto text-sm">
                <TableHeader>
                  <TableRow className="text-muted-foreground">
                    <TableHead className={cn("w-12", ALIGNMENT_CLASS.center)}>
                      <input
                        type="checkbox"
                        className="h-4 w-4 rounded border-muted-foreground"
                        checked={contractAllSelected}
                        onChange={(event) => handleContractSelectAll(event.target.checked, sortedContracts)}
                      />
                    </TableHead>
                    {visibleContractColumns.map((column) => (
                      <TableHead
                        key={column.id}
                        draggable
                        onDragStart={(event) => handleContractColumnDragStart(event, column.id)}
                        onDragOver={handleContractColumnDragOver}
                        onDrop={(event) => handleContractColumnDrop(event, column.id)}
                        onDragEnd={handleContractColumnDragEnd}
                        onClick={column.sortable === false ? undefined : () => handleContractSortClick(column.id)}
                        className={cn("select-none whitespace-normal break-words", ALIGNMENT_CLASS.left)}
                      >
                        <span className="flex items-center gap-1">
                          {column.label}
                          {column.sortable !== false && contractSort.columnId === column.id && (
                            <span>{contractSort.direction === "asc" ? "↑" : "↓"}</span>
                          )}
                        </span>
                      </TableHead>
                    ))}
                    <TableHead className={cn("w-14", ALIGNMENT_CLASS.center)}>操作</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {sortedContracts.length === 0 ? (
                    <TableRow>
                      <TableCell colSpan={visibleContractColumns.length + 1} className="py-6 text-center text-muted-foreground">
                        暂无符合条件的入住记录
                      </TableCell>
                    </TableRow>
                  ) : (
                    sortedContracts.map((contract) => {
                      const checked = selectedContractIds.includes(contract.id);
                      const statusValue = contract.status || "active";
                      return (
                        <TableRow
                          key={contract.id}
                          className="text-sm hover:bg-muted/50"
                          onDoubleClick={() => handleEditContract(contract)}
                        >
                          <TableCell className={ALIGNMENT_CLASS.center}>
                            <input
                              type="checkbox"
                              className="h-4 w-4 rounded border-muted-foreground"
                              checked={checked}
                              onChange={() => toggleContractSelection(contract.id)}
                            />
                          </TableCell>
                          {visibleContractColumns.map((column) => (
                            <TableCell key={column.id} className={cn("align-top text-sm whitespace-normal break-words", ALIGNMENT_CLASS.left)}>
                              {column.render(contract, contractColumnHelpers)}
                            </TableCell>
                          ))}
                          <TableCell className={cn("w-14", ALIGNMENT_CLASS.center)}>
                            <DropdownMenu>
                              <DropdownMenuTrigger asChild>
                                <Button variant="ghost" size="icon" className="h-8 w-8" aria-label="入住操作">
                                  <MoreHorizontal className="h-4 w-4" />
                                </Button>
                              </DropdownMenuTrigger>
                          <DropdownMenuContent align="end">
                                <DropdownMenuItem onClick={() => handleEditContract(contract)} className="gap-2">
                                  <Eye className="h-4 w-4" />
                                  查看详情
                                </DropdownMenuItem>
                                {/* P7.1：办理退宿为修改合同状态，需 dormitory.edit 权限 */}
                                <RequirePermission resource="dormitory" action="edit">
                                  <DropdownMenuItem
                                    disabled={(statusValue || "active") !== "active"}
                                    onClick={() => handleOpenCheckoutDialog(contract)}
                                    className="gap-2"
                                  >
                                    <LogOut className="h-4 w-4" />
                                    办理退宿
                                  </DropdownMenuItem>
                                </RequirePermission>
                                {/* P7.1：撤销退宿为修改合同状态，需 dormitory.edit 权限 */}
                                <RequirePermission resource="dormitory" action="edit">
                                  <DropdownMenuItem
                                    disabled={statusValue !== "completed"}
                                    onClick={() => handleRevokeCheckout(contract)}
                                    className="gap-2"
                                  >
                                    <RotateCcw className="h-4 w-4" />
                                    撤销退宿
                                  </DropdownMenuItem>
                                </RequirePermission>
                                {/* P7.1：删除入住记录需 dormitory.delete 权限 */}
                                <RequirePermission resource="dormitory" action="delete">
                                  <DropdownMenuItem className="gap-2 text-destructive" onClick={() => requestDeleteContract(contract)}>
                                    <Trash2 className="h-4 w-4" />
                                    删除记录
                                  </DropdownMenuItem>
                                </RequirePermission>
                              </DropdownMenuContent>
                            </DropdownMenu>
                          </TableCell>
                        </TableRow>
                      );
                    })
                  )}
                </TableBody>
              </Table>
              <ScrollBar orientation="horizontal" />
            </DataTableWrapper>

        </CardContent>
      </Card>
    );
  };

const renderMeterManagement = () => {
  const currentRows = meterListTab === "personal" ? sortedPersonalRows : sortedCompanyRows;
  const currentSummary = meterListTab === "personal" ? meterPersonalSummary : meterCompanySummary;
  const currentRowIdSet = new Set(currentRows.map((row) => row.source.id));
  const selectionCount = meterSelectedIds.filter((id) => currentRowIdSet.has(id)).length;
  const meterCountLabel = `${currentRows.length} 条记录`;
  const meterAllSelected = currentRows.length > 0 && currentRows.every((row) => meterSelectedIds.includes(row.source.id));

  const renderToolbar = () => {
    if (selectionCount === 0) {
      return (
        <div className="flex flex-wrap gap-2">
          {/* P7.1：抄表录入为创建抄表记录，需 dormitory.create 权限 */}
          <RequirePermission resource="dormitory" action="create">
            <Button
              size="sm"
              className="gap-1"
              onClick={() => {
                handleResetMeterForm({
                  overrideBillingMonth: meterPeriodFilter !== "all" ? meterPeriodFilter : null,
                });
                setMeterDialogOpen(true);
              }}
            >
              <Gauge className="h-4 w-4" /> 抄表录入
            </Button>
          </RequirePermission>
          {/* P7.1：导入抄表记录需 dormitory.create 权限 */}
          <RequirePermission resource="dormitory" action="create">
            <Button variant="outline" size="sm" className="gap-1" onClick={() => handleMeterImportDialogToggle(true)}>
              <Upload className="h-4 w-4" />
              导入
            </Button>
          </RequirePermission>
        </div>
      );
    }
    return (
      <div className="flex flex-wrap gap-2">
        {/* P7.1：抄表导出需 dormitory.view 权限 */}
        <RequirePermission resource="dormitory" action="view">
          <Button variant="outline" size="sm" className="gap-1" onClick={() => handleMeterExport("selected", currentRows)}>
            <Download className="h-4 w-4" />
            导出
          </Button>
        </RequirePermission>
        {/* P7.1：抄表打印为查看类操作，需 dormitory.view 权限 */}
        <RequirePermission resource="dormitory" action="view">
          <Button variant="outline" size="sm" className="gap-1" onClick={() => handleMeterPrintRequest("selected", currentRows)}>
            <Printer className="h-4 w-4" />
            打印
          </Button>
        </RequirePermission>
        {/* P7.1：生成账单为创建操作，需 dormitory.create 权限 */}
        <RequirePermission resource="dormitory" action="create">
          <Button variant="outline" size="sm" className="gap-1" onClick={handleGenerateBillsFromSelection} disabled={billGenerating}>
            <FileText className="h-4 w-4" />
            {billGenerating ? "生成中..." : "生成账单"}
          </Button>
        </RequirePermission>
        {/* P7.1：批量删除抄表记录需 dormitory.delete 权限 */}
        <RequirePermission resource="dormitory" action="delete">
          <Button variant="destructive" size="sm" className="gap-1" onClick={requestBulkDeleteMeterRecords}>
            <Trash2 className="h-4 w-4" />
            批量删除
          </Button>
        </RequirePermission>
      </div>
    );
  };

  return (
    <div className="space-y-6 pb-36 lg:pb-32">
      <Card className="shadow-sm border overflow-hidden">
        <CardHeader className="space-y-3">
          <div className="flex flex-col gap-3">
            <Tabs value={meterListTab} onValueChange={(value) => setMeterListTab(value as "personal" | "company")}>
              <TabsList className="inline-flex rounded-full border bg-muted/30 p-0.5">
                <TabsTrigger
                  value="personal"
                  className="rounded-full px-4 py-1 text-sm data-[state=active]:bg-primary data-[state=active]:text-primary-foreground"
                >
                  个人计费
                </TabsTrigger>
                <TabsTrigger
                  value="company"
                  className="rounded-full px-4 py-1 text-sm data-[state=active]:bg-primary data-[state=active]:text-primary-foreground"
                >
                  单位计费
                </TabsTrigger>
              </TabsList>
            </Tabs>
            <div className="flex flex-wrap items-center justify-between gap-3">
              <p className="text-sm text-muted-foreground">当前记录：{meterCountLabel}</p>
              <div className="flex flex-wrap items-center gap-3">
                <Select value={meterPeriodFilter} onValueChange={(value) => setMeterPeriodFilter(value)}>
                  <SelectTrigger className="w-36">
                    <SelectValue placeholder="全部账期" />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="all">全部账期</SelectItem>
                    {meterPeriodOptions.map((option) => (
                      <SelectItem key={option.key} value={option.key}>
                        {option.label}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
                {renderToolbar()}
              </div>
            </div>
          </div>
          <div className="flex flex-wrap items-center gap-3">
            <div className="relative min-w-[220px] flex-1">
              <Search className="pointer-events-none absolute left-3 top-3 h-4 w-4 text-muted-foreground" />
              <Input
                placeholder="搜索房号、楼栋或入住人"
                value={meterSearch}
                onChange={(event) => setMeterSearch(event.target.value)}
                className="pl-9"
              />
              {meterSearch && (
                <button
                  type="button"
                  className="absolute right-3 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground"
                  onClick={() => setMeterSearch("")}
                  aria-label="清除搜索"
                >
                  <X className="h-4 w-4" />
                </button>
              )}
            </div>
            <Select value={meterSiteFilter === "all" ? "all" : String(meterSiteFilter)} onValueChange={(value) => setMeterSiteFilter(value === "all" ? "all" : Number(value))}>
              <SelectTrigger className="w-40">
                <SelectValue placeholder="地点筛选" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">全部地点</SelectItem>
                {sites.map((site) => (
                  <SelectItem key={site.id} value={String(site.id)}>
                    {site.name}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
            <Select value={meterBuildingFilter === "all" ? "all" : String(meterBuildingFilter)} onValueChange={(value) => setMeterBuildingFilter(value === "all" ? "all" : Number(value))}>
              <SelectTrigger className="w-40">
                <SelectValue placeholder="楼栋筛选" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">全部楼栋</SelectItem>
                {meterFilterBuildings.map((building) => (
                  <SelectItem key={building.id} value={String(building.id)}>
                    {building.name || `楼栋${building.id}`}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
            <Dialog open={showMeterFieldSelector} onOpenChange={setShowMeterFieldSelector}>
              <DialogTrigger asChild>
                <Button variant="outline" size="sm" className="gap-1">
                  <Settings className="h-4 w-4" /> 显示字段
                </Button>
              </DialogTrigger>
              <DialogContent className={DIALOG_SIZES.sm}>
                <DialogHeader>
                  <DialogTitle className="flex items-center gap-2">
                    <Settings className="h-4 w-4" /> 自定义显示字段
                  </DialogTitle>
                  <DialogDescription>选择需要在抄表计费列表中显示的字段。</DialogDescription>
                </DialogHeader>
                <div className="max-h-80 overflow-y-auto space-y-2">
                  {effectiveMeterColumnConfig.map((column) => (
                    <label key={column.id} className="flex items-center gap-2 text-sm font-medium text-foreground">
                      <input
                        type="checkbox"
                        className="h-4 w-4 rounded border-muted-foreground"
                        checked={meterColumnVisibility[column.id] !== false}
                        onChange={() => handleMeterFieldToggle(column.id)}
                      />
                      <span>{column.label}</span>
                    </label>
                  ))}
                  {supplementalChargeKeys.length > 0 && (
                    <div className="space-y-2 pt-2">
                      <p className="text-xs text-muted-foreground">附加费用字段</p>
                      {supplementalChargeKeys.map((key) => (
                        <label key={`extra-field-${key}`} className="flex items-center gap-2 text-sm font-medium text-foreground">
                          <input
                            type="checkbox"
                            className="h-4 w-4 rounded border-muted-foreground"
                            checked={meterExtraColumnVisibility[key] !== false}
                            onChange={() => handleMeterExtraFieldToggle(key)}
                          />
                          <span>{getChargeDefinition(key)?.label || key}</span>
                        </label>
                      ))}
                    </div>
                  )}
                </div>
                <DialogFooter className="flex flex-col gap-2 sm:flex-row sm:justify-end">
                  <Button variant="outline" onClick={resetMeterFieldVisibility}>
                    恢复默认
                  </Button>
                  <Button onClick={() => setShowMeterFieldSelector(false)}>完成</Button>
                </DialogFooter>
              </DialogContent>
            </Dialog>
          </div>
        </CardHeader>
        <CardContent>
          <DataTableWrapper height="h-[65vh]">
              <Table className="min-w-full table-auto text-sm">
                <TableHeader>
                  <TableRow className="text-muted-foreground">
                    <TableHead className={cn("w-12", ALIGNMENT_CLASS.center)}>
                      <input
                        type="checkbox"
                        className="h-4 w-4 rounded border-muted-foreground"
                        checked={meterAllSelected}
                        onChange={(event) => handleMeterSelectAll(event.target.checked, currentRows)}
                        aria-label="选择全部抄表记录"
                      />
                    </TableHead>
                    {visibleMeterColumns.map((column) => (
                      <TableHead
                        key={column.id}
                        draggable
                        onDragStart={(event) => handleMeterColumnDragStart(event, column.id)}
                        onDragOver={handleMeterColumnDragOver}
                        onDrop={(event) => handleMeterColumnDrop(event, column.id)}
                        onDragEnd={handleMeterColumnDragEnd}
                        onClick={() => handleMeterSortClick(column.id)}
                        className={cn("select-none whitespace-nowrap", ALIGNMENT_CLASS.left)}
                      >
                        <span className="flex items-center gap-1">
                          {column.label}
                          {meterSort.key === column.id && <span>{meterSort.direction === "asc" ? "↑" : "↓"}</span>}
                        </span>
                      </TableHead>
                    ))}
                    {visibleSupplementalKeys.map((key) => (
                      <TableHead key={`extra-head-${key}`} className={cn("whitespace-nowrap", ALIGNMENT_CLASS.right)}>
                        {(getChargeDefinition(key)?.label || key) + "金额"}
                      </TableHead>
                    ))}
                    <TableHead className={cn("w-14", ALIGNMENT_CLASS.center)}>操作</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {currentRows.length === 0 ? (
                    <TableRow>
                      <TableCell colSpan={visibleMeterColumns.length + visibleSupplementalKeys.length + 2} className="py-6 text-center text-muted-foreground">
                        暂无抄表记录
                      </TableCell>
                    </TableRow>
                  ) : (
                    currentRows.map((row) => (
                      <TableRow
                        key={row.key}
                        className={meterSelectedIds.includes(row.source.id) ? "bg-muted/30" : undefined}
                        onDoubleClick={() => handleEditMeterRecord(row.source)}
                      >
                        <TableCell className={ALIGNMENT_CLASS.center}>
                          <input
                            type="checkbox"
                            className="h-4 w-4 rounded border-muted-foreground"
                            checked={meterSelectedIds.includes(row.source.id)}
                            onChange={() => toggleMeterSelection(row.source.id)}
                            aria-label="选择抄表记录"
                          />
                        </TableCell>
                        {visibleMeterColumns.map((column) => (
                          <TableCell key={column.id} className={cn("align-top whitespace-nowrap", ALIGNMENT_CLASS.left)}>
                            {getMeterColumnDisplay(row.display, column.id)}
                          </TableCell>
                        ))}
                        {visibleSupplementalKeys.map((key) => {
                          const amount = row.display.charges.find((charge) => charge.key === key)?.amount ?? null;
                          return (
                            <TableCell key={`extra-${row.key}-${key}`} className={cn("align-top tabular-nums whitespace-nowrap", ALIGNMENT_CLASS.right)}>
                              {amount != null ? formatCurrencyValue(amount) : "--"}
                            </TableCell>
                          );
                        })}
                        <TableCell className={cn("w-14", ALIGNMENT_CLASS.center)}>
                          <DropdownMenu>
                            <DropdownMenuTrigger asChild>
                              <Button variant="ghost" size="icon" className="h-8 w-8">
                                <MoreHorizontal className="h-4 w-4" />
                              </Button>
                            </DropdownMenuTrigger>
                            <DropdownMenuContent align="end">
                              {/* P7.1：编辑抄表记录需 dormitory.edit 权限 */}
                              <RequirePermission resource="dormitory" action="edit">
                                <DropdownMenuItem className="gap-2" onClick={() => handleEditMeterRecord(row.source)}>
                                  <PenSquare className="h-4 w-4" /> 编辑
                                </DropdownMenuItem>
                              </RequirePermission>
                              {/* P7.1：删除抄表记录需 dormitory.delete 权限 */}
                              <RequirePermission resource="dormitory" action="delete">
                                <DropdownMenuItem className="gap-2 text-destructive" onClick={() => requestDeleteMeterRecord(row.source)}>
                                  <Trash2 className="h-4 w-4" /> 删除
                                </DropdownMenuItem>
                              </RequirePermission>
                            </DropdownMenuContent>
                          </DropdownMenu>
                        </TableCell>
                      </TableRow>
                    ))
                  )}
                  {currentRows.length > 0 && (
                    <TableRow className="bg-muted/40 font-medium">
                      <TableCell className={ALIGNMENT_CLASS.center}>合计</TableCell>
                      {visibleMeterColumns.map((column) => {
                        if (column.id === "electricFee") {
                          return (
                            <TableCell key={column.id} className={cn("tabular-nums", ALIGNMENT_CLASS.right)}>
                              {formatCurrencyValue(currentSummary.electric)}
                            </TableCell>
                          );
                        }
                        if (column.id === "waterFee") {
                          return (
                            <TableCell key={column.id} className={cn("tabular-nums", ALIGNMENT_CLASS.right)}>
                              {formatCurrencyValue(currentSummary.water)}
                            </TableCell>
                          );
                        }
                        if (column.id === "gasFee") {
                          return (
                            <TableCell key={column.id} className={cn("tabular-nums", ALIGNMENT_CLASS.right)}>
                              {formatCurrencyValue(currentSummary.gas)}
                            </TableCell>
                          );
                        }
                        return <TableCell key={column.id} />;
                      })}
                      {visibleSupplementalKeys.map((key) => (
                        <TableCell key={`extra-sum-${key}`} className={cn("tabular-nums", ALIGNMENT_CLASS.right)}>
                          {formatCurrencyValue(currentSummary.extras[key] || 0)}
                        </TableCell>
                      ))}
                      <TableCell />
                    </TableRow>
                  )}
                </TableBody>
              </Table>
              <ScrollBar orientation="horizontal" />
            </DataTableWrapper>

        </CardContent>
      </Card>
    </div>
  );
};

const renderBillCard = () => (
    <Card className="shadow-sm">
      <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
        <CardTitle className="text-base flex items-center gap-2">
          <FileText className="h-4 w-4" />待处理账单
        </CardTitle>
      </CardHeader>
      <CardContent>
        {bills.length === 0 ? (
          <p className="text-sm text-muted-foreground">暂无账单</p>
        ) : (
          <div className="space-y-3">
            {bills.map((bill) => (
              <div key={bill.id} className="flex items-center justify-between rounded-md border p-3 text-sm">
                <div>
                  <p className="font-medium">账单号：{bill.bill_code || bill.id}</p>
                  <p className="text-xs text-muted-foreground">{bill.employee_name || "未指定"} | {bill.period_label || "当前周期"}</p>
                </div>
                <div className="text-right">
                  <p className="font-semibold text-primary">¥{bill.amount_due?.toFixed(2) || "0.00"}</p>
                  <Badge variant="outline" className="mt-1">{bill.status}</Badge>
                </div>
              </div>
            ))}
          </div>
        )}
      </CardContent>
    </Card>
  );

  const handleDeleteSite = async () => {
    if (!siteDeleteTarget) return;
    setSiteDeleting(true);
    try {
      await deleteDormSite(siteDeleteTarget.id);
      setSites((prev) => prev.filter((item) => item.id !== siteDeleteTarget.id));
      const key = memoKey(siteDeleteTarget.id);
      setSiteHouseExtras((prev) => {
        const next = { ...prev };
        delete next[key];
        return next;
      });
      setSiteContractExtras((prev) => {
        const next = { ...prev };
        delete next[key];
        return next;
      });
      setSiteMemos((prev) => {
        const next = { ...prev };
        delete next[key];
        return next;
      });
      toast.success("地点已删除");
    } catch (error) {
      console.error("[Dormitory] delete site failed", error);
      toast.error(error instanceof Error ? error.message : "删除失败");
    } finally {
      setSiteDeleting(false);
      setSiteDeleteTarget(null);
    }
  };

  const handleRoomColumnDragStart = (event: React.DragEvent<HTMLTableHeaderCellElement>, columnId: string) => {
    setDraggingColumn(columnId);
    event.dataTransfer.effectAllowed = "move";
  };

  const handleRoomColumnDragOver = (event: React.DragEvent<HTMLTableHeaderCellElement>) => {
    event.preventDefault();
    event.dataTransfer.dropEffect = "move";
  };

  const handleRoomColumnDrop = (event: React.DragEvent<HTMLTableHeaderCellElement>, targetId: string) => {
    event.preventDefault();
    if (!draggingColumn || draggingColumn === targetId) return;
    const currentOrder = [...roomColumnOrder];
    const fromIndex = currentOrder.indexOf(draggingColumn);
    const toIndex = currentOrder.indexOf(targetId);
    if (fromIndex === -1 || toIndex === -1) return;
    currentOrder.splice(fromIndex, 1);
    currentOrder.splice(toIndex, 0, draggingColumn);
    setRoomColumnOrder(currentOrder);
    setDraggingColumn(null);
  };

  const handleRoomColumnDragEnd = () => setDraggingColumn(null);

  const handleMeterColumnDragStart = (event: React.DragEvent<HTMLTableHeaderCellElement>, columnId: MeterColumnId) => {
    setDraggingMeterColumn(columnId);
    event.dataTransfer.effectAllowed = "move";
  };

  const handleMeterColumnDragOver = (event: React.DragEvent<HTMLTableHeaderCellElement>) => {
    event.preventDefault();
    event.dataTransfer.dropEffect = "move";
  };

  const handleMeterColumnDrop = (event: React.DragEvent<HTMLTableHeaderCellElement>, targetId: MeterColumnId) => {
    event.preventDefault();
    if (!draggingMeterColumn || draggingMeterColumn === targetId) return;
    setMeterColumnOrder((prev) => {
      const next = [...prev];
      const fromIndex = next.indexOf(draggingMeterColumn);
      const toIndex = next.indexOf(targetId);
      if (fromIndex === -1 || toIndex === -1) return prev;
      next.splice(fromIndex, 1);
      next.splice(toIndex, 0, draggingMeterColumn);
      return next;
    });
    setDraggingMeterColumn(null);
  };

  const handleMeterColumnDragEnd = () => setDraggingMeterColumn(null);

  const handleRoomSortClick = (columnId: string) => {
    setRoomSort((prev) => {
      if (prev.columnId === columnId) {
        return { columnId, direction: prev.direction === "asc" ? "desc" : "asc" };
      }
      return { columnId, direction: "asc" };
    });
  };

  const handleContractColumnDragStart = (event: React.DragEvent<HTMLTableHeaderCellElement>, columnId: string) => {
    setDraggingContractColumn(columnId);
    event.dataTransfer.effectAllowed = "move";
  };

  const handleContractColumnDrop = (event: React.DragEvent<HTMLTableHeaderCellElement>, targetId: string) => {
    event.preventDefault();
    if (!draggingContractColumn || draggingContractColumn === targetId) return;
    setContractColumnOrder((prev) => {
      const next = [...prev];
      const from = next.indexOf(draggingContractColumn);
      const to = next.indexOf(targetId);
      if (from === -1 || to === -1) return prev;
      next.splice(from, 1);
      next.splice(to, 0, draggingContractColumn);
      return next;
    });
    setDraggingContractColumn(null);
  };

  const handleContractColumnDragOver = (event: React.DragEvent<HTMLTableHeaderCellElement>) => {
    event.preventDefault();
    event.dataTransfer.dropEffect = "move";
  };

  const handleContractColumnDragEnd = () => setDraggingContractColumn(null);

  const handleContractSortClick = (columnId: string) => {
    setContractSort((prev) => {
      if (prev.columnId === columnId) {
        return { columnId, direction: prev.direction === "asc" ? "desc" : "asc" };
      }
      return { columnId, direction: "asc" };
    });
  };

  return (
    <PageTransition className="mx-auto flex w-full max-w-none flex-col gap-6 p-6 pb-16">
      <header className="flex flex-col gap-2">
        <div className="flex flex-wrap items-center justify-between gap-4">
          <div>
            <h1 className="text-3xl font-bold tracking-tight">宿舍管理</h1>
            <p className="text-muted-foreground">统一维护宿舍地点、房间资源与入住分配</p>
          </div>
        </div>
      </header>

      <Tabs value={activeTab} onValueChange={setActiveTab} className="space-y-6">
        <TabsList className="w-full justify-start overflow-x-auto">
          <TabsTrigger value="overview">基础信息</TabsTrigger>
          <TabsTrigger value="contracts">入住管理</TabsTrigger>
          <TabsTrigger value="billing">抄表计费</TabsTrigger>
          <TabsTrigger value="bills">账单中心</TabsTrigger>
        </TabsList>

        <TabsContent value="overview" className="space-y-6">
          {loading ? (
            <p className="text-muted-foreground text-sm">正在加载宿舍信息...</p>
          ) : (
            <>
              {renderSiteCards()}
              {renderRoomTable()}
            </>
          )}
        </TabsContent>

        <TabsContent value="contracts">
          {renderContractTable()}
        </TabsContent>

        <TabsContent value="billing">
          {renderMeterManagement()}
        </TabsContent>

        <TabsContent value="bills">
          {renderBillCard()}
        </TabsContent>
      </Tabs>

      {/* 地点表单 */}
      <Dialog
        open={siteDialogOpen}
        onOpenChange={(open) => {
          setSiteDialogOpen(open);
          if (!open) {
            setEditingSite(null);
            setSiteForm(blankSiteForm);
            setSiteHouseForm(blankHouseForm);
            setSiteContractForm(createBlankSiteContractForm());
            setSiteChargeItems(createDefaultChargeSettings());
            setSiteInventoryItems(createDefaultInventorySettings());
          }
        }}
      >
        <DialogContent className={RESPONSIVE_DIALOG_CLASS}>
          <DialogHeader className="px-6 pt-6">
            <DialogTitle className="flex items-center gap-2">
              <MapPin className="h-4 w-4" />
              {editingSite ? "宿舍详情" : "新增地点"}
            </DialogTitle>
            <p className="text-sm text-muted-foreground">录入地点、楼栋与合同信息后，可在房间表内继续配置。</p>
          </DialogHeader>
          <Tabs value={siteDialogTab} onValueChange={setSiteDialogTab} className="flex flex-1 min-h-0 flex-col">
            <div className="px-6">
              <TabsList className="w-full justify-start overflow-x-auto rounded-lg border bg-muted/40 p-1">
                <TabsTrigger value="house" className="whitespace-nowrap">
                  房屋信息
                </TabsTrigger>
                <TabsTrigger value="contract" className="whitespace-nowrap">
                  合同信息
                </TabsTrigger>
                <TabsTrigger value="memo" className="whitespace-nowrap">
                  备忘录
                </TabsTrigger>
              </TabsList>
            </div>
            <ScrollArea className={DIALOG_SCROLL_CLASS}>
              <div className="space-y-6 px-6 pb-6 min-h-0">
                <TabsContent value="house" className="space-y-6 pb-6">
                  <section className="space-y-4">
                    <h4 className="text-sm font-semibold text-foreground">地点信息</h4>
                    <div className={RESPONSIVE_FIELD_GRID_CLASS}>
                      <div>
                        <Label className="text-xs font-medium text-muted-foreground">地点名称 *</Label>
                        <Input className="mt-1" value={siteForm.name} onChange={(event) => setSiteForm((prev) => ({ ...prev, name: event.target.value }))} />
                      </div>
                      <div>
                        <Label className="text-xs font-medium text-muted-foreground">坐落地址</Label>
                        <Input className="mt-1" value={siteForm.address} onChange={(event) => setSiteForm((prev) => ({ ...prev, address: event.target.value }))} />
                      </div>
                      <div>
                        <Label className="text-xs font-medium text-muted-foreground">楼栋名称</Label>
                        <Input className="mt-1" value={siteForm.building_name} onChange={(event) => setSiteForm((prev) => ({ ...prev, building_name: event.target.value }))} />
                      </div>
                      <div>
                        <Label className="text-xs font-medium text-muted-foreground">楼栋号</Label>
                        <Input className="mt-1" value={siteForm.building_number} onChange={(event) => setSiteForm((prev) => ({ ...prev, building_number: event.target.value }))} />
                      </div>
                      <div>
                        <Label className="text-xs font-medium text-muted-foreground">物业公司</Label>
                        <Input className="mt-1" value={siteHouseForm.propertyCompany} onChange={(event) => setSiteHouseForm((prev) => ({ ...prev, propertyCompany: event.target.value }))} />
                      </div>
                      <div>
                        <Label className="text-xs font-medium text-muted-foreground">联系信息</Label>
                        <Input className="mt-1" value={siteHouseForm.propertyContact} onChange={(event) => setSiteHouseForm((prev) => ({ ...prev, propertyContact: event.target.value }))} />
                      </div>
                      <div className="sm:col-span-2">
                        <Label className="text-xs font-medium text-muted-foreground">客服微信链接 / 二维码</Label>
                        <Input
                          className="mt-1"
                          placeholder="支持粘贴二维码图片链接或 weixin:// 协议"
                          value={siteForm.support_wechat}
                          onChange={(event) => setSiteForm((prev) => ({ ...prev, support_wechat: event.target.value }))}
                        />
                        <p className="mt-1 text-[11px] text-muted-foreground">仅管理员可配置，所有用户将在底部工具栏看到客服入口。</p>
                      </div>
                      <div className="sm:col-span-2">
                        <Label className="text-xs font-medium text-muted-foreground">备注</Label>
                        <Input className="mt-1" value={siteForm.description} onChange={(event) => setSiteForm((prev) => ({ ...prev, description: event.target.value }))} />
                      </div>
                    </div>
                  </section>
                  <section className="space-y-3">
                    <div>
                      <h4 className="text-sm font-semibold text-foreground">扣费项目</h4>
                      <p className="mt-1 text-xs text-muted-foreground">勾选后，该地点所有房间会默认启用相同扣费项，电费/水费为必选项。</p>
                    </div>
                    <div className="space-y-3">
                      {CHARGE_GROUPS.map((group) => {
                        const groupSet = new Set(group.keys);
                        const groupItems = siteChargeItems.filter((item) => groupSet.has(item.key));
                        if (groupItems.length === 0) return null;
                        return (
                          <div key={group.title} className="space-y-2 rounded-xl border bg-muted/10 px-3 py-3">
                            <p className="text-xs font-semibold text-muted-foreground">{group.title}</p>
                            <div className="grid gap-2 max-[550px]:grid-cols-2 max-[420px]:grid-cols-1 grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5 xl:grid-cols-6 2xl:grid-cols-7">
                              {groupItems.map((item) => {
                                const mandatory = isMandatoryChargeKey(item.key);
                                return (
                                  <label key={item.key} className="flex min-w-0 cursor-pointer items-center gap-2 rounded-md border border-transparent px-2 py-1 text-sm hover:border-border min-w-[120px]">
                                    <Checkbox
                                      checked={item.enabled}
                                      disabled={mandatory}
                                      onCheckedChange={(checked) => handleSiteChargeToggle(item.key, checked === true)}
                                    />
                                    <span className="flex-1 truncate">{item.label}</span>
                                  </label>
                                );
                              })}
                            </div>
                          </div>
                        );
                      })}
                    </div>
                  </section>
                  <section className="space-y-3">
                    <div>
                      <h4 className="text-sm font-semibold text-foreground">物品清单</h4>
                      <p className="mt-1 text-xs text-muted-foreground">根据分类收纳默认物品，勾选后可填写数量。</p>
                    </div>
                    <DormItemSelector
                      categories={SITE_INVENTORY_CATEGORY_DATA}
                      value={siteInventorySelectorValue}
                      onChange={handleInventorySelectorChange}
                    />
                  </section>
                </TabsContent>

                <TabsContent value="contract" className="space-y-6 pb-6">
                  <section className="space-y-4">
                    <h4 className="text-sm font-semibold text-foreground">合同信息</h4>
                    <div className={RESPONSIVE_FIELD_GRID_CLASS}>
                      <div>
                        <Label className="text-xs font-medium text-muted-foreground">甲方</Label>
                        <Input className="mt-1" value={siteContractForm.partyA} onChange={(event) => setSiteContractForm((prev) => ({ ...prev, partyA: event.target.value }))} />
                      </div>
                      <div>
                        <Label className="text-xs font-medium text-muted-foreground">乙方</Label>
                        <Input className="mt-1" value={siteContractForm.partyB} onChange={(event) => setSiteContractForm((prev) => ({ ...prev, partyB: event.target.value }))} />
                      </div>
                      <div>
                        <Label className="text-xs font-medium text-muted-foreground">甲方经办人</Label>
                        <Input className="mt-1" value={siteContractForm.agentA} onChange={(event) => setSiteContractForm((prev) => ({ ...prev, agentA: event.target.value }))} />
                      </div>
                      <div>
                        <Label className="text-xs font-medium text-muted-foreground">乙方经办人</Label>
                        <Input className="mt-1" value={siteContractForm.agentB} onChange={(event) => setSiteContractForm((prev) => ({ ...prev, agentB: event.target.value }))} />
                      </div>
                      <div>
                        <Label className="text-xs font-medium text-muted-foreground">签署日期</Label>
                        <Input type="date" className="mt-1" value={siteContractForm.signingDate} onChange={(event) => setSiteContractForm((prev) => ({ ...prev, signingDate: event.target.value }))} />
                      </div>
                      <div>
                        <Label className="text-xs font-medium text-muted-foreground">合同开始日期</Label>
                        <Input type="date" className="mt-1" value={siteContractForm.contractStartDate} onChange={(event) => setSiteContractForm((prev) => ({ ...prev, contractStartDate: event.target.value }))} />
                      </div>
                      <div>
                        <Label className="text-xs font-medium text-muted-foreground">合同结束日期</Label>
                        <Input type="date" className="mt-1" value={siteContractForm.contractEndDate} onChange={(event) => setSiteContractForm((prev) => ({ ...prev, contractEndDate: event.target.value }))} />
                      </div>
                      <div>
                        <Label className="text-xs font-medium text-muted-foreground">缴费周期</Label>
                        <Select
                          disabled={paymentCycleOptions.length === 0}
                          value={siteContractForm.paymentCycle || undefined}
                          onValueChange={(value: PaymentCycle) => setSiteContractForm((prev) => ({ ...prev, paymentCycle: value }))}
                        >
                          <SelectTrigger className="mt-1">
                            <SelectValue placeholder="请选择" />
                          </SelectTrigger>
                          <SelectContent>
                            {paymentCycleOptions.map((cycle) => (
                              <SelectItem key={cycle} value={cycle}>
                                {PAYMENT_CYCLE_LABELS[cycle]}
                              </SelectItem>
                            ))}
                          </SelectContent>
                        </Select>
                        {paymentCycleOptions.length === 0 && <p className="mt-1 text-[11px] text-muted-foreground">请先在“扣费项目”中勾选月付、季付等周期项</p>}
                      </div>
                      <div>
                        <Label className="text-xs font-medium text-muted-foreground">最近一次付款时间</Label>
                        <Input type="date" className="mt-1" value={siteContractForm.lastPaymentDate} onChange={(event) => setSiteContractForm((prev) => ({ ...prev, lastPaymentDate: event.target.value }))} />
                      </div>
                      <div>
                        <Label className="text-xs font-medium text-muted-foreground">下一次付款时间</Label>
                        <Input type="date" className="mt-1" value={siteContractForm.nextPaymentDate} readOnly placeholder="根据缴费周期自动计算" />
                      </div>
                      <div className="sm:col-span-2 space-y-3 rounded-lg border bg-muted/30 px-3 py-3">
                        <label className="flex items-start gap-3 text-sm">
                          <Checkbox
                            checked={siteContractForm.paymentReminderEnabled ?? false}
                            onCheckedChange={(checked) =>
                              setSiteContractForm((prev) => ({ ...prev, paymentReminderEnabled: Boolean(checked) }))
                            }
                          />
                          <div className="space-y-1">
                            <p className="font-medium text-foreground">缴费提醒</p>
                            <p className="text-xs text-muted-foreground">开启后将在距离下一次缴费 30/20/10 天时自动添加备忘，若距离超过 30 天则不提醒。</p>
                          </div>
                        </label>
                        <label className="flex items-start gap-3 text-sm">
                          <Checkbox
                            checked={siteContractForm.contractReminderEnabled ?? false}
                            onCheckedChange={(checked) =>
                              setSiteContractForm((prev) => ({ ...prev, contractReminderEnabled: Boolean(checked) }))
                            }
                          />
                          <div className="space-y-1">
                            <p className="font-medium text-foreground">合同签署提醒</p>
                            <p className="text-xs text-muted-foreground">根据合同开始/结束日期，在距离目标日期 30/20/10 天内自动添加提示。</p>
                          </div>
                        </label>
                      </div>
                    </div>
                  </section>
                  <section className="space-y-4">
                    <Label className="text-xs font-medium text-muted-foreground">合同备注</Label>
                    <Textarea
                      rows={4}
                      placeholder="补充付款节点、费用分摊等信息"
                      value={siteContractForm.notes ?? ""}
                      onChange={(event) => setSiteContractForm((prev) => ({ ...prev, notes: event.target.value }))}
                    />
                  </section>
                </TabsContent>

                <TabsContent value="memo" className="space-y-6 pb-6">
                  {editingSite ? (
                    <>
                      <section className="space-y-3">
                        <h4 className="text-sm font-semibold text-foreground">新增备忘</h4>
                        <div className={RESPONSIVE_FIELD_GRID_CLASS}>
                          <div>
                            <Label className="text-xs font-medium text-muted-foreground">开始时间 *</Label>
                            <Input
                              type="datetime-local"
                              className="mt-1"
                              value={`${memoForm.startDate}T${memoForm.startTime}`}
                              onChange={(event) => {
                                const value = event.target.value;
                                const [date, time] = value.split("T");
                                setMemoForm((prev) => ({ ...prev, startDate: date || "", startTime: time || "" }));
                              }}
                            />
                          </div>
                          <div>
                            <Label className="text-xs font-medium text-muted-foreground">结束时间</Label>
                            <Input
                              type="datetime-local"
                              className="mt-1"
                              value={memoForm.endDate && memoForm.endTime ? `${memoForm.endDate}T${memoForm.endTime}` : ""}
                              disabled={memoForm.recurrence && memoForm.recurrence !== "none"}
                              onChange={(event) => {
                                const value = event.target.value;
                                const [date, time] = value.split("T");
                                setMemoForm((prev) => ({ ...prev, endDate: date || "", endTime: time || "" }));
                              }}
                            />
                            {memoForm.recurrence && memoForm.recurrence !== "none" && (
                              <p className="mt-1 text-[11px] text-muted-foreground">循环提醒时无需设置结束时间</p>
                            )}
                          </div>
                          <div>
                        <Label className="text-xs font-medium text-muted-foreground">紧急程度</Label>
                        <RadioGroup
                          className="mt-2 flex flex-wrap gap-4"
                          value={memoForm.priority}
                          onValueChange={(value: SiteMemoPriority) => setMemoForm((prev) => ({ ...prev, priority: value }))}
                        >
                          <label className="flex items-center gap-2 text-xs text-foreground">
                            <RadioGroupItem value="low" /> 普通
                          </label>
                          <label className="flex items-center gap-2 text-xs text-foreground">
                            <RadioGroupItem value="normal" /> 重要
                          </label>
                          <label className="flex items-center gap-2 text-xs text-foreground">
                            <RadioGroupItem value="urgent" /> 紧急
                          </label>
                        </RadioGroup>
                      </div>
                          <div>
                            <Label className="text-xs font-medium text-muted-foreground">循环周期</Label>
                            <Select
                              value={memoForm.recurrence}
                              onValueChange={(value: SiteMemoRecurrence) =>
                                setMemoForm((prev) => ({ ...prev, recurrence: value, endDate: value !== "none" ? "" : prev.endDate, endTime: value !== "none" ? "" : prev.endTime }))
                              }
                              disabled={Boolean(memoForm.endDate)}
                            >
                              <SelectTrigger className="mt-1">
                                <SelectValue placeholder="请选择" />
                              </SelectTrigger>
                              <SelectContent>
                                {Object.entries(SITE_MEMO_RECURRENCE_LABELS).map(([key, label]) => (
                                  <SelectItem key={key} value={key}>
                                    {label}
                                  </SelectItem>
                                ))}
                              </SelectContent>
                            </Select>
                            {memoForm.endDate && (
                              <p className="mt-1 text-[11px] text-muted-foreground">结束时间已设置，如需循环请先清除结束时间</p>
                            )}
                          </div>
                        </div>
                        <div>
                          <Label className="text-xs font-medium text-muted-foreground">待办事项 *</Label>
                          <Textarea
                            className="mt-1"
                            rows={3}
                            placeholder="示例：周五前完成消防巡检"
                            value={memoForm.content}
                            onChange={(event) => setMemoForm((prev) => ({ ...prev, content: event.target.value }))}
                          />
                        </div>
                        <div className="flex justify-end">
                          {/* P7.1：新增备忘为修改宿舍信息，需 dormitory.edit 权限 */}
                          <RequirePermission resource="dormitory" action="edit">
                            <Button size="sm" variant="secondary" onClick={handleAddSiteMemo}>
                              添加备忘
                            </Button>
                          </RequirePermission>
                        </div>
                      </section>
                      <section className="space-y-3">
                        <h4 className="text-sm font-semibold text-foreground">待办事项</h4>
                        {editingSiteMemoSections.todo.length === 0 ? (
                          <p className="text-xs text-muted-foreground">暂无待办，新增备忘或启用自动提醒后即可在此跟踪。</p>
                        ) : (
                          <div className="rounded-xl border bg-background">
                            <div className="max-h-[220px] overflow-y-auto divide-y">
                              {editingSiteMemoSections.todo.map((memo) => {
                                if (!memo) return null;
                                const display = getMemoDisplayDate(memo);
                                const recurrenceLabel = memo.recurrence && memo.recurrence !== "none" ? SITE_MEMO_RECURRENCE_LABELS[memo.recurrence] : null;
                                const overdue = isMemoExpired(memo);
                                const remainingSource = memo.targetDate || display.date || memo.date || memo.createdAt;
                                const remainingLabel = formatRemainingDaysLabel(remainingSource);
                                return (
                                  <div key={memo.id} className="flex flex-wrap items-center gap-2 px-3 py-2 text-xs">
                                    <span className="font-medium text-muted-foreground">
                                      {display.date || memo.date} {display.time || memo.time || "--:--"}
                                    </span>
                                    <span className={`rounded-full border px-2 py-0.5 text-[11px] ${PRIORITY_BADGE_CLASS[memo.priority]}`}>
                                      {PRIORITY_LABELS[memo.priority]}
                                    </span>
                                    <span className="rounded-full border border-dashed px-2 py-0.5 text-[10px] text-muted-foreground">{remainingLabel}</span>
                                    {recurrenceLabel && (
                                      <span className="rounded-full border border-dashed px-2 py-0.5 text-[10px] text-muted-foreground">{recurrenceLabel}</span>
                                    )}
                                    {overdue && <span className="rounded-full border border-destructive/50 px-2 py-0.5 text-[10px] text-destructive">已延期</span>}
                                    <span className={`flex-1 truncate text-sm ${MEMO_PRIORITY_TEXT_CLASS[memo.priority]}`}>{memo.content}</span>
                                    <div className="flex items-center gap-1">
                                      {/* P7.1：备忘操作属修改宿舍信息，需 dormitory.edit 权限 */}
                                      <RequirePermission resource="dormitory" action="edit">
                                        <Button
                                          variant="outline"
                                          size="sm"
                                          className="h-6 px-2 text-[11px]"
                                          onClick={() => handleToggleSiteMemoCompletion(editingSite.id, memo.id, true)}
                                        >
                                          完成
                                        </Button>
                                        <Button variant="ghost" size="icon" className="h-6 w-6 text-muted-foreground" onClick={() => handleRemoveSiteMemo(editingSite.id, memo.id)}>
                                          <X className="h-3.5 w-3.5" />
                                        </Button>
                                      </RequirePermission>
                                    </div>
                                  </div>
                                );
                              })}
                            </div>
                          </div>
                        )}
                      </section>
                      <section className="space-y-3">
                        <h4 className="text-sm font-semibold text-foreground">已完成事项</h4>
                        {editingSiteMemoSections.completed.length === 0 ? (
                          <p className="text-xs text-muted-foreground">暂无完成记录。</p>
                        ) : (
                          <div className="rounded-xl border border-dashed bg-muted/30">
                            <div className="max-h-[220px] overflow-y-auto divide-y">
                              {editingSiteMemoSections.completed.map((memo) => {
                                if (!memo) return null;
                                const completedAtLabel = memo.completedAt ? formatDateLabel(memo.completedAt) : formatDateLabel(memo.date || memo.targetDate || memo.createdAt);
                                return (
                                  <div key={memo.id} className="flex flex-wrap items-center gap-2 px-3 py-2 text-xs text-muted-foreground">
                                    <span className="font-medium">完成于 {completedAtLabel}</span>
                                    <span className={`rounded-full border px-2 py-0.5 text-[11px] ${PRIORITY_BADGE_CLASS[memo.priority]}`}>
                                      {PRIORITY_LABELS[memo.priority]}
                                    </span>
                                    <span className="flex-1 truncate text-[13px] text-foreground/80">{memo.content}</span>
                                    <div className="flex items-center gap-1">
                                      {/* P7.1：备忘操作属修改宿舍信息，需 dormitory.edit 权限 */}
                                      <RequirePermission resource="dormitory" action="edit">
                                        <Button
                                          variant="outline"
                                          size="sm"
                                          className="h-6 px-2 text-[11px]"
                                          onClick={() => handleToggleSiteMemoCompletion(editingSite.id, memo.id, false)}
                                        >
                                          撤销
                                        </Button>
                                        <Button variant="ghost" size="icon" className="h-6 w-6 text-muted-foreground" onClick={() => handleRemoveSiteMemo(editingSite.id, memo.id)}>
                                          <X className="h-3.5 w-3.5" />
                                        </Button>
                                      </RequirePermission>
                                    </div>
                                  </div>
                                );
                              })}
                            </div>
                          </div>
                        )}
                      </section>
                    </>
                  ) : (
                    <Card className="border-dashed bg-muted/20">
                      <CardContent className="py-8 text-center text-sm text-muted-foreground">请先保存地点，随后即可在此记录巡检备忘。</CardContent>
                    </Card>
                  )}
                </TabsContent>
              </div>
            </ScrollArea>
          </Tabs>
          <DialogFooter className="border-t bg-background px-6 py-4 flex flex-col gap-2 sm:flex-row sm:justify-end shrink-0">
            <Button variant="outline" onClick={() => setSiteDialogOpen(false)} className="sm:min-w-[96px]">
              取消
            </Button>
            {/* P7.1：保存宿舍地点需 dormitory.edit 权限 */}
            <RequirePermission resource="dormitory" action="edit">
              <Button onClick={handleSaveSite} className="sm:min-w-[96px]">
                保存
              </Button>
            </RequirePermission>
          </DialogFooter>
        </DialogContent>
      </Dialog>
      {/* 房间表单 */}
      <Dialog
        open={roomDialogOpen}
        onOpenChange={(open) => {
          setRoomDialogOpen(open);
          if (!open) {
            setRoomForm({ ...initialRoomForm });
            setEditingRoomId(null);
            setRoomDialogTab("detail");
            setRoomChargeOverrides({});
            setRoomChargeRecords({});
          }
        }}
      >
        <DialogContent className={RESPONSIVE_DIALOG_CLASS}>
          <DialogHeader className="px-6 pt-6">
            <DialogTitle className="flex items-center gap-2">
              <Home className="h-4 w-4" />
              {editingRoomId ? "房间详情" : "新增房间"}
            </DialogTitle>
            <p className="text-sm text-muted-foreground">配置床位数量后会自动生成床位编号，费用字段均支持留空。</p>
          </DialogHeader>
          <div className="flex flex-wrap items-center justify-end gap-4 px-6 pb-2 text-sm text-muted-foreground">
            <div className="flex items-center gap-2">
              <span>维护</span>
              <Switch
                checked={roomForm.status === "维护中"}
                onCheckedChange={(checked) =>
                  setRoomForm((prev) => ({
                    ...prev,
                    status: checked ? "维护中" : "",
                  }))
                }
              />
            </div>
            <div className="flex items-center gap-2">
              <span className="text-xs text-muted-foreground">宿舍类型</span>
              <div className="inline-flex overflow-hidden rounded-full border bg-muted/30 p-0.5 text-xs">
                {(["company", "personal"] as const).map((mode) => (
                  <button
                    key={mode}
                    type="button"
                    className={cn(
                      "px-3 py-1 transition-colors",
                      roomForm.cost_bearing_mode === mode
                        ? "bg-primary text-primary-foreground shadow"
                        : "text-muted-foreground",
                    )}
                    onClick={() =>
                      setRoomForm((prev) => ({
                        ...prev,
                        cost_bearing_mode: mode,
                      }))
                    }
                  >
                    {mode === "company" ? "公司宿舍" : "个人宿舍"}
                  </button>
                ))}
              </div>
            </div>
          </div>
          <ScrollArea className={DIALOG_SCROLL_CLASS}>
            {editingRoomId ? (
              <div className="px-6 pb-6">
                <Tabs value={roomDialogTab} onValueChange={(value) => setRoomDialogTab(value as "detail" | "records" | "history")} className="space-y-4">
                  <TabsList className="w-full justify-start overflow-x-auto rounded-lg border bg-muted/30 p-1">
                    <TabsTrigger value="detail" className="flex-1 whitespace-nowrap">
                      房间详情
                    </TabsTrigger>
                    <TabsTrigger value="records" className="flex-1 whitespace-nowrap">
                      数据记录
                    </TabsTrigger>
                    <TabsTrigger value="history" className="flex-1 whitespace-nowrap">
                      历史记录
                    </TabsTrigger>
                  </TabsList>
                  <TabsContent value="detail">
                    <div className="space-y-6 pt-2 pb-6">{renderRoomDetailSections()}</div>
                  </TabsContent>
                  <TabsContent value="records">
                    <div className="space-y-4 pt-2 pb-6">
                      {renderRoomRecordSections(
                        activeRoomChargeItems,
                        roomChargeRecords,
                        handleRoomRecordValueChange,
                        roomRecordCycleOptions,
                        roomForm.cost_bearing_mode as ShareMode,
                        roomForm.company_name,
                      )}
                    </div>
                  </TabsContent>
                  <TabsContent value="history">
                    <div className="space-y-4 pt-2 pb-6">{renderRoomHistoryTimeline()}</div>
                  </TabsContent>
                </Tabs>
              </div>
            ) : (
              <div className="space-y-6 px-6 pb-8">
                {renderRoomDetailSections()}
                <section className="space-y-4">
                  {renderRoomRecordSections(
                    activeRoomChargeItems,
                    roomChargeRecords,
                    handleRoomRecordValueChange,
                    roomRecordCycleOptions,
                    roomForm.cost_bearing_mode as ShareMode,
                    roomForm.company_name,
                  )}
                </section>
              </div>
            )}
          </ScrollArea>
          <DialogFooter className="border-t bg-background px-6 py-4">
            {editingRoomId ? (
              <div className="flex w-full flex-col gap-2 sm:flex-row sm:justify-end">
                <div className="flex flex-wrap justify-end gap-2">
                  <Button
                    variant="outline"
                    onClick={() => handleRoomNavigate("prev")}
                    disabled={!roomNavigationState.hasPrev}
                  >
                    上一条
                  </Button>
                  <Button
                    variant="outline"
                    onClick={() => handleRoomNavigate("next")}
                    disabled={!roomNavigationState.hasNext}
                  >
                    下一条
                  </Button>
                  <Button className="bg-black text-white hover:bg-black/90" onClick={() => handleSaveRoom({ keepOpen: true })}>
                    应用
                  </Button>
                </div>
              </div>
            ) : (
              <div className="flex w-full flex-col gap-2 sm:flex-row sm:justify-end">
                <div className="flex flex-wrap justify-end gap-2">
                  <Button variant="outline" onClick={() => setRoomDialogOpen(false)}>
                    取消
                  </Button>
                  {/* P7.1：保存房间需 dormitory.edit 权限 */}
                  <RequirePermission resource="dormitory" action="edit">
                    <Button onClick={() => handleSaveRoom()}>保存</Button>
                  </RequirePermission>
                </div>
              </div>
            )}
          </DialogFooter>
      </DialogContent>
    </Dialog>

      <Dialog open={roomImportDialogOpen} onOpenChange={handleRoomImportDialogToggle}>
        <DialogContent className={DIALOG_SIZES.lg}>
          <DialogHeader>
            <DialogTitle className="flex items-center gap-2">
              <Upload className="h-4 w-4" />
              导入房间
            </DialogTitle>
            <DialogDescription>下载模板填表后上传，系统会自动匹配宿舍地点与楼栋。</DialogDescription>
          </DialogHeader>
          <div className="grid gap-6 py-4 md:grid-cols-[1.2fr,1fr]">
            <section className="space-y-4 text-sm text-muted-foreground">
              <div className="rounded-lg border bg-muted/20 px-4 py-3">
                <p className="font-medium text-foreground">导入说明</p>
                <ul className="mt-2 list-disc pl-4 text-xs leading-relaxed">
                  <li>宿舍地点、楼栋名称、房号为必填项，缺失将自动跳过。</li>
                  <li>若地点/楼栋不存在，系统会自动创建默认记录。</li>
                  <li>费用与床位信息可后续在房间详情中补齐。</li>
                </ul>
              </div>
              {roomImportResult && (
                <div className="rounded-md border bg-muted/30 px-3 py-2 text-xs text-muted-foreground">
                  导入统计：成功 {roomImportResult.inserted} 条 · 跳过 {roomImportResult.skipped} 条
                </div>
              )}
            </section>
            <section className="space-y-3">
              <div className="space-y-3">
                <Label className="text-xs font-medium text-muted-foreground">上传已填写的模板 *</Label>
                <Input ref={roomImportFileInputRef} type="file" accept=".xls,.xlsx" onChange={handleRoomImportFileChange} disabled={roomImporting} />
                {roomImportFile && <p className="text-xs text-muted-foreground">已选择：{roomImportFile.name}</p>}
                {roomImportError && <p className="text-xs text-destructive">{roomImportError}</p>}
                <p className="text-xs text-muted-foreground">支持 .xls/.xlsx 文件，建议一次导入不超过 500 条。</p>
                {/* P7.1：下载模板为查看类操作，需 dormitory.view 权限 */}
                <RequirePermission resource="dormitory" action="view">
                  <Button className="w-full bg-black text-white hover:bg-black/90" onClick={handleDownloadRoomTemplate}>
                    <Download className="h-4 w-4" /> 下载导入模板
                  </Button>
                </RequirePermission>
              </div>
            </section>
          </div>
          <DialogFooter className="flex flex-col gap-2 sm:flex-row sm:justify-end">
            <Button variant="outline" onClick={() => handleRoomImportDialogToggle(false)} disabled={roomImporting}>
              取消
            </Button>
            {/* P7.1：批量导入房间为新增数据，需 dormitory.create 权限 */}
            <RequirePermission resource="dormitory" action="create">
              <Button onClick={handleImportRooms} disabled={roomImporting}>
                {roomImporting ? "导入中..." : "开始导入"}
              </Button>
            </RequirePermission>
          </DialogFooter>
      </DialogContent>
    </Dialog>

    <Dialog open={meterImportDialogOpen} onOpenChange={handleMeterImportDialogToggle}>
        <DialogContent className={DIALOG_SIZES.lg}>
          <DialogHeader>
            <DialogTitle className="flex items-center gap-2">
              <Upload className="h-4 w-4" />
              导入抄表记录
            </DialogTitle>
            <DialogDescription>下载模板填写度数与时间后再导入，系统会自动匹配房间与费用。</DialogDescription>
          </DialogHeader>
          <div className="grid gap-6 py-4 md:grid-cols-[1.2fr,1fr]">
            <section className="space-y-4 text-sm text-muted-foreground">
              <div className="rounded-lg border bg-muted/20 px-4 py-3">
                <p className="font-medium text-foreground">导入说明</p>
                <ul className="mt-2 list-disc pl-4 text-xs leading-relaxed">
                  <li>房号与楼栋为必填项，用于匹配宿舍房间。</li>
                  <li>电表/水表起止度仅支持整数或小数，气费可留空。</li>
                  <li>未填写抄表日期时将使用当前日期。</li>
                </ul>
              </div>
              {/* P7.1：下载模板为查看类操作，需 dormitory.view 权限 */}
              <RequirePermission resource="dormitory" action="view">
                <Button variant="outline" className="w-full gap-2" onClick={handleDownloadMeterTemplate}>
                  <Download className="h-4 w-4" /> 下载导入模板
                </Button>
              </RequirePermission>
              {meterImportResult && (
                <div className="rounded-md border bg-muted/30 px-3 py-2 text-xs text-muted-foreground">
                  导入统计：成功 {meterImportResult.inserted} 条 · 跳过 {meterImportResult.skipped} 条
                </div>
              )}
            </section>
            <section className="space-y-3 text-sm">
              <div className="space-y-2">
                <Label className="text-xs font-medium text-muted-foreground">上传模板 *</Label>
                <Input ref={meterImportFileInputRef} type="file" accept=".xls,.xlsx" onChange={handleMeterImportFileChange} disabled={meterImporting} />
                {meterImportFile && <p className="text-xs text-muted-foreground">已选择：{meterImportFile.name}</p>}
                {meterImportError && <p className="text-xs text-destructive">{meterImportError}</p>}
                <p className="text-xs text-muted-foreground">建议单次导入不超过 300 条记录。</p>
              </div>
            </section>
          </div>
          <DialogFooter className="flex flex-col gap-2 sm:flex-row sm:justify-end">
            <Button variant="outline" onClick={() => handleMeterImportDialogToggle(false)} disabled={meterImporting}>
              取消
            </Button>
            {/* P7.1：批量导入抄表为新增数据，需 dormitory.create 权限 */}
            <RequirePermission resource="dormitory" action="create">
              <Button onClick={handleImportMeterRecords} disabled={meterImporting}>
                {meterImporting ? "导入中..." : "开始导入"}
              </Button>
            </RequirePermission>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Dialog
        open={meterDialogOpen}
        onOpenChange={(open) => {
          setMeterDialogOpen(open);
          if (!open) {
            handleResetMeterForm();
          }
        }}
      >
        <DialogContent className={RESPONSIVE_DIALOG_CLASS}>
          <DialogHeader className="px-6 pt-6">
            <div>
              <DialogTitle className="flex items-center gap-2 text-lg">
                <Gauge className="h-4 w-4" />
                {meterFormMode === "edit" ? "编辑抄表记录" : "抄表录入"}
              </DialogTitle>
              <DialogDescription className="leading-relaxed">
                选择房间后录入水电气度数与账期，系统会自动计算费用；若房间设置为个人宿舍，会自动按入住人员分摊。
              </DialogDescription>
            </div>
          </DialogHeader>
          <ScrollArea className={DIALOG_SCROLL_CLASS}>
            <div className="space-y-6 px-6 pb-6">
              <section className="space-y-3">
                <h4 className="text-sm font-semibold text-foreground">房间与抄表信息</h4>
                <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-5">
                  <div>
                    <Label className="text-xs font-medium text-muted-foreground">地点 *</Label>
                    <Select
                      value={meterFormSiteId || SELECT_EMPTY_VALUE}
                      onValueChange={(value) => {
                        if (value === SELECT_EMPTY_VALUE) {
                          setMeterFormSiteId("");
                          setMeterFormBuildingId("all");
                          handleMeterFormChange("room_id", "");
                          return;
                        }
                        setMeterFormSiteId(value);
                        setMeterFormBuildingId("all");
                        handleMeterFormChange("room_id", "");
                      }}
                      disabled={sites.length === 0}
                    >
                      <SelectTrigger className="mt-1">
                        <SelectValue placeholder={sites.length === 0 ? "暂无地点" : "请选择地点"} />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectItem value={SELECT_EMPTY_VALUE} disabled>
                          请选择地点
                        </SelectItem>
                        {sites.map((site) => (
                          <SelectItem key={site.id} value={String(site.id)}>
                            {site.name}
                          </SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
                  </div>
                  <div>
                    <Label className="text-xs font-medium text-muted-foreground">楼栋 *</Label>
                    <Select
                      value={meterFormBuildingId === "all" ? "all" : String(meterFormBuildingId)}
                      disabled={!meterFormSiteId}
                      onValueChange={(value) => {
                        if (value === "all") {
                          setMeterFormBuildingId("all");
                        } else {
                          const numeric = Number(value);
                          setMeterFormBuildingId(Number.isFinite(numeric) ? numeric : "all");
                        }
                        handleMeterFormChange("room_id", "");
                      }}
                    >
                      <SelectTrigger className="mt-1">
                        <SelectValue placeholder={meterFormSiteId ? "请选择楼栋" : "请先选择地点"} />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectItem value="all" disabled>
                          请选择楼栋
                        </SelectItem>
                        {meterFormBuildingOptions.map((building) => (
                          <SelectItem key={building.id} value={String(building.id)}>
                            {building.name || `楼栋${building.id}`}
                          </SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
                  </div>
                  <div>
                    <Label className="text-xs font-medium text-muted-foreground">房号 *</Label>
                    <Select
                      value={meterForm.room_id || SELECT_EMPTY_VALUE}
                      disabled={meterFormBuildingId === "all"}
                      onValueChange={(value) => handleMeterFormChange("room_id", value)}
                    >
                      <SelectTrigger className="mt-1">
                        <SelectValue placeholder={meterFormBuildingId === "all" ? "请先选择楼栋" : "请选择房间"} />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectItem value={SELECT_EMPTY_VALUE} disabled>
                          {meterFormBuildingId === "all" ? "请选择楼栋后再选择房号" : "请选择房间"}
                        </SelectItem>
                        {meterFormRoomOptions.map((room) => (
                          <SelectItem key={room.id} value={String(room.id)}>
                            {room.room_number || `房间${room.id}`}（{buildingById.get(room.building_id)?.name || "未分配楼栋"}）
                          </SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
                    {selectedMeterRoom && (
                      <p className="mt-1 text-[11px] text-muted-foreground">入住人员：{meterOccupantDisplay}</p>
                    )}
                  </div>
                  <div>
                    <Label className="text-xs font-medium text-muted-foreground">抄表日期 *</Label>
                    <Input
                      type="date"
                      className="mt-1"
                      value={meterForm.meter_date}
                      onChange={(event) => handleMeterFormChange("meter_date", event.target.value)}
                    />
                  </div>
                  <div>
                    <Label className="text-xs font-medium text-muted-foreground">抄表人</Label>
                    <Input className="mt-1" value={meterForm.inspector} onChange={(event) => handleMeterFormChange("inspector", event.target.value)} />
                  </div>
                </div>
              </section>
              <section className="space-y-3">
                <h4 className="text-sm font-semibold text-foreground">账期管理</h4>
                <div className="grid grid-cols-1 gap-4 md:grid-cols-3">
                  <div>
                    <Label className="text-xs font-medium text-muted-foreground">账期月份</Label>
                    <Input
                      type="month"
                      className="mt-1"
                      value={meterForm.billing_month}
                      onChange={(event) => handleMeterFormChange("billing_month", event.target.value)}
                    />
                  </div>
                  <div>
                    <Label className="text-xs font-medium text-muted-foreground">账单起始日期 *</Label>
                    <Input
                      type="date"
                      className="mt-1"
                      value={meterForm.billing_start}
                      onChange={(event) => handleMeterFormChange("billing_start", event.target.value)}
                    />
                  </div>
                  <div>
                    <Label className="text-xs font-medium text-muted-foreground">账单结束日期 *</Label>
                    <Input
                      type="date"
                      className="mt-1"
                      value={meterForm.billing_end}
                      onChange={(event) => handleMeterFormChange("billing_end", event.target.value)}
                    />
                  </div>
                </div>
              </section>
              <section className="space-y-3">
                <h4 className="text-sm font-semibold text-foreground">度数与费用</h4>
                <div className="space-y-4">
                  {(["electric", "water", "gas"] as const)
                    .filter((key) => meterChargeSettingMap.has(key))
                    .map((key) => {
                      const definition = getChargeDefinition(key);
                      const label = definition?.label || LEGACY_CHARGE_LABELS[key as LegacyChargeKey] || key;
                      const startField =
                        key === "electric" ? "electric_start" : key === "water" ? "water_start" : "gas_start";
                      const endField = key === "electric" ? "electric_end" : key === "water" ? "water_end" : "gas_end";
                      const usageValue =
                        key === "electric"
                          ? meterFormPreview.electricUsage
                          : key === "water"
                            ? meterFormPreview.waterUsage
                            : meterFormPreview.gasUsage;
                      const amountValue =
                        key === "electric"
                          ? meterFormPreview.electricFee
                          : key === "water"
                            ? meterFormPreview.waterFee
                            : meterFormPreview.gasFee;
                      const usageDisplay = usageValue != null ? String(usageValue) : "";
                      const amountDisplay = amountValue != null ? amountValue.toFixed(2) : "";
                      const startKey = startField as keyof MeterFormState;
                      const endKey = endField as keyof MeterFormState;
                      const startValue = meterForm[startKey] as string;
                      const endValue = meterForm[endKey] as string;
                      return (
                        <div key={key} className="grid gap-3 sm:grid-cols-4">
                          <div>
                            <Label className="text-xs font-medium text-muted-foreground">
                              {label}起度
                            </Label>
                            <Input
                              type="number"
                              step="0.01"
                              className="mt-1"
                              value={startValue}
                              onChange={(event) => handleMeterFormChange(startKey, event.target.value)}
                            />
                          </div>
                          <div>
                            <Label className="text-xs font-medium text-muted-foreground">
                              {label}止度
                            </Label>
                            <Input
                              type="number"
                              step="0.01"
                              className="mt-1"
                              value={endValue}
                              onChange={(event) => handleMeterFormChange(endKey, event.target.value)}
                            />
                          </div>
                          <div>
                            <Label className="text-xs font-medium text-muted-foreground">用量</Label>
                            <Input className="mt-1" readOnly value={usageDisplay} placeholder="自动计算" />
                          </div>
                          <div>
                            <Label className="text-xs font-medium text-muted-foreground">
                              {label}费用
                              {definition?.unitLabel ? <span className="text-[11px] text-muted-foreground">（{definition.unitLabel}）</span> : null}
                            </Label>
                            <Input className="mt-1" readOnly value={amountDisplay} placeholder="自动计算" />
                          </div>
                        </div>
                      );
                    })}
                </div>
              </section>
              {supplementalChargesForCurrentRoom.length > 0 && (
                <section className="space-y-3">
                  <div className="flex items-center justify-between">
                    <h4 className="text-sm font-semibold text-foreground">附加费用</h4>
                    <p className="text-xs text-muted-foreground">自动计算为主，可按需覆盖具体金额</p>
                  </div>
                  <div className="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-4">
                    {supplementalChargesForCurrentRoom.map((item) => {
                      const autoValue = autoChargeAmounts.get(item.key) ?? null;
                      const resolvedAmount = getResolvedSupplementalAmount(item.key);
                      const manualValue = meterExtraChargeInputs[item.key];
                      const isWaterSupply = item.key === "water_supply";
                      const enabled = meterExtraChargeToggles[item.key] !== false;
                      const inputValue =
                        manualValue !== undefined
                          ? manualValue
                          : enabled && resolvedAmount != null
                            ? String(resolvedAmount)
                            : "";
                      return (
                        <div key={item.key} className="space-y-2 rounded-md border border-dashed bg-background/60 p-3 text-sm">
                          <div className="flex items-center justify-between text-xs">
                            <span className="font-medium text-foreground">{item.label}</span>
                            <span className="text-muted-foreground">{item.unitLabel || "元"}</span>
                          </div>
                          {isWaterSupply ? (
                            <div className="flex items-center gap-2 text-[11px] text-muted-foreground">
                              <Checkbox
                                checked={enabled}
                                onCheckedChange={(checked) => {
                                  setMeterExtraChargeToggles((prev) => ({ ...prev, [item.key]: checked === true }));
                                  if (checked !== true) {
                                    setMeterExtraChargeInputs((prev) => {
                                      if (!(item.key in prev)) {
                                        return prev;
                                      }
                                      const nextState = { ...prev };
                                      delete nextState[item.key];
                                      return nextState;
                                    });
                                  }
                                }}
                              />
                              <span>{enabled ? "已启用二次供水" : "停用二次供水"}</span>
                            </div>
                          ) : (
                            <p className="text-[11px] text-muted-foreground">
                              {autoValue != null ? `自动金额：${formatCurrencyValue(autoValue)}` : "可直接输入具体金额"}
                            </p>
                          )}
                          <Input
                            type="number"
                            inputMode="decimal"
                            step="0.01"
                            placeholder={autoValue != null ? `可覆盖自动金额 ${autoValue}` : `请输入${item.label}`}
                            value={enabled ? inputValue : ""}
                            disabled={isWaterSupply && !enabled}
                            onChange={(event) => {
                              const nextValue = event.target.value;
                              setMeterExtraChargeInputs((prev) => {
                                if (!nextValue.trim()) {
                                  if (!(item.key in prev)) {
                                    return prev;
                                  }
                                  const nextState = { ...prev };
                                  delete nextState[item.key];
                                  return nextState;
                                }
                                return { ...prev, [item.key]: nextValue };
                              });
                            }}
                          />
                        </div>
                      );
                    })}
                  </div>
                </section>
              )}
              {selectedMeterRoom?.cost_bearing_mode === "personal" && (
                <section className="space-y-3">
                  <div className="flex items-center justify-between">
                    <h4 className="text-sm font-semibold text-foreground">成员分摊</h4>
                    <p className="text-xs text-muted-foreground">默认均分，可调整比例与金额</p>
                  </div>
                  {meterMemberEntries.length === 0 ? (
                    <div className="rounded-lg border border-dashed bg-muted/30 px-4 py-6 text-center text-xs text-muted-foreground">
                      当前房间暂无入住人员，无法执行成员计费。
                    </div>
                  ) : meterMemberSharePreview.length === 0 ||
                    MEMBER_SHARE_CHARGE_KEYS.every((key) => resolvedChargeTotals[key] == null) ? (
                    <div className="rounded-lg border border-dashed bg-muted/30 px-4 py-6 text-center text-xs text-muted-foreground">
                      当前账期暂无可分摊费用，请先录入度数或附加费用。
                    </div>
                  ) : (
                    <div className="overflow-x-auto rounded-lg border">
                      <Table className="min-w-[720px] text-sm">
                        <TableHeader>
                          <TableRow>
                            <TableHead className="w-40 text-center">成员</TableHead>
                            <TableHead className="w-32 text-center">分摊比例</TableHead>
                            {MEMBER_SHARE_CHARGE_KEYS.filter((key) => resolvedChargeTotals[key] != null).map((chargeKey) => (
                              <TableHead key={chargeKey} className="text-center">
                                {MEMBER_CHARGE_LABELS[chargeKey]}
                              </TableHead>
                            ))}
                            <TableHead className="text-center">合计</TableHead>
                          </TableRow>
                        </TableHeader>
                        <TableBody>
                          {meterMemberSharePreview.map((row) => (
                            <TableRow key={row.key}>
                              <TableCell className="font-medium">{row.name}</TableCell>
                              <TableCell>
                                <Input
                                  type="number"
                                  inputMode="decimal"
                                  step="0.1"
                                  placeholder="1"
                                  value={meterMemberRatios[row.key] ?? ""}
                                  onChange={(event) => {
                                    const next = event.target.value;
                                    setMeterMemberRatios((prev) => {
                                      if (!next.trim()) {
                                        if (!(row.key in prev)) {
                                          return prev;
                                        }
                                        const nextState = { ...prev };
                                        delete nextState[row.key];
                                        return nextState;
                                      }
                                      return { ...prev, [row.key]: next };
                                    });
                                  }}
                                />
                              </TableCell>
                              {MEMBER_SHARE_CHARGE_KEYS.filter((key) => resolvedChargeTotals[key] != null).map((chargeKey) => {
                                const autoValue = row.autoCharges[chargeKey];
                                const overrideValue = meterMemberChargeOverrides[row.key]?.[chargeKey];
                                const inputValue =
                                  overrideValue !== undefined
                                    ? overrideValue
                                    : autoValue != null
                                      ? String(autoValue)
                                      : "";
                                return (
                                  <TableCell key={`${row.key}-${chargeKey}`} className="text-right">
                                    <Input
                                      type="number"
                                      inputMode="decimal"
                                      step="0.01"
                                      value={inputValue}
                                      onChange={(event) => {
                                        const next = event.target.value;
                                        setMeterMemberChargeOverrides((prev) => {
                                          const current = prev[row.key] ?? {};
                                          if (!next.trim()) {
                                            if (!prev[row.key]) return prev;
                                            const nextCharges = { ...current };
                                            delete nextCharges[chargeKey];
                                            if (Object.keys(nextCharges).length === 0) {
                                              const nextRows = { ...prev };
                                              delete nextRows[row.key];
                                              return nextRows;
                                            }
                                            return { ...prev, [row.key]: nextCharges };
                                          }
                                          return { ...prev, [row.key]: { ...current, [chargeKey]: next } };
                                        });
                                      }}
                                      placeholder="0.00"
                                    />
                                  </TableCell>
                                );
                              })}
                              <TableCell className="text-right font-semibold text-foreground">
                                {row.totalAmount != null ? formatCurrencyValue(row.totalAmount) : "--"}
                              </TableCell>
                            </TableRow>
                          ))}
                        </TableBody>
                      </Table>
                    </div>
                  )}
                </section>
              )}
            </div>
          </ScrollArea>
          <DialogFooter className="flex flex-col gap-2 px-6 pb-6 sm:flex-row sm:justify-end">
            <Button
              variant="outline"
              onClick={() => {
                setMeterDialogOpen(false);
                handleResetMeterForm();
              }}
            >
              取消
            </Button>
            {/* P7.1：保存抄表记录需 dormitory.edit 权限 */}
            <RequirePermission resource="dormitory" action="edit">
              <Button variant="outline" onClick={() => handleSubmitMeterForm({ stayOpen: true })}>
                保存并继续
              </Button>
              <Button onClick={() => handleSubmitMeterForm()}>保存</Button>
            </RequirePermission>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Dialog
        open={roomPrintDialogOpen}
        onOpenChange={(open) => {
          if (!open) {
            closeRoomPrintDialog();
          }
        }}
      >
        <DialogContent className={DIALOG_SIZES.sm}>
          <DialogHeader>
            <DialogTitle className="flex items-center gap-2">
              <Printer className="h-4 w-4" />
              打印设置
            </DialogTitle>
            <DialogDescription>配置标题、水印与排版后生成房间列表 PDF。</DialogDescription>
          </DialogHeader>
          <div className="space-y-4 py-2">
            <div className="space-y-2">
              <Label className="text-sm font-medium">打印标题</Label>
              <Input
                value={roomPrintTitle}
                onChange={(event) => setRoomPrintTitle(event.target.value)}
                placeholder={roomPrintSuggestedTitle || "房间列表打印清单"}
              />
            </div>
            <div className="space-y-2">
              <Label className="text-sm font-medium">水印</Label>
              <Input
                value={roomPrintWatermark}
                onChange={(event) => setRoomPrintWatermark(event.target.value)}
                placeholder="内部资料 请勿外传"
              />
              <p className="text-xs text-muted-foreground">为空时将使用默认水印「内部资料 请勿外传」。</p>
            </div>
            <div className="space-y-2">
              <Label className="text-sm font-medium">排版方向</Label>
              <Select value={roomPrintOrientation} onValueChange={(value) => setRoomPrintOrientation(value as PrintOrientation)}>
                <SelectTrigger>
                  <SelectValue placeholder="自动适配" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="auto">自动适配</SelectItem>
                  <SelectItem value="portrait">纵向（A4）</SelectItem>
                  <SelectItem value="landscape">横向（A4）</SelectItem>
                </SelectContent>
              </Select>
            </div>
            <div className="rounded-md border border-dashed border-muted-foreground/50 bg-muted/30 p-3 text-xs leading-relaxed text-muted-foreground">
              {roomPrintContext.length === 0 ? (
                <p>未选择房间，请在列表中勾选后再尝试打印。</p>
              ) : (
                <>
                  <div>打印房间数：{roomPrintContext.length} 条</div>
                  <div>可见字段：{visibleRoomColumns.length} 列</div>
                  <div>提示：生成的 PDF 将在新窗口打开，可直接打印或保存。</div>
                </>
              )}
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={closeRoomPrintDialog}>
              取消
            </Button>
            {/* P7.1：生成打印文件为查看类操作，需 dormitory.view 权限 */}
            <RequirePermission resource="dormitory" action="view">
              <Button onClick={handleGenerateRoomPrint} disabled={roomPrintContext.length === 0}>
                生成打印文件
              </Button>
            </RequirePermission>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Dialog
        open={meterPrintDialogOpen}
        onOpenChange={(open) => {
          if (!open) {
            closeMeterPrintDialog();
          }
        }}
      >
        <DialogContent className={DIALOG_SIZES.sm}>
          <DialogHeader>
            <DialogTitle className="flex items-center gap-2">
              <Printer className="h-4 w-4" />
              抄表打印设置
            </DialogTitle>
            <DialogDescription>配置标题、水印与排版后生成抄表列表 PDF。</DialogDescription>
          </DialogHeader>
          <div className="space-y-4 py-2">
            <div className="space-y-2">
              <Label className="text-sm font-medium">打印标题</Label>
              <Input value={meterPrintTitle} onChange={(event) => setMeterPrintTitle(event.target.value)} placeholder={meterPrintSuggestedTitle || "抄表记录打印"} />
            </div>
            <div className="space-y-2">
              <Label className="text-sm font-medium">水印</Label>
              <Input value={meterPrintWatermark} onChange={(event) => setMeterPrintWatermark(event.target.value)} placeholder="内部资料 请勿外传" />
            </div>
            <div className="space-y-2">
              <Label className="text-sm font-medium">排版方向</Label>
              <Select value={meterPrintOrientation} onValueChange={(value) => setMeterPrintOrientation(value as PrintOrientation)}>
                <SelectTrigger>
                  <SelectValue placeholder="自动适配" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="auto">自动适配</SelectItem>
                  <SelectItem value="portrait">纵向</SelectItem>
                  <SelectItem value="landscape">横向</SelectItem>
                </SelectContent>
              </Select>
            </div>
            <div className="rounded-md border border-dashed border-muted-foreground/50 bg-muted/30 p-3 text-xs leading-relaxed text-muted-foreground">
              {meterPrintContext.length === 0 ? (
                <p>未选择抄表记录，请返回列表勾选数据后再试。</p>
              ) : (
                <>
                  <div>打印记录数：{meterPrintContext.length} 条</div>
                  <div>可见字段：{visibleMeterColumns.length} 列</div>
                  <div>提示：生成的 PDF 将在新窗口打开，可直接打印或保存。</div>
                </>
              )}
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={closeMeterPrintDialog}>
              取消
            </Button>
            {/* P7.1：生成打印文件为查看类操作，需 dormitory.view 权限 */}
            <RequirePermission resource="dormitory" action="view">
              <Button onClick={handleGenerateMeterPrint} disabled={meterPrintContext.length === 0}>
                生成打印文件
              </Button>
            </RequirePermission>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Dialog
        open={contractPrintDialogOpen}
        onOpenChange={(open) => {
          if (!open) {
            closeContractPrintDialog();
          }
        }}
      >
        <DialogContent className={DIALOG_SIZES.sm}>
          <DialogHeader>
            <DialogTitle className="flex items-center gap-2">
              <Printer className="h-4 w-4" />
              入住列表打印设置
            </DialogTitle>
            <DialogDescription>仅打印当前选择的入住记录，可自定义标题、水印与排版方向。</DialogDescription>
          </DialogHeader>
          <div className="space-y-4 py-2">
            <div className="space-y-2">
              <Label className="text-sm font-medium">打印标题</Label>
              <Input
                value={contractPrintTitle}
                onChange={(event) => setContractPrintTitle(event.target.value)}
                placeholder={contractPrintSuggestedTitle || "入住列表打印清单"}
              />
            </div>
            <div className="space-y-2">
              <Label className="text-sm font-medium">水印</Label>
              <Input
                value={contractPrintWatermark}
                onChange={(event) => setContractPrintWatermark(event.target.value)}
                placeholder="内部资料 请勿外传"
              />
              <p className="text-xs text-muted-foreground">为空将使用默认水印「内部资料 请勿外传」。</p>
            </div>
            <div className="space-y-2">
              <Label className="text-sm font-medium">排版方向</Label>
              <Select value={contractPrintOrientation} onValueChange={(value) => setContractPrintOrientation(value as PrintOrientation)}>
                <SelectTrigger>
                  <SelectValue placeholder="自动适配" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="auto">自动适配</SelectItem>
                  <SelectItem value="portrait">纵向（A4）</SelectItem>
                  <SelectItem value="landscape">横向（A4）</SelectItem>
                </SelectContent>
              </Select>
            </div>
            <div className="rounded-md border border-dashed border-muted-foreground/50 bg-muted/30 p-3 text-xs leading-relaxed text-muted-foreground">
              {contractPrintContext.length === 0 ? (
                <p>未选择入住记录，请在列表中勾选后再尝试打印。</p>
              ) : (
                <>
                  <div>打印记录数：{contractPrintContext.length} 条</div>
                  <div>可见列：{visibleContractColumns.length} 列</div>
                  <div>提示：生成的 PDF 将在新窗口打开，可直接打印或保存。</div>
                </>
              )}
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={closeContractPrintDialog}>
              取消
            </Button>
            {/* P7.1：生成打印文件为查看类操作，需 dormitory.view 权限 */}
            <RequirePermission resource="dormitory" action="view">
              <Button onClick={handleGenerateContractPrint} disabled={contractPrintContext.length === 0}>
                生成打印文件
              </Button>
            </RequirePermission>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Dialog open={contractImportDialogOpen} onOpenChange={handleContractImportDialogToggle}>
        <DialogContent className={DIALOG_SIZES.md}>
          <DialogHeader>
            <DialogTitle className="flex items-center gap-2">
              <Upload className="h-4 w-4" />
              入住导入
            </DialogTitle>
            <DialogDescription>批量录入入住人员信息，系统将按房号与床位自动匹配。</DialogDescription>
          </DialogHeader>
          <div className="space-y-4">
            <p className="text-sm text-muted-foreground">
              提示：模板中“姓名”“房号”“入住日期”为必填字段，床位未填写时将自动匹配空闲床位。
            </p>
            <div className="space-y-3">
              <div className="space-y-2">
                <Label className="text-xs font-medium text-muted-foreground">上传已填写的模板 *</Label>
                <Input ref={contractImportFileInputRef} type="file" accept=".xls,.xlsx" />
                <p className="text-xs text-muted-foreground">支持 .xls / .xlsx 格式文件</p>
              </div>
              {/* P7.1：下载模板为查看类操作，需 dormitory.view 权限 */}
              <RequirePermission resource="dormitory" action="view">
                <Button className="w-full gap-2 bg-black text-white hover:bg-black/90" onClick={handleDownloadContractTemplate}>
                  <Download className="h-4 w-4" /> 下载导入模板
                </Button>
              </RequirePermission>
            </div>
            {contractImportResult && (
              <div className="rounded-md border bg-muted/40 px-3 py-2 text-xs text-muted-foreground">
                <p>导入统计：成功 {contractImportResult.inserted} 条 · 跳过 {contractImportResult.skipped} 条</p>
              </div>
            )}
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => handleContractImportDialogToggle(false)}>
              关闭
            </Button>
            {/* P7.1：批量导入入住记录为创建合同，需 dormitory.create 权限 */}
            <RequirePermission resource="dormitory" action="create">
              <Button onClick={handleImportContracts} disabled={contractImporting}>
                {contractImporting ? "导入中..." : "开始导入"}
              </Button>
            </RequirePermission>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* 入住表单 */}
      <Dialog open={contractDialogOpen} onOpenChange={handleContractDialogToggle}>
        <DialogContent className={RESPONSIVE_DIALOG_CLASS}>
          <DialogHeader className="px-6 pt-6">
            <DialogTitle className="flex items-center gap-2">
              <BedDouble className="h-4 w-4" />
              {contractDialogTitle}
            </DialogTitle>
            <DialogDescription>从在职员工中选择入住人、床位及费用信息，自动同步到房间状态。</DialogDescription>
          </DialogHeader>
                    <ScrollArea className={DIALOG_SCROLL_CLASS}>
            {contractDialogTitle === "入住详情" && editingContractId ? (
              <div className="px-6 pb-6">
                <Tabs value={contractDialogTab} onValueChange={(value) => setContractDialogTab(value as "detail" | "history")} className="space-y-4">
                  <TabsList className="w-full justify-start overflow-x-auto rounded-lg border bg-muted/30 p-1">
                    <TabsTrigger value="detail" className="flex-1 whitespace-nowrap">
                      入住详情
                    </TabsTrigger>
                    <TabsTrigger value="history" className="flex-1 whitespace-nowrap">
                      历史记录
                    </TabsTrigger>
                  </TabsList>
                  <TabsContent value="detail">
                    <div className="space-y-6 pt-2">{renderContractDetailSections()}</div>
                  </TabsContent>
                  <TabsContent value="history">
                    <div className="space-y-4 pt-2">{renderContractHistoryTimeline()}</div>
                  </TabsContent>
                </Tabs>
              </div>
            ) : (
              <div className="space-y-6 px-6 pb-6 pt-4">{renderContractDetailSections()}</div>
            )}
          </ScrollArea>
          <DialogFooter className="border-t bg-background px-6 py-4">
            {editingContractId ? (
              <div className="flex w-full flex-col gap-2 sm:flex-row sm:justify-end">
                <div className="flex flex-wrap justify-end gap-2">
                  <Button
                    variant="outline"
                    onClick={() => handleContractNavigate("prev")}
                    disabled={!contractNavigationState.hasPrev}
                  >
                    上一条
                  </Button>
                  <Button
                    variant="outline"
                    onClick={() => handleContractNavigate("next")}
                    disabled={!contractNavigationState.hasNext}
                  >
                    下一条
                  </Button>
                  {/* P7.1：办理/编辑入住为创建或修改合同，需 dormitory.create/edit 权限 */}
                  <RequirePermission
                    resource="dormitory"
                    action={editingContractId != null ? "edit" : "create"}
                  >
                    <Button
                      className="bg-black text-white hover:bg-black/90"
                      onClick={() => handleSaveContract({ keepOpen: true })}
                      disabled={contractSaving}
                    >
                      {contractSaving ? "应用中..." : "应用"}
                    </Button>
                  </RequirePermission>
                </div>
              </div>
            ) : (
              <div className="flex w-full flex-col gap-2 sm:flex-row sm:justify-end">
                <div className="flex flex-wrap justify-end gap-2">
                  <Button variant="outline" onClick={() => handleContractDialogToggle(false)}>
                    取消
                  </Button>
                  {/* P7.1：办理/编辑入住为创建或修改合同，需 dormitory.create/edit 权限 */}
                  <RequirePermission
                    resource="dormitory"
                    action={editingContractId != null ? "edit" : "create"}
                  >
                    <Button onClick={() => handleSaveContract()} disabled={contractSaving}>
                      {contractSaving ? "保存中..." : "保存"}
                    </Button>
                  </RequirePermission>
                </div>
              </div>
            )}
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Dialog open={checkoutDialogOpen} onOpenChange={setCheckoutDialogOpen}>
        <DialogContent className={RESPONSIVE_DIALOG_CLASS}>
          <DialogHeader className="px-6 pt-6">
            <DialogTitle className="flex items-center gap-2">
              <LogOut className="h-4 w-4" />
              办理退宿
            </DialogTitle>
            <DialogDescription>选择已入住人员，填写水电读数、押金及物品情况，系统会同步更新床位状态。</DialogDescription>
          </DialogHeader>
          <ScrollArea className={DIALOG_SCROLL_CLASS}>
            <div className="space-y-6 px-6 pb-6 pt-4">
              <section className="space-y-4">
                <div className="flex items-center justify-between">
                  <h4 className="text-sm font-semibold text-foreground">退宿信息</h4>
                </div>
                <div className={RESPONSIVE_FIELD_GRID_CLASS}>
                  <div>
                    <Label className="text-xs font-medium text-muted-foreground">选择入住记录 *</Label>
                    <select
                      className="mt-1 w-full rounded-md border border-input bg-background px-3 py-2 text-sm"
                      value={checkoutForm.contract_id}
                      onChange={(event) => {
                        const value = event.target.value;
                        const contract = contractById.get(Number(value));
                        const guaranteeFee = contract?.room?.guarantee_fee;
                        setCheckoutForm((prev) => ({
                          ...prev,
                          contract_id: value,
                          deposit_collected: contract?.deposit_amount ? String(contract.deposit_amount) : prev.deposit_collected,
                          deposit_return: contract?.deposit_amount ? String(contract.deposit_amount) : prev.deposit_return,
                          guarantee_collected:
                            typeof guaranteeFee === "number" && Number.isFinite(guaranteeFee)
                              ? String(guaranteeFee)
                              : prev.guarantee_collected,
                          guarantee_return:
                            typeof guaranteeFee === "number" && Number.isFinite(guaranteeFee)
                              ? String(guaranteeFee)
                              : prev.guarantee_return,
                        }));
                      }}
                    >
                      <option value="">请选择</option>
                      {activeContracts.map((contract) => (
                        <option key={contract.id} value={contract.id}>
                          {contract.employee_name} · {contract.room?.room_number || "未分配"}
                        </option>
                      ))}
                    </select>
                  </div>
                  <div>
                    <Label className="text-xs font-medium text-muted-foreground">退宿日期 *</Label>
                    <Input
                      type="date"
                      className="mt-1"
                      value={checkoutForm.checkout_date}
                      onChange={(event) => setCheckoutForm((prev) => ({ ...prev, checkout_date: event.target.value }))}
                    />
                  </div>
                  <div>
                    <Label className="text-xs font-medium text-muted-foreground">检查人</Label>
                    <Input
                      className="mt-1"
                      value={checkoutForm.inspector}
                      onChange={(event) => setCheckoutForm((prev) => ({ ...prev, inspector: event.target.value }))}
                    />
                  </div>
                  <div>
                    <Label className="text-xs font-medium text-muted-foreground">设施情况</Label>
                    <Input
                      className="mt-1"
                      placeholder="例如：物品齐全"
                      value={checkoutForm.items_status}
                      onChange={(event) => setCheckoutForm((prev) => ({ ...prev, items_status: event.target.value }))}
                    />
                  </div>
                </div>
              </section>

              <section className="space-y-4">
                <div className="flex items-center justify-between">
                  <h4 className="text-sm font-semibold text-foreground">水电抄表</h4>
                </div>
                <div className={RESPONSIVE_FIELD_GRID_CLASS}>
                  <div>
                    <Label className="text-xs font-medium text-muted-foreground">退宿水表度数</Label>
                    <Input className="mt-1" value={checkoutForm.water_end} onChange={(event) => setCheckoutForm((prev) => ({ ...prev, water_end: event.target.value }))} />
                  </div>
                  <div>
                    <Label className="text-xs font-medium text-muted-foreground">退宿电表度数</Label>
                    <Input className="mt-1" value={checkoutForm.electric_end} onChange={(event) => setCheckoutForm((prev) => ({ ...prev, electric_end: event.target.value }))} />
                  </div>
                </div>
              </section>

              <section className="space-y-4">
                <div className="flex flex-col gap-1 sm:flex-row sm:items-center sm:justify-between">
                  <h4 className="text-sm font-semibold text-foreground">费用情况</h4>
                  <p className="text-xs text-muted-foreground">
                    {showDepositFields || showGuaranteeFields ? "系统仅展示地点已启用的押金 / 保证金项目" : "地点未启用押金或保证金，费用区块将自动隐藏"}
                  </p>
                </div>
                {!showDepositFields && !showGuaranteeFields ? (
                  <div className="rounded-md border border-dashed bg-muted/30 px-4 py-3 text-xs text-muted-foreground">
                    当前地点未配置押金或保证金，无需填写。
                  </div>
                ) : (
                  <div className="space-y-4">
                    {showDepositFields && (
                      <div className={RESPONSIVE_FIELD_GRID_CLASS}>
                        <div>
                          <Label className="text-xs font-medium text-muted-foreground">押金收取金额 (¥)</Label>
                          <Input
                            className="mt-1"
                            type="number"
                            value={checkoutForm.deposit_collected}
                            onChange={(event) => setCheckoutForm((prev) => ({ ...prev, deposit_collected: event.target.value }))}
                          />
                        </div>
                        <div>
                          <Label className="text-xs font-medium text-muted-foreground">扣除金额 (¥)</Label>
                          <Input
                            className="mt-1"
                            type="number"
                            value={checkoutForm.deposit_deduct}
                            onChange={(event) => setCheckoutForm((prev) => ({ ...prev, deposit_deduct: event.target.value }))}
                          />
                        </div>
                        <div>
                          <Label className="text-xs font-medium text-muted-foreground">退还金额 (¥)</Label>
                          <Input
                            className="mt-1"
                            type="number"
                            value={checkoutForm.deposit_return}
                            onChange={(event) => setCheckoutForm((prev) => ({ ...prev, deposit_return: event.target.value }))}
                          />
                        </div>
                        <div>
                          <Label className="text-xs font-medium text-muted-foreground">退还日期</Label>
                          <Input
                            type="date"
                            className="mt-1"
                            value={checkoutForm.deposit_return_date}
                            onChange={(event) => setCheckoutForm((prev) => ({ ...prev, deposit_return_date: event.target.value }))}
                          />
                        </div>
                      </div>
                    )}
                    {showGuaranteeFields && (
                      <div className={RESPONSIVE_FIELD_GRID_CLASS}>
                        <div>
                          <Label className="text-xs font-medium text-muted-foreground">保证金收取金额 (¥)</Label>
                          <Input
                            className="mt-1"
                            type="number"
                            value={checkoutForm.guarantee_collected}
                            onChange={(event) => setCheckoutForm((prev) => ({ ...prev, guarantee_collected: event.target.value }))}
                          />
                        </div>
                        <div>
                          <Label className="text-xs font-medium text-muted-foreground">扣除金额 (¥)</Label>
                          <Input
                            className="mt-1"
                            type="number"
                            value={checkoutForm.guarantee_deduct}
                            onChange={(event) => setCheckoutForm((prev) => ({ ...prev, guarantee_deduct: event.target.value }))}
                          />
                        </div>
                        <div>
                          <Label className="text-xs font-medium text-muted-foreground">退还金额 (¥)</Label>
                          <Input
                            className="mt-1"
                            type="number"
                            value={checkoutForm.guarantee_return}
                            onChange={(event) => setCheckoutForm((prev) => ({ ...prev, guarantee_return: event.target.value }))}
                          />
                        </div>
                        <div>
                          <Label className="text-xs font-medium text-muted-foreground">退还日期</Label>
                          <Input
                            type="date"
                            className="mt-1"
                            value={checkoutForm.guarantee_return_date}
                            onChange={(event) => setCheckoutForm((prev) => ({ ...prev, guarantee_return_date: event.target.value }))}
                          />
                        </div>
                      </div>
                    )}
                  </div>
                )}
              </section>

              <section className="space-y-4">
                <div className="flex items-center justify-between">
                  <h4 className="text-sm font-semibold text-foreground">物品与费用说明</h4>
                </div>
                <div className={RESPONSIVE_FIELD_GRID_CLASS}>
                  <div className="sm:col-span-2">
                    <Label className="text-xs font-medium text-muted-foreground">物品情况</Label>
                    <Textarea
                      rows={3}
                      className="mt-1"
                      placeholder="列出设施损坏、缺失情况"
                      value={checkoutForm.damage_report}
                      onChange={(event) => setCheckoutForm((prev) => ({ ...prev, damage_report: event.target.value }))}
                    />
                  </div>
                  <div className="sm:col-span-2">
                    <Label className="text-xs font-medium text-muted-foreground">费用说明</Label>
                    <Textarea
                      rows={3}
                      className="mt-1"
                      placeholder="填写扣除/退还费用说明"
                      value={checkoutForm.fee_summary}
                      onChange={(event) => setCheckoutForm((prev) => ({ ...prev, fee_summary: event.target.value }))}
                    />
                  </div>
                </div>
              </section>

              <section className="space-y-4">
                <div className="flex items-center justify-between">
                  <h4 className="text-sm font-semibold text-foreground">附件</h4>
                  <Button variant="outline" size="sm" className="gap-1" onClick={handleGenerateCheckoutPdf}>
                    <FileText className="h-4 w-4" /> 生成表单
                  </Button>
                </div>
                <div className="space-y-2">
                  <Label className="text-xs font-medium text-muted-foreground">上传附件</Label>
                  <Input type="file" multiple onChange={handleCheckoutAttachmentChange} />
                  <p className="text-xs text-muted-foreground">支持上传退宿表、交接单等 PDF/图片 文件。</p>
                </div>
                {checkoutForm.attachments.length > 0 && (
                  <div className="rounded-md border bg-muted/20 p-3">
                    <p className="text-xs font-medium text-muted-foreground">已上传附件</p>
                    <div className="mt-2 flex flex-wrap gap-2">
                      {checkoutForm.attachments.map((attachment) => (
                        <Badge key={attachment.name} variant="secondary" className="flex items-center gap-1">
                          {attachment.name}
                          <button type="button" className="text-xs" onClick={() => handleRemoveCheckoutAttachment(attachment.name)}>
                            ×
                          </button>
                        </Badge>
                      ))}
                    </div>
                  </div>
                )}
              </section>
            </div>
          </ScrollArea>
          <DialogFooter className="border-t bg-background px-6 py-4 flex flex-col gap-2 sm:flex-row sm:justify-end">
            <Button variant="outline" onClick={() => setCheckoutDialogOpen(false)}>
              取消
            </Button>
            {/* P7.1：退宿为修改入住记录，需 dormitory.edit 权限 */}
            <RequirePermission resource="dormitory" action="edit">
              <Button onClick={handleCheckoutSubmit} disabled={checkoutSubmitting}>
                {checkoutSubmitting ? "提交中..." : "确认退宿"}
              </Button>
            </RequirePermission>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <AlertDialog
        open={meterDeleteContext !== null}
        onOpenChange={(open) => {
          if (!open) {
            setMeterDeleteContext(null);
          }
        }}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>确认删除抄表记录？</AlertDialogTitle>
            <AlertDialogDescription>
              {meterDeleteContext?.mode === "single" && meterDeleteContext.record
                ? `删除后将无法恢复 ${meterDeleteContext.record.room_number || "该房间"} (${formatDateLabel(meterDeleteContext.record.meter_date)}) 的抄表数据。`
                : `本次将删除选中的 ${meterSelectedIds.length} 条抄表记录，操作不可撤销，请确认。`}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>取消</AlertDialogCancel>
            {/* P7.1：删除抄表记录需 dormitory.delete 权限 */}
            <RequirePermission resource="dormitory" action="delete">
              <AlertDialogAction className="bg-destructive text-destructive-foreground hover:bg-destructive/90" onClick={handleConfirmDeleteMeterRecords}>
                删除
              </AlertDialogAction>
            </RequirePermission>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      <Dialog open={wechatDialogOpen} onOpenChange={setWechatDialogOpen}>
        <DialogContent className={DIALOG_SIZES.sm}>
          <DialogHeader>
            <DialogTitle className="flex items-center gap-2">
              <MessageCircle className="h-4 w-4" />
              客服微信
            </DialogTitle>
            <DialogDescription>{activeWechatConfig ? `来自 ${activeWechatConfig.siteName} 的技术支持联系方式。` : "管理员尚未配置客服微信。"}</DialogDescription>
          </DialogHeader>
          {supportWechatConfigs.length === 0 ? (
            <p className="text-sm text-muted-foreground">管理员尚未配置客服微信。</p>
          ) : (
            <div className="space-y-4">
              {supportWechatConfigs.length > 1 && (
                <div className="flex flex-wrap items-center justify-between gap-2 text-xs text-muted-foreground">
                  <span>选择地点</span>
                  <Select
                    value={String(activeWechatConfig?.siteId ?? supportWechatConfigs[0].siteId)}
                    onValueChange={(value) => setActiveWechatSiteId(Number(value))}
                  >
                    <SelectTrigger className="h-8 w-full sm:w-48">
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      {supportWechatConfigs.map((config) => (
                        <SelectItem key={config.siteId} value={String(config.siteId)}>
                          {config.siteName}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </div>
              )}
              {activeWechatConfig && (/^data:image\//.test(activeWechatConfig.value) || /\.(png|jpe?g|gif|webp|svg)$/i.test(activeWechatConfig.value)) ? (
                <div className="space-y-2 text-center">
                  <Image
                    src={activeWechatConfig.value}
                    alt="客服微信二维码"
                    width={320}
                    height={320}
                    unoptimized
                    className="mx-auto h-auto w-full max-w-[320px] rounded-lg border object-contain"
                  />
                  <p className="text-xs text-muted-foreground">请使用微信扫描二维码添加客服。</p>
                </div>
              ) : activeWechatConfig ? (
                <div className="space-y-3 text-sm">
                  <p>客服链接：</p>
                  <div className="rounded-lg border bg-muted/30 px-3 py-2 text-xs break-all">{activeWechatConfig.value}</div>
                  {/^(https?:)/i.test(activeWechatConfig.value) && (
                    <Button variant="outline" onClick={() => window.open(activeWechatConfig.value, "_blank")}>
                      打开链接
                    </Button>
                  )}
                </div>
              ) : null}
            </div>
          )}
        </DialogContent>
      </Dialog>

      <AlertDialog
        open={Boolean(contractDeleteTarget)}
        onOpenChange={(open) => {
          if (!open && !contractDeleting) {
            setContractDeleteTarget(null);
          }
        }}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>删除入住记录</AlertDialogTitle>
            <AlertDialogDescription>
              将删除 {contractDeleteTarget?.employee_name || "该员工"} 的入住记录，相关水电收费将不再关联，请确认是否继续。
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel disabled={contractDeleting}>取消</AlertDialogCancel>
            {/* P7.1：删除入住记录需 dormitory.delete 权限 */}
            <RequirePermission resource="dormitory" action="delete">
              <AlertDialogAction onClick={handleDeleteContract} disabled={contractDeleting}>
                {contractDeleting ? "删除中..." : "确认删除"}
              </AlertDialogAction>
            </RequirePermission>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      <AlertDialog
        open={Boolean(siteDeleteTarget)}
        onOpenChange={(open) => {
          if (!open && !siteDeleting) {
            setSiteDeleteTarget(null);
          }
        }}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>确认删除地点？</AlertDialogTitle>
            <AlertDialogDescription>
              将删除「{siteDeleteTarget?.name ?? "未命名地点"}」及其房间、床位与入住关联数据，该操作无法撤销，请谨慎操作。
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel disabled={siteDeleting}>取消</AlertDialogCancel>
            {/* P7.1：删除地点需 dormitory.delete 权限 */}
            <RequirePermission resource="dormitory" action="delete">
              <AlertDialogAction onClick={handleDeleteSite} disabled={siteDeleting}>
                {siteDeleting ? "正在删除..." : "确认删除"}
              </AlertDialogAction>
            </RequirePermission>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      <AlertDialog
        open={Boolean(roomDeleteTarget)}
        onOpenChange={(open) => {
          if (!open && !roomDeleting) {
            setRoomDeleteTarget(null);
          }
        }}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>确认删除房间？</AlertDialogTitle>
            <AlertDialogDescription>
              删除房间「{roomDeleteTarget?.room_number ?? "-"}」将同步清理床位与入住记录，且无法恢复，请确认。
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel disabled={roomDeleting}>取消</AlertDialogCancel>
            {/* P7.1：删除房间需 dormitory.delete 权限 */}
            <RequirePermission resource="dormitory" action="delete">
              <AlertDialogAction onClick={handleDeleteRoom} disabled={roomDeleting}>
                {roomDeleting ? "正在删除..." : "确认删除"}
              </AlertDialogAction>
            </RequirePermission>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </PageTransition>
  );
}
