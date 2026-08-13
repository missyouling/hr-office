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
	if err := db.AutoMigrate(
		&models.ChatSession{}, &models.ChatMessage{}, &models.ModelConfig{}, &models.ModelUsageLog{},
		&models.User{}, &models.Role{}, &models.UserRole{},
		&models.KnowledgeBase{}, &models.KBFieldMask{}, &models.KBAccessRule{},
		&models.Document{}, &models.DocumentChunk{},
	); err != nil {
		t.Fatalf("迁移测试表失败: %v", err)
	}
	if err := db.Create(&models.User{ID: 1, Username: "chatuser1", Email: "chatuser1@test.local", FullName: "聊天用户1"}).Error; err != nil {
		t.Fatalf("创建默认用户失败: %v", err)
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
	svc.StreamChat(response, 1, "stream-success", "测试问题", 0)

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
	svc.StreamChat(response, 1, "stream-save-failure", "测试问题", 0)

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

// ============================================================
// P9.2 脱敏与 KB 范围测试
// ============================================================

// seedChatUser 创建测试用户（自动分配 ID，避免与默认用户 1 冲突）
func seedChatUser(t *testing.T, db *gorm.DB, username string) models.User {
	t.Helper()
	user := models.User{Username: username, Email: username + "@test.local", FullName: username}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("创建用户失败: %v", err)
	}
	return user
}

// seedChatKB 创建私有知识库
func seedChatKB(t *testing.T, db *gorm.DB, name string, ownerID uint) models.KnowledgeBase {
	t.Helper()
	kb := models.KnowledgeBase{Name: name, Visibility: "private", SourceModule: "custom", OwnerID: &ownerID}
	if err := db.Create(&kb).Error; err != nil {
		t.Fatalf("创建知识库失败: %v", err)
	}
	return kb
}

// seedChatDocument 插入影子文档（模拟 ingest 产物）
func seedChatDocument(t *testing.T, db *gorm.DB, userID uint, kbID uint, code, fileName, content string) {
	t.Helper()
	doc := models.Document{
		UserID: userID, DocumentCode: code, FileName: fileName, ContentText: content,
		SourceType: "custom", SourceID: 1, SourceKBID: &kbID,
		Status: "active", OCRStatus: "completed",
	}
	if err := db.Create(&doc).Error; err != nil {
		t.Fatalf("创建文档失败: %v", err)
	}
}

// newMaskingLLMServer 非流式 LLM mock：返回包含身份证号的答案
func newMaskingLLMServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"员工身份证号是 110101199001011234"}}]}`))
	}))
}

// newMaskingStreamLLMServer 流式 LLM mock：分片返回包含身份证号的答案
func newMaskingStreamLLMServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"员工身份证号是 1101011990\"}}]}\n\n"))
		_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"01011234\"}}]}\n\n"))
		_, _ = w.Write([]byte("data: [DONE]\n"))
	}))
}

// TestChat_Masking 非流式 chat 脱敏：Sources 与最终答案均不得泄露原文
func TestChat_Masking(t *testing.T) {
	server := newMaskingLLMServer()
	defer server.Close()
	svc, db := setupChatTestService(t, server.URL)

	user := seedChatUser(t, db, "maskuser")
	kb := seedChatKB(t, db, "脱敏知识库", user.ID)
	// 业务字段规则 id_card front3back4（映射到 content/snippet）
	if err := db.Create(&models.KBFieldMask{
		KnowledgeBaseID: kb.ID, FieldName: "id_card", MaskPattern: "front3back4",
	}).Error; err != nil {
		t.Fatalf("创建脱敏规则失败: %v", err)
	}
	seedChatDocument(t, db, user.ID, kb.ID, "MASK-1", "员工档案", "脱敏测试内容 110101199001011234")

	resp, err := svc.Chat(user.ID, "mask-session", "脱敏测试", kb.ID)
	if err != nil {
		t.Fatalf("Chat 失败: %v", err)
	}

	// Sources.Snippet 必须被脱敏（业务字段规则映射到 snippet，不得返回原文）
	if len(resp.Sources) == 0 {
		t.Fatal("Chat 应返回检索来源")
	}
	if strings.Contains(resp.Sources[0].Snippet, "110101199001011234") {
		t.Errorf("Sources.Snippet 泄露原文: %s", resp.Sources[0].Snippet)
	}

	// 最终答案必须脱敏（防御层：模型复述的身份证号被脱敏）
	if strings.Contains(resp.Answer, "110101199001011234") {
		t.Errorf("Answer 泄露完整身份证号: %s", resp.Answer)
	}
}

// TestStreamChat_Masking SSE 流式增量脱敏：token 事件拼接后不得泄露完整身份证号
func TestStreamChat_Masking(t *testing.T) {
	server := newMaskingStreamLLMServer()
	defer server.Close()
	svc, db := setupChatTestService(t, server.URL)

	user := seedChatUser(t, db, "streammask")
	kb := seedChatKB(t, db, "流式脱敏知识库", user.ID)
	seedChatDocument(t, db, user.ID, kb.ID, "SMASK-1", "员工档案", "流式脱敏测试内容")

	response := httptest.NewRecorder()
	svc.StreamChat(response, user.ID, "stream-mask", "流式脱敏测试", kb.ID)

	// 拼接所有 token 事件，断言不含完整身份证号
	var streamed strings.Builder
	for _, event := range sseEvents(t, response.Body.String()) {
		var eventType string
		if err := json.Unmarshal(event["type"], &eventType); err != nil || eventType != "token" {
			continue
		}
		var content string
		if err := json.Unmarshal(event["content"], &content); err != nil {
			continue
		}
		streamed.WriteString(content)
	}
	if strings.Contains(streamed.String(), "110101199001011234") {
		t.Errorf("SSE 流式输出泄露完整身份证号: %s", streamed.String())
	}
}

// TestStreamChat_KBScope SSE 透传 kb_id：检索范围限定在指定 KB 内
func TestStreamChat_KBScope(t *testing.T) {
	server := newStreamLLMServer()
	defer server.Close()
	svc, db := setupChatTestService(t, server.URL)

	user1 := seedChatUser(t, db, "scopeuser1")
	user2 := seedChatUser(t, db, "scopeuser2")
	kb1 := seedChatKB(t, db, "范围知识库1", user1.ID)
	kb2 := seedChatKB(t, db, "范围知识库2", user2.ID)
	seedChatDocument(t, db, user1.ID, kb1.ID, "SCOPE-1", "员工-郭靖", "郭靖 降龙十八掌")
	seedChatDocument(t, db, user2.ID, kb2.ID, "SCOPE-2", "员工-杨过", "杨过 黯然销魂掌")

	response := httptest.NewRecorder()
	// 用户 1 指定 kb1 范围提问，检索只应命中 kb1 的文档
	svc.StreamChat(response, user1.ID, "stream-scope", "郭靖", kb1.ID)

	// 断言保存的 assistant 消息 sources 只含 kb1 内容
	var message models.ChatMessage
	if err := db.Where("session_id = ? AND role = ?", "stream-scope", "assistant").First(&message).Error; err != nil {
		t.Fatalf("查询助手消息失败: %v", err)
	}
	var sources []SearchResult
	if err := json.Unmarshal(message.Sources, &sources); err != nil {
		t.Fatalf("解析 sources 失败: %v", err)
	}
	if len(sources) == 0 {
		t.Fatal("SSE 应返回 kb1 范围内的检索来源")
	}
	for _, src := range sources {
		if src.KBID != kb1.ID {
			t.Errorf("SSE 检索越出 kb1 范围: KBID=%d", src.KBID)
		}
	}
}
