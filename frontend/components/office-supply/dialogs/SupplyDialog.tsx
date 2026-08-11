"use client";

import { useEffect, useState } from "react";
import { toast } from "sonner";
import { Loader2 } from "lucide-react";

import type { OfficeSupply, OfficeCategory } from "@/lib/api-office";
import { Button } from "@/components/ui/button";
import { Dialog, DialogContent, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Checkbox } from "@/components/ui/checkbox";
import { categoriesApi, suppliesApi } from "../api";

// 表单记录类型：在 lib 正式类型基础上追加实际 API 响应的扩展字段
interface SupplyRecord extends OfficeSupply {
  spec?: string;
  reference_price?: number;
  status?: string;
  remark?: string;
}

interface SupplyDialogProps {
  open: boolean;
  onClose: (refresh: boolean) => void;
  supply?: SupplyRecord;
}

export default function SupplyDialog({ open, onClose, supply }: SupplyDialogProps) {
  const isEdit = !!supply;
  const [cats, setCats] = useState<OfficeCategory[]>([]);
  const [units, setUnits] = useState<string[]>(["个", "包", "箱", "瓶", "支", "双", "卷", "盒", "条", "袋"]);
  const [loading, setLoading] = useState(false);
  const [continuous, setContinuous] = useState(false);
  const [form, setForm] = useState({
    name: "",
    spec: "",
    unit: "个",
    reference_price: "",
    category_id: "",
    status: "active",
    remark: "",
  });

  useEffect(() => {
    categoriesApi.list().then((r) => setCats(r.items)).catch(() => {});
    suppliesApi.listUnits().then((r) => { if (r.units?.length) setUnits(r.units); }).catch(() => {});
  }, []);

  useEffect(() => {
    if (!open) return;
    if (supply) {
      setForm({
        name: supply.name || "",
        spec: supply.spec || "",
        unit: supply.unit || "个",
        reference_price: String(supply.reference_price || ""),
        category_id: String(supply.category_id || ""),
        status: supply.status || "active",
        remark: supply.remark || "",
      });
      setContinuous(false);
    } else if (!continuous) {
      setForm({ name: "", spec: "", unit: "个", reference_price: "", category_id: "", status: "active", remark: "" });
    } else {
      setForm((p) => ({ ...p, name: "", spec: "", reference_price: "" }));
    }
  }, [open, supply, continuous]);

  const handleSubmit = async () => {
    if (!form.name.trim()) { toast.error("品名不能为空"); return; }
    const price = parseFloat(form.reference_price);
    if (Number.isNaN(price) || price < 0) { toast.error("请输入有效单价"); return; }
    if (!form.category_id) { toast.error("请选择分类"); return; }

    setLoading(true);
    try {
      const data = {
        name: form.name.trim(),
        spec: form.spec.trim(),
        unit: form.unit,
        reference_price: price,
        category_id: parseInt(form.category_id, 10),
        status: form.status,
        remark: form.remark.trim(),
      };
      if (isEdit) {
        await suppliesApi.update(supply.id, data);
        toast.success("用品已更新");
      } else {
        await suppliesApi.create(data);
        toast.success("用品已添加");
        if (continuous) {
          setForm((p) => ({ ...p, name: "", spec: "", reference_price: "" }));
          return;
        }
      }
      onClose(true);
    } catch (e: unknown) {
      const msg = e instanceof Error ? e.message : "保存失败";
      toast.error("保存失败", { description: msg });
    } finally {
      setLoading(false);
    }
  };

  return (
    <Dialog open={open} onOpenChange={(v) => { if (!v) onClose(false); }}>
      <DialogContent className="sm:max-w-[520px]">
        <DialogHeader><DialogTitle>{isEdit ? "编辑用品" : "新增用品"}</DialogTitle></DialogHeader>
        <div className="grid gap-4 py-2">
          <div className="grid grid-cols-4 items-center gap-3">
            <Label className="text-right">品名 <span className="text-red-500">*</span></Label>
            <Input className="col-span-3" value={form.name} onChange={(e) => setForm({ ...form, name: e.target.value })} placeholder="输入用品名称" />
          </div>
          <div className="grid grid-cols-4 items-center gap-3">
            <Label className="text-right">规格</Label>
            <Input className="col-span-3" value={form.spec} onChange={(e) => setForm({ ...form, spec: e.target.value })} placeholder="如：70g 500张/包" />
          </div>
          <div className="grid grid-cols-4 items-center gap-3">
            <Label className="text-right">单位</Label>
            <Input className="col-span-1" value={form.unit} onChange={(e) => setForm({ ...form, unit: e.target.value })} list="unit-list" />
            <datalist id="unit-list">{units.map((u) => <option key={u} value={u} />)}</datalist>
            <Label className="text-right col-start-3">参考单价 <span className="text-red-500">*</span></Label>
            <Input type="number" step="0.01" min="0" className="col-span-1" value={form.reference_price} onChange={(e) => setForm({ ...form, reference_price: e.target.value })} placeholder="0.00" />
          </div>
          <div className="grid grid-cols-4 items-center gap-3">
            <Label className="text-right">分类 <span className="text-red-500">*</span></Label>
            <Select value={form.category_id} onValueChange={(v) => setForm({ ...form, category_id: v })}>
              <SelectTrigger className="col-span-3"><SelectValue placeholder="选择分类" /></SelectTrigger>
              <SelectContent>{cats.map((c) => <SelectItem key={c.id} value={String(c.id)}>{c.name}</SelectItem>)}</SelectContent>
            </Select>
          </div>
          <div className="grid grid-cols-4 items-center gap-3">
            <Label className="text-right">状态</Label>
            <Select value={form.status} onValueChange={(v) => setForm({ ...form, status: v })}>
              <SelectTrigger className="col-span-3"><SelectValue /></SelectTrigger>
              <SelectContent>
                <SelectItem value="active">启用</SelectItem>
                <SelectItem value="inactive">停用</SelectItem>
              </SelectContent>
            </Select>
          </div>
          <div className="grid grid-cols-4 items-start gap-3">
            <Label className="text-right pt-2">备注</Label>
            <textarea className="col-span-3 flex h-20 w-full rounded-md border border-input bg-transparent px-3 py-2 text-sm" value={form.remark} onChange={(e) => setForm({ ...form, remark: e.target.value })} placeholder="可选备注信息" />
          </div>
          {!isEdit && (
            <div className="flex items-center gap-2 pl-16">
              <Checkbox id="continuous" checked={continuous} onCheckedChange={(v) => setContinuous(v as boolean)} />
              <Label htmlFor="continuous" className="text-sm cursor-pointer">连续添加（保存后继续录入下一项）</Label>
            </div>
          )}
        </div>
        <div className="flex justify-end gap-3 pt-2">
          <Button variant="outline" onClick={() => onClose(false)}>取消</Button>
          <Button onClick={handleSubmit} disabled={loading}>
            {loading ? <Loader2 className="mr-2 h-4 w-4 animate-spin" /> : null}
            保存
          </Button>
        </div>
      </DialogContent>
    </Dialog>
  );
}
