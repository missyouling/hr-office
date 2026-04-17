"use client";

import { useState, useEffect, Fragment } from "react";
import { useRouter } from "next/navigation";
import { toast } from "sonner";
import { useAuth } from "@/lib/auth";
import { getAuditLogs, fetchRoles, fetchPermissions, fetchRolePermissions, updateRolePermissions, updateRole, deleteRole, type Role, type Permission, fetchAnnouncements, createAnnouncement, updateAnnouncement, deleteAnnouncement, type Announcement, fetchDocumentCategories, fetchFieldDefinitions, createFieldDefinition, updateFieldDefinition, deleteFieldDefinition, type ArchiveFieldDefinition, type DocumentCategory, fetchRetentionPeriods, createRetentionPeriod, updateRetentionPeriod, deleteRetentionPeriod, type RetentionPeriod, fetchStorageLocations, createStorageLocation, updateStorageLocation, deleteStorageLocation, type StorageLocation, fetchCodeRules, createCodeRule, updateCodeRule, deleteCodeRule, getCodeRulePreview, type CodeRule, type CodeRulePreview, updateCategoryCode, createCategoryCode, deleteCategory, listStorageConfigs, createStorageConfig, updateStorageConfig, deleteStorageConfig, testStorageConnection, uploadStorageFile, listStorageFiles, deleteStorageFile, getStorageFileDownloadUrl, listNotificationConfigs, createNotificationConfig, updateNotificationConfig, testNotification, type NotificationConfig } from "@/lib/api";
import type { AuditLog, StorageConfig, SysFile } from "@/lib/types";
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
import { RefreshCw, Download, Eye, Plus, Trash2, Edit, Upload, HardDrive, Cloud, Server } from "lucide-react";
import { format } from "date-fns";
import { ModelSettings } from "./model-settings";

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
      { id: "archive-categories", label: "档案分类管理" },
      { id: "archive-fields", label: "档案字段" },
      { id: "retention-periods", label: "保管期限" },
      { id: "storage-locations", label: "存档地点" },
      { id: "code-rules", label: "编码规则" },
    ],
  },
  {
    group: "权限配置",
    icon: "👥",
    items: [
      { id: "roles", label: "角色权限" },
    ],
  },
  {
    group: "存储管理",
    icon: "💾",
    items: [
      { id: "storage-configs", label: "存储配置" },
      { id: "storage-files", label: "文件管理" },
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
        if (c.channel === "smtp") setSmtpConfig({ enabled: c.enabled, host: c.config?.host || "", port: c.config?.port || "587", username: c.config?.username || "", password: c.config?.password || "", from: c.config?.from || "", from_name: c.config?.from_name || "人事系统", reply_to: c.config?.reply_to || "", encryption: c.config?.encryption || "ssl", server_name: c.config?.server_name || "" });
        if (c.channel === "sms") setSmsConfig({ enabled: c.enabled, access_key_id: c.config?.access_key_id || "", access_key_secret: c.config?.access_key_secret || "", sign_name: c.config?.sign_name || "", template_code: c.config?.template_code || "" });
        if (c.channel === "telegram") setTelegramConfig({ enabled: c.enabled, bot_token: c.config?.bot_token || "", chat_id: c.config?.chat_id || "" });
        if (c.channel === "webhook") setWebhookConfig({ enabled: c.enabled, url: c.config?.url || "", method: c.config?.method || "POST", auth: c.config?.auth || "" });
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

// ============ 模型使用统计 Tab ============
function ModelUsageTab() {
  const [timeRange, setTimeRange] = useState("today");
  const [customFrom, setCustomFrom] = useState("");
  const [customTo, setCustomTo] = useState("");
  const [loading, setLoading] = useState(false);

  const allData = [
    { model: "Qwen/Qwen3-8B", config_type: "llm", caller: "admin", calls: 342, tokens_in: 128_400, tokens_out: 87_300, avg_latency_ms: 1240, success_rate: 99.1 },
    { model: "Qwen/Qwen3-8B", config_type: "llm", caller: "张伟", calls: 210, tokens_in: 79_200, tokens_out: 53_600, avg_latency_ms: 1380, success_rate: 98.6 },
    { model: "PaddleOCR-VL-1.5", config_type: "ocr", caller: "admin", calls: 890, tokens_in: 0, tokens_out: 0, avg_latency_ms: 420, success_rate: 99.4 },
    { model: "PaddleOCR-VL-1.5", config_type: "ocr", caller: "李娜", calls: 234, tokens_in: 0, tokens_out: 0, avg_latency_ms: 440, success_rate: 98.7 },
    { model: "BAAI/bge-m3", config_type: "embedding", caller: "admin", calls: 567, tokens_in: 234_000, tokens_out: 0, avg_latency_ms: 180, success_rate: 99.8 },
    { model: "BAAI/bge-reranker-v2-m3", config_type: "rerank", caller: "admin", calls: 234, tokens_in: 56_000, tokens_out: 0, avg_latency_ms: 95, success_rate: 97.9 },
  ];

  const usageData = allData;
  const totalTokensIn = usageData.reduce((s, r) => s + r.tokens_in, 0);
  const totalTokensOut = usageData.reduce((s, r) => s + r.tokens_out, 0);
  const totalCalls = usageData.reduce((s, r) => s + r.calls, 0);
  const avgLatency = totalCalls > 0 ? Math.round(usageData.reduce((s, r) => s + r.avg_latency_ms * r.calls, 0) / totalCalls) : 0;

  const byModel = usageData.reduce((acc: Record<string, number>, r) => {
    acc[r.model] = (acc[r.model] ?? 0) + r.calls;
    return acc;
  }, {});

  const chartItems = Object.entries(byModel).sort((a, b) => b[1] - a[1]);
  const maxCalls = chartItems[0]?.[1] ?? 1;

  const handleRefresh = () => {
    setLoading(true);
    setTimeout(() => setLoading(false), 600);
  };

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

      <div className="grid grid-cols-2 sm:grid-cols-4 gap-3">
        <Card className="p-3">
          <div className="text-xs text-muted-foreground mb-1">Token 输入（期间）</div>
          <div className="text-2xl font-bold text-blue-600">{formatNum(totalTokensIn)}</div>
          <div className="text-xs text-muted-foreground mt-0.5">累计 {formatNum(totalTokensIn * 30)}</div>
        </Card>
        <Card className="p-3">
          <div className="text-xs text-muted-foreground mb-1">Token 输出（期间）</div>
          <div className="text-2xl font-bold text-green-600">{formatNum(totalTokensOut)}</div>
          <div className="text-xs text-muted-foreground mt-0.5">累计 {formatNum(totalTokensOut * 30)}</div>
        </Card>
        <Card className="p-3">
          <div className="text-xs text-muted-foreground mb-1">总调用次数</div>
          <div className="text-2xl font-bold text-purple-600">{totalCalls.toLocaleString()}</div>
          <div className="text-xs text-muted-foreground mt-0.5">{usageData.length} 条记录</div>
        </Card>
        <Card className="p-3">
          <div className="text-xs text-muted-foreground mb-1">平均响应时间</div>
          <div className="text-2xl font-bold text-orange-600">{avgLatency}ms</div>
          <div className="text-xs text-muted-foreground mt-0.5">加权均值</div>
        </Card>
      </div>

      <Card>
        <CardHeader className="pb-2">
          <CardTitle className="text-sm">调用量分布</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="space-y-2">
            {chartItems.map(([model, calls]) => (
              <div key={model} className="flex items-center gap-2 text-xs">
                <div className="w-48 truncate text-muted-foreground shrink-0" title={model}>
                  {model}
                </div>
                <div className="flex-1 h-5 bg-muted rounded overflow-hidden">
                  <div className="h-full bg-primary rounded transition-all duration-500" style={{ width: `${Math.max(2, (calls / maxCalls) * 100)}%` }} />
                </div>
                <div className="w-12 text-right font-medium shrink-0">{formatNum(calls)}</div>
              </div>
            ))}
          </div>
        </CardContent>
      </Card>

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
                    <TableCell className="font-medium max-w-[160px] truncate" title={item.model}>
                      {item.model}
                    </TableCell>
                    <TableCell>
                      <Badge variant="outline" className="text-xs">
                        {item.config_type.toUpperCase()}
                      </Badge>
                    </TableCell>
                    <TableCell>{item.caller}</TableCell>
                    <TableCell className="text-right">{item.calls.toLocaleString()}</TableCell>
                    <TableCell className="text-right">{formatNum(item.tokens_in)}</TableCell>
                    <TableCell className="text-right">{formatNum(item.tokens_out)}</TableCell>
                    <TableCell className="text-right">{item.avg_latency_ms}ms</TableCell>
                    <TableCell className="text-right">{item.success_rate}%</TableCell>
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

// ============ 系统日志 Tab ============
function SystemLogsTab() {
  const [logs, setLogs] = useState<AuditLog[]>([]);
  const [loading, setLoading] = useState(false);
  const [filterUser, setFilterUser] = useState("");
  const [filterAction, setFilterAction] = useState("");

  useEffect(() => {
    loadLogs();
  }, []);

  const loadLogs = async () => {
    setLoading(true);
    try {
      const { logs: logsData } = await getAuditLogs();
      setLogs(logsData);
    } catch (error) {
      console.error("加载日志失败:", error);
      toast.error("加载日志失败");
    } finally {
      setLoading(false);
    }
  };

  const filteredLogs = logs.filter((log) => {
    if (filterUser && !log.username?.includes(filterUser)) return false;
    if (filterAction && !log.action.includes(filterAction)) return false;
    return true;
  });

  return (
    <Card>
      <CardHeader>
        <CardTitle>系统日志</CardTitle>
        <CardDescription>查看系统操作审计日志</CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        <div className="flex gap-2">
          <Input placeholder="筛选用户..." value={filterUser} onChange={(e) => setFilterUser(e.target.value)} className="max-w-xs" />
          <Input placeholder="筛选操作..." value={filterAction} onChange={(e) => setFilterAction(e.target.value)} className="max-w-xs" />
          <Button variant="outline" onClick={loadLogs} disabled={loading}>
            <RefreshCw className={cn("w-4 h-4", loading && "animate-spin")} />
          </Button>
        </div>

        {loading ? (
          <div className="text-center py-8 text-muted-foreground">加载中...</div>
        ) : (
          <div className="overflow-x-auto">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>用户</TableHead>
                  <TableHead>操作</TableHead>
                  <TableHead>资源</TableHead>
                  <TableHead>状态</TableHead>
                  <TableHead>时间</TableHead>
                </TableRow>
              </TableHeader>
               <TableBody>
                 {filteredLogs.map((log) => (
                   <TableRow key={log.id}>
                     <TableCell>{log.username}</TableCell>
                     <TableCell>{log.action}</TableCell>
                     <TableCell className="max-w-xs truncate">{log.resource_type}</TableCell>
                     <TableCell>
                       <Badge variant={log.status === "SUCCESS" ? "default" : "destructive"}>{log.status}</Badge>
                     </TableCell>
                     <TableCell>{format(new Date(log.timestamp), "yyyy-MM-dd HH:mm:ss")}</TableCell>
                   </TableRow>
                 ))}
               </TableBody>
            </Table>
          </div>
        )}
      </CardContent>
    </Card>
  );
}

// ============ 档案分类管理 Tab ============
function ArchiveCategoriesTab() {
  const [categories, setCategories] = useState<DocumentCategory[]>([]);
  const [loading, setLoading] = useState(false);
  const [showDialog, setShowDialog] = useState(false);
  const [editingCategory, setEditingCategory] = useState<DocumentCategory | null>(null);
  const [categoryName, setCategoryName] = useState("");
  const [categoryCode, setCategoryCode] = useState("");

  useEffect(() => {
    loadCategories();
  }, []);

  const loadCategories = async () => {
    setLoading(true);
    try {
      const data = await fetchDocumentCategories();
      setCategories(data);
    } catch (error) {
      console.error("加载分类失败:", error);
      toast.error("加载分类失败");
    } finally {
      setLoading(false);
    }
  };

  const handleSave = async () => {
    if (!categoryName.trim() || !categoryCode.trim()) {
      toast.error("分类名称和代码不能为空");
      return;
    }

    try {
      if (editingCategory) {
        await updateCategoryCode(editingCategory.id, { name: categoryName, code: categoryCode });
        toast.success("分类已更新");
      } else {
        await createCategoryCode({ name: categoryName, code: categoryCode });
        toast.success("分类已创建");
      }
      setShowDialog(false);
      setCategoryName("");
      setCategoryCode("");
      loadCategories();
    } catch (error) {
      console.error("保存失败:", error);
      toast.error("保存失败");
    }
  };

  const handleDelete = async (id: number) => {
    if (confirm("确定要删除此分类吗？")) {
      try {
        await deleteCategory(id);
        toast.success("分类已删除");
        loadCategories();
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
            <CardTitle>档案分类管理</CardTitle>
            <CardDescription>管理档案分类和编码</CardDescription>
          </div>
          <Button onClick={() => { setEditingCategory(null); setCategoryName(""); setCategoryCode(""); setShowDialog(true); }}>
            <Plus className="w-4 h-4 mr-2" />
            新增分类
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
                  <TableHead>分类名称</TableHead>
                  <TableHead>代码</TableHead>
                  <TableHead>操作</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {categories.map((cat) => (
                  <TableRow key={cat.id}>
                    <TableCell>{cat.name}</TableCell>
                    <TableCell>{cat.code}</TableCell>
                    <TableCell>
                      <div className="flex gap-2">
                        <Button variant="outline" size="sm" onClick={() => { setEditingCategory(cat); setCategoryName(cat.name); setCategoryCode(cat.code); setShowDialog(true); }}>
                          <Edit className="w-4 h-4" />
                        </Button>
                        <Button variant="outline" size="sm" onClick={() => handleDelete(cat.id)}>
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
            <DialogTitle>{editingCategory ? "编辑分类" : "新增分类"}</DialogTitle>
          </DialogHeader>
          <div className="space-y-4">
            <div className="grid gap-2">
              <Label htmlFor="cat-name">分类名称</Label>
              <Input id="cat-name" value={categoryName} onChange={(e) => setCategoryName(e.target.value)} />
            </div>
            <div className="grid gap-2">
              <Label htmlFor="cat-code">代码</Label>
              <Input id="cat-code" value={categoryCode} onChange={(e) => setCategoryCode(e.target.value)} />
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

// ============ 档案字段管理 Tab ============
function ArchiveFieldsTab() {
  const [fields, setFields] = useState<ArchiveFieldDefinition[]>([]);
  const [loading, setLoading] = useState(false);
  const [showDialog, setShowDialog] = useState(false);
  const [editingField, setEditingField] = useState<ArchiveFieldDefinition | null>(null);
  const [fieldName, setFieldName] = useState("");
  const [fieldType, setFieldType] = useState("text");

  useEffect(() => {
    loadFields();
  }, []);

  const loadFields = async () => {
    setLoading(true);
    try {
      const data = await fetchFieldDefinitions();
      setFields(data);
    } catch (error) {
      console.error("加载字段失败:", error);
      toast.error("加载字段失败");
    } finally {
      setLoading(false);
    }
  };

   const handleSave = async () => {
     if (!fieldName.trim()) {
       toast.error("字段名称不能为空");
       return;
     }

     try {
       if (editingField) {
         await updateFieldDefinition(editingField.id, { field_name: fieldName, field_type: fieldType as "text" | "textarea" | "number" | "date" | "select" | "multiselect" | "checkbox" });
         toast.success("字段已更新");
       } else {
         await createFieldDefinition({ field_name: fieldName, field_type: fieldType as "text" | "textarea" | "number" | "date" | "select" | "multiselect" | "checkbox" });
         toast.success("字段已创建");
       }
       setShowDialog(false);
       setFieldName("");
       setFieldType("text");
       loadFields();
     } catch (error) {
       console.error("保存失败:", error);
       toast.error("保存失败");
     }
   };

   const handleDelete = async (id: number) => {
     if (confirm("确定要删除此字段吗？")) {
       try {
         await deleteFieldDefinition(id);
         toast.success("字段已删除");
         loadFields();
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
            <CardTitle>档案字段</CardTitle>
            <CardDescription>管理档案字段定义</CardDescription>
          </div>
          <Button onClick={() => { setEditingField(null); setFieldName(""); setFieldType("text"); setShowDialog(true); }}>
            <Plus className="w-4 h-4 mr-2" />
            新增字段
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
                  <TableHead>字段名称</TableHead>
                  <TableHead>类型</TableHead>
                  <TableHead>操作</TableHead>
                </TableRow>
              </TableHeader>
               <TableBody>
                 {fields.map((field) => (
                   <TableRow key={field.id}>
                     <TableCell>{field.field_label}</TableCell>
                     <TableCell>{field.field_type}</TableCell>
                     <TableCell>
                       <div className="flex gap-2">
                         <Button variant="outline" size="sm" onClick={() => { setEditingField(field); setFieldName(field.field_label); setFieldType(field.field_type); setShowDialog(true); }}>
                           <Edit className="w-4 h-4" />
                         </Button>
                         <Button variant="outline" size="sm" onClick={() => handleDelete(field.id)}>
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
            <DialogTitle>{editingField ? "编辑字段" : "新增字段"}</DialogTitle>
          </DialogHeader>
          <div className="space-y-4">
            <div className="grid gap-2">
              <Label htmlFor="field-name">字段名称</Label>
              <Input id="field-name" value={fieldName} onChange={(e) => setFieldName(e.target.value)} />
            </div>
            <div className="grid gap-2">
              <Label htmlFor="field-type">类型</Label>
              <Select value={fieldType} onValueChange={setFieldType}>
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="text">文本</SelectItem>
                  <SelectItem value="number">数字</SelectItem>
                  <SelectItem value="date">日期</SelectItem>
                  <SelectItem value="select">下拉选择</SelectItem>
                </SelectContent>
              </Select>
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

// ============ 保管期限管理 Tab ============
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
function StorageConfigTab() {
  const [configs, setConfigs] = useState<StorageConfig[]>([]);
  const [loading, setLoading] = useState(false);
  const [showCreateDialog, setShowCreateDialog] = useState(false);
  const [editingConfig, setEditingConfig] = useState<StorageConfig | null>(null);
  const [testingId, setTestingId] = useState<number | null>(null);

  const [formData, setFormData] = useState({
    name: "",
    type: "local" as "local" | "s3" | "webdav",
    enabled: true,
    config: {} as Record<string, unknown>,
  });

  useEffect(() => {
    const load = async () => {
      setLoading(true);
      try {
        const data = await listStorageConfigs();
        setConfigs(data);
      } catch (error) {
        toast.error("加载存储配置失败");
      } finally {
        setLoading(false);
      }
    };
    load();
  }, []);

  const handleCreate = () => {
    setEditingConfig(null);
    setFormData({ name: "", type: "local", enabled: true, config: {} });
    setShowCreateDialog(true);
  };

  const handleEdit = (config: StorageConfig) => {
    setEditingConfig(config);
    setFormData({
      name: config.name,
      type: config.type as "local" | "s3" | "webdav",
      enabled: config.enabled,
      config: config.config,
    });
    setShowCreateDialog(true);
  };

  const handleSave = async () => {
    if (!formData.name.trim()) {
      toast.error("请输入存储配置名称");
      return;
    }

    try {
      if (editingConfig) {
        await updateStorageConfig(editingConfig.id, formData);
        toast.success("存储配置已更新");
      } else {
        await createStorageConfig(formData);
        toast.success("存储配置已创建");
      }
      setShowCreateDialog(false);
      const data = await listStorageConfigs();
      setConfigs(data);
    } catch (error) {
      toast.error("保存失败");
    }
  };

  const handleDelete = async (id: number) => {
    try {
      await deleteStorageConfig(id);
      toast.success("存储配置已删除");
      const data = await listStorageConfigs();
      setConfigs(data);
    } catch (error) {
      toast.error("删除失败");
    }
  };

  const handleTest = async (config: StorageConfig) => {
    setTestingId(config.id);
    try {
      const result = await testStorageConnection({
        type: config.type,
        config: config.config,
      });
      if (result.success) {
        toast.success(`连接成功 (${result.latency_ms}ms)`);
      } else {
        toast.error(`连接失败: ${result.message}`);
      }
    } catch (error) {
      toast.error("测试连接失败");
    } finally {
      setTestingId(null);
    }
  };

  const getTypeIcon = (type: string) => {
    switch (type) {
      case "s3":
        return <Cloud className="w-4 h-4" />;
      case "webdav":
        return <Server className="w-4 h-4" />;
      default:
        return <HardDrive className="w-4 h-4" />;
    }
  };

  const getTypeLabel = (type: string) => {
    switch (type) {
      case "s3":
        return "S3";
      case "webdav":
        return "WebDAV";
      default:
        return "本地";
    }
  };

  return (
    <Card>
      <CardHeader className="flex flex-row items-center justify-between">
        <div>
          <CardTitle>存储配置</CardTitle>
          <CardDescription>管理文件存储位置和配置</CardDescription>
        </div>
        <Button onClick={handleCreate} size="sm">
          <Plus className="w-4 h-4 mr-2" />
          新增配置
        </Button>
      </CardHeader>
      <CardContent>
        {loading ? (
          <div className="text-center py-8 text-muted-foreground">加载中...</div>
        ) : configs.length === 0 ? (
          <div className="text-center py-8 text-muted-foreground">暂无存储配置</div>
        ) : (
          <div className="grid gap-4 md:grid-cols-2">
            {configs.map((config) => (
              <Card key={config.id} className="border">
                <CardContent className="pt-6">
                  <div className="space-y-3">
                    <div className="flex items-start justify-between">
                      <div className="flex items-center gap-2">
                        {getTypeIcon(config.type)}
                        <div>
                          <p className="font-medium">{config.name}</p>
                          <p className="text-xs text-muted-foreground">{getTypeLabel(config.type)}</p>
                        </div>
                      </div>
                      {config.is_default && <Badge variant="default">默认</Badge>}
                    </div>
                    <div className="flex items-center gap-2">
                      <span className="text-xs text-muted-foreground">状态:</span>
                      <Badge variant={config.enabled ? "outline" : "secondary"}>
                        {config.enabled ? "启用" : "禁用"}
                      </Badge>
                    </div>
                    <div className="flex gap-2 pt-2">
                      <Button
                        variant="outline"
                        size="sm"
                        onClick={() => handleTest(config)}
                        disabled={testingId === config.id}
                      >
                        <RefreshCw className="w-3 h-3 mr-1" />
                        {testingId === config.id ? "测试中" : "测试"}
                      </Button>
                      <Button variant="outline" size="sm" onClick={() => handleEdit(config)}>
                        <Edit className="w-3 h-3 mr-1" />
                        编辑
                      </Button>
                      <Button
                        variant="outline"
                        size="sm"
                        onClick={() => handleDelete(config.id)}
                      >
                        <Trash2 className="w-3 h-3 mr-1" />
                        删除
                      </Button>
                    </div>
                  </div>
                </CardContent>
              </Card>
            ))}
          </div>
        )}
      </CardContent>

      <Dialog open={showCreateDialog} onOpenChange={setShowCreateDialog}>
        <DialogContent className="max-w-md">
          <DialogHeader>
            <DialogTitle>{editingConfig ? "编辑存储配置" : "新增存储配置"}</DialogTitle>
          </DialogHeader>
          <div className="space-y-4">
            <div className="grid gap-2">
              <Label htmlFor="config-name">配置名称</Label>
              <Input
                id="config-name"
                value={formData.name}
                onChange={(e) => setFormData({ ...formData, name: e.target.value })}
                placeholder="例如: 主存储"
              />
            </div>
            <div className="grid gap-2">
              <Label htmlFor="config-type">存储类型</Label>
              <Select value={formData.type} onValueChange={(v) => setFormData({ ...formData, type: v as "local" | "s3" | "webdav" })}>
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="local">本地存储</SelectItem>
                  <SelectItem value="s3">S3 兼容</SelectItem>
                  <SelectItem value="webdav">WebDAV</SelectItem>
                </SelectContent>
              </Select>
            </div>
            <div className="flex items-center gap-2">
              <Switch
                id="config-enabled"
                checked={formData.enabled}
                onCheckedChange={(checked) => setFormData({ ...formData, enabled: checked })}
              />
              <Label htmlFor="config-enabled">启用此配置</Label>
            </div>
            <div className="grid gap-2">
              <Label htmlFor="config-json">配置 JSON</Label>
              <Textarea
                id="config-json"
                value={JSON.stringify(formData.config, null, 2)}
                onChange={(e) => {
                  try {
                    setFormData({ ...formData, config: JSON.parse(e.target.value) });
                  } catch {
                    // 保持原值
                  }
                }}
                placeholder='{"root_path": "/data"}'
                rows={4}
              />
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setShowCreateDialog(false)}>
              取消
            </Button>
            <Button onClick={handleSave}>保存</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </Card>
  );
}

// 文件管理 Tab
function FileManagementTab() {
  const [files, setFiles] = useState<SysFile[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(false);
  const [page, setPage] = useState(0);
  const [configs, setConfigs] = useState<StorageConfig[]>([]);
  const [selectedConfigId, setSelectedConfigId] = useState<number | null>(null);
  const [uploading, setUploading] = useState(false);
  const [deleteConfirmId, setDeleteConfirmId] = useState<number | null>(null);

  const pageSize = 20;

  useEffect(() => {
    const load = async () => {
      try {
        const data = await listStorageConfigs();
        setConfigs(data);
        if (data.length > 0 && selectedConfigId === null) {
          setSelectedConfigId(data[0].id);
        }
      } catch (error) {
        toast.error("加载存储配置失败");
      }
    };
    load();
  }, [selectedConfigId]);

  useEffect(() => {
    const load = async () => {
      setLoading(true);
      try {
        const response = await listStorageFiles({
          storage_config_id: selectedConfigId || undefined,
          limit: pageSize,
          offset: page * pageSize,
        });
        setFiles(response.files);
        setTotal(response.total);
      } catch (error) {
        toast.error("加载文件列表失败");
      } finally {
        setLoading(false);
      }
    };
    load();
  }, [page, selectedConfigId]);

  const handleFileUpload = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file || !selectedConfigId) {
      toast.error("请选择文件和存储配置");
      return;
    }

    setUploading(true);
    try {
      await uploadStorageFile(file, selectedConfigId);
      toast.success("文件上传成功");
      setPage(0);
    } catch (error) {
      toast.error("文件上传失败");
    } finally {
      setUploading(false);
    }
  };

  const handleDownload = (fileId: number) => {
    const url = getStorageFileDownloadUrl(fileId);
    const token = typeof window !== "undefined" ? localStorage.getItem("token") : null;
    const headers = token ? { Authorization: `Bearer ${token}` } : {};
    window.open(url, "_blank");
  };

  const handleDelete = async (fileId: number) => {
    try {
      await deleteStorageFile(fileId);
      toast.success("文件已删除");
      setPage(0);
    } catch (error) {
      toast.error("删除失败");
    } finally {
      setDeleteConfirmId(null);
    }
  };

  const formatFileSize = (bytes: number): string => {
    if (bytes === 0) return "0 B";
    const k = 1024;
    const sizes = ["B", "KB", "MB", "GB"];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return Math.round((bytes / Math.pow(k, i)) * 100) / 100 + " " + sizes[i];
  };

  const totalPages = Math.ceil(total / pageSize);

  return (
    <Card>
      <CardHeader>
        <CardTitle>文件管理</CardTitle>
        <CardDescription>上传和管理存储文件</CardDescription>
      </CardHeader>
      <CardContent className="space-y-6">
        <div className="space-y-4">
          <div className="grid gap-4 md:grid-cols-3 items-end">
            <div className="grid gap-2">
              <Label htmlFor="storage-select">选择存储配置</Label>
              <Select
                value={selectedConfigId?.toString() || ""}
                onValueChange={(v) => {
                  setSelectedConfigId(Number(v));
                  setPage(0);
                }}
              >
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {configs.map((config) => (
                    <SelectItem key={config.id} value={config.id.toString()}>
                      {config.name}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
            <div className="grid gap-2">
              <Label htmlFor="file-input">选择文件</Label>
              <Input
                id="file-input"
                type="file"
                onChange={handleFileUpload}
                disabled={uploading || !selectedConfigId}
              />
            </div>
            <Button disabled={uploading} className="w-full">
              <Upload className="w-4 h-4 mr-2" />
              {uploading ? "上传中..." : "上传"}
            </Button>
          </div>
        </div>

        {loading ? (
          <div className="text-center py-8 text-muted-foreground">加载中...</div>
        ) : files.length === 0 ? (
          <div className="text-center py-8 text-muted-foreground">暂无文件</div>
        ) : (
          <>
            <div className="border rounded-lg overflow-hidden">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>文件名</TableHead>
                    <TableHead>大小</TableHead>
                    <TableHead>类型</TableHead>
                    <TableHead>存储类型</TableHead>
                    <TableHead>创建时间</TableHead>
                    <TableHead className="text-right">操作</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {files.map((file) => (
                    <TableRow key={file.id}>
                      <TableCell className="font-medium">{file.original_name}</TableCell>
                      <TableCell>{formatFileSize(file.size)}</TableCell>
                      <TableCell>{file.content_type}</TableCell>
                      <TableCell>{file.storage_type}</TableCell>
                      <TableCell>{format(new Date(file.created_at), "yyyy-MM-dd HH:mm")}</TableCell>
                      <TableCell className="text-right">
                        <div className="flex justify-end gap-2">
                          <Button
                            variant="outline"
                            size="sm"
                            onClick={() => handleDownload(file.id)}
                          >
                            <Download className="w-3 h-3 mr-1" />
                            下载
                          </Button>
                          <Button
                            variant="outline"
                            size="sm"
                            onClick={() => setDeleteConfirmId(file.id)}
                          >
                            <Trash2 className="w-3 h-3 mr-1" />
                            删除
                          </Button>
                        </div>
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </div>

            {totalPages > 1 && (
              <div className="flex items-center justify-between">
                <p className="text-sm text-muted-foreground">
                  共 {total} 个文件，第 {page + 1} / {totalPages} 页
                </p>
                <div className="flex gap-2">
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => setPage(Math.max(0, page - 1))}
                    disabled={page === 0}
                  >
                    上一页
                  </Button>
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => setPage(Math.min(totalPages - 1, page + 1))}
                    disabled={page === totalPages - 1}
                  >
                    下一页
                  </Button>
                </div>
              </div>
            )}
          </>
        )}
      </CardContent>

      <AlertDialog open={deleteConfirmId !== null} onOpenChange={(open) => !open && setDeleteConfirmId(null)}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>确认删除</AlertDialogTitle>
            <AlertDialogDescription>确定要删除这个文件吗？此操作无法撤销。</AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>取消</AlertDialogCancel>
            <AlertDialogAction onClick={() => deleteConfirmId && handleDelete(deleteConfirmId)}>
              删除
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </Card>
  );
}

// ============ 主组件 ============
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
      case "archive-categories":
        return <ArchiveCategoriesTab />;
      case "archive-fields":
        return <ArchiveFieldsTab />;
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
        return <SystemLogsTab />;
      case "maintenance":
        return <SystemMaintenanceTab />;
      case "storage-configs":
        return <StorageConfigTab />;
      case "storage-files":
        return <FileManagementTab />;
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
