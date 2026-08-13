package service

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"gorm.io/gorm"

	"siapp/internal/models"
)

// ============================================================
// P9.2 字段级精准脱敏与安全失败测试
// ============================================================

// TestChat_Masking_Precise 业务字段规则精准替换：
// 仅替换敏感值，保留非敏感上下文；持久化 sources 不得含原文
func TestChat_Masking_Precise(t *testing.T) {
	server := newMaskingLLMServer()
	defer server.Close()
	svc, db := setupChatTestService(t, server.URL)

	user := seedChatUser(t, db, "preciseuser")
	kb := seedChatKB(t, db, "精准脱敏知识库", user.ID)
	// address 业务字段规则 front3back4（对 content/snippet 做字段级精准替换）
	if err := db.Create(&models.KBFieldMask{
		KnowledgeBaseID: kb.ID, FieldName: "address", MaskPattern: "front3back4",
	}).Error; err != nil {
		t.Fatalf("创建脱敏规则失败: %v", err)
	}
	seedChatDocument(t, db, user.ID, kb.ID, "PRECISE-1", "员工档案", "员工档案 家庭住址：北京市朝阳区望京SOHO，备注：正常")

	resp, err := svc.Chat(user.ID, "precise-session", "家庭住址", kb.ID)
	if err != nil {
		t.Fatalf("Chat 失败: %v", err)
	}

	if len(resp.Sources) == 0 {
		t.Fatal("Chat 应返回检索来源")
	}
	snippet := resp.Sources[0].Snippet
	// 不得泄露地址原文
	if strings.Contains(snippet, "北京市朝阳区望京SOHO") {
		t.Errorf("Sources.Snippet 泄露地址原文: %s", snippet)
	}
	// 必须保留非敏感上下文（不得整段破坏内容）
	if !strings.Contains(snippet, "员工档案") || !strings.Contains(snippet, "备注：正常") {
		t.Errorf("Sources.Snippet 丢失非敏感上下文: %s", snippet)
	}
	// 键值标签应保留（仅替换值部分）
	if !strings.Contains(snippet, "家庭住址") {
		t.Errorf("Sources.Snippet 丢失字段标签: %s", snippet)
	}

	// 持久化 sources 同样不得含原文
	var message models.ChatMessage
	if err := db.Where("session_id = ? AND role = ?", "precise-session", "assistant").First(&message).Error; err != nil {
		t.Fatalf("查询助手消息失败: %v", err)
	}
	var persisted []SearchResult
	if err := json.Unmarshal(message.Sources, &persisted); err != nil {
		t.Fatalf("解析持久化 sources 失败: %v", err)
	}
	if len(persisted) == 0 {
		t.Fatal("持久化 sources 为空")
	}
	if strings.Contains(persisted[0].Snippet, "北京市朝阳区望京SOHO") {
		t.Errorf("持久化 sources 泄露地址原文: %s", persisted[0].Snippet)
	}
}

// capturePromptLLMServer 非流式 LLM mock：捕获请求体中的 user prompt，返回含地址原文的答案
func capturePromptLLMServer(t *testing.T, captured *string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Messages []struct {
				Role    string `json:"role"`
				Content string `json:"content"`
			} `json:"messages"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		for _, m := range body.Messages {
			if m.Role == "user" {
				*captured = m.Content
				break
			}
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"该员工住址为北京市朝阳区望京SOHO"}}]}`))
	}))
}

// TestChat_PromptAndAnswer_FieldRules address + 自定义字段规则：
// 模型收到的 Prompt 必须是脱敏文本；最终答案复述原文也被防御层替换
func TestChat_PromptAndAnswer_FieldRules(t *testing.T) {
	var captured string
	server := capturePromptLLMServer(t, &captured)
	defer server.Close()
	svc, db := setupChatTestService(t, server.URL)

	user := seedChatUser(t, db, "promptuser")
	kb := seedChatKB(t, db, "字段规则知识库", user.ID)
	// address 与自定义字段 employee_no 规则
	for _, rule := range []struct{ field, pattern string }{
		{"address", "front3back4"},
		{"employee_no", "front3back4"},
	} {
		if err := db.Create(&models.KBFieldMask{
			KnowledgeBaseID: kb.ID, FieldName: rule.field, MaskPattern: rule.pattern,
		}).Error; err != nil {
			t.Fatalf("创建脱敏规则失败: %v", err)
		}
	}
	seedChatDocument(t, db, user.ID, kb.ID, "PROMPT-1", "员工档案",
		"员工档案 家庭住址：北京市朝阳区望京SOHO，工号 employee_no：E10001")

	resp, err := svc.Chat(user.ID, "prompt-session", "家庭住址", kb.ID)
	if err != nil {
		t.Fatalf("Chat 失败: %v", err)
	}

	// 1. 模型收到的 Prompt 必须是脱敏文本（不得含地址/工号原文）
	if strings.Contains(captured, "北京市朝阳区望京SOHO") || strings.Contains(captured, "E10001") {
		t.Errorf("Prompt 泄露原文: %s", captured)
	}
	if !strings.Contains(captured, "家庭住址") || !strings.Contains(captured, "employee_no") {
		t.Errorf("Prompt 应保留字段标签上下文: %s", captured)
	}

	// 2. 最终答案复述地址原文 → 防御层按 KB 字段规则替换
	if strings.Contains(resp.Answer, "北京市朝阳区望京SOHO") {
		t.Errorf("Answer 泄露地址原文: %s", resp.Answer)
	}
}

// TestChat_MaskRuleQueryFailure 脱敏规则查询失败：Chat 必须安全失败，不得返回原文
func TestChat_MaskRuleQueryFailure(t *testing.T) {
	server := newMaskingLLMServer()
	defer server.Close()
	svc, db := setupChatTestService(t, server.URL)

	user := seedChatUser(t, db, "failmaskuser")
	kb := seedChatKB(t, db, "规则查询失败知识库", user.ID)
	seedChatDocument(t, db, user.ID, kb.ID, "FAILMASK-1", "员工档案", "脱敏测试内容 110101199001011234")

	// 注入 kb_field_masks 查询失败（ApplyMaskToResults 路径）
	db.Callback().Query().Before("gorm:query").Register("fail_mask_rule_query", func(tx *gorm.DB) {
		if tx.Statement.Schema != nil && tx.Statement.Schema.Table == "kb_field_masks" {
			tx.AddError(errors.New("注入规则查询失败"))
		}
	})
	defer func() {
		_ = db.Callback().Query().Before("gorm:query").Remove("fail_mask_rule_query")
	}()

	_, err := svc.Chat(user.ID, "failmask-session", "脱敏测试", kb.ID)
	if err == nil {
		t.Fatal("规则查询失败时 Chat 应返回错误（安全失败，不得返回原文）")
	}
}

// fieldRuleStreamLLMServer 流式 LLM mock：单片返回含完整地址原文的答案
// （地址完整落在同一片内，保证敏感值映射可命中；跨片切分属流式已知局限）
func fieldRuleStreamLLMServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"该员工现住址为北京市朝阳区望京SOHO，请知悉。\"}}]}\n\n"))
		_, _ = w.Write([]byte("data: [DONE]\n"))
	}))
}

// TestStreamChat_FieldRuleAnswer SSE 增量：字段规则提取的敏感值映射
// 用于流式增量脱敏，拼接结果不得含地址原文
func TestStreamChat_FieldRuleAnswer(t *testing.T) {
	server := fieldRuleStreamLLMServer()
	defer server.Close()
	svc, db := setupChatTestService(t, server.URL)

	user := seedChatUser(t, db, "streamfielduser")
	kb := seedChatKB(t, db, "流式字段规则知识库", user.ID)
	if err := db.Create(&models.KBFieldMask{
		KnowledgeBaseID: kb.ID, FieldName: "address", MaskPattern: "front3back4",
	}).Error; err != nil {
		t.Fatalf("创建脱敏规则失败: %v", err)
	}
	seedChatDocument(t, db, user.ID, kb.ID, "STREAMFIELD-1", "员工档案", "员工档案 家庭住址：北京市朝阳区望京SOHO")

	response := httptest.NewRecorder()
	svc.StreamChat(response, user.ID, "stream-field", "家庭住址", kb.ID)

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
	if strings.Contains(streamed.String(), "北京市朝阳区望京SOHO") {
		t.Errorf("SSE 流式输出泄露地址原文: %s", streamed.String())
	}
}

// chunkedStreamLLMServer 分片流式 LLM mock：按给定 chunks 逐片返回 content
// （用于验证长敏感值跨多个 SSE token 时不泄露原文）
func chunkedStreamLLMServer(chunks ...string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		for _, c := range chunks {
			payload, _ := json.Marshal(map[string]any{
				"choices": []map[string]any{
					{"delta": map[string]any{"content": c}},
				},
			})
			_, _ = w.Write([]byte("data: " + string(payload) + "\n\n"))
		}
		_, _ = w.Write([]byte("data: [DONE]\n"))
	}))
}

// streamTokenContents 提取 SSE 响应中所有 type=token 事件的 content（按输出顺序）
func streamTokenContents(t *testing.T, body string) []string {
	t.Helper()
	var contents []string
	for _, event := range sseEvents(t, body) {
		var eventType string
		if err := json.Unmarshal(event["type"], &eventType); err != nil || eventType != "token" {
			continue
		}
		var content string
		if err := json.Unmarshal(event["content"], &content); err != nil {
			continue
		}
		contents = append(contents, content)
	}
	return contents
}

// TestStreamChat_LongAddressNoLeak 长 address（>19 字符）跨多个 SSE token：
// 每个已输出 token 不含原文、拼接不含原文、持久化 Content/Sources 不含原文
func TestStreamChat_LongAddressNoLeak(t *testing.T) {
	const addr = "北京市朝阳区望京街道望京SOHO塔楼A座第66层6601室"
	server := chunkedStreamLLMServer(
		"该员工现住址为北京市朝阳区望京街道望京",
		"SOHO塔楼A座第66层6601室，请知悉",
	)
	defer server.Close()
	svc, db := setupChatTestService(t, server.URL)

	user := seedChatUser(t, db, "longaddruser")
	kb := seedChatKB(t, db, "长地址知识库", user.ID)
	if err := db.Create(&models.KBFieldMask{
		KnowledgeBaseID: kb.ID, FieldName: "address", MaskPattern: "front3back4",
	}).Error; err != nil {
		t.Fatalf("创建脱敏规则失败: %v", err)
	}
	seedChatDocument(t, db, user.ID, kb.ID, "LONGADDR-1", "员工档案",
		"员工档案 家庭住址："+addr+"，备注：正常")

	response := httptest.NewRecorder()
	svc.StreamChat(response, user.ID, "longaddr-session", "家庭住址", kb.ID)

	tokens := streamTokenContents(t, response.Body.String())
	joined := strings.Join(tokens, "")
	wantMasked := applyFront3Back4(addr)
	// 每个已输出 token 均不含地址原文
	for i, tok := range tokens {
		if strings.Contains(tok, addr) {
			t.Errorf("第 %d 个已输出 token 泄露地址原文: %q", i, tok)
		}
	}
	// 拼接输出不含地址原文
	if strings.Contains(joined, addr) {
		t.Errorf("SSE 拼接输出泄露地址原文: %q", joined)
	}
	// 必须含完整脱敏值（完整映射替换，而非截断前缀后泄露尾部）
	if !strings.Contains(joined, wantMasked) {
		t.Errorf("SSE 输出应含完整脱敏值 %q（防截断前缀泄露尾部），实际: %q", wantMasked, joined)
	}
	// 地址被脱敏而非整体丢弃（front3back4 保留后4字符：addr 末 4 字符为 "601室"）
	if !strings.Contains(joined, "601室") {
		t.Errorf("SSE 输出应保留地址脱敏后的后4字符，实际: %q", joined)
	}

	// 持久化 assistant Content 与 Sources 不含原文
	var msg models.ChatMessage
	if err := db.Where("session_id = ? AND role = ?", "longaddr-session", "assistant").First(&msg).Error; err != nil {
		t.Fatalf("查询助手消息失败: %v", err)
	}
	if strings.Contains(msg.Content, addr) {
		t.Errorf("持久化 assistant Content 泄露地址原文: %s", msg.Content)
	}
	var persisted []SearchResult
	if err := json.Unmarshal(msg.Sources, &persisted); err != nil {
		t.Fatalf("解析持久化 sources 失败: %v", err)
	}
	for i, s := range persisted {
		if strings.Contains(s.Snippet, addr) {
			t.Errorf("持久化 sources[%d] 泄露地址原文: %s", i, s.Snippet)
		}
		if !strings.Contains(s.Snippet, wantMasked) {
			t.Errorf("持久化 sources[%d] 应含完整脱敏值（防截断前缀泄露尾部）: %s", i, s.Snippet)
		}
	}
}

// TestStreamChat_LongCustomFieldNoLeak 长自定义字段值跨多个 SSE token：
// 每个已输出 token 不含原文、拼接不含原文、持久化 Content/Sources 不含原文
func TestStreamChat_LongCustomFieldNoLeak(t *testing.T) {
	const customVal = "这是一个超长的自定义字段值内部包含敏感编号和详细信息需要完整脱敏处理"
	server := chunkedStreamLLMServer(
		"该员工备注为这是一个超长的自定义字段值内部包含敏感",
		"编号和详细信息需要完整脱敏处理，请知悉",
	)
	defer server.Close()
	svc, db := setupChatTestService(t, server.URL)

	user := seedChatUser(t, db, "longcustomuser")
	kb := seedChatKB(t, db, "长自定义字段知识库", user.ID)
	if err := db.Create(&models.KBFieldMask{
		KnowledgeBaseID: kb.ID, FieldName: "remark", MaskPattern: "front3back4",
	}).Error; err != nil {
		t.Fatalf("创建脱敏规则失败: %v", err)
	}
	seedChatDocument(t, db, user.ID, kb.ID, "LONGCUSTOM-1", "员工档案",
		"员工档案 备注 remark："+customVal)

	response := httptest.NewRecorder()
	svc.StreamChat(response, user.ID, "longcustom-session", "备注", kb.ID)

	tokens := streamTokenContents(t, response.Body.String())
	joined := strings.Join(tokens, "")
	wantMasked := applyFront3Back4(customVal)
	for i, tok := range tokens {
		if strings.Contains(tok, customVal) {
			t.Errorf("第 %d 个已输出 token 泄露自定义字段原文: %q", i, tok)
		}
	}
	if strings.Contains(joined, customVal) {
		t.Errorf("SSE 拼接输出泄露自定义字段原文: %q", joined)
	}
	// 必须含完整脱敏值（完整映射替换，而非截断前缀后泄露尾部）
	if !strings.Contains(joined, wantMasked) {
		t.Errorf("SSE 输出应含完整脱敏值 %q（防截断前缀泄露尾部），实际: %q", wantMasked, joined)
	}
	// front3back4 保留后4字符
	if !strings.Contains(joined, "脱敏处理") {
		t.Errorf("SSE 输出应保留自定义字段脱敏后的后4字符，实际: %q", joined)
	}

	var msg models.ChatMessage
	if err := db.Where("session_id = ? AND role = ?", "longcustom-session", "assistant").First(&msg).Error; err != nil {
		t.Fatalf("查询助手消息失败: %v", err)
	}
	if strings.Contains(msg.Content, customVal) {
		t.Errorf("持久化 assistant Content 泄露自定义字段原文: %s", msg.Content)
	}
	var persisted []SearchResult
	if err := json.Unmarshal(msg.Sources, &persisted); err != nil {
		t.Fatalf("解析持久化 sources 失败: %v", err)
	}
	for i, s := range persisted {
		if strings.Contains(s.Snippet, customVal) {
			t.Errorf("持久化 sources[%d] 泄露自定义字段原文: %s", i, s.Snippet)
		}
		if !strings.Contains(s.Snippet, wantMasked) {
			t.Errorf("持久化 sources[%d] 应含完整脱敏值（防截断前缀泄露尾部）: %s", i, s.Snippet)
		}
	}
}

// TestStreamChat_LongMixedFieldNoLeak 中文+ASCII 混合超长字段（远超 60 字节）跨多个 SSE token：
// 每个 token、拼接、最终答案、Sources Snippet/Content、持久化 Content/Sources 均不含完整原文
// 及被遮蔽的中间/尾部独特标记，且含完整脱敏值（后4字符保留）。
func TestStreamChat_LongMixedFieldNoLeak(t *testing.T) {
	const mixed = "北京市朝阳区望京街道SOHO塔楼A座第66层6601室内部编号XZ8899"
	server := chunkedStreamLLMServer(
		"该员工现住址为北京市朝阳区望京街道SOHO",
		"塔楼A座第66层6601室内部编号XZ8899，请知悉",
	)
	defer server.Close()
	svc, db := setupChatTestService(t, server.URL)

	user := seedChatUser(t, db, "longmixeduser")
	kb := seedChatKB(t, db, "混合长字段知识库", user.ID)
	if err := db.Create(&models.KBFieldMask{
		KnowledgeBaseID: kb.ID, FieldName: "address", MaskPattern: "front3back4",
	}).Error; err != nil {
		t.Fatalf("创建脱敏规则失败: %v", err)
	}
	seedChatDocument(t, db, user.ID, kb.ID, "LONGMIXED-1", "员工档案",
		"员工档案 家庭住址："+mixed+"，备注：正常")

	response := httptest.NewRecorder()
	svc.StreamChat(response, user.ID, "longmixed-session", "家庭住址", kb.ID)

	tokens := streamTokenContents(t, response.Body.String())
	joined := strings.Join(tokens, "")
	wantMasked := applyFront3Back4(mixed)

	// 每个已输出 token：不含完整原文，也不含中间/尾部独特标记
	for i, tok := range tokens {
		if strings.Contains(tok, mixed) {
			t.Errorf("第 %d 个已输出 token 泄露完整原文: %q", i, tok)
		}
		if strings.Contains(tok, "内部编号XZ") {
			t.Errorf("第 %d 个已输出 token 泄露中间/尾部原文: %q", i, tok)
		}
	}
	// 拼接输出：不含完整原文与中间/尾部独特标记，含完整脱敏值
	if strings.Contains(joined, mixed) {
		t.Errorf("SSE 拼接输出泄露完整原文: %q", joined)
	}
	if strings.Contains(joined, "内部编号XZ") {
		t.Errorf("SSE 拼接输出泄露中间/尾部原文: %q", joined)
	}
	if !strings.Contains(joined, wantMasked) {
		t.Errorf("SSE 输出应含完整脱敏值 %q，实际: %q", wantMasked, joined)
	}
	// front3back4：前3字符（北京市）+ 后4字符（8899）保留，中间星号遮蔽
	if !strings.Contains(joined, "北京市") || !strings.Contains(joined, "8899") {
		t.Errorf("SSE 输出应保留前3后4字符，实际: %q", joined)
	}

	// 持久化 assistant Content 与 Sources 均不含原文及中间/尾部标记
	var msg models.ChatMessage
	if err := db.Where("session_id = ? AND role = ?", "longmixed-session", "assistant").First(&msg).Error; err != nil {
		t.Fatalf("查询助手消息失败: %v", err)
	}
	if strings.Contains(msg.Content, mixed) || strings.Contains(msg.Content, "内部编号XZ") {
		t.Errorf("持久化 assistant Content 泄露原文: %s", msg.Content)
	}
	var persisted []SearchResult
	if err := json.Unmarshal(msg.Sources, &persisted); err != nil {
		t.Fatalf("解析持久化 sources 失败: %v", err)
	}
	for i, s := range persisted {
		if strings.Contains(s.Snippet, mixed) || strings.Contains(s.Content, mixed) ||
			strings.Contains(s.Snippet, "内部编号XZ") || strings.Contains(s.Content, "内部编号XZ") {
			t.Errorf("持久化 sources[%d] 泄露原文: snippet=%q content=%q", i, s.Snippet, s.Content)
		}
		if !strings.Contains(s.Snippet, wantMasked) {
			t.Errorf("持久化 sources[%d] 应含完整脱敏值: %s", i, s.Snippet)
		}
	}
}

// TestStreamChat_MaskRuleQueryFailureNoToken 脱敏规则查询失败：
// StreamChat 安全失败，不输出任何 token 事件（仅 error 事件）
func TestStreamChat_MaskRuleQueryFailureNoToken(t *testing.T) {
	server := newStreamLLMServer()
	defer server.Close()
	svc, db := setupChatTestService(t, server.URL)

	user := seedChatUser(t, db, "streamfailmask")
	kb := seedChatKB(t, db, "流式规则失败知识库", user.ID)
	seedChatDocument(t, db, user.ID, kb.ID, "SFAIL-1", "员工档案", "脱敏测试内容 110101199001011234")

	db.Callback().Query().Before("gorm:query").Register("fail_mask_rule_query_stream", func(tx *gorm.DB) {
		if tx.Statement.Schema != nil && tx.Statement.Schema.Table == "kb_field_masks" {
			tx.AddError(errors.New("注入规则查询失败"))
		}
	})
	defer func() {
		_ = db.Callback().Query().Before("gorm:query").Remove("fail_mask_rule_query_stream")
	}()

	response := httptest.NewRecorder()
	svc.StreamChat(response, user.ID, "stream-failmask", "脱敏测试", kb.ID)

	events := sseEvents(t, response.Body.String())
	for _, event := range events {
		var eventType string
		_ = json.Unmarshal(event["type"], &eventType)
		if eventType == "token" {
			t.Errorf("规则查询失败时不应输出任何 token 事件，实际输出了: %s", event["content"])
		}
	}
	eventByType(t, events, "error")
}

// TestStreamChat_ExemptRoleQueryFailureNoToken 豁免角色查询失败：
// StreamChat 安全失败，不输出任何 token 事件（仅 error 事件）
func TestStreamChat_ExemptRoleQueryFailureNoToken(t *testing.T) {
	server := newStreamLLMServer()
	defer server.Close()
	svc, db := setupChatTestService(t, server.URL)

	user := seedChatUser(t, db, "streamexemptfail")
	kb := seedChatKB(t, db, "流式豁免失败知识库", user.ID)
	exemptRole := "admin"
	if err := db.Create(&models.KBFieldMask{
		KnowledgeBaseID: kb.ID, FieldName: "address", MaskPattern: "front3back4", ExemptRole: &exemptRole,
	}).Error; err != nil {
		t.Fatalf("创建脱敏规则失败: %v", err)
	}
	seedChatDocument(t, db, user.ID, kb.ID, "SEXEMPT-1", "员工档案", "员工档案 家庭住址：北京市朝阳区望京SOHO")

	// 注入 roles 表查询失败（userExemptFromMask 路径）
	db.Callback().Query().Before("gorm:query").Register("fail_exempt_role_query_stream", func(tx *gorm.DB) {
		if tx.Statement.Schema != nil && tx.Statement.Schema.Table == "roles" {
			tx.AddError(errors.New("注入豁免角色查询失败"))
		}
	})
	defer func() {
		_ = db.Callback().Query().Before("gorm:query").Remove("fail_exempt_role_query_stream")
	}()

	response := httptest.NewRecorder()
	svc.StreamChat(response, user.ID, "stream-exemptfail", "家庭住址", kb.ID)

	events := sseEvents(t, response.Body.String())
	for _, event := range events {
		var eventType string
		_ = json.Unmarshal(event["type"], &eventType)
		if eventType == "token" {
			t.Errorf("豁免角色查询失败时不应输出任何 token 事件，实际输出了: %s", event["content"])
		}
	}
	eventByType(t, events, "error")
}

// grantAdminRole 给用户授予 admin 角色（豁免测试用）
func grantAdminRole(t *testing.T, db *gorm.DB, userID uint) {
	t.Helper()
	role := models.Role{Name: "admin", Label: "管理员"}
	if err := db.Create(&role).Error; err != nil {
		t.Fatalf("创建 admin 角色失败: %v", err)
	}
	if err := db.Create(&models.UserRole{UserID: userID, RoleID: role.ID}).Error; err != nil {
		t.Fatalf("绑定角色失败: %v", err)
	}
}

// TestChat_ExemptRole ExemptRole 完整豁免语义：
// 命中豁免角色时 Sources 保留原文，模型最终答案、持久化 assistant Content 也保留原文；
// 未命中豁免时 Sources、最终答案、持久化 Content 均脱敏（非豁免数字字段通用脱敏仍生效）
func TestChat_ExemptRole(t *testing.T) {
	server := newMaskingLLMServer()
	defer server.Close()

	const idCard = "110101199001011234"
	const idCardMasked = "110***********1234" // front3back4 脱敏结果

	// 场景 A：ExemptRole=admin，用户无 admin 角色 → 脱敏生效
	svc, db := setupChatTestService(t, server.URL)
	user := seedChatUser(t, db, "exemptuser")
	kb := seedChatKB(t, db, "豁免知识库", user.ID)
	exemptRole := "admin"
	if err := db.Create(&models.KBFieldMask{
		KnowledgeBaseID: kb.ID, FieldName: "id_card", MaskPattern: "front3back4", ExemptRole: &exemptRole,
	}).Error; err != nil {
		t.Fatalf("创建脱敏规则失败: %v", err)
	}
	seedChatDocument(t, db, user.ID, kb.ID, "EXEMPT-1", "员工档案", "脱敏测试内容 "+idCard)

	resp, err := svc.Chat(user.ID, "exempt-session", "脱敏测试", kb.ID)
	if err != nil {
		t.Fatalf("Chat 失败: %v", err)
	}
	if len(resp.Sources) == 0 {
		t.Fatal("Chat 应返回检索来源")
	}
	srcA := resp.Sources[0]
	if strings.Contains(srcA.Snippet, idCard) {
		t.Errorf("无豁免角色用户应被脱敏，Sources.Snippet 泄露: %s", srcA.Snippet)
	}
	if srcA.Content != "" && strings.Contains(srcA.Content, idCard) {
		t.Errorf("无豁免角色用户应被脱敏，Sources.Content 泄露: %s", srcA.Content)
	}
	// 最终答案：模型复述原文被防御层替换，且含通用脱敏值（非豁免数字字段防御仍生效）
	if strings.Contains(resp.Answer, idCard) {
		t.Errorf("无豁免角色用户应被脱敏，Answer 泄露: %s", resp.Answer)
	}
	if !strings.Contains(resp.Answer, idCardMasked) {
		t.Errorf("无豁免角色用户 Answer 应含通用脱敏值 %q，实际: %q", idCardMasked, resp.Answer)
	}
	// 持久化 assistant Content 与 Sources 不得泄露原文
	var msgA models.ChatMessage
	if err := db.Where("session_id = ? AND role = ?", "exempt-session", "assistant").First(&msgA).Error; err != nil {
		t.Fatalf("查询助手消息失败: %v", err)
	}
	if strings.Contains(msgA.Content, idCard) {
		t.Errorf("无豁免角色用户持久化 assistant Content 泄露原文: %s", msgA.Content)
	}
	var persistedA []SearchResult
	if err := json.Unmarshal(msgA.Sources, &persistedA); err != nil {
		t.Fatalf("解析持久化 sources 失败: %v", err)
	}
	if len(persistedA) == 0 {
		t.Fatal("持久化 sources 为空")
	}
	if strings.Contains(persistedA[0].Snippet, idCard) {
		t.Errorf("无豁免角色用户持久化 sources.Snippet 泄露原文: %s", persistedA[0].Snippet)
	}

	// 场景 B：授予 admin 角色 → 豁免生效，Sources/Answer/持久化 Content 保留原文
	grantAdminRole(t, db, user.ID)
	resp2, err := svc.Chat(user.ID, "exempt-session2", "脱敏测试", kb.ID)
	if err != nil {
		t.Fatalf("Chat 失败: %v", err)
	}
	if len(resp2.Sources) == 0 {
		t.Fatal("Chat 应返回检索来源")
	}
	srcB := resp2.Sources[0]
	if !strings.Contains(srcB.Snippet, idCard) {
		t.Errorf("豁免角色用户应看到原文，Sources.Snippet 被脱敏: %s", srcB.Snippet)
	}
	if srcB.Content != "" && !strings.Contains(srcB.Content, idCard) {
		t.Errorf("豁免角色用户应看到原文，Sources.Content 被脱敏: %s", srcB.Content)
	}
	// 最终答案：豁免值在防御层被跳过，模型复述原文保留（避免先替换后无法恢复）
	if !strings.Contains(resp2.Answer, idCard) {
		t.Errorf("豁免角色用户应看到原文，Answer 被脱敏: %s", resp2.Answer)
	}
	// 持久化 assistant Content 与 Sources 保留原文
	var msgB models.ChatMessage
	if err := db.Where("session_id = ? AND role = ?", "exempt-session2", "assistant").First(&msgB).Error; err != nil {
		t.Fatalf("查询助手消息失败: %v", err)
	}
	if !strings.Contains(msgB.Content, idCard) {
		t.Errorf("豁免角色用户持久化 assistant Content 应保留原文，实际被脱敏: %s", msgB.Content)
	}
	var persistedB []SearchResult
	if err := json.Unmarshal(msgB.Sources, &persistedB); err != nil {
		t.Fatalf("解析持久化 sources 失败: %v", err)
	}
	if len(persistedB) == 0 {
		t.Fatal("持久化 sources 为空")
	}
	if !strings.Contains(persistedB[0].Snippet, idCard) {
		t.Errorf("豁免角色用户持久化 sources.Snippet 应保留原文，实际被脱敏: %s", persistedB[0].Snippet)
	}
}

// TestChat_KBIDZeroScope kb_id=0：检索全部可见知识库，不泄露他人私有 KB
func TestChat_KBIDZeroScope(t *testing.T) {
	server := newStreamLLMServer()
	defer server.Close()
	svc, db := setupChatTestService(t, server.URL)

	user1 := seedChatUser(t, db, "zeroscope1")
	user2 := seedChatUser(t, db, "zeroscope2")
	kb2 := seedChatKB(t, db, "他人私有知识库", user2.ID)
	seedChatDocument(t, db, user2.ID, kb2.ID, "ZEROSCOPE-1", "员工-杨过", "杨过 黯然销魂掌")

	resp, err := svc.Chat(user1.ID, "zero-scope-session", "杨过", 0)
	if err != nil {
		t.Fatalf("Chat 失败: %v", err)
	}
	// 用户 1 无权限访问 kb2（私有），kb_id=0 不应命中他人文档
	if len(resp.Sources) != 0 {
		t.Errorf("kb_id=0 检索应过滤他人私有 KB，实际命中 %d 条: %+v", len(resp.Sources), resp.Sources)
	}
}

// assertStreamExemptPersisted 校验流式持久化结果（assistant Content / Sources Snippet/Content）：
// wantExempt=true 时身份证原文必须保留；false 时身份证/手机号原文均不得出现
func assertStreamExemptPersisted(t *testing.T, db *gorm.DB, sessionID, idCard, phone string, wantExempt bool) {
	t.Helper()
	var msg models.ChatMessage
	if err := db.Where("session_id = ? AND role = ?", sessionID, "assistant").First(&msg).Error; err != nil {
		t.Fatalf("查询助手消息失败: %v", err)
	}
	if wantExempt {
		if !strings.Contains(msg.Content, idCard) {
			t.Errorf("豁免角色用户持久化 assistant Content 应保留身份证原文: %s", msg.Content)
		}
	} else if strings.Contains(msg.Content, idCard) || strings.Contains(msg.Content, phone) {
		t.Errorf("无豁免角色用户持久化 assistant Content 泄露原文: %s", msg.Content)
	}
	var persisted []SearchResult
	if err := json.Unmarshal(msg.Sources, &persisted); err != nil {
		t.Fatalf("解析持久化 sources 失败: %v", err)
	}
	if len(persisted) == 0 {
		t.Fatal("持久化 sources 为空")
	}
	for i, s := range persisted {
		if wantExempt {
			if !strings.Contains(s.Snippet, idCard) {
				t.Errorf("豁免角色用户持久化 sources[%d].Snippet 应保留身份证原文: %s", i, s.Snippet)
			}
		} else if strings.Contains(s.Snippet, idCard) || strings.Contains(s.Snippet, phone) {
			t.Errorf("无豁免角色用户持久化 sources[%d].Snippet 泄露原文: %s", i, s.Snippet)
		}
		if s.Content != "" {
			if wantExempt && !strings.Contains(s.Content, idCard) {
				t.Errorf("豁免角色用户持久化 sources[%d].Content 应保留身份证原文: %s", i, s.Content)
			}
			if !wantExempt && strings.Contains(s.Content, idCard) {
				t.Errorf("无豁免角色用户持久化 sources[%d].Content 泄露原文: %s", i, s.Content)
			}
		}
	}
}

// TestStreamChat_ExemptRole ExemptRole 完整豁免语义（SSE）：
// 规则：id_card 豁免 admin、phone 不豁免。
//   - 无豁免角色：token 拼接不含原文、含脱敏值；持久化 Content/Sources 不含原文
//   - 豁免角色：多个豁免值（两个身份证）跨 token 拆分后拼接仍保留完整原文；
//     非豁免数字字段（手机号）仍被通用脱敏；持久化 Content/Sources 保留豁免原文
func TestStreamChat_ExemptRole(t *testing.T) {
	const idCard1 = "110101199001011234"
	const idCard2 = "110101199001011235"
	const phone = "13800138000"
	const idCard1Masked = "110***********1234"
	const idCard2Masked = "110***********1235"

	// 流式 mock：两个身份证分片跨 token 返回，手机号分两片返回
	server := chunkedStreamLLMServer(
		"证件 "+idCard1+" 与 1101",
		"01199001011235，手机号 13800138",
		"000，请知悉",
	)
	defer server.Close()
	svc, db := setupChatTestService(t, server.URL)
	user := seedChatUser(t, db, "streamexempt")
	kb := seedChatKB(t, db, "流式豁免知识库", user.ID)
	exemptRole := "admin"
	// id_card 规则豁免 admin；phone 规则不豁免（始终脱敏）
	if err := db.Create(&models.KBFieldMask{
		KnowledgeBaseID: kb.ID, FieldName: "id_card", MaskPattern: "front3back4", ExemptRole: &exemptRole,
	}).Error; err != nil {
		t.Fatalf("创建脱敏规则失败: %v", err)
	}
	if err := db.Create(&models.KBFieldMask{
		KnowledgeBaseID: kb.ID, FieldName: "phone", MaskPattern: "front3back4",
	}).Error; err != nil {
		t.Fatalf("创建脱敏规则失败: %v", err)
	}
	seedChatDocument(t, db, user.ID, kb.ID, "SEXEMPT-1", "员工档案",
		"身份证号："+idCard1+"，备用证件："+idCard2+"，手机号："+phone)

	// 场景 A：无 admin 角色 → 身份证/手机号均脱敏
	responseA := httptest.NewRecorder()
	svc.StreamChat(responseA, user.ID, "sexempt-a", "证件", kb.ID)
	joinedA := strings.Join(streamTokenContents(t, responseA.Body.String()), "")
	if strings.Contains(joinedA, idCard1) || strings.Contains(joinedA, idCard2) || strings.Contains(joinedA, phone) {
		t.Errorf("无豁免角色用户 SSE 拼接泄露原文: %q", joinedA)
	}
	if !strings.Contains(joinedA, idCard1Masked) || !strings.Contains(joinedA, idCard2Masked) || !strings.Contains(joinedA, "138****8000") {
		t.Errorf("无豁免角色用户 SSE 应含通用脱敏值，实际: %q", joinedA)
	}
	assertStreamExemptPersisted(t, db, "sexempt-a", idCard1, phone, false)

	// 场景 B：授予 admin 角色 → 豁免身份证保留原文（多个豁免值跨 token），手机号仍脱敏
	grantAdminRole(t, db, user.ID)
	responseB := httptest.NewRecorder()
	svc.StreamChat(responseB, user.ID, "sexempt-b", "证件", kb.ID)
	joinedB := strings.Join(streamTokenContents(t, responseB.Body.String()), "")
	if !strings.Contains(joinedB, idCard1) || !strings.Contains(joinedB, idCard2) {
		t.Errorf("豁免角色用户 SSE 拼接应保留多个身份证原文，实际: %q", joinedB)
	}
	if strings.Contains(joinedB, phone) {
		t.Errorf("豁免角色用户 SSE 拼接不应泄露非豁免手机号原文: %q", joinedB)
	}
	if !strings.Contains(joinedB, "138****8000") {
		t.Errorf("豁免角色用户 SSE 拼接中非豁免手机号仍应通用脱敏，实际: %q", joinedB)
	}
	assertStreamExemptPersisted(t, db, "sexempt-b", idCard1, phone, true)
}
