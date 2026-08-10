"use client";

import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { IncomePanel, RechargePanel, RefundPanel, ResourceFeePanel } from "./income-panels";

/** 每日收入 Tab：每日收入 | 饭卡充值 | 饭卡退费 | 资源占用费 */
export default function IncomeTab() {
  return (
    <Tabs defaultValue="income">
      <TabsList className="justify-start">
        <TabsTrigger value="income">每日收入</TabsTrigger>
        <TabsTrigger value="recharge">饭卡充值</TabsTrigger>
        <TabsTrigger value="refund">饭卡退费</TabsTrigger>
        <TabsTrigger value="resource">资源占用费</TabsTrigger>
      </TabsList>
      <TabsContent value="income"><IncomePanel /></TabsContent>
      <TabsContent value="recharge"><RechargePanel /></TabsContent>
      <TabsContent value="refund"><RefundPanel /></TabsContent>
      <TabsContent value="resource"><ResourceFeePanel /></TabsContent>
    </Tabs>
  );
}
