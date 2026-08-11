"use client";

import { useState, useEffect, Fragment, useCallback } from "react";
import { useRouter } from "next/navigation";
import { toast } from "sonner";
import { useAuth } from "@/lib/auth";
import {
  fetchRoles,
  fetchPermissions,
  fetchRolePermissions,
  updateRolePermissions,
  updateRole,
  deleteRole,
  type Role,
  type Permission,
  fetchAnnouncements,
  createAnnouncement,
  updateAnnouncement,
  deleteAnnouncement,
  type Announcement,
  fetchDocumentCategories,
  fetchFieldDefinitions,
  createFieldDefinition,
  updateFieldDefinition,
  deleteFieldDefinition,
  type ArchiveFieldDefinition,
  type DocumentCategory,
  type DocumentSubCategory,
  fetchRetentionPeriods,
  createRetentionPeriod,
  updateRetentionPeriod,
  deleteRetentionPeriod,
  type RetentionPeriod,
  fetchStorageLocations,
  createStorageLocation,
  updateStorageLocation,
  deleteStorageLocation,
  type StorageLocation,
  fetchCodeRules,
  createCodeRule,
  updateCodeRule,
  deleteCodeRule,
  getCodeRulePreview,
  type CodeRule,
  type CodeRulePreview,
  updateCategoryCode,
  createCategoryCode,
  deleteCategory,
  createSubCategory,
  updateSubCategoryCode,
  deleteSubCategory,
  fetchFieldsBySubCategory,
  listStorageConfigs,
  createStorageConfig,
  updateStorageConfig,
  deleteStorageConfig,
  testStorageConnection,
  uploadStorageFile,
  listStorageFiles,
  deleteStorageFile,
  getStorageFileDownloadUrl,
  listNotificationConfigs,
  createNotificationConfig,
  updateNotificationConfig,
  testNotification,
  type NotificationConfig,
  fetchModelUsageStats,
  fetchModelUsageByModel,
  fetchModelUsageTrend,
  type ModelUsageStatsResponse,
  type ModelUsageByModelItem,
  type ModelUsageTrendItem,
  listStorageModules,
  listStorageRulesEnhanced,
  createStorageRuleEnhanced,
  updateStorageRuleEnhanced,
  deleteStorageRuleEnhanced,
  listStorageDirectoriesEnhanced,
} from "@/lib/api";
import type { AuditLog, StorageConfig, SysFile, StorageModuleConfig, StorageRule } from "@/lib/types";
import { cn } from "@/lib/utils";


import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Switch } from "@/components/ui/switch";
import { Checkbox } from "@/components/ui/checkbox";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { Dialog, DialogContent, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { AlertDialog, AlertDialogAction, AlertDialogCancel, AlertDialogContent, AlertDialogDescription, AlertDialogFooter, AlertDialogHeader, AlertDialogTitle } from "@/components/ui/alert-dialog";
import { Badge } from "@/components/ui/badge";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Textarea } from "@/components/ui/textarea";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { RefreshCw, Download, Eye, Plus, Trash2, Edit, Upload, HardDrive, Cloud, Server, Pencil, ShieldCheck, Search, Settings2, Info, Sliders, Circle, Database } from "lucide-react";
import { format } from "date-fns";
import { ModelSettings } from "./model-settings";
import { SystemLogs } from "./system-logs";
import { LineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip, Legend, ResponsiveContainer, PieChart, Pie, Cell, Area } from "recharts";

// ============ 方案 A：一级 Tab + 二级侧边栏分组结构 ============

interface SettingsTabItem {
  id: string;
  label: string;
  icon?: string;
}

interface SettingsTabGroup {
  group: string;
  icon: string;
  items: SettingsTabItem[];
}

const SETTINGS_TAB_GROUPS: SettingsTabGroup[] = [
  {
    group: "基础配置",
    icon: "🔧",
    items: [
      { id: "announcements", label: "公告管理" },
      { id: "logs", label: "系统日志" },
      { id: "maintenance", label: "系统维护" },
      { id: "model-usage", label: "模型分布" },
      { id: "storage", label: "存储配置" },
    ],
  },
  {
    group: "全局配置",
    icon: "⚙️",
    items: [
      { id: "ai", label: "模型配置" },
      { id: "notification", label: "通知配置" },
    ],
  },
  {
    group: "档案配置",
    icon: "📁",
    items: [
      { id: "archive-classification", label: "档案分类" },
      { id: "archive-global", label: "全局配置" },
    ],
  },
  {
    group: "权限配置",
    icon: "👥",
    items: [
      { id: "roles", label: "角色权限" },
    ],
  },
];

// 用于 flat 查找
const SETTINGS_TABS = SETTINGS_TAB_GROUPS.flatMap((g) => g.items);

// 根据 tab id 查找所属分组
function getGroupForTab(tabId: string): SettingsTabGroup | undefined {
  return SETTINGS_TAB_GROUPS.find((g) => g.items.some((item) => item.id === tabId));
}

// 模型配置 Tab
function ModelConfigTab() {
  const [activeTab, setActiveTab] = useState<"chat" | "embedding" | "rerank">("chat");
  const [testing, setTesting] = useState<Record<string, boolean>>({});
  const [statuses, setStatuses] = useState<Record<string, "idle" | "success" | "error">>({});

  // 通用大模型配置
  const [chatConfig, setChatConfig] = useState({
    provider: "openai",
    api_key: "",
    model: "gpt-4o",
    endpoint: "",
    enabled: false,
  });

  // 向量模型配置
  const [embeddingConfig, setEmbeddingConfig] = useState({
    provider: "openai",
    api_key: "",
    model: "text-embedding-3-small",
    endpoint: "",
    enabled: false,
  });

  // 重排模型配置
  const [rerankConfig, setRerankConfig] = useState({
    provider: "cohere",
    api_key: "",
    model: "rerank-multilingual-v2.0",
    endpoint: "",
    enabled: false,
  });

  // 测试连接
  const handleTest = async (type: string, config: Record<string, unknown>) => {
    setTesting({ ...testing, [type]: true });
    setStatuses({ ...statuses, [type]: "idle" });
    try {
      // TODO: 调用后端 API 测试连接
      await new Promise(resolve => setTimeout(resolve, 1000));
      setStatuses({ ...statuses, [type]: "success" });
      toast.success("连接测试成功");
    } catch (error) {
      setStatuses({ ...statuses, [type]: "error" });
      toast.error("连接失败");
    } finally {
      setTesting({ ...testing, [type]: false });
    }
  };

  // 保存配置
  const handleSave = () => {
    toast.success("模型配置已保存");
  };

  // 状态指示器组件
  const StatusIndicator = ({ status }: { status: "idle" | "success" | "error" }) => {
    if (status === "idle") return <span className="text-muted-foreground text-sm">未测试</span>;
    if (status === "success") return <span className="text-green-600 text-sm flex items-center gap-1"><span className="w-2 h-2 rounded-full bg-green-500"></span> 已连接</span>;
    return <span className="text-red-600 text-sm flex items-center gap-1"><span className="w-2 h-2 rounded-full bg-red-500"></span> 连接失败</span>;
  };

  // 通用配置表单
  const renderChatConfig = () => (
    <div className="space-y-4">
      <div className="flex items-center space-x-2">
        <Switch
          id="chat-enabled"
          checked={chatConfig.enabled}
          onCheckedChange={(checked) => setChatConfig({ ...chatConfig, enabled: checked })}
        />
        <Label htmlFor="chat-enabled">启用通用大模型</Label>
        <StatusIndicator status={statuses.chat || "idle"} />
      </div>

      <div className="grid gap-2">
        <Label htmlFor="chat-provider">AI 厂商</Label>
        <Select value={chatConfig.provider} onValueChange={(v) => setChatConfig({ ...chatConfig, provider: v })}>
          <SelectTrigger>
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="openai">OpenAI</SelectItem>
            <SelectItem value="azure">Azure OpenAI</SelectItem>
            <SelectItem value="qwen">阿里 Qwen</SelectItem>
            <SelectItem value="local">本地模型 (Ollama)</SelectItem>
          </SelectContent>
        </Select>
      </div>

      <div className="grid gap-2">
        <Label htmlFor="chat-endpoint">API 地址</Label>
        <Input
          id="chat-endpoint"
          placeholder="https://api.openai.com/v1"
          value={chatConfig.endpoint}
          onChange={(e) => setChatConfig({ ...chatConfig, endpoint: e.target.value })}
        />
      </div>

      <div className="grid gap-2">
        <Label htmlFor="chat-model">模型</Label>
        <Input
          id="chat-model"
          placeholder="gpt-4o"
          value={chatConfig.model}
          onChange={(e) => setChatConfig({ ...chatConfig, model: e.target.value })}
        />
      </div>

      <div className="grid gap-2">
        <Label htmlFor="chat-key">API Key</Label>
        <Input
          id="chat-key"
          type="password"
          placeholder="sk-..."
          value={chatConfig.api_key}
          onChange={(e) => setChatConfig({ ...chatConfig, api_key: e.target.value })}
        />
      </div>

      <div className="flex gap-2">
        <Button variant="outline" onClick={() => handleTest("chat", chatConfig)} disabled={testing.chat}>
          {testing.chat ? "测试中..." : "测试连接"}
        </Button>
        <Button onClick={handleSave}>保存配置</Button>
      </div>
    </div>
  );

  // 向量模型配置表单
  const renderEmbeddingConfig = () => (
    <div className="space-y-4">
      <div className="flex items-center space-x-2">
        <Switch
          id="embedding-enabled"
          checked={embeddingConfig.enabled}
          onCheckedChange={(checked) => setEmbeddingConfig({ ...embeddingConfig, enabled: checked })}
        />
        <Label htmlFor="embedding-enabled">启用向量模型</Label>
        <StatusIndicator status={statuses.embedding || "idle"} />
      </div>

      <div className="grid gap-2">
        <Label htmlFor="embedding-provider">向量模型厂商</Label>
        <Select value={embeddingConfig.provider} onValueChange={(v) => setEmbeddingConfig({ ...embeddingConfig, provider: v })}>
          <SelectTrigger>
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="openai">OpenAI</SelectItem>
            <SelectItem value="azure">Azure OpenAI</SelectItem>
            <SelectItem value="qwen">阿里 Qwen</SelectItem>
            <SelectItem value="zhipuai">智谱 AI</SelectItem>
            <SelectItem value="local">本地模型</SelectItem>
          </SelectContent>
        </Select>
      </div>

      <div className="grid gap-2">
        <Label htmlFor="embedding-endpoint">API 地址</Label>
        <Input
          id="embedding-endpoint"
          placeholder="https://api.openai.com/v1"
          value={embeddingConfig.endpoint}
          onChange={(e) => setEmbeddingConfig({ ...embeddingConfig, endpoint: e.target.value })}
        />
      </div>

      <div className="grid gap-2">
        <Label htmlFor="embedding-model">向量模型</Label>
        <Input
          id="embedding-model"
          placeholder="text-embedding-3-small"
          value={embeddingConfig.model}
          onChange={(e) => setEmbeddingConfig({ ...embeddingConfig, model: e.target.value })}
        />
      </div>

      <div className="grid gap-2">
        <Label htmlFor="embedding-key">API Key</Label>
        <Input
          id="embedding-key"
          type="password"
          placeholder="sk-..."
          value={embeddingConfig.api_key}
          onChange={(e) => setEmbeddingConfig({ ...embeddingConfig, api_key: e.target.value })}
        />
      </div>

      <div className="flex gap-2">
        <Button variant="outline" onClick={() => handleTest("embedding", embeddingConfig)} disabled={testing.embedding}>
          {testing.embedding ? "测试中..." : "测试连接"}
        </Button>
        <Button onClick={handleSave}>保存配置</Button>
      </div>
    </div>
  );

  // 重排模型配置表单
  const renderRerankConfig = () => (
    <div className="space-y-4">
      <div className="flex items-center space-x-2">
        <Switch
          id="rerank-enabled"
          checked={rerankConfig.enabled}
          onCheckedChange={(checked) => setRerankConfig({ ...rerankConfig, enabled: checked })}
        />
        <Label htmlFor="rerank-enabled">启重视排模型</Label>
        <StatusIndicator status={statuses.rerank || "idle"} />
      </div>

      <div className="grid gap-2">
        <Label htmlFor="rerank-provider">重排模型厂商</Label>
        <Select value={rerankConfig.provider} onValueChange={(v) => setRerankConfig({ ...rerankConfig, provider: v })}>
          <SelectTrigger>
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="cohere">Cohere</SelectItem>
            <SelectItem value="openai">OpenAI</SelectItem>
            <SelectItem value="qwen">阿里 Qwen</SelectItem>
            <SelectItem value="local">本地模型</SelectItem>
          </SelectContent>
        </Select>
      </div>

      <div className="grid gap-2">
        <Label htmlFor="rerank-endpoint">API 地址</Label>
        <Input
          id="rerank-endpoint"
          placeholder="https://api.cohere.ai/v1"
          value={rerankConfig.endpoint}
          onChange={(e) => setRerankConfig({ ...rerankConfig, endpoint: e.target.value })}
        />
      </div>

      <div className="grid gap-2">
        <Label htmlFor="rerank-model">重排模型</Label>
        <Input
          id="rerank-model"
          placeholder="rerank-multilingual-v2.0"
          value={rerankConfig.model}
          onChange={(e) => setRerankConfig({ ...rerankConfig, model: e.target.value })}
        />
      </div>

      <div className="grid gap-2">
        <Label htmlFor="rerank-key">API Key</Label>
        <Input
          id="rerank-key"
          type="password"
          placeholder="..."
          value={rerankConfig.api_key}
          onChange={(e) => setRerankConfig({ ...rerankConfig, api_key: e.target.value })}
        />
      </div>

      <div className="flex gap-2">
        <Button variant="outline" onClick={() => handleTest("rerank", rerankConfig)} disabled={testing.rerank}>
          {testing.rerank ? "测试中..." : "测试连接"}
        </Button>
        <Button onClick={handleSave}>保存配置</Button>
      </div>
    </div>
  );

  return (
    <Card>
      <CardHeader>
        <CardTitle>模型配置</CardTitle>
        <CardDescription>配置通用大模型、向量模型和重排模型</CardDescription>
      </CardHeader>
        <CardContent className="space-y-6">
          {/* 按钮式标签栏 */}
          <div className="flex gap-2">
            <Button
              variant={activeTab === "chat" ? "default" : "ghost"}
              onClick={() => setActiveTab("chat")}
            >
              通用大模型
            </Button>
            <Button
              variant={activeTab === "embedding" ? "default" : "ghost"}
              onClick={() => setActiveTab("embedding")}
            >
              向量模型
            </Button>
            <Button
              variant={activeTab === "rerank" ? "default" : "ghost"}
              onClick={() => setActiveTab("rerank")}
            >
              重排模型
            </Button>
          </div>

          {/* 内容区域 */}
          <div className="mt-4 space-y-4">
            {activeTab === "chat" && renderChatConfig()}
            {activeTab === "embedding" && renderEmbeddingConfig()}
            {activeTab === "rerank" && renderRerankConfig()}
          </div>
      </CardContent>
    </Card>
  );
}

// ============ SMTP 配置 Tab ============
function SMTPConfigTab() {
  const [activeChannel, setActiveChannel] = useState<string | null>("smtp");
  const [loading, setLoading] = useState(true);
  const [configs, setConfigs] = useState<NotificationConfig[]>([]);

  const [smtpConfig, setSmtpConfig] = useState({
    enabled: false,
    host: "",
    port: "465",
    username: "",
    password: "",
    from: "",
    from_name: "人事系统",
    reply_to: "",
    encryption: "ssl",
    server_name: "",
  });
  const [smsConfig, setSmsConfig] = useState({
    enabled: false,
    access_key_id: "",
    access_key_secret: "",
    sign_name: "",
    template_code: "",
  });
  const [telegramConfig, setTelegramConfig] = useState({
    enabled: false,
    bot_token: "",
    chat_id: "",
  });
  const [webhookConfig, setWebhookConfig] = useState({
    enabled: false,
    url: "",
    method: "POST",
    auth: "",
  });
  const [testing, setTesting] = useState(false);
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    loadConfigs();
  }, []);

  const loadConfigs = async () => {
    setLoading(true);
    try {
      const data = await listNotificationConfigs();
      setConfigs(data);
      data.forEach(c => {
        const cfg = (c.config ?? {}) as Record<string, string>;
        if (c.channel === "smtp") setSmtpConfig({ enabled: c.enabled, host: cfg.host || "", port: cfg.port || "587", username: cfg.username || "", password: cfg.password || "", from: cfg.from || "", from_name: cfg.from_name || "人事系统", reply_to: cfg.reply_to || "", encryption: cfg.encryption || "ssl", server_name: cfg.server_name || "" });
        if (c.channel === "sms") setSmsConfig({ enabled: c.enabled, access_key_id: cfg.access_key_id || "", access_key_secret: cfg.access_key_secret || "", sign_name: cfg.sign_name || "", template_code: cfg.template_code || "" });
        if (c.channel === "telegram") setTelegramConfig({ enabled: c.enabled, bot_token: cfg.bot_token || "", chat_id: cfg.chat_id || "" });
        if (c.channel === "webhook") setWebhookConfig({ enabled: c.enabled, url: cfg.url || "", method: cfg.method || "POST", auth: cfg.auth || "" });
      });
    } catch (e) { console.error("加载配置失败:", e); }
    finally { setLoading(false); }
  };

  const handleSave = async (channel: string) => {
    setSaving(true);
    try {
      const existing = configs.find(c => c.channel === channel);
      const configData = channel === "smtp" ? smtpConfig : channel === "sms" ? smsConfig : channel === "telegram" ? telegramConfig : webhookConfig;
      const payload = { channel, name: channel.toUpperCase(), enabled: configData.enabled, config: configData };
      if (existing?.id) { await updateNotificationConfig(existing.id, payload); }
      else { await createNotificationConfig(payload); }
      toast.success(channel.toUpperCase() + " 配置已保存");
      loadConfigs();
    } catch (error) { console.error("保存失败:", error); toast.error("保存失败"); }
    finally { setSaving(false); }
  };

  const handleTest = async (channel: string) => {
    setTesting(true);
    try {
      const configData = channel === "smtp" ? smtpConfig : channel === "sms" ? smsConfig : channel === "telegram" ? telegramConfig : webhookConfig;
      await testNotification(channel, channel === "smtp" ? smtpConfig.from : channel === "sms" ? "13800138000" : channel === "telegram" ? telegramConfig.chat_id : "", configData);
      toast.success("测试消息发送成功");
    } catch (error) { console.error("测试失败:", error); toast.error("测试发送失败"); }
    finally { setTesting(false); }
  };

  if (loading) return <div className="p-4">加载中...</div>;

  return (
    <Card>
      <CardHeader><CardTitle>通知配置</CardTitle><CardDescription>配置邮件、短信、Telegram、Webhook 等通知渠道</CardDescription></CardHeader>
      <CardContent className="space-y-4">
        {/* 按钮式标签栏 */}
        <div className="flex gap-2">
          <Button variant={activeChannel === "smtp" ? "default" : "ghost"} onClick={() => setActiveChannel("smtp")}>SMTP</Button>
          <Button variant={activeChannel === "sms" ? "default" : "ghost"} onClick={() => setActiveChannel("sms")}>阿里短信</Button>
          <Button variant={activeChannel === "telegram" ? "default" : "ghost"} onClick={() => setActiveChannel("telegram")}>Telegram</Button>
          <Button variant={activeChannel === "webhook" ? "default" : "ghost"} onClick={() => setActiveChannel("webhook")}>Webhook</Button>
        </div>

        {activeChannel === "smtp" && <div className="mt-4">
          <div className="space-y-4">
            <div className="flex items-center space-x-2"><Switch checked={smtpConfig.enabled} onCheckedChange={(v) => setSmtpConfig({ ...smtpConfig, enabled: v })} /><Label>启用 SMTP</Label></div>
            <div className="grid gap-2"><Label>SMTP 地址</Label><Input value={smtpConfig.host} onChange={(e) => setSmtpConfig({ ...smtpConfig, host: e.target.value })} placeholder="smtp.example.com" /></div>
            <div className="grid gap-2">
              <Label>加密方式</Label>
              <Select value={smtpConfig.encryption} onValueChange={(v) => { const m = { ssl: "465", tls: "587", none: "25" }; setSmtpConfig({ ...smtpConfig, encryption: v, port: m[v as keyof typeof m] || smtpConfig.port }); }}>
                <SelectTrigger><SelectValue /></SelectTrigger>
                <SelectContent>
                  <SelectItem value="none">无加密 (端口 25)</SelectItem>
                  <SelectItem value="ssl">SSL/TLS (端口 465)</SelectItem>
                  <SelectItem value="tls">STARTTLS (端口 587)</SelectItem>
                </SelectContent>
              </Select>
            </div>
            <div className="grid gap-2"><Label>端口</Label><Input value={smtpConfig.port} onChange={(e) => setSmtpConfig({ ...smtpConfig, port: e.target.value })} placeholder="587" /></div>
            <div className="grid gap-2"><Label>发件人邮箱</Label><Input value={smtpConfig.from} onChange={(e) => setSmtpConfig({ ...smtpConfig, from: e.target.value })} placeholder="noreply@example.com" /></div>
            <div className="grid gap-2"><Label>发件人名称</Label><Input value={smtpConfig.from_name} onChange={(e) => setSmtpConfig({ ...smtpConfig, from_name: e.target.value })} placeholder="人事系统" /></div>
            <div className="grid gap-2"><Label>用户名</Label><Input value={smtpConfig.username} onChange={(e) => setSmtpConfig({ ...smtpConfig, username: e.target.value })} /></div>
            <div className="grid gap-2"><Label>密码</Label><Input type="password" value={smtpConfig.password} onChange={(e) => setSmtpConfig({ ...smtpConfig, password: e.target.value })} /></div>
            <div className="grid gap-2"><Label>Server Name (TLS验证)</Label><Input value={smtpConfig.server_name} onChange={(e) => setSmtpConfig({ ...smtpConfig, server_name: e.target.value })} placeholder="smtp.example.com (可选)" /></div>
            <div className="grid gap-2"><Label>回复地址 (Reply-To)</Label><Input value={smtpConfig.reply_to} onChange={(e) => setSmtpConfig({ ...smtpConfig, reply_to: e.target.value })} placeholder="reply@example.com (可选)" /></div>
            <div className="flex gap-2"><Button onClick={() => handleSave("smtp")} disabled={saving}>保存</Button><Button variant="outline" onClick={() => handleTest("smtp")} disabled={testing}>测试</Button></div>
          </div>
        </div>}
        {activeChannel === "sms" && <div className="mt-4">
          <div className="space-y-4">
            <div className="flex items-center space-x-2"><Switch checked={smsConfig.enabled} onCheckedChange={(v) => setSmsConfig({ ...smsConfig, enabled: v })} /><Label>启用阿里短信</Label></div>
            <div className="grid gap-2"><Label>AccessKeyId</Label><Input value={smsConfig.access_key_id} onChange={(e) => setSmsConfig({ ...smsConfig, access_key_id: e.target.value })} /></div>
            <div className="grid gap-2"><Label>AccessKeySecret</Label><Input type="password" value={smsConfig.access_key_secret} onChange={(e) => setSmsConfig({ ...smsConfig, access_key_secret: e.target.value })} /></div>
            <div className="grid gap-2"><Label>签名</Label><Input value={smsConfig.sign_name} onChange={(e) => setSmsConfig({ ...smsConfig, sign_name: e.target.value })} placeholder="【公司名】" /></div>
            <div className="grid gap-2"><Label>模板码</Label><Input value={smsConfig.template_code} onChange={(e) => setSmsConfig({ ...smsConfig, template_code: e.target.value })} /></div>
            <div className="flex gap-2"><Button onClick={() => handleSave("sms")} disabled={saving}>保存</Button><Button variant="outline" onClick={() => handleTest("sms")} disabled={testing}>测试</Button></div>
          </div>
        </div>}
        {activeChannel === "telegram" && <div className="mt-4">
          <div className="space-y-4">
            <div className="flex items-center space-x-2"><Switch checked={telegramConfig.enabled} onCheckedChange={(v) => setTelegramConfig({ ...telegramConfig, enabled: v })} /><Label>启用 Telegram</Label></div>
            <div className="grid gap-2"><Label>Bot Token</Label><Input value={telegramConfig.bot_token} onChange={(e) => setTelegramConfig({ ...telegramConfig, bot_token: e.target.value })} placeholder="123456:ABC-DEF" /></div>
            <div className="grid gap-2"><Label>Chat ID</Label><Input value={telegramConfig.chat_id} onChange={(e) => setTelegramConfig({ ...telegramConfig, chat_id: e.target.value })} placeholder="123456789" /></div>
            <div className="flex gap-2"><Button onClick={() => handleSave("telegram")} disabled={saving}>保存</Button><Button variant="outline" onClick={() => handleTest("telegram")} disabled={testing}>测试</Button></div>
          </div>
        </div>}
        {activeChannel === "webhook" && <div className="mt-4">
          <div className="space-y-4">
            <div className="flex items-center space-x-2"><Switch checked={webhookConfig.enabled} onCheckedChange={(v) => setWebhookConfig({ ...webhookConfig, enabled: v })} /><Label>启用 Webhook</Label></div>
            <div className="grid gap-2"><Label>URL</Label><Input value={webhookConfig.url} onChange={(e) => setWebhookConfig({ ...webhookConfig, url: e.target.value })} placeholder="https://example.com/webhook" /></div>
            <div className="grid gap-2"><Label>方法</Label><Input value={webhookConfig.method} onChange={(e) => setWebhookConfig({ ...webhookConfig, method: e.target.value })} placeholder="POST" /></div>
            <div className="grid gap-2"><Label>认证</Label><Input value={webhookConfig.auth} onChange={(e) => setWebhookConfig({ ...webhookConfig, auth: e.target.value })} placeholder="Bearer token" /></div>
            <div className="flex gap-2"><Button onClick={() => handleSave("webhook")} disabled={saving}>保存</Button><Button variant="outline" onClick={() => handleTest("webhook")} disabled={testing}>测试</Button></div>
          </div>
        </div>}
      </CardContent>
    </Card>
  );
}

function formatNum(n: number): string {
  if (n >= 1_000_000) return `${(n / 1_000_000).toFixed(1)}M`;
  if (n >= 1_000) return `${(n / 1_000).toFixed(1)}K`;
  return String(n);
}

// 货币格式化函数（CNY）
function formatCny(amount: number): string {
  const formatter = new Intl.NumberFormat('zh-CN', {
    style: 'currency',
    currency: 'CNY',
    minimumFractionDigits: 2,
    maximumFractionDigits: 2,
  });
  const parts = formatter.formatToParts(amount);
  return parts.map(part => {
    if (part.type === 'currency') {
      return '¥';
    }
    return part.value;
  }).join('');
}

// ============ 模型使用统计 Tab ============
function ModelUsageTab() {
  const [timeRange, setTimeRange] = useState("today");
  const [customFrom, setCustomFrom] = useState("");
  const [customTo, setCustomTo] = useState("");
  const [loading, setLoading] = useState(false);
  const [usageData, setUsageData] = useState<ModelUsageByModelItem[]>([]);
  const [stats, setStats] = useState<ModelUsageStatsResponse | null>(null);
  const [trendData, setTrendData] = useState<ModelUsageTrendItem[]>([]);
  const [selectedConfigType, setSelectedConfigType] = useState("");
  const [selectedModel, setSelectedModel] = useState("");

  const [allConfigTypes, setAllConfigTypes] = useState<string[]>([]);
  const [allModelNames, setAllModelNames] = useState<string[]>([]);

  const buildDateRangeParams = () => {
    const params: { start_date?: string; end_date?: string } = {};
    const now = new Date();
    if (timeRange === "today") {
      params.start_date = now.toISOString().split("T")[0];
      params.end_date = now.toISOString().split("T")[0];
    } else if (timeRange === "week") {
      const weekAgo = new Date(now);
      weekAgo.setDate(weekAgo.getDate() - 7);
      params.start_date = weekAgo.toISOString().split("T")[0];
      params.end_date = now.toISOString().split("T")[0];
    } else if (timeRange === "month") {
      const monthAgo = new Date(now);
      monthAgo.setMonth(monthAgo.getMonth() - 1);
      params.start_date = monthAgo.toISOString().split("T")[0];
      params.end_date = now.toISOString().split("T")[0];
    } else if (timeRange === "custom" && customFrom && customTo) {
      params.start_date = customFrom;
      params.end_date = customTo;
    }
    return params;
  };

  const loadData = async () => {
    if (timeRange === "custom" && customFrom && customTo && customFrom > customTo) {
      toast.error("开始日期不能晚于结束日期");
      return;
    }

    setLoading(true);
    try {
      const dateRange = buildDateRangeParams();
      const statsParams: { start_date?: string; end_date?: string; config_type?: string; model_name?: string } = { ...dateRange };
      const trendParams: { period?: string; config_type?: string; model_name?: string; start_date?: string; end_date?: string } = { ...dateRange, period: "day" };
      const byModelParams: { config_type?: string; model_name?: string; start_date?: string; end_date?: string } = { ...dateRange };

      if (selectedConfigType) {
        statsParams.config_type = selectedConfigType;
        trendParams.config_type = selectedConfigType;
        byModelParams.config_type = selectedConfigType;
      }
      if (selectedModel) {
        statsParams.model_name = selectedModel;
        trendParams.model_name = selectedModel;
        byModelParams.model_name = selectedModel;
      }

      const [statsData, byModelData, trendChartData, optionsData] = await Promise.all([
        fetchModelUsageStats(statsParams),
        fetchModelUsageByModel(byModelParams),
        fetchModelUsageTrend(trendParams),
        fetchModelUsageByModel(dateRange),
      ]);
      const safeByModelData = Array.isArray(byModelData) ? byModelData : [];
      const safeOptionsData = Array.isArray(optionsData) ? optionsData : [];

      setStats(statsData);
      setUsageData(safeByModelData);
      setTrendData(Array.isArray(trendChartData) ? trendChartData : []);
      setAllConfigTypes(Array.from(new Set(safeOptionsData.map(item => item.config_type))).sort());
      setAllModelNames(Array.from(new Set(safeOptionsData.map(item => item.model_name))).sort());
    } catch (error) {
      console.error("Failed to load usage data:", error);
      setUsageData([]);
      setStats(null);
      setTrendData([]);
      setAllConfigTypes([]);
      setAllModelNames([]);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadData();
  }, [timeRange, customFrom, customTo, selectedConfigType, selectedModel]);

  const handleRefresh = () => {
    loadData();
  };

  const totalTokensIn = stats?.input_tokens ?? 0;
  const totalTokensOut = stats?.output_tokens ?? 0;
  const avgLatency = Math.round(stats?.avg_duration_ms ?? 0);
  const totalCost = stats?.total_cost ?? 0;
  const todayTokens = stats?.today_input_tokens ? (stats.today_input_tokens + (stats.today_output_tokens || 0)) : 0;
  const totalTokens = (stats?.input_tokens || 0) + (stats?.output_tokens || 0);

  // 模型颜色 —— 使用 chart-1~5 CSS 变量循环，替代 hex 硬编码色板
  // TODO: 若模型数量超过 5 个，可扩展 chart 体系或在 globals.css 中新增 --chart-6~15
  const COLORS = [
    "var(--chart-1)", "var(--chart-2)", "var(--chart-3)", "var(--chart-4)", "var(--chart-5)",
    "var(--chart-1)", "var(--chart-2)", "var(--chart-3)", "var(--chart-4)", "var(--chart-5)",
    "var(--chart-1)", "var(--chart-2)", "var(--chart-3)", "var(--chart-4)", "var(--chart-5)",
  ];
  
  const getColorForModel = (model: string) => {
    let hash = 0;
    for (let i = 0; i < model.length; i++) {
      hash = model.charCodeAt(i) + ((hash << 5) - hash);
    }
    return COLORS[Math.abs(hash) % COLORS.length];
  };

  const getStandardCost = (configType: string, inputTokens: number, outputTokens: number) => {
    const rates: Record<string, { inRate: number; outRate: number }> = {
      llm: { inRate: 0.5, outRate: 1.5 },
      embedding: { inRate: 0.1, outRate: 0 },
      rerank: { inRate: 0.5, outRate: 0 },
      ocr: { inRate: 1.5, outRate: 0 },
    };
    const rate = rates[configType] || { inRate: 0.5, outRate: 1.0 };
    return inputTokens / 1_000_000 * rate.inRate + outputTokens / 1_000_000 * rate.outRate;
  };

  const modelRows = usageData.reduce((acc: Record<string, { model: string; calls: number; tokens: number; actualCost: number; standardCost: number }>, item) => {
    if (!acc[item.model_name]) {
      acc[item.model_name] = {
        model: item.model_name,
        calls: 0,
        tokens: 0,
        actualCost: 0,
        standardCost: 0,
      };
    }
    acc[item.model_name].calls += item.total_calls;
    acc[item.model_name].tokens += item.total_tokens;
    acc[item.model_name].actualCost += item.total_cost;
    acc[item.model_name].standardCost += getStandardCost(item.config_type, item.input_tokens, item.output_tokens);
    return acc;
  }, {});

  const modelDistributionRows = Object.values(modelRows).sort((a, b) => b.calls - a.calls);

  const trendChartData = trendData.map((item) => {
    const cacheCreation = item.failed_calls;
    const cacheRead = item.success_calls;
    const cacheHitRate = item.total_calls > 0 ? Number(((item.success_calls / item.total_calls) * 100).toFixed(2)) : 0;
    return {
      ...item,
      input: item.input_tokens,
      output: item.output_tokens,
      cacheCreation,
      cacheRead,
      cacheHitRate,
    };
  });

  const timeRangeOptions = [
    { value: "today", label: "今日" },
    { value: "week", label: "本周" },
    { value: "month", label: "本月" },
    { value: "custom", label: "自定义" },
  ];


  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between gap-3 flex-wrap">
        <div>
          <h3 className="font-semibold text-base">模型分布 & 使用统计</h3>
          <p className="text-xs text-muted-foreground mt-0.5">各模型调用量、Token 用量与响应性能</p>
        </div>
        <div className="flex items-center gap-2 flex-wrap">
          <div className="flex border rounded-md overflow-hidden text-xs">
            {timeRangeOptions.map((opt) => (
              <button key={opt.value} onClick={() => setTimeRange(opt.value)} className={cn("px-3 py-1.5 transition-colors", timeRange === opt.value ? "bg-primary text-primary-foreground" : "bg-background hover:bg-muted text-muted-foreground")}>
                {opt.label}
              </button>
            ))}
          </div>
          {timeRange === "custom" && (
            <div className="flex items-center gap-1 text-xs">
              <Input type="date" className="h-7 text-xs w-32" value={customFrom} onChange={(e) => setCustomFrom(e.target.value)} />
              <span className="text-muted-foreground">—</span>
              <Input type="date" className="h-7 text-xs w-32" value={customTo} onChange={(e) => setCustomTo(e.target.value)} />
            </div>
          )}
          <Button variant="outline" size="sm" onClick={handleRefresh} disabled={loading} className="h-7 px-2">
            <RefreshCw className={cn("w-3.5 h-3.5", loading && "animate-spin")} />
          </Button>
        </div>
      </div>

      <div className="flex items-center gap-2 flex-wrap text-xs">
        <Select value={selectedConfigType || "__all__"} onValueChange={(v) => setSelectedConfigType(v === "__all__" ? "" : v)}>
          <SelectTrigger className="w-28 h-7 text-xs">
            <SelectValue placeholder="配置类型" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="__all__">全部类型</SelectItem>
            {allConfigTypes.map((ct) => (
              <SelectItem key={ct} value={ct}>{ct.toUpperCase()}</SelectItem>
            ))}
          </SelectContent>
        </Select>

        <Select value={selectedModel || "__all__"} onValueChange={(v) => setSelectedModel(v === "__all__" ? "" : v)}>
          <SelectTrigger className="w-28 h-7 text-xs">
            <SelectValue placeholder="模型名称" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="__all__">全部模型</SelectItem>
            {allModelNames.map((mn) => (
              <SelectItem key={mn} value={mn}>{mn}</SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>

      <div className="grid grid-cols-2 sm:grid-cols-5 gap-3">
        <Card className="p-3">
          <div className="text-xs text-muted-foreground mb-1">今日请求</div>
          <div className="text-2xl font-bold text-blue-600">{stats?.today_calls || 0}</div>
          <div className="text-xs text-muted-foreground mt-0.5">RPM: {stats?.rpm || 0}</div>
        </Card>
        <Card className="p-3">
          <div className="text-xs text-muted-foreground mb-1">今日消费</div>
          <div className="text-2xl font-bold text-red-600">¥{(stats?.today_cost || 0).toFixed(4)}</div>
          <div className="text-xs text-muted-foreground mt-0.5">累计 ¥{totalCost.toFixed(2)}</div>
        </Card>
        <Card className="p-3">
          <div className="text-xs text-muted-foreground mb-1">今日 Token</div>
          <div className="text-2xl font-bold text-emerald-600">{formatNum(todayTokens)}</div>
          <div className="text-xs text-muted-foreground mt-0.5">TPM: {formatNum(stats?.tpm || 0)}</div>
        </Card>
        <Card className="p-3">
          <div className="text-xs text-muted-foreground mb-1">累计 Token</div>
          <div className="text-2xl font-bold text-purple-600">{formatNum(totalTokens)}</div>
          <div className="text-xs text-muted-foreground mt-0.5">In/Out {formatNum(totalTokensIn)}/{formatNum(totalTokensOut)}</div>
        </Card>
        <Card className="p-3">
          <div className="text-xs text-muted-foreground mb-1">平均延迟</div>
          <div className="text-2xl font-bold text-orange-600">{avgLatency}ms</div>
          <div className="text-xs text-muted-foreground mt-0.5">P95 响应性能</div>
        </Card>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
        <Card className="flex flex-col">
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-semibold">模型分布</CardTitle>
          </CardHeader>
          <CardContent className="flex-1 py-3">
            {modelDistributionRows.length > 0 ? (
              <div className="flex flex-col sm:flex-row gap-4">
                <div className="w-44 h-44 shrink-0 mx-auto sm:mx-0">
                  <ResponsiveContainer width="100%" height="100%">
                    <PieChart>
                      <Pie
                        data={modelDistributionRows.map((row) => ({ name: row.model, value: row.calls }))}
                        innerRadius={46}
                        outerRadius={86}
                        paddingAngle={2}
                        dataKey="value"
                        animationDuration={800}
                      >
                        {modelDistributionRows.map((row, index) => (
                          <Cell key={`cell-${index}`} fill={getColorForModel(row.model)} className="outline-none" />
                        ))}
                      </Pie>
                      <Tooltip
                        formatter={(value, name) => [formatNum(Number(value ?? 0)), String(name)]}
                        contentStyle={{ borderRadius: "10px", border: "1px solid #e5e7eb", boxShadow: "0 8px 20px rgba(15,23,42,0.08)" }}
                      />
                    </PieChart>
                  </ResponsiveContainer>
                </div>

                <div className="flex-1 min-w-0 overflow-x-auto">
                  <Table>
                    <TableHeader>
                      <TableRow>
                        <TableHead>模型</TableHead>
                        <TableHead className="text-right">请求</TableHead>
                        <TableHead className="text-right">Token</TableHead>
                        <TableHead className="text-right">实际</TableHead>
                        <TableHead className="text-right">标准</TableHead>
                      </TableRow>
                    </TableHeader>
                    <TableBody>
                      {modelDistributionRows.slice(0, 8).map((row) => (
                        <TableRow key={row.model}>
                          <TableCell className="max-w-[200px] truncate" title={row.model}>
                            <div className="flex items-center gap-2">
                              <span className="w-2.5 h-2.5 rounded-full inline-block" style={{ backgroundColor: getColorForModel(row.model) }} />
                              <span className="truncate">{row.model}</span>
                            </div>
                          </TableCell>
                          <TableCell className="text-right tabular-nums">{row.calls.toLocaleString()}</TableCell>
                          <TableCell className="text-right tabular-nums">{formatNum(row.tokens)}</TableCell>
                          <TableCell className="text-right tabular-nums text-emerald-600 font-medium">${row.actualCost.toFixed(4)}</TableCell>
                          <TableCell className="text-right tabular-nums text-slate-400 font-medium">${row.standardCost.toFixed(4)}</TableCell>
                        </TableRow>
                      ))}
                    </TableBody>
                  </Table>
                </div>
              </div>
            ) : (
              <div className="h-48 flex items-center justify-center text-muted-foreground text-sm border-2 border-dashed rounded-xl bg-muted/5">暂无调用分布数据</div>
            )}
          </CardContent>
        </Card>

        <Card className="flex flex-col">
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-semibold">Token 使用趋势</CardTitle>
          </CardHeader>
          <CardContent className="flex-1">
            {trendChartData.length > 0 ? (
              <ResponsiveContainer width="100%" height={250}>
                <LineChart data={trendChartData} margin={{ top: 10, right: 10, left: 6, bottom: 0 }}>
                  <CartesianGrid strokeDasharray="3 3" stroke="#e5e7eb" />
                  <XAxis dataKey="date" tick={{ fontSize: 10 }} />
                  <YAxis tick={{ fontSize: 10 }} tickFormatter={(v) => formatNum(Number(v))} />
                  <YAxis yAxisId="right" orientation="right" domain={[0, 100]} tick={{ fontSize: 10 }} tickFormatter={(v) => `${v}%`} />
                  <Tooltip
                    contentStyle={{ borderRadius: "10px", border: "1px solid #e5e7eb", boxShadow: "0 8px 20px rgba(15,23,42,0.08)" }}
                    formatter={(value, name) => {
                      if (name === "Cache Hit Rate") return [`${Number(value ?? 0)}%`, name];
                      return [formatNum(Number(value ?? 0)), name];
                    }}
                  />
                  <Legend iconType="circle" wrapperStyle={{ fontSize: "12px" }} />
                  <Area type="monotone" dataKey="cacheRead" name="Cache Read" stroke="#06b6d4" fill="#06b6d4" fillOpacity={0.12} strokeWidth={2} />
                  <Line type="monotone" dataKey="input" name="Input" stroke="#3b82f6" strokeWidth={2} dot={{ r: 2 }} />
                  <Line type="monotone" dataKey="output" name="Output" stroke="#22c55e" strokeWidth={2} dot={{ r: 2 }} />
                  <Line type="monotone" dataKey="cacheCreation" name="Cache Creation" stroke="#f59e0b" strokeWidth={2} dot={{ r: 2 }} />
                  <Line type="monotone" dataKey="cacheHitRate" name="Cache Hit Rate" yAxisId="right" stroke="#8b5cf6" strokeWidth={2.5} strokeDasharray="4 4" dot={{ r: 2 }} />
                </LineChart>
              </ResponsiveContainer>
            ) : (
              <div className="h-48 flex items-center justify-center text-sm text-muted-foreground border-2 border-dashed rounded-xl bg-muted/5">暂无趋势统计数据</div>
            )}
          </CardContent>
        </Card>
      </div>

      <Card>
        <CardHeader className="pb-2">
          <CardTitle className="text-sm">调用明细</CardTitle>
        </CardHeader>
        <CardContent className="p-0">
          <div className="overflow-x-auto">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>模型</TableHead>
                  <TableHead>类型</TableHead>
                  <TableHead>调用者</TableHead>
                  <TableHead className="text-right">调用次数</TableHead>
                  <TableHead className="text-right">Token 输入</TableHead>
                  <TableHead className="text-right">Token 输出</TableHead>
                  <TableHead className="text-right">平均延迟</TableHead>
                  <TableHead className="text-right">成功率</TableHead>
                </TableRow>
              </TableHeader>
               <TableBody>
                {usageData.map((item, i) => (
                    <TableRow key={i}>
                      <TableCell className="font-medium max-w-[160px] truncate" title={item.model_name}>
                        {item.model_name}
                      </TableCell>
                      <TableCell>
                        <Badge variant="outline" className="text-xs">
                          {item.config_type.toUpperCase()}
                        </Badge>
                      </TableCell>
                      <TableCell>{item.provider}</TableCell>
                      <TableCell className="text-right">{item.total_calls.toLocaleString()}</TableCell>
                      <TableCell className="text-right">{formatNum(item.input_tokens)}</TableCell>
                      <TableCell className="text-right">{formatNum(item.output_tokens)}</TableCell>
                      <TableCell className="text-right">{Math.round(item.avg_duration_ms)}ms</TableCell>
                      <TableCell className="text-right">{item.success_rate.toFixed(2)}%</TableCell>
                    </TableRow>
                  ))}
               </TableBody>
            </Table>
          </div>
        </CardContent>
      </Card>
    </div>
  );
}

// ============ 角色权限管理 Tab ============
function RolePermissionTab() {
  const [roles, setRoles] = useState<Role[]>([]);
  const [permissions, setPermissions] = useState<Permission[]>([]);
  const [loading, setLoading] = useState(false);
  const [editingRole, setEditingRole] = useState<Role | null>(null);
  const [showDialog, setShowDialog] = useState(false);
  const [newRoleName, setNewRoleName] = useState("");
  const [selectedPermissions, setSelectedPermissions] = useState<number[]>([]);

  useEffect(() => {
    loadData();
  }, []);

  const loadData = async () => {
    setLoading(true);
    try {
      const [rolesData, permsData] = await Promise.all([fetchRoles(), fetchPermissions()]);
      setRoles(rolesData);
      setPermissions(permsData);
    } catch (error) {
      console.error("加载数据失败:", error);
      toast.error("加载数据失败");
    } finally {
      setLoading(false);
    }
  };

  const handleEditRole = async (role: Role) => {
    setEditingRole(role);
    setNewRoleName(role.name);
    try {
      const perms = await fetchRolePermissions(role.id);
      setSelectedPermissions(perms.map((p) => p.id) as number[]);
    } catch (error) {
      console.error("加载权限失败:", error);
    }
    setShowDialog(true);
  };

  const handleSaveRole = async () => {
    if (!editingRole || !newRoleName.trim()) {
      toast.error("角色名称不能为空");
      return;
    }

    try {
      await updateRole(editingRole.id, { label: newRoleName, description: editingRole.description });
      await updateRolePermissions(editingRole.id, selectedPermissions);
      toast.success("角色已更新");
      setShowDialog(false);
      loadData();
    } catch (error) {
      console.error("保存失败:", error);
      toast.error("保存失败");
    }
  };

  const handleDeleteRole = async (roleId: number) => {
    if (confirm("确定要删除此角色吗？")) {
      try {
        await deleteRole(roleId);
        toast.success("角色已删除");
        loadData();
      } catch (error) {
        console.error("删除失败:", error);
        toast.error("删除失败");
      }
    }
  };

  return (
    <Card>
      <CardHeader>
        <div className="flex items-center justify-between">
          <div>
            <CardTitle>角色权限管理</CardTitle>
            <CardDescription>管理系统角色和权限分配</CardDescription>
          </div>
          <Button onClick={() => { setEditingRole(null); setNewRoleName(""); setSelectedPermissions([]); setShowDialog(true); }}>
            <Plus className="w-4 h-4 mr-2" />
            新增角色
          </Button>
        </div>
      </CardHeader>
      <CardContent>
        {loading ? (
          <div className="text-center py-8 text-muted-foreground">加载中...</div>
        ) : (
          <div className="space-y-4">
            {roles.map((role) => (
              <div key={role.id} className="border rounded-lg p-4 flex items-center justify-between">
                <div>
                  <h4 className="font-medium">{role.name}</h4>
                  <p className="text-sm text-muted-foreground">ID: {role.id}</p>
                </div>
                <div className="flex gap-2">
                  <Button variant="outline" size="sm" onClick={() => handleEditRole(role)}>
                    <Edit className="w-4 h-4" />
                  </Button>
                  <Button variant="outline" size="sm" onClick={() => handleDeleteRole(role.id)}>
                    <Trash2 className="w-4 h-4" />
                  </Button>
                </div>
              </div>
            ))}
          </div>
        )}
      </CardContent>

      <Dialog open={showDialog} onOpenChange={setShowDialog}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{editingRole ? "编辑角色" : "新增角色"}</DialogTitle>
          </DialogHeader>
          <div className="space-y-4">
            <div className="grid gap-2">
              <Label htmlFor="role-name">角色名称</Label>
              <Input id="role-name" value={newRoleName} onChange={(e) => setNewRoleName(e.target.value)} />
            </div>
            <div className="grid gap-2">
              <Label>权限</Label>
              <ScrollArea className="h-48 border rounded-md p-4">
                <div className="space-y-2">
                   {permissions.map((perm) => (
                     <div key={perm.id} className="flex items-center space-x-2">
                       <Checkbox id={String(perm.id)} checked={selectedPermissions.includes(perm.id)} onCheckedChange={(checked) => {
                         if (checked) {
                           setSelectedPermissions([...selectedPermissions, perm.id]);
                         } else {
                           setSelectedPermissions(selectedPermissions.filter((p) => p !== perm.id));
                         }
                       }} />
                       <Label htmlFor={String(perm.id)} className="cursor-pointer">
                         {perm.label}
                       </Label>
                     </div>
                   ))}
                </div>
              </ScrollArea>
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setShowDialog(false)}>
              取消
            </Button>
            <Button onClick={handleSaveRole}>保存</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </Card>
  );
}

// ============ 公告管理 Tab ============
function AnnouncementTab() {
  const [announcements, setAnnouncements] = useState<Announcement[]>([]);
  const [loading, setLoading] = useState(false);
  const [showDialog, setShowDialog] = useState(false);
  const [editingAnnouncement, setEditingAnnouncement] = useState<Announcement | null>(null);
  const [title, setTitle] = useState("");
  const [content, setContent] = useState("");

  useEffect(() => {
    loadAnnouncements();
  }, []);

  const loadAnnouncements = async () => {
    setLoading(true);
    try {
      const data = await fetchAnnouncements();
      setAnnouncements(data);
    } catch (error) {
      console.error("加载公告失败:", error);
      toast.error("加载公告失败");
    } finally {
      setLoading(false);
    }
  };

  const handleSave = async () => {
    if (!title.trim() || !content.trim()) {
      toast.error("标题和内容不能为空");
      return;
    }

    try {
      if (editingAnnouncement) {
        await updateAnnouncement(editingAnnouncement.id, { title, content });
        toast.success("公告已更新");
      } else {
        await createAnnouncement({ title, content });
        toast.success("公告已创建");
      }
      setShowDialog(false);
      setTitle("");
      setContent("");
      loadAnnouncements();
    } catch (error) {
      console.error("保存失败:", error);
      toast.error("保存失败");
    }
  };

  const handleDelete = async (id: number) => {
    if (confirm("确定要删除此公告吗？")) {
      try {
        await deleteAnnouncement(id);
        toast.success("公告已删除");
        loadAnnouncements();
      } catch (error) {
        console.error("删除失败:", error);
        toast.error("删除失败");
      }
    }
  };

  return (
    <Card>
      <CardHeader>
        <div className="flex items-center justify-between">
          <div>
            <CardTitle>公告管理</CardTitle>
            <CardDescription>发布系统公告和通知</CardDescription>
          </div>
          <Button onClick={() => { setEditingAnnouncement(null); setTitle(""); setContent(""); setShowDialog(true); }}>
            <Plus className="w-4 h-4 mr-2" />
            新增公告
          </Button>
        </div>
      </CardHeader>
      <CardContent>
        {loading ? (
          <div className="text-center py-8 text-muted-foreground">加载中...</div>
        ) : (
          <div className="space-y-4">
            {announcements.map((ann) => (
              <div key={ann.id} className="border rounded-lg p-4">
                <div className="flex items-start justify-between mb-2">
                  <h4 className="font-medium">{ann.title}</h4>
                  <div className="flex gap-2">
                    <Button variant="outline" size="sm" onClick={() => { setEditingAnnouncement(ann); setTitle(ann.title); setContent(ann.content); setShowDialog(true); }}>
                      <Edit className="w-4 h-4" />
                    </Button>
                    <Button variant="outline" size="sm" onClick={() => handleDelete(ann.id)}>
                      <Trash2 className="w-4 h-4" />
                    </Button>
                  </div>
                </div>
                <p className="text-sm text-muted-foreground">{ann.content}</p>
                <p className="text-xs text-muted-foreground mt-2">{format(new Date(ann.created_at), "yyyy-MM-dd HH:mm")}</p>
              </div>
            ))}
          </div>
        )}
      </CardContent>

      <Dialog open={showDialog} onOpenChange={setShowDialog}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{editingAnnouncement ? "编辑公告" : "新增公告"}</DialogTitle>
          </DialogHeader>
          <div className="space-y-4">
            <div className="grid gap-2">
              <Label htmlFor="ann-title">标题</Label>
              <Input id="ann-title" value={title} onChange={(e) => setTitle(e.target.value)} />
            </div>
            <div className="grid gap-2">
              <Label htmlFor="ann-content">内容</Label>
              <Textarea id="ann-content" value={content} onChange={(e) => setContent(e.target.value)} rows={6} />
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setShowDialog(false)}>
              取消
            </Button>
            <Button onClick={handleSave}>保存</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </Card>
  );
}

// ============ 系统维护 Tab ============
function SystemMaintenanceTab() {
  const [isMaintenanceMode, setIsMaintenanceMode] = useState(false);
  const [maintenanceMessage, setMaintenanceMessage] = useState("");

  return (
    <Card>
      <CardHeader>
        <CardTitle>系统维护</CardTitle>
        <CardDescription>管理系统维护模式和清理操作</CardDescription>
      </CardHeader>
      <CardContent className="space-y-6">
        <div className="border rounded-lg p-4">
          <div className="flex items-center justify-between mb-4">
            <div>
              <h4 className="font-medium">维护模式</h4>
              <p className="text-sm text-muted-foreground">启用后，只有管理员可以访问系统</p>
            </div>
            <Switch checked={isMaintenanceMode} onCheckedChange={setIsMaintenanceMode} />
          </div>
          {isMaintenanceMode && (
            <div className="grid gap-2">
              <Label htmlFor="maintenance-msg">维护消息</Label>
              <Textarea id="maintenance-msg" value={maintenanceMessage} onChange={(e) => setMaintenanceMessage(e.target.value)} placeholder="系统正在维护中，请稍后访问..." rows={3} />
            </div>
          )}
        </div>

        <div className="border rounded-lg p-4">
          <h4 className="font-medium mb-4">数据清理</h4>
          <div className="space-y-2">
            <Button variant="outline" className="w-full justify-start">
              <Trash2 className="w-4 h-4 mr-2" />
              清理过期的临时文件
            </Button>
            <Button variant="outline" className="w-full justify-start">
              <Trash2 className="w-4 h-4 mr-2" />
              清理过期的日志记录
            </Button>
            <Button variant="outline" className="w-full justify-start">
              <Trash2 className="w-4 h-4 mr-2" />
              清理缓存数据
            </Button>
          </div>
        </div>
      </CardContent>
    </Card>
  );
}



// ============ 档案分类管理 Tab ============
function ArchiveClassificationTab() {
  const [categories, setCategories] = useState<DocumentCategory[]>([]);
  const [loading, setLoading] = useState(false);
  const [showCategoryModal, setShowCategoryModal] = useState(false);
  const [selectedCategory, setSelectedCategory] = useState<DocumentCategory | null>(null);

  const loadCategories = useCallback(async () => {
    setLoading(true);
    try {
      const data = await fetchDocumentCategories();
      setCategories(data);
      
      // 如果当前有选中的大类，同步更新其数据以刷新子类列表等
      if (selectedCategory) {
        const updated = data.find(c => c.id === selectedCategory.id);
        if (updated) setSelectedCategory(updated);
      }
    } catch {
      toast.error("加载分类失败");
    } finally {
      setLoading(false);
    }
  }, [selectedCategory]);

  useEffect(() => {
    loadCategories();
  }, [loadCategories]);

  const handleEditCategory = (category: DocumentCategory) => {
    setSelectedCategory(category);
    setShowCategoryModal(true);
  };

  const handleAddCategory = () => {
    setSelectedCategory(null);
    setShowCategoryModal(true);
  };

  const handleDeleteCategory = async (id: number) => {
    if (!confirm("确定要删除此分类及其下所有子分类吗？")) return;
    try {
      await deleteCategory(id);
      toast.success("分类已删除");
      loadCategories();
    } catch {
      toast.error("删除失败");
    }
  };

  return (
    <div className="space-y-4">
      <div className="flex justify-between items-center">
        <h3 className="text-lg font-medium">档案大类</h3>
        <Button size="sm" onClick={handleAddCategory}>
          <Plus className="w-4 h-4 mr-2" />
          新增大类
        </Button>
      </div>

      <div className="border rounded-md">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead className="w-[100px]">代码</TableHead>
              <TableHead>名称</TableHead>
              <TableHead>排序</TableHead>
              <TableHead>子类数量</TableHead>
              <TableHead className="text-right">操作</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {loading ? (
              <TableRow>
                <TableCell colSpan={5} className="text-center py-8 text-muted-foreground">加载中...</TableCell>
              </TableRow>
            ) : categories.length === 0 ? (
              <TableRow>
                <TableCell colSpan={5} className="text-center py-8 text-muted-foreground">暂无分类数据</TableCell>
              </TableRow>
            ) : (
              categories.map((cat) => (
                <TableRow key={cat.id}>
                  <TableCell className="font-mono">{cat.code}</TableCell>
                  <TableCell className="font-medium">{cat.name}</TableCell>
                  <TableCell>{cat.sort_order}</TableCell>
                  <TableCell>{cat.sub_categories?.length || 0}</TableCell>
                  <TableCell className="text-right">
                    <div className="flex justify-end gap-2">
                      <Button variant="ghost" size="sm" onClick={() => handleEditCategory(cat)}>
                        <Edit className="w-4 h-4" />
                      </Button>
                      <Button variant="ghost" size="sm" onClick={() => handleDeleteCategory(cat.id)}>
                        <Trash2 className="w-4 h-4 text-destructive" />
                      </Button>
                    </div>
                  </TableCell>
                </TableRow>
              ))
            )}
          </TableBody>
        </Table>
      </div>

      <CategoryModal
        open={showCategoryModal}
        onOpenChange={setShowCategoryModal}
        category={selectedCategory}
        onSaved={loadCategories}
      />
    </div>
  );
}

function CategoryModal({
  open,
  onOpenChange,
  category,
  onSaved
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  category: DocumentCategory | null;
  onSaved: () => void;
}) {
  const [name, setName] = useState("");
  const [code, setCode] = useState("");
  const [description, setDescription] = useState("");
  const [sortOrder, setSortOrder] = useState(0);
  const [saving, setSaving] = useState(false);
  
  const [showSubModal, setShowSubModal] = useState(false);
  const [selectedSubCategory, setSelectedSubCategory] = useState<DocumentSubCategory | null>(null);

  useEffect(() => {
    if (category) {
      setName(category.name);
      setCode(category.code);
      setDescription(category.description || "");
      setSortOrder(category.sort_order || 0);
      
      // 如果子分类正在编辑，也同步更新它以显示最新数据
      if (selectedSubCategory && category.sub_categories) {
        const updated = category.sub_categories.find(s => s.id === selectedSubCategory.id);
        if (updated) setSelectedSubCategory(updated);
      }
    } else {
      setName("");
      setCode("");
      setDescription("");
      setSortOrder(0);
    }
  }, [category, open, selectedSubCategory]);

  const handleSave = async () => {
    if (!name.trim() || !code.trim()) {
      toast.error("名称和代码不能为空");
      return;
    }
    setSaving(true);
    try {
      if (category) {
        await updateCategoryCode(category.id, { name, code, description, sort_order: sortOrder });
        toast.success("更新成功");
      } else {
        await createCategoryCode({ name, code, description, sort_order: sortOrder });
        toast.success("创建成功");
      }
      onSaved();
      if (!category) onOpenChange(false);
    } catch {
      toast.error("保存失败");
    } finally {
      setSaving(false);
    }
  };

  const handleEditSub = (sub: DocumentSubCategory) => {
    setSelectedSubCategory(sub);
    setShowSubModal(true);
  };

  const handleAddSub = () => {
    setSelectedSubCategory(null);
    setShowSubModal(true);
  };

  const handleDeleteSub = async (subId: number) => {
    if (!confirm("确定要删除此子分类吗？")) return;
    try {
      await deleteSubCategory(subId);
      toast.success("子分类已删除");
      onSaved();
    } catch {
      toast.error("删除失败");
    }
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-4xl max-h-[90vh] flex flex-col p-0">
        <DialogHeader className="p-6 pb-0">
          <DialogTitle>{category ? `编辑分类: ${category.name}` : "新增大类"}</DialogTitle>
        </DialogHeader>
        
        <div className="flex-1 overflow-y-auto p-6 space-y-6">
          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-2">
              <Label>分类名称</Label>
              <Input value={name} onChange={(e) => setName(e.target.value)} placeholder="如: 行政档案" />
            </div>
            <div className="space-y-2">
              <Label>分类代码</Label>
              <Input value={code} onChange={(e) => setCode(e.target.value)} placeholder="如: ADMIN" />
            </div>
            <div className="space-y-2">
              <Label>排序权重</Label>
              <Input type="number" value={sortOrder} onChange={(e) => setSortOrder(parseInt(e.target.value) || 0)} />
            </div>
            <div className="space-y-2 col-span-2">
              <Label>描述</Label>
              <Textarea value={description} onChange={(e) => setDescription(e.target.value)} placeholder="分类描述信息" rows={2} />
            </div>
          </div>

          <div className="flex justify-end">
            <Button onClick={handleSave} disabled={saving}>
              {saving ? "保存中..." : "保存基本信息"}
            </Button>
          </div>

          {category && (
            <div className="space-y-4 border-t pt-6">
              <div className="flex justify-between items-center">
                <h4 className="font-semibold text-sm">子分类列表</h4>
                <Button size="sm" variant="outline" onClick={handleAddSub}>
                  <Plus className="w-3 h-3 mr-1" />
                  新增子类
                </Button>
              </div>

              <div className="border rounded-md">
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>代码</TableHead>
                      <TableHead>名称</TableHead>
                      <TableHead>排序</TableHead>
                      <TableHead className="text-right">操作</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {!category.sub_categories || category.sub_categories.length === 0 ? (
                      <TableRow>
                        <TableCell colSpan={4} className="text-center py-4 text-muted-foreground text-sm">暂无子类</TableCell>
                      </TableRow>
                    ) : (
                      category.sub_categories.map((sub) => (
                        <TableRow key={sub.id}>
                          <TableCell className="font-mono text-xs">{sub.code}</TableCell>
                          <TableCell className="text-sm font-medium">{sub.name}</TableCell>
                          <TableCell className="text-sm">{sub.sort_order || 0}</TableCell>
                          <TableCell className="text-right">
                            <div className="flex justify-end gap-1">
                              <Button variant="ghost" size="sm" onClick={() => handleEditSub(sub)}>
                                <Edit className="w-3 h-3" />
                              </Button>
                              <Button variant="ghost" size="sm" onClick={() => handleDeleteSub(sub.id)}>
                                <Trash2 className="w-3 h-3 text-destructive" />
                              </Button>
                            </div>
                          </TableCell>
                        </TableRow>
                      ))
                    )}
                  </TableBody>
                </Table>
              </div>
            </div>
          )}
        </div>

        {category && (
          <SubCategoryModal
            open={showSubModal}
            onOpenChange={setShowSubModal}
            categoryId={category.id}
            subCategory={selectedSubCategory}
            onSaved={onSaved}
          />
        )}
      </DialogContent>
    </Dialog>
  );
}

function FieldModal({
  open,
  onOpenChange,
  subCategoryId,
  field,
  onSaved
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  subCategoryId: number;
  field: ArchiveFieldDefinition | null;
  onSaved: () => void;
}) {
  const [formData, setFormData] = useState({
    field_name: "",
    field_label: "",
    field_type: "text" as ArchiveFieldDefinition["field_type"],
    required: false,
    options: "",
    placeholder: "",
    help_text: "",
    default_value: "",
    condition_config: "",
  });
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    if (field) {
      setFormData({
        field_name: field.field_name,
        field_label: field.field_label,
        field_type: field.field_type,
        required: field.required,
        options: field.options || "",
        placeholder: field.placeholder || "",
        help_text: field.help_text || "",
        default_value: field.default_value || "",
        condition_config: field.condition_config ? JSON.stringify(field.condition_config, null, 2) : "",
      });
    } else {
      setFormData({
        field_name: "",
        field_label: "",
        field_type: "text",
        required: false,
        options: "",
        placeholder: "",
        help_text: "",
        default_value: "",
        condition_config: "",
      });
    }
  }, [field, open]);

  const handleSave = async () => {
    if (!formData.field_name || !formData.field_label) {
      toast.error("字段名和标签不能为空");
      return;
    }

    let conditionConfigObj = null;
    if (formData.condition_config.trim()) {
      try {
        conditionConfigObj = JSON.parse(formData.condition_config);
      } catch {
        toast.error("条件配置格式错误，必须是有效的 JSON");
        return;
      }
    }

    setSaving(true);
    try {
      const payload = {
        ...formData,
        condition_config: conditionConfigObj,
      };
      if (field) {
        await updateFieldDefinition(field.id, payload);
      } else {
        await createFieldDefinition({ ...payload, sub_category_id: subCategoryId });
      }
      toast.success("保存成功");
      onSaved();
      onOpenChange(false);
    } catch {
      toast.error("保存失败");
    } finally {
      setSaving(false);
    }
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-2xl max-h-[90vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle>{field ? `编辑字段: ${field.field_label}` : "新增字段"}</DialogTitle>
        </DialogHeader>

        <div className="grid gap-4 py-4">
          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-2">
              <Label>字段标签 (显示名称)</Label>
              <Input value={formData.field_label} onChange={(e) => setFormData({ ...formData, field_label: e.target.value })} placeholder="如: 合同金额" />
            </div>
            <div className="space-y-2">
              <Label>字段名称 (英文ID)</Label>
              <Input value={formData.field_name} onChange={(e) => setFormData({ ...formData, field_name: e.target.value })} placeholder="如: contract_amount" />
            </div>
          </div>

          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-2">
              <Label>字段类型</Label>
              <Select value={formData.field_type} onValueChange={(v) => setFormData({ ...formData, field_type: v as ArchiveFieldDefinition["field_type"] })}>
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="text">文本</SelectItem>
                  <SelectItem value="number">数字</SelectItem>
                  <SelectItem value="date">日期</SelectItem>
                  <SelectItem value="select">下拉选择</SelectItem>
                  <SelectItem value="textarea">多行文本</SelectItem>
                  <SelectItem value="checkbox">复选框</SelectItem>
                </SelectContent>
              </Select>
            </div>
            <div className="flex items-center space-x-2 pt-8">
              <Switch checked={formData.required} onCheckedChange={(v) => setFormData({ ...formData, required: v })} />
              <Label>设为必填</Label>
            </div>
          </div>

          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-2">
              <Label>默认值</Label>
              <Input value={formData.default_value} onChange={(e) => setFormData({ ...formData, default_value: e.target.value })} placeholder="输入默认值" />
            </div>
            {(formData.field_type === "select" || formData.field_type === "multiselect") && (
              <div className="space-y-2">
                <Label>选项列表 (英文逗号分隔)</Label>
                <Input value={formData.options} onChange={(e) => setFormData({ ...formData, options: e.target.value })} placeholder="选项1,选项2,选项3" />
              </div>
            )}
          </div>

          <div className="space-y-2">
            <Label>提示文字 (Placeholder)</Label>
            <Input value={formData.placeholder} onChange={(e) => setFormData({ ...formData, placeholder: e.target.value })} />
          </div>

          <div className="space-y-2">
            <Label>帮助信息</Label>
            <Textarea value={formData.help_text} onChange={(e) => setFormData({ ...formData, help_text: e.target.value })} />
          </div>

          <div className="space-y-2">
            <Label>条件显示规则 (JSON)</Label>
            <Textarea 
              value={formData.condition_config} 
              onChange={(e) => setFormData({ ...formData, condition_config: e.target.value })} 
              placeholder='{"field_name": "category", "operator": "equals", "value": "A"}'
              rows={4}
              className="font-mono text-xs"
            />
          </div>
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>取消</Button>
          <Button onClick={handleSave} disabled={saving}>
            {saving ? "保存中..." : "保存字段"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

function SubCategoryModal({
  open,
  onOpenChange,
  categoryId,
  subCategory,
  onSaved
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  categoryId: number;
  subCategory: DocumentSubCategory | null;
  onSaved: () => void;
}) {
  const [name, setName] = useState("");
  const [code, setCode] = useState("");
  const [description, setDescription] = useState("");
  const [sortOrder, setSortOrder] = useState(0);
  const [saving, setSaving] = useState(false);
  
  const [fields, setFields] = useState<ArchiveFieldDefinition[]>([]);
  const [loadingFields, setLoadingFields] = useState(false);
  const [showFieldModal, setShowFieldModal] = useState(false);
  const [selectedField, setSelectedField] = useState<ArchiveFieldDefinition | null>(null);

  const loadFields = useCallback(async () => {
    if (!subCategory) return;
    setLoadingFields(true);
    try {
      const data = await fetchFieldsBySubCategory(subCategory.id);
      const allFields = [...data.ungrouped, ...data.groups.flatMap(g => g.fields)];
      setFields(allFields);
    } catch {
      toast.error("加载字段失败");
    } finally {
      setLoadingFields(false);
    }
  }, [subCategory]);

  useEffect(() => {
    if (subCategory) {
      setName(subCategory.name);
      setCode(subCategory.code);
      setDescription(subCategory.description || "");
      setSortOrder(subCategory.sort_order || 0);
      loadFields();
    } else {
      setName("");
      setCode("");
      setDescription("");
      setSortOrder(0);
      setFields([]);
    }
  }, [subCategory, open, loadFields]);

  const handleSave = async () => {
    if (!name.trim() || !code.trim()) {
      toast.error("名称和代码不能为空");
      return;
    }
    setSaving(true);
    try {
      if (subCategory) {
        await updateSubCategoryCode(subCategory.id, { code, name, description, sort_order: sortOrder });
        toast.success("更新成功");
      } else {
        await createSubCategory({ category_id: categoryId, name, code, description, sort_order: sortOrder });
        toast.success("创建成功");
      }
      onSaved();
      if (!subCategory) onOpenChange(false);
    } catch {
      toast.error("保存失败");
    } finally {
      setSaving(false);
    }
  };

  const handleEditField = (field: ArchiveFieldDefinition) => {
    setSelectedField(field);
    setShowFieldModal(true);
  };

  const handleAddField = () => {
    setSelectedField(null);
    setShowFieldModal(true);
  };

  const handleDeleteField = async (fieldId: number) => {
    if (!confirm("确定要删除此字段吗？")) return;
    try {
      await deleteFieldDefinition(fieldId);
      toast.success("字段已删除");
      loadFields();
    } catch {
      toast.error("删除失败");
    }
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-4xl max-h-[85vh] flex flex-col p-0">
        <DialogHeader className="p-6 pb-0">
          <DialogTitle>{subCategory ? `编辑子类: ${subCategory.name}` : "新增子类"}</DialogTitle>
        </DialogHeader>

        <div className="flex-1 overflow-y-auto p-6 space-y-6">
          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-2">
              <Label>子类名称</Label>
              <Input value={name} onChange={(e) => setName(e.target.value)} placeholder="如: 劳动合同" />
            </div>
            <div className="space-y-2">
              <Label>子类代码</Label>
              <Input value={code} onChange={(e) => setCode(e.target.value)} placeholder="如: CONTRACT" />
            </div>
            <div className="space-y-2">
              <Label>排序权重</Label>
              <Input type="number" value={sortOrder} onChange={(e) => setSortOrder(parseInt(e.target.value) || 0)} />
            </div>
            <div className="space-y-2 col-span-2">
              <Label>描述</Label>
              <Textarea value={description} onChange={(e) => setDescription(e.target.value)} placeholder="子类描述信息" rows={2} />
            </div>
          </div>

          <div className="flex justify-end">
            <Button onClick={handleSave} disabled={saving}>
              {saving ? "保存中..." : "保存基本信息"}
            </Button>
          </div>

          {subCategory && (
            <div className="space-y-4 border-t pt-6">
              <div className="flex justify-between items-center">
                <h4 className="font-semibold text-sm">业务字段 (Metadata)</h4>
                <Button size="sm" variant="outline" onClick={handleAddField}>
                  <Plus className="w-3 h-3 mr-1" />
                  新增字段
                </Button>
              </div>

              <div className="border rounded-md">
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>标签</TableHead>
                      <TableHead>名称(Key)</TableHead>
                      <TableHead>类型</TableHead>
                      <TableHead>必填</TableHead>
                      <TableHead className="text-right">操作</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {loadingFields ? (
                      <TableRow>
                        <TableCell colSpan={5} className="text-center py-4 text-muted-foreground">加载中...</TableCell>
                      </TableRow>
                    ) : fields.length === 0 ? (
                      <TableRow>
                        <TableCell colSpan={5} className="text-center py-4 text-muted-foreground text-sm">暂无字段</TableCell>
                      </TableRow>
                    ) : (
                      fields.map((field) => (
                        <TableRow key={field.id}>
                          <TableCell className="text-sm font-medium">{field.field_label}</TableCell>
                          <TableCell className="text-xs font-mono">{field.field_name}</TableCell>
                          <TableCell className="text-xs uppercase text-muted-foreground">{field.field_type}</TableCell>
                          <TableCell>{field.required ? <Badge variant="secondary">必填</Badge> : "-"}</TableCell>
                          <TableCell className="text-right">
                            <div className="flex justify-end gap-1">
                              <Button variant="ghost" size="sm" onClick={() => handleEditField(field)}>
                                <Edit className="w-3 h-3" />
                              </Button>
                              <Button variant="ghost" size="sm" onClick={() => handleDeleteField(field.id)}>
                                <Trash2 className="w-3 h-3 text-destructive" />
                              </Button>
                            </div>
                          </TableCell>
                        </TableRow>
                      ))
                    )}
                  </TableBody>
                </Table>
              </div>
            </div>
          )}
        </div>

        {subCategory && (
          <FieldModal
            open={showFieldModal}
            onOpenChange={setShowFieldModal}
            subCategoryId={subCategory.id}
            field={selectedField}
            onSaved={loadFields}
          />
        )}
      </DialogContent>
    </Dialog>
  );
}

function AdvancedOptions() {
  return (
    <div className="space-y-6">
      <Card>
        <CardHeader>
          <CardTitle className="text-base flex items-center gap-2">
            <ShieldCheck className="w-4 h-4 text-blue-500" />
            字段分组配置 (待开发)
          </CardTitle>
          <CardDescription>配置业务字段的逻辑分组与表单展示顺序</CardDescription>
        </CardHeader>
        <CardContent className="h-32 flex items-center justify-center border-2 border-dashed rounded-lg bg-accent/20">
          <p className="text-muted-foreground text-sm">该功能正在内测中，敬请期待</p>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle className="text-base flex items-center gap-2">
            <Search className="w-4 h-4 text-orange-500" />
            条件显示规则 (待开发)
          </CardTitle>
          <CardDescription>配置基于字段值的联动显示或必填逻辑</CardDescription>
        </CardHeader>
        <CardContent className="h-32 flex items-center justify-center border-2 border-dashed rounded-lg bg-accent/20">
          <p className="text-muted-foreground text-sm">需要配合高级字段引擎使用</p>
        </CardContent>
      </Card>
    </div>
  );
}

// ============ SMTP 配置 Tab ============
// 货币格式化函数（CNY）
// ============ 模型使用统计 Tab ============
// ============ 角色权限管理 Tab ============
// ============ 公告管理 Tab ============
// ============ 系统维护 Tab ============
const getTypeIcon = (type: string) => {
  switch (type) {
    case "s3":
      return <Cloud className="w-3.5 h-3.5" />;
    case "webdav":
      return <Server className="w-3.5 h-3.5" />;
    default:
      return <HardDrive className="w-3.5 h-3.5" />;
  }
};

const getTypeLabel = (type: string) => {
  switch (type) {
    case "s3":
      return "S3 兼容";
    case "webdav":
      return "WebDAV";
    default:
      return "本地存储";
  }
};

function StorageStatusIndicator({ config, res }: { config: StorageConfig; res?: { success: boolean; testing?: boolean } }) {
  const isEnabled = config.enabled;
  const isOnline = res?.success === true;
  const isTesting = res?.testing === true;

  if (!isEnabled) return <Circle className="w-2.5 h-2.5 text-gray-400 fill-gray-400" />;
  if (isTesting) return <Circle className="w-2.5 h-2.5 text-yellow-500 fill-yellow-500 animate-pulse" />;
  if (isOnline) return <Circle className="w-2.5 h-2.5 text-green-500 fill-green-500" />;
  return <Circle className="w-2.5 h-2.5 text-red-500 fill-red-500" />;
}

function StorageTab() {
  const [configs, setConfigs] = useState<StorageConfig[]>([]);
  const [modules, setModules] = useState<StorageModuleConfig[]>([]);
  const [rules, setRules] = useState<StorageRule[]>([]);
  const [testResults, setTestResults] = useState<Record<number, { success: boolean; latency?: number; message?: string; testing?: boolean }>>({});
  const [selectedModule, setSelectedModule] = useState<StorageModuleConfig | null>(null);
  
   const [showConfigDialog, setShowConfigDialog] = useState(false);
   const [editingConfig, setEditingConfig] = useState<StorageConfig | null>(null);
   const [configForm, setConfigForm] = useState({
     name: "",
     type: "s3" as "s3" | "webdav",
     enabled: true,
     s3: { endpoint: "", bucket: "", region: "", access_key: "", secret_key: "", provider: "aliyun" },
     webdav: { url: "", username: "", password: "", directory: "/" }
   });

  const [ruleForm, setRuleForm] = useState({
    storage_id: 0,
    base_path: "",
    enabled: true
  });

  const [restoreConfirm, setRestoreConfirm] = useState(false);
  const [deleteConfigConfirmOpen, setDeleteConfigConfirmOpen] = useState(false);
  const [deleteRuleConfirmId, setDeleteRuleConfirmId] = useState<number | null>(null);
  const [deleteTargetConfigId, setDeleteTargetId] = useState<number | null>(null);
  const [savingRule, setSavingRule] = useState(false);
  const [webdavDirs, setWebdavDirs] = useState<string[]>([]);
  const [fetchingDirs, setFetchingDirs] = useState(false);
  const [showWebdavPassword, setShowWebdavPassword] = useState(false);

   const fetchWebdavDirs = async (type: string, config: Record<string, unknown>) => {
     if (type !== "webdav") return;
     setFetchingDirs(true);
     console.log("[WebDAV] fetchWebdavDirs called with config:", JSON.stringify(config));
     try {
       const webdavConfig = {
         webdav_url: (config.webdav_url || config.url || config.webdavURL || "") as string,
         webdav_username: (config.webdav_username || config.username || config.webdavUsername || "") as string,
         webdav_password: (config.webdav_password || config.password || config.webdavPassword || "") as string,
         directory: (config.directory || "/") as string
       };
       console.log("[WebDAV] Transformed webdavConfig:", JSON.stringify(webdavConfig));
       console.log("[WebDAV] Calling listStorageDirectoriesEnhanced with:", { type, config: webdavConfig });
       const res = await listStorageDirectoriesEnhanced({ type, config: webdavConfig });
       console.log("[WebDAV] listStorageDirectoriesEnhanced result:", res);
       setWebdavDirs(res.directories || []);
     } catch (err) {
       const error = err instanceof Error ? err : new Error(String(err));
       console.error("[WebDAV] fetchWebdavDirs error:", error);
       toast.error("获取 WebDAV 目录失败: " + (error?.message || ""));
     } finally {
       setFetchingDirs(false);
     }
   };

  const handleTest = async (config: StorageConfig, silent = false) => {
    setTestResults(prev => ({ ...prev, [config.id]: { ...prev[config.id], testing: true } }));
    try {
      const result = await testStorageConnection({
        type: config.type,
        config: config.config
      });
      setTestResults(prev => ({ 
        ...prev, 
        [config.id]: { 
          testing: false, 
          success: result.success, 
          latency: result.latency_ms, 
          message: result.message 
        } 
      }));
      if (!silent) {
        if (result.success) toast.success(`连接成功 (${result.latency_ms}ms)`);
        else toast.error(`连接失败: ${result.message}`);
      }
    } catch {
      setTestResults(prev => ({ ...prev, [config.id]: { testing: false, success: false } }));
      if (!silent) toast.error("测试连接失败");
    }
  };

  const loadData = useCallback(async () => {
    try {
      const [configsData, modulesData, rulesData] = await Promise.all([
        listStorageConfigs(),
        listStorageModules(),
        listStorageRulesEnhanced()
      ]);
      setConfigs(configsData);
      
      const displayModules = modulesData.length > 0 ? modulesData : ([
        { module_code: "employee", module_name: "员工管理" },
        { module_code: "provident", module_name: "社保管理" },
        { module_code: "dormitory", module_name: "宿舍管理" },
        { module_code: "daily", module_name: "日常事务" },
        { module_code: "archives", module_name: "档案管理" }
      ] as StorageModuleConfig[]);
      setModules(displayModules);
      setRules(rulesData);
      
      if (displayModules.length > 0 && !selectedModule) {
        setSelectedModule(displayModules[0]);
      }

      configsData.forEach(conf => handleTest(conf, true));
    } catch {
      toast.error("加载存储数据失败");
    }
  }, [selectedModule]);

  useEffect(() => {
    loadData();
  }, [loadData]);

  const existingRule = selectedModule ? rules.find(r => r.module_code === selectedModule.module_code) : null;

  useEffect(() => {
    if (selectedModule) {
      const rule = rules.find(r => r.module_code === selectedModule.module_code);
      if (rule) {
        const configObj = configs.find(c => c.id === rule.storage_id);
        const resolvedPath = rule.base_path || `/upload/${selectedModule.module_code}/{YYYY}`;
        setRuleForm({
          storage_id: rule.storage_id,
          base_path: resolvedPath,
          enabled: rule.enabled
        });
      } else {
        const defaultLocal = configs.find(c => c.is_default && c.type === "local") || configs.find(c => c.type === "local");
        const basePath = `/upload/${selectedModule.module_code}/{YYYY}`;
        setRuleForm({
          storage_id: defaultLocal?.id || 0,
          base_path: basePath,
          enabled: true
        });
      }
    }
  }, [selectedModule, rules, configs]);

  const handleSaveConfig = async () => {
    if (!configForm.name.trim()) {
      toast.error("请输入配置名称");
      return;
    }

    const storageConfig = configForm.type === "s3" ? configForm.s3 : configForm.webdav;
    
    try {
      const testResult = await testStorageConnection({ type: configForm.type, config: storageConfig });
      if (!testResult.success) {
        toast.error(`连接测试失败: ${testResult.message}，无法保存`);
        return;
      }
    } catch {
      toast.error("连接测试异常，无法保存");
      return;
    }

    // WebDAV: 提示目录列表失败但允许保存（用户可手动指定路径）
    if (configForm.type === "webdav") {
      const webdavConfig = configForm.webdav;
      if (webdavDirs.length === 0 && !fetchingDirs && webdavConfig.url) {
        const confirmed = await new Promise<boolean>(resolve => {
          const dialogConfirmed = confirm(
            "⚠️ 无法获取 WebDAV 目录列表。\n\n" +
            "这可能是由于：\n" +
            "• AliyunDrive 等服务的 API 限流\n" +
            "• 服务商不支持目录列表功能\n" +
            "• 认证信息不正确\n\n" +
            `将使用目录: ${webdavConfig.directory || '/'}\n\n` +
            "是否继续保存配置？"
          );
          resolve(dialogConfirmed);
        });
        if (!confirmed) return;
      }
    }

    const payload = {
      name: configForm.name,
      type: configForm.type,
      enabled: configForm.enabled,
      config: storageConfig
    };

    try {
      if (editingConfig) {
        await updateStorageConfig(editingConfig.id, payload);
        toast.success("配置已更新");
      } else {
        await createStorageConfig(payload);
        toast.success("配置已创建");
      }
      setShowConfigDialog(false);
      loadData();
    } catch {
      toast.error("保存失败");
    }
  };

  const handleDeleteConfig = async (id: number) => {
    try {
      await deleteStorageConfig(id);
      toast.success("配置已删除");
      loadData();
    } catch {
      toast.error("删除失败");
    }
  };

  const handleSaveRule = async () => {
    if (!selectedModule) return;
    setSavingRule(true);
    try {
      const existingRule = rules.find(r => r.module_code === selectedModule.module_code);
      const payload = {
        module_code: selectedModule.module_code,
        storage_id: ruleForm.storage_id,
        base_path: ruleForm.base_path,
        enabled: ruleForm.enabled,
        priority: 0,
        name: `${selectedModule.module_name}存储规则`
      };

      if (existingRule) {
        await updateStorageRuleEnhanced(existingRule.id, payload);
      } else {
        await createStorageRuleEnhanced(payload);
      }
      toast.success("规则已保存");
      loadData();
    } catch {
      toast.error("保存规则失败");
    } finally {
      setSavingRule(false);
    }
  };

  const handleDeleteRule = async (id: number) => {
    try {
      await deleteStorageRuleEnhanced(id);
      toast.success("规则已删除");
      loadData();
    } catch {
      toast.error("删除失败");
    }
  };

  const handleRestoreDefault = async () => {
    if (!selectedModule) return;
    try {
      const defaultLocal = configs.find(c => c.is_default && c.type === "local") || configs.find(c => c.type === "local");
      const basePath = `/upload/${selectedModule.module_code}/{YYYY}`;
      
      setRuleForm({
        storage_id: defaultLocal?.id || 0,
        base_path: basePath,
        enabled: true
      });
      toast.success("已重置为默认值，请点击保存以生效");
    } catch {
      toast.error("恢复失败");
    } finally {
      setRestoreConfirm(false);
    }
  };

  const executeDeleteRule = async () => {
    if (deleteRuleConfirmId === null) return;
    try {
      await deleteStorageRuleEnhanced(deleteRuleConfirmId);
      toast.success("规则已删除");
      loadData();
    } catch {
      toast.error("删除失败");
    } finally {
      setDeleteRuleConfirmId(null);
    }
  };

  const getRuleDescription = (rule: StorageRule) => {
    const config = configs.find(c => c.id === rule.storage_id);
    const configName = config ? config.name : "未知存储";
    const moduleName = modules.find(m => m.module_code === rule.module_code)?.module_name || rule.module_code;
    
    let archiveType = "实时存储数据";
    const path = rule.base_path || "";
    if (path.includes("{YYYY}") && path.includes("{MM}") && path.includes("{DD}")) {
      archiveType = "并按照日(YYYY-MM-DD)归档";
    } else if (path.includes("{YYYY}") && path.includes("{MM}")) {
      archiveType = "并按照月(YYYY-MM)归档";
    } else if (path.includes("{YYYY}")) {
      archiveType = "并按照年(YYYY)归档";
    }

    return `「${moduleName}」存储于「${configName}」${archiveType}`;
  };

   const handleS3ProviderChange = (provider: string) => {
     const endpoints: Record<string, string> = {
       aliyun: "https://oss-cn-hangzhou.aliyuncs.com",
       tencent: "https://cos.ap-guangzhou.myqcloud.com",
       bitiful: "https://s3.bitiful.net",
       qiniu: "https://s3-cn-east-1.qiniucs.com"
     };
     setConfigForm(prev => ({
       ...prev,
       s3: { 
         ...prev.s3, 
         provider, 
         endpoint: provider === "custom" ? prev.s3.endpoint : endpoints[provider]
       }
     }));
   };

  return (
    <div className="space-y-6 animate-in fade-in duration-500 pb-10">
      <div className="flex flex-col gap-3">
        <div className="flex items-center justify-between">
          <h3 className="text-sm font-semibold flex items-center gap-2">
            <HardDrive className="w-4 h-4 text-primary" />
            存储节点
          </h3>
          <Button variant="ghost" size="sm" className="h-8 text-xs font-medium" onClick={() => {
            setEditingConfig(null);
            setConfigForm({
              name: "",
              type: "s3",
              enabled: true,
              s3: { endpoint: "https://oss-cn-hangzhou.aliyuncs.com", bucket: "", region: "", access_key: "", secret_key: "", provider: "aliyun" },
              webdav: { url: "", username: "", password: "", directory: "/" }
            });
            setShowConfigDialog(true);
          }}>
            <Plus className="w-3.5 h-3.5 mr-1" />
            添加节点
          </Button>
        </div>
        <ScrollArea className="w-full whitespace-nowrap pb-4">
          <div className="flex gap-4">
            {configs.map(config => {
              const res = testResults[config.id];
              const isDefaultLocal = config.is_default && config.type === "local";
              
              return (
                <Card key={config.id} className={cn(
                  "w-[260px] shrink-0 transition-all hover:shadow-md",
                  !config.enabled && "opacity-60 grayscale-[0.5]"
                )}>
                  <CardContent className="p-4 space-y-3">
                    <div className="flex items-start justify-between">
                      <div className="flex items-center gap-2.5">
                        <div className="p-2 bg-muted rounded-lg text-muted-foreground group-hover:text-primary transition-colors">
                          {getTypeIcon(config.type)}
                        </div>
                        <div className="flex flex-col min-w-0">
                          <span className="text-sm font-bold truncate leading-none mb-1">{config.name}</span>
                          <span className="text-[10px] text-muted-foreground font-medium uppercase tracking-tight">{getTypeLabel(config.type)}</span>
                        </div>
                      </div>
                      <div className="flex flex-col items-end gap-1">
                        <StorageStatusIndicator config={config} res={res} />
                        {res?.latency && config.enabled && (
                          <span className="text-[10px] font-mono text-muted-foreground">{res.latency}ms</span>
                        )}
                      </div>
                    </div>
                    
                    <div className="flex items-center justify-between pt-2 border-t mt-1">
                      <Button 
                        variant="ghost" 
                        size="sm" 
                        className="h-7 px-2 text-[11px] font-medium hover:bg-primary/5 hover:text-primary"
                        onClick={() => handleTest(config)}
                        disabled={res?.testing}
                      >
                        <RefreshCw className={cn("w-3 h-3 mr-1.5", res?.testing && "animate-spin")} />
                        测试
                      </Button>
                      <div className="flex gap-0.5">
                        {!isDefaultLocal && (
                          <>
                             <Button variant="ghost" size="sm" className="h-7 w-7 p-0" onClick={() => {
                               setEditingConfig(config);
                               const webdavConfig = config.type === "webdav" ? (config.config as unknown as typeof configForm.webdav) : { url: "", username: "", password: "", directory: "/" };
                               const s3ConfigData = config.type === "s3" ? (config.config as unknown as typeof configForm.s3) : { endpoint: "", bucket: "", region: "", access_key: "", secret_key: "", provider: "aliyun" };
                               setConfigForm({
                                 name: config.name,
                                 type: config.type as "s3" | "webdav",
                                 enabled: config.enabled,
                                 s3: s3ConfigData,
                                 webdav: webdavConfig
                               });
                               setShowConfigDialog(true);
                               if (config.type === "webdav") {
                                 fetchWebdavDirs("webdav", {
                                   webdav_url: webdavConfig.url,
                                   webdav_username: webdavConfig.username,
                                   webdav_password: webdavConfig.password,
                                   directory: "/"
                                 });
                               }
                             }}>
                              <Edit className="w-3.5 h-3.5" />
                            </Button>
                            <Button variant="ghost" size="sm" className="h-7 w-7 p-0 text-destructive hover:text-destructive hover:bg-destructive/5" onClick={() => {
                              setDeleteTargetId(config.id);
                              setDeleteConfigConfirmOpen(true);
                            }}>
                              <Trash2 className="w-3.5 h-3.5" />
                            </Button>
                          </>
                        )}
                        {isDefaultLocal && (
                          <Badge variant="outline" className="text-[9px] h-5 px-1.5 font-normal border-muted-foreground/20 text-muted-foreground">系统默认</Badge>
                        )}
                      </div>
                    </div>
                  </CardContent>
                </Card>
              );
            })}
          </div>
        </ScrollArea>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-[220px_1fr_1fr] gap-6 items-stretch">
        <Card className="flex flex-col transition-all hover:shadow-md border bg-card">
          <CardHeader className="py-4 px-5 border-b bg-muted/5">
            <CardTitle className="text-base font-bold text-foreground">业务模块</CardTitle>
          </CardHeader>
          <CardContent className="p-1 flex-1">
            <ScrollArea className="h-full max-h-[480px]">
              <div className="p-2 space-y-1">
                {modules.map(mod => {
                  const rule = rules.find(r => r.module_code === mod.module_code && r.enabled);
                  const storageId = rule?.storage_id;
                  const status = storageId ? testResults[storageId] : null;
                  const isOffline = status?.success === false;
                  const isHighLatency = status?.latency && status.latency > 500;

                  return (
                    <button
                      key={mod.module_code}
                      onClick={() => setSelectedModule(mod)}
                      className={cn(
                        "w-full text-left px-3 py-3 text-[13px] rounded-md transition-all flex items-center justify-between group",
                        selectedModule?.module_code === mod.module_code 
                          ? "bg-primary text-primary-foreground font-bold shadow-md ring-1 ring-primary/20" 
                          : "hover:bg-muted text-muted-foreground hover:text-foreground"
                      )}
                    >
                      <span className="truncate">{mod.module_name}</span>
                      {rule && (
                        <div className={cn(
                          "w-2 h-2 rounded-full",
                          selectedModule?.module_code === mod.module_code 
                            ? "bg-primary-foreground" 
                            : (isOffline ? "bg-red-500" : (isHighLatency ? "bg-orange-500" : "bg-emerald-500"))
                        )} />
                      )}
                    </button>
                  );
                })}
              </div>
            </ScrollArea>
          </CardContent>
        </Card>

        <Card className="flex flex-col transition-all hover:shadow-md border bg-card">
          <CardHeader className="py-4 px-6 border-b bg-muted/5">
            <CardTitle className="text-base font-bold flex items-center gap-2">
              <Settings2 className="w-5 h-5 text-primary" />
              规则配置: {selectedModule?.module_name}
            </CardTitle>
          </CardHeader>
          <CardContent className="p-6 space-y-8 flex-1">
            <div className="space-y-2.5">
              <Label className="text-[13px] font-medium">目标存储卷</Label>
              <Select value={String(ruleForm.storage_id)} onValueChange={v => setRuleForm(prev => ({ ...prev, storage_id: Number(v) }))}>
                <SelectTrigger className="h-10 text-sm">
                  <SelectValue placeholder="选择存储节点" />
                </SelectTrigger>
                <SelectContent>
                  {configs.filter(c => c.enabled).map(c => (
                    <SelectItem key={c.id} value={String(c.id)} className="text-sm">
                      <div className="flex items-center gap-2">
                        {getTypeIcon(c.type)}
                        <span>{c.name}</span>
                        {c.is_default && <Badge variant="secondary" className="text-[10px] h-4 px-1 ml-1 font-normal opacity-70">默认</Badge>}
                      </div>
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>

            <div className="space-y-2.5">
              <div className="flex items-center justify-between">
                <Label className="text-[13px] font-medium">基础映射路径 (Base Path)</Label>
                <div className="flex gap-1">
                  {["{YYYY}", "{MM}", "{DD}"].map(v => (
                    <button 
                      key={v}
                      className="text-[10px] px-1.5 py-0.5 bg-muted hover:bg-primary/10 hover:text-primary rounded-sm transition-colors border"
                      onClick={() => setRuleForm(prev => ({ ...prev, base_path: prev.base_path + v }))}
                    >
                      {v}
                    </button>
                  ))}
                </div>
              </div>
              <Input 
                value={ruleForm.base_path} 
                onChange={e => setRuleForm(prev => ({ ...prev, base_path: e.target.value }))}
                placeholder="例如: /uploads/{YYYY}/{MM}/"
                className="h-10 text-sm font-mono"
              />
              <p className="text-[11px] text-muted-foreground flex items-center gap-1.5 italic px-0.5">
                <Info className="w-3.5 h-3.5" />
                支持动态变量，例如: /archives/{"{YYYY}"}/{"{MM}"}
              </p>
            </div>

            <div className="pt-4 border-t">
              <div className="flex items-center justify-between p-4 rounded-xl border bg-muted/20">
                <div className="space-y-0.5">
                  <Label htmlFor="rule-enabled" className="text-sm font-bold cursor-pointer">启用此独立规则</Label>
                  <p className="text-[11px] text-muted-foreground">如果关闭，该模块将使用全局默认存储路径</p>
                </div>
                <Switch checked={ruleForm.enabled} onCheckedChange={v => setRuleForm(prev => ({ ...prev, enabled: v }))} id="rule-enabled" />
              </div>
            </div>
          </CardContent>
          <div className="px-6 py-4 border-t bg-muted/5 flex justify-end gap-3 shrink-0">
            {existingRule && (
              <Button onClick={() => setRestoreConfirm(true)} variant="outline" size="sm" className="h-9 px-4 text-xs font-medium">
                恢复默认
              </Button>
            )}
            <Button onClick={handleSaveRule} size="sm" className="h-9 px-8 text-xs font-bold shadow-md" disabled={savingRule}>
              {savingRule ? "正在保存..." : "保存"}
            </Button>
          </div>
        </Card>

        <Card className="flex flex-col transition-all hover:shadow-md border overflow-hidden bg-card">
          <CardHeader className="py-4 px-6 border-b bg-muted/5 flex flex-row items-center justify-between">
            <CardTitle className="text-sm font-bold uppercase tracking-wider text-muted-foreground">已生效规则</CardTitle>
            <Badge variant="outline" className="text-[10px] font-bold px-2 py-0.5">{rules.length} 条配置</Badge>
          </CardHeader>
          <CardContent className="p-0 flex-1 overflow-hidden">
            <ScrollArea className="h-full max-h-[450px]">
              <Table>
                <TableHeader className="bg-muted/10 sticky top-0 z-10">
                  <TableRow className="hover:bg-transparent border-none">
                    <TableHead className="text-sm h-10 px-4 text-center font-bold">业务模块</TableHead>
                    <TableHead className="text-sm h-10 px-4 text-center font-bold">状态显示</TableHead>
                    <TableHead className="text-sm h-10 px-4 text-center font-bold">配置说明</TableHead>
                    <TableHead className="text-sm h-10 px-4 text-center font-bold w-[60px]">操作</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {rules.length === 0 ? (
                    <TableRow>
                      <TableCell colSpan={4} className="text-center py-20 text-muted-foreground text-xs italic">
                        暂无自定义规则
                      </TableCell>
                    </TableRow>
                  ) : (
                    rules.map(rule => {
                      const mName = modules.find(m => m.module_code === rule.module_code)?.module_name || rule.module_code;
                      const cName = configs.find(c => c.id === rule.storage_id)?.name || "未知";
                      const isOffline = testResults[rule.storage_id]?.success === false;
                      const isSelected = selectedModule?.module_code === rule.module_code;

                      return (
                        <TableRow 
                          key={rule.id} 
                          className={cn(
                            "cursor-pointer transition-all group border-b",
                            isSelected && "bg-primary/[0.03]"
                          )}
                          onClick={() => setSelectedModule(modules.find(m => m.module_code === rule.module_code) || null)}
                        >
                          <TableCell className="py-4 px-4 text-center">
                            <div className="flex flex-col items-center">
                              <span className="text-[13px] font-bold">{mName}</span>
                              <span className="text-[10px] text-muted-foreground mt-0.5">{cName}</span>
                            </div>
                          </TableCell>
                          <TableCell className="py-4 px-4 text-center">
                             <div className="flex flex-col items-center gap-1">
                                <div className="flex items-center gap-1.5">
                                  <Circle className={cn(
                                    "w-2 h-2 fill-current", 
                                    !rule.enabled ? "text-muted-foreground" : (isOffline ? "text-red-500" : (testResults[rule.storage_id]?.latency && testResults[rule.storage_id]!.latency! > 500 ? "text-orange-500" : "text-emerald-500"))
                                  )} />
                                  <span className={cn(
                                    "text-xs font-bold",
                                    !rule.enabled ? "text-muted-foreground" : (isOffline ? "text-red-500" : (testResults[rule.storage_id]?.latency && testResults[rule.storage_id]!.latency! > 500 ? "text-orange-500" : "text-emerald-500"))
                                  )}>
                                    {!rule.enabled ? "已停用" : (isOffline ? "断联" : (testResults[rule.storage_id]?.latency && testResults[rule.storage_id]!.latency! > 500 ? "延迟高" : "正常"))}
                                  </span>
                                </div>
                                {testResults[rule.storage_id]?.latency && rule.enabled && !isOffline && (
                                  <span className="text-[10px] font-mono text-muted-foreground">{testResults[rule.storage_id]?.latency}ms</span>
                                )}
                             </div>
                          </TableCell>
                          <TableCell className="py-4 px-4">
                            <div className="flex items-center justify-center gap-2">
                              <div className={cn(
                                "w-1.5 h-1.5 rounded-full shrink-0",
                                rule.enabled ? (isOffline ? "bg-red-500" : "bg-emerald-500") : "bg-muted"
                              )} />
                              <span className="text-[11px] text-muted-foreground leading-snug line-clamp-2 max-w-[250px] text-center" title={getRuleDescription(rule)}>
                                {getRuleDescription(rule)}
                                {isOffline && rule.enabled && <span className="text-red-500 ml-1 font-bold">!</span>}
                              </span>
                            </div>
                          </TableCell>
                          <TableCell className="py-4 px-4 text-center">
                            <Button variant="ghost" size="sm" className="h-8 w-8 p-0 text-destructive transition-all hover:bg-destructive/10" onClick={(e) => {
                              e.stopPropagation();
                              setDeleteRuleConfirmId(rule.id);
                            }}>
                              <Trash2 className="w-4 h-4" />
                            </Button>
                          </TableCell>
                        </TableRow>
                      );
                    })
                  )}
                </TableBody>
              </Table>
            </ScrollArea>
          </CardContent>
        </Card>
      </div>

      <Dialog open={showConfigDialog} onOpenChange={setShowConfigDialog}>
        <DialogContent className="max-w-md">
          <DialogHeader>
            <DialogTitle className="flex items-center gap-2">
              {editingConfig ? <Edit className="w-4 h-4" /> : <Plus className="w-4 h-4" />}
              {editingConfig ? "编辑存储节点" : "新增存储节点"}
            </DialogTitle>
          </DialogHeader>
          <div className="space-y-4 py-2">
            <div className="grid grid-cols-2 gap-4">
              <div className="space-y-2">
                <Label>显示名称</Label>
                <Input value={configForm.name} onChange={e => setConfigForm(prev => ({ ...prev, name: e.target.value }))} placeholder="如: 缤纷云 S3" />
              </div>
              <div className="space-y-2">
                <Label>存储协议</Label>
                <Select value={configForm.type} onValueChange={v => setConfigForm(prev => ({ ...prev, type: v as "s3" | "webdav" }))}>
                  <SelectTrigger><SelectValue /></SelectTrigger>
                  <SelectContent>
                    <SelectItem value="s3">S3 兼容</SelectItem>
                    <SelectItem value="webdav">WebDAV</SelectItem>
                  </SelectContent>
                </Select>
              </div>
            </div>

            {configForm.type === "s3" ? (
              <div className="space-y-3 p-3 bg-muted/30 rounded-lg border border-muted">
                <div className="space-y-1.5">
                  <Label className="text-xs">服务商预设</Label>
                  <Select value={configForm.s3.provider} onValueChange={handleS3ProviderChange}>
                    <SelectTrigger className="h-8 text-xs"><SelectValue /></SelectTrigger>
                    <SelectContent>
                      <SelectItem value="aliyun">阿里云 OSS</SelectItem>
                      <SelectItem value="tencent">腾讯云 COS</SelectItem>
                      <SelectItem value="bitiful">缤纷云 Bitiful</SelectItem>
                      <SelectItem value="qiniu">七牛云 Kodo</SelectItem>
                      <SelectItem value="custom">自定义 (S3 兼容)</SelectItem>
                    </SelectContent>
                  </Select>
                </div>
                <div className="space-y-1.5">
                  <Label className="text-xs">Endpoint (访问端点)</Label>
                  <Input value={configForm.s3.endpoint} onChange={e => setConfigForm(prev => ({ ...prev, s3: { ...prev.s3, endpoint: e.target.value } }))} placeholder="https://..." className="h-8 text-xs" />
                </div>
                <div className="grid grid-cols-2 gap-3">
                  <div className="space-y-1.5">
                    <Label className="text-xs">Bucket (存储桶)</Label>
                    <Input value={configForm.s3.bucket} onChange={e => setConfigForm(prev => ({ ...prev, s3: { ...prev.s3, bucket: e.target.value } }))} placeholder="my-bucket" className="h-8 text-xs" />
                  </div>
                  <div className="space-y-1.5">
                    <Label className="text-xs">Region (区域)</Label>
                    <Input value={configForm.s3.region} onChange={e => setConfigForm(prev => ({ ...prev, s3: { ...prev.s3, region: e.target.value } }))} placeholder="cn-hangzhou" className="h-8 text-xs" />
                  </div>
                </div>
                <div className="space-y-1.5">
                  <Label className="text-xs">Access Key</Label>
                  <Input value={configForm.s3.access_key} onChange={e => setConfigForm(prev => ({ ...prev, s3: { ...prev.s3, access_key: e.target.value } }))} className="h-8 text-xs" />
                </div>
                <div className="space-y-1.5">
                  <Label className="text-xs">Secret Key</Label>
                  <Input type="password" value={configForm.s3.secret_key} onChange={e => setConfigForm(prev => ({ ...prev, s3: { ...prev.s3, secret_key: e.target.value } }))} className="h-8 text-xs" />
                </div>
              </div>
            ) : (
              <div className="space-y-3 p-3 bg-muted/30 rounded-lg border border-muted">
                <div className="space-y-1.5">
                  <Label className="text-xs">WebDAV URL</Label>
                  <Input value={configForm.webdav.url} onChange={e => setConfigForm(prev => ({ ...prev, webdav: { ...prev.webdav, url: e.target.value } }))} placeholder="https://..." className="h-8 text-xs" />
                  <p className="text-[10px] text-muted-foreground">如: https://al.mozui.cn/dav (自动追加 /dav 后缀)</p>
                </div>
                <div className="grid grid-cols-2 gap-3">
                  <div className="space-y-1.5">
                    <Label className="text-xs">用户名</Label>
                    <Input value={configForm.webdav.username} onChange={e => setConfigForm(prev => ({ ...prev, webdav: { ...prev.webdav, username: e.target.value } }))} className="h-8 text-xs" />
                  </div>
                  <div className="space-y-1.5">
                    <Label className="text-xs">密码</Label>
                    <div className="relative">
                      <Input 
                        type={showWebdavPassword ? "text" : "password"} 
                        value={configForm.webdav.password} 
                        onChange={e => setConfigForm(prev => ({ ...prev, webdav: { ...prev.webdav, password: e.target.value } }))} 
                        className="h-8 text-xs pr-8" 
                      />
                      <button
                        type="button"
                        onClick={() => setShowWebdavPassword(!showWebdavPassword)}
                        className="absolute right-2 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground"
                      >
                        {showWebdavPassword ? (
                          <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><path d="M17.94 17.94A10.07 10.07 0 0 1 12 20c-7 0-11-8-11-8a18.45 18.45 0 0 1 5.06-5.94M9.9 4.24A9.12 9.12 0 0 1 12 4c7 0 11 8 11 8a18.5 18.5 0 0 1-2.16 3.19m-6.72-1.07a3 3 0 1 1-4.24-4.24"/><line x1="1" x2="23" y1="1" y2="23"/></svg>
                        ) : (
                          <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z"/><circle cx="12" cy="12" r="3"/></svg>
                        )}
                      </button>
                    </div>
                  </div>
                </div>

                <div className="space-y-1.5 pt-2 border-t mt-2">
                  <Label className="text-xs text-primary font-semibold">选择存储目录</Label>
                  {webdavDirs.length > 0 ? (
                    <Select value={configForm.webdav.directory || "/"} onValueChange={(v) => setConfigForm(prev => ({ ...prev, webdav: { ...prev.webdav, directory: v } }))}>
                      <SelectTrigger className="h-8 text-xs"><SelectValue placeholder="选择目录" /></SelectTrigger>
                      <SelectContent>
                        <SelectItem value="/" className="text-xs">/ (根目录)</SelectItem>
                        {webdavDirs.map(d => <SelectItem key={d} value={d} className="text-xs">{d}</SelectItem>)}
                      </SelectContent>
                    </Select>
                  ) : (
                    <Input 
                      value={configForm.webdav.directory || "/"} 
                      onChange={e => setConfigForm(prev => ({ ...prev, webdav: { ...prev.webdav, directory: e.target.value } }))} 
                      placeholder="/ (根目录)" 
                      className="h-8 text-xs" 
                    />
                  )}
                </div>

                {fetchingDirs && (
                  <div className="space-y-1 bg-yellow-50 border border-yellow-200 rounded p-2">
                    <p className="text-[10px] text-yellow-900 font-semibold animate-pulse">⏳ 正在扫描 WebDAV 目录...</p>
                  </div>
                )}
              </div>
            )}
          </div>
          <DialogFooter className="gap-2">
<Button variant="outline" size="sm" onClick={() => {
                if (configForm.type === "webdav") {
                  const conf = {
                    type: "webdav",
                    config: {
                      url: configForm.webdav.url,
                      webdav_username: configForm.webdav.username,
                      webdav_password: configForm.webdav.password
                    }
                  };
                  testStorageConnection(conf).then(res => {
                    if (res.success) {
                      toast.success(`连接测试成功 (${res.latency_ms}ms)`);
                      fetchWebdavDirs("webdav", {
                      webdav_url: configForm.webdav.url,
                      webdav_username: configForm.webdav.username,
                      webdav_password: configForm.webdav.password
                    });
                    }
                    else toast.error(`连接测试失败: ${res.message}`);
                  });
                } else {
                  const conf = {
                    type: "s3",
                    config: {
                      endpoint: configForm.s3.endpoint,
                      bucket: configForm.s3.bucket,
                      region: configForm.s3.region,
                      access_key: configForm.s3.access_key,
                      secret_key: configForm.s3.secret_key
                    }
                  };
                  testStorageConnection(conf).then(res => {
                    if (res.success) toast.success(`连接测试成功 (${res.latency_ms}ms)`);
                    else toast.error(`连接测试失败: ${res.message}`);
                  });
                }
            }}>测试连接</Button>
            <Button size="sm" onClick={handleSaveConfig}>保存配置</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <AlertDialog open={restoreConfirm} onOpenChange={setRestoreConfirm}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>确认恢复默认？</AlertDialogTitle>
            <AlertDialogDescription>这将重置当前模块的存储策略。目标存储将回退到“本地存储”，基础路径将恢复为标准路径。确定继续吗？</AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>取消</AlertDialogCancel>
            <AlertDialogAction onClick={handleRestoreDefault} className="bg-primary text-primary-foreground hover:bg-primary/90">确认恢复</AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      <AlertDialog open={deleteConfigConfirmOpen} onOpenChange={setDeleteConfigConfirmOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>确认删除存储节点？</AlertDialogTitle>
            <AlertDialogDescription>此操作不可撤销。删除此存储节点可能会导致依赖它的存储规则失效。确定要继续吗？</AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>取消</AlertDialogCancel>
            <AlertDialogAction 
              onClick={() => {
                if (deleteTargetConfigId) handleDeleteConfig(deleteTargetConfigId);
                setDeleteConfigConfirmOpen(false);
              }} 
              className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
            >
              确认删除
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      <AlertDialog open={deleteRuleConfirmId !== null} onOpenChange={(open) => !open && setDeleteRuleConfirmId(null)}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>确认删除存储规则？</AlertDialogTitle>
            <AlertDialogDescription>删除该规则后，对应的业务模块将回退到全局默认存储设置。确定要继续吗？</AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>取消</AlertDialogCancel>
            <AlertDialogAction onClick={executeDeleteRule} className="bg-destructive text-destructive-foreground hover:bg-destructive/90">确认删除</AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
}


function ArchiveGlobalTab() {
  const [activeTab, setActiveTab] = useState("retention");

  const [periods, setPeriods] = useState<RetentionPeriod[]>([]);
  const [loadingRetention, setLoadingRetention] = useState(false);
  const [showRetentionDialog, setShowRetentionDialog] = useState(false);
  const [editingRetention, setEditingRetention] = useState<RetentionPeriod | null>(null);
  const [retentionName, setRetentionName] = useState("");
  const [retentionYears, setRetentionYears] = useState("1");

  const [locations, setLocations] = useState<StorageLocation[]>([]);
  const [loadingLocations, setLoadingLocations] = useState(false);
  const [showLocationDialog, setShowLocationDialog] = useState(false);
  const [editingLocation, setEditingLocation] = useState<StorageLocation | null>(null);
  const [locationName, setLocationName] = useState("");
  const [locationDesc, setLocationDesc] = useState("");

  const [rules, setRules] = useState<CodeRule[]>([]);
  const [loadingRules, setLoadingRules] = useState(false);
  const [showRuleDialog, setShowRuleDialog] = useState(false);
  const [editingRule, setEditingRule] = useState<CodeRule | null>(null);
  const [ruleName, setRuleName] = useState("");
  const [rulePattern, setRulePattern] = useState("");
  const [rulePreview, setRulePreview] = useState<CodeRulePreview | null>(null);

  const loadRetentionData = useCallback(async () => {
    setLoadingRetention(true);
    try {
      const data = await fetchRetentionPeriods();
      setPeriods(data);
    } catch { toast.error("加载保管期限失败"); }
    finally { setLoadingRetention(false); }
  }, []);

  const loadLocationData = useCallback(async () => {
    setLoadingLocations(true);
    try {
      const data = await fetchStorageLocations();
      setLocations(data);
    } catch { toast.error("加载存档地点失败"); }
    finally { setLoadingLocations(false); }
  }, []);

  const loadRuleData = useCallback(async () => {
    setLoadingRules(true);
    try {
      const data = await fetchCodeRules();
      setRules(data);
    } catch { toast.error("加载编码规则失败"); }
    finally { setLoadingRules(false); }
  }, []);

  useEffect(() => {
    if (activeTab === "retention") loadRetentionData();
    else if (activeTab === "locations") loadLocationData();
    else if (activeTab === "code-rules") loadRuleData();
  }, [activeTab, loadRetentionData, loadLocationData, loadRuleData]);

  const handleSaveRetention = async () => {
    if (!retentionName.trim()) { toast.error("名称不能为空"); return; }
    try {
      if (editingRetention) await updateRetentionPeriod(editingRetention.id, { name: retentionName, years: parseInt(retentionYears) });
      else await createRetentionPeriod({ name: retentionName, years: parseInt(retentionYears) });
      toast.success("保存成功");
      setShowRetentionDialog(false);
      loadRetentionData();
    } catch { toast.error("保存失败"); }
  };

  const handleDeleteRetention = async (id: number) => {
    if (!confirm("确定要删除此保管期限吗?")) return;
    try { await deleteRetentionPeriod(id); toast.success("已删除"); loadRetentionData(); } catch { toast.error("删除失败"); }
  };

  const handleSaveLocation = async () => {
    if (!locationName.trim()) { toast.error("名称不能为空"); return; }
    try {
      if (editingLocation) await updateStorageLocation(editingLocation.id, { name: locationName, description: locationDesc });
      else await createStorageLocation({ name: locationName, description: locationDesc });
      toast.success("保存成功");
      setShowLocationDialog(false);
      loadLocationData();
    } catch { toast.error("保存失败"); }
  };

  const handleDeleteLocation = async (id: number) => {
    if (!confirm("确定要删除此存档地点吗?")) return;
    try { await deleteStorageLocation(id); toast.success("已删除"); loadLocationData(); } catch { toast.error("删除失败"); }
  };

  const handleSaveRule = async () => {
    if (!ruleName.trim() || !rulePattern.trim()) { toast.error("名称和模式不能为空"); return; }
    try {
      if (editingRule) await updateCodeRule(editingRule.id, { name: ruleName, pattern: rulePattern });
      else await createCodeRule({ name: ruleName, pattern: rulePattern });
      toast.success("保存成功");
      setShowRuleDialog(false);
      loadRuleData();
    } catch { toast.error("保存失败"); }
  };

  const handleDeleteRule = async (id: number) => {
    if (!confirm("确定要删除此编码规则吗?")) return;
    try { await deleteCodeRule(id); toast.success("已删除"); loadRuleData(); } catch { toast.error("删除失败"); }
  };

  const handlePreviewRule = async () => {
    try {
      const res = await getCodeRulePreview("", "", new Date().getFullYear());
      setRulePreview(res);
    } catch { toast.error("预览失败"); }
  };

  return (
    <Card className="w-full">
      <CardHeader>
        <CardTitle>全局配置</CardTitle>
        <CardDescription>管理档案保管期限、存档地点及系统全局编码规则</CardDescription>
      </CardHeader>
      <CardContent>
        <Tabs value={activeTab} onValueChange={setActiveTab} className="space-y-4">
          <TabsList className="grid w-full grid-cols-3">
            <TabsTrigger value="retention" className="flex items-center gap-2">
              <ShieldCheck className="w-4 h-4" />
              保管期限
            </TabsTrigger>
            <TabsTrigger value="locations" className="flex items-center gap-2">
              <Database className="w-4 h-4" />
              存档地点
            </TabsTrigger>
            <TabsTrigger value="code-rules" className="flex items-center gap-2">
              <Sliders className="w-4 h-4" />
              编码规则
            </TabsTrigger>
          </TabsList>

          <TabsContent value="retention" className="space-y-4">
            <div className="flex justify-between items-center mb-4">
               <div>
                 <h4 className="text-sm font-medium">保管期限列表</h4>
                 <p className="text-xs text-muted-foreground">定义档案可供选择的存放年限</p>
               </div>
               <Button size="sm" onClick={() => { setEditingRetention(null); setRetentionName(""); setRetentionYears("1"); setShowRetentionDialog(true); }}>
                 <Plus className="w-4 h-4 mr-2" />
                 新增期限
               </Button>
            </div>
            <div className="border rounded-md">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>名称</TableHead>
                    <TableHead>年数</TableHead>
                    <TableHead className="text-right">操作</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {loadingRetention ? (
                    <TableRow><TableCell colSpan={3} className="text-center py-4 text-muted-foreground">加载中...</TableCell></TableRow>
                  ) : periods.length === 0 ? (
                    <TableRow><TableCell colSpan={3} className="text-center py-4 text-muted-foreground">暂无数据</TableCell></TableRow>
                  ) : (
                    periods.map(p => (
                      <TableRow key={p.id}>
                        <TableCell className="text-sm font-medium">{p.name}</TableCell>
                        <TableCell className="text-sm">{p.years} 年</TableCell>
                        <TableCell className="text-right">
                          <div className="flex justify-end gap-1">
                            <Button variant="ghost" size="sm" onClick={() => { setEditingRetention(p); setRetentionName(p.name); setRetentionYears(String(p.years)); setShowRetentionDialog(true); }}>
                              <Edit className="w-4 h-4" />
                            </Button>
                            <Button variant="ghost" size="sm" onClick={() => handleDeleteRetention(p.id)}>
                              <Trash2 className="w-4 h-4 text-destructive" />
                            </Button>
                          </div>
                        </TableCell>
                      </TableRow>
                    ))
                  )}
                </TableBody>
              </Table>
            </div>
          </TabsContent>

          <TabsContent value="locations" className="space-y-4">
            <div className="flex justify-between items-center mb-4">
               <div>
                 <h4 className="text-sm font-medium">存档地点列表</h4>
                 <p className="text-xs text-muted-foreground">物理档案的存放库房或货架位置</p>
               </div>
               <Button size="sm" onClick={() => { setEditingLocation(null); setLocationName(""); setLocationDesc(""); setShowLocationDialog(true); }}>
                 <Plus className="w-4 h-4 mr-2" />
                 新增地点
               </Button>
            </div>
            <div className="border rounded-md">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>名称</TableHead>
                    <TableHead>描述/详细地址</TableHead>
                    <TableHead className="text-right">操作</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {loadingLocations ? (
                    <TableRow><TableCell colSpan={3} className="text-center py-4 text-muted-foreground">加载中...</TableCell></TableRow>
                  ) : locations.length === 0 ? (
                    <TableRow><TableCell colSpan={3} className="text-center py-4 text-muted-foreground">暂无数据</TableCell></TableRow>
                  ) : (
                    locations.map(l => (
                      <TableRow key={l.id}>
                        <TableCell className="text-sm font-medium">{l.name}</TableCell>
                        <TableCell className="text-sm text-muted-foreground">{l.description || "-"}</TableCell>
                        <TableCell className="text-right">
                          <div className="flex justify-end gap-1">
                            <Button variant="ghost" size="sm" onClick={() => { setEditingLocation(l); setLocationName(l.name); setLocationDesc(l.description || ""); setShowLocationDialog(true); }}>
                              <Edit className="w-4 h-4" />
                            </Button>
                            <Button variant="ghost" size="sm" onClick={() => handleDeleteLocation(l.id)}>
                              <Trash2 className="w-4 h-4 text-destructive" />
                            </Button>
                          </div>
                        </TableCell>
                      </TableRow>
                    ))
                  )}
                </TableBody>
              </Table>
            </div>
          </TabsContent>

          <TabsContent value="code-rules" className="space-y-4">
            <div className="flex justify-between items-center mb-4">
               <div>
                 <h4 className="text-sm font-medium">编码规则列表</h4>
                 <p className="text-xs text-muted-foreground">定义档案编号自动生成的模板格式</p>
               </div>
               <Button size="sm" onClick={() => { setEditingRule(null); setRuleName(""); setRulePattern(""); setRulePreview(null); setShowRuleDialog(true); }}>
                 <Plus className="w-4 h-4 mr-2" />
                 新增规则
               </Button>
            </div>
            <div className="border rounded-md">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>名称</TableHead>
                    <TableHead>模式</TableHead>
                    <TableHead className="text-right">操作</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {loadingRules ? (
                    <TableRow><TableCell colSpan={3} className="text-center py-4 text-muted-foreground">加载中...</TableCell></TableRow>
                  ) : rules.length === 0 ? (
                    <TableRow><TableCell colSpan={3} className="text-center py-4 text-muted-foreground">暂无数据</TableCell></TableRow>
                  ) : (
                    rules.map(r => (
                      <TableRow key={r.id}>
                        <TableCell className="text-sm font-medium">{r.name}</TableCell>
                        <TableCell className="text-xs font-mono text-muted-foreground">{r.pattern}</TableCell>
                        <TableCell className="text-right">
                          <div className="flex justify-end gap-1">
                            <Button variant="ghost" size="sm" onClick={() => { setEditingRule(r); setRuleName(r.name); setRulePattern(r.pattern); setRulePreview(null); setShowRuleDialog(true); }}>
                              <Edit className="w-4 h-4" />
                            </Button>
                            <Button variant="ghost" size="sm" onClick={() => handleDeleteRule(r.id)}>
                              <Trash2 className="w-4 h-4 text-destructive" />
                            </Button>
                          </div>
                        </TableCell>
                      </TableRow>
                    ))
                  )}
                </TableBody>
              </Table>
            </div>
          </TabsContent>
        </Tabs>
      </CardContent>

      <Dialog open={showRetentionDialog} onOpenChange={setShowRetentionDialog}>
        <DialogContent>
          <DialogHeader><DialogTitle>{editingRetention ? "编辑" : "新增"}保管期限</DialogTitle></DialogHeader>
          <div className="space-y-4 py-2">
            <div className="space-y-2"><Label>名称</Label><Input value={retentionName} onChange={e => setRetentionName(e.target.value)} placeholder="如: 长期保存" /></div>
            <div className="space-y-2"><Label>年数 (0 表示永久)</Label><Input type="number" value={retentionYears} onChange={e => setRetentionYears(e.target.value)} /></div>
          </div>
          <DialogFooter><Button onClick={handleSaveRetention}>保存</Button></DialogFooter>
        </DialogContent>
      </Dialog>

      <Dialog open={showLocationDialog} onOpenChange={setShowLocationDialog}>
        <DialogContent>
          <DialogHeader><DialogTitle>{editingLocation ? "编辑" : "新增"}存档地点</DialogTitle></DialogHeader>
          <div className="space-y-4 py-2">
            <div className="space-y-2"><Label>名称</Label><Input value={locationName} onChange={e => setLocationName(e.target.value)} placeholder="如: 档案室A" /></div>
            <div className="space-y-2"><Label>详细说明</Label><Textarea value={locationDesc} onChange={e => setLocationDesc(e.target.value)} placeholder="货架编号、地址等" /></div>
          </div>
          <DialogFooter><Button onClick={handleSaveLocation}>保存</Button></DialogFooter>
        </DialogContent>
      </Dialog>

      <Dialog open={showRuleDialog} onOpenChange={setShowRuleDialog}>
        <DialogContent className="max-w-2xl">
          <DialogHeader><DialogTitle>{editingRule ? "编辑" : "新增"}编码规则</DialogTitle></DialogHeader>
          <div className="space-y-4 py-2">
            <div className="space-y-2"><Label>名称</Label><Input value={ruleName} onChange={e => setRuleName(e.target.value)} placeholder="如: 合同类编码" /></div>
            <div className="space-y-2">
              <Label>模式</Label>
              <Textarea value={rulePattern} onChange={e => setRulePattern(e.target.value)} placeholder="{YYYY}{MM}{DD}-{SEQ:4}" rows={3} />
              <p className="text-[10px] text-muted-foreground">支持占位符: {'{YYYY}'}, {'{MM}'}, {'{DD}'}, {'{SEQ:4}'}, {'{CAT}'}, {'{SUBCAT}'}</p>
            </div>
            {rulePreview && (
              <div className="p-3 bg-muted rounded-md text-xs font-mono">预览示例: {rulePreview.sample_code}</div>
            )}
            <Button variant="secondary" size="sm" className="w-full" onClick={handlePreviewRule}><Eye className="w-3.5 h-3.5 mr-1" />实时预览</Button>
          </div>
          <DialogFooter><Button onClick={handleSaveRule}>保存规则</Button></DialogFooter>
        </DialogContent>
      </Dialog>
    </Card>
  );
}

// ============ 主组件 ============


function RetentionPeriodsTab() {
  const [periods, setPeriods] = useState<RetentionPeriod[]>([]);
  const [loading, setLoading] = useState(false);
  const [showDialog, setShowDialog] = useState(false);
  const [editingPeriod, setEditingPeriod] = useState<RetentionPeriod | null>(null);
  const [periodName, setPeriodName] = useState("");
  const [periodYears, setPeriodYears] = useState("1");

  useEffect(() => {
    loadPeriods();
  }, []);

  const loadPeriods = async () => {
    setLoading(true);
    try {
      const data = await fetchRetentionPeriods();
      setPeriods(data);
    } catch (error) {
      console.error("加载期限失败:", error);
      toast.error("加载期限失败");
    } finally {
      setLoading(false);
    }
  };

  const handleSave = async () => {
    if (!periodName.trim() || !periodYears.trim()) {
      toast.error("期限名称和年数不能为空");
      return;
    }

    try {
      if (editingPeriod) {
        await updateRetentionPeriod(editingPeriod.id, { name: periodName, years: parseInt(periodYears) });
        toast.success("期限已更新");
      } else {
        await createRetentionPeriod({ name: periodName, years: parseInt(periodYears) });
        toast.success("期限已创建");
      }
      setShowDialog(false);
      setPeriodName("");
      setPeriodYears("1");
      loadPeriods();
    } catch (error) {
      console.error("保存失败:", error);
      toast.error("保存失败");
    }
  };

   const handleDelete = async (id: number) => {
     if (confirm("确定要删除此期限吗？")) {
       try {
         await deleteRetentionPeriod(id);
         toast.success("期限已删除");
         loadPeriods();
       } catch (error) {
         console.error("删除失败:", error);
         toast.error("删除失败");
       }
     }
   };

  return (
    <Card>
      <CardHeader>
        <div className="flex items-center justify-between">
          <div>
            <CardTitle>保管期限</CardTitle>
            <CardDescription>管理档案保管期限</CardDescription>
          </div>
          <Button onClick={() => { setEditingPeriod(null); setPeriodName(""); setPeriodYears("1"); setShowDialog(true); }}>
            <Plus className="w-4 h-4 mr-2" />
            新增期限
          </Button>
        </div>
      </CardHeader>
      <CardContent>
        {loading ? (
          <div className="text-center py-8 text-muted-foreground">加载中...</div>
        ) : (
          <div className="overflow-x-auto">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>期限名称</TableHead>
                  <TableHead>年数</TableHead>
                  <TableHead>操作</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {periods.map((period) => (
                  <TableRow key={period.id}>
                    <TableCell>{period.name}</TableCell>
                    <TableCell>{period.years} 年</TableCell>
                    <TableCell>
                      <div className="flex gap-2">
                        <Button variant="outline" size="sm" onClick={() => { setEditingPeriod(period); setPeriodName(period.name); setPeriodYears(String(period.years)); setShowDialog(true); }}>
                          <Edit className="w-4 h-4" />
                        </Button>
                        <Button variant="outline" size="sm" onClick={() => handleDelete(period.id)}>
                          <Trash2 className="w-4 h-4" />
                        </Button>
                      </div>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </div>
        )}
      </CardContent>

      <Dialog open={showDialog} onOpenChange={setShowDialog}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{editingPeriod ? "编辑期限" : "新增期限"}</DialogTitle>
          </DialogHeader>
          <div className="space-y-4">
            <div className="grid gap-2">
              <Label htmlFor="period-name">期限名称</Label>
              <Input id="period-name" value={periodName} onChange={(e) => setPeriodName(e.target.value)} />
            </div>
            <div className="grid gap-2">
              <Label htmlFor="period-years">年数</Label>
              <Input id="period-years" type="number" value={periodYears} onChange={(e) => setPeriodYears(e.target.value)} />
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setShowDialog(false)}>
              取消
            </Button>
            <Button onClick={handleSave}>保存</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </Card>
  );
}

// ============ 存档地点管理 Tab ============
function StorageLocationsTab() {
  const [locations, setLocations] = useState<StorageLocation[]>([]);
  const [loading, setLoading] = useState(false);
  const [showDialog, setShowDialog] = useState(false);
  const [editingLocation, setEditingLocation] = useState<StorageLocation | null>(null);
  const [locationName, setLocationName] = useState("");
  const [locationAddress, setLocationAddress] = useState("");

  useEffect(() => {
    loadLocations();
  }, []);

  const loadLocations = async () => {
    setLoading(true);
    try {
      const data = await fetchStorageLocations();
      setLocations(data);
    } catch (error) {
      console.error("加载地点失败:", error);
      toast.error("加载地点失败");
    } finally {
      setLoading(false);
    }
  };

   const handleSave = async () => {
     if (!locationName.trim() || !locationAddress.trim()) {
       toast.error("地点名称和地址不能为空");
       return;
     }

     try {
       if (editingLocation) {
         await updateStorageLocation(editingLocation.id, { name: locationName, description: locationAddress });
         toast.success("地点已更新");
       } else {
         await createStorageLocation({ name: locationName, description: locationAddress });
         toast.success("地点已创建");
       }
       setShowDialog(false);
       setLocationName("");
       setLocationAddress("");
       loadLocations();
     } catch (error) {
       console.error("保存失败:", error);
       toast.error("保存失败");
     }
   };

   const handleDelete = async (id: number) => {
     if (confirm("确定要删除此地点吗？")) {
       try {
         await deleteStorageLocation(id);
         toast.success("地点已删除");
         loadLocations();
       } catch (error) {
         console.error("删除失败:", error);
         toast.error("删除失败");
       }
     }
   };

  return (
    <Card>
      <CardHeader>
        <div className="flex items-center justify-between">
          <div>
            <CardTitle>存档地点</CardTitle>
            <CardDescription>管理档案存档地点</CardDescription>
          </div>
          <Button onClick={() => { setEditingLocation(null); setLocationName(""); setLocationAddress(""); setShowDialog(true); }}>
            <Plus className="w-4 h-4 mr-2" />
            新增地点
          </Button>
        </div>
      </CardHeader>
      <CardContent>
        {loading ? (
          <div className="text-center py-8 text-muted-foreground">加载中...</div>
        ) : (
          <div className="overflow-x-auto">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>地点名称</TableHead>
                  <TableHead>地址</TableHead>
                  <TableHead>操作</TableHead>
                </TableRow>
              </TableHeader>
               <TableBody>
                 {locations.map((loc) => (
                   <TableRow key={loc.id}>
                     <TableCell>{loc.name}</TableCell>
                     <TableCell>{loc.description}</TableCell>
                     <TableCell>
                       <div className="flex gap-2">
                         <Button variant="outline" size="sm" onClick={() => { setEditingLocation(loc); setLocationName(loc.name); setLocationAddress(loc.description || ""); setShowDialog(true); }}>
                           <Edit className="w-4 h-4" />
                         </Button>
                         <Button variant="outline" size="sm" onClick={() => handleDelete(loc.id)}>
                           <Trash2 className="w-4 h-4" />
                         </Button>
                       </div>
                     </TableCell>
                   </TableRow>
                 ))}
               </TableBody>
            </Table>
          </div>
        )}
      </CardContent>

      <Dialog open={showDialog} onOpenChange={setShowDialog}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{editingLocation ? "编辑地点" : "新增地点"}</DialogTitle>
          </DialogHeader>
          <div className="space-y-4">
            <div className="grid gap-2">
              <Label htmlFor="loc-name">地点名称</Label>
              <Input id="loc-name" value={locationName} onChange={(e) => setLocationName(e.target.value)} />
            </div>
            <div className="grid gap-2">
              <Label htmlFor="loc-address">地址</Label>
              <Input id="loc-address" value={locationAddress} onChange={(e) => setLocationAddress(e.target.value)} />
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setShowDialog(false)}>
              取消
            </Button>
            <Button onClick={handleSave}>保存</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </Card>
  );
}

// ============ 编码规则管理 Tab ============
function CodeRulesTab() {
  const [rules, setRules] = useState<CodeRule[]>([]);
  const [loading, setLoading] = useState(false);
  const [showDialog, setShowDialog] = useState(false);
  const [editingRule, setEditingRule] = useState<CodeRule | null>(null);
  const [ruleName, setRuleName] = useState("");
  const [rulePattern, setRulePattern] = useState("");
  const [preview, setPreview] = useState<CodeRulePreview | null>(null);

  useEffect(() => {
    loadRules();
  }, []);

  const loadRules = async () => {
    setLoading(true);
    try {
      const data = await fetchCodeRules();
      setRules(data);
    } catch (error) {
      console.error("加载规则失败:", error);
      toast.error("加载规则失败");
    } finally {
      setLoading(false);
    }
  };

   const handlePreview = async () => {
     if (!rulePattern.trim()) {
       toast.error("规则不能为空");
       return;
     }

     try {
       const result = await getCodeRulePreview("", "", new Date().getFullYear());
       setPreview(result);
     } catch (error) {
       console.error("预览失败:", error);
       toast.error("预览失败");
     }
   };

  const handleSave = async () => {
    if (!ruleName.trim() || !rulePattern.trim()) {
      toast.error("规则名称和规则不能为空");
      return;
    }

    try {
      if (editingRule) {
        await updateCodeRule(editingRule.id, { name: ruleName, pattern: rulePattern });
        toast.success("规则已更新");
      } else {
        await createCodeRule({ name: ruleName, pattern: rulePattern });
        toast.success("规则已创建");
      }
      setShowDialog(false);
      setRuleName("");
      setRulePattern("");
      setPreview(null);
      loadRules();
    } catch (error) {
      console.error("保存失败:", error);
      toast.error("保存失败");
    }
  };

   const handleDelete = async (id: number) => {
     if (confirm("确定要删除此规则吗？")) {
       try {
         await deleteCodeRule(id);
         toast.success("规则已删除");
         loadRules();
       } catch (error) {
         console.error("删除失败:", error);
         toast.error("删除失败");
       }
     }
   };

  return (
    <Card>
      <CardHeader>
        <div className="flex items-center justify-between">
          <div>
            <CardTitle>编码规则</CardTitle>
            <CardDescription>管理档案编码生成规则</CardDescription>
          </div>
          <Button onClick={() => { setEditingRule(null); setRuleName(""); setRulePattern(""); setPreview(null); setShowDialog(true); }}>
            <Plus className="w-4 h-4 mr-2" />
            新增规则
          </Button>
        </div>
      </CardHeader>
      <CardContent>
        {loading ? (
          <div className="text-center py-8 text-muted-foreground">加载中...</div>
        ) : (
          <div className="overflow-x-auto">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>规则名称</TableHead>
                  <TableHead>规则模式</TableHead>
                  <TableHead>操作</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {rules.map((rule) => (
                  <TableRow key={rule.id}>
                    <TableCell>{rule.name}</TableCell>
                    <TableCell className="font-mono text-sm">{rule.pattern}</TableCell>
                    <TableCell>
                      <div className="flex gap-2">
                        <Button variant="outline" size="sm" onClick={() => { setEditingRule(rule); setRuleName(rule.name); setRulePattern(rule.pattern); setShowDialog(true); }}>
                          <Edit className="w-4 h-4" />
                        </Button>
                        <Button variant="outline" size="sm" onClick={() => handleDelete(rule.id)}>
                          <Trash2 className="w-4 h-4" />
                        </Button>
                      </div>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </div>
        )}
      </CardContent>

      <Dialog open={showDialog} onOpenChange={setShowDialog}>
        <DialogContent className="max-w-2xl">
          <DialogHeader>
            <DialogTitle>{editingRule ? "编辑规则" : "新增规则"}</DialogTitle>
          </DialogHeader>
          <div className="space-y-4">
            <div className="grid gap-2">
              <Label htmlFor="rule-name">规则名称</Label>
              <Input id="rule-name" value={ruleName} onChange={(e) => setRuleName(e.target.value)} />
            </div>
            <div className="grid gap-2">
              <Label htmlFor="rule-pattern">规则模式</Label>
              <Textarea id="rule-pattern" value={rulePattern} onChange={(e) => setRulePattern(e.target.value)} placeholder="例如: {YYYY}{MM}{DD}-{SEQ:4}" rows={3} />
            </div>
             {preview && (
               <div className="border rounded-lg p-3 bg-muted">
                 <p className="text-sm font-medium mb-2">预览示例：</p>
                 <p className="font-mono text-sm">{preview.sample_code}</p>
               </div>
             )}
            <Button variant="outline" onClick={handlePreview} className="w-full">
              <Eye className="w-4 h-4 mr-2" />
              预览
            </Button>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setShowDialog(false)}>
              取消
            </Button>
            <Button onClick={handleSave}>保存</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </Card>
  );
}

// 存储配置 Tab

export function SystemSettings() {
  const { user } = useAuth();
  const router = useRouter();
  const [activeGroup, setActiveGroup] = useState("基础配置");
  const [activeSubTab, setActiveSubTab] = useState("announcements");

  // 权限校验
  useEffect(() => {
    if (user && !["admin", "super_admin"].includes(user.role)) {
      toast.error("无权限访问系统设置");
      router.push("/");
    }
  }, [user, router]);

  // 切换一级分组时，自动选中该分组的第一个子 tab
  const handleGroupChange = (group: string) => {
    setActiveGroup(group);
    const groupData = SETTINGS_TAB_GROUPS.find((g) => g.group === group);
    if (groupData && groupData.items.length > 0) {
      setActiveSubTab(groupData.items[0].id);
    }
  };

  // 渲染对应的 Tab 内容
  const renderTabContent = () => {
    switch (activeSubTab) {
      case "ai":
        return <ModelSettings />;
      case "notification":
        return <SMTPConfigTab />;
      case "model-usage":
        return <ModelUsageTab />;
      case "archive-classification":
        return <ArchiveClassificationTab />;
      case "archive-global":
        return <ArchiveGlobalTab />;
      case "retention-periods":
        return <RetentionPeriodsTab />;
      case "storage-locations":
        return <StorageLocationsTab />;
      case "code-rules":
        return <CodeRulesTab />;
      case "roles":
        return <RolePermissionTab />;
      case "announcements":
        return <AnnouncementTab />;
      case "logs":
        return <SystemLogs />;
      case "maintenance":
        return <SystemMaintenanceTab />;
      case "storage":
        return <StorageTab />;
      default:
        return null;
    }
  };

  if (!user) {
    return null;
  }

  return (
    <div className="mx-auto flex w-full max-w-none flex-col gap-6 p-6 pb-16 bg-card text-foreground min-h-[calc(100vh-4rem)]">
      <header className="flex flex-col gap-2">
        <div className="flex flex-wrap items-center justify-between gap-4">
          <div>
            <h1 className="text-3xl font-bold tracking-tight">系统设置</h1>
            <p className="text-muted-foreground">配置系统参数、管理用户角色和权限</p>
          </div>
        </div>
      </header>

      <div className="flex gap-6">
        <div className="w-56 shrink-0 space-y-1">
          {SETTINGS_TAB_GROUPS.map((group) => (
            <div key={group.group} className="mb-4">
              <div className="px-3 py-1.5 text-base font-semibold text-foreground">{group.group}</div>
              <div className="space-y-0.5 mt-1">
                {group.items.map((item) => (
                  <button
                    key={item.id}
                    onClick={() => {
                      setActiveGroup(group.group);
                      setActiveSubTab(item.id);
                    }}
                    className={cn(
                      "w-full text-left px-3 py-1.5 text-sm rounded-md transition-all duration-200",
                      activeSubTab === item.id ? "bg-primary/10 text-primary font-medium" : "text-muted-foreground hover:bg-accent/50 hover:text-foreground"
                    )}
                  >
                    {item.label}
                  </button>
                ))}
              </div>
            </div>
          ))}
        </div>

        <div className="flex-1 min-w-0 animate-in fade-in slide-in-from-bottom-2 duration-300">{renderTabContent()}</div>
      </div>
    </div>
  );
}
