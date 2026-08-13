package service

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"siapp/internal/models"
)

func setupChatTestService(t *testing.T, endpoint string) (*ChatService, *gorm.DB) {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开测试数据库失败: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("获取测试数据库连接失败: %v", err)
	}
	sqlDB.SetMaxOpenConns(1)
	if err := db.AutoMigrate(&models.ChatSession{}, &models.ChatMessage{}, &models.ModelConfig{}, &models.ModelUsageLog{}); err != nil {
		t.Fatalf("迁移测试表失败: %v", err)
	}
	if err := db.Create(&models.ModelConfig{
		ConfigType: "llm", Provider: "test", ModelName: "test-model", APIEndpoint: endpoint, Enabled: true,
	}).Error; err != nil {
		t.Fatalf("创建模型配置失败: %v", err)
	}

	embeddingService := NewEmbeddingService(db)
	return NewChatService(db, NewRetrievalService(db, embeddingService)), db
}

func newStreamLLMServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"测试回答\"}}]}\n\ndata: [DONE]\n"))
	}))
}

func sseEvents(t *testing.T, body string) []map[string]json.RawMessage {
	t.Helper()

	var events []map[string]json.RawMessage
	for _, line := range strings.Split(body, "\n") {
		data, found := strings.CutPrefix(line, "data: ")
		if !found {
			continue
		}
		var event map[string]json.RawMessage
		if err := json.Unmarshal([]byte(data), &event); err != nil {
			t.Fatalf("解析 SSE 事件失败: %v", err)
		}
		events = append(events, event)
	}
	return events
}

func eventByType(t *testing.T, events []map[string]json.RawMessage, eventType string) map[string]json.RawMessage {
	t.Helper()
	for _, event := range events {
		var currentType string
		if err := json.Unmarshal(event["type"], &currentType); err == nil && currentType == eventType {
			return event
		}
	}
	t.Fatalf("未找到 %s SSE 事件", eventType)
	return nil
}

func TestStreamChat_DoneEventContainsSavedAssistantMessageID(t *testing.T) {
	server := newStreamLLMServer()
	defer server.Close()
	svc, db := setupChatTestService(t, server.URL)

	response := httptest.NewRecorder()
	svc.StreamChat(response, 1, "stream-success", "测试问题")

	done := eventByType(t, sseEvents(t, response.Body.String()), "done")
	messageID, exists := done["message_id"]
	if !exists {
		t.Fatal("done 事件缺少 message_id")
	}
	var actualID uint
	if err := json.Unmarshal(messageID, &actualID); err != nil {
		t.Fatalf("解析 message_id 失败: %v", err)
	}
	var message models.ChatMessage
	if err := db.Where("session_id = ? AND role = ?", "stream-success", "assistant").First(&message).Error; err != nil {
		t.Fatalf("查询助手消息失败: %v", err)
	}
	if actualID != message.ID {
		t.Errorf("done message_id = %d，实际助手消息 ID = %d", actualID, message.ID)
	}
}

func TestStreamChat_AssistantSaveFailureDoesNotSendMessageID(t *testing.T) {
	server := newStreamLLMServer()
	defer server.Close()
	svc, db := setupChatTestService(t, server.URL)
	db.Callback().Create().Before("gorm:create").Register("fail_assistant_message", func(tx *gorm.DB) {
		message, ok := tx.Statement.Dest.(*models.ChatMessage)
		if ok && message.Role == "assistant" {
			tx.AddError(errors.New("助手消息保存失败"))
		}
	})

	response := httptest.NewRecorder()
	svc.StreamChat(response, 1, "stream-save-failure", "测试问题")

	events := sseEvents(t, response.Body.String())
	errorEvent := eventByType(t, events, "error")
	if _, exists := errorEvent["message_id"]; exists {
		t.Error("保存失败的 error 事件不应包含 message_id")
	}
	for _, event := range events {
		if _, exists := event["message_id"]; exists {
			t.Errorf("保存失败路径不应返回 message_id，事件: %s", event["type"])
		}
	}
}
