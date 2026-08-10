"use client";

import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { PurchasePanel, ExpensePanel } from "./purchase-panels";

/** 采购费用 Tab：食材采购 | 其他费用 */
export default function PurchaseTab() {
  return (
    <Tabs defaultValue="purchase">
      <TabsList className="justify-start">
        <TabsTrigger value="purchase">食材采购</TabsTrigger>
        <TabsTrigger value="expense">其他费用</TabsTrigger>
      </TabsList>
      <TabsContent value="purchase"><PurchasePanel /></TabsContent>
      <TabsContent value="expense"><ExpensePanel /></TabsContent>
    </Tabs>
  );
}
