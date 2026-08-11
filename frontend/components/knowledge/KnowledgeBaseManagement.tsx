"use client";

import { useState } from "react";
import { ArrowLeft } from "lucide-react";

import { Card, CardContent } from "@/components/ui/card";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { useAuth } from "@/lib/auth";
import KnowledgeListTab from "./tabs/KnowledgeListTab";
import IngestTab from "./tabs/IngestTab";
import PermissionsTab from "./tabs/PermissionsTab";
import MaskingTab from "./tabs/MaskingTab";

interface KnowledgeBaseManagementProps {
  onBack?: () => void;
}

export default function KnowledgeBaseManagement({ onBack }: KnowledgeBaseManagementProps) {
  const { user } = useAuth();
  const [activeTab, setActiveTab] = useState("knowledge-list");

  const userInitial = user?.full_name?.[0] || user?.username?.[0] || "U";

  return (
    <div className="mx-auto flex w-full flex-col gap-4 p-2 md:p-4">
      {/* 页头：标题 + 用户卡片 + 返回按钮 */}
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
              <h1 className="text-3xl font-bold tracking-tight">知识库</h1>
              <p className="text-muted-foreground">智能问答的底层知识来源管理</p>
            </div>
          </div>
          <Card className="border-0 shadow-none">
            <CardContent className="flex items-center gap-2 p-2">
              <div className="flex h-8 w-8 items-center justify-center rounded-full bg-primary text-primary-foreground text-xs font-medium">
                {userInitial}
              </div>
              <span className="text-sm text-muted-foreground">
                {user?.full_name || user?.username || "当前用户"}
              </span>
            </CardContent>
          </Card>
        </div>
      </div>

      {/* 4 Tab 容器 */}
      <Tabs value={activeTab} onValueChange={setActiveTab} className="space-y-4">
        <TabsList className="flex w-full justify-start">
          <TabsTrigger value="knowledge-list">知识库列表</TabsTrigger>
          <TabsTrigger value="ingest">入库管理</TabsTrigger>
          <TabsTrigger value="permissions">权限配置</TabsTrigger>
          <TabsTrigger value="masking">脱敏规则</TabsTrigger>
        </TabsList>

        <TabsContent value="knowledge-list" className="space-y-4">
          <KnowledgeListTab />
        </TabsContent>
        <TabsContent value="ingest" className="space-y-4">
          <IngestTab />
        </TabsContent>
        <TabsContent value="permissions" className="space-y-4">
          <PermissionsTab />
        </TabsContent>
        <TabsContent value="masking" className="space-y-4">
          <MaskingTab />
        </TabsContent>
      </Tabs>
    </div>
  );
}
