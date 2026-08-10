# P6 新模块 UI/UX 设计规范：办公劳保 + 食堂管理

> 范围：仅做设计研究与规范输出，不编写业务代码。  
> 目标：让两个新模块融入 hr-office 现有视觉体系，复用 shadcn/ui 组件，功能保持不变但界面更规范、美观。

---

## 1. 信息架构

### 1.1 入口层级

两个模块都挂在**日常事务（DailyAffairsHub）卡片墙**中，与现有「档案管理」「车队管理」等同级。

```
人事行政管理系统
└── 日常事务（DailyAffairsHub）
    ├── 档案管理
    ├── 车队管理
    ├── 食堂管理          ← 本次新增/增强
    ├── 办公劳保          ← 本次新增
    ├── 发票管理
    ├── 培训管理
    ├── 职业卫生
    └── 社保业务
```

**集成方式**（与现有 `archives` 一致）：

- 在 `DailyAffairsHub` 的 `MODULES` 数组中新增 `office-supplies`、保留/更新 `canteen`。
- 点击卡片后 `selectedModule` 进入对应管理视图，顶部提供「返回」按钮回到卡片墙。
- 不新增侧边栏主菜单项，避免与「日常事务」入口语义重复。

### 1.2 模块内部结构

#### 办公劳保（OfficeSuppliesManagement）

采用「单页面 + Tabs」模式，与宿舍管理、档案管理保持一致。

| Tab 标签 | 对应源功能 | 说明 |
| --- | --- | --- |
| 用品字典 | `DictionaryPage` | 办公用品主数据（CRUD、导入/导出 CSV） |
| 采购单 | `PurchasesPage` | 采购单列表 + 明细行编辑/查看/打印 |
| 请款单 | `PaymentsPage` | 请款单 + 关联未付款采购单 + 打印 |
| 数据分析 | `AnalyticsPage` | KPI、趋势、分类占比、价格异常 |
| 基础数据 | 分类/供应商 | 用品分类、供应商管理（源项目独立页面，合并到一页两个子 Tab） |

#### 食堂管理（CanteenManagement）

同样采用「单页面 + Tabs」，与源项目的 5 Tab 容器对齐。

| Tab 标签 | 对应源功能 | 说明 |
| --- | --- | --- |
| 数据字典 | `DictionaryTab` | 食材分类、食材字典、费用科目、供应商入口 |
| 采购费用 | `PurchaseTab` | 食材采购 + 其他费用（水电气/工资/维护） |
| 每日收入 | `IncomeTab` | 每日刷卡收入 + 资源占用费 + 饭卡充值退费 + CSV 导入 |
| 每周菜单 | `MenuTab` | 周菜单编辑、模板、打印 |
| 数据分析 | `AnalyticsTab` | 收支盈亏、每日趋势、支出构成、Top10 食材 |

### 1.3 页面命名规范

- 页面组件：`OfficeSuppliesManagement`、`CanteenManagement`
- 内部子组件：`OfficeSuppliesDictionary`、`OfficeSuppliesPurchases`、`OfficeSuppliesPayments`、`OfficeSuppliesAnalytics`、`OfficeSuppliesSettings`
- 食堂子组件：`CanteenDictionary`、`CanteenPurchase`、`CanteenIncome`、`CanteenMenu`、`CanteenAnalytics`
- 函数/Hook：camelCase，如 `useOfficeSuppliesList`、`formatCurrencyCn`

---

## 2. 页面布局规范

### 2.1 整体骨架

每个 Tab 视图统一采用「页头 + 筛选/工具栏 + 内容区」三层结构：

```
┌─────────────────────────────────────────────────────────────┐
│ 标题行：h1 标题 + 主操作按钮（新增/导入/打印）                  │
├─────────────────────────────────────────────────────────────┤
│ 筛选/工具栏 Card：搜索框 + 下拉筛选 + 日期 + 重置              │
├─────────────────────────────────────────────────────────────┤
│ 数据区 Card：                                                 │
│   · 表格（带滚动、sticky 表头、合计行）                        │
│   · 图表网格（数据分析页）                                     │
│   · 表单网格（菜单编辑页）                                     │
└─────────────────────────────────────────────────────────────┘
```

### 2.2 页头

```tsx
<div className="flex flex-col gap-2">
  <div className="flex flex-wrap items-center justify-between gap-4">
    <div>
      <h1 className="text-3xl font-bold tracking-tight">办公劳保</h1>
      <p className="text-muted-foreground">办公用品采购与请款管理</p>
    </div>
    <div className="flex items-center gap-2">
      {/* 主操作 */}
    </div>
  </div>
</div>
```

- 标题：`text-3xl font-bold tracking-tight`，与现有页面一致。
- 副标题：`text-muted-foreground`。
- 主操作区按钮右对齐，常用操作放最右（如「新增」），次要操作使用 `variant="outline"`。

### 2.3 Tabs 容器

```tsx
<Tabs defaultValue="dictionary" className="space-y-4">
  <TabsList className="flex w-full justify-start">
    <TabsTrigger value="dictionary">用品字典</TabsTrigger>
    <TabsTrigger value="purchases">采购单</TabsTrigger>
    {/* ... */}
  </TabsList>
  <TabsContent value="dictionary" className="space-y-4">
    {/* 内容 */}
  </TabsContent>
</Tabs>
```

- TabsList 左对齐，不要占满整行，与宿舍管理一致。
- 每个 `TabsContent` 内部保持 `space-y-4` 的垂直节奏。

### 2.4 筛选/工具栏 Card

```tsx
<Card>
  <CardContent className="pt-4">
    <div className="flex flex-wrap items-center gap-3">
      <div className="relative flex-1 min-w-[200px]">
        <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
        <Input placeholder="搜索品名 / 规格..." className="pl-9" />
      </div>
      <Select>
        <SelectTrigger className="w-[160px]">
          <SelectValue placeholder="全部分类" />
        </SelectTrigger>
        {/* ... */}
      </Select>
      <Button variant="outline">重置</Button>
    </div>
  </CardContent>
</Card>
```

- 筛选项横向排列，使用 `flex-wrap` 保证响应式。
- 搜索框带图标，左侧内边距 `pl-9`。
- 下拉、日期选择器统一宽度（`w-[140px]` ~ `w-[180px]`）。
- 「重置」使用 `variant="outline"`。

### 2.5 表格区 Card

```tsx
<Card>
  <CardContent className="p-0">
    <ScrollArea className="h-[calc(100vh-340px)] rounded-md border">
      <Table>
        <TableHeader className="sticky top-0 bg-muted">
          <TableRow>
            <TableHead>品名</TableHead>
            {/* ... */}
          </TableRow>
        </TableHeader>
        <TableBody>{/* ... */}</TableBody>
      </Table>
    </ScrollArea>
  </CardContent>
</Card>
```

- 表格必须包裹在 `ScrollArea` 或带 `overflow-auto` 的容器内。
- 表头 `sticky top-0 bg-muted`，避免滚动后迷失。
- 表格高度使用 `calc(100vh - 340px)` 类的动态高度，适配不同屏幕。
- 操作列按钮紧凑：`variant="ghost" size="icon" className="h-8 w-8"`。
- 合计行放在 `TableBody` 末尾，使用 `bg-muted/50 font-semibold`。

### 2.6 数据分析页布局

```tsx
<div className="space-y-4">
  {/* KPI 卡片行 */}
  <div className="grid grid-cols-2 gap-4 lg:grid-cols-4">
    <Card>...</Card>
  </div>

  {/* 图表双列 */}
  <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
    <ChartCard title="每日收支趋势">...</ChartCard>
    <ChartCard title="支出构成">...</ChartCard>
  </div>

  {/* 明细表格 */}
  <Card>...</Card>
</div>
```

- KPI 卡片：2 列（移动端）/ 4 列（桌面端）。
- 图表卡片：1 列（移动端）/ 2 列（桌面端），高度固定 `h-72`。
- 卡片标题使用 `CardHeader + CardTitle className="text-sm"`。

### 2.7 每周菜单编辑页布局

```tsx
<Card>
  <CardContent className="p-4">
    <div className="flex flex-wrap items-center justify-between gap-2">
      <div className="flex items-center gap-1">
        <Button variant="outline" size="sm"><ChevronLeft /></Button>
        <Input type="date" className="h-8 w-40" />
        <Button variant="outline" size="sm"><ChevronRight /></Button>
        <Button variant="ghost" size="sm">本周</Button>
      </div>
      <div className="flex gap-2">
        <Button variant="outline" size="sm"><Copy /> 复制上周</Button>
        <Button size="sm"><Save /> 保存</Button>
      </div>
    </div>
  </CardContent>
</Card>

<Card>
  <CardContent className="p-4">
    <Table>
      {/* 周一 ~ 周日 × 早/午/晚 */}
    </Table>
  </CardContent>
</Card>
```

---

## 3. 组件复用清单

### 3.1 直接复用现有 shadcn/ui 组件

hr-office 现有 `frontend/components/ui/` 共 25 个组件，新模块复用清单如下：

| 组件 | 用途 | 场景示例 |
| --- | --- | --- |
| `card` | 卡片容器 | 筛选区、表格区、KPI、图表卡片 |
| `button` | 所有按钮 | 新增、保存、导入、导出、打印 |
| `tabs` | 模块内视图切换 | 办公劳保 5 Tab、食堂 5 Tab |
| `dialog` | 新增/编辑/查看弹窗 | 用品编辑、采购单录入、收入录入 |
| `alert-dialog` | 删除确认 | 删除用品、采购单、收入记录 |
| `table` | 数据列表 | 字典表、采购单列表、收入表 |
| `input` | 文本/数字/日期输入 | 搜索框、表单字段 |
| `label` | 表单标签 | 弹窗表单 |
| `select` | 下拉选择 | 分类筛选、供应商选择 |
| `badge` | 状态标签 | 启用/停用、草稿/已提交 |
| `scroll-area` | 表格滚动容器 | 长列表表格 |
| `textarea` | 多行文本 | 备注、菜单内容 |
| `dropdown-menu` | 行内更多操作 | 表格行「更多」菜单 |
| `checkbox` | 多选 | 批量删除、关联采购单 |
| `radio-group` | 单选 | 支付方式、餐别 |
| `switch` | 开关 | 状态启用/停用 |
| `sheet` | 侧滑详情 | 采购单明细侧滑查看 |
| `skeleton` | 加载占位 | 数据分析页、表格首屏 |
| `separator` | 分隔线 | 弹窗内分区 |
| `tooltip` | 提示 | 操作按钮 hover 提示 |
| `progress` | 进度 | CSV 导入进度 |
| `sonner` | Toast 提示 | 保存成功、删除成功、错误提示 |

### 3.2 建议新增的小型业务组件

以下组件无法直接由现有 shadcn 组件完整表达，需基于现有组件封装，但保持 hr-office 风格：

| 组件名 | 功能 | 实现基础 |
| --- | --- | --- |
| `CsvImportDialog` | CSV 导入通用弹窗（文件选择、编码检测、预览、确认） | `dialog` + `button` + `table` |
| `AmountCnDisplay` | 金额大写展示框 | `div` + 现有边框样式，如请款单中的蓝色高亮框 |
| `InfiniteScrollTable` | 无限滚动表格容器（监听滚动底部加载） | `scroll-area` + 自定义 IntersectionObserver |
| `ChartCard` | 图表卡片容器（标题 + 固定高度 + 空状态） | `card` + `recharts` |
| `SupplySelector` | 用品/食材选择器（搜索 + 下拉列表） | `input` + `popover`（可基于现有 `dialog` 或自行轻量封装） |
| `MonthSelector` | 年月选择器 | `input type="month"` 样式统一封装 |
| `PrintPreviewButton` | 打印预览按钮（统一打开新窗口方式） | `button` + `Printer` icon |
| `EmptyState` | 表格空状态 | 现有 `PackageOpen`/图标 + 文字居中 |
| `SummaryTags` | 汇总标签行（如月度收入合计） | `badge` 组合 |

> 说明：源项目使用了大量 `window.open` + HTML 字符串实现打印预览。新模块应保持此打印能力，但把打印按钮和统一样式收敛到 `PrintPreviewButton` 与打印工具函数中。

---

## 4. 视觉与交互规范

### 4.1 配色

沿用 hr-office 现有 CSS 变量，**不引入新的主色**：

- 背景：`bg-background` / `bg-card`
- 主文字：`text-foreground`
- 次要文字：`text-muted-foreground`
- 主按钮：`bg-primary text-primary-foreground`
- 次要按钮：`variant="outline"` / `variant="secondary"`
- 危险操作：`variant="destructive"`
- 成功/收入：绿色系 `text-green-600` / `bg-green-50`
- 警告/支出：红色系 `text-red-600` / `bg-red-50`
- 金额：使用 `font-mono` 保持等宽对齐

### 4.2 间距

- 页面级：`space-y-4` 或 `space-y-6`
- Card 内部：`p-4` 或 `p-6`
- 表单字段间距：`gap-3` 或 `gap-4`
- 按钮间距：`gap-2`
- 表格行高：保持默认 `Table` 密度，不额外压缩

### 4.3 表格密度

- 表头：`text-xs font-medium text-muted-foreground`
- 单元格：`text-sm`
- 序号列：`text-xs text-muted-foreground text-center`
- 操作列按钮：`h-8 w-8`（紧凑但不难点击）
- 金额列：右对齐 `text-right font-mono`
- 状态列：居中 `text-center`

### 4.4 弹窗尺寸

| 类型 | 尺寸 | 说明 |
| --- | --- | --- |
| 简单表单 | `sm:max-w-[420px]` | 分类编辑、食材编辑 |
| 中等表单 | `sm:max-w-[560px]` | 请款单、收入录入 |
| 复杂表单/明细 | `sm:max-w-[800px]` | 采购单录入、采购单详情 |
| 全屏式大表单 | `w-full max-w-5xl h-[90vh]` | 参考宿舍管理的响应式大弹窗 |
| 确认弹窗 | `sm:max-w-[400px]` | 删除确认 |

弹窗内部滚动：内容区使用 `max-h-[85vh] overflow-y-auto` 或 `flex flex-col` + `flex-1 min-h-0 overflow-y-auto`。

### 4.5 表单布局

```tsx
<div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
  <div className="space-y-1.5">
    <Label>字段名</Label>
    <Input />
  </div>
  {/* ... */}
</div>
```

- 两列表单：`grid-cols-2 gap-3`
- 三列表单：`grid-cols-3 gap-3`
- 每个字段使用 `space-y-1.5` 分隔 label 与 input
- 必填项：在 Label 后加红色星号 `text-red-500`

### 4.6 空状态

```tsx
<TableRow>
  <TableCell colSpan={8} className="h-32 text-center text-muted-foreground">
    <PackageOpen className="mx-auto h-10 w-10 mb-2 opacity-40" />
    <p>暂无用品数据</p>
  </TableCell>
</TableRow>
```

- 统一使用图标 + 文字，图标 `opacity-40`。
- 空状态高度 `h-32`，文字 `text-muted-foreground`。

### 4.7 加载态

- 表格首屏：行内显示「加载中…」居中。
- 数据分析页：使用 `skeleton` 占位动画。
- 按钮加载：使用 `Loader2` 图标 `animate-spin`，禁用按钮。

```tsx
<Button disabled={saving}>
  {saving ? <Loader2 className="mr-2 h-4 w-4 animate-spin" /> : null}
  保存
</Button>
```

### 4.8 错误提示

统一使用 `sonner`：

```tsx
import { toast } from "sonner";

toast.success("已保存");
toast.error("保存失败", { description: error.message });
```

- 成功：简短主文案，不带表情符号，保持与 hr-office 一致。
- 失败：主文案说明操作，description 放具体错误。
- 不继续使用源项目的 `showToast('标题', '内容', 'destructive')` 形式。

---

## 5. 数据可视化规范

### 5.1 图表容器

所有 recharts 图表封装在统一卡片内：

```tsx
<Card>
  <CardHeader className="pb-2">
    <CardTitle className="text-sm">每日收支趋势</CardTitle>
  </CardHeader>
  <CardContent className="h-72">
    <ResponsiveContainer width="100%" height="100%">
      <ComposedChart data={data}>
        {/* ... */}
      </ComposedChart>
    </ResponsiveContainer>
  </CardContent>
</Card>
```

- 标题统一 `text-sm`，避免与分析页大标题冲突。
- 图表高度固定：`h-64` 或 `h-72`。
- 无数据时显示空状态：`text-sm text-muted-foreground text-center pt-24`。

### 5.2 图表配色

使用 hr-office 现有 CSS 变量 `chart-1` ~ `chart-5`，避免源项目硬编码颜色：

| 语义 | 建议颜色 |
| --- | --- |
| 收入/盈余 | `var(--chart-2)`（青绿） |
| 支出/亏损 | `var(--chart-1)`（橙红） |
| 盈亏趋势线 | `var(--chart-3)`（蓝） |
| 人次/数量 | `var(--chart-4)`（黄） |
| 其他 | `var(--chart-5)`（粉紫） |

### 5.3 坐标轴与图例

- X 轴：`tick={{ fontSize: 10 }}`，日期标签必要时旋转或省略。
- Y 轴：`tick={{ fontSize: 10 }}`，金额自动格式化。
- Tooltip：`formatter={(v) => formatCurrency(v)}`。
- Legend：统一显示中文名称，如「收入」「支出」「盈亏」。

### 5.4 饼图/环形图

- 使用 `innerRadius={50}` 做成环形图，更现代。
- 标签显示分类名 + 百分比。
- 颜色循环使用 `chart-1` ~ `chart-5`。

---

## 6. 迁移优先级建议

### 6.1 高优先级（先做高质量）

| 模块 | 功能 | 理由 |
| --- | --- | --- |
| 办公劳保 | 采购单 | 高频核心流程，含明细行编辑，体验必须流畅 |
| 办公劳保 | 请款单 | 与采购单强关联，财务高频使用 |
| 食堂管理 | 每日收入 | 每日录入，含 CSV 导入，是食堂核心数据入口 |
| 食堂管理 | 食材采购 | 与每日盈亏分析直接相关，录入频率高 |

### 6.2 中优先级

| 模块 | 功能 | 理由 |
| --- | --- | --- |
| 办公劳保 | 用品字典 | 基础数据，但交互相对标准 |
| 食堂管理 | 数据字典 | 分类、食材、费用科目，标准 CRUD |
| 食堂管理 | 数据分析 | 依赖前面数据，但可视化需统一打磨 |

### 6.3 可简化/后置

| 模块 | 功能 | 建议 |
| --- | --- | --- |
| 食堂管理 | 饭卡充值退费 | 源项目功能较复杂（CSV 列映射），首期可只做列表展示 + 简单导入 |
| 食堂管理 | 资源占用费 | 属于附加收费，首期保留 CRUD 即可 |
| 办公劳保 | 数据分析 | 先把表格和 KPI 卡片做好，复杂优化建议可后续迭代 |
| 食堂管理 | 每周菜单 | 表单表格编辑即可，模板套用可后续增强 |

---

## 7. 关键设计决策摘要

1. **信息架构**：两个模块均作为「日常事务」卡片墙入口；内部采用单页面 + Tabs 组织，与宿舍管理/档案管理保持一致。
2. **视觉克制**：完全沿用 hr-office 现有 CSS 变量与 shadcn/ui 组件，不使用源项目的蓝色主色和硬编码图表色。
3. **表格体验**：所有长列表表格使用 `ScrollArea` + sticky 表头，金额等宽右对齐，操作按钮紧凑。
4. **弹窗分层**：按复杂度选择 420px / 560px / 800px / 全屏大弹窗，避免所有表单都用同一尺寸。
5. **Toast 统一**：全部使用 `sonner`，废弃源项目自定义 `showToast` 风格。
6. **图表统一**：基于 `card` + `recharts` 的 `ChartCard` 容器，使用 CSS 变量 `chart-1~5` 配色。
7. **复用优先**：25 个现有 ui 组件基本覆盖需求，仅新增 9 个小型业务组件以收敛 CSV 导入、金额大写、无限滚动、打印预览等重复逻辑。
