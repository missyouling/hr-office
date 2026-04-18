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
import { 
  fetchAlertRules, 
  createAlertRule, 
  updateAlertRule, 
  deleteAlertRule,
  type AlertRule 
} from "@/lib/api";

interface AlertSettingsDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

export function AlertSettingsDialog({ open, onOpenChange }: AlertSettingsDialogProps) {
  const [rules, setRules] = useState<AlertRule[]>([]);
  const [loading, setLoading] = useState(false);

  // New rule form
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
      const data = await fetchAlertRules();
      setRules(data);
    } catch (error) {
      console.error("加载告警规则失败:", error);
    } finally {
      setLoading(false);
    }
  };

  const handleCreateRule = async () => {
    if (!newRuleName.trim()) {
      toast.error("请输入规则名称");
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
      setNewRuleName("");
      setNewKeywords("");
      setNewThreshold("5");
      setNewTimeWindow("60");
      loadRules();
    } catch (error) {
      console.error("创建规则失败:", error);
      toast.error("创建规则失败");
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

  const handleDeleteRule = async (id: number) => {
    if (!confirm("确定要删除这条规则吗？")) return;
    try {
      await deleteAlertRule(id);
      toast.success("规则已删除");
      loadRules();
    } catch (error) {
      console.error("删除规则失败:", error);
      toast.error("删除规则失败");
    }
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-2xl max-h-[80vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle>告警设置</DialogTitle>
        </DialogHeader>
        
        <div className="space-y-6">
          {/* Create new rule form */}
          <div className="border rounded-lg p-4 space-y-4">
            <h3 className="font-medium">新建告警规则</h3>
            
            <div className="grid grid-cols-2 gap-4">
              <div className="space-y-2">
                <Label>规则名称</Label>
                <Input 
                  placeholder="如：登录失败告警" 
                  value={newRuleName}
                  onChange={(e) => setNewRuleName(e.target.value)}
                />
              </div>
              
              <div className="space-y-2">
                <Label>通知渠道</Label>
                <Select value={newChannel} onValueChange={setNewChannel}>
                  <SelectTrigger>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="站内信">站内信</SelectItem>
                    <SelectItem value="邮件">邮件</SelectItem>
                    <SelectItem value="SMS">短信</SelectItem>
                  </SelectContent>
                </Select>
              </div>
              
              <div className="space-y-2">
                <Label>关键词（逗号分隔）</Label>
                <Input 
                  placeholder="如：error,failed,exception" 
                  value={newKeywords}
                  onChange={(e) => setNewKeywords(e.target.value)}
                />
              </div>
              
              <div className="space-y-2">
                <Label>触发阈值（次数）</Label>
                <Input 
                  type="number"
                  placeholder="5" 
                  value={newThreshold}
                  onChange={(e) => setNewThreshold(e.target.value)}
                />
              </div>
              
              <div className="space-y-2">
                <Label>时间窗口（分钟）</Label>
                <Input 
                  type="number"
                  placeholder="60" 
                  value={newTimeWindow}
                  onChange={(e) => setNewTimeWindow(e.target.value)}
                />
              </div>
            </div>
            
            <div className="flex justify-end">
              <Button onClick={handleCreateRule}>创建规则</Button>
            </div>
          </div>
          
          {/* Existing rules list */}
          <div className="border rounded-lg p-4">
            <h3 className="font-medium mb-4">已配置的告警规则</h3>
            
            {loading ? (
              <p className="text-sm text-muted-foreground">加载中...</p>
            ) : rules.length === 0 ? (
              <p className="text-sm text-muted-foreground">暂无告警规则</p>
            ) : (
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>规则名称</TableHead>
                    <TableHead>关键词</TableHead>
                    <TableHead>阈值/时间窗口</TableHead>
                    <TableHead>通知渠道</TableHead>
                    <TableHead>状态</TableHead>
                    <TableHead>操作</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {rules && Array.isArray(rules) && rules.length > 0 ? rules.map((rule) => (
                    <TableRow key={rule.id}>
                      <TableCell className="font-medium">{rule.name}</TableCell>
                      <TableCell className="text-sm">
                        {rule.keywords.join(", ")}
                      </TableCell>
                      <TableCell className="text-sm">
                        {rule.threshold}次/{rule.time_window}分钟
                      </TableCell>
                      <TableCell>{rule.notification_channel}</TableCell>
                      <TableCell>
                        <Switch 
                          checked={rule.enabled} 
                          onCheckedChange={() => handleToggleRule(rule)}
                        />
                      </TableCell>
                      <TableCell>
                        <Button 
                          variant="ghost" 
                          size="sm"
                          onClick={() => handleDeleteRule(rule.id)}
                          className="text-red-500 hover:text-red-600"
                        >
                          删除
                        </Button>
                      </TableCell>
                    </TableRow>
                  )) : (
                    <TableRow>
                      <TableCell colSpan={5} className="h-24 text-center text-muted-foreground">
                        暂无告警规则
                      </TableCell>
                    </TableRow>
                  )}
                </TableBody>
              </Table>
            )}
          </div>
        </div>
      </DialogContent>
    </Dialog>
  );
}
