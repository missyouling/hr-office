"use client";

import { useState, useEffect, useRef, useMemo, useCallback } from "react";
import {
  Search, Download, Upload, Printer, FileText,
  X, File, Image, Video, FileUp, Share2, Link2,
  Copy, Check, Loader2, Trash, RefreshCw, Save, ChevronLeft, ChevronRight,
  Settings, Tag, Plus
} from "lucide-react";
import { toast } from "sonner";

import {
  fetchDocumentCategories, fetchDocuments, createDocument, updateDocument,
  deleteDocument, uploadDocumentFile, uploadWithOCR,
  batchDownloadDocuments, generateShareLink,
  fetchStorageLocations,
  fetchFieldsBySubCategory,
  fetchColumnConfig, saveColumnConfig,
  setDocumentTags,
  type Document, type DocumentCategory, type DocumentSubCategory, type StorageLocation,
  type OCRExtractResult
} from "@/lib/api";
import { createReportPdf } from "@/lib/pdf";

import { Button } from "@/components/ui/button";
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle, DialogTrigger } from "@/components/ui/dialog";
import { AlertDialog, AlertDialogCancel, AlertDialogContent, AlertDialogDescription, AlertDialogFooter, AlertDialogHeader, AlertDialogTitle } from "@/components/ui/alert-dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Badge } from "@/components/ui/badge";
import { Progress } from "@/components/ui/progress";

import { ArchiveFormRenderer } from "./archive-form-renderer";
import { ArchiveTableRenderer } from "./archive-table-renderer";
import { FolderTree } from "./folder-tree";
import { TagFilter } from "./tag-filter";
import { generateFormSchema, generateTableSchema, type FormFieldSchema, type TableColumnSchema } from "@/lib/archive-schema";

const RETENTION_OPTIONS = [
  { value: "永久", label: "永久" },
  { value: "30年", label: "30年" },
  { value: "10年", label: "10年" },
  { value: "5年", label: "5年" },
];

const SUPPORTED_FILE_TYPES = {
  images: ['.jpg', '.jpeg', '.png', '.gif', '.bmp', '.webp', '.tiff'],
  filteredDocuments: ['.pdf', '.doc', '.docx', '.xls', '.xlsx', '.ppt', '.pptx', '.txt', '.text'],
  drawings: ['.dwg', '.dxf', '.cad'],
  archives: ['.zip', '.rar', '.7z', '.tar', '.gz'],
  text: ['.txt', '.text', '.log', '.md', '.json', '.xml', '.yaml', '.yml', '.csv'],
};

// 待上传文件项
interface PendingFile {
  id: string;
  file: File;
  status: 'pending' | 'uploading' | 'success' | 'error';
  progress: number;
  document?: Partial<Document>;
  error?: string;
}

export function ArchivesManagement() {
  const [categories, setCategories] = useState<DocumentCategory[]>([]);
  const [filteredDocuments, setFilteredDocuments] = useState<Document[]>([]);
  const [total, setTotal] = useState(0);
  const [storageLocations, setStorageLocations] = useState<StorageLocation[]>([]);
  const [page, setPage] = useState(1);
  const [pageSize] = useState(20);
  const [loading, setLoading] = useState(false);
  const [activeTab, setActiveTab] = useState("all");
  const [searchKeyword, setSearchKeyword] = useState("");

  // 排序状态
  const [sortField, setSortField] = useState<string>("");
  const [sortDirection, setSortDirection] = useState<"asc" | "desc">("desc");

  // 筛选状态
  const [filterRetentionPeriod, setFilterRetentionPeriod] = useState<string>("");
  const [filterStatus, setFilterStatus] = useState<string>("");

  // 字段显隐状态
  const [isColumnsDialogOpen, setIsColumnsDialogOpen] = useState(false);
  const [visibleColumns, setVisibleColumns] = useState<string[]>([]);

  // 多选相关
  const [selectedIds, setSelectedIds] = useState<number[]>([]);
  const [selectAll, setSelectAll] = useState(false);

  // 组件卸载标记，防止异步回调更新已卸载组件的 state
  const mountedRef = useRef(true);
  useEffect(() => {
    return () => { mountedRef.current = false; };
  }, []);

  // 文件夹树 & 标签筛选
  const [selectedFolderPath, setSelectedFolderPath] = useState<string | null>(null);
  const [selectedTagNames, setSelectedTagNames] = useState<string[]>([]);

  // 批量标签操作
  const [isBulkTagOpen, setIsBulkTagOpen] = useState(false);
  const [bulkTagInput, setBulkTagInput] = useState("");
  const [bulkTagProcessing, setBulkTagProcessing] = useState(false);

  // 弹窗状态
  const [isFormOpen, setIsFormOpen] = useState(false);
  const [isPreviewOpen, setIsPreviewOpen] = useState(false);
  const [isUploadOpen, setIsUploadOpen] = useState(false);
  const [isShareOpen, setIsShareOpen] = useState(false);
  const [editingDoc, setEditingDoc] = useState<Document | null>(null);
  const [previewDoc, setPreviewDoc] = useState<Document | null>(null);
  const [previewUrl, setPreviewUrl] = useState<string | null>(null);
  const [loadingPreview, setLoadingPreview] = useState(false);
  const previewIframeRef = useRef<HTMLIFrameElement>(null);

  // 上传相关
  const [pendingFiles, setPendingFiles] = useState<PendingFile[]>([]);
  const [isUploading, setIsUploading] = useState(false);
  const [uploadCategory, setUploadCategory] = useState("WS");
  const [uploadSubCategory, setUploadSubCategory] = useState("01");
  const fileInputRef = useRef<HTMLInputElement>(null);
  // 批量上传分页
  const [batchPage, setBatchPage] = useState(1);
  const [batchPageSize] = useState(9);
  const totalBatchPages = Math.ceil(pendingFiles.length / batchPageSize);
  // 批量上传展开编辑的文件ID
  const [expandedFileId, setExpandedFileId] = useState<string | null>(null);
  // 快速填充模板
  const [quickFillTemplate, setQuickFillTemplate] = useState<{
    retention_period: string;
    summary: string;
    remarks: string;
  } | null>(null);
  const singleFileInputRef = useRef<HTMLInputElement>(null);
  // 上传模式：single=单个上传，batch=批量上传
  const [uploadMode, setUploadMode] = useState<'single' | 'batch'>('batch');
  // 单个上传相关
  const [singleFile, setSingleFile] = useState<File | null>(null);
  const [singleFileName, setSingleFileName] = useState("");
  const [singleFileCategory, setSingleFileCategory] = useState("WS");
  const [singleFileSubCategory, setSingleFileSubCategory] = useState("01");
  const [singleUploadProgress, setSingleUploadProgress] = useState(0);
  const [singleUploadStatus, setSingleUploadStatus] = useState<'idle' | 'uploading' | 'success' | 'error' | 'editing'>('idle');
  const [singleUploadError, setSingleUploadError] = useState("");
  // 单文件上传后的补充字段
  const [singleRetentionPeriod, setSingleRetentionPeriod] = useState("永久");
  const [singleSummary, setSingleSummary] = useState("");
  const [singleTags, setSingleTags] = useState<string[]>([]);
  const [singleStorageLocation, setSingleStorageLocation] = useState("");
  const [singleRemarks, setSingleRemarks] = useState("");
  const [uploadedDocId, setUploadedDocId] = useState<number | null>(null);
  
  // OCR 相关状态
  const [ocrLoading, setOcrLoading] = useState(false);
  const [ocrResult, setOcrResult] = useState<OCRExtractResult | null>(null);

  // 文件重复检测
  const [duplicateDialogOpen, setDuplicateDialogOpen] = useState(false);
  const [duplicateFileInfo, setDuplicateFileInfo] = useState<{
    existingDoc: Document;
    newFile: File;
    newFileName: string;
  } | null>(null);

  // 分享相关
  const [shareLinks, setShareLinks] = useState<{ id: number; file_name: string; link: string; copied: boolean }[]>([]);
  const [shareExpiry, setShareExpiry] = useState("24");
  const [isGeneratingLink, setIsGeneratingLink] = useState(false);

  // 打印相关
  const [showPrintDialog, setShowPrintDialog] = useState(false);
  const [printDoc, setPrintDoc] = useState<Document | null>(null);

  // 动态加载的分类和字段
  const [categoryTabs, setCategoryTabs] = useState<{ code: string; name: string }[]>([]);
  const [printTitle, setPrintTitle] = useState(() => {
    if (typeof window !== "undefined") {
      const saved = localStorage.getItem("archive-print-settings");
      if (saved) {
        try {
          return JSON.parse(saved).title ?? "";
        } catch {
          return "";
        }
      }
    }
    return "";
  });
  const [printWatermark, setPrintWatermark] = useState(() => {
    if (typeof window !== "undefined") {
      const saved = localStorage.getItem("archive-print-settings");
      if (saved) {
        try {
          return JSON.parse(saved).watermark ?? "内部资料 请勿外传";
        } catch {
          return "内部资料 请勿外传";
        }
      }
    }
    return "内部资料 请勿外传";
  });
  const [printOrientation, setPrintOrientation] = useState<"auto" | "portrait" | "landscape">(() => {
    if (typeof window !== "undefined") {
      const saved = localStorage.getItem("archive-print-settings");
      if (saved) {
        const parsed = JSON.parse(saved);
        if (parsed.orientation === "portrait" || parsed.orientation === "landscape") {
          return parsed.orientation;
        }
      }
    }
    return "auto";
  });
  const [textPreviewContent, setTextPreviewContent] = useState<string>("");

  const [formData, setFormData] = useState<Record<string, string>>({});
  const [subCategories, setSubCategories] = useState<DocumentSubCategory[]>([]);
  const [selectedSubCategory, setSelectedSubCategory] = useState("");

  const [formSchema, setFormSchema] = useState<FormFieldSchema[]>([]);
  const [tableSchema, setTableSchema] = useState<TableColumnSchema[]>([]);

  // 加载分类数据
  useEffect(() => {
    loadCategories();
    loadStorageLocations();
  }, []);

  // 动态生成表单和表格架构
  useEffect(() => {
    if (selectedSubCategory) {
      fetchFieldsBySubCategory(Number(selectedSubCategory))
        .then((res) => {
          const allFields = [...(res.groups || []).flatMap(g => g.fields || []), ...(res.ungrouped || [])];
          setFormSchema(generateFormSchema(allFields));
          setTableSchema(generateTableSchema(allFields));
        })
        .catch((err) => {
          console.error("加载字段架构失败:", err);
          setFormSchema([]);
          setTableSchema([]);
        });
    }
  }, [selectedSubCategory]);

  // 加载文档列表
  useEffect(() => {
    loadDocuments();
  }, [page, activeTab, filterRetentionPeriod, filterStatus, sortField, sortDirection, selectedFolderPath, selectedTagNames]);

  // 全选逻辑
  useEffect(() => {
    if (selectAll) {
      setSelectedIds(filteredDocuments.map(d => d.id));
    } else if (selectedIds.length === filteredDocuments.length && filteredDocuments.length > 0) {
      // 已经全部选中了，不需要改
    } else if (!selectAll && selectedIds.length === filteredDocuments.length) {
      // 保持当前选择
    }
  }, [selectAll, filteredDocuments]);

  // 保存打印设置到 localStorage
  useEffect(() => {
    const settings = {
      title: printTitle,
      watermark: printWatermark,
      orientation: printOrientation,
    };
    localStorage.setItem("archive-print-settings", JSON.stringify(settings));
  }, [printTitle, printWatermark, printOrientation]);

  const loadCategories = async () => {
    try {
      const data = await fetchDocumentCategories();
      setCategories(data);
      setCategoryTabs(data.map(c => ({ code: c.code, name: c.name })));
    } catch (error) {
      console.error("加载分类失败:", error);
    }
  };

  const loadStorageLocations = async () => {
    try {
      const data = await fetchStorageLocations();
      setStorageLocations(data);
    } catch (error) {
      console.error("加载存储地点失败:", error);
    }
  };

  const loadDocuments = async () => {
    if (!mountedRef.current) return;
    setLoading(true);
    try {
      const params: Record<string, string | number | string[]> = { page, page_size: pageSize };
      if (activeTab !== "all") params.category_code = activeTab;
      if (searchKeyword) params.keyword = searchKeyword;
      if (filterRetentionPeriod) params.retention_period = filterRetentionPeriod;
      if (filterStatus) params.status = filterStatus;
      if (sortField) {
        params.sort_field = sortField;
        params.sort_direction = sortDirection;
      }
      if (selectedFolderPath !== null) params.folder_path = selectedFolderPath;
      if (selectedTagNames.length > 0) params.tag_names = selectedTagNames;

      const response = await fetchDocuments(params as Parameters<typeof fetchDocuments>[0]);
      setFilteredDocuments(response.items);
      setTotal(response.total);
    } catch (error) {
      console.error("加载文档失败:", error);
      toast.error("加载文档失败");
    } finally {
      setLoading(false);
    }
  };

  const handleTabChange = useCallback((value: string) => {
    setActiveTab(value);
    setPage(1);
    setSelectedIds([]);
    setSelectAll(false);
  }, []);

  const handleSearch = () => {
    setPage(1);
    loadDocuments();
  };

  // 选中/取消选中单个
  const toggleSelect = (id: number) => {
    const next = selectedIds.includes(id)
      ? selectedIds.filter(i => i !== id)
      : [...selectedIds, id];
    setSelectedIds(next);
    setSelectAll(next.length === filteredDocuments.length);
  };

  // 全选/取消全选
  const toggleSelectAll = () => {
    if (selectAll) {
      setSelectedIds([]);
      setSelectAll(false);
    } else {
      setSelectedIds(filteredDocuments.map(d => d.id));
      setSelectAll(true);
    }
  };

  const handleEdit = (doc?: Document) => {
    if (!doc) {
      handleCategoryChange(activeTab === "all" ? "WS" : activeTab);
    }

    if (doc) {
      setEditingDoc(doc);
      // 解析标签
      let parsedTags: string[] = [];
      if (doc.tags) {
        try {
          parsedTags = JSON.parse(doc.tags as unknown as string);
        } catch {
          parsedTags = [];
        }
      } else if (doc.summary) {
        // 尝试从 summary 解析
        try {
          parsedTags = JSON.parse(doc.summary as unknown as string);
        } catch {
          parsedTags = [];
        }
      }
      
      const newFormData: Record<string, string> = {
        file_name: doc.file_name || "",
        tags: parsedTags as unknown as string,
        remarks: doc.remarks || "",
      };
      
      formSchema.forEach(field => {
        const value = (doc as unknown as Record<string, unknown>)[field.name];
        if (value !== undefined && value !== null) {
          newFormData[field.name] = String(value);
        }
      });

      if (doc.summary) {
        try {
          const parsed = JSON.parse(doc.summary as unknown as string);
          if (Array.isArray(parsed)) {
            newFormData.summary = parsed as unknown as string;
          }
        } catch (e) {
          console.debug("Non-JSON summary", e);
        }
      }
      
      setFormData(newFormData);
      setSelectedSubCategory(doc.sub_category_code);
      const cat = categories.find(c => c.code === doc.category_code);
      if (cat?.sub_categories) {
        setSubCategories(cat.sub_categories);
      }
    } else {
      setEditingDoc(null);
      setFormData({});
      setSelectedSubCategory("");
    }
    setIsFormOpen(true);
  };

  const handleCategoryChange = (categoryCode: string) => {
    const cat = categories.find(c => c.code === categoryCode);
    if (cat?.sub_categories) {
      setSubCategories(cat.sub_categories);
      if (cat.sub_categories.length > 0) {
        setSelectedSubCategory(cat.sub_categories[0].code);
      }
    }
  };

  const handleSave = async () => {
    try {
      // 处理标签：如果 tags 是数组，转换为 JSON 字符串
      let tagsValue = formData.tags;
      if (Array.isArray(tagsValue)) {
        tagsValue = JSON.stringify(tagsValue) as unknown as string;
      }

      const data = {
        category_code: activeTab === "all" ? "WS" : activeTab,
        sub_category_code: selectedSubCategory || "01",
        ...formData,
        tags: tagsValue,
      };

      if (editingDoc) {
        await updateDocument(editingDoc.id, data);
        toast.success("更新成功");
      } else {
        await createDocument(data);
        toast.success("创建成功");
      }

      setIsFormOpen(false);
      loadDocuments();
    } catch (error) {
      console.error("保存失败:", error);
      toast.error("保存失败");
    }
  };

  const handleDelete = async (docId: number) => {
    if (!confirm("确定删除此档案？")) return;

    try {
      await deleteDocument(docId);
      toast.success("删除成功");
      setSelectedIds(prev => prev.filter(id => id !== docId));
      loadDocuments();
    } catch (error) {
      console.error("删除失败:", error);
      toast.error("删除失败");
    }
  };

  const handlePreview = async (doc: Document) => {
    setPreviewDoc(doc);
    setPreviewUrl(null);
    setTextPreviewContent("");
    setLoadingPreview(true);
    setIsPreviewOpen(true);

    // 如果有文件路径，尝试加载预览
    if (doc.file_path) {
      try {
        const token = localStorage.getItem("token");
        const res = await fetch(`${process.env.NEXT_PUBLIC_API_BASE_URL || 'http://localhost:8080/api'}/archives/filteredDocuments/${doc.id}/download`, {
          headers: token ? { Authorization: `Bearer ${token}` } : {},
        });
        if (res.ok) {
          const blob = await res.blob();
          // 判断是否为文本文件
          const isTextFile = doc.file_type === "text" ||
            doc.file_name?.match(/\.(txt|text|log|md|json|xml|yaml|yml|csv)$/i);
          if (isTextFile) {
            const text = await blob.text();
            setTextPreviewContent(text);
          } else {
            const url = URL.createObjectURL(blob);
            setPreviewUrl(url);
          }
        }
      } catch (err) {
        console.error("加载预览失败:", err);
      }
    }
    setLoadingPreview(false);
  };

  // 关闭预览弹窗时清理URL
  const handleClosePreview = () => {
    if (previewUrl) {
      URL.revokeObjectURL(previewUrl);
    }
    setPreviewUrl(null);
    setTextPreviewContent("");
    setIsPreviewOpen(false);
  };

  // 打开打印对话框
  const openPrintDialog = (doc: Document) => {
    setPrintDoc(doc);
    setShowPrintDialog(true);
  };

  // 关闭打印对话框
  const handleClosePrintDialog = () => {
    setShowPrintDialog(false);
    setPrintDoc(null);
  };

  // 生成打印预览
  const handleGeneratePrint = async () => {
    if (!printDoc) {
      toast.error("未选择要打印的文档");
      return;
    }

    if (typeof window === "undefined" || typeof document === "undefined") {
      toast.error("当前环境不支持打印预览");
      return;
    }

    const title = (printTitle.trim() || `档案文档打印 - ${printDoc.file_name}`).trim();
    const watermark = (printWatermark.trim() || "内部资料 请勿外传").trim();

    const loadingToastId = toast.loading("正在生成打印预览，请稍候...");
    try {
      // 构建文档元数据列
      const columns = ["档案编号", "档案类型", "文件名", "档案地点", "上传日期", "保存期限", "摘要"];
      const rows = [[
        printDoc.document_code || "-",
        printDoc.sub_type || printDoc.sub_category_code || "-",
        printDoc.file_name || "-",
        printDoc.storage_location || "-",
        printDoc.created_at?.slice(0, 10) || "-",
        printDoc.retention_period || "-",
        printDoc.summary || "-",
      ]];

      const blob = await createReportPdf({
        title,
        watermark,
        columns,
        rows,
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
      console.error("[ArchivesManagement] generate pdf failed", error);
      toast.error("生成打印预览失败，请稍后重试");
    } finally {
      toast.dismiss(loadingToastId);
    }
  };

  // 打开批量上传弹窗
  const handleOpenUpload = () => {
    setPendingFiles([]);
    setUploadCategory(activeTab === "all" ? "WS" : activeTab);
    const cat = categories.find(c => c.code === uploadCategory);
    if (cat?.sub_categories && cat.sub_categories.length > 0) {
      setUploadSubCategory(cat.sub_categories[0].code);
    }
    setIsUploadOpen(true);
  };

  // 处理文件选择
  const handleFileSelect = (e: React.ChangeEvent<HTMLInputElement>) => {
    const files = Array.from(e.target.files || []);
    if (files.length === 0) return;

    const validFiles: PendingFile[] = files.map(file => ({
      id: `${Date.now()}-${Math.random().toString(36).slice(2)}`,
      file,
      status: 'pending' as const,
      progress: 0,
      document: {
        file_name: file.name.replace(/\.[^.]+$/, ''), // 去掉扩展名作为默认名称
        file_size: file.size,
        file_type: file.type,
        file_format: getFileExtension(file.name), // 自动提取扩展名作为文件格式
        category_code: uploadCategory,
        sub_category_code: uploadSubCategory,
        storage_location: getDefaultStorageLocation(), // 使用默认存储地点
      },
    }));

    // 检查是否有重名文件（基于文件名，不包含扩展名）
    const fileNameWithoutExt = (f: File) => f.name.replace(/\.[^.]+$/, '');
    for (const newFile of files) {
      const duplicateDoc = filteredDocuments.find(doc =>
        doc.file_name?.toLowerCase() === fileNameWithoutExt(newFile).toLowerCase()
      );
      if (duplicateDoc) {
        setDuplicateFileInfo({
          existingDoc: duplicateDoc,
          newFile: newFile,
          newFileName: fileNameWithoutExt(newFile),
        });
        setDuplicateDialogOpen(true);
        // 清空 input
        if (fileInputRef.current) {
          fileInputRef.current.value = '';
        }
        return;
      }
    }

    setPendingFiles(prev => [...prev, ...validFiles]);
    setBatchPage(1); // 重置到第一页

    // 清空 input 以便再次选择相同文件
    if (fileInputRef.current) {
      fileInputRef.current.value = '';
    }
  };

  // 移除待上传文件
  const removePendingFile = (id: string) => {
    setPendingFiles(prev => prev.filter(f => f.id !== id));
  };

  // 更新待上传文件的元数据
  const updatePendingFileMeta = (id: string, updates: Partial<Document>) => {
    setPendingFiles(prev => prev.map(f =>
      f.id === id ? { ...f, document: { ...f.document, ...updates } } : f
    ));
  };

  // 批量上传
  const handleBatchUpload = async () => {
    if (pendingFiles.length === 0) {
      toast.error("请先选择要上传的文件");
      return;
    }

    // 验证必填字段
    const invalidFiles = pendingFiles.filter(f => !f.document?.file_name);
    if (invalidFiles.length > 0) {
      toast.error("请填写所有文件的名称");
      return;
    }

    setIsUploading(true);

    const updatedFiles = [...pendingFiles];
    for (let i = 0; i < updatedFiles.length; i++) {
      const item = updatedFiles[i];
      item.status = 'uploading';
      item.progress = 0;
      setPendingFiles([...updatedFiles]);

      try {
        // 先创建文档记录
        const docData = {
          category_code: uploadCategory,
          sub_category_code: uploadSubCategory,
          file_name: item.document?.file_name || item.file.name,
          retention_period: item.document?.retention_period || "永久",
          summary: item.document?.summary || "",
          remarks: item.document?.remarks || "",
          storage_location: item.document?.storage_location || getDefaultStorageLocation(),
          file_format: item.document?.file_format || getFileExtension(item.file.name),
          tags: item.document?.tags,
          ...item.document,
        };

        const created = await createDocument(docData);

        // 上传文件
        await uploadDocumentFile(created.id, item.file);

        item.status = 'success';
        item.progress = 100;
        item.document = created;
      } catch (error) {
        console.error(`上传失败: ${item.file.name}`, error);
        item.status = 'error';
        item.error = error instanceof Error ? error.message : "上传失败";
      }

      setPendingFiles([...updatedFiles]);
    }

    setIsUploading(false);
    setExpandedFileId(null); // 重置展开状态
    toast.success(`批量上传完成，成功 ${updatedFiles.filter(f => f.status === 'success').length} 个`);

    // 延迟关闭弹窗并刷新列表
    setTimeout(() => {
      setIsUploadOpen(false);
      setPendingFiles([]);
      setBatchPage(1);
      loadDocuments();
    }, 1500);
  };

  // 单个文件上传
  const handleSingleFileSelect = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file) return;
    setSingleFile(file);
    setSingleFileName(file.name.replace(/\.[^.]+$/, ''));
    setSingleUploadStatus('idle');
    setSingleUploadError('');
    setOcrResult(null);
    setOcrLoading(false);
    
    // 触发 OCR 识别
    triggerOCRExtraction(file);
    
    if (singleFileInputRef.current) {
      singleFileInputRef.current.value = '';
    }
  };

  // 触发 OCR 识别
  const triggerOCRExtraction = async (file: File) => {
    setOcrLoading(true);
    try {
      const result = await uploadWithOCR(file, singleFileSubCategory);
      setOcrResult(result);
      
      // 自动填充共用字段
      if (result.shared_fields) {
        if (result.shared_fields.archive_title) {
          setSingleFileName(result.shared_fields.archive_title as string);
        }
        if (result.shared_fields.summary) {
          setSingleSummary(result.shared_fields.summary as string);
        }
      }
      
      if (result.ocr_status === 'failed') {
        toast.warning('OCR识别失败，请手动填写相关字段');
      } else {
        toast.success('OCR识别完成，已自动填充字段');
      }
    } catch (err) {
      console.error('OCR识别异常:', err);
      toast.error('OCR服务异常，请手动填写');
    } finally {
      setOcrLoading(false);
    }
  };

  // 提取文件扩展名
  const getFileExtension = (filename: string): string => {
    const parts = filename.split('.');
    return parts.length > 1 ? parts[parts.length - 1].toLowerCase() : '';
  };

  // 获取默认存储地点
  const getDefaultStorageLocation = (): string => {
    const defaultLoc = storageLocations.find(loc => loc.name);
    return defaultLoc?.name || '';
  };

  // 单个文件上传提交
  const handleSingleUpload = async () => {
    if (!singleFile) {
      toast.error("请先选择文件");
      return;
    }
    if (!singleFileName.trim()) {
      toast.error("请输入文件名称");
      return;
    }

    setSingleUploadStatus('uploading');
    setSingleUploadProgress(0);
    setSingleUploadError('');

    try {
      // 提取文件扩展名作为 file_format
      const fileFormat = getFileExtension(singleFile.name);

      // 先创建文档记录
      const docData = {
        category_code: singleFileCategory,
        sub_category_code: singleFileSubCategory,
        file_name: singleFileName,
        file_format: fileFormat,
        retention_period: singleRetentionPeriod,
        storage_location: singleStorageLocation || getDefaultStorageLocation(),
        tags: singleTags.length > 0 ? JSON.stringify(singleTags) : undefined,
        remarks: singleRemarks,
      };

      const created = await createDocument(docData);
      setUploadedDocId(created.id);
      setSingleUploadProgress(50);

      // 上传文件
      await uploadDocumentFile(created.id, singleFile);
      setSingleUploadProgress(100);

      // 上传成功后进入编辑模式
      setSingleUploadStatus('editing');
      toast.success("上传成功，请完善档案信息");
    } catch (error) {
      console.error("上传失败:", error);
      setSingleUploadStatus('error');
      setSingleUploadError(error instanceof Error ? error.message : "上传失败");
    }
  };

  // 保存单文件档案补充信息
  const handleSaveSingleFileMeta = async () => {
    if (!uploadedDocId) {
      toast.error("文档信息不存在");
      return;
    }

    try {
      await updateDocument(uploadedDocId, {
        retention_period: singleRetentionPeriod,
        summary: singleSummary,
        tags: singleTags.length > 0 ? JSON.stringify(singleTags) : undefined,
        storage_location: singleStorageLocation || getDefaultStorageLocation(),
        remarks: singleRemarks,
      });
      toast.success("档案信息保存成功");
      // 关闭弹窗并重置状态
      setIsUploadOpen(false);
      setSingleFile(null);
      setSingleFileName("");
      setSingleRetentionPeriod("永久");
      setSingleSummary("");
      setSingleTags([]);
      setSingleStorageLocation("");
      setSingleRemarks("");
      setUploadedDocId(null);
      setSingleUploadStatus('idle');
      loadDocuments();
    } catch (error) {
      console.error("保存失败:", error);
      toast.error("保存失败，请稍后重试");
    }
  };

  // 重置单文件上传状态
  const resetSingleUpload = () => {
    setSingleFile(null);
    setSingleFileName("");
    setSingleUploadStatus('idle');
    setSingleUploadError('');
    setSingleRetentionPeriod("永久");
    setSingleSummary("");
    setSingleTags([]);
    setSingleStorageLocation("");
    setSingleRemarks("");
    setUploadedDocId(null);
  };

  // 批量下载
  const handleBatchDownload = async () => {
    if (selectedIds.length === 0) {
      toast.error("请先选择要下载的文件");
      return;
    }

    try {
      toast.loading("正在打包下载...");
      const blob = await batchDownloadDocuments(selectedIds);

      // 创建下载链接
      const url = URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = `档案_${new Date().toISOString().slice(0, 10)}.zip`;
      document.body.appendChild(a);
      a.click();
      document.body.removeChild(a);
      URL.revokeObjectURL(url);

      toast.success("下载完成");
    } catch (error) {
      console.error("批量下载失败:", error);
      toast.error("批量下载失败");
    }
  };

  // 批量标签操作
  const handleBulkTagApply = async () => {
    const tagName = bulkTagInput.trim();
    if (!tagName) {
      toast.error("请输入标签名称");
      return;
    }
    if (selectedIds.length === 0) {
      toast.error("请先选择文档");
      return;
    }

    setBulkTagProcessing(true);
    const ids = selectedIds;
    let successCount = 0;
    let failCount = 0;

    try {
      for (const docId of ids) {
        try {
          await setDocumentTags(docId, [tagName]);
          successCount++;
        } catch {
          failCount++;
        }
      }

      if (failCount === 0) {
        toast.success(`已为 ${successCount} 个文档设置标签「${tagName}」`);
      } else {
        toast.warning(`完成：${successCount} 个成功，${failCount} 个失败`);
      }
      setIsBulkTagOpen(false);
      setBulkTagInput("");
      loadDocuments();
    } catch (error) {
      console.error("批量标签操作失败:", error);
      toast.error("批量操作失败");
    } finally {
      setBulkTagProcessing(false);
    }
  };

  // 文件夹选择
  const handleFolderSelect = useCallback((path: string | null) => {
    setSelectedFolderPath(path);
    setPage(1);
    setSelectedIds([]);
    setSelectAll(false);
  }, []);

  // 标签筛选
  const handleTagFilter = useCallback((tagNames: string[]) => {
    setSelectedTagNames(tagNames);
    setPage(1);
    setSelectedIds([]);
    setSelectAll(false);
  }, []);

  // 生成分享链接
  const handleShare = async () => {
    if (selectedIds.length === 0) {
      toast.error("请先选择要分享的文件");
      return;
    }

    setIsGeneratingLink(true);
    try {
      const links = await generateShareLink(selectedIds, parseInt(shareExpiry));
      setShareLinks(links.map(l => ({ ...l, copied: false })));
      setIsShareOpen(true);
    } catch (error) {
      console.error("生成分享链接失败:", error);
      toast.error("生成分享链接失败");
    } finally {
      setIsGeneratingLink(false);
    }
  };

  // 复制分享链接
  const copyLink = (id: number, link: string) => {
    navigator.clipboard.writeText(link);
    setShareLinks(prev => prev.map(l => l.id === id ? { ...l, copied: true } : l));
    toast.success("链接已复制到剪贴板");

    setTimeout(() => {
      setShareLinks(prev => prev.map(l => l.id === id ? { ...l, copied: false } : l));
    }, 2000);
  };

  // 获取表格列配置
  const columns = tableSchema;

  const defaultVisibleColumns = useMemo(() => {
    const defaultFields = ["document_code", "sub_type", "file_name", "file_format", "storage_location", "created_at", "retention_period", "summary", "status"];
    const defaultSet = new Set(defaultFields);
    const filtered = columns.filter((column) => defaultSet.has(column.key)).map((column) => column.key);
    return filtered.length > 0 ? filtered : columns.map((column) => column.key);
  }, [columns]);

  // 初始化可见列（仅在首次挂载时）- 使用 ref 确保只执行一次
  const initRef = useRef(false);
  useEffect(() => {
    if (!initRef.current && columns.length > 0) {
      initRef.current = true;
      // 从后端加载列配置
      const loadColumnConfig = async () => {
        try {
          const config = await fetchColumnConfig(selectedSubCategory || "01");
          if (config.column_keys && config.column_keys.length > 0) {
            setVisibleColumns(config.column_keys);
          } else {
            setVisibleColumns(defaultVisibleColumns);
          }
        } catch (error) {
          console.error("加载列配置失败:", error);
          setVisibleColumns(defaultVisibleColumns);
        }
      };
      loadColumnConfig();
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [columns, defaultVisibleColumns]);

  // 当 tab 切换时，重置可见列
  useEffect(() => {
    setVisibleColumns(defaultVisibleColumns);
    setSortField("");
    setSortDirection("desc");
    setFilterRetentionPeriod("");
    setFilterStatus("");
  }, [activeTab, defaultVisibleColumns]);

  // 获取文件类型图标
  const getFileIcon = (fileType?: string) => {
    if (!fileType) return <File className="h-8 w-8 text-muted-foreground" />;
    if (fileType.startsWith('image/')) return <Image className="h-8 w-8 text-blue-500" />;
    if (fileType === 'application/pdf') return <FileText className="h-8 w-8 text-red-500" />;
    if (fileType.startsWith('video/')) return <Video className="h-8 w-8 text-purple-500" />;
    if (fileType === 'text') return <FileText className="h-8 w-8 text-green-500" />;
    return <File className="h-8 w-8 text-muted-foreground" />;
  };

  // 格式化文件大小
  const formatFileSize = (bytes: number) => {
    if (bytes < 1024) return `${bytes} B`;
    if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
    return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
  };

  return (
    <div className="flex h-[calc(100vh-12rem)] gap-0">
      {/* 左侧栏：文件夹树 */}
      <aside className="w-60 shrink-0 overflow-hidden rounded-l-xl border border-r-0 bg-card">
        <FolderTree
          categoryCode={activeTab === "all" ? (categoryTabs[0]?.code || "WS") : activeTab}
          onSelect={handleFolderSelect}
        />
      </aside>

      {/* 右侧主内容区 */}
      <Card className="flex flex-1 flex-col overflow-hidden rounded-l-none border-l-0">
      <CardHeader>
        <div className="flex flex-wrap items-center justify-between gap-4">
          <div>
            <CardTitle>档案管理</CardTitle>
          </div>
          <div className="flex flex-wrap items-center gap-3 text-sm">
            <span className="text-sm text-muted-foreground">
              {loading ? "加载中..." : `共 ${total} 条记录`}
            </span>
            {selectedIds.length > 0 && <Badge variant="default">已选择 {selectedIds.length}</Badge>}
            <div className="flex items-center gap-2">
              <Button variant="outline" onClick={handleOpenUpload}>
                <Upload className="h-4 w-4 mr-2" />
                上传
              </Button>
              {selectedIds.length > 0 && (
                <>
                  <Button variant="outline" onClick={handleBatchDownload}>
                    <Download className="h-4 w-4 mr-2" />
                    下载 ({selectedIds.length})
                  </Button>
                  <Button variant="outline" onClick={handleShare}>
                    <Share2 className="h-4 w-4 mr-2" />
                    分享 ({selectedIds.length})
                  </Button>
                </>
              )}
            </div>
          </div>
        </div>
      </CardHeader>
      <CardContent>
        {/* 标签筛选栏 */}
        <div className="mb-4 rounded-lg border bg-card/50 p-3">
          <TagFilter selectedTags={selectedTagNames} onFilter={handleTagFilter} />
        </div>

        {/* 批量操作栏 */}
        {selectedIds.length > 0 && (
          <div className="mb-4 flex flex-wrap items-center gap-2 rounded-lg border border-primary/20 bg-primary/5 px-4 py-2.5">
            <span className="text-sm font-medium text-primary">
              已选择 {selectedIds.length} 个文档
            </span>
            <span className="text-xs text-muted-foreground">—</span>
            <Dialog open={isBulkTagOpen} onOpenChange={setIsBulkTagOpen}>
              <DialogTrigger asChild>
                <Button variant="outline" size="sm" className="h-8">
                  <Tag className="h-3.5 w-3.5 mr-1.5" />
                  批量标签
                </Button>
              </DialogTrigger>
              <DialogContent className="max-w-sm">
                <DialogHeader>
                  <DialogTitle>批量设置标签</DialogTitle>
                  <DialogDescription>
                    为选中的 {selectedIds.length} 个文档统一设置标签。已输入标签名即可自动创建或关联。
                  </DialogDescription>
                </DialogHeader>
                <div className="space-y-3 py-2">
                  <div>
                    <Label htmlFor="bulk-tag-input">标签名称</Label>
                    <Input
                      id="bulk-tag-input"
                      value={bulkTagInput}
                      onChange={(e) => setBulkTagInput(e.target.value)}
                      placeholder="输入标签名（如：合同、重要）"
                      onKeyDown={(e) => e.key === "Enter" && handleBulkTagApply()}
                    />
                  </div>
                </div>
                <DialogFooter>
                  <Button variant="outline" onClick={() => setIsBulkTagOpen(false)}>
                    取消
                  </Button>
                  <Button
                    onClick={handleBulkTagApply}
                    disabled={bulkTagProcessing || !bulkTagInput.trim()}
                  >
                    {bulkTagProcessing ? (
                      <>
                        <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                        处理中...
                      </>
                    ) : (
                      <>
                        <Plus className="h-4 w-4 mr-2" />
                        应用标签
                      </>
                    )}
                  </Button>
                </DialogFooter>
              </DialogContent>
            </Dialog>
          </div>
        )}

        {/* 搜索和筛选 */}
        <div className="flex flex-wrap items-center justify-between gap-4 mb-4">
          <div className="flex flex-1 flex-wrap items-center gap-4">
            <div className="relative flex-1 min-w-[220px]">
              <Search className="absolute left-3 top-3 h-4 w-4 text-muted-foreground" />
              <Input
                placeholder="搜索档案编号、文件名..."
                value={searchKeyword}
                onChange={(e) => setSearchKeyword(e.target.value)}
                onKeyDown={(e) => e.key === "Enter" && handleSearch()}
                className="pl-10 pr-10"
              />
              {searchKeyword && (
                <button
                  type="button"
                  onClick={() => setSearchKeyword("")}
                  className="absolute right-2 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground"
                  aria-label="清除搜索"
                >
                  <X className="h-4 w-4" />
                </button>
              )}
            </div>
            {/* 分类筛选 */}
            <Select
              value={activeTab === "all" ? "__all__" : activeTab}
              onValueChange={(v) => handleTabChange(v === "__all__" ? "all" : v)}
            >
              <SelectTrigger className="w-36">
                <SelectValue placeholder="全部分类" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="__all__">全部分类</SelectItem>
                {categoryTabs.map((cat) => (
                  <SelectItem key={cat.code} value={cat.code}>
                    {cat.name}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
            {/* 状态筛选 */}
            <Select value={filterStatus || "__all__"} onValueChange={(v) => setFilterStatus(v === "__all__" ? "" : v)}>
              <SelectTrigger className="w-28">
                <SelectValue placeholder="状态" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="__all__">全部状态</SelectItem>
                <SelectItem value="active">正常</SelectItem>
                <SelectItem value="archived">已归档</SelectItem>
                <SelectItem value="expired">已到期</SelectItem>
              </SelectContent>
            </Select>
          </div>
          {/* 字段显隐控制 */}
          <Dialog open={isColumnsDialogOpen} onOpenChange={setIsColumnsDialogOpen}>
            <DialogTrigger asChild>
              <Button variant="outline" size="sm">
                <Settings className="h-4 w-4 mr-2" />
                显示字段
              </Button>
            </DialogTrigger>
            <DialogContent className="max-w-md">
              <DialogHeader>
                <DialogTitle>自定义显示字段</DialogTitle>
                <DialogDescription>勾选后即可在表格中显示对应列</DialogDescription>
              </DialogHeader>
              <div className="max-h-[60vh] space-y-3 overflow-y-auto py-2">
                {columns.map((col) => (
                  <div key={col.key} className="flex items-center space-x-2">
                    <input
                      type="checkbox"
                      id={`archive-field-${col.key}`}
                      className="h-4 w-4 rounded border-gray-300"
                      checked={visibleColumns.includes(col.key)}
                      onChange={(event) => {
                        if (event.target.checked) {
                          if (visibleColumns.includes(col.key)) {
                            return;
                          }
                          setVisibleColumns([...visibleColumns, col.key]);
                          return;
                        }
                        const next = visibleColumns.filter((key) => key !== col.key);
                        setVisibleColumns(next.length > 0 ? next : defaultVisibleColumns);
                      }}
                    />
                    <label
                      htmlFor={`archive-field-${col.key}`}
                      className="text-sm font-medium leading-none peer-disabled:cursor-not-allowed peer-disabled:opacity-70"
                    >
                      {col.label}
                    </label>
                  </div>
                ))}
              </div>
               <DialogFooter className="flex flex-col gap-2 sm:flex-row sm:justify-end">
                 <Button variant="outline" onClick={() => setVisibleColumns(defaultVisibleColumns)}>
                   恢复默认
                 </Button>
                 <Button variant="default" onClick={async () => {
                   try {
                     await saveColumnConfig(selectedSubCategory || "01", visibleColumns);
                     toast.success("列配置已保存为默认");
                   } catch (error) {
                     console.error("保存列配置失败:", error);
                     toast.error("保存列配置失败");
                   }
                 }}>
                   保存为默认
                 </Button>
                 <Button onClick={() => setIsColumnsDialogOpen(false)}>完成</Button>
               </DialogFooter>
            </DialogContent>
          </Dialog>
        </div>

      {/* 表格 */}
      <ArchiveTableRenderer
        schema={tableSchema}
        data={filteredDocuments as unknown as Array<Record<string, unknown> & { id: number }>}
        visibleColumns={visibleColumns}
        onEdit={handleEdit as unknown as (row: Record<string, unknown> & { id: number }) => void}
        onDelete={handleDelete}
        onView={handlePreview as unknown as (row: Record<string, unknown> & { id: number }) => void}
        selectedIds={selectedIds}
        onSelect={toggleSelect}
        onSelectAll={toggleSelectAll}
        selectAll={selectAll}
      />

      {/* 分页 */}
      <div className="flex items-center justify-between">
        <div className="text-sm text-muted-foreground">
          共 {total} 条记录，第 {page} 页，已选择 {selectedIds.length} 项
        </div>
        <div className="flex items-center gap-2">
          <Button
            variant="outline"
            size="sm"
            disabled={page <= 1}
            onClick={() => setPage(page - 1)}
          >
            上一页
          </Button>
          <Button
            variant="outline"
            size="sm"
            disabled={page * pageSize >= total}
            onClick={() => setPage(page + 1)}
          >
            下一页
          </Button>
        </div>
      </div>

      {/* 表单弹窗 - 统一所有字段 */}
      <Dialog open={isFormOpen} onOpenChange={setIsFormOpen}>
        <DialogContent className="max-w-3xl max-h-[90vh] overflow-y-auto">
          <DialogHeader>
            <DialogTitle>{editingDoc ? "编辑档案" : "新增档案"}</DialogTitle>
          </DialogHeader>
          <div className="py-4">
            <ArchiveFormRenderer
              schema={formSchema}
              data={formData as unknown as Record<string, unknown>}
              onChange={(data) => setFormData(data as unknown as Record<string, string>)}
              storageLocations={storageLocations as unknown as Record<string, unknown>[]}
              retentionPeriods={RETENTION_OPTIONS.map(opt => ({ name: opt.value }))}
            />
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setIsFormOpen(false)}>取消</Button>
            <Button onClick={handleSave}>保存</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* 预览弹窗 */}
      <Dialog open={isPreviewOpen} onOpenChange={(open) => {
        if (!open) handleClosePreview();
      }}>
        <DialogContent className="max-w-5xl max-h-[90vh] flex flex-col">
          <DialogHeader>
            <DialogTitle>文档预览 - {previewDoc?.file_name}</DialogTitle>
          </DialogHeader>
          <div className="flex-1 overflow-hidden border-2 border-dashed rounded-lg">
            {loadingPreview ? (
              <div className="flex h-full flex-col items-center justify-center gap-4">
                <Loader2 className="h-12 w-12 animate-spin text-muted-foreground" />
                <p className="text-muted-foreground">加载中...</p>
              </div>
            ) : previewUrl ? (
              <>
                {/* 图片预览 */}
                {previewDoc?.file_type === "image" && (
                  <div className="flex h-full items-center justify-center overflow-auto p-4 bg-muted/20">
                    <img
                      src={previewUrl}
                      alt={previewDoc?.file_name}
                      className="max-w-full max-h-full object-contain"
                    />
                  </div>
                )}
                {/* PDF预览 */}
                {previewDoc?.file_type === "pdf" && (
                  <iframe
                    ref={previewIframeRef}
                    src={previewUrl}
                    className="w-full h-full"
                    title="PDF预览"
                    sandbox="allow-scripts allow-same-origin"
                  />
                )}
                {/* 视频预览 */}
                {previewDoc?.file_type === "video" && (
                  <div className="flex h-full items-center justify-center bg-black p-4">
                    <video
                      src={previewUrl}
                      controls
                      className="max-w-full max-h-full"
                    >
                      您的浏览器不支持视频播放
                    </video>
                  </div>
                )}
                {/* 文本预览 */}
                {previewDoc?.file_type === "text" && (
                  <ScrollArea className="h-full">
                    <pre className="p-4 text-sm font-mono whitespace-pre-wrap break-words">
                      {textPreviewContent || "加载中..."}
                    </pre>
                  </ScrollArea>
                )}
                {/* 不支持的类型 */}
                {!["image", "pdf", "video", "text"].includes(previewDoc?.file_type || "") && (
                  <div className="flex h-full flex-col items-center justify-center gap-4">
                    {getFileIcon(previewDoc?.file_type)}
                    <p className="text-muted-foreground">该文件类型暂不支持在线预览</p>
                    <p className="text-sm text-muted-foreground">
                      您可以下载后查看，或使用打印功能
                    </p>
                  </div>
                )}
              </>
            ) : (
              <div className="flex h-full flex-col items-center justify-center gap-4">
                <File className="h-32 w-32 text-muted-foreground" />
                <p className="text-muted-foreground">暂无文件预览</p>
              </div>
            )}
          </div>
          <DialogFooter className="flex-row justify-end gap-2">
            <Button variant="outline" onClick={handleClosePreview}>关闭</Button>
            {previewDoc && (
              <Button onClick={() => openPrintDialog(previewDoc)}>
                <Printer className="mr-2 h-4 w-4" />
                打印设置
              </Button>
            )}
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* 打印设置对话框 */}
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
                placeholder={printDoc?.file_name ? `档案文档打印 - ${printDoc.file_name}` : "例如：档案文档打印"}
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
              <Select value={printOrientation} onValueChange={(value) => setPrintOrientation(value as "auto" | "portrait" | "landscape")}>
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
              {printDoc ? (
                <>
                  <div>文档名称：{printDoc.file_name}</div>
                  <div>档案分类：{printDoc.category_code} - {printDoc.sub_category_code}</div>
                  <div>提示：生成的 PDF 将在新窗口打开，可直接打印或保存。</div>
                </>
              ) : (
                <div>未选择文档，请重新选择。</div>
              )}
            </div>
          </div>
          <DialogFooter className="flex flex-col gap-2 sm:flex-row sm:justify-end">
            <Button variant="outline" onClick={handleClosePrintDialog}>
              取消
            </Button>
            <Button onClick={handleGeneratePrint} disabled={!printDoc}>
              生成预览
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* 上传弹窗 */}
      <Dialog open={isUploadOpen} onOpenChange={setIsUploadOpen}>
        <DialogContent className="max-w-4xl max-h-[90vh] overflow-hidden flex flex-col">
          <DialogHeader>
            <DialogTitle>上传档案文件</DialogTitle>
            <DialogDescription>
              支持上传扫描件、图片、图纸等实体文件。
            </DialogDescription>
          </DialogHeader>

          {/* 文件重复确认对话框 */}
          <AlertDialog open={duplicateDialogOpen} onOpenChange={setDuplicateDialogOpen}>
            <AlertDialogContent className="max-w-lg">
              <AlertDialogHeader>
                <AlertDialogTitle>检测到重名文件</AlertDialogTitle>
                <AlertDialogDescription className="space-y-3">
                  <p>系统中已存在相同文件名的档案，请选择处理方式：</p>
                  {duplicateFileInfo && (
                    <div className="grid grid-cols-2 gap-4 py-3 border rounded-lg bg-muted/30">
                      <div className="space-y-2">
                        <div className="font-medium text-sm text-foreground">现有文件</div>
                        <div className="text-xs space-y-1">
                          <div><span className="text-muted-foreground">文件名：</span>{duplicateFileInfo.existingDoc.file_name}</div>
                          <div><span className="text-muted-foreground">上传时间：</span>{duplicateFileInfo.existingDoc.created_at ? new Date(duplicateFileInfo.existingDoc.created_at).toLocaleString() : '-'}</div>
                          <div><span className="text-muted-foreground">文件类型：</span>{duplicateFileInfo.existingDoc.file_format || '-'}</div>
                        </div>
                      </div>
                      <div className="space-y-2">
                        <div className="font-medium text-sm text-foreground">新上传文件</div>
                        <div className="text-xs space-y-1">
                          <div><span className="text-muted-foreground">文件名：</span>{duplicateFileInfo.newFileName}</div>
                          <div><span className="text-muted-foreground">文件大小：</span>{formatFileSize(duplicateFileInfo.newFile.size)}</div>
                          <div><span className="text-muted-foreground">修改时间：</span>{new Date(duplicateFileInfo.newFile.lastModified).toLocaleString()}</div>
                        </div>
                      </div>
                    </div>
                  )}
                </AlertDialogDescription>
              </AlertDialogHeader>
              <AlertDialogFooter className="flex-col sm:flex-col gap-2">
                <Button
                  variant="destructive"
                  onClick={async () => {
                    // 覆盖模式：更新现有文档的文件
                    if (duplicateFileInfo) {
                      try {
                        await updateDocument(duplicateFileInfo.existingDoc.id, {
                          file_name: duplicateFileInfo.newFileName,
                          remarks: `文件已替换，原始文件上传于 ${new Date(duplicateFileInfo.existingDoc.created_at).toLocaleString()}`
                        });
                        // 上传新文件
                        await uploadDocumentFile(duplicateFileInfo.existingDoc.id, duplicateFileInfo.newFile);
                        toast.success("文件已覆盖");
                        loadDocuments();
                      } catch (error) {
                        toast.error("操作失败");
                      }
                    }
                    setDuplicateDialogOpen(false);
                    setDuplicateFileInfo(null);
                  }}
                >
                  <FileUp className="h-4 w-4 mr-1" />
                  覆盖保存
                </Button>
                <Button
                  variant="default"
                  onClick={() => {
                    // 保留两者模式：自动重命名新文件
                    if (duplicateFileInfo) {
                      const renamedFiles: PendingFile[] = [{
                        id: `${Date.now()}-${Math.random().toString(36).slice(2)}`,
                        file: duplicateFileInfo.newFile,
                        status: 'pending' as const,
                        progress: 0,
                        document: {
                          file_name: `${duplicateFileInfo.newFileName}_${Date.now()}`,
                          file_size: duplicateFileInfo.newFile.size,
                          file_type: duplicateFileInfo.newFile.type,
                          file_format: getFileExtension(duplicateFileInfo.newFile.name),
                          category_code: uploadCategory,
                          sub_category_code: uploadSubCategory,
                          storage_location: getDefaultStorageLocation(),
                        },
                      }];
                      setPendingFiles(prev => [...prev, ...renamedFiles]);
                      setBatchPage(1);
                      toast.success("文件已重命名添加");
                    }
                    setDuplicateDialogOpen(false);
                    setDuplicateFileInfo(null);
                  }}
                >
                  <Copy className="h-4 w-4 mr-1" />
                  保留两者（自动重命名）
                </Button>
                <AlertDialogCancel
                  onClick={() => {
                    setDuplicateDialogOpen(false);
                    setDuplicateFileInfo(null);
                  }}
                >
                  取消上传
                </AlertDialogCancel>
              </AlertDialogFooter>
            </AlertDialogContent>
          </AlertDialog>

          {/* 模式切换 */}
          <Tabs value={uploadMode} onValueChange={(v) => setUploadMode(v as 'single' | 'batch')}>
            <TabsList className="grid w-full grid-cols-2">
              <TabsTrigger value="single">单个上传</TabsTrigger>
              <TabsTrigger value="batch">批量上传</TabsTrigger>
            </TabsList>
          </Tabs>

          <div className="flex-1 overflow-hidden flex flex-col gap-4">
            {/* 单个上传模式 */}
            {uploadMode === 'single' && (
              <div className="space-y-4">
                {/* 分类选择 */}
                <div className="grid grid-cols-2 gap-4">
                  <div>
                    <Label className="mb-2 block">档案分类</Label>
                    <Select value={singleFileCategory} onValueChange={(v) => {
                      setSingleFileCategory(v);
                      const cat = categories.find(c => c.code === v);
                      if (cat?.sub_categories && cat.sub_categories.length > 0) {
                        setSingleFileSubCategory(cat.sub_categories[0].code);
                      }
                    }}>
                      <SelectTrigger>
                        <SelectValue placeholder="选择档案分类" />
                      </SelectTrigger>
                      <SelectContent>
                        {categories.map(cat => (
                          <SelectItem key={cat.code} value={cat.code}>{cat.name}</SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
                  </div>
                  <div>
                    <Label className="mb-2 block">二级分类</Label>
                    <Select value={singleFileSubCategory} onValueChange={setSingleFileSubCategory}>
                      <SelectTrigger>
                        <SelectValue placeholder="选择二级分类" />
                      </SelectTrigger>
                      <SelectContent>
                        {categories.find(c => c.code === singleFileCategory)?.sub_categories?.map(sub => (
                          <SelectItem key={sub.code} value={sub.code}>{sub.name}</SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
                  </div>
                </div>

                {/* 文件选择 */}
                <div
                  className="border-2 border-dashed rounded-lg p-8 text-center cursor-pointer hover:border-primary/50 transition-colors"
                  onClick={() => singleFileInputRef.current?.click()}
                >
                  {singleFile ? (
                    <div className="flex flex-col items-center gap-2">
                      {getFileIcon(singleFile.type)}
                      <p className="text-lg font-medium">{singleFile.name}</p>
                      <p className="text-sm text-muted-foreground">
                        {formatFileSize(singleFile.size)}
                      </p>
                      <Button
                        variant="outline"
                        size="sm"
                        onClick={(e) => {
                          e.stopPropagation();
                          setSingleFile(null);
                          setSingleFileName("");
                        }}
                      >
                        移除
                      </Button>
                    </div>
                  ) : (
                    <>
                      <FileUp className="h-12 w-12 mx-auto text-muted-foreground mb-4" />
                      <p className="text-lg font-medium">点击选择文件</p>
                      <p className="text-sm text-muted-foreground mt-2">
                        支持：JPG, PNG, PDF, DOC, DWG 等常见格式
                      </p>
                    </>
                  )}
                  <input
                    ref={singleFileInputRef}
                    type="file"
                    className="hidden"
                    onChange={handleSingleFileSelect}
                    accept={[
                      ...SUPPORTED_FILE_TYPES.images,
                      ...SUPPORTED_FILE_TYPES.filteredDocuments,
                      ...SUPPORTED_FILE_TYPES.drawings,
                    ].join(',')}
                  />
                </div>

                {/* 文件名称 */}
                <div>
                  <Label className="mb-2 block">文件名称</Label>
                  <Input
                    value={singleFileName}
                    onChange={(e) => setSingleFileName(e.target.value)}
                    placeholder="输入文件名称"
                    disabled={singleUploadStatus !== 'idle' && singleUploadStatus !== 'editing'}
                  />
                </div>

                {/* OCR 进度 */}
                {ocrLoading && (
                  <div className="flex items-center gap-2 p-3 bg-blue-50 rounded-lg">
                    <Loader2 className="h-4 w-4 animate-spin text-blue-500" />
                    <span className="text-sm text-blue-600">正在OCR识别中...</span>
                  </div>
                )}

                {/* OCR 失败警告 */}
                {ocrResult?.ocr_status === 'failed' && (
                  <div className="flex items-center gap-2 p-3 bg-yellow-50 rounded-lg border border-yellow-200">
                    <span className="text-sm text-yellow-700">⚠️ OCR识别失败：{ocrResult.error_message}，请手动填写相关字段</span>
                  </div>
                )}

                {/* 上传状态 */}
                {singleUploadStatus === 'uploading' && (
                  <div className="space-y-2">
                    <div className="flex items-center justify-between text-sm">
                      <span className="text-primary">上传中...</span>
                      <span>{singleUploadProgress}%</span>
                    </div>
                    <Progress value={singleUploadProgress} />
                  </div>
                )}

      {singleUploadStatus === 'editing' && (
        <div className="mt-4 p-4 border rounded-md bg-muted/30">
          <h4 className="font-medium mb-3">完善档案信息</h4>
          <ArchiveFormRenderer
            schema={formSchema}
            data={{
              ...formData,
              file_name: singleFileName,
              retention_period: singleRetentionPeriod,
              storage_location: singleStorageLocation,
              summary: singleSummary,
              remarks: singleRemarks,
              tags: singleTags as unknown as string,
            } as unknown as Record<string, unknown>}
            onChange={(data) => {
              const d = data as unknown as Record<string, unknown>;
              setSingleFileName(String(d.file_name || ""));
              setSingleRetentionPeriod(String(d.retention_period || "永久"));
              setSingleStorageLocation(String(d.storage_location || ""));
              setSingleSummary(String(d.summary || ""));
              setSingleRemarks(String(d.remarks || ""));
              if (Array.isArray(d.tags)) setSingleTags(d.tags);
            }}
            storageLocations={storageLocations as unknown as Record<string, unknown>[]}
            retentionPeriods={RETENTION_OPTIONS.map(opt => ({ name: opt.value }))}
          />
          <div className="flex justify-end mt-4">
            <Button onClick={handleSaveSingleFileMeta}>保存档案信息</Button>
          </div>
        </div>
      )}

                {singleUploadStatus === 'success' && (
                  <div className="text-center text-green-600">
                    上传成功！
                  </div>
                )}

                {singleUploadStatus === 'error' && (
                  <div className="text-center text-destructive">
                    {singleUploadError || "上传失败"}
                  </div>
                )}
              </div>
            )}

            {/* 批量上传模式 */}
            {uploadMode === 'batch' && (
              <>
                {/* 分类选择 */}
                <div className="grid grid-cols-2 gap-4">
                  <div>
                    <Label className="mb-2 block">档案分类</Label>
                    <Select value={uploadCategory} onValueChange={(v) => {
                      setUploadCategory(v);
                      const cat = categories.find(c => c.code === v);
                      if (cat?.sub_categories && cat.sub_categories.length > 0) {
                        setUploadSubCategory(cat.sub_categories[0].code);
                      }
                    }}>
                      <SelectTrigger>
                        <SelectValue placeholder="选择档案分类" />
                      </SelectTrigger>
                      <SelectContent>
                        {categories.map(cat => (
                          <SelectItem key={cat.code} value={cat.code}>{cat.name}</SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
                  </div>
                  <div>
                    <Label className="mb-2 block">二级分类</Label>
                    <Select value={uploadSubCategory} onValueChange={setUploadSubCategory}>
                      <SelectTrigger>
                        <SelectValue placeholder="选择二级分类" />
                      </SelectTrigger>
                      <SelectContent>
                        {subCategories.map(sub => (
                          <SelectItem key={sub.code} value={sub.code}>{sub.name}</SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
                  </div>
                </div>

                {/* 文件拖拽上传区域 */}
                <div
                  className="border-2 border-dashed rounded-lg p-8 text-center cursor-pointer hover:border-primary/50 transition-colors"
                  onClick={() => fileInputRef.current?.click()}
                  onDragOver={(e) => {
                    e.preventDefault();
                    e.currentTarget.classList.add('border-primary');
                  }}
                  onDragLeave={(e) => {
                    e.currentTarget.classList.remove('border-primary');
                  }}
                  onDrop={(e) => {
                    e.preventDefault();
                    e.currentTarget.classList.remove('border-primary');
                    const files = Array.from(e.dataTransfer.files);
                    if (files.length > 0) {
                      const validFiles: PendingFile[] = files.map(file => ({
                        id: `${Date.now()}-${Math.random().toString(36).slice(2)}`,
                        file,
                        status: 'pending' as const,
                        progress: 0,
                        document: {
                          file_name: file.name.replace(/\.[^.]+$/, ''),
                          file_size: file.size,
                          file_type: file.type,
                          file_format: getFileExtension(file.name),
                          category_code: uploadCategory,
                          sub_category_code: uploadSubCategory,
                          storage_location: getDefaultStorageLocation(),
                        },
                      }));
                      setPendingFiles(prev => [...prev, ...validFiles]);
                      setBatchPage(1); // 重置到第一页
                    }
                  }}
                >
                  <FileUp className="h-12 w-12 mx-auto text-muted-foreground mb-4" />
                  <p className="text-lg font-medium">点击选择文件或拖拽文件到此处</p>
                  <p className="text-sm text-muted-foreground mt-2">
                    支持：JPG, PNG, PDF, DOC, DWG 等常见格式
                  </p>
                  <input
                    ref={fileInputRef}
                    type="file"
                    multiple
                    className="hidden"
                    onChange={handleFileSelect}
                    accept={[
                      ...SUPPORTED_FILE_TYPES.images,
                      ...SUPPORTED_FILE_TYPES.filteredDocuments,
                      ...SUPPORTED_FILE_TYPES.drawings,
                    ].join(',')}
                  />
                </div>

                {/* 文件列表 - 卡片网格模式 */}
                {pendingFiles.length > 0 && (
                  <div className="flex-1 overflow-hidden flex flex-col">
                    {/* 工具栏 */}
                    <div className="flex items-center justify-between mb-3">
                      <span className="text-sm font-medium">待上传文件 ({pendingFiles.length})</span>
                      <div className="flex items-center gap-2">
                        {/* 快速填充按钮 */}
                        <Button
                          variant="outline"
                          size="sm"
                          onClick={() => {
                            if (quickFillTemplate) {
                              // 应用模板到当前页
                              const startIdx = (batchPage - 1) * batchPageSize;
                              const endIdx = Math.min(startIdx + batchPageSize, pendingFiles.length);
                              const currentPageFiles = pendingFiles.slice(startIdx, endIdx);
                              currentPageFiles.forEach(f => {
                                if (f.status === 'pending') {
                                  updatePendingFileMeta(f.id, {
                                    retention_period: quickFillTemplate.retention_period,
                                    summary: quickFillTemplate.summary,
                                    remarks: quickFillTemplate.remarks,
                                  });
                                }
                              });
                              toast.success(`已应用模板到当前页 ${currentPageFiles.length} 个文件`);
                            } else {
                              // 设为模板：取第一个pending文件作为模板
                              const firstPending = pendingFiles.find(f => f.status === 'pending');
                              if (firstPending) {
                                setQuickFillTemplate({
                                  retention_period: firstPending.document?.retention_period || "永久",
                                  summary: firstPending.document?.summary || "",
                                  remarks: firstPending.document?.remarks || "",
                                });
                                toast.success("已设置快速填充模板，可点击「应用」应用到当前页");
                              } else {
                                toast.error("没有待处理的文件");
                              }
                            }
                          }}
                          disabled={pendingFiles.filter(f => f.status === 'pending').length === 0}
                        >
                          <Copy className="h-4 w-4 mr-1" />
                          {quickFillTemplate ? "应用模板" : "设为模板"}
                        </Button>
                        <Button
                          variant="outline"
                          size="sm"
                          onClick={() => {
                            setQuickFillTemplate(null);
                            toast.info("已清除模板");
                          }}
                          disabled={!quickFillTemplate}
                        >
                          清除模板
                        </Button>
                        <Button
                          variant="ghost"
                          size="sm"
                          onClick={() => {
                            setPendingFiles([]);
                            setBatchPage(1);
                            setExpandedFileId(null);
                            setQuickFillTemplate(null);
                          }}
                        >
                          <Trash className="h-4 w-4 mr-1" />
                          清空
                        </Button>
                      </div>
                    </div>

                    {/* 模板提示 */}
                    {quickFillTemplate && (
                      <div className="mb-3 px-3 py-2 rounded-md bg-muted/50 text-xs text-muted-foreground">
                        快速填充模板：保管期限={quickFillTemplate.retention_period}，
                        摘要={quickFillTemplate.summary || "无"}，
                        备注={quickFillTemplate.remarks || "无"}
                      </div>
                    )}

                    {/* 卡片网格 */}
                    <ScrollArea className="flex-1">
                      <div className="grid grid-cols-3 gap-3 mb-3">
                        {pendingFiles
                          .slice((batchPage - 1) * batchPageSize, batchPage * batchPageSize)
                          .map((item) => (
                            <div
                              key={item.id}
                              className={`relative rounded-lg border transition-all duration-200 ${
                                item.status === 'error' ? 'bg-destructive/10 border-destructive' :
                                item.status === 'success' ? 'bg-green-50/50 border-green-500/50' :
                                expandedFileId === item.id ? 'border-primary shadow-md' :
                                'bg-card hover:border-primary/50'
                              }`}
                            >
                              {/* 卡片头部 */}
                              <div className="p-3">
                                <div className="flex items-start gap-2">
                                  <div className="shrink-0 mt-0.5">
                                    {getFileIcon(item.document?.file_type)}
                                  </div>
                                  <div className="flex-1 min-w-0">
                                    <Input
                                      value={item.document?.file_name || ''}
                                      onChange={(e) => updatePendingFileMeta(item.id, { file_name: e.target.value })}
                                      placeholder="文件名称"
                                      className="h-7 text-sm mb-1"
                                      disabled={item.status !== 'pending'}
                                    />
                                    <div className="flex items-center gap-1.5 text-xs text-muted-foreground">
                                      <span className="truncate max-w-[80px]">{item.file.name}</span>
                                      <span>•</span>
                                      <span>{formatFileSize(item.file.size)}</span>
                                    </div>
                                    {/* 状态 */}
                                    <div className="mt-1.5">
                                      {item.status === 'uploading' && (
                                        <div className="space-y-1">
                                          <Progress value={item.progress} className="h-1" />
                                          <span className="text-xs text-primary">上传中 {item.progress}%</span>
                                        </div>
                                      )}
                                      {item.status === 'success' && (
                                        <span className="inline-flex items-center gap-1 text-xs text-green-600">
                                          <Check className="h-3 w-3" /> 上传成功
                                        </span>
                                      )}
                                      {item.status === 'error' && (
                                        <span className="inline-flex items-center gap-1 text-xs text-destructive">
                                          <X className="h-3 w-3" /> {item.error || "失败"}
                                        </span>
                                      )}
                                    </div>
                                  </div>
                                </div>
                              </div>

                              {/* 展开的编辑表单 */}
                              {expandedFileId === item.id && item.status === 'pending' && (
                                <div className="px-3 pb-3 space-y-2 border-t pt-2">
                                  <div className="grid grid-cols-2 gap-2">
                                    <div className="col-span-2">
                                      <Label className="text-xs">保管期限</Label>
                                      <Select
                                        value={item.document?.retention_period || "永久"}
                                        onValueChange={(v) => updatePendingFileMeta(item.id, { retention_period: v })}
                                      >
                                        <SelectTrigger className="h-7 text-xs">
                                          <SelectValue />
                                        </SelectTrigger>
                                        <SelectContent>
                                          <SelectItem value="永久">永久</SelectItem>
                                          <SelectItem value="10年">10年</SelectItem>
                                          <SelectItem value="30年">30年</SelectItem>
                                          <SelectItem value="5年">5年</SelectItem>
                                        </SelectContent>
                                      </Select>
                                    </div>
                                    <div className="col-span-2">
                                      <Label className="text-xs">档案地点</Label>
                                      <Select
                                        value={item.document?.storage_location || getDefaultStorageLocation()}
                                        onValueChange={(v) => updatePendingFileMeta(item.id, { storage_location: v })}
                                      >
                                        <SelectTrigger className="h-7 text-xs">
                                          <SelectValue placeholder="选择地点" />
                                        </SelectTrigger>
                                        <SelectContent>
                                          {storageLocations.map((loc) => (
                                            <SelectItem key={loc.id} value={loc.name}>{loc.name}</SelectItem>
                                          ))}
                                        </SelectContent>
                                      </Select>
                                    </div>
                                    <div className="col-span-2">
                                      <Label className="text-xs">标签</Label>
                                      <Input
                                        value={item.document?.tagInput || ''}
                                        onChange={(e) => updatePendingFileMeta(item.id, { tagInput: e.target.value } as unknown as Partial<Document>)}
                                        placeholder="输入标签（逗号分隔）"
                                        className="h-7 text-xs"
                                      />
                                    </div>
                                    <div className="col-span-2">
                                      <Label className="text-xs">备注</Label>
                                      <Textarea
                                        value={item.document?.remarks || ''}
                                        onChange={(e) => updatePendingFileMeta(item.id, { remarks: e.target.value })}
                                        placeholder="可选"
                                        className="min-h-[60px] text-xs"
                                      />
                                    </div>
                                  </div>
                                </div>
                              )}

                              {/* 卡片底部操作 */}
                              <div className="flex items-center justify-between px-3 py-2 border-t bg-muted/20 rounded-b-lg">
                                {item.status === 'pending' ? (
                                  <>
                                    <Button
                                      variant="ghost"
                                      size="sm"
                                      className="h-6 px-2 text-xs"
                                      onClick={() => setExpandedFileId(expandedFileId === item.id ? null : item.id)}
                                    >
                                      {expandedFileId === item.id ? "收起" : "编辑"}
                                    </Button>
                                    <Button
                                      variant="ghost"
                                      size="icon"
                                      className="h-6 w-6"
                                      onClick={() => removePendingFile(item.id)}
                                    >
                                      <X className="h-3 w-3" />
                                    </Button>
                                  </>
                                ) : (
                                  <span className="text-xs text-muted-foreground">
                                    {item.status === 'success' ? "已完成" : item.status === 'error' ? "失败" : "处理中"}
                                  </span>
                                )}
                              </div>
                            </div>
                          ))}
                      </div>

                      {/* 分页 */}
                      {totalBatchPages > 1 && (
                        <div className="flex items-center justify-center gap-4 py-2">
                          <Button
                            variant="outline"
                            size="sm"
                            onClick={() => setBatchPage(p => Math.max(1, p - 1))}
                            disabled={batchPage === 1}
                          >
                            <ChevronLeft className="h-4 w-4 mr-1" />
                            上一页
                          </Button>
                          <span className="text-sm text-muted-foreground">
                            第 {batchPage} / {totalBatchPages} 页
                          </span>
                          <Button
                            variant="outline"
                            size="sm"
                            onClick={() => setBatchPage(p => Math.min(totalBatchPages, p + 1))}
                            disabled={batchPage === totalBatchPages}
                          >
                            下一页
                            <ChevronRight className="h-4 w-4 ml-1" />
                          </Button>
                        </div>
                      )}
                    </ScrollArea>
                  </div>
                )}
              </>
            )}
          </div>

          <DialogFooter>
            <Button variant="outline" onClick={() => {
              setIsUploadOpen(false);
              resetSingleUpload();
            }}>取消</Button>
            {uploadMode === 'single' ? (
              singleUploadStatus === 'editing' ? (
                <Button onClick={handleSaveSingleFileMeta}>
                  <Save className="mr-2 h-4 w-4" />
                  保存档案信息
                </Button>
              ) : (
                <Button
                  onClick={handleSingleUpload}
                  disabled={!singleFile || !singleFileName.trim() || singleUploadStatus === 'uploading'}
                >
                  {singleUploadStatus === 'uploading' ? (
                    <>
                      <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                      上传中...
                    </>
                  ) : (
                    <>
                      <Upload className="mr-2 h-4 w-4" />
                      上传
                    </>
                  )}
                </Button>
              )
            ) : (
              <Button
                onClick={handleBatchUpload}
                disabled={pendingFiles.length === 0 || isUploading}
              >
                {isUploading ? (
                  <>
                    <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                    上传中...
                  </>
                ) : (
                  <>
                    <Upload className="mr-2 h-4 w-4" />
                    开始上传 ({pendingFiles.length})
                  </>
                )}
              </Button>
            )}
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* 分享弹窗 */}
      <Dialog open={isShareOpen} onOpenChange={setIsShareOpen}>
        <DialogContent className="max-w-md">
          <DialogHeader>
            <DialogTitle>分享文件</DialogTitle>
            <DialogDescription>
              生成分享链接，有效期 {shareExpiry} 小时
            </DialogDescription>
          </DialogHeader>

          <div className="space-y-4">
            <div>
              <Label>链接有效期</Label>
              <Select value={shareExpiry} onValueChange={setShareExpiry}>
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="1">1 小时</SelectItem>
                  <SelectItem value="6">6 小时</SelectItem>
                  <SelectItem value="24">24 小时</SelectItem>
                  <SelectItem value="72">72 小时</SelectItem>
                  <SelectItem value="168">7 天</SelectItem>
                </SelectContent>
              </Select>
            </div>

            <div className="space-y-2">
              <Label>分享链接 ({shareLinks.length} 个文件)</Label>
              <ScrollArea className="h-[200px]">
                <div className="space-y-2">
                  {shareLinks.map((link) => (
                    <div
                      key={link.id}
                      className="space-y-1 p-2 rounded-lg bg-muted/50"
                    >
                      <div className="text-xs font-medium truncate">{link.file_name}</div>
                      <div className="flex items-center gap-2">
                        <Link2 className="h-4 w-4 text-muted-foreground shrink-0" />
                        <Input
                          value={link.link}
                          readOnly
                          className="flex-1 text-xs h-8"
                        />
                        <Button
                          variant="ghost"
                          size="icon"
                          onClick={() => copyLink(link.id, link.link)}
                        >
                          {link.copied ? (
                            <Check className="h-4 w-4 text-green-600" />
                          ) : (
                            <Copy className="h-4 w-4" />
                          )}
                        </Button>
                      </div>
                    </div>
                  ))}
                </div>
              </ScrollArea>
            </div>
          </div>

          <DialogFooter>
            <Button variant="outline" onClick={() => setIsShareOpen(false)}>关闭</Button>
            <Button onClick={handleShare} disabled={isGeneratingLink}>
              {isGeneratingLink ? (
                <>
                  <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                  生成中...
                </>
              ) : (
                <>
                  <RefreshCw className="mr-2 h-4 w-4" />
                  重新生成
                </>
              )}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </CardContent>
    </Card>

      {/* 批量标签对话框已内嵌在上方批量操作栏中 */}

    </div>
  );
}
