# P9 全站 UI 治理设计规范 v2

> 范围：仅输出设计规范，不编写业务代码。  
> 目标：统一 hr-office 全站视觉语言，治理历史大模块的 20+ 粗糙点，同时借鉴 LINUX DO CDK 项目的渐变、毛玻璃与动效亮点，保持企业管理系统应有的专业克制。  
> 驱动批次：P10.2 / P10.3 / P10.4。

---

## 1. 设计原则

### 1.1 视觉语言定位

hr-office 是面向人事行政场景的企业管理系统，视觉定位是**专业克制 + 适度个性**：

- **专业**：信息层级清晰、表单/表格高效、颜色语义稳定，避免过度装饰影响操作效率。
- **克制**：不追逐潮流特效，动效服务于状态反馈与空间引导。
- **适度个性**：在首页欢迎卡、日常事务卡片墙、浮动 Dock 三个高频触点引入 CDK 风格的渐变、毛玻璃与弹性动效，形成品牌记忆点。

### 1.2 三条铁律

1. **不引入新主色**：全站继续使用现有 oklch CSS 变量体系（`/home/hr-office/frontend/app/globals.css`），禁止为治理而新增 `--primary` 级别的主色。
2. **复用现有 25 个 shadcn/ui 组件**：不因为"更好看"而新增基础组件；所有亮点效果通过现有 `card/button/dialog/tabs/scroll-area/skeleton/sonner` 等组件组合实现。
3. **深浅双模式完整**：任何颜色、背景、边框、阴影都必须同时适配 light/dark，禁止只写亮色的硬编码色值。

---

## 2. Design Token 体系

### 2.1 颜色使用规则

颜色全部来自 `/home/hr-office/frontend/app/globals.css` 已定义的变量，按以下语义使用：

| 变量 | 使用场景 | 对应类名 |
| --- | --- | --- |
| `--background` | 页面底层背景、侧边栏背景 | `bg-background` |
| `--foreground` | 主文字、图标主色 | `text-foreground` |
| `--card` | 内容区卡片、模块根容器 | `bg-card` |
| `--card-foreground` | 卡片内主文字 | `text-card-foreground` |
| `--popover` | 下拉菜单、浮层面板 | `bg-popover` |
| `--muted` | 表头、骨架屏、次要区块 | `bg-muted` / `text-muted-foreground` |
| `--primary` | 主按钮、激活态、品牌强调 | `bg-primary` / `text-primary-foreground` |
| `--secondary` | 次级按钮、标签背景 | `bg-secondary` / `text-secondary-foreground` |
| `--accent` | 悬停高亮、选中背景 | `bg-accent` / `text-accent-foreground` |
| `--border` | 边框、分割线 | `border-border` |
| `--input` | 输入框边框 | `border-input` |
| `--ring` | 焦点环 | `ring-ring` / `focus-visible:ring-ring` |
| `--destructive` | 删除、错误、必填星号 | `bg-destructive` / `text-destructive` |
| `--chart-1` ~ `--chart-5` | recharts 图表专用 | `text-chart-1` ~ `text-chart-5` / `var(--chart-N)` |
| `--sidebar-*` | 侧边栏专属语义 | `bg-sidebar-primary` 等 |

**强制规则**：

- 禁止在组件内写死 `#xxxxxx` / `rgb(...)` / `bg-white` / `text-gray-900` 等 light-only 颜色。
- 状态色（成功/警告/错误/信息）仅用于小型徽章或文字点缀，不得用作大面积背景：
  - 成功：`text-green-600 bg-green-50 dark:text-green-400 dark:bg-green-950/30`
  - 警告：`text-yellow-600 bg-yellow-50 dark:text-yellow-400 dark:bg-yellow-950/30`
  - 错误：`text-red-600 bg-red-50 dark:text-red-400 dark:bg-red-950/30`
  - 信息：`text-blue-600 bg-blue-50 dark:text-blue-400 dark:bg-blue-950/30`
- 图表颜色只能使用 `--chart-1` ~ `--chart-5`，禁止像 `/home/hr-office/frontend/components/system-settings.tsx` 第 750 行那样维护 15 色 hex 硬编码色板。
- 需要新增一次性强调色时，必须在 `globals.css` 中定义 CSS 变量，并在 `:root` 与 `.dark` 下同时赋值。

### 2.2 圆角层级

圆角统一使用 Tailwind 语义类，与 shadcn/ui v4 的 `--radius` 体系对齐：

| 元素 | 圆角 | 说明 |
| --- | --- | --- |
| 按钮 | `rounded-md`（8px） | shadcn Button 默认，保持 |
| 输入框 | `rounded-md`（6px） | 搜索框可升级为 `rounded-lg` |
| 普通卡片 | `rounded-xl`（14px） | Card 默认，对应 `calc(var(--radius) + 4px)` |
| 大容器 / 页面内容区 | `rounded-3xl`（24px） | 如 `page.tsx` 的内容区外壳 |
| 横幅 / Hero | `rounded-[32px]`（32px） | 首页欢迎卡专用 |
| 头像 / 图标按钮 | `rounded-full` | 圆形元素 |

### 2.3 阴影层级

全站统一为 3 档阴影，禁止随意新增阴影值：

| 档位 | 类名/值 | 使用场景 |
| --- | --- | --- |
| 卡片阴影 | `shadow-sm` / `hover:shadow-md` | 普通卡片、表格容器、KPI 卡片 |
| 浮层阴影 | `shadow-lg` | 下拉菜单、对话框、浮动 Dock、聊天面板 |
| 品牌大阴影 | `shadow-[0_12px_40px_-16px_rgba(0,0,0,0.35)]` | 首页欢迎卡、重点 KPI 卡片 |

深色模式下阴影需降低浓度：

```
dark:shadow-white/5
```

### 2.4 字体

- 正文：使用 `font-sans`（Geist Sans），保持默认字重与行高。
- 等宽：使用 `font-mono`（Geist Mono）展示金额、时间戳、统计数字、百分比、ID 类信息，并视情况加 `tabular-nums` 保证对齐。
- 标题：页面大标题 `text-3xl font-bold tracking-tight`，卡片标题 `text-base font-semibold`。

### 2.5 渐变规范

全站仅允许两类受控渐变，其他位置原则上不使用渐变背景：

#### A. 品牌横幅渐变（首页欢迎卡）

```
bg-gradient-to-r from-blue-600 via-blue-700 to-purple-800
```

- 文字使用 `text-white`。
- 装饰光斑使用白色半透明模糊圆：
  ```
  absolute -right-20 -top-20 h-64 w-64 rounded-full bg-white/10 blur-3xl
  absolute -bottom-16 -left-16 h-48 w-48 rounded-full bg-white/5 blur-2xl
  ```

#### B. 日常事务卡片墙业务语义渐变（8 张卡片）

| 模块 | 图标区渐变类 |
| --- | --- |
| 档案管理 | `bg-gradient-to-br from-blue-500 to-blue-700` |
| 办公劳保 | `bg-gradient-to-br from-teal-500 to-emerald-700` |
| 车队管理 | `bg-gradient-to-br from-green-500 to-lime-700` |
| 食堂管理 | `bg-gradient-to-br from-orange-500 to-amber-700` |
| 发票管理 | `bg-gradient-to-br from-purple-500 to-violet-700` |
| 培训管理 | `bg-gradient-to-br from-yellow-400 to-orange-600` |
| 职业卫生 | `bg-gradient-to-br from-red-500 to-rose-700` |
| 社保业务 | `bg-gradient-to-br from-indigo-500 to-blue-700` |

卡片本身仍使用 `bg-card` 作为底色，渐变只集中在图标容器上，避免色彩失控。

---

## 3. 弹窗尺寸四档规范（P10.2.1）

### 3.1 统一尺寸常量

所有新增/治理的弹窗必须从以下四档中选择，禁止随意写 `sm:max-w-[xxxpx]`：

| 档位 | 宽度 | 使用场景 | 建议常量 |
| --- | --- | --- | --- |
| sm | 420px | 简单表单、二次确认、删除确认 | `DIALOG_SIZE_CLASSES.sm` |
| md | 560px | 标准表单（80% 业务场景） | `DIALOG_SIZE_CLASSES.md` |
| lg | 800px | 复杂表单、详情预览、多列表单 | `DIALOG_SIZE_CLASSES.lg` |
| full | 全屏大弹窗 | 多栏编辑、大段预览 | `DIALOG_SIZE_CLASSES.full` |

常量建议值：

```ts
export const DIALOG_SIZE_CLASSES = {
  sm: "sm:max-w-[420px]",
  md: "sm:max-w-[560px]",
  lg: "sm:max-w-[800px]",
  full: "w-full max-w-5xl h-[90vh]",
};
```

### 3.2 内部滚动模式

复杂弹窗必须使用 **flex 布局 + flex-1 min-h-0 overflow-y-auto**，禁止让整个弹窗内容区无限制撑高：

```tsx
<DialogContent className={DIALOG_SIZE_CLASSES.lg}>
  <DialogHeader>
    <DialogTitle>标题</DialogTitle>
  </DialogHeader>

  {/* 可滚动内容区 */}
  <div className="flex-1 min-h-0 overflow-y-auto py-2">
    {/* 表单 / 详情 */}
  </div>

  <DialogFooter>
    <Button variant="outline">取消</Button>
    <Button>保存</Button>
  </DialogFooter>
</DialogContent>
```

对于简单表单弹窗，可退化为：

```
<DialogContent className="sm:max-w-[560px] max-h-[85vh] overflow-y-auto">
```

---

## 4. 三态组件规范（P10.2.2）

### 4.1 加载态：TableLoading

表格首屏加载统一使用骨架屏组件，**禁止出现英文 "Loading" 文案**。

**组件规范**：

```tsx
function TableLoading({ columns }: { columns: number }) {
  return (
    <>
      {Array.from({ length: 5 }).map((_, rowIdx) => (
        <TableRow key={rowIdx}>
          {Array.from({ length: columns }).map((_, colIdx) => (
            <TableCell key={colIdx}>
              <Skeleton className="h-4 w-[85%]" />
            </TableCell>
          ))}
        </TableRow>
      ))}
    </>
  );
}
```

**规则**：

- 默认渲染 5 行，列数与当前表头一致。
- 每行最后一列（通常是操作列）可缩短为 `w-[40%]`。
- 非表格区域加载使用 `Skeleton` 矩形占位，文案统一写 `加载中…`。
- 按钮加载使用 `Loader2` 图标 + `animate-spin` + `disabled`。

### 4.2 空状态：EmptyState

参考 `/home/cdk/frontend/components/common/layout/EmptyState.tsx` 的动效模式，封装全站统一的 `EmptyState` 组件。

**视觉规范**：

```tsx
<motion.div
  className="flex flex-col items-center justify-center text-center p-8 h-full"
  initial="hidden"
  animate="visible"
  variants={containerVariants}
>
  <motion.div
    className="mx-auto mb-4 flex h-12 w-12 items-center justify-center rounded-full bg-muted"
    variants={iconVariants}
  >
    <Icon className="h-6 w-6 text-muted-foreground" />
  </motion.div>
  <motion.div className="mb-2 text-base font-semibold" variants={itemVariants}>
    {title}
  </motion.div>
  <motion.div className="mb-4 text-sm text-muted-foreground" variants={itemVariants}>
    {description}
  </motion.div>
  {children && <motion.div variants={itemVariants}>{children}</motion.div>}
</motion.div>
```

**动效 variants**：

```ts
const containerVariants = {
  hidden: { opacity: 0, y: 20 },
  visible: {
    opacity: 1,
    y: 0,
    transition: { duration: 0.5, ease: "easeOut", staggerChildren: 0.1 },
  },
};

const itemVariants = {
  hidden: { opacity: 0, y: 10, scale: 0.95 },
  visible: { opacity: 1, y: 0, scale: 1, transition: { duration: 0.4, ease: "easeOut" } },
};

const iconVariants = {
  hidden: { opacity: 0, scale: 0.8 },
  visible: { opacity: 1, scale: 1, transition: { duration: 0.5, ease: "backOut" } },
};
```

**规则**：

- 图标区固定 48px（`h-12 w-12`），灰色底圆（`bg-muted`）。
- 标题 `text-base font-semibold`，描述 `text-sm text-muted-foreground`。
- 表格内空状态最小高度 `h-32`，页面级空状态最小高度 `h-64`。
- 必须带 motion 弹性入场，禁止静态图标居中。

### 4.3 错误态

错误态使用**内嵌错误卡片 + 重新加载按钮 + sonner toast** 三层兜底：

```tsx
<Card className="border-destructive/50 bg-destructive/5">
  <CardContent className="flex flex-col items-center justify-center gap-3 p-8">
    <AlertCircle className="h-10 w-10 text-destructive" />
    <div className="text-center">
      <p className="font-semibold">加载失败</p>
      <p className="text-sm text-muted-foreground">{error.message}</p>
    </div>
    <Button variant="outline" onClick={onRetry}>
      <RotateCcw className="mr-2 h-4 w-4" />
      重新加载
    </Button>
  </CardContent>
</Card>
```

同时通过 `toast.error("操作失败", { description: error.message })` 进行全局提示。

---

## 5. 表格规范（P10.2.4）

### 5.1 统一容器

所有长表格必须包裹在 `ScrollArea` 内，表头使用 `sticky`：

```tsx
<Card>
  <CardContent className="p-0">
    <ScrollArea className="h-[calc(100vh-340px)] rounded-md border">
      <Table>
        <TableHeader className="sticky top-0 z-10 bg-muted">
          <TableRow>
            <TableHead className="text-xs font-medium text-muted-foreground">列名</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {loading ? <TableLoading columns={N} /> : rows}
          {!loading && rows.length === 0 && <EmptyStateRow colSpan={N} />}
        </TableBody>
      </Table>
    </ScrollArea>
  </CardContent>
</Card>
```

### 5.2 标准高度取值

根据页面结构选择以下标准值，避免每个页面写不同数字：

| 页面结构 | 建议高度 |
| --- | --- |
| 有页头 + Tabs + 工具栏 | `h-[calc(100vh-340px)]` |
| 有返回按钮 + 页头 + 工具栏 | `h-[calc(100vh-300px)]` |
| 全屏大弹窗内的表格 | `h-[calc(90vh-180px)]` |
| 简单页面无工具栏 | `h-[calc(100vh-260px)]` |

### 5.3 表头与行规范

- 表头：`bg-muted` + `text-xs font-medium text-muted-foreground`。
- 行悬停：`hover:bg-muted/50`。
- **禁止斑马纹**。
- 操作列右对齐，表头也右对齐：`className="text-right"`。
- 金额列右对齐 + 等宽：`className="text-right font-mono tabular-nums"`。
- 状态列居中：`className="text-center"`。

### 5.4 移除全局硬编码表头

`/home/hr-office/frontend/app/globals.css` 第 138-149 行对 `table thead tr` 的硬编码颜色必须删除或替换为：

```css
@layer base {
  table thead tr {
    background-color: var(--muted);
  }
  table thead tr:hover {
    background-color: var(--muted);
  }
}
```

---

## 6. 深色模式规范（P10.2.5）

### 6.1 修复清单

| 文件 | 问题位置 | 修复要求 |
| --- | --- | --- |
| `/home/hr-office/frontend/app/page.tsx` 第 154 行 | 内容区 `bg-card` 与模块内部 `bg-card` 嵌套 | 保留外层单一 `bg-card`，模块根容器改为透明或 `bg-background` |
| `/home/hr-office/frontend/components/daily-affairs-hub.tsx` 第 119 行 | 根容器 `bg-card` 导致与外层卡片背景重叠 | 移除该 `bg-card`，使用透明背景 |
| `/home/hr-office/frontend/components/chat-panel.tsx` 第 370 行 | `bg-white`、`from-blue-50 to-white`、大量 `text-gray-*`、`border-gray-200`、`bg-blue-600` | 全部替换为主题变量或 Tailwind 语义色 |
| `/home/hr-office/frontend/components/layout/nav-main.tsx` 第 32 行 | 激活态 `bg-black text-white dark:bg-white dark:text-black` | 改为 `bg-sidebar-primary text-sidebar-primary-foreground` |
| `/home/hr-office/frontend/components/layout/management-bar.tsx` 第 81 行 | `left: calc(16rem + 1rem)` 硬编码 | 改为 `calc(var(--sidebar-width) + 1rem)` |
| `/home/hr-office/frontend/components/system-settings.tsx` 第 750-754 行 | `COLORS` 15 色 hex 硬编码色板 | 使用 `--chart-1` ~ `--chart-5` 循环 |
| `/home/hr-office/frontend/app/globals.css` 第 138-149 行 | 表头硬编码 `#f5f5f5` / `#161616` | 删除或替换为 `var(--muted)` |

### 6.2 深色适配通用规则

1. **禁止 `bg-white` / `bg-black` / `text-gray-900` / `text-gray-500` 等单向硬编码色**，所有背景与文字必须走变量。
2. **渐变在深色下可以不变**，但装饰光斑透明度可适当降低：`dark:bg-white/5`。
3. **阴影在深色下必须减弱**：`dark:shadow-white/5`。
4. **状态色必须带 dark 变体**：例如 `dark:text-green-400 dark:bg-green-950/30`。
5. **新增一次性颜色必须成对定义**：在 `:root` 与 `.dark` 下同时提供变量值。

---

## 7. 布局规范（P10.3）

### 7.1 侧边栏激活态

`/home/hr-office/frontend/components/layout/nav-main.tsx` 当前激活态使用硬编码黑白，必须改为：

```tsx
<SidebarMenuButton
  className={cn(
    "h-10 w-full items-center gap-3 rounded-lg px-3 transition-colors",
    active
      ? "bg-sidebar-primary text-sidebar-primary-foreground"
      : "text-sidebar-foreground hover:bg-sidebar-accent hover:text-sidebar-accent-foreground"
  )}
  isActive={active}
  onClick={() => onSelect(item.id)}
>
  <Icon className="h-4 w-4" />
  <span className="text-sm font-medium">{item.label}</span>
</SidebarMenuButton>
```

### 7.2 ManagementBar 折叠响应式

`/home/hr-office/frontend/components/layout/management-bar.tsx` 第 81 行禁止硬编码 `16rem`，必须使用 CSS 变量：

```tsx
<div
  className="pointer-events-none fixed bottom-8 z-50 flex flex-col items-start gap-2"
  style={{ left: "calc(var(--sidebar-width) + 1rem)" }}
>
```

若需响应侧边栏折叠状态，建议通过父级 `group-data-[state=collapsed]` 或监听 `useSidebar()` 的 `state` 动态切换：

```tsx
const { state } = useSidebar();
const left = state === "collapsed" ? "calc(var(--sidebar-width-icon) + 1rem)" : "calc(var(--sidebar-width) + 1rem)";
```

### 7.3 内容区：单一容器

`/home/hr-office/frontend/app/page.tsx` 内容区应保证**单层卡片背景**，避免模块根容器再次添加 `bg-card`：

```tsx
<div className="flex-1 overflow-auto p-6 bg-card md:min-h-[800px] md:rounded-3xl md:shadow-sm">
  {renderMainContent()}
</div>
```

各模块根组件应使用透明背景：`className="flex flex-col gap-4"`。

### 7.4 Tabs 统一左对齐

沿用 P6 / P8 规范：

```tsx
<TabsList className="flex w-full justify-start">
  <TabsTrigger value="...">...</TabsTrigger>
</TabsList>
```

`TabsContent` 内部统一 `className="space-y-4"`。

---

## 8. 视觉亮点（P10.3.3 - P10.3.5）

### 8.1 首页欢迎卡

`/home/hr-office/frontend/app/page.tsx` 第 199 行的硬编码渐变需替换为品牌渐变，并增加装饰光斑：

```tsx
<div className="relative overflow-hidden rounded-[32px] bg-gradient-to-r from-blue-600 via-blue-700 to-purple-800 p-6 shadow-[0_12px_40px_-16px_rgba(0,0,0,0.35)]">
  {/* 装饰光斑 */}
  <div className="pointer-events-none absolute -right-20 -top-20 h-64 w-64 rounded-full bg-white/10 blur-3xl" />
  <div className="pointer-events-none absolute -bottom-16 -left-16 h-48 w-48 rounded-full bg-white/5 blur-2xl" />

  {/* 轮播内容 */}
  <div className="relative flex items-center justify-between text-white">
    ...
  </div>
</div>
```

- 左右切换按钮：`bg-white/20 backdrop-blur hover:bg-white/30 text-white rounded-full`。
- 指示器：当前 `w-6 bg-white`，非当前 `w-1.5 bg-white/40`。
- 轮播切换使用 `AnimatePresence` 实现淡入淡出（可选）。

### 8.2 日常事务卡片墙

`/home/hr-office/frontend/components/daily-affairs-hub.tsx` 的 `MODULES` 数组应移除 `color: "bg-blue-500"` 这类单色，改用第 2.5 节定义的渐变：

```tsx
const MODULES = [
  {
    id: "archives",
    name: "档案管理",
    description: "...",
    icon: FileText,
    gradient: "bg-gradient-to-br from-blue-500 to-blue-700",
  },
  // ...
];
```

卡片渲染示例：

```tsx
<Card
  className="group relative cursor-pointer overflow-hidden rounded-2xl transition-all duration-200 hover:scale-[1.02] hover:shadow-lg"
  onClick={() => handleModuleClick(module.id)}
>
  <CardContent className="relative flex flex-col items-center gap-4 p-6">
    {/* 装饰模糊 */}
    <div className={`absolute -right-6 -top-6 h-24 w-24 rounded-full bg-gradient-to-br ${module.gradient} opacity-20 blur-2xl`} />
    <div className={`relative flex h-16 w-16 items-center justify-center rounded-2xl ${module.gradient} text-white shadow-md`}>
      <Icon className="h-8 w-8" />
    </div>
    <div className="text-center">
      <h3 className="text-lg font-semibold">{module.name}</h3>
      <p className="mt-1 text-sm text-muted-foreground">{module.description}</p>
    </div>
  </CardContent>
</Card>
```

### 8.3 浮动 Dock

`/home/hr-office/frontend/components/ui/floating-dock.tsx` 当前桌面版使用硬编码 `#E5E7EB` / `#292929`，需改为毛玻璃主题化：

```tsx
<motion.div
  className={cn(
    "mx-auto hidden h-12 items-end gap-2 rounded-xl px-2 pb-2 md:flex",
    "bg-background/70 backdrop-blur-md border border-border/60 shadow-lg shadow-black/10 dark:shadow-white/5",
    className
  )}
>
```

`/home/hr-office/frontend/components/layout/management-bar.tsx` 传参保持glass风格：

```tsx
<FloatingDock
  items={dockItems}
  desktopClassName="pointer-events-auto bg-background/70 backdrop-blur-md border border-border/60 shadow-lg shadow-black/10 dark:shadow-white/5"
  mobileClassName="pointer-events-auto"
  mobileButtonClassName="bg-background/80 backdrop-blur"
/>
```

---

## 9. 动效规范（P10.4）

### 9.1 页面入场

普通页面使用 stagger 容器 + item 位移，避免所有元素同时出现：

```tsx
const containerVariants = {
  hidden: { opacity: 0 },
  visible: {
    opacity: 1,
    transition: { staggerChildren: 0.05, delayChildren: 0.05 },
  },
};

const itemVariants = {
  hidden: { opacity: 0, y: 16 },
  visible: {
    opacity: 1,
    y: 0,
    transition: { duration: 0.4, ease: [0.25, 0.1, 0.25, 1] },
  },
};
```

### 9.2 数字滚动

KPI 统计数字使用 CDK 风格的 `CountingNumber`：

```tsx
<CountingNumber
  number={value}
  className="font-mono text-2xl font-bold tabular-nums"
  transition={{ stiffness: 90, damping: 50 }}
/>
```

### 9.3 弹窗动效

保持 shadcn/ui 默认的 `zoom-in-95`，不自定义复杂弹窗动画：

```
data-[state=open]:animate-in data-[state=closed]:animate-out
data-[state=closed]:fade-out-0 data-[state=open]:fade-in-0
data-[state=closed]:zoom-out-95 data-[state=open]:zoom-in-95
```

### 9.4 空状态弹性入场

见第 4.2 节 `iconVariants` 的 `ease: "backOut"`。

### 9.5 悬停与反馈

- 卡片悬停：`transition-all duration-200 hover:scale-[1.02] hover:-translate-y-0.5 hover:shadow-md`。
- 按钮悬停：shadcn 默认 `transition-colors` 即可，禁止加弹跳。
- 操作成功：使用 `sonner` toast，不额外加庆祝动画。

### 9.6 禁止项

以下动效在企业管理系统中不适用，**明确禁止**：

- 液体按钮（liquid button）
- 鼠标跟随光标
- 全屏背景线条/粒子
- 视差滚动
- 过度弹跳 / rubber band
- 页面切换的 3D 翻转

---

## 10. 表单规范

### 10.1 标签与必填

统一使用 shadcn `Label`，必填项在标签后加红色星号：

```tsx
<div className="space-y-1.5">
  <Label>
    字段名称
    <span className="ml-0.5 text-destructive">*</span>
  </Label>
  <Input placeholder="请输入..." />
  <p className="text-xs text-destructive">{error}</p>
</div>
```

### 10.2 输入框焦点态

不自定义 focus 颜色，统一使用 `--ring`：

```
focus-visible:ring-1 focus-visible:ring-ring
```

### 10.3 表单布局

```tsx
<div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
  <div className="space-y-1.5">...</div>
</div>
```

- 两列：`sm:grid-cols-2`
- 三列：`lg:grid-cols-3`
- 单列长表单：保持 `grid-cols-1`
- 每个字段之间 `space-y-1.5`

### 10.4 表单操作

弹窗表单底部统一使用 `DialogFooter`，主操作在右侧：

```tsx
<DialogFooter className="gap-2 sm:gap-0">
  <Button variant="outline" onClick={onCancel}>取消</Button>
  <Button disabled={saving}>
    {saving && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
    保存
  </Button>
</DialogFooter>
```

---

## 11. 实施优先级与批次映射

| 规范条目 | 实施批次 | 交付物/目标 |
| --- | --- | --- |
| 2.1 颜色：禁止硬编码色、状态色规范 | P10.2.5 | `globals.css` 补充 dark 状态色；`system-settings.tsx` 移除 `COLORS` |
| 2.2 圆角层级 | P10.3 | 统一卡片、按钮、横幅圆角 |
| 2.3 阴影层级 | P10.3 | 统一 3 档阴影，深色模式降级 |
| 2.5 渐变规范 | P10.3.3 / P10.3.4 | 首页欢迎卡 + 卡片墙 8 色渐变 |
| 3. 弹窗尺寸四档 | P10.2.1 | 定义 `DIALOG_SIZE_CLASSES`；治理所有 Dialog |
| 4.1 加载态 TableLoading | P10.2.2 | 封装 `TableLoading`；替换所有英文 Loading |
| 4.2 空状态 EmptyState | P10.2.2 / P10.4 | 封装 `EmptyState`；全站空状态接入 motion |
| 4.3 错误态 | P10.2.2 | 统一错误卡片 + toast |
| 5. 表格规范 | P10.2.4 | 统一 ScrollArea + sticky 表头 + 高度标准 |
| 6. 深色模式修复清单 | P10.2.5 | 修复 chat-panel、nav-main、management-bar、page.tsx 等 |
| 7. 布局规范 | P10.3 | 侧边栏激活态、ManagementBar 响应式、单一容器、Tabs 左对齐 |
| 8.1 首页欢迎卡 | P10.3.3 | 品牌渐变 + 光斑 + 轮播 |
| 8.2 日常事务卡片墙 | P10.3.4 | 8 张卡片语义渐变 |
| 8.3 浮动 Dock | P10.3.5 | 毛玻璃 + 主题化阴影 |
| 9. 动效规范 | P10.4 | 页面 stagger、CountingNumber、hover 微动效 |
| 10. 表单规范 | P10.2 / P10.3 | Label、必填星号、焦点态、DialogFooter |

**建议执行顺序**：

1. P10.2.1 弹窗尺寸 → P10.2.2 三态组件 → P10.2.4 表格 → P10.2.5 深色修复。
2. P10.3 布局治理 → P10.3.3 首页欢迎卡 → P10.3.4 卡片墙 → P10.3.5 浮动 Dock。
3. P10.4 动效统一收尾，确保性能与体验稳定。

---

## 12. 验收清单

治理完成后，按以下清单进行全站检查：

- [ ] 全站无硬编码 `#xxxxxx` / `rgb(...)` / `bg-white` / `text-gray-900` 等 light-only 颜色，除第 2.5 节批准的渐变外。
- [ ] `/home/hr-office/frontend/components/system-settings.tsx` 第 750 行附近的 15 色 `COLORS` 数组已移除，图表使用 `--chart-1` ~ `--chart-5`。
- [ ] 所有表格使用 `ScrollArea` + sticky 表头 + `bg-muted`，无斑马纹。
- [ ] 所有表格空状态使用 `EmptyState`，所有表格加载使用 `TableLoading`，无英文 "Loading"。
- [ ] 所有弹窗符合 sm / md / lg / full 四档尺寸之一，且内容区遵循滚动规范。
- [ ] 深色模式下：`chat-panel` 无 `bg-white` / 硬编码蓝色；`nav-main` 激活态使用 `sidebar-primary`；`management-bar` 使用 `var(--sidebar-width)`。
- [ ] `/home/hr-office/frontend/app/globals.css` 第 138-149 行硬编码表头颜色已删除或替换为变量。
- [ ] `page.tsx` 内容区与模块根容器无双层 `bg-card` 嵌套。
- [ ] `TabsList` 全部左对齐：`flex w-full justify-start`。
- [ ] 首页欢迎卡使用品牌渐变 + 装饰光斑 + 轮播指示器。
- [ ] 日常事务 8 张卡片使用第 2.5 节语义渐变，并带 hover 微动效。
- [ ] 浮动 Dock 使用毛玻璃 `bg-background/70 backdrop-blur-md` + 主题化阴影，无 `#E5E7EB` / `#292929`。
- [ ] KPI 统计数字使用 `CountingNumber` 或等效数字滚动效果。
- [ ] 表单标签含必填星号，输入框焦点环使用 `--ring`。
- [ ] 无液体按钮、鼠标跟随、背景线条等禁止动效。
- [ ] `npm run lint` 与 `npm run build` 通过。

---

**参考文件路径**：

- 当前设计系统：`/home/hr-office/frontend/app/globals.css`
- 现有规范：`/home/hr-office/docs/ui-design-p6.md`、`/home/hr-office/docs/ui-design-p8.md`
- CDK 参考：`/home/cdk/frontend/app/globals.css`、`/home/cdk/frontend/components/common/project/constants.ts`、`/home/cdk/frontend/components/common/layout/EmptyState.tsx`、`/home/cdk/frontend/app/(main)/layout.tsx`、`/home/cdk/frontend/components/animate-ui/`
- hr-office 治理对象：`/home/hr-office/frontend/app/page.tsx`、`/home/hr-office/frontend/components/daily-affairs-hub.tsx`、`/home/hr-office/frontend/components/chat-panel.tsx`、`/home/hr-office/frontend/components/layout/app-sidebar.tsx`、`/home/hr-office/frontend/components/layout/management-bar.tsx`、`/home/hr-office/frontend/components/layout/nav-main.tsx`、`/home/hr-office/frontend/components/ui/floating-dock.tsx`、`/home/hr-office/frontend/components/system-settings.tsx`
