# 存储配置系统架构文档

## 系统架构概览

存储系统采用插件化后端设计与基于规则的动态路由机制。核心目标是将业务逻辑与底层的存储基础设施解耦。

- **Storage Backend Layer**: 提供统一的接口适配器，屏蔽本地文件系统、S3 及 WebDAV 的差异。
- **Router Layer (StorageRouter)**: 负责根据业务请求的上下文（模块、资源类型）解析出应使用的存储配置。
- **Service Layer**: 处理文件上传、下载及元数据管理。

## 数据模型

### 1. StorageConfig (存储配置)
描述物理存储后端的信息。
- `Type`: 存储类型 (`local`, `s3`, `webdav`)。
- `Config`: JSON 格式存储的具体参数。
- `IsDefault`: 是否作为全局兜底后端。

### 2. StorageModuleConfig (模块配置)
定义业务模块层面的默认行为。
- `ModuleCode`: 模块唯一标识。
- `DefaultConfigID`: 指向 `StorageConfig` 的外键。

### 3. StorageRule (存储规则)
最细粒度的路由映射。
- `ModuleCode`: 关联模块。
- `ResourceType`: 关联资源类型（可为空，表示通配）。
- `ConfigID`: 指向 `StorageConfig` 的外键。

### 4. SysFile (文件实体)
记录文件在系统中的元数据。
- `StorageConfigID`: 记录该文件具体存储在哪个后端。
- `Path`: 在后端中的相对路径。

## API 端点列表

所有管理端接口均位于 `/api/admin` 路径下：

### 存储后端管理
- `GET /api/admin/storage`: 获取所有存储后端列表。
- `POST /api/admin/storage`: 创建新的存储后端。
- `PUT /api/admin/storage`: 更新存储后端。
- `DELETE /api/admin/storage`: 删除存储后端（需检查引用）。

### 模块配置管理
- `GET /api/admin/storage/modules`: 获取模块配置。
- `POST /api/admin/storage/modules`: 创建模块配置。
- `PUT /api/admin/storage/modules/{id}`: 更新模块配置。
- `DELETE /api/admin/storage/modules/{id}`: 删除模块配置。

### 存储规则管理
- `GET /api/admin/storage/rules`: 支持按 `module_code` 和 `resource_type` 过滤。
- `POST /api/admin/storage/rules`: 创建规则。
- `PUT /api/admin/storage/rules/{id}`: 更新规则。
- `DELETE /api/admin/storage/rules/{id}`: 删除规则。

## StorageRouter 路由解析流程

当业务层发起存储请求时，`StorageRouter` 执行以下逻辑：

1.  **输入**: `module_code`, `resource_type`。
2.  **精确匹配**: 查询数据库中 `module_code` 且 `resource_type` 均吻合的 `StorageRule`。
3.  **模块回退**: 若未找到，查找对应 `module_code` 的 `StorageModuleConfig` 中定义的默认后端。
4.  **全局兜底**: 若仍未找到，获取被标记为 `is_default` 的 `StorageConfig`。
5.  **输出**: 返回对应的 `StorageConfig` 实体及其执行引擎。

**存储路径格式**:
`/{module_code}/{resource_type}/{YYYY-MM-DD}/{filename}`

## 前端组件结构

存储管理界面集成在系统设置中，主要组件位于 `frontend/components/system-settings.tsx`:

- **StorageConfigTab**: 后端列表展示、新增及编辑表单。
- **StorageRuleTab**: 规则映射管理，包含模块与后端的关联操作。
- **状态管理**: 使用 React 19 的 Action 或统一的 API 请求处理。

## 扩展指南

### 添加新模块
1.  在后端业务代码中定义新的 `module_code` 常量。
2.  在数据库或通过界面添加对应的 `StorageModuleConfig`。

### 添加新存储后端类型
1.  在 `backend/internal/service/storage` 目录下实现新的存储引擎接口（需支持 `Save`, `Get`, `Delete` 方法）。
2.  在 `StorageConfig` 模型中添加对应的配置解析逻辑。
3.  更新前端表单以支持新类型的参数输入。
