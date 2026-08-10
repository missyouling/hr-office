"use client";

import { useState } from "react";
import { ArrowLeft } from "lucide-react";

import { Card, CardContent } from "@/components/ui/card";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { useAuth } from "@/lib/auth";
import DictionaryTab from "./tabs/DictionaryTab";
import PurchaseTab from "./tabs/PurchaseTab";
import IncomeTab from "./tabs/IncomeTab";
import MenuTab from "./tabs/MenuTab";
import AnalyticsTab from "./tabs/AnalyticsTab";

interface CanteenManagementProps {
  onBack?: () => void;
}

export default function CanteenManagement({ onBack }: CanteenManagementProps) {
  const { user } = useAuth();
  const [activeTab, setActiveTab] = useState("dictionary");

  const userInitial = user?.full_name?.[0] || user?.username?.[0] || "U";

  return (
    <div className="mx-auto flex w-full flex-col gap-4 p-2 md:p-4">
      <div className="flex flex-col gap-2">
        <div className="flex flex-wrap items-center justify-between gap-4">
          <div className="flex items-center gap-3">
            {onBack && (
              <button
                onClick={onBack}
                className="flex h-8 w-8 items-center justify-center rounded-full hover:bg-muted"
                aria-label="返回"
              >
                <ArrowLeft className="h-5 w-5" />
              </button>
            )}
            <div>
              <h1 className="text-3xl font-bold tracking-tight">食堂管理</h1>
              <p className="text-muted-foreground">食材采购、收入与菜单管理</p>
            </div>
          </div>
          <Card className="border-0 shadow-none">
            <CardContent className="flex items-center gap-2 p-2">
              <div className="flex h-8 w-8 items-center justify-center rounded-full bg-primary text-primary-foreground text-xs font-medium">
                {userInitial}
              </div>
              <span className="text-sm text-muted-foreground">{user?.full_name || user?.username || "当前用户"}</span>
            </CardContent>
          </Card>
        </div>
      </div>

      <Tabs value={activeTab} onValueChange={setActiveTab} className="space-y-4">
        <TabsList className="flex w-full justify-start">
          <TabsTrigger value="dictionary">数据字典</TabsTrigger>
          <TabsTrigger value="purchase">采购费用</TabsTrigger>
          <TabsTrigger value="income">每日收入</TabsTrigger>
          <TabsTrigger value="menu">每周菜单</TabsTrigger>
          <TabsTrigger value="analytics">数据分析</TabsTrigger>
        </TabsList>
        <TabsContent value="dictionary" className="space-y-4">
          <DictionaryTab />
        </TabsContent>
        <TabsContent value="purchase" className="space-y-4">
          <PurchaseTab />
        </TabsContent>
        <TabsContent value="income" className="space-y-4">
          <IncomeTab />
        </TabsContent>
        <TabsContent value="menu" className="space-y-4">
          <MenuTab />
        </TabsContent>
        <TabsContent value="analytics" className="space-y-4">
          <AnalyticsTab />
        </TabsContent>
      </Tabs>
    </div>
  );
}
