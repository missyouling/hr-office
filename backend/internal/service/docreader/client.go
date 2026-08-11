// Package docreader 提供 WeKnora docreader 文档解析微服务的 HTTP 客户端。
//
// docreader 是一个 Python 独立微服务，负责将 Word/Excel/PPT/Markdown/PDF
// 等多格式文档解析为结构化 markdown 文本。hr-office 通过此客户端以 HTTP REST
// 方式调用 docreader 的解析接口，避免引入 gRPC proto 编译链。
package docreader

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// ParseRequest 文档解析请求
type ParseRequest struct {
	FilePath string // 本地文件路径
	FileType string // 文件后缀（如 pdf/docx/xlsx/pptx/md/txt）
}

// ParseResult 文档解析结果
type ParseResult struct {
	FullText string            `json:"full_text"` // 全文纯文本（markdown 格式）
	Sections []ParseSection    `json:"sections"`  // 章节/段落列表
	Metadata map[string]string `json:"metadata"`  // 元数据（标题/作者/页数等）
	Duration float64           `json:"duration"`  // 解析耗时（秒）
}

// ParseSection 文档章节/段落
type ParseSection struct {
	Title   string `json:"title"`   // 章节标题
	Content string `json:"content"` // 章节文本内容
	Level   int    `json:"level"`   // 层级（1=一级标题，2=二级标题...）
}

// Client docreader HTTP 客户端
// 采用懒连接模式：NewClient 不验证连接，首次 Parse/Health 调用时才建立真实连接。
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// NewClient 创建 docreader 客户端
// addr 格式: "host:port"（如 "docreader:50052"），内部拼装为 http://addr。
// 创建时不验证连接（懒连接），允许 docreader 服务后启动。
func NewClient(addr string) *Client {
	return &Client{
		baseURL: "http://" + addr,
		httpClient: &http.Client{
			Timeout: 120 * time.Second, // 大文件解析可能耗时较长
		},
	}
}

// Parse 解析文档文件
// 将文件以 multipart/form-data 方式上传到 docreader 的 /parse 端点，
// 返回结构化解析结果。
func (c *Client) Parse(ctx context.Context, req ParseRequest) (*ParseResult, error) {
	fileContent, err := os.ReadFile(req.FilePath)
	if err != nil {
		return nil, fmt.Errorf("读取文件失败: %w", err)
	}

	// 构建 multipart/form-data 请求体
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	// file 字段携带文件内容和文件名
	fileName := filepath.Base(req.FilePath)
	part, err := writer.CreateFormFile("file", fileName)
	if err != nil {
		return nil, fmt.Errorf("创建表单文件字段失败: %w", err)
	}
	if _, err := part.Write(fileContent); err != nil {
		return nil, fmt.Errorf("写入表单文件数据失败: %w", err)
	}

	// 可选：传递文件类型提示
	if req.FileType != "" {
		_ = writer.WriteField("file_type", req.FileType)
	}

	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("关闭表单写入器失败: %w", err)
	}

	// 发起 POST 请求
	httpReq, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		c.baseURL+"/parse",
		&buf,
	)
	if err != nil {
		return nil, fmt.Errorf("创建 HTTP 请求失败: %w", err)
	}
	httpReq.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("HTTP 请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("docreader 返回状态 %d: %s", resp.StatusCode, string(body))
	}

	var result ParseResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("解析 docreader 响应失败: %w", err)
	}

	return &result, nil
}

// Health 健康检查
// 调用 docreader 的 GET /health 端点，确认服务可用。
func (c *Client) Health(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/health", nil)
	if err != nil {
		return fmt.Errorf("创建健康检查请求失败: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("健康检查失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("docreader 健康检查返回状态 %d", resp.StatusCode)
	}
	return nil
}

// Close 关闭客户端（懒连接模式下为空操作）
func (c *Client) Close() error {
	c.httpClient.CloseIdleConnections()
	return nil
}
