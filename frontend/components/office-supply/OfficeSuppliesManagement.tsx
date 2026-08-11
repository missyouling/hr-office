"use client";

import { useState } from "react";
import { ArrowLeft } from "lucide-react";

import { PageTransition } from "@/components/motion/page-transition";
import { Card, CardContent } from "@/components/ui/card";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { useAuth } from "@/lib/auth";
import DictionaryTab from "./tabs/DictionaryTab";
import PurchasesTab from "./tabs/PurchasesTab";
import PaymentsTab from "./tabs/PaymentsTab";
import AnalyticsTab from "./tabs/AnalyticsTab";
import BasicDataTab from "./tabs/BasicDataTab";

interface OfficeSuppliesManagementProps {
  onBack?: () => void;
}

export default function OfficeSuppliesManagement({ onBack }: OfficeSuppliesManagementProps) {
  const { user } = useAuth();
  const [activeTab, setActiveTab] = useState("dictionary");

  const userInitial = user?.full_name?.[0] || user?.username?.[0] || "U";

  return (
    <PageTransition className="mx-auto flex w-full flex-col gap-4 p-2 md:p-4">
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
              <h1 className="text-3xl font-bold tracking-tight">办公劳保</h1>
              <p className="text-muted-foreground">办公用品采购与请款管理</p>
            </div>
          </div>
          <div className="flex items-center gap-3">
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
      </div>

      <Tabs value={activeTab} onValueChange={setActiveTab} className="space-y-4">
        <TabsList className="flex w-full justify-start">
          <TabsTrigger value="dictionary">用品字典</TabsTrigger>
          <TabsTrigger value="purchases">采购单</TabsTrigger>
          <TabsTrigger value="payments">请款单</TabsTrigger>
          <TabsTrigger value="analytics">数据分析</TabsTrigger>
          <TabsTrigger value="basic">基础数据</TabsTrigger>
        </TabsList>
        <TabsContent value="dictionary" className="space-y-4">
          <DictionaryTab />
        </TabsContent>
        <TabsContent value="purchases" className="space-y-4">
          <PurchasesTab />
        </TabsContent>
        <TabsContent value="payments" className="space-y-4">
          <PaymentsTab />
        </TabsContent>
        <TabsContent value="analytics" className="space-y-4">
          <AnalyticsTab />
        </TabsContent>
        <TabsContent value="basic" className="space-y-4">
          <BasicDataTab />
        </TabsContent>
      </Tabs>
    </PageTransition>
  );
}
