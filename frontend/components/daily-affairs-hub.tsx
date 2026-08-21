"use client";

import { useEffect, useState } from "react";
import { FileText, Truck, Utensils, Receipt, GraduationCap, Shield, ArrowLeft, PackageOpen } from "lucide-react";

import { Card, CardContent } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { ArchivesManagement } from "./archives-management";
import OfficeSuppliesManagement from "./office-supply/OfficeSuppliesManagement";
import CanteenManagement from "./canteen/CanteenManagement";
import InvoiceManagement from "./invoice/InvoiceManagement";

interface DailyAffairsHubProps {
  onNavigate?: (module: string) => void;
  defaultModule?: string | null;
}

export const DAILY_AFFAIRS_MODULES = ["archives", "office-supplies", "canteen", "invoice"] as const;

export function isDailyAffairsModule(value: unknown): value is (typeof DAILY_AFFAIRS_MODULES)[number] {
  return typeof value === "string" && (DAILY_AFFAIRS_MODULES as readonly string[]).includes(value);
}

const MODULES = [
  {
    id: "archives",
    name: "档案管理",
    description: "文书、科技、电子、专门档案管理",
    icon: FileText,
    gradient: "from-blue-500 to-blue-700",
    iconColor: "text-blue-500",
  },
  {
    id: "office-supplies",
    name: "办公劳保",
    description: "办公用品字典、采购单、请款与分析",
    icon: PackageOpen,
    gradient: "from-teal-500 to-emerald-700",
    iconColor: "text-teal-500",
  },
  {
    id: "fleet",
    name: "车队管理",
    description: "车辆调度、加油、维修管理",
    icon: Truck,
    gradient: "from-green-500 to-lime-700",
    iconColor: "text-green-500",
  },
  {
    id: "canteen",
    name: "食堂管理",
    description: "食材采购、菜品管理",
    icon: Utensils,
    gradient: "from-orange-500 to-amber-700",
    iconColor: "text-orange-500",
  },
  {
    id: "invoice",
    name: "发票管理",
    description: "发票录入、报销管理",
    icon: Receipt,
    gradient: "from-purple-500 to-violet-700",
    iconColor: "text-purple-500",
  },
  {
    id: "training",
    name: "培训管理",
    description: "培训计划、课程管理",
    icon: GraduationCap,
    gradient: "from-yellow-400 to-orange-600",
    iconColor: "text-yellow-500",
  },
  {
    id: "occupational",
    name: "职业卫生",
    description: "职业健康检查管理",
    icon: Shield,
    gradient: "from-red-500 to-rose-700",
    iconColor: "text-red-500",
  },
];

export function DailyAffairsHub({ onNavigate, defaultModule = null }: DailyAffairsHubProps) {
  const [selectedModule, setSelectedModule] = useState<string | null>(isDailyAffairsModule(defaultModule) ? defaultModule : null);

  useEffect(() => {
    setSelectedModule(isDailyAffairsModule(defaultModule) ? defaultModule : null);
  }, [defaultModule]);

  const handleModuleClick = (moduleId: string) => {
    setSelectedModule(moduleId);
    onNavigate?.(moduleId);
  };

  const handleBack = () => {
    setSelectedModule(null);
  };

  // 如果选择了档案管理，显示档案管理页面
  if (selectedModule === "archives") {
    return (
      <div className="flex flex-col gap-4">
        <div className="flex items-center gap-4">
          <Button variant="ghost" size="icon" onClick={handleBack}>
            <ArrowLeft className="h-5 w-5" />
          </Button>
          <h1 className="text-2xl font-bold">档案管理</h1>
        </div>
        <ArchivesManagement />
      </div>
    );
  }

  // 办公劳保模块（P6 合并）
  if (selectedModule === "office-supplies") {
    return <OfficeSuppliesManagement onBack={handleBack} />;
  }

  // 食堂管理模块（P6 合并）
  if (selectedModule === "canteen") {
    return <CanteenManagement onBack={handleBack} />;
  }

  // 发票管理模块（P7.3）
  if (selectedModule === "invoice") {
    return <InvoiceManagement onBack={handleBack} />;
  }

  return (
    <div className="mx-auto flex w-full max-w-none flex-col gap-6 p-6 pb-16 bg-card text-foreground">
      {/* 页面标题 */}
      <header className="flex flex-col gap-2">
        <div className="flex flex-wrap items-center justify-between gap-4">
          <div>
            <h1 className="text-3xl font-bold tracking-tight">日常事务</h1>
            <p className="text-muted-foreground">快捷访问日常工作模块</p>
          </div>
        </div>
      </header>

      <div className="grid grid-cols-1 gap-6 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
        {MODULES.map((module) => {
          const Icon = module.icon;
          return (
            <Card
              key={module.id}
              className="group relative cursor-pointer overflow-hidden rounded-2xl transition-all duration-300 hover:scale-[1.02] hover:shadow-lg"
              onClick={() => handleModuleClick(module.id)}
            >
              <CardContent className="relative flex flex-col items-center gap-4 p-6">
                {/* 装饰模糊光斑 */}
                <div className={`pointer-events-none absolute -right-6 -top-6 h-24 w-24 rounded-full bg-gradient-to-br ${module.gradient} opacity-20 blur-2xl`} />
                <div className={`relative flex h-16 w-16 items-center justify-center rounded-2xl bg-gradient-to-br ${module.gradient} text-white shadow-md`}>
                  <Icon className="h-8 w-8" />
                </div>
                <div className="text-center">
                  <h3 className="text-lg font-semibold">{module.name}</h3>
                  <p className="mt-1 text-sm text-muted-foreground">{module.description}</p>
                </div>
              </CardContent>
            </Card>
          );
        })}
      </div>
    </div>
  );
}
