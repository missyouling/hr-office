"use client";

import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { CategoryPanel, SupplyPanel, ExpenseCategoryPanel, SupplierPanel } from "./dictionary-panels";

/** 数据字典 Tab：食材分类 | 食材字典 | 费用科目 | 供应商 */
export default function DictionaryTab() {
  return (
    <Tabs defaultValue="category">
      <TabsList className="justify-start">
        <TabsTrigger value="category">食材分类</TabsTrigger>
        <TabsTrigger value="supply">食材字典</TabsTrigger>
        <TabsTrigger value="expense-category">费用科目</TabsTrigger>
        <TabsTrigger value="supplier">供应商</TabsTrigger>
      </TabsList>
      <TabsContent value="category"><CategoryPanel /></TabsContent>
      <TabsContent value="supply"><SupplyPanel /></TabsContent>
      <TabsContent value="expense-category"><ExpenseCategoryPanel /></TabsContent>
      <TabsContent value="supplier"><SupplierPanel /></TabsContent>
    </Tabs>
  );
}
