package service

import (
	"sort"
	"strings"

	"gorm.io/gorm"
)

// FolderNode 文件夹树节点。
// 每个节点代表知识库中的一个目录，包含直接位于该目录下的文档数以及递归汇总后的总数。
type FolderNode struct {
	Path          string        `json:"path"`           // 完整相对路径，如 "2026/合同"
	Name          string        `json:"name"`           // 仅文件夹名，如 "合同"
	DocumentCount int64         `json:"document_count"` // 直接存于本文件夹的文档数
	TotalCount    int64         `json:"total_count"`    // 含所有后代文件夹的总数
	Children      []*FolderNode `json:"children"`       // 子文件夹
}

// FolderTreeResult 文件夹树构建结果，包含根目录统计与顶层文件夹列表。
type FolderTreeResult struct {
	RootDocumentCount  int64         `json:"root_document_count"`  // 根目录文档数
	TotalDocumentCount int64         `json:"total_document_count"` // 知识库总文档数
	Folders            []*FolderNode `json:"folders"`              // 顶层文件夹列表
}

// folderCountItem 数据库查询返回的扁平路径计数。
type folderCountItem struct {
	FolderPath string `gorm:"column:folder_path"`
	Count      int64  `gorm:"column:cnt"`
}

// BuildFolderTree 根据用户与分类编码，将 documents 表中的 folder_path 计数聚合为文件夹树。
//
// 主要步骤：
//  1. 查询所有非空 folder_path 的计数，并单独统计根目录文档数；
//  2. 归一化路径，按层级构建节点映射，自动物化中间文件夹；
//  3. 按深度从深到浅汇总各节点的 TotalCount；
//  4. 同级子节点按名称排序。
func BuildFolderTree(db *gorm.DB, userID uint, categoryCode string) (*FolderTreeResult, error) {
	var items []folderCountItem
	err := db.Raw(
		`SELECT folder_path, COUNT(*) as cnt FROM documents `+
			`WHERE user_id = ? AND category_code = ? AND folder_path IS NOT NULL AND folder_path != '' `+
			`GROUP BY folder_path`,
		userID, categoryCode,
	).Scan(&items).Error
	if err != nil {
		return nil, err
	}

	var rootCount int64
	err = db.Raw(
		`SELECT COUNT(*) as cnt FROM documents `+
			`WHERE user_id = ? AND category_code = ? AND (folder_path IS NULL OR folder_path = '')`,
		userID, categoryCode,
	).Scan(&rootCount).Error
	if err != nil {
		return nil, err
	}

	// nodeMap 以归一化路径为键存储节点，根目录使用空字符串作为键。
	nodeMap := make(map[string]*FolderNode)
	nodeMap[""] = &FolderNode{
		Path:          "",
		Name:          "",
		DocumentCount: rootCount,
		TotalCount:    rootCount,
		Children:      make([]*FolderNode, 0),
	}

	// 第一步：为每个存在文档的路径创建节点，并沿途物化中间文件夹。
	for _, item := range items {
		path := normalizeFolderPath(item.FolderPath)
		if path == "" {
			// 已计入根目录，跳过。
			continue
		}

		parts := strings.Split(path, "/")
		for i := range parts {
			segmentPath := strings.Join(parts[:i+1], "/")
			if _, exists := nodeMap[segmentPath]; !exists {
				nodeMap[segmentPath] = &FolderNode{
					Path:          segmentPath,
					Name:          parts[i],
					DocumentCount: 0,
					TotalCount:    0,
					Children:      make([]*FolderNode, 0),
				}
			}
		}
		nodeMap[path].DocumentCount += item.Count
		nodeMap[path].TotalCount += item.Count
	}

	// 第二步：按路径深度从深到浅，将 TotalCount 汇总到父节点。
	paths := make([]string, 0, len(nodeMap))
	for p := range nodeMap {
		if p == "" {
			continue
		}
		paths = append(paths, p)
	}
	sort.Slice(paths, func(i, j int) bool {
		return pathDepth(paths[i]) > pathDepth(paths[j])
	})

	for _, p := range paths {
		node := nodeMap[p]
		parentPath := parentFolderPath(p)
		parent, exists := nodeMap[parentPath]
		if exists {
			parent.TotalCount += node.TotalCount
		}
	}

	// 第三步：建立父子关系。
	for _, p := range paths {
		node := nodeMap[p]
		parentPath := parentFolderPath(p)
		parent, exists := nodeMap[parentPath]
		if exists {
			parent.Children = append(parent.Children, node)
		}
	}

	// 第四步：同级子节点按名称排序。
	for _, node := range nodeMap {
		sort.Slice(node.Children, func(i, j int) bool {
			return node.Children[i].Name < node.Children[j].Name
		})
	}

	root := nodeMap[""]
	totalCount := root.TotalCount

	return &FolderTreeResult{
		RootDocumentCount:  rootCount,
		TotalDocumentCount: totalCount,
		Folders:            root.Children,
	}, nil
}

// normalizeFolderPath 归一化文件夹路径：统一为正斜杠、去除首尾分隔符、剔除 . 与 .. 段。
func normalizeFolderPath(path string) string {
	if path == "" {
		return ""
	}
	// 统一分隔符。
	path = strings.ReplaceAll(path, "\\", "/")
	parts := strings.Split(path, "/")
	filtered := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" || part == "." {
			continue
		}
		if part == ".." {
			// 回到上一级，若已在根目录则忽略。
			if len(filtered) > 0 {
				filtered = filtered[:len(filtered)-1]
			}
			continue
		}
		filtered = append(filtered, part)
	}
	return strings.Join(filtered, "/")
}

// pathDepth 返回路径的层级深度，顶层文件夹深度为 1。
func pathDepth(path string) int {
	if path == "" {
		return 0
	}
	return strings.Count(path, "/") + 1
}

// parentFolderPath 返回给定路径的父路径，顶层文件夹的父路径为空字符串（代表根目录）。
func parentFolderPath(path string) string {
	if path == "" {
		return ""
	}
	idx := strings.LastIndex(path, "/")
	if idx < 0 {
		return ""
	}
	return path[:idx]
}
