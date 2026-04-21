"use client";

import { useState, useEffect } from "react";
import { toast } from "sonner";
import { 
  Dialog, 
  DialogContent,
  DialogHeader,
  DialogTitle
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Switch } from "@/components/ui/switch";
import { 
  Select, 
  SelectContent, 
  SelectItem, 
  SelectTrigger, 
  SelectValue 
} from "@/components/ui/select";
import { 
  Table, 
  TableBody, 
  TableCell, 
  TableHead, 
  TableHeader, 
  TableRow 
} from "@/components/ui/table";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { 
  AlertDialog, 
  AlertDialogAction, 
  AlertDialogCancel, 
  AlertDialogContent, 
  AlertDialogDescription, 
  AlertDialogFooter, 
  AlertDialogHeader, 
  AlertDialogTitle 
} from "@/components/ui/alert-dialog";
import { 
  fetchAlertRules, 
  createAlertRule, 
  updateAlertRule, 
  deleteAlertRule,
  listNotificationConfigs,
  type AlertRule 
} from "@/lib/api";
import { Bell, ShieldAlert, Plus, List, Trash2, Settings2, Zap } from "lucide-react";

interface AlertSettingsDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

const templates = [
  {
    name: "严重错误告警",
    keywords: "error,fatal,critical",
    threshold: 1,
    time_window: 10,
    channel: "站内信+邮件"
  },
  {
    name: "频繁登录失败",
    keywords: "login failed,invalid password,auth failed",
    threshold: 5,
    time_window: 5,
    channel: "站内信"
  },
  {
    name: "系统异常监测",
    keywords: "exception,unhandled,panic",
    threshold: 3,
    time_window: 30,
    channel: "站内信+邮件"
  },
  {
    name: "由于限流导致的请求失败",
    keywords: "rate limit,too many requests",
    threshold: 10,
    time_window: 1,
    channel: "站内信"
  }
];

export function AlertSettingsDialog({ open, onOpenChange }: AlertSettingsDialogProps) {
  const [rules, setRules] = useState<AlertRule[]>([]);
  const [loading, setLoading] = useState(false);
  const [activeTab, setActiveTab] = useState("existing");
  const [availableChannels, setAvailableChannels] = useState<string[]>(["站内信"]);
  
  const [ruleToDelete, setRuleToDelete] = useState<number | null>(null);
  const [showDeleteDialog, setShowDeleteDialog] = useState(false);

  const [newRuleName, setNewRuleName] = useState("");
  const [newKeywords, setNewKeywords] = useState("");
  const [newThreshold, setNewThreshold] = useState("5");
  const [newTimeWindow, setNewTimeWindow] = useState("60");
  const [newChannel, setNewChannel] = useState("站内信");

  useEffect(() => {
    if (open) {
      loadRules();
    }
  }, [open]);

  const loadRules = async () => {
    setLoading(true);
    try {
      const [rulesRaw, configsData] = await Promise.all([
        fetchAlertRules(),
        listNotificationConfigs()
      ]);
      const rulesData = Array.isArray(rulesRaw)
        ? rulesRaw
        : (rulesRaw as { data?: AlertRule[] })?.data ?? [];
      setRules(rulesData);
      
      const channels = ["站内信"];
      const smtpEnabled = configsData.some(c => c.channel === "smtp" && c.enabled);
      const smsEnabled = configsData.some(c => c.channel === "sms" && c.enabled);
      
      if (smtpEnabled) channels.push("站内信+邮件");
      if (smsEnabled) channels.push("站内信+短信");
      if (smtpEnabled && smsEnabled) channels.push("全渠道");
      
      setAvailableChannels(channels);
    } catch (error) {
      console.error("加载告警配置失败:", error);
    } finally {
      setLoading(false);
    }
  };

  const handleCreateRule = async () => {
    if (!newRuleName.trim()) {
      toast.error("请输入规则名称");
      return;
    }
    
    if (rules.some(r => r.name === newRuleName.trim())) {
      toast.error(`已存在名为 "${newRuleName.trim()}" 的规则`);
      return;
    }

    if (!newKeywords.trim()) {
      toast.error("请输入关键词");
      return;
    }

    try {
      await createAlertRule({
        name: newRuleName.trim(),
        keywords: newKeywords.split(",").map(k => k.trim()).filter(Boolean),
        threshold: parseInt(newThreshold) || 5,
        time_window: parseInt(newTimeWindow) || 60,
        enabled: true,
        notification_channel: newChannel,
      });
      toast.success("规则创建成功");
      resetForm();
      loadRules();
      setActiveTab("existing");
    } catch (error) {
      console.error("创建规则失败:", error);
      toast.error("创建规则失败");
    }
  };

  const resetForm = () => {
    setNewRuleName("");
    setNewKeywords("");
    setNewThreshold("5");
    setNewTimeWindow("60");
    setNewChannel("站内信");
  };

  const applyTemplate = async (template: typeof templates[0]) => {
    if (rules.some(r => r.name === template.name)) {
      toast.error(`已存在名为 "${template.name}" 的规则`);
      return;
    }

    try {
      await createAlertRule({
        name: template.name,
        keywords: template.keywords.split(",").map(k => k.trim()).filter(Boolean),
        threshold: template.threshold,
        time_window: template.time_window,
        enabled: true,
        notification_channel: template.channel,
      });
      toast.success(`模板 "${template.name}" 应用成功`);
      loadRules();
      setActiveTab("existing");
    } catch (error) {
      console.error("应用模板失败:", error);
      toast.error("应用模板失败");
    }
  };

  const handleToggleRule = async (rule: AlertRule) => {
    try {
      await updateAlertRule(rule.id, { enabled: !rule.enabled });
      loadRules();
    } catch (error) {
      console.error("更新规则失败:", error);
      toast.error("更新规则失败");
    }
  };

  const confirmDeleteRule = (id: number) => {
    setRuleToDelete(id);
    setShowDeleteDialog(true);
  };

  const handleDeleteRule = async () => {
    if (ruleToDelete === null) return;
    try {
      await deleteAlertRule(ruleToDelete);
      toast.success("规则已删除");
      loadRules();
    } catch (error) {
      console.error("删除规则失败:", error);
      toast.error("删除规则失败");
    } finally {
      setRuleToDelete(null);
      setShowDeleteDialog(false);
    }
  };

  return (
    <>
      <Dialog open={open} onOpenChange={onOpenChange}>
        <DialogContent 
          className="max-w-none sm:max-w-[1000px] w-[90vw] max-h-[90vh] flex flex-col overflow-hidden p-0 bg-transparent border-none shadow-none" 
          style={{ maxWidth: 'min(90vw, 1000px)' }}
        >
          <DialogHeader className="sr-only">
            <DialogTitle>告警设置</DialogTitle>
          </DialogHeader>
          <Card className="flex-1 flex flex-col overflow-hidden shadow-2xl border-none">
            <CardHeader className="border-b bg-muted/30 pb-4">
              <div className="flex items-center justify-between">
                <div className="flex items-center gap-2">
                  <div className="p-2 bg-orange-100 rounded-lg">
                    <ShieldAlert className="w-5 h-5 text-orange-600" />
                  </div>
                  <div>
                    <CardTitle className="text-xl">告警策略配置</CardTitle>
                    <CardDescription>配置日志关键词匹配规则，当异常发生时及时发送通知</CardDescription>
                  </div>
                </div>
                <Button variant="ghost" size="icon" onClick={() => onOpenChange(false)}>
                </Button>
              </div>
            </CardHeader>


            <CardContent className="flex-1 overflow-hidden p-0 flex flex-col">
              <Tabs value={activeTab} onValueChange={setActiveTab} className="flex-1 flex flex-col">
                <div className="px-6 py-2 border-b bg-muted/10">
                  <TabsList className="grid w-full max-w-[400px] grid-cols-2">
                    <TabsTrigger value="existing" className="flex items-center gap-2">
                      <List className="w-4 h-4" />
                      已配置规则
                    </TabsTrigger>
                    <TabsTrigger value="new" className="flex items-center gap-2">
                      <Plus className="w-4 h-4" />
                      新建规则
                    </TabsTrigger>
                  </TabsList>
                </div>

                <div className="flex-1 overflow-y-auto p-6">
                  <TabsContent value="existing" className="m-0 space-y-4">
                    {loading ? (
                      <div className="flex flex-col items-center justify-center py-20 text-muted-foreground">
                        <Settings2 className="w-10 h-10 animate-spin mb-4 opacity-20" />
                        <p>加载配置中...</p>
                      </div>
                    ) : rules.length === 0 ? (
                      <div className="flex flex-col items-center justify-center py-20 border-2 border-dashed rounded-xl bg-muted/5">
                        <Bell className="w-12 h-12 text-muted-foreground/30 mb-4" />
                        <h3 className="text-lg font-medium text-muted-foreground">暂无告警规则</h3>
                        <p className="text-sm text-muted-foreground mb-6">您可以点击上方“新建规则”或从右侧选择模板快速开始</p>
                        <Button onClick={() => setActiveTab("new")} variant="outline">
                          立刻创建
                        </Button>
                      </div>
                    ) : (
                      <div className="border rounded-xl overflow-hidden shadow-sm bg-card">
                        <Table>
                          <TableHeader className="bg-muted/50">
                            <TableRow>
                              <TableHead className="w-[200px]">规则名称</TableHead>
                              <TableHead>关键词</TableHead>
                              <TableHead className="w-[150px]">阈值/窗口</TableHead>
                              <TableHead className="w-[150px]">通知渠道</TableHead>
                              <TableHead className="w-[80px]">状态</TableHead>
                              <TableHead className="w-[80px] text-right">操作</TableHead>
                            </TableRow>
                          </TableHeader>
                          <TableBody>
                            {rules.map((rule) => (
                              <TableRow key={rule.id} className="hover:bg-muted/30 transition-colors">
                                <TableCell className="font-semibold">{rule.name}</TableCell>
                                <TableCell>
                                  <div className="flex flex-wrap gap-1">
                                    {rule.keywords.map((k, i) => (
                                      <span key={i} className="px-2 py-0.5 bg-orange-50 text-orange-700 text-xs rounded border border-orange-100">
                                        {k}
                                      </span>
                                    ))}
                                  </div>
                                </TableCell>
                                <TableCell className="text-sm">
                                  <span className="font-medium text-orange-600">{rule.threshold}</span> 次 / 
                                  <span className="font-medium"> {rule.time_window}</span> 分钟
                                </TableCell>
                                <TableCell className="text-sm">{rule.notification_channel}</TableCell>
                                <TableCell>
                                  <Switch 
                                    checked={rule.enabled} 
                                    onCheckedChange={() => handleToggleRule(rule)}
                                    className="data-[state=checked]:bg-orange-500"
                                  />
                                </TableCell>
                                <TableCell className="text-right">
                                  <Button 
                                    variant="ghost" 
                                    size="sm"
                                    onClick={() => confirmDeleteRule(rule.id)}
                                    className="text-destructive hover:text-destructive hover:bg-destructive/10"
                                  >
                                    <Trash2 className="w-4 h-4" />
                                  </Button>
                                </TableCell>
                              </TableRow>
                            ))}
                          </TableBody>
                        </Table>
                      </div>
                    )}
                  </TabsContent>

                  <TabsContent value="new" className="m-0">
                    <div className="grid grid-cols-1 lg:grid-cols-3 gap-8">
                      <div className="lg:col-span-2 space-y-6">
                        <div className="bg-muted/20 p-6 rounded-xl border space-y-4">
                          <h3 className="text-lg font-semibold flex items-center gap-2">
                            <Plus className="w-5 h-5 text-orange-500" />
                            自定义规则内容
                          </h3>
                          
                          <div className="grid grid-cols-1 sm:grid-cols-2 gap-6">
                            <div className="space-y-2">
                              <Label className="text-sm font-medium">规则名称</Label>
                              <Input 
                                placeholder="如：系统关键错误实时告警" 
                                value={newRuleName}
                                onChange={(e) => setNewRuleName(e.target.value)}
                                className="bg-background"
                              />
                            </div>
                            
                            <div className="space-y-2">
                              <Label className="text-sm font-medium">通知渠道</Label>
                              <Select value={newChannel} onValueChange={setNewChannel}>
                                <SelectTrigger className="bg-background">
                                  <SelectValue />
                                </SelectTrigger>
                                <SelectContent>
                                  {availableChannels.map(c => (
                                    <SelectItem key={c} value={c}>{c}</SelectItem>
                                  ))}
                                </SelectContent>
                              </Select>
                            </div>
                            
                            <div className="space-y-2 sm:col-span-2">
                              <Label className="text-sm font-medium">关键词匹配 (英文逗号分隔)</Label>
                              <Input 
                                placeholder="如：error, failed, CRITICAL, timeout" 
                                value={newKeywords}
                                onChange={(e) => setNewKeywords(e.target.value)}
                                className="bg-background"
                              />
                              <p className="text-[11px] text-muted-foreground">系统将监控日志中包含以上任意关键词的条目</p>
                            </div>
                            
                            <div className="space-y-2">
                              <Label className="text-sm font-medium">触发阈值 (连续出现次数)</Label>
                              <div className="relative">
                                <Input 
                                  type="number"
                                  placeholder="5" 
                                  value={newThreshold}
                                  onChange={(e) => setNewThreshold(e.target.value)}
                                  className="bg-background pr-10"
                                />
                                <span className="absolute right-3 top-1/2 -translate-y-1/2 text-xs text-muted-foreground">次</span>
                              </div>
                            </div>
                            
                            <div className="space-y-2">
                              <Label className="text-sm font-medium">检测窗口 (分钟)</Label>
                              <div className="relative">
                                <Input 
                                  type="number"
                                  placeholder="60" 
                                  value={newTimeWindow}
                                  onChange={(e) => setNewTimeWindow(e.target.value)}
                                  className="bg-background pr-10"
                                />
                                <span className="absolute right-3 top-1/2 -translate-y-1/2 text-xs text-muted-foreground">分钟</span>
                              </div>
                            </div>
                          </div>
                          
                          <div className="pt-4 flex justify-end gap-3">
                            <Button variant="ghost" onClick={resetForm}>重置表单</Button>
                            <Button onClick={handleCreateRule} className="bg-orange-500 hover:bg-orange-600 text-white px-8">
                              保存并启用规则
                            </Button>
                          </div>
                        </div>
                      </div>

                      <div className="space-y-4">
                        <div className="flex items-center gap-2 mb-2">
                          <Zap className="w-4 h-4 text-yellow-500 fill-yellow-500" />
                          <h4 className="font-semibold">预置告警模板</h4>
                        </div>
                        <div className="grid grid-cols-1 gap-3">
                          {templates.map((tpl, i) => (
                            <button
                              key={i}
                              onClick={() => applyTemplate(tpl)}
                              className="text-left p-4 rounded-xl border hover:border-orange-200 hover:bg-orange-50/30 transition-all group relative overflow-hidden"
                            >
                              <div className="font-medium text-sm group-hover:text-orange-600 transition-colors mb-1">
                                {tpl.name}
                              </div>
                              <div className="text-[11px] text-muted-foreground line-clamp-2">
                                关键词: {tpl.keywords}
                              </div>
                              <div className="mt-2 flex items-center gap-3 text-[10px] text-muted-foreground">
                                <span className="flex items-center gap-1">
                                  <Settings2 className="w-3 h-3" />
                                  {tpl.threshold}次/{tpl.time_window}分
                                </span>
                                <span className="px-1.5 py-0.5 bg-muted rounded">
                                  {tpl.channel}
                                </span>
                              </div>
                              <div className="absolute top-1/2 -right-2 -translate-y-1/2 opacity-0 group-hover:opacity-10 group-hover:right-2 transition-all">
                                <Plus className="w-8 h-8 text-orange-500" />
                              </div>
                            </button>
                          ))}
                        </div>
                      </div>
                    </div>
                  </TabsContent>
                </div>
              </Tabs>
            </CardContent>
          </Card>
        </DialogContent>
      </Dialog>

      <AlertDialog open={showDeleteDialog} onOpenChange={setShowDeleteDialog}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>确认删除告警规则？</AlertDialogTitle>
            <AlertDialogDescription>
              删除后该规则将失效，系统将不再监控对应的关键词。此操作不可撤销。
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>取消</AlertDialogCancel>
            <AlertDialogAction onClick={handleDeleteRule} className="bg-destructive text-destructive-foreground hover:bg-destructive/90">
              确认删除
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </>
  );
}
