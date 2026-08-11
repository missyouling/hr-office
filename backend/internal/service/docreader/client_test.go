package docreader

import (
	"context"
	"testing"
	"time"
)

// TestNewClient 验证客户端创建与默认配置
func TestNewClient(t *testing.T) {
	// 懒连接模式：NewClient 始终成功，不验证地址可达性
	t.Run("正常地址", func(t *testing.T) {
		c := NewClient("docreader:50052")
		if c == nil {
			t.Fatal("NewClient 返回 nil")
		}
		if c.baseURL != "http://docreader:50052" {
			t.Errorf("baseURL = %q, want %q", c.baseURL, "http://docreader:50052")
		}
		if c.httpClient == nil {
			t.Fatal("httpClient 为 nil")
		}
		if c.httpClient.Timeout != 120*time.Second {
			t.Errorf("timeout = %v, want 120s", c.httpClient.Timeout)
		}
	})

	t.Run("IP地址", func(t *testing.T) {
		c := NewClient("127.0.0.1:50052")
		if c.baseURL != "http://127.0.0.1:50052" {
			t.Errorf("baseURL = %q, want %q", c.baseURL, "http://127.0.0.1:50052")
		}
	})

	t.Run("空地址", func(t *testing.T) {
		c := NewClient("")
		if c.baseURL != "http://" {
			t.Errorf("baseURL = %q, want %q", c.baseURL, "http://")
		}
	})
}

// TestParseRequest 验证请求结构体字段
func TestParseRequest(t *testing.T) {
	req := ParseRequest{
		FilePath: "/data/test.docx",
		FileType: "docx",
	}

	if req.FilePath != "/data/test.docx" {
		t.Errorf("FilePath = %q, want %q", req.FilePath, "/data/test.docx")
	}
	if req.FileType != "docx" {
		t.Errorf("FileType = %q, want %q", req.FileType, "docx")
	}

	// 验证零值
	zeroReq := ParseRequest{}
	if zeroReq.FilePath != "" {
		t.Error("零值 FilePath 应为空字符串")
	}
	if zeroReq.FileType != "" {
		t.Error("零值 FileType 应为空字符串")
	}
}

// TestParseResultFields 验证解析结果结构体字段
func TestParseResultFields(t *testing.T) {
	result := ParseResult{
		FullText: "## 第一章\n内容文本",
		Sections: []ParseSection{
			{Title: "第一章", Content: "内容文本", Level: 2},
		},
		Metadata: map[string]string{"author": "张三"},
		Duration: 1.5,
	}

	if result.FullText == "" {
		t.Error("FullText 不应为空")
	}
	if len(result.Sections) != 1 {
		t.Errorf("Sections len = %d, want 1", len(result.Sections))
	}
	if result.Sections[0].Title != "第一章" {
		t.Errorf("Section[0].Title = %q, want %q", result.Sections[0].Title, "第一章")
	}
	if result.Sections[0].Level != 2 {
		t.Errorf("Section[0].Level = %d, want 2", result.Sections[0].Level)
	}
	if v, ok := result.Metadata["author"]; !ok || v != "张三" {
		t.Errorf("Metadata[author] = %q, want %q", v, "张三")
	}
	if result.Duration != 1.5 {
		t.Errorf("Duration = %f, want 1.5", result.Duration)
	}
}

// TestClientHealth_NotConnected 验证未连接时的健康检查行为
func TestClientHealth_NotConnected(t *testing.T) {
	// 指向一个几乎确定不存在的地址
	c := NewClient("127.255.255.255:1")
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	err := c.Health(ctx)
	if err == nil {
		t.Error("期望健康检查失败（地址不可达）")
	}
}

// TestClientClose 验证 Close 不返回错误
func TestClientClose(t *testing.T) {
	c := NewClient("docreader:50052")
	if err := c.Close(); err != nil {
		t.Errorf("Close 返回错误: %v", err)
	}
}
