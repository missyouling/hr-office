"use client";

import { useState } from "react";
import {
  Building2,
  Plus,
  Edit,
  Trash2,
  ChevronRight,
  ChevronDown,
  Settings,
  Search,
  AlertTriangle,
} from "lucide-react";
import { toast } from "sonner";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";

// 组织机构数据结构（三层架构：集团-子公司-部门）
interface OrganizationNode {
  id: string;
  name: string;
  code: string;
  type: "group" | "subsidiary" | "department";
  parentId?: string;
  level: number;
  sort: number;
  employeeIdPrefix?: string;
  employeeIdRule?: string;
  description?: string;
  children?: OrganizationNode[];
  expanded?: boolean;
}

export function OrganizationManagement() {
  // 组织机构数据（三层架构：集团-子公司-部门）
  const [organizationData, setOrganizationData] = useState<OrganizationNode[]>([
    {
      id: "1",
      name: "某某集团有限公司",
      code: "GROUP",
      type: "group",
      level: 1,
      sort: 1,
      description: "集团总部",
      expanded: true,
      children: [
        {
          id: "2",
          name: "生产子公司",
          code: "PROD",
          type: "subsidiary",
          parentId: "1",
          level: 2,
          sort: 1,
          description: "生产制造子公司",
          expanded: true,
          children: [
            {
              id: "3",
              name: "总经办",
              code: "C020100",
              type: "department",
              parentId: "2",
              level: 3,
              sort: 1,
              employeeIdPrefix: "C020100",
              employeeIdRule: "C020100001",
            },
            {
              id: "4",
              name: "财务部",
              code: "C020200",
              type: "department",
              parentId: "2",
              level: 3,
              sort: 2,
              employeeIdPrefix: "C020200",
              employeeIdRule: "C020200001",
            },
            {
              id: "5",
              name: "销售部",
              code: "C020300",
              type: "department",
              parentId: "2",
              level: 3,
              sort: 3,
              employeeIdPrefix: "C020300",
              employeeIdRule: "C020300001",
            },
            {
              id: "6",
              name: "生产部",
              code: "C020500",
              type: "department",
              parentId: "2",
              level: 3,
              sort: 4,
              employeeIdPrefix: "C020500",
              employeeIdRule: "C020500001",
            },
            {
              id: "7",
              name: "机电部",
              code: "C020510",
              type: "department",
              parentId: "2",
              level: 3,
              sort: 5,
              employeeIdPrefix: "C020510",
              employeeIdRule: "C020510001",
            },
            {
              id: "8",
              name: "技术质量部",
              code: "C020600",
              type: "department",
              parentId: "2",
              level: 3,
              sort: 6,
              employeeIdPrefix: "C020600",
              employeeIdRule: "C020600001",
            },
            {
              id: "9",
              name: "人事行政部",
              code: "C020700",
              type: "department",
              parentId: "2",
              level: 3,
              sort: 7,
              employeeIdPrefix: "C020700",
              employeeIdRule: "C020700001",
            },
            {
              id: "10",
              name: "仓库",
              code: "C020400",
              type: "department",
              parentId: "2",
              level: 3,
              sort: 8,
              employeeIdPrefix: "C020400",
              employeeIdRule: "C020400001",
            },
          ],
        },
        {
          id: "11",
          name: "营销子公司",
          code: "MARKET",
          type: "subsidiary",
          parentId: "1",
          level: 2,
          sort: 2,
          description: "市场营销子公司",
          expanded: false,
          children: [
            {
              id: "12",
              name: "市场部",
              code: "M010100",
              type: "department",
              parentId: "11",
              level: 3,
              sort: 1,
              employeeIdPrefix: "M010100",
              employeeIdRule: "M010100001",
            },
            {
              id: "13",
              name: "客服部",
              code: "M010200",
              type: "department",
              parentId: "11",
              level: 3,
              sort: 2,
              employeeIdPrefix: "M010200",
              employeeIdRule: "M010200001",
            },
          ],
        },
      ],
    },
  ]);

  // 状态管理
  const [showAddDialog, setShowAddDialog] = useState(false);
  const [showEditDialog, setShowEditDialog] = useState(false);
  const [showDeleteDialog, setShowDeleteDialog] = useState(false);
  const [selectedNode, setSelectedNode] = useState<OrganizationNode | null>(null);
  const [nodeToDelete, setNodeToDelete] = useState<OrganizationNode | null>(null);
  const [searchTerm, setSearchTerm] = useState("");
  const [viewMode, setViewMode] = useState<"tree" | "list">("tree");

  // 新增节点表单数据
  const [newNode, setNewNode] = useState<Partial<OrganizationNode>>({
    name: "",
    code: "",
    type: undefined,
    parentId: "",
    description: "",
    employeeIdPrefix: "",
    employeeIdRule: "",
  });

  // 编辑节点表单数据
  const [editNode, setEditNode] = useState<Partial<OrganizationNode>>({});

  // 递归渲染树形结构
  const renderTreeNode = (node: OrganizationNode, depth = 0): React.ReactNode => {
    const hasChildren = node.children && node.children.length > 0;
    const paddingLeft = depth * 20;

    return (
      <div key={node.id} className="select-none">
        <div
          className="flex items-center py-2 px-2 hover:bg-muted/50 rounded cursor-pointer"
          style={{ paddingLeft: `${paddingLeft}px` }}
          onClick={() => handleNodeClick(node)}
        >
          <div className="flex items-center gap-2 flex-1">
            {hasChildren ? (
              <Button
                variant="ghost"
                size="sm"
                className="p-0 h-auto w-4"
                onClick={(e) => {
                  e.stopPropagation();
                  toggleNodeExpansion(node.id);
                }}
              >
                {node.expanded ? (
                  <ChevronDown className="h-4 w-4" />
                ) : (
                  <ChevronRight className="h-4 w-4" />
                )}
              </Button>
            ) : (
              <div className="w-4" />
            )}

            <div className="flex items-center gap-2">
              <Building2 className="h-4 w-4 text-muted-foreground" />
              <span className="font-medium">{node.name}</span>
              <span className="text-sm text-muted-foreground">({node.code})</span>
              <span className="text-xs px-2 py-1 bg-muted rounded">
                {getNodeTypeLabel(node.type)}
              </span>
            </div>
          </div>

          <div className="flex gap-1">
            <Button
              variant="ghost"
              size="sm"
              onClick={(e) => {
                e.stopPropagation();
                handleEditNode(node);
              }}
            >
              <Edit className="h-4 w-4" />
            </Button>
            <Button
              variant="ghost"
              size="sm"
              onClick={(e) => {
                e.stopPropagation();
                handleDeleteNode(node.id);
              }}
            >
              <Trash2 className="h-4 w-4" />
            </Button>
          </div>
        </div>

        {node.expanded && hasChildren && (
          <div>
            {node.children!.map(child => renderTreeNode(child, depth + 1))}
          </div>
        )}
      </div>
    );
  };

  // 获取节点类型标签
  const getNodeTypeLabel = (type: string) => {
    const labels = {
      group: "集团",
      subsidiary: "子公司",
      department: "部门",
    };
    return labels[type as keyof typeof labels] || type;
  };

  // 获取可选择的节点类型（根据父节点类型限制）
  const getAvailableTypes = (parentId?: string) => {
    if (!parentId) {
      // 没有父节点，只能创建集团
      return [{ value: "group", label: "集团" }];
    }

    const parentNode = getFlatList(organizationData).find(n => n.id === parentId);
    if (!parentNode) return [];

    switch (parentNode.type) {
      case "group":
        return [{ value: "subsidiary", label: "子公司" }];
      case "subsidiary":
        return [{ value: "department", label: "部门" }];
      default:
        return []; // 部门下不能再创建子节点
    }
  };

  // 切换节点展开状态
  const toggleNodeExpansion = (nodeId: string) => {
    const updateNode = (nodes: OrganizationNode[]): OrganizationNode[] => {
      return nodes.map(node => {
        if (node.id === nodeId) {
          return { ...node, expanded: !node.expanded };
        }
        if (node.children) {
          return { ...node, children: updateNode(node.children) };
        }
        return node;
      });
    };

    setOrganizationData(updateNode(organizationData));
  };

  // 处理节点点击
  const handleNodeClick = (node: OrganizationNode) => {
    setSelectedNode(node);
  };

  // 处理编辑节点
  const handleEditNode = (node: OrganizationNode) => {
    setEditNode({ ...node });
    setShowEditDialog(true);
  };

  // 处理删除节点
  // 查找节点（递归查找）
  const findNodeById = (nodes: OrganizationNode[], targetId: string): OrganizationNode | null => {
    for (const node of nodes) {
      if (node.id === targetId) {
        return node;
      }
      if (node.children) {
        const found = findNodeById(node.children, targetId);
        if (found) return found;
      }
    }
    return null;
  };

  const handleDeleteNode = (nodeId: string) => {
    // 查找要删除的节点
    const targetNode = findNodeById(organizationData, nodeId);
    if (!targetNode) {
      toast.error("找不到要删除的节点");
      return;
    }

    // 检查是否有子节点
    if (targetNode.children && targetNode.children.length > 0) {
      toast.error(`该节点有 ${targetNode.children.length} 个子节点，请先删除子节点`);
      return;
    }

    // 显示自定义删除确认对话框
    setNodeToDelete(targetNode);
    setShowDeleteDialog(true);
  };

  // 确认删除
  const confirmDelete = () => {
    if (!nodeToDelete) return;

    // 删除节点（递归删除）
    const deleteNodeFromTree = (nodes: OrganizationNode[]): OrganizationNode[] => {
      return nodes.filter(node => {
        if (node.id === nodeToDelete.id) {
          return false; // 过滤掉要删除的节点
        }
        if (node.children) {
          // 递归处理子节点
          node.children = deleteNodeFromTree(node.children);
        }
        return true; // 保留其他节点
      });
    };

    setOrganizationData(deleteNodeFromTree(organizationData));
    toast.success(`节点 "${nodeToDelete.name}" 删除成功`);

    // 如果删除的是当前选中的节点，清除选中状态
    if (selectedNode?.id === nodeToDelete.id) {
      setSelectedNode(null);
    }

    // 关闭对话框并清除状态
    setShowDeleteDialog(false);
    setNodeToDelete(null);
  };

  // 取消删除
  const cancelDelete = () => {
    setShowDeleteDialog(false);
    setNodeToDelete(null);
  };

  // 处理新增节点
  const handleAddNode = () => {
    if (!newNode.name || !newNode.code || !newNode.type) {
      toast.error("请填写所有必填字段（名称、编码、类型）");
      return;
    }

    // 部门类型的额外验证
    if (newNode.type === "department" && !newNode.employeeIdPrefix) {
      toast.error("部门类型必须设置工号前缀");
      return;
    }

    // 检查编码是否重复
    const flatList = getFlatList(organizationData);
    if (flatList.some(node => node.code === newNode.code)) {
      toast.error("节点编码已存在，请使用其他编码");
      return;
    }

    // 生成新节点ID
    const newId = (Math.max(...flatList.map(n => parseInt(n.id))) + 1).toString();

    // 计算层级和排序
    let level = 1;
    let sort = 1;
    if (newNode.parentId && newNode.parentId !== "root") {
      const parentNode = flatList.find(n => n.id === newNode.parentId);
      if (parentNode) {
        level = parentNode.level + 1;
        // 计算在相同父节点下的排序
        const siblings = flatList.filter(n => n.parentId === newNode.parentId);
        sort = siblings.length > 0 ? Math.max(...siblings.map(s => s.sort)) + 1 : 1;
      }
    } else {
      // 根节点排序
      const rootNodes = organizationData;
      sort = rootNodes.length > 0 ? Math.max(...rootNodes.map(n => n.sort)) + 1 : 1;
    }

    // 自动生成部门的工号相关字段
    let employeeIdPrefix = newNode.employeeIdPrefix;
    let employeeIdRule = newNode.employeeIdRule;
    if (newNode.type === "department" && newNode.code) {
      if (!employeeIdPrefix) {
        employeeIdPrefix = newNode.code;
      }
      if (!employeeIdRule) {
        employeeIdRule = `${employeeIdPrefix}001`;
      }
    }

    // 创建新节点
    const nodeToAdd: OrganizationNode = {
      id: newId,
      name: newNode.name!,
      code: newNode.code!,
      type: newNode.type as "group" | "subsidiary" | "department",
      parentId: newNode.parentId === "root" ? undefined : newNode.parentId || undefined,
      level,
      sort,
      description: newNode.description,
      employeeIdPrefix: newNode.type === "department" ? employeeIdPrefix : undefined,
      employeeIdRule: newNode.type === "department" ? employeeIdRule : undefined,
      children: [],
      expanded: false,
    };

    // 添加到组织结构中
    const addNodeToTree = (nodes: OrganizationNode[]): OrganizationNode[] => {
      if (!newNode.parentId || newNode.parentId === "root") {
        // 添加为根节点
        return [...nodes, nodeToAdd];
      }

      return nodes.map(node => {
        if (node.id === newNode.parentId) {
          // 找到父节点，添加为子节点
          const updatedChildren = [...(node.children || []), nodeToAdd];
          // 按sort排序
          updatedChildren.sort((a, b) => a.sort - b.sort);
          return {
            ...node,
            children: updatedChildren,
            expanded: true, // 展开父节点以显示新增的子节点
          };
        }
        if (node.children) {
          return {
            ...node,
            children: addNodeToTree(node.children),
          };
        }
        return node;
      });
    };

    setOrganizationData(addNodeToTree(organizationData));
    toast.success(`节点 "${newNode.name}" 新增成功`);

    // 重置表单
    setShowAddDialog(false);
    setNewNode({
      name: "",
      code: "",
      type: undefined,
      parentId: "",
      description: "",
      employeeIdPrefix: "",
      employeeIdRule: "",
    });
  };

  // 处理保存编辑
  const handleSaveEdit = () => {
    if (!editNode.name || !editNode.code) {
      toast.error("请填写必填字段");
      return;
    }

    // 检查编码是否与其他节点重复（除了自己）
    const flatList = getFlatList(organizationData);
    if (flatList.some(node => node.code === editNode.code && node.id !== editNode.id)) {
      toast.error("节点编码已存在，请使用其他编码");
      return;
    }

    // 更新组织结构中的节点
    const updateNodeInTree = (nodes: OrganizationNode[]): OrganizationNode[] => {
      return nodes.map(node => {
        if (node.id === editNode.id) {
          // 找到要编辑的节点，更新数据
          return {
            ...node,
            name: editNode.name!,
            code: editNode.code!,
            description: editNode.description,
            employeeIdPrefix: editNode.type === "department" ? editNode.employeeIdPrefix : undefined,
            employeeIdRule: editNode.type === "department" ? editNode.employeeIdRule : undefined,
          };
        }
        if (node.children) {
          return {
            ...node,
            children: updateNodeInTree(node.children),
          };
        }
        return node;
      });
    };

    setOrganizationData(updateNodeInTree(organizationData));

    // 更新选中节点的显示
    if (selectedNode && selectedNode.id === editNode.id) {
      setSelectedNode({
        ...selectedNode,
        name: editNode.name!,
        code: editNode.code!,
        description: editNode.description,
        employeeIdPrefix: editNode.employeeIdPrefix,
        employeeIdRule: editNode.employeeIdRule,
      });
    }

    toast.success(`节点 "${editNode.name}" 编辑成功`);
    setShowEditDialog(false);
    setEditNode({});
  };

  // 获取扁平化列表
  const getFlatList = (nodes: OrganizationNode[]): OrganizationNode[] => {
    const result: OrganizationNode[] = [];

    const traverse = (nodeList: OrganizationNode[]) => {
      nodeList.forEach(node => {
        result.push(node);
        if (node.children) {
          traverse(node.children);
        }
      });
    };

    traverse(nodes);
    return result;
  };

  // 过滤搜索结果
  const filteredData = searchTerm
    ? getFlatList(organizationData).filter(node =>
        node.name.toLowerCase().includes(searchTerm.toLowerCase()) ||
        node.code.toLowerCase().includes(searchTerm.toLowerCase())
      )
    : organizationData;

  return (
    <div className="min-h-screen bg-background p-6">
      <div className="mx-auto max-w-7xl space-y-6">
        {/* 页面标题 */}
        <header className="flex flex-col gap-2">
          <div className="flex items-center justify-between">
            <div>
              <h1 className="text-3xl font-bold tracking-tight">组织机构管理</h1>
              <p className="text-muted-foreground">
                集团公司层级架构设置和部门工号规则管理
              </p>
            </div>
          </div>
        </header>

        {/* 主要内容 */}
        <Tabs defaultValue="structure" className="w-full">
          <TabsList className="grid w-full grid-cols-2">
            <TabsTrigger value="structure">组织架构</TabsTrigger>
            <TabsTrigger value="settings">工号设置</TabsTrigger>
          </TabsList>

          {/* 组织架构管理 */}
          <TabsContent value="structure" className="space-y-6">
            <Card>
              <CardHeader>
                <div className="flex items-center justify-between">
                  <div>
                    <CardTitle>组织架构树</CardTitle>
                    <CardDescription>
                      管理集团公司的层级组织结构
                    </CardDescription>
                  </div>
                  <div className="flex gap-2">
                    <Dialog open={showAddDialog} onOpenChange={setShowAddDialog}>
                      <DialogTrigger asChild>
                        <Button size="sm">
                          <Plus className="h-4 w-4 mr-2" />
                          新增节点
                        </Button>
                      </DialogTrigger>
                      <DialogContent className="max-w-2xl">
                        <DialogHeader>
                          <DialogTitle>新增组织节点</DialogTitle>
                          <DialogDescription>
                            添加新的组织架构节点
                          </DialogDescription>
                        </DialogHeader>
                        <div className="grid gap-4 py-4">
                          <div className="grid grid-cols-2 gap-4">
                            <div className="space-y-2">
                              <Label htmlFor="add-name">节点名称 *</Label>
                              <Input
                                id="add-name"
                                value={newNode.name || ""}
                                onChange={(e) => setNewNode(prev => ({ ...prev, name: e.target.value }))}
                                placeholder="如：销售部"
                              />
                            </div>
                            <div className="space-y-2">
                              <Label htmlFor="add-code">节点编码 *</Label>
                              <Input
                                id="add-code"
                                value={newNode.code || ""}
                                onChange={(e) => {
                                  const code = e.target.value;
                                  setNewNode(prev => {
                                    const updated = { ...prev, code };

                                    // 如果是部门类型，自动更新工号前缀
                                    if (prev.type === "department" && code) {
                                      updated.employeeIdPrefix = code;
                                      updated.employeeIdRule = `${code}001`;
                                    }

                                    return updated;
                                  });
                                }}
                                placeholder="如：C020300"
                              />
                            </div>
                          </div>
                          <div className="grid grid-cols-2 gap-4">
                            <div className="space-y-2">
                              <Label htmlFor="add-type">节点类型</Label>
                              <Select
                                value={newNode.type || ""}
                                onValueChange={(value) => {
                                  setNewNode(prev => {
                                    const updated = {
                                      ...prev,
                                      type: value as "group" | "subsidiary" | "department",
                                    };

                                    if (value === "department") {
                                      // 部门类型：自动填充工号前缀
                                      if (prev.code && !prev.employeeIdPrefix) {
                                        updated.employeeIdPrefix = prev.code;
                                        updated.employeeIdRule = `${prev.code}001`;
                                      }
                                    } else {
                                      // 非部门类型：清除工号相关字段
                                      updated.employeeIdPrefix = "";
                                      updated.employeeIdRule = "";
                                    }

                                    return updated;
                                  });
                                }}
                              >
                                <SelectTrigger>
                                  <SelectValue placeholder="选择类型" />
                                </SelectTrigger>
                                <SelectContent>
                                  {getAvailableTypes(newNode.parentId).map(option => (
                                    <SelectItem key={option.value} value={option.value}>
                                      {option.label}
                                    </SelectItem>
                                  ))}
                                </SelectContent>
                              </Select>
                            </div>
                            <div className="space-y-2">
                              <Label htmlFor="add-parent">上级节点</Label>
                              <Select
                                value={newNode.parentId || ""}
                                onValueChange={(value) => {
                                  setNewNode(prev => ({
                                    ...prev,
                                    parentId: value,
                                    // 重置类型选择
                                    type: undefined,
                                    employeeIdPrefix: "",
                                    employeeIdRule: "",
                                  }));
                                }}
                              >
                                <SelectTrigger>
                                  <SelectValue placeholder="选择上级节点" />
                                </SelectTrigger>
                                <SelectContent>
                                  <SelectItem value="root">无上级（创建集团）</SelectItem>
                                  {getFlatList(organizationData)
                                    .filter(node => {
                                      // 只显示可以作为父节点的节点（部门不能有子节点）
                                      return node.type !== "department";
                                    })
                                    .map(node => (
                                      <SelectItem key={node.id} value={node.id}>
                                        {getNodeTypeLabel(node.type)} - {node.name} ({node.code})
                                      </SelectItem>
                                    ))}
                                </SelectContent>
                              </Select>
                            </div>
                          </div>
                          <div className="space-y-2">
                            <Label htmlFor="add-description">描述</Label>
                            <Input
                              id="add-description"
                              value={newNode.description || ""}
                              onChange={(e) => setNewNode(prev => ({ ...prev, description: e.target.value }))}
                              placeholder="节点描述"
                            />
                          </div>
                          {newNode.type === "department" && (
                            <div className="grid grid-cols-2 gap-4">
                              <div className="space-y-2">
                                <Label htmlFor="add-prefix">工号前缀 *</Label>
                                <Input
                                  id="add-prefix"
                                  value={newNode.employeeIdPrefix || ""}
                                  onChange={(e) => {
                                    const prefix = e.target.value;
                                    setNewNode(prev => ({
                                      ...prev,
                                      employeeIdPrefix: prefix,
                                      // 同步更新工号规则
                                      employeeIdRule: prefix ? `${prefix}001` : "",
                                    }));
                                  }}
                                  placeholder="如：C020300"
                                />
                              </div>
                              <div className="space-y-2">
                                <Label htmlFor="add-rule">工号规则示例</Label>
                                <Input
                                  id="add-rule"
                                  value={newNode.employeeIdRule || ""}
                                  onChange={(e) => setNewNode(prev => ({ ...prev, employeeIdRule: e.target.value }))}
                                  placeholder="如：C020300001"
                                />
                              </div>
                            </div>
                          )}
                        </div>
                        <DialogFooter>
                          <Button variant="outline" onClick={() => setShowAddDialog(false)}>
                            取消
                          </Button>
                          <Button onClick={handleAddNode}>确认新增</Button>
                        </DialogFooter>
                      </DialogContent>
                    </Dialog>

                    <div className="flex gap-2">
                      <Button
                        variant={viewMode === "tree" ? "default" : "outline"}
                        size="sm"
                        onClick={() => setViewMode("tree")}
                      >
                        树形视图
                      </Button>
                      <Button
                        variant={viewMode === "list" ? "default" : "outline"}
                        size="sm"
                        onClick={() => setViewMode("list")}
                      >
                        列表视图
                      </Button>
                    </div>
                  </div>
                </div>
              </CardHeader>
              <CardContent>
                {/* 搜索 */}
                <div className="mb-4">
                  <div className="relative">
                    <Search className="absolute left-3 top-3 h-4 w-4 text-muted-foreground" />
                    <Input
                      placeholder="搜索组织节点..."
                      value={searchTerm}
                      onChange={(e) => setSearchTerm(e.target.value)}
                      className="pl-10"
                    />
                  </div>
                </div>

                {/* 内容显示 */}
                <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
                  {/* 树形/列表视图 */}
                  <div className="lg:col-span-2">
                    <div className="border rounded-lg p-4 max-h-[600px] overflow-y-auto">
                      {viewMode === "tree" ? (
                        <div className="space-y-1">
                          {(searchTerm ? filteredData : organizationData).map(node =>
                            searchTerm ? (
                              <div key={node.id} className="flex items-center gap-2 py-2 px-2 hover:bg-muted/50 rounded cursor-pointer">
                                <Building2 className="h-4 w-4 text-muted-foreground" />
                                <span className="font-medium">{node.name}</span>
                                <span className="text-sm text-muted-foreground">({node.code})</span>
                                <span className="text-xs px-2 py-1 bg-muted rounded">
                                  {getNodeTypeLabel(node.type)}
                                </span>
                              </div>
                            ) : renderTreeNode(node)
                          )}
                        </div>
                      ) : (
                        <Table>
                          <TableHeader>
                            <TableRow>
                              <TableHead>名称</TableHead>
                              <TableHead>编码</TableHead>
                              <TableHead>类型</TableHead>
                              <TableHead>层级</TableHead>
                              <TableHead>操作</TableHead>
                            </TableRow>
                          </TableHeader>
                          <TableBody>
                            {getFlatList(organizationData).map(node => (
                              <TableRow key={node.id}>
                                <TableCell className="font-medium">{node.name}</TableCell>
                                <TableCell>{node.code}</TableCell>
                                <TableCell>
                                  <span className="text-xs px-2 py-1 bg-muted rounded">
                                    {getNodeTypeLabel(node.type)}
                                  </span>
                                </TableCell>
                                <TableCell>{node.level}</TableCell>
                                <TableCell>
                                  <div className="flex gap-2">
                                    <Button variant="outline" size="sm" onClick={() => handleEditNode(node)}>
                                      编辑
                                    </Button>
                                    <Button variant="outline" size="sm" onClick={() => handleDeleteNode(node.id)}>
                                      删除
                                    </Button>
                                  </div>
                                </TableCell>
                              </TableRow>
                            ))}
                          </TableBody>
                        </Table>
                      )}
                    </div>
                  </div>

                  {/* 详情面板 */}
                  <div className="lg:col-span-1">
                    <Card>
                      <CardHeader>
                        <CardTitle className="text-base">节点详情</CardTitle>
                      </CardHeader>
                      <CardContent>
                        {selectedNode ? (
                          <div className="space-y-3">
                            <div>
                              <Label className="text-sm font-medium">名称</Label>
                              <p className="text-sm text-muted-foreground">{selectedNode.name}</p>
                            </div>
                            <div>
                              <Label className="text-sm font-medium">编码</Label>
                              <p className="text-sm text-muted-foreground">{selectedNode.code}</p>
                            </div>
                            <div>
                              <Label className="text-sm font-medium">类型</Label>
                              <p className="text-sm text-muted-foreground">{getNodeTypeLabel(selectedNode.type)}</p>
                            </div>
                            <div>
                              <Label className="text-sm font-medium">层级</Label>
                              <p className="text-sm text-muted-foreground">{selectedNode.level}</p>
                            </div>
                            {selectedNode.description && (
                              <div>
                                <Label className="text-sm font-medium">描述</Label>
                                <p className="text-sm text-muted-foreground">{selectedNode.description}</p>
                              </div>
                            )}
                            {selectedNode.type === "department" && selectedNode.employeeIdPrefix && (
                              <>
                                <div>
                                  <Label className="text-sm font-medium">工号前缀</Label>
                                  <p className="text-sm text-muted-foreground">{selectedNode.employeeIdPrefix}</p>
                                </div>
                                <div>
                                  <Label className="text-sm font-medium">工号规则</Label>
                                  <p className="text-sm text-muted-foreground">{selectedNode.employeeIdRule}</p>
                                </div>
                              </>
                            )}
                          </div>
                        ) : (
                          <p className="text-sm text-muted-foreground">请选择一个节点查看详情</p>
                        )}
                      </CardContent>
                    </Card>
                  </div>
                </div>
              </CardContent>
            </Card>
          </TabsContent>

          {/* 工号设置 */}
          <TabsContent value="settings" className="space-y-6">
            <Card>
              <CardHeader>
                <CardTitle>部门工号规则设置</CardTitle>
                <CardDescription>
                  配置各部门的员工工号生成规则
                </CardDescription>
              </CardHeader>
              <CardContent>
                <div className="space-y-4">
                  {getFlatList(organizationData)
                    .filter(node => node.type === "department")
                    .map(dept => (
                      <Card key={dept.id}>
                        <CardHeader className="pb-3">
                          <CardTitle className="text-base">{dept.name}</CardTitle>
                        </CardHeader>
                        <CardContent>
                          <div className="grid grid-cols-2 gap-4">
                            <div className="space-y-2">
                              <Label>部门编码</Label>
                              <Input value={dept.code} readOnly />
                            </div>
                            <div className="space-y-2">
                              <Label>工号前缀</Label>
                              <Input value={dept.employeeIdPrefix || ""} readOnly />
                            </div>
                            <div className="space-y-2">
                              <Label>工号规则示例</Label>
                              <Input value={dept.employeeIdRule || ""} readOnly />
                            </div>
                            <div className="space-y-2">
                              <Label>当前状态</Label>
                              <div className="flex items-center gap-2">
                                <span className="text-xs px-2 py-1 bg-green-100 text-green-800 rounded">
                                  正常
                                </span>
                                <Button variant="outline" size="sm">
                                  <Settings className="h-4 w-4 mr-2" />
                                  配置
                                </Button>
                              </div>
                            </div>
                          </div>
                        </CardContent>
                      </Card>
                    ))}
                </div>
              </CardContent>
            </Card>
          </TabsContent>
        </Tabs>

        {/* 编辑对话框 */}
        <Dialog open={showEditDialog} onOpenChange={setShowEditDialog}>
          <DialogContent className="max-w-2xl">
            <DialogHeader>
              <DialogTitle>编辑组织节点</DialogTitle>
              <DialogDescription>
                修改组织架构节点信息
              </DialogDescription>
            </DialogHeader>
            <div className="grid gap-4 py-4">
              <div className="grid grid-cols-2 gap-4">
                <div className="space-y-2">
                  <Label htmlFor="edit-name">节点名称 *</Label>
                  <Input
                    id="edit-name"
                    value={editNode.name || ""}
                    onChange={(e) => setEditNode(prev => ({ ...prev, name: e.target.value }))}
                  />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="edit-code">节点编码 *</Label>
                  <Input
                    id="edit-code"
                    value={editNode.code || ""}
                    onChange={(e) => setEditNode(prev => ({ ...prev, code: e.target.value }))}
                  />
                </div>
              </div>
              <div className="space-y-2">
                <Label htmlFor="edit-description">描述</Label>
                <Input
                  id="edit-description"
                  value={editNode.description || ""}
                  onChange={(e) => setEditNode(prev => ({ ...prev, description: e.target.value }))}
                />
              </div>
              {editNode.type === "department" && (
                <div className="grid grid-cols-2 gap-4">
                  <div className="space-y-2">
                    <Label htmlFor="edit-prefix">工号前缀</Label>
                    <Input
                      id="edit-prefix"
                      value={editNode.employeeIdPrefix || ""}
                      onChange={(e) => setEditNode(prev => ({ ...prev, employeeIdPrefix: e.target.value }))}
                    />
                  </div>
                  <div className="space-y-2">
                    <Label htmlFor="edit-rule">工号规则示例</Label>
                    <Input
                      id="edit-rule"
                      value={editNode.employeeIdRule || ""}
                      onChange={(e) => setEditNode(prev => ({ ...prev, employeeIdRule: e.target.value }))}
                    />
                  </div>
                </div>
              )}
            </div>
            <DialogFooter>
              <Button variant="outline" onClick={() => setShowEditDialog(false)}>
                取消
              </Button>
              <Button onClick={handleSaveEdit}>保存修改</Button>
            </DialogFooter>
          </DialogContent>
        </Dialog>

        {/* 删除确认对话框 */}
        <Dialog open={showDeleteDialog} onOpenChange={setShowDeleteDialog}>
          <DialogContent className="max-w-md">
            <DialogHeader>
              <div className="flex items-center gap-3">
                <div className="flex h-12 w-12 items-center justify-center rounded-full bg-red-100">
                  <AlertTriangle className="h-6 w-6 text-red-600" />
                </div>
                <div>
                  <DialogTitle className="text-left">确认删除</DialogTitle>
                  <DialogDescription className="text-left">
                    此操作不可撤销，请谨慎操作
                  </DialogDescription>
                </div>
              </div>
            </DialogHeader>

            <div className="py-4">
              <div className="rounded-lg bg-muted p-4">
                <div className="flex items-center gap-3">
                  <Building2 className="h-5 w-5 text-muted-foreground" />
                  <div>
                    <p className="font-medium">{nodeToDelete?.name}</p>
                    <div className="flex items-center gap-2 text-sm text-muted-foreground">
                      <span>编码：{nodeToDelete?.code}</span>
                      <span>•</span>
                      <span>类型：{nodeToDelete ? getNodeTypeLabel(nodeToDelete.type) : ''}</span>
                    </div>
                    {nodeToDelete?.description && (
                      <p className="text-sm text-muted-foreground mt-1">
                        {nodeToDelete.description}
                      </p>
                    )}
                  </div>
                </div>
              </div>

              <p className="mt-4 text-sm text-muted-foreground">
                您确定要删除这个节点吗？删除后无法恢复。
              </p>
            </div>

            <DialogFooter className="gap-2">
              <Button variant="outline" onClick={cancelDelete}>
                取消
              </Button>
              <Button
                variant="destructive"
                onClick={confirmDelete}
                className="bg-red-600 hover:bg-red-700 focus:ring-red-600"
              >
                确认删除
              </Button>
            </DialogFooter>
          </DialogContent>
        </Dialog>
      </div>
    </div>
  );
}