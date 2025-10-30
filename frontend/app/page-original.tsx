"use client";

import { useEffect, useMemo, useRef, useState } from "react";
import { toast } from "sonner";

import {
  clearAdjustments,
  clearFiles,
  createPeriod,
  deletePeriod,
  downloadChargesExcel,
  downloadRosterTemplate,
  downloadSchemeChargesExcel,
  getCharges,
  getRoster,
  getSchemeCharges,
  getSummary,
  importLatestRoster,
  listFiles,
  listPeriods,
  processAdjustments,
  processPeriod,
  resetPeriod,
  uploadAdjustmentsBatch,
  uploadRoster,
  uploadSourceFile,
  uploadSourceFilesBatch,
  // New API functions
  changePassword,
  getAuditLogs,
  getAuditStats,
  getSystemMetrics,
  getDatabaseStatus,
  getSystemInfo,
  runMaintenance,
} from "@/lib/api";
import type {
  Part,
  Period,
  PeriodSummary,
  PersonalCharge,
  RosterEntry,
  Scheme,
  SchemeChargeDetail,
  SourceFile,
  UnitCharge,
  AuditLog,
  AuditStats,
  SystemMetrics,
  DatabaseStatus,
  SystemInfo,
} from "@/lib/types";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Progress } from "@/components/ui/progress";
import { ScrollArea } from "@/components/ui/scroll-area";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Skeleton } from "@/components/ui/skeleton";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import {
  Tabs,
  TabsContent,
  TabsList,
  TabsTrigger,
} from "@/components/ui/tabs";
import { useAuth, useRequireAuth } from "@/lib/auth";

const PART_LABELS: Record<Part, string> = {
  personal: "个人",
  unit: "单位",
};

const SCHEME_LABELS: Record<Scheme, string> = {
  pension: "养老保险",
  medical: "医疗保险",
  serious_illness: "大额医疗",
  unemployment: "失业保险",
  injury: "工伤保险",
};

const STATUS_LABELS: Record<string, string> = {
  draft: "草稿",
  processing: "处理中",
  processed: "已处理",
  completed: "已完成",
  archived: "已归档",
};

const SCHEME_OPTIONS: Array<{
  value: Scheme;
  label: string;
  allowedParts: Part[];
}> = [
  { value: "pension", label: "养老保险", allowedParts: ["personal", "unit"] },
  { value: "medical", label: "医疗保险", allowedParts: ["personal", "unit"] },
  {
    value: "serious_illness",
    label: "大额医疗 / 生育",
    allowedParts: ["personal", "unit"],
  },
  {
    value: "unemployment",
    label: "失业保险",
    allowedParts: ["personal", "unit"],
  },
  { value: "injury", label: "工伤保险", allowedParts: ["unit"] },
];

const REQUIRED_COMBINATIONS: Array<{ part: Part; scheme: Scheme }> = [
  { part: "personal", scheme: "pension" },
  { part: "personal", scheme: "medical" },
  { part: "personal", scheme: "serious_illness" },
  { part: "personal", scheme: "unemployment" },
  { part: "unit", scheme: "pension" },
  { part: "unit", scheme: "medical" },
  { part: "unit", scheme: "serious_illness" },
  { part: "unit", scheme: "unemployment" },
  { part: "unit", scheme: "injury" },
];

const SCHEME_ORDER: Scheme[] = [
  "pension",
  "medical",
  "serious_illness",
  "unemployment",
  "injury",
];

type BatchDraft = {
  file: File;
  scheme: Scheme | "";
  part: Part | "";
};

const currency = new Intl.NumberFormat("zh-CN", {
  minimumFractionDigits: 2,
  maximumFractionDigits: 2,
});

function formatCurrency(value: number) {
  return currency.format(value);
}

function formatDate(value: string) {
  return new Date(value).toLocaleString("zh-CN", {
    hour12: false,
  });
}

function guessPartScheme(fileName: string): { part: Part | ""; scheme: Scheme | "" } {
  const name = fileName.toLowerCase();
  let part: Part | "" = "";
  if (name.includes("单位")) {
    part = "unit";
  } else if (name.includes("个人")) {
    part = "personal";
  }

  let scheme: Scheme | "" = "";
  if (name.includes("工伤")) {
    scheme = "injury";
  } else if (name.includes("失业")) {
    scheme = "unemployment";
  } else if (name.includes("大额") || name.includes("互助")) {
    scheme = "serious_illness";
  } else if (name.includes("养老")) {
    scheme = "pension";
  } else if (name.includes("医疗")) {
    scheme = "medical";
  }

  return { part, scheme };
}

export default function Home() {
  const { isAuthenticated, isLoading } = useRequireAuth();
  const { user, logout } = useAuth();

  // All state declarations must come before any conditional returns
  const [periods, setPeriods] = useState<Period[]>([]);
  const [periodsLoading, setPeriodsLoading] = useState(true);
  const [selectedPeriodId, setSelectedPeriodId] = useState<number | null>(null);
  const [periodDataLoading, setPeriodDataLoading] = useState(false);
  const [files, setFiles] = useState<SourceFile[]>([]);
  const [summary, setSummary] = useState<PeriodSummary[]>([]);
  const [personalCharges, setPersonalCharges] = useState<PersonalCharge[]>([]);
  const [unitCharges, setUnitCharges] = useState<UnitCharge[]>([]);
  const [rosterEntries, setRosterEntries] = useState<RosterEntry[]>([]);
  const [processing, setProcessing] = useState(false);
  const [newPeriod, setNewPeriod] = useState(() => {
    const now = new Date();
    const year = now.getFullYear();
    const month = String(now.getMonth() + 1).padStart(2, '0');
    return `${year}-${month}`;
  });
  const [uploadPart, setUploadPart] = useState<Part>("personal");
  const [uploadScheme, setUploadScheme] = useState<Scheme>("pension");
  const [uploading, setUploading] = useState(false);
  const [rosterUploading, setRosterUploading] = useState(false);
  const [rosterImporting, setRosterImporting] = useState(false);
  const [templateDownloading, setTemplateDownloading] = useState(false);
  const [batchDialogOpen, setBatchDialogOpen] = useState(false);
  const [batchDrafts, setBatchDrafts] = useState<BatchDraft[]>([]);
  const [adjustmentDialogOpen, setAdjustmentDialogOpen] = useState(false);
  const [selectedAdjustmentFiles, setSelectedAdjustmentFiles] = useState<File[]>([]);
  const [adjustmentUploading, setAdjustmentUploading] = useState(false);
  const [adjustmentProcessing, setAdjustmentProcessing] = useState(false);
  const [batchUploading, setBatchUploading] = useState(false);
  const [exportingPart, setExportingPart] = useState<Part | null>(null);
  const [clearingFiles, setClearingFiles] = useState(false);
  const [clearingAdjustments, setClearingAdjustments] = useState(false);
  const [schemeModalOpen, setSchemeModalOpen] = useState(false);
  const [selectedScheme, setSelectedScheme] = useState<Scheme | null>(null);
  const [selectedPart, setSelectedPart] = useState<Part | null>(null);
  const [schemeCharges, setSchemeCharges] = useState<SchemeChargeDetail[]>([]);
  const [schemeChargesLoading, setSchemeChargesLoading] = useState(false);
  const [schemeExporting, setSchemeExporting] = useState(false);
  const [resetDialogOpen, setResetDialogOpen] = useState(false);
  const [resetting, setResetting] = useState(false);
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false);
  const [deleting, setDeleting] = useState(false);

  // New feature states
  const [passwordDialogOpen, setPasswordDialogOpen] = useState(false);
  const [oldPassword, setOldPassword] = useState("");
  const [newPassword, setNewPassword] = useState("");
  const [confirmPassword, setConfirmPassword] = useState("");
  const [changingPassword, setChangingPassword] = useState(false);

  const [auditDialogOpen, setAuditDialogOpen] = useState(false);
  const [auditLogs, setAuditLogs] = useState<AuditLog[]>([]);
  const [auditStats, setAuditStats] = useState<AuditStats | null>(null);
  const [loadingAudit, setLoadingAudit] = useState(false);

  const [monitoringDialogOpen, setMonitoringDialogOpen] = useState(false);
  const [systemMetrics, setSystemMetrics] = useState<SystemMetrics | null>(null);
  const [databaseStatus, setDatabaseStatus] = useState<DatabaseStatus | null>(null);
  const [systemInfo, setSystemInfo] = useState<SystemInfo | null>(null);
  const [loadingMonitoring, setLoadingMonitoring] = useState(false);
  const [personalSearchText, setPersonalSearchText] = useState("");
  const [personalSearchDept, setPersonalSearchDept] = useState("__all__");
  const [unitSearchText, setUnitSearchText] = useState("");
  const [unitSearchDept, setUnitSearchDept] = useState("__all__");

  // All refs must also come before conditional returns
  const adjustmentFileInputRef = useRef<HTMLInputElement>(null);
  const fileInputRef = useRef<HTMLInputElement | null>(null);
  const rosterInputRef = useRef<HTMLInputElement | null>(null);
  const batchFileInputRef = useRef<HTMLInputElement | null>(null);

  // All useMemo and derived state must come before conditional returns
  const normalFiles = useMemo(() =>
    files.filter(file => file.file_type === 'normal' || !file.file_type),
    [files]
  );
  const adjustmentFiles = useMemo(() =>
    files.filter(file => file.file_type === 'adjustment'),
    [files]
  );

  const selectedPeriod = useMemo(
    () => periods.find((item) => item.id === selectedPeriodId) ?? null,
    [periods, selectedPeriodId],
  );

  const uploadKeySet = useMemo(() => {
    const set = new Set<string>();
    normalFiles.forEach((file) => {
      set.add(`${file.part}-${file.scheme}`);
    });
    return set;
  }, [normalFiles]);

  const missingUploads = useMemo(
    () =>
      REQUIRED_COMBINATIONS.filter(
        (combo) => !uploadKeySet.has(`${combo.part}-${combo.scheme}`),
      ),
    [uploadKeySet],
  );

  const uploadProgress = useMemo(() => {
    return Math.round(
      (uploadKeySet.size / REQUIRED_COMBINATIONS.length) * 100,
    );
  }, [uploadKeySet.size]);

  const filteredPersonalCharges = useMemo(() => {
    const filtered = personalCharges.filter((charge) => {
      const searchMatch = !personalSearchText ||
        charge.name.toLowerCase().includes(personalSearchText.toLowerCase()) ||
        charge.id_number.includes(personalSearchText);
      const deptMatch = !personalSearchDept || personalSearchDept === "__all__" || (charge.department && charge.department.toLowerCase().includes(personalSearchDept.toLowerCase()));
      return searchMatch && deptMatch;
    });

    const groupedMap = new Map<string, PersonalCharge[]>();

    filtered.forEach((charge) => {
      const key = `${charge.name}_${charge.id_number}`;
      if (!groupedMap.has(key)) {
        groupedMap.set(key, []);
      }
      groupedMap.get(key)!.push(charge);
    });

    const result: (PersonalCharge & { isFirstInGroup?: boolean; groupKey?: string })[] = [];

    for (const [groupKey, charges] of groupedMap.entries()) {
      charges.sort((a, b) => (a.is_adjustment ? 1 : 0) - (b.is_adjustment ? 1 : 0));
      charges.forEach((charge, index) => {
        result.push({
          ...charge,
          isFirstInGroup: index === 0,
          groupKey
        });
      });
    }

    return result;
  }, [personalCharges, personalSearchText, personalSearchDept]);

  const filteredUnitCharges = useMemo(() => {
    const filtered = unitCharges.filter((charge) => {
      const searchMatch = !unitSearchText ||
        charge.name.toLowerCase().includes(unitSearchText.toLowerCase()) ||
        charge.id_number.includes(unitSearchText);
      const deptMatch = !unitSearchDept || unitSearchDept === "__all__" || (charge.department && charge.department.toLowerCase().includes(unitSearchDept.toLowerCase()));
      return searchMatch && deptMatch;
    });

    const groupedMap = new Map<string, UnitCharge[]>();

    filtered.forEach((charge) => {
      const key = `${charge.name}_${charge.id_number}`;
      if (!groupedMap.has(key)) {
        groupedMap.set(key, []);
      }
      groupedMap.get(key)!.push(charge);
    });

    const result: (UnitCharge & { isFirstInGroup?: boolean; groupKey?: string })[] = [];

    for (const [groupKey, charges] of groupedMap.entries()) {
      charges.sort((a, b) => (a.is_adjustment ? 1 : 0) - (b.is_adjustment ? 1 : 0));
      charges.forEach((charge, index) => {
        result.push({
          ...charge,
          isFirstInGroup: index === 0,
          groupKey
        });
      });
    }

    return result;
  }, [unitCharges, unitSearchText, unitSearchDept]);

  const personalDepartments = useMemo(() => {
    const depts = new Set(personalCharges.map(charge => charge.department).filter(Boolean));
    return Array.from(depts).sort();
  }, [personalCharges]);

  const unitDepartments = useMemo(() => {
    const depts = new Set(unitCharges.map(charge => charge.department).filter(Boolean));
    return Array.from(depts).sort();
  }, [unitCharges]);

  const allowedSchemes = useMemo(
    () =>
      SCHEME_OPTIONS.filter((option) =>
        option.allowedParts.includes(uploadPart),
      ),
    [uploadPart],
  );

  // All useEffect hooks must also come before conditional returns
  useEffect(() => {
    // Only load periods if user is authenticated and not loading
    if (!isAuthenticated || isLoading) {
      setPeriodsLoading(false);
      return;
    }

    const load = async () => {
      try {
        setPeriodsLoading(true);
        const data = await listPeriods();
        setPeriods(data);
        if (data.length > 0) {
          setSelectedPeriodId((prev) =>
            prev ? prev : data[data.length - 1]?.id ?? null,
          );
        }
      } catch (error) {
        console.error(error);
        toast.error("加载账期失败");
      } finally {
        setPeriodsLoading(false);
      }
    };
    load();
  }, [isAuthenticated, isLoading]);

  useEffect(() => {
    if (!selectedPeriodId) {
      setFiles([]);
      setSummary([]);
      setPersonalCharges([]);
      setUnitCharges([]);
      setRosterEntries([]);
      return;
    }
    const loadPeriodData = async (periodId: number) => {
      setPeriodDataLoading(true);
      try {
        const [fileList, summaryList, personalList, unitList, rosterList] =
          await Promise.all(
            [
              listFiles(periodId),
              getSummary(periodId).catch(() => []),
              getCharges(periodId, "personal").catch(() => []),
              getCharges(periodId, "unit").catch(() => []),
              getRoster(periodId).catch(() => []),
            ],
          );
        setFiles(fileList);
        setSummary(summaryList);
        setPersonalCharges(personalList as PersonalCharge[]);
        setUnitCharges(unitList as UnitCharge[]);
        setRosterEntries(rosterList as RosterEntry[]);
      } catch (error) {
        console.error(error);
        toast.error("加载账期数据失败");
      } finally {
        setPeriodDataLoading(false);
      }
    };
    loadPeriodData(selectedPeriodId);
  }, [selectedPeriodId]);

  useEffect(() => {
    if (!allowedSchemes.some((item) => item.value === uploadScheme)) {
      setUploadScheme(allowedSchemes[0]?.value ?? "pension");
    }
  }, [allowedSchemes, uploadScheme]);

  // Show loading screen while checking authentication
  if (isLoading) {
    return (
      <div className="flex items-center justify-center min-h-screen">
        <div className="text-center space-y-4">
          <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary mx-auto"></div>
          <p className="text-muted-foreground">正在验证身份...</p>
        </div>
      </div>
    );
  }

  // If not authenticated, the useRequireAuth hook will redirect
  if (!isAuthenticated) {
    return null;
  }

  const handleCreatePeriod = async () => {
    if (!newPeriod) {
      toast.info("请选择账期");
      return;
    }
    try {
      const created = await createPeriod(newPeriod);
      toast.success(`账期 ${created.year_month} 已创建`);
      const data = await listPeriods();
      setPeriods(data);
      setSelectedPeriodId(created.id);
      setNewPeriod("");
    } catch (error) {
      console.error(error);
      toast.error("创建账期失败");
    }
  };

  const handleUpload = async () => {
    const file = fileInputRef.current?.files?.[0];
    if (!file) {
      toast.info("请选择需要上传的文件");
      return;
    }
    if (!selectedPeriodId) {
      toast.info("请先选择账期");
      return;
    }
    setUploading(true);
    try {
      await uploadSourceFile({
        periodId: selectedPeriodId,
        scheme: uploadScheme,
        part: uploadPart,
        file,
      });
      toast.success("上传成功");
      if (fileInputRef.current) {
        fileInputRef.current.value = "";
      }
      const [fileList] = await Promise.all([listFiles(selectedPeriodId)]);
      setFiles(fileList);
    } catch (error) {
      console.error(error);
      toast.error(error instanceof Error ? error.message : "上传失败");
    } finally {
      setUploading(false);
    }
  };

  const handleRosterUpload = async () => {
    const file = rosterInputRef.current?.files?.[0];
    if (!file) {
      toast.info("请选择花名册文件");
      return;
    }
    if (!selectedPeriodId) {
      toast.info("请先选择账期");
      return;
    }
    setRosterUploading(true);
    try {
      await uploadRoster(selectedPeriodId, file);
      toast.success("花名册导入成功");
      if (rosterInputRef.current) {
        rosterInputRef.current.value = "";
      }
      const rosterList = await getRoster(selectedPeriodId).catch(() => []);
      setRosterEntries(rosterList as RosterEntry[]);
    } catch (error) {
      console.error(error);
      toast.error(error instanceof Error ? error.message : "花名册导入失败");
    } finally {
      setRosterUploading(false);
    }
  };

  const handleDownloadTemplate = async () => {
    setTemplateDownloading(true);
    try {
      const blob = await downloadRosterTemplate();
      const url = URL.createObjectURL(blob);
      const link = document.createElement("a");
      link.href = url;
      link.download = "花名册模板.xlsx";
      document.body.appendChild(link);
      link.click();
      document.body.removeChild(link);
      URL.revokeObjectURL(url);
      toast.success("模板下载成功");
    } catch (error) {
      console.error(error);
      toast.error(error instanceof Error ? error.message : "模板下载失败");
    } finally {
      setTemplateDownloading(false);
    }
  };

  const handleRosterImport = async () => {
    if (!selectedPeriodId) {
      toast.info("请先选择账期");
      return;
    }
    setRosterImporting(true);
    try {
      const result = await importLatestRoster(selectedPeriodId);

      if (result.imported === 0) {
        toast.warning("系统内暂无可导入的花名册数据。请先在其他账期上传花名册文件，或手动上传本账期的花名册文件。");
      } else {
        toast.success(`一键导入成功：已导入 ${result.imported} 人。${result.message}`);
      }

      const rosterList = await getRoster(selectedPeriodId).catch(() => []);
      setRosterEntries(rosterList as RosterEntry[]);
    } catch (error) {
      const errorMessage = error instanceof Error ? error.message : "一键导入失败";

      // 检查是否是"无可导入数据"的错误
      if (errorMessage.includes("no existing roster data found") || errorMessage.includes("暂无可导入")) {
        toast.warning("系统内暂无可导入的花名册数据。请先在其他账期上传花名册文件，或手动上传本账期的花名册文件。");
      } else {
        console.error(error);
        toast.error(errorMessage);
      }
    } finally {
      setRosterImporting(false);
    }
  };

  const handleBatchFilesChange = (files: FileList | null) => {
    if (!files) return;
    const drafts: BatchDraft[] = Array.from(files).map((file) => {
      const guess = guessPartScheme(file.name);
      return {
        file,
        part: guess.part,
        scheme: guess.scheme,
      };
    });
    setBatchDrafts(drafts);
  };

  const handleBatchFieldChange = (
    index: number,
    field: "part" | "scheme",
    value: string,
  ) => {
    setBatchDrafts((prev) =>
      prev.map((item, idx) =>
        idx === index
          ? {
              ...item,
              [field]:
                field === "part" ? (value as Part | "") : (value as Scheme | ""),
            }
          : item,
      ),
    );
  };

  const handleBatchUploadSubmit = async () => {
    if (!selectedPeriodId) {
      toast.info("请先选择账期");
      return;
    }
    if (batchDrafts.length === 0) {
      toast.info("请先选择文件");
      return;
    }
    const incomplete = batchDrafts.some(
      (item) => !item.part || !item.scheme,
    );
    if (incomplete) {
      toast.error("请为所有文件指定险种与扣款部分");
      return;
    }

    setBatchUploading(true);
    try {
      const payload = batchDrafts.map((item) => ({
        file: item.file,
        part: item.part as Part,
        scheme: item.scheme as Scheme,
      }));
      const response = await uploadSourceFilesBatch({
        periodId: selectedPeriodId,
        items: payload,
      });
      const successCount = response.items.filter((item) => !item.error).length;
      const failItems = response.items.filter((item) => item.error);
      if (successCount > 0) {
        toast.success(`成功上传 ${successCount} 份文件`);
      }
      failItems.forEach((item) => {
        toast.error(`${item.original_name}: ${item.error}`);
      });
      setBatchDrafts([]);
      if (batchFileInputRef.current) {
        batchFileInputRef.current.value = "";
      }
      setBatchDialogOpen(false);
      const fileList = await listFiles(selectedPeriodId);
      setFiles(fileList);
    } catch (error) {
      console.error(error);
      toast.error(error instanceof Error ? error.message : "批量上传失败");
    } finally {
      setBatchUploading(false);
    }
  };

  // 补退文件上传
  const handleAdjustmentUpload = async () => {
    if (!selectedPeriodId) {
      toast.info("请先选择账期");
      return;
    }
    if (selectedAdjustmentFiles.length === 0) {
      toast.info("请先选择补退文件");
      return;
    }

    setAdjustmentUploading(true);
    try {
      const response = await uploadAdjustmentsBatch(selectedPeriodId, selectedAdjustmentFiles);
      const successCount = response.items.filter((item) => !item.error).length;
      const failItems = response.items.filter((item) => item.error);

      if (failItems.length > 0) {
        console.log("Failed items:", failItems);
        toast.error(
          `部分文件上传失败：${failItems.length} 个文件。成功上传：${successCount} 个文件。`
        );
      } else {
        toast.success(`补退文件批量上传成功，共处理 ${successCount} 个文件`);
      }

      setAdjustmentDialogOpen(false);
      setSelectedAdjustmentFiles([]);

      // 刷新文件列表显示补退文件
      const fileList = await listFiles(selectedPeriodId);
      setFiles(fileList);
    } catch (error) {
      console.error(error);
      toast.error(error instanceof Error ? error.message : "补退文件上传失败");
    } finally {
      setAdjustmentUploading(false);
    }
  };

  // 处理补退数据
  const handleProcessAdjustments = async () => {
    if (!selectedPeriodId) {
      toast.info("请先选择账期");
      return;
    }

    setAdjustmentProcessing(true);
    try {
      const result = await processAdjustments(selectedPeriodId);
      toast.success("补退数据处理完成，已累加到现有扣款明细中");

      // 刷新数据
      const [summaryData, personalCharges, unitCharges] = await Promise.all([
        getSummary(selectedPeriodId),
        getCharges(selectedPeriodId, "personal"),
        getCharges(selectedPeriodId, "unit"),
      ]);
      setSummary(summaryData);
      setPersonalCharges(personalCharges as PersonalCharge[]);
      setUnitCharges(unitCharges as UnitCharge[]);
    } catch (error) {
      console.error(error);
      toast.error(error instanceof Error ? error.message : "补退数据处理失败");
    } finally {
      setAdjustmentProcessing(false);
    }
  };

  const handleExport = async (part: Part) => {
    if (!selectedPeriodId) return;
    setExportingPart(part);
    try {
      const blob = await downloadChargesExcel(selectedPeriodId, part);
      const url = URL.createObjectURL(blob);
      const link = document.createElement("a");
      const label = part === "personal" ? "个人" : "单位";
      link.href = url;
      link.download = `${selectedPeriod?.year_month ?? "period"}-${label}扣款明细.xlsx`;
      document.body.appendChild(link);
      link.click();
      document.body.removeChild(link);
      URL.revokeObjectURL(url);
      toast.success(`${label}扣款明细已导出`);
    } catch (error) {
      console.error(error);
      toast.error(error instanceof Error ? error.message : "导出失败");
    } finally {
      setExportingPart(null);
    }
  };

  const handleProcess = async () => {
    if (!selectedPeriodId) return;
    setProcessing(true);
    try {
      const result = await processPeriod(selectedPeriodId);
      toast.success("数据处理成功");
      setSummary(result.summary);
      setPersonalCharges(result.personal);
      setUnitCharges(result.unit);
    } catch (error) {
      console.error(error);
      toast.error(error instanceof Error ? error.message : "处理失败");
    } finally {
      setProcessing(false);
    }
  };

  const handleSchemeClick = async (scheme: Scheme, part: Part, isAdjustment?: boolean) => {
    if (!selectedPeriodId) return;
    setSelectedScheme(scheme);
    setSelectedPart(part);
    setSchemeModalOpen(true);
    setSchemeChargesLoading(true);

    try {
      const charges = await getSchemeCharges(selectedPeriodId, scheme, part, isAdjustment);
      setSchemeCharges(charges);
    } catch (error) {
      console.error(error);
      toast.error(error instanceof Error ? error.message : "获取明细失败");
    } finally {
      setSchemeChargesLoading(false);
    }
  };

  const handleSchemeExport = async () => {
    if (!selectedPeriodId || !selectedScheme || !selectedPart) return;
    setSchemeExporting(true);
    try {
      const blob = await downloadSchemeChargesExcel(selectedPeriodId, selectedScheme, selectedPart);
      const url = URL.createObjectURL(blob);
      const link = document.createElement("a");
      const schemeLabel = SCHEME_LABELS[selectedScheme];
      const partLabel = selectedPart === "personal" ? "个人" : "单位";
      link.href = url;
      link.download = `${selectedPeriod?.year_month ?? "period"}-${schemeLabel}-${partLabel}明细.xlsx`;
      document.body.appendChild(link);
      link.click();
      document.body.removeChild(link);
      URL.revokeObjectURL(url);
      toast.success(`${schemeLabel}-${partLabel}明细已导出`);
    } catch (error) {
      console.error(error);
      toast.error(error instanceof Error ? error.message : "导出失败");
    } finally {
      setSchemeExporting(false);
    }
  };

  const handleReset = async () => {
    if (!selectedPeriodId) return;
    setResetting(true);
    try {
      await resetPeriod(selectedPeriodId);
      toast.success("账期已重置，所有数据已清空");
      setResetDialogOpen(false);

      // 重新加载账期列表和数据
      const [updatedPeriods] = await Promise.all([
        listPeriods(),
      ]);
      setPeriods(updatedPeriods);

      // 清空当前显示的数据
      setFiles([]);
      setSummary([]);
      setPersonalCharges([]);
      setUnitCharges([]);
      setRosterEntries([]);
    } catch (error) {
      console.error(error);
      toast.error(error instanceof Error ? error.message : "重置失败");
    } finally {
      setResetting(false);
    }
  };

  // Delete period function
  const handleDeletePeriod = async () => {
    if (!selectedPeriodId) return;
    setDeleting(true);
    try {
      await deletePeriod(selectedPeriodId);
      toast.success("账期已删除");
      setDeleteDialogOpen(false);

      // 重新加载账期列表
      const updatedPeriods = await listPeriods();
      setPeriods(updatedPeriods);

      // 如果删除的是当前选中的账期，需要重新选择
      if (updatedPeriods.length > 0) {
        const newPeriodId = updatedPeriods[0].id;
        setSelectedPeriodId(newPeriodId);
      } else {
        setSelectedPeriodId(null);
        // 清空所有数据
        setFiles([]);
        setSummary([]);
        setPersonalCharges([]);
        setUnitCharges([]);
        setRosterEntries([]);
      }
    } catch (error) {
      console.error(error);
      toast.error(error instanceof Error ? error.message : "删除失败");
    } finally {
      setDeleting(false);
    }
  };

  const handleClearFiles = async () => {
    if (!selectedPeriodId) return;
    setClearingFiles(true);
    try {
      await clearFiles(selectedPeriodId);
      toast.success("社保文件已清空");

      // 重新加载文件列表和数据
      const fileList = await listFiles(selectedPeriodId);
      setFiles(fileList);

      // 清空相关数据显示
      setSummary([]);
      setPersonalCharges([]);
      setUnitCharges([]);
    } catch (error) {
      console.error(error);
      toast.error(error instanceof Error ? error.message : "清空失败");
    } finally {
      setClearingFiles(false);
    }
  };

  const handleClearAdjustments = async () => {
    if (!selectedPeriodId) return;
    setClearingAdjustments(true);
    try {
      await clearAdjustments(selectedPeriodId);
      toast.success("补退文件已清空");

      // 重新加载文件列表和数据
      const fileList = await listFiles(selectedPeriodId);
      setFiles(fileList);

      // 清空相关数据显示
      setSummary([]);
      setPersonalCharges([]);
      setUnitCharges([]);
    } catch (error) {
      console.error(error);
      toast.error(error instanceof Error ? error.message : "清空失败");
    } finally {
      setClearingAdjustments(false);
    }
  };

  // Password change handler
  const handlePasswordChange = async () => {
    if (!oldPassword || !newPassword || !confirmPassword) {
      toast.error("请填写所有密码字段");
      return;
    }
    if (newPassword !== confirmPassword) {
      toast.error("新密码与确认密码不匹配");
      return;
    }
    if (newPassword.length < 6) {
      toast.error("新密码长度至少为6位");
      return;
    }

    try {
      setChangingPassword(true);
      await changePassword(oldPassword, newPassword);
      toast.success("密码修改成功");
      setPasswordDialogOpen(false);
      setOldPassword("");
      setNewPassword("");
      setConfirmPassword("");
    } catch (error) {
      console.error(error);
      toast.error(`密码修改失败: ${error}`);
    } finally {
      setChangingPassword(false);
    }
  };

  // Load audit logs
  const handleLoadAuditLogs = async () => {
    try {
      setLoadingAudit(true);
      const [logsResult, statsResult] = await Promise.all([
        getAuditLogs({ limit: 50 }),
        getAuditStats(),
      ]);
      setAuditLogs(logsResult.logs);
      setAuditStats(statsResult);
    } catch (error) {
      console.error(error);
      toast.error("加载审计日志失败");
    } finally {
      setLoadingAudit(false);
    }
  };

  // Load monitoring data
  const handleLoadMonitoringData = async () => {
    try {
      setLoadingMonitoring(true);
      const [metrics, dbStatus, sysInfo] = await Promise.all([
        getSystemMetrics(),
        getDatabaseStatus(),
        getSystemInfo(),
      ]);
      setSystemMetrics(metrics);
      setDatabaseStatus(dbStatus);
      setSystemInfo(sysInfo);
    } catch (error) {
      console.error(error);
      toast.error("加载监控数据失败");
    } finally {
      setLoadingMonitoring(false);
    }
  };

  return (
    <div className="min-h-screen bg-muted/30">
      <div className="mx-auto flex w-full max-w-7xl flex-col gap-6 p-6 pb-16">
        <header className="flex flex-col gap-2">
          <div className="flex flex-wrap items-center justify-between gap-4">
            <div>
              <h1 className="text-3xl font-bold tracking-tight">
                社保数据整合工具
              </h1>
              <p className="text-muted-foreground">
                上传各险种明细，生成社保总表与扣款明细。
              </p>
              {user && (
                <p className="text-sm text-muted-foreground mt-1">
                  欢迎，{user.full_name || user.username}
                </p>
              )}
            </div>
            <div className="flex flex-wrap items-center gap-4">
              {user && (
                <>
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => {
                      setPasswordDialogOpen(true);
                    }}
                  >
                    修改密码
                  </Button>
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => {
                      setAuditDialogOpen(true);
                      handleLoadAuditLogs();
                    }}
                  >
                    审计日志
                  </Button>
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => {
                      setMonitoringDialogOpen(true);
                      handleLoadMonitoringData();
                    }}
                  >
                    系统监控
                  </Button>
                  <Button variant="outline" size="sm" onClick={logout}>
                    退出登录
                  </Button>
                </>
              )}
              {periods.length > 0 && (
                <div className="flex items-center gap-3 bg-gray-50 rounded-lg px-4 py-2 border">
                  <Label htmlFor="period-switcher" className="text-sm font-semibold text-gray-700">
                    当前账期：
                  </Label>
                  <Select
                    value={selectedPeriodId ? String(selectedPeriodId) : ""}
                    onValueChange={(value) => setSelectedPeriodId(Number(value))}
                    disabled={periodsLoading}
                  >
                    <SelectTrigger id="period-switcher" className="w-36 h-8 border-gray-300">
                      <SelectValue placeholder="选择账期" />
                    </SelectTrigger>
                    <SelectContent>
                      {periods.map((period) => (
                        <SelectItem key={period.id} value={String(period.id)}>
                          {period.year_month}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                  {selectedPeriod && (
                    <Badge
                      className={`text-xs font-medium ${
                        selectedPeriod.status === 'draft' ? 'bg-gray-100 text-gray-700 border-gray-300' :
                        selectedPeriod.status === 'processing' ? 'bg-blue-100 text-blue-700 border-blue-300' :
                        selectedPeriod.status === 'processed' ? 'bg-green-100 text-green-700 border-green-300' :
                        selectedPeriod.status === 'completed' ? 'bg-emerald-100 text-emerald-700 border-emerald-300' :
                        selectedPeriod.status === 'archived' ? 'bg-orange-100 text-orange-700 border-orange-300' :
                        'bg-gray-100 text-gray-700 border-gray-300'
                      }`}
                      variant="outline"
                    >
                      {STATUS_LABELS[selectedPeriod.status] || selectedPeriod.status}
                    </Badge>
                  )}
                </div>
              )}
            </div>
          </div>
        </header>

        <Tabs defaultValue="import" className="w-full">
          <TabsList className="grid w-full max-w-md grid-cols-2">
            <TabsTrigger value="import">数据导入</TabsTrigger>
            <TabsTrigger value="result">数据处理</TabsTrigger>
          </TabsList>

          <TabsContent value="import" className="space-y-6">
            <Card>
              <CardHeader>
                <CardTitle>账期管理</CardTitle>
                <CardDescription>
                  新增账期或选择已有账期进行数据导入。
                </CardDescription>
              </CardHeader>
              <CardContent className="space-y-4">
                <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
                  <div className="space-y-2">
                    <Label htmlFor="year-month">新建账期（YYYY-MM）</Label>
                    <div className="flex gap-2">
                      <Input
                        id="year-month"
                        type="month"
                        value={newPeriod}
                        onChange={(event) => setNewPeriod(event.target.value)}
                        className="w-full"
                      />
                      <Button onClick={handleCreatePeriod}>创建</Button>
                    </div>
                  </div>
                  <div className="space-y-2">
                    <Label>账期操作</Label>
                    <div className="flex gap-2">
                      <Dialog open={resetDialogOpen} onOpenChange={setResetDialogOpen}>
                        <DialogTrigger asChild>
                          <Button
                            variant="destructive"
                            disabled={!selectedPeriodId}
                            className="flex-1"
                          >
                            重置账期
                          </Button>
                        </DialogTrigger>
                        <DialogContent>
                          <DialogHeader>
                            <DialogTitle>确认重置账期</DialogTitle>
                            <DialogDescription>
                              此操作将清空当前账期的所有数据，包括：
                              <br />• 已上传的社保文件
                              <br />• 花名册数据
                              <br />• 处理结果
                              <br />• 扣款明细
                              <br /><br />
                              <strong>此操作不可撤销，请确认是否继续？</strong>
                            </DialogDescription>
                          </DialogHeader>
                          <DialogFooter>
                            <Button
                              variant="outline"
                              onClick={() => setResetDialogOpen(false)}
                              disabled={resetting}
                            >
                              取消
                            </Button>
                            <Button
                              variant="destructive"
                              onClick={handleReset}
                              disabled={resetting}
                            >
                              {resetting ? "重置中..." : "确认重置"}
                            </Button>
                          </DialogFooter>
                        </DialogContent>
                      </Dialog>
                      <Dialog open={deleteDialogOpen} onOpenChange={setDeleteDialogOpen}>
                        <DialogTrigger asChild>
                          <Button
                            variant="destructive"
                            disabled={!selectedPeriodId}
                            className="flex-1"
                          >
                            删除账期
                          </Button>
                        </DialogTrigger>
                        <DialogContent>
                          <DialogHeader>
                            <DialogTitle>确认删除账期</DialogTitle>
                            <DialogDescription>
                              此操作将永久删除当前账期及其所有数据，包括：
                              <br />• 账期记录
                              <br />• 已上传的社保文件
                              <br />• 花名册数据
                              <br />• 处理结果
                              <br />• 扣款明细
                              <br /><br />
                              <strong className="text-red-600">此操作不可撤销，账期将被永久删除！</strong>
                            </DialogDescription>
                          </DialogHeader>
                          <DialogFooter>
                            <Button
                              variant="outline"
                              onClick={() => setDeleteDialogOpen(false)}
                              disabled={deleting}
                            >
                              取消
                            </Button>
                            <Button
                              variant="destructive"
                              onClick={handleDeletePeriod}
                              disabled={deleting}
                            >
                              {deleting ? "删除中..." : "确认删除"}
                            </Button>
                          </DialogFooter>
                        </DialogContent>
                      </Dialog>
                    </div>
                  </div>
                </div>
                {periodsLoading && (
                  <div className="flex gap-4">
                    <Skeleton className="h-10 w-40" />
                    <Skeleton className="h-10 flex-1" />
                  </div>
                )}
              </CardContent>
            </Card>

            <Card>
              <CardHeader>
                <div className="flex flex-wrap items-center justify-between gap-3">
                  <div className="space-y-1.5">
                    <CardTitle>花名册导入</CardTitle>
                    <CardDescription>
                      上传花名册以补全部门等信息，文件需包含“姓名”“证件号码”“部门”列。
                    </CardDescription>
                  </div>
                  <Badge variant="outline">
                    已导入 {rosterEntries.length} 人
                  </Badge>
                </div>
              </CardHeader>
              <CardContent className="space-y-4">
                <div className="grid gap-4 lg:grid-cols-[2fr_3fr]">
                  <div className="space-y-3">
                    <Label htmlFor="roster-upload">花名册文件</Label>
                    <Input
                      id="roster-upload"
                      type="file"
                      accept=".xlsx,.xls,.csv"
                      ref={rosterInputRef}
                      disabled={!selectedPeriodId}
                    />
                    <div className="flex gap-2">
                      <Button
                        onClick={handleRosterUpload}
                        disabled={!selectedPeriodId || rosterUploading}
                        className="flex-1"
                      >
                        {rosterUploading ? "上传中..." : "导入花名册"}
                      </Button>
                      <Button
                        variant="outline"
                        onClick={handleRosterImport}
                        disabled={!selectedPeriodId || rosterImporting}
                      >
                        {rosterImporting ? "导入中..." : "一键导入"}
                      </Button>
                      <Button
                        variant="outline"
                        onClick={handleDownloadTemplate}
                        disabled={templateDownloading}
                      >
                        {templateDownloading ? "下载中..." : "下载模板"}
                      </Button>
                    </div>
                    <p className="text-sm text-muted-foreground">
                      可先下载模板文件，确保包含姓名、证件号码、部门等必需列。点击&ldquo;一键导入&rdquo;可导入系统内已有的最新花名册数据。
                    </p>
                  </div>
                  <div className="space-y-2">
                    <div className="flex items-center justify-between">
                      <h3 className="text-sm font-semibold">花名册列表</h3>
                      <span className="text-xs text-muted-foreground">
                        共 {rosterEntries.length} 人
                      </span>
                    </div>
                    <ScrollArea className="h-48 rounded-md border bg-background">
                      <Table>
                        <TableHeader>
                          <TableRow>
                            <TableHead>姓名</TableHead>
                            <TableHead>证件号码</TableHead>
                            <TableHead>部门</TableHead>
                            <TableHead>岗位</TableHead>
                          </TableRow>
                        </TableHeader>
                        <TableBody>
                          {rosterEntries.length === 0 ? (
                            <TableRow>
                              <TableCell
                                colSpan={4}
                                className="h-20 text-center text-muted-foreground"
                              >
                                暂未导入花名册。
                              </TableCell>
                            </TableRow>
                          ) : (
                            rosterEntries.map((item) => (
                              <TableRow
                                key={`${item.id_number}-${item.department}-${item.name}`}
                              >
                                <TableCell>{item.name || "-"}</TableCell>
                                <TableCell className="font-mono text-xs">
                                  {item.id_number}
                                </TableCell>
                                <TableCell>{item.department || "-"}</TableCell>
                                <TableCell>{item.title || "-"}</TableCell>
                              </TableRow>
                            ))
                          )}
                        </TableBody>
                      </Table>
                    </ScrollArea>
                  </div>
                </div>
              </CardContent>
            </Card>

            <Card>
              <CardHeader>
                <div className="flex flex-wrap items-center justify-between gap-3">
                  <div className="space-y-1.5">
                    <CardTitle>社保文件上传</CardTitle>
                    <CardDescription>
                      上传社保局导出的五险数据Excel文件，系统会自动覆盖同险种旧文件。
                    </CardDescription>
                  </div>
                  <div className="flex items-center gap-3">
                    <span className="text-sm text-muted-foreground">
                      已选择 {batchDrafts.length} 个社保文件
                    </span>
                  </div>
                </div>
              </CardHeader>
              <CardContent className="space-y-6">
                <div className="flex justify-end">
                  <div className="flex gap-2">
                  {missingUploads.length > 0 && (
                    <Dialog>
                      <DialogTrigger asChild>
                        <Button variant="outline">
                          查看缺失文件
                        </Button>
                      </DialogTrigger>
                      <DialogContent>
                        <DialogHeader>
                          <DialogTitle>缺失的社保文件</DialogTitle>
                        </DialogHeader>
                        <div className="space-y-2">
                          {missingUploads.map((item, index) => (
                            <div
                              key={`${item.part}-${item.scheme}-${index}`}
                              className="flex items-center justify-between rounded-md border bg-muted/40 px-3 py-2"
                            >
                              <span>{SCHEME_LABELS[item.scheme]}</span>
                              <Badge>{PART_LABELS[item.part]}</Badge>
                            </div>
                          ))}
                        </div>
                      </DialogContent>
                    </Dialog>
                  )}
                  <Dialog
                    open={batchDialogOpen}
                    onOpenChange={(open) => {
                      setBatchDialogOpen(open);
                      if (!open) {
                        setBatchDrafts([]);
                        if (batchFileInputRef.current) {
                          batchFileInputRef.current.value = "";
                        }
                      }
                    }}
                  >
                    <DialogTrigger asChild>
                      <Button variant="outline" disabled={!selectedPeriodId}>
                        选择文件
                      </Button>
                    </DialogTrigger>
                    <DialogContent className="max-w-[95vw] sm:max-w-4xl max-h-[90vh] overflow-y-auto">
                      <DialogHeader>
                        <DialogTitle>批量上传社保文件</DialogTitle>
                        <DialogDescription>
                          一次选择多份社保局导出的明细文件，并为每个文件确认扣款部分与险种。
                        </DialogDescription>
                      </DialogHeader>
                      <input
                        ref={batchFileInputRef}
                        type="file"
                        accept=".xlsx,.xls,.csv"
                        multiple
                        className="hidden"
                        onChange={(event) =>
                          handleBatchFilesChange(event.target.files)
                        }
                      />
                      <div className="flex flex-wrap items-center gap-2">
                        <Button
                          onClick={() => batchFileInputRef.current?.click()}
                          variant="outline"
                        >
                          选择文件
                        </Button>
                        {batchDrafts.length > 0 && (
                          <Button
                            variant="ghost"
                            onClick={() => {
                              setBatchDrafts([]);
                              if (batchFileInputRef.current) {
                                batchFileInputRef.current.value = "";
                              }
                            }}
                          >
                            清空
                          </Button>
                        )}
                      </div>
                      {batchDrafts.length === 0 ? (
                        <p className="text-sm text-muted-foreground">
                          尚未选择文件，请点击“选择文件”进行批量导入。
                        </p>
                      ) : (
                        <ScrollArea className="h-72 rounded-md border">
                          <Table>
                            <TableHeader>
                              <TableRow>
                                <TableHead className="w-12">序号</TableHead>
                                <TableHead>文件名</TableHead>
                                <TableHead className="w-32">扣款部分</TableHead>
                                <TableHead className="w-56">险种</TableHead>
                              </TableRow>
                            </TableHeader>
                            <TableBody>
                              {batchDrafts.map((item, index) => {
                                const availableOptions =
                                  item.part &&
                                  (item.part === "personal" ||
                                    item.part === "unit")
                                    ? SCHEME_OPTIONS.filter((option) =>
                                        option.allowedParts.includes(
                                          item.part as Part,
                                        ),
                                      )
                                    : SCHEME_OPTIONS;
                                return (
                                  <TableRow
                                    key={`${item.file.name}-${index}`}
                                  >
                                    <TableCell>{index + 1}</TableCell>
                                    <TableCell className="max-w-xs truncate">
                                      {item.file.name}
                                    </TableCell>
                                    <TableCell>
                                      <Select
                                        value={item.part || ""}
                                        onValueChange={(value: Part) =>
                                          handleBatchFieldChange(
                                            index,
                                            "part",
                                            value,
                                          )
                                        }
                                      >
                                        <SelectTrigger>
                                          <SelectValue placeholder="选择部分" />
                                        </SelectTrigger>
                                        <SelectContent>
                                          <SelectItem value="personal">
                                            个人
                                          </SelectItem>
                                          <SelectItem value="unit">
                                            单位
                                          </SelectItem>
                                        </SelectContent>
                                      </Select>
                                    </TableCell>
                                    <TableCell>
                                      <Select
                                        value={item.scheme || ""}
                                        onValueChange={(value: Scheme) =>
                                          handleBatchFieldChange(
                                            index,
                                            "scheme",
                                            value,
                                          )
                                        }
                                      >
                                        <SelectTrigger>
                                          <SelectValue placeholder="选择险种" />
                                        </SelectTrigger>
                                        <SelectContent>
                                          {availableOptions.map((option) => (
                                            <SelectItem
                                              key={option.value}
                                              value={option.value}
                                            >
                                              {option.label}
                                            </SelectItem>
                                          ))}
                                        </SelectContent>
                                      </Select>
                                    </TableCell>
                                  </TableRow>
                                );
                              })}
                            </TableBody>
                          </Table>
                        </ScrollArea>
                      )}
                      <DialogFooter>
                        <Button
                          onClick={handleBatchUploadSubmit}
                          disabled={
                            batchDrafts.length === 0 || batchUploading
                          }
                        >
                          {batchUploading ? "上传中..." : "开始上传"}
                        </Button>
                      </DialogFooter>
                    </DialogContent>
                  </Dialog>
                  {batchDrafts.length > 0 && batchDrafts.every(item => item.part && item.scheme) && (
                    <Button
                      onClick={handleBatchUploadSubmit}
                      disabled={batchUploading}
                      className="bg-blue-600 hover:bg-blue-700"
                    >
                      {batchUploading ? "上传中..." : "上传社保文件"}
                    </Button>
                  )}
                  {selectedPeriodId && missingUploads.length === 0 && normalFiles.length > 0 && (
                    <Button
                      onClick={handleProcess}
                      disabled={
                        !selectedPeriodId ||
                        processing
                      }
                      className="bg-green-600 hover:bg-green-700"
                    >
                      {processing ? "处理中..." : "处理社保数据"}
                    </Button>
                  )}
                  <Button
                    variant="destructive"
                    disabled={!selectedPeriodId || clearingFiles}
                    onClick={handleClearFiles}
                  >
                    {clearingFiles ? "清空中..." : "清空记录"}
                  </Button>
                  </div>
                </div>

                {batchDrafts.length > 0 && (
                  <div className="border rounded-lg p-4 bg-blue-50/50">
                    <h4 className="text-sm font-medium mb-2 text-blue-700">
                      已选择 {batchDrafts.length} 个社保文件:
                    </h4>
                    <ScrollArea className="h-32">
                      <div className="space-y-1">
                        {batchDrafts.map((item, index) => (
                          <div
                            key={index}
                            className="flex items-center justify-between text-xs p-2 bg-white border rounded"
                          >
                            <span className="truncate flex-1">{item.file.name}</span>
                            <div className="flex items-center gap-2 ml-2">
                              {item.part && item.scheme && (
                                <span className="text-xs bg-green-100 text-green-700 px-2 py-1 rounded">
                                  已配置
                                </span>
                              )}
                            </div>
                          </div>
                        ))}
                      </div>
                    </ScrollArea>
                  </div>
                )}

                <div className="space-y-3">
                    <div className="flex items-center justify-between">
                      <h3 className="text-sm font-semibold">
                        社保文件列表
                      </h3>
                      {missingUploads.length > 0 ? (
                        <Badge variant="destructive">
                          缺少 {missingUploads.length} 份
                        </Badge>
                      ) : normalFiles.length > 0 ? (
                        <Badge variant="default" className="bg-green-100 text-green-700">可处理数据</Badge>
                      ) : (
                        <Badge variant="outline">暂无文件</Badge>
                      )}
                    </div>
                    <ScrollArea className="h-60 rounded-md border bg-background">
                      <Table>
                        <TableHeader>
                          <TableRow>
                            <TableHead>险种</TableHead>
                            <TableHead>部分</TableHead>
                            <TableHead>记录数</TableHead>
                            <TableHead>上传时间</TableHead>
                          </TableRow>
                        </TableHeader>
                        <TableBody>
                          {normalFiles.length === 0 ? (
                            <TableRow>
                              <TableCell
                                colSpan={4}
                                className="h-24 text-center text-muted-foreground"
                              >
                                暂无数据，请上传文件。
                              </TableCell>
                            </TableRow>
                          ) : (
                            normalFiles
                              .slice()
                              .sort(
                                (a, b) =>
                                  SCHEME_ORDER.indexOf(a.scheme) -
                                  SCHEME_ORDER.indexOf(b.scheme),
                              )
                              .map((file) => (
                                <TableRow key={file.id}>
                                  <TableCell>{SCHEME_LABELS[file.scheme]}</TableCell>
                                  <TableCell>{PART_LABELS[file.part]}</TableCell>
                                  <TableCell>{file.rows}</TableCell>
                                  <TableCell>{formatDate(file.uploaded_at)}</TableCell>
                                </TableRow>
                              ))
                          )}
                        </TableBody>
                      </Table>
                    </ScrollArea>


                    <div className="text-sm text-muted-foreground border rounded-lg p-3">
                      <ul className="list-disc list-inside space-y-1 text-xs">
                        <li>点击&ldquo;选择文件&rdquo;按钮批量选择社保文件</li>
                        <li>为每个文件选择对应的险种和缴费部分</li>
                        <li>确认文件信息无误后点击&ldquo;批量上传&rdquo;</li>
                        <li>支持Excel(.xlsx/.xls)和CSV格式文件</li>
                      </ul>
                    </div>
                  </div>
              </CardContent>
            </Card>

            {/* 补退文件上传卡片 */}
            <Card>
              <CardHeader>
                <div className="flex flex-wrap items-center justify-between gap-3">
                  <div className="space-y-1.5">
                    <CardTitle className="text-orange-700">补退文件上传</CardTitle>
                    <CardDescription>
                      上传补退文件，系统将自动识别险种和缴费部分，并将补退金额累加到现有扣款明细中。
                    </CardDescription>
                  </div>
                  <div className="flex items-center gap-3">
                    <span className="text-sm text-muted-foreground">
                      已选择 {selectedAdjustmentFiles.length} 个补退文件
                    </span>
                  </div>
                </div>
              </CardHeader>
              <CardContent className="space-y-6">
                <div className="flex justify-end">
                  <div className="flex gap-2">
                  <Dialog
                    open={adjustmentDialogOpen}
                    onOpenChange={(open) => {
                      setAdjustmentDialogOpen(open);
                      if (!open) {
                        setSelectedAdjustmentFiles([]);
                        if (adjustmentFileInputRef.current) {
                          adjustmentFileInputRef.current.value = "";
                        }
                      }
                    }}
                  >
                    <DialogTrigger asChild>
                      <Button variant="outline" disabled={!selectedPeriodId}>
                        选择文件
                      </Button>
                    </DialogTrigger>
                    <DialogContent className="max-w-[95vw] sm:max-w-4xl max-h-[90vh] overflow-y-auto">
                      <DialogHeader>
                        <DialogTitle>批量上传补退文件</DialogTitle>
                        <DialogDescription>
                          一次选择多份补退文件，系统将自动识别险种和缴费部分。
                        </DialogDescription>
                      </DialogHeader>
                      <input
                        ref={adjustmentFileInputRef}
                        type="file"
                        accept=".xlsx,.xls,.csv"
                        multiple
                        className="hidden"
                        onChange={(e) => {
                          if (e.target.files) {
                            setSelectedAdjustmentFiles(Array.from(e.target.files));
                          }
                        }}
                      />
                      <div className="flex flex-wrap items-center gap-2">
                        <Button
                          onClick={() => adjustmentFileInputRef.current?.click()}
                          variant="outline"
                        >
                          选择文件
                        </Button>
                        {selectedAdjustmentFiles.length > 0 && (
                          <Button
                            variant="ghost"
                            onClick={() => {
                              setSelectedAdjustmentFiles([]);
                              if (adjustmentFileInputRef.current) {
                                adjustmentFileInputRef.current.value = "";
                              }
                            }}
                          >
                            清空
                          </Button>
                        )}
                      </div>
                      {selectedAdjustmentFiles.length === 0 ? (
                        <p className="text-sm text-muted-foreground">
                          尚未选择文件，请点击&ldquo;选择文件&rdquo;进行批量导入。
                        </p>
                      ) : (
                        <ScrollArea className="h-72 rounded-md border">
                          <Table>
                            <TableHeader>
                              <TableRow>
                                <TableHead className="w-12">序号</TableHead>
                                <TableHead>文件名</TableHead>
                                <TableHead>大小</TableHead>
                                <TableHead>状态</TableHead>
                              </TableRow>
                            </TableHeader>
                            <TableBody>
                              {selectedAdjustmentFiles.map((file, index) => (
                                <TableRow key={`${file.name}-${index}`}>
                                  <TableCell>{index + 1}</TableCell>
                                  <TableCell className="max-w-xs truncate">
                                    {file.name}
                                  </TableCell>
                                  <TableCell>
                                    {(file.size / 1024 / 1024).toFixed(2)} MB
                                  </TableCell>
                                  <TableCell>
                                    <span className="text-xs bg-green-100 text-green-700 px-2 py-1 rounded">
                                      就绪
                                    </span>
                                  </TableCell>
                                </TableRow>
                              ))}
                            </TableBody>
                          </Table>
                        </ScrollArea>
                      )}
                      <DialogFooter>
                        <Button
                          onClick={async () => {
                            await handleAdjustmentUpload();
                            setAdjustmentDialogOpen(false);
                          }}
                          disabled={
                            selectedAdjustmentFiles.length === 0 || adjustmentUploading
                          }
                          className="bg-orange-600 hover:bg-orange-700"
                        >
                          {adjustmentUploading ? "上传中..." : "开始上传"}
                        </Button>
                      </DialogFooter>
                    </DialogContent>
                  </Dialog>
                    {adjustmentFiles.length > 0 && (
                      <Button
                        onClick={handleProcessAdjustments}
                        disabled={adjustmentProcessing}
                        className="bg-green-600 hover:bg-green-700"
                      >
                        {adjustmentProcessing ? "处理中..." : "处理补退数据"}
                      </Button>
                    )}
                    <Button
                      variant="destructive"
                      disabled={!selectedPeriodId || clearingAdjustments}
                      onClick={handleClearAdjustments}
                    >
                      {clearingAdjustments ? "清空中..." : "清空记录"}
                    </Button>
                  </div>
                </div>

                <div className="space-y-3">
                  <div className="flex items-center justify-between">
                    <h3 className="text-sm font-semibold">
                      补退文件列表
                    </h3>
                    <Badge variant="outline" className="border-orange-200 text-orange-700">
                      {adjustmentFiles.length} 个文件
                    </Badge>
                  </div>
                  <ScrollArea className="h-60 rounded-md border bg-background">
                    <Table>
                      <TableHeader>
                        <TableRow>
                          <TableHead>原始文件名</TableHead>
                          <TableHead>险种</TableHead>
                          <TableHead>部分</TableHead>
                          <TableHead>记录数</TableHead>
                          <TableHead>上传时间</TableHead>
                        </TableRow>
                      </TableHeader>
                      <TableBody>
                        {adjustmentFiles.length === 0 ? (
                          <TableRow>
                            <TableCell
                              colSpan={5}
                              className="h-24 text-center text-muted-foreground"
                            >
                              暂无补退文件，请上传补退文件。
                            </TableCell>
                          </TableRow>
                        ) : (
                          adjustmentFiles
                            .slice()
                            .sort(
                              (a, b) =>
                                SCHEME_ORDER.indexOf(a.scheme) -
                                SCHEME_ORDER.indexOf(b.scheme),
                            )
                            .map((file) => (
                              <TableRow key={file.id}>
                                <TableCell className="max-w-xs truncate" title={file.original_name}>
                                  {file.original_name}
                                </TableCell>
                                <TableCell>{SCHEME_LABELS[file.scheme]}</TableCell>
                                <TableCell>{PART_LABELS[file.part]}</TableCell>
                                <TableCell>{file.rows}</TableCell>
                                <TableCell>{formatDate(file.uploaded_at)}</TableCell>
                              </TableRow>
                            ))
                        )}
                      </TableBody>
                    </Table>
                  </ScrollArea>

                  <div className="text-sm text-muted-foreground border rounded-lg p-3">
                    <p className="mb-2 font-medium">文件命名要求：</p>
                    <ul className="list-disc list-inside space-y-1 text-xs">
                      <li>包含险种名称：养老保险、失业保险、工伤保险、医疗保险、大额医疗</li>
                      <li>包含缴费部分：(个人缴纳)、(单位缴纳)或个人缴纳、单位缴纳</li>
                      <li>示例：张英俊职工基本养老保险(个人缴纳)_2025-01至2025-01_未申报信息明细.xlsx</li>
                    </ul>
                  </div>

                </div>
              </CardContent>
            </Card>

          </TabsContent>

          <TabsContent value="result" className="space-y-6">
            <Card>
              <CardHeader>
                <CardTitle>社保总表汇总</CardTitle>
                <CardDescription>
                  每个险种的参保人数、基数合计与应缴金额。
                </CardDescription>
              </CardHeader>
              <CardContent>
                {periodDataLoading ? (
                  <Skeleton className="h-40 w-full" />
                ) : (
                  <Table>
                    <TableHeader>
                      <TableRow>
                        <TableHead>险种</TableHead>
                        <TableHead>部分</TableHead>
                        <TableHead>人数</TableHead>
                        <TableHead>缴费基数合计</TableHead>
                        <TableHead>应缴金额合计</TableHead>
                      </TableRow>
                    </TableHeader>
                    <TableBody>
                      {summary.length === 0 ? (
                        <TableRow>
                          <TableCell
                            colSpan={5}
                            className="h-24 text-center text-muted-foreground"
                          >
                            暂无处理数据，请先执行&ldquo;处理社保数据&rdquo;。
                          </TableCell>
                        </TableRow>
                      ) : (
                        <>
                          {summary.map((item) => (
                            <TableRow
                              key={item.id}
                              className={item.is_adjustment ? "bg-orange-50/50 border-l-4 border-l-orange-400" : ""}
                            >
                              <TableCell>
                                <div className="flex items-center gap-2">
                                  <button
                                    onClick={() => handleSchemeClick(item.scheme, item.part, item.is_adjustment)}
                                    className="group inline-flex items-center gap-1 font-medium text-foreground transition-colors hover:text-primary focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2 rounded-sm px-1 py-0.5"
                                  >
                                    <span className="group-hover:text-primary transition-colors">
                                      {SCHEME_LABELS[item.scheme]}
                                    </span>
                                    <svg
                                      className="w-3 h-3 opacity-0 group-hover:opacity-100 transition-opacity text-primary"
                                      fill="none"
                                      stroke="currentColor"
                                      viewBox="0 0 24 24"
                                    >
                                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
                                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z" />
                                    </svg>
                                  </button>
                                  {item.is_adjustment && (
                                    <Badge variant="outline" className="text-xs bg-orange-100 text-orange-700 border-orange-300">
                                      补退
                                    </Badge>
                                  )}
                                </div>
                              </TableCell>
                              <TableCell>{PART_LABELS[item.part]}</TableCell>
                              <TableCell>{item.headcount}</TableCell>
                              <TableCell>{formatCurrency(item.base_total)}</TableCell>
                              <TableCell>{formatCurrency(item.amount_total)}</TableCell>
                            </TableRow>
                          ))}
                          {summary.length > 0 && (
                            <TableRow className="border-t-2 border-primary bg-muted/50 font-semibold">
                              <TableCell>合计</TableCell>
                              <TableCell>-</TableCell>
                              <TableCell>
                                {summary.reduce((total, item) => total + item.headcount, 0)}
                              </TableCell>
                              <TableCell>
                                {formatCurrency(summary.reduce((total, item) => total + item.base_total, 0))}
                              </TableCell>
                              <TableCell>
                                {formatCurrency(summary.reduce((total, item) => total + item.amount_total, 0))}
                              </TableCell>
                            </TableRow>
                          )}
                        </>
                      )}
                    </TableBody>
                  </Table>
                )}
              </CardContent>
            </Card>

            <Card>
              <CardHeader>
                <CardTitle>扣款明细</CardTitle>
                <CardDescription>
                  生成的个人与单位扣款数据，可用于导出报表及复核。
                </CardDescription>
              </CardHeader>
              <CardContent>
                {periodDataLoading ? (
                  <Skeleton className="h-80 w-full" />
                ) : (
                  <Tabs defaultValue="personal">
                    <TabsList className="mb-4">
                      <TabsTrigger value="personal">个人部分</TabsTrigger>
                      <TabsTrigger value="unit">单位部分</TabsTrigger>
                    </TabsList>
                    <TabsContent value="personal" className="space-y-3">
                      <div className="space-y-3">
                        <div className="grid gap-3 md:grid-cols-3">
                          <div>
                            <Label htmlFor="personal-search-text" className="text-xs">姓名/身份证号查询</Label>
                            <Input
                              id="personal-search-text"
                              placeholder="输入姓名或身份证号"
                              value={personalSearchText}
                              onChange={(e) => setPersonalSearchText(e.target.value)}
                              className="h-8 mt-1"
                            />
                          </div>
                          <div>
                            <Label htmlFor="personal-search-dept" className="text-xs">按部门筛选</Label>
                            <Select value={personalSearchDept} onValueChange={setPersonalSearchDept}>
                              <SelectTrigger className="h-8 mt-1">
                                <SelectValue placeholder="选择部门" />
                              </SelectTrigger>
                              <SelectContent>
                                <SelectItem value="__all__">全部部门</SelectItem>
                                {personalDepartments.map((dept) => (
                                  <SelectItem key={dept} value={dept}>
                                    {dept}
                                  </SelectItem>
                                ))}
                              </SelectContent>
                            </Select>
                          </div>
                          <div>
                            <Label className="text-xs text-transparent">操作</Label>
                            <Button
                              onClick={() => handleExport("personal")}
                              disabled={
                                personalCharges.length === 0 ||
                                exportingPart === "personal"
                              }
                              className="h-8 w-full mt-1 text-sm bg-black text-white hover:bg-gray-800"
                            >
                              {exportingPart === "personal"
                                ? "导出中..."
                                : "导出 Excel"}
                            </Button>
                          </div>
                        </div>
                        <div className="text-sm text-muted-foreground">
                          显示 {filteredPersonalCharges.length} / {personalCharges.length} 条记录
                        </div>
                      </div>
                      <ScrollArea className="h-[420px] rounded-md border">
                        <Table>
                          <TableHeader>
                            <TableRow>
                              <TableHead className="w-16">序号</TableHead>
                              <TableHead>姓名</TableHead>
                              <TableHead>部门</TableHead>
                              <TableHead>基数</TableHead>
                              <TableHead>养老</TableHead>
                              <TableHead>医疗+生育</TableHead>
                              <TableHead>大额医疗</TableHead>
                              <TableHead>失业</TableHead>
                              <TableHead>小计</TableHead>
                            </TableRow>
                          </TableHeader>
                          <TableBody>
                            {filteredPersonalCharges.length === 0 ? (
                              <TableRow>
                                <TableCell
                                  colSpan={9}
                                  className="h-24 text-center text-muted-foreground"
                                >
                                  {personalCharges.length === 0 ? "暂无数据。" : "没有符合条件的记录。"}
                                </TableCell>
                              </TableRow>
                            ) : (
                              <>
                                {(() => {
                                  let groupIndex = 0;
                                  return filteredPersonalCharges.map((row) => {
                                    if (row.isFirstInGroup) {
                                      groupIndex++;
                                    }
                                    return (
                                      <TableRow
                                        key={row.id}
                                        className={row.is_adjustment ? "bg-orange-50/50 border-l-4 border-l-orange-400" : ""}
                                      >
                                        <TableCell className="text-center">
                                          {row.isFirstInGroup ? groupIndex : ""}
                                        </TableCell>
                                        <TableCell>
                                          <div className="flex items-center gap-2">
                                            <span>{row.name}</span>
                                            {row.is_adjustment && (
                                              <Badge variant="outline" className="text-xs bg-orange-100 text-orange-700 border-orange-300">
                                                补退
                                              </Badge>
                                            )}
                                          </div>
                                        </TableCell>
                                    <TableCell>{row.department || "-"}</TableCell>
                                    <TableCell>{formatCurrency(row.base)}</TableCell>
                                    <TableCell>{formatCurrency(row.pension)}</TableCell>
                                    <TableCell>
                                      {formatCurrency(row.medical_maternity)}
                                    </TableCell>
                                    <TableCell>
                                      {formatCurrency(row.serious_illness)}
                                    </TableCell>
                                    <TableCell>
                                      {formatCurrency(row.unemployment)}
                                    </TableCell>
                                    <TableCell>{formatCurrency(row.subtotal)}</TableCell>
                                  </TableRow>
                                    );
                                  });
                                })()}
                                {filteredPersonalCharges.length > 0 && (
                                  <TableRow className="border-t-2 border-primary bg-muted/50 font-semibold">
                                    <TableCell className="text-center">-</TableCell>
                                    <TableCell>合计</TableCell>
                                    <TableCell>-</TableCell>
                                    <TableCell>
                                      {formatCurrency(filteredPersonalCharges.reduce((total, row) => total + row.base, 0))}
                                    </TableCell>
                                    <TableCell>
                                      {formatCurrency(filteredPersonalCharges.reduce((total, row) => total + row.pension, 0))}
                                    </TableCell>
                                    <TableCell>
                                      {formatCurrency(filteredPersonalCharges.reduce((total, row) => total + row.medical_maternity, 0))}
                                    </TableCell>
                                    <TableCell>
                                      {formatCurrency(filteredPersonalCharges.reduce((total, row) => total + row.serious_illness, 0))}
                                    </TableCell>
                                    <TableCell>
                                      {formatCurrency(filteredPersonalCharges.reduce((total, row) => total + row.unemployment, 0))}
                                    </TableCell>
                                    <TableCell>
                                      {formatCurrency(filteredPersonalCharges.reduce((total, row) => total + row.subtotal, 0))}
                                    </TableCell>
                                  </TableRow>
                                )}
                              </>
                            )}
                          </TableBody>
                        </Table>
                      </ScrollArea>
                    </TabsContent>
                    <TabsContent value="unit" className="space-y-3">
                      <div className="space-y-3">
                        <div className="grid gap-3 md:grid-cols-3">
                          <div>
                            <Label htmlFor="unit-search-text" className="text-xs">姓名/身份证号查询</Label>
                            <Input
                              id="unit-search-text"
                              placeholder="输入姓名或身份证号"
                              value={unitSearchText}
                              onChange={(e) => setUnitSearchText(e.target.value)}
                              className="h-8 mt-1"
                            />
                          </div>
                          <div>
                            <Label htmlFor="unit-search-dept" className="text-xs">按部门筛选</Label>
                            <Select value={unitSearchDept} onValueChange={setUnitSearchDept}>
                              <SelectTrigger className="h-8 mt-1">
                                <SelectValue placeholder="选择部门" />
                              </SelectTrigger>
                              <SelectContent>
                                <SelectItem value="__all__">全部部门</SelectItem>
                                {unitDepartments.map((dept) => (
                                  <SelectItem key={dept} value={dept}>
                                    {dept}
                                  </SelectItem>
                                ))}
                              </SelectContent>
                            </Select>
                          </div>
                          <div>
                            <Label className="text-xs text-transparent">操作</Label>
                            <Button
                              onClick={() => handleExport("unit")}
                              disabled={
                                unitCharges.length === 0 ||
                                exportingPart === "unit"
                              }
                              className="h-8 w-full mt-1 text-sm bg-black text-white hover:bg-gray-800"
                            >
                              {exportingPart === "unit"
                                ? "导出中..."
                                : "导出 Excel"}
                            </Button>
                          </div>
                        </div>
                        <div className="text-sm text-muted-foreground">
                          显示 {filteredUnitCharges.length} / {unitCharges.length} 条记录
                        </div>
                      </div>
                      <ScrollArea className="h-[420px] rounded-md border">
                        <Table>
                          <TableHeader>
                            <TableRow>
                              <TableHead className="w-16">序号</TableHead>
                              <TableHead>姓名</TableHead>
                              <TableHead>部门</TableHead>
                              <TableHead>基数</TableHead>
                              <TableHead>养老</TableHead>
                              <TableHead>医疗+生育</TableHead>
                              <TableHead>工伤</TableHead>
                              <TableHead>失业</TableHead>
                              <TableHead>小计</TableHead>
                            </TableRow>
                          </TableHeader>
                          <TableBody>
                            {filteredUnitCharges.length === 0 ? (
                              <TableRow>
                                <TableCell
                                  colSpan={9}
                                  className="h-24 text-center text-muted-foreground"
                                >
                                  {unitCharges.length === 0 ? "暂无数据。" : "没有符合条件的记录。"}
                                </TableCell>
                              </TableRow>
                            ) : (
                              <>
                                {(() => {
                                  let groupIndex = 0;
                                  return filteredUnitCharges.map((row) => {
                                    if (row.isFirstInGroup) {
                                      groupIndex++;
                                    }
                                    return (
                                      <TableRow
                                        key={row.id}
                                        className={row.is_adjustment ? "bg-orange-50/50 border-l-4 border-l-orange-400" : ""}
                                      >
                                        <TableCell className="text-center">
                                          {row.isFirstInGroup ? groupIndex : ""}
                                        </TableCell>
                                        <TableCell>
                                          <div className="flex items-center gap-2">
                                            <span>{row.name}</span>
                                            {row.is_adjustment && (
                                              <Badge variant="outline" className="text-xs bg-orange-100 text-orange-700 border-orange-300">
                                                补退
                                              </Badge>
                                            )}
                                          </div>
                                        </TableCell>
                                    <TableCell>{row.department || "-"}</TableCell>
                                    <TableCell>{formatCurrency(row.base)}</TableCell>
                                    <TableCell>{formatCurrency(row.pension)}</TableCell>
                                    <TableCell>
                                      {formatCurrency(row.medical_maternity)}
                                    </TableCell>
                                    <TableCell>
                                      {formatCurrency(row.injury)}
                                    </TableCell>
                                    <TableCell>
                                      {formatCurrency(row.unemployment)}
                                    </TableCell>
                                    <TableCell>{formatCurrency(row.subtotal)}</TableCell>
                                  </TableRow>
                                    );
                                  });
                                })()}
                                {filteredUnitCharges.length > 0 && (
                                  <TableRow className="border-t-2 border-primary bg-muted/50 font-semibold">
                                    <TableCell className="text-center">-</TableCell>
                                    <TableCell>合计</TableCell>
                                    <TableCell>-</TableCell>
                                    <TableCell>
                                      {formatCurrency(filteredUnitCharges.reduce((total, row) => total + row.base, 0))}
                                    </TableCell>
                                    <TableCell>
                                      {formatCurrency(filteredUnitCharges.reduce((total, row) => total + row.pension, 0))}
                                    </TableCell>
                                    <TableCell>
                                      {formatCurrency(filteredUnitCharges.reduce((total, row) => total + row.medical_maternity, 0))}
                                    </TableCell>
                                    <TableCell>
                                      {formatCurrency(filteredUnitCharges.reduce((total, row) => total + row.injury, 0))}
                                    </TableCell>
                                    <TableCell>
                                      {formatCurrency(filteredUnitCharges.reduce((total, row) => total + row.unemployment, 0))}
                                    </TableCell>
                                    <TableCell>
                                      {formatCurrency(filteredUnitCharges.reduce((total, row) => total + row.subtotal, 0))}
                                    </TableCell>
                                  </TableRow>
                                )}
                              </>
                            )}
                          </TableBody>
                        </Table>
                      </ScrollArea>
                    </TabsContent>
                  </Tabs>
                )}
              </CardContent>
            </Card>
          </TabsContent>
        </Tabs>
      </div>

      {/* 险种明细模态窗 */}
      <Dialog open={schemeModalOpen} onOpenChange={setSchemeModalOpen}>
        <DialogContent className="max-w-[95vw] sm:max-w-5xl max-h-[90vh] overflow-y-auto">
          <DialogHeader>
            <DialogTitle>
              {selectedScheme && selectedPart && (
                <>
                  {SCHEME_LABELS[selectedScheme]} - {selectedPart === "personal" ? "个人" : "单位"} 明细
                </>
              )}
            </DialogTitle>
            <DialogDescription>
              查看该险种下所有人员的缴费基数和应缴金额明细
            </DialogDescription>
          </DialogHeader>

          <div className="space-y-4">
            <div className="flex justify-between items-center">
              <span className="text-sm text-muted-foreground">
                共 {schemeCharges.length} 人
              </span>
              <Button
                size="sm"
                onClick={handleSchemeExport}
                disabled={schemeExporting || schemeCharges.length === 0}
                className="bg-black text-white hover:bg-gray-800"
              >
                {schemeExporting ? "导出中..." : "导出 Excel"}
              </Button>
            </div>

            <ScrollArea className="h-[40vh] min-h-[300px] max-h-[420px] rounded-md border">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead className="w-16">序号</TableHead>
                    <TableHead>姓名</TableHead>
                    <TableHead>身份证号</TableHead>
                    <TableHead>部门</TableHead>
                    <TableHead>缴费基数</TableHead>
                    <TableHead>应缴金额</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {schemeChargesLoading ? (
                    <TableRow>
                      <TableCell colSpan={6} className="h-24 text-center">
                        <Skeleton className="h-4 w-full" />
                      </TableCell>
                    </TableRow>
                  ) : schemeCharges.length === 0 ? (
                    <TableRow>
                      <TableCell
                        colSpan={6}
                        className="h-24 text-center text-muted-foreground"
                      >
                        暂无数据
                      </TableCell>
                    </TableRow>
                  ) : (
                    <>
                      {schemeCharges.map((row, index) => (
                        <TableRow key={`${row.id_number}-${index}`}>
                          <TableCell className="text-center">{index + 1}</TableCell>
                          <TableCell>{row.name}</TableCell>
                          <TableCell className="font-mono text-xs">{row.id_number}</TableCell>
                          <TableCell>{row.department || "-"}</TableCell>
                          <TableCell>{formatCurrency(row.base)}</TableCell>
                          <TableCell>{formatCurrency(row.amount)}</TableCell>
                        </TableRow>
                      ))}
                      {schemeCharges.length > 0 && (
                        <TableRow className="border-t-2 border-primary bg-muted/50 font-semibold">
                          <TableCell className="text-center">-</TableCell>
                          <TableCell>合计</TableCell>
                          <TableCell>-</TableCell>
                          <TableCell>-</TableCell>
                          <TableCell>
                            {formatCurrency(schemeCharges.reduce((total, row) => total + row.base, 0))}
                          </TableCell>
                          <TableCell>
                            {formatCurrency(schemeCharges.reduce((total, row) => total + row.amount, 0))}
                          </TableCell>
                        </TableRow>
                      )}
                    </>
                  )}
                </TableBody>
              </Table>
            </ScrollArea>
          </div>

          <DialogFooter>
            <Button variant="outline" onClick={() => setSchemeModalOpen(false)}>
              关闭
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Password Change Dialog */}
      <Dialog open={passwordDialogOpen} onOpenChange={setPasswordDialogOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>修改密码</DialogTitle>
            <DialogDescription>
              请输入您的当前密码和新密码
            </DialogDescription>
          </DialogHeader>
          <div className="space-y-4">
            <div>
              <Label htmlFor="old-password">当前密码</Label>
              <Input
                id="old-password"
                type="password"
                value={oldPassword}
                onChange={(e) => setOldPassword(e.target.value)}
                placeholder="请输入当前密码"
              />
            </div>
            <div>
              <Label htmlFor="new-password">新密码</Label>
              <Input
                id="new-password"
                type="password"
                value={newPassword}
                onChange={(e) => setNewPassword(e.target.value)}
                placeholder="请输入新密码（至少6位）"
              />
            </div>
            <div>
              <Label htmlFor="confirm-password">确认新密码</Label>
              <Input
                id="confirm-password"
                type="password"
                value={confirmPassword}
                onChange={(e) => setConfirmPassword(e.target.value)}
                placeholder="请再次输入新密码"
              />
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setPasswordDialogOpen(false)}>
              取消
            </Button>
            <Button onClick={handlePasswordChange} disabled={changingPassword}>
              {changingPassword ? "修改中..." : "确认修改"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Audit Logs Dialog */}
      <Dialog open={auditDialogOpen} onOpenChange={setAuditDialogOpen}>
        <DialogContent className="max-w-6xl max-h-[90vh] overflow-hidden">
          <DialogHeader>
            <DialogTitle>审计日志</DialogTitle>
            <DialogDescription>
              查看系统操作记录和统计信息
            </DialogDescription>
          </DialogHeader>
          <div className="space-y-4">
            {auditStats && (
              <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
                <Card className="p-3">
                  <div className="text-sm text-muted-foreground">总日志数</div>
                  <div className="text-lg font-semibold">{auditStats.stats.total_events}</div>
                </Card>
                <Card className="p-3">
                  <div className="text-sm text-muted-foreground">成功操作</div>
                  <div className="text-lg font-semibold text-green-600">{auditStats.success_count}</div>
                </Card>
                <Card className="p-3">
                  <div className="text-sm text-muted-foreground">失败操作</div>
                  <div className="text-lg font-semibold text-red-600">{auditStats.failure_count}</div>
                </Card>
                <Card className="p-3">
                  <div className="text-sm text-muted-foreground">活跃用户</div>
                  <div className="text-lg font-semibold">{auditStats.unique_users}</div>
                </Card>
              </div>
            )}
            <ScrollArea className="h-[400px]">
              {loadingAudit ? (
                <div className="flex items-center justify-center py-8">
                  <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary"></div>
                </div>
              ) : (
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>时间</TableHead>
                      <TableHead>用户</TableHead>
                      <TableHead>操作</TableHead>
                      <TableHead>状态</TableHead>
                      <TableHead>IP地址</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {auditLogs.map((log) => (
                      <TableRow key={log.id}>
                        <TableCell className="text-sm">
                          {new Date(log.timestamp).toLocaleString('zh-CN')}
                        </TableCell>
                        <TableCell>{log.username || '系统'}</TableCell>
                        <TableCell>{log.action}</TableCell>
                        <TableCell>
                          <Badge variant={log.status === 'SUCCESS' ? 'default' : 'destructive'}>
                            {log.status === 'SUCCESS' ? '成功' : '失败'}
                          </Badge>
                        </TableCell>
                        <TableCell className="text-sm">{log.ip_address}</TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              )}
            </ScrollArea>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setAuditDialogOpen(false)}>
              关闭
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* System Monitoring Dialog */}
      <Dialog open={monitoringDialogOpen} onOpenChange={setMonitoringDialogOpen}>
        <DialogContent className="max-w-4xl max-h-[90vh] overflow-hidden">
          <DialogHeader>
            <DialogTitle>系统监控</DialogTitle>
            <DialogDescription>
              查看系统运行状态和性能指标
            </DialogDescription>
          </DialogHeader>
          <ScrollArea className="h-[500px]">
            {loadingMonitoring ? (
              <div className="flex items-center justify-center py-8">
                <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary"></div>
              </div>
            ) : (
              <div className="space-y-6">
                {/* System Metrics */}
                {systemMetrics && (
                  <div>
                    <h3 className="text-lg font-semibold mb-3">系统指标</h3>
                    <div className="grid grid-cols-2 md:grid-cols-3 gap-4">
                      <Card className="p-3">
                        <div className="text-sm text-muted-foreground">CPU使用率</div>
                        <div className="text-lg font-semibold">{systemMetrics.cpu_usage.toFixed(1)}%</div>
                      </Card>
                      <Card className="p-3">
                        <div className="text-sm text-muted-foreground">内存使用率</div>
                        <div className="text-lg font-semibold">{systemMetrics.memory_usage.toFixed(1)}%</div>
                      </Card>
                      <Card className="p-3">
                        <div className="text-sm text-muted-foreground">磁盘使用率</div>
                        <div className="text-lg font-semibold">{systemMetrics.disk_usage.toFixed(1)}%</div>
                      </Card>
                      <Card className="p-3">
                        <div className="text-sm text-muted-foreground">活跃连接</div>
                        <div className="text-lg font-semibold">{systemMetrics.active_connections}</div>
                      </Card>
                      <Card className="p-3">
                        <div className="text-sm text-muted-foreground">运行时间</div>
                        <div className="text-lg font-semibold">
                          {Math.floor(systemMetrics.uptime_seconds / 3600)}小时
                        </div>
                      </Card>
                      <Card className="p-3">
                        <div className="text-sm text-muted-foreground">数据库连接</div>
                        <div className="text-lg font-semibold">{systemMetrics.database_connections}</div>
                      </Card>
                    </div>
                  </div>
                )}

                {/* Database Status */}
                {databaseStatus && (
                  <div>
                    <h3 className="text-lg font-semibold mb-3">数据库状态</h3>
                    <div className="space-y-3">
                      <div className="flex items-center gap-2">
                        <span className="text-sm text-muted-foreground">状态:</span>
                        <Badge variant={databaseStatus.status === 'healthy' ? 'default' : 'destructive'}>
                          {databaseStatus.status === 'healthy' ? '健康' : '异常'}
                        </Badge>
                      </div>
                      <div className="grid grid-cols-2 gap-4">
                        <Card className="p-3">
                          <div className="text-sm text-muted-foreground">连接数</div>
                          <div className="text-lg font-semibold">
                            {databaseStatus.connection_count} / {databaseStatus.max_connections}
                          </div>
                        </Card>
                        {databaseStatus.database_size && (
                          <Card className="p-3">
                            <div className="text-sm text-muted-foreground">数据库大小</div>
                            <div className="text-lg font-semibold">{databaseStatus.database_size}</div>
                          </Card>
                        )}
                      </div>
                      {databaseStatus.tables.length > 0 && (
                        <div>
                          <h4 className="font-medium mb-2">数据表统计</h4>
                          <div className="grid grid-cols-2 md:grid-cols-3 gap-2">
                            {databaseStatus.tables.map((table) => (
                              <div key={table.name} className="flex justify-between text-sm bg-muted/50 p-2 rounded">
                                <span>{table.name}</span>
                                <span>{table.rows} 行</span>
                              </div>
                            ))}
                          </div>
                        </div>
                      )}
                    </div>
                  </div>
                )}

                {/* System Info */}
                {systemInfo && (
                  <div>
                    <h3 className="text-lg font-semibold mb-3">系统信息</h3>
                    <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                      <div className="space-y-2">
                        <div className="flex justify-between">
                          <span className="text-sm text-muted-foreground">主机名:</span>
                          <span className="text-sm">{systemInfo.hostname}</span>
                        </div>
                        <div className="flex justify-between">
                          <span className="text-sm text-muted-foreground">平台:</span>
                          <span className="text-sm">{systemInfo.platform}</span>
                        </div>
                        <div className="flex justify-between">
                          <span className="text-sm text-muted-foreground">CPU核心:</span>
                          <span className="text-sm">{systemInfo.cpu_cores}</span>
                        </div>
                      </div>
                      <div className="space-y-2">
                        <div className="flex justify-between">
                          <span className="text-sm text-muted-foreground">总内存:</span>
                          <span className="text-sm">{(systemInfo.total_memory / 1024 / 1024 / 1024).toFixed(1)} GB</span>
                        </div>
                        <div className="flex justify-between">
                          <span className="text-sm text-muted-foreground">Go版本:</span>
                          <span className="text-sm">{systemInfo.go_version}</span>
                        </div>
                        <div className="flex justify-between">
                          <span className="text-sm text-muted-foreground">应用版本:</span>
                          <span className="text-sm">{systemInfo.version}</span>
                        </div>
                      </div>
                    </div>
                  </div>
                )}
              </div>
            )}
          </ScrollArea>
          <DialogFooter>
            <Button variant="outline" onClick={() => setMonitoringDialogOpen(false)}>
              关闭
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
