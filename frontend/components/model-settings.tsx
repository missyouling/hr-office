"use client";

import { useState, useEffect } from "react";
import { toast } from "sonner";
import {
  listModelConfigs,
  createModelConfig,
  updateModelConfig,
  deleteModelConfig,
  testModelConfig,
  fetchModelsByEndpoint,
  type ModelConfig,
} from "@/lib/api";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Switch } from "@/components/ui/switch";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { AlertDialog, AlertDialogAction, AlertDialogCancel, AlertDialogContent, AlertDialogDescription, AlertDialogFooter, AlertDialogHeader, AlertDialogTitle } from "@/components/ui/alert-dialog";
import { Plus, Edit, Trash2, Eye, EyeOff, Circle, Loader2 } from "lucide-react";
import { cn } from "@/lib/utils";

interface ModelConfigForm {
  config_type: string;
  provider: string;
  model_name: string;
  api_key: string;
  api_endpoint: string;
  enabled: boolean;
  is_default: boolean;
  is_built_in?: boolean;
}

interface TestResult {
  configId: number;
  success: boolean | undefined;
  latency?: number;
  error?: string;
}

const CONFIG_TYPES = [
  { value: "ocr", label: "OCR 文字识别" },
  { value: "llm", label: "LLM 大语言模型" },
  { value: "embedding", label: "Embedding 向量模型" },
  { value: "rerank", label: "Reranker 重排模型" },
];

const PROVIDER_ENDPOINTS: Record<string, string> = {
  siliconflow: "https://api.siliconflow.cn/v1",
  openai: "https://api.openai.com/v1",
  anthropic: "https://api.anthropic.com",
  zhipuai: "https://open.bigmodel.cn/api/paas/v4",
  cohere: "https://api.cohere.ai/v1",
  paddleocr: "https://i0lau9j0n9iavbk3.aistudio-app.com",
  laduan: "https://api.laduan.com",
  tesseract: "",
  custom: "",
  azure: "",
  qwen: "",
  claude: "",
};

const PROVIDERS = {
  ocr: [
    { value: "siliconflow", label: "硅基流动" },
    { value: "laduan", label: "LaDuan AI" },
    { value: "custom", label: "自定义供应商" },
  ],
  llm: [
    { value: "siliconflow", label: "硅基流动" },
    { value: "laduan", label: "LaDuan AI" },
    { value: "openai", label: "OpenAI" },
    { value: "custom", label: "自定义供应商" },
  ],
  embedding: [
    { value: "siliconflow", label: "硅基流动" },
    { value: "custom", label: "自定义供应商" },
  ],
  rerank: [
    { value: "siliconflow", label: "硅基流动" },
    { value: "custom", label: "自定义供应商" },
  ],
};

const SAMPLE_COUNT = 144;

const getHeartbeatColor = (latency?: number, success?: boolean) => {
  if (!success) return "bg-red-500";
  if (!latency) return "bg-gray-300";
  return latency > 500 ? "bg-yellow-500" : "bg-green-500";
};

const getHeartbeatHeight = (latency?: number) => {
  if (!latency) return "h-1";
  if (latency < 300) return "h-1";
  if (latency < 500) return "h-2";
  if (latency < 1000) return "h-3";
  return "h-4";
};

const getCardLabel = (config: ModelConfig) => {
  const labels: { text: string; cls: string }[] = [];
  if (config.is_built_in) labels.push({ text: "内置", cls: "bg-blue-100 text-blue-700 dark:bg-blue-900 dark:text-blue-300" });
  if (config.is_default) labels.push({ text: "默认", cls: "bg-green-100 text-green-700 dark:bg-green-900 dark:text-green-300" });
  else if (config.role === "backup") labels.push({ text: "备用", cls: "bg-gray-100 text-gray-600 dark:bg-gray-800 dark:text-gray-400" });
  return labels;
};

function StatusIndicator({ config, testResult }: { config?: ModelConfig; testResult?: TestResult }) {
  const isEnabled = config?.enabled ?? false;
  const testSuccess = testResult?.success;
  const hasTest = testResult !== undefined;
  const testPending = hasTest && testSuccess === undefined;
  const testPassed = hasTest && testSuccess === true;
  const testFailed = hasTest && testSuccess === false;

  if (isEnabled) {
    if (testPassed) return <Circle className="w-2.5 h-2.5 text-green-500 fill-green-500" />;
    if (testFailed) return <Circle className="w-2.5 h-2.5 text-red-500 fill-red-500" />;
    return <Circle className="w-2.5 h-2.5 text-yellow-500 fill-yellow-500" />;
  }

  if (testFailed || testPending) {
    return <Circle className="w-2.5 h-2.5 text-red-500 fill-red-500" />;
  }
  return <Circle className="w-2.5 h-2.5 text-gray-400 fill-gray-400" />;
}

export function ModelSettings() {
  const [configs, setConfigs] = useState<ModelConfig[]>([]);
  const [loading, setLoading] = useState(true);
  const [isDialogOpen, setIsDialogOpen] = useState(false);
  const [editingId, setEditingId] = useState<number | null>(null);
  const [showApiKey, setShowApiKey] = useState<Record<number, boolean>>({});
  const [testResults, setTestResults] = useState<Record<number, TestResult>>({});
  const [testHistory, setTestHistory] = useState<Record<number, Array<{timestamp: number; success: boolean; latency?: number}>>>({});
  const [deleteConfirmOpen, setDeleteConfirmOpen] = useState(false);
  const [deleteTargetId, setDeleteTargetId] = useState<number | null>(null);
  const [availableModels, setAvailableModels] = useState<string[]>([]);
  const [loadingModels, setLoadingModels] = useState(false);
  const [activeType, setActiveType] = useState("ocr");
  const [detailConfig, setDetailConfig] = useState<ModelConfig | null>(null);

  const [form, setForm] = useState<ModelConfigForm>({
    config_type: "ocr",
    provider: "paddleocr",
    model_name: "",
    api_key: "",
    api_endpoint: "",
    enabled: true,
    is_default: false,
  });

  useEffect(() => {
    loadConfigs();
  }, []);



  const handleAutoTest = async (id: number) => {
    try {
      const startTime = Date.now();
      const result = await testModelConfig(id);
      const latency = Date.now() - startTime;
      const newRecord = { timestamp: Date.now(), success: result.success, latency };
      setTestResults((prev) => ({
        ...prev,
        [id]: { configId: id, success: result.success, latency, error: result.success ? undefined : result.message },
      }));
      setTestHistory((prev) => {
        const history = prev[id] || [];
        const newHistory = [...history, newRecord].slice(-SAMPLE_COUNT);
        return { ...prev, [id]: newHistory };
      });
      if (!result.success) {
        toast.error(`模型 #${id} 连接测试失败: ${result.message}`);
      }
    } catch {
      const newRecord = { timestamp: Date.now(), success: false };
      setTestResults((prev) => ({
        ...prev,
        [id]: { configId: id, success: false, error: "测试失败" },
      }));
      setTestHistory((prev) => {
        const history = prev[id] || [];
        const newHistory = [...history, newRecord].slice(-SAMPLE_COUNT);
        return { ...prev, [id]: newHistory };
      });
    }
  };

  const loadConfigs = async () => {
    try {
      setLoading(true);
      const data = await listModelConfigs();
      setConfigs(data);
      for (const config of data) {
        if (config.enabled) {
          handleAutoTest(config.id);
        }
      }
    } catch (error) {
      toast.error("加载模型配置失败");
      console.error(error);
    } finally {
      setLoading(false);
    }
  };

  const fetchAvailableModels = async (apiKey: string, endpoint: string, provider?: string) => {
    if (!endpoint || !apiKey) {
      setAvailableModels([]);
      return;
    }
    try {
      setLoadingModels(true);
      const finalEndpoint = endpoint || (provider ? PROVIDER_ENDPOINTS[provider] : "");
      if (!finalEndpoint) {
        setAvailableModels([]);
        return;
      }
      const models = await fetchModelsByEndpoint(finalEndpoint, apiKey);
      setAvailableModels(models.map((m) => m.id || m.name));
    } catch {
      setAvailableModels([]);
    } finally {
      setLoadingModels(false);
    }
  };

  const handleOpenDialog = (config?: ModelConfig) => {
    setAvailableModels([]);
    if (config) {
      setEditingId(config.id);
      const providerList = PROVIDERS[config.config_type as keyof typeof PROVIDERS] || [];
      const providerExists = providerList.some(p => p.value === config.provider);
      const provider = providerExists ? config.provider : "custom";

      setForm({
        config_type: config.config_type,
        provider: provider,
        model_name: config.model_name,
        api_key: config.api_key,
        api_endpoint: config.api_endpoint,
        enabled: config.enabled,
        is_default: config.is_default,
      });
      if (config.api_key && config.api_endpoint) {
        fetchAvailableModels(config.api_key, config.api_endpoint, provider);
      }
    } else {
      const defaultType = activeType || "ocr";
      const defaultProvider = PROVIDERS[defaultType as keyof typeof PROVIDERS]?.[0]?.value || "siliconflow";
      setEditingId(null);
      setForm({
        config_type: defaultType,
        provider: defaultProvider,
        model_name: "",
        api_key: "",
        api_endpoint: PROVIDER_ENDPOINTS[defaultProvider] || "",
        enabled: true,
        is_default: false,
        is_built_in: false,
      });
    }
    setIsDialogOpen(true);
  };

  const handleCloseDialog = () => {
    setIsDialogOpen(false);
    setEditingId(null);
    setAvailableModels([]);
  };

  const handleSave = async () => {
    // 仅新增时检查重复，编辑时不检查
    if (!editingId) {
      const duplicates = configs.filter(c =>
        c.api_endpoint === form.api_endpoint &&
        c.api_key === form.api_key &&
        c.model_name === form.model_name
      );
      if (duplicates.length > 0) {
        toast.error("已存在相同配置，无需重复添加");
        return;
      }
    }
    if (!form.model_name.trim()) {
      toast.error("请选择或输入模型");
      return;
    }
    if (!form.api_key.trim()) {
      toast.error("请输入 API Key");
      return;
    }
    if (!form.api_endpoint.trim()) {
      toast.error("请输入 API 端点");
      return;
    }

    try {
      if (editingId) {
        await updateModelConfig(editingId, form);
        toast.success("模型配置已更新");
      } else {
        await createModelConfig(form);
        toast.success("模型配置已创建");
      }
      handleCloseDialog();
      await loadConfigs();
    } catch (error: unknown) {
      const err = error as Record<string, string>;
      const msg: string = err?.message || err?.error || "";
      if (msg.includes("验证失败")) {
        toast.error("验证失败，您所选择的模型不支持当前业务，请重试！");
      } else {
        toast.error(msg || (editingId ? "更新失败" : "创建失败"));
      }
      console.error(error);
    }
  };

  const handleDelete = async () => {
    if (!deleteTargetId) return;
    try {
      await deleteModelConfig(deleteTargetId);
      toast.success("模型配置已删除");
      setDeleteConfirmOpen(false);
      setDeleteTargetId(null);
      await loadConfigs();
    } catch (error) {
      toast.error("删除失败");
      console.error(error);
    }
  };

  const toggleEnabled = async (config: ModelConfig) => {
    try {
      await updateModelConfig(config.id, { enabled: !config.enabled });
      toast.success(config.enabled ? "已停用" : "已启用");
      await loadConfigs();
    } catch {
      toast.error("操作失败");
    }
  };

  const handleCardDoubleClick = (config: ModelConfig) => {
    handleOpenDialog(config);
    setDetailConfig(null);
  };

  const groupedConfigs = CONFIG_TYPES.reduce(
    (acc, type) => {
      acc[type.value] = configs.filter((c) => c.config_type === type.value);
      return acc;
    },
    {} as Record<string, ModelConfig[]>
  );

  if (loading) {
    return (
      <Card>
        <CardContent className="pt-6">
          <div className="text-center text-muted-foreground">加载中...</div>
        </CardContent>
      </Card>
    );
  }

  return (
    <Card className="space-y-2">
      <CardHeader>
        <div className="flex items-center justify-between">
          <div>
            <CardTitle>模型配置管理</CardTitle>
            <CardDescription>管理 OCR、LLM、Embedding、Reranker 等模型配置</CardDescription>
          </div>
          <Button size="sm" onClick={() => handleOpenDialog()}>
            <Plus className="w-3.5 h-3.5 mr-1" />
            新增配置
          </Button>
        </div>
      </CardHeader>
      <CardContent className="space-y-4">
        {/* 按钮式标签栏 */}
        <div className="flex gap-2">
          {CONFIG_TYPES.map((type) => (
            <Button
              key={type.value}
              variant={activeType === type.value ? "default" : "ghost"}
              onClick={() => setActiveType(type.value)}
            >
              {type.label}
            </Button>
          ))}
        </div>

        {/* 内容区域 */}
        {CONFIG_TYPES.map((type) => {
          const typeConfigs = groupedConfigs[type.value];
          const primary = typeConfigs.find((c) => c.role === "primary");

          return (
            <div key={type.value} style={{display: activeType === type.value ? 'block' : 'none'}}>
              <Card>
                <CardHeader className="py-1.5 px-3">
                  <CardTitle className="text-xs flex items-center gap-2">
                    {type.label}
                    {primary && <StatusIndicator config={primary} testResult={testResults[primary.id]} />}
                  </CardTitle>
                  <CardDescription className="text-xs">
                    {typeConfigs.length} 个配置
                  </CardDescription>
                </CardHeader>
                <CardContent className="px-3 pb-2 pt-0">
                  {typeConfigs.length === 0 ? (
                    <div className="text-center py-4 text-muted-foreground text-sm">
                      暂无配置
                    </div>
                  ) : (
                    <div className="grid grid-cols-2 gap-2">
                      {[...typeConfigs]
                        .sort((a, b) => {
                          const pa = a.priority ?? (a.role === "primary" ? 1 : 99);
                          const pb = b.priority ?? (b.role === "primary" ? 1 : 99);
                          return pa - pb;
                        })
                        .map(config => {
                        const status = testResults[config.id];
                        const isOnline = status?.success === true;
                        const isOffline = status?.success === false;
                        const isPending = !status;
                        const label = getCardLabel(config);
                        const history = (testHistory[config.id] || []).slice(-SAMPLE_COUNT);
                        const emptyCount = Math.max(0, SAMPLE_COUNT - history.length);

                        return (
                          <div
                            key={config.id}
                            onClick={() => setDetailConfig(config)}
                            onDoubleClick={() => handleCardDoubleClick(config)}
                            className={cn(
                              "border rounded-lg p-2 cursor-pointer transition-all hover:shadow-md",
                              !config.enabled && "opacity-60",
                              isOnline && config.enabled && "border-green-500 bg-green-50 dark:bg-green-950",
                              isOffline && "border-red-500 bg-red-50 dark:bg-red-950",
                              isPending && config.enabled && "border-yellow-500 bg-yellow-50 dark:bg-yellow-950",
                              !config.enabled && "border-gray-300 bg-gray-50 dark:bg-gray-900"
                            )}
                          >
                            <div className="flex items-center justify-between mb-1">
                              <div className="flex items-center gap-1.5 min-w-0">
                                <span className={cn(
                                  "w-2 h-2 rounded-full animate-pulse flex-shrink-0",
                                  isOnline && config.enabled && "bg-green-500",
                                  isOffline && "bg-red-500",
                                  isPending && config.enabled && "bg-yellow-500",
                                  !config.enabled && "bg-gray-400 animate-none"
                                )} />
                                <span className="text-xs font-medium truncate">{config.model_name}</span>
                              </div>
                              <div className="flex items-center gap-1 flex-shrink-0 ml-1">
                                {!config.enabled && (
                                  <span className="text-[10px] bg-gray-200 dark:bg-gray-700 text-gray-500 dark:text-gray-400 px-1 py-0.5 rounded">停用</span>
                                )}
                                {status?.latency && config.enabled && (
                                  <span className="text-xs text-muted-foreground">{status.latency}ms</span>
                                )}
                                <div className="flex gap-1">
                                  {label.map((l, i) => (
                                    <span key={i} className={cn("text-[10px] px-1 py-0.5 rounded font-medium", l.cls)}>{l.text}</span>
                                  ))}
                                </div>
                              </div>
                            </div>

                            <div className="flex items-end gap-px h-4 mb-1 overflow-hidden">
                              {history.map((record, i) => (
                                <div
                                  key={i}
                                  className={cn(
                                    "w-px rounded-sm flex-shrink-0",
                                    getHeartbeatColor(record.latency, record.success),
                                    getHeartbeatHeight(record.latency)
                                  )}
                                  title={`${new Date(record.timestamp).toLocaleString()} - ${record.success ? '正常' : '失败'}${record.latency ? ` (${record.latency}ms)` : ''}`}
                                />
                              ))}
                              {[...Array(emptyCount)].map((_, i) => (
                                <div key={`empty-${i}`} className="w-px h-1 rounded-sm bg-gray-200 flex-shrink-0" />
                              ))}
                            </div>

                            <div className="flex items-center justify-between">
                              <span className="text-xs text-muted-foreground">{config.provider}</span>
                              <div className="flex gap-1" onClick={e => e.stopPropagation()}>
                                <Button
                                  variant="ghost"
                                  size="sm"
                                  className="h-6 w-6 p-0"
                                  onClick={() => toggleEnabled(config)}
                                  title={config.enabled ? "停用" : "启用"}
                                >
                                  <Circle className={cn("w-3 h-3", config.enabled ? "text-green-500 fill-green-500" : "text-gray-400 fill-gray-400")} />
                                </Button>
                                <Button
                                  variant="ghost"
                                  size="sm"
                                  className="h-6 w-6 p-0"
                                  onClick={() => handleOpenDialog(config)}
                                  title="编辑"
                                >
                                  <Edit className="w-3 h-3" />
                                </Button>
                                <Button
                                  variant="ghost"
                                  size="sm"
                                  className="h-6 w-6 p-0 text-red-500"
                                  onClick={() => {
                                    setDeleteTargetId(config.id);
                                    setDeleteConfirmOpen(true);
                                  }}
                                  title="删除"
                                >
                                  <Trash2 className="w-3 h-3" />
                                </Button>
                              </div>
                            </div>
                          </div>
                        );
                      })}
                    </div>
                  )}
                </CardContent>
              </Card>
            </div>
          );
        })}

      <Dialog open={!!detailConfig} onOpenChange={(open) => !open && setDetailConfig(null)}>
        <DialogContent className="max-w-2xl">
          <DialogHeader>
            <DialogTitle>{detailConfig?.model_name}</DialogTitle>
            <DialogDescription>{detailConfig?.config_type} · {detailConfig?.provider}</DialogDescription>
          </DialogHeader>
          <div className="space-y-4">
            <div className="grid grid-cols-2 gap-4">
              <div>
                <Label className="text-xs text-muted-foreground">供应商</Label>
                <div className="mt-1 text-sm">{detailConfig?.provider}</div>
              </div>
              <div>
                <Label className="text-xs text-muted-foreground">端点</Label>
                <div className="mt-1 text-sm break-all">{detailConfig?.api_endpoint || "—"}</div>
              </div>
              <div>
                <Label className="text-xs text-muted-foreground">角色</Label>
                <div className="mt-1 text-sm">{detailConfig?.role || "—"}</div>
              </div>
              <div>
                <Label className="text-xs text-muted-foreground">状态</Label>
                <div className="mt-1 text-sm">{detailConfig?.enabled ? "启用" : "停用"}</div>
              </div>
            </div>
            <div>
              <Label className="text-xs text-muted-foreground">心跳历史（最近 144 次）</Label>
              <div className="flex flex-wrap gap-px mt-2">
                {(testHistory[detailConfig?.id || 0] || []).slice(-144).map((r, i) => (
                  <div
                    key={i}
                    className={cn("w-1 h-3 rounded-sm", getHeartbeatColor(r.latency, r.success))}
                    title={`${new Date(r.timestamp).toLocaleString()} - ${r.success ? r.latency + "ms" : "失败"}`}
                  />
                ))}
                {(testHistory[detailConfig?.id || 0] || []).length === 0 && (
                  <span className="text-xs text-muted-foreground">暂无心跳记录</span>
                )}
              </div>
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setDetailConfig(null)}>关闭</Button>
            <Button onClick={() => detailConfig && handleCardDoubleClick(detailConfig)}>编辑</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Dialog open={isDialogOpen} onOpenChange={setIsDialogOpen}>
        <DialogContent className="max-w-md">
          <DialogHeader>
            <DialogTitle>
              {editingId ? "编辑模型配置" : "新增模型配置"}
            </DialogTitle>
            <DialogDescription>
              {editingId
                ? "修改现有的模型配置信息"
                : "添加新的模型配置"}
            </DialogDescription>
          </DialogHeader>

          <div className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="config-type">配置类型</Label>
              <Select
                value={form.config_type}
                onValueChange={(v) => {
                  setForm({
                    ...form,
                    config_type: v,
                    provider: PROVIDERS[v as keyof typeof PROVIDERS]?.[0]?.value || "",
                    api_endpoint: PROVIDER_ENDPOINTS[PROVIDERS[v as keyof typeof PROVIDERS]?.[0]?.value || ""] ?? "",
                  });
                  setAvailableModels([]);
                }}
              >
                <SelectTrigger id="config-type">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {CONFIG_TYPES.map((type) => (
                    <SelectItem key={type.value} value={type.value}>
                      {type.label}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>

            <div className="space-y-2">
              <Label htmlFor="provider">供应商</Label>
              <Select
                value={form.provider}
                onValueChange={(v) => {
                  const endpoint = PROVIDER_ENDPOINTS[v] ?? "";
                  setForm({ ...form, provider: v, api_endpoint: endpoint });
                  setAvailableModels([]);
                }}
              >
                <SelectTrigger id="provider">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {(PROVIDERS[form.config_type as keyof typeof PROVIDERS] || []).map(
                    (provider) => (
                      <SelectItem key={provider.value} value={provider.value}>
                        {provider.label}
                      </SelectItem>
                    )
                  )}
                </SelectContent>
              </Select>
            </div>

            <div className="space-y-2">
              <Label htmlFor="api-endpoint">API 端点</Label>
              <Input
                id="api-endpoint"
                placeholder="e.g., https://api.openai.com/v1"
                value={form.api_endpoint}
                onChange={(e) => {
                  const newEndpoint = e.target.value;
                  setForm({ ...form, api_endpoint: newEndpoint });
                  if (form.api_key && newEndpoint) {
                    fetchAvailableModels(form.api_key, newEndpoint, form.provider);
                  } else {
                    setAvailableModels([]);
                  }
                }}
              />
            </div>

            <div className="space-y-2">
              <Label htmlFor="api-key">API Key</Label>
              <div className="flex gap-2">
                <Input
                  id="api-key"
                  type={showApiKey[editingId || 0] ? "text" : "password"}
                  placeholder="sk-..."
                  value={form.api_key}
                  onChange={(e) => {
                    const newKey = e.target.value;
                    setForm({ ...form, api_key: newKey });
                    if (newKey && form.api_endpoint) {
                      fetchAvailableModels(newKey, form.api_endpoint, form.provider);
                    } else {
                      setAvailableModels([]);
                    }
                  }}
                />
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={() =>
                    setShowApiKey({
                      ...showApiKey,
                      [editingId || 0]: !showApiKey[editingId || 0],
                    })
                  }
                >
                  {showApiKey[editingId || 0] ? (
                    <EyeOff className="w-4 h-4" />
                  ) : (
                    <Eye className="w-4 h-4" />
                  )}
                </Button>
              </div>
            </div>

            <div className="space-y-2">
              <Label htmlFor="model-name">
                模型
                {loadingModels && <Loader2 className="inline w-3 h-3 ml-1 animate-spin" />}
              </Label>
              {!loadingModels && availableModels.length === 0 ? (
                <Input
                  id="model-name"
                  placeholder="无法加载模型，请手动输入模型名称"
                  value={form.model_name}
                  onChange={(e) => setForm({ ...form, model_name: e.target.value })}
                />
              ) : (
                <Select
                  value={form.model_name}
                  onValueChange={(v) => setForm({ ...form, model_name: v })}
                  disabled={loadingModels || availableModels.length === 0}
                >
                  <SelectTrigger id="model-name">
                    <SelectValue
                      placeholder={
                        loadingModels
                          ? "正在加载模型列表..."
                          : availableModels.length === 0
                          ? "请先填写 API Key 和端点"
                          : "选择模型"
                      }
                    />
                  </SelectTrigger>
                  <SelectContent>
                    {availableModels.map((m) => (
                      <SelectItem key={m} value={m}>
                        {m}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              )}
            </div>

            <div className="flex items-center space-x-2">
              <Switch
                id="enabled"
                checked={form.enabled}
                onCheckedChange={(checked) =>
                  setForm({ ...form, enabled: checked })
                }
              />
              <Label htmlFor="enabled">启用此配置</Label>
            </div>

            <div className="flex items-center space-x-2">
              <Switch
                id="is-default"
                checked={form.is_default}
                onCheckedChange={(checked) =>
                  setForm({ ...form, is_default: checked })
                }
              />
              <Label htmlFor="is-default">设为默认配置</Label>
            </div>
          </div>

          <DialogFooter>
            <Button variant="outline" onClick={handleCloseDialog}>
              取消
            </Button>
            <Button onClick={handleSave}>
              {editingId ? "更新" : "创建"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <AlertDialog open={deleteConfirmOpen} onOpenChange={setDeleteConfirmOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>确认删除</AlertDialogTitle>
            <AlertDialogDescription>
              确定要删除这个模型配置吗？此操作无法撤销。
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>取消</AlertDialogCancel>
            <AlertDialogAction onClick={handleDelete}>
              删除
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
      </CardContent>
    </Card>
  );
}
