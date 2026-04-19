"use client";

import { useState, useEffect, useCallback } from "react";
import { toast } from "sonner";
import {
  fetchDocumentCategories,
  createCategoryCode,
  updateCategoryCode,
  deleteCategory,
  createSubCategory,
  updateSubCategoryCode,
  deleteSubCategory,
  fetchFieldDefinitions,
  createFieldDefinition,
  updateFieldDefinition,
  deleteFieldDefinition,
  fetchRetentionPeriods,
  createRetentionPeriod,
  updateRetentionPeriod,
  deleteRetentionPeriod,
  fetchStorageLocations,
  createStorageLocation,
  updateStorageLocation,
  deleteStorageLocation,
  fetchCodeRules,
  createCodeRule,
  updateCodeRule,
  deleteCodeRule,
  getCodeRulePreview,
  type DocumentCategory,
  type DocumentSubCategory,
  type ArchiveFieldDefinition,
  type RetentionPeriod,
  type StorageLocation,
  type CodeRule,
  type CodeRulePreview,
} from "@/lib/api";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { Dialog, DialogContent, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Badge } from "@/components/ui/badge";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Plus, Trash2, Edit, Eye, Search, Settings2, ShieldCheck, Database, Sliders } from "lucide-react";
import { Textarea } from "@/components/ui/textarea";
import { Switch } from "@/components/ui/switch";

export function ArchiveConfigTab() {
  const [activeTab, setActiveTab] = useState("classification");

  return (
    <Card className="w-full">
      <CardHeader>
        <CardTitle>档案配置</CardTitle>
        <CardDescription>管理档案分类、字段、保管期限及系统全局规则</CardDescription>
      </CardHeader>
      <CardContent>
        <Tabs value={activeTab} onValueChange={setActiveTab} className="space-y-4">
          <TabsList className="grid w-full grid-cols-3">
            <TabsTrigger value="classification" className="flex items-center gap-2">
              <Settings2 className="w-4 h-4" />
              分类管理
            </TabsTrigger>
            <TabsTrigger value="global" className="flex items-center gap-2">
              <Database className="w-4 h-4" />
              全局配置
            </TabsTrigger>
            <TabsTrigger value="advanced" className="flex items-center gap-2">
              <Sliders className="w-4 h-4" />
              高级选项
            </TabsTrigger>
          </TabsList>

          <TabsContent value="classification" className="space-y-4">
            <ClassificationManagement />
          </TabsContent>

          <TabsContent value="global" className="space-y-4">
            <GlobalConfiguration />
          </TabsContent>

          <TabsContent value="advanced" className="space-y-4">
            <AdvancedOptions />
          </TabsContent>
        </Tabs>
      </CardContent>
    </Card>
  );
}

function ClassificationManagement() {
  const [categories, setCategories] = useState<DocumentCategory[]>([]);
  const [loading, setLoading] = useState(false);
  const [showCategoryModal, setShowCategoryModal] = useState(false);
  const [editingCategory, setEditingCategory] = useState<DocumentCategory | null>(null);

  const loadCategories = useCallback(async () => {
    setLoading(true);
    try {
      const data = await fetchDocumentCategories();
      setCategories(data);
    } catch {
      toast.error("加载分类失败");
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    loadCategories();
  }, [loadCategories]);

  const handleEditCategory = (category: DocumentCategory) => {
    setEditingCategory(category);
    setShowCategoryModal(true);
  };

  const handleAddCategory = () => {
    setEditingCategory(null);
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
              <TableHead>子类数量</TableHead>
              <TableHead className="text-right">操作</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {loading ? (
              <TableRow>
                <TableCell colSpan={4} className="text-center py-8 text-muted-foreground">加载中...</TableCell>
              </TableRow>
            ) : categories.length === 0 ? (
              <TableRow>
                <TableCell colSpan={4} className="text-center py-8 text-muted-foreground">暂无分类数据</TableCell>
              </TableRow>
            ) : (
              categories.map((cat) => (
                <TableRow key={cat.id}>
                  <TableCell className="font-mono">{cat.code}</TableCell>
                  <TableCell className="font-medium">{cat.name}</TableCell>
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
        category={editingCategory}
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
  const [saving, setSaving] = useState(false);
  
  const [showSubModal, setShowSubModal] = useState(false);
  const [editingSub, setEditingSub] = useState<DocumentSubCategory | null>(null);

  useEffect(() => {
    if (category) {
      setName(category.name);
      setCode(category.code);
    } else {
      setName("");
      setCode("");
    }
  }, [category, open]);

  const handleSave = async () => {
    if (!name.trim() || !code.trim()) {
      toast.error("名称和代码不能为空");
      return;
    }
    setSaving(true);
    try {
      if (category) {
        await updateCategoryCode(category.id, { name, code });
        toast.success("更新成功");
      } else {
        await createCategoryCode({ name, code });
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
    setEditingSub(sub);
    setShowSubModal(true);
  };

  const handleAddSub = () => {
    setEditingSub(null);
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
      <DialogContent className="max-w-4xl max-h-[90vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle>{category ? `编辑分类: ${category.name}` : "新增大类"}</DialogTitle>
        </DialogHeader>
        
        <div className="grid gap-6 py-4">
          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-2">
              <Label>分类名称</Label>
              <Input value={name} onChange={(e) => setName(e.target.value)} placeholder="如: 行政档案" />
            </div>
            <div className="space-y-2">
              <Label>分类代码</Label>
              <Input value={code} onChange={(e) => setCode(e.target.value)} placeholder="如: ADMIN" />
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
                      <TableHead className="text-right">操作</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {!category.sub_categories || category.sub_categories.length === 0 ? (
                      <TableRow>
                        <TableCell colSpan={3} className="text-center py-4 text-muted-foreground text-sm">暂无子类</TableCell>
                      </TableRow>
                    ) : (
                      category.sub_categories.map((sub) => (
                        <TableRow key={sub.id}>
                          <TableCell className="font-mono text-xs">{sub.code}</TableCell>
                          <TableCell className="text-sm font-medium">{sub.name}</TableCell>
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
            subCategory={editingSub}
            onSaved={onSaved}
          />
        )}
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
  const [saving, setSaving] = useState(false);
  
  const [fields, setFields] = useState<ArchiveFieldDefinition[]>([]);
  const [loadingFields, setLoadingFields] = useState(false);
  const [showFieldModal, setShowFieldModal] = useState(false);
  const [editingField, setEditingField] = useState<ArchiveFieldDefinition | null>(null);

  const loadFields = useCallback(async () => {
    if (!subCategory) return;
    setLoadingFields(true);
    try {
      const data = await fetchFieldDefinitions(subCategory.id);
      setFields(data);
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
      loadFields();
    } else {
      setName("");
      setCode("");
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
        await updateSubCategoryCode(subCategory.id, { code, name });
        toast.success("更新成功");
      } else {
        await createSubCategory({ category_id: categoryId, name, code });
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
    setEditingField(field);
    setShowFieldModal(true);
  };

  const handleAddField = () => {
    setEditingField(null);
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
      <DialogContent className="max-w-3xl max-h-[85vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle>{subCategory ? `编辑子类: ${subCategory.name}` : "新增子类"}</DialogTitle>
        </DialogHeader>

        <div className="grid gap-6 py-4">
          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-2">
              <Label>子类名称</Label>
              <Input value={name} onChange={(e) => setName(e.target.value)} placeholder="如: 劳动合同" />
            </div>
            <div className="space-y-2">
              <Label>子类代码</Label>
              <Input value={code} onChange={(e) => setCode(e.target.value)} placeholder="如: CONTRACT" />
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
                      <TableHead>类型</TableHead>
                      <TableHead>必填</TableHead>
                      <TableHead className="text-right">操作</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {loadingFields ? (
                      <TableRow>
                        <TableCell colSpan={4} className="text-center py-4 text-muted-foreground">加载中...</TableCell>
                      </TableRow>
                    ) : fields.length === 0 ? (
                      <TableRow>
                        <TableCell colSpan={4} className="text-center py-4 text-muted-foreground text-sm">暂无字段</TableCell>
                      </TableRow>
                    ) : (
                      fields.map((field) => (
                        <TableRow key={field.id}>
                          <TableCell className="text-sm font-medium">{field.field_label}</TableCell>
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
            field={editingField}
            onSaved={loadFields}
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

function GlobalConfiguration() {
  return (
    <div className="space-y-8">
      <RetentionPeriodsSection />
      <StorageLocationsSection />
      <CodeRulesSection />
    </div>
  );
}

function RetentionPeriodsSection() {
  const [periods, setPeriods] = useState<RetentionPeriod[]>([]);
  const [showDialog, setShowDialog] = useState(false);
  const [editing, setEditing] = useState<RetentionPeriod | null>(null);
  const [name, setName] = useState("");
  const [years, setYears] = useState("1");

  const loadData = useCallback(async () => {
    try {
      const data = await fetchRetentionPeriods();
      setPeriods(data);
    } catch { toast.error("加载失败"); }
  }, []);

  useEffect(() => { loadData(); }, [loadData]);

  const handleSave = async () => {
    try {
      if (editing) await updateRetentionPeriod(editing.id, { name, years: parseInt(years) });
      else await createRetentionPeriod({ name, years: parseInt(years) });
      toast.success("已保存");
      setShowDialog(false);
      loadData();
    } catch { toast.error("失败"); }
  };

  const handleDelete = async (id: number) => {
    if (!confirm("确定吗?")) return;
    try { await deleteRetentionPeriod(id); toast.success("已删除"); loadData(); } catch { toast.error("失败"); }
  };

  return (
    <Card>
      <CardHeader className="flex flex-row items-center justify-between">
        <div>
          <CardTitle className="text-base">保管期限</CardTitle>
          <CardDescription className="text-xs">定义档案可供选择的存放年限</CardDescription>
        </div>
        <Button size="sm" variant="outline" onClick={() => { setEditing(null); setName(""); setYears("1"); setShowDialog(true); }}>
          <Plus className="w-4 h-4 mr-1" />
          新增
        </Button>
      </CardHeader>
      <CardContent>
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>名称</TableHead>
              <TableHead>年数</TableHead>
              <TableHead className="text-right">操作</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {periods.map(p => (
              <TableRow key={p.id}>
                <TableCell className="text-sm">{p.name}</TableCell>
                <TableCell className="text-sm">{p.years} 年</TableCell>
                <TableCell className="text-right">
                  <div className="flex justify-end gap-1">
                    <Button variant="ghost" size="sm" onClick={() => { setEditing(p); setName(p.name); setYears(String(p.years)); setShowDialog(true); }}>
                      <Edit className="w-3.5 h-3.5" />
                    </Button>
                    <Button variant="ghost" size="sm" onClick={() => handleDelete(p.id)}>
                      <Trash2 className="w-3.5 h-3.5 text-destructive" />
                    </Button>
                  </div>
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </CardContent>

      <Dialog open={showDialog} onOpenChange={setShowDialog}>
        <DialogContent>
          <DialogHeader><DialogTitle>{editing ? "编辑" : "新增"}保管期限</DialogTitle></DialogHeader>
          <div className="space-y-4 py-2">
            <div className="space-y-2"><Label>名称</Label><Input value={name} onChange={e => setName(e.target.value)} placeholder="如: 长期保存" /></div>
            <div className="space-y-2"><Label>年数 (0 表示永久)</Label><Input type="number" value={years} onChange={e => setYears(e.target.value)} /></div>
          </div>
          <DialogFooter><Button onClick={handleSave}>保存</Button></DialogFooter>
        </DialogContent>
      </Dialog>
    </Card>
  );
}

function StorageLocationsSection() {
  const [locations, setLocations] = useState<StorageLocation[]>([]);
  const [showDialog, setShowDialog] = useState(false);
  const [editing, setEditing] = useState<StorageLocation | null>(null);
  const [name, setName] = useState("");
  const [desc, setDesc] = useState("");

  const loadData = useCallback(async () => {
    try {
      const data = await fetchStorageLocations();
      setLocations(data);
    } catch { toast.error("加载失败"); }
  }, []);

  useEffect(() => { loadData(); }, [loadData]);

  const handleSave = async () => {
    try {
      if (editing) await updateStorageLocation(editing.id, { name, description: desc });
      else await createStorageLocation({ name, description: desc });
      toast.success("已保存");
      setShowDialog(false);
      loadData();
    } catch { toast.error("失败"); }
  };

  const handleDelete = async (id: number) => {
    if (!confirm("确定吗?")) return;
    try { await deleteStorageLocation(id); toast.success("已删除"); loadData(); } catch { toast.error("失败"); }
  };

  return (
    <Card>
      <CardHeader className="flex flex-row items-center justify-between">
        <div>
          <CardTitle className="text-base">存档地点</CardTitle>
          <CardDescription className="text-xs">物理档案的存放库房或货架位置</CardDescription>
        </div>
        <Button size="sm" variant="outline" onClick={() => { setEditing(null); setName(""); setDesc(""); setShowDialog(true); }}>
          <Plus className="w-4 h-4 mr-1" />
          新增
        </Button>
      </CardHeader>
      <CardContent>
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>名称</TableHead>
              <TableHead>描述/详细地址</TableHead>
              <TableHead className="text-right">操作</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {locations.map(l => (
              <TableRow key={l.id}>
                <TableCell className="text-sm font-medium">{l.name}</TableCell>
                <TableCell className="text-sm text-muted-foreground">{l.description}</TableCell>
                <TableCell className="text-right">
                  <div className="flex justify-end gap-1">
                    <Button variant="ghost" size="sm" onClick={() => { setEditing(l); setName(l.name); setDesc(l.description || ""); setShowDialog(true); }}>
                      <Edit className="w-3.5 h-3.5" />
                    </Button>
                    <Button variant="ghost" size="sm" onClick={() => handleDelete(l.id)}>
                      <Trash2 className="w-3.5 h-3.5 text-destructive" />
                    </Button>
                  </div>
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </CardContent>

      <Dialog open={showDialog} onOpenChange={setShowDialog}>
        <DialogContent>
          <DialogHeader><DialogTitle>{editing ? "编辑" : "新增"}存档地点</DialogTitle></DialogHeader>
          <div className="space-y-4 py-2">
            <div className="space-y-2"><Label>名称</Label><Input value={name} onChange={e => setName(e.target.value)} placeholder="如: 档案室A" /></div>
            <div className="space-y-2"><Label>详细说明</Label><Textarea value={desc} onChange={e => setDesc(e.target.value)} placeholder="货架编号、地址等" /></div>
          </div>
          <DialogFooter><Button onClick={handleSave}>保存</Button></DialogFooter>
        </DialogContent>
      </Dialog>
    </Card>
  );
}

function CodeRulesSection() {
  const [rules, setRules] = useState<CodeRule[]>([]);
  const [showDialog, setShowDialog] = useState(false);
  const [editing, setEditing] = useState<CodeRule | null>(null);
  const [name, setName] = useState("");
  const [pattern, setPattern] = useState("");
  const [preview, setPreview] = useState<CodeRulePreview | null>(null);

  const loadData = useCallback(async () => {
    try {
      const data = await fetchCodeRules();
      setRules(data);
    } catch { toast.error("加载失败"); }
  }, []);

  useEffect(() => { loadData(); }, [loadData]);

  const handleSave = async () => {
    try {
      if (editing) await updateCodeRule(editing.id, { name, pattern });
      else await createCodeRule({ name, pattern });
      toast.success("已保存");
      setShowDialog(false);
      loadData();
    } catch { toast.error("失败"); }
  };

  const handleDelete = async (id: number) => {
    if (!confirm("确定吗?")) return;
    try { await deleteCodeRule(id); toast.success("已删除"); loadData(); } catch { toast.error("失败"); }
  };

  const handlePreview = async () => {
    try {
      const res = await getCodeRulePreview("", "", new Date().getFullYear());
      setPreview(res);
    } catch { toast.error("预览失败"); }
  };

  return (
    <Card>
      <CardHeader className="flex flex-row items-center justify-between">
        <div>
          <CardTitle className="text-base">编码规则</CardTitle>
          <CardDescription className="text-xs">定义档案编号自动生成的模板格式</CardDescription>
        </div>
        <Button size="sm" variant="outline" onClick={() => { setEditing(null); setName(""); setPattern(""); setPreview(null); setShowDialog(true); }}>
          <Plus className="w-4 h-4 mr-1" />
          新增
        </Button>
      </CardHeader>
      <CardContent>
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>名称</TableHead>
              <TableHead>模式</TableHead>
              <TableHead className="text-right">操作</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {rules.map(r => (
              <TableRow key={r.id}>
                <TableCell className="text-sm font-medium">{r.name}</TableCell>
                <TableCell className="text-xs font-mono text-muted-foreground">{r.pattern}</TableCell>
                <TableCell className="text-right">
                  <div className="flex justify-end gap-1">
                    <Button variant="ghost" size="sm" onClick={() => { setEditing(r); setName(r.name); setPattern(r.pattern); setShowDialog(true); }}>
                      <Edit className="w-3.5 h-3.5" />
                    </Button>
                    <Button variant="ghost" size="sm" onClick={() => handleDelete(r.id)}>
                      <Trash2 className="w-3.5 h-3.5 text-destructive" />
                    </Button>
                  </div>
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </CardContent>

      <Dialog open={showDialog} onOpenChange={setShowDialog}>
        <DialogContent className="max-w-2xl">
          <DialogHeader><DialogTitle>{editing ? "编辑" : "新增"}编码规则</DialogTitle></DialogHeader>
          <div className="space-y-4 py-2">
            <div className="space-y-2"><Label>名称</Label><Input value={name} onChange={e => setName(e.target.value)} placeholder="如: 合同类编码" /></div>
            <div className="space-y-2">
              <Label>模式</Label>
              <Textarea value={pattern} onChange={e => setPattern(e.target.value)} placeholder="{YYYY}{MM}{DD}-{SEQ:4}" rows={3} />
              <p className="text-[10px] text-muted-foreground">支持占位符: {'{YYYY}'}, {'{MM}'}, {'{DD}'}, {'{SEQ:4}'}, {'{CAT}'}, {'{SUBCAT}'}</p>
            </div>
            {preview && (
              <div className="p-3 bg-muted rounded-md text-xs font-mono">
                预览示例: {preview.sample_code}
              </div>
            )}
            <Button variant="secondary" size="sm" className="w-full" onClick={handlePreview}>
              <Eye className="w-3.5 h-3.5 mr-1" />
              实时预览
            </Button>
          </div>
          <DialogFooter><Button onClick={handleSave}>保存规则</Button></DialogFooter>
        </DialogContent>
      </Dialog>
    </Card>
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
