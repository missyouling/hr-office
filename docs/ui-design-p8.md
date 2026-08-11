# P8.3 知识库模块 UI/UX 设计规范

> 范围：仅做 UI/UX 设计规范输出，不编写业务代码。  
> 目标：让知识库模块（KnowledgeBaseManagement）融入 hr-office 现有视觉体系，复用 `frontend/components/ui/` 已有 shadcn/ui 组件，功能与 P8.1/P8.2 后端能力对齐。

---

## 1. 信息架构

### 1.1 入口层级

知识库模块挂在**日常事务（DailyAffairsHub）卡片墙**中，与「档案管理」「办公劳保」「食堂管理」等同级，点击卡片进入 `KnowledgeBaseManagement` 单页面。

```
人事行政管理系统
└── 日常事务（DailyAffairsHub）
    ├── 档案管理
    ├── 车队管理
    ├── 食堂管理
    ├── 办公劳保
    ├── 知识库          ← 本次新增（P8.3 实施）
    ├── 发票管理
    ├── 培训管理
    ├── 职业卫生
    └── 社保业务
```

**集成方式**：

- 在 `DailyAffairsHub` 的 `MODULES` 数组中新增 `knowledge-base` 入口（若 P8.3 同步实施）。
- 进入后使用「单页面 + Tabs」模式，与 `OfficeSuppliesManagement`、`dormitory-management` 保持一致。
- 顶部保留「返回」按钮回到卡片墙。
- 悬浮聊天面板 `ChatPanel` 已在 P8.0 完成 401 自动刷新升级；本模块只负责知识库配置，不直接改动聊天组件。

### 1.2 模块内部结构

`KnowledgeBaseManagement` 采用 4 个主 Tab，与现有骨架保持一致：

| Tab 标签 | value | 对应后端能力 | 说明 |
| --- | --- | --- | --- |
| 知识库列表 | `knowledge-list` | `GET /api/knowledge-bases`、`POST/GET/PUT/DELETE /api/knowledge-bases/{id}` | 全部可见 KB 的查看、创建、编辑、删除入口 |
| 入库管理 | `ingest` | `POST /api/knowledge-bases/{id}/ingest` | 选择 KB、上传/录入文档、预览分块、提交入库 |
| 权限配置 | `permissions` | `GET/POST/DELETE /api/knowledge-bases/{id}/rules` | 按 KB 配置访问规则（可见范围/读写权限） |
| 脱敏规则 | `masking` | `GET/POST/DELETE /api/knowledge-bases/{id}/masks` | 按 KB 配置敏感信息匹配与替换规则 |

> 说明：`GET /api/knowledge-bases/stats` 的统计能力直接融入「知识库列表」顶部 KPI 卡片区，不再单独增加一个 Tab，避免破坏现有 4 Tab 骨架。

### 1.3 每个 Tab 内的子面板/视图结构

#### 知识库列表（knowledge-list）

```
┌─────────────────────────────────────────────────────────────┐
│ 标题行：知识库 + 副标题 + 主操作「新建知识库」                   │
├─────────────────────────────────────────────────────────────┤
│ 概览卡片行：总数 / 自建库 / 系统库 / 今日入库文档数               │
├─────────────────────────────────────────────────────────────┤
│ 筛选/工具栏 Card：搜索库名 + 可见性筛选 + 重置                   │
├─────────────────────────────────────────────────────────────┤
│ 数据区 Card：                                                 │
│   · 卡片网格（每个 KB 一张 Card）                              │
│   · 或紧凑表格切换视图（备选）                                  │
└─────────────────────────────────────────────────────────────┘
```

每张知识库卡片包含：名称、描述、可见性（公开/私有/指定范围）、文档数/字符数、最近更新时间、操作按钮（编辑、权限、脱敏、删除）。

#### 入库管理（ingest）

```
┌─────────────────────────────────────────────────────────────┐
│ 标题行：入库管理 + 当前选中知识库名称                            │
├─────────────────────────────────────────────────────────────┤
│ 上下文选择区：Select 选择要入库的知识库 KB                       │
├─────────────────────────────────────────────────────────────┤
│ 数据源 Card：                                                 │
│   · 文件上传区（拖拽/点击）                                     │
│   · 或纯文本输入区 / URL 输入区                                │
├─────────────────────────────────────────────────────────────┤
│ 预览/分块 Card：                                              │
│   · 左侧：提取出的文本/分片列表                                 │
│   · 右侧：选中分片的原始内容高亮                                │
├─────────────────────────────────────────────────────────────┤
│ 操作栏：开始入库（带 Progress）+ 取消                           │
└─────────────────────────────────────────────────────────────┘
```

#### 权限配置（permissions）

```
┌─────────────────────────────────────────────────────────────┐
│ 标题行：权限配置 + 当前选中知识库                                │
├─────────────────────────────────────────────────────────────┤
│ 上下文选择区：Select 选择要配置权限的 KB                         │
├─────────────────────────────────────────────────────────────┤
│ 规则工具栏：搜索主体 + 新增规则按钮                              │
├─────────────────────────────────────────────────────────────┤
│ 规则表格：主体类型 / 主体标识 / 权限 / 生效时间 / 操作           │
└─────────────────────────────────────────────────────────────┘
```

权限规则的新增/编辑使用 Dialog 表单；删除使用 AlertDialog 确认。

#### 脱敏规则（masking）

```
┌─────────────────────────────────────────────────────────────┐
│ 标题行：脱敏规则 + 当前选中知识库                                │
├─────────────────────────────────────────────────────────────┤
│ 上下文选择区：Select 选择要配置脱敏的 KB                         │
├─────────────────────────────────────────────────────────────┤
│ 规则工具栏：搜索规则名 + 新增规则按钮                            │
├─────────────────────────────────────────────────────────────┤
│ 规则表格：名称 / 匹配模式 / 替换文本 / 启用状态 / 操作           │
├─────────────────────────────────────────────────────────────┤
│ 测试区 Card：输入样本文本 → 实时显示脱敏后结果                   │
└─────────────────────────────────────────────────────────────┘
```

### 1.4 页面命名规范

- 页面组件：`KnowledgeBaseManagement`
- 内部子组件：
  - `KnowledgeBaseList`
  - `KnowledgeBaseFormDialog`
  - `KnowledgeBaseIngestPanel`
  - `KnowledgeBasePermissionPanel`
  - `KnowledgeBaseMaskingPanel`
  - `KnowledgeBaseStatsCards`
  - `KnowledgeBaseEmptyState`
- 函数/Hook：`useKnowledgeBases`、`useKnowledgeBaseRules`、`useKnowledgeBaseMasks`、`formatBytesCn` 等，统一 `camelCase`。

---

## 2. 页面布局规范

### 2.1 整体骨架

每个 Tab 视图统一采用「页头 + 上下文/工具栏 + 内容区」三层结构，与现有模块节奏一致：

```
┌─────────────────────────────────────────────────────────────┐
│ 标题行：h1 标题 + 副标题 + 当前用户卡片 + 主操作按钮            │
├─────────────────────────────────────────────────────────────┤
│ 工具栏 Card：Select 上下文 + 搜索 + 筛选 + 重置                │
├─────────────────────────────────────────────────────────────┤
│ 数据区 Card：                                                 │
│   · 卡片网格 / 表格                                           │
│   · 表单弹窗 / 侧滑面板                                       │
└─────────────────────────────────────────────────────────────┘
```

### 2.2 页头

沿用 `OfficeSuppliesManagement` 与现有骨架的页头结构：

```tsx
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
        <span className="text-sm text-muted-foreground">{user?.full_name || user?.username || "当前用户"}</span>
      </CardContent>
    </Card>
  </div>
</div>
```

### 2.3 Tabs 容器

```tsx
<Tabs value={activeTab} onValueChange={setActiveTab} className="space-y-4">
  <TabsList className="flex w-full justify-start">
    <TabsTrigger value="knowledge-list">知识库列表</TabsTrigger>
    <TabsTrigger value="ingest">入库管理</TabsTrigger>
    <TabsTrigger value="permissions">权限配置</TabsTrigger>
    <TabsTrigger value="masking">脱敏规则</TabsTrigger>
  </TabsList>
  <TabsContent value="knowledge-list" className="space-y-4">
    {/* ... */}
  </TabsContent>
</Tabs>
```

- `TabsList` 左对齐，不要占满整行。
- `TabsContent` 内部统一 `space-y-4`。

### 2.4 知识库列表布局

#### 概览卡片行

```tsx
<div className="grid grid-cols-2 gap-4 lg:grid-cols-4">
  <Card>...</Card>
</div>
```

#### 筛选/工具栏 Card

```tsx
<Card>
  <CardContent className="pt-4">
    <div className="flex flex-wrap items-center gap-3">
      <div className="relative flex-1 min-w-[200px]">
        <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
        <Input placeholder="搜索知识库名称..." className="pl-9" />
      </div>
      <Select>
        <SelectTrigger className="w-[160px]">
          <SelectValue placeholder="全部可见性" />
        </SelectTrigger>
        {/* 全部 / 公开 / 私有 / 指定范围 */}
      </Select>
      <Button variant="outline">重置</Button>
    </div>
  </CardContent>
</Card>
```

#### 知识库卡片网格

```tsx
<div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3 2xl:grid-cols-4">
  <Card>
    <CardContent className="p-4">
      {/* 名称、描述、元信息、操作 */}
    </CardContent>
  </Card>
</div>
```

- 参考宿舍管理的 site card 网格，每张卡片信息密度适中，留出呼吸感。
- 操作按钮使用 `DropdownMenu` 收敛「编辑 / 权限 / 脱敏 / 删除」，避免卡片上按钮过多。

### 2.5 入库管理布局

采用「上下分块 + 左右预览」的复合布局：

```tsx
<div className="space-y-4">
  <Card>{/* 选择 KB + 数据源表单 */}</Card>
  <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
    <Card>{/* 分块列表 */}</Card>
    <Card>{/* 选中分片预览 */}</Card>
  </div>
</div>
```

- 当未选择 KB 或数据源为空时，右侧预览区显示空状态。
- 入库进度使用 `Progress` 组件放在操作按钮上方。

### 2.6 权限/脱敏规则布局

统一使用「工具栏 + 表格」：

```tsx
<Card>
  <CardContent className="pt-4">
    <div className="flex flex-wrap items-center justify-between gap-3">
      <div className="flex flex-wrap items-center gap-3">{/* 搜索 + 筛选 */}</div>
      <Button>新增规则</Button>
    </div>
  </CardContent>
</Card>

<Card>
  <CardContent className="p-0">
    <ScrollArea className="h-[calc(100vh-380px)] rounded-md border">
      <Table>{/* ... */}</Table>
    </ScrollArea>
  </CardContent>
</Card>
```

- 规则表格需要 `ScrollArea` + sticky 表头。
- 脱敏规则页在表格下方再放置一个「规则测试」Card，用于实时验证。

### 2.7 弹窗与侧滑尺寸

| 类型 | 尺寸 | 说明 |
| --- | --- | --- |
| 新建/编辑知识库 | `sm:max-w-[560px]` | 名称、描述、可见性、Embedding 模型等 |
| 删除确认 | `sm:max-w-[400px]` | 标准 AlertDialog |
| 权限规则编辑 | `sm:max-w-[560px]` | 主体选择 + 权限级别 |
| 脱敏规则编辑 | `sm:max-w-[560px]` | 名称、正则、替换文本、启用开关 |
| 入库详情/分片预览 | `w-full max-w-5xl h-[90vh]` 或 Sheet `w-[480px]` | 复杂预览使用大弹窗；简单元数据使用侧滑 |

---

## 3. 组件复用清单

### 3.1 直接复用现有 shadcn/ui 组件

`frontend/components/ui/` 目录目前包含 25 个组件（含 `floating-dock`）。知识库模块复用其中 24 个通用组件，不引入新的基础 UI 依赖：

| 组件 | 用途 | 场景示例 |
| --- | --- | --- |
| `card` | 卡片容器 | 概览卡片、筛选区、表格区、规则测试区 |
| `button` | 所有按钮 | 新建、保存、删除、入库、重置 |
| `tabs` | 模块内视图切换 | 4 个主 Tab |
| `dialog` | 新增/编辑弹窗 | 知识库表单、规则表单 |
| `alert-dialog` | 删除确认 | 删除 KB、删除规则 |
| `table` | 数据列表 | 权限规则表、脱敏规则表、入库记录表 |
| `input` | 文本/数字/日期输入 | 搜索框、规则名称、替换文本 |
| `label` | 表单标签 | 弹窗表单 |
| `select` | 下拉选择 | 选择知识库、可见性、权限级别 |
| `badge` | 状态标签 | 公开/私有、启用/停用、系统库/自定义 |
| `scroll-area` | 表格滚动容器 | 规则表格、分块列表 |
| `textarea` | 多行文本 | 知识库描述、脱敏测试文本、文档原文 |
| `dropdown-menu` | 行内更多操作 | 卡片/表格行「更多」菜单 |
| `checkbox` | 多选 | 批量选择规则 |
| `radio-group` | 单选 | 权限级别单选、数据源类型单选 |
| `switch` | 开关 | 规则启用/停用、KB 状态开关 |
| `sheet` | 侧滑详情 | 知识库详情、分片预览 |
| `skeleton` | 加载占位 | 概览卡片、表格首屏 |
| `separator` | 分隔线 | 弹窗内分区、测试区上下分块 |
| `tooltip` | 提示 | 操作按钮 hover 提示 |
| `progress` | 进度 | 文件上传进度、入库进度 |
| `sonner` | Toast 提示 | 保存成功、删除成功、错误提示 |
| `accordion` | 折叠面板 | 入库高级选项、规则说明 |
| `tooltip` | 图标提示 | 正则语法提示、权限说明 |

> 说明：`floating-dock` 属于悬浮 Dock 组件，与知识库主页面无直接交集，暂不纳入复用清单。

### 3.2 建议新增的小型业务组件

以下组件无法由现有 shadcn/ui 组件完整表达，需要基于现有组件封装，但保持 hr-office 风格：

| 组件名 | 功能 | 实现基础 |
| --- | --- | --- |
| `KnowledgeBaseCard` | 知识库卡片（名称、描述、可见性、文档数、操作菜单） | `card` + `badge` + `dropdown-menu` |
| `KnowledgeBaseFormDialog` | 新建/编辑知识库弹窗 | `dialog` + `input` + `select` + `textarea` |
| `KnowledgeBaseStatsCards` | 顶部 KPI 概览卡片组 | `card` + `skeleton` |
| `KnowledgeBaseIngestPanel` | 入库数据源 + 预览分块组合面板 | `card` + `input` + `textarea` + `progress` |
| `KnowledgeBasePermissionPanel` | 权限规则表格 + 新增规则表单 | `table` + `dialog` + `select` |
| `KnowledgeBaseMaskingPanel` | 脱敏规则表格 + 实时测试区 | `table` + `dialog` + `switch` + `textarea` |
| `KnowledgeBaseEmptyState` | 空状态（无 KB / 无规则 / 无文档） | 现有图标 + 文字居中 |
| `KnowledgeBaseDeleteAlert` | 删除 KB/规则确认弹窗 | `alert-dialog` |
| `MaskingTestBox` | 脱敏规则测试输入/输出对比框 | `textarea` + `separator` + `badge` |

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
- 成功/启用：绿色系 `text-green-600` / `bg-green-50`
- 警告/私有：红色系 `text-red-600` / `bg-red-50`
- 数字/存储大小：使用 `font-mono` 保持等宽对齐

### 4.2 间距

- 页面级：`space-y-4` 或 `space-y-6`
- Card 内部：`p-4` 或 `p-6`
- 表单字段间距：`gap-3` 或 `gap-4`
- 按钮间距：`gap-2`
- 卡片网格间距：`gap-4`
- 表格行高：保持默认 `Table` 密度，不额外压缩

### 4.3 知识库卡片密度

```tsx
<Card className="group relative flex flex-col">
  <CardContent className="flex flex-1 flex-col p-4">
    <div className="flex items-start justify-between gap-2">
      <h3 className="text-base font-semibold">{kb.name}</h3>
      <Badge variant={kb.visibility === "public" ? "default" : "secondary"}>
        {visibilityLabel}
      </Badge>
    </div>
    <p className="mt-1 line-clamp-2 text-sm text-muted-foreground">{kb.description}</p>
    <div className="mt-4 flex items-center gap-3 text-xs text-muted-foreground">
      <span>{kb.docCount} 文档</span>
      <span>{formatBytesCn(kb.size)}</span>
      <span>更新于 {kb.updatedAt}</span>
    </div>
  </CardContent>
</Card>
```

- 卡片标题：`text-base font-semibold`
- 描述文字：`text-sm text-muted-foreground`，最多两行 `line-clamp-2`
- 元信息：`text-xs text-muted-foreground`

### 4.4 表格密度

- 表头：`text-xs font-medium text-muted-foreground`
- 单元格：`text-sm`
- 规则名/主体列：左对齐
- 权限/状态列：居中 `text-center`
- 操作列按钮：`h-8 w-8`，使用 `variant="ghost" size="icon"`

### 4.5 弹窗尺寸与表单布局

- 简单表单：`sm:max-w-[420px]`
- 中等表单：`sm:max-w-[560px]`
- 复杂预览：`w-full max-w-5xl h-[90vh]`

表单字段网格：

```tsx
<div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
  <div className="space-y-1.5">
    <Label>规则名称</Label>
    <Input />
  </div>
  {/* ... */}
</div>
```

- 每个字段使用 `space-y-1.5` 分隔 label 与 input。
- 必填项在 Label 后加红色星号 `text-red-500`。

### 4.6 空状态

```tsx
<div className="flex h-64 flex-col items-center justify-center text-muted-foreground">
  <BookOpen className="mb-3 h-12 w-12 opacity-40" />
  <p className="text-base">暂无知识库</p>
  <p className="mt-1 text-sm">点击「新建知识库」开始创建</p>
</div>
```

- 统一使用图标 + 文字，图标 `opacity-40`。
- 空状态高度 `h-64`，文字 `text-muted-foreground`。
- 规则表格空状态高度 `h-32`。

### 4.7 加载态

- 概览卡片：`skeleton` 矩形占位。
- 表格首屏：行内显示「加载中…」居中，或骨架行。
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

- 成功：简短主文案，不带表情符号。
- 失败：主文案说明操作，`description` 放具体错误。
- 权限不足、后端校验失败等错误统一通过 `toast.error` 呈现，不在页面内嵌大段红色提示。

### 4.9 权限感知渲染

- 「新建知识库」按钮仅对管理员或拥有 `knowledge_bases:create` 权限的用户显示；普通用户只读列表。
- 「删除」操作对非管理员隐藏或在 DropdownMenu 中禁用并带 Tooltip 说明。
- 「权限配置」「脱敏规则」Tab 内的规则编辑仅对 KB 所有者/管理员开放，普通成员进入该 Tab 时表格只读或隐藏编辑入口。

---

## 5. 数据可视化规范

### 5.1 统计入口

`GET /api/knowledge-bases/stats` 返回的统计数据用于「知识库列表」顶部 KPI 卡片与图表区：

```tsx
<div className="space-y-4">
  {/* KPI 卡片行 */}
  <div className="grid grid-cols-2 gap-4 lg:grid-cols-4">
    <Card>...</Card>
  </div>

  {/* 图表双列 */}
  <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
    <ChartCard title="知识库类型分布">...</ChartCard>
    <ChartCard title="文档入库趋势">...</ChartCard>
  </div>
</div>
```

### 5.2 图表容器

所有 recharts 图表封装在统一卡片内：

```tsx
<Card>
  <CardHeader className="pb-2">
    <CardTitle className="text-sm">知识库类型分布</CardTitle>
  </CardHeader>
  <CardContent className="h-72">
    <ResponsiveContainer width="100%" height="100%">
      <PieChart>{/* ... */}</PieChart>
    </ResponsiveContainer>
  </CardContent>
</Card>
```

- 标题统一 `text-sm`，避免与页面大标题冲突。
- 图表高度固定：`h-72`。
- 无数据时显示空状态：`text-sm text-muted-foreground text-center pt-24`。

### 5.3 图表配色

使用 hr-office 现有 CSS 变量 `chart-1` ~ `chart-5`，避免硬编码颜色：

| 语义 | 建议颜色 |
| --- | --- |
| 系统库 | `var(--chart-1)`（橙红） |
| 自定义库 | `var(--chart-2)`（青绿） |
| 入库文档数 | `var(--chart-3)`（蓝） |
| 存储占用 | `var(--chart-4)`（黄） |
| 其他 | `var(--chart-5)`（粉紫） |

### 5.4 坐标轴与图例

- X 轴：`tick={{ fontSize: 10 }}`，日期标签必要时旋转或省略。
- Y 轴：`tick={{ fontSize: 10 }}`，数量/存储大小自动格式化。
- Tooltip：数量直接展示，存储大小使用 `formatBytesCn`。
- Legend：统一显示中文名称，如「系统库」「自定义库」「文档数」。

### 5.5 饼图/环形图

- 使用 `innerRadius={50}` 做成环形图。
- 标签显示分类名 + 百分比。
- 颜色循环使用 `chart-1` ~ `chart-5`。

---

## 6. 迁移优先级建议

按「用户高频路径 → 依赖后置 → 复杂配置最后」的原则排序：

### 6.1 高优先级

| Tab | 核心交付 | 理由 |
| --- | --- | --- |
| 知识库列表 | 卡片网格、新建/编辑/删除弹窗、KPI 概览 | 管理员和普通用户的高频入口，决定模块第一印象 |
| 权限配置 | 规则表格、新增/删除规则弹窗、KB 上下文选择 | 权限是 P8 核心安全能力，必须尽早可用 |

### 6.2 中优先级

| Tab | 核心交付 | 理由 |
| --- | --- | --- |
| 入库管理 | 文件上传/文本录入、分块预览、入库进度 | 依赖知识库列表已可用，是知识库产生价值的关键动作 |
| 统计图表 | KPI 卡片、类型分布、入库趋势 | 依赖列表和入库数据，可视化可后续打磨 |

### 6.3 后置/可细化

| Tab | 核心交付 | 理由 |
| --- | --- | --- |
| 脱敏规则 | 规则 CRUD、实时测试区 | 属于高级安全能力，首期保证规则可配可用即可 |

### 6.4 建议实施顺序

1. **知识库列表**：先跑通 `GET/POST/PUT/DELETE` 的 UI 闭环。
2. **权限配置**：在列表页可跳转或直接切换 Tab 配置规则。
3. **入库管理**：选择 KB → 上传/录入 → 预览 → 入库。
4. **脱敏规则**：选择 KB → 维护规则 → 测试效果。

---

## 7. 关键设计决策摘要

1. **信息架构**：`KnowledgeBaseManagement` 沿用 4 Tab 骨架（知识库列表、入库管理、权限配置、脱敏规则），统计能力嵌入列表顶部，不新增独立 Tab。
2. **页面布局**：页头 + TabsList 左对齐 + 工具栏 Card + 内容区 Card，与 `OfficeSuppliesManagement` 和宿舍管理保持一致。
3. **组件复用**：复用 24 个现有 shadcn/ui 组件，新增 9 个小型业务组件封装 KB 卡片、规则面板、入库预览、脱敏测试等重复逻辑。
4. **视觉克制**：完全沿用 hr-office CSS 变量与间距体系，知识库列表使用卡片网格提升容器感，规则页使用密集表格提高效率。
5. **数据可视化**：统计图表基于 `card` + `recharts` 容器，高度 `h-72`，配色使用 `chart-1~5` 变量，无数据时显示统一空状态。
6. **迁移优先级**：知识库列表 > 权限配置 > 入库管理 > 脱敏规则，优先打通高频查看与权限配置闭环。
