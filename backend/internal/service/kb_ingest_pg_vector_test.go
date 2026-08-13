package service

import (
	"context"
	"math"
	"testing"

	"siapp/internal/models"
)

// ============================================================
// 3. 768 维 embedding 原生列 + JSON 双写后读取一致
// ============================================================

func TestP9PG_EmbeddingDualWriteConsistency(t *testing.T) {
	db, prefix := setupP9PGDB(t)
	if db == nil {
		return
	}
	user := seedP9PGUser(t, db, prefix, 1)
	kb := seedP9PGKB(t, db, prefix)
	seedP9PGEmployee(t, db, user.ID, prefix, "张三")

	// mock embedding API：返回 768 维固定向量
	srv := p9PGEmbeddingServer(t, p9Vec768())
	seedP9PGEmbeddingConfig(t, db, prefix, user.ID, srv.URL)

	svc := newP9PGService(t, db, p9FixedParser())
	if _, err := svc.Ingest(context.Background(), user.ID, IngestRequest{KBID: kb.ID, SourceModule: "employee"}); err != nil {
		t.Fatalf("Ingest 失败: %v", err)
	}

	var chunks []models.DocumentChunk
	if err := db.Find(&chunks).Error; err != nil {
		t.Fatalf("查询 chunks 失败: %v", err)
	}
	if len(chunks) == 0 {
		t.Fatal("期望存在 chunks")
	}
	for _, c := range chunks {
		if c.IndexStatus != models.IndexStatusReady {
			t.Errorf("chunk %d index_status 期望 ready，实际 %q", c.ID, c.IndexStatus)
		}
		// 原生向量列读回
		var vecText string
		if err := db.Raw("SELECT embedding::text FROM document_chunks WHERE id = ?", c.ID).Scan(&vecText).Error; err != nil {
			t.Fatalf("读取 embedding 原生列失败: %v", err)
		}
		pgVec, err := parsePGVector(vecText)
		if err != nil {
			t.Fatalf("解析 pgvector 文本失败: %v", err)
		}
		if len(pgVec) != embeddingDim {
			t.Fatalf("原生列维度期望 %d，实际 %d", embeddingDim, len(pgVec))
		}
		// JSON 副本读回
		jsonVec, err := VectorFromJSON(c.EmbeddingJSON)
		if err != nil {
			t.Fatalf("解析 embedding_json 失败: %v", err)
		}
		if len(jsonVec) != embeddingDim {
			t.Fatalf("JSON 维度期望 %d，实际 %d", embeddingDim, len(jsonVec))
		}
		// 双写一致（vectorToPGString 用 %.6f 格式化，容差 1e-5）
		for i := range pgVec {
			if math.Abs(pgVec[i]-jsonVec[i]) > 1e-5 {
				t.Fatalf("双写不一致: 原生列[%d]=%f, JSON[%d]=%f", i, pgVec[i], i, jsonVec[i])
			}
		}
	}
}

// ============================================================
// 4. pgvector <=> 向量检索命中（真实链路：Ingest 写原生列 → 检索命中）
// ============================================================

func TestP9PG_VectorSearchHit(t *testing.T) {
	db, prefix := setupP9PGDB(t)
	if db == nil {
		return
	}
	user := seedP9PGUser(t, db, prefix, 1)
	kb := seedP9PGKB(t, db, prefix)
	seedP9PGEmployee(t, db, user.ID, prefix, "小龙女")

	// 入库与查询同源向量 → 余弦相似度 ≈ 1
	srv := p9PGEmbeddingServer(t, p9Vec768())
	seedP9PGEmbeddingConfig(t, db, prefix, user.ID, srv.URL)

	svc := newP9PGService(t, db, p9FixedParser())
	if _, err := svc.Ingest(context.Background(), user.ID, IngestRequest{KBID: kb.ID, SourceModule: "employee"}); err != nil {
		t.Fatalf("Ingest 失败: %v", err)
	}

	retrieval := NewRetrievalService(db, NewEmbeddingService(db))
	results, err := retrieval.vectorSearch(user.ID, "小龙女", 10, kb.ID)
	if err != nil {
		t.Fatalf("vectorSearch 失败: %v", err)
	}
	if len(results) == 0 {
		t.Fatal("pgvector 检索应命中影子文档，实际无结果")
	}
	if results[0].MatchType != "vector" {
		t.Errorf("MatchType 期望 vector，实际 %q", results[0].MatchType)
	}
	if results[0].Score < 0.9 {
		t.Errorf("同源向量余弦相似度应接近 1，实际 %f", results[0].Score)
	}
}
